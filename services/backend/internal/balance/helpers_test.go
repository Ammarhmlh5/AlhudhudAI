package balance

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *pgxpool.Pool {
	// Use environment variable or default to local test database
	connString := "postgres://postgres:postgres@localhost:5432/sms_gateway_test?sslmode=disable"
	
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		t.Skipf("Cannot connect to test database: %v", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("Cannot ping test database: %v", err)
	}

	return pool
}

func createTestTenant(t *testing.T, pool *pgxpool.Pool, initialBalance decimal.Decimal) uuid.UUID {
	ctx := context.Background()
	
	query := `
		INSERT INTO tenants (id, name, email, balance, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'ACTIVE', NOW(), NOW())
		RETURNING id
	`
	
	tenantID := uuid.New()
	var returnedID uuid.UUID
	err := pool.QueryRow(ctx, query,
		tenantID,
		fmt.Sprintf("Test Tenant %d", time.Now().UnixNano()),
		fmt.Sprintf("test%d@example.com", time.Now().UnixNano()),
		initialBalance,
	).Scan(&returnedID)
	
	require.NoError(t, err)
	return returnedID
}

func cleanupTestTenant(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) {
	ctx := context.Background()
	
	// Delete balance logger history
	_, err := pool.Exec(ctx, "DELETE FROM balance_history WHERE tenant_id = $1", tenantID)
	if err != nil {
		t.Logf("Warning: Failed to cleanup balance history: %v", err)
	}
	
	// Delete tenant
	_, err = pool.Exec(ctx, "DELETE FROM tenants WHERE id = $1", tenantID)
	if err != nil {
		t.Logf("Warning: Failed to cleanup tenant: %v", err)
	}
}

func getBalanceHistoryCount(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) int {
	ctx := context.Background()
	
	query := `SELECT COUNT(*) FROM balance_history WHERE tenant_id = $1`
	
	var count int
	err := pool.QueryRow(ctx, query, tenantID).Scan(&count)
	require.NoError(t, err, "Failed to get balance history count")
	
	return count
}
