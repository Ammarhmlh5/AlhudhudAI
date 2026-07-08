# Server Health Monitor

This package provides comprehensive server health monitoring capabilities for dedicated servers in the admin interface enhancements feature.

## Overview

The Server Health Monitor system performs periodic health checks on dedicated servers, classifies their health status, triggers alerts on status changes, and maintains a 30-day history of health check results.

## Components

### 1. Health Checker (`checker.go`)

Performs periodic HTTP health checks on all active dedicated servers.

**Features:**
- Runs health checks every 30 seconds
- Measures response time and calculates error rates
- Concurrent health checking for multiple servers
- Configurable timeout (default: 10 seconds)
- Automatic error rate calculation from recent history

**Usage:**
```go
// Create dependencies
classifier := health.NewStatusClassifier()
alertTrigger := health.NewAlertTrigger(db)
logger := health.NewHistoryLogger(db)

// Create health checker
checker := health.NewHealthChecker(db, classifier, alertTrigger, logger)

// Start periodic checks
ctx := context.Background()
if err := checker.Start(ctx); err != nil {
    log.Fatal(err)
}

// Stop when done
defer checker.Stop()

// Perform a single check on a specific server
result, err := checker.CheckServerHealthOnce(ctx, serverID)
```

### 2. Status Classifier (`classifier.go`)

Classifies server health status based on response time and error rate.

**Classification Criteria:**
- **Healthy**: response_time < 200ms AND error_rate < 1%
- **Degraded**: response_time 200-500ms OR error_rate 1-5%
- **Unhealthy**: response_time > 500ms OR error_rate > 5%

**Usage:**
```go
classifier := health.NewStatusClassifier()

// Classify based on metrics
status := classifier.ClassifyHealthStatus(150, 0.5) // Returns: Healthy

// Check specific conditions
isHealthy := classifier.IsHealthy(150, 0.5)
isDegraded := classifier.IsDegraded(300, 2.0)
isUnhealthy := classifier.IsUnhealthy(600, 10.0)

// Custom thresholds
customClassifier := health.NewStatusClassifierWithThresholds(
    100, 100, 300,  // Response time thresholds (ms)
    0.5, 0.5, 3.0,  // Error rate thresholds (%)
)
```

### 3. Alert Trigger (`alert.go`)

Triggers alerts when server health status changes.

**Features:**
- Detects status changes by comparing with previous status
- Creates alerts with appropriate severity (critical, warning, info)
- Prevents duplicate alerts within 5 minutes
- Caches last known status for performance
- Integrates with the alerts table

**Usage:**
```go
alertTrigger := health.NewAlertTrigger(db)

// Called automatically by health checker
err := alertTrigger.OnStatusChange(ctx, healthCheckResult)

// Clear cache if needed
alertTrigger.ClearStatusCache()
```

### 4. History Logger (`logger.go`)

Logs health check results and provides querying capabilities.

**Features:**
- Saves all health check results to database
- Retrieves health history with flexible filtering
- Generates health summaries with statistics
- Automatic cleanup of logs older than 30 days
- Uptime percentage calculation

**Usage:**
```go
logger := health.NewHistoryLogger(db)

// Log a health check result
err := logger.LogHealthCheck(ctx, result)

// Get health history with filters
filter := health.HealthHistoryFilter{
    ServerID:  serverID,
    StartTime: &startTime,
    EndTime:   &endTime,
    Status:    &health.Healthy,
    Limit:     100,
    Offset:    0,
}
history, err := logger.GetHealthHistory(ctx, filter)

// Get latest health check
latest, err := logger.GetLatestHealthCheck(ctx, serverID)

// Get health summary for last 24 hours
summary, err := logger.GetHealthSummary(ctx, serverID, 24*time.Hour)

// Cleanup old logs (older than 30 days)
rowsDeleted, err := logger.CleanupOldLogs(ctx)

// Get health check counts by status
counts, err := logger.GetHealthCheckCountByStatus(ctx, serverID, 7*24*time.Hour)
```

## Data Models

### HealthCheckResult
```go
type HealthCheckResult struct {
    ServerID       uuid.UUID
    TenantID       uuid.UUID
    CheckTimestamp time.Time
    ResponseTimeMs int
    StatusCode     int
    IsAvailable    bool
    ErrorMessage   string
    ErrorCount     int
    ErrorRate      float64
    HealthStatus   HealthStatus
}
```

### HealthSummary
```go
type HealthSummary struct {
    ServerID           uuid.UUID
    TenantID           uuid.UUID
    CurrentStatus      HealthStatus
    LastCheckTime      time.Time
    AvgResponseTimeMs  int
    CurrentErrorRate   float64
    UptimePercentage   float64
    TotalChecks        int64
    FailedChecks       int64
}
```

## Database Schema

The health monitor uses the `server_health_logs` table:

```sql
CREATE TABLE server_health_logs (
    id BIGSERIAL PRIMARY KEY,
    server_id UUID NOT NULL REFERENCES dedicated_gateways(id),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    check_timestamp TIMESTAMP NOT NULL,
    response_time_ms INTEGER,
    status_code INTEGER,
    is_available BOOLEAN NOT NULL,
    error_message TEXT,
    error_count INTEGER NOT NULL DEFAULT 0,
    error_rate DECIMAL(5,2),
    health_status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

## Integration

### With Alert System

The health monitor integrates with the alerts system to notify administrators of status changes:

```go
// Alerts are automatically created in the alerts table
// with appropriate severity and context
```

### With Routing Engine

The routing engine can query server health before routing:

```go
// Get latest health status
latest, err := logger.GetLatestHealthCheck(ctx, serverID)
if err != nil || latest.HealthStatus == health.Unhealthy {
    return ErrServerUnavailable
}
```

## Configuration

### Health Check Interval
Default: 30 seconds (configurable in `NewHealthChecker`)

### Health Check Timeout
Default: 10 seconds (configurable in `NewHealthChecker`)

### Retention Period
Default: 30 days (automatic cleanup via `CleanupOldLogs`)

### Classification Thresholds
- Healthy: < 200ms, < 1% error rate
- Degraded: 200-500ms, 1-5% error rate
- Unhealthy: > 500ms, > 5% error rate

## Best Practices

1. **Run health checks in background**: Use `Start()` to run periodic checks
2. **Handle errors gracefully**: Health check failures should not crash the system
3. **Monitor alert volume**: Too many alerts may indicate threshold tuning needed
4. **Regular cleanup**: Schedule periodic cleanup of old logs
5. **Cache optimization**: The alert trigger caches last status for performance

## Requirements Mapping

This implementation satisfies the following requirements:

- **Requirement 5.1**: Health check using HTTP endpoints
- **Requirement 5.2**: Response time measurement
- **Requirement 5.3**: Error rate tracking
- **Requirement 5.4**: Status classification (healthy/degraded/unhealthy)
- **Requirement 5.5**: Status updates within 5 seconds
- **Requirement 5.7**: 30-day health history retention
- **Requirement 2.6**: Alert generation on status changes

## Testing

Unit tests and property-based tests are provided in:
- `checker_test.go`
- `classifier_test.go`
- `alert_test.go`
- `logger_test.go`
- `health_property_test.go`

Run tests:
```bash
go test ./internal/admin/health/...
```

## Future Enhancements

- WebSocket support for real-time health updates
- Custom health check endpoints per server
- Health check scheduling based on server priority
- Advanced anomaly detection
- Health trend analysis and predictions
