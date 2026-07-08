package alerts

import (
	"time"

	"github.com/google/uuid"
)

// AlertType represents the type of alert
type AlertType string

const (
	AlertTypeRatingChange        AlertType = "rating_change"
	AlertTypeServerFailure       AlertType = "server_failure"
	AlertTypeThresholdViolation  AlertType = "threshold_violation"
	AlertTypeDeviceFailure       AlertType = "device_failure"
	AlertTypeTimingViolation     AlertType = "timing_violation"
	AlertTypeEnforcementAction   AlertType = "enforcement_action"
	
	// System-wide alerts integrated from transient alerting system
	AlertTypeQueueBackup    AlertType = "queue_backup"
	AlertTypeHighErrorRate  AlertType = "high_error_rate"
	AlertTypeSystemResource AlertType = "system_resource"
	AlertTypeDatabaseFailure AlertType = "database_failure"
	AlertTypeLowBalance     AlertType = "low_balance"
	AlertTypeSystemHealth   AlertType = "system_health"
	AlertTypeChannelOffline AlertType = "channel_offline"
)

// AlertSeverity represents the severity level of an alert
type AlertSeverity string

const (
	AlertSeverityCritical AlertSeverity = "critical"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityInfo     AlertSeverity = "info"
)

// AlertStatus represents the current status of an alert
type AlertStatus string

const (
	AlertStatusActive       AlertStatus = "active"
	AlertStatusAcknowledged AlertStatus = "acknowledged"
	AlertStatusResolved     AlertStatus = "resolved"
)

// Alert represents an alert in the system
type Alert struct {
	ID              int64                  `json:"id" db:"id"`
	TenantID        *uuid.UUID             `json:"tenant_id,omitempty" db:"tenant_id"`
	AlertType       AlertType              `json:"alert_type" db:"alert_type"`
	Severity        AlertSeverity          `json:"severity" db:"severity"`
	Title           string                 `json:"title" db:"title"`
	Message         string                 `json:"message" db:"message"`
	Context         map[string]interface{} `json:"context,omitempty" db:"context"`
	Status          AlertStatus            `json:"status" db:"status"`
	AcknowledgedAt  *time.Time             `json:"acknowledged_at,omitempty" db:"acknowledged_at"`
	AcknowledgedBy  *uuid.UUID             `json:"acknowledged_by,omitempty" db:"acknowledged_by"`
	ResolvedAt      *time.Time             `json:"resolved_at,omitempty" db:"resolved_at"`
	DedupKey        string                 `json:"dedup_key,omitempty" db:"dedup_key"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
}

// CreateAlertRequest represents a request to create a new alert
type CreateAlertRequest struct {
	TenantID  *uuid.UUID             `json:"tenant_id,omitempty"`
	AlertType AlertType              `json:"alert_type"`
	Severity  AlertSeverity          `json:"severity"`
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Context   map[string]interface{} `json:"context,omitempty"`
	DedupKey  string                 `json:"dedup_key,omitempty"`
}

// AlertFilter represents filter criteria for querying alerts
type AlertFilter struct {
	TenantID  *uuid.UUID      `json:"tenant_id,omitempty"`
	AlertType *AlertType      `json:"alert_type,omitempty"`
	Severity  *AlertSeverity  `json:"severity,omitempty"`
	Status    *AlertStatus    `json:"status,omitempty"`
	StartTime *time.Time      `json:"start_time,omitempty"`
	EndTime   *time.Time      `json:"end_time,omitempty"`
	Limit     int             `json:"limit,omitempty"`
	Offset    int             `json:"offset,omitempty"`
}

// AlertListResponse represents a paginated list of alerts
type AlertListResponse struct {
	Alerts     []*Alert `json:"alerts"`
	TotalCount int64    `json:"total_count"`
	Limit      int      `json:"limit"`
	Offset     int      `json:"offset"`
}
