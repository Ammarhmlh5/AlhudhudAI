# Error Handling Package

نظام موحد لمعالجة الأخطاء في جميع مكونات النظام مع دعم Stack Trace والتسجيل المفصل.

## المحتويات

- [نظرة عامة](#نظرة-عامة)
- [أنواع الأخطاء](#أنواع-الأخطاء)
- [Error Response Format](#error-response-format)
- [Stack Trace Logging](#stack-trace-logging)
- [Retry Mechanism](#retry-mechanism)
- [Circuit Breaker](#circuit-breaker)
- [أمثلة الاستخدام](#أمثلة-الاستخدام)

## نظرة عامة

هذا المكون يوفر نظام معالجة أخطاء موحد يتضمن:

- ✅ تعريف Error Types واضحة لكل نوع خطأ
- ✅ Error Response Format موحد
- ✅ Stack Trace logging تلقائي
- ✅ Retry mechanism مع Exponential Backoff
- ✅ Circuit Breaker pattern
- ✅ تصنيف الأخطاء (Validation, Database, Network, Business, System)
- ✅ HTTP Status Code mapping تلقائي

## أنواع الأخطاء

### Error Categories

```go
const (
    CategoryValidation = "VALIDATION"  // أخطاء التحقق من البيانات
    CategoryDatabase   = "DATABASE"    // أخطاء قاعدة البيانات
    CategoryNetwork    = "NETWORK"     // أخطاء الشبكة
    CategoryBusiness   = "BUSINESS"    // أخطاء منطق الأعمال
    CategorySystem     = "SYSTEM"      // أخطاء النظام
    CategoryAuth       = "AUTH"        // أخطاء المصادقة
)
```

### Error Codes

#### Validation Errors (1xxx)
- `ERR_1001`: Invalid Request
- `ERR_1002`: Invalid Phone Number
- `ERR_1003`: Content Too Long
- `ERR_1004`: Invalid Tenant ID
- `ERR_1005`: Invalid Message ID
- `ERR_1006`: Invalid Status

#### Business Logic Errors (2xxx)
- `ERR_2001`: Insufficient Balance
- `ERR_2002`: Tenant Not Found
- `ERR_2003`: Tenant Suspended
- `ERR_2004`: Message Not Found
- `ERR_2005`: Channel Not Found
- `ERR_2006`: No Channel Available
- `ERR_2007`: Channel Offline
- `ERR_2008`: Channel Busy
- `ERR_2009`: Duplicate Report
- `ERR_2010`: Rate Limit Exceeded

#### Database Errors (3xxx)
- `ERR_3001`: Database Connection
- `ERR_3002`: Database Query
- `ERR_3003`: Database Transaction
- `ERR_3004`: Concurrent Update
- `ERR_3005`: Deadlock

#### Network Errors (4xxx)
- `ERR_4001`: Connection Closed
- `ERR_4002`: Connection Timeout
- `ERR_4003`: Send Timeout
- `ERR_4004`: Buffer Full

#### System Errors (5xxx)
- `ERR_5001`: Internal Error
- `ERR_5002`: Service Unavailable
- `ERR_5003`: Circuit Breaker Open

## Error Response Format

### AppError Structure

```go
type AppError struct {
    Code       string                 `json:"code"`
    Category   string                 `json:"category"`
    Message    string                 `json:"message"`
    Details    map[string]interface{} `json:"details,omitempty"`
    Err        error                  `json:"-"`
    StackTrace string                 `json:"stack_trace,omitempty"`
}
```

### HTTP Response Format

```json
{
  "error": "VALIDATION",
  "code": "ERR_1002",
  "message": "Invalid phone number format",
  "details": {
    "field": "phone_number",
    "value": "123"
  }
}
```

### HTTP Status Code Mapping

| Error Category | HTTP Status Code |
|---------------|------------------|
| Validation | 400 Bad Request |
| Insufficient Balance | 402 Payment Required |
| Not Found | 404 Not Found |
| Forbidden | 403 Forbidden |
| Rate Limit | 429 Too Many Requests |
| Conflict | 409 Conflict |
| Database | 500 Internal Server Error |
| Network | 503 Service Unavailable |
| System | 500 Internal Server Error |

## Stack Trace Logging

### تسجيل الأخطاء مع Stack Trace

```go
import (
    "context"
    "log/slog"
    apperrors "backend/internal/errors"
)

func main() {
    logger := slog.Default()
    errLogger := apperrors.NewLogger(logger)
    
    ctx := context.Background()
    
    // تسجيل خطأ مع stack trace
    err := someFunction()
    if err != nil {
        errLogger.LogError(ctx, err, "Failed to process message",
            "message_id", "123",
            "tenant_id", 456,
        )
    }
}
```

### تسجيل خطأ حرج

```go
// للأخطاء الحرجة التي تحتاج انتباه فوري
errLogger.LogCritical(ctx, err, "Critical database failure",
    "database", "postgres",
    "operation", "balance_deduction",
)
```

## Retry Mechanism

### استخدام Retry مع Exponential Backoff

```go
import (
    "context"
    apperrors "backend/internal/errors"
)

func deductBalance(ctx context.Context, tenantID int64, amount decimal.Decimal) error {
    config := apperrors.DefaultRetryConfig()
    
    return apperrors.Retry(ctx, config, func() error {
        return db.DeductBalance(ctx, tenantID, amount)
    })
}
```

### Retry مع نتيجة

```go
func getChannel(ctx context.Context, tenantID int64) (*Channel, error) {
    config := apperrors.RetryConfig{
        MaxRetries:        3,
        InitialBackoff:    100 * time.Millisecond,
        MaxBackoff:        2 * time.Second,
        BackoffMultiplier: 2.0,
    }
    
    return apperrors.RetryWithResult(ctx, config, func() (*Channel, error) {
        return db.SelectChannel(ctx, tenantID)
    })
}
```

### الأخطاء القابلة لإعادة المحاولة

الأخطاء التالية يتم إعادة محاولتها تلقائياً:
- `ERR_3004`: Concurrent Update
- `ERR_3005`: Deadlock
- `ERR_4002`: Connection Timeout
- `ERR_4003`: Send Timeout
- `ERR_5002`: Service Unavailable

## Circuit Breaker

### استخدام Circuit Breaker

```go
import (
    "context"
    "time"
    apperrors "backend/internal/errors"
)

// إنشاء circuit breaker
cb := apperrors.NewCircuitBreaker(
    3,              // maxFailures
    5*time.Second,  // timeout
)

// استخدام circuit breaker
func sendMessage(ctx context.Context, msg Message) error {
    return cb.Call(ctx, func() error {
        return ws.Send(msg)
    })
}
```

### Circuit Breaker مع نتيجة

```go
func fetchData(ctx context.Context) (interface{}, error) {
    return cb.CallWithResult(ctx, func() (interface{}, error) {
        return api.FetchData()
    })
}
```

### فحص حالة Circuit Breaker

```go
if cb.IsOpen() {
    log.Warn("Circuit breaker is open, service unavailable")
}

if cb.IsClosed() {
    log.Info("Circuit breaker is closed, service available")
}

state := cb.GetState() // CLOSED, OPEN, HALF_OPEN
failures := cb.GetFailures()
```

## أمثلة الاستخدام

### 1. إنشاء خطأ جديد

```go
import apperrors "backend/internal/errors"

// خطأ بسيط
err := apperrors.New(
    apperrors.CodeInvalidRequest,
    apperrors.CategoryValidation,
    "Invalid phone number format",
)

// خطأ مع تفاصيل
err = apperrors.ErrInvalidPhoneNumber.
    WithDetails("field", "phone_number").
    WithDetails("value", phoneNumber)
```

### 2. تغليف خطأ موجود

```go
dbErr := db.Query(ctx, sql)
if dbErr != nil {
    return apperrors.Wrap(
        dbErr,
        apperrors.CodeDatabaseQuery,
        apperrors.CategoryDatabase,
        "Failed to query messages",
    ).WithDetails("tenant_id", tenantID)
}
```

### 3. معالجة الأخطاء في API Handler

```go
import (
    "net/http"
    apperrors "backend/internal/errors"
)

func handleSendMessage(w http.ResponseWriter, r *http.Request) {
    err := sendMessage(r.Context(), msg)
    if err != nil {
        // تسجيل الخطأ
        errLogger.LogError(r.Context(), err, "Failed to send message")
        
        // إرجاع استجابة JSON
        apperrors.WriteJSONError(w, err)
        return
    }
    
    // Success response
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(response)
}
```

### 4. استخدام الأخطاء المعرفة مسبقاً

```go
// التحقق من الرصيد
if balance < cost {
    return apperrors.ErrInsufficientBalance.
        WithDetails("balance", balance).
        WithDetails("cost", cost).
        WithDetails("tenant_id", tenantID)
}

// قناة غير متاحة
if channel == nil {
    return apperrors.ErrNoChannelAvailable.
        WithDetails("tenant_id", tenantID)
}

// تقرير مكرر
if exists {
    return apperrors.ErrDuplicateReport.
        WithDetails("message_id", messageID).
        WithDetails("status", status)
}
```

### 5. فحص نوع الخطأ

```go
import "errors"

err := someOperation()
if err != nil {
    var appErr *apperrors.AppError
    if errors.As(err, &appErr) {
        switch appErr.Code {
        case apperrors.CodeInsufficientBalance:
            // معالجة خاصة لنقص الرصيد
            notifyLowBalance(tenantID)
        case apperrors.CodeNoChannelAvailable:
            // معالجة خاصة لعدم توفر قنوات
            updateMessageStatus(messageID, "WAITING")
        }
    }
}
```

### 6. التحقق من قابلية إعادة المحاولة

```go
err := db.Update(ctx, data)
if err != nil {
    if apperrors.IsRetryable(err) {
        // إعادة المحاولة
        time.Sleep(100 * time.Millisecond)
        err = db.Update(ctx, data)
    }
}
```

### 7. التحقق من الأخطاء الحرجة

```go
err := processMessage(ctx, msg)
if err != nil {
    if apperrors.IsCritical(err) {
        // إرسال تنبيه فوري
        alerting.SendCriticalAlert(err)
    }
    
    errLogger.LogError(ctx, err, "Failed to process message")
}
```

## Best Practices

### 1. استخدم الأخطاء المعرفة مسبقاً

```go
// ✅ جيد
return apperrors.ErrInsufficientBalance

// ❌ سيء
return errors.New("insufficient balance")
```

### 2. أضف تفاصيل مفيدة

```go
// ✅ جيد
return apperrors.ErrMessageNotFound.
    WithDetails("message_id", messageID).
    WithDetails("tenant_id", tenantID)

// ❌ سيء
return apperrors.ErrMessageNotFound
```

### 3. سجل الأخطاء مع السياق

```go
// ✅ جيد
errLogger.LogError(ctx, err, "Failed to deduct balance",
    "tenant_id", tenantID,
    "amount", amount,
    "balance", currentBalance,
)

// ❌ سيء
log.Println("Error:", err)
```

### 4. استخدم Retry للأخطاء المؤقتة

```go
// ✅ جيد
err := apperrors.Retry(ctx, config, func() error {
    return db.Update(ctx, data)
})

// ❌ سيء
err := db.Update(ctx, data)
if err != nil {
    return err // لا إعادة محاولة
}
```

### 5. استخدم Circuit Breaker للخدمات الخارجية

```go
// ✅ جيد
err := cb.Call(ctx, func() error {
    return externalAPI.Call()
})

// ❌ سيء
err := externalAPI.Call() // بدون حماية
```

## الاختبارات

### تشغيل الاختبارات

```bash
cd services/backend/internal/errors
go test -v
```

### تغطية الاختبارات

```bash
go test -cover
```

## المتطلبات المحققة

- ✅ **المتطلب 19.1**: تسجيل الأخطاء بالتفصيل مع Stack Trace
- ✅ **المتطلب 19.2**: إعادة محاولة العمليات الفاشلة مع Exponential Backoff
- ✅ **المتطلب 19.3**: استخدام Circuit Breaker لحماية الخدمات الخارجية
- ✅ **المتطلب 19.7**: تسجيل Stack Trace كامل للأخطاء غير المتوقعة

## الملفات

```
internal/errors/
├── types.go              # تعريف Error Types والأخطاء المعرفة مسبقاً
├── response.go           # Error Response Format و HTTP mapping
├── logger.go             # Stack Trace logging
├── retry.go              # Retry mechanism مع Exponential Backoff
├── circuit_breaker.go    # Circuit Breaker pattern
├── types_test.go         # اختبارات Error Types
├── circuit_breaker_test.go # اختبارات Circuit Breaker
└── README.md             # هذا الملف
```

## الخطوات التالية

1. ✅ تطبيق Error Handling في Balance Manager
2. ✅ تطبيق Error Handling في API Gateway
3. ✅ تطبيق Error Handling في Routing Engine
4. ✅ تطبيق Error Handling في WebSocket Hub
5. ✅ تطبيق Error Handling في Status Report Handler
6. ✅ تطبيق Error Handling في Batch Processor
7. ✅ تطبيق Error Handling في Worker Pool

## الدعم

للمزيد من المعلومات، راجع:
- [Design Document](.kiro/specs/message-lifecycle-rebuild/design.md)
- [Requirements Document](.kiro/specs/message-lifecycle-rebuild/requirements.md)


## Fallback Strategies

### نظرة عامة

استراتيجيات Fallback توفر آلية تلقائية للتحول لتطبيقات بديلة عند فشل الخدمات الأساسية. تعمل جنباً إلى جنب مع Circuit Breakers لضمان استمرارية العمل.

### الاستراتيجيات المتاحة

#### 1. Redis Fallback Strategy

التحول للمعالجة المباشرة من قاعدة البيانات عند فشل Redis:

```go
redisFallback := apperrors.NewRedisFallbackStrategy(redisCircuitBreaker)

// Cache lookup with database fallback
channels, err := redisFallback.ExecuteWithResult(
    ctx,
    // Primary: Try Redis cache
    func() (interface{}, error) {
        return redisClient.Get(ctx, cacheKey)
    },
    // Fallback: Query database
    func() (interface{}, error) {
        return db.Query(ctx, "SELECT * FROM channels WHERE tenant_id = $1", tenantID)
    },
)
```

**الفوائد:**
- ✅ النظام يستمر في العمل حتى عند فشل Redis
- ✅ الأداء قد يتأثر قليلاً لكن الوظيفة تبقى متاحة
- ✅ تسجيل تلقائي لاستخدام Fallback

#### 2. Database Fallback Strategy

التحول لقاعدة البيانات الاحتياطية عند فشل الرئيسية:

```go
dbFallback := apperrors.NewDatabaseFallbackStrategy(primaryCB, backupCB)

// Database operation with backup fallback
result, err := dbFallback.ExecuteWithResult(
    ctx,
    // Primary: Try primary database
    func() (interface{}, error) {
        return primaryDB.Exec(ctx, query, args...)
    },
    // Fallback: Use backup database
    func() (interface{}, error) {
        return backupDB.Exec(ctx, query, args...)
    },
)
```

**الفوائد:**
- ✅ استمرارية العمل حتى عند فشل قاعدة البيانات الرئيسية
- ✅ حماية من فقدان البيانات
- ✅ Uptime أعلى (99.9%+)

#### 3. WebSocket Fallback Strategy

تخزين الرسائل في Queue عند فشل WebSocket:

```go
wsFallback := apperrors.NewWebSocketFallbackStrategy(wsCircuitBreaker)

// WebSocket send with queue fallback
err := wsFallback.Execute(
    ctx,
    // Primary: Try WebSocket send
    func() error {
        return wsHub.SendMessage(ctx, message)
    },
    // Fallback: Queue message for later
    func() error {
        err := messageQueue.Enqueue(ctx, message)
        if err != nil {
            return err
        }
        return db.UpdateMessageStatus(ctx, message.ID, "WAITING")
    },
)
```

**الفوائد:**
- ✅ لا تُفقد الرسائل عند انقطاع الاتصال
- ✅ إعادة الإرسال التلقائي عند عودة الاتصال
- ✅ تجربة مستخدم أفضل

### مراقبة Fallback Metrics

```go
metrics := fallbackStrategy.GetMetrics()

log.Printf("Primary Success Rate: %.2f%%", metrics.GetPrimarySuccessRate())
log.Printf("Fallback Usage Rate: %.2f%%", metrics.GetFallbackUsageRate())
log.Printf("Fallback Success Rate: %.2f%%", metrics.GetFallbackSuccessRate())

// Alert if fallback usage is high
if metrics.GetFallbackUsageRate() > 50.0 {
    alertService.SendAlert("High fallback usage - primary service may be degraded")
}
```

### التكامل مع Circuit Breakers

```go
// Setup circuit breakers (من Task 13.3)
redisCircuitBreaker := apperrors.NewCircuitBreaker(3, 10*time.Second)
primaryDBCircuitBreaker := apperrors.NewCircuitBreaker(5, 30*time.Second)
backupDBCircuitBreaker := apperrors.NewCircuitBreaker(5, 30*time.Second)
wsCircuitBreaker := apperrors.NewCircuitBreaker(5, 30*time.Second)

// Setup fallback strategies (Task 13.4)
redisFallback := apperrors.NewRedisFallbackStrategy(redisCircuitBreaker)
dbFallback := apperrors.NewDatabaseFallbackStrategy(primaryDBCircuitBreaker, backupDBCircuitBreaker)
wsFallback := apperrors.NewWebSocketFallbackStrategy(wsCircuitBreaker)
```

### أمثلة إضافية

راجع `fallback_examples.go` لأمثلة شاملة تشمل:
- Redis Cache Miss → Database Query
- Redis Buffer Push → Direct Database Insert
- Primary Database → Backup Database
- WebSocket Send → Queue Message
- مراقبة المقاييس
- التعامل مع Circuit Breaker المفتوح

## المتطلبات المحققة (محدثة)

- ✅ **المتطلب 19.1**: تسجيل الأخطاء بالتفصيل مع Stack Trace
- ✅ **المتطلب 19.2**: إعادة محاولة العمليات الفاشلة مع Exponential Backoff
- ✅ **المتطلب 19.3**: استخدام Circuit Breaker لحماية الخدمات الخارجية
- ✅ **المتطلب 19.5**: التحول لمعالجة مباشرة عند فشل Redis
- ✅ **المتطلب 19.7**: تسجيل Stack Trace كامل للأخطاء غير المتوقعة
- ✅ **المتطلب 24.2**: التحول لمعالجة مباشرة من قاعدة البيانات عند فشل Redis
- ✅ **المتطلب 24.3**: التحول لقاعدة البيانات الاحتياطية عند فشل الرئيسية

## الملفات (محدثة)

```
internal/errors/
├── types.go                    # تعريف Error Types والأخطاء المعرفة مسبقاً
├── response.go                 # Error Response Format و HTTP mapping
├── logger.go                   # Stack Trace logging
├── retry.go                    # Retry mechanism مع Exponential Backoff
├── circuit_breaker.go          # Circuit Breaker pattern
├── fallback.go                 # Fallback Strategies (جديد)
├── types_test.go               # اختبارات Error Types
├── circuit_breaker_test.go     # اختبارات Circuit Breaker
├── fallback_test.go            # اختبارات Fallback Strategies (جديد)
├── circuit_breaker_examples.go # أمثلة Circuit Breaker
├── fallback_examples.go        # أمثلة Fallback Strategies (جديد)
└── README.md                   # هذا الملف
```

## الخطوات التالية (محدثة)

1. ✅ تطبيق Error Handling في Balance Manager
2. ✅ تطبيق Error Handling في API Gateway
3. ✅ تطبيق Error Handling في Routing Engine
4. ✅ تطبيق Error Handling في WebSocket Hub
5. ✅ تطبيق Error Handling في Status Report Handler
6. ✅ تطبيق Error Handling في Batch Processor
7. ✅ تطبيق Error Handling في Worker Pool
8. ✅ تطبيق Circuit Breaker للخدمات الخارجية (Task 13.3)
9. ✅ تطبيق Fallback Strategies (Task 13.4)
10. ⏳ إضافة Prometheus Metrics للـ Fallback
11. ⏳ إنشاء Grafana Dashboard للمراقبة
12. ⏳ كتابة Integration Tests

## الدعم (محدث)

للمزيد من المعلومات، راجع:
- [Design Document](.kiro/specs/message-lifecycle-rebuild/design.md)
- [Requirements Document](.kiro/specs/message-lifecycle-rebuild/requirements.md)
- [Task 13.3 Summary](.kiro/specs/message-lifecycle-rebuild/TASK_13.3_CIRCUIT_BREAKER_SUMMARY.md)
- [Task 13.4 Summary](.kiro/specs/message-lifecycle-rebuild/TASK_13.4_FALLBACK_STRATEGIES_SUMMARY.md)
