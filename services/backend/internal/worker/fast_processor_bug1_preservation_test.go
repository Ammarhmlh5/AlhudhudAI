package worker

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"

	"github.com/al-hudhud/backend/internal/redis"
)

// Bug 1 Preservation Test: Message Enqueue FIFO Order
// **Validates: Requirements 3.1**
//
// **Property 2: Preservation** - Message Processing Continues
//
// This test MUST PASS on unfixed code - it verifies existing functionality.
// Tests that messages enqueued to Redis maintain FIFO order within each priority level.
//
// CRITICAL: This test verifies behavior that should NOT change after the fix.
// On UNFIXED code: Test PASSES (existing functionality works)
// On FIXED code: Test PASSES (functionality preserved)
func TestProperty_Bug1_Preservation_MessageEnqueueFIFO(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	// Setup Redis client with miniredis
	redisClient, mr := setupTestRedis(t)
	defer mr.Close()
	ctx := context.Background()

	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
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

			err := messageQueue.Enqueue(ctx, msg)
			require.NoError(t, err, "Failed to enqueue message %d", i)
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
			"Preservation: All enqueued messages should be dequeued")

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

// Bug 1 Preservation Test: Priority Queue Ordering
// **Validates: Requirements 3.1**
//
// **Property 2: Preservation** - Message Processing Continues
//
// This test verifies that high priority messages are dequeued before normal/low priority.
// MUST PASS on unfixed code.
func TestProperty_Bug1_Preservation_PriorityOrdering(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	redisClient, mr := setupTestRedis(t)
	defer mr.Close()
	ctx := context.Background()

	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		messageQueue := redis.NewMessageQueue(redisClient)

		// Clear queue
		err := messageQueue.Clear(ctx, tenantID)
		require.NoError(t, err)

		// Enqueue messages with different priorities
		highPriorityMsg := &redis.MessageQueueItem{
			MessageID:   uuid.New(),
			TenantID:    tenantID,
			PhoneNumber: "+966500000001",
			Content:     "High priority message",
			PartsCount:  1,
			Cost:        0.05,
			Priority:    1, // High
			EnqueuedAt:  time.Now(),
		}

		normalPriorityMsg := &redis.MessageQueueItem{
			MessageID:   uuid.New(),
			TenantID:    tenantID,
			PhoneNumber: "+966500000002",
			Content:     "Normal priority message",
			PartsCount:  1,
			Cost:        0.05,
			Priority:    2, // Normal
			EnqueuedAt:  time.Now(),
		}

		lowPriorityMsg := &redis.MessageQueueItem{
			MessageID:   uuid.New(),
			TenantID:    tenantID,
			PhoneNumber: "+966500000003",
			Content:     "Low priority message",
			PartsCount:  1,
			Cost:        0.05,
			Priority:    3, // Low
			EnqueuedAt:  time.Now(),
		}

		// Enqueue in reverse order (low, normal, high)
		err = messageQueue.Enqueue(ctx, lowPriorityMsg)
		require.NoError(t, err)
		err = messageQueue.Enqueue(ctx, normalPriorityMsg)
		require.NoError(t, err)
		err = messageQueue.Enqueue(ctx, highPriorityMsg)
		require.NoError(t, err)

		// Dequeue and verify priority order
		msg1, err := messageQueue.Dequeue(ctx, tenantID)
		require.NoError(t, err)
		assert.Equal(t, highPriorityMsg.MessageID, msg1.MessageID,
			"Preservation: First dequeued message should be high priority")

		msg2, err := messageQueue.Dequeue(ctx, tenantID)
		require.NoError(t, err)
		assert.Equal(t, normalPriorityMsg.MessageID, msg2.MessageID,
			"Preservation: Second dequeued message should be normal priority")

		msg3, err := messageQueue.Dequeue(ctx, tenantID)
		require.NoError(t, err)
		assert.Equal(t, lowPriorityMsg.MessageID, msg3.MessageID,
			"Preservation: Third dequeued message should be low priority")

		// Cleanup
		messageQueue.Clear(ctx, tenantID)
	})
}

// Bug 1 Preservation Test: Batch Enqueue/Dequeue
// **Validates: Requirements 3.1, 3.2**
//
// **Property 2: Preservation** - Message Processing Continues
//
// This test verifies that batch operations maintain FIFO order and complete successfully.
// MUST PASS on unfixed code.
func TestProperty_Bug1_Preservation_BatchOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	redisClient, mr := setupTestRedis(t)
	defer mr.Close()
	ctx := context.Background()

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

		// Enqueue batch
		err = messageQueue.EnqueueBatch(ctx, messages)
		require.NoError(t, err, "Preservation: Batch enqueue should succeed")

		// Verify queue size
		queueSize, err := messageQueue.GetQueueSize(ctx, tenantID)
		require.NoError(t, err)
		assert.Equal(t, int64(batchSize), queueSize,
			"Preservation: Queue size should match batch size")

		// Dequeue batch
		dequeuedMessages, err := messageQueue.DequeueBatch(ctx, tenantID, batchSize)
		require.NoError(t, err, "Preservation: Batch dequeue should succeed")
		require.Len(t, dequeuedMessages, batchSize,
			"Preservation: Should dequeue all messages")

		// Verify FIFO order in batch
		for i := 0; i < batchSize; i++ {
			assert.Equal(t, messages[i].MessageID, dequeuedMessages[i].MessageID,
				"Preservation: Batch dequeue should maintain FIFO order at position %d", i)
		}

		// Cleanup
		messageQueue.Clear(ctx, tenantID)
	})
}

// Bug 1 Preservation Test: Queue Statistics
// **Validates: Requirements 3.1**
//
// **Property 2: Preservation** - Message Processing Continues
//
// This test verifies that queue statistics are accurate.
// MUST PASS on unfixed code.
func TestProperty_Bug1_Preservation_QueueStatistics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	redisClient, mr := setupTestRedis(t)
	defer mr.Close()
	ctx := context.Background()

	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		messageQueue := redis.NewMessageQueue(redisClient)

		// Clear queue
		err := messageQueue.Clear(ctx, tenantID)
		require.NoError(t, err)

		// Generate random counts for each priority
		highCount := rapid.IntRange(0, 10).Draw(t, "highCount")
		normalCount := rapid.IntRange(0, 10).Draw(t, "normalCount")
		lowCount := rapid.IntRange(0, 10).Draw(t, "lowCount")

		// Enqueue messages with different priorities
		for i := 0; i < highCount; i++ {
			msg := &redis.MessageQueueItem{
				MessageID:   uuid.New(),
				TenantID:    tenantID,
				PhoneNumber: "+966500000001",
				Content:     "High priority",
				PartsCount:  1,
				Cost:        0.05,
				Priority:    1,
				EnqueuedAt:  time.Now(),
			}
			err := messageQueue.Enqueue(ctx, msg)
			require.NoError(t, err)
		}

		for i := 0; i < normalCount; i++ {
			msg := &redis.MessageQueueItem{
				MessageID:   uuid.New(),
				TenantID:    tenantID,
				PhoneNumber: "+966500000002",
				Content:     "Normal priority",
				PartsCount:  1,
				Cost:        0.05,
				Priority:    2,
				EnqueuedAt:  time.Now(),
			}
			err := messageQueue.Enqueue(ctx, msg)
			require.NoError(t, err)
		}

		for i := 0; i < lowCount; i++ {
			msg := &redis.MessageQueueItem{
				MessageID:   uuid.New(),
				TenantID:    tenantID,
				PhoneNumber: "+966500000003",
				Content:     "Low priority",
				PartsCount:  1,
				Cost:        0.05,
				Priority:    3,
				EnqueuedAt:  time.Now(),
			}
			err := messageQueue.Enqueue(ctx, msg)
			require.NoError(t, err)
		}

		// Get queue statistics
		stats, err := messageQueue.GetQueueStats(ctx, tenantID)
		require.NoError(t, err, "Preservation: GetQueueStats should succeed")

		// Verify statistics
		assert.Equal(t, int64(highCount), stats["high_priority"],
			"Preservation: High priority count should be accurate")
		assert.Equal(t, int64(normalCount), stats["normal_priority"],
			"Preservation: Normal priority count should be accurate")
		assert.Equal(t, int64(lowCount), stats["low_priority"],
			"Preservation: Low priority count should be accurate")
		assert.Equal(t, int64(highCount+normalCount+lowCount), stats["total"],
			"Preservation: Total count should be sum of all priorities")

		// Cleanup
		messageQueue.Clear(ctx, tenantID)
	})
}

// Example-based preservation test: Simple enqueue/dequeue
// **Validates: Requirements 3.1**
//
// This test provides a concrete example of preservation behavior.
// EXPECTED OUTCOME on UNFIXED code: PASSES (existing functionality works)
func TestBug1_Preservation_SimpleEnqueueDequeue(t *testing.T) {
	redisClient, mr := setupTestRedis(t)
	defer mr.Close()
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

	err = messageQueue.Enqueue(ctx, msg)
	require.NoError(t, err, "Preservation: Enqueue should succeed")

	// Dequeue the message
	dequeuedMsg, err := messageQueue.Dequeue(ctx, tenantID)
	require.NoError(t, err, "Preservation: Dequeue should succeed")
	require.NotNil(t, dequeuedMsg, "Preservation: Dequeued message should not be nil")

	// Verify message content
	assert.Equal(t, msg.MessageID, dequeuedMsg.MessageID,
		"Preservation: Dequeued message should match enqueued message")
	assert.Equal(t, msg.PhoneNumber, dequeuedMsg.PhoneNumber,
		"Preservation: Phone number should be preserved")
	assert.Equal(t, msg.Content, dequeuedMsg.Content,
		"Preservation: Content should be preserved")
	assert.Equal(t, msg.Priority, dequeuedMsg.Priority,
		"Preservation: Priority should be preserved")

	// Cleanup
	messageQueue.Clear(ctx, tenantID)
}
