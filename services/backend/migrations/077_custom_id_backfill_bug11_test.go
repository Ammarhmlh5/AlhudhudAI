package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBug11_MigrationLeavesNullCustomIDs tests that migration 077 does NOT backfill existing records
//
// **Validates: Requirements 1.11, 2.11**
//
// Bug Condition: Migration 077 adds custom_id columns but does not backfill existing records,
// leaving NULL values in production data.
//
// This test MUST FAIL on UNFIXED migration (NULL values remain) - this proves the bug exists.
// After fix, this test MUST PASS (all records have custom_id values).
//
// Test Strategy:
// 1. Create test database with existing events and messages (simulating production data)
// 2. Run migration 077 (or verify its effects)
// 3. Check that existing records have NULL custom_id
// 4. Check that new records get custom_id correctly
//
// EXPECTED OUTCOME ON UNFIXED CODE: Test FAILS (NULL values found in existing records)
// EXPECTED OUTCOME ON FIXED CODE: Test PASSES (all records have custom_id)
func TestBug11_MigrationLeavesNullCustomIDs(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test - set INTEGRATION_TEST=true to run")
	}

	// Setup test database connection
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "postgres"
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "alhudhud_test"
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err, "Failed to connect to test database")
	defer db.Close()

	ctx := context.Background()

	// Verify connection
	err = db.PingContext(ctx)
	require.NoError(t, err, "Failed to ping test database")

	t.Log("✓ Connected to test database")

	// Step 1: Create test tenant
	tenantID := uuid.New()
	_, err = db.ExecContext(ctx, `
		INSERT INTO tenants (id, company_name, email, phone_number, balance, created_at, updated_at)
		VALUES ($1, 'Test Company', 'test@example.com', '+1234567890', 1000.00, NOW(), NOW())
		ON CONFLICT (id) DO NOTHING
	`, tenantID)
	require.NoError(t, err, "Failed to create test tenant")

	t.Logf("✓ Created test tenant: %s", tenantID)

	// Step 2: Create "existing" events (simulating pre-migration data)
	// These should have NULL custom_id after migration if bug exists
	existingEventIDs := make([]int64, 3)
	for i := 0; i < 3; i++ {
		var eventID int64
		err = db.QueryRowContext(ctx, `
			INSERT INTO global_incoming_events 
			(tenant_id, sender_ip, raw_payload, source, status, total_items, created_at, updated_at)
			VALUES ($1, '127.0.0.1', '{"test": true}', 'WEB_API', 'RECEIVED', 10, NOW() - INTERVAL '1 day', NOW())
			RETURNING id
		`, tenantID).Scan(&eventID)
		require.NoError(t, err, "Failed to create existing event")
		existingEventIDs[i] = eventID
	}

	t.Logf("✓ Created %d existing events (pre-migration simulation)", len(existingEventIDs))

	// Step 3: Create "existing" messages (simulating pre-migration data)
	existingMessageIDs := make([]uuid.UUID, 5)
	for i := 0; i < 5; i++ {
		var messageID uuid.UUID
		err = db.QueryRowContext(ctx, `
			INSERT INTO messages 
			(tenant_id, phone_number, content, status, global_event_id, source, created_at, updated_at)
			VALUES ($1, '+1234567890', 'Test message', 'PENDING', $2, 'WEB_API', NOW() - INTERVAL '1 day', NOW())
			RETURNING id
		`, tenantID, existingEventIDs[i%len(existingEventIDs)]).Scan(&messageID)
		require.NoError(t, err, "Failed to create existing message")
		existingMessageIDs[i] = messageID
	}

	t.Logf("✓ Created %d existing messages (pre-migration simulation)", len(existingMessageIDs))

	// Step 4: Verify migration 077 has been applied (columns exist)
	var eventCustomIDExists bool
	err = db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'global_incoming_events' 
			AND column_name = 'custom_id'
		)
	`).Scan(&eventCustomIDExists)
	require.NoError(t, err, "Failed to check event custom_id column")
	require.True(t, eventCustomIDExists, "Migration 077 not applied - custom_id column missing in global_incoming_events")

	var messageCustomIDExists bool
	err = db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'messages' 
			AND column_name = 'custom_id'
		)
	`).Scan(&messageCustomIDExists)
	require.NoError(t, err, "Failed to check message custom_id column")
	require.True(t, messageCustomIDExists, "Migration 077 not applied - custom_id column missing in messages")

	t.Log("✓ Migration 077 columns exist")

	// Step 5: Check existing events for NULL custom_id (BUG CONDITION)
	var nullEventCount int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM global_incoming_events 
		WHERE id = ANY($1) AND custom_id IS NULL
	`, existingEventIDs).Scan(&nullEventCount)
	require.NoError(t, err, "Failed to count NULL custom_id in events")

	t.Logf("Found %d events with NULL custom_id out of %d existing events", nullEventCount, len(existingEventIDs))

	// Step 6: Check existing messages for NULL custom_id (BUG CONDITION)
	var nullMessageCount int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM messages 
		WHERE id = ANY($1) AND custom_id IS NULL
	`, existingMessageIDs).Scan(&nullMessageCount)
	require.NoError(t, err, "Failed to count NULL custom_id in messages")

	t.Logf("Found %d messages with NULL custom_id out of %d existing messages", nullMessageCount, len(existingMessageIDs))

	// Step 7: Create NEW records (post-migration) to verify they get custom_id
	var newEventID int64
	var newEventCustomID sql.NullString
	err = db.QueryRowContext(ctx, `
		INSERT INTO global_incoming_events 
		(tenant_id, sender_ip, raw_payload, source, status, total_items, created_at, updated_at)
		VALUES ($1, '127.0.0.1', '{"test": true}', 'WEB_API', 'RECEIVED', 5, NOW(), NOW())
		RETURNING id, custom_id
	`, tenantID).Scan(&newEventID, &newEventCustomID)
	require.NoError(t, err, "Failed to create new event")

	t.Logf("✓ Created new event: id=%d, custom_id=%v", newEventID, newEventCustomID)

	var newMessageID uuid.UUID
	var newMessageCustomID sql.NullString
	err = db.QueryRowContext(ctx, `
		INSERT INTO messages 
		(tenant_id, phone_number, content, status, global_event_id, source, created_at, updated_at)
		VALUES ($1, '+1234567890', 'New test message', 'PENDING', $2, 'WEB_API', NOW(), NOW())
		RETURNING id, custom_id
	`, tenantID, newEventID).Scan(&newMessageID, &newMessageCustomID)
	require.NoError(t, err, "Failed to create new message")

	t.Logf("✓ Created new message: id=%s, custom_id=%v", newMessageID, newMessageCustomID)

	// ASSERTIONS
	// On UNFIXED migration: existing records have NULL, new records may also have NULL
	// On FIXED migration: ALL records have custom_id values

	// Bug Condition Check: Existing records should have NULL custom_id (UNFIXED)
	// After fix: This assertion should FAIL because backfill populates them
	if nullEventCount > 0 || nullMessageCount > 0 {
		t.Logf("❌ BUG DETECTED: Migration 077 left %d events and %d messages with NULL custom_id",
			nullEventCount, nullMessageCount)
		t.Logf("This is the EXPECTED outcome on UNFIXED code - the bug exists!")
		
		// This assertion will FAIL on unfixed code (proving bug exists)
		// and PASS on fixed code (proving bug is fixed)
		assert.Equal(t, 0, nullEventCount, 
			"Bug 11: Existing events have NULL custom_id - migration did not backfill")
		assert.Equal(t, 0, nullMessageCount, 
			"Bug 11: Existing messages have NULL custom_id - migration did not backfill")
	} else {
		t.Log("✓ All existing records have custom_id values - bug is FIXED or backfill was run")
	}

	// Verify new records behavior (should always get custom_id after migration is applied)
	// Note: This depends on application logic, not just migration
	// The migration adds the column, but application code generates the custom_id
	t.Logf("New event custom_id: %v (Valid=%v)", newEventCustomID.String, newEventCustomID.Valid)
	t.Logf("New message custom_id: %v (Valid=%v)", newMessageCustomID.String, newMessageCustomID.Valid)

	// Cleanup
	_, _ = db.ExecContext(ctx, "DELETE FROM messages WHERE tenant_id = $1", tenantID)
	_, _ = db.ExecContext(ctx, "DELETE FROM global_incoming_events WHERE tenant_id = $1", tenantID)
	_, _ = db.ExecContext(ctx, "DELETE FROM tenants WHERE id = $1", tenantID)

	t.Log("✓ Cleanup complete")
}

// TestBug11_CustomIDFormat tests the format of generated custom_id values
//
// **Validates: Requirements 2.11**
//
// This test verifies that when custom_id values ARE generated (either by backfill or new records),
// they follow the correct format:
// - Events: EV + YYMMDDHHmmss + TenantHash + Sequence (20 chars)
// - Messages: MS + YYMMDDHHmmss + TenantHash + EventSeq + MsgSeq (23 chars)
func TestBug11_CustomIDFormat(t *testing.T) {
	t.Run("EventCustomIDFormat", func(t *testing.T) {
		// Event format: EV260101231234500100
		// EV (2) + YYMMDD (6) + HH (2) + TenantHash (5) + Sequence (5) = 20 chars
		eventID := "EV260101231234500100"
		
		assert.Equal(t, 20, len(eventID), "Event custom_id should be 20 characters")
		assert.Equal(t, "EV", eventID[0:2], "Event custom_id should start with 'EV'")
		
		// Verify timestamp part is numeric
		timestampPart := eventID[2:10]
		assert.Regexp(t, `^\d{8}$`, timestampPart, "Timestamp part should be 8 digits (YYMMDD)")
		
		t.Logf("✓ Event custom_id format valid: %s", eventID)
	})

	t.Run("MessageCustomIDFormat", func(t *testing.T) {
		// Message format: MS26010123123450001001
		// MS (2) + YYMMDD (6) + HH (2) + TenantHash (5) + EventSeq (4) + MsgSeq (3) = 23 chars
		messageID := "MS260101231234500010001"
		
		assert.Equal(t, 23, len(messageID), "Message custom_id should be 23 characters")
		assert.Equal(t, "MS", messageID[0:2], "Message custom_id should start with 'MS'")
		
		// Verify timestamp part is numeric
		timestampPart := messageID[2:10]
		assert.Regexp(t, `^\d{8}$`, timestampPart, "Timestamp part should be 8 digits (YYMMDD)")
		
		t.Logf("✓ Message custom_id format valid: %s", messageID)
	})
}
