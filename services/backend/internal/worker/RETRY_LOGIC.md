# Retry Logic و Dead Letter Queue

## نظرة عامة

هذا المستند يشرح آلية إعادة المحاولة (Retry Logic) ونظام Dead Letter Queue (DLQ) المطبق في Worker Pool.

## المتطلبات المطبقة

- **Requirement 17.4**: إعادة محاولة الرسائل الفاشلة 3 مرات مع exponential backoff
- **Requirement 17.5**: نقل الرسائل الفاشلة نهائياً إلى Dead Letter Queue
- **Requirement 17.6**: الاحتفاظ بالرسائل في DLQ لمدة 7 أيام

## آلية Retry Logic

### عدد المحاولات

- **المحاولات الإجمالية**: 3 محاولات (المحاولة الأولى + 2 إعادة محاولة)
- **التكوين**: يتم تحديد عدد المحاولات في `WorkerPoolConfig.RetryPolicy.MaxAttempts`

### Exponential Backoff

يستخدم النظام exponential backoff لتأخير إعادة المحاولات:

```
المحاولة 1: فوري (0s)
المحاولة 2: بعد 1 ثانية
المحاولة 3: بعد 2 ثانية (إجمالي 3 ثوانٍ من المحاولة الأولى)
```

**الصيغة**: `delay = InitialInterval * (Multiplier ^ (attempt - 1))`

حيث:
- `InitialInterval = 1 second`
- `Multiplier = 2.0`
- `MaxInterval = 1 minute`

### تسجيل المحاولات

يتم تسجيل كل محاولة مع المعلومات التالية:
- رقم المحاولة الحالية
- العدد الأقصى للمحاولات
- سبب الفشل (إن وجد)
- Worker ID
- وقت المعالجة

مثال على Log:

```json
{
  "level": "info",
  "msg": "Processing message job",
  "tenant_id": "123e4567-e89b-12d3-a456-426614174000",
  "message_id": "987fcdeb-51a2-43f7-8765-123456789abc",
  "attempt": 2,
  "max_attempts": 3,
  "worker_id": 5
}
```

## Dead Letter Queue (DLQ)

### متى يتم نقل الرسالة إلى DLQ؟

يتم نقل الرسالة إلى DLQ في الحالات التالية:

1. **فشل جميع المحاولات**: عندما تفشل الرسالة في المحاولة الثالثة (النهائية)
2. **أسباب الفشل المحتملة**:
   - فشل في الحصول على بيانات الرسالة من قاعدة البيانات
   - فشل في خصم الرصيد (رصيد غير كافٍ)
   - فشل في اختيار قناة إرسال متاحة
   - فشل في إرسال الرسالة عبر WebSocket

### بنية جدول DLQ

```sql
CREATE TABLE dead_letter_queue (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,
    message_id UUID NOT NULL,
    phone_number VARCHAR(20) NOT NULL,
    content TEXT NOT NULL,
    original_status VARCHAR(20) NOT NULL,
    fail_reason TEXT NOT NULL,
    retry_count INT NOT NULL DEFAULT 0,
    cost DECIMAL(10, 4) NOT NULL DEFAULT 0,
    moved_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

### العمليات المتاحة

#### 1. نقل رسالة إلى DLQ

```go
err := dlq.MoveToDeadLetter(ctx, tenantID, messageID, reason, retryCount)
```

**الوظيفة**:
- نقل بيانات الرسالة إلى جدول `dead_letter_queue`
- تحديث حالة الرسالة في جدول `messages` إلى `DEAD_LETTER`
- تسجيل سبب الفشل وعدد المحاولات

#### 2. استرجاع رسائل DLQ

```go
messages, err := dlq.GetDeadLetterMessages(ctx, tenantID, limit)
```

**الوظيفة**:
- استرجاع رسائل DLQ لشركة معينة
- ترتيب حسب تاريخ النقل (الأحدث أولاً)
- دعم Pagination عبر `limit`

#### 3. تنظيف الرسائل القديمة

```go
rowsAffected, err := dlq.CleanupOldMessages(ctx, retentionDays)
```

**الوظيفة**:
- حذف الرسائل الأقدم من فترة الاحتفاظ المحددة
- الافتراضي: 7 أيام (حسب Requirement 17.6)
- يجب تشغيلها بشكل دوري (مثلاً: يومياً)

### مثال على الاستخدام

```go
// في Worker Pool
pool, err := NewRiverWorkerPool(config, db, logger, dependencies)
if err != nil {
    return err
}

// الحصول على DLQ
dlq := pool.GetDeadLetterQueue()

// استرجاع رسائل DLQ لشركة معينة
messages, err := dlq.GetDeadLetterMessages(ctx, tenantID, 100)
if err != nil {
    log.Error("Failed to get DLQ messages", zap.Error(err))
}

// تنظيف الرسائل القديمة (يُنفذ بشكل دوري)
rowsAffected, err := dlq.CleanupOldMessages(ctx, 7)
if err != nil {
    log.Error("Failed to cleanup old DLQ messages", zap.Error(err))
}
```

## تدفق المعالجة الكامل

```
1. استقبال Job من River Queue
   ↓
2. المحاولة 1: معالجة الرسالة
   ↓
   [نجاح] → تحديث الحالة إلى PROCESSING ✓
   [فشل] → تسجيل الخطأ
   ↓
3. المحاولة 2: إعادة المحاولة بعد 1 ثانية
   ↓
   [نجاح] → تحديث الحالة إلى PROCESSING ✓
   [فشل] → تسجيل الخطأ
   ↓
4. المحاولة 3: إعادة المحاولة بعد 2 ثانية (نهائية)
   ↓
   [نجاح] → تحديث الحالة إلى PROCESSING ✓
   [فشل] → نقل إلى DLQ
   ↓
5. Dead Letter Queue
   - تحديث حالة الرسالة إلى DEAD_LETTER
   - حفظ سبب الفشل وعدد المحاولات
   - الاحتفاظ لمدة 7 أيام
```

## المراقبة والتنبيهات

### Metrics المتاحة

- **Total Failed Jobs**: إجمالي الرسائل الفاشلة
- **DLQ Size**: عدد الرسائل في DLQ
- **Retry Attempts**: عدد محاولات إعادة الإرسال
- **Success Rate**: نسبة النجاح بعد إعادة المحاولات

### Logs المهمة

```go
// عند فشل المحاولة
logger.Error("Failed to process message",
    zap.String("tenant_id", tenantID),
    zap.String("message_id", messageID),
    zap.Int("attempt", attempt),
    zap.Error(err),
)

// عند نقل إلى DLQ
logger.Info("Message moved to dead letter queue",
    zap.String("tenant_id", tenantID),
    zap.String("message_id", messageID),
    zap.String("reason", reason),
    zap.Int("retry_count", retryCount),
)
```

## أفضل الممارسات

### 1. مراقبة حجم DLQ

- راقب عدد الرسائل في DLQ بشكل مستمر
- إذا زاد الحجم بشكل كبير، قد يشير إلى مشكلة في النظام

### 2. تحليل أسباب الفشل

- راجع `fail_reason` في DLQ بشكل دوري
- حدد الأنماط المتكررة للفشل
- اتخذ إجراءات تصحيحية

### 3. تنظيف دوري

- قم بتشغيل `CleanupOldMessages` يومياً
- احتفظ بالرسائل لمدة 7 أيام على الأقل
- يمكن زيادة الفترة حسب الحاجة

### 4. إعادة معالجة الرسائل

- راجع رسائل DLQ يدوياً
- حدد الرسائل القابلة لإعادة المعالجة
- أنشئ Jobs جديدة للرسائل المختارة

## التكوين

### Worker Pool Config

```go
config := &WorkerPoolConfig{
    WorkerCount:  50,
    QueueName:    "default",
    MaxWorkers:   50,
    PollInterval: 100 * time.Millisecond,
    JobTimeout:   5 * time.Minute,
    RetryPolicy: RetryPolicy{
        MaxAttempts:     3,              // 3 محاولات
        InitialInterval: 1 * time.Second, // تأخير أولي 1 ثانية
        MaxInterval:     1 * time.Minute, // حد أقصى 1 دقيقة
        Multiplier:      2.0,             // exponential backoff
    },
}
```

### DLQ Retention

```go
// تنظيف الرسائل الأقدم من 7 أيام
retentionDays := 7
rowsAffected, err := dlq.CleanupOldMessages(ctx, retentionDays)
```

## الاختبار

### اختبار Retry Logic

```go
// محاكاة فشل متكرر
func TestRetryLogic(t *testing.T) {
    // إنشاء job يفشل في المحاولتين الأوليين
    // التحقق من نجاحه في المحاولة الثالثة
}
```

### اختبار DLQ

```go
// محاكاة فشل جميع المحاولات
func TestDeadLetterQueue(t *testing.T) {
    // إنشاء job يفشل 3 مرات
    // التحقق من نقله إلى DLQ
    // التحقق من تحديث حالة الرسالة إلى DEAD_LETTER
}
```

## الخلاصة

- ✅ **3 محاولات** مع exponential backoff (1s, 2s)
- ✅ **تسجيل شامل** لجميع المحاولات
- ✅ **DLQ** للرسائل الفاشلة نهائياً
- ✅ **احتفاظ 7 أيام** في DLQ
- ✅ **تنظيف تلقائي** للرسائل القديمة
- ✅ **عزل تام** بين الشركات

---

**تاريخ الإنشاء**: 2026-02-25  
**الإصدار**: 1.0  
**الحالة**: مطبق ✓
