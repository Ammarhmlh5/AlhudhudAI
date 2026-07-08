# Alerting Setup Guide - Tariff Pricing System

**Task 35.6: Alerting Setup**  
**Requirements: 24.1, 24.2, 24.3, 24.4, 24.5, 24.6, 24.7, 24.9**

## Overview

This guide covers the complete alerting setup for the Tariff Pricing System, including:
- Prometheus alert rules configuration
- Alertmanager setup with email and Slack integration
- Alert routing and grouping strategies
- Notification templates
- Testing and validation

## Architecture

```
┌─────────────────┐
│   Prometheus    │
│  (Alert Rules)  │
└────────┬────────┘
         │ Evaluates rules every 30s
         │ Fires alerts when conditions met
         ▼
┌─────────────────┐
│  Alertmanager   │
│  (Routing &     │
│   Grouping)     │
└────────┬────────┘
         │
         ├──────────────┬──────────────┬──────────────┐
         ▼              ▼              ▼              ▼
    ┌────────┐    ┌────────┐    ┌────────┐    ┌────────┐
    │ Email  │    │ Slack  │    │PagerDuty│   │Webhook │
    └────────┘    └────────┘    └────────┘    └────────┘
```

## Alert Categories

### 1. Subscription Alerts (Requirements 24.1, 24.2)
- **HighSubscriptionRenewalFailures**: Warning when renewal failures > 5/sec
- **CriticalSubscriptionRenewalFailures**: Critical when > 20/sec
- **ExpiringSubscriptionsNotRenewed**: Warning when > 100 subscriptions expiring soon

### 2. Rollover Alerts (Requirement 24.3)
- **HighRolloverForfeitures**: Warning when > 50 forfeitures/hour
- **RolloverMessagesExpiringSoon**: Info when > 1000 messages expiring

### 3. Gateway Performance Alerts (Requirement 24.4)
- **HighGatewayResourceUsage**: Warning at 80% resource usage
- **CriticalGatewayResourceUsage**: Critical at 95% resource usage
- **GatewayQuotaExceeded**: Warning when quota exceeded

### 4. Tariff Calculation Alerts (Requirement 24.5)
- **HighTariffCalculationErrors**: Warning when > 1 error/sec
- **MissingTariffDefinitions**: Critical when tariffs missing

### 5. Balance Alerts (Requirement 24.6)
- **HighInsufficientBalanceRate**: Warning when > 10 attempts/sec
- **BalanceDeductionFailures**: Critical when > 1 failure/sec

### 6. Database Performance Alerts (Requirement 24.7)
- **SlowTariffQueries**: Warning when p95 > 0.5s
- **CriticalTariffQueryPerformance**: Critical when p95 > 2s

### 7. System Health Alerts (Requirement 24.9)
- **HighTariffSystemErrorRate**: Warning when > 0.05 errors/sec
- **CriticalTariffSystemErrorRate**: Critical when > 0.2 errors/sec
- **SlowTariffAPIResponses**: Warning when p95 > 1s

## Configuration Files

### 1. Alert Rules (`prometheus/tariff_alert_rules.yml`)

Alert rules define the conditions that trigger alerts:

```yaml
groups:
  - name: tariff_subscriptions
    interval: 30s
    rules:
      - alert: HighSubscriptionRenewalFailures
        expr: rate(tariff_subscription_renewal_failures_total[5m]) > 5
        for: 5m
        labels:
          severity: warning
          component: subscriptions
        annotations:
          summary: "High subscription renewal failure rate"
          description: "Rate is {{ $value }} failures/second"
          impact: "Customers losing service access"
          action: "Check balance deduction system"
```

**Key Fields:**
- `expr`: PromQL expression that triggers the alert
- `for`: Duration condition must be true before firing
- `labels`: Metadata for routing and grouping
- `annotations`: Human-readable information

### 2. Alertmanager Configuration (`prometheus/alertmanager.yml`)

Alertmanager handles alert routing, grouping, and notifications:

```yaml
route:
  receiver: 'default'
  group_by: ['alertname', 'component', 'severity']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
  
  routes:
    - match:
        severity: critical
      receiver: 'critical-alerts'
      group_wait: 10s
      repeat_interval: 30m
```

**Routing Strategy:**
- Group alerts by name, component, and severity
- Critical alerts get faster notification (10s vs 30s)
- Critical alerts repeat more frequently (30m vs 4h)
- Different teams get different alert types

## Email Configuration

### SMTP Setup

Configure in `alertmanager.yml`:

```yaml
global:
  smtp_smarthost: 'smtp.gmail.com:587'
  smtp_from: 'alerts@alhudhud.com'
  smtp_auth_username: 'alerts@alhudhud.com'
  smtp_auth_password: 'your-smtp-password'
  smtp_require_tls: true
```

### Email Templates

Custom HTML templates in `prometheus/templates/email.tmpl`:

- Color-coded by severity (red=critical, orange=warning, blue=info)
- Includes all alert details (summary, description, impact, action)
- Shows alert labels and timestamps
- Responsive design for mobile viewing

### Email Receivers

```yaml
receivers:
  - name: 'critical-alerts'
    email_configs:
      - to: 'critical-alerts@alhudhud.com'
        headers:
          Subject: '[CRITICAL] {{ .GroupLabels.alertname }}'
        send_resolved: true
```

## Slack Integration

### Setup Steps

1. **Create Slack Incoming Webhook:**
   - Go to https://api.slack.com/apps
   - Create new app or select existing
   - Enable "Incoming Webhooks"
   - Add webhook to workspace
   - Copy webhook URL

2. **Configure Environment Variable:**
   ```bash
   export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
   ```

3. **Update docker-compose.monitoring.yml:**
   ```yaml
   alertmanager:
     environment:
       - SLACK_WEBHOOK_URL=${SLACK_WEBHOOK_URL}
   ```

### Slack Channels

Different alert types go to different channels:

- `#critical-alerts`: Critical system issues
- `#finance-alerts`: Financial and billing alerts
- `#enterprise-support`: Gateway and enterprise customer alerts
- `#operations`: Subscription and operational alerts
- `#database-alerts`: Database performance issues

### Slack Message Format

```yaml
slack_configs:
  - api_url: '${SLACK_WEBHOOK_URL}'
    channel: '#critical-alerts'
    username: 'Tariff System Alerts'
    icon_emoji: ':rotating_light:'
    title: ':rotating_light: CRITICAL: {{ .GroupLabels.alertname }}'
    text: |
      *Summary:* {{ .Annotations.summary }}
      *Description:* {{ .Annotations.description }}
      *Impact:* {{ .Annotations.impact }}
      *Action Required:* {{ .Annotations.action }}
    send_resolved: true
    color: '{{ if eq .Status "firing" }}danger{{ else }}good{{ end }}'
```

**Features:**
- Custom username and emoji per alert type
- Color-coded (red=firing, green=resolved)
- Includes all critical information
- Sends resolution notifications

## Alert Routing Strategy

### Routing Tree

```
Root (default receiver)
├── Critical Alerts (10s wait, 30m repeat)
│   ├── Email: critical-alerts@alhudhud.com
│   └── Slack: #critical-alerts
├── Financial Alerts (1m wait, 2h repeat)
│   ├── Email: finance@alhudhud.com
│   └── Slack: #finance-alerts
├── Gateway Alerts (30s wait, 1h repeat)
│   ├── Email: enterprise-support@alhudhud.com
│   └── Slack: #enterprise-support
├── Subscription Alerts (1m wait, 2h repeat)
│   ├── Email: operations@alhudhud.com
│   └── Slack: #operations
└── Database Alerts (30s wait, 1h repeat)
    ├── Email: database@alhudhud.com
    └── Slack: #database-alerts
```

### Grouping Strategy

Alerts are grouped by:
1. **alertname**: Same alert type grouped together
2. **component**: Same system component grouped together
3. **severity**: Same severity level grouped together

**Benefits:**
- Reduces notification spam
- Provides context by grouping related alerts
- Easier to identify systemic issues

### Inhibition Rules

Suppress less important alerts when more critical ones are firing:

```yaml
inhibit_rules:
  # Suppress warnings when critical alert firing
  - source_match:
      severity: 'critical'
    target_match:
      severity: 'warning'
    equal: ['alertname', 'component']
  
  # Suppress specific alerts when system down
  - source_match:
      alertname: 'CriticalTariffSystemErrorRate'
    target_match_re:
      alertname: '(SlowTariffQueries|LowTariffCacheHitRate)'
```

## Deployment

### 1. Start Monitoring Stack

```bash
# Start Prometheus, Alertmanager, and Grafana
./start-monitoring.ps1
```

### 2. Verify Alertmanager

```bash
# Check Alertmanager is running
curl http://localhost:9093/-/healthy

# View current configuration
curl http://localhost:9093/api/v1/status
```

### 3. Verify Alert Rules

```bash
# Check Prometheus loaded alert rules
curl http://localhost:9090/api/v1/rules

# View active alerts
curl http://localhost:9090/api/v1/alerts
```

### 4. Access Web UIs

- **Prometheus**: http://localhost:9090
  - View alerts: http://localhost:9090/alerts
  - View rules: http://localhost:9090/rules

- **Alertmanager**: http://localhost:9093
  - View alerts: http://localhost:9093/#/alerts
  - Silence alerts: http://localhost:9093/#/silences

- **Grafana**: http://localhost:3001
  - View dashboards with alert annotations

## Testing Alerts

### 1. Test Email Configuration

```bash
# Send test alert via Alertmanager API
curl -X POST http://localhost:9093/api/v1/alerts \
  -H 'Content-Type: application/json' \
  -d '[{
    "labels": {
      "alertname": "TestAlert",
      "severity": "warning",
      "component": "test"
    },
    "annotations": {
      "summary": "Test alert for email configuration",
      "description": "This is a test alert to verify email delivery",
      "impact": "No impact - this is a test",
      "action": "No action required - this is a test"
    }
  }]'
```

### 2. Test Slack Integration

```bash
# Send test notification to Slack
curl -X POST ${SLACK_WEBHOOK_URL} \
  -H 'Content-Type: application/json' \
  -d '{
    "text": "Test notification from Tariff System Alerting",
    "username": "Tariff System Alerts",
    "icon_emoji": ":test_tube:"
  }'
```

### 3. Trigger Real Alerts

```bash
# Run test script to trigger various alerts
./test-monitoring.ps1
```

This will:
- Generate high error rates
- Create slow queries
- Simulate balance failures
- Trigger resource usage alerts

### 4. Verify Alert Delivery

Check that alerts are received:
- ✅ Email inbox for alert emails
- ✅ Slack channels for notifications
- ✅ Alertmanager UI shows active alerts
- ✅ Prometheus UI shows firing alerts

## Alert Silencing

### When to Silence Alerts

- Planned maintenance windows
- Known issues being worked on
- False positives being investigated

### How to Silence

**Via Alertmanager UI:**
1. Go to http://localhost:9093/#/silences
2. Click "New Silence"
3. Add matchers (e.g., `alertname=HighGatewayResourceUsage`)
4. Set duration
5. Add comment explaining reason
6. Click "Create"

**Via API:**
```bash
curl -X POST http://localhost:9093/api/v1/silences \
  -H 'Content-Type: application/json' \
  -d '{
    "matchers": [
      {
        "name": "alertname",
        "value": "HighGatewayResourceUsage",
        "isRegex": false
      }
    ],
    "startsAt": "2024-01-01T00:00:00Z",
    "endsAt": "2024-01-01T02:00:00Z",
    "createdBy": "admin",
    "comment": "Planned maintenance window"
  }'
```

## Troubleshooting

### Alerts Not Firing

1. **Check Prometheus is scraping metrics:**
   ```bash
   curl http://localhost:9090/api/v1/targets
   ```

2. **Verify alert rules are loaded:**
   ```bash
   curl http://localhost:9090/api/v1/rules
   ```

3. **Check alert evaluation:**
   - Go to http://localhost:9090/alerts
   - Look for "Pending" or "Firing" status
   - Check "Active Since" timestamp

4. **Test PromQL expression:**
   - Go to http://localhost:9090/graph
   - Enter alert expression
   - Verify it returns data

### Alerts Not Sending Notifications

1. **Check Alertmanager is receiving alerts:**
   ```bash
   curl http://localhost:9093/api/v1/alerts
   ```

2. **Verify routing configuration:**
   ```bash
   curl http://localhost:9093/api/v1/status
   ```

3. **Check Alertmanager logs:**
   ```bash
   docker logs alertmanager
   ```

4. **Test receiver configuration:**
   - Send test alert via API
   - Check receiver logs (email server, Slack)

### Email Not Delivering

1. **Verify SMTP configuration:**
   - Check credentials are correct
   - Verify SMTP server allows connections
   - Check firewall rules

2. **Test SMTP connection:**
   ```bash
   telnet smtp.gmail.com 587
   ```

3. **Check email logs:**
   ```bash
   docker logs alertmanager | grep -i email
   ```

4. **Common issues:**
   - Gmail: Enable "Less secure app access" or use App Password
   - Office 365: Use modern authentication
   - Corporate SMTP: Check firewall and proxy settings

### Slack Not Receiving Notifications

1. **Verify webhook URL is correct:**
   ```bash
   echo $SLACK_WEBHOOK_URL
   ```

2. **Test webhook directly:**
   ```bash
   curl -X POST $SLACK_WEBHOOK_URL \
     -H 'Content-Type: application/json' \
     -d '{"text":"Test message"}'
   ```

3. **Check Alertmanager logs:**
   ```bash
   docker logs alertmanager | grep -i slack
   ```

4. **Common issues:**
   - Webhook URL not set in environment
   - Webhook URL expired or revoked
   - Slack app not installed in workspace
   - Channel doesn't exist or bot not invited

## Best Practices

### 1. Alert Fatigue Prevention

- **Set appropriate thresholds**: Not too sensitive, not too lenient
- **Use `for` duration**: Avoid alerts for transient issues
- **Group related alerts**: Reduce notification spam
- **Implement inhibition rules**: Suppress redundant alerts
- **Regular review**: Adjust thresholds based on experience

### 2. Alert Quality

- **Clear summary**: What is the problem?
- **Detailed description**: Why is it happening?
- **Impact statement**: What's affected?
- **Action items**: What should be done?
- **Severity levels**: Critical, Warning, Info

### 3. On-Call Practices

- **Critical alerts only**: Wake people for real emergencies
- **Runbooks**: Link to resolution procedures
- **Escalation**: Define escalation paths
- **Post-mortems**: Learn from incidents
- **Regular testing**: Ensure alerts work when needed

### 4. Maintenance

- **Regular review**: Monthly alert review meetings
- **Tune thresholds**: Adjust based on system behavior
- **Update runbooks**: Keep resolution steps current
- **Test regularly**: Verify alert delivery works
- **Document changes**: Track alert rule modifications

## Monitoring the Monitors

### Alertmanager Health

```yaml
- alert: AlertmanagerDown
  expr: up{job="alertmanager"} == 0
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "Alertmanager is down"
    action: "Restart Alertmanager immediately"
```

### Prometheus Health

```yaml
- alert: PrometheusDown
  expr: up{job="prometheus"} == 0
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "Prometheus is down"
    action: "Restart Prometheus immediately"
```

### Alert Rule Evaluation

```yaml
- alert: AlertRuleEvaluationSlow
  expr: prometheus_rule_group_last_duration_seconds > 60
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "Alert rule evaluation is slow"
    action: "Check Prometheus performance"
```

## Integration with Grafana

Grafana dashboards can show alert states:

1. **Alert annotations**: Show when alerts fired on graphs
2. **Alert list panel**: Display current alerts
3. **Alert history**: Track alert patterns over time

Configure in Grafana:
- Add Alertmanager as data source
- Enable alert annotations on dashboards
- Create alert list panels

## Requirements Mapping

| Requirement | Alert Rules | Coverage |
|-------------|-------------|----------|
| 24.1 | Gateway failure alerts | ✅ |
| 24.2 | Subscription renewal alerts | ✅ |
| 24.3 | Resource usage alerts | ✅ |
| 24.4 | Performance degradation alerts | ✅ |
| 24.5 | Financial anomaly alerts | ✅ |
| 24.6 | Balance alerts | ✅ |
| 24.7 | Database performance alerts | ✅ |
| 24.9 | Notification channels (Email, Slack) | ✅ |

## Next Steps

1. ✅ Configure SMTP credentials
2. ✅ Set up Slack webhooks
3. ✅ Test alert delivery
4. ✅ Create runbooks for common alerts
5. ✅ Train team on alert response
6. ✅ Set up on-call rotation
7. ✅ Schedule regular alert reviews

## Related Documentation

- [Monitoring Setup Guide](./MONITORING_SETUP_GUIDE.md)
- [Prometheus Configuration](./prometheus-tariff.yml)
- [Alert Rules](./tariff_alert_rules.yml)
- [Alertmanager Configuration](./alertmanager.yml)
- [Grafana Dashboards](../grafana/dashboards/)

## Support

For issues or questions:
- Check Prometheus documentation: https://prometheus.io/docs/alerting/
- Check Alertmanager documentation: https://prometheus.io/docs/alerting/alertmanager/
- Review logs: `docker logs alertmanager`
- Contact DevOps team: devops@alhudhud.com
