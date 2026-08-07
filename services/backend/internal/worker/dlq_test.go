package worker

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/config"
	"github.com/al-hudhud/backend/internal/database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// newTestDatabase opens a connection to the sms_gateway_test database using the
// configured pool limits and the PGPASSWORD env var (falling back to "postgres").
func newTestDatabase(t *testing.T) *database.DB {
	t.Helper()
	password := os.Getenv("PGPASSWORD")
	if password == "" {
		password = "postgres"
	}
	db, err := database.NewDatabase(&config.Config{
		Database: config.DatabaseConfig{
			Host:            "localhost",
			Port:            "5432",
			User:            "postgres",
			Password:        password,
			Name:            "sms_gateway_test",
			SSLMode:         "disable",
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
		},
	}, zap.NewNop())
	require.NoError(t, err)
	return db
}

// TestDeadLetterQueue_MoveToDeadLetter tests moving a message to DLQ
func TestDeadLetterQueue_MoveToDeadLetter(t *testing.T) {
	// Skip if no database connection
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	db := newTestDatabase(t)
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
	`, tenantID, messageID, "+1234567890", "Test message", "FAILED", "0.05", "NORMAL")
	require.NoError(t, err)

	// Move to DLQ
	err = dlq.MoveToDeadLetter(ctx, tenantID, messageID, "Test failure reason", 3)
	assert.NoError(t, err)

	// Verify message in DLQ
	var dlqID int64
	var dlqReason string
	var dlqRetryCount int
	err = db.Pool.QueryRow(ctx, `
		SELECT id, fail_reason, retry_count
		FROM dead_letter_queue
		WHERE tenant_id = $1 AND message_id = $2
	`, tenantID, messageID).Scan(&dlqID, &dlqReason, &dlqRetryCount)
	require.NoError(t, err)

	assert.Greater(t, dlqID, int64(0))
	assert.Equal(t, "Test failure reason", dlqReason)
	assert.Equal(t, 3, dlqRetryCount)

	// Verify message status updated
	var messageStatus string
	err = db.Pool.QueryRow(ctx, `
		SELECT status
		FROM messages
		WHERE tenant_id = $1 AND message_id = $2
	`, tenantID, messageID).Scan(&messageStatus)
	require.NoError(t, err)
	assert.Equal(t, "DEAD_LETTER", messageStatus)

	// Cleanup
	_, _ = db.Pool.Exec(ctx, "DELETE FROM dead_letter_queue WHERE tenant_id = $1", tenantID)
	_, _ = db.Pool.Exec(ctx, "DELETE FROM messages WHERE tenant_id = $1", tenantID)
}

// TestDeadLetterQueue_GetDeadLetterMessages tests retrieving DLQ messages
func TestDeadLetterQueue_GetDeadLetterMessages(t *testing.T) {
	// Skip if no database connection
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	db := newTestDatabase(t)
	var err error
	defer db.Close()

	logger := zap.NewNop()
	dlq := NewDeadLetterQueue(db, logger)

	// Create test tenant
	tenantID := createWorkerTestTenant(t, db)

	// Insert test DLQ messages
	messageID1 := uuid.New()
	messageID2 := uuid.New()

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO dead_letter_queue (
			tenant_id, message_id, phone_number, content,
			original_status, fail_reason, retry_count, cost
		) VALUES 
		($1, $2, '+1234567890', 'Test message 1', 'FAILED', 'Reason 1', 3, '0.05'),
		($1, $3, '+0987654321', 'Test message 2', 'FAILED', 'Reason 2', 3, '0.05')
	`, tenantID, messageID1, messageID2)
	require.NoError(t, err)

	// Get DLQ messages
	messages, err := dlq.GetDeadLetterMessages(ctx, tenantID, 10)
	assert.NoError(t, err)
	assert.Len(t, messages, 2)

	// Verify messages
	assert.Equal(t, tenantID, messages[0].TenantID)
	assert.Contains(t, []uuid.UUID{messageID1, messageID2}, messages[0].MessageID)
	assert.Equal(t, 3, messages[0].RetryCount)

	// Cleanup
	_, _ = db.Pool.Exec(ctx, "DELETE FROM dead_letter_queue WHERE tenant_id = $1", tenantID)
}

// TestDeadLetterQueue_CleanupOldMessages tests cleanup of old DLQ messages
func TestDeadLetterQueue_CleanupOldMessages(t *testing.T) {
	// Skip if no database connection
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	db := newTestDatabase(t)
	var err error
	defer db.Close()

	logger := zap.NewNop()
	dlq := NewDeadLetterQueue(db, logger)

	// Create test tenant
	tenantID := createWorkerTestTenant(t, db)

	// Insert old message (8 days ago)
	oldMessageID := uuid.New()
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO dead_letter_queue (
			tenant_id, message_id, phone_number, content,
			original_status, fail_reason, retry_count, cost, moved_at
		) VALUES ($1, $2, '+1234567890', 'Old message', 'FAILED', 'Old reason', 3, '0.05', NOW() - INTERVAL '8 days')
	`, tenantID, oldMessageID)
	require.NoError(t, err)

	// Insert recent message (1 day ago)
	recentMessageID := uuid.New()
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO dead_letter_queue (
			tenant_id, message_id, phone_number, content,
			original_status, fail_reason, retry_count, cost, moved_at
		) VALUES ($1, $2, '+0987654321', 'Recent message', 'FAILED', 'Recent reason', 3, '0.05', NOW() - INTERVAL '1 day')
	`, tenantID, recentMessageID)
	require.NoError(t, err)

	// Cleanup messages older than 7 days
	rowsAffected, err := dlq.CleanupOldMessages(ctx, 7)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)

	// Verify old message deleted
	var count int
	err = db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM dead_letter_queue WHERE message_id = $1
	`, oldMessageID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Verify recent message still exists
	err = db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM dead_letter_queue WHERE message_id = $1
	`, recentMessageID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Cleanup
	_, _ = db.Pool.Exec(ctx, "DELETE FROM dead_letter_queue WHERE tenant_id = $1", tenantID)
}

// TestDeadLetterQueue_TenantIsolation tests tenant isolation in DLQ
func TestDeadLetterQueue_TenantIsolation(t *testing.T) {
	// Skip if no database connection
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	db := newTestDatabase(t)
	var err error
	defer db.Close()

	logger := zap.NewNop()
	dlq := NewDeadLetterQueue(db, logger)

	// Create two test tenants
	tenant1ID := createWorkerTestTenant(t, db)
	tenant2ID := createWorkerTestTenant(t, db)

	// Insert DLQ messages for both tenants
	message1ID := uuid.New()
	message2ID := uuid.New()

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO dead_letter_queue (
			tenant_id, message_id, phone_number, content,
			original_status, fail_reason, retry_count, cost
		) VALUES 
		($1, $2, '+1111111111', 'Tenant 1 message', 'FAILED', 'Reason 1', 3, '0.05'),
		($3, $4, '+2222222222', 'Tenant 2 message', 'FAILED', 'Reason 2', 3, '0.05')
	`, tenant1ID, message1ID, tenant2ID, message2ID)
	require.NoError(t, err)

	// Get DLQ messages for tenant 1
	messages1, err := dlq.GetDeadLetterMessages(ctx, tenant1ID, 10)
	assert.NoError(t, err)
	assert.Len(t, messages1, 1)
	assert.Equal(t, tenant1ID, messages1[0].TenantID)
	assert.Equal(t, message1ID, messages1[0].MessageID)

	// Get DLQ messages for tenant 2
	messages2, err := dlq.GetDeadLetterMessages(ctx, tenant2ID, 10)
	assert.NoError(t, err)
	assert.Len(t, messages2, 1)
	assert.Equal(t, tenant2ID, messages2[0].TenantID)
	assert.Equal(t, message2ID, messages2[0].MessageID)

	// Verify tenant 1 cannot see tenant 2's messages
	assert.NotEqual(t, messages1[0].MessageID, messages2[0].MessageID)

	// Cleanup
	_, _ = db.Pool.Exec(ctx, "DELETE FROM dead_letter_queue WHERE tenant_id IN ($1, $2)", tenant1ID, tenant2ID)
}

// TestDeadLetterQueue_RetryCountTracking tests retry count tracking
func TestDeadLetterQueue_RetryCountTracking(t *testing.T) {
	// Skip if no database connection
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	db := newTestDatabase(t)
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
	`, tenantID, messageID, "+1234567890", "Test message", "FAILED", "0.05", "NORMAL")
	require.NoError(t, err)

	// Move to DLQ with retry count = 3
	err = dlq.MoveToDeadLetter(ctx, tenantID, messageID, "Failed after 3 attempts", 3)
	assert.NoError(t, err)

	// Verify retry count
	messages, err := dlq.GetDeadLetterMessages(ctx, tenantID, 10)
	assert.NoError(t, err)
	require.Len(t, messages, 1)
	assert.Equal(t, 3, messages[0].RetryCount)
	assert.Equal(t, "Failed after 3 attempts", messages[0].FailReason)

	// Cleanup
	_, _ = db.Pool.Exec(ctx, "DELETE FROM dead_letter_queue WHERE tenant_id = $1", tenantID)
	_, _ = db.Pool.Exec(ctx, "DELETE FROM messages WHERE tenant_id = $1", tenantID)
}
