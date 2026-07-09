# Performance Optimization - Quick Start Guide

## Overview

This guide provides quick instructions for using the performance optimization tools implemented in Task 35.

## Quick Commands

### Database Optimization

```powershell
# Run full optimization
cd services/backend
.\run-database-optimization.ps1
# Select option 1

# View slow queries
.\run-database-optimization.ps1
# Select option 6

# Check cache hit ratio
.\run-database-optimization.ps1
# Select option 9
```

### Cache Warming

```powershell
# Warm all caches
cd services/backend
.\scripts\warm-cache.ps1
```

### Monitoring

```
# Access Grafana
http://localhost:3000

# Access Prometheus
http://localhost:9090

# View Tariff Dashboard
http://localhost:3000/d/tariff-system-dashboard
```

## Performance Targets

| Metric | Target | Current |
|--------|--------|---------|
| Query Response Time | < 50ms | Monitor |
| Cache Hit Rate | > 80% | Monitor |
| Index Usage | > 95% | Monitor |
| API Response Time | < 100ms | Monitor |

## Common Tasks

### 1. Check System Performance

```powershell
# Check database performance
.\run-database-optimization.ps1
# Select option 6 (View slow queries)
# Select option 7 (Check index usage)
# Select option 9 (Check cache hit ratio)
```

### 2. Optimize Database

```powershell
# Add missing indexes
.\run-database-optimization.ps1
# Select option 3

# Update statistics
.\run-database-optimization.ps1
# Select option 4

# Vacuum tables
.\run-database-optimization.ps1
# Select option 5
```

### 3. Warm Cache

```powershell
# Warm all caches
.\scripts\warm-cache.ps1
```

### 4. View Metrics

1. Open Grafana: http://localhost:3000
2. Navigate to "Tariff Pricing System Dashboard"
3. Review key metrics:
   - Subscriptions
   - Rollover Messages
   - Query Duration
   - Cache Hit Rate
   - Gateway Performance

### 5. Check Alerts

1. Open Prometheus: http://localhost:9090
2. Click "Alerts" in the top menu
3. Review active alerts
4. Investigate any critical alerts

## Troubleshooting

### Slow Queries

**Symptoms:**
- API responses > 100ms
- Dashboard shows high query duration

**Solution:**
```powershell
# 1. Check slow queries
.\run-database-optimization.ps1
# Select option 6

# 2. Add missing indexes
.\run-database-optimization.ps1
# Select option 3

# 3. Update statistics
.\run-database-optimization.ps1
# Select option 4
```

### Low Cache Hit Rate

**Symptoms:**
- Cache hit rate < 80%
- High database load

**Solution:**
```powershell
# 1. Warm cache
.\scripts\warm-cache.ps1

# 2. Check Redis health
# Open Grafana and check Redis metrics

# 3. Review TTL settings
# Check internal/cache/query_cache.go
```

### High Alert Volume

**Symptoms:**
- Many alerts firing
- Alert fatigue

**Solution:**
1. Review alert thresholds in `prometheus/tariff_alert_rules.yml`
2. Adjust thresholds based on actual usage
3. Group related alerts
4. Document common resolutions

## Maintenance Schedule

### Daily
- [ ] Review Grafana dashboards
- [ ] Check active alerts
- [ ] Monitor cache hit rates

### Weekly
- [ ] Run VACUUM ANALYZE
- [ ] Review slow query log
- [ ] Check index usage

### Monthly
- [ ] Full database optimization
- [ ] Review alert thresholds
- [ ] Analyze performance trends

## Key Files

| File | Purpose |
|------|---------|
| `scripts/database-optimization.sql` | Database optimization script |
| `run-database-optimization.ps1` | PowerShell runner |
| `scripts/warm-cache.ps1` | Cache warming script |
| `internal/cache/query_cache.go` | Query cache implementation |
| `internal/monitoring/redis_monitor.go` | Redis monitoring |
| `grafana/dashboards/tariff-system-dashboard.json` | Grafana dashboard |
| `prometheus/tariff_alert_rules.yml` | Alert rules |

## Support

For detailed information, see:
- `.kiro/specs/tariff-pricing-system/TASK_35_PERFORMANCE_OPTIMIZATION_SUMMARY.md`
- `.kiro/specs/tariff-pricing-system/guides/TASK_35_PERFORMANCE_OPTIMIZATION_GUIDE.md`

## Quick Reference

### Database Optimization Menu
1. Run full optimization
2. Enable slow query logging
3. Add missing indexes
4. Update table statistics
5. Vacuum and cleanup
6. View slow queries
7. Check index usage
8. Check table sizes
9. Check cache hit ratio
0. Exit

### Cache TTL Settings
- Tariffs: 1 hour
- Packages: 30 minutes
- Subscriptions: 5 minutes
- Balance: 1 minute
- Usage: 30 seconds
- Stats: 10 minutes

### Alert Severity Levels
- **Info:** Informational, no action required
- **Warning:** Requires attention within 1 hour
- **Critical:** Requires immediate action

## Performance Checklist

Before deploying to production:
- [ ] Run full database optimization
- [ ] Warm all caches
- [ ] Verify Grafana dashboards are accessible
- [ ] Test alert delivery (Slack/Email)
- [ ] Review alert thresholds
- [ ] Document baseline metrics
- [ ] Train team on monitoring tools

## Contact

For questions or issues:
- Check documentation in `.kiro/specs/tariff-pricing-system/`
- Review logs in Grafana
- Check Prometheus alerts
