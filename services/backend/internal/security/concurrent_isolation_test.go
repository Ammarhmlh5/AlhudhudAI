package security

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestConcurrentTenantOperations tests that concurrent operations from different tenants
// don't interfere with each other
func TestConcurrentTenantOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent isolation test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Create multiple tenants
	numTenants := 10
	tenants := make([]uuid.UUID, numTenants)
	for i := 0; i < numTenants; i++ {
		tenantID := createTestTenant(t, db, fmt.Sprintf("Tenant %d", i))
		tenants[i] = tenantID
	}

	// Ensure the service referenced by the tariff rows exists (FK constraint)
	conn, err := db.DB()
	require.NoError(t, err)
	_, err = conn.ExecContext(ctx, `
		INSERT INTO services (id, tenant_id, name, code, type, is_active)
		VALUES (1, $1, 'Test Service', 'TEST_SVC', 'SMS', true)
		ON CONFLICT (id) DO NOTHING
	`, tenants[0])
	require.NoError(t, err)

	// Number of concurrent operations per tenant
	opsPerTenant := 50

	var wg sync.WaitGroup
	errors := make(chan error, numTenants*opsPerTenant)

	// Launch concurrent operations for each tenant
	for _, tenantID := range tenants {
		for i := 0; i < opsPerTenant; i++ {
			wg.Add(1)
			go func(tid uuid.UUID, opNum int) {
				defer wg.Done()

				// Perform various operations
				sqlDB, _ := db.DB()
				if err := performTenantOperation(ctx, sqlDB, tid, opNum); err != nil {
					errors <- fmt.Errorf("tenant %s op %d: %w", tid, opNum, err)
				}
			}(tenantID, i)
		}
	}

	// Wait for all operations to complete
	wg.Wait()
	close(errors)

	// Check for errors
	var errorList []error
	for err := range errors {
		errorList = append(errorList, err)
	}

	if len(errorList) > 0 {
		t.Errorf("Found %d errors during concurrent operations:", len(errorList))
		for _, err := range errorList {
			t.Logf("  - %v", err)
		}
		t.FailNow()
	}

	// Verify data integrity for each tenant
	sqlDB, _ := db.DB()
	for _, tenantID := range tenants {
		verifyTenantDataIntegrity(t, sqlDB, tenantID, opsPerTenant)
	}
}

// TestConcurrentBalanceOperations tests concurrent balance operations across tenants
func TestConcurrentBalanceOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent balance test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Create tenants with initial balance
	numTenants := 5
	initialBalance := 10000.0
	tenants := make([]uuid.UUID, numTenants)

	for i := 0; i < numTenants; i++ {
		tenantID := createTestTenant(t, db, fmt.Sprintf("Balance Tenant %d", i))
		tenants[i] = tenantID

		// Set initial balance
		sqlDB, _ := db.DB()
		_, err := sqlDB.ExecContext(ctx, `
			UPDATE tenants SET balance = $1 WHERE id = $2
		`, initialBalance, tenantID)
		require.NoError(t, err)
	}

	// Concurrent deductions per tenant
	deductionsPerTenant := 100
	deductionAmount := 10.0

	var wg sync.WaitGroup
	errors := make(chan error, numTenants*deductionsPerTenant)

	// Launch concurrent balance deductions
	for _, tenantID := range tenants {
		for i := 0; i < deductionsPerTenant; i++ {
			wg.Add(1)
			go func(tid uuid.UUID) {
				defer wg.Done()

				// Deduct balance
				sqlDB, _ := db.DB()
				result, err := sqlDB.ExecContext(ctx, `
					UPDATE tenants 
					SET balance = balance - $1 
					WHERE id = $2 AND balance >= $1
				`, deductionAmount, tid)

				if err != nil {
					errors <- fmt.Errorf("deduction error for tenant %s: %w", tid, err)
					return
				}

				rows, err := result.RowsAffected()
				if err != nil {
					errors <- fmt.Errorf("rows affected error for tenant %s: %w", tid, err)
					return
				}

				if rows == 0 {
					errors <- fmt.Errorf("insufficient balance for tenant %s", tid)
				}
			}(tenantID)
		}
	}

	wg.Wait()
	close(errors)

	// Check for errors
	var errorList []error
	for err := range errors {
		errorList = append(errorList, err)
	}

	assert.Empty(t, errorList, "Should have no errors during concurrent balance operations")

	// Verify final balances
	for _, tenantID := range tenants {
		var finalBalance float64
		sqlDB, _ := db.DB()
		err := sqlDB.QueryRowContext(ctx, `
			SELECT balance FROM tenants WHERE id = $1
		`, tenantID).Scan(&finalBalance)
		require.NoError(t, err)

		expectedBalance := initialBalance - (float64(deductionsPerTenant) * deductionAmount)
		assert.Equal(t, expectedBalance, finalBalance,
			"Tenant %s should have correct final balance", tenantID)
	}
}

// TestConcurrentDataIsolation verifies no data leakage between tenants under concurrent load
func TestConcurrentDataIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent data isolation test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Create two tenants
	tenant1 := createTestTenant(t, db, "Isolated Tenant 1")
	tenant2 := createTestTenant(t, db, "Isolated Tenant 2")

	// Create test data for each tenant
	numRecords := 100

	var wg sync.WaitGroup
	errors := make(chan error, numRecords*2)

	// Tenant 1: Create tariffs concurrently
	for i := 0; i < numRecords; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			sqlDB, _ := db.DB()
			_, err := sqlDB.ExecContext(ctx, `
				INSERT INTO tariffs (
					tenant_id, service_id, country_code, network_operator,
					cost_per_message, currency, effective_from, is_active
				) VALUES ($1, 1, 'YE', $2, 10.0, 'YER', CURRENT_DATE, true)
			`, tenant1, fmt.Sprintf("Operator-T1-%d", idx))

			if err != nil {
				errors <- fmt.Errorf("tenant1 insert %d: %w", idx, err)
			}
		}(i)
	}

	// Tenant 2: Create tariffs concurrently
	for i := 0; i < numRecords; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			sqlDB, _ := db.DB()
			_, err := sqlDB.ExecContext(ctx, `
				INSERT INTO tariffs (
					tenant_id, service_id, country_code, network_operator,
					cost_per_message, currency, effective_from, is_active
				) VALUES ($1, 1, 'YE', $2, 15.0, 'YER', CURRENT_DATE, true)
			`, tenant2, fmt.Sprintf("Operator-T2-%d", idx))

			if err != nil {
				errors <- fmt.Errorf("tenant2 insert %d: %w", idx, err)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	var errorList []error
	for err := range errors {
		errorList = append(errorList, err)
	}
	assert.Empty(t, errorList, "Should have no errors during concurrent inserts")

	// Verify data isolation
	sqlDB, _ := db.DB()
	var count1 int
	err := sqlDB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM tariffs WHERE tenant_id = $1
	`, tenant1).Scan(&count1)
	require.NoError(t, err)
	assert.Equal(t, numRecords, count1, "Tenant 1 should have exactly %d records", numRecords)

	var count2 int
	err = sqlDB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM tariffs WHERE tenant_id = $1
	`, tenant2).Scan(&count2)
	require.NoError(t, err)
	assert.Equal(t, numRecords, count2, "Tenant 2 should have exactly %d records", numRecords)

	// Verify no cross-tenant data
	var crossCount int
	err = sqlDB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM tariffs 
		WHERE tenant_id = $1 AND network_operator LIKE 'Operator-T2-%'
	`, tenant1).Scan(&crossCount)
	require.NoError(t, err)
	assert.Equal(t, 0, crossCount, "Tenant 1 should not have Tenant 2's data")

	err = sqlDB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM tariffs 
		WHERE tenant_id = $1 AND network_operator LIKE 'Operator-T1-%'
	`, tenant2).Scan(&crossCount)
	require.NoError(t, err)
	assert.Equal(t, 0, crossCount, "Tenant 2 should not have Tenant 1's data")
}

// TestConcurrentRLSEnforcement tests Row Level Security under concurrent load
func TestConcurrentRLSEnforcement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent RLS test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Create tenants
	tenant1 := createTestTenant(t, db, "RLS Tenant 1")
	tenant2 := createTestTenant(t, db, "RLS Tenant 2")

	// Create test data
	sqlDB, _ := db.DB()
	_, err := sqlDB.ExecContext(ctx, `
		INSERT INTO tariffs (
			tenant_id, service_id, country_code, network_operator,
			cost_per_message, currency, effective_from, is_active
		) VALUES 
		($1, 1, 'YE', 'Yemen Mobile', 9.0, 'YER', CURRENT_DATE, true),
		($2, 1, 'YE', 'Sabafon', 13.0, 'YER', CURRENT_DATE, true)
	`, tenant1, tenant2)
	require.NoError(t, err)

	numQueries := 200
	var wg sync.WaitGroup
	results := make(chan int, numQueries*2)

	// Tenant 1: Concurrent queries with RLS
	for i := 0; i < numQueries; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Set tenant context
			sqlDB, _ := db.DB()
			_, err := sqlDB.ExecContext(ctx, `SET app.current_tenant = $1`, tenant1.String())
			if err != nil {
				t.Logf("Error setting tenant context: %v", err)
				return
			}

			var count int
			err = sqlDB.QueryRowContext(ctx, `
				SELECT COUNT(*) FROM tariffs WHERE is_active = true
			`).Scan(&count)

			if err != nil {
				t.Logf("Error querying tariffs: %v", err)
				return
			}

			results <- count
		}()
	}

	// Tenant 2: Concurrent queries with RLS
	for i := 0; i < numQueries; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Set tenant context
			sqlDB, _ := db.DB()
			_, err := sqlDB.ExecContext(ctx, `SET app.current_tenant = $1`, tenant2.String())
			if err != nil {
				t.Logf("Error setting tenant context: %v", err)
				return
			}

			var count int
			err = sqlDB.QueryRowContext(ctx, `
				SELECT COUNT(*) FROM tariffs WHERE is_active = true
			`).Scan(&count)

			if err != nil {
				t.Logf("Error querying tariffs: %v", err)
				return
			}

			results <- count
		}()
	}

	wg.Wait()
	close(results)

	// Verify all queries returned correct isolated counts
	for count := range results {
		assert.Equal(t, 1, count, "Each tenant should see only their own record")
	}
}

// TestConcurrentCacheIsolation tests Redis cache isolation under concurrent load
func TestConcurrentCacheIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent cache isolation test in short mode")
	}

	ctx := context.Background()
	redisClient := setupTestRedis(t)
	defer cleanupTestRedis(t, redisClient)

	// Create tenants
	numTenants := 5
	tenants := make([]uuid.UUID, numTenants)
	for i := 0; i < numTenants; i++ {
		tenants[i] = uuid.New()
	}

	// Concurrent cache operations
	opsPerTenant := 100
	var wg sync.WaitGroup
	errors := make(chan error, numTenants*opsPerTenant)
	written := make(chan struct{ key, value string }, numTenants*opsPerTenant)

	for _, tenantID := range tenants {
		for i := 0; i < opsPerTenant; i++ {
			wg.Add(1)
			go func(tid uuid.UUID, opNum int) {
				defer wg.Done()

				// Write tenant-specific data
				key := fmt.Sprintf("tenant:%s:data:%d", tid.String(), opNum)
				value := fmt.Sprintf("value-%s-%d", tid.String(), opNum)

				// The full-suite run saturates the WSL2 Redis relay and can
				// transiently exceed a single round trip. Retry briefly;
				// isolation is the property under test, not raw throughput.
				var err error
				for attempt := 0; attempt < 3; attempt++ {
					err = redisClient.Set(ctx, key, value, 5*time.Minute).Err()
					if err == nil {
						break
					}
				}
				if err != nil {
					errors <- fmt.Errorf("cache set error: %w", err)
					return
				}

				// Read back and verify
				retrieved, err := redisClient.Get(ctx, key).Result()
				if err != nil {
					errors <- fmt.Errorf("cache get error: %w", err)
					return
				}

				if retrieved != value {
					errors <- fmt.Errorf("cache value mismatch: expected %s, got %s", value, retrieved)
					return
				}
				written <- struct{ key, value string }{key, value}
			}(tenantID, i)
		}
	}

	wg.Wait()
	close(errors)
	close(written)

	// Check for errors
	var errorList []error
	for err := range errors {
		errorList = append(errorList, err)
	}
	assert.Empty(t, errorList, "Should have no cache isolation errors")

	// Verify no cross-tenant cache pollution for the keys that were written
	for kv := range written {
		retrieved, err := redisClient.Get(ctx, kv.key).Result()
		require.NoError(t, err)
		assert.Equal(t, kv.value, retrieved, "Cache value should match for tenant %s", kv.key)
	}
}

// TestConcurrentTransactionIsolation tests transaction isolation between tenants
func TestConcurrentTransactionIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent transaction isolation test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Create tenants
	tenant1 := createTestTenant(t, db, "Transaction Tenant 1")
	tenant2 := createTestTenant(t, db, "Transaction Tenant 2")

	// Set initial balances
	sqlDB, _ := db.DB()
	_, err := sqlDB.ExecContext(ctx, `
		UPDATE tenants SET balance = 5000.0 WHERE id IN ($1, $2)
	`, tenant1, tenant2)
	require.NoError(t, err)

	numTransactions := 50
	var wg sync.WaitGroup
	errors := make(chan error, numTransactions*2)

	// Tenant 1: Concurrent transactions
	for i := 0; i < numTransactions; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			sqlDB, _ := db.DB()
			tx, err := sqlDB.BeginTx(ctx, nil)
			if err != nil {
				errors <- fmt.Errorf("begin tx error: %w", err)
				return
			}
			defer tx.Rollback()

			// Deduct balance
			_, err = tx.ExecContext(ctx, `
				UPDATE tenants SET balance = balance - 10.0 
				WHERE id = $1 AND balance >= 10.0
			`, tenant1)
			if err != nil {
				errors <- fmt.Errorf("update error: %w", err)
				return
			}

			// Record transaction
			_, err = tx.ExecContext(ctx, `
				INSERT INTO accounting_transactions (
					tenant_id, transaction_type, amount, currency, description
				) VALUES ($1, 'DEDUCT', 10.0, 'USD', $2)
			`, tenant1, fmt.Sprintf("Transaction %d", idx))
			if err != nil {
				errors <- fmt.Errorf("insert error: %w", err)
				return
			}

			if err := tx.Commit(); err != nil {
				errors <- fmt.Errorf("commit error: %w", err)
			}
		}(i)
	}

	// Tenant 2: Concurrent transactions
	for i := 0; i < numTransactions; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			sqlDB, _ := db.DB()
			tx, err := sqlDB.BeginTx(ctx, nil)
			if err != nil {
				errors <- fmt.Errorf("begin tx error: %w", err)
				return
			}
			defer tx.Rollback()

			// Deduct balance
			_, err = tx.ExecContext(ctx, `
				UPDATE tenants SET balance = balance - 15.0 
				WHERE id = $1 AND balance >= 15.0
			`, tenant2)
			if err != nil {
				errors <- fmt.Errorf("update error: %w", err)
				return
			}

			// Record transaction
			_, err = tx.ExecContext(ctx, `
				INSERT INTO accounting_transactions (
					tenant_id, transaction_type, amount, currency, description
				) VALUES ($1, 'DEDUCT', 15.0, 'USD', $2)
			`, tenant2, fmt.Sprintf("Transaction %d", idx))
			if err != nil {
				errors <- fmt.Errorf("insert error: %w", err)
				return
			}

			if err := tx.Commit(); err != nil {
				errors <- fmt.Errorf("commit error: %w", err)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	var errorList []error
	for err := range errors {
		errorList = append(errorList, err)
	}
	assert.Empty(t, errorList, "Should have no transaction errors")

	// Verify final state
	sqlDB, _ = db.DB()
	var balance1, balance2 float64
	err = sqlDB.QueryRowContext(ctx, `SELECT balance FROM tenants WHERE id = $1`, tenant1).Scan(&balance1)
	require.NoError(t, err)
	err = sqlDB.QueryRowContext(ctx, `SELECT balance FROM tenants WHERE id = $1`, tenant2).Scan(&balance2)
	require.NoError(t, err)

	assert.Equal(t, 5000.0-(float64(numTransactions)*10.0), balance1, "Tenant 1 balance should be correct")
	assert.Equal(t, 5000.0-(float64(numTransactions)*15.0), balance2, "Tenant 2 balance should be correct")

	// Verify transaction counts
	var count1, count2 int
	err = sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM accounting_transactions WHERE tenant_id = $1`, tenant1).Scan(&count1)
	require.NoError(t, err)
	err = sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM accounting_transactions WHERE tenant_id = $1`, tenant2).Scan(&count2)
	require.NoError(t, err)

	assert.Equal(t, numTransactions, count1, "Tenant 1 should have correct transaction count")
	assert.Equal(t, numTransactions, count2, "Tenant 2 should have correct transaction count")
}

// Helper functions

func performTenantOperation(ctx context.Context, db *sql.DB, tenantID uuid.UUID, opNum int) error {
	// Insert a tariff
	_, err := db.ExecContext(ctx, `
		INSERT INTO tariffs (
			tenant_id, service_id, country_code, network_operator,
			cost_per_message, currency, effective_from, is_active
		) VALUES ($1, 1, 'YE', $2, 10.0, 'YER', CURRENT_DATE, true)
	`, tenantID, fmt.Sprintf("Op-%d", opNum))

	return err
}

func verifyTenantDataIntegrity(t *testing.T, db *sql.DB, tenantID uuid.UUID, expectedCount int) {
	ctx := context.Background()

	var count int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM tariffs WHERE tenant_id = $1
	`, tenantID).Scan(&count)

	require.NoError(t, err)
	assert.Equal(t, expectedCount, count,
		"Tenant %s should have exactly %d records", tenantID, expectedCount)
}

func createTestTenant(t *testing.T, db *gorm.DB, name string) uuid.UUID {
	tenantID := uuid.New()
	err := db.Exec("INSERT INTO tenants (id, business_name, contact_name, email, status) VALUES (?, ?, 'Test Contact', ?, ?)",
		tenantID, name, fmt.Sprintf("%s@example.com", name), "ACTIVE").Error
	require.NoError(t, err)
	return tenantID
}

func setupTestRedis(t *testing.T) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:         "localhost:6379",
		DB:           10, // Dedicated test DB so FlushDB does not collide with other packages
		DialTimeout:  10 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		PoolSize:     100,
	})
	return client
}

func cleanupTestRedis(t *testing.T, client *redis.Client) {
	// FlushDB is contained to this package's dedicated test DB (10).
	client.FlushDB(context.Background())
	client.Close()
}
