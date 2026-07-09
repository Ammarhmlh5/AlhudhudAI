# دليل اختبار العمليات المتزامنة (Concurrent Operations Test Guide)

## نظرة عامة

هذا الدليل يشرح كيفية تشغيل واستخدام اختبارات العمليات المتزامنة للتحقق من عدم وجود Race Conditions في نظام إدارة الرصيد.

## المتطلبات الأساسية

### 1. PostgreSQL
يجب أن يكون PostgreSQL مثبتاً ويعمل:
```bash
# التحقق من تشغيل PostgreSQL
psql --version

# الاتصال بقاعدة البيانات
psql -h localhost -U postgres
```

### 2. قاعدة بيانات الاختبار
يجب إنشاء قاعدة بيانات للاختبار:
```sql
CREATE DATABASE alhudhud_test;
```

### 3. تطبيق Migrations
يجب تطبيق جميع migrations على قاعدة بيانات الاختبار:
```bash
# من مجلد services/backend
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/alhudhud_test?sslmode=disable"

# تطبيق migrations
for file in migrations/*.sql; do
    psql -h localhost -U postgres -d alhudhud_test < "$file"
done
```

## طرق التشغيل

### الطريقة 1: استخدام السكريبت الآلي (موصى به)

#### على Windows (PowerShell):
```powershell
cd services/backend
.\run-concurrent-test.ps1
```

#### على Linux/Mac (Bash):
```bash
cd services/backend
chmod +x run-concurrent-test.sh
./run-concurrent-test.sh
```

السكريبت سيقوم بـ:
- ✅ التحقق من وجود قاعدة البيانات
- ✅ إنشاء قاعدة البيانات إذا لم تكن موجودة
- ✅ تطبيق جميع migrations
- ✅ تشغيل جميع اختبارات Concurrent Operations
- ✅ عرض النتائج بشكل منظم

### الطريقة 2: تشغيل يدوي

#### تشغيل جميع اختبارات Concurrent:
```bash
cd services/backend
go test -v ./internal/balance -run TestConcurrent -timeout 5m
```

#### تشغيل اختبار محدد:

**اختبار 100 عملية متزامنة:**
```bash
go test -v ./internal/balance -run TestConcurrentBalanceDeductions -timeout 5m
```

**اختبار العمليات المختلطة:**
```bash
go test -v ./internal/balance -run TestConcurrentMixedOperations -timeout 5m
```

**اختبار التنافس الشديد:**
```bash
go test -v ./internal/balance -run TestHighContentionScenario -timeout 5m
```

### الطريقة 3: تشغيل مع تفاصيل إضافية

```bash
# مع verbose output
go test -v ./internal/balance -run TestConcurrent -timeout 5m

# مع race detector
go test -race -v ./internal/balance -run TestConcurrent -timeout 5m

# مع coverage
go test -cover -v ./internal/balance -run TestConcurrent -timeout 5m
```

## فهم النتائج

### مثال على نتيجة ناجحة:

```
=== RUN   TestConcurrentBalanceDeductions
=== RUN   TestConcurrentBalanceDeductions/Verify_Final_Balance_Correctness
=== RUN   TestConcurrentBalanceDeductions/Verify_No_Race_Conditions
=== RUN   TestConcurrentBalanceDeductions/Verify_All_Operations_Completed
=== RUN   TestConcurrentBalanceDeductions/Verify_Insufficient_Balance_Errors
=== RUN   TestConcurrentBalanceDeductions/Verify_Balance_History_Completeness
=== RUN   TestConcurrentBalanceDeductions/Verify_Balance_History_Invariant
=== RUN   TestConcurrentBalanceDeductions/Performance_Metrics
    concurrent_test.go:XXX: Concurrent Operations Performance:
    concurrent_test.go:XXX:   Total operations: 100
    concurrent_test.go:XXX:   Successful: 95
    concurrent_test.go:XXX:   Failed: 5
    concurrent_test.go:XXX:   Total duration: 2.5s
    concurrent_test.go:XXX:   Average per operation: 25 ms
    concurrent_test.go:XXX:   Initial balance: 1000
    concurrent_test.go:XXX:   Final balance: 525
    concurrent_test.go:XXX:   Total deducted: 475
--- PASS: TestConcurrentBalanceDeductions (2.50s)
```

### تفسير النتائج:

#### ✅ PASS - الاختبار نجح
- الرصيد النهائي صحيح
- لا توجد race conditions
- جميع العمليات معالجة بشكل صحيح

#### ❌ FAIL - الاختبار فشل
إذا فشل الاختبار، تحقق من:
1. **Final Balance Mismatch**: الرصيد النهائي لا يطابق المتوقع
   - قد يشير إلى race condition
   - تحقق من atomic operations في manager.go

2. **Balance History Mismatch**: عدد السجلات لا يطابق العمليات
   - قد يشير إلى فشل في logging
   - تحقق من history_logger.go

3. **Negative Balance**: الرصيد أصبح سالباً
   - خطأ حرج في WHERE condition
   - تحقق من DeductBalance في manager.go

## معايير النجاح

### 1. الاتساق المالي ✅
```
Final Balance = Initial Balance - (Successful Operations × Amount)
```

### 2. اكتمال السجلات ✅
```
COUNT(balance_history) = Number of Successful Operations
```

### 3. Invariant المالي ✅
```
SUM(balance_history.amount) = Initial Balance - Final Balance
```

### 4. الأداء ✅
```
100 operations complete in < 30 seconds
Average per operation < 300ms
```

## استكشاف الأخطاء

### المشكلة: قاعدة البيانات غير متاحة
```
Error: connection refused
```

**الحل:**
```bash
# تأكد من تشغيل PostgreSQL
sudo systemctl start postgresql  # Linux
brew services start postgresql   # Mac
# أو تشغيل من Services على Windows
```

### المشكلة: قاعدة البيانات غير موجودة
```
Error: database "alhudhud_test" does not exist
```

**الحل:**
```bash
psql -h localhost -U postgres -c "CREATE DATABASE alhudhud_test"
```

### المشكلة: الجداول غير موجودة
```
Error: relation "tenants" does not exist
```

**الحل:**
```bash
# تطبيق migrations
cd services/backend
for file in migrations/*.sql; do
    psql -h localhost -U postgres -d alhudhud_test < "$file"
done
```

### المشكلة: الاختبار بطيء جداً
```
Test timeout after 5m
```

**الحل:**
- تحقق من أداء قاعدة البيانات
- تأكد من وجود indexes مناسبة
- قلل عدد العمليات للاختبار السريع

### المشكلة: Race conditions detected
```
WARNING: DATA RACE
```

**الحل:**
- هذا خطأ حرج يجب إصلاحه فوراً
- راجع atomic operations في manager.go
- تأكد من استخدام SERIALIZABLE isolation level

## تخصيص الاختبارات

### تغيير عدد العمليات:
```go
// في concurrent_test.go
numOperations := 100  // غير هذا الرقم
```

### تغيير مبلغ الخصم:
```go
deductionAmount := decimal.NewFromFloat(5.0)  // غير المبلغ
```

### تغيير الرصيد الأولي:
```go
initialBalance := decimal.NewFromFloat(1000.0)  // غير الرصيد
```

### تغيير اتصال قاعدة البيانات:
```go
// في concurrent_test.go
func getTestDBConnection() string {
    return "postgres://user:pass@host:port/dbname?sslmode=disable"
}
```

## التكامل مع CI/CD

### GitHub Actions:
```yaml
name: Concurrent Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: alhudhud_test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Run migrations
        run: |
          for file in services/backend/migrations/*.sql; do
            psql -h localhost -U postgres -d alhudhud_test < "$file"
          done
        env:
          PGPASSWORD: postgres
      
      - name: Run concurrent tests
        run: |
          cd services/backend
          go test -v ./internal/balance -run TestConcurrent -timeout 5m
```

### GitLab CI:
```yaml
concurrent-tests:
  stage: test
  services:
    - postgres:15
  variables:
    POSTGRES_DB: alhudhud_test
    POSTGRES_USER: postgres
    POSTGRES_PASSWORD: postgres
  script:
    - cd services/backend
    - for file in migrations/*.sql; do psql -h postgres -U postgres -d alhudhud_test < "$file"; done
    - go test -v ./internal/balance -run TestConcurrent -timeout 5m
```

## الأسئلة الشائعة

### س: كم من الوقت يستغرق تشغيل الاختبارات؟
**ج:** عادة 3-5 دقائق لجميع الاختبارات الثلاثة.

### س: هل يمكن تشغيل الاختبارات بالتوازي؟
**ج:** نعم، لكن تأكد من استخدام tenants مختلفة لتجنب التداخل.

### س: ماذا لو فشل اختبار واحد فقط؟
**ج:** راجع السجلات لفهم سبب الفشل. قد يكون مشكلة في الأداء أو race condition.

### س: هل الاختبارات تؤثر على قاعدة بيانات الإنتاج؟
**ج:** لا، الاختبارات تستخدم قاعدة بيانات منفصلة (alhudhud_test).

### س: كيف أتخطى الاختبارات في الوضع السريع؟
**ج:** استخدم `go test -short` وسيتم تخطي اختبارات Concurrent تلقائياً.

## الموارد الإضافية

- [وثيقة التصميم الكاملة](.kiro/specs/message-lifecycle-rebuild/design.md)
- [ملخص تنفيذ المهمة](.kiro/specs/message-lifecycle-rebuild/TASK_14.6_EXECUTION_SUMMARY.md)
- [Balance Manager README](./README.md)
- [كود الاختبار](./concurrent_test.go)

## الدعم

إذا واجهت أي مشاكل:
1. راجع قسم استكشاف الأخطاء أعلاه
2. تحقق من السجلات (logs)
3. تأكد من تطبيق جميع migrations
4. تحقق من اتصال قاعدة البيانات

---

**آخر تحديث:** 2024
**الحالة:** ✅ مكتمل ومختبر
