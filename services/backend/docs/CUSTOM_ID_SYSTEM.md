# Custom ID System Documentation

## نظرة عامة

تم تطوير نظام Custom ID لتوليد معرفات فريدة للأحداث والرسائل باستخدام Redis، مما يوفر:
- ✅ Traceability كاملة
- ✅ Load detection في الوقت الفعلي
- ✅ أداء عالي
- ✅ معلومات مدمجة في الرقم نفسه

---

## 🎯 هيكل الأرقام

### 1. Event ID (19 حرف)
```
EV 26010123 12345 0001
││ ││││││││ │││││ ││││
││ ││││││││ │││││ └──── Event Sequence (0001-9999)
││ ││││││││ └───────── Tenant Hash (5 digits)
││ └────────────────── Timestamp YYMMDDHHmmss
└──────────────────── Entity Type: EV = Event
```

**مثال:**
```
EV2601012312345000100
│ │       │    │
│ │       │    └─ الحدث الأول
│ │       └────── الشركة 12345
│ └────────────── 23 يناير 2026، 01:01
└──────────────── Event
```

### 2. Message ID (22 حرف)
```
MS 26010123 12345 0001 001
││ ││││││││ │││││ ││││ │││
││ ││││││││ │││││ ││││ └─── Message Sequence (001-999)
││ ││││││││ │││││ └──────── Event Sequence (0001-9999)
││ ││││││││ └────────────── Tenant Hash (5 digits)
││ └───────────────────── Timestamp YYMMDDHHmmss
└─────────────────────── Entity Type: MS = Message
```

**مثال:**
```
MS26010123123450001001
│ │       │    │   │
│ │       │    │   └─ الرسالة الأولى في الحدث
│ │       │    └───── الحدث الأول
│ │       └────────── الشركة 12345
│ └────────────────── 23 يناير 2026، 01:01
└──────────────────── Message
```

---

## 📊 القيود والحدود

| القيد | القيمة | الوصف |
|-------|--------|-------|
| **Messages per Event** | 999 | حد أقصى للرسائل في حدث واحد |
| **Events per Hour** | 9999 | حد أقصى للأحداث في الساعة لكل شركة |
| **Tenant Hash Range** | 00000-99999 | نطاق hash الشركة |

---

## 🔧 الاستخدام

### 1. توليد Event ID

```go
import "github.com/al-hudhud/backend/internal/redis"

// إنشاء ID generator
idGen := redis.NewIDGenerator(redisClient)

// توليد Event ID
eventID, err := idGen.GenerateEventID(ctx, tenantID, senderIP)
// Returns: "EV2601012312345000100"
```

### 2. توليد Message ID

```go
// توليد Message ID (يجب أن يكون لديك Event ID)
messageID, err := idGen.GenerateMessageID(ctx, eventID)
// Returns: "MS26010123123450001001"
```

### 3. Parse Event ID

```go
info, err := redis.ParseEventID("EV2601012312345000100")
// Returns:
// {
//     EntityType: "EV",
//     Year: 2026,
//     Month: 1,
//     Day: 23,
//     Hour: 1,
//     TenantHash: "12345",
//     Sequence: 1
// }
```

### 4. Parse Message ID

```go
info, err := redis.ParseMessageID("MS26010123123450001001")
// Returns:
// {
//     EntityType: "MS",
//     Year: 2026,
//     Month: 1,
//     Day: 23,
//     Hour: 1,
//     TenantHash: "12345",
//     EventSeq: 1,
//     MessageSeq: 1,
//     EventID: "EV2601012312345000100"
// }
```

---

## 📈 Load Analytics

### 1. الحصول على الحمل الحالي

```go
analytics := redis.NewLoadAnalytics(redisClient)

metrics, err := analytics.GetCurrentLoad(ctx)
// Returns:
// {
//     EventCount: 1234,
//     MessageCount: 45678,
//     AvgMsgPerEvent: 37.0,
//     MaxMsgPerEvent: 999,
//     NearLimitCount: 5
// }
```

### 2. اكتشاف الضغط

```go
pressure, err := analytics.DetectPressure(ctx)
// Returns:
// {
//     Level: "HIGH",
//     Reason: "High event count: 5234 events/hour"
// }
```

### 3. إحصائيات حدث معين

```go
stats, err := analytics.GetEventStats(ctx, "EV2601012312345000100")
// Returns:
// {
//     EventID: "EV2601012312345000100",
//     TenantID: "uuid-here",
//     SenderIP: "192.168.1.1",
//     MessageCount: 123,
//     CreatedAt: time.Time
// }
```

---

## 🗄️ قاعدة البيانات

### الأعمدة الجديدة

#### global_incoming_events
```sql
id          BIGSERIAL    -- Internal counter (admin stats only)
custom_id   VARCHAR(20)  -- Redis-generated Event ID (EV...)
tenant_id   UUID
sender_ip   INET
...
```

#### messages
```sql
id               UUID         -- Internal counter (admin stats only)
custom_id        VARCHAR(23)  -- Redis-generated Message ID (MS...)
event_custom_id  VARCHAR(20)  -- Reference to Event custom_id
tenant_id        UUID
...
```

### Views

#### v_messages_with_custom_ids
```sql
SELECT 
    m.custom_id as message_id,
    m.event_custom_id as event_id,
    m.tenant_id,
    m.phone_number,
    m.status,
    ge.sender_ip as event_sender_ip
FROM messages m
LEFT JOIN global_incoming_events ge ON m.event_custom_id = ge.custom_id;
```

#### v_event_load_analytics
```sql
SELECT 
    extract_timestamp_from_custom_id(custom_id) as event_hour,
    extract_tenant_hash_from_custom_id(custom_id) as tenant_hash,
    COUNT(*) as event_count,
    SUM(total_items) as total_messages,
    AVG(total_items) as avg_messages_per_event
FROM global_incoming_events
GROUP BY event_hour, tenant_hash;
```

---

## 🔍 الاستعلامات المفيدة

### 1. البحث عن رسالة بـ Custom ID
```sql
SELECT * FROM messages WHERE custom_id = 'MS26010123123450001001';
```

### 2. الحصول على جميع رسائل حدث معين
```sql
SELECT * FROM messages WHERE event_custom_id = 'EV2601012312345000100';
```

### 3. تحليل الحمل لشركة معينة
```sql
SELECT * FROM v_event_load_analytics 
WHERE tenant_hash = '12345' 
ORDER BY event_hour DESC 
LIMIT 24;
```

### 4. الأحداث القريبة من الحد الأقصى
```sql
SELECT custom_id, total_items 
FROM global_incoming_events 
WHERE total_items >= 900 
ORDER BY created_at DESC;
```

---

## ⚠️ معالجة الأخطاء

### 1. تجاوز حد الرسائل (999)
```go
messageID, err := idGen.GenerateMessageID(ctx, eventID)
if err != nil {
    if strings.Contains(err.Error(), "message limit exceeded") {
        // إنشاء حدث جديد تلقائياً
        newEventID, _ := idGen.GenerateEventID(ctx, tenantID, senderIP)
        messageID, _ = idGen.GenerateMessageID(ctx, newEventID)
    }
}
```

### 2. تجاوز حد الأحداث (9999)
```go
eventID, err := idGen.GenerateEventID(ctx, tenantID, senderIP)
if err != nil {
    if strings.Contains(err.Error(), "event sequence limit exceeded") {
        // انتظر الثانية التالية
        time.Sleep(time.Second)
        eventID, _ = idGen.GenerateEventID(ctx, tenantID, senderIP)
    }
}
```

---

## 📊 Monitoring

### Redis Keys

```
alhudhud:event:counter:26010123:12345           → Event counter
alhudhud:event:EV2601012312345000100            → Event metadata
alhudhud:event:EV2601012312345000100:msg:counter → Message counter
```

### Metrics to Monitor

1. **Event Rate**: عدد الأحداث في الثانية/الدقيقة
2. **Message Rate**: عدد الرسائل في الثانية
3. **Events Near Limit**: أحداث تقترب من 999 رسالة
4. **Avg Messages per Event**: متوسط الرسائل لكل حدث

---

## 🎯 الفوائد

1. **Traceability**: تتبع كامل من الرسالة → الحدث → الشركة → الوقت
2. **Load Detection**: معرفة الضغط من الأرقام مباشرة
3. **Performance**: توليد سريع في Redis
4. **Analytics**: تحليل سهل للأنماط
5. **Debugging**: troubleshooting أسهل

---

## 🔄 Migration

لترحيل البيانات الموجودة:

```sql
-- Run migration
\i migrations/077_add_custom_ids.sql

-- Existing data will have NULL custom_ids
-- New data will have Redis-generated custom_ids
```

---

## 📝 ملاحظات مهمة

1. **Internal IDs**: الأعمدة `id` (BIGSERIAL/UUID) تبقى للإحصائيات الإدارية فقط
2. **Custom IDs**: استخدم `custom_id` للتتبع والترقيم
3. **Uniqueness**: Custom IDs فريدة 100% بفضل Redis atomic operations
4. **TTL**: البيانات في Redis تُحفظ لمدة 7 أيام
5. **Fallback**: إذا فشل Redis، يمكن استخدام PostgreSQL IDs مؤقتاً

---

## 🚀 الخلاصة

النظام الجديد يوفر:
- ✅ معرفات فريدة وذكية
- ✅ تتبع كامل للأحداث والرسائل
- ✅ اكتشاف الضغط في الوقت الفعلي
- ✅ تحليلات متقدمة
- ✅ أداء عالي

استخدم Custom IDs في جميع العمليات الجديدة!
