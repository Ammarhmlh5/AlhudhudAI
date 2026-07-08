package api

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/al-hudhud/backend/internal/admin/health"
	"github.com/gin-gonic/gin"
)

// AdminHealthHandler handles admin health API endpoints
type AdminHealthHandler struct {
	healthChecker *health.HealthChecker
}

// NewAdminHealthHandler creates a new admin health handler
func NewAdminHealthHandler(healthChecker *health.HealthChecker) *AdminHealthHandler {
	return &AdminHealthHandler{
		healthChecker: healthChecker,
	}
}

// GetServers handles GET /api/v1/admin/health/servers
func (h *AdminHealthHandler) GetServers(c *gin.Context) {
	status := c.Query("status")
	tenantIDStr := c.Query("tenant_id")

	var filterTenantID *uuid.UUID
	if tenantIDStr != "" {
		if uid, err := uuid.Parse(tenantIDStr); err == nil {
			filterTenantID = &uid
		}
	}

	var statusPtr *string
	if status != "" {
		statusPtr = &status
	}

	servers, err := h.healthChecker.GetAllServers(c.Request.Context(), statusPtr, filterTenantID)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "internal_error", "Failed to retrieve servers", nil)
		return
	}

	RespondWithSuccess(c, http.StatusOK, servers, "")
}

// GetServerDetails handles GET /api/v1/admin/health/servers/:server_id
func (h *AdminHealthHandler) GetServerDetails(c *gin.Context) {
	serverIDStr := c.Param("server_id")
	serverID, err := uuid.Parse(serverIDStr)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_server_id", "Invalid server ID format", nil)
		return
	}

	server, err := h.healthChecker.GetServer(c.Request.Context(), serverID)
	if err != nil {
		RespondWithError(c, http.StatusNotFound, "server_not_found", "Server not found", nil)
		return
	}

	RespondWithSuccess(c, http.StatusOK, server, "")
}

// GetHealthHistory handles GET /api/v1/admin/health/servers/:server_id/history
func (h *AdminHealthHandler) GetHealthHistory(c *gin.Context) {
	serverIDStr := c.Param("server_id")
	_, err := uuid.Parse(serverIDStr) // Validate UUID
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_server_id", "Invalid server ID format", nil)
		return
	}

	limitStr := c.DefaultQuery("limit", "100")
	_, _ = strconv.Atoi(limitStr) // Validate limit

	// History query not fully implemented in service layer yet.
	RespondWithError(c, http.StatusNotImplemented, "not_implemented", "Health history query not fully implemented in service layer", nil)
}
