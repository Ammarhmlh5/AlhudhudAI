package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/al-hudhud/backend/internal/database"
	"github.com/al-hudhud/backend/internal/queue"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"go.uber.org/zap"
)

// RecoveryManager handles automatic recovery of pending messages after system failure
// Implements Requirements 24.4, 24.6
type RecoveryManager struct {
	db     *database.DB
	logger *zap.Logger
	client *river.Client[pgx.Tx]
	config *RecoveryConfig
}

// RecoveryConfig represents recovery configuration
type RecoveryConfig struct {
	// EnableAutoRecovery enables automatic recovery on startup
	// Default: true (as per Requirement 24.4)
	EnableAutoRecovery bool

	// RecoveryBatchSize is the number of messages to recover per batch
	// Default: 100
	RecoveryBatchSize int

	// RecoveryTimeout is the maximum time for recovery process
	// Default: 5 minutes
	RecoveryTimeout time.Duration

	// StaleMessageThreshold is the time after which a message is considered stale
	// Messages in PROCESSING state for longer than this are recovered
	// Default: 5 minutes
	StaleMessageThreshold time.Duration

	// MaxRecoveryAttempts is the maximum number of recovery attempts per message
	// Default: 3
	MaxRecoveryAttempts int
}

// DefaultRecoveryConfig returns default recovery configuration
func DefaultRecoveryConfig() *RecoveryConfig {
	return &RecoveryConfig{
		EnableAutoRecovery:    true,
		RecoveryBatchSize:     100,
		RecoveryTimeout:       5 * time.Minute,
		StaleMessageThreshold: 5 * time.Minute,
		MaxRecoveryAttempts:   3,
	}
}

// NewRecoveryManager creates a new recovery manager
func NewRecoveryManager(
	db *database.DB,
	logger *zap.Logger,
	client *river.Client[pgx.Tx],
	config *RecoveryConfig,
) *RecoveryManager {
	if config == nil {
		config = DefaultRecoveryConfig()
	}

	return &RecoveryManager{
		db:     db,
		logger: logger,
		client: client,
		config: config,
	}
}

// RecoverPendingMessages recovers all pending messages after system restart
// Implements Requirement 24.6: استكمال معالجة الرسائل المتبقية بعد عطل
func (rm *RecoveryManager) RecoverPendingMessages(ctx context.Context) error {
	if !rm.config.EnableAutoRecovery {
		rm.logger.Info("Automatic recovery is disabled")
		return nil
	}

	rm.logger.Info("Starting automatic recovery of pending messages",
		zap.Int("batch_size", rm.config.RecoveryBatchSize),
		zap.Duration("timeout", rm.config.RecoveryTimeout),
		zap.Duration("stale_threshold", rm.config.StaleMessageThreshold),
	)

	// Create context with timeout
	recoveryCtx, cancel := context.WithTimeout(ctx, rm.config.RecoveryTimeout)
	defer cancel()

	// Step 1: Recover stale PROCESSING messages
	// These are messages that were being processed when the system crashed
	staleCount, err := rm.recoverStaleMessages(recoveryCtx)
	if err != nil {
		rm.logger.Error("Failed to recover stale messages", zap.Error(err))
		return fmt.Errorf("recover stale messages: %w", err)
	}

	// Step 2: Recover PENDING messages without River jobs
	// These are messages that were inserted but jobs weren't created
	pendingCount, err := rm.recoverPendingMessages(recoveryCtx)
	if err != nil {
		rm.logger.Error("Failed to recover pending messages", zap.Error(err))
		return fmt.Errorf("recover pending messages: %w", err)
	}

	// Step 3: Recover failed River jobs that should be retried
	// These are jobs that failed but haven't reached max attempts
	failedCount, err := rm.recoverFailedJobs(recoveryCtx)
	if err != nil {
		rm.logger.Error("Failed to recover failed jobs", zap.Error(err))
		return fmt.Errorf("recover failed jobs: %w", err)
	}

	totalRecovered := staleCount + pendingCount + failedCount

	rm.logger.Info("Automatic recovery completed successfully",
		zap.Int("stale_messages", staleCount),
		zap.Int("pending_messages", pendingCount),
		zap.Int("failed_jobs", failedCount),
		zap.Int("total_recovered", totalRecovered),
	)

	return nil
}

// recoverStaleMessages recovers messages stuck in PROCESSING state
// Implements Requirement 24.6: استكمال معالجة الرسائل المتبقية بعد عطل
func (rm *RecoveryManager) recoverStaleMessages(ctx context.Context) (int, error) {
	rm.logger.Info("Recovering stale PROCESSING messages",
		zap.Duration("stale_threshold", rm.config.StaleMessageThreshold),
	)

	// Find messages in PROCESSING state for longer than threshold
	staleTime := time.Now().Add(-rm.config.StaleMessageThreshold)

	rows, err := rm.db.Pool.Query(ctx, `
		SELECT 
			tenant_id,
			message_id,
			updated_at
		FROM messages
		WHERE 
			status = 'PROCESSING'
			AND updated_at < $1
			AND deleted_at IS NULL
		ORDER BY updated_at ASC
		LIMIT $2
	`, staleTime, rm.config.RecoveryBatchSize)

	if err != nil {
		return 0, fmt.Errorf("query stale messages: %w", err)
	}
	defer rows.Close()

	recoveredCount := 0

	for rows.Next() {
		var tenantID, messageID uuid.UUID
		var updatedAt time.Time

		if err := rows.Scan(&tenantID, &messageID, &updatedAt); err != nil {
			rm.logger.Error("Failed to scan stale message",
				zap.Error(err),
			)
			continue
		}

		// Reset message status to PENDING and create new River job
		if err := rm.resetMessageAndCreateJob(ctx, tenantID, messageID); err != nil {
			rm.logger.Error("Failed to recover stale message",
				zap.Error(err),
				zap.String("tenant_id", tenantID.String()),
				zap.String("message_id", messageID.String()),
				zap.Time("stale_since", updatedAt),
			)
			continue
		}

		recoveredCount++

		rm.logger.Info("Recovered stale message",
			zap.String("tenant_id", tenantID.String()),
			zap.String("message_id", messageID.String()),
			zap.Duration("stale_duration", time.Since(updatedAt)),
		)
	}

	if err := rows.Err(); err != nil {
		return recoveredCount, fmt.Errorf("iterate stale messages: %w", err)
	}

	rm.logger.Info("Stale messages recovery completed",
		zap.Int("recovered_count", recoveredCount),
	)

	return recoveredCount, nil
}

// recoverPendingMessages recovers PENDING messages without River jobs
// Implements Requirement 24.6: استكمال معالجة الرسائل المتبقية بعد عطل
func (rm *RecoveryManager) recoverPendingMessages(ctx context.Context) (int, error) {
	rm.logger.Info("Recovering PENDING messages without River jobs")

	// Find PENDING messages that don't have corresponding River jobs
	rows, err := rm.db.Pool.Query(ctx, `
		SELECT 
			m.tenant_id,
			m.message_id,
			m.created_at
		FROM messages m
		LEFT JOIN river_job rj ON 
			rj.args->>'tenant_id' = m.tenant_id::text 
			AND rj.args->>'message_id' = m.message_id::text
		WHERE 
			m.status = 'PENDING'
			AND m.deleted_at IS NULL
			AND rj.id IS NULL
		ORDER BY m.created_at ASC
		LIMIT $1
	`, rm.config.RecoveryBatchSize)

	if err != nil {
		return 0, fmt.Errorf("query pending messages: %w", err)
	}
	defer rows.Close()

	recoveredCount := 0

	for rows.Next() {
		var tenantID, messageID uuid.UUID
		var createdAt time.Time

		if err := rows.Scan(&tenantID, &messageID, &createdAt); err != nil {
			rm.logger.Error("Failed to scan pending message",
				zap.Error(err),
			)
			continue
		}

		// Create River job for this message
		if err := rm.createRiverJob(ctx, tenantID, messageID); err != nil {
			rm.logger.Error("Failed to create job for pending message",
				zap.Error(err),
				zap.String("tenant_id", tenantID.String()),
				zap.String("message_id", messageID.String()),
			)
			continue
		}

		recoveredCount++

		rm.logger.Info("Recovered pending message",
			zap.String("tenant_id", tenantID.String()),
			zap.String("message_id", messageID.String()),
			zap.Duration("pending_duration", time.Since(createdAt)),
		)
	}

	if err := rows.Err(); err != nil {
		return recoveredCount, fmt.Errorf("iterate pending messages: %w", err)
	}

	rm.logger.Info("Pending messages recovery completed",
		zap.Int("recovered_count", recoveredCount),
	)

	return recoveredCount, nil
}

// recoverFailedJobs recovers failed River jobs that should be retried
// Implements Requirement 24.6: استكمال معالجة الرسائل المتبقية بعد عطل
func (rm *RecoveryManager) recoverFailedJobs(ctx context.Context) (int, error) {
	rm.logger.Info("Recovering failed River jobs")

	// Find failed jobs that haven't reached max attempts
	rows, err := rm.db.Pool.Query(ctx, `
		SELECT 
			id,
			args->>'tenant_id' as tenant_id,
			args->>'message_id' as message_id,
			attempt,
			max_attempts,
			errors
		FROM river_job
		WHERE 
			state = 'retryable'
			AND attempt < max_attempts
			AND scheduled_at < NOW()
		ORDER BY scheduled_at ASC
		LIMIT $1
	`, rm.config.RecoveryBatchSize)

	if err != nil {
		return 0, fmt.Errorf("query failed jobs: %w", err)
	}
	defer rows.Close()

	recoveredCount := 0

	for rows.Next() {
		var jobID int64
		var tenantIDStr, messageIDStr string
		var attempt, maxAttempts int
		var errors []string

		if err := rows.Scan(&jobID, &tenantIDStr, &messageIDStr, &attempt, &maxAttempts, &errors); err != nil {
			rm.logger.Error("Failed to scan failed job",
				zap.Error(err),
			)
			continue
		}

		// Parse UUIDs
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			rm.logger.Error("Failed to parse tenant ID",
				zap.Error(err),
				zap.String("tenant_id", tenantIDStr),
			)
			continue
		}

		messageID, err := uuid.Parse(messageIDStr)
		if err != nil {
			rm.logger.Error("Failed to parse message ID",
				zap.Error(err),
				zap.String("message_id", messageIDStr),
			)
			continue
		}

		// Reschedule the job for immediate retry
		_, err = rm.db.Pool.Exec(ctx, `
			UPDATE river_job
			SET 
				state = 'available',
				scheduled_at = NOW(),
				updated_at = NOW()
			WHERE id = $1
		`, jobID)

		if err != nil {
			rm.logger.Error("Failed to reschedule failed job",
				zap.Error(err),
				zap.Int64("job_id", jobID),
				zap.String("tenant_id", tenantID.String()),
				zap.String("message_id", messageID.String()),
			)
			continue
		}

		recoveredCount++

		rm.logger.Info("Recovered failed job",
			zap.Int64("job_id", jobID),
			zap.String("tenant_id", tenantID.String()),
			zap.String("message_id", messageID.String()),
			zap.Int("attempt", attempt),
			zap.Int("max_attempts", maxAttempts),
		)
	}

	if err := rows.Err(); err != nil {
		return recoveredCount, fmt.Errorf("iterate failed jobs: %w", err)
	}

	rm.logger.Info("Failed jobs recovery completed",
		zap.Int("recovered_count", recoveredCount),
	)

	return recoveredCount, nil
}

// resetMessageAndCreateJob resets message status and creates new River job
func (rm *RecoveryManager) resetMessageAndCreateJob(ctx context.Context, tenantID, messageID uuid.UUID) error {
	// Start transaction
	tx, err := rm.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Reset message status to PENDING
	_, err = tx.Exec(ctx, `
		UPDATE messages
		SET 
			status = 'PENDING',
			updated_at = NOW()
		WHERE 
			tenant_id = $1 
			AND message_id = $2
			AND deleted_at IS NULL
	`, tenantID, messageID)

	if err != nil {
		return fmt.Errorf("reset message status: %w", err)
	}

	// Create new River job
	_, err = rm.client.InsertTx(ctx, tx, queue.SendSMSArgs{
		TenantID:  tenantID,
		MessageID: messageID,
	}, &river.InsertOpts{
		MaxAttempts: rm.config.MaxRecoveryAttempts,
		Priority:    1, // Normal priority
	})

	if err != nil {
		return fmt.Errorf("create river job: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// createRiverJob creates a River job for a message
func (rm *RecoveryManager) createRiverJob(ctx context.Context, tenantID, messageID uuid.UUID) error {
	// Start transaction
	tx, err := rm.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Create River job
	_, err = rm.client.InsertTx(ctx, tx, queue.SendSMSArgs{
		TenantID:  tenantID,
		MessageID: messageID,
	}, &river.InsertOpts{
		MaxAttempts: rm.config.MaxRecoveryAttempts,
		Priority:    1, // Normal priority
	})

	if err != nil {
		return fmt.Errorf("create river job: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// GetRecoveryStats returns recovery statistics
func (rm *RecoveryManager) GetRecoveryStats(ctx context.Context) (*RecoveryStats, error) {
	stats := &RecoveryStats{}

	// Count stale messages
	err := rm.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM messages
		WHERE 
			status = 'PROCESSING'
			AND updated_at < $1
			AND deleted_at IS NULL
	`, time.Now().Add(-rm.config.StaleMessageThreshold)).Scan(&stats.StaleMessages)

	if err != nil {
		return nil, fmt.Errorf("count stale messages: %w", err)
	}

	// Count pending messages without jobs
	err = rm.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM messages m
		LEFT JOIN river_job rj ON 
			rj.args->>'tenant_id' = m.tenant_id::text 
			AND rj.args->>'message_id' = m.message_id::text
		WHERE 
			m.status = 'PENDING'
			AND m.deleted_at IS NULL
			AND rj.id IS NULL
	`).Scan(&stats.PendingMessages)

	if err != nil {
		return nil, fmt.Errorf("count pending messages: %w", err)
	}

	// Count failed jobs
	err = rm.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM river_job
		WHERE 
			state = 'retryable'
			AND attempt < max_attempts
			AND scheduled_at < NOW()
	`).Scan(&stats.FailedJobs)

	if err != nil {
		return nil, fmt.Errorf("count failed jobs: %w", err)
	}

	stats.TotalRecoverable = stats.StaleMessages + stats.PendingMessages + stats.FailedJobs

	return stats, nil
}

// RecoveryStats represents recovery statistics
type RecoveryStats struct {
	// StaleMessages is the number of messages stuck in PROCESSING
	StaleMessages int

	// PendingMessages is the number of PENDING messages without jobs
	PendingMessages int

	// FailedJobs is the number of failed jobs that can be retried
	FailedJobs int

	// TotalRecoverable is the total number of recoverable items
	TotalRecoverable int
}
