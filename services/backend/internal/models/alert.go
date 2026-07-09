package models

import (
	"time"

	"github.com/google/uuid"
)

type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "INFO"
	SeverityWarning  AlertSeverity = "WARNING"
	SeverityCritical AlertSeverity = "CRITICAL"
)

type SystemAlert struct {
	ID        uuid.UUID     `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	TenantID  uuid.UUID     `gorm:"type:uuid;index" json:"tenant_id"`
	Severity  AlertSeverity `gorm:"type:varchar(20);default:'INFO'" json:"severity"`
	Title     string        `gorm:"not null" json:"title"`
	Message   string        `gorm:"type:text" json:"message"`
	IsRead    bool          `gorm:"default:false" json:"is_read"`
	RawData   string        `gorm:"type:jsonb" json:"raw_data,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
}
