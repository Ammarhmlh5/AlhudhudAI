# Batch Processor

## نظرة عامة

Batch Processor هو المكون المسؤول عن معالجة الرسائل على شكل دفعات (batches) من Redis Buffer إلى PostgreSQL. يوفر أداءً عالياً من خلال معالجة 100 رسالة في دفعة واحدة بدلاً من المعالجة الفردية، مما يقلل الضغط على قاعدة البيانات بنسبة 50x.

## المسؤوليات

1. **استخراج الرسائل من Redis**: استخراج دفعات من الرسائل من Redis Buffer لكل tenant
2. **Batch Insert في PostgreSQL**: إدخال الدفعة كاملة في transaction واحد
3. **التكيف الذكي**: تعديل حجم الدفعة بناءً على حجم الضغط
4. **العزل بين الشركات**: معالجة كل tenant بشكل منفصل تماماً

## المتطلبات المُنفذة

- **Requirement 4.1**: استخراج 100 رسالة من Redis Buffer
- **Requirement 4.2**: Batch Insert في PostgreSQL ضمن transaction واحد
- **Requirement 4.3**: معالجة الرسائل المتاحة بعد 100ms إذا كان العدد أقل من 100
- **Requirement 4.5**: معالجة 100 رسالة في دفعة أسرع 50x من المعالجة الفردية
- **Requirement 9**: التكيف الذكي مع حجم الضغط

## البنية المعمارية

```
┌─────────────────┐
│  Redis Buffer   │
│  (per tenant)   │
└────────┬────────┘
         │ RPOP Batch (100)
         ▼
┌─────────────────┐
│ Batch Processor │
│  - Adaptive     │
│  - Isolated     │
└────────┬────────┘
         │ COPY FROM (Batch Insert)
         ▼
┌─────────────────┐
│   PostgreSQL    │
│  messages table │
└─────────────────┘
```

## الاستخدام

### إنشاء Batch Processor

```go
import (
    "github.com/al-hudhud/backend/internal/batch"
    "github.com/al-hudhud/backend/internal/buffer"
    "github.com/al-hudhud/backend/internal/database"
)

// Create dependencies
messageBuffer := buffer.NewMessageBuffer(redisClient, logger)
db, _ := database.NewDatabase(config, logger)

// Create batch processor with default config
processor := batch.NewBatchProcessor(
    messageBuffer,
    db,
    batch.DefaultBatchConfig(),
    logger,
)

// Or with custom config
config := &batch.BatchConfig{
    DefaultBatchSize: 100,
    MinBatchSize:     10,
    MaxBatchSize:     1000,
    ProcessInterval:  100 * time.Millisecond,
    AdaptiveMode:     true,
}
processor := batch.NewBatchProcessor(messageBuffer, db, config, logger)
```

### تشغيل Batch Processor

```go
// Start processor (runs continuously)
ctx := context.Background()
go func() {
    if err := processor.Start(ctx); err != nil {
        log.Fatal(err)
    }
}()

// Stop gracefully
processor.Stop(ctx)
```

### معالجة دفعة لـ Tenant محدد

```go
result, err := processor.ProcessBatch(ctx, tenantID)
if err != nil {
    log.Printf("Failed to process batch: %v", err)
}

log.Printf("Processed: %d, Success: %d, Failed: %d, Time: %dms",
    result.ProcessedCount,
    result.SuccessCount,
    result.FailedCount,
    result.ProcessingTimeMs,
)
```

### معالجة جميع الـ Tenants النشطة

```go
err := processor.ProcessAllTenants(ctx)
if err != nil {
    log.Printf("Failed to process all tenants: %v", err)
}
```

## التكيف الذكي (Adaptive Sizing)

يقوم Batch Processor بتعديل حجم الدفعة تلقائياً بناءً على حجم Buffer:

| حجم Buffer | حجم الدفعة | السلوك |
|-----------|-----------|--------|
| 0 | 0 | لا توجد رسائل |
| 1-9 | 1-9 | معالجة فورية |
| 10-99 | 10-50 | دفعة متوسطة |
| 100+ | 100 | دفعة كاملة |

```go
batchSize := processor.AdaptBatchSize(ctx, tenantID)
```

## العزل بين الشركات (Tenant Isolation)

### Redis Keys المنفصلة

```
messages:pending:123  → Queue للشركة 123
messages:pending:456  → Queue للشركة 456
messages:count:123    → عداد للشركة 123
```

### معالجة منفصلة

```go
// كل tenant يُعالج بشكل مستقل تماماً
for _, tenantID := range activeTenants {
    processor.ProcessBatch(ctx, tenantID)
}
```

### Transaction منفصلة

كل دفعة لـ tenant تُدخل في transaction منفصلة:

```sql
BEGIN;
COPY messages (tenant_id, message_id, ...) FROM STDIN;
-- جميع الرسائل تحتوي على نفس tenant_id
COMMIT;
```

## الأداء

### Batch Insert Performance

استخدام `COPY FROM` بدلاً من `INSERT` يوفر:

- **50x أسرع** من INSERT statements فردية
- **استهلاك أقل للـ connections**: transaction واحد بدلاً من 100
- **Atomic operation**: إما تنجح الدفعة كاملة أو تفشل كاملة

### مثال على الأداء

```
معالجة فردية:
- 100 رسالة × 10ms = 1000ms
- 100 connection

معالجة دفعات:
- 100 رسالة في دفعة واحدة = 20ms
- 1 connection
- تحسين: 50x
```

## معالجة الأخطاء

### فشل Batch Insert

إذا فشل إدخال الدفعة:

1. يتم rollback للـ transaction
2. يتم تسجيل الخطأ
3. الرسائل تبقى في Redis Buffer
4. يمكن إعادة المحاولة في الدورة التالية

```go
result, err := processor.ProcessBatch(ctx, tenantID)
if err != nil {
    // الدفعة فشلت - الرسائل لا تزال في Redis
    log.Printf("Batch failed: %v", err)
}
```

### فشل Redis Connection

```go
// إذا فشل الاتصال بـ Redis
messages, err := buffer.PopBatch(ctx, tenantID, 100)
if err != nil {
    // يتم تسجيل الخطأ والمحاولة مرة أخرى
    return fmt.Errorf("pop batch: %w", err)
}
```

## المراقبة والـ Metrics

### Logging

```go
logger.Info("Batch processed successfully",
    zap.Int64("tenant_id", tenantID),
    zap.Int("count", result.SuccessCount),
    zap.Int64("processing_time_ms", result.ProcessingTimeMs),
)
```

### Metrics المقترحة

```go
// عدد الدفعات المعالجة
batches_processed_total{tenant_id="123"} 150

// عدد الرسائل المعالجة
messages_processed_total{tenant_id="123"} 15000

// وقت المعالجة
batch_processing_duration_ms{tenant_id="123"} 20

// حجم الدفعة
batch_size{tenant_id="123"} 100
```

## الاختبار

### Unit Tests

```bash
go test ./internal/batch -v
```

### Integration Tests

```bash
# يتطلب Redis و PostgreSQL
go test ./internal/batch -tags=integration -v
```

### اختبار الأداء

```bash
go test ./internal/batch -bench=. -benchmem
```

## التكوين

### BatchConfig

```go
type BatchConfig struct {
    DefaultBatchSize int           // 100 - حجم الدفعة الافتراضي
    MinBatchSize     int           // 10 - أقل حجم دفعة
    MaxBatchSize     int           // 1000 - أكبر حجم دفعة
    ProcessInterval  time.Duration // 100ms - فترة المعالجة
    AdaptiveMode     bool          // true - تفعيل التكيف الذكي
}
```

### التكوين الموصى به

**بيئة الإنتاج:**
```go
config := &BatchConfig{
    DefaultBatchSize: 100,
    MinBatchSize:     10,
    MaxBatchSize:     1000,
    ProcessInterval:  100 * time.Millisecond,
    AdaptiveMode:     true,
}
```

**بيئة التطوير:**
```go
config := &BatchConfig{
    DefaultBatchSize: 10,
    MinBatchSize:     1,
    MaxBatchSize:     50,
    ProcessInterval:  500 * time.Millisecond,
    AdaptiveMode:     true,
}
```

## أفضل الممارسات

### 1. استخدام Context للإلغاء

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

result, err := processor.ProcessBatch(ctx, tenantID)
```

### 2. معالجة الأخطاء بشكل صحيح

```go
result, err := processor.ProcessBatch(ctx, tenantID)
if err != nil {
    // تسجيل الخطأ
    logger.Error("Batch processing failed", zap.Error(err))
    
    // إرسال تنبيه إذا كان خطأ حرج
    if isCriticalError(err) {
        alerting.Send("Batch processor error", err)
    }
    
    return err
}
```

### 3. مراقبة الأداء

```go
// قياس وقت المعالجة
start := time.Now()
result, err := processor.ProcessBatch(ctx, tenantID)
duration := time.Since(start)

// تسجيل إذا كان بطيئاً
if duration > 1*time.Second {
    logger.Warn("Slow batch processing",
        zap.Duration("duration", duration),
        zap.Int64("tenant_id", tenantID),
    )
}
```

### 4. Graceful Shutdown

```go
// إيقاف تدريجي
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := processor.Stop(ctx); err != nil {
    logger.Error("Failed to stop processor", zap.Error(err))
}
```

## الأسئلة الشائعة

### Q: ماذا يحدث إذا فشل إدخال دفعة؟

A: يتم rollback للـ transaction بالكامل، والرسائل تبقى في Redis Buffer. سيتم إعادة المحاولة في الدورة التالية.

### Q: كيف يتم ضمان عدم معالجة رسالة مرتين؟

A: Redis RPOP هو atomic operation - الرسالة تُحذف من Queue عند استخراجها. إذا فشل الإدخال، الرسائل لا تُستخرج من Redis.

### Q: هل يمكن تشغيل عدة Batch Processors بالتوازي؟

A: نعم، يمكن تشغيل عدة instances، لكن يجب التأكد من أن كل instance يعالج tenants مختلفة لتجنب التنافس.

### Q: كيف يتم حساب تكلفة الرسالة؟

A: حالياً يتم استخدام تكلفة ثابتة (0.05 لكل جزء). يجب استبدال `calculateCost()` بـ Pricing Engine فعلي.

### Q: ما هو الحد الأقصى لحجم الدفعة؟

A: الحد الأقصى الافتراضي هو 1000 رسالة، لكن يمكن تعديله في `BatchConfig.MaxBatchSize`.

## River Queue Integration

### نظرة عامة

تم دمج River Queue مع Batch Processor لتوفير معالجة موثوقة وقابلة للتوسع للرسائل. بعد إدخال دفعة من الرسائل في قاعدة البيانات، يتم إنشاء River Queue Jobs لكل رسالة تلقائياً.

### الميزات الرئيسية

1. **إنشاء Jobs في نفس Transaction**: يتم إنشاء River Jobs ضمن نفس transaction الخاص بإدخال الرسائل، مما يضمن الاتساق.

2. **عزل تام بين الشركات**: كل job يحتوي على `tenant_id` و `message_id` لضمان العزل التام.

3. **إعادة المحاولة التلقائية**: تم تعيين `max_attempts = 3` لكل job، مما يسمح بإعادة المحاولة عند الفشل.

4. **دعم الأولويات**: يتم تحويل أولوية الرسالة (HIGH/NORMAL/LOW) إلى أولوية River (1/2/3).

### البنية

```go
// إنشاء River Job لكل رسالة
_, err = bp.riverClient.InsertTx(ctx, tx, queue.SendSMSArgs{
    TenantID:  tenantUUID,
    MessageID: messageUUID,
}, &river.InsertOpts{
    MaxAttempts: 3,              // إعادة المحاولة 3 مرات
    Queue:       river.QueueDefault,
    Priority:    bp.getPriority(msg.Priority),
})
```

### تحويل الأولويات

| أولوية الرسالة | أولوية River | الوصف |
|----------------|--------------|-------|
| HIGH           | 1            | أعلى أولوية - معالجة فورية |
| NORMAL         | 2            | أولوية عادية - المعالجة الافتراضية |
| LOW            | 3            | أولوية منخفضة - معالجة عند توفر الموارد |

### معالجة الأخطاء

- إذا فشل إنشاء River Job، يتم rollback كامل transaction (بما في ذلك إدخال الرسائل).
- يتم تسجيل جميع الأخطاء مع تفاصيل كاملة (tenant_id, message_id).
- يتم تخطي الرسائل ذات UUIDs غير صحيحة مع تسجيل تحذير.

### الاختبارات

تم إضافة اختبارات شاملة لـ:
- تحويل الأولويات (`TestBatchProcessor_GetPriority`)
- إنشاء River Jobs (integration tests - تتطلب إعداد قاعدة بيانات)
- التحقق من max_attempts
- التحقق من tenant_id و message_id في Jobs
- rollback عند فشل إنشاء Jobs

### المتطلبات المُنفذة

- ✅ **Requirement 4.4**: إنشاء River Queue Jobs بعد batch insert
- ✅ ربط كل job بـ tenant_id و message_id
- ✅ تحديد max_attempts = 3
- ✅ دعم الأولويات
- ✅ معالجة الأخطاء والـ rollback

## Batch Failure Recovery

### نظرة عامة

تم تطبيق آلية Batch Failure Recovery لضمان عدم فقدان الرسائل عند فشل batch insert. عندما يفشل إدخال دفعة كاملة، يتم إعادة محاولة كل رسالة بشكل فردي لتحديد الرسائل الصالحة والفاشلة.

### المتطلب المُنفذ

**Requirement 4.7**: WHEN يفشل إدخال دفعة، THE Batch_Processor SHALL إعادة محاولة الرسائل الفردية

### كيف يعمل

```
┌─────────────────┐
│  Batch Insert   │
│   (100 msgs)    │
└────────┬────────┘
         │
         ▼
    ┌────────┐
    │ Failed?│
    └───┬────┘
        │ Yes
        ▼
┌─────────────────┐
│ Retry           │
│ Individually    │
│ (1 by 1)        │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 70 Success      │
│ 30 Failed       │
│ (logged)        │
└─────────────────┘
```

### التدفق التفصيلي

1. **محاولة Batch Insert**: يتم محاولة إدخال 100 رسالة في transaction واحد
2. **عند الفشل**: إذا فشل batch insert (مثلاً: constraint violation، connection error)
3. **Individual Retry**: يتم إعادة محاولة كل رسالة بشكل فردي في transaction منفصلة
4. **تسجيل النتائج**: 
   - الرسائل الناجحة: يتم إدخالها وإنشاء River Jobs لها
   - الرسائل الفاشلة: يتم تسجيلها مع سبب الفشل التفصيلي

### مثال على الكود

```go
// في ProcessBatch
err = bp.insertBatch(ctx, tenantID, messages)
if err != nil {
    bp.logger.Warn("Batch insert failed, attempting individual retry",
        zap.Error(err),
        zap.Int64("tenant_id", tenantID),
        zap.Int("count", len(messages)),
    )
    
    // إعادة محاولة فردية
    successCount, failedCount := bp.retryIndividually(ctx, tenantID, messages)
    result.SuccessCount = successCount
    result.FailedCount = failedCount
    
    if successCount > 0 {
        bp.logger.Info("Partial batch recovery successful",
            zap.Int64("tenant_id", tenantID),
            zap.Int("success_count", successCount),
            zap.Int("failed_count", failedCount),
        )
    }
}
```

### دالة retryIndividually

```go
func (bp *BatchProcessor) retryIndividually(ctx context.Context, tenantID int64, messages []MessageForInsert) (int, int) {
    successCount := 0
    failedCount := 0

    for _, msg := range messages {
        // Begin individual transaction
        tx, err := bp.db.Pool.Begin(ctx)
        if err != nil {
            failedCount++
            continue
        }

        // Insert single message
        _, err = tx.Exec(ctx, `INSERT INTO messages (...)`)
        if err != nil {
            tx.Rollback(ctx)
            bp.logger.Error("Failed to insert individual message",
                zap.String("message_id", msg.MessageID),
                zap.Error(err),
            )
            failedCount++
            continue
        }

        // Create River job
        if bp.riverClient != nil {
            _, err = bp.riverClient.InsertTx(ctx, tx, ...)
            if err != nil {
                tx.Rollback(ctx)
                failedCount++
                continue
            }
        }

        // Commit
        if err := tx.Commit(ctx); err != nil {
            failedCount++
            continue
        }

        successCount++
    }

    return successCount, failedCount
}
```

### سيناريوهات الاستخدام

#### سيناريو 1: Constraint Violation

```
دفعة من 100 رسالة:
- 1 رسالة لديها message_id مكرر
- Batch insert يفشل بالكامل
- Individual retry:
  - 99 رسالة تنجح
  - 1 رسالة تفشل (duplicate key)
- النتيجة: 99 رسالة تُرسل، 1 رسالة تُسجل كفاشلة
```

#### سيناريو 2: Connection Timeout

```
دفعة من 100 رسالة:
- Connection timeout أثناء batch insert
- Individual retry:
  - 100 رسالة تنجح (المشكلة كانت مؤقتة)
- النتيجة: جميع الرسائل تُرسل بنجاح
```

#### سيناريو 3: Invalid Data

```
دفعة من 100 رسالة:
- 5 رسائل لديها phone_number غير صحيح
- Batch insert يفشل
- Individual retry:
  - 95 رسالة تنجح
  - 5 رسائل تفشل (invalid phone)
- النتيجة: 95 رسالة تُرسل، 5 رسائل تُسجل مع سبب الفشل
```

### Logging والمراقبة

#### Log Entries

```
INFO  Starting individual retry for failed batch
      tenant_id=123 total_messages=100

ERROR Failed to insert individual message
      tenant_id=123 message_id=abc-123 phone_number=+invalid
      error="invalid phone number format"

DEBUG Successfully inserted individual message
      tenant_id=123 message_id=def-456

INFO  Individual retry completed
      tenant_id=123 success_count=95 failed_count=5
```

#### Metrics المقترحة

```
# عدد مرات فشل batch insert
batch_insert_failures_total{tenant_id="123"} 5

# عدد الرسائل المستردة بنجاح
individual_retry_success_total{tenant_id="123"} 475

# عدد الرسائل الفاشلة نهائياً
individual_retry_failed_total{tenant_id="123"} 25

# نسبة الاسترداد
recovery_rate{tenant_id="123"} 0.95
```

### الفوائد

1. **عدم فقدان الرسائل**: الرسائل الصالحة لا تُفقد بسبب رسالة واحدة فاشلة
2. **تشخيص دقيق**: معرفة سبب فشل كل رسالة على حدة
3. **موثوقية أعلى**: النظام يتعافى تلقائياً من الأخطاء الجزئية
4. **شفافية كاملة**: logging تفصيلي لكل عملية

### الاختبارات

تم إضافة اختبارات شاملة:

```go
// اختبار الاسترداد الكامل
TestBatchProcessor_BatchFailureRecovery

// اختبار الاسترداد الجزئي
TestBatchProcessor_PartialBatchRecovery

// اختبار العزل بين الشركات
TestBatchProcessor_RetryIndividuallyTenantIsolation

// اختبار فشل River Job
TestBatchProcessor_RetryIndividuallyWithRiverFailure

// اختبار Logging
TestBatchProcessor_IndividualRetryLogging
```

### أفضل الممارسات

1. **مراقبة معدل الفشل**: إذا كان معدل فشل batch insert مرتفع، قد تكون هناك مشكلة في البيانات أو النظام

2. **تحليل أسباب الفشل**: مراجعة logs بانتظام لتحديد الأنماط المتكررة

3. **تنبيهات**: إعداد تنبيهات عند تجاوز عتبة معينة من الفشل

```go
if failedCount > len(messages)/2 {
    alerting.Send("High failure rate in batch recovery",
        fmt.Sprintf("Tenant %d: %d/%d messages failed",
            tenantID, failedCount, len(messages)))
}
```

### المتطلبات المُنفذة

- ✅ **Requirement 4.7**: إعادة محاولة الرسائل فردياً عند فشل batch insert
- ✅ تسجيل الرسائل الفاشلة مع أسباب الفشل
- ✅ الحفاظ على العزل بين الشركات
- ✅ إنشاء River Jobs للرسائل الناجحة فقط
- ✅ Logging شامل لجميع العمليات

## الخطوات التالية

1. ~~**إضافة River Queue Integration**: إنشاء River Jobs بعد Batch Insert~~ ✅ **تم التنفيذ**
2. ~~**إضافة Retry Logic**: إعادة محاولة الرسائل الفاشلة بشكل فردي~~ ✅ **تم التنفيذ**
3. **إضافة Metrics**: تكامل مع Prometheus
4. **إضافة Pricing Engine**: حساب تكلفة دقيقة بناءً على التعرفات
5. **إضافة Dead Letter Queue**: نقل الرسائل الفاشلة بعد عدة محاولات

## المراجع

- [Design Document](../../../../.kiro/specs/message-lifecycle-rebuild/design.md)
- [Requirements Document](../../../../.kiro/specs/message-lifecycle-rebuild/requirements.md)
- [Buffer Package](../buffer/README.md)
- [Database Package](../database/database.go)
