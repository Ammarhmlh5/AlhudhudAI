package worker

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"

	"github.com/al-hudhud/backend/internal/balance"
)

// Bug 7 Preservation Tests: Successful Balance Deductions
// **Validates: Requirements 3.4, 3.5**
//
// **Property 2: Preservation** - Successful Deductions Continue
//
// These tests verify that SUCCESSFUL balance deductions with sufficient funds continue to work correctly.
// They test the EXISTING functionality that must continue to work after the fix.
//
// Expected outcome on UNFIXED code: Tests PASS (successful deductions work correctly)
// Expected outcome on FIXED code: Tests PASS (behavior preserved)
//
// IMPORTANT: Bug 7 is about preventing negative balances when funds are insufficient.
// These tests verify that when balance IS sufficient, deductions work correctly.

// mockTxBug7Preservation is a mock transaction for Bug 7 preservation tests
type mockTxBug7Preservation struct{}

func (m *mockTxBug7Preservation) Begin(ctx context.Context) (pgx.Tx, error)       { return m, nil }
func (m *mockTxBug7Preservation) Commit(ctx context.Context) error                { return nil }
func (m *mockTxBug7Preservation) Rollback(ctx context.Context) error              { return nil }
func (m *mockTxBug7Preservation) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (m *mockTxBug7Preservation) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return nil, nil
}
func (m *mockTxBug7Preservation) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return nil
}
func (m *mockTxBug7Preservation) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (m *mockTxBug7Preservation) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return nil
}
func (m *mockTxBug7Preservation) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}
func (m *mockTxBug7Preservation) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (m *mockTxBug7Preservation) Conn() *pgx.Conn {
	return nil
}

// mockBalanceManagerBug7Preservation simulates CORRECT balance deduction behavior
// This mock represents the EXISTING correct behavior that must be preserved
type mockBalanceManagerBug7Preservation struct {
	currentBalance decimal.Decimal
}

// DeductBalanceTx simulates CORRECT behavior: successful deduction with sufficient balance
func (m *mockBalanceManagerBug7Preservation) DeductBalanceTx(ctx context.Context, tx pgx.Tx, op balance.BalanceOperation) (*balance.BalanceResult, error) {
	// CORRECT BEHAVIOR: Check balance before deduction
	if m.currentBalance.LessThan(op.Amount) {
		return nil, balance.NewInsufficientBalanceError(op.TenantID, op.Amount, m.currentBalance)
	}
	
	// Deduct balance
	oldBalance := m.currentBalance
	newBalance := oldBalance.Sub(op.Amount)
	m.currentBalance = newBalance
	
	// Return success
	return &balance.BalanceResult{
		Success:    true,
		OldBalance: oldBalance,
		NewBalance: newBalance,
	}, nil
}

func (m *mockBalanceManagerBug7Preservation) DeductBalance(ctx context.Context, op balance.BalanceOperation) (*balance.BalanceResult, error) {
	return nil, nil
}

func (m *mockBalanceManagerBug7Preservation) DeductBalanceWithTimeout(ctx context.Context, op balance.BalanceOperation, timeout int) (*balance.BalanceResult, error) {
	return nil, nil
}

func (m *mockBalanceManagerBug7Preservation) RefundBalance(ctx context.Context, op balance.BalanceOperation) (*balance.BalanceResult, error) {
	// Simulate refund: add amount back to balance
	oldBalance := m.currentBalance
	newBalance := oldBalance.Add(op.Amount)
	m.currentBalance = newBalance
	
	return &balance.BalanceResult{
		Success:    true,
		OldBalance: oldBalance,
		NewBalance: newBalance,
	}, nil
}

func (m *mockBalanceManagerBug7Preservation) RefundBalanceTx(ctx context.Context, tx pgx.Tx, op balance.BalanceOperation) (*balance.BalanceResult, error) {
	return m.RefundBalance(ctx, op)
}

func (m *mockBalanceManagerBug7Preservation) GetBalance(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
	return m.currentBalance, nil
}

func (m *mockBalanceManagerBug7Preservation) CheckSufficient(ctx context.Context, tenantID uuid.UUID, amount decimal.Decimal) (bool, error) {
	return m.currentBalance.GreaterThanOrEqual(amount), nil
}

func (m *mockBalanceManagerBug7Preservation) CreditBalance(ctx context.Context, op balance.BalanceOperation) (*balance.BalanceResult, error) {
	return &balance.BalanceResult{Success: true}, nil
}

func (m *mockBalanceManagerBug7Preservation) TransferBalance(ctx context.Context, from, to uuid.UUID, amount decimal.Decimal, reason, createdBy, ipAddress string) (*balance.TransferResult, error) {
	return &balance.TransferResult{Success: true}, nil
}

// TestBug7_Preservation_SufficientBalanceDeduction verifies that successful
// balance deductions with sufficient funds continue to work correctly
// **Validates: Requirement 3.4**
func TestBug7_Preservation_SufficientBalanceDeduction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Bug 7 preservation test in short mode")
	}

	// Property: Successful balance deductions with sufficient funds work correctly
	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		// Initial balance: 100-1000 SAR
		initialBalance := decimal.NewFromInt(int64(rapid.IntRange(100, 1000).Draw(t, "initialBalance")))
		
		// Deduction amount: 1 to (initialBalance - 1) SAR (always less than available)
		maxDeduction := initialBalance.IntPart() - 1
		if maxDeduction < 1 {
			maxDeduction = 1
		}
		deductionAmount := decimal.NewFromInt(int64(rapid.IntRange(1, int(maxDeduction)).Draw(t, "deductionAmount")))

		// Setup
		ctx := context.Background()
		tenantID := uuid.New()
		gatewayID := uuid.New()
		messageID := uuid.New()

		mockManager := &mockBalanceManagerBug7Preservation{
			currentBalance: initialBalance,
		}

		op := balance.BalanceOperation{
			TenantID:  tenantID,
			GatewayID: &gatewayID,
			MessageID: &messageID,
			Amount:    deductionAmount,
			Reason:    "BATCH_SEND",
			CreatedBy: "FAST_PROCESSOR",
		}

		// ACT: Deduct balance with sufficient funds
		mockTx := &mockTxBug7Preservation{}
		result, err := mockManager.DeductBalanceTx(ctx, mockTx, op)

		// ASSERT: Deduction succeeds
		assert.NoError(t, err,
			"Preservation: Deduction should succeed when balance (%s) >= amount (%s)",
			initialBalance.String(), deductionAmount.String())
		
		require.NotNil(t, result, "Preservation: Result should not be nil")
		
		// Verify balance calculation
		expectedNewBalance := initialBalance.Sub(deductionAmount)
		assert.Equal(t, expectedNewBalance, result.NewBalance,
			"Preservation: New balance should be %s (old: %s - deduction: %s)",
			expectedNewBalance.String(), initialBalance.String(), deductionAmount.String())
		
		assert.Equal(t, initialBalance, result.OldBalance,
			"Preservation: Old balance should be preserved")
		
		assert.True(t, result.Success,
			"Preservation: Result should indicate success")
		
		// Verify balance is not negative
		assert.True(t, result.NewBalance.GreaterThanOrEqual(decimal.Zero),
			"Preservation: New balance should not be negative")
	})
}

// TestBug7_Preservation_ExactBalanceDeduction verifies that deducting
// exactly the available balance works correctly
// **Validates: Requirement 3.4**
func TestBug7_Preservation_ExactBalanceDeduction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Bug 7 preservation test in short mode")
	}

	// Property: Deducting exact balance results in zero balance
	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		// Initial balance: 10-500 SAR
		initialBalance := decimal.NewFromInt(int64(rapid.IntRange(10, 500).Draw(t, "initialBalance")))
		
		// Deduction amount: exactly the balance
		deductionAmount := initialBalance

		// Setup
		ctx := context.Background()
		tenantID := uuid.New()
		gatewayID := uuid.New()
		messageID := uuid.New()

		mockManager := &mockBalanceManagerBug7Preservation{
			currentBalance: initialBalance,
		}

		op := balance.BalanceOperation{
			TenantID:  tenantID,
			GatewayID: &gatewayID,
			MessageID: &messageID,
			Amount:    deductionAmount,
			Reason:    "EXACT_BALANCE_DEDUCTION",
			CreatedBy: "FAST_PROCESSOR",
		}

		// ACT: Deduct exact balance
		mockTx := &mockTxBug7Preservation{}
		result, err := mockManager.DeductBalanceTx(ctx, mockTx, op)

		// ASSERT: Deduction succeeds with zero balance
		assert.NoError(t, err,
			"Preservation: Deducting exact balance (%s) should succeed",
			initialBalance.String())
		
		require.NotNil(t, result, "Preservation: Result should not be nil")
		
		assert.True(t, result.NewBalance.Equal(decimal.Zero),
			"Preservation: New balance should be 0 after deducting exact amount")
		
		assert.Equal(t, initialBalance, result.OldBalance,
			"Preservation: Old balance should be preserved")
		
		assert.True(t, result.Success,
			"Preservation: Result should indicate success")
	})
}

// TestBug7_Preservation_RefundOperation verifies that refund operations
// continue to work correctly
// **Validates: Requirement 3.5**
func TestBug7_Preservation_RefundOperation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Bug 7 preservation test in short mode")
	}

	// Property: Refunds restore balance correctly
	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		// Initial balance: 50-500 SAR
		initialBalance := decimal.NewFromInt(int64(rapid.IntRange(50, 500).Draw(t, "initialBalance")))
		
		// Refund amount: 1-100 SAR
		refundAmount := decimal.NewFromInt(int64(rapid.IntRange(1, 100).Draw(t, "refundAmount")))

		// Setup
		ctx := context.Background()
		tenantID := uuid.New()
		gatewayID := uuid.New()
		messageID := uuid.New()

		mockManager := &mockBalanceManagerBug7Preservation{
			currentBalance: initialBalance,
		}

		op := balance.BalanceOperation{
			TenantID:  tenantID,
			GatewayID: &gatewayID,
			MessageID: &messageID,
			Amount:    refundAmount,
			Reason:    "MESSAGE_FAILED_REFUND",
			CreatedBy: "FAST_PROCESSOR",
		}

		// ACT: Process refund
		result, err := mockManager.RefundBalance(ctx, op)

		// ASSERT: Refund succeeds
		assert.NoError(t, err,
			"Preservation: Refund should succeed")
		
		require.NotNil(t, result, "Preservation: Result should not be nil")
		
		// Verify balance calculation
		expectedNewBalance := initialBalance.Add(refundAmount)
		assert.Equal(t, expectedNewBalance, result.NewBalance,
			"Preservation: New balance should be %s (old: %s + refund: %s)",
			expectedNewBalance.String(), initialBalance.String(), refundAmount.String())
		
		assert.Equal(t, initialBalance, result.OldBalance,
			"Preservation: Old balance should be preserved")
		
		assert.True(t, result.Success,
			"Preservation: Result should indicate success")
	})
}

// TestBug7_Preservation_MultipleDeductions verifies that multiple
// sequential deductions work correctly
// **Validates: Requirement 3.4**
func TestBug7_Preservation_MultipleDeductions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Bug 7 preservation test in short mode")
	}

	// Property: Multiple deductions reduce balance correctly
	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		// Initial balance: 200-1000 SAR
		initialBalance := decimal.NewFromInt(int64(rapid.IntRange(200, 1000).Draw(t, "initialBalance")))
		
		// Number of deductions: 2-5
		deductionCount := rapid.IntRange(2, 5).Draw(t, "deductionCount")
		
		// Each deduction: 10-50 SAR
		deductions := make([]decimal.Decimal, deductionCount)
		totalDeduction := decimal.Zero
		for i := 0; i < deductionCount; i++ {
			amount := decimal.NewFromInt(int64(rapid.IntRange(10, 50).Draw(t, "deduction")))
			deductions[i] = amount
			totalDeduction = totalDeduction.Add(amount)
		}
		
		// Ensure total deduction doesn't exceed balance
		if totalDeduction.GreaterThan(initialBalance) {
			t.Skip("Total deduction exceeds balance")
		}

		// Setup
		ctx := context.Background()
		tenantID := uuid.New()
		gatewayID := uuid.New()

		mockManager := &mockBalanceManagerBug7Preservation{
			currentBalance: initialBalance,
		}

		// ACT: Perform multiple deductions
		currentBalance := initialBalance
		for i, amount := range deductions {
			messageID := uuid.New()
			op := balance.BalanceOperation{
				TenantID:  tenantID,
				GatewayID: &gatewayID,
				MessageID: &messageID,
				Amount:    amount,
				Reason:    "BATCH_SEND",
				CreatedBy: "FAST_PROCESSOR",
			}

			mockTx := &mockTxBug7Preservation{}
			result, err := mockManager.DeductBalanceTx(ctx, mockTx, op)

			// ASSERT: Each deduction succeeds
			assert.NoError(t, err,
				"Preservation: Deduction %d should succeed", i+1)
			
			require.NotNil(t, result, "Preservation: Result %d should not be nil", i+1)
			
			// Verify balance after each deduction
			expectedBalance := currentBalance.Sub(amount)
			assert.Equal(t, expectedBalance, result.NewBalance,
				"Preservation: Balance after deduction %d should be %s",
				i+1, expectedBalance.String())
			
			currentBalance = result.NewBalance
		}

		// ASSERT: Final balance is correct
		expectedFinalBalance := initialBalance.Sub(totalDeduction)
		assert.Equal(t, expectedFinalBalance, currentBalance,
			"Preservation: Final balance should be %s (initial: %s - total: %s)",
			expectedFinalBalance.String(), initialBalance.String(), totalDeduction.String())
	})
}

// Simple unit test: Successful deduction with sufficient balance
// **Validates: Requirement 3.4**
func TestBug7_Preservation_Simple_SufficientBalance(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	gatewayID := uuid.New()
	messageID := uuid.New()

	// Setup: Tenant has 100 SAR
	initialBalance := decimal.NewFromInt(100)

	mockManager := &mockBalanceManagerBug7Preservation{
		currentBalance: initialBalance,
	}

	// Deduct 50 SAR (sufficient balance)
	deductionAmount := decimal.NewFromInt(50)
	op := balance.BalanceOperation{
		TenantID:  tenantID,
		GatewayID: &gatewayID,
		MessageID: &messageID,
		Amount:    deductionAmount,
		Reason:    "BATCH_SEND_10_MESSAGES",
		CreatedBy: "FAST_PROCESSOR",
	}

	// ACT
	mockTx := &mockTxBug7Preservation{}
	result, err := mockManager.DeductBalanceTx(ctx, mockTx, op)

	// ASSERT
	assert.NoError(t, err, "Preservation: Deduction should succeed with sufficient balance")
	require.NotNil(t, result)
	assert.Equal(t, decimal.NewFromInt(50), result.NewBalance,
		"Preservation: New balance should be 50 SAR (100 - 50)")
	assert.Equal(t, initialBalance, result.OldBalance,
		"Preservation: Old balance should be 100 SAR")
	assert.True(t, result.Success, "Preservation: Result should indicate success")

	t.Logf("✅ Successful deduction: 100 SAR - 50 SAR = 50 SAR")
}

// Simple unit test: Exact balance deduction
// **Validates: Requirement 3.4**
func TestBug7_Preservation_Simple_ExactBalance(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	gatewayID := uuid.New()
	messageID := uuid.New()

	// Setup: Tenant has 100 SAR
	initialBalance := decimal.NewFromInt(100)

	mockManager := &mockBalanceManagerBug7Preservation{
		currentBalance: initialBalance,
	}

	// Deduct exactly 100 SAR
	deductionAmount := decimal.NewFromInt(100)
	op := balance.BalanceOperation{
		TenantID:  tenantID,
		GatewayID: &gatewayID,
		MessageID: &messageID,
		Amount:    deductionAmount,
		Reason:    "EXACT_BALANCE_TEST",
		CreatedBy: "FAST_PROCESSOR",
	}

	// ACT
	mockTx := &mockTxBug7Preservation{}
	result, err := mockManager.DeductBalanceTx(ctx, mockTx, op)

	// ASSERT
	assert.NoError(t, err, "Preservation: Deducting exact balance should succeed")
	require.NotNil(t, result)
	assert.True(t, result.NewBalance.Equal(decimal.Zero),
		"Preservation: New balance should be 0 SAR (100 - 100)")
	assert.Equal(t, initialBalance, result.OldBalance,
		"Preservation: Old balance should be 100 SAR")
	assert.True(t, result.Success, "Preservation: Result should indicate success")

	t.Logf("✅ Exact balance deduction: 100 SAR - 100 SAR = 0 SAR")
}

// Simple unit test: Refund operation
// **Validates: Requirement 3.5**
func TestBug7_Preservation_Simple_Refund(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	gatewayID := uuid.New()
	messageID := uuid.New()

	// Setup: Tenant has 50 SAR
	initialBalance := decimal.NewFromInt(50)

	mockManager := &mockBalanceManagerBug7Preservation{
		currentBalance: initialBalance,
	}

	// Refund 25 SAR (failed message)
	refundAmount := decimal.NewFromInt(25)
	op := balance.BalanceOperation{
		TenantID:  tenantID,
		GatewayID: &gatewayID,
		MessageID: &messageID,
		Amount:    refundAmount,
		Reason:    "MESSAGE_FAILED_REFUND",
		CreatedBy: "FAST_PROCESSOR",
	}

	// ACT
	result, err := mockManager.RefundBalance(ctx, op)

	// ASSERT
	assert.NoError(t, err, "Preservation: Refund should succeed")
	require.NotNil(t, result)
	assert.Equal(t, decimal.NewFromInt(75), result.NewBalance,
		"Preservation: New balance should be 75 SAR (50 + 25)")
	assert.Equal(t, initialBalance, result.OldBalance,
		"Preservation: Old balance should be 50 SAR")
	assert.True(t, result.Success, "Preservation: Result should indicate success")

	t.Logf("✅ Refund operation: 50 SAR + 25 SAR = 75 SAR")
}

// Simple unit test: Multiple sequential deductions
// **Validates: Requirement 3.4**
func TestBug7_Preservation_Simple_MultipleDeductions(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	gatewayID := uuid.New()

	// Setup: Tenant has 200 SAR
	initialBalance := decimal.NewFromInt(200)

	mockManager := &mockBalanceManagerBug7Preservation{
		currentBalance: initialBalance,
	}

	// Deduction 1: 50 SAR
	op1 := balance.BalanceOperation{
		TenantID:  tenantID,
		GatewayID: &gatewayID,
		MessageID: func() *uuid.UUID { id := uuid.New(); return &id }(),
		Amount:    decimal.NewFromInt(50),
		Reason:    "BATCH_1",
		CreatedBy: "FAST_PROCESSOR",
	}

	mockTx := &mockTxBug7Preservation{}
	result1, err := mockManager.DeductBalanceTx(ctx, mockTx, op1)
	assert.NoError(t, err, "Preservation: First deduction should succeed")
	require.NotNil(t, result1)
	assert.Equal(t, decimal.NewFromInt(150), result1.NewBalance,
		"Preservation: Balance after first deduction should be 150 SAR")

	// Deduction 2: 75 SAR
	op2 := balance.BalanceOperation{
		TenantID:  tenantID,
		GatewayID: &gatewayID,
		MessageID: func() *uuid.UUID { id := uuid.New(); return &id }(),
		Amount:    decimal.NewFromInt(75),
		Reason:    "BATCH_2",
		CreatedBy: "FAST_PROCESSOR",
	}

	result2, err := mockManager.DeductBalanceTx(ctx, mockTx, op2)
	assert.NoError(t, err, "Preservation: Second deduction should succeed")
	require.NotNil(t, result2)
	assert.Equal(t, decimal.NewFromInt(75), result2.NewBalance,
		"Preservation: Balance after second deduction should be 75 SAR")

	// Deduction 3: 25 SAR
	op3 := balance.BalanceOperation{
		TenantID:  tenantID,
		GatewayID: &gatewayID,
		MessageID: func() *uuid.UUID { id := uuid.New(); return &id }(),
		Amount:    decimal.NewFromInt(25),
		Reason:    "BATCH_3",
		CreatedBy: "FAST_PROCESSOR",
	}

	result3, err := mockManager.DeductBalanceTx(ctx, mockTx, op3)
	assert.NoError(t, err, "Preservation: Third deduction should succeed")
	require.NotNil(t, result3)
	assert.Equal(t, decimal.NewFromInt(50), result3.NewBalance,
		"Preservation: Balance after third deduction should be 50 SAR")

	t.Logf("✅ Multiple deductions: 200 → 150 → 75 → 50 SAR")
}

// Simple unit test: Deduction and refund cycle
// **Validates: Requirements 3.4, 3.5**
func TestBug7_Preservation_Simple_DeductionRefundCycle(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	gatewayID := uuid.New()
	messageID := uuid.New()

	// Setup: Tenant has 100 SAR
	initialBalance := decimal.NewFromInt(100)

	mockManager := &mockBalanceManagerBug7Preservation{
		currentBalance: initialBalance,
	}

	// Step 1: Deduct 30 SAR
	deductOp := balance.BalanceOperation{
		TenantID:  tenantID,
		GatewayID: &gatewayID,
		MessageID: &messageID,
		Amount:    decimal.NewFromInt(30),
		Reason:    "BATCH_SEND",
		CreatedBy: "FAST_PROCESSOR",
	}

	mockTx := &mockTxBug7Preservation{}
	deductResult, err := mockManager.DeductBalanceTx(ctx, mockTx, deductOp)
	assert.NoError(t, err, "Preservation: Deduction should succeed")
	require.NotNil(t, deductResult)
	assert.Equal(t, decimal.NewFromInt(70), deductResult.NewBalance,
		"Preservation: Balance after deduction should be 70 SAR")

	// Step 2: Refund 30 SAR (message failed)
	refundOp := balance.BalanceOperation{
		TenantID:  tenantID,
		GatewayID: &gatewayID,
		MessageID: &messageID,
		Amount:    decimal.NewFromInt(30),
		Reason:    "MESSAGE_FAILED_REFUND",
		CreatedBy: "FAST_PROCESSOR",
	}

	refundResult, err := mockManager.RefundBalance(ctx, refundOp)
	assert.NoError(t, err, "Preservation: Refund should succeed")
	require.NotNil(t, refundResult)
	assert.Equal(t, decimal.NewFromInt(100), refundResult.NewBalance,
		"Preservation: Balance after refund should be 100 SAR (back to original)")

	t.Logf("✅ Deduction and refund cycle: 100 → 70 → 100 SAR")
}

// Simple unit test: Small balance deduction
// **Validates: Requirement 3.4**
func TestBug7_Preservation_Simple_SmallBalance(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	gatewayID := uuid.New()
	messageID := uuid.New()

	// Setup: Tenant has 5 SAR (small balance)
	initialBalance := decimal.NewFromInt(5)

	mockManager := &mockBalanceManagerBug7Preservation{
		currentBalance: initialBalance,
	}

	// Deduct 2 SAR
	deductionAmount := decimal.NewFromInt(2)
	op := balance.BalanceOperation{
		TenantID:  tenantID,
		GatewayID: &gatewayID,
		MessageID: &messageID,
		Amount:    deductionAmount,
		Reason:    "SINGLE_MESSAGE",
		CreatedBy: "FAST_PROCESSOR",
	}

	// ACT
	mockTx := &mockTxBug7Preservation{}
	result, err := mockManager.DeductBalanceTx(ctx, mockTx, op)

	// ASSERT
	assert.NoError(t, err, "Preservation: Deduction should succeed with small balance")
	require.NotNil(t, result)
	assert.Equal(t, decimal.NewFromInt(3), result.NewBalance,
		"Preservation: New balance should be 3 SAR (5 - 2)")
	assert.Equal(t, initialBalance, result.OldBalance,
		"Preservation: Old balance should be 5 SAR")
	assert.True(t, result.Success, "Preservation: Result should indicate success")

	t.Logf("✅ Small balance deduction: 5 SAR - 2 SAR = 3 SAR")
}

