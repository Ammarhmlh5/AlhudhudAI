package testutils

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/shopspring/decimal"
)

// TestDB - Database for testing
type TestDB struct {
	*sqlx.DB
	tenantID uuid.UUID
}

// NewTestDB - Create a new test database connection
func NewTestDB() (*TestDB, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		getEnv("TEST_DB_HOST", "localhost"),
		getEnv("TEST_DB_PORT", "5432"),
		getEnv("TEST_DB_USER", "postgres"),
		getEnv("TEST_DB_PASSWORD", "postgres"),
		getEnv("TEST_DB_NAME", "sms_gateway_test"),
		getEnv("TEST_DB_SSLMODE", "disable"),
	)

	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to test database: %w", err)
	}

	return &TestDB{DB: db}, nil
}

// getEnv - Get environment variable with default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// CreateTestTenant - Create a test tenant
func (tdb *TestDB) CreateTestTenant() uuid.UUID {
	if tdb.tenantID != uuid.Nil {
		return tdb.tenantID
	}

	tenantID := uuid.New()
	tdb.tenantID = tenantID

	query := `
		INSERT INTO tenants (id, name, email, balance, status)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO NOTHING
	`

	_, err := tdb.Exec(query, tenantID, "Test Tenant", 
		fmt.Sprintf("test-%s@alhudhud.com", tenantID.String()[:8]),
		decimal.NewFromInt(10000), "ACTIVE")
	if err != nil {
		panic(fmt.Sprintf("failed to create test tenant: %v", err))
	}

	return tenantID
}

// CreateTestUser - Create a test user
func (tdb *TestDB) CreateTestUser(tenantID uuid.UUID) uuid.UUID {
	userID := uuid.New()

	query := `
		INSERT INTO users (id, tenant_id, email, password_hash, name, role, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO NOTHING
	`

	_, err := tdb.Exec(query, userID, tenantID,
		fmt.Sprintf("user-%s@test.com", userID.String()[:8]),
		"$2a$10$dummyhash", "Test User", "admin", "ACTIVE")
	if err != nil {
		panic(fmt.Sprintf("failed to create test user: %v", err))
	}

	return userID
}

// CreateTestMessage - Create a test message
func (tdb *TestDB) CreateTestMessage(tenantID uuid.UUID) uuid.UUID {
	messageID := uuid.New()

	query := `
		INSERT INTO messages (id, message_id, tenant_id, phone_number, content, status, priority)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := tdb.Exec(query, messageID, messageID.String(), tenantID,
		"+966501234567", "Test message content", "DELIVERED", "normal")
	if err != nil {
		panic(fmt.Sprintf("failed to create test message: %v", err))
	}

	return messageID
}

// SetTenantBalance - Set tenant balance
func (tdb *TestDB) SetTenantBalance(tenantID uuid.UUID, balance decimal.Decimal) {
	query := `UPDATE tenants SET balance = $1 WHERE id = $2`
	_, err := tdb.Exec(query, balance, tenantID)
	if err != nil {
		panic(fmt.Sprintf("failed to set tenant balance: %v", err))
	}
}

// Cleanup - Clean up test data
func (tdb *TestDB) Cleanup() {
	if tdb.tenantID != uuid.Nil {
		// Clean up in correct order due to foreign keys
		tdb.Exec(`DELETE FROM messages WHERE tenant_id = $1`, tdb.tenantID)
		tdb.Exec(`DELETE FROM balance_history WHERE tenant_id = $1`, tdb.tenantID)
		tdb.Exec(`DELETE FROM users WHERE tenant_id = $1`, tdb.tenantID)
		tdb.Exec(`DELETE FROM tenants WHERE id = $1`, tdb.tenantID)
	}
}

// SeedTestData - Seed comprehensive test data
func (tdb *TestDB) SeedTestData() {
	tenantID := tdb.CreateTestTenant()

	// Create multiple users
	for i := 0; i < 5; i++ {
		tdb.CreateTestUser(tenantID)
	}

	// Create multiple messages
	for i := 0; i < 20; i++ {
		tdb.CreateTestMessage(tenantID)
	}

	// Create balance history
	for i := 0; i < 10; i++ {
		query := `
			INSERT INTO balance_history (id, tenant_id, type, amount, old_balance, new_balance, reason)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`
		tdb.Exec(query, uuid.New(), tenantID, "DEDUCTION",
			decimal.NewFromInt(-10), decimal.NewFromInt(1000), decimal.NewFromInt(990),
			"Test deduction")
	}
}

// MockTx - Create a mock transaction for testing
func (tdb *TestDB) MockTx() *sqlx.Tx {
	tx, err := tdb.Beginx()
	if err != nil {
		panic(fmt.Sprintf("failed to begin transaction: %v", err))
	}
	return tx
}

// WithTransaction - Execute function within transaction
func (tdb *TestDB) WithTransaction(ctx context.Context, fn func(tx *sqlx.Tx) error) error {
	tx, err := tdb.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// GetTestTenantID - Get the test tenant ID
func (tdb *TestDB) GetTestTenantID() uuid.UUID {
	return tdb.tenantID
}

// WaitForDB - Wait for database to be ready
func (tdb *TestDB) WaitForDB(maxAttempts int, delay time.Duration) error {
	for i := 0; i < maxAttempts; i++ {
		if err := tdb.Ping(); err != nil {
			time.Sleep(delay)
			continue
		}
		return nil
	}
	return fmt.Errorf("database not available after %d attempts", maxAttempts)
}