# دليل Property-Based Tests للـ Balance Manager

## نظرة عامة

هذا الدليل يشرح Property-Based Tests المطبقة على Balance Manager للتحقق من الخصائص الحرجة للنظام.

---

## Property 1: Atomic Balance Deduction Correctness

### الوصف
يتحقق من صحة العمليات الذرية للخصم من الرصيد في بيئة متزامنة.

### الخاصية المختبرة
```
الرصيد النهائي = الرصيد الأولي - مجموع الخصومات الناجحة
```

### المتطلبات المغطاة
- 1.1: Atomic balance operations
- 1.2: No race conditions
- 1.3: Correct balance calculation
- 11.1: Balance integrity
- 11.7: Concurrent operation safety

### كيفية العمل

1. **إنشاء tenant** مع رصيد أولي عشوائي (10,000-20,000)
2. **توليد عمليات خصم عشوائية** (50 عملية متزامنة)
3. **تنفيذ العمليات بشكل متزامن** باستخدام goroutines
4. **حساب مجموع الخصومات الناجحة**
5. **التحقق من الرصيد النهائي** = الأولي - المجموع

### معايير النجاح

✅ الرصيد النهائي يطابق الحساب المتوقع (±0.01)  
✅ الرصيد لا يصبح سالباً أبداً  
✅ عدد سجلات balance_history = عدد الخصومات الناجحة  

### مثال على التشغيل

```bash
# تشغيل Property Test
cd services/backend
go test -v ./internal/balance -run TestProperty_AtomicBalanceDeductionCorrectness

# تشغيل مع عدد تكرارات أكبر
go test -v ./internal/balance -run TestProperty_AtomicBalanceDeductionCorrectness -count=20
```

### النتائج المتوقعة

```
=== RUN   TestProperty_AtomicBalanceDeductionCorrectness
=== RUN   TestProperty_AtomicBalanceDeductionCorrectness/Iteration_1
    manager_property_test.go:89: Iteration 1: Initial=15234.00, Final=12456.78, Deducted=2777.22, Success=45/50
=== RUN   TestProperty_AtomicBalanceDeductionCorrectness/Iteration_2
    manager_property_test.go:89: Iteration 2: Initial=18901.00, Final=15123.45, Deducted=3777.55, Success=48/50
...
--- PASS: TestProperty_AtomicBalanceDeductionCorrectness (2.34s)
```

---

## Property 2: No Race Conditions

### الوصف
يكتشف أي race conditions في عمليات الرصيد المتزامنة.

### الخاصية المختبرة
```
عند تنفيذ N عملية متزامنة بنفس المبلغ:
الرصيد النهائي = الرصيد الأولي - (عدد العمليات الناجحة × المبلغ)
```

### كيفية العمل

1. **إنشاء tenant** مع رصيد كبير (50,000)
2. **تنفيذ 100 عملية متزامنة** بنفس المبلغ (100)
3. **حساب عدد العمليات الناجحة**
4. **التحقق من دقة الرصيد النهائي**

### معايير النجاح

✅ لا توجد فروقات في الرصيد (دقة ±0.01)  
✅ جميع العمليات الناجحة مسجلة بشكل صحيح  

---

## Property 3: Idempotency

### الوصف
يتحقق من أن محاولات الخصم المكررة لا تؤثر على الرصيد.

### الخاصية المختبرة
```
عند محاولة خصم نفس message_id مرتين:
الرصيد بعد المحاولة الثانية = الرصيد بعد المحاولة الأولى
```

### كيفية العمل

1. **خصم مبلغ** باستخدام message_id معين
2. **محاولة الخصم مرة أخرى** بنفس message_id
3. **التحقق من عدم تغير الرصيد**

### معايير النجاح

✅ الرصيد لا يتغير عند إعادة المحاولة  
✅ الخصم يحدث مرة واحدة فقط  

---

## Property 4: Monotonicity

### الوصف
يتحقق من أن الرصيد يتناقص بشكل رتيب (بدون refunds).

### الخاصية المختبرة
```
في سلسلة من عمليات الخصم (بدون استرداد):
الرصيد الحالي ≤ الرصيد السابق
```

### كيفية العمل

1. **تنفيذ سلسلة من عمليات الخصم** (20 عملية)
2. **التحقق بعد كل عملية** من أن الرصيد لم يزد
3. **تسجيل التغيرات** في الرصيد

### معايير النجاح

✅ الرصيد يتناقص أو يبقى ثابتاً (لا يزيد)  
✅ لا توجد قفزات غير متوقعة في الرصيد  

---

## تشغيل جميع Property Tests

```bash
# تشغيل جميع Property Tests
cd services/backend
go test -v ./internal/balance -run TestProperty

# تشغيل مع تغطية الكود
go test -v -cover ./internal/balance -run TestProperty

# تشغيل مع race detector
go test -v -race ./internal/balance -run TestProperty

# تشغيل مع عدد تكرارات أكبر
go test -v ./internal/balance -run TestProperty -count=50
```

---

## تفسير النتائج

### نجاح الاختبار ✅

```
--- PASS: TestProperty_AtomicBalanceDeductionCorrectness (2.34s)
--- PASS: TestProperty_NoRaceConditionsInBalanceOperations (1.89s)
--- PASS: TestProperty_BalanceOperationIdempotency (0.45s)
--- PASS: TestProperty_BalanceMonotonicity (0.78s)
PASS
ok      internal/balance    5.467s
```

### فشل الاختبار ❌

```
--- FAIL: TestProperty_AtomicBalanceDeductionCorrectness (2.34s)
    --- FAIL: TestProperty_AtomicBalanceDeductionCorrectness/Iteration_3 (0.23s)
        manager_property_test.go:85: Property violated: Final balance should equal initial balance minus successful deductions
            Initial: 15234.00, Final: 12450.00, Expected: 12456.78, Total Deducted: 2777.22, Success Count: 45/50
```

**التفسير**: وُجد race condition أدى إلى فرق 6.78 في الرصيد.

---

## استكشاف الأخطاء

### المشكلة: Property Test يفشل بشكل متقطع

**الأسباب المحتملة:**
1. Race conditions في الكود
2. عدم استخدام transactions بشكل صحيح
3. مشاكل في database isolation level

**الحل:**
```bash
# تشغيل مع race detector
go test -v -race ./internal/balance -run TestProperty

# زيادة عدد التكرارات للكشف عن المشاكل
go test -v ./internal/balance -run TestProperty -count=100
```

### المشكلة: Property Test بطيء جداً

**الحل:**
```bash
# تقليل عدد العمليات المتزامنة في الكود
# أو تشغيل في وضع short
go test -v -short ./internal/balance
```

---

## أفضل الممارسات

### 1. عدد التكرارات

- **Development**: 10-20 تكرار
- **CI/CD**: 50-100 تكرار
- **Pre-release**: 200+ تكرار

### 2. عدد العمليات المتزامنة

- **Quick test**: 20-50 عملية
- **Standard**: 50-100 عملية
- **Stress test**: 200+ عملية

### 3. تشغيل منتظم

```bash
# في CI/CD pipeline
go test -v -race ./internal/balance -run TestProperty -count=50
```

---

## الخلاصة

Property-Based Tests توفر:

✅ **اكتشاف Race Conditions** بشكل موثوق  
✅ **التحقق من الخصائص الرياضية** للنظام  
✅ **اختبار شامل** لسيناريوهات متعددة  
✅ **ثقة عالية** في صحة العمليات المالية  

---

**تاريخ الإنشاء**: 2026-02-26  
**الإصدار**: 1.0  
**الحالة**: جاهز للاستخدام
