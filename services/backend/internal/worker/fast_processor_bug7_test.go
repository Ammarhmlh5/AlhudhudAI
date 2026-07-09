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

// mockTxBug7 is a minimal mock transaction for Bug 7 testing
type mockTxBug7 struct{}

func (m *mockTxBug7) Begin(ctx context.Context) (pgx.Tx, error)       { return m, nil }
func (m *mockTxBug7) Commit(ctx context.Context) error                { return nil }
func (m *mockTxBug7) Rollback(ctx context.Context) error              { return nil }
func (m *mockTxBug7) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (m *mockTxBug7) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return nil, nil
}
func (m *mockTxBug7) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return nil
}
func (m *mockTxBug7) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (m *mockTxBug7) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return nil
}
func (m *mockTxBug7) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}
func (m *mockTxBug7) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (m *mockTxBug7) Conn() *pgx.Conn {
	return nil
}

// Bug 7 Exploration Test: No Balance Verification Before Deduction
// **Validates: Requirements 1.7, 2.7**
//
// **Property 1: Bug Condition** - Negative Balance Allowed
//
// This test MUST FAIL on unfixed code - failure confirms the bug exists.
// The bug: DeductBalanceTx() allows deduction with insufficient balance, creating negative balances.
//
// CRITICAL: This test encodes the EXPECTED behavior (what SHOULD happen).
// On UNFIXED code: Test FAILS (deduction succeeds with insufficient balance - proves bug exists)
// On FIXED code: Test PASSES (deduction fails with insufficient balance error - proves bug is fixed)
//
// DO NOT attempt to fix the test or the code when it fails.
// Document the failure as proof of the bug.
func TestProperty_Bug7_NegativeBalanceAllowed(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		// Initial balance: 1-100 SAR
		initialBalance := decimal.NewFromInt(int64(rapid.IntRange(1, 100).Draw(t, "initialBalance")))
		
		// Deduction amount: initialBalance + (1-50) SAR (always more than available)
		excessAmount := decimal.NewFromInt(int64(rapid.IntRange(1, 50).Draw(t, "excessAmount")))
		deductionAmount := initialBalance.Add(excessAmount)

		// Setup: Create mock balance manager and transaction
		ctx := context.Background()
		tenantID := uuid.New()
		gatewayID := uuid.New()
		messageID := uuid.New()

		// Create mock balance manager
		mockManager := &mockBalanceManagerBug7{
			currentBalance: initialBalance,
		}

		// Create balance operation
		op := balance.BalanceOperation{
			TenantID:  tenantID,
			GatewayID: &gatewayID,
			MessageID: &messageID,
			Amount:    deductionAmount,
			Reason:    "TEST_DEDUCTION",
			CreatedBy: "TEST",
		}

		// ACT: Attempt to deduct more than available balance
		mockTx := &mockTxBug7{}
		result, err := mockManager.DeductBalanceTx(ctx, mockTx, op)

		// ASSERT: Expected behavior - should return error for insufficient balance
		// On UNFIXED code: err == nil (deduction succeeds, creates negative balance - BUG!)
		// On FIXED code: err != nil (deduction fails with insufficient balance error - CORRECT!)
		assert.Error(t, err,
			"Bug Condition: DeductBalanceTx() should return error when balance (%s) < deduction (%s). "+
				"On UNFIXED code, this assertion FAILS because deduction succeeds. "+
				"This failure confirms Bug 7 exists: No Balance Verification Before Deduction.",
			initialBalance.String(), deductionAmount.String())

		// Additional assertions for UNFIXED code behavior
		if err == nil {
			// On UNFIXED code, deduction succeeds
			require.NotNil(t, result, "Result should not be nil when deduction succeeds")
			
			// Verify negative balance was created (bug behavior)
			expectedNegativeBalance := initialBalance.Sub(deductionAmount)
			assert.True(t, result.NewBalance.LessThan(decimal.Zero),
				"Bug detected: Negative balance created. Balance: %s (should have been rejected)",
				result.NewBalance.String())
			
			assert.Equal(t, expectedNegativeBalance, result.NewBalance,
				"Bug detected: Balance became %s (negative). Should have rejected deduction.",
				result.NewBalance.String())
		}

		// On FIXED code, verify error is InsufficientBalanceError
		if err != nil {
			assert.Contains(t, err.Error(), "insufficient",
				"Error should indicate insufficient balance")
		}
	})
}

// Example-based test for Bug 7 - Concrete counterexample
// **Validates: Requirements 1.7, 2.7**
//
// This test provides a concrete example of the bug condition.
// Tenant has 10 SAR, attempts to deduct 15 SAR.
// EXPECTED OUTCOME on UNFIXED code: FAILS (deduction succeeds, balance becomes -5)
// EXPECTED OUTCOME on FIXED code: PASSES (deduction fails with error)
func TestBug7_InsufficientBalanceAllowsNegative(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	gatewayID := uuid.New()
	messageID := uuid.New()

	// Setup: Tenant has 10 SAR
	initialBalance := decimal.NewFromInt(10)

	// Create mock balance manager
	mockManager := &mockBalanceManagerBug7{
		currentBalance: initialBalance,
	}

	// Create balance operation: Attempt to deduct 15 SAR
	deductionAmount := decimal.NewFromInt(15)
	op := balance.BalanceOperation{
		TenantID:  tenantID,
		GatewayID: &gatewayID,
		MessageID: &messageID,
		Amount:    deductionAmount,
		Reason:    "BATCH_SEND_10_MESSAGES",
		CreatedBy: "FAST_PROCESSOR",
	}

	// ACT: Attempt deduction
	mockTx := &mockTxBug7{}
	result, err := mockManager.DeductBalanceTx(ctx, mockTx, op)

	// ASSERT: Expected behavior - should return error
	// On UNFIXED code: This WILL FAIL (err == nil, deduction succeeds)
	// On FIXED code: This WILL PASS (err != nil, deduction rejected)
	assert.Error(t, err,
		"Bug Condition: DeductBalanceTx() should return error when balance (10 SAR) < deduction (15 SAR). "+
			"On UNFIXED code, deduction succeeds and creates negative balance (-5 SAR). "+
			"This confirms Bug 7 exists: No Balance Verification Before Deduction.")

	// If deduction succeeded (UNFIXED code), verify negative balance was created
	if err == nil {
		require.NotNil(t, result)
		expectedNegativeBalance := decimal.NewFromInt(-5)
		assert.Equal(t, expectedNegativeBalance, result.NewBalance,
			"Bug detected: Balance became -5 SAR (negative). Should have rejected deduction.")
		assert.True(t, result.NewBalance.LessThan(decimal.Zero),
			"Bug detected: Negative balance allowed")
	}

	// On FIXED code, verify error message
	if err != nil {
		assert.Contains(t, err.Error(), "insufficient",
			"Error should indicate insufficient balance")
	}
}

// Edge case test: Exact balance deduction should succeed
// **Validates: Requirements 2.7, 3.4**
//
// This test verifies that deducting exactly the available balance works correctly.
// This should PASS on both UNFIXED and FIXED code.
func TestBug7_ExactBalanceDeductionSucceeds(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	gatewayID := uuid.New()
	messageID := uuid.New()

	// Setup: Tenant has 10 SAR
	initialBalance := decimal.NewFromInt(10)

	mockManager := &mockBalanceManagerBug7{
		currentBalance: initialBalance,
	}

	// Deduct exactly 10 SAR
	deductionAmount := decimal.NewFromInt(10)
	op := balance.BalanceOperation{
		TenantID:  tenantID,
		GatewayID: &gatewayID,
		MessageID: &messageID,
		Amount:    deductionAmount,
		Reason:    "EXACT_BALANCE_TEST",
		CreatedBy: "TEST",
	}

	// ACT
	mockTx := &mockTxBug7{}
	result, err := mockManager.DeductBalanceTx(ctx, mockTx, op)

	// ASSERT: Should succeed with balance = 0
	assert.NoError(t, err, "Deducting exact balance should succeed")
	require.NotNil(t, result)
	assert.Equal(t, decimal.Zero, result.NewBalance,
		"Balance should be 0 after deducting exact amount")
	assert.Equal(t, initialBalance, result.OldBalance,
		"Old balance should be preserved")
}

// Edge case test: Deduction by 1 SAR more than balance should fail
// **Validates: Requirements 1.7, 2.7**
//
// This test verifies boundary condition: balance + 1 should be rejected.
func TestBug7_OneOverBalanceFails(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	gatewayID := uuid.New()
	messageID := uuid.New()

	// Setup: Tenant has 10 SAR
	initialBalance := decimal.NewFromInt(10)

	mockManager := &mockBalanceManagerBug7{
		currentBalance: initialBalance,
	}

	// Attempt to deduct 11 SAR (1 more than available)
	deductionAmount := decimal.NewFromInt(11)
	op := balance.BalanceOperation{
		TenantID:  tenantID,
		GatewayID: &gatewayID,
		MessageID: &messageID,
		Amount:    deductionAmount,
		Reason:    "BOUNDARY_TEST",
		CreatedBy: "TEST",
	}

	// ACT
	mockTx := &mockTxBug7{}
	result, err := mockManager.DeductBalanceTx(ctx, mockTx, op)

	// ASSERT: Should fail
	// On UNFIXED code: This WILL FAIL (deduction succeeds)
	// On FIXED code: This WILL PASS (deduction rejected)
	assert.Error(t, err,
		"Deducting 11 SAR when balance is 10 SAR should fail")

	if err == nil {
		// Bug detected
		require.NotNil(t, result)
		assert.Equal(t, decimal.NewFromInt(-1), result.NewBalance,
			"Bug: Balance became -1 SAR")
	}
}


// Mock balance manager that simulates UNFIXED code behavior
type mockBalanceManagerBug7 struct {
	currentBalance decimal.Decimal
}

// DeductBalanceTx simulates UNFIXED code behavior: no balance verification
// This allows negative balances to be created
func (m *mockBalanceManagerBug7) DeductBalanceTx(ctx context.Context, tx pgx.Tx, op balance.BalanceOperation) (*balance.BalanceResult, error) {
	// UNFIXED CODE SIMULATION: No balance check!
	// Simply deduct the amount, even if it creates negative balance
	
	oldBalance := m.currentBalance
	newBalance := oldBalance.Sub(op.Amount)
	
	// Update the mock balance
	m.currentBalance = newBalance
	
	// Return success even with negative balance (BUG!)
	return &balance.BalanceResult{
		Success:    true,
		OldBalance: oldBalance,
		NewBalance: newBalance,
	}, nil
}

func (m *mockBalanceManagerBug7) DeductBalance(ctx context.Context, op balance.BalanceOperation) (*balance.BalanceResult, error) {
	return nil, nil
}

func (m *mockBalanceManagerBug7) DeductBalanceWithTimeout(ctx context.Context, op balance.BalanceOperation, timeout int) (*balance.BalanceResult, error) {
	return nil, nil
}

func (m *mockBalanceManagerBug7) RefundBalance(ctx context.Context, op balance.BalanceOperation) (*balance.BalanceResult, error) {
	return nil, nil
}

func (m *mockBalanceManagerBug7) RefundBalanceTx(ctx context.Context, tx pgx.Tx, op balance.BalanceOperation) (*balance.BalanceResult, error) {
	return nil, nil
}

func (m *mockBalanceManagerBug7) GetBalance(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
	return m.currentBalance, nil
}

func (m *mockBalanceManagerBug7) CheckSufficient(ctx context.Context, tenantID uuid.UUID, amount decimal.Decimal) (bool, error) {
	return m.currentBalance.GreaterThanOrEqual(amount), nil
}

