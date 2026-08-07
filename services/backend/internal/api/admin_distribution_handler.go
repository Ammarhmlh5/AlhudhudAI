package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/al-hudhud/backend/internal/admin/distribution"
	"github.com/gin-gonic/gin"
)

// AdminDistributionHandler handles admin distribution API endpoints
type AdminDistributionHandler struct {
	poolManager      *distribution.DevicePoolManager
	strategySelector *distribution.StrategySelector
}

// NewAdminDistributionHandler creates a new admin distribution handler
func NewAdminDistributionHandler(poolManager *distribution.DevicePoolManager, strategySelector *distribution.StrategySelector) *AdminDistributionHandler {
	return &AdminDistributionHandler{
		poolManager:      poolManager,
		strategySelector: strategySelector,
	}
}

// GetPools handles GET /api/v1/admin/distribution/pools
func (h *AdminDistributionHandler) GetPools(c *gin.Context) {
	tenantIDStr := c.Query("tenant_id")
	
	if tenantIDStr == "" {
		RespondWithError(c, http.StatusBadRequest, "missing_tenant_id", "Tenant ID is required for pools", nil)
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_tenant_id", "Invalid tenant ID format", nil)
		return
	}

	devices, err := h.poolManager.GetAllDevices(c.Request.Context(), tenantID)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "internal_error", "Failed to retrieve devices", nil)
		return
	}

	pool := distribution.DevicePool{
		TenantID: tenantID,
		Devices:  devices,
	}

	RespondWithSuccess(c, http.StatusOK, []distribution.DevicePool{pool}, "")
}

// GetPoolDetails handles GET /api/v1/admin/distribution/pools/:pool_id
func (h *AdminDistributionHandler) GetPoolDetails(c *gin.Context) {
	// In this implementation, pool_id is effectively the tenant_id
	poolIDStr := c.Param("pool_id")
	poolID, err := uuid.Parse(poolIDStr)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_pool_id", "Invalid pool ID format", nil)
		return
	}

	devices, err := h.poolManager.GetAllDevices(c.Request.Context(), poolID)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "internal_error", "Failed to retrieve devices", nil)
		return
	}

	pool := distribution.DevicePool{
		TenantID: poolID,
		Devices:  devices,
	}

	RespondWithSuccess(c, http.StatusOK, pool, "")
}

// GetConfig handles GET /api/v1/admin/distribution/:tenant_id
func (h *AdminDistributionHandler) GetConfig(c *gin.Context) {
	tenantID, ok := parseTenantIDParam(c, "tenant_id")
	if !ok {
		return
	}

	config, err := h.strategySelector.GetStrategyConfig(c.Request.Context(), tenantID)
	if err != nil {
		if errors.Is(err, distribution.ErrStrategyNotConfigured) {
			c.JSON(http.StatusOK, distribution.DistributionConfig{
				TenantID:       tenantID,
				StrategyType:   distribution.StrategyRoundRobin,
				StrategyConfig: map[string]interface{}{},
				IsActive:       true,
			})
			return
		}
		RespondWithError(c, http.StatusInternalServerError, "internal_error", "Failed to retrieve distribution config", nil)
		return
	}

	c.JSON(http.StatusOK, config)
}

// UpdateConfig handles PUT /api/v1/admin/distribution/:tenant_id
func (h *AdminDistributionHandler) UpdateConfig(c *gin.Context) {
	tenantID, ok := parseTenantIDParam(c, "tenant_id")
	if !ok {
		return
	}

	var input struct {
		StrategyType   string                 `json:"strategy_type"`
		StrategyConfig map[string]interface{} `json:"strategy_config"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_json", "Invalid JSON format", nil)
		return
	}

	if input.StrategyConfig == nil {
		input.StrategyConfig = map[string]interface{}{}
	}

	strategyType := distribution.StrategyType(input.StrategyType)

	var customRules map[string]interface{}
	if strategyType == distribution.StrategyCustom {
		if rules, ok := input.StrategyConfig["custom_rules"].(map[string]interface{}); ok {
			customRules = rules
		} else {
			customRules = input.StrategyConfig
		}
	}

	err := h.strategySelector.UpdateStrategyConfig(c.Request.Context(), tenantID, strategyType, input.StrategyConfig, customRules)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "validation_failed", err.Error(), nil)
		return
	}

	config, err := h.strategySelector.GetStrategyConfig(c.Request.Context(), tenantID)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "internal_error", "Failed to retrieve updated distribution config", nil)
		return
	}

	c.JSON(http.StatusOK, config)
}

// GetDevices handles GET /api/v1/admin/distribution/:tenant_id/devices
func (h *AdminDistributionHandler) GetDevices(c *gin.Context) {
	tenantID, ok := parseTenantIDParam(c, "tenant_id")
	if !ok {
		return
	}

	devices, err := h.poolManager.GetAllDevices(c.Request.Context(), tenantID)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "internal_error", "Failed to retrieve devices", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"devices": devices})
}

// GetMetrics handles GET /api/v1/admin/distribution/:tenant_id/metrics
func (h *AdminDistributionHandler) GetMetrics(c *gin.Context) {
	tenantID, ok := parseTenantIDParam(c, "tenant_id")
	if !ok {
		return
	}

	end := time.Now()
	start := end.AddDate(0, 0, -7)

	metrics, err := h.poolManager.GetTenantMetrics(c.Request.Context(), tenantID, start, end)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "internal_error", "Failed to retrieve distribution metrics", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"metrics": metrics})
}
