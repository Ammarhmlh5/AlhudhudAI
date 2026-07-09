package models

import (
	"time"

	"github.com/google/uuid"
)

// GlobalIncomingEvent handles Phase 1: Raw Ingestion of Bulk Messages
type GlobalIncomingEvent struct {
	ID         int64      `gorm:"primaryKey;autoIncrement" json:"id"` // Internal counter for admin stats only
	CustomID   string     `gorm:"type:varchar(20);uniqueIndex" json:"custom_id"` // Redis-generated Event ID (EV...)
	TenantID   *uuid.UUID `gorm:"type:uuid;index" json:"tenant_id,omitempty"`
	SenderIP   string     `gorm:"type:inet" json:"sender_ip"`
	RawPayload string     `gorm:"type:jsonb;not null" json:"raw_payload"`
	Source     string     `gorm:"index" json:"source"` // Origin: WEB_UI, WEB_API, ACCOUNTING, MOBILE
	Status     string     `gorm:"default:'RECEIVED';index" json:"status"`
	
	// Progress tracking (Requirement Phase 7)
	TotalItems           int        `gorm:"default:0" json:"total_items"`
	ProcessedItems       int        `gorm:"default:0" json:"processed_items"`
	FailedItems          int        `gorm:"default:0" json:"failed_items"`
	ProcessingStartedAt  *time.Time `json:"processing_started_at,omitempty"`
	ProcessingFinishedAt *time.Time `json:"processing_finished_at,omitempty"`

	CreatedAt  time.Time  `gorm:"index" json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func (GlobalIncomingEvent) TableName() string {
	return "global_incoming_events"
}
