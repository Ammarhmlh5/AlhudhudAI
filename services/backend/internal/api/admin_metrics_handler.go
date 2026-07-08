package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/al-hudhud/backend/internal/admin/metrics"
	"github.com/gin-gonic/gin"
)

// AdminMetricsHandler handles admin metrics API endpoints
type AdminMetricsHandler struct {
	metricsRepo *metrics.Repository
}

// NewAdminMetricsHandler creates a new admin metrics handler
func NewAdminMetricsHandler(metricsRepo *metrics.Repository) *AdminMetricsHandler {
	return &AdminMetricsHandler{
		metricsRepo: metricsRepo,
	}
}

// GetCompanyMetrics handles GET /api/admin/companies/{tenant_id}/metrics
func (h *AdminMetricsHandler) GetCompanyMetrics(c *gin.Context) {
	tenantIDStr := c.Param("tenant_id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_tenant_id", "Invalid tenant ID format", nil)
		return
	}

	filter := metrics.MetricsFilter{
		TenantID: tenantID,
		Limit:    100,
		Offset:   0,
	}

	// Parse period_type
	if periodType := c.Query("period_type"); periodType != "" {
		if periodType != "hourly" && periodType != "daily" && periodType != "monthly" {
			RespondWithError(c, http.StatusBadRequest, "invalid_period_type", 
				"Invalid period_type. Must be: hourly, daily, or monthly", nil)
			return
		}
		filter.PeriodType = &periodType
	}

	// Parse period_start
	if periodStartStr := c.Query("period_start"); periodStartStr != "" {
		periodStart, err := time.Parse(time.RFC3339, periodStartStr)
		if err != nil {
			RespondWithError(c, http.StatusBadRequest, "invalid_period_start", 
				"Invalid period_start format. Use RFC3339 format", nil)
			return
		}
		filter.StartTime = &periodStart
	}

	// Parse period_end
	if periodEndStr := c.Query("period_end"); periodEndStr != "" {
		periodEnd, err := time.Parse(time.RFC3339, periodEndStr)
		if err != nil {
			RespondWithError(c, http.StatusBadRequest, "invalid_period_end", 
				"Invalid period_end format. Use RFC3339 format", nil)
			return
		}
		filter.EndTime = &periodEnd
	}

	// Parse pagination
	if limitStr := c.Query("limit"); limitStr != "" {
		var limit int
		if _, err := fmt.Sscanf(limitStr, "%d", &limit); err == nil && limit > 0 && limit <= 1000 {
			filter.Limit = limit
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		var offset int
		if _, err := fmt.Sscanf(offsetStr, "%d", &offset); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	// Get metrics
	metricsData, err := h.metricsRepo.GetMetrics(c.Request.Context(), filter)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "internal_error", 
			"Failed to retrieve metrics", nil)
		return
	}

	RespondWithSuccess(c, http.StatusOK, map[string]interface{}{
		"tenant_id": tenantID,
		"metrics":   metricsData,
		"count":     len(metricsData),
		"limit":     filter.Limit,
		"offset":    filter.Offset,
		"filters": map[string]interface{}{
			"period_type":  filter.PeriodType,
			"period_start": filter.StartTime,
			"period_end":   filter.EndTime,
		},
	}, "")
}
