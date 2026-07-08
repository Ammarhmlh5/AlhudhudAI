# Admin System Monitoring

This package provides Prometheus metrics for monitoring the admin system.

## Available Metrics

### Rating System Metrics

- `admin_rating_calculation_duration_seconds` - Duration of rating calculations
- `admin_rating_calculation_total` - Total number of rating calculations
- `admin_company_rating_score` - Current rating score for companies

### Server Health Metrics

- `admin_server_health_check_duration_seconds` - Duration of server health checks
- `admin_server_health_check_total` - Total number of server health checks
- `admin_server_health_status` - Current health status (0=unhealthy, 1=degraded, 2=healthy)
- `admin_server_response_time_ms` - Current response time in milliseconds

### Distribution Metrics

- `admin_message_distribution_total` - Total number of messages distributed
- `admin_device_load_current` - Current load on devices
- `admin_device_status_active` - Device active status (0=inactive, 1=active)
- `admin_distribution_strategy_usage_total` - Total usage count of distribution strategies

### Signal/Push Timing Metrics

- `admin_timing_violation_total` - Total number of timing violations
- `admin_message_gap_duration_ms` - Duration between messages in milliseconds
- `admin_current_messages_per_minute` - Current messages per minute rate
- `admin_anomaly_detection_score` - Current anomaly detection score
- `admin_enforcement_action_total` - Total number of enforcement actions taken

### Alert Metrics

- `admin_alert_created_total` - Total number of alerts created
- `admin_alert_acknowledged_total` - Total number of alerts acknowledged
- `admin_alert_resolved_total` - Total number of alerts resolved
- `admin_active_alerts_count` - Current number of active alerts

### API Metrics

- `admin_api_request_duration_seconds` - Duration of API requests
- `admin_api_request_total` - Total number of API requests

### Routing Metrics

- `admin_routing_decision_duration_seconds` - Duration of routing decisions
- `admin_routing_decision_total` - Total number of routing decisions
- `admin_dedicated_server_availability` - Availability of dedicated servers

### Performance Metrics

- `admin_performance_metrics_collection_duration_seconds` - Duration of metrics collection
- `admin_message_success_rate` - Message success rate percentage
- `admin_message_volume_total` - Total message volume

## Usage

### 1. Setup Metrics Endpoint

```go
import "your-project/internal/monitoring"

func main() {
    router := gin.Default()
    
    // Add Prometheus middleware
    router.Use(monitoring.PrometheusMiddleware())
    
    // Setup metrics endpoint
    monitoring.SetupMetricsEndpoint(router)
    
    router.Run(":8080")
}
```

### 2. Record Metrics in Your Code

```go
import (
    "time"
    "your-project/internal/monitoring"
)

// Example: Recording rating calculation
func CalculateRating(tenantID string) {
    start := time.Now()
    
    // ... your calculation logic ...
    
    duration := time.Since(start).Seconds()
    monitoring.RatingCalculationDuration.WithLabelValues(tenantID).Observe(duration)
    monitoring.RatingCalculationTotal.WithLabelValues(tenantID, "excellent").Inc()
    monitoring.CompanyRatingScore.WithLabelValues(tenantID, "excellent").Set(95.5)
}

// Example: Recording server health check
func CheckServerHealth(serverID, tenantID string, responseTime float64, status string) {
    monitoring.ServerResponseTime.WithLabelValues(serverID, tenantID).Set(responseTime)
    monitoring.ServerHealthStatus.WithLabelValues(serverID, tenantID).Set(
        monitoring.HealthStatusToValue(status),
    )
    monitoring.ServerHealthCheckTotal.WithLabelValues(serverID, tenantID, status).Inc()
}

// Example: Recording message distribution
func DistributeMessage(tenantID, deviceID, strategy string) {
    monitoring.MessageDistributionTotal.WithLabelValues(tenantID, deviceID, strategy).Inc()
    monitoring.DistributionStrategyUsage.WithLabelValues(tenantID, strategy).Inc()
}

// Example: Recording timing violation
func RecordTimingViolation(tenantID, messageType, severity string, gap float64) {
    monitoring.TimingViolationTotal.WithLabelValues(tenantID, messageType, severity).Inc()
    monitoring.MessageGapDuration.WithLabelValues(tenantID, messageType).Observe(gap)
}

// Example: Recording alert
func CreateAlert(tenantID, alertType, severity string) {
    monitoring.AlertCreatedTotal.WithLabelValues(tenantID, alertType, severity).Inc()
    monitoring.ActiveAlertsCount.WithLabelValues(tenantID, severity).Inc()
}
```

### 3. Configure Prometheus

Add this job to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'admin-backend'
    scrape_interval: 15s
    static_configs:
      - targets: ['localhost:8080']
```

### 4. Access Metrics

Metrics are available at: `http://localhost:8080/metrics`

## Prometheus Queries Examples

### Rating System

```promql
# Average rating calculation duration
rate(admin_rating_calculation_duration_seconds_sum[5m]) / rate(admin_rating_calculation_duration_seconds_count[5m])

# Companies by rating category
sum by (category) (admin_company_rating_score)

# Rating calculations per second
rate(admin_rating_calculation_total[5m])
```

### Server Health

```promql
# Servers by health status
sum by (status) (admin_server_health_status)

# Average server response time
avg(admin_server_response_time_ms)

# Health check failure rate
rate(admin_server_health_check_total{status="unhealthy"}[5m])
```

### Distribution

```promql
# Messages distributed per second
rate(admin_message_distribution_total[5m])

# Device load distribution
avg by (device_id) (admin_device_load_current)

# Strategy usage distribution
sum by (strategy_type) (admin_distribution_strategy_usage_total)
```

### Signal/Push

```promql
# Timing violations per minute
rate(admin_timing_violation_total[1m]) * 60

# Average message gap
rate(admin_message_gap_duration_ms_sum[5m]) / rate(admin_message_gap_duration_ms_count[5m])

# Anomaly detection score
admin_anomaly_detection_score

# Enforcement actions per hour
rate(admin_enforcement_action_total[1h]) * 3600
```

### Alerts

```promql
# Active alerts by severity
sum by (severity) (admin_active_alerts_count)

# Alert creation rate
rate(admin_alert_created_total[5m])

# Alert resolution time (requires custom recording)
# (alert_resolved_timestamp - alert_created_timestamp)
```

### API Performance

```promql
# API request rate
rate(admin_api_request_total[5m])

# Average API response time
rate(admin_api_request_duration_seconds_sum[5m]) / rate(admin_api_request_duration_seconds_count[5m])

# 95th percentile API response time
histogram_quantile(0.95, rate(admin_api_request_duration_seconds_bucket[5m]))

# Error rate
rate(admin_api_request_total{status_code=~"5.."}[5m])
```

## Best Practices

1. **Label Cardinality**: Be careful with high-cardinality labels (like user IDs). Use tenant_id instead of individual user IDs.

2. **Metric Naming**: Follow Prometheus naming conventions:
   - Use `_total` suffix for counters
   - Use `_duration_seconds` for durations
   - Use descriptive names

3. **Recording Rules**: For expensive queries, use Prometheus recording rules to pre-compute them.

4. **Alerting**: Set up alerts for critical metrics (high error rates, slow response times, etc.)

5. **Retention**: Configure appropriate retention periods based on your needs.

## Integration with Grafana

See `grafana/dashboards/admin-system-dashboard.json` for pre-built dashboards.
