# دليل تكامل Dead Letter Queue

## نظرة عامة

هذا الدليل يوضح كيفية تكامل Dead Letter Queue (DLQ) مع التطبيق الرئيسي.

## الخطوات المطلوبة

### 1. تسجيل DLQ API Routes

في `main.go`، بعد إنشاء Worker Pool:

```go
// إنشاء Worker Pool
workerPool, err := worker.NewRiverWorkerPool(
    workerConfig,
    db,
    logger,
    workerDependencies,
)
if err != nil {
    logger.Fatal("Failed to create worker pool", zap.Error(err))
}

// بدء Worker Pool
if err := workerPool.Start(ctx); err != nil {
    logger.Fatal("Failed to start worker pool", zap.Error(err))
}

// تسجيل DLQ API routes
dlqHandler := worker.NewDLQHandler(
    workerPool.GetDeadLetterQueue(),
    logger,
)
dlqHandler.RegisterRoutes(router)

logger.Info("DLQ API routes registered")
```

### 2. تشغيل Cleanup Scheduler

في `main.go`، بعد تسجيل routes:

```go
// إنشاء وتشغيل DLQ Cleanup Scheduler
dlqScheduler := worker.NewDLQScheduler(
    workerPool.GetDeadLetterQueue(),
    logger,
).WithRetentionDays(7).WithCleanupInterval(24 * time.Hour)

// تشغيل scheduler
dlqScheduler.Start(ctx)

logger.Info("DLQ cleanup scheduler started",
    zap.Int("retention_days", 7),
    zap.Duration("cleanup_interval", 24*time.Hour),
)

// إيقاف scheduler عند shutdown
defer func() {
    logger.Info("Stopping DLQ cleanup scheduler")
    dlqScheduler.Stop()
}()
```

### 3. مثال كامل للتكامل

```go
package main

import (
    "context"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/al-hudhud/backend/internal/database"
    "github.com/al-hudhud/backend/internal/worker"
    "github.com/gorilla/mux"
    "go.uber.org/zap"
)

func main() {
    // إنشاء logger
    logger, _ := zap.NewProduction()
    defer logger.Sync()

    // إنشاء context مع cancellation
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // الاتصال بقاعدة البيانات
    db, err := database.NewDB(ctx, &database.Config{
        Host:     os.Getenv("DB_HOST"),
        Port:     5432,
        User:     os.Getenv("DB_USER"),
        Password: os.Getenv("DB_PASSWORD"),
        Database: os.Getenv("DB_NAME"),
    })
    if err != nil {
        logger.Fatal("Failed to connect to database", zap.Error(err))
    }
    defer db.Close()

    // إنشاء router
    router := mux.NewRouter()

    // إنشاء Worker Pool dependencies
    workerDependencies := &worker.MessageWorkerDependencies{
        BalanceManager: balanceManager,
        RoutingEngine:  routingEngine,
        WebSocketHub:   wsHub,
        Logger:         logger,
    }

    // إنشاء Worker Pool
    workerPool, err := worker.NewRiverWorkerPool(
        worker.DefaultWorkerPoolConfig(),
        db,
        logger,
        workerDependencies,
    )
    if err != nil {
        logger.Fatal("Failed to create worker pool", zap.Error(err))
    }

    // بدء Worker Pool
    if err := workerPool.Start(ctx); err != nil {
        logger.Fatal("Failed to start worker pool", zap.Error(err))
    }
    defer workerPool.Stop(ctx)

    // تسجيل DLQ API routes
    dlqHandler := worker.NewDLQHandler(
        workerPool.GetDeadLetterQueue(),
        logger,
    )
    dlqHandler.RegisterRoutes(router)
    logger.Info("DLQ API routes registered")

    // إنشاء وتشغيل DLQ Cleanup Scheduler
    dlqScheduler := worker.NewDLQScheduler(
        workerPool.GetDeadLetterQueue(),
        logger,
    ).WithRetentionDays(7).WithCleanupInterval(24 * time.Hour)

    dlqScheduler.Start(ctx)
    logger.Info("DLQ cleanup scheduler started")
    defer dlqScheduler.Stop()

    // إنشاء HTTP server
    srv := &http.Server{
        Addr:         ":8080",
        Handler:      router,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    // بدء server في goroutine منفصلة
    go func() {
        logger.Info("Starting HTTP server", zap.String("addr", srv.Addr))
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Fatal("Failed to start server", zap.Error(err))
        }
    }()

    // انتظار إشارة shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    logger.Info("Shutting down server...")

    // إيقاف server بشكل graceful
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer shutdownCancel()

    if err := srv.Shutdown(shutdownCtx); err != nil {
        logger.Error("Server forced to shutdown", zap.Error(err))
    }

    logger.Info("Server exited")
}
```

## API Endpoints المتاحة

بعد التكامل، ستكون الـ endpoints التالية متاحة:

### 1. استرجاع رسائل DLQ
```
GET /api/dlq/{tenant_id}?limit=100
```

### 2. إعادة معالجة رسالة واحدة
```
POST /api/dlq/{tenant_id}/reprocess/{dlq_id}
```

### 3. إعادة معالجة جميع الرسائل
```
POST /api/dlq/{tenant_id}/reprocess-all
```

### 4. تنظيف يدوي
```
POST /api/dlq/cleanup
Content-Type: application/json

{
  "retention_days": 7
}
```

## التكوين (Configuration)

### متغيرات البيئة

يمكن تخصيص DLQ عبر متغيرات البيئة:

```bash
# فترة الاحتفاظ بالرسائل (بالأيام)
DLQ_RETENTION_DAYS=7

# فترة تشغيل cleanup (بالساعات)
DLQ_CLEANUP_INTERVAL_HOURS=24
```

### استخدام المتغيرات في الكود

```go
retentionDays := 7
if envDays := os.Getenv("DLQ_RETENTION_DAYS"); envDays != "" {
    if days, err := strconv.Atoi(envDays); err == nil {
        retentionDays = days
    }
}

cleanupHours := 24
if envHours := os.Getenv("DLQ_CLEANUP_INTERVAL_HOURS"); envHours != "" {
    if hours, err := strconv.Atoi(envHours); err == nil {
        cleanupHours = hours
    }
}

dlqScheduler := worker.NewDLQScheduler(
    workerPool.GetDeadLetterQueue(),
    logger,
).WithRetentionDays(retentionDays).
  WithCleanupInterval(time.Duration(cleanupHours) * time.Hour)
```

## المراقبة والـ Logging

### Logs المتوقعة

عند التشغيل الطبيعي:

```
INFO  DLQ API routes registered
INFO  DLQ cleanup scheduler started  retention_days=7  cleanup_interval=24h0m0s
INFO  Running DLQ cleanup  retention_days=7
INFO  DLQ cleanup completed  deleted_count=5  duration=123ms  retention_days=7
```

عند نقل رسالة إلى DLQ:

```
INFO  Message moved to dead letter queue  tenant_id=xxx  message_id=yyy  reason="Failed after 3 attempts"  retry_count=3
```

عند إعادة معالجة:

```
INFO  Message reprocessed from dead letter queue  tenant_id=xxx  message_id=yyy  dlq_id=123
```

### Metrics (اختياري)

يمكن إضافة metrics لمراقبة DLQ:

```go
// عدد الرسائل في DLQ
dlqSizeGauge := prometheus.NewGaugeVec(
    prometheus.GaugeOpts{
        Name: "dlq_messages_total",
        Help: "Total number of messages in dead letter queue",
    },
    []string{"tenant_id"},
)

// معدل إعادة المعالجة
dlqReprocessCounter := prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "dlq_reprocess_total",
        Help: "Total number of messages reprocessed from DLQ",
    },
    []string{"tenant_id", "status"},
)

// معدل التنظيف
dlqCleanupCounter := prometheus.NewCounter(
    prometheus.CounterOpts{
        Name: "dlq_cleanup_total",
        Help: "Total number of messages cleaned up from DLQ",
    },
)
```

## الأمان

### Authentication & Authorization

يُنصح بإضافة middleware للتحقق من الصلاحيات:

```go
// Middleware للتحقق من tenant_id
func TenantAuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        vars := mux.Vars(r)
        tenantID := vars["tenant_id"]
        
        // التحقق من أن المستخدم لديه صلاحية الوصول لهذا tenant
        if !hasAccess(r.Context(), tenantID) {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}

// تطبيق middleware على DLQ routes
dlqRouter := router.PathPrefix("/api/dlq").Subrouter()
dlqRouter.Use(TenantAuthMiddleware)
dlqHandler.RegisterRoutes(dlqRouter)
```

### Rate Limiting

يُنصح بإضافة rate limiting على DLQ endpoints:

```go
// Rate limiter لـ DLQ operations
dlqLimiter := rate.NewLimiter(rate.Every(time.Second), 10) // 10 requests/second

func DLQRateLimitMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !dlqLimiter.Allow() {
            http.Error(w, "Too many requests", http.StatusTooManyRequests)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

## استكشاف الأخطاء

### مشكلة: Scheduler لا يعمل

**الحل:**
- تأكد من استدعاء `scheduler.Start(ctx)`
- تأكد من أن context لم يتم إلغاؤه
- تحقق من logs للأخطاء

### مشكلة: لا يمكن إعادة معالجة الرسائل

**الحل:**
- تأكد من أن tenant_id صحيح
- تأكد من أن dlq_id موجود
- تحقق من أن الرسالة الأصلية موجودة في جدول messages

### مشكلة: Cleanup يحذف رسائل حديثة

**الحل:**
- تحقق من قيمة `retention_days`
- تأكد من أن `moved_at` timestamp صحيح
- راجع logs لمعرفة عدد الرسائل المحذوفة

## الخلاصة

بعد اتباع هذا الدليل، سيكون DLQ متكاملاً بالكامل مع التطبيق ويعمل بشكل تلقائي لإدارة الرسائل الفاشلة.
