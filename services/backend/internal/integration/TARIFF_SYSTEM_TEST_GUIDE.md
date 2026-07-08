# دليل اختبارات نظام التعرفات الشاملة

## نظرة عامة

هذا الدليل يشرح اختبارات Integration الشاملة لنظام التعرفات والتسعير. الاختبارات تغطي جميع التدفقات الرئيسية من البداية للنهاية.

## المتطلبات

### البرمجيات المطلوبة
- Go 1.21+
- PostgreSQL 15+ (يجب أن يكون قيد التشغيل)
- Redis 7+ (يجب أن يكون قيد التشغيل)
- Docker (لتشغيل PostgreSQL و Redis)

### إعداد البيئة

```bash
# تشغيل PostgreSQL و Redis عبر Docker
docker-compose up -d postgres redis

# التحقق من التشغيل
docker ps
```

## تشغيل الاختبارات

### الطريقة 1: استخدام السكريبت (موصى به)

```powershell
# من مجلد services/backend
.\run-tariff-integration-tests.ps1
```

السكريبت يقوم بـ:
1. التحقق من تشغيل Docker
2. إنشاء قاعدة بيانات الاختبار (إذا لم تكن موجودة)
3. تشغيل Migrations
4. تشغيل الاختبارات
5. عرض النتائج

### الطريقة 2: تشغيل يدوي

```bash
# إنشاء قاعدة بيانات الاختبار
docker exec postgres psql -U postgres -c "CREATE DATABASE sms_gateway_test;"

# تشغيل Migrations
DATABASE_URL="postgres://postgres:postgres@localhost:5432/sms_gateway_test?sslmode=disable" \
go run cmd/migrate/main.go

# تشغيل الاختبارات
go test -v -timeout 10m ./internal/integration -run TestTariffSystemSuite
```

## الاختبارات المتضمنة

### 1. TestCompleteSubscriptionFlow
**يختبر:** التدفق الكامل للاشتراك من البداية للنهاية

**الخطوات:**
1. إنشاء tenant برصيد 100 دولار
2. إنشاء باقة Standard (20 دولار، 3000 رسالة)
3. الاشتراك في الباقة
4. التحقق من خصم قيمة الباقة (100 - 20 = 80)
5. إرسال 100 رسالة
6. التحقق من خصم من عداد الرسائل (مستخدم: 100، متبقي: 2900)
7. محاكاة نهاية الشهر
8. التحقق من التجديد التلقائي
9. التحقق من إعادة تعيين العدادات
10. التحقق من خصم قيمة التجديد (80 - 20 = 60)

**المتطلبات المغطاة:** 4.1, 4.2, 4.3, 4.4, 4.5, 4.6, 14.2, 14.3

### 2. TestOverageFlow
**يختبر:** تدفق الاستخدام الإضافي (Overage)

**الخطوات:**
1. إنشاء tenant والاشتراك في باقة Basic (10 دولار، 1000 رسالة)
2. استخدام جميع الرسائل المضمنة (1000)
3. إرسال 100 رسالة إضافية
4. التحقق من حساب تكلفة Overage (100 × 0.01 = 1 دولار)
5. التحقق من خصم التكلفة من الرصيد (90 - 1 = 89)

**المتطلبات المغطاة:** 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7

### 3. TestRolloverFlow
**يختبر:** تدفق ترحيل الرسائل (Rollover)

**الخطوات:**
1. إنشاء باقة Professional مع Rollover مفعّل (40 دولار، 8000 رسالة، حد أقصى: 2000)
2. استخدام 6000 رسالة (متبقي: 2000)
3. محاكاة نهاية الشهر
4. التحقق من إنشاء سجل Rollover بـ 2000 رسالة
5. التجديد التلقائي
6. التحقق من إضافة الرسائل المرحّلة (8000 + 2000 = 10000)
7. استخدام 500 رسالة والتحقق من استخدام Rollover أولاً (FIFO)

**المتطلبات المغطاة:** 6.2, 6.3, 6.4, 6.5, 6.6, 6.7, 6.8, 6.9, 6.10, 6.11, 6.12

### 4. TestRolloverWithLimit
**يختبر:** احترام الحد الأقصى للترحيل

**الخطوات:**
1. إنشاء باقة مع حد أقصى للترحيل = 1000
2. استخدام 1000 رسالة فقط (متبقي: 4000)
3. محاكاة نهاية الشهر
4. التحقق من أن الترحيل محدود بـ 1000 (وليس 4000)

**المتطلبات المغطاة:** 6.9, 6.10

### 5. TestAutoRenewalFailure
**يختبر:** فشل التجديد التلقائي مع رصيد غير كافي

**الخطوات:**
1. إنشاء tenant برصيد 35 دولار
2. الاشتراك في باقة 20 دولار (متبقي: 15)
3. محاكاة نهاية الشهر
4. محاولة التجديد (يجب أن تفشل)
5. التحقق من دخول الاشتراك في فترة السماح
6. التحقق من عدم تغيير الرصيد

**المتطلبات المغطاة:** 14.4, 14.5, 14.6, 14.11

### 6. TestConditionalRollover
**يختبر:** الترحيل المشروط بناءً على نجاح الدفع

**الخطوات:**
1. إنشاء باقة مع Rollover مفعّل
2. استخدام جزء من الرسائل
3. محاكاة نهاية الشهر
4. إنشاء Rollover بحالة 'pending'
5. التجديد الناجح
6. التحقق من تفعيل Rollover (حالة: 'active')

**المتطلبات المغطاة:** 6.5, 6.6, 6.15, 6.16, 6.17

## فهم النتائج

### نجاح الاختبار
```
=== RUN   TestTariffSystemSuite
=== RUN   TestTariffSystemSuite/TestCompleteSubscriptionFlow
--- PASS: TestTariffSystemSuite/TestCompleteSubscriptionFlow (2.34s)
=== RUN   TestTariffSystemSuite/TestOverageFlow
--- PASS: TestTariffSystemSuite/TestOverageFlow (1.87s)
...
--- PASS: TestTariffSystemSuite (15.23s)
PASS
```

### فشل الاختبار
```
=== RUN   TestTariffSystemSuite/TestCompleteSubscriptionFlow
    tariff_system_test.go:123: Balance should be 80 after subscription
        Expected: 80.0
        Actual: 100.0
--- FAIL: TestTariffSystemSuite/TestCompleteSubscriptionFlow (1.23s)
```

## استكشاف الأخطاء

### خطأ: "Failed to connect to database"
**الحل:**
```bash
# تأكد من تشغيل PostgreSQL
docker-compose up -d postgres

# تحقق من الاتصال
docker exec postgres psql -U postgres -c "SELECT 1"
```

### خطأ: "Failed to connect to Redis"
**الحل:**
```bash
# تأكد من تشغيل Redis
docker-compose up -d redis

# تحقق من الاتصال
docker exec redis redis-cli ping
```

### خطأ: "Table does not exist"
**الحل:**
```bash
# شغّل Migrations
DATABASE_URL="postgres://postgres:postgres@localhost:5432/sms_gateway_test?sslmode=disable" \
go run cmd/migrate/main.go
```

### الاختبارات بطيئة جداً
**الحل:**
- تأكد من أن قاعدة البيانات و Redis يعملان محلياً (وليس عبر شبكة بطيئة)
- استخدم SSD بدلاً من HDD
- أغلق التطبيقات الأخرى التي تستهلك الموارد

## أفضل الممارسات

### 1. تشغيل الاختبارات بانتظام
```bash
# قبل كل commit
.\run-tariff-integration-tests.ps1

# في CI/CD pipeline
go test -v -timeout 10m ./internal/integration -run TestTariffSystemSuite
```

### 2. عزل بيئة الاختبار
- استخدم قاعدة بيانات منفصلة (`sms_gateway_test`)
- استخدم Redis DB منفصل (DB 2)
- لا تشغّل الاختبارات على بيانات الإنتاج أبداً

### 3. تنظيف البيانات
- الاختبارات تنظف البيانات تلقائياً قبل كل اختبار
- إذا فشل اختبار، قد تحتاج لتنظيف يدوي:
```sql
-- الاتصال بقاعدة الاختبار
psql -U postgres -d sms_gateway_test

-- حذف بيانات الاختبار
DELETE FROM rollover_messages WHERE tenant_id IN (SELECT id FROM tenants WHERE email LIKE 'test_%');
DELETE FROM subscriptions WHERE tenant_id IN (SELECT id FROM tenants WHERE email LIKE 'test_%');
DELETE FROM packages WHERE tenant_id IN (SELECT id FROM tenants WHERE email LIKE 'test_%');
DELETE FROM balance_history WHERE tenant_id IN (SELECT id FROM tenants WHERE email LIKE 'test_%');
DELETE FROM accounting_transactions WHERE tenant_id IN (SELECT id FROM tenants WHERE email LIKE 'test_%');
DELETE FROM tenants WHERE email LIKE 'test_%';
```

### 4. مراقبة الأداء
```bash
# تشغيل مع profiling
go test -v -timeout 10m -cpuprofile=cpu.prof -memprofile=mem.prof \
    ./internal/integration -run TestTariffSystemSuite

# تحليل النتائج
go tool pprof cpu.prof
go tool pprof mem.prof
```

## إضافة اختبارات جديدة

### مثال: اختبار تدفق جديد

```go
// TestNewFlow tests a new feature flow
// **Validates: Requirements X.Y, X.Z**
func (suite *TariffSystemTestSuite) TestNewFlow() {
    t := suite.T()

    // 1. Setup
    tenantID := suite.createTestTenant("test_new_flow@example.com", 100.0)
    
    // 2. Execute
    // ... your test logic here
    
    // 3. Verify
    assert.Equal(t, expected, actual, "Description of what should happen")
}
```

### نصائح للاختبارات الجديدة
1. استخدم أسماء واضحة ووصفية
2. وثّق المتطلبات المغطاة
3. اشرح الخطوات في التعليقات
4. استخدم helper methods للكود المتكرر
5. نظف البيانات بعد الاختبار

## الخلاصة

هذه الاختبارات تضمن أن نظام التعرفات يعمل بشكل صحيح من البداية للنهاية. شغّل الاختبارات بانتظام للتأكد من عدم كسر أي وظيفة موجودة.

**Requirements Covered:** 4.1-4.6, 5.1-5.7, 6.2-6.17, 14.2-14.6, 14.11

## المراجع

- [دليل الاختبارات الشاملة](../../.kiro/specs/tariff-pricing-system/guides/TASK_34_COMPREHENSIVE_TESTING_GUIDE.md)
- [مستند التصميم](../../.kiro/specs/tariff-pricing-system/design.md)
- [مستند المتطلبات](../../.kiro/specs/tariff-pricing-system/requirements.md)
