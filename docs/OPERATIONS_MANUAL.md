# دليل العمليات - نظام إدارة الرسائل SMS

## المهام اليومية

### 1. فحص صحة النظام

```bash
# Check all services
docker-compose ps

# Check backend health
curl https://api.example.com/health

# Check logs for errors
docker-compose logs --tail=100 backend | grep ERROR
```

### 2. مراقبة الأداء

```bash
# Check Grafana dashboards
# URL: https://grafana.example.com
# - Performance Dashboard
# - Resources Dashboard
# - Financial Dashboard

# Check Prometheus alerts
# URL: https://prometheus.example.com/alerts
```

### 3. مراجعة الإحصائيات

```bash
# Daily message statistics
docker exec postgres psql -U postgres -d sms_gateway -c "
SELECT 
    DATE(created_at) as date,
    COUNT(*) as total,
    COUNT(CASE WHEN status = 'DELIVERED' THEN 1 END) as delivered,
    COUNT(CASE WHEN status = 'FAILED' THEN 1 END) as failed,
    ROUND(COUNT(CASE WHEN status = 'DELIVERED' THEN 1 END)::numeric / COUNT(*)::numeric * 100, 2) as success_rate
FROM messages
WHERE created_at >= CURRENT_DATE
GROUP BY DATE(created_at);
"
```

---

## المهام الأسبوعية

### 1. مراجعة Logs

```bash
# Archive old logs
docker-compose logs --since 7d > logs/week_$(date +%Y%m%d).log

# Check for recurring errors
grep ERROR logs/week_*.log | sort | uniq -c | sort -rn | head -20
```

### 2. تنظيف قاعدة البيانات

```bash
# Clean old DLQ entries (> 7 days)
docker exec postgres psql -U postgres -d sms_gateway -c "
DELETE FROM dead_letter_queue
WHERE moved_to_dlq_at < NOW() - INTERVAL '7 days';
"

# Archive old messages (> 6 months)
# Run archival script
./scripts/archive-old-messages.sh
```

### 3. مراجعة الأمان

```bash
# Check failed login attempts
docker exec postgres psql -U postgres -d sms_gateway -c "
SELECT 
    DATE(created_at) as date,
    COUNT(*) as failed_attempts
FROM audit_trail
WHERE action = 'LOGIN_FAILED'
  AND created_at >= NOW() - INTERVAL '7 days'
GROUP BY DATE(created_at);
"

# Review audit trail
docker exec postgres psql -U postgres -d sms_gateway -c "
SELECT 
    action,
    COUNT(*) as count
FROM audit_trail
WHERE created_at >= NOW() - INTERVAL '7 days'
GROUP BY action
ORDER BY count DESC;
"
```

---

## المهام الشهرية

### 1. النسخ الاحتياطي الكامل

```bash
# Full database backup
pg_dump -h localhost -U postgres -d sms_gateway | gzip > /backups/monthly/backup_$(date +%Y%m).sql.gz

# Backup application files
tar -czf /backups/monthly/app_$(date +%Y%m).tar.gz /opt/sms-gateway

# Upload to cloud storage
aws s3 cp /backups/monthly/ s3://backups/sms-gateway/ --recursive
```

### 2. تحديث النظام

```bash
# Update system packages
sudo apt update && sudo apt upgrade -y

# Update Docker images
docker-compose pull

# Restart services
docker-compose up -d
```

### 3. مراجعة الأداء

```bash
# Generate monthly report
./scripts/generate-monthly-report.sh

# Review metrics
# - Average throughput
# - Success rate
# - Response time
# - Resource usage
```

---

## إدارة المستخدمين

### إنشاء مستخدم جديد

```bash
# Create admin user
docker exec backend ./bin/create-user \
  --email admin@example.com \
  --password secure_password \
  --role admin \
  --tenant-id 1

# Create client user
docker exec backend ./bin/create-user \
  --email client@example.com \
  --password secure_password \
  --role client \
  --tenant-id 2
```

### تعطيل مستخدم

```bash
docker exec postgres psql -U postgres -d sms_gateway -c "
UPDATE users
SET status = 'SUSPENDED'
WHERE email = 'user@example.com';
"
```

### إعادة تعيين كلمة المرور

```bash
docker exec backend ./bin/reset-password \
  --email user@example.com \
  --new-password new_secure_password
```

---

## إدارة الشركات (Tenants)

### إنشاء شركة جديدة

```bash
docker exec backend ./bin/create-tenant \
  --name "اسم الشركة" \
  --email tenant@example.com \
  --initial-balance 1000
```

### شحن رصيد

```bash
docker exec backend ./bin/add-balance \
  --tenant-id 1 \
  --amount 500 \
  --reason "Manual recharge"
```

### تعليق حساب شركة

```bash
docker exec postgres psql -U postgres -d sms_gateway -c "
UPDATE tenants
SET status = 'SUSPENDED'
WHERE id = 1;
"
```

---

## إدارة القنوات والأجهزة

### إضافة قناة جديدة

```bash
docker exec postgres psql -U postgres -d sms_gateway -c "
INSERT INTO channels (tenant_id, name, priority, max_load)
VALUES (1, 'قناة جديدة', 5, 100);
"
```

### تسجيل جهاز جديد

```bash
docker exec backend ./bin/register-device \
  --tenant-id 1 \
  --channel-id 1 \
  --name "جهاز 1" \
  --phone-number "+966501234567"
```

### تعطيل جهاز

```bash
docker exec postgres psql -U postgres -d sms_gateway -c "
UPDATE devices
SET status = 'OFFLINE'
WHERE id = 1;
"
```

---

## إدارة Dead Letter Queue

### عرض الرسائل الفاشلة

```bash
docker exec postgres psql -U postgres -d sms_gateway -c "
SELECT 
    id,
    message_id,
    phone_number,
    attempts,
    last_error,
    moved_to_dlq_at
FROM dead_letter_queue
WHERE reprocessed = FALSE
ORDER BY moved_to_dlq_at DESC
LIMIT 20;
"
```

### إعادة معالجة رسالة

```bash
docker exec backend ./bin/reprocess-dlq \
  --dlq-id 123
```

### حذف رسائل قديمة

```bash
docker exec postgres psql -U postgres -d sms_gateway -c "
DELETE FROM dead_letter_queue
WHERE moved_to_dlq_at < NOW() - INTERVAL '7 days';
"
```

---

## المراقبة والتنبيهات

### إعداد تنبيهات Slack

```yaml
# alertmanager.yml
receivers:
  - name: 'slack'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/YOUR/WEBHOOK/URL'
        channel: '#alerts'
        title: 'SMS Gateway Alert'
        text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'
```

### إعداد تنبيهات Email

```yaml
# alertmanager.yml
receivers:
  - name: 'email'
    email_configs:
      - to: 'ops@example.com'
        from: 'alerts@example.com'
        smarthost: 'smtp.example.com:587'
        auth_username: 'alerts@example.com'
        auth_password: 'password'
```

---

## الصيانة

### إعادة تشغيل الخدمات

```bash
# Restart all services
docker-compose restart

# Restart specific service
docker-compose restart backend

# Restart with rebuild
docker-compose up -d --build backend
```

### تنظيف Docker

```bash
# Remove unused images
docker image prune -a

# Remove unused volumes
docker volume prune

# Remove unused networks
docker network prune
```

### تحسين قاعدة البيانات

```bash
# Vacuum database
docker exec postgres psql -U postgres -d sms_gateway -c "VACUUM ANALYZE;"

# Reindex database
docker exec postgres psql -U postgres -d sms_gateway -c "REINDEX DATABASE sms_gateway;"
```

---

## النسخ الاحتياطي والاسترداد

### النسخ الاحتياطي اليومي

```bash
#!/bin/bash
# /opt/sms-gateway/scripts/daily-backup.sh

BACKUP_DIR="/backups/daily"
DATE=$(date +%Y%m%d)

# Database backup
pg_dump -h localhost -U postgres -d sms_gateway | gzip > $BACKUP_DIR/db_$DATE.sql.gz

# Redis backup
docker exec redis redis-cli BGSAVE
sleep 5
docker cp redis:/data/dump.rdb $BACKUP_DIR/redis_$DATE.rdb

# Keep only last 7 days
find $BACKUP_DIR -name "db_*.sql.gz" -mtime +7 -delete
find $BACKUP_DIR -name "redis_*.rdb" -mtime +7 -delete
```

### الاسترداد من النسخة الاحتياطية

```bash
# Restore database
gunzip < /backups/daily/db_YYYYMMDD.sql.gz | docker exec -i postgres psql -U postgres sms_gateway

# Restore Redis
docker cp /backups/daily/redis_YYYYMMDD.rdb redis:/data/dump.rdb
docker restart redis
```

---

## الأمان

### تحديث كلمات المرور

```bash
# Update database password
docker exec postgres psql -U postgres -c "ALTER USER postgres PASSWORD 'new_password';"

# Update Redis password
docker exec redis redis-cli CONFIG SET requirepass "new_password"

# Update .env file
nano .env.production
```

### مراجعة الصلاحيات

```bash
# List users and roles
docker exec postgres psql -U postgres -d sms_gateway -c "
SELECT 
    email,
    role,
    status,
    last_login
FROM users
ORDER BY last_login DESC;
"
```

### تفعيل Two-Factor Authentication

```bash
# Enable 2FA for user
docker exec backend ./bin/enable-2fa \
  --email user@example.com
```

---

## الاستكشاف السريع

### النظام بطيء

```bash
# Check CPU usage
docker stats

# Check database connections
docker exec postgres psql -U postgres -c "SELECT count(*) FROM pg_stat_activity;"

# Check Redis memory
docker exec redis redis-cli INFO memory

# Check queue size
docker exec redis redis-cli LLEN messages:pending:1
```

### رسائل لا تُرسل

```bash
# Check workers
docker-compose logs backend | grep "worker"

# Check channels
docker exec postgres psql -U postgres -d sms_gateway -c "
SELECT id, name, status, current_load, max_load
FROM channels
WHERE status = 'ONLINE';
"

# Check devices
docker exec postgres psql -U postgres -d sms_gateway -c "
SELECT id, name, status, last_seen
FROM devices
WHERE status = 'ONLINE';
"
```

### أخطاء في الرصيد

```bash
# Check balance history
docker exec postgres psql -U postgres -d sms_gateway -c "
SELECT *
FROM balance_history
WHERE tenant_id = 1
ORDER BY created_at DESC
LIMIT 20;
"

# Verify balance integrity
docker exec backend ./bin/verify-balance --tenant-id 1
```

---

## جهات الاتصال

### فريق الدعم

- **Operations Team**: ops@example.com
- **Development Team**: dev@example.com
- **Security Team**: security@example.com

### الطوارئ

- **Hotline**: +966-XXX-XXXX
- **On-Call**: oncall@example.com
- **Slack**: #emergency

---

**تاريخ الإنشاء**: 2026-02-26  
**الإصدار**: 1.0  
**آخر تحديث**: 2026-02-26
