package api

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/al-hudhud/backend/internal/admin/distribution"
	"github.com/gin-gonic/gin"
)

// AdminDistributionHandler handles admin distribution API endpoints
type AdminDistributionHandler struct {
	poolManager *distribution.DevicePoolManager
}

// NewAdminDistributionHandler creates a new admin distribution handler
func NewAdminDistributionHandler(poolManager *distribution.DevicePoolManager) *AdminDistributionHandler {
	return &AdminDistributionHandler{
		poolManager: poolManager,
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
