package redis

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
)

// Bug 9 Preservation Tests: Successful Redis Operations
// **Validates: Requirement 3.8**
//
// **Property 2: Preservation** - Successful Redis Operations
//
// These tests verify that when Redis is available and operations succeed,
// the system continues to provide fast queue operations and caching.
//
// CRITICAL: These tests MUST PASS on UNFIXED code - they test existing working functionality.
// They ensure that adding retry logic does not break or slow down successful operations.

// TestPreservation_Bug9_FastEnqueueOperations verifies that successful enqueue operations
// remain fast (< 10ms) when Redis is available.
// **Validates: Requirement 3.8**
func TestPreservation_Bug9_FastEnqueueOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping preservation test in short mode")
	}

	rapid.Check(t, func(t *rapid.T) {
		// Setup miniredis
		mr := miniredis.RunT(t)
		defer mr.Close()

		client := goredis.NewClient(&goredis.Options{
			Addr:       mr.Addr(),
			MaxRetries: 0,
		})
		defer client.Close()

		logger := zap.NewNop()
		redisClient := &Client{
			Client: client,
			Logger: logger,
		}

		messageQueue := NewMessageQueue(redisClient)
		ctx := context.Background()

		// Generate random test data
		tenantID := uuid.New()
		priority := rapid.IntRange(1, 3).Draw(t, "priority")
		messageCount := rapid.IntRange(1, 100).Draw(t, "messageCount")

		// ACT: Enqueue messages and measure performance
		startTime := time.Now()
		
		for i := 0; i < messageCount; i++ {
			item := &MessageQueueItem{
				MessageID:   uuid.New(),
				TenantID:    tenantID,
				PhoneNumber: "+966500000000",
				Content:     rapid.StringN(1, 160, 160).Draw(t, "content"),
				PartsCount:  1,
				Cost:        0.05,
				Priority:    priority,
				EnqueuedAt:  time.Now(),
			}
			
			err := messageQueue.Enqueue(ctx, item)
			require.NoError(t, err, "Enqueue should succeed when Redis is available")
		}
		
		duration := time.Since(startTime)
		avgDuration := duration / time.Duration(messageCount)

		// ASSERT: Operations should be fast
		assert.Less(t, avgDuration, 10*time.Millisecond,
			"Average enqueue time should be < 10ms (actual: %v for %d messages)",
			avgDuration, messageCount)

		// Verify messages were actually enqueued
		queueSize, err := messageQueue.GetQueueSize(ctx, tenantID)
		require.NoError(t, err)
		assert.Equal(t, int64(messageCount), queueSize,
			"All messages should be enqueued")

		t.Logf("✓ Preservation verified: Enqueued %d messages in %v (avg: %v per message)",
			messageCount, duration, avgDuration)
	})
}

// TestPreservation_Bug9_FastDequeueOperations verifies that successful dequeue operations
// remain fast when Redis is available.
// **Validates: Requirement 3.8**
func TestPreservation_Bug9_FastDequeueOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping preservation test in short mode")
	}

	rapid.Check(t, func(t *rapid.T) {
		// Setup miniredis
		mr := miniredis.RunT(t)
		defer mr.Close()

		client := goredis.NewClient(&goredis.Options{
			Addr:       mr.Addr(),
			MaxRetries: 0,
		})
		defer client.Close()

		logger := zap.NewNop()
		redisClient := &Client{
			Client: client,
			Logger: logger,
		}

		messageQueue := NewMessageQueue(redisClient)
		ctx := context.Background()

		// Generate random test data
		tenantID := uuid.New()
		priority := rapid.IntRange(1, 3).Draw(t, "priority")
		messageCount := rapid.IntRange(10, 100).Draw(t, "messageCount")

		// Enqueue messages first
		for i := 0; i < messageCount; i++ {
			item := &MessageQueueItem{
				MessageID:   uuid.New(),
				TenantID:    tenantID,
				PhoneNumber: "+966500000000",
				Content:     "Test message",
				PartsCount:  1,
				Cost:        0.05,
				Priority:    priority,
				EnqueuedAt:  time.Now(),
			}
			
			err := messageQueue.Enqueue(ctx, item)
			require.NoError(t, err)
		}

		// ACT: Dequeue messages and measure performance
		startTime := time.Now()
		dequeuedCount := 0
		
		for i := 0; i < messageCount; i++ {
			item, err := messageQueue.Dequeue(ctx, tenantID)
			if err == nil && item != nil {
				dequeuedCount++
			}
		}
		
		duration := time.Since(startTime)
		avgDuration := duration / time.Duration(dequeuedCount)

		// ASSERT: Operations should be fast
		assert.Less(t, avgDuration, 10*time.Millisecond,
			"Average dequeue time should be < 10ms (actual: %v for %d messages)",
			avgDuration, dequeuedCount)

		assert.Equal(t, messageCount, dequeuedCount,
			"All messages should be dequeued")

		t.Logf("✓ Preservation verified: Dequeued %d messages in %v (avg: %v per message)",
			dequeuedCount, duration, avgDuration)
	})
}

// TestPreservation_Bug9_BatchOperationsPerformance verifies that batch operations
// remain efficient when Redis is available.
// **Validates: Requirement 3.8**
func TestPreservation_Bug9_BatchOperationsPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping preservation test in short mode")
	}

	rapid.Check(t, func(t *rapid.T) {
		// Setup miniredis
		mr := miniredis.RunT(t)
		defer mr.Close()

		client := goredis.NewClient(&goredis.Options{
			Addr:       mr.Addr(),
			MaxRetries: 0,
		})
		defer client.Close()

		logger := zap.NewNop()
		redisClient := &Client{
			Client: client,
			Logger: logger,
		}

		messageQueue := NewMessageQueue(redisClient)
		ctx := context.Background()

		// Generate random test data
		tenantID := uuid.New()
		batchSize := rapid.IntRange(10, 100).Draw(t, "batchSize")

		// Create batch of messages
		items := make([]*MessageQueueItem, batchSize)
		for i := 0; i < batchSize; i++ {
			items[i] = &MessageQueueItem{
				MessageID:   uuid.New(),
				TenantID:    tenantID,
				PhoneNumber: "+966500000000",
				Content:     "Test message",
				PartsCount:  1,
				Cost:        0.05,
				Priority:    2,
				EnqueuedAt:  time.Now(),
			}
		}

		// ACT: Batch enqueue and measure performance
		startTime := time.Now()
		err := messageQueue.EnqueueBatch(ctx, items)
		enqueueDuration := time.Since(startTime)

		require.NoError(t, err, "Batch enqueue should succeed")

		// Batch operations should be significantly faster than individual operations
		// Expected: < 1ms per message for batch operations
		avgEnqueueTime := enqueueDuration / time.Duration(batchSize)
		assert.Less(t, avgEnqueueTime, 1*time.Millisecond,
			"Average batch enqueue time should be < 1ms per message (actual: %v)",
			avgEnqueueTime)

		// ACT: Batch dequeue and measure performance
		startTime = time.Now()
		dequeuedItems, err := messageQueue.DequeueBatch(ctx, tenantID, batchSize)
		dequeueDuration := time.Since(startTime)

		require.NoError(t, err, "Batch dequeue should succeed")
		assert.Equal(t, batchSize, len(dequeuedItems),
			"All messages should be dequeued")

		avgDequeueTime := dequeueDuration / time.Duration(len(dequeuedItems))
		assert.Less(t, avgDequeueTime, 1*time.Millisecond,
			"Average batch dequeue time should be < 1ms per message (actual: %v)",
			avgDequeueTime)

		t.Logf("✓ Preservation verified: Batch operations for %d messages", batchSize)
		t.Logf("  Enqueue: %v total, %v per message", enqueueDuration, avgEnqueueTime)
		t.Logf("  Dequeue: %v total, %v per message", dequeueDuration, avgDequeueTime)
	})
}

// TestPreservation_Bug9_DataIntegrity verifies that message data integrity
// is maintained during successful operations.
// **Validates: Requirement 3.8**
func TestPreservation_Bug9_DataIntegrity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping preservation test in short mode")
	}

	rapid.Check(t, func(t *rapid.T) {
		// Setup miniredis
		mr := miniredis.RunT(t)
		defer mr.Close()

		client := goredis.NewClient(&goredis.Options{
			Addr:       mr.Addr(),
			MaxRetries: 0,
		})
		defer client.Close()

		logger := zap.NewNop()
		redisClient := &Client{
			Client: client,
			Logger: logger,
		}

		messageQueue := NewMessageQueue(redisClient)
		ctx := context.Background()

		// Generate random test data
		tenantID := uuid.New()
		messageID := uuid.New()
		phoneNumber := rapid.StringMatching(`\+966[0-9]{9}`).Draw(t, "phoneNumber")
		content := rapid.StringN(1, 160, 160).Draw(t, "content")
		priority := rapid.IntRange(1, 3).Draw(t, "priority")
		cost := rapid.Float64Range(0.01, 1.0).Draw(t, "cost")

		originalItem := &MessageQueueItem{
			MessageID:   messageID,
			TenantID:    tenantID,
			PhoneNumber: phoneNumber,
			Content:     content,
			PartsCount:  1,
			Cost:        cost,
			Priority:    priority,
			EnqueuedAt:  time.Now(),
		}

		// ACT: Enqueue and dequeue
		err := messageQueue.Enqueue(ctx, originalItem)
		require.NoError(t, err, "Enqueue should succeed")

		dequeuedItem, err := messageQueue.Dequeue(ctx, tenantID)
		require.NoError(t, err, "Dequeue should succeed")
		require.NotNil(t, dequeuedItem, "Dequeued item should not be nil")

		// ASSERT: Data integrity preserved
		assert.Equal(t, originalItem.MessageID, dequeuedItem.MessageID,
			"Message ID should be preserved")
		assert.Equal(t, originalItem.TenantID, dequeuedItem.TenantID,
			"Tenant ID should be preserved")
		assert.Equal(t, originalItem.PhoneNumber, dequeuedItem.PhoneNumber,
			"Phone number should be preserved")
		assert.Equal(t, originalItem.Content, dequeuedItem.Content,
			"Content should be preserved")
		assert.Equal(t, originalItem.Priority, dequeuedItem.Priority,
			"Priority should be preserved")
		assert.Equal(t, originalItem.Cost, dequeuedItem.Cost,
			"Cost should be preserved")

		t.Logf("✓ Preservation verified: Data integrity maintained for message %s",
			messageID.String())
	})
}

// TestPreservation_Bug9_ConcurrentOperations verifies that concurrent operations
// work correctly when Redis is available.
// **Validates: Requirement 3.8**
func TestPreservation_Bug9_ConcurrentOperations(t *testing.T) {
	// Setup miniredis
	mr := miniredis.RunT(t)
	defer mr.Close()

	client := goredis.NewClient(&goredis.Options{
		Addr:       mr.Addr(),
		MaxRetries: 0,
	})
	defer client.Close()

	logger := zap.NewNop()
	redisClient := &Client{
		Client: client,
		Logger: logger,
	}

	messageQueue := NewMessageQueue(redisClient)
	ctx := context.Background()

	tenantID := uuid.New()
	goroutineCount := 10
	messagesPerGoroutine := 100
	totalMessages := goroutineCount * messagesPerGoroutine

	// ACT: Concurrent enqueue operations
	startTime := time.Now()
	done := make(chan bool, goroutineCount)

	for g := 0; g < goroutineCount; g++ {
		go func(goroutineID int) {
			for i := 0; i < messagesPerGoroutine; i++ {
				item := &MessageQueueItem{
					MessageID:   uuid.New(),
					TenantID:    tenantID,
					PhoneNumber: "+966500000000",
					Content:     "Test message",
					PartsCount:  1,
					Cost:        0.05,
					Priority:    2,
					EnqueuedAt:  time.Now(),
				}
				
				err := messageQueue.Enqueue(ctx, item)
				if err != nil {
					t.Errorf("Concurrent enqueue failed: %v", err)
				}
			}
			done <- true
		}(g)
	}

	// Wait for all goroutines
	for g := 0; g < goroutineCount; g++ {
		<-done
	}
	duration := time.Since(startTime)

	// ASSERT: All messages enqueued successfully
	queueSize, err := messageQueue.GetQueueSize(ctx, tenantID)
	require.NoError(t, err)
	assert.Equal(t, int64(totalMessages), queueSize,
		"All concurrent messages should be enqueued")

	// Performance should still be reasonable with concurrency
	avgTime := duration / time.Duration(totalMessages)
	assert.Less(t, avgTime, 10*time.Millisecond,
		"Average time per message should be < 10ms even with concurrency (actual: %v)",
		avgTime)

	t.Logf("✓ Preservation verified: %d concurrent operations completed in %v",
		totalMessages, duration)
	t.Logf("  Goroutines: %d, Messages per goroutine: %d",
		goroutineCount, messagesPerGoroutine)
	t.Logf("  Average time per message: %v", avgTime)
}

// TestPreservation_Bug9_ActiveTenantsTracking verifies that active tenants
// tracking continues to work correctly.
// **Validates: Requirement 3.8**
func TestPreservation_Bug9_ActiveTenantsTracking(t *testing.T) {
	// Setup miniredis
	mr := miniredis.RunT(t)
	defer mr.Close()

	client := goredis.NewClient(&goredis.Options{
		Addr:       mr.Addr(),
		MaxRetries: 0,
	})
	defer client.Close()

	logger := zap.NewNop()
	redisClient := &Client{
		Client: client,
		Logger: logger,
	}

	messageQueue := NewMessageQueue(redisClient)
	ctx := context.Background()

	// Create multiple tenants
	tenantCount := 5
	tenants := make([]uuid.UUID, tenantCount)
	for i := 0; i < tenantCount; i++ {
		tenants[i] = uuid.New()
	}

	// ACT: Enqueue messages for each tenant
	for _, tenantID := range tenants {
		item := &MessageQueueItem{
			MessageID:   uuid.New(),
			TenantID:    tenantID,
			PhoneNumber: "+966500000000",
			Content:     "Test message",
			PartsCount:  1,
			Cost:        0.05,
			Priority:    2,
			EnqueuedAt:  time.Now(),
		}
		
		err := messageQueue.Enqueue(ctx, item)
		require.NoError(t, err, "Enqueue should succeed")
	}

	// ASSERT: All tenants should be in active set
	activeTenants, err := messageQueue.GetActiveTenants(ctx)
	require.NoError(t, err, "GetActiveTenants should succeed")
	assert.Equal(t, tenantCount, len(activeTenants),
		"All tenants should be in active set")

	// Verify each tenant is present
	activeTenantMap := make(map[string]bool)
	for _, tenantStr := range activeTenants {
		activeTenantMap[tenantStr] = true
	}

	for _, tenantID := range tenants {
		assert.True(t, activeTenantMap[tenantID.String()],
			"Tenant %s should be in active set", tenantID.String())
	}

	t.Logf("✓ Preservation verified: Active tenants tracking works correctly")
	t.Logf("  Tracked %d active tenants", len(activeTenants))
}

// TestPreservation_Bug9_QueueStatsAccuracy verifies that queue statistics
// remain accurate during successful operations.
// **Validates: Requirement 3.8**
func TestPreservation_Bug9_QueueStatsAccuracy(t *testing.T) {
	// Setup miniredis
	mr := miniredis.RunT(t)
	defer mr.Close()

	client := goredis.NewClient(&goredis.Options{
		Addr:       mr.Addr(),
		MaxRetries: 0,
	})
	defer client.Close()

	logger := zap.NewNop()
	redisClient := &Client{
		Client: client,
		Logger: logger,
	}

	messageQueue := NewMessageQueue(redisClient)
	ctx := context.Background()

	tenantID := uuid.New()

	// Enqueue messages with different priorities
	highPriorityCount := 10
	normalPriorityCount := 20
	lowPriorityCount := 5

	// High priority
	for i := 0; i < highPriorityCount; i++ {
		item := &MessageQueueItem{
			MessageID:   uuid.New(),
			TenantID:    tenantID,
			PhoneNumber: "+966500000000",
			Content:     "High priority",
			PartsCount:  1,
			Cost:        0.05,
			Priority:    1,
			EnqueuedAt:  time.Now(),
		}
		err := messageQueue.Enqueue(ctx, item)
		require.NoError(t, err)
	}

	// Normal priority
	for i := 0; i < normalPriorityCount; i++ {
		item := &MessageQueueItem{
			MessageID:   uuid.New(),
			TenantID:    tenantID,
			PhoneNumber: "+966500000000",
			Content:     "Normal priority",
			PartsCount:  1,
			Cost:        0.05,
			Priority:    2,
			EnqueuedAt:  time.Now(),
		}
		err := messageQueue.Enqueue(ctx, item)
		require.NoError(t, err)
	}

	// Low priority
	for i := 0; i < lowPriorityCount; i++ {
		item := &MessageQueueItem{
			MessageID:   uuid.New(),
			TenantID:    tenantID,
			PhoneNumber: "+966500000000",
			Content:     "Low priority",
			PartsCount:  1,
			Cost:        0.05,
			Priority:    3,
			EnqueuedAt:  time.Now(),
		}
		err := messageQueue.Enqueue(ctx, item)
		require.NoError(t, err)
	}

	// ACT: Get queue stats
	stats, err := messageQueue.GetQueueStats(ctx, tenantID)
	require.NoError(t, err, "GetQueueStats should succeed")

	// ASSERT: Stats should be accurate
	assert.Equal(t, int64(highPriorityCount), stats["high_priority"],
		"High priority count should be accurate")
	assert.Equal(t, int64(normalPriorityCount), stats["normal_priority"],
		"Normal priority count should be accurate")
	assert.Equal(t, int64(lowPriorityCount), stats["low_priority"],
		"Low priority count should be accurate")
	assert.Equal(t, int64(highPriorityCount+normalPriorityCount+lowPriorityCount), stats["total"],
		"Total count should be accurate")

	t.Logf("✓ Preservation verified: Queue stats are accurate")
	t.Logf("  High: %d, Normal: %d, Low: %d, Total: %d",
		stats["high_priority"], stats["normal_priority"],
		stats["low_priority"], stats["total"])
}
