# Alert Manager - دليل البدء السريع

## نظرة عامة

Alert Manager هو نظام مراقبة وتنبيه تلقائي للبوابات المخصصة. يفحص استخدام الموارد مقابل الحصص المحددة ويرسل تنبيهات عند الاقتراب من الحدود أو تجاوزها.

## المتطلبات المحققة

- ✅ **المتطلب 9.10**: إرسال تنبيه عند اقتراب البوابة من حدها الموردي
- ✅ **المتطلب 17.8**: إرسال تنبيه للشركة المؤسسية عند اقتراب البوابة من حدها الموردي
- ✅ **المتطلب 24.3**: إرسال تنبيه عند تجاوز البوابة 90% من حصتها الموردية

## المزايا الرئيسية

### 1. فحص العتبات التلقائي
- فحص دوري لاستخدام الموارد (كل دقيقة)
- مقارنة مع الحصص المحددة لكل بوابة
- دعم عتبات مخصصة لكل مقياس

### 2. مستويات التنبيه
- **WARNING (تحذير)**: عند الوصول لـ 90% من الحد
- **CRITICAL (حرج)**: عند الوصول لـ 100% أو تجاوز الحد

### 3. أنواع المقاييس المراقبة
- **CPU**: استخدام المعالج
- **MEMORY**: استخدام الذاكرة
- **BANDWIDTH**: استخدام النطاق الترددي
- **DATABASE**: اتصالات قاعدة البيانات
- **REDIS**: استخدام ذاكرة Redis

## البنية المعمارية

```
┌─────────────────────────────────────────────────────────┐
│                    Alert Manager                         │
│                                                          │
│  ┌────────────────────────────────────────────────────┐ │
│  │         CheckThresholds (كل دقيقة)                 │ │
│  │                                                     │ │
│  │  1. Get Active Gateways                            │ │
│  │  2. Get Recent Usage (من Logger)                  │ │
│  │  3. Get Resource Quotas                            │ │
│  │  4. Compare Usage vs Quotas                        │ │
│  │  5. Send Alerts if Threshold Exceeded              │ │
│  └────────────────────────────────────────────────────┘ │
│                                                          │
│  ┌────────────────────────────────────────────────────┐ │
│  │              Threshold Checkers                     │ │
│  │                                                     │ │
│  │  • checkCPUThreshold                               │ │
│  │  • checkMemoryThreshold                            │ │
│  │  • checkBandwidthThreshold                         │ │
│  │  • checkDatabaseThreshold                          │ │
│  │  • checkRedisThreshold                             │ │
│  └────────────────────────────────────────────────────┘ │
│                                                          │
│  ┌────────────────────────────────────────────────────┐ │
│  │              Alert Delivery                         │ │
│  │                                                     │ │
│  │  1. Log Alert to Database                          │ │
│  │  2. Send via Notification Service                  │ │
│  │     - Email                                         │ │
│  │     - SMS                                           │ │
│  │     - In-App                                        │ │
│  └────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

## الاستخدام

### 1. إنشاء Alert Manager

```go
import (
    "database/sql"
    "github.com/yourusername/sms-gateway/internal/monitoring"
    "github.com/yourusername/sms-gateway/internal/notifications"
)

// إنشاء المكونات المطلوبة
db := getDatabase()
redis := getRedisClient()

collector := monitoring.NewCollector(db, redis)
logger := monitoring.NewLogger(db, collector)
notificationSvc := notifications.NewService(zapLogger)

// إنشاء Alert Manager
alertManager := monitoring.NewAlertManager(db, logger, notificationSvc)
```

### 2. بدء الفحص الدوري

```go
ctx := context.Background()

// فحص كل دقيقة
go alertManager.StartPeriodicChecking(ctx, 1*time.Minute)
```

### 3. فحص يدوي

```go
// فحص جميع البوابات النشطة
err := alertManager.CheckThresholds(ctx)
if err != nil {
    log.Printf("Error checking thresholds: %v", err)
}
```

### 4. إرسال تنبيه مخصص

```go
alert := &monitoring.Alert{
    ID:          uuid.New(),
    GatewayID:   gatewayID,
    TenantID:    tenantID,
    AlertType:   "CPU",
    Severity:    "WARNING",
    Message:     "CPU usage warning: 85.00%",
    MetricValue: 85.0,
    Threshold:   80.0,
    Timestamp:   time.Now(),
    Resolved:    false,
}

err := alertManager.SendAlert(ctx, alert)
if err != nil {
    log.Printf("Error sending alert: %v", err)
}
```

## العتبات الافتراضية

```go
defaultThreshold := &AlertThreshold{
    CPUWarning:      80.0,  // 80% تحذير
    CPUCritical:     95.0,  // 95% حرج
    MemoryWarning:   80.0,  // 80% تحذير
    MemoryCritical:  95.0,  // 95% حرج
    BandwidthLimit:  1000.0, // 1000 KB/s
    ConnectionLimit: 90,    // 90% من الحد الأقصى
}
```

## أمثلة التنبيهات

### تنبيه CPU - تحذير

```
Alert Type: CPU
Severity: WARNING
Message: CPU usage warning: 85.00% (threshold: 80.00%)
Metric Value: 85.0
Threshold: 80.0
```

### تنبيه Memory - حرج

```
Alert Type: MEMORY
Severity: CRITICAL
Message: Memory usage critical: 950 MB / 1000 MB (95.00%, threshold: 95.00%)
Metric Value: 95.0
Threshold: 95.0
```

### تنبيه Bandwidth - تحذير

```
Alert Type: BANDWIDTH
Severity: WARNING
Message: Bandwidth usage warning: 1200.00 KB/s / 1280 KB/s (93.75%)
Metric Value: 93.75
Threshold: 90.0
```

### تنبيه Database - حرج

```
Alert Type: DATABASE
Severity: CRITICAL
Message: Database connections critical: 50 / 50 (100.00%)
Metric Value: 100.0
Threshold: 100.0
```

### تنبيه Redis - تحذير

```
Alert Type: REDIS
Severity: WARNING
Message: Redis memory warning: 450 MB / 500 MB (90.00%)
Metric Value: 90.0
Threshold: 90.0
```

## التكامل مع Notification Service

Alert Manager يتكامل مع Notification Service لإرسال التنبيهات:

```go
type NotificationService interface {
    SendAlert(ctx context.Context, alert *Alert) error
}
```

يمكنك تطبيق هذه الواجهة لإرسال التنبيهات عبر:
- البريد الإلكتروني
- SMS
- Slack
- Webhook
- In-App Notifications

## جدول monitoring_alerts

يتم تسجيل جميع التنبيهات في جدول `monitoring_alerts`:

```sql
CREATE TABLE monitoring_alerts (
    id UUID PRIMARY KEY,
    gateway_id INTEGER NOT NULL,
    tenant_id UUID NOT NULL,
    alert_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    message TEXT NOT NULL,
    metric_value DECIMAL(10, 2),
    threshold DECIMAL(10, 2),
    timestamp TIMESTAMP NOT NULL,
    resolved BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## الاستعلامات المفيدة

### عرض التنبيهات النشطة

```sql
SELECT 
    a.id,
    a.gateway_id,
    dg.name as gateway_name,
    a.alert_type,
    a.severity,
    a.message,
    a.timestamp
FROM monitoring_alerts a
JOIN dedicated_gateways dg ON a.gateway_id = dg.id
WHERE a.resolved = FALSE
  AND a.tenant_id = 'your-tenant-id'
ORDER BY a.timestamp DESC;
```

### عرض التنبيهات الحرجة

```sql
SELECT 
    a.gateway_id,
    dg.name as gateway_name,
    a.alert_type,
    a.message,
    a.timestamp
FROM monitoring_alerts a
JOIN dedicated_gateways dg ON a.gateway_id = dg.id
WHERE a.severity = 'CRITICAL'
  AND a.resolved = FALSE
  AND a.tenant_id = 'your-tenant-id'
ORDER BY a.timestamp DESC;
```

### إحصائيات التنبيهات

```sql
SELECT 
    alert_type,
    severity,
    COUNT(*) as count
FROM monitoring_alerts
WHERE tenant_id = 'your-tenant-id'
  AND timestamp >= NOW() - INTERVAL '24 hours'
GROUP BY alert_type, severity
ORDER BY count DESC;
```

## أفضل الممارسات

### 1. تخصيص العتبات
قم بتخصيص العتبات حسب احتياجات كل بوابة:

```go
// في المستقبل، يمكن تخزين العتبات المخصصة في جدول gateway_alert_thresholds
customThreshold := &AlertThreshold{
    CPUWarning:      70.0,  // عتبة أقل للبوابات الحساسة
    CPUCritical:     90.0,
    MemoryWarning:   75.0,
    MemoryCritical:  90.0,
}
```

### 2. تجنب التنبيهات المتكررة
استخدم آلية لتجنب إرسال نفس التنبيه بشكل متكرر:

```go
// تحقق من آخر تنبيه مشابه
lastAlert := getLastAlert(ctx, gatewayID, alertType)
if lastAlert != nil && time.Since(lastAlert.Timestamp) < 5*time.Minute {
    // لا ترسل تنبيه جديد إذا كان آخر تنبيه قبل أقل من 5 دقائق
    return nil
}
```

### 3. حل التنبيهات تلقائياً
عند عودة الاستخدام لمستوى طبيعي، قم بحل التنبيهات:

```go
if cpuUsage < threshold.CPUWarning {
    // حل جميع تنبيهات CPU النشطة
    resolveAlerts(ctx, gatewayID, "CPU")
}
```

### 4. مراقبة أداء Alert Manager
راقب أداء Alert Manager نفسه:

```go
start := time.Now()
err := alertManager.CheckThresholds(ctx)
duration := time.Since(start)

if duration > 10*time.Second {
    log.Printf("WARNING: Alert checking took too long: %v", duration)
}
```

## الاختبار

### تشغيل الاختبارات

```bash
# جميع الاختبارات
go test ./internal/monitoring/...

# اختبارات Alert Manager فقط
go test ./internal/monitoring/alert_manager_test.go -v

# مع تغطية الكود
go test ./internal/monitoring/alert_manager_test.go -cover
```

### اختبار يدوي

```bash
# إنشاء بوابة مخصصة للاختبار
psql -d sms_gateway -c "
INSERT INTO dedicated_gateways (tenant_id, name, status) 
VALUES ('your-tenant-id', 'Test Gateway', 'active');
"

# إنشاء حصص موردية
psql -d sms_gateway -c "
INSERT INTO resource_quotas (gateway_id, tenant_id, cpu_quota_cores, memory_quota_mb)
VALUES (1, 'your-tenant-id', 2.0, 1000);
"

# تشغيل Alert Manager
go run cmd/alert-manager/main.go
```

## التكامل مع المهام الأخرى

### المهمة 19.1: Metrics Collector
Alert Manager يستخدم Metrics Collector لجمع المقاييس:

```go
metrics, err := collector.CollectForGateway(ctx, gatewayID, tenantID)
```

### المهمة 19.2: Usage Logger
Alert Manager يستخدم Usage Logger للحصول على البيانات المسجلة:

```go
records, err := logger.GetRecentUsage(ctx, gatewayID, tenantID, 1)
```

### المهمة 13: Notification Service
Alert Manager يستخدم Notification Service لإرسال التنبيهات:

```go
err := notificationSvc.SendAlert(ctx, alert)
```

## الخطوات التالية

1. ✅ **المهمة 19.3**: Alert Manager (مكتمل)
2. ⏳ **المهمة 19.4**: Statistics Aggregator
3. ⏳ **المهمة 19.5**: Balance Monitor للبوابات المخصصة
4. ⏳ **المهمة 19.6**: Subscription Billing للبوابات المخصصة

## الدعم والمساعدة

للمزيد من المعلومات:
- راجع `internal/monitoring/README.md`
- راجع `internal/monitoring/types.go` للأنواع المستخدمة
- راجع `internal/monitoring/alert_manager_test.go` لأمثلة الاستخدام

## الملاحظات المهمة

1. **Tenant Isolation**: جميع التنبيهات معزولة بـ tenant_id
2. **Row Level Security**: يتم تطبيق RLS على جدول monitoring_alerts
3. **Performance**: Alert Manager مصمم للعمل بكفاءة مع آلاف البوابات
4. **Scalability**: يمكن توزيع Alert Manager على عدة خوادم
5. **Reliability**: يتم تسجيل جميع التنبيهات في قاعدة البيانات قبل الإرسال

## الخلاصة

Alert Manager هو نظام مراقبة وتنبيه شامل يضمن:
- ✅ مراقبة مستمرة لاستخدام الموارد
- ✅ تنبيهات فورية عند الاقتراب من الحدود
- ✅ عزل تام بين الشركات (Multi-Tenancy)
- ✅ قابلية للتوسع والتخصيص
- ✅ تكامل سلس مع باقي النظام
