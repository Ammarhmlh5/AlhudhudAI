package balance

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestCreditBalance_Success verifies that a credit increases the tenant's
// wallet balance and records a history entry.
func TestCreditBalance_Success(t *testing.T) {
	ctx := context.Background()
	pool := setupTestDB(t)
	defer pool.Close()

	manager := NewManager(pool, zap.NewNop())

	tenantID := createTestTenant(t, pool, decimal.NewFromFloat(100.0))
	defer cleanupTestTenant(t, pool, tenantID)

	op := BalanceOperation{
		TenantID:  tenantID,
		Amount:    decimal.NewFromFloat(50.5),
		Type:      OperationTypeCharge,
		Reason:    "admin funding",
		CreatedBy: "admin@test.local",
		IPAddress: "127.0.0.1",
	}

	result, err := manager.CreditBalance(ctx, op)
	require.NoError(t, err)
	require.True(t, result.Success)
	require.Equal(t, "100", result.OldBalance.String())
	require.Equal(t, "150.5", result.NewBalance.String())

	bal, err := manager.GetBalance(ctx, tenantID)
	require.NoError(t, err)
	require.Equal(t, "150.5", bal.String())

	require.Equal(t, 1, getBalanceHistoryCount(t, pool, tenantID))
}

// TestCreditBalance_InvalidAmount verifies that non-positive amounts are rejected.
func TestCreditBalance_InvalidAmount(t *testing.T) {
	ctx := context.Background()
	pool := setupTestDB(t)
	defer pool.Close()

	manager := NewManager(pool, zap.NewNop())
	tenantID := createTestTenant(t, pool, decimal.Zero)
	defer cleanupTestTenant(t, pool, tenantID)

	for _, amount := range []decimal.Decimal{decimal.Zero, decimal.NewFromFloat(-10)} {
		_, err := manager.CreditBalance(ctx, BalanceOperation{
			TenantID: tenantID,
			Amount:   amount,
			Type:     OperationTypeCharge,
			Reason:   "invalid amount",
		})
		require.Error(t, err)
		var be *BalanceError
		require.ErrorAs(t, err, &be)
		require.Equal(t, ErrorDatabaseError, be.Type)
	}

	bal, err := manager.GetBalance(ctx, tenantID)
	require.NoError(t, err)
	require.Equal(t, "0", bal.String())
}

// TestCreditBalance_NonExistentTenant verifies that crediting a missing tenant fails.
func TestCreditBalance_NonExistentTenant(t *testing.T) {
	ctx := context.Background()
	pool := setupTestDB(t)
	defer pool.Close()

	manager := NewManager(pool, zap.NewNop())

	_, err := manager.CreditBalance(ctx, BalanceOperation{
		TenantID: uuid.New(),
		Amount:   decimal.NewFromFloat(10),
		Type:     OperationTypeCharge,
		Reason:   "missing tenant",
	})
	require.Error(t, err)
	var be *BalanceError
	require.ErrorAs(t, err, &be)
	require.Equal(t, ErrorTenantNotFound, be.Type)
}

// TestTransferBalance_Success verifies an atomic transfer updates both wallets
// and records history entries on both sides.
func TestTransferBalance_Success(t *testing.T) {
	ctx := context.Background()
	pool := setupTestDB(t)
	defer pool.Close()

	manager := NewManager(pool, zap.NewNop())

	fromID := createTestTenant(t, pool, decimal.NewFromFloat(100.0))
	toID := createTestTenant(t, pool, decimal.NewFromFloat(10.0))
	defer cleanupTestTenant(t, pool, fromID)
	defer cleanupTestTenant(t, pool, toID)

	result, err := manager.TransferBalance(ctx, fromID, toID, decimal.NewFromFloat(30.0), "customer funding", "reseller@test.local", "127.0.0.1")
	require.NoError(t, err)
	require.True(t, result.Success)
	require.Equal(t, "100", result.FromOldBalance.String())
	require.Equal(t, "70", result.FromNewBalance.String())
	require.Equal(t, "10", result.ToOldBalance.String())
	require.Equal(t, "40", result.ToNewBalance.String())

	fromBal, err := manager.GetBalance(ctx, fromID)
	require.NoError(t, err)
	require.Equal(t, "70", fromBal.String())

	toBal, err := manager.GetBalance(ctx, toID)
	require.NoError(t, err)
	require.Equal(t, "40", toBal.String())

	require.Equal(t, 1, getBalanceHistoryCount(t, pool, fromID))
	require.Equal(t, 1, getBalanceHistoryCount(t, pool, toID))
}

// TestTransferBalance_Insufficient verifies that a transfer with insufficient
// funds fails entirely with no partial withdrawal.
func TestTransferBalance_Insufficient(t *testing.T) {
	ctx := context.Background()
	pool := setupTestDB(t)
	defer pool.Close()

	manager := NewManager(pool, zap.NewNop())

	fromID := createTestTenant(t, pool, decimal.NewFromFloat(10.0))
	toID := createTestTenant(t, pool, decimal.Zero)
	defer cleanupTestTenant(t, pool, fromID)
	defer cleanupTestTenant(t, pool, toID)

	_, err := manager.TransferBalance(ctx, fromID, toID, decimal.NewFromFloat(30.0), "too much", "reseller@test.local", "127.0.0.1")
	require.Error(t, err)
	var be *BalanceError
	require.ErrorAs(t, err, &be)
	require.Equal(t, ErrorInsufficientBalance, be.Type)

	// No partial withdrawal and no history entries.
	fromBal, err := manager.GetBalance(ctx, fromID)
	require.NoError(t, err)
	require.Equal(t, "10", fromBal.String())

	toBal, err := manager.GetBalance(ctx, toID)
	require.NoError(t, err)
	require.Equal(t, "0", toBal.String())

	require.Equal(t, 0, getBalanceHistoryCount(t, pool, fromID))
	require.Equal(t, 0, getBalanceHistoryCount(t, pool, toID))
}

// TestTransferBalance_DestinationNotFound verifies that a transfer to a missing
// tenant is rolled back entirely.
func TestTransferBalance_DestinationNotFound(t *testing.T) {
	ctx := context.Background()
	pool := setupTestDB(t)
	defer pool.Close()

	manager := NewManager(pool, zap.NewNop())

	fromID := createTestTenant(t, pool, decimal.NewFromFloat(100.0))
	defer cleanupTestTenant(t, pool, fromID)

	_, err := manager.TransferBalance(ctx, fromID, uuid.New(), decimal.NewFromFloat(30.0), "to missing", "reseller@test.local", "127.0.0.1")
	require.Error(t, err)
	var be *BalanceError
	require.ErrorAs(t, err, &be)
	require.Equal(t, ErrorTenantNotFound, be.Type)

	fromBal, err := manager.GetBalance(ctx, fromID)
	require.NoError(t, err)
	require.Equal(t, "100", fromBal.String())
	require.Equal(t, 0, getBalanceHistoryCount(t, pool, fromID))
}

// TestTransferBalance_SameTenant verifies self-transfers are rejected.
func TestTransferBalance_SameTenant(t *testing.T) {
	ctx := context.Background()
	pool := setupTestDB(t)
	defer pool.Close()

	manager := NewManager(pool, zap.NewNop())

	tenantID := createTestTenant(t, pool, decimal.NewFromFloat(100.0))
	defer cleanupTestTenant(t, pool, tenantID)

	_, err := manager.TransferBalance(ctx, tenantID, tenantID, decimal.NewFromFloat(10.0), "self", "reseller@test.local", "127.0.0.1")
	require.Error(t, err)

	bal, err := manager.GetBalance(ctx, tenantID)
	require.NoError(t, err)
	require.Equal(t, "100", bal.String())
}

// TestTransferBalance_Concurrent verifies that concurrent transfers from the
// same wallet never exceed the available balance: only one of two competing
// transfers succeeds and the final balance is exactly correct.
func TestTransferBalance_Concurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent transfer test in short mode")
	}

	ctx := context.Background()
	pool := setupTestDB(t)
	defer pool.Close()

	manager := NewManager(pool, zap.NewNop())

	fromID := createTestTenant(t, pool, decimal.NewFromFloat(100.0))
	toA := createTestTenant(t, pool, decimal.Zero)
	toB := createTestTenant(t, pool, decimal.Zero)
	defer cleanupTestTenant(t, pool, fromID)
	defer cleanupTestTenant(t, pool, toA)
	defer cleanupTestTenant(t, pool, toB)

	var wg sync.WaitGroup
	results := make(chan error, 2)

	for _, dest := range []uuid.UUID{toA, toB} {
		wg.Add(1)
		go func(dest uuid.UUID) {
			defer wg.Done()
			_, err := manager.TransferBalance(ctx, fromID, dest, decimal.NewFromFloat(60.0), "concurrent", "reseller@test.local", "127.0.0.1")
			results <- err
		}(dest)
	}

	wg.Wait()
	close(results)

	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}

	// Exactly one transfer (60) can succeed from a wallet holding 100.
	require.Equal(t, 1, successes)

	fromBal, err := manager.GetBalance(ctx, fromID)
	require.NoError(t, err)
	require.Equal(t, "40", fromBal.String())

	toABal, _ := manager.GetBalance(ctx, toA)
	toBBal, _ := manager.GetBalance(ctx, toB)
	require.Equal(t, "60", toABal.Add(toBBal).String())
}
