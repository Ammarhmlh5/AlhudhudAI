package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Contact struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	TenantID    uuid.UUID      `gorm:"type:uuid;index" json:"tenant_id"`
	Name        string         `gorm:"not null" json:"name"`
	PhoneNumber string         `gorm:"not null" json:"phone_number"`
	Group       string         `json:"group"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}
