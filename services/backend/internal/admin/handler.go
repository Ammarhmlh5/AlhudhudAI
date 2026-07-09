package admin

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/al-hudhud/backend/internal/admin/alerts"
	"github.com/al-hudhud/backend/internal/admin/distribution"
	"github.com/al-hudhud/backend/internal/admin/health"
	"github.com/al-hudhud/backend/internal/admin/metrics"
	"github.com/al-hudhud/backend/internal/admin/rating"
	"github.com/al-hudhud/backend/internal/admin/routing"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/internal/queue"
	"github.com/al-hudhud/backend/internal/redis"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Handler struct {
	Redis           *redis.Client
	DB              *sqlx.DB
	GormDB          *gorm.DB
	Logger          *zap.Logger
	RatingEngine    *rating.Calculator
	MetricsRepo     *metrics.Repository
	RoutingEngine   *routing.RoutingEngine
	RoutingConfig   *routing.RoutingConfigManager
	PoolManager     *distribution.DevicePoolManager
	HealthChecker   *health.HealthChecker
	AlertManager    *alerts.Manager
}

// NewHandler creates a new admin handler
func NewHandler(
	redisClient *redis.Client,
	dbConn *sqlx.DB,
	gormDB *gorm.DB,
	logger *zap.Logger,
	ratingEngine *rating.Calculator,
	metricsRepo *metrics.Repository,
	routingEngine *routing.RoutingEngine,
	routingConfig *routing.RoutingConfigManager,
	poolManager *distribution.DevicePoolManager,
	healthChecker *health.HealthChecker,
	alertManager *alerts.Manager,
) *Handler {
	return &Handler{
		Redis:          redisClient,
		DB:             dbConn,
		GormDB:         gormDB,
		Logger:         logger,
		RatingEngine:   ratingEngine,
		MetricsRepo:    metricsRepo,
		RoutingEngine:  routingEngine,
		RoutingConfig:  routingConfig,
		PoolManager:    poolManager,
		HealthChecker:  healthChecker,
		AlertManager:   alertManager,
	}
}

// GetStats returns pre-aggregated admin statistics from admin_statistics table.
// This is O(1) - no expensive COUNT(*) queries on large tables.
func (h *Handler) GetStats(c *gin.Context) {
	var stats models.AdminStatistics

	if err := db.GormDB.First(&stats, 1).Error; err != nil {
		// If the row doesn't exist yet, return zeros gracefully
		c.JSON(http.StatusOK, gin.H{
			"total_tenants":         0,
			"active_tenants":        0,
			"total_devices":         0,
			"active_devices":        0,
			"pending_messages":      0,
			"queued_messages":       0,
			"processing_messages":   0,
			"total_messages_today":  0,
			"sent_messages_today":   0,
			"failed_messages_today": 0,
			"total_balance":         0,
			"total_events_today":    0,
			"parsed_events_today":   0,
			"failed_events_today":   0,
			"last_updated":          nil,
			"note":                  "statistics initializing, try again shortly",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetSystemHealth returns real-time infrastructure health metrics
func (h *Handler) GetSystemHealth(c *gin.Context) {
	dbStats := db.PgxPool.Stat()
	
	health := gin.H{
		"db": gin.H{
			"total_conns": dbStats.TotalConns(),
			"idle_conns":  dbStats.IdleConns(),
			"busy_conns":  dbStats.AcquiredConns(),
			"max_conns":   dbStats.MaxConns(),
		},
		"redis": gin.H{
			"status": "connected", // Simplified for now
		},
		"uptime": time.Since(time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC)).String(), // Mock uptime
		"service": "al-hudhud-backend",
		"version": "3.2.4",
	}

	c.JSON(http.StatusOK, health)
}

// ListEvents returns a paginated list of global incoming events for auditing
func (h *Handler) ListEvents(c *gin.Context) {
	var events []models.GlobalIncomingEvent
	status := c.Query("status")
	tenantID := c.Query("tenant_id")

	query := db.GormDB.Order("created_at DESC").Limit(100)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}

	if err := query.Find(&events).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list events"})
		return
	}

	c.JSON(http.StatusOK, events)
}

// GetEvent returns full details of a single event including raw payload
func (h *Handler) GetEvent(c *gin.Context) {
	id := c.Param("id")
	var event models.GlobalIncomingEvent
	if err := db.GormDB.First(&event, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}
	c.JSON(http.StatusOK, event)
}

// RetryEvent manually re-queues a failed event for parsing
func (h *Handler) RetryEvent(c *gin.Context) {
	id := c.Param("id")
	var event models.GlobalIncomingEvent
	if err := db.GormDB.First(&event, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	// Reset status to RECEIVED
	if err := db.GormDB.Model(&event).Update("status", "RECEIVED").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset status"})
		return
	}

	// Re-insert into River queue using tenant-specific client
	if event.TenantID == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Event has no tenant ID"})
		return
	}

	tenantClient, err := queue.InitTenantClient(c.Request.Context(), *event.TenantID, h.Redis)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to init tenant queue: " + err.Error()})
		return
	}

	_, err = tenantClient.Insert(c.Request.Context(), queue.GlobalEventArgs{
		GlobalEventID: int64(event.ID),
		TenantID:      *event.TenantID,
	}, nil)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retry event: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Event re-queued successfully"})
}

// ListAlerts returns real-time system alerts (suspicious activity, etc)
func (h *Handler) ListAlerts(c *gin.Context) {
	var alerts []models.SystemAlert
	if err := db.GormDB.Order("created_at DESC").Limit(50).Find(&alerts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list alerts"})
		return
	}
	c.JSON(http.StatusOK, alerts)
}

// GetDashboardOverview returns a comprehensive overview of system status
// Implements Requirement 8.1: Real-time system status overview
func (h *Handler) GetDashboardOverview(c *gin.Context) {
	ctx := c.Request.Context()
	
	// Use a WaitGroup to collect data in parallel
	var wg sync.WaitGroup
	var mu sync.Mutex
	
	results := gin.H{
		"timestamp": time.Now(),
		"status":    "healthy",
	}
	
	// 1. Get stats from models (existing logic)
	wg.Add(1)
	go func() {
		defer wg.Done()
		var stats models.AdminStatistics
		if err := h.GormDB.First(&stats, 1).Error; err == nil {
			mu.Lock()
			results["statistics"] = stats
			mu.Unlock()
		}
	}()
	
	// 2. Get active alerts count
	wg.Add(1)
	go func() {
		defer wg.Done()
		status := alerts.AlertStatusActive
		filter := alerts.AlertFilter{Status: &status}
		alertsList, err := h.AlertManager.GetAlerts(ctx, &filter)
		if err == nil {
			mu.Lock()
			results["active_alerts_count"] = alertsList.TotalCount
			mu.Unlock()
		}
	}()
	
	// 3. Get system health summary
	wg.Add(1)
	go func() {
		defer wg.Done()
		dbStats := h.DB.Stats()
		mu.Lock()
		results["health"] = gin.H{
			"db": gin.H{
				"open_connections": dbStats.OpenConnections,
				"in_use":           dbStats.InUse,
				"idle":             dbStats.Idle,
			},
			"redis": "connected",
		}
		mu.Unlock()
	}()
	
	wg.Wait()
	c.JSON(http.StatusOK, results)
}

// GetAlerts returns a filtered list of system alerts
func (h *Handler) GetAlerts(c *gin.Context) {
	ctx := c.Request.Context()
	
	filter := alerts.AlertFilter{
		Limit:  50,
		Offset: 0,
	}
	
	if tenantIDStr := c.Query("tenant_id"); tenantIDStr != "" {
		if tid, err := uuid.Parse(tenantIDStr); err == nil {
			filter.TenantID = &tid
		}
	}
	
	if severity := c.Query("severity"); severity != "" {
		s := alerts.AlertSeverity(severity)
		filter.Severity = &s
	}
	
	if status := c.Query("status"); status != "" {
		s := alerts.AlertStatus(status)
		filter.Status = &s
	}
	
	alertsList, err := h.AlertManager.GetAlerts(ctx, &filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve alerts"})
		return
	}
	
	c.JSON(http.StatusOK, alertsList)
}

// AcknowledgeAlert marks an alert as acknowledged
func (h *Handler) AcknowledgeAlert(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid alert ID"})
		return
	}
	
	// In a real scenario, we'd get the user ID from the auth context
	userID := uuid.Nil 
	
	if err := h.AlertManager.AcknowledgeAlert(c.Request.Context(), id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Alert acknowledged"})
}

// ListDevices returns all devices in the pool for a tenant
func (h *Handler) ListDevices(c *gin.Context) {
	tenantIDStr := c.Query("tenant_id")
	if tenantIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}
	
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant_id"})
		return
	}
	
	devices, err := h.PoolManager.GetAllDevices(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch devices"})
		return
	}
	
	c.JSON(http.StatusOK, devices)
}

// GetRoutingStats returns routing performance for a tenant
func (h *Handler) GetRoutingStats(c *gin.Context) {
	tenantIDStr := c.Param("tenant_id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant_id"})
		return
	}
	
	stats, err := h.RoutingEngine.GetRoutingStats(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch routing stats"})
		return
	}
	
	c.JSON(http.StatusOK, stats)
}

// UpdateRoutingConfig updates dedicated server settings for a tenant
func (h *Handler) UpdateRoutingConfig(c *gin.Context) {
	var input routing.UpdateRoutingConfigInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	updated, err := h.RoutingConfig.UpdateRoutingConfig(c.Request.Context(), &input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, updated)
}


