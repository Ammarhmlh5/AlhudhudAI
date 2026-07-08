package api

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/al-hudhud/backend/internal/admin/alerts"
	"github.com/al-hudhud/backend/internal/admin/distribution"
	"github.com/al-hudhud/backend/internal/admin/health"
	"github.com/al-hudhud/backend/internal/admin/metrics"
	"github.com/al-hudhud/backend/internal/admin/rating"
	"github.com/al-hudhud/backend/internal/admin/routing"
	"github.com/gin-gonic/gin"
)

// AdminDashboardHandler handles admin dashboard API endpoints
type AdminDashboardHandler struct {
	ratingEngine     *rating.Calculator
	metricsRepo      *metrics.Repository
	routingEngine    *routing.RoutingEngine
	poolManager      *distribution.DevicePoolManager
	healthChecker    *health.HealthChecker
	alertManager     *alerts.Manager
}

// NewAdminDashboardHandler creates a new admin dashboard handler
func NewAdminDashboardHandler(
	ratingEngine *rating.Calculator,
	metricsRepo *metrics.Repository,
	routingEngine *routing.RoutingEngine,
	poolManager *distribution.DevicePoolManager,
	healthChecker *health.HealthChecker,
	alertManager *alerts.Manager,
) *AdminDashboardHandler {
	return &AdminDashboardHandler{
		ratingEngine:  ratingEngine,
		metricsRepo:   metricsRepo,
		routingEngine: routingEngine,
		poolManager:   poolManager,
		healthChecker: healthChecker,
		alertManager:  alertManager,
	}
}

// DashboardOverview represents the aggregated dashboard statistics
type DashboardOverview struct {
	SystemStats      SystemStats              `json:"system_stats"`
	CompanyStats     CompanyStats             `json:"company_stats"`
	MessageStats     MessageStats             `json:"message_stats"`
	ServerStats      ServerStats              `json:"server_stats"`
	DeviceStats      DeviceStats              `json:"device_stats"`
	AlertStats       AlertStats               `json:"alert_stats"`
	TopCompanies     []TopCompany             `json:"top_companies"`
	RecentAlerts     []alerts.Alert           `json:"recent_alerts"`
	Timestamp        time.Time                `json:"timestamp"`
}

type SystemStats struct {
	TotalCompanies      int     `json:"total_companies"`
	ActiveCompanies     int     `json:"active_companies"`
	TotalServers        int     `json:"total_servers"`
	HealthyServers      int     `json:"healthy_servers"`
	TotalDevices        int     `json:"total_devices"`
	OnlineDevices       int     `json:"online_devices"`
	SystemHealthScore   float64 `json:"system_health_score"`
}

type CompanyStats struct {
	ExcellentRating     int     `json:"excellent_rating"`
	GoodRating          int     `json:"good_rating"`
	AverageRatingCount  int     `json:"average_rating_count"`
	PoorRating          int     `json:"poor_rating"`
	AverageRatingScore  float64 `json:"average_rating_score"`
}

type MessageStats struct {
	TotalToday          int64   `json:"total_today"`
	SuccessfulToday     int64   `json:"successful_today"`
	FailedToday         int64   `json:"failed_today"`
	SuccessRateToday    float64 `json:"success_rate_today"`
	TotalThisWeek       int64   `json:"total_this_week"`
	TotalThisMonth      int64   `json:"total_this_month"`
	AverageDeliveryTime int     `json:"average_delivery_time_ms"`
}

type ServerStats struct {
	TotalServers     int     `json:"total_servers"`
	HealthyServers   int     `json:"healthy_servers"`
	DegradedServers  int     `json:"degraded_servers"`
	UnhealthyServers int     `json:"unhealthy_servers"`
	AverageLatency   int     `json:"average_latency_ms"`
	HealthPercentage float64 `json:"health_percentage"`
}

type DeviceStats struct {
	TotalDevices    int     `json:"total_devices"`
	OnlineDevices   int     `json:"online_devices"`
	OfflineDevices  int     `json:"offline_devices"`
	FailedDevices   int     `json:"failed_devices"`
	ExcludedDevices int     `json:"excluded_devices"`
	OnlinePercentage float64 `json:"online_percentage"`
}

type AlertStats struct {
	TotalActive    int `json:"total_active"`
	Critical       int `json:"critical"`
	Warning        int `json:"warning"`
	Info           int `json:"info"`
	Acknowledged   int `json:"acknowledged"`
	Unacknowledged int `json:"unacknowledged"`
}



type TopCompany struct {
	TenantID      uuid.UUID `json:"tenant_id"`
	CompanyName   string    `json:"company_name"`
	Rating        float64   `json:"rating"`
	Category      string    `json:"category"`
	MessagesToday int64     `json:"messages_today"`
	SuccessRate   float64   `json:"success_rate"`
}

// GetDashboardOverview handles GET /api/v1/admin/dashboard/overview
func (h *AdminDashboardHandler) GetDashboardOverview(c *gin.Context) {
	ctx := c.Request.Context()

	// Initialize overview
	overview := DashboardOverview{
		Timestamp: time.Now(),
	}

	// Aggregate system stats (run in parallel)
	systemStatsChan := make(chan SystemStats, 1)
	companyStatsChan := make(chan CompanyStats, 1)
	messageStatsChan := make(chan MessageStats, 1)
	serverStatsChan := make(chan ServerStats, 1)
	deviceStatsChan := make(chan DeviceStats, 1)
	alertStatsChan := make(chan AlertStats, 1)
	topCompaniesChan := make(chan []TopCompany, 1)
	recentAlertsChan := make(chan []alerts.Alert, 1)

	go func() { systemStatsChan <- h.collectSystemStats(ctx) }()
	go func() { companyStatsChan <- h.collectCompanyStats(ctx) }()
	go func() { messageStatsChan <- h.collectMessageStats(ctx) }()
	go func() { serverStatsChan <- h.collectServerStats(ctx) }()
	go func() { deviceStatsChan <- h.collectDeviceStats(ctx) }()
	go func() { alertStatsChan <- h.collectAlertStats(ctx) }()
	go func() { topCompaniesChan <- h.collectTopCompanies(ctx, 10) }()
	go func() { recentAlertsChan <- h.collectRecentAlerts(ctx, 10) }()

	// Wait for all goroutines to complete
	overview.SystemStats = <-systemStatsChan
	overview.CompanyStats = <-companyStatsChan
	overview.MessageStats = <-messageStatsChan
	overview.ServerStats = <-serverStatsChan
	overview.DeviceStats = <-deviceStatsChan
	overview.AlertStats = <-alertStatsChan
	overview.TopCompanies = <-topCompaniesChan
	overview.RecentAlerts = <-recentAlertsChan

	RespondWithSuccess(c, http.StatusOK, overview, "")
}

// Helper methods to collect stats from different systems

func (h *AdminDashboardHandler) collectSystemStats(ctx context.Context) SystemStats {
	// For now, return placeholder data
	return SystemStats{
		TotalCompanies:    100,
		ActiveCompanies:   85,
		TotalServers:      20,
		HealthyServers:    18,
		TotalDevices:      150,
		OnlineDevices:     120,
		SystemHealthScore: 92.5,
	}
}

func (h *AdminDashboardHandler) collectCompanyStats(ctx context.Context) CompanyStats {
	return CompanyStats{
		ExcellentRating:    30,
		GoodRating:         40,
		AverageRatingCount: 20,
		PoorRating:         10,
		AverageRatingScore: 78.5,
	}
}

func (h *AdminDashboardHandler) collectMessageStats(ctx context.Context) MessageStats {
	return MessageStats{
		TotalToday:          50000,
		SuccessfulToday:     47500,
		FailedToday:         2500,
		SuccessRateToday:    95.0,
		TotalThisWeek:       300000,
		TotalThisMonth:      1200000,
		AverageDeliveryTime: 1500,
	}
}

func (h *AdminDashboardHandler) collectServerStats(ctx context.Context) ServerStats {
	return ServerStats{
		TotalServers:     20,
		HealthyServers:   18,
		DegradedServers:  1,
		UnhealthyServers: 1,
		AverageLatency:   150,
		HealthPercentage: 90.0,
	}
}

func (h *AdminDashboardHandler) collectDeviceStats(ctx context.Context) DeviceStats {
	return DeviceStats{
		TotalDevices:     150,
		OnlineDevices:    120,
		OfflineDevices:   25,
		FailedDevices:    3,
		ExcludedDevices:  2,
		OnlinePercentage: 80.0,
	}
}

func (h *AdminDashboardHandler) collectAlertStats(ctx context.Context) AlertStats {
	return AlertStats{
		TotalActive:    15,
		Critical:       3,
		Warning:        8,
		Info:           4,
		Acknowledged:   5,
		Unacknowledged: 10,
	}
}



func (h *AdminDashboardHandler) collectTopCompanies(ctx context.Context, limit int) []TopCompany {
	return []TopCompany{
		{
			TenantID:      uuid.New(),
			CompanyName:   "Company A",
			Rating:        95.5,
			Category:      "excellent",
			MessagesToday: 5000,
			SuccessRate:   98.5,
		},
	}
}

func (h *AdminDashboardHandler) collectRecentAlerts(ctx context.Context, limit int) []alerts.Alert {
	statusVal := alerts.AlertStatus("active")
	filter := alerts.AlertFilter{
		Status: &statusVal,
		Limit:  limit,
		Offset: 0,
	}
	
	alertsList, err := h.alertManager.GetAlerts(ctx, &filter)
	if err != nil {
		return []alerts.Alert{}
	}
	
	if alertsList == nil || alertsList.Alerts == nil {
		return []alerts.Alert{}
	}
	
	result := make([]alerts.Alert, len(alertsList.Alerts))
	for i, alertPtr := range alertsList.Alerts {
		result[i] = *alertPtr
	}
	
	return result
}
