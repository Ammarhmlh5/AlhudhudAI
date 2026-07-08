# API Reference - نظام إدارة الرسائل SMS

## نظرة عامة

هذا المستند يوثق جميع API endpoints المتاحة في نظام إدارة الرسائل SMS.

**Base URL**: `http://localhost:8080/api`  
**Authentication**: JWT Bearer Token  
**Content-Type**: `application/json`

## المصادقة (Authentication)

### تسجيل الدخول

```http
POST /api/auth/login
```

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**Response (200 OK):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 1,
    "email": "user@example.com",
    "role": "admin",
    "tenant_id": 1
  }
}
```

**Error Responses:**
- `400 Bad Request`: بيانات غير صحيحة
- `401 Unauthorized`: بيانات اعتماد خاطئة

---

### تسجيل مستخدم جديد

```http
POST /api/auth/register
```

**Request Body:**
```json
{
  "email": "newuser@example.com",
  "password": "password123",
  "name": "اسم المستخدم",
  "role": "client"
}
```

**Response (201 Created):**
```json
{
  "id": 2,
  "email": "newuser@example.com",
  "name": "اسم المستخدم",
  "role": "client",
  "tenant_id": 2
}
```

---

## إدارة الرسائل (Messages)

### إرسال رسالة واحدة

```http
POST /api/messages
Authorization: Bearer {token}
```

**Request Body:**
```json
{
  "phone_number": "+966501234567",
  "content": "مرحباً، هذه رسالة تجريبية",
  "priority": "normal"
}
```

**Response (200 OK):**
```json
{
  "message_id": "msg_1234567890",
  "status": "QUEUED",
  "phone_number": "+966501234567",
  "created_at": "2026-02-26T10:30:00Z",
  "estimated_cost": 0.05
}
```

**Validation Rules:**
- `phone_number`: يجب أن يبدأ بـ `+` ويحتوي على 10-15 رقم
- `content`: الحد الأقصى 1600 حرف (10 رسائل)
- `priority`: `low`, `normal`, `high`

**Error Responses:**
- `400 Bad Request`: بيانات غير صحيحة
- `401 Unauthorized`: غير مصرح
- `402 Payment Required`: رصيد غير كافٍ
- `429 Too Many Requests`: تجاوز حد الطلبات

---

### إرسال رسائل متعددة (Bulk)

```http
POST /api/messages/bulk
Authorization: Bearer {token}
```

**Request Body:**
```json
{
  "messages": [
    {
      "phone_number": "+966501234567",
      "content": "رسالة 1"
    },
    {
      "phone_number": "+966507654321",
      "content": "رسالة 2"
    }
  ]
}
```

**Response (200 OK):**
```json
{
  "total": 2,
  "queued": 2,
  "failed": 0,
  "messages": [
    {
      "message_id": "msg_1234567890",
      "phone_number": "+966501234567",
      "status": "QUEUED"
    },
    {
      "message_id": "msg_1234567891",
      "phone_number": "+966507654321",
      "status": "QUEUED"
    }
  ]
}
```

**Limits:**
- الحد الأقصى: 1000 رسالة في الطلب الواحد

---

### الحصول على حالة الرسالة

```http
GET /api/messages/{message_id}
Authorization: Bearer {token}
```

**Response (200 OK):**
```json
{
  "message_id": "msg_1234567890",
  "phone_number": "+966501234567",
  "content": "مرحباً، هذه رسالة تجريبية",
  "status": "DELIVERED",
  "created_at": "2026-02-26T10:30:00Z",
  "sent_at": "2026-02-26T10:30:05Z",
  "delivered_at": "2026-02-26T10:30:10Z",
  "cost": 0.05,
  "channel_id": 1,
  "device_id": 5
}
```

**Message Statuses:**
- `QUEUED`: في قائمة الانتظار
- `PROCESSING`: قيد المعالجة
- `BUFFERED`: في ذاكرة الجهاز
- `SENT`: تم الإرسال
- `DELIVERED`: تم التسليم
- `FAILED`: فشل الإرسال
- `WAITING`: في انتظار قناة متاحة

---

### الحصول على قائمة الرسائل

```http
GET /api/messages?page=1&limit=50&status=DELIVERED&from_date=2026-02-01
Authorization: Bearer {token}
```

**Query Parameters:**
- `page`: رقم الصفحة (default: 1)
- `limit`: عدد النتائج (default: 50, max: 100)
- `status`: فلترة حسب الحالة
- `from_date`: من تاريخ (YYYY-MM-DD)
- `to_date`: إلى تاريخ (YYYY-MM-DD)
- `phone_number`: فلترة حسب رقم الهاتف

**Response (200 OK):**
```json
{
  "total": 150,
  "page": 1,
  "limit": 50,
  "messages": [
    {
      "message_id": "msg_1234567890",
      "phone_number": "+966501234567",
      "status": "DELIVERED",
      "created_at": "2026-02-26T10:30:00Z",
      "cost": 0.05
    }
  ]
}
```

---

### حذف رسالة

```http
DELETE /api/messages/{message_id}
Authorization: Bearer {token}
```

**Response (200 OK):**
```json
{
  "message": "تم حذف الرسالة بنجاح"
}
```

**Note:** يمكن حذف الرسائل في حالة `QUEUED` أو `WAITING` فقط.

---

## إدارة القنوات (Channels)

### الحصول على قائمة القنوات

```http
GET /api/channels
Authorization: Bearer {token}
```

**Response (200 OK):**
```json
{
  "channels": [
    {
      "id": 1,
      "name": "قناة 1",
      "status": "ONLINE",
      "priority": 10,
      "current_load": 5,
      "max_load": 100,
      "device_count": 3,
      "success_rate": 0.985
    }
  ]
}
```

---

### إنشاء قناة جديدة

```http
POST /api/channels
Authorization: Bearer {token}
```

**Request Body:**
```json
{
  "name": "قناة جديدة",
  "priority": 5,
  "max_load": 100
}
```

**Response (201 Created):**
```json
{
  "id": 2,
  "name": "قناة جديدة",
  "status": "OFFLINE",
  "priority": 5,
  "max_load": 100
}
```

---

### تحديث قناة

```http
PUT /api/channels/{channel_id}
Authorization: Bearer {token}
```

**Request Body:**
```json
{
  "name": "قناة محدثة",
  "priority": 8,
  "status": "ONLINE"
}
```

**Response (200 OK):**
```json
{
  "id": 2,
  "name": "قناة محدثة",
  "status": "ONLINE",
  "priority": 8
}
```

---

### حذف قناة

```http
DELETE /api/channels/{channel_id}
Authorization: Bearer {token}
```

**Response (200 OK):**
```json
{
  "message": "تم حذف القناة بنجاح"
}
```

---

## إدارة الأجهزة (Devices)

### الحصول على قائمة الأجهزة

```http
GET /api/devices
Authorization: Bearer {token}
```

**Response (200 OK):**
```json
{
  "devices": [
    {
      "id": 1,
      "name": "جهاز 1",
      "phone_number": "+966501234567",
      "status": "ONLINE",
      "channel_id": 1,
      "last_seen": "2026-02-26T10:30:00Z",
      "messages_sent_today": 150
    }
  ]
}
```

---

### تسجيل جهاز جديد

```http
POST /api/devices
Authorization: Bearer {token}
```

**Request Body:**
```json
{
  "name": "جهاز جديد",
  "phone_number": "+966501234567",
  "channel_id": 1,
  "device_token": "fcm_token_here"
}
```

**Response (201 Created):**
```json
{
  "id": 2,
  "name": "جهاز جديد",
  "phone_number": "+966501234567",
  "status": "OFFLINE",
  "api_key": "device_api_key_here"
}
```

---

### تحديث حالة الجهاز

```http
PUT /api/devices/{device_id}/status
Authorization: Bearer {token}
```

**Request Body:**
```json
{
  "status": "ONLINE"
}
```

**Response (200 OK):**
```json
{
  "id": 2,
  "status": "ONLINE",
  "last_seen": "2026-02-26T10:35:00Z"
}
```

---

## إدارة الرصيد (Balance)

### الحصول على الرصيد الحالي

```http
GET /api/balance
Authorization: Bearer {token}
```

**Response (200 OK):**
```json
{
  "tenant_id": 1,
  "balance": 1500.50,
  "currency": "SAR",
  "last_updated": "2026-02-26T10:30:00Z"
}
```

---

### شحن الرصيد

```http
POST /api/balance/recharge
Authorization: Bearer {token}
```

**Request Body:**
```json
{
  "amount": 500.00,
  "payment_method": "credit_card",
  "transaction_id": "txn_1234567890"
}
```

**Response (200 OK):**
```json
{
  "old_balance": 1500.50,
  "amount": 500.00,
  "new_balance": 2000.50,
  "transaction_id": "txn_1234567890"
}
```

---

### الحصول على سجل الرصيد

```http
GET /api/balance/history?page=1&limit=50
Authorization: Bearer {token}
```

**Response (200 OK):**
```json
{
  "total": 200,
  "page": 1,
  "limit": 50,
  "history": [
    {
      "id": 1,
      "type": "DEDUCTION",
      "amount": -0.05,
      "old_balance": 1500.50,
      "new_balance": 1500.45,
      "reason": "Message sent",
      "message_id": "msg_1234567890",
      "created_at": "2026-02-26T10:30:00Z"
    },
    {
      "id": 2,
      "type": "RECHARGE",
      "amount": 500.00,
      "old_balance": 1000.50,
      "new_balance": 1500.50,
      "reason": "Manual recharge",
      "created_at": "2026-02-25T15:00:00Z"
    }
  ]
}
```

---

## إدارة التعرفات (Tariffs)

### الحصول على قائمة التعرفات

```http
GET /api/tariffs
Authorization: Bearer {token}
```

**Response (200 OK):**
```json
{
  "tariffs": [
    {
      "id": 1,
      "country_code": "SA",
      "country_name": "السعودية",
      "cost_per_message": 0.05,
      "currency": "SAR"
    }
  ]
}
```

---

### تحديث تعرفة

```http
PUT /api/tariffs/{tariff_id}
Authorization: Bearer {token}
```

**Request Body:**
```json
{
  "cost_per_message": 0.06
}
```

**Response (200 OK):**
```json
{
  "id": 1,
  "country_code": "SA",
  "cost_per_message": 0.06
}
```

---

## الإحصائيات (Statistics)

### الحصول على إحصائيات عامة

```http
GET /api/stats/overview
Authorization: Bearer {token}
```

**Response (200 OK):**
```json
{
  "today": {
    "messages_sent": 1500,
    "messages_delivered": 1470,
    "messages_failed": 30,
    "success_rate": 0.98,
    "total_cost": 75.00
  },
  "this_month": {
    "messages_sent": 45000,
    "messages_delivered": 44100,
    "messages_failed": 900,
    "success_rate": 0.98,
    "total_cost": 2250.00
  },
  "balance": {
    "current": 1500.50,
    "estimated_messages_remaining": 30010
  }
}
```

---

### الحصول على إحصائيات مفصلة

```http
GET /api/stats/detailed?from_date=2026-02-01&to_date=2026-02-26
Authorization: Bearer {token}
```

**Response (200 OK):**
```json
{
  "period": {
    "from": "2026-02-01",
    "to": "2026-02-26"
  },
  "daily_stats": [
    {
      "date": "2026-02-26",
      "messages_sent": 1500,
      "messages_delivered": 1470,
      "messages_failed": 30,
      "success_rate": 0.98
    }
  ],
  "by_channel": [
    {
      "channel_id": 1,
      "channel_name": "قناة 1",
      "messages_sent": 1000,
      "success_rate": 0.99
    }
  ],
  "by_status": {
    "DELIVERED": 44100,
    "FAILED": 900,
    "QUEUED": 50,
    "PROCESSING": 20
  }
}
```

---

## Dead Letter Queue (DLQ)

### الحصول على الرسائل الفاشلة

```http
GET /api/dlq?page=1&limit=50
Authorization: Bearer {token}
```

**Response (200 OK):**
```json
{
  "total": 30,
  "page": 1,
  "limit": 50,
  "messages": [
    {
      "id": 1,
      "message_id": "msg_1234567890",
      "phone_number": "+966501234567",
      "content": "رسالة فاشلة",
      "attempts": 3,
      "last_error": "Device offline",
      "created_at": "2026-02-26T10:30:00Z",
      "moved_to_dlq_at": "2026-02-26T10:35:00Z"
    }
  ]
}
```

---

### إعادة معالجة رسالة من DLQ

```http
POST /api/dlq/{dlq_id}/reprocess
Authorization: Bearer {token}
```

**Response (200 OK):**
```json
{
  "message": "تم إعادة معالجة الرسالة بنجاح",
  "new_message_id": "msg_9876543210"
}
```

---

### حذف رسالة من DLQ

```http
DELETE /api/dlq/{dlq_id}
Authorization: Bearer {token}
```

**Response (200 OK):**
```json
{
  "message": "تم حذف الرسالة من DLQ بنجاح"
}
```

---

## WebSocket API

### الاتصال بـ WebSocket

```
ws://localhost:8080/ws?token={jwt_token}&device_id={device_id}
```

**Authentication:**
- يتم المصادقة عبر JWT token في query parameter
- يجب تحديد device_id

---

### استقبال رسالة جديدة

**Server → Client:**
```json
{
  "type": "NEW_MESSAGE",
  "data": {
    "message_id": "msg_1234567890",
    "phone_number": "+966501234567",
    "content": "مرحباً، هذه رسالة تجريبية",
    "priority": "normal",
    "created_at": "2026-02-26T10:30:00Z"
  }
}
```

---

### إرسال تقرير حالة

**Client → Server:**
```json
{
  "type": "STATUS_REPORT",
  "data": {
    "message_id": "msg_1234567890",
    "status": "DELIVERED",
    "timestamp": "2026-02-26T10:30:10Z",
    "error_message": null
  }
}
```

**Possible Statuses:**
- `BUFFERED`: تم حفظ الرسالة في ذاكرة الجهاز
- `SENT`: تم إرسال الرسالة
- `DELIVERED`: تم تسليم الرسالة
- `FAILED`: فشل الإرسال

---

### Heartbeat

**Server → Client (كل 30 ثانية):**
```json
{
  "type": "PING",
  "timestamp": "2026-02-26T10:30:00Z"
}
```

**Client → Server:**
```json
{
  "type": "PONG",
  "timestamp": "2026-02-26T10:30:00Z"
}
```

---

## Error Codes

### HTTP Status Codes

| Code | Description |
|------|-------------|
| 200 | OK - نجح الطلب |
| 201 | Created - تم إنشاء المورد |
| 400 | Bad Request - بيانات غير صحيحة |
| 401 | Unauthorized - غير مصرح |
| 402 | Payment Required - رصيد غير كافٍ |
| 403 | Forbidden - ممنوع |
| 404 | Not Found - غير موجود |
| 429 | Too Many Requests - تجاوز حد الطلبات |
| 500 | Internal Server Error - خطأ في الخادم |
| 503 | Service Unavailable - الخدمة غير متاحة |

### Application Error Codes

```json
{
  "error": {
    "code": "INSUFFICIENT_BALANCE",
    "message": "رصيدك غير كافٍ لإرسال هذه الرسالة",
    "details": {
      "required": 0.05,
      "available": 0.02
    }
  }
}
```

**Error Codes:**
- `INVALID_PHONE_NUMBER`: رقم هاتف غير صحيح
- `INVALID_CONTENT`: محتوى غير صحيح
- `INSUFFICIENT_BALANCE`: رصيد غير كافٍ
- `RATE_LIMIT_EXCEEDED`: تجاوز حد الطلبات
- `NO_CHANNEL_AVAILABLE`: لا توجد قنوات متاحة
- `DEVICE_OFFLINE`: الجهاز غير متصل
- `MESSAGE_NOT_FOUND`: الرسالة غير موجودة
- `UNAUTHORIZED`: غير مصرح
- `FORBIDDEN`: ممنوع

---

## Rate Limiting

جميع API endpoints محمية بـ rate limiting:

**الحدود الافتراضية:**
- 100 طلب/دقيقة لكل tenant
- 1000 رسالة/دقيقة لكل tenant

**Response Headers:**
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1709028600
```

**عند تجاوز الحد:**
```http
HTTP/1.1 429 Too Many Requests
Retry-After: 60

{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "تجاوزت حد الطلبات المسموح به",
    "retry_after": 60
  }
}
```

---

## Pagination

جميع endpoints التي تُرجع قوائم تدعم pagination:

**Query Parameters:**
- `page`: رقم الصفحة (default: 1)
- `limit`: عدد النتائج (default: 50, max: 100)

**Response Format:**
```json
{
  "total": 500,
  "page": 1,
  "limit": 50,
  "pages": 10,
  "data": [...]
}
```

---

## Versioning

API version يتم تحديده في URL:

```
/api/v1/messages
/api/v2/messages
```

**Current Version**: v1  
**Deprecated Versions**: None

---

## Support

للدعم الفني:
- Email: support@example.com
- Documentation: https://docs.example.com
- Status Page: https://status.example.com

---

**تاريخ الإنشاء**: 2026-02-26  
**الإصدار**: 1.0  
**آخر تحديث**: 2026-02-26
