# خطة نظام الحسابات — سندات القبض والصرف والصناديق

الموقع: تبويب **العمليات المالية** (لوحة الإدارة)
الحالة: خطة مفصّلة — لم تُنفَّذ بعد

---

## 1. الأهداف

بناء نظام حسابات متكامل داخل تبويب "العمليات المالية" يشمل:

1. **الصناديق** (Cashboxes): صندوق النقد، الحسابات البنكية، محافظ الدفع — لكل صندوق رصيد مستقل.
2. **سندات القبض** (Receipt Vouchers): توثيق المبالغ **الداخلة** إلى النظام/الصندوق.
3. **سندات الصرف** (Payment Vouchers): توثيق المبالغ **الخارجة** من الصندوق.
4. **سجل حركة الصندوق** (Cashbox Statement): دفتر يعرض كل حركة (قبض/صرف) مع الرصيد اللحظي.
5. (مرحلة لاحقة) **دفتر اليومية والأستاذ**: حسابات + قيود مزدوجة القيد.

الأساس: كل حركة يجب أن تكون **موثّقة برقم سند تسلسلي**، تؤثر على رصيد صندوق واحد، مع إمكانية **الإلغاء** (عكس) بأمان.

---

## 2. الوضع الحالي (ما يوجد بالفعل)

| العنصر | الحالة |
|---|---|
| `balance_history` | سجل أرصدة المحافظ (DEDUCT/REFUND/CHARGE/ADJUSTMENT) — سجل أحادي القيد |
| `accounting_transactions` | تسوية الرسائل/الخدمات (مع `reference_type/reference_id`) |
| `billing_invoices` + `payment_transactions` | فواتير ومدفوعات بوابات الدفع |
| `bank_accounts` | جدول موجود في قاعدة البيانات **بدون** model أو API أو واجهة |
| FinanceManager.jsx | واجهة **تجريبية (mock)** بلا أي اتصال بالخادم |
| سند قبض / سند صرف / صندوق / دفتر يومية | **غير موجود نهائياً** — بناء جديد بالكامل |

نقاط مهمة تُصلَح ضمن التنفيذ:
- مسار `POST /admin/tenants/:id/balance` (إضافة رصيد من صفحة الشركة) **غير مسجَّل** في main.go → يعمل مكسوراً.
- `payment_handler.go` كامل لكن **لا يوجد في** main.go (لم يُسجَّل).
- `Tenant.Balance` GORM `decimal(12,2)` بينما قاعدة البيانات `DECIMAL(12,4)`.

---

## 3. تصميم قاعدة البيانات (مخططات جديدة 093+)

### 3.1 الصناديق `cashboxes`

يمثل الصندوق "مكاناً مالياً": نقدي، بنكي، أو محفظة.

```sql
CREATE TABLE IF NOT EXISTS cashboxes (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code             VARCHAR(20)  NOT NULL UNIQUE,          -- مثل CASH-01, BANK-01
    name             VARCHAR(100) NOT NULL,                 -- الاسم بالعربية
    name_en          VARCHAR(100),                          -- الاسم بالإنجليزية
    type             VARCHAR(20)  NOT NULL DEFAULT 'CASH'
                     CHECK (type IN ('CASH','BANK','WALLET','OTHER')),
    currency         VARCHAR(3)   NOT NULL DEFAULT 'SAR',
    bank_name        VARCHAR(100),                          -- للصندوق البنكي
    account_holder   VARCHAR(100),
    account_number   VARCHAR(50),
    iban             VARCHAR(34),
    opening_balance  DECIMAL(15,4) NOT NULL DEFAULT 0,      -- رصيد افتتاحي
    current_balance  DECIMAL(15,4) NOT NULL DEFAULT 0,      -- الرصيد الحالي (يُحدَّث تلقائياً)
    min_balance      DECIMAL(15,4) NOT NULL DEFAULT 0,      -- حد إنذار انخفاض الرصيد
    status           VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE'
                     CHECK (status IN ('ACTIVE','INACTIVE')),
    description      TEXT,
    created_by       UUID,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now()
);
```

### 3.2 مولّد الأرقام التسلسلية `voucher_counters`

يضمن تسلسل أرقام السندات بلا تكرار وبطريقة ذرّية (لا يعتمد على MAX + 1).

```sql
CREATE TABLE IF NOT EXISTS voucher_counters (
    voucher_type VARCHAR(20) NOT NULL CHECK (voucher_type IN ('RECEIPT','PAYMENT')),
    fiscal_year  INT          NOT NULL,
    last_number  INT          NOT NULL DEFAULT 0,
    PRIMARY KEY (voucher_type, fiscal_year)
);
```

صيغة الرقم: `QABD-2026-000001` للقبض، `SARF-2026-000001` للصرف.

### 3.3 سندات القبض `receipt_vouchers`

```sql
CREATE TABLE IF NOT EXISTS receipt_vouchers (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    voucher_number   VARCHAR(40) NOT NULL UNIQUE,
    voucher_date     DATE        NOT NULL DEFAULT CURRENT_DATE,
    cashbox_id       UUID        NOT NULL REFERENCES cashboxes(id),
    amount           DECIMAL(15,4) NOT NULL CHECK (amount > 0),
    currency         VARCHAR(3)  NOT NULL DEFAULT 'SAR',
    payer_name       VARCHAR(200) NOT NULL,                 -- المدفوع منه
    payer_type       VARCHAR(20) NOT NULL DEFAULT 'OTHER'
                     CHECK (payer_type IN ('TENANT','RESELLER','CUSTOMER','OTHER')),
    tenant_id        UUID,                                  -- إذا كان من شركة/مستأجر
    reseller_id      UUID,                                  -- إذا كان من وكيل
    payment_method   VARCHAR(20) NOT NULL DEFAULT 'CASH'
                     CHECK (payment_method IN ('CASH','BANK_TRANSFER','CHEQUE','CARD','WALLET')),
    reference_number VARCHAR(100),                          -- رقم التحويل/الشيك
    description      TEXT,
    status           VARCHAR(20) NOT NULL DEFAULT 'POSTED'
                     CHECK (status IN ('DRAFT','POSTED','REVERSED')),
    reversal_of      UUID,                                  -- مرجع عند العكس
    credited_wallet  BOOLEAN     NOT NULL DEFAULT FALSE,    -- هل أُضيف لمحفظة الشركة/الوكيل؟
    created_by       UUID,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    posted_at        TIMESTAMPTZ,
    posted_by        UUID,
    reversed_at      TIMESTAMPTZ,
    reversed_by      UUID
);
```

### 3.4 سندات الصرف `payment_vouchers`

مطابقة للسندات أعلاه مع الاختلاف في الدلالة (صرف/خارج):

```sql
CREATE TABLE IF NOT EXISTS payment_vouchers (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    voucher_number   VARCHAR(40) NOT NULL UNIQUE,
    voucher_date     DATE        NOT NULL DEFAULT CURRENT_DATE,
    cashbox_id       UUID        NOT NULL REFERENCES cashboxes(id),
    amount           DECIMAL(15,4) NOT NULL CHECK (amount > 0),
    currency         VARCHAR(3)  NOT NULL DEFAULT 'SAR',
    payee_name       VARCHAR(200) NOT NULL,                 -- المدفوع له
    payee_type       VARCHAR(20) NOT NULL DEFAULT 'OTHER'
                     CHECK (payee_type IN ('TENANT','RESELLER','SUPPLIER','OTHER')),
    tenant_id        UUID,
    reseller_id      UUID,
    payment_method   VARCHAR(20) NOT NULL DEFAULT 'CASH'
                     CHECK (payment_method IN ('CASH','BANK_TRANSFER','CHEQUE','CARD','WALLET')),
    reference_number VARCHAR(100),
    description      TEXT,
    status           VARCHAR(20) NOT NULL DEFAULT 'POSTED'
                     CHECK (status IN ('DRAFT','POSTED','REVERSED')),
    reversal_of      UUID,
    debited_wallet   BOOLEAN     NOT NULL DEFAULT FALSE,
    created_by       UUID,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    posted_at        TIMESTAMPTZ,
    posted_by        UUID,
    reversed_at      TIMESTAMPTZ,
    reversed_by      UUID
);
```

### 3.5 سجل حركة الصندوق `cashbox_movements`

يسجّل كل حركة (قبض أو صرف) على كل صندوق مع **الرصيد بعد الحركة** — هو أساس كشف حساب الصندوق.

```sql
CREATE TABLE IF NOT EXISTS cashbox_movements (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cashbox_id     UUID NOT NULL REFERENCES cashboxes(id),
    voucher_type   VARCHAR(20) NOT NULL CHECK (voucher_type IN ('RECEIPT','PAYMENT')),
    voucher_id     UUID NOT NULL,
    voucher_number VARCHAR(40) NOT NULL,
    movement_date  TIMESTAMPTZ NOT NULL DEFAULT now(),
    amount         DECIMAL(15,4) NOT NULL CHECK (amount <> 0),  -- موجب للقبض، سالب للصرف
    balance_after  DECIMAL(15,4) NOT NULL,
    description    TEXT,
    created_by     UUID
);
CREATE INDEX IF NOT EXISTS idx_cashbox_movements_box_date
    ON cashbox_movements (cashbox_id, movement_date DESC);
```

### 3.6 (المرحلة 3 — مزدوج القيد) الحسابات واليومية

دليل حسابات + قيود يومية + بنود — تُنشأ تلقائياً من السندات (مثال: قبض → مدين صندوق/دائن إيراد).

```sql
CREATE TABLE IF NOT EXISTS accounts (
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code     VARCHAR(20) NOT NULL UNIQUE,
    name     VARCHAR(150) NOT NULL,
    name_en  VARCHAR(150),
    type     VARCHAR(20) NOT NULL CHECK (type IN ('ASSET','LIABILITY','EQUITY','INCOME','EXPENSE')),
    parent_id UUID REFERENCES accounts(id),
    is_active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS journal_entries (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_number VARCHAR(40) NOT NULL UNIQUE,
    voucher_type VARCHAR(20),
    voucher_id   UUID,
    entry_date   DATE NOT NULL DEFAULT CURRENT_DATE,
    description  TEXT,
    total_debit  DECIMAL(15,4) NOT NULL DEFAULT 0,
    total_credit DECIMAL(15,4) NOT NULL DEFAULT 0,
    status       VARCHAR(20) NOT NULL DEFAULT 'POSTED',
    created_by   UUID,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS journal_entry_lines (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    journal_entry_id UUID NOT NULL REFERENCES journal_entries(id) ON DELETE CASCADE,
    account_id       UUID NOT NULL REFERENCES accounts(id),
    debit            DECIMAL(15,4) NOT NULL DEFAULT 0,
    credit           DECIMAL(15,4) NOT NULL DEFAULT 0,
    notes            TEXT
);
```

---

## 4. منطق العمل (Workflow)

### 4.1 إنشاء سند قبض وترحيله

1. يُنشأ السند (حالة `DRAFT` أو مباشرة `POSTED` حسب الإعداد).
2. عند **الترحيل** (POST):
   - قفل صف الصندوق `SELECT ... FOR UPDATE` (أمان التزامن).
   - توليد الرقم التسلسلي ذرّياً من `voucher_counters`.
   - زيادة `cashboxes.current_balance`.
   - إدراج حركة في `cashbox_movements` (amount موجب).
   - تحديث حالة السند إلى `POSTED` + `posted_at`/`posted_by`.
   - (اختياري) إذا كان `payer_type = TENANT/RESELLER` فعّل "إضافة للمحفظة":
     - يُستدعى `balance.Manager.CreditBalance` (يكتب `balance_history` + `accounting_transactions`).
   - كل ذلك داخل **معاملة واحدة** (Transaction) — إما أن تنجح كلها أو تُلغى.

### 4.2 سند صرف

نفس الخطوات مع **التحقق من كفاية الرصيد** في الصندوق قبل الترحيل، وتنقيص الرصيد، وحركة amount سالب، و(اختياري) خصم من محفظة الشركة/الوكيل عبر `DeductBalance`.

### 4.3 العكس (الإلغاء)

- يُسمح فقط لسند في حالة `POSTED` لم يُعكس سابقاً.
- يُنشأ سند جديد `reversal_of = الأصل`، وتُطبَّق الحركة المعاكسة على الصندوق.
- يُسجَّل `reversed_at/reversed_by` على الأصل. **لا حذف فعلي** أبداً — للمحاسبة والمراجعة.

### 4.4 حالات السند

`DRAFT` (مسودة، لا أثر) → `POSTED` (مرحّل، يؤثر على الصندوق) → `REVERSED` (معكوس).

---

## 5. واجهات API (تحت مجموعة /admin — SUPER_ADMIN أو STAFF بصلاحية approve_payments)

| الطريقة | المسار | الوظيفة |
|---|---|---|
| GET | `/api/v1/admin/accounting/cashboxes` | قائمة الصناديق (مع الأرصدة) |
| POST | `/api/v1/admin/accounting/cashboxes` | إنشاء صندوق |
| GET | `/api/v1/admin/accounting/cashboxes/:id` | تفاصيل صندوق |
| PUT | `/api/v1/admin/accounting/cashboxes/:id` | تعديل صندوق |
| POST | `/api/v1/admin/accounting/cashboxes/:id/statement` | كشف حساب الصندوق (مع `from/to`) |
| GET | `/api/v1/admin/accounting/receipts` | قائمة سندات القبض (تصفية: تاريخ/صندوق/حالة) |
| POST | `/api/v1/admin/accounting/receipts` | إنشاء سند قبض |
| GET | `/api/v1/admin/accounting/receipts/:id` | تفاصيل سند قبض |
| POST | `/api/v1/admin/accounting/receipts/:id/post` | ترحيل |
| POST | `/api/v1/admin/accounting/receipts/:id/reverse` | عكس |
| GET | `/api/v1/admin/accounting/payments` | قائمة سندات الصرف |
| POST | `/api/v1/admin/accounting/payments` | إنشاء سند صرف |
| GET | `/api/v1/admin/accounting/payments/:id` | تفاصيل سند صرف |
| POST | `/api/v1/admin/accounting/payments/:id/post` | ترحيل |
| POST | `/api/v1/admin/accounting/payments/:id/reverse` | عكس |
| GET | `/api/v1/admin/accounting/numbering/:type` | معاينة الرقم التسلسلي التالي |

(المرحلة 3: `GET /accounting/journal`, `GET /accounting/ledger`, `GET /accounting/trial-balance`)

### هيكل الحزمة في الباك-إند

```
services/backend/internal/accounting/
├── models.go      -- نماذج GORM (أو مباشرة pgx)
├── cashbox.go     -- CRUD + statement
├── receipt.go     -- إنشاء/ترحيل/عكس سند قبض
├── payment.go     -- إنشاء/ترحيل/عكس سند صرف
├── numbering.go   -- مولّد الأرقام الذرّي
├── journal.go     -- (المرحلة 3) مزدوج القيد
└── router.go      -- تسجيل المسارات في main.go
```

---

## 6. واجهة المستخدم (تبويب العمليات المالية)

تُضاف **4 تبويبات فرعية** تحت "العمليات المالية" في `NAV_SECTIONS`:

| التبويب الفرعي | الملف المقترح | المحتوى |
|---|---|---|
| الإدارة المالية (قائمة) | `FinanceManager.jsx` | بطاقة ملخص: أرصدة الصناديق + آخر السندات |
| **الصناديق** | `pages/Accounting/Cashboxes.jsx` | قائمة الصناديق + إنشاء/تعديل + زر "كشف حساب" |
| **سندات القبض** | `pages/Accounting/ReceiptVouchers.jsx` | قائمة السندات + نموذج إنشاء + ترحيل + عكس |
| **سندات الصرف** | `pages/Accounting/PaymentVouchers.jsx` | قائمة السندات + نموذج إنشاء + ترحيل + عكس |

نموذج السند (قبض/صرف):
- التاريخ، الصندوق (قائمة منسدلة من `cashboxes` النشطة)
- المبلغ، العملة، طريقة الدفع (نقدي/تحويل/شيك/بطاقة/محفظة)
- اسم الطرف + نوعه (شركة/وكيل/عميل/مورد) + رقم مرجعي + بيان
- رقم السند يُعرض تلقائياً (معاينة قبل الحفظ)
- أزرار: حفظ كمسودة / ترحيل مباشرة / عكس (للسندات المرحّلة)

كشف حساب الصندوق: جدول (تاريخ، رقم السند، النوع، البيان، قبض/صرف، الرصيد) + فلترة من/إلى.

---

## 7. الصلاحيات والأمان

- **SUPER_ADMIN**: كل العمليات.
- **STAFF**:
  - `view_finances` → قراءة فقط (عرض القوائم والكشوف).
  - `approve_payments` → إنشاء وترحيل وعكس السندات وإدارة الصناديق.
- كل جدول فيه `created_by / posted_by / reversed_by` لمراجعة كل حركة.
- العمليات المالية كلها داخل **معاملات** (transactions) بذرّية كاملة.
- رصيد الصندوق يُقفل (FOR UPDATE) أثناء الترحيل لتفادي سباقات التحديث.

---

## 8. مراحل التنفيذ

### المرحلة 1 — الأساس (MVP)
- [x] Migration 093: `cashboxes` + `voucher_counters`
- [x] Migration 094: `receipt_vouchers` + `payment_vouchers` + `cashbox_movements`
- [x] حزمة `internal/accounting`: الصناديق + الترقيم + ترحيل/عكس السندات (بدون المحفظة)
- [x] تسجيل المسارات في main.go
- [x] الصفحات الثلاث في الواجهة + إضافة التبويبات الفرعية

### المرحلة 2 — الربط بالمحافظ والصلاحيات
- [ ] ربط سند قبض/صرف بمحافظ الشركات والوكلاء (CreditBalance / DeductBalance)
- [ ] تفعيل صلاحيات `approve_payments` / `view_finances` في الواجهة
- [ ] إصلاح مسارات مكسورة: `POST /admin/tenants/:id/balance`، تسجيل `payment_handler`
- [ ] تصدير الكشوف (Excel/PDF)

### المرحلة 3 — المحاسبة المزدوجة
- [ ] Migration: `accounts` + `journal_entries` + `journal_entry_lines`
- [ ] توليد قيود اليومية تلقائياً من السندات
- [ ] دفتر الأستاذ + ميزان المراجعة (Trial Balance)
- [ ] إعداد دليل حسابات افتراضي (نقدية/بنك/إيرادات/مصروفات)

---

## 9. الاختبار

- **اختبارات قاعدة البيانات**: تطبيق migrations مرتين (idempotent) + فحوصات CHECK.
- **اختبارات الوحدة (Go)**:
  - ترحيل سند قبض يزيد الرصيد ويسجّل الحركة ويرقم تسلسلياً.
  - ترحيل سند صرف فوق الرصيد المتاح **يفشل** (insufficient funds).
  - العكس يعيد الرصيد ولا يسمح بعكس سند معكوس.
  - التزامن: ترحيل سندين في آنٍ لا يكرر الرقم.
- **اختبارات API**: سيناريوهات كاملة لكل مسار (إنشاء/ترحيل/عكس/أذونات).
- **يدوي**: إنشاء صندوق ← سند قبض ← سند صرف ← كشف حساب ← عكس.

---

## 10. مخاطر وملاحظات

1. الواجهة الحالية `FinanceManager.jsx` وهمية بلا API — سيتم إعادة بنائها على البيانات الحقيقية.
2. `bank_accounts` موجود بدون تكامل — سيُدمج مع نوع الصناديق البنكية أو يُستبدل.
3. `Tenant.Balance` في GORM بدقة 2 خانات بينما DB بدقة 4 — يجب توحيدها لتجنب فقدان كسور.
4. العملة الافتراضية SAR مع إمكانية التوسع عبر جدول `currencies`.
5. أي مبلغ يُحرَّك يجب أن يُرصد في `cashbox_movements` قبل أي تحديث للرصيد (مبدأ التسلسل).
```
