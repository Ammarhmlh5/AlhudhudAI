# دليل التكامل - Error Handling System

هذا الدليل يوضح كيفية دمج نظام Error Handling الموحد مع المكونات الموجودة في النظام.

## المكونات المطلوب تحديثها

1. ✅ Balance Manager
2. ✅ API Gateway
3. ✅ Routing Engine
4. ✅ WebSocket Hub
5. ✅ Status Report Handler
6. ✅ Batch Processor
7. ✅ Worker Pool

## 1. تحديث Balance Manager

### قبل التحديث

```go
func (bm *BalanceManager) DeductBalance(ctx context.Context, tenantID int64, amount decimal.Decimal) error {
    // Old error handling
    if balance < amount {
        return fmt.Errorf("insufficient balance")
    }
    return nil
}
```

### بعد التحديث

```go
import apperrors "github.com/al-hudhud/backend/internal/errors"

func (bm *BalanceManager) DeductBalance(ctx context.Context, tenantID int64, amount decimal.Decimal) error {
    // Get current balance
    balance, err := bm.getBalance(ctx, tenantID)
    if err != nil {
        return apperrors.Wrap(err, 
            apperrors.CodeDatabaseQuery,
            apperrors.CategoryDatabase,
            "Failed to get balance",
        ).WithDetails("tenant_id", tenantID)
    }

    // Check sufficient balance
    if balance.LessThan(amount) {
        return apperrors.ErrInsufficientBalance.
            WithDetails("tenant_id", tenantID).
            WithDetails("balance", balance).
            WithDetails("amount", amount)
    }

    // Deduct with retry
    config := apperrors.DefaultRetryConfig()
    return apperrors.Retry(ctx, config, func() error {
        return bm.atomicDeduct(ctx, tenantID, amount)
    })
}
```

## 2. تحديث API Gateway

### قبل التحديث

```go
func (api *APIGateway) HandleSendMessage(w http.ResponseWriter, r *http.Request) {
    if err := validate(req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
}
```

### بعد التحديث

```go
import apperrors "github.com/al-hudhud/backend/internal/errors"

func (api *APIGateway) HandleSendMessage(w http.ResponseWriter, r *http.Request) {
    var req MessageRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        apperrors.WriteJSONError(w, apperrors.Wrap(err,
            apperrors.CodeInvalidRequest,
            apperrors.CategoryValidation,
            "Failed to decode request",
        ))
        return
    }

    // Validate
    if err := api.validator.Validate(req); err != nil {
        apperrors.WriteJSONError(w, err)
        return
    }

    // Check rate limit
    if err := api.rateLimiter.Check(r.Context(), req.TenantID); err != nil {
        apperrors.WriteJSONError(w, apperrors.ErrRateLimitExceeded.
            WithDetails("tenant_id", req.TenantID))
        return
    }

    // Process message
    resp, err := api.processMessage(r.Context(), req)
    if err != nil {
        api.errLogger.LogError(r.Context(), err, "Failed to process message",
            "tenant_id", req.TenantID)
        apperrors.WriteJSONError(w, err)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(resp)
}
```

## 3. تحديث Routing Engine

### قبل التحديث

```go
func (re *RoutingEngine) SelectChannel(ctx context.Context, tenantID int64) (*Channel, error) {
    channels, err := re.getChannels(ctx, tenantID)
    if err != nil {
        return nil, err
    }
    
    if len(channels) == 0 {
        return nil, fmt.Errorf("no channel available")
    }
    
    return channels[0], nil
}
```

### بعد التحديث

```go
import apperrors "github.com/al-hudhud/backend/internal/errors"

func (re *RoutingEngine) SelectChannel(ctx context.Context, tenantID int64) (*Channel, error) {
    // Try cache first
    cacheKey := fmt.Sprintf("channels:cache:%d", tenantID)
    channels, err := re.cache.Get(ctx, cacheKey)
    
    if err != nil {
        // Cache miss - query database with retry
        config := apperrors.DefaultRetryConfig()
        channels, err = apperrors.RetryWithResult(ctx, config, func() ([]*Channel, error) {
            return re.db.GetChannels(ctx, tenantID)
        })
        
        if err != nil {
            return nil, apperrors.Wrap(err,
                apperrors.CodeDatabaseQuery,
                apperrors.CategoryDatabase,
                "Failed to get channels",
            ).WithDetails("tenant_id", tenantID)
        }
        
        // Cache result
        re.cache.Set(ctx, cacheKey, channels, 5*time.Minute)
    }
    
    // Check if channels available
    if len(channels) == 0 {
        return nil, apperrors.ErrNoChannelAvailable.
            WithDetails("tenant_id", tenantID)
    }
    
    // Select best channel
    return re.selectBest(channels), nil
}
```

## 4. تحديث WebSocket Hub

### قبل التحديث

```go
func (hub *WebSocketHub) SendMessage(ctx context.Context, msg WebSocketMessage) error {
    select {
    case hub.sendChan <- msg:
        return nil
    case <-time.After(100 * time.Millisecond):
        return fmt.Errorf("send timeout")
    }
}
```

### بعد التحديث

```go
import apperrors "github.com/al-hudhud/backend/internal/errors"

func (hub *WebSocketHub) SendMessage(ctx context.Context, msg WebSocketMessage) error {
    // Use circuit breaker for protection
    return hub.circuitBreaker.Call(ctx, func() error {
        select {
        case hub.sendChan <- msg:
            return nil
        case <-time.After(100 * time.Millisecond):
            return apperrors.ErrSendTimeout.
                WithDetails("message_id", msg.MessageID).
                WithDetails("tenant_id", msg.TenantID)
        case <-ctx.Done():
            return apperrors.Wrap(ctx.Err(),
                apperrors.CodeInternalError,
                apperrors.CategorySystem,
                "Context cancelled",
            )
        }
    })
}

// Initialize circuit breaker in constructor
func NewWebSocketHub() *WebSocketHub {
    return &WebSocketHub{
        circuitBreaker: apperrors.NewCircuitBreaker(3, 5*time.Second),
        sendChan:       make(chan WebSocketMessage, 1000),
    }
}
```

## 5. تحديث Status Report Handler

### قبل التحديث

```go
func (sh *StatusReportHandler) ProcessReport(ctx context.Context, report StatusReport) error {
    // Check idempotency
    exists, _ := sh.redis.Exists(ctx, key)
    if exists {
        return nil // Already processed
    }
    
    // Process report
    return sh.updateStatus(ctx, report)
}
```

### بعد التحديث

```go
import apperrors "github.com/al-hudhud/backend/internal/errors"

func (sh *StatusReportHandler) ProcessReport(ctx context.Context, report StatusReport) error {
    // Generate idempotency key
    key := fmt.Sprintf("%d:%s:%s", report.TenantID, report.MessageID, report.Status)
    idempotencyKey := fmt.Sprintf("idempotency:%s", key)
    
    // Check idempotency
    exists, err := sh.redis.Exists(ctx, idempotencyKey)
    if err != nil {
        return apperrors.Wrap(err,
            apperrors.CodeDatabaseQuery,
            apperrors.CategoryDatabase,
            "Failed to check idempotency",
        ).WithDetails("key", idempotencyKey)
    }
    
    if exists {
        return apperrors.ErrDuplicateReport.
            WithDetails("message_id", report.MessageID).
            WithDetails("status", report.Status)
    }
    
    // Set idempotency key
    if err := sh.redis.Set(ctx, idempotencyKey, "1", 24*time.Hour); err != nil {
        return apperrors.Wrap(err,
            apperrors.CodeDatabaseQuery,
            apperrors.CategoryDatabase,
            "Failed to set idempotency",
        ).WithDetails("key", idempotencyKey)
    }
    
    // Process with retry
    config := apperrors.DefaultRetryConfig()
    return apperrors.Retry(ctx, config, func() error {
        return sh.processReportInternal(ctx, report)
    })
}
```

## 6. تحديث Batch Processor

### قبل التحديث

```go
func (bp *BatchProcessor) ProcessBatch(ctx context.Context, messages []Message) error {
    if err := bp.db.BatchInsert(ctx, messages); err != nil {
        return err
    }
    return nil
}
```

### بعد التحديث

```go
import apperrors "github.com/al-hudhud/backend/internal/errors"

func (bp *BatchProcessor) ProcessBatch(ctx context.Context, messages []Message) error {
    // Try batch insert with retry
    config := apperrors.DefaultRetryConfig()
    err := apperrors.Retry(ctx, config, func() error {
        return bp.db.BatchInsert(ctx, messages)
    })
    
    if err == nil {
        return nil
    }
    
    // Log batch failure
    bp.errLogger.LogError(ctx, err, "Batch insert failed, falling back to individual inserts",
        "batch_size", len(messages))
    
    // Fallback to individual inserts
    var failedMessages []Message
    for _, msg := range messages {
        if err := bp.db.InsertSingle(ctx, msg); err != nil {
            bp.errLogger.LogError(ctx, err, "Individual insert failed",
                "message_id", msg.MessageID,
                "tenant_id", msg.TenantID)
            failedMessages = append(failedMessages, msg)
        }
    }
    
    if len(failedMessages) > 0 {
        return apperrors.New(
            apperrors.CodeDatabaseQuery,
            apperrors.CategoryDatabase,
            fmt.Sprintf("Failed to insert %d messages", len(failedMessages)),
        ).WithDetails("failed_count", len(failedMessages))
    }
    
    return nil
}
```

## 7. تحديث Worker Pool

### قبل التحديث

```go
func (w *Worker) Work(ctx context.Context, job *river.Job) error {
    // Process job
    if err := w.processMessage(ctx, job); err != nil {
        return err
    }
    return nil
}
```

### بعد التحديث

```go
import apperrors "github.com/al-hudhud/backend/internal/errors"

func (w *Worker) Work(ctx context.Context, job *river.Job) error {
    // Parse job data
    var msg Message
    if err := json.Unmarshal(job.Args, &msg); err != nil {
        return apperrors.Wrap(err,
            apperrors.CodeInvalidRequest,
            apperrors.CategoryValidation,
            "Failed to parse job data",
        ).WithDetails("job_id", job.ID)
    }
    
    // Deduct balance with retry
    err := w.balanceManager.DeductBalance(ctx, msg.TenantID, msg.Cost)
    if err != nil {
        var appErr *apperrors.AppError
        if errors.As(err, &appErr) {
            switch appErr.Code {
            case apperrors.CodeInsufficientBalance:
                // Don't retry - update message status
                w.updateMessageStatus(ctx, msg.ID, "FAILED", "Insufficient balance")
                return nil // Job completed (failed message)
            }
        }
        
        // Log and return for retry
        w.errLogger.LogError(ctx, err, "Failed to deduct balance",
            "message_id", msg.ID,
            "tenant_id", msg.TenantID)
        return err
    }
    
    // Select channel
    channel, err := w.routingEngine.SelectChannel(ctx, msg.TenantID)
    if err != nil {
        var appErr *apperrors.AppError
        if errors.As(err, &appErr) {
            switch appErr.Code {
            case apperrors.CodeNoChannelAvailable:
                // Update status to WAITING
                w.updateMessageStatus(ctx, msg.ID, "WAITING", "No channel available")
                return nil // Job completed (waiting for channel)
            }
        }
        
        w.errLogger.LogError(ctx, err, "Failed to select channel",
            "message_id", msg.ID,
            "tenant_id", msg.TenantID)
        return err
    }
    
    // Send message
    if err := w.wsHub.SendMessage(ctx, msg); err != nil {
        w.errLogger.LogError(ctx, err, "Failed to send message",
            "message_id", msg.ID,
            "tenant_id", msg.TenantID,
            "channel_id", channel.ID)
        return err
    }
    
    return nil
}
```

## خطوات التكامل

### 1. إضافة Import

في كل ملف تريد استخدام Error Handling فيه:

```go
import apperrors "github.com/al-hudhud/backend/internal/errors"
```

### 2. إنشاء Error Logger

في main.go أو في constructor المكون:

```go
logger := slog.Default()
errLogger := apperrors.NewLogger(logger)
```

### 3. استبدال الأخطاء القديمة

ابحث عن:
- `fmt.Errorf()`
- `errors.New()`
- `return err`

واستبدلها بـ:
- `apperrors.New()` أو `apperrors.Wrap()`
- إضافة Details مفيدة
- استخدام الأخطاء المعرفة مسبقاً

### 4. إضافة Logging

في كل مكان تحدث فيه أخطاء:

```go
if err != nil {
    errLogger.LogError(ctx, err, "Operation failed",
        "key1", value1,
        "key2", value2,
    )
    return err
}
```

### 5. إضافة Retry للعمليات المؤقتة

للعمليات التي قد تفشل مؤقتاً:

```go
config := apperrors.DefaultRetryConfig()
err := apperrors.Retry(ctx, config, func() error {
    return operation()
})
```

### 6. إضافة Circuit Breaker للخدمات الخارجية

```go
cb := apperrors.NewCircuitBreaker(3, 5*time.Second)
err := cb.Call(ctx, func() error {
    return externalService.Call()
})
```

## الاختبار

بعد التكامل، تأكد من:

1. ✅ جميع الأخطاء تُسجل مع Stack Trace
2. ✅ HTTP Responses تحتوي على Error Code و Message واضحة
3. ✅ Retry يعمل للأخطاء المؤقتة
4. ✅ Circuit Breaker يحمي الخدمات الخارجية
5. ✅ Error Details تحتوي على معلومات مفيدة للتتبع

## الأولويات

1. **عالية**: Balance Manager, API Gateway (أخطاء مالية حرجة)
2. **متوسطة**: Status Report Handler, Worker Pool
3. **منخفضة**: Routing Engine, WebSocket Hub, Batch Processor

## الدعم

للمزيد من المعلومات:
- [README.md](./README.md)
- [Design Document](.kiro/specs/message-lifecycle-rebuild/design.md)
