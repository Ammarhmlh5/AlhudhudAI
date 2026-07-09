package gateway

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Create tables
	err = db.AutoMigrate(&DedicatedGateway{}, &ResourceQuota{})
	require.NoError(t, err)

	// Create tenants table for foreign key
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS tenants (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			balance DECIMAL(10, 2) DEFAULT 0.00,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	require.NoError(t, err)

	return db
}

// createTestTenant creates a test tenant
func createTestTenant(t *testing.T, db *gorm.DB) uuid.UUID {
	tenantID := uuid.New()
	err := db.Exec("INSERT INTO tenants (id, name) VALUES (?, ?)", tenantID.String(), "Test Tenant").Error
	require.NoError(t, err)
	return tenantID
}

func TestNewManager(t *testing.T) {
	db := setupTestDB(t)

	t.Run("valid encryption key", func(t *testing.T) {
		key := "12345678901234567890123456789012" // 32 bytes
		manager, err := NewManager(db, key)
		assert.NoError(t, err)
		assert.NotNil(t, manager)
	})

	t.Run("invalid encryption key length", func(t *testing.T) {
		key := "short" // Less than 32 bytes
		manager, err := NewManager(db, key)
		assert.Error(t, err)
		assert.Nil(t, manager)
		assert.Contains(t, err.Error(), "32 bytes")
	})
}

func TestCreateGateway(t *testing.T) {
	db := setupTestDB(t)
	tenantID := createTestTenant(t, db)
	key := "12345678901234567890123456789012"
	manager, err := NewManager(db, key)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		req := &CreateGatewayRequest{
			TenantID:           tenantID,
			GatewayName:        "Test Gateway",
			GatewayCode:        "TEST-GW-001",
			Description:        "Test gateway for unit tests",
			TelecomProvider:    "Test Telecom",
			TelecomAPIEndpoint: "https://api.test-telecom.com",
			TelecomUsername:    "testuser",
			TelecomPassword:    "testpassword",
			TelecomAPIKey:      "test-api-key",
			ServerType:         "shared",
			MaxThroughput:      100,
			SubscriptionType:   "monthly",
			SubscriptionFee:    100.00,
			ServerRentalFee:    50.00,
			SetupFee:           25.00,
			BillingCycle:       "monthly",
			ContractStartDate:  time.Now(),
		}

		gateway, err := manager.CreateGateway(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, gateway)
		assert.Equal(t, req.GatewayName, gateway.GatewayName)
		assert.Equal(t, req.GatewayCode, gateway.GatewayCode)
		assert.Equal(t, "active", gateway.Status)
		assert.NotEmpty(t, gateway.TelecomPasswordEncrypted)
		assert.NotEmpty(t, gateway.TelecomAPIKeyEncrypted)

		// Verify resource quotas were created
		var quota ResourceQuota
		err = db.Where("gateway_id = ?", gateway.ID).First(&quota).Error
		assert.NoError(t, err)
		assert.NotNil(t, quota.CPUCores)
		assert.NotNil(t, quota.MemoryMB)
	})

	t.Run("duplicate gateway code", func(t *testing.T) {
		req := &CreateGatewayRequest{
			TenantID:           tenantID,
			GatewayName:        "Another Gateway",
			GatewayCode:        "TEST-GW-001", // Same code as above
			TelecomProvider:    "Test Telecom",
			TelecomAPIEndpoint: "https://api.test-telecom.com",
			ServerType:         "shared",
			MaxThroughput:      100,
			SubscriptionType:   "monthly",
			BillingCycle:       "monthly",
			ContractStartDate:  time.Now(),
		}

		gateway, err := manager.CreateGateway(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, gateway)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("invalid tenant", func(t *testing.T) {
		invalidTenantID := uuid.New()
		req := &CreateGatewayRequest{
			TenantID:           invalidTenantID,
			GatewayName:        "Test Gateway",
			GatewayCode:        "TEST-GW-002",
			TelecomProvider:    "Test Telecom",
			TelecomAPIEndpoint: "https://api.test-telecom.com",
			ServerType:         "shared",
			MaxThroughput:      100,
			SubscriptionType:   "monthly",
			BillingCycle:       "monthly",
			ContractStartDate:  time.Now(),
		}

		gateway, err := manager.CreateGateway(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, gateway)
		assert.Contains(t, err.Error(), "tenant")
	})

	t.Run("with custom resource quotas", func(t *testing.T) {
		cpuCores := 8.0
		memoryMB := 16384
		req := &CreateGatewayRequest{
			TenantID:           tenantID,
			GatewayName:        "Custom Gateway",
			GatewayCode:        "TEST-GW-003",
			TelecomProvider:    "Test Telecom",
			TelecomAPIEndpoint: "https://api.test-telecom.com",
			ServerType:         "dedicated",
			MaxThroughput:      500,
			SubscriptionType:   "yearly",
			BillingCycle:       "yearly",
			ContractStartDate:  time.Now(),
			ResourceQuotas: &ResourceQuotasRequest{
				CPUCores:  &cpuCores,
				MemoryMB:  &memoryMB,
				EnforcementMode: "hard",
			},
		}

		gateway, err := manager.CreateGateway(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, gateway)

		// Verify custom quotas
		var quota ResourceQuota
		err = db.Where("gateway_id = ?", gateway.ID).First(&quota).Error
		assert.NoError(t, err)
		assert.Equal(t, cpuCores, *quota.CPUCores)
		assert.Equal(t, memoryMB, *quota.MemoryMB)
		assert.Equal(t, "hard", quota.EnforcementMode)
	})
}

func TestUpdateGateway(t *testing.T) {
	db := setupTestDB(t)
	tenantID := createTestTenant(t, db)
	key := "12345678901234567890123456789012"
	manager, err := NewManager(db, key)
	require.NoError(t, err)

	ctx := context.Background()

	// Create a gateway first
	createReq := &CreateGatewayRequest{
		TenantID:           tenantID,
		GatewayName:        "Original Gateway",
		GatewayCode:        "TEST-GW-UPDATE",
		TelecomProvider:    "Test Telecom",
		TelecomAPIEndpoint: "https://api.test-telecom.com",
		ServerType:         "shared",
		MaxThroughput:      100,
		SubscriptionType:   "monthly",
		SubscriptionFee:    100.00,
		BillingCycle:       "monthly",
		ContractStartDate:  time.Now(),
	}

	gateway, err := manager.CreateGateway(ctx, createReq)
	require.NoError(t, err)

	t.Run("successful update", func(t *testing.T) {
		newName := "Updated Gateway"
		newFee := 150.00
		updateReq := &UpdateGatewayRequest{
			GatewayID:       gateway.ID,
			TenantID:        tenantID,
			GatewayName:     &newName,
			SubscriptionFee: &newFee,
		}

		updated, err := manager.UpdateGateway(ctx, updateReq)
		assert.NoError(t, err)
		assert.NotNil(t, updated)
		assert.Equal(t, newName, updated.GatewayName)
		assert.Equal(t, newFee, updated.SubscriptionFee)
	})

	t.Run("update with wrong tenant", func(t *testing.T) {
		wrongTenantID := createTestTenant(t, db)
		newName := "Hacked Gateway"
		updateReq := &UpdateGatewayRequest{
			GatewayID:   gateway.ID,
			TenantID:    wrongTenantID,
			GatewayName: &newName,
		}

		updated, err := manager.UpdateGateway(ctx, updateReq)
		assert.Error(t, err)
		assert.Nil(t, updated)
		assert.Contains(t, err.Error(), "not found or access denied")
	})

	t.Run("update credentials", func(t *testing.T) {
		newPassword := "newpassword123"
		newAPIKey := "new-api-key-456"
		updateReq := &UpdateGatewayRequest{
			GatewayID:       gateway.ID,
			TenantID:        tenantID,
			TelecomPassword: &newPassword,
			TelecomAPIKey:   &newAPIKey,
		}

		updated, err := manager.UpdateGateway(ctx, updateReq)
		assert.NoError(t, err)
		assert.NotNil(t, updated)
		assert.NotEmpty(t, updated.TelecomPasswordEncrypted)
		assert.NotEmpty(t, updated.TelecomAPIKeyEncrypted)
		// Verify encryption changed
		assert.NotEqual(t, gateway.TelecomPasswordEncrypted, updated.TelecomPasswordEncrypted)
	})
}

func TestSuspendGateway(t *testing.T) {
	db := setupTestDB(t)
	tenantID := createTestTenant(t, db)
	key := "12345678901234567890123456789012"
	manager, err := NewManager(db, key)
	require.NoError(t, err)

	ctx := context.Background()

	// Create a gateway
	createReq := &CreateGatewayRequest{
		TenantID:           tenantID,
		GatewayName:        "Gateway to Suspend",
		GatewayCode:        "TEST-GW-SUSPEND",
		TelecomProvider:    "Test Telecom",
		TelecomAPIEndpoint: "https://api.test-telecom.com",
		ServerType:         "shared",
		MaxThroughput:      100,
		SubscriptionType:   "monthly",
		BillingCycle:       "monthly",
		ContractStartDate:  time.Now(),
	}

	gateway, err := manager.CreateGateway(ctx, createReq)
	require.NoError(t, err)

	t.Run("successful suspension", func(t *testing.T) {
		err := manager.SuspendGateway(ctx, gateway.ID, tenantID, "Payment overdue")
		assert.NoError(t, err)

		// Verify status changed
		var updated DedicatedGateway
		err = db.Where("id = ?", gateway.ID).First(&updated).Error
		assert.NoError(t, err)
		assert.Equal(t, "suspended", updated.Status)
	})

	t.Run("suspend with wrong tenant", func(t *testing.T) {
		wrongTenantID := createTestTenant(t, db)
		err := manager.SuspendGateway(ctx, gateway.ID, wrongTenantID, "Test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found or access denied")
	})
}

func TestTerminateGateway(t *testing.T) {
	db := setupTestDB(t)
	tenantID := createTestTenant(t, db)
	key := "12345678901234567890123456789012"
	manager, err := NewManager(db, key)
	require.NoError(t, err)

	ctx := context.Background()

	// Create a gateway
	createReq := &CreateGatewayRequest{
		TenantID:           tenantID,
		GatewayName:        "Gateway to Terminate",
		GatewayCode:        "TEST-GW-TERMINATE",
		TelecomProvider:    "Test Telecom",
		TelecomAPIEndpoint: "https://api.test-telecom.com",
		ServerType:         "shared",
		MaxThroughput:      100,
		SubscriptionType:   "monthly",
		BillingCycle:       "monthly",
		ContractStartDate:  time.Now(),
	}

	gateway, err := manager.CreateGateway(ctx, createReq)
	require.NoError(t, err)

	t.Run("successful termination", func(t *testing.T) {
		err := manager.TerminateGateway(ctx, gateway.ID, tenantID, "Contract ended")
		assert.NoError(t, err)

		// Verify status changed
		var updated DedicatedGateway
		err = db.Where("id = ?", gateway.ID).First(&updated).Error
		assert.NoError(t, err)
		assert.Equal(t, "terminated", updated.Status)
	})
}

func TestPerformHealthCheck(t *testing.T) {
	db := setupTestDB(t)
	tenantID := createTestTenant(t, db)
	key := "12345678901234567890123456789012"
	manager, err := NewManager(db, key)
	require.NoError(t, err)

	ctx := context.Background()

	// Create a gateway
	createReq := &CreateGatewayRequest{
		TenantID:           tenantID,
		GatewayName:        "Gateway for Health Check",
		GatewayCode:        "TEST-GW-HEALTH",
		TelecomProvider:    "Test Telecom",
		TelecomAPIEndpoint: "https://api.test-telecom.com",
		ServerType:         "shared",
		MaxThroughput:      100,
		SubscriptionType:   "monthly",
		BillingCycle:       "monthly",
		ContractStartDate:  time.Now(),
	}

	gateway, err := manager.CreateGateway(ctx, createReq)
	require.NoError(t, err)

	t.Run("successful health check", func(t *testing.T) {
		result, err := manager.PerformHealthCheck(ctx, gateway.ID, tenantID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.IsHealthy)
		assert.Equal(t, "healthy", result.Status)
		assert.True(t, result.TelecomConnection)

		// Verify last_health_check was updated
		var updated DedicatedGateway
		err = db.Where("id = ?", gateway.ID).First(&updated).Error
		assert.NoError(t, err)
		assert.NotNil(t, updated.LastHealthCheck)
		assert.Equal(t, "healthy", updated.HealthStatus)
	})

	t.Run("health check with wrong tenant", func(t *testing.T) {
		wrongTenantID := createTestTenant(t, db)
		result, err := manager.PerformHealthCheck(ctx, gateway.ID, wrongTenantID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not found or access denied")
	})
}

func TestEncryptDecrypt(t *testing.T) {
	db := setupTestDB(t)
	key := "12345678901234567890123456789012"
	manager, err := NewManager(db, key)
	require.NoError(t, err)

	t.Run("encrypt and decrypt", func(t *testing.T) {
		plaintext := "sensitive-password-123"
		
		encrypted, err := manager.encrypt(plaintext)
		assert.NoError(t, err)
		assert.NotEmpty(t, encrypted)
		assert.NotEqual(t, plaintext, encrypted)

		decrypted, err := manager.decrypt(encrypted)
		assert.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("encrypt empty string", func(t *testing.T) {
		encrypted, err := manager.encrypt("")
		assert.NoError(t, err)
		assert.Empty(t, encrypted)
	})

	t.Run("decrypt empty string", func(t *testing.T) {
		decrypted, err := manager.decrypt("")
		assert.NoError(t, err)
		assert.Empty(t, decrypted)
	})
}

func TestCalculateNextBillingDate(t *testing.T) {
	db := setupTestDB(t)
	key := "12345678901234567890123456789012"
	manager, err := NewManager(db, key)
	require.NoError(t, err)

	startDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	t.Run("monthly billing", func(t *testing.T) {
		nextDate := manager.calculateNextBillingDate(startDate, "monthly")
		expected := time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC)
		assert.Equal(t, expected, nextDate)
	})

	t.Run("yearly billing", func(t *testing.T) {
		nextDate := manager.calculateNextBillingDate(startDate, "yearly")
		expected := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		assert.Equal(t, expected, nextDate)
	})

	t.Run("default to monthly", func(t *testing.T) {
		nextDate := manager.calculateNextBillingDate(startDate, "invalid")
		expected := time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC)
		assert.Equal(t, expected, nextDate)
	})
}
