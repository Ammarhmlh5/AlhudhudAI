package dedicated_gateway_test

import (
	"context"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/dedicated_gateway"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockGatewayRepository - Mock for gateway operations
type MockGatewayRepository struct {
	mock.Mock
}

func (m *MockGatewayRepository) CreateGateway(ctx context.Context, gw *dedicated_gateway.Gateway) error {
	args := m.Called(ctx, gw)
	return args.Error(0)
}

func (m *MockGatewayRepository) GetGatewayByID(ctx context.Context, gwID uuid.UUID) (*dedicated_gateway.Gateway, error) {
	args := m.Called(ctx, gwID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dedicated_gateway.Gateway), args.Error(1)
}

func (m *MockGatewayRepository) GetGatewaysByTenant(ctx context.Context, tenantID uuid.UUID) ([]dedicated_gateway.Gateway, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dedicated_gateway.Gateway), args.Error(1)
}

func (m *MockGatewayRepository) UpdateGatewayStatus(ctx context.Context, gwID uuid.UUID, status string) error {
	args := m.Called(ctx, gwID, status)
	return args.Error(0)
}

func (m *MockGatewayRepository) UpdateGatewayQuota(ctx context.Context, gwID uuid.UUID, quota *dedicated_gateway.ResourceQuota) error {
	args := m.Called(ctx, gwID, quota)
	return args.Error(0)
}

func (m *MockGatewayRepository) RecordUsage(ctx context.Context, usage *dedicated_gateway.ResourceUsage) error {
	args := m.Called(ctx, usage)
	return args.Error(0)
}

func (m *MockGatewayRepository) GetUsageStats(ctx context.Context, gwID uuid.UUID, start, end time.Time) (*dedicated_gateway.UsageStats, error) {
	args := m.Called(ctx, gwID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dedicated_gateway.UsageStats), args.Error(1)
}

func TestGatewayManager_CreateGateway_Success(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	gw := &dedicated_gateway.Gateway{
		Name:        "My Gateway",
		Description: "Custom SMS gateway",
		Status:      "PENDING",
	}

	mockRepo := new(MockGatewayRepository)
	mockRepo.On("CreateGateway", mock.Anything, mock.AnythingOfType("*dedicated_gateway.Gateway")).Return(nil)

	manager := dedicated_gateway.NewManager(mockRepo)

	// Act
	result, err := manager.CreateGateway(context.Background(), tenantID, gw)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result.ID)
	assert.NotEmpty(t, result.APIKey)
	assert.Equal(t, "PENDING", result.Status)
	mockRepo.AssertExpectations(t)
}

func TestGatewayManager_ActivateGateway(t *testing.T) {
	// Arrange
	gwID := uuid.New()

	mockRepo := new(MockGatewayRepository)
	mockRepo.On("UpdateGatewayStatus", mock.Anything, gwID, "ACTIVE").Return(nil)

	manager := dedicated_gateway.NewManager(mockRepo)

	// Act
	err := manager.ActivateGateway(context.Background(), gwID)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestGatewayManager_GetGatewayByID(t *testing.T) {
	// Arrange
	gwID := uuid.New()
	expected := &dedicated_gateway.Gateway{
		ID:     gwID,
		Name:   "Test Gateway",
		Status: "ACTIVE",
	}

	mockRepo := new(MockGatewayRepository)
	mockRepo.On("GetGatewayByID", mock.Anything, gwID).Return(expected, nil)

	manager := dedicated_gateway.NewManager(mockRepo)

	// Act
	result, err := manager.GetGatewayByID(context.Background(), gwID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Test Gateway", result.Name)
	mockRepo.AssertExpectations(t)
}

func TestGatewayManager_SetResourceQuota(t *testing.T) {
	// Arrange
	gwID := uuid.New()
	quota := &dedicated_gateway.ResourceQuota{
		MaxMessagesPerDay:    10000,
		MaxAPIcallsPerMinute: 100,
		MaxConcurrentSessions: 5,
	}

	mockRepo := new(MockGatewayRepository)
	mockRepo.On("UpdateGatewayQuota", mock.Anything, gwID, quota).Return(nil)

	manager := dedicated_gateway.NewManager(mockRepo)

	// Act
	err := manager.SetResourceQuota(context.Background(), gwID, quota)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestGatewayManager_RecordUsage(t *testing.T) {
	// Arrange
	gwID := uuid.New()
	usage := &dedicated_gateway.ResourceUsage{
		GatewayID:  gwID,
		Date:       time.Now(),
		MessagesSent: 500,
		APICalls:    1000,
	}

	mockRepo := new(MockGatewayRepository)
	mockRepo.On("RecordUsage", mock.Anything, usage).Return(nil)

	manager := dedicated_gateway.NewManager(mockRepo)

	// Act
	err := manager.RecordUsage(context.Background(), usage)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestGatewayManager_GetUsageStats(t *testing.T) {
	// Arrange
	gwID := uuid.New()
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()

	stats := &dedicated_gateway.UsageStats{
		TotalMessages:  5000,
		TotalAPICalls:  10000,
		SuccessRate:    0.99,
		AverageLatency: 150,
	}

	mockRepo := new(MockGatewayRepository)
	mockRepo.On("GetUsageStats", mock.Anything, gwID, start, end).Return(stats, nil)

	manager := dedicated_gateway.NewManager(mockRepo)

	// Act
	result, err := manager.GetUsageStats(context.Background(), gwID, start, end)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 5000, result.TotalMessages)
	assert.Equal(t, 0.99, result.SuccessRate)
	mockRepo.AssertExpectations(t)
}

func TestGatewayManager_GenerateAPIKey(t *testing.T) {
	// Arrange
	manager := dedicated_gateway.NewManager(nil)

	// Act
	apiKey := manager.GenerateAPIKey()

	// Assert
	assert.NotEmpty(t, apiKey)
	assert.Len(t, apiKey, 64) // 32 bytes = 64 hex characters
}

func TestGatewayManager_ValidateQuota(t *testing.T) {
	// Arrange
	manager := dedicated_gateway.NewManager(nil)

	// Test within quota
	quota := &dedicated_gateway.ResourceQuota{
		MaxMessagesPerDay: 10000,
	}
	usage := &dedicated_gateway.ResourceUsage{
		MessagesSent: 5000,
	}
	assert.NoError(t, manager.ValidateQuota(quota, usage))

	// Test over quota
	usage = &dedicated_gateway.ResourceUsage{
		MessagesSent: 15000,
	}
	assert.Error(t, manager.ValidateQuota(quota, usage))
}

func TestGatewayManager_SuspendGateway(t *testing.T) {
	// Arrange
	gwID := uuid.New()

	mockRepo := new(MockGatewayRepository)
	mockRepo.On("UpdateGatewayStatus", mock.Anything, gwID, "SUSPENDED").Return(nil)

	manager := dedicated_gateway.NewManager(mockRepo)

	// Act
	err := manager.SuspendGateway(context.Background(), gwID, "Exceeded quota")

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestGatewayManager_GetGatewaysByTenant(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	gateways := []dedicated_gateway.Gateway{
		{ID: uuid.New(), Name: "Gateway 1", Status: "ACTIVE"},
		{ID: uuid.New(), Name: "Gateway 2", Status: "ACTIVE"},
	}

	mockRepo := new(MockGatewayRepository)
	mockRepo.On("GetGatewaysByTenant", mock.Anything, tenantID).Return(gateways, nil)

	manager := dedicated_gateway.NewManager(mockRepo)

	// Act
	result, err := manager.GetGatewaysByTenant(context.Background(), tenantID)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	mockRepo.AssertExpectations(t)
}