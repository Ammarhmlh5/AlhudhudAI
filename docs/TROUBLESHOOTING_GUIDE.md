# دليل استكشاف الأخطاء - نظام إدارة الرسائل SMS

## المشاكل الشائعة والحلول

### 1. النظام لا يعمل

#### الأعراض
- لا يمكن الوصول إلى API
- صفحة خطأ 502/503
- الخدمات لا تستجيب

#### التشخيص

```bash
# Check all services
docker-compose ps

# Check logs
docker-compose logs --tail=100

# Check system resources
docker stats
```

#### الحلول

**الحل 1: إعادة تشغيل الخدمات**
```bash
docker-compose restart
```

**الحل 2: إعادة بناء الخدمات**
```bash
docker-compose down
docker-compose up -d --build
```

**الحل 3: فحص الموارد**
```bash
# Check disk space
df -h

# Check memory
free -h

# Clean Docker
docker system prune -a
```

---

### 2. قاعدة البيانات لا تعمل

#### الأعراض
- خطأ "connection refused"
- خطأ "database does not exist"
- بطء شديد في الاستعلامات

#### التشخيص

```bash
# Check PostgreSQL status
docker exec postgres pg_isready -U postgres

# Check connections
docker exec postgres psql -U postgres -c "SELECT count(*) FROM pg_stat_activity;"

# Check database size
docker exec postgres psql -U postgres -c "SELECT pg_size_pretty(pg_database_size('sms_gateway'));"
```

#### الحلول

**الحل 1: إعادة تشغيل PostgreSQL**
```bash
docker-compose restart postgres
```

**الحل 2: فحص الاتصالات**
```bash
# Kill idle connections
docker exec postgres psql -U postgres -c "
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE state = 'idle'
  AND state_change < NOW() - INTERVAL '1 hour';
"
```

**الحل 3: تحسين الأداء**
```bash
# Vacuum database
docker exec postgres psql -U postgres -d sms_gateway -c "VACUUM ANALYZE;"

# Reindex
docker exec postgres psql -U postgres -d sms_gateway -c "REINDEX DATABASE sms_gateway;"
```

---

### 3. Redis لا يعمل

#### الأعراض
- خطأ "connection refused"
- خطأ "NOAUTH Authentication required"
- استهلاك ذاكرة عالي

#### التشخيص

```bash
# Check Redis status
docker exec redis redis-cli ping

# Check memory usage
docker exec redis redis-cli INFO memory

# Check keys count
docker exec redis redis-cli DBSIZE
```

#### الحلول

**الحل 1: إعادة تشغيل Redis**
```bash
docker-compose restart redis
```

**الحل 2: تنظيف الذاكرة**
```bash
# Flush expired keys
docker exec redis redis-cli --scan --pattern "*" | xargs docker exec redis redis-cli DEL

# Or flush all (CAUTION!)
docker exec redis redis-cli FLUSHALL
```

**الحل 3: زيادة الذاكرة**
```yaml
# docker-compose.yml
services:
  redis:
    command: redis-server --maxmemory 4gb --maxmemory-policy allkeys-lru
```

---

### 4. الرسائل لا تُرسل

#### الأعراض
- الرسائل عالقة في حالة QUEUED
- الرسائل في حالة WAITING
- معدل الإرسال منخفض

#### التشخيص

```bash
# Check message statuses
docker exec postgres psql -U postgres -d sms_gateway -c "
SELECT status, COUNT(*) 
FROM messages 
WHERE created_at > NOW() - INTERVAL '1 hour'
GROUP BY status;
"

# Check workers
docker-compose logs backend | grep "worker"

# Check channels
docker exec postgres psql -U postgres -d sms_gateway -c "
SELECT id, name, status, current_load, max_load
FROM channels;
"

# Check devices
docker exec postgres psql -U postgres -d sms_gateway -c "
SELECT id, name, status, last_seen
FROM devices
WHERE status = 'ONLINE';
"
```

#### الحلول

**الحل 1: لا توجد قنوات متاحة**
```bash
# Check channels
docker exec postgres psql -U postgres -d sms_gateway -c "
UPDATE channels
SET status = 'ONLINE'
WHERE id = 1;
"
```

**الحل 2: لا توجد أجهزة متصلة**
```bash
# Check devices
docker exec postgres psql -U postgres -d sms_gateway -c "
SELECT id, name, status, last_seen
FROM devices;
"

# Restart mobile gateway app
```

**الحل 3: Workers لا تعمل**
```bash
# Restart backend
docker-compose restart backend

# Check worker logs
docker-compose logs backend | grep "worker started"
```

---

### 5. أخطاء في الرصيد

#### الأعراض
- رصيد سالب
- عدم تطابق الرصيد
- خصم مضاعف

#### التشخيص

```bash
# Check balance
docker exec postgres psql -U postgres -d sms_gateway -c "
SELECT id, name, balance
FROM tenants;
"

# Check balance history
docker exec postgres psql -U postgres -d sms_gateway -c "
SELECT *
FROM balance_history
WHERE tenant_id = 1
ORDER BY created_at DESC
LIMIT 20;
"

# Verify balance integrity
docker exec postgres psql -U postgres -d sms_gateway -c "
SELECT 
    t.id,
    t.balance as current_balance,
    COALESCE(SUM(bh.amount), 0) as history_sum,
    t.balance - COALESCE(SUM(bh.amount), 0) as difference
FROM tenants t
LEFT JOIN balance_history bh ON t.id = bh.tenant_id
GROUP BY t.id, t.balance;
"
```

#### الحلول

**الحل 1: تصحيح الرصيد**
```bash
# Manual adjustment
docker exec postgres psql -U postgres -d sms_gateway -c "
UPDATE tenants
SET balance = 1000.00
WHERE id = 1;

INSERT INTO balance_history (tenant_id, type, amount, old_balance, new_balance, reason)
VALUES (1, 'ADJUSTMENT', 1000.00, 0, 1000.00, 'Manual correction');
"
```

**الحل 2: إعادة حساب الرصيد**
```bash
docker exec backend ./bin/recalculate-balance --tenant-id 1
```

---

### 6. أداء بطيء

#### الأعراض
- Response time > 100ms
- Throughput < 1000 msg/min
- استهلاك CPU/Memory عالي

#### التشخيص

```bash
# Check system resources
docker stats

# Check database performance
docker exec postgres psql -U postgres -d sms_gateway -c "
SELECT 
    query,
    calls,
    total_time,
    mean_time
FROM pg_stat_statements
ORDER BY mean_time DESC
LIMIT 10;
"

# Check Redis performance
docker exec redis redis-cli --latency

# Check queue size
docker exec redis redis-cli LLEN messages:pending:1
```

#### الحلول

**الحل 1: تحسين قاعدة البيانات**
```bash
# Analyze and vacuum
docker exec postgres psql -U postgres -d sms_gateway -c "VACUUM ANALYZE;"

# Add missing indexes
docker exec postgres psql -U postgres -d sms_gateway -c "
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at DESC);
"
```

**الحل 2: زيادة Workers**
```yaml
# .env
WORKER_COUNT=100
```

**الحل 3: تحسين Batch Size**
```yaml
# .env
BATCH_SIZE=200
BATCH_INTERVAL=50ms
```

---

### 7. WebSocket لا يعمل

#### الأعراض
- الأجهزة لا تتصل
- انقطاع متكرر
- رسائل لا تصل للأجهزة

#### التشخيص

```bash
# Check WebSocket connections
docker-compose logs backend | grep "websocket"

# Check device status
docker exec postgres psql -U postgres -d sms_gateway -c "
SELECT id, name, status, last_seen
FROM devices
ORDER BY last_seen DESC;
"
```

#### الحلول

**الحل 1: إعادة تشغيل Backend**
```bash
docker-compose restart backend
```

**الحل 2: فحص Nginx Configuration**
```nginx
# /etc/nginx/sites-available/sms-gateway
location /ws {
    proxy_pass http://localhost:8080/ws;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
}
```

**الحل 3: إعادة تشغيل Mobile Gateway**
```bash
# On mobile device
# Force close app
# Clear cache
# Restart app
```

---

### 8. Rate Limiting مشاكل

#### الأعراض
- خطأ 429 Too Many Requests
- الطلبات مرفوضة بشكل خاطئ

#### التشخيص

```bash
# Check rate limit counters
docker exec redis redis-cli GET rate_limit:1

# Check rate limit config
cat .env | grep RATE_LIMIT
```

#### الحلول

**الحل 1: زيادة الحد**
```yaml
# .env
RATE_LIMIT_PER_MINUTE=200
```

**الحل 2: إعادة تعيين العداد**
```bash
docker exec redis redis-cli DEL rate_limit:1
```

---

### 9. Prometheus/Grafana لا يعمل

#### الأعراض
- لا توجد metrics
- Dashboards فارغة
- Alerts لا تعمل

#### التشخيص

```bash
# Check Prometheus
curl http://localhost:9090/-/healthy

# Check Grafana
curl http://localhost:3000/api/health

# Check metrics endpoint
curl http://localhost:8080/metrics
```

#### الحلول

**الحل 1: إعادة تشغيل الخدمات**
```bash
docker-compose restart prometheus grafana
```

**الحل 2: فحص Configuration**
```yaml
# prometheus/prometheus.yml
scrape_configs:
  - job_name: 'backend'
    static_configs:
      - targets: ['backend:8080']
```

---

### 10. Migration فشل

#### الأعراض
- خطأ أثناء تشغيل migrations
- جداول مفقودة
- بيانات غير متطابقة

#### التشخيص

```bash
# Check migration status
docker exec postgres psql -U postgres -d sms_gateway -c "
SELECT * FROM schema_migrations ORDER BY version;
"

# Check migration logs
cat /opt/sms-gateway/migrations/migration.log
```

#### الحلول

**الحل 1: إعادة تشغيل Migration**
```bash
# Rollback failed migration
docker exec postgres psql -U postgres -d sms_gateway -c "
DELETE FROM schema_migrations WHERE version = 'XXX';
"

# Re-run migrations
./run-migrations.sh
```

**الحل 2: Manual Migration**
```bash
# Run migration SQL manually
docker exec postgres psql -U postgres -d sms_gateway < migrations/XXX_migration.sql
```

---

## أدوات التشخيص

### 1. Health Check Script

```bash
#!/bin/bash
# scripts/health-check.sh

echo "=== System Health Check ==="

# Backend
echo -n "Backend: "
if curl -f http://localhost:8080/health > /dev/null 2>&1; then
    echo "✓ OK"
else
    echo "✗ FAILED"
fi

# Database
echo -n "Database: "
if docker exec postgres pg_isready -U postgres > /dev/null 2>&1; then
    echo "✓ OK"
else
    echo "✗ FAILED"
fi

# Redis
echo -n "Redis: "
if docker exec redis redis-cli ping > /dev/null 2>&1; then
    echo "✓ OK"
else
    echo "✗ FAILED"
fi

# Prometheus
echo -n "Prometheus: "
if curl -f http://localhost:9090/-/healthy > /dev/null 2>&1; then
    echo "✓ OK"
else
    echo "✗ FAILED"
fi

# Grafana
echo -n "Grafana: "
if curl -f http://localhost:3000/api/health > /dev/null 2>&1; then
    echo "✓ OK"
else
    echo "✗ FAILED"
fi
```

### 2. Performance Check Script

```bash
#!/bin/bash
# scripts/performance-check.sh

echo "=== Performance Check ==="

# API Response Time
echo -n "API Response Time: "
time curl -s http://localhost:8080/health > /dev/null

# Database Connections
echo -n "Database Connections: "
docker exec postgres psql -U postgres -c "SELECT count(*) FROM pg_stat_activity;" -t

# Redis Memory
echo -n "Redis Memory: "
docker exec redis redis-cli INFO memory | grep used_memory_human

# Queue Size
echo -n "Queue Size: "
docker exec redis redis-cli LLEN messages:pending:1
```

---

## Logs المهمة

### Backend Logs

```bash
# All logs
docker-compose logs backend

# Errors only
docker-compose logs backend | grep ERROR

# Specific time range
docker-compose logs --since 1h backend

# Follow logs
docker-compose logs -f backend
```

### Database Logs

```bash
# PostgreSQL logs
docker-compose logs postgres

# Slow queries
docker exec postgres psql -U postgres -d sms_gateway -c "
SELECT query, calls, total_time, mean_time
FROM pg_stat_statements
WHERE mean_time > 100
ORDER BY mean_time DESC;
"
```

### Redis Logs

```bash
# Redis logs
docker-compose logs redis

# Redis slow log
docker exec redis redis-cli SLOWLOG GET 10
```

---

## جهات الاتصال للدعم

### المستوى 1 (L1) - الدعم الأساسي
- Email: support@example.com
- Response Time: 1 hour

### المستوى 2 (L2) - الدعم الفني
- Email: tech-support@example.com
- Response Time: 30 minutes

### المستوى 3 (L3) - الطوارئ
- Hotline: +966-XXX-XXXX
- Email: emergency@example.com
- Response Time: 15 minutes

---

**تاريخ الإنشاء**: 2026-02-26  
**الإصدار**: 1.0  
**آخر تحديث**: 2026-02-26
