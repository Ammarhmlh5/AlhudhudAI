package gateway

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupUserManagerTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// :memory: sqlite is per-connection; pin the pool to a single connection
	// so tables created below are visible to later queries.
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	// Create tables
	err = db.Exec(`
		CREATE TABLE tenants (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			email TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE dedicated_gateways (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			gateway_name TEXT NOT NULL,
			gateway_code TEXT UNIQUE NOT NULL,
			status TEXT DEFAULT 'active',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE gateway_users (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			gateway_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			role TEXT NOT NULL CHECK (role IN ('admin', 'operator', 'viewer')),
			permissions TEXT DEFAULT '{}',
			is_active BOOLEAN DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(gateway_id, user_id)
		)
	`).Error
	require.NoError(t, err)

	return db
}

// seedTestData inserts test data
func seedUserManagerTestData(t *testing.T, db *gorm.DB) (tenantID uuid.UUID, gatewayID uuid.UUID, userID uuid.UUID) {
	tenantID = uuid.New()
	userID = uuid.New()
	gatewayID = uuid.New()

	// Insert tenant
	err := db.Exec("INSERT INTO tenants (id, name) VALUES (?, ?)", tenantID.String(), "Test Tenant").Error
	require.NoError(t, err)

	// Insert user
	err = db.Exec("INSERT INTO users (id, tenant_id, email) VALUES (?, ?, ?)", 
		userID.String(), tenantID.String(), "test@example.com").Error
	require.NoError(t, err)

	// Insert gateway
	err = db.Exec("INSERT INTO dedicated_gateways (id, tenant_id, gateway_name, gateway_code) VALUES (?, ?, ?, ?)",
		gatewayID.String(), tenantID.String(), "Test Gateway", "TEST_GW").Error
	require.NoError(t, err)

	return tenantID, gatewayID, userID
}

func TestAddUser(t *testing.T) {
	db := setupUserManagerTestDB(t)
	tenantID, gatewayID, userID := seedUserManagerTestData(t, db)
	
	um := NewUserManager(db)
	ctx := context.Background()

	t.Run("Add admin user successfully", func(t *testing.T) {
		req := &AddUserRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			UserID:    userID,
			Role:      RoleAdmin,
		}

		gatewayUser, err := um.AddUser(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, gatewayUser)
		assert.Equal(t, tenantID, gatewayUser.TenantID)
		assert.Equal(t, gatewayID, gatewayUser.GatewayID)
		assert.Equal(t, userID, gatewayUser.UserID)
		assert.Equal(t, RoleAdmin, gatewayUser.Role)
		assert.True(t, gatewayUser.IsActive)

		// Check default admin permissions
		assert.Equal(t, true, gatewayUser.Permissions[PermissionSendMessages])
		assert.Equal(t, true, gatewayUser.Permissions[PermissionViewReports])
		assert.Equal(t, true, gatewayUser.Permissions[PermissionManageUsers])
		assert.Equal(t, true, gatewayUser.Permissions[PermissionManageGateway])
		assert.Equal(t, true, gatewayUser.Permissions[PermissionManageAPIKeys])
	})

	t.Run("Add operator user successfully", func(t *testing.T) {
		// Create another user
		operatorUserID := uuid.New()
		err := db.Exec("INSERT INTO users (id, tenant_id, email) VALUES (?, ?, ?)", 
			operatorUserID.String(), tenantID.String(), "operator@example.com").Error
		require.NoError(t, err)

		req := &AddUserRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			UserID:    operatorUserID,
			Role:      RoleOperator,
		}

		gatewayUser, err := um.AddUser(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, gatewayUser)
		assert.Equal(t, RoleOperator, gatewayUser.Role)

		// Check default operator permissions
		assert.Equal(t, true, gatewayUser.Permissions[PermissionSendMessages])
		assert.Equal(t, true, gatewayUser.Permissions[PermissionViewReports])
		assert.Equal(t, false, gatewayUser.Permissions[PermissionManageUsers])
		assert.Equal(t, false, gatewayUser.Permissions[PermissionManageGateway])
	})

	t.Run("Add viewer user successfully", func(t *testing.T) {
		// Create another user
		viewerUserID := uuid.New()
		err := db.Exec("INSERT INTO users (id, tenant_id, email) VALUES (?, ?, ?)", 
			viewerUserID.String(), tenantID.String(), "viewer@example.com").Error
		require.NoError(t, err)

		req := &AddUserRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			UserID:    viewerUserID,
			Role:      RoleViewer,
		}

		gatewayUser, err := um.AddUser(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, gatewayUser)
		assert.Equal(t, RoleViewer, gatewayUser.Role)

		// Check default viewer permissions
		assert.Equal(t, false, gatewayUser.Permissions[PermissionSendMessages])
		assert.Equal(t, true, gatewayUser.Permissions[PermissionViewReports])
		assert.Equal(t, false, gatewayUser.Permissions[PermissionManageUsers])
		assert.Equal(t, false, gatewayUser.Permissions[PermissionExportReports])
	})

	t.Run("Add user with custom permissions", func(t *testing.T) {
		// Create another user
		customUserID := uuid.New()
		err := db.Exec("INSERT INTO users (id, tenant_id, email) VALUES (?, ?, ?)", 
			customUserID.String(), tenantID.String(), "custom@example.com").Error
		require.NoError(t, err)

		customPermissions := map[string]interface{}{
			PermissionSendMessages: true,
			PermissionViewReports:  true,
			PermissionManageUsers:  false,
		}

		req := &AddUserRequest{
			TenantID:    tenantID,
			GatewayID:   gatewayID,
			UserID:      customUserID,
			Role:        RoleOperator,
			Permissions: customPermissions,
		}

		gatewayUser, err := um.AddUser(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, gatewayUser)
		assert.Equal(t, customPermissions[PermissionSendMessages], gatewayUser.Permissions[PermissionSendMessages])
	})

	t.Run("Fail to add duplicate user", func(t *testing.T) {
		req := &AddUserRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			UserID:    userID, // Already added
			Role:      RoleAdmin,
		}

		_, err := um.AddUser(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already added")
	})

	t.Run("Fail with invalid role", func(t *testing.T) {
		invalidUserID := uuid.New()
		err := db.Exec("INSERT INTO users (id, tenant_id, email) VALUES (?, ?, ?)", 
			invalidUserID.String(), tenantID.String(), "invalid@example.com").Error
		require.NoError(t, err)

		req := &AddUserRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			UserID:    invalidUserID,
			Role:      "invalid_role",
		}

		_, err = um.AddUser(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid role")
	})

	t.Run("Fail with non-existent tenant", func(t *testing.T) {
		nonExistentTenantID := uuid.New()
		nonExistentUserID := uuid.New()

		req := &AddUserRequest{
			TenantID:  nonExistentTenantID,
			GatewayID: gatewayID,
			UserID:    nonExistentUserID,
			Role:      RoleAdmin,
		}

		_, err := um.AddUser(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tenant")
	})
}

func TestUpdatePermissions(t *testing.T) {
	db := setupUserManagerTestDB(t)
	tenantID, gatewayID, userID := seedUserManagerTestData(t, db)
	
	um := NewUserManager(db)
	ctx := context.Background()

	// First add a user
	addReq := &AddUserRequest{
		TenantID:  tenantID,
		GatewayID: gatewayID,
		UserID:    userID,
		Role:      RoleOperator,
	}
	_, err := um.AddUser(ctx, addReq)
	require.NoError(t, err)

	t.Run("Update permissions successfully", func(t *testing.T) {
		newPermissions := map[string]interface{}{
			PermissionSendMessages:  false,
			PermissionViewReports:   true,
			PermissionManageUsers:   true,
			PermissionManageGateway: false,
		}

		req := &UpdatePermissionsRequest{
			TenantID:    tenantID,
			GatewayID:   gatewayID,
			UserID:      userID,
			Permissions: newPermissions,
		}

		gatewayUser, err := um.UpdatePermissions(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, gatewayUser)
		assert.Equal(t, false, gatewayUser.Permissions[PermissionSendMessages])
		assert.Equal(t, true, gatewayUser.Permissions[PermissionViewReports])
		assert.Equal(t, true, gatewayUser.Permissions[PermissionManageUsers])
	})

	t.Run("Fail with invalid permission", func(t *testing.T) {
		invalidPermissions := map[string]interface{}{
			"invalid_permission": true,
		}

		req := &UpdatePermissionsRequest{
			TenantID:    tenantID,
			GatewayID:   gatewayID,
			UserID:      userID,
			Permissions: invalidPermissions,
		}

		_, err := um.UpdatePermissions(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid permission")
	})

	t.Run("Fail with non-existent user", func(t *testing.T) {
		nonExistentUserID := uuid.New()

		req := &UpdatePermissionsRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			UserID:    nonExistentUserID,
			Permissions: map[string]interface{}{
				PermissionSendMessages: true,
			},
		}

		_, err := um.UpdatePermissions(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestCheckGatewayPermission(t *testing.T) {
	db := setupUserManagerTestDB(t)
	tenantID, gatewayID, userID := seedUserManagerTestData(t, db)
	
	um := NewUserManager(db)
	ctx := context.Background()

	// Add an admin user
	addReq := &AddUserRequest{
		TenantID:  tenantID,
		GatewayID: gatewayID,
		UserID:    userID,
		Role:      RoleAdmin,
	}
	_, err := um.AddUser(ctx, addReq)
	require.NoError(t, err)

	t.Run("Check permission - has permission", func(t *testing.T) {
		req := &CheckGatewayPermissionRequest{
			TenantID:   tenantID,
			GatewayID:  gatewayID,
			UserID:     userID,
			Permission: PermissionManageUsers,
		}

		hasPermission, err := um.CheckGatewayPermission(ctx, req)
		require.NoError(t, err)
		assert.True(t, hasPermission)
	})

	t.Run("Check permission - does not have permission", func(t *testing.T) {
		// Add a viewer user
		viewerUserID := uuid.New()
		err := db.Exec("INSERT INTO users (id, tenant_id, email) VALUES (?, ?, ?)", 
			viewerUserID.String(), tenantID.String(), "viewer@example.com").Error
		require.NoError(t, err)

		addReq := &AddUserRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			UserID:    viewerUserID,
			Role:      RoleViewer,
		}
		_, err = um.AddUser(ctx, addReq)
		require.NoError(t, err)

		req := &CheckGatewayPermissionRequest{
			TenantID:   tenantID,
			GatewayID:  gatewayID,
			UserID:     viewerUserID,
			Permission: PermissionSendMessages,
		}

		hasPermission, err := um.CheckGatewayPermission(ctx, req)
		require.NoError(t, err)
		assert.False(t, hasPermission)
	})

	t.Run("Check permission - user not found", func(t *testing.T) {
		nonExistentUserID := uuid.New()

		req := &CheckGatewayPermissionRequest{
			TenantID:   tenantID,
			GatewayID:  gatewayID,
			UserID:     nonExistentUserID,
			Permission: PermissionSendMessages,
		}

		hasPermission, err := um.CheckGatewayPermission(ctx, req)
		require.NoError(t, err)
		assert.False(t, hasPermission)
	})
}

func TestUpdateUserStatus(t *testing.T) {
	db := setupUserManagerTestDB(t)
	tenantID, gatewayID, userID := seedUserManagerTestData(t, db)
	
	um := NewUserManager(db)
	ctx := context.Background()

	// Add a user
	addReq := &AddUserRequest{
		TenantID:  tenantID,
		GatewayID: gatewayID,
		UserID:    userID,
		Role:      RoleOperator,
	}
	_, err := um.AddUser(ctx, addReq)
	require.NoError(t, err)

	t.Run("Deactivate user successfully", func(t *testing.T) {
		req := &UpdateUserStatusRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			UserID:    userID,
			IsActive:  false,
		}

		err := um.UpdateUserStatus(ctx, req)
		require.NoError(t, err)

		// Verify status changed
		var gatewayUser GatewayUser
		err = db.Where("gateway_id = ? AND user_id = ?", gatewayID, userID).First(&gatewayUser).Error
		require.NoError(t, err)
		assert.False(t, gatewayUser.IsActive)
	})

	t.Run("Reactivate user successfully", func(t *testing.T) {
		req := &UpdateUserStatusRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			UserID:    userID,
			IsActive:  true,
		}

		err := um.UpdateUserStatus(ctx, req)
		require.NoError(t, err)

		// Verify status changed
		var gatewayUser GatewayUser
		err = db.Where("gateway_id = ? AND user_id = ?", gatewayID, userID).First(&gatewayUser).Error
		require.NoError(t, err)
		assert.True(t, gatewayUser.IsActive)
	})

	t.Run("Fail with non-existent user", func(t *testing.T) {
		nonExistentUserID := uuid.New()

		req := &UpdateUserStatusRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			UserID:    nonExistentUserID,
			IsActive:  false,
		}

		err := um.UpdateUserStatus(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestGetUsersByGateway(t *testing.T) {
	db := setupUserManagerTestDB(t)
	tenantID, gatewayID, userID := seedUserManagerTestData(t, db)
	
	um := NewUserManager(db)
	ctx := context.Background()

	// Add multiple users
	addReq1 := &AddUserRequest{
		TenantID:  tenantID,
		GatewayID: gatewayID,
		UserID:    userID,
		Role:      RoleAdmin,
	}
	_, err := um.AddUser(ctx, addReq1)
	require.NoError(t, err)

	user2ID := uuid.New()
	err = db.Exec("INSERT INTO users (id, tenant_id, email) VALUES (?, ?, ?)", 
		user2ID.String(), tenantID.String(), "user2@example.com").Error
	require.NoError(t, err)

	addReq2 := &AddUserRequest{
		TenantID:  tenantID,
		GatewayID: gatewayID,
		UserID:    user2ID,
		Role:      RoleOperator,
	}
	_, err = um.AddUser(ctx, addReq2)
	require.NoError(t, err)

	t.Run("Get all users for gateway", func(t *testing.T) {
		req := &GetUsersByGatewayRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
		}

		users, err := um.GetUsersByGateway(ctx, req)
		require.NoError(t, err)
		assert.Len(t, users, 2)
	})

	t.Run("Fail with non-existent gateway", func(t *testing.T) {
		req := &GetUsersByGatewayRequest{
			TenantID:  tenantID,
			GatewayID: uuid.New(),
		}

		_, err := um.GetUsersByGateway(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "gateway")
	})
}

func TestRemoveUser(t *testing.T) {
	db := setupUserManagerTestDB(t)
	tenantID, gatewayID, userID := seedUserManagerTestData(t, db)
	
	um := NewUserManager(db)
	ctx := context.Background()

	// Add a user
	addReq := &AddUserRequest{
		TenantID:  tenantID,
		GatewayID: gatewayID,
		UserID:    userID,
		Role:      RoleOperator,
	}
	_, err := um.AddUser(ctx, addReq)
	require.NoError(t, err)

	t.Run("Remove user successfully", func(t *testing.T) {
		req := &RemoveUserRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			UserID:    userID,
		}

		err := um.RemoveUser(ctx, req)
		require.NoError(t, err)

		// Verify user removed
		var count int64
		err = db.Model(&GatewayUser{}).
			Where("gateway_id = ? AND user_id = ?", gatewayID, userID).
			Count(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("Fail with non-existent user", func(t *testing.T) {
		nonExistentUserID := uuid.New()

		req := &RemoveUserRequest{
			TenantID:  tenantID,
			GatewayID: gatewayID,
			UserID:    nonExistentUserID,
		}

		err := um.RemoveUser(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestRoleBasedAccessControl(t *testing.T) {
	db := setupUserManagerTestDB(t)
	tenantID, gatewayID, _ := seedUserManagerTestData(t, db)
	
	um := NewUserManager(db)
	ctx := context.Background()

	// Create users with different roles
	adminUserID := uuid.New()
	operatorUserID := uuid.New()
	viewerUserID := uuid.New()

	// Insert users
	for _, userID := range []uuid.UUID{adminUserID, operatorUserID, viewerUserID} {
		err := db.Exec("INSERT INTO users (id, tenant_id, email) VALUES (?, ?, ?)", 
			userID.String(), tenantID.String(), userID.String()+"@example.com").Error
		require.NoError(t, err)
	}

	// Add admin
	_, err := um.AddUser(ctx, &AddUserRequest{
		TenantID:  tenantID,
		GatewayID: gatewayID,
		UserID:    adminUserID,
		Role:      RoleAdmin,
	})
	require.NoError(t, err)

	// Add operator
	_, err = um.AddUser(ctx, &AddUserRequest{
		TenantID:  tenantID,
		GatewayID: gatewayID,
		UserID:    operatorUserID,
		Role:      RoleOperator,
	})
	require.NoError(t, err)

	// Add viewer
	_, err = um.AddUser(ctx, &AddUserRequest{
		TenantID:  tenantID,
		GatewayID: gatewayID,
		UserID:    viewerUserID,
		Role:      RoleViewer,
	})
	require.NoError(t, err)

	t.Run("Admin has all permissions", func(t *testing.T) {
		permissions := []string{
			PermissionSendMessages,
			PermissionViewReports,
			PermissionManageUsers,
			PermissionManageGateway,
			PermissionManageAPIKeys,
		}

		for _, perm := range permissions {
			hasPermission, err := um.CheckGatewayPermission(ctx, &CheckGatewayPermissionRequest{
				TenantID:   tenantID,
				GatewayID:  gatewayID,
				UserID:     adminUserID,
				Permission: perm,
			})
			require.NoError(t, err)
			assert.True(t, hasPermission, "Admin should have permission: %s", perm)
		}
	})

	t.Run("Operator has limited permissions", func(t *testing.T) {
		// Should have
		hasPermission, err := um.CheckGatewayPermission(ctx, &CheckGatewayPermissionRequest{
			TenantID:   tenantID,
			GatewayID:  gatewayID,
			UserID:     operatorUserID,
			Permission: PermissionSendMessages,
		})
		require.NoError(t, err)
		assert.True(t, hasPermission)

		// Should not have
		hasPermission, err = um.CheckGatewayPermission(ctx, &CheckGatewayPermissionRequest{
			TenantID:   tenantID,
			GatewayID:  gatewayID,
			UserID:     operatorUserID,
			Permission: PermissionManageUsers,
		})
		require.NoError(t, err)
		assert.False(t, hasPermission)
	})

	t.Run("Viewer has minimal permissions", func(t *testing.T) {
		// Should have
		hasPermission, err := um.CheckGatewayPermission(ctx, &CheckGatewayPermissionRequest{
			TenantID:   tenantID,
			GatewayID:  gatewayID,
			UserID:     viewerUserID,
			Permission: PermissionViewReports,
		})
		require.NoError(t, err)
		assert.True(t, hasPermission)

		// Should not have
		hasPermission, err = um.CheckGatewayPermission(ctx, &CheckGatewayPermissionRequest{
			TenantID:   tenantID,
			GatewayID:  gatewayID,
			UserID:     viewerUserID,
			Permission: PermissionSendMessages,
		})
		require.NoError(t, err)
		assert.False(t, hasPermission)

		hasPermission, err = um.CheckGatewayPermission(ctx, &CheckGatewayPermissionRequest{
			TenantID:   tenantID,
			GatewayID:  gatewayID,
			UserID:     viewerUserID,
			Permission: PermissionExportReports,
		})
		require.NoError(t, err)
		assert.False(t, hasPermission)
	})
}

