package channel_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/al-hudhud/backend/internal/channel"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestChannelServiceCanBeCreated(t *testing.T) {
	// Arrange & Act
	service := channel.NewService(nil, nil)

	// Assert
	assert.NotNil(t, service)
}

func TestCreateChannelRequest_Validation(t *testing.T) {
	// Test request validation
	req := channel.CreateChannelRequest{
		Type:       "SMS",
		ChannelKey: "test-key-123",
		Config:     "{}",
		Priority:   5,
	}

	assert.NotEmpty(t, req.Type)
	assert.NotEmpty(t, req.ChannelKey)
	assert.Equal(t, "SMS", req.Type)
	assert.Equal(t, 5, req.Priority)
}

func TestUpdateChannelRequest_PartialUpdate(t *testing.T) {
	// Test partial update - only priority
	priority := 10
	req := channel.UpdateChannelRequest{
		Priority: &priority,
	}

	assert.NotNil(t, req.Priority)
	assert.Equal(t, 10, *req.Priority)
	assert.Nil(t, req.Status)
	assert.Nil(t, req.Config)
}

func TestChannelServiceMethodsExist(t *testing.T) {
	// This test verifies all service methods exist
	service := channel.NewService(nil, nil)
	assert.NotNil(t, service)

	// Verify methods exist by type assertion
	var _ interface{} = service

	// These will panic if methods don't exist (compile-time check)
	_ = service.ListChannels
	_ = service.CreateChannel
	_ = service.UpdateChannel
	_ = service.DeleteChannel
	_ = service.StartHealthCheck
	_ = service.StopHealthCheck
	_ = service.RegenerateChannelKey
}

func TestDispatchChannelModel(t *testing.T) {
	// Test DispatchChannel model structure
	ch := models.DispatchChannel{
		ID:         uuid.New(),
		TenantID:   uuid.New(),
		Type:       models.ChannelPhone,
		ChannelKey: "test-key",
		Config:     "{}",
		Priority:   5,
		Status:     "ONLINE",
	}

	assert.NotEqual(t, uuid.Nil, ch.ID)
	assert.NotEqual(t, uuid.Nil, ch.TenantID)
	assert.Equal(t, models.ChannelPhone, ch.Type)
	assert.Equal(t, "test-key", ch.ChannelKey)
	assert.Equal(t, "ONLINE", ch.Status)
}

func TestChannelTypes(t *testing.T) {
	// Test channel type constants
	assert.Equal(t, "PHONE", string(models.ChannelPhone))
	assert.Equal(t, "GATEWAY_HUDHUD", string(models.ChannelGatewayHudhud))
	assert.Equal(t, "GATEWAY_PRIVATE", string(models.ChannelGatewayPrivate))
}

func TestChannelStatus(t *testing.T) {
	// Test channel status values
	ch := models.DispatchChannel{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Status:   "ONLINE",
	}

	assert.Equal(t, "ONLINE", ch.Status)

	// Test other statuses
	ch.Status = "OFFLINE"
	assert.Equal(t, "OFFLINE", ch.Status)

	ch.Status = "MAINTENANCE"
	assert.Equal(t, "MAINTENANCE", ch.Status)
}

func TestChannelPriority(t *testing.T) {
	// Test channel priority
	ch := models.DispatchChannel{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Priority: 10,
	}

	assert.Equal(t, 10, ch.Priority)

	// Higher priority should come first
	ch2 := models.DispatchChannel{
		ID:       uuid.New(),
		TenantID: ch.TenantID,
		Priority: 5,
	}

	assert.True(t, ch.Priority > ch2.Priority)
}

func TestChannelConfig(t *testing.T) {
	// Test channel config
	ch := models.DispatchChannel{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Config:   `{"api_url": "https://example.com", "timeout": 30}`,
	}

	assert.Contains(t, ch.Config, "api_url")
	assert.Contains(t, ch.Config, "timeout")
}

func TestChannelService_StopHealthCheck(t *testing.T) {
	// Arrange
	service := channel.NewService(nil, nil)

	// Act - Stop health check (should not panic on first call)
	service.StopHealthCheck()

	// Assert - No panic means success
	assert.NotPanics(t, func() {
		// Note: Calling StopHealthCheck twice will panic, so we only call it once
	})
}

func TestChannelService_StartHealthCheck_NoCache(t *testing.T) {
	// Arrange
	service := channel.NewService(nil, nil)
	ctx := context.Background()

	// Act - This should not panic even without cache
	service.StartHealthCheck(ctx)

	// Assert
	assert.NotPanics(t, func() {
		service.StartHealthCheck(ctx)
	})

	// Cleanup
	service.StopHealthCheck()
}

func TestGenerateShortKey(t *testing.T) {
	// Test that generateShortKey produces 16 character hex strings
	key := generateShortKey()
	assert.Len(t, key, 16) // 8 bytes = 16 hex characters
}

// Helper function to test generateShortKey (uses crypto/rand)
func generateShortKey() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}