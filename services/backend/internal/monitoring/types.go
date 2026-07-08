package monitoring

import (
	"time"

	"github.com/google/uuid"
)

// SystemMetrics represents system-level metrics for a dedicated gateway
type SystemMetrics struct {
	CPUUsagePercent    float64 // CPU usage percentage (0-100)
	MemoryUsageMB      float64 // Memory usage in MB
	MemoryUsagePercent float64 // Memory usage percentage (0-100)
	BandwidthUsageKBps float64 // Bandwidth usage in KB/s
}

// DatabaseMetrics represents database-related metrics
type DatabaseMetrics struct {
	ActiveConnections int     // Number of active database connections
	IdleConnections   int     // Number of idle connections
	MaxConnections    int     // Maximum allowed connections
	QueriesPerSecond  float64 // Number of queries per second
	AvgQueryTime      float64 // Average query time in milliseconds
}


// GatewayMetrics represents comprehensive metrics for a dedicated gateway
type GatewayMetrics struct {
	GatewayID       int             // Gateway ID
	TenantID        uuid.UUID       // Tenant ID
	Timestamp       time.Time       // Timestamp of collection
	SystemMetrics   SystemMetrics   // System-level metrics
	DatabaseMetrics DatabaseMetrics // Database metrics
	RedisMetrics    RedisMetrics    // Redis metrics
}

// MetricsSnapshot represents a point-in-time snapshot of all metrics
type MetricsSnapshot struct {
	Timestamp time.Time
	Gateways  []GatewayMetrics
}

// AlertThreshold represents threshold values for alerting
type AlertThreshold struct {
	CPUWarning      float64 // CPU usage warning threshold (e.g., 80%)
	CPUCritical     float64 // CPU usage critical threshold (e.g., 95%)
	MemoryWarning   float64 // Memory usage warning threshold
	MemoryCritical  float64 // Memory usage critical threshold
	BandwidthLimit  float64 // Bandwidth limit in KB/s
	ConnectionLimit int     // Maximum database connections
}

// Alert represents a monitoring alert
type Alert struct {
	ID          uuid.UUID
	GatewayID   int
	TenantID    uuid.UUID
	AlertType   string    // "CPU", "MEMORY", "BANDWIDTH", "DATABASE", "REDIS"
	Severity    string    // "WARNING", "CRITICAL"
	Message     string
	MetricValue float64
	Threshold   float64
	Timestamp   time.Time
	Resolved    bool
	ResolvedAt  *time.Time
}
