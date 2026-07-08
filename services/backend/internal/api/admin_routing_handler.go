package api

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/al-hudhud/backend/internal/admin/routing"
	"github.com/gin-gonic/gin"
)

// AdminRoutingHandler handles admin routing API endpoints
type AdminRoutingHandler struct {
	routingEngine *routing.RoutingEngine
	configManager *routing.RoutingConfigManager
}

// NewAdminRoutingHandler creates a new admin routing handler
func NewAdminRoutingHandler(routingEngine *routing.RoutingEngine, configManager *routing.RoutingConfigManager) *AdminRoutingHandler {
	return &AdminRoutingHandler{
		routingEngine: routingEngine,
		configManager: configManager,
	}
}

// GetRoutingStats handles GET /api/v1/admin/routing/stats
func (h *AdminRoutingHandler) GetRoutingStats(c *gin.Context) {
	tenantIDStr := c.Query("tenant_id")
	
	if tenantIDStr == "" {
		RespondWithError(c, http.StatusBadRequest, "missing_tenant_id", "Tenant ID is required", nil)
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_tenant_id", "Invalid tenant ID format", nil)
		return
	}

	stats, err := h.routingEngine.GetRoutingStats(c.Request.Context(), tenantID)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "internal_error", "Failed to retrieve routing stats", nil)
		return
	}

	RespondWithSuccess(c, http.StatusOK, stats, "")
}

// UpdateRoutingConfig handles PUT /api/v1/admin/routing/config
func (h *AdminRoutingHandler) UpdateRoutingConfig(c *gin.Context) {
	var input routing.UpdateRoutingConfigInput
	if err := c.ShouldBindJSON(&input); err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_json", "Invalid JSON format", nil)
		return
	}

	updatedConfig, err := h.configManager.UpdateRoutingConfig(c.Request.Context(), &input)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "internal_error", "Failed to update routing config", nil)
		return
	}

	RespondWithSuccess(c, http.StatusOK, updatedConfig, "Routing config updated successfully")
}
