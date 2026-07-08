# Tenant Worker Pool - دليل الاستخدام

## نظرة عامة

يوفر `TenantWorkerPool` نظام Worker Pool محدود لكل بوابة مخصصة، مع عزل كامل بين البوابات المختلفة. يتم تحديد عدد Workers وحجم Queue بناءً على `max_worker_threads` من `resource_quotas`.

## المتطلبات المُنفذة

- ✅ **Requirement 9.9**: دعم تحديد عدد Worker Threads المسموحة لكل بوابة مخصصة
- ✅ **Bounded Queue**: تطبيق Job Queue محدود لمنع استهلاك الذاكرة
- ✅ **Complete Isolation**: عزل كامل بين البوابات المختلفة
- ✅ **Rate Limiting**: تطبيق Rate Limiting لكل بوابة
- ✅ **Graceful Shutdown**: إيقاف آمن مع انتظار إكمال Jobs الحالية

## المكونات الرئيسية

### 1. TenantWorkerPool

Worker Pool مخصص لبوابة واحدة مع عدد محدود من Workers وQueue محدود.

```go
type TenantWorkerPool struct {
    tenantID    uuid.UUID
    gatewayID   int
    workers     []*Worker
    jobQueue    chan Job      // Bounded queue
    maxWorkers  int           // من resource_quotas.max_worker_threads
    queueSize   int           // حجم Queue المحدود
    rateLimiter *rate.Limiter // Rate limiting per gateway
}
```

### 2. Worker

Worker فردي يعالج Jobs من Queue.

```go
type Worker struct {
    id          int
    tenantID    uuid.UUID
    gatewayID   int
    jobQueue    <-chan Job
    // Metrics
    processedCount atomic.Int64
    failedCount    atomic.Int64
    lastProcessed  atomic.Value
}
```

### 3. Job Interface

واجهة يجب أن تُطبقها جميع Jobs.

```go
type Job interface {
    Execute(ctx context.Context) error
    GetID() string
    GetTenantID() uuid.UUID
    GetGatewayID() int
}
```

### 4. TenantWorkerPoolManager

مدير لإدارة عدة Worker Pools لبوابات مختلفة.

```go
type TenantWorkerPoolManager struct {
    pools  map[string]*TenantWorkerPool // key: "tenantID:gatewayID"
}
```

## الاستخدام

### 1. إنشاء Worker Pool

```go
import (
    "github.com/al-hudhud/backend/internal/resources"
    "github.com/google/uuid"
    "go.uber.org/zap"
)

// إنشاء configuration
config := &resources.TenantWorkerPoolConfig{
    TenantID:   tenantID,
    GatewayID:  gatewayID,
    MaxWorkers: 10,        // من resource_quotas.max_worker_threads
    QueueSize:  100,       // حجم Queue المحدود (افتراضياً: MaxWorkers * 10)
    RateLimit:  1000,      // Jobs per second (افتراضياً: 1000)
}

// إنشاء Worker Pool
logger := zap.NewProduction()
pool := resources.NewTenantWorkerPool(config, logger)

// تشغيل Pool
ctx := context.Background()
err := pool.Start(ctx)
if err != nil {
    log.Fatal(err)
}
defer pool.Stop(context.Background())
```

### 2. تطبيق Job Interface

```go
type MessageJob struct {
    id        string
    tenantID  uuid.UUID
    gatewayID int
    message   *Message
}

func (j *MessageJob) Execute(ctx context.Context) error {
    // معالجة الرسالة
    return processMessage(ctx, j.message)
}

func (j *MessageJob) GetID() string {
    return j.id
}

func (j *MessageJob) GetTenantID() uuid.UUID {
    return j.tenantID
}

func (j *MessageJob) GetGatewayID() int {
    return j.gatewayID
}
```

### 3. إرسال Jobs

```go
// إنشاء Job
job := &MessageJob{
    id:        uuid.New().String(),
    tenantID:  tenantID,
    gatewayID: gatewayID,
    message:   message,
}

// إرسال Job مع انتظار (يطبق Rate Limiting)
err := pool.Submit(ctx, job)
if err != nil {
    log.Printf("Failed to submit job: %v", err)
}

// أو إرسال Job بدون انتظار (يرفض فوراً إذا كان Queue ممتلئ)
err = pool.TrySubmit(job)
if err != nil {
    log.Printf("Job rejected: %v", err)
}
```

### 4. مراقبة الحالة

```go
// الحصول على حالة Pool
status := pool.GetStatus()
fmt.Printf("Pool Status:\n")
fmt.Printf("  Running: %v\n", status.IsRunning)
fmt.Printf("  Workers: %d/%d\n", status.ActiveWorkers, status.MaxWorkers)
fmt.Printf("  Queue: %d/%d\n", status.QueueLength, status.QueueSize)
fmt.Printf("  Submitted: %d\n", status.SubmittedCount)
fmt.Printf("  Rejected: %d\n", status.RejectedCount)
fmt.Printf("  Processed: %d\n", status.ProcessedCount)
fmt.Printf("  Failed: %d\n", status.FailedCount)

// الحصول على Queue metrics
queueLength := pool.GetQueueLength()
queueCapacity := pool.GetQueueCapacity()
queueUtilization := pool.GetQueueUtilization() // 0-100%
isQueueFull := pool.IsQueueFull()

fmt.Printf("Queue Utilization: %.2f%%\n", queueUtilization)
```

### 5. استخدام Pool Manager

```go
// إنشاء Manager
manager := resources.NewTenantWorkerPoolManager(logger)

// إنشاء أو الحصول على Pool
pool, err := manager.GetOrCreatePool(ctx, config)
if err != nil {
    log.Fatal(err)
}

// الحصول على Pool موجود
pool, err = manager.GetPool(tenantID, gatewayID)
if err != nil {
    log.Printf("Pool not found: %v", err)
}

// إيقاف Pool محدد
err = manager.StopPool(ctx, tenantID, gatewayID)

// إيقاف جميع Pools
err = manager.StopAll(ctx)

// الحصول على حالة جميع Pools
statuses := manager.GetAllStatuses()
for key, status := range statuses {
    fmt.Printf("Pool %s: %d workers, %d queued\n", 
        key, status.ActiveWorkers, status.QueueLength)
}
```

## المزايا الرئيسية

### 1. Bounded Queue (Queue محدود)

```go
// Queue محدود يمنع استهلاك الذاكرة
config := &TenantWorkerPoolConfig{
    MaxWorkers: 10,
    QueueSize:  100,  // حد أقصى 100 job في Queue
}

// عند امتلاء Queue، يتم رفض Jobs الجديدة
err := pool.TrySubmit(job)
if err != nil {
    // Queue full - handle rejection
    log.Printf("Queue full: %v", err)
}
```

### 2. Complete Isolation (عزل كامل)

```go
// كل بوابة لها Pool منفصل تماماً
pool1 := NewTenantWorkerPool(&TenantWorkerPoolConfig{
    TenantID:  tenant1,
    GatewayID: gateway1,
    MaxWorkers: 5,
})

pool2 := NewTenantWorkerPool(&TenantWorkerPoolConfig{
    TenantID:  tenant2,
    GatewayID: gateway2,
    MaxWorkers: 10,
})

// Jobs من tenant1 لا يمكن إرسالها لـ pool2
job := &MessageJob{tenantID: tenant1, gatewayID: gateway1}
err := pool2.Submit(ctx, job) // Error: tenant_id mismatch
```

### 3. Rate Limiting

```go
// Rate limiting لكل بوابة
config := &TenantWorkerPoolConfig{
    RateLimit: 100, // 100 jobs per second
}

// يتم تطبيق Rate Limiting تلقائياً
for i := 0; i < 1000; i++ {
    err := pool.Submit(ctx, job) // ينتظر حسب Rate Limit
}
```

### 4. Graceful Shutdown

```go
// إيقاف آمن مع انتظار إكمال Jobs الحالية
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

err := pool.Stop(ctx)
if err != nil {
    log.Printf("Shutdown timeout: %v", err)
}
```

## التكامل مع Resource Quotas

```go
import (
    "database/sql"
    "github.com/al-hudhud/backend/internal/resources"
)

// الحصول على max_worker_threads من resource_quotas
func CreatePoolFromQuota(db *sql.DB, tenantID uuid.UUID, gatewayID int) (*resources.TenantWorkerPool, error) {
    // استعلام resource_quotas
    var maxWorkerThreads int
    err := db.QueryRow(`
        SELECT max_worker_threads
        FROM resource_quotas
        WHERE gateway_id = $1 AND is_active = true
    `, gatewayID).Scan(&maxWorkerThreads)
    
    if err != nil {
        return nil, err
    }
    
    // إنشاء Pool بناءً على Quota
    config := &resources.TenantWorkerPoolConfig{
        TenantID:   tenantID,
        GatewayID:  gatewayID,
        MaxWorkers: maxWorkerThreads,
        QueueSize:  maxWorkerThreads * 10,
        RateLimit:  1000,
    }
    
    logger := zap.NewProduction()
    pool := resources.NewTenantWorkerPool(config, logger)
    
    ctx := context.Background()
    err = pool.Start(ctx)
    if err != nil {
        return nil, err
    }
    
    return pool, nil
}
```

## مثال كامل

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/al-hudhud/backend/internal/resources"
    "github.com/google/uuid"
    "go.uber.org/zap"
)

// تطبيق Job Interface
type MessageJob struct {
    id        string
    tenantID  uuid.UUID
    gatewayID int
    content   string
}

func (j *MessageJob) Execute(ctx context.Context) error {
    fmt.Printf("Processing message: %s\n", j.content)
    time.Sleep(100 * time.Millisecond) // محاكاة معالجة
    return nil
}

func (j *MessageJob) GetID() string       { return j.id }
func (j *MessageJob) GetTenantID() uuid.UUID { return j.tenantID }
func (j *MessageJob) GetGatewayID() int   { return j.gatewayID }

func main() {
    logger, _ := zap.NewProduction()
    defer logger.Sync()
    
    tenantID := uuid.New()
    gatewayID := 1
    
    // إنشاء Worker Pool
    config := &resources.TenantWorkerPoolConfig{
        TenantID:   tenantID,
        GatewayID:  gatewayID,
        MaxWorkers: 5,
        QueueSize:  50,
        RateLimit:  100,
    }
    
    pool := resources.NewTenantWorkerPool(config, logger)
    
    ctx := context.Background()
    err := pool.Start(ctx)
    if err != nil {
        logger.Fatal("Failed to start pool", zap.Error(err))
    }
    defer pool.Stop(context.Background())
    
    // إرسال Jobs
    for i := 0; i < 20; i++ {
        job := &MessageJob{
            id:        uuid.New().String(),
            tenantID:  tenantID,
            gatewayID: gatewayID,
            content:   fmt.Sprintf("Message %d", i),
        }
        
        err := pool.Submit(ctx, job)
        if err != nil {
            logger.Error("Failed to submit job", zap.Error(err))
        }
    }
    
    // انتظار المعالجة
    time.Sleep(3 * time.Second)
    
    // عرض الحالة
    status := pool.GetStatus()
    fmt.Printf("\nFinal Status:\n")
    fmt.Printf("  Submitted: %d\n", status.SubmittedCount)
    fmt.Printf("  Processed: %d\n", status.ProcessedCount)
    fmt.Printf("  Failed: %d\n", status.FailedCount)
    fmt.Printf("  Rejected: %d\n", status.RejectedCount)
}
```

## Best Practices

### 1. تحديد حجم Queue المناسب

```go
// قاعدة عامة: QueueSize = MaxWorkers * 10
config := &TenantWorkerPoolConfig{
    MaxWorkers: 10,
    QueueSize:  100, // 10x workers
}

// للأحمال العالية: استخدم حجم أكبر
config := &TenantWorkerPoolConfig{
    MaxWorkers: 10,
    QueueSize:  500, // 50x workers
}
```

### 2. معالجة Queue Full

```go
// استخدام TrySubmit للتعامل مع Queue الممتلئ
err := pool.TrySubmit(job)
if err != nil {
    if strings.Contains(err.Error(), "queue full") {
        // Queue ممتلئ - يمكن:
        // 1. إعادة المحاولة لاحقاً
        // 2. حفظ Job في قاعدة البيانات
        // 3. رفض الطلب مع رسالة مناسبة
        return handleQueueFull(job)
    }
    return err
}
```

### 3. مراقبة Queue Utilization

```go
// مراقبة استخدام Queue
utilization := pool.GetQueueUtilization()
if utilization > 80.0 {
    // Queue شبه ممتلئ - اتخاذ إجراء
    logger.Warn("Queue utilization high",
        zap.Float64("utilization", utilization),
        zap.Int("gateway_id", gatewayID),
    )
    
    // يمكن:
    // 1. زيادة عدد Workers
    // 2. تطبيق Backpressure
    // 3. إرسال تنبيه
}
```

### 4. Graceful Shutdown

```go
// إيقاف آمن مع timeout
func shutdownPool(pool *resources.TenantWorkerPool) error {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    logger.Info("Shutting down worker pool...")
    
    err := pool.Stop(ctx)
    if err != nil {
        logger.Error("Shutdown failed", zap.Error(err))
        return err
    }
    
    logger.Info("Worker pool stopped successfully")
    return nil
}
```

## الاختبارات

تم تطبيق اختبارات شاملة في `tenant_pool_test.go`:

- ✅ `TestNewTenantWorkerPool`: إنشاء Pool
- ✅ `TestTenantWorkerPool_StartStop`: تشغيل وإيقاف
- ✅ `TestTenantWorkerPool_SubmitJob`: إرسال Jobs
- ✅ `TestTenantWorkerPool_MultipleJobs`: معالجة عدة Jobs
- ✅ `TestTenantWorkerPool_BoundedQueue`: Queue محدود
- ✅ `TestTenantWorkerPool_TenantIsolation`: عزل Tenants
- ✅ `TestTenantWorkerPool_GatewayIsolation`: عزل Gateways
- ✅ `TestTenantWorkerPool_RateLimiting`: Rate Limiting
- ✅ `TestTenantWorkerPool_FailedJobs`: معالجة Jobs الفاشلة
- ✅ `TestTenantWorkerPool_QueueUtilization`: Queue metrics
- ✅ `TestTenantWorkerPoolManager`: Pool Manager
- ✅ `TestTenantWorkerPool_ConcurrentSubmissions`: إرسال متزامن
- ✅ `TestTenantWorkerPool_GracefulShutdown`: إيقاف آمن

### تشغيل الاختبارات

```bash
# تشغيل جميع الاختبارات
cd services/backend
go test -v ./internal/resources -run TestTenantWorkerPool

# تشغيل اختبار محدد
go test -v ./internal/resources -run TestTenantWorkerPool_BoundedQueue

# تشغيل مع Coverage
go test -v -cover ./internal/resources -run TestTenantWorkerPool
```

## الملفات ذات الصلة

- `tenant_pool.go`: التطبيق الرئيسي
- `tenant_pool_test.go`: الاختبارات الشاملة
- `limiter.go`: Resource Limiter و Rate Limiter
- `limiter_test.go`: اختبارات Resource Limiter

## الخطوات التالية

بعد تطبيق Task 18.3، يمكن الانتقال إلى:

- **Task 18.4**: تطوير Enforcement Policies (Soft/Hard Limits)
- **Task 18.5**: كتابة Property Test لـ Rate Limiting per Tenant
- **Task 19**: تطوير Monitoring Service

## الدعم

للمزيد من المعلومات، راجع:
- [Resource Quotas README](./README.md)
- [Rate Limiter README](./RATE_LIMITER_README.md)
- [Design Document](.kiro/specs/tariff-pricing-system/design.md)
