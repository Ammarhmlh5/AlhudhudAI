package models

import (
	"time"

	"github.com/google/uuid"
)

// Tariff represents a tariff in the pricing system
// Supports dynamic tariffs based on network operator, country, and date ranges
// with full tenant isolation using Row Level Security
type Tariff struct {
	ID              int       `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID        uuid.UUID `gorm:"type:uuid;not null;index:idx_tariffs_tenant_id" json:"tenant_id"`
	ServiceID       int       `gorm:"not null;index:idx_tariffs_tenant_service,priority:2" json:"service_id"`
	CountryCode     string    `gorm:"type:varchar(2);not null;index:idx_tariffs_tenant_country,priority:2" json:"country_code"`
	NetworkOperator string    `gorm:"type:varchar(100);index:idx_tariffs_tenant_operator,priority:2" json:"network_operator"`
	MessageType     string    `gorm:"type:varchar(50);default:'standard'" json:"message_type"` // 'standard', 'unicode'
	CostPerMessage  float64   `gorm:"type:decimal(10,4);not null" json:"cost_per_message"`
	Currency        string    `gorm:"type:varchar(3);default:'YER'" json:"currency"` // 'YER', 'USD', 'SAR'
	EffectiveFrom   time.Time `gorm:"type:date;not null;index:idx_tariffs_tenant_effective,priority:2" json:"effective_from"`
	EffectiveTo     *time.Time `gorm:"type:date;index:idx_tariffs_tenant_effective,priority:3" json:"effective_to"`
	IsActive        bool      `gorm:"default:true;index:idx_tariffs_tenant_active,priority:2" json:"is_active"`
	CreatedAt       time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt       time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`

	// Relationships
	Tenant  *Tenant  `gorm:"foreignKey:TenantID;constraint:OnDelete:CASCADE" json:"tenant,omitempty"`
	Service *Service `gorm:"foreignKey:ServiceID;constraint:OnDelete:CASCADE" json:"service,omitempty"`
}

// TableName specifies the table name for the Tariff model
func (Tariff) TableName() string {
	return "tariffs"
}

// Validate validates the tariff data
func (t *Tariff) Validate() error {
	if t.TenantID == uuid.Nil {
		return ErrInvalidTenantID
	}

	if t.ServiceID == 0 {
		return &ValidationError{Field: "service_id", Message: "service_id is required"}
	}

	if t.CountryCode == "" {
		return &ValidationError{Field: "country_code", Message: "country_code is required"}
	}

	if t.CostPerMessage <= 0 {
		return &ValidationError{Field: "cost_per_message", Message: "cost_per_message must be positive"}
	}

	if t.Currency == "" {
		return &ValidationError{Field: "currency", Message: "currency is required"}
	}

	// Validate effective dates
	if t.EffectiveTo != nil && t.EffectiveTo.Before(t.EffectiveFrom) {
		return &ValidationError{Field: "effective_to", Message: "effective_to must be after effective_from"}
	}

	return nil
}

// IsEffectiveOn checks if the tariff is effective on a given date
func (t *Tariff) IsEffectiveOn(date time.Time) bool {
	if !t.IsActive {
		return false
	}

	if date.Before(t.EffectiveFrom) {
		return false
	}

	if t.EffectiveTo != nil && date.After(*t.EffectiveTo) {
		return false
	}

	return true
}

// IsCurrentlyEffective checks if the tariff is currently effective
func (t *Tariff) IsCurrentlyEffective() bool {
	return t.IsEffectiveOn(time.Now())
}
