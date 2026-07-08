package alhudhud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/internal/redis"
	"github.com/al-hudhud/backend/internal/ws"
	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Worker struct {
	redisClient  *redis.Client
	httpClient   *http.Client
	ctx          context.Context
	cancel       context.CancelFunc
	hub          *ws.Hub // WebSocket Hub for real-time status push

	// Per-Tenant Isolated Queues for Fair Parallelism
	tenantQueues map[uuid.UUID]chan OfficialDispatchPayload
	queuesMutex  sync.RWMutex

	// Gateway config caching to avoid DB bottleneck under load
	gatewayCache *models.AlhudhudGateway
	cacheMutex   sync.RWMutex
}

func NewWorker(redisClient *redis.Client, hub *ws.Hub) *Worker {
	ctx, cancel := context.WithCancel(context.Background())
	return &Worker{
		redisClient: redisClient,
		hub:         hub,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		ctx:          ctx,
		cancel:       cancel,
		tenantQueues: make(map[uuid.UUID]chan OfficialDispatchPayload),
	}
}

type OfficialDispatchPayload struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	PhoneNumber string    `json:"phone_number"`
	Content     string    `json:"content"`
	CustomID    string    `json:"custom_id"`
	Provider    string    `json:"provider"`
}

// Start begins the subscription and workers pool
func (w *Worker) Start() {
	if w.redisClient == nil {
		logger.GetLogger().Warn("Redis client is nil, Alhudhud worker will not start")
		return
	}

	// Initial cache load and start background refresh
	w.refreshGatewayCache()
	go w.cacheRefresher()

	// Start listening to Redis
	go w.listen()
	logger.GetLogger().Info("Alhudhud Official Gateway Worker started with Per-Tenant Parallelism")
}

// Stop gracefully shuts down the worker
func (w *Worker) Stop() {
	w.cancel()
}

func (w *Worker) cacheRefresher() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.refreshGatewayCache()
		}
	}
}

func (w *Worker) refreshGatewayCache() {
	database := db.Get(context.Background())
	var gateway models.AlhudhudGateway
	if err := database.Where("is_active = ?", true).First(&gateway).Error; err != nil {
		logger.GetLogger().Warn("Failed to refresh active official gateway config", zap.Error(err))
		return
	}
	
	w.cacheMutex.Lock()
	w.gatewayCache = &gateway
	w.cacheMutex.Unlock()
}

func (w *Worker) getOrCreateTenantQueue(tenantID uuid.UUID) chan OfficialDispatchPayload {
	w.queuesMutex.RLock()
	ch, exists := w.tenantQueues[tenantID]
	w.queuesMutex.RUnlock()

	if exists {
		return ch
	}

	w.queuesMutex.Lock()
	defer w.queuesMutex.Unlock()
	
	// Double check pattern
	if ch, exists = w.tenantQueues[tenantID]; exists {
		return ch
	}

	// Create a new queue for this tenant with a reasonable buffer
	newCh := make(chan OfficialDispatchPayload, 1000)
	w.tenantQueues[tenantID] = newCh

	// Launch a dedicated worker routine for this specific tenant!
	// This guarantees that Tenant A cannot block Tenant B
	go w.tenantWorkerRoutine(tenantID, newCh)

	logger.GetLogger().Info("Created dedicated worker and queue for tenant", zap.String("tenant_id", tenantID.String()))
	return newCh
}

func (w *Worker) tenantWorkerRoutine(tenantID uuid.UUID, queue chan OfficialDispatchPayload) {
	for {
		select {
		case <-w.ctx.Done():
			return
		case payload := <-queue:
			w.processMessage(payload)
		}
	}
}

func (w *Worker) listen() {
	pubsub := w.redisClient.Client.PSubscribe(w.ctx, "alhudhud:dispatch:official:*")
	defer pubsub.Close()

	ch := pubsub.Channel()

	for {
		select {
		case <-w.ctx.Done():
			return
		case msg := <-ch:
			var payload OfficialDispatchPayload
			if err := json.Unmarshal([]byte(msg.Payload), &payload); err != nil {
				logger.GetLogger().Error("Failed to unmarshal official dispatch payload", zap.Error(err))
				continue
			}

			// Get the tenant-specific isolated queue
			tenantQueue := w.getOrCreateTenantQueue(payload.TenantID)

			// Push to tenant's queue
			select {
			case tenantQueue <- payload:
			default:
				logger.GetLogger().Warn("Tenant's Alhudhud worker queue is full, processing synchronously as fallback", zap.String("tenant_id", payload.TenantID.String()))
				go w.processMessage(payload)
			}
		}
	}
}

func (w *Worker) processMessage(payload OfficialDispatchPayload) {
	log := logger.GetLogger().With(
		zap.String("message_id", payload.ID.String()),
		zap.String("tenant_id", payload.TenantID.String()),
	)

	database := db.Get(w.ctx)

	// Get active Alhudhud Gateway config securely from cache
	w.cacheMutex.RLock()
	gateway := w.gatewayCache
	w.cacheMutex.RUnlock()

	if gateway == nil {
		log.Error("No active official gateway configured in cache")
		w.updateMessageStatus(database, payload.ID, payload.TenantID, string(models.MessageFailed), "No active official gateway")
		return
	}

	// Prepare payload for Telecom Provider (Bawabaty format)
	// Modify this struct to match Bawabaty's exact JSON requirements
	bawabatyPayload := map[string]interface{}{
		"sender_id": gateway.SenderID,
		"numbers":   payload.PhoneNumber,
		"message":   payload.Content,
		"api_key":   gateway.APIKey,
	}

	payloadBytes, err := json.Marshal(bawabatyPayload)
	if err != nil {
		log.Error("Failed to marshal Bawabaty payload", zap.Error(err))
		w.updateMessageStatus(database, payload.ID, payload.TenantID, string(models.MessageFailed), "Payload encoding error")
		return
	}

	req, err := http.NewRequestWithContext(w.ctx, "POST", gateway.BaseURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Error("Failed to create HTTP request", zap.Error(err))
		w.updateMessageStatus(database, payload.ID, payload.TenantID, string(models.MessageFailed), "Request creation error")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	// Send to Bawabaty
	resp, err := w.httpClient.Do(req)
	if err != nil {
		log.Error("Failed to call telecom API", zap.Error(err))
		w.updateMessageStatus(database, payload.ID, payload.TenantID, string(models.MessageFailed), "API connection error")
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	// If HTTP status is 200/201, assume DELIVERED or SENT
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Info("Message delivered via official gateway", zap.Int("status_code", resp.StatusCode), zap.String("response", string(bodyBytes)))
		w.updateMessageStatus(database, payload.ID, payload.TenantID, string(models.MessageDelivered), "")
	} else {
		log.Error("Telecom API returned error", zap.Int("status_code", resp.StatusCode), zap.String("response", string(bodyBytes)))
		w.updateMessageStatus(database, payload.ID, payload.TenantID, string(models.MessageFailed), fmt.Sprintf("API Error: %d", resp.StatusCode))
	}
}

func (w *Worker) updateMessageStatus(database *gorm.DB, messageID uuid.UUID, tenantID uuid.UUID, status string, errorDetails string) {
	database.Model(&models.Message{}).
		Where("id = ? AND tenant_id = ?", messageID, tenantID).
		Updates(map[string]interface{}{
			"status":        status,
			"error_details": errorDetails,
		})

	// Push live status update via WebSocket to tenant dashboard
	if w.hub != nil {
		msg := ws.WebSocketMessage{
			Type:      "MESSAGE_STATUS_UPDATE",
			MessageID: messageID.String(),
			TenantID:  tenantID,
			Payload: map[string]string{
				"status":        status,
				"error_details": errorDetails,
				"gateway":       "ALHUDHUD",
			},
		}
		if err := w.hub.BroadcastToTenant(w.ctx, tenantID, msg); err != nil {
			logger.GetLogger().Warn("Failed to broadcast Alhudhud status update via WebSocket", zap.Error(err), zap.String("message_id", messageID.String()))
		}
	}
}
