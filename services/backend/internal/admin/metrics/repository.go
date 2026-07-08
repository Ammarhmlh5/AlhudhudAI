package metrics

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository handles database operations for performance metrics
type Repository struct {
	db *sqlx.DB
	
	// Prepared statements for performance
	saveStmt      *sqlx.NamedStmt
	getStmt       *sqlx.Stmt
	aggregateStmt *sqlx.Stmt
}

// NewRepository creates a new metrics repository with prepared statements
func NewRepository(db *sqlx.DB) (*Repository, error) {
	repo := &Repository{db: db}
	
	// Prepare save statement
	saveQuery := `
		INSERT INTO performance_metrics (
			tenant_id, period_start, period_end, period_type,
			total_messages, successful_messages, failed_messages,
			average_delivery_time_ms, p95_delivery_time_ms, p99_delivery_time_ms,
			error_count, total_payments, on_time_payments
		) VALUES (
			:tenant_id, :period_start, :period_end, :period_type,
			:total_messages, :successful_messages, :failed_messages,
			:average_delivery_time_ms, :p95_delivery_time_ms, :p99_delivery_time_ms,
			:error_count, :total_payments, :on_time_payments
		)
		ON CONFLICT (tenant_id, period_start, period_type)
		DO UPDATE SET
			period_end = EXCLUDED.period_end,
			total_messages = EXCLUDED.total_messages,
			successful_messages = EXCLUDED.successful_messages,
			failed_messages = EXCLUDED.failed_messages,
			average_delivery_time_ms = EXCLUDED.average_delivery_time_ms,
			p95_delivery_time_ms = EXCLUDED.p95_delivery_time_ms,
			p99_delivery_time_ms = EXCLUDED.p99_delivery_time_ms,
			error_count = EXCLUDED.error_count,
			total_payments = EXCLUDED.total_payments,
			on_time_payments = EXCLUDED.on_time_payments
		RETURNING id, success_rate, error_rate, payment_compliance, created_at
	`
	
	saveStmt, err := db.PrepareNamed(saveQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare save statement: %w", err)
	}
	repo.saveStmt = saveStmt
	
	return repo, nil
}

// Close closes all prepared statements
func (r *Repository) Close() error {
	if r.saveStmt != nil {
		if err := r.saveStmt.Close(); err != nil {
			return fmt.Errorf("failed to close save statement: %w", err)
		}
	}
	return nil
}

// SaveMetrics saves or updates performance metrics for a tenant
// Requirements: 4.1, 4.2, 4.6
func (r *Repository) SaveMetrics(ctx context.Context, metrics *PerformanceMetrics) error {
	// Validate input
	if metrics.TenantID == uuid.Nil {
		return fmt.Errorf("tenant_id is required")
	}
	if metrics.PeriodStart.IsZero() || metrics.PeriodEnd.IsZero() {
		return fmt.Errorf("period_start and period_end are required")
	}
	if metrics.PeriodEnd.Before(metrics.PeriodStart) || metrics.PeriodEnd.Equal(metrics.PeriodStart) {
		return fmt.Errorf("period_end must be after period_start")
	}
	if metrics.PeriodType != "hourly" && metrics.PeriodType != "daily" && metrics.PeriodType != "monthly" {
		return fmt.Errorf("period_type must be one of: hourly, daily, monthly")
	}
	
	// Execute prepared statement
	err := r.saveStmt.GetContext(ctx, metrics, metrics)
	if err != nil {
		return fmt.Errorf("failed to save metrics: %w", err)
	}
	
	return nil
}

// GetMetrics retrieves performance metrics for a tenant with filtering
// Requirements: 4.1, 4.2, 4.6
func (r *Repository) GetMetrics(ctx context.Context, filter MetricsFilter) ([]PerformanceMetrics, error) {
	// Validate input
	if filter.TenantID == uuid.Nil {
		return nil, fmt.Errorf("tenant_id is required")
	}
	
	// Build query with filters
	query := `
		SELECT 
			id, tenant_id, period_start, period_end, period_type,
			total_messages, successful_messages, failed_messages, success_rate,
			average_delivery_time_ms, p95_delivery_time_ms, p99_delivery_time_ms,
			error_count, error_rate,
			total_payments, on_time_payments, payment_compliance,
			created_at
		FROM performance_metrics
		WHERE tenant_id = $1
	`
	
	args := []interface{}{filter.TenantID}
	argIndex := 2
	
	// Add period_type filter
	if filter.PeriodType != nil && *filter.PeriodType != "" {
		query += fmt.Sprintf(" AND period_type = $%d", argIndex)
		args = append(args, *filter.PeriodType)
		argIndex++
	}
	
	// Add time range filter
	if filter.StartTime != nil {
		query += fmt.Sprintf(" AND period_start >= $%d", argIndex)
		args = append(args, *filter.StartTime)
		argIndex++
	}
	
	if filter.EndTime != nil {
		query += fmt.Sprintf(" AND period_end <= $%d", argIndex)
		args = append(args, *filter.EndTime)
		argIndex++
	}
	
	// Add ordering
	query += " ORDER BY period_start DESC"
	
	// Add limit and offset
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++
	}
	
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filter.Offset)
	}
	
	// Execute query
	var metrics []PerformanceMetrics
	err := r.db.SelectContext(ctx, &metrics, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}
	
	return metrics, nil
}

// AggregateMetrics aggregates metrics data for a specific period
// Requirements: 4.2
func (r *Repository) AggregateMetrics(ctx context.Context, tenantID uuid.UUID, periodType string, periodStart, periodEnd time.Time) (*PerformanceMetrics, error) {
	// Validate input
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("tenant_id is required")
	}
	if periodType != "hourly" && periodType != "daily" && periodType != "monthly" {
		return nil, fmt.Errorf("period_type must be one of: hourly, daily, monthly")
	}
	if periodEnd.Before(periodStart) || periodEnd.Equal(periodStart) {
		return nil, fmt.Errorf("period_end must be after period_start")
	}
	
	// Query to aggregate metrics from source data
	// This aggregates from the messages and payments tables
	query := `
		WITH message_stats AS (
			SELECT
				COUNT(*) as total_messages,
				COUNT(*) FILTER (WHERE status = 'delivered') as successful_messages,
				COUNT(*) FILTER (WHERE status = 'failed') as failed_messages,
				AVG(EXTRACT(EPOCH FROM (delivered_at - created_at)) * 1000)::INTEGER as avg_delivery_ms,
				PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY EXTRACT(EPOCH FROM (delivered_at - created_at)) * 1000)::INTEGER as p95_delivery_ms,
				PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY EXTRACT(EPOCH FROM (delivered_at - created_at)) * 1000)::INTEGER as p99_delivery_ms,
				COUNT(*) FILTER (WHERE error_message IS NOT NULL) as error_count
			FROM messages
			WHERE tenant_id = $1
				AND created_at >= $2
				AND created_at < $3
		),
		payment_stats AS (
			SELECT
				COUNT(*) as total_payments,
				COUNT(*) FILTER (WHERE paid_at <= due_date) as on_time_payments
			FROM billing_invoices
			WHERE tenant_id = $1
				AND created_at >= $2
				AND created_at < $3
		)
		SELECT
			COALESCE(m.total_messages, 0) as total_messages,
			COALESCE(m.successful_messages, 0) as successful_messages,
			COALESCE(m.failed_messages, 0) as failed_messages,
			m.avg_delivery_ms as average_delivery_time_ms,
			m.p95_delivery_ms as p95_delivery_time_ms,
			m.p99_delivery_ms as p99_delivery_time_ms,
			COALESCE(m.error_count, 0) as error_count,
			COALESCE(p.total_payments, 0) as total_payments,
			COALESCE(p.on_time_payments, 0) as on_time_payments
		FROM message_stats m
		CROSS JOIN payment_stats p
	`
	
	var result struct {
		TotalMessages         int64        `db:"total_messages"`
		SuccessfulMessages    int64        `db:"successful_messages"`
		FailedMessages        int64        `db:"failed_messages"`
		AverageDeliveryTimeMs sql.NullInt32 `db:"average_delivery_time_ms"`
		P95DeliveryTimeMs     sql.NullInt32 `db:"p95_delivery_time_ms"`
		P99DeliveryTimeMs     sql.NullInt32 `db:"p99_delivery_time_ms"`
		ErrorCount            int          `db:"error_count"`
		TotalPayments         int          `db:"total_payments"`
		OnTimePayments        int          `db:"on_time_payments"`
	}
	
	err := r.db.GetContext(ctx, &result, query, tenantID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate metrics: %w", err)
	}
	
	// Build metrics object
	metrics := &PerformanceMetrics{
		TenantID:           tenantID,
		PeriodStart:        periodStart,
		PeriodEnd:          periodEnd,
		PeriodType:         periodType,
		TotalMessages:      result.TotalMessages,
		SuccessfulMessages: result.SuccessfulMessages,
		FailedMessages:     result.FailedMessages,
		ErrorCount:         result.ErrorCount,
		TotalPayments:      result.TotalPayments,
		OnTimePayments:     result.OnTimePayments,
	}
	
	// Handle nullable fields
	if result.AverageDeliveryTimeMs.Valid {
		avgMs := int(result.AverageDeliveryTimeMs.Int32)
		metrics.AverageDeliveryTimeMs = &avgMs
	}
	if result.P95DeliveryTimeMs.Valid {
		p95Ms := int(result.P95DeliveryTimeMs.Int32)
		metrics.P95DeliveryTimeMs = &p95Ms
	}
	if result.P99DeliveryTimeMs.Valid {
		p99Ms := int(result.P99DeliveryTimeMs.Int32)
		metrics.P99DeliveryTimeMs = &p99Ms
	}
	
	return metrics, nil
}

// GetLatestMetrics retrieves the most recent metrics for a tenant
func (r *Repository) GetLatestMetrics(ctx context.Context, tenantID uuid.UUID, periodType string) (*PerformanceMetrics, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("tenant_id is required")
	}
	
	query := `
		SELECT 
			id, tenant_id, period_start, period_end, period_type,
			total_messages, successful_messages, failed_messages, success_rate,
			average_delivery_time_ms, p95_delivery_time_ms, p99_delivery_time_ms,
			error_count, error_rate,
			total_payments, on_time_payments, payment_compliance,
			created_at
		FROM performance_metrics
		WHERE tenant_id = $1 AND period_type = $2
		ORDER BY period_start DESC
		LIMIT 1
	`
	
	var metrics PerformanceMetrics
	err := r.db.GetContext(ctx, &metrics, query, tenantID, periodType)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest metrics: %w", err)
	}
	
	return &metrics, nil
}

// DeleteOldMetrics deletes metrics older than the specified retention period
func (r *Repository) DeleteOldMetrics(ctx context.Context, retentionMonths int) (int64, error) {
	if retentionMonths <= 0 {
		return 0, fmt.Errorf("retention_months must be positive")
	}
	
	cutoffDate := time.Now().AddDate(0, -retentionMonths, 0)
	
	query := `
		DELETE FROM performance_metrics
		WHERE period_start < $1
	`
	
	result, err := r.db.ExecContext(ctx, query, cutoffDate)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old metrics: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	return rowsAffected, nil
}
