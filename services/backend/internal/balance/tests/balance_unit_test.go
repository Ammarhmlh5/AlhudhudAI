package balance_test

import (
	"context"
	"testing"

	"github.com/al-hudhud/backend/internal/balance"
	"github.com/al-hudhud/backend/internal/testutils"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockBalanceRepository - Mock for database operations
type MockBalanceRepository struct {
	mock.Mock
}

func (m *MockBalanceRepository) GetBalance(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return decimal.Zero, args.Error(1)
	}
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

func (m *MockBalanceRepository) DeductBalance(ctx context.Context, tenantID uuid.UUID, amount decimal.Decimal) error {
	args := m.Called(ctx, tenantID, amount)
	return args.Error(0)
}

func (m *MockBalanceRepository) AddBalance(ctx context.Context, tenantID uuid.UUID, amount decimal.Decimal) error {
	args := m.Called(ctx, tenantID, amount)
	return args.Error(0)
}

func (m *MockBalanceRepository) GetBalanceHistory(ctx context.Context, tenantID uuid.UUID, limit int) ([]balance.BalanceHistory, error) {
	args := m.Called(ctx, tenantID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]balance.BalanceHistory), args.Error(1)
}

// TestMockBalanceRepository_Setup tests that mock repository can be created and configured
func TestMockBalanceRepository_Setup(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	expectedBalance := decimal.NewFromInt(1000)
	
	mockRepo := new(MockBalanceRepository)
	
	// Configure mock behavior
	mockRepo.On("GetBalance", mock.Anything, tenantID).Return(expectedBalance, nil)
	
	// Act - Call the mock directly
	result, err := mockRepo.GetBalance(context.Background(), tenantID)
	
	// Assert
	assert.NoError(t, err)
	assert.True(t, result.Equal(expectedBalance))
	mockRepo.AssertExpectations(t)
}

// TestMockBalanceRepository_DeductBalance tests mock deduct behavior
func TestMockBalanceRepository_DeductBalance(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	amount := decimal.NewFromInt(100)
	
	mockRepo := new(MockBalanceRepository)
	mockRepo.On("DeductBalance", mock.Anything, tenantID, amount).Return(nil)
	
	// Act
	err := mockRepo.DeductBalance(context.Background(), tenantID, amount)
	
	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// TestMockBalanceRepository_InsufficientFunds tests mock insufficient funds error
func TestMockBalanceRepository_InsufficientFunds(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	amount := decimal.NewFromInt(10000)
	
	mockRepo := new(MockBalanceRepository)
	mockRepo.On("DeductBalance", mock.Anything, tenantID, amount).Return(&balance.BalanceError{
		Type: balance.ErrorInsufficientBalance,
		Message: "insufficient funds",
	})
	
	// Act
	err := mockRepo.DeductBalance(context.Background(), tenantID, amount)
	
	// Assert
	assert.Error(t, err)
	assert.Equal(t, balance.ErrorInsufficientBalance, err.(*balance.BalanceError).Type)
	mockRepo.AssertExpectations(t)
}

// TestMockBalanceRepository_AddBalance tests mock add behavior
func TestMockBalanceRepository_AddBalance(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	amount := decimal.NewFromInt(500)
	
	mockRepo := new(MockBalanceRepository)
	mockRepo.On("AddBalance", mock.Anything, tenantID, amount).Return(nil)
	
	// Act
	err := mockRepo.AddBalance(context.Background(), tenantID, amount)
	
	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// TestMockBalanceRepository_ConcurrentAccess tests mock concurrent access
func TestMockBalanceRepository_ConcurrentAccess(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	amount := decimal.NewFromInt(50)
	
	mockRepo := new(MockBalanceRepository)
	mockRepo.On("DeductBalance", mock.Anything, tenantID, amount).Return(nil)
	
	// Act - Simulate concurrent calls
	errChan := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func() {
			err := mockRepo.DeductBalance(context.Background(), tenantID, amount)
			errChan <- err
		}()
	}
	
	// Assert
	for i := 0; i < 10; i++ {
		assert.NoError(t, <-errChan)
	}
	mockRepo.AssertNumberOfCalls(t, "DeductBalance", 10)
}

// TestMockBalanceRepository_GetBalanceHistory tests mock history retrieval
func TestMockBalanceRepository_GetBalanceHistory(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	limit := 10
	
	mockRepo := new(MockBalanceRepository)
	mockRepo.On("GetBalanceHistory", mock.Anything, tenantID, limit).Return([]balance.BalanceHistory{
		{ID: uuid.New(), Type: balance.OperationTypeDeduct, Amount: decimal.NewFromInt(-10)},
		{ID: uuid.New(), Type: balance.OperationTypeCharge, Amount: decimal.NewFromInt(100)},
	}, nil)
	
	// Act
	result, err := mockRepo.GetBalanceHistory(context.Background(), tenantID, limit)
	
	// Assert
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	mockRepo.AssertExpectations(t)
}

// TestBalanceTypes tests the balance types and constants
func TestBalanceTypes(t *testing.T) {
	// Test OperationType constants
	assert.Equal(t, balance.OperationTypeDeduct, balance.OperationType("DEDUCT"))
	assert.Equal(t, balance.OperationTypeRefund, balance.OperationType("REFUND"))
	assert.Equal(t, balance.OperationTypeCharge, balance.OperationType("CHARGE"))
	assert.Equal(t, balance.OperationTypeAdjustment, balance.OperationType("ADJUSTMENT"))
	
	// Test BalanceOperation structure
	op := balance.BalanceOperation{
		TenantID:  uuid.New(),
		Amount:    decimal.NewFromInt(100),
		Type:      balance.OperationTypeDeduct,
		Reason:    "Test operation",
	}
	assert.NotEqual(t, uuid.Nil, op.TenantID)
	assert.True(t, op.Amount.Equal(decimal.NewFromInt(100)))
	assert.Equal(t, balance.OperationTypeDeduct, op.Type)
	
	// Test BalanceResult structure
	result := balance.BalanceResult{
		Success:    true,
		OldBalance: decimal.NewFromInt(1000),
		NewBalance: decimal.NewFromInt(900),
	}
	assert.True(t, result.Success)
	assert.True(t, result.OldBalance.Equal(decimal.NewFromInt(1000)))
	assert.True(t, result.NewBalance.Equal(decimal.NewFromInt(900)))
}

// TestBalanceErrorTypes tests error types
func TestBalanceErrorTypes(t *testing.T) {
	assert.Equal(t, balance.ErrorInsufficientBalance, balance.ErrorType("INSUFFICIENT_BALANCE"))
	assert.Equal(t, balance.ErrorTenantNotFound, balance.ErrorType("TENANT_NOT_FOUND"))
	assert.Equal(t, balance.ErrorConcurrentUpdate, balance.ErrorType("CONCURRENT_UPDATE"))
	assert.Equal(t, balance.ErrorDatabaseError, balance.ErrorType("DATABASE_ERROR"))
}

// TestIntegrationWithTestDB tests integration with test utilities (requires database)
func TestIntegrationWithTestDB(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// This test requires a real database connection
	testDB, err := testutils.NewTestDB()
	if err != nil {
		t.Skip("Skipping: No test database available")
	}
	defer testDB.Close()
	
	// Create test tenant
	tenantID := testDB.CreateTestTenant()
	
	// Verify tenant was created
	assert.NotEqual(t, uuid.Nil, tenantID)
	
	// Test getting balance using raw query
	var balance float64
	err = testDB.QueryRow("SELECT balance FROM tenants WHERE id = $1", tenantID).Scan(&balance)
	assert.NoError(t, err)
	assert.True(t, balance >= 0)
}