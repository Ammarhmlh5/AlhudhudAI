# Database Failure Recovery Test - دليل البدء السريع

## نظرة عامة

هذا الاختبار يتحقق من قدرة النظام على التعافي من فشل قاعدة البيانات والتحول التلقائي لقاعدة بيانات احتياطية.

**المتطلب المرتبط:** 24.3 - استرداد من الأعطال

## المتطلبات الأساسية

### 1. البنية التحتية
```bash
# تشغيل PostgreSQL
docker-compose up -d postgres

# تشغيل Redis (للاختبارات الأخرى)
docker-compose up -d redis
```

### 2. قاعدة البيانات
```bash
# تطبيق الهجرات
cd services/backend
./run-migrations.sh  # Linux/Mac
# أو
run-migrations.bat   # Windows
```

## تشغيل الاختبار

### الطريقة 1: تشغيل مباشر
```bash
cd services/backend
go test -v ./internal/integration -run TestDatabaseFailureRecovery
```

### الطريقة 2: استخدام السكريبت (PowerShell)
```powershell
cd services/backend
.\run-database-failure-test.ps1
```

### الطريقة 3: استخدام السكريبت (Bash)
```bash
cd services/backend
./run-database-failure-test.sh
```

## السيناريوهات المختبرة

### 1. Primary Database Failure
- **الهدف:** محاكاة فشل قاعدة البيانات الرئيسية
- **الخطوات:**
  1. إجراء عملية ناجحة على القاعدة الرئيسية
  2. محاكاة فشل القاعدة الرئيسية
  3. التحول لقاعدة البيانات الاحتياطية
  4. إجراء عملية على القاعدة الاحتياطية
  5. التحقق من عدم فقدان البيانات

### 2. Automatic Failover With Operations
- **الهدف:** التحقق من التحول التلقائي أثناء العمليات الجارية
- **الخطوات:**
  1. بدء عمليات متزامنة
  2. محاكاة فشل القاعدة أثناء العمليات
  3. التحول التلقائي للقاعدة الاحتياطية
  4. استكمال العمليات المتبقية
  5. التحقق من نجاح بعض العمليات على الأقل

### 3. Reconnection To Primary
- **الهدف:** التحقق من إعادة الاتصال بالقاعدة الرئيسية عند عودتها
- **الخطوات:**
  1. إجراء عملية على القاعدة الاحتياطية
  2. محاكاة عودة القاعدة الرئيسية
  3. التحول للقاعدة الرئيسية
  4. إجراء عملية على القاعدة الرئيسية
  5. التحقق من اتساق البيانات

### 4. Connection Pool Behavior
- **الهدف:** التحقق من سلوك Connection Pool أثناء الفشل
- **الخطوات:**
  1. التحقق من صحة Pool
  2. الحصول على اتصالات من Pool
  3. إطلاق الاتصالات
  4. التحقق من استرداد Pool

## النتائج المتوقعة

### ✅ نجاح الاختبار
```
=== RUN   TestDatabaseFailureRecovery
=== RUN   TestDatabaseFailureRecovery/PrimaryDatabaseFailure
    Step 1: Performing operation on primary database
    ✓ Balance deducted successfully: 100 -> 99
    Step 2: Simulating primary database failure
    ✓ Primary database is unavailable
    Step 3: Connecting to backup database
    ✓ Backup database is operational
    Step 4: Performing operation on backup database
    ✓ Balance deducted on backup: 99 -> 98
    Step 5: Verifying no data loss
    ✓ No data loss detected. Current balance: 98
--- PASS: TestDatabaseFailureRecovery/PrimaryDatabaseFailure (0.25s)

=== RUN   TestDatabaseFailureRecovery/AutomaticFailoverWithOperations
    Step 1: Starting concurrent operations
    Step 2: Simulating database failure during operations
    Step 3: Reconnecting to backup database
    ✓ Switched to backup database
    ✓ Operations completed: 3 successful, 2 failed
--- PASS: TestDatabaseFailureRecovery/AutomaticFailoverWithOperations (0.18s)

=== RUN   TestDatabaseFailureRecovery/ReconnectionToPrimary
    Step 1: Performing operation on backup database
    ✓ Operation on backup successful: 100 -> 99
    Step 2: Simulating primary database recovery
    ✓ Primary database is operational again
    Step 3: Switching back to primary database
    ✓ Operation on primary successful: 99 -> 98
    Step 4: Verifying data consistency
    ✓ Data is consistent. Current balance: 98
--- PASS: TestDatabaseFailureRecovery/ReconnectionToPrimary (0.22s)

=== RUN   TestDatabaseFailureRecovery/ConnectionPoolBehavior
    Step 1: Verifying connection pool is healthy
    ✓ Pool stats: Total=2, Idle=2, Acquired=0
    Step 2: Acquiring connections from pool
    ✓ Acquired 3 connections
    Step 3: Releasing connections
    ✓ All connections released
    ✓ Pool stats after release: Total=3, Idle=3, Acquired=0
--- PASS: TestDatabaseFailureRecovery/ConnectionPoolBehavior (0.15s)

--- PASS: TestDatabaseFailureRecovery (0.80s)
PASS
```

## معايير النجاح

### المتطلب 24.3 - استرداد من الأعطال
- ✅ **24.3.1:** اكتشاف فشل قاعدة البيانات الرئيسية
- ✅ **24.3.2:** التحول التلقائي لقاعدة البيانات الاحتياطية
- ✅ **24.3.3:** استمرار عمل النظام بدون فقدان بيانات
- ✅ **24.3.4:** إعادة الاتصال بالقاعدة الرئيسية عند عودتها
- ✅ **24.3.5:** سلوك صحيح لـ Connection Pool أثناء الفشل

## استكشاف الأخطاء

### خطأ: Failed to connect to database
```
Error: failed to connect to database: connection refused
```

**الحل:**
```bash
# تأكد من تشغيل PostgreSQL
docker-compose ps postgres

# إذا لم يكن يعمل
docker-compose up -d postgres

# انتظر حتى يصبح جاهزاً
docker-compose logs -f postgres
```

### خطأ: Failed to create test tenant
```
Error: failed to create test tenant: relation "tenants" does not exist
```

**الحل:**
```bash
# تطبيق الهجرات
cd services/backend
./run-migrations.sh
```

### خطأ: Context deadline exceeded
```
Error: context deadline exceeded
```

**الحل:**
- تأكد من أن قاعدة البيانات ليست محملة بشكل زائد
- زيادة timeout في الاختبار إذا لزم الأمر

## ملاحظات مهمة

### 1. بيئة الاختبار
- يستخدم الاختبار قاعدة بيانات `sms_gateway_test`
- يتم إنشاء بيانات اختبار مؤقتة
- يتم تنظيف البيانات بعد كل اختبار

### 2. محاكاة الفشل
- يتم محاكاة الفشل بإغلاق الاتصال
- في الإنتاج، يجب استخدام آليات failover حقيقية
- يمكن استخدام PostgreSQL Replication للـ High Availability

### 3. الأداء
- الاختبار يستغرق حوالي 0.8 ثانية
- يتضمن عمليات I/O حقيقية على قاعدة البيانات
- قد يستغرق وقتاً أطول على أجهزة بطيئة

## التكامل مع CI/CD

### GitHub Actions
```yaml
- name: Run Database Failure Recovery Test
  run: |
    cd services/backend
    go test -v ./internal/integration -run TestDatabaseFailureRecovery
```

### GitLab CI
```yaml
test:database-failure:
  script:
    - cd services/backend
    - go test -v ./internal/integration -run TestDatabaseFailureRecovery
```

## الخطوات التالية

بعد نجاح هذا الاختبار:
1. ✅ تنفيذ آلية Failover تلقائية في الإنتاج
2. ✅ إعداد PostgreSQL Replication
3. ✅ تكوين Health Checks للقاعدة الرئيسية
4. ✅ إعداد Monitoring للكشف عن الفشل
5. ✅ توثيق إجراءات الطوارئ

## الموارد الإضافية

- [دليل الاختبار الكامل](./DATABASE_FAILURE_TEST_GUIDE.md)
- [PostgreSQL High Availability](https://www.postgresql.org/docs/current/high-availability.html)
- [Connection Pooling Best Practices](https://www.postgresql.org/docs/current/runtime-config-connection.html)

---

**آخر تحديث:** 2024
**الحالة:** ✅ جاهز للاستخدام
