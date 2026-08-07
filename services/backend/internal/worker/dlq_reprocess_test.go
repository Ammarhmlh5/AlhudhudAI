package worker

import (
	"context"
	"testing"

	"github.com/al-hudhud/backend/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// createDLQReprocessTestDB creates a *database.DB for testing
func createDLQReprocessTestDB(t *testing.T) *database.DB {
	t.Helper()
	ctx := context.Background()
	connStr := "postgres://postgres@localhost:5432/sms_gateway_test?sslmode=disable"
	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	return database.NewFromPool(pool)
}

// TestDeadLetterQueue_ReprocessMessage tests reprocessing a single message
func TestDeadLetterQueue_ReprocessMessage(t *testing.T) {
	// Skip if no database connection
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := createDLQReprocessTestDB(t)
	ctx := context.Background()
	var err error
	defer db.Close()

	logger := zap.NewNop()
	dlq := NewDeadLetterQueue(db, logger)

	// Create test tenant and message
	tenantID := createWorkerTestTenant(t, db)
	messageID := uuid.New()

	// Insert test message
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO messages (
			tenant_id, message_id, phone_number, message_content,
			status, cost, priority
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, tenantID, messageID, "+1234567890", "Test message", "DEAD_LETTER", "0.05", "NORMAL")
	require.NoError(t, err)

	// Insert into DLQ
	var dlqID int64
	err = db.Pool.QueryRow(ctx, `
		INSERT INTO dead_letter_queue (
			tenant_id, message_id, phone_number, content,
			original_status, fail_reason, retry_count, cost
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`, tenantID, messageID, "+1234567890", "Test message", "FAILED", "Test failure", 3, "0.05").Scan(&dlqID)
	require.NoError(t, err)

	// Reprocess message
	err = dlq.ReprocessMessage(ctx, tenantID, dlqID)
	assert.NoError(t, err)

	// Verify message status updated to PENDING
	var messageStatus string
	err = db.Pool.QueryRow(ctx, `
		SELECT status
		FROM messages
		WHERE tenant_id = $1 AND message_id = $2
	`, tenantID, messageID).Scan(&messageStatus)
	require.NoError(t, err)
	assert.Equal(t, "PENDING", messageStatus)

	// Verify message removed from DLQ
	var dlqCount int
	err = db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM dead_letter_queue WHERE id = $1
	`, dlqID).Scan(&dlqCount)
	require.NoError(t, err)
	assert.Equal(t, 0, dlqCount)

	// Cleanup
	_, _ = db.Pool.Exec(ctx, "DELETE FROM messages WHERE tenant_id = $1", tenantID)
}

// TestDeadLetterQueue_ReprocessAllMessages tests reprocessing all messages
func TestDeadLetterQueue_ReprocessAllMessages(t *testing.T) {
	// Skip if no database connection
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := createDLQReprocessTestDB(t)
	ctx := context.Background()
	var err error
	defer db.Close()

	logger := zap.NewNop()
	dlq := NewDeadLetterQueue(db, logger)

	// Create test tenant
	tenantID := createWorkerTestTenant(t, db)

	// Insert test messages
	messageID1 := uuid.New()
	messageID2 := uuid.New()

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO messages (
			tenant_id, message_id, phone_number, message_content,
			status, cost, priority
		) VALUES 
		($1, $2, '+1111111111', 'Test message 1', 'DEAD_LETTER', '0.05', 'NORMAL'),
		($1, $3, '+2222222222', 'Test message 2', 'DEAD_LETTER', '0.05', 'NORMAL')
	`, tenantID, messageID1, messageID2)
	require.NoError(t, err)

	// Insert into DLQ
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO dead_letter_queue (
			tenant_id, message_id, phone_number, content,
			original_status, fail_reason, retry_count, cost
		) VALUES 
		($1, $2, '+1111111111', 'Test message 1', 'FAILED', 'Reason 1', 3, '0.05'),
		($1, $3, '+2222222222', 'Test message 2', 'FAILED', 'Reason 2', 3, '0.05')
	`, tenantID, messageID1, messageID2)
	require.NoError(t, err)

	// Reprocess all messages
	count, err := dlq.ReprocessAllMessages(ctx, tenantID)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// Verify all messages status updated to PENDING
	var pendingCount int
	err = db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM messages 
		WHERE tenant_id = $1 AND status = 'PENDING'
	`, tenantID).Scan(&pendingCount)
	require.NoError(t, err)
	assert.Equal(t, 2, pendingCount)

	// Verify all messages removed from DLQ
	var dlqCount int
	err = db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM dead_letter_queue WHERE tenant_id = $1
	`, tenantID).Scan(&dlqCount)
	require.NoError(t, err)
	assert.Equal(t, 0, dlqCount)

	// Cleanup
	_, _ = db.Pool.Exec(ctx, "DELETE FROM messages WHERE tenant_id = $1", tenantID)
}

// TestDeadLetterQueue_ReprocessTenantIsolation tests tenant isolation in reprocessing
func TestDeadLetterQueue_ReprocessTenantIsolation(t *testing.T) {
	// Skip if no database connection
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := createDLQReprocessTestDB(t)
	ctx := context.Background()
	var err error
	defer db.Close()

	logger := zap.NewNop()
	dlq := NewDeadLetterQueue(db, logger)

	// Create two test tenants
	tenant1ID := createWorkerTestTenant(t, db)
	tenant2ID := createWorkerTestTenant(t, db)

	// Insert test messages for both tenants
	message1ID := uuid.New()
	message2ID := uuid.New()

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO messages (
			tenant_id, message_id, phone_number, message_content,
			status, cost, priority
		) VALUES 
		($1, $2, '+1111111111', 'Tenant 1 message', 'DEAD_LETTER', '0.05', 'NORMAL'),
		($3, $4, '+2222222222', 'Tenant 2 message', 'DEAD_LETTER', '0.05', 'NORMAL')
	`, tenant1ID, message1ID, tenant2ID, message2ID)
	require.NoError(t, err)

	// Insert into DLQ
	var dlq1ID, dlq2ID int64
	err = db.Pool.QueryRow(ctx, `
		INSERT INTO dead_letter_queue (
			tenant_id, message_id, phone_number, content,
			original_status, fail_reason, retry_count, cost
		) VALUES ($1, $2, '+1111111111', 'Tenant 1 message', 'FAILED', 'Reason 1', 3, '0.05')
		RETURNING id
	`, tenant1ID, message1ID).Scan(&dlq1ID)
	require.NoError(t, err)

	err = db.Pool.QueryRow(ctx, `
		INSERT INTO dead_letter_queue (
			tenant_id, message_id, phone_number, content,
			original_status, fail_reason, retry_count, cost
		) VALUES ($1, $2, '+2222222222', 'Tenant 2 message', 'FAILED', 'Reason 2', 3, '0.05')
		RETURNING id
	`, tenant2ID, message2ID).Scan(&dlq2ID)
	require.NoError(t, err)

	// Reprocess tenant 1's message
	err = dlq.ReprocessMessage(ctx, tenant1ID, dlq1ID)
	assert.NoError(t, err)

	// Verify tenant 1's message reprocessed
	var status1 string
	err = db.Pool.QueryRow(ctx, `
		SELECT status FROM messages WHERE message_id = $1
	`, message1ID).Scan(&status1)
	require.NoError(t, err)
	assert.Equal(t, "PENDING", status1)

	// Verify tenant 2's message NOT reprocessed
	var status2 string
	err = db.Pool.QueryRow(ctx, `
		SELECT status FROM messages WHERE message_id = $1
	`, message2ID).Scan(&status2)
	require.NoError(t, err)
	assert.Equal(t, "DEAD_LETTER", status2)

	// Verify tenant 1's DLQ entry removed
	var dlq1Count int
	err = db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM dead_letter_queue WHERE id = $1
	`, dlq1ID).Scan(&dlq1Count)
	require.NoError(t, err)
	assert.Equal(t, 0, dlq1Count)

	// Verify tenant 2's DLQ entry still exists
	var dlq2Count int
	err = db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM dead_letter_queue WHERE id = $1
	`, dlq2ID).Scan(&dlq2Count)
	require.NoError(t, err)
	assert.Equal(t, 1, dlq2Count)

	// Cleanup
	_, _ = db.Pool.Exec(ctx, "DELETE FROM dead_letter_queue WHERE tenant_id IN ($1, $2)", tenant1ID, tenant2ID)
	_, _ = db.Pool.Exec(ctx, "DELETE FROM messages WHERE tenant_id IN ($1, $2)", tenant1ID, tenant2ID)
}

// TestDeadLetterQueue_ReprocessNonExistent tests reprocessing non-existent message
func TestDeadLetterQueue_ReprocessNonExistent(t *testing.T) {
	// Skip if no database connection
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	db := createDLQReprocessTestDB(t)
	defer db.Close()

	logger := zap.NewNop()
	dlq := NewDeadLetterQueue(db, logger)

	// Try to reprocess non-existent message
	tenantID := createWorkerTestTenant(t, db)
	err := dlq.ReprocessMessage(ctx, tenantID, 99999)
	assert.Error(t, err)
}
