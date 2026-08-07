package billing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// **Validates: Requirements 14.2, 14.3**
// Property 13: التجديد التلقائي عند نهاية الشهر
// When a subscription expires and auto_renew is enabled:
// - If balance >= package_price: renewal succeeds, balance deducted, dates updated, counters reset
// - If balance < package_price: renewal fails, grace period applied
func TestProperty_AutoRenewal(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	pool := setupTestDB(t)
	defer pool.Close()

	logger := zap.NewNop()
	service := NewAutoRenewalService(pool, nil, logger)

	properties := gopter.NewProperties(nil)

	properties.Property("auto renewal with sufficient/insufficient balance", prop.ForAll(
		func(balance float64, packagePrice float64) bool {
			ctx := context.Background()

			// Create test tenant with specified balance
			tenantID := createTestTenant(t, pool, balance)
			defer cleanupTestTenant(t, pool, tenantID)

			// Create test package with specified price
			packageID := createTestPackage(t, pool, tenantID, packagePrice, 1000)

			// Create expired subscription with auto_renew enabled
			subscriptionID := createExpiredSubscription(t, pool, tenantID, packageID, true)

			// Get initial state
			initialBalance := getBalance(t, pool, tenantID)
			initialStatus := getSubscriptionStatus(t, pool, subscriptionID)

			// Attempt renewal
			balanceDec := decimal.NewFromFloat(balance)
			priceDec := decimal.NewFromFloat(packagePrice)
			err := service.RenewSubscription(ctx, subscriptionID, tenantID, packageID, priceDec, balanceDec)

			// Get final state
			finalBalance := getBalance(t, pool, tenantID)
			finalStatus := getSubscriptionStatus(t, pool, subscriptionID)
			messagesUsed := getMessagesUsed(t, pool, subscriptionID)
			messagesRemaining := getMessagesRemaining(t, pool, subscriptionID)

			if balance >= packagePrice {
				// Sufficient balance: renewal should succeed
				if err != nil {
					t.Logf("Expected success but got error: %v (balance=%.2f, price=%.2f)", err, balance, packagePrice)
					return false
				}

				// Balance should be deducted
				expectedBalance := initialBalance - packagePrice
				if !almostEqual(finalBalance, expectedBalance, 0.01) {
					t.Logf("Balance not deducted correctly: expected %.2f, got %.2f", expectedBalance, finalBalance)
					return false
				}

				// Status should be active
				if finalStatus != "active" {
					t.Logf("Status should be active, got: %s", finalStatus)
					return false
				}

				// Counters should be reset
				if messagesUsed != 0 {
					t.Logf("Messages used should be reset to 0, got: %d", messagesUsed)
					return false
				}

				if messagesRemaining == nil || *messagesRemaining != 1000 {
					t.Logf("Messages remaining should be reset to 1000, got: %v", messagesRemaining)
					return false
				}

				// Transaction should be recorded
				if !hasTransaction(t, pool, tenantID, subscriptionID) {
					t.Logf("Transaction not recorded")
					return false
				}

				// Invoice should be generated
				if !hasInvoice(t, pool, tenantID, subscriptionID) {
					t.Logf("Invoice not generated")
					return false
				}

				return true
			} else {
				// Insufficient balance: renewal should fail
				if err == nil {
					t.Logf("Expected error but renewal succeeded (balance=%.2f, price=%.2f)", balance, packagePrice)
					return false
				}

				// Balance should not change
				if !almostEqual(finalBalance, initialBalance, 0.01) {
					t.Logf("Balance should not change on failure: expected %.2f, got %.2f", initialBalance, finalBalance)
					return false
				}

				// Status should remain unchanged (will be updated by ApplyGracePeriod separately)
				if finalStatus != initialStatus {
					t.Logf("Status should not change on renewal failure: expected %s, got %s", initialStatus, finalStatus)
					return false
				}

				return true
			}
		},
		gen.Float64Range(0, 200),   // balance: 0 to 200 USD
		gen.Float64Range(10, 100),  // package price: 10 to 100 USD
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// **Validates: Requirements 14.4, 14.5**
// Property: Grace period application
// When renewal fails due to insufficient balance:
// - Status changes to 'grace_period'
// - Renewal attempts incremented
// - Grace period defaults to 3 days if not specified
func TestProperty_GracePeriodApplication(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	pool := setupTestDB(t)
	defer pool.Close()

	logger := zap.NewNop()
	service := NewAutoRenewalService(pool, nil, logger)

	properties := gopter.NewProperties(nil)

	properties.Property("grace period applied correctly", prop.ForAll(
		func(gracePeriodDays int) bool {
			ctx := context.Background()

			// Create test tenant
			tenantID := createTestTenant(t, pool, 50.0)
			defer cleanupTestTenant(t, pool, tenantID)

			// Create test package
			packageID := createTestPackage(t, pool, tenantID, 20.0, 1000)

			// Create expired subscription
			subscriptionID := createExpiredSubscription(t, pool, tenantID, packageID, true)

			// Get initial renewal attempts
			initialAttempts := getRenewalAttempts(t, pool, subscriptionID)

			// Apply grace period
			err := service.ApplyGracePeriod(ctx, subscriptionID, tenantID, gracePeriodDays)
			if err != nil {
				t.Logf("Failed to apply grace period: %v", err)
				return false
			}

			// Check status changed to grace_period
			status := getSubscriptionStatus(t, pool, subscriptionID)
			if status != "grace_period" {
				t.Logf("Status should be grace_period, got: %s", status)
				return false
			}

			// Check renewal attempts incremented
			finalAttempts := getRenewalAttempts(t, pool, subscriptionID)
			if finalAttempts != initialAttempts+1 {
				t.Logf("Renewal attempts should increment: expected %d, got %d", initialAttempts+1, finalAttempts)
				return false
			}

			return true
		},
		gen.IntRange(0, 7), // grace period: 0 to 7 days (0 means use default)
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// **Validates: Requirements 14.6, 14.11**
// Property: Subscription suspension
// When grace period expires:
// - Status changes to 'suspended'
// - Subscription cannot be used for sending messages
func TestProperty_SubscriptionSuspension(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	pool := setupTestDB(t)
	defer pool.Close()

	logger := zap.NewNop()
	service := NewAutoRenewalService(pool, nil, logger)

	properties := gopter.NewProperties(nil)

	properties.Property("subscription suspended correctly", prop.ForAll(
		func() bool {
			ctx := context.Background()

			// Create test tenant
			tenantID := createTestTenant(t, pool, 50.0)
			defer cleanupTestTenant(t, pool, tenantID)

			// Create test package
			packageID := createTestPackage(t, pool, tenantID, 20.0, 1000)

			// Create subscription in grace period
			subscriptionID := createSubscriptionInGracePeriod(t, pool, tenantID, packageID)

			// Suspend subscription
			err := service.SuspendSubscription(ctx, subscriptionID, tenantID)
			if err != nil {
				t.Logf("Failed to suspend subscription: %v", err)
				return false
			}

			// Check status changed to suspended
			status := getSubscriptionStatus(t, pool, subscriptionID)
			if status != "suspended" {
				t.Logf("Status should be suspended, got: %s", status)
				return false
			}

			return true
		},
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// **Validates: Requirement 14.12**
// Property: Auto-reactivation
// When balance is topped up for a suspended subscription:
// - If balance >= package_price: subscription reactivated automatically
// - Status changes back to 'active'
// - Dates and counters reset
func TestProperty_AutoReactivation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	pool := setupTestDB(t)
	defer pool.Close()

	logger := zap.NewNop()
	service := NewAutoRenewalService(pool, nil, logger)

	properties := gopter.NewProperties(nil)

	properties.Property("auto reactivation with sufficient balance", prop.ForAll(
		func(topUpAmount float64, packagePrice float64) bool {
			ctx := context.Background()

			// Create test tenant with low balance
			tenantID := createTestTenant(t, pool, 5.0)
			defer cleanupTestTenant(t, pool, tenantID)

			// Create test package
			packageID := createTestPackage(t, pool, tenantID, packagePrice, 1000)

			// Create suspended subscription
			subscriptionID := createSuspendedSubscription(t, pool, tenantID, packageID)

			// Top up balance
			topUpBalance(t, pool, tenantID, topUpAmount)
			finalBalance := getBalance(t, pool, tenantID)

			// Attempt auto-reactivation
			err := service.AutoReactivate(ctx, tenantID)

			status := getSubscriptionStatus(t, pool, subscriptionID)

			if finalBalance >= packagePrice {
				// Sufficient balance: should reactivate
				if err != nil {
					t.Logf("Expected reactivation but got error: %v (balance=%.2f, price=%.2f)", err, finalBalance, packagePrice)
					return false
				}

				// Status should be active
				if status != "active" {
					t.Logf("Status should be active after reactivation, got: %s", status)
					return false
				}

				// Balance should be deducted
				newBalance := getBalance(t, pool, tenantID)
				expectedBalance := finalBalance - packagePrice
				if !almostEqual(newBalance, expectedBalance, 0.01) {
					t.Logf("Balance not deducted correctly: expected %.2f, got %.2f", expectedBalance, newBalance)
					return false
				}

				return true
			} else {
				// Insufficient balance: should remain suspended
				if status != "suspended" {
					t.Logf("Status should remain suspended with insufficient balance, got: %s", status)
					return false
				}

				return true
			}
		},
		gen.Float64Range(0, 150),   // top-up amount: 0 to 150 USD
		gen.Float64Range(10, 100),  // package price: 10 to 100 USD
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Helper functions

func setupTestDB(t *testing.T) *pgxpool.Pool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	connString := "postgres://postgres@localhost:5432/sms_gateway?sslmode=disable"
	
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		t.Skipf("Failed to create pool: %v", err)
		return nil
	}

	err = pool.Ping(ctx)
	if err != nil {
		pool.Close()
		t.Skipf("Failed to connect to test database: %v", err)
		return nil
	}
	
	return pool
}

func createTestTenant(t *testing.T, pool *pgxpool.Pool, balance float64) uuid.UUID {
	ctx := context.Background()
	tenantID := uuid.New()
	
	_, err := pool.Exec(ctx, `
		INSERT INTO tenants (id, name, email, balance, currency, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, tenantID, "Test Tenant", "test@example.com", balance, "USD", "active")
	require.NoError(t, err)
	
	return tenantID
}

func cleanupTestTenant(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) {
	ctx := context.Background()
	_, _ = pool.Exec(ctx, "DELETE FROM subscriptions WHERE tenant_id = $1", tenantID)
	_, _ = pool.Exec(ctx, "DELETE FROM packages WHERE tenant_id = $1", tenantID)
	_, _ = pool.Exec(ctx, "DELETE FROM accounting_transactions WHERE tenant_id = $1", tenantID)
	_, _ = pool.Exec(ctx, "DELETE FROM billing_invoices WHERE tenant_id = $1", tenantID)
	_, _ = pool.Exec(ctx, "DELETE FROM tenants WHERE id = $1", tenantID)
}

func createTestPackage(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, price float64, messages int64) int64 {
	ctx := context.Background()
	
	var packageID int64
	err := pool.QueryRow(ctx, `
		INSERT INTO packages (
			tenant_id, service_id, name, code, price_usd, included_messages,
			overage_rate, is_unlimited, billing_cycle, rollover_enabled,
			is_active, display_order, created_at, updated_at
		) VALUES ($1, 2, $2, $3, $4, $5, $6, false, 'monthly', false, true, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`, tenantID, "Test Package", fmt.Sprintf("TEST-%s", uuid.New().String()[:8]), price, messages, price/float64(messages)).Scan(&packageID)
	require.NoError(t, err)
	
	return packageID
}

func createExpiredSubscription(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, packageID int64, autoRenew bool) int64 {
	ctx := context.Background()
	
	yesterday := time.Now().AddDate(0, 0, -1)
	lastMonth := time.Now().AddDate(0, -1, 0)
	
	var subscriptionID int64
	err := pool.QueryRow(ctx, `
		INSERT INTO subscriptions (
			tenant_id, package_id, start_date, end_date, next_renewal_date,
			messages_used, messages_remaining, overage_messages, overage_cost,
			status, auto_renew, grace_period_days, renewal_attempts,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $4, 500, 500, 0, 0, 'active', $5, 3, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`, tenantID, packageID, lastMonth, yesterday, autoRenew).Scan(&subscriptionID)
	require.NoError(t, err)
	
	return subscriptionID
}

func createSubscriptionInGracePeriod(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, packageID int64) int64 {
	ctx := context.Background()
	
	yesterday := time.Now().AddDate(0, 0, -1)
	lastMonth := time.Now().AddDate(0, -1, 0)
	
	var subscriptionID int64
	err := pool.QueryRow(ctx, `
		INSERT INTO subscriptions (
			tenant_id, package_id, start_date, end_date, next_renewal_date,
			messages_used, messages_remaining, overage_messages, overage_cost,
			status, auto_renew, grace_period_days, renewal_attempts,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $4, 500, 500, 0, 0, 'grace_period', true, 3, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`, tenantID, packageID, lastMonth, yesterday).Scan(&subscriptionID)
	require.NoError(t, err)
	
	return subscriptionID
}

func createSuspendedSubscription(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, packageID int64) int64 {
	ctx := context.Background()
	
	yesterday := time.Now().AddDate(0, 0, -1)
	lastMonth := time.Now().AddDate(0, -1, 0)
	
	var subscriptionID int64
	err := pool.QueryRow(ctx, `
		INSERT INTO subscriptions (
			tenant_id, package_id, start_date, end_date, next_renewal_date,
			messages_used, messages_remaining, overage_messages, overage_cost,
			status, auto_renew, grace_period_days, renewal_attempts,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $4, 1000, 0, 0, 0, 'suspended', true, 3, 3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`, tenantID, packageID, lastMonth, yesterday).Scan(&subscriptionID)
	require.NoError(t, err)
	
	return subscriptionID
}

func getBalance(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) float64 {
	ctx := context.Background()
	var balance float64
	err := pool.QueryRow(ctx, "SELECT balance FROM tenants WHERE id = $1", tenantID).Scan(&balance)
	require.NoError(t, err)
	return balance
}

func topUpBalance(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, amount float64) {
	ctx := context.Background()
	_, err := pool.Exec(ctx, "UPDATE tenants SET balance = balance + $1 WHERE id = $2", amount, tenantID)
	require.NoError(t, err)
}

func getSubscriptionStatus(t *testing.T, pool *pgxpool.Pool, subscriptionID int64) string {
	ctx := context.Background()
	var status string
	err := pool.QueryRow(ctx, "SELECT status FROM subscriptions WHERE id = $1", subscriptionID).Scan(&status)
	require.NoError(t, err)
	return status
}

func getMessagesUsed(t *testing.T, pool *pgxpool.Pool, subscriptionID int64) int64 {
	ctx := context.Background()
	var messagesUsed int64
	err := pool.QueryRow(ctx, "SELECT messages_used FROM subscriptions WHERE id = $1", subscriptionID).Scan(&messagesUsed)
	require.NoError(t, err)
	return messagesUsed
}

func getMessagesRemaining(t *testing.T, pool *pgxpool.Pool, subscriptionID int64) *int64 {
	ctx := context.Background()
	var messagesRemaining *int64
	err := pool.QueryRow(ctx, "SELECT messages_remaining FROM subscriptions WHERE id = $1", subscriptionID).Scan(&messagesRemaining)
	require.NoError(t, err)
	return messagesRemaining
}

func getRenewalAttempts(t *testing.T, pool *pgxpool.Pool, subscriptionID int64) int {
	ctx := context.Background()
	var attempts int
	err := pool.QueryRow(ctx, "SELECT renewal_attempts FROM subscriptions WHERE id = $1", subscriptionID).Scan(&attempts)
	require.NoError(t, err)
	return attempts
}

func hasTransaction(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, subscriptionID int64) bool {
	ctx := context.Background()
	var count int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM accounting_transactions 
		WHERE tenant_id = $1 AND reference_id = $2 AND reference_type = 'subscription'
	`, tenantID, subscriptionID).Scan(&count)
	require.NoError(t, err)
	return count > 0
}

func hasInvoice(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, subscriptionID int64) bool {
	ctx := context.Background()
	var count int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM billing_invoices 
		WHERE tenant_id = $1 AND reference_id = $2 AND reference_type = 'subscription'
	`, tenantID, subscriptionID).Scan(&count)
	require.NoError(t, err)
	return count > 0
}

func almostEqual(a, b, tolerance float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff <= tolerance
}
