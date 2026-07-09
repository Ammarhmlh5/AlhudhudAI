# Tenant Worker Pool - دليل البدء السريع

## نظرة عامة

تم تطبيق **Task 18.3: تطوير Worker Pool محدود** بنجاح! 🎉

يوفر `TenantWorkerPool` نظام Worker Pool محدود لكل بوابة مخصصة مع:
- ✅ عدد محدود من Workers (من `max_worker_threads`)
- ✅ Job Queue محدود لمنع استهلاك الذاكرة
- ✅ عزل كامل بين البوابات المختلفة
- ✅ Rate Limiting لكل بوابة
- ✅ Graceful Shutdown

## البدء السريع

### 1. إنشاء Worker Pool

```go
import (
    "context"
    "github.com/al-hudhud/backend/internal/resources"
    "github.com/google/uuid"
    "go.uber.org/zap"
)

func main() {
    logger, _ := zap.NewProduction()
    
    // Configuration
    config := &resources.TenantWorkerPoolConfig{
        TenantID:   uuid.New(),
        GatewayID:  1,
        MaxWorkers: 10,    // من resource_quotas.max_worker_threads
        QueueSize:  100,   // حجم Queue المحدود
        RateLimit:  1000,  // Jobs per second
    }
    
    // إنشاء وتشغيل Pool
    pool := resources.NewTenantWorkerPool(config, logger)
    ctx := context.Background()
    pool.Start(ctx)
    defer pool.Stop(context.Background())
}
```

### 2. تطبيق Job Interface

```go
type MessageJob struct {
    id        string
    tenantID  uuid.UUID
    gatewayID int
    message   string
}

func (j *MessageJob) Execute(ctx context.Context) error {
    // معالجة الرسالة
    fmt.Printf("Processing: %s\n", j.message)
    return nil
}

func (j *MessageJob) GetID() string       { return j.id }
func (j *MessageJob) GetTenantID() uuid.UUID { return j.tenantID }
func (j *MessageJob) GetGatewayID() int   { return j.gatewayID }
```

### 3. إرسال Jobs

```go
// إنشاء Job
job := &MessageJob{
    id:        uuid.New().String(),
    tenantID:  config.TenantID,
    gatewayID: config.GatewayID,
    message:   "Hello World",
}

// إرسال Job
err := pool.Submit(ctx, job)
if err != nil {
    log.Printf("Failed: %v", err)
}
```

### 4. مراقبة الحالة

```go
status := pool.GetStatus()
fmt.Printf("Workers: %d/%d\n", status.ActiveWorkers, status.MaxWorkers)
fmt.Printf("Queue: %d/%d\n", status.QueueLength, status.QueueSize)
fmt.Printf("Processed: %d\n", status.ProcessedCount)
fmt.Printf("Utilization: %.2f%%\n", pool.GetQueueUtilization())
```

## المزايا الرئيسية

### Bounded Queue

```go
// Queue محدود يمنع استهلاك الذاكرة
config := &TenantWorkerPoolConfig{
    QueueSize: 100,  // حد أقصى 100 job
}

// عند امتلاء Queue، يتم رفض Jobs
err := pool.TrySubmit(job)
if err != nil {
    // Queue full
}
```

### Complete Isolation

```go
// كل بوابة لها Pool منفصل
pool1 := NewTenantWorkerPool(&TenantWorkerPoolConfig{
    TenantID: tenant1, GatewayID: 1, MaxWorkers: 5,
})

pool2 := NewTenantWorkerPool(&TenantWorkerPoolConfig{
    TenantID: tenant2, GatewayID: 2, MaxWorkers: 10,
})

// Jobs من tenant1 لا يمكن إرسالها لـ pool2
```

### Rate Limiting

```go
config := &TenantWorkerPoolConfig{
    RateLimit: 100, // 100 jobs/sec
}

// يتم تطبيق Rate Limiting تلقائياً
```

## استخدام Pool Manager

```go
// إنشاء Manager
manager := resources.NewTenantWorkerPoolManager(logger)

// إنشاء أو الحصول على Pool
pool, err := manager.GetOrCreatePool(ctx, config)

// الحصول على Pool موجود
pool, err = manager.GetPool(tenantID, gatewayID)

// إيقاف Pool محدد
manager.StopPool(ctx, tenantID, gatewayID)

// إيقاف جميع Pools
manager.StopAll(ctx)
```

## التكامل مع Resource Quotas

```go
// الحصول على max_worker_threads من قاعدة البيانات
var maxWorkerThreads int
db.QueryRow(`
    SELECT max_worker_threads
    FROM resource_quotas
    WHERE gateway_id = $1 AND is_active = true
`, gatewayID).Scan(&maxWorkerThreads)

// إنشاء Pool بناءً على Quota
config := &TenantWorkerPoolConfig{
    MaxWorkers: maxWorkerThreads,
    QueueSize:  maxWorkerThreads * 10,
}
```

## الاختبارات

```bash
# تشغيل جميع الاختبارات
cd services/backend
go test -v ./internal/resources -run TestTenantWorkerPool

# النتيجة: جميع الاختبارات نجحت ✅
# - TestNewTenantWorkerPool
# - TestTenantWorkerPool_StartStop
# - TestTenantWorkerPool_SubmitJob
# - TestTenantWorkerPool_MultipleJobs
# - TestTenantWorkerPool_BoundedQueue
# - TestTenantWorkerPool_TenantIsolation
# - TestTenantWorkerPool_GatewayIsolation
# - TestTenantWorkerPool_RateLimiting
# - TestTenantWorkerPool_FailedJobs
# - TestTenantWorkerPool_QueueUtilization
# - TestTenantWorkerPoolManager
# - TestTenantWorkerPool_ConcurrentSubmissions
# - TestTenantWorkerPool_GracefulShutdown
```

## الملفات المُنفذة

```
services/backend/internal/resources/
├── tenant_pool.go              # التطبيق الرئيسي
├── tenant_pool_test.go         # الاختبارات الشاملة (12 اختبار)
├── TENANT_WORKER_POOL_README.md # الدليل الكامل
└── limiter.go                  # Rate Limiter (Task 18.2)
```

## المتطلبات المُنفذة

- ✅ **Requirement 9.9**: دعم تحديد عدد Worker Threads المسموحة
- ✅ **Bounded Queue**: Job Queue محدود لمنع استهلاك الذاكرة
- ✅ **Complete Isolation**: عزل كامل بين البوابات
- ✅ **Rate Limiting**: تطبيق Rate Limiting لكل بوابة
- ✅ **Graceful Shutdown**: إيقاف آمن مع انتظار Jobs الحالية
- ✅ **Comprehensive Tests**: 12 اختبار شامل

## الخطوات التالية

بعد Task 18.3، يمكن الانتقال إلى:

1. **Task 18.4**: تطوير Enforcement Policies (Soft/Hard Limits)
2. **Task 18.5**: كتابة Property Test لـ Rate Limiting per Tenant
3. **Task 19**: تطوير Monitoring Service

## الوثائق الكاملة

للمزيد من التفاصيل، راجع:
- [Tenant Worker Pool README](./internal/resources/TENANT_WORKER_POOL_README.md)
- [Rate Limiter README](./internal/resources/RATE_LIMITER_README.md)
- [Resource Quotas README](./internal/resources/README.md)

## الدعم

للأسئلة أو المساعدة:
- راجع الوثائق الكاملة
- راجع الاختبارات للأمثلة
- راجع Design Document في `.kiro/specs/tariff-pricing-system/design.md`
