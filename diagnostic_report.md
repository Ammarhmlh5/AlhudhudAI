# تقرير تشخيص مشكلة عدم وصول الرسائل من الموقع إلى تطبيق الهاتف

**التاريخ:** 2026-07-08  
**النظام:** ALHudhudAI v3.2.7  
**الخبير:** Kiro AI

---

## ملخص تنفيذي

تم تحليل كامل لمسار تدفق الرسائل من الموقع (Web UI / API) إلى تطبيق الهاتف (Mobile Gateway). تم اكتشاف **3 أخطاء حرجة** و**4 مشاكل ثانوية** تمنع وصول الرسائل إلى التطبيق. الخطأ الأبرز هو أن `MessageActivationWorker` يتجاهل جميع الرسائل غير القادمة من `ACCOUNTING_SYSTEM` بشكل صامت.

---

## الأخطاء الحرجة (Critical Bugs)

### الخطأ رقم 1: عامل تصفية المصدر في MessageActivationWorker يمنع جميع رسائل الويب

**الملف:** `internal/queue/jobs.go:372-378`

```go
if msg.Source != "ACCOUNTING_SYSTEM" {
    log.Debug("Skipping mobile dispatch for non-accounting system message")
    return nil  // ◄◄◄ يعود بدون إرسال أي شيء!
}
```

**الشرح:**  
عندما يتم إرسال رسالة من الموقع (Web UI/API)، يكون مصدرها (`Source`) = `"WEB_UI"` أو `"WEB_API"`. هذا الكود يتحقق من أن المصدر هو `"ACCOUNTING_SYSTEM"` فقط، وإلا فإنه **يرجع فوراً** (`return nil`) بدون إرسال الرسالة إلى تطبيق الهاتف عبر WebSocket. هذا هو السبب **الأكثر احتمالاً** لعدم وصول الرسائل.

**كيفية الوصول للخطأ:**
1. المستخدم يرسل رسالة من الموقع ← `POST /api/v1/messaging/send`
2. `StageMessage()` يحفظ الرسالة بحالة `PENDING` ومصدر `WEB_UI`
3. `TenantDispatcher` يلتقطها ويُدرجها في River كـ `SendSMSArgs`
4. أو `GlobalEventWorker` يُدخل `MessageActivationArgs`
5. `MessageActivationWorker.Work()` يتحقق من `Source` ← `"WEB_UI"` ≠ `"ACCOUNTING_SYSTEM"` ← **يتجاهل الرسالة!**

---

### الخطأ رقم 2: عدم تطابق المعرفات (ID Mismatch) في مسار PHONES السريع

**الملفات المعنية:**
- `internal/receiver/phones_path.go:25-35` ← يستخدم `PairedDevice.ID`
- `internal/device/service.go:323-343` ← يعيد `DispatchChannel.ID`
- `internal/ws/hub.go:1031-1043` ← يطابق `conn.DeviceID`
- `main.go:476` ← يمرر `channelID` كـ `deviceID`

**الشرح:**  
عند استخدام المسار السريع `PHONES`:

```
📤 DispatchToPhone() → Redis channel:  alhudhud:dispatch:{PairedDevice.ID}
📱 WebSocket Connection.DeviceID = DispatchChannel.ID (قيمة مختلفة)
❌ Redis subscriber يقارن: conn.DeviceID == deviceID → لا يطابق → لا يوصل!
```

**التدفق الكامل:**
1. `DispatchToPhone()` ينشر على Redis: `alhudhud:dispatch:{targetDeviceID}`
   - `targetDeviceID` = `companyStatus.DeviceID` (عادةً `PairedDevice.ID`)
2. اتصال WebSocket للجهاز المحمول: `Connection.DeviceID` = قيمة `channelID` من `GetTenantIDByAPIKey()`
   - `channelID` = `device.ChannelID` (أي `DispatchChannel.ID`)
3. `subscribeToRedis()` يبحث عن `conn.DeviceID == deviceID` ← **لا يوجد تطابق**

---

### الخطأ رقم 3: انقسام مسار المعالجة (Split Pipeline Confusion)

**الملفات المعنية:**
- `internal/queue/jobs.go:47-160` ← `SendSMSWorker` (يرسل إلى بوابات خارجية)
- `internal/worker/pool.go:440-571` ← `MessageWorker` (يرسل عبر WebSocket)

**الشرح:**  
يوجد **مُعامِلا River Worker** مختلفان يتعاملان مع نفس نوع الوظيفة `SendSMSArgs` (kind: `"send_sms"`):

1. **`SendSMSWorker`** (في `queue/jobs.go`): مسجل عبر `queue.InitTenantClient()` لكل tenant
   - يقوم بالتوجيه عبر `RoutingEngine.GetBestChannel()`
   - يرسل عبر `provider.Send()` ← بوابات SMS خارجية (Bawabaty)
   - **لا يرسل إلى WebSocket الجهاز المحمول!**

2. **`MessageWorker`** (في `worker/pool.go`): مسجل في `RiverWorkerPool` العام
   - يقوم بالتوجيه عبر `RoutingEngine.SelectChannel()`
   - يرسل عبر `WebSocketHub.SendMessage()` ← الجهاز المحمول
   - **لكن هل هذا الـ pool قيد التشغيل فعلاً؟**

النتيجة: عندما تمر الرسائل عبر المسار الطبيعي (`PENDING → TenantDispatcher → River → SendSMSWorker`)، فإنها تذهب إلى بوابة SMS خارجية ولا تصل إلى تطبيق الهاتف أبداً.

---

## المشاكل الثانوية (Secondary Issues)

### مشكلة رقم 4: `deliverPendingMessagesToDevice` لا تعمل فعلياً

**الملف:** `internal/ws/hub.go:228-285`

```go
var pendingMessages []struct{...}  // ← مصفوفة فارغة، لا يوجد استعلام DB!
// ...
if len(pendingMessages) > 0 {  // ← هذا الشرط永远不会 true
```

الدالة تُنشئ مصفوفة محلية فارغة ولا تقوم بأي استعلام لقاعدة البيانات لملئها. لذلك، عند إعادة اتصال الجهاز لا يتم تسليم الرسائل المعلقة.

---

### مشكلة رقم 5: `FastMessageProcessor` يرسل إلى WebSocket ولكن المسار الطبيعي لا يستخدمه

**الملف:** `internal/worker/fast_processor.go:247-276`

`FastMessageProcessor` يرسل الرسائل عبر `ws.Hub.SendMessage()` بشكل صحيح إلى WebSocket. لكن هذا المعالج يعمل فقط على رسائل **Redis MessageQueue** (المسار السريع)، وليس على رسائل **River Queue** (المسار الطبيعي).

الرسائل التي تمر عبر `TenantDispatcher` ← River ← `SendSMSWorker` **لا تمر عبر `FastMessageProcessor` أبداً**.

---

### مشكلة رقم 6: `broadcastEvent` قد يرسل لكل اتصالات الـ tenant

**الملف:** `internal/ws/hub.go:288-315`

`broadcastEvent()` يرسل لجميع اتصالات الـ tenant (بما في ذلك جلسات الويب داشبورد)، بينما `SendMessage()` يرسل لاتصال محدد. الخلط بينهما قد يؤدي إلى إرسال رسائل SMS غير مناسبة لواجهة الويب.

---

### مشكلة رقم 7: `fmt.Sscanf` المستخدم في تحليل قناة Redis قد يفشل مع المعرفات المركبة

**الملف:** `internal/ws/hub.go:1021`

```go
fmt.Sscanf(msg.Channel, "alhudhud:dispatch:%s", &deviceIDStr)
```

`fmt.Sscanf` مع `%s` سيتوقف عند أول مسافة أو شرطة. إذا كان اسم القناة يحتوي على أجزاء إضافية، فقد يقرأ جزءاً فقط من المعرف.

---

## تحليل تدفق الرسالة الكامل

```
📱 الموقع (Web UI)
    │
    ▼ POST /api/v1/messaging/send
📥 StageMessage()
    │
    ├── [إذا isActive = true (شركة نشطة)]
    │   │
    │   ├── ChannelType = "PHONES"
    │   │   └── DispatchToPhone() → Redis alhudhud:dispatch:{device_id}
    │   │       └── WebSocket Hub.subscribeToRedis() ← BUG #2: ID MISMATCH
    │   │
    │   ├── ChannelType = "ALHUDHUD"
    │   │   └── DispatchToAlhudhud() → Redis alhudhud:dispatch:official:{tid}
    │   │       └── Alhudhud Worker → Bawabaty API → SMS ← صحيح ولكن ليس عبر التطبيق
    │   │
    │   └── ChannelType = "PRIVATE"
    │       └── DispatchToPrivateServer() → Redis alhudhud:dispatch:private:{tid}
    │           └── Private Worker → خادم خاص → SMS
    │
    └── [إذا isActive = false أو فشل المسار السريع]
        │
        └── حفظ كـ PENDING
            │
            ▼ كل 2 ثانية
        TenantDispatcher.fetchAndQueueMessages()
            │
            ├── GetConnectionStatus() ← متصل؟
            │   ├── لا → WAITING
            │   └── نعم → QUEUED
            │
            ▼
        River Job: SendSMSArgs
            │
            ├── SendSMSWorker.Work()  ← BUG #3: لا يرسل إلى WebSocket
            │   └── provider.Send() → بوابة SMS خارجية
            │
            └── MessageWorker.Work()  ← BUG #1: قد لا يكون نشطاً
                └── WebSocketHub.SendMessage() → جهاز محمول
```

---

## توصيات الحلول العاجلة

### الحل العاجل رقم 1: إزالة فلتر المصدر في MessageActivationWorker

**الملف:** `internal/queue/jobs.go:372-378`

```go
// إزالة الشرط أو تعديله ليشمل جميع المصادر
// قبل:
if msg.Source != "ACCOUNTING_SYSTEM" {
    return nil
}

// بعد: إزالة الشرط بالكامل
// (يُترك الكود ليكمل إلى WebSocket dispatch)
```

### الحل العاجل رقم 2: إصلاح مطابقة المعرفات في Redis Pub/Sub

**الملف:** `internal/receiver/phones_path.go`

تغيير `DispatchToPhone()` لاستخدام نفس المعرف الذي يستخدمه WebSocket Connection:

```go
// بدلاً من PairedDevice.ID، استخدم ChannelID (DispatchChannel.ID) المربوط بالجهاز
channelName := fmt.Sprintf("alhudhud:dispatch:%s", device.ChannelID.String())
// أو: استخدم device.ID من جدول paired_devices مباشرة
```

### الحل العاجل رقم 3: تصحيح `deliverPendingMessagesToDevice`

**الملف:** `internal/ws/hub.go:228-285`

إضافة استعلام فعلي لقاعدة البيانات لجلب الرسائل المعلقة:

```go
// استعلام الرسائل ذات الحالة PENDING/QUEUED لهذا الـ tenant
db.Get(ctx).Where("tenant_id = ? AND status IN ('PENDING', 'QUEUED')", conn.TenantID).Find(&pendingMessages)
```

### الحل العاجل رقم 4: توحيد مسار الإرسال

التأكد من أن `SendSMSWorker` في `queue/jobs.go` يرسل أيضاً عبر WebSocket بعد الإرسال إلى المزوّد، أو استخدام `MessageWorker` في `worker/pool.go` حصرياً للرسائل الموجهة للأجهزة المحمولة.

---

## فحص سريع للتشخيص

لتأكيد التشخيص، قم بتشغيل الفحوصات التالية:

```bash
# 1. التحقق من حالة الرسائل في قاعدة البيانات
SELECT custom_id, phone_number, status, source, created_at 
FROM messages 
ORDER BY created_at DESC 
LIMIT 10;

# 2. التحقق من وجود أجهزة مقترنة
SELECT * FROM paired_devices;

# 3. التحقق من حالة قنوات التوزيع
SELECT * FROM dispatch_channels;

# 4. التحقق من حالة الشركة (is_active, channel_type)
SELECT * FROM company_status;
```

---

## ملخص

| # | المشكلة | النوع | التأثير | الأولوية |
|---|---------|-------|---------|----------|
| 1 | فلتر `ACCOUNTING_SYSTEM` في `MessageActivationWorker` | خطأ برمجي | يمنع جميع رسائل الويب | **حرجة** |
| 2 | عدم تطابق المعرفات في مسار PHONES | خطأ برمجي | يمنع الإرسال السريع للأجهزة | **حرجة** |
| 3 | انقسام مسار المعالجة (SendSMSWorker vs MessageWorker) | تصميم | إرسال لبوابة خارجية بدلاً من التطبيق | **حرجة** |
| 4 | `deliverPendingMessagesToDevice` لا يعمل | خطأ برمجي | لا يتم تسليم الرسائل المعلقة عند إعادة الاتصال | متوسطة |
| 5 | FastMessageProcessor يعمل فقط على Redis queue | تصميم | المسار الطبيعي لا يستخدم WebSocket | متوسطة |
| 6 | `broadcastEvent` يرسل لكل الاتصالات | تصميم | إرسال غير دقيق | منخفضة |
| 7 | استخدام `fmt.Sscanf` لتحليل قناة Redis | خطأ برمجي | قد لا يقرأ المعرف كاملاً | منخفضة |

**التوصية:** إعطاء أولوية قصوى للحلول رقم 1 و 2 و 3، فهي تمنع وصول جميع الرسائل من الموقع إلى التطبيق.
