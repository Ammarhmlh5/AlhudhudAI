# دليل البدء السريع - Property Tests للـ Balance Manager

## التشغيل السريع

### 1. تشغيل جميع Property Tests

```bash
cd services/backend
go test -v ./internal/balance -run TestProperty
```

### 2. تشغيل مع Race Detector

```bash
go test -v -race ./internal/balance -run TestProperty
```

### 3. تشغيل مع تكرارات متعددة

```bash
go test -v ./internal/balance -run TestProperty -count=20
```

---

## الاختبارات المتوفرة

| الاختبار | الوصف | الوقت المتوقع |
|---------|-------|---------------|
| `TestProperty_AtomicBalanceDeductionCorrectness` | التحقق من صحة العمليات الذرية | ~2-3 ثانية |
| `TestProperty_NoRaceConditionsInBalanceOperations` | كشف Race Conditions | ~1-2 ثانية |
| `TestProperty_BalanceOperationIdempotency` | التحقق من Idempotency | ~0.5 ثانية |
| `TestProperty_BalanceMonotonicity` | التحقق من التناقص الرتيب | ~0.8 ثانية |

---

## أمثلة الاستخدام

### مثال 1: اختبار سريع

```bash
# تشغيل في وضع short (يتخطى Property Tests)
go test -v -short ./internal/balance
```

### مثال 2: اختبار شامل

```bash
# تشغيل مع race detector وتكرارات متعددة
go test -v -race ./internal/balance -run TestProperty -count=50
```

### مثال 3: اختبار محدد

```bash
# تشغيل اختبار واحد فقط
go test -v ./internal/balance -run TestProperty_AtomicBalanceDeductionCorrectness
```

---

## النتائج المتوقعة

### نجاح ✅

```
=== RUN   TestProperty_AtomicBalanceDeductionCorrectness
=== RUN   TestProperty_AtomicBalanceDeductionCorrectness/Iteration_1
    manager_property_test.go:89: Iteration 1: Initial=15234.00, Final=12456.78, Deducted=2777.22, Success=45/50
--- PASS: TestProperty_AtomicBalanceDeductionCorrectness (2.34s)
PASS
ok      internal/balance    2.345s
```

### فشل ❌

```
--- FAIL: TestProperty_AtomicBalanceDeductionCorrectness (2.34s)
    manager_property_test.go:85: Property violated: Final balance should equal initial balance minus successful deductions
FAIL
```

---

## استكشاف الأخطاء السريع

### المشكلة: الاختبار يفشل بشكل متقطع

```bash
# زيادة عدد التكرارات للتأكد
go test -v ./internal/balance -run TestProperty -count=100
```

### المشكلة: الاختبار بطيء

```bash
# تشغيل في وضع short
go test -v -short ./internal/balance
```

### المشكلة: Race condition detected

```bash
# تشغيل مع race detector للتفاصيل
go test -v -race ./internal/balance -run TestProperty
```

---

## التكامل مع CI/CD

### GitHub Actions

```yaml
- name: Run Property Tests
  run: |
    cd services/backend
    go test -v -race ./internal/balance -run TestProperty -count=50
```

### GitLab CI

```yaml
test:property:
  script:
    - cd services/backend
    - go test -v -race ./internal/balance -run TestProperty -count=50
```

---

## الخطوات التالية

1. ✅ تشغيل الاختبارات محلياً
2. ✅ التحقق من النتائج
3. ✅ إضافة إلى CI/CD pipeline
4. ✅ مراجعة [PROPERTY_TEST_GUIDE.md](./PROPERTY_TEST_GUIDE.md) للتفاصيل

---

**تاريخ الإنشاء**: 2026-02-26  
**الإصدار**: 1.0
