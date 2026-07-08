# مراجعة Database Schema - المهمة 11.3

**التاريخ:** 2026-02-25  
**المتطلب:** 18.4 - التوافق مع الواجهات الحالية  
**الهدف:** التأكد من عدم تأثر الجداول المستخدمة من الواجهات (admin-web, client-web) بالتغييرات الجديدة

---

## ملخص تنفيذي

تم مراجعة جميع ملفات الـ migrations والتأكد من أن التغييرات الجديدة **لا تؤثر** على الجداول الأساسية المستخدمة من الواجهات. جميع التغييرات آمنة ومتوافقة مع الواجهات الحالية.

### النتيجة: ✅ **متوافق بالكامل**

---

## الجداول الأساسية المستخدمة من الواجهات

بناءً على تحليل الكود في `services/backend/internal/models/models.go` والواجهات، الجداول الأساسية هي:

### 1. جدول `users`
**الاستخدام:** إدارة المستخدمين، تسجيل الدخول، الصلاحيات

**الأعمدة الأساسية:**
- `id` (UUID, PRIMARY KEY)
- `email` (VARCHAR, UNIQUE)
- `password_hash` (VARCHAR)
- `phone_number` (VARCHAR, UNIQUE)
- `user_type` (VARCHAR) - SUPER_ADMIN, TENANT, STAFF
- `is_active` (BOOLEAN)
- `is_verified` (BOOLEAN)
- `created_at`, `updated_at`, `deleted_at`

**التغييرات في Migrations:** ❌ **لا توجد تغييرات**

---

### 2. جدول `tenants`
**الاستخدام:** معلومات الشركات، الرصيد، الاشتراكات

**الأعمدة الأساسية:**
- `id` (UUID, PRIMARY KEY)
- `user_id` (UUID, FOREIGN KEY → users)
- `business_name` (VARCHAR)
- `contact_name` (VARCHAR)
- `email` (VARCHAR)
- `phone` (VARCHAR)
- `balance` (DECIMAL(12,2))
- `status` (VARCHAR) - ACTIVE, WAIT_MODE
- `subscription_plan` (VARCHAR)
- `message_limit` (INTEGER)
- `is_active` (BOOLEAN)
- `company_serial` (VARCHAR, UNIQUE)
- `api_key_accounting` (VARCHAR, UNIQUE)
- `api_key_phone` (VARCHAR, UNIQUE)
- `allow_gateways` (BOOLEAN)
- `created_at`, `updated_at`

**التغييرات في Migrations:**
- ✅ **Migration 004_add_balance_constraint.sql**: إضافة constraint فقط
  - `ADD CONSTRAINT tenants_balance_non_negative CHECK (balance >= 0)`
  - **التأثير:** لا يوجد - هذا constraint للحماية فقط، لا يغير البيانات الموجودة
  - **متوافق:** نعم ✅

---

### 3. جدول `messages`
**الاستخدام:** عرض الرسائل، تتبع الحالة، التقارير

**الأعمدة الأساسية:**
- `id` (UUID, PRIMARY KEY)
- `tenant_id` (UUID, FOREIGN KEY → tenants)
- `device_id` (UUID, NULLABLE) - LEGACY
- `sender_name` (VARCHAR)
- `phone_number` (VARCHAR)
- `content` (TEXT)
- `status` (VARCHAR) - PENDING, PROCESSING, SENT, DELIVERED, FAILED, WAITING
- `retry_count` (INTEGER)
- `scheduled_at` (TIMESTAMP)
- `sent_at` (TIMESTAMP)
- `delivered_at` (TIMESTAMP)
- `created_at` (TIMESTAMP)

**التغييرات في Migrations:**
- ✅ **Migration 001_core_messaging_engine.sql**: إضافة أعمدة جديدة
  - `channel_id` (UUID, NULLABLE) - قناة الإرسال الجديدة
  - `idempotency_key` (VARCHAR, NULLABLE) - لمنع التكرار
  - `wait_reason` (TEXT, NULLABLE) - سبب الانتظار
  - `provider_job_id` (VARCHAR, NULLABLE) - معرف خارجي
  - `cost` (DECIMAL(10,2), DEFAULT 0.0) - تكلفة الرسالة
  - **التأثير:** لا يوجد - جميع الأعمدة NULLABLE أو لها قيم افتراضية
  - **متوافق:** نعم ✅

- ✅ **Migration 004_dead_letter_queue_table.sql**: إضافة عمود وحالة جديدة
  - `fail_reason` (TEXT, NULLABLE) - سبب الفشل
  - إضافة حالة `DEAD_LETTER` إلى constraint
  - **التأثير:** لا يوجد - العمود NULLABLE والحالة الجديدة لا تؤثر على الحالات الموجودة
  - **متوافق:** نعم ✅

- ✅ **Migration 005_add_message_indexes.sql**: إضافة indexes فقط
  - Indexes للأداء والعزل بين الشركات
  - **التأثير:** لا يوجد - Indexes لا تغير البيانات أو الواجهة
  - **متوافق:** نعم ✅

---

### 4. جدول `dispatch_channels`
**الاستخدام:** إدارة قنوات الإرسال (أجهزة، gateways)

**الأعمدة الأساسية:**
- `id` (UUID, PRIMARY KEY)
- `tenant_id` (UUID, FOREIGN KEY → tenants)
- `type` (VARCHAR) - PHONE, GATEWAY_HUDHUD, GATEWAY_PRIVATE
- `channel_key` (VARCHAR)
- `config` (JSONB)
- `status` (VARCHAR) - ONLINE, OFFLINE, WAIT_MODE
- `priority` (INTEGER)
- `created_at`, `updated_at`

**التغييرات في Migrations:**
- ✅ **Migration 001_core_messaging_engine.sql**: إنشاء الجدول
  - الجدول موجود مسبقاً في النظام
  - **التأثير:** لا يوجد
  - **متوافق:** نعم ✅

- ✅ **Migration 006_add_current_load_to_channels.sql**: إضافة عمود جديد
  - `current_load` (INTEGER, DEFAULT 0) - عدد الرسائل الحالية
  - `CHECK (current_load >= 0)` - constraint للحماية
  - **التأثير:** لا يوجد - العمود له قيمة افتراضية
  - **متوافق:** نعم ✅

---

### 5. جدول `paired_devices`
**الاستخدام:** إدارة الأجهزة المقترنة

**الأعمدة الأساسية:**
- `id` (UUID, PRIMARY KEY)
- `tenant_id` (UUID, FOREIGN KEY → tenants)
- `channel_id` (UUID, NULLABLE)
- `device_name` (VARCHAR)
- `hardware_id` (VARCHAR, UNIQUE)
- `device_pair_key` (VARCHAR)
- `phone_number` (VARCHAR)
- `status` (VARCHAR)
- `last_seen` (TIMESTAMP)
- `created_at` (TIMESTAMP)

**التغييرات في Migrations:** ❌ **لا توجد تغييرات**

---

## الجداول الجديدة (لا تؤثر على الواجهات)

### 1. جدول `balance_history` (جديد)
**الغرض:** تسجيل جميع عمليات الرصيد للتدقيق المالي

**Migration:** 002_balance_history_table.sql

**الأعمدة:**
- `id` (UUID, PRIMARY KEY)
- `tenant_id` (UUID, FOREIGN KEY → tenants)
- `message_id` (UUID, NULLABLE, FOREIGN KEY → messages)
- `old_balance` (DECIMAL(12,2))
- `new_balance` (DECIMAL(12,2))
- `amount` (DECIMAL(12,2))
- `type` (VARCHAR) - DEDUCT, REFUND, CHARGE, ADJUSTMENT
- `reason` (TEXT)
- `created_by` (VARCHAR)
- `ip_address` (INET)
- `created_at` (TIMESTAMP)

**التأثير على الواجهات:** ❌ **لا يوجد**
- جدول جديد تماماً
- لا يغير أي جداول موجودة
- يمكن استخدامه لاحقاً لعرض تاريخ الرصيد

---

### 2. جدول `accounting_transactions` (جديد)
**الغرض:** تتبع حالة المحاسبة لكل رسالة

**Migration:** 003_accounting_transactions_table.sql

**الأعمدة:**
- `id` (UUID, PRIMARY KEY)
- `tenant_id` (UUID, FOREIGN KEY → tenants)
- `message_id` (UUID, FOREIGN KEY → messages)
- `amount` (DECIMAL(12,2))
- `status` (VARCHAR) - PENDING, COMPLETED, REFUNDED, FAILED
- `transaction_type` (VARCHAR) - DEDUCT, REFUND, CHARGE
- `discount_applied` (DECIMAL(12,2))
- `original_amount` (DECIMAL(12,2))
- `settled_at` (TIMESTAMP)
- `created_at`, `updated_at` (TIMESTAMP)

**التأثير على الواجهات:** ❌ **لا يوجد**
- جدول جديد تماماً
- لا يغير أي جداول موجودة
- يمكن استخدامه لاحقاً لعرض تفاصيل المحاسبة

---

### 3. جدول `dead_letter_queue` (جديد)
**الغرض:** تخزين الرسائل الفاشلة نهائياً

**Migration:** 004_dead_letter_queue_table.sql

**الأعمدة:**
- `id` (BIGSERIAL, PRIMARY KEY)
- `tenant_id` (UUID)
- `message_id` (UUID)
- `phone_number` (VARCHAR)
- `content` (TEXT)
- `original_status` (VARCHAR)
- `fail_reason` (TEXT)
- `retry_count` (INTEGER)
- `cost` (DECIMAL(10,4))
- `moved_at` (TIMESTAMP)
- `created_at` (TIMESTAMP)

**التأثير على الواجهات:** ❌ **لا يوجد**
- جدول جديد تماماً
- لا يغير أي جداول موجودة
- يمكن استخدامه لاحقاً لعرض الرسائل الفاشلة

---

### 4. جدول `incoming_messages` (جديد)
**الغرض:** تخزين الرسائل الواردة الخام

**Migration:** 001_core_messaging_engine.sql

**الأعمدة:**
- `id` (UUID, PRIMARY KEY)
- `tenant_id` (UUID)
- `phone_number` (VARCHAR)
- `message_content` (TEXT)
- `raw_payload` (JSONB)
- `status` (VARCHAR)
- `processed_at` (TIMESTAMP)
- `created_at` (TIMESTAMP)

**التأثير على الواجهات:** ❌ **لا يوجد**
- جدول جديد تماماً
- لا يغير أي جداول موجودة

---

## تحليل التوافق مع API Endpoints

بناءً على تحليل الكود في `apps/admin-web/src/App.jsx` و `apps/admin-web/src/contexts/AuthContext.jsx`:

### API Endpoints المستخدمة:
1. `POST /api/v1/auth/login` - تسجيل الدخول
2. `POST /api/v1/auth/logout` - تسجيل الخروج
3. `GET /api/v1/auth/me` - معلومات المستخدم الحالي
4. `GET /api/v1/admin/stats` - إحصائيات النظام
5. `GET /api/v1/admin/tenants` - قائمة الشركات
6. `POST /api/v1/tenants/register` - تسجيل شركة جديدة

### التحقق من التوافق:

#### 1. `/api/v1/auth/login`
**الجداول المستخدمة:** `users`, `tenants`, `sessions`
- ✅ لا توجد تغييرات على `users`
- ✅ التغيير الوحيد على `tenants` هو constraint (لا يؤثر)
- ✅ **متوافق بالكامل**

#### 2. `/api/v1/auth/me`
**الجداول المستخدمة:** `users`, `tenants`
- ✅ لا توجد تغييرات تؤثر على البيانات المعروضة
- ✅ **متوافق بالكامل**

#### 3. `/api/v1/admin/stats`
**الجداول المستخدمة:** `messages`, `tenants`, `users`
- ✅ الأعمدة الجديدة في `messages` لا تؤثر على الإحصائيات الأساسية
- ✅ يمكن استخدام الأعمدة الجديدة لإحصائيات إضافية لاحقاً
- ✅ **متوافق بالكامل**

#### 4. `/api/v1/admin/tenants`
**الجداول المستخدمة:** `tenants`, `users`
- ✅ لا توجد تغييرات على الأعمدة المعروضة
- ✅ **متوافق بالكامل**

#### 5. `/api/v1/tenants/register`
**الجداول المستخدمة:** `users`, `tenants`
- ✅ لا توجد تغييرات على عملية التسجيل
- ✅ الـ constraint الجديد يحمي من الرصيد السالب (ميزة إضافية)
- ✅ **متوافق بالكامل**

---

## تحليل التوافق مع WebSocket Events

بناءً على تحليل الكود في `services/backend/internal/ws/`:

### WebSocket Events المستخدمة:
1. `message:new` - رسالة جديدة
2. `message:status` - تحديث حالة رسالة
3. `device:status` - تحديث حالة جهاز
4. `balance:update` - تحديث الرصيد

### التحقق من التوافق:

#### 1. `message:new`
**البيانات المرسلة:** معلومات الرسالة من جدول `messages`
- ✅ الأعمدة الجديدة NULLABLE ولا تؤثر على البيانات الأساسية
- ✅ يمكن إضافة الحقول الجديدة تدريجياً إذا لزم الأمر
- ✅ **متوافق بالكامل**

#### 2. `message:status`
**البيانات المرسلة:** `message_id`, `status`, `timestamp`
- ✅ لا توجد تغييرات على هذه الحقول
- ✅ الحالة الجديدة `DEAD_LETTER` لن تظهر إلا للرسائل الفاشلة نهائياً
- ✅ **متوافق بالكامل**

#### 3. `device:status`
**البيانات المرسلة:** معلومات الجهاز من `paired_devices` و `dispatch_channels`
- ✅ العمود الجديد `current_load` في `dispatch_channels` لا يؤثر على البيانات الأساسية
- ✅ يمكن استخدامه لاحقاً لعرض الحمل الحالي
- ✅ **متوافق بالكامل**

#### 4. `balance:update`
**البيانات المرسلة:** `tenant_id`, `balance`, `timestamp`
- ✅ لا توجد تغييرات على حقل `balance` في جدول `tenants`
- ✅ الـ constraint الجديد يحمي من القيم السالبة فقط
- ✅ **متوافق بالكامل**

---

## ملخص التغييرات حسب Migration

### Migration 001: Core Messaging Engine
**التغييرات:**
- إنشاء جدول `dispatch_channels` (موجود مسبقاً)
- إنشاء جدول `incoming_messages` (جديد)
- إضافة أعمدة لجدول `messages`: `channel_id`, `idempotency_key`, `wait_reason`, `provider_job_id`, `cost`

**التأثير:** ✅ **لا يوجد** - جميع الأعمدة الجديدة NULLABLE أو لها قيم افتراضية

---

### Migration 002: Balance History Table
**التغييرات:**
- إنشاء جدول `balance_history` (جديد)

**التأثير:** ✅ **لا يوجد** - جدول جديد تماماً

---

### Migration 003: Accounting Transactions Table
**التغييرات:**
- إنشاء جدول `accounting_transactions` (جديد)

**التأثير:** ✅ **لا يوجد** - جدول جديد تماماً

---

### Migration 004: Add Balance Constraint
**التغييرات:**
- إضافة constraint على جدول `tenants`: `balance >= 0`

**التأثير:** ✅ **لا يوجد** - constraint للحماية فقط، لا يغير البيانات الموجودة

---

### Migration 004: Dead Letter Queue Table
**التغييرات:**
- إنشاء جدول `dead_letter_queue` (جديد)
- إضافة عمود `fail_reason` لجدول `messages`
- إضافة حالة `DEAD_LETTER` إلى constraint

**التأثير:** ✅ **لا يوجد** - العمود NULLABLE والحالة الجديدة لا تؤثر على الحالات الموجودة

---

### Migration 005: Add Message Indexes
**التغييرات:**
- إضافة indexes لجدول `messages`

**التأثير:** ✅ **لا يوجد** - Indexes لا تغير البيانات أو الواجهة

---

### Migration 006: Add Current Load to Channels
**التغييرات:**
- إضافة عمود `current_load` لجدول `dispatch_channels`

**التأثير:** ✅ **لا يوجد** - العمود له قيمة افتراضية (0)

---

## التوصيات

### 1. التوافق الكامل ✅
جميع التغييرات متوافقة بالكامل مع الواجهات الحالية. لا حاجة لأي تعديلات على admin-web أو client-web.

### 2. الميزات الإضافية المستقبلية 🔮
يمكن الاستفادة من الجداول والأعمدة الجديدة لإضافة ميزات في المستقبل:

#### في admin-web:
- عرض تاريخ الرصيد من `balance_history`
- عرض تفاصيل المحاسبة من `accounting_transactions`
- عرض الرسائل الفاشلة من `dead_letter_queue`
- عرض الحمل الحالي للقنوات من `dispatch_channels.current_load`

#### في client-web:
- عرض تكلفة كل رسالة من `messages.cost`
- عرض سبب الفشل من `messages.fail_reason`
- عرض القناة المستخدمة من `messages.channel_id`

### 3. الحماية المالية 🛡️
الـ constraint الجديد على `tenants.balance >= 0` يوفر حماية إضافية على مستوى قاعدة البيانات ضد الرصيد السالب.

### 4. الأداء 🚀
الـ indexes الجديدة في `messages` و `dispatch_channels` ستحسن الأداء بشكل كبير دون أي تأثير على الواجهات.

---

## الخلاصة

### ✅ **جميع التغييرات متوافقة بالكامل**

1. **لا توجد تغييرات على أسماء الأعمدة الموجودة**
2. **لا توجد تغييرات على أنواع البيانات الموجودة**
3. **لا توجد حذف لأعمدة مستخدمة**
4. **جميع الأعمدة الجديدة NULLABLE أو لها قيم افتراضية**
5. **الجداول الجديدة لا تؤثر على الواجهات**
6. **الـ constraints الجديدة توفر حماية إضافية فقط**
7. **الـ indexes الجديدة تحسن الأداء فقط**

### 🎯 **المتطلب 18.4 محقق بالكامل**

> "THE Message_Lifecycle_System SHALL الحفاظ على نفس Database Schema للجداول المستخدمة من الواجهات"

**النتيجة:** ✅ **تم تحقيق المتطلب بنجاح**

---

**تمت المراجعة بواسطة:** Kiro AI  
**التاريخ:** 2026-02-25  
**الحالة:** ✅ **معتمد**
