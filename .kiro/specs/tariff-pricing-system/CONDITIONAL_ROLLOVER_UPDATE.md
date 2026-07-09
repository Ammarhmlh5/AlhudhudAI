# تحديث سياسة الترحيل المشروط (Conditional Rollover Policy)

## التاريخ
2026-02-28

## الملخص التنفيذي

تم تحديث نظام ترحيل الرسائل ليصبح مشروطاً بنجاح عملية الدفع التلقائي، بدلاً من الترحيل التلقائي غير المشروط. هذا التحديث يحمي المنصة من الخسائر المالية عند فشل الدفع مع الحفاظ على مرونة النظام.

## المشكلة

في النظام الحالي، يتم ترحيل الرسائل المتبقية تلقائياً إلى الشهر التالي بغض النظر عن نجاح أو فشل عملية الدفع التلقائي. هذا يسبب:
- خسائر مالية للمنصة عندما يفشل الدفع ويستمر العميل في استخدام الرسائل المرحّلة
- عدم وجود آلية لربط الترحيل بنجاح الدفع

## الحل المقترح

### 1. سياسة الترحيل المشروط

**المبدأ الأساسي:**
- عند نهاية الشهر، يتم حساب الرسائل المتبقية وحفظها بحالة "معلقة" (pending)
- عند نجاح الدفع التلقائي، يتم تفعيل الرسائل المرحّلة (تصبح active)
- عند فشل الدفع التلقائي، يتم مصادرة الرسائل المرحّلة (تصبح forfeited)

### 2. المرونة

- يمكن تفعيل أو تعطيل هذه السياسة من واجهة الإدارة
- عند التعطيل، يعود النظام للسلوك القديم (ترحيل مباشر)
- يمكن تطبيق السياسة على مستوى النظام أو على مستوى كل باقة

## التغييرات التقنية

### 1. تحديثات قاعدة البيانات

#### جدول rollover_messages

**حقول جديدة:**
```sql
-- الحالة والترحيل المشروط
status VARCHAR(20) DEFAULT 'pending', -- 'pending', 'active', 'expired', 'consumed', 'forfeited'
payment_status VARCHAR(20), -- 'pending', 'success', 'failed'
payment_verified_at TIMESTAMP, -- تاريخ التحقق من الدفع
forfeited_reason VARCHAR(255), -- سبب المصادرة
```

**الحالات الجديدة:**
- `pending`: الرسائل محفوظة مؤقتاً بانتظار نتيجة الدفع
- `active`: الرسائل مفعّلة ويمكن استخدامها (بعد نجاح الدفع)
- `forfeited`: الرسائل مصادرة بسبب فشل الدفع

**Index جديد:**
```sql
CREATE INDEX idx_rollover_tenant_payment ON rollover_messages(tenant_id, payment_status);
```

### 2. تحديثات الكود

#### Rollover Manager (internal/rollover/manager.go)

**وظائف جديدة:**

1. **ActivateRolloverOnPaymentSuccess**
   - تفعيل الرسائل المرحّلة عند نجاح الدفع
   - تحديث status من 'pending' إلى 'active'
   - تحديث payment_status إلى 'success'
   - تسجيل payment_verified_at

2. **ForfeitRolloverOnPaymentFailure**
   - مصادرة الرسائل المرحّلة عند فشل الدفع
   - تحديث status من 'pending' إلى 'forfeited'
   - تحديث payment_status إلى 'failed'
   - تسجيل forfeited_reason
   - إرسال إشعار للعميل

**تحديثات على الوظائف الموجودة:**

1. **RolloverMessages**
   - دعم حفظ الرسائل بحالة 'pending' أو 'active' حسب إعدادات السياسة
   - إذا كانت السياسة مفعّلة: حفظ بحالة 'pending'
   - إذا كانت السياسة معطّلة: حفظ بحالة 'active' (السلوك القديم)

2. **UseRolloverMessages**
   - استخدام الرسائل بحالة 'active' فقط
   - تجاهل الرسائل بحالة 'pending' أو 'forfeited'

3. **CleanupExpiredRollover**
   - حذف الرسائل المنتهية
   - حذف الرسائل بحالة 'forfeited' أيضاً

#### Auto Renewal Service (internal/billing/auto_renewal.go)

**تحديثات:**

1. **RenewSubscription**
   - عند نجاح الدفع: استدعاء `ActivateRolloverOnPaymentSuccess`
   - تفعيل جميع الرسائل المرحّلة المعلقة للاشتراك

2. **SuspendSubscription**
   - عند التعليق النهائي: استدعاء `ForfeitRolloverOnPaymentFailure`
   - مصادرة جميع الرسائل المرحّلة المعلقة

3. **AutoReactivate**
   - عند نجاح إعادة التفعيل: تفعيل الرسائل المرحّلة إذا كانت لا تزال بحالة 'pending'

### 3. تحديثات الواجهة

#### واجهة الإدارة (apps/admin-web)

**PackageSettings Component:**
- إضافة زر تحكم (Toggle) لتفعيل/تعطيل سياسة الترحيل المشروط
- إضافة شرح توضيحي: "عند التفعيل، يتم ترحيل الرسائل فقط عند نجاح الدفع التلقائي"
- عرض حالة السياسة الحالية (مفعّلة/معطّلة)

#### واجهة العميل (apps/client-web)

**MessageUsage Component:**
- عرض الرسائل المرحّلة بحالة 'active' فقط
- إخفاء الرسائل بحالة 'pending' (لأنها غير مؤكدة بعد)
- عرض رسالة توضيحية عند وجود رسائل معلقة

## تدفق العمل (Workflow)

### السيناريو 1: نجاح الدفع التلقائي

```
1. نهاية الشهر
   └─> حساب الرسائل المتبقية (مثلاً: 500 رسالة)
   └─> إنشاء سجل rollover بحالة 'pending'
       - status = 'pending'
       - payment_status = 'pending'
       - messages_count = 500

2. موعد التجديد
   └─> محاولة خصم قيمة الباقة
   └─> نجاح الخصم ✓
   └─> استدعاء ActivateRolloverOnPaymentSuccess
       - status = 'active'
       - payment_status = 'success'
       - payment_verified_at = NOW()

3. النتيجة
   └─> العميل يمكنه استخدام 500 رسالة مرحّلة
   └─> الرسائل متاحة للاستخدام بترتيب FIFO
```

### السيناريو 2: فشل الدفع التلقائي

```
1. نهاية الشهر
   └─> حساب الرسائل المتبقية (مثلاً: 500 رسالة)
   └─> إنشاء سجل rollover بحالة 'pending'
       - status = 'pending'
       - payment_status = 'pending'
       - messages_count = 500

2. موعد التجديد
   └─> محاولة خصم قيمة الباقة
   └─> فشل الخصم ✗ (رصيد غير كافي)
   └─> تطبيق فترة السماح (3 أيام)
   └─> الرسائل المرحّلة تبقى بحالة 'pending'

3. نهاية فترة السماح
   └─> لا يزال الرصيد غير كافي
   └─> استدعاء SuspendSubscription
   └─> استدعاء ForfeitRolloverOnPaymentFailure
       - status = 'forfeited'
       - payment_status = 'failed'
       - forfeited_reason = 'payment_failed'
   └─> إرسال إشعار للعميل

4. النتيجة
   └─> العميل يفقد 500 رسالة مرحّلة
   └─> حماية المنصة من الخسائر
```

### السيناريو 3: إعادة التفعيل بعد التعليق

```
1. الاشتراك معلق
   └─> الرسائل المرحّلة بحالة 'forfeited'

2. العميل يشحن رصيد كافي
   └─> استدعاء AutoReactivate
   └─> نجاح التجديد ✓
   └─> لكن الرسائل المرحّلة السابقة ضاعت (forfeited)

3. النتيجة
   └─> الاشتراك مفعّل مع رسائل جديدة
   └─> الرسائل المرحّلة السابقة لا تعود
```

## الإشعارات

### إشعارات جديدة

1. **عند مصادرة الرسائل المرحّلة:**
   ```
   العنوان: تنبيه: فقدان الرسائل المرحّلة
   الرسالة: للأسف، تم فقدان 500 رسالة مرحّلة بسبب فشل تجديد الاشتراك.
           يرجى شحن رصيدك لتجنب فقدان الرسائل في المستقبل.
   ```

2. **عند تفعيل الرسائل المرحّلة:**
   ```
   العنوان: تم تفعيل الرسائل المرحّلة
   الرسالة: تم تجديد اشتراكك بنجاح! لديك الآن 500 رسالة مرحّلة متاحة للاستخدام.
   ```

## الفوائد

### للمنصة
1. **حماية من الخسائر المالية**: عدم السماح باستخدام رسائل مرحّلة عند فشل الدفع
2. **تحفيز الدفع في الوقت المحدد**: العملاء سيحرصون على توفير رصيد كافي
3. **عدالة في الخدمة**: من يدفع يحصل على الخدمة

### للعميل
1. **الشفافية**: معرفة واضحة بأن الترحيل مرتبط بالدفع
2. **المرونة**: إمكانية الاستفادة من الرسائل المرحّلة عند الدفع في الوقت
3. **الإشعارات**: تنبيهات واضحة عند فقدان الرسائل

## التوافق مع الأنظمة الأخرى

### المحفظة (Wallet System) - مستقبلاً
- عند تكامل نظام المحفظة، سيتم الخصم من المحفظة أولاً
- نفس المنطق ينطبق: نجاح الخصم = تفعيل الترحيل، فشل الخصم = مصادرة

### بوابات الدفع (Payment Gateways) - مستقبلاً
- عند تكامل بوابات الدفع الإلكترونية (بطاقات، إلخ)
- نفس المنطق ينطبق: نجاح الدفع = تفعيل الترحيل، فشل الدفع = مصادرة

## الاختبارات المطلوبة

### Property-Based Tests

1. **Property: الترحيل المشروط عند نجاح الدفع**
   - عند نجاح الدفع، جميع الرسائل المرحّلة المعلقة يجب أن تصبح 'active'
   - payment_status يجب أن يكون 'success'

2. **Property: مصادرة الترحيل عند فشل الدفع**
   - عند فشل الدفع النهائي، جميع الرسائل المرحّلة المعلقة يجب أن تصبح 'forfeited'
   - payment_status يجب أن يكون 'failed'

3. **Property: عدم استخدام رسائل معلقة أو مصادرة**
   - UseRolloverMessages يجب أن يستخدم فقط الرسائل بحالة 'active'
   - الرسائل بحالة 'pending' أو 'forfeited' يجب تجاهلها

### Integration Tests

1. **تدفق كامل: نجاح الدفع**
   - إنشاء اشتراك → نهاية الشهر → ترحيل معلق → نجاح الدفع → تفعيل الترحيل

2. **تدفق كامل: فشل الدفع**
   - إنشاء اشتراك → نهاية الشهر → ترحيل معلق → فشل الدفع → مصادرة الترحيل

3. **تدفق كامل: إعادة التفعيل**
   - فشل الدفع → مصادرة → شحن رصيد → إعادة تفعيل → رسائل جديدة فقط

## Migration Script

سيتم إنشاء migration script لتحديث الجدول الموجود:

```sql
-- Migration: Update rollover_messages for conditional rollover
-- Version: 1.1
-- Date: 2026-02-28

BEGIN;

-- 1. Add new columns
ALTER TABLE rollover_messages
ADD COLUMN payment_status VARCHAR(20),
ADD COLUMN payment_verified_at TIMESTAMP,
ADD COLUMN forfeited_reason VARCHAR(255);

-- 2. Update status column to support new values
ALTER TABLE rollover_messages
DROP CONSTRAINT IF EXISTS check_rollover_status_valid;

ALTER TABLE rollover_messages
ADD CONSTRAINT check_rollover_status_valid 
CHECK (status IN ('pending', 'active', 'expired', 'consumed', 'forfeited'));

-- 3. Add payment_status constraint
ALTER TABLE rollover_messages
ADD CONSTRAINT check_payment_status_valid 
CHECK (payment_status IN ('pending', 'success', 'failed') OR payment_status IS NULL);

-- 4. Update existing records
-- All existing 'active' records are considered successfully paid
UPDATE rollover_messages
SET payment_status = 'success',
    payment_verified_at = created_at
WHERE status = 'active';

-- 5. Create new index
CREATE INDEX idx_rollover_tenant_payment ON rollover_messages(tenant_id, payment_status);

-- 6. Update functions to handle new statuses
-- (يتم تحديث الدوال الموجودة لدعم الحالات الجديدة)

COMMIT;
```

## الجدول الزمني للتنفيذ

1. **المرحلة 1 (أسبوع 1)**: تحديث قاعدة البيانات
   - إنشاء migration script
   - تحديث الجدول والـ constraints
   - اختبار على بيئة التطوير

2. **المرحلة 2 (أسبوع 2)**: تحديث الكود
   - تحديث Rollover Manager
   - تحديث Auto Renewal Service
   - كتابة Unit Tests

3. **المرحلة 3 (أسبوع 3)**: تحديث الواجهات
   - إضافة Toggle في واجهة الإدارة
   - تحديث عرض الرسائل المرحّلة في واجهة العميل
   - اختبار UI/UX

4. **المرحلة 4 (أسبوع 4)**: الاختبار والنشر
   - Property-Based Tests
   - Integration Tests
   - اختبار على بيئة Staging
   - النشر التدريجي (Gradual Rollout)

## الخلاصة

هذا التحديث يحول نظام الترحيل من نظام غير مشروط إلى نظام مشروط يحمي المنصة من الخسائر مع الحفاظ على تجربة مستخدم جيدة. التنفيذ مرن ويسمح بالتحكم الكامل من واجهة الإدارة.

## الملفات المحدثة

1. `.kiro/specs/tariff-pricing-system/requirements.md` - تحديث المتطلب 6
2. `.kiro/specs/tariff-pricing-system/design.md` - تحديث schema و workflow
3. `.kiro/specs/tariff-pricing-system/tasks.md` - إضافة مهام جديدة (8.8, 8.9)
4. `services/backend/migrations/018_rollover_messages_table.sql` - سيتم تحديثه
5. `services/backend/internal/rollover/manager.go` - سيتم تحديثه
6. `services/backend/internal/billing/auto_renewal.go` - سيتم تحديثه
7. `apps/admin-web/src/components/PackageSettings.tsx` - سيتم تحديثه
