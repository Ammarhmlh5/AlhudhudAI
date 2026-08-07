package device

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/al-hudhud/backend/internal/auth"
	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Handler struct {
	service *Service
}

func NewHandler() *Handler {
	return &Handler{
		service: NewService(),
	}
}

// Pair handles device pairing requests
func (h *Handler) Pair(c *gin.Context) {
	tenantID, ok := auth.GetTenantIDOrAbort(c)
	if !ok {
		return
	}

	var req PairDeviceRequest
	
	// DEBUG
	if c.Request.Body != nil {
		bodyBytes, _ := io.ReadAll(c.Request.Body)
		fmt.Printf("[DEBUG] Pair handler body len: %d\n", len(bodyBytes))
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.GetLogger().Error("Pairing request binding failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pairKey, err := h.service.PairDevice(c, tenantID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device_pair_key": pairKey,
		"message":         "Device paired successfully. Use this key in your mobile app.",
	})
}

// List returns all devices for the current tenant
func (h *Handler) List(c *gin.Context) {
	tenantID, ok := auth.GetTenantIDOrAbort(c)
	if !ok {
		return
	}
	devices, err := h.service.ListDevices(c, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, devices)
}

// FetchMessagesHandler handles requests from mobile devices to fetch pending messages
func (h *Handler) FetchMessagesHandler(c *gin.Context) {
	tenantID, ok := auth.GetTenantIDOrAbort(c)
	if !ok {
		return
	}

	deviceIDStr := c.Query("device_id")
	if deviceIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}

	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device_id format"})
		return
	}

	messages, err := h.service.FetchMessages(c, tenantID, deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": messages,
		"count":    len(messages),
	})
}

// ReportStatusHandler handles delivery reports from mobile devices
func (h *Handler) ReportStatusHandler(c *gin.Context) {
	tenantID, ok := auth.GetTenantIDOrAbort(c)
	if !ok {
		return
	}

	var req struct {
		MessageID   uuid.UUID  `json:"message_id" binding:"required"`
		Status      string     `json:"status" binding:"required"` // SENT, DELIVERED, FAILED
		DeliveredAt *time.Time `json:"delivered_at"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	err := h.service.ReportStatus(c, tenantID, req.MessageID, req.Status, req.DeliveredAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Status reported successfully"})
}

// Delete removes a paired device
func (h *Handler) Delete(c *gin.Context) {
	tenantID, ok := auth.GetTenantIDOrAbort(c)
	if !ok {
		return
	}

	deviceIDStr := c.Param("id")
	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		// Try deleting by hardware_id if it's not a UUID
		err = h.service.DeleteDeviceByHardwareID(c, tenantID, deviceIDStr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		err = h.service.DeleteDevice(c, tenantID, deviceID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Device deleted successfully"})
}

// Ping checks if the device is connected
func (h *Handler) Ping(c *gin.Context) {
	tenantID, ok := auth.GetTenantIDOrAbort(c)
	if !ok {
		return
	}

	deviceIDStr := c.Param("id")
	
	status, err := h.service.PingDevice(c, tenantID, deviceIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": status, // "ONLINE" or "OFFLINE"
		"message": "Ping completed",
	})
}
