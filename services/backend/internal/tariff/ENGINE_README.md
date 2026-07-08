# Tariff Engine - محرك التعرفات

## نظرة عامة

محرك التعرفات مسؤول عن حساب تكلفة الرسائل بناءً على التعرفات الديناميكية وقواعد تقسيم الرسائل.

## المميزات الرئيسية

### 1. حساب التكلفة الديناميكي
- حساب التكلفة بناءً على التعرفات النشطة
- دعم تعرفات مختلفة حسب:
  - الخدمة (Service)
  - الدولة (Country)
  - شركة الاتصالات (Network Operator)
  - نوع الرسالة (GSM-7 / Unicode)

### 2. حساب أجزاء الرسائل
- تقسيم تلقائي للرسائل الطويلة
- دعم GSM-7 و Unicode
- قواعد تقسيم قابلة للتخصيص

### 3. التخزين المؤقت (Caching)
- تخزين التعرفات في Redis لمدة ساعة
- تحسين الأداء وتقليل الضغط على قاعدة البيانات
- إبطال التخزين المؤقت عند تحديث التعرفات

## الاستخدام

### حساب تكلفة رسالة

```go
engine := tariff.NewEngine(db, cache, logger)

req := tariff.CostCalculationRequest{
    TenantID:        1,
    ServiceID:       1,
    CountryCode:     "YE",
    NetworkOperator: "Yemen Mobile",
    MessageType:     "GSM-7",
    MessageLength:   200,
}

result, err := engine.CalculateCost(ctx, req)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Total Cost: %s %s\n", result.TotalCost, result.CurrencyCode)
fmt.Printf("Segments: %d\n", result.Segments)
fmt.Printf("Cost per Segment: %s\n", result.CostPerSegment)
```

### حساب عدد الأجزاء فقط

```go
segments, err := engine.CalculateSegments(ctx, 200, "GSM-7")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Message will be split into %d segments\n", segments)
```

### الحصول على التعرفة النشطة

```go
req := tariff.TariffRequest{
    TenantID:        1,
    ServiceID:       1,
    CountryCode:     "YE",
    NetworkOperator: "Yemen Mobile",
    MessageType:     "GSM-7",
}

tariff, err := engine.GetActiveTariff(ctx, req)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Tariff: %s per message\n", tariff.CostPerMessage)
```

### إبطال التخزين المؤقت

```go
// عند تحديث تعرفة، يجب إبطال التخزين المؤقت
err := engine.InvalidateCache(ctx, 1, "YE", "Yemen Mobile", "GSM-7")
if err != nil {
    log.Error("Failed to invalidate cache", zap.Error(err))
}
```

## قواعد تقسيم الرسائل

### GSM-7 (الافتراضي)
- الجزء الأول: 160 حرف
- الجزء الثاني: 153 حرف
- الجزء الثالث: 153 حرف
- الأجزاء التالية: 153 حرف لكل جزء

### Unicode (للغات غير اللاتينية)
- الجزء الأول: 70 حرف
- الجزء الثاني: 67 حرف
- الجزء الثالث: 67 حرف
- الأجزاء التالية: 67 حرف لكل جزء

## أمثلة حساب الأجزاء

### GSM-7
```
100 حرف  → 1 جزء
160 حرف  → 1 جزء
161 حرف  → 2 جزء
313 حرف  → 2 جزء
314 حرف  → 3 جزء
467 حرف  → 4 جزء
1000 حرف → 7 جزء
```

### Unicode
```
50 حرف  → 1 جزء
70 حرف  → 1 جزء
71 حرف  → 2 جزء
137 حرف → 2 جزء
138 حرف → 3 جزء
205 حرف → 4 جزء
```

## التخزين المؤقت (Caching)

### مفتاح التخزين المؤقت
```
tariff:{tenant_id}:{service_id}:{country_code}:{network_operator}:{message_type}
```

### مثال
```
tariff:1:1:YE:Yemen Mobile:GSM-7
```

### مدة التخزين
- **TTL**: ساعة واحدة
- **السبب**: التعرفات لا تتغير بشكل متكرر

### إبطال التخزين المؤقت
يتم إبطال التخزين المؤقت تلقائياً عند:
- تحديث تعرفة موجودة
- إضافة تعرفة جديدة
- حذف تعرفة

## اختيار التعرفة النشطة

### الأولوية
1. تعرفة محددة لشركة الاتصالات ونوع الرسالة
2. تعرفة محددة لشركة الاتصالات فقط
3. تعرفة محددة لنوع الرسالة فقط
4. تعرفة عامة للدولة

### الشروط
- `is_active = true`
- `effective_from <= CURRENT_DATE`
- `effective_to IS NULL OR effective_to >= CURRENT_DATE`

## معالجة الأخطاء

### أخطاء شائعة

#### 1. تعرفة غير موجودة
```go
// Error: active tariff not found for service_id=1, country=YE, operator=Yemen Mobile
```

**الحل**: تأكد من وجود تعرفة نشطة للمعاملات المحددة

#### 2. قاعدة تقسيم غير موجودة
```go
// Warning: Segmentation rule not found, using defaults
```

**الحل**: سيتم استخدام القيم الافتراضية تلقائياً

#### 3. فشل التخزين المؤقت
```go
// Warning: Failed to cache tariff
```

**الحل**: العملية ستستمر، لكن الأداء قد يتأثر

## الأداء

### التخزين المؤقت
- **Cache Hit**: ~1ms
- **Cache Miss**: ~10-50ms (استعلام قاعدة البيانات)

### حساب الأجزاء
- **O(1)**: عملية حسابية بسيطة

### التوصيات
1. استخدم التخزين المؤقت دائماً
2. قم بإبطال التخزين المؤقت عند تحديث التعرفات
3. راقب معدل Cache Hit Rate

## الاختبارات

### تشغيل الاختبارات
```bash
# جميع الاختبارات
go test -v ./internal/tariff

# اختبارات محددة
go test -v ./internal/tariff -run TestCalculateSegments

# مع التغطية
go test -v -cover ./internal/tariff
```

### أنواع الاختبارات
1. **Unit Tests**: اختبار الوظائف الفردية
2. **Integration Tests**: اختبار مع قاعدة البيانات و Redis
3. **Property Tests**: اختبار الخصائص العامة

## التكامل مع Balance Manager

```go
// 1. حساب التكلفة
result, err := tariffEngine.CalculateCost(ctx, req)
if err != nil {
    return err
}

// 2. خصم الرصيد
balanceOp := balance.BalanceOperation{
    TenantID:  req.TenantID,
    MessageID: messageID,
    Amount:    result.TotalCost,
    Type:      balance.OperationTypeDeduct,
    Reason:    "MESSAGE_SENT",
    CreatedBy: "system",
}

balanceResult, err := balanceManager.DeductBalance(ctx, balanceOp)
if err != nil {
    return err
}
```

## المتطلبات المغطاة

- **Requirement 2.5**: استخدام التعرفة النشطة الصحيحة
- **Requirement 2.6**: حساب التكلفة حسب الشبكة والدولة
- **Requirement 7.1**: تقسيم الرسائل حسب GSM-7
- **Requirement 7.2**: تقسيم الرسائل حسب Unicode
- **Requirement 7.3**: قواعد تقسيم قابلة للتخصيص
- **Requirement 7.6**: حساب عدد الأجزاء بدقة
- **Requirement 27.9**: التخزين المؤقت للتعرفات (TTL: 1 ساعة)

## الخطوات التالية

1. ✅ تطوير محرك حساب التكلفة
2. ⏳ كتابة Property Test لاستخدام التعرفة النشطة
3. ⏳ تطوير محرك حساب أجزاء الرسائل (مدمج)
4. ⏳ كتابة Property Test لحساب أجزاء الرسالة
5. ⏳ تطوير Tariff Cache Manager (مدمج)

## الملاحظات

- المحرك يدعم Multi-Tenancy بشكل كامل
- جميع العمليات thread-safe
- التخزين المؤقت اختياري (يعمل بدونه)
- معالجة الأخطاء شاملة مع Logging مفصل
