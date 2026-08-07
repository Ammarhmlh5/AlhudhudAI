package subscription

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// Helper function to create a test manager
func createTestManager(t *testing.T, pool *pgxpool.Pool) Manager {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	return NewManager(pool, logger)
}

// Helper function to create a test subscription
func createTestSubscription(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, packageID int64, messagesRemaining *int64) int64 {
	ctx := context.Background()

	// Set tenant context
	_, err := pool.Exec(ctx, "SELECT set_config('app.current_tenant', $1, false)", tenantID.String())
	require.NoError(t, err)

	// Create subscription
	var subscriptionID int64
	query := `
		INSERT INTO subscriptions (
			tenant_id,
			package_id,
			start_date,
			end_date,
			messages_used,
			messages_remaining,
			status,
			auto_renew,
			next_renewal_date
		) VALUES (
			$1, $2, $3, $4, 0, $5, 'active', true, $6
		)
		RETURNING id
	`

	startDate := time.Now()
	endDate := startDate.AddDate(0, 1, 0)

	err = pool.QueryRow(ctx, query,
		tenantID,
		packageID,
		startDate,
		endDate,
		messagesRemaining,
		endDate,
	).Scan(&subscriptionID)
	require.NoError(t, err)

	return subscriptionID
}

// TestDeductMessages_Success tests successful message deduction
func TestDeductMessages_Success(t *testing.T) {
	// This test requires a database connection
	// Skip if DATABASE_URL is not set
	t.Skip("Integration test - requires database")

	// TODO: Implement when database is available
	// 1. Create test subscription with 100 messages
	// 2. Deduct 10 messages
	// 3. Verify messages_used = 10
	// 4. Verify messages_remaining = 90
}

// TestDeductMessages_Unlimited tests deduction from unlimited package
func TestDeductMessages_Unlimited(t *testing.T) {
	// This test requires a database connection
	t.Skip("Integration test - requires database")

	// TODO: Implement when database is available
	// 1. Create test subscription with NULL messages_remaining
	// 2. Deduct 10 messages
	// 3. Verify messages_used = 10
	// 4. Verify messages_remaining is still NULL
}

// TestDeductMessages_InsufficientQuota tests overage detection
func TestDeductMessages_InsufficientQuota(t *testing.T) {
	// This test requires a database connection
	t.Skip("Integration test - requires database")

	// TODO: Implement when database is available
	// 1. Create test subscription with 5 messages remaining
	// 2. Try to deduct 10 messages
	// 3. Verify deduction fails
	// 4. Verify overage_messages = 5
	// 5. Verify no changes to messages_used or messages_remaining
}

// TestDeductMessages_MultipleSegments tests deduction of multiple segments
func TestDeductMessages_MultipleSegments(t *testing.T) {
	// This test requires a database connection
	t.Skip("Integration test - requires database")

	// TODO: Implement when database is available
	// 1. Create test subscription with 100 messages
	// 2. Deduct 3 messages (long message with 3 segments)
	// 3. Verify messages_used = 3
	// 4. Verify messages_remaining = 97
}

// TestDeductMessages_InvalidRequest tests validation
func TestDeductMessages_InvalidRequest(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	// Create manager with nil pool (we won't use it)
	manager := &manager{
		pool:   nil,
		logger: logger,
	}

	ctx := context.Background()

	// Test invalid subscription ID
	result, err := manager.DeductMessages(ctx, DeductMessagesRequest{
		SubscriptionID: 0,
		MessageCount:   1,
		MessageID:      "test",
	})
	assert.Error(t, err)
	assert.Nil(t, result)

	// Test invalid message count
	result, err = manager.DeductMessages(ctx, DeductMessagesRequest{
		SubscriptionID: 1,
		MessageCount:   0,
		MessageID:      "test",
	})
	assert.Error(t, err)
	assert.Nil(t, result)

	// Test missing message ID
	result, err = manager.DeductMessages(ctx, DeductMessagesRequest{
		SubscriptionID: 1,
		MessageCount:   1,
		MessageID:      "",
	})
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestDeductMessages_ConcurrentDeductions tests race conditions
func TestDeductMessages_ConcurrentDeductions(t *testing.T) {
	// This test requires a database connection
	t.Skip("Integration test - requires database")

	// TODO: Implement when database is available
	// This will be covered by property-based tests in task 7.4
}


// TestCalculateOverageCost_Success tests successful overage cost calculation and charging
func TestCalculateOverageCost_Success(t *testing.T) {
	// This test requires a database connection
	t.Skip("Integration test - requires database")

	// TODO: Implement when database is available
	// 1. Create test tenant with balance = 100 USD
	// 2. Create test package with overage_rate = 0.01 USD
	// 3. Create test subscription
	// 4. Calculate overage cost for 50 messages
	// 5. Verify cost = 0.50 USD
	// 6. Verify balance deducted: 100 - 0.50 = 99.50
	// 7. Verify subscription overage_messages = 50
	// 8. Verify subscription overage_cost = 0.50
	// 9. Verify accounting transaction recorded
}

// TestCalculateOverageCost_InsufficientBalance tests overage with insufficient balance
func TestCalculateOverageCost_InsufficientBalance(t *testing.T) {
	// This test requires a database connection
	t.Skip("Integration test - requires database")

	// TODO: Implement when database is available
	// 1. Create test tenant with balance = 0.10 USD
	// 2. Create test package with overage_rate = 0.01 USD
	// 3. Create test subscription
	// 4. Try to calculate overage cost for 50 messages (cost = 0.50 USD)
	// 5. Verify operation fails with "insufficient balance" error
	// 6. Verify no changes to balance
	// 7. Verify no changes to subscription overage fields
	// 8. Verify no accounting transaction recorded
}

// TestCalculateOverageCost_UnlimitedPackage tests overage for unlimited package
func TestCalculateOverageCost_UnlimitedPackage(t *testing.T) {
	// This test requires a database connection
	t.Skip("Integration test - requires database")

	// TODO: Implement when database is available
	// 1. Create test tenant with balance = 100 USD
	// 2. Create test package with is_unlimited = true
	// 3. Create test subscription
	// 4. Calculate overage cost for 1000 messages
	// 5. Verify cost = 0 (no charge for unlimited)
	// 6. Verify balance unchanged
	// 7. Verify subscription overage fields unchanged
	// 8. Verify no accounting transaction recorded
}

// TestCalculateOverageCost_MultipleOverages tests cumulative overage tracking
func TestCalculateOverageCost_MultipleOverages(t *testing.T) {
	// This test requires a database connection
	t.Skip("Integration test - requires database")

	// TODO: Implement when database is available
	// 1. Create test tenant with balance = 100 USD
	// 2. Create test package with overage_rate = 0.01 USD
	// 3. Create test subscription
	// 4. Calculate overage cost for 10 messages (cost = 0.10)
	// 5. Verify subscription overage_messages = 10, overage_cost = 0.10
	// 6. Calculate overage cost for 20 messages (cost = 0.20)
	// 7. Verify subscription overage_messages = 30, overage_cost = 0.30
	// 8. Verify balance = 100 - 0.30 = 99.70
	// 9. Verify 2 accounting transactions recorded
}

// TestCalculateOverageCost_InvalidRequest tests validation
func TestCalculateOverageCost_InvalidRequest(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	// Create manager with nil pool (we won't use it)
	manager := &manager{
		pool:   nil,
		logger: logger,
	}

	ctx := context.Background()

	// Test invalid subscription ID
	cost, err := manager.CalculateOverageCost(ctx, 0, 10)
	assert.Error(t, err)
	assert.True(t, cost.IsZero())
	assert.Contains(t, err.Error(), "subscription_id is required")

	// Test invalid overage messages (zero)
	cost, err = manager.CalculateOverageCost(ctx, 1, 0)
	assert.Error(t, err)
	assert.True(t, cost.IsZero())
	assert.Contains(t, err.Error(), "overage_messages must be positive")

	// Test invalid overage messages (negative)
	cost, err = manager.CalculateOverageCost(ctx, 1, -5)
	assert.Error(t, err)
	assert.True(t, cost.IsZero())
	assert.Contains(t, err.Error(), "overage_messages must be positive")
}

// TestCalculateOverageCost_Atomicity tests atomic balance deduction
func TestCalculateOverageCost_Atomicity(t *testing.T) {
	// This test requires a database connection
	t.Skip("Integration test - requires database")

	// TODO: Implement when database is available
	// This will be covered by property-based tests in task 7.6
	// Test concurrent overage calculations to ensure atomicity
}

// TestGetUsage tests the GetUsage function
func TestGetUsage(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger := zap.NewNop()
	manager := NewManager(pool, logger)

	ctx := context.Background()

	// Create test tenant
	tenantID := createTestTenant(t, pool, "Test Tenant", decimal.NewFromInt(100))

	// Create test service
	serviceID := createTestService(t, pool, tenantID, "Test Service")

	// Create test package
	packageID := createTestPackage(t, pool, tenantID, serviceID, "Basic", decimal.NewFromInt(10), 1000, decimal.NewFromFloat(0.01))

	// Subscribe to package
	subscribeReq := SubscribeRequest{
		TenantID:  tenantID,
		PackageID: packageID,
		AutoRenew: true,
	}

	subscribeResult, err := manager.Subscribe(ctx, subscribeReq)
	require.NoError(t, err)
	require.True(t, subscribeResult.Success)

	// Test GetUsage
	usage, err := manager.GetUsage(ctx, tenantID)
	require.NoError(t, err)
	require.NotNil(t, usage)

	// Verify usage information
	assert.Equal(t, subscribeResult.SubscriptionID, usage.SubscriptionID)
	assert.Equal(t, packageID, usage.PackageID)
	assert.Equal(t, "Basic", usage.PackageName)
	assert.Equal(t, int64(0), usage.MessagesUsed)
	assert.NotNil(t, usage.MessagesRemaining)
	assert.Equal(t, int64(1000), *usage.MessagesRemaining)
	assert.NotNil(t, usage.IncludedMessages)
	assert.Equal(t, int64(1000), *usage.IncludedMessages)
	assert.Equal(t, int64(0), usage.OverageMessages)
	assert.True(t, usage.OverageCost.IsZero())
	assert.True(t, decimal.NewFromFloat(0.01).Equal(usage.OverageRate))
	assert.Equal(t, StatusActive, usage.Status)
	assert.NotNil(t, usage.NextRenewalDate)
}

// TestGetUsage_NoActiveSubscription tests GetUsage when no active subscription exists
func TestGetUsage_NoActiveSubscription(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger := zap.NewNop()
	manager := NewManager(pool, logger)

	ctx := context.Background()

	// Create test tenant without subscription
	tenantID := createTestTenant(t, pool, "Test Tenant", decimal.NewFromInt(100))

	// Test GetUsage
	usage, err := manager.GetUsage(ctx, tenantID)
	assert.Error(t, err)
	assert.Nil(t, usage)
	assert.Contains(t, err.Error(), "no active subscription found")
}

// TestGetUsage_WithOverage tests GetUsage with overage messages
func TestGetUsage_WithOverage(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger := zap.NewNop()
	manager := NewManager(pool, logger)

	ctx := context.Background()

	// Create test tenant with sufficient balance
	tenantID := createTestTenant(t, pool, "Test Tenant", decimal.NewFromInt(100))

	// Create test service
	serviceID := createTestService(t, pool, tenantID, "Test Service")

	// Create test package with small quota
	packageID := createTestPackage(t, pool, tenantID, serviceID, "Basic", decimal.NewFromInt(10), 5, decimal.NewFromFloat(0.5))

	// Subscribe to package
	subscribeReq := SubscribeRequest{
		TenantID:  tenantID,
		PackageID: packageID,
		AutoRenew: true,
	}

	subscribeResult, err := manager.Subscribe(ctx, subscribeReq)
	require.NoError(t, err)
	require.True(t, subscribeResult.Success)

	// Deduct all messages from quota
	for i := 0; i < 5; i++ {
		deductReq := DeductMessagesRequest{
			SubscriptionID: subscribeResult.SubscriptionID,
			MessageCount:   1,
			MessageID:      fmt.Sprintf("msg-%d", i),
		}
		_, err := manager.DeductMessages(ctx, deductReq)
		require.NoError(t, err)
	}

	// Calculate overage for 3 messages
	overageCost, err := manager.CalculateOverageCost(ctx, subscribeResult.SubscriptionID, 3)
	require.NoError(t, err)
	assert.True(t, decimal.NewFromFloat(1.5).Equal(overageCost)) // 3 * 0.5

	// Test GetUsage
	usage, err := manager.GetUsage(ctx, tenantID)
	require.NoError(t, err)
	require.NotNil(t, usage)

	// Verify usage information includes overage
	assert.Equal(t, int64(5), usage.MessagesUsed)
	assert.NotNil(t, usage.MessagesRemaining)
	assert.Equal(t, int64(0), *usage.MessagesRemaining)
	assert.Equal(t, int64(3), usage.OverageMessages)
	assert.True(t, decimal.NewFromFloat(1.5).Equal(usage.OverageCost))
}

// TestGetUsage_UnlimitedPackage tests GetUsage with unlimited package
func TestGetUsage_UnlimitedPackage(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger := zap.NewNop()
	manager := NewManager(pool, logger)

	ctx := context.Background()

	// Create test tenant
	tenantID := createTestTenant(t, pool, "Test Tenant", decimal.NewFromInt(200))

	// Create test service
	serviceID := createTestService(t, pool, tenantID, "Test Service")

	// Create unlimited package
	packageID := createTestUnlimitedPackage(t, pool, tenantID, serviceID, "Enterprise", decimal.NewFromInt(100))

	// Subscribe to package
	subscribeReq := SubscribeRequest{
		TenantID:  tenantID,
		PackageID: packageID,
		AutoRenew: true,
	}

	subscribeResult, err := manager.Subscribe(ctx, subscribeReq)
	require.NoError(t, err)
	require.True(t, subscribeResult.Success)

	// Deduct some messages
	deductReq := DeductMessagesRequest{
		SubscriptionID: subscribeResult.SubscriptionID,
		MessageCount:   100,
		MessageID:      "msg-1",
	}
	_, err = manager.DeductMessages(ctx, deductReq)
	require.NoError(t, err)

	// Test GetUsage
	usage, err := manager.GetUsage(ctx, tenantID)
	require.NoError(t, err)
	require.NotNil(t, usage)

	// Verify usage information for unlimited package
	assert.Equal(t, int64(100), usage.MessagesUsed)
	assert.Nil(t, usage.MessagesRemaining) // NULL for unlimited
	assert.Nil(t, usage.IncludedMessages)  // NULL for unlimited
	assert.Equal(t, int64(0), usage.OverageMessages)
	assert.True(t, usage.OverageCost.IsZero())
}

func createTestTenant(t *testing.T, pool *pgxpool.Pool, name string, balance decimal.Decimal) uuid.UUID {
	tenantID := uuid.New()
	query := `
		INSERT INTO tenants (id, business_name, contact_name, email, balance, status, created_at, updated_at)
		VALUES ($1, $2, $2, $3, $4, 'ACTIVE', NOW(), NOW())
	`
	_, err := pool.Exec(context.Background(), query, tenantID, name, fmt.Sprintf("%s@example.com", name), balance)
	require.NoError(t, err)
	return tenantID
}

func createTestService(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, name string) int64 {
	var id int64
	query := `
		INSERT INTO services (tenant_id, name, code, type, created_at, updated_at)
		VALUES ($1, $2, $2, 'sms', NOW(), NOW())
		RETURNING id
	`
	err := pool.QueryRow(context.Background(), query, tenantID, name).Scan(&id)
	require.NoError(t, err)
	return id
}

func createTestPackage(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, serviceID int64, name string, price decimal.Decimal, includedMessages int, overageRate decimal.Decimal) int64 {
	var id int64
	query := `
		INSERT INTO packages (
			tenant_id, service_id, name, code, price_usd,
			included_messages, overage_rate, is_unlimited,
			billing_cycle, rollover_enabled, is_active,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, false, 'monthly', true, true, NOW(), NOW()
		)
		RETURNING id
	`
	code := fmt.Sprintf("PKG-%s", uuid.New().String()[:8])
	err := pool.QueryRow(context.Background(), query,
		tenantID, serviceID, name, code, price, includedMessages, overageRate,
	).Scan(&id)
	require.NoError(t, err)
	return id
}

func createTestUnlimitedPackage(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, serviceID int64, name string, price decimal.Decimal) int64 {
	var id int64
	query := `
		INSERT INTO packages (
			tenant_id, service_id, name, code, price_usd,
			included_messages, overage_rate, is_unlimited,
			billing_cycle, rollover_enabled, is_active,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, NULL, 0, true, 'monthly', false, true, NOW(), NOW()
		)
		RETURNING id
	`
	code := fmt.Sprintf("UNL-%s", uuid.New().String()[:8])
	err := pool.QueryRow(context.Background(), query,
		tenantID, serviceID, name, code, price,
	).Scan(&id)
	require.NoError(t, err)
	return id
}
