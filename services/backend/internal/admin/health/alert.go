package health

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// AlertTrigger handles triggering alerts when server health status changes
type AlertTrigger struct {
	db              *sqlx.DB
	mu              sync.RWMutex
	lastStatusCache map[uuid.UUID]HealthStatus
}

// NewAlertTrigger creates a new alert trigger instance
func NewAlertTrigger(db *sqlx.DB) *AlertTrigger {
	return &AlertTrigger{
		db:              db,
		lastStatusCache: make(map[uuid.UUID]HealthStatus),
	}
}

// OnStatusChange checks if the server status has changed and triggers an alert if needed
func (at *AlertTrigger) OnStatusChange(ctx context.Context, result *HealthCheckResult) error {
	// Get the last known status for this server
	lastStatus, err := at.getLastStatus(ctx, result.ServerID)
	if err != nil {
		return fmt.Errorf("failed to get last status: %w", err)
	}
	
	// Check if status has changed
	if lastStatus != result.HealthStatus {
		// Status has changed, trigger an alert
		if err := at.triggerAlert(ctx, result, lastStatus); err != nil {
			return fmt.Errorf("failed to trigger alert: %w", err)
		}
		
		// Update the cache
		at.updateStatusCache(result.ServerID, result.HealthStatus)
	}
	
	return nil
}

// getLastStatus retrieves the last known health status for a server
func (at *AlertTrigger) getLastStatus(ctx context.Context, serverID uuid.UUID) (HealthStatus, error) {
	// Check cache first
	at.mu.RLock()
	if status, exists := at.lastStatusCache[serverID]; exists {
		at.mu.RUnlock()
		return status, nil
	}
	at.mu.RUnlock()
	
	// Query database for last status
	query := `
		SELECT health_status
		FROM server_health_logs
		WHERE server_id = $1
		ORDER BY check_timestamp DESC
		LIMIT 1
	`
	
	var status string
	err := at.db.QueryRowContext(ctx, query, serverID).Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			// No previous status, return empty (will be treated as new server)
			return "", nil
		}
		return "", fmt.Errorf("failed to query last status: %w", err)
	}
	
	healthStatus := HealthStatus(status)
	
	// Update cache
	at.updateStatusCache(serverID, healthStatus)
	
	return healthStatus, nil
}

// updateStatusCache updates the in-memory status cache
func (at *AlertTrigger) updateStatusCache(serverID uuid.UUID, status HealthStatus) {
	at.mu.Lock()
	defer at.mu.Unlock()
	at.lastStatusCache[serverID] = status
}

// triggerAlert creates an alert for the status change
func (at *AlertTrigger) triggerAlert(ctx context.Context, result *HealthCheckResult, previousStatus HealthStatus) error {
	// Determine alert severity based on new status
	severity := at.determineSeverity(result.HealthStatus)
	
	// Create alert title and message
	title := fmt.Sprintf("Server Health Status Changed: %s", result.ServerID)
	message := at.buildAlertMessage(result, previousStatus)
	
	// Create dedup key to prevent duplicate alerts within 5 minutes
	dedupKey := fmt.Sprintf("server_health_%s_%s", result.ServerID, result.HealthStatus)
	
	// Check if a similar alert was created recently (within 5 minutes)
	if at.isDuplicateAlert(ctx, dedupKey, 5*time.Minute) {
		// Skip creating duplicate alert
		return nil
	}
	
	// Insert alert into database
	query := `
		INSERT INTO alerts (
			tenant_id,
			alert_type,
			severity,
			title,
			message,
			status,
			context,
			dedup_key,
			created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
	`
	
	contextJSON := fmt.Sprintf(`{
		"server_id": "%s",
		"previous_status": "%s",
		"new_status": "%s",
		"response_time_ms": %d,
		"error_rate": %.2f,
		"is_available": %t
	}`, result.ServerID, previousStatus, result.HealthStatus, 
		result.ResponseTimeMs, result.ErrorRate, result.IsAvailable)
	
	_, err := at.db.ExecContext(
		ctx,
		query,
		result.TenantID,
		"server_health",
		severity,
		title,
		message,
		"active",
		contextJSON,
		dedupKey,
		time.Now(),
	)
	
	if err != nil {
		return fmt.Errorf("failed to insert alert: %w", err)
	}
	
	return nil
}

// determineSeverity determines the alert severity based on health status
func (at *AlertTrigger) determineSeverity(status HealthStatus) string {
	switch status {
	case Unhealthy:
		return "critical"
	case Degraded:
		return "warning"
	case Healthy:
		return "info"
	default:
		return "info"
	}
}

// buildAlertMessage builds a descriptive alert message
func (at *AlertTrigger) buildAlertMessage(result *HealthCheckResult, previousStatus HealthStatus) string {
	if previousStatus == "" {
		return fmt.Sprintf(
			"Server health status is now %s. Response time: %dms, Error rate: %.2f%%, Available: %t",
			result.HealthStatus,
			result.ResponseTimeMs,
			result.ErrorRate,
			result.IsAvailable,
		)
	}
	
	return fmt.Sprintf(
		"Server health status changed from %s to %s. Response time: %dms, Error rate: %.2f%%, Available: %t",
		previousStatus,
		result.HealthStatus,
		result.ResponseTimeMs,
		result.ErrorRate,
		result.IsAvailable,
	)
}

// isDuplicateAlert checks if a similar alert was created recently
func (at *AlertTrigger) isDuplicateAlert(ctx context.Context, dedupKey string, window time.Duration) bool {
	query := `
		SELECT COUNT(*)
		FROM alerts
		WHERE dedup_key = $1
		AND created_at > $2
		AND status = 'active'
	`
	
	var count int
	err := at.db.QueryRowContext(ctx, query, dedupKey, time.Now().Add(-window)).Scan(&count)
	if err != nil {
		// If error, assume not duplicate to be safe
		return false
	}
	
	return count > 0
}

// ClearStatusCache clears the in-memory status cache
func (at *AlertTrigger) ClearStatusCache() {
	at.mu.Lock()
	defer at.mu.Unlock()
	at.lastStatusCache = make(map[uuid.UUID]HealthStatus)
}

// GetCachedStatus returns the cached status for a server (for testing)
func (at *AlertTrigger) GetCachedStatus(serverID uuid.UUID) (HealthStatus, bool) {
	at.mu.RLock()
	defer at.mu.RUnlock()
	status, exists := at.lastStatusCache[serverID]
	return status, exists
}
