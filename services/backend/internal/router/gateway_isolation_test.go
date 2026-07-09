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

// TestGatewayIsolation_DifferentTenants tests that tenants cannot access each other's gateways
func TestGatewayIsolation_DifferentTenants(t *testing.T) {
	if os.Getenv("DB_HOST") == "" {
		t.Skip("Skipping test requires database")
	}
	db.InitGormDB()

	tenantA := uuid.New()
	tenantB := uuid.New()

	// Create gateway for Tenant A
	err := db.RunInTenantScope(tenantA, false, func(tx *gorm.DB) error {
		gateway := models.DispatchChannel{
			TenantID:   tenantA,
			Type:       "GATEWAY_PRIVATE",
			ChannelKey: "private_gateway_A",
			Config:     `{"api_key": "secret_key_A"}`,
			Status:     "ONLINE",
			Priority:   5,
		}
		return tx.Create(&gateway).Error
	})
	assert.NoError(t, err)

	// Create gateway for Tenant B
	err = db.RunInTenantScope(tenantB, false, func(tx *gorm.DB) error {
		gateway := models.DispatchChannel{
			TenantID:   tenantB,
			Type:       "GATEWAY_PRIVATE",
			ChannelKey: "private_gateway_B",
			Config:     `{"api_key": "secret_key_B"}`,
			Status:     "ONLINE",
			Priority:   5,
		}
		return tx.Create(&gateway).Error
	})
	assert.NoError(t, err)

	// Test: Tenant A should only see their own gateway
	err = db.RunInTenantScope(tenantA, false, func(tx *gorm.DB) error {
		var gateways []models.DispatchChannel
		if err := tx.Where("tenant_id = ?", tenantA).Find(&gateways).Error; err != nil {
			return err
		}

		assert.Equal(t, 1, len(gateways), "Tenant A should only see 1 gateway")
		assert.Equal(t, "private_gateway_A", gateways[0].ChannelKey)
		assert.Contains(t, gateways[0].Config, "secret_key_A")
		assert.NotContains(t, gateways[0].Config, "secret_key_B")

		return nil
	})
	assert.NoError(t, err)

	// Test: Tenant B should only see their own gateway
	err = db.RunInTenantScope(tenantB, false, func(tx *gorm.DB) error {
		var gateways []models.DispatchChannel
		if err := tx.Where("tenant_id = ?", tenantB).Find(&gateways).Error; err != nil {
			return err
		}

		assert.Equal(t, 1, len(gateways), "Tenant B should only see 1 gateway")
		assert.Equal(t, "private_gateway_B", gateways[0].ChannelKey)
		assert.Contains(t, gateways[0].Config, "secret_key_B")
		assert.NotContains(t, gateways[0].Config, "secret_key_A")

		return nil
	})
	assert.NoError(t, err)
}

// TestGatewayIsolation_RoutingEngine tests that routing engine respects tenant isolation
func TestGatewayIsolation_RoutingEngine(t *testing.T) {
	if os.Getenv("DB_HOST") == "" {
		t.Skip("Skipping test requires database")
	}
	db.InitGormDB()

	tenantA := uuid.New()
	tenantB := uuid.New()

	// Create high-priority gateway for Tenant A
	err := db.RunInTenantScope(tenantA, false, func(tx *gorm.DB) error {
		gateway := models.DispatchChannel{
			TenantID:   tenantA,
			Type:       "GATEWAY_PRIVATE",
			ChannelKey: "high_priority_A",
			Status:     "ONLINE",
			Priority:   100, // Very high priority
			Config:     "{}",
		}
		return tx.Create(&gateway).Error
	})
	assert.NoError(t, err)

	// Create low-priority gateway for Tenant B
	err = db.RunInTenantScope(tenantB, false, func(tx *gorm.DB) error {
		gateway := models.DispatchChannel{
			TenantID:   tenantB,
			Type:       "PHONE",
			ChannelKey: "low_priority_B",
			Status:     "ONLINE",
			Priority:   1, // Low priority
			Config:     "{}",
		}
		return tx.Create(&gateway).Error
	})
	assert.NoError(t, err)

	// Test: Tenant B should get their low-priority gateway, NOT Tenant A's high-priority one
	engine := NewRoutingEngine()
	msg := &models.Message{}

	channel, err := engine.GetBestChannel(context.Background(), tenantB, msg)
	require.NoError(t, err)
	require.NotNil(t, channel, "Should create gateway for tenant B")
	assert.Equal(t, tenantB, channel.TenantID, "Should select Tenant B's gateway")
	assert.Equal(t, "low_priority_B", channel.ChannelKey)
	assert.NotEqual(t, "high_priority_A", channel.ChannelKey, "Should NOT select Tenant A's gateway")
}

// TestGatewayIsolation_UniqueChannelKey tests that channel_key can be duplicated across tenants
func TestGatewayIsolation_UniqueChannelKey(t *testing.T) {
	if os.Getenv("DB_HOST") == "" {
		t.Skip("Skipping test requires database")
	}
	db.InitGormDB()

	tenantA := uuid.New()
	tenantB := uuid.New()

	// Create gateway with key "my_gateway" for Tenant A
	err := db.RunInTenantScope(tenantA, false, func(tx *gorm.DB) error {
		gateway := models.DispatchChannel{
			TenantID:   tenantA,
			Type:       "GATEWAY_PRIVATE",
			ChannelKey: "my_gateway",
			Status:     "ONLINE",
			Priority:   5,
			Config:     "{}",
		}
		return tx.Create(&gateway).Error
	})
	assert.NoError(t, err)

	// Create gateway with SAME key "my_gateway" for Tenant B - should succeed
	err = db.RunInTenantScope(tenantB, false, func(tx *gorm.DB) error {
		gateway := models.DispatchChannel{
			TenantID:   tenantB,
			Type:       "GATEWAY_PRIVATE",
			ChannelKey: "my_gateway", // Same key, different tenant
			Status:     "ONLINE",
			Priority:   5,
			Config:     "{}",
		}
		return tx.Create(&gateway).Error
	})
	assert.NoError(t, err, "Should allow same channel_key for different tenants")

	// Try to create duplicate key for same tenant - should fail
	err = db.RunInTenantScope(tenantA, false, func(tx *gorm.DB) error {
		gateway := models.DispatchChannel{
			TenantID:   tenantA,
			Type:       "PHONE",
			ChannelKey: "my_gateway", // Duplicate for same tenant
			Status:     "ONLINE",
			Priority:   3,
			Config:     "{}",
		}
		return tx.Create(&gateway).Error
	})
	assert.Error(t, err, "Should NOT allow duplicate channel_key for same tenant")
}
