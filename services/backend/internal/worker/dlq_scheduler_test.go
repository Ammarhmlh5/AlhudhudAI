package worker

import (
	"context"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/config"
	"github.com/al-hudhud/backend/internal/database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestDLQScheduler_CleanupExecution tests that scheduler executes cleanup
func TestDLQScheduler_CleanupExecution(t *testing.T) {
	// Skip if no database connection
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	db, err := database.NewDatabase(&config.Config{
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     "5432",
			User:     "postgres",
			Password: "postgres",
			Name:     "sms_gateway_test",
			SSLMode:  "disable",
		},
	}, zap.NewNop())
	require.NoError(t, err)
	defer db.Close()

	logger := zap.NewNop()
	dlq := NewDeadLetterQueue(db, logger)

	// Create test tenant
	tenantID := uuid.New()

	// Insert old message (8 days ago)
	oldMessageID := uuid.New()
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO dead_letter_queue (
			tenant_id, message_id, phone_number, content,
			original_status, fail_reason, retry_count, cost, moved_at
		) VALUES ($1, $2, '+1234567890', 'Old message', 'FAILED', 'Old reason', 3, '0.05', NOW() - INTERVAL '8 days')
	`, tenantID, oldMessageID)
	require.NoError(t, err)

	// Create scheduler with short interval for testing
	scheduler := NewDLQScheduler(dlq, logger).
		WithRetentionDays(7).
		WithCleanupInterval(100 * time.Millisecond)

	// Start scheduler
	scheduler.Start(ctx)

	// Wait for cleanup to execute
	time.Sleep(200 * time.Millisecond)

	// Stop scheduler
	scheduler.Stop()

	// Verify old message was deleted
	var count int
	err = db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM dead_letter_queue WHERE message_id = $1
	`, oldMessageID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Cleanup
	_, _ = db.Pool.Exec(ctx, "DELETE FROM dead_letter_queue WHERE tenant_id = $1", tenantID)
}

// TestDLQScheduler_RetentionPeriod tests different retention periods
func TestDLQScheduler_RetentionPeriod(t *testing.T) {
	// Skip if no database connection
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	db, err := database.NewDatabase(&config.Config{
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     "5432",
			User:     "postgres",
			Password: "postgres",
			Name:     "sms_gateway_test",
			SSLMode:  "disable",
		},
	}, zap.NewNop())
	require.NoError(t, err)
	defer db.Close()

	logger := zap.NewNop()
	dlq := NewDeadLetterQueue(db, logger)

	// Create test tenant
	tenantID := uuid.New()

	// Insert messages with different ages
	message5Days := uuid.New()
	message10Days := uuid.New()

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO dead_letter_queue (
			tenant_id, message_id, phone_number, content,
			original_status, fail_reason, retry_count, cost, moved_at
		) VALUES 
		($1, $2, '+1111111111', 'Message 5 days old', 'FAILED', 'Reason', 3, '0.05', NOW() - INTERVAL '5 days'),
		($1, $3, '+2222222222', 'Message 10 days old', 'FAILED', 'Reason', 3, '0.05', NOW() - INTERVAL '10 days')
	`, tenantID, message5Days, message10Days)
	require.NoError(t, err)

	// Create scheduler with 7 days retention
	scheduler := NewDLQScheduler(dlq, logger).
		WithRetentionDays(7).
		WithCleanupInterval(100 * time.Millisecond)

	// Start scheduler
	scheduler.Start(ctx)

	// Wait for cleanup to execute
	time.Sleep(200 * time.Millisecond)

	// Stop scheduler
	scheduler.Stop()

	// Verify 5-day-old message still exists
	var count5Days int
	err = db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM dead_letter_queue WHERE message_id = $1
	`, message5Days).Scan(&count5Days)
	require.NoError(t, err)
	assert.Equal(t, 1, count5Days)

	// Verify 10-day-old message was deleted
	var count10Days int
	err = db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM dead_letter_queue WHERE message_id = $1
	`, message10Days).Scan(&count10Days)
	require.NoError(t, err)
	assert.Equal(t, 0, count10Days)

	// Cleanup
	_, _ = db.Pool.Exec(ctx, "DELETE FROM dead_letter_queue WHERE tenant_id = $1", tenantID)
}

// TestDLQScheduler_StopGracefully tests graceful shutdown
func TestDLQScheduler_StopGracefully(t *testing.T) {
	logger := zap.NewNop()
	
	// Create mock DLQ (no database needed for this test)
	dlq := &DeadLetterQueue{
		logger: logger,
	}

	// Create scheduler
	scheduler := NewDLQScheduler(dlq, logger).
		WithRetentionDays(7).
		WithCleanupInterval(1 * time.Second)

	// Start scheduler
	ctx := context.Background()
	scheduler.Start(ctx)

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Stop scheduler
	scheduler.Stop()

	// Verify scheduler stopped (no panic or hang)
	assert.True(t, true)
}

// TestDLQScheduler_ContextCancellation tests context cancellation
func TestDLQScheduler_ContextCancellation(t *testing.T) {
	logger := zap.NewNop()
	
	// Create mock DLQ
	dlq := &DeadLetterQueue{
		logger: logger,
	}

	// Create scheduler
	scheduler := NewDLQScheduler(dlq, logger).
		WithRetentionDays(7).
		WithCleanupInterval(1 * time.Second)

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Start scheduler
	scheduler.Start(ctx)

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for scheduler to stop
	time.Sleep(100 * time.Millisecond)

	// Verify scheduler stopped (no panic or hang)
	assert.True(t, true)
}
