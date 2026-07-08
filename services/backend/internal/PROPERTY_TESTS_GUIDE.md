# دليل Property-Based Tests الشامل
# نظام التعرفات والتسعير

## المقدمة

هذا الدليل يوفر نظرة شاملة على جميع Property-Based Tests المطبقة في نظام التعرفات والتسعير. Property-Based Testing هو أسلوب اختبار يتحقق من أن خصائص معينة (Properties) تظل صحيحة عبر مجموعة واسعة من المدخلات العشوائية.

## ما هو Property-Based Testing؟

Property-Based Testing يختلف عن Unit Testing التقليدي:

- **Unit Testing**: يختبر حالات محددة (مثال: `balance = 100`, `deduction = 50`)
- **Property-Based Testing**: يختبر خصائص عامة عبر آلاف الحالات العشوائية

### مثال توضيحي

```go
// Unit Test (حالة واحدة)
func TestDeductBalance(t *testing.T) {
    balance := 100.0
    deduction := 50.0
    result := balance - deduction
    assert.Equal(t, 50.0, result)
}

// Property Test (آلاف الحالات)
func TestProperty_BalanceNeverNegative(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        balance := rapid.Float64Range(0, 10000).Draw(t, "balance")
        deduction := rapid.Float64Range(0, 20000).Draw(t, "deduction")
        
        result := DeductBalance(balance, deduction)
        
        // Property: الرصيد لا يصبح سالباً أبداً
        assert.GreaterOrEqual(t, result, 0.0)
    })
}
```

## قائمة Properties المطبقة

### المرحلة 1: الأساسيات والبنية التحتية

#### Property 1-2: Balance Operations
**الملف**: `internal/balance/manager_property_test.go`  
**الوظيفة**: `TestProperty_AtomicBalanceOperations`  
**المتطلبات**: Requirements 16.1, 16.2

**الخاصية**: تحديث التعرفة يحفظ التاريخ والاحتفاظ بالسجل التاريخي

**ما يتم اختباره**:
- تحديث التعرفة يحفظ `effective_from` للتعرفة الجديدة
- النسخة القديمة تبقى في قاعدة البيانات مع `effective_to`
- السجل التاريخي محفوظ بشكل صحيح

**كيفية التشغيل**:
```powershell
cd services/backend
go test -v ./internal/balance -run TestProperty_AtomicBalanceOperations
```

---

#### Property 15: Atomic Balance Operations
**الملف**: `internal/balance/manager_property_test.go`  
**الوظيفة**: `TestProperty_NoRaceConditionsInBalanceOperations`  
**المتطلبات**: Requirements 16.1, 16.2

**الخاصية**: لأي عمليتي خصم متزامنتين على نفس الحساب، يجب أن تنجح واحدة فقط إذا كان الرصيد كافياً لواحدة فقط، ويجب ألا يصبح الرصيد سالباً أبداً

**ما يتم اختباره**:
1. Atomic balance deduction مع حماية من Race Conditions
2. الرصيد لا يصبح سالباً أبداً تحت العمليات المتزامنة
3. عملية واحدة فقط تنجح عندما يكون الرصيد كافياً لواحدة فقط
4. جميع العمليات الناجحة مسجلة في `balance_history`

**الاختبارات الفرعية**:
- `TestProperty_AtomicBalanceOperations`: اختبار العمليات الذرية
- `TestProperty_NoRaceConditionsInBalanceOperations`: اختبار عدم وجود Race Conditions
- `TestProperty_ExactlyOneSucceedsWhenBalanceSufficientForOne`: اختبار نجاح عملية واحدة فقط
- `TestProperty_AllSuccessfulOperationsAreLogged`: اختبار تسجيل جميع العمليات
- `TestProperty_BalanceNeverGoesNegative`: اختبار عدم سلبية الرصيد

**كيفية التشغيل**:
```powershell
cd services/backend
go test -v ./internal/balance -run TestProperty_NoRaceConditionsInBalanceOperations
```

**مثال على النتيجة**:
```
Iteration 1: Initial=15234.50, Final=8456.30, Deducted=6778.20, Success=42/50
Iteration 2: Initial=18923.10, Final=12345.67, Deducted=6577.43, Success=38/50
...
PASS: All properties verified across 10 iterations
```

---

#### Property 3: Active Tariff Usage
**الملف**: `internal/tariff/engine_property_test.go`  
**الوظيفة**: `TestProperty_UseActiveTariffOnly`  
**المتطلبات**: Requirements 2.5

**الخاصية**: لأي رسالة مع شبكة محددة، عند حساب التكلفة، يجب أن يستخدم النظام التعرفة النشطة المطابقة للشبكة

**ما يتم اختباره**:
1. فقط التعرفات النشطة (`is_active = true`) تُستخدم
2. التعرفات تحترم `effective_from` و `effective_to`
3. التعرفة الأكثر تحديداً يتم اختيارها (network_operator > country)
4. التعرفات غير النشطة لا تُستخدم أبداً

**الاختبارات الفرعية**:
- `TestProperty_UseActiveTariffOnly`: اختبار استخدام التعرفة النشطة فقط
- `TestProperty_TariffEffectiveDateBoundaries`: اختبار حدود تواريخ السريان
- `TestProperty_MostSpecificTariffSelection`: اختبار اختيار التعرفة الأكثر تحديداً
- `TestProperty_InactiveTariffsNeverUsed`: اختبار عدم استخدام التعرفات غير النشطة
- `TestProperty_TariffCacheConsistency`: اختبار اتساق الـ Cache

**كيفية التشغيل**:
```powershell
cd services/backend
go test -v ./internal/tariff -run TestProperty_UseActiveTariffOnly
```

---

#### Property 12: Message Segmentation
**الملف**: `internal/tariff/segmentation_property_test.go`  
**الوظيفة**: `TestProperty_MessageSegmentation`  
**المتطلبات**: Requirements 7.6

**الخاصية**: لأي نص رسالة، يجب أن يحسب النظام عدد الأجزاء بناءً على قواعد التقسيم المطابقة لنوع الترميز

**ما يتم اختباره**:
1. حساب عدد الأجزاء بدقة حسب قواعد التقسيم
2. دعم GSM-7 و Unicode
3. تطبيق القواعد: الجزء الأول 70، الثاني 65، الثالث 62
4. حساب التكلفة الإجمالية بشكل صحيح

**كيفية التشغيل**:
```powershell
cd services/backend
go test -v ./internal/tariff -run TestProperty_MessageSegmentation
```

---

### المرحلة 2: الباقات الشهرية

#### Property 5: Safe Subscription
**الملف**: `internal/subscription/manager_property_test.go`  
**الوظيفة**: `TestProperty_SafeSubscription`  
**المتطلبات**: Requirements 4.1, 4.2

**الخاصية**: لأي باقة وعميل، عند محاولة الاشتراك، إذا كان الرصيد >= سعر الباقة، يجب أن ينجح الاشتراك ويخصم السعر من الرصيد بشكل ذري

**ما يتم اختباره**:
1. الاشتراك ينجح فقط عندما يكون الرصيد كافياً
2. الرصيد يُخصم بشكل ذري (Atomic)
3. الاشتراك يُنشأ بشكل صحيح
4. معاملة المحاسبة مسجلة
5. لا توجد Race Conditions أثناء الاشتراكات المتزامنة

**الاختبارات الفرعية**:
- `TestProperty_SafeSubscription`: اختبار الاشتراك الآمن
- `TestProperty_AtomicSubscription`: اختبار الذرية
- `TestProperty_ConcurrentSubscriptions`: اختبار الاشتراكات المتزامنة
- `TestProperty_SubscriptionIdempotency`: اختبار Idempotency

**كيفية التشغيل**:
```powershell
cd services/backend
go test -v ./internal/subscription -run TestProperty_SafeSubscription
```

---

#### Property 6: Message Deduction
**الملف**: `internal/subscription/manager_property_test.go`  
**الوظيفة**: `TestProperty_MessageDeduction`  
**المتطلبات**: Requirements 4.5

**الخاصية**: لأي اشتراك نشط مع رسائل متاحة > 0، عند إرسال رسالة، يجب أن ينخفض عداد `messages_remaining` بمقدار 1

**ما يتم اختباره**:
1. `messages_remaining` ينخفض بشكل صحيح
2. `messages_used` يزداد بشكل صحيح
3. مجموع `messages_used + messages_remaining` يساوي الحصة الأصلية
4. الخصومات المتزامنة لا تسبب Race Conditions
5. الخصم ذري (إما ينجح كلياً أو يفشل كلياً)
6. الباقات غير المحدودة تتعامل مع الخصومات بشكل صحيح

**الاختبارات الفرعية**:
- `TestProperty_MessageDeduction`: اختبار خصم الرسائل
- `TestProperty_ConcurrentMessageDeduction`: اختبار الخصومات المتزامنة
- `TestProperty_UnlimitedPackageDeduction`: اختبار الباقات غير المحدودة
- `TestProperty_MessageDeductionAtomicity`: اختبار الذرية
- `TestProperty_MessageDeductionInvariant`: اختبار الثابت

**كيفية التشغيل**:
```powershell
cd services/backend
go test -v ./internal/subscription -run TestProperty_MessageDeduction
```

---

#### Property 8: Overage Charging
**الملف**: `internal/subscription/manager_property_test.go`  
**الوظيفة**: `TestProperty_OverageCharging`  
**المتطلبات**: Requirements 5.3, 5.4

**الخاصية**: لأي رسالة إضافية (overage) مع رصيد >= تكلفة الرسالة، يجب أن يخصم النظام التكلفة من الرصيد بشكل ذري

**ما يتم اختباره**:
1. تكلفة Overage تُحسب بشكل صحيح باستخدام `overage_rate`
2. رصيد العميل يُفحص قبل الخصم
3. الرصيد يُخصم بشكل ذري عندما يكون كافياً
4. المعاملة تفشل عندما يكون الرصيد غير كافي
5. `overage_messages` و `overage_cost` يتم تحديثهما بشكل صحيح
6. معاملة المحاسبة مسجلة
7. الباقات غير المحدودة لا تخصم تكلفة Overage

**كيفية التشغيل**:
```powershell
cd services/backend
go test -v ./internal/subscription -run TestProperty_OverageCharging
```

---

### المرحلة 3: نظام Rollover

#### Property 9: Calculate Remaining Messages
**الملف**: `internal/rollover/manager_property_test.go`  
**الوظيفة**: `TestProperty_CalculateRemainingMessages`  
**المتطلبات**: Requirements 6.2

**الخاصية**: لأي اشتراك نشط مع `rollover_enabled = true`، عند انتهاء الشهر، يجب أن يحسب النظام `messages_remaining` ويسجلها

**ما يتم اختباره**:
1. يعيد 0 عندما يكون Rollover معطلاً
2. يعيد 0 عندما تكون الباقة غير محدودة
3. يعيد 0 عندما تكون `messages_remaining` صفر أو سالبة
4. يعيد `messages_remaining` الصحيحة عندما يكون Rollover مفعلاً
5. يتعامل مع الحالات الحدية بشكل صحيح

**الاختبارات الفرعية**:
- `TestProperty_CalculateRemainingMessages`: اختبار حساب الرسائل المتبقية
- `TestProperty_CalculateRemainingMessages_Idempotency`: اختبار Idempotency
- `TestProperty_CalculateRemainingMessages_BoundaryValues`: اختبار القيم الحدية

**كيفية التشغيل**:
```powershell
cd services/backend
go test -v ./internal/rollover -run TestProperty_CalculateRemainingMessages
```

---

#### Property 10: Rollover Limit Respect
**الملف**: `internal/rollover/manager_property_test.go`  
**الوظيفة**: `TestProperty_RolloverMessages_RespectLimit`  
**المتطلبات**: Requirements 6.5

**الخاصية**: لأي باقة مع `rollover_limit` محدد، عند ترحيل الرسائل، يجب أن يكون عدد الرسائل المرحّلة <= `rollover_limit`

**ما يتم اختباره**:
1. لا يتم ترحيل أكثر من `rollover_limit` أبداً
2. يتم ترحيل جميع الرسائل إذا كانت `remaining < limit`
3. يتم ترحيل `limit` بالضبط إذا كانت `remaining >= limit`
4. يتعامل مع `rollover_limit` NULL (ترحيل غير محدود)

**الاختبارات الفرعية**:
- `TestProperty_RolloverMessages_RespectLimit`: اختبار احترام الحد الأقصى
- `TestProperty_RolloverMessages_ExpiryDate`: اختبار تاريخ الانتهاء
- `TestProperty_RolloverMessages_Idempotency`: اختبار Idempotency

**كيفية التشغيل**:
```powershell
cd services/backend
go test -v ./internal/rollover -run TestProperty_RolloverMessages_RespectLimit
```

---

#### Property 11: FIFO Rollover Usage
**الملف**: `internal/rollover/manager_property_test.go`  
**الوظيفة**: `TestProperty_UseRolloverMessages_FIFO`  
**المتطلبات**: Requirements 6.8

**الخاصية**: لأي عميل مع رسائل مرحّلة من عدة أشهر، عند إرسال رسالة، يجب أن يستخدم النظام الرسائل من أقدم `source_month` أولاً

**ما يتم اختباره**:
1. يستخدم أقدم رسائل Rollover أولاً (FIFO order)
2. يحدّث أو يحذف سجلات Rollover بشكل صحيح
3. يعيد عدد الرسائل المستخدمة الصحيح
4. يتعامل مع الاستهلاك الجزئي بشكل صحيح

**الاختبارات الفرعية**:
- `TestProperty_UseRolloverMessages_FIFO`: اختبار FIFO
- `TestProperty_UseRolloverMessages_FullConsumption`: اختبار الاستهلاك الكامل
- `TestProperty_UseRolloverMessages_PartialConsumption`: اختبار الاستهلاك الجزئي
- `TestProperty_UseRolloverMessages_NoAvailable`: اختبار عدم وجود رسائل متاحة

**كيفية التشغيل**:
```powershell
cd services/backend
go test -v ./internal/rollover -run TestProperty_UseRolloverMessages_FIFO
```

---

### المرحلة 4: الفوترة التلقائية

#### Property 13: Auto Renewal
**الملف**: `internal/billing/auto_renewal_property_test.go`  
**الوظيفة**: `TestProperty_AutoRenewal`  
**المتطلبات**: Requirements 14.2, 14.3

**الخاصية**: لأي اشتراك مع `auto_renew = true` و `end_date <= CURRENT_DATE`، إذا كان الرصيد >= سعر الباقة، يجب أن يجدد النظام الاشتراك تلقائياً

**ما يتم اختباره**:
1. التجديد ينجح عندما يكون الرصيد كافياً
2. الرصيد يُخصم بشكل صحيح
3. التواريخ تُحدّث بشكل صحيح
4. العدادات تُعاد تعيينها
5. التجديد يفشل عندما يكون الرصيد غير كافي
6. فترة السماح تُطبق عند الفشل

**الاختبارات الفرعية**:
- `TestProperty_AutoRenewal`: اختبار التجديد التلقائي
- `TestProperty_GracePeriodApplication`: اختبار تطبيق فترة السماح
- `TestProperty_SubscriptionSuspension`: اختبار تعليق الاشتراك
- `TestProperty_AutoReactivation`: اختبار إعادة التفعيل التلقائي

**كيفية التشغيل**:
```powershell
cd services/backend
go test -v ./internal/billing -run TestProperty_AutoRenewal
```

---

### المرحلة 5: البوابات المخصصة

#### Property 20: Rate Limiting per Tenant
**الملف**: `internal/resources/limiter_property_test.go`  
**الوظيفة**: `TestProperty_RateLimitingPerTenant`  
**المتطلبات**: Requirements 9.8

**الخاصية**: لأي tenant يتجاوز حد الـ rate limit الخاص به، يجب أن يرفض النظام طلباته دون التأثير على tenants أخرى

**ما يتم اختباره**:
1. Rate limits تُطبق لكل tenant
2. tenant واحد يتجاوز حده لا يؤثر على tenants أخرى
3. Rate limits معزولة بين tenants
4. Token bucket algorithm يعمل بشكل صحيح
5. Rate limits يمكن تحديثها ديناميكياً

**الاختبارات الفرعية**:
- `TestProperty_RateLimitingPerTenant`: اختبار Rate Limiting لكل Tenant
- `TestProperty_RateLimitIsolationUnderLoad`: اختبار العزل تحت الحمل
- `TestProperty_RateLimitDoesNotExceedQuota`: اختبار عدم تجاوز الحصة
- `TestProperty_RateLimitRecoveryAfterWait`: اختبار الاسترداد بعد الانتظار
- `TestProperty_MessageRateLimitIsolation`: اختبار عزل حدود الرسائل
- `TestProperty_QueryRateLimitIsolation`: اختبار عزل حدود الاستعلامات
- `TestProperty_RateLimitUpdatesDynamically`: اختبار التحديث الديناميكي
- `TestProperty_RateLimitStatusReporting`: اختبار تقرير الحالة

**كيفية التشغيل**:
```powershell
cd services/backend
go test -v ./internal/resources -run TestProperty_RateLimitingPerTenant
```

---

## Properties غير المطبقة (للمرجع)

### Property 4: حساب سعر الرسالة الواحدة تلقائياً
**الحالة**: غير مطبق كـ Property Test (مغطى بـ Unit Tests)  
**السبب**: حساب بسيط لا يحتاج Property Testing

### Property 7: السماح بالاستمرار عند نفاد الرسائل (Overage)
**الحالة**: مغطى ضمن Property 8

### Property 14: تسجيل كل عملية مالية
**الحالة**: مغطى ضمن Properties 15, 5, 8, 13

### Property 16: Tenant Isolation في قاعدة البيانات
**الحالة**: مغطى بـ Integration Tests في `internal/middleware/tenant_isolation_integration_test.go`

### Property 17: Tenant Isolation في Redis
**الحالة**: مغطى بـ Integration Tests

### Property 18: Cross-Tenant Access Prevention
**الحالة**: مغطى بـ Integration Tests في `internal/middleware/cross_tenant_prevention_test.go`

### Property 19: Worker Pool Isolation
**الحالة**: مغطى بـ Integration Tests في `internal/resources/tenant_pool_test.go`

---

## كيفية تشغيل جميع Property Tests

### تشغيل جميع الاختبارات
```powershell
cd services/backend
.\run-all-property-tests.ps1
```

### تشغيل اختبارات package محدد
```powershell
.\run-all-property-tests.ps1 -Package balance
.\run-all-property-tests.ps1 -Package tariff
.\run-all-property-tests.ps1 -Package subscription
.\run-all-property-tests.ps1 -Package rollover
.\run-all-property-tests.ps1 -Package billing
.\run-all-property-tests.ps1 -Package resources
```

### تشغيل مع Verbose Output
```powershell
.\run-all-property-tests.ps1 -Verbose
```

### تشغيل في Short Mode (أسرع)
```powershell
.\run-all-property-tests.ps1 -Short
```

---

## فهم نتائج Property Tests

### نتيجة ناجحة
```
✓ PASSED
Duration: 12.45s
Iteration 1: Initial=15234.50, Final=8456.30, Deducted=6778.20, Success=42/50
Iteration 2: Initial=18923.10, Final=12345.67, Deducted=6577.43, Success=38/50
...
PASS: All properties verified across 10 iterations
```

### نتيجة فاشلة
```
✗ FAILED
Duration: 5.23s
Property violated: Balance should never be negative
  Initial: 100.00
  Final: -50.00
  Deduction: 150.00
Counterexample found after 234 iterations
```

---

## أفضل الممارسات

### 1. عدد التكرارات
- **Development**: 10-20 تكرار (سريع)
- **CI/CD**: 100-200 تكرار (متوسط)
- **Pre-Release**: 1000+ تكرار (شامل)

### 2. توليد البيانات العشوائية
```go
// جيد: نطاق واقعي
balance := rapid.Float64Range(0, 100000).Draw(t, "balance")

// سيء: نطاق غير واقعي
balance := rapid.Float64Range(-1000000, 1000000).Draw(t, "balance")
```

### 3. التحقق من Properties
```go
// جيد: property واضح ومحدد
assert.GreaterOrEqual(t, finalBalance, 0.0, 
    "Balance should never be negative")

// سيء: property غامض
assert.True(t, finalBalance >= 0.0)
```

### 4. Cleanup
```go
// دائماً استخدم defer للتنظيف
defer cleanupTestData(t, ctx, pool, tenantID)
```

---

## استكشاف الأخطاء

### مشكلة: الاختبارات تفشل بشكل عشوائي
**الحل**: قد تكون هناك Race Condition. استخدم `-race` flag:
```powershell
go test -race -v ./internal/balance -run TestProperty_AtomicBalanceOperations
```

### مشكلة: الاختبارات بطيئة جداً
**الحل**: استخدم `-short` flag أو قلل عدد التكرارات:
```powershell
go test -short -v ./internal/balance
```

### مشكلة: فشل الاتصال بقاعدة البيانات
**الحل**: تأكد من تشغيل PostgreSQL وأن قاعدة البيانات موجودة:
```powershell
psql -U postgres -c "CREATE DATABASE sms_gateway_test;"
```

---

## الخلاصة

Property-Based Testing يوفر:
- ✅ تغطية أوسع من Unit Tests التقليدية
- ✅ اكتشاف حالات حدية غير متوقعة
- ✅ ثقة أكبر في صحة النظام
- ✅ توثيق تلقائي للخصائص المطلوبة

جميع الـ Properties الحرجة (1-20) مغطاة إما بـ Property Tests أو Integration Tests، مما يضمن موثوقية نظام التعرفات والتسعير.

---

## المراجع

- [Property-Based Testing Guide](https://hypothesis.works/articles/what-is-property-based-testing/)
- [Rapid Documentation](https://pkg.go.dev/pgregory.net/rapid)
- [Gopter Documentation](https://pkg.go.dev/github.com/leanovate/gopter)
- [Design Document](../../.kiro/specs/tariff-pricing-system/design.md)
- [Requirements Document](../../.kiro/specs/tariff-pricing-system/requirements.md)
