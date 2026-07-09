package worker

import (
	"context"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/database"
	"github.com/al-hudhud/backend/internal/queue"
	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestDefaultRecoveryConfig(t *testing.T) {
	config := DefaultRecoveryConfig()

	assert.True(t, config.EnableAutoRecovery)
	assert.Equal(t, 100, config.RecoveryBatchSize)
	assert.Equal(t, 5*time.Minute, config.RecoveryTimeout)
	assert.Equal(t, 5*time.Minute, config.StaleMessageThreshold)
	assert.Equal(t, 3, config.MaxRecoveryAttempts)
}

func TestNewRecoveryManager(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	defer db.Close()

	logger := zap.NewNop()

	// Create River client
	driver := riverpgxv5.New(db.Pool)
	workers := river.NewWorkers()
	client, err := river.NewClient(driver, &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 1},
		},
		Workers: workers,
	})
	require.NoError(t, err)

	// Test with default config
	rm := NewRecoveryManager(db, logger, client, nil)
	assert.NotNil(t, rm)
	assert.NotNil(t, rm.config)
	assert.True(t, rm.config.EnableAutoRecovery)

	// Test with custom config
	customConfig := &RecoveryConfig{
		EnableAutoRecovery:    false,
		RecoveryBatchSize:     50,
		RecoveryTimeout:       2 * time.Minute,
		StaleMessageThreshold: 3 * time.Minute,
		MaxRecoveryAttempts:   5,
	}

	rm = NewRecoveryManager(db, logger, client, customConfig)
	assert.NotNil(t, rm)
	assert.Equal(t, customConfig, rm.config)
	assert.False(t, rm.config.EnableAutoRecovery)
	assert.Equal(t, 50, rm.config.RecoveryBatchSize)
}

func TestRecoverStaleMessages(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	logger := zap.NewNop()

	// Create River client
	driver := riverpgxv5.New(db.Pool)
	workers := river.NewWorkers()
	
	// Register a dummy worker
	river.AddWorker(workers, &MessageWorker{
		dependencies: &MessageWorkerDependencies{
			Logger: logger,
		},
	})
	
	client, err := river.NewClient(driver, &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 1},
		},
		Workers: workers,
	})
	require.NoError(t, err)

	// Start client
	err = client.Start(ctx)
	require.NoError(t, err)
	defer client.Stop(ctx)

	rm := NewRecoveryManager(db, logger, client, DefaultRecoveryConfig())

	// Create test data - stale message in PROCESSING state
	tenantID := uuid.New()
	messageID := uuid.New()

	// Insert message with old updated_at (stale)
	staleTime := time.Now().Add(-10 * time.Minute)
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO messages (
			tenant_id, message_id, phone_number, message_content,
			status, cost, priority, updated_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, tenantID, messageID, "+1234567890", "Test message",
		"PROCESSING", 0.05, "NORMAL", staleTime, staleTime)
	require.NoError(t, err)

	// Recover stale messages
	count, err := rm.recoverStaleMessages(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Verify message status reset to PENDING
	var status string
	err = db.Pool.QueryRow(ctx, `
		SELECT status FROM messages 
		WHERE tenant_id = $1 AND message_id = $2
	`, tenantID, messageID).Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, "PENDING", status)

	// Verify River job created
	var jobCount int
	err = db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM river_job
		WHERE args->>'tenant_id' = $1 AND args->>'message_id' = $2
	`, tenantID.String(), messageID.String()).Scan(&jobCount)
	require.NoError(t, err)
	assert.Equal(t, 1, jobCount)
}

func TestRecoverPendingMessages(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	logger := zap.NewNop()

	// Create River client
	driver := riverpgxv5.New(db.Pool)
	workers := river.NewWorkers()
	
	// Register a dummy worker
	river.AddWorker(workers, &MessageWorker{
		dependencies: &MessageWorkerDependencies{
			Logger: logger,
		},
	})
	
	client, err := river.NewClient(driver, &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 1},
		},
		Workers: workers,
	})
	require.NoError(t, err)

	// Start client
	err = client.Start(ctx)
	require.NoError(t, err)
	defer client.Stop(ctx)

	rm := NewRecoveryManager(db, logger, client, DefaultRecoveryConfig())

	// Create test data - pending message without River job
	tenantID := uuid.New()
	messageID := uuid.New()

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO messages (
			tenant_id, message_id, phone_number, message_content,
			status, cost, priority
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, tenantID, messageID, "+1234567890", "Test message",
		"PENDING", 0.05, "NORMAL")
	require.NoError(t, err)

	// Recover pending messages
	count, err := rm.recoverPendingMessages(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Verify River job created
	var jobCount int
	err = db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM river_job
		WHERE args->>'tenant_id' = $1 AND args->>'message_id' = $2
	`, tenantID.String(), messageID.String()).Scan(&jobCount)
	require.NoError(t, err)
	assert.Equal(t, 1, jobCount)
}

func TestRecoverFailedJobs(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	logger := zap.NewNop()

	// Create River client
	driver := riverpgxv5.New(db.Pool)
	workers := river.NewWorkers()
	
	// Register a dummy worker
	river.AddWorker(workers, &MessageWorker{
		dependencies: &MessageWorkerDependencies{
			Logger: logger,
		},
	})
	
	client, err := river.NewClient(driver, &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 1},
		},
		Workers: workers,
	})
	require.NoError(t, err)

	// Start client
	err = client.Start(ctx)
	require.NoError(t, err)
	defer client.Stop(ctx)

	rm := NewRecoveryManager(db, logger, client, DefaultRecoveryConfig())

	// Create test data - failed job
	tenantID := uuid.New()
	messageID := uuid.New()

	// Insert message
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO messages (
			tenant_id, message_id, phone_number, message_content,
			status, cost, priority
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, tenantID, messageID, "+1234567890", "Test message",
		"PENDING", 0.05, "NORMAL")
	require.NoError(t, err)

	// Insert failed River job
	tx, err := db.Pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	_, err = client.InsertTx(ctx, tx, queue.SendSMSArgs{
		TenantID:  tenantID,
		MessageID: messageID,
	}, &river.InsertOpts{
		MaxAttempts: 3,
	})
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	// Manually set job to retryable state
	_, err = db.Pool.Exec(ctx, `
		UPDATE river_job
		SET state = 'retryable', attempt = 1, scheduled_at = NOW() - INTERVAL '1 minute'
		WHERE args->>'tenant_id' = $1 AND args->>'message_id' = $2
	`, tenantID.String(), messageID.String())
	require.NoError(t, err)

	// Recover failed jobs
	count, err := rm.recoverFailedJobs(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Verify job state changed to available
	var jobState string
	err = db.Pool.QueryRow(ctx, `
		SELECT state FROM river_job
		WHERE args->>'tenant_id' = $1 AND args->>'message_id' = $2
	`, tenantID.String(), messageID.String()).Scan(&jobState)
	require.NoError(t, err)
	assert.Equal(t, "available", jobState)
}

func TestGetRecoveryStats(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	logger := zap.NewNop()

	// Create River client
	driver := riverpgxv5.New(db.Pool)
	workers := river.NewWorkers()
	client, err := river.NewClient(driver, &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 1},
		},
		Workers: workers,
	})
	require.NoError(t, err)

	rm := NewRecoveryManager(db, logger, client, DefaultRecoveryConfig())

	// Create test data
	tenantID := uuid.New()

	// Stale message
	staleTime := time.Now().Add(-10 * time.Minute)
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO messages (
			tenant_id, message_id, phone_number, message_content,
			status, cost, priority, updated_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, tenantID, uuid.New(), "+1234567890", "Stale message",
		"PROCESSING", 0.05, "NORMAL", staleTime, staleTime)
	require.NoError(t, err)

	// Pending message without job
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO messages (
			tenant_id, message_id, phone_number, message_content,
			status, cost, priority
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, tenantID, uuid.New(), "+1234567890", "Pending message",
		"PENDING", 0.05, "NORMAL")
	require.NoError(t, err)

	// Get recovery stats
	stats, err := rm.GetRecoveryStats(ctx)
	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 1, stats.StaleMessages)
	assert.Equal(t, 1, stats.PendingMessages)
	assert.Equal(t, 2, stats.TotalRecoverable)
}

func TestRecoverPendingMessages_DisabledAutoRecovery(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	logger := zap.NewNop()

	// Create River client
	driver := riverpgxv5.New(db.Pool)
	workers := river.NewWorkers()
	client, err := river.NewClient(driver, &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 1},
		},
		Workers: workers,
	})
	require.NoError(t, err)

	// Create recovery manager with auto recovery disabled
	config := DefaultRecoveryConfig()
	config.EnableAutoRecovery = false

	rm := NewRecoveryManager(db, logger, client, config)

	// Try to recover - should return immediately
	err = rm.RecoverPendingMessages(ctx)
	require.NoError(t, err)
}

func TestRecoverPendingMessages_Integration(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	logger := zap.NewNop()

	// Create River client
	driver := riverpgxv5.New(db.Pool)
	workers := river.NewWorkers()
	
	// Register a dummy worker
	river.AddWorker(workers, &MessageWorker{
		dependencies: &MessageWorkerDependencies{
			Logger: logger,
		},
	})
	
	client, err := river.NewClient(driver, &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 1},
		},
		Workers: workers,
	})
	require.NoError(t, err)

	// Start client
	err = client.Start(ctx)
	require.NoError(t, err)
	defer client.Stop(ctx)

	rm := NewRecoveryManager(db, logger, client, DefaultRecoveryConfig())

	// Create test data - mix of stale and pending messages
	tenantID := uuid.New()

	// Stale message
	staleTime := time.Now().Add(-10 * time.Minute)
	staleMessageID := uuid.New()
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO messages (
			tenant_id, message_id, phone_number, message_content,
			status, cost, priority, updated_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, tenantID, staleMessageID, "+1234567890", "Stale message",
		"PROCESSING", 0.05, "NORMAL", staleTime, staleTime)
	require.NoError(t, err)

	// Pending message
	pendingMessageID := uuid.New()
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO messages (
			tenant_id, message_id, phone_number, message_content,
			status, cost, priority
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, tenantID, pendingMessageID, "+1234567890", "Pending message",
		"PENDING", 0.05, "NORMAL")
	require.NoError(t, err)

	// Recover all pending messages
	err = rm.RecoverPendingMessages(ctx)
	require.NoError(t, err)

	// Verify stale message recovered
	var staleStatus string
	err = db.Pool.QueryRow(ctx, `
		SELECT status FROM messages 
		WHERE tenant_id = $1 AND message_id = $2
	`, tenantID, staleMessageID).Scan(&staleStatus)
	require.NoError(t, err)
	assert.Equal(t, "PENDING", staleStatus)

	// Verify River jobs created for both messages
	var jobCount int
	err = db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM river_job
		WHERE args->>'tenant_id' = $1
	`, tenantID.String()).Scan(&jobCount)
	require.NoError(t, err)
	assert.Equal(t, 2, jobCount)
}

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) *database.DB {
	// This is a placeholder - in real tests, you would set up a test database
	// For now, we'll skip tests that require a real database
	t.Skip("Skipping test that requires database connection")
	return nil
}
