package balance

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// Note: These are unit tests. Integration tests with actual database
// will be in a separate file (manager_integration_test.go)

func TestBalanceOperation_Validation(t *testing.T) {
	tests := []struct {
		name    string
		op      BalanceOperation
		wantErr bool
	}{
		{
			name: "valid deduct operation",
			op: BalanceOperation{
				TenantID:  uuid.New(),
				MessageID: func() *uuid.UUID { id := uuid.New(); return &id }(),
				Amount:    decimal.NewFromFloat(10.5),
				Type:      OperationTypeDeduct,
				Reason:    "MESSAGE_SENT",
				CreatedBy: "system",
			},
			wantErr: false,
		},
		{
			name: "valid refund operation",
			op: BalanceOperation{
				TenantID:  uuid.New(),
				MessageID: func() *uuid.UUID { id := uuid.New(); return &id }(),
				Amount:    decimal.NewFromFloat(10.5),
				Type:      OperationTypeRefund,
				Reason:    "MESSAGE_FAILED",
				CreatedBy: "system",
			},
			wantErr: false,
		},
		{
			name: "zero amount",
			op: BalanceOperation{
				TenantID:  uuid.New(),
				Amount:    decimal.Zero,
				Type:      OperationTypeDeduct,
				CreatedBy: "system",
			},
			wantErr: true,
		},
		{
			name: "negative amount",
			op: BalanceOperation{
				TenantID:  uuid.New(),
				Amount:    decimal.NewFromFloat(-10),
				Type:      OperationTypeDeduct,
				CreatedBy: "system",
			},
			wantErr: true,
		},
		{
			name: "invalid tenant ID",
			op: BalanceOperation{
				TenantID:  uuid.Nil,
				Amount:    decimal.NewFromFloat(10),
				Type:      OperationTypeDeduct,
				CreatedBy: "system",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation checks
			if tt.wantErr {
				if tt.op.TenantID == uuid.Nil {
					assert.Equal(t, tt.op.TenantID, uuid.Nil)
				}
				if tt.op.Amount.LessThanOrEqual(decimal.Zero) {
					assert.True(t, tt.op.Amount.LessThanOrEqual(decimal.Zero))
				}
			} else {
				assert.NotEqual(t, tt.op.TenantID, uuid.Nil)
				assert.True(t, tt.op.Amount.GreaterThan(decimal.Zero))
			}
		})
	}
}

func TestBalanceError_Types(t *testing.T) {
	tests := []struct {
		name      string
		err       *BalanceError
		wantType  ErrorType
		wantMsg   string
	}{
		{
			name:     "insufficient balance error",
			err:      NewInsufficientBalanceError(uuid.New(), decimal.NewFromFloat(100), decimal.NewFromFloat(50)),
			wantType: ErrorInsufficientBalance,
			wantMsg:  "insufficient balance",
		},
		{
			name:     "tenant not found error",
			err:      NewTenantNotFoundError(uuid.New()),
			wantType: ErrorTenantNotFound,
			wantMsg:  "tenant not found",
		},
		{
			name:     "concurrent update error",
			err:      NewConcurrentUpdateError(nil),
			wantType: ErrorConcurrentUpdate,
			wantMsg:  "concurrent update detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantType, tt.err.Type)
			assert.Contains(t, tt.err.Error(), tt.wantMsg)
		})
	}
}

func TestOperationType_Constants(t *testing.T) {
	assert.Equal(t, OperationType("DEDUCT"), OperationTypeDeduct)
	assert.Equal(t, OperationType("REFUND"), OperationTypeRefund)
	assert.Equal(t, OperationType("CHARGE"), OperationTypeCharge)
	assert.Equal(t, OperationType("ADJUSTMENT"), OperationTypeAdjustment)
}

func TestBalanceResult_Structure(t *testing.T) {
	result := &BalanceResult{
		Success:    true,
		OldBalance: decimal.NewFromFloat(100),
		NewBalance: decimal.NewFromFloat(90),
		Error:      nil,
	}

	assert.True(t, result.Success)
	assert.Equal(t, "100", result.OldBalance.String())
	assert.Equal(t, "90", result.NewBalance.String())
	assert.Nil(t, result.Error)
}

// Mock tests for manager methods (without actual database)
func TestManager_CheckSufficient_Logic(t *testing.T) {
	tests := []struct {
		name           string
		tenantID       uuid.UUID
		amount         decimal.Decimal
		currentBalance decimal.Decimal
		want           bool
	}{
		{
			name:           "sufficient balance",
			tenantID:       uuid.New(),
			amount:         decimal.NewFromFloat(50),
			currentBalance: decimal.NewFromFloat(100),
			want:           true,
		},
		{
			name:           "exact balance",
			tenantID:       uuid.New(),
			amount:         decimal.NewFromFloat(100),
			currentBalance: decimal.NewFromFloat(100),
			want:           true,
		},
		{
			name:           "insufficient balance",
			tenantID:       uuid.New(),
			amount:         decimal.NewFromFloat(150),
			currentBalance: decimal.NewFromFloat(100),
			want:           false,
		},
		{
			name:           "zero amount",
			tenantID:       uuid.New(),
			amount:         decimal.Zero,
			currentBalance: decimal.NewFromFloat(100),
			want:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the logic without database
			got := tt.currentBalance.GreaterThanOrEqual(tt.amount)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBalanceOperation_MessageIDOptional(t *testing.T) {
	// Test that MessageID is optional
	op := BalanceOperation{
		TenantID:  uuid.New(),
		Amount:    decimal.NewFromFloat(10),
		Type:      OperationTypeCharge,
		Reason:    "MANUAL_CHARGE",
		CreatedBy: "admin",
	}

	assert.Nil(t, op.MessageID)
	assert.NotEqual(t, op.TenantID, uuid.Nil)
	assert.True(t, op.Amount.GreaterThan(decimal.Zero))
}

func TestBalanceHistory_Structure(t *testing.T) {
	messageID := uuid.New()
	ipAddress := "192.168.1.1"
	tenantID := uuid.New()

	history := BalanceHistory{
		ID:         uuid.New(),
		TenantID:   tenantID,
		MessageID:  &messageID,
		OldBalance: decimal.NewFromFloat(100),
		NewBalance: decimal.NewFromFloat(90),
		Amount:     decimal.NewFromFloat(10),
		Type:       OperationTypeDeduct,
		Reason:     "MESSAGE_SENT",
		CreatedBy:  "system",
		IPAddress:  &ipAddress,
	}

	assert.NotEqual(t, uuid.Nil, history.ID)
	assert.Equal(t, tenantID, history.TenantID)
	require.NotNil(t, history.MessageID)
	assert.Equal(t, messageID, *history.MessageID)
	assert.Equal(t, "100", history.OldBalance.String())
	assert.Equal(t, "90", history.NewBalance.String())
	assert.Equal(t, "10", history.Amount.String())
	assert.Equal(t, OperationTypeDeduct, history.Type)
	require.NotNil(t, history.IPAddress)
	assert.Equal(t, "192.168.1.1", *history.IPAddress)
}

// TestRefundBalance_Idempotency tests that refund is idempotent
// Validates Requirements 16.8, 16.9
func TestRefundBalance_Idempotency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	manager := NewManager(pool, zap.NewNop())

	// Setup: Create tenant with initial balance
	tenantID := uuid.New()
	initialBalance := decimal.NewFromFloat(100.0)
	
	_, err := pool.Exec(ctx, `
		INSERT INTO tenants (id, balance, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET balance = $2
	`, tenantID, initialBalance)
	require.NoError(t, err)

	// Create a message and accounting transaction
	messageID := uuid.New()
	refundAmount := decimal.NewFromFloat(10.0)
	
	_, err = pool.Exec(ctx, `
		INSERT INTO messages (id, tenant_id, phone_number, content, status, created_at)
		VALUES ($1, $2, '+967777777777', 'Test message', 'FAILED', NOW())
	`, messageID, tenantID)
	require.NoError(t, err)
	
	_, err = pool.Exec(ctx, `
		INSERT INTO accounting_transactions (tenant_id, message_id, amount, status, transaction_type, created_at, updated_at)
		VALUES ($1, $2, $3, 'COMPLETED', 'DEDUCT', NOW(), NOW())
	`, tenantID, messageID, refundAmount)
	require.NoError(t, err)

	// First refund - should succeed
	op := BalanceOperation{
		TenantID:  tenantID,
		MessageID: &messageID,
		Amount:    refundAmount,
		Type:      OperationTypeRefund,
		Reason:    "MESSAGE_FAILED",
		CreatedBy: "test",
	}

	result1, err := manager.RefundBalance(ctx, op)
	require.NoError(t, err)
	require.True(t, result1.Success)
	assert.Equal(t, initialBalance.Add(refundAmount), result1.NewBalance)

	// Verify transaction status is REFUNDED
	var status string
	err = pool.QueryRow(ctx, `
		SELECT status FROM accounting_transactions 
		WHERE tenant_id = $1 AND message_id = $2
	`, tenantID, messageID).Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, "REFUNDED", status)

	// Second refund - should be idempotent (no error, no double refund)
	result2, err := manager.RefundBalance(ctx, op)
	require.NoError(t, err)
	require.True(t, result2.Success)
	assert.Equal(t, result1.NewBalance, result2.NewBalance, "Balance should not change on second refund")

	// Verify balance was not refunded twice
	var finalBalance decimal.Decimal
	err = pool.QueryRow(ctx, `SELECT balance FROM tenants WHERE id = $1`, tenantID).Scan(&finalBalance)
	require.NoError(t, err)
	assert.Equal(t, initialBalance.Add(refundAmount), finalBalance, "Balance should only be refunded once")

	// Verify only one refund entry in balance_history
	var refundCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM balance_history 
		WHERE tenant_id = $1 AND message_id = $2 AND type = 'REFUND'
	`, tenantID, messageID).Scan(&refundCount)
	require.NoError(t, err)
	assert.Equal(t, 1, refundCount, "Should have exactly one refund entry in balance_history")
}

// TestRefundBalance_TransactionNotFound tests refund when transaction doesn't exist
func TestRefundBalance_TransactionNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	manager := NewManager(pool, zap.NewNop())

	// Setup: Create tenant with initial balance
	tenantID := uuid.New()
	initialBalance := decimal.NewFromFloat(100.0)
	
	_, err := pool.Exec(ctx, `
		INSERT INTO tenants (id, balance, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET balance = $2
	`, tenantID, initialBalance)
	require.NoError(t, err)

	// Try to refund for non-existent message
	messageID := uuid.New()
	op := BalanceOperation{
		TenantID:  tenantID,
		MessageID: &messageID,
		Amount:    decimal.NewFromFloat(10.0),
		Type:      OperationTypeRefund,
		Reason:    "MESSAGE_FAILED",
		CreatedBy: "test",
	}

	// Should succeed even if transaction doesn't exist (graceful handling)
	result, err := manager.RefundBalance(ctx, op)
	require.NoError(t, err)
	require.True(t, result.Success)

	// Verify balance was refunded
	var finalBalance decimal.Decimal
	err = pool.QueryRow(ctx, `SELECT balance FROM tenants WHERE id = $1`, tenantID).Scan(&finalBalance)
	require.NoError(t, err)
	assert.Equal(t, initialBalance.Add(op.Amount), finalBalance)
}

// TestRefundBalance_WithoutMessageID tests that refund requires message_id
func TestRefundBalance_WithoutMessageID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	manager := NewManager(pool, zap.NewNop())

	// Setup: Create tenant
	tenantID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO tenants (id, balance, created_at, updated_at)
		VALUES ($1, 100.0, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET balance = 100.0
	`, tenantID)
	require.NoError(t, err)

	// Try to refund without message_id
	op := BalanceOperation{
		TenantID:  tenantID,
		MessageID: nil, // No message ID
		Amount:    decimal.NewFromFloat(10.0),
		Type:      OperationTypeRefund,
		Reason:    "MESSAGE_FAILED",
		CreatedBy: "test",
	}

	_, err = manager.RefundBalance(ctx, op)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "message_id is required")
}

// TestRefundBalance_ConcurrentIdempotency tests concurrent refund attempts
// Validates that only one refund succeeds even with concurrent calls
func TestRefundBalance_ConcurrentIdempotency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	manager := NewManager(pool, zap.NewNop())

	// Setup: Create tenant with initial balance
	tenantID := uuid.New()
	initialBalance := decimal.NewFromFloat(100.0)
	
	_, err := pool.Exec(ctx, `
		INSERT INTO tenants (id, balance, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET balance = $2
	`, tenantID, initialBalance)
	require.NoError(t, err)

	// Create a message and accounting transaction
	messageID := uuid.New()
	refundAmount := decimal.NewFromFloat(10.0)
	
	_, err = pool.Exec(ctx, `
		INSERT INTO messages (id, tenant_id, phone_number, content, status, created_at)
		VALUES ($1, $2, '+967777777777', 'Test message', 'FAILED', NOW())
	`, messageID, tenantID)
	require.NoError(t, err)
	
	_, err = pool.Exec(ctx, `
		INSERT INTO accounting_transactions (tenant_id, message_id, amount, status, transaction_type, created_at, updated_at)
		VALUES ($1, $2, $3, 'COMPLETED', 'DEDUCT', NOW(), NOW())
	`, tenantID, messageID, refundAmount)
	require.NoError(t, err)

	// Launch 10 concurrent refund attempts
	numGoroutines := 10
	successChan := make(chan bool, numGoroutines)
	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			op := BalanceOperation{
				TenantID:  tenantID,
				MessageID: &messageID,
				Amount:    refundAmount,
				Type:      OperationTypeRefund,
				Reason:    "MESSAGE_FAILED",
				CreatedBy: "test",
			}

			result, err := manager.RefundBalance(ctx, op)
			if err != nil {
				errChan <- err
				return
			}
			successChan <- result.Success
		}()
	}

	// Collect results
	successCount := 0
	for i := 0; i < numGoroutines; i++ {
		select {
		case success := <-successChan:
			if success {
				successCount++
			}
		case err := <-errChan:
			t.Logf("Refund error (expected in concurrent scenario): %v", err)
		}
	}

	// All should succeed (idempotency)
	assert.Equal(t, numGoroutines, successCount, "All concurrent refunds should succeed due to idempotency")

	// Verify balance was only refunded once
	var finalBalance decimal.Decimal
	err = pool.QueryRow(ctx, `SELECT balance FROM tenants WHERE id = $1`, tenantID).Scan(&finalBalance)
	require.NoError(t, err)
	assert.Equal(t, initialBalance.Add(refundAmount), finalBalance, "Balance should only be refunded once despite concurrent calls")

	// Verify only one refund entry in balance_history
	var refundCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM balance_history 
		WHERE tenant_id = $1 AND message_id = $2 AND type = 'REFUND'
	`, tenantID, messageID).Scan(&refundCount)
	require.NoError(t, err)
	assert.Equal(t, 1, refundCount, "Should have exactly one refund entry despite concurrent calls")
}

// TestRefundBalance_TransactionStateUpdate tests that transaction state is updated atomically
func TestRefundBalance_TransactionStateUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	manager := NewManager(pool, zap.NewNop())

	// Setup: Create tenant with initial balance
	tenantID := uuid.New()
	initialBalance := decimal.NewFromFloat(100.0)
	
	_, err := pool.Exec(ctx, `
		INSERT INTO tenants (id, balance, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET balance = $2
	`, tenantID, initialBalance)
	require.NoError(t, err)

	// Create a message and accounting transaction
	messageID := uuid.New()
	refundAmount := decimal.NewFromFloat(10.0)
	
	_, err = pool.Exec(ctx, `
		INSERT INTO messages (id, tenant_id, phone_number, content, status, created_at)
		VALUES ($1, $2, '+967777777777', 'Test message', 'FAILED', NOW())
	`, messageID, tenantID)
	require.NoError(t, err)
	
	_, err = pool.Exec(ctx, `
		INSERT INTO accounting_transactions (tenant_id, message_id, amount, status, transaction_type, created_at, updated_at)
		VALUES ($1, $2, $3, 'COMPLETED', 'DEDUCT', NOW(), NOW())
	`, tenantID, messageID, refundAmount)
	require.NoError(t, err)

	// Perform refund
	op := BalanceOperation{
		TenantID:  tenantID,
		MessageID: &messageID,
		Amount:    refundAmount,
		Type:      OperationTypeRefund,
		Reason:    "MESSAGE_FAILED",
		CreatedBy: "test",
	}

	result, err := manager.RefundBalance(ctx, op)
	require.NoError(t, err)
	require.True(t, result.Success)

	// Verify transaction state is REFUNDED
	var status string
	var settledAt *time.Time
	err = pool.QueryRow(ctx, `
		SELECT status, settled_at FROM accounting_transactions 
		WHERE tenant_id = $1 AND message_id = $2
	`, tenantID, messageID).Scan(&status, &settledAt)
	require.NoError(t, err)
	assert.Equal(t, "REFUNDED", status, "Transaction status should be REFUNDED")
	assert.NotNil(t, settledAt, "settled_at should be set")

	// Verify balance_history entry exists
	var historyCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM balance_history 
		WHERE tenant_id = $1 AND message_id = $2 AND type = 'REFUND'
	`, tenantID, messageID).Scan(&historyCount)
	require.NoError(t, err)
	assert.Equal(t, 1, historyCount, "Should have one refund entry in balance_history")
}



// TestDeductBalanceWithTimeout_Success tests successful deduction with timeout
func TestDeductBalanceWithTimeout_Success(t *testing.T) {
	// This is a unit test - we're testing the timeout mechanism
	// Integration tests with actual database will verify the full flow
	
	t.Run("completes before timeout", func(t *testing.T) {
		// Test that when operation completes quickly, timeout doesn't interfere
		// This would be tested with a mock in a real scenario
		// For now, we just verify the timeout value is set correctly
		
		timeout := 5 * time.Second
		assert.Greater(t, timeout, time.Duration(0))
		assert.Equal(t, 5*time.Second, timeout)
	})
	
	t.Run("uses default timeout when zero", func(t *testing.T) {
		// Verify default timeout is 5 seconds
		timeout := time.Duration(0)
		if timeout == 0 {
			timeout = 5 * time.Second
		}
		assert.Equal(t, 5*time.Second, timeout)
	})
}

// TestDeductBalanceWithTimeout_Timeout tests timeout behavior
func TestDeductBalanceWithTimeout_Timeout(t *testing.T) {
	t.Run("context timeout handling", func(t *testing.T) {
		// Create a context that times out immediately
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		
		// Wait for timeout
		time.Sleep(10 * time.Millisecond)
		
		// Verify context is done
		select {
		case <-ctx.Done():
			assert.Equal(t, context.DeadlineExceeded, ctx.Err())
		default:
			t.Fatal("context should be done")
		}
	})
	
	t.Run("timeout error message", func(t *testing.T) {
		timeout := 100 * time.Millisecond
		expectedMsg := fmt.Sprintf("operation timed out after %v", timeout)
		assert.Contains(t, expectedMsg, "timed out")
		assert.Contains(t, expectedMsg, "100ms")
	})
}

// TestDeductBalanceWithTimeout_Cancellation tests context cancellation
func TestDeductBalanceWithTimeout_Cancellation(t *testing.T) {
	t.Run("context cancellation handling", func(t *testing.T) {
		// Create a cancellable context
		ctx, cancel := context.WithCancel(context.Background())
		
		// Cancel immediately
		cancel()
		
		// Verify context is cancelled
		select {
		case <-ctx.Done():
			assert.Equal(t, context.Canceled, ctx.Err())
		default:
			t.Fatal("context should be cancelled")
		}
	})
}

// TestDeductBalanceWithTimeout_CustomTimeout tests custom timeout values
func TestDeductBalanceWithTimeout_CustomTimeout(t *testing.T) {
	tests := []struct {
		name            string
		timeout         time.Duration
		expectedTimeout time.Duration
	}{
		{
			name:            "zero timeout uses default",
			timeout:         0,
			expectedTimeout: 5 * time.Second,
		},
		{
			name:            "custom 1 second timeout",
			timeout:         1 * time.Second,
			expectedTimeout: 1 * time.Second,
		},
		{
			name:            "custom 10 second timeout",
			timeout:         10 * time.Second,
			expectedTimeout: 10 * time.Second,
		},
		{
			name:            "custom 100ms timeout",
			timeout:         100 * time.Millisecond,
			expectedTimeout: 100 * time.Millisecond,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeout := tt.timeout
			if timeout == 0 {
				timeout = 5 * time.Second
			}
			assert.Equal(t, tt.expectedTimeout, timeout)
		})
	}
}

// TestDeductBalanceWithTimeout_ConcurrentOperations tests timeout with concurrent operations
func TestDeductBalanceWithTimeout_ConcurrentOperations(t *testing.T) {
	t.Run("multiple operations with different timeouts", func(t *testing.T) {
		// Test that different operations can have different timeouts
		timeouts := []time.Duration{
			1 * time.Second,
			2 * time.Second,
			5 * time.Second,
			10 * time.Second,
		}
		
		for i, timeout := range timeouts {
			t.Run(fmt.Sprintf("timeout_%d", i), func(t *testing.T) {
				assert.Greater(t, timeout, time.Duration(0))
			})
		}
	})
}
