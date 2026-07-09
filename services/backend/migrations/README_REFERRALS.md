# Referrals Table Migration

## نظرة عامة

هذا الـ Migration ينشئ جدول `referrals` لنظام الإحالة (Referral System) الذي يسمح للعملاء بدعوة أصدقائهم والحصول على مكافآت.

## المتطلبات المغطاة

- **Requirement 33.1**: توليد كود إحالة فريد لكل عميل
- **Requirement 33.3**: تسجيل العلاقة بين المُحيل والمُحال
- **Requirement 33.8**: تسجيل جميع عمليات الإحالة والمكافآت

## بنية الجدول

### الحقول الأساسية

| الحقل | النوع | الوصف |
|------|------|-------|
| `id` | SERIAL | المعرف الفريد للإحالة |
| `referrer_tenant_id` | UUID | معرف المُحيل (الذي دعا صديقه) |
| `referee_tenant_id` | UUID | معرف المُحال (الذي تم دعوته) |
| `referral_code` | VARCHAR(50) | كود الإحالة المستخدم |

### حالة الإحالة

| الحقل | النوع | القيم المسموحة | الوصف |
|------|------|----------------|-------|
| `status` | VARCHAR(50) | pending, completed, rewarded, cancelled | حالة الإحالة |

**الحالات:**
- `pending`: الإحالة معلقة (المُحال سجّل لكن لم يستوفِ الشروط)
- `completed`: الإحالة مكتملة (المُحال استوفى الشروط)
- `rewarded`: تم منح المكافأة للمُحيل
- `cancelled`: الإحالة ملغاة

### معلومات المكافأة

| الحقل | النوع | القيم المسموحة | الوصف |
|------|------|----------------|-------|
| `reward_type` | VARCHAR(50) | balance, messages, discount | نوع المكافأة |
| `reward_amount` | DECIMAL(10,2) | > 0 | قيمة المكافأة |
| `reward_granted` | BOOLEAN | true/false | هل تم منح المكافأة؟ |
| `reward_granted_at` | TIMESTAMP | - | تاريخ منح المكافأة |

**أنواع المكافآت:**
- `balance`: رصيد مالي (بالدولار أو الريال)
- `messages`: رسائل مجانية
- `discount`: خصم بالنسبة المئوية

### شروط المكافأة

| الحقل | النوع | القيم المسموحة | الوصف |
|------|------|----------------|-------|
| `reward_condition` | VARCHAR(100) | first_payment, first_subscription, immediate | شرط منح المكافأة |
| `condition_met` | BOOLEAN | true/false | هل تم استيفاء الشرط؟ |
| `condition_met_at` | TIMESTAMP | - | تاريخ استيفاء الشرط |

**الشروط:**
- `immediate`: منح المكافأة فوراً عند التسجيل
- `first_subscription`: منح المكافأة عند أول اشتراك
- `first_payment`: منح المكافأة عند أول دفعة

### حقول إضافية

| الحقل | النوع | الوصف |
|------|------|-------|
| `notes` | TEXT | ملاحظات إضافية |
| `metadata` | JSONB | بيانات إضافية مرنة |
| `created_at` | TIMESTAMP | تاريخ الإنشاء |
| `updated_at` | TIMESTAMP | تاريخ آخر تحديث |

## القيود (Constraints)

### 1. المُحيل والمُحال مختلفان
```sql
CONSTRAINT check_referrer_referee_different 
CHECK (referrer_tenant_id != referee_tenant_id)
```
لا يمكن للعميل إحالة نفسه.

### 2. قيمة المكافأة موجبة
```sql
CONSTRAINT check_reward_amount_positive 
CHECK (reward_amount IS NULL OR reward_amount > 0)
```

### 3. حالة صحيحة
```sql
CONSTRAINT check_status_valid 
CHECK (status IN ('pending', 'completed', 'rewarded', 'cancelled'))
```

### 4. نوع مكافأة صحيح
```sql
CONSTRAINT check_reward_type_valid 
CHECK (reward_type IS NULL OR reward_type IN ('balance', 'messages', 'discount'))
```

## Indexes

### 1. Index للمُحيل
```sql
CREATE INDEX idx_referrals_referrer 
ON referrals(referrer_tenant_id, created_at DESC);
```
**الاستخدام:** البحث عن جميع إحالات مُحيل معين.

### 2. Index للمُحال
```sql
CREATE INDEX idx_referrals_referee 
ON referrals(referee_tenant_id);
```
**الاستخدام:** التحقق من عدم استخدام العميل لكود إحالة سابقاً.

### 3. Index لكود الإحالة
```sql
CREATE INDEX idx_referrals_code 
ON referrals(referral_code);
```
**الاستخدام:** البحث السريع عن إحالة بكود معين.

### 4. Index للحالة
```sql
CREATE INDEX idx_referrals_status 
ON referrals(status, created_at DESC);
```
**الاستخدام:** معالجة الإحالات المعلقة أو المكتملة.

### 5. Index للمكافآت المعلقة
```sql
CREATE INDEX idx_referrals_reward_pending 
ON referrals(reward_granted, condition_met) 
WHERE reward_granted = false AND condition_met = true;
```
**الاستخدام:** معالجة المكافآت التي استوفت الشروط ولم تُمنح بعد.

## Row Level Security (RLS)

### Policy للقراءة
```sql
CREATE POLICY referrals_select_policy ON referrals
    FOR SELECT
    USING (
        referrer_tenant_id = current_setting('app.current_tenant')::uuid
        OR referee_tenant_id = current_setting('app.current_tenant')::uuid
    );
```
**القاعدة:** يمكن للمُحيل والمُحال رؤية الإحالة فقط.

### Policy للإدراج
```sql
CREATE POLICY referrals_insert_policy ON referrals
    FOR INSERT
    WITH CHECK (referee_tenant_id = current_setting('app.current_tenant')::uuid);
```
**القاعدة:** يمكن إنشاء إحالة للمُحال فقط (عند التسجيل).

### Policy للتحديث
```sql
CREATE POLICY referrals_update_policy ON referrals
    FOR UPDATE
    USING (true)
    WITH CHECK (true);
```
**القاعدة:** يتم التحكم عبر application logic (للنظام فقط).

### Policy للحذف
```sql
CREATE POLICY referrals_delete_policy ON referrals
    FOR DELETE
    USING (false);
```
**القاعدة:** لا يُسمح بالحذف للحفاظ على سجل التدقيق.

## Views

### referral_stats
عرض لإحصائيات الإحالات لكل مُحيل:

```sql
CREATE VIEW referral_stats AS
SELECT 
    referrer_tenant_id,
    COUNT(*) as total_referrals,
    COUNT(*) FILTER (WHERE status = 'completed') as completed_referrals,
    COUNT(*) FILTER (WHERE status = 'rewarded') as rewarded_referrals,
    COUNT(*) FILTER (WHERE status = 'pending') as pending_referrals,
    SUM(CASE WHEN reward_granted THEN reward_amount ELSE 0 END) as total_rewards_granted,
    MAX(created_at) as last_referral_date
FROM referrals
GROUP BY referrer_tenant_id;
```

**الحقول:**
- `total_referrals`: إجمالي الإحالات
- `completed_referrals`: الإحالات المكتملة
- `rewarded_referrals`: الإحالات التي تم منح مكافأتها
- `pending_referrals`: الإحالات المعلقة
- `total_rewards_granted`: إجمالي المكافآت الممنوحة
- `last_referral_date`: تاريخ آخر إحالة

## أمثلة الاستخدام

### 1. إنشاء إحالة جديدة
```sql
INSERT INTO referrals (
    referrer_tenant_id,
    referee_tenant_id,
    referral_code,
    reward_type,
    reward_amount,
    reward_condition
) VALUES (
    '123e4567-e89b-12d3-a456-426614174000',  -- المُحيل
    '987fcdeb-51a2-43f7-b890-123456789abc',  -- المُحال
    'REF-JOHN-2024',
    'balance',
    10.00,
    'first_payment'
);
```

### 2. البحث عن إحالات مُحيل معين
```sql
SELECT * FROM referrals
WHERE referrer_tenant_id = '123e4567-e89b-12d3-a456-426614174000'
ORDER BY created_at DESC;
```

### 3. التحقق من استخدام كود إحالة
```sql
SELECT * FROM referrals
WHERE referee_tenant_id = '987fcdeb-51a2-43f7-b890-123456789abc';
```

### 4. تحديث حالة الإحالة عند استيفاء الشرط
```sql
UPDATE referrals
SET 
    condition_met = true,
    condition_met_at = CURRENT_TIMESTAMP,
    status = 'completed'
WHERE id = 1;
```

### 5. منح المكافأة
```sql
UPDATE referrals
SET 
    reward_granted = true,
    reward_granted_at = CURRENT_TIMESTAMP,
    status = 'rewarded'
WHERE id = 1 AND condition_met = true;
```

### 6. عرض إحصائيات مُحيل
```sql
SELECT * FROM referral_stats
WHERE referrer_tenant_id = '123e4567-e89b-12d3-a456-426614174000';
```

### 7. البحث عن المكافآت المعلقة
```sql
SELECT * FROM referrals
WHERE reward_granted = false 
  AND condition_met = true
ORDER BY condition_met_at ASC;
```

## تدفق العمل (Workflow)

### 1. التسجيل باستخدام كود إحالة
```
1. المُحال يدخل كود الإحالة عند التسجيل
2. النظام يتحقق من صحة الكود
3. النظام ينشئ سجل في جدول referrals بحالة 'pending'
4. النظام يحدد نوع المكافأة والشرط
```

### 2. استيفاء شرط المكافأة
```
1. المُحال يقوم بأول دفعة/اشتراك
2. النظام يحدّث condition_met = true
3. النظام يحدّث status = 'completed'
4. النظام يسجل condition_met_at
```

### 3. منح المكافأة
```
1. Cron Job يبحث عن الإحالات المكتملة غير الممنوحة
2. النظام يمنح المكافأة للمُحيل (رصيد/رسائل/خصم)
3. النظام يحدّث reward_granted = true
4. النظام يحدّث status = 'rewarded'
5. النظام يرسل إشعار للمُحيل
```

## الأمان

### 1. Row Level Security
- كل مُحيل يرى إحالاته فقط
- كل مُحال يرى الإحالة التي استخدمها فقط
- لا يمكن للعملاء رؤية إحالات الآخرين

### 2. منع الاحتيال
- لا يمكن للعميل إحالة نفسه
- لا يمكن للعميل استخدام أكثر من كود إحالة واحد
- لا يمكن حذف الإحالات (للحفاظ على سجل التدقيق)

### 3. التحكم في التحديثات
- فقط النظام يمكنه تحديث حالة الإحالات
- فقط النظام يمكنه منح المكافآت

## الصيانة

### 1. تنظيف الإحالات الملغاة القديمة
```sql
-- حذف الإحالات الملغاة الأقدم من سنة (اختياري)
DELETE FROM referrals
WHERE status = 'cancelled' 
  AND created_at < CURRENT_TIMESTAMP - INTERVAL '1 year';
```

### 2. إعادة بناء الـ Indexes
```sql
REINDEX TABLE referrals;
```

### 3. تحليل الجدول
```sql
ANALYZE referrals;
```

## الأداء

### توقعات الحجم
- **عدد الإحالات المتوقع:** 10,000 - 100,000 إحالة/شهر
- **حجم السجل الواحد:** ~200 bytes
- **حجم الجدول المتوقع:** 2-20 MB/شهر

### تحسينات الأداء
1. **Partitioning:** يمكن تقسيم الجدول حسب الشهر إذا زاد الحجم
2. **Archiving:** أرشفة الإحالات القديمة (أقدم من سنة)
3. **Caching:** تخزين إحصائيات المُحيلين في Redis

## التكامل

### مع جدول tenants
```sql
-- Foreign Keys
referrer_tenant_id REFERENCES tenants(id)
referee_tenant_id REFERENCES tenants(id)
```

### مع جدول accounting_transactions
عند منح المكافأة، يتم تسجيل معاملة مالية:
```sql
INSERT INTO accounting_transactions (
    tenant_id,
    transaction_type,
    amount,
    description
) VALUES (
    referrer_tenant_id,
    'REFERRAL_REWARD',
    reward_amount,
    'Referral reward for inviting ' || referee_email
);
```

## الاختبار

### 1. اختبار الإنشاء
```sql
-- يجب أن ينجح
INSERT INTO referrals (referrer_tenant_id, referee_tenant_id, referral_code)
VALUES ('uuid1', 'uuid2', 'CODE123');

-- يجب أن يفشل (نفس المُحيل والمُحال)
INSERT INTO referrals (referrer_tenant_id, referee_tenant_id, referral_code)
VALUES ('uuid1', 'uuid1', 'CODE123');
```

### 2. اختبار RLS
```sql
-- تعيين tenant
SET app.current_tenant = 'uuid1';

-- يجب أن يرى إحالاته فقط
SELECT * FROM referrals;
```

### 3. اختبار الـ Indexes
```sql
EXPLAIN ANALYZE
SELECT * FROM referrals
WHERE referrer_tenant_id = 'uuid1'
ORDER BY created_at DESC;
```

## الترحيل (Migration)

### تطبيق الـ Migration
```bash
psql -U postgres -d sms_gateway -f 027_referrals_table.sql
```

### التراجع (Rollback)
```sql
DROP VIEW IF EXISTS referral_stats;
DROP TRIGGER IF EXISTS trigger_update_referrals_updated_at ON referrals;
DROP FUNCTION IF EXISTS update_referrals_updated_at();
DROP TABLE IF EXISTS referrals CASCADE;
```

## الدعم

للمزيد من المعلومات:
- راجع `docs/REFERRAL_SYSTEM.md`
- راجع `internal/referral/README.md`
- راجع المتطلب 33 في `requirements.md`

## التاريخ

- **2026-02-27**: إنشاء الـ Migration الأولي
- **Version**: 1.0
- **Status**: Ready for Testing
