package subscription_test

import (
	"context"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/subscription"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSubscriptionRepository - Mock for subscription operations
type MockSubscriptionRepository struct {
	mock.Mock
}

func (m *MockSubscriptionRepository) CreateSubscription(ctx context.Context, sub *subscription.Subscription) error {
	args := m.Called(ctx, sub)
	return args.Error(0)
}

func (m *MockSubscriptionRepository) GetSubscriptionByID(ctx context.Context, subID uuid.UUID) (*subscription.Subscription, error) {
	args := m.Called(ctx, subID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*subscription.Subscription), args.Error(1)
}

func (m *MockSubscriptionRepository) GetActiveSubscription(ctx context.Context, tenantID uuid.UUID) (*subscription.Subscription, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*subscription.Subscription), args.Error(1)
}

func (m *MockSubscriptionRepository) UpdateSubscriptionStatus(ctx context.Context, subID uuid.UUID, status string) error {
	args := m.Called(ctx, subID, status)
	return args.Error(0)
}

func (m *MockSubscriptionRepository) CancelSubscription(ctx context.Context, subID uuid.UUID, reason string) error {
	args := m.Called(ctx, subID, reason)
	return args.Error(0)
}

func (m *MockSubscriptionRepository) GetSubscriptionsByTenant(ctx context.Context, tenantID uuid.UUID) ([]subscription.Subscription, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]subscription.Subscription), args.Error(1)
}

func TestSubscriptionService_CreateSubscription_Success(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	sub := &subscription.Subscription{
		TenantID:     tenantID,
		PackageID:    1,
		Status:       "PENDING",
		StartDate:     time.Now(),
		EndDate:    time.Now().Add(30 * 24 * time.Hour),
	}

	mockRepo := new(MockSubscriptionRepository)
	mockRepo.On("CreateSubscription", mock.Anything, mock.AnythingOfType("*subscription.Subscription")).Return(nil)

	service := subscription.NewService(mockRepo)

	// Act
	result, err := service.CreateSubscription(context.Background(), sub)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result.ID)
	assert.Equal(t, subscription.SubscriptionStatus("PENDING"), result.Status)
	mockRepo.AssertExpectations(t)
}

func TestSubscriptionService_ActivateSubscription(t *testing.T) {
	// Arrange
	subID := uuid.New()

	mockRepo := new(MockSubscriptionRepository)
	mockRepo.On("UpdateSubscriptionStatus", mock.Anything, subID, "ACTIVE").Return(nil)

	service := subscription.NewService(mockRepo)

	// Act
	err := service.ActivateSubscription(context.Background(), subID)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSubscriptionService_GetActiveSubscription(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	expectedSub := &subscription.Subscription{
		ID:         1,
		TenantID:   tenantID,
		PackageID:  1,
		Status:     "ACTIVE",
		StartDate:   time.Now().Add(-15 * 24 * time.Hour),
		EndDate:  time.Now().Add(15 * 24 * time.Hour),
	}

	mockRepo := new(MockSubscriptionRepository)
	mockRepo.On("GetActiveSubscription", mock.Anything, tenantID).Return(expectedSub, nil)

	service := subscription.NewService(mockRepo)

	// Act
	result, err := service.GetActiveSubscription(context.Background(), tenantID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, subscription.SubscriptionStatus("ACTIVE"), result.Status)
	mockRepo.AssertExpectations(t)
}

func TestSubscriptionService_GetActiveSubscription_Expired(t *testing.T) {
	// Arrange
	tenantID := uuid.New()

	mockRepo := new(MockSubscriptionRepository)
	mockRepo.On("GetActiveSubscription", mock.Anything, tenantID).Return(nil, subscription.ErrNoActiveSubscription)

	service := subscription.NewService(mockRepo)

	// Act
	result, err := service.GetActiveSubscription(context.Background(), tenantID)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, subscription.ErrNoActiveSubscription, err)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestSubscriptionService_CancelSubscription(t *testing.T) {
	// Arrange
	subID := uuid.New()
	reason := "Customer requested cancellation"

	mockRepo := new(MockSubscriptionRepository)
	mockRepo.On("CancelSubscription", mock.Anything, subID, reason).Return(nil)

	service := subscription.NewService(mockRepo)

	// Act
	err := service.CancelSubscription(context.Background(), subID, reason)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSubscriptionService_RenewSubscription(t *testing.T) {
	// Arrange
	subID := uuid.New()
	renewalPeriod := 30 * 24 * time.Hour

	mockRepo := new(MockSubscriptionRepository)
	mockRepo.On("GetSubscriptionByID", mock.Anything, subID).Return(&subscription.Subscription{
		ID:      1,
		Status:  "ACTIVE",
		EndDate: time.Now().Add(5 * 24 * time.Hour),
	}, nil)
	mockRepo.On("UpdateSubscriptionStatus", mock.Anything, subID, "ACTIVE").Return(nil)

	service := subscription.NewService(mockRepo)

	// Act
	err := service.RenewSubscription(context.Background(), subID, renewalPeriod)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSubscriptionService_CheckAutoRenewal(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	expiringSub := &subscription.Subscription{
		ID:        1,
		TenantID:  tenantID,
		Status:    "ACTIVE",
		AutoRenew: true,
		EndDate:   time.Now().Add(6 * 24 * time.Hour), // Expires in 6 days (within the 7-day renewal window)
		PackageID: 1,
	}

	mockRepo := new(MockSubscriptionRepository)
	mockRepo.On("GetActiveSubscription", mock.Anything, tenantID).Return(expiringSub, nil)

	service := subscription.NewService(mockRepo)

	// Act
	needsRenewal, err := service.CheckAutoRenewal(context.Background(), tenantID)

	// Assert
	assert.NoError(t, err)
	assert.True(t, needsRenewal)
	mockRepo.AssertExpectations(t)
}

func TestSubscriptionService_UpgradePackage(t *testing.T) {
	// Arrange
	subID := uuid.New()
	newPackageID := int64(2)
	upgradePrice := decimal.NewFromFloat(50.00)

	mockRepo := new(MockSubscriptionRepository)
	mockRepo.On("UpdateSubscriptionStatus", mock.Anything, subID, "ACTIVE").Return(nil)

	service := subscription.NewService(mockRepo)

	// Act
	err := service.UpgradePackage(context.Background(), subID, newPackageID, upgradePrice)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSubscriptionService_GetSubscriptionHistory(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	subscriptions := []subscription.Subscription{
		{ID: 1, Status: "EXPIRED", PackageID: 1},
		{ID: 2, Status: "CANCELLED", PackageID: 2},
		{ID: 3, Status: "ACTIVE", PackageID: 3},
	}

	mockRepo := new(MockSubscriptionRepository)
	mockRepo.On("GetSubscriptionsByTenant", mock.Anything, tenantID).Return(subscriptions, nil)

	service := subscription.NewService(mockRepo)

	// Act
	result, err := service.GetSubscriptionHistory(context.Background(), tenantID)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, result, 3)
	mockRepo.AssertExpectations(t)
}

func TestSubscriptionService_ValidateSubscription(t *testing.T) {
	// Arrange
	service := subscription.NewService(nil)

	// Test valid subscription
	validSub := &subscription.Subscription{
		Status:    "ACTIVE",
		StartDate:  time.Now().Add(-15 * 24 * time.Hour),
		EndDate: time.Now().Add(15 * 24 * time.Hour),
	}
	assert.NoError(t, service.ValidateSubscription(validSub))

	// Test expired subscription
	expiredSub := &subscription.Subscription{
		Status:    "ACTIVE",
		StartDate:  time.Now().Add(-60 * 24 * time.Hour),
		EndDate: time.Now().Add(-15 * 24 * time.Hour),
	}
	assert.Error(t, service.ValidateSubscription(expiredSub))

	// Test cancelled subscription
	cancelledSub := &subscription.Subscription{
		Status:    subscription.StatusCancelled,
		StartDate: time.Now().Add(-15 * 24 * time.Hour),
		EndDate:   time.Now().Add(15 * 24 * time.Hour),
	}
	assert.Error(t, service.ValidateSubscription(cancelledSub))
}