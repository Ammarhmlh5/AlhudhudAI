# API Gateway Package

## نظرة عامة

هذا الـ package مسؤول عن استقبال طلبات إرسال الرسائل من العملاء والتحقق من صحتها (Validation) قبل قبولها في النظام.

## المسؤوليات

1. **استقبال الطلبات**: استقبال طلبات HTTP POST لإرسال الرسائل
2. **التحقق من الصحة (Validation)**: التحقق من صحة البيانات المرسلة
3. **رفض الطلبات غير الصحيحة**: إرجاع HTTP 400 للطلبات غير الصحيحة مع رسائل خطأ واضحة
4. **إنشاء الرسائل**: تحويل الطلبات الصحيحة إلى رسائل جاهزة للمعالجة

## المكونات

### 1. Types (`types.go`)

يحتوي على تعريفات الأنواع (structs) المستخدمة:

- **MessageRequest**: طلب إرسال رسالة واحدة
- **MessageResponse**: استجابة API بعد قبول الرسالة
- **BulkMessageRequest**: طلب إرسال رسائل متعددة (حتى 10,000)
- **BulkResponse**: استجابة API للطلبات المجمعة
- **ValidationError**: تفاصيل خطأ التحقق من الصحة
- **ErrorResponse**: استجابة الخطأ
- **BufferedMessage**: رسالة جاهزة للتخزين في Buffer

### 2. Validator (`validator.go`)

يحتوي على منطق التحقق من الصحة:

#### قواعد التحقق من phone_number:
- **مطلوب**: لا يمكن أن يكون فارغاً
- **التنسيق**: يجب أن يكون بصيغة E.164 (`+[country code][number]`)
- **الطول**: 10-15 رقم (بدون علامة +)
- **أمثلة صحيحة**:
  - `+1234567890` (10 أرقام)
  - `+966501234567` (12 رقم)
  - `+123456789012345` (15 رقم)
- **أمثلة خاطئة**:
  - `1234567890` (بدون +)
  - `+123456789` (أقل من 10 أرقام)
  - `+1234567890123456` (أكثر من 15 رقم)
  - `+0123456789` (يبدأ بصفر)

#### قواعد التحقق من content:
- **مطلوب**: لا يمكن أن يكون فارغاً أو مسافات فقط
- **الطول الأقصى**: 1600 حرف
- **دعم Unicode**: يدعم جميع اللغات بما فيها العربية

#### قواعد التحقق من priority (اختياري):
- **القيم المسموحة**: `HIGH`, `NORMAL`, `LOW`
- **حساس لحالة الأحرف**: يجب أن تكون بأحرف كبيرة

#### قواعد التحقق من tenant_id:
- **مطلوب**: يجب أن يكون موجوداً
- **موجب**: يجب أن يكون أكبر من صفر

### 3. Rate Limiter (`rate_limiter.go`)

يحتوي على منطق Rate Limiting لكل Tenant باستخدام Redis:

#### المواصفات:
- **الحد الافتراضي**: 100 طلب/دقيقة لكل Tenant
- **النافذة الزمنية**: 1 دقيقة (60 ثانية)
- **التخزين**: Redis مع TTL تلقائي
- **العزل**: عدادات منفصلة تماماً لكل Tenant

#### الآلية:
1. عند كل طلب، يتم زيادة العداد في Redis: `rate_limit:{tenant_id}`
2. إذا كان الطلب الأول، يتم تعيين TTL = 60 ثانية
3. إذا تجاوز العداد 100، يتم رفض الطلب مع HTTP 429
4. يتم إرجاع headers للعميل:
   - `X-RateLimit-Limit`: الحد الأقصى (100)
   - `X-RateLimit-Remaining`: الطلبات المتبقية
   - `X-RateLimit-Reset`: وقت إعادة التعيين (Unix timestamp)

#### مثال الاستخدام:

```go
import (
    "github.com/redis/go-redis/v9"
    "path/to/gateway"
)

// Create Redis client
redisClient := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})

// Create rate limiter
rateLimiter := gateway.NewRateLimiter(redisClient)

// Check if request is allowed
allowed, info, err := rateLimiter.Allow(ctx, tenantID)
if err != nil {
    // Handle error
}

if !allowed {
    // Rate limit exceeded
    // info contains: Limit, Remaining, Reset
}
```

### 4. Handler (`handler.go`)

يحتوي على معالجات HTTP:

#### Endpoints:

##### POST /api/messages
إرسال رسالة واحدة

**Request Body:**
```json
{
  "tenant_id": 1,
  "phone_number": "+966501234567",
  "content": "مرحبا بك في نظام الرسائل",
  "priority": "NORMAL"
}
```

**Success Response (200 OK):**
```json
{
  "message_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "QUEUED",
  "queued_at": "2024-02-25T10:30:00Z"
}
```

**Response Headers:**
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1708857600
```

**Error Response (429 Too Many Requests):**
```json
{
  "error": "RATE_LIMIT_EXCEEDED",
  "message": "Too many requests. Please try again later.",
  "errors": null
}
```

**Error Response (400 Bad Request):**
```json
{
  "error": "VALIDATION_ERROR",
  "message": "Request validation failed",
  "errors": [
    {
      "field": "phone_number",
      "message": "phone_number must be in E.164 format (+[country code][number], 10-15 digits total)"
    },
    {
      "field": "content",
      "message": "content must not exceed 1600 characters (found 1650)"
    }
  ]
}
```

##### POST /api/messages/bulk
إرسال رسائل متعددة (حتى 10,000)

**Request Body:**
```json
{
  "tenant_id": 1,
  "messages": [
    {
      "tenant_id": 1,
      "phone_number": "+966501234567",
      "content": "رسالة 1"
    },
    {
      "tenant_id": 1,
      "phone_number": "+966501234568",
      "content": "رسالة 2"
    }
  ]
}
```

**Success Response (200 OK):**
```json
{
  "batch_id": "batch_20240225103000",
  "status": "PROCESSING",
  "total_count": 2,
  "accepted_count": 2,
  "rejected_count": 0,
  "queued_at": "2024-02-25T10:30:00Z"
}
```

## الاستخدام

### إنشاء Handler

```go
import (
    "github.com/redis/go-redis/v9"
    "path/to/gateway"
)

// Create Redis client
redisClient := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})

// Create rate limiter
rateLimiter := gateway.NewRateLimiter(redisClient)

// Create handler with rate limiter
handler := gateway.NewHandler(rateLimiter)

// Register routes
mux := http.NewServeMux()
handler.RegisterRoutes(mux)

// Start server
http.ListenAndServe(":8080", mux)
```

### استخدام Validator مباشرة

```go
import "path/to/gateway"

validator := gateway.NewValidator()

req := gateway.MessageRequest{
    TenantID:    1,
    PhoneNumber: "+966501234567",
    Content:     "مرحبا",
}

// Validate
errors := validator.ValidateMessageRequest(req)
if len(errors) > 0 {
    // Handle validation errors
    for _, err := range errors {
        fmt.Printf("Field: %s, Error: %s\n", err.Field, err.Message)
    }
}

// Or use IsValid for simple check
if validator.IsValid(req) {
    // Request is valid
}
```

## الاختبارات

### تشغيل الاختبارات

```bash
# Run all tests
go test ./internal/gateway/...

# Run with coverage
go test -cover ./internal/gateway/...

# Run with verbose output
go test -v ./internal/gateway/...
```

### تغطية الاختبارات

الاختبارات تغطي:
- ✅ التحقق من phone_number (10 حالات اختبار)
- ✅ التحقق من content (7 حالات اختبار)
- ✅ التحقق من priority (6 حالات اختبار)
- ✅ التحقق من MessageRequest الكامل (7 حالات اختبار)
- ✅ التحقق من BulkMessageRequest (4 حالات اختبار)
- ✅ Rate Limiter (6 حالات اختبار):
  - الطلب الأول
  - طلبات متعددة
  - تجاوز الحد
  - عزل Tenants
  - إعادة التعيين
  - تحويل Headers
- ✅ Handler مع Rate Limiting (2 حالات اختبار):
  - رفض الطلبات الزائدة
  - عزل Rate Limiting بين Tenants

## المتطلبات المحققة

هذا الـ package يحقق المتطلبات التالية من وثيقة المتطلبات:

- **المتطلب 3.4**: تطبيق Rate Limiting بحد 100 طلب/دقيقة لكل Tenant ✅
- **المتطلب 3.5**: رفض الطلبات الزائدة مع HTTP 429 Too Many Requests ✅
- **المتطلب 3.6**: التحقق من صحة البيانات (phone_number, content) قبل القبول ✅
- **المتطلب 3.7**: رفض الطلبات IF طول المحتوى يتجاوز 1600 حرف ✅

## الخطوات القادمة

1. ~~**Rate Limiting**: تطبيق حد الطلبات (100 طلب/دقيقة لكل Tenant)~~ ✅ **تم**
2. **التكامل مع Redis Buffer**: إضافة منطق حفظ الرسائل في Redis
3. **Metrics**: إضافة مقاييس Prometheus للمراقبة
4. **Authentication**: إضافة التحقق من صلاحيات Tenant

## الأمان

### العزل بين الشركات (Tenant Isolation)

- كل طلب يحتوي على `tenant_id` صريح
- سيتم التحقق من صلاحيات الوصول في المراحل القادمة
- لا يمكن لـ Tenant إرسال رسائل باسم tenant آخر

### التحقق من الصحة

- جميع المدخلات يتم التحقق منها قبل المعالجة
- رفض الطلبات غير الصحيحة فوراً مع رسائل خطأ واضحة
- منع SQL Injection و XSS من خلال التحقق الصارم

## الأداء

- **زمن الاستجابة المستهدف**: < 10ms
- **معدل الإنتاجية**: 5000+ طلب/دقيقة
- **استهلاك الذاكرة**: منخفض (validation فقط، بدون معالجة ثقيلة)

## الأخطاء الشائعة

### 400 Bad Request - VALIDATION_ERROR

**السبب**: البيانات المرسلة غير صحيحة

**الحل**: تحقق من:
- phone_number بصيغة E.164 صحيحة
- content غير فارغ وأقل من 1600 حرف
- tenant_id موجب
- priority (إذا كان موجوداً) من القيم المسموحة

### 429 Too Many Requests - RATE_LIMIT_EXCEEDED

**السبب**: تجاوز حد الطلبات (100 طلب/دقيقة)

**الحل**: 
- انتظر حتى نهاية النافذة الزمنية (تحقق من `X-RateLimit-Reset` header)
- قلل معدل الطلبات
- استخدم Bulk API لإرسال رسائل متعددة في طلب واحد

### 405 Method Not Allowed

**السبب**: استخدام HTTP method غير POST

**الحل**: استخدم POST method فقط

## المساهمة

عند إضافة قواعد validation جديدة:

1. أضف القاعدة في `validator.go`
2. أضف اختبارات في `validator_test.go`
3. حدّث التوثيق في `README.md`
4. تأكد من تغطية الاختبارات > 80%

## الترخيص

هذا الكود جزء من نظام SMS Gateway الداخلي.


## التكامل مع Redis Buffer ✅

تم تطبيق التكامل الكامل مع Redis Buffer في المهمة 3.4:

### MessageBuffer Interface

```go
type MessageBuffer interface {
    Push(ctx context.Context, tenantID int64, msg BufferedMessage) error
    GetSize(ctx context.Context, tenantID int64) (int64, error)
}
```

### استخدام Buffer في Handler

```go
// Create handler with buffer
buffer := buffer.NewMessageBuffer(redisClient, logger)
handler := gateway.NewHandler(rateLimiter, buffer)

// Handler automatically pushes messages to buffer
handler.ReceiveMessage(w, r)
// Message is now in Redis: messages:pending:{tenant_id}
```

### تدفق معالجة الرسالة

1. **Validation**: التحقق من صحة البيانات
2. **Rate Limiting**: فحص حد الطلبات
3. **Buffer Push**: حفظ في Redis Buffer
4. **Response**: إرجاع استجابة فورية (< 10ms)

### عزل الـ Tenants

- كل tenant له queue منفصلة: `messages:pending:{tenant_id}`
- لا يمكن لـ tenant الوصول لرسائل tenant آخر
- Rate limiting منفصل لكل tenant
- Message count منفصل لكل tenant

### معالجة الأخطاء

- `BUFFER_ERROR`: فشل في حفظ الرسالة في Redis
- يتم تسجيل جميع الأخطاء مع تفاصيل كاملة
- إرجاع HTTP 500 عند فشل Buffer

### Redis Data Structures

```
# Message Queue per Tenant (List)
messages:pending:{tenant_id} → List of JSON-encoded BufferedMessage

# Message Count per Tenant (String)
messages:count:{tenant_id} → Integer counter

# Active Tenants (Set)
tenants:active → Set of tenant_ids with pending messages
```

### Performance

- Push operation: O(1) - constant time
- Response time: < 10ms
- Supports high throughput (5000+ messages/minute)

## التطوير المستقبلي

1. ~~**Rate Limiting**: تطبيق حد الطلبات (100 طلب/دقيقة لكل Tenant)~~ ✅ **تم**
2. ~~**التكامل مع Redis Buffer**: إضافة منطق حفظ الرسائل في Redis~~ ✅ **تم**
3. **Metrics**: إضافة مقاييس Prometheus للمراقبة
4. **Authentication**: إضافة التحقق من صلاحيات Tenant
5. **Batch Processor**: معالجة الرسائل من Buffer
