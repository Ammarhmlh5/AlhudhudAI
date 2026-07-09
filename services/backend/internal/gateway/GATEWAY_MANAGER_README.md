# Gateway Manager

## نظرة عامة

Gateway Manager هو مكون أساسي في نظام التعرفات والتسعير يوفر إدارة شاملة للبوابات المخصصة للشركات المؤسسية.

## المتطلبات المغطاة

- **Requirement 8.1-8.15**: إدارة البوابات المخصصة
- **Requirement 26.1**: الأمان والتشفير (AES-256)
- **Requirement 28.5**: Health Checks

## الميزات الرئيسية

### 1. إنشاء بوابة مخصصة (CreateGateway)

إنشاء بوابة مخصصة جديدة مع:
- ✅ Tenant Validation - التحقق من صحة الشركة
- ✅ تشفير بيانات الاعتماد باستخدام AES-256
- ✅ إنشاء Resource Quotas افتراضية
- ✅ حساب تاريخ الفوترة التالي تلقائياً
- ✅ دعم أنواع السيرفرات المختلفة (dedicated, vps, container, shared)

**مثال الاستخدام:**

```go
manager, err := gateway.NewManager(db, "32-byte-encryption-key-here!!")
if err != nil {
    log.Fatal(err)
}

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

fmt.Printf("Gateway created: %s (ID: %d)\n", gateway.GatewayName, gateway.ID)
```

### 2. تحديث إعدادات البوابة (UpdateGateway)

تحديث إعدادات البوابة مع:
- ✅ Tenant Validation
- ✅ تحديث انتقائي للحقول
- ✅ تشفير بيانات الاعتماد الجديدة
- ✅ دعم تحديث الحدود والرسوم

**مثال الاستخدام:**

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

fmt.Printf("Gateway updated: %s\n", updated.GatewayName)
```

### 3. تعليق البوابة (SuspendGateway)

تعليق البوابة مؤقتاً مع:
- ✅ Tenant Validation
- ✅ تحديث الحالة إلى "suspended"
- ✅ إرسال إشعار للشركة المؤسسية (TODO)

**مثال الاستخدام:**

```go
err := manager.SuspendGateway(ctx, gatewayID, tenantID, "Payment overdue")
if err != nil {
    log.Fatal(err)
}

fmt.Println("Gateway suspended successfully")
```

### 4. إيقاف البوابة (TerminateGateway)

إيقاف البوابة نهائياً مع:
- ✅ Tenant Validation
- ✅ تحديث الحالة إلى "terminated"
- ✅ إرسال إشعار للشركة المؤسسية (TODO)

**مثال الاستخدام:**

```go
err := manager.TerminateGateway(ctx, gatewayID, tenantID, "Contract ended")
if err != nil {
    log.Fatal(err)
}

fmt.Println("Gateway terminated successfully")
```

### 5. فحص صحة البوابة (PerformHealthCheck)

فحص صحة البوابة مع:
- ✅ Tenant Validation
- ✅ فحص الاتصال بشركة الاتصالات (TODO: تطبيق فعلي)
- ✅ قياس وقت الاستجابة
- ✅ تحديث last_health_check و health_status

**مثال الاستخدام:**

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

## نموذج البيانات

### DedicatedGateway

```go
type DedicatedGateway struct {
    ID                       int
    TenantID                 uuid.UUID
    GatewayName              string
    GatewayCode              string // فريد
    Description              string
    TelecomProvider          string
    TelecomAPIEndpoint       string
    TelecomUsername          string
    TelecomPasswordEncrypted string // مشفر بـ AES-256
    TelecomAPIKeyEncrypted   string // مشفر بـ AES-256
    ServerType               string // dedicated, vps, container, shared
    ServerIP                 string
    ServerHostname           string
    ServerSpecs              map[string]interface{}
    MaxThroughput            int
    DailyLimit               *int
    MonthlyLimit             *int
    SubscriptionType         string // monthly, yearly
    SubscriptionFee          float64
    ServerRentalFee          float64
    SetupFee                 float64
    BillingCycle             string // monthly, yearly
    NextBillingDate          *time.Time
    TelecomBalance           *float64 // للمراقبة فقط
    BalanceAlertThreshold    *float64
    LastBalanceCheck         *time.Time
    BalanceCheckEnabled      bool
    ContractStartDate        time.Time
    ContractEndDate          *time.Time
    AutoRenew                bool
    Status                   string // active, suspended, maintenance, terminated
    LastHealthCheck          *time.Time
    HealthStatus             string // healthy, degraded, unhealthy, unknown
    CreatedAt                time.Time
    UpdatedAt                time.Time
}
```

### ResourceQuota

```go
type ResourceQuota struct {
    ID                       int
    GatewayID                int
    CPUCores                 *float64
    MemoryMB                 *int
    BandwidthMbps            *int
    MaxConcurrentConnections *int
    StorageGB                *int
    MaxDBConnections         *int
    MaxQueriesPerSecond      *int
    RedisMemoryMB            *int
    RedisMaxKeys             *int
    MaxMessagesPerHour       *int
    MaxMessagesPerDay        *int
    MaxQueueDepth            *int
    APIRateLimitPerMinute    *int
    APIRateLimitPerHour      *int
    MaxWorkerThreads         *int
    IsActive                 bool
    EnforcementMode          string // soft, hard
    CreatedAt                time.Time
    UpdatedAt                time.Time
}
```

## الأمان

### تشفير بيانات الاعتماد

يستخدم Gateway Manager تشفير AES-256-GCM لحماية بيانات الاعتماد الحساسة:

- **TelecomPassword**: كلمة مرور شركة الاتصالات
- **TelecomAPIKey**: مفتاح API لشركة الاتصالات

**ملاحظات الأمان:**

1. مفتاح التشفير يجب أن يكون 32 بايت بالضبط (AES-256)
2. يتم تخزين المفتاح في متغيرات البيئة، ليس في الكود
3. يستخدم GCM mode للتشفير المصادق عليه
4. كل عملية تشفير تستخدم nonce عشوائي فريد

**مثال إعداد المفتاح:**

```bash
# في ملف .env
GATEWAY_ENCRYPTION_KEY=your-32-byte-encryption-key!!
```

```go
// في الكود
encryptionKey := os.Getenv("GATEWAY_ENCRYPTION_KEY")
manager, err := gateway.NewManager(db, encryptionKey)
```

### Tenant Isolation

جميع العمليات تتحقق من tenant_id لضمان:
- عدم الوصول لبوابات شركات أخرى
- عدم تعديل بيانات شركات أخرى
- عزل تام بين الشركات

## الحصص الموردية الافتراضية

عند إنشاء بوابة جديدة، يتم إنشاء حصص موردية افتراضية (Medium Gateway):

```go
CPUCores:                 4.0 vCPU
MemoryMB:                 8192 MB (8 GB)
BandwidthMbps:            200 Mbps
MaxConcurrentConnections: 1000
StorageGB:                100 GB
MaxDBConnections:         50
MaxQueriesPerSecond:      200
RedisMemoryMB:            1024 MB (1 GB)
RedisMaxKeys:             500,000
MaxMessagesPerHour:       50,000
MaxMessagesPerDay:        500,000
MaxQueueDepth:            5,000
APIRateLimitPerMinute:    300
APIRateLimitPerHour:      15,000
MaxWorkerThreads:         20
EnforcementMode:          "soft"
```

يمكن تخصيص هذه الحصص عند الإنشاء أو تحديثها لاحقاً.

## الاختبارات

### تشغيل الاختبارات

```bash
# تشغيل جميع اختبارات Gateway Manager
cd services/backend
go test -v ./internal/gateway/...

# تشغيل اختبار محدد
go test -v -run TestCreateGateway ./internal/gateway/...

# مع CGO enabled (مطلوب لـ SQLite)
CGO_ENABLED=1 go test -v ./internal/gateway/...
```

### تغطية الاختبارات

- ✅ TestNewManager - إنشاء Manager
- ✅ TestCreateGateway - إنشاء بوابة
- ✅ TestUpdateGateway - تحديث بوابة
- ✅ TestSuspendGateway - تعليق بوابة
- ✅ TestTerminateGateway - إيقاف بوابة
- ✅ TestPerformHealthCheck - فحص الصحة
- ✅ TestEncryptDecrypt - التشفير وفك التشفير
- ✅ TestCalculateNextBillingDate - حساب تاريخ الفوترة

## نموذج العمل المالي

**ملاحظة مهمة:** الشركة المؤسسية تدفع لشركة الاتصالات مباشرة لتكلفة الرسائل.

المنصة تتقاضى فقط:
1. **رسوم الاشتراك** (subscription_fee) - شهري أو سنوي
2. **رسوم إيجار السيرفر** (server_rental_fee) - شهري

دور المنصة:
- مراقبة رصيد الشركة لدى شركة الاتصالات (للقراءة فقط)
- إرسال تنبيهات عند انخفاض الرصيد
- **لا تعالج** أي دفعات متعلقة بتكلفة الرسائل

## التكامل مع المكونات الأخرى

### Notification Service (TODO)

عند تعليق أو إيقاف بوابة، يجب إرسال إشعار للشركة المؤسسية:

```go
// في SuspendGateway
notificationService.SendGatewaySuspendedNotification(ctx, tenantID, gatewayID, reason)

// في TerminateGateway
notificationService.SendGatewayTerminatedNotification(ctx, tenantID, gatewayID, reason)
```

### Balance Monitor (TODO)

مراقبة دورية لرصيد الشركات لدى شركات الاتصالات:

```go
// Cron job يومي
func MonitorGatewayBalances(ctx context.Context) {
    gateways := getActiveGatewaysWithBalanceCheckEnabled()
    
    for _, gateway := range gateways {
        balance := fetchBalanceFromTelecom(gateway)
        
        if balance < gateway.BalanceAlertThreshold {
            notificationService.SendLowBalanceAlert(ctx, gateway.TenantID, gateway.ID, balance)
        }
        
        updateGatewayBalance(gateway.ID, balance)
    }
}
```

### Billing Service (TODO)

فوترة تلقائية لرسوم الاشتراك والإيجار:

```go
// Cron job يومي
func ProcessGatewayBilling(ctx context.Context) {
    gateways := getGatewaysDueForBilling()
    
    for _, gateway := range gateways {
        totalFee := gateway.SubscriptionFee + gateway.ServerRentalFee
        
        err := billingService.ChargeGatewayFees(ctx, gateway.TenantID, gateway.ID, totalFee)
        if err != nil {
            // Handle billing failure
            continue
        }
        
        updateNextBillingDate(gateway.ID, gateway.BillingCycle)
    }
}
```

## المهام القادمة

- [ ] تطبيق Health Check فعلي (الاتصال بشركة الاتصالات)
- [ ] تكامل مع Notification Service
- [ ] تطوير Balance Monitor
- [ ] تطوير Billing Service للبوابات
- [ ] إضافة APIs للواجهات
- [ ] تطوير Resource Limiter
- [ ] تطوير Monitoring Service

## الملفات ذات الصلة

- `internal/gateway/manager.go` - المنطق الأساسي
- `internal/gateway/types.go` - النماذج
- `internal/gateway/manager_test.go` - الاختبارات
- `migrations/020_dedicated_gateways_table.sql` - جدول البوابات
- `migrations/021_resource_quotas_table.sql` - جدول الحصص

## المراجع

- [Requirements Document](../../../.kiro/specs/tariff-pricing-system/requirements.md)
- [Design Document](../../../.kiro/specs/tariff-pricing-system/design.md)
- [Tasks Document](../../../.kiro/specs/tariff-pricing-system/tasks.md)
