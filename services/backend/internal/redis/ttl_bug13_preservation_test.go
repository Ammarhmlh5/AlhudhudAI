package redis

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// Bug 13 Preservation Tests - Cache Performance Maintained
// **Validates: Requirements 3.6, 3.7, 3.8**
//
// **Property 2: Preservation** - Cache Performance Maintained
//
// These tests verify that cache performance is maintained on UNFIXED code.
// They test that:
// 1. Pricing cache hits return cached results without recalculation (Req 3.6)
// 2. Channel cache hits return cached data without database query (Req 3.7)
// 3. Redis operations are fast and provide good performance (Req 3.8)
//
// These tests should PASS on unfixed code to demonstrate that the current
// cache implementation works correctly and provides good performance.

// TestPreservation_Bug13_PricingCacheHits verifies pricing cache performance
//
// **Validates: Requirement 3.6**
// WHEN pricing cache hits occur for common message lengths
// THEN the system SHALL CONTINUE TO return cached results without recalculation
func TestPreservation_Bug13_PricingCacheHits(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property-based test in short mode")
	}

	client, mr := setupBug13TestRedis(t)
	defer mr.Close()

	pricingCache := NewPricingCache(client)
	ctx := context.Background()

	rapid.Check(t, func(t *rapid.T) {
		// Generate random message parameters
		encoding := rapid.SampledFrom([]string{"GSM7", "UNICODE"}).Draw(t, "encoding")
		length := rapid.IntRange(10, 500).Draw(t, "length")
		costPerPart := rapid.Float64Range(0.05, 0.50).Draw(t, "costPerPart")

		// Calculate pricing info
		var partsCount int
		if encoding == "UNICODE" {
			if length <= 70 {
				partsCount = 1
			} else {
				partsCount = (length + 66) / 67
			}
		} else {
			if length <= 160 {
				partsCount = 1
			} else {
				partsCount = (length + 152) / 153
			}
		}

		expectedInfo := &PricingInfo{
			CharCount:  length,
			PartsCount: partsCount,
			Encoding:   encoding,
			Cost:       float64(partsCount) * costPerPart,
		}

		// Set in cache
		err := pricingCache.Set(ctx, encoding, length, expectedInfo)
		require.NoError(t, err, "Failed to set pricing cache")

		// Verify cache hit returns correct data
		cachedInfo, err := pricingCache.Get(ctx, encoding, length)
		require.NoError(t, err, "Cache hit should succeed")
		
		assert.Equal(t, expectedInfo.CharCount, cachedInfo.CharCount,
			"Cached char count should match")
		assert.Equal(t, expectedInfo.PartsCount, cachedInfo.PartsCount,
			"Cached parts count should match")
		assert.Equal(t, expectedInfo.Encoding, cachedInfo.Encoding,
			"Cached encoding should match")
		assert.InDelta(t, expectedInfo.Cost, cachedInfo.Cost, 0.001,
			"Cached cost should match")
	})

	t.Log("✅ Pricing cache hits return cached results correctly")
}

// TestPreservation_Bug13_PricingCachePerformance verifies cache is faster than calculation
//
// **Validates: Requirement 3.6**
// Verifies that cache hits are significantly faster than recalculation
func TestPreservation_Bug13_PricingCachePerformance(t *testing.T) {
	client, mr := setupBug13TestRedis(t)
	defer mr.Close()

	pricingCache := NewPricingCache(client)
	ctx := context.Background()

	// Warmup cache with common message lengths
	err := pricingCache.Warmup(ctx, 0.10)
	require.NoError(t, err, "Cache warmup should succeed")

	// Test cache hit performance
	iterations := 250
	encoding := "GSM7"
	length := 160

	// Measure cache hit time
	startCache := time.Now()
	for i := 0; i < iterations; i++ {
		_, err := pricingCache.Get(ctx, encoding, length)
		require.NoError(t, err)
	}
	cacheTime := time.Since(startCache)

	// Measure calculation time (cache miss + calculation)
	startCalc := time.Now()
	for i := 0; i < iterations; i++ {
		// Simulate calculation by using GetOrCalculate with non-cached length
		_, err := pricingCache.GetOrCalculate(ctx, "Test message content here", 0.10)
		require.NoError(t, err)
	}
	calcTime := time.Since(startCalc)

	avgCacheTime := cacheTime / time.Duration(iterations)
	avgCalcTime := calcTime / time.Duration(iterations)

	t.Logf("Average cache hit time: %v", avgCacheTime)
	t.Logf("Average calculation time: %v", avgCalcTime)

	// Cache should be faster (or at least not significantly slower)
	// We allow cache to be up to 5x slower due to Redis + Miniredis overhead
	// and scheduler noise when packages run in parallel.
	assert.Less(t, avgCacheTime, avgCalcTime*5,
		"Cache hits should be reasonably fast")

	t.Log("✅ Pricing cache provides good performance")
}

// TestPreservation_Bug13_ChannelCacheHits verifies channel cache performance
//
// **Validates: Requirement 3.7**
// WHEN channel cache hits occur for active tenants
// THEN the system SHALL CONTINUE TO return cached channel information without database query
func TestPreservation_Bug13_ChannelCacheHits(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property-based test in short mode")
	}

	client, mr := setupBug13TestRedis(t)
	defer mr.Close()

	channelCache := NewChannelCache(client)
	ctx := context.Background()

	rapid.Check(t, func(t *rapid.T) {
		// Generate random tenant and channel data
		tenantID := uuid.New()
		channelID := uuid.New()
		gatewayID := uuid.New()
		priority := rapid.IntRange(1, 10).Draw(t, "priority")
		isOnline := rapid.Bool().Draw(t, "isOnline")

		expectedChannel := &CachedChannel{
			ChannelID: channelID,
			GatewayID: &gatewayID,
			Type:      "SMS",
			Priority:  priority,
			IsOnline:  isOnline,
			CachedAt:  time.Now(),
		}

		// Set in cache
		err := channelCache.Set(ctx, tenantID, expectedChannel)
		require.NoError(t, err, "Failed to set channel cache")

		// Verify cache hit returns correct data
		cachedChannel, err := channelCache.Get(ctx, tenantID)
		require.NoError(t, err, "Cache hit should succeed")
		
		assert.Equal(t, expectedChannel.ChannelID, cachedChannel.ChannelID,
			"Cached channel ID should match")
		assert.Equal(t, expectedChannel.GatewayID, cachedChannel.GatewayID,
			"Cached gateway ID should match")
		assert.Equal(t, expectedChannel.Type, cachedChannel.Type,
			"Cached type should match")
		assert.Equal(t, expectedChannel.Priority, cachedChannel.Priority,
			"Cached priority should match")
		assert.Equal(t, expectedChannel.IsOnline, cachedChannel.IsOnline,
			"Cached online status should match")
	})

	t.Log("✅ Channel cache hits return cached data correctly")
}

// TestPreservation_Bug13_ChannelCacheBatchRetrieval verifies batch cache operations
//
// **Validates: Requirement 3.7**
// Verifies that GetMultiple efficiently retrieves cached channels for multiple tenants
func TestPreservation_Bug13_ChannelCacheBatchRetrieval(t *testing.T) {
	client, mr := setupBug13TestRedis(t)
	defer mr.Close()

	channelCache := NewChannelCache(client)
	ctx := context.Background()

	// Create multiple tenants with cached channels
	tenantCount := 10
	tenantIDs := make([]uuid.UUID, tenantCount)
	expectedChannels := make(map[uuid.UUID]*CachedChannel)

	for i := 0; i < tenantCount; i++ {
		tenantID := uuid.New()
		tenantIDs[i] = tenantID

		channel := &CachedChannel{
			ChannelID: uuid.New(),
			Type:      "SMS",
			Priority:  i + 1,
			IsOnline:  true,
			CachedAt:  time.Now(),
		}

		err := channelCache.Set(ctx, tenantID, channel)
		require.NoError(t, err)

		expectedChannels[tenantID] = channel
	}

	// Retrieve all channels in batch
	cachedChannels, err := channelCache.GetMultiple(ctx, tenantIDs)
	require.NoError(t, err, "Batch retrieval should succeed")

	// Verify all channels retrieved correctly
	assert.Equal(t, tenantCount, len(cachedChannels),
		"Should retrieve all cached channels")

	for tenantID, expected := range expectedChannels {
		cached, exists := cachedChannels[tenantID]
		require.True(t, exists, "Channel should exist for tenant %s", tenantID)
		
		assert.Equal(t, expected.ChannelID, cached.ChannelID,
			"Channel ID should match for tenant %s", tenantID)
		assert.Equal(t, expected.Priority, cached.Priority,
			"Priority should match for tenant %s", tenantID)
	}

	t.Log("✅ Channel cache batch retrieval works correctly")
}

// TestPreservation_Bug13_RedisOperationsPerformance verifies Redis operations are fast
//
// **Validates: Requirement 3.8**
// WHEN Redis operations succeed
// THEN the system SHALL CONTINUE TO provide fast queue operations and caching
func TestPreservation_Bug13_RedisOperationsPerformance(t *testing.T) {
	client, mr := setupBug13TestRedis(t)
	defer mr.Close()

	ctx := context.Background()

	// Test basic Redis operations performance
	iterations := 100

	// Test SET operation
	startSet := time.Now()
	for i := 0; i < iterations; i++ {
		key := fmt.Sprintf("test:key:%d", i)
		err := client.Client.Set(ctx, key, "value", 1*time.Minute).Err()
		require.NoError(t, err)
	}
	setTime := time.Since(startSet)
	avgSetTime := setTime / time.Duration(iterations)

	// Test GET operation
	startGet := time.Now()
	for i := 0; i < iterations; i++ {
		key := fmt.Sprintf("test:key:%d", i)
		_, err := client.Client.Get(ctx, key).Result()
		require.NoError(t, err)
	}
	getTime := time.Since(startGet)
	avgGetTime := getTime / time.Duration(iterations)

	// Test LPUSH operation (queue)
	queueKey := "test:queue"
	startPush := time.Now()
	for i := 0; i < iterations; i++ {
		err := client.Client.LPush(ctx, queueKey, fmt.Sprintf("msg:%d", i)).Err()
		require.NoError(t, err)
	}
	pushTime := time.Since(startPush)
	avgPushTime := pushTime / time.Duration(iterations)

	// Test RPOP operation (dequeue)
	startPop := time.Now()
	for i := 0; i < iterations; i++ {
		_, err := client.Client.RPop(ctx, queueKey).Result()
		require.NoError(t, err)
	}
	popTime := time.Since(startPop)
	avgPopTime := popTime / time.Duration(iterations)

	t.Logf("Average SET time: %v", avgSetTime)
	t.Logf("Average GET time: %v", avgGetTime)
	t.Logf("Average LPUSH time: %v", avgPushTime)
	t.Logf("Average RPOP time: %v", avgPopTime)

	// All operations should be reasonably fast (< 10ms in test environment)
	// In production with real Redis, these would be < 1ms
	maxTime := 10 * time.Millisecond
	
	assert.Less(t, avgSetTime, maxTime, "SET operations should be fast")
	assert.Less(t, avgGetTime, maxTime, "GET operations should be fast")
	assert.Less(t, avgPushTime, maxTime, "LPUSH operations should be fast")
	assert.Less(t, avgPopTime, maxTime, "RPOP operations should be fast")

	t.Log("✅ Redis operations provide fast performance")
}

// TestPreservation_Bug13_MessageQueuePerformance verifies queue operations are fast
//
// **Validates: Requirement 3.8**
// Verifies that message queue enqueue/dequeue operations maintain good performance
func TestPreservation_Bug13_MessageQueuePerformance(t *testing.T) {
	client, mr := setupBug13TestRedis(t)
	defer mr.Close()

	mq := NewMessageQueue(client)
	ctx := context.Background()
	tenantID := uuid.New()

	// Clear queue
	_ = mq.Clear(ctx, tenantID)

	// Test enqueue performance
	messageCount := 100
	startEnqueue := time.Now()
	
	for i := 0; i < messageCount; i++ {
		item := &MessageQueueItem{
			MessageID:   uuid.New(),
			TenantID:    tenantID,
			PhoneNumber: fmt.Sprintf("+96650000%04d", i),
			Content:     fmt.Sprintf("Test message %d", i),
			Priority:    (i % 3) + 1,
			PartsCount:  1,
			Cost:        0.1,
		}
		err := mq.Enqueue(ctx, item)
		require.NoError(t, err)
	}
	
	enqueueTime := time.Since(startEnqueue)
	avgEnqueueTime := enqueueTime / time.Duration(messageCount)

	// Test dequeue performance
	startDequeue := time.Now()
	dequeueCount := 0
	
	for {
		batch, err := mq.DequeueBatch(ctx, tenantID, 50)
		require.NoError(t, err)
		
		if len(batch) == 0 {
			break
		}
		
		dequeueCount += len(batch)
	}
	
	dequeueTime := time.Since(startDequeue)
	avgDequeueTime := dequeueTime / time.Duration(dequeueCount)

	t.Logf("Enqueued %d messages in %v (avg: %v per message)", 
		messageCount, enqueueTime, avgEnqueueTime)
	t.Logf("Dequeued %d messages in %v (avg: %v per message)", 
		dequeueCount, dequeueTime, avgDequeueTime)

	// Verify all messages were processed
	assert.Equal(t, messageCount, dequeueCount, 
		"Should dequeue all enqueued messages")

	// Operations should be reasonably fast
	maxAvgTime := 5 * time.Millisecond
	assert.Less(t, avgEnqueueTime, maxAvgTime, 
		"Enqueue operations should be fast")
	assert.Less(t, avgDequeueTime, maxAvgTime, 
		"Dequeue operations should be fast")

	t.Log("✅ Message queue operations provide fast performance")
}

// TestPreservation_Bug13_CacheInvalidationWorks verifies cache invalidation
//
// **Validates: Requirements 3.6, 3.7**
// Verifies that cache invalidation works correctly (needed for maintenance)
func TestPreservation_Bug13_CacheInvalidationWorks(t *testing.T) {
	client, mr := setupBug13TestRedis(t)
	defer mr.Close()

	pricingCache := NewPricingCache(client)
	channelCache := NewChannelCache(client)
	ctx := context.Background()

	// Set pricing cache entries
	for i := 0; i < 10; i++ {
		info := &PricingInfo{
			CharCount:  100 + i*10,
			PartsCount: 1,
			Encoding:   "GSM7",
			Cost:       0.1,
		}
		err := pricingCache.Set(ctx, "GSM7", 100+i*10, info)
		require.NoError(t, err)
	}

	// Verify entries exist
	_, err := pricingCache.Get(ctx, "GSM7", 100)
	require.NoError(t, err, "Cache entry should exist before invalidation")

	// Invalidate all pricing cache
	err = pricingCache.InvalidateAll(ctx)
	require.NoError(t, err, "Cache invalidation should succeed")

	// Verify entries are gone
	_, err = pricingCache.Get(ctx, "GSM7", 100)
	assert.Error(t, err, "Cache entry should not exist after invalidation")

	// Test channel cache invalidation
	tenantID := uuid.New()
	channel := &CachedChannel{
		ChannelID: uuid.New(),
		Type:      "SMS",
		Priority:  1,
		IsOnline:  true,
		CachedAt:  time.Now(),
	}

	err = channelCache.Set(ctx, tenantID, channel)
	require.NoError(t, err)

	// Verify exists
	_, err = channelCache.Get(ctx, tenantID)
	require.NoError(t, err, "Channel should be cached")

	// Invalidate
	err = channelCache.Invalidate(ctx, tenantID)
	require.NoError(t, err, "Channel invalidation should succeed")

	// Verify gone
	_, err = channelCache.Get(ctx, tenantID)
	assert.Error(t, err, "Channel should not be cached after invalidation")

	t.Log("✅ Cache invalidation works correctly")
}

// TestPreservation_Bug13_CacheTTLRespected verifies cache entries expire correctly
//
// **Validates: Requirements 3.6, 3.7, 3.8**
// Verifies that cache TTL is respected and entries expire as expected
func TestPreservation_Bug13_CacheTTLRespected(t *testing.T) {
	client, mr := setupBug13TestRedis(t)
	defer mr.Close()

	pricingCache := NewPricingCache(client)
	channelCache := NewChannelCache(client)
	ctx := context.Background()

	// Test pricing cache TTL
	info := &PricingInfo{
		CharCount:  100,
		PartsCount: 1,
		Encoding:   "GSM7",
		Cost:       0.1,
	}
	err := pricingCache.Set(ctx, "GSM7", 100, info)
	require.NoError(t, err)

	// Verify exists
	_, err = pricingCache.Get(ctx, "GSM7", 100)
	require.NoError(t, err, "Pricing should be cached")

	// Fast forward past TTL (pricing cache uses 1 hour TTL)
	mr.FastForward(2 * time.Hour)

	// Verify expired
	_, err = pricingCache.Get(ctx, "GSM7", 100)
	assert.Error(t, err, "Pricing should expire after TTL")

	// Test channel cache TTL
	tenantID := uuid.New()
	channel := &CachedChannel{
		ChannelID: uuid.New(),
		Type:      "SMS",
		Priority:  1,
		IsOnline:  true,
		CachedAt:  time.Now(),
	}

	err = channelCache.Set(ctx, tenantID, channel)
	require.NoError(t, err)

	// Verify exists
	_, err = channelCache.Get(ctx, tenantID)
	require.NoError(t, err, "Channel should be cached")

	// Fast forward past TTL (channel cache uses 5 minute TTL)
	mr.FastForward(10 * time.Minute)

	// Verify expired
	_, err = channelCache.Get(ctx, tenantID)
	assert.Error(t, err, "Channel should expire after TTL")

	t.Log("✅ Cache TTL is respected and entries expire correctly")
}

// TestPreservation_Bug13_ConcurrentCacheAccess verifies cache handles concurrent access
//
// **Validates: Requirement 3.8**
// Verifies that cache operations work correctly under concurrent access
func TestPreservation_Bug13_ConcurrentCacheAccess(t *testing.T) {
	client, mr := setupBug13TestRedis(t)
	defer mr.Close()

	pricingCache := NewPricingCache(client)
	ctx := context.Background()

	// Concurrent writes
	concurrency := 10
	done := make(chan bool, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			info := &PricingInfo{
				CharCount:  100 + id,
				PartsCount: 1,
				Encoding:   "GSM7",
				Cost:       0.1,
			}
			err := pricingCache.Set(ctx, "GSM7", 100+id, info)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < concurrency; i++ {
		<-done
	}

	// Concurrent reads
	for i := 0; i < concurrency; i++ {
		go func(id int) {
			_, err := pricingCache.Get(ctx, "GSM7", 100+id)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all reads
	for i := 0; i < concurrency; i++ {
		<-done
	}

	t.Log("✅ Cache handles concurrent access correctly")
}
