# Resource Usage Table - دليل البدء السريع

## نظرة عامة

جدول `resource_usage` يسجل استخدام الموارد لكل بوابة مخصصة كل دقيقة. يغطي المتطلبات 10.1 إلى 10.10 من نظام التعرفات والتسعير.

## التثبيت السريع

### 1. تشغيل Migration

```powershell
# Windows PowerShell
cd services/backend
.\run-resource-usage-migration.ps1
```

```bash
# Linux/Mac
cd services/backend
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f migrations/022_resource_usage_table.sql
```

### 2. التحقق من التثبيت

```powershell
# تشغيل الاختبارات
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f test_resource_usage_migration.sql
```

## الاستخدام الأساسي

### تسجيل استخدام جديد

```sql
-- تعيين tenant_id للجلسة
SET app.current_tenant = 'your-tenant-uuid';

-- إدراج سجل استخدام
INSERT INTO resource_usage (
    gateway_id,
    tenant_id,
    cpu_usage_percent,
    memory_usage_mb,
    bandwidth_usage_mbps,
    active_connections,
    db_connections_active,
    queries_per_second,
    redis_memory_usage_mb,
    redis_keys_count,
    messages_processed_last_hour,
    messages_processed_today,
    current_queue_depth,
    api_requests_last_minute,
    api_requests_last_hour,
    active_worker_threads
) VALUES (
    1,                  -- gateway_id
    'tenant-uuid',      -- tenant_id
    45.50,              -- cpu_usage_percent
    2048,               -- memory_usage_mb
    150.75,             -- bandwidth_usage_mbps
    25,                 -- active_connections
    15,                 -- db_connections_active
    120,                -- queries_per_second
    512,                -- redis_memory_usage_mb
    50000,              -- redis_keys_count
    5000,               -- messages_processed_last_hour
    45000,              -- messages_processed_today
    150,                -- current_queue_depth
    80,                 -- api_requests_last_minute
    4500,               -- api_requests_last_hour
    8                   -- active_worker_threads
);
```

### الاستعلام عن الاستخدام الحالي

```sql
-- آخر تسجيل لبوابة معينة
SELECT *
FROM resource_usage
WHERE gateway_id = 1
ORDER BY timestamp DESC
LIMIT 1;
```

### متوسط الاستخدام في الساعة الأخيرة

```sql
SELECT 
    gateway_id,
    AVG(cpu_usage_percent) as avg_cpu,
    AVG(memory_usage_mb) as avg_memory,
    MAX(current_queue_depth) as max_queue_depth
FROM resource_usage
WHERE gateway_id = 1
  AND timestamp >= NOW() - INTERVAL '1 hour'
GROUP BY gateway_id;
```

## المزايا الرئيسية

### ✅ Partitioning حسب الشهر

- تحسين الأداء للاستعلامات على البيانات الحديثة
- سهولة حذف البيانات القديمة
- تقليل حجم الـ Indexes

```sql
-- إنشاء partition جديد
SELECT create_resource_usage_partition('2025-08-01');

-- حذف partitions أقدم من 90 يوم
SELECT cleanup_old_resource_usage_partitions();
```

### ✅ Row Level Security (RLS)

- عزل تام بين الشركات
- كل شركة ترى بياناتها فقط
- تطبيق تلقائي عند تعيين `app.current_tenant`

```sql
-- تعيين tenant_id
SET app.current_tenant = 'tenant-uuid';

-- جميع الاستعلامات تُطبق RLS تلقائياً
SELECT * FROM resource_usage; -- يعرض بيانات هذه الشركة فقط
```

### ✅ Indexes محسّنة

- `idx_resource_usage_gateway_time`: للاستعلامات حسب البوابة والوقت
- `idx_resource_usage_tenant`: للاستعلامات حسب الشركة
- `idx_resource_usage_tenant_gateway`: للاستعلامات المركبة

## التكامل مع Go

### نموذج البيانات

```go
// internal/models/resource_usage.go
package models

import (
    "time"
    "github.com/google/uuid"
)

type ResourceUsage struct {
    ID                        int64     `gorm:"primaryKey"`
    GatewayID                 int       `gorm:"not null"`
    TenantID                  uuid.UUID `gorm:"type:uuid;not null"`
    Timestamp                 time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
    
    // CPU & Memory
    CPUUsagePercent           *float64  `gorm:"type:decimal(5,2)"`
    MemoryUsageMB             *int      
    
    // Network
    BandwidthUsageMbps        *float64  `gorm:"type:decimal(10,2)"`
    ActiveConnections         *int      
    
    // Database
    DBConnectionsActive       *int      
    QueriesPerSecond          *int      
    
    // Redis
    RedisMemoryUsageMB        *int      
    RedisKeysCount            *int      
    
    // Messages
    MessagesProcessedLastHour *int      
    MessagesProcessedToday    *int      
    CurrentQueueDepth         *int      
    
    // API
    APIRequestsLastMinute     *int      
    APIRequestsLastHour       *int      
    
    // Workers
    ActiveWorkerThreads       *int      
    
    CreatedAt                 time.Time `gorm:"default:CURRENT_TIMESTAMP"`
}

func (ResourceUsage) TableName() string {
    return "resource_usage"
}
```

### تسجيل الاستخدام

```go
// internal/monitoring/collector.go
package monitoring

import (
    "context"
    "time"
    "github.com/google/uuid"
    "gorm.io/gorm"
    "your-project/internal/models"
)

type MetricsCollector struct {
    db       *gorm.DB
    tenantID uuid.UUID
}

func NewMetricsCollector(db *gorm.DB, tenantID uuid.UUID) *MetricsCollector {
    return &MetricsCollector{
        db:       db,
        tenantID: tenantID,
    }
}

func (mc *MetricsCollector) LogResourceUsage(ctx context.Context, gatewayID int) error {
    // جمع المقاييس
    cpuUsage := mc.getCPUUsage()
    memoryUsage := mc.getMemoryUsage()
    bandwidthUsage := mc.getBandwidthUsage()
    
    // إنشاء السجل
    usage := &models.ResourceUsage{
        GatewayID:                 gatewayID,
        TenantID:                  mc.tenantID,
        CPUUsagePercent:           &cpuUsage,
        MemoryUsageMB:             &memoryUsage,
        BandwidthUsageMbps:        &bandwidthUsage,
        ActiveConnections:         mc.getActiveConnections(),
        DBConnectionsActive:       mc.getDBConnections(),
        QueriesPerSecond:          mc.getQueriesPerSecond(),
        RedisMemoryUsageMB:        mc.getRedisMemoryUsage(),
        RedisKeysCount:            mc.getRedisKeysCount(),
        MessagesProcessedLastHour: mc.getMessagesLastHour(),
        MessagesProcessedToday:    mc.getMessagesToday(),
        CurrentQueueDepth:         mc.getQueueDepth(),
        APIRequestsLastMinute:     mc.getAPIRequestsLastMinute(),
        APIRequestsLastHour:       mc.getAPIRequestsLastHour(),
        ActiveWorkerThreads:       mc.getActiveWorkerThreads(),
    }
    
    // تعيين tenant_id في الجلسة
    if err := mc.db.Exec("SET app.current_tenant = ?", mc.tenantID.String()).Error; err != nil {
        return err
    }
    
    // إدراج السجل
    return mc.db.WithContext(ctx).Create(usage).Error
}

// Helper methods لجمع المقاييس
func (mc *MetricsCollector) getCPUUsage() float64 {
    // تنفيذ جمع مقاييس CPU
    return 45.50
}

func (mc *MetricsCollector) getMemoryUsage() int {
    // تنفيذ جمع مقاييس Memory
    return 2048
}

// ... باقي الـ helper methods
```

### الاستعلام عن الاستخدام

```go
// internal/monitoring/usage_service.go
package monitoring

import (
    "context"
    "time"
    "github.com/google/uuid"
    "gorm.io/gorm"
    "your-project/internal/models"
)

type UsageService struct {
    db       *gorm.DB
    tenantID uuid.UUID
}

func NewUsageService(db *gorm.DB, tenantID uuid.UUID) *UsageService {
    return &UsageService{
        db:       db,
        tenantID: tenantID,
    }
}

// الحصول على آخر استخدام
func (us *UsageService) GetLatestUsage(ctx context.Context, gatewayID int) (*models.ResourceUsage, error) {
    // تعيين tenant_id
    if err := us.db.Exec("SET app.current_tenant = ?", us.tenantID.String()).Error; err != nil {
        return nil, err
    }
    
    var usage models.ResourceUsage
    err := us.db.WithContext(ctx).
        Where("gateway_id = ?", gatewayID).
        Order("timestamp DESC").
        First(&usage).Error
    
    return &usage, err
}

// الحصول على متوسط الاستخدام
func (us *UsageService) GetAverageUsage(ctx context.Context, gatewayID int, duration time.Duration) (*AverageUsage, error) {
    // تعيين tenant_id
    if err := us.db.Exec("SET app.current_tenant = ?", us.tenantID.String()).Error; err != nil {
        return nil, err
    }
    
    var result AverageUsage
    err := us.db.WithContext(ctx).
        Model(&models.ResourceUsage{}).
        Select(`
            AVG(cpu_usage_percent) as avg_cpu,
            AVG(memory_usage_mb) as avg_memory,
            AVG(bandwidth_usage_mbps) as avg_bandwidth,
            MAX(current_queue_depth) as max_queue_depth
        `).
        Where("gateway_id = ? AND timestamp >= ?", gatewayID, time.Now().Add(-duration)).
        Scan(&result).Error
    
    return &result, err
}

type AverageUsage struct {
    AvgCPU         float64
    AvgMemory      float64
    AvgBandwidth   float64
    MaxQueueDepth  int
}
```

## الصيانة

### Cron Jobs المطلوبة

```go
// internal/cron/resource_usage_maintenance.go
package cron

import (
    "context"
    "time"
    "gorm.io/gorm"
)

type ResourceUsageMaintenance struct {
    db *gorm.DB
}

func NewResourceUsageMaintenance(db *gorm.DB) *ResourceUsageMaintenance {
    return &ResourceUsageMaintenance{db: db}
}

// يومياً: إنشاء partition للشهر القادم
func (rum *ResourceUsageMaintenance) CreateNextMonthPartition(ctx context.Context) error {
    nextMonth := time.Now().AddDate(0, 2, 0)
    
    _, err := rum.db.Raw("SELECT create_resource_usage_partition(?)", nextMonth).Rows()
    return err
}

// أسبوعياً: حذف partitions أقدم من 90 يوم
func (rum *ResourceUsageMaintenance) CleanupOldPartitions(ctx context.Context) error {
    _, err := rum.db.Raw("SELECT cleanup_old_resource_usage_partitions()").Rows()
    return err
}

// شهرياً: تحليل الأداء
func (rum *ResourceUsageMaintenance) AnalyzePerformance(ctx context.Context) error {
    return rum.db.Exec("VACUUM ANALYZE resource_usage").Error
}
```

## الموارد

- **Migration**: `migrations/022_resource_usage_table.sql`
- **README**: `migrations/README_RESOURCE_USAGE.md`
- **Tests**: `test_resource_usage_migration.sql`
- **Script**: `run-resource-usage-migration.ps1`
- **Requirements**: `.kiro/specs/tariff-pricing-system/requirements.md` (المتطلب 10)

## الدعم

للمزيد من المعلومات:
1. راجع `README_RESOURCE_USAGE.md` للتوثيق الكامل
2. راجع المتطلب 10 في مستند المتطلبات
3. راجع التصميم في `design.md`

## الخلاصة

✅ **جاهز للاستخدام**:
- Partitioning حسب الشهر للأداء
- RLS للعزل التام بين الشركات
- Indexes محسّنة للاستعلامات
- Functions للصيانة التلقائية
- احتفاظ بالبيانات لمدة 90 يوم

🚀 **الخطوات التالية**:
1. تطوير Monitoring Service لتسجيل البيانات
2. تطوير Alert Manager لفحص الحدود
3. جدولة Cron Jobs للصيانة
4. تطوير Dashboard لعرض الاستخدام
