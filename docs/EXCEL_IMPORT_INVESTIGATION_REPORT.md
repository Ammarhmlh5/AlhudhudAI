# تقرير التحقيق: مشكلة الإرسال من ملف إكسل (Excel Import)

**التاريخ:** 2026-08-03
**المُعد بواسطة:** فحص عميق لمشروع الهدهد (backend + client-web)
**الحالة:** مكتمل — جذور المشكلة محددّة ومؤكدة بالأدلة

---

## 1. الملخص التنفيذي

أرسل المستأجر ملف إكسل يحتوي على **183 رسالة** (BULK) من تبويب «إرسال عبر ملف إكسل»
في واجهة المستأجرين (`client-web`)، وظهرت رسالة «تم استلام البيانات بنجاح»، لكن:

- الرسائل **لم تظهر في تطبيق الهاتف** إطلاقاً.
- الرسائل **لم تظهر في تبويب الرسائل** بواجهة الويب.

**السبب الجذري (مؤكد):** الواجهة تستدعي المسار الخاطئ `/api/v1/messaging/ingest`
(مسار «الرسائل الخام المفردة» المخصص للأنظمة المحاسبية)، فيُخزَّن كامل الـ JSON
(وفيه مصفوفة `messages`) في صفّ واحد بجدول `incoming_messages` **دون** تعبئة
`phone_number` و`content`. مهمة `message_analysis` تحاول بعدها إنشاء رسالة بمحتوى
فارغ فترفضها قاعدة البيانات (قيد `messages_content_not_empty`) وتبقى تتعطل
وتُعاد المحاولة إلى الأبد. لذلك **لم يُنشأ أي سجل في جدول `messages`**،
وبالتالي لا يظهر شيء في الويب ولا يُسحب إلى الهاتف.

---

## 2. الأعراض المسجَّلة

| الأعراض | الملاحظة |
|---|---|
| رسالة نجاح في الواجهة بعد لصق الملف | ✔ تظهر (الـ endpoint يستجيب 202) |
| الرسائل في تطبيق الهاتف | ✘ لا توجد |
| الرسائل في تبويب «الرسائل» بالويب | ✘ لا توجد |
| أي صف في جدول `messages` للمستأجر | ✘ لا يوجد (لم يُنشأ من هذه العملية) |
| أي حدث في `global_incoming_events` | ✘ لا يوجد (المسار الجماعي لم يُستخدم) |

---

## 3. المسار المتوقع (التصميم المقصود)

```
ExcelImport.jsx
  └─ POST /api/v1/messaging/excel-ingest        ← المسار الجماعي المقصود
       └─ BulkIngestExcelHandler (receiver/handler.go)
            └─ IngestGlobalEvent → global_incoming_events  (Phase 1)
                 └─ GlobalEventWorker (queue/jobs.go)      (Phase 2)
                      └─ يوزّع مصفوفة messages → ينشئ 183 صفاً في messages
                           └─ MessageActivation → إتاحة للهاتف عبر fetch/WS
```

---

## 4. الأدلة (فحص قاعدة البيانات والسجلات)

### 4.1 العملية وصلت فعلاً للخادم لكن بمسار خاطئ

```sql
-- incoming_messages (آخر عمليات إكسل)
 id                     | tenant_id                        | sender_name       | is_processed | payload
 6536db77-...           | cd2fa5fc-...                     | Accounting System | f            | {"api_key": "ACC-67794f81-...", "messages": [...183...]}
 e36020cd-...           | cd2fa5fc-...                     | Accounting System | f            | {"api_key": "ACC-67794f81-...", "messages": [...]}
 b29fd668-...           | cd2fa5fc-...                     | Accounting System | f            | {"api_key": "ACC-67794f81-...", "messages": [...]}
```

- `is_processed = f` في كل الحالات → لم يُعالَج أي منها.
- `sender_name = "Accounting System"` رغم أن الواجهة أرسلت `"Web-Excel-Import"` في الـ body
  (السبب: الـ handler يقرأ `sender_name` من **query param** وليس من الـ body).

### 4.2 مهمة المعالجة تفشل دائماً بسبب قيد قاعدة البيانات

```sql
-- river_job في schema المستأجر
 id | kind             | state     | attempt | errors (آخر خطأ)
 93 | message_analysis | retryable |       5 | ERROR: new row for relation "messages" violates
                                                check constraint "messages_content_not_empty" (SQLSTATE 23514)
 92 | message_analysis | running   |       5 | نفس الخطأ
 59 | message_analysis | retryable |      14 | نفس الخطأ (من 2026-08-02، 14 محاولة فاشلة على مدى يومين)
```

```sql
-- القيود الفعلية على جدول messages
 messages_content_not_empty      | CHECK ((length(content) > 0))
 messages_phone_number_not_empty | CHECK ((length(phone_number) > 0))
```

### 4.3 لا يوجد أي حدث جماعي

```sql
SELECT * FROM global_incoming_events WHERE tenant_id='cd2fa5fc-...';  -- 0 rows
```

---

## 5. جذور المشكلة (Root Causes)

### المشكلة 1 — الواجهة تستدعي المسار الخاطئ (السبب الرئيسي)
- الملف: `apps/client-web/src/components/ExcelImport.jsx:117`
- الكود: `await apiClient.post('/api/v1/messaging/ingest', ingestionPayload)`
- الصحيح: `/api/v1/messaging/excel-ingest` (الخاص بالاستيراد الجماعي من إكسل).
- ملاحظة: الواجهة ترسل `api_key` داخل الـ body بينما المصادقة تتم عبر JWT (Cookie)
  و/أو رأس `X-API-Key`؛ حقل `api_key` في الـ body لا يُستخدم — غير حاصرة لكنها مضللة.

### المشكلة 2 — مسار `/excel-ingest` غير مُسجَّل في الخادم إطلاقاً
- الملف: `services/backend/main.go:702-712` — مجموعة `/messaging` تسجّل:
  `/send`, `/bulk-send`, `/bulk/:id/progress`, `/ingest`, `/list`,
  `/bulk-delete`, `/bulk-resend`, `/incoming` — **ولا يوجد `/excel-ingest`**.
- المعالج `BulkIngestExcelHandler` موجود في `internal/receiver/handler.go:194`
  لكنه **غير مربوط بأي route**.
- (سجلات قديمة مثل `log.txt` كانت تعرض المسار، أي أنه كان مسجلاً سابقاً ثم فقد خلال تعديلات main.go.)

### المشكلة 3 — مسار `/ingest` (التراثي) معطوب جوهرياً حتى للرسالة المفردة
- `internal/receiver/handler.go:151` `IngestMessageHandler` →
  `internal/receiver/service.go:404` `IngestIncomingMessage`:
  يخزّن الـ payload كاملاً في `RawPayload` فقط، ويترك `PhoneNumber` و`MessageContent` فارغين.
- `internal/queue/jobs.go:501` `AnalysisWorker` يقرأ `incoming.PhoneNumber` و
  `incoming.MessageContent` (فارغان دائماً) → يحاول إنشاء `Message` بمحتوى/رقم فارغ
  → يرفضه القيدان `messages_content_not_empty` / `messages_phone_number_not_empty`
  → فشل أبدي + إعادة محاولة (حتى لو أُرسلت رسالة محاسبية مفردة صحيحة فستفشل).
- كما أنه حتى لو نجح، ينشئ **رسالة واحدة فقط** وليس مصفوفة الـ 183 — أي أنه لا يصلح
  للاستيراد الجماعي إطلاقاً.

### المشكلة 4 — إهدار اسم المُرسِل
- `internal/receiver/handler.go:165`: `c.DefaultQuery("sender_name", "Accounting System")`
  بينما الواجهة ترسله داخل الـ body → يُفقد ويظهر «Accounting System».
- وفي `BulkIngestExcelHandler` (handler.go:213) يُثبَّت `SenderName: "Excel Import"`
  بدلاً من قراءته من الطلب.

---

## 6. الأثر

- **183 رسالة** من تجربة المستخدم الأخيرة لم تُجدول إطلاقاً (ولم تُرسل).
- عمليات إكسل سابقة (اليوم 20:29 و2026-08-02) عالقة بنفس الطريقة.
- طاقة حاسوبية مهدرة: مهمة `river_job` تتعطل وتُعاد محاولتها بلا جدوى (job 59 أعادت 14 مرة).
- المستخدم لا يرى أي أثر في الواجهة رغم «النجاح» الوهمي → ثقة مكسورة بالنظام.

---

## 7. خطة الإصلاح المقترحة

### الإصلاح 1 (خادم) — تسجيل المسار الجماعي
في `services/backend/main.go` داخل مجموعة `/messaging`:
```go
messagingGroup.POST("/excel-ingest", receiverHandler.BulkIngestExcelHandler)
```

### الإصلاح 2 (واجهة) — توجيه تبويب إكسل للمسار الصحيح
في `apps/client-web/src/components/ExcelImport.jsx:117`:
```js
await apiClient.post('/api/v1/messaging/excel-ingest', {
    messages: ingestionPayload.messages,   // أو البنية المتوقعة من BulkIngestExcelHandler
    message_type: '',
});
```
> ملاحظة: بنية `/excel-ingest` المتوقعة هي `{ messages: [...], message_type?: string }`.

### الإصلاح 3 (خادم، معالجة الجذر التراثي) — إصلاح مسار `/ingest`
إحدى الطريقتين (يُفضّل تنفيذ كلتيهما معاً):
- **أ**: في `IngestIncomingMessage`/`IngestBulkIncomingMessages`: استخراج
  `phone_number`/`content` من الـ payload وتعبئة العمودين قبل الحفظ.
- **ب**: في `AnalysisWorker`: إذا كان `MessageContent`/`PhoneNumber` فارغين،
  التفكيك من `RawPayload` كحل احتياطي.

### الإصلاح 4 (جودة) — قراءة `sender_name` من الـ body/المصدر الصحيح
- تعديل `IngestMessageHandler` و`BulkIngestExcelHandler` لأخذ `sender_name` من الـ payload
  مع خيار query كاحتياطي.

---

## 8. خطوات التحقق المقترحة (بعد الإصلاح)

1. إعادة بناء الباكند وتشغيله، ثم `POST /api/v1/messaging/excel-ingest` بـ 3 رسائل تجريبية.
2. التأكد من: صف `global_incoming_events` + 3 صفوف في `messages` بحالة PENDING.
3. التأكد من ظهورها في `GET /api/v1/messaging/list` (تبويب الرسائل).
4. التأكد من سحبها عبر `GET /api/v1/devices/fetch` وإرسالها من الهاتف وتقارير SENT/DELIVERED.
5. إعادة محاولة المستخدم مع ملفه (183) والتأكد من اكتمال المعالجة.

---

## 9. الملفات ذات الصلة

| الملف | الدور |
|---|---|
| `apps/client-web/src/components/ExcelImport.jsx` | تبويب الإرسال من إكسل (يستدعي `/ingest` — خاطئ) |
| `services/backend/main.go` (702-712) | تسجيل مسارات `/messaging` (ينقصه `/excel-ingest`) |
| `services/backend/internal/receiver/handler.go` | `IngestMessageHandler` (151) و`BulkIngestExcelHandler` (194) |
| `services/backend/internal/receiver/service.go` | `IngestIncomingMessage` (404) / `IngestBulkIncomingMessages` (430) / `IngestGlobalEvent` (349) |
| `services/backend/internal/queue/jobs.go` | `GlobalEventWorker` (189) و`AnalysisWorker` (501) |
| `services/backend/internal/queue/client.go` | تسجيل العاملين (شامل GlobalEventWorker ✓) |
