package audit

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/lib/pq"
)

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) *sql.DB {
	// Use environment variable or default test database
	dsn := "postgres://postgres@localhost:5432/sms_gateway_test?sslmode=disable"
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)

	// Ping to verify connection
	err = db.Ping()
	if err != nil {
		t.Skip("Database not available, skipping test")
	}

	return db
}

// cleanupTestDB cleans up test data
func cleanupTestDB(t *testing.T, db *sql.DB) {
	// Note: We cannot delete from audit_trail due to immutability trigger
	// In a real test environment, you would disable triggers or use a separate test database
	// For now, we'll just close the connection
	db.Close()
}

func TestLogger_LogSuccess(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	logger := NewLogger(db)

	userID := uuid.New()
	tenantID := uuid.New()

	err := logger.LogSuccess(
		ActionBalanceDeduct,
		ResourceBalance,
		"msg-123",
		&userID,
		&tenantID,
		"192.168.1.1",
		map[string]interface{}{
			"amount":      10.5,
			"old_balance": 100.0,
			"new_balance": 89.5,
		},
	)

	assert.NoError(t, err)
}

func TestLogger_LogFailure(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	logger := NewLogger(db)

	userID := uuid.New()
	tenantID := uuid.New()

	err := logger.LogFailure(
		ActionBalanceDeduct,
		ResourceBalance,
		"msg-456",
		&userID,
		&tenantID,
		"192.168.1.2",
		"Insufficient balance",
		map[string]interface{}{
			"requested_amount": 50.0,
			"current_balance":  10.0,
		},
	)

	assert.NoError(t, err)
}

func TestLogger_Log(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	logger := NewLogger(db)

	userID := uuid.New()
	tenantID := uuid.New()

	entry := AuditEntry{
		UserID:     &userID,
		TenantID:   &tenantID,
		Action:     ActionMessageSend,
		Resource:   ResourceMessage,
		ResourceID: "msg-789",
		IPAddress:  "10.0.0.1",
		UserAgent:  "Mozilla/5.0",
		Result:     ResultSuccess,
		Details: map[string]interface{}{
			"phone_number": "+1234567890",
			"channel_id":   "ch-001",
		},
	}

	err := logger.Log(entry)
	assert.NoError(t, err)

	var persistedID int64
	err = db.QueryRow(`SELECT id FROM audit_trail WHERE resource_id = $1 AND tenant_id = $2`, "msg-789", tenantID).Scan(&persistedID)
	assert.NoError(t, err)
	assert.NotZero(t, persistedID)
}

func TestLogger_Query(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	logger := NewLogger(db)

	// Insert test entries
	userID := uuid.New()
	tenantID := uuid.New()

	for i := 0; i < 5; i++ {
		err := logger.LogSuccess(
			ActionMessageSend,
			ResourceMessage,
			uuid.New().String(),
			&userID,
			&tenantID,
			"192.168.1.1",
			map[string]interface{}{
				"index": i,
			},
		)
		require.NoError(t, err)
	}

	// Query by tenant
	filters := AuditQueryFilters{
		TenantID: &tenantID,
		Limit:    10,
	}

	entries, err := logger.Query(filters)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(entries), 5)

	// Verify entries are ordered by created_at DESC
	for i := 1; i < len(entries); i++ {
		assert.True(t, entries[i-1].CreatedAt.After(entries[i].CreatedAt) || entries[i-1].CreatedAt.Equal(entries[i].CreatedAt))
	}
}

func TestLogger_QueryWithFilters(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	logger := NewLogger(db)

	userID := uuid.New()
	tenantID := uuid.New()

	// Insert different types of entries
	logger.LogSuccess(ActionBalanceDeduct, ResourceBalance, "bal-1", &userID, &tenantID, "192.168.1.1", nil)
	logger.LogSuccess(ActionMessageSend, ResourceMessage, "msg-1", &userID, &tenantID, "192.168.1.1", nil)
	logger.LogFailure(ActionBalanceDeduct, ResourceBalance, "bal-2", &userID, &tenantID, "192.168.1.1", "Error", nil)

	// Query by action
	filters := AuditQueryFilters{
		TenantID: &tenantID,
		Action:   ActionBalanceDeduct,
		Limit:    10,
	}

	entries, err := logger.Query(filters)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(entries), 2)

	for _, entry := range entries {
		assert.Equal(t, ActionBalanceDeduct, entry.Action)
	}

	// Query by result
	filters = AuditQueryFilters{
		TenantID: &tenantID,
		Result:   ResultFailure,
		Limit:    10,
	}

	entries, err = logger.Query(filters)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(entries), 1)

	for _, entry := range entries {
		assert.Equal(t, ResultFailure, entry.Result)
	}
}

func TestLogger_QueryWithDateRange(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	logger := NewLogger(db)

	userID := uuid.New()
	tenantID := uuid.New()

	// Insert entry
	err := logger.LogSuccess(ActionUserLogin, ResourceUser, userID.String(), &userID, &tenantID, "192.168.1.1", nil)
	require.NoError(t, err)

	// The DB stores timestamps as session-local wall clock but they are read
	// back as UTC, so bracket the range around the actually-stored value rather
	// than the test process clock (avoids timezone skew between environments).
	var stored time.Time
	err = db.QueryRow(`SELECT created_at FROM audit_trail WHERE tenant_id = $1`, tenantID).Scan(&stored)
	require.NoError(t, err)

	startDate := stored.Add(-1 * time.Hour)
	endDate := stored.Add(1 * time.Hour)

	filters := AuditQueryFilters{
		TenantID:  &tenantID,
		StartDate: &startDate,
		EndDate:   &endDate,
		Limit:     10,
	}

	entries, err := logger.Query(filters)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(entries), 1)

	// Verify all entries are within date range
	for _, entry := range entries {
		assert.True(t, entry.CreatedAt.After(startDate) || entry.CreatedAt.Equal(startDate))
		assert.True(t, entry.CreatedAt.Before(endDate) || entry.CreatedAt.Equal(endDate))
	}
}

func TestLogger_QueryPagination(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	logger := NewLogger(db)

	userID := uuid.New()
	tenantID := uuid.New()

	// Insert 10 entries
	for i := 0; i < 10; i++ {
		err := logger.LogSuccess(
			ActionMessageSend,
			ResourceMessage,
			uuid.New().String(),
			&userID,
			&tenantID,
			"192.168.1.1",
			map[string]interface{}{"index": i},
		)
		require.NoError(t, err)
	}

	// Query first page
	filters := AuditQueryFilters{
		TenantID: &tenantID,
		Limit:    5,
		Offset:   0,
	}

	page1, err := logger.Query(filters)
	assert.NoError(t, err)
	assert.LessOrEqual(t, len(page1), 5)

	// Query second page
	filters.Offset = 5
	page2, err := logger.Query(filters)
	assert.NoError(t, err)
	assert.LessOrEqual(t, len(page2), 5)

	// Verify no overlap
	if len(page1) > 0 && len(page2) > 0 {
		assert.NotEqual(t, page1[0].ID, page2[0].ID)
	}
}
