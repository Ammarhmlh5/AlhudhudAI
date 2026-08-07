package subscription

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"pgregory.net/rapid"
)

// Property 5: الاشتراك الآمن في الباقة
// Validates: Requirements 4.1, 4.2
//
// This property verifies that:
// 1. Subscription succeeds only when balance is sufficient
// 2. Balance is deducted atomically
// 3. Subscription is created correctly
// 4. Accounting transaction is recorded
// 5. No race conditions occur during concurrent subscriptions

func TestProperty_SafeSubscription(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	// Setup test database connection
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	manager := NewManager(pool, logger)

	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		tenantID := uuid.New()
		initialBalance := rapid.IntRange(0, 200).Draw(t, "initial_balance")
		packagePrice := rapid.IntRange(10, 100).Draw(t, "package_price")

		// Setup: Create tenant with initial balance
		setupTenant(t, pool, tenantID, decimal.NewFromInt(int64(initialBalance)))

		// Setup: Create test package
		packageID := setupTestPackage(t, pool, tenantID, decimal.NewFromInt(int64(packagePrice)))

		// Execute: Subscribe to package
		result, err := manager.Subscribe(context.Background(), SubscribeRequest{
			TenantID:  tenantID,
			PackageID: packageID,
			AutoRenew: true,
		})

		// Verify: Check result based on balance
		if initialBalance >= packagePrice {
			// Should succeed
			require.NoError(t, err, "Subscribe should succeed with sufficient balance")
			require.NotNil(t, result, "Result should not be nil")
			assert.True(t, result.Success, "Subscription should be successful")
			assert.Greater(t, result.SubscriptionID, int64(0), "Subscription ID should be positive")

			// Verify balance was deducted correctly
			expectedBalance := decimal.NewFromInt(int64(initialBalance - packagePrice))
			assert.True(t, result.NewBalance.Equal(expectedBalance),
				"Balance should be deducted correctly: expected %s, got %s",
				expectedBalance, result.NewBalance)

			// Verify subscription was created
			sub, err := manager.GetActiveSubscription(context.Background(), tenantID)
			require.NoError(t, err, "Should be able to get active subscription")
			assert.Equal(t, result.SubscriptionID, sub.ID, "Subscription ID should match")
			assert.Equal(t, packageID, sub.PackageID, "Package ID should match")
			assert.Equal(t, StatusActive, sub.Status, "Status should be active")

		} else {
			// Should fail with insufficient balance
			require.NoError(t, err, "Subscribe should not return error for insufficient balance")
			require.NotNil(t, result, "Result should not be nil")
			assert.False(t, result.Success, "Subscription should fail with insufficient balance")
			assert.Equal(t, "Insufficient balance", result.Message, "Should indicate insufficient balance")

			// Verify balance was not changed
			assert.True(t, result.NewBalance.Equal(result.OldBalance),
				"Balance should not change on failed subscription")
		}

		// Cleanup
		cleanupTenant(t, pool, tenantID)
	})
}

// TestProperty_AtomicSubscription tests that subscription is atomic
// Either all operations succeed or all fail
func TestProperty_AtomicSubscription(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	manager := NewManager(pool, logger)

	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		initialBalance := decimal.NewFromInt(100)
		packagePrice := decimal.NewFromInt(50)

		// Setup
		setupTenant(t, pool, tenantID, initialBalance)
		packageID := setupTestPackage(t, pool, tenantID, packagePrice)

		// Get balance before
		balanceBefore := getTenantBalance(t, pool, tenantID)

		// Subscribe
		result, err := manager.Subscribe(context.Background(), SubscribeRequest{
			TenantID:  tenantID,
			PackageID: packageID,
			AutoRenew: true,
		})

		require.NoError(t, err)
		require.True(t, result.Success)

		// Get balance after
		balanceAfter := getTenantBalance(t, pool, tenantID)

		// Verify atomicity: balance should be exactly initial - price
		expectedBalance := balanceBefore.Sub(packagePrice)
		assert.True(t, balanceAfter.Equal(expectedBalance),
			"Balance should be exactly initial - price (atomic operation)")

		// Verify subscription exists
		sub, err := manager.GetActiveSubscription(context.Background(), tenantID)
		require.NoError(t, err)
		assert.Equal(t, result.SubscriptionID, sub.ID)

		// Verify accounting transaction exists
		txExists := checkAccountingTransaction(t, pool, tenantID, "SUBSCRIPTION")
		assert.True(t, txExists, "Accounting transaction should exist")

		// Cleanup
		cleanupTenant(t, pool, tenantID)
	})
}

// TestProperty_ConcurrentSubscriptions tests concurrent subscription attempts
// Verifies no race conditions occur
func TestProperty_ConcurrentSubscriptions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	manager := NewManager(pool, logger)

	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		initialBalance := decimal.NewFromInt(100)
		packagePrice := decimal.NewFromInt(10)
		concurrency := rapid.IntRange(2, 5).Draw(t, "concurrency")

		// Setup
		setupTenant(t, pool, tenantID, initialBalance)
		packageID := setupTestPackage(t, pool, tenantID, packagePrice)

		// Execute concurrent subscriptions
		var wg sync.WaitGroup
		results := make([]*SubscribeResult, concurrency)
		errors := make([]error, concurrency)

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				result, err := manager.Subscribe(context.Background(), SubscribeRequest{
					TenantID:  tenantID,
					PackageID: packageID,
					AutoRenew: true,
				})
				results[index] = result
				errors[index] = err
			}(i)
		}

		wg.Wait()

		// Count successful subscriptions
		successCount := 0
		for i := 0; i < concurrency; i++ {
			if errors[i] == nil && results[i] != nil && results[i].Success {
				successCount++
			}
		}

		// Verify: Number of successful subscriptions should not exceed what balance allows
		maxPossibleSubscriptions := int(initialBalance.Div(packagePrice).IntPart())
		assert.LessOrEqual(t, successCount, maxPossibleSubscriptions,
			"Successful subscriptions should not exceed what balance allows")

		// Verify final balance
		finalBalance := getTenantBalance(t, pool, tenantID)
		expectedBalance := initialBalance.Sub(packagePrice.Mul(decimal.NewFromInt(int64(successCount))))
		assert.True(t, finalBalance.Equal(expectedBalance),
			"Final balance should match expected after concurrent subscriptions")

		// Cleanup
		cleanupTenant(t, pool, tenantID)
	})
}

// TestProperty_SubscriptionIdempotency tests that multiple subscription attempts
// with same parameters don't cause issues
func TestProperty_SubscriptionIdempotency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	manager := NewManager(pool, logger)

	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		initialBalance := decimal.NewFromInt(100)
		packagePrice := decimal.NewFromInt(10)

		// Setup
		setupTenant(t, pool, tenantID, initialBalance)
		packageID := setupTestPackage(t, pool, tenantID, packagePrice)

		// First subscription
		result1, err1 := manager.Subscribe(context.Background(), SubscribeRequest{
			TenantID:  tenantID,
			PackageID: packageID,
			AutoRenew: true,
		})

		require.NoError(t, err1)
		require.True(t, result1.Success)

		// Second subscription (should also succeed but create new subscription)
		result2, err2 := manager.Subscribe(context.Background(), SubscribeRequest{
			TenantID:  tenantID,
			PackageID: packageID,
			AutoRenew: true,
		})

		require.NoError(t, err2)
		require.True(t, result2.Success)

		balanceAfterSecond := getTenantBalance(t, pool, tenantID)

		// Verify both subscriptions were created
		assert.NotEqual(t, result1.SubscriptionID, result2.SubscriptionID,
			"Each subscription should have unique ID")

		// Verify balance was deducted twice
		expectedBalance := initialBalance.Sub(packagePrice.Mul(decimal.NewFromInt(2)))
		assert.True(t, balanceAfterSecond.Equal(expectedBalance),
			"Balance should be deducted for each subscription")

		// Cleanup
		cleanupTenant(t, pool, tenantID)
	})
}

// Helper functions

func setupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	// This should connect to a test database
	// For now, we'll skip if no test database is available
	connString := "postgres://postgres@localhost:5432/alhudhud_test?sslmode=disable"
	
	pool, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		t.Skip("Test database not available")
	}

	// Test connection
	if err := pool.Ping(context.Background()); err != nil {
		t.Skip("Test database not available")
	}

	cleanup := func() {
		pool.Close()
	}

	return pool, cleanup
}

func setupTenant(t *rapid.T, pool *pgxpool.Pool, tenantID uuid.UUID, balance decimal.Decimal) {
	query := `
		INSERT INTO tenants (id, business_name, contact_name, email, balance, created_at, updated_at)
		VALUES ($1, $2, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET balance = $4
	`
	_, err := pool.Exec(context.Background(), query,
		tenantID,
		fmt.Sprintf("Test Tenant %s", tenantID.String()[:8]),
		fmt.Sprintf("test-%s@example.com", tenantID.String()[:8]),
		balance,
	)
	if err != nil {
		t.Fatalf("Failed to setup tenant: %v", err)
	}
}

func setupTestPackage(t *rapid.T, pool *pgxpool.Pool, tenantID uuid.UUID, price decimal.Decimal) int64 {
	// Set tenant context for RLS
	_, err := pool.Exec(context.Background(), "SELECT set_config('app.current_tenant', $1, false)", tenantID.String())
	if err != nil {
		t.Fatalf("Failed to set tenant context: %v", err)
	}

	query := `
		INSERT INTO packages (
			tenant_id, service_id, name, code, price_usd,
			included_messages, overage_rate, is_unlimited,
			billing_cycle, rollover_enabled, is_active
		) VALUES (
			$1, 2, 'Test Package', $2, $3, 1000, 0.01, false, 'monthly', true, true
		)
		RETURNING id
	`

	var packageID int64
	code := fmt.Sprintf("TEST_%s", uuid.New().String()[:8])
	err = pool.QueryRow(context.Background(), query, tenantID, code, price).Scan(&packageID)
	if err != nil {
		t.Fatalf("Failed to setup package: %v", err)
	}

	return packageID
}

func getTenantBalance(t *rapid.T, pool *pgxpool.Pool, tenantID uuid.UUID) decimal.Decimal {
	var balance decimal.Decimal
	query := `SELECT balance FROM tenants WHERE id = $1`
	err := pool.QueryRow(context.Background(), query, tenantID).Scan(&balance)
	if err != nil {
		t.Fatalf("Failed to get tenant balance: %v", err)
	}
	return balance
}

func checkAccountingTransaction(t *rapid.T, pool *pgxpool.Pool, tenantID uuid.UUID, txType string) bool {
	var exists bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM accounting_transactions 
			WHERE tenant_id = $1 AND type = $2
		)
	`
	err := pool.QueryRow(context.Background(), query, tenantID, txType).Scan(&exists)
	if err != nil {
		t.Fatalf("Failed to check accounting transaction: %v", err)
	}
	return exists
}

func cleanupTenant(t *rapid.T, pool *pgxpool.Pool, tenantID uuid.UUID) {
	// Delete in reverse order of foreign keys
	queries := []string{
		`DELETE FROM accounting_transactions WHERE tenant_id = $1`,
		`DELETE FROM subscriptions WHERE tenant_id = $1`,
		`DELETE FROM packages WHERE tenant_id = $1`,
		`DELETE FROM tenants WHERE id = $1`,
	}

	for _, query := range queries {
		_, err := pool.Exec(context.Background(), query, tenantID)
		if err != nil {
			// Log but don't fail - cleanup is best effort
			t.Logf("Cleanup warning: %v", err)
		}
	}
}

// Property 6: خصم من عداد الرسائل عند الإرسال
// Validates: Requirements 4.5
//
// This property verifies that:
// 1. messages_remaining decreases correctly when messages are deducted
// 2. messages_used increases correctly when messages are deducted
// 3. The sum of messages_used + messages_remaining equals the original quota (for limited packages)
// 4. Concurrent deductions don't cause race conditions
// 5. Deduction is atomic (either succeeds completely or fails completely)
// 6. Unlimited packages handle deductions correctly (NULL remaining)

func TestProperty_MessageDeduction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	manager := NewManager(pool, logger)

	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		tenantID := uuid.New()
		initialBalance := decimal.NewFromInt(1000)
		packagePrice := decimal.NewFromInt(50)
		includedMessages := rapid.Int64Range(100, 1000).Draw(t, "included_messages")
		messageCount := rapid.Int64Range(1, 50).Draw(t, "message_count")

		// Setup: Create tenant and package
		setupTenant(t, pool, tenantID, initialBalance)
		packageID := setupTestPackageWithMessages(t, pool, tenantID, packagePrice, includedMessages)

		// Subscribe to package
		subResult, err := manager.Subscribe(context.Background(), SubscribeRequest{
			TenantID:  tenantID,
			PackageID: packageID,
			AutoRenew: true,
		})
		require.NoError(t, err)
		require.True(t, subResult.Success)

		// Get initial state
		sub, err := manager.GetActiveSubscription(context.Background(), tenantID)
		require.NoError(t, err)
		require.NotNil(t, sub.MessagesRemaining)

		initialUsed := sub.MessagesUsed
		initialRemaining := *sub.MessagesRemaining

		// Execute: Deduct messages
		result, err := manager.DeductMessages(context.Background(), DeductMessagesRequest{
			SubscriptionID: subResult.SubscriptionID,
			MessageCount:   messageCount,
			MessageID:      uuid.New().String(),
		})

		// Verify based on available messages
		if initialRemaining >= messageCount {
			// Should succeed
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.True(t, result.Success, "Deduction should succeed with sufficient messages")
			assert.Equal(t, messageCount, result.MessagesDeducted, "Should deduct requested count")
			assert.Equal(t, int64(0), result.OverageMessages, "Should have no overage")

			// Get updated state
			subAfter, err := manager.GetActiveSubscription(context.Background(), tenantID)
			require.NoError(t, err)
			require.NotNil(t, subAfter.MessagesRemaining)

			// Property 1: messages_remaining decreases correctly
			expectedRemaining := initialRemaining - messageCount
			assert.Equal(t, expectedRemaining, *subAfter.MessagesRemaining,
				"messages_remaining should decrease by message_count")

			// Property 2: messages_used increases correctly
			expectedUsed := initialUsed + messageCount
			assert.Equal(t, expectedUsed, subAfter.MessagesUsed,
				"messages_used should increase by message_count")

			// Property 3: Invariant - sum equals original quota
			totalMessages := subAfter.MessagesUsed + *subAfter.MessagesRemaining
			assert.Equal(t, includedMessages, totalMessages,
				"messages_used + messages_remaining should equal original quota")

		} else {
			// Should fail with insufficient messages
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.False(t, result.Success, "Deduction should fail with insufficient messages")
			assert.Equal(t, int64(0), result.MessagesDeducted, "Should not deduct any messages")
			assert.Greater(t, result.OverageMessages, int64(0), "Should indicate overage")

			// Verify state unchanged
			subAfter, err := manager.GetActiveSubscription(context.Background(), tenantID)
			require.NoError(t, err)
			assert.Equal(t, initialUsed, subAfter.MessagesUsed, "messages_used should not change")
			assert.Equal(t, initialRemaining, *subAfter.MessagesRemaining, "messages_remaining should not change")
		}

		// Cleanup
		cleanupTenant(t, pool, tenantID)
	})
}

// TestProperty_ConcurrentMessageDeduction tests concurrent message deductions
// Verifies no race conditions occur
func TestProperty_ConcurrentMessageDeduction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	manager := NewManager(pool, logger)

	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		initialBalance := decimal.NewFromInt(1000)
		packagePrice := decimal.NewFromInt(50)
		includedMessages := int64(100)
		messageCount := int64(5)
		concurrency := rapid.IntRange(5, 15).Draw(t, "concurrency")

		// Setup
		setupTenant(t, pool, tenantID, initialBalance)
		packageID := setupTestPackageWithMessages(t, pool, tenantID, packagePrice, includedMessages)

		// Subscribe
		subResult, err := manager.Subscribe(context.Background(), SubscribeRequest{
			TenantID:  tenantID,
			PackageID: packageID,
			AutoRenew: true,
		})
		require.NoError(t, err)
		require.True(t, subResult.Success)

		// Execute concurrent deductions
		var wg sync.WaitGroup
		results := make([]*DeductMessagesResult, concurrency)
		errors := make([]error, concurrency)

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				result, err := manager.DeductMessages(context.Background(), DeductMessagesRequest{
					SubscriptionID: subResult.SubscriptionID,
					MessageCount:   messageCount,
					MessageID:      uuid.New().String(),
				})
				results[index] = result
				errors[index] = err
			}(i)
		}

		wg.Wait()

		// Count successful deductions
		successCount := 0
		for i := 0; i < concurrency; i++ {
			if errors[i] == nil && results[i] != nil && results[i].Success {
				successCount++
			}
		}

		// Verify: Number of successful deductions should not exceed quota
		maxPossibleDeductions := int(includedMessages / messageCount)
		assert.LessOrEqual(t, successCount, maxPossibleDeductions,
			"Successful deductions should not exceed quota allows")

		// Verify final state
		subAfter, err := manager.GetActiveSubscription(context.Background(), tenantID)
		require.NoError(t, err)

		// Property: Total deducted should match successful operations
		expectedUsed := int64(successCount) * messageCount
		assert.Equal(t, expectedUsed, subAfter.MessagesUsed,
			"messages_used should equal successful deductions * message_count")

		// Property: Invariant holds
		if subAfter.MessagesRemaining != nil {
			totalMessages := subAfter.MessagesUsed + *subAfter.MessagesRemaining
			assert.Equal(t, includedMessages, totalMessages,
				"Invariant should hold after concurrent deductions")
		}

		// Cleanup
		cleanupTenant(t, pool, tenantID)
	})
}

// TestProperty_UnlimitedPackageDeduction tests message deduction for unlimited packages
func TestProperty_UnlimitedPackageDeduction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	manager := NewManager(pool, logger)

	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		initialBalance := decimal.NewFromInt(1000)
		packagePrice := decimal.NewFromInt(100)
		messageCount := rapid.Int64Range(1, 100).Draw(t, "message_count")

		// Setup: Create unlimited package
		setupTenant(t, pool, tenantID, initialBalance)
		packageID := setupUnlimitedPackage(t, pool, tenantID, packagePrice)

		// Subscribe
		subResult, err := manager.Subscribe(context.Background(), SubscribeRequest{
			TenantID:  tenantID,
			PackageID: packageID,
			AutoRenew: true,
		})
		require.NoError(t, err)
		require.True(t, subResult.Success)

		// Get initial state
		sub, err := manager.GetActiveSubscription(context.Background(), tenantID)
		require.NoError(t, err)
		assert.Nil(t, sub.MessagesRemaining, "Unlimited package should have NULL messages_remaining")

		initialUsed := sub.MessagesUsed

		// Execute: Deduct messages
		result, err := manager.DeductMessages(context.Background(), DeductMessagesRequest{
			SubscriptionID: subResult.SubscriptionID,
			MessageCount:   messageCount,
			MessageID:      uuid.New().String(),
		})

		// Verify: Should always succeed for unlimited packages
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Success, "Deduction should always succeed for unlimited packages")
		assert.Equal(t, messageCount, result.MessagesDeducted, "Should deduct requested count")
		assert.Nil(t, result.MessagesRemaining, "Should return NULL for unlimited packages")

		// Get updated state
		subAfter, err := manager.GetActiveSubscription(context.Background(), tenantID)
		require.NoError(t, err)

		// Property: messages_used increases correctly
		expectedUsed := initialUsed + messageCount
		assert.Equal(t, expectedUsed, subAfter.MessagesUsed,
			"messages_used should increase by message_count")

		// Property: messages_remaining stays NULL
		assert.Nil(t, subAfter.MessagesRemaining,
			"messages_remaining should stay NULL for unlimited packages")

		// Cleanup
		cleanupTenant(t, pool, tenantID)
	})
}

// TestProperty_MessageDeductionAtomicity tests atomicity of message deduction
func TestProperty_MessageDeductionAtomicity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	manager := NewManager(pool, logger)

	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		initialBalance := decimal.NewFromInt(1000)
		packagePrice := decimal.NewFromInt(50)
		includedMessages := int64(100)
		messageCount := rapid.Int64Range(1, 20).Draw(t, "message_count")

		// Setup
		setupTenant(t, pool, tenantID, initialBalance)
		packageID := setupTestPackageWithMessages(t, pool, tenantID, packagePrice, includedMessages)

		// Subscribe
		subResult, err := manager.Subscribe(context.Background(), SubscribeRequest{
			TenantID:  tenantID,
			PackageID: packageID,
			AutoRenew: true,
		})
		require.NoError(t, err)
		require.True(t, subResult.Success)

		// Get state before
		subBefore, err := manager.GetActiveSubscription(context.Background(), tenantID)
		require.NoError(t, err)

		// Execute deduction
		result, err := manager.DeductMessages(context.Background(), DeductMessagesRequest{
			SubscriptionID: subResult.SubscriptionID,
			MessageCount:   messageCount,
			MessageID:      uuid.New().String(),
		})

		require.NoError(t, err)
		require.NotNil(t, result)

		// Get state after
		subAfter, err := manager.GetActiveSubscription(context.Background(), tenantID)
		require.NoError(t, err)

		if result.Success {
			// Property: Atomic update - both fields change together
			usedDelta := subAfter.MessagesUsed - subBefore.MessagesUsed
			remainingDelta := *subBefore.MessagesRemaining - *subAfter.MessagesRemaining

			assert.Equal(t, messageCount, usedDelta,
				"messages_used should increase by exactly message_count")
			assert.Equal(t, messageCount, remainingDelta,
				"messages_remaining should decrease by exactly message_count")
			assert.Equal(t, usedDelta, remainingDelta,
				"Increase in used should equal decrease in remaining (atomicity)")

		} else {
			// Property: On failure, no changes should occur
			assert.Equal(t, subBefore.MessagesUsed, subAfter.MessagesUsed,
				"messages_used should not change on failure")
			assert.Equal(t, *subBefore.MessagesRemaining, *subAfter.MessagesRemaining,
				"messages_remaining should not change on failure")
		}

		// Cleanup
		cleanupTenant(t, pool, tenantID)
	})
}

// TestProperty_MessageDeductionInvariant tests that the invariant always holds
func TestProperty_MessageDeductionInvariant(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	manager := NewManager(pool, logger)

	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		initialBalance := decimal.NewFromInt(1000)
		packagePrice := decimal.NewFromInt(50)
		includedMessages := rapid.Int64Range(50, 200).Draw(t, "included_messages")
		
		// Generate multiple deduction operations
		numOperations := rapid.IntRange(3, 10).Draw(t, "num_operations")
		messageCounts := make([]int64, numOperations)
		for i := 0; i < numOperations; i++ {
			messageCounts[i] = rapid.Int64Range(1, 10).Draw(t, fmt.Sprintf("message_count_%d", i))
		}

		// Setup
		setupTenant(t, pool, tenantID, initialBalance)
		packageID := setupTestPackageWithMessages(t, pool, tenantID, packagePrice, includedMessages)

		// Subscribe
		subResult, err := manager.Subscribe(context.Background(), SubscribeRequest{
			TenantID:  tenantID,
			PackageID: packageID,
			AutoRenew: true,
		})
		require.NoError(t, err)
		require.True(t, subResult.Success)

		// Execute multiple deductions
		for i, msgCount := range messageCounts {
			result, err := manager.DeductMessages(context.Background(), DeductMessagesRequest{
				SubscriptionID: subResult.SubscriptionID,
				MessageCount:   msgCount,
				MessageID:      fmt.Sprintf("msg-%d-%s", i, uuid.New().String()),
			})

			require.NoError(t, err)
			require.NotNil(t, result)

			// After each operation, verify invariant
			sub, err := manager.GetActiveSubscription(context.Background(), tenantID)
			require.NoError(t, err)

			if sub.MessagesRemaining != nil {
				// Property: Invariant must hold after every operation
				totalMessages := sub.MessagesUsed + *sub.MessagesRemaining
				assert.Equal(t, includedMessages, totalMessages,
					"Invariant (used + remaining = quota) must hold after operation %d", i)
			}

			// If deduction failed, stop trying
			if !result.Success {
				break
			}
		}

		// Cleanup
		cleanupTenant(t, pool, tenantID)
	})
}

// Helper function to setup package with specific message count
func setupTestPackageWithMessages(t *rapid.T, pool *pgxpool.Pool, tenantID uuid.UUID, price decimal.Decimal, messages int64) int64 {
	// Set tenant context for RLS
	_, err := pool.Exec(context.Background(), "SELECT set_config('app.current_tenant', $1, false)", tenantID.String())
	if err != nil {
		t.Fatalf("Failed to set tenant context: %v", err)
	}

	query := `
		INSERT INTO packages (
			tenant_id, service_id, name, code, price_usd,
			included_messages, overage_rate, is_unlimited,
			billing_cycle, rollover_enabled, is_active
		) VALUES (
			$1, 2, 'Test Package', $2, $3, $4, 0.01, false, 'monthly', true, true
		)
		RETURNING id
	`

	var packageID int64
	code := fmt.Sprintf("TEST_%s", uuid.New().String()[:8])
	err = pool.QueryRow(context.Background(), query, tenantID, code, price, messages).Scan(&packageID)
	if err != nil {
		t.Fatalf("Failed to setup package: %v", err)
	}

	return packageID
}

// Helper function to setup unlimited package
func setupUnlimitedPackage(t *rapid.T, pool *pgxpool.Pool, tenantID uuid.UUID, price decimal.Decimal) int64 {
	// Set tenant context for RLS
	_, err := pool.Exec(context.Background(), "SELECT set_config('app.current_tenant', $1, false)", tenantID.String())
	if err != nil {
		t.Fatalf("Failed to set tenant context: %v", err)
	}

	query := `
		INSERT INTO packages (
			tenant_id, service_id, name, code, price_usd,
			included_messages, overage_rate, is_unlimited,
			billing_cycle, rollover_enabled, is_active
		) VALUES (
			$1, 2, 'Unlimited Package', $2, $3, NULL, 0, true, 'monthly', false, true
		)
		RETURNING id
	`

	var packageID int64
	code := fmt.Sprintf("UNLIMITED_%s", uuid.New().String()[:8])
	err = pool.QueryRow(context.Background(), query, tenantID, code, price).Scan(&packageID)
	if err != nil {
		t.Fatalf("Failed to setup unlimited package: %v", err)
	}

	return packageID
}

// Property 8: خصم تكلفة الرسائل الإضافية من الرصيد
// Validates: Requirements 5.3, 5.4
//
// This property verifies that:
// 1. Overage cost is calculated correctly using package overage_rate
// 2. Tenant balance is checked before charging
// 3. Balance is deducted atomically when sufficient
// 4. Transaction fails when balance is insufficient
// 5. Subscription overage_messages and overage_cost are updated correctly
// 6. Accounting transaction is recorded
// 7. Unlimited packages don't charge overage

func TestProperty_OverageCharging(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	manager := NewManager(pool, logger)

	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		tenantID := uuid.New()
		initialBalance := rapid.IntRange(50, 100).Draw(t, "initial_balance")
		packagePrice := decimal.NewFromInt(50)
		overageRate := decimal.NewFromFloat(0.05) // $0.05 per message
		overageMessages := rapid.Int64Range(1, 50).Draw(t, "overage_messages")

		// Calculate expected overage cost
		expectedCost := overageRate.Mul(decimal.NewFromInt(overageMessages))

		// Setup: Create tenant and package
		setupTenant(t, pool, tenantID, decimal.NewFromInt(int64(initialBalance)))
		packageID := setupTestPackageWithOverageRate(t, pool, tenantID, packagePrice, overageRate, 100)

		// Subscribe to package
		subResult, err := manager.Subscribe(context.Background(), SubscribeRequest{
			TenantID:  tenantID,
			PackageID: packageID,
			AutoRenew: true,
		})
		require.NoError(t, err)
		require.True(t, subResult.Success)

		// Get balance before overage
		balanceBefore := getTenantBalance(t, pool, tenantID)

		// Execute: Calculate and charge overage
		cost, err := manager.CalculateOverageCost(context.Background(), subResult.SubscriptionID, overageMessages)

		// Verify based on balance
		if balanceBefore.GreaterThanOrEqual(expectedCost) {
			// Should succeed
			require.NoError(t, err, "CalculateOverageCost should succeed with sufficient balance")
			assert.True(t, cost.Equal(expectedCost),
				"Returned cost should match expected: expected %s, got %s",
				expectedCost, cost)

			// Verify balance was deducted
			balanceAfter := getTenantBalance(t, pool, tenantID)
			expectedBalance := balanceBefore.Sub(expectedCost)
			assert.True(t, balanceAfter.Equal(expectedBalance),
				"Balance should be deducted correctly: expected %s, got %s",
				expectedBalance, balanceAfter)

			// Verify subscription was updated
			sub, err := manager.GetActiveSubscription(context.Background(), tenantID)
			require.NoError(t, err)
			assert.Equal(t, overageMessages, sub.OverageMessages,
				"Subscription overage_messages should be updated")
			assert.True(t, sub.OverageCost.Equal(expectedCost),
				"Subscription overage_cost should be updated")

			// Verify accounting transaction exists
			txExists := checkAccountingTransaction(t, pool, tenantID, "OVERAGE")
			assert.True(t, txExists, "Accounting transaction should exist")

		} else {
			// Should fail with insufficient balance
			require.Error(t, err, "CalculateOverageCost should fail with insufficient balance")
			assert.Contains(t, err.Error(), "insufficient balance",
				"Error should indicate insufficient balance")

			// Verify balance was not changed
			balanceAfter := getTenantBalance(t, pool, tenantID)
			assert.True(t, balanceAfter.Equal(balanceBefore),
				"Balance should not change on failed overage charge")

			// Verify subscription was not updated
			sub, err := manager.GetActiveSubscription(context.Background(), tenantID)
			require.NoError(t, err)
			assert.Equal(t, int64(0), sub.OverageMessages,
				"Subscription overage_messages should not be updated on failure")
			assert.True(t, sub.OverageCost.IsZero(),
				"Subscription overage_cost should not be updated on failure")
		}

		// Cleanup
		cleanupTenant(t, pool, tenantID)
	})
}

// TestProperty_MultipleOverageCharges tests multiple overage charges accumulate correctly
func TestProperty_MultipleOverageCharges(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	manager := NewManager(pool, logger)

	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		initialBalance := decimal.NewFromInt(1000)
		packagePrice := decimal.NewFromInt(50)
		overageRate := decimal.NewFromFloat(0.05)
		
		// Generate multiple overage operations
		numOperations := rapid.IntRange(2, 5).Draw(t, "num_operations")
		overageCounts := make([]int64, numOperations)
		for i := 0; i < numOperations; i++ {
			overageCounts[i] = rapid.Int64Range(1, 10).Draw(t, fmt.Sprintf("overage_%d", i))
		}

		// Setup
		setupTenant(t, pool, tenantID, initialBalance)
		packageID := setupTestPackageWithOverageRate(t, pool, tenantID, packagePrice, overageRate, 100)

		// Subscribe
		subResult, err := manager.Subscribe(context.Background(), SubscribeRequest{
			TenantID:  tenantID,
			PackageID: packageID,
			AutoRenew: true,
		})
		require.NoError(t, err)
		require.True(t, subResult.Success)

		// Track cumulative values
		var cumulativeMessages int64
		cumulativeCost := decimal.Zero

		// Execute multiple overage charges
		for i, msgCount := range overageCounts {
			cost, err := manager.CalculateOverageCost(context.Background(), subResult.SubscriptionID, msgCount)
			require.NoError(t, err, "Overage charge %d should succeed", i)

			expectedCost := overageRate.Mul(decimal.NewFromInt(msgCount))
			assert.True(t, cost.Equal(expectedCost),
				"Cost for operation %d should match expected", i)

			cumulativeMessages += msgCount
			cumulativeCost = cumulativeCost.Add(expectedCost)

			// Verify cumulative values in subscription
			sub, err := manager.GetActiveSubscription(context.Background(), tenantID)
			require.NoError(t, err)
			
			assert.Equal(t, cumulativeMessages, sub.OverageMessages,
				"Cumulative overage_messages should match after operation %d", i)
			assert.True(t, sub.OverageCost.Equal(cumulativeCost),
				"Cumulative overage_cost should match after operation %d: expected %s, got %s",
				i, cumulativeCost, sub.OverageCost)
		}

		// Verify final balance
		finalBalance := getTenantBalance(t, pool, tenantID)
		expectedBalance := initialBalance.Sub(packagePrice).Sub(cumulativeCost)
		assert.True(t, finalBalance.Equal(expectedBalance),
			"Final balance should match expected after all operations")

		// Cleanup
		cleanupTenant(t, pool, tenantID)
	})
}

// TestProperty_ConcurrentOverageCharges tests concurrent overage charges
func TestProperty_ConcurrentOverageCharges(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	manager := NewManager(pool, logger)

	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		initialBalance := decimal.NewFromInt(100)
		packagePrice := decimal.NewFromInt(50)
		overageRate := decimal.NewFromFloat(0.05)
		overageMessages := int64(5)
		concurrency := rapid.IntRange(3, 8).Draw(t, "concurrency")

		// Setup
		setupTenant(t, pool, tenantID, initialBalance)
		packageID := setupTestPackageWithOverageRate(t, pool, tenantID, packagePrice, overageRate, 100)

		// Subscribe
		subResult, err := manager.Subscribe(context.Background(), SubscribeRequest{
			TenantID:  tenantID,
			PackageID: packageID,
			AutoRenew: true,
		})
		require.NoError(t, err)
		require.True(t, subResult.Success)

		// Get balance after subscription
		balanceAfterSub := getTenantBalance(t, pool, tenantID)

		// Execute concurrent overage charges
		var wg sync.WaitGroup
		costs := make([]decimal.Decimal, concurrency)
		errors := make([]error, concurrency)

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				cost, err := manager.CalculateOverageCost(context.Background(), subResult.SubscriptionID, overageMessages)
				costs[index] = cost
				errors[index] = err
			}(i)
		}

		wg.Wait()

		// Count successful charges
		successCount := 0
		totalCost := decimal.Zero
		for i := 0; i < concurrency; i++ {
			if errors[i] == nil {
				successCount++
				totalCost = totalCost.Add(costs[i])
			}
		}

		// Verify: Number of successful charges should not exceed what balance allows
		expectedCostPerCharge := overageRate.Mul(decimal.NewFromInt(overageMessages))
		maxPossibleCharges := int(balanceAfterSub.Div(expectedCostPerCharge).IntPart())
		assert.LessOrEqual(t, successCount, maxPossibleCharges,
			"Successful charges should not exceed what balance allows")

		// Verify final balance
		finalBalance := getTenantBalance(t, pool, tenantID)
		expectedBalance := balanceAfterSub.Sub(totalCost)
		assert.True(t, finalBalance.Equal(expectedBalance),
			"Final balance should match expected after concurrent charges")

		// Verify subscription cumulative values
		sub, err := manager.GetActiveSubscription(context.Background(), tenantID)
		require.NoError(t, err)
		
		expectedMessages := int64(successCount) * overageMessages
		assert.Equal(t, expectedMessages, sub.OverageMessages,
			"Cumulative overage_messages should match successful operations")
		assert.True(t, sub.OverageCost.Equal(totalCost),
			"Cumulative overage_cost should match total cost")

		// Cleanup
		cleanupTenant(t, pool, tenantID)
	})
}

// TestProperty_UnlimitedPackageNoOverage tests that unlimited packages don't charge overage
func TestProperty_UnlimitedPackageNoOverage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	manager := NewManager(pool, logger)

	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		initialBalance := decimal.NewFromInt(1000)
		packagePrice := decimal.NewFromInt(100)
		overageMessages := rapid.Int64Range(1, 100).Draw(t, "overage_messages")

		// Setup: Create unlimited package
		setupTenant(t, pool, tenantID, initialBalance)
		packageID := setupUnlimitedPackage(t, pool, tenantID, packagePrice)

		// Subscribe
		subResult, err := manager.Subscribe(context.Background(), SubscribeRequest{
			TenantID:  tenantID,
			PackageID: packageID,
			AutoRenew: true,
		})
		require.NoError(t, err)
		require.True(t, subResult.Success)

		// Get balance before overage attempt
		balanceBefore := getTenantBalance(t, pool, tenantID)

		// Execute: Try to charge overage
		cost, err := manager.CalculateOverageCost(context.Background(), subResult.SubscriptionID, overageMessages)

		// Verify: Should return zero cost with no error (Requirement 5.8)
		require.NoError(t, err, "CalculateOverageCost should not error for unlimited packages")
		assert.True(t, cost.IsZero(),
			"Overage cost should be zero for unlimited packages")

		// Verify balance was not changed
		balanceAfter := getTenantBalance(t, pool, tenantID)
		assert.True(t, balanceAfter.Equal(balanceBefore),
			"Balance should not change for unlimited packages")

		// Verify subscription was not updated
		sub, err := manager.GetActiveSubscription(context.Background(), tenantID)
		require.NoError(t, err)
		assert.Equal(t, int64(0), sub.OverageMessages,
			"Overage messages should not be recorded for unlimited packages")
		assert.True(t, sub.OverageCost.IsZero(),
			"Overage cost should remain zero for unlimited packages")

		// Cleanup
		cleanupTenant(t, pool, tenantID)
	})
}

// TestProperty_OverageAtomicity tests atomicity of overage charging
func TestProperty_OverageAtomicity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	manager := NewManager(pool, logger)

	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		initialBalance := rapid.IntRange(50, 100).Draw(t, "initial_balance")
		packagePrice := decimal.NewFromInt(50)
		overageRate := decimal.NewFromFloat(0.05)
		overageMessages := rapid.Int64Range(1, 20).Draw(t, "overage_messages")

		// Setup
		setupTenant(t, pool, tenantID, decimal.NewFromInt(int64(initialBalance)))
		packageID := setupTestPackageWithOverageRate(t, pool, tenantID, packagePrice, overageRate, 100)

		// Subscribe
		subResult, err := manager.Subscribe(context.Background(), SubscribeRequest{
			TenantID:  tenantID,
			PackageID: packageID,
			AutoRenew: true,
		})
		require.NoError(t, err)
		require.True(t, subResult.Success)

		// Get state before
		balanceBefore := getTenantBalance(t, pool, tenantID)
		subBefore, err := manager.GetActiveSubscription(context.Background(), tenantID)
		require.NoError(t, err)

		// Execute overage charge
		_, err = manager.CalculateOverageCost(context.Background(), subResult.SubscriptionID, overageMessages)

		// Get state after
		balanceAfter := getTenantBalance(t, pool, tenantID)
		subAfter, err2 := manager.GetActiveSubscription(context.Background(), tenantID)
		require.NoError(t, err2)

		expectedCost := overageRate.Mul(decimal.NewFromInt(overageMessages))

		if err == nil {
			// Success case: All changes should be applied atomically
			
			// Property 1: Balance deducted correctly
			expectedBalance := balanceBefore.Sub(expectedCost)
			assert.True(t, balanceAfter.Equal(expectedBalance),
				"Balance should be deducted atomically")

			// Property 2: Subscription updated correctly
			assert.Equal(t, subBefore.OverageMessages+overageMessages, subAfter.OverageMessages,
				"Overage messages should be incremented atomically")
			expectedOverageCost := subBefore.OverageCost.Add(expectedCost)
			assert.True(t, subAfter.OverageCost.Equal(expectedOverageCost),
				"Overage cost should be incremented atomically")

			// Property 3: Accounting transaction exists
			txExists := checkAccountingTransaction(t, pool, tenantID, "OVERAGE")
			assert.True(t, txExists,
				"Accounting transaction should exist (atomicity)")

		} else {
			// Failure case: No changes should be applied
			
			// Property 1: Balance unchanged
			assert.True(t, balanceAfter.Equal(balanceBefore),
				"Balance should not change on failure (atomicity)")

			// Property 2: Subscription unchanged
			assert.Equal(t, subBefore.OverageMessages, subAfter.OverageMessages,
				"Overage messages should not change on failure (atomicity)")
			assert.True(t, subAfter.OverageCost.Equal(subBefore.OverageCost),
				"Overage cost should not change on failure (atomicity)")
		}

		// Cleanup
		cleanupTenant(t, pool, tenantID)
	})
}

// TestProperty_OverageInvariant tests that overage cost always equals rate * messages
func TestProperty_OverageInvariant(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	manager := NewManager(pool, logger)

	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		initialBalance := decimal.NewFromInt(1000)
		packagePrice := decimal.NewFromInt(50)
		overageRate := rapid.Float64Range(0.01, 0.10).Draw(t, "overage_rate")
		overageRateDecimal := decimal.NewFromFloat(overageRate)
		
		// Generate multiple overage operations
		numOperations := rapid.IntRange(2, 5).Draw(t, "num_operations")
		overageCounts := make([]int64, numOperations)
		for i := 0; i < numOperations; i++ {
			overageCounts[i] = rapid.Int64Range(1, 10).Draw(t, fmt.Sprintf("overage_%d", i))
		}

		// Setup
		setupTenant(t, pool, tenantID, initialBalance)
		packageID := setupTestPackageWithOverageRate(t, pool, tenantID, packagePrice, overageRateDecimal, 100)

		// Subscribe
		subResult, err := manager.Subscribe(context.Background(), SubscribeRequest{
			TenantID:  tenantID,
			PackageID: packageID,
			AutoRenew: true,
		})
		require.NoError(t, err)
		require.True(t, subResult.Success)

		// Execute multiple overage charges
		for i, msgCount := range overageCounts {
			_, err := manager.CalculateOverageCost(context.Background(), subResult.SubscriptionID, msgCount)
			require.NoError(t, err, "Overage charge %d should succeed", i)

			// After each operation, verify invariant
			sub, err := manager.GetActiveSubscription(context.Background(), tenantID)
			require.NoError(t, err)

			// Property: overage_cost = overage_rate * overage_messages
			expectedCost := overageRateDecimal.Mul(decimal.NewFromInt(sub.OverageMessages))
			
			// Allow rounding differences: overage_cost is stored as numeric(10,2),
			// so each operation rounds the accumulated total, drifting by up to
			// 0.005 per operation (~0.025 worst case across numOperations).
			diff := sub.OverageCost.Sub(expectedCost).Abs()
			assert.True(t, diff.LessThan(decimal.NewFromFloat(0.05)),
				"Invariant (cost = rate * messages) must hold after operation %d: expected %s, got %s, diff %s",
				i, expectedCost, sub.OverageCost, diff)
		}

		// Cleanup
		cleanupTenant(t, pool, tenantID)
	})
}

// Helper function to setup package with specific overage rate
func setupTestPackageWithOverageRate(t *rapid.T, pool *pgxpool.Pool, tenantID uuid.UUID, price decimal.Decimal, overageRate decimal.Decimal, messages int64) int64 {
	// Set tenant context for RLS
	_, err := pool.Exec(context.Background(), "SELECT set_config('app.current_tenant', $1, false)", tenantID.String())
	if err != nil {
		t.Fatalf("Failed to set tenant context: %v", err)
	}

	query := `
		INSERT INTO packages (
			tenant_id, service_id, name, code, price_usd,
			included_messages, overage_rate, is_unlimited,
			billing_cycle, rollover_enabled, is_active
		) VALUES (
			$1, 2, 'Test Package', $2, $3, $4, $5, false, 'monthly', true, true
		)
		RETURNING id
	`

	var packageID int64
	code := fmt.Sprintf("TEST_%s", uuid.New().String()[:8])
	err = pool.QueryRow(context.Background(), query, tenantID, code, price, messages, overageRate).Scan(&packageID)
	if err != nil {
		t.Fatalf("Failed to setup package: %v", err)
	}

	return packageID
}
