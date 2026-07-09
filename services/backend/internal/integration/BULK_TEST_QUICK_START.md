# دليل البدء السريع - اختبار معالجة الرسائل بالجملة

## ما هو هذا الاختبار؟

اختبار يرسل ويعالج **10,000 رسالة دفعة واحدة** للتحقق من:
- ✅ قدرة النظام على معالجة كميات كبيرة
- ✅ عدم فقدان أي رسالة
- ✅ صحة العمليات المالية
- ✅ تحقيق الأداء المستهدف (5000+ رسالة/دقيقة)

## التشغيل السريع

### 1. تأكد من تشغيل البنية التحتية

```bash
# Linux/Mac
./start-infrastructure.sh

# Windows
.\start-infrastructure.bat
```

### 2. شغل الاختبار

```bash
# Linux/Mac
./services/backend/run-bulk-test.sh

# Windows (PowerShell)
.\services\backend\run-bulk-test.ps1
```

### 3. انتظر النتائج (~2 دقيقة)

الاختبار سيطبع تقدم العمل:

```
🚀 Starting bulk message processing test: 10000 messages
📊 Expected total cost: 500.00
💰 Initial balance: 1000.0000

📤 Phase 1: Sending 10,000 messages...
✅ Sent 10000 messages in 8.5s

🔍 Phase 2: Verifying messages in Redis Buffer...
✅ Buffer size: 10000 messages

⚙️  Phase 3: Processing all batches...
   Processed ~1000 messages (buffer remaining: 9000)
   Processed ~2000 messages (buffer remaining: 8000)
   ...
✅ Batch processing completed in 45s

... (المزيد من المراحل)

================================================================================
📊 BULK PROCESSING TEST RESULTS
================================================================================
Total messages: 10000
Messages sent successfully: 10000
Overall throughput: 5,714 messages/minute
✅ Test PASSED
================================================================================
```

## النتائج المتوقعة

### ✅ نجاح الاختبار
```
Total messages: 10,000
Success rate: 95%+
Throughput: >= 5,000 messages/minute
Total time: <= 2 minutes
```

### ❌ فشل الاختبار

إذا فشل الاختبار، تحقق من:

1. **البنية التحتية:**
   ```bash
   docker ps | grep postgres
   docker ps | grep redis
   ```

2. **قاعدة البيانات:**
   ```bash
   # تأكد من تشغيل migrations
   ./run-migrations.sh  # Linux/Mac
   .\run-migrations.bat  # Windows
   ```

3. **الموارد:**
   - تأكد من توفر ذاكرة كافية (4GB+)
   - تأكد من توفر CPU كافٍ

## الأوامر المفيدة

### تشغيل الاختبار مع تفاصيل أكثر
```bash
cd services/backend
go test -v -timeout 10m -run TestBulkMessageProcessing ./internal/integration/
```

### تشغيل الاختبار مع وقت انتظار أطول
```bash
cd services/backend
go test -v -timeout 15m -run TestBulkMessageProcessing ./internal/integration/
```

### مراقبة قاعدة البيانات أثناء الاختبار
```bash
# في terminal منفصل
docker exec -it postgres psql -U postgres -d sms_gateway_test

# ثم في psql:
SELECT COUNT(*) FROM messages;
SELECT COUNT(*) FROM balance_history;
SELECT COUNT(*) FROM accounting_transactions;
```

## المزيد من المعلومات

للحصول على دليل شامل، راجع:
- [BULK_TEST_GUIDE.md](./BULK_TEST_GUIDE.md) - دليل مفصل
- [e2e_flow_test.go](./e2e_flow_test.go) - اختبار التدفق الكامل
- [README.md](./README.md) - دليل اختبارات التكامل

---

**وقت التشغيل المتوقع:** ~2 دقيقة  
**المتطلبات:** PostgreSQL + Redis  
**الحالة:** جاهز للاستخدام ✅
