package audit

import (
	"net/http"
	"strconv"
	"time"

	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Handler handles HTTP requests for audit logs
type Handler struct {
	logger *Logger
}

// NewHandler creates a new audit handler
func NewHandler(l *Logger) *Handler {
	return &Handler{
		logger: l,
	}
}

// GetAuditLogs returns a paginated list of audit logs with filters
func (h *Handler) GetAuditLogs(c *gin.Context) {
	filters := AuditQueryFilters{}

	// Basic pagination
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")
	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)
	filters.Limit = limit
	filters.Offset = offset

	// Filters
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if uid, err := uuid.Parse(userIDStr); err == nil {
			filters.UserID = &uid
		}
	}

	if tenantIDStr := c.Query("tenant_id"); tenantIDStr != "" {
		if tid, err := uuid.Parse(tenantIDStr); err == nil {
			filters.TenantID = &tid
		}
	}

	filters.Action = c.Query("action")
	filters.Resource = c.Query("resource")
	filters.ResourceID = c.Query("resource_id")
	filters.Result = c.Query("result")

	// Date filters
	if startStr := c.Query("start_date"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			filters.StartDate = &t
		}
	}

	if endStr := c.Query("end_date"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			filters.EndDate = &t
		}
	}

	entries, err := h.logger.Query(filters)
	if err != nil {
		logger.GetLogger().Error("Failed to query audit logs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve audit logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"entries": entries,
		"filters": filters,
		"count":   len(entries),
	})
}
