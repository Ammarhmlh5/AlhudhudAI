package tariff

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/al-hudhud/backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository handles database operations for tariffs
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new tariff repository
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create creates a new tariff
func (r *Repository) Create(ctx context.Context, tariff *models.Tariff) error {
	// Validate tariff
	if err := tariff.Validate(); err != nil {
		return err
	}

	// Create tariff
	if err := r.db.WithContext(ctx).Create(tariff).Error; err != nil {
		return fmt.Errorf("failed to create tariff: %w", err)
	}

	return nil
}

// GetByID retrieves a tariff by ID and tenant ID
func (r *Repository) GetByID(ctx context.Context, tenantID uuid.UUID, tariffID int) (*models.Tariff, error) {
	var tariff models.Tariff

	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, tariffID).
		First(&tariff).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("tariff not found")
		}
		return nil, fmt.Errorf("failed to get tariff: %w", err)
	}

	return &tariff, nil
}

// GetActiveTariff retrieves the active tariff for a specific service/country/operator combination
func (r *Repository) GetActiveTariff(
	ctx context.Context,
	tenantID uuid.UUID,
	serviceID int,
	countryCode string,
	networkOperator string,
	messageType string,
	date time.Time,
) (*models.Tariff, error) {
	var tariff models.Tariff

	// Query for active tariff
	// Prefer specific operator over 'Other'
	query := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Where("service_id = ?", serviceID).
		Where("country_code = ?", countryCode).
		Where("(network_operator = ? OR network_operator = ?)", networkOperator, "Other").
		Where("message_type = ?", messageType).
		Where("is_active = ?", true).
		Where("effective_from <= ?", date).
		Where("effective_to IS NULL OR effective_to >= ?", date).
		Order(fmt.Sprintf("CASE WHEN network_operator = '%s' THEN 0 ELSE 1 END", networkOperator)).
		Order("effective_from DESC")

	if err := query.First(&tariff).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("no active tariff found for service %d, country %s, operator %s", 
				serviceID, countryCode, networkOperator)
		}
		return nil, fmt.Errorf("failed to get active tariff: %w", err)
	}

	return &tariff, nil
}

// GetByService retrieves all tariffs for a specific service
func (r *Repository) GetByService(ctx context.Context, tenantID uuid.UUID, serviceID int) ([]*models.Tariff, error) {
	var tariffs []*models.Tariff

	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND service_id = ?", tenantID, serviceID).
		Order("country_code ASC, network_operator ASC, effective_from DESC").
		Find(&tariffs).Error; err != nil {
		return nil, fmt.Errorf("failed to get tariffs by service: %w", err)
	}

	return tariffs, nil
}

// GetActive retrieves all active tariffs for a tenant
func (r *Repository) GetActive(ctx context.Context, tenantID uuid.UUID) ([]*models.Tariff, error) {
	var tariffs []*models.Tariff

	now := time.Now()

	if err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Where("is_active = ?", true).
		Where("effective_from <= ?", now).
		Where("effective_to IS NULL OR effective_to >= ?", now).
		Order("service_id ASC, country_code ASC, network_operator ASC").
		Find(&tariffs).Error; err != nil {
		return nil, fmt.Errorf("failed to get active tariffs: %w", err)
	}

	return tariffs, nil
}

// Update updates a tariff
func (r *Repository) Update(ctx context.Context, tariff *models.Tariff) error {
	// Validate tariff
	if err := tariff.Validate(); err != nil {
		return err
	}

	// Check if tariff exists
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.Tariff{}).
		Where("tenant_id = ? AND id = ?", tariff.TenantID, tariff.ID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check tariff existence: %w", err)
	}

	if count == 0 {
		return fmt.Errorf("tariff not found")
	}

	// Update tariff
	if err := r.db.WithContext(ctx).
		Model(&models.Tariff{}).
		Where("tenant_id = ? AND id = ?", tariff.TenantID, tariff.ID).
		Updates(map[string]interface{}{
			"cost_per_message": tariff.CostPerMessage,
			"currency":         tariff.Currency,
			"effective_from":   tariff.EffectiveFrom,
			"effective_to":     tariff.EffectiveTo,
			"is_active":        tariff.IsActive,
			"updated_at":       time.Now(),
		}).Error; err != nil {
		return fmt.Errorf("failed to update tariff: %w", err)
	}

	return nil
}

// Deactivate deactivates a tariff
func (r *Repository) Deactivate(ctx context.Context, tenantID uuid.UUID, tariffID int) error {
	result := r.db.WithContext(ctx).
		Model(&models.Tariff{}).
		Where("tenant_id = ? AND id = ?", tenantID, tariffID).
		Update("is_active", false)

	if result.Error != nil {
		return fmt.Errorf("failed to deactivate tariff: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("tariff not found")
	}

	return nil
}

// InsertDefaultTariffs inserts the default Hudud Gateway tariffs for a tenant
func (r *Repository) InsertDefaultTariffs(ctx context.Context, tenantID uuid.UUID) error {
	// Get the Hudud Gateway service for this tenant
	var service models.Service
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND code = ?", tenantID, models.ServiceCodeHududGateway).
		First(&service).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("Hudud Gateway service not found for tenant %s", tenantID)
		}
		return fmt.Errorf("failed to get Hudud Gateway service: %w", err)
	}

	// Define default tariffs
	defaultTariffs := []*models.Tariff{
		{
			TenantID:        tenantID,
			ServiceID:       service.ID,
			CountryCode:     "YE",
			NetworkOperator: "Yemen Mobile",
			MessageType:     "standard",
			CostPerMessage:  9.0000,
			Currency:        "YER",
			EffectiveFrom:   time.Now(),
			EffectiveTo:     nil,
			IsActive:        true,
		},
		{
			TenantID:        tenantID,
			ServiceID:       service.ID,
			CountryCode:     "YE",
			NetworkOperator: "Other",
			MessageType:     "standard",
			CostPerMessage:  13.0000,
			Currency:        "YER",
			EffectiveFrom:   time.Now(),
			EffectiveTo:     nil,
			IsActive:        true,
		},
	}

	// Insert tariffs in a transaction
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, tariff := range defaultTariffs {
			// Check if tariff already exists
			var count int64
			if err := tx.Model(&models.Tariff{}).
				Where("tenant_id = ? AND service_id = ? AND country_code = ? AND network_operator = ?",
					tenantID, tariff.ServiceID, tariff.CountryCode, tariff.NetworkOperator).
				Count(&count).Error; err != nil {
				return fmt.Errorf("failed to check tariff existence: %w", err)
			}

			// Skip if already exists
			if count > 0 {
				continue
			}

			// Create tariff
			if err := tx.Create(tariff).Error; err != nil {
				return fmt.Errorf("failed to create tariff for %s: %w", tariff.NetworkOperator, err)
			}
		}

		return nil
	})
}
