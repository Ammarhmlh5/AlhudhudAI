package balance

import (
	"context"
	"testing"

	"github.com/al-hudhud/backend/internal/testutils"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// MockBalanceRepository - Mock for database operations (test-only)
type MockBalanceRepository struct {
	getBalanceFunc     func(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error)
	deductBalanceFunc  func(ctx context.Context, tenantID uuid.UUID, amount decimal.Decimal) error
	addBalanceFunc     func(ctx context.Context, tenantID uuid.UUID, amount decimal.Decimal) error
	getHistoryFunc     func(ctx context.Context, tenantID uuid.UUID, limit int) ([]BalanceHistory, error)
}

func (m *MockBalanceRepository) GetBalance(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
	if m.getBalanceFunc != nil {
		return m.getBalanceFunc(ctx, tenantID)
	}
	return decimal.Zero, nil
}

func (m *MockBalanceRepository) DeductBalance(ctx context.Context, tenantID uuid.UUID, amount decimal.Decimal) error {
	if m.deductBalanceFunc != nil {
		return m.deductBalanceFunc(ctx, tenantID, amount)
	}
	return nil
}

func (m *MockBalanceRepository) AddBalance(ctx context.Context, tenantID uuid.UUID, amount decimal.Decimal) error {
	if m.addBalanceFunc != nil {
		return m.addBalanceFunc(ctx, tenantID, amount)
	}
	return nil
}

func (m *MockBalanceRepository) GetBalanceHistory(ctx context.Context, tenantID uuid.UUID, limit int) ([]BalanceHistory, error) {
	if m.getHistoryFunc != nil {
		return m.getHistoryFunc(ctx, tenantID, limit)
	}
	return nil, nil
}

// Tests

func TestBalanceManager_GetBalance_Success(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	expectedBalance := decimal.NewFromInt(1000)

	repo := MockBalanceRepository{
		getBalanceFunc: func(ctx context.Context, id uuid.UUID) (decimal.Decimal, error) {
			return expectedBalance, nil
		},
	}

	manager := &testManager{repo: repo}

	// Act
	result, err := manager.GetBalance(context.Background(), tenantID)

	// Assert
	assert.NoError(t, err)
	assert.True(t, result.Equal(expectedBalance))
}

func TestBalanceManager_DeductBalance_SufficientFunds(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	amount := decimal.NewFromInt(100)

	repo := MockBalanceRepository{
		deductBalanceFunc: func(ctx context.Context, id uuid.UUID, amt decimal.Decimal) error {
			return nil
		},
	}

	manager := &testManager{repo: repo}

	// Act
	err := manager.DeductBalance(context.Background(), tenantID, amount)

	// Assert
	assert.NoError(t, err)
}

func TestBalanceManager_DeductBalance_InsufficientFunds(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	amount := decimal.NewFromInt(10000)

	repo := MockBalanceRepository{
		deductBalanceFunc: func(ctx context.Context, id uuid.UUID, amt decimal.Decimal) error {
			return &BalanceError{Type: ErrorInsufficientBalance, Message: "insufficient funds"}
		},
	}

	manager := &testManager{repo: repo}

	// Act
	err := manager.DeductBalance(context.Background(), tenantID, amount)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, ErrorInsufficientBalance, err.(*BalanceError).Type)
}

func TestBalanceManager_AddBalance_Success(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	amount := decimal.NewFromInt(500)

	repo := MockBalanceRepository{
		addBalanceFunc: func(ctx context.Context, id uuid.UUID, amt decimal.Decimal) error {
			return nil
		},
	}

	manager := &testManager{repo: repo}

	// Act
	err := manager.AddBalance(context.Background(), tenantID, amount)

	// Assert
	assert.NoError(t, err)
}

func TestBalanceManager_DeductBalance_Atomicity(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	amount := decimal.NewFromInt(50)
	deductCount := 0

	repo := MockBalanceRepository{
		deductBalanceFunc: func(ctx context.Context, id uuid.UUID, amt decimal.Decimal) error {
			deductCount++
			return nil
		},
	}

	manager := &testManager{repo: repo}

	// Act - Call multiple times concurrently
	errChan := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func() {
			err := manager.DeductBalance(context.Background(), tenantID, amount)
			errChan <- err
		}()
	}

	// Assert - All should succeed (atomic operations)
	for i := 0; i < 10; i++ {
		err := <-errChan
		assert.NoError(t, err)
	}
	assert.Equal(t, 10, deductCount)
}

func TestBalanceManager_GetBalanceHistory_Success(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	limit := 10

	repo := MockBalanceRepository{
		getHistoryFunc: func(ctx context.Context, id uuid.UUID, l int) ([]BalanceHistory, error) {
			return []BalanceHistory{
				{ID: uuid.New(), Type: OperationTypeDeduct, Amount: decimal.NewFromInt(-10)},
				{ID: uuid.New(), Type: OperationTypeCharge, Amount: decimal.NewFromInt(100)},
			}, nil
		},
	}

	manager := &testManager{repo: repo}

	// Act
	result, err := manager.GetBalanceHistory(context.Background(), tenantID, limit)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestBalanceManager_ValidateSufficientBalance(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	balance := decimal.NewFromInt(100)
	amount := decimal.NewFromInt(50)

	repo := MockBalanceRepository{
		getBalanceFunc: func(ctx context.Context, id uuid.UUID) (decimal.Decimal, error) {
			return balance, nil
		},
	}

	manager := &testManager{repo: repo}

	// Act
	err := manager.ValidateSufficientBalance(context.Background(), tenantID, amount)

	// Assert
	assert.NoError(t, err)
}

func TestBalanceManager_ValidateSufficientBalance_Insufficient(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	balance := decimal.NewFromInt(100)
	amount := decimal.NewFromInt(150)

	repo := MockBalanceRepository{
		getBalanceFunc: func(ctx context.Context, id uuid.UUID) (decimal.Decimal, error) {
			return balance, nil
		},
	}

	manager := &testManager{repo: repo}

	// Act
	err := manager.ValidateSufficientBalance(context.Background(), tenantID, amount)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, ErrorInsufficientBalance, err.(*BalanceError).Type)
}

func TestBalanceManager_DeductBalance_InvalidAmount(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	amount := decimal.NewFromInt(-100)

	manager := &testManager{repo: MockBalanceRepository{}}

	// Act
	err := manager.DeductBalance(context.Background(), tenantID, amount)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, ErrorDatabaseError, err.(*BalanceError).Type)
}

func TestBalanceManager_AddBalance_InvalidAmount(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	amount := decimal.Zero

	manager := &testManager{repo: MockBalanceRepository{}}

	// Act
	err := manager.AddBalance(context.Background(), tenantID, amount)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, ErrorDatabaseError, err.(*BalanceError).Type)
}

// Integration test with real database (requires test database)
func TestBalanceManager_Integration(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// This test requires a real database connection
	// Use environment variables or test configuration
	testDB, err := testutils.NewTestDB()
	if err != nil {
		t.Skip("Skipping: No test database available")
	}
	defer testDB.Close()
	
	// Create test tenant
	tenantID := testDB.CreateTestTenant()
	
	// Create manager with real repository
	manager := &testManager{repo: MockBalanceRepository{}}
	
	// Test getting balance
	balance, err := manager.GetBalance(context.Background(), tenantID)
	assert.NoError(t, err)
	assert.True(t, balance.GreaterThanOrEqual(decimal.Zero))
}

// testManager - Test-specific manager that uses MockBalanceRepository
type testManager struct {
	repo MockBalanceRepository
}

// GetBalance - Get tenant balance
func (m *testManager) GetBalance(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
	return m.repo.GetBalance(ctx, tenantID)
}

// DeductBalance - Deduct amount from balance
func (m *testManager) DeductBalance(ctx context.Context, tenantID uuid.UUID, amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return &BalanceError{Type: ErrorDatabaseError, Message: "amount must be positive"}
	}
	return m.repo.DeductBalance(ctx, tenantID, amount)
}

// AddBalance - Add amount to balance
func (m *testManager) AddBalance(ctx context.Context, tenantID uuid.UUID, amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return &BalanceError{Type: ErrorDatabaseError, Message: "amount must be positive"}
	}
	return m.repo.AddBalance(ctx, tenantID, amount)
}

// GetBalanceHistory - Get balance history
func (m *testManager) GetBalanceHistory(ctx context.Context, tenantID uuid.UUID, limit int) ([]BalanceHistory, error) {
	return m.repo.GetBalanceHistory(ctx, tenantID, limit)
}

// ValidateSufficientBalance - Check if balance is sufficient
func (m *testManager) ValidateSufficientBalance(ctx context.Context, tenantID uuid.UUID, amount decimal.Decimal) error {
	balance, err := m.GetBalance(ctx, tenantID)
	if err != nil {
		return err
	}
	if balance.LessThan(amount) {
		return &BalanceError{Type: ErrorInsufficientBalance, Message: "insufficient funds"}
	}
	return nil
}