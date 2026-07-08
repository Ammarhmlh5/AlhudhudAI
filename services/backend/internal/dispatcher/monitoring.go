package dispatcher

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// MonitoringHandler provides monitoring endpoints for dispatchers
type MonitoringHandler struct {
	manager *Manager
}

// NewMonitoringHandler creates a new monitoring handler
func NewMonitoringHandler(manager *Manager) *MonitoringHandler {
	return &MonitoringHandler{
		manager: manager,
	}
}

// GetStats returns statistics for all dispatchers
func (h *MonitoringHandler) GetStats(c *gin.Context) {
	stats := h.manager.GetStats()
	c.JSON(http.StatusOK, stats)
}

// GetTenantStats returns statistics for a specific tenant
func (h *MonitoringHandler) GetTenantStats(c *gin.Context) {
	tenantID := c.Param("tenant_id")

	h.manager.mu.RLock()
	defer h.manager.mu.RUnlock()

	dispatcher, exists := h.manager.dispatchers[mustParseTenantID(tenantID)]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tenant dispatcher not found"})
		return
	}

	stats := dispatcher.GetStats()
	c.JSON(http.StatusOK, stats)
}

// HealthCheck returns health status of the dispatcher system
func (h *MonitoringHandler) HealthCheck(c *gin.Context) {
	h.manager.mu.RLock()
	activeCount := len(h.manager.dispatchers)
	h.manager.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"status":             "healthy",
		"active_dispatchers": activeCount,
		"system":             "tenant_isolation",
	})
}
