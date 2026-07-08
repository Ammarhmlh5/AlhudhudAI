package health

import (
	"time"

	"github.com/google/uuid"
)

// HealthStatus represents the health status of a server
type HealthStatus string

const (
	// Healthy: response_time < 200ms AND error_rate < 1%
	Healthy HealthStatus = "healthy"
	// Degraded: response_time 200-500ms OR error_rate 1-5%
	Degraded HealthStatus = "degraded"
	// Unhealthy: response_time > 500ms OR error_rate > 5%
	Unhealthy HealthStatus = "unhealthy"
)

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	ServerID       uuid.UUID    `json:"server_id" db:"server_id"`
	TenantID       uuid.UUID    `json:"tenant_id" db:"tenant_id"`
	CheckTimestamp time.Time    `json:"check_timestamp" db:"check_timestamp"`
	ResponseTimeMs int          `json:"response_time_ms" db:"response_time_ms"`
	StatusCode     int          `json:"status_code" db:"status_code"`
	IsAvailable    bool         `json:"is_available" db:"is_available"`
	ErrorMessage   string       `json:"error_message,omitempty" db:"error_message"`
	ErrorCount     int          `json:"error_count" db:"error_count"`
	ErrorRate      float64      `json:"error_rate" db:"error_rate"`
	HealthStatus   HealthStatus `json:"health_status" db:"health_status"`
}

// ServerInfo represents basic server information
type ServerInfo struct {
	ID               uuid.UUID `json:"id" db:"id"`
	TenantID         uuid.UUID `json:"tenant_id" db:"tenant_id"`
	ServerURL        string    `json:"server_url" db:"server_url"`
	HealthEndpoint   string    `json:"health_endpoint" db:"health_endpoint"`
	IsActive         bool      `json:"is_active" db:"is_active"`
}

// HealthHistoryFilter represents filters for querying health history
type HealthHistoryFilter struct {
	ServerID  uuid.UUID     `json:"server_id"`
	TenantID  *uuid.UUID    `json:"tenant_id,omitempty"`
	StartTime *time.Time    `json:"start_time,omitempty"`
	EndTime   *time.Time    `json:"end_time,omitempty"`
	Status    *HealthStatus `json:"status,omitempty"`
	Limit     int           `json:"limit"`
	Offset    int           `json:"offset"`
}

// HealthSummary represents a summary of server health
type HealthSummary struct {
	ServerID           uuid.UUID    `json:"server_id"`
	TenantID           uuid.UUID    `json:"tenant_id"`
	CurrentStatus      HealthStatus `json:"current_status"`
	LastCheckTime      time.Time    `json:"last_check_time"`
	AvgResponseTimeMs  int          `json:"avg_response_time_ms"`
	CurrentErrorRate   float64      `json:"current_error_rate"`
	UptimePercentage   float64      `json:"uptime_percentage"`
	TotalChecks        int64        `json:"total_checks"`
	FailedChecks       int64        `json:"failed_checks"`
}
