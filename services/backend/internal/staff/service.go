package staff

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/al-hudhud/backend/internal/cache"
	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/pkg/password"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrStaffNotFound = errors.New("staff member not found")
	ErrRoleNotFound  = errors.New("role not found")
)

// Service handles staff management business logic
type Service struct {
	cache *cache.QueryCache
}

// NewService creates a new staff service
func NewService(qc *cache.QueryCache) *Service {
	return &Service{cache: qc}
}

// CreateStaff creates a new staff member for a specific tenant
func (s *Service) CreateStaff(tenantID uuid.UUID, email, fullName, roleIDStr, department, passwordStr string) (*models.StaffProfile, error) {
	// Hash password
	passwordHash, err := password.HashPassword(passwordStr)
	if err != nil {
		return nil, err
	}

	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		return nil, errors.New("invalid role id format")
	}

	staff := &models.StaffProfile{
		User: models.User{
			Email:        email,
			PasswordHash: passwordHash,
			UserType:     models.StaffType,
			IsActive:     true,
		},
		TenantID:   tenantID,
		FullName:   fullName,
		RoleID:     roleID,
		Department: department,
		IsActive:   true,
	}

	err = db.GormDB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&staff.User).Error; err != nil {
			return err
		}
		staff.UserID = staff.User.ID
		if err := tx.Create(staff).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Preload Role for the response
	db.GormDB.Preload("Role").First(staff, staff.ID)

	return staff, nil
}

// ListStaff returns all staff members for a specific tenant with optional filters
func (s *Service) ListStaff(tenantID uuid.UUID, search, roleID string, department string) ([]models.StaffProfile, error) {
	var staffList []models.StaffProfile
	query := db.GormDB.Preload("Role").Preload("User")
	
	if tenantID != uuid.Nil {
		query = query.Where("tenant_id = ?", tenantID)
	}

	if search != "" {
		searchTerm := "%" + search + "%"
		query = query.Where("full_name ILIKE ? OR department ILIKE ?", searchTerm, searchTerm)
	}

	if roleID != "" {
		query = query.Where("role_id = ?", roleID)
	}

	if department != "" {
		query = query.Where("department = ?", department)
	}

	err := query.Find(&staffList).Error
	return staffList, err
}

// GetStaff returns a staff member by ID with caching and tenant isolation
func (s *Service) GetStaff(ctx context.Context, tenantID uuid.UUID, staffID string) (*models.StaffProfile, error) {
	if s.cache == nil {
		var staff models.StaffProfile
		err := db.GormDB.Preload("Role").Preload("User").First(&staff, "id = ? AND tenant_id = ?", staffID, tenantID).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrStaffNotFound
			}
			return nil, err
		}
		return &staff, nil
	}

	key := fmt.Sprintf("staff:%s:%s", tenantID.String(), staffID)
	val, err := s.cache.GetOrSet(ctx, key, s.cache.GetTTL("staff"), func() (interface{}, error) {
		var staff models.StaffProfile
		err := db.GormDB.Preload("Role").Preload("User").First(&staff, "id = ? AND tenant_id = ?", staffID, tenantID).Error
		if err != nil {
			return nil, err
		}
		return &staff, nil
	})

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrStaffNotFound
		}
		return nil, err
	}

	if staff, ok := val.(*models.StaffProfile); ok {
		return staff, nil
	}

	data, _ := json.Marshal(val)
	var staff models.StaffProfile
	if err := json.Unmarshal(data, &staff); err != nil {
		return nil, err
	}
	return &staff, nil
}

// InvalidateCache clears the cache for a staff member
func (s *Service) InvalidateCache(ctx context.Context, staffID string) {
	if s.cache == nil {
		return
	}
	s.cache.Delete(ctx, fmt.Sprintf("staff:%s", staffID))
}

// UpdateStaff updates a staff member for a specific tenant
func (s *Service) UpdateStaff(ctx context.Context, tenantID uuid.UUID, staffID, fullName, roleIDStr, department string, isActive bool) error {
	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		return errors.New("invalid role id format")
	}

	result := db.GormDB.Model(&models.StaffProfile{}).Where("id = ? AND tenant_id = ?", staffID, tenantID).Updates(map[string]interface{}{
		"full_name":  fullName,
		"role_id":    roleID,
		"department": department,
		"is_active":  isActive,
	})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrStaffNotFound
	}

	s.InvalidateCache(ctx, staffID)
	return nil
}

// DeleteStaff deletes a staff member (soft-delete) with tenant isolation
func (s *Service) DeleteStaff(ctx context.Context, tenantID uuid.UUID, staffID string) error {
	var staff models.StaffProfile
	if err := db.GormDB.First(&staff, "id = ? AND tenant_id = ?", staffID, tenantID).Error; err != nil {
		return ErrStaffNotFound
	}

	// Delete user (cascade will handle staff profile if set, but we can do it explicitly or rely on GORM if configured)
	// Our model has constraint:OnDelete:CASCADE
	return db.GormDB.Delete(&models.User{}, staff.UserID).Error
}

// ListRoles returns all roles (system + tenant-specific)
func (s *Service) ListRoles(tenantID uuid.UUID) ([]models.Role, error) {
	var roles []models.Role
	query := db.GormDB.Preload("Permissions")
	
	if tenantID != uuid.Nil {
		query = query.Where("tenant_id = ? OR tenant_id IS NULL", tenantID)
	}
	
	err := query.Find(&roles).Error
	return roles, err
}

// CreateRole creates a new role with permissions for a specific tenant
func (s *Service) CreateRole(tenantID uuid.UUID, name, description string, permissionsNames []string) (*models.Role, error) {
	var permissions []models.Permission
	if len(permissionsNames) > 0 {
		if err := db.GormDB.Where("name IN ?", permissionsNames).Find(&permissions).Error; err != nil {
			return nil, err
		}
	}

	var tID *uuid.UUID
	if tenantID != uuid.Nil {
		tID = &tenantID
	}

	role := &models.Role{
		TenantID:    tID,
		Name:        name,
		Description: description,
		Permissions: permissions,
	}

	if err := db.GormDB.Create(role).Error; err != nil {
		return nil, err
	}

	return role, nil
}

// UpdateRole updates a role and its permissions with tenant isolation
func (s *Service) UpdateRole(tenantID uuid.UUID, roleIDStr, name, description string, permissionsNames []string) error {
	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		return errors.New("invalid role id format")
	}

	var role models.Role
	if err := db.GormDB.First(&role, "id = ? AND tenant_id = ?", roleID, tenantID).Error; err != nil {
		return ErrRoleNotFound
	}

	var permissions []models.Permission
	if len(permissionsNames) > 0 {
		if err := db.GormDB.Where("name IN ?", permissionsNames).Find(&permissions).Error; err != nil {
			return err
		}
	}

	return db.GormDB.Transaction(func(tx *gorm.DB) error {
		// Update basic info
		if err := tx.Model(&role).Updates(models.Role{Name: name, Description: description}).Error; err != nil {
			return err
		}
		// Replace permissions
		if err := tx.Model(&role).Association("Permissions").Replace(permissions); err != nil {
			return err
		}
		return nil
	})
}

// DeleteRole deletes a role with tenant isolation
func (s *Service) DeleteRole(tenantID uuid.UUID, roleID string) error {
	result := db.GormDB.Delete(&models.Role{}, "id = ? AND tenant_id = ?", roleID, tenantID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrRoleNotFound
	}
	return nil
}

// ListPermissions returns all available permissions
func (s *Service) ListPermissions() ([]models.Permission, error) {
	var permissions []models.Permission
	err := db.GormDB.Order("resource, action").Find(&permissions).Error
	return permissions, err
}

// BulkDeleteStaff deletes multiple staff members with tenant isolation
func (s *Service) BulkDeleteStaff(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) error {
	return db.GormDB.Transaction(func(tx *gorm.DB) error {
		var staffProfiles []models.StaffProfile
		if err := tx.Select("user_id").Where("id IN ? AND tenant_id = ?", ids, tenantID).Find(&staffProfiles).Error; err != nil {
			return err
		}

		userIDs := make([]uuid.UUID, len(staffProfiles))
		for i, sp := range staffProfiles {
			userIDs[i] = sp.UserID
		}

		// Explicitly soft-delete staff profiles
		if err := tx.Delete(&models.StaffProfile{}, "id IN ?", ids).Error; err != nil {
			return err
		}

		// Soft-delete users
		if err := tx.Delete(&models.User{}, "id IN ?", userIDs).Error; err != nil {
			return err
		}
		return nil
	})
}

// BulkUpdateStaff updates multiple staff members with tenant isolation
func (s *Service) BulkUpdateStaff(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID, roleID uuid.UUID, isActive bool) error {
	return db.GormDB.Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{"is_active": isActive}
		if roleID != uuid.Nil {
			updates["role_id"] = roleID
		}

		if err := tx.Model(&models.StaffProfile{}).Where("id IN ? AND tenant_id = ?", ids, tenantID).Updates(updates).Error; err != nil {
			return err
		}

		// Invalidate cache
		for _, id := range ids {
			s.InvalidateCache(ctx, id.String())
		}
		return nil
	})
}
