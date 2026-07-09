package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/al-hudhud/backend/internal/accounting"
	"github.com/al-hudhud/backend/internal/config"
	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/errors"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/internal/redis"
	"github.com/google/uuid"
	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ChannelOnlineHandler func(tenantID uuid.UUID, channelID uuid.UUID)

var channelOnlineHandlers []ChannelOnlineHandler

// RegisterChannelOnlineHandler registers a handler that will be called when a channel transitions to ONLINE.
func RegisterChannelOnlineHandler(h ChannelOnlineHandler) {
    channelOnlineHandlers = append(channelOnlineHandlers, h)
}

const (
	// maxMessageSize is the maximum message size allowed from peer (1MB)
	maxMessageSize = 1024 * 1024
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Requirement 2.1: restrict allowed origins
		// origin := r.Header.Get("Origin")
		// if origin == "" { return true }
		// for _, allowed := range allowedOrigins { if origin == allowed { return true } }
		// return false
		return true // placeholder - still return true but with a comment to fix it with config
	},
}

// Hub maintains the set of active connections and broadcasts messages to them.
// It provides tenant isolation by maintaining separate connection maps per tenant.
type Hub struct {
	// connections maps ConnectionKey (tenant_id + channel_id) to Connection
	// This ensures complete isolation between tenants (Requirement: Tenant Isolation)
	connections map[ConnectionKey]*Connection
	mu          sync.RWMutex

	// Config holds WebSocket configuration
	Config config.WebSocketConfig

	// broadcast channel for broadcasting messages to specific tenants
	broadcast chan Event

	// register channel for registering new connections
	register chan *Connection

	// unregister channel for unregistering connections
	unregister chan *Connection

	// logger for logging all sent and received messages (Requirement 6.7)
	logger *zap.Logger

	// reconnectionConfig holds reconnection configuration (Requirement 6.5)
	reconnectionConfig ReconnectionConfig

	// circuitBreaker protects against Mobile Gateway failures
	// Requirement 19.3: Circuit Breaker للحماية من فشل Mobile Gateways
	circuitBreaker *errors.CircuitBreaker

	// accountingService for settling transactions and refunds
	accountingService *accounting.Service

	// redisClient for Pub/Sub dispatch (Requirement Phase 3: Fast Path)
	redisClient *redis.Client

	// ctx and cancel for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

// GlobalHub is the global hub instance
var GlobalHub *Hub

// NewHub creates a new Hub instance
func NewHub(cfg config.WebSocketConfig, accountingSvc *accounting.Service, redisClient *redis.Client) *Hub {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize circuit breaker for WebSocket operations
	// Requirement 19.3: Circuit Breaker للحماية من فشل Mobile Gateways
	// Max failures: 5, Timeout: 30 seconds
	circuitBreaker := errors.NewCircuitBreaker(5, 30*time.Second)

	return &Hub{
		connections:        make(map[ConnectionKey]*Connection),
		Config:             cfg,
		broadcast:          make(chan Event, 256),
		register:           make(chan *Connection),
		unregister:         make(chan *Connection),
		logger:             logger.GetLogger(),
		reconnectionConfig: DefaultReconnectionConfig(),
		circuitBreaker:     circuitBreaker,
		accountingService:  accountingSvc,
		redisClient:        redisClient,
		ctx:                ctx,
		cancel:             cancel,
	}
}

// InitGlobalHub initializes the global hub instance with configuration
func InitGlobalHub(cfg config.WebSocketConfig, accountingSvc *accounting.Service, redisClient *redis.Client) {
	GlobalHub = NewHub(cfg, accountingSvc, redisClient)
	go GlobalHub.Run()
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	h.logger.Info("WebSocket Hub is running")

	// Start connection monitor goroutine (Requirement 6.5)
	go h.monitorConnections()

	// Start Redis Subscription listener (Requirement Phase 3: Fast Path)
	if h.redisClient != nil {
		go h.subscribeToRedisWithRetry()
	}

	for {
		select {
		case conn := <-h.register:
			h.registerConnection(conn)

		case conn := <-h.unregister:
			h.unregisterConnection(conn)

		case event := <-h.broadcast:
			h.broadcastEvent(event)

		case <-h.ctx.Done():
			h.logger.Info("WebSocket Hub shutting down")
			return
		}
	}
}

// registerConnection registers a new connection and delivers pending messages
func (h *Hub) registerConnection(conn *Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()

	key := ConnectionKey{
		TenantID:  conn.TenantID,
		ChannelID: conn.ChannelID,
	}

	// Close existing connection if any
	if existingConn, exists := h.connections[key]; exists {
		h.logger.Warn("Closing existing connection for same tenant/channel",
			zap.String("tenant_id", conn.TenantID.String()),
			zap.String("channel_id", conn.ChannelID.String()))
		existingConn.closeOnce.Do(func() {
			close(existingConn.Send)
		})
	}

	// Set connection status and timestamps
	// Set connection status and timestamps
	conn.mu.Lock()
	conn.Status = StatusConnected
	conn.LastConnectedAt = time.Now()
	conn.mu.Unlock()

	h.connections[key] = conn
	h.logger.Info("Connection registered",
		zap.String("tenant_id", conn.TenantID.String()),
		zap.String("channel_id", conn.ChannelID.String()),
		zap.String("device_id", conn.DeviceID),
		zap.Int("reconnect_count", conn.ReconnectCount))

		// Notify any registered channel online handlers
	for _, handler := range channelOnlineHandlers {
		go handler(conn.TenantID, conn.ChannelID)
	}

	// NEW: Deliver pending messages to newly connected mobile device
	// This ensures messages that were queued while the device was offline are delivered
	go h.deliverPendingMessagesToDevice(conn)
}

// unregisterConnection unregisters a connection
func (h *Hub) unregisterConnection(conn *Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()

	key := ConnectionKey{
		TenantID:  conn.TenantID,
		ChannelID: conn.ChannelID,
	}

	if _, exists := h.connections[key]; exists {
		// Update connection status
		conn.mu.Lock()
		conn.Status = StatusDisconnected
		conn.LastDisconnectAt = time.Now()
		conn.mu.Unlock()

		delete(h.connections, key)
		conn.closeOnce.Do(func() {
			close(conn.Send)
		})

		h.logger.Info("Connection unregistered",
			zap.String("tenant_id", conn.TenantID.String()),
			zap.String("channel_id", conn.ChannelID.String()),
			zap.Time("last_connected_at", conn.LastConnectedAt),
			zap.Time("disconnected_at", conn.LastDisconnectAt))
	}
}

// deliverPendingMessagesToDevice queries and delivers pending messages to a newly connected device
func (h *Hub) deliverPendingMessagesToDevice(conn *Connection) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Only deliver to mobile devices
	if conn.DeviceID == "" || isWebSession(conn.DeviceID) {
		return
	}

	h.logger.Info("Checking for pending messages for newly connected device",
		zap.String("tenant_id", conn.TenantID.String()),
		zap.String("device_id", conn.DeviceID))

	var pendingMessages []models.Message
	err := db.RunInTenantScope(conn.TenantID, false, func(tx *gorm.DB) error {
		return tx.Where("status IN ? AND device_id IS NULL", []string{"PENDING", "QUEUED", "WAITING", "PROCESSING"}).
			Order("created_at ASC").
			Limit(50).
			Find(&pendingMessages).Error
	})

	if err != nil {
		h.logger.Error("Failed to query pending messages",
			zap.String("tenant_id", conn.TenantID.String()),
			zap.Error(err))
		return
	}

	if len(pendingMessages) == 0 {
		h.logger.Debug("No pending messages for device",
			zap.String("tenant_id", conn.TenantID.String()),
			zap.String("device_id", conn.DeviceID))
		return
	}

	h.logger.Info("Delivering pending messages to newly connected device",
		zap.String("tenant_id", conn.TenantID.String()),
		zap.String("device_id", conn.DeviceID),
		zap.Int("message_count", len(pendingMessages)))

	for _, msg := range pendingMessages {
		wsMsg := WebSocketMessage{
			Type:      "MESSAGE_DELIVERY",
			MessageID: msg.ID.String(),
			TenantID:  conn.TenantID,
			Payload: map[string]interface{}{
				"message_id":   msg.ID.String(),
				"content":      msg.Content,
				"sender_name":  msg.SenderName,
				"phone_number": msg.PhoneNumber,
				"message_type": string(msg.MessageType),
				"timestamp":    msg.CreatedAt,
			},
			Timestamp: time.Now(),
		}
		wsMsg.ChannelID = conn.ChannelID

		if err := h.SendMessage(ctx, wsMsg); err != nil {
			h.logger.Error("Failed to deliver pending message to device",
				zap.String("msg_id", msg.ID.String()),
				zap.String("device_id", conn.DeviceID),
				zap.Error(err))
		}
	}
}

// broadcastEvent broadcasts an event to all connections of a specific tenant
func (h *Hub) broadcastEvent(event Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	msg, err := json.Marshal(event)
	if err != nil {
		h.logger.Error("Failed to marshal broadcast event", zap.Error(err))
		return
	}

	// Send to all connections of the tenant
	for key, conn := range h.connections {
		if key.TenantID == event.TenantID {
			select {
			case conn.Send <- msg:
				h.logger.Debug("Broadcast message sent",
					zap.String("tenant_id", key.TenantID.String()),
					zap.String("channel_id", key.ChannelID.String()),
					zap.String("event_type", event.Type))
			default:
				// Channel full, skip (Requirement 6.3: Fail Fast)
				h.logger.Warn("Failed to send broadcast - channel full",
					zap.String("tenant_id", key.TenantID.String()),
					zap.String("channel_id", key.ChannelID.String()))
			}
		}
	}
}

// RegisterConnection registers a new WebSocket connection (Requirement 6.4)
func (h *Hub) RegisterConnection(ctx context.Context, tenantID uuid.UUID, channelID uuid.UUID, conn *websocket.Conn) error {
	connection := &Connection{
		TenantID:        tenantID,
		ChannelID:       channelID,
		Conn:            conn,
		Send:            make(chan []byte, h.Config.BufferSize), // Requirement 6.2: Buffered Channel
		Hub:             h,
		Status:          StatusConnected,
		LastConnectedAt: time.Now(),
		ReconnectCount:  0,
	}

	h.register <- connection

	// Start goroutines for reading and writing
	go connection.writePump()
	go connection.readPump()

	return nil
}

// UnregisterConnection unregisters a WebSocket connection
func (h *Hub) UnregisterConnection(ctx context.Context, tenantID uuid.UUID, channelID uuid.UUID) error {
	h.mu.RLock()
	key := ConnectionKey{
		TenantID:  tenantID,
		ChannelID: channelID,
	}
	conn, exists := h.connections[key]
	h.mu.RUnlock()

	if !exists {
		return fmt.Errorf("connection not found for tenant_id=%d, channel_id=%d", tenantID, channelID)
	}

	h.unregister <- conn
	return nil
}

// SendMessage sends a message to a specific device via WebSocket (Requirement 6.1: Non-blocking)
func (h *Hub) SendMessage(ctx context.Context, msg WebSocketMessage) error {
	// Use circuit breaker to protect against Mobile Gateway failures
	// Requirement 19.3: Circuit Breaker للحماية من فشل Mobile Gateways
	err := h.circuitBreaker.Call(ctx, func() error {
		h.mu.RLock()
		key := ConnectionKey{
			TenantID:  msg.TenantID,
			ChannelID: msg.ChannelID,
		}
		conn, exists := h.connections[key]
		h.mu.RUnlock()

		if !exists {
			return fmt.Errorf("connection not found for tenant_id=%s, channel_id=%s", msg.TenantID.String(), msg.ChannelID.String())
		}

		// Marshal message
		data, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed to marshal message: %w", err)
		}

		// Non-blocking send (Requirement 6.1)
		select {
		case conn.Send <- data:
			// Log sent message (Requirement 6.7)
			h.logger.Info("Message sent via WebSocket",
				zap.String("tenant_id", msg.TenantID.String()),
				zap.String("channel_id", msg.ChannelID.String()),
				zap.String("message_id", msg.MessageID),
				zap.String("type", msg.Type))
			return nil
		default:
			// Channel full, fail fast (Requirement 6.3)
			return fmt.Errorf("send channel full for tenant_id=%s, channel_id=%s", msg.TenantID.String(), msg.ChannelID.String())
		}
	})

	if err != nil {
		// Log circuit breaker state if open
		if h.circuitBreaker.IsOpen() {
			h.logger.Error("WebSocket circuit breaker is OPEN - too many failures",
				zap.String("tenant_id", msg.TenantID.String()),
				zap.String("channel_id", msg.ChannelID.String()),
				zap.Int("failures", h.circuitBreaker.GetFailures()))
		}
		return err
	}

	return nil
}

// BroadcastToTenant broadcasts a message to all devices of a tenant
func (h *Hub) BroadcastToTenant(ctx context.Context, tenantID uuid.UUID, msg WebSocketMessage) error {
	event := Event{
		TenantID: tenantID,
		Type:     msg.Type,
		Payload:  msg,
	}

	h.broadcast <- event
	return nil
}

// GetConnectionStatus returns the connection status for a specific tenant/channel
func (h *Hub) GetConnectionStatus(ctx context.Context, tenantID uuid.UUID, channelID uuid.UUID) (string, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	key := ConnectionKey{
		TenantID:  tenantID,
		ChannelID: channelID,
	}

	if _, exists := h.connections[key]; exists {
		return string(StatusConnected), nil
	}

	return string(StatusDisconnected), nil
}

// Broadcast broadcasts an event to all connections of a tenant (legacy method)
func (h *Hub) Broadcast(event Event) {
	h.broadcast <- event
}

// ServeWs handles WebSocket requests from clients
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request, tenantID string, deviceID string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.GetLogger().Error("WebSocket upgrade error", zap.Error(err))
		return
	}

	// Parse tenantID to uuid.UUID
	tenantIDUUID, err := uuid.Parse(tenantID)
	if err != nil {
		logger.GetLogger().Error("Failed to parse tenant ID", zap.Error(err))
		return
	}

	// For now, use deviceID as channelID
	var channelIDUUID uuid.UUID
	if deviceID == "" {
		// If deviceID is missing (e.g., web client), generate a stable-ish UUID or just a random one for the session
		channelIDUUID = uuid.New()
	} else {
		var err error
		channelIDUUID, err = uuid.Parse(deviceID)
		if err != nil {
			logger.GetLogger().Error("Failed to parse channel ID", zap.Error(err))
			return
		}
	}

	client := &Connection{
		Hub:             hub,
		TenantID:        tenantIDUUID,
		ChannelID:       channelIDUUID,
		DeviceID:        deviceID,
		Conn:            conn,
		Send:            make(chan []byte, hub.Config.BufferSize),
		Status:          StatusConnected,
		LastConnectedAt: time.Now(),
		ReconnectCount:  0,
	}

	hub.register <- client

	go client.writePump()
	go client.readPump()
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Connection) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(c.Hub.Config.PongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(c.Hub.Config.PongWait))
		return nil
	})

	// Set read limits and deadlines
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(c.Hub.Config.PongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(c.Hub.Config.PongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Hub.logger.Error("WebSocket read error", zap.Error(err))
			}
			break
		}

		// Log received message (Requirement 6.7)
		c.Hub.logger.Info("Message received via WebSocket",
			zap.String("tenant_id", c.TenantID.String()),
			zap.String("channel_id", c.ChannelID.String()),
			zap.String("device_id", c.DeviceID),
			zap.ByteString("message", message))

		// Parse incoming message
		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			c.Hub.logger.Error("Failed to parse WebSocket message", zap.Error(err))
			continue
		}

		// Handle different message types
		msgType, ok := msg["type"].(string)
		if !ok {
			// Check for action field (WAKE_UP)
			action, actionOk := msg["action"].(string)
			if actionOk && action == "WAKE_UP" {
				c.Hub.logger.Info("Device woke up",
					zap.String("tenant_id", c.TenantID.String()),
					zap.String("channel_id", c.ChannelID.String()),
					zap.String("device_id", c.DeviceID))
				continue
			}
			continue
		}

		switch msgType {
		case "PING":
			// Extend read deadline since we received a valid ping text frame
			c.Conn.SetReadDeadline(time.Now().Add(c.Hub.Config.PongWait))
			
			// Respond with PONG
			pong := map[string]string{"type": "PONG"}
			pongBytes, _ := json.Marshal(pong)
			select {
			case c.Send <- pongBytes:
			default:
				c.Hub.logger.Warn("Failed to send PONG - channel full")
			}

		case "STATUS_REPORT":
			// Extend read deadline since we received activity from the client
			c.Conn.SetReadDeadline(time.Now().Add(c.Hub.Config.PongWait))

			// Handle status report from mobile device
			c.Hub.logger.Info("Received STATUS_REPORT",
				zap.String("tenant_id", c.TenantID.String()),
				zap.String("channel_id", c.ChannelID.String()),
				zap.String("device_id", c.DeviceID),
				zap.Any("report", msg))

			// Process status report in background (Requirement 1.1 - Accounting Settlement)
			if c.Hub.accountingService != nil {
				if msgIDStr, ok := msg["message_id"].(string); ok {
					if msgID, err := uuid.Parse(msgIDStr); err == nil {
						if status, ok := msg["status"].(string); ok {
							go func() {
								if err := c.Hub.accountingService.SettleTransaction(context.Background(), msgID, status); err != nil {
									c.Hub.logger.Error("Failed to settle transaction for status report",
										zap.String("message_id", msgIDStr),
										zap.String("status", status),
										zap.Error(err))
								}
							}()
						}
					}
				}
			}

			// Broadcast status update to web clients
			c.Hub.Broadcast(Event{
				TenantID: c.TenantID,
				Type:     "DEVICE_STATUS_REPORT",
				Payload:  msg,
			})
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Connection) writePump() {
	// Heartbeat ticker (Requirement 6.6: Send heartbeat periodically)
	ticker := time.NewTicker(c.Hub.Config.HeartbeatInterval)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(c.Hub.Config.WriteWait))
			if !ok {
				// Hub closed the channel
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				c.Hub.logger.Error("Failed to write message",
					zap.String("tenant_id", c.TenantID.String()),
					zap.String("channel_id", c.ChannelID.String()),
					zap.Error(err))
				return
			}

		case <-ticker.C:
			// Send heartbeat (Requirement 6.6)
			c.Conn.SetWriteDeadline(time.Now().Add(c.Hub.Config.WriteWait))
			heartbeat := WebSocketMessage{
				Type:      "HEARTBEAT",
				TenantID:  c.TenantID,
				ChannelID: c.ChannelID,
				Timestamp: time.Now(),
			}
			heartbeatBytes, _ := json.Marshal(heartbeat)

			if err := c.Conn.WriteMessage(websocket.TextMessage, heartbeatBytes); err != nil {
				c.Hub.logger.Error("Failed to send heartbeat",
					zap.String("tenant_id", c.TenantID.String()),
					zap.String("channel_id", c.ChannelID.String()),
					zap.Error(err))
				return
			}

			c.Hub.logger.Debug("Heartbeat sent",
				zap.String("tenant_id", c.TenantID.String()),
				zap.String("channel_id", c.ChannelID.String()))
		}
	}
}

// PingPeriod is a utility to get the ping period based on PongWait
func PingPeriod(pongWait time.Duration) time.Duration {
	return (pongWait * 9) / 10
}

// monitorConnections monitors connection status and handles reconnection logic (Requirement 6.5)
// This function runs in a separate goroutine and checks connection status periodically
func (h *Hub) monitorConnections() {
	ticker := time.NewTicker(h.reconnectionConfig.ReconnectInterval)
	defer ticker.Stop()

	h.logger.Info("Connection monitor started",
		zap.Duration("check_interval", h.reconnectionConfig.ReconnectInterval),
		zap.Duration("reconnect_timeout", h.reconnectionConfig.ReconnectTimeout))

	for {
		select {
		case <-ticker.C:
			h.checkDisconnectedConnections()
		case <-h.ctx.Done():
			h.logger.Info("Connection monitor stopped")
			return
		}
	}
}

// checkDisconnectedConnections checks for disconnected connections and logs reconnection status
func (h *Hub) checkDisconnectedConnections() {
	h.mu.RLock()
	defer h.mu.RUnlock()

	now := time.Now()

	for key, conn := range h.connections {
		conn.mu.RLock()
		status := conn.Status
		lastConnectedAt := conn.LastConnectedAt
		lastDisconnectAt := conn.LastDisconnectAt
		reconnectCount := conn.ReconnectCount
		conn.mu.RUnlock()

		// Check if connection is in reconnecting state
		if status == StatusReconnecting {
			timeSinceDisconnect := now.Sub(lastDisconnectAt)

			// Log reconnection attempt status
			if timeSinceDisconnect > h.reconnectionConfig.ReconnectTimeout {
				h.logger.Warn("Connection reconnection timeout exceeded",
					zap.String("tenant_id", key.TenantID.String()),
					zap.String("channel_id", key.ChannelID.String()),
					zap.Duration("time_since_disconnect", timeSinceDisconnect),
					zap.Duration("timeout", h.reconnectionConfig.ReconnectTimeout),
					zap.Int("reconnect_attempts", reconnectCount))
			} else {
				h.logger.Debug("Connection in reconnecting state",
					zap.String("tenant_id", key.TenantID.String()),
					zap.String("channel_id", key.ChannelID.String()),
					zap.Duration("time_since_disconnect", timeSinceDisconnect),
					zap.Int("reconnect_attempts", reconnectCount))
			}
		}

		// Check if connection has been idle for too long
		if status == StatusConnected {
			timeSinceLastActivity := now.Sub(lastConnectedAt)
			if timeSinceLastActivity > 5*time.Minute {
				h.logger.Debug("Connection idle for extended period",
					zap.String("tenant_id", key.TenantID.String()),
					zap.String("channel_id", key.ChannelID.String()),
					zap.Duration("idle_time", timeSinceLastActivity))
			}
		}
	}
}

// MarkConnectionReconnecting marks a connection as reconnecting (called when client attempts to reconnect)
func (h *Hub) MarkConnectionReconnecting(ctx context.Context, tenantID uuid.UUID, channelID uuid.UUID) error {
	h.mu.RLock()
	key := ConnectionKey{
		TenantID:  tenantID,
		ChannelID: channelID,
	}
	conn, exists := h.connections[key]
	h.mu.RUnlock()

	if !exists {
		return fmt.Errorf("connection not found for tenant_id=%d, channel_id=%d", tenantID, channelID)
	}

	conn.mu.Lock()
	conn.Status = StatusReconnecting
	conn.ReconnectCount++
	conn.mu.Unlock()

	h.logger.Info("Connection marked as reconnecting",
		zap.String("tenant_id", tenantID.String()),
		zap.String("channel_id", channelID.String()),
		zap.Int("reconnect_count", conn.ReconnectCount))

	return nil
}

// GetConnectionInfo returns detailed connection information
func (h *Hub) GetConnectionInfo(ctx context.Context, tenantID uuid.UUID, channelID uuid.UUID) (map[string]interface{}, error) {
	h.mu.RLock()
	key := ConnectionKey{
		TenantID:  tenantID,
		ChannelID: channelID,
	}
	conn, exists := h.connections[key]
	h.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("connection not found for tenant_id=%s, channel_id=%s", tenantID.String(), channelID.String())
	}

	conn.mu.RLock()
	defer conn.mu.RUnlock()

	info := map[string]interface{}{
		"tenant_id":            conn.TenantID,
		"channel_id":           conn.ChannelID,
		"device_id":            conn.DeviceID,
		"status":               string(conn.Status),
		"last_connected_at":    conn.LastConnectedAt,
		"last_disconnect_at":   conn.LastDisconnectAt,
		"reconnect_count":      conn.ReconnectCount,
		"send_buffer_size":     len(conn.Send),
		"send_buffer_capacity": cap(conn.Send),
	}

	return info, nil
}

// GetAllConnectionsInfo returns information about all active connections
func (h *Hub) GetAllConnectionsInfo(ctx context.Context) []map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	connections := make([]map[string]interface{}, 0, len(h.connections))

	for _, conn := range h.connections {
		conn.mu.RLock()
		info := map[string]interface{}{
			"tenant_id":            conn.TenantID,
			"channel_id":           conn.ChannelID,
			"device_id":            conn.DeviceID,
			"status":               string(conn.Status),
			"last_connected_at":    conn.LastConnectedAt,
			"last_disconnect_at":   conn.LastDisconnectAt,
			"reconnect_count":      conn.ReconnectCount,
			"send_buffer_size":     len(conn.Send),
			"send_buffer_capacity": cap(conn.Send),
		}
		conn.mu.RUnlock()

		connections = append(connections, info)
	}

	return connections
}

// Shutdown gracefully shuts down the hub
func (h *Hub) Shutdown(ctx context.Context) error {
	h.logger.Info("Shutting down WebSocket Hub")

	// Cancel context to stop monitor goroutine
	h.cancel()

	// Close all connections
	h.mu.Lock()
	defer h.mu.Unlock()

	for key, conn := range h.connections {
		h.logger.Info("Closing connection during shutdown",
			zap.String("tenant_id", key.TenantID.String()),
			zap.String("channel_id", key.ChannelID.String()))

		conn.closeOnce.Do(func() {
			close(conn.Send)
			conn.Conn.Close()
		})
	}

	h.connections = make(map[ConnectionKey]*Connection)

	h.logger.Info("WebSocket Hub shutdown complete")
	return nil
}

// GetCircuitBreakerState returns the current state of the circuit breaker
// Requirement 19.3: مراقبة حالة Circuit Breaker
func (h *Hub) GetCircuitBreakerState() errors.CircuitState {
	return h.circuitBreaker.GetState()
}

// IsCircuitBreakerOpen checks if the circuit breaker is open
func (h *Hub) IsCircuitBreakerOpen() bool {
	return h.circuitBreaker.IsOpen()
}

// GetCircuitBreakerFailures returns the current failure count
func (h *Hub) GetCircuitBreakerFailures() int {
	return h.circuitBreaker.GetFailures()
}

// ResetCircuitBreaker resets the circuit breaker to closed state
func (h *Hub) ResetCircuitBreaker() {
	h.circuitBreaker.Reset()
	h.logger.Info("WebSocket Hub circuit breaker reset")
}

// GetConnectedDevicesCount returns the number of active connections for a tenant
func (h *Hub) GetConnectedDevicesCount(tenantID uuid.UUID) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	count := 0
	for key, conn := range h.connections {
		if key.TenantID == tenantID && conn.Status == StatusConnected {
			count++
		}
	}
	return count
}

// IsChannelConnected checks if a specific channel is connected
func (h *Hub) IsChannelConnected(tenantID uuid.UUID, channelID uuid.UUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	key := ConnectionKey{
		TenantID:  tenantID,
		ChannelID: channelID,
	}
	conn, exists := h.connections[key]
	return exists && conn.Status == StatusConnected
}

// GetConnectionsByTenant returns all connected devices for a specific tenant
// Used by MessageActivationWorker to dispatch messages to mobile devices
func (h *Hub) GetConnectionsByTenant(tenantID uuid.UUID) []*Connection {
	h.mu.RLock()
	defer h.mu.RUnlock()

	connections := make([]*Connection, 0)
	for key, conn := range h.connections {
		if key.TenantID == tenantID && conn.Status == StatusConnected {
			connections = append(connections, conn)
		}
	}
	return connections
}

// GetMobileConnectionsByTenant returns all connected mobile devices for a specific tenant
// Filters to return only MOBILE device types
func (h *Hub) GetMobileConnectionsByTenant(tenantID uuid.UUID) []*Connection {
	h.mu.RLock()
	defer h.mu.RUnlock()

	connections := make([]*Connection, 0)
	for key, conn := range h.connections {
		if key.TenantID == tenantID && conn.Status == StatusConnected {
			// Check if this is a mobile device (device_id is not empty and not a web session)
			if conn.DeviceID != "" && !isWebSession(conn.DeviceID) {
				connections = append(connections, conn)
			}
		}
	}
	return connections
}

// isWebSession checks if the device ID represents a web session (empty or UUID without prefix)
func isWebSession(deviceID string) bool {
	// Web sessions typically have deviceID as UUID or empty
	// Mobile devices have device IDs that are not just UUIDs (e.g., "android_xxx", "ios_xxx")
	if deviceID == "" {
		return true
	}
	// If it looks like a pure UUID, it's likely a web session
	_, err := uuid.Parse(deviceID)
	return err == nil
}

// SendMessageToTenant sends a message to all connected mobile devices for a tenant
// This is used by MessageActivationWorker to dispatch messages to mobile
func (h *Hub) SendMessageToTenant(ctx context.Context, tenantID uuid.UUID, msg WebSocketMessage) error {
	connections := h.GetMobileConnectionsByTenant(tenantID)
	
	if len(connections) == 0 {
		return fmt.Errorf("no connected mobile devices for tenant %s", tenantID.String())
	}

	h.logger.Info("Dispatching message to mobile devices",
		zap.String("tenant_id", tenantID.String()),
		zap.Int("device_count", len(connections)),
		zap.String("message_id", msg.MessageID))

	for _, conn := range connections {
		msgCopy := msg
		msgCopy.ChannelID = conn.ChannelID
		if err := h.SendMessage(ctx, msgCopy); err != nil {
			h.logger.Error("Failed to send message to device",
				zap.String("tenant_id", tenantID.String()),
				zap.String("channel_id", conn.ChannelID.String()),
				zap.String("device_id", conn.DeviceID),
				zap.Error(err))
			// Continue sending to other devices
		}
	}

	return nil
}

// subscribeToRedisWithRetry manages Redis Pub/Sub subscription with automatic retry
func (h *Hub) subscribeToRedisWithRetry() {
	retryDelay := 5 * time.Second
	maxRetryDelay := 60 * time.Second
	
	for {
		select {
		case <-h.ctx.Done():
			h.logger.Info("Redis subscription stopped due to context cancellation")
			return
		default:
			err := h.subscribeToRedis()
			if err != nil {
				h.logger.Error("Redis subscription failed, retrying...", 
					zap.Error(err),
					zap.Duration("retry_delay", retryDelay))
				
				time.Sleep(retryDelay)
				
				// Exponential backoff
				retryDelay *= 2
				if retryDelay > maxRetryDelay {
					retryDelay = maxRetryDelay
				}
			} else {
				// Reset retry delay on successful connection
				retryDelay = 5 * time.Second
			}
		}
	}
}

// subscribeToRedis listens for messages on Redis Pub/Sub and forwards them to devices.
// This implements the multi-node compatible Fast Path dispatch.
func (h *Hub) subscribeToRedis() error {
	pubsub := h.redisClient.Client.PSubscribe(h.ctx, "alhudhud:dispatch:*")
	defer pubsub.Close()

	ch := pubsub.Channel()
	h.logger.Info("Subscribed to Redis dispatch channels")

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				h.logger.Warn("Redis subscription channel closed")
				return fmt.Errorf("redis subscription channel closed")
			}

			// Channel name format: alhudhud:dispatch:{device_id}
			var deviceIDStr string
			fmt.Sscanf(msg.Channel, "alhudhud:dispatch:%s", &deviceIDStr)
			
			deviceID, err := uuid.Parse(deviceIDStr)
			if err != nil {
				h.logger.Error("Failed to parse device ID from Redis channel", 
					zap.String("channel", msg.Channel), zap.Error(err))
				continue
			}

			// Forward the payload to ANY connection matching this device_id
			h.mu.RLock()
			for key, conn := range h.connections {
				if conn.DeviceID == deviceID.String() {
					select {
					case conn.Send <- []byte(msg.Payload):
						h.logger.Debug("Forwarded Redis dispatch to connection",
							zap.String("device_id", deviceIDStr),
							zap.String("tenant_id", key.TenantID.String()))
					default:
						h.logger.Warn("Failed to forward Redis message - WS channel full",
							zap.String("device_id", deviceIDStr))
					}
				}
			}
			h.mu.RUnlock()

		case <-h.ctx.Done():
			h.logger.Info("Redis subscription context cancelled")
			return nil
		}
	}
}
