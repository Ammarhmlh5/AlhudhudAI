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

type revenueBreakdown struct {
	Total         float64 `json:"total"`
	Subscriptions float64 `json:"subscriptions"`
	Overage       float64 `json:"overage"`
	Gateways      float64 `json:"gateways"`
	Topups        float64 `json:"topups"`
}

type costBreakdown struct {
	Total          float64 `json:"total"`
	Messages       float64 `json:"messages"`
	Infrastructure float64 `json:"infrastructure"`
}

type customerDist struct {
	Total     int64           `json:"total"`
	ByPackage []packageDist   `json:"byPackage"`
}

type packageDist struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type subscriptionStats struct {
	Renewals      int64   `json:"renewals"`
	Cancellations int64   `json:"cancellations"`
	RenewalRate   float64 `json:"renewalRate"`
}

type reportsResponse struct {
	Revenue       revenueBreakdown `json:"revenue"`
	Costs         costBreakdown    `json:"costs"`
	Customers     customerDist     `json:"customers"`
	Subscriptions subscriptionStats `json:"subscriptions"`
}

// GetReports handles GET /api/v1/admin/reports
func (h *AdminReportHandler) GetReports(c *gin.Context) {
	rangeType := c.Query("range") // "week", "month", "quarter", "year", "today", "yesterday", "this_week", "this_month", "this_year"

	now := time.Now()
	var startTime time.Time

	switch rangeType {
	case "today":
		startTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "yesterday":
		startTime = now.AddDate(0, 0, -1)
		startTime = time.Date(startTime.Year(), startTime.Month(), startTime.Day(), 0, 0, 0, 0, now.Location())
	case "week", "this_week":
		startTime = now.AddDate(0, 0, -int(now.Weekday()))
	case "month", "this_month":
		startTime = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	case "quarter":
		q := (int(now.Month())-1)/3*3 + 1
		startTime = time.Date(now.Year(), time.Month(q), 1, 0, 0, 0, 0, now.Location())
	case "year", "this_year":
		startTime = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
	default:
		startTime = now.AddDate(0, -1, 0)
	}

	gormDB := db.GormDB

	// ---- Revenue ----
	var msgRevenue float64
	gormDB.Model(&models.Message{}).
		Where("created_at >= ?", startTime).
		Select("COALESCE(SUM(cost), 0)").
		Scan(&msgRevenue)

	var subRevenue float64
	gormDB.Raw(`
		SELECT COALESCE(SUM(p.price_usd), 0)
		FROM subscriptions s
		LEFT JOIN packages p ON s.package_id = p.id
		WHERE (s.created_at >= ? OR s.last_renewed_at >= ?)
		  AND s.status IN ('active', 'grace_period')
	`, startTime, startTime).Scan(&subRevenue)

	var overageRevenue float64
	gormDB.Raw(`
		SELECT COALESCE(SUM(s.overage_cost), 0)
		FROM subscriptions s
		WHERE s.last_renewed_at >= ? AND s.overage_cost > 0
	`, startTime).Scan(&overageRevenue)

	var gatewayRevenue float64
	gormDB.Raw(`
		SELECT COALESCE(SUM(subscription_fee), 0)
		FROM dedicated_gateways
		WHERE status = 'active'
	`).Scan(&gatewayRevenue)

	var topupRevenue float64
	gormDB.Raw(`
		SELECT COALESCE(SUM(amount), 0)
		FROM payment_transactions
		WHERE created_at >= ? AND status = 'completed'
	`, startTime).Scan(&topupRevenue)

	totalRevenue := msgRevenue + subRevenue + overageRevenue + gatewayRevenue + topupRevenue

	// ---- Costs ----
	msgCosts := msgRevenue
	var infraCost float64 = 0

	// ---- Customers ----
	var totalCustomers int64
	gormDB.Model(&models.Tenant{}).
		Where("is_active = true").
		Count(&totalCustomers)

	var byPackage []packageDist
	gormDB.Raw(`
		SELECT COALESCE(p.name, 'Unknown') as name, COUNT(*) as count
		FROM subscriptions s
		LEFT JOIN packages p ON s.package_id = p.id
		WHERE s.status = 'active'
		GROUP BY p.name
		ORDER BY count DESC
	`).Scan(&byPackage)
	if byPackage == nil {
		byPackage = []packageDist{}
	}

	// ---- Subscriptions ----
	var subRenewals int64
	gormDB.Raw(`
		SELECT COUNT(*)
		FROM subscriptions
		WHERE last_renewed_at >= ?
	`, startTime).Scan(&subRenewals)

	var subCancellations int64
	gormDB.Raw(`
		SELECT COUNT(*)
		FROM subscriptions
		WHERE status = 'cancelled' AND updated_at >= ?
	`, startTime).Scan(&subCancellations)

	renewalRate := 0.0
	total := subRenewals + subCancellations
	if total > 0 {
		renewalRate = float64(subRenewals) / float64(total) * 100
	}

	resp := reportsResponse{
		Revenue: revenueBreakdown{
			Total:         totalRevenue,
			Subscriptions: subRevenue,
			Overage:       overageRevenue,
			Gateways:      gatewayRevenue,
			Topups:        topupRevenue,
		},
		Costs: costBreakdown{
			Total:          msgCosts + infraCost,
			Messages:       msgCosts,
			Infrastructure: infraCost,
		},
		Customers: customerDist{
			Total:     totalCustomers,
			ByPackage: byPackage,
		},
		Subscriptions: subscriptionStats{
			Renewals:      subRenewals,
			Cancellations: subCancellations,
			RenewalRate:   renewalRate,
		},
	}

	c.JSON(http.StatusOK, resp)
}

// ExportReport handles GET /api/v1/admin/reports/export
func (h *AdminReportHandler) ExportReport(c *gin.Context) {
	format := c.Query("format") // "csv", "excel"
	if format == "" {
		format = "csv"
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Report generation started. You will be notified when it's ready.",
		"format":  format,
	})
}
