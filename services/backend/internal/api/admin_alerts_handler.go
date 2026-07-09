package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/al-hudhud/backend/internal/admin/alerts"
	"github.com/gin-gonic/gin"
)

// AdminAlertsHandler handles admin alerts API endpoints
type AdminAlertsHandler struct {
	alertManager *alerts.Manager
}

// NewAdminAlertsHandler creates a new admin alerts handler
func NewAdminAlertsHandler(alertManager *alerts.Manager) *AdminAlertsHandler {
	return &AdminAlertsHandler{
		alertManager: alertManager,
	}
}

// GetAlerts handles GET /api/admin/alerts
func (h *AdminAlertsHandler) GetAlerts(c *gin.Context) {
	filter := alerts.AlertFilter{
		Limit:  100,
		Offset: 0,
	}

	// Parse tenant_id filter
	if tenantIDStr := c.Query("tenant_id"); tenantIDStr != "" {
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			RespondWithError(c, http.StatusBadRequest, "invalid_tenant_id", "Invalid tenant ID format", nil)
			return
		}
		filter.TenantID = &tenantID
	}

	// Parse severity filter
	if severity := c.Query("severity"); severity != "" {
		if severity != "critical" && severity != "warning" && severity != "info" {
			RespondWithError(c, http.StatusBadRequest, "invalid_severity", 
				"Invalid severity. Must be: critical, warning, or info", nil)
			return
		}
		severityVal := alerts.AlertSeverity(severity)
		filter.Severity = &severityVal
	}

	// Parse type filter
	if alertType := c.Query("type"); alertType != "" {
		validTypes := map[string]bool{
			"rating_change":        true,
			"server_failure":       true,
			"threshold_violation":  true,
			"device_failure":       true,
			"timing_violation":     true,
			"enforcement_action":   true,
		}
		if !validTypes[alertType] {
			RespondWithError(c, http.StatusBadRequest, "invalid_type", 
				"Invalid alert type", nil)
			return
		}
		alertTypeVal := alerts.AlertType(alertType)
		filter.AlertType = &alertTypeVal
	}

	// Parse status filter
	if status := c.Query("status"); status != "" {
		if status != "active" && status != "acknowledged" && status != "resolved" {
			RespondWithError(c, http.StatusBadRequest, "invalid_status", 
				"Invalid status. Must be: active, acknowledged, or resolved", nil)
			return
		}
		statusVal := alerts.AlertStatus(status)
		filter.Status = &statusVal
	}

	// Parse time range
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		startTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			RespondWithError(c, http.StatusBadRequest, "invalid_start_time", 
				"Invalid start_time format. Use RFC3339 format", nil)
			return
		}
		filter.StartTime = &startTime
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		endTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			RespondWithError(c, http.StatusBadRequest, "invalid_end_time", 
				"Invalid end_time format. Use RFC3339 format", nil)
			return
		}
		filter.EndTime = &endTime
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

	// Get alerts
	alertsList, err := h.alertManager.GetAlerts(c.Request.Context(), &filter)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "internal_error", 
			"Failed to retrieve alerts", nil)
		return
	}

	RespondWithSuccess(c, http.StatusOK, map[string]interface{}{
		"alerts": alertsList.Alerts,
		"count":  alertsList.TotalCount,
		"limit":  filter.Limit,
		"offset": filter.Offset,
		"filters": map[string]interface{}{
			"tenant_id":  filter.TenantID,
			"severity":   filter.Severity,
			"type":       filter.AlertType,
			"status":     filter.Status,
			"start_time": filter.StartTime,
			"end_time":   filter.EndTime,
		},
	}, "")
}

// GetAlert handles GET /api/admin/alerts/{alert_id}
func (h *AdminAlertsHandler) GetAlert(c *gin.Context) {
	alertIDStr := c.Param("alert_id")
	alertID, err := strconv.ParseInt(alertIDStr, 10, 64)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_alert_id", "Invalid alert ID format", nil)
		return
	}

	// Get alert details
	alert, err := h.alertManager.GetAlertByID(c.Request.Context(), alertID, uuid.Nil)
	if err != nil {
		if err.Error() == "alert not found" {
			RespondWithError(c, http.StatusNotFound, "alert_not_found", 
				"Alert not found", nil)
			return
		}
		RespondWithError(c, http.StatusInternalServerError, "internal_error", 
			"Failed to retrieve alert", nil)
		return
	}

	RespondWithSuccess(c, http.StatusOK, alert, "")
}

// AcknowledgeAlert handles PUT /api/admin/alerts/{alert_id}/acknowledge
func (h *AdminAlertsHandler) AcknowledgeAlert(c *gin.Context) {
	alertIDStr := c.Param("alert_id")
	alertID, err := strconv.ParseInt(alertIDStr, 10, 64)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_alert_id", "Invalid alert ID format", nil)
		return
	}

	var input struct {
		AcknowledgedBy uuid.UUID `json:"acknowledged_by"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_json", "Invalid JSON format", nil)
		return
	}

	// Acknowledge alert
	err = h.alertManager.AcknowledgeAlert(c.Request.Context(), alertID, input.AcknowledgedBy)
	if err != nil {
		if err.Error() == "alert not found" {
			RespondWithError(c, http.StatusNotFound, "alert_not_found", 
				"Alert not found", nil)
			return
		}
		RespondWithError(c, http.StatusInternalServerError, "internal_error", 
			"Failed to acknowledge alert", nil)
		return
	}
	alert, _ := h.alertManager.GetAlertByID(c.Request.Context(), alertID, uuid.Nil)

	RespondWithSuccess(c, http.StatusOK, alert, "Alert acknowledged successfully")
}

// ResolveAlert handles PUT /api/admin/alerts/{alert_id}/resolve
func (h *AdminAlertsHandler) ResolveAlert(c *gin.Context) {
	alertIDStr := c.Param("alert_id")
	alertID, err := strconv.ParseInt(alertIDStr, 10, 64)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_alert_id", "Invalid alert ID format", nil)
		return
	}

	// Resolve alert
	err = h.alertManager.ResolveAlert(c.Request.Context(), alertID)
	if err != nil {
		if err.Error() == "alert not found" {
			RespondWithError(c, http.StatusNotFound, "alert_not_found", 
				"Alert not found", nil)
			return
		}
		RespondWithError(c, http.StatusInternalServerError, "internal_error", 
			"Failed to resolve alert", nil)
		return
	}
	alert, _ := h.alertManager.GetAlertByID(c.Request.Context(), alertID, uuid.Nil)

	RespondWithSuccess(c, http.StatusOK, alert, "Alert resolved successfully")
}

// ConfigureAlerts handles POST /api/admin/alerts/config
func (h *AdminAlertsHandler) ConfigureAlerts(c *gin.Context) {
	RespondWithError(c, http.StatusNotImplemented, "not_implemented", "Alert configuration is not yet implemented", nil)
}
