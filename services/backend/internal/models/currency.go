package models

import (
	"time"
)

// Currency represents a supported currency in the system
type Currency struct {
	ID            int       `gorm:"primaryKey" json:"id"`
	Code          string    `gorm:"type:varchar(3);uniqueIndex;not null" json:"code"`
	Name          string    `gorm:"type:varchar(100);not null" json:"name"`
	Symbol        string    `gorm:"type:varchar(10)" json:"symbol"`
	DecimalPlaces int       `gorm:"default:2" json:"decimal_places"`
	IsActive      bool      `gorm:"default:true" json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// TableName specifies the table name for Currency model
func (Currency) TableName() string {
	return "currencies"
}

// ExchangeRate represents currency exchange rates with historical tracking
type ExchangeRate struct {
	ID            int       `gorm:"primaryKey" json:"id"`
	FromCurrency  string    `gorm:"type:varchar(3);not null;index:idx_exchange_rates_from" json:"from_currency"`
	ToCurrency    string    `gorm:"type:varchar(3);not null;index:idx_exchange_rates_to" json:"to_currency"`
	Rate          float64   `gorm:"type:decimal(15,6);not null" json:"rate"`
	EffectiveFrom time.Time `gorm:"not null;default:CURRENT_TIMESTAMP;index:idx_exchange_rates_effective" json:"effective_from"`
	EffectiveTo   *time.Time `gorm:"index:idx_exchange_rates_effective" json:"effective_to,omitempty"`
	Source        string    `gorm:"type:varchar(100)" json:"source"`
	Notes         string    `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// TableName specifies the table name for ExchangeRate model
func (ExchangeRate) TableName() string {
	return "exchange_rates"
}

// IsActive checks if the exchange rate is currently active
func (er *ExchangeRate) IsActive() bool {
	now := time.Now()
	if er.EffectiveTo == nil {
		return now.After(er.EffectiveFrom) || now.Equal(er.EffectiveFrom)
	}
	return (now.After(er.EffectiveFrom) || now.Equal(er.EffectiveFrom)) && now.Before(*er.EffectiveTo)
}
