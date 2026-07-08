package billing

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"go.uber.org/zap"
)

// DailyRenewalCheckArgs represents the arguments for the daily renewal check job
type DailyRenewalCheckArgs struct{}

// Kind returns the unique identifier for this job type
func (DailyRenewalCheckArgs) Kind() string {
	return "daily_renewal_check"
}

// DailyRenewalCheckWorker handles the daily renewal check job
type DailyRenewalCheckWorker struct {
	river.WorkerDefaults[DailyRenewalCheckArgs]
	service *AutoRenewalService
	logger  *zap.Logger
}

// NewDailyRenewalCheckWorker creates a new daily renewal check worker
func NewDailyRenewalCheckWorker(service *AutoRenewalService, logger *zap.Logger) *DailyRenewalCheckWorker {
	return &DailyRenewalCheckWorker{
		service: service,
		logger:  logger,
	}
}

// Work executes the daily renewal check
func (w *DailyRenewalCheckWorker) Work(ctx context.Context, job *river.Job[DailyRenewalCheckArgs]) error {
	w.logger.Info("Starting daily renewal check job",
		zap.Int64("job_id", job.ID),
		zap.Time("created_at", job.CreatedAt),
	)

	if err := w.service.CheckExpiredSubscriptions(ctx); err != nil {
		w.logger.Error("Daily renewal check failed",
			zap.Int64("job_id", job.ID),
			zap.Error(err),
		)
		return fmt.Errorf("daily renewal check failed: %w", err)
	}

	w.logger.Info("Daily renewal check completed successfully",
		zap.Int64("job_id", job.ID),
	)

	return nil
}

// ScheduleDailyRenewalCheck schedules the daily renewal check job
// This should be called once at application startup
func ScheduleDailyRenewalCheck(ctx context.Context, client *river.Client[pgx.Tx], logger *zap.Logger) error {
	logger.Info("Scheduling daily renewal check job")

	// Schedule job to run daily at 2 AM
	_, err := client.InsertTx(ctx, nil, DailyRenewalCheckArgs{}, &river.InsertOpts{
		Queue:    "billing",
		Priority: 1,
		// Schedule for next 2 AM
		ScheduledAt: getNext2AM(),
	})
	if err != nil {
		return fmt.Errorf("failed to schedule daily renewal check: %w", err)
	}

	logger.Info("Daily renewal check job scheduled successfully")
	return nil
}

// getNext2AM returns the next 2 AM time
func getNext2AM() time.Time {
	// This will be implemented to return the next 2 AM
	// For now, we'll use a simple implementation
	return time.Now().Add(24 * time.Hour)
}
