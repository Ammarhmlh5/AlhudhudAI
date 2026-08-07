// Package gateway provides API key management functionality tests
package gateway

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupAPIKeyTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// :memory: sqlite is per-connection; pin the pool to a single connection
	// so tables created by AutoMigrate are visible to later queries.
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	// Create tables
	err = db.AutoMigrate(&GatewayAPIKey{}, &DedicatedGateway{})
	require.NoError(t, err)

	// Create tenants table
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS tenants (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	require.NoError(t, err)

	return db
}

// createTestTenantForAPIKey creates a test tenant (for api key tests to avoid redeclaration)
func createTestTenantForAPIKey(t *testing.T, db *gorm.DB) uuid.UUID {
	tenantID := uuid.New()
	err := db.Exec("INSERT INTO tenants (id, name) VALUES (?, ?)", tenantID.String(), "Test Tenant").Error
	require.NoError(t, err)
	return tenantID
}

// createTestGateway creates a test gateway
func createTestGateway(t *testing.T, db *gorm.DB, tenantID uuid.UUID) uuid.UUID {
	gateway := &DedicatedGateway{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		GatewayName:        "Test Gateway",
		GatewayCode:        "TEST-GW-001",
		TelecomProvider:    "Test Provider",
		TelecomAPIEndpoint: "https://api.test.com",
		Status:             "active",
		ContractStartDate:  time.Now(),
	}
	err := db.Create(gateway).Error
	require.NoError(t, err)
	return gateway.ID
}

// skipIfNonPostgresAllowedIPs skips tests that depend on AllowedIPs round-tripping
// through a Postgres text[] array. SQLite has no array type, so these tests can
// only run against a real Postgres database.
func skipIfNonPostgresAllowedIPs(t *testing.T, db *gorm.DB) {
	t.Helper()
	if db.Dialector.Name() != "postgres" {
		t.Skip("AllowedIPs relies on Postgres text[] arrays; requires Postgres driver")
	}
}

// TestCreateAPIKey tests the CreateAPIKey function
func TestCreateAPIKey(t *testing.T) {
	db := setupAPIKeyTestDB(t)
	manager := NewAPIKeyManager(db)
	ctx := context.Background()

	tenantID := createTestTenantForAPIKey(t, db)
	gatewayID := createTestGateway(t, db, tenantID)

	t.Run("Success - Create API key with default settings", func(t *testing.T) {
		req := &CreateAPIKeyRequest{
			TenantID:    tenantID,
			GatewayID:   gatewayID,
			KeyName:     "Test API Key",
			Description: "Test Description",
		}

		resp, err := manager.CreateAPIKey(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.NotEmpty(t, resp.APIKey)
		assert.True(t, len(resp.APIKey) > 10)
		assert.Equal(t, "Test API Key", resp.KeyInfo.KeyName)
		assert.Equal(t, "Test Description", resp.KeyInfo.Description)
		assert.Equal(t, 60, resp.KeyInfo.RateLimit) // Default rate limit
		assert.True(t, resp.KeyInfo.IsActive)
		assert.NotEmpty(t, resp.KeyInfo.Permissions)
	})

	t.Run("Success - Create API key with custom settings", func(t *testing.T) {
		skipIfNonPostgresAllowedIPs(t, db)

		expiresAt := time.Now().Add(30 * 24 * time.Hour)
		req := &CreateAPIKeyRequest{
			TenantID:    tenantID,
			GatewayID:   gatewayID,
			KeyName:     "Custom API Key",
			Description: "Custom Description",
			RateLimit:   120,
			AllowedIPs:  []string{"192.168.1.1", "10.0.0.0/24"},
			ExpiresAt:   &expiresAt,
			Permissions: map[string]interface{}{
				"send_sms":     true,
				"view_stats":   false,
				"view_reports": true,
			},
		}

		resp, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, 120, resp.KeyInfo.RateLimit)
		assert.Equal(t, 2, len(resp.KeyInfo.AllowedIPs))
		assert.NotNil(t, resp.KeyInfo.ExpiresAt)
	})

	t.Run("Error - Duplicate key name", func(t *testing.T) {
		req := &CreateAPIKeyRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			KeyName:   "Test API Key", // Already exists
		}

		_, err := manager.CreateAPIKey(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("Error - Invalid tenant", func(t *testing.T) {
		req := &CreateAPIKeyRequest{
			TenantID:  uuid.New(), // Non-existent tenant
			GatewayID: gatewayID,
			KeyName:   "Invalid Tenant Key",
		}

		_, err := manager.CreateAPIKey(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tenant")
	})

	t.Run("Error - Invalid gateway", func(t *testing.T) {
		req := &CreateAPIKeyRequest{
			TenantID:  tenantID,
			GatewayID: uuid.New(), // Non-existent gateway
			KeyName:   "Invalid Gateway Key",
		}

		_, err := manager.CreateAPIKey(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "gateway")
	})

	t.Run("Error - Invalid IP address", func(t *testing.T) {
		req := &CreateAPIKeyRequest{
			TenantID:   tenantID,
			GatewayID:  gatewayID,
			KeyName:    "Invalid IP Key",
			AllowedIPs: []string{"invalid-ip"},
		}

		_, err := manager.CreateAPIKey(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid IP")
	})
}

// TestSetPermissions tests the SetPermissions function
func TestSetPermissions(t *testing.T) {
	db := setupAPIKeyTestDB(t)
	manager := NewAPIKeyManager(db)
	ctx := context.Background()

	tenantID := createTestTenantForAPIKey(t, db)
	gatewayID := createTestGateway(t, db, tenantID)

	// Create an API key first
	createReq := &CreateAPIKeyRequest{
		TenantID:  tenantID,
		GatewayID: gatewayID,
		KeyName:   "Permissions Test Key",
	}
	createResp, err := manager.CreateAPIKey(ctx, createReq)
	require.NoError(t, err)

	t.Run("Success - Update permissions", func(t *testing.T) {
		req := &SetPermissionsRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			KeyID:     createResp.KeyInfo.ID,
			Permissions: map[string]interface{}{
				"send_sms":        true,
				"view_stats":      true,
				"view_reports":    false,
				"manage_gateway":  true,
				"manage_users":    false,
				"manage_api_keys": false,
			},
		}

		apiKey, err := manager.SetPermissions(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, apiKey)
		assert.Equal(t, true, apiKey.Permissions["send_sms"])
		assert.Equal(t, false, apiKey.Permissions["view_reports"])
		assert.Equal(t, true, apiKey.Permissions["manage_gateway"])
	})

	t.Run("Error - Invalid permission", func(t *testing.T) {
		req := &SetPermissionsRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			KeyID:     createResp.KeyInfo.ID,
			Permissions: map[string]interface{}{
				"invalid_permission": true,
			},
		}

		_, err := manager.SetPermissions(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid permission")
	})

	t.Run("Error - API key not found", func(t *testing.T) {
		req := &SetPermissionsRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			KeyID:     uuid.New(),
			Permissions: map[string]interface{}{
				"send_sms": true,
			},
		}

		_, err := manager.SetPermissions(ctx, req)
		assert.Error(t, err)
	})
}

// TestSetRateLimit tests the SetRateLimit function
func TestSetRateLimit(t *testing.T) {
	db := setupAPIKeyTestDB(t)
	manager := NewAPIKeyManager(db)
	ctx := context.Background()

	tenantID := createTestTenantForAPIKey(t, db)
	gatewayID := createTestGateway(t, db, tenantID)

	// Create an API key first
	createReq := &CreateAPIKeyRequest{
		TenantID:  tenantID,
		GatewayID: gatewayID,
		KeyName:   "Rate Limit Test Key",
	}
	createResp, err := manager.CreateAPIKey(ctx, createReq)
	require.NoError(t, err)

	t.Run("Success - Update rate limit", func(t *testing.T) {
		req := &SetRateLimitRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			KeyID:     createResp.KeyInfo.ID,
			RateLimit: 200,
		}

		apiKey, err := manager.SetRateLimit(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 200, apiKey.RateLimit)
	})

	t.Run("Error - Invalid rate limit (zero)", func(t *testing.T) {
		req := &SetRateLimitRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			KeyID:     createResp.KeyInfo.ID,
			RateLimit: 0,
		}

		_, err := manager.SetRateLimit(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "greater than 0")
	})

	t.Run("Error - Invalid rate limit (negative)", func(t *testing.T) {
		req := &SetRateLimitRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			KeyID:     createResp.KeyInfo.ID,
			RateLimit: -10,
		}

		_, err := manager.SetRateLimit(ctx, req)
		assert.Error(t, err)
	})
}

// TestIPWhitelisting tests AddAllowedIP and RemoveAllowedIP functions
func TestIPWhitelisting(t *testing.T) {
	db := setupAPIKeyTestDB(t)
	skipIfNonPostgresAllowedIPs(t, db)
	manager := NewAPIKeyManager(db)
	ctx := context.Background()

	tenantID := createTestTenantForAPIKey(t, db)
	gatewayID := createTestGateway(t, db, tenantID)

	// Create an API key first
	createReq := &CreateAPIKeyRequest{
		TenantID:  tenantID,
		GatewayID: gatewayID,
		KeyName:   "IP Whitelist Test Key",
	}
	createResp, err := manager.CreateAPIKey(ctx, createReq)
	require.NoError(t, err)

	t.Run("Success - Add IPv4 address", func(t *testing.T) {
		req := &AddAllowedIPRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			KeyID:     createResp.KeyInfo.ID,
			IPAddress: "192.168.1.100",
		}

		apiKey, err := manager.AddAllowedIP(ctx, req)
		require.NoError(t, err)
		assert.Contains(t, apiKey.AllowedIPs, "192.168.1.100")
	})

	t.Run("Success - Add CIDR notation", func(t *testing.T) {
		req := &AddAllowedIPRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			KeyID:     createResp.KeyInfo.ID,
			IPAddress: "10.0.0.0/24",
		}

		apiKey, err := manager.AddAllowedIP(ctx, req)
		require.NoError(t, err)
		assert.Contains(t, apiKey.AllowedIPs, "10.0.0.0/24")
	})

	t.Run("Success - Add IPv6 address", func(t *testing.T) {
		req := &AddAllowedIPRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			KeyID:     createResp.KeyInfo.ID,
			IPAddress: "2001:db8::1",
		}

		apiKey, err := manager.AddAllowedIP(ctx, req)
		require.NoError(t, err)
		assert.Contains(t, apiKey.AllowedIPs, "2001:db8::1")
	})

	t.Run("Error - Duplicate IP", func(t *testing.T) {
		req := &AddAllowedIPRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			KeyID:     createResp.KeyInfo.ID,
			IPAddress: "192.168.1.100", // Already added
		}

		_, err := manager.AddAllowedIP(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already in whitelist")
	})

	t.Run("Error - Invalid IP address", func(t *testing.T) {
		req := &AddAllowedIPRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			KeyID:     createResp.KeyInfo.ID,
			IPAddress: "invalid-ip",
		}

		_, err := manager.AddAllowedIP(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid IP")
	})

	t.Run("Success - Remove IP address", func(t *testing.T) {
		req := &RemoveAllowedIPRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			KeyID:     createResp.KeyInfo.ID,
			IPAddress: "192.168.1.100",
		}

		apiKey, err := manager.RemoveAllowedIP(ctx, req)
		require.NoError(t, err)
		assert.NotContains(t, apiKey.AllowedIPs, "192.168.1.100")
	})

	t.Run("Error - IP not in whitelist", func(t *testing.T) {
		req := &RemoveAllowedIPRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			KeyID:     createResp.KeyInfo.ID,
			IPAddress: "192.168.1.200", // Not in whitelist
		}

		_, err := manager.RemoveAllowedIP(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found in whitelist")
	})
}

// TestSetExpiry tests the SetExpiry function
func TestSetExpiry(t *testing.T) {
	db := setupAPIKeyTestDB(t)
	manager := NewAPIKeyManager(db)
	ctx := context.Background()

	tenantID := createTestTenantForAPIKey(t, db)
	gatewayID := createTestGateway(t, db, tenantID)

	// Create an API key first
	createReq := &CreateAPIKeyRequest{
		TenantID:  tenantID,
		GatewayID: gatewayID,
		KeyName:   "Expiry Test Key",
	}
	createResp, err := manager.CreateAPIKey(ctx, createReq)
	require.NoError(t, err)

	t.Run("Success - Set expiry date", func(t *testing.T) {
		expiresAt := time.Now().Add(30 * 24 * time.Hour)
		req := &SetExpiryRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			KeyID:     createResp.KeyInfo.ID,
			ExpiresAt: &expiresAt,
		}

		apiKey, err := manager.SetExpiry(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, apiKey.ExpiresAt)
		assert.True(t, apiKey.ExpiresAt.After(time.Now()))
	})

	t.Run("Success - Remove expiry date", func(t *testing.T) {
		req := &SetExpiryRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			KeyID:     createResp.KeyInfo.ID,
			ExpiresAt: nil, // Remove expiry
		}

		apiKey, err := manager.SetExpiry(ctx, req)
		require.NoError(t, err)
		assert.Nil(t, apiKey.ExpiresAt)
	})

	t.Run("Error - Expiry date in the past", func(t *testing.T) {
		pastDate := time.Now().Add(-24 * time.Hour)
		req := &SetExpiryRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			KeyID:     createResp.KeyInfo.ID,
			ExpiresAt: &pastDate,
		}

		_, err := manager.SetExpiry(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "future")
	})
}


// TestValidateAPIKey tests the ValidateAPIKey function
func TestValidateAPIKey(t *testing.T) {
	db := setupAPIKeyTestDB(t)
	skipIfNonPostgresAllowedIPs(t, db)
	manager := NewAPIKeyManager(db)
	ctx := context.Background()

	tenantID := createTestTenantForAPIKey(t, db)
	gatewayID := createTestGateway(t, db, tenantID)

	// Create an API key
	createReq := &CreateAPIKeyRequest{
		TenantID:   tenantID,
		GatewayID:  gatewayID,
		KeyName:    "Validation Test Key",
		AllowedIPs: []string{"192.168.1.1", "10.0.0.0/24"},
	}
	createResp, err := manager.CreateAPIKey(ctx, createReq)
	require.NoError(t, err)
	apiKey := createResp.APIKey

	t.Run("Success - Valid API key with allowed IP", func(t *testing.T) {
		req := &ValidateAPIKeyRequest{
			APIKey:    apiKey,
			ClientIP:  "192.168.1.1",
			GatewayID: gatewayID,
		}

		resp, err := manager.ValidateAPIKey(ctx, req)
		require.NoError(t, err)
		assert.True(t, resp.IsValid)
		assert.Equal(t, gatewayID, resp.GatewayID)
		assert.Equal(t, tenantID, resp.TenantID)
		assert.NotEmpty(t, resp.Permissions)
		assert.Equal(t, 60, resp.RateLimit)
	})

	t.Run("Success - Valid API key with IP in CIDR range", func(t *testing.T) {
		req := &ValidateAPIKeyRequest{
			APIKey:    apiKey,
			ClientIP:  "10.0.0.50", // Within 10.0.0.0/24
			GatewayID: gatewayID,
		}

		resp, err := manager.ValidateAPIKey(ctx, req)
		require.NoError(t, err)
		assert.True(t, resp.IsValid)
	})

	t.Run("Error - Invalid API key", func(t *testing.T) {
		req := &ValidateAPIKeyRequest{
			APIKey:    "gw_invalid_key",
			ClientIP:  "192.168.1.1",
			GatewayID: gatewayID,
		}

		resp, err := manager.ValidateAPIKey(ctx, req)
		require.NoError(t, err)
		assert.False(t, resp.IsValid)
		assert.Equal(t, "Invalid API key", resp.Error)
	})

	t.Run("Error - IP not whitelisted", func(t *testing.T) {
		req := &ValidateAPIKeyRequest{
			APIKey:    apiKey,
			ClientIP:  "192.168.2.1", // Not in whitelist
			GatewayID: gatewayID,
		}

		resp, err := manager.ValidateAPIKey(ctx, req)
		require.NoError(t, err)
		assert.False(t, resp.IsValid)
		assert.Equal(t, "IP address not whitelisted", resp.Error)
	})

	t.Run("Error - Expired API key", func(t *testing.T) {
		// Create an expired key
		pastDate := time.Now().Add(-24 * time.Hour)
		expiredReq := &CreateAPIKeyRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			KeyName:   "Expired Key",
			ExpiresAt: &pastDate,
		}
		expiredResp, err := manager.CreateAPIKey(ctx, expiredReq)
		require.NoError(t, err)

		req := &ValidateAPIKeyRequest{
			APIKey:    expiredResp.APIKey,
			ClientIP:  "192.168.1.1",
			GatewayID: gatewayID,
		}

		resp, err := manager.ValidateAPIKey(ctx, req)
		require.NoError(t, err)
		assert.False(t, resp.IsValid)
		assert.Equal(t, "API key has expired", resp.Error)
	})

	t.Run("Error - Inactive API key", func(t *testing.T) {
		// Deactivate the key
		err := manager.UpdateAPIKeyStatus(ctx, &UpdateAPIKeyStatusRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			KeyID:     createResp.KeyInfo.ID,
			IsActive:  false,
		})
		require.NoError(t, err)

		req := &ValidateAPIKeyRequest{
			APIKey:    apiKey,
			ClientIP:  "192.168.1.1",
			GatewayID: gatewayID,
		}

		resp, err := manager.ValidateAPIKey(ctx, req)
		require.NoError(t, err)
		assert.False(t, resp.IsValid)
		assert.Equal(t, "API key is not active", resp.Error)
	})

	t.Run("Error - Invalid client IP", func(t *testing.T) {
		req := &ValidateAPIKeyRequest{
			APIKey:    apiKey,
			ClientIP:  "invalid-ip",
			GatewayID: gatewayID,
		}

		resp, err := manager.ValidateAPIKey(ctx, req)
		require.NoError(t, err)
		assert.False(t, resp.IsValid)
		assert.Equal(t, "Invalid client IP address", resp.Error)
	})
}

// TestUpdateAPIKeyStatus tests the UpdateAPIKeyStatus function
func TestUpdateAPIKeyStatus(t *testing.T) {
	db := setupAPIKeyTestDB(t)
	manager := NewAPIKeyManager(db)
	ctx := context.Background()

	tenantID := createTestTenantForAPIKey(t, db)
	gatewayID := createTestGateway(t, db, tenantID)

	// Create an API key
	createReq := &CreateAPIKeyRequest{
		TenantID:  tenantID,
		GatewayID: gatewayID,
		KeyName:   "Status Test Key",
	}
	createResp, err := manager.CreateAPIKey(ctx, createReq)
	require.NoError(t, err)

	t.Run("Success - Deactivate API key", func(t *testing.T) {
		req := &UpdateAPIKeyStatusRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			KeyID:     createResp.KeyInfo.ID,
			IsActive:  false,
		}

		err := manager.UpdateAPIKeyStatus(ctx, req)
		require.NoError(t, err)

		// Verify status changed
		var apiKey GatewayAPIKey
		err = db.First(&apiKey, createResp.KeyInfo.ID).Error
		require.NoError(t, err)
		assert.False(t, apiKey.IsActive)
	})

	t.Run("Success - Reactivate API key", func(t *testing.T) {
		req := &UpdateAPIKeyStatusRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			KeyID:     createResp.KeyInfo.ID,
			IsActive:  true,
		}

		err := manager.UpdateAPIKeyStatus(ctx, req)
		require.NoError(t, err)

		// Verify status changed
		var apiKey GatewayAPIKey
		err = db.First(&apiKey, createResp.KeyInfo.ID).Error
		require.NoError(t, err)
		assert.True(t, apiKey.IsActive)
	})

	t.Run("Error - API key not found", func(t *testing.T) {
		req := &UpdateAPIKeyStatusRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			KeyID:     uuid.New(),
			IsActive:  false,
		}

		err := manager.UpdateAPIKeyStatus(ctx, req)
		assert.Error(t, err)
	})
}

// TestGetAPIKeysByGateway tests the GetAPIKeysByGateway function
func TestGetAPIKeysByGateway(t *testing.T) {
	db := setupAPIKeyTestDB(t)
	manager := NewAPIKeyManager(db)
	ctx := context.Background()

	tenantID := createTestTenantForAPIKey(t, db)
	gatewayID := createTestGateway(t, db, tenantID)

	// Create multiple API keys
	for i := 1; i <= 3; i++ {
		req := &CreateAPIKeyRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			KeyName:   "Test Key " + string(rune('0'+i)),
		}
		_, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)
	}

	t.Run("Success - Get all API keys for gateway", func(t *testing.T) {
		req := &GetAPIKeysByGatewayRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
		}

		apiKeys, err := manager.GetAPIKeysByGateway(ctx, req)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(apiKeys), 3)
	})

	t.Run("Error - Invalid tenant", func(t *testing.T) {
		req := &GetAPIKeysByGatewayRequest{
			TenantID:  uuid.New(),
			GatewayID: gatewayID,
		}

		_, err := manager.GetAPIKeysByGateway(ctx, req)
		assert.Error(t, err)
	})
}

// TestDeleteAPIKey tests the DeleteAPIKey function
func TestDeleteAPIKey(t *testing.T) {
	db := setupAPIKeyTestDB(t)
	manager := NewAPIKeyManager(db)
	ctx := context.Background()

	tenantID := createTestTenantForAPIKey(t, db)
	gatewayID := createTestGateway(t, db, tenantID)

	// Create an API key
	createReq := &CreateAPIKeyRequest{
		TenantID:  tenantID,
		GatewayID: gatewayID,
		KeyName:   "Delete Test Key",
	}
	createResp, err := manager.CreateAPIKey(ctx, createReq)
	require.NoError(t, err)

	t.Run("Success - Delete API key", func(t *testing.T) {
		req := &DeleteAPIKeyRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			KeyID:     createResp.KeyInfo.ID,
		}

		err := manager.DeleteAPIKey(ctx, req)
		require.NoError(t, err)

		// Verify key is deleted
		var apiKey GatewayAPIKey
		err = db.First(&apiKey, createResp.KeyInfo.ID).Error
		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})

	t.Run("Error - API key not found", func(t *testing.T) {
		req := &DeleteAPIKeyRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			KeyID:     uuid.New(),
		}

		err := manager.DeleteAPIKey(ctx, req)
		assert.Error(t, err)
	})
}

// TestGenerateAPIKey tests the generateAPIKey helper function
func TestGenerateAPIKey(t *testing.T) {
	t.Run("Success - Generate unique keys", func(t *testing.T) {
		key1, err := generateAPIKey()
		require.NoError(t, err)
		assert.NotEmpty(t, key1)
		assert.True(t, len(key1) > 10)
		assert.Contains(t, key1, "gw_")

		key2, err := generateAPIKey()
		require.NoError(t, err)
		assert.NotEmpty(t, key2)
		assert.NotEqual(t, key1, key2) // Keys should be unique
	})
}

// TestHashAPIKey tests the hashAPIKey helper function
func TestHashAPIKey(t *testing.T) {
	t.Run("Success - Hash API key", func(t *testing.T) {
		apiKey := "gw_test_key_12345"
		hash, err := hashAPIKey(apiKey)
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.NotEqual(t, apiKey, hash)

		// Verify hash can be compared
		err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(apiKey))
		assert.NoError(t, err)
	})

	t.Run("Success - Different keys produce different hashes", func(t *testing.T) {
		key1 := "gw_key_1"
		key2 := "gw_key_2"

		hash1, err := hashAPIKey(key1)
		require.NoError(t, err)

		hash2, err := hashAPIKey(key2)
		require.NoError(t, err)

		assert.NotEqual(t, hash1, hash2)
	})
}

// TestValidateIPAddresses tests the validateIPAddresses helper function
func TestValidateIPAddresses(t *testing.T) {
	t.Run("Success - Valid IPv4", func(t *testing.T) {
		err := validateIPAddresses([]string{"192.168.1.1", "10.0.0.1"})
		assert.NoError(t, err)
	})

	t.Run("Success - Valid IPv6", func(t *testing.T) {
		err := validateIPAddresses([]string{"2001:db8::1", "::1"})
		assert.NoError(t, err)
	})

	t.Run("Success - Valid CIDR", func(t *testing.T) {
		err := validateIPAddresses([]string{"192.168.0.0/24", "10.0.0.0/8"})
		assert.NoError(t, err)
	})

	t.Run("Success - Mixed valid addresses", func(t *testing.T) {
		err := validateIPAddresses([]string{"192.168.1.1", "10.0.0.0/24", "2001:db8::1"})
		assert.NoError(t, err)
	})

	t.Run("Error - Invalid IP", func(t *testing.T) {
		err := validateIPAddresses([]string{"invalid-ip"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid IP")
	})

	t.Run("Error - Invalid CIDR", func(t *testing.T) {
		err := validateIPAddresses([]string{"192.168.1.1/99"})
		assert.Error(t, err)
	})
}
