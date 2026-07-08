# ملخص إصلاح مشاكل قاعدة البيانات

**تاريخ الإصلاح:** 24 مايو 2026  
**ملف الإصلاح:** `migrations/090_database_fixes_and_optimizations.sql`  
**الحالة:** ✅ تم إنشاؤه

---

## 📋 المشاكل المُصلحة

### 🔴 مشاكل حرجة (تم الإصلاح)

| # | المشكلة | الحل |
|---|---------|------|
| 1 | **تعارض أنواع البيانات** - `messages` يستخدم INTEGER بينما الجداول الأخرى تستخدم UUID | إضافة أعمدة UUID جديدة (`tenant_uuid`, `channel_uuid`, `device_uuid`) |
| 2 | **عدم وجود RLS** على `messages`, `balance_history`, `devices`, `channels` | تفعيل RLS وإنشاء سياسات العزل |
| 3 | **API Keys غير مشفرة** | إنشاء جدول `api_keys` جديد مع تشفير |
| 4 | **قيود مفقودة** على صحة البيانات | إضافة 8 CHECK constraints جديدة |

### 🟠 مشاكل كبرى (تم الإصلاح)

| # | المشكلة | الحل |
|---|---------|------|
| 1 | **فهارس مفقودة** (7 فهارس) | إضافة فهارس مركبة ومغطية |
| 2 | **Foreign Keys مفقودة** (7 قيود) | إضافة قيود التكامل المرجعي |
| 3 | **عدم وجود جدول للتدقيق** | إنشاء `schema_changes` table |
| 4 | **فقدان جدول الإحصائيات** | إنشاء `tenant_statistics` table |

---

## 📁 الملفات المُنشأة

```
services/backend/migrations/
└── 090_database_fixes_and_optimizations.sql  ✅ (الإصلاح الشامل)

docs/
└── DATABASE_FIXES_SUMMARY.md                  ✅ (هذا الملف)
```

---

## 🔧 التفاصيل التقنية

### 1. إصلاح تعارض أنواع البيانات

```sql
-- قبل الإصلاح:
channel_id INTEGER  -- في messages
channel_id UUID     -- في dispatch_channels

-- بعد الإصلاح:
ALTER TABLE messages ADD COLUMN channel_uuid UUID;
ALTER TABLE messages ADD CONSTRAINT fk_messages_channel_uuid 
FOREIGN KEY (channel_uuid) REFERENCES dispatch_channels(id);
```

### 2. تفعيل Row Level Security (RLS)

```sql
-- تفعيل RLS على 6 جداول جديدة
ALTER TABLE messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE balance_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE paired_devices ENABLE ROW LEVEL SECURITY;
ALTER TABLE dispatch_channels ENABLE ROW LEVEL SECURITY;
ALTER TABLE dead_letter_queue ENABLE ROW LEVEL SECURITY;
ALTER TABLE accounting_transactions ENABLE ROW LEVEL SECURITY;

-- إنشاء سياسات العزل
CREATE POLICY messages_tenant_isolation ON messages
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant', true)::INTEGER)
    WITH CHECK (tenant_id = current_setting('app.current_tenant', true)::INTEGER);
```

### 3. إضافة القيود المفقودة

```sql
-- التحقق من صحة رقم الهاتف
ALTER TABLE messages ADD CONSTRAINT phone_number_format 
CHECK (phone_number ~ '^\+?[0-9]{10,15}$');

-- التحقق من حالة الرسالة
ALTER TABLE messages ADD CONSTRAINT valid_status 
CHECK (status IN ('PENDING', 'QUEUED', 'PROCESSING', 'SENT', 'DELIVERED', 'FAILED'));

-- التحقق من أولوية الرسالة
ALTER TABLE messages ADD CONSTRAINT valid_priority 
CHECK (priority IN ('low', 'normal', 'high', 'urgent'));

-- التحقق من تاريخ الاشتراك
ALTER TABLE subscriptions ADD CONSTRAINT subscription_dates 
CHECK (expires_at > starts_at);

-- التحقق من مبلغ الفاتورة
ALTER TABLE billing_invoices ADD CONSTRAINT invoice_amount_positive 
CHECK (total_amount > 0);
```

### 4. إضافة الفهارس المفقودة

```sql
-- فهرس مركب للاستعلامات الشائعة
CREATE INDEX idx_messages_tenant_status_created 
ON messages(tenant_id, status, created_at DESC);

-- فهرس مغطي للقراءة السريعة
CREATE INDEX idx_messages_covering 
ON messages(tenant_id, status, created_at DESC) 
INCLUDE (message_id, phone_number, content, cost);

-- فهارس للجداول الأخرى
CREATE INDEX idx_audit_created_at ON audit_trail(created_at DESC);
CREATE INDEX idx_dlq_attempts ON dead_letter_queue(attempts DESC);
CREATE INDEX idx_subscriptions_tenant_expiry ON subscriptions(tenant_id, expires_at);
CREATE INDEX idx_payments_tenant_status_date ON payment_transactions(tenant_id, status, created_at DESC);
```

### 5. إضافة Foreign Keys المفقودة

```sql
-- ربط balance_history بـ accounting_transactions
ALTER TABLE balance_history ADD CONSTRAINT fk_balance_accounting
FOREIGN KEY (tenant_id, message_id)
REFERENCES accounting_transactions(tenant_id, message_id);

-- ربط subscriptions بـ packages
ALTER TABLE subscriptions ADD CONSTRAINT fk_subscription_package
FOREIGN KEY (package_id) REFERENCES packages(id);

-- ربط billing_invoices بـ subscriptions
ALTER TABLE billing_invoices ADD CONSTRAINT fk_invoice_subscription
FOREIGN KEY (subscription_id) REFERENCES subscriptions(id);
```

### 6. تحسينات الأمان

```sql
-- إنشاء جدول API Keys آمن
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    key_hash VARCHAR(255) NOT NULL,
    key_prefix VARCHAR(10) NOT NULL,
    permissions JSONB DEFAULT '{}',
    rate_limit INTEGER DEFAULT 1000,
    expires_at TIMESTAMP,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW()
);

-- إضافة أعمدة الأمان للمستخدمين
ALTER TABLE users 
ADD COLUMN failed_login_attempts INTEGER DEFAULT 0,
ADD COLUMN locked_until TIMESTAMP,
ADD COLUMN last_login_at TIMESTAMP;
```

### 7. تحسينات الأداء

```sql
-- إنشاء Materialized View للملخص اليومي
CREATE MATERIALIZED VIEW messages_daily_summary AS
SELECT 
    tenant_id,
    DATE(created_at) as stat_date,
    COUNT(*) as total_messages,
    COUNT(CASE WHEN status = 'DELIVERED' THEN 1 END) as delivered_count,
    COUNT(CASE WHEN status = 'FAILED' THEN 1 END) as failed_count,
    SUM(COALESCE(cost, 0)) as total_cost
FROM messages
GROUP BY tenant_id, DATE(created_at);

-- إنشاء جدول إحصائيات المستأجرين
CREATE TABLE tenant_statistics (
    tenant_id UUID NOT NULL,
    stat_date DATE NOT NULL,
    total_messages BIGINT DEFAULT 0,
    delivered_messages BIGINT DEFAULT 0,
    failed_messages BIGINT DEFAULT 0,
    total_cost DECIMAL(12, 4) DEFAULT 0,
    PRIMARY KEY (tenant_id, stat_date)
);
```

### 8. التحقق من سلامة البيانات

```sql
-- دالة للتحقق من اتساق الرصيد
CREATE OR REPLACE FUNCTION validate_balance_consistency()
RETURNS TABLE (
    tenant_id INTEGER,
    current_balance DECIMAL,
    calculated_balance DECIMAL,
    difference DECIMAL,
    is_valid BOOLEAN
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        t.id,
        t.balance,
        COALESCE(SUM(CASE 
            WHEN bt.type IN ('DEDUCTION', 'PAYMENT') THEN -bt.amount
            WHEN bt.type IN ('REFUND', 'RECHARGE') THEN bt.amount
            ELSE 0
        END), 0) as calculated_balance,
        t.balance - COALESCE(SUM(CASE 
            WHEN bt.type IN ('DEDUCTION', 'PAYMENT') THEN -bt.amount
            WHEN bt.type IN ('REFUND', 'RECHARGE') THEN bt.amount
            ELSE 0
        END), 0) as difference,
        t.balance = COALESCE(SUM(CASE 
            WHEN bt.type IN ('DEDUCTION', 'PAYMENT') THEN -bt.amount
            WHEN bt.type IN ('REFUND', 'RECHARGE') THEN bt.amount
            ELSE 0
        END), 0) as is_valid
    FROM tenants t
    LEFT JOIN balance_history bt ON t.id = bt.tenant_id
    GROUP BY t.id, t.balance;
END;
$$ LANGUAGE plpgsql;
```

---

## 🚀 كيفية التنفيذ

### الخطوة 1: تشغيل الـ Migration

```bash
cd services/backend

# تشغيل الـ migration
psql -h localhost -U postgres -d sms_gateway -f migrations/090_database_fixes_and_optimizations.sql

# أو إذا كان لديك أداة migration
go run cmd/migrate/main.go up 090
```

### الخطوة 2: التحقق من التنفيذ

```sql
-- التحقق من القيود المضافة
SELECT constraint_name, table_name 
FROM information_schema.table_constraints 
WHERE constraint_name LIKE '%valid_%' OR constraint_name LIKE '%fk_%';

-- التحقق من تفعيل RLS
SELECT relname, relrowsecurity FROM pg_class 
WHERE relname IN ('messages', 'balance_history', 'paired_devices', 
                  'dispatch_channels', 'dead_letter_queue', 'accounting_transactions');

-- التحقق من الفهارس الجديدة
SELECT indexname FROM pg_indexes 
WHERE tablename = 'messages' AND indexname LIKE 'idx_%';
```

### الخطوة 3: التحقق من اتساق البيانات

```sql
-- تشغيل دالة التحقق من اتساق الرصيد
SELECT * FROM validate_balance_consistency();

-- إصلاح أي تناقضات
SELECT fix_balance_inconsistency(tenant_id) 
FROM validate_balance_consistency() WHERE NOT is_valid;
```

---

## 📊 ملخص التغييرات

| الفئة | عدد التغييرات |
|-------|---------------|
| قيود CHECK جديدة | 8 |
| فهارس جديدة | 12 |
| Foreign Keys جديدة | 7 |
| جداول جديدة | 3 |
| سياسات RLS جديدة | 6 |
| دوال SQL جديدة | 5 |

---

## ⚠️ ملاحظات هامة

1. **النسخ الاحتياطي:** يُنصح بأخذ نسخة احتياطية قبل التنفيذ:
   ```bash
   pg_dump -h localhost -U postgres -d sms_gateway > backup_before_090.sql
   ```

2. **وقت التنفيذ:** قد يستغرق التنفيذ عدة دقائق حسب حجم قاعدة البيانات

3. **الـ Rollback:** ملف الـ migration يحتوي على قسمROLLBACK في النهاية

4. **الاختبار:** يُنصح بالاختبار على بيئة التطوير أولاً

---

## 📞 الدعم

للأسئلة أو المشاكل:

1. راجع ملف الـ migration الكامل: `migrations/090_database_fixes_and_optimizations.sql`
2. راجع تقرير المراجعة: `.kiro/DATABASE_REVIEW_REPORT.md`
3. راجع توثيق قاعدة البيانات: `docs/DATABASE_SCHEMA.md`

---

**تم إنشاء هذا الملف:** 24 مايو 2026  
**المشروع:** ALHudhudAI SMS Gateway Platform  
**الحالة:** ✅ جاهز للتنفيذ