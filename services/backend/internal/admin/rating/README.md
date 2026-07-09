# Company Rating Engine

نظام تقييم الشركات بناءً على مؤشرات الأداء (Company Rating System based on Performance Metrics)

## Overview

The Company Rating Engine calculates and maintains ratings for companies based on their performance metrics. It integrates with multiple data sources (gateway statistics, tariff system, resource usage) to provide comprehensive company ratings.

## Components

### 1. Data Models (`types.go`)

Defines the core data structures:
- **CompanyRating**: Represents a company's rating at a specific point in time
- **PerformanceMetrics**: Performance metrics used to calculate ratings
- **RatingWeights**: Weights used in rating calculation (default: 40% success rate, 20% volume, 25% payment, 15% quality)
- **RatingCategory**: Classification (Excellent ≥90%, Good 75-89%, Average 60-74%, Poor <60%)

### 2. Rating Calculator (`calculator.go`)

Calculates company ratings based on weighted formulas:

```go
calculator := rating.NewDefaultCalculator()
rating := calculator.CalculateRating(metrics)
category := calculator.ClassifyRating(rating)
```

**Rating Formula:**
```
Rating = (SuccessRate × 0.40) + (VolumeScore × 0.20) + (PaymentCompliance × 0.25) + (QualityScore × 0.15)
```

**Quality Score Calculation:**
- Base score: 100
- Error rate penalty: 0-50 points (based on error rate)
- Delivery time penalty: 0-30 points (based on average delivery time)
- Consistency bonus: 0-20 points (based on P95/P99 consistency)

### 3. Metrics Collector (`collector.go`)

Collects performance metrics from multiple sources:

```go
collector := rating.NewCollector(db)
metrics, err := collector.CollectCurrentMetrics(ctx, tenantID)
```

**Data Sources:**
- **gateway_statistics**: Message statistics (sent, delivered, failed, response times)
- **billing_invoices**: Payment compliance data
- **resource_usage**: Resource utilization metrics

**Collection Methods:**
- `CollectCurrentMetrics()`: Last 24 hours
- `CollectHourlyMetrics()`: Specific hour
- `CollectDailyMetrics()`: Specific day
- `CollectMonthlyMetrics()`: Specific month

### 4. Rating Update Trigger (`trigger.go`)

Handles rating update events with debouncing (max once per 5 seconds):

```go
trigger := rating.NewTrigger(db, collector, calculator, history)

// Event handlers
trigger.OnMessageSent(ctx, tenantID)
trigger.OnMessageFailed(ctx, tenantID)
trigger.OnPaymentReceived(ctx, tenantID)

// Manual update
trigger.UpdateRating(ctx, tenantID)

// Periodic updates
go trigger.StartPeriodicUpdates(ctx, time.Hour)
```

**Features:**
- Debouncing: Prevents excessive updates (5-second minimum interval)
- Asynchronous updates: Non-blocking event handling
- Category change alerts: Automatic alert generation when rating category changes
- Batch updates: Update all companies at once

### 5. History Manager (`history.go`)

Manages rating history with 12-month retention:

```go
history := rating.NewHistoryManager(db)

// Save rating
history.SaveRatingHistory(ctx, rating)

// Get latest rating
latest, err := history.GetLatestRating(ctx, tenantID)

// Get history with filters
filter := &rating.RatingHistoryFilter{
    TenantID:  tenantID,
    StartDate: &startDate,
    EndDate:   &endDate,
    Category:  &category,
    Limit:     100,
    Offset:    0,
}
ratings, err := history.GetRatingHistory(ctx, filter)

// Cleanup old records (>12 months)
deleted, err := history.CleanupOldRecords(ctx)
```

**Features:**
- Temporal validity: Tracks valid_from and valid_to for each rating
- Historical queries: Get rating at any point in time
- Trend analysis: Get rating trends over time periods
- Category distribution: Get distribution of companies across categories
- Automatic cleanup: Removes records older than 12 months

## Usage Example

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/jmoiron/sqlx"
    "your-project/internal/admin/rating"
)

func main() {
    // Initialize database connection
    db, err := sqlx.Connect("postgres", "connection-string")
    if err != nil {
        panic(err)
    }
    defer db.Close()
    
    // Create components
    collector := rating.NewCollector(db)
    calculator := rating.NewDefaultCalculator()
    history := rating.NewHistoryManager(db)
    trigger := rating.NewTrigger(db, collector, calculator, history)
    
    ctx := context.Background()
    tenantID := uuid.MustParse("your-tenant-id")
    
    // Collect metrics and calculate rating
    metrics, err := collector.CollectCurrentMetrics(ctx, tenantID)
    if err != nil {
        panic(err)
    }
    
    ratingValue := calculator.CalculateRating(metrics)
    category := calculator.ClassifyRating(ratingValue)
    
    fmt.Printf("Rating: %.2f (%s)\n", ratingValue, category)
    
    // Save to history
    newRating := &rating.CompanyRating{
        TenantID:             tenantID,
        SuccessRate:          metrics.SuccessRate,
        SendingVolume:        metrics.TotalMessages,
        PaymentCompliance:    metrics.PaymentCompliance,
        ServiceQualityScore:  calculator.CalculateQualityScore(metrics),
        OverallRating:        ratingValue,
        RatingCategory:       category,
        CalculationTimestamp: time.Now(),
        ValidFrom:            time.Now(),
    }
    
    if err := history.SaveRatingHistory(ctx, newRating); err != nil {
        panic(err)
    }
    
    // Start periodic updates (every hour)
    go trigger.StartPeriodicUpdates(ctx, time.Hour)
    
    // Handle events
    trigger.OnMessageSent(ctx, tenantID)
}
```

## Database Schema

### company_ratings Table

```sql
CREATE TABLE company_ratings (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    success_rate DECIMAL(5,2) NOT NULL,
    sending_volume BIGINT NOT NULL,
    payment_compliance DECIMAL(5,2) NOT NULL,
    service_quality_score DECIMAL(5,2) NOT NULL,
    overall_rating DECIMAL(5,2) NOT NULL,
    rating_category VARCHAR(20) NOT NULL,
    calculation_timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
    valid_from TIMESTAMP NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMP,
    
    INDEX idx_company_ratings_tenant (tenant_id, calculation_timestamp DESC),
    INDEX idx_company_ratings_category (rating_category),
    INDEX idx_company_ratings_valid (valid_from, valid_to)
);
```

### performance_metrics Table

```sql
CREATE TABLE performance_metrics (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    period_start TIMESTAMP NOT NULL,
    period_end TIMESTAMP NOT NULL,
    period_type VARCHAR(20) NOT NULL,
    total_messages BIGINT NOT NULL DEFAULT 0,
    successful_messages BIGINT NOT NULL DEFAULT 0,
    failed_messages BIGINT NOT NULL DEFAULT 0,
    success_rate DECIMAL(5,2) GENERATED ALWAYS AS (...) STORED,
    average_delivery_time_ms INTEGER,
    p95_delivery_time_ms INTEGER,
    p99_delivery_time_ms INTEGER,
    error_count INTEGER NOT NULL DEFAULT 0,
    error_rate DECIMAL(5,2) GENERATED ALWAYS AS (...) STORED,
    total_payments INTEGER NOT NULL DEFAULT 0,
    on_time_payments INTEGER NOT NULL DEFAULT 0,
    payment_compliance DECIMAL(5,2) GENERATED ALWAYS AS (...) STORED,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    UNIQUE INDEX idx_performance_metrics_unique (tenant_id, period_start, period_type)
);
```

## Integration Points

### 1. Gateway Statistics
- **Table**: `gateway_statistics`
- **Data**: Message counts, delivery times, success rates
- **Usage**: Primary source for message performance metrics

### 2. Tariff System
- **Table**: `billing_invoices`
- **Data**: Payment status, due dates, paid dates
- **Usage**: Calculate payment compliance

### 3. Resource Usage
- **Table**: `resource_usage`
- **Data**: CPU, memory, queue depth
- **Usage**: Additional quality metrics

### 4. Alert System
- **Table**: `alerts`
- **Integration**: Automatic alert generation on category changes
- **Severity**: Critical (Poor), Warning (downgrade), Info (upgrade)

## Performance Considerations

1. **Debouncing**: Prevents excessive updates (5-second minimum interval per company)
2. **Asynchronous Updates**: Event handlers trigger updates in goroutines
3. **Indexed Queries**: All queries use indexed columns (tenant_id, timestamps)
4. **Batch Processing**: `UpdateAllRatings()` for periodic batch updates
5. **Automatic Cleanup**: `CleanupOldRecords()` removes data older than 12 months

## Testing

### Unit Tests
- Test specific rating calculations
- Test edge cases (zero messages, negative values)
- Test error conditions (invalid weights, missing data)

### Property Tests (Optional)
- **Property 1**: Rating Calculation Correctness
- **Property 2**: Rating Category Classification
- Use `gopter` for property-based testing

## Monitoring

### Metrics to Monitor
- Rating calculation duration
- Number of rating updates per hour
- Category distribution
- Alert generation rate
- Database query performance

### Alerts to Configure
- Rating calculation failures
- Database connection errors
- Excessive update rate (potential infinite loop)
- Category distribution anomalies

## Maintenance

### Daily Tasks
- Monitor rating update logs
- Check alert generation

### Weekly Tasks
- Review category distribution
- Analyze rating trends

### Monthly Tasks
- Run `CleanupOldRecords()` to remove old data
- Review and adjust rating weights if needed
- Analyze performance metrics

## Requirements Validation

This implementation validates the following requirements:

- **Requirement 1.1**: ✅ Calculate ratings based on success rate, volume, payment, quality
- **Requirement 1.2**: ✅ Recalculate within 5 seconds (debouncing)
- **Requirement 1.3**: ✅ Display KPIs (via metrics collection)
- **Requirement 1.4**: ✅ Classify into categories (Excellent, Good, Average, Poor)
- **Requirement 1.5**: ✅ Notify on category change (via alerts)
- **Requirement 1.6**: ✅ Maintain 12-month history
- **Requirement 1.7**: ✅ Filtering and sorting capabilities

## Future Enhancements

1. **Machine Learning**: Predictive rating trends
2. **Custom Weights**: Per-company weight configuration
3. **Real-time Updates**: WebSocket notifications
4. **Advanced Analytics**: Correlation analysis, anomaly detection
5. **Benchmarking**: Compare companies within same industry/size
