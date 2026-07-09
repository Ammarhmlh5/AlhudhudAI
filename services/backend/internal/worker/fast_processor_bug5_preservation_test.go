package worker

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"pgregory.net/rapid"

	"github.com/al-hudhud/backend/internal/redis"
)

// Bug 5 Preservation Tests: Successful Operations Continue
// **Validates: Requirements 3.1, 3.2, 3.3**
//
// **Property 2: Preservation** - Successful Operations Continue
//
// These tests verify that SUCCESSFUL operations continue to work correctly on UNFIXED code.
// They test the EXISTING functionality that must continue to work after the fix.
//
// Expected outcome on UNFIXED code: Tests PASS (existing behavior works)
// Expected outcome on FIXED code: Tests PASS (behavior preserved)
//
// IMPORTANT: These tests focus on the HAPPY PATH - when operations succeed,
// they should continue to work as expected. Bug 5 is about error handling,
// not about breaking successful operations.

// TestBug5_Preservation_MessageEnqueueFIFO verifies that successful message
// enqueue operations maintain FIFO order within each priority level
// **Validates: Requirement 3.1**
func TestBug5_Preservation_MessageEnqueueFIFO(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Bug 5 preservation test in short mode")
	}

	// Setup Redis client with miniredis
	mr := miniredis.RunT(t)
	defer mr.Close()

	client := goredis.NewClient(&goredis.Options{
		Addr: mr.Addr(),
	})

	logger := zap.NewNop()
	redisClient := &redis.Client{
		Client: client,
		Logger: logger,
	}

	ctx := context.Background()

	// Property: Successful enqueue operations maintain FIFO order
	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		priority := rapid.IntRange(1, 3).Draw(t, "priority")
		messageCount := rapid.IntRange(2, 10).Draw(t, "messageCount")

		// Create message queue
		messageQueue := redis.NewMessageQueue(redisClient)

		// Clear any existing messages
		err := messageQueue.Clear(ctx, tenantID)
		require.NoError(t, err)

		// Enqueue messages in sequence
		enqueuedMessages := make([]*redis.MessageQueueItem, messageCount)
		for i := 0; i < messageCount; i++ {
			msg := &redis.MessageQueueItem{
				MessageID:   uuid.New(),
				TenantID:    tenantID,
				PhoneNumber: rapid.StringMatching(`\+966[0-9]{9}`).Draw(t, "phone"),
				Content:     rapid.StringN(1, 160, -1).Draw(t, "content"),
				PartsCount:  1,
				Cost:        0.05,
				Priority:    priority,
				EnqueuedAt:  time.Now().Add(time.Duration(i) * time.Millisecond),
			}
			enqueuedMessages[i] = msg

			// ACT: Enqueue message (successful operation)
			err := messageQueue.Enqueue(ctx, msg)
			require.NoError(t, err, "Preservation: Successful enqueue should work")
		}

		// Dequeue all messages
		dequeuedMessages := make([]*redis.MessageQueueItem, 0, messageCount)
		for i := 0; i < messageCount; i++ {
			msg, err := messageQueue.Dequeue(ctx, tenantID)
			if err != nil {
				break
			}
			dequeuedMessages = append(dequeuedMessages, msg)
		}

		// ASSERT: FIFO order preserved - first enqueued is first dequeued
		require.Len(t, dequeuedMessages, messageCount,
			"Preservation: All successfully enqueued messages should be dequeued")

		// Verify FIFO order by checking message IDs match in order
		for i := 0; i < messageCount; i++ {
			assert.Equal(t, enqueuedMessages[i].MessageID, dequeuedMessages[i].MessageID,
				"Preservation: Message at position %d should maintain FIFO order. "+
					"Expected MessageID %s, Got %s",
				i, enqueuedMessages[i].MessageID, dequeuedMessages[i].MessageID)
		}

		// Cleanup
		messageQueue.Clear(ctx, tenantID)
	})
}

// TestBug5_Preservation_BatchEnqueueSuccess verifies that successful batch
// enqueue operations work correctly
// **Validates: Requirement 3.1**
func TestBug5_Preservation_BatchEnqueueSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Bug 5 preservation test in short mode")
	}

	mr := miniredis.RunT(t)
	defer mr.Close()

	client := goredis.NewClient(&goredis.Options{
		Addr: mr.Addr(),
	})

	logger := zap.NewNop()
	redisClient := &redis.Client{
		Client: client,
		Logger: logger,
	}

	ctx := context.Background()

	// Property: Successful batch enqueue operations work correctly
	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		priority := rapid.IntRange(1, 3).Draw(t, "priority")
		batchSize := rapid.IntRange(5, 20).Draw(t, "batchSize")

		messageQueue := redis.NewMessageQueue(redisClient)

		// Clear queue
		err := messageQueue.Clear(ctx, tenantID)
		require.NoError(t, err)

		// Create batch of messages
		messages := make([]*redis.MessageQueueItem, batchSize)
		for i := 0; i < batchSize; i++ {
			messages[i] = &redis.MessageQueueItem{
				MessageID:   uuid.New(),
				TenantID:    tenantID,
				PhoneNumber: rapid.StringMatching(`\+966[0-9]{9}`).Draw(t, "phone"),
				Content:     rapid.StringN(1, 160, -1).Draw(t, "content"),
				PartsCount:  1,
				Cost:        0.05,
				Priority:    priority,
				EnqueuedAt:  time.Now().Add(time.Duration(i) * time.Millisecond),
			}
		}

		// ACT: Enqueue batch (successful operation)
		err = messageQueue.EnqueueBatch(ctx, messages)
		require.NoError(t, err, "Preservation: Successful batch enqueue should work")

		// Verify queue size
		queueSize, err := messageQueue.GetQueueSize(ctx, tenantID)
		require.NoError(t, err)
		assert.Equal(t, int64(batchSize), queueSize,
			"Preservation: Queue size should match batch size after successful enqueue")

		// Verify tenant added to active set
		activeTenants, err := messageQueue.GetActiveTenants(ctx)
		require.NoError(t, err)
		assert.Contains(t, activeTenants, tenantID.String(),
			"Preservation: Tenant should be in active set after successful enqueue")

		// Cleanup
		messageQueue.Clear(ctx, tenantID)
	})
}

// TestBug5_Preservation_PricingCacheHit verifies that successful cache
// operations work correctly
// **Validates: Requirement 3.1 (cache operations support message processing)**
func TestBug5_Preservation_PricingCacheHit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Bug 5 preservation test in short mode")
	}

	mr := miniredis.RunT(t)
	defer mr.Close()

	client := goredis.NewClient(&goredis.Options{
		Addr: mr.Addr(),
	})

	logger := zap.NewNop()
	redisClient := &redis.Client{
		Client: client,
		Logger: logger,
	}

	ctx := context.Background()

	// Property: Successful cache operations work correctly
	rapid.Check(t, func(t *rapid.T) {
		encoding := rapid.SampledFrom([]string{"GSM7", "UNICODE"}).Draw(t, "encoding")
		charCount := rapid.IntRange(1, 500).Draw(t, "charCount")

		pricingCache := redis.NewPricingCache(redisClient)

		// Create pricing info
		info := &redis.PricingInfo{
			CharCount:  charCount,
			PartsCount: (charCount / 160) + 1,
			Encoding:   encoding,
			Cost:       0.05 * float64((charCount/160)+1),
		}

		// ACT: Set cache (successful operation)
		err := pricingCache.Set(ctx, encoding, charCount, info)
		require.NoError(t, err, "Preservation: Successful cache set should work")

		// ACT: Get from cache (successful operation)
		cached, err := pricingCache.Get(ctx, encoding, charCount)
		require.NoError(t, err, "Preservation: Successful cache get should work")
		require.NotNil(t, cached, "Preservation: Cached data should not be nil")

		// ASSERT: Cached data matches original
		assert.Equal(t, info.CharCount, cached.CharCount,
			"Preservation: Cached char count should match")
		assert.Equal(t, info.PartsCount, cached.PartsCount,
			"Preservation: Cached parts count should match")
		assert.Equal(t, info.Encoding, cached.Encoding,
			"Preservation: Cached encoding should match")
		assert.Equal(t, info.Cost, cached.Cost,
			"Preservation: Cached cost should match")
	})
}

// TestBug5_Preservation_GetOrCalculateSuccess verifies that successful
// GetOrCalculate operations work correctly
// **Validates: Requirement 3.1**
func TestBug5_Preservation_GetOrCalculateSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Bug 5 preservation test in short mode")
	}

	mr := miniredis.RunT(t)
	defer mr.Close()

	client := goredis.NewClient(&goredis.Options{
		Addr: mr.Addr(),
	})

	logger := zap.NewNop()
	redisClient := &redis.Client{
		Client: client,
		Logger: logger,
	}

	ctx := context.Background()

	// Property: Successful GetOrCalculate operations work correctly
	rapid.Check(t, func(t *rapid.T) {
		content := rapid.StringN(1, 300, -1).Draw(t, "content")
		costPerPart := 0.05

		pricingCache := redis.NewPricingCache(redisClient)

		// ACT: GetOrCalculate (successful operation)
		result, err := pricingCache.GetOrCalculate(ctx, content, costPerPart)
		require.NoError(t, err, "Preservation: Successful GetOrCalculate should work")
		require.NotNil(t, result, "Preservation: Result should not be nil")

		// ASSERT: Result has valid data
		assert.Greater(t, result.CharCount, 0,
			"Preservation: Char count should be positive")
		assert.Greater(t, result.PartsCount, 0,
			"Preservation: Parts count should be positive")
		assert.NotEmpty(t, result.Encoding,
			"Preservation: Encoding should not be empty")
		assert.Greater(t, result.Cost, 0.0,
			"Preservation: Cost should be positive")

		// Call again - should hit cache
		result2, err := pricingCache.GetOrCalculate(ctx, content, costPerPart)
		require.NoError(t, err, "Preservation: Cache hit should work")
		assert.Equal(t, result.CharCount, result2.CharCount,
			"Preservation: Cache hit should return same data")
	})
}

// TestBug5_Preservation_ActiveTenantsTracking verifies that successful
// active tenant tracking works correctly
// **Validates: Requirement 3.1**
func TestBug5_Preservation_ActiveTenantsTracking(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Bug 5 preservation test in short mode")
	}

	mr := miniredis.RunT(t)
	defer mr.Close()

	client := goredis.NewClient(&goredis.Options{
		Addr: mr.Addr(),
	})

	logger := zap.NewNop()
	redisClient := &redis.Client{
		Client: client,
		Logger: logger,
	}

	ctx := context.Background()

	// Property: Successful active tenant tracking works correctly
	rapid.Check(t, func(t *rapid.T) {
		tenantCount := rapid.IntRange(1, 10).Draw(t, "tenantCount")

		messageQueue := redis.NewMessageQueue(redisClient)

		// Clear active set
		activeKey := redis.AppNamespace + ":active:tenants"
		_ = client.Del(ctx, activeKey).Err()

		// Create messages for multiple tenants
		tenantIDs := make([]uuid.UUID, tenantCount)
		for i := 0; i < tenantCount; i++ {
			tenantID := uuid.New()
			tenantIDs[i] = tenantID

			msg := &redis.MessageQueueItem{
				MessageID:   uuid.New(),
				TenantID:    tenantID,
				PhoneNumber: "+966500000001",
				Content:     "Test message",
				PartsCount:  1,
				Cost:        0.05,
				Priority:    2,
				EnqueuedAt:  time.Now(),
			}

			// ACT: Enqueue message (successful operation)
			err := messageQueue.Enqueue(ctx, msg)
			require.NoError(t, err, "Preservation: Successful enqueue should work")
		}

		// ACT: Get active tenants (successful operation)
		activeTenants, err := messageQueue.GetActiveTenants(ctx)
		require.NoError(t, err, "Preservation: GetActiveTenants should work")

		// ASSERT: All tenants are in active set
		assert.GreaterOrEqual(t, len(activeTenants), tenantCount,
			"Preservation: All tenants should be in active set after successful enqueue")

		for _, tenantID := range tenantIDs {
			assert.Contains(t, activeTenants, tenantID.String(),
				"Preservation: Tenant %s should be in active set", tenantID)
		}

		// Cleanup
		for _, tenantID := range tenantIDs {
			messageQueue.Clear(ctx, tenantID)
		}
	})
}

// Simple unit test: Successful message enqueue
// **Validates: Requirement 3.1**
func TestBug5_Preservation_Simple_MessageEnqueue(t *testing.T) {
	mr := miniredis.RunT(t)
	defer mr.Close()

	client := goredis.NewClient(&goredis.Options{
		Addr: mr.Addr(),
	})

	logger := zap.NewNop()
	redisClient := &redis.Client{
		Client: client,
		Logger: logger,
	}

	ctx := context.Background()

	tenantID := uuid.New()
	messageQueue := redis.NewMessageQueue(redisClient)

	// Clear queue
	err := messageQueue.Clear(ctx, tenantID)
	require.NoError(t, err)

	// Create and enqueue a message
	msg := &redis.MessageQueueItem{
		MessageID:   uuid.New(),
		TenantID:    tenantID,
		PhoneNumber: "+966500000001",
		Content:     "Test message",
		PartsCount:  1,
		Cost:        0.05,
		Priority:    2,
		EnqueuedAt:  time.Now(),
	}

	// ACT: Enqueue message (successful operation)
	err = messageQueue.Enqueue(ctx, msg)
	require.NoError(t, err, "Preservation: Successful enqueue should work")

	// Verify queue size
	queueSize, err := messageQueue.GetQueueSize(ctx, tenantID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), queueSize,
		"Preservation: Queue size should be 1 after successful enqueue")

	// Dequeue the message
	dequeuedMsg, err := messageQueue.Dequeue(ctx, tenantID)
	require.NoError(t, err, "Preservation: Successful dequeue should work")
	require.NotNil(t, dequeuedMsg, "Preservation: Dequeued message should not be nil")

	// Verify message content
	assert.Equal(t, msg.MessageID, dequeuedMsg.MessageID,
		"Preservation: Dequeued message should match enqueued message")

	t.Logf("✅ Successful message enqueue/dequeue works correctly")
}

// Simple unit test: Successful cache operations
// **Validates: Requirement 3.1**
func TestBug5_Preservation_Simple_CacheOperations(t *testing.T) {
	mr := miniredis.RunT(t)
	defer mr.Close()

	client := goredis.NewClient(&goredis.Options{
		Addr: mr.Addr(),
	})

	logger := zap.NewNop()
	redisClient := &redis.Client{
		Client: client,
		Logger: logger,
	}

	ctx := context.Background()

	pricingCache := redis.NewPricingCache(redisClient)

	// Create pricing info
	info := &redis.PricingInfo{
		CharCount:  100,
		PartsCount: 1,
		Encoding:   "GSM7",
		Cost:       0.05,
	}

	// ACT: Set cache (successful operation)
	err := pricingCache.Set(ctx, "GSM7", 100, info)
	require.NoError(t, err, "Preservation: Successful cache set should work")

	// ACT: Get from cache (successful operation)
	cached, err := pricingCache.Get(ctx, "GSM7", 100)
	require.NoError(t, err, "Preservation: Successful cache get should work")
	require.NotNil(t, cached, "Preservation: Cached data should not be nil")

	// Verify cached data
	assert.Equal(t, info.CharCount, cached.CharCount,
		"Preservation: Cached data should match original")
	assert.Equal(t, info.PartsCount, cached.PartsCount,
		"Preservation: Cached data should match original")
	assert.Equal(t, info.Encoding, cached.Encoding,
		"Preservation: Cached data should match original")
	assert.Equal(t, info.Cost, cached.Cost,
		"Preservation: Cached data should match original")

	t.Logf("✅ Successful cache operations work correctly")
}

// Simple unit test: Successful GetOrCalculate
// **Validates: Requirement 3.1**
func TestBug5_Preservation_Simple_GetOrCalculate(t *testing.T) {
	mr := miniredis.RunT(t)
	defer mr.Close()

	client := goredis.NewClient(&goredis.Options{
		Addr: mr.Addr(),
	})

	logger := zap.NewNop()
	redisClient := &redis.Client{
		Client: client,
		Logger: logger,
	}

	ctx := context.Background()

	pricingCache := redis.NewPricingCache(redisClient)

	content := "Hello, this is a test message"
	costPerPart := 0.05

	// ACT: GetOrCalculate (successful operation - cache miss)
	result1, err := pricingCache.GetOrCalculate(ctx, content, costPerPart)
	require.NoError(t, err, "Preservation: Successful GetOrCalculate should work")
	require.NotNil(t, result1, "Preservation: Result should not be nil")

	// Verify result
	assert.Greater(t, result1.CharCount, 0,
		"Preservation: Char count should be positive")
	assert.Greater(t, result1.PartsCount, 0,
		"Preservation: Parts count should be positive")
	assert.NotEmpty(t, result1.Encoding,
		"Preservation: Encoding should not be empty")

	// ACT: GetOrCalculate again (successful operation - cache hit)
	result2, err := pricingCache.GetOrCalculate(ctx, content, costPerPart)
	require.NoError(t, err, "Preservation: Cache hit should work")
	require.NotNil(t, result2, "Preservation: Result should not be nil")

	// Verify cache hit returns same data
	assert.Equal(t, result1.CharCount, result2.CharCount,
		"Preservation: Cache hit should return same data")
	assert.Equal(t, result1.PartsCount, result2.PartsCount,
		"Preservation: Cache hit should return same data")
	assert.Equal(t, result1.Encoding, result2.Encoding,
		"Preservation: Cache hit should return same data")
	assert.Equal(t, result1.Cost, result2.Cost,
		"Preservation: Cache hit should return same data")

	t.Logf("✅ Successful GetOrCalculate operations work correctly (cache miss and hit)")
}

// Simple unit test: Successful batch operations
// **Validates: Requirement 3.2**
func TestBug5_Preservation_Simple_BatchOperations(t *testing.T) {
	mr := miniredis.RunT(t)
	defer mr.Close()

	client := goredis.NewClient(&goredis.Options{
		Addr: mr.Addr(),
	})

	logger := zap.NewNop()
	redisClient := &redis.Client{
		Client: client,
		Logger: logger,
	}

	ctx := context.Background()

	tenantID := uuid.New()
	messageQueue := redis.NewMessageQueue(redisClient)

	// Clear queue
	err := messageQueue.Clear(ctx, tenantID)
	require.NoError(t, err)

	// Create batch of messages
	batchSize := 10
	messages := make([]*redis.MessageQueueItem, batchSize)
	for i := 0; i < batchSize; i++ {
		messages[i] = &redis.MessageQueueItem{
			MessageID:   uuid.New(),
			TenantID:    tenantID,
			PhoneNumber: "+966500000001",
			Content:     "Test message",
			PartsCount:  1,
			Cost:        0.05,
			Priority:    2,
			EnqueuedAt:  time.Now(),
		}
	}

	// ACT: Enqueue batch (successful operation)
	err = messageQueue.EnqueueBatch(ctx, messages)
	require.NoError(t, err, "Preservation: Successful batch enqueue should work")

	// Verify queue size
	queueSize, err := messageQueue.GetQueueSize(ctx, tenantID)
	require.NoError(t, err)
	assert.Equal(t, int64(batchSize), queueSize,
		"Preservation: Queue size should match batch size")

	// ACT: Dequeue batch (successful operation)
	dequeuedMessages, err := messageQueue.DequeueBatch(ctx, tenantID, batchSize)
	require.NoError(t, err, "Preservation: Successful batch dequeue should work")
	assert.Len(t, dequeuedMessages, batchSize,
		"Preservation: Should dequeue all messages")

	t.Logf("✅ Successful batch operations work correctly: enqueued %d, dequeued %d",
		batchSize, len(dequeuedMessages))
}
