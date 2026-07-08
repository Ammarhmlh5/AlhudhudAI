package rollover

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

// setupRolloverDB creates a test database connection
func setupRolloverDB(t *testing.T) *pgxpool.Pool {
	// Use test database connection
	connString := "postgres://postgres:postgres@localhost:5432/alhudhud_test?sslmode=disable"
	
	pool, err := pgxpool.New(context.Background(), connString)
	require.NoError(t, err)
	
	return pool
}

// createTestTenant creates a test tenant
func createTestTenant(t *testing.T, pool *pgxpool.Pool, name string, balance decimal.Decimal) uuid.UUID {
	tenantID := uuid.New()
	
	query := `
		INSERT INTO tenants (id, name, balance, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`
	
	_, err := pool.Exec(context.Background(), query, tenantID, name, balance)
	require.NoError(t, err)
	
	return tenantID
}

// createTestService creates a test service
func createTestService(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, name string) int64 {
	var serviceID int64
	
	query := `
		INSERT INTO tariff_services (tenant_id, name, code, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, true, NOW(), NOW())
		RETURNING id
	`
	
	err := pool.QueryRow(context.Background(), query, tenantID, name, name).Scan(&serviceID)
	require.NoError(t, err)
	
	return serviceID
}

// createRolloverPackage creates a test package with rollover enabled
func createRolloverPackage(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, serviceID int64, name string, price decimal.Decimal, includedMessages int64, overageRate decimal.Decimal, rolloverEnabled bool, rolloverLimit *int64, rolloverExpiryMonths *int) int64 {
	var packageID int64
	
	query := `
		INSERT INTO packages (
			tenant_id, service_id, name, code, price_usd,
			included_messages, overage_rate, is_unlimited, billing_cycle,
			rollover_enabled, rollover_limit, rollover_expiry_months,
			is_active, display_order, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, false, 'monthly',
			$8, $9, $10,
			true, 1, NOW(), NOW()
		)
		RETURNING id
	`
	
	err := pool.QueryRow(context.Background(), query,
		tenantID, serviceID, name, name, price,
		includedMessages, overageRate,
		rolloverEnabled, rolloverLimit, rolloverExpiryMonths,
	).Scan(&packageID)
	require.NoError(t, err)
	
	return packageID
}

// createRolloverSubscription creates a test subscription
func createRolloverSubscription(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, packageID int64, messagesRemaining int64) int64 {
	var subscriptionID int64
	
	startDate := time.Now()
	endDate := startDate.AddDate(0, 1, 0)
	
	query := `
		INSERT INTO subscriptions (
			tenant_id, package_id, start_date, end_date,
			messages_used, messages_remaining, overage_messages, overage_cost,
			status, auto_renew, grace_period_days, next_renewal_date,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			0, $5, 0, 0,
			'active', true, 3, $4,
			NOW(), NOW()
		)
		RETURNING id
	`
	
	err := pool.QueryRow(context.Background(), query,
		tenantID, packageID, startDate, endDate, messagesRemaining,
	).Scan(&subscriptionID)
	require.NoError(t, err)
	
	return subscriptionID
}

// TestCalculateRemainingMessages tests the CalculateRemainingMessages function
func TestCalculateRemainingMessages(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	pool := setupRolloverDB(t)
	defer pool.Close()

	logger := zap.NewNop()
	manager := NewManager(pool, logger)

	ctx := context.Background()

	// Create test tenant
	tenantID := createTestTenant(t, pool, "Test Tenant", decimal.NewFromInt(100))

	// Create test service
	serviceID := createTestService(t, pool, tenantID, "Test Service")

	// Create test package with rollover enabled
	rolloverLimit := int64(500)
	rolloverExpiryMonths := 3
	packageID := createRolloverPackage(t, pool, tenantID, serviceID, "Basic", decimal.NewFromInt(10), 1000, decimal.NewFromFloat(0.01), true, &rolloverLimit, &rolloverExpiryMonths)

	// Create test subscription with 300 messages remaining
	subscriptionID := createRolloverSubscription(t, pool, tenantID, packageID, 300)

	// Test CalculateRemainingMessages
	remaining, err := manager.CalculateRemainingMessages(ctx, subscriptionID)
	require.NoError(t, err)
	assert.Equal(t, int64(300), remaining)
}

// TestCalculateRemainingMessages_RolloverDisabled tests when rollover is disabled
func TestCalculateRemainingMessages_RolloverDisabled(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	pool := setupRolloverDB(t)
	defer pool.Close()

	logger := zap.NewNop()
	manager := NewManager(pool, logger)

	ctx := context.Background()

	// Create test tenant
	tenantID := createTestTenant(t, pool, "Test Tenant", decimal.NewFromInt(100))

	// Create test service
	serviceID := createTestService(t, pool, tenantID, "Test Service")

	// Create test package with rollover DISABLED
	packageID := createRolloverPackage(t, pool, tenantID, serviceID, "Basic", decimal.NewFromInt(10), 1000, decimal.NewFromFloat(0.01), false, nil, nil)

	// Create test subscription with 300 messages remaining
	subscriptionID := createRolloverSubscription(t, pool, tenantID, packageID, 300)

	// Test CalculateRemainingMessages - should return 0 because rollover is disabled
	remaining, err := manager.CalculateRemainingMessages(ctx, subscriptionID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), remaining)
}

// TestCalculateRemainingMessages_NoMessagesRemaining tests when no messages remaining
func TestCalculateRemainingMessages_NoMessagesRemaining(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	pool := setupRolloverDB(t)
	defer pool.Close()

	logger := zap.NewNop()
	manager := NewManager(pool, logger)

	ctx := context.Background()

	// Create test tenant
	tenantID := createTestTenant(t, pool, "Test Tenant", decimal.NewFromInt(100))

	// Create test service
	serviceID := createTestService(t, pool, tenantID, "Test Service")

	// Create test package with rollover enabled
	rolloverLimit := int64(500)
	rolloverExpiryMonths := 3
	packageID := createRolloverPackage(t, pool, tenantID, serviceID, "Basic", decimal.NewFromInt(10), 1000, decimal.NewFromFloat(0.01), true, &rolloverLimit, &rolloverExpiryMonths)

	// Create test subscription with 0 messages remaining
	subscriptionID := createRolloverSubscription(t, pool, tenantID, packageID, 0)

	// Test CalculateRemainingMessages - should return 0 because no messages remaining
	remaining, err := manager.CalculateRemainingMessages(ctx, subscriptionID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), remaining)
}

// TestGetRolloverMessages tests the GetRolloverMessages function
func TestGetRolloverMessages(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	pool := setupRolloverDB(t)
	defer pool.Close()

	logger := zap.NewNop()
	manager := NewManager(pool, logger)

	ctx := context.Background()

	// Create test tenant
	tenantID := createTestTenant(t, pool, "Test Tenant", decimal.NewFromInt(100))

	// Create test service
	serviceID := createTestService(t, pool, tenantID, "Test Service")

	// Create test package with rollover enabled
	rolloverLimit := int64(500)
	rolloverExpiryMonths := 3
	packageID := createRolloverPackage(t, pool, tenantID, serviceID, "Basic", decimal.NewFromInt(10), 1000, decimal.NewFromFloat(0.01), true, &rolloverLimit, &rolloverExpiryMonths)

	// Create test subscription
	subscriptionID := createRolloverSubscription(t, pool, tenantID, packageID, 300)

	// Insert test rollover messages
	sourceMonth := time.Now().AddDate(0, -1, 0)
	expiryDate := time.Now().AddDate(0, 3, 0)

	insertQuery := `
		INSERT INTO rollover_messages (
			subscription_id, tenant_id, messages_count, source_month, expiry_date, created_at
		) VALUES ($1, $2, $3, $4, $5, NOW())
	`

	_, err := pool.Exec(ctx, insertQuery, subscriptionID, tenantID, 200, sourceMonth, expiryDate)
	require.NoError(t, err)

	// Test GetRolloverMessages
	messages, err := manager.GetRolloverMessages(ctx, subscriptionID)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	assert.Equal(t, int64(200), messages[0].MessagesCount)
	assert.Equal(t, subscriptionID, messages[0].SubscriptionID)
}
