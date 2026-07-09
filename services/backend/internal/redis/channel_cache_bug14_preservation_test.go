package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// Bug 14 Preservation Tests
// **Validates: Requirements 3.4**
//
// **Property 2: Preservation** - Channel Cache Performance
//
// These tests observe and verify that channel cache operations for ACTIVE channels
// continue to work correctly with the same TTL (1 minute) and performance.
// They test on UNFIXED code to establish baseline behavior, and must PASS on both
// UNFIXED and FIXED code.
//
// IMPORTANT: These tests verify that implementing active invalidation for offline
// channels does NOT break existing cache functionality for active channels.

// setupBug14TestRedis creates a test Redis client with miniredis
func setupBug14TestRedis(t *testing.T) (*Client, *miniredis.Miniredis) {
	mr := miniredis.RunT(t)
	
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return &Client{
		Client: client,
	}, mr
}

// TestProperty_Bug14_Preservation_ActiveChannelCacheTTL verifies cache TTL for active channels
// **Validates: Requirements 3.4**
//
// This test MUST PASS on both UNFIXED and FIXED code.
// Tests that active channels maintain the same TTL (1 minute) behavior.
func TestProperty_Bug14_Preservation_ActiveChannelCacheTTL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	client, mr := setupBug14TestRedis(t)
	defer mr.Close()
	ctx := context.Background()

	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		channelID := uuid.New()
		gatewayID := uuid.New()

		// Create channel cache
		channelCache := NewChannelCache(client)

		// Observe: Current TTL is 1 minute (from channel_cache.go:33)
		expectedTTL := 1 * time.Minute

		// Create an ACTIVE channel to cache
		channel := &CachedChannel{
			ChannelID: channelID,
			GatewayID: &gatewayID,
			Type:      "WEBSOCKET",
			Priority:  rapid.IntRange(1, 10).Draw(t, "priority"),
			IsOnline:  true, // ACTIVE channel
			CachedAt:  time.Now(),
		}

		// ACT: Set active channel in cache
		err := channelCache.Set(ctx, tenantID, channel)
		require.NoError(t, err, "Preservation: Cache Set should succeed for active channel")

		// ACT: Verify channel is cached immediately
		cachedChannel, err := channelCache.Get(ctx, tenantID)

		// ASSERT: Cache hit should work for active channels
		// On UNFIXED code: PASSES (cache operations work for active channels)
		// On FIXED code: PASSES (cache operations still work for active channels)
		assert.NoError(t, err,
			"Preservation: Cache Get should succeed for active channel. "+
				"This test verifies that implementing active invalidation for offline channels "+
				"does not break cache behavior for active channels.")

		if err == nil {
			assert.Equal(t, channelID, cachedChannel.ChannelID,
				"Preservation: Cached channel ID should match")
			assert.True(t, cachedChannel.IsOnline,
				"Preservation: Cached channel should be online (active)")
		}

		// Observe: Verify TTL is set correctly (1 minute)
		key := channelCache.getCacheKey(tenantID)
		ttl := mr.TTL(key)
		
		// ASSERT: TTL should be 1 minute for active channels
		// On UNFIXED code: PASSES (TTL is 1 minute)
		// On FIXED code: PASSES (TTL remains 1 minute for active channels)
		assert.InDelta(t, expectedTTL.Seconds(), ttl.Seconds(), 2.0,
			"Preservation: Cache TTL should be 1 minute for active channels. "+
				"This test verifies that active invalidation does not change TTL for active channels. "+
				"Expected: %v, Got: %v", expectedTTL, ttl)

		// Cleanup
		channelCache.Invalidate(ctx, tenantID)
	})
}

// TestProperty_Bug14_Preservation_ActiveChannelCacheHitRate verifies cache hit rates
// **Validates: Requirements 3.4**
//
// This test MUST PASS on both UNFIXED and FIXED code.
// Tests that cache hit rates for active channels remain high.
func TestProperty_Bug14_Preservation_ActiveChannelCacheHitRate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	client, mr := setupBug14TestRedis(t)
	defer mr.Close()
	ctx := context.Background()

	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		channelID := uuid.New()
		gatewayID := uuid.New()

		// Create channel cache
		channelCache := NewChannelCache(client)

		// Create an ACTIVE channel
		channel := &CachedChannel{
			ChannelID: channelID,
			GatewayID: &gatewayID,
			Type:      "WEBSOCKET",
			Priority:  1,
			IsOnline:  true, // ACTIVE channel
			CachedAt:  time.Now(),
		}

		// Set channel in cache
		err := channelCache.Set(ctx, tenantID, channel)
		require.NoError(t, err)

		// ACT: Perform multiple cache reads
		iterations := rapid.IntRange(10, 50).Draw(t, "iterations")
		hits := 0
		misses := 0

		for i := 0; i < iterations; i++ {
			_, err := channelCache.Get(ctx, tenantID)
			if err == nil {
				hits++
			} else {
				misses++
			}
		}

		// ASSERT: Cache hit rate should be 100% for active channels
		// On UNFIXED code: PASSES (cache hits work consistently)
		// On FIXED code: PASSES (cache hits still work consistently)
		hitRate := float64(hits) / float64(iterations) * 100
		assert.Equal(t, 100.0, hitRate,
			"Preservation: Cache hit rate should be 100%% for active channels. "+
				"This test verifies that active invalidation does not affect cache hit rates for active channels. "+
				"Hits: %d, Misses: %d, Rate: %.2f%%", hits, misses, hitRate)

		// Cleanup
		channelCache.Invalidate(ctx, tenantID)
	})
}

// TestProperty_Bug14_Preservation_ActiveChannelCachePerformance verifies cache performance
// **Validates: Requirements 3.4**
//
// This test MUST PASS on both UNFIXED and FIXED code.
// Tests that cache operations for active channels maintain same performance.
func TestProperty_Bug14_Preservation_ActiveChannelCachePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	client, mr := setupBug14TestRedis(t)
	defer mr.Close()
	ctx := context.Background()

	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		channelID := uuid.New()
		gatewayID := uuid.New()

		// Create channel cache
		channelCache := NewChannelCache(client)

		// Create an ACTIVE channel
		channel := &CachedChannel{
			ChannelID: channelID,
			GatewayID: &gatewayID,
			Type:      "WEBSOCKET",
			Priority:  1,
			IsOnline:  true, // ACTIVE channel
			CachedAt:  time.Now(),
		}

		// Set channel in cache
		err := channelCache.Set(ctx, tenantID, channel)
		require.NoError(t, err)

		// ACT: Measure cache read performance
		iterations := 100
		start := time.Now()
		for i := 0; i < iterations; i++ {
			_, err := channelCache.Get(ctx, tenantID)
			require.NoError(t, err)
		}
		duration := time.Since(start)
		avgTime := duration / time.Duration(iterations)

		// ASSERT: Cache operations should be fast (< 10ms average)
		// On UNFIXED code: PASSES (cache is fast)
		// On FIXED code: PASSES (cache remains fast for active channels)
		assert.Less(t, avgTime, 10*time.Millisecond,
			"Preservation: Cache read should be fast (< 10ms average) for active channels. "+
				"This test verifies that active invalidation does not degrade cache performance. "+
				"Average time: %v", avgTime)

		t.Logf("Active channel cache performance: %d iterations in %v (avg: %v per read)",
			iterations, duration, avgTime)

		// Cleanup
		channelCache.Invalidate(ctx, tenantID)
	})
}

// TestProperty_Bug14_Preservation_ActiveChannelCacheUpdate verifies cache update mechanism
// **Validates: Requirements 3.4**
//
// This test MUST PASS on both UNFIXED and FIXED code.
// Tests that cache update mechanism for active channels continues to work.
func TestProperty_Bug14_Preservation_ActiveChannelCacheUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	client, mr := setupBug14TestRedis(t)
	defer mr.Close()
	ctx := context.Background()

	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		channelID1 := uuid.New()
		channelID2 := uuid.New()
		gatewayID := uuid.New()

		// Create channel cache
		channelCache := NewChannelCache(client)

		// Create first ACTIVE channel
		channel1 := &CachedChannel{
			ChannelID: channelID1,
			GatewayID: &gatewayID,
			Type:      "WEBSOCKET",
			Priority:  1,
			IsOnline:  true, // ACTIVE channel
			CachedAt:  time.Now(),
		}

		// Set first channel
		err := channelCache.Set(ctx, tenantID, channel1)
		require.NoError(t, err)

		// Verify first channel is cached
		cached, err := channelCache.Get(ctx, tenantID)
		require.NoError(t, err)
		require.Equal(t, channelID1, cached.ChannelID)

		// ACT: Update cache with second ACTIVE channel
		channel2 := &CachedChannel{
			ChannelID: channelID2,
			GatewayID: &gatewayID,
			Type:      "WEBSOCKET",
			Priority:  2,
			IsOnline:  true, // ACTIVE channel
			CachedAt:  time.Now(),
		}

		err = channelCache.Set(ctx, tenantID, channel2)
		assert.NoError(t, err, "Preservation: Cache update should succeed for active channel")

		// ACT: Get updated channel
		updatedChannel, err := channelCache.Get(ctx, tenantID)

		// ASSERT: Cache should return updated active channel
		// On UNFIXED code: PASSES (cache update works)
		// On FIXED code: PASSES (cache update still works for active channels)
		assert.NoError(t, err,
			"Preservation: Cache Get should succeed after update")

		if err == nil {
			assert.Equal(t, channelID2, updatedChannel.ChannelID,
				"Preservation: Cache should return updated channel ID. "+
					"This test verifies that active invalidation does not break cache updates for active channels.")
			assert.Equal(t, 2, updatedChannel.Priority,
				"Preservation: Updated channel priority should match")
			assert.True(t, updatedChannel.IsOnline,
				"Preservation: Updated channel should still be active")
		}

		// Cleanup
		channelCache.Invalidate(ctx, tenantID)
	})
}

// TestProperty_Bug14_Preservation_MultipleActiveChannelsCache verifies multi-tenant cache
// **Validates: Requirements 3.4**
//
// This test MUST PASS on both UNFIXED and FIXED code.
// Tests that multiple active channels are cached independently per tenant.
func TestProperty_Bug14_Preservation_MultipleActiveChannelsCache(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	client, mr := setupBug14TestRedis(t)
	defer mr.Close()
	ctx := context.Background()

	rapid.Check(t, func(t *rapid.T) {
		// Generate multiple tenants with active channels
		tenantCount := rapid.IntRange(2, 5).Draw(t, "tenantCount")
		tenants := make([]uuid.UUID, tenantCount)
		channels := make([]*CachedChannel, tenantCount)

		// Create channel cache
		channelCache := NewChannelCache(client)

		// Cache different ACTIVE channels for each tenant
		for i := 0; i < tenantCount; i++ {
			tenants[i] = uuid.New()
			gatewayID := uuid.New()
			channels[i] = &CachedChannel{
				ChannelID: uuid.New(),
				GatewayID: &gatewayID,
				Type:      "WEBSOCKET",
				Priority:  i + 1,
				IsOnline:  true, // ACTIVE channel
				CachedAt:  time.Now(),
			}

			err := channelCache.Set(ctx, tenants[i], channels[i])
			require.NoError(t, err, "Failed to cache active channel for tenant %d", i)
		}

		// ACT: Verify each tenant gets their own cached active channel
		for i := 0; i < tenantCount; i++ {
			cachedChannel, err := channelCache.Get(ctx, tenants[i])

			// ASSERT: Each tenant should get their own active channel
			// On UNFIXED code: PASSES (multi-tenant cache isolation works)
			// On FIXED code: PASSES (multi-tenant cache isolation still works)
			assert.NoError(t, err,
				"Preservation: Cache should hit for tenant %d with active channel", i)

			if err == nil {
				assert.Equal(t, channels[i].ChannelID, cachedChannel.ChannelID,
					"Preservation: Tenant %d should get their own cached active channel. "+
						"This test verifies that active invalidation does not break multi-tenant cache isolation.",
					i)
				assert.True(t, cachedChannel.IsOnline,
					"Preservation: Cached channel should be active for tenant %d", i)
			}
		}

		// Cleanup
		for i := 0; i < tenantCount; i++ {
			channelCache.Invalidate(ctx, tenants[i])
		}
	})
}

// Example-based preservation test: Active channel cache behavior
// **Validates: Requirements 3.4**
//
// This test provides a concrete example of preservation behavior for active channels.
// EXPECTED OUTCOME on UNFIXED code: PASSES (existing cache functionality works)
func TestBug14_Preservation_ActiveChannelCacheBehavior(t *testing.T) {
	client, mr := setupBug14TestRedis(t)
	defer mr.Close()
	ctx := context.Background()

	tenantID := uuid.New()
	channelID := uuid.New()
	gatewayID := uuid.New()

	// Create channel cache
	channelCache := NewChannelCache(client)

	// Observe: Current TTL is 1 minute
	expectedTTL := 1 * time.Minute

	// Create an ACTIVE channel to cache
	channel := &CachedChannel{
		ChannelID: channelID,
		GatewayID: &gatewayID,
		Type:      "WEBSOCKET",
		Priority:  1,
		IsOnline:  true, // ACTIVE channel
		CachedAt:  time.Now(),
	}

	// Set active channel in cache
	err := channelCache.Set(ctx, tenantID, channel)
	require.NoError(t, err, "Preservation: Cache Set should succeed for active channel")

	// Get active channel from cache
	cachedChannel, err := channelCache.Get(ctx, tenantID)
	require.NoError(t, err, "Preservation: Cache Get should succeed for active channel")
	require.NotNil(t, cachedChannel, "Preservation: Cached channel should not be nil")

	// Verify channel content
	assert.Equal(t, channelID, cachedChannel.ChannelID,
		"Preservation: Cached channel ID should match")
	assert.Equal(t, "WEBSOCKET", cachedChannel.Type,
		"Preservation: Channel type should be preserved")
	assert.Equal(t, 1, cachedChannel.Priority,
		"Preservation: Channel priority should be preserved")
	assert.True(t, cachedChannel.IsOnline,
		"Preservation: Channel should be active (online)")

	// Verify TTL is 1 minute
	key := channelCache.getCacheKey(tenantID)
	ttl := mr.TTL(key)
	assert.InDelta(t, expectedTTL.Seconds(), ttl.Seconds(), 2.0,
		"Preservation: Cache TTL should be 1 minute for active channels. "+
			"Expected: %v, Got: %v", expectedTTL, ttl)

	t.Logf("Active channel cache TTL: %v (expected: %v)", ttl, expectedTTL)

	// Cleanup
	channelCache.Invalidate(ctx, tenantID)
}

// TestBug14_Preservation_ActiveChannelCacheConsistency verifies cache consistency
// **Validates: Requirements 3.4**
//
// This test MUST PASS on both UNFIXED and FIXED code.
// Tests that cache returns consistent data for active channels across multiple reads.
func TestBug14_Preservation_ActiveChannelCacheConsistency(t *testing.T) {
	client, mr := setupBug14TestRedis(t)
	defer mr.Close()
	ctx := context.Background()

	tenantID := uuid.New()
	channelID := uuid.New()
	gatewayID := uuid.New()

	// Create channel cache
	channelCache := NewChannelCache(client)

	// Create an ACTIVE channel
	channel := &CachedChannel{
		ChannelID: channelID,
		GatewayID: &gatewayID,
		Type:      "WEBSOCKET",
		Priority:  5,
		IsOnline:  true, // ACTIVE channel
		CachedAt:  time.Now(),
	}

	// Set channel in cache
	err := channelCache.Set(ctx, tenantID, channel)
	require.NoError(t, err)

	// ACT: Read cache multiple times
	reads := 10
	for i := 0; i < reads; i++ {
		cachedChannel, err := channelCache.Get(ctx, tenantID)
		
		// ASSERT: Each read should return consistent data
		// On UNFIXED code: PASSES (cache is consistent)
		// On FIXED code: PASSES (cache remains consistent for active channels)
		assert.NoError(t, err,
			"Preservation: Cache read %d should succeed", i+1)
		
		if err == nil {
			assert.Equal(t, channelID, cachedChannel.ChannelID,
				"Preservation: Channel ID should be consistent across reads (read %d)", i+1)
			assert.Equal(t, 5, cachedChannel.Priority,
				"Preservation: Channel priority should be consistent across reads (read %d)", i+1)
			assert.True(t, cachedChannel.IsOnline,
				"Preservation: Channel should remain active across reads (read %d)", i+1)
		}
	}

	t.Logf("Active channel cache consistency verified across %d reads", reads)

	// Cleanup
	channelCache.Invalidate(ctx, tenantID)
}
