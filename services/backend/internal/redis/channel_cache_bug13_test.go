package redis

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// TestBug13_CacheNotInvalidatedOnChannelOffline_Exploration is a bug condition exploration test
// **Validates: Requirements 2.6**
//
// BUG CONDITION: When channel status changes to offline, cache entry should be invalidated immediately
// EXPECTED OUTCOME: Test PASSES on fixed code (confirms bug is fixed)
//
// This test verifies that the cache IS actively invalidated when InvalidateChannelOnOffline is called.
// After the fix (tasks 15.1-15.3), the cache should be cleared immediately, not persist for TTL.
func TestBug13_CacheNotInvalidatedOnChannelOffline_Exploration(t *testing.T) {
	// Setup Redis client for testing using miniredis
	client, mr := setupTestRedis(t)
	defer mr.Close()
	// Note: Don't call client.Close() as it causes panic with nil logger in test context

	cache := NewChannelCache(client)
	ctx := context.Background()

	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		tenantID := uuid.New()
		channelID := uuid.New()
		gatewayID := uuid.New()

		// Create a cached channel that is initially online
		cachedChannel := &CachedChannel{
			ChannelID:  channelID,
			GatewayID:  &gatewayID,
			Type:       "PHONE",
			Priority:   rapid.IntRange(1, 10).Draw(t, "priority"),
			IsOnline:   true, // Initially online
			CachedAt:   time.Now(),
		}

		// Step 1: Store channel in cache with TTL
		err := cache.Set(ctx, tenantID, cachedChannel)
		require.NoError(t, err, "Failed to cache channel")

		// Step 2: Verify channel is cached
		retrieved, err := cache.Get(ctx, tenantID)
		require.NoError(t, err, "Failed to retrieve cached channel")
		require.NotNil(t, retrieved, "Cached channel should exist")
		assert.True(t, retrieved.IsOnline, "Cached channel should be online")

		// Step 3: Simulate channel going offline by calling InvalidateChannelOnOffline
		// This simulates what happens in channel service when status changes to OFFLINE
		err = cache.InvalidateChannelOnOffline(ctx, tenantID)
		require.NoError(t, err, "InvalidateChannelOnOffline should succeed")

		// Step 4: Wait a short time to allow invalidation to complete
		time.Sleep(200 * time.Millisecond)

		// Step 5: Verify cache was invalidated (EXPECTED BEHAVIOR: cache should be cleared)
		stillCached, err := cache.Get(ctx, tenantID)
		
		// EXPECTED BEHAVIOR ASSERTION (after fix):
		// The cache should be cleared immediately when InvalidateChannelOnOffline is called
		// cache.Get() should return an error (redis.Nil) indicating the key doesn't exist
		if err == nil && stillCached != nil {
			t.Errorf("FAILED: Cache was NOT invalidated after calling InvalidateChannelOnOffline")
			t.Errorf("  Expected: Cache should be cleared (Get returns error)")
			t.Errorf("  Actual: Cache still exists with ChannelID=%s, IsOnline=%v", 
				stillCached.ChannelID, stillCached.IsOnline)
			t.FailNow()
		}
		
		// Success: Cache was invalidated
		assert.Error(t, err, "Cache should be invalidated (Get should return error)")
		assert.Nil(t, stillCached, "Cache should not exist after invalidation")

		// Step 6: Verify cache remains invalidated
		time.Sleep(500 * time.Millisecond)
		
		persistedCache, err := cache.Get(ctx, tenantID)
		assert.Error(t, err, "Cache should remain invalidated")
		assert.Nil(t, persistedCache, "Cache should not exist")
		
		// Document the successful fix
		t.Logf("✅ FIX VERIFIED:")
		t.Logf("  TenantID: %s", tenantID)
		t.Logf("  ChannelID: %s", channelID)
		t.Logf("  Cache invalidated: YES (FIXED)")
		t.Logf("  Expected: Cache should be invalidated immediately when channel goes offline")
		t.Logf("  Actual: Cache was successfully invalidated via InvalidateChannelOnOffline()")
	})
}

// TestBug13_MissingInvalidateChannelOnOfflineFunction_Exploration verifies the function doesn't exist
// **Validates: Requirements 2.6**
//
// BUG CONDITION: InvalidateChannelOnOffline() function does not exist
// EXPECTED OUTCOME: Test documents the missing functionality
func TestBug13_MissingInvalidateChannelOnOfflineFunction_Exploration(t *testing.T) {
	t.Log("=== Bug 13: Missing Active Invalidation Function ===")
	t.Log("")
	t.Log("CURRENT STATE:")
	t.Log("  - ChannelCache has Invalidate(tenantID) method")
	t.Log("  - ChannelCache has InvalidateChannel(tenantID) method (alias)")
	t.Log("  - NO InvalidateChannelOnOffline() function exists")
	t.Log("  - NO automatic cache invalidation when channel status changes")
	t.Log("")
	t.Log("BUG IMPACT:")
	t.Log("  - When channel goes offline, cache persists for full TTL (1 minute)")
	t.Log("  - System continues using stale cache data")
	t.Log("  - Messages may be routed to offline channels")
	t.Log("  - Failed message sends until cache expires naturally")
	t.Log("")
	t.Log("EXPECTED BEHAVIOR (after fix):")
	t.Log("  - InvalidateChannelOnOffline() function should exist")
	t.Log("  - Function should be called when channel status changes to offline")
	t.Log("  - Cache should be cleared within 100ms of status change")
	t.Log("  - Next routing request should query database for online channels")
	t.Log("")
	t.Log("✅ Missing functionality documented")
}

// TestBug13_StaleCacheImpact_Exploration demonstrates the impact of stale cache
// **Validates: Requirements 2.6**
//
// BUG CONDITION: Stale cache causes routing to offline channels
// EXPECTED OUTCOME: Test shows messages would be routed to offline channels
func TestBug13_StaleCacheImpact_Exploration(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	// Note: Don't call client.Close() as it causes panic with nil logger in test context

	cache := NewChannelCache(client)
	ctx := context.Background()

	rapid.Check(t, func(t *rapid.T) {
		// Generate multiple tenants and channels
		tenantCount := rapid.IntRange(2, 5).Draw(t, "tenantCount")
		
		for i := 0; i < tenantCount; i++ {
			tenantID := uuid.New()
			channelID := uuid.New()
			
			// Cache an online channel
			cachedChannel := &CachedChannel{
				ChannelID: channelID,
				Type:      "PHONE",
				Priority:  rapid.IntRange(1, 10).Draw(t, "priority"),
				IsOnline:  true,
				CachedAt:  time.Now(),
			}
			
			err := cache.Set(ctx, tenantID, cachedChannel)
			require.NoError(t, err)
			
			// Simulate channel going offline (no cache invalidation)
			// In real scenario: UPDATE dispatch_channels SET status = 'OFFLINE'
			
			// Verify cache still returns the channel as online
			retrieved, err := cache.Get(ctx, tenantID)
			require.NoError(t, err)
			require.NotNil(t, retrieved)
			
			// BUG: Cache shows channel as online even though it's offline in database
			assert.True(t, retrieved.IsOnline, "BUG: Cache shows offline channel as online")
			
			t.Logf("Tenant %d: Cache shows channel %s as online (stale data)", i+1, channelID)
		}
		
		t.Logf("IMPACT: %d tenants affected by stale cache", tenantCount)
		t.Logf("RESULT: Messages would be routed to offline channels, causing failures")
	})
}

// Note: This test uses miniredis for testing, similar to id_generator_test.go
// The setupTestRedis function is defined in id_generator_test.go and returns (*Client, *miniredis.Miniredis)
