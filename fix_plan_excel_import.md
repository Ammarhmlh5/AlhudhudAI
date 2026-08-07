# خطة إصلاح مشكلة الإرسال من ملف إكسل (Excel Import)

**التاريخ:** 2026-08-03
**المرجع:** `docs/EXCEL_IMPORT_INVESTIGATION_REPORT.md` (تقرير التحقيق المفصّل)
**الحالة:** مقترحة — لم تُنفَّذ بعد

---

## 1. ملخص

المشكلة: إرسال ملف إكسل من تبويب «إرسال عبر ملف إكسل» في واجهة المستأجرين يعرض
«نجاحاً» لكن **لا تظهر الرسائل في الويب ولا تُرسل إلى الهاتف**.

السبب: الواجهة تستدعي المسار الخاطئ `/api/v1/messaging/ingest` (مسار الرسائل الخام
المفردة)، بينما المسار الجماعي الصحيح `/api/v1/messaging/excel-ingest` **غير مسجَّل
أساساً في الخادم**. وبإضافة إلى ذلك، مسار `/ingest` التراثي معطوب جوهرياً ولا يملأ
`phone_number`/`content` فيفشل دائماً في قيود قاعدة البيانات.

هذه الخطة تعالج: المسار، البنية، المعالجة التراثية، واسم المُرسِل.

---

## 2. جذور المشكلة والحلول

### المشكلة 1 — مسار `/excel-ingest` غير مسجَّل في الخادم (حرجة)
- **الملف:** `services/backend/main.go` (مجموعة `/messaging`، الأسطر 702-712)
- **الوصف:** المعالج `BulkIngestExcelHandler` موجود في
  `internal/receiver/handler.go:194` لكنه غير مربوط بأي route.
- **الحل:** إضافة السطر داخل مجموعة `/messaging`:
  ```go
  messagingGroup.POST("/excel-ingest", receiverHandler.BulkIngestExcelHandler)
  ```

### المشكلة 2 — الواجهة تستدعي مساراً خاطئاً وبنية خاطئة (حرجة)
- **الملف:** `apps/client-web/src/components/ExcelImport.jsx` (الأسطر 99-117)
- **الوصف:** ترسل `/api/v1/messaging/ingest` ببنية
  `{ api_key, sender_name, send_type, messages }`. المسار الصحيح هو
  `/api/v1/messaging/excel-ingest` ويتوقع بنية `{ messages, message_type }`.
- **الحل:** تعديل `handleUpload`:
  ```js
  const ingestionPayload = {
      messages: excelData.map(row => ({
          phone_number: row[mapping.phone] ? String(row[mapping.phone]) : '',
          content: row[mapping.message] ? String(row[mapping.message]) : '',
      })).filter(item => item.phone_number && item.content)
  };

  await apiClient.post('/api/v1/messaging/excel-ingest', ingestionPayload);
  ```

### المشكلة 3 — مسار `/ingest` التراثي معطوب جوهرياً (خطرة، إصلاح وقائي)
- **الملف:** `internal/receiver/service.go:404` (`IngestIncomingMessage`) و
  `internal/queue/jobs.go:501` (`AnalysisWorker`)
- **الوصف:** يُخزَّن الـ payload كاملاً في `RawPayload` دون تعبئة `PhoneNumber`/
  `MessageContent`، والـ `AnalysisWorker` يقرأ هذين الحقلين الفارغين فينشئ رسالة
  بمحتوى فارغ ترفضها القيود `messages_content_not_empty`/`messages_phone_number_not_empty`
  → مهمة `message_analysis` تتعطل للأبد.
- **الحل (يُنفَّذ الثنائي):**
  - **أ)** في `IngestIncomingMessage`: بعد قراءة `rawJSON`، تحليل الـ JSON واستخراج
    `phone_number` و`content` (إن وُجدا في المستوى الأعلى) وتعبئة العمودين قبل الحفظ.
  - **ب)** في `AnalysisWorker`: احتياطياً، إذا كان `incoming.MessageContent`/
    `incoming.PhoneNumber` فارغين، محاولة فكّ `incoming.RawPayload` واستخراجهما.

### المشكلة 4 — إهدار اسم المُرسِل (متوسطة)
- **الملف:** `internal/receiver/handler.go:151` و`194`
- **الوصف:** `IngestMessageHandler` يقرأ `sender_name` من query param بينما الواجهة
  ترسله في الـ body → «Accounting System». و`BulkIngestExcelHandler` يُثبّت
  `"Excel Import"` بلا قراءة من الطلب.
- **الحل:** قراءة `sender_name` من الـ body أولاً مع الاحتفاظ بـ query كاحتياطي
  في كلا المعالجين.

---

## 3. خطة التنفيذ (بالترتيب)

### الخطوة 1 — الخادم: تسجيل المسار
- تعديل `main.go` لإضافة `POST /excel-ingest`.
- إعادة البناء: `go build -o backend_new.exe .` (داخل `services/backend`).

### الخطوة 2 — الواجهة: توجيه التبويب للمسار الصحيح
- تعديل `ExcelImport.jsx` (المسار + البنية).
- إعادة بناء `client-web` ونشر التحديث.

### الخطوة 3 — الخادم: إصلاح المسار التراثي (وقائي)
- تعديل `service.go` (`IngestIncomingMessage`) و`jobs.go` (`AnalysisWorker`).
- إعادة البناء والتشغيل.

### الخطوة 4 — الخادم: اسم المُرسِل
- تعديل `handler.go` لكلا المعالجين.
- إعادة البناء والتشغيل.

> **ملاحظة تنفيذ:** الخطوات 1 و2 كافيتان لحل مشكلة المستخدم. الخطوات 3 و4
> إصلاحات وقائية/جودة لمنع تكرار المشكلة عبر مسارات أخرى.

---

## 4. خطة التحقق (Verification)

1. **اختبار واجهة (API):** إرسال `POST /api/v1/messaging/excel-ingest` بثلاث رسائل
   تجريبية عبر `curl` بمصادقة صالحة.
2. **قاعدة البيانات:**
   - صف جديد في `global_incoming_events` بحالة `PROCESSING`.
   - ثلاثة صفوف في `messages` بحالة `PENDING` (بأرقام/محتوى صحيحة).
3. **الويب:** ظهورها في `GET /api/v1/messaging/list` (تبويب الرسائل).
4. **الهاتف:** سحبها عبر `GET /api/v1/devices/fetch`، ثم إرسالها من التطبيق
   (test_mode) وتقارير SENT/DELIVERED.
5. **إعادة محاولة المستخدم:** لصق ملفه (183 رسالة) والتأكد من اكتمال المعالجة
   ووصولها للهاتف وتبويب الرسائل.
6. **عدم التكرار التراثي:** التأكد من أن مهمة `message_analysis` لم تعد تفشل
   بخطأ `messages_content_not_empty`.

---

## 5. خطة الاسترجاع (Rollback)

- **الخادم:** الاحتفاظ بنسخة `backend_new.exe` القديمة قبل إعادة البناء —
  عند أي تراجع تُستعاد النسخة ويُعاد التشغيل.
- **الواجهة:** التعديل في ملف واحد (`ExcelImport.jsx`) — يمكن الرجوع عبر
  git (`git checkout -- <file>`).
- لا توجد تغييرات على مخطط قاعدة البيانات أو الترحيلات (Migrations).

---

## 6. الملفات المعنية

| الملف | التغيير |
|---|---|
| `services/backend/main.go` | إضافة route `/excel-ingest` |
| `services/backend/internal/receiver/handler.go` | قراءة `sender_name` من الـ body (المعالجان) |
| `services/backend/internal/receiver/service.go` | `IngestIncomingMessage`: تعبئة `phone_number`/`content` من الـ raw |
| `services/backend/internal/queue/jobs.go` | `AnalysisWorker`: فكّ `RawPayload` احتياطياً |
| `apps/client-web/src/components/ExcelImport.jsx` | توجيه `/excel-ingest` + بنية `{ messages }` |
