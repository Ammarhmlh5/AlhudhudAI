package messages_test

import (
	"context"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/messages"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockMessageRepository - Mock for message operations
type MockMessageRepository struct {
	mock.Mock
}

func (m *MockMessageRepository) CreateMessage(ctx context.Context, msg *messages.Message) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *MockMessageRepository) GetMessageByID(ctx context.Context, messageID uuid.UUID) (*messages.Message, error) {
	args := m.Called(ctx, messageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*messages.Message), args.Error(1)
}

func (m *MockMessageRepository) GetMessagesByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]messages.Message, error) {
	args := m.Called(ctx, tenantID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]messages.Message), args.Error(1)
}

func (m *MockMessageRepository) UpdateMessageStatus(ctx context.Context, messageID uuid.UUID, status string) error {
	args := m.Called(ctx, messageID, status)
	return args.Error(0)
}

func (m *MockMessageRepository) UpdateMessageDeliveryTime(ctx context.Context, messageID uuid.UUID, deliveredAt time.Time) error {
	args := m.Called(ctx, messageID, deliveredAt)
	return args.Error(0)
}

func (m *MockMessageRepository) IncrementAttempts(ctx context.Context, messageID uuid.UUID) error {
	args := m.Called(ctx, messageID)
	return args.Error(0)
}

func (m *MockMessageRepository) SetMessageError(ctx context.Context, messageID uuid.UUID, errorMsg string) error {
	args := m.Called(ctx, messageID, errorMsg)
	return args.Error(0)
}

func TestMessageManager_CreateMessage_Success(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	msg := &messages.Message{
		TenantID:    tenantID,
		PhoneNumber: "+966501234567",
		Content:     "Test message",
		Priority:    "normal",
	}

	mockRepo := new(MockMessageRepository)
	mockRepo.On("CreateMessage", mock.Anything, mock.AnythingOfType("*messages.Message")).Return(nil)

	manager := messages.NewManager(mockRepo)

	// Act
	result, err := manager.CreateMessage(context.Background(), msg)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result.MessageID)
	assert.Equal(t, tenantID, result.TenantID)
	assert.Equal(t, "+966501234567", result.PhoneNumber)
	assert.Equal(t, "QUEUED", result.Status)
	mockRepo.AssertExpectations(t)
}

func TestMessageManager_ValidatePhoneNumber_Valid(t *testing.T) {
	// Arrange
	manager := messages.NewManager(nil)

	// Act & Assert
	assert.NoError(t, manager.ValidatePhoneNumber("+966501234567"))
	assert.NoError(t, manager.ValidatePhoneNumber("966501234567"))
	assert.NoError(t, manager.ValidatePhoneNumber("0501234567"))
}

func TestMessageManager_ValidatePhoneNumber_Invalid(t *testing.T) {
	// Arrange
	manager := messages.NewManager(nil)

	// Act & Assert
	assert.Error(t, manager.ValidatePhoneNumber("invalid"))
	assert.Error(t, manager.ValidatePhoneNumber(""))
	assert.Error(t, manager.ValidatePhoneNumber("+123"))
}

func TestMessageManager_UpdateStatus_Success(t *testing.T) {
	// Arrange
	messageID := uuid.New()

	mockRepo := new(MockMessageRepository)
	mockRepo.On("UpdateMessageStatus", mock.Anything, messageID, "SENT").Return(nil)

	manager := messages.NewManager(mockRepo)

	// Act
	err := manager.UpdateStatus(context.Background(), messageID, "SENT")

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMessageManager_MarkDelivered(t *testing.T) {
	// Arrange
	messageID := uuid.New()

	mockRepo := new(MockMessageRepository)
	mockRepo.On("UpdateMessageDeliveryTime", mock.Anything, messageID, mock.AnythingOfType("time.Time")).Return(nil)

	manager := messages.NewManager(mockRepo)

	// Act
	err := manager.MarkDelivered(context.Background(), messageID)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMessageManager_RecordFailure(t *testing.T) {
	// Arrange
	messageID := uuid.New()
	errorMsg := "Connection timeout"

	mockRepo := new(MockMessageRepository)
	mockRepo.On("IncrementAttempts", mock.Anything, messageID).Return(nil)
	mockRepo.On("SetMessageError", mock.Anything, messageID, errorMsg).Return(nil)

	manager := messages.NewManager(mockRepo)

	// Act
	err := manager.RecordFailure(context.Background(), messageID, errorMsg)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMessageManager_GetMessagesWithPagination(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	tenantMessages := []messages.Message{
		{ID: uuid.New(), MessageID: uuid.New().String(), TenantID: tenantID, Content: "Message 1"},
		{ID: uuid.New(), MessageID: uuid.New().String(), TenantID: tenantID, Content: "Message 2"},
		{ID: uuid.New(), MessageID: uuid.New().String(), TenantID: tenantID, Content: "Message 3"},
	}

	mockRepo := new(MockMessageRepository)
	mockRepo.On("GetMessagesByTenant", mock.Anything, tenantID, 10, 0).Return(tenantMessages, nil)

	manager := messages.NewManager(mockRepo)

	// Act
	result, err := manager.GetMessagesByTenant(context.Background(), tenantID, 10, 0)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, result, 3)
	mockRepo.AssertExpectations(t)
}

func TestMessageManager_CalculateCost(t *testing.T) {
	// Arrange
	manager := messages.NewManager(nil)

	// Act & Assert
	cost, err := manager.CalculateCost("SA", 1)
	assert.NoError(t, err)
	assert.True(t, cost.GreaterThan(decimal.Zero))

	// Test with unknown country
	cost, err = manager.CalculateCost("XX", 1)
	assert.Error(t, err)
	assert.Equal(t, decimal.Zero, cost)
}

func TestMessageManager_Idempotency(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	idempotencyKey := "unique-key-123"
	
	msg1 := &messages.Message{
		TenantID:       tenantID,
		PhoneNumber:    "+966501234567",
		Content:        "Test message",
		IdempotencyKey: idempotencyKey,
	}

	msg2 := &messages.Message{
		TenantID:       tenantID,
		PhoneNumber:    "+966501234567",
		Content:        "Test message",
		IdempotencyKey: idempotencyKey,
	}

	mockRepo := new(MockMessageRepository)
	mockRepo.On("CreateMessage", mock.Anything, mock.AnythingOfType("*messages.Message")).Return(nil)

	manager := messages.NewManager(mockRepo)

	// Act - Create first message
	result1, err := manager.CreateMessage(context.Background(), msg1)
	assert.NoError(t, err)

	// Act - Try to create duplicate
	result2, err := manager.CreateMessage(context.Background(), msg2)
	
	// Assert - Should return same message ID
	assert.NoError(t, err)
	assert.Equal(t, result1.MessageID, result2.MessageID)
}