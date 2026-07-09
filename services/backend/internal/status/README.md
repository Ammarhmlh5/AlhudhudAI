# Status Report Handler

## نظرة عامة

Status Report Handler هو المكون المسؤول عن استقبال ومعالجة تقارير حالة الرسائل من الأجهزة المحمولة. يضمن هذا المكون:

- **Idempotency**: عدم معالجة نفس التقرير مرتين
- **Atomic Operations**: تحديث الحالة والمحاسبة بشكل ذري
- **Tenant Isolation**: عزل تام بين الشركات
- **Automatic Refund**: استرداد الرصيد تلقائياً للرسائل الفاشلة

## المسؤوليات

1. **استقبال تقارير الحالة**: من WebSocket Hub
2. **التحقق من Idempotency**: منع المعالجة المكررة
3. **تحديث حالة الرسائل**: في قاعدة البيانات
4. **تسوية المحاسبة**: 
   - استرداد الرصيد للرسائل الفاشلة (FAILED)
   - تحديث حالة المعاملة للرسائل الناجحة (SENT/DELIVERED)

## حالات الرسائل المدعومة

- `BUFFERED`: الرسالة محفوظة في الجهاز
- `SENT`: الرسالة أُرسلت عبر الشبكة
- `DELIVERED`: الرسالة وصلت للمستلم
- `FAILED`: فشل إرسال الرسالة

## البنية المعمارية

```
┌─────────────────┐
│ Mobile Gateway  │
└────────┬────────┘
         │ Status Report
         ↓
┌─────────────────┐
│ WebSocket Hub   │
└────────┬────────┘
         │
         ↓
┌─────────────────────────────────────┐
│   Status Report Handler             │
│                                     │
│  1. Check Idempotency (Redis)      │
│  2. Update Message Status (DB)     │
│  3. Settle Accounting              │
│     - FAILED → Refund Balance      │
│     - SENT/DELIVERED → Complete    │
└─────────────────────────────────────┘
```

## آلية Idempotency

### مفتاح Idempotency

```
idempotency:{tenant_id}:{message_id}:{status}
```

### مثال

```
idempotency:1:550e8400-e29b-41d4-a716-446655440000:SENT
```

### TTL

24 ساعة - بعدها يُحذف المفتاح تلقائياً

### كيف يعمل

1. عند استقبال تقرير، يتم توليد مفتاح idempotency
2. التحقق من وجود المفتاح في Redis
3. إذا موجود → تخطي المعالجة (already processed)
4. إذا غير موجود → حفظ المفتاح ومعالجة التقرير

## تسوية المحاسبة

### للرسائل الفاشلة (FAILED)

```sql
-- 1. الحصول على تكلفة الرسالة
SELECT cost, tenant_id FROM messages 
WHERE message_id = ? AND tenant_id = ?;

-- 2. استرداد الرصيد بشكل ذري
UPDATE tenants 
SET balance = balance + cost
WHERE id = tenant_id
RETURNING balance, balance - cost;

-- 3. تسجيل في balance_history
INSERT INTO balance_history (
    tenant_id, message_id, old_balance, new_balance,
    amount, type, reason, created_at
) VALUES (?, ?, ?, ?, ?, 'REFUND', 'MESSAGE_FAILED', NOW());

-- 4. تحديث accounting_transaction
UPDATE accounting_transactions 
SET status = 'REFUNDED'
WHERE message_id = ? AND tenant_id = ?;
```

### للرسائل الناجحة (SENT/DELIVERED)

```sql
-- تحديث حالة المعاملة إلى COMPLETED
UPDATE accounting_transactions 
SET status = 'COMPLETED'
WHERE message_id = ? AND tenant_id = ?;
```

## العزل بين الشركات (Tenant Isolation)

جميع العمليات تحتوي على `WHERE tenant_id = ?` لضمان:

- عدم الوصول لبيانات شركة أخرى
- عدم تأثير عمليات شركة على أخرى
- حماية الرصيد المالي لكل شركة

## الاستخدام

### إنشاء Handler

```go
import (
    "database/sql"
    "github.com/go-redis/redis/v8"
    "go.uber.org/zap"
    "your-project/internal/status"
)

// إنشاء handler
handler := status.NewHandler(db, redisClient, logger)
```

### معالجة تقرير

```go
report := status.StatusReport{
    MessageID:  "550e8400-e29b-41d4-a716-446655440000",
    TenantID:   1,
    ChannelID:  5,
    Status:     status.StatusSent,
    Timestamp:  time.Now(),
}

err := handler.ProcessReport(ctx, report)
if err != nil {
    log.Error("Failed to process report", zap.Error(err))
}
```

### التحقق من Idempotency

```go
key := "idempotency:1:550e8400-e29b-41d4-a716-446655440000:SENT"
exists, err := handler.CheckIdempotency(ctx, key)
if err != nil {
    log.Error("Failed to check idempotency", zap.Error(err))
}

if exists {
    log.Info("Report already processed")
}
```

## معالجة الأخطاء

### أنواع الأخطاء

1. **Redis Errors**: فشل في التحقق من/حفظ idempotency key
2. **Database Errors**: فشل في تحديث الحالة أو المحاسبة
3. **Transaction Errors**: فشل في commit/rollback
4. **Not Found Errors**: رسالة غير موجودة أو محذوفة

### استراتيجية المعالجة

- جميع الأخطاء يتم تسجيلها مع تفاصيل كاملة
- Transaction rollback تلقائي عند الفشل
- إرجاع أخطاء واضحة للمستدعي

## الاختبارات

### Unit Tests

```bash
go test ./internal/status -v
```

### اختبارات التغطية

```bash
go test ./internal/status -cover
```

### اختبارات مع Mock

الاختبارات تستخدم:
- `sqlmock` لـ database mocking
- `miniredis` لـ Redis mocking (يُنصح به)

## المقاييس والمراقبة

### Logs

جميع العمليات يتم تسجيلها مع:
- `message_id`
- `tenant_id`
- `status`
- `idempotency_key`
- أي أخطاء

### مستويات Log

- `INFO`: معالجة ناجحة، تخطي بسبب idempotency
- `DEBUG`: تفاصيل التحديثات
- `ERROR`: أي فشل في المعالجة

## الأداء

### زمن المعالجة المتوقع

- **BUFFERED**: ~10-20ms
- **SENT/DELIVERED**: ~15-25ms (تحديث accounting)
- **FAILED**: ~30-50ms (refund + accounting + history)

### التحسينات

- استخدام Redis للـ idempotency (O(1))
- Transactions لضمان الذرية
- Prepared statements للأداء

## Retry Logic مع Exponential Backoff

### المتطلب 2.7

> IF فشلت معالجة التقرير، THE Status_Report_Handler SHALL إعادة المحاولة مع Exponential Backoff

### التطبيق

تم تطبيق retry logic ذكي يميز بين الأخطاء المؤقتة والدائمة:

#### المواصفات

- **حد أقصى للمحاولات**: 3 محاولات
- **Exponential Backoff**: 1s, 2s, 4s
- **إجمالي وقت الانتظار**: 7 ثوانٍ كحد أقصى

#### الأخطاء القابلة لإعادة المحاولة (Retryable)

- أخطاء الاتصال بقاعدة البيانات (connection refused, timeout)
- أخطاء الشبكة (network errors)
- Deadlocks
- أخطاء Transaction المؤقتة
- أخطاء SQL العامة

#### الأخطاء غير القابلة لإعادة المحاولة (Non-retryable)

- رسالة غير موجودة (message not found)
- رسالة محذوفة (already deleted)
- حالة غير صحيحة (unknown status)
- بيانات غير صحيحة (invalid data)
- تقرير معالج مسبقاً (already processed)

### الاستخدام

```go
// استخدام ProcessReportWithRetry للحصول على retry logic تلقائي
err := handler.ProcessReportWithRetry(ctx, report)
if err != nil {
    // فشلت جميع المحاولات (3 محاولات)
    log.Error("Failed to process report after retries", zap.Error(err))
}

// أو استخدام ProcessReport مباشرة بدون retry
err := handler.ProcessReport(ctx, report)
if err != nil {
    // فشلت المحاولة الأولى فقط
    log.Error("Failed to process report", zap.Error(err))
}
```

### مثال على التدفق

```
Attempt 1: Failed (connection timeout) → Wait 1s
Attempt 2: Failed (deadlock) → Wait 2s
Attempt 3: Success ✓
```

### Logs

```
INFO  Processing status report (attempt=1/3)
WARN  Status report processing failed, will retry (backoff=1s)
INFO  Waiting before retry (next_attempt=2)
INFO  Processing status report (attempt=2/3)
INFO  Status report processed successfully after retry (attempt=2)
```

## المتطلبات المحققة

- ✅ **المتطلب 2.1**: تحديث حالة الرسائل عند SENT/DELIVERED
- ✅ **المتطلب 2.2**: استرداد الرصيد عند FAILED
- ✅ **المتطلب 2.3**: Idempotency لمنع المعالجة المكررة
- ✅ **المتطلب 2.4**: نفس النتيجة عند إعادة المعالجة
- ✅ **المتطلب 2.5**: Atomic operations للاسترداد
- ✅ **المتطلب 2.6**: تسجيل في Audit Trail (balance_history)
- ✅ **المتطلب 2.7**: إعادة المحاولة مع Exponential Backoff

## التطوير المستقبلي

### المخطط

1. **Batch Processing**: معالجة تقارير متعددة دفعة واحدة
2. **Metrics**: Prometheus metrics للمراقبة
3. **Webhooks**: إشعار العملاء بتغيير الحالة

### التحسينات المقترحة

- استخدام connection pool للأداء
- Caching لبيانات الرسائل المتكررة
- Async processing للتقارير غير الحرجة
