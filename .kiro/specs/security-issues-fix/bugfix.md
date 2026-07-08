# وثيقة متطلبات إصلاح المشاكل الأمنية والثغرات

## Introduction

بناءً على تقرير الفحص الأمني الشامل لمشروع ALHudhudAI، تم تحديد 15 مشكلة أمنية وثغرة تقنية تتطلب إصلاحاً فورياً. هذه الوثيقة تحدد السلوك الحالي المعيب والسلوك المتوقع الصحيح والسلوك الذي يجب الحفاظ عليه لمنع أي انحدار في الوظائف الموجودة.

**نطاق الإصلاح:** إصلاح جميع المشاكل الأمنية والثغرات التقنية المكتشفة في قاعدة البيانات والكود.

**الأولوية:** المشاكل الحرجة أولاً (المستوى الأحمر)، ثم المشاكل العالية (المستوى البرتقالي)، ثم المشاكل المتوسطة (المستوى الأصفر).

**الملاحظات:**
- استخدام migrations جديدة للإصلاحات (مثلاً 050، 051، إلخ)
- عدم تعديل ملفات migration الموجودة
- استخدام ALTER TABLE بدلاً من DROP/CREATE
- الحفاظ على التوافق مع الكود الموجود

---

## Bug Analysis

### Current Behavior (Defect)

### 1. السلوك الحالي (العيوب)

#### 1.1 بيانات اعتماد مكشوفة في ملف .env

**الموقع:**
- `apps/admin-web/.env`
- `apps/client-web/.env`
- `apps/mobile_gateway/.env.example`
- `apps/backend/.env`

**المشكلة:** تحتوي الملفات على بيانات اعتماد حساسة بدون أي تشفير أو حماية.

1.1 WHEN يتم فحص ملف .env THEN يتم العثور على DB_PASSWORD بشكل نص واضح
1.2 WHEN يتم فحص ملف .env THEN يتم العثور على REDIS_PASSWORD بشكل نص واضح
1.3 WHEN يتم فحص ملف .env THEN يتم العثور على JWT_SECRET بشكل نص واضح
1.4 WHEN يتم فحص ملف .env THEN يتم العثور على ADMIN_PASSWORD بشكل نص واضح

#### 1.2 نظام الصلاحيات (RBAC) غير مكتمل

**المشكلة:** جداول role_permissions و user_roles مفقودة من قاعدة البيانات.

1.5 WHEN يتم فحص هيكل قاعدة البيانات THEN لا يوجد جدول role_permissions
1.6 WHEN يتم فحص هيكل قاعدة البيانات THEN لا يوجد جدول user_roles
1.7 WHEN يتم محاولة تطبيق نظام صلاحيات دقيق THEN النظام لا يدعمه بسبب غياب الجداول اللازمة

#### 1.3 أعمدة حرجة بدون قيود NOT NULL

**الجداول المتأثرة:**
- messages
- dispatch_channels
- accounting_transactions

1.8 WHEN يتم فحص جدول messages THEN أعمدة مهمة مثل channel_id و device_id تسمح بقيم NULL بدون قيود
1.9 WHEN يتم فحص جدول dispatch_channels THEN أعمدة أساسية تسمح بقيم NULL
1.10 WHEN يتم فحص جدول accounting_transactions THEN أعمدة مثل type و status تسمح بقيم NULL

#### 1.4 عدم توحيد الدقة في الأعمدة المالية

**المشكلة:** بعض الأعمدة المالية تستخدم DECIMAL(12,2) وأخرى تستخدم DECIMAL(12,4).

1.11 WHEN يتم فحص الأعمدة المالية THEN يتم العثور على DECIMAL(12,2) في بعض الجداول
1.12 WHEN يتم فحص الأعمدة المالية THEN يتم العثور على DECIMAL(12,4) في جداول أخرى
1.13 WHEN يتم إجراء عمليات حسابية مالية THEN النتائج قد تختلف بسبب اختلاف الدقة

#### 1.5 تشفير SSL/TLS معطل للاتصال بقاعدة البيانات

**المشكلة:** DB_SSLMODE=disable في ملفات البيئة.

1.14 WHEN يتم فحص ملف .env THEN يتم العثور على DB_SSLMODE=disable
1.15 WHEN يتم الاتصال بقاعدة البيانات THEN الاتصال يتم بدون تشفير TLS/SSL
1.16 WHEN يتم اعتراض حركة الشبكة THEN البيانات المنقولة تكون واضحة ومكشوفة

#### 1.6 Row Level Security (RLS) غير مفعل على 6 جداول

**الجداول المتأثرة:**
- messages
- balance_history
- devices
- channels
- users
- audit_trail

1.17 WHEN يتم فحص جدول messages THEN لا يوجد Row Level Security مفعل
1.18 WHEN يتم فحص جدول balance_history THEN لا يوجد Row Level Security مفعل
1.19 WHEN يتم فحص جدول devices THEN لا يوجد Row Level Security مفعل
1.20 WHEN يتم فحص جدول channels THEN لا يوجد Row Level Security مفعل
1.21 WHEN يتم فحص جدول users THEN لا يوجد Row Level Security مفعل
1.22 WHEN يتم فحص جدول audit_trail THEN لا يوجد Row Level Security مفعل

#### 1.7 فهارس مفقودة على أعمدة tenant_id

**الجداول المتأثرة:**
- messages
- balance_history
- accounting_transactions
- subscriptions

1.23 WHEN يتم فحص جدول messages THEN لا يوجد فهرس على tenant_id
1.24 WHEN يتم فحص جدول balance_history THEN لا يوجد فهرس مركب على tenant_id و type و created_at
1.25 WHEN يتم فحص جدول accounting_transactions THEN لا يوجد فهرس على tenant_id و status
1.26 WHEN يتم فحص جدول subscriptions THEN لا يوجد فهرس على tenant_id و expires_at

#### 1.8 قيود CHECK مفقودة للأعمدة المالية

**الأعمدة المتأثرة:**
- messages.cost
- balance_history.amount
- accounting_transactions.amount

1.27 WHEN يتم فحص جدول messages THEN لا يوجد CHECK constraint لضمان cost >= 0
1.28 WHEN يتم فحص جدول balance_history THEN لا يوجد CHECK constraint لضمان amount != 0
1.29 WHEN يتم فحص جدول accounting_transactions THEN لا يوجد CHECK constraint لضمان amount > 0

#### 1.9 Circuit Breaker بدون Logger

**الموقع:** `apps/backend/internal/resilience/`

1.30 WHEN يحدث خطأ في Circuit Breaker THEN لا يتم تسجيل أي logs
1.31 WHEN يتم فتح الدائرة (open state) THEN لا يوجد تنبيه أو إشعار
1.32 WHEN يتم تجاوز عتبة الفشل THEN لا يوجد سجل للأحداث

#### 1.10 كود مكرر بين admin-web و client-web

**المكونات المتأثرة:**
- API axios configuration
- Authentication hooks
- Form validation utilities

1.33 WHEN يتم فحص ملف axios.js في admin-web THEN يتم العثور على كود مكرر من client-web
1.34 WHEN يتم فحص hooks المصادقة THEN نفس المنطق موجود في كلا المشروعين
1.35 WHEN يتم تعديل كود مشترك THEN يجب تعديله في مكانين مختلفين

#### 1.11 Rate Limiting قابل للتحايل

**الموقع:** `apps/backend/internal/middleware/`

1.36 WHEN يتم فحص Rate Limiting middleware THEN يمكن تجاوزه عبر تغيير IP
1.37 WHEN يتم إرسال طلبات سريعة من عناوين IP مختلفة THEN لا يتم حجبها
1.38 WHEN يتم استخدام Proxy أو VPN THEN يتم تجاوز Rate Limiting

#### 1.12 إدارة CSRF Token بسيطة

**الموقع:**
- `apps/admin-web/src/api/axios.js`
- `apps/client-web/src/api/axios.js`

1.39 WHEN يتم فحص إدارة CSRF Token THEN لا يوجد تجديد تلقائي للـ token
1.40 WHEN تنتهي صلاحية CSRF Token THEN لا يوجد آلية للتعامل مع ذلك
1.41 WHEN يتم إرسال طلب بدون CSRF Token صحيح THEN لا يوجد معال��ة مناسبة للخطأ

#### 1.13 لا توجد Triggers لتحديث updated_at

**الجداول المتأثرة:**
- tenants
- users
- channels
- devices
- messages

1.42 WHEN يتم تعديل صف في أي جدول THEN لا يوجد Trigger لتحديث updated_at تلقائياً
1.43 WHEN يتم تعديل بيانات مستأجر THEN الحقل updated_at يبقى بدون تغيير
1.44 WHEN يتم تعديل رسالة THEN الحقل updated_at يبقى بدون تغيير

#### 1.15 عدم اتساق في UUID Generation

**المشكلة:** بعض الجداول تستخدم SERIAL (INTEGER) وأخرى تستخدم UUID.

1.45 WHEN يتم فحص جدول tenants THEN يستخدم SERIAL للمعرف
1.46 WHEN يتم فحص جدول messages THEN يستخدم VARCHAR(255) لـ message_id
1.47 WHEN يتم فحص جداول أخرى THEN تستخدم UUID للمعرفات
1.48 WHEN يتم الربط بين الجداول THEN يحدث تعارض في أنواع البيانات

#### 1.16 توثيق ومعلومات غير واضحة

**المشكلة:** التوثيق الموجود غير كافٍ وغير واضح.

1.49 WHEN يتم قراءة التوثيق THEN لا يوجد شرح واضح للإصلاحات المطلوبة
1.50 WHEN يتم محاولة فهم هيكل المشروع THEN التوثيق ناقص ومبهم
1.51 WHEN يتم البحث عن معلومات أمنية THEN لا يوجد قسم مخصص للأمان

---

### Expected Behavior (Correct)

#### 2.1 بيانات اعتماد محمية ومشفرة

2.1 WHEN يتم فحص ملف .env THEN يجب أن يحتوي على Placeholders فقط
2.2 WHEN يتم فحص ملف .env THEN يجب أن تكون البيانات الحساسة مشفرة أو في متغيرات بيئة خارجية
2.3 WHEN يتم رفع الكود لمستودع Git THEN لا يجب أن يحتوي على بيانات اعتماد حقيقية
2.4 WHEN يتم إنشاء ملف .env.example THEN يجب أن يحتوي على نموذج بدون قيم حقيقية

#### 2.2 نظام صلاحيات (RBAC) كامل

2.5 WHEN يتم فحص قاعدة البيانات THEN يجب أن يوجد جدول role_permissions
2.6 WHEN يتم فحص قاعدة البيانات THEN يجب أن يوجد جدول user_roles
2.7 WHEN يتم تطبيق نظام صلاحيات THEN يجب أن يدعم الأدوار والصلاحيات الدقيقة
2.8 WHEN يتم ربط المستخدمين بالأدوار THEN يجب أن يكون الربط عبر جدول user_roles

#### 2.3 قيود NOT NULL على الأعمدة الحرجة

2.9 WHEN يتم فحص جدول messages THEN يجب أن تكون الأعمدة channel_id و device_id بـ NOT NULL
2.10 WHEN يتم فحص جدول dispatch_channels THEN يجب أن تكون جميع الأعمدة الأساسية بـ NOT NULL
2.11 WHEN يتم فحص جدول accounting_transactions THEN يجب أن تكون الأعمدة type و status بـ NOT NULL
2.12 WHEN يتم إدخال بيانات بدون قيم مطلوبة THEN يجب أن يرفض قاعدة البيانات العملية

#### 2.4 توحيد الدقة في الأعمدة المالية

2.13 WHEN يتم فحص الأعمدة المالية THEN يجب أن تستخدم جميعها DECIMAL(12,4)
2.14 WHEN يتم إجراء عمليات حسابية مالية THEN يجب أن تكون النتائج متسقة
2.15 WHEN يتم تخزين المبالغ المالية THEN يجب أن تكون الدقة موحدة في جميع الجداول

#### 2.5 تشفير SSL/TLS مفعل للاتصال بقاعدة البيانات

2.16 WHEN يتم فحص ملف .env THEN يجب أن يكون DB_SSLMODE=require أو verify-full
2.17 WHEN يتم الاتصال بقاعدة البيانات THEN يجب أن يكون الاتصال مشفراً بـ TLS
2.18 WHEN يتم اعتراض حركة الشبكة THEN يجب أن تكون البيانات مشفرة وغير قابلة للقراءة

#### 2.6 Row Level Security (RLS) مفعل على جميع الجداول

2.19 WHEN يتم فحص جدول messages THEN يجب أن يكون RLS مفعلاً
2.20 WHEN يتم فحص جدول balance_history THEN يجب أن يكون RLS مفعلاً
2.21 WHEN يتم فحص جدول devices THEN يجب أن يكون RLS مفعلاً
2.22 WHEN يتم فحص جدول channels THEN يجب أن يكون RLS مفعلاً
2.23 WHEN يتم فحص جدول users THEN يجب أن يكون RLS مفعلاً
2.24 WHEN يتم فحص جدول audit_trail THEN يجب أن يكون RLS مفعلاً
2.25 WHEN يحاول مستأجر الوصول لبيانات مستأجر آخر THEN يجب أن يتم رفض الوصول

#### 2.7 فهارس موجودة على أعمدة tenant_id

2.26 WHEN يتم فحص جدول messages THEN يجب أن يوجد فهرس مركب على (tenant_id, status, created_at)
2.27 WHEN يتم فحص جدول balance_history THEN يجب أن يوجد فهرس مركب على (tenant_id, type, created_at)
2.28 WHEN يتم فحص جدول accounting_transactions THEN يجب أن يوجد فهرس على tenant_id و status
2.29 WHEN يتم فحص جدول subscriptions THEN يجب أن يوجد فهرس على tenant_id و expires_at
2.30 WHEN يتم تنفيذ استعلامات على tenant_id THEN يجب أن تكون سريعة ومؤشرات

#### 2.8 قيود CHECK موجودة للأعمدة المالية

2.31 WHEN يتم فحص جدول messages THEN يجب أن يوجد CHECK constraint لـ cost >= 0
2.32 WHEN يتم فحص جدول balance_history THEN يجب أن يوجد CHECK constraint لـ amount != 0
2.33 WHEN يتم فحص جدول accounting_transactions THEN يجب أن يوجد CHECK constraint لـ amount > 0
2.34 WHEN يتم إدخال قيمة سالبة في عمود مالي THEN يجب أن ترفض قاعدة البيانات العملية

#### 2.9 Circuit Breaker مع Logger كامل

2.35 WHEN يحدث خطأ في Circuit Breaker THEN يجب تسجيل جميع الأحداث في Logger
2.36 WHEN تدخل الدائرة في حالة نصف مفتوحة (half-open) THEN يجب تسجيل ذلك
2.37 WHEN تعود الدائرة لحالة مغلقة (closed) THEN يجب تسجيل ذلك
2.38 WHEN يتم فتح الدائرة (open state) THEN يجب إرسال تنبيه أو إشعار

#### 2.10 كود مشترك وموحد

2.39 WHEN يتم فحص axios.js في admin-web THEN يجب أن يستورد من مكتبة مشتركة
2.40 WHEN يتم فحص hooks المصادقة THEN يجب أن تكون موحدة في مكتبة مشتركة
2.41 WHEN يتم تعديل كود مشترك THEN يجب تعديله في مكان واحد فقط
2.42 WHEN يتم إضافة ميزة جديدة THEN يجب أن تكون متاحة لكلا المشروعين

#### 2.11 Rate Limiting غير قابل للتحايل

2.43 WHEN يتم فحص Rate Limiting middleware THEN يجب أن يعتمد على معرف المستخدم وليس فقط IP
2.44 WHEN يتم تجاوز Rate Limit THEN يجب حجب الطلب بغض النظر عن IP
2.45 WHEN يتم استخدام Proxy أو VPN THEN يجب أن يتم الحجب بشكل صحيح
2.46 WHEN يتم تجاوز Rate Limit THEN يجب إرجاع رمز حالة 429

#### 2.12 إدارة CSRF Token متقدمة

2.47 WHEN يتم فحص إدارة CSRF Token THEN يجب أن يوجد تجديد تلقائي للـ token
2.48 WHEN تنتهي صلاحية CSRF Token THEN يجب تجديدها تلقائياً
2.49 WHEN يتم إرسال طلب بدون CSRF Token صحيح THEN يجب إرجاع خطأ واضح
2.50 WHEN يتم تحميل التطبيق THEN يجب جلب CSRF Token تلقائياً

#### 2.13 Triggers موجودة لتحديث updated_at

2.51 WHEN يتم تعديل صف في أي جدول THEN يجب أن يو��د Trigger لتحديث updated_at
2.52 WHEN يتم تعديل بيانات مستأجر THEN يجب تحديث updated_at تلقائياً
2.53 WHEN يتم تعديل رسالة THEN يجب تحديث updated_at تلقائياً
2.54 WHEN يتم تعديل أي صف في الجداول المتأثرة THEN يجب تحديث updated_at

#### 2.14 توحيد UUID Generation

2.55 WHEN يتم فحص جدول tenants THEN يجب أن يستخدم UUID للمعرف
2.56 WHEN يتم فحص جدول messages THEN يجب أن يستخدم UUID لـ message_id
2.57 WHEN يتم فحص جميع الجداول THEN يجب أن تكون أنواع البيانات متسقة
2.58 WHEN يتم الربط بين الجداول THEN يجب أن تكون أنواع البيانات متوافقة

#### 2.15 توثيق واضح وكامل

2.59 WHEN يتم قراءة التوثيق THEN يجب أن يوجد شرح واضح للإصلاحات المطلوبة
2.60 WHEN يتم محاولة فهم هيكل المشروع THEN يجب أن يكون التوثيق شاملاً
2.61 WHEN يتم البحث عن معلومات أمنية THEN يجب أن يوجد قسم مخصص للأمان
2.62 WHEN يتم إنشاء migration جديدة THEN يجب توثيقها بشكل واضح

---

### Unchanged Behavior (Regression Prevention)

#### 3.1 البيانات الموجودة

3.1 WHEN يتم إضافة قيود NOT NULL THEN يجب ألا يتم حذف البيانات الموجودة
3.2 WHEN يتم إضافة قيود CHECK THEN يجب ألا تفشل على البيانات الموجودة
3.3 WHEN يتم تفعيل RLS THEN يجب ألا تؤثر على البيانات الموجودة
3.4 WHEN يتم إضافة فهارس جديدة THEN يجب ألا تؤثر على أداء النظام بشكل كبير

#### 3.2 الوظائف الموجودة

3.5 WHEN يتم إرسال رسائل THEN يجب أن تستمر الوظيفة في العمل بشكل صحيح
3.6 WHEN يتم عرض الرصيد THEN يجب أن تستمر البيانات في الظهور بشكل صحيح
3.7 WHEN يتم تسجيل الدخول THEN يجب أن تستمر عملية المصادقة في العمل
3.8 WHEN يتم عرض التقارير THEN يجب أن تستمر البيانات في الظهور بشكل صحيح

#### 3.3 APIs الموجودة

3.9 WHEN يتم استدعاء APIs الموجودة THEN يجب أن تستمر في إرجاع نفس الاستجابات
3.10 WHEN يتم إنشاء APIs جديدة للصلاحيات THEN يجب ألا تؤثر على APIs الموجودة
3.11 WHEN يتم تعديل Circuit Breaker THEN يجب ألا يؤثر على الوظائف الموجودة

#### 3.4 الاختبارات الموجودة

3.12 WHEN يتم تشغيل الاختبارات الموجودة THEN يجب أن تستمر جميعها في النجاح
3.13 WHEN يتم إضافة اختبارات جديدة للصلاحيات THEN يجب ألا تفشل الاختبارات الموجودة
3.14 WHEN يتم تشغيل Property Tests الموجودة THEN يجب أن تستمر في النجاح

#### 3.5 التوافق مع الكود الموجود

3.15 WHEN يتم تعديل ملفات البيئة THEN يجب ألا يؤثر على الكود الموجود
3.16 WHEN يتم إضافة متغيرات جديدة للبيئة THEN يجب أن تكون اختيارية
3.17 WHEN يتم تعديل Circuit Breaker Logger THEN يجب ألا يكسر الكود الموجود
3.18 WHEN يتم توحيد UUID Generation THEN يجب توفير دالة توافقية

---

## ملخص المشاكل والأولويات

### المشاكل الحرجة (5)

| الرقم | المشكلة | الملفات/الجداول المتأثرة | طريقة الإصلاح |
|-------|---------|-------------------------|---------------|
| 1 | بيانات اعتماد مكشوفة | .env files | تشفير وإزالة من Git |
| 2 | RBAC غير مكتمل | قاعدة البيانات | إنشاء الجداول المفقودة |
| 3 | قيود NOT NULL مفقودة | messages, dispatch_channels, accounting_transactions | ALTER TABLE |
| 4 | عدم توحيد الدقة المالية | جميع الجداول المالية | ALTER COLUMN |
| 5 | SSL/TLS معطل | .env files | تغيير DB_SSLMODE |

### المشاكل العالية (5)

| الرقم | المشكلة | الملفات/الجداول المتأثرة | طريقة الإصلاح |
|-------|---------|-------------------------|---------------|
| 6 | RLS غير مفعل | 6 جداول | ALTER TABLE ENABLE ROW LEVEL SECURITY |
| 7 | فهارس مفقودة | tenant_id columns | CREATE INDEX |
| 8 | قيود CHECK مفقودة | الأعمدة المالية | ALTER TABLE ADD CONSTRAINT |
| 9 | Circuit Breaker بدون Logger | backend resilience | إضافة Logger |
| 10 | كود مكرر | admin-web, client-web | إنشاء مكتبة مشتركة |

### المشاكل المتوسطة (5)

| الرقم | المشكلة | الملفات/الجداول المتأثرة | طريقة الإصلاح |
|-------|---------|-------------------------|---------------|
| 11 | Rate Limiting قابل للتحايل | backend middleware | تحسين المنطق |
| 12 | CSRF Token بسيط | frontend axios | تحسين الإدارة |
| 13 | Triggers مفقودة | updated_at columns | CREATE TRIGGER |
| 14 | عدم اتساق UUID | tenants, messages | توحيد الأنواع |
| 15 | توثيق غير واضح | docs folder | تحسين التوثيق |

---

## المتطلبات للتوثيق

لكل إصلاح، يجب توثيق الآتي:
- وصف المشكلة
- الملفات/الجداول المتأثرة
- خطوات الإصلاح
- طريقة الاختبار
- التحقق من الإصلاح

---

**تم إنشاء هذه الوثيقة:** 24 مايو 2026  
**نطاق العمل:** إصلاح جميع المشاكل الأمنية والثغرات التقنية