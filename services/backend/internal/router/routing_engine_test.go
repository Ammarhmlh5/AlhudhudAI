package router

import (
	"context"
	"os"
	"testing"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestRoutingEngine_GetBestChannel_Priority tests that the routing engine selects channels by priority
func TestRoutingEngine_GetBestChannel_Priority(t *testing.T) {
	if os.Getenv("DB_HOST") == "" {
		t.Skip("Skipping test requires database")
	}
	db.InitGormDB()
	tenantID := uuid.New()

	// Create test channels with different priorities
	err := db.RunInTenantScope(tenantID, false, func(tx *gorm.DB) error {
		channels := []models.DispatchChannel{
			{
				TenantID:   tenantID,
				Type:       "PHONE",
				ChannelKey: "phone_1",
				Status:     "ONLINE",
				Priority:   1,
				Config:     "{}",
			},
			{
				TenantID:   tenantID,
				Type:       "GATEWAY_HUDHUD",
				ChannelKey: "hudhud_1",
				Status:     "ONLINE",
				Priority:   10, // Higher priority
				Config:     "{}",
			},
		}
		return tx.Create(&channels).Error
	})
	require.NoError(t, err)

	// Test routing
	engine := NewRoutingEngine()
	msg := &models.Message{}

	channel, err := engine.GetBestChannel(context.Background(), tenantID, msg)
	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, 10, channel.Priority, "Should select highest priority channel")
	assert.Equal(t, models.ChannelGatewayHudhud, channel.Type)
}

// TestRoutingEngine_WaitMode tests that WAIT_MODE is triggered when all channels are offline
func TestRoutingEngine_WaitMode(t *testing.T) {
	if os.Getenv("DB_HOST") == "" {
		t.Skip("Skipping test requires database")
	}
	db.InitGormDB()
	tenantID := uuid.New()

	// Create offline channels
	err := db.RunInTenantScope(tenantID, false, func(tx *gorm.DB) error {
		channels := []models.DispatchChannel{
			{
				TenantID:   tenantID,
				Type:       "PHONE",
				ChannelKey: "phone_offline",
				Status:     "OFFLINE",
				Priority:   5,
				Config:     "{}",
			},
		}
		return tx.Create(&channels).Error
	})
	assert.NoError(t, err)

	// Test routing
	engine := NewRoutingEngine()
	msg := &models.Message{}

	channel, err := engine.GetBestChannel(context.Background(), tenantID, msg)
	assert.Error(t, err)
	assert.Nil(t, channel)
	assert.Equal(t, "WAIT_MODE_REQUIRED", err.Error())
}

// TestRoutingEngine_ChannelFallback tests fallback to lower priority channels
func TestRoutingEngine_ChannelFallback(t *testing.T) {
	if os.Getenv("DB_HOST") == "" {
		t.Skip("Skipping test requires database")
	}
	db.InitGormDB()
	tenantID := uuid.New()

	// Create channels where highest priority is offline
	err := db.RunInTenantScope(tenantID, false, func(tx *gorm.DB) error {
		channels := []models.DispatchChannel{
			{
				TenantID:   tenantID,
				Type:       "GATEWAY_HUDHUD",
				ChannelKey: "hudhud_offline",
				Status:     "OFFLINE",
				Priority:   10,
				Config:     "{}",
			},
			{
				TenantID:   tenantID,
				Type:       "PHONE",
				ChannelKey: "phone_online",
				Status:     "ONLINE",
				Priority:   5,
				Config:     "{}",
			},
		}
		return tx.Create(&channels).Error
	})
	require.NoError(t, err)

	// Test routing
	engine := NewRoutingEngine()
	msg := &models.Message{}

	channel, err := engine.GetBestChannel(context.Background(), tenantID, msg)
	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, models.ChannelPhone, channel.Type, "Should fallback to lower priority online channel")
}
