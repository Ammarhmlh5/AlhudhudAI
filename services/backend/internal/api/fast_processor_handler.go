package api

import (
	"fmt"
	"net/http"

	"github.com/al-hudhud/backend/internal/auth"
	"github.com/al-hudhud/backend/internal/redis"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// FastProcessorHandler handles fast processor statistics and monitoring
type FastProcessorHandler struct {
	messageQueue *redis.MessageQueue
	logger       *zap.Logger
}

// NewFastProcessorHandler creates a new handler
func NewFastProcessorHandler(messageQueue *redis.MessageQueue, logger *zap.Logger) *FastProcessorHandler {
	return &FastProcessorHandler{
		messageQueue: messageQueue,
		logger:       logger,
	}
}

// GetQueueStats returns queue statistics for the tenant
func (h *FastProcessorHandler) GetQueueStats(c *gin.Context) {
	tenantID, ok := auth.GetTenantIDOrAbort(c)
	if !ok {
		return
	}

	if h.messageQueue == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Fast processor not available"})
		return
	}

	stats, err := h.messageQueue.GetQueueStats(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Error("Failed to get queue stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get queue stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"queue_stats": stats,
		"tenant_id":   tenantID.String(),
	})
}

// GetQueueSize returns the current queue size
func (h *FastProcessorHandler) GetQueueSize(c *gin.Context) {
	tenantID, ok := auth.GetTenantIDOrAbort(c)
	if !ok {
		return
	}

	if h.messageQueue == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Fast processor not available"})
		return
	}

	size, err := h.messageQueue.GetQueueSize(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Error("Failed to get queue size", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get queue size"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"queue_size": size,
		"tenant_id":  tenantID.String(),
	})
}

// PeekQueue returns messages in queue without removing them
func (h *FastProcessorHandler) PeekQueue(c *gin.Context) {
	tenantID, ok := auth.GetTenantIDOrAbort(c)
	if !ok {
		return
	}

	if h.messageQueue == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Fast processor not available"})
		return
	}

	count := 10
	if c.Query("count") != "" {
		var err error
		if _, err = fmt.Sscanf(c.Query("count"), "%d", &count); err != nil {
			count = 10
		}
	}

	messages, err := h.messageQueue.Peek(c.Request.Context(), tenantID, count)
	if err != nil {
		h.logger.Error("Failed to peek queue", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to peek queue"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"messages":  messages,
		"count":     len(messages),
		"tenant_id": tenantID.String(),
	})
}

// ClearQueue clears all messages from queue (admin only)
func (h *FastProcessorHandler) ClearQueue(c *gin.Context) {
	tenantIDStr := c.Param("tenant_id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	if h.messageQueue == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Fast processor not available"})
		return
	}

	if err := h.messageQueue.Clear(c.Request.Context(), tenantID); err != nil {
		h.logger.Error("Failed to clear queue", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear queue"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Queue cleared successfully",
		"tenant_id": tenantID.String(),
	})
}
