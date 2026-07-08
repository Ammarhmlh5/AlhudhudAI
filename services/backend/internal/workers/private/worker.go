package private

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/al-hudhud/backend/internal/redis"
	"github.com/al-hudhud/backend/internal/ws"
	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Worker struct {
	redisClient *redis.Client
	httpClient  *http.Client
	ctx         context.Context
	cancel      context.CancelFunc
	hub         *ws.Hub // WebSocket Hub for real-time status push

	// Per-Tenant Isolated Queues for Fair Parallelism
	tenantQueues map[uuid.UUID]chan PrivateDispatchPayload
	queuesMutex  sync.RWMutex
}

func NewWorker(redisClient *redis.Client, hub *ws.Hub) *Worker {
	ctx, cancel := context.WithCancel(context.Background())
	return &Worker{
		redisClient: redisClient,
		hub:         hub,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        500,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		ctx:          ctx,
		cancel:       cancel,
		tenantQueues: make(map[uuid.UUID]chan PrivateDispatchPayload),
	}
}

type PrivateDispatchPayload struct {
	ID          uuid.UUID `json:"id"`
	CustomID    string    `json:"custom_id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	PhoneNumber string    `json:"phone_number"`
	Content     string    `json:"content"`
	ServerURL   string    `json:"server_url"`
}

func (w *Worker) Start() {
	if w.redisClient == nil {
		logger.GetLogger().Warn("Redis client is nil, Private gateway worker will not start")
		return
	}

	go w.listen()
	logger.GetLogger().Info("Private Gateway Worker started with Per-Tenant Parallelism")
}

func (w *Worker) Stop() {
	w.cancel()
}

func (w *Worker) getOrCreateTenantQueue(tenantID uuid.UUID) chan PrivateDispatchPayload {
	w.queuesMutex.RLock()
	ch, exists := w.tenantQueues[tenantID]
	w.queuesMutex.RUnlock()

	if exists {
		return ch
	}

	w.queuesMutex.Lock()
	defer w.queuesMutex.Unlock()

	if ch, exists = w.tenantQueues[tenantID]; exists {
		return ch
	}

	newCh := make(chan PrivateDispatchPayload, 1000)
	w.tenantQueues[tenantID] = newCh

	// Launch a dedicated worker routine for this specific tenant!
	go w.tenantWorkerRoutine(tenantID, newCh)

	logger.GetLogger().Info("Created dedicated worker and queue for private tenant", zap.String("tenant_id", tenantID.String()))
	return newCh
}

func (w *Worker) tenantWorkerRoutine(tenantID uuid.UUID, queue chan PrivateDispatchPayload) {
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
	pubsub := w.redisClient.Client.PSubscribe(w.ctx, "alhudhud:dispatch:private:*")
	defer pubsub.Close()

	ch := pubsub.Channel()

	for {
		select {
		case <-w.ctx.Done():
			return
		case msg := <-ch:
			var payload PrivateDispatchPayload
			if err := json.Unmarshal([]byte(msg.Payload), &payload); err != nil {
				logger.GetLogger().Error("Failed to unmarshal private dispatch payload", zap.Error(err))
				continue
			}

			tenantQueue := w.getOrCreateTenantQueue(payload.TenantID)

			select {
			case tenantQueue <- payload:
			default:
				logger.GetLogger().Warn("Tenant's private worker queue is full, processing synchronously as fallback", zap.String("tenant_id", payload.TenantID.String()))
				go w.processMessage(payload)
			}
		}
	}
}

func (w *Worker) processMessage(payload PrivateDispatchPayload) {
	log := logger.GetLogger().With(
		zap.String("message_id", payload.ID.String()),
		zap.String("tenant_id", payload.TenantID.String()),
		zap.String("server_url", payload.ServerURL),
	)

	b, _ := json.Marshal(map[string]interface{}{
		"message_id":   payload.ID.String(),
		"custom_id":    payload.CustomID,
		"phone_number": payload.PhoneNumber,
		"content":      payload.Content,
	})

	req, err := http.NewRequestWithContext(w.ctx, "POST", payload.ServerURL, bytes.NewBuffer(b))
	if err != nil {
		log.Error("PRIVATE path: Failed to build HTTP request", zap.Error(err))
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Message-ID", payload.ID.String())

	resp, err := w.httpClient.Do(req)
	if err != nil {
		log.Error("PRIVATE path: Failed to deliver message to private server", zap.Error(err))
		// Broadcast FAILED status via WebSocket
		w.broadcastStatus(payload.TenantID, payload.ID, "FAILED", "Delivery to private server failed")
		return
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	log.Info("PRIVATE path: Message delivered to private server", zap.Int("status_code", resp.StatusCode))

	// Broadcast SENT status via WebSocket (final DLR status comes via /api/v1/private-gateway/dlr webhook)
	w.broadcastStatus(payload.TenantID, payload.ID, "SENT", "")
}

func (w *Worker) broadcastStatus(tenantID uuid.UUID, messageID uuid.UUID, status string, errDetail string) {
	if w.hub == nil {
		return
	}
	msg := ws.WebSocketMessage{
		Type:      "MESSAGE_STATUS_UPDATE",
		MessageID: messageID.String(),
		TenantID:  tenantID,
		Payload: map[string]string{
			"status":        status,
			"error_details": errDetail,
			"gateway":       "PRIVATE",
		},
	}
	if err := w.hub.BroadcastToTenant(w.ctx, tenantID, msg); err != nil {
		logger.GetLogger().Warn("Failed to broadcast Private status update via WebSocket",
			zap.Error(err),
			zap.String("message_id", messageID.String()),
		)
	}
}
