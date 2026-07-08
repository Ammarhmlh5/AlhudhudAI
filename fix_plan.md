# خطة الإصلاح: Fix Plan لمشكلة وصول الرسائل من الموقع إلى التطبيق

## المشاكل المكتشفة والحلول

### المشكلة 1: فلتر `ACCOUNTING_SYSTEM` يمنع جميع رسائل الموقع (حرجة)
**الملف:** `services/backend/internal/queue/jobs.go:373`
**الوصف:** الكود يشترط `msg.Source == "ACCOUNTING_SYSTEM` لإرسال الرسائل عبر WebSocket. رسائل الموقع (Web UI) مصدرها مختلف (`WEB_UI`, `BULK_API`, `MANUAL`, إلخ) لذلك يتم تجاهلها and return nil.

**الحل:** إزالة الشرط الحصر `ACCOUNTING_SYSTEM` والسماح لكل المصادر التي تريد إرسالاً للهاتف. تعديل الشرط ليكون `msg.Source != "INBOUND_SMS"` (أي نمنع فقط الرسائل الواردة من SMS).

### المشكلة 2: عدم تطابق المعرفات في Redis Dispatch/WebSocket (حرجة)
**الملف:** `services/backend/internal/queue/jobs.go:427-436`
**الوصف:** إرسال Redis PublishMessage يستخدم `msg.DeviceID` الذي قد يكون `nil` للرسائل الواردة من الموقع (لأنها لم تمر عبر `DispatchToPhone`). حتى لو كان `DeviceID` موجوداً، قد لا يتطابق مع ChannelID المستخدم في اتصال WebSocket.

**الحل:** إزالة اعتماد Redis PublishMessage لصالح WebSocket dispatch الموحد. استخدام `ws.GlobalHub.SendMessageToTenant` أو التكرار على `GetMobileConnectionsByTenant` مباشرةً.

### المشكلة 3: `deliverPendingMessagesToDevice` لا يعمل فعلياً (حرجة)
**الملف:** `services/backend/internal/ws/hub.go:227-285`
**الوصف:** الدالة `deliverPendingMessagesToDevice` تنشئ متغير `pendingMessages` لكن لا تستعلم من قاعدة البيانات فعلياً - تترك `pendingMessages` فارغة! لذلك عند اتصال جهاز جديد، لا تصل الرسائل القديمة.

**الحل:** إضافة استعلام حقيقي من قاعدة البيانات لجلب الرسائل المعلقة (`PENDING`, `QUEUED`, `WAITING`) التي لم ترسل للجهاز بعد.

### المشكلة 4: لا يوجد Event Table في تطبيق الهاتف (تحسين كبير)
**الوصف:** حالياً عند وصول رسالة عبر WebSocket، تحفظ مباشرةً في جدول `messages` ويتم إرسالها للمعالجة فوراً. هذا يسبب ازدحاماً وقد يفقد الرسائل إذا كان التطبيق مشغولاً.

**الحل:** 
- إنشاء جدول `message_events` في SQLite لتخزين الرسائل الواردة فوراً (buffer).
- `WebSocketService` يحفظ الرسالة في `message_events` فقط ثم يرد ACK.
- `SmsQueueManager` يقرأ من `message_events` تدريجياً وبشكل منتظم (مهما كانت سرعة المعالجة).
- بعد المعالجة، ينقل إلى جدول `messages` أو يحذف من `message_events`.

### المشكلة 5: `sendStatusReport` قد تفشل بدون Queue (متوسطة)
**الوصف:** عند إرسال `STATUS_REPORT` عبر WebSocket، إذا كان الاتصال غير مستقر، يفقد التقرير. لا يوجد retry queue للتقارير.

**الحل:** إضافة `status_reports` جدول في SQLite يحفظ التقارير التي لم ترسل بعد، مع retry تلقائي.

---

## خطة التنفيذ

### الخطوة 1: إصلاح backend (files: jobs.go, hub.go)
- تعديل فلتر المصدر ليمرر كل الرسائل ما عدا `INBOUND_SMS`
- إزالة Redis PublishMessage القديم (أو تركه كـ fallback)
- تفعيل WebSocket dispatch الموحد لكل المصادر
- إضافة استعلام حقيقي في `deliverPendingMessagesToDevice`

### الخطوة 2: إنشاء Event Table في تطبيق الهاتف (files: migrations.ts, DatabaseService.ts)
- إضافة جدول `message_events` في الـ migration
- إضافة دوال في DatabaseService للتعامل مع event table

### الخطوة 3: تعديل WebSocketService.ts
- عند استلام `MESSAGE_DELIVERY` → حفظ في `message_events` فقط
- إرسال `BUFFERED` فوراً
- عدم استدعاء `processMessage` مباشرةً

### الخطوة 4: تعديل SmsQueueManager.ts
- Polling منتظم (كل 1-2 ثانية) من `message_events`
- معالجة تدريجية بعدد محدود في كل دورة
- تحديث الحالة وإرسال التقارير

### الخطوة 5: إضافة Status Report Queue
- جدول `status_reports` لحفظ التقارير غير المرسلة
- Retry تلقائي في الخلفية
