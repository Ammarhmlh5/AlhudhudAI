package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PendingRegistration stores temporary registration data until all steps are completed
type PendingRegistration struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Email        string    `gorm:"uniqueIndex;not null" json:"email"`
	PhoneNumber  *string   `gorm:"uniqueIndex" json:"phone_number"`
	PasswordHash string    `gorm:"not null" json:"-"`

	// Step 2 data (Company Information)
	BusinessName string `json:"business_name"`
	ContactName  string `json:"contact_name"`
	Address      string `json:"address"`
	Sector       string `json:"sector"`
	CountryCode  string `json:"country_code"`

	// Status tracking
	Step       int  `gorm:"default:1" json:"step"` // 1=email/pass, 2=OTP verified, 3=company info
	IsVerified bool `gorm:"default:false" json:"is_verified"`

	// Timestamps
	CreatedAt time.Time      `json:"created_at"`
	ExpiresAt time.Time      `json:"expires_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
