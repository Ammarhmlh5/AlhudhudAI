package metrics

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

// Aggregator handles automatic aggregation of performance metrics
type Aggregator struct {
	repo   *Repository
	cron   *cron.Cron
	logger *log.Logger
}

// NewAggregator creates a new metrics aggregator
func NewAggregator(repo *Repository, logger *log.Logger) *Aggregator {
	if logger == nil {
		logger = log.Default()
	}
	
	return &Aggregator{
		repo:   repo,
		cron:   cron.New(),
		logger: logger,
	}
}

// Start starts the automatic aggregation cron jobs
// Requirements: 4.2
func (a *Aggregator) Start() error {
	configs := DefaultAggregationConfigs()
	
	for _, config := range configs {
		config := config // capture loop variable
		
		_, err := a.cron.AddFunc(config.CronSchedule, func() {
			ctx := context.Background()
			
			switch config.PeriodType {
			case "hourly":
				if err := a.AggregateHourlyForAllTenants(ctx); err != nil {
					a.logger.Printf("Error aggregating hourly metrics: %v", err)
				}
			case "daily":
				if err := a.AggregateDailyForAllTenants(ctx); err != nil {
					a.logger.Printf("Error aggregating daily metrics: %v", err)
				}
			case "monthly":
				if err := a.AggregateMonthlyForAllTenants(ctx); err != nil {
					a.logger.Printf("Error aggregating monthly metrics: %v", err)
				}
			}
		})
		
		if err != nil {
			return fmt.Errorf("failed to add cron job for %s: %w", config.PeriodType, err)
		}
		
		a.logger.Printf("Scheduled %s aggregation with cron: %s", config.PeriodType, config.CronSchedule)
	}
	
	a.cron.Start()
	a.logger.Println("Metrics aggregator started")
	
	return nil
}

// Stop stops the aggregator
func (a *Aggregator) Stop() {
	if a.cron != nil {
		a.cron.Stop()
		a.logger.Println("Metrics aggregator stopped")
	}
}

// AggregateHourly aggregates metrics for the previous hour for a specific tenant
// Requirements: 4.2
func (a *Aggregator) AggregateHourly(ctx context.Context, tenantID uuid.UUID) error {
	now := time.Now()
	
	// Calculate the previous hour period
	periodEnd := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	periodStart := periodEnd.Add(-1 * time.Hour)
	
	return a.aggregateForPeriod(ctx, tenantID, "hourly", periodStart, periodEnd)
}

// AggregateDaily aggregates metrics for the previous day for a specific tenant
// Requirements: 4.2
func (a *Aggregator) AggregateDaily(ctx context.Context, tenantID uuid.UUID) error {
	now := time.Now()
	
	// Calculate the previous day period
	periodEnd := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	periodStart := periodEnd.AddDate(0, 0, -1)
	
	return a.aggregateForPeriod(ctx, tenantID, "daily", periodStart, periodEnd)
}

// AggregateMonthly aggregates metrics for the previous month for a specific tenant
// Requirements: 4.2
func (a *Aggregator) AggregateMonthly(ctx context.Context, tenantID uuid.UUID) error {
	now := time.Now()
	
	// Calculate the previous month period
	periodEnd := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	periodStart := periodEnd.AddDate(0, -1, 0)
	
	return a.aggregateForPeriod(ctx, tenantID, "monthly", periodStart, periodEnd)
}

// aggregateForPeriod aggregates metrics for a specific period
func (a *Aggregator) aggregateForPeriod(ctx context.Context, tenantID uuid.UUID, periodType string, periodStart, periodEnd time.Time) error {
	// Aggregate metrics from source data
	metrics, err := a.repo.AggregateMetrics(ctx, tenantID, periodType, periodStart, periodEnd)
	if err != nil {
		return fmt.Errorf("failed to aggregate %s metrics for tenant %s: %w", periodType, tenantID, err)
	}
	
	// Save aggregated metrics
	err = a.repo.SaveMetrics(ctx, metrics)
	if err != nil {
		return fmt.Errorf("failed to save %s metrics for tenant %s: %w", periodType, tenantID, err)
	}
	
	a.logger.Printf("Aggregated %s metrics for tenant %s: %d messages, %.2f%% success rate",
		periodType, tenantID, metrics.TotalMessages, metrics.SuccessRate)
	
	return nil
}

// AggregateHourlyForAllTenants aggregates hourly metrics for all active tenants
func (a *Aggregator) AggregateHourlyForAllTenants(ctx context.Context) error {
	tenantIDs, err := a.getActiveTenantIDs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active tenants: %w", err)
	}
	
	a.logger.Printf("Starting hourly aggregation for %d tenants", len(tenantIDs))
	
	successCount := 0
	errorCount := 0
	
	for _, tenantID := range tenantIDs {
		if err := a.AggregateHourly(ctx, tenantID); err != nil {
			a.logger.Printf("Error aggregating hourly metrics for tenant %s: %v", tenantID, err)
			errorCount++
		} else {
			successCount++
		}
	}
	
	a.logger.Printf("Hourly aggregation completed: %d succeeded, %d failed", successCount, errorCount)
	
	return nil
}

// AggregateDailyForAllTenants aggregates daily metrics for all active tenants
func (a *Aggregator) AggregateDailyForAllTenants(ctx context.Context) error {
	tenantIDs, err := a.getActiveTenantIDs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active tenants: %w", err)
	}
	
	a.logger.Printf("Starting daily aggregation for %d tenants", len(tenantIDs))
	
	successCount := 0
	errorCount := 0
	
	for _, tenantID := range tenantIDs {
		if err := a.AggregateDaily(ctx, tenantID); err != nil {
			a.logger.Printf("Error aggregating daily metrics for tenant %s: %v", tenantID, err)
			errorCount++
		} else {
			successCount++
		}
	}
	
	a.logger.Printf("Daily aggregation completed: %d succeeded, %d failed", successCount, errorCount)
	
	return nil
}

// AggregateMonthlyForAllTenants aggregates monthly metrics for all active tenants
func (a *Aggregator) AggregateMonthlyForAllTenants(ctx context.Context) error {
	tenantIDs, err := a.getActiveTenantIDs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active tenants: %w", err)
	}
	
	a.logger.Printf("Starting monthly aggregation for %d tenants", len(tenantIDs))
	
	successCount := 0
	errorCount := 0
	
	for _, tenantID := range tenantIDs {
		if err := a.AggregateMonthly(ctx, tenantID); err != nil {
			a.logger.Printf("Error aggregating monthly metrics for tenant %s: %v", tenantID, err)
			errorCount++
		} else {
			successCount++
		}
	}
	
	a.logger.Printf("Monthly aggregation completed: %d succeeded, %d failed", successCount, errorCount)
	
	return nil
}

// getActiveTenantIDs retrieves all active tenant IDs from the database
func (a *Aggregator) getActiveTenantIDs(ctx context.Context) ([]uuid.UUID, error) {
	query := `
		SELECT id FROM tenants
		WHERE is_active = true
		ORDER BY id
	`
	
	var tenantIDs []uuid.UUID
	err := a.repo.db.SelectContext(ctx, &tenantIDs, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active tenants: %w", err)
	}
	
	return tenantIDs, nil
}

// AggregateCustomPeriod aggregates metrics for a custom time period
func (a *Aggregator) AggregateCustomPeriod(ctx context.Context, tenantID uuid.UUID, periodStart, periodEnd time.Time) (*PerformanceMetrics, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("tenant_id is required")
	}
	if periodEnd.Before(periodStart) || periodEnd.Equal(periodStart) {
		return nil, fmt.Errorf("period_end must be after period_start")
	}
	
	// Determine period type based on duration
	duration := periodEnd.Sub(periodStart)
	var periodType string
	
	switch {
	case duration <= 2*time.Hour:
		periodType = "hourly"
	case duration <= 48*time.Hour:
		periodType = "daily"
	default:
		periodType = "monthly"
	}
	
	// Aggregate metrics
	metrics, err := a.repo.AggregateMetrics(ctx, tenantID, periodType, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate custom period metrics: %w", err)
	}
	
	return metrics, nil
}

// CleanupOldMetrics removes metrics older than the retention period
func (a *Aggregator) CleanupOldMetrics(ctx context.Context, retentionMonths int) (int64, error) {
	if retentionMonths <= 0 {
		return 0, fmt.Errorf("retention_months must be positive")
	}
	
	deleted, err := a.repo.DeleteOldMetrics(ctx, retentionMonths)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old metrics: %w", err)
	}
	
	a.logger.Printf("Cleaned up %d old metrics records (retention: %d months)", deleted, retentionMonths)
	
	return deleted, nil
}
