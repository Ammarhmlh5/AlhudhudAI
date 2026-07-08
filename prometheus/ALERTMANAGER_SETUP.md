# Alertmanager Setup Guide

## Overview

This guide explains how to set up Alertmanager to receive and route alerts from Prometheus for the Al-Hudhud SMS Gateway system.

## What is Alertmanager?

Alertmanager handles alerts sent by Prometheus server. It takes care of:
- **Deduplicating** alerts (grouping identical alerts)
- **Grouping** alerts (combining related alerts)
- **Routing** alerts to different receivers (Email, Slack, PagerDuty, etc.)
- **Silencing** alerts temporarily
- **Inhibiting** alerts (suppressing certain alerts when others are firing)

## Prerequisites

- Docker and Docker Compose installed
- Prometheus already configured (included in docker-compose.yml)
- Alert rules configured (prometheus/alert_rules.yml)

## Installation Steps

### 1. Create Alertmanager Configuration

Create a file `prometheus/alertmanager.yml`:

```yaml
# Alertmanager Configuration for Al-Hudhud SMS Gateway
global:
  # SMTP configuration for email alerts
  smtp_smarthost: 'smtp.gmail.com:587'
  smtp_from: 'alerts@alhudhud.com'
  smtp_auth_username: 'your-email@gmail.com'
  smtp_auth_password: 'your-app-password'
  smtp_require_tls: true

  # Slack webhook URL (optional)
  slack_api_url: 'https://hooks.slack.com/services/YOUR/WEBHOOK/URL'

# Templates for alert messages
templates:
  - '/etc/alertmanager/templates/*.tmpl'

# Route tree for alert routing
route:
  # Default receiver for all alerts
  receiver: 'default-receiver'
  
  # Group alerts by these labels
  group_by: ['alertname', 'component', 'severity']
  
  # Wait time before sending initial notification
  group_wait: 30s
  
  # Wait time before sending notification about new alerts in group
  group_interval: 5m
  
  # Wait time before re-sending notification
  repeat_interval: 4h

  # Child routes for specific alert types
  routes:
    # Critical alerts - immediate notification
    - match:
        severity: critical
      receiver: 'critical-alerts'
      group_wait: 10s
      repeat_interval: 1h
      continue: true

    # Financial alerts - notify finance team
    - match:
        component: financial
      receiver: 'finance-team'
      group_wait: 1m
      repeat_interval: 2h

    # Database alerts - notify database team
    - match:
        component: database
      receiver: 'database-team'
      group_wait: 2m
      repeat_interval: 3h

    # Performance alerts - notify operations team
    - match_re:
        component: (performance|queue|workers)
      receiver: 'operations-team'
      group_wait: 2m
      repeat_interval: 3h

# Inhibition rules - suppress certain alerts when others are firing
inhibit_rules:
  # Suppress warning alerts when critical alert is firing
  - source_match:
      severity: 'critical'
    target_match:
      severity: 'warning'
    equal: ['alertname', 'component']

  # Suppress low throughput when queue is high
  - source_match:
      alertname: 'HighQueueSize'
    target_match:
      alertname: 'LowMessageThroughput'

# Receivers - where to send alerts
receivers:
  # Default receiver - email to operations
  - name: 'default-receiver'
    email_configs:
      - to: 'operations@alhudhud.com'
        headers:
          Subject: '[Al-Hudhud] {{ .GroupLabels.alertname }}'
        html: |
          <h2>Alert: {{ .GroupLabels.alertname }}</h2>
          <p><strong>Severity:</strong> {{ .GroupLabels.severity }}</p>
          <p><strong>Component:</strong> {{ .GroupLabels.component }}</p>
          {{ range .Alerts }}
          <hr>
          <p><strong>Summary:</strong> {{ .Annotations.summary }}</p>
          <p><strong>Description:</strong> {{ .Annotations.description }}</p>
          <p><strong>Impact:</strong> {{ .Annotations.impact }}</p>
          <p><strong>Action:</strong> {{ .Annotations.action }}</p>
          <p><strong>Started:</strong> {{ .StartsAt }}</p>
          {{ end }}

  # Critical alerts - email + Slack + SMS
  - name: 'critical-alerts'
    email_configs:
      - to: 'oncall@alhudhud.com,cto@alhudhud.com'
        headers:
          Subject: '[CRITICAL] {{ .GroupLabels.alertname }}'
        send_resolved: true
    slack_configs:
      - channel: '#critical-alerts'
        title: 'CRITICAL: {{ .GroupLabels.alertname }}'
        text: |
          *Severity:* {{ .GroupLabels.severity }}
          *Component:* {{ .GroupLabels.component }}
          {{ range .Alerts }}
          *Summary:* {{ .Annotations.summary }}
          *Description:* {{ .Annotations.description }}
          *Action:* {{ .Annotations.action }}
          {{ end }}
        send_resolved: true

  # Finance team alerts
  - name: 'finance-team'
    email_configs:
      - to: 'finance@alhudhud.com'
        headers:
          Subject: '[Finance Alert] {{ .GroupLabels.alertname }}'

  # Database team alerts
  - name: 'database-team'
    email_configs:
      - to: 'database@alhudhud.com'
        headers:
          Subject: '[Database Alert] {{ .GroupLabels.alertname }}'

  # Operations team alerts
  - name: 'operations-team'
    email_configs:
      - to: 'operations@alhudhud.com'
        headers:
          Subject: '[Operations Alert] {{ .GroupLabels.alertname }}'
    slack_configs:
      - channel: '#operations'
        send_resolved: true
```

### 2. Update Docker Compose

Add Alertmanager service to `docker-compose.yml`:

```yaml
  alertmanager:
    image: prom/alertmanager:latest
    container_name: hudhud-alertmanager
    restart: always
    ports:
      - "9093:9093"
    volumes:
      - ./prometheus/alertmanager.yml:/etc/alertmanager/alertmanager.yml
      - alertmanager_data:/alertmanager
    command:
      - '--config.file=/etc/alertmanager/alertmanager.yml'
      - '--storage.path=/alertmanager'
      - '--web.external-url=http://localhost:9093'
```

Add volume for Alertmanager data:

```yaml
volumes:
  db_data:
  redis_data:
  prometheus_data:
  grafana_data:
  alertmanager_data:  # Add this line
```

### 3. Update Prometheus Configuration

Update `prometheus.yml` to connect to Alertmanager:

```yaml
# Alertmanager configuration
alerting:
  alertmanagers:
    - static_configs:
        - targets: ['alertmanager:9093']
```

### 4. Start Services

```bash
# Start all services including Alertmanager
docker-compose up -d

# Check Alertmanager logs
docker logs hudhud-alertmanager

# Check if Alertmanager is running
curl http://localhost:9093/-/healthy
```

## Configuration for Different Notification Channels

### Email (Gmail)

1. Enable 2-factor authentication on your Gmail account
2. Generate an App Password: https://myaccount.google.com/apppasswords
3. Update `alertmanager.yml`:

```yaml
global:
  smtp_smarthost: 'smtp.gmail.com:587'
  smtp_from: 'your-email@gmail.com'
  smtp_auth_username: 'your-email@gmail.com'
  smtp_auth_password: 'your-app-password'
  smtp_require_tls: true
```

### Slack

1. Create a Slack Incoming Webhook:
   - Go to https://api.slack.com/apps
   - Create new app → Incoming Webhooks
   - Activate and create webhook for your channel
   - Copy webhook URL

2. Update `alertmanager.yml`:

```yaml
global:
  slack_api_url: 'https://hooks.slack.com/services/YOUR/WEBHOOK/URL'

receivers:
  - name: 'slack-alerts'
    slack_configs:
      - channel: '#alerts'
        title: '{{ .GroupLabels.alertname }}'
        text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'
```

### PagerDuty

1. Get your PagerDuty Integration Key
2. Add to `alertmanager.yml`:

```yaml
receivers:
  - name: 'pagerduty-critical'
    pagerduty_configs:
      - service_key: 'YOUR_PAGERDUTY_SERVICE_KEY'
        description: '{{ .GroupLabels.alertname }}'
```

### Telegram

1. Create a Telegram bot and get bot token
2. Get your chat ID
3. Add to `alertmanager.yml`:

```yaml
receivers:
  - name: 'telegram-alerts'
    telegram_configs:
      - bot_token: 'YOUR_BOT_TOKEN'
        chat_id: YOUR_CHAT_ID
        message: |
          {{ range .Alerts }}
          *Alert:* {{ .Annotations.summary }}
          *Description:* {{ .Annotations.description }}
          {{ end }}
```

## Testing Alerts

### 1. Test Alert Rules

Check if alert rules are loaded:

```bash
# Check Prometheus rules
curl http://localhost:9090/api/v1/rules | jq

# Check active alerts
curl http://localhost:9090/api/v1/alerts | jq
```

### 2. Trigger Test Alert

Create a test alert by simulating high queue size:

```bash
# In Prometheus console (http://localhost:9090)
# Execute query: hudhud_queue_size
# If value > 1000, alert should fire after 2 minutes
```

### 3. Check Alertmanager

```bash
# View active alerts in Alertmanager
curl http://localhost:9093/api/v1/alerts | jq

# View Alertmanager status
curl http://localhost:9093/api/v1/status | jq
```

### 4. Silence Alerts (Temporary)

```bash
# Silence an alert for 2 hours
curl -X POST http://localhost:9093/api/v1/silences \
  -H "Content-Type: application/json" \
  -d '{
    "matchers": [
      {
        "name": "alertname",
        "value": "HighQueueSize",
        "isRegex": false
      }
    ],
    "startsAt": "2024-01-01T00:00:00Z",
    "endsAt": "2024-01-01T02:00:00Z",
    "createdBy": "admin",
    "comment": "Maintenance window"
  }'
```

## Alert Severity Levels

### Warning (severity: warning)
- **Response Time:** Within 30 minutes
- **Examples:** High queue size, degraded success rate, low cache hit rate
- **Action:** Monitor and investigate during business hours

### Critical (severity: critical)
- **Response Time:** Immediate (within 5 minutes)
- **Examples:** Very low success rate, critical queue size, system failures
- **Action:** Immediate investigation and resolution required

## Alert Response Procedures

### High Queue Size Alert
1. Check worker pool status: `docker logs hudhud-backend`
2. Check database performance: `docker exec -it hudhud-db psql -U hudhud_admin -d al_hudhud -c "SELECT * FROM pg_stat_activity;"`
3. Check Redis health: `docker exec -it hudhud-redis redis-cli INFO`
4. Consider scaling workers if needed

### High Latency Alert
1. Check database slow queries
2. Check Redis cache hit rate
3. Check network latency
4. Review recent code changes

### Low Success Rate Alert
1. Check channel availability
2. Check network connectivity
3. Review error logs for common failures
4. Check mobile gateway status

### Low Balance Alert
1. Identify affected tenant
2. Send notification to tenant
3. Check if auto-recharge is configured
4. Monitor for service interruption

## Monitoring Alertmanager

### Health Check

```bash
# Check if Alertmanager is healthy
curl http://localhost:9093/-/healthy

# Check Alertmanager metrics
curl http://localhost:9093/metrics
```

### Web UI

Access Alertmanager web interface:
- URL: http://localhost:9093
- Features:
  - View active alerts
  - Create silences
  - View alert history
  - Manage alert routing

## Troubleshooting

### Alerts Not Firing

1. Check if alert rules are loaded:
   ```bash
   curl http://localhost:9090/api/v1/rules | jq
   ```

2. Check if metrics are being collected:
   ```bash
   curl http://localhost:9090/api/v1/query?query=hudhud_queue_size
   ```

3. Check Prometheus logs:
   ```bash
   docker logs hudhud-prometheus
   ```

### Alerts Not Being Sent

1. Check Alertmanager configuration:
   ```bash
   docker exec hudhud-alertmanager amtool check-config /etc/alertmanager/alertmanager.yml
   ```

2. Check Alertmanager logs:
   ```bash
   docker logs hudhud-alertmanager
   ```

3. Test email configuration:
   ```bash
   docker exec hudhud-alertmanager amtool alert add test_alert severity=warning
   ```

### Email Not Sending

1. Verify SMTP credentials
2. Check if 2FA is enabled and app password is used
3. Check firewall rules for SMTP port (587)
4. Review Alertmanager logs for SMTP errors

## Best Practices

1. **Alert Fatigue Prevention:**
   - Set appropriate thresholds
   - Use inhibition rules to suppress redundant alerts
   - Group related alerts together
   - Set reasonable repeat intervals

2. **Alert Routing:**
   - Route critical alerts to on-call team
   - Route component-specific alerts to relevant teams
   - Use multiple notification channels for critical alerts

3. **Alert Documentation:**
   - Include clear descriptions in annotations
   - Provide actionable remediation steps
   - Document impact of each alert

4. **Testing:**
   - Regularly test alert rules
   - Verify notification channels work
   - Practice alert response procedures

5. **Maintenance:**
   - Review and update alert thresholds regularly
   - Remove obsolete alerts
   - Update contact information

## Related Requirements

- **Requirement 16.5:** Alert when queue_size > 1000
- **Requirement 16.6:** Alert when average_latency > 2 seconds
- **Requirement 16.7:** Critical alert when success_rate < 95%

## Additional Resources

- [Prometheus Alerting Documentation](https://prometheus.io/docs/alerting/latest/overview/)
- [Alertmanager Configuration](https://prometheus.io/docs/alerting/latest/configuration/)
- [Alert Rule Best Practices](https://prometheus.io/docs/practices/alerting/)

## Support

For issues or questions:
- Check logs: `docker logs hudhud-alertmanager`
- Review configuration: `docker exec hudhud-alertmanager cat /etc/alertmanager/alertmanager.yml`
- Contact: operations@alhudhud.com
