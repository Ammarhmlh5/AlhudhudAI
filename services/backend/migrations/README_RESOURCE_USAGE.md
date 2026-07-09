# جدول Resource Usage - دليل الاستخدام

## نظرة عامة

جدول `resource_usage` يسجل استخدام الموارد لكل بوابة مخصصة كل دقيقة. يستخدم الجدول **Partitioning حسب الشهر** لتحسين الأداء ويحتفظ بالبيانات لمدة **90 يوم** على الأقل.

## المتطلبات المغطاة

- **Requirement 10.1**: تسجيل استخدام CPU كل دقيقة
- **Requirement 10.2**: تسجيل استخدام RAM كل دقيقة
- **Requirement 10.3**: تسجيل استخدام Bandwidth كل دقيقة
- **Requirement 10.4**: تسجيل عدد الاتصالات النشطة
- **Requirement 10.5**: تسجيل عدد الاستعلامات في الثانية
- **Requirement 10.6**: تسجيل استخدام Redis Memory وعدد المفاتيح
- **Requirement 10.7**: تسجيل عدد الرسائل المعالجة في الساعة واليوم
- **Requirement 10.8**: تسجيل عمق قائمة الانتظار
- **Requirement 10.9**: تسجيل عدد طلبات API في الدقيقة والساعة
- **Requirement 10.10**: تسجيل عدد Worker Threads النشطة
- **Requirement 10.11**: الاحتفاظ بالبيانات لمدة 90 يوم

## هيكل الجدول

### الحقول الأساسية

```sql
id                          BIGSERIAL       -- معرّف فريد
gateway_id                  INTEGER         -- معرّف البوابة المخصصة
tenant_id                   UUID            -- معرّف الشركة (للعزل)
timestamp                   TIMESTAMP       -- وقت التسجيل
```

### موارد CPU والذاكرة

```sql
cpu_usage_percent           DECIMAL(5,2)    -- نسبة استخدام المعالج (0-100)
memory_usage_mb             INTEGER         -- استخدام الذاكرة بالميجابايت
```

### موارد الشبكة

```sql
bandwidth_usage_mbps        DECIMAL(10,2)   -- استخدام عرض النطاق الترددي
active_connections          INTEGER         -- عدد الاتصالات النشطة
```

### موارد قاعدة البيانات

```sql
db_connections_active       INTEGER         -- اتصالات قاعدة البيانات النشطة
queries_per_second          INTEGER         -- عدد الاستعلامات في الثانية
```

### موارد Redis

```sql
redis_memory_usage_mb       INTEGER         -- استخدام ذاكرة Redis
redis_keys_count            INTEGER         -- عدد المفاتيح في Redis
```

### معالجة الرسائل

```sql
messages_processed_last_hour INTEGER        -- رسائل الساعة الأخيرة
messages_processed_today     INTEGER        -- رسائل اليوم
current_queue_depth          INTEGER        -- عمق قائمة الانتظار
```

### طلبات API

```sql
api_requests_last_minute    INTEGER         -- طلبات الدقيقة الأخيرة
api_requests_last_hour      INTEGER         -- طلبات الساعة الأخيرة
```

### Workers

```sql
active_worker_threads       INTEGER         -- عدد Worker Threads النشطة
```

## Partitioning Strategy

### لماذا Partitioning؟

- **الأداء**: تحسين سرعة الاستعلامات على البيانات الحديثة
- **الصيانة**: سهولة حذف البيانات القديمة
- **التخزين**: تقليل حجم الـ Indexes

### كيف يعمل؟

الجدول مقسم حسب **الشهر**:

```
resource_usage (Parent Table)
├── resource_usage_2024_12 (Dec 2024)
├── resource_usage_2025_01 (Jan 2025)
├── resource_usage_2025_02 (Feb 2025)
└── ...
```

### إنشاء Partition جديد

```sql
-- إنشاء partition لشهر معين
SELECT create_resource_usage_partition('2025-07-01');
```

### حذف Partitions القديمة

```sql
-- حذف partitions أقدم من 90 يوم
SELECT cleanup_old_resource_usage_partitions();
```

## Row Level Security (RLS)

الجدول محمي بـ RLS لضمان العزل التام بين الشركات:

```sql
-- كل شركة ترى بياناتها فقط
SET app.current_tenant = 'tenant-uuid-here';
SELECT * FROM resource_usage; -- يعرض بيانات هذه الشركة فقط
```

## Indexes

### Index رئيسي

```sql
idx_resource_usage_gateway_time (gateway_id, timestamp DESC)
```

استخدام: الاستعلامات حسب البوابة والوقت

### Index للعزل

```sql
idx_resource_usage_tenant (tenant_id, timestamp DESC)
```

استخدام: الاستعلامات حسب الشركة

### Index مركب

```sql
idx_resource_usage_tenant_gateway (tenant_id, gateway_id, timestamp DESC)
```

استخدام: الاستعلامات المركبة

## أمثلة الاستخدام

### 1. تسجيل استخدام جديد

```sql
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
    1,                                      -- gateway_id
    'tenant-uuid',                          -- tenant_id
    45.50,                                  -- cpu_usage_percent
    2048,                                   -- memory_usage_mb
    150.75,                                 -- bandwidth_usage_mbps
    25,                                     -- active_connections
    15,                                     -- db_connections_active
    120,                                    -- queries_per_second
    512,                                    -- redis_memory_usage_mb
    50000,                                  -- redis_keys_count
    5000,                                   -- messages_processed_last_hour
    45000,                                  -- messages_processed_today
    150,                                    -- current_queue_depth
    80,                                     -- api_requests_last_minute
    4500,                                   -- api_requests_last_hour
    8                                       -- active_worker_threads
);
```

### 2. الاستعلام عن الاستخدام الحالي

```sql
-- آخر تسجيل لبوابة معينة
SELECT *
FROM resource_usage
WHERE gateway_id = 1
ORDER BY timestamp DESC
LIMIT 1;
```

### 3. الاستعلام عن متوسط الاستخدام في الساعة الأخيرة

```sql
SELECT 
    gateway_id,
    AVG(cpu_usage_percent) as avg_cpu,
    AVG(memory_usage_mb) as avg_memory,
    AVG(bandwidth_usage_mbps) as avg_bandwidth,
    MAX(current_queue_depth) as max_queue_depth
FROM resource_usage
WHERE gateway_id = 1
  AND timestamp >= NOW() - INTERVAL '1 hour'
GROUP BY gateway_id;
```

### 4. الاستعلام عن الاستخدام اليومي

```sql
SELECT 
    DATE(timestamp) as date,
    gateway_id,
    AVG(cpu_usage_percent) as avg_cpu,
    MAX(cpu_usage_percent) as max_cpu,
    AVG(memory_usage_mb) as avg_memory,
    MAX(memory_usage_mb) as max_memory,
    SUM(messages_processed_last_hour) as total_messages
FROM resource_usage
WHERE gateway_id = 1
  AND timestamp >= CURRENT_DATE - INTERVAL '7 days'
GROUP BY DATE(timestamp), gateway_id
ORDER BY date DESC;
```

### 5. مقارنة الاستخدام مع الحصص

```sql
SELECT 
    ru.gateway_id,
    dg.gateway_name,
    AVG(ru.cpu_usage_percent) as avg_cpu_usage,
    rq.cpu_cores * 100 as cpu_quota_percent,
    AVG(ru.memory_usage_mb) as avg_memory_usage,
    rq.memory_mb as memory_quota,
    AVG(ru.db_connections_active) as avg_db_connections,
    rq.max_db_connections as db_connections_quota
FROM resource_usage ru
JOIN dedicated_gateways dg ON ru.gateway_id = dg.id
JOIN resource_quotas rq ON ru.gateway_id = rq.gateway_id
WHERE ru.timestamp >= NOW() - INTERVAL '1 hour'
GROUP BY ru.gateway_id, dg.gateway_name, rq.cpu_cores, rq.memory_mb, rq.max_db_connections;
```

### 6. البحث عن البوابات التي تقترب من الحدود

```sql
WITH current_usage AS (
    SELECT 
        gateway_id,
        AVG(cpu_usage_percent) as avg_cpu,
        AVG(memory_usage_mb) as avg_memory,
        AVG(db_connections_active) as avg_db_connections
    FROM resource_usage
    WHERE timestamp >= NOW() - INTERVAL '5 minutes'
    GROUP BY gateway_id
)
SELECT 
    cu.gateway_id,
    dg.gateway_name,
    cu.avg_cpu,
    rq.cpu_cores * 100 as cpu_quota,
    (cu.avg_cpu / (rq.cpu_cores * 100)) * 100 as cpu_usage_percent,
    cu.avg_memory,
    rq.memory_mb as memory_quota,
    (cu.avg_memory::DECIMAL / rq.memory_mb) * 100 as memory_usage_percent
FROM current_usage cu
JOIN dedicated_gateways dg ON cu.gateway_id = dg.id
JOIN resource_quotas rq ON cu.gateway_id = rq.gateway_id
WHERE 
    (cu.avg_cpu / (rq.cpu_cores * 100)) * 100 > 90  -- CPU فوق 90%
    OR (cu.avg_memory::DECIMAL / rq.memory_mb) * 100 > 90  -- Memory فوق 90%
ORDER BY cpu_usage_percent DESC;
```

## الصيانة

### إنشاء Partitions للأشهر القادمة

يُنصح بإنشاء partitions للأشهر القادمة مسبقاً:

```sql
-- إنشاء partitions للأشهر الستة القادمة
DO $
DECLARE
    month_date DATE;
BEGIN
    FOR i IN 0..5 LOOP
        month_date := DATE_TRUNC('month', CURRENT_DATE + (i || ' months')::INTERVAL);
        PERFORM create_resource_usage_partition(month_date);
    END LOOP;
END $;
```

### جدولة الصيانة

يُنصح بجدولة المهام التالية:

1. **يومياً**: فحص وإنشاء partition للشهر القادم إذا لزم الأمر
2. **أسبوعياً**: حذف partitions أقدم من 90 يوم
3. **شهرياً**: تحليل الأداء وإعادة بناء الـ indexes إذا لزم الأمر

```sql
-- Cron Job يومي
SELECT create_resource_usage_partition(CURRENT_DATE + INTERVAL '2 months');

-- Cron Job أسبوعي
SELECT cleanup_old_resource_usage_partitions();

-- Cron Job شهري
VACUUM ANALYZE resource_usage;
```

## الأداء

### نصائح لتحسين الأداء

1. **استخدم WHERE clause مع timestamp**: دائماً حدد نطاق زمني في الاستعلامات
2. **استخدم gateway_id في WHERE**: يساعد في استخدام الـ Index الصحيح
3. **تجنب SELECT ***: اختر الحقول المطلوبة فقط
4. **استخدم LIMIT**: عند عرض البيانات الحديثة

### مثال على استعلام محسّن

```sql
-- ✅ جيد
SELECT cpu_usage_percent, memory_usage_mb, timestamp
FROM resource_usage
WHERE gateway_id = 1
  AND timestamp >= NOW() - INTERVAL '1 hour'
ORDER BY timestamp DESC
LIMIT 60;

-- ❌ سيء
SELECT *
FROM resource_usage
WHERE gateway_id = 1
ORDER BY timestamp DESC;
```

## الأمان

### Row Level Security

- كل شركة ترى بياناتها فقط
- يتم تطبيق RLS تلقائياً عند تعيين `app.current_tenant`
- لا يمكن لأي شركة الوصول لبيانات شركة أخرى

### Constraints

- جميع القيم يجب أن تكون موجبة أو NULL
- `cpu_usage_percent` يجب أن يكون بين 0 و 100
- لا يمكن إدراج بيانات لـ tenant_id مختلف عن الجلسة الحالية

## التكامل مع النظام

### Monitoring Service

سيقوم `internal/monitoring/collector.go` بتسجيل البيانات كل دقيقة:

```go
func (mc *MetricsCollector) LogResourceUsage(ctx context.Context, gatewayID int) error {
    usage := &models.ResourceUsage{
        GatewayID:                gatewayID,
        TenantID:                 mc.tenantID,
        CPUUsagePercent:          mc.getCPUUsage(),
        MemoryUsageMB:            mc.getMemoryUsage(),
        BandwidthUsageMbps:       mc.getBandwidthUsage(),
        ActiveConnections:        mc.getActiveConnections(),
        DBConnectionsActive:      mc.getDBConnections(),
        QueriesPerSecond:         mc.getQueriesPerSecond(),
        RedisMemoryUsageMB:       mc.getRedisMemoryUsage(),
        RedisKeysCount:           mc.getRedisKeysCount(),
        MessagesProcessedLastHour: mc.getMessagesLastHour(),
        MessagesProcessedToday:   mc.getMessagesToday(),
        CurrentQueueDepth:        mc.getQueueDepth(),
        APIRequestsLastMinute:    mc.getAPIRequestsLastMinute(),
        APIRequestsLastHour:      mc.getAPIRequestsLastHour(),
        ActiveWorkerThreads:      mc.getActiveWorkerThreads(),
    }
    
    return mc.db.Create(usage).Error
}
```

### Alert Manager

سيقوم `internal/monitoring/alert_manager.go` بفحص الحدود:

```go
func (am *AlertManager) CheckResourceThresholds(ctx context.Context, gatewayID int) error {
    // الحصول على آخر استخدام
    usage, err := am.getLatestUsage(gatewayID)
    if err != nil {
        return err
    }
    
    // الحصول على الحصص
    quotas, err := am.getResourceQuotas(gatewayID)
    if err != nil {
        return err
    }
    
    // فحص CPU
    if usage.CPUUsagePercent >= quotas.CPUCores * 100 * 0.9 {
        am.sendAlert("CPU usage approaching limit", gatewayID)
    }
    
    // فحص Memory
    if usage.MemoryUsageMB >= quotas.MemoryMB * 0.9 {
        am.sendAlert("Memory usage approaching limit", gatewayID)
    }
    
    // ... فحص باقي الموارد
    
    return nil
}
```

## الخلاصة

جدول `resource_usage` هو جزء أساسي من نظام مراقبة البوابات المخصصة. يوفر:

- ✅ تسجيل دقيق لجميع الموارد كل دقيقة
- ✅ أداء عالي باستخدام Partitioning
- ✅ عزل تام بين الشركات باستخدام RLS
- ✅ صيانة سهلة مع Functions تلقائية
- ✅ احتفاظ بالبيانات لمدة 90 يوم

للمزيد من المعلومات، راجع:
- `services/backend/internal/monitoring/collector.go`
- `services/backend/internal/monitoring/alert_manager.go`
- `.kiro/specs/tariff-pricing-system/requirements.md` (المتطلب 10)
