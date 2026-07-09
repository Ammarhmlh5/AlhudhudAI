# دليل اختبار معالجة الرسائل بالجملة (Bulk Message Processing Test)

## نظرة عامة

هذا الاختبار يتحقق من قدرة النظام على معالجة **10,000 رسالة دفعة واحدة** بكفاءة وموثوقية.

**المتطلبات المختبرة:**
- المتطلب 8.1: قبول طلبات Bulk تحتوي حتى 10,000 رسالة
- المتطلب 8.7: معالجة 10,000 رسالة خلال 2 دقيقة

## ما يختبره هذا الاختبار

### 1. إرسال الرسائل بالجملة (Bulk Sending)
- ✅ إرسال 10,000 رسالة بالتوازي
- ✅ قياس وقت الإرسال
- ✅ حساب معدل الإرسال (messages/minute)

### 2. التخزين المؤقت (Buffering)
- ✅ التحقق من وجود جميع الرسائل في Redis Buffer
- ✅ عدم فقدان أي رسالة

### 3. معالجة الدفعات (Batch Processing)
- ✅ معالجة جميع الرسائل على شكل دفعات
- ✅ إدخال الرسائل في قاعدة البيانات
- ✅ إنشاء River Jobs لكل رسالة
- ✅ قياس وقت المعالجة

### 4. العمليات المالية (Financial Operations)
- ✅ خصم الرصيد لجميع الرسائل
- ✅ التحقق من صحة الرصيد النهائي
- ✅ تسجيل جميع العمليات في balance_history

### 5. المحاسبة (Accounting)
- ✅ إنشاء معاملات محاسبية لجميع الرسائل
- ✅ تسجيل جميع المعاملات في accounting_transactions

### 6. الأداء (Performance)
- ✅ قياس الوقت الإجمالي للمعالجة
- ✅ حساب معدل المعالجة (throughput)
- ✅ التحقق من تحقيق الأداء المستهدف (5000+ رسالة/دقيقة)

## المتطلبات

### البنية التحتية
- PostgreSQL (running on localhost:5432)
- Redis (running on localhost:6379)
- Go 1.21+

### قاعدة البيانات
يجب أن تحتوي قاعدة البيانات على الجداول التالية:
- `tenants` - معلومات الشركات
- `channels` - قنوات الإرسال (الأجهزة)
- `messages` - الرسائل
- `balance_history` - سجل العمليات المالية
- `accounting_transactions` - المعاملات المحاسبية
- `river_job` - وظائف River Queue

## كيفية التشغيل

### الطريقة 1: استخدام السكريبت (موصى به)

#### Linux/Mac:
```bash
cd /path/to/project
chmod +x services/backend/run-bulk-test.sh
./services/backend/run-bulk-test.sh
```

#### Windows (PowerShell):
```powershell
cd C:\path\to\project
.\services\backend\run-bulk-test.ps1
```

### الطريقة 2: تشغيل مباشر

```bash
cd services/backend
go test -v -timeout 10m -run TestBulkMessageProcessing ./internal/integration/
```

## مراحل الاختبار

الاختبار يمر بـ **11 مرحلة** متسلسلة:

### المرحلة 1: إرسال 10,000 رسالة
- إرسال الرسائل بالتوازي باستخدام 100 goroutine
- كل goroutine يرسل 100 رسالة
- قياس وقت الإرسال الكامل

### المرحلة 2: التحقق من Redis Buffer
- التأكد من وجود جميع الرسائل في Buffer
- حجم Buffer يجب أن يكون 10,000

### المرحلة 3: معالجة الدفعات
- معالجة الرسائل على شكل دفعات (100 رسالة/دفعة)
- إدخال الدفعات في قاعدة البيانات
- قياس وقت المعالجة

### المرحلة 4: التحقق من قاعدة البيانات
- التأكد من وجود جميع الرسائل في جدول `messages`
- العدد يجب أن يكون 10,000

### المرحلة 5: التحقق من River Jobs
- التأكد من إنشاء River Job لكل رسالة
- العدد يجب أن يكون >= 10,000

### المرحلة 6: خصم الرصيد
- محاكاة Worker processing
- خصم الرصيد لكل رسالة (0.05 لكل رسالة)
- قياس وقت الخصم

### المرحلة 7: التحقق من الرصيد النهائي
- حساب الرصيد المتوقع
- مقارنة مع الرصيد الفعلي
- السماح بفرق صغير جداً (< 0.01)

### المرحلة 8: التحقق من balance_history
- التأكد من تسجيل جميع عمليات الخصم
- العدد يجب أن يساوي عدد الخصومات الناجحة

### المرحلة 9: إنشاء المعاملات المحاسبية
- إنشاء accounting_transaction لكل رسالة
- حالة المعاملة: COMPLETED

### المرحلة 10: التحقق من accounting_transactions
- التأكد من وجود معاملة لكل خصم ناجح
- العدد يجب أن يكون >= عدد الخصومات الناجحة

### المرحلة 11: حساب النتائج النهائية
- الوقت الإجمالي
- معدل المعالجة (throughput)
- ملخص مالي

## النتائج المتوقعة

### الأداء
```
Total messages: 10,000
Messages sent successfully: 10,000
Messages in database: 10,000
Balance deductions successful: ~9,500-10,000 (95%+)
Balance history entries: ~9,500-10,000
Accounting transactions: ~9,500-10,000

⏱️  Total processing time: < 2 minutes
   - Send time: ~5-10 seconds
   - Batch processing time: ~30-60 seconds
   - Balance deduction time: ~20-40 seconds
   - Transaction creation time: ~10-20 seconds

📈 Throughput:
   - Send: ~60,000-120,000 messages/minute
   - Processing: ~10,000-20,000 messages/minute
   - Overall: >= 5,000 messages/minute

💰 Financial summary:
   - Initial balance: 1000.00
   - Total deducted: ~475-500 (depending on success rate)
   - Final balance: ~500-525
```

### معايير النجاح
- ✅ جميع الرسائل تُرسل بنجاح (10,000)
- ✅ جميع الرسائل في قاعدة البيانات (10,000)
- ✅ معدل نجاح الخصم >= 95%
- ✅ الرصيد النهائي صحيح (فرق < 0.01)
- ✅ جميع العمليات مسجلة في balance_history
- ✅ جميع المعاملات مسجلة في accounting_transactions
- ✅ معدل المعالجة >= 5,000 رسالة/دقيقة
- ✅ الوقت الإجمالي <= 2 دقيقة

## استكشاف الأخطاء

### خطأ: "Failed to connect to database"
**الحل:**
```bash
# تأكد من تشغيل PostgreSQL
docker ps | grep postgres

# إذا لم يكن يعمل، شغل البنية التحتية
./start-infrastructure.sh  # Linux/Mac
.\start-infrastructure.bat  # Windows
```

### خطأ: "Failed to connect to Redis"
**الحل:**
```bash
# تأكد من تشغيل Redis
docker ps | grep redis

# إذا لم يكن يعمل، شغل البنية التحتية
./start-infrastructure.sh  # Linux/Mac
.\start-infrastructure.bat  # Windows
```

### خطأ: "Balance deduction failed"
**السبب المحتمل:** رصيد غير كافٍ

**الحل:**
- الاختبار يبدأ برصيد 1000 وحدة
- التكلفة الإجمالية: 10,000 × 0.05 = 500 وحدة
- إذا فشل الخصم، تحقق من الرصيد الأولي

### خطأ: "Throughput below 5000 messages/minute"
**الأسباب المحتملة:**
1. موارد النظام محدودة (CPU/Memory)
2. قاعدة البيانات بطيئة
3. Redis بطيء

**الحلول:**
- زيادة موارد Docker
- تحسين إعدادات PostgreSQL
- تحسين إعدادات Redis

### خطأ: "Test timeout"
**الحل:**
```bash
# زيادة وقت الانتظار
go test -v -timeout 15m -run TestBulkMessageProcessing ./internal/integration/
```

## الإحصائيات المفصلة

الاختبار يطبع إحصائيات مفصلة في النهاية:

```
================================================================================
📊 BULK PROCESSING TEST RESULTS
================================================================================
Total messages: 10000
Messages sent successfully: 10000
Messages in database: 10000
Balance deductions successful: 9876
Balance history entries: 9876
Accounting transactions: 9876

⏱️  Total processing time: 1m45s
   - Send time: 8.5s
   - Batch processing time: 45s
   - Balance deduction time: 32s
   - Transaction creation time: 15s

📈 Throughput:
   - Send: 70,588 messages/minute
   - Processing: 13,333 messages/minute
   - Overall: 5,714 messages/minute

💰 Financial summary:
   - Initial balance: 1000.0000
   - Total deducted: 493.8000
   - Final balance: 506.2000
================================================================================
```

## التكامل مع CI/CD

يمكن إضافة هذا الاختبار إلى pipeline CI/CD:

```yaml
# .github/workflows/bulk-test.yml
name: Bulk Message Processing Test

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  bulk-test:
    runs-on: ubuntu-latest
    
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: sms_gateway_test
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
      
      - name: Run migrations
        run: |
          cd services/backend
          go run migrations/*.sql
      
      - name: Run bulk test
        run: |
          cd services/backend
          go test -v -timeout 10m -run TestBulkMessageProcessing ./internal/integration/
```

## الملاحظات

### الأداء
- الاختبار يستخدم 100 goroutine للإرسال المتوازي
- معالجة الدفعات تتم بحجم 100 رسالة/دفعة
- خصم الرصيد يتم بدفعات من 100 عملية

### الموثوقية
- الاختبار يسمح بمعدل فشل 5% في خصم الرصيد (بسبب race conditions محتملة)
- يتم التحقق من صحة الرصيد النهائي مع السماح بفرق صغير جداً (< 0.01)

### التنظيف
- الاختبار ينظف جميع البيانات التجريبية قبل البدء
- يستخدم Redis DB 1 (منفصل عن الإنتاج)
- يستخدم قاعدة بيانات اختبار منفصلة

## المراجع

- [E2E Flow Test](./e2e_flow_test.go) - اختبار التدفق الكامل لرسالة واحدة
- [Requirements Document](../../../.kiro/specs/message-lifecycle-rebuild/requirements.md) - وثيقة المتطلبات
- [Design Document](../../../.kiro/specs/message-lifecycle-rebuild/design.md) - وثيقة التصميم

## الدعم

إذا واجهت أي مشاكل:
1. تحقق من تشغيل البنية التحتية (PostgreSQL, Redis)
2. تحقق من وجود جميع الجداول المطلوبة
3. راجع logs الاختبار للحصول على تفاصيل الأخطاء
4. تأكد من توفر موارد كافية (CPU, Memory)

---

**تاريخ الإنشاء:** 2026-02-25  
**الإصدار:** 1.0  
**الحالة:** جاهز للاستخدام
