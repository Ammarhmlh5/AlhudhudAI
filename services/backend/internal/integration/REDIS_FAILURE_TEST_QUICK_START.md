# Redis Failure Fallback Test - دليل البدء السريع

## نظرة عامة

هذا الاختبار يتحقق من قدرة النظام على التعامل مع فشل Redis والتحول التلقائي للمعالجة المباشرة (Direct Processing) بدون فقدان الرسائل.

**المتطلبات المرتبطة:** 19.5, 24.2

## المتطلبات الأساسية

### 1. البنية التحتية
```bash
# تشغيل PostgreSQL
docker-compose up -d postgres

# تشغيل Redis
docker-compose up -d redis
```

### 2. قاعدة البيانات
```bash
# تطبيق Migrations
cd services/backend
go run cmd/migrate/main.go up
```

## تشغيل الاختبار

### الطريقة 1: تشغيل جميع السيناريوهات
```bash
cd services/backend
go test -v ./internal/integration -run TestRedisFailureFallback -timeout 5m
```

### الطريقة 2: تشغيل سيناريو محدد
```bash
# سيناريو 1: Redis Failure with Fallback
go test -v ./internal/integration -run TestRedisFailureFallback/RedisFailureWithFallback

# سيناريو 2: System Continues Without Redis
go test -v ./internal/integration -run TestRedisFailureFallback/SystemContinuesWithoutRedis

# سيناريو 3: Automatic Reconnection
go test -v ./internal/integration -run TestRedisFailureFallback/AutomaticReconnectionToRedis

# سيناريو 4: Batch Processing with Redis Failure
go test -v ./internal/integration -run TestRedisFailureFallback/BatchProcessingWithRedisFailure

# سيناريو 5: Circuit Breaker Behavior
go test -v ./internal/integration -run TestRedisFailureFallback/CircuitBreakerBehavior
```

### الطريقة 3: استخدام السكريبت
```bash
# Windows
.\run-redis-failure-test.ps1

# Linux/Mac
./run-redis-failure-test.sh
```

## السيناريوهات المختبرة

### 1. Redis Failure with Fallback
- ✅ اكتشاف فشل Redis تلقائياً
- ✅ التحول للمعالجة المباشرة (Direct Database Insert)
- ✅ عدم فقدان الرسائل
- ✅ تتبع مقاييس Fallback

**الخطوات:**
1. إرسال رسالة إلى Redis بنجاح
2. محاكاة فشل Redis (إغلاق الاتصال)
3. محاولة إرسال رسالة جديدة
4. التحقق من التحول التلقائي للـ Fallback
5. التحقق من حفظ الرسالة في قاعدة البيانات

### 2. System Continues Without Redis
- ✅ النظام يستمر في العمل بدون Redis
- ✅ معالجة رسائل متعددة عبر Fallback
- ✅ عدم فقدان أي رسالة
- ✅ جميع الرسائل تُحفظ في قاعدة البيانات

**الخطوات:**
1. بدء النظام مع Redis غير متاح
2. معالجة 5 رسائل عبر Fallback
3. التحقق من حفظ جميع الرسائل في قاعدة البيانات
4. التحقق من عدم فقدان أي رسالة

### 3. Automatic Reconnection to Redis
- ✅ إعادة الاتصال بـ Redis عند عودته
- ✅ استئناف استخدام Redis بعد الاتصال
- ✅ إعادة تعيين Circuit Breaker

**الخطوات:**
1. إرسال رسالة إلى Redis بنجاح
2. محاكاة فشل Redis
3. معالجة رسالة عبر Fallback
4. إعادة الاتصال بـ Redis
5. إرسال رسالة جديدة إلى Redis بنجاح

### 4. Batch Processing with Redis Failure
- ✅ معالجة دفعات كبيرة بدون Redis
- ✅ جميع الرسائل تُعالج عبر Fallback
- ✅ تتبع مقاييس الدفعات

**الخطوات:**
1. معالجة 10 رسائل بدون Redis
2. التحقق من معالجة جميع الرسائل عبر Fallback
3. التحقق من مقاييس Fallback

### 5. Circuit Breaker Behavior
- ✅ فتح Circuit Breaker بعد 3 فشل
- ✅ استخدام Fallback فوراً عند فتح الدائرة
- ✅ التحول لحالة Half-Open بعد Timeout

**الخطوات:**
1. فشل 3 عمليات Redis لفتح Circuit Breaker
2. التحقق من فتح الدائرة
3. معالجة رسالة (يجب استخدام Fallback فوراً)
4. الانتظار لحالة Half-Open

## النتائج المتوقعة

### ✅ نجاح الاختبار
```
=== RUN   TestRedisFailureFallback
=== RUN   TestRedisFailureFallback/RedisFailureWithFallback
    ✓ Redis is initially available
    ✓ Message pushed to Redis successfully
    ✓ Buffer size: 1
    ✓ Redis is unavailable
    ✓ Message processed via fallback (direct database insert)
    ✓ Message found in database (fallback successful)
    ✓ Fallback metrics: Attempts=1, Successes=1, Failures=0
--- PASS: TestRedisFailureFallback/RedisFailureWithFallback (0.25s)

=== RUN   TestRedisFailureFallback/SystemContinuesWithoutRedis
    ✓ Redis is unavailable from start
    ✓ Processed 5/5 messages successfully without Redis
    ✓ Found 5 messages in database
    ✓ No message loss detected
--- PASS: TestRedisFailureFallback/SystemContinuesWithoutRedis (0.15s)

=== RUN   TestRedisFailureFallback/AutomaticReconnectionToRedis
    ✓ Redis is available
    ✓ Message pushed to Redis
    ✓ Redis connection closed
    ✓ Message processed via fallback
    ✓ Reconnected to Redis
    ✓ Circuit breaker reset
    ✓ Message pushed to Redis after reconnection
    ✓ Buffer size after reconnection: 1
--- PASS: TestRedisFailureFallback/AutomaticReconnectionToRedis (0.30s)

=== RUN   TestRedisFailureFallback/BatchProcessingWithRedisFailure
    ✓ Processed 10/10 batch messages via fallback
    ✓ Batch metrics: Primary attempts=10, Fallback attempts=10, Fallback successes=10
    ✓ Fallback usage rate: 100.00%
--- PASS: TestRedisFailureFallback/BatchProcessingWithRedisFailure (0.20s)

=== RUN   TestRedisFailureFallback/CircuitBreakerBehavior
    ✓ Circuit breaker is OPEN
    ✓ Fallback used immediately (circuit breaker working)
    ✓ Message processed via fallback
    ✓ Circuit breaker state after timeout: HALF_OPEN
--- PASS: TestRedisFailureFallback/CircuitBreakerBehavior (6.15s)

PASS
ok      github.com/al-hudhud/backend/internal/integration    7.050s
```

## استكشاف الأخطاء

### المشكلة: Redis connection refused
```
Error: dial tcp [::1]:6379: connect: connection refused
```

**الحل:**
```bash
# تأكد من تشغيل Redis
docker-compose up -d redis

# تحقق من حالة Redis
docker-compose ps redis
redis-cli ping
```

### المشكلة: Database connection failed
```
Error: failed to connect to database
```

**الحل:**
```bash
# تأكد من تشغيل PostgreSQL
docker-compose up -d postgres

# تحقق من الاتصال
psql -h localhost -U postgres -d sms_gateway_test -c "SELECT 1"
```

### المشكلة: Test timeout
```
Error: test timed out after 5m
```

**الحل:**
```bash
# زيادة وقت الانتظار
go test -v ./internal/integration -run TestRedisFailureFallback -timeout 10m
```

### المشكلة: Circuit breaker not opening
```
Error: Circuit breaker should be open
```

**الحل:**
- تأكد من فشل 3 عمليات Redis على الأقل
- تحقق من إعدادات Circuit Breaker (threshold = 3)

## التحقق اليدوي

### 1. فحص Redis
```bash
# الاتصال بـ Redis
redis-cli

# فحص المفاتيح
KEYS messages:pending:*

# فحص حجم Queue
LLEN messages:pending:1

# فحص Circuit Breaker state
GET circuit:redis:state
```

### 2. فحص قاعدة البيانات
```sql
-- عرض الرسائل المعالجة
SELECT 
    message_id, 
    phone_number, 
    status, 
    created_at 
FROM messages 
WHERE tenant_id = '<tenant_id>'
ORDER BY created_at DESC 
LIMIT 10;

-- عدد الرسائل حسب الحالة
SELECT status, COUNT(*) 
FROM messages 
WHERE tenant_id = '<tenant_id>'
GROUP BY status;
```

### 3. فحص Fallback Metrics
```bash
# عرض مقاييس Fallback من Logs
grep "FALLBACK" services/backend/logs/app.log

# عرض مقاييس Circuit Breaker
grep "circuit breaker" services/backend/logs/app.log
```

## المقاييس المهمة

### Fallback Metrics
- **Primary Attempts**: عدد محاولات استخدام Redis
- **Primary Successes**: عدد العمليات الناجحة على Redis
- **Primary Failures**: عدد العمليات الفاشلة على Redis
- **Fallback Attempts**: عدد محاولات استخدام Fallback
- **Fallback Successes**: عدد عمليات Fallback الناجحة
- **Fallback Failures**: عدد عمليات Fallback الفاشلة
- **Fallback Usage Rate**: نسبة استخدام Fallback (%)

### Circuit Breaker States
- **CLOSED**: الدائرة مغلقة، Redis يعمل بشكل طبيعي
- **OPEN**: الدائرة مفتوحة، استخدام Fallback فوراً
- **HALF_OPEN**: الدائرة نصف مفتوحة، اختبار Redis

## الخطوات التالية

بعد نجاح الاختبار:
1. ✅ تحديث حالة المهمة 15.7 إلى مكتملة
2. ✅ الانتقال للمهمة 15.8 (Checkpoint - مراجعة اختبارات التكامل)
3. ✅ مراجعة جميع اختبارات التكامل
4. ✅ التأكد من عدم وجود أخطاء حرجة

## الموارد الإضافية

- [دليل الاختبار المفصل](./REDIS_FAILURE_TEST_GUIDE.md)
- [وثيقة المتطلبات](../../../.kiro/specs/message-lifecycle-rebuild/requirements.md)
- [وثيقة التصميم](../../../.kiro/specs/message-lifecycle-rebuild/design.md)
- [خطة المهام](../../../.kiro/specs/message-lifecycle-rebuild/tasks.md)
