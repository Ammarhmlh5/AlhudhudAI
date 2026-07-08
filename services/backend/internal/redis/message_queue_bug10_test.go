package redis

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// Bug 10 Exploration Test: Improper Priority Queue Ordering
// **Validates: Requirements 1.10, 2.10**
//
// **Property 1: Bug Condition** - Priority Ordering Not Enforced
//
// This test verifies whether DequeueBatch() properly enforces strict priority ordering.
// The bug hypothesis: DequeueBatch() may not fully drain high priority queue before
// moving to normal priority, and may not fully drain normal before moving to low.
//
// CRITICAL: This test encodes the EXPECTED behavior (strict priority ordering).
// On UNFIXED code: Test MAY FAIL if priority ordering is violated
// On FIXED code: Test PASSES (strict ordering enforced)
//
// IMPORTANT: This bug may not exist - the current implementation may already be correct.
// If tests pass, it means the code already enforces strict priority ordering.
//
// DO NOT attempt to fix the test or the code when it fails.
// Document the failure as proof of the bug.
func TestProperty_Bug10_PriorityOrderingNotEnforced(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property-based test in short mode")
	}

	client, mr := setupTestRedis(t)
	defer mr.Close()

	mq := NewMessageQueue(client)
	ctx := context.Background()

	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		
		// Clear queue before each test
		_ = mq.Clear(ctx, tenantID)

		// Generate random message counts for each priority
		highCount := rapid.IntRange(50, 200).Draw(t, "highCount")
		normalCount := rapid.IntRange(50, 200).Draw(t, "normalCount")
		lowCount := rapid.IntRange(20, 100).Draw(t, "lowCount")

		// Enqueue high priority messages
		for i := 0; i < highCount; i++ {
			item := &MessageQueueItem{
				MessageID:   uuid.New(),
				TenantID:    tenantID,
				PhoneNumber: "+966500000001",
				Content:     "High priority message",
				Priority:    1, // High
				PartsCount:  1,
				Cost:        0.1,
			}
			err := mq.Enqueue(ctx, item)
			require.NoError(t, err)
		}

		// Enqueue normal priority messages
		for i := 0; i < normalCount; i++ {
			item := &MessageQueueItem{
				MessageID:   uuid.New(),
				TenantID:    tenantID,
				PhoneNumber: "+966500000002",
				Content:     "Normal priority message",
				Priority:    2, // Normal
				PartsCount:  1,
				Cost:        0.1,
			}
			err := mq.Enqueue(ctx, item)
			require.NoError(t, err)
		}

		// Enqueue low priority messages
		for i := 0; i < lowCount; i++ {
			item := &MessageQueueItem{
				MessageID:   uuid.New(),
				TenantID:    tenantID,
				PhoneNumber: "+966500000003",
				Content:     "Low priority message",
				Priority:    3, // Low
				PartsCount:  1,
				Cost:        0.1,
			}
			err := mq.Enqueue(ctx, item)
			require.NoError(t, err)
		}

		// Dequeue in batches and verify strict priority ordering
		batchSize := 50
		var allMessages []*MessageQueueItem
		
		for {
			batch, err := mq.DequeueBatch(ctx, tenantID, batchSize)
			require.NoError(t, err)
			
			if len(batch) == 0 {
				break
			}
			
			allMessages = append(allMessages, batch...)
		}

		// Verify we got all messages
		totalExpected := highCount + normalCount + lowCount
		assert.Equal(t, totalExpected, len(allMessages), 
			"Should dequeue all messages (high=%d, normal=%d, low=%d)", 
			highCount, normalCount, lowCount)

		// Verify strict priority ordering:
		// All high priority (1) should come before any normal (2)
		// All normal priority (2) should come before any low (3)
		lastSeenPriority := 0
		highSeen := 0
		normalSeen := 0
		lowSeen := 0

		for i, msg := range allMessages {
			if msg.Priority < lastSeenPriority {
				t.Fatalf("Priority ordering violation at index %d: "+
					"saw priority %d after priority %d (high=%d, normal=%d, low=%d seen so far)",
					i, msg.Priority, lastSeenPriority, highSeen, normalSeen, lowSeen)
			}
			
			if msg.Priority > lastSeenPriority {
				lastSeenPriority = msg.Priority
			}

			switch msg.Priority {
			case 1:
				highSeen++
			case 2:
				normalSeen++
			case 3:
				lowSeen++
			}
		}

		// Verify counts match
		assert.Equal(t, highCount, highSeen, "High priority count mismatch")
		assert.Equal(t, normalCount, normalSeen, "Normal priority count mismatch")
		assert.Equal(t, lowCount, lowSeen, "Low priority count mismatch")
	})
}

// TestBug10_StrictPriorityOrdering tests specific scenario with known counts
// **Validates: Requirements 1.10, 2.10**
//
// This test uses the exact scenario from the design document:
// 150 high, 100 normal, 50 low priority messages, dequeued in batches of 50.
func TestBug10_StrictPriorityOrdering(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	mq := NewMessageQueue(client)
	ctx := context.Background()
	tenantID := uuid.New()

	// Clear queue
	_ = mq.Clear(ctx, tenantID)

	// Enqueue 150 high, 100 normal, 50 low
	for i := 0; i < 150; i++ {
		item := &MessageQueueItem{
			MessageID:   uuid.New(),
			TenantID:    tenantID,
			PhoneNumber: "+966500000001",
			Content:     "High priority",
			Priority:    1,
			PartsCount:  1,
			Cost:        0.1,
		}
		err := mq.Enqueue(ctx, item)
		require.NoError(t, err)
	}

	for i := 0; i < 100; i++ {
		item := &MessageQueueItem{
			MessageID:   uuid.New(),
			TenantID:    tenantID,
			PhoneNumber: "+966500000002",
			Content:     "Normal priority",
			Priority:    2,
			PartsCount:  1,
			Cost:        0.1,
		}
		err := mq.Enqueue(ctx, item)
		require.NoError(t, err)
	}

	for i := 0; i < 50; i++ {
		item := &MessageQueueItem{
			MessageID:   uuid.New(),
			TenantID:    tenantID,
			PhoneNumber: "+966500000003",
			Content:     "Low priority",
			Priority:    3,
			PartsCount:  1,
			Cost:        0.1,
		}
		err := mq.Enqueue(ctx, item)
		require.NoError(t, err)
	}

	// Dequeue in batches of 50
	var allMessages []*MessageQueueItem
	batchSize := 50

	for {
		batch, err := mq.DequeueBatch(ctx, tenantID, batchSize)
		require.NoError(t, err)

		if len(batch) == 0 {
			break
		}

		allMessages = append(allMessages, batch...)
	}

	// Verify total count
	assert.Equal(t, 300, len(allMessages), "Should dequeue all 300 messages")

	// Verify strict ordering
	// First 150 should be high priority
	for i := 0; i < 150; i++ {
		assert.Equal(t, 1, allMessages[i].Priority, 
			"Message %d should be high priority", i)
	}

	// Next 100 should be normal priority
	for i := 150; i < 250; i++ {
		assert.Equal(t, 2, allMessages[i].Priority, 
			"Message %d should be normal priority", i)
	}

	// Last 50 should be low priority
	for i := 250; i < 300; i++ {
		assert.Equal(t, 3, allMessages[i].Priority, 
			"Message %d should be low priority", i)
	}

	t.Log("✅ Strict priority ordering verified: high → normal → low")
}

// TestBug10_PriorityOrderingWithSmallBatches tests with very small batch sizes
// **Validates: Requirements 1.10, 2.10**
//
// This test verifies that priority ordering is maintained even when using
// very small batch sizes that might expose edge cases in the batching logic.
func TestBug10_PriorityOrderingWithSmallBatches(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	mq := NewMessageQueue(client)
	ctx := context.Background()
	tenantID := uuid.New()

	// Clear queue
	_ = mq.Clear(ctx, tenantID)

	// Enqueue 10 of each priority
	for priority := 1; priority <= 3; priority++ {
		for i := 0; i < 10; i++ {
			item := &MessageQueueItem{
				MessageID:   uuid.New(),
				TenantID:    tenantID,
				PhoneNumber: "+966500000000",
				Content:     "Test message",
				Priority:    priority,
				PartsCount:  1,
				Cost:        0.1,
			}
			err := mq.Enqueue(ctx, item)
			require.NoError(t, err)
		}
	}

	// Dequeue with very small batches (5 messages at a time)
	var allMessages []*MessageQueueItem
	batchSize := 5

	for {
		batch, err := mq.DequeueBatch(ctx, tenantID, batchSize)
		require.NoError(t, err)

		if len(batch) == 0 {
			break
		}

		allMessages = append(allMessages, batch...)
	}

	// Verify strict ordering even with small batches
	lastPriority := 0
	for i, msg := range allMessages {
		if msg.Priority < lastPriority {
			t.Fatalf("Priority violation at index %d: saw %d after %d", 
				i, msg.Priority, lastPriority)
		}
		if msg.Priority > lastPriority {
			lastPriority = msg.Priority
		}
	}

	t.Log("✅ Strict priority ordering maintained with small batches")
}

// TestBug10_PriorityOrderingWithLargeBatches tests with batch size larger than queue
// **Validates: Requirements 1.10, 2.10**
//
// This test verifies that priority ordering is maintained when the batch size
// is larger than the number of messages in any single priority queue.
func TestBug10_PriorityOrderingWithLargeBatches(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	mq := NewMessageQueue(client)
	ctx := context.Background()
	tenantID := uuid.New()

	// Clear queue
	_ = mq.Clear(ctx, tenantID)

	// Enqueue 20 high, 15 normal, 10 low
	for i := 0; i < 20; i++ {
		item := &MessageQueueItem{
			MessageID:   uuid.New(),
			TenantID:    tenantID,
			PhoneNumber: "+966500000001",
			Content:     "High priority",
			Priority:    1,
			PartsCount:  1,
			Cost:        0.1,
		}
		err := mq.Enqueue(ctx, item)
		require.NoError(t, err)
	}

	for i := 0; i < 15; i++ {
		item := &MessageQueueItem{
			MessageID:   uuid.New(),
			TenantID:    tenantID,
			PhoneNumber: "+966500000002",
			Content:     "Normal priority",
			Priority:    2,
			PartsCount:  1,
			Cost:        0.1,
		}
		err := mq.Enqueue(ctx, item)
		require.NoError(t, err)
	}

	for i := 0; i < 10; i++ {
		item := &MessageQueueItem{
			MessageID:   uuid.New(),
			TenantID:    tenantID,
			PhoneNumber: "+966500000003",
			Content:     "Low priority",
			Priority:    3,
			PartsCount:  1,
			Cost:        0.1,
		}
		err := mq.Enqueue(ctx, item)
		require.NoError(t, err)
	}

	// Dequeue with batch size of 100 (larger than any single queue)
	batch, err := mq.DequeueBatch(ctx, tenantID, 100)
	require.NoError(t, err)

	// Should get all 45 messages
	assert.Equal(t, 45, len(batch), "Should dequeue all messages in one batch")

	// Verify strict ordering
	// First 20 should be high priority
	for i := 0; i < 20; i++ {
		assert.Equal(t, 1, batch[i].Priority, 
			"Message %d should be high priority", i)
	}

	// Next 15 should be normal priority
	for i := 20; i < 35; i++ {
		assert.Equal(t, 2, batch[i].Priority, 
			"Message %d should be normal priority", i)
	}

	// Last 10 should be low priority
	for i := 35; i < 45; i++ {
		assert.Equal(t, 3, batch[i].Priority, 
			"Message %d should be low priority", i)
	}

	t.Log("✅ Strict priority ordering maintained with large batch size")
}

// TestBug10_PriorityOrderingAcrossMultipleCalls tests ordering across multiple dequeue calls
// **Validates: Requirements 1.10, 2.10**
//
// This test verifies that priority ordering is maintained across multiple
// DequeueBatch calls, ensuring that we don't start processing lower priority
// messages while higher priority messages still exist.
func TestBug10_PriorityOrderingAcrossMultipleCalls(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	mq := NewMessageQueue(client)
	ctx := context.Background()
	tenantID := uuid.New()

	// Clear queue
	_ = mq.Clear(ctx, tenantID)

	// Enqueue 75 high, 50 normal, 25 low
	for i := 0; i < 75; i++ {
		item := &MessageQueueItem{
			MessageID:   uuid.New(),
			TenantID:    tenantID,
			PhoneNumber: "+966500000001",
			Content:     "High priority",
			Priority:    1,
			PartsCount:  1,
			Cost:        0.1,
		}
		err := mq.Enqueue(ctx, item)
		require.NoError(t, err)
	}

	for i := 0; i < 50; i++ {
		item := &MessageQueueItem{
			MessageID:   uuid.New(),
			TenantID:    tenantID,
			PhoneNumber: "+966500000002",
			Content:     "Normal priority",
			Priority:    2,
			PartsCount:  1,
			Cost:        0.1,
		}
		err := mq.Enqueue(ctx, item)
		require.NoError(t, err)
	}

	for i := 0; i < 25; i++ {
		item := &MessageQueueItem{
			MessageID:   uuid.New(),
			TenantID:    tenantID,
			PhoneNumber: "+966500000003",
			Content:     "Low priority",
			Priority:    3,
			PartsCount:  1,
			Cost:        0.1,
		}
		err := mq.Enqueue(ctx, item)
		require.NoError(t, err)
	}

	// Dequeue in batches of 30
	batchSize := 30
	var allMessages []*MessageQueueItem

	// First batch: should get 30 high priority
	batch1, err := mq.DequeueBatch(ctx, tenantID, batchSize)
	require.NoError(t, err)
	assert.Equal(t, 30, len(batch1))
	for i, msg := range batch1 {
		assert.Equal(t, 1, msg.Priority, "Batch 1, message %d should be high priority", i)
	}
	allMessages = append(allMessages, batch1...)

	// Second batch: should get 30 high priority
	batch2, err := mq.DequeueBatch(ctx, tenantID, batchSize)
	require.NoError(t, err)
	assert.Equal(t, 30, len(batch2))
	for i, msg := range batch2 {
		assert.Equal(t, 1, msg.Priority, "Batch 2, message %d should be high priority", i)
	}
	allMessages = append(allMessages, batch2...)

	// Third batch: should get 15 high + 15 normal
	batch3, err := mq.DequeueBatch(ctx, tenantID, batchSize)
	require.NoError(t, err)
	assert.Equal(t, 30, len(batch3))
	for i := 0; i < 15; i++ {
		assert.Equal(t, 1, batch3[i].Priority, "Batch 3, message %d should be high priority", i)
	}
	for i := 15; i < 30; i++ {
		assert.Equal(t, 2, batch3[i].Priority, "Batch 3, message %d should be normal priority", i)
	}
	allMessages = append(allMessages, batch3...)

	// Fourth batch: should get 30 normal
	batch4, err := mq.DequeueBatch(ctx, tenantID, batchSize)
	require.NoError(t, err)
	assert.Equal(t, 30, len(batch4))
	for i, msg := range batch4 {
		assert.Equal(t, 2, msg.Priority, "Batch 4, message %d should be normal priority", i)
	}
	allMessages = append(allMessages, batch4...)

	// Fifth batch: should get 5 normal + 25 low
	batch5, err := mq.DequeueBatch(ctx, tenantID, batchSize)
	require.NoError(t, err)
	assert.Equal(t, 30, len(batch5))
	for i := 0; i < 5; i++ {
		assert.Equal(t, 2, batch5[i].Priority, "Batch 5, message %d should be normal priority", i)
	}
	for i := 5; i < 30; i++ {
		assert.Equal(t, 3, batch5[i].Priority, "Batch 5, message %d should be low priority", i)
	}
	allMessages = append(allMessages, batch5...)

	// Verify total count
	assert.Equal(t, 150, len(allMessages), "Should have dequeued all 150 messages")

	// Verify overall strict ordering
	lastPriority := 0
	for i, msg := range allMessages {
		if msg.Priority < lastPriority {
			t.Fatalf("Priority violation at index %d: saw %d after %d", 
				i, msg.Priority, lastPriority)
		}
		if msg.Priority > lastPriority {
			lastPriority = msg.Priority
		}
	}

	t.Log("✅ Strict priority ordering maintained across multiple DequeueBatch calls")
}
