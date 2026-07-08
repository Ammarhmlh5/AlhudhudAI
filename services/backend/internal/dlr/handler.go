package dlr

import (
	"net/http"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/dispatcher"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/internal/queue"
	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Handler handles Delivery Report callbacks from mobile gateways.
// Design principles for Phase 4:
//  - Non-blocking: accepts the call, queues the update, returns 200 immediately.
//  - Keyed on message_id ONLY → no JOINs, no lookups, pure indexed UPDATE.
//  - Idempotent: setting DELIVERED on an already-DELIVERED message is a no-op.

type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

// DeliveryReportRequest is the payload sent by mobile gateways / providers
type DeliveryReportRequest struct {
	MessageID  string `json:"message_id" binding:"required"` // Our internal UUID
	Status     string `json:"status" binding:"required"`     // SENT | DELIVERED | FAILED
	ExternalID string `json:"external_id"`                   // Provider's own job ID (optional)
	ErrorCode  string `json:"error_code"`                    // Failure code if any
	Timestamp  string `json:"timestamp"`                     // Provider delivery timestamp (optional)
}

// ReceiveDLR is the ultra-light async DLR endpoint.
// URL: POST /api/v1/internal/dlr
// Route does NOT require tenant auth – it is protected by API key header instead.
func (h *Handler) ReceiveDLR(c *gin.Context) {
	var req DeliveryReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Parse message ID
	msgID, err := uuid.Parse(req.MessageID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message_id format"})
		return
	}

	// Map provider status → our internal status
	newStatus := mapStatus(req.Status)
	if newStatus == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown status: " + req.Status})
		return
	}

	// Acknowledge immediately – work happens async via River
	c.JSON(http.StatusOK, gin.H{"accepted": true, "message_id": req.MessageID})

	// 1. Identify Tenant (Needed for schema isolation in River)
	var tenantID uuid.UUID
	if err := db.GormDB.Model(&models.Message{}).Select("tenant_id").Where("id = ?", msgID).Scan(&tenantID).Error; err != nil {
		logger.GetLogger().Error("DLR: Failed to find tenant for message", zap.String("msg_id", req.MessageID), zap.Error(err))
		return
	}

	// 2. Insert into Queue
	if dispatcher.GlobalManager != nil {
		client, err := dispatcher.GlobalManager.GetRiverClient(tenantID)
		if err == nil && client != nil {
			_, err = client.Insert(c.Request.Context(), queue.DLRProcessArgs{
				MessageID:  msgID,
				NewStatus:  newStatus,
				ExternalID: req.ExternalID,
				TenantID:   tenantID,
			}, nil)
			if err != nil {
				logger.GetLogger().Error("DLR: Failed to insert queue job", zap.Error(err))
			}
		}
	}
}


// mapStatus maps provider-specific status strings to our internal MessageStatus
func mapStatus(status string) models.MessageStatus {
	switch status {
	case "SENT", "sent":
		return models.MessageSent
	case "DELIVERED", "delivered", "DELIVRD":
		return models.MessageDelivered
	case "FAILED", "failed", "UNDELIVERABLE", "UNDELIV":
		return models.MessageFailed
	default:
		return ""
	}
}
