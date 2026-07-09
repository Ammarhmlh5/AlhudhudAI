# Prometheus Alert Rules for Admin System

This directory contains Prometheus alert rules for monitoring the admin system.

## Alert Rules File

- `admin_alert_rules.yml` - Comprehensive alert rules for all admin system components

## Alert Groups

### 1. API Alerts (`admin_api_alerts`)

Monitors API performance and availability.

**Alerts:**
- `HighAPIResponseTime` - Warning when p95 response time > 0.5s
- `CriticalAPIResponseTime` - Critical when p95 response time > 1.0s
- `HighAPIErrorRate` - Warning when error rate > 5%
- `CriticalAPIErrorRate` - Critical when error rate > 10%

### 2. Server Health Alerts (`admin_server_health_alerts`)

Monitors dedicated server health and performance.

**Alerts:**
- `ServerUnhealthy` - Critical when server status is unhealthy
- `ServerDegraded` - Warning when server status is degraded
- `HighServerResponseTime` - Warning when response time > 500ms
- `CriticalServerResponseTime` - Critical when response time > 1000ms
- `ServerHealthCheckFailures` - Warning when health check failure rate is high

### 3. Distribution Alerts (`admin_distribution_alerts`)

Monitors message distribution system.

**Alerts:**
- `NoActiveDevices` - Critical when no devices are active
- `HighDeviceLoad` - Warning when device load > 80
- `CriticalDeviceLoad` - Critical when device load > 95

### 4. Timing Alerts (`admin_timing_alerts`)

Monitors Signal/Push timing violations and anomalies.

**Alerts:**
- `HighTimingViolationRate` - Warning when violations > 10/minute
- `CriticalTimingViolationRate` - Critical when severe violations > 5/minute
- `HighAnomalyScore` - Warning when anomaly score > 2
- `CriticalAnomalyScore` - Critical when anomaly score > 3
- `EnforcementActionTriggered` - Warning when enforcement action is taken

### 5. Alert System Alerts (`admin_alert_system_alerts`)

Monitors the alert system itself.

**Alerts:**
- `HighActiveAlerts` - Warning when active alerts > 50
- `CriticalActiveAlerts` - Critical when critical alerts > 10
- `HighAlertCreationRate` - Warning when alert creation rate > 1/second

### 6. Rating Alerts (`admin_rating_alerts`)

Monitors company rating system.

**Alerts:**
- `CompanyRatingDropped` - Warning when rating drops > 10 points in 15 minutes
- `CriticalCompanyRating` - Critical when rating < 40
- `SlowRatingCalculation` - Warning when calculation time > 10ms

### 7. Routing Alerts (`admin_routing_alerts`)

Monitors message routing system.

**Alerts:**
- `DedicatedServerUnavailable` - Critical when dedicated server is unavailable
- `HighRoutingFailureRate` - Warning when routing failure rate > 5%
- `SlowRoutingDecisions` - Warning when p95 routing latency > 50ms

### 8. Performance Alerts (`admin_performance_alerts`)

Monitors overall system performance.

**Alerts:**
- `LowMessageSuccessRate` - Warning when success rate < 95%
- `CriticalMessageSuccessRate` - Critical when success rate < 90%
- `SlowMetricsCollection` - Warning when metrics collection > 5s

## Installation

### 1. Add Alert Rules to Prometheus

Add the following to your `prometheus.yml`:

```yaml
rule_files:
  - "admin_alert_rules.yml"
```

### 2. Configure Alertmanager

Create or update `alertmanager.yml`:

```yaml
global:
  resolve_timeout: 5m

route:
  group_by: ['alertname', 'component']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 12h
  receiver: 'default'
  routes:
    - match:
        severity: critical
      receiver: 'critical'
      continue: true
    - match:
        severity: warning
      receiver: 'warning'

receivers:
  - name: 'default'
    webhook_configs:
      - url: 'http://localhost:5001/webhook'

  - name: 'critical'
    email_configs:
      - to: 'admin@example.com'
        from: 'alertmanager@example.com'
        smarthost: 'smtp.example.com:587'
        auth_username: 'alertmanager@example.com'
        auth_password: 'password'
        headers:
          Subject: '[CRITICAL] {{ .GroupLabels.alertname }}'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK'
        channel: '#critical-alerts'
        title: '[CRITICAL] {{ .GroupLabels.alertname }}'
        text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'

  - name: 'warning'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK'
        channel: '#warnings'
        title: '[WARNING] {{ .GroupLabels.alertname }}'
        text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'

inhibit_rules:
  - source_match:
      severity: 'critical'
    target_match:
      severity: 'warning'
    equal: ['alertname', 'component']
```

### 3. Reload Prometheus Configuration

```bash
# Send SIGHUP to Prometheus to reload configuration
kill -HUP $(pidof prometheus)

# Or use the API
curl -X POST http://localhost:9090/-/reload
```

### 4. Verify Alert Rules

Check that alert rules are loaded:

```bash
# List all alert rules
curl http://localhost:9090/api/v1/rules

# Check specific rule group
curl http://localhost:9090/api/v1/rules?type=alert
```

## Testing Alerts

### 1. Trigger Test Alerts

You can manually trigger alerts for testing:

```bash
# Simulate high API response time
# (This would require actually slowing down your API)

# Check pending/firing alerts
curl http://localhost:9090/api/v1/alerts
```

### 2. Use Alertmanager amtool

```bash
# Install amtool
go install github.com/prometheus/alertmanager/cmd/amtool@latest

# Check alert status
amtool alert query

# Silence an alert
amtool silence add alertname=HighAPIResponseTime

# List silences
amtool silence query
```

## Alert Severity Levels

### Critical
- Requires immediate attention
- Indicates service degradation or outage
- Sent to on-call engineers via email/SMS
- Examples: Server unhealthy, no active devices, critical error rate

### Warning
- Requires attention but not immediate
- Indicates potential issues
- Sent to team channels (Slack, etc.)
- Examples: High response time, degraded server, high load

## Alert Best Practices

### 1. Avoid Alert Fatigue
- Set appropriate thresholds
- Use `for` clause to avoid flapping alerts
- Group related alerts
- Use inhibit rules to suppress redundant alerts

### 2. Actionable Alerts
- Each alert should have a clear action
- Include relevant context in annotations
- Link to runbooks or documentation

### 3. Alert Tuning
- Monitor alert frequency
- Adjust thresholds based on actual system behavior
- Remove alerts that are never actionable

### 4. Testing
- Test alerts in staging environment
- Verify notification channels work
- Document expected behavior

## Customization

### Adjusting Thresholds

Edit `admin_alert_rules.yml` and modify the threshold values:

```yaml
# Example: Change API response time threshold
- alert: HighAPIResponseTime
  expr: histogram_quantile(0.95, rate(admin_api_request_duration_seconds_bucket[5m])) > 0.5  # Change this value
  for: 5m  # Change this duration
```

### Adding New Alerts

Add new alert rules to the appropriate group:

```yaml
- name: admin_api_alerts
  interval: 30s
  rules:
    # ... existing rules ...
    
    # New alert
    - alert: YourNewAlert
      expr: your_metric > threshold
      for: 5m
      labels:
        severity: warning
        component: api
      annotations:
        summary: "Brief description"
        description: "Detailed description with {{ $value }}"
```

### Creating New Alert Groups

Add a new group to `admin_alert_rules.yml`:

```yaml
groups:
  # ... existing groups ...
  
  - name: your_new_group
    interval: 30s
    rules:
      - alert: YourAlert
        expr: your_metric > threshold
        for: 5m
        labels:
          severity: warning
          component: your_component
        annotations:
          summary: "Brief description"
          description: "Detailed description"
```

## Notification Channels

### Email
```yaml
email_configs:
  - to: 'team@example.com'
    from: 'alertmanager@example.com'
    smarthost: 'smtp.example.com:587'
    auth_username: 'alertmanager@example.com'
    auth_password: 'password'
```

### Slack
```yaml
slack_configs:
  - api_url: 'https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK'
    channel: '#alerts'
    title: '{{ .GroupLabels.alertname }}'
    text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'
```

### PagerDuty
```yaml
pagerduty_configs:
  - service_key: 'YOUR_PAGERDUTY_SERVICE_KEY'
    description: '{{ .GroupLabels.alertname }}'
```

### Webhook
```yaml
webhook_configs:
  - url: 'http://your-webhook-endpoint.com/alert'
    send_resolved: true
```

## Monitoring Alerts

### Grafana Dashboard

Create a Grafana dashboard to monitor alerts:

```promql
# Number of firing alerts
ALERTS{alertstate="firing"}

# Number of pending alerts
ALERTS{alertstate="pending"}

# Alerts by severity
sum by (severity) (ALERTS{alertstate="firing"})

# Alerts by component
sum by (component) (ALERTS{alertstate="firing"})
```

### Prometheus Queries

```promql
# Check if specific alert is firing
ALERTS{alertname="HighAPIResponseTime", alertstate="firing"}

# Count firing alerts
count(ALERTS{alertstate="firing"})

# Alert history (requires recording rules)
rate(ALERTS_FOR_STATE[5m])
```

## Troubleshooting

### Alerts Not Firing

1. Check that Prometheus is scraping metrics correctly
2. Verify the alert expression evaluates to true
3. Check the `for` duration hasn't been exceeded
4. Verify Alertmanager is configured correctly

### Too Many Alerts

1. Review and adjust thresholds
2. Add `for` clauses to prevent flapping
3. Use inhibit rules to suppress redundant alerts
4. Group related alerts

### Alerts Not Being Sent

1. Check Alertmanager logs
2. Verify notification channel configuration
3. Test notification channels manually
4. Check network connectivity

## Additional Resources

- [Prometheus Alerting Documentation](https://prometheus.io/docs/alerting/latest/overview/)
- [Alertmanager Documentation](https://prometheus.io/docs/alerting/latest/alertmanager/)
- [Alert Rule Best Practices](https://prometheus.io/docs/practices/alerting/)
