# Monitoring Setup Guide - Tariff Pricing System

## Task 35.5: Monitoring Setup
**Requirements:** 29.1, 29.9, 29.10, 29.12

## Overview

This guide covers the complete monitoring setup for the Tariff Pricing System using Prometheus for metrics collection and Grafana for visualization.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Monitoring Stack                          │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────┐      ┌──────────────┐                     │
│  │   Backend    │─────▶│  Prometheus  │                     │
│  │  /metrics    │      │   (Scraper)  │                     │
│  └──────────────┘      └──────┬───────┘                     │
│                               │                              │
│  ┌──────────────┐             │        ┌──────────────┐     │
│  │  PostgreSQL  │─────────────┼───────▶│   Grafana    │     │
│  │   Exporter   │             │        │ (Dashboards) │     │
│  └──────────────┘             │        └──────────────┘     │
│                               │                              │
│  ┌──────────────┐             │        ┌──────────────┐     │
│  │    Redis     │─────────────┼───────▶│ Alertmanager │     │
│  │   Exporter   │             │        │   (Alerts)   │     │
│  └──────────────┘             │        └──────────────┘     │
│                               │                              │
│  ┌──────────────┐             │                              │
│  │     Node     │─────────────┘                              │
│  │   Exporter   │                                            │
│  └──────────────┘                                            │
│                                                               │
│  ┌──────────────┐                                            │
│  │   cAdvisor   │                                            │
│  │  (Containers)│                                            │
│  └──────────────┘                                            │
└─────────────────────────────────────────────────────────────┘
```

## Components

### 1. Prometheus
- **Purpose:** Metrics collection and storage
- **Port:** 9090
- **Retention:** 30 days
- **Scrape Interval:** 15 seconds

### 2. Grafana
- **Purpose:** Metrics visualization
- **Port:** 3001
- **Default Credentials:** admin/admin123
- **Dashboards:** Pre-configured for tariff system

### 3. Alertmanager
- **Purpose:** Alert routing and notification
- **Port:** 9093
- **Channels:** Email, Slack (optional), PagerDuty (optional)

### 4. Exporters
- **PostgreSQL Exporter:** Database metrics (port 9187)
- **Redis Exporter:** Cache metrics (port 9121)
- **Node Exporter:** System metrics (port 9100)
- **cAdvisor:** Container metrics (port 8081)

## Quick Start

### 1. Start Monitoring Stack

```bash
# Start all monitoring services
docker-compose -f docker-compose.monitoring.yml up -d

# Check status
docker-compose -f docker-compose.monitoring.yml ps

# View logs
docker-compose -f docker-compose.monitoring.yml logs -f
```

### 2. Access Dashboards

- **Prometheus:** http://localhost:9090
- **Grafana:** http://localhost:3001 (admin/admin123)
- **Alertmanager:** http://localhost:9093

### 3. Verify Metrics Collection

```bash
# Check if Prometheus is scraping targets
curl http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | {job: .labels.job, health: .health}'

# Check backend metrics
curl http://localhost:8080/metrics

# Query a specific metric
curl 'http://localhost:9090/api/v1/query?query=tariff_subscriptions_total' | jq
```

## Metrics Exposed

### Subscription Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `tariff_subscriptions_total` | Gauge | Total active subscriptions by status |
| `tariff_subscription_renewals_total` | Counter | Total subscription renewals |
| `tariff_subscription_renewal_failures_total` | Counter | Failed renewal attempts |
| `tariff_subscriptions_expiring_soon` | Gauge | Subscriptions expiring within 24h |

### Rollover Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `tariff_rollover_messages` | Gauge | Current rollover messages |
| `tariff_rollover_activated_total` | Counter | Rollover messages activated |
| `tariff_rollover_forfeited_total` | Counter | Rollover messages forfeited |
| `tariff_rollover_expiring_soon` | Gauge | Rollover messages expiring within 7 days |

### Gateway Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `tariff_dedicated_gateways_total` | Gauge | Total dedicated gateways by status |
| `tariff_gateway_messages_total` | Counter | Messages processed per gateway |
| `tariff_gateway_resource_usage` | Gauge | Resource usage percentage per gateway |
| `tariff_gateway_quota_exceeded_total` | Counter | Quota exceeded events |

### Tariff Calculation Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `tariff_calculations_total` | Counter | Total tariff calculations |
| `tariff_calculation_errors_total` | Counter | Calculation errors |
| `tariff_missing_definitions_total` | Counter | Missing tariff definitions |
| `tariff_calculation_duration_seconds` | Histogram | Calculation duration |

### Balance Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `tariff_balance_deductions_total` | Counter | Total balance deductions |
| `tariff_balance_deductions_sum` | Counter | Sum of deducted amounts |
| `tariff_balance_deduction_failures_total` | Counter | Failed deductions |
| `tariff_insufficient_balance_total` | Counter | Insufficient balance attempts |

### Database Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `tariff_query_duration_seconds` | Histogram | Query execution time |
| `tariff_db_connections_active` | Gauge | Active database connections |
| `tariff_db_connections_idle` | Gauge | Idle database connections |

### Cache Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `tariff_cache_hits_total` | Counter | Cache hits |
| `tariff_cache_misses_total` | Counter | Cache misses |
| `tariff_cache_hit_rate` | Gauge | Cache hit rate (0-1) |
| `tariff_cache_evictions_total` | Counter | Cache evictions |

### API Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `tariff_api_requests_total` | Counter | Total API requests |
| `tariff_api_response_time_seconds` | Histogram | API response time |
| `tariff_errors_total` | Counter | Total errors |

## Grafana Dashboards

### 1. Tariff System Overview Dashboard

**Panels:**
- Total Subscriptions (stat)
- Rollover Messages (stat)
- Dedicated Gateways (stat)
- Error Rate (stat)
- Subscriptions by Status (pie chart)
- Subscriptions by Package (pie chart)
- Balance Deductions Over Time (time series)
- Rollover Messages Over Time (time series)
- Query Duration (time series)
- Cache Hit Rate (gauge)
- Messages by Gateway (time series)
- Gateway Resource Usage (table)

**Access:** http://localhost:3001/d/tariff-system-dashboard

### 2. Creating Custom Dashboards

```bash
# Import dashboard from JSON
curl -X POST http://admin:admin123@localhost:3001/api/dashboards/db \
  -H "Content-Type: application/json" \
  -d @grafana/dashboards/my-dashboard.json
```

## Alert Rules

### Configured Alerts

1. **Subscription Alerts**
   - High renewal failures (>5/sec for 5m)
   - Critical renewal failures (>20/sec for 2m)
   - Many expiring subscriptions (>100 for 1h)

2. **Rollover Alerts**
   - High forfeitures (>50/hour for 30m)
   - Many expiring soon (>1000 for 1h)

3. **Gateway Alerts**
   - High resource usage (>80% for 10m)
   - Critical resource usage (>95% for 5m)
   - Quota exceeded

4. **Tariff Calculation Alerts**
   - High calculation errors (>1/sec for 5m)
   - Missing tariff definitions

5. **Balance Alerts**
   - High insufficient balance rate (>10/sec for 5m)
   - Balance deduction failures (>1/sec for 5m)

6. **Performance Alerts**
   - Slow queries (>0.5s p95 for 10m)
   - Low cache hit rate (<80% for 15m)
   - High error rate (>0.05/sec for 5m)

### Testing Alerts

```bash
# Check alert rules are loaded
curl http://localhost:9090/api/v1/rules | jq '.data.groups[].rules[] | {alert: .name, state: .state}'

# Check active alerts
curl http://localhost:9090/api/v1/alerts | jq '.data.alerts[] | {name: .labels.alertname, state: .state}'

# Trigger test alert (simulate high error rate)
# This would require generating actual errors in the system
```

## Alertmanager Configuration

### Email Notifications

Edit `prometheus/alertmanager.yml`:

```yaml
global:
  smtp_smarthost: 'smtp.gmail.com:587'
  smtp_from: 'alerts@alhudhud.com'
  smtp_auth_username: 'alerts@alhudhud.com'
  smtp_auth_password: 'your-password'
```

### Slack Notifications (Optional)

```yaml
receivers:
  - name: 'critical-alerts'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK'
        channel: '#critical-alerts'
        title: '[CRITICAL] {{ .GroupLabels.alertname }}'
```

### PagerDuty Integration (Optional)

```yaml
receivers:
  - name: 'critical-alerts'
    pagerduty_configs:
      - service_key: 'your-pagerduty-service-key'
```

## Maintenance Tasks

### 1. Reload Prometheus Configuration

```bash
# Reload without restart
curl -X POST http://localhost:9090/-/reload

# Or restart container
docker restart tariff-prometheus
```

### 2. Backup Prometheus Data

```bash
# Create snapshot
curl -X POST http://localhost:9090/api/v1/admin/tsdb/snapshot

# Backup volume
docker run --rm -v prometheus-data:/data -v $(pwd):/backup \
  alpine tar czf /backup/prometheus-backup-$(date +%Y%m%d).tar.gz /data
```

### 3. Clean Old Data

```bash
# Prometheus automatically cleans data older than retention period (30 days)
# To manually clean:
docker exec tariff-prometheus promtool tsdb clean-tombstones /prometheus
```

### 4. Update Dashboards

```bash
# Export dashboard
curl http://admin:admin123@localhost:3001/api/dashboards/uid/tariff-system-dashboard \
  | jq '.dashboard' > grafana/dashboards/tariff-system-dashboard.json

# Import updated dashboard
docker restart tariff-grafana
```

## Troubleshooting

### Prometheus Not Scraping Targets

```bash
# Check targets status
curl http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | select(.health != "up")'

# Check Prometheus logs
docker logs tariff-prometheus

# Verify backend metrics endpoint
curl http://localhost:8080/metrics
```

### Grafana Dashboard Not Loading

```bash
# Check Grafana logs
docker logs tariff-grafana

# Verify datasource connection
curl http://admin:admin123@localhost:3001/api/datasources

# Test Prometheus query
curl 'http://localhost:9090/api/v1/query?query=up'
```

### Alerts Not Firing

```bash
# Check alert rules
curl http://localhost:9090/api/v1/rules

# Check Alertmanager status
curl http://localhost:9093/api/v1/status

# View Alertmanager logs
docker logs tariff-alertmanager
```

### High Memory Usage

```bash
# Check Prometheus memory
docker stats tariff-prometheus

# Reduce retention if needed
# Edit docker-compose.monitoring.yml:
# --storage.tsdb.retention.time=15d
# --storage.tsdb.retention.size=25GB
```

## Performance Tuning

### 1. Optimize Scrape Intervals

```yaml
# For high-frequency metrics
scrape_interval: 10s

# For low-frequency metrics
scrape_interval: 60s
```

### 2. Reduce Cardinality

```go
// Avoid high-cardinality labels
// BAD: user_id as label
prometheus.NewCounterVec(opts, []string{"user_id"})

// GOOD: aggregate by tenant
prometheus.NewCounterVec(opts, []string{"tenant_id"})
```

### 3. Use Recording Rules

```yaml
# Pre-calculate expensive queries
groups:
  - name: tariff_recording_rules
    interval: 1m
    rules:
      - record: tariff:cache_hit_rate:5m
        expr: rate(tariff_cache_hits_total[5m]) / (rate(tariff_cache_hits_total[5m]) + rate(tariff_cache_misses_total[5m]))
```

## Security Considerations

### 1. Enable Authentication

```yaml
# Prometheus basic auth
basic_auth_users:
  admin: $2y$10$...hashed_password...
```

### 2. Use TLS

```yaml
# Enable HTTPS
tls_server_config:
  cert_file: /etc/prometheus/cert.pem
  key_file: /etc/prometheus/key.pem
```

### 3. Restrict Access

```yaml
# Firewall rules
# Only allow access from specific IPs
```

## Best Practices

1. **Set Appropriate Retention:**
   - Balance storage costs with data needs
   - Default: 30 days for detailed metrics
   - Use remote write for long-term storage

2. **Use Meaningful Labels:**
   - Keep cardinality low
   - Use consistent naming
   - Document label meanings

3. **Create Actionable Alerts:**
   - Include clear descriptions
   - Specify impact and action
   - Avoid alert fatigue

4. **Regular Review:**
   - Review dashboards monthly
   - Update thresholds based on trends
   - Remove obsolete metrics

5. **Document Everything:**
   - Document custom metrics
   - Explain alert thresholds
   - Maintain runbooks

## Related Files

- `prometheus/prometheus-tariff.yml` - Prometheus configuration
- `prometheus/tariff_alert_rules.yml` - Alert rules
- `prometheus/alertmanager.yml` - Alertmanager configuration
- `docker-compose.monitoring.yml` - Monitoring stack
- `grafana/dashboards/tariff-system-dashboard.json` - Main dashboard

## Support

For issues or questions:
- Check logs: `docker-compose -f docker-compose.monitoring.yml logs`
- Verify configuration: `promtool check config prometheus/prometheus-tariff.yml`
- Contact: operations@alhudhud.com
