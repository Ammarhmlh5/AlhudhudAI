package models

import (
	"time"

	"github.com/google/uuid"
)

type SystemStatus string

const (
	SystemActive  SystemStatus = "ACTIVE"  // أرسل رسالة خلال آخر دقيقة
	SystemStandby SystemStatus = "STANDBY" // لم يرسل رسائل لكن قد يكون متصل
	SystemOffline SystemStatus = "OFFLINE" // غير متصل لفترة طويلة
)

// AccountingSystem tracks the connection status of accounting systems
type AccountingSystem struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	TenantID uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"tenant_id"`
	Tenant   Tenant    `gorm:"foreignKey:TenantID;constraint:OnDelete:CASCADE" json:"-"`

	// System information
	SystemName    string  `json:"system_name"`
	SystemVersion *string `json:"system_version,omitempty"`

	// Connection status tracking
	Status       SystemStatus `gorm:"default:'STANDBY';index" json:"status"`
	LastActivity time.Time    `gorm:"index;not null;default:CURRENT_TIMESTAMP" json:"last_activity"`

	// Statistics
	TotalMessages int64      `gorm:"default:0" json:"total_messages"`
	LastMessageAt *time.Time `json:"last_message_at,omitempty"`

	// Timestamps
	FirstConnectedAt time.Time `json:"first_connected_at"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
