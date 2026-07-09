// Package gateway provides user management functionality for dedicated gateways
// Requirements: 11.1-11.9
package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserManager handles gateway user operations
type UserManager struct {
	db *gorm.DB
}

// NewUserManager creates a new user manager instance
func NewUserManager(db *gorm.DB) *UserManager {
	return &UserManager{
		db: db,
	}
}

// Role constants
const (
	RoleAdmin    = "admin"
	RoleOperator = "operator"
	RoleViewer   = "viewer"
)

// Permission constants
const (
	PermissionSendMessages   = "send_messages"
	PermissionViewReports    = "view_reports"
	PermissionManageUsers    = "manage_users"
	PermissionManageGateway  = "manage_gateway"
	PermissionManageAPIKeys  = "manage_api_keys"
	PermissionViewStatistics = "view_statistics"
	PermissionExportReports  = "export_reports"
)

// GatewayUser represents a user with access to a dedicated gateway
type GatewayUser struct {
	ID          uuid.UUID              `gorm:"type:uuid;primaryKey" json:"id"`
	TenantID    uuid.UUID              `gorm:"type:uuid;not null" json:"tenant_id"`
	GatewayID   uuid.UUID              `gorm:"type:uuid;not null" json:"gateway_id"`
	UserID      uuid.UUID              `gorm:"type:uuid;not null" json:"user_id"`
	Role        string                 `gorm:"size:50;not null" json:"role"`
	Permissions map[string]interface{} `gorm:"type:jsonb;default:'{}'" json:"permissions"`
	IsActive    bool                   `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time              `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time              `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName specifies the table name for GatewayUser
func (GatewayUser) TableName() string {
	return "gateway_users"
}

// AddUserRequest represents the request to add a user to a gateway
type AddUserRequest struct {
	TenantID    uuid.UUID              `json:"tenant_id" validate:"required"`
	GatewayID   uuid.UUID              `json:"gateway_id" validate:"required"`
	UserID      uuid.UUID              `json:"user_id" validate:"required"`
	Role        string                 `json:"role" validate:"required,oneof=admin operator viewer"`
	Permissions map[string]interface{} `json:"permissions"` // Optional, will use defaults if not provided
}

// AddUser adds a user to a gateway with specified role and permissions
// Requirements: 11.1, 11.2, 11.3, 11.4
func (um *UserManager) AddUser(ctx context.Context, req *AddUserRequest) (*GatewayUser, error) {
	// Validate tenant exists
	if err := um.validateTenant(ctx, req.TenantID); err != nil {
		return nil, fmt.Errorf("tenant validation failed: %w", err)
	}

	// Validate gateway exists and belongs to tenant
	if err := um.validateGateway(ctx, req.GatewayID, req.TenantID); err != nil {
		return nil, fmt.Errorf("gateway validation failed: %w", err)
	}

	// Validate user exists and belongs to tenant
	if err := um.validateUser(ctx, req.UserID, req.TenantID); err != nil {
		return nil, fmt.Errorf("user validation failed: %w", err)
	}

	// Validate role
	if !isValidRole(req.Role) {
		return nil, fmt.Errorf("invalid role: %s. Must be admin, operator, or viewer", req.Role)
	}

	// Check if user is already added to this gateway
	var existingUser GatewayUser
	err := um.db.WithContext(ctx).
		Where("gateway_id = ? AND user_id = ?", req.GatewayID, req.UserID).
		First(&existingUser).Error
	
	if err == nil {
		return nil, errors.New("user is already added to this gateway")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	// Set default permissions based on role if not provided
	permissions := req.Permissions
	if permissions == nil || len(permissions) == 0 {
		permissions = getDefaultPermissions(req.Role)
	}

	// Create gateway user
	gatewayUser := &GatewayUser{
		ID:          uuid.New(),
		TenantID:    req.TenantID,
		GatewayID:   req.GatewayID,
		UserID:      req.UserID,
		Role:        req.Role,
		Permissions: permissions,
		IsActive:    true,
	}

	// Insert into database
	if err := um.db.WithContext(ctx).Create(gatewayUser).Error; err != nil {
		return nil, fmt.Errorf("failed to add user to gateway: %w", err)
	}

	return gatewayUser, nil
}

// UpdatePermissionsRequest represents the request to update user permissions
type UpdatePermissionsRequest struct {
	TenantID    uuid.UUID              `json:"tenant_id" validate:"required"`
	GatewayID   uuid.UUID              `json:"gateway_id" validate:"required"`
	UserID      uuid.UUID              `json:"user_id" validate:"required"`
	Permissions map[string]interface{} `json:"permissions" validate:"required"`
}

// UpdatePermissions updates fine-grained permissions for a gateway user
// Requirements: 11.4
func (um *UserManager) UpdatePermissions(ctx context.Context, req *UpdatePermissionsRequest) (*GatewayUser, error) {
	// Validate tenant
	if err := um.validateTenant(ctx, req.TenantID); err != nil {
		return nil, fmt.Errorf("tenant validation failed: %w", err)
	}

	// Get existing gateway user
	var gatewayUser GatewayUser
	err := um.db.WithContext(ctx).
		Where("tenant_id = ? AND gateway_id = ? AND user_id = ?", req.TenantID, req.GatewayID, req.UserID).
		First(&gatewayUser).Error
	
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("gateway user not found or access denied")
		}
		return nil, fmt.Errorf("failed to fetch gateway user: %w", err)
	}

	// Validate permissions
	if err := validatePermissions(req.Permissions); err != nil {
		return nil, fmt.Errorf("invalid permissions: %w", err)
	}

	// Update permissions
	if err := um.db.WithContext(ctx).Model(&gatewayUser).Update("permissions", req.Permissions).Error; err != nil {
		return nil, fmt.Errorf("failed to update permissions: %w", err)
	}

	// Reload user
	if err := um.db.WithContext(ctx).Where("id = ?", gatewayUser.ID).First(&gatewayUser).Error; err != nil {
		return nil, fmt.Errorf("failed to reload gateway user: %w", err)
	}

	return &gatewayUser, nil
}

// CheckGatewayPermissionRequest represents the request to check user permission
type CheckGatewayPermissionRequest struct {
	TenantID   uuid.UUID `json:"tenant_id" validate:"required"`
	GatewayID  uuid.UUID `json:"gateway_id" validate:"required"`
	UserID     uuid.UUID `json:"user_id" validate:"required"`
	Permission string    `json:"permission" validate:"required"`
}

// CheckGatewayPermission checks if a user has a specific permission for a gateway
// Requirements: 11.7, 11.8, 11.9
func (um *UserManager) CheckGatewayPermission(ctx context.Context, req *CheckGatewayPermissionRequest) (bool, error) {
	// Get gateway user
	var gatewayUser GatewayUser
	err := um.db.WithContext(ctx).
		Where("tenant_id = ? AND gateway_id = ? AND user_id = ? AND is_active = ?", 
			req.TenantID, req.GatewayID, req.UserID, true).
		First(&gatewayUser).Error
	
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil // User not found or not active
		}
		return false, fmt.Errorf("failed to fetch gateway user: %w", err)
	}

	// Check permission
	hasPermission := checkPermission(gatewayUser.Permissions, req.Permission)
	return hasPermission, nil
}

// GetUsersByGatewayRequest represents the request to get users for a gateway
type GetUsersByGatewayRequest struct {
	TenantID  uuid.UUID `json:"tenant_id" validate:"required"`
	GatewayID uuid.UUID `json:"gateway_id" validate:"required"`
}

// GetUsersByGateway retrieves all users for a specific gateway
func (um *UserManager) GetUsersByGateway(ctx context.Context, req *GetUsersByGatewayRequest) ([]GatewayUser, error) {
	// Validate tenant
	if err := um.validateTenant(ctx, req.TenantID); err != nil {
		return nil, fmt.Errorf("tenant validation failed: %w", err)
	}

	// Validate gateway
	if err := um.validateGateway(ctx, req.GatewayID, req.TenantID); err != nil {
		return nil, fmt.Errorf("gateway validation failed: %w", err)
	}

	var users []GatewayUser
	err := um.db.WithContext(ctx).
		Where("tenant_id = ? AND gateway_id = ?", req.TenantID, req.GatewayID).
		Order("created_at DESC").
		Find(&users).Error
	
	if err != nil {
		return nil, fmt.Errorf("failed to fetch gateway users: %w", err)
	}

	return users, nil
}

// UpdateUserStatusRequest represents the request to update user status
type UpdateUserStatusRequest struct {
	TenantID  uuid.UUID `json:"tenant_id" validate:"required"`
	GatewayID uuid.UUID `json:"gateway_id" validate:"required"`
	UserID    uuid.UUID `json:"user_id" validate:"required"`
	IsActive  bool      `json:"is_active"`
}

// UpdateUserStatus activates or deactivates a gateway user
// Requirements: 11.5
func (um *UserManager) UpdateUserStatus(ctx context.Context, req *UpdateUserStatusRequest) error {
	// Validate tenant
	if err := um.validateTenant(ctx, req.TenantID); err != nil {
		return fmt.Errorf("tenant validation failed: %w", err)
	}

	// Update status
	result := um.db.WithContext(ctx).
		Model(&GatewayUser{}).
		Where("tenant_id = ? AND gateway_id = ? AND user_id = ?", req.TenantID, req.GatewayID, req.UserID).
		Update("is_active", req.IsActive)
	
	if result.Error != nil {
		return fmt.Errorf("failed to update user status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.New("gateway user not found or access denied")
	}

	return nil
}

// RemoveUserRequest represents the request to remove a user from a gateway
type RemoveUserRequest struct {
	TenantID  uuid.UUID `json:"tenant_id" validate:"required"`
	GatewayID uuid.UUID `json:"gateway_id" validate:"required"`
	UserID    uuid.UUID `json:"user_id" validate:"required"`
}

// RemoveUser removes a user from a gateway
func (um *UserManager) RemoveUser(ctx context.Context, req *RemoveUserRequest) error {
	// Validate tenant
	if err := um.validateTenant(ctx, req.TenantID); err != nil {
		return fmt.Errorf("tenant validation failed: %w", err)
	}

	// Delete user
	result := um.db.WithContext(ctx).
		Where("tenant_id = ? AND gateway_id = ? AND user_id = ?", req.TenantID, req.GatewayID, req.UserID).
		Delete(&GatewayUser{})
	
	if result.Error != nil {
		return fmt.Errorf("failed to remove user from gateway: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.New("gateway user not found or access denied")
	}

	return nil
}

// GetUserRole retrieves the user's role for a specific gateway
func (um *UserManager) GetUserRole(ctx context.Context, tenantID uuid.UUID, gatewayID uuid.UUID, userID uuid.UUID) (string, error) {
	var gatewayUser GatewayUser
	err := um.db.WithContext(ctx).
		Where("tenant_id = ? AND gateway_id = ? AND user_id = ? AND is_active = ?", 
			tenantID, gatewayID, userID, true).
		First(&gatewayUser).Error
	
	if err != nil {
		return "", err
	}

	return gatewayUser.Role, nil
}

// Helper methods

// validateTenant checks if tenant exists
func (um *UserManager) validateTenant(ctx context.Context, tenantID uuid.UUID) error {
	var count int64
	if err := um.db.WithContext(ctx).Table("tenants").Where("id = ?", tenantID).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to validate tenant: %w", err)
	}
	if count == 0 {
		return errors.New("tenant not found")
	}
	return nil
}

// validateGateway checks if gateway exists and belongs to tenant
func (um *UserManager) validateGateway(ctx context.Context, gatewayID uuid.UUID, tenantID uuid.UUID) error {
	var count int64
	err := um.db.WithContext(ctx).
		Table("dedicated_gateways").
		Where("id = ? AND tenant_id = ?", gatewayID, tenantID).
		Count(&count).Error
	
	if err != nil {
		return fmt.Errorf("failed to validate gateway: %w", err)
	}
	if count == 0 {
		return errors.New("gateway not found or does not belong to tenant")
	}
	return nil
}

// validateUser checks if user exists and belongs to tenant
func (um *UserManager) validateUser(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) error {
	var count int64
	err := um.db.WithContext(ctx).
		Table("users").
		Where("id = ? AND tenant_id = ?", userID, tenantID).
		Count(&count).Error
	
	if err != nil {
		return fmt.Errorf("failed to validate user: %w", err)
	}
	if count == 0 {
		return errors.New("user not found or does not belong to tenant")
	}
	return nil
}

// isValidRole checks if the role is valid
func isValidRole(role string) bool {
	return role == RoleAdmin || role == RoleOperator || role == RoleViewer
}

// getDefaultPermissions returns default permissions based on role
// Requirements: 11.7, 11.8, 11.9
func getDefaultPermissions(role string) map[string]interface{} {
	switch role {
	case RoleAdmin:
		// Admin: Full access to everything
		return map[string]interface{}{
			PermissionSendMessages:   true,
			PermissionViewReports:    true,
			PermissionManageUsers:    true,
			PermissionManageGateway:  true,
			PermissionManageAPIKeys:  true,
			PermissionViewStatistics: true,
			PermissionExportReports:  true,
		}
	case RoleOperator:
		// Operator: Can send messages and view reports
		return map[string]interface{}{
			PermissionSendMessages:   true,
			PermissionViewReports:    true,
			PermissionManageUsers:    false,
			PermissionManageGateway:  false,
			PermissionManageAPIKeys:  false,
			PermissionViewStatistics: true,
			PermissionExportReports:  true,
		}
	case RoleViewer:
		// Viewer: Can only view reports
		return map[string]interface{}{
			PermissionSendMessages:   false,
			PermissionViewReports:    true,
			PermissionManageUsers:    false,
			PermissionManageGateway:  false,
			PermissionManageAPIKeys:  false,
			PermissionViewStatistics: true,
			PermissionExportReports:  false,
		}
	default:
		return map[string]interface{}{}
	}
}

// validatePermissions validates that all permissions are valid
func validatePermissions(permissions map[string]interface{}) error {
	validPermissions := map[string]bool{
		PermissionSendMessages:   true,
		PermissionViewReports:    true,
		PermissionManageUsers:    true,
		PermissionManageGateway:  true,
		PermissionManageAPIKeys:  true,
		PermissionViewStatistics: true,
		PermissionExportReports:  true,
	}

	for key := range permissions {
		if !validPermissions[key] {
			return fmt.Errorf("invalid permission: %s", key)
		}
	}

	return nil
}

// checkPermission checks if a specific permission is granted
func checkPermission(permissions map[string]interface{}, permission string) bool {
	if permissions == nil {
		return false
	}

	value, exists := permissions[permission]
	if !exists {
		return false
	}

	// Handle different types of values
	switch v := value.(type) {
	case bool:
		return v
	case float64:
		return v != 0
	case string:
		return v == "true" || v == "1"
	case json.Number:
		num, _ := v.Float64()
		return num != 0
	default:
		return false
	}
}

