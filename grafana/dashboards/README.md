# Admin System Grafana Dashboards

This directory contains pre-built Grafana dashboards for monitoring the admin system.

## Available Dashboards

### Admin System Dashboard (`admin-system-dashboard.json`)

A comprehensive dashboard that provides real-time monitoring of all admin system components.

#### Dashboard Panels

1. **Total Companies** - Gauge showing the total number of companies in the system
2. **Companies by Rating Category** - Time series showing distribution of companies across rating categories
3. **API Response Time** - Time series showing p95, p99, and average API response times
4. **API Request Rate** - Time series showing requests per second for each endpoint
5. **Server Health Status** - Table showing current health status of all servers
6. **Server Response Time** - Time series showing response times for each server
7. **Message Distribution Rate** - Time series showing message distribution rate by strategy
8. **Device Load Distribution** - Bar chart showing current load on each device
9. **Timing Violations per Minute** - Time series showing timing violations by severity
10. **Anomaly Detection Score** - Time series showing anomaly scores for each tenant
11. **Active Alerts** - Gauge showing total number of active alerts
12. **Alert Creation Rate** - Time series showing alert creation rate by severity

## Installation

### 1. Import Dashboard to Grafana

#### Option A: Using Grafana UI

1. Open Grafana web interface
2. Navigate to **Dashboards** → **Import**
3. Click **Upload JSON file**
4. Select `admin-system-dashboard.json`
5. Select your Prometheus data source
6. Click **Import**

#### Option B: Using Grafana API

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d @admin-system-dashboard.json \
  http://localhost:3000/api/dashboards/db
```

#### Option C: Using Grafana Provisioning

1. Copy the dashboard file to Grafana's provisioning directory:

```bash
cp admin-system-dashboard.json /etc/grafana/provisioning/dashboards/
```

2. Create a provisioning configuration file `/etc/grafana/provisioning/dashboards/admin.yaml`:

```yaml
apiVersion: 1

providers:
  - name: 'Admin System'
    orgId: 1
    folder: 'Admin'
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /etc/grafana/provisioning/dashboards
```

3. Restart Grafana:

```bash
systemctl restart grafana-server
```

### 2. Configure Prometheus Data Source

If you haven't already configured Prometheus as a data source:

1. Navigate to **Configuration** → **Data Sources**
2. Click **Add data source**
3. Select **Prometheus**
4. Configure the URL (e.g., `http://localhost:9090`)
5. Click **Save & Test**

## Dashboard Features

### Auto-Refresh

The dashboard is configured to auto-refresh every 30 seconds. You can change this in the dashboard settings.

### Time Range

Default time range is last 6 hours. You can adjust this using the time picker in the top-right corner.

### Variables

Currently, the dashboard doesn't use variables, but you can add them for filtering by:
- Tenant ID
- Server ID
- Device ID
- Alert Severity

To add variables:
1. Click the dashboard settings (gear icon)
2. Navigate to **Variables**
3. Click **Add variable**

### Alerts

You can configure alerts on any panel:
1. Edit the panel
2. Navigate to the **Alert** tab
3. Configure alert conditions
4. Set notification channels

## Customization

### Adding New Panels

1. Click **Add panel** in the dashboard
2. Select **Add new panel**
3. Configure the query using Prometheus metrics (see `services/backend/internal/monitoring/README.md` for available metrics)
4. Customize visualization options
5. Click **Apply**

### Modifying Existing Panels

1. Hover over a panel and click the title
2. Select **Edit**
3. Modify the query or visualization options
4. Click **Apply**

### Example Queries

Here are some useful Prometheus queries you can add:

```promql
# Error rate by endpoint
rate(admin_api_request_total{status_code=~"5.."}[5m])

# Average rating calculation time
rate(admin_rating_calculation_duration_seconds_sum[5m]) / rate(admin_rating_calculation_duration_seconds_count[5m])

# Servers by health status
sum by (status) (admin_server_health_status == 2)  # Healthy
sum by (status) (admin_server_health_status == 1)  # Degraded
sum by (status) (admin_server_health_status == 0)  # Unhealthy

# Distribution strategy usage
sum by (strategy_type) (rate(admin_distribution_strategy_usage_total[5m]))

# Enforcement actions per hour
rate(admin_enforcement_action_total[1h]) * 3600

# Message success rate
sum(rate(admin_message_volume_total{status="successful"}[5m])) / sum(rate(admin_message_volume_total[5m])) * 100
```

## Troubleshooting

### Dashboard Not Loading

1. Check that Prometheus data source is configured correctly
2. Verify that the admin backend is exposing metrics at `/metrics`
3. Check Prometheus is scraping the admin backend (check Prometheus targets page)

### No Data in Panels

1. Verify the time range includes data
2. Check that metrics are being exported (visit `http://localhost:8080/metrics`)
3. Verify Prometheus is successfully scraping metrics
4. Check the Prometheus query in the panel

### Slow Dashboard Performance

1. Reduce the time range
2. Increase the refresh interval
3. Optimize Prometheus queries using recording rules
4. Consider using Prometheus query caching

## Best Practices

1. **Use Recording Rules**: For expensive queries, create Prometheus recording rules to pre-compute them
2. **Set Appropriate Refresh Intervals**: Don't refresh too frequently (30s-1m is usually sufficient)
3. **Use Variables**: Add variables for filtering to make the dashboard more flexible
4. **Configure Alerts**: Set up alerts for critical metrics
5. **Document Changes**: Keep track of dashboard modifications
6. **Version Control**: Store dashboard JSON in version control

## Additional Resources

- [Grafana Documentation](https://grafana.com/docs/)
- [Prometheus Query Language](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [Grafana Dashboard Best Practices](https://grafana.com/docs/grafana/latest/best-practices/)
