package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/internal/ws"
	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PrivateGatewayHandler struct {
	hub *ws.Hub
}

func NewPrivateGatewayHandler(hub *ws.Hub) *PrivateGatewayHandler {
	return &PrivateGatewayHandler{hub: hub}
}

type DLRRequest struct {
	MessageID string `json:"message_id" binding:"required"`
	Status    string `json:"status" binding:"required"` // DELIVERED, FAILED
	ErrorCode string `json:"error_code,omitempty"`
}

// HandleDLR receives Delivery Reports from the tenant's private server
func (h *PrivateGatewayHandler) HandleDLR(c *gin.Context) {
	tenantIDRaw, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant context missing or unauthorized"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDRaw.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid tenant ID format"})
		return
	}

	var req DLRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	// Validate status
	if req.Status != string(models.MessageDelivered) && req.Status != string(models.MessageFailed) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status value. Allowed values: DELIVERED, FAILED"})
		return
	}

	messageID, err := uuid.Parse(req.MessageID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID format"})
		return
	}

	// Update the message status in the database
	database := db.Get(c.Request.Context())
	
	// Fast lock-free update ensuring message belongs to the tenant
	result := database.Model(&models.Message{}).
		Where("id = ? AND tenant_id = ?", messageID, tenantID).
		Updates(map[string]interface{}{
			"status":        req.Status,
			"error_details": req.ErrorCode,
		})

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Message not found or does not belong to your tenant"})
			return
		}
		logger.GetLogger().Error("Failed to update message status from Private Gateway",
			zap.String("message_id", req.MessageID),
			zap.Error(result.Error),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error while updating status"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
		return
	}

	logger.GetLogger().Info("Private Gateway DLR processed",
		zap.String("message_id", req.MessageID),
		zap.String("status", req.Status),
	)

	// Push live status update via WebSocket to tenant dashboard
	if h.hub != nil {
		wsMsg := ws.WebSocketMessage{
			Type:      "MESSAGE_STATUS_UPDATE",
			MessageID: req.MessageID,
			TenantID:  tenantID,
			Timestamp: time.Now(),
			Payload: map[string]string{
				"status":        req.Status,
				"error_details": req.ErrorCode,
				"gateway":       "PRIVATE",
			},
		}
		if err := h.hub.BroadcastToTenant(c.Request.Context(), tenantID, wsMsg); err != nil {
			logger.GetLogger().Warn("Failed to broadcast DLR status via WebSocket",
				zap.String("tenant_id", tenantID.String()),
				zap.Error(err),
			)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Delivery report processed successfully",
		"status":  "success",
	})
}
