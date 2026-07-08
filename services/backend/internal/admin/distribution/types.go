package distribution

import (
	"time"

	"github.com/google/uuid"
)

// DistributionStrategy defines the interface for message distribution strategies
type DistributionStrategy interface {
	// SelectDevice selects the next device for message distribution
	SelectDevice() (*Device, error)
}

// Device represents a phone device in the distribution pool
type Device struct {
	ID                      uuid.UUID `json:"id"`
	TenantID                uuid.UUID `json:"tenant_id"`
	DeviceName              string    `json:"device_name"`
	DeviceModel             string    `json:"device_model"`
	OSVersion               string    `json:"os_version"`
	AppVersion              string    `json:"app_version"`
	PriorityLevel           int       `json:"priority_level"`
	MaxConcurrentMessages   int       `json:"max_concurrent_messages"`
	Status                  string    `json:"status"` // 'online', 'offline', 'failed', 'excluded'
	LastSeen                time.Time `json:"last_seen"`
	LastFailure             *time.Time `json:"last_failure,omitempty"`
	ExcludedUntil           *time.Time `json:"excluded_until,omitempty"`
	TotalMessagesSent       int64     `json:"total_messages_sent"`
	SuccessfulMessages      int64     `json:"successful_messages"`
	FailedMessages          int64     `json:"failed_messages"`
	CurrentLoad             int       `json:"current_load"`
	AverageProcessingTimeMs int       `json:"average_processing_time_ms"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

// DevicePool represents a pool of available devices for distribution
type DevicePool struct {
	TenantID uuid.UUID `json:"tenant_id"`
	Devices  []*Device `json:"devices"`
}

// DistributionMetrics tracks distribution statistics per device
type DistributionMetrics struct {
	ID                      int64     `json:"id"`
	TenantID                uuid.UUID `json:"tenant_id"`
	DeviceID                uuid.UUID `json:"device_id"`
	PeriodStart             time.Time `json:"period_start"`
	PeriodEnd               time.Time `json:"period_end"`
	MessagesAssigned        int64     `json:"messages_assigned"`
	MessagesSent            int64     `json:"messages_sent"`
	MessagesFailed          int64     `json:"messages_failed"`
	AverageProcessingTimeMs int       `json:"average_processing_time_ms"`
	MinProcessingTimeMs     int       `json:"min_processing_time_ms"`
	MaxProcessingTimeMs     int       `json:"max_processing_time_ms"`
	CreatedAt               time.Time `json:"created_at"`
}

// StrategyType represents the type of distribution strategy
type StrategyType string

const (
	StrategyRoundRobin   StrategyType = "round_robin"
	StrategyLoadBalancing StrategyType = "load_balancing"
	StrategyPriority     StrategyType = "priority"
	StrategyCustom       StrategyType = "custom"
)

// DistributionConfig represents the distribution strategy configuration for a tenant
type DistributionConfig struct {
	ID             int64        `json:"id"`
	TenantID       uuid.UUID    `json:"tenant_id"`
	StrategyType   StrategyType `json:"strategy_type"`
	StrategyConfig map[string]interface{} `json:"strategy_config,omitempty"`
	CustomRules    map[string]interface{} `json:"custom_rules,omitempty"`
	IsActive       bool         `json:"is_active"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}

// DeviceStatus constants
const (
	DeviceStatusOnline   = "online"
	DeviceStatusOffline  = "offline"
	DeviceStatusFailed   = "failed"
	DeviceStatusExcluded = "excluded"
)
