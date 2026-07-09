# دليل سريع - اختبار Failed Message Handling

## نظرة عامة

هذا الدليل يشرح كيفية تشغيل اختبارات معالجة الرسائل الفاشلة (Failed Message Handling Tests).

## السيناريوهات المختبرة

### 1. Temporary Failure (فشل مؤقت)
- **الوصف**: رسالة تفشل في المحاولة الأولى ثم تنجح في إعادة المحاولة
- **التحقق من**:
  - ✅ إعادة المحاولة تعمل بشكل صحيح
  - ✅ الرصيد يبقى مخصوماً (لأن الرسالة نجحت في النهاية)
  - ✅ حالة المعاملة المحاسبية: COMPLETED
  - ✅ عدد المحاولات مسجل بشكل صحيح

### 2. Permanent Failure (فشل دائم)
- **الوصف**: رسالة تفشل في جميع المحاولات الثلاث
- **التحقق من**:
  - ✅ إعادة المحاولة 3 مرات بالضبط
  - ✅ استخدام exponential backoff (1s, 2s, 4s)
  - ✅ نقل الرسالة إلى Dead Letter Queue (DLQ)
  - ✅ استرداد الرصيد بالكامل
  - ✅ حالة المعاملة المحاسبية: REFUNDED
  - ✅ تسجيل DEDUCT و REFUND في balance_history

### 3. Channel Unavailable (عدم توفر قناة)
- **الوصف**: رسالة لا يمكن إرسالها بسبب عدم وجود قنوات متاحة
- **التحقق من**:
  - ✅ تحديث حالة الرسالة إلى WAITING
  - ✅ عدم خصم الرصيد (لأن الرسالة لم تُرسل)
  - ✅ إعادة المحاولة عند توفر قناة
  - ✅ إكمال الإرسال بنجاح بعد توفر القناة

### 4. Network Error (خطأ في الشبكة)
- **الوصف**: رسالة تفشل بسبب مشاكل في الشبكة مع exponential backoff
- **التحقق من**:
  - ✅ تطبيق exponential backoff بين المحاولات
  - ✅ تسجيل جميع المحاولات والأخطاء
  - ✅ النجاح في المحاولة الثالثة
  - ✅ الرصيد يبقى مخصوماً

## المتطلبات

### 1. قاعدة البيانات
```bash
# تأكد من تشغيل PostgreSQL
docker-compose up -d postgres

# تأكد من وجود قاعدة بيانات الاختبار
psql -U postgres -c "CREATE DATABASE sms_gateway_test;"
```

### 2. Redis
```bash
# تأكد من تشغيل Redis
docker-compose up -d redis
```

### 3. تشغيل Migrations
```bash
cd services/backend
go run cmd/migrate/main.go -database sms_gateway_test
```

## تشغيل الاختبارات

### تشغيل جميع اختبارات Failed Message
```bash
cd services/backend
go test -v ./internal/integration -run TestFailedMessage
```

### تشغيل سيناريو محدد

#### 1. Temporary Failure
```bash
go test -v ./internal/integration -run TestFailedMessage_TemporaryFailure
```

#### 2. Permanent Failure
```bash
go test -v ./internal/integration -run TestFailedMessage_PermanentFailure
```

#### 3. Channel Unavailable
```bash
go test -v ./internal/integration -run TestFailedMessage_ChannelUnavailable
```

#### 4. Network Error
```bash
go test -v ./internal/integration -run TestFailedMessage_NetworkError
```

#### 5. جميع السيناريوهات
```bash
go test -v ./internal/integration -run TestFailedMessage_AllScenarios
```

## النتائج المتوقعة

### نجاح الاختبار
```
=== RUN   TestFailedMessage_TemporaryFailure
=== Test Scenario: Temporary Failure with Successful Retry ===
Step 1: Sending message
✓ Message queued: xxx
Step 2: Processing batch
✓ Message in database
Step 3: Deducting balance
✓ Balance deducted: 100.0000 → 99.9500
Step 4: Simulating first attempt failure (temporary network error)
✓ First attempt failed, retry scheduled
Step 5: Simulating successful retry (attempt 2)
✓ Retry succeeded, status report processed
Step 6: Verifying final state
✓ Message status: SENT (attempts: 2)
✓ Balance: 99.9500 (deducted, not refunded)
✓ Transaction status: COMPLETED
✓ Balance history: 1 DEDUCT, 0 REFUND
✅ Temporary failure scenario completed successfully!
--- PASS: TestFailedMessage_TemporaryFailure (0.15s)
PASS
```

### فشل الاختبار
إذا فشل أي اختبار، ستظهر رسالة خطأ واضحة تشير إلى المشكلة:
```
--- FAIL: TestFailedMessage_PermanentFailure (0.20s)
    failed_message_test.go:XXX: Balance should be fully refunded
        Expected: 100.0000
        Actual: 99.9500
```

## استكشاف الأخطاء

### 1. خطأ في الاتصال بقاعدة البيانات
```
Error: Failed to connect to database
```
**الحل**: تأكد من تشغيل PostgreSQL وأن قاعدة البيانات موجودة

### 2. خطأ في الاتصال بـ Redis
```
Error: Failed to connect to Redis
```
**الحل**: تأكد من تشغيل Redis على المنفذ 6379

### 3. جداول غير موجودة
```
Error: relation "messages" does not exist
```
**الحل**: قم بتشغيل migrations على قاعدة بيانات الاختبار

### 4. بيانات قديمة تؤثر على الاختبار
```
Error: Expected 1 message, found 5
```
**الحل**: الاختبارات تقوم بتنظيف البيانات تلقائياً، لكن يمكنك التنظيف يدوياً:
```sql
-- في قاعدة بيانات الاختبار
DELETE FROM balance_history;
DELETE FROM accounting_transactions;
DELETE FROM river_job;
DELETE FROM messages;
DELETE FROM channels;
DELETE FROM tenants;
```

## التحقق من النتائج يدوياً

### 1. فحص الرسائل في DLQ
```sql
SELECT * FROM dead_letter_queue 
WHERE created_at > NOW() - INTERVAL '1 hour'
ORDER BY created_at DESC;
```

### 2. فحص balance_history
```sql
SELECT 
    tenant_id,
    message_id,
    old_balance,
    new_balance,
    amount,
    type,
    reason,
    created_at
FROM balance_history
WHERE created_at > NOW() - INTERVAL '1 hour'
ORDER BY created_at DESC;
```

### 3. فحص accounting_transactions
```sql
SELECT 
    tenant_id,
    message_id,
    amount,
    status,
    transaction_type,
    created_at
FROM accounting_transactions
WHERE created_at > NOW() - INTERVAL '1 hour'
ORDER BY created_at DESC;
```

### 4. فحص محاولات الرسائل
```sql
SELECT 
    message_id,
    status,
    attempt_count,
    last_error,
    created_at,
    updated_at
FROM messages
WHERE created_at > NOW() - INTERVAL '1 hour'
ORDER BY created_at DESC;
```

## المقاييس المهمة

### 1. معدل النجاح
- **المتوقع**: 100% من الاختبارات تنجح
- **الحد الأدنى المقبول**: 95%

### 2. وقت التنفيذ
- **Temporary Failure**: ~0.15 ثانية
- **Permanent Failure**: ~0.25 ثانية
- **Channel Unavailable**: ~0.20 ثانية
- **Network Error**: ~0.20 ثانية
- **All Scenarios**: ~0.80 ثانية

### 3. تغطية المتطلبات
- ✅ المتطلب 2.2 - استرداد الرصيد عند الفشل
- ✅ المتطلب 17.4 - إعادة المحاولة 3 مرات
- ✅ المتطلب 17.5 - نقل إلى DLQ بعد الفشل النهائي

## التكامل مع CI/CD

### GitHub Actions
```yaml
- name: Run Failed Message Tests
  run: |
    cd services/backend
    go test -v ./internal/integration -run TestFailedMessage
```

### GitLab CI
```yaml
test:failed-messages:
  script:
    - cd services/backend
    - go test -v ./internal/integration -run TestFailedMessage
```

## الخطوات التالية

بعد نجاح هذه الاختبارات:
1. ✅ المهمة 15.3 مكتملة
2. ➡️ الانتقال إلى المهمة 15.4 (Multi-Tenant Scenarios)
3. ➡️ الانتقال إلى المهمة 15.5 (WebSocket Reconnection)

## الدعم

إذا واجهت أي مشاكل:
1. راجع ملف `FAILED_MESSAGE_TEST_GUIDE.md` للتفاصيل الكاملة
2. تحقق من logs في `services/backend/logs/`
3. راجع ملف `README.md` في مجلد integration tests

---

**آخر تحديث**: 2024
**الإصدار**: 1.0
