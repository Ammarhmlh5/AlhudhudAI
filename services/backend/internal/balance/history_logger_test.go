package balance

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestHistoryLogger_LogDeduction(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	logger := zap.NewNop()
	historyLogger := NewHistoryLogger(pool, logger)

	ctx := context.Background()

	// Create test tenant
	tenantID := createTestTenant(t, pool, decimal.NewFromFloat(100.0))

	// Start transaction
	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	// Test logging deduction
	messageID := uuid.New()
	entry := HistoryEntry{
		TenantID:   tenantID,
		MessageID:  &messageID,
		OldBalance: decimal.NewFromFloat(100.0),
		NewBalance: decimal.NewFromFloat(95.0),
		Amount:     decimal.NewFromFloat(5.0),
		Reason:     "MESSAGE_SENT",
		CreatedBy:  "system",
		IPAddress:  "192.168.1.1",
	}

	err = historyLogger.LogDeduction(ctx, tx, entry)
	require.NoError(t, err)

	// Commit transaction
	err = tx.Commit(ctx)
	require.NoError(t, err)

	// Verify history was logged
	history, err := historyLogger.GetHistory(ctx, tenantID, 10)
	require.NoError(t, err)
	require.Len(t, history, 1)

	h := history[0]
	assert.Equal(t, tenantID, h.TenantID)
	require.NotNil(t, h.MessageID)
	assert.Equal(t, messageID, *h.MessageID)
	assert.Equal(t, "100", h.OldBalance.String())
	assert.Equal(t, "95", h.NewBalance.String())
	assert.Equal(t, "5", h.Amount.String())
	assert.Equal(t, OperationTypeDeduct, h.Type)
	assert.Equal(t, "MESSAGE_SENT", h.Reason)
	assert.Equal(t, "system", h.CreatedBy)
	assert.Equal(t, "192.168.1.1", *h.IPAddress)
}

func TestHistoryLogger_LogRefund(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	logger := zap.NewNop()
	historyLogger := NewHistoryLogger(pool, logger)

	ctx := context.Background()

	// Create test tenant
	tenantID := createTestTenant(t, pool, decimal.NewFromFloat(95.0))

	// Start transaction
	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	// Test logging refund
	messageID := uuid.New()
	entry := HistoryEntry{
		TenantID:   tenantID,
		MessageID:  &messageID,
		OldBalance: decimal.NewFromFloat(95.0),
		NewBalance: decimal.NewFromFloat(100.0),
		Amount:     decimal.NewFromFloat(5.0),
		Reason:     "MESSAGE_FAILED",
		CreatedBy:  "system",
		IPAddress:  "192.168.1.2",
	}

	err = historyLogger.LogRefund(ctx, tx, entry)
	require.NoError(t, err)

	// Commit transaction
	err = tx.Commit(ctx)
	require.NoError(t, err)

	// Verify history was logged
	history, err := historyLogger.GetHistory(ctx, tenantID, 10)
	require.NoError(t, err)
	require.Len(t, history, 1)

	h := history[0]
	assert.Equal(t, tenantID, h.TenantID)
	require.NotNil(t, h.MessageID)
	assert.Equal(t, messageID, *h.MessageID)
	assert.Equal(t, "95", h.OldBalance.String())
	assert.Equal(t, "100", h.NewBalance.String())
	assert.Equal(t, "5", h.Amount.String())
	assert.Equal(t, OperationTypeRefund, h.Type)
	assert.Equal(t, "MESSAGE_FAILED", h.Reason)
	assert.Equal(t, "system", h.CreatedBy)
	assert.Equal(t, "192.168.1.2", *h.IPAddress)
}

func TestHistoryLogger_LogCharge(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	logger := zap.NewNop()
	historyLogger := NewHistoryLogger(pool, logger)

	ctx := context.Background()

	// Create test tenant
	tenantID := createTestTenant(t, pool, decimal.NewFromFloat(100.0))

	// Start transaction
	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	// Test logging charge
	entry := HistoryEntry{
		TenantID:   tenantID,
		OldBalance: decimal.NewFromFloat(100.0),
		NewBalance: decimal.NewFromFloat(200.0),
		Amount:     decimal.NewFromFloat(100.0),
		Reason:     "BALANCE_RECHARGE",
		CreatedBy:  "admin",
		IPAddress:  "10.0.0.1",
	}

	err = historyLogger.LogCharge(ctx, tx, entry)
	require.NoError(t, err)

	// Commit transaction
	err = tx.Commit(ctx)
	require.NoError(t, err)

	// Verify history was logged
	history, err := historyLogger.GetHistory(ctx, tenantID, 10)
	require.NoError(t, err)
	require.Len(t, history, 1)

	h := history[0]
	assert.Equal(t, tenantID, h.TenantID)
	assert.Nil(t, h.MessageID) // No message_id for charge
	assert.Equal(t, "100", h.OldBalance.String())
	assert.Equal(t, "200", h.NewBalance.String())
	assert.Equal(t, "100", h.Amount.String())
	assert.Equal(t, OperationTypeCharge, h.Type)
	assert.Equal(t, "BALANCE_RECHARGE", h.Reason)
	assert.Equal(t, "admin", h.CreatedBy)
	assert.Equal(t, "10.0.0.1", *h.IPAddress)
}

func TestHistoryLogger_LogAdjustment(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	logger := zap.NewNop()
	historyLogger := NewHistoryLogger(pool, logger)

	ctx := context.Background()

	// Create test tenant
	tenantID := createTestTenant(t, pool, decimal.NewFromFloat(100.0))

	// Start transaction
	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	// Test logging adjustment
	entry := HistoryEntry{
		TenantID:   tenantID,
		OldBalance: decimal.NewFromFloat(100.0),
		NewBalance: decimal.NewFromFloat(105.0),
		Amount:     decimal.NewFromFloat(5.0),
		Reason:     "MANUAL_CORRECTION",
		CreatedBy:  "admin",
		IPAddress:  "10.0.0.2",
	}

	err = historyLogger.LogAdjustment(ctx, tx, entry)
	require.NoError(t, err)

	// Commit transaction
	err = tx.Commit(ctx)
	require.NoError(t, err)

	// Verify history was logged
	history, err := historyLogger.GetHistory(ctx, tenantID, 10)
	require.NoError(t, err)
	require.Len(t, history, 1)

	h := history[0]
	assert.Equal(t, tenantID, h.TenantID)
	assert.Nil(t, h.MessageID)
	assert.Equal(t, "100", h.OldBalance.String())
	assert.Equal(t, "105", h.NewBalance.String())
	assert.Equal(t, "5", h.Amount.String())
	assert.Equal(t, OperationTypeAdjustment, h.Type)
	assert.Equal(t, "MANUAL_CORRECTION", h.Reason)
	assert.Equal(t, "admin", h.CreatedBy)
	assert.Equal(t, "10.0.0.2", *h.IPAddress)
}

func TestHistoryLogger_MultipleEntries(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	logger := zap.NewNop()
	historyLogger := NewHistoryLogger(pool, logger)

	ctx := context.Background()

	// Create test tenant
	tenantID := createTestTenant(t, pool, decimal.NewFromFloat(100.0))

	// Log multiple operations
	operations := []struct {
		logFunc func(context.Context, interface{}, HistoryEntry) error
		entry   HistoryEntry
	}{
		{
			logFunc: func(ctx context.Context, tx interface{}, e HistoryEntry) error {
				return historyLogger.LogDeduction(ctx, tx.(pgx.Tx), e)
			},
			entry: HistoryEntry{
				TenantID:   tenantID,
				MessageID:  func() *uuid.UUID { id := uuid.New(); return &id }(),
				OldBalance: decimal.NewFromFloat(100.0),
				NewBalance: decimal.NewFromFloat(95.0),
				Amount:     decimal.NewFromFloat(5.0),
				Reason:     "MESSAGE_SENT",
				CreatedBy:  "system",
				IPAddress:  "192.168.1.1",
			},
		},
		{
			logFunc: func(ctx context.Context, tx interface{}, e HistoryEntry) error {
				return historyLogger.LogDeduction(ctx, tx.(pgx.Tx), e)
			},
			entry: HistoryEntry{
				TenantID:   tenantID,
				MessageID:  func() *uuid.UUID { id := uuid.New(); return &id }(),
				OldBalance: decimal.NewFromFloat(95.0),
				NewBalance: decimal.NewFromFloat(90.0),
				Amount:     decimal.NewFromFloat(5.0),
				Reason:     "MESSAGE_SENT",
				CreatedBy:  "system",
				IPAddress:  "192.168.1.1",
			},
		},
		{
			logFunc: func(ctx context.Context, tx interface{}, e HistoryEntry) error {
				return historyLogger.LogRefund(ctx, tx.(pgx.Tx), e)
			},
			entry: HistoryEntry{
				TenantID:   tenantID,
				MessageID:  func() *uuid.UUID { id := uuid.New(); return &id }(),
				OldBalance: decimal.NewFromFloat(90.0),
				NewBalance: decimal.NewFromFloat(95.0),
				Amount:     decimal.NewFromFloat(5.0),
				Reason:     "MESSAGE_FAILED",
				CreatedBy:  "system",
				IPAddress:  "192.168.1.1",
			},
		},
	}

	for _, op := range operations {
		tx, err := pool.Begin(ctx)
		require.NoError(t, err)

		err = op.logFunc(ctx, tx, op.entry)
		require.NoError(t, err)

		err = tx.Commit(ctx)
		require.NoError(t, err)
	}

	// Verify all entries were logged
	history, err := historyLogger.GetHistory(ctx, tenantID, 10)
	require.NoError(t, err)
	require.Len(t, history, 3)

	// Verify order (most recent first)
	assert.Equal(t, OperationTypeRefund, history[0].Type)
	assert.Equal(t, OperationTypeDeduct, history[1].Type)
	assert.Equal(t, OperationTypeDeduct, history[2].Type)
}

func TestHistoryLogger_InvalidTenantID(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	logger := zap.NewNop()
	historyLogger := NewHistoryLogger(pool, logger)

	ctx := context.Background()

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	// Test with invalid tenant_id
	entry := HistoryEntry{
		TenantID:   uuid.Nil, // Invalid
		OldBalance: decimal.NewFromFloat(100.0),
		NewBalance: decimal.NewFromFloat(95.0),
		Amount:     decimal.NewFromFloat(5.0),
		Reason:     "TEST",
		CreatedBy:  "system",
	}

	err = historyLogger.LogDeduction(ctx, tx, entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid tenant_id")
}

func TestHistoryLogger_GetHistoryWithLimit(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	logger := zap.NewNop()
	historyLogger := NewHistoryLogger(pool, logger)

	ctx := context.Background()

	// Create test tenant
	tenantID := createTestTenant(t, pool, decimal.NewFromFloat(100.0))

	// Log 5 operations
	for i := 0; i < 5; i++ {
		tx, err := pool.Begin(ctx)
		require.NoError(t, err)

		entry := HistoryEntry{
			TenantID:   tenantID,
			MessageID:  func() *uuid.UUID { id := uuid.New(); return &id }(),
			OldBalance: decimal.NewFromFloat(100.0 - float64(i*5)),
			NewBalance: decimal.NewFromFloat(95.0 - float64(i*5)),
			Amount:     decimal.NewFromFloat(5.0),
			Reason:     "MESSAGE_SENT",
			CreatedBy:  "system",
		}

		err = historyLogger.LogDeduction(ctx, tx, entry)
		require.NoError(t, err)

		err = tx.Commit(ctx)
		require.NoError(t, err)
	}

	// Test with limit
	history, err := historyLogger.GetHistory(ctx, tenantID, 3)
	require.NoError(t, err)
	assert.Len(t, history, 3)
}

func TestHistoryLogger_IPAddressLogging(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	logger := zap.NewNop()
	historyLogger := NewHistoryLogger(pool, logger)

	ctx := context.Background()

	// Create test tenant
	tenantID := createTestTenant(t, pool, decimal.NewFromFloat(100.0))

	testCases := []struct {
		name      string
		ipAddress string
		wantIP    *string
	}{
		{
			name:      "with IP address",
			ipAddress: "203.0.113.42",
			wantIP:    stringPtr("203.0.113.42"),
		},
		{
			name:      "without IP address",
			ipAddress: "",
			wantIP:    nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tx, err := pool.Begin(ctx)
			require.NoError(t, err)
			defer tx.Rollback(ctx)

			messageID := uuid.New()
			entry := HistoryEntry{
				TenantID:   tenantID,
				MessageID:  &messageID,
				OldBalance: decimal.NewFromFloat(100.0),
				NewBalance: decimal.NewFromFloat(95.0),
				Amount:     decimal.NewFromFloat(5.0),
				Reason:     "TEST",
				CreatedBy:  "system",
				IPAddress:  tc.ipAddress,
			}

			err = historyLogger.LogDeduction(ctx, tx, entry)
			require.NoError(t, err)

			err = tx.Commit(ctx)
			require.NoError(t, err)

			// Verify IP address was logged correctly
			history, err := historyLogger.GetHistory(ctx, tenantID, 1)
			require.NoError(t, err)
			require.Len(t, history, 1)

			if tc.wantIP == nil {
				assert.Nil(t, history[0].IPAddress)
			} else {
				require.NotNil(t, history[0].IPAddress)
				assert.Equal(t, *tc.wantIP, *history[0].IPAddress)
			}

			// Clean up for next test case
			_, err = pool.Exec(ctx, "DELETE FROM balance_history WHERE tenant_id = $1", tenantID)
			require.NoError(t, err)
		})
	}
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
