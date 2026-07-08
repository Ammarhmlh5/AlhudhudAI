package models

import "time"

type Package struct {
	ID               int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID         string    `gorm:"type:uuid;index" json:"tenant_id"`
	Name             string    `gorm:"type:varchar(255)" json:"name"`
	Code             string    `gorm:"type:varchar(50);uniqueIndex" json:"code"`
	PriceUSD         float64   `gorm:"type:decimal(12,4)" json:"price_usd"`
	IsActive         bool      `gorm:"default:true" json:"is_active"`
	IncludedMessages int64     `json:"included_messages"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
