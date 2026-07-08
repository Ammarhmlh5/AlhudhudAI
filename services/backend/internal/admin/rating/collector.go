package rating

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Collector collects performance metrics from multiple sources
type Collector struct {
	db *sqlx.DB
}

// NewCollector creates a new Collector
func NewCollector(db *sqlx.DB) *Collector {
	return &Collector{
		db: db,
	}
}

// CollectMetrics collects all performance metrics for a company
func (c *Collector) CollectMetrics(ctx context.Context, tenantID uuid.UUID, periodType string, periodStart, periodEnd time.Time) (*PerformanceMetrics, error) {
	metrics := &PerformanceMetrics{
		TenantID:    tenantID,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		PeriodType:  periodType,
	}

	// Collect message statistics from gateway_statistics
	if err := c.collectMessageStatistics(ctx, tenantID, periodStart, periodEnd, metrics); err != nil {
		return nil, fmt.Errorf("failed to collect message statistics: %w", err)
	}

	// Collect payment data from tariff system
	if err := c.collectPaymentData(ctx, tenantID, periodStart, periodEnd, metrics); err != nil {
		return nil, fmt.Errorf("failed to collect payment data: %w", err)
	}

	// Collect resource usage data
	if err := c.collectResourceUsage(ctx, tenantID, periodStart, periodEnd); err != nil {
		return nil, fmt.Errorf("failed to collect resource usage: %w", err)
	}

	// Calculate derived metrics
	c.calculateDerivedMetrics(metrics)

	return metrics, nil
}

// collectMessageStatistics collects message statistics from gateway_statistics table
func (c *Collector) collectMessageStatistics(ctx context.Context, tenantID uuid.UUID, periodStart, periodEnd time.Time, metrics *PerformanceMetrics) error {
	query := `
		SELECT 
			COALESCE(SUM(messages_sent), 0) as total_messages,
			COALESCE(SUM(messages_delivered), 0) as successful_messages,
			COALESCE(SUM(messages_failed), 0) as failed_messages,
			COALESCE(AVG(avg_response_time_ms), 0)::INTEGER as avg_delivery_time_ms,
			COALESCE(MAX(max_response_time_ms), 0) as p99_delivery_time_ms,
			COUNT(*) as error_count
		FROM gateway_statistics
		WHERE tenant_id = $1
		  AND date >= $2::DATE
		  AND date <= $3::DATE
	`

	var avgDeliveryTime, p99DeliveryTime int
	err := c.db.QueryRowContext(ctx, query, tenantID, periodStart, periodEnd).Scan(
		&metrics.TotalMessages,
		&metrics.SuccessfulMessages,
		&metrics.FailedMessages,
		&avgDeliveryTime,
		&p99DeliveryTime,
		&metrics.ErrorCount,
	)

	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to query gateway_statistics: %w", err)
	}

	// Set delivery time metrics if available
	if avgDeliveryTime > 0 {
		metrics.AverageDeliveryTimeMs = &avgDeliveryTime
	}
	if p99DeliveryTime > 0 {
		metrics.P99DeliveryTimeMs = &p99DeliveryTime
	}

	// Estimate P95 as 90% of P99 if not available
	if metrics.P99DeliveryTimeMs != nil {
		p95 := int(float64(*metrics.P99DeliveryTimeMs) * 0.9)
		metrics.P95DeliveryTimeMs = &p95
	}

	return nil
}

// collectPaymentData collects payment compliance data from tariff system
func (c *Collector) collectPaymentData(ctx context.Context, tenantID uuid.UUID, periodStart, periodEnd time.Time, metrics *PerformanceMetrics) error {
	// Query payment transactions from billing_invoices and payment_transactions
	query := `
		SELECT 
			COUNT(*) as total_payments,
			COUNT(*) FILTER (WHERE bi.status = 'paid' AND bi.paid_at <= bi.due_date) as on_time_payments
		FROM billing_invoices bi
		WHERE bi.tenant_id = $1
		  AND bi.created_at >= $2
		  AND bi.created_at <= $3
	`

	err := c.db.QueryRowContext(ctx, query, tenantID, periodStart, periodEnd).Scan(
		&metrics.TotalPayments,
		&metrics.OnTimePayments,
	)

	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to query payment data: %w", err)
	}

	return nil
}

// collectResourceUsage collects resource usage data
func (c *Collector) collectResourceUsage(ctx context.Context, tenantID uuid.UUID, periodStart, periodEnd time.Time) error {
	// Query resource usage for additional quality metrics
	query := `
		SELECT 
			COALESCE(AVG(cpu_usage_percent), 0) as avg_cpu,
			COALESCE(AVG(memory_usage_mb), 0) as avg_memory,
			COALESCE(AVG(current_queue_depth), 0) as avg_queue_depth
		FROM resource_usage
		WHERE tenant_id = $1
		  AND timestamp >= $2
		  AND timestamp <= $3
	`

	var avgCPU, avgMemory, avgQueueDepth float64
	err := c.db.QueryRowContext(ctx, query, tenantID, periodStart, periodEnd).Scan(
		&avgCPU,
		&avgMemory,
		&avgQueueDepth,
	)

	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to query resource usage: %w", err)
	}

	// Resource usage data can be used for quality score calculation
	// Store in context or use for additional metrics if needed

	return nil
}

// calculateDerivedMetrics calculates derived metrics from collected data
func (c *Collector) calculateDerivedMetrics(metrics *PerformanceMetrics) {
	// Calculate success rate
	if metrics.TotalMessages > 0 {
		metrics.SuccessRate = (float64(metrics.SuccessfulMessages) / float64(metrics.TotalMessages)) * 100
	} else {
		metrics.SuccessRate = 0
	}

	// Calculate error rate
	if metrics.TotalMessages > 0 {
		metrics.ErrorRate = (float64(metrics.ErrorCount) / float64(metrics.TotalMessages)) * 100
	} else {
		metrics.ErrorRate = 0
	}

	// Calculate payment compliance
	if metrics.TotalPayments > 0 {
		metrics.PaymentCompliance = (float64(metrics.OnTimePayments) / float64(metrics.TotalPayments)) * 100
	} else {
		metrics.PaymentCompliance = 0
	}
}

// CollectCurrentMetrics collects metrics for the current period (last 24 hours)
func (c *Collector) CollectCurrentMetrics(ctx context.Context, tenantID uuid.UUID) (*PerformanceMetrics, error) {
	now := time.Now()
	periodStart := now.Add(-24 * time.Hour)
	return c.CollectMetrics(ctx, tenantID, "daily", periodStart, now)
}

// CollectHourlyMetrics collects metrics for a specific hour
func (c *Collector) CollectHourlyMetrics(ctx context.Context, tenantID uuid.UUID, hour time.Time) (*PerformanceMetrics, error) {
	periodStart := hour.Truncate(time.Hour)
	periodEnd := periodStart.Add(time.Hour)
	return c.CollectMetrics(ctx, tenantID, "hourly", periodStart, periodEnd)
}

// CollectDailyMetrics collects metrics for a specific day
func (c *Collector) CollectDailyMetrics(ctx context.Context, tenantID uuid.UUID, day time.Time) (*PerformanceMetrics, error) {
	periodStart := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	periodEnd := periodStart.Add(24 * time.Hour)
	return c.CollectMetrics(ctx, tenantID, "daily", periodStart, periodEnd)
}

// CollectMonthlyMetrics collects metrics for a specific month
func (c *Collector) CollectMonthlyMetrics(ctx context.Context, tenantID uuid.UUID, month time.Time) (*PerformanceMetrics, error) {
	periodStart := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, month.Location())
	periodEnd := periodStart.AddDate(0, 1, 0)
	return c.CollectMetrics(ctx, tenantID, "monthly", periodStart, periodEnd)
}

// SaveMetrics saves performance metrics to the database
func (c *Collector) SaveMetrics(ctx context.Context, metrics *PerformanceMetrics) error {
	query := `
		INSERT INTO performance_metrics (
			tenant_id, period_start, period_end, period_type,
			total_messages, successful_messages, failed_messages,
			average_delivery_time_ms, p95_delivery_time_ms, p99_delivery_time_ms,
			error_count, total_payments, on_time_payments
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
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
	`

	_, err := c.db.ExecContext(ctx, query,
		metrics.TenantID,
		metrics.PeriodStart,
		metrics.PeriodEnd,
		metrics.PeriodType,
		metrics.TotalMessages,
		metrics.SuccessfulMessages,
		metrics.FailedMessages,
		metrics.AverageDeliveryTimeMs,
		metrics.P95DeliveryTimeMs,
		metrics.P99DeliveryTimeMs,
		metrics.ErrorCount,
		metrics.TotalPayments,
		metrics.OnTimePayments,
	)

	if err != nil {
		return fmt.Errorf("failed to save metrics: %w", err)
	}

	return nil
}
