# خطة هيكلة نظام إرسال الرسائل والاشتراكات

## ملخص الفكرة (التي تم الاتفاق عليها)
الهدف هو بناء نظام إرسال سريع، معزول، ولا يرهق الخادم بالحسابات المعقدة لأجزاء الرسائل وتسعيراتها، وذلك من خلال فصل عملية الإرسال إلى مرحلتين، والاعتماد على نظام "العمليات" (Operations) بدلاً من الحساب الدقيق للأجزاء.

---

## المرحلة الأولى: الاستلام الفوري (API Layer)
تتم هذه المرحلة عندما تقوم الشركة بطلب إرسال رسالة من خلال الـ API أو لوحة التحكم.
1. **استقبال الطلب:** يقوم الخادم باستلام الطلب والتحقق السريع من صلاحية مفتاح الشركة (`Company API Key`).
2. **التحقق من الرصيد:** التأكد من أن اشتراك الشركة نشط ويحتوي على رصيد عمليات متاح (`Operations Quota > 0`).
3. **حفظ مبدئي:** حفظ الرسالة في قاعدة البيانات بحالة `Queued` (في الانتظار).
4. **توجيه للطابور:** تحويل المهمة فوراً إلى طابور المعالجة في الخلفية (River Worker Pool).
5. **استجابة سريعة:** إرجاع رد فوري للشركة (HTTP 202 Accepted) يحتوي على `MessageID` لضمان عدم تأخير العميل.

---

## المرحلة الثانية: الإرسال والخصم (Worker Layer)
تتم هذه المرحلة في الخلفية (Background) عن طريق العمال (Workers) لضمان عدم تأثر أداء النظام المباشر.
1. **التقاط المهمة:** يقوم الـ Worker بسحب الرسالة من الطابور.
2. **الخصم الفوري:** يتم خصم **"عملية واحدة (1 Operation)"** من خطة اشتراك العميل، وذلك بغض النظر عن نجاح أو فشل الإرسال النهائي.
3. **الإرسال للمزود:** توجيه الرسالة إلى البوابة المناسبة (بوابة الهاتف Mobile Gateway أو مزود SMS خارجي).
4. **تحديث الحالة:** 
   - إذا تم التسليم للمزود بنجاح: تحديث حالة الرسالة في قاعدة البيانات إلى `Sent`.
   - إذا فشل التسليم: تحديث الحالة إلى `Failed` (الرصيد يظل مخصوماً لأن المحاولة تمت).

---

## سياسة الأرصدة والتعرفات

> تعتمد جميع الخطط على مبدأ واحد: **يتم تحصيل عملية واحدة لكل طلب إرسال، بغض النظر عن نجاحه أو فشله أو عدد أجزاء الرسالة.**  
> في الخطط التي يكون الرصيد فيها عند المزود الخارجي، نقوم فقط بعرض الرصيد ولا نتدخل في حساباته.

---

## 🗂️ تصنيف خطط الاشتراك الأربع

### 📱 النوع الأول: خطط الهواتف (Mobile Gateway Plans)

| الوصف | تعتمد على إرسال الرسائل عبر هواتف أندرويد مقترنة بالنظام (تطبيق Mobile Gateway) |
|---|---|
| **نوع الفوترة** | `ops_based` — خصم عملية واحدة لكل رسالة مُرسَلة |
| **الرصيد** | يُدار داخلياً في جدول `subscriptions.messages_remaining` |
| **العزل** | كل شركة لها Worker Pool خاص + طابور مستقل |
| **الرفض** | إذا نفذت العمليات → HTTP 429 Quota Exceeded |
| **أمثلة على الخطط** | خطة 500 عملية / خطة 2000 عملية / خطة غير محدودة |
| **المعرّف في الكود** | `plan_category = "mobile_gateway"` |

---

### 📡 النوع الثاني: خطط بوابة الهدهد (Hudhud Gateway Plans)

| الوصف | الإرسال عبر بوابة الاتصالات الخاصة بالهدهد (سواء كانت تعتمد على الهاتف أو اتصال بوابة SMS مباشر) |
|---|---|
| **نوع الفوترة** | `ops_based` — خصم عملية واحدة لكل رسالة |
| **الرصيد** | يُدار داخلياً في جدول `subscriptions.messages_remaining` |
| **العزل** | طابور مستقل خاص ببوابة الهدهد، لا يتأثر بخطط الهواتف |
| **الرفض** | إذا نفذت العمليات → HTTP 429 Quota Exceeded |
| **أمثلة على الخطط** | خطة 1000 عملية شهرياً / خطة 10,000 عملية |
| **المعرّف في الكود** | `plan_category = "hudhud_gateway"` |

---

### 🔌 النوع الثالث: خطط ربط البوابة الخاصة (Private Gateway Plans)

| الوصف | الشركة تمتلك بوابة SMS خاصة بها وتريد توصيلها بمنصة الهدهد للإدارة والإرسال عبرها |
|---|---|
| **نوع الفوترة** | `balance_based` — الرصيد يُدار عند مزود الشركة الخارجي |
| **الرصيد** | **لا نتدخل** — نعرض فقط الرصيد المسحوب من API المزود الخارجي |
| **العزل** | الشركة تضيف بيانات اعتماد (Credentials) بوابتها في إعداداتها |
| **الرفض** | لا رفض من جانبنا — الرفض يأتي من المزود الخارجي نفسه |
| **أمثلة على الخطط** | خطة توصيل بوابة أساسية / بوابة متقدمة مع إحصائيات |
| **المعرّف في الكود** | `plan_category = "private_gateway"` |

---

### 🖥️ النوع الرابع: خطط استئجار السيرفر (Dedicated Server Plans)

| الوصف | الشركة تستأجر سيرفراً مستقلاً من الهدهد، عليه نسخة مستقلة من النظام |
|---|---|
| **نوع الفوترة** | اشتراك شهري/سنوي ثابت (لا يتعلق بعدد الرسائل) |
| **الرصيد** | لا يوجد رصيد عمليات — الاشتراك يُفعّل الخدمة كاملة |
| **العزل** | **عزل كامل** — سيرفر منفصل، قاعدة بيانات منفصلة، نطاق فرعي مستقل |
| **الرفض** | لا رفض طالما الاشتراك نشط |
| **أمثلة على الخطط** | سيرفر مشترك / سيرفر مخصص / سيرفر أعمال |
| **المعرّف في الكود** | `plan_category = "dedicated_server"` |

---

## تحليل الكود الحالي (الواقع الموجود)

بعد فحص المشروع، هذه هي الأنظمة الموجودة فعلياً:

### 1. نموذج الخطط (`Package` في `subscription/types.go`)
الحقول الموجودة:
- `IncludedMessages` — عدد الرسائل المضمنة (NULL = غير محدود)
- `OverageRate` — سعر الرسالة الزائدة (Overage)
- `IsUnlimited` — علامة للخطة غير المحدودة
- `BillingCycle` — monthly / yearly
- `RolloverEnabled` / `RolloverLimit` — ترحيل الرصيد
- `PriceUSD` — سعر الخطة

### 2. نموذج الاشتراك (`Subscription` في `subscription/types.go`)
- `MessagesUsed` / `MessagesRemaining` — تتبع الاستهلاك
- `OverageMessages` / `OverageCost` — تتبع التجاوز
- `Status` — active / grace_period / suspended / cancelled / expired

### 3. نموذج الشركة (`Tenant` في `models/models.go`)
- `SubscriptionPlan` — نص بسيط فقط (مشكلة: لا يرتبط بجدول الخطط)
- `MessageLimit` — رقم بسيط (مشكلة: مكرر مع جدول subscriptions)
- `Balance` — الرصيد الحالي
- `APIKeyAccounting` / `APIKeyPhone` / `APIKeyWeb` — مفاتيح متعددة

### 4. أنواع الخدمات (`ServiceType` في `models/tariff_service.go`)
```
GATEWAY  → pay-as-you-go (مزود SMS خارجي)
MOBILE   → subscription (Mobile Gateway)
DEDICATED → enterprise
```

### 5. حالات الرسالة (`MessageStatus` في `models/models.go`)
```
PENDING → QUEUED → WAITING → PROCESSING → SENT → DELIVERED / FAILED
```

---

## الفجوات المكتشفة (ما ينقص لتطبيق الخطة الجديدة)

| # | المشكلة | الملف | الحل المطلوب |
|---|---------|-------|-------------|
| 1 | `Tenant.SubscriptionPlan` نص حر غير مرتبط بجدول الخطط | `models.go` | إضافة حقل `ActiveSubscriptionID *int64` أو الاعتماد على جدول `subscriptions` |
| 2 | `Tenant.MessageLimit` مكرر ومعزول | `models.go` | إهمال الحقل والاعتماد على `subscriptions.messages_remaining` |
| 3 | `Package` لا يحتوي على نوع الخطة بوضوح (`ops_based` أو `balance_based`) | `subscription/types.go` | إضافة حقل `PlanType` |
| 4 | `DeductMessages` في `manager.go` تخصم بناءً على عدد الرسائل، لكن القرار الجديد هو خصم **عملية واحدة دائماً** | `subscription/manager.go` | تعديل منطق الخصم |
| 5 | لا يوجد `Endpoint` لجلب رصيد المزود الخارجي | — | إنشاء `GET /api/company/balance/provider` |
| 6 | لا يوجد تمييز واضح في `DeductMessages` بين خطة العمليات وخطة الرصيد الخارجي | `subscription/manager.go` | إضافة فرع منطقي بناءً على `PlanType` |

---

## التعديلات المطلوبة (مرتبة بالأولوية)

### 🔴 الأولوية 1 — تعريف أنواع الخطط بوضوح
**الملف:** `services/backend/internal/subscription/types.go`

إضافة نوع جديد:
```go
// PlanType defines how operations are billed
type PlanType string

const (
    // PlanTypeOps: الخصم بعدد العمليات (1 عملية = 1 رسالة بغض النظر عن الأجزاء)
    PlanTypeOps PlanType = "ops_based"
    // PlanTypeBalance: الرصيد يُدار من المزود الخارجي، نحن نعرضه فقط
    PlanTypeBalance PlanType = "balance_based"
)
```

إضافة الحقل إلى `Package`:
```go
PlanType PlanType `json:"plan_type"`
```

---

### 🔴 الأولوية 2 — تعديل منطق الخصم
**الملف:** `services/backend/internal/subscription/manager.go`

تعديل `DeductMessages` بحيث:
- إذا كانت الخطة `ops_based` → خصم **1 عملية** دائماً (بغض النظر عن `MessageCount`)
- إذا كانت الخطة `balance_based` → **لا خصم** من جانبنا، فقط تسجيل العملية

---

### 🟡 الأولوية 3 — إضافة نقطة جلب رصيد المزود الخارجي
**ملف جديد:** `services/backend/internal/api/provider_balance_handler.go`

```go
// GET /api/company/provider-balance
// يقرأ مفتاح API الخاص بالمزود من config الشركة
// ويعيد الرصيد المباشر من واجهة المزود
```

---

### 🟡 الأولوية 4 — تنظيف نموذج `Tenant`
**الملف:** `services/backend/internal/models/models.go`

- إهمال `SubscriptionPlan string` واستبدالها بـ `ActiveSubscriptionID *int64`
- توثيق أن `MessageLimit` مهمل ولا يستخدم في المنطق الجديد

---

### 🟢 الأولوية 5 — تعديل Migration للـ Database
**ملف جديد:** `services/backend/migrations/XXXX_add_plan_type_to_packages.sql`

```sql
ALTER TABLE packages ADD COLUMN plan_type VARCHAR(20) NOT NULL DEFAULT 'ops_based';
COMMENT ON COLUMN packages.plan_type IS 'ops_based: charge per operation | balance_based: external provider manages balance';
```

---

## خريطة تدفق الإرسال الجديدة

```
[الشركة] → POST /api/send
    ↓
[التحقق من API Key] → جلب TenantID + ActiveSubscription
    ↓
[فحص نوع الخطة]
    ├─ ops_based:
    │   ├─ فحص: MessagesRemaining > 0 ؟
    │   │   ├─ نعم → حفظ بحالة QUEUED + إرسال للطابور + خصم 1 عملية
    │   │   └─ لا  → رفض الطلب (HTTP 429 Quota Exceeded)
    │   └─ Worker: إرسال للمزود + تحديث الحالة (SENT/FAILED)
    │
    └─ balance_based:
        ├─ حفظ بحالة QUEUED + إرسال للطابور (بدون خصم محلي)
        └─ Worker: إرسال للمزود + تحديث الحالة
            └─ الرصيد يُعرض فقط من GET /provider-balance
```

---
*هذا الملف قابل للتحديث وسنقوم بإضافة التعديلات والقرارات الجديدة إليه تباعاً خلال عملنا.*
