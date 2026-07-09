package reseller

import (
	"context"
	"errors"
	"fmt"

	"github.com/al-hudhud/backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) ListTenants(ctx context.Context, resellerID uuid.UUID) ([]models.Tenant, error) {
	var tenants []models.Tenant
	err := s.db.WithContext(ctx).Where("parent_id = ?", resellerID).Find(&tenants).Error
	return tenants, err
}

func (s *Service) CreateTenant(ctx context.Context, resellerID uuid.UUID, tenant *models.Tenant, user *models.User) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Create User
		user.UserType = models.TenantType
		if err := tx.Create(user).Error; err != nil {
			return err
		}

		// 2. Setup Tenant
		tenant.UserID = user.ID
		tenant.ParentID = &resellerID
		tenant.IsReseller = false
		if tenant.CompanySerial == "" {
			tenant.CompanySerial = fmt.Sprintf("C-%d", user.ID.ID()) // Simple serial generator
		}

		if err := tx.Create(tenant).Error; err != nil {
			return err
		}

		return nil
	})
}

func (s *Service) GetTenant(ctx context.Context, resellerID, tenantID uuid.UUID) (*models.Tenant, error) {
	var tenant models.Tenant
	err := s.db.WithContext(ctx).Where("id = ? AND parent_id = ?", tenantID, resellerID).First(&tenant).Error
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (s *Service) UpdateTenantAPIKeys(ctx context.Context, resellerID, tenantID uuid.UUID, accountingKey string) error {
	result := s.db.WithContext(ctx).Model(&models.Tenant{}).
		Where("id = ? AND parent_id = ?", tenantID, resellerID).
		Update("api_key_accounting", accountingKey)
	
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("tenant not found or not owned by reseller")
	}
	return nil
}

func (s *Service) ListStaff(ctx context.Context, resellerID uuid.UUID) ([]models.StaffProfile, error) {
	var staff []models.StaffProfile
	err := s.db.WithContext(ctx).Preload("User").Preload("Role").Where("tenant_id = ?", resellerID).Find(&staff).Error
	return staff, err
}
