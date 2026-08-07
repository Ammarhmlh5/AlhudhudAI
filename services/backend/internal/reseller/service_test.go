package reseller_test

import (
	"context"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/balance"
	"github.com/al-hudhud/backend/internal/reseller"
	"github.com/al-hudhud/backend/internal/subscription"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const testDSN = "postgres://postgres@localhost:5432/sms_gateway_test?sslmode=disable"

func setupResellerTestDB(t *testing.T) (*pgxpool.Pool, *gorm.DB) {
	t.Helper()
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, testDSN)
	if err != nil {
		t.Skipf("Cannot connect to test database: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("Cannot ping test database: %v", err)
	}

	gdb, err := gorm.Open(postgres.Open(testDSN), &gorm.Config{})
	require.NoError(t, err)

	return pool, gdb
}

// createTestTenant inserts a user + tenant and returns the tenant ID.
func createTestTenant(t *testing.T, pool *pgxpool.Pool, isReseller bool, parentID *uuid.UUID) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	userID := uuid.New()
	email := "resellertest-" + uuid.New().String() + "@test.local"
	_, err := pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, user_type, is_active, is_verified, created_at, updated_at)
		VALUES ($1, $2, 'x', $3, true, true, NOW(), NOW())
	`, userID, email, "RESELLER")
	require.NoError(t, err)

	tenantID := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO tenants (id, user_id, business_name, contact_name, email, company_serial,
			api_key_accounting, api_key_phone, api_key_web, balance, status, is_active,
			is_reseller, parent_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'ACTIVE', true, $11, $12, NOW(), NOW())
	`,
		tenantID, userID,
		"Reseller Test "+uuid.New().String()[:8],
		"Test Contact",
		email,
		"RSL-"+tenantID.String(),
		"acc-"+tenantID.String(),
		"phone-"+tenantID.String(),
		"web-"+tenantID.String(),
		"1000",
		isReseller,
		parentID,
	)
	require.NoError(t, err)

	return tenantID
}

func cleanupTestTenant(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, "DELETE FROM tenants WHERE id = $1", tenantID)
	if err != nil {
		t.Logf("Warning: cleanup tenant failed: %v", err)
	}
}

type mockBalanceManager struct {
	mock.Mock
}

func (m *mockBalanceManager) DeductBalance(ctx context.Context, op balance.BalanceOperation) (*balance.BalanceResult, error) {
	args := m.Called(ctx, op)
	return args.Get(0).(*balance.BalanceResult), args.Error(1)
}
func (m *mockBalanceManager) DeductBalanceTx(ctx context.Context, tx pgx.Tx, op balance.BalanceOperation) (*balance.BalanceResult, error) {
	args := m.Called(ctx, tx, op)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*balance.BalanceResult), args.Error(1)
}
func (m *mockBalanceManager) DeductBalanceWithTimeout(ctx context.Context, op balance.BalanceOperation, timeout time.Duration) (*balance.BalanceResult, error) {
	return nil, nil
}
func (m *mockBalanceManager) RefundBalance(ctx context.Context, op balance.BalanceOperation) (*balance.BalanceResult, error) {
	return nil, nil
}
func (m *mockBalanceManager) RefundBalanceTx(ctx context.Context, tx pgx.Tx, op balance.BalanceOperation) (*balance.BalanceResult, error) {
	args := m.Called(ctx, tx, op)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*balance.BalanceResult), args.Error(1)
}
func (m *mockBalanceManager) GetBalance(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}
func (m *mockBalanceManager) CheckSufficient(ctx context.Context, tenantID uuid.UUID, amount decimal.Decimal) (bool, error) {
	args := m.Called(ctx, tenantID, amount)
	return args.Bool(0), args.Error(1)
}
func (m *mockBalanceManager) CreditBalance(ctx context.Context, op balance.BalanceOperation) (*balance.BalanceResult, error) {
	args := m.Called(ctx, op)
	return args.Get(0).(*balance.BalanceResult), args.Error(1)
}
func (m *mockBalanceManager) TransferBalance(ctx context.Context, from, to uuid.UUID, amount decimal.Decimal, reason, createdBy, ipAddress string) (*balance.TransferResult, error) {
	args := m.Called(ctx, from, to, amount, reason, createdBy, ipAddress)
	return args.Get(0).(*balance.TransferResult), args.Error(1)
}

type mockSubscriptionManager struct {
	mock.Mock
}

func (m *mockSubscriptionManager) Subscribe(ctx context.Context, req subscription.SubscribeRequest) (*subscription.SubscribeResult, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*subscription.SubscribeResult), args.Error(1)
}
func (m *mockSubscriptionManager) DeductMessages(ctx context.Context, req subscription.DeductMessagesRequest) (*subscription.DeductMessagesResult, error) {
	return nil, nil
}
func (m *mockSubscriptionManager) CalculateOverageCost(ctx context.Context, subscriptionID int64, overageMessages int64) (decimal.Decimal, error) {
	return decimal.Zero, nil
}
func (m *mockSubscriptionManager) GetUsage(ctx context.Context, tenantID uuid.UUID) (*subscription.UsageInfo, error) {
	return nil, nil
}
func (m *mockSubscriptionManager) GetActiveSubscription(ctx context.Context, tenantID uuid.UUID) (*subscription.Subscription, error) {
	return nil, nil
}
func (m *mockSubscriptionManager) GetPackage(ctx context.Context, packageID int64) (*subscription.Package, error) {
	return nil, nil
}

func newTestService(pool *pgxpool.Pool, gdb *gorm.DB, bm *mockBalanceManager, sm *mockSubscriptionManager) *reseller.Service {
	return reseller.NewService(gdb, pool, bm, sm, zap.NewNop())
}

func TestGetTenant_OwnershipEnforced(t *testing.T) {
	pool, gdb := setupResellerTestDB(t)
	defer pool.Close()

	resellerA := createTestTenant(t, pool, true, nil)
	resellerB := createTestTenant(t, pool, true, nil)
	customer := createTestTenant(t, pool, false, &resellerA)
	defer cleanupTestTenant(t, pool, customer)
	defer cleanupTestTenant(t, pool, resellerA)
	defer cleanupTestTenant(t, pool, resellerB)

	svc := newTestService(pool, gdb, new(mockBalanceManager), new(mockSubscriptionManager))
	ctx := context.Background()

	// Reseller A can read its own customer.
	got, err := svc.GetTenant(ctx, resellerA, customer)
	require.NoError(t, err)
	assert.Equal(t, customer, got.ID)

	// Reseller B cannot read A's customer.
	_, err = svc.GetTenant(ctx, resellerB, customer)
	assert.Error(t, err)
}

func TestListTenants_OnlyOwnCustomers(t *testing.T) {
	pool, gdb := setupResellerTestDB(t)
	defer pool.Close()

	resellerA := createTestTenant(t, pool, true, nil)
	resellerB := createTestTenant(t, pool, true, nil)
	customerA := createTestTenant(t, pool, false, &resellerA)
	customerB := createTestTenant(t, pool, false, &resellerB)
	defer cleanupTestTenant(t, pool, customerA)
	defer cleanupTestTenant(t, pool, customerB)
	defer cleanupTestTenant(t, pool, resellerA)
	defer cleanupTestTenant(t, pool, resellerB)

	svc := newTestService(pool, gdb, new(mockBalanceManager), new(mockSubscriptionManager))

	tenants, err := svc.ListTenants(context.Background(), resellerA)
	require.NoError(t, err)

	ids := make([]uuid.UUID, 0, len(tenants))
	for _, tn := range tenants {
		ids = append(ids, tn.ID)
	}
	assert.Contains(t, ids, customerA)
	assert.NotContains(t, ids, customerB)
}

func TestListResellers_ExcludesCustomers(t *testing.T) {
	pool, gdb := setupResellerTestDB(t)
	defer pool.Close()

	resellerA := createTestTenant(t, pool, true, nil)
	customer := createTestTenant(t, pool, false, &resellerA)
	defer cleanupTestTenant(t, pool, customer)
	defer cleanupTestTenant(t, pool, resellerA)

	svc := newTestService(pool, gdb, new(mockBalanceManager), new(mockSubscriptionManager))

	resellers, err := svc.ListResellers(context.Background())
	require.NoError(t, err)

	for _, r := range resellers {
		if r.ID == resellerA {
			assert.Equal(t, int64(1), r.ConnectedCompanies)
		}
		assert.NotEqual(t, customer, r.ID)
	}
}

func TestTransferToTenant_Success(t *testing.T) {
	pool, gdb := setupResellerTestDB(t)
	defer pool.Close()

	resellerA := createTestTenant(t, pool, true, nil)
	customer := createTestTenant(t, pool, false, &resellerA)
	defer cleanupTestTenant(t, pool, customer)
	defer cleanupTestTenant(t, pool, resellerA)

	bm := new(mockBalanceManager)
	bm.On("TransferBalance", mock.Anything, resellerA, customer, decimal.NewFromInt(500), "funding", "admin@test", "127.0.0.1").
		Return(&balance.TransferResult{Success: true, Amount: decimal.NewFromInt(500)}, nil)

	svc := newTestService(pool, gdb, bm, new(mockSubscriptionManager))

	result, err := svc.TransferToTenant(context.Background(), resellerA, customer,
		decimal.NewFromInt(500), "funding", "admin@test", "127.0.0.1")
	require.NoError(t, err)
	assert.True(t, result.Success)
	bm.AssertExpectations(t)
}

func TestTransferToTenant_OwnershipDenied(t *testing.T) {
	pool, gdb := setupResellerTestDB(t)
	defer pool.Close()

	resellerA := createTestTenant(t, pool, true, nil)
	resellerB := createTestTenant(t, pool, true, nil)
	customer := createTestTenant(t, pool, false, &resellerA)
	defer cleanupTestTenant(t, pool, customer)
	defer cleanupTestTenant(t, pool, resellerA)
	defer cleanupTestTenant(t, pool, resellerB)

	bm := new(mockBalanceManager)
	svc := newTestService(pool, gdb, bm, new(mockSubscriptionManager))

	_, err := svc.TransferToTenant(context.Background(), resellerB, customer,
		decimal.NewFromInt(500), "funding", "admin@test", "127.0.0.1")
	assert.Error(t, err)
	bm.AssertNotCalled(t, "TransferBalance", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestCreditReseller_NonResellerRejected(t *testing.T) {
	pool, gdb := setupResellerTestDB(t)
	defer pool.Close()

	customer := createTestTenant(t, pool, false, nil)
	defer cleanupTestTenant(t, pool, customer)

	bm := new(mockBalanceManager)
	svc := newTestService(pool, gdb, bm, new(mockSubscriptionManager))

	_, err := svc.CreditReseller(context.Background(), customer, decimal.NewFromInt(1000), "fund", "admin@test", "127.0.0.1")
	assert.Error(t, err)
	bm.AssertNotCalled(t, "CreditBalance", mock.Anything, mock.Anything)
}

func TestCreditReseller_Success(t *testing.T) {
	pool, gdb := setupResellerTestDB(t)
	defer pool.Close()

	resellerA := createTestTenant(t, pool, true, nil)
	defer cleanupTestTenant(t, pool, resellerA)

	bm := new(mockBalanceManager)
	bm.On("CreditBalance", mock.Anything, mock.MatchedBy(func(op balance.BalanceOperation) bool {
		return op.TenantID == resellerA && op.Amount.Equal(decimal.NewFromInt(1000))
	})).Return(&balance.BalanceResult{Success: true, NewBalance: decimal.NewFromInt(2000)}, nil)

	svc := newTestService(pool, gdb, bm, new(mockSubscriptionManager))

	result, err := svc.CreditReseller(context.Background(), resellerA, decimal.NewFromInt(1000), "fund", "admin@test", "127.0.0.1")
	require.NoError(t, err)
	assert.True(t, result.Success)
	bm.AssertExpectations(t)
}

func TestCreditReseller_MaxCreditEnforced(t *testing.T) {
	pool, gdb := setupResellerTestDB(t)
	defer pool.Close()

	resellerA := createTestTenant(t, pool, true, nil)
	defer cleanupTestTenant(t, pool, resellerA)

	t.Setenv("RESELLER_MAX_CREDIT", "500")

	bm := new(mockBalanceManager)
	svc := newTestService(pool, gdb, bm, new(mockSubscriptionManager))

	_, err := svc.CreditReseller(context.Background(), resellerA, decimal.NewFromInt(1000), "fund", "admin@test", "127.0.0.1")
	require.ErrorIs(t, err, reseller.ErrCreditLimitExceeded)
	bm.AssertNotCalled(t, "CreditBalance", mock.Anything, mock.Anything)
}

func TestCreditReseller_MaxCreditAtLimitAllowed(t *testing.T) {
	pool, gdb := setupResellerTestDB(t)
	defer pool.Close()

	resellerA := createTestTenant(t, pool, true, nil)
	defer cleanupTestTenant(t, pool, resellerA)

	t.Setenv("RESELLER_MAX_CREDIT", "500")

	bm := new(mockBalanceManager)
	bm.On("CreditBalance", mock.Anything, mock.MatchedBy(func(op balance.BalanceOperation) bool {
		return op.TenantID == resellerA && op.Amount.Equal(decimal.NewFromInt(500))
	})).Return(&balance.BalanceResult{Success: true, NewBalance: decimal.NewFromInt(1500)}, nil)

	svc := newTestService(pool, gdb, bm, new(mockSubscriptionManager))

	result, err := svc.CreditReseller(context.Background(), resellerA, decimal.NewFromInt(500), "fund", "admin@test", "127.0.0.1")
	require.NoError(t, err)
	assert.True(t, result.Success)
	bm.AssertExpectations(t)
}

func TestCreditReseller_MaxCreditUnlimitedByDefault(t *testing.T) {
	pool, gdb := setupResellerTestDB(t)
	defer pool.Close()

	resellerA := createTestTenant(t, pool, true, nil)
	defer cleanupTestTenant(t, pool, resellerA)

	bm := new(mockBalanceManager)
	bm.On("CreditBalance", mock.Anything, mock.MatchedBy(func(op balance.BalanceOperation) bool {
		return op.TenantID == resellerA
	})).Return(&balance.BalanceResult{Success: true, NewBalance: decimal.NewFromInt(1000000)}, nil)

	svc := newTestService(pool, gdb, bm, new(mockSubscriptionManager))

	result, err := svc.CreditReseller(context.Background(), resellerA, decimal.NewFromInt(999999), "fund", "admin@test", "127.0.0.1")
	require.NoError(t, err)
	assert.True(t, result.Success)
	bm.AssertExpectations(t)
}

func TestSubscribeTenant_Success(t *testing.T) {
	pool, gdb := setupResellerTestDB(t)
	defer pool.Close()

	resellerA := createTestTenant(t, pool, true, nil)
	customer := createTestTenant(t, pool, false, &resellerA)
	defer cleanupTestTenant(t, pool, customer)
	defer cleanupTestTenant(t, pool, resellerA)

	sm := new(mockSubscriptionManager)
	sm.On("Subscribe", mock.Anything, mock.MatchedBy(func(req subscription.SubscribeRequest) bool {
		return req.TenantID == customer && req.PackageID == 7 && req.AutoRenew
	})).Return(&subscription.SubscribeResult{
		Success:        true,
		SubscriptionID: 42,
		Message:        "Subscribed",
		NewBalance:     decimal.NewFromInt(500),
	}, nil)

	svc := newTestService(pool, gdb, new(mockBalanceManager), sm)

	result, err := svc.SubscribeTenant(context.Background(), resellerA, customer, 7, true)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, int64(42), result.SubscriptionID)
	sm.AssertExpectations(t)
}

func TestSubscribeTenant_OwnershipDenied(t *testing.T) {
	pool, gdb := setupResellerTestDB(t)
	defer pool.Close()

	resellerA := createTestTenant(t, pool, true, nil)
	resellerB := createTestTenant(t, pool, true, nil)
	customer := createTestTenant(t, pool, false, &resellerA)
	defer cleanupTestTenant(t, pool, customer)
	defer cleanupTestTenant(t, pool, resellerA)
	defer cleanupTestTenant(t, pool, resellerB)

	sm := new(mockSubscriptionManager)
	svc := newTestService(pool, gdb, new(mockBalanceManager), sm)

	_, err := svc.SubscribeTenant(context.Background(), resellerB, customer, 7, true)
	assert.Error(t, err)
	sm.AssertNotCalled(t, "Subscribe", mock.Anything, mock.Anything)
}

func TestGetTenantBalance_Success(t *testing.T) {
	pool, gdb := setupResellerTestDB(t)
	defer pool.Close()

	resellerA := createTestTenant(t, pool, true, nil)
	customer := createTestTenant(t, pool, false, &resellerA)
	defer cleanupTestTenant(t, pool, customer)
	defer cleanupTestTenant(t, pool, resellerA)

	bm := new(mockBalanceManager)
	bm.On("GetBalance", mock.Anything, customer).Return(decimal.NewFromInt(1234), nil)

	svc := newTestService(pool, gdb, bm, new(mockSubscriptionManager))

	bal, err := svc.GetTenantBalance(context.Background(), resellerA, customer)
	require.NoError(t, err)
	assert.True(t, bal.Equal(decimal.NewFromInt(1234)))
	bm.AssertExpectations(t)
}

func TestGetWalletBalance_Success(t *testing.T) {
	pool, gdb := setupResellerTestDB(t)
	defer pool.Close()

	resellerA := createTestTenant(t, pool, true, nil)
	defer cleanupTestTenant(t, pool, resellerA)

	bm := new(mockBalanceManager)
	bm.On("GetBalance", mock.Anything, resellerA).Return(decimal.NewFromInt(5000), nil)

	svc := newTestService(pool, gdb, bm, new(mockSubscriptionManager))

	bal, err := svc.GetWalletBalance(context.Background(), resellerA)
	require.NoError(t, err)
	assert.True(t, bal.Equal(decimal.NewFromInt(5000)))
	bm.AssertExpectations(t)
}
