# دليل اختبار أداء Cache - Routing Engine

## نظرة عامة

هذا الدليل يشرح كيفية تشغيل وفهم اختبارات أداء Redis Cache في Routing Engine.

## المتطلبات المسبقة

### 1. تشغيل البنية التحتية

```bash
# تشغيل PostgreSQL و Redis
docker-compose up -d postgres redis

# أو استخدام السكريبت
./start-infrastructure.sh  # Linux/Mac
./start-infrastructure.bat # Windows
```

### 2. تشغيل Migrations

```bash
cd services/backend
go run cmd/migrate/main.go up
```

### 3. التحقق من الاتصال

```bash
# PostgreSQL
psql -h localhost -U postgres -d alhudhud_test -c "SELECT 1"

# Redis
redis-cli -h localhost -p 6379 ping
```

## تشغيل الاختبارات

### تشغيل جميع اختبارات الأداء

```bash
cd services/backend
go test -v -run TestCachePerformance ./internal/routing/
```

### تشغيل اختبار محدد

```bash
# 1. Cache Hit Rate (الأهم - المتطلب 5.7)
go test -v -run TestCachePerformance_HitRate ./internal/routing/

# 2. Database Load Reduction (المتطلب 5.7 الأساسي)
go test -v -run TestCachePerformance_DatabaseLoadReduction ./internal/routing/

# 3. Latency Comparison
go test -v -run TestCachePerformance_LatencyComparison ./internal/routing/

# 4. Concurrent Access
go test -v -run TestCachePerformance_ConcurrentAccess ./internal/routing/

# 5. TTL Expiration
go test -v -run TestCachePerformance_TTLExpiration ./internal/routing/

# 6. Multi-Tenant Isolation
go test -v -run TestCachePerformance_MultiTenantIsolation ./internal/routing/
```

### تخطي اختبارات الأداء

```bash
# تشغيل الاختبارات السريعة فقط (يتخطى اختبارات الأداء)
go test -short ./internal/routing/
```

## فهم النتائج

### 1. Cache Hit Rate Test

**الهدف:** التحقق من أن Cache Hit Rate > 95%

**مثال على النتيجة:**
```
Cache Performance Results:
  Total Operations: 100
  Cache Hits: 95
  Cache Misses: 5
  Hit Rate: 95.00%
  Total Duration: 250ms
  Average Latency: 2.5ms
```

**التفسير:**
- ✅ Hit Rate = 95% (يحقق المتطلب)
- ✅ Average Latency = 2.5ms (ممتاز)
- ❌ إذا كان Hit Rate < 95%، هناك مشكلة في Cache

### 2. Database Load Reduction Test

**الهدف:** التحقق من تقليل Database Queries بنسبة 99%

**مثال على النتيجة:**
```
Database Load Reduction:
  Operations: 100
  DB Queries Without Cache: 100
  DB Queries With Cache: 1
  Query Reduction: 99.00%
  Duration Without Cache: 1.5s
  Duration With Cache: 120ms
  Speedup: 12.5x
```

**التفسير:**
- ✅ Query Reduction = 99% (يحقق المتطلب 5.7)
- ✅ Speedup = 12.5x (تحسن ممتاز)
- ✅ مع Cache: استعلام واحد فقط من 100 عملية

### 3. Latency Comparison Test

**الهدف:** قياس الفرق في الأداء بين Cache Hit و Cache Miss

**مثال على النتيجة:**
```
Latency Comparison:
  Cache Miss Latency: 15ms
  Cache Hit Latency (avg): 1.2ms
  Performance Improvement: 12.5x
```

**التفسير:**
- ✅ Cache Hit أسرع 12.5x من Cache Miss
- ✅ Cache Hit < 2ms (ممتاز)
- ℹ️ Cache Miss ~15ms (طبيعي لاستعلام Database)

### 4. Concurrent Access Test

**الهدف:** التحقق من الأداء تحت الضغط المتزامن

**مثال على النتيجة:**
```
Concurrent Access Performance:
  Goroutines: 50
  Iterations per Goroutine: 20
  Total Operations: 1000
  Overall Duration: 850ms
  Avg Duration per Goroutine: 45ms
  Throughput: 1176.47 ops/sec
```

**التفسير:**
- ✅ Throughput > 1000 ops/sec (ممتاز)
- ✅ جميع العمليات نجحت
- ✅ لا تعارض أو أخطاء

### 5. TTL Expiration Test

**الهدف:** التحقق من سلوك Cache بعد انتهاء TTL

**مثال على النتيجة:**
```
TTL Expiration Test:
  Call 1 (miss): 14ms
  Call 2 (hit): 1.1ms
  Call 3 (miss after TTL): 15ms
  Call 4 (hit after refresh): 1.2ms
```

**التفسير:**
- ✅ Cache Hits أسرع من Cache Misses
- ✅ بعد انتهاء TTL، يتم إعادة التحميل تلقائياً
- ✅ الأداء يعود للمستوى الطبيعي

### 6. Multi-Tenant Test

**الهدف:** التحقق من عدم تأثير العزل على الأداء

**مثال على النتيجة:**
```
Multi-Tenant Performance:
  Tenants: 10
  Iterations per Tenant: 20
  Total Operations: 200
  Total Duration: 350ms
  Average Latency: 1.75ms
  Throughput: 571.43 ops/sec
```

**التفسير:**
- ✅ Average Latency < 5ms (ممتاز)
- ✅ Throughput > 500 ops/sec (جيد)
- ✅ لا تداخل بين Tenants

## معايير النجاح

| الاختبار | المعيار | الأهمية |
|---------|---------|---------|
| Cache Hit Rate | > 95% | ⭐⭐⭐ حرج |
| Database Query Reduction | > 99% | ⭐⭐⭐ حرج |
| Cache Hit Latency | < 2ms | ⭐⭐⭐ حرج |
| Performance Improvement | > 5x | ⭐⭐⭐ حرج |
| Concurrent Throughput | > 1000 ops/sec | ⭐⭐ مهم |
| Multi-Tenant Throughput | > 500 ops/sec | ⭐⭐ مهم |

## استكشاف الأخطاء

### مشكلة: فشل الاتصال بـ PostgreSQL

```
Error: failed to connect to test database
```

**الحل:**
```bash
# تحقق من تشغيل PostgreSQL
docker ps | grep postgres

# إعادة تشغيل
docker-compose restart postgres

# تحقق من الاتصال
psql -h localhost -U postgres -d alhudhud_test -c "SELECT 1"
```

### مشكلة: فشل الاتصال بـ Redis

```
Error: failed to ping Redis
```

**الحل:**
```bash
# تحقق من تشغيل Redis
docker ps | grep redis

# إعادة تشغيل
docker-compose restart redis

# تحقق من الاتصال
redis-cli -h localhost -p 6379 ping
```

### مشكلة: Cache Hit Rate منخفض

```
Cache hit rate should be greater than 95%
Expected: > 95.0
Actual: 85.0
```

**الأسباب المحتملة:**
1. Redis لا يعمل بشكل صحيح
2. TTL قصير جداً
3. Cache يتم إبطاله بشكل متكرر

**الحل:**
```bash
# تحقق من Redis
redis-cli -h localhost -p 6379 INFO stats

# تحقق من TTL
redis-cli -h localhost -p 6379 TTL "channels:cache:*"

# مراجعة Logs
tail -f services/backend/logs/app.log | grep cache
```

### مشكلة: Database Query Reduction أقل من 99%

```
Cache should reduce database queries by at least 99%
Expected: > 99.0
Actual: 90.0
```

**الأسباب المحتملة:**
1. Cache لا يعمل
2. Cache يتم إبطاله بشكل متكرر
3. مشكلة في منطق Caching

**الحل:**
```bash
# تحقق من عدد الاستعلامات
# راجع engine.go - SelectChannel()
# تأكد من البحث في Cache أولاً
```

## التكامل مع CI/CD

### GitHub Actions

```yaml
name: Cache Performance Tests

on: [push, pull_request]

jobs:
  cache-performance:
    runs-on: ubuntu-latest
    
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: alhudhud_test
        ports:
          - 5432:5432
      
      redis:
        image: redis:7
        ports:
          - 6379:6379
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Run Migrations
        run: |
          cd services/backend
          go run cmd/migrate/main.go up
      
      - name: Run Cache Performance Tests
        run: |
          cd services/backend
          go test -v -run TestCachePerformance ./internal/routing/
```

## المراقبة في Production

### Prometheus Metrics

```go
// في engine.go
metrics.RecordCacheHit("channels")   // عند Cache Hit
metrics.RecordCacheMiss("channels")  // عند Cache Miss
```

### Grafana Dashboard

```
Panel 1: Cache Hit Rate
Query: rate(cache_hits_total[5m]) / (rate(cache_hits_total[5m]) + rate(cache_misses_total[5m])) * 100

Panel 2: Cache Latency
Query: histogram_quantile(0.95, rate(cache_latency_seconds_bucket[5m]))

Panel 3: Database Query Reduction
Query: rate(database_queries_total[5m])
```

### Alerts

```yaml
# alert_rules.yml
- alert: LowCacheHitRate
  expr: cache_hit_rate < 90
  for: 5m
  annotations:
    summary: "Cache hit rate is below 90%"
    description: "Current hit rate: {{ $value }}%"

- alert: HighCacheMissRate
  expr: rate(cache_misses_total[5m]) > 100
  for: 5m
  annotations:
    summary: "High cache miss rate detected"
```

## الخلاصة

اختبارات أداء Cache توفر:
- ✅ التحقق من المتطلب 5.7 (تقليل Database Queries بنسبة 99%)
- ✅ قياس Cache Hit Rate (يجب أن يكون > 95%)
- ✅ مقارنة الأداء مع وبدون Cache
- ✅ اختبار تحت الضغط المتزامن
- ✅ التحقق من العزل بين Tenants

للمزيد من المعلومات، راجع:
- `cache_performance_test.go` - الكود الكامل للاختبارات
- `TASK_14.7_EXECUTION_SUMMARY.md` - ملخص التنفيذ والنتائج
- `engine.go` - تطبيق Caching في Routing Engine

---

**آخر تحديث:** 2024
**الإصدار:** 1.0
