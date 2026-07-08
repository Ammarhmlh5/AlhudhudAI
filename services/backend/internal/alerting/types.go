package alerting

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// AlertLevel represents the severity of an alert
type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "INFO"
	AlertLevelWarning  AlertLevel = "WARNING"
	AlertLevelCritical AlertLevel = "CRITICAL"
)

// AlertType represents the specific type of system alert
type AlertType string

const (
	AlertTypeQueueBackup    AlertType = "QUEUE_BACKUP"
	AlertTypeHighErrorRate  AlertType = "HIGH_ERROR_RATE"
	AlertTypeSystemResource AlertType = "SYSTEM_RESOURCE"
	AlertTypeDatabaseFailure AlertType = "DATABASE_FAILURE"
	AlertTypeLowBalance     AlertType = "LOW_BALANCE"
	AlertTypeSystemHealth   AlertType = "SYSTEM_HEALTH"
	AlertTypeChannelOffline AlertType = "CHANNEL_OFFLINE"
)

// Alert represents a system alert
type Alert struct {
	Level     AlertLevel             `json:"level"`
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
	Type      AlertType              `json:"type"`
	Source    string                 `json:"source"`
	TenantID  *uuid.UUID             `json:"tenant_id,omitempty"`
}

// NewAlert creates a new alert instance
func NewAlert(alertType AlertType, level AlertLevel, title, message, source string) Alert {
	return Alert{
		Level:     level,
		Type:      alertType,
		Title:     title,
		Message:   message,
		Source:    source,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
}

// WithTenantID adds tenant ID to alert
func (a Alert) WithTenantID(tenantID uuid.UUID) Alert {
	a.TenantID = &tenantID
	if a.Metadata == nil {
		a.Metadata = make(map[string]interface{})
	}
	a.Metadata["tenant_id"] = tenantID.String()
	return a
}

// WithMetadata adds custom metadata to the alert
func (a Alert) WithMetadata(key string, value interface{}) Alert {
	if a.Metadata == nil {
		a.Metadata = make(map[string]interface{})
	}
	a.Metadata[key] = value
	return a
}

// AlertChannel defines the interface for sending alerts
type AlertChannel interface {
	Send(alert Alert) error
	Name() string
	IsEnabled() bool
	SupportsLevel(level AlertLevel) bool
}

// SyncContext is a helper for synchronous sending if needed, though manager handles it
type SyncContext interface {
	context.Context
}
