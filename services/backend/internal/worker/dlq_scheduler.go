package worker

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// DLQScheduler handles scheduled cleanup of old DLQ messages
// Implements Requirement 17.6: الاحتفاظ بالرسائل في Dead Letter Queue لمدة 7 أيام
type DLQScheduler struct {
	dlq             *DeadLetterQueue
	logger          *zap.Logger
	retentionDays   int
	cleanupInterval time.Duration
	stopChan        chan struct{}
}

// NewDLQScheduler creates a new DLQ scheduler
func NewDLQScheduler(dlq *DeadLetterQueue, logger *zap.Logger) *DLQScheduler {
	return &DLQScheduler{
		dlq:             dlq,
		logger:          logger,
		retentionDays:   7,                // Default: 7 days retention
		cleanupInterval: 24 * time.Hour,   // Default: cleanup once per day
		stopChan:        make(chan struct{}),
	}
}

// WithRetentionDays sets the retention period in days
func (s *DLQScheduler) WithRetentionDays(days int) *DLQScheduler {
	s.retentionDays = days
	return s
}

// WithCleanupInterval sets the cleanup interval
func (s *DLQScheduler) WithCleanupInterval(interval time.Duration) *DLQScheduler {
	s.cleanupInterval = interval
	return s
}

// Start starts the cleanup scheduler
func (s *DLQScheduler) Start(ctx context.Context) {
	s.logger.Info("Starting DLQ cleanup scheduler",
		zap.Int("retention_days", s.retentionDays),
		zap.Duration("cleanup_interval", s.cleanupInterval),
	)

	// Run cleanup immediately on start
	go s.runCleanup(ctx)

	// Schedule periodic cleanup
	ticker := time.NewTicker(s.cleanupInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.runCleanup(ctx)
			case <-s.stopChan:
				s.logger.Info("DLQ cleanup scheduler stopped")
				return
			case <-ctx.Done():
				s.logger.Info("DLQ cleanup scheduler stopped due to context cancellation")
				return
			}
		}
	}()
}

// Stop stops the cleanup scheduler
func (s *DLQScheduler) Stop() {
	close(s.stopChan)
}

// runCleanup performs the cleanup operation
func (s *DLQScheduler) runCleanup(ctx context.Context) {
	s.logger.Info("Running DLQ cleanup",
		zap.Int("retention_days", s.retentionDays),
	)

	startTime := time.Now()
	count, err := s.dlq.CleanupOldMessages(ctx, s.retentionDays)
	duration := time.Since(startTime)

	if err != nil {
		s.logger.Error("DLQ cleanup failed",
			zap.Error(err),
			zap.Duration("duration", duration),
		)
		return
	}

	if count > 0 {
		s.logger.Info("DLQ cleanup completed",
			zap.Int64("deleted_count", count),
			zap.Duration("duration", duration),
			zap.Int("retention_days", s.retentionDays),
		)
	} else {
		s.logger.Debug("DLQ cleanup completed - no old messages found",
			zap.Duration("duration", duration),
			zap.Int("retention_days", s.retentionDays),
		)
	}
}
