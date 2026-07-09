# دليل تحسين الأداء - المهمة 35

## نظرة عامة

هذا الدليل يغطي تحسين أداء نظام التعرفات والتسعير في جميع المجالات:
- Database Optimization
- Redis Optimization
- Connection Pooling
- Query Optimization
- Monitoring Setup
- Alerting Setup

## المتطلبات

### البرمجيات المطلوبة
- PostgreSQL 15+
- Redis 7+
- Prometheus
- Grafana
- AlertManager

### الأدوات المطلوبة
```bash
# PostgreSQL tools
sudo apt-get install postgresql-contrib

# Monitoring tools
# تثبيت Prometheus و Grafana عبر Docker
```

## المهمة 35.1: Database Optimization

### الهدف
تحسين أداء قاعدة البيانات وتسريع الاستعلامات.

### خطوات التنفيذ

#### 1. مراجعة جميع الاستعلامات

```sql
-- تفعيل تسجيل الاستعلامات البطيئة
ALTER SYSTEM SET log_min_duration_statement = 100; -- 100ms
ALTER SYSTEM SET log_statement = 'all';
SELECT pg_reload_conf();

-- عرض الاستعلامات البطيئة
SELECT 
    query,
    calls,
    total_time,
    mean_time,
    max_time
FROM pg_stat_statements
WHERE mean_time > 100
ORDER BY mean_time DESC
LIMIT 20;
```

#### 2. إضافة Indexes المفقودة

```sql
-- تحليل الاستعلامات لإيجاد Indexes المفقودة
SELECT 
    schemaname,
    tablename,
    attname,
    n_distinct,
    correlation
FROM pg_stats
WHERE schemaname = 'public'
AND tablename IN ('subscriptions', 'tariffs', 'packages', 'rollover_messages')
ORDER BY n_distinct DESC;

-- إضافة Indexes مركبة مفيدة
CREATE INDEX CONCURRENTLY idx_subscriptions_tenant_status_renewal 
ON subscriptions(tenant_id, status, next_renewal_date)
WHERE status IN ('active', 'grace_period');

CREATE INDEX CONCURRENTLY idx_rollover_tenant_subscription_status
ON rollover_messages(tenant_id, subscription_id, status)
WHERE status = 'active';

CREATE INDEX CONCURRENTLY idx_accounting_tenant_created_type
ON accounting_transactions(tenant_id, created_at DESC, transaction_type);
```

#### 3. تحسين الاستعلامات البطيئة

```sql
-- قبل التحسين
EXPLAIN ANALYZE
SELECT * FROM subscriptions 
WHERE tenant_id = 'xxx' 
AND status = 'active'
ORDER BY created_at DESC;

-- بعد التحسين: استخدام Index-Only Scan
EXPLAIN ANALYZE
SELECT id, package_id, status, next_renewal_date
FROM subscriptions 
WHERE tenant_id = 'xxx' 
AND status = 'active'
ORDER BY created_at DESC;
```

#### 4. تطبيق Query Caching

```go
// services/backend/internal/cache/query_cache.go
package cache

import (
    "context"
    "encoding/json"
    "time"
    
    "github.com/redis/go-redis/v9"
)

type QueryCache struct {
    redis *redis.Client
}

func NewQueryCache(redis *redis.Client) *QueryCache {
    return &QueryCache{redis: redis}
}

func (qc *QueryCache) GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() (interface{}, error)) (interface{}, error) {
    // محاولة الحصول من Cache
    val, err := qc.redis.Get(ctx, key).Result()
    if err == nil {
        var result interface{}
        if err := json.Unmarshal([]byte(val), &result); err == nil {
            return result, nil
        }
    }
    
    // تنفيذ الاستعلام
    result, err := fn()
    if err != nil {
        return nil, err
    }
    
    // حفظ في Cache
    data, _ := json.Marshal(result)
    qc.redis.Set(ctx, key, data, ttl)
    
    return result, nil
}
```

### Checklist للتحقق

- [ ] جميع الاستعلامات البطيئة محسّنة
- [ ] Indexes مناسبة لجميع الاستعلامات الشائعة
- [ ] Query Caching مطبّق للاستعلامات المتكررة
- [ ] متوسط وقت الاستعلام < 50ms

## المهمة 35.2: Redis Optimization

### الهدف
تحسين استخدام Redis وتسريع عمليات الـ Caching.

### خطوات التنفيذ

#### 1. مراجعة استراتيجية الـ Caching

```go
// services/backend/internal/cache/strategy.go
package cache

import "time"

// استراتيجيات TTL مختلفة حسب نوع البيانات
var CacheTTL = map[string]time.Duration{
    "tariff":       1 * time.Hour,      // التعرفات نادراً ما تتغير
    "package":      30 * time.Minute,   // الباقات قد تتغير
    "subscription": 5 * time.Minute,    // الاشتراكات تتغير بشكل متكرر
    "balance":      1 * time.Minute,    // الرصيد يتغير باستمرار
    "usage":        30 * time.Second,   // الاستخدام يتغير بسرعة
    "stats":        10 * time.Minute,   // الإحصائيات
}

// Cache Warming للبيانات الشائعة
func (qc *QueryCache) WarmCache(ctx context.Context) error {
    // تحميل التعرفات النشطة
    tariffs, err := qc.loadActiveTariffs(ctx)
    if err != nil {
        return err
    }
    
    for _, tariff := range tariffs {
        key := fmt.Sprintf("tariff:%d", tariff.ID)
        qc.Set(ctx, key, tariff, CacheTTL["tariff"])
    }
    
    // تحميل الباقات النشطة
    packages, err := qc.loadActivePackages(ctx)
    if err != nil {
        return err
    }
    
    for _, pkg := range packages {
        key := fmt.Sprintf("package:%d", pkg.ID)
        qc.Set(ctx, key, pkg, CacheTTL["package"])
    }
    
    return nil
}
```

#### 2. تحسين TTL للمفاتيح المختلفة

```go
// تطبيق TTL ديناميكي حسب نوع Tenant
func (qc *QueryCache) GetTTL(tenantType string, dataType string) time.Duration {
    baseTTL := CacheTTL[dataType]
    
    // Enterprise tenants يحصلون على TTL أطول
    if tenantType == "enterprise" {
        return baseTTL * 2
    }
    
    return baseTTL
}
```

#### 3. تطبيق Cache Warming

```bash
# services/backend/scripts/warm-cache.sh
#!/bin/bash

echo "Warming cache..."

# استدعاء API لتحميل البيانات الشائعة
curl -X POST http://localhost:8080/admin/cache/warm \
    -H "Authorization: Bearer $ADMIN_TOKEN"

echo "Cache warmed successfully"
```

#### 4. مراقبة أداء Redis

```go
// services/backend/internal/monitoring/redis_monitor.go
package monitoring

import (
    "context"
    "time"
    
    "github.com/redis/go-redis/v9"
)

type RedisMonitor struct {
    client *redis.Client
}

func (rm *RedisMonitor) CollectMetrics(ctx context.Context) (*RedisMetrics, error) {
    info, err := rm.client.Info(ctx, "stats").Result()
    if err != nil {
        return nil, err
    }
    
    return &RedisMetrics{
        HitRate:        rm.calculateHitRate(info),
        MemoryUsed:     rm.parseMemoryUsed(info),
        KeysCount:      rm.parseKeysCount(info),
        EvictedKeys:    rm.parseEvictedKeys(info),
        ConnectedClients: rm.parseConnectedClients(info),
    }, nil
}
```

### Checklist للتحقق

- [ ] استراتيجية Caching محسّنة
- [ ] TTL مناسب لكل نوع بيانات
- [ ] Cache Warming مطبّق
- [ ] Hit Rate > 80%
- [ ] Memory Usage ضمن الحدود

## المهمة 35.3: Connection Pooling

### الهدف
تحسين إدارة الاتصالات لقاعدة البيانات و Redis.

### خطوات التنفيذ

#### 1. تحسين Database Connection Pool

```go
// services/backend/internal/database/pool.go
package database

import (
    "time"
    "gorm.io/gorm"
)

func OptimizeConnectionPool(db *gorm.DB) error {
    sqlDB, err := db.DB()
    if err != nil {
        return err
    }
    
    // الحد الأقصى للاتصالات المفتوحة
    sqlDB.SetMaxOpenConns(100)
    
    // الحد الأقصى للاتصالات الخاملة
    sqlDB.SetMaxIdleConns(10)
    
    // مدة حياة الاتصال
    sqlDB.SetConnMaxLifetime(time.Hour)
    
    // مدة خمول الاتصال
    sqlDB.SetConnMaxIdleTime(10 * time.Minute)
    
    return nil
}
```

#### 2. تحسين Redis Connection Pool

```go
// services/backend/internal/redis/pool.go
package redis

import (
    "time"
    "github.com/redis/go-redis/v9"
)

func NewOptimizedRedisClient(addr string) *redis.Client {
    return redis.NewClient(&redis.Options{
        Addr:         addr,
        PoolSize:     100,              // حجم Pool
        MinIdleConns: 10,               // الحد الأدنى للاتصالات الخاملة
        MaxRetries:   3,                // عدد المحاولات
        DialTimeout:  5 * time.Second,  // Timeout للاتصال
        ReadTimeout:  3 * time.Second,  // Timeout للقراءة
        WriteTimeout: 3 * time.Second,  // Timeout للكتابة
        PoolTimeout:  4 * time.Second,  // Timeout للحصول على اتصال من Pool
    })
}
```

### Checklist للتحقق

- [ ] Connection Pool محسّن
- [ ] لا توجد Connection Leaks
- [ ] Connection Timeout مناسب
- [ ] Pool Size مناسب للحمل

## المهمة 35.4: Query Optimization

### الهدف
تحسين الاستعلامات باستخدام EXPLAIN ANALYZE.

### خطوات التنفيذ

#### 1. تحليل الاستعلامات

```sql
-- استخدام EXPLAIN ANALYZE
EXPLAIN (ANALYZE, BUFFERS, VERBOSE)
SELECT s.*, p.name as package_name
FROM subscriptions s
JOIN packages p ON s.package_id = p.id
WHERE s.tenant_id = 'xxx'
AND s.status = 'active'
ORDER BY s.created_at DESC
LIMIT 10;
```

#### 2. إعادة كتابة الاستعلامات البطيئة

```sql
-- قبل: استعلام بطيء
SELECT * FROM subscriptions
WHERE tenant_id = 'xxx'
AND next_renewal_date <= CURRENT_DATE + INTERVAL '3 days';

-- بعد: استعلام محسّن
SELECT id, package_id, next_renewal_date
FROM subscriptions
WHERE tenant_id = 'xxx'
AND next_renewal_date BETWEEN CURRENT_DATE AND CURRENT_DATE + INTERVAL '3 days'
AND status = 'active';
```

#### 3. تطبيق Prepared Statements

```go
// services/backend/internal/database/prepared.go
package database

import (
    "context"
    "database/sql"
)

type PreparedStatements struct {
    getSubscription *sql.Stmt
    deductBalance   *sql.Stmt
    updateUsage     *sql.Stmt
}

func NewPreparedStatements(db *sql.DB) (*PreparedStatements, error) {
    getSubscription, err := db.Prepare(`
        SELECT id, tenant_id, package_id, status, messages_remaining
        FROM subscriptions
        WHERE id = $1 AND tenant_id = $2
    `)
    if err != nil {
        return nil, err
    }
    
    deductBalance, err := db.Prepare(`
        UPDATE tenants
        SET balance = balance - $2
        WHERE id = $1 AND balance >= $2
        RETURNING balance
    `)
    if err != nil {
        return nil, err
    }
    
    return &PreparedStatements{
        getSubscription: getSubscription,
        deductBalance:   deductBalance,
    }, nil
}
```

### Checklist للتحقق

- [ ] جميع الاستعلامات محللة بـ EXPLAIN
- [ ] الاستعلامات البطيئة معاد كتابتها
- [ ] Prepared Statements مطبّقة
- [ ] Query Plan محسّن

## المهمة 35.5: Monitoring Setup

### الهدف
إعداد Prometheus و Grafana لمراقبة النظام.

### خطوات التنفيذ

#### 1. إعداد Prometheus

```yaml
# prometheus/prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'tariff-system'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```

#### 2. إضافة Metrics للتطبيق

```go
// services/backend/internal/metrics/tariff_metrics.go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    SubscriptionsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "tariff_subscriptions_total",
            Help: "Total number of subscriptions",
        },
        []string{"package", "status"},
    )
    
    BalanceDeductions = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "tariff_balance_deductions",
            Help:    "Balance deduction amounts",
            Buckets: prometheus.LinearBuckets(0, 10, 10),
        },
        []string{"tenant_type"},
    )
    
    RolloverMessages = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "tariff_rollover_messages",
            Help: "Number of rollover messages",
        },
        []string{"tenant_id", "subscription_id"},
    )
)
```

#### 3. إنشاء Dashboards في Grafana

```json
// grafana/dashboards/tariff-dashboard.json
{
  "dashboard": {
    "title": "Tariff System Dashboard",
    "panels": [
      {
        "title": "Subscriptions by Status",
        "targets": [
          {
            "expr": "sum(tariff_subscriptions_total) by (status)"
          }
        ]
      },
      {
        "title": "Balance Deductions",
        "targets": [
          {
            "expr": "rate(tariff_balance_deductions_sum[5m])"
          }
        ]
      }
    ]
  }
}
```

### Checklist للتحقق

- [ ] Prometheus يجمع المقاييس
- [ ] Grafana يعرض Dashboards
- [ ] جميع المقاييس الحرجة مراقبة
- [ ] Dashboards واضحة ومفيدة

## المهمة 35.6: Alerting Setup

### الهدف
إعداد التنبيهات التلقائية للمشاكل الحرجة.

### خطوات التنفيذ

#### 1. إعداد Alert Rules

```yaml
# prometheus/alert_rules.yml
groups:
  - name: tariff_alerts
    interval: 30s
    rules:
      - alert: HighErrorRate
        expr: rate(tariff_errors_total[5m]) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value }} errors/sec"
      
      - alert: SlowQueries
        expr: histogram_quantile(0.95, rate(tariff_query_duration_seconds_bucket[5m])) > 0.5
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Slow queries detected"
          description: "95th percentile query time is {{ $value }}s"
      
      - alert: LowCacheHitRate
        expr: tariff_cache_hit_rate < 0.7
        for: 15m
        labels:
          severity: warning
        annotations:
          summary: "Low cache hit rate"
          description: "Cache hit rate is {{ $value }}"
```

#### 2. تكامل مع Slack

```yaml
# alertmanager/alertmanager.yml
global:
  slack_api_url: 'https://hooks.slack.com/services/YOUR/WEBHOOK/URL'

route:
  receiver: 'slack-notifications'
  group_by: ['alertname', 'severity']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 1h

receivers:
  - name: 'slack-notifications'
    slack_configs:
      - channel: '#tariff-alerts'
        title: '{{ .GroupLabels.alertname }}'
        text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'
```

### Checklist للتحقق

- [ ] Alert Rules محددة
- [ ] AlertManager يرسل التنبيهات
- [ ] Slack Integration يعمل
- [ ] التنبيهات واضحة ومفيدة

## أفضل الممارسات

### 1. Database Optimization
- استخدم Indexes بحكمة
- راقب الاستعلامات البطيئة
- نظف البيانات القديمة دورياً

### 2. Redis Optimization
- استخدم TTL مناسب
- راقب Memory Usage
- طبّق Cache Warming

### 3. Monitoring
- راقب المقاييس الحرجة
- أنشئ Dashboards واضحة
- راجع المقاييس بانتظام

### 4. Alerting
- حدد عتبات مناسبة
- تجنب Alert Fatigue
- وثّق إجراءات الاستجابة

## الخلاصة

هذا الدليل يغطي جميع جوانب تحسين الأداء للمهمة 35. اتبع الخطوات بعناية وراقب النتائج.

**Requirements Covered:** 27.1-27.12, 29.1-29.12
