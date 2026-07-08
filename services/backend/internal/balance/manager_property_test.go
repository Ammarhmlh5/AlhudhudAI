package balance

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"


	"github.com/google/uuid"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// Property 15: Atomic Balance Operations
// **Validates: Requirements 16.1, 16.2**
// 
// لأي عمليتي خصم متزامنتين على نفس الحساب، يجب أن تنجح واحدة فقط إذا كان الرصيد كافياً لواحدة فقط،
// ويجب ألا يصبح الرصيد سالباً أبداً
//
// This property test validates:
// 1. Atomic balance deduction with race condition protection
// 2. Balance never goes negative under concurrent operations
// 3. Exactly one operation succeeds when balance is sufficient for only one
// 4. All successful operations are logged in balance_history

func TestProperty_AtomicBalanceOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	// Setup
	ctx := context.Background()
	pool := setupTestDB(t)
	defer pool.Close()

	logger, _ := zap.NewDevelopment()
	manager := NewManager(pool, logger)

	// Test parameters
	numIterations := 10 // عدد التكرارات
	numConcurrentOps := 50 // عدد العمليات المتزامنة

	for iteration := 0; iteration < numIterations; iteration++ {
		t.Run(fmt.Sprintf("Iteration_%d", iteration+1), func(t *testing.T) {
			// إنشاء tenant جديد لكل تكرار
			initialBalance := decimal.NewFromFloat(float64(10000 + rand.Intn(10000))) // 10000-20000
			tenantID := createTestTenant(t, pool, initialBalance)

			// توليد عمليات خصم عشوائية
			deductions := make([]decimal.Decimal, numConcurrentOps)
			for i := 0; i < numConcurrentOps; i++ {
				// مبالغ عشوائية بين 10 و 500
				deductions[i] = decimal.NewFromFloat(float64(10 + rand.Intn(490)))
			}

			// تنفيذ العمليات بشكل متزامن
			var wg sync.WaitGroup
			successfulDeductions := make([]decimal.Decimal, numConcurrentOps)
			errors := make([]error, numConcurrentOps)

			for i := 0; i < numConcurrentOps; i++ {
				wg.Add(1)
				go func(index int, amount decimal.Decimal) {
					defer wg.Done()

					op := BalanceOperation{
						TenantID:  tenantID,
						MessageID: func() *uuid.UUID { id := uuid.New(); return &id }(),
						Amount:    amount,
						Type:      OperationTypeDeduct,
						Reason:    "PROPERTY_TEST",
						CreatedBy: "test",
						IPAddress: "127.0.0.1",
					}

					_, err := manager.DeductBalance(ctx, op)
					
					if err == nil {
						successfulDeductions[index] = amount
					}
					errors[index] = err
				}(i, deductions[i])
			}

			wg.Wait()

			// حساب مجموع الخصومات الناجحة
			totalSuccessfulDeductions := decimal.Zero
			successCount := 0
			for i := 0; i < numConcurrentOps; i++ {
				if errors[i] == nil {
					totalSuccessfulDeductions = totalSuccessfulDeductions.Add(successfulDeductions[i])
					successCount++
				}
			}

			// التحقق من الرصيد النهائي
			finalBalance, err := manager.GetBalance(ctx, tenantID)
			require.NoError(t, err)

			expectedBalance := initialBalance.Sub(totalSuccessfulDeductions)

			// Property: الرصيد النهائي = الرصيد الأولي - مجموع الخصومات الناجحة
			assert.Equal(t, expectedBalance.String(), finalBalance.String(),
				"Property violated: Final balance should equal initial balance minus successful deductions\n"+
					"Initial: %s, Final: %s, Expected: %s, Total Deducted: %s, Success Count: %d/%d",
				initialBalance.String(), finalBalance.String(), expectedBalance.String(), 
				totalSuccessfulDeductions.String(), successCount, numConcurrentOps)

			// التحقق من أن الرصيد لم يصبح سالباً أبداً
			assert.True(t, finalBalance.GreaterThanOrEqual(decimal.Zero),
				"Property violated: Balance should never be negative, got: %s", finalBalance.String())

			// التحقق من عدد السجلات في balance_history
			historyCount := getBalanceHistoryCount(t, pool, tenantID)
			assert.Equal(t, successCount, historyCount,
				"Property violated: History count should match successful deductions")

			t.Logf("Iteration %d: Initial=%s, Final=%s, Deducted=%s, Success=%d/%d",
				iteration+1, initialBalance.String(), finalBalance.String(), 
				totalSuccessfulDeductions.String(), successCount, numConcurrentOps)
		})
	}
}


// Property Test: Race Condition Detection
// Tests that concurrent operations on the same account don't cause race conditions
// **Validates: Requirements 16.1, 16.2**
func TestProperty_NoRaceConditionsInBalanceOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	// Setup
	ctx := context.Background()
	pool := setupTestDB(t)
	defer pool.Close()

	logger, _ := zap.NewDevelopment()
	manager := NewManager(pool, logger)

	// Test parameters
	numIterations := 5
	numConcurrentOps := 100

	for iteration := 0; iteration < numIterations; iteration++ {
		t.Run(fmt.Sprintf("RaceDetection_Iteration_%d", iteration+1), func(t *testing.T) {
			initialBalance := decimal.NewFromFloat(50000) // رصيد كبير لتقليل الرفض
			tenantID := createTestTenant(t, pool, initialBalance)

			// تنفيذ عمليات متزامنة بنفس المبلغ
			deductionAmount := decimal.NewFromFloat(100.0)
			var wg sync.WaitGroup
			successCount := 0
			var mu sync.Mutex

			for i := 0; i < numConcurrentOps; i++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()

					op := BalanceOperation{
						TenantID:  tenantID,
						MessageID: func() *uuid.UUID { id := uuid.New(); return &id }(),
						Amount:    deductionAmount,
						Type:      OperationTypeDeduct,
						Reason:    "RACE_TEST",
						CreatedBy: "test",
						IPAddress: "127.0.0.1",
					}

					_, err := manager.DeductBalance(ctx, op)
					
					if err == nil {
						mu.Lock()
						successCount++
						mu.Unlock()
					}
				}(i)
			}

			wg.Wait()

			// التحقق من الرصيد النهائي
			finalBalance, err := manager.GetBalance(ctx, tenantID)
			require.NoError(t, err)

			expectedBalance := initialBalance.Sub(deductionAmount.Mul(decimal.NewFromInt(int64(successCount))))

			// Property: لا race conditions - الرصيد دقيق
			assert.Equal(t, expectedBalance.String(), finalBalance.String(),
				"Race condition detected: Balance mismatch\n"+
					"Initial: %s, Final: %s, Expected: %s, Success: %d",
				initialBalance.String(), finalBalance.String(), expectedBalance.String(), successCount)

			t.Logf("Race test %d: Success=%d/%d, Balance=%s (expected %s)",
				iteration+1, successCount, numConcurrentOps, finalBalance.String(), expectedBalance.String())
		})
	}
}

// Property Test: Exactly One Succeeds When Balance Sufficient For One
// Tests the critical property: when balance is sufficient for only one operation,
// exactly one should succeed and balance should never go negative
// **Validates: Requirements 16.1, 16.2**
func TestProperty_ExactlyOneSucceedsWhenBalanceSufficientForOne(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	// Setup
	ctx := context.Background()
	pool := setupTestDB(t)
	defer pool.Close()

	logger, _ := zap.NewDevelopment()
	manager := NewManager(pool, logger)

	numIterations := 20

	for iteration := 0; iteration < numIterations; iteration++ {
		t.Run(fmt.Sprintf("OneSucceeds_Iteration_%d", iteration+1), func(t *testing.T) {
			// Set balance to exactly the amount needed for one operation
			operationAmount := decimal.NewFromFloat(100.0)
			initialBalance := operationAmount // Balance sufficient for exactly one operation
			tenantID := createTestTenant(t, pool, initialBalance)

			// Launch multiple concurrent operations
			numConcurrentOps := 10
			var wg sync.WaitGroup
			successCount := 0
			var mu sync.Mutex

			for i := 0; i < numConcurrentOps; i++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()

					op := BalanceOperation{
						TenantID:  tenantID,
						MessageID: func() *uuid.UUID { id := uuid.New(); return &id }(),
						Amount:    operationAmount,
						Type:      OperationTypeDeduct,
						Reason:    "ONE_SUCCEEDS_TEST",
						CreatedBy: "test",
						IPAddress: "127.0.0.1",
					}

					_, err := manager.DeductBalance(ctx, op)
					
					if err == nil {
						mu.Lock()
						successCount++
						mu.Unlock()
					}
				}(i)
			}

			wg.Wait()

			// Property: Exactly one operation should succeed
			assert.Equal(t, 1, successCount,
				"Property violated: Exactly one operation should succeed when balance is sufficient for one\n"+
					"Initial balance: %s, Operation amount: %s, Success count: %d",
				initialBalance.String(), operationAmount.String(), successCount)

			// Verify final balance
			finalBalance, err := manager.GetBalance(ctx, tenantID)
			require.NoError(t, err)

			// Property: Final balance should be zero (one operation succeeded)
			assert.Equal(t, decimal.Zero.String(), finalBalance.String(),
				"Property violated: Final balance should be zero after one successful operation\n"+
					"Got: %s", finalBalance.String())

			// Property: Balance never went negative
			assert.True(t, finalBalance.GreaterThanOrEqual(decimal.Zero),
				"Property violated: Balance should never be negative, got: %s", finalBalance.String())

			t.Logf("One succeeds test %d: Success=%d/%d, Final balance=%s",
				iteration+1, successCount, numConcurrentOps, finalBalance.String())
		})
	}
}


// Property Test: All Successful Operations Are Logged
// Verifies that every successful balance operation is recorded in balance_history
// **Validates: Requirements 16.1, 16.2**
func TestProperty_AllSuccessfulOperationsAreLogged(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	// Setup
	ctx := context.Background()
	pool := setupTestDB(t)
	defer pool.Close()

	logger, _ := zap.NewDevelopment()
	manager := NewManager(pool, logger)

	numIterations := 10

	for iteration := 0; iteration < numIterations; iteration++ {
		t.Run(fmt.Sprintf("Logging_Iteration_%d", iteration+1), func(t *testing.T) {
			initialBalance := decimal.NewFromFloat(5000)
			tenantID := createTestTenant(t, pool, initialBalance)

			numOperations := 20
			var wg sync.WaitGroup
			successCount := 0
			var mu sync.Mutex

			for i := 0; i < numOperations; i++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()

					amount := decimal.NewFromFloat(float64(50 + rand.Intn(100)))
					op := BalanceOperation{
						TenantID:  tenantID,
						MessageID: func() *uuid.UUID { id := uuid.New(); return &id }(),
						Amount:    amount,
						Type:      OperationTypeDeduct,
						Reason:    "LOGGING_TEST",
						CreatedBy: "test",
						IPAddress: "127.0.0.1",
					}

					_, err := manager.DeductBalance(ctx, op)
					
					if err == nil {
						mu.Lock()
						successCount++
						mu.Unlock()
					}
				}(i)
			}

			wg.Wait()

			// Property: History count must equal success count
			historyCount := getBalanceHistoryCount(t, pool, tenantID)
			assert.Equal(t, successCount, historyCount,
				"Property violated: All successful operations must be logged\n"+
					"Success count: %d, History count: %d",
				successCount, historyCount)

			t.Logf("Logging test %d: Success=%d, History=%d",
				iteration+1, successCount, historyCount)
		})
	}
}

// Property Test: Balance Never Goes Negative Under Any Circumstances
// Stress test with many concurrent operations to ensure balance constraint
// **Validates: Requirements 16.1, 16.2**
func TestProperty_BalanceNeverGoesNegative(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	// Setup
	ctx := context.Background()
	pool := setupTestDB(t)
	defer pool.Close()

	logger, _ := zap.NewDevelopment()
	manager := NewManager(pool, logger)

	numIterations := 15

	for iteration := 0; iteration < numIterations; iteration++ {
		t.Run(fmt.Sprintf("NeverNegative_Iteration_%d", iteration+1), func(t *testing.T) {
			// Small initial balance to increase pressure
			initialBalance := decimal.NewFromFloat(float64(500 + rand.Intn(500)))
			tenantID := createTestTenant(t, pool, initialBalance)

			// Many concurrent operations with random amounts
			numConcurrentOps := 100
			var wg sync.WaitGroup

			for i := 0; i < numConcurrentOps; i++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()

					amount := decimal.NewFromFloat(float64(10 + rand.Intn(100)))
					op := BalanceOperation{
						TenantID:  tenantID,
						MessageID: func() *uuid.UUID { id := uuid.New(); return &id }(),
						Amount:    amount,
						Type:      OperationTypeDeduct,
						Reason:    "NEVER_NEGATIVE_TEST",
						CreatedBy: "test",
						IPAddress: "127.0.0.1",
					}

					// Ignore errors - we expect many to fail due to insufficient balance
					manager.DeductBalance(ctx, op)
				}(i)
			}

			wg.Wait()

			// Property: Balance must never be negative
			finalBalance, err := manager.GetBalance(ctx, tenantID)
			require.NoError(t, err)

			assert.True(t, finalBalance.GreaterThanOrEqual(decimal.Zero),
				"CRITICAL PROPERTY VIOLATED: Balance went negative!\n"+
					"Initial: %s, Final: %s",
				initialBalance.String(), finalBalance.String())

			t.Logf("Never negative test %d: Initial=%s, Final=%s",
				iteration+1, initialBalance.String(), finalBalance.String())
		})
	}
}


// Helper functions for property tests


