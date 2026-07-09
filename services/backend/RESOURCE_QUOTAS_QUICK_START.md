# Resource Quotas - دليل البدء السريع

## نظرة عامة

جدول `resource_quotas` يدير الحصص الموردية للبوابات المخصصة. هذا الدليل يساعدك على البدء بسرعة.

## التشغيل السريع

### 1. تشغيل Migration

```powershell
# Windows PowerShell
.\run-resource-quotas-migration.ps1
```

```bash
# Linux/Mac
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f migrations/021_resource_quotas_table.sql
```

### 2. تشغيل الاختبارات

```powershell
# Windows PowerShell
$env:PGPASSWORD = $env:DB_PASSWORD
psql -h $env:DB_HOST -p $env:DB_PORT -U $env:DB_USER -d $env:DB_NAME -f test_resource_quotas_migration.sql
```

## الحصص الافتراضية

### Small Gateway
```sql
INSERT INTO resource_quotas (
    gateway_id, cpu_cores, memory_mb, bandwidth_mbps,
    storage_gb, max_db_connections, max_queries_per_second,
    redis_memory_mb, redis_max_keys, max_messages_per_hour,
    max_messages_per_day, max_queue_depth,
    api_rate_limit_per_minute, api_rate_limit_per_hour,
    max_worker_threads, enforcement_mode
) VALUES (
    1, -- gateway_id
    2.00, 4096, 100, 50, 20, 100, 512, 100000,
    10000, 100000, 1000, 100, 5000, 10, 'soft'
);
```

### Medium Gateway
```sql
INSERT INTO resource_quotas (
    gateway_id, cpu_cores, memory_mb, bandwidth_mbps,
    storage_gb, max_db_connections, max_queries_per_second,
    redis_memory_mb, redis_max_keys, max_messages_per_hour,
    max_messages_per_day, max_queue_depth,
    api_rate_limit_per_minute, api_rate_limit_per_hour,
    max_worker_threads, enforcement_mode
) VALUES (
    2, -- gateway_id
    4.00, 8192, 200, 100, 50, 200, 1024, 500000,
    50000, 500000, 5000, 300, 15000, 20, 'soft'
);
```

### Large Gateway
```sql
INSERT INTO resource_quotas (
    gateway_id, cpu_cores, memory_mb, bandwidth_mbps,
    storage_gb, max_db_connections, max_queries_per_second,
    redis_memory_mb, redis_max_keys, max_messages_per_hour,
    max_messages_per_day, max_queue_depth,
    api_rate_limit_per_minute, api_rate_limit_per_hour,
    max_worker_threads, enforcement_mode
) VALUES (
    3, -- gateway_id
    8.00, 16384, 500, 200, 100, 500, 2048, 1000000,
    200000, 2000000, 10000, 1000, 50000, 50, 'hard'
);
```

## الاستعلامات الشائعة

### 1. عرض حصص بوابة معينة
```sql
SELECT 
    rq.*,
    dg.gateway_name,
    dg.company_name
FROM resource_quotas rq
JOIN dedicated_gateways dg ON rq.gateway_id = dg.id
WHERE dg.id = 1;
```

### 2. عرض جميع البوابات مع حصصها
```sql
SELECT 
    dg.gateway_name,
    dg.company_name,
    rq.cpu_cores,
    rq.memory_mb,
    rq.bandwidth_mbps,
    rq.max_messages_per_day,
    rq.enforcement_mode
FROM resource_quotas rq
JOIN dedicated_gateways dg ON rq.gateway_id = dg.id
ORDER BY rq.cpu_cores DESC;
```

### 3. ترقية البوابة من Small إلى Medium
```sql
UPDATE resource_quotas
SET 
    cpu_cores = 4.00,
    memory_mb = 8192,
    bandwidth_mbps = 200,
    storage_gb = 100,
    max_db_connections = 50,
    max_queries_per_second = 200,
    redis_memory_mb = 1024,
    redis_max_keys = 500000,
    max_messages_per_hour = 50000,
    max_messages_per_day = 500000,
    max_queue_depth = 5000,
    api_rate_limit_per_minute = 300,
    api_rate_limit_per_hour = 15000,
    max_worker_threads = 20
WHERE gateway_id = 1;
```

### 4. تغيير Enforcement Mode
```sql
-- من Soft إلى Hard
UPDATE resource_quotas
SET enforcement_mode = 'hard'
WHERE gateway_id = 1;

-- من Hard إلى Soft
UPDATE resource_quotas
SET enforcement_mode = 'soft'
WHERE gateway_id = 1;
```

### 5. البوابات مع Soft Limit
```sql
SELECT 
    dg.gateway_name,
    dg.company_name,
    rq.enforcement_mode
FROM resource_quotas rq
JOIN dedicated_gateways dg ON rq.gateway_id = dg.id
WHERE rq.enforcement_mode = 'soft';
```

### 6. البوابات مع Hard Limit
```sql
SELECT 
    dg.gateway_name,
    dg.company_name,
    rq.enforcement_mode
FROM resource_quotas rq
JOIN dedicated_gateways dg ON rq.gateway_id = dg.id
WHERE rq.enforcement_mode = 'hard';
```

## Go Integration

### Struct Definition
```go
type ResourceQuota struct {
    ID                      int       `json:"id" db:"id"`
    GatewayID               int       `json:"gateway_id" db:"gateway_id"`
    
    // CPU & Memory
    CPUCores                *float64  `json:"cpu_cores" db:"cpu_cores"`
    MemoryMB                *int      `json:"memory_mb" db:"memory_mb"`
    
    // Network
    BandwidthMbps           *int      `json:"bandwidth_mbps" db:"bandwidth_mbps"`
    MaxConcurrentConns      *int      `json:"max_concurrent_connections" db:"max_concurrent_connections"`
    
    // Storage
    StorageGB               *int      `json:"storage_gb" db:"storage_gb"`
    
    // Database
    MaxDBConnections        *int      `json:"max_db_connections" db:"max_db_connections"`
    MaxQueriesPerSecond     *int      `json:"max_queries_per_second" db:"max_queries_per_second"`
    
    // Redis
    RedisMemoryMB           *int      `json:"redis_memory_mb" db:"redis_memory_mb"`
    RedisMaxKeys            *int      `json:"redis_max_keys" db:"redis_max_keys"`
    
    // Messages
    MaxMessagesPerHour      *int      `json:"max_messages_per_hour" db:"max_messages_per_hour"`
    MaxMessagesPerDay       *int      `json:"max_messages_per_day" db:"max_messages_per_day"`
    MaxQueueDepth           *int      `json:"max_queue_depth" db:"max_queue_depth"`
    
    // API
    APIRateLimitPerMinute   *int      `json:"api_rate_limit_per_minute" db:"api_rate_limit_per_minute"`
    APIRateLimitPerHour     *int      `json:"api_rate_limit_per_hour" db:"api_rate_limit_per_hour"`
    
    // Workers
    MaxWorkerThreads        *int      `json:"max_worker_threads" db:"max_worker_threads"`
    
    // Status
    IsActive                bool      `json:"is_active" db:"is_active"`
    EnforcementMode         string    `json:"enforcement_mode" db:"enforcement_mode"`
    
    CreatedAt               time.Time `json:"created_at" db:"created_at"`
    UpdatedAt               time.Time `json:"updated_at" db:"updated_at"`
}
```

### Repository Methods
```go
// GetByGatewayID retrieves quotas for a specific gateway
func (r *ResourceQuotaRepository) GetByGatewayID(ctx context.Context, gatewayID int) (*ResourceQuota, error) {
    var quota ResourceQuota
    
    query := `
        SELECT * FROM resource_quotas
        WHERE gateway_id = $1
    `
    
    err := r.db.GetContext(ctx, &quota, query, gatewayID)
    if err != nil {
        return nil, err
    }
    
    return &quota, nil
}

// Create creates new resource quotas
func (r *ResourceQuotaRepository) Create(ctx context.Context, quota *ResourceQuota) error {
    query := `
        INSERT INTO resource_quotas (
            gateway_id, cpu_cores, memory_mb, bandwidth_mbps,
            storage_gb, max_db_connections, max_queries_per_second,
            redis_memory_mb, redis_max_keys, max_messages_per_hour,
            max_messages_per_day, max_queue_depth,
            api_rate_limit_per_minute, api_rate_limit_per_hour,
            max_worker_threads, enforcement_mode
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
            $11, $12, $13, $14, $15, $16
        )
        RETURNING id, created_at, updated_at
    `
    
    return r.db.QueryRowContext(
        ctx, query,
        quota.GatewayID, quota.CPUCores, quota.MemoryMB, quota.BandwidthMbps,
        quota.StorageGB, quota.MaxDBConnections, quota.MaxQueriesPerSecond,
        quota.RedisMemoryMB, quota.RedisMaxKeys, quota.MaxMessagesPerHour,
        quota.MaxMessagesPerDay, quota.MaxQueueDepth,
        quota.APIRateLimitPerMinute, quota.APIRateLimitPerHour,
        quota.MaxWorkerThreads, quota.EnforcementMode,
    ).Scan(&quota.ID, &quota.CreatedAt, &quota.UpdatedAt)
}

// Update updates existing quotas
func (r *ResourceQuotaRepository) Update(ctx context.Context, quota *ResourceQuota) error {
    query := `
        UPDATE resource_quotas
        SET cpu_cores = $2,
            memory_mb = $3,
            bandwidth_mbps = $4,
            storage_gb = $5,
            max_db_connections = $6,
            max_queries_per_second = $7,
            redis_memory_mb = $8,
            redis_max_keys = $9,
            max_messages_per_hour = $10,
            max_messages_per_day = $11,
            max_queue_depth = $12,
            api_rate_limit_per_minute = $13,
            api_rate_limit_per_hour = $14,
            max_worker_threads = $15,
            enforcement_mode = $16,
            is_active = $17
        WHERE gateway_id = $1
    `
    
    _, err := r.db.ExecContext(
        ctx, query,
        quota.GatewayID, quota.CPUCores, quota.MemoryMB, quota.BandwidthMbps,
        quota.StorageGB, quota.MaxDBConnections, quota.MaxQueriesPerSecond,
        quota.RedisMemoryMB, quota.RedisMaxKeys, quota.MaxMessagesPerHour,
        quota.MaxMessagesPerDay, quota.MaxQueueDepth,
        quota.APIRateLimitPerMinute, quota.APIRateLimitPerHour,
        quota.MaxWorkerThreads, quota.EnforcementMode, quota.IsActive,
    )
    
    return err
}
```

## Enforcement Modes

### Soft Limit
- يسمح بتجاوز الحدود مؤقتاً
- يرسل تنبيهات فقط
- مناسب للبوابات الجديدة

### Hard Limit
- يرفض الطلبات عند الوصول للحد
- يحمي النظام من الاستنزاف
- مناسب للبوابات ذات الاستخدام المرتفع

## التنبيهات

يجب إرسال تنبيهات عند:
- 90% من الحد: تحذير مبكر
- 100% من الحد (Soft): تحذير حرج
- 100% من الحد (Hard): رفض الطلبات

## الخطوات التالية

1. ✅ **تم**: جدول `resource_quotas`
2. ⏳ **التالي**: جدول `resource_usage` (المهمة 16.3)
3. ⏳ تطوير Resource Limiter
4. ⏳ تطوير Monitoring Service
5. ⏳ تطوير Alert Manager

## الملفات ذات الصلة

- `migrations/021_resource_quotas_table.sql`: Migration Script
- `migrations/README_RESOURCE_QUOTAS.md`: التوثيق الكامل
- `test_resource_quotas_migration.sql`: اختبارات SQL
- `run-resource-quotas-migration.ps1`: سكريبت التشغيل

## المتطلبات المرتبطة

- **Requirement 9.1**: دعم تحديد حصة CPU ✓
- **Requirement 9.2**: دعم تحديد حصة RAM ✓
- **Requirement 9.3**: دعم تحديد حصة Bandwidth ✓
- **Requirement 9.4**: دعم تحديد حصة Storage ✓
- **Requirement 9.5**: دعم تحديد عدد اتصالات قاعدة البيانات ✓
- **Requirement 9.6**: دعم تحديد حصة Redis Memory ✓
- **Requirement 9.7**: دعم تحديد الحد الأقصى للرسائل ✓
- **Requirement 9.8**: دعم تحديد حد API Rate Limit ✓
- **Requirement 9.9**: دعم تحديد عدد Worker Threads ✓

## الدعم

للمزيد من المعلومات:
- راجع `migrations/README_RESOURCE_QUOTAS.md`
- راجع `.kiro/specs/tariff-pricing-system/design.md`
- راجع `.kiro/specs/tariff-pricing-system/requirements.md`
