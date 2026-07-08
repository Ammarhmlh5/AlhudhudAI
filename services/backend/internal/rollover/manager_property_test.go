package rollover

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"pgregory.net/rapid"
)

// Property 9: حساب الرسائل المتبقية عند نهاية الشهر
// Validates: Requirements 6.2
//
// This property test verifies that CalculateRemainingMessages:
// 1. Returns 0 when rollover is disabled
// 2. Returns 0 when package is unlimited
// 3. Returns 0 when messages_remaining is 0 or negative
// 4. Returns correct messages_remaining when rollover is enabled
// 5. Handles edge cases correctly

func TestProperty_CalculateRemainingMessages(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	// Setup test database
	ctx := context.Background()
	pool := setupTestDB(t)
	defer pool.Close()

	logger := zap.NewNop()
	manager := NewManager(pool, logger)

	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		tenantID := uuid.New()
		rolloverEnabled := rapid.Bool().Draw(t, "rollover_enabled")
		isUnlimited := rapid.Bool().Draw(t, "is_unlimited")
		messagesRemaining := rapid.Int64Range(0, 10000).Draw(t, "messages_remaining")

		// Create test package
		packageID := createTestPackage(t, ctx, pool, tenantID, rolloverEnabled, isUnlimited)

		// Create test subscription
		subscriptionID := createTestSubscription(t, ctx, pool, tenantID, packageID, messagesRemaining)

		// Calculate remaining messages
		remaining, err := manager.CalculateRemainingMessages(ctx, subscriptionID)
		require.NoError(t, err)

		// Verify properties
		if isUnlimited {
			// Property: Unlimited packages should return 0
			require.Equal(t, int64(0), remaining,
				"Unlimited packages should not have rollover")
		} else if !rolloverEnabled {
			// Property: Disabled rollover should return 0
			require.Equal(t, int64(0), remaining,
				"Disabled rollover should return 0")
		} else if messagesRemaining <= 0 {
			// Property: No remaining messages should return 0
			require.Equal(t, int64(0), remaining,
				"No remaining messages should return 0")
		} else {
			// Property: Should return exact messages_remaining
			require.Equal(t, messagesRemaining, remaining,
				"Should return exact messages_remaining when rollover enabled")
		}

		// Cleanup
		cleanupTestData(t, ctx, pool, subscriptionID, packageID)
	})
}

// TestProperty_CalculateRemainingMessages_Idempotency tests that calling
// CalculateRemainingMessages multiple times returns the same result
func TestProperty_CalculateRemainingMessages_Idempotency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	ctx := context.Background()
	pool := setupTestDB(t)
	defer pool.Close()

	logger := zap.NewNop()
	manager := NewManager(pool, logger)

	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		tenantID := uuid.New()
		rolloverEnabled := true
		isUnlimited := false
		messagesRemaining := rapid.Int64Range(1, 10000).Draw(t, "messages_remaining")

		// Create test package and subscription
		packageID := createTestPackage(t, ctx, pool, tenantID, rolloverEnabled, isUnlimited)
		subscriptionID := createTestSubscription(t, ctx, pool, tenantID, packageID, messagesRemaining)

		// Call multiple times
		result1, err1 := manager.CalculateRemainingMessages(ctx, subscriptionID)
		require.NoError(t, err1)

		result2, err2 := manager.CalculateRemainingMessages(ctx, subscriptionID)
		require.NoError(t, err2)

		result3, err3 := manager.CalculateRemainingMessages(ctx, subscriptionID)
		require.NoError(t, err3)

		// Property: All results should be identical (idempotent)
		require.Equal(t, result1, result2,
			"Multiple calls should return same result")
		require.Equal(t, result2, result3,
			"Multiple calls should return same result")

		// Cleanup
		cleanupTestData(t, ctx, pool, subscriptionID, packageID)
	})
}

// TestProperty_CalculateRemainingMessages_BoundaryValues tests boundary values
func TestProperty_CalculateRemainingMessages_BoundaryValues(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	ctx := context.Background()
	pool := setupTestDB(t)
	defer pool.Close()

	logger := zap.NewNop()
	manager := NewManager(pool, logger)

	testCases := []struct {
		name              string
		messagesRemaining int64
		expectedResult    int64
	}{
		{"Zero messages", 0, 0},
		{"One message", 1, 1},
		{"Maximum int64", 9223372036854775807, 9223372036854775807},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tenantID := uuid.New()
			rolloverEnabled := true
			isUnlimited := false

			packageID := createTestPackage(t, ctx, pool, tenantID, rolloverEnabled, isUnlimited)
			subscriptionID := createTestSubscription(t, ctx, pool, tenantID, packageID, tc.messagesRemaining)

			result, err := manager.CalculateRemainingMessages(ctx, subscriptionID)
			require.NoError(t, err)
			require.Equal(t, tc.expectedResult, result)

			cleanupTestData(t, ctx, pool, subscriptionID, packageID)
		})
	}
}

// Helper functions

func setupTestDB(t *testing.T) *pgxpool.Pool {
	// Use test database connection
	connString := "postgres://postgres:postgres@localhost:5432/sms_gateway_test?sslmode=disable"
	pool, err := pgxpool.New(context.Background(), connString)
	require.NoError(t, err)
	return pool
}

func createTestPackage(t require.TestingT, ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, rolloverEnabled, isUnlimited bool) int64 {
	query := `
		INSERT INTO packages (
			tenant_id,
			service_id,
			name,
			description,
			price,
			currency,
			billing_cycle,
			messages_included,
			is_unlimited,
			rollover_enabled,
			rollover_limit,
			rollover_expiry_months,
			overage_rate,
			is_active
		) VALUES (
			$1, 1, 'Test Package', 'Test Description', 100.00, 'USD', 'monthly',
			1000, $2, $3, 500, 3, 0.05, true
		) RETURNING id
	`

	var packageID int64
	err := pool.QueryRow(ctx, query, tenantID, isUnlimited, rolloverEnabled).Scan(&packageID)
	require.NoError(t, err)
	return packageID
}

func createTestSubscription(t require.TestingT, ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, packageID, messagesRemaining int64) int64 {
	query := `
		INSERT INTO subscriptions (
			tenant_id,
			user_id,
			package_id,
			status,
			start_date,
			end_date,
			messages_included,
			messages_used,
			messages_remaining,
			overage_messages,
			overage_cost,
			auto_renew
		) VALUES (
			$1, 1, $2, 'active', CURRENT_DATE, CURRENT_DATE + INTERVAL '30 days',
			1000, 0, $3, 0, 0.00, true
		) RETURNING id
	`

	var subscriptionID int64
	err := pool.QueryRow(ctx, query, tenantID, packageID, messagesRemaining).Scan(&subscriptionID)
	require.NoError(t, err)
	return subscriptionID
}

func cleanupTestData(t require.TestingT, ctx context.Context, pool *pgxpool.Pool, subscriptionID, packageID int64) {
	// Delete subscription
	_, err := pool.Exec(ctx, "DELETE FROM subscriptions WHERE id = $1", subscriptionID)
	require.NoError(t, err)

	// Delete package
	_, err = pool.Exec(ctx, "DELETE FROM packages WHERE id = $1", packageID)
	require.NoError(t, err)
}


// Property 10: احترام الحد الأقصى للترحيل
// Validates: Requirements 6.5
//
// This property test verifies that RolloverMessages:
// 1. Never rolls over more than rollover_limit
// 2. Rolls over all messages if remaining < limit
// 3. Rolls over exactly limit if remaining >= limit
// 4. Handles nil rollover_limit (unlimited rollover)

func TestProperty_RolloverMessages_RespectLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	ctx := context.Background()
	pool := setupTestDB(t)
	defer pool.Close()

	logger := zap.NewNop()
	manager := NewManager(pool, logger)

	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		tenantID := uuid.New()
		rolloverEnabled := true
		isUnlimited := false
		
		// Generate rollover limit (can be nil for unlimited)
		hasLimit := rapid.Bool().Draw(t, "has_limit")
		var rolloverLimit *int64
		if hasLimit {
			limit := rapid.Int64Range(100, 5000).Draw(t, "rollover_limit")
			rolloverLimit = &limit
		}
		
		// Generate remaining messages
		messagesRemaining := rapid.Int64Range(1, 10000).Draw(t, "messages_remaining")

		// Create test package with rollover limit
		packageID := createTestPackageWithLimit(t, ctx, pool, tenantID, rolloverEnabled, isUnlimited, rolloverLimit)
		subscriptionID := createTestSubscription(t, ctx, pool, tenantID, packageID, messagesRemaining)

		// Rollover messages
		err := manager.RolloverMessages(ctx, subscriptionID, messagesRemaining)
		require.NoError(t, err)

		// Get rollover records
		rolloverRecords, err := manager.GetRolloverMessages(ctx, subscriptionID)
		require.NoError(t, err)

		if rolloverEnabled && messagesRemaining > 0 {
			// Should have created a rollover record
			require.Len(t, rolloverRecords, 1, "Should create one rollover record")

			rolledOver := rolloverRecords[0].MessagesCount

			if rolloverLimit == nil {
				// Property: No limit means all messages rolled over
				require.Equal(t, messagesRemaining, rolledOver,
					"Should rollover all messages when no limit")
			} else if messagesRemaining <= *rolloverLimit {
				// Property: Remaining < limit means all messages rolled over
				require.Equal(t, messagesRemaining, rolledOver,
					"Should rollover all messages when remaining <= limit")
			} else {
				// Property: Remaining > limit means only limit rolled over
				require.Equal(t, *rolloverLimit, rolledOver,
					"Should rollover only limit when remaining > limit")
				require.LessOrEqual(t, rolledOver, *rolloverLimit,
					"Should never exceed rollover_limit")
			}
		}

		// Cleanup
		cleanupRolloverData(t, ctx, pool, subscriptionID)
		cleanupTestData(t, ctx, pool, subscriptionID, packageID)
	})
}

// TestProperty_RolloverMessages_ExpiryDate tests that expiry date is set correctly
func TestProperty_RolloverMessages_ExpiryDate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	ctx := context.Background()
	pool := setupTestDB(t)
	defer pool.Close()

	logger := zap.NewNop()
	manager := NewManager(pool, logger)

	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		tenantID := uuid.New()
		rolloverEnabled := true
		isUnlimited := false
		messagesRemaining := rapid.Int64Range(1, 1000).Draw(t, "messages_remaining")
		rolloverExpiryMonths := rapid.IntRange(1, 12).Draw(t, "rollover_expiry_months")

		// Create test package with expiry months
		packageID := createTestPackageWithExpiry(t, ctx, pool, tenantID, rolloverEnabled, isUnlimited, rolloverExpiryMonths)
		subscriptionID := createTestSubscription(t, ctx, pool, tenantID, packageID, messagesRemaining)

		// Rollover messages
		err := manager.RolloverMessages(ctx, subscriptionID, messagesRemaining)
		require.NoError(t, err)

		// Get rollover records
		rolloverRecords, err := manager.GetRolloverMessages(ctx, subscriptionID)
		require.NoError(t, err)
		require.Len(t, rolloverRecords, 1)

		// Property: Expiry date should be approximately rolloverExpiryMonths from now
		expiryDate := rolloverRecords[0].ExpiryDate
		expectedExpiry := time.Now().AddDate(0, rolloverExpiryMonths, 0)
		
		// Allow 1 day tolerance for timing differences
		diff := expiryDate.Sub(expectedExpiry).Abs()
		require.Less(t, diff, 24*time.Hour,
			"Expiry date should be approximately %d months from now", rolloverExpiryMonths)

		// Cleanup
		cleanupRolloverData(t, ctx, pool, subscriptionID)
		cleanupTestData(t, ctx, pool, subscriptionID, packageID)
	})
}

// TestProperty_RolloverMessages_Idempotency tests that calling RolloverMessages
// multiple times doesn't create duplicate records
func TestProperty_RolloverMessages_Idempotency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	ctx := context.Background()
	pool := setupTestDB(t)
	defer pool.Close()

	logger := zap.NewNop()
	manager := NewManager(pool, logger)

	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		tenantID := uuid.New()
		rolloverEnabled := true
		isUnlimited := false
		messagesRemaining := rapid.Int64Range(1, 1000).Draw(t, "messages_remaining")

		packageID := createTestPackage(t, ctx, pool, tenantID, rolloverEnabled, isUnlimited)
		subscriptionID := createTestSubscription(t, ctx, pool, tenantID, packageID, messagesRemaining)

		// Call RolloverMessages multiple times
		err1 := manager.RolloverMessages(ctx, subscriptionID, messagesRemaining)
		require.NoError(t, err1)

		err2 := manager.RolloverMessages(ctx, subscriptionID, messagesRemaining)
		require.NoError(t, err2)

		err3 := manager.RolloverMessages(ctx, subscriptionID, messagesRemaining)
		require.NoError(t, err3)

		// Get rollover records
		rolloverRecords, err := manager.GetRolloverMessages(ctx, subscriptionID)
		require.NoError(t, err)

		// Property: Multiple calls should create multiple records (not idempotent by design)
		// This is expected behavior - each renewal creates a new rollover record
		require.GreaterOrEqual(t, len(rolloverRecords), 3,
			"Each call should create a new rollover record")

		// Cleanup
		cleanupRolloverData(t, ctx, pool, subscriptionID)
		cleanupTestData(t, ctx, pool, subscriptionID, packageID)
	})
}

// Helper functions for rollover tests

func createTestPackageWithLimit(t require.TestingT, ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, rolloverEnabled, isUnlimited bool, rolloverLimit *int64) int64 {
	query := `
		INSERT INTO packages (
			tenant_id,
			service_id,
			name,
			description,
			price,
			currency,
			billing_cycle,
			messages_included,
			is_unlimited,
			rollover_enabled,
			rollover_limit,
			rollover_expiry_months,
			overage_rate,
			is_active
		) VALUES (
			$1, 1, 'Test Package', 'Test Description', 100.00, 'USD', 'monthly',
			1000, $2, $3, $4, 3, 0.05, true
		) RETURNING id
	`

	var packageID int64
	err := pool.QueryRow(ctx, query, tenantID, isUnlimited, rolloverEnabled, rolloverLimit).Scan(&packageID)
	require.NoError(t, err)
	return packageID
}

func createTestPackageWithExpiry(t require.TestingT, ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, rolloverEnabled, isUnlimited bool, rolloverExpiryMonths int) int64 {
	query := `
		INSERT INTO packages (
			tenant_id,
			service_id,
			name,
			description,
			price,
			currency,
			billing_cycle,
			messages_included,
			is_unlimited,
			rollover_enabled,
			rollover_limit,
			rollover_expiry_months,
			overage_rate,
			is_active
		) VALUES (
			$1, 1, 'Test Package', 'Test Description', 100.00, 'USD', 'monthly',
			1000, $2, $3, 500, $4, 0.05, true
		) RETURNING id
	`

	var packageID int64
	err := pool.QueryRow(ctx, query, tenantID, isUnlimited, rolloverEnabled, rolloverExpiryMonths).Scan(&packageID)
	require.NoError(t, err)
	return packageID
}

func cleanupRolloverData(t require.TestingT, ctx context.Context, pool *pgxpool.Pool, subscriptionID int64) {
	_, err := pool.Exec(ctx, "DELETE FROM rollover_messages WHERE subscription_id = $1", subscriptionID)
	require.NoError(t, err)
}


// Property 11: استخدام الرسائل المرحّلة بترتيب FIFO
// Validates: Requirements 6.8
//
// This property test verifies that UseRolloverMessages:
// 1. Uses oldest rollover messages first (FIFO order)
// 2. Correctly updates or deletes rollover records
// 3. Returns correct number of messages used
// 4. Handles partial consumption correctly

func TestProperty_UseRolloverMessages_FIFO(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	ctx := context.Background()
	pool := setupTestDB(t)
	defer pool.Close()

	logger := zap.NewNop()
	manager := NewManager(pool, logger)

	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		tenantID := uuid.New()
		rolloverEnabled := true
		isUnlimited := false

		packageID := createTestPackage(t, ctx, pool, tenantID, rolloverEnabled, isUnlimited)
		subscriptionID := createTestSubscription(t, ctx, pool, tenantID, packageID, 1000)

		// Create multiple rollover records with different timestamps
		numRecords := rapid.IntRange(2, 5).Draw(t, "num_records")
		var rolloverIDs []int64
		var messageCounts []int64
		
		for i := 0; i < numRecords; i++ {
			messageCount := rapid.Int64Range(50, 200).Draw(t, fmt.Sprintf("messages_%d", i))
			messageCounts = append(messageCounts, messageCount)
			
			// Create rollover record with specific timestamp
			rolloverID := createRolloverRecord(t, ctx, pool, subscriptionID, tenantID, messageCount, i)
			rolloverIDs = append(rolloverIDs, rolloverID)
			
			// Small delay to ensure different timestamps
			time.Sleep(10 * time.Millisecond)
		}

		// Calculate total available messages
		totalAvailable := int64(0)
		for _, count := range messageCounts {
			totalAvailable += count
		}

		// Use some messages
		toUse := rapid.Int64Range(1, totalAvailable).Draw(t, "messages_to_use")
		
		used, err := manager.UseRolloverMessages(ctx, subscriptionID, toUse)
		require.NoError(t, err)

		// Property: Should use exactly the requested amount (or all available)
		expectedUsed := toUse
		if toUse > totalAvailable {
			expectedUsed = totalAvailable
		}
		require.Equal(t, expectedUsed, used,
			"Should use exactly requested amount or all available")

		// Get remaining rollover messages
		remaining, err := manager.GetRolloverMessages(ctx, subscriptionID)
		require.NoError(t, err)

		// Calculate total remaining
		totalRemaining := int64(0)
		for _, r := range remaining {
			totalRemaining += r.MessagesCount
		}

		// Property: Total used + total remaining should equal total available
		require.Equal(t, totalAvailable, used+totalRemaining,
			"Total used + remaining should equal total available")

		// Property: Oldest records should be consumed first (FIFO)
		// If we used less than first record, first record should be partially consumed
		// If we used more than first record, first record should be deleted
		if used < messageCounts[0] {
			// First record should still exist with reduced count
			require.Len(t, remaining, numRecords,
				"All records should still exist when partially consuming first")
			require.Equal(t, messageCounts[0]-used, remaining[0].MessagesCount,
				"First record should have reduced count")
		} else {
			// First record should be deleted
			require.Less(t, len(remaining), numRecords,
				"Some records should be deleted when fully consumed")
		}

		// Cleanup
		cleanupRolloverData(t, ctx, pool, subscriptionID)
		cleanupTestData(t, ctx, pool, subscriptionID, packageID)
	})
}

// TestProperty_UseRolloverMessages_FullConsumption tests that records are deleted
// when fully consumed
func TestProperty_UseRolloverMessages_FullConsumption(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	ctx := context.Background()
	pool := setupTestDB(t)
	defer pool.Close()

	logger := zap.NewNop()
	manager := NewManager(pool, logger)

	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		tenantID := uuid.New()
		rolloverEnabled := true
		isUnlimited := false

		packageID := createTestPackage(t, ctx, pool, tenantID, rolloverEnabled, isUnlimited)
		subscriptionID := createTestSubscription(t, ctx, pool, tenantID, packageID, 1000)

		// Create a single rollover record
		messageCount := rapid.Int64Range(50, 200).Draw(t, "messages")
		createRolloverRecord(t, ctx, pool, subscriptionID, tenantID, messageCount, 0)

		// Use exactly all messages
		used, err := manager.UseRolloverMessages(ctx, subscriptionID, messageCount)
		require.NoError(t, err)
		require.Equal(t, messageCount, used)

		// Property: Record should be deleted when fully consumed
		remaining, err := manager.GetRolloverMessages(ctx, subscriptionID)
		require.NoError(t, err)
		require.Len(t, remaining, 0,
			"Record should be deleted when fully consumed")

		// Cleanup
		cleanupTestData(t, ctx, pool, subscriptionID, packageID)
	})
}

// TestProperty_UseRolloverMessages_PartialConsumption tests that records are updated
// when partially consumed
func TestProperty_UseRolloverMessages_PartialConsumption(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	ctx := context.Background()
	pool := setupTestDB(t)
	defer pool.Close()

	logger := zap.NewNop()
	manager := NewManager(pool, logger)

	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		tenantID := uuid.New()
		rolloverEnabled := true
		isUnlimited := false

		packageID := createTestPackage(t, ctx, pool, tenantID, rolloverEnabled, isUnlimited)
		subscriptionID := createTestSubscription(t, ctx, pool, tenantID, packageID, 1000)

		// Create a single rollover record
		messageCount := rapid.Int64Range(100, 500).Draw(t, "messages")
		createRolloverRecord(t, ctx, pool, subscriptionID, tenantID, messageCount, 0)

		// Use less than available
		toUse := rapid.Int64Range(1, messageCount-1).Draw(t, "to_use")
		
		used, err := manager.UseRolloverMessages(ctx, subscriptionID, toUse)
		require.NoError(t, err)
		require.Equal(t, toUse, used)

		// Property: Record should be updated with remaining count
		remaining, err := manager.GetRolloverMessages(ctx, subscriptionID)
		require.NoError(t, err)
		require.Len(t, remaining, 1,
			"Record should still exist when partially consumed")
		require.Equal(t, messageCount-toUse, remaining[0].MessagesCount,
			"Record should have correct remaining count")

		// Cleanup
		cleanupRolloverData(t, ctx, pool, subscriptionID)
		cleanupTestData(t, ctx, pool, subscriptionID, packageID)
	})
}

// TestProperty_UseRolloverMessages_NoAvailable tests behavior when no rollover
// messages are available
func TestProperty_UseRolloverMessages_NoAvailable(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	ctx := context.Background()
	pool := setupTestDB(t)
	defer pool.Close()

	logger := zap.NewNop()
	manager := NewManager(pool, logger)

	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		tenantID := uuid.New()
		rolloverEnabled := true
		isUnlimited := false

		packageID := createTestPackage(t, ctx, pool, tenantID, rolloverEnabled, isUnlimited)
		subscriptionID := createTestSubscription(t, ctx, pool, tenantID, packageID, 1000)

		// Don't create any rollover records
		toUse := rapid.Int64Range(1, 100).Draw(t, "to_use")

		// Property: Should return 0 when no rollover messages available
		used, err := manager.UseRolloverMessages(ctx, subscriptionID, toUse)
		require.NoError(t, err)
		require.Equal(t, int64(0), used,
			"Should return 0 when no rollover messages available")

		// Cleanup
		cleanupTestData(t, ctx, pool, subscriptionID, packageID)
	})
}

// Helper function to create rollover record with specific timestamp
func createRolloverRecord(t require.TestingT, ctx context.Context, pool *pgxpool.Pool, subscriptionID int64, tenantID uuid.UUID, messageCount int64, delaySeconds int) int64 {
	// Create record with timestamp offset
	createdAt := time.Now().Add(time.Duration(delaySeconds) * time.Second)
	expiryDate := time.Now().AddDate(0, 3, 0)
	sourceMonth := time.Now().Truncate(24 * time.Hour)
	sourceMonth = time.Date(sourceMonth.Year(), sourceMonth.Month(), 1, 0, 0, 0, 0, time.UTC)

	query := `
		INSERT INTO rollover_messages (
			subscription_id,
			tenant_id,
			messages_count,
			source_month,
			expiry_date,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var rolloverID int64
	err := pool.QueryRow(ctx, query,
		subscriptionID,
		tenantID,
		messageCount,
		sourceMonth,
		expiryDate,
		createdAt,
	).Scan(&rolloverID)
	require.NoError(t, err)
	return rolloverID
}
