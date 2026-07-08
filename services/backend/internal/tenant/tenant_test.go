package tenant_test

import (
	"context"
	"testing"

	"github.com/al-hudhud/backend/internal/tenant"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTenantRepository - Mock for tenant operations
type MockTenantRepository struct {
	mock.Mock
}

func (m *MockTenantRepository) GetTenantByID(ctx context.Context, tenantID uuid.UUID) (*tenant.Tenant, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tenant.Tenant), args.Error(1)
}

func (m *MockTenantRepository) GetTenantByEmail(ctx context.Context, email string) (*tenant.Tenant, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tenant.Tenant), args.Error(1)
}

func (m *MockTenantRepository) CreateTenant(ctx context.Context, t *tenant.Tenant) error {
	args := m.Called(ctx, t)
	return args.Error(0)
}

func (m *MockTenantRepository) UpdateTenantStatus(ctx context.Context, tenantID uuid.UUID, status string) error {
	args := m.Called(ctx, tenantID, status)
	return args.Error(0)
}

func (m *MockTenantRepository) UpdateTenantBalance(ctx context.Context, tenantID uuid.UUID, newBalance decimal.Decimal) error {
	args := m.Called(ctx, tenantID, newBalance)
	return args.Error(0)
}

func (m *MockTenantRepository) DeleteTenant(ctx context.Context, tenantID uuid.UUID) error {
	args := m.Called(ctx, tenantID)
	return args.Error(0)
}

func (m *MockTenantRepository) ListTenants(ctx context.Context, limit, offset int) ([]tenant.Tenant, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]tenant.Tenant), args.Error(1)
}

func TestTenantManager_GetTenantByID_Success(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	expectedTenant := &tenant.Tenant{
		ID:     tenantID,
		Name:   "Test Company",
		Email:  "test@company.com",
		Balance: decimal.NewFromInt(5000),
		Status: "ACTIVE",
	}

	mockRepo := new(MockTenantRepository)
	mockRepo.On("GetTenantByID", mock.Anything, tenantID).Return(expectedTenant, nil)

	manager := tenant.NewManager(mockRepo)

	// Act
	result, err := manager.GetTenantByID(context.Background(), tenantID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedTenant.Name, result.Name)
	assert.Equal(t, expectedTenant.Email, result.Email)
	mockRepo.AssertExpectations(t)
}

func TestTenantManager_GetTenantByID_NotFound(t *testing.T) {
	// Arrange
	tenantID := uuid.New()

	mockRepo := new(MockTenantRepository)
	mockRepo.On("GetTenantByID", mock.Anything, tenantID).Return(nil, tenant.ErrTenantNotFound)

	manager := tenant.NewManager(mockRepo)

	// Act
	result, err := manager.GetTenantByID(context.Background(), tenantID)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, tenant.ErrTenantNotFound, err)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestTenantManager_CreateTenant_Success(t *testing.T) {
	// Arrange
	newTenant := &tenant.Tenant{
		Name:   "New Company",
		Email:  "new@company.com",
		Balance: decimal.NewFromInt(1000),
		Status: "ACTIVE",
	}

	mockRepo := new(MockTenantRepository)
	mockRepo.On("CreateTenant", mock.Anything, mock.AnythingOfType("*tenant.Tenant")).Return(nil)

	manager := tenant.NewManager(mockRepo)

	// Act
	err := manager.CreateTenant(context.Background(), newTenant)

	// Assert
	assert.NoError(t, err)
	assert.NotEmpty(t, newTenant.ID)
	mockRepo.AssertExpectations(t)
}

func TestTenantManager_ValidateTenantData(t *testing.T) {
	// Arrange
	manager := tenant.NewManager(nil)

	// Act & Assert - Valid data
	err := manager.ValidateTenantData("Valid Company", "valid@email.com")
	assert.NoError(t, err)

	// Act & Assert - Invalid email
	err = manager.ValidateTenantData("Valid Company", "invalid-email")
	assert.Error(t, err)

	// Act & Assert - Empty name
	err = manager.ValidateTenantData("", "valid@email.com")
	assert.Error(t, err)

	// Act & Assert - Empty email
	err = manager.ValidateTenantData("Valid Company", "")
	assert.Error(t, err)
}

func TestTenantManager_SuspendTenant(t *testing.T) {
	// Arrange
	tenantID := uuid.New()

	mockRepo := new(MockTenantRepository)
	mockRepo.On("UpdateTenantStatus", mock.Anything, tenantID, "SUSPENDED").Return(nil)

	manager := tenant.NewManager(mockRepo)

	// Act
	err := manager.SuspendTenant(context.Background(), tenantID)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestTenantManager_ActivateTenant(t *testing.T) {
	// Arrange
	tenantID := uuid.New()

	mockRepo := new(MockTenantRepository)
	mockRepo.On("UpdateTenantStatus", mock.Anything, tenantID, "ACTIVE").Return(nil)

	manager := tenant.NewManager(mockRepo)

	// Act
	err := manager.ActivateTenant(context.Background(), tenantID)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestTenantManager_ListTenants_Pagination(t *testing.T) {
	// Arrange
	tenants := []tenant.Tenant{
		{ID: uuid.New(), Name: "Company 1", Email: "c1@test.com"},
		{ID: uuid.New(), Name: "Company 2", Email: "c2@test.com"},
		{ID: uuid.New(), Name: "Company 3", Email: "c3@test.com"},
	}

	mockRepo := new(MockTenantRepository)
	mockRepo.On("ListTenants", mock.Anything, 10, 0).Return(tenants, nil)

	manager := tenant.NewManager(mockRepo)

	// Act
	result, err := manager.ListTenants(context.Background(), 10, 0)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, result, 3)
	mockRepo.AssertExpectations(t)
}

func TestTenantManager_DeleteTenant(t *testing.T) {
	// Arrange
	tenantID := uuid.New()

	mockRepo := new(MockTenantRepository)
	mockRepo.On("DeleteTenant", mock.Anything, tenantID).Return(nil)

	manager := tenant.NewManager(mockRepo)

	// Act
	err := manager.DeleteTenant(context.Background(), tenantID)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestTenantManager_GetTenantByEmail(t *testing.T) {
	// Arrange
	email := "existing@company.com"
	expectedTenant := &tenant.Tenant{
		ID:    uuid.New(),
		Name:  "Existing Company",
		Email: email,
	}

	mockRepo := new(MockTenantRepository)
	mockRepo.On("GetTenantByEmail", mock.Anything, email).Return(expectedTenant, nil)

	manager := tenant.NewManager(mockRepo)

	// Act
	result, err := manager.GetTenantByEmail(context.Background(), email)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedTenant.ID, result.ID)
	mockRepo.AssertExpectations(t)
}

func TestTenantManager_UpdateBalance(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	newBalance := decimal.NewFromInt(7500)

	mockRepo := new(MockTenantRepository)
	mockRepo.On("UpdateTenantBalance", mock.Anything, tenantID, newBalance).Return(nil)

	manager := tenant.NewManager(mockRepo)

	// Act
	err := manager.UpdateBalance(context.Background(), tenantID, newBalance)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestTenantManager_UpdateBalance_Negative(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	negativeBalance := decimal.NewFromInt(-100)

	manager := tenant.NewManager(nil)

	// Act
	err := manager.UpdateBalance(context.Background(), tenantID, negativeBalance)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, tenant.ErrInvalidBalance, err)
}