# Usage Logger - دليل البدء السريع

## نظرة عامة

**Usage Logger** هو مكون يسجل استخدام الموارد للبوابات المخصصة في جدول `resource_usage` كل دقيقة. يعمل مع **Metrics Collector** لجمع المقاييس وتسجيلها في قاعدة البيانات.

### المتطلبات المغطاة

- **10.1**: تسجيل استخدام CPU كل دقيقة
- **10.2**: تسجيل استخدام RAM كل دقيقة
- **10.3**: تسجيل استخدام Bandwidth كل دقيقة

### المكونات

```
internal/monitoring/
├── collector.go          # جمع المقاييس من النظام
├── logger.go            # تسجيل المقاييس في قاعدة البيانات ✨ جديد
├── types.go             # أنواع البيانات
├── collector_test.go    # اختبارات Collector
└── logger_test.go       # اختبارات Logger ✨ جديد
```

---

## التثبيت السريع

### 1. التحقق من وجود جدول resource_usage

```bash
# تأكد من تشغيل Migration 022
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT COUNT(*) FROM resource_usage;"
```

### 2. استخدام Usage Logger في التطبيق

```go
package main

import (
    "context"
    "database/sql"
    "log"
    
    "github.com/al-hudhud/backend/internal/monitoring"
    "github.com/redis/go-redis/v9"
)

func main() {
    // إعداد قاعدة البيانات
    db, err := sql.Open("postgres", "your-connection-string")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // إعداد Redis
    redisClient := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    
    // إنشاء Collector و Logger
    collector := monitoring.NewCollector(db, redisClient)
    logger := monitoring.NewLogger(db, collector)
    
    // بدء التسجيل الدوري (كل دقيقة)
    ctx := context.Background()
    go logger.StartPeriodicLogging(ctx)
    
    // أو تسجيل مرة واحدة
    if err := logger.LogResourceUsage(ctx); err != nil {
        log.Printf("Error logging resource usage: %v", err)
    }
    
    // الانتظار...
    select {}
}
```

---

## الاستخدام

### 1. تسجيل دوري (كل دقيقة)

```go
// بدء التسجيل الدوري
ctx := context.Background()
go logger.StartPeriodicLogging(ctx)

// سيتم تسجيل المقاييس تلقائياً كل دقيقة
// للإيقاف، قم بإلغاء الـ context
```

### 2. تسجيل يدوي لجميع البوابات

```go
ctx := context.Background()

// تسجيل المقاييس لجميع البوابات النشطة
err := logger.LogResourceUsage(ctx)
if err != nil {
    log.Printf("Error: %v", err)
}
```

### 3. تسجيل لبوابة محددة

```go
ctx := context.Background()
gatewayID := 1
tenantID := uuid.MustParse("your-tenant-id")

// تسجيل المقاييس لبوابة محددة
err := logger.LogForGateway(ctx, gatewayID, tenantID)
if err != nil {
    log.Printf("Error: %v", err)
}
```

### 4. استرجاع السجلات الأخيرة

```go
ctx := context.Background()
gatewayID := 1
tenantID := uuid.MustParse("your-tenant-id")
limit := 10

// الحصول على آخر 10 سجلات
records, err := logger.GetRecentUsage(ctx, gatewayID, tenantID, limit)
if err != nil {
    log.Printf("Error: %v", err)
    return
}

for _, record := range records {
    fmt.Printf("Timestamp: %v\n", record.Timestamp)
    fmt.Printf("CPU: %.2f%%\n", record.CPUUsagePercent)
    fmt.Printf("Memory: %d MB\n", record.MemoryUsageMB)
    fmt.Printf("Bandwidth: %.2f MB/s\n", record.BandwidthUsageMBps)
    fmt.Printf("Messages (last hour): %d\n", record.MessagesProcessedLastHour)
    fmt.Println("---")
}
```

---

## البيانات المسجلة

### المقاييس الأساسية

| المقياس | الوصف | الوحدة |
|---------|--------|--------|
| `cpu_usage_percent` | نسبة استخدام المعالج | 0-100% |
| `memory_usage_mb` | استخدام الذاكرة | ميجابايت |
| `bandwidth_usage_mbps` | استخدام عرض النطاق | ميجابت/ثانية |
| `active_connections` | عدد الاتصالات النشطة | عدد |

### مقاييس قاعدة البيانات

| المقياس | الوصف |
|---------|--------|
| `db_connections_active` | عدد اتصالات قاعدة البيانات النشطة |
| `queries_per_second` | عدد الاستعلامات في الثانية |

### مقاييس Redis

| المقياس | الوصف |
|---------|--------|
| `redis_memory_usage_mb` | استخدام ذاكرة Redis (ميجابايت) |
| `redis_keys_count` | عدد المفاتيح في Redis |

### مقاييس معالجة الرسائل

| المقياس | الوصف |
|---------|--------|
| `messages_processed_last_hour` | عدد الرسائل المعالجة في الساعة الأخيرة |
| `messages_processed_today` | عدد الرسائل المعالجة اليوم |
| `current_queue_depth` | عمق قائمة الانتظار الحالي |

### مقاييس API

| المقياس | الوصف |
|---------|--------|
| `api_requests_last_minute` | عدد طلبات API في الدقيقة الأخيرة |
| `api_requests_last_hour` | عدد طلبات API في الساعة الأخيرة |

### مقاييس Workers

| المقياس | الوصف |
|---------|--------|
| `active_worker_threads` | عدد Worker Threads النشطة |

---

## التكامل مع Cron Job

### استخدام River Queue

```go
package main

import (
    "context"
    "database/sql"
    "log"
    "time"
    
    "github.com/al-hudhud/backend/internal/monitoring"
    "github.com/redis/go-redis/v9"
    "github.com/riverqueue/river"
    "github.com/riverqueue/river/riverdriver/riverpgxv5"
)

// UsageLoggerJob is a River job for logging resource usage
type UsageLoggerJob struct {
    river.JobArgs
}

func (UsageLoggerJob) Kind() string { return "usage_logger" }

func (j UsageLoggerJob) Work(ctx context.Context, job *river.Job[UsageLoggerJob]) error {
    // Get logger from context or create new one
    logger := getLoggerFromContext(ctx)
    
    return logger.LogResourceUsage(ctx)
}

func main() {
    // إعداد River
    workers := river.NewWorkers()
    river.AddWorker(workers, &UsageLoggerJob{})
    
    riverClient, err := river.NewClient(riverpgxv5.New(dbPool), &river.Config{
        Queues: map[string]river.QueueConfig{
            river.QueueDefault: {MaxWorkers: 10},
        },
        Workers: workers,
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // جدولة Job كل دقيقة
    _, err = riverClient.PeriodicJobs().Add(&river.PeriodicJobArgs{
        ConstructorFunc: func() (river.JobArgs, error) {
            return UsageLoggerJob{}, nil
        },
        RunOnStart: true,
        ScheduleFunc: func(t time.Time) time.Time {
            return t.Add(1 * time.Minute)
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // بدء River
    if err := riverClient.Start(context.Background()); err != nil {
        log.Fatal(err)
    }
    
    // الانتظار...
    select {}
}
```

### استخدام Ticker بسيط

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/al-hudhud/backend/internal/monitoring"
)

func startUsageLogger(logger *monitoring.Logger) {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()
    
    ctx := context.Background()
    
    // تسجيل فوري عند البدء
    if err := logger.LogResourceUsage(ctx); err != nil {
        log.Printf("Error logging resource usage: %v", err)
    }
    
    for {
        select {
        case <-ticker.C:
            if err := logger.LogResourceUsage(ctx); err != nil {
                log.Printf("Error logging resource usage: %v", err)
            }
        }
    }
}

func main() {
    // إعداد Logger
    logger := setupLogger()
    
    // بدء التسجيل الدوري
    go startUsageLogger(logger)
    
    // الانتظار...
    select {}
}
```

---

## الاستعلامات المفيدة

### 1. آخر تسجيل لبوابة معينة

```sql
SELECT *
FROM resource_usage
WHERE gateway_id = 1
ORDER BY timestamp DESC
LIMIT 1;
```

### 2. متوسط الاستخدام في الساعة الأخيرة

```sql
SELECT
    gateway_id,
    AVG(cpu_usage_percent) as avg_cpu,
    AVG(memory_usage_mb) as avg_memory,
    AVG(bandwidth_usage_mbps) as avg_bandwidth
FROM resource_usage
WHERE gateway_id = 1
  AND timestamp >= NOW() - INTERVAL '1 hour'
GROUP BY gateway_id;
```

### 3. ذروة الاستخدام اليوم

```sql
SELECT
    gateway_id,
    MAX(cpu_usage_percent) as max_cpu,
    MAX(memory_usage_mb) as max_memory,
    MAX(bandwidth_usage_mbps) as max_bandwidth,
    MAX(current_queue_depth) as max_queue_depth
FROM resource_usage
WHERE gateway_id = 1
  AND DATE(timestamp) = CURRENT_DATE
GROUP BY gateway_id;
```

### 4. الاستخدام على مدار اليوم (كل ساعة)

```sql
SELECT
    DATE_TRUNC('hour', timestamp) as hour,
    AVG(cpu_usage_percent) as avg_cpu,
    AVG(memory_usage_mb) as avg_memory,
    SUM(messages_processed_last_hour) as total_messages
FROM resource_usage
WHERE gateway_id = 1
  AND DATE(timestamp) = CURRENT_DATE
GROUP BY DATE_TRUNC('hour', timestamp)
ORDER BY hour;
```

---

## المزايا

### ✅ تسجيل تلقائي كل دقيقة

يعمل Logger بشكل دوري تلقائياً دون تدخل يدوي.

### ✅ عزل تام بين الشركات (Tenant Isolation)

جميع العمليات تحترم Row Level Security (RLS) وتستخدم `app.current_tenant`.

### ✅ معالجة الأخطاء بشكل آمن

إذا فشل تسجيل بوابة واحدة، يستمر Logger في تسجيل البوابات الأخرى.

### ✅ دعم Partitioning

جدول `resource_usage` يستخدم Partitioning حسب الشهر لتحسين الأداء.

### ✅ استرجاع سهل للبيانات

يوفر Logger وظيفة `GetRecentUsage` لاسترجاع السجلات الأخيرة بسهولة.

---

## الاختبارات

### تشغيل الاختبارات

```bash
cd services/backend
go test -v ./internal/monitoring/...
```

### الاختبارات المتوفرة

- ✅ `TestNewLogger`: إنشاء Logger جديد
- ✅ `TestLogGatewayMetrics`: تسجيل مقاييس بوابة واحدة
- ✅ `TestLogResourceUsage`: تسجيل مقاييس جميع البوابات
- ✅ `TestGetRecentUsage`: استرجاع السجلات الأخيرة
- ✅ `TestGetRecentUsage_WithNullValues`: معالجة القيم NULL
- ✅ `TestLogForGateway`: تسجيل لبوابة محددة
- ✅ `TestGetMessageMetrics`: جمع مقاييس الرسائل
- ✅ `TestGetMessageMetrics_NoMessages`: معالجة حالة عدم وجود رسائل
- ✅ `TestGetAPIMetrics`: جمع مقاييس API
- ✅ `TestGetWorkerMetrics`: جمع مقاييس Workers

---

## الأداء

### تحسينات الأداء

1. **Partitioning**: جدول `resource_usage` مقسم حسب الشهر
2. **Indexes**: Indexes محسّنة للاستعلامات الشائعة
3. **Batch Processing**: يمكن تسجيل عدة بوابات في نفس الوقت
4. **Error Handling**: فشل بوابة واحدة لا يؤثر على الأخرى

### الاستهلاك المتوقع

- **تسجيل واحد**: ~10ms
- **تسجيل 10 بوابات**: ~100ms
- **حجم السجل الواحد**: ~200 bytes
- **حجم البيانات اليومي** (10 بوابات): ~2.8 MB

---

## الصيانة

### حذف البيانات القديمة

```sql
-- حذف partitions أقدم من 90 يوم
SELECT cleanup_old_resource_usage_partitions();
```

### إنشاء Partition جديد

```sql
-- إنشاء partition للشهر القادم
SELECT create_resource_usage_partition('2025-08-01');
```

### تحليل الأداء

```sql
-- تحليل جدول resource_usage
VACUUM ANALYZE resource_usage;
```

---

## التكامل مع المهام التالية

### المهمة 19.3: Alert Manager

سيستخدم Alert Manager البيانات المسجلة لفحص العتبات:

```go
// مثال: فحص استخدام CPU
records, _ := logger.GetRecentUsage(ctx, gatewayID, tenantID, 1)
if len(records) > 0 && records[0].CPUUsagePercent > 90 {
    alertManager.SendAlert("CPU usage critical", gatewayID)
}
```

### المهمة 19.4: Statistics Aggregator

سيستخدم Statistics Aggregator البيانات المسجلة لحساب الإحصائيات اليومية:

```sql
-- حساب الإحصائيات اليومية
INSERT INTO gateway_statistics (gateway_id, date, avg_cpu, max_cpu, ...)
SELECT
    gateway_id,
    DATE(timestamp),
    AVG(cpu_usage_percent),
    MAX(cpu_usage_percent),
    ...
FROM resource_usage
WHERE DATE(timestamp) = CURRENT_DATE
GROUP BY gateway_id, DATE(timestamp);
```

---

## الموارد

- **الكود**: `internal/monitoring/logger.go`
- **الاختبارات**: `internal/monitoring/logger_test.go`
- **Migration**: `migrations/022_resource_usage_table.sql`
- **المتطلبات**: `.kiro/specs/tariff-pricing-system/requirements.md` (المتطلب 10)
- **التصميم**: `.kiro/specs/tariff-pricing-system/design.md`

---

## الخطوات التالية

1. ✅ **المهمة 19.1**: Metrics Collector (مكتمل)
2. ✅ **المهمة 19.2**: Usage Logger (مكتمل)
3. ⏳ **المهمة 19.3**: Alert Manager - فحص العتبات وإرسال التنبيهات
4. ⏳ **المهمة 19.4**: Statistics Aggregator - حساب الإحصائيات اليومية
5. ⏳ **المهمة 19.5**: Balance Monitor - مراقبة رصيد البوابات المخصصة
6. ⏳ **المهمة 19.6**: Subscription Billing - فوترة اشتراكات البوابات

---

## الدعم

للمزيد من المعلومات:
1. راجع `internal/monitoring/README.md` للتوثيق الكامل
2. راجع المتطلب 10 في مستند المتطلبات
3. راجع التصميم في `design.md`
4. راجع `MONITORING_QUICK_START.md` لدليل Metrics Collector
