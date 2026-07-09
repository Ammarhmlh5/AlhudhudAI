# جدول Resource Quotas - الحصص الموردية

## نظرة عامة

جدول `resource_quotas` يدير الحصص الموردية المخصصة لكل بوابة مخصصة (Dedicated Gateway). يضمن هذا الجدول عدم تأثير أي شركة على أداء النظام أو الشركات الأخرى من خلال تحديد حدود واضحة للموارد.

## الهدف

- **عزل الموارد**: ضمان عدم استنزاف بوابة واحدة لموارد النظام
- **الأداء المتوقع**: تحديد حدود واضحة لكل شركة مؤسسية
- **المرونة**: دعم حصص مخصصة حسب حجم الشركة واحتياجاتها
- **المراقبة**: تمكين النظام من إرسال تنبيهات عند الاقتراب من الحدود

## البنية

### الحقول الرئيسية

#### 1. CPU & Memory (المعالج والذاكرة)
- `cpu_cores`: عدد أنوية المعالج المخصصة (vCPU)
  - مثال: `2.00` = 2 vCPU
  - مثال: `4.50` = 4.5 vCPU
- `memory_mb`: الذاكرة المخصصة بالميجابايت
  - مثال: `4096` = 4 GB
  - مثال: `8192` = 8 GB

#### 2. Network (الشبكة)
- `bandwidth_mbps`: عرض النطاق الترددي بالميجابت/ثانية
  - مثال: `100` = 100 Mbps
- `max_concurrent_connections`: الحد الأقصى للاتصالات المتزامنة
  - مثال: `1000` اتصال متزامن

#### 3. Storage (التخزين)
- `storage_gb`: مساحة التخزين المخصصة بالجيجابايت
  - مثال: `50` = 50 GB
  - مثال: `100` = 100 GB

#### 4. Database (قاعدة البيانات)
- `max_db_connections`: الحد الأقصى لاتصالات قاعدة البيانات
  - مثال: `20` اتصال
- `max_queries_per_second`: الحد الأقصى للاستعلامات في الثانية
  - مثال: `100` استعلام/ثانية

#### 5. Redis/Cache (التخزين المؤقت)
- `redis_memory_mb`: ذاكرة Redis المخصصة بالميجابايت
  - مثال: `512` = 512 MB
- `redis_max_keys`: الحد الأقصى لعدد المفاتيح في Redis
  - مثال: `100000` مفتاح

#### 6. Message Processing (معالجة الرسائل)
- `max_messages_per_hour`: الحد الأقصى للرسائل في الساعة
  - مثال: `10000` رسالة/ساعة
- `max_messages_per_day`: الحد الأقصى للرسائل في اليوم
  - مثال: `100000` رسالة/يوم
- `max_queue_depth`: الحد الأقصى لعمق قائمة الانتظار
  - مثال: `1000` رسالة في الانتظار

#### 7. API Rate Limiting (تحديد معدل API)
- `api_rate_limit_per_minute`: الحد الأقصى لطلبات API في الدقيقة
  - مثال: `100` طلب/دقيقة
- `api_rate_limit_per_hour`: الحد الأقصى لطلبات API في الساعة
  - مثال: `5000` طلب/ساعة

#### 8. Workers (العمال)
- `max_worker_threads`: الحد الأقصى لعدد Worker Threads
  - مثال: `10` threads

#### 9. السياسات
- `is_active`: هل الحصص نشطة
- `enforcement_mode`: سياسة التطبيق
  - `soft`: تحذير فقط عند التجاوز
  - `hard`: رفض الطلبات عند التجاوز

## الحصص الافتراضية

### Small Gateway (بوابة صغيرة)
مناسبة للشركات الصغيرة والمتوسطة:
```
CPU: 2.00 vCPU
Memory: 4 GB
Bandwidth: 100 Mbps
Storage: 50 GB
DB Connections: 20
Queries/sec: 100
Redis Memory: 512 MB
Redis Keys: 100,000
Messages/hour: 10,000
Messages/day: 100,000
Queue Depth: 1,000
API Rate (minute): 100
API Rate (hour): 5,000
Worker Threads: 10
```

### Medium Gateway (بوابة متوسطة)
مناسبة للشركات المتوسطة:
```
CPU: 4.00 vCPU
Memory: 8 GB
Bandwidth: 200 Mbps
Storage: 100 GB
DB Connections: 50
Queries/sec: 200
Redis Memory: 1 GB
Redis Keys: 500,000
Messages/hour: 50,000
Messages/day: 500,000
Queue Depth: 5,000
API Rate (minute): 300
API Rate (hour): 15,000
Worker Threads: 20
```

### Large Gateway (بوابة كبيرة)
مناسبة للشركات الكبيرة:
```
CPU: 8.00 vCPU
Memory: 16 GB
Bandwidth: 500 Mbps
Storage: 200 GB
DB Connections: 100
Queries/sec: 500
Redis Memory: 2 GB
Redis Keys: 1,000,000
Messages/hour: 200,000
Messages/day: 2,000,000
Queue Depth: 10,000
API Rate (minute): 1,000
API Rate (hour): 50,000
Worker Threads: 50
```

## العلاقات

```
dedicated_gateways (1) ←→ (1) resource_quotas
```

- كل بوابة مخصصة لها حصص موردية واحدة فقط
- العلاقة محمية بـ `UNIQUE(gateway_id)`
- عند حذف البوابة، تُحذف الحصص تلقائياً (`ON DELETE CASCADE`)

## الاستخدام

### 1. إنشاء حصص لبوابة جديدة

```sql
-- إنشاء حصص افتراضية (Small)
INSERT INTO resource_quotas (
    gateway_id,
    cpu_cores,
    memory_mb,
    bandwidth_mbps,
    storage_gb,
    max_db_connections,
    max_queries_per_second,
    redis_memory_mb,
    redis_max_keys,
    max_messages_per_hour,
    max_messages_per_day,
    max_queue_depth,
    api_rate_limit_per_minute,
    api_rate_limit_per_hour,
    max_worker_threads,
    enforcement_mode
) VALUES (
    1, -- gateway_id
    2.00, -- cpu_cores
    4096, -- memory_mb (4 GB)
    100, -- bandwidth_mbps
    50, -- storage_gb
    20, -- max_db_connections
    100, -- max_queries_per_second
    512, -- redis_memory_mb
    100000, -- redis_max_keys
    10000, -- max_messages_per_hour
    100000, -- max_messages_per_day
    1000, -- max_queue_depth
    100, -- api_rate_limit_per_minute
    5000, -- api_rate_limit_per_hour
    10, -- max_worker_threads
    'soft' -- enforcement_mode
);
```

### 2. تحديث الحصص

```sql
-- ترقية البوابة من Small إلى Medium
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

### 3. تغيير سياسة التطبيق

```sql
-- تغيير من Soft إلى Hard Limit
UPDATE resource_quotas
SET enforcement_mode = 'hard'
WHERE gateway_id = 1;
```

### 4. الاستعلام عن الحصص

```sql
-- عرض حصص بوابة معينة
SELECT 
    rq.*,
    dg.gateway_name,
    dg.company_name
FROM resource_quotas rq
JOIN dedicated_gateways dg ON rq.gateway_id = dg.id
WHERE dg.id = 1;
```

### 5. البحث عن البوابات القريبة من الحدود

```sql
-- البوابات التي تستخدم أكثر من 90% من حصة الرسائل اليومية
-- (يتطلب جدول resource_usage)
SELECT 
    dg.gateway_name,
    dg.company_name,
    rq.max_messages_per_day,
    ru.messages_sent_today,
    ROUND((ru.messages_sent_today::DECIMAL / rq.max_messages_per_day) * 100, 2) AS usage_percentage
FROM resource_quotas rq
JOIN dedicated_gateways dg ON rq.gateway_id = dg.id
JOIN resource_usage ru ON ru.gateway_id = dg.id
WHERE ru.messages_sent_today > (rq.max_messages_per_day * 0.9)
AND ru.date = CURRENT_DATE;
```

## Enforcement Modes (أنماط التطبيق)

### Soft Limit (حد مرن)
- يسمح بتجاوز الحدود مؤقتاً
- يرسل تنبيهات للشركة المؤسسية
- يسجل التجاوزات في السجلات
- مناسب للبوابات الجديدة أو فترات الذروة المتوقعة

### Hard Limit (حد صارم)
- يرفض الطلبات عند الوصول للحد
- يمنع التجاوز بشكل كامل
- يحمي النظام من الاستنزاف
- مناسب للبوابات ذات الاستخدام المرتفع

## التنبيهات

يجب على النظام إرسال تنبيهات عند:

1. **90% من الحد**: تحذير مبكر
   ```
   "تنبيه: بوابة {gateway_name} وصلت إلى 90% من حصة {resource_type}"
   ```

2. **100% من الحد (Soft)**: تحذير حرج
   ```
   "تحذير حرج: بوابة {gateway_name} تجاوزت حصة {resource_type}"
   ```

3. **100% من الحد (Hard)**: رفض الطلبات
   ```
   "خطأ: تم رفض الطلب - بوابة {gateway_name} وصلت للحد الأقصى من {resource_type}"
   ```

## الأمان

### 1. Constraints
- جميع القيم يجب أن تكون موجبة
- `enforcement_mode` يجب أن يكون `soft` أو `hard`
- `gateway_id` يجب أن يكون فريداً

### 2. Indexes
- `idx_resource_quotas_gateway`: للبحث السريع بـ gateway_id
- `idx_resource_quotas_active`: لتصفية الحصص النشطة
- `idx_resource_quotas_enforcement`: لتصفية حسب نمط التطبيق

### 3. Triggers
- `trigger_update_resource_quotas_updated_at`: تحديث `updated_at` تلقائياً

## التكامل مع الكود

### Go Struct

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

## Best Practices

### 1. تحديد الحصص
- ابدأ بحصص محافظة (Small) للبوابات الجديدة
- راقب الاستخدام الفعلي لمدة شهر
- قم بالترقية حسب الحاجة

### 2. Enforcement Mode
- استخدم `soft` للبوابات الجديدة
- انتقل إلى `hard` بعد فترة تجريبية
- استخدم `soft` خلال فترات الذروة المتوقعة

### 3. المراقبة
- راقب الاستخدام مقابل الحصص يومياً
- أرسل تنبيهات عند 90% من الحد
- راجع الحصص شهرياً

### 4. التوثيق
- وثّق سبب كل تغيير في الحصص
- احتفظ بسجل للترقيات والتخفيضات
- شارك التقارير مع الشركات المؤسسية

## الخطوات التالية

بعد إنشاء هذا الجدول، يجب:

1. إنشاء جدول `resource_usage` لتسجيل الاستخدام الفعلي
2. تطوير Resource Limiter في الكود
3. تطوير Monitoring Service لجمع المقاييس
4. تطوير Alert Manager لإرسال التنبيهات
5. إنشاء واجهة إدارة الحصص في Admin Panel

## المتطلبات المرتبطة

- **Requirement 9.1**: دعم تحديد حصة CPU
- **Requirement 9.2**: دعم تحديد حصة RAM
- **Requirement 9.3**: دعم تحديد حصة Bandwidth
- **Requirement 9.4**: دعم تحديد حصة Storage
- **Requirement 9.5**: دعم تحديد عدد اتصالات قاعدة البيانات
- **Requirement 9.6**: دعم تحديد حصة Redis Memory
- **Requirement 9.7**: دعم تحديد الحد الأقصى للرسائل
- **Requirement 9.8**: دعم تحديد حد API Rate Limit
- **Requirement 9.9**: دعم تحديد عدد Worker Threads

## الملفات ذات الصلة

- `021_resource_quotas_table.sql`: Migration Script
- `020_dedicated_gateways_table.sql`: جدول البوابات المخصصة
- `test_resource_quotas_migration.sql`: اختبارات SQL
- `run-resource-quotas-migration.ps1`: سكريبت التشغيل
