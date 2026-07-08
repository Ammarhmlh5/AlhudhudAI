# Alerting Reference Card

Quick reference for Tariff System alerting.

## 🚀 Quick Start

```powershell
# 1. Configure email
# Edit prometheus/alertmanager.yml - set SMTP credentials

# 2. Configure Slack (optional)
$env:SLACK_WEBHOOK_URL="https://hooks.slack.com/services/YOUR/WEBHOOK"

# 3. Start monitoring
./start-monitoring.ps1

# 4. Test alerting
./test-alerting.ps1
```

## 🔗 Web UIs

- **Prometheus**: http://localhost:9090/alerts
- **Alertmanager**: http://localhost:9093/#/alerts
- **Grafana**: http://localhost:3001

## 📧 Email Recipients

| Team | Email | Alerts |
|------|-------|--------|
| Critical | critical-alerts@alhudhud.com | All critical issues |
| Finance | finance@alhudhud.com | Financial alerts |
| Enterprise | enterprise-support@alhudhud.com | Gateway alerts |
| Operations | operations@alhudhud.com | Subscription alerts |
| Database | database@alhudhud.com | Database alerts |

## 💬 Slack Channels

| Channel | Purpose |
|---------|---------|
| #critical-alerts | Critical system issues |
| #finance-alerts | Financial alerts |
| #enterprise-support | Gateway alerts |
| #operations | Operational alerts |
| #database-alerts | Database alerts |

## 🚨 Alert Severity

| Level | Color | Response Time | Examples |
|-------|-------|---------------|----------|
| Critical | 🔴 Red | Immediate | System down, data loss risk |
| Warning | 🟡 Yellow | Within 1 hour | High load, degraded performance |
| Info | 🔵 Blue | Next business day | Trends, recommendations |

## 📊 Key Alert Rules

### Critical Alerts
- **CriticalSubscriptionRenewalFailures**: >20 failures/sec
- **CriticalGatewayResourceUsage**: >95% resource usage
- **BalanceDeductionFailures**: >1 failure/sec
- **CriticalTariffSystemErrorRate**: >0.2 errors/sec

### Warning Alerts
- **HighSubscriptionRenewalFailures**: >5 failures/sec
- **HighGatewayResourceUsage**: >80% resource usage
- **SlowTariffQueries**: p95 >0.5s
- **LowTariffCacheHitRate**: <80%

## 🔧 Common Commands

### View Alerts
```powershell
# Active alerts
curl http://localhost:9090/api/v1/alerts

# Alertmanager alerts
curl http://localhost:9093/api/v1/alerts
```

### Send Test Alert
```powershell
curl -X POST http://localhost:9093/api/v1/alerts `
  -H 'Content-Type: application/json' `
  -d '[{"labels":{"alertname":"Test","severity":"info"},"annotations":{"summary":"Test"}}]'
```

### Silence Alert
```powershell
# Via UI: http://localhost:9093/#/silences
# Click "New Silence" → Add matcher → Set duration
```

## 🐛 Troubleshooting

### Alerts Not Firing
1. Check Prometheus: `curl http://localhost:9090/-/healthy`
2. Check metrics: `curl http://localhost:9090/api/v1/query?query=up`
3. Check rules: `curl http://localhost:9090/api/v1/rules`

### Email Not Working
1. Verify SMTP config in `alertmanager.yml`
2. Check logs: `docker logs alertmanager`
3. Test SMTP: `telnet smtp.gmail.com 587`

### Slack Not Working
1. Check webhook: `echo $env:SLACK_WEBHOOK_URL`
2. Test webhook: `curl -X POST $env:SLACK_WEBHOOK_URL -d '{"text":"Test"}'`
3. Check logs: `docker logs alertmanager | Select-String slack`

## 📚 Documentation

- **Quick Start**: [ALERTING_QUICK_START.md](./ALERTING_QUICK_START.md)
- **Full Guide**: [ALERTING_SETUP_GUIDE.md](./ALERTING_SETUP_GUIDE.md)
- **Alert Rules**: [tariff_alert_rules.yml](./tariff_alert_rules.yml)
- **Alertmanager**: [alertmanager.yml](./alertmanager.yml)

## 🎯 Response Playbook

### Critical Alert Response
1. Acknowledge alert in Alertmanager
2. Check Grafana dashboards for context
3. Review Prometheus metrics
4. Check application logs
5. Take corrective action
6. Document in incident log
7. Update runbook if needed

### Warning Alert Response
1. Review alert details
2. Check trends in Grafana
3. Assess impact
4. Plan corrective action
5. Schedule maintenance if needed

## 🔄 Regular Maintenance

- **Daily**: Review active alerts
- **Weekly**: Check alert trends
- **Monthly**: Review and tune thresholds
- **Quarterly**: Update runbooks

## 📞 Escalation

1. **Level 1**: On-call engineer
2. **Level 2**: Team lead
3. **Level 3**: Engineering manager
4. **Level 4**: CTO

## ⚙️ Configuration Files

```
prometheus/
├── alertmanager.yml          # Alertmanager config
├── tariff_alert_rules.yml    # Tariff alert rules
├── alert_rules.yml           # General alert rules
└── templates/
    └── email.tmpl            # Email template
```

## 🧪 Testing

```powershell
# Run all tests
./test-alerting.ps1

# Test specific component
curl http://localhost:9090/-/healthy  # Prometheus
curl http://localhost:9093/-/healthy  # Alertmanager
```

## 💡 Tips

- Use silences for planned maintenance
- Group related alerts to reduce noise
- Keep alert messages actionable
- Include runbook links in alerts
- Test alert delivery regularly
- Review and tune thresholds monthly

## 🆘 Emergency Contacts

- **DevOps Team**: devops@alhudhud.com
- **On-Call**: +1-XXX-XXX-XXXX
- **Slack**: #incident-response

---

**Last Updated**: Task 35.6 Completion  
**Version**: 1.0
