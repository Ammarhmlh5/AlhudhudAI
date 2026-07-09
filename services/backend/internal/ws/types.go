package ws

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// WebSocketMessage represents message sent via WebSocket
type WebSocketMessage struct {
	Type        string      `json:"type"` // MESSAGE, HEARTBEAT, ACK, STATUS_REPORT
	MessageID   string      `json:"message_id"`
	TenantID    uuid.UUID   `json:"tenant_id"`
	ChannelID   uuid.UUID   `json:"channel_id"`
	PhoneNumber string      `json:"phone_number,omitempty"`
	Content     string      `json:"content,omitempty"`
	Timestamp   time.Time   `json:"timestamp"`
	Payload     interface{} `json:"payload,omitempty"`
}

// ConnectionKey represents unique connection identifier
type ConnectionKey struct {
	TenantID  uuid.UUID
	ChannelID uuid.UUID
}

// Connection represents a WebSocket connection with metadata
type Connection struct {
	TenantID         uuid.UUID
	ChannelID        uuid.UUID
	DeviceID         string
	Conn             *websocket.Conn
	Send             chan []byte
	Hub              *Hub
	Status           ConnectionStatus // Current connection status
	LastConnectedAt  time.Time        // Last successful connection time
	LastDisconnectAt time.Time        // Last disconnection time
	ReconnectCount   int              // Number of reconnection attempts
	mu               sync.RWMutex     // Mutex for connection state
	closeOnce        sync.Once        // Ensure Send channel is closed once
}

// ConnectionStatus represents connection status
type ConnectionStatus string

const (
	StatusConnected    ConnectionStatus = "CONNECTED"
	StatusDisconnected ConnectionStatus = "DISCONNECTED"
	StatusReconnecting ConnectionStatus = "RECONNECTING"
)

// ReconnectionConfig holds reconnection configuration
type ReconnectionConfig struct {
	MaxReconnectAttempts int           // Maximum reconnection attempts (0 = unlimited)
	ReconnectTimeout     time.Duration // Time to wait before marking as disconnected (5 seconds)
	ReconnectInterval    time.Duration // Interval between reconnection checks (1 second)
}

// DefaultReconnectionConfig returns default reconnection configuration
func DefaultReconnectionConfig() ReconnectionConfig {
	return ReconnectionConfig{
		MaxReconnectAttempts: 0,               // Unlimited attempts
		ReconnectTimeout:     5 * time.Second, // 5 seconds timeout (Requirement 6.5)
		ReconnectInterval:    1 * time.Second, // Check every second
	}
}

// Event represents broadcast event
type Event struct {
	TenantID uuid.UUID   `json:"tenant_id"`
	Type     string      `json:"type"`
	Payload  interface{} `json:"payload"`
}
