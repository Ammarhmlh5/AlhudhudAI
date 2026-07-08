# Gateway Manager - دليل البدء السريع

## نظرة سريعة

Gateway Manager يوفر إدارة شاملة للبوابات المخصصة للشركات المؤسسية.

## التثبيت والإعداد

### 1. إعداد مفتاح التشفير

```bash
# في ملف .env
GATEWAY_ENCRYPTION_KEY=your-32-byte-encryption-key!!
```

⚠️ **مهم**: المفتاح يجب أن يكون 32 بايت بالضبط (AES-256)

### 2. إنشاء Manager Instance

```go
import (
    "github.com/yourusername/yourproject/internal/gateway"
    "os"
)

// الحصول على مفتاح التشفير من البيئة
encryptionKey := os.Getenv("GATEWAY_ENCRYPTION_KEY")

// إنشاء Manager
manager, err := gateway.NewManager(db, encryptionKey)
if err != nil {
    log.Fatal(err)
}
```

## الاستخدام الأساسي

### إنشاء بوابة مخصصة

```go
ctx := context.Background()

req := &gateway.CreateGatewayRequest{
    TenantID:           tenantID,
    GatewayName:        "بوابة البنك الوطني",
    GatewayCode:        "GW-BANK-001",
    Description:        "بوابة مخصصة للبنك الوطني",
    TelecomProvider:    "Yemen Mobile",
    TelecomAPIEndpoint: "https://api.yemenmobile.com",
    TelecomUsername:    "bank_user",
    TelecomPassword:    "secure_password", // سيتم تشفيره
    TelecomAPIKey:      "api_key_123",     // سيتم تشفيره
    ServerType:         "dedicated",
    MaxThroughput:      500,
    DailyLimit:         intPtr(100000),
    MonthlyLimit:       intPtr(2000000),
    SubscriptionType:   "yearly",
    SubscriptionFee:    5000.00,
    ServerRentalFee:    1000.00,
    SetupFee:           500.00,
    BillingCycle:       "yearly",
    ContractStartDate:  time.Now(),
    AutoRenew:          true,
}

gateway, err := manager.CreateGateway(ctx, req)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("✅ Gateway created: %s (ID: %d)\n", gateway.GatewayName, gateway.ID)
```

### تحديث إعدادات البوابة

```go
newFee := 6000.00
newThreshold := 5000.00

req := &gateway.UpdateGatewayRequest{
    GatewayID:             gatewayID,
    TenantID:              tenantID,
    SubscriptionFee:       &newFee,
    BalanceAlertThreshold: &newThreshold,
}

updated, err := manager.UpdateGateway(ctx, req)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("✅ Gateway updated: %s\n", updated.GatewayName)
```

### تعليق البوابة

```go
err := manager.SuspendGateway(ctx, gatewayID, tenantID, "Payment overdue")
if err != nil {
    log.Fatal(err)
}

fmt.Println("✅ Gateway suspended successfully")
```

### إيقاف البوابة

```go
err := manager.TerminateGateway(ctx, gatewayID, tenantID, "Contract ended")
if err != nil {
    log.Fatal(err)
}

fmt.Println("✅ Gateway terminated successfully")
```

### فحص صحة البوابة

```go
result, err := manager.PerformHealthCheck(ctx, gatewayID, tenantID)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Health Check Result:\n")
fmt.Printf("  Status: %s\n", result.Status)
fmt.Printf("  Healthy: %v\n", result.IsHealthy)
fmt.Printf("  Response Time: %dms\n", result.ResponseTime)
fmt.Printf("  Telecom Connection: %v\n", result.TelecomConnection)
```

## أمثلة متقدمة

### إنشاء بوابة مع حصص موردية مخصصة

```go
cpuCores := 8.0
memoryMB := 16384
maxMessages := 200000

req := &gateway.CreateGatewayRequest{
    TenantID:           tenantID,
    GatewayName:        "بوابة شركة الاتصالات",
    GatewayCode:        "GW-TELCO-001",
    TelecomProvider:    "Yemen Mobile",
    TelecomAPIEndpoint: "https://api.yemenmobile.com",
    ServerType:         "dedicated",
    MaxThroughput:      1000,
    SubscriptionType:   "yearly",
    SubscriptionFee:    10000.00,
    ServerRentalFee:    2000.00,
    BillingCycle:       "yearly",
    ContractStartDate:  time.Now(),
    
    // حصص موردية مخصصة
    ResourceQuotas: &gateway.ResourceQuotasRequest{
        CPUCores:           &cpuCores,
        MemoryMB:           &memoryMB,
        MaxMessagesPerDay:  &maxMessages,
        EnforcementMode:    "hard", // رفض الطلبات عند التجاوز
    },
}

gateway, err := manager.CreateGateway(ctx, req)
```

### تحديث بيانات الاعتماد

```go
newPassword := "new_secure_password_123"
newAPIKey := "new_api_key_456"

req := &gateway.UpdateGatewayRequest{
    GatewayID:       gatewayID,
    TenantID:        tenantID,
    TelecomPassword: &newPassword, // سيتم تشفيره
    TelecomAPIKey:   &newAPIKey,   // سيتم تشفيره
}

updated, err := manager.UpdateGateway(ctx, req)
```

## الاختبارات

### تشغيل الاختبارات

```bash
# تشغيل جميع الاختبارات
cd services/backend
go test -v ./internal/gateway/...

# تشغيل اختبار محدد
go test -v -run TestCreateGateway ./internal/gateway/...

# مع تغطية الكود
go test -v -cover ./internal/gateway/...

# مع CGO enabled (مطلوب لـ SQLite)
CGO_ENABLED=1 go test -v ./internal/gateway/...
```

### مثال اختبار

```go
func TestCreateGateway(t *testing.T) {
    db := setupTestDB(t)
    tenantID := createTestTenant(t, db)
    key := "12345678901234567890123456789012"
    manager, err := gateway.NewManager(db, key)
    require.NoError(t, err)

    req := &gateway.CreateGatewayRequest{
        TenantID:           tenantID,
        GatewayName:        "Test Gateway",
        GatewayCode:        "TEST-GW-001",
        TelecomProvider:    "Test Telecom",
        TelecomAPIEndpoint: "https://api.test.com",
        ServerType:         "shared",
        MaxThroughput:      100,
        SubscriptionType:   "monthly",
        BillingCycle:       "monthly",
        ContractStartDate:  time.Now(),
    }

    gateway, err := manager.CreateGateway(context.Background(), req)
    assert.NoError(t, err)
    assert.NotNil(t, gateway)
    assert.Equal(t, "active", gateway.Status)
}
```

## نموذج العمل المالي

⚠️ **ملاحظة مهمة**: الشركة المؤسسية تدفع لشركة الاتصالات مباشرة لتكلفة الرسائل.

المنصة تتقاضى فقط:
1. **رسوم الاشتراك** (subscription_fee) - شهري أو سنوي
2. **رسوم إيجار السيرفر** (server_rental_fee) - شهري

دور المنصة:
- مراقبة رصيد الشركة لدى شركة الاتصالات (للقراءة فقط)
- إرسال تنبيهات عند انخفاض الرصيد
- **لا تعالج** أي دفعات متعلقة بتكلفة الرسائل

## الأمان

### تشفير بيانات الاعتماد

- ✅ AES-256-GCM للتشفير المصادق عليه
- ✅ Nonce عشوائي فريد لكل عملية تشفير
- ✅ المفتاح يُخزن في متغيرات البيئة فقط

### Tenant Isolation

- ✅ جميع العمليات تتحقق من tenant_id
- ✅ الاستعلامات تتضمن tenant_id
- ✅ رسائل خطأ واضحة عند محاولة الوصول لبوابات شركات أخرى

## الحصص الموردية الافتراضية

```
Small Gateway:
- CPU: 2.0 vCPU
- Memory: 4 GB
- Messages/Day: 100,000

Medium Gateway (افتراضي):
- CPU: 4.0 vCPU
- Memory: 8 GB
- Messages/Day: 500,000

Large Gateway:
- CPU: 8.0 vCPU
- Memory: 16 GB
- Messages/Day: 2,000,000
```

## الأخطاء الشائعة

### 1. مفتاح التشفير غير صحيح

```
Error: encryption key must be 32 bytes for AES-256
```

**الحل**: تأكد من أن المفتاح 32 بايت بالضبط

### 2. Tenant غير موجود

```
Error: tenant validation failed: tenant not found
```

**الحل**: تأكد من أن tenant_id صحيح وموجود في قاعدة البيانات

### 3. Gateway Code مكرر

```
Error: gateway code already exists
```

**الحل**: استخدم gateway_code فريد

### 4. Cross-Tenant Access

```
Error: gateway not found or access denied
```

**الحل**: تأكد من أن tenant_id المستخدم يملك البوابة

## Helper Functions

```go
// للتحويل إلى pointer
func intPtr(i int) *int {
    return &i
}

func floatPtr(f float64) *float64 {
    return &f
}

func stringPtr(s string) *string {
    return &s
}

func timePtr(t time.Time) *time.Time {
    return &t
}
```

## المراجع

- [README الكامل](internal/gateway/GATEWAY_MANAGER_README.md)
- [ملخص المهمة](.kiro/specs/tariff-pricing-system/TASK_17_GATEWAY_MANAGER_SUMMARY.md)
- [Requirements](../.kiro/specs/tariff-pricing-system/requirements.md)
- [Design](../.kiro/specs/tariff-pricing-system/design.md)

## الدعم

للمزيد من المعلومات أو المساعدة:
- راجع التوثيق الكامل في `internal/gateway/GATEWAY_MANAGER_README.md`
- راجع الاختبارات في `internal/gateway/manager_test.go`
- راجع ملخص المهمة في `.kiro/specs/tariff-pricing-system/TASK_17_GATEWAY_MANAGER_SUMMARY.md`
