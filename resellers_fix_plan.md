# خطة الإصلاح الكاملة — منظومة الوكلاء (Software Companies / Resellers)

> المرجع: `C:\ALHudhud\ALHudhud\ALHudhudAI`
> آخر تحديث: بعد توضيح المستخدم لوظيفة التبويب.

---

## 0) مفهوم تبويب الوكلاء (المتطلبات — المتفق عليها)

كل **وكيل** (شركة برمجية / Reseller) يتم إنشاؤه سيكون قادرًا على:

1. **إنشاء مستأجرين** (شركات عميل `Tenant`) مرتبطين به عبر `tenants.parent_id = الوكيل`.
2. **تحويل مبالغ مالية إلى محافظ مستأجريه** (خصم من محفظة الوكيل + إيداع في محفظة المستأجر).
3. **تسديد اشتراك مستأجريه** — تفعيل/تجديد اشتراك للمستأجر، **تُخصم قيمته من محفظة المستأجر** (يسبقها تحويل من الوكيل لتمويلها).

### القرارات التصميمية المعتمدة
| القرار | الاختيار | الأثر |
|---|---|---|
| مصدر خصم الاشتراك | **محفظة المستأجر** | إعادة استخدام `subscription.Manager.Subscribe` كما هو (نفس منطق الخصم الحالي) |
| تمويل محفظة الوكيل | **المشرف** | endpoint إداري جديد لإيداع رصيد في محفظة الوكيل |
| عزل البيانات | الوكيل يتحكم فقط في مستأجريه (`parent_id`) | فحص ملكية في كل عملية |

---

## 1) الوضع الحالي

### تم إنجازه في الجلسة السابقة (✅)
| العنصر | الملف |
|---|---|
| `GET /api/v1/admin/resellers` (قائمة الوكلاء + عدد الشركات المرتبطة) | `internal/reseller/service.go` + `handler.go` + `main.go:761` |
| دعم إنشاء وكيل `is_reseller` في `AdminCreateTenant` | `internal/tenant/service.go` |
| صفحة `ResellersList.jsx` + التبويب + المسار `/resellers` | `apps/admin-web/src/pages/Resellers/ResellersList.jsx` + `App.jsx` |
| `RESELLER_PORTAL_URL` (5175) | `apps/admin-web/src/config.js` |
| مفتاح `nav.resellers` عربي/إنجليزي | `apps/admin-web/src/locales/{en,ar}.json` |
| التحقق: `go build` + `vite build` | — |

### ملاحظة الترجمة
ملفات `ar.json` في admin-web و client-web **سليمة (UTF-8 عربي صحيح)** — «التشوه» كان خطأ عرض فقط، لا إصلاح مطلوب.

---

## 2) البنية المستهدفة

```
[واجهة الإدارة admin-web]
   │ GET /admin/resellers          قائمة الوكلاء + رصيد + عدد الشركات
   │ GET /admin/resellers/:id      تفاصيل + مستأجروه
   │ POST /admin/resellers/:id/credit   ← تمويل محفظة الوكيل (المشرف)
   │ POST /admin/tenants           إنشاء وكيل (is_reseller=true)
   ▼
[الخلفية Go (8085)]
   │
   │== مسارات الوكيل /api/v1/reseller/* (RESELLER / SUPER_ADMIN) ==
   │ GET  /tenants                          قائمة مستأجريه (مع الرصيد + حالة الاشتراك)
   │ POST /tenants                          إنشاء مستأجر (مفاتيح API + schema)
   │ GET  /tenants/:id/balance              رصيد المستأجر
   │ POST /tenants/:id/transfer             تحويل مبلغ لمحفظة المستأجر  ← (خصم الوكيل + إيداع المستأجر)
   │ POST /tenants/:id/subscribe            تسديد اشتراك المستأجر       ← (يُخصم من محفظة المستأجر)
   │ POST /tenants/:id/api-key              تجديد مفتاح API للمستأجر
   │ GET  /packages                         قائمة الباقات (لاختيار الاشتراك)
   │ GET  /subscriptions/active             اشتراك الوكيل نفسه
   │ GET  /subscriptions/usage              استخدام الوكيل
   ▼
[محافظ (balance)]  ← نضيف: إيداع عام (CreditBalance) + تحويل ثنائي (Transfer)
```

---

## 3) الفجوات المكتشفة

| # | الفجوة | المرجع | الأثر |
|---|---|---|---|
| G1 | `CreateTenant` (reseller) **لا يولّد مفاتيح API** ولا **ينشئ schema** للمستأجر | `internal/reseller/service.go:55-80` | المستأجر الجديد غير قابل للعمل |
| G2 | `CreateTenant` حقول افتراضية ناقصة (serial, country, phone...) | نفسه | بيانات ناقصة |
| G3 | **لا عملية عامة لإيداع/تحويل محفظة** — `balance.Manager` يدعم الخصم والاسترجاع فقط (استرجاع مقيد بـ message_id) | `internal/balance/manager.go` | لا يمكن تنفيذ «تحويل المبالغ» ولا «تمويل الوكيل» |
| G4 | **لا endpoint لتحويل الوكيل → محفظة المستأجر** | `internal/reseller/` | الميزة 2 غير موجودة |
| G5 | **لا endpoint لتسديد اشتراك مستأجر من الوكيل** (مع فحص الملكية) | `internal/reseller/` | الميزة 3 غير موجودة |
| G6 | **لا endpoint إداري لتمويل محفظة الوكيل** | `main.go` | مصدر أموال الوكيل غير موجود |
| G7 | بوابة الموزع بلا واجهة إنشاء مستأجر / تحويل / اشتراك (قراءة فقط + إحصائيات وهمية + تبويبات ميتة + بلا i18n) | `apps/reseller-web/src/App.jsx` | لا يمكن للوكيل فعل أي شيء من الوظائف الثلاث |
| G8 | قائمة الشركات الإدارية `ListTenants` لا تستثني الوكلاء | `internal/tenant/service.go:57` | الوكلاء يختلطون بشركات العملاء |
| G9 | لا صفحة تفاصيل وكيل في الإدارة (مستأجروه + رصيده + تمويله) | `apps/admin-web/src/pages/Resellers/` | لا إشراف مالي |
| G10 | صفحة الوكلاء نصوص إنجليزية ثابتة (بلا i18n) | `ResellersList.jsx` | عدم اتساق |
| G11 | لا اختبارات للـ reseller | `internal/reseller/` | لا حماية من الانحدار |
| G12 | لا migration صريح لعمودي `is_reseller`/`parent_id` (AutoMigrate) | `internal/models/models.go:59-61` | تأكيد الـ schema مطلوب |

---

## 4) خطة الإصلاح على مراحل

### المرحلة 1 — عمليات المحفظة (Backend · balance) — أولوية عالية
**الملفات: `internal/balance/manager.go` (+ interface في بداية الملف) + `internal/reseller/`**

**1.1 إضافة `CreditBalance` عامة لـ `balance.Manager`**
- التوقيع المقترح: `CreditBalance(ctx, op BalanceOperation) (*BalanceResult, error)` (لا يتطلب message_id).
- منطق مشابه لـ `RefundBalance` لكن لأي سبب: `UPDATE tenants SET balance = balance + $1 ... RETURNING` + سجل تاريخ + إدخال `accounting_transactions` بنوع `'CREDIT'` + audit + إشعار.
- تُستخدم لتمويل محفظة الوكيل (المشرف) ولجانب الإيداع من التحويل.

**1.2 إضافة `TransferBalance` (تحويل بين محفظتين في معاملة واحدة)**
- التوقيع المقترح: `TransferBalance(ctx, from, to uuid.UUID, amount decimal.Decimal, reason, createdBy string) (*TransferResult, error)`.
- في `pgx` transaction واحدة:
  1. خصم آمن من `from`: `UPDATE tenants SET balance = balance - $1 WHERE id=$2 AND balance >= $1 AND deleted_at IS NULL RETURNING balance` (يفشل عند الرصيد غير الكافي).
  2. إيداع في `to`: `UPDATE tenants SET balance = balance + $1 ... RETURNING balance`.
  3. سجل تاريخ للطرفين (`history_logger`) + إدخالا محاسبة (`TRANSFER_OUT`/`TRANSFER_IN`) + audit + RLS للطرفين.
- **معيار القبول:** التحويل إما أن يتم كاملًا أو لا يحدث أبدًا؛ لا سحب جزئي عند الفشل.

### المرحلة 2 — استكمال الـ reseller backend — أولوية عالية
**الملفات: `internal/reseller/service.go` + `handler.go` + `main.go`**

**2.1 إصلاح `CreateTenant` (G1 + G2)**
- توليد مفاتيح ACC/PHO/WEB + `db.CreateTenantSchema`/`MigrateTenantSchema` + حقول افتراضية (serial `HUD-...`, country, address, sector, status) + ربط `PhoneNumber` — بنمط `internal/tenant/service.go:448-489`.

**2.2 Endpoints الوكيل الجديدة** (كلها داخل `resellerGroup` مع فحص الملكية `parent_id = الوكيل`):
- `GET /reseller/tenants/:id/balance` — رصيد المستأجر (عبر `balance.GetBalance`).
- `POST /reseller/tenants/:id/transfer` — `{ amount, note }` → `TransferBalance(الوكيل، المستأجر)`.
- `POST /reseller/tenants/:id/subscribe` — `{ package_id, auto_renew }` → فحص ملكية ثم `subscription.Manager.Subscribe{ TenantID: المستأجر }` (يُخصم من محفظة المستأجر — القرار المعتمد) + تعطيل الاشتراكات السابقة (نمط `admin_subscription_handler.go:108-123`).
- `GET /reseller/tenants` (الحالي) — توسيع الاستجابة لتشمل `balance` + `subscription` الحالة (استعلام فرعي أو join).

**2.3 Endpoint إداري لتمويل الوكيل (G6)**
- `POST /admin/resellers/:id/credit` — `{ amount, note }` → `CreditBalance(محفظة الوكيل)`، مقيّد بـ SUPER_ADMIN، مسجّل في audit.

**2.4 فصل الوكلاء عن شركات العملاء (G8)**
- `internal/tenant/service.go:57` — `ListTenants` يفلتر `is_reseller = false` افتراضيًا.

### المرحلة 3 — بوابة الموزع (reseller-web) — أولوية عالية ✅
**الملف الرئيسي: `apps/reseller-web/src/App.jsx`**

**3.1 تبويب Customers — استكمال الوظائف الثلاث:**
- إنشاء مستأجر: نموذج → `POST /reseller/tenants`.
- صف المستأجر يعرض: الرصيد + حالة الاشتراك + `api_key_accounting` (نسخ/تجديد).
- زر **"Transfer"** → نافذة `{ amount, note }` → `POST /reseller/tenants/:id/transfer`.
- زر **"Subscribe"** → نافذة اختيار باقة (من `GET /reseller/packages`) → `POST /reseller/tenants/:id/subscribe`.
- إظهار **رصيد محفظة الوكيل** في الترويسة/لوحة المعلومات.

**3.2 لوحة معلومات حقيقية (G7)**
- أرقام فعلية: عدد المستأجرين، عدد الفريق، الاستخدام (`/subscriptions/usage`)، رصيد المحفظة.

**3.3 تنظيف**
- إزالة/إصلاح تبويبي `api` و `settings` الميتين، وزر Invite Staff غير الفعّال (أو تأجيل مع إزالة الزر).
- **الحالة: مُنفّذ.** حُذف قسم "Integrations" وعنصر `Key` المكرر من الشريط الجانبي (كان يفتح نفس تبويب Customers)، وزر Invite Staff غير موجود (إدارة الفريق قراءة فقط)، و`npm run build` لـ reseller-web خضراء.

**3.4 i18n + RTL**
- إنشاء `src/locales/{en,ar}.json` + `src/i18n.js` بنمط `admin-web`، و`dir="rtl"` للعربية.
- **الحالة: مُنفّذ.** `i18n.js` يطبّق `dir=rtl`/`lang` عند تغيير اللغة، الافتراضي `ar`.

### المرحلة 4 — واجهة الإدارة (admin-web) — أولوية متوسطة ✅
**3.1 صفحة تفاصيل الوكيل `/resellers/:id`** (`pages/Resellers/ResellerDetails.jsx`)
- معلومات الوكيل + **رصيد محفظته** + عدد مستأجريه.
- جدول مستأجريه (من `GET /admin/resellers/:id`): الاسم، الرصيد، حالة الاشتراك، المفاتيح.
- زر **"Fund Wallet"** → `POST /admin/resellers/:id/credit`.
- ربط الصفوف بالتنقل من `ResellersList.jsx`.
- **الحالة: مُنفّذ.** `ResellerDetails.jsx` جديد + مسار `/resellers/:id` في `App.jsx` + `ResellersList.jsx` مُعادت كتابته بالتوطين والتنقل وعمود المحفظة + مفاتيح `resellers.*`.

**3.2 توطين الصفحة** (مفاتيح `resellers.*` في en/ar).

### المرحلة 5 — البيانات — أولوية متوسطة ✅
- تأكيد وجود عمودي `is_reseller`/`parent_id` في `tenants` (`information_schema.columns`).
- **الحالة: مُنفّذ.** الأعمدة موجودة (boolean / uuid، nullable) في `sms_gateway_test`؛ وتم استبدال السيناريو اليدوي باختبار تكامل E2E آلي (أدناه). `accounting_transactions.message_id` nullable → لا عائق لإيداعات CREDIT.
- إنشاء أول وكيل عبر الواجهة، ثم: المشرف يموّله → الوكيل يدخل البوابة 5175 → ينشئ مستأجرًا → يحوّل له → يفعّل اشتراكه → الشركة تستطيع الاتصال بمفتاحها.

### المرحلة 6 — الاختبار والتحقق ✅
| الاختبار | الطريقة |
|---|---|
| وحدة المحفظة | `go test ./internal/balance/...` (CreditBalance + TransferBalance، حالات: رصيد غير كافٍ، توافقية/توازٍ، عزل) — **أُضيفت 8 اختبارات في `transfer_credit_test.go`** |
| وحدة الـ reseller | `go test ./internal/reseller/...` (ملكية المستأجر، انتقال الأموال) |
| **تكامل E2E** | **جديد:** `integration_test.go` — دورة كاملة بمدراء حقيقيين: تمويل الوكيل (500) → إنشاء مستأجر (مفاتيح + serial + schema حقيقي) → تحويل (200) → اشتراك بباقة حقيقية (خصم 50 من محفظة المستأجر) → `ListTenantsWithDetails` يعرض `150` + `active` → عزل ملكية (وكيل آخر مرفوض) |
| بناء | `go build ./...` + `npm run build` لـ admin-web و reseller-web — كلها خضراء |
| E2E يدوي | المشرف يُنشئ ويموّل وكيلًا → الوكيل ينشئ مستأجرًا → يحوّل → يسدد الاشتراك → الشركة تتصل |
| أمان | محاولة الوكيل لمس مستأجر ليس من `parent_id` له → مرفوض (401/403) — مُغطاة آليًا |

---

## 5) الترتيب والتقدير

| # | المهمة | التقدير |
|---|---|---|
| 1 | المرحلة 1 (CreditBalance + TransferBalance) | 2-3 ساعات |
| 2 | المرحلة 2 (إنشاء/تحويل/اشتراك/تمويل/فلترة) | 3-4 ساعات |
| 3 | المرحلة 3 (بوابة الموزع) | 3-4 ساعات |
| 4 | المرحلة 4 (تفاصيل الوكيل + توطين) | 1-2 ساعة |
| 5 | المرحلة 5 + 6 (بيانات + تحقق) | 1-2 ساعة |

**الإجمالي التقريبي: 10-15 ساعة عمل.**

---

## 6) معايير القبول النهائية
1. الوكيل ينشئ مستأجرين بمفاتيح API صالحة + schema خاص — ✅ (مُثبت في `TestResellerFullFlow_Integration`).
2. الوكيل يحوّل مبلغًا من محفظته إلى محفظة المستأجر (ضمن معاملة واحدة آمنة) — ✅.
3. الوكيل يسدد اشتراك مستأجره ويُخصم المبلغ من **محفظة المستأجر** — ✅.
4. المشرف يموّل محفظة الوكيل من صفحة تفاصيله ويرى أرصدة الجميع — ✅.
5. الوكلاء لا يظهرون في صفحة Companies العادية — ✅.
6. بوابة الموزع + واجهة الإدارة: عربي سليم (UTF-8) و RTL في الموزع — ✅.
7. كل الاختبارات خضراء (go test + vite build) — ✅ (`go vet ./...` نظيف، `go test ./internal/reseller/...` و `./internal/balance/...` خضراء).

---

## 7) المخاطر والتنبيهات
- **تحويل الأموال** يتطلب معاملة ذرّية حقيقية عبر `pgx` وليس GORM — يجب مراعاة RLS لكل محفظة (ضبط `app.current_tenant` لكل من الطرفين داخل المعاملة، كما في `subscription/manager.go:75`).
- **`accounting_transactions.message_id`** قد يكون NOT NULL — تحقق من الـ schema قبل إدراج إدخالات `CREDIT`/`TRANSFER` بلا message_id، أو اجعلها قابلة للصفر.
- **تمويل الوكيل دون سقف** قد يكون خطرًا ماليًا — **مُعالج ✅**: سقف قابل للضبط لكل عملية تمويل عبر `RESELLER_MAX_CREDIT` (افتراضي: 0 = بلا سقف)؛ الطلبات الأكبر تُرفض بـ `ErrCreditLimitExceeded` (HTTP 400) مع تسجيل تحذيري في السجل، و3 اختبارات جديدة في `service_test.go` (تجاوز/حد/بلا سقف).
- لا endpoint لإضافة staff — إبقاء زر Invite معطّلاً أو إزالته حتى يُنفذ لاحقًا.
