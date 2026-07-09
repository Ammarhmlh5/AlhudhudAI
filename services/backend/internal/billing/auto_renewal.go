package billing

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// AutoRenewalService handles automatic subscription renewals
type AutoRenewalService struct {
	pool   *pgxpool.Pool
	client *river.Client[pgx.Tx]
	logger *zap.Logger
}

// NewAutoRenewalService creates a new auto renewal service
func NewAutoRenewalService(pool *pgxpool.Pool, client *river.Client[pgx.Tx], logger *zap.Logger) *AutoRenewalService {
	return &AutoRenewalService{
		pool:   pool,
		client: client,
		logger: logger,
	}
}

// CheckExpiredSubscriptions checks for expired subscriptions and attempts renewal
// Implements Requirement 14.1: Daily check for expired subscriptions
func (s *AutoRenewalService) CheckExpiredSubscriptions(ctx context.Context) error {
	s.logger.Info("Starting daily check for expired subscriptions")

	// Query for subscriptions that need renewal
	query := `
		SELECT 
			s.id,
			s.tenant_id,
			s.package_id,
			s.end_date,
			s.auto_renew,
			s.status,
			s.grace_period_days,
			s.renewal_attempts,
			s.last_renewal_attempt,
			p.price_usd,
			t.balance
		FROM subscriptions s
		JOIN packages p ON s.package_id = p.id
		JOIN tenants t ON s.tenant_id = t.id
		WHERE s.auto_renew = true
		  AND s.status IN ('active', 'grace_period')
		  AND s.end_date <= CURRENT_DATE
		ORDER BY s.end_date ASC
	`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		s.logger.Error("Failed to query expired subscriptions", zap.Error(err))
		return fmt.Errorf("failed to query expired subscriptions: %w", err)
	}
	defer rows.Close()

	var (
		successCount int
		failureCount int
		graceCount   int
		suspendCount int
	)

	for rows.Next() {
		var (
			subscriptionID   int64
			tenantID         uuid.UUID
			packageID        int64
			endDate          time.Time
			autoRenew        bool
			status           string
			gracePeriodDays  int
			renewalAttempts  int
			lastRenewalAttempt *time.Time
			priceUSD         decimal.Decimal
			balance          decimal.Decimal
		)

		if err := rows.Scan(
			&subscriptionID,
			&tenantID,
			&packageID,
			&endDate,
			&autoRenew,
			&status,
			&gracePeriodDays,
			&renewalAttempts,
			&lastRenewalAttempt,
			&priceUSD,
			&balance,
		); err != nil {
			s.logger.Error("Failed to scan subscription row", zap.Error(err))
			continue
		}

		s.logger.Info("Processing expired subscription",
			zap.Int64("subscription_id", subscriptionID),
			zap.String("tenant_id", tenantID.String()),
			zap.String("status", status),
		)

		// Check if subscription is in grace period
		if status == "grace_period" {
			// Check if grace period has expired
			gracePeriodEnd := endDate.AddDate(0, 0, gracePeriodDays)
			if time.Now().After(gracePeriodEnd) {
				// Suspend subscription
				if err := s.SuspendSubscription(ctx, subscriptionID, tenantID); err != nil {
					s.logger.Error("Failed to suspend subscription",
						zap.Int64("subscription_id", subscriptionID),
						zap.Error(err),
					)
					failureCount++
				} else {
					suspendCount++
				}
				continue
			}

			// Still in grace period, attempt renewal
			if err := s.RenewSubscription(ctx, subscriptionID, tenantID, packageID, priceUSD, balance); err != nil {
				s.logger.Warn("Renewal failed during grace period",
					zap.Int64("subscription_id", subscriptionID),
					zap.Error(err),
				)
				failureCount++
			} else {
				successCount++
			}
			continue
		}

		// Active subscription that just expired, attempt renewal
		if err := s.RenewSubscription(ctx, subscriptionID, tenantID, packageID, priceUSD, balance); err != nil {
			// Renewal failed, apply grace period
			if err := s.ApplyGracePeriod(ctx, subscriptionID, tenantID, gracePeriodDays); err != nil {
				s.logger.Error("Failed to apply grace period",
					zap.Int64("subscription_id", subscriptionID),
					zap.Error(err),
				)
				failureCount++
			} else {
				graceCount++
			}
		} else {
			successCount++
		}
	}

	if err := rows.Err(); err != nil {
		s.logger.Error("Error iterating subscription rows", zap.Error(err))
		return fmt.Errorf("error iterating subscription rows: %w", err)
	}

	s.logger.Info("Completed daily check for expired subscriptions",
		zap.Int("success_count", successCount),
		zap.Int("failure_count", failureCount),
		zap.Int("grace_count", graceCount),
		zap.Int("suspend_count", suspendCount),
	)

	return nil
}

// RenewSubscription attempts to renew a subscription
// Implements Requirements 14.2, 14.3, 14.7: Verify balance, deduct atomically, record transaction
func (s *AutoRenewalService) RenewSubscription(
	ctx context.Context,
	subscriptionID int64,
	tenantID uuid.UUID,
	packageID int64,
	priceUSD decimal.Decimal,
	currentBalance decimal.Decimal,
) error {
	s.logger.Info("Attempting subscription renewal",
		zap.Int64("subscription_id", subscriptionID),
		zap.String("tenant_id", tenantID.String()),
	)

	// Requirement 14.2: Verify balance is sufficient
	if currentBalance.LessThan(priceUSD) {
		s.logger.Warn("Insufficient balance for renewal",
			zap.Int64("subscription_id", subscriptionID),
			zap.String("required", priceUSD.String()),
			zap.String("available", currentBalance.String()),
		)
		return fmt.Errorf("insufficient balance: required %s, available %s", priceUSD.String(), currentBalance.String())
	}

	// Requirement 14.3: Deduct package price and renew subscription atomically
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Set tenant context for RLS
	if _, err := tx.Exec(ctx, "SET LOCAL app.current_tenant = $1", tenantID.String()); err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}

	// Deduct balance atomically
	var newBalance decimal.Decimal
	err = tx.QueryRow(ctx, `
		UPDATE tenants
		SET balance = balance - $1,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND balance >= $1
		RETURNING balance
	`, priceUSD, tenantID).Scan(&newBalance)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("insufficient balance or tenant not found")
		}
		return fmt.Errorf("failed to deduct balance: %w", err)
	}

	// Get package details for message quota
	var includedMessages *int64
	err = tx.QueryRow(ctx, `
		SELECT included_messages
		FROM packages
		WHERE id = $1 AND tenant_id = $2
	`, packageID, tenantID).Scan(&includedMessages)
	if err != nil {
		return fmt.Errorf("failed to get package details: %w", err)
	}

	// Update subscription: reset dates, counters, and status
	now := time.Now()
	nextMonth := now.AddDate(0, 1, 0)
	
	_, err = tx.Exec(ctx, `
		UPDATE subscriptions
		SET start_date = $1,
		    end_date = $2,
		    next_renewal_date = $2,
		    messages_used = 0,
		    messages_remaining = $3,
		    overage_messages = 0,
		    overage_cost = 0,
		    status = 'active',
		    renewal_attempts = 0,
		    last_renewed_at = $1,
		    last_renewal_attempt = $1,
		    updated_at = $1
		WHERE id = $4 AND tenant_id = $5
	`, now, nextMonth, includedMessages, subscriptionID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	// Requirement 14.7: Record billing operation in transactions
	_, err = tx.Exec(ctx, `
		INSERT INTO accounting_transactions (
			tenant_id,
			transaction_type,
			amount,
			currency,
			balance_before,
			balance_after,
			description,
			reference_type,
			reference_id,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`,
		tenantID,
		"SUBSCRIPTION_RENEWAL",
		priceUSD,
		"USD",
		currentBalance,
		newBalance,
		fmt.Sprintf("Automatic renewal of subscription #%d", subscriptionID),
		"subscription",
		subscriptionID,
		now,
	)
	if err != nil {
		return fmt.Errorf("failed to record transaction: %w", err)
	}

	// Generate invoice
	invoiceNumber := fmt.Sprintf("INV-%s-%d", time.Now().Format("20060102"), subscriptionID)
	_, err = tx.Exec(ctx, `
		INSERT INTO billing_invoices (
			tenant_id,
			invoice_number,
			invoice_type,
			amount,
			currency,
			tax_amount,
			total_amount,
			status,
			issue_date,
			due_date,
			description,
			reference_type,
			reference_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`,
		tenantID,
		invoiceNumber,
		"subscription_renewal",
		priceUSD,
		"USD",
		decimal.Zero,
		priceUSD,
		"paid",
		now,
		now,
		fmt.Sprintf("Subscription renewal for package #%d", packageID),
		"subscription",
		subscriptionID,
	)
	if err != nil {
		return fmt.Errorf("failed to create invoice: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Info("Subscription renewed successfully",
		zap.Int64("subscription_id", subscriptionID),
		zap.String("tenant_id", tenantID.String()),
		zap.String("new_balance", newBalance.String()),
	)

	// TODO: Send success notification (Task 13)

	return nil
}

// ApplyGracePeriod applies grace period when renewal fails
// Implements Requirements 14.4, 14.5: Send notification and grant grace period
func (s *AutoRenewalService) ApplyGracePeriod(
	ctx context.Context,
	subscriptionID int64,
	tenantID uuid.UUID,
	gracePeriodDays int,
) error {
	s.logger.Info("Applying grace period",
		zap.Int64("subscription_id", subscriptionID),
		zap.String("tenant_id", tenantID.String()),
		zap.Int("grace_period_days", gracePeriodDays),
	)

	// Default grace period is 3 days (Requirement 14.5)
	if gracePeriodDays == 0 {
		gracePeriodDays = 3
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Set tenant context for RLS
	if _, err := tx.Exec(ctx, "SET LOCAL app.current_tenant = $1", tenantID.String()); err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}

	// Update subscription status to grace_period
	now := time.Now()
	_, err = tx.Exec(ctx, `
		UPDATE subscriptions
		SET status = 'grace_period',
		    renewal_attempts = renewal_attempts + 1,
		    last_renewal_attempt = $1,
		    updated_at = $1
		WHERE id = $2 AND tenant_id = $3
	`, now, subscriptionID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to update subscription status: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Info("Grace period applied successfully",
		zap.Int64("subscription_id", subscriptionID),
		zap.String("tenant_id", tenantID.String()),
	)

	// Requirement 14.4: Send notification
	// TODO: Implement notification sending (Task 13)

	return nil
}

// SuspendSubscription suspends a subscription after grace period expires
// Implements Requirements 14.6, 14.11: Suspend subscription and prevent new messages
func (s *AutoRenewalService) SuspendSubscription(
	ctx context.Context,
	subscriptionID int64,
	tenantID uuid.UUID,
) error {
	s.logger.Info("Suspending subscription",
		zap.Int64("subscription_id", subscriptionID),
		zap.String("tenant_id", tenantID.String()),
	)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Set tenant context for RLS
	if _, err := tx.Exec(ctx, "SET LOCAL app.current_tenant = $1", tenantID.String()); err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}

	// Update subscription status to suspended
	now := time.Now()
	_, err = tx.Exec(ctx, `
		UPDATE subscriptions
		SET status = 'suspended',
		    updated_at = $1
		WHERE id = $2 AND tenant_id = $3
	`, now, subscriptionID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to suspend subscription: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Info("Subscription suspended successfully",
		zap.Int64("subscription_id", subscriptionID),
		zap.String("tenant_id", tenantID.String()),
	)

	// TODO: Send suspension notification (Task 13)

	return nil
}

// AutoReactivate attempts to reactivate suspended subscriptions when balance is topped up
// Implements Requirement 14.12: Auto-reactivate when sufficient balance added
func (s *AutoRenewalService) AutoReactivate(ctx context.Context, tenantID uuid.UUID) error {
	s.logger.Info("Checking for suspended subscriptions to reactivate",
		zap.String("tenant_id", tenantID.String()),
	)

	// Query for suspended subscriptions with sufficient balance
	query := `
		SELECT 
			s.id,
			s.package_id,
			p.price_usd,
			t.balance
		FROM subscriptions s
		JOIN packages p ON s.package_id = p.id
		JOIN tenants t ON s.tenant_id = t.id
		WHERE s.tenant_id = $1
		  AND s.status = 'suspended'
		  AND s.auto_renew = true
		  AND t.balance >= p.price_usd
		ORDER BY s.end_date ASC
		LIMIT 1
	`

	var (
		subscriptionID int64
		packageID      int64
		priceUSD       decimal.Decimal
		balance        decimal.Decimal
	)

	err := s.pool.QueryRow(ctx, query, tenantID).Scan(
		&subscriptionID,
		&packageID,
		&priceUSD,
		&balance,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			// No suspended subscriptions to reactivate
			return nil
		}
		return fmt.Errorf("failed to query suspended subscriptions: %w", err)
	}

	// Attempt renewal
	if err := s.RenewSubscription(ctx, subscriptionID, tenantID, packageID, priceUSD, balance); err != nil {
		s.logger.Error("Failed to reactivate subscription",
			zap.Int64("subscription_id", subscriptionID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to reactivate subscription: %w", err)
	}

	s.logger.Info("Subscription reactivated successfully",
		zap.Int64("subscription_id", subscriptionID),
		zap.String("tenant_id", tenantID.String()),
	)

	// TODO: Send reactivation notification (Task 13)

	return nil
}
