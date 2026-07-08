package billing

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// setupGatewayBillingTest sets up test environment
func setupGatewayBillingTest(t *testing.T) (*pgxpool.Pool, *GatewayBillingService, func()) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Database connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	connString := "postgres://postgres:postgres@localhost:5432/sms_gateway_test?sslmode=disable"
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		t.Skipf("Failed to create pool: %v", err)
		return nil, nil, nil
	}

	// Ping to check if DB is actually available
	err = pool.Ping(ctx)
	if err != nil {
		pool.Close()
		t.Skipf("Failed to connect to test database: %v", err)
		return nil, nil, nil
	}

	// Logger
	logger, _ := zap.NewDevelopment()

	// Service
	service := NewGatewayBillingService(pool, logger)

	// Cleanup function
	cleanup := func() {
		pool.Close()
	}

	return pool, service, cleanup
}

// createGatewayBillingTestTenant creates a test tenant with balance
func createGatewayBillingTestTenant(t *testing.T, pool *pgxpool.Pool, balance decimal.Decimal) uuid.UUID {
	tenantID := uuid.New()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO tenants (id, name, email, balance, created_at, updated_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, tenantID, "Test Enterprise", "test@enterprise.com", balance)
	require.NoError(t, err, "Failed to create test tenant")
	return tenantID
}

// createTestGateway creates a test dedicated gateway
func createTestGateway(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, subscriptionFee, serverRentalFee decimal.Decimal, billingCycle string) int64 {
	var gatewayID int64
	err := pool.QueryRow(context.Background(), `
		INSERT INTO dedicated_gateways (
			tenant_id,
			gateway_name,
			gateway_code,
			telecom_provider,
			telecom_api_endpoint,
			subscription_type,
			subscription_fee,
			server_rental_fee,
			billing_cycle,
			next_billing_date,
			contract_start_date,
			status,
			created_at,
			updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`,
		tenantID,
		"Test Gateway",
		"TEST-GW-001",
		"Yemen Mobile",
		"https://api.yemenmobile.com",
		billingCycle,
		subscriptionFee,
		serverRentalFee,
		billingCycle,
		time.Now(), // Due today
		time.Now(),
		"active",
	).Scan(&gatewayID)
	require.NoError(t, err, "Failed to create test gateway")
	return gatewayID
}

// cleanupTestData removes test data
func cleanupTestData(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) {
	_, err := pool.Exec(context.Background(), "DELETE FROM dedicated_gateways WHERE tenant_id = $1", tenantID)
	assert.NoError(t, err)
	_, err = pool.Exec(context.Background(), "DELETE FROM accounting_transactions WHERE tenant_id = $1", tenantID)
	assert.NoError(t, err)
	_, err = pool.Exec(context.Background(), "DELETE FROM billing_invoices WHERE tenant_id = $1", tenantID)
	assert.NoError(t, err)
	_, err = pool.Exec(context.Background(), "DELETE FROM tenants WHERE id = $1", tenantID)
	assert.NoError(t, err)
}

// TestBillGatewaySubscription_Monthly tests monthly billing with sufficient balance
func TestBillGatewaySubscription_Monthly(t *testing.T) {
	pool, service, cleanup := setupGatewayBillingTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create tenant with sufficient balance
	subscriptionFee := decimal.NewFromFloat(100.00)
	serverRentalFee := decimal.NewFromFloat(50.00)
	totalAmount := subscriptionFee.Add(serverRentalFee) // 150.00
	initialBalance := decimal.NewFromFloat(500.00)

	tenantID := createGatewayBillingTestTenant(t, pool, initialBalance)
	defer cleanupTestData(t, pool, tenantID)

	// Create gateway
	gatewayID := createTestGateway(t, pool, tenantID, subscriptionFee, serverRentalFee, "monthly")

	// Get billing info
	info, err := service.GetGatewayBillingInfo(ctx, tenantID, gatewayID)
	require.NoError(t, err)

	// Bill the gateway
	err = service.BillGatewaySubscription(ctx, info)
	require.NoError(t, err, "Billing should succeed with sufficient balance")

	// Verify balance was deducted
	var newBalance decimal.Decimal
	err = pool.QueryRow(ctx, "SELECT balance FROM tenants WHERE id = $1", tenantID).Scan(&newBalance)
	require.NoError(t, err)
	expectedBalance := initialBalance.Sub(totalAmount)
	assert.True(t, newBalance.Equal(expectedBalance), "Balance should be deducted correctly")

	// Verify transaction was recorded
	var transactionCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM accounting_transactions 
		WHERE tenant_id = $1 AND transaction_type = 'GATEWAY_SUBSCRIPTION'
	`, tenantID).Scan(&transactionCount)
	require.NoError(t, err)
	assert.Equal(t, 1, transactionCount, "Transaction should be recorded")

	// Verify invoice was created
	var invoiceCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM billing_invoices 
		WHERE tenant_id = $1 AND invoice_type = 'gateway_subscription'
	`, tenantID).Scan(&invoiceCount)
	require.NoError(t, err)
	assert.Equal(t, 1, invoiceCount, "Invoice should be created")

	// Verify next billing date was updated (should be 1 month from now)
	var nextBillingDate time.Time
	err = pool.QueryRow(ctx, "SELECT next_billing_date FROM dedicated_gateways WHERE id = $1", gatewayID).Scan(&nextBillingDate)
	require.NoError(t, err)
	expectedNextBilling := time.Now().AddDate(0, 1, 0)
	assert.WithinDuration(t, expectedNextBilling, nextBillingDate, 24*time.Hour, "Next billing date should be 1 month from now")
}

// TestBillGatewaySubscription_Yearly tests yearly billing
func TestBillGatewaySubscription_Yearly(t *testing.T) {
	pool, service, cleanup := setupGatewayBillingTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create tenant with sufficient balance
	subscriptionFee := decimal.NewFromFloat(1000.00) // Yearly subscription
	serverRentalFee := decimal.NewFromFloat(50.00)   // Monthly rental
	_ = subscriptionFee.Add(serverRentalFee)
	initialBalance := decimal.NewFromFloat(2000.00)

	tenantID := createGatewayBillingTestTenant(t, pool, initialBalance)
	defer cleanupTestData(t, pool, tenantID)

	// Create gateway with yearly billing
	gatewayID := createTestGateway(t, pool, tenantID, subscriptionFee, serverRentalFee, "yearly")

	// Get billing info
	info, err := service.GetGatewayBillingInfo(ctx, tenantID, gatewayID)
	require.NoError(t, err)

	// Bill the gateway
	err = service.BillGatewaySubscription(ctx, info)
	require.NoError(t, err)

	// Verify next billing date was updated (should be 1 year from now)
	var nextBillingDate time.Time
	err = pool.QueryRow(ctx, "SELECT next_billing_date FROM dedicated_gateways WHERE id = $1", gatewayID).Scan(&nextBillingDate)
	require.NoError(t, err)
	expectedNextBilling := time.Now().AddDate(1, 0, 0)
	assert.WithinDuration(t, expectedNextBilling, nextBillingDate, 24*time.Hour, "Next billing date should be 1 year from now")
}

// TestBillGatewaySubscription_InsufficientBalance tests billing with insufficient balance
func TestBillGatewaySubscription_InsufficientBalance(t *testing.T) {
	pool, service, cleanup := setupGatewayBillingTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create tenant with insufficient balance
	subscriptionFee := decimal.NewFromFloat(100.00)
	serverRentalFee := decimal.NewFromFloat(50.00)
	initialBalance := decimal.NewFromFloat(100.00) // Less than required 150.00

	tenantID := createGatewayBillingTestTenant(t, pool, initialBalance)
	defer cleanupTestData(t, pool, tenantID)

	// Create gateway
	gatewayID := createTestGateway(t, pool, tenantID, subscriptionFee, serverRentalFee, "monthly")

	// Get billing info
	info, err := service.GetGatewayBillingInfo(ctx, tenantID, gatewayID)
	require.NoError(t, err)

	// Bill the gateway - should fail
	err = service.BillGatewaySubscription(ctx, info)
	require.Error(t, err, "Billing should fail with insufficient balance")
	assert.Contains(t, err.Error(), "insufficient balance")

	// Verify balance was NOT deducted
	var currentBalance decimal.Decimal
	err = pool.QueryRow(ctx, "SELECT balance FROM tenants WHERE id = $1", tenantID).Scan(&currentBalance)
	require.NoError(t, err)
	assert.True(t, currentBalance.Equal(initialBalance), "Balance should not be deducted on failure")

	// Verify no transaction was recorded
	var transactionCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM accounting_transactions 
		WHERE tenant_id = $1 AND transaction_type = 'GATEWAY_SUBSCRIPTION'
	`, tenantID).Scan(&transactionCount)
	require.NoError(t, err)
	assert.Equal(t, 0, transactionCount, "No transaction should be recorded on failure")

	// Verify no invoice was created
	var invoiceCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM billing_invoices 
		WHERE tenant_id = $1 AND invoice_type = 'gateway_subscription'
	`, tenantID).Scan(&invoiceCount)
	require.NoError(t, err)
	assert.Equal(t, 0, invoiceCount, "No invoice should be created on failure")
}

// TestBillGatewaySubscription_NoMessageCosts tests that message costs are NOT charged
// Requirement 8.15: The system SHALL NOT process any payments related to message costs
func TestBillGatewaySubscription_NoMessageCosts(t *testing.T) {
	pool, service, cleanup := setupGatewayBillingTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create tenant
	subscriptionFee := decimal.NewFromFloat(100.00)
	serverRentalFee := decimal.NewFromFloat(50.00)
	initialBalance := decimal.NewFromFloat(500.00)

	tenantID := createGatewayBillingTestTenant(t, pool, initialBalance)
	defer cleanupTestData(t, pool, tenantID)

	// Create gateway
	gatewayID := createTestGateway(t, pool, tenantID, subscriptionFee, serverRentalFee, "monthly")

	// Get billing info
	info, err := service.GetGatewayBillingInfo(ctx, tenantID, gatewayID)
	require.NoError(t, err)

	// Bill the gateway
	err = service.BillGatewaySubscription(ctx, info)
	require.NoError(t, err)

	// Verify transaction description does NOT mention message costs
	var description string
	err = pool.QueryRow(ctx, `
		SELECT description FROM accounting_transactions 
		WHERE tenant_id = $1 AND transaction_type = 'GATEWAY_SUBSCRIPTION'
	`, tenantID).Scan(&description)
	require.NoError(t, err)
	assert.NotContains(t, description, "message", "Transaction should not mention message costs")
	assert.NotContains(t, description, "SMS", "Transaction should not mention SMS costs")
	assert.Contains(t, description, "Subscription", "Transaction should mention subscription")
	assert.Contains(t, description, "Server Rental", "Transaction should mention server rental")

	// Verify invoice description does NOT mention message costs
	var invoiceDescription string
	err = pool.QueryRow(ctx, `
		SELECT description FROM billing_invoices 
		WHERE tenant_id = $1 AND invoice_type = 'gateway_subscription'
	`, tenantID).Scan(&invoiceDescription)
	require.NoError(t, err)
	assert.NotContains(t, invoiceDescription, "message cost", "Invoice should not mention message costs")
	assert.Contains(t, invoiceDescription, "Subscription Fee", "Invoice should mention subscription fee")
	assert.Contains(t, invoiceDescription, "Server Rental", "Invoice should mention server rental")
}

// TestCheckDueGateways tests the daily check for due gateways
func TestCheckDueGateways(t *testing.T) {
	pool, service, cleanup := setupGatewayBillingTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create multiple tenants and gateways
	subscriptionFee := decimal.NewFromFloat(100.00)
	serverRentalFee := decimal.NewFromFloat(50.00)
	initialBalance := decimal.NewFromFloat(500.00)

	// Tenant 1 with due gateway
	tenantID1 := createGatewayBillingTestTenant(t, pool, initialBalance)
	defer cleanupTestData(t, pool, tenantID1)
	createTestGateway(t, pool, tenantID1, subscriptionFee, serverRentalFee, "monthly")

	// Tenant 2 with due gateway
	tenantID2 := createGatewayBillingTestTenant(t, pool, initialBalance)
	defer cleanupTestData(t, pool, tenantID2)
	createTestGateway(t, pool, tenantID2, subscriptionFee, serverRentalFee, "monthly")

	// Run check
	err := service.CheckDueGateways(ctx)
	require.NoError(t, err)

	// Verify both gateways were billed
	var count1 int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM accounting_transactions 
		WHERE tenant_id = $1 AND transaction_type = 'GATEWAY_SUBSCRIPTION'
	`, tenantID1).Scan(&count1)
	require.NoError(t, err)
	assert.Equal(t, 1, count1, "Tenant 1 gateway should be billed")

	var count2 int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM accounting_transactions 
		WHERE tenant_id = $2 AND transaction_type = 'GATEWAY_SUBSCRIPTION'
	`, tenantID2).Scan(&count2)
	require.NoError(t, err)
	assert.Equal(t, 1, count2, "Tenant 2 gateway should be billed")
}

// TestGetUpcomingBillings tests retrieving upcoming billings
func TestGetUpcomingBillings(t *testing.T) {
	pool, service, cleanup := setupGatewayBillingTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create tenant and gateway
	subscriptionFee := decimal.NewFromFloat(100.00)
	serverRentalFee := decimal.NewFromFloat(50.00)
	initialBalance := decimal.NewFromFloat(500.00)

	tenantID := createGatewayBillingTestTenant(t, pool, initialBalance)
	defer cleanupTestData(t, pool, tenantID)
	createTestGateway(t, pool, tenantID, subscriptionFee, serverRentalFee, "monthly")

	// Get upcoming billings (next 7 days)
	billings, err := service.GetUpcomingBillings(ctx, 7)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(billings), 1, "Should have at least one upcoming billing")

	// Verify billing info
	found := false
	for _, billing := range billings {
		if billing.TenantID == tenantID {
			found = true
			assert.Equal(t, "Test Gateway", billing.GatewayName)
			assert.True(t, billing.SubscriptionFee.Equal(subscriptionFee))
			assert.True(t, billing.ServerRentalFee.Equal(serverRentalFee))
			break
		}
	}
	assert.True(t, found, "Should find the test gateway in upcoming billings")
}

// TestBillGatewaySubscription_Atomicity tests transaction atomicity
func TestBillGatewaySubscription_Atomicity(t *testing.T) {
	pool, service, cleanup := setupGatewayBillingTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create tenant with exact balance (edge case)
	subscriptionFee := decimal.NewFromFloat(100.00)
	serverRentalFee := decimal.NewFromFloat(50.00)
	totalAmount := subscriptionFee.Add(serverRentalFee)
	initialBalance := totalAmount // Exact amount

	tenantID := createGatewayBillingTestTenant(t, pool, initialBalance)
	defer cleanupTestData(t, pool, tenantID)

	// Create gateway
	gatewayID := createTestGateway(t, pool, tenantID, subscriptionFee, serverRentalFee, "monthly")

	// Get billing info
	info, err := service.GetGatewayBillingInfo(ctx, tenantID, gatewayID)
	require.NoError(t, err)

	// Bill the gateway
	err = service.BillGatewaySubscription(ctx, info)
	require.NoError(t, err)

	// Verify balance is exactly zero
	var newBalance decimal.Decimal
	err = pool.QueryRow(ctx, "SELECT balance FROM tenants WHERE id = $1", tenantID).Scan(&newBalance)
	require.NoError(t, err)
	assert.True(t, newBalance.IsZero(), "Balance should be exactly zero")

	// Verify all records were created atomically
	var transactionCount, invoiceCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM accounting_transactions WHERE tenant_id = $1
	`, tenantID).Scan(&transactionCount)
	require.NoError(t, err)
	assert.Equal(t, 1, transactionCount)

	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM billing_invoices WHERE tenant_id = $1
	`, tenantID).Scan(&invoiceCount)
	require.NoError(t, err)
	assert.Equal(t, 1, invoiceCount)
}
