# Balance Monitor - Quick Start Guide

## نظرة عامة

Balance Monitor هو نظام مراقبة رصيد الشركات المؤسسية لدى شركات الاتصالات للبوابات المخصصة.

## نموذج العمل المالي - مهم جداً

**الشركة المؤسسية تدفع لشركة الاتصالات مباشرة لتكلفة الرسائل**

- المنصة **لا تعالج** أي دفعات متعلقة بتكلفة الرسائل
- المنصة تتقاضى فقط:
  1. رسوم الاشتراك (شهري أو سنوي)
  2. رسوم إيجار السيرفر
- دور المنصة: **مراقبة الرصيد وإرسال تنبيهات فقط**

## المتطلبات المغطاة

- ✅ **المتطلب 8.13**: مراقبة رصيد الشركة لدى شركة الاتصالات (للقراءة فقط)
- ✅ **المتطلب 8.14**: إرسال تنبيه عند انخفاض الرصيد عن balance_alert_threshold
- ✅ **المتطلب 8.15**: عدم معالجة أي دفعات متعلقة بتكلفة الرسائل
- ✅ **المتطلب 17.8**: إرسال تنبيه للشركة المؤسسية عند اقتراب البوابة من حدها

## الميزات الرئيسية

### 1. فحص الرصيد الدوري (Cron Job)

```go
// فحص الرصيد كل ساعة (افتراضياً)
monitor := monitoring.NewBalanceMonitor(db, alertManager)
monitor.SetCheckInterval(1 * time.Hour)

// بدء الفحص الدوري
go monitor.StartPeriodicBalanceCheck(context.Background())
```

### 2. فحص رصيد بوابة محددة

```go
gatewayID := 1
tenantID := uuid.MustParse("...")

err := monitor.CheckTelecomBalance(ctx, gatewayID, tenantID)
if err != nil {
    log.Printf("Error checking balance: %v", err)
}
```

### 3. تسجيل عميل شركة اتصالات

```go
// تطبيق واجهة TelecomBalanceChecker
type YemenMobileClient struct {
    apiKey string
}

func (c *YemenMobileClient) CheckBalance(ctx context.Context, creds TelecomCredentials) (float64, error) {
    // استدعاء API شركة الاتصالات
    balance, err := callYemenMobileAPI(creds)
    return balance, err
}

// تسجيل العميل
yemenMobileClient := &YemenMobileClient{apiKey: "..."}
monitor.RegisterTelecomClient("YemenMobile", yemenMobileClient)
```

## كيفية العمل

### 1. الفحص الدوري

```
كل ساعة (قابل للتخصيص):
├── جلب جميع البوابات النشطة مع balance_check_enabled = true
├── لكل بوابة:
│   ├── استدعاء API شركة الاتصالات
│   ├── تحديث telecom_balance في قاعدة البيانات
│   ├── تحديث last_balance_check
│   └── إذا الرصيد < balance_alert_threshold:
│       └── إرسال تنبيه للشركة المؤسسية
```

### 2. التنبيهات

عند انخفاض الرصيد عن العتبة المحددة:

```go
Alert {
    AlertType: "TELECOM_BALANCE",
    Severity: "WARNING",
    Message: "Telecom balance low for gateway 'Gateway Name': 500.00 (threshold: 1000.00). 
             Please top up your balance with the telecom provider.",
    MetricValue: 500.00,
    Threshold: 1000.00,
}
```

## إعداد البوابة

### 1. إنشاء بوابة مع مراقبة الرصيد

```sql
INSERT INTO dedicated_gateways (
    tenant_id,
    gateway_name,
    gateway_code,
    telecom_provider,
    telecom_api_endpoint,
    telecom_username,
    telecom_password_encrypted,
    telecom_api_key_encrypted,
    balance_alert_threshold,
    balance_check_enabled,
    subscription_type,
    subscription_fee,
    server_rental_fee,
    contract_start_date,
    status
) VALUES (
    '123e4567-e89b-12d3-a456-426614174000',
    'شركة الاتصالات الكبرى',
    'TELECOM_CORP_001',
    'YemenMobile',
    'https://api.yemenmobile.com/balance',
    'telecom_user',
    'encrypted_password_here',
    'encrypted_api_key_here',
    1000.00,  -- تنبيه عند انخفاض الرصيد عن 1000
    true,     -- تفعيل مراقبة الرصيد
    'monthly',
    500.00,   -- رسوم الاشتراك الشهري
    200.00,   -- رسوم إيجار السيرفر
    '2024-01-01',
    'active'
);
```

### 2. تعطيل مراقبة الرصيد

```sql
UPDATE dedicated_gateways
SET balance_check_enabled = false
WHERE id = 1;
```

### 3. تحديث عتبة التنبيه

```sql
UPDATE dedicated_gateways
SET balance_alert_threshold = 2000.00
WHERE id = 1;
```

## مثال كامل

```go
package main

import (
    "context"
    "database/sql"
    "log"
    "time"

    "github.com/al-hudhud/backend/internal/monitoring"
    _ "github.com/lib/pq"
)

func main() {
    // الاتصال بقاعدة البيانات
    db, err := sql.Open("postgres", "postgres://user:pass@localhost/sms_gateway")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // إنشاء Alert Manager
    logger := monitoring.NewLogger(db, nil)
    alertManager := monitoring.NewAlertManager(db, logger, nil)

    // إنشاء Balance Monitor
    monitor := monitoring.NewBalanceMonitor(db, alertManager)

    // تسجيل عملاء شركات الاتصالات
    yemenMobileClient := &YemenMobileClient{}
    monitor.RegisterTelecomClient("YemenMobile", yemenMobileClient)

    sabafonClient := &SabafonClient{}
    monitor.RegisterTelecomClient("Sabafon", sabafonClient)

    // تخصيص فترة الفحص (اختياري)
    monitor.SetCheckInterval(30 * time.Minute) // كل 30 دقيقة

    // بدء الفحص الدوري
    ctx := context.Background()
    go monitor.StartPeriodicBalanceCheck(ctx)

    // الانتظار
    select {}
}

// مثال على تطبيق عميل شركة اتصالات
type YemenMobileClient struct{}

func (c *YemenMobileClient) CheckBalance(ctx context.Context, creds monitoring.TelecomCredentials) (float64, error) {
    // استدعاء API شركة الاتصالات
    // هذا مثال فقط - يجب تطبيق الاستدعاء الفعلي
    
    // مثال: GET https://api.yemenmobile.com/balance
    // Authorization: Bearer {creds.APIKey}
    
    balance := 1500.00 // الرصيد من API
    return balance, nil
}
```

## الاختبار

### تشغيل الاختبارات

```bash
# جميع اختبارات Balance Monitor
go test -v ./internal/monitoring -run "Balance"

# اختبار محدد
go test -v ./internal/monitoring -run TestCheckTelecomBalance_Success
```

### اختبار يدوي

```go
// استخدام Mock Client للاختبار
mockClient := &monitoring.MockTelecomClient{
    MockBalance: 1500.0,
    MockError:   nil,
}

monitor.RegisterTelecomClient("TestProvider", mockClient)

// فحص الرصيد
err := monitor.CheckTelecomBalance(ctx, gatewayID, tenantID)
```

## الأمان

### 1. تشفير بيانات الاعتماد

```go
// TODO: تطبيق AES-256 للتشفير
// حالياً: placeholder - يجب تطبيق التشفير الفعلي

func encryptPassword(password string) (string, error) {
    // استخدام AES-256
    // ...
}

func decryptPassword(encrypted string) (string, error) {
    // فك التشفير
    // ...
}
```

### 2. Row Level Security (RLS)

جميع الاستعلامات تستخدم RLS للعزل بين الشركات:

```go
// تعيين tenant_id في الجلسة
db.ExecContext(ctx, "SET app.current_tenant = $1", tenantID.String())
```

## المراقبة والتسجيل

### 1. سجل الفحص

```sql
-- آخر فحص للرصيد
SELECT 
    gateway_name,
    telecom_balance,
    balance_alert_threshold,
    last_balance_check
FROM dedicated_gateways
WHERE balance_check_enabled = true
ORDER BY last_balance_check DESC;
```

### 2. سجل التنبيهات

```sql
-- التنبيهات الأخيرة
SELECT 
    alert_type,
    severity,
    message,
    metric_value,
    threshold,
    timestamp
FROM monitoring_alerts
WHERE alert_type = 'TELECOM_BALANCE'
ORDER BY timestamp DESC
LIMIT 10;
```

## الأسئلة الشائعة

### س: هل المنصة تخصم تكلفة الرسائل من رصيد الشركة؟

**لا.** الشركة المؤسسية تدفع لشركة الاتصالات مباشرة. المنصة تراقب الرصيد فقط وترسل تنبيهات.

### س: ماذا يحدث عند انخفاض الرصيد؟

يتم إرسال تنبيه للشركة المؤسسية. الشركة مسؤولة عن تعبئة الرصيد مع شركة الاتصالات.

### س: كم مرة يتم فحص الرصيد؟

افتراضياً كل ساعة، ولكن يمكن تخصيص الفترة باستخدام `SetCheckInterval()`.

### س: هل يمكن تعطيل مراقبة الرصيد؟

نعم، عن طريق تعيين `balance_check_enabled = false` في جدول `dedicated_gateways`.

### س: كيف يتم تأمين بيانات الاعتماد؟

يتم تشفير `telecom_password_encrypted` و `telecom_api_key_encrypted` باستخدام AES-256.

## الخطوات التالية

1. ✅ تطبيق Balance Monitor الأساسي
2. ⏳ تطبيق عملاء شركات الاتصالات الفعلية
3. ⏳ تطبيق تشفير AES-256 لبيانات الاعتماد
4. ⏳ إضافة جدول balance_history لتتبع التغييرات
5. ⏳ تطبيق Subscription Billing (Task 19.6)

## الدعم

للمزيد من المعلومات، راجع:
- `services/backend/internal/monitoring/balance_monitor.go`
- `services/backend/internal/monitoring/balance_monitor_test.go`
- `services/backend/migrations/020_dedicated_gateways_table.sql`
- `.kiro/specs/tariff-pricing-system/requirements.md` (Requirements 8.13-8.15, 17.8)
