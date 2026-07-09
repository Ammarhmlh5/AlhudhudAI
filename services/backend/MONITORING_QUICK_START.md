# Monitoring Quick Start - Tariff Pricing System

## Task 35.5: Monitoring Setup

## 🚀 Quick Start (5 Minutes)

### 1. Start Monitoring Stack

```bash
# Start all monitoring services
docker-compose -f docker-compose.monitoring.yml up -d

# Wait for services to be ready (30 seconds)
sleep 30

# Check status
docker-compose -f docker-compose.monitoring.yml ps
```

### 2. Access Dashboards

Open in your browser:

- **Grafana:** http://localhost:3001
  - Username: `admin`
  - Password: `admin123`
  
- **Prometheus:** http://localhost:9090

- **Alertmanager:** http://localhost:9093

### 3. View Tariff System Dashboard

1. Go to http://localhost:3001
2. Login with admin/admin123
3. Navigate to: Dashboards → Tariff Pricing System Dashboard
4. You should see real-time metrics!

## 📊 Key Metrics to Watch

### Subscriptions
- Total active subscriptions
- Renewal success rate
- Expiring subscriptions

### Rollover
- Active rollover messages
- Forfeiture rate
- Expiration warnings

### Gateways
- Resource usage per gateway
- Message throughput
- Quota status

### Performance
- Query duration (p95, p99)
- Cache hit rate
- API response time

## 🔔 Alert Status

Check active alerts:

```bash
# View all alerts
curl http://localhost:9090/api/v1/alerts | jq '.data.alerts[] | {name: .labels.alertname, state: .state}'

# View firing alerts only
curl http://localhost:9090/api/v1/alerts | jq '.data.alerts[] | select(.state == "firing") | {name: .labels.alertname}'
```

## 🧪 Test Metrics Collection

```bash
# Check if backend is exposing metrics
curl http://localhost:8080/metrics | grep tariff

# Query specific metric
curl 'http://localhost:9090/api/v1/query?query=tariff_subscriptions_total' | jq

# Check all targets health
curl http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | {job: .labels.job, health: .health}'
```

## 📈 Common Queries

### Subscription Metrics

```promql
# Total subscriptions
sum(tariff_subscriptions_total)

# Subscriptions by status
sum by (status) (tariff_subscriptions_total)

# Renewal failure rate
rate(tariff_subscription_renewal_failures_total[5m])
```

### Performance Metrics

```promql
# Query duration (95th percentile)
histogram_quantile(0.95, rate(tariff_query_duration_seconds_bucket[5m]))

# Cache hit rate
tariff_cache_hit_rate

# API error rate
rate(tariff_errors_total[5m])
```

### Gateway Metrics

```promql
# Messages per gateway
sum by (gateway_id) (tariff_gateway_messages_total)

# Resource usage
tariff_gateway_resource_usage

# Quota exceeded events
rate(tariff_gateway_quota_exceeded_total[5m])
```

## 🛠️ Troubleshooting

### No Data in Grafana?

```bash
# 1. Check Prometheus is scraping backend
curl http://localhost:9090/api/v1/targets | grep backend

# 2. Check backend metrics endpoint
curl http://localhost:8080/metrics

# 3. Restart monitoring stack
docker-compose -f docker-compose.monitoring.yml restart
```

### Alerts Not Showing?

```bash
# 1. Check alert rules are loaded
curl http://localhost:9090/api/v1/rules | jq '.data.groups[].name'

# 2. Check Alertmanager is running
docker ps | grep alertmanager

# 3. View Prometheus logs
docker logs tariff-prometheus
```

### Dashboard Not Loading?

```bash
# 1. Check Grafana logs
docker logs tariff-grafana

# 2. Verify datasource
curl http://admin:admin123@localhost:3001/api/datasources

# 3. Restart Grafana
docker restart tariff-grafana
```

## 📝 Next Steps

1. **Customize Alerts:**
   - Edit `prometheus/tariff_alert_rules.yml`
   - Adjust thresholds based on your needs
   - Reload: `curl -X POST http://localhost:9090/-/reload`

2. **Configure Notifications:**
   - Edit `prometheus/alertmanager.yml`
   - Add email/Slack/PagerDuty
   - Restart: `docker restart tariff-alertmanager`

3. **Create Custom Dashboards:**
   - Use Grafana UI to create panels
   - Export JSON and save to `grafana/dashboards/`
   - Share with team

4. **Review Documentation:**
   - Full guide: `MONITORING_SETUP_GUIDE.md`
   - Alert rules: `prometheus/README.md`
   - Metrics reference: Backend `/metrics` endpoint

## 🎯 Success Criteria

✅ All monitoring services running
✅ Grafana dashboard showing data
✅ Prometheus scraping all targets
✅ Alert rules loaded
✅ No errors in logs

## 📞 Support

- Full Documentation: `MONITORING_SETUP_GUIDE.md`
- Prometheus Docs: https://prometheus.io/docs/
- Grafana Docs: https://grafana.com/docs/
- Contact: operations@alhudhud.com
