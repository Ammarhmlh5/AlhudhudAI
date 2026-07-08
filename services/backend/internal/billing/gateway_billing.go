package billing

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

// GatewayBillingService handles subscription billing for dedicated gateways
// Implements Requirements 8.10, 8.11, 8.15
//
// نموذج العمل المالي:
// - الشركة المؤسسية تدفع لشركة الاتصالات مباشرة لتكلفة الرسائل
// - المنصة تتقاضى فقط:
//   1. رسوم الاشتراك (subscription_fee) - شهري أو سنوي
//   2. رسوم إيجار السيرفر (server_rental_fee) - شهري
// - لا يتم خصم أي تكلفة متعلقة بالرسائل
type GatewayBillingService struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewGatewayBillingService creates a new gateway billing service
func NewGatewayBillingService(pool *pgxpool.Pool, logger *zap.Logger) *GatewayBillingService {
	return &GatewayBillingService{
		pool:   pool,
		logger: logger,
	}
}

// GatewayBillingInfo contains billing information for a gateway
type GatewayBillingInfo struct {
	GatewayID          int64
	TenantID           uuid.UUID
	GatewayName        string
	GatewayCode        string
	SubscriptionType   string          // 'monthly' or 'yearly'
	SubscriptionFee    decimal.Decimal // رسوم الاشتراك
	ServerRentalFee    decimal.Decimal // رسوم إيجار السيرفر
	BillingCycle       string          // 'monthly' or 'yearly'
	NextBillingDate    time.Time
	TenantBalance      decimal.Decimal
	Status             string
}

// CheckDueGateways checks for gateways that need billing
// Implements Requirement 8.10, 8.11: Check for gateways with due billing dates
func (gbs *GatewayBillingService) CheckDueGateways(ctx context.Context) error {
	gbs.logger.Info("Starting daily check for gateway billing")

	// Query for gateways that need billing
	query := `
		SELECT 
			dg.id,
			dg.tenant_id,
			dg.gateway_name,
			dg.gateway_code,
			dg.subscription_type,
			dg.subscription_fee,
			dg.server_rental_fee,
			dg.billing_cycle,
			dg.next_billing_date,
			dg.status,
			t.balance
		FROM dedicated_gateways dg
		JOIN tenants t ON dg.tenant_id = t.id
		WHERE dg.status = 'active'
		  AND dg.next_billing_date <= CURRENT_DATE
		ORDER BY dg.next_billing_date ASC
	`

	rows, err := gbs.pool.Query(ctx, query)
	if err != nil {
		gbs.logger.Error("Failed to query due gateways", zap.Error(err))
		return fmt.Errorf("failed to query due gateways: %w", err)
	}
	defer rows.Close()

	var (
		successCount int
		failureCount int
	)

	for rows.Next() {
		var info GatewayBillingInfo

		if err := rows.Scan(
			&info.GatewayID,
			&info.TenantID,
			&info.GatewayName,
			&info.GatewayCode,
			&info.SubscriptionType,
			&info.SubscriptionFee,
			&info.ServerRentalFee,
			&info.BillingCycle,
			&info.NextBillingDate,
			&info.Status,
			&info.TenantBalance,
		); err != nil {
			gbs.logger.Error("Failed to scan gateway row", zap.Error(err))
			continue
		}

		gbs.logger.Info("Processing gateway billing",
			zap.Int64("gateway_id", info.GatewayID),
			zap.String("tenant_id", info.TenantID.String()),
			zap.String("gateway_code", info.GatewayCode),
		)

		// Bill the gateway
		if err := gbs.BillGatewaySubscription(ctx, &info); err != nil {
			gbs.logger.Error("Failed to bill gateway",
				zap.Int64("gateway_id", info.GatewayID),
				zap.Error(err),
			)
			failureCount++
		} else {
			successCount++
		}
	}

	if err := rows.Err(); err != nil {
		gbs.logger.Error("Error iterating gateway rows", zap.Error(err))
		return fmt.Errorf("error iterating gateway rows: %w", err)
	}

	gbs.logger.Info("Completed daily check for gateway billing",
		zap.Int("success_count", successCount),
		zap.Int("failure_count", failureCount),
	)

	return nil
}

// BillGatewaySubscription bills a dedicated gateway for subscription and server rental
// Implements Requirements 8.10, 8.11, 8.15
//
// ملاحظة مهمة: لا يتم خصم أي تكلفة متعلقة بالرسائل
// الشركة تدفع مباشرة لشركة الاتصالات لتكلفة الرسائل
func (gbs *GatewayBillingService) BillGatewaySubscription(ctx context.Context, info *GatewayBillingInfo) error {
	gbs.logger.Info("Billing gateway subscription",
		zap.Int64("gateway_id", info.GatewayID),
		zap.String("tenant_id", info.TenantID.String()),
		zap.String("subscription_type", info.SubscriptionType),
	)

	// Calculate total billing amount
	// Requirement 8.10: subscription_fee (monthly or yearly)
	// Requirement 8.11: server_rental_fee (always monthly)
	totalAmount := info.SubscriptionFee.Add(info.ServerRentalFee)

	gbs.logger.Info("Calculated billing amount",
		zap.String("subscription_fee", info.SubscriptionFee.String()),
		zap.String("server_rental_fee", info.ServerRentalFee.String()),
		zap.String("total_amount", totalAmount.String()),
	)

	// Check if balance is sufficient
	if info.TenantBalance.LessThan(totalAmount) {
		gbs.logger.Warn("Insufficient balance for gateway billing",
			zap.Int64("gateway_id", info.GatewayID),
			zap.String("required", totalAmount.String()),
			zap.String("available", info.TenantBalance.String()),
		)
		return fmt.Errorf("insufficient balance: required %s, available %s", 
			totalAmount.String(), info.TenantBalance.String())
	}

	// Begin transaction
	tx, err := gbs.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Set tenant context for RLS
	if _, err := tx.Exec(ctx, "SET LOCAL app.current_tenant = $1", info.TenantID.String()); err != nil {
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
	`, totalAmount, info.TenantID).Scan(&newBalance)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("insufficient balance or tenant not found")
		}
		return fmt.Errorf("failed to deduct balance: %w", err)
	}

	// Calculate next billing date
	now := time.Now()
	var nextBillingDate time.Time
	if info.BillingCycle == "yearly" {
		nextBillingDate = now.AddDate(1, 0, 0) // Add 1 year
	} else {
		nextBillingDate = now.AddDate(0, 1, 0) // Add 1 month
	}

	// Update gateway next billing date
	_, err = tx.Exec(ctx, `
		UPDATE dedicated_gateways
		SET next_billing_date = $1,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND tenant_id = $3
	`, nextBillingDate, info.GatewayID, info.TenantID)
	if err != nil {
		return fmt.Errorf("failed to update gateway billing date: %w", err)
	}

	// Record transaction in accounting_transactions
	// Requirement 8.15: Record only subscription and rental fees, NOT message costs
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
		info.TenantID,
		"GATEWAY_SUBSCRIPTION",
		totalAmount,
		"USD",
		info.TenantBalance,
		newBalance,
		fmt.Sprintf("Gateway subscription billing for %s (Subscription: %s, Server Rental: %s)",
			info.GatewayCode,
			info.SubscriptionFee.String(),
			info.ServerRentalFee.String(),
		),
		"dedicated_gateway",
		info.GatewayID,
		now,
	)
	if err != nil {
		return fmt.Errorf("failed to record transaction: %w", err)
	}

	// Generate invoice
	invoiceNumber := fmt.Sprintf("GW-INV-%s-%d", time.Now().Format("20060102"), info.GatewayID)
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
		info.TenantID,
		invoiceNumber,
		"gateway_subscription",
		totalAmount,
		"USD",
		decimal.Zero,
		totalAmount,
		"paid",
		now,
		now,
		fmt.Sprintf("Gateway subscription billing for %s - Subscription Fee: %s USD, Server Rental: %s USD (Billing Cycle: %s)",
			info.GatewayName,
			info.SubscriptionFee.String(),
			info.ServerRentalFee.String(),
			info.BillingCycle,
		),
		"dedicated_gateway",
		info.GatewayID,
	)
	if err != nil {
		return fmt.Errorf("failed to create invoice: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	gbs.logger.Info("Gateway subscription billed successfully",
		zap.Int64("gateway_id", info.GatewayID),
		zap.String("tenant_id", info.TenantID.String()),
		zap.String("invoice_number", invoiceNumber),
		zap.String("total_amount", totalAmount.String()),
		zap.String("new_balance", newBalance.String()),
		zap.Time("next_billing_date", nextBillingDate),
	)

	// TODO: Send invoice to enterprise company (Task 14 - Invoice Generator integration)

	return nil
}

// GetGatewayBillingInfo retrieves billing information for a specific gateway
func (gbs *GatewayBillingService) GetGatewayBillingInfo(ctx context.Context, tenantID uuid.UUID, gatewayID int64) (*GatewayBillingInfo, error) {
	query := `
		SELECT 
			dg.id,
			dg.tenant_id,
			dg.gateway_name,
			dg.gateway_code,
			dg.subscription_type,
			dg.subscription_fee,
			dg.server_rental_fee,
			dg.billing_cycle,
			dg.next_billing_date,
			dg.status,
			t.balance
		FROM dedicated_gateways dg
		JOIN tenants t ON dg.tenant_id = t.id
		WHERE dg.id = $1 AND dg.tenant_id = $2
	`

	var info GatewayBillingInfo
	err := gbs.pool.QueryRow(ctx, query, gatewayID, tenantID).Scan(
		&info.GatewayID,
		&info.TenantID,
		&info.GatewayName,
		&info.GatewayCode,
		&info.SubscriptionType,
		&info.SubscriptionFee,
		&info.ServerRentalFee,
		&info.BillingCycle,
		&info.NextBillingDate,
		&info.Status,
		&info.TenantBalance,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("gateway not found")
		}
		return nil, fmt.Errorf("failed to get gateway billing info: %w", err)
	}

	return &info, nil
}

// GetUpcomingBillings retrieves all gateways with upcoming billing dates
func (gbs *GatewayBillingService) GetUpcomingBillings(ctx context.Context, daysAhead int) ([]GatewayBillingInfo, error) {
	query := `
		SELECT 
			dg.id,
			dg.tenant_id,
			dg.gateway_name,
			dg.gateway_code,
			dg.subscription_type,
			dg.subscription_fee,
			dg.server_rental_fee,
			dg.billing_cycle,
			dg.next_billing_date,
			dg.status,
			t.balance
		FROM dedicated_gateways dg
		JOIN tenants t ON dg.tenant_id = t.id
		WHERE dg.status = 'active'
		  AND dg.next_billing_date <= CURRENT_DATE + $1
		ORDER BY dg.next_billing_date ASC
	`

	rows, err := gbs.pool.Query(ctx, query, daysAhead)
	if err != nil {
		return nil, fmt.Errorf("failed to query upcoming billings: %w", err)
	}
	defer rows.Close()

	var billings []GatewayBillingInfo
	for rows.Next() {
		var info GatewayBillingInfo
		if err := rows.Scan(
			&info.GatewayID,
			&info.TenantID,
			&info.GatewayName,
			&info.GatewayCode,
			&info.SubscriptionType,
			&info.SubscriptionFee,
			&info.ServerRentalFee,
			&info.BillingCycle,
			&info.NextBillingDate,
			&info.Status,
			&info.TenantBalance,
		); err != nil {
			gbs.logger.Error("Failed to scan billing row", zap.Error(err))
			continue
		}
		billings = append(billings, info)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating billing rows: %w", err)
	}

	return billings, nil
}
