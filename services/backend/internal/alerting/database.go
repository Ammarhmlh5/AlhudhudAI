package alerting

import (
	"context"
	"fmt"
	"strings"

	adminAlerts "github.com/al-hudhud/backend/internal/admin/alerts"
)

// DatabaseChannel persists alerts to the database using the admin alerts manager
type DatabaseChannel struct {
	adminManager *adminAlerts.Manager
	enabled      bool
}

// NewDatabaseChannel creates a new database alert channel
func NewDatabaseChannel(adminManager *adminAlerts.Manager) *DatabaseChannel {
	return &DatabaseChannel{
		adminManager: adminManager,
		enabled:      true,
	}
}

// Name returns the channel name
func (dc *DatabaseChannel) Name() string {
	return "DatabaseChannel"
}

// IsEnabled returns whether the channel is enabled
func (dc *DatabaseChannel) IsEnabled() bool {
	return dc.enabled && dc.adminManager != nil
}

// SupportsLevel returns whether the channel supports the alert level
func (dc *DatabaseChannel) SupportsLevel(level AlertLevel) bool {
	// Persist all levels for audit purposes
	return true
}

// Send persists an alert to the database
func (dc *DatabaseChannel) Send(alert Alert) error {
	if !dc.IsEnabled() {
		return nil
	}

	// Map Alert to CreateAlertRequest
	req := &adminAlerts.CreateAlertRequest{
		TenantID:  alert.TenantID,
		Title:     alert.Title,
		Message:   alert.Message,
		Severity:  mapSeverity(alert.Level),
		AlertType: mapType(alert.Type),
		Context:   alert.Metadata,
		DedupKey:  fmt.Sprintf("%s:%s", string(alert.Type), alert.Source),
	}

	// Persist alert
	_, err := dc.adminManager.CreateAlert(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to persist alert: %w", err)
	}

	return nil
}

// mapSeverity maps alerting.AlertLevel to adminAlerts.AlertSeverity
func mapSeverity(level AlertLevel) adminAlerts.AlertSeverity {
	switch level {
	case AlertLevelCritical:
		return adminAlerts.AlertSeverityCritical
	case AlertLevelWarning:
		return adminAlerts.AlertSeverityWarning
	case AlertLevelInfo:
		return adminAlerts.AlertSeverityInfo
	default:
		return adminAlerts.AlertSeverityInfo
	}
}

// mapType maps alerting.AlertType to adminAlerts.AlertType
func mapType(at AlertType) adminAlerts.AlertType {
	// The strings in adminAlerts.AlertType are lowercase (e.g., "low_balance")
	// while in alerting.AlertType they are uppercase (e.g., "LOW_BALANCE").
	return adminAlerts.AlertType(strings.ToLower(string(at)))
}
