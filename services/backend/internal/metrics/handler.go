package metrics

import (
	"net/http"
	"time"

	"github.com/al-hudhud/backend/internal/auth"
	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Handler provides HTTP endpoints for metrics
type Handler struct{}

// NewHandler creates a new metrics handler
func NewHandler() *Handler {
	return &Handler{}
}

// PrometheusHandler returns Prometheus metrics endpoint
func (h *Handler) PrometheusHandler(c *gin.Context) {
	handler := promhttp.Handler()
	handler.ServeHTTP(c.Writer, c.Request)
}

// GetMetricsHandler returns current system metrics for a tenant
func (h *Handler) GetMetricsHandler(c *gin.Context) {
	tenantID, ok := auth.GetTenantIDOrAbort(c)
	if !ok {
		return
	}

	// Calculate metrics in real-time from the database for this specific tenant
	var metrics struct {
		TotalSent      int64   `json:"total_sent"`
		TotalDelivered int64   `json:"total_delivered"`
		TotalFailed    int64   `json:"total_failed"`
		TotalWaiting   int64   `json:"total_waiting"`
		SuccessRate    float64 `json:"success_rate"`
	}

	// Use the GORM DB from context (tenant isolated via middleware)
	tx := db.Get(c)

	// Count messages by status
	tx.Model(&models.Message{}).Where("tenant_id = ?", tenantID).Count(&metrics.TotalSent) // Actually total created
	tx.Model(&models.Message{}).Where("tenant_id = ? AND status = ?", tenantID, models.MessageSent).Count(&metrics.TotalSent)
	tx.Model(&models.Message{}).Where("tenant_id = ? AND status = ?", tenantID, models.MessageDelivered).Count(&metrics.TotalDelivered)
	tx.Model(&models.Message{}).Where("tenant_id = ? AND status = ?", tenantID, models.MessageFailed).Count(&metrics.TotalFailed)
	tx.Model(&models.Message{}).Where("tenant_id = ? AND status = ?", tenantID, models.MessageWait).Count(&metrics.TotalWaiting)

	if metrics.TotalSent > 0 {
		metrics.SuccessRate = float64(metrics.TotalDelivered) / float64(metrics.TotalSent) * 100
	}

	c.JSON(http.StatusOK, gin.H{
		"messages":     metrics,
		"last_updated": time.Now().Format(time.RFC3339),
	})
}

// GetHealthHandler returns system health status
func (h *Handler) GetHealthHandler(c *gin.Context) {
	metrics := GetMetrics()
	snapshot := metrics.GetSnapshot()

	// Simple health check logic
	isHealthy := true
	issues := []string{}

	// Check if failure rate is too high
	if snapshot.FailureRate() > 10.0 {
		isHealthy = false
		issues = append(issues, "High failure rate detected")
	}

	// Check if too many tenants in wait mode
	if snapshot.TenantsInWaitMode > 10 {
		isHealthy = false
		issues = append(issues, "Many tenants in wait mode")
	}

	status := "healthy"
	if !isHealthy {
		status = "degraded"
	}

	c.JSON(http.StatusOK, gin.H{
		"status": status,
		"issues": issues,
		"metrics": gin.H{
			"messages_waiting": snapshot.TotalMessagesWaiting,
			"failure_rate":     snapshot.FailureRate(),
			"tenants_waiting":  snapshot.TenantsInWaitMode,
		},
	})
}
