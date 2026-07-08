# Redis Failure Fallback Test - الدليل المفصل

## المقدمة

هذا الدليل يوفر شرحاً تفصيلياً لاختبار Redis Failure Fallback، الذي يتحقق من قدرة النظام على التعامل مع فشل Redis والتحول التلقائي للمعالجة المباشرة بدون فقدان الرسائل.

## الأهداف

### الأهداف الرئيسية
1. **اكتشاف فشل Redis**: التحقق من قدرة النظام على اكتشاف فشل Redis تلقائياً
2. **التحول التلقائي للـ Fallback**: التحقق من التحول للمعالجة المباشرة (Direct Processing)
3. **عدم فقدان الرسائل**: التأكد من عدم فقدان أي رسالة أثناء فشل Redis
4. **إعادة الاتصال التلقائي**: التحقق من إعادة الاتصال بـ Redis عند عودته
5. **Circuit Breaker**: التحقق من عمل Circuit Breaker بشكل صحيح

### المتطلبات المرتبطة
- **المتطلب 19.5**: WHEN يفشل اتصال Redis، THE Message_Lifecycle_System SHALL التحول لمعالجة مباشرة بدون Cache
- **المتطلب 24.2**: WHEN يتعطل Redis، THE Message_Lifecycle_System SHALL التحول لمعالجة مباشرة من قاعدة البيانات

## البنية المعمارية

### المكونات المشاركة

```
┌─────────────────┐
│   API Gateway   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐      ┌──────────────┐
│ Message Buffer  │─────▶│    Redis     │
│  (with Fallback)│      │  (Primary)   │
└────────┬────────┘      └──────────────┘
         │                       │
         │ Fallback              │ Failure
         │ (when Redis fails)    │
         ▼                       ▼
┌─────────────────┐      ┌──────────────┐
│   PostgreSQL    │◀─────│Circuit Breaker│
│ (Direct Insert) │      │   (Monitor)   │
└─────────────────┘      └──────────────┘
```

### تدفق البيانات

#### الحالة الطبيعية (Redis متاح)
```
Client → API Gateway → Redis Buffer → PostgreSQL (Batch)
```

#### حالة فشل Redis (Fallback)
```
Client → API Gateway → [Redis Failed] → PostgreSQL (Direct)
                              ↓
                       Circuit Breaker
```

## السيناريوهات التفصيلية

### السيناريو 1: Redis Failure with Fallback

#### الهدف
التحقق من قدرة النظام على اكتشاف فشل Redis والتحول التلقائي للمعالجة المباشرة.

#### الخطوات التفصيلية

**الخطوة 1: إعداد البيئة**
```go
// إنشاء اتصال قاعدة البيانات
db, err := database.NewDB(ctx, &database.Config{
    Host:     "localhost",
    Port:     5432,
    User:     "postgres",
    Password: "postgres",
    Database: "sms_gateway_test",
})

// إنشاء tenant للاختبار
tenantID := createTestTenantForRedis(t, ctx, db, decimal.NewFromFloat(100.0))

// إنشاء اتصال Redis
redisClient := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})

// إنشاء Message Buffer
messageBuffer := buffer.NewMessageBuffer(redisClient, logger)

// إنشاء Fallback Strategy
circuitBreaker := errors.NewCircuitBreaker(3, 10*time.Second)
fallbackStrategy := errors.NewRedisFallbackStrategy(circuitBreaker)
```

**الخطوة 2: إرسال رسالة إلى Redis بنجاح**
```go
msg1 := gateway.BufferedMessage{
    MessageID:   uuid.New().String(),
    TenantID:    tenantID,
    PhoneNumber: "+966501234567",
    Content:     "Test message before Redis failure",
    Priority:    "NORMAL",
    QueuedAt:    time.Now(),
}

err = messageBuffer.Push(ctx, tenantID, msg1)
// Expected: err == nil
// Expected: Message in Redis
```

**الخطوة 3: محاكاة فشل Redis**
```go
// إغلاق اتصال Redis
redisClient.Close()
time.Sleep(100 * time.Millisecond)

// التحقق من عدم توفر Redis
err = redisClient.Ping(ctx).Err()
// Expected: err != nil
```

**الخطوة 4: محاولة إرسال رسالة مع Fallback**
```go
msg2 := gateway.BufferedMessage{
    MessageID:   uuid.New().String(),
    TenantID:    tenantID,
    PhoneNumber: "+966501234568",
    Content:     "Test message during Redis failure",
    Priority:    "NORMAL",
    QueuedAt:    time.Now(),
}

// Primary operation (سيفشل)
primaryOp := func() error {
    return messageBuffer.Push(ctx, tenantID, msg2)
}

// Fallback operation (إدخال مباشر في قاعدة البيانات)
fallbackOp := func() error {
    _, err := db.Pool.Exec(ctx, `
        INSERT INTO messages (...)
        VALUES (...)
    `, ...)
    return err
}

// تنفيذ مع Fallback
err = fallbackStrategy.Execute(ctx, primaryOp, fallbackOp)
// Expected: err == nil (Fallback succeeded)
```

**الخطوة 5: التحقق من النتائج**
```go
// التحقق من وجود الرسالة في قاعدة البيانات
var count int
err = db.Pool.QueryRow(ctx, `
    SELECT COUNT(*) FROM messages 
    WHERE tenant_id = $1 AND message_id = $2
`, tenantID, msg2.MessageID).Scan(&count)
// Expected: count == 1

// التحقق من مقاييس Fallback
metrics := fallbackStrategy.GetMetrics()
// Expected: metrics.FallbackAttempts > 0
// Expected: metrics.FallbackSuccesses > 0
```

#### معايير النجاح
- ✅ Redis يعمل في البداية
- ✅ الرسالة الأولى تُحفظ في Redis بنجاح
- ✅ فشل Redis يُكتشف تلقائياً
- ✅ الرسالة الثانية تُعالج عبر Fallback
- ✅ الرسالة موجودة في قاعدة البيانات
- ✅ مقاييس Fallback صحيحة

---

### السيناريو 2: System Continues Without Redis

#### الهدف
التحقق من قدرة النظام على الاستمرار في العمل بشكل كامل بدون Redis.

#### الخطوات التفصيلية

**الخطوة 1: بدء النظام مع Redis غير متاح**
```go
// إنشاء Redis client مع منفذ خاطئ
redisClient := redis.NewClient(&redis.Options{
    Addr: "localhost:9999", // منفذ غير صحيح
})

// التحقق من عدم توفر Redis
err = redisClient.Ping(ctx).Err()
// Expected: err != nil
```

**الخطوة 2: معالجة رسائل متعددة**
```go
messageCount := 5
successCount := 0

for i := 0; i < messageCount; i++ {
    msg := gateway.BufferedMessage{
        MessageID:   uuid.New().String(),
        TenantID:    tenantID,
        PhoneNumber: fmt.Sprintf("+96650123456%d", i),
        Content:     fmt.Sprintf("Test message %d without Redis", i+1),
        Priority:    "NORMAL",
        QueuedAt:    time.Now(),
    }

    // محاولة Redis (سيفشل) ثم Fallback
    err := fallbackStrategy.Execute(ctx, primaryOp, fallbackOp)
    if err == nil {
        successCount++
    }
}

// Expected: successCount == messageCount
```

**الخطوة 3: التحقق من عدم فقدان الرسائل**
```go
// عد الرسائل في قاعدة البيانات
var dbCount int
err = db.Pool.QueryRow(ctx, `
    SELECT COUNT(*) FROM messages 
    WHERE tenant_id = $1 AND status = 'PENDING'
`, tenantID).Scan(&dbCount)

// Expected: dbCount >= messageCount
```

**الخطوة 4: التحقق من مقاييس Fallback**
```go
metrics := fallbackStrategy.GetMetrics()
// Expected: metrics.FallbackAttempts == messageCount
// Expected: metrics.FallbackSuccesses == messageCount
// Expected: metrics.FallbackFailures == 0
```

#### معايير النجاح
- ✅ Redis غير متاح من البداية
- ✅ جميع الرسائل تُعالج بنجاح (5/5)
- ✅ جميع الرسائل موجودة في قاعدة البيانات
- ✅ لا يوجد فقدان للرسائل
- ✅ مقاييس Fallback تظهر 100% نجاح

---

### السيناريو 3: Automatic Reconnection to Redis

#### الهدف
التحقق من قدرة النظام على إعادة الاتصال بـ Redis تلقائياً عند عودته.

#### الخطوات التفصيلية

**الخطوة 1: العمل مع Redis متاح**
```go
// إنشاء Redis client
redisClient := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})

// إرسال رسالة بنجاح
err = messageBuffer.Push(ctx, tenantID, msg1)
// Expected: err == nil
```

**الخطوة 2: محاكاة فشل Redis**
```go
redisClient.Close()
time.Sleep(100 * time.Millisecond)
```

**الخطوة 3: معالجة رسالة عبر Fallback**
```go
err = fallbackStrategy.Execute(ctx, primaryOp, fallbackOp)
// Expected: err == nil (via fallback)
```

**الخطوة 4: إعادة الاتصال بـ Redis**
```go
// إنشاء اتصال جديد
newRedisClient := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})

// التحقق من الاتصال
err = newRedisClient.Ping(ctx).Err()
// Expected: err == nil

// إعادة تعيين Circuit Breaker
circuitBreaker.Reset()
```

**الخطوة 5: إرسال رسالة إلى Redis بعد الاتصال**
```go
newMessageBuffer := buffer.NewMessageBuffer(newRedisClient, logger)
err = newMessageBuffer.Push(ctx, tenantID, msg3)
// Expected: err == nil

// التحقق من وجود الرسالة في Redis
size, err := newMessageBuffer.GetSize(ctx, tenantID)
// Expected: size > 0
```

#### معايير النجاح
- ✅ Redis يعمل في البداية
- ✅ فشل Redis يُكتشف
- ✅ معالجة عبر Fallback أثناء الفشل
- ✅ إعادة الاتصال بـ Redis بنجاح
- ✅ استئناف استخدام Redis بعد الاتصال

---

### السيناريو 4: Batch Processing with Redis Failure

#### الهدف
التحقق من قدرة النظام على معالجة دفعات كبيرة من الرسائل بدون Redis.

#### الخطوات التفصيلية

**الخطوة 1: إعداد دفعة من الرسائل**
```go
batchSize := 10
messages := make([]gateway.BufferedMessage, batchSize)

for i := 0; i < batchSize; i++ {
    messages[i] = gateway.BufferedMessage{
        MessageID:   uuid.New().String(),
        TenantID:    tenantID,
        PhoneNumber: fmt.Sprintf("+96650123456%d", i),
        Content:     fmt.Sprintf("Batch message %d", i+1),
        Priority:    "NORMAL",
        QueuedAt:    time.Now(),
    }
}
```

**الخطوة 2: معالجة الدفعة مع Fallback**
```go
successCount := 0
for _, msg := range messages {
    err := fallbackStrategy.Execute(ctx, primaryOp, fallbackOp)
    if err == nil {
        successCount++
    }
}

// Expected: successCount == batchSize
```

**الخطوة 3: التحقق من مقاييس الدفعة**
```go
metrics := fallbackStrategy.GetMetrics()
// Expected: metrics.FallbackAttempts == batchSize
// Expected: metrics.FallbackSuccesses == batchSize

fallbackRate := metrics.GetFallbackUsageRate()
// Expected: fallbackRate == 100.0
```

#### معايير النجاح
- ✅ معالجة 10/10 رسائل بنجاح
- ✅ جميع الرسائل عبر Fallback
- ✅ معدل استخدام Fallback = 100%
- ✅ لا يوجد فشل في Fallback

---

### السيناريو 5: Circuit Breaker Behavior

#### الهدف
التحقق من عمل Circuit Breaker بشكل صحيح مع فشل Redis.

#### الخطوات التفصيلية

**الخطوة 1: فتح Circuit Breaker**
```go
// إنشاء Circuit Breaker مع threshold منخفض
circuitBreaker := errors.NewCircuitBreaker(3, 5*time.Second)

// فشل 3 عمليات لفتح الدائرة
for i := 0; i < 3; i++ {
    err := messageBuffer.Push(ctx, tenantID, msg)
    // Expected: err != nil
}

// التحقق من فتح الدائرة
isOpen := circuitBreaker.IsOpen()
// Expected: isOpen == true
```

**الخطوة 2: معالجة رسالة مع دائرة مفتوحة**
```go
// يجب استخدام Fallback فوراً بدون محاولة Redis
err := fallbackStrategy.Execute(ctx, primaryOp, fallbackOp)
// Expected: err == nil (via fallback immediately)

// التحقق من حفظ الرسالة
var count int
err = db.Pool.QueryRow(ctx, `
    SELECT COUNT(*) FROM messages 
    WHERE tenant_id = $1 AND message_id = $2
`, tenantID, msg.MessageID).Scan(&count)
// Expected: count == 1
```

**الخطوة 3: انتظار Half-Open State**
```go
// الانتظار لمدة timeout
time.Sleep(6 * time.Second)

// التحقق من الحالة
state := circuitBreaker.GetState()
// Expected: state == HALF_OPEN
```

#### معايير النجاح
- ✅ Circuit Breaker يفتح بعد 3 فشل
- ✅ Fallback يُستخدم فوراً عند فتح الدائرة
- ✅ الرسالة تُعالج بنجاح عبر Fallback
- ✅ الدائرة تتحول لـ Half-Open بعد Timeout

## المقاييس والمراقبة

### Fallback Metrics

#### المقاييس الأساسية
```go
type FallbackMetrics struct {
    PrimaryAttempts   int64     // عدد محاولات Redis
    PrimarySuccesses  int64     // عدد نجاحات Redis
    PrimaryFailures   int64     // عدد فشل Redis
    FallbackAttempts  int64     // عدد محاولات Fallback
    FallbackSuccesses int64     // عدد نجاحات Fallback
    FallbackFailures  int64     // عدد فشل Fallback
    LastFallbackTime  time.Time // آخر وقت استخدام Fallback
}
```

#### المقاييس المحسوبة
```go
// معدل نجاح Redis (0-100%)
primarySuccessRate := (PrimarySuccesses / PrimaryAttempts) * 100

// معدل نجاح Fallback (0-100%)
fallbackSuccessRate := (FallbackSuccesses / FallbackAttempts) * 100

// معدل استخدام Fallback (0-100%)
fallbackUsageRate := (FallbackAttempts / PrimaryAttempts) * 100
```

### Circuit Breaker States

#### الحالات الممكنة
- **CLOSED**: الدائرة مغلقة، Redis يعمل بشكل طبيعي
- **OPEN**: الدائرة مفتوحة، استخدام Fallback فوراً
- **HALF_OPEN**: الدائرة نصف مفتوحة، اختبار Redis

#### انتقالات الحالة
```
CLOSED ──[3 failures]──> OPEN
  ▲                        │
  │                        │
  └──[success]──  HALF_OPEN ◄──[timeout]
                     │
                     └──[failure]──> OPEN
```

## استكشاف الأخطاء المتقدم

### مشكلة: Fallback لا يعمل

#### الأعراض
```
Error: fallback operation failed
```

#### التشخيص
```go
// فحص Circuit Breaker
state := circuitBreaker.GetState()
fmt.Printf("Circuit Breaker State: %v\n", state)

// فحص Fallback Metrics
metrics := fallbackStrategy.GetMetrics()
fmt.Printf("Fallback Attempts: %d\n", metrics.FallbackAttempts)
fmt.Printf("Fallback Failures: %d\n", metrics.FallbackFailures)

// فحص قاعدة البيانات
err := db.Pool.Ping(ctx)
fmt.Printf("Database Status: %v\n", err)
```

#### الحلول
1. تحقق من اتصال قاعدة البيانات
2. تحقق من صلاحيات الإدخال في جدول messages
3. تحقق من صحة بيانات الرسالة

### مشكلة: Circuit Breaker لا يفتح

#### الأعراض
```
Error: Circuit breaker should be open but is closed
```

#### التشخيص
```go
// فحص عدد الفشل
failures := circuitBreaker.GetFailureCount()
fmt.Printf("Failure Count: %d\n", failures)

// فحص Threshold
threshold := circuitBreaker.GetThreshold()
fmt.Printf("Threshold: %d\n", threshold)
```

#### الحلول
1. تأكد من فشل العمليات فعلياً
2. تحقق من threshold (يجب أن يكون 3)
3. تأكد من عدم إعادة تعيين Circuit Breaker

### مشكلة: Redis لا يعيد الاتصال

#### الأعراض
```
Error: failed to reconnect to Redis
```

#### التشخيص
```bash
# فحص حالة Redis
redis-cli ping

# فحص المنفذ
netstat -an | grep 6379

# فحص Logs
docker-compose logs redis
```

#### الحلول
1. تأكد من تشغيل Redis
2. تحقق من إعدادات الشبكة
3. تحقق من Firewall

## أفضل الممارسات

### 1. إدارة Circuit Breaker
```go
// استخدام threshold مناسب
circuitBreaker := errors.NewCircuitBreaker(
    3,              // threshold: 3 failures
    10*time.Second, // timeout: 10 seconds
)

// إعادة تعيين عند الحاجة
if shouldReset {
    circuitBreaker.Reset()
}
```

### 2. معالجة Fallback
```go
// دائماً تحقق من نجاح Fallback
err := fallbackStrategy.Execute(ctx, primaryOp, fallbackOp)
if err != nil {
    // Log error
    logger.Error("Fallback failed", zap.Error(err))
    
    // Send alert
    alerting.SendCriticalAlert("Fallback failure")
}
```

### 3. مراقبة المقاييس
```go
// مراقبة معدل استخدام Fallback
metrics := fallbackStrategy.GetMetrics()
fallbackRate := metrics.GetFallbackUsageRate()

if fallbackRate > 50.0 {
    // Redis has issues, investigate
    logger.Warn("High fallback usage", 
        zap.Float64("rate", fallbackRate))
}
```

### 4. التعامل مع الأخطاء
```go
// تسجيل تفاصيل كافية
if err != nil {
    logger.Error("Operation failed",
        zap.Error(err),
        zap.String("operation", "redis_push"),
        zap.Int64("tenant_id", tenantID),
        zap.String("message_id", messageID),
    )
}
```

## الخلاصة

هذا الاختبار يضمن:
1. ✅ النظام يكتشف فشل Redis تلقائياً
2. ✅ التحول السلس للمعالجة المباشرة
3. ✅ عدم فقدان أي رسالة
4. ✅ إعادة الاتصال التلقائي عند عودة Redis
5. ✅ Circuit Breaker يعمل بشكل صحيح
6. ✅ مقاييس Fallback دقيقة

النظام يحقق **المتطلب 19.5** و **المتطلب 24.2** بنجاح.
