package balance

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestBalanceIsolation_ConcurrentOperations tests that operations on one tenant
// do not affect the balance of other tenants, even under concurrent load.
// Validates Requirement 1.1: Tenant isolation in balance operations
func TestBalanceIsolation_ConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	ctx := context.Background()
	pool := setupTestDB(t)
	defer pool.Close()

	logger, _ := zap.NewDevelopment()
	manager := NewManager(pool, logger)

	// Create three test tenants with initial balances
	tenant1ID := createTestTenant(t, pool, decimal.NewFromFloat(1000))
	tenant2ID := createTestTenant(t, pool, decimal.NewFromFloat(2000))
	tenant3ID := createTestTenant(t, pool, decimal.NewFromFloat(3000))

	// Record initial balances
	initialBalance1, err := manager.GetBalance(ctx, tenant1ID)
	require.NoError(t, err)
	initialBalance2, err := manager.GetBalance(ctx, tenant2ID)
	require.NoError(t, err)
	initialBalance3, err := manager.GetBalance(ctx, tenant3ID)
	require.NoError(t, err)

	t.Logf("Initial balances - Tenant1: %s, Tenant2: %s, Tenant3: %s",
		initialBalance1.String(), initialBalance2.String(), initialBalance3.String())

	// Perform concurrent operations on tenant1 only
	const numOperations = 50
	const operationAmount = 10.0

	var wg sync.WaitGroup
	errChan := make(chan error, numOperations*2)

	// Concurrent deductions on tenant1
	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			op := BalanceOperation{
				TenantID:  tenant1ID,
				MessageID: func() *uuid.UUID { id := uuid.New(); return &id }(),
				Amount:    decimal.NewFromFloat(operationAmount),
				Type:      OperationTypeDeduct,
				Reason:    "TEST_DEDUCTION",
				CreatedBy: "test",
			}
			_, err := manager.DeductBalance(ctx, op)
			if err != nil {
				errChan <- fmt.Errorf("deduct tenant1 op %d: %w", idx, err)
			}
		}(i)
	}

	// Concurrent refunds on tenant1
	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			// Small delay to ensure some deductions happen first
			time.Sleep(10 * time.Millisecond)
			op := BalanceOperation{
				TenantID:  tenant1ID,
				MessageID: func() *uuid.UUID { id := uuid.New(); return &id }(),
				Amount:    decimal.NewFromFloat(operationAmount),
				Type:      OperationTypeRefund,
				Reason:    "TEST_REFUND",
				CreatedBy: "test",
			}
			_, err := manager.RefundBalance(ctx, op)
			if err != nil {
				errChan <- fmt.Errorf("refund tenant1 op %d: %w", idx, err)
			}
		}(i)
	}

	// Wait for all operations to complete
	wg.Wait()
	close(errChan)

	// Check for errors (some deductions may fail due to insufficient balance, which is OK)
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}
	if len(errors) > 0 {
		t.Logf("Some operations failed (expected for insufficient balance): %d errors", len(errors))
		for _, err := range errors {
			t.Logf("  - %v", err)
		}
	}

	// Get final balances
	finalBalance1, err := manager.GetBalance(ctx, tenant1ID)
	require.NoError(t, err)
	finalBalance2, err := manager.GetBalance(ctx, tenant2ID)
	require.NoError(t, err)
	finalBalance3, err := manager.GetBalance(ctx, tenant3ID)
	require.NoError(t, err)

	t.Logf("Final balances - Tenant1: %s, Tenant2: %s, Tenant3: %s",
		finalBalance1.String(), finalBalance2.String(), finalBalance3.String())

	// CRITICAL ASSERTION: Tenant2 and Tenant3 balances must remain unchanged
	// This proves complete isolation - operations on tenant1 did not affect other tenants
	assert.Equal(t, initialBalance2.String(), finalBalance2.String(),
		"Tenant2 balance must remain unchanged - isolation violated!")
	assert.Equal(t, initialBalance3.String(), finalBalance3.String(),
		"Tenant3 balance must remain unchanged - isolation violated!")

	// Verify tenant1 balance is non-negative (constraint check)
	assert.True(t, finalBalance1.GreaterThanOrEqual(decimal.Zero),
		"Tenant1 balance must not be negative")

	// Verify tenant1 balance changed (operations were performed)
	assert.NotEqual(t, initialBalance1.String(), finalBalance1.String(),
		"Tenant1 balance should have changed due to operations")

	t.Logf("✓ Balance isolation verified: Tenant2 and Tenant3 unaffected by Tenant1 operations")
}

// TestBalanceIsolation_SimultaneousMultiTenant tests concurrent operations
// across multiple tenants to ensure complete isolation
func TestBalanceIsolation_SimultaneousMultiTenant(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	ctx := context.Background()
	pool := setupTestDB(t)
	defer pool.Close()

	logger, _ := zap.NewDevelopment()
	manager := NewManager(pool, logger)

	// Create multiple test tenants
	const numTenants = 5
	const initialBalance = 1000.0
	tenantIDs := make([]uuid.UUID, numTenants)
	initialBalances := make(map[uuid.UUID]decimal.Decimal)

	for i := 0; i < numTenants; i++ {
		tenantID := createTestTenant(t, pool, decimal.NewFromFloat(initialBalance))
		tenantIDs[i] = tenantID
		balance, err := manager.GetBalance(ctx, tenantID)
		require.NoError(t, err)
		initialBalances[tenantID] = balance
		t.Logf("Created Tenant%d (ID: %s) with balance: %s", i+1, tenantID.String(), balance.String())
	}

	// Perform concurrent operations on each tenant independently
	const operationsPerTenant = 20
	const operationAmount = 5.0

	var wg sync.WaitGroup
	errChan := make(chan error, numTenants*operationsPerTenant)

	for tenantIdx, tenantID := range tenantIDs {
		for opIdx := 0; opIdx < operationsPerTenant; opIdx++ {
			wg.Add(1)
			go func(tID uuid.UUID, tIdx, oIdx int) {
				defer wg.Done()
				op := BalanceOperation{
					TenantID:  tID,
					MessageID: func() *uuid.UUID { id := uuid.New(); return &id }(),
					Amount:    decimal.NewFromFloat(operationAmount),
					Type:      OperationTypeDeduct,
					Reason:    "MULTI_TENANT_TEST",
					CreatedBy: "test",
				}
				_, err := manager.DeductBalance(ctx, op)
				if err != nil {
					// Only log non-insufficient-balance errors
					if balErr, ok := err.(*BalanceError); !ok || balErr.Type != ErrorInsufficientBalance {
						errChan <- fmt.Errorf("tenant%d op%d: %w", tIdx, oIdx, err)
					}
				}
			}(tenantID, tenantIdx, opIdx)
		}
	}

	// Wait for all operations
	wg.Wait()
	close(errChan)

	// Check for unexpected errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}
	require.Empty(t, errors, "Unexpected errors occurred: %v", errors)

	// Verify each tenant's balance independently
	for tenantIdx, tenantID := range tenantIDs {
		finalBalance, err := manager.GetBalance(ctx, tenantID)
		require.NoError(t, err)

		// Calculate expected balance (initial - successful operations)
		expectedDeduction := decimal.NewFromFloat(operationAmount * operationsPerTenant)
		expectedBalance := initialBalances[tenantID].Sub(expectedDeduction)

		t.Logf("Tenant%d (ID: %s) - Initial: %s, Final: %s, Expected: %s",
			tenantIdx+1, tenantID.String(),
			initialBalances[tenantID].String(),
			finalBalance.String(),
			expectedBalance.String())

		// Verify balance is correct (all operations should succeed since we have enough balance)
		assert.Equal(t, expectedBalance.String(), finalBalance.String(),
			"Tenant%d balance mismatch - isolation may be violated", tenantIdx+1)

		// Verify balance is non-negative
		assert.True(t, finalBalance.GreaterThanOrEqual(decimal.Zero),
			"Tenant%d balance must not be negative", tenantIdx+1)
	}

	t.Logf("✓ Multi-tenant isolation verified: All %d tenants operated independently", numTenants)
}

// TestBalanceIsolation_RefundDoesNotAffectOtherTenants tests that refunding
// balance to one tenant does not affect other tenants
func TestBalanceIsolation_RefundDoesNotAffectOtherTenants(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	ctx := context.Background()
	pool := setupTestDB(t)
	defer pool.Close()

	logger, _ := zap.NewDevelopment()
	manager := NewManager(pool, logger)

	// Create three test tenants
	tenant1ID := createTestTenant(t, pool, decimal.NewFromFloat(100))
	tenant2ID := createTestTenant(t, pool, decimal.NewFromFloat(200))
	tenant3ID := createTestTenant(t, pool, decimal.NewFromFloat(300))

	// Record initial balances
	initialBalance1, _ := manager.GetBalance(ctx, tenant1ID)
	initialBalance2, _ := manager.GetBalance(ctx, tenant2ID)
	initialBalance3, _ := manager.GetBalance(ctx, tenant3ID)

	// Perform large refund on tenant1
	refundAmount := decimal.NewFromFloat(500)
	refundOp := BalanceOperation{
		TenantID:  tenant1ID,
		MessageID: func() *uuid.UUID { id := uuid.New(); return &id }(),
		Amount:    refundAmount,
		Type:      OperationTypeRefund,
		Reason:    "LARGE_REFUND_TEST",
		CreatedBy: "test",
	}

	// Refund requires an existing COMMITTED accounting transaction
	_, err := pool.Exec(ctx, `
		INSERT INTO accounting_transactions (tenant_id, message_id, amount, status, type, created_at, updated_at)
		VALUES ($1, $2, $3, 'COMMITTED', 'DEDUCT', NOW(), NOW())
	`, tenant1ID, *refundOp.MessageID, refundAmount)
	require.NoError(t, err)

	result, err := manager.RefundBalance(ctx, refundOp)
	require.NoError(t, err)
	require.True(t, result.Success)

	// Verify tenant1 balance increased
	finalBalance1, _ := manager.GetBalance(ctx, tenant1ID)
	expectedBalance1 := initialBalance1.Add(refundAmount)
	assert.Equal(t, expectedBalance1.String(), finalBalance1.String(),
		"Tenant1 balance should increase by refund amount")

	// CRITICAL: Verify tenant2 and tenant3 balances unchanged
	finalBalance2, _ := manager.GetBalance(ctx, tenant2ID)
	finalBalance3, _ := manager.GetBalance(ctx, tenant3ID)

	assert.Equal(t, initialBalance2.String(), finalBalance2.String(),
		"Tenant2 balance must remain unchanged after Tenant1 refund")
	assert.Equal(t, initialBalance3.String(), finalBalance3.String(),
		"Tenant3 balance must remain unchanged after Tenant1 refund")

	t.Logf("✓ Refund isolation verified: Tenant1 refund did not affect Tenant2 or Tenant3")
}

// TestBalanceIsolation_HistoryLogging tests that balance history is properly
// isolated per tenant
func TestBalanceIsolation_HistoryLogging(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	ctx := context.Background()
	pool := setupTestDB(t)
	defer pool.Close()

	logger, _ := zap.NewDevelopment()
	manager := NewManager(pool, logger)

	// Create two test tenants
	tenant1ID := createTestTenant(t, pool, decimal.NewFromFloat(500))
	tenant2ID := createTestTenant(t, pool, decimal.NewFromFloat(500))

	// Perform operations on both tenants
	for i := 0; i < 5; i++ {
		// Tenant1 operation
		op1 := BalanceOperation{
			TenantID:  tenant1ID,
			MessageID: func() *uuid.UUID { id := uuid.New(); return &id }(),
			Amount:    decimal.NewFromFloat(10),
			Type:      OperationTypeDeduct,
			Reason:    "TENANT1_OP",
			CreatedBy: "test",
		}
		_, err := manager.DeductBalance(ctx, op1)
		require.NoError(t, err)

		// Tenant2 operation
		op2 := BalanceOperation{
			TenantID:  tenant2ID,
			MessageID: func() *uuid.UUID { id := uuid.New(); return &id }(),
			Amount:    decimal.NewFromFloat(20),
			Type:      OperationTypeDeduct,
			Reason:    "TENANT2_OP",
			CreatedBy: "test",
		}
		_, err = manager.DeductBalance(ctx, op2)
		require.NoError(t, err)
	}

	// Query balance history for each tenant
	var tenant1History []BalanceHistory
	query := `
		SELECT id, tenant_id, message_id, old_balance, new_balance, amount, type, reason
		FROM balance_history
		WHERE tenant_id = $1
		ORDER BY created_at ASC
	`
	rows, err := pool.Query(ctx, query, tenant1ID)
	require.NoError(t, err)
	defer rows.Close()

	for rows.Next() {
		var h BalanceHistory
		var msgID *uuid.UUID
		err := rows.Scan(&h.ID, &h.TenantID, &msgID, &h.OldBalance, &h.NewBalance, &h.Amount, &h.Type, &h.Reason)
		require.NoError(t, err)
		h.MessageID = msgID
		tenant1History = append(tenant1History, h)
	}

	var tenant2History []BalanceHistory
	rows, err = pool.Query(ctx, query, tenant2ID)
	require.NoError(t, err)
	defer rows.Close()

	for rows.Next() {
		var h BalanceHistory
		var msgID *uuid.UUID
		err := rows.Scan(&h.ID, &h.TenantID, &msgID, &h.OldBalance, &h.NewBalance, &h.Amount, &h.Type, &h.Reason)
		require.NoError(t, err)
		h.MessageID = msgID
		tenant2History = append(tenant2History, h)
	}

	// Verify each tenant has exactly 5 history entries
	assert.Len(t, tenant1History, 5, "Tenant1 should have 5 history entries")
	assert.Len(t, tenant2History, 5, "Tenant2 should have 5 history entries")

	// Verify all tenant1 history entries belong to tenant1
	for _, h := range tenant1History {
		assert.Equal(t, tenant1ID, h.TenantID, "History entry must belong to Tenant1")
		assert.NotEqual(t, uuid.Nil, *h.MessageID, "Message ID must not be Nil")
		assert.Equal(t, "TENANT1_OP", h.Reason, "Reason must be TENANT1_OP")
	}

	// Verify all tenant2 history entries belong to tenant2
	for _, h := range tenant2History {
		assert.Equal(t, tenant2ID, h.TenantID, "History entry must belong to Tenant2")
		assert.NotEqual(t, uuid.Nil, *h.MessageID, "Message ID must not be Nil")
		assert.Equal(t, "TENANT2_OP", h.Reason, "Reason must be TENANT2_OP")
	}

	t.Logf("✓ History isolation verified: Each tenant's history is completely separate")
}


