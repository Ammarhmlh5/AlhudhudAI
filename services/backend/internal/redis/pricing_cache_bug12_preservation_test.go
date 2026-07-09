package redis

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBug12_Preservation_CacheInvalidationWorks tests preservation property for Bug 12
//
// **Validates: Requirements 3.6, 3.7**
//
// **Property 2: Preservation** - Cache Invalidation Works
//
// This test verifies that cache invalidation continues to work correctly
// after any enhancements to the SCAN implementation. The core functionality
// of clearing all pricing cache entries must be preserved.
//
// Preservation Requirements:
// - Cache invalidation must delete all matching keys
// - Cache operations must continue to work after invalidation
// - Performance must remain acceptable
//
// **EXPECTED OUTCOME**: Test PASSES on UNFIXED code (current implementation works)
func TestBug12_Preservation_CacheInvalidationWorks(t *testing.T) {
	ctx := context.Background()
	client, mr := setupBug12PreservationTestRedis(t)
	defer mr.Close()

	pc := NewPricingCache(client)

	// Clean up before test
	pattern := fmt.Sprintf("%s:cache:pricing:*", AppNamespace)
	cleanupKeysPreservation(t, ctx, client, pattern)

	// Create test dataset with various pricing entries
	t.Log("Creating test pricing cache entries...")
	testEntries := []struct {
		encoding string
		length   int
		info     *PricingInfo
	}{
		{"GSM7", 50, &PricingInfo{CharCount: 50, PartsCount: 1, Encoding: "GSM7", Cost: 0.05}},
		{"GSM7", 160, &PricingInfo{CharCount: 160, PartsCount: 1, Encoding: "GSM7", Cost: 0.05}},
		{"GSM7", 320, &PricingInfo{CharCount: 320, PartsCount: 3, Encoding: "GSM7", Cost: 0.15}},
		{"UNICODE", 70, &PricingInfo{CharCount: 70, PartsCount: 1, Encoding: "UNICODE", Cost: 0.05}},
		{"UNICODE", 140, &PricingInfo{CharCount: 140, PartsCount: 3, Encoding: "UNICODE", Cost: 0.15}},
	}

	for _, entry := range testEntries {
		err := pc.Set(ctx, entry.encoding, entry.length, entry.info)
		require.NoError(t, err, "Failed to set cache entry")
	}

	// Verify entries exist
	iter := client.Client.Scan(ctx, 0, pattern, 1000).Iterator()
	keysBefore := 0
	for iter.Next(ctx) {
		keysBefore++
	}
	require.NoError(t, iter.Err())
	assert.Equal(t, len(testEntries), keysBefore, "All test entries should be cached")

	// **PRESERVATION TEST**: Call InvalidateAll()
	t.Log("Testing cache invalidation...")
	err := pc.InvalidateAll(ctx)
	require.NoError(t, err, "InvalidateAll should succeed")

	// **VERIFICATION 1**: All keys should be deleted
	iter = client.Client.Scan(ctx, 0, pattern, 1000).Iterator()
	keysAfter := 0
	for iter.Next(ctx) {
		keysAfter++
	}
	require.NoError(t, iter.Err())
	assert.Equal(t, 0, keysAfter, "All pricing cache keys should be deleted")

	// **VERIFICATION 2**: Cache should work after invalidation
	t.Log("Verifying cache works after invalidation...")
	newInfo := &PricingInfo{
		CharCount:  100,
		PartsCount: 1,
		Encoding:   "GSM7",
		Cost:       0.05,
	}
	err = pc.Set(ctx, "GSM7", 100, newInfo)
	require.NoError(t, err, "Cache set should work after invalidation")

	retrieved, err := pc.Get(ctx, "GSM7", 100)
	require.NoError(t, err, "Cache get should work after invalidation")
	assert.Equal(t, newInfo.CharCount, retrieved.CharCount)
	assert.Equal(t, newInfo.Encoding, retrieved.Encoding)

	t.Log("✅ Preservation test passed: Cache invalidation works correctly")
}

// TestBug12_Preservation_CachePerformance tests that cache performance is maintained
//
// **Validates: Requirements 3.6, 3.7**
//
// This test verifies that cache hit rates and performance remain good
// after any SCAN implementation changes
func TestBug12_Preservation_CachePerformance(t *testing.T) {
	ctx := context.Background()
	client, mr := setupBug12PreservationTestRedis(t)
	defer mr.Close()

	pc := NewPricingCache(client)

	// Clean up
	pattern := fmt.Sprintf("%s:cache:pricing:*", AppNamespace)
	cleanupKeysPreservation(t, ctx, client, pattern)

	// Test cache hit performance
	t.Log("Testing cache hit performance...")
	
	// Set a value
	info := &PricingInfo{
		CharCount:  160,
		PartsCount: 1,
		Encoding:   "GSM7",
		Cost:       0.05,
	}
	err := pc.Set(ctx, "GSM7", 160, info)
	require.NoError(t, err)

	// Measure cache hit time (should be fast)
	start := time.Now()
	for i := 0; i < 100; i++ {
		_, err := pc.Get(ctx, "GSM7", 160)
		require.NoError(t, err)
	}
	duration := time.Since(start)
	avgTime := duration / 100

	t.Logf("Average cache hit time: %v", avgTime)
	assert.Less(t, avgTime.Milliseconds(), int64(10), 
		"Cache hits should be fast (< 10ms average)")

	t.Log("✅ Cache performance preserved")
}

// TestBug12_Preservation_LargeDatasetInvalidation tests invalidation with larger datasets
//
// **Validates: Requirements 3.6, 3.7**
//
// This test ensures that invalidation works correctly even with larger
// cache datasets, which is important for production scenarios
func TestBug12_Preservation_LargeDatasetInvalidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large dataset test in short mode")
	}

	ctx := context.Background()
	client, mr := setupBug12PreservationTestRedis(t)
	defer mr.Close()

	pc := NewPricingCache(client)

	// Clean up
	pattern := fmt.Sprintf("%s:cache:pricing:*", AppNamespace)
	cleanupKeysPreservation(t, ctx, client, pattern)

	// Create larger dataset (5000 entries)
	t.Log("Creating 5000 cache entries...")
	const entryCount = 5000
	
	for i := 0; i < entryCount; i++ {
		encoding := "GSM7"
		if i%2 == 0 {
			encoding = "UNICODE"
		}
		length := 50 + (i % 300)
		
		info := &PricingInfo{
			CharCount:  length,
			PartsCount: (length / 160) + 1,
			Encoding:   encoding,
			Cost:       0.05,
		}
		
		err := pc.Set(ctx, encoding, length, info)
		require.NoError(t, err)
	}

	// Verify entries created
	iter := client.Client.Scan(ctx, 0, pattern, 1000).Iterator()
	keyCount := 0
	for iter.Next(ctx) {
		keyCount++
	}
	require.NoError(t, iter.Err())
	t.Logf("Created %d cache keys", keyCount)
	assert.Greater(t, keyCount, 100, "Should have many cache entries")

	// Test invalidation
	t.Log("Testing invalidation of large dataset...")
	start := time.Now()
	err := pc.InvalidateAll(ctx)
	duration := time.Since(start)
	
	require.NoError(t, err, "Invalidation should succeed")
	t.Logf("Invalidation completed in %v", duration)

	// Verify all deleted
	iter = client.Client.Scan(ctx, 0, pattern, 1000).Iterator()
	remainingKeys := 0
	for iter.Next(ctx) {
		remainingKeys++
	}
	require.NoError(t, iter.Err())
	
	assert.Equal(t, 0, remainingKeys, "All keys should be deleted")
	assert.Less(t, duration.Seconds(), 5.0, 
		"Large dataset invalidation should complete in reasonable time")

	t.Log("✅ Large dataset invalidation preserved")
}

// TestBug12_Preservation_PartialInvalidation tests that SCAN handles partial results correctly
//
// **Validates: Requirements 3.6, 3.7**
//
// This test verifies that the SCAN-based invalidation correctly handles
// cases where results span multiple iterations
func TestBug12_Preservation_PartialInvalidation(t *testing.T) {
	ctx := context.Background()
	client, mr := setupBug12PreservationTestRedis(t)
	defer mr.Close()

	pc := NewPricingCache(client)

	// Clean up
	pattern := fmt.Sprintf("%s:cache:pricing:*", AppNamespace)
	cleanupKeysPreservation(t, ctx, client, pattern)

	// Create exactly 2500 entries (more than one SCAN batch of 1000)
	t.Log("Creating 2500 cache entries to test multi-batch SCAN...")
	for i := 0; i < 2500; i++ {
		encoding := "GSM7"
		length := i + 1 // Unique length for each entry
		
		info := &PricingInfo{
			CharCount:  length,
			PartsCount: 1,
			Encoding:   encoding,
			Cost:       0.05,
		}
		
		err := pc.Set(ctx, encoding, length, info)
		require.NoError(t, err)
	}

	// Verify entries
	iter := client.Client.Scan(ctx, 0, pattern, 1000).Iterator()
	keyCount := 0
	for iter.Next(ctx) {
		keyCount++
	}
	require.NoError(t, iter.Err())
	t.Logf("Created %d cache keys", keyCount)

	// Test invalidation (should handle multiple SCAN iterations)
	err := pc.InvalidateAll(ctx)
	require.NoError(t, err, "Multi-batch invalidation should succeed")

	// Verify complete deletion
	iter = client.Client.Scan(ctx, 0, pattern, 1000).Iterator()
	remainingKeys := 0
	for iter.Next(ctx) {
		remainingKeys++
	}
	require.NoError(t, iter.Err())
	
	assert.Equal(t, 0, remainingKeys, 
		"All keys should be deleted even across multiple SCAN iterations")

	t.Log("✅ Multi-batch SCAN invalidation preserved")
}

// Helper function to clean up keys for preservation tests
func cleanupKeysPreservation(t *testing.T, ctx context.Context, client *Client, pattern string) {
	iter := client.Client.Scan(ctx, 0, pattern, 1000).Iterator()
	keys := []string{}
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	require.NoError(t, iter.Err())
	
	if len(keys) > 0 {
		pipe := client.Client.Pipeline()
		for _, key := range keys {
			pipe.Del(ctx, key)
		}
		_, err := pipe.Exec(ctx)
		require.NoError(t, err)
	}
}

// Helper function to setup test Redis client for preservation tests
func setupBug12PreservationTestRedis(t *testing.T) (*Client, *miniredis.Miniredis) {
	mr := miniredis.RunT(t)
	
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return &Client{
		Client: client,
	}, mr
}
