package metrics

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ThresholdMonitor monitors performance metrics against configured thresholds
type ThresholdMonitor struct {
	db     *sqlx.DB
	repo   *Repository
	logger *log.Logger
}

// NewThresholdMonitor creates a new threshold monitor
func NewThresholdMonitor(db *sqlx.DB, repo *Repository, logger *log.Logger) *ThresholdMonitor {
	if logger == nil {
		logger = log.Default()
	}
	
	return &ThresholdMonitor{
		db:     db,
		repo:   repo,
		logger: logger,
	}
}

// CheckThresholds checks performance metrics against configured thresholds
// Requirements: 4.8
func (tm *ThresholdMonitor) CheckThresholds(ctx context.Context, tenantID uuid.UUID) ([]ThresholdViolation, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("tenant_id is required")
	}
	
	// Get threshold configuration for tenant
	config, err := tm.GetThresholdConfig(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get threshold config: %w", err)
	}
	
	// If no thresholds configured, return empty
	if config == nil {
		return []ThresholdViolation{}, nil
	}
	
	// Get latest metrics
	metrics, err := tm.repo.GetLatestMetrics(ctx, tenantID, "hourly")
	if err != nil {
		return nil, fmt.Errorf("failed to get latest metrics: %w", err)
	}
	
	// If no metrics available, return empty
	if metrics == nil {
		return []ThresholdViolation{}, nil
	}
	
	// Check each threshold
	var violations []ThresholdViolation
	
	// Check success rate threshold
	if config.MinSuccessRate != nil && metrics.SuccessRate < *config.MinSuccessRate {
		violations = append(violations, ThresholdViolation{
			TenantID:       tenantID,
			MetricName:     "success_rate",
			CurrentValue:   metrics.SuccessRate,
			ThresholdValue: *config.MinSuccessRate,
			Severity:       tm.determineSeverity(metrics.SuccessRate, *config.MinSuccessRate, "min"),
			DetectedAt:     time.Now(),
		})
	}
	
	// Check error rate threshold
	if config.MaxErrorRate != nil && metrics.ErrorRate > *config.MaxErrorRate {
		violations = append(violations, ThresholdViolation{
			TenantID:       tenantID,
			MetricName:     "error_rate",
			CurrentValue:   metrics.ErrorRate,
			ThresholdValue: *config.MaxErrorRate,
			Severity:       tm.determineSeverity(metrics.ErrorRate, *config.MaxErrorRate, "max"),
			DetectedAt:     time.Now(),
		})
	}
	
	// Check payment compliance threshold
	if config.MinPaymentCompliance != nil && metrics.PaymentCompliance < *config.MinPaymentCompliance {
		violations = append(violations, ThresholdViolation{
			TenantID:       tenantID,
			MetricName:     "payment_compliance",
			CurrentValue:   metrics.PaymentCompliance,
			ThresholdValue: *config.MinPaymentCompliance,
			Severity:       tm.determineSeverity(metrics.PaymentCompliance, *config.MinPaymentCompliance, "min"),
			DetectedAt:     time.Now(),
		})
	}
	
	// Check delivery time threshold
	if config.MaxDeliveryTimeMs != nil && metrics.AverageDeliveryTimeMs != nil {
		if *metrics.AverageDeliveryTimeMs > *config.MaxDeliveryTimeMs {
			violations = append(violations, ThresholdViolation{
				TenantID:       tenantID,
				MetricName:     "average_delivery_time_ms",
				CurrentValue:   float64(*metrics.AverageDeliveryTimeMs),
				ThresholdValue: float64(*config.MaxDeliveryTimeMs),
				Severity:       tm.determineSeverity(float64(*metrics.AverageDeliveryTimeMs), float64(*config.MaxDeliveryTimeMs), "max"),
				DetectedAt:     time.Now(),
			})
		}
	}
	
	return violations, nil
}

// TriggerAlert triggers an alert for threshold violations
// Requirements: 4.8
func (tm *ThresholdMonitor) TriggerAlert(ctx context.Context, violation ThresholdViolation) error {
	// Create alert message
	message := tm.formatAlertMessage(violation)
	
	// Determine alert type
	alertType := "threshold_violation"
	
	// Create alert in database
	alert := Alert{
		TenantID:  &violation.TenantID,
		AlertType: alertType,
		Severity:  violation.Severity,
		Title:     fmt.Sprintf("Performance Threshold Violation: %s", violation.MetricName),
		Message:   message,
		Context: map[string]interface{}{
			"metric_name":     violation.MetricName,
			"current_value":   violation.CurrentValue,
			"threshold_value": violation.ThresholdValue,
			"detected_at":     violation.DetectedAt,
		},
		Status:    "active",
		DedupKey:  fmt.Sprintf("threshold_%s_%s", violation.TenantID, violation.MetricName),
		CreatedAt: time.Now(),
	}
	
	// Save alert to database
	err := tm.saveAlert(ctx, &alert)
	if err != nil {
		return fmt.Errorf("failed to save alert: %w", err)
	}
	
	tm.logger.Printf("Alert triggered for tenant %s: %s", violation.TenantID, message)
	
	return nil
}

// CheckThresholdsForAllTenants checks thresholds for all active tenants
func (tm *ThresholdMonitor) CheckThresholdsForAllTenants(ctx context.Context) error {
	// Get all active tenants
	tenantIDs, err := tm.getActiveTenantIDs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active tenants: %w", err)
	}
	
	tm.logger.Printf("Checking thresholds for %d tenants", len(tenantIDs))
	
	totalViolations := 0
	
	for _, tenantID := range tenantIDs {
		violations, err := tm.CheckThresholds(ctx, tenantID)
		if err != nil {
			tm.logger.Printf("Error checking thresholds for tenant %s: %v", tenantID, err)
			continue
		}
		
		// Trigger alerts for violations
		for _, violation := range violations {
			if err := tm.TriggerAlert(ctx, violation); err != nil {
				tm.logger.Printf("Error triggering alert for tenant %s: %v", tenantID, err)
			}
		}
		
		totalViolations += len(violations)
	}
	
	tm.logger.Printf("Threshold check completed: %d violations found", totalViolations)
	
	return nil
}

// GetThresholdConfig retrieves threshold configuration for a tenant
func (tm *ThresholdMonitor) GetThresholdConfig(ctx context.Context, tenantID uuid.UUID) (*ThresholdConfig, error) {
	query := `
		SELECT 
			tenant_id,
			min_success_rate,
			max_error_rate,
			min_payment_compliance,
			max_delivery_time_ms
		FROM threshold_configs
		WHERE tenant_id = $1
	`
	
	var config ThresholdConfig
	var minSuccessRate, maxErrorRate, minPaymentCompliance sql.NullFloat64
	var maxDeliveryTimeMs sql.NullInt32
	
	err := tm.db.QueryRowContext(ctx, query, tenantID).Scan(
		&config.TenantID,
		&minSuccessRate,
		&maxErrorRate,
		&minPaymentCompliance,
		&maxDeliveryTimeMs,
	)
	
	if err == sql.ErrNoRows {
		// Return default thresholds if none configured
		return tm.getDefaultThresholdConfig(tenantID), nil
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to query threshold config: %w", err)
	}
	
	// Convert nullable fields
	if minSuccessRate.Valid {
		val := minSuccessRate.Float64
		config.MinSuccessRate = &val
	}
	if maxErrorRate.Valid {
		val := maxErrorRate.Float64
		config.MaxErrorRate = &val
	}
	if minPaymentCompliance.Valid {
		val := minPaymentCompliance.Float64
		config.MinPaymentCompliance = &val
	}
	if maxDeliveryTimeMs.Valid {
		val := int(maxDeliveryTimeMs.Int32)
		config.MaxDeliveryTimeMs = &val
	}
	
	return &config, nil
}

// SaveThresholdConfig saves threshold configuration for a tenant
func (tm *ThresholdMonitor) SaveThresholdConfig(ctx context.Context, config *ThresholdConfig) error {
	if config.TenantID == uuid.Nil {
		return fmt.Errorf("tenant_id is required")
	}
	
	query := `
		INSERT INTO threshold_configs (
			tenant_id,
			min_success_rate,
			max_error_rate,
			min_payment_compliance,
			max_delivery_time_ms,
			updated_at
		) VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (tenant_id)
		DO UPDATE SET
			min_success_rate = EXCLUDED.min_success_rate,
			max_error_rate = EXCLUDED.max_error_rate,
			min_payment_compliance = EXCLUDED.min_payment_compliance,
			max_delivery_time_ms = EXCLUDED.max_delivery_time_ms,
			updated_at = NOW()
	`
	
	_, err := tm.db.ExecContext(ctx, query,
		config.TenantID,
		config.MinSuccessRate,
		config.MaxErrorRate,
		config.MinPaymentCompliance,
		config.MaxDeliveryTimeMs,
	)
	
	if err != nil {
		return fmt.Errorf("failed to save threshold config: %w", err)
	}
	
	return nil
}

// getDefaultThresholdConfig returns default threshold configuration
func (tm *ThresholdMonitor) getDefaultThresholdConfig(tenantID uuid.UUID) *ThresholdConfig {
	minSuccessRate := 95.0
	maxErrorRate := 5.0
	minPaymentCompliance := 90.0
	maxDeliveryTimeMs := 5000
	
	return &ThresholdConfig{
		TenantID:             tenantID,
		MinSuccessRate:       &minSuccessRate,
		MaxErrorRate:         &maxErrorRate,
		MinPaymentCompliance: &minPaymentCompliance,
		MaxDeliveryTimeMs:    &maxDeliveryTimeMs,
	}
}

// determineSeverity determines alert severity based on how far the value is from threshold
func (tm *ThresholdMonitor) determineSeverity(currentValue, thresholdValue float64, thresholdType string) string {
	var deviation float64
	
	if thresholdType == "min" {
		// For minimum thresholds (e.g., success rate should be >= threshold)
		deviation = (thresholdValue - currentValue) / thresholdValue * 100
	} else {
		// For maximum thresholds (e.g., error rate should be <= threshold)
		deviation = (currentValue - thresholdValue) / thresholdValue * 100
	}
	
	// Determine severity based on deviation
	if deviation >= 50 {
		return "critical"
	} else if deviation >= 20 {
		return "warning"
	}
	
	return "info"
}

// formatAlertMessage formats a human-readable alert message
func (tm *ThresholdMonitor) formatAlertMessage(violation ThresholdViolation) string {
	switch violation.MetricName {
	case "success_rate":
		return fmt.Sprintf("Success rate (%.2f%%) is below threshold (%.2f%%)",
			violation.CurrentValue, violation.ThresholdValue)
	case "error_rate":
		return fmt.Sprintf("Error rate (%.2f%%) exceeds threshold (%.2f%%)",
			violation.CurrentValue, violation.ThresholdValue)
	case "payment_compliance":
		return fmt.Sprintf("Payment compliance (%.2f%%) is below threshold (%.2f%%)",
			violation.CurrentValue, violation.ThresholdValue)
	case "average_delivery_time_ms":
		return fmt.Sprintf("Average delivery time (%.0fms) exceeds threshold (%.0fms)",
			violation.CurrentValue, violation.ThresholdValue)
	default:
		return fmt.Sprintf("Metric %s: current value %.2f, threshold %.2f",
			violation.MetricName, violation.CurrentValue, violation.ThresholdValue)
	}
}

// saveAlert saves an alert to the database
func (tm *ThresholdMonitor) saveAlert(ctx context.Context, alert *Alert) error {
	// Check for duplicate alert using dedup_key
	if alert.DedupKey != "" {
		exists, err := tm.checkDuplicateAlert(ctx, alert.DedupKey)
		if err != nil {
			return fmt.Errorf("failed to check duplicate alert: %w", err)
		}
		if exists {
			tm.logger.Printf("Duplicate alert suppressed: %s", alert.DedupKey)
			return nil
		}
	}
	
	query := `
		INSERT INTO alerts (
			tenant_id,
			alert_type,
			severity,
			title,
			message,
			context,
			status,
			dedup_key,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	
	_, err := tm.db.ExecContext(ctx, query,
		alert.TenantID,
		alert.AlertType,
		alert.Severity,
		alert.Title,
		alert.Message,
		alert.Context,
		alert.Status,
		alert.DedupKey,
		alert.CreatedAt,
	)
	
	if err != nil {
		return fmt.Errorf("failed to insert alert: %w", err)
	}
	
	return nil
}

// checkDuplicateAlert checks if a duplicate alert exists within the last 5 minutes
func (tm *ThresholdMonitor) checkDuplicateAlert(ctx context.Context, dedupKey string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM alerts
			WHERE dedup_key = $1
			  AND created_at > NOW() - INTERVAL '5 minutes'
			  AND status = 'active'
		)
	`
	
	var exists bool
	err := tm.db.QueryRowContext(ctx, query, dedupKey).Scan(&exists)
	if err != nil {
		return false, err
	}
	
	return exists, nil
}

// getActiveTenantIDs retrieves all active tenant IDs
func (tm *ThresholdMonitor) getActiveTenantIDs(ctx context.Context) ([]uuid.UUID, error) {
	query := `
		SELECT id FROM tenants
		WHERE is_active = true
		ORDER BY id
	`
	
	var tenantIDs []uuid.UUID
	err := tm.db.SelectContext(ctx, &tenantIDs, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active tenants: %w", err)
	}
	
	return tenantIDs, nil
}

// Alert represents an alert in the system
type Alert struct {
	ID             int64                  `json:"id" db:"id"`
	TenantID       *uuid.UUID             `json:"tenant_id,omitempty" db:"tenant_id"`
	AlertType      string                 `json:"alert_type" db:"alert_type"`
	Severity       string                 `json:"severity" db:"severity"`
	Title          string                 `json:"title" db:"title"`
	Message        string                 `json:"message" db:"message"`
	Context        map[string]interface{} `json:"context,omitempty" db:"context"`
	Status         string                 `json:"status" db:"status"`
	AcknowledgedAt *time.Time             `json:"acknowledged_at,omitempty" db:"acknowledged_at"`
	AcknowledgedBy *uuid.UUID             `json:"acknowledged_by,omitempty" db:"acknowledged_by"`
	ResolvedAt     *time.Time             `json:"resolved_at,omitempty" db:"resolved_at"`
	DedupKey       string                 `json:"dedup_key,omitempty" db:"dedup_key"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
}

// StartPeriodicChecking starts periodic threshold checking
func (tm *ThresholdMonitor) StartPeriodicChecking(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	tm.logger.Printf("Starting periodic threshold checking (interval: %v)", interval)
	
	// Check immediately on start
	if err := tm.CheckThresholdsForAllTenants(ctx); err != nil {
		tm.logger.Printf("Error checking thresholds: %v", err)
	}
	
	for {
		select {
		case <-ctx.Done():
			tm.logger.Println("Stopping periodic threshold checking")
			return
		case <-ticker.C:
			if err := tm.CheckThresholdsForAllTenants(ctx); err != nil {
				tm.logger.Printf("Error checking thresholds: %v", err)
			}
		}
	}
}
