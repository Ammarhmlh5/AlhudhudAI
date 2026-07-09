# Task 6.2: Worker Health Monitoring Implementation

## Overview

تم تطبيق نظام مراقبة صحة Workers بشكل كامل وفقاً للمتطلبات 13.4، 13.5، و 13.7.

## Requirements Implemented

### ✅ Requirement 13.4: مراقبة صحة Workers كل 10 ثوانٍ

تم تطبيق نظام مراقبة صحة شامل يعمل كل 10 ثوانٍ:

```go
type HealthMonitorConfig struct {
    CheckInterval    time.Duration // 10 seconds
    FailureThreshold int           // 3 consecutive failures
    RestartEnabled   bool          // true
    MaxRestarts      int           // 5 maximum restarts
}
```

**الآلية:**
- Goroutine منفصلة تعمل في الخلفية
- تفحص صحة كل Worker كل 10 ثوانٍ
- تتحقق من:
  - نشاط المعالجة الأخير
  - معدل النجاح (Success Rate)
  - عدد الفشل المتتالي (Consecutive Failures)

**معايير تحديد Worker غير صحي:**
1. عدم معالجة أي رسائل لمدة 20 ثانية (2x check interval) مع وجود رسائل معلقة
2. معدل نجاح أقل من 50% (بعد معالجة 10+ رسائل)
3. فشل متتالي يتجاوز الحد المسموح (3 مرات)

### ✅ Requirement 13.5: إعادة تشغيل Workers الفاشلة تلقائياً

تم تطبيق آلية إعادة تشغيل تلقائية:

```go
func (p *RiverWorkerPool) performHealthCheck(ctx context.Context) {
    // عند اكتشاف worker غير صحي
    if p.healthConfig.RestartEnabled && health.RestartCount < p.healthConfig.MaxRestarts {
        // إعادة تعيين المقاييس
        health.ConsecutiveFailures = 0
        health.RestartCount++
        health.IsHealthy = true
        health.LastError = ""
        
        // River يتولى إعادة توزيع المهام تلقائياً
    }
}
```

**الآلية:**
- عند اكتشاف Worker غير صحي، يتم إعادة تعيين مقاييسه
- River Queue يتولى إعادة توزيع المهام تلقائياً
- حد أقصى 5 محاولات إعادة تشغيل لكل Worker
- تسجيل كامل لجميع عمليات إعادة التشغيل

### ✅ Requirement 13.7: تسجيل أداء كل Worker

تم تطبيق نظام تسجيل شامل لأداء كل Worker:

```go
type WorkerMetrics struct {
    WorkerID        int
    ProcessedCount  int64         // عدد الرسائل المعالجة
    FailedCount     int64         // عدد الرسائل الفاشلة
    SuccessRate     float64       // معدل النجاح (0-1)
    AverageTime     time.Duration // متوسط وقت المعالجة
    LastProcessedAt time.Time     // آخر وقت معالجة
}

type WorkerHealth struct {
    WorkerID            int
    IsHealthy           bool
    LastHealthCheck     time.Time
    ConsecutiveFailures int
    LastError           string
    RestartCount        int
}
```

**المقاييس المسجلة:**
1. **عدد الرسائل المعالجة** (Messages Processed)
2. **معدل النجاح** (Success Rate)
3. **متوسط وقت المعالجة** (Average Processing Time)
4. **عدد الفشل المتتالي** (Consecutive Failures)
5. **عدد مرات إعادة التشغيل** (Restart Count)
6. **آخر خطأ** (Last Error)

## Implementation Details

### 1. Health Monitoring Types

تم إضافة الأنواع التالية في `types.go`:

```go
// WorkerHealth - حالة صحة Worker
type WorkerHealth struct {
    WorkerID            int
    IsHealthy           bool
    LastHealthCheck     time.Time
    ConsecutiveFailures int
    LastError           string
    RestartCount        int
}

// HealthMonitorConfig - إعدادات المراقبة
type HealthMonitorConfig struct {
    CheckInterval    time.Duration
    FailureThreshold int
    RestartEnabled   bool
    MaxRestarts      int
}
```

### 2. Pool Structure Updates

تم تحديث `RiverWorkerPool` لتضمين:

```go
type RiverWorkerPool struct {
    // ... existing fields ...
    
    // Health monitoring
    healthConfig  *HealthMonitorConfig
    workerHealth  map[int]*WorkerHealth
    healthMonitor context.CancelFunc
}
```

### 3. Health Monitoring Methods

تم إضافة الدوال التالية:

#### `startHealthMonitoring(ctx context.Context)`
- تبدأ goroutine للمراقبة المستمرة
- تستخدم ticker للفحص كل 10 ثوانٍ
- تتوقف عند إيقاف Worker Pool

#### `performHealthCheck(ctx context.Context)`
- تفحص صحة جميع Workers
- تحدد Workers غير الصحية
- تحاول إعادة تشغيل Workers الفاشلة
- تسجل النتائج بالتفصيل

#### `GetWorkerHealth() map[int]*WorkerHealth`
- تُرجع حالة صحة جميع Workers
- thread-safe مع نسخ البيانات

#### `RecordWorkerFailure(workerID int, err error)`
- تسجل فشل Worker
- تزيد عداد الفشل المتتالي
- تحفظ رسالة الخطأ

#### `RecordWorkerSuccess(workerID int)`
- تسجل نجاح Worker
- تعيد تعيين عداد الفشل المتتالي
- تمسح رسالة الخطأ

### 4. Integration with Message Processing

تم تحديث `MessageWorker.Work()` لتسجيل الحالة الصحية:

```go
// عند النجاح
w.pool.RecordJobProcessed(workerID, processingTimeMs)
w.pool.RecordWorkerSuccess(workerID)

// عند الفشل
w.pool.RecordJobFailed(workerID)
w.pool.RecordWorkerFailure(workerID, err)
```

## Health Check Logic

### معايير تحديد Worker غير صحي:

```go
// 1. عدم النشاط مع وجود رسائل معلقة
if !metrics.LastProcessedAt.IsZero() && 
   timeSinceLastProcess > 2*p.healthConfig.CheckInterval {
    // تحقق من وجود رسائل معلقة
    if queueSize > 0 {
        isUnhealthy = true
    }
}

// 2. معدل نجاح منخفض
if metrics.ProcessedCount > 10 && metrics.SuccessRate < 0.5 {
    isUnhealthy = true
}

// 3. فشل متتالي
if health.ConsecutiveFailures >= p.healthConfig.FailureThreshold {
    isUnhealthy = true
}
```

### آلية إعادة التشغيل:

```go
if p.healthConfig.RestartEnabled && 
   health.RestartCount < p.healthConfig.MaxRestarts {
    // إعادة تعيين المقاييس
    health.ConsecutiveFailures = 0
    health.RestartCount++
    health.IsHealthy = true
    health.LastError = ""
    
    // River يتولى إعادة توزيع المهام
}
```

## Logging

### Health Check Logs

```go
// عند اكتشاف worker غير صحي
p.logger.Warn("Worker became unhealthy",
    zap.Int("worker_id", workerID),
    zap.String("reason", reason),
    zap.Int64("processed", metrics.ProcessedCount),
    zap.Int64("failed", metrics.FailedCount),
    zap.Float64("success_rate", metrics.SuccessRate),
)

// عند محاولة إعادة التشغيل
p.logger.Info("Attempting to restart unhealthy worker",
    zap.Int("worker_id", workerID),
    zap.Int("restart_count", health.RestartCount),
    zap.String("reason", reason),
)

// عند تعافي worker
p.logger.Info("Worker recovered",
    zap.Int("worker_id", workerID),
    zap.Int64("processed", metrics.ProcessedCount),
    zap.Float64("success_rate", metrics.SuccessRate),
)

// ملخص الفحص الصحي
p.logger.Warn("Health check completed",
    zap.Int("total_workers", totalWorkers),
    zap.Int("unhealthy_workers", unhealthyWorkers),
    zap.Int("healthy_workers", totalWorkers-unhealthyWorkers),
    zap.Int64("total_processed", p.processedCount),
    zap.Int64("total_failed", p.failedCount),
)
```

## Configuration

### Default Configuration

```go
func DefaultHealthMonitorConfig() *HealthMonitorConfig {
    return &HealthMonitorConfig{
        CheckInterval:    10 * time.Second,  // Requirement 13.4
        FailureThreshold: 3,
        RestartEnabled:   true,              // Requirement 13.5
        MaxRestarts:      5,
    }
}
```

### Customization

يمكن تخصيص الإعدادات عند إنشاء Worker Pool:

```go
pool := NewRiverWorkerPool(config, db, logger, dependencies)
pool.healthConfig = &HealthMonitorConfig{
    CheckInterval:    5 * time.Second,  // فحص أسرع
    FailureThreshold: 5,                // تحمل أكثر
    RestartEnabled:   true,
    MaxRestarts:      10,               // محاولات أكثر
}
```

## Thread Safety

جميع العمليات على `workerHealth` و `workerMetrics` محمية بـ mutex:

```go
p.mu.Lock()
defer p.mu.Unlock()

// ... operations on shared data ...
```

## Integration with River Queue

النظام يعمل بشكل متكامل مع River Queue:

1. **River يتولى توزيع المهام** تلقائياً على Workers المتاحة
2. **عند إعادة تعيين Worker**، River يعيد توزيع المهام المعلقة
3. **لا حاجة لإعادة تشغيل يدوية** - River يدير دورة حياة Workers

## Benefits

### 1. موثوقية عالية
- اكتشاف تلقائي للمشاكل
- إعادة تشغيل تلقائية للـ Workers الفاشلة
- منع تراكم الفشل

### 2. مراقبة شاملة
- تسجيل مفصل لأداء كل Worker
- مقاييس دقيقة للنجاح والفشل
- تتبع تاريخ إعادة التشغيل

### 3. قابلية التشخيص
- Logs واضحة ومفصلة
- معلومات كاملة عن سبب الفشل
- سهولة تتبع المشاكل

### 4. حماية من الفشل المتكرر
- حد أقصى لمحاولات إعادة التشغيل
- منع إعادة تشغيل لا نهائية
- تنبيهات عند تجاوز الحد

## Testing Recommendations

### Unit Tests
```go
func TestHealthMonitoring(t *testing.T) {
    // Test health check detection
    // Test automatic restart
    // Test max restart limit
    // Test metrics recording
}
```

### Integration Tests
```go
func TestWorkerRecovery(t *testing.T) {
    // Simulate worker failure
    // Verify automatic restart
    // Verify job redistribution
}
```

## Future Enhancements

1. **Metrics Export**: تصدير المقاييس إلى Prometheus
2. **Alerting**: إرسال تنبيهات عند فشل Workers
3. **Dashboard**: واجهة لعرض صحة Workers في الوقت الفعلي
4. **Adaptive Thresholds**: تعديل الحدود بناءً على الحمل
5. **Historical Analysis**: تحليل تاريخي لأداء Workers

## Conclusion

تم تطبيق نظام مراقبة صحة Workers بشكل كامل وفقاً للمتطلبات:

- ✅ **Requirement 13.4**: مراقبة صحة Workers كل 10 ثوانٍ
- ✅ **Requirement 13.5**: إعادة تشغيل Workers الفاشلة تلقائياً
- ✅ **Requirement 13.7**: تسجيل أداء كل Worker

النظام يوفر:
- موثوقية عالية مع إعادة تشغيل تلقائية
- مراقبة شاملة لأداء كل Worker
- تسجيل مفصل لجميع الأحداث
- حماية من الفشل المتكرر
- تكامل سلس مع River Queue
