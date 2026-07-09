package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/al-hudhud/backend/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// DeadLetterQueue handles messages that failed all retry attempts
// Implements Requirements 17.5, 17.6
type DeadLetterQueue struct {
	db     *database.DB
	logger *zap.Logger
}

// NewDeadLetterQueue creates a new dead letter queue
func NewDeadLetterQueue(db *database.DB, logger *zap.Logger) *DeadLetterQueue {
	return &DeadLetterQueue{
		db:     db,
		logger: logger,
	}
}

// MoveToDeadLetter moves a message to the dead letter queue
// Implements Requirement 17.5: نقل الرسائل الفاشلة نهائياً إلى Dead Letter Queue
func (dlq *DeadLetterQueue) MoveToDeadLetter(ctx context.Context, tenantID, messageID uuid.UUID, reason string, retryCount int) error {
	tx, err := dlq.db.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get message details
	var phoneNumber, content, originalStatus string
	var cost string
	err = tx.QueryRow(ctx, `
		SELECT phone_number, message_content, status, cost
		FROM messages
		WHERE tenant_id = $1 AND message_id = $2 AND deleted_at IS NULL
	`, tenantID, messageID).Scan(&phoneNumber, &content, &originalStatus, &cost)

	if err != nil {
		return fmt.Errorf("get message details: %w", err)
	}

	// Insert into dead letter queue
	// Implements Requirement 17.6: الاحتفاظ بالرسائل في Dead Letter Queue لمدة 7 أيام
	_, err = tx.Exec(ctx, `
		INSERT INTO dead_letter_queue (
			tenant_id, message_id, phone_number, content,
			original_status, fail_reason, retry_count, cost, moved_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
	`, tenantID, messageID, phoneNumber, content, originalStatus, reason, retryCount, cost)

	if err != nil {
		return fmt.Errorf("insert into dead letter queue: %w", err)
	}

	// Update message status to DEAD_LETTER
	_, err = tx.Exec(ctx, `
		UPDATE messages
		SET status = 'DEAD_LETTER', 
		    fail_reason = $3,
		    updated_at = NOW()
		WHERE tenant_id = $1 AND message_id = $2 AND deleted_at IS NULL
	`, tenantID, messageID, reason)

	if err != nil {
		return fmt.Errorf("update message status: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	dlq.logger.Info("Message moved to dead letter queue",
		zap.String("tenant_id", tenantID.String()),
		zap.String("message_id", messageID.String()),
		zap.String("reason", reason),
		zap.Int("retry_count", retryCount),
	)

	return nil
}

// GetDeadLetterMessages retrieves messages from dead letter queue
// Implements Requirement 17.6: توفير API لإعادة معالجة رسائل DLQ
func (dlq *DeadLetterQueue) GetDeadLetterMessages(ctx context.Context, tenantID uuid.UUID, limit int) ([]DeadLetterMessage, error) {
	rows, err := dlq.db.Pool.Query(ctx, `
		SELECT 
			id, tenant_id, message_id, phone_number, content,
			original_status, fail_reason, retry_count, cost, moved_at
		FROM dead_letter_queue
		WHERE tenant_id = $1
		ORDER BY moved_at DESC
		LIMIT $2
	`, tenantID, limit)

	if err != nil {
		return nil, fmt.Errorf("query dead letter queue: %w", err)
	}
	defer rows.Close()

	var messages []DeadLetterMessage
	for rows.Next() {
		var msg DeadLetterMessage
		err := rows.Scan(
			&msg.ID,
			&msg.TenantID,
			&msg.MessageID,
			&msg.PhoneNumber,
			&msg.Content,
			&msg.OriginalStatus,
			&msg.FailReason,
			&msg.RetryCount,
			&msg.Cost,
			&msg.MovedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan dead letter message: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// CleanupOldMessages removes messages older than retention period
// Implements Requirement 17.6: الاحتفاظ بالرسائل في Dead Letter Queue لمدة 7 أيام
func (dlq *DeadLetterQueue) CleanupOldMessages(ctx context.Context, retentionDays int) (int64, error) {
	if dlq.db == nil || dlq.db.Pool == nil {
		return 0, fmt.Errorf("database connection is nil")
	}
	result, err := dlq.db.Pool.Exec(ctx, `
		DELETE FROM dead_letter_queue
		WHERE moved_at < NOW() - INTERVAL '1 day' * $1
	`, retentionDays)

	if err != nil {
		return 0, fmt.Errorf("cleanup old messages: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected > 0 {
		dlq.logger.Info("Cleaned up old dead letter messages",
			zap.Int64("rows_affected", rowsAffected),
			zap.Int("retention_days", retentionDays),
		)
	}

	return rowsAffected, nil
}

// ReprocessMessage reprocesses a message from dead letter queue
// Implements Requirement 17.6: توفير API لإعادة معالجة رسائل DLQ
func (dlq *DeadLetterQueue) ReprocessMessage(ctx context.Context, tenantID uuid.UUID, dlqID int64) error {
	tx, err := dlq.db.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get DLQ message details
	var messageID uuid.UUID
	var phoneNumber, content, cost string
	err = tx.QueryRow(ctx, `
		SELECT message_id, phone_number, content, cost
		FROM dead_letter_queue
		WHERE id = $1 AND tenant_id = $2
	`, dlqID, tenantID).Scan(&messageID, &phoneNumber, &content, &cost)

	if err != nil {
		return fmt.Errorf("get DLQ message: %w", err)
	}

	// Update original message status back to PENDING for reprocessing
	_, err = tx.Exec(ctx, `
		UPDATE messages
		SET status = 'PENDING',
		    fail_reason = NULL,
		    updated_at = NOW()
		WHERE tenant_id = $1 AND message_id = $2 AND deleted_at IS NULL
	`, tenantID, messageID)

	if err != nil {
		return fmt.Errorf("update message status: %w", err)
	}

	// Remove from dead letter queue
	_, err = tx.Exec(ctx, `
		DELETE FROM dead_letter_queue
		WHERE id = $1 AND tenant_id = $2
	`, dlqID, tenantID)

	if err != nil {
		return fmt.Errorf("delete from DLQ: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	dlq.logger.Info("Message reprocessed from dead letter queue",
		zap.String("tenant_id", tenantID.String()),
		zap.String("message_id", messageID.String()),
		zap.Int64("dlq_id", dlqID),
	)

	return nil
}

// ReprocessAllMessages reprocesses all messages from dead letter queue for a tenant
// Implements Requirement 17.6: توفير API لإعادة معالجة رسائل DLQ
func (dlq *DeadLetterQueue) ReprocessAllMessages(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	tx, err := dlq.db.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get all DLQ messages for tenant
	rows, err := tx.Query(ctx, `
		SELECT message_id
		FROM dead_letter_queue
		WHERE tenant_id = $1
	`, tenantID)

	if err != nil {
		return 0, fmt.Errorf("query DLQ messages: %w", err)
	}
	defer rows.Close()

	var messageIDs []uuid.UUID
	for rows.Next() {
		var messageID uuid.UUID
		if err := rows.Scan(&messageID); err != nil {
			return 0, fmt.Errorf("scan message ID: %w", err)
		}
		messageIDs = append(messageIDs, messageID)
	}

	if len(messageIDs) == 0 {
		return 0, nil
	}

	// Update all messages status back to PENDING
	result, err := tx.Exec(ctx, `
		UPDATE messages
		SET status = 'PENDING',
		    fail_reason = NULL,
		    updated_at = NOW()
		WHERE tenant_id = $1 
		  AND message_id = ANY($2)
		  AND deleted_at IS NULL
	`, tenantID, messageIDs)

	if err != nil {
		return 0, fmt.Errorf("update messages status: %w", err)
	}

	// Remove all from dead letter queue
	_, err = tx.Exec(ctx, `
		DELETE FROM dead_letter_queue
		WHERE tenant_id = $1
	`, tenantID)

	if err != nil {
		return 0, fmt.Errorf("delete from DLQ: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit transaction: %w", err)
	}

	rowsAffected := result.RowsAffected()
	dlq.logger.Info("All messages reprocessed from dead letter queue",
		zap.String("tenant_id", tenantID.String()),
		zap.Int64("count", rowsAffected),
	)

	return rowsAffected, nil
}

// DeadLetterMessage represents a message in the dead letter queue
type DeadLetterMessage struct {
	ID             int64
	TenantID       uuid.UUID
	MessageID      uuid.UUID
	PhoneNumber    string
	Content        string
	OriginalStatus string
	FailReason     string
	RetryCount     int
	Cost           string
	MovedAt        time.Time
}
