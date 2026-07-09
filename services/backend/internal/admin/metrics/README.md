# Performance Metrics System

This package implements the Performance Metrics System for the Admin Interface Enhancements feature.

## Overview

The Performance Metrics System tracks and aggregates company performance metrics including:
- Success rate (successful messages / total messages)
- Sending volume (total messages per time period)
- Payment compliance (on-time payments / total payments)
- Service quality (delivery speed, error rate)

## Components

### Repository (`repository.go`)

Handles database operations for performance metrics:
- `SaveMetrics()` - Saves or updates metrics (uses prepared statements for performance)
- `GetMetrics()` - Retrieves metrics with filtering by period type and time range
- `AggregateMetrics()` - Aggregates raw data from messages and payments tables
- `GetLatestMetrics()` - Gets the most recent metrics for a tenant
- `DeleteOldMetrics()` - Cleans up old metrics based on retention period

**Requirements**: 4.1, 4.2, 4.6

### Aggregator (`aggregator.go`)

Handles automatic aggregation of metrics using cron jobs:
- `AggregateHourly()` - Aggregates metrics for the previous hour
- `AggregateDaily()` - Aggregates metrics for the previous day
- `AggregateMonthly()` - Aggregates metrics for the previous month
- `Start()` - Starts automatic aggregation with cron schedules:
  - Hourly: `0 * * * *` (every hour at minute 0)
  - Daily: `0 0 * * *` (every day at midnight)
  - Monthly: `0 0 1 * *` (first day of month at midnight)

**Requirements**: 4.2

### Threshold Monitor (`threshold.go`)

Monitors metrics against configured thresholds and triggers alerts:
- `CheckThresholds()` - Checks metrics against thresholds for a tenant
- `TriggerAlert()` - Creates an alert for threshold violations
- `GetThresholdConfig()` - Retrieves threshold configuration
- `SaveThresholdConfig()` - Saves threshold configuration
- `StartPeriodicChecking()` - Starts periodic threshold checking

**Requirements**: 4.8

## Usage

### Initialize the System

```go
import (
    "github.com/jmoiron/sqlx"
    "log"
    "your-project/internal/admin/metrics"
)

// Create repository
repo, err := metrics.NewRepository(db)
if err != nil {
    log.Fatal(err)
}
defer repo.Close()

// Create aggregator
logger := log.Default()
aggregator := metrics.NewAggregator(repo, logger)

// Start automatic aggregation
if err := aggregator.Start(); err != nil {
    log.Fatal(err)
}
defer aggregator.Stop()

// Create threshold monitor
monitor := metrics.NewThresholdMonitor(db, repo, logger)

// Start periodic threshold checking (every 5 minutes)
go monitor.StartPeriodicChecking(context.Background(), 5*time.Minute)
```

### Save Metrics

```go
metrics := &metrics.PerformanceMetrics{
    TenantID:           tenantID,
    PeriodStart:        time.Now().Add(-1 * time.Hour),
    PeriodEnd:          time.Now(),
    PeriodType:         "hourly",
    TotalMessages:      1000,
    SuccessfulMessages: 950,
    FailedMessages:     50,
    ErrorCount:         10,
    TotalPayments:      5,
    OnTimePayments:     4,
}

err := repo.SaveMetrics(ctx, metrics)
```

### Query Metrics

```go
filter := metrics.MetricsFilter{
    TenantID:   tenantID,
    PeriodType: stringPtr("daily"),
    StartTime:  timePtr(time.Now().AddDate(0, 0, -7)),
    EndTime:    timePtr(time.Now()),
    Limit:      100,
}

results, err := repo.GetMetrics(ctx, filter)
```

### Configure Thresholds

```go
minSuccessRate := 95.0
maxErrorRate := 5.0
minPaymentCompliance := 90.0
maxDeliveryTimeMs := 5000

config := &metrics.ThresholdConfig{
    TenantID:             tenantID,
    MinSuccessRate:       &minSuccessRate,
    MaxErrorRate:         &maxErrorRate,
    MinPaymentCompliance: &minPaymentCompliance,
    MaxDeliveryTimeMs:    &maxDeliveryTimeMs,
}

err := monitor.SaveThresholdConfig(ctx, config)
```

### Check Thresholds

```go
violations, err := monitor.CheckThresholds(ctx, tenantID)
if err != nil {
    log.Fatal(err)
}

for _, violation := range violations {
    // Trigger alert
    if err := monitor.TriggerAlert(ctx, violation); err != nil {
        log.Printf("Error triggering alert: %v", err)
    }
}
```

## Database Schema

### performance_metrics Table

Stores aggregated performance metrics:
- `tenant_id` - Company/tenant identifier
- `period_start`, `period_end` - Time period boundaries
- `period_type` - Type of period: hourly, daily, monthly
- `total_messages`, `successful_messages`, `failed_messages` - Message counts
- `success_rate` - Computed: (successful / total) * 100
- `error_count`, `error_rate` - Error tracking
- `total_payments`, `on_time_payments` - Payment tracking
- `payment_compliance` - Computed: (on_time / total) * 100
- `average_delivery_time_ms`, `p95_delivery_time_ms`, `p99_delivery_time_ms` - Delivery metrics

### threshold_configs Table (to be created)

Stores threshold configurations per tenant:
- `tenant_id` - Company/tenant identifier
- `min_success_rate` - Minimum acceptable success rate
- `max_error_rate` - Maximum acceptable error rate
- `min_payment_compliance` - Minimum acceptable payment compliance
- `max_delivery_time_ms` - Maximum acceptable delivery time

## Default Thresholds

If no custom thresholds are configured, the system uses these defaults:
- Minimum Success Rate: 95%
- Maximum Error Rate: 5%
- Minimum Payment Compliance: 90%
- Maximum Delivery Time: 5000ms

## Alert Severity Levels

Alerts are assigned severity based on deviation from threshold:
- **Critical**: Deviation >= 50%
- **Warning**: Deviation >= 20%
- **Info**: Deviation < 20%

## Alert Deduplication

Alerts are deduplicated using a `dedup_key` to prevent duplicate alerts within 5 minutes for the same metric and tenant.

## Integration with Alert System

The threshold monitor integrates with the alerts table (migration 039) to store alerts. Alerts include:
- Alert type: "threshold_violation"
- Severity: "critical", "warning", or "info"
- Context: JSON with metric details
- Dedup key: Prevents duplicate alerts

## Performance Considerations

1. **Prepared Statements**: The repository uses prepared statements for SaveMetrics() to improve performance
2. **Indexes**: Database indexes on (tenant_id, period_start) for fast queries
3. **Aggregation**: Metrics are pre-aggregated hourly/daily/monthly to avoid expensive queries
4. **Cron Jobs**: Automatic aggregation runs at off-peak times (midnight for daily, first of month for monthly)

## Testing

Property-based tests should verify:
- Success rate calculation correctness (Property 16)
- Volume aggregation accuracy (Property 17)
- Payment compliance calculation (Property 18)
- Threshold violation detection
- Alert deduplication

## Future Enhancements

1. Add support for custom aggregation periods
2. Implement trend analysis and anomaly detection
3. Add support for metric comparisons across tenants
4. Implement metric forecasting
5. Add support for custom alert channels (email, SMS, webhook)
