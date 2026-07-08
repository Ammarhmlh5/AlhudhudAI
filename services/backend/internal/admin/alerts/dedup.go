package alerts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// Deduplicator handles alert deduplication
// Implements Requirement 6.8
type Deduplicator struct {
	db     *sqlx.DB
	logger *zap.Logger
	window time.Duration // Deduplication window (default: 5 minutes)
}

// NewDeduplicator creates a new alert deduplicator
func NewDeduplicator(db *sqlx.DB, logger *zap.Logger) *Deduplicator {
	return &Deduplicator{
		db:     db,
		logger: logger,
		window: 5 * time.Minute, // Default 5 minutes
	}
}

// SetWindow sets the deduplication window
func (d *Deduplicator) SetWindow(window time.Duration) {
	d.window = window
}

// CheckDuplicate checks if a duplicate alert exists within the deduplication window
// Returns true if a duplicate exists, false otherwise
// Implements Requirement 6.8
func (d *Deduplicator) CheckDuplicate(ctx context.Context, dedupKey string) (bool, error) {
	if dedupKey == "" {
		// No dedup key provided, allow the alert
		return false, nil
	}

	// Check for existing alert with same dedup_key within the window
	query := `
		SELECT COUNT(*)
		FROM alerts
		WHERE dedup_key = $1
		  AND created_at >= $2
		  AND status != $3
	`

	windowStart := time.Now().Add(-d.window)

	var count int
	err := d.db.QueryRowContext(
		ctx,
		query,
		dedupKey,
		windowStart,
		AlertStatusResolved, // Don't count resolved alerts
	).Scan(&count)

	if err != nil {
		d.logger.Error("Failed to check duplicate alert",
			zap.String("dedup_key", dedupKey),
			zap.Error(err),
		)
		return false, fmt.Errorf("failed to check duplicate: %w", err)
	}

	isDuplicate := count > 0

	if isDuplicate {
		d.logger.Info("Duplicate alert detected",
			zap.String("dedup_key", dedupKey),
			zap.Int("count", count),
			zap.Duration("window", d.window),
		)
	}

	return isDuplicate, nil
}

// GenerateDedupKey generates a deduplication key from alert components
// This ensures consistent key generation for similar alerts
func (d *Deduplicator) GenerateDedupKey(alertType AlertType, tenantID string, contextKey string) string {
	// Create a composite key from alert type, tenant, and context
	composite := fmt.Sprintf("%s:%s:%s", alertType, tenantID, contextKey)
	
	// Hash the composite key for consistent length
	hash := sha256.Sum256([]byte(composite))
	return hex.EncodeToString(hash[:])
}

// GenerateRatingChangeDedupKey generates a dedup key for rating change alerts
func (d *Deduplicator) GenerateRatingChangeDedupKey(tenantID string, oldCategory string, newCategory string) string {
	contextKey := fmt.Sprintf("rating:%s->%s", oldCategory, newCategory)
	return d.GenerateDedupKey(AlertTypeRatingChange, tenantID, contextKey)
}

// GenerateServerFailureDedupKey generates a dedup key for server failure alerts
func (d *Deduplicator) GenerateServerFailureDedupKey(tenantID string, serverID string) string {
	contextKey := fmt.Sprintf("server:%s", serverID)
	return d.GenerateDedupKey(AlertTypeServerFailure, tenantID, contextKey)
}

// GenerateThresholdViolationDedupKey generates a dedup key for threshold violation alerts
func (d *Deduplicator) GenerateThresholdViolationDedupKey(tenantID string, metricName string, thresholdType string) string {
	contextKey := fmt.Sprintf("threshold:%s:%s", metricName, thresholdType)
	return d.GenerateDedupKey(AlertTypeThresholdViolation, tenantID, contextKey)
}

// GenerateDeviceFailureDedupKey generates a dedup key for device failure alerts
func (d *Deduplicator) GenerateDeviceFailureDedupKey(tenantID string, deviceID string) string {
	contextKey := fmt.Sprintf("device:%s", deviceID)
	return d.GenerateDedupKey(AlertTypeDeviceFailure, tenantID, contextKey)
}

// CleanupOldAlerts removes resolved alerts older than the retention period
// This helps maintain database performance
func (d *Deduplicator) CleanupOldAlerts(ctx context.Context, retentionPeriod time.Duration) (int64, error) {
	query := `
		DELETE FROM alerts
		WHERE status = $1
		  AND resolved_at < $2
	`

	cutoffTime := time.Now().Add(-retentionPeriod)

	result, err := d.db.ExecContext(
		ctx,
		query,
		AlertStatusResolved,
		cutoffTime,
	)

	if err != nil {
		d.logger.Error("Failed to cleanup old alerts",
			zap.Error(err),
			zap.Duration("retention_period", retentionPeriod),
		)
		return 0, fmt.Errorf("failed to cleanup old alerts: %w", err)
	}

	rowsDeleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsDeleted > 0 {
		d.logger.Info("Cleaned up old alerts",
			zap.Int64("rows_deleted", rowsDeleted),
			zap.Duration("retention_period", retentionPeriod),
		)
	}

	return rowsDeleted, nil
}

// GetDuplicateCount returns the number of duplicate alerts for a given dedup key
// within the deduplication window
func (d *Deduplicator) GetDuplicateCount(ctx context.Context, dedupKey string) (int, error) {
	if dedupKey == "" {
		return 0, nil
	}

	query := `
		SELECT COUNT(*)
		FROM alerts
		WHERE dedup_key = $1
		  AND created_at >= $2
	`

	windowStart := time.Now().Add(-d.window)

	var count int
	err := d.db.QueryRowContext(ctx, query, dedupKey, windowStart).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get duplicate count: %w", err)
	}

	return count, nil
}
