package tariff

import (
	"context"
	"errors"
	"fmt"

	"github.com/al-hudhud/backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ServiceRepository handles database operations for services
type ServiceRepository struct {
	db *gorm.DB
}

// NewServiceRepository creates a new service repository
func NewServiceRepository(db *gorm.DB) *ServiceRepository {
	return &ServiceRepository{db: db}
}

// Create creates a new service
func (r *ServiceRepository) Create(ctx context.Context, service *models.Service) error {
	// Validate service
	if err := service.Validate(); err != nil {
		return err
	}

	// Check if service with same code already exists for this tenant
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.Service{}).
		Where("tenant_id = ? AND code = ?", service.TenantID, service.Code).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check service existence: %w", err)
	}

	if count > 0 {
		return models.ErrServiceAlreadyExists
	}

	// Create service
	if err := r.db.WithContext(ctx).Create(service).Error; err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	return nil
}

// GetByID retrieves a service by ID and tenant ID
func (r *ServiceRepository) GetByID(ctx context.Context, tenantID uuid.UUID, serviceID int) (*models.Service, error) {
	var service models.Service

	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, serviceID).
		First(&service).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, models.ErrServiceNotFound
		}
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	return &service, nil
}

// GetByCode retrieves a service by code and tenant ID
func (r *ServiceRepository) GetByCode(ctx context.Context, tenantID uuid.UUID, code models.ServiceCode) (*models.Service, error) {
	var service models.Service

	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND code = ?", tenantID, code).
		First(&service).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, models.ErrServiceNotFound
		}
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	return &service, nil
}

// GetByType retrieves all services of a specific type for a tenant
func (r *ServiceRepository) GetByType(ctx context.Context, tenantID uuid.UUID, serviceType models.ServiceType) ([]*models.Service, error) {
	var services []*models.Service

	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND type = ?", tenantID, serviceType).
		Order("id ASC").
		Find(&services).Error; err != nil {
		return nil, fmt.Errorf("failed to get services by type: %w", err)
	}

	return services, nil
}

// GetAll retrieves all services for a tenant
func (r *ServiceRepository) GetAll(ctx context.Context, tenantID uuid.UUID) ([]*models.Service, error) {
	var services []*models.Service

	if err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("id ASC").
		Find(&services).Error; err != nil {
		return nil, fmt.Errorf("failed to get all services: %w", err)
	}

	return services, nil
}

// GetActive retrieves all active services for a tenant
func (r *ServiceRepository) GetActive(ctx context.Context, tenantID uuid.UUID) ([]*models.Service, error) {
	var services []*models.Service

	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND is_active = ?", tenantID, true).
		Order("id ASC").
		Find(&services).Error; err != nil {
		return nil, fmt.Errorf("failed to get active services: %w", err)
	}

	return services, nil
}

// Update updates a service
func (r *ServiceRepository) Update(ctx context.Context, service *models.Service) error {
	// Validate service
	if err := service.Validate(); err != nil {
		return err
	}

	// Check if service exists
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.Service{}).
		Where("tenant_id = ? AND id = ?", service.TenantID, service.ID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check service existence: %w", err)
	}

	if count == 0 {
		return models.ErrServiceNotFound
	}

	// Update service
	if err := r.db.WithContext(ctx).
		Model(&models.Service{}).
		Where("tenant_id = ? AND id = ?", service.TenantID, service.ID).
		Updates(map[string]interface{}{
			"name":        service.Name,
			"description": service.Description,
			"is_active":   service.IsActive,
			"updated_at":  gorm.Expr("CURRENT_TIMESTAMP"),
		}).Error; err != nil {
		return fmt.Errorf("failed to update service: %w", err)
	}

	return nil
}

// Activate activates a service
func (r *ServiceRepository) Activate(ctx context.Context, tenantID uuid.UUID, serviceID int) error {
	result := r.db.WithContext(ctx).
		Model(&models.Service{}).
		Where("tenant_id = ? AND id = ?", tenantID, serviceID).
		Update("is_active", true)

	if result.Error != nil {
		return fmt.Errorf("failed to activate service: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return models.ErrServiceNotFound
	}

	return nil
}

// Deactivate deactivates a service
func (r *ServiceRepository) Deactivate(ctx context.Context, tenantID uuid.UUID, serviceID int) error {
	result := r.db.WithContext(ctx).
		Model(&models.Service{}).
		Where("tenant_id = ? AND id = ?", tenantID, serviceID).
		Update("is_active", false)

	if result.Error != nil {
		return fmt.Errorf("failed to deactivate service: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return models.ErrServiceNotFound
	}

	return nil
}

// Delete deletes a service (soft delete by setting is_active to false)
func (r *ServiceRepository) Delete(ctx context.Context, tenantID uuid.UUID, serviceID int) error {
	// We use soft delete by setting is_active to false
	return r.Deactivate(ctx, tenantID, serviceID)
}

// InsertDefaultServices inserts the three default services for a tenant
func (r *ServiceRepository) InsertDefaultServices(ctx context.Context, tenantID uuid.UUID) error {
	// Define the three default services
	defaultServices := []*models.Service{
		{
			TenantID:    tenantID,
			Name:        "خدمة بوابة الهدهد",
			Code:        models.ServiceCodeHududGateway,
			Type:        models.ServiceTypeGateway,
			Description: "إرسال الرسائل عبر بوابة الهدهد بنموذج الدفع حسب الاستخدام",
			IsActive:    true,
		},
		{
			TenantID:    tenantID,
			Name:        "خدمة الإرسال عبر الهاتف",
			Code:        models.ServiceCodeMobileGateway,
			Type:        models.ServiceTypeMobile,
			Description: "إرسال الرسائل عبر الهاتف بنموذج الاشتراك الشهري",
			IsActive:    true,
		},
		{
			TenantID:    tenantID,
			Name:        "خدمة إدارة البوابات",
			Code:        models.ServiceCodeDedicatedGateway,
			Type:        models.ServiceTypeDedicated,
			Description: "بوابات مخصصة للشركات المؤسسية",
			IsActive:    true,
		},
	}

	// Insert services in a transaction
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, service := range defaultServices {
			// Check if service already exists
			var count int64
			if err := tx.Model(&models.Service{}).
				Where("tenant_id = ? AND code = ?", tenantID, service.Code).
				Count(&count).Error; err != nil {
				return fmt.Errorf("failed to check service existence: %w", err)
			}

			// Skip if already exists
			if count > 0 {
				continue
			}

			// Create service
			if err := tx.Create(service).Error; err != nil {
				return fmt.Errorf("failed to create service %s: %w", service.Code, err)
			}
		}

		return nil
	})
}
