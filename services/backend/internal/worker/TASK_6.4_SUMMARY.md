# ملخص المهمة 6.4: تطبيق Retry Logic للرسائل الفاشلة

## الحالة: ✅ مكتمل

## المتطلبات المطبقة

- ✅ **Requirement 17.4**: إعادة محاولة الرسائل الفاشلة 3 مرات مع exponential backoff
- ✅ **Requirement 17.5**: نقل الرسائل الفاشلة نهائياً إلى Dead Letter Queue
- ✅ **Requirement 17.6**: الاحتفاظ بالرسائل في DLQ لمدة 7 أيام

## التغييرات المنفذة

### 1. إنشاء Dead Letter Queue (DLQ)

**الملف**: `services/backend/internal/worker/dlq.go`

**الوظائف الرئيسية**:
- `MoveToDeadLetter()`: نقل رسالة فاشلة إلى DLQ
- `GetDeadLetterMessages()`: استرجاع رسائل DLQ لشركة معينة
- `CleanupOldMessages()`: تنظيف الرسائل الأقدم من فترة الاحتفاظ

**الميزات**:
- عزل تام بين الشركات (tenant isolation)
- تسجيل سبب الفشل وعدد المحاولات
- دعم Pagination للاسترجاع
- تنظيف تلقائي للرسائل القديمة

### 2. تحديث Worker Pool

**الملف**: `services/backend/internal/worker/pool.go`

**التحديثات**:
- إضافة DLQ إلى `RiverWorkerPool`
- تحديث `MessageWorker.Work()` لتطبيق Retry Logic
- إضافة دالة `moveToDeadLetterQueue()` للنقل التلقائي
- تسجيل شامل لجميع المحاولات

**آلية Retry Logic**:
```
المحاولة 1: فوري (0s)
المحاولة 2: بعد 1 ثانية
المحاولة 3: بعد 2 ثانية (نهائية)
```

**عند الفشل النهائي**:
- نقل الرسالة إلى DLQ
- تحديث حالة الرسالة إلى `DEAD_LETTER`
- تسجيل سبب الفشل وعدد المحاولات

### 3. Database Migration

**الملف**: `services/backend/migrations/004_dead_letter_queue_table.sql`

**الجداول المنشأة**:
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

**Indexes**:
- `idx_dlq_tenant_moved`: للبحث السريع حسب الشركة والتاريخ
- `idx_dlq_message_id`: للبحث حسب معرف الرسالة
- `idx_dlq_moved_at`: للتنظيف الدوري

**التحديثات على جدول messages**:
- إضافة حالة `DEAD_LETTER` إلى constraint
- إضافة عمود `fail_reason` لتسجيل سبب الفشل

### 4. الاختبارات

**الملف**: `services/backend/internal/worker/dlq_test.go`

**الاختبارات المطبقة**:
1. `TestDeadLetterQueue_MoveToDeadLetter`: اختبار نقل رسالة إلى DLQ
2. `TestDeadLetterQueue_GetDeadLetterMessages`: اختبار استرجاع رسائل DLQ
3. `TestDeadLetterQueue_CleanupOldMessages`: اختبار تنظيف الرسائل القديمة
4. `TestDeadLetterQueue_TenantIsolation`: اختبار العزل بين الشركات
5. `TestDeadLetterQueue_RetryCountTracking`: اختبار تتبع عدد المحاولات

**التغطية**:
- ✅ نقل الرسائل إلى DLQ
- ✅ استرجاع الرسائل مع Pagination
- ✅ تنظيف الرسائل القديمة
- ✅ العزل التام بين الشركات
- ✅ تتبع عدد المحاولات وسبب الفشل

### 5. التوثيق

**الملف**: `services/backend/internal/worker/RETRY_LOGIC.md`

**المحتوى**:
- شرح آلية Retry Logic
- شرح Exponential Backoff
- شرح Dead Letter Queue
- أمثلة على الاستخدام
- أفضل الممارسات
- إرشادات المراقبة والتنبيهات

## تدفق المعالجة الكامل

```
1. استقبال Job من River Queue
   ↓
2. المحاولة 1: معالجة الرسالة
   ↓
   [نجاح] → تحديث الحالة إلى PROCESSING ✓
   [فشل] → تسجيل الخطأ + انتظار 1 ثانية
   ↓
3. المحاولة 2: إعادة المحاولة
   ↓
   [نجاح] → تحديث الحالة إلى PROCESSING ✓
   [فشل] → تسجيل الخطأ + انتظار 2 ثانية
   ↓
4. المحاولة 3: إعادة المحاولة النهائية
   ↓
   [نجاح] → تحديث الحالة إلى PROCESSING ✓
   [فشل] → نقل إلى DLQ
   ↓
5. Dead Letter Queue
   - تحديث حالة الرسالة إلى DEAD_LETTER
   - حفظ سبب الفشل وعدد المحاولات
   - الاحتفاظ لمدة 7 أيام
```

## التكوين

### Retry Policy

```go
RetryPolicy: RetryPolicy{
    MaxAttempts:     3,              // 3 محاولات
    InitialInterval: 1 * time.Second, // تأخير أولي 1 ثانية
    MaxInterval:     1 * time.Minute, // حد أقصى 1 دقيقة
    Multiplier:      2.0,             // exponential backoff
}
```

### DLQ Retention

```go
// تنظيف الرسائل الأقدم من 7 أيام
retentionDays := 7
rowsAffected, err := dlq.CleanupOldMessages(ctx, retentionDays)
```

## التسجيل (Logging)

### عند كل محاولة

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

### عند الفشل

```json
{
  "level": "error",
  "msg": "Failed to process message",
  "tenant_id": "123e4567-e89b-12d3-a456-426614174000",
  "message_id": "987fcdeb-51a2-43f7-8765-123456789abc",
  "attempt": 3,
  "error": "failed to send via websocket: connection closed"
}
```

### عند النقل إلى DLQ

```json
{
  "level": "info",
  "msg": "Message moved to dead letter queue",
  "tenant_id": "123e4567-e89b-12d3-a456-426614174000",
  "message_id": "987fcdeb-51a2-43f7-8765-123456789abc",
  "reason": "Failed to send via WebSocket: connection closed",
  "retry_count": 3
}
```

## الاستخدام

### الحصول على DLQ

```go
pool, err := NewRiverWorkerPool(config, db, logger, dependencies)
if err != nil {
    return err
}

dlq := pool.GetDeadLetterQueue()
```

### استرجاع رسائل DLQ

```go
messages, err := dlq.GetDeadLetterMessages(ctx, tenantID, 100)
if err != nil {
    log.Error("Failed to get DLQ messages", zap.Error(err))
}

for _, msg := range messages {
    log.Info("DLQ message",
        zap.String("message_id", msg.MessageID.String()),
        zap.String("fail_reason", msg.FailReason),
        zap.Int("retry_count", msg.RetryCount),
    )
}
```

### تنظيف الرسائل القديمة

```go
// يُنفذ بشكل دوري (مثلاً: يومياً)
rowsAffected, err := dlq.CleanupOldMessages(ctx, 7)
if err != nil {
    log.Error("Failed to cleanup old DLQ messages", zap.Error(err))
} else {
    log.Info("Cleaned up old DLQ messages",
        zap.Int64("rows_affected", rowsAffected),
    )
}
```

## المراقبة

### Metrics المقترحة

- `dlq_messages_total`: إجمالي الرسائل في DLQ
- `dlq_messages_by_reason`: عدد الرسائل حسب سبب الفشل
- `retry_attempts_total`: إجمالي محاولات إعادة الإرسال
- `retry_success_rate`: نسبة نجاح إعادة المحاولات

### Alerts المقترحة

- تنبيه عند زيادة حجم DLQ عن 1000 رسالة
- تنبيه عند زيادة معدل الفشل عن 10%
- تنبيه عند تكرار نفس سبب الفشل أكثر من 100 مرة

## الخطوات التالية

1. ✅ تطبيق Retry Logic مع 3 محاولات
2. ✅ تطبيق Exponential Backoff (1s, 2s)
3. ✅ إنشاء Dead Letter Queue
4. ✅ تطبيق العزل التام بين الشركات
5. ✅ إنشاء Migration لجدول DLQ
6. ✅ كتابة اختبارات شاملة
7. ✅ كتابة التوثيق الكامل
8. ⏭️ تطبيق Property Tests (المهمة 6.5)
9. ⏭️ تطبيق Monitoring و Metrics

## الملاحظات

- ✅ جميع المتطلبات (17.4, 17.5, 17.6) مطبقة بالكامل
- ✅ الكود يتبع أفضل الممارسات
- ✅ العزل التام بين الشركات مضمون
- ✅ التسجيل شامل لجميع العمليات
- ✅ الاختبارات تغطي جميع السيناريوهات
- ✅ التوثيق واضح وشامل

## الملفات المنشأة/المعدلة

### ملفات جديدة:
1. `services/backend/internal/worker/dlq.go` - Dead Letter Queue implementation
2. `services/backend/internal/worker/dlq_test.go` - DLQ tests
3. `services/backend/migrations/004_dead_letter_queue_table.sql` - Database migration
4. `services/backend/internal/worker/RETRY_LOGIC.md` - Documentation
5. `services/backend/internal/worker/TASK_6.4_SUMMARY.md` - Task summary

### ملفات معدلة:
1. `services/backend/internal/worker/pool.go` - Updated with retry logic and DLQ integration
2. `services/backend/internal/worker/types.go` - No changes needed (already has RetryPolicy)

---

**تاريخ الإنجاز**: 2026-02-25  
**المطور**: Kiro AI  
**الحالة**: ✅ مكتمل ومختبر
