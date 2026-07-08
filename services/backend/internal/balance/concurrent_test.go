package balance

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestConcurrentBalanceDeductions tests 100 concurrent balance deduction operations
// This test verifies:
// 1. No race conditions occur during concurrent operations
// 2. Final balance = Initial balance - Sum of successful deductions
// 3. All operations are atomic and consistent
// Validates: Requirements 1.3 (Atomic operations produce correct final balance)
func TestConcurrentBalanceDeductions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent operations test in short mode")
	}

	// Setup
	ctx := context.Background()
	pool := setupTestDB(t)
	defer pool.Close()

	logger, _ := zap.NewDevelopment()
	manager := NewManager(pool, logger)

	// Create test tenant with initial balance
	tenantID := createTestTenant(t, pool, decimal.NewFromFloat(1000.0))
	initialBalance := decimal.NewFromFloat(1000.0)

	// Test parameters
	numOperations := 100
	deductionAmount := decimal.NewFromFloat(5.0) // Each deduction is 5 units

	// Track results
	var wg sync.WaitGroup
	results := make(chan *BalanceResult, numOperations)
	errors := make(chan error, numOperations)

	// Start time for performance measurement
	startTime := time.Now()

	// Launch 100 concurrent deduction operations
	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(operationNum int) {
			defer wg.Done()

			op := BalanceOperation{
				TenantID:  tenantID,
				MessageID: func() *uuid.UUID { id := uuid.New(); return &id }(),
				Amount:    deductionAmount,
				Type:      OperationTypeDeduct,
				Reason:    "CONCURRENT_TEST",
				CreatedBy: "test",
				IPAddress: "127.0.0.1",
			}

			result, err := manager.DeductBalance(ctx, op)
			if err != nil {
				errors <- err
			} else {
				results <- result
			}
		}(i)
	}

	// Wait for all operations to complete
	wg.Wait()
	close(results)
	close(errors)

	duration := time.Since(startTime)

	// Collect results
	var successfulResults []*BalanceResult
	var failedErrors []error

	for result := range results {
		successfulResults = append(successfulResults, result)
	}

	for err := range errors {
		failedErrors = append(failedErrors, err)
	}

	// Get final balance from database
	finalBalance, err := manager.GetBalance(ctx, tenantID)
	require.NoError(t, err, "Failed to get final balance")

	// Calculate expected balance
	// Expected = Initial - (Number of successful operations * deduction amount)
	numSuccessful := len(successfulResults)
	totalDeducted := deductionAmount.Mul(decimal.NewFromInt(int64(numSuccessful)))
	expectedBalance := initialBalance.Sub(totalDeducted)

	// Assertions
	t.Run("Verify Final Balance Correctness", func(t *testing.T) {
		assert.Equal(t, expectedBalance.String(), finalBalance.String(),
			"Final balance should equal initial balance minus sum of successful deductions")
	})

	t.Run("Verify No Race Conditions", func(t *testing.T) {
		// If there were race conditions, the final balance would be incorrect
		// The atomic operations should ensure consistency
		assert.True(t, finalBalance.GreaterThanOrEqual(decimal.Zero),
			"Balance should never go negative due to race conditions")
	})

	t.Run("Verify All Operations Completed", func(t *testing.T) {
		totalOperations := numSuccessful + len(failedErrors)
		assert.Equal(t, numOperations, totalOperations,
			"All operations should either succeed or fail, none should be lost")
	})

	t.Run("Verify Insufficient Balance Errors", func(t *testing.T) {
		// Some operations may fail due to insufficient balance
		// This is expected and correct behavior
		for _, err := range failedErrors {
			balErr, ok := err.(*BalanceError)
			require.True(t, ok, "Error should be of type BalanceError")
			assert.Equal(t, ErrorInsufficientBalance, balErr.Type,
				"Failed operations should be due to insufficient balance")
		}
	})

	t.Run("Verify Balance History Completeness", func(t *testing.T) {
		// Query balance history to verify all successful operations were logged
		var historyCount int
		query := `SELECT COUNT(*) FROM balance_history WHERE tenant_id = $1 AND type = 'DEDUCT'`
		err := pool.QueryRow(ctx, query, tenantID).Scan(&historyCount)
		require.NoError(t, err, "Failed to query balance history")

		assert.Equal(t, numSuccessful, historyCount,
			"Balance history should contain exactly one entry per successful operation")
	})

	t.Run("Verify Balance History Invariant", func(t *testing.T) {
		// Sum of all balance changes should equal the difference between initial and final balance
		var totalChange decimal.Decimal
		query := `
			SELECT COALESCE(SUM(amount), 0) 
			FROM balance_history 
			WHERE tenant_id = $1 AND type = 'DEDUCT'
		`
		err := pool.QueryRow(ctx, query, tenantID).Scan(&totalChange)
		require.NoError(t, err, "Failed to query balance history sum")

		actualChange := initialBalance.Sub(finalBalance)
		assert.Equal(t, actualChange.String(), totalChange.String(),
			"Sum of balance history changes should equal actual balance change")
	})

	t.Run("Performance Metrics", func(t *testing.T) {
		avgDuration := duration.Milliseconds() / int64(numOperations)
		t.Logf("Concurrent Operations Performance:")
		t.Logf("  Total operations: %d", numOperations)
		t.Logf("  Successful: %d", numSuccessful)
		t.Logf("  Failed: %d", len(failedErrors))
		t.Logf("  Total duration: %v", duration)
		t.Logf("  Average per operation: %d ms", avgDuration)
		t.Logf("  Initial balance: %s", initialBalance.String())
		t.Logf("  Final balance: %s", finalBalance.String())
		t.Logf("  Total deducted: %s", totalDeducted.String())

		// Performance assertion: operations should complete in reasonable time
		assert.Less(t, duration.Seconds(), 30.0,
			"100 concurrent operations should complete within 30 seconds")
	})

	// Cleanup
	cleanupTestTenant(t, pool, tenantID)
}

// TestConcurrentMixedOperations tests concurrent deductions and refunds
// This verifies that mixed operations maintain consistency
func TestConcurrentMixedOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent mixed operations test in short mode")
	}

	ctx := context.Background()
	pool := setupTestDB(t)
	defer pool.Close()

	logger, _ := zap.NewDevelopment()
	manager := NewManager(pool, logger)

	// Create test tenant
	tenantID := createTestTenant(t, pool, decimal.NewFromFloat(500.0))
	initialBalance := decimal.NewFromFloat(500.0)

	// Test parameters
	numDeductions := 50
	numRefunds := 50
	operationAmount := decimal.NewFromFloat(5.0)

	var wg sync.WaitGroup
	successfulDeductions := make(chan bool, numDeductions)
	successfulRefunds := make(chan bool, numRefunds)

	// Launch concurrent deductions
	for i := 0; i < numDeductions; i++ {
		wg.Add(1)
		go func(num int) {
			defer wg.Done()
			op := BalanceOperation{
				TenantID:  tenantID,
				MessageID: func() *uuid.UUID { id := uuid.New(); return &id }(),
				Amount:    operationAmount,
				Type:      OperationTypeDeduct,
				Reason:    "MIXED_TEST_DEDUCT",
				CreatedBy: "test",
			}
			_, err := manager.DeductBalance(ctx, op)
			if err == nil {
				successfulDeductions <- true
			}
		}(i)
	}

	// Launch concurrent refunds
	for i := 0; i < numRefunds; i++ {
		wg.Add(1)
		go func(num int) {
			defer wg.Done()
			op := BalanceOperation{
				TenantID:  tenantID,
				MessageID: func() *uuid.UUID { id := uuid.New(); return &id }(),
				Amount:    operationAmount,
				Type:      OperationTypeRefund,
				Reason:    "MIXED_TEST_REFUND",
				CreatedBy: "test",
			}
			_, err := manager.RefundBalance(ctx, op)
			if err == nil {
				successfulRefunds <- true
			}
		}(i)
	}

	wg.Wait()
	close(successfulDeductions)
	close(successfulRefunds)

	// Count successful operations
	deductCount := len(successfulDeductions)
	refundCount := len(successfulRefunds)

	// Get final balance
	finalBalance, err := manager.GetBalance(ctx, tenantID)
	require.NoError(t, err)

	// Calculate expected balance
	expectedBalance := initialBalance.
		Sub(operationAmount.Mul(decimal.NewFromInt(int64(deductCount)))).
		Add(operationAmount.Mul(decimal.NewFromInt(int64(refundCount))))

	t.Run("Verify Mixed Operations Balance", func(t *testing.T) {
		assert.Equal(t, expectedBalance.String(), finalBalance.String(),
			"Final balance should reflect all successful deductions and refunds")
	})

	t.Run("Verify Balance Never Negative", func(t *testing.T) {
		assert.True(t, finalBalance.GreaterThanOrEqual(decimal.Zero),
			"Balance should never go negative even with mixed concurrent operations")
	})

	t.Logf("Mixed Operations Results:")
	t.Logf("  Successful deductions: %d/%d", deductCount, numDeductions)
	t.Logf("  Successful refunds: %d/%d", refundCount, numRefunds)
	t.Logf("  Initial balance: %s", initialBalance.String())
	t.Logf("  Final balance: %s", finalBalance.String())
	t.Logf("  Expected balance: %s", expectedBalance.String())

	cleanupTestTenant(t, pool, tenantID)
}

// TestHighContentionScenario tests extreme contention with many operations on same tenant
func TestHighContentionScenario(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping high contention test in short mode")
	}

	ctx := context.Background()
	pool := setupTestDB(t)
	defer pool.Close()

	logger, _ := zap.NewDevelopment()
	manager := NewManager(pool, logger)

	// Create tenant with high initial balance
	tenantID := createTestTenant(t, pool, decimal.NewFromFloat(10000.0))
	initialBalance := decimal.NewFromFloat(10000.0)

	// Extreme contention: 200 operations
	numOperations := 200
	deductionAmount := decimal.NewFromFloat(10.0)

	var wg sync.WaitGroup
	successCount := make(chan bool, numOperations)

	startTime := time.Now()

	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(num int) {
			defer wg.Done()
			op := BalanceOperation{
				TenantID:  tenantID,
				MessageID: func() *uuid.UUID { id := uuid.New(); return &id }(),
				Amount:    deductionAmount,
				Type:      OperationTypeDeduct,
				Reason:    "HIGH_CONTENTION_TEST",
				CreatedBy: "test",
			}
			_, err := manager.DeductBalance(ctx, op)
			if err == nil {
				successCount <- true
			}
		}(i)
	}

	wg.Wait()
	close(successCount)
	duration := time.Since(startTime)

	numSuccessful := len(successCount)
	finalBalance, err := manager.GetBalance(ctx, tenantID)
	require.NoError(t, err)

	expectedBalance := initialBalance.Sub(deductionAmount.Mul(decimal.NewFromInt(int64(numSuccessful))))

	t.Run("Verify High Contention Consistency", func(t *testing.T) {
		assert.Equal(t, expectedBalance.String(), finalBalance.String(),
			"Balance should be consistent even under high contention")
	})

	t.Logf("High Contention Test Results:")
	t.Logf("  Total operations: %d", numOperations)
	t.Logf("  Successful: %d", numSuccessful)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Operations/second: %.2f", float64(numOperations)/duration.Seconds())

	cleanupTestTenant(t, pool, tenantID)
}








