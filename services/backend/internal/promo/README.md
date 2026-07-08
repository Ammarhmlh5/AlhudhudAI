# نظام الخصومات والعروض الترويجية (Promo Codes System)

## نظرة عامة

نظام شامل لإدارة أكواد الخصم والعروض الترويجية مع دعم كامل لـ Multi-Tenancy Isolation.

## المتطلبات المغطاة

- **32.1**: إنشاء أكواد خصم (Promo Codes)
- **32.2**: دعم خصومات بنسبة مئوية أو مبلغ ثابت
- **32.3**: التحقق من صلاحية الكود
- **32.4**: تطبيق الخصم على الاشتراكات
- **32.5**: دعم خصومات بنسبة مئوية أو مبلغ ثابت
- **32.6**: تحديد شروط الاستخدام (حد أقصى، تاريخ انتهاء)
- **32.7**: تحديد الخدمات المؤهلة
- **32.8**: منع الاستخدام المتكرر
- **32.9**: تسجيل استخدام الأكواد
- **32.10**: تتبع الاستخدام لكل عميل
- **32.11**: تقارير أداء العروض

## المكونات الرئيسية

### 1. Types (types.go)

يحتوي على جميع الأنواع والهياكل المستخدمة في النظام:

```go
// أنواع الخصم
type DiscountType string
const (
    DiscountTypePercentage  DiscountType = "percentage"
    DiscountTypeFixedAmount DiscountType = "fixed_amount"
)

// نطاق التطبيق
type ApplicableTo string
const (
    ApplicableToAll      ApplicableTo = "all"
    ApplicableToPackages ApplicableTo = "packages"
    ApplicableToTopup    ApplicableTo = "topup"
    ApplicableToGateway  ApplicableTo = "gateway"
)
```

### 2. Manager (manager.go)

المدير الرئيسي لأكواد الخصم:

#### الوظائف الأساسية:

- **CreatePromoCode**: إنشاء كود خصم جديد
- **ValidatePromoCode**: التحقق من صلاحية الكود
- **ApplyPromoCode**: تطبيق الكود وحساب الخصم
- **GetPromoPerformance**: الحصول على أداء الكود
- **ListPromoCodes**: عرض قائمة الأكواد
- **UpdatePromoCode**: تحديث كود موجود
- **DeletePromoCode**: حذف كود (soft delete)

#### مثال الاستخدام:

```go
manager := promo.NewManager(db)

// إنشاء كود خصم 20%
req := &promo.CreatePromoCodeRequest{
    Code:          "SUMMER2026",
    Description:   "خصم صيفي 20%",
    DiscountType:  promo.DiscountTypePercentage,
    DiscountValue: 20.0,
    ValidFrom:     time.Now(),
    ValidUntil:    time.Now().Add(30 * 24 * time.Hour),
    PerUserLimit:  1,
    ApplicableTo:  promo.ApplicableToAll,
}

promoCode, err := manager.CreatePromoCode(ctx, tenantID, req)
```

### 3. Usage Tracker (tracker.go)

متتبع استخدام أكواد الخصم:

#### الوظائف الأساسية:

- **RecordUsage**: تسجيل استخدام كود
- **GetUserUsageCount**: عدد مرات استخدام المستخدم
- **HasUserUsedPromoCode**: التحقق من استخدام المستخدم
- **CanUserUsePromoCode**: التحقق من إمكانية الاستخدام
- **GetPromoCodeUsageStats**: إحصائيات الاستخدام
- **GetUsageByDateRange**: الاستخدامات في فترة زمنية
- **GetUsageByOrderType**: الاستخدامات حسب نوع الطلب
- **GetTopUsers**: أكثر المستخدمين استخداماً
- **GetDailyUsageStats**: إحصائيات يومية
- **PreventDuplicateUsage**: منع الاستخدام المتكرر

#### مثال الاستخدام:

```go
tracker := promo.NewUsageTracker(db)

// الحصول على إحصائيات الاستخدام
stats, err := tracker.GetPromoCodeUsageStats(ctx, tenantID, promoCodeID)

// الحصول على أكثر 10 مستخدمين استخداماً
topUsers, err := tracker.GetTopUsers(ctx, tenantID, promoCodeID, 10)
```

### 4. API Handler (api/promo_handler.go)

معالج API لأكواد الخصم:

#### المسارات المتاحة:

**مسارات الإدارة (Admin):**
- `POST /api/admin/promo-codes` - إنشاء كود خصم
- `GET /api/admin/promo-codes` - قائمة الأكواد
- `GET /api/admin/promo-codes/:id` - تفاصيل كود
- `PUT /api/admin/promo-codes/:id` - تحديث كود
- `DELETE /api/admin/promo-codes/:id` - حذف كود
- `GET /api/admin/promo-codes/:id/performance` - أداء الكود
- `GET /api/admin/promo-codes/:id/stats` - إحصائيات الاستخدام
- `GET /api/admin/promo-codes/:id/top-users` - أكثر المستخدمين
- `GET /api/admin/promo-codes/:id/daily-stats` - إحصائيات يومية

**مسارات العملاء (Client):**
- `POST /api/promo-codes/validate` - التحقق من صلاحية كود
- `POST /api/promo-codes/apply` - تطبيق كود خصم

## قاعدة البيانات

### جدول promo_codes

```sql
CREATE TABLE promo_codes (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    code VARCHAR(50) NOT NULL,
    description TEXT,
    discount_type VARCHAR(20) NOT NULL,
    discount_value DECIMAL(10, 2) NOT NULL,
    max_discount_amount DECIMAL(10, 2),
    min_purchase_amount DECIMAL(10, 2),
    valid_from TIMESTAMP NOT NULL,
    valid_until TIMESTAMP NOT NULL,
    usage_limit INT,
    usage_count INT NOT NULL DEFAULT 0,
    per_user_limit INT DEFAULT 1,
    applicable_to VARCHAR(20) NOT NULL DEFAULT 'all',
    applicable_package_ids BIGINT[],
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT
);
```

### جدول promo_code_usage

```sql
CREATE TABLE promo_code_usage (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    promo_code_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    order_type VARCHAR(20) NOT NULL,
    order_id BIGINT,
    original_amount DECIMAL(10, 2) NOT NULL,
    discount_amount DECIMAL(10, 2) NOT NULL,
    final_amount DECIMAL(10, 2) NOT NULL,
    used_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

## Multi-Tenancy Isolation

النظام يطبق عزل تام بين الشركات:

### 1. Row Level Security (RLS)

جميع الجداول محمية بـ RLS Policies:

```sql
ALTER TABLE promo_codes ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON promo_codes
    USING (tenant_id = current_setting('app.current_tenant')::BIGINT);
```

### 2. Tenant ID في جميع الاستعلامات

جميع الاستعلامات تتضمن `tenant_id`:

```go
query := `
    SELECT * FROM promo_codes 
    WHERE tenant_id = $1 AND code = $2
`
```

### 3. Indexes مركبة

جميع الـ Indexes تبدأ بـ `tenant_id`:

```sql
CREATE INDEX idx_promo_codes_tenant_code ON promo_codes(tenant_id, code);
CREATE INDEX idx_promo_codes_tenant_active ON promo_codes(tenant_id, is_active);
```

## أمثلة الاستخدام

### 1. إنشاء كود خصم بنسبة مئوية

```go
req := &promo.CreatePromoCodeRequest{
    Code:          "SUMMER20",
    Description:   "خصم صيفي 20%",
    DiscountType:  promo.DiscountTypePercentage,
    DiscountValue: 20.0,
    ValidFrom:     time.Now(),
    ValidUntil:    time.Now().Add(30 * 24 * time.Hour),
    PerUserLimit:  1,
    ApplicableTo:  promo.ApplicableToAll,
}

promoCode, err := manager.CreatePromoCode(ctx, tenantID, req)
```

### 2. إنشاء كود خصم بمبلغ ثابت

```go
maxDiscount := 50.0
minPurchase := 100.0
usageLimit := 100

req := &promo.CreatePromoCodeRequest{
    Code:              "FIXED50",
    Description:       "خصم 50 ريال",
    DiscountType:      promo.DiscountTypeFixedAmount,
    DiscountValue:     50.0,
    MaxDiscountAmount: &maxDiscount,
    MinPurchaseAmount: &minPurchase,
    ValidFrom:         time.Now(),
    ValidUntil:        time.Now().Add(30 * 24 * time.Hour),
    UsageLimit:        &usageLimit,
    PerUserLimit:      2,
    ApplicableTo:      promo.ApplicableToPackages,
    ApplicablePackageIDs: []int64{1, 2, 3},
}

promoCode, err := manager.CreatePromoCode(ctx, tenantID, req)
```

### 3. التحقق من صلاحية كود

```go
req := &promo.ApplyPromoCodeRequest{
    Code:      "SUMMER20",
    UserID:    userID,
    OrderType: promo.OrderTypeSubscription,
    Amount:    100.0,
    PackageID: &packageID,
}

promoCode, err := manager.ValidatePromoCode(ctx, tenantID, req)
if err != nil {
    // معالجة الخطأ
    if err == promo.ErrPromoCodeExpired {
        // الكود منتهي
    }
}
```

### 4. تطبيق كود خصم

```go
req := &promo.ApplyPromoCodeRequest{
    Code:      "SUMMER20",
    UserID:    userID,
    OrderType: promo.OrderTypeSubscription,
    Amount:    100.0,
}

response, err := manager.ApplyPromoCode(ctx, tenantID, req)
if err == nil {
    fmt.Printf("المبلغ الأصلي: %.2f\n", response.OriginalAmount)
    fmt.Printf("قيمة الخصم: %.2f\n", response.DiscountAmount)
    fmt.Printf("المبلغ النهائي: %.2f\n", response.FinalAmount)
}
```

### 5. الحصول على أداء كود خصم

```go
performance, err := manager.GetPromoPerformance(ctx, tenantID, promoCodeID)
if err == nil {
    fmt.Printf("عدد الاستخدامات: %d\n", performance.TotalUsageCount)
    fmt.Printf("إجمالي الخصم: %.2f\n", performance.TotalDiscountGiven)
    fmt.Printf("إجمالي الإيرادات: %.2f\n", performance.TotalRevenue)
    fmt.Printf("عدد المستخدمين: %d\n", performance.UniqueUsers)
}
```

## الاختبارات

### تشغيل الاختبارات

```bash
# اختبارات Manager
go test -v ./internal/promo -run TestManager

# اختبارات Tracker
go test -v ./internal/promo -run TestTracker

# اختبارات API
go test -v ./internal/api -run TestPromo

# جميع الاختبارات
go test -v ./internal/promo/... ./internal/api/...
```

### تغطية الاختبارات

```bash
go test -cover ./internal/promo/...
```

## معالجة الأخطاء

النظام يوفر أخطاء واضحة ومفيدة:

```go
var (
    ErrPromoCodeNotFound          = &ValidationError{...}
    ErrPromoCodeExpired           = &ValidationError{...}
    ErrPromoCodeNotStarted        = &ValidationError{...}
    ErrPromoCodeInactive          = &ValidationError{...}
    ErrPromoCodeUsageLimitReached = &ValidationError{...}
    ErrPromoCodeUserLimitReached  = &ValidationError{...}
    ErrPromoCodeNotApplicable     = &ValidationError{...}
    ErrMinPurchaseNotMet          = &ValidationError{...}
    ErrPromoCodeAlreadyExists     = &ValidationError{...}
)
```

## الأمان

### 1. Tenant Isolation

- جميع العمليات معزولة بـ `tenant_id`
- RLS Policies على جميع الجداول
- Indexes مركبة تبدأ بـ `tenant_id`

### 2. Validation

- التحقق من صحة جميع المدخلات
- منع SQL Injection
- التحقق من الصلاحيات

### 3. Audit Trail

- تسجيل جميع عمليات الاستخدام
- تتبع المستخدم والوقت
- سجل كامل للتغييرات

## الأداء

### 1. Indexes

جميع الاستعلامات محسّنة بـ Indexes مناسبة:

```sql
CREATE INDEX idx_promo_codes_tenant_code ON promo_codes(tenant_id, code);
CREATE INDEX idx_promo_codes_tenant_active ON promo_codes(tenant_id, is_active);
CREATE INDEX idx_promo_usage_tenant ON promo_code_usage(tenant_id);
CREATE INDEX idx_promo_usage_promo_code ON promo_code_usage(tenant_id, promo_code_id);
```

### 2. Triggers

Triggers تلقائية لتحديث العدادات:

```sql
CREATE TRIGGER increment_promo_usage_trigger
    AFTER INSERT ON promo_code_usage
    FOR EACH ROW
    EXECUTE FUNCTION increment_promo_code_usage();
```

### 3. Caching

يمكن إضافة Caching للأكواد النشطة:

```go
// مثال على Caching
key := fmt.Sprintf("promo:tenant:%d:code:%s", tenantID, code)
cached, err := redis.Get(ctx, key).Result()
```

## التكامل مع الأنظمة الأخرى

### 1. Subscription System

```go
// تطبيق كود خصم على اشتراك
req := &promo.ApplyPromoCodeRequest{
    Code:      promoCode,
    UserID:    userID,
    OrderType: promo.OrderTypeSubscription,
    OrderID:   &subscriptionID,
    Amount:    packagePrice,
    PackageID: &packageID,
}

response, err := promoManager.ApplyPromoCode(ctx, tenantID, req)
finalPrice := response.FinalAmount
```

### 2. Balance System

```go
// خصم المبلغ النهائي بعد تطبيق الكود
err := balanceManager.DeductBalance(ctx, tenantID, response.FinalAmount, "subscription_with_promo")
```

### 3. Accounting System

```go
// تسجيل المعاملة مع تفاصيل الخصم
transaction := &AccountingTransaction{
    TenantID:       tenantID,
    Type:           "subscription",
    Amount:         response.OriginalAmount,
    DiscountAmount: response.DiscountAmount,
    FinalAmount:    response.FinalAmount,
    PromoCodeID:    &promoCodeID,
}
```

## الخلاصة

نظام شامل ومرن لإدارة أكواد الخصم مع:

- ✅ دعم كامل لـ Multi-Tenancy Isolation
- ✅ أنواع خصم متعددة (نسبة مئوية، مبلغ ثابت)
- ✅ شروط استخدام مرنة
- ✅ تتبع دقيق للاستخدام
- ✅ تقارير أداء شاملة
- ✅ API كامل للإدارة والاستخدام
- ✅ اختبارات شاملة
- ✅ أمان عالي
- ✅ أداء محسّن

---

**تاريخ الإنشاء:** 2026-03-02  
**الإصدار:** 1.0  
**الحالة:** جاهز للإنتاج
