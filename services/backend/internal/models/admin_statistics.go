package models

import "time"

// AdminStatistics is a single-row table that aggregates system-wide metrics
// for the Super Admin dashboard. It is refreshed by a background job, so
// the dashboard reads it in O(1) instead of expensive full-table COUNTs.
type AdminStatistics struct {
	ID int `gorm:"primaryKey;autoIncrement:false" json:"id"`

	// Tenant metrics
	TotalTenants  int `gorm:"default:0" json:"total_tenants"`
	ActiveTenants int `gorm:"default:0" json:"active_tenants"`

	// Device metrics
	TotalDevices  int `gorm:"default:0" json:"total_devices"`
	ActiveDevices int `gorm:"default:0" json:"active_devices"`

	// Message metrics
	PendingMessages      int `gorm:"default:0" json:"pending_messages"`
	QueuedMessages       int `gorm:"default:0" json:"queued_messages"`
	ProcessingMessages   int `gorm:"default:0" json:"processing_messages"`
	TotalMessagesToday   int `gorm:"default:0" json:"total_messages_today"`
	SentMessagesToday    int `gorm:"default:0" json:"sent_messages_today"`
	FailedMessagesToday  int `gorm:"default:0" json:"failed_messages_today"`

	// Balance aggregates
	TotalBalance float64 `gorm:"type:decimal(14,4);default:0" json:"total_balance"`

	// Raw ingestion metrics
	TotalEventsToday  int `gorm:"default:0" json:"total_events_today"`
	ParsedEventsToday int `gorm:"default:0" json:"parsed_events_today"`
	FailedEventsToday int `gorm:"default:0" json:"failed_events_today"`

	LastUpdated time.Time `gorm:"default:NOW()" json:"last_updated"`
}

func (AdminStatistics) TableName() string {
	return "admin_statistics"
}
