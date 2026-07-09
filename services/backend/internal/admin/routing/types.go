package routing

import (
	"time"

	"github.com/google/uuid"
)

// RoutingType represents the type of routing for a tenant
type RoutingType string

const (
	// RoutingTypeShared indicates routing to shared infrastructure
	RoutingTypeShared RoutingType = "shared"
	// RoutingTypeDedicated indicates routing to dedicated server
	RoutingTypeDedicated RoutingType = "dedicated"
)

// HealthStatus represents the health status of a dedicated server
type HealthStatus string

const (
	// HealthStatusHealthy indicates server is healthy (response < 200ms, error rate < 1%)
	HealthStatusHealthy HealthStatus = "healthy"
	// HealthStatusDegraded indicates server is degraded (response 200-500ms or error rate 1-5%)
	HealthStatusDegraded HealthStatus = "degraded"
	// HealthStatusUnhealthy indicates server is unhealthy (response > 500ms or error rate > 5%)
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// RoutingConfig represents a routing configuration for a tenant
type RoutingConfig struct {
	ID                   int64      `json:"id" db:"id"`
	TenantID             uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	RoutingType          string     `json:"routing_type" db:"routing_type"`
	DedicatedServerID    *uuid.UUID `json:"dedicated_server_id,omitempty" db:"dedicated_server_id"`
	DedicatedServerURL   *string    `json:"dedicated_server_url,omitempty" db:"dedicated_server_url"`
	DedicatedServerAPIKey *string   `json:"dedicated_server_api_key,omitempty" db:"dedicated_server_api_key"`
	TotalRequests        int64      `json:"total_requests" db:"total_requests"`
	SuccessfulRequests   int64      `json:"successful_requests" db:"successful_requests"`
	FailedRequests       int64      `json:"failed_requests" db:"failed_requests"`
	AverageLatencyMs     *int       `json:"average_latency_ms,omitempty" db:"average_latency_ms"`
	IsActive             bool       `json:"is_active" db:"is_active"`
	LastHealthCheck      *time.Time `json:"last_health_check,omitempty" db:"last_health_check"`
	HealthStatus         *string    `json:"health_status,omitempty" db:"health_status"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at" db:"updated_at"`
}

// Message represents a message to be routed
type Message struct {
	ID        uuid.UUID              `json:"id"`
	TenantID  uuid.UUID              `json:"tenant_id"`
	Content   string                 `json:"content"`
	Recipient string                 `json:"recipient"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// RoutingDecision represents the result of a routing decision
type RoutingDecision struct {
	TenantID          uuid.UUID  `json:"tenant_id"`
	RoutingType       string     `json:"routing_type"`
	DedicatedServerID *uuid.UUID `json:"dedicated_server_id,omitempty"`
	DedicatedServerURL *string   `json:"dedicated_server_url,omitempty"`
	HealthStatus      *string    `json:"health_status,omitempty"`
	ShouldRoute       bool       `json:"should_route"`
	Error             error      `json:"error,omitempty"`
}

// RoutingStats represents routing statistics for a tenant
type RoutingStats struct {
	TenantID           uuid.UUID  `json:"tenant_id"`
	RoutingType        string     `json:"routing_type"`
	TotalRequests      int64      `json:"total_requests"`
	SuccessfulRequests int64      `json:"successful_requests"`
	FailedRequests     int64      `json:"failed_requests"`
	SuccessRate        *float64   `json:"success_rate,omitempty"`
	AverageLatencyMs   *int       `json:"average_latency_ms,omitempty"`
	HealthStatus       *string    `json:"health_status,omitempty"`
	LastHealthCheck    *time.Time `json:"last_health_check,omitempty"`
}

// RoutingMetrics represents routing metrics for a tenant
type RoutingMetrics struct {
	TenantID           uuid.UUID `json:"tenant_id"`
	TotalRequests      int64     `json:"total_requests"`
	SuccessfulRequests int64     `json:"successful_requests"`
	FailedRequests     int64     `json:"failed_requests"`
	SuccessRate        *float64  `json:"success_rate,omitempty"`
	AverageLatencyMs   *int      `json:"average_latency_ms,omitempty"`
}

// UpdateRoutingConfigInput represents input for updating routing configuration
type UpdateRoutingConfigInput struct {
	TenantID             uuid.UUID  `json:"tenant_id"`
	RoutingType          string     `json:"routing_type"`
	DedicatedServerID    *uuid.UUID `json:"dedicated_server_id,omitempty"`
	DedicatedServerURL   *string    `json:"dedicated_server_url,omitempty"`
	DedicatedServerAPIKey *string   `json:"dedicated_server_api_key,omitempty"`
	IsActive             *bool      `json:"is_active,omitempty"`
}

// TenantMetrics holds in-memory metrics for a tenant
type TenantMetrics struct {
	TenantID           uuid.UUID
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	TotalLatencyMs     int64
	LastUpdated        time.Time
}
