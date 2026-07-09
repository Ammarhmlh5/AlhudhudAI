package status

import (
	"time"

	"github.com/google/uuid"
)

// StatusReport represents a status report from a mobile device
// It contains information about the delivery status of a message
type StatusReport struct {
	MessageID  string    `json:"message_id" validate:"required"`
	TenantID   uuid.UUID `json:"tenant_id" validate:"required"`
	ChannelID  uuid.UUID `json:"channel_id" validate:"required"`
	Status     string    `json:"status" validate:"required,oneof=BUFFERED SENT DELIVERED FAILED"`
	Timestamp  time.Time `json:"timestamp" validate:"required"`
	FailReason string    `json:"fail_reason,omitempty"`
}

// MessageStatus represents the possible states of a message
const (
	StatusBuffered  = "BUFFERED"
	StatusSent      = "SENT"
	StatusDelivered = "DELIVERED"
	StatusFailed    = "FAILED"
)

// ProcessResult represents the result of processing a status report
type ProcessResult struct {
	Success       bool
	AlreadyExists bool
	Error         error
}
