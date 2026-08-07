package worker

import (
	"context"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// createTestDB creates a *database.DB for integration tests using pgxpool
func createTestDB(t *testing.T) *database.DB {
	t.Helper()
	ctx := context.Background()
	connStr := "postgres://postgres@localhost:5432/sms_gateway_test?sslmode=disable"
	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err, "Failed to connect to test database")

	// Verify connection
	ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	require.NoError(t, pool.Ping(ctx2), "Failed to ping test database")

	return database.NewFromPool(pool)
}

// createWorkerTestTenant inserts a tenant row so the messages/dead_letter_queue
// foreign key constraints are satisfied, and returns its ID.
func createWorkerTestTenant(t *testing.T, db *database.DB) uuid.UUID {
	t.Helper()
	tenantID := uuid.New()
	_, err := db.Pool.Exec(context.Background(), `
		INSERT INTO tenants (id, business_name, contact_name, email, status)
		VALUES ($1, 'Worker Test Tenant', 'Test Contact', 'worker@example.com', 'ACTIVE')
	`, tenantID)
	require.NoError(t, err)
	return tenantID
}

// TestDLQ_FullWorkflow tests the complete DLQ workflow
// 1. Message fails all retries
// 2. Message moved to DLQ
// 3. Message retrieved from DLQ
// 4. Message reprocessed
// 5. Old messages cleaned up
func TestDLQ_FullWorkflow(t *testing.T) {
	// Skip if no database connection
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := createTestDB(t)
	ctx := context.Background()
	var err error
	defer db.Close()

	logger := zap.NewNop()
	dlq := NewDeadLetterQueue(db, logger)

	// Create test tenant
	tenantID := createWorkerTestTenant(t, db)

	// Step 1: Create a failed message
	messageID := uuid.New()
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO messages (
			tenant_id, message_id, phone_number, message_content,
			status, cost, priority
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, tenantID, messageID, "+1234567890", "Test message", "FAILED", "0.05", "NORMAL")
	require.NoError(t, err)

	// Step 2: Move to DLQ (simulating Worker Pool behavior)
	err = dlq.MoveToDeadLetter(ctx, tenantID, messageID, "Failed after 3 attempts", 3)
	assert.NoError(t, err)

	// Verify message in DLQ
	var dlqID int64
	var dlqStatus string
	err = db.Pool.QueryRow(ctx, `
		SELECT id, original_status
		FROM dead_letter_queue
		WHERE tenant_id = $1 AND message_id = $2
	`, tenantID, messageID).Scan(&dlqID, &dlqStatus)
	require.NoError(t, err)
	assert.Equal(t, "FAILED", dlqStatus)

	// Verify message status updated to DEAD_LETTER
	var messageStatus string
	err = db.Pool.QueryRow(ctx, `
		SELECT status FROM messages WHERE message_id = $1
	`, messageID).Scan(&messageStatus)
	require.NoError(t, err)
	assert.Equal(t, "DEAD_LETTER", messageStatus)

	// Step 3: Retrieve DLQ messages
	messages, err := dlq.GetDeadLetterMessages(ctx, tenantID, 10)
	assert.NoError(t, err)
	assert.Len(t, messages, 1)
	assert.Equal(t, messageID, messages[0].MessageID)
	assert.Equal(t, 3, messages[0].RetryCount)

	// Step 4: Reprocess message
	err = dlq.ReprocessMessage(ctx, tenantID, dlqID)
	assert.NoError(t, err)

	// Verify message status back to PENDING
	err = db.Pool.QueryRow(ctx, `
		SELECT status FROM messages WHERE message_id = $1
	`, messageID).Scan(&messageStatus)
	require.NoError(t, err)
	assert.Equal(t, "PENDING", messageStatus)

	// Verify message removed from DLQ
	var dlqCount int
	err = db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM dead_letter_queue WHERE id = $1
	`, dlqID).Scan(&dlqCount)
	require.NoError(t, err)
	assert.Equal(t, 0, dlqCount)

	// Step 5: Test cleanup (add old message)
	oldMessageID := uuid.New()
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO messages (
			tenant_id, message_id, phone_number, message_content,
			status, cost, priority
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, tenantID, oldMessageID, "+9876543210", "Old message", "DEAD_LETTER", "0.05", "NORMAL")
	require.NoError(t, err)

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO dead_letter_queue (
			tenant_id, message_id, phone_number, content,
			original_status, fail_reason, retry_count, cost, moved_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW() - INTERVAL '8 days')
	`, tenantID, oldMessageID, "+9876543210", "Old message", "FAILED", "Old failure", 3, "0.05")
	require.NoError(t, err)

	// Run cleanup
	deletedCount, err := dlq.CleanupOldMessages(ctx, 7)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), deletedCount)

	// Verify old message deleted
	err = db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM dead_letter_queue WHERE message_id = $1
	`, oldMessageID).Scan(&dlqCount)
	require.NoError(t, err)
	assert.Equal(t, 0, dlqCount)

	// Cleanup
	_, _ = db.Pool.Exec(ctx, "DELETE FROM messages WHERE tenant_id = $1", tenantID)
	_, _ = db.Pool.Exec(ctx, "DELETE FROM dead_letter_queue WHERE tenant_id = $1", tenantID)
}

// TestDLQ_MultiTenantIsolation tests complete isolation between tenants
func TestDLQ_MultiTenantIsolation(t *testing.T) {
	// Skip if no database connection
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := createTestDB(t)
	ctx := context.Background()
	var err error
	defer db.Close()

	logger := zap.NewNop()
	dlq := NewDeadLetterQueue(db, logger)

	// Create two tenants
	tenant1ID := createWorkerTestTenant(t, db)
	tenant2ID := createWorkerTestTenant(t, db)

	// Create messages for both tenants
	message1ID := uuid.New()
	message2ID := uuid.New()

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO messages (
			tenant_id, message_id, phone_number, message_content,
			status, cost, priority
		) VALUES 
		($1, $2, '+1111111111', 'Tenant 1 message', 'FAILED', '0.05', 'NORMAL'),
		($3, $4, '+2222222222', 'Tenant 2 message', 'FAILED', '0.05', 'NORMAL')
	`, tenant1ID, message1ID, tenant2ID, message2ID)
	require.NoError(t, err)

	// Move both to DLQ
	err = dlq.MoveToDeadLetter(ctx, tenant1ID, message1ID, "Tenant 1 failure", 3)
	assert.NoError(t, err)

	err = dlq.MoveToDeadLetter(ctx, tenant2ID, message2ID, "Tenant 2 failure", 3)
	assert.NoError(t, err)

	// Verify tenant 1 can only see their messages
	messages1, err := dlq.GetDeadLetterMessages(ctx, tenant1ID, 10)
	assert.NoError(t, err)
	assert.Len(t, messages1, 1)
	assert.Equal(t, tenant1ID, messages1[0].TenantID)
	assert.Equal(t, message1ID, messages1[0].MessageID)

	// Verify tenant 2 can only see their messages
	messages2, err := dlq.GetDeadLetterMessages(ctx, tenant2ID, 10)
	assert.NoError(t, err)
	assert.Len(t, messages2, 1)
	assert.Equal(t, tenant2ID, messages2[0].TenantID)
	assert.Equal(t, message2ID, messages2[0].MessageID)

	// Reprocess all for tenant 1
	count, err := dlq.ReprocessAllMessages(ctx, tenant1ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Verify tenant 1's messages reprocessed
	messages1, err = dlq.GetDeadLetterMessages(ctx, tenant1ID, 10)
	assert.NoError(t, err)
	assert.Len(t, messages1, 0)

	// Verify tenant 2's messages still in DLQ
	messages2, err = dlq.GetDeadLetterMessages(ctx, tenant2ID, 10)
	assert.NoError(t, err)
	assert.Len(t, messages2, 1)

	// Cleanup
	_, _ = db.Pool.Exec(ctx, "DELETE FROM dead_letter_queue WHERE tenant_id IN ($1, $2)", tenant1ID, tenant2ID)
	_, _ = db.Pool.Exec(ctx, "DELETE FROM messages WHERE tenant_id IN ($1, $2)", tenant1ID, tenant2ID)
}

// TestDLQ_ConcurrentOperations tests concurrent DLQ operations
func TestDLQ_ConcurrentOperations(t *testing.T) {
	// Skip if no database connection
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := createTestDB(t)
	ctx := context.Background()
	var err error
	defer db.Close()

	logger := zap.NewNop()
	dlq := NewDeadLetterQueue(db, logger)

	// Create test tenant
	tenantID := createWorkerTestTenant(t, db)

	// Create multiple messages
	messageCount := 10
	messageIDs := make([]uuid.UUID, messageCount)

	for i := 0; i < messageCount; i++ {
		messageIDs[i] = uuid.New()
		_, err = db.Pool.Exec(ctx, `
			INSERT INTO messages (
				tenant_id, message_id, phone_number, message_content,
				status, cost, priority
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, tenantID, messageIDs[i], "+1234567890", "Test message", "FAILED", "0.05", "NORMAL")
		require.NoError(t, err)
	}

	// Move all to DLQ concurrently
	done := make(chan bool, messageCount)
	for i := 0; i < messageCount; i++ {
		go func(msgID uuid.UUID) {
			err := dlq.MoveToDeadLetter(ctx, tenantID, msgID, "Concurrent failure", 3)
			assert.NoError(t, err)
			done <- true
		}(messageIDs[i])
	}

	// Wait for all to complete
	for i := 0; i < messageCount; i++ {
		<-done
	}

	// Verify all messages in DLQ
	messages, err := dlq.GetDeadLetterMessages(ctx, tenantID, 100)
	assert.NoError(t, err)
	assert.Len(t, messages, messageCount)

	// Cleanup
	_, _ = db.Pool.Exec(ctx, "DELETE FROM dead_letter_queue WHERE tenant_id = $1", tenantID)
	_, _ = db.Pool.Exec(ctx, "DELETE FROM messages WHERE tenant_id = $1", tenantID)
}

// TestDLQ_RetryCountTracking tests accurate retry count tracking
func TestDLQ_RetryCountTracking(t *testing.T) {
	// Skip if no database connection
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := createTestDB(t)
	ctx := context.Background()
	var err error
	defer db.Close()

	logger := zap.NewNop()
	dlq := NewDeadLetterQueue(db, logger)

	// Create test tenant
	tenantID := createWorkerTestTenant(t, db)

	// Test different retry counts
	testCases := []struct {
		retryCount int
		reason     string
	}{
		{1, "Failed after 1 attempt"},
		{2, "Failed after 2 attempts"},
		{3, "Failed after 3 attempts"},
	}

	for _, tc := range testCases {
		messageID := uuid.New()
		
		// Insert message
		_, err = db.Pool.Exec(ctx, `
			INSERT INTO messages (
				tenant_id, message_id, phone_number, message_content,
				status, cost, priority
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, tenantID, messageID, "+1234567890", "Test message", "FAILED", "0.05", "NORMAL")
		require.NoError(t, err)

		// Move to DLQ with specific retry count
		err = dlq.MoveToDeadLetter(ctx, tenantID, messageID, tc.reason, tc.retryCount)
		assert.NoError(t, err)

		// Verify retry count
		var retryCount int
		var reason string
		err = db.Pool.QueryRow(ctx, `
			SELECT retry_count, fail_reason
			FROM dead_letter_queue
			WHERE message_id = $1
		`, messageID).Scan(&retryCount, &reason)
		require.NoError(t, err)
		assert.Equal(t, tc.retryCount, retryCount)
		assert.Equal(t, tc.reason, reason)
	}

	// Cleanup
	_, _ = db.Pool.Exec(ctx, "DELETE FROM dead_letter_queue WHERE tenant_id = $1", tenantID)
	_, _ = db.Pool.Exec(ctx, "DELETE FROM messages WHERE tenant_id = $1", tenantID)
}
