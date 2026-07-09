package channel

import (
	"net/http"
	"strings"

	"github.com/al-hudhud/backend/internal/auth"
	"github.com/al-hudhud/backend/internal/redis"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Handler exposes HTTP endpoints for dispatch channel management.
type Handler struct {
	service *Service
}

func NewHandler(channelCache *redis.ChannelCache, logger *zap.Logger) *Handler {
	return &Handler{service: NewService(channelCache, logger)}
}

// GetService returns the underlying service for direct access (e.g., for starting health checks)
func (h *Handler) GetService() *Service {
	return h.service
}

// List returns all channels for the authenticated tenant.
// GET /api/v1/channels/list
func (h *Handler) List(c *gin.Context) {
	tenantID, ok := auth.GetTenantIDOrAbort(c)
	if !ok {
		return
	}

	channels, err := h.service.ListChannels(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, channels)
}

// Create adds a new dispatch channel for the authenticated tenant.
// POST /api/v1/channels/create
func (h *Handler) Create(c *gin.Context) {
	tenantID, ok := auth.GetTenantIDOrAbort(c)
	if !ok {
		return
	}

	var req CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	ch, err := h.service.CreateChannel(c.Request.Context(), tenantID, req)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, gin.H{"error": "Channel already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, ch)
}

// Update modifies priority, status, or config of a channel.
// PUT /api/v1/channels/:id
func (h *Handler) Update(c *gin.Context) {
	tenantID, ok := auth.GetTenantIDOrAbort(c)
	if !ok {
		return
	}

	channelIDStr := c.Param("id")
	channelID, err := uuid.Parse(channelIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	var req UpdateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	ch, err := h.service.UpdateChannel(c.Request.Context(), tenantID, channelID, req)
	if err != nil {
		if err.Error() == "failed to update channel: channel not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ch)
}

// Delete removes a dispatch channel.
// DELETE /api/v1/channels/:id
func (h *Handler) Delete(c *gin.Context) {
	tenantID, ok := auth.GetTenantIDOrAbort(c)
	if !ok {
		return
	}

	channelIDStr := c.Param("id")
	channelID, err := uuid.Parse(channelIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	if err := h.service.DeleteChannel(c.Request.Context(), tenantID, channelID); err != nil {
		if err.Error() == "failed to delete channel: channel not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Channel deleted successfully"})
}

// RegenerateKey creates a new unique key for the channel.
func (h *Handler) RegenerateKey(c *gin.Context) {
	tenantID, ok := auth.GetTenantIDOrAbort(c)
	if !ok {
		return
	}

	channelIDStr := c.Param("id")
	channelID, err := uuid.Parse(channelIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	ch, err := h.service.RegenerateChannelKey(c.Request.Context(), tenantID, channelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"channel_key": ch.ChannelKey,
		"message":     "Channel key regenerated successfully",
	})
}
