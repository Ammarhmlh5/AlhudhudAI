# Alerting Quick Start Guide

**Task 35.6: Alerting Setup**

## 5-Minute Setup

### 1. Configure Email (Required)

Edit `prometheus/alertmanager.yml`:

```yaml
global:
  smtp_smarthost: 'smtp.gmail.com:587'
  smtp_from: 'your-email@gmail.com'
  smtp_auth_username: 'your-email@gmail.com'
  smtp_auth_password: 'your-app-password'  # Use App Password for Gmail
```

**Gmail App Password Setup:**
1. Go to https://myaccount.google.com/security
2. Enable 2-Step Verification
3. Go to App Passwords
4. Generate password for "Mail"
5. Use generated password in config

### 2. Configure Slack (Optional)

**Get Webhook URL:**
1. Go to https://api.slack.com/apps
2. Create app or select existing
3. Enable "Incoming Webhooks"
4. Add webhook to workspace
5. Copy webhook URL

**Set Environment Variable:**

Windows PowerShell:
```powershell
$env:SLACK_WEBHOOK_URL="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
```

Linux/Mac:
```bash
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
```

**Update docker-compose.monitoring.yml:**
```yaml
alertmanager:
  environment:
    - SLACK_WEBHOOK_URL=${SLACK_WEBHOOK_URL}
```

### 3. Start Monitoring Stack

```powershell
# Start all monitoring services
./start-monitoring.ps1
```

This starts:
- Prometheus (port 9090)
- Alertmanager (port 9093)
- Grafana (port 3001)

### 4. Verify Setup

**Check Services:**
```powershell
# Prometheus
curl http://localhost:9090/-/healthy

# Alertmanager
curl http://localhost:9093/-/healthy

# View alerts
curl http://localhost:9090/api/v1/alerts
```

**Access Web UIs:**
- Prometheus: http://localhost:9090/alerts
- Alertmanager: http://localhost:9093/#/alerts
- Grafana: http://localhost:3001

### 5. Test Alerts

**Send Test Alert:**
```powershell
curl -X POST http://localhost:9093/api/v1/alerts `
  -H 'Content-Type: application/json' `
  -d '[{
    "labels": {
      "alertname": "TestAlert",
      "severity": "warning",
      "component": "test"
    },
    "annotations": {
      "summary": "Test alert",
      "description": "Testing alert delivery",
      "impact": "No impact",
      "action": "No action required"
    }
  }]'
```

**Check Email and Slack:**
- Email should arrive within 30 seconds
- Slack notification should appear immediately

### 6. Trigger Real Alerts

```powershell
# Run monitoring tests
./test-monitoring.ps1
```

This generates:
- High error rates
- Slow queries
- Resource usage spikes
- Balance failures

## Alert Channels

### Email Recipients

- `critical-alerts@alhudhud.com` - Critical system issues
- `finance@alhudhud.com` - Financial alerts
- `enterprise-support@alhudhud.com` - Gateway alerts
- `operations@alhudhud.com` - Subscription alerts
- `database@alhudhud.com` - Database alerts

### Slack Channels

- `#critical-alerts` - Critical issues
- `#finance-alerts` - Financial alerts
- `#enterprise-support` - Gateway alerts
- `#operations` - Operational alerts
- `#database-alerts` - Database alerts

## Common Alert Types

### Critical Alerts (Immediate Action)

- **CriticalSubscriptionRenewalFailures**: Mass renewal failures
- **CriticalGatewayResourceUsage**: Gateway at 95% capacity
- **BalanceDeductionFailures**: Financial integrity issue
- **CriticalTariffSystemErrorRate**: System instability

### Warning Alerts (Monitor & Plan)

- **HighSubscriptionRenewalFailures**: Elevated failure rate
- **HighGatewayResourceUsage**: Gateway at 80% capacity
- **SlowTariffQueries**: Database performance degraded
- **LowTariffCacheHitRate**: Cache effectiveness reduced

### Info Alerts (Awareness)

- **RolloverMessagesExpiringSoon**: Customer notifications needed
- **HighOverageUsage**: Package upgrade opportunities

## Quick Actions

### Silence an Alert

```powershell
# Via Alertmanager UI
# 1. Go to http://localhost:9093/#/silences
# 2. Click "New Silence"
# 3. Add matcher: alertname=HighGatewayResourceUsage
# 4. Set duration and comment
# 5. Click "Create"
```

### View Active Alerts

```powershell
# Via API
curl http://localhost:9093/api/v1/alerts

# Via UI
# Go to http://localhost:9093/#/alerts
```

### Check Alert Rules

```powershell
# Via API
curl http://localhost:9090/api/v1/rules

# Via UI
# Go to http://localhost:9090/rules
```

## Troubleshooting

### Alerts Not Firing

1. Check Prometheus is scraping:
   ```powershell
   curl http://localhost:9090/api/v1/targets
   ```

2. Verify metrics exist:
   ```powershell
   curl http://localhost:9090/api/v1/query?query=tariff_subscription_renewal_failures_total
   ```

3. Check alert rules loaded:
   ```powershell
   curl http://localhost:9090/api/v1/rules
   ```

### Email Not Delivering

1. Check SMTP credentials in `alertmanager.yml`
2. Verify Gmail App Password (not regular password)
3. Check Alertmanager logs:
   ```powershell
   docker logs alertmanager
   ```

### Slack Not Working

1. Verify webhook URL is set:
   ```powershell
   echo $env:SLACK_WEBHOOK_URL
   ```

2. Test webhook directly:
   ```powershell
   curl -X POST $env:SLACK_WEBHOOK_URL `
     -H 'Content-Type: application/json' `
     -d '{"text":"Test"}'
   ```

3. Check Alertmanager logs:
   ```powershell
   docker logs alertmanager | Select-String "slack"
   ```

## Configuration Files

- **Alert Rules**: `prometheus/tariff_alert_rules.yml`
- **Alertmanager Config**: `prometheus/alertmanager.yml`
- **Email Templates**: `prometheus/templates/email.tmpl`
- **Docker Compose**: `docker-compose.monitoring.yml`

## Next Steps

1. ✅ Customize email recipients
2. ✅ Set up Slack channels
3. ✅ Adjust alert thresholds
4. ✅ Create runbooks for alerts
5. ✅ Train team on alert response
6. ✅ Set up on-call rotation

## Full Documentation

See [ALERTING_SETUP_GUIDE.md](./ALERTING_SETUP_GUIDE.md) for complete documentation.

## Support

- Prometheus Docs: https://prometheus.io/docs/alerting/
- Alertmanager Docs: https://prometheus.io/docs/alerting/alertmanager/
- Slack API: https://api.slack.com/messaging/webhooks
