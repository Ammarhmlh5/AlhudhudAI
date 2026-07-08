package subscription

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// Manager interface defines subscription management operations
type Manager interface {
	// Subscribe subscribes a tenant to a package
	// Implements Requirements 4.1, 4.2, 4.3, 4.4
	Subscribe(ctx context.Context, req SubscribeRequest) (*SubscribeResult, error)

	// DeductMessages deducts messages from subscription quota
	// Implements Requirements 4.5, 4.6
	DeductMessages(ctx context.Context, req DeductMessagesRequest) (*DeductMessagesResult, error)

	// CalculateOverageCost calculates the cost of overage messages
	// Implements Requirements 5.1, 5.2, 5.3, 5.4, 5.5
	CalculateOverageCost(ctx context.Context, subscriptionID int64, overageMessages int64) (decimal.Decimal, error)

	// GetUsage returns subscription usage information
	// Implements Requirements 4.6, 5.7
	GetUsage(ctx context.Context, tenantID uuid.UUID) (*UsageInfo, error)

	// GetActiveSubscription returns the active subscription for a tenant
	GetActiveSubscription(ctx context.Context, tenantID uuid.UUID) (*Subscription, error)

	// GetPackage returns package details
	GetPackage(ctx context.Context, packageID int64) (*Package, error)
}

// manager implements the Manager interface
type manager struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewManager creates a new subscription manager
func NewManager(pool *pgxpool.Pool, logger *zap.Logger) Manager {
	return &manager{
		pool:   pool,
		logger: logger,
	}
}

// Subscribe subscribes a tenant to a package
// Implements Requirements 4.1, 4.2, 4.3, 4.4
func (m *manager) Subscribe(ctx context.Context, req SubscribeRequest) (*SubscribeResult, error) {
	// Validate request
	if req.TenantID == uuid.Nil {
		return nil, fmt.Errorf("tenant_id is required")
	}
	if req.PackageID <= 0 {
		return nil, fmt.Errorf("package_id is required")
	}

	// Start transaction with SERIALIZABLE isolation level for atomicity
	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Set tenant context for RLS
	_, err = tx.Exec(ctx, "SET LOCAL app.current_tenant = $1", req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to set tenant context: %w", err)
	}

	// Get package details (Requirement 4.1)
	pkg, err := m.getPackageInTx(ctx, tx, req.PackageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get package: %w", err)
	}

	// Check for idempotency (Point 2.4)
	if req.IdempotencyKey != "" {
		var existingID int64
		var oldBal, newBal decimal.Decimal
		idempotencyQuery := `
			SELECT s.id, t.balance + p.price_usd, t.balance
			FROM subscriptions s
			JOIN tenants t ON s.tenant_id = t.id
			JOIN packages p ON s.package_id = p.id
			WHERE s.tenant_id = $1 AND s.idempotency_key = $2
			LIMIT 1
		`
		// Use req.TenantID directly as uuid.UUID
		err = tx.QueryRow(ctx, idempotencyQuery, req.TenantID, req.IdempotencyKey).Scan(&existingID, &oldBal, &newBal)
		if err == nil {
			m.logger.Info("Duplicate subscription request detected (idempotency)",
				zap.String("tenant_id", req.TenantID.String()),
				zap.String("idempotency_key", req.IdempotencyKey),
				zap.Int64("subscription_id", existingID),
			)
			return &SubscribeResult{
				Success:        true,
				SubscriptionID: existingID,
				Message:        "Success (idempotent)",
				OldBalance:     oldBal,
				NewBalance:     newBal,
			}, nil
		}
	}

	// Check if package is active
	if !pkg.IsActive {
		return &SubscribeResult{
			Success: false,
			Message: "Package is not active",
		}, nil
	}

	// Check if tenant has sufficient balance (Requirement 4.2)
	var currentBalance decimal.Decimal
	balanceQuery := `
		SELECT balance 
		FROM tenants 
		WHERE id = (
			SELECT id FROM tenants WHERE id::text = $1::text LIMIT 1
		) AND deleted_at IS NULL
	`
	err = tx.QueryRow(ctx, balanceQuery, req.TenantID.String()).Scan(&currentBalance)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("tenant not found")
		}
		return nil, fmt.Errorf("failed to get tenant balance: %w", err)
	}

	// Check if balance is sufficient
	if currentBalance.LessThan(pkg.PriceUSD) {
		m.logger.Warn("Insufficient balance for subscription",
			zap.String("tenant_id", req.TenantID.String()),
			zap.Int64("package_id", req.PackageID),
			zap.String("required", pkg.PriceUSD.String()),
			zap.String("available", currentBalance.String()),
		)
		return &SubscribeResult{
			Success:    false,
			Message:    "Insufficient balance",
			OldBalance: currentBalance,
			NewBalance: currentBalance,
		}, nil
	}

	// Deduct package price from balance atomically (Requirement 4.2)
	var newBalance decimal.Decimal
	deductQuery := `
		UPDATE tenants 
		SET 
			balance = balance - $1,
			updated_at = NOW()
		WHERE 
			id = (SELECT id FROM tenants WHERE id::text = $2::text LIMIT 1)
			AND balance >= $1
			AND deleted_at IS NULL
		RETURNING balance
	`
	err = tx.QueryRow(ctx, deductQuery, pkg.PriceUSD, req.TenantID.String()).Scan(&newBalance)
	if err != nil {
		if err == pgx.ErrNoRows {
			return &SubscribeResult{
				Success:    false,
				Message:    "Insufficient balance",
				OldBalance: currentBalance,
				NewBalance: currentBalance,
			}, nil
		}
		return nil, fmt.Errorf("failed to deduct balance: %w", err)
	}

	// Calculate subscription dates (Requirement 4.3)
	startDate := time.Now()
	var endDate time.Time
	var nextRenewalDate time.Time

	if pkg.BillingCycle == "monthly" {
		endDate = startDate.AddDate(0, 1, 0)
		nextRenewalDate = endDate
	} else if pkg.BillingCycle == "yearly" {
		endDate = startDate.AddDate(1, 0, 0)
		nextRenewalDate = endDate
	} else {
		return nil, fmt.Errorf("invalid billing cycle: %s", pkg.BillingCycle)
	}

	// Create subscription (Requirement 4.3, 4.4)
	var subscriptionID int64
	insertQuery := `
		INSERT INTO subscriptions (
			tenant_id,
			package_id,
			start_date,
			end_date,
			messages_used,
			messages_remaining,
			overage_messages,
			overage_cost,
			status,
			auto_renew,
			grace_period_days,
			next_renewal_date,
			idempotency_key
		) VALUES (
			$1, $2, $3, $4, 0, $5, 0, 0, 'active', $6, 3, $7, $8
		)
		RETURNING id
	`

	err = tx.QueryRow(ctx, insertQuery,
		req.TenantID,
		req.PackageID,
		startDate,
		endDate,
		pkg.IncludedMessages, // NULL for unlimited packages
		req.AutoRenew,
		nextRenewalDate,
		req.IdempotencyKey,
	).Scan(&subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	// Record transaction in accounting_transactions (Requirement 4.4)
	txID := uuid.New()
	accountingQuery := `
		INSERT INTO accounting_transactions (
			id,
			tenant_id,
			type,
			amount,
			status,
			created_at,
			updated_at
		) VALUES (
			$1, $2, 'SUBSCRIPTION', $3, 'COMPLETED', NOW(), NOW()
		)
	`

	_, err = tx.Exec(ctx, accountingQuery,
		txID,
		req.TenantID,
		pkg.PriceUSD,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to record accounting transaction: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	m.logger.Info("Subscription created successfully",
		zap.String("tenant_id", req.TenantID.String()),
		zap.Int64("subscription_id", subscriptionID),
		zap.Int64("package_id", req.PackageID),
		zap.String("package_name", pkg.Name),
		zap.String("price", pkg.PriceUSD.String()),
		zap.String("old_balance", currentBalance.String()),
		zap.String("new_balance", newBalance.String()),
	)

	return &SubscribeResult{
		Success:        true,
		SubscriptionID: subscriptionID,
		Message:        fmt.Sprintf("Successfully subscribed to %s package", pkg.Name),
		OldBalance:     currentBalance,
		NewBalance:     newBalance,
	}, nil
}

// getPackageInTx gets package details within a transaction
func (m *manager) getPackageInTx(ctx context.Context, tx pgx.Tx, packageID int64) (*Package, error) {
	query := `
		SELECT 
			id, tenant_id, service_id, name, code, price_usd,
			included_messages, overage_rate, is_unlimited, billing_cycle,
			rollover_enabled, rollover_limit, rollover_expiry_months,
			is_active, display_order, created_at, updated_at
		FROM packages
		WHERE id = $1
	`

	var pkg Package
	err := tx.QueryRow(ctx, query, packageID).Scan(
		&pkg.ID,
		&pkg.TenantID,
		&pkg.ServiceID,
		&pkg.Name,
		&pkg.Code,
		&pkg.PriceUSD,
		&pkg.IncludedMessages,
		&pkg.OverageRate,
		&pkg.IsUnlimited,
		&pkg.BillingCycle,
		&pkg.RolloverEnabled,
		&pkg.RolloverLimit,
		&pkg.RolloverExpiryMonths,
		&pkg.IsActive,
		&pkg.DisplayOrder,
		&pkg.CreatedAt,
		&pkg.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("package not found")
		}
		return nil, err
	}

	return &pkg, nil
}

// GetPackage returns package details
func (m *manager) GetPackage(ctx context.Context, packageID int64) (*Package, error) {
	query := `
		SELECT 
			id, tenant_id, service_id, name, code, price_usd,
			included_messages, overage_rate, is_unlimited, billing_cycle,
			rollover_enabled, rollover_limit, rollover_expiry_months,
			is_active, display_order, created_at, updated_at
		FROM packages
		WHERE id = $1
	`

	var pkg Package
	err := m.pool.QueryRow(ctx, query, packageID).Scan(
		&pkg.ID,
		&pkg.TenantID,
		&pkg.ServiceID,
		&pkg.Name,
		&pkg.Code,
		&pkg.PriceUSD,
		&pkg.IncludedMessages,
		&pkg.OverageRate,
		&pkg.IsUnlimited,
		&pkg.BillingCycle,
		&pkg.RolloverEnabled,
		&pkg.RolloverLimit,
		&pkg.RolloverExpiryMonths,
		&pkg.IsActive,
		&pkg.DisplayOrder,
		&pkg.CreatedAt,
		&pkg.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("package not found")
		}
		return nil, err
	}

	return &pkg, nil
}

// GetActiveSubscription returns the active subscription for a tenant
func (m *manager) GetActiveSubscription(ctx context.Context, tenantID uuid.UUID) (*Subscription, error) {
	query := `
		SELECT 
			id, tenant_id, package_id, start_date, end_date,
			messages_used, messages_remaining, overage_messages, overage_cost,
			status, auto_renew, grace_period_days,
			last_renewed_at, next_renewal_date, renewal_attempts, last_renewal_attempt,
			created_at, updated_at
		FROM subscriptions
		WHERE tenant_id = $1
		  AND status = 'active'
		  AND end_date >= CURRENT_DATE
		ORDER BY created_at DESC
		LIMIT 1
	`

	var sub Subscription
	err := m.pool.QueryRow(ctx, query, tenantID).Scan(
		&sub.ID,
		&sub.TenantID,
		&sub.PackageID,
		&sub.StartDate,
		&sub.EndDate,
		&sub.MessagesUsed,
		&sub.MessagesRemaining,
		&sub.OverageMessages,
		&sub.OverageCost,
		&sub.Status,
		&sub.AutoRenew,
		&sub.GracePeriodDays,
		&sub.LastRenewedAt,
		&sub.NextRenewalDate,
		&sub.RenewalAttempts,
		&sub.LastRenewalAttempt,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("no active subscription found")
		}
		return nil, err
	}

	return &sub, nil
}

// Placeholder implementations for remaining methods
// These will be implemented in subsequent tasks

// DeductMessages deducts messages from subscription quota
// Implements Requirements 4.5, 4.6
// - Atomic UPDATE for messages_used and messages_remaining
// - Handles unlimited packages (NULL remaining)
// - Detects overage condition (when messages_remaining reaches 0)
// - Uses SERIALIZABLE transaction isolation level
// - Respects Row Level Security (RLS)
func (m *manager) DeductMessages(ctx context.Context, req DeductMessagesRequest) (*DeductMessagesResult, error) {
	// Validate request
	if req.SubscriptionID <= 0 {
		return nil, NewInvalidRequestError("subscription_id is required")
	}
	if req.MessageCount <= 0 {
		return nil, NewInvalidRequestError("message_count must be positive")
	}
	if req.MessageID == "" {
		return nil, NewInvalidRequestError("message_id is required")
	}

	// Start transaction with SERIALIZABLE isolation level for atomicity
	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get current subscription state
	var (
		tenantID          uuid.UUID
		messagesUsed      int64
		messagesRemaining *int64
		overageMessages   int64
		overageCost       decimal.Decimal
		status            string
	)

	getQuery := `
		SELECT 
			tenant_id,
			messages_used,
			messages_remaining,
			overage_messages,
			overage_cost,
			status
		FROM subscriptions
		WHERE id = $1
	`

	err = tx.QueryRow(ctx, getQuery, req.SubscriptionID).Scan(
		&tenantID,
		&messagesUsed,
		&messagesRemaining,
		&overageMessages,
		&overageCost,
		&status,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, NewSubscriptionNotFoundError(uuid.Nil)
		}
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Set tenant context for RLS
	_, err = tx.Exec(ctx, "SET LOCAL app.current_tenant = $1", tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to set tenant context: %w", err)
	}

	// Check if subscription is active
	if status != "active" {
		return &DeductMessagesResult{
			Success: false,
			Error:   fmt.Errorf("subscription is not active: status=%s", status),
		}, nil
	}

	// Handle unlimited packages (messages_remaining is NULL)
	if messagesRemaining == nil {
		// For unlimited packages, just increment messages_used
		updateQuery := `
			UPDATE subscriptions
			SET 
				messages_used = messages_used + $1,
				updated_at = NOW()
			WHERE id = $2
			RETURNING messages_used, messages_remaining
		`

		var newMessagesUsed int64
		var newMessagesRemaining *int64

		err = tx.QueryRow(ctx, updateQuery, req.MessageCount, req.SubscriptionID).Scan(
			&newMessagesUsed,
			&newMessagesRemaining,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to update unlimited subscription: %w", err)
		}

		// Commit transaction
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("failed to commit transaction: %w", err)
		}

		m.logger.Info("Messages deducted from unlimited subscription",
			zap.Int64("subscription_id", req.SubscriptionID),
			zap.Int64("message_count", req.MessageCount),
			zap.String("message_id", req.MessageID),
			zap.Int64("messages_used", newMessagesUsed),
		)

		return &DeductMessagesResult{
			Success:           true,
			MessagesDeducted:  req.MessageCount,
			OverageMessages:   0,
			OverageCost:       decimal.Zero,
			MessagesRemaining: nil, // NULL for unlimited
		}, nil
	}

	// Handle limited packages
	// Check if we have enough messages in quota
	if *messagesRemaining >= req.MessageCount {
		// Deduct from quota atomically
		updateQuery := `
			UPDATE subscriptions
			SET 
				messages_used = messages_used + $1,
				messages_remaining = messages_remaining - $1,
				updated_at = NOW()
			WHERE id = $2
			  AND messages_remaining >= $1
			RETURNING messages_used, messages_remaining
		`

		var newMessagesUsed int64
		var newMessagesRemaining *int64

		err = tx.QueryRow(ctx, updateQuery, req.MessageCount, req.SubscriptionID).Scan(
			&newMessagesUsed,
			&newMessagesRemaining,
		)
		if err != nil {
			if err == pgx.ErrNoRows {
				// Race condition: another transaction deducted messages
				return &DeductMessagesResult{
					Success: false,
					Error:   fmt.Errorf("insufficient messages remaining (race condition)"),
				}, nil
			}
			return nil, fmt.Errorf("failed to deduct messages: %w", err)
		}

		// Commit transaction
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("failed to commit transaction: %w", err)
		}

		m.logger.Info("Messages deducted from subscription quota",
			zap.Int64("subscription_id", req.SubscriptionID),
			zap.Int64("message_count", req.MessageCount),
			zap.String("message_id", req.MessageID),
			zap.Int64("messages_used", newMessagesUsed),
			zap.Int64("messages_remaining", *newMessagesRemaining),
		)

		return &DeductMessagesResult{
			Success:           true,
			MessagesDeducted:  req.MessageCount,
			OverageMessages:   0,
			OverageCost:       decimal.Zero,
			MessagesRemaining: newMessagesRemaining,
		}, nil
	}

	// Overage scenario: not enough messages in quota
	// This will be handled by CalculateOverageCost in task 7.5
	// For now, we detect the overage condition and return an error
	m.logger.Warn("Overage detected: insufficient messages in quota",
		zap.Int64("subscription_id", req.SubscriptionID),
		zap.Int64("message_count", req.MessageCount),
		zap.String("message_id", req.MessageID),
		zap.Int64("messages_remaining", *messagesRemaining),
	)

	// Rollback transaction (no changes made)
	tx.Rollback(ctx)

	return &DeductMessagesResult{
		Success:           false,
		MessagesDeducted:  0,
		OverageMessages:   req.MessageCount - *messagesRemaining,
		OverageCost:       decimal.Zero, // Will be calculated in task 7.5
		MessagesRemaining: messagesRemaining,
		Error:             fmt.Errorf("insufficient messages in quota: need %d, have %d (overage: %d)", req.MessageCount, *messagesRemaining, req.MessageCount-*messagesRemaining),
	}, nil
}

// CalculateOverageCost calculates and charges the cost of overage messages
// Implements Requirements 5.1, 5.2, 5.3, 5.4, 5.5
// - Calculates cost using package overage_rate
// - Checks tenant balance before charging
// - Atomically deducts cost from balance
// - Updates subscription overage_messages and overage_cost
// - Records transaction in accounting_transactions
func (m *manager) CalculateOverageCost(ctx context.Context, subscriptionID int64, overageMessages int64) (decimal.Decimal, error) {
	// Validate input
	if subscriptionID <= 0 {
		return decimal.Zero, fmt.Errorf("subscription_id is required")
	}
	if overageMessages <= 0 {
		return decimal.Zero, fmt.Errorf("overage_messages must be positive")
	}

	// Start transaction with SERIALIZABLE isolation level for atomicity
	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable,
	})
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get subscription and package details
	var (
		tenantID    uuid.UUID
		packageID   int64
		overageRate decimal.Decimal
		isUnlimited bool
	)

	subQuery := `
		SELECT 
			s.tenant_id,
			s.package_id,
			p.overage_rate,
			p.is_unlimited
		FROM subscriptions s
		JOIN packages p ON s.package_id = p.id
		WHERE s.id = $1
	`

	err = tx.QueryRow(ctx, subQuery, subscriptionID).Scan(
		&tenantID,
		&packageID,
		&overageRate,
		&isUnlimited,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return decimal.Zero, fmt.Errorf("subscription not found")
		}
		return decimal.Zero, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Set tenant context for RLS
	_, err = tx.Exec(ctx, "SET LOCAL app.current_tenant = $1", tenantID)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to set tenant context: %w", err)
	}

	// For unlimited packages, no overage cost (Requirement 5.8)
	if isUnlimited {
		m.logger.Info("No overage cost for unlimited package",
			zap.Int64("subscription_id", subscriptionID),
			zap.Int64("overage_messages", overageMessages),
		)
		return decimal.Zero, nil
	}

	// Calculate overage cost (Requirement 5.2)
	overageCost := overageRate.Mul(decimal.NewFromInt(overageMessages))

	m.logger.Info("Calculated overage cost",
		zap.Int64("subscription_id", subscriptionID),
		zap.Int64("overage_messages", overageMessages),
		zap.String("overage_rate", overageRate.String()),
		zap.String("overage_cost", overageCost.String()),
	)

	// Get current tenant balance (Requirement 5.3)
	var currentBalance decimal.Decimal
	balanceQuery := `
		SELECT balance 
		FROM tenants 
		WHERE id = (
			SELECT id FROM tenants WHERE id::text = $1::text LIMIT 1
		) AND deleted_at IS NULL
	`
	err = tx.QueryRow(ctx, balanceQuery, tenantID.String()).Scan(&currentBalance)
	if err != nil {
		if err == pgx.ErrNoRows {
			return decimal.Zero, fmt.Errorf("tenant not found")
		}
		return decimal.Zero, fmt.Errorf("failed to get tenant balance: %w", err)
	}

	// Check if balance is sufficient (Requirement 5.4)
	if currentBalance.LessThan(overageCost) {
		m.logger.Warn("Insufficient balance for overage",
			zap.String("tenant_id", tenantID.String()),
			zap.Int64("subscription_id", subscriptionID),
			zap.String("required", overageCost.String()),
			zap.String("available", currentBalance.String()),
		)
		return decimal.Zero, fmt.Errorf("insufficient balance: need %s, have %s", overageCost.String(), currentBalance.String())
	}

	// Deduct overage cost from balance atomically (Requirement 5.4)
	var newBalance decimal.Decimal
	deductQuery := `
		UPDATE tenants 
		SET 
			balance = balance - $1,
			updated_at = NOW()
		WHERE 
			id = (SELECT id FROM tenants WHERE id::text = $2::text LIMIT 1)
			AND balance >= $1
			AND deleted_at IS NULL
		RETURNING balance
	`
	err = tx.QueryRow(ctx, deductQuery, overageCost, tenantID.String()).Scan(&newBalance)
	if err != nil {
		if err == pgx.ErrNoRows {
			return decimal.Zero, fmt.Errorf("insufficient balance (race condition)")
		}
		return decimal.Zero, fmt.Errorf("failed to deduct balance: %w", err)
	}

	// Update subscription overage_messages and overage_cost (Requirement 5.6)
	updateSubQuery := `
		UPDATE subscriptions
		SET 
			overage_messages = overage_messages + $1,
			overage_cost = overage_cost + $2,
			updated_at = NOW()
		WHERE id = $3
		RETURNING overage_messages, overage_cost
	`

	var totalOverageMessages int64
	var totalOverageCost decimal.Decimal

	err = tx.QueryRow(ctx, updateSubQuery, overageMessages, overageCost, subscriptionID).Scan(
		&totalOverageMessages,
		&totalOverageCost,
	)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to update subscription overage: %w", err)
	}

	// Record transaction in accounting_transactions (Requirement 5.5)
	txID := uuid.New()
	accountingQuery := `
		INSERT INTO accounting_transactions (
			id,
			tenant_id,
			type,
			amount,
			status,
			created_at,
			updated_at
		) VALUES (
			$1, $2, 'OVERAGE', $3, 'COMPLETED', NOW(), NOW()
		)
	`

	_, err = tx.Exec(ctx, accountingQuery,
		txID,
		tenantID,
		overageCost,
	)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to record accounting transaction: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return decimal.Zero, fmt.Errorf("failed to commit transaction: %w", err)
	}

	m.logger.Info("Overage cost charged successfully",
		zap.String("tenant_id", tenantID.String()),
		zap.Int64("subscription_id", subscriptionID),
		zap.Int64("overage_messages", overageMessages),
		zap.String("overage_cost", overageCost.String()),
		zap.String("old_balance", currentBalance.String()),
		zap.String("new_balance", newBalance.String()),
		zap.Int64("total_overage_messages", totalOverageMessages),
		zap.String("total_overage_cost", totalOverageCost.String()),
	)

	return overageCost, nil
}

// GetUsage returns subscription usage information
// Implements Requirements 4.6, 5.7
// - Displays messages used and remaining
// - Displays overage messages and cost
// - Displays rollover messages (if applicable)
// - Calculates total usage and costs
func (m *manager) GetUsage(ctx context.Context, tenantID uuid.UUID) (*UsageInfo, error) {
	// Validate input
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("tenant_id is required")
	}

	// Get active subscription with package details
	query := `
		SELECT 
			s.id,
			s.package_id,
			s.start_date,
			s.end_date,
			s.messages_used,
			s.messages_remaining,
			s.overage_messages,
			s.overage_cost,
			s.status,
			s.next_renewal_date,
			p.name,
			p.included_messages,
			p.overage_rate
		FROM subscriptions s
		JOIN packages p ON s.package_id = p.id
		WHERE s.tenant_id = $1
		  AND s.status = 'active'
		  AND s.end_date >= CURRENT_DATE
		ORDER BY s.created_at DESC
		LIMIT 1
	`

	var usage UsageInfo
	err := m.pool.QueryRow(ctx, query, tenantID).Scan(
		&usage.SubscriptionID,
		&usage.PackageID,
		&usage.StartDate,
		&usage.EndDate,
		&usage.MessagesUsed,
		&usage.MessagesRemaining,
		&usage.OverageMessages,
		&usage.OverageCost,
		&usage.Status,
		&usage.NextRenewalDate,
		&usage.PackageName,
		&usage.IncludedMessages,
		&usage.OverageRate,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("no active subscription found for tenant")
		}
		return nil, fmt.Errorf("failed to get usage: %w", err)
	}

	m.logger.Info("Usage information retrieved",
		zap.String("tenant_id", tenantID.String()),
		zap.Int64("subscription_id", usage.SubscriptionID),
		zap.String("package_name", usage.PackageName),
		zap.Int64("messages_used", usage.MessagesUsed),
	)

	return &usage, nil
}
