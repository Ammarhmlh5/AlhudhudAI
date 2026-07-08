package receiver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/al-hudhud/backend/internal/auth"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/internal/queue"
	"github.com/al-hudhud/backend/internal/redis"
	"github.com/al-hudhud/backend/internal/ws"
	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Handler struct {
	service *Service
}

func NewHandler(redisClient *redis.Client) *Handler {
	return &Handler{
		service: NewService(redisClient),
	}
}

// SendMessageHandler handles single message sending requests
func (h *Handler) SendMessageHandler(c *gin.Context) {
	tenantID, ok := auth.GetTenantIDOrAbort(c)
	if !ok {
		return
	}

	var req struct {
		PhoneNumber    string                 `json:"phone_number" binding:"required"`
		Content        string                 `json:"content" binding:"required"`
		ScheduledAt    *time.Time             `json:"scheduled_at"`
		MessageType    models.MessageCategory `json:"message_type"`
		SenderUserID   *uuid.UUID             `json:"sender_user_id"`
		SenderDeviceID *uuid.UUID             `json:"sender_device_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	idempotencyKey := c.GetHeader("X-Idempotency-Key")

	// Pass the new fields to StageMessage
	msg, err := h.service.StageMessage(c, tenantID.String(), req.PhoneNumber, req.Content, req.ScheduledAt, idempotencyKey, req.MessageType, req.SenderUserID, req.SenderDeviceID)
	if err != nil {
		logger.GetLogger().Error("Failed to stage message",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.GetLogger().Info("Message staged",
		zap.String("tenant_id", tenantID.String()),
		zap.String("message_id", msg.ID.String()),
	)

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Message staged successfully",
		"id":      msg.ID,
		"status":  msg.Status,
	})
}

// BulkSendMessageHandler handles bulk message sending requests (Phase 1: Raw Ingestion)
func (h *Handler) BulkSendMessageHandler(c *gin.Context) {
	tenantID, ok := auth.GetTenantIDOrAbort(c)
	if !ok {
		return
	}

	// 1. Read Raw Body without full parsing to keep extremely low latency
	rawData, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// 2. Ingest the event globally
	event, err := h.service.IngestGlobalEvent(c, tenantID, c.ClientIP(), string(rawData))
	if err != nil {
		logger.GetLogger().Error("Failed to stage global event",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to receive payload"})
		return
	}

	// 3. Return Event ID immediately
	c.JSON(http.StatusAccepted, gin.H{
		"message":   "Bulk messages received successfully",
		"event_id":  event.CustomID, // Return custom ID for tracking
		"status":    event.Status,
		"_internal": event.ID, // Internal ID for admin only
	})
}

// BulkProgressHandler returns the current progress of a bulk ingestion
func (h *Handler) BulkProgressHandler(c *gin.Context) {
	tenantID, ok := auth.GetTenantIDOrAbort(c)
	if !ok {
		return
	}

	var req struct {
		EventID string `uri:"id" binding:"required"`
	}

	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	event, err := h.service.GetBulkEvent(c, tenantID, req.EventID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bulk event not found"})
		return
	}

	progress := 0.0
	if event.TotalItems > 0 {
		progress = float64(event.ProcessedItems+event.FailedItems) / float64(event.TotalItems) * 100
	}

	c.JSON(http.StatusOK, gin.H{
		"event_id":        event.CustomID, // Return custom ID for tracking
		"status":          event.Status,
		"total_items":     event.TotalItems,
		"processed_items": event.ProcessedItems,
		"failed_items":    event.FailedItems,
		"progress":        fmt.Sprintf("%.2f%%", progress),
		"started_at":      event.ProcessingStartedAt,
		"finished_at":     event.ProcessingFinishedAt,
		"_internal_id":    event.ID, // Internal ID for admin only
	})
}

// IngestMessageHandler handles raw JSON ingestion from accounting systems
func (h *Handler) IngestMessageHandler(c *gin.Context) {
	tenantID, ok := auth.GetTenantIDOrAbort(c)
	if !ok {
		return
	}

	// Read Raw Body
	rawData, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Sender Name from Query or Default
	senderName := c.DefaultQuery("sender_name", "Accounting System")

	msg, err := h.service.IngestIncomingMessage(c, tenantID.String(), senderName, string(rawData))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 4. Broadcast live update to UI (Sign 1: Received/Saved)
	ws.GlobalHub.Broadcast(ws.Event{
		TenantID: tenantID,
		Type:     "MESSAGE_RECEIVED",
		Payload: gin.H{
			"id":          msg.ID,
			"sender":      msg.SenderName,
			"received_at": msg.ReceivedAt.Format("15:04:05.000"),
			"status":      "SAVED",
		},
	})

	c.JSON(http.StatusAccepted, gin.H{
		"message":     "Data received and saved",
		"id":          msg.ID,
		"sender":      msg.SenderName,
		"received_at": msg.ReceivedAt.Format("2006-01-02 15:04:05.000"), // Return with ms as requested
	})
}

// BulkIngestExcelHandler handles bulk ingestion specifically for Excel to follow the standard movement
func (h *Handler) BulkIngestExcelHandler(c *gin.Context) {
	tenantID, ok := auth.GetTenantIDOrAbort(c)
	if !ok {
		return
	}

	var req struct {
		Messages    []BulkMessageItem      `json:"messages" binding:"required,dive"`
		MessageType models.MessageCategory `json:"message_type"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// 2. Map to BulkMessagePayload for Unified Ingestion
	payload := queue.BulkMessagePayload{
		MessageType: string(req.MessageType),
		SenderName:  "Excel Import",
	}

	if payload.MessageType == "" {
		payload.MessageType = "EXCEL_IMPORT"
	}
	for _, m := range req.Messages {
		payload.Messages = append(payload.Messages, queue.SingleMessageItem{
			PhoneNumber: m.PhoneNumber,
			Content:     m.Content,
			ScheduledAt: m.ScheduledAt,
		})
	}
	rawJSON, _ := json.Marshal(payload)

	// 3. Ingest as Global Event (Durable & Observable)
	event, err := h.service.IngestGlobalEvent(c, tenantID, c.ClientIP(), string(rawJSON))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to ingest Excel data"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":   "Excel data received successfully",
		"event_id":  event.CustomID, // Return custom ID for tracking
		"status":    event.Status,
		"count":     len(req.Messages),
		"_internal": event.ID, // Internal ID for admin only
	})
}

// ListMessagesHandler retrieves history of messages for the tenant
func (h *Handler) ListMessagesHandler(c *gin.Context) {
	tenantID, ok := auth.GetTenantIDOrAbort(c)
	if !ok {
		return
	}

	// Build comprehensive filters from query params
	filters := listMessagesFilters(c)

	// Pagination params
	page := 1
	pageSize := 50

	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if ps := c.Query("page_size"); ps != "" {
		fmt.Sscanf(ps, "%d", &pageSize)
	}
	if ps := c.Query("limit"); ps != "" {
		fmt.Sscanf(ps, "%d", &pageSize)
	}

	result, err := h.service.ListMessages(c, tenantID.String(), filters, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// listMessagesFilters builds a MessageFilters struct from gin query params
func listMessagesFilters(c *gin.Context) MessageFilters {
	return MessageFilters{
		Status:      models.MessageStatus(c.Query("status")),
		PhoneNumber: c.Query("phone_number"),
		SenderName:  c.Query("sender_name"),
		Source:      c.Query("source"),
		MessageType: c.Query("message_type"),
		ChannelID:   c.Query("channel_id"),
		DateFrom:    c.Query("date_from"),
		DateTo:      c.Query("date_to"),
	}
}

// ListIncomingMessagesHandler retrieves history of raw incoming messages
func (h *Handler) ListIncomingMessagesHandler(c *gin.Context) {
	tenantID, ok := auth.GetTenantIDOrAbort(c)
	if !ok {
		return
	}

	limit := 50 // Default limit
	incoming, err := h.service.ListIncomingMessages(c, tenantID.String(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": incoming,
		"count":    len(incoming),
	})
}
// BulkDeleteMessagesHandler handles bulk message deletion
func (h *Handler) BulkDeleteMessagesHandler(c *gin.Context) {
	tenantID, ok := auth.GetTenantIDOrAbort(c)
	if !ok {
		return
	}

	var req struct {
		IDs []uuid.UUID `json:"ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.service.BulkDeleteMessages(c.Request.Context(), tenantID.String(), req.IDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Messages deleted successfully"})
}

// BulkResendMessagesHandler handles bulk message resending
func (h *Handler) BulkResendMessagesHandler(c *gin.Context) {
	tenantID, ok := auth.GetTenantIDOrAbort(c)
	if !ok {
		return
	}

	var req struct {
		IDs []uuid.UUID `json:"ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.service.BulkResendMessages(c.Request.Context(), tenantID.String(), req.IDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Resend process initiated for failed messages"})
}
