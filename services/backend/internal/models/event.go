package models

import (
	"time"

	"github.com/google/uuid"
)

type EventStatus string

const (
	EventReceived   EventStatus = "RECEIVED"
	EventProcessing EventStatus = "PROCESSING"
	EventCompleted  EventStatus = "COMPLETED"
	EventFailed     EventStatus = "FAILED"
)

// MessageEvent represents a bulk ingestion batch or a significant system event.
// It allows the UI to show progress for a group of messages instead of just individual rows.
type MessageEvent struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	TenantID  uuid.UUID `gorm:"type:uuid;index" json:"tenant_id"`
	
	// Metadata
	Name        string      `json:"name"`        // e.g. "Invoice Batch March 2026"
	Source      string      `json:"source"`      // e.g. "API", "EXCEL", "SYSTEM"
	Status      EventStatus `gorm:"default:'RECEIVED';index" json:"status"`
	MessageType string      `json:"message_type"` // e.g. "INVOICE", "OTP"
	
	// Stats
	TotalItems     int `gorm:"default:0" json:"total_items"`
	ProcessedItems int `gorm:"default:0" json:"processed_items"`
	FailedItems    int `gorm:"default:0" json:"failed_items"`
	
	// Timestamps
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func (MessageEvent) TableName() string {
	return "message_events"
}
