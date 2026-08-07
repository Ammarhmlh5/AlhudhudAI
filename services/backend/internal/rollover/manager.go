package rollover

import (
	"context"
	"fmt"
	"time"

	"github.com/al-hudhud/backend/internal/notifications"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// Manager interface defines rollover management operations
type Manager interface {
	// CalculateRemainingMessages calculates remaining messages at end of month
	// Implements Requirement 6.2
	CalculateRemainingMessages(ctx context.Context, subscriptionID int64) (int64, error)

	// RolloverMessages rolls over remaining messages to next month
	// Implements Requirements 6.3, 6.4, 6.5, 6.6, 6.7
	RolloverMessages(ctx context.Context, subscriptionID int64, remainingMessages int64) error

	// UseRolloverMessages uses rolled over messages (FIFO)
	// Implements Requirement 6.8
	UseRolloverMessages(ctx context.Context, subscriptionID int64, messageCount int64) (int64, error)

	// CleanupExpiredRollover deletes expired rollover messages
	// Implements Requirement 6.7
	CleanupExpiredRollover(ctx context.Context) error

	// GetRolloverMessages returns rollover messages for a subscription
	// Implements Requirement 6.9
	GetRolloverMessages(ctx context.Context, subscriptionID int64) ([]RolloverMessage, error)

	// ActivateRolloverOnPaymentSuccess activates pending rollover messages when payment succeeds
	// Implements Requirement 6.5
	ActivateRolloverOnPaymentSuccess(ctx context.Context, subscriptionID int64) error

	// ForfeitRolloverOnPaymentFailure forfeits pending rollover messages when payment fails
	// Implements Requirements 6.6, 6.15
	ForfeitRolloverOnPaymentFailure(ctx context.Context, subscriptionID int64, reason string) error
}

// manager implements the Manager interface
type manager struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewManager creates a new rollover manager
func NewManager(pool *pgxpool.Pool, logger *zap.Logger) Manager {
	return &manager{
		pool:   pool,
		logger: logger,
	}
}

// CalculateRemainingMessages calculates remaining messages at end of month
// Implements Requirement 6.2
// - Checks if rollover is enabled for the package
// - Calculates messages_remaining from subscription
// - Returns 0 if rollover is disabled
func (m *manager) CalculateRemainingMessages(ctx context.Context, subscriptionID int64) (int64, error) {
	// Validate input
	if subscriptionID <= 0 {
		return 0, fmt.Errorf("subscription_id is required")
	}

	// Get subscription and package details
	query := `
		SELECT 
			s.messages_remaining,
			p.rollover_enabled,
			p.is_unlimited
		FROM subscriptions s
		JOIN packages p ON s.package_id = p.id
		WHERE s.id = $1
	`

	var (
		messagesRemaining *int64
		rolloverEnabled   bool
		isUnlimited       bool
	)

	err := m.pool.QueryRow(ctx, query, subscriptionID).Scan(
		&messagesRemaining,
		&rolloverEnabled,
		&isUnlimited,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Unlimited packages don't have rollover
	if isUnlimited {
		m.logger.Info("Unlimited package, no rollover",
			zap.Int64("subscription_id", subscriptionID),
		)
		return 0, nil
	}

	// If rollover is disabled, return 0
	if !rolloverEnabled {
		m.logger.Info("Rollover disabled for package",
			zap.Int64("subscription_id", subscriptionID),
		)
		return 0, nil
	}

	// If no messages remaining, return 0
	if messagesRemaining == nil || *messagesRemaining <= 0 {
		m.logger.Info("No messages remaining to rollover",
			zap.Int64("subscription_id", subscriptionID),
		)
		return 0, nil
	}

	m.logger.Info("Calculated remaining messages for rollover",
		zap.Int64("subscription_id", subscriptionID),
		zap.Int64("remaining_messages", *messagesRemaining),
	)

	return *messagesRemaining, nil
}

// RolloverMessages rolls over remaining messages to next month
// Implements Requirements 6.3, 6.4, 6.5, 6.6, 6.7
// - Respects rollover_limit from package
// - Records in rollover_messages table
// - Sets expiry_date based on rollover_expiry_months
// - Only rolls over if rollover_enabled is true
func (m *manager) RolloverMessages(ctx context.Context, subscriptionID int64, remainingMessages int64) error {
	// Validate input
	if subscriptionID <= 0 {
		return fmt.Errorf("subscription_id is required")
	}

	if remainingMessages <= 0 {
		m.logger.Info("No messages to rollover",
			zap.Int64("subscription_id", subscriptionID),
		)
		return nil
	}

	// Start transaction
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get subscription and package details
	query := `
		SELECT 
			s.tenant_id,
			p.rollover_enabled,
			p.rollover_limit,
			p.rollover_expiry_months,
			p.is_unlimited
		FROM subscriptions s
		JOIN packages p ON s.package_id = p.id
		WHERE s.id = $1
	`

	var (
		tenantID             uuid.UUID
		rolloverEnabled      bool
		rolloverLimit        *int64
		rolloverExpiryMonths int
		isUnlimited          bool
	)

	err = tx.QueryRow(ctx, query, subscriptionID).Scan(
		&tenantID,
		&rolloverEnabled,
		&rolloverLimit,
		&rolloverExpiryMonths,
		&isUnlimited,
	)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// Check if rollover is enabled
	if !rolloverEnabled || isUnlimited {
		m.logger.Info("Rollover not enabled for this package",
			zap.Int64("subscription_id", subscriptionID),
			zap.Bool("rollover_enabled", rolloverEnabled),
			zap.Bool("is_unlimited", isUnlimited),
		)
		return nil
	}

	// Apply rollover limit (Requirement 6.5)
	messagesToRollover := remainingMessages
	if rolloverLimit != nil && *rolloverLimit > 0 {
		if messagesToRollover > *rolloverLimit {
			m.logger.Info("Applying rollover limit",
				zap.Int64("subscription_id", subscriptionID),
				zap.Int64("remaining_messages", remainingMessages),
				zap.Int64("rollover_limit", *rolloverLimit),
			)
			messagesToRollover = *rolloverLimit
		}
	}

	// Calculate expiry date (Requirement 6.6, 6.7)
	expiryDate := time.Now().AddDate(0, rolloverExpiryMonths, 0)

	// Get current month for source_month
	sourceMonth := time.Now().Truncate(24 * time.Hour)
	sourceMonth = time.Date(sourceMonth.Year(), sourceMonth.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Insert rollover record (Requirement 6.3)
	insertQuery := `
		INSERT INTO rollover_messages (
			subscription_id,
			tenant_id,
			messages_count,
			messages_remaining,
			source_month,
			expiry_date
		) VALUES ($1, $2, $3, $3, $4, $5)
		RETURNING id
	`

	var rolloverID int64
	err = tx.QueryRow(ctx, insertQuery,
		subscriptionID,
		tenantID,
		messagesToRollover,
		sourceMonth,
		expiryDate,
	).Scan(&rolloverID)
	if err != nil {
		return fmt.Errorf("failed to insert rollover record: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	m.logger.Info("Successfully rolled over messages",
		zap.Int64("subscription_id", subscriptionID),
		zap.Int64("rollover_id", rolloverID),
		zap.Int64("messages_rolled_over", messagesToRollover),
		zap.Time("expiry_date", expiryDate),
	)

	return nil
}

// UseRolloverMessages uses rolled over messages (FIFO)
// Implements Requirement 6.8
// - Uses oldest rollover messages first (FIFO)
// - Updates messages_used and messages_remaining in rollover_messages
// - Returns number of messages actually used
// - Deletes rollover records when fully consumed
func (m *manager) UseRolloverMessages(ctx context.Context, subscriptionID int64, messageCount int64) (int64, error) {
	// Validate input
	if subscriptionID <= 0 {
		return 0, fmt.Errorf("subscription_id is required")
	}

	if messageCount <= 0 {
		return 0, nil
	}

	// Start transaction
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get rollover messages ordered by created_at (FIFO)
	query := `
		SELECT 
			id,
			messages_count
		FROM rollover_messages
		WHERE subscription_id = $1
		  AND expiry_date >= CURRENT_DATE
		  AND messages_count > 0
		ORDER BY created_at ASC
		FOR UPDATE
	`

	rows, err := tx.Query(ctx, query, subscriptionID)
	if err != nil {
		return 0, fmt.Errorf("failed to get rollover messages: %w", err)
	}
	defer rows.Close()

	var rolloverRecords []struct {
		ID            int64
		MessagesCount int64
	}

	for rows.Next() {
		var record struct {
			ID            int64
			MessagesCount int64
		}
		if err := rows.Scan(&record.ID, &record.MessagesCount); err != nil {
			return 0, fmt.Errorf("failed to scan rollover record: %w", err)
		}
		rolloverRecords = append(rolloverRecords, record)
	}

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("error iterating rollover records: %w", err)
	}

	// No rollover messages available
	if len(rolloverRecords) == 0 {
		m.logger.Info("No rollover messages available",
			zap.Int64("subscription_id", subscriptionID),
		)
		return 0, nil
	}

	// Use messages from oldest to newest (FIFO)
	remainingToUse := messageCount
	totalUsed := int64(0)

	for _, record := range rolloverRecords {
		if remainingToUse <= 0 {
			break
		}

		// Determine how many messages to use from this record
		toUseFromRecord := remainingToUse
		if toUseFromRecord > record.MessagesCount {
			toUseFromRecord = record.MessagesCount
		}

		// Update or delete the rollover record
		newCount := record.MessagesCount - toUseFromRecord
		
		if newCount <= 0 {
			// Fully consumed - delete the record
			deleteQuery := `DELETE FROM rollover_messages WHERE id = $1`
			_, err := tx.Exec(ctx, deleteQuery, record.ID)
			if err != nil {
				return 0, fmt.Errorf("failed to delete rollover record: %w", err)
			}
			
			m.logger.Info("Deleted fully consumed rollover record",
				zap.Int64("rollover_id", record.ID),
				zap.Int64("messages_used", toUseFromRecord),
			)
		} else {
			// Partially consumed - update the count
			updateQuery := `
				UPDATE rollover_messages 
				SET messages_count = $1
				WHERE id = $2
			`
			_, err := tx.Exec(ctx, updateQuery, newCount, record.ID)
			if err != nil {
				return 0, fmt.Errorf("failed to update rollover record: %w", err)
			}
			
			m.logger.Info("Updated rollover record",
				zap.Int64("rollover_id", record.ID),
				zap.Int64("messages_used", toUseFromRecord),
				zap.Int64("messages_remaining", newCount),
			)
		}

		totalUsed += toUseFromRecord
		remainingToUse -= toUseFromRecord
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	m.logger.Info("Successfully used rollover messages",
		zap.Int64("subscription_id", subscriptionID),
		zap.Int64("requested", messageCount),
		zap.Int64("used", totalUsed),
	)

	return totalUsed, nil
}

// CleanupExpiredRollover deletes expired rollover messages
// Implements Requirement 6.7
// - Deletes rollover messages where expiry_date < current_date
// - Should be run as a daily cron job
// - Sends notification to customer before expiry (handled by notification service)
func (m *manager) CleanupExpiredRollover(ctx context.Context) error {
	// Delete expired rollover messages
	query := `
		DELETE FROM rollover_messages
		WHERE expiry_date < CURRENT_DATE
		RETURNING id, subscription_id, messages_count
	`

	rows, err := m.pool.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to delete expired rollover messages: %w", err)
	}
	defer rows.Close()

	deletedCount := 0
	totalMessages := int64(0)

	for rows.Next() {
		var (
			id             int64
			subscriptionID int64
			messagesCount  int64
		)
		
		if err := rows.Scan(&id, &subscriptionID, &messagesCount); err != nil {
			return fmt.Errorf("failed to scan deleted rollover: %w", err)
		}

		deletedCount++
		totalMessages += messagesCount

		m.logger.Info("Deleted expired rollover message",
			zap.Int64("rollover_id", id),
			zap.Int64("subscription_id", subscriptionID),
			zap.Int64("messages_count", messagesCount),
		)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating deleted rollovers: %w", err)
	}

	m.logger.Info("Cleanup expired rollover messages completed",
		zap.Int("deleted_records", deletedCount),
		zap.Int64("total_messages_expired", totalMessages),
	)

	return nil
}

// GetRolloverMessages returns rollover messages for a subscription
// Implements Requirement 6.9
func (m *manager) GetRolloverMessages(ctx context.Context, subscriptionID int64) ([]RolloverMessage, error) {
	// Validate input
	if subscriptionID <= 0 {
		return nil, fmt.Errorf("subscription_id is required")
	}

	query := `
		SELECT 
			id,
			subscription_id,
			tenant_id,
			messages_count,
			source_month,
			expiry_date,
			created_at
		FROM rollover_messages
		WHERE subscription_id = $1
		  AND expiry_date >= CURRENT_DATE
		ORDER BY created_at ASC
	`

	rows, err := m.pool.Query(ctx, query, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rollover messages: %w", err)
	}
	defer rows.Close()

	var messages []RolloverMessage
	for rows.Next() {
		var msg RolloverMessage
		err := rows.Scan(
			&msg.ID,
			&msg.SubscriptionID,
			&msg.TenantID,
			&msg.MessagesCount,
			&msg.SourceMonth,
			&msg.ExpiryDate,
			&msg.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rollover message: %w", err)
		}
		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rollover messages: %w", err)
	}

	m.logger.Info("Retrieved rollover messages",
		zap.Int64("subscription_id", subscriptionID),
		zap.Int("count", len(messages)),
	)

	return messages, nil
}

// ActivateRolloverOnPaymentSuccess activates pending rollover messages when payment succeeds
// Implements Requirement 6.5
// - Updates status from 'pending' to 'active'
// - Updates payment_status to 'success'
// - Records payment_verified_at timestamp
func (m *manager) ActivateRolloverOnPaymentSuccess(ctx context.Context, subscriptionID int64) error {
	m.logger.Info("Activating rollover messages on payment success",
		zap.Int64("subscription_id", subscriptionID))

	query := `
		UPDATE rollover_messages
		SET 
			status = 'active',
			payment_status = 'success',
			payment_verified_at = NOW(),
			updated_at = NOW()
		WHERE subscription_id = $1
		  AND status = 'pending'
		  AND payment_status = 'pending'
		  AND expiry_date > NOW()
	`

	result, err := m.pool.Exec(ctx, query, subscriptionID)
	if err != nil {
		m.logger.Error("Failed to activate rollover messages",
			zap.Int64("subscription_id", subscriptionID),
			zap.Error(err))
		return fmt.Errorf("failed to activate rollover messages: %w", err)
	}

	rowsAffected := result.RowsAffected()
	m.logger.Info("Rollover messages activated successfully",
		zap.Int64("subscription_id", subscriptionID),
		zap.Int64("messages_activated", rowsAffected))

	return nil
}

// ForfeitRolloverOnPaymentFailure forfeits pending rollover messages when payment fails
// Implements Requirements 6.6, 6.15
// - Updates status from 'pending' to 'forfeited'
// - Updates payment_status to 'failed'
// - Records forfeited_reason
// - Sends notification to customer
func (m *manager) ForfeitRolloverOnPaymentFailure(ctx context.Context, subscriptionID int64, reason string) error {
	m.logger.Info("Forfeiting rollover messages on payment failure",
		zap.Int64("subscription_id", subscriptionID),
		zap.String("reason", reason))

	// First, get the messages that will be forfeited for notification
	var totalMessages int64
	queryCount := `
		SELECT COALESCE(SUM(messages_count), 0)
		FROM rollover_messages
		WHERE subscription_id = $1
		  AND status = 'pending'
		  AND payment_status = 'pending'
	`
	
	err := m.pool.QueryRow(ctx, queryCount, subscriptionID).Scan(&totalMessages)
	if err != nil {
		m.logger.Error("Failed to count forfeited messages",
			zap.Int64("subscription_id", subscriptionID),
			zap.Error(err))
		return fmt.Errorf("failed to count forfeited messages: %w", err)
	}

	// Update the status to forfeited
	query := `
		UPDATE rollover_messages
		SET 
			status = 'forfeited',
			payment_status = 'failed',
			forfeited_reason = $2,
			forfeited_at = NOW(),
			updated_at = NOW()
		WHERE subscription_id = $1
		  AND status = 'pending'
		  AND payment_status = 'pending'
	`

	result, err := m.pool.Exec(ctx, query, subscriptionID, reason)
	if err != nil {
		m.logger.Error("Failed to forfeit rollover messages",
			zap.Int64("subscription_id", subscriptionID),
			zap.Error(err))
		return fmt.Errorf("failed to forfeit rollover messages: %w", err)
	}

	rowsAffected := result.RowsAffected()
	m.logger.Info("Rollover messages forfeited successfully",
		zap.Int64("subscription_id", subscriptionID),
		zap.Int64("records_forfeited", rowsAffected),
		zap.Int64("total_messages_lost", totalMessages),
		zap.String("reason", reason))

	// Notify the customer about forfeited rollover messages
	var tenantID uuid.UUID
	err = m.pool.QueryRow(ctx,
		"SELECT tenant_id FROM subscriptions WHERE id = $1", subscriptionID).Scan(&tenantID)
	if err == nil && totalMessages > 0 {
		notifications.SendTenantNotification(ctx, m.pool, tenantID, "alert",
			"ترحيل رسائل منتهي",
			fmt.Sprintf("تم إلغاء %d رسالة مُرحّلة لعدم اكتمال الدفع (%s).", totalMessages, reason),
			map[string]any{
				"subscription_id":    subscriptionID,
				"messages_forfeited": totalMessages,
				"reason":             reason,
			}, m.logger)
	}

	return nil
}
