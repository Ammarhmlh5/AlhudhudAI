package monitoring

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/google/uuid"
)

// AlertSeverity represents the severity level of an alert
type AlertSeverity string

const (
	SeverityWarning  AlertSeverity = "WARNING"  // 90% threshold
	SeverityCritical AlertSeverity = "CRITICAL" // 100% threshold
)

// AlertManager manages threshold checking and alert sending
type AlertManager struct {
	db               *sql.DB
	logger           *Logger
	notificationSvc  NotificationService
	defaultThreshold *AlertThreshold
}

// NotificationService interface for sending alerts
type NotificationService interface {
	SendAlert(ctx context.Context, alert *Alert) error
}

// NewAlertManager creates a new alert manager
func NewAlertManager(db *sql.DB, logger *Logger, notificationSvc NotificationService) *AlertManager {
	return &AlertManager{
		db:              db,
		logger:          logger,
		notificationSvc: notificationSvc,
		defaultThreshold: &AlertThreshold{
			CPUWarning:      80.0,  // 80% warning
			CPUCritical:     95.0,  // 95% critical
			MemoryWarning:   80.0,  // 80% warning
			MemoryCritical:  95.0,  // 95% critical
			BandwidthLimit:  1000.0, // 1000 KB/s
			ConnectionLimit: 90,    // 90% of max connections
		},
	}
}

// CheckThresholds checks resource usage against thresholds and sends alerts
// Implements Requirements 9.10, 17.8, 24.3
func (am *AlertManager) CheckThresholds(ctx context.Context) error {
	// Get all active dedicated gateways
	gateways, err := am.getActiveDedicatedGateways(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active gateways: %w", err)
	}

	for _, gateway := range gateways {
		if err := am.checkGatewayThresholds(ctx, gateway.ID, gateway.TenantID); err != nil {
			// Log error but continue with other gateways
			fmt.Printf("Error checking thresholds for gateway %d: %v\n", gateway.ID, err)
			continue
		}
	}

	return nil
}

// checkGatewayThresholds checks thresholds for a specific gateway
func (am *AlertManager) checkGatewayThresholds(ctx context.Context, gatewayID int, tenantID uuid.UUID) error {
	// Get recent usage records
	records, err := am.logger.GetRecentUsage(ctx, gatewayID, tenantID, 1)
	if err != nil {
		return fmt.Errorf("failed to get recent usage: %w", err)
	}

	if len(records) == 0 {
		return nil // No data to check
	}

	record := records[0]

	// Get gateway-specific thresholds or use defaults
	threshold, err := am.getGatewayThreshold(ctx, gatewayID, tenantID)
	if err != nil {
		// Use default thresholds if custom ones not found
		threshold = am.defaultThreshold
	}

	// Check CPU usage
	if err := am.checkCPUThreshold(ctx, gatewayID, tenantID, record.CPUUsagePercent, threshold); err != nil {
		fmt.Printf("Error checking CPU threshold: %v\n", err)
	}

	// Check Memory usage
	if err := am.checkMemoryThreshold(ctx, gatewayID, tenantID, record.MemoryUsageMB, threshold); err != nil {
		fmt.Printf("Error checking memory threshold: %v\n", err)
	}

	// Check Bandwidth usage
	if err := am.checkBandwidthThreshold(ctx, gatewayID, tenantID, record.BandwidthUsageMBps*1024, threshold); err != nil {
		fmt.Printf("Error checking bandwidth threshold: %v\n", err)
	}

	// Check Database connections
	if err := am.checkDatabaseThreshold(ctx, gatewayID, tenantID, record.DBConnectionsActive, threshold); err != nil {
		fmt.Printf("Error checking database threshold: %v\n", err)
	}

	// Check Redis memory
	if err := am.checkRedisThreshold(ctx, gatewayID, tenantID, record.RedisMemoryUsageMB, threshold); err != nil {
		fmt.Printf("Error checking Redis threshold: %v\n", err)
	}

	return nil
}

// checkCPUThreshold checks CPU usage against thresholds
func (am *AlertManager) checkCPUThreshold(ctx context.Context, gatewayID int, tenantID uuid.UUID, cpuUsage float64, threshold *AlertThreshold) error {
	// Check critical threshold (100% or above)
	if cpuUsage >= threshold.CPUCritical {
		alert := &Alert{
			ID:          uuid.New(),
			GatewayID:   gatewayID,
			TenantID:    tenantID,
			AlertType:   "CPU",
			Severity:    string(SeverityCritical),
			Message:     fmt.Sprintf("CPU usage critical: %.2f%% (threshold: %.2f%%)", cpuUsage, threshold.CPUCritical),
			MetricValue: cpuUsage,
			Threshold:   threshold.CPUCritical,
			Timestamp:   time.Now(),
			Resolved:    false,
		}
		return am.SendAlert(ctx, alert)
	}

	// Check warning threshold (90% or above)
	if cpuUsage >= threshold.CPUWarning {
		alert := &Alert{
			ID:          uuid.New(),
			GatewayID:   gatewayID,
			TenantID:    tenantID,
			AlertType:   "CPU",
			Severity:    string(SeverityWarning),
			Message:     fmt.Sprintf("CPU usage warning: %.2f%% (threshold: %.2f%%)", cpuUsage, threshold.CPUWarning),
			MetricValue: cpuUsage,
			Threshold:   threshold.CPUWarning,
			Timestamp:   time.Now(),
			Resolved:    false,
		}
		return am.SendAlert(ctx, alert)
	}

	return nil
}

// checkMemoryThreshold checks memory usage against thresholds
func (am *AlertManager) checkMemoryThreshold(ctx context.Context, gatewayID int, tenantID uuid.UUID, memoryUsageMB int, threshold *AlertThreshold) error {
	// Get total memory from resource_quotas
	totalMemoryMB, err := am.getGatewayMemoryQuota(ctx, gatewayID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get memory quota: %w", err)
	}

	if totalMemoryMB == 0 {
		return nil // No quota set
	}

	memoryPercent := float64(memoryUsageMB) / float64(totalMemoryMB) * 100

	// Check critical threshold
	if memoryPercent >= threshold.MemoryCritical {
		alert := &Alert{
			ID:          uuid.New(),
			GatewayID:   gatewayID,
			TenantID:    tenantID,
			AlertType:   "MEMORY",
			Severity:    string(SeverityCritical),
			Message:     fmt.Sprintf("Memory usage critical: %d MB / %d MB (%.2f%%, threshold: %.2f%%)", memoryUsageMB, totalMemoryMB, memoryPercent, threshold.MemoryCritical),
			MetricValue: memoryPercent,
			Threshold:   threshold.MemoryCritical,
			Timestamp:   time.Now(),
			Resolved:    false,
		}
		return am.SendAlert(ctx, alert)
	}

	// Check warning threshold
	if memoryPercent >= threshold.MemoryWarning {
		alert := &Alert{
			ID:          uuid.New(),
			GatewayID:   gatewayID,
			TenantID:    tenantID,
			AlertType:   "MEMORY",
			Severity:    string(SeverityWarning),
			Message:     fmt.Sprintf("Memory usage warning: %d MB / %d MB (%.2f%%, threshold: %.2f%%)", memoryUsageMB, totalMemoryMB, memoryPercent, threshold.MemoryWarning),
			MetricValue: memoryPercent,
			Threshold:   threshold.MemoryWarning,
			Timestamp:   time.Now(),
			Resolved:    false,
		}
		return am.SendAlert(ctx, alert)
	}

	return nil
}

// checkBandwidthThreshold checks bandwidth usage against threshold
func (am *AlertManager) checkBandwidthThreshold(ctx context.Context, gatewayID int, tenantID uuid.UUID, bandwidthKBps float64, threshold *AlertThreshold) error {
	// Get bandwidth quota
	bandwidthQuota, err := am.getGatewayBandwidthQuota(ctx, gatewayID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get bandwidth quota: %w", err)
	}

	if bandwidthQuota == 0 {
		return nil // No quota set
	}

	bandwidthPercent := bandwidthKBps / float64(bandwidthQuota) * 100

	// Check critical threshold (100% or above)
	if bandwidthPercent >= 100.0 {
		alert := &Alert{
			ID:          uuid.New(),
			GatewayID:   gatewayID,
			TenantID:    tenantID,
			AlertType:   "BANDWIDTH",
			Severity:    string(SeverityCritical),
			Message:     fmt.Sprintf("Bandwidth usage critical: %.2f KB/s / %d KB/s (%.2f%%)", bandwidthKBps, bandwidthQuota, bandwidthPercent),
			MetricValue: bandwidthPercent,
			Threshold:   100.0,
			Timestamp:   time.Now(),
			Resolved:    false,
		}
		return am.SendAlert(ctx, alert)
	}

	// Check warning threshold (90% or above)
	if bandwidthPercent >= 90.0 {
		alert := &Alert{
			ID:          uuid.New(),
			GatewayID:   gatewayID,
			TenantID:    tenantID,
			AlertType:   "BANDWIDTH",
			Severity:    string(SeverityWarning),
			Message:     fmt.Sprintf("Bandwidth usage warning: %.2f KB/s / %d KB/s (%.2f%%)", bandwidthKBps, bandwidthQuota, bandwidthPercent),
			MetricValue: bandwidthPercent,
			Threshold:   90.0,
			Timestamp:   time.Now(),
			Resolved:    false,
		}
		return am.SendAlert(ctx, alert)
	}

	return nil
}

// checkDatabaseThreshold checks database connections against threshold
func (am *AlertManager) checkDatabaseThreshold(ctx context.Context, gatewayID int, tenantID uuid.UUID, activeConnections int, threshold *AlertThreshold) error {
	// Get database connection quota
	maxConnections, err := am.getGatewayDBConnectionQuota(ctx, gatewayID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get DB connection quota: %w", err)
	}

	if maxConnections == 0 {
		return nil // No quota set
	}

	connectionPercent := float64(activeConnections) / float64(maxConnections) * 100

	// Check critical threshold (100% or above)
	if connectionPercent >= 100.0 {
		alert := &Alert{
			ID:          uuid.New(),
			GatewayID:   gatewayID,
			TenantID:    tenantID,
			AlertType:   "DATABASE",
			Severity:    string(SeverityCritical),
			Message:     fmt.Sprintf("Database connections critical: %d / %d (%.2f%%)", activeConnections, maxConnections, connectionPercent),
			MetricValue: connectionPercent,
			Threshold:   100.0,
			Timestamp:   time.Now(),
			Resolved:    false,
		}
		return am.SendAlert(ctx, alert)
	}

	// Check warning threshold (90% or above)
	if connectionPercent >= 90.0 {
		alert := &Alert{
			ID:          uuid.New(),
			GatewayID:   gatewayID,
			TenantID:    tenantID,
			AlertType:   "DATABASE",
			Severity:    string(SeverityWarning),
			Message:     fmt.Sprintf("Database connections warning: %d / %d (%.2f%%)", activeConnections, maxConnections, connectionPercent),
			MetricValue: connectionPercent,
			Threshold:   90.0,
			Timestamp:   time.Now(),
			Resolved:    false,
		}
		return am.SendAlert(ctx, alert)
	}

	return nil
}

// checkRedisThreshold checks Redis memory usage against threshold
func (am *AlertManager) checkRedisThreshold(ctx context.Context, gatewayID int, tenantID uuid.UUID, redisMemoryMB int, threshold *AlertThreshold) error {
	// Get Redis memory quota
	redisQuota, err := am.getGatewayRedisQuota(ctx, gatewayID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get Redis quota: %w", err)
	}

	if redisQuota == 0 {
		return nil // No quota set
	}

	redisPercent := float64(redisMemoryMB) / float64(redisQuota) * 100

	// Check critical threshold (100% or above)
	if redisPercent >= 100.0 {
		alert := &Alert{
			ID:          uuid.New(),
			GatewayID:   gatewayID,
			TenantID:    tenantID,
			AlertType:   "REDIS",
			Severity:    string(SeverityCritical),
			Message:     fmt.Sprintf("Redis memory critical: %d MB / %d MB (%.2f%%)", redisMemoryMB, redisQuota, redisPercent),
			MetricValue: redisPercent,
			Threshold:   100.0,
			Timestamp:   time.Now(),
			Resolved:    false,
		}
		return am.SendAlert(ctx, alert)
	}

	// Check warning threshold (90% or above)
	if redisPercent >= 90.0 {
		alert := &Alert{
			ID:          uuid.New(),
			GatewayID:   gatewayID,
			TenantID:    tenantID,
			AlertType:   "REDIS",
			Severity:    string(SeverityWarning),
			Message:     fmt.Sprintf("Redis memory warning: %d MB / %d MB (%.2f%%)", redisMemoryMB, redisQuota, redisPercent),
			MetricValue: redisPercent,
			Threshold:   90.0,
			Timestamp:   time.Now(),
			Resolved:    false,
		}
		return am.SendAlert(ctx, alert)
	}

	return nil
}

// SendAlert sends an alert through the notification service
func (am *AlertManager) SendAlert(ctx context.Context, alert *Alert) error {
	// Log the alert to database first
	if err := am.logAlert(ctx, alert); err != nil {
		return fmt.Errorf("failed to log alert: %w", err)
	}

	// Send notification
	if am.notificationSvc != nil {
		if err := am.notificationSvc.SendAlert(ctx, alert); err != nil {
			return fmt.Errorf("failed to send alert notification: %w", err)
		}
	}

	return nil
}

// logAlert logs an alert to the database
func (am *AlertManager) logAlert(ctx context.Context, alert *Alert) error {
	query := `
		INSERT INTO monitoring_alerts (
			id,
			gateway_id,
			tenant_id,
			alert_type,
			severity,
			message,
			metric_value,
			threshold,
			timestamp,
			resolved,
			resolved_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	err := db.RunInTenantTx(ctx, alert.TenantID, false, am.db, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx,
			query,
			alert.ID,
			alert.GatewayID,
			alert.TenantID,
			alert.AlertType,
			alert.Severity,
			alert.Message,
			alert.MetricValue,
			alert.Threshold,
			alert.Timestamp,
			alert.Resolved,
			alert.ResolvedAt,
		)
		return err
	})

	return err
}

// getActiveDedicatedGateways retrieves all active dedicated gateways
func (am *AlertManager) getActiveDedicatedGateways(ctx context.Context) ([]struct {
	ID       int
	TenantID uuid.UUID
}, error) {
	query := `
		SELECT id, tenant_id
		FROM dedicated_gateways
		WHERE status = 'active'
		  AND deleted_at IS NULL
	`

	rows, err := am.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gateways []struct {
		ID       int
		TenantID uuid.UUID
	}

	for rows.Next() {
		var gateway struct {
			ID       int
			TenantID uuid.UUID
		}
		if err := rows.Scan(&gateway.ID, &gateway.TenantID); err != nil {
			return nil, err
		}
		gateways = append(gateways, gateway)
	}

	return gateways, rows.Err()
}

// getGatewayThreshold retrieves custom thresholds for a gateway
func (am *AlertManager) getGatewayThreshold(ctx context.Context, gatewayID int, tenantID uuid.UUID) (*AlertThreshold, error) {
	// For now, return default thresholds
	// In the future, this could query a gateway_alert_thresholds table
	return am.defaultThreshold, nil
}

// getGatewayMemoryQuota retrieves memory quota for a gateway
func (am *AlertManager) getGatewayMemoryQuota(ctx context.Context, gatewayID int, tenantID uuid.UUID) (int, error) {
	query := `
		SELECT memory_quota_mb
		FROM resource_quotas
		WHERE gateway_id = $1
	`

	var memoryQuota int
	err := db.RunInTenantTx(ctx, tenantID, false, am.db, func(tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, query, gatewayID).Scan(&memoryQuota)
	})

	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return memoryQuota, nil
}

// getGatewayBandwidthQuota retrieves bandwidth quota for a gateway
func (am *AlertManager) getGatewayBandwidthQuota(ctx context.Context, gatewayID int, tenantID uuid.UUID) (int, error) {
	query := `
		SELECT bandwidth_quota_mbps
		FROM resource_quotas
		WHERE gateway_id = $1
	`

	var bandwidthQuota int
	err := db.RunInTenantTx(ctx, tenantID, false, am.db, func(tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, query, gatewayID).Scan(&bandwidthQuota)
	})

	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	// Convert Mbps to KBps (1 Mbps = 128 KB/s)
	return bandwidthQuota * 128, nil
}

// getGatewayDBConnectionQuota retrieves database connection quota for a gateway
func (am *AlertManager) getGatewayDBConnectionQuota(ctx context.Context, gatewayID int, tenantID uuid.UUID) (int, error) {
	query := `
		SELECT max_db_connections
		FROM resource_quotas
		WHERE gateway_id = $1
	`

	var maxConnections int
	err := db.RunInTenantTx(ctx, tenantID, false, am.db, func(tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, query, gatewayID).Scan(&maxConnections)
	})

	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return maxConnections, nil
}

// getGatewayRedisQuota retrieves Redis memory quota for a gateway
func (am *AlertManager) getGatewayRedisQuota(ctx context.Context, gatewayID int, tenantID uuid.UUID) (int, error) {
	query := `
		SELECT redis_memory_quota_mb
		FROM resource_quotas
		WHERE gateway_id = $1
	`

	var redisQuota int
	err := db.RunInTenantTx(ctx, tenantID, false, am.db, func(tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, query, gatewayID).Scan(&redisQuota)
	})

	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return redisQuota, nil
}

// StartPeriodicChecking starts checking thresholds periodically
func (am *AlertManager) StartPeriodicChecking(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Check immediately on start
	if err := am.CheckThresholds(ctx); err != nil {
		fmt.Printf("Error checking thresholds: %v\n", err)
	}

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Stopping periodic threshold checking")
			return
		case <-ticker.C:
			if err := am.CheckThresholds(ctx); err != nil {
				fmt.Printf("Error checking thresholds: %v\n", err)
			}
		}
	}
}
