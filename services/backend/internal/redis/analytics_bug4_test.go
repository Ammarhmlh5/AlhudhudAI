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

// Bug 4 Exploration Test: GetCurrentLoad() returns MessageCount: 0 despite messages existing
// **Validates: Requirements 1.4, 2.4**
//
// This test MUST FAIL on unfixed code - failure confirms the bug exists.
// The bug: GetCurrentLoad() does not count actual messages, returning MessageCount: 0
// even when messages exist in the system.
//
// Expected outcome on UNFIXED code: Test FAILS (MessageCount is 0)
// Expected outcome on FIXED code: Test PASSES (MessageCount > 0)

func TestBug4_GetCurrentLoad_ReturnsZeroMessages(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Bug 4 exploration test in short mode")
	}

	// Setup Redis client with miniredis
	client, mr := setupBug4TestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	analytics := NewLoadAnalytics(client)
	messageQueue := NewMessageQueue(client)
	idGen := NewIDGenerator(client)

	// Property: When messages exist in the system, GetCurrentLoad() should return MessageCount > 0
	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		tenantID := uuid.New()
		messageCount := rapid.IntRange(5, 50).Draw(t, "messageCount")
		
		// Create an event with messages
		eventID, err := idGen.GenerateEventID(ctx, tenantID, "192.168.1.1")
		require.NoError(t, err, "Failed to generate event ID")

		// Generate message IDs for this event (this updates the event metadata)
		for i := 0; i < messageCount; i++ {
			_, err := idGen.GenerateMessageID(ctx, eventID)
			require.NoError(t, err, "Failed to generate message ID")
		}

		// Enqueue messages to the queue
		for i := 0; i < messageCount; i++ {
			item := &MessageQueueItem{
				MessageID:   uuid.New(),
				TenantID:    tenantID,
				PhoneNumber: "+966500000000",
				Content:     "Test message",
				PartsCount:  1,
				Cost:        0.05,
				Priority:    2, // normal priority
				EnqueuedAt:  time.Now(),
			}
			err := messageQueue.Enqueue(ctx, item)
			require.NoError(t, err, "Failed to enqueue message")
		}

		// Give Redis a moment to process
		time.Sleep(10 * time.Millisecond)

		// Call GetCurrentLoad()
		metrics, err := analytics.GetCurrentLoad(ctx)
		require.NoError(t, err, "GetCurrentLoad should not return error")
		require.NotNil(t, metrics, "Metrics should not be nil")

		// BUG CONDITION: On unfixed code, MessageCount will be 0
		// This assertion will FAIL on unfixed code, confirming the bug exists
		assert.Greater(t, metrics.MessageCount, int64(0),
			"BUG DETECTED: GetCurrentLoad() returned MessageCount=0 despite %d messages existing in system. "+
				"Event ID: %s, EventCount: %d, Expected MessageCount: >0, Actual: %d",
			messageCount, eventID, metrics.EventCount, metrics.MessageCount)

		// Additional verification: MessageCount should be at least the number of messages we created
		assert.GreaterOrEqual(t, metrics.MessageCount, int64(messageCount),
			"MessageCount should be at least %d (messages created), got %d",
			messageCount, metrics.MessageCount)

		// Cleanup for next iteration
		_ = messageQueue.Clear(ctx, tenantID)
		eventKey := AppNamespace + ":event:" + eventID
		_ = client.Client.Del(ctx, eventKey).Err()
		counterKey := MessageCounterKey(eventID)
		_ = client.Client.Del(ctx, counterKey).Err()
	})
}

// Simple unit test version for quick verification
func TestBug4_Simple_GetCurrentLoad_ReturnsZeroMessages(t *testing.T) {
	// Setup Redis client with miniredis
	client, mr := setupBug4TestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	analytics := NewLoadAnalytics(client)
	messageQueue := NewMessageQueue(client)
	idGen := NewIDGenerator(client)

	// Create a tenant and event
	tenantID := uuid.New()
	eventID, err := idGen.GenerateEventID(ctx, tenantID, "192.168.1.1")
	require.NoError(t, err)

	// Generate 10 message IDs (updates event metadata)
	for i := 0; i < 10; i++ {
		_, err := idGen.GenerateMessageID(ctx, eventID)
		require.NoError(t, err)
	}

	// Enqueue 10 messages to the queue
	for i := 0; i < 10; i++ {
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
		require.NoError(t, err)
	}

	// Verify messages are in queue
	queueSize, err := messageQueue.GetQueueSize(ctx, tenantID)
	require.NoError(t, err)
	assert.Equal(t, int64(10), queueSize, "Queue should have 10 messages")

	// Verify event metadata has message count
	metadata, err := idGen.GetEventMetadata(ctx, eventID)
	require.NoError(t, err)
	t.Logf("Event metadata: %+v", metadata)

	// Call GetCurrentLoad()
	metrics, err := analytics.GetCurrentLoad(ctx)
	require.NoError(t, err)
	require.NotNil(t, metrics)

	t.Logf("GetCurrentLoad() returned: EventCount=%d, MessageCount=%d, AvgMsgPerEvent=%.2f",
		metrics.EventCount, metrics.MessageCount, metrics.AvgMsgPerEvent)

	// BUG CONDITION: On unfixed code, MessageCount will be 0
	// This assertion will FAIL on unfixed code
	assert.Greater(t, metrics.MessageCount, int64(0),
		"BUG DETECTED: GetCurrentLoad() returned MessageCount=0 despite 10 messages existing. "+
			"EventCount: %d, MessageCount: %d",
		metrics.EventCount, metrics.MessageCount)

	// Should count at least the 10 messages we created
	assert.GreaterOrEqual(t, metrics.MessageCount, int64(10),
		"MessageCount should be at least 10, got %d", metrics.MessageCount)
}

// Helper function for Bug 4 test setup (uses miniredis like other tests)
func setupBug4TestRedis(t *testing.T) (*Client, *miniredis.Miniredis) {
	mr := miniredis.RunT(t)
	
	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return &Client{
		Client: redisClient,
	}, mr
}
