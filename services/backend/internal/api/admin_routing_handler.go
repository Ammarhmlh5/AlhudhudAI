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

// GetRoutingConfigByTenant handles GET /api/v1/admin/routing/:tenant_id
func (h *AdminRoutingHandler) GetRoutingConfigByTenant(c *gin.Context) {
	tenantID, ok := parseTenantIDParam(c, "tenant_id")
	if !ok {
		return
	}

	config, err := h.configManager.GetRoutingConfig(c.Request.Context(), tenantID)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "internal_error", "Failed to retrieve routing config", nil)
		return
	}

	c.JSON(http.StatusOK, config)
}

// UpdateRoutingConfigByTenant handles PUT /api/v1/admin/routing/:tenant_id
func (h *AdminRoutingHandler) UpdateRoutingConfigByTenant(c *gin.Context) {
	tenantID, ok := parseTenantIDParam(c, "tenant_id")
	if !ok {
		return
	}

	var input struct {
		RoutingType           string  `json:"routing_type"`
		DedicatedServerID     string  `json:"dedicated_server_id"`
		DedicatedServerURL    *string `json:"dedicated_server_url"`
		DedicatedServerAPIKey *string `json:"dedicated_server_api_key"`
		IsActive              *bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_json", "Invalid JSON format", nil)
		return
	}

	var serverID *uuid.UUID
	if input.DedicatedServerID != "" {
		parsed, err := uuid.Parse(input.DedicatedServerID)
		if err != nil {
			RespondWithError(c, http.StatusBadRequest, "invalid_server_id", "Invalid dedicated server ID format", nil)
			return
		}
		serverID = &parsed
	}

	updateInput := &routing.UpdateRoutingConfigInput{
		TenantID:              tenantID,
		RoutingType:           input.RoutingType,
		DedicatedServerID:     serverID,
		DedicatedServerURL:    input.DedicatedServerURL,
		DedicatedServerAPIKey: input.DedicatedServerAPIKey,
		IsActive:              input.IsActive,
	}

	updatedConfig, err := h.configManager.UpdateRoutingConfig(c.Request.Context(), updateInput)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "validation_failed", err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, updatedConfig)
}

// GetRoutingMetricsByTenant handles GET /api/v1/admin/routing/:tenant_id/metrics
func (h *AdminRoutingHandler) GetRoutingMetricsByTenant(c *gin.Context) {
	tenantID, ok := parseTenantIDParam(c, "tenant_id")
	if !ok {
		return
	}

	stats, err := h.routingEngine.GetRoutingStats(c.Request.Context(), tenantID)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "internal_error", "Failed to retrieve routing metrics", nil)
		return
	}

	metrics := routing.RoutingMetrics{
		TenantID:           stats.TenantID,
		TotalRequests:      stats.TotalRequests,
		SuccessfulRequests: stats.SuccessfulRequests,
		FailedRequests:     stats.FailedRequests,
		SuccessRate:        stats.SuccessRate,
		AverageLatencyMs:   stats.AverageLatencyMs,
	}

	c.JSON(http.StatusOK, metrics)
}

// parseTenantIDParam parses a UUID path parameter, responding with an error on failure.
func parseTenantIDParam(c *gin.Context, param string) (uuid.UUID, bool) {
	tenantID, err := uuid.Parse(c.Param(param))
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_tenant_id", "Invalid tenant ID format", nil)
		return uuid.Nil, false
	}
	return tenantID, true
}
