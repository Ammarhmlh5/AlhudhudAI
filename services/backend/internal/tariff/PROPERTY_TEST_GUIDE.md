# دليل Property-Based Tests لمحرك التعرفات

## نظرة عامة

هذا الدليل يشرح Property-Based Tests المطبقة على Tariff Engine للتحقق من الخصائص الحرجة للنظام.

---

## Property 3: استخدام التعرفة النشطة الصحيحة

### الوصف
يتحقق من أن النظام يستخدم فقط التعرفات النشطة التي تقع ضمن نطاق التواريخ الصحيح.

### الخاصية المختبرة
```
لأي رسالة مع شبكة محددة، عند حساب التكلفة:
يجب أن يستخدم النظام التعرفة النشطة 
(is_active = true AND effective_from <= CURRENT_DATE AND (effective_to IS NULL OR effective_to >= CURRENT_DATE))
المطابقة للشبكة
```

### المتطلبات المغطاة
- **Requirement 2.5**: Active tariff selection with date validation

### الاختبارات المطبقة

#### 1. TestProperty_UseActiveTariffOnly
**الهدف**: التحقق من استخدام التعرفة النشطة فقط

**السيناريو**:
1. إنشاء 4 تعرفات لنفس الخدمة والشبكة:
   - تعرفة نشطة (9 ريال) - يجب استخدامها ✓
   - تعرفة غير نشطة (15 ريال) - يجب تجاهلها
   - تعرفة مستقبلية (12 ريال) - يجب تجاهلها
   - تعرفة منتهية (8 ريال) - يجب تجاهلها

2. حساب التكلفة

3. التحقق من استخدام التعرفة النشطة فقط

**معايير النجاح**:
- ✅ يتم استخدام التعرفة النشطة (9 ريال)
- ✅ يتم تجاهل جميع التعرفات الأخرى
- ✅ التكلفة المحسوبة تطابق التعرفة النشطة

#### 2. TestProperty_TariffEffectiveDateBoundaries
**الهدف**: التحقق من حدود التواريخ الفعالة

**السيناريو**:
- إنشاء تعرفة تصبح نشطة اليوم (effective_from = today)
- حساب التكلفة
- التحقق من استخدام التعرفة

**معايير النجاح**:
- ✅ التعرفة التي تصبح نشطة اليوم يتم استخدامها
- ✅ لا يتم استخدام تعرفات مستقبلية

#### 3. TestProperty_MostSpecificTariffSelection
**الهدف**: التحقق من اختيار التعرفة الأكثر تحديداً

**السيناريو**:
1. إنشاء تعرفتين:
   - تعرفة عامة (دولة فقط، بدون شبكة) - 15 ريال
   - تعرفة محددة (دولة + شبكة) - 9 ريال

2. حساب التكلفة مع تحديد الشبكة

3. التحقق من استخدام التعرفة المحددة

**معايير النجاح**:
- ✅ يتم اختيار التعرفة الأكثر تحديداً (مع الشبكة)
- ✅ يتم تجاهل التعرفة العامة

#### 4. TestProperty_InactiveTariffsNeverUsed
**الهدف**: التحقق من عدم استخدام التعرفات غير النشطة أبداً

**السيناريو**:
- إنشاء 5 تعرفات غير نشطة فقط
- محاولة حساب التكلفة
- التحقق من فشل العملية

**معايير النجاح**:
- ✅ العملية تفشل مع رسالة خطأ واضحة
- ✅ لا يتم استخدام أي تعرفة غير نشطة

#### 5. TestProperty_TariffCacheConsistency
**الهدف**: التحقق من تطابق التعرفات المخزنة مؤقتاً مع قاعدة البيانات

**السيناريو**:
1. حساب التكلفة (يتم التخزين في Cache)
2. حساب التكلفة مرة أخرى (يتم القراءة من Cache)
3. التحقق من تطابق النتائج

**معايير النجاح**:
- ✅ النتيجة من Cache تطابق النتيجة من قاعدة البيانات
- ✅ التعرفة المستخدمة هي نفسها في الحالتين

---

## Property 12: حساب أجزاء الرسالة بدقة

### الوصف
يتحقق من أن النظام يحسب عدد أجزاء الرسالة بدقة بناءً على قواعد التقسيم.

### الخاصية المختبرة
```
لأي نص رسالة، يجب أن يحسب النظام عدد الأجزاء بناءً على قواعد التقسيم
(segmentation_rules) المطابقة لنوع الترميز (encoding_type)
```

### المتطلبات المغطاة
- **Requirement 7.1**: GSM-7 encoding support
- **Requirement 7.2**: Unicode encoding support
- **Requirement 7.3**: Segmentation rules (160/153 for GSM-7, 70/67 for Unicode)
- **Requirement 7.6**: Accurate segment calculation

### قواعد التقسيم

#### GSM-7 Encoding
- **الجزء الأول**: 160 حرف
- **الجزء الثاني**: 153 حرف
- **الجزء الثالث وما بعده**: 153 حرف لكل جزء

**أمثلة**:
- 50 حرف → 1 جزء
- 160 حرف → 1 جزء
- 161 حرف → 2 جزء
- 313 حرف (160+153) → 2 جزء
- 314 حرف → 3 أجزاء

#### Unicode Encoding
- **الجزء الأول**: 70 حرف
- **الجزء الثاني**: 67 حرف
- **الجزء الثالث وما بعده**: 67 حرف لكل جزء

**أمثلة**:
- 30 حرف → 1 جزء
- 70 حرف → 1 جزء
- 71 حرف → 2 جزء
- 137 حرف (70+67) → 2 جزء
- 138 حرف → 3 أجزاء

### الاختبارات المطبقة

#### 1. TestProperty_CorrectSegmentCalculation
**الهدف**: التحقق من دقة حساب الأجزاء

**السيناريو**:
- اختبار 16 حالة معروفة (8 لـ GSM-7، 8 لـ Unicode)
- التحقق من تطابق النتائج مع القيم المتوقعة

**معايير النجاح**:
- ✅ جميع الحالات تعطي النتائج الصحيحة
- ✅ القواعد مطبقة بدقة

#### 2. TestProperty_SegmentsNeverLessThanOne
**الهدف**: التحقق من أن عدد الأجزاء لا يقل عن 1

**السيناريو**:
- اختبار 50 طول عشوائي (1-1000 حرف)
- التحقق من أن النتيجة >= 1

**معايير النجاح**:
- ✅ جميع النتائج >= 1
- ✅ لا توجد نتائج سالبة أو صفرية

#### 3. TestProperty_SegmentsIncreaseMonotonically
**الهدف**: التحقق من أن عدد الأجزاء يزداد بشكل رتيب

**السيناريو**:
- اختبار أطوال متزايدة (1، 10، 20، ..., 1000)
- التحقق من أن عدد الأجزاء لا ينقص أبداً

**معايير النجاح**:
- ✅ عدد الأجزاء يزداد أو يبقى ثابتاً
- ✅ لا توجد قفزات غير متوقعة

#### 4. TestProperty_GSM7SegmentationRules
**الهدف**: التحقق من تطبيق قواعد GSM-7 بدقة

**السيناريو**:
- اختبار حدود الأجزاء (160، 161، 313، 314، إلخ)
- التحقق من النتائج المتوقعة

**معايير النجاح**:
- ✅ 160 حرف = 1 جزء
- ✅ 161 حرف = 2 جزء
- ✅ 313 حرف = 2 جزء
- ✅ 314 حرف = 3 أجزاء

#### 5. TestProperty_UnicodeSegmentationRules
**الهدف**: التحقق من تطبيق قواعد Unicode بدقة

**السيناريو**:
- اختبار حدود الأجزاء (70، 71، 137، 138، إلخ)
- التحقق من النتائج المتوقعة

**معايير النجاح**:
- ✅ 70 حرف = 1 جزء
- ✅ 71 حرف = 2 جزء
- ✅ 137 حرف = 2 جزء
- ✅ 138 حرف = 3 أجزاء

#### 6. TestProperty_RandomMessageLengths
**الهدف**: اختبار شامل بأطوال عشوائية

**السيناريو**:
- اختبار 100 طول عشوائي (1-2000 حرف)
- التحقق من صحة الحسابات

**معايير النجاح**:
- ✅ جميع النتائج موجبة
- ✅ النتائج تطابق الحسابات اليدوية

#### 7. TestProperty_DefaultEncodingIsGSM7
**الهدف**: التحقق من أن الترميز الافتراضي هو GSM-7

**السيناريو**:
- حساب الأجزاء بدون تحديد الترميز
- حساب الأجزاء مع GSM-7 صريح
- التحقق من تطابق النتائج

**معايير النجاح**:
- ✅ الترميز الفارغ يعامل كـ GSM-7
- ✅ النتائج متطابقة

#### 8. TestProperty_InvalidMessageLength
**الهدف**: التحقق من معالجة الأطوال غير الصحيحة

**السيناريو**:
- اختبار أطوال غير صحيحة (0، -1، -10، -100)
- التحقق من إرجاع خطأ

**معايير النجاح**:
- ✅ جميع الأطوال غير الصحيحة ترجع خطأ
- ✅ رسالة الخطأ واضحة

---

## تشغيل Property Tests

### تشغيل جميع الاختبارات

```bash
# Windows PowerShell
cd services/backend
.\run-tariff-property-tests.ps1

# Linux/Mac
cd services/backend
chmod +x run-tariff-property-tests.sh
./run-tariff-property-tests.sh
```

### تشغيل اختبارات محددة

```bash
# Property 3: Active Tariff Selection
go test -v ./internal/tariff -run TestProperty_UseActiveTariffOnly

# Property 12: Segment Calculation
go test -v ./internal/tariff -run TestProperty_CorrectSegmentCalculation

# جميع اختبارات Property 3
go test -v ./internal/tariff -run TestProperty.*Tariff

# جميع اختبارات Property 12
go test -v ./internal/tariff -run TestProperty.*Segment
```

### تشغيل مع عدد تكرارات أكبر

```bash
# تشغيل 50 مرة للكشف عن مشاكل متقطعة
go test -v ./internal/tariff -run TestProperty -count=50

# تشغيل مع race detector
go test -v -race ./internal/tariff -run TestProperty
```

---

## تفسير النتائج

### نجاح الاختبار ✅

```
=== RUN   TestProperty_UseActiveTariffOnly
=== RUN   TestProperty_UseActiveTariffOnly/Iteration_1
    engine_property_test.go:89: Iteration 1: Active tariff used correctly (ID=5, Cost=9.0000)
=== RUN   TestProperty_UseActiveTariffOnly/Iteration_2
    engine_property_test.go:89: Iteration 2: Active tariff used correctly (ID=10, Cost=9.0000)
...
--- PASS: TestProperty_UseActiveTariffOnly (1.23s)
```

### فشل الاختبار ❌

```
=== RUN   TestProperty_UseActiveTariffOnly
=== RUN   TestProperty_UseActiveTariffOnly/Iteration_3
    engine_property_test.go:85: Property violated: Should use active tariff ID
        Expected: 5 (active), Got: 8 (inactive)
--- FAIL: TestProperty_UseActiveTariffOnly (0.45s)
```

**التفسير**: النظام استخدم تعرفة غير نشطة بدلاً من التعرفة النشطة.

---

## استكشاف الأخطاء

### المشكلة: Property Test يفشل بشكل متقطع

**الأسباب المحتملة**:
1. مشاكل في التخزين المؤقت (Cache)
2. مشاكل في التواريخ (timezone issues)
3. تعارض في البيانات الاختبارية

**الحل**:
```bash
# تنظيف Redis
docker exec -it redis redis-cli FLUSHDB

# تشغيل مع عدد تكرارات أكبر
go test -v ./internal/tariff -run TestProperty -count=100
```

### المشكلة: اختبارات التقسيم تفشل

**الأسباب المحتملة**:
1. قواعد التقسيم غير موجودة في قاعدة البيانات
2. قواعد التقسيم غير صحيحة

**الحل**:
```bash
# التحقق من قواعد التقسيم
psql -U postgres -d sms_gateway_test -c "SELECT * FROM segmentation_rules;"

# إعادة تشغيل Migration
go run cmd/migrate/main.go
```

### المشكلة: Property Test بطيء جداً

**الحل**:
```bash
# تقليل عدد التكرارات في الكود
# أو تشغيل في وضع short
go test -v -short ./internal/tariff
```

---

## أفضل الممارسات

### 1. عدد التكرارات

- **Development**: 10-20 تكرار
- **CI/CD**: 50-100 تكرار
- **Pre-release**: 200+ تكرار

### 2. تنظيف البيانات

```go
// تنظيف قاعدة البيانات بعد كل اختبار
defer cleanupTestData(t, db, tenantID)
```

### 3. عزل الاختبارات

```go
// استخدام tenant_id مختلف لكل اختبار
tenantID := createTestTenantForTariff(t, db)
```

### 4. تشغيل منتظم

```bash
# في CI/CD pipeline
go test -v -race ./internal/tariff -run TestProperty -count=50
```

---

## الخلاصة

Property-Based Tests توفر:

✅ **التحقق من الخصائص الرياضية** للنظام  
✅ **اختبار شامل** لسيناريوهات متعددة  
✅ **ثقة عالية** في صحة محرك التعرفات  
✅ **كشف مبكر** للمشاكل المحتملة  

### Properties المطبقة

- **Property 3**: استخدام التعرفة النشطة الصحيحة
- **Property 12**: حساب أجزاء الرسالة بدقة

### Requirements المغطاة

- ✅ **Requirement 2.5**: Active tariff selection
- ✅ **Requirement 7.1**: GSM-7 encoding support
- ✅ **Requirement 7.2**: Unicode encoding support
- ✅ **Requirement 7.3**: Segmentation rules
- ✅ **Requirement 7.6**: Accurate segment calculation
- ✅ **Requirement 27.9**: Tariff caching

---

**تاريخ الإنشاء**: 2026-02-27  
**الإصدار**: 1.0  
**الحالة**: جاهز للاستخدام  
**المهام المغطاة**: Task 3.2, Task 3.4
