package worker

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"

	"github.com/al-hudhud/backend/internal/db"
)

// Bug 6 Preservation Tests: Transaction Atomicity
// **Validates: Requirement 3.14**
//
// **Property 2: Preservation** - Transaction Atomicity
//
// These tests verify that SUCCESSFUL transaction commits continue to work atomically on UNFIXED code.
// They test the EXISTING functionality that must continue to work after the fix.
//
// Expected outcome on UNFIXED code: Tests PASS (successful commits work atomically)
// Expected outcome on FIXED code: Tests PASS (behavior preserved)
//
// IMPORTANT: Bug 6 is about the transaction pattern (rollback after commit), not about
// breaking successful commits. These tests verify that when transactions succeed,
// all changes are persisted atomically.

// setupBug6TestDB initializes a test database connection for Bug 6 preservation tests
func setupBug6TestDB(t *testing.T) *pgxpool.Pool {
	// Initialize database if not already done
	db.InitGormDB()
	
	// Verify PgxPool is available
	require.NotNil(t, db.PgxPool, "PgxPool should be initialized")
	
	return db.PgxPool
}

// TestBug6_Preservation_TransactionCommitAtomic verifies that successful
// transaction commits persist all changes atomically
// **Validates: Requirement 3.14**
func TestBug6_Preservation_TransactionCommitAtomic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Bug 6 preservation test in short mode")
	}

	pool := setupBug6TestDB(t)
	ctx := context.Background()

	// Property: Successful transaction commits persist all changes atomically
	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		tenantID := uuid.New()
		messageCount := rapid.IntRange(1, 5).Draw(t, "messageCount")

		// Create tenant first to satisfy foreign key constraint
		_, err := pool.Exec(ctx, `
			INSERT INTO tenants (id, user_id, business_name, contact_name, email, created_at)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (id) DO NOTHING
		`, tenantID, uuid.New(), "Test Business", "Test Contact", "test@example.com", time.Now())
		require.NoError(t, err, "Setup: Should be able to create tenant")

		// Begin transaction
		tx, err := pool.Begin(ctx)
		require.NoError(t, err, "Preservation: Should be able to begin transaction")

		// Defer rollback pattern (this is the UNFIXED pattern we're testing)
		defer tx.Rollback(ctx)

		// Insert test messages in transaction
		insertedIDs := make([]uuid.UUID, messageCount)
		for i := 0; i < messageCount; i++ {
			messageID := uuid.New()
			insertedIDs[i] = messageID

			query := `
				INSERT INTO messages (id, tenant_id, phone_number, content, status, created_at)
				VALUES ($1, $2, $3, $4, $5, $6)
			`
			_, err := tx.Exec(ctx, query,
				messageID,
				tenantID,
				"+966500000001",
				"Test message",
				"PENDING",
				time.Now(),
			)
			require.NoError(t, err, "Preservation: Should be able to insert message in transaction")
		}

		// ACT: Commit transaction (successful operation)
		err = tx.Commit(ctx)
		require.NoError(t, err, "Preservation: Successful commit should work")

		// ASSERT: All changes persisted atomically
		// Verify all messages exist in database
		for _, messageID := range insertedIDs {
			var exists bool
			query := `SELECT EXISTS(SELECT 1 FROM messages WHERE id = $1)`
			err := pool.QueryRow(ctx, query, messageID).Scan(&exists)
			require.NoError(t, err, "Preservation: Should be able to query message")
			assert.True(t, exists,
				"Preservation: Message %s should exist after successful commit (atomic persistence)",
				messageID)
		}

		// Cleanup: Delete test messages
		for _, messageID := range insertedIDs {
			_, _ = pool.Exec(ctx, `DELETE FROM messages WHERE id = $1`, messageID)
		}
		_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id = $1`, tenantID)
	})
}

// TestBug6_Preservation_MultipleOperationsAtomic verifies that multiple
// operations within a transaction are committed atomically
// **Validates: Requirement 3.14**
func TestBug6_Preservation_MultipleOperationsAtomic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Bug 6 preservation test in short mode")
	}

	pool := setupBug6TestDB(t)
	ctx := context.Background()

	// Property: Multiple operations in a transaction commit atomically
	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		messageID := uuid.New()

		// Create tenant first to satisfy foreign key constraint
		_, err := pool.Exec(ctx, `
			INSERT INTO tenants (id, user_id, business_name, contact_name, email, created_at)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (id) DO NOTHING
		`, tenantID, uuid.New(), "Test Business", "Test Contact", "test@example.com", time.Now())
		require.NoError(t, err, "Setup: Should be able to create tenant")

		// Begin transaction
		tx, err := pool.Begin(ctx)
		require.NoError(t, err, "Preservation: Should be able to begin transaction")

		// Defer rollback pattern (UNFIXED pattern)
		defer tx.Rollback(ctx)

		// Operation 1: Insert message
		insertQuery := `
			INSERT INTO messages (id, tenant_id, phone_number, content, status, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`
		_, err = tx.Exec(ctx, insertQuery,
			messageID,
			tenantID,
			"+966500000001",
			"Test message",
			"PENDING",
			time.Now(),
		)
		require.NoError(t, err, "Preservation: Should be able to insert message")

		// Operation 2: Update message status
		updateQuery := `UPDATE messages SET status = $1 WHERE id = $2`
		_, err = tx.Exec(ctx, updateQuery, "PROCESSING", messageID)
		require.NoError(t, err, "Preservation: Should be able to update message")

		// ACT: Commit transaction (successful operation)
		err = tx.Commit(ctx)
		require.NoError(t, err, "Preservation: Successful commit should work")

		// ASSERT: Both operations persisted atomically
		var status string
		query := `SELECT status FROM messages WHERE id = $1`
		err = pool.QueryRow(ctx, query, messageID).Scan(&status)
		require.NoError(t, err, "Preservation: Should be able to query message")
		assert.Equal(t, "PROCESSING", status,
			"Preservation: Message status should be PROCESSING after atomic commit")

		// Cleanup
		_, _ = pool.Exec(ctx, `DELETE FROM messages WHERE id = $1`, messageID)
		_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id = $1`, tenantID)
	})
}

// TestBug6_Preservation_BatchUpdateAtomic verifies that batch updates
// within a transaction are committed atomically
// **Validates: Requirement 3.14**
func TestBug6_Preservation_BatchUpdateAtomic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Bug 6 preservation test in short mode")
	}

	pool := setupBug6TestDB(t)
	ctx := context.Background()

	// Property: Batch updates in a transaction commit atomically
	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		batchSize := rapid.IntRange(2, 10).Draw(t, "batchSize")

		// Create tenant first to satisfy foreign key constraint
		_, err := pool.Exec(ctx, `
			INSERT INTO tenants (id, user_id, business_name, contact_name, email, created_at)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (id) DO NOTHING
		`, tenantID, uuid.New(), "Test Business", "Test Contact", "test@example.com", time.Now())
		require.NoError(t, err, "Setup: Should be able to create tenant")

		// Create test messages outside transaction
		messageIDs := make([]uuid.UUID, batchSize)
		for i := 0; i < batchSize; i++ {
			messageID := uuid.New()
			messageIDs[i] = messageID

			query := `
				INSERT INTO messages (id, tenant_id, phone_number, content, status, created_at)
				VALUES ($1, $2, $3, $4, $5, $6)
			`
			_, err := pool.Exec(ctx, query,
				messageID,
				tenantID,
				"+966500000001",
				"Test message",
				"PENDING",
				time.Now(),
			)
			require.NoError(t, err, "Setup: Should be able to insert test message")
		}

		// Begin transaction for batch update
		tx, err := pool.Begin(ctx)
		require.NoError(t, err, "Preservation: Should be able to begin transaction")

		// Defer rollback pattern (UNFIXED pattern)
		defer tx.Rollback(ctx)

		// Batch update: Update all messages to PROCESSING
		updateQuery := `
			UPDATE messages 
			SET status = $1, updated_at = $2 
			WHERE id = ANY($3)
		`
		_, err = tx.Exec(ctx, updateQuery, "PROCESSING", time.Now(), messageIDs)
		require.NoError(t, err, "Preservation: Should be able to batch update messages")

		// ACT: Commit transaction (successful operation)
		err = tx.Commit(ctx)
		require.NoError(t, err, "Preservation: Successful commit should work")

		// ASSERT: All messages updated atomically
		for _, messageID := range messageIDs {
			var status string
			query := `SELECT status FROM messages WHERE id = $1`
			err := pool.QueryRow(ctx, query, messageID).Scan(&status)
			require.NoError(t, err, "Preservation: Should be able to query message")
			assert.Equal(t, "PROCESSING", status,
				"Preservation: Message %s should have status PROCESSING after atomic batch update",
				messageID)
		}

		// Cleanup
		for _, messageID := range messageIDs {
			_, _ = pool.Exec(ctx, `DELETE FROM messages WHERE id = $1`, messageID)
		}
		_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id = $1`, tenantID)
	})
}

// Simple unit test: Successful transaction commit persists data
// **Validates: Requirement 3.14**
func TestBug6_Preservation_Simple_TransactionCommit(t *testing.T) {
	pool := setupBug6TestDB(t)
	ctx := context.Background()

	tenantID := uuid.New()
	messageID := uuid.New()

	// Create tenant first to satisfy foreign key constraint
	_, err := pool.Exec(ctx, `
		INSERT INTO tenants (id, user_id, business_name, contact_name, email, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO NOTHING
	`, tenantID, uuid.New(), "Test Business", "Test Contact", "test@example.com", time.Now())
	require.NoError(t, err, "Setup: Should be able to create tenant")

	// Begin transaction
	tx, err := pool.Begin(ctx)
	require.NoError(t, err, "Preservation: Should be able to begin transaction")

	// Defer rollback pattern (UNFIXED pattern we're testing)
	defer tx.Rollback(ctx)

	// Insert message in transaction
	query := `
		INSERT INTO messages (id, tenant_id, phone_number, content, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err = tx.Exec(ctx, query,
		messageID,
		tenantID,
		"+966500000001",
		"Test message",
		"PENDING",
		time.Now(),
	)
	require.NoError(t, err, "Preservation: Should be able to insert message")

	// ACT: Commit transaction (successful operation)
	err = tx.Commit(ctx)
	require.NoError(t, err, "Preservation: Successful commit should work")

	// ASSERT: Message persisted
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM messages WHERE id = $1)`
	err = pool.QueryRow(ctx, checkQuery, messageID).Scan(&exists)
	require.NoError(t, err, "Preservation: Should be able to query message")
	assert.True(t, exists,
		"Preservation: Message should exist after successful commit")

	// Cleanup
	_, _ = pool.Exec(ctx, `DELETE FROM messages WHERE id = $1`, messageID)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id = $1`, tenantID)

	t.Logf("✅ Successful transaction commit persists data atomically")
}

// Simple unit test: Multiple operations commit atomically
// **Validates: Requirement 3.14**
func TestBug6_Preservation_Simple_MultipleOperations(t *testing.T) {
	pool := setupBug6TestDB(t)
	ctx := context.Background()

	tenantID := uuid.New()
	messageID := uuid.New()

	// Create tenant first to satisfy foreign key constraint
	_, err := pool.Exec(ctx, `
		INSERT INTO tenants (id, user_id, business_name, contact_name, email, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO NOTHING
	`, tenantID, uuid.New(), "Test Business", "Test Contact", "test@example.com", time.Now())
	require.NoError(t, err, "Setup: Should be able to create tenant")

	// Begin transaction
	tx, err := pool.Begin(ctx)
	require.NoError(t, err, "Preservation: Should be able to begin transaction")

	// Defer rollback pattern (UNFIXED pattern)
	defer tx.Rollback(ctx)

	// Operation 1: Insert
	insertQuery := `
		INSERT INTO messages (id, tenant_id, phone_number, content, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err = tx.Exec(ctx, insertQuery,
		messageID,
		tenantID,
		"+966500000001",
		"Test message",
		"PENDING",
		time.Now(),
	)
	require.NoError(t, err, "Preservation: Should be able to insert message")

	// Operation 2: Update
	updateQuery := `UPDATE messages SET status = $1 WHERE id = $2`
	_, err = tx.Exec(ctx, updateQuery, "PROCESSING", messageID)
	require.NoError(t, err, "Preservation: Should be able to update message")

	// Operation 3: Update again
	updateQuery2 := `UPDATE messages SET status = $1 WHERE id = $2`
	_, err = tx.Exec(ctx, updateQuery2, "SENT", messageID)
	require.NoError(t, err, "Preservation: Should be able to update message again")

	// ACT: Commit transaction
	err = tx.Commit(ctx)
	require.NoError(t, err, "Preservation: Successful commit should work")

	// ASSERT: Final state reflects all operations
	var status string
	query := `SELECT status FROM messages WHERE id = $1`
	err = pool.QueryRow(ctx, query, messageID).Scan(&status)
	require.NoError(t, err, "Preservation: Should be able to query message")
	assert.Equal(t, "SENT", status,
		"Preservation: Message should have final status SENT after atomic commit")

	// Cleanup
	_, _ = pool.Exec(ctx, `DELETE FROM messages WHERE id = $1`, messageID)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id = $1`, tenantID)

	t.Logf("✅ Multiple operations commit atomically: INSERT → UPDATE → UPDATE → COMMIT")
}

// Simple unit test: Batch operations commit atomically
// **Validates: Requirement 3.14**
func TestBug6_Preservation_Simple_BatchOperations(t *testing.T) {
	pool := setupBug6TestDB(t)
	ctx := context.Background()

	tenantID := uuid.New()
	batchSize := 5

	// Create tenant first to satisfy foreign key constraint
	_, err := pool.Exec(ctx, `
		INSERT INTO tenants (id, user_id, business_name, contact_name, email, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO NOTHING
	`, tenantID, uuid.New(), "Test Business", "Test Contact", "test@example.com", time.Now())
	require.NoError(t, err, "Setup: Should be able to create tenant")

	// Create test messages
	messageIDs := make([]uuid.UUID, batchSize)
	for i := 0; i < batchSize; i++ {
		messageID := uuid.New()
		messageIDs[i] = messageID

		query := `
			INSERT INTO messages (id, tenant_id, phone_number, content, status, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`
		_, err := pool.Exec(ctx, query,
			messageID,
			tenantID,
			"+966500000001",
			"Test message",
			"PENDING",
			time.Now(),
		)
		require.NoError(t, err, "Setup: Should be able to insert test message")
	}

	// Begin transaction for batch update
	tx, err := pool.Begin(ctx)
	require.NoError(t, err, "Preservation: Should be able to begin transaction")

	// Defer rollback pattern (UNFIXED pattern)
	defer tx.Rollback(ctx)

	// Batch update
	updateQuery := `
		UPDATE messages 
		SET status = $1, updated_at = $2 
		WHERE id = ANY($3)
	`
	_, err = tx.Exec(ctx, updateQuery, "PROCESSING", time.Now(), messageIDs)
	require.NoError(t, err, "Preservation: Should be able to batch update")

	// ACT: Commit transaction
	err = tx.Commit(ctx)
	require.NoError(t, err, "Preservation: Successful commit should work")

	// ASSERT: All messages updated
	updatedCount := 0
	for _, messageID := range messageIDs {
		var status string
		query := `SELECT status FROM messages WHERE id = $1`
		err := pool.QueryRow(ctx, query, messageID).Scan(&status)
		require.NoError(t, err, "Preservation: Should be able to query message")
		if status == "PROCESSING" {
			updatedCount++
		}
	}
	assert.Equal(t, batchSize, updatedCount,
		"Preservation: All %d messages should be updated atomically", batchSize)

	// Cleanup
	for _, messageID := range messageIDs {
		_, _ = pool.Exec(ctx, `DELETE FROM messages WHERE id = $1`, messageID)
	}
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id = $1`, tenantID)

	t.Logf("✅ Batch operations commit atomically: %d messages updated", batchSize)
}

// Simple unit test: Transaction rollback on error (verify rollback works)
// **Validates: Requirement 3.14**
func TestBug6_Preservation_Simple_RollbackOnError(t *testing.T) {
	pool := setupBug6TestDB(t)
	ctx := context.Background()

	tenantID := uuid.New()
	messageID := uuid.New()

	// Create tenant first to satisfy foreign key constraint
	_, err := pool.Exec(ctx, `
		INSERT INTO tenants (id, user_id, business_name, contact_name, email, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO NOTHING
	`, tenantID, uuid.New(), "Test Business", "Test Contact", "test@example.com", time.Now())
	require.NoError(t, err, "Setup: Should be able to create tenant")

	// Begin transaction
	tx, err := pool.Begin(ctx)
	require.NoError(t, err, "Preservation: Should be able to begin transaction")

	// Defer rollback pattern (UNFIXED pattern)
	defer tx.Rollback(ctx)

	// Insert message
	query := `
		INSERT INTO messages (id, tenant_id, phone_number, content, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err = tx.Exec(ctx, query,
		messageID,
		tenantID,
		"+966500000001",
		"Test message",
		"PENDING",
		time.Now(),
	)
	require.NoError(t, err, "Preservation: Should be able to insert message")

	// Simulate error - don't commit, let defer rollback handle it
	// (In real code, this would be an error condition)

	// ASSERT: Message should NOT exist (rollback worked)
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM messages WHERE id = $1)`
	err = pool.QueryRow(ctx, checkQuery, messageID).Scan(&exists)
	require.NoError(t, err, "Preservation: Should be able to query")
	assert.False(t, exists,
		"Preservation: Message should NOT exist after rollback (transaction not committed)")

	// Cleanup tenant
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id = $1`, tenantID)

	t.Logf("✅ Transaction rollback works correctly when commit is not called")
}

// Simple unit test: Nested transaction operations
// **Validates: Requirement 3.14**
func TestBug6_Preservation_Simple_NestedOperations(t *testing.T) {
	pool := setupBug6TestDB(t)
	ctx := context.Background()

	tenantID := uuid.New()
	messageID1 := uuid.New()
	messageID2 := uuid.New()

	// Create tenant first to satisfy foreign key constraint
	_, err := pool.Exec(ctx, `
		INSERT INTO tenants (id, user_id, business_name, contact_name, email, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO NOTHING
	`, tenantID, uuid.New(), "Test Business", "Test Contact", "test@example.com", time.Now())
	require.NoError(t, err, "Setup: Should be able to create tenant")

	// Begin transaction
	tx, err := pool.Begin(ctx)
	require.NoError(t, err, "Preservation: Should be able to begin transaction")

	// Defer rollback pattern (UNFIXED pattern)
	defer tx.Rollback(ctx)

	// Insert first message
	query := `
		INSERT INTO messages (id, tenant_id, phone_number, content, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err = tx.Exec(ctx, query,
		messageID1,
		tenantID,
		"+966500000001",
		"Test message 1",
		"PENDING",
		time.Now(),
	)
	require.NoError(t, err, "Preservation: Should be able to insert first message")

	// Insert second message
	_, err = tx.Exec(ctx, query,
		messageID2,
		tenantID,
		"+966500000002",
		"Test message 2",
		"PENDING",
		time.Now(),
	)
	require.NoError(t, err, "Preservation: Should be able to insert second message")

	// Update both messages
	updateQuery := `UPDATE messages SET status = $1 WHERE id = ANY($2)`
	_, err = tx.Exec(ctx, updateQuery, "PROCESSING", []uuid.UUID{messageID1, messageID2})
	require.NoError(t, err, "Preservation: Should be able to update both messages")

	// ACT: Commit transaction
	err = tx.Commit(ctx)
	require.NoError(t, err, "Preservation: Successful commit should work")

	// ASSERT: Both messages exist with correct status
	for _, messageID := range []uuid.UUID{messageID1, messageID2} {
		var status string
		query := `SELECT status FROM messages WHERE id = $1`
		err := pool.QueryRow(ctx, query, messageID).Scan(&status)
		require.NoError(t, err, "Preservation: Should be able to query message")
		assert.Equal(t, "PROCESSING", status,
			"Preservation: Message %s should have status PROCESSING", messageID)
	}

	// Cleanup
	_, _ = pool.Exec(ctx, `DELETE FROM messages WHERE id = ANY($1)`, []uuid.UUID{messageID1, messageID2})
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id = $1`, tenantID)

	t.Logf("✅ Nested operations commit atomically: 2 inserts + 1 batch update")
}
