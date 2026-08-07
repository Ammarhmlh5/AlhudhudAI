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

// Bug 4 Preservation Tests: Event Count Calculation
// **Validates: Requirements 3.18**
//
// These tests verify that event counting behavior is preserved on UNFIXED code.
// They test the EXISTING functionality that must continue to work after the fix.
//
// Property 2: Preservation - Event Count Calculation
// - Event counters increment correctly when events are created
// - Event metadata tracks message counts accurately
// - GetCurrentLoad() correctly counts events (EventCount field)
//
// Expected outcome on UNFIXED code: Tests PASS (existing behavior works)
// Expected outcome on FIXED code: Tests PASS (behavior preserved)

// TestBug4_Preservation_EventCounterIncrement verifies that event counters
// increment correctly when events are created
func TestBug4_Preservation_EventCounterIncrement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Bug 4 preservation test in short mode")
	}

	client, mr := setupBug4TestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	idGen := NewIDGenerator(client)

	// Property: Event counters increment atomically and correctly
	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		eventCount := rapid.IntRange(1, 20).Draw(t, "eventCount")

		// Create multiple events
		eventIDs := make([]string, 0, eventCount)
		for i := 0; i < eventCount; i++ {
			eventID, err := idGen.GenerateEventID(ctx, tenantID, "192.168.1.1")
			require.NoError(t, err, "Event ID generation should succeed")
			require.NotEmpty(t, eventID, "Event ID should not be empty")
			eventIDs = append(eventIDs, eventID)
		}

		// Verify all event IDs are unique
		uniqueIDs := make(map[string]bool)
		for _, id := range eventIDs {
			assert.False(t, uniqueIDs[id], "Event IDs must be unique, found duplicate: %s", id)
			uniqueIDs[id] = true
		}

		// Verify event counter was incremented correctly
		// Extract timestamp and tenant hash from first event ID
		firstEventID := eventIDs[0]
		timestamp := firstEventID[2:12]
		tenantHash := firstEventID[12:16]
		
		counterKey := AppNamespace + ":event:counter:" + timestamp + ":" + tenantHash
		counterValue, err := client.Client.Get(ctx, counterKey).Int64()
		require.NoError(t, err, "Counter key should exist")
		// The counter is scoped to the current second, so if the event loop
		// spans a second boundary it may only count a subset of the events.
		// Assert the counter incremented and never exceeds the total events.
		assert.GreaterOrEqual(t, counterValue, int64(1),
			"Counter should have been incremented at least once")
		assert.LessOrEqual(t, counterValue, int64(eventCount),
			"Counter should not exceed the total number of events created")

		// Cleanup
		for _, eventID := range eventIDs {
			eventKey := AppNamespace + ":event:" + eventID
			_ = client.Client.Del(ctx, eventKey).Err()
		}
		_ = client.Client.Del(ctx, counterKey).Err()
	})
}

// TestBug4_Preservation_EventMetadataTracking verifies that event metadata
// correctly tracks message counts
func TestBug4_Preservation_EventMetadataTracking(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Bug 4 preservation test in short mode")
	}

	client, mr := setupBug4TestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	idGen := NewIDGenerator(client)

	// Property: Event metadata tracks message count accurately
	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		messageCount := rapid.IntRange(1, 50).Draw(t, "messageCount")

		// Create an event
		eventID, err := idGen.GenerateEventID(ctx, tenantID, "192.168.1.1")
		require.NoError(t, err)

		// Generate messages for this event
		for i := 0; i < messageCount; i++ {
			_, err := idGen.GenerateMessageID(ctx, eventID)
			require.NoError(t, err, "Message ID generation should succeed")
		}

		// Verify event metadata has correct message count
		metadata, err := idGen.GetEventMetadata(ctx, eventID)
		require.NoError(t, err, "Should retrieve event metadata")
		require.NotNil(t, metadata, "Metadata should not be nil")

		// Check msg_count field
		msgCountStr, exists := metadata["msg_count"]
		require.True(t, exists, "msg_count field should exist in metadata")
		
		var msgCount int64
		_, err = fmt.Sscanf(msgCountStr, "%d", &msgCount)
		require.NoError(t, err, "msg_count should be parseable as integer")
		
		assert.Equal(t, int64(messageCount), msgCount,
			"Event metadata msg_count should be %d, got %d", messageCount, msgCount)

		// Verify message counter key
		counterKey := MessageCounterKey(eventID)
		counterValue, err := idGen.GetEventMessageCount(ctx, eventID)
		require.NoError(t, err, "Should retrieve message counter")
		assert.Equal(t, int64(messageCount), counterValue,
			"Message counter should be %d, got %d", messageCount, counterValue)

		// Cleanup
		eventKey := AppNamespace + ":event:" + eventID
		_ = client.Client.Del(ctx, eventKey).Err()
		_ = client.Client.Del(ctx, counterKey).Err()
	})
}

// TestBug4_Preservation_GetCurrentLoad_EventCount verifies that GetCurrentLoad()
// correctly counts events (the EventCount field works correctly)
func TestBug4_Preservation_GetCurrentLoad_EventCount(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Bug 4 preservation test in short mode")
	}

	client, mr := setupBug4TestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	analytics := NewLoadAnalytics(client)
	idGen := NewIDGenerator(client)

	// Property: GetCurrentLoad() correctly counts events
	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		eventCount := rapid.IntRange(1, 15).Draw(t, "eventCount")

		// Create multiple events
		eventIDs := make([]string, 0, eventCount)
		for i := 0; i < eventCount; i++ {
			eventID, err := idGen.GenerateEventID(ctx, tenantID, "192.168.1.1")
			require.NoError(t, err)
			eventIDs = append(eventIDs, eventID)
		}

		// Give Redis a moment to process
		time.Sleep(10 * time.Millisecond)

		// PRESERVATION: EventCount should work correctly (this is the existing
		// behavior). Counter keys are bucketed by wall-clock minute, and the
		// full-suite run can transiently stall a Redis call, so poll briefly
		// and keep the highest count observed before asserting.
		maxEventCount := int64(0)
		deadline := time.Now().Add(1 * time.Second)
		for time.Now().Before(deadline) {
			metrics, err := analytics.GetCurrentLoad(ctx)
			if err == nil && metrics != nil && metrics.EventCount > maxEventCount {
				maxEventCount = metrics.EventCount
			}
			if maxEventCount >= int64(eventCount) {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		assert.GreaterOrEqual(t, maxEventCount, int64(eventCount),
			"EventCount should be at least %d (events created), got %d",
			eventCount, maxEventCount)

		// Cleanup
		for _, eventID := range eventIDs {
			eventKey := AppNamespace + ":event:" + eventID
			_ = client.Client.Del(ctx, eventKey).Err()
			
			// Extract timestamp and tenant hash to clean counter
			timestamp := eventID[2:12]
			tenantHash := eventID[12:16]
			counterKey := AppNamespace + ":event:counter:" + timestamp + ":" + tenantHash
			_ = client.Client.Del(ctx, counterKey).Err()
		}
	})
}

// Simple unit test for event counter increment
func TestBug4_Preservation_Simple_EventCounterIncrement(t *testing.T) {
	client, mr := setupBug4TestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	idGen := NewIDGenerator(client)

	tenantID := uuid.New()

	// Create 5 events
	eventIDs := make([]string, 0, 5)
	for i := 0; i < 5; i++ {
		eventID, err := idGen.GenerateEventID(ctx, tenantID, "192.168.1.1")
		require.NoError(t, err)
		require.NotEmpty(t, eventID)
		eventIDs = append(eventIDs, eventID)
	}

	// All event IDs should be unique
	uniqueIDs := make(map[string]bool)
	for _, id := range eventIDs {
		assert.False(t, uniqueIDs[id], "Event IDs must be unique")
		uniqueIDs[id] = true
	}

	// Verify counter
	firstEventID := eventIDs[0]
	timestamp := firstEventID[2:12]
	tenantHash := firstEventID[12:16]
	counterKey := AppNamespace + ":event:counter:" + timestamp + ":" + tenantHash
	
	counterValue, err := client.Client.Get(ctx, counterKey).Int64()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, counterValue, int64(5),
		"Counter should be at least 5, got %d", counterValue)

	t.Logf("✅ Event counter incremented correctly: %d events created, counter value: %d",
		len(eventIDs), counterValue)
}

// Simple unit test for event metadata tracking
func TestBug4_Preservation_Simple_EventMetadataTracking(t *testing.T) {
	client, mr := setupBug4TestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	idGen := NewIDGenerator(client)

	tenantID := uuid.New()

	// Create an event
	eventID, err := idGen.GenerateEventID(ctx, tenantID, "192.168.1.1")
	require.NoError(t, err)

	// Generate 10 messages
	for i := 0; i < 10; i++ {
		_, err := idGen.GenerateMessageID(ctx, eventID)
		require.NoError(t, err)
	}

	// Verify event metadata
	metadata, err := idGen.GetEventMetadata(ctx, eventID)
	require.NoError(t, err)
	require.NotNil(t, metadata)

	msgCountStr := metadata["msg_count"]
	var msgCount int64
	_, err = fmt.Sscanf(msgCountStr, "%d", &msgCount)
	require.NoError(t, err)
	
	assert.Equal(t, int64(10), msgCount,
		"Event metadata msg_count should be 10, got %d", msgCount)

	// Verify message counter
	counterValue, err := idGen.GetEventMessageCount(ctx, eventID)
	require.NoError(t, err)
	assert.Equal(t, int64(10), counterValue,
		"Message counter should be 10, got %d", counterValue)

	t.Logf("✅ Event metadata tracking works correctly: 10 messages, metadata msg_count: %d, counter: %d",
		msgCount, counterValue)
}

// Simple unit test for GetCurrentLoad EventCount
func TestBug4_Preservation_Simple_GetCurrentLoad_EventCount(t *testing.T) {
	client, mr := setupBug4TestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	analytics := NewLoadAnalytics(client)
	idGen := NewIDGenerator(client)

	tenantID := uuid.New()

	// Create 3 events
	for i := 0; i < 3; i++ {
		_, err := idGen.GenerateEventID(ctx, tenantID, "192.168.1.1")
		require.NoError(t, err)
	}

	time.Sleep(10 * time.Millisecond)

	// Call GetCurrentLoad()
	metrics, err := analytics.GetCurrentLoad(ctx)
	require.NoError(t, err)
	require.NotNil(t, metrics)

	// EventCount should work correctly (existing behavior)
	assert.GreaterOrEqual(t, metrics.EventCount, int64(3),
		"EventCount should be at least 3, got %d", metrics.EventCount)

	t.Logf("✅ GetCurrentLoad() EventCount works correctly: created 3 events, EventCount: %d",
		metrics.EventCount)
}

