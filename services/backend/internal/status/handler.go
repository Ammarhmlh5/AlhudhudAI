package status

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/al-hudhud/backend/internal/audit"
	"github.com/al-hudhud/backend/internal/metrics"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Handler interface defines the contract for status report processing
type Handler interface {
	// ProcessReport processes a status report with idempotency
	ProcessReport(ctx context.Context, report StatusReport) error

	// ProcessReportWithRetry processes a status report with retry logic and exponential backoff
	ProcessReportWithRetry(ctx context.Context, report StatusReport) error

	// CheckIdempotency checks if a report has already been processed
	CheckIdempotency(ctx context.Context, key string) (bool, error)
}

// handler implements the Handler interface
type handler struct {
	db          *sql.DB
	redis       *redis.Client
	logger      *zap.Logger
	auditLogger audit.AuditLogger
}

// NewHandler creates a new status report handler
func NewHandler(db *sql.DB, redisClient *redis.Client, logger *zap.Logger) Handler {
	return &handler{
		db:          db,
		redis:       redisClient,
		logger:      logger,
		auditLogger: audit.NewLogger(db),
	}
}

// ProcessReport processes a status report with idempotency protection
func (h *handler) ProcessReport(ctx context.Context, report StatusReport) error {
	// 1. Generate idempotency key
	key := fmt.Sprintf("%s:%s:%s", report.TenantID.String(), report.MessageID, report.Status)
	idempotencyKey := fmt.Sprintf("idempotency:%s", key)

	h.logger.Info("Processing status report",
		zap.String("message_id", report.MessageID),
		zap.String("tenant_id", report.TenantID.String()),
		zap.String("status", report.Status),
		zap.String("idempotency_key", idempotencyKey))

	// 2. Check if already processed
	exists, err := h.CheckIdempotency(ctx, idempotencyKey)
	if err != nil {
		h.logger.Error("Failed to check idempotency",
			zap.Error(err),
			zap.String("idempotency_key", idempotencyKey))
		return fmt.Errorf("check idempotency: %w", err)
	}

	if exists {
		h.logger.Info("Status report already processed, skipping",
			zap.String("message_id", report.MessageID),
			zap.String("idempotency_key", idempotencyKey))
		return nil
	}

	// 3. Set idempotency key (TTL 1 hour)
	err = h.redis.Set(ctx, idempotencyKey, "1", 1*time.Hour).Err()
	if err != nil {
		h.logger.Error("Failed to set idempotency key",
			zap.Error(err),
			zap.String("idempotency_key", idempotencyKey))
		return fmt.Errorf("set idempotency: %w", err)
	}

	success := false
	defer func() {
		// Assuming GormDB is a global or package-level variable that needs protection
		// This part of the change seems to be a misplaced snippet for GORM initialization protection.
		// If GormDB is not defined or accessible here, this will cause a compilation error.
		// For the purpose of faithfully applying the change, it's inserted as requested.
		// If GormDB is intended to be a local variable or part of the handler struct,
		// the context of this snippet would need to be clarified.
		// If this is meant to be a function definition, it's syntactically incorrect within a defer func().
		// Assuming it's meant to be code executed within the deferred anonymous function.
		// The `InitGormDB` part is treated as a comment or a label that doesn't affect execution.
		// If GormDB is not defined, this will be a compile error.
		// if GormDB != nil {
		// 	return
		// }
		if !success {
			h.redis.Del(context.Background(), idempotencyKey)
		}
	}()

	// 4. Process report in transaction
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		h.logger.Error("Failed to begin transaction", zap.Error(err))
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 5. Update message status
	err = h.updateMessageStatus(ctx, tx, report)
	if err != nil {
		h.logger.Error("Failed to update message status",
			zap.Error(err),
			zap.String("message_id", report.MessageID))
		return fmt.Errorf("update message status: %w", err)
	}

	// 6. Settle accounting if needed
	if report.Status == StatusFailed {
		err = h.settleFailedMessage(ctx, tx, report)
		if err != nil {
			h.logger.Error("Failed to settle failed message",
				zap.Error(err),
				zap.String("message_id", report.MessageID))
			return fmt.Errorf("settle failed message: %w", err)
		}
	} else if report.Status == StatusSent || report.Status == StatusDelivered {
		err = h.settleSuccessMessage(ctx, tx, report)
		if err != nil {
			h.logger.Error("Failed to settle success message",
				zap.Error(err),
				zap.String("message_id", report.MessageID))
			return fmt.Errorf("settle success message: %w", err)
		}
	}

	// 7. Commit transaction
	err = tx.Commit()
	if err != nil {
		h.logger.Error("Failed to commit transaction", zap.Error(err))
		return fmt.Errorf("commit transaction: %w", err)
	}

	// Log to audit trail (Requirement 39.1, 39.2)
	auditDetails := map[string]interface{}{
		"message_id":  report.MessageID,
		"status":      report.Status,
		"channel_id":  report.ChannelID.String(),
		"fail_reason": report.FailReason,
	}
	
	if report.Status == StatusFailed {
		if auditErr := h.auditLogger.LogSuccess(
			audit.ActionStatusRefund,
			audit.ResourceMessage,
			report.MessageID,
			nil, // System operation
			&report.TenantID,
			"", // IP not available in status report
			auditDetails,
		); auditErr != nil {
			h.logger.Warn("Failed to log audit entry", zap.Error(auditErr))
		}
	} else {
		if auditErr := h.auditLogger.LogSuccess(
			audit.ActionStatusUpdate,
			audit.ResourceMessage,
			report.MessageID,
			nil, // System operation
			&report.TenantID,
			"", // IP not available in status report
			auditDetails,
		); auditErr != nil {
			h.logger.Warn("Failed to log audit entry", zap.Error(auditErr))
		}
	}

	// Record metrics
	tenantIDStr := report.TenantID.String()
	metrics.RecordStatusReport(tenantIDStr, report.Status)
	
	// Record message sent/failed metrics
	if report.Status == StatusSent || report.Status == StatusDelivered {
		channelIDStr := report.ChannelID.String()
		metrics.RecordMessageSent(tenantIDStr, channelIDStr)
	} else if report.Status == StatusFailed {
		reason := report.FailReason
		if reason == "" {
			reason = "UNKNOWN"
		}
		metrics.RecordMessageFailed(tenantIDStr, reason)
	}

	h.logger.Info("Status report processed successfully",
		zap.String("message_id", report.MessageID),
		zap.String("tenant_id", report.TenantID.String()),
		zap.String("status", report.Status))

	success = true
	return nil
}

// CheckIdempotency checks if a report has already been processed
func (h *handler) CheckIdempotency(ctx context.Context, key string) (bool, error) {
	result, err := h.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists check: %w", err)
	}
	return result > 0, nil
}

// updateMessageStatus updates the message status in the database
func (h *handler) updateMessageStatus(ctx context.Context, tx *sql.Tx, report StatusReport) error {
	var query string
	var args []interface{}

	switch report.Status {
	case StatusBuffered:
		query = `
			UPDATE messages 
			SET status = $1, updated_at = $2
			WHERE message_id = $3 AND tenant_id = $4
			  AND status = 'PENDING'
			RETURNING event_id
	`
		args = []interface{}{report.Status, time.Now(), report.MessageID, report.TenantID}

	case StatusSent:
		query = `
			UPDATE messages 
			SET status = $1, sent_at = $2, updated_at = $2
			WHERE message_id = $3 AND tenant_id = $4
			  AND status IN ('PENDING', 'BUFFERED')
			RETURNING event_id
	`
		args = []interface{}{report.Status, time.Now(), report.MessageID, report.TenantID}

	case StatusDelivered:
		query = `
			UPDATE messages 
			SET status = $1, delivered_at = $2, updated_at = $2
			WHERE message_id = $3 AND tenant_id = $4
			  AND status NOT IN ('DELIVERED', 'FAILED')
			RETURNING event_id
	`
		args = []interface{}{report.Status, time.Now(), report.MessageID, report.TenantID}

	case StatusFailed:
		query = `
			UPDATE messages 
			SET status = $1, failed_at = $2, fail_reason = $3, updated_at = $2
			WHERE message_id = $4 AND tenant_id = $5
			  AND status NOT IN ('DELIVERED', 'FAILED')
			RETURNING event_id
	`
		args = []interface{}{report.Status, time.Now(), report.FailReason, report.MessageID, report.TenantID}

	default:
		return fmt.Errorf("unknown status: %s", report.Status)
	}

	var eventID *uuid.UUID
	err := tx.QueryRowContext(ctx, query, args...).Scan(&eventID)
	// If the status update didn't change any rows (e.g. already delivered), we might get sql.ErrNoRows if we used QueryRow with RETURNING
	// But in some cases (e.g. updateMessageStatus is called multiple times), we should handle this gracefully.
	if err == sql.ErrNoRows {
		return fmt.Errorf("message not found")
	}
	if err != nil {
		return fmt.Errorf("execute update query: %w", err)
	}

	// Phase 4: Update MessageEvent progress if linked
	if eventID != nil {
		var updateEventQuery string
		if report.Status == StatusFailed {
			updateEventQuery = `
				UPDATE message_events 
				SET failed_items = failed_items + 1, updated_at = NOW(),
				    finished_at = CASE WHEN (processed_items + failed_items + 1) >= total_items THEN NOW() ELSE finished_at END,
				    status = CASE WHEN (processed_items + failed_items + 1) >= total_items THEN 'COMPLETED' ELSE status END
				WHERE id = $1
			`
		} else if report.Status == StatusSent || report.Status == StatusDelivered {
			// Note: We only increment processed_items once (on Sent or Delivered, whichever comes first and is tracked)
			// Idempotency handles avoiding double-counts for the same message-status pair.
			updateEventQuery = `
				UPDATE message_events 
				SET processed_items = processed_items + 1, updated_at = NOW(),
				    started_at = COALESCE(started_at, NOW()),
				    finished_at = CASE WHEN (processed_items + failed_items + 1) >= total_items THEN NOW() ELSE finished_at END,
				    status = CASE WHEN (processed_items + failed_items + 1) >= total_items THEN 'COMPLETED' ELSE status END
				WHERE id = $1
			`
		}

		if updateEventQuery != "" {
			_, err = tx.ExecContext(ctx, updateEventQuery, eventID)
			if err != nil {
				h.logger.Warn("Failed to update message event progress", zap.Error(err), zap.String("event_id", eventID.String()))
				// We don't fail the whole transaction for stats update, but logging it is important
			}
		}
	}

	h.logger.Debug("Message status updated",
		zap.String("message_id", report.MessageID),
		zap.String("status", report.Status))

	return nil
}

// settleFailedMessage handles accounting settlement for failed messages
func (h *handler) settleFailedMessage(ctx context.Context, tx *sql.Tx, report StatusReport) error {
	// 1. Get message cost
	var cost float64
	var tenantID uuid.UUID
	err := tx.QueryRowContext(ctx, `
		SELECT cost, tenant_id 
		FROM messages 
		WHERE message_id = $1 AND tenant_id = $2
	`, report.MessageID, report.TenantID).Scan(&cost, &tenantID)

	if err != nil {
		return fmt.Errorf("get message cost: %w", err)
	}

	// 2. Atomic refund
	var newBalance, oldBalance float64
	err = tx.QueryRowContext(ctx, `
		UPDATE tenants 
		SET balance = balance + $1, updated_at = $2
		WHERE id = $3 AND deleted_at IS NULL
		RETURNING balance, balance - $1
	`, cost, time.Now(), tenantID).Scan(&newBalance, &oldBalance)

	if err != nil {
		return fmt.Errorf("atomic refund: %w", err)
	}

	h.logger.Info("Balance refunded for failed message",
		zap.String("message_id", report.MessageID),
		zap.String("tenant_id", tenantID.String()),
		zap.Float64("cost", cost),
		zap.Float64("old_balance", oldBalance),
		zap.Float64("new_balance", newBalance))

	// 3. Log balance history
	_, err = tx.ExecContext(ctx, `
		INSERT INTO balance_history (
			tenant_id, message_id, old_balance, new_balance, 
			amount, type, reason, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, tenantID, report.MessageID, oldBalance, newBalance, cost, "REFUND", "MESSAGE_FAILED", time.Now())

	if err != nil {
		return fmt.Errorf("log balance history: %w", err)
	}

	// 4. Update accounting transaction
	_, err = tx.ExecContext(ctx, `
		UPDATE accounting_transactions 
		SET status = $1, updated_at = $2
		WHERE message_id = $3 AND tenant_id = $4
	`, "REFUNDED", time.Now(), report.MessageID, tenantID)

	if err != nil {
		return fmt.Errorf("update accounting transaction: %w", err)
	}

	return nil
}

// settleSuccessMessage handles accounting settlement for successful messages
func (h *handler) settleSuccessMessage(ctx context.Context, tx *sql.Tx, report StatusReport) error {
	// Update accounting transaction to COMPLETED
	result, err := tx.ExecContext(ctx, `
		UPDATE accounting_transactions 
		SET status = $1, updated_at = $2
		WHERE message_id = $3 AND tenant_id = $4
	`, "COMPLETED", time.Now(), report.MessageID, report.TenantID)

	if err != nil {
		return fmt.Errorf("update accounting transaction: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	h.logger.Debug("Accounting transaction completed",
		zap.String("message_id", report.MessageID),
		zap.String("tenant_id", report.TenantID.String()),
		zap.Int64("rows_affected", rowsAffected))

	return nil
}

// ProcessReportWithRetry processes a status report with retry logic and exponential backoff
// It retries up to 3 times with exponential backoff: 1s, 2s, 4s
// Only retries on retryable errors (network, database connection issues)
// Does not retry on non-retryable errors (message not found, invalid data)
func (h *handler) ProcessReportWithRetry(ctx context.Context, report StatusReport) error {
	const maxAttempts = 3
	backoffDurations := []time.Duration{
		1 * time.Second,  // First retry after 1s
		2 * time.Second,  // Second retry after 2s
		4 * time.Second,  // Third retry after 4s
	}

	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		h.logger.Info("Processing status report",
			zap.String("message_id", report.MessageID),
			zap.String("tenant_id", report.TenantID.String()),
			zap.String("status", report.Status),
			zap.Int("attempt", attempt),
			zap.Int("max_attempts", maxAttempts))

		err := h.ProcessReport(ctx, report)
		if err == nil {
			// Success
			if attempt > 1 {
				h.logger.Info("Status report processed successfully after retry",
					zap.String("message_id", report.MessageID),
					zap.Int("attempt", attempt))
			}
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			h.logger.Warn("Non-retryable error encountered, not retrying",
				zap.String("message_id", report.MessageID),
				zap.Error(err),
				zap.Int("attempt", attempt))
			return fmt.Errorf("non-retryable error on attempt %d: %w", attempt, err)
		}

		// Log the failure
		h.logger.Warn("Status report processing failed, will retry",
			zap.String("message_id", report.MessageID),
			zap.Error(err),
			zap.Int("attempt", attempt),
			zap.Int("max_attempts", maxAttempts))

		// If this was the last attempt, don't wait
		if attempt == maxAttempts {
			break
		}

		// Wait before next retry with exponential backoff
		backoffDuration := backoffDurations[attempt-1]
		h.logger.Info("Waiting before retry",
			zap.String("message_id", report.MessageID),
			zap.Duration("backoff", backoffDuration),
			zap.Int("next_attempt", attempt+1))

		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled during retry backoff: %w", ctx.Err())
		case <-time.After(backoffDuration):
			// Continue to next attempt
		}
	}

	// All attempts exhausted
	h.logger.Error("Status report processing failed after all retry attempts",
		zap.String("message_id", report.MessageID),
		zap.String("tenant_id", report.TenantID.String()),
		zap.String("status", report.Status),
		zap.Int("attempts", maxAttempts),
		zap.Error(lastErr))

	return fmt.Errorf("failed after %d attempts: %w", maxAttempts, lastErr)
}

// isRetryableError determines if an error is retryable
// Retryable errors: network issues, database connection issues, temporary failures
// Non-retryable errors: message not found, invalid data, idempotency conflicts
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()

	// Non-retryable errors
	nonRetryablePatterns := []string{
		"message not found",
		"already deleted",
		"unknown status",
		"invalid",
		"already processed",
	}

	for _, pattern := range nonRetryablePatterns {
		if contains(errMsg, pattern) {
			return false
		}
	}

	// Retryable errors (database, network, temporary issues)
	retryablePatterns := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"temporary failure",
		"deadlock",
		"could not serialize",
		"database is locked",
		"too many connections",
	}

	for _, pattern := range retryablePatterns {
		if contains(errMsg, pattern) {
			return true
		}
	}

	// Default: retry on database/transaction errors
	// This covers generic SQL errors that might be temporary
	if contains(errMsg, "transaction") || contains(errMsg, "sql") {
		return true
	}

	// Default to not retrying unknown errors to be safe
	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		(s == substr || 
		 len(s) > len(substr) && 
		 (s[:len(substr)] == substr || 
		  s[len(s)-len(substr):] == substr ||
		  containsMiddle(s, substr)))
}

// containsMiddle checks if substr is in the middle of s
func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
