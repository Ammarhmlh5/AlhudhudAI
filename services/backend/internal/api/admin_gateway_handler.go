package api

import (
	"net/http"

	"github.com/al-hudhud/backend/internal/admin/alerts"
	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/gin-gonic/gin"
)

// AdminGatewayHandler handles gateway (dispatch channel) management
type AdminGatewayHandler struct {
	alertManager *alerts.Manager
}

// NewAdminGatewayHandler creates a new admin gateway handler
func NewAdminGatewayHandler(alertManager *alerts.Manager) *AdminGatewayHandler {
	return &AdminGatewayHandler{
		alertManager: alertManager,
	}
}

// ListGateways handles GET /api/v1/admin/gateways
func (h *AdminGatewayHandler) ListGateways(c *gin.Context) {
	var channels []models.DispatchChannel
	if err := db.GormDB.Find(&channels).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list gateways"})
		return
	}
	c.JSON(http.StatusOK, channels)
}

// CreateGateway handles POST /api/v1/admin/gateways
func (h *AdminGatewayHandler) CreateGateway(c *gin.Context) {
	var channel models.DispatchChannel
	if err := c.ShouldBindJSON(&channel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := db.GormDB.Create(&channel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create gateway"})
		return
	}

	c.JSON(http.StatusCreated, channel)
}

// GetGatewayAlerts handles GET /api/v1/admin/gateways/alerts
func (h *AdminGatewayHandler) GetGatewayAlerts(c *gin.Context) {
	// Filter for gateway/device failure alerts
	alertType := alerts.AlertType("device_failure")
	filter := alerts.AlertFilter{
		AlertType: &alertType,
		Limit:     50,
	}

	alertsList, err := h.alertManager.GetAlerts(c.Request.Context(), &filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve gateway alerts"})
		return
	}

	c.JSON(http.StatusOK, alertsList.Alerts)
}

// GetGatewayQuotas handles GET /api/v1/admin/gateways/:id/quotas
func (h *AdminGatewayHandler) GetGatewayQuotas(c *gin.Context) {
	// Mock response for now as requested by UI
	c.JSON(http.StatusOK, gin.H{
		"daily_limit":   5000,
		"daily_used":    1200,
		"monthly_limit": 100000,
		"monthly_used":  45000,
		"status":        "healthy",
	})
}

// GetGatewayUsage handles GET /api/v1/admin/gateways/:id/usage
func (h *AdminGatewayHandler) GetGatewayUsage(c *gin.Context) {
	// Aggregating usage statistics from messages table
	id := c.Param("id")
	
	var stats struct {
		TotalSent     int64   `json:"total_sent"`
		TotalFailed   int64   `json:"total_failed"`
		SuccessRate   float64 `json:"success_rate"`
		AvgLatencyMs  int     `json:"avg_latency_ms"`
	}

	db.GormDB.Model(&models.Message{}).
		Where("channel_id = ?", id).
		Select("count(*) as total_sent, count(CASE WHEN status = 'FAILED' THEN 1 END) as total_failed").
		Scan(&stats)

	if stats.TotalSent > 0 {
		stats.SuccessRate = float64(stats.TotalSent-stats.TotalFailed) / float64(stats.TotalSent) * 100
	}
	stats.AvgLatencyMs = 1200 // Mock latency for now

	c.JSON(http.StatusOK, stats)
}
