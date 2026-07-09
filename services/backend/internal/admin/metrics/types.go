package metrics

import (
	"time"

	"github.com/google/uuid"
)

// PerformanceMetrics represents the performance metrics for a tenant
type PerformanceMetrics struct {
	ID                    int64      `json:"id" db:"id"`
	TenantID              uuid.UUID  `json:"tenant_id" db:"tenant_id" validate:"required"`
	PeriodStart           time.Time  `json:"period_start" db:"period_start" validate:"required"`
	PeriodEnd             time.Time  `json:"period_end" db:"period_end" validate:"required"`
	PeriodType            string     `json:"period_type" db:"period_type" validate:"required,oneof=hourly daily monthly"`
	TotalMessages         int64      `json:"total_messages" db:"total_messages" validate:"gte=0"`
	SuccessfulMessages    int64      `json:"successful_messages" db:"successful_messages" validate:"gte=0"`
	FailedMessages        int64      `json:"failed_messages" db:"failed_messages" validate:"gte=0"`
	SuccessRate           float64    `json:"success_rate" db:"success_rate"`
	AverageDeliveryTimeMs *int       `json:"average_delivery_time_ms,omitempty" db:"average_delivery_time_ms"`
	P95DeliveryTimeMs     *int       `json:"p95_delivery_time_ms,omitempty" db:"p95_delivery_time_ms"`
	P99DeliveryTimeMs     *int       `json:"p99_delivery_time_ms,omitempty" db:"p99_delivery_time_ms"`
	ErrorCount            int        `json:"error_count" db:"error_count" validate:"gte=0"`
	ErrorRate             float64    `json:"error_rate" db:"error_rate"`
	TotalPayments         int        `json:"total_payments" db:"total_payments" validate:"gte=0"`
	OnTimePayments        int        `json:"on_time_payments" db:"on_time_payments" validate:"gte=0"`
	PaymentCompliance     float64    `json:"payment_compliance" db:"payment_compliance"`
	CreatedAt             time.Time  `json:"created_at" db:"created_at"`
}

// MetricsFilter represents filters for querying performance metrics
type MetricsFilter struct {
	TenantID   uuid.UUID  `json:"tenant_id" validate:"required"`
	PeriodType *string    `json:"period_type,omitempty"`
	StartTime  *time.Time `json:"start_time,omitempty"`
	EndTime    *time.Time `json:"end_time,omitempty"`
	Limit      int        `json:"limit"`
	Offset     int        `json:"offset"`
}

// AggregationConfig represents configuration for metrics aggregation
type AggregationConfig struct {
	PeriodType string `json:"period_type" validate:"required,oneof=hourly daily monthly"`
	// Cron schedule for automatic aggregation
	CronSchedule string `json:"cron_schedule"`
}

// DefaultAggregationConfigs returns default aggregation configurations
func DefaultAggregationConfigs() []AggregationConfig {
	return []AggregationConfig{
		{
			PeriodType:   "hourly",
			CronSchedule: "0 * * * *", // Every hour at minute 0
		},
		{
			PeriodType:   "daily",
			CronSchedule: "0 0 * * *", // Every day at midnight
		},
		{
			PeriodType:   "monthly",
			CronSchedule: "0 0 1 * *", // First day of every month at midnight
		},
	}
}

// ThresholdConfig represents threshold configuration for alerts
type ThresholdConfig struct {
	TenantID              uuid.UUID `json:"tenant_id" validate:"required"`
	MinSuccessRate        *float64  `json:"min_success_rate,omitempty"`
	MaxErrorRate          *float64  `json:"max_error_rate,omitempty"`
	MinPaymentCompliance  *float64  `json:"min_payment_compliance,omitempty"`
	MaxDeliveryTimeMs     *int      `json:"max_delivery_time_ms,omitempty"`
}

// ThresholdViolation represents a threshold violation
type ThresholdViolation struct {
	TenantID      uuid.UUID `json:"tenant_id"`
	MetricName    string    `json:"metric_name"`
	CurrentValue  float64   `json:"current_value"`
	ThresholdValue float64   `json:"threshold_value"`
	Severity      string    `json:"severity"` // "warning", "critical"
	DetectedAt    time.Time `json:"detected_at"`
}

// MetricsSummary represents a summary of metrics for dashboard display
type MetricsSummary struct {
	TenantID              uuid.UUID `json:"tenant_id"`
	Period                string    `json:"period"` // "last_hour", "last_24h", "last_7d", "last_30d"
	TotalMessages         int64     `json:"total_messages"`
	SuccessRate           float64   `json:"success_rate"`
	ErrorRate             float64   `json:"error_rate"`
	PaymentCompliance     float64   `json:"payment_compliance"`
	AverageDeliveryTimeMs *int      `json:"average_delivery_time_ms,omitempty"`
}
