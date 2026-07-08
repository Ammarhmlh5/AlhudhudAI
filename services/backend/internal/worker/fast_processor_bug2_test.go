package worker

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"pgregory.net/rapid"

	"github.com/al-hudhud/backend/internal/balance"
	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/redis"
)

// Bug 2 Exploration Test: Nil PgxPool Crash
// **Validates: Requirements 1.2, 2.2**
//
// **Property 1: Bug Condition** - PgxPool Nil Pointer Crash
//
// This test MUST FAIL on unfixed code - failure confirms the bug exists.
// The bug: db.PgxPool is nil, causing panic when accessed.
//
// CRITICAL: This test encodes the EXPECTED behavior (what SHOULD happen).
// On UNFIXED code: Test FAILS with panic (proves bug exists)
// On FIXED code: Test PASSES (proves PgxPool is initialized)
//
// DO NOT attempt to fix the test or the code when it fails.
// Document the failure as proof of the bug.
func TestProperty_Bug2_PgxPoolNilPointerCrash(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	// Initialize database to test that PgxPool is properly initialized
	db.InitGormDB()

	ctx := context.Background()

	rapid.Check(t, func(t *rapid.T) {
		// Generate random tenant and message data
		tenantID := uuid.New()
		messageCount := rapid.IntRange(1, 5).Draw(t, "messageCount")
		
		messages := make([]*redis.MessageQueueItem, messageCount)
		for i := 0; i < messageCount; i++ {
			messages[i] = &redis.MessageQueueItem{
				MessageID:   uuid.New(),
				TenantID:    tenantID,
				PhoneNumber: rapid.StringMatching(`\+966[0-9]{9}`).Draw(t, "phone"),
				Content:     rapid.StringN(10, 100, -1).Draw(t, "content"),
				Cost:        rapid.Float64Range(0.01, 1.0).Draw(t, "cost"),
				Priority:    rapid.IntRange(1, 3).Draw(t, "priority"),
			}
		}

		// Create mock balance manager
		mockBalanceManager := &mockBalanceManager{}

		// Create processor (without Redis - we're only testing PgxPool access)
		logger := zap.NewNop()
		
		processor := &FastMessageProcessor{
			messageQueue:   nil, // Not needed for this test
			channelCache:   nil, // Not needed for this test
			balanceManager: mockBalanceManager,
			wsHub:          nil, // Will be handled by nil check in Bug 8
			logger:         logger,
			batchSize:      100,
			workerCount:    10,
		}

		// ACT & ASSERT: Attempt to process batch
		// On UNFIXED code: This will PANIC with "nil pointer dereference" when accessing db.PgxPool
		// On FIXED code: This will NOT panic (PgxPool is initialized)
		
		// We use a defer/recover to catch the panic and convert it to a test failure
		var panicOccurred bool
		var panicValue interface{}
		
		func() {
			defer func() {
				if r := recover(); r != nil {
					panicOccurred = true
					panicValue = r
				}
			}()
			
			// This should trigger db.PgxPool.Begin(ctx) on line 138
			_ = processor.processBatchWithTransaction(ctx, tenantID, messages)
		}()

		// ASSERT: Expected behavior - should NOT panic
		// On UNFIXED code: panicOccurred = true (bug exists)
		// On FIXED code: panicOccurred = false (bug fixed)
		assert.False(t, panicOccurred,
			"Bug Condition: Accessing db.PgxPool causes nil pointer panic. "+
				"Expected: No panic (PgxPool initialized). "+
				"Got: Panic occurred with value: %v. "+
				"This failure confirms Bug 2 exists: Nil PgxPool Crash",
			panicValue)
	})
}

// Example-based test for Bug 2 - Simpler counterexample
// **Validates: Requirements 1.2, 2.2**
//
// This test provides a concrete example of the bug condition.
// EXPECTED OUTCOME on UNFIXED code: FAILS with panic (proves bug exists)
func TestBug2_PgxPoolNilCrashOnTransaction(t *testing.T) {
	// Initialize database to test that PgxPool is properly initialized
	db.InitGormDB()

	ctx := context.Background()

	// Create test data
	tenantID := uuid.New()
	messages := []*redis.MessageQueueItem{
		{
			MessageID:   uuid.New(),
			TenantID:    tenantID,
			PhoneNumber: "+966501234567",
			Content:     "Test message",
			Cost:        0.05,
			Priority:    1,
		},
	}

	// Create mock balance manager
	mockBalanceManager := &mockBalanceManager{}

	// Create processor (without Redis - we're only testing PgxPool access)
	logger := zap.NewNop()
	
	processor := &FastMessageProcessor{
		messageQueue:   nil, // Not needed for this test
		channelCache:   nil, // Not needed for this test
		balanceManager: mockBalanceManager,
		wsHub:          nil,
		logger:         logger,
		batchSize:      100,
		workerCount:    10,
	}

	// ACT & ASSERT: Attempt to process batch
	var panicOccurred bool
	var panicValue interface{}
	
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicOccurred = true
				panicValue = r
			}
		}()
		
		// This will attempt to call db.PgxPool.Begin(ctx)
		_ = processor.processBatchWithTransaction(ctx, tenantID, messages)
	}()

	// ASSERT: Expected behavior - should NOT panic
	// On UNFIXED code: This WILL FAIL (panic occurs)
	// On FIXED code: This WILL PASS (no panic)
	assert.False(t, panicOccurred,
		"Bug Condition: db.PgxPool is nil, causing panic on Begin(). "+
			"Panic value: %v. "+
			"This confirms Bug 2 exists: Nil PgxPool Crash",
		panicValue)
}

// Test: Direct PgxPool access should not panic
// **Validates: Requirements 2.2**
//
// This test verifies that db.PgxPool can be accessed directly without panic.
func TestBug2_DirectPgxPoolAccess(t *testing.T) {
	// Initialize database to test that PgxPool is properly initialized
	db.InitGormDB()

	ctx := context.Background()

	// ACT & ASSERT: Direct access to db.PgxPool
	var panicOccurred bool
	var panicValue interface{}
	
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicOccurred = true
				panicValue = r
			}
		}()
		
		// Attempt to access db.PgxPool
		if db.PgxPool != nil {
			// Try to begin a transaction
			_, _ = db.PgxPool.Begin(ctx)
		}
	}()

	// ASSERT: Should not panic
	// On UNFIXED code: May panic if PgxPool is nil
	// On FIXED code: Should not panic
	assert.False(t, panicOccurred,
		"Direct access to db.PgxPool should not panic. "+
			"Panic value: %v. "+
			"PgxPool should be initialized in db.InitGormDB()",
		panicValue)
}

// Test: PgxPool should be non-nil after initialization
// **Validates: Requirements 2.2**
//
// This test verifies that db.PgxPool is properly initialized.
func TestBug2_PgxPoolInitialized(t *testing.T) {
	// Initialize database
	db.InitGormDB()
	
	// ASSERT: db.PgxPool should be initialized
	// On UNFIXED code: This WILL FAIL (PgxPool is nil)
	// On FIXED code: This WILL PASS (PgxPool is initialized)
	assert.NotNil(t, db.PgxPool,
		"Bug Condition: db.PgxPool is nil. "+
			"Expected: PgxPool initialized in db.InitGormDB(). "+
			"Got: nil. "+
			"This confirms Bug 2 exists: Nil PgxPool Crash")
}

// mockBalanceManager is a simple mock for testing
type mockBalanceManager struct{}

func (m *mockBalanceManager) DeductBalance(ctx context.Context, op balance.BalanceOperation) (*balance.BalanceResult, error) {
	return &balance.BalanceResult{
		Success:    true,
		OldBalance: decimal.NewFromFloat(100.0),
		NewBalance: decimal.NewFromFloat(95.0),
	}, nil
}

func (m *mockBalanceManager) DeductBalanceTx(ctx context.Context, tx pgx.Tx, op balance.BalanceOperation) (*balance.BalanceResult, error) {
	// Return a mock transaction
	return &balance.BalanceResult{
		Success:    true,
		OldBalance: decimal.NewFromFloat(100.0),
		NewBalance: decimal.NewFromFloat(95.0),
	}, nil
}

func (m *mockBalanceManager) DeductBalanceWithTimeout(ctx context.Context, op balance.BalanceOperation, timeout time.Duration) (*balance.BalanceResult, error) {
	return &balance.BalanceResult{
		Success:    true,
		OldBalance: decimal.NewFromFloat(100.0),
		NewBalance: decimal.NewFromFloat(95.0),
	}, nil
}

func (m *mockBalanceManager) RefundBalance(ctx context.Context, op balance.BalanceOperation) (*balance.BalanceResult, error) {
	return &balance.BalanceResult{
		Success:    true,
		OldBalance: decimal.NewFromFloat(95.0),
		NewBalance: decimal.NewFromFloat(100.0),
	}, nil
}

func (m *mockBalanceManager) RefundBalanceTx(ctx context.Context, tx pgx.Tx, op balance.BalanceOperation) (*balance.BalanceResult, error) {
	return &balance.BalanceResult{
		Success:    true,
		OldBalance: decimal.NewFromFloat(95.0),
		NewBalance: decimal.NewFromFloat(100.0),
	}, nil
}

func (m *mockBalanceManager) GetBalance(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
	return decimal.NewFromFloat(100.0), nil
}

func (m *mockBalanceManager) CheckSufficient(ctx context.Context, tenantID uuid.UUID, amount decimal.Decimal) (bool, error) {
	return true, nil
}
