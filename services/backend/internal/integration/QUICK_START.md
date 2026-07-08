# Quick Start - End-to-End Integration Tests

## دليل البدء السريع

### الخطوة 1: تشغيل البنية التحتية

```bash
# تشغيل PostgreSQL و Redis
docker-compose up -d postgres redis

# انتظر حتى تصبح الخدمات جاهزة (5-10 ثوانٍ)
sleep 5
```

### الخطوة 2: تطبيق Migrations

```bash
# تطبيق جميع migrations على قاعدة بيانات الاختبار
./run-migrations.sh
```

### الخطوة 3: تشغيل الاختبارات

**Windows:**
```powershell
.\run-e2e-test.ps1
```

**Linux/Mac:**
```bash
chmod +x run-e2e-test.sh
./run-e2e-test.sh
```

### النتيجة المتوقعة

```
========================================
  End-to-End Integration Test Runner
========================================

✓ Docker is running
✓ PostgreSQL is running
✓ Redis is running
✓ Services ready

========================================
  Running End-to-End Tests
========================================

=== RUN   TestE2E_CompleteMessageFlow
    e2e_flow_test.go:XX: Cleaning up test data...
    e2e_flow_test.go:XX: ✓ Test data cleaned
    e2e_flow_test.go:XX: ✓ Test tenant created: xxx (balance: 100)
    e2e_flow_test.go:XX: ✓ Test channel created: 1
    e2e_flow_test.go:XX: Step 1: Sending message via API Gateway
    e2e_flow_test.go:XX: ✓ Message queued with ID: xxx (response time: 5ms)
    e2e_flow_test.go:XX: Step 2: Verifying message in Redis Buffer
    e2e_flow_test.go:XX: ✓ Message found in Redis Buffer (size: 1)
    e2e_flow_test.go:XX: Step 3: Processing batch
    e2e_flow_test.go:XX: ✓ Message inserted into database with status: PENDING
    e2e_flow_test.go:XX: Step 4: Verifying River Job created
    e2e_flow_test.go:XX: ✓ River Job created with ID: 1
    e2e_flow_test.go:XX: Step 5: Getting initial balance
    e2e_flow_test.go:XX: ✓ Initial balance: 100
    e2e_flow_test.go:XX: Step 6: Simulating Worker processing - deducting balance
    e2e_flow_test.go:XX: ✓ Balance deducted: 0.05 (new balance: 99.95)
    e2e_flow_test.go:XX: Step 7: Selecting channel
    e2e_flow_test.go:XX: ✓ Channel selected: 1
    e2e_flow_test.go:XX: Step 8: Updating message status to PROCESSING
    e2e_flow_test.go:XX: ✓ Message status updated to PROCESSING
    e2e_flow_test.go:XX: Step 9: Simulating WebSocket message send
    e2e_flow_test.go:XX: ✓ WebSocket message prepared for sending
    e2e_flow_test.go:XX: Step 10: Processing successful status report (SENT)
    e2e_flow_test.go:XX: ✓ Message status updated to: SENT
    e2e_flow_test.go:XX: Step 11: Verifying accounting transaction completed
    e2e_flow_test.go:XX: ✓ Accounting transaction status: COMPLETED
    e2e_flow_test.go:XX: Step 12: Verifying balance history logged
    e2e_flow_test.go:XX: ✓ Balance history entries: 1
    e2e_flow_test.go:XX: Step 13: Verifying final balance
    e2e_flow_test.go:XX: ✓ Final balance: 99.95 (expected: 99.95)
    e2e_flow_test.go:XX: ✅ End-to-End flow completed successfully!
--- PASS: TestE2E_CompleteMessageFlow (2.34s)

=== RUN   TestE2E_FailedMessageFlow
    e2e_flow_test.go:XX: Cleaning up test data...
    e2e_flow_test.go:XX: ✓ Test data cleaned
    e2e_flow_test.go:XX: ✓ Test tenant created: xxx (balance: 100)
    e2e_flow_test.go:XX: ✓ Test channel created: 2
    e2e_flow_test.go:XX: Step 1: Sending message via API Gateway
    e2e_flow_test.go:XX: ✓ Message queued with ID: xxx
    e2e_flow_test.go:XX: Step 2: Processing batch
    e2e_flow_test.go:XX: ✓ Message inserted into database
    e2e_flow_test.go:XX: Step 3: Getting initial balance
    e2e_flow_test.go:XX: ✓ Initial balance: 100
    e2e_flow_test.go:XX: Step 4: Deducting balance
    e2e_flow_test.go:XX: ✓ Balance deducted: 0.05 (new balance: 99.95)
    e2e_flow_test.go:XX: Step 5: Creating accounting transaction
    e2e_flow_test.go:XX: ✓ Accounting transaction created
    e2e_flow_test.go:XX: Step 7: Processing FAILED status report
    e2e_flow_test.go:XX: ✓ Status report processed
    e2e_flow_test.go:XX: Step 8: Verifying message status updated to FAILED
    e2e_flow_test.go:XX: ✓ Message status: FAILED
    e2e_flow_test.go:XX: Step 9: Verifying balance refunded
    e2e_flow_test.go:XX: ✓ Balance refunded: 0.05 (final balance: 100)
    e2e_flow_test.go:XX: Step 10: Verifying accounting transaction refunded
    e2e_flow_test.go:XX: ✓ Accounting transaction status: REFUNDED
    e2e_flow_test.go:XX: Step 11: Verifying balance history has both DEDUCT and REFUND
    e2e_flow_test.go:XX: ✓ Balance history: 1 DEDUCT, 1 REFUND
    e2e_flow_test.go:XX: ✅ Failed message flow with refund completed successfully!
--- PASS: TestE2E_FailedMessageFlow (2.12s)

PASS
ok      github.com/al-hudhud/backend/internal/integration      4.567s

========================================
✅ All tests passed!
========================================
```

---

## استكشاف الأخطاء السريع

### المشكلة: Docker غير متاح

```bash
# تشغيل Docker Desktop (Windows/Mac)
# أو تشغيل Docker daemon (Linux)
sudo systemctl start docker
```

### المشكلة: PostgreSQL غير متاح

```bash
docker-compose up -d postgres
sleep 5
```

### المشكلة: Redis غير متاح

```bash
docker-compose up -d redis
sleep 2
```

### المشكلة: Migrations غير مطبقة

```bash
./run-migrations.sh
```

---

## الأوامر المفيدة

```bash
# تشغيل اختبار واحد فقط
go test -v ./internal/integration/ -run TestE2E_CompleteMessageFlow

# تشغيل مع عرض تفاصيل أكثر
go test -v -count=1 ./internal/integration/...

# تشغيل بدون cache
go test -v -count=1 ./internal/integration/...

# تنظيف قاعدة البيانات يدوياً
psql -h localhost -U postgres -d sms_gateway_test -c "
DELETE FROM balance_history;
DELETE FROM accounting_transactions;
DELETE FROM river_job;
DELETE FROM messages;
DELETE FROM channels;
DELETE FROM tenants;
"

# تنظيف Redis يدوياً
redis-cli -n 1 FLUSHDB
```

---

## ما يتم اختباره؟

### ✅ TestE2E_CompleteMessageFlow
- إرسال رسالة ناجحة من البداية للنهاية
- خصم الرصيد بشكل صحيح
- تسجيل جميع العمليات
- تحديث حالة الرسالة إلى SENT

### ✅ TestE2E_FailedMessageFlow
- إرسال رسالة فاشلة
- استرداد الرصيد تلقائياً
- تحديث المعاملة المحاسبية إلى REFUNDED
- تسجيل DEDUCT و REFUND في Balance History

---

## الوقت المتوقع

- **الإعداد الأولي:** 1-2 دقيقة
- **تشغيل الاختبارات:** 5-6 ثوانٍ
- **الإجمالي:** ~2 دقيقة

---

## الخطوات التالية

بعد نجاح الاختبارات:

1. ✅ راجع الكود في `e2e_flow_test.go`
2. ✅ اقرأ التوثيق الكامل في `README.md`
3. ✅ أضف اختبارات جديدة حسب الحاجة
4. ✅ قم بدمج الاختبارات في CI/CD pipeline

---

**نصيحة:** احفظ هذا الملف كمرجع سريع! 📌
