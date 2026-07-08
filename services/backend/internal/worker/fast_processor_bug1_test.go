package worker

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"pgregory.net/rapid"

	"github.com/al-hudhud/backend/internal/redis"
)

// setupTestRedis creates a test Redis client using miniredis
func setupTestRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	mr := miniredis.RunT(t)

	client := goredis.NewClient(&goredis.Options{
		Addr: mr.Addr(),
	})

	logger := zap.NewNop()

	redisClient := &redis.Client{
		Client: client,
		Logger: logger,
	}

	return redisClient, mr
}

// Bug 1 Exploration Test: Empty Active Tenants List
// **Validates: Requirements 1.1, 2.1**
//
// **Property 1: Bug Condition** - Active Tenants List Returns Empty
//
// This test MUST FAIL on unfixed code - failure confirms the bug exists.
// The bug: getActiveTenants() returns empty list when Redis set has tenant UUIDs.
//
// CRITICAL: This test encodes the EXPECTED behavior (what SHOULD happen).
// On UNFIXED code: Test FAILS (proves bug exists)
// On FIXED code: Test PASSES (proves bug is fixed)
//
// DO NOT attempt to fix the test or the code when it fails.
// Document the failure as proof of the bug.
func TestProperty_Bug1_ActiveTenantsListReturnsEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	// Setup Redis client with miniredis
	redisClient, mr := setupTestRedis(t)
	defer mr.Close()
	ctx := context.Background()

	// Ensure clean state
	defer func() {
		activeKey := redis.AppNamespace + ":active:tenants"
		redisClient.Client.Del(ctx, activeKey)
	}()

	rapid.Check(t, func(t *rapid.T) {
		// Generate 1-10 random tenant UUIDs
		tenantCount := rapid.IntRange(1, 10).Draw(t, "tenantCount")
		tenantIDs := make([]uuid.UUID, tenantCount)
		
		for i := 0; i < tenantCount; i++ {
			tenantIDs[i] = uuid.New()
		}

		// Setup: Add tenant UUIDs to Redis active set
		activeKey := redis.AppNamespace + ":active:tenants"
		
		// Clear any existing data
		redisClient.Client.Del(ctx, activeKey)
		
		// Add tenants to active set
		for _, tenantID := range tenantIDs {
			err := redisClient.Client.SAdd(ctx, activeKey, tenantID.String()).Err()
			require.NoError(t, err, "Failed to add tenant to active set")
		}

		// Verify tenants were added to Redis
		members, err := redisClient.Client.SMembers(ctx, activeKey).Result()
		require.NoError(t, err, "Failed to read active tenants from Redis")
		require.Len(t, members, tenantCount, "Redis set should contain all added tenants")

		// Create a minimal FastMessageProcessor to test getActiveTenants()
		messageQueue := redis.NewMessageQueue(redisClient)
		logger := zap.NewNop()
		processor := &FastMessageProcessor{
			messageQueue: messageQueue,
			logger:       logger,
		}

		// ACT: Call getActiveTenants()
		activeTenants, err := processor.getActiveTenants(ctx)
		require.NoError(t, err, "getActiveTenants should not return error")

		// ASSERT: Expected behavior - should return the tenant UUIDs from Redis
		// On UNFIXED code: This assertion WILL FAIL because getActiveTenants() returns []
		// On FIXED code: This assertion WILL PASS
		assert.Len(t, activeTenants, tenantCount,
			"Bug Condition: getActiveTenants() returns empty list when Redis set has %d tenant UUIDs. "+
				"Expected: %d tenants, Got: %d tenants. "+
				"This failure confirms the bug exists.",
			tenantCount, tenantCount, len(activeTenants))

		// Additional assertion: Verify the returned UUIDs match what we added
		if len(activeTenants) > 0 {
			returnedMap := make(map[uuid.UUID]bool)
			for _, id := range activeTenants {
				returnedMap[id] = true
			}

			for _, expectedID := range tenantIDs {
				assert.True(t, returnedMap[expectedID],
					"Returned tenant list should contain tenant %s", expectedID)
			}
		}

		// Cleanup for next iteration
		redisClient.Client.Del(ctx, activeKey)
	})
}

// Example-based test for Bug 1 - Simpler counterexample
// **Validates: Requirements 1.1, 2.1**
//
// This test provides a concrete example of the bug condition.
// EXPECTED OUTCOME on UNFIXED code: FAILS (proves bug exists)
func TestBug1_EmptyActiveTenantsWithRedisData(t *testing.T) {
	// Setup Redis client with miniredis
	redisClient, mr := setupTestRedis(t)
	defer mr.Close()
	ctx := context.Background()

	// Cleanup
	activeKey := redis.AppNamespace + ":active:tenants"
	defer redisClient.Client.Del(ctx, activeKey)

	// Setup: Create 3 test tenants
	tenant1 := uuid.New()
	tenant2 := uuid.New()
	tenant3 := uuid.New()

	// Add tenants to Redis active set
	err := redisClient.Client.SAdd(ctx, activeKey, tenant1.String()).Err()
	require.NoError(t, err)
	err = redisClient.Client.SAdd(ctx, activeKey, tenant2.String()).Err()
	require.NoError(t, err)
	err = redisClient.Client.SAdd(ctx, activeKey, tenant3.String()).Err()
	require.NoError(t, err)

	// Verify Redis has the data
	members, err := redisClient.Client.SMembers(ctx, activeKey).Result()
	require.NoError(t, err)
	require.Len(t, members, 3, "Redis should have 3 tenants")

	// Create processor
	messageQueue := redis.NewMessageQueue(redisClient)
	logger := zap.NewNop()
	processor := &FastMessageProcessor{
		messageQueue: messageQueue,
		logger:       logger,
	}

	// ACT: Call getActiveTenants()
	activeTenants, err := processor.getActiveTenants(ctx)
	require.NoError(t, err)

	// ASSERT: Expected behavior
	// On UNFIXED code: This WILL FAIL (returns empty list)
	// On FIXED code: This WILL PASS (returns 3 tenants)
	assert.Len(t, activeTenants, 3,
		"Bug Condition: getActiveTenants() returns empty list. "+
			"Redis has 3 tenants but function returned %d. "+
			"This confirms Bug 1 exists: Empty Active Tenants List",
		len(activeTenants))

	// Verify the specific tenant IDs are returned
	if len(activeTenants) == 3 {
		tenantMap := make(map[uuid.UUID]bool)
		for _, id := range activeTenants {
			tenantMap[id] = true
		}

		assert.True(t, tenantMap[tenant1], "Should return tenant1")
		assert.True(t, tenantMap[tenant2], "Should return tenant2")
		assert.True(t, tenantMap[tenant3], "Should return tenant3")
	}
}

// Edge case test: Empty Redis set should return empty list (valid behavior)
// **Validates: Requirements 2.1**
//
// This test verifies that when Redis set is truly empty, returning empty list is correct.
// This should PASS on both UNFIXED and FIXED code.
func TestBug1_EmptyRedisSetReturnsEmptyList(t *testing.T) {
	redisClient, mr := setupTestRedis(t)
	defer mr.Close()
	ctx := context.Background()

	// Ensure Redis set is empty
	activeKey := redis.AppNamespace + ":active:tenants"
	redisClient.Client.Del(ctx, activeKey)

	// Create processor
	messageQueue := redis.NewMessageQueue(redisClient)
	logger := zap.NewNop()
	processor := &FastMessageProcessor{
		messageQueue: messageQueue,
		logger:       logger,
	}

	// ACT
	activeTenants, err := processor.getActiveTenants(ctx)
	require.NoError(t, err)

	// ASSERT: Empty Redis set should return empty list (this is correct behavior)
	assert.Len(t, activeTenants, 0,
		"When Redis set is empty, getActiveTenants() should return empty list")
}

// Test: Invalid UUID in Redis set should be skipped
// **Validates: Requirements 2.1**
//
// This test verifies that invalid UUIDs in Redis are handled gracefully.
func TestBug1_InvalidUUIDsAreSkipped(t *testing.T) {
	redisClient, mr := setupTestRedis(t)
	defer mr.Close()
	ctx := context.Background()

	activeKey := redis.AppNamespace + ":active:tenants"
	defer redisClient.Client.Del(ctx, activeKey)

	// Add valid and invalid UUIDs
	validTenant := uuid.New()
	redisClient.Client.SAdd(ctx, activeKey, validTenant.String())
	redisClient.Client.SAdd(ctx, activeKey, "invalid-uuid-123")
	redisClient.Client.SAdd(ctx, activeKey, "not-a-uuid")

	// Create processor
	messageQueue := redis.NewMessageQueue(redisClient)
	logger := zap.NewNop()
	processor := &FastMessageProcessor{
		messageQueue: messageQueue,
		logger:       logger,
	}

	// ACT
	activeTenants, err := processor.getActiveTenants(ctx)
	require.NoError(t, err)

	// ASSERT: Should return only the valid UUID
	// On UNFIXED code: Returns 0 (bug)
	// On FIXED code: Returns 1 (correct - only valid UUID)
	assert.Len(t, activeTenants, 1,
		"Should return 1 valid tenant, skipping invalid UUIDs. Got: %d", len(activeTenants))

	if len(activeTenants) == 1 {
		assert.Equal(t, validTenant, activeTenants[0],
			"Should return the valid tenant UUID")
	}
}
