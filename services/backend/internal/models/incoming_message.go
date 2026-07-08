package models

import (
	"time"

	"github.com/google/uuid"
)

// IncomingMessage represents raw messages received from accounting systems
// Messages are stored as-is without any modification for fast ingestion
type IncomingMessage struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	TenantID uuid.UUID `gorm:"type:uuid;index;not null" json:"tenant_id"`
	Tenant   Tenant    `gorm:"foreignKey:TenantID;constraint:OnDelete:CASCADE" json:"-"`

	// Raw message data as received from accounting system
	RawPayload string `gorm:"type:jsonb;not null" json:"raw_payload"`

	// Sender information (Name of the project or accounting system)
	SenderName string `gorm:"index" json:"sender_name"`

	// Message metadata
	PhoneNumber    string `gorm:"index" json:"phone_number"`
	MessageContent string `gorm:"type:text" json:"message_content"`

	// Tracking fields
	ReceivedAt  time.Time  `gorm:"index;not null;default:CURRENT_TIMESTAMP" json:"received_at"`
	ProcessedAt *time.Time `gorm:"index" json:"processed_at"`
	IsProcessed bool       `gorm:"default:false;index" json:"is_processed"`

	// Error handling
	ProcessingError *string `gorm:"type:text" json:"processing_error,omitempty"`
	RetryCount      int     `gorm:"default:0" json:"retry_count"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
