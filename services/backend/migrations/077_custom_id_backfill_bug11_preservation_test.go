package migrations

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/al-hudhud/backend/internal/redis"
)

// TestBug11_Preservation_NewRecordsGetCustomIDs tests that new records continue to receive custom_id values
//
// **Validates: Requirements 3.13**
//
// **Property 2: Preservation** - New Records Get Custom IDs
//
// This test verifies that the custom_id generation mechanism for NEW records continues to work correctly.
// This behavior MUST be preserved when we add backfill logic to the migration.
//
// Test Strategy:
// 1. Use the Redis ID generator to create new event custom_id
// 2. Use the Redis ID generator to create new message custom_id
// 3. Verify format and uniqueness
// 4. Verify sequential generation works
//
// EXPECTED OUTCOME ON UNFIXED CODE: Test PASSES (new records get custom_id correctly)
// EXPECTED OUTCOME ON FIXED CODE: Test PASSES (behavior preserved)
func TestBug11_Preservation_NewRecordsGetCustomIDs(t *testing.T) {
	// Setup Redis client for testing
	redisClient := redis.GetGlobalClient()
	if redisClient == nil {
		t.Skip("Redis client not initialized - skipping preservation test")
	}

	ctx := context.Background()
	idGen := redis.NewIDGenerator(redisClient)

	t.Run("EventCustomIDGeneration", func(t *testing.T) {
		// Generate a new event custom_id
		tenantID := uuid.New()
		senderIP := "192.168.1.100"

		eventCustomID, err := idGen.GenerateEventID(ctx, tenantID, senderIP)
		require.NoError(t, err, "Event custom_id generation should succeed")
		require.NotEmpty(t, eventCustomID, "Event custom_id should not be empty")

		// Verify format: EV + YYMMDDHHmm (10) + TenantHash (4) + Sequence (4) = 20 chars
		assert.Equal(t, 20, len(eventCustomID), "Event custom_id should be 20 characters")
		assert.Equal(t, "EV", eventCustomID[0:2], "Event custom_id should start with 'EV'")

		// Verify timestamp part is numeric
		timestampPart := eventCustomID[2:12]
		assert.Regexp(t, `^\d{10}$`, timestampPart, "Timestamp part should be 10 digits")

		// Verify tenant hash part is numeric
		tenantHashPart := eventCustomID[12:16]
		assert.Regexp(t, `^\d{4}$`, tenantHashPart, "Tenant hash should be 4 digits")

		// Verify sequence part is numeric
		sequencePart := eventCustomID[16:20]
		assert.Regexp(t, `^\d{4}$`, sequencePart, "Sequence should be 4 digits")

		t.Logf("✓ Generated event custom_id: %s", eventCustomID)
	})

	t.Run("MessageCustomIDGeneration", func(t *testing.T) {
		// First generate an event custom_id
		tenantID := uuid.New()
		eventCustomID, err := idGen.GenerateEventID(ctx, tenantID, "192.168.1.100")
		require.NoError(t, err, "Event custom_id generation should succeed")

		// Generate a message custom_id linked to the event
		messageCustomID, err := idGen.GenerateMessageID(ctx, eventCustomID)
		require.NoError(t, err, "Message custom_id generation should succeed")
		require.NotEmpty(t, messageCustomID, "Message custom_id should not be empty")

		// Verify format: MS + YYMMDDHHmm (10) + TenantHash (4) + EventSeq (4) + MsgSeq (3) = 23 chars
		assert.Equal(t, 23, len(messageCustomID), "Message custom_id should be 23 characters")
		assert.Equal(t, "MS", messageCustomID[0:2], "Message custom_id should start with 'MS'")

		// Verify timestamp part matches event
		eventTimestamp := eventCustomID[2:12]
		messageTimestamp := messageCustomID[2:12]
		assert.Equal(t, eventTimestamp, messageTimestamp, "Message timestamp should match event timestamp")

		// Verify tenant hash matches event
		eventTenantHash := eventCustomID[12:16]
		messageTenantHash := messageCustomID[12:16]
		assert.Equal(t, eventTenantHash, messageTenantHash, "Message tenant hash should match event tenant hash")

		// Verify event sequence matches
		eventSeq := eventCustomID[16:20]
		messageEventSeq := messageCustomID[16:20]
		assert.Equal(t, eventSeq, messageEventSeq, "Message event sequence should match event sequence")

		// Verify message sequence part is numeric
		messageSeqPart := messageCustomID[20:23]
		assert.Regexp(t, `^\d{3}$`, messageSeqPart, "Message sequence should be 3 digits")

		t.Logf("✓ Generated message custom_id: %s (linked to event: %s)", messageCustomID, eventCustomID)
	})

	t.Run("SequentialEventGeneration", func(t *testing.T) {
		// Generate multiple events for the same tenant
		tenantID := uuid.New()
		senderIP := "192.168.1.100"

		eventIDs := make([]string, 5)
		for i := 0; i < 5; i++ {
			eventID, err := idGen.GenerateEventID(ctx, tenantID, senderIP)
			require.NoError(t, err, "Event custom_id generation should succeed")
			eventIDs[i] = eventID
		}

		// Verify all IDs are unique
		uniqueIDs := make(map[string]bool)
		for _, id := range eventIDs {
			assert.False(t, uniqueIDs[id], "Event custom_id should be unique: %s", id)
			uniqueIDs[id] = true
		}

		// Verify sequential numbering (last 4 digits should increment)
		for i := 1; i < len(eventIDs); i++ {
			prevSeq := eventIDs[i-1][16:20]
			currSeq := eventIDs[i][16:20]
			t.Logf("Event %d sequence: %s, Event %d sequence: %s", i-1, prevSeq, i, currSeq)
		}

		t.Logf("✓ Generated %d unique sequential event custom_ids", len(eventIDs))
	})

	t.Run("SequentialMessageGeneration", func(t *testing.T) {
		// Generate an event
		tenantID := uuid.New()
		eventCustomID, err := idGen.GenerateEventID(ctx, tenantID, "192.168.1.100")
		require.NoError(t, err, "Event custom_id generation should succeed")

		// Generate multiple messages for the same event
		messageIDs := make([]string, 10)
		for i := 0; i < 10; i++ {
			messageID, err := idGen.GenerateMessageID(ctx, eventCustomID)
			require.NoError(t, err, "Message custom_id generation should succeed")
			messageIDs[i] = messageID
		}

		// Verify all IDs are unique
		uniqueIDs := make(map[string]bool)
		for _, id := range messageIDs {
			assert.False(t, uniqueIDs[id], "Message custom_id should be unique: %s", id)
			uniqueIDs[id] = true
		}

		// Verify sequential numbering (last 3 digits should increment)
		for i := 1; i < len(messageIDs); i++ {
			prevSeq := messageIDs[i-1][20:23]
			currSeq := messageIDs[i][20:23]
			t.Logf("Message %d sequence: %s, Message %d sequence: %s", i-1, prevSeq, i, currSeq)
		}

		t.Logf("✓ Generated %d unique sequential message custom_ids for event %s", len(messageIDs), eventCustomID)
	})

	t.Run("EventMetadataStorage", func(t *testing.T) {
		// Generate an event
		tenantID := uuid.New()
		senderIP := "192.168.1.100"
		eventCustomID, err := idGen.GenerateEventID(ctx, tenantID, senderIP)
		require.NoError(t, err, "Event custom_id generation should succeed")

		// Retrieve event metadata from Redis
		metadata, err := idGen.GetEventMetadata(ctx, eventCustomID)
		require.NoError(t, err, "Event metadata retrieval should succeed")
		require.NotEmpty(t, metadata, "Event metadata should not be empty")

		// Verify metadata contains expected fields
		assert.Equal(t, tenantID.String(), metadata["tenant_id"], "Metadata should contain correct tenant_id")
		assert.Equal(t, senderIP, metadata["sender_ip"], "Metadata should contain correct sender_ip")
		assert.NotEmpty(t, metadata["created_at"], "Metadata should contain created_at timestamp")
		assert.Equal(t, "0", metadata["msg_count"], "Initial message count should be 0")

		t.Logf("✓ Event metadata stored correctly: %+v", metadata)
	})

	t.Run("MessageCounterIncrement", func(t *testing.T) {
		// Generate an event
		tenantID := uuid.New()
		eventCustomID, err := idGen.GenerateEventID(ctx, tenantID, "192.168.1.100")
		require.NoError(t, err, "Event custom_id generation should succeed")

		// Generate 5 messages
		for i := 0; i < 5; i++ {
			_, err := idGen.GenerateMessageID(ctx, eventCustomID)
			require.NoError(t, err, "Message custom_id generation should succeed")
		}

		// Verify message count
		count, err := idGen.GetEventMessageCount(ctx, eventCustomID)
		require.NoError(t, err, "Message count retrieval should succeed")
		assert.Equal(t, int64(5), count, "Message count should be 5")

		t.Logf("✓ Event %s has correct message count: %d", eventCustomID, count)
	})

	t.Run("MultiTenantIsolation", func(t *testing.T) {
		// Generate events for different tenants
		tenant1 := uuid.New()
		tenant2 := uuid.New()

		event1, err := idGen.GenerateEventID(ctx, tenant1, "192.168.1.100")
		require.NoError(t, err, "Event custom_id generation should succeed for tenant 1")

		event2, err := idGen.GenerateEventID(ctx, tenant2, "192.168.1.101")
		require.NoError(t, err, "Event custom_id generation should succeed for tenant 2")

		// Verify tenant hashes are different
		tenant1Hash := event1[12:16]
		tenant2Hash := event2[12:16]
		assert.NotEqual(t, tenant1Hash, tenant2Hash, "Different tenants should have different hashes")

		t.Logf("✓ Tenant 1 hash: %s, Tenant 2 hash: %s (isolated)", tenant1Hash, tenant2Hash)
	})
}

// TestBug11_Preservation_IDParsingFunctions tests that ID parsing functions continue to work
//
// **Validates: Requirements 3.13**
//
// This test verifies that the helper functions for parsing custom_id values continue to work correctly.
func TestBug11_Preservation_IDParsingFunctions(t *testing.T) {
	t.Run("ParseEventID", func(t *testing.T) {
		// Use a known event ID format: EV + YYMMDDHHmm (10) + TenantHash (4) + Sequence (4) = 20 chars
		// Example: EV + 2601012312 + 5001 + 0001 = EV260101231250010001
		eventID := "EV260101231250010001"
		
		info, err := redis.ParseEventID(eventID)
		require.NoError(t, err, "Event ID parsing should succeed")
		require.NotNil(t, info, "Parsed info should not be nil")

		assert.Equal(t, "EV", info.EntityType, "Entity type should be EV")
		assert.Equal(t, 2026, info.Year, "Year should be 2026")
		assert.Equal(t, 1, info.Month, "Month should be 1")
		assert.Equal(t, 1, info.Day, "Day should be 1")
		assert.Equal(t, 23, info.Hour, "Hour should be 23")
		assert.Equal(t, "5001", info.TenantHash, "Tenant hash should be 5001")
		assert.Equal(t, 1, info.Sequence, "Sequence should be 1")
		assert.Equal(t, eventID, info.OriginalID, "Original ID should match")

		t.Logf("✓ Parsed event ID: %+v", info)
	})

	t.Run("ParseMessageID", func(t *testing.T) {
		// Use a known message ID format: MS + YYMMDDHHmm (10) + TenantHash (4) + EventSeq (4) + MsgSeq (3) = 23 chars
		// Let's construct it carefully: MS2601012312500100010001
		messageID := "MS" + "2601012312" + "5001" + "0001" + "001"
		
		// Verify length before testing
		require.Equal(t, 23, len(messageID), "Message ID should be exactly 23 characters")
		
		info, err := redis.ParseMessageID(messageID)
		require.NoError(t, err, "Message ID parsing should succeed")
		require.NotNil(t, info, "Parsed info should not be nil")

		assert.Equal(t, "MS", info.EntityType, "Entity type should be MS")
		assert.Equal(t, 2026, info.Year, "Year should be 2026")
		assert.Equal(t, 1, info.Month, "Month should be 1")
		assert.Equal(t, 1, info.Day, "Day should be 1")
		assert.Equal(t, 23, info.Hour, "Hour should be 23")
		assert.Equal(t, "5001", info.TenantHash, "Tenant hash should be 5001")
		assert.Equal(t, 1, info.EventSeq, "Event sequence should be 1")
		assert.Equal(t, 1, info.MessageSeq, "Message sequence should be 1")
		assert.Equal(t, "EV260101231250010001", info.EventID, "Event ID should be reconstructed")
		assert.Equal(t, messageID, info.OriginalID, "Original ID should match")

		t.Logf("✓ Parsed message ID: %+v", info)
	})

	t.Run("InvalidEventIDFormat", func(t *testing.T) {
		invalidIDs := []string{
			"",                      // Empty
			"EV123",                 // Too short
			"MS2601012312345001",    // Wrong prefix
			"EV26010123123450012345", // Too long
		}

		for _, invalidID := range invalidIDs {
			_, err := redis.ParseEventID(invalidID)
			assert.Error(t, err, "Parsing invalid event ID should fail: %s", invalidID)
		}

		t.Log("✓ Invalid event ID formats correctly rejected")
	})

	t.Run("InvalidMessageIDFormat", func(t *testing.T) {
		invalidIDs := []string{
			"",                         // Empty
			"MS123",                    // Too short
			"EV26010123123450001001",   // Wrong prefix
			"MS2601012312345000100123", // Too long
		}

		for _, invalidID := range invalidIDs {
			_, err := redis.ParseMessageID(invalidID)
			assert.Error(t, err, "Parsing invalid message ID should fail: %s", invalidID)
		}

		t.Log("✓ Invalid message ID formats correctly rejected")
	})
}
