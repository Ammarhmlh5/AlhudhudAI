package db

import (
	"github.com/al-hudhud/backend/internal/models"
)

// GetUserPermissions retrieves all permissions for a user using GORM
func GetUserPermissions(userID string) ([]string, error) {
	var user models.User
	if err := GormDB.Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}

	// Super admin has all permissions
	if user.UserType == models.SuperAdmin {
		return []string{"*"}, nil
	}

	// Staff members get permissions from their role
	if user.UserType == models.StaffType {
		var permissions []string

		err := GormDB.Table("permissions").
			Select("DISTINCT permissions.name").
			Joins("JOIN role_permissions ON permissions.id = role_permissions.permission_id").
			Joins("JOIN staff_profiles ON role_permissions.role_id = staff_profiles.role_id").
			Where("staff_profiles.user_id = ? AND staff_profiles.is_active = ?", userID, true).
			Pluck("name", &permissions).Error

		if err != nil {
			return nil, err
		}
		return permissions, nil
	}

	// Tenants have full permissions within their isolated scope
	if user.UserType == models.TenantType {
		return []string{"*"}, nil
	}

	return []string{}, nil
}

// HasPermission checks if user has a specific permission
func HasPermission(userID string, permission string) (bool, error) {
	permissions, err := GetUserPermissions(userID)
	if err != nil {
		return false, err
	}

	for _, p := range permissions {
		if p == permission || p == "*" {
			return true, nil
		}
	}

	return false, nil
}
