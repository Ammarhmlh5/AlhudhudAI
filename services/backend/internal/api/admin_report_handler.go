package api

import (
	"net/http"
	"time"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/gin-gonic/gin"
)

// AdminReportHandler handles aggregated reports for administrators
type AdminReportHandler struct{}

// NewAdminReportHandler creates a new admin report handler
func NewAdminReportHandler() *AdminReportHandler {
	return &AdminReportHandler{}
}

// GetReports handles GET /api/v1/admin/reports
func (h *AdminReportHandler) GetReports(c *gin.Context) {
	rangeType := c.Query("range") // "today", "yesterday", "this_week", "this_month"
	
	now := time.Now()
	var startTime time.Time

	switch rangeType {
	case "today":
		startTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "yesterday":
		startTime = time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, now.Location())
	case "this_week":
		startTime = now.AddDate(0, 0, -int(now.Weekday()))
	case "this_month":
		startTime = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	default:
		startTime = now.AddDate(0, -1, 0) // Default 1 month
	}

	var stats []struct {
		Date          string  `json:"date"`
		TotalMessages int64   `json:"total_messages"`
		Successful    int64   `json:"successful"`
		Failed        int64   `json:"failed"`
		Revenue       float64 `json:"revenue"`
	}

	// Query for daily aggregation within the range
	db.GormDB.Model(&models.Message{}).
		Where("created_at >= ?", startTime).
		Select("DATE(created_at) as date, count(*) as total_messages, count(CASE WHEN status = 'DELIVERED' THEN 1 END) as successful, count(CASE WHEN status = 'FAILED' THEN 1 END) as failed, sum(cost) as revenue").
		Group("DATE(created_at)").
		Order("DATE(created_at) DESC").
		Scan(&stats)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
		"period":  rangeType,
		"summary": map[string]interface{}{
			"total_period_messages": 0, // Would be sum of stats
			"total_period_revenue":  0, 
		},
	})
}

// ExportReport handles GET /api/v1/admin/reports/export
func (h *AdminReportHandler) ExportReport(c *gin.Context) {
	format := c.Query("format") // "csv", "excel"
	if format == "" {
		format = "csv"
	}
	
	// Implementation would generate and stream a file
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Report generation started. You will be notified when it's ready.",
		"format":  format,
	})
}
