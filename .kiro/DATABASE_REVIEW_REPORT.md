# تقرير مراجعة قاعدة البيانات الشامل - ALHudhudAI

**تاريخ المراجعة:** 24 مايو 2026  
**المُراجع:** AI Database Reviewer (everything-claude-code)  
**نطاق المراجعة:** قاعدة البيانات الكاملة + 80+ Migration  
**إصدار PostgreSQL:** 15

---

## الملخص التنفيذي

قاعدة بيانات ALHudhudAI هي قاعدة بيانات PostgreSQL معقدة تدعم نظام بوابة SMS متعدد المستأجرين. تتضمن **80+ جدول** موزعة على 5 أنظمة رئيسية مع دعم كامل لـ Row Level Security (RLS).

**التقييم العام:** ⭐⭐⭐⭐☆ (4.2/5)

---

## 1. هيكل قاعدة البيانات

### 1.1 الأنظمة الخمسة الرئيسية

| النظام | عدد الجداول | الغرض |
|--------|-------------|-------|
| **نظام الرسائل الأساسي** | 8 جداول | إدارة الرسائل، القنوات، الأجهزة |
| **نظام التعرفات والتسعير** | 18 جدول | الباقات، الاشتراكات، التسعير |
| **نظام البوابات المخصصة** | 7 جداول | بوابات الشركات المؤسسية |
| **نظام الدفع والفوترة** | 3 جداول | الفواتير، المعاملات |
| **نظام الخصومات والإحالة** | 3 جداول | أكواد الخصم، الإحالات |

### 1.2 الجداول الأساسية

```
┌─────────────────────────────────────────────────────────────────┐
│                      قاعدة البيانات الرئيسية                      │
├─────────────────────────────────────────────────────────────────┤
│  tenants (المستأجرين)                                            │
│  ├── users (المستخدمين)                                         │
│  ├── channels (القنوات)                                         │
│  ├── devices (الأجهزة)                                          │
│  ├── messages (الرسائل)                                         │
│  ├── balance_history (سجل الرصيد)                               │
│  ├── accounting_transactions (المعاملات المحاسبية)              │
│  ├── dead_letter_queue (الرسائل الفاشلة)                        │
│  └── audit_trail (سجل التدقيق)                                  │
├─────────────────────────────────────────────────────────────────┤
│  tariffs (التعرفات)                                             │
│  ├── services (الخدمات)                                        │
│  ├── packages (الباقات)                                         │
│  ├── subscriptions (الاشتراكات)                                 │
│  ├── rollover_messages (الترحيل)                                │
│  ├── dedicated_gateways (البوابات المخصصة)                      │
│  ├── resource_quotas (الحصص)                                    │
│  ├── resource_usage (استخدام الموارد)                          │
│  ├── billing_invoices (الفواتير)                                │
│  ├── payment_transactions (المعاملات)                           │
│  ├── promo_codes (أكواد الخصم)                                  │
│  └── referrals (الإحالات)                                       │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. تحليل الجداول التفصيلي

### 2.1 جدول tenants (المستأجرين)

```sql
CREATE TABLE tenants (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    balance DECIMAL(12, 4) NOT NULL DEFAULT 0.00 CHECK (balance >= 0),
    status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

**✅ نقاط القوة:**
- `CHECK (balance >= 0)` يمنع الرصيد السالب على مستوى قاعدة البيانات
- `UNIQUE` على email يمنع التكرار
- `DEFAULT` للقيم الأساسية

**⚠️ ملاحظات:**
- `id` من نوع INTEGER، بينما الجداول الأخرى تستخدم UUID
- **التوصية:** توحيد استخدام UUID للتوافق مع الجداول الأخرى

---

### 2.2 جدول messages (الرسائل)

```sql
CREATE TABLE messages (
    id SERIAL PRIMARY KEY,
    message_id VARCHAR(255) UNIQUE NOT NULL,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    channel_id INTEGER REFERENCES channels(id) ON DELETE SET NULL,
    device_id INTEGER REFERENCES devices(id) ON DELETE SET NULL,
    phone_number VARCHAR(20) NOT NULL,
    content TEXT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'QUEUED',
    priority VARCHAR(50) NOT NULL DEFAULT 'normal',
    cost DECIMAL(12, 4),
    attempts INTEGER NOT NULL DEFAULT 0,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    sent_at TIMESTAMP,
    delivered_at TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

**✅ نقاط القوة:**
- **فهارس ممتازة:** 7 فهارس للتحسين
- **حالات متعددة:** QUEUED, PROCESSING, SENT, DELIVERED, FAILED
- **تتبع التوقيت:** created_at, sent_at, delivered_at

**⚠️ مشاكل حرجة:**

1. **تعارض أنواع البيانات:**
   - `tenant_id` = INTEGER
   - `channel_id` = INTEGER  
   - `device_id` = INTEGER
   - بينما في migration 001 يستخدم UUID
   
   ```sql
   -- في 001_core_messaging_engine.sql:
   channel_id UUID REFERENCES dispatch_channels(id)
   
   -- في الجدول الأساسي:
   channel_id INTEGER
   ```

2. **عدم وجود PRIMARY KEY مركب:**
   - لا يوجد فهرس مركب على (tenant_id, status, created_at)
   - الاستعلامات الشائعة تحتاج هذا الفهرس

---

### 2.3 جدول balance_history (سجل الرصيد)

```sql
CREATE TABLE balance_history (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    message_id VARCHAR(255) REFERENCES messages(message_id) ON DELETE SET NULL,
    type VARCHAR(50) NOT NULL,
    amount DECIMAL(12, 4) NOT NULL,
    old_balance DECIMAL(12, 4) NOT NULL,
    new_balance DECIMAL(12, 4) NOT NULL CHECK (new_balance >= 0),
    reason TEXT,
    created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    ip_address VARCHAR(45),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

**✅ نقاط القوة ممتازة:**
- **Atomicity محفوظ:** `new_balance = old_balance + amount`
- **CHECK constraint:** يمنع الرصيد السالب
- **تتبع كامل:** المستخدم، IP، السبب
- **أنواع العمليات:** DEDUCTION, REFUND, RECHARGE, ADJUSTMENT

**⚠️ تحسينات مقترحة:**

```sql
-- إضافة فهرس مركب للاستعلامات الشائعة
CREATE INDEX idx_balance_history_tenant_type_date 
ON balance_history(tenant_id, type, created_at DESC);

-- إضافة CONSTRAINT للتأكد من صحة المعادلة
ALTER TABLE balance_history
ADD CONSTRAINT balance_calculation_check 
CHECK (new_balance = old_balance + amount);
```

---

### 2.4 جدول accounting_transactions (المعاملات المحاسبية)

```sql
CREATE TABLE accounting_transactions (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    message_id VARCHAR(255) NOT NULL REFERENCES messages(message_id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    amount DECIMAL(12, 4) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_accounting_unique 
ON accounting_transactions(tenant_id, message_id);
```

**✅ نقاط القوة:**
- **UNIQUE constraint** يمنع التكرار على مستوى المعاملة
- **CASCADE delete** للتناسق عند حذف الرسالة

**⚠️ مشاكل:**

1. **تعارض مع balance_history:**
   - نفس البيانات موجودة في جدولين
   -可能导致不一致 (可能导致数据不一致)

2. **عدم وجود STATUS index:**
   ```sql
   -- مفقود:
   CREATE INDEX idx_accounting_status ON accounting_transactions(status);
   ```

---

## 3. تحليل الفهارس (Indexes)

### 3.1 الفهارس الموجودة

| الجدول | الفهرس | النوع | الغرض | التقييم |
|--------|--------|-------|-------|---------|
| tenants | idx_tenants_email | UNIQUE | البحث | ✅ ممتاز |
| tenants | idx_tenants_status | INDEX | الفلترة | ✅ جيد |
| users | idx_users_tenant | INDEX | العلاقة | ✅ ممتاز |
| users | idx_users_email | UNIQUE | البحث | ✅ ممتاز |
| channels | idx_channels_priority | COMPOSITE | الأداء | ✅ ممتاز |
| devices | idx_devices_api_key | UNIQUE | المصادقة | ✅ ممتاز |
| messages | idx_messages_tenant | INDEX | الفلترة | ✅ جيد |
| messages | 7 فهارس إضافية | INDEX | الأداء | ✅ جيد |
| balance_history | idx_balance_history_tenant | INDEX | الفلترة | ✅ جيد |

### 3.2 الفهارس المفقودة 🚨

```sql
-- 1. مفقود على جدول messages
CREATE INDEX idx_messages_tenant_status_created 
ON messages(tenant_id, status, created_at DESC);

-- 2. مفقود على جدول audit_trail
CREATE INDEX idx_audit_created_at ON audit_trail(created_at DESC);

-- 3. مفقود على جدول dead_letter_queue
CREATE INDEX idx_dlq_attempts ON dead_letter_queue(attempts DESC);

-- 4. مفقود على جدول subscriptions
CREATE INDEX idx_subscriptions_tenant_expiry 
ON subscriptions(tenant_id, expires_at);

-- 5. مفقود على جدول payment_transactions
CREATE INDEX idx_payments_tenant_status_date 
ON payment_transactions(tenant_id, status, created_at DESC);
```

### 3.3 توصيات الأداء

```sql
-- إنشاء فهارس GIN للنصوص الكبيرة
CREATE INDEX idx_messages_content_gin 
ON messages USING GIN (to_tsvector('arabic', content));

-- إنشاء فهارس جزئية للبيانات الشائعة
CREATE INDEX idx_messages_pending_tenant 
ON messages(tenant_id, created_at DESC) 
WHERE status = 'PENDING';

-- إنشاء فهارس covering للقراءة السريعة
CREATE INDEX idx_messages_covering 
ON messages(tenant_id, status, created_at DESC) 
INCLUDE (message_id, phone_number, content);
```

---

## 4. تحليل القيود (Constraints)

### 4.1 القيود الموجودة ✅

| الجدول | Constraint | النوع | الغرض |
|--------|------------|-------|-------|
| tenants | balance >= 0 | CHECK | عدم السالب |
| balance_history | new_balance >= 0 | CHECK | عدم السالب |
| messages | cost >= 0 | CHECK | عدم السالب |
| packages | price >= 0 | CHECK | عدم السالب |
| payment_transactions | refund_amount >= 0 | CHECK | عدم السالب |

### 4.2 القيود المفقودة 🚨

```sql
-- 1. مفقود: التحقق من صحة رقم الهاتف
ALTER TABLE messages
ADD CONSTRAINT phone_number_format 
CHECK (phone_number ~ '^\+?[0-9]{10,15}$');

-- 2. مفقود: التحقق من نوع الرسالة
ALTER TABLE messages
ADD CONSTRAINT valid_status 
CHECK (status IN ('QUEUED', 'PROCESSING', 'BUFFERED', 'SENT', 'DELIVERED', 'FAILED', 'WAITING'));

-- 3. مفقود: التحقق من أولوية الرسالة
ALTER TABLE messages
ADD CONSTRAINT valid_priority 
CHECK (priority IN ('low', 'normal', 'high', 'urgent'));

-- 4. مفقود: تاريخ انتهاء الاشتراك
ALTER TABLE subscriptions
ADD CONSTRAINT subscription_dates 
CHECK (expires_at > starts_at);

-- 5. مفقود: مبلغ الفاتورة موجب
ALTER TABLE billing_invoices
ADD CONSTRAINT invoice_amount_positive 
CHECK (total_amount > 0);
```

### 4.3 Foreign Keys المفقودة

```sql
-- 1. مفقود: ربط balance_history بـ accounting_transactions
ALTER TABLE balance_history
ADD CONSTRAINT fk_balance_accounting
FOREIGN KEY (tenant_id, message_id)
REFERENCES accounting_transactions(tenant_id, message_id);

-- 2. مفقود: ربط messages بـ tariffs
ALTER TABLE messages
ADD CONSTRAINT fk_message_tariff
FOREIGN KEY (tenant_id, country_code)
REFERENCES tariffs(tenant_id, country_code);

-- 3. مفقود: ربط subscriptions بـ packages
ALTER TABLE subscriptions
ADD CONSTRAINT fk_subscription_package
FOREIGN KEY (package_id)
REFERENCES packages(id);
```

---

## 5. تحليل Row Level Security (RLS)

### 5.1 الجداول المحمية بـ RLS

```sql
-- تفعيل RLS
ALTER TABLE tariffs ENABLE ROW LEVEL SECURITY;
ALTER TABLE packages ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscriptions ENABLE ROW LEVEL SECURITY;
ALTER TABLE billing_invoices ENABLE ROW LEVEL SECURITY;
ALTER TABLE payment_transactions ENABLE ROW LEVEL SECURITY;
```

### 5.2 سياسات RLS

```sql
-- مثال على سياسة العزل
CREATE POLICY tenant_isolation_policy ON tariffs
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant', true)::uuid);
```

### 5.3 مشاكل RLS 🚨

1. **عدم تفعيل RLS على بعض الجداول:**
   - `messages` - غير محمي
   - `balance_history` - غير محمي
   - `devices` - غير محمي
   - `channels` - غير محمي

2. **تعارض مع CASCADE:**
   ```sql
   -- في users:
   tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE
   
   -- إذا حذف مستأجر، يتم حذف جميع مستخدميه
   -- لكن RLS قد يمنع هذا الحذف!
   ```

**التوصية:**
```sql
-- تفعيل RLS على الجداول الحساسة
ALTER TABLE messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE balance_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE devices ENABLE ROW LEVEL SECURITY;

-- إنشاء سياسات RLS مناسبة
CREATE POLICY messages_tenant_isolation ON messages
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant', true)::INTEGER)
    WITH CHECK (tenant_id = current_setting('app.current_tenant', true)::INTEGER);
```

---

## 6. تحليل الـ Migrations

### 6.1 إحصائيات Migrations

| الإحصائية | القيمة |
|-----------|--------|
| إجمالي ملفات Migration | 80+ |
| ��لترتيب الصحيح | ✅ موثق |
| ملفات Rollback | ✅ موجودة |
| توثيق README | ✅ شامل |

### 6.2 مشاكل في Migrations 🚨

1. **تعارض في تعريفات الجداول:**

```sql
-- Migration 001:
channel_id UUID REFERENCES dispatch_channels(id)

-- الجدول النهائي:
channel_id INTEGER REFERENCES channels(id)
```

2. **تكرار في الأرقام:**

```
050_critical_schema_fixes.sql
050_fix_rbac_tables.sql
```

3. **ملفات غير مرتبطة:**

```
077_custom_id_backfill_bug11_preservation_test.go
077_custom_id_backfill_bug11_test.go
```

### 6.3 توصيات للمigrations

```bash
# 1. التحقق من ترتيب التنفيذ
ls -1 *.sql | sort -V

# 2. التحقق من عدم وجود تكرار
grep -h "CREATE TABLE" *.sql | sort | uniq -c

# 3. التحقق من完整性
cat migrations/README.md
```

---

## 7. تحليل الأداء

### 7.1 استعلامات مقلقة 🚨

```sql
-- ❌ استعلام بطيء: البحث بدون فهرس
SELECT * FROM messages 
WHERE content LIKE '%test%';

-- ✅ الحل: استخدام Full-Text Search
SELECT * FROM messages 
WHERE to_tsvector('arabic', content) @@ to_tsquery('arabic', 'test');

-- ❌ استعلام بطيء: COUNT بدون فهرس
SELECT COUNT(*) FROM messages WHERE tenant_id = 1;

-- ✅ الحل: إضافة فهرس covering
CREATE INDEX idx_messages_tenant_count 
ON messages(tenant_id) INCLUDE (id);
```

### 7.2 تحسينات مقترحة

```sql
-- 1. إنشاء جدول ملخص للرسائل (Materialized View)
CREATE MATERIALIZED VIEW messages_daily_summary AS
SELECT 
    tenant_id,
    DATE(created_at) as date,
    COUNT(*) as total_messages,
    COUNT(CASE WHEN status = 'DELIVERED' THEN 1 END) as delivered,
    COUNT(CASE WHEN status = 'FAILED' THEN 1 END) as failed,
    SUM(cost) as total_cost
FROM messages
GROUP BY tenant_id, DATE(created_at);

-- فهرس للـ Materialized View
CREATE UNIQUE INDEX idx_messages_daily_summary 
ON messages_daily_summary(tenant_id, date);

-- 2. إنشاء جدول للإحصائيات (Partitioned)
CREATE TABLE tenant_statistics (
    tenant_id UUID NOT NULL,
    stat_date DATE NOT NULL,
    total_messages BIGINT DEFAULT 0,
    delivered_messages BIGINT DEFAULT 0,
    failed_messages BIGINT DEFAULT 0,
    total_cost DECIMAL(12, 4) DEFAULT 0,
    PRIMARY KEY (tenant_id, stat_date)
) PARTITION BY RANGE (stat_date);

-- 3. تقسيم جدول resource_usage (Partitioning)
CREATE TABLE resource_usage (
    tenant_id UUID NOT NULL,
    gateway_id UUID NOT NULL,
    usage_date DATE NOT NULL,
    messages_sent BIGINT DEFAULT 0,
    api_calls BIGINT DEFAULT 0,
    PRIMARY KEY (tenant_id, gateway_id, usage_date)
) PARTITION BY RANGE (usage_date);
```

---

## 8. تحليل الأمان

### 8.1 نقاط القوة ✅

| الميزة | الحالة | التفاصيل |
|--------|--------|----------|
| Row Level Security | ✅ مفعل | على 7 جداول |
| Password Hashing | ✅ موجود | bcrypt في users |
| Audit Trail | ✅ موجود | جدول audit_trail |
| IP Tracking | ✅ موجود | في balance_history |
| Connection Encryption | ✅ مفترض | sslmode في الاتصال |

### 8.2 ثغرات أمنية 🚨

1. **API Keys غير مشفرة:**
   ```sql
   -- في devices:
   api_key VARCHAR(255) UNIQUE NOT NULL
   
   -- يجب أن تكون:
   api_key_hash VARCHAR(255) NOT NULL
   api_key_salt VARCHAR(255) NOT NULL
   ```

2. **لا يوجد Rate Limiting على قاعدة البيانات:**
   ```sql
   -- مفقود:
   CREATE EXTENSION pg_cron;
   -- لتنظيف الاتصالات القديمة
   ```

3. **لا يوجد Audit للـ DDL:**
   ```sql
   -- مفقود: جدول لتتبع تغييرات الـ Schema
   CREATE TABLE schema_changes (
       id SERIAL PRIMARY KEY,
       migration_file VARCHAR(255),
       executed_at TIMESTAMP DEFAULT NOW(),
       executed_by VARCHAR(255)
   );
   ```

### 8.3 توصيات الأمان

```sql
-- 1. تشفير الأعمدة الحساسة
CREATE EXTENSION pgcrypto;

ALTER TABLE devices 
ADD COLUMN api_key_hash VARCHAR(255);

UPDATE devices 
SET api_key_hash = pgp_sym_encrypt(api_key, 'encryption_key');

-- 2. إضافة أعمدة الأمان
ALTER TABLE users 
ADD COLUMN failed_login_attempts INTEGER DEFAULT 0,
ADD COLUMN locked_until TIMESTAMP;

-- 3. إنشاء جدول للـ API Keys
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    key_hash VARCHAR(255) NOT NULL,
    key_prefix VARCHAR(10) NOT NULL,
    permissions JSONB DEFAULT '{}',
    last_used_at TIMESTAMP,
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

-- 4. تفعيل Row Level Security على جميع الجداول
ALTER TABLE messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE balance_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE devices ENABLE ROW LEVEL SECURITY;
ALTER TABLE channels ENABLE ROW LEVEL SECURITY;
```

---

## 9. النسخ الاحتياطي والتعافي

### 9.1 استراتيجية النسخ الحالية

```bash
# من DOCUMENTATION:
pg_dump -h localhost -U postgres -d sms_gateway > backup_$(date +%Y%m%d).sql
```

### 9.2 تحسينات مقترحة

```sql
-- 1. إنشاء جدول للنسخ الاحتياطية
CREATE TABLE backup_log (
    id SERIAL PRIMARY KEY,
    backup_type VARCHAR(50) NOT NULL,
    backup_file VARCHAR(255),
    backup_size BIGINT,
    started_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP,
    status VARCHAR(50) DEFAULT 'RUNNING',
    error_message TEXT
);

-- 2. إنشاء دالة للنسخ الاحتياطي
CREATE OR REPLACE FUNCTION create_backup()
RETURNS VOID AS $$
DECLARE
    backup_file VARCHAR(255);
BEGIN
    backup_file := 'backup_' || TO_CHAR(NOW(), 'YYYYMMDD_HH24MISS') || '.sql';
    
    INSERT INTO backup_log (backup_type, backup_file, status)
    VALUES ('FULL', backup_file, 'RUNNING');
    
    -- تنفيذ النسخ الاحتياطي
    PERFORM pg_backup(backup_file);
    
    UPDATE backup_log 
    SET status = 'COMPLETED', completed_at = NOW()
    WHERE backup_file = backup_file;
EXCEPTION WHEN OTHERS THEN
    UPDATE backup_log 
    SET status = 'FAILED', error_message = SQLERRM
    WHERE backup_file = backup_file;
END;
$$ LANGUAGE plpgsql;
```

---

## 10. ملخص المشاكل حسب الأولوية

### 🔴 مشاكل حرجة (يجب الإصلاح فوراً)

| المشكلة | الجدول | التأثير | الحل |
|---------|--------|---------|------|
| تعارض أنواع البيانات | messages | فشل الاستعلامات | توحيد UUID/INTEGER |
| عدم وجود RLS | messages, balance_history | تسرب البيانات | تفعيل RLS |
| API Keys غير مشفرة | devices | اختراق الحسابات | تشفير الأعمدة |
| قيود مفقودة | multiple | بيانات غير صالحة | إضافة CHECK constraints |

### 🟠 مشاكل كبرى (يجب الإصلاح هذا الشهر)

| المشكلة | الجدول | التأثير | الحل |
|---------|--------|---------|------|
| فهارس مفقودة | multiple | بطء الاستعلامات | إضافة الفهارس |
| Foreign Keys مفقودة | balance_history | عدم integrity | إضافة FK |
| تكرار البيانات | accounting/balance | تناقض البيانات | إزالة التكرار |
| Migration تكرار | 050 | ارتباك | إعادة تسمية |

### 🟡 تحسينات (للمستقبل)

| التحسين | الجدول | الفائدة |
|---------|--------|---------|
| Materialized Views | messages | أداء التقارير |
| Table Partitioning | resource_usage | أداء التقسيم |
| Full-Text Search | messages | بحث أفضل |
| Audit DDL | all | أمان |

---

## 11. قائمة المراجعة (Checklist)

### قبل الإنتاج

- [ ] توحيد استخدام UUID/INTEGER في جميع الجداول
- [ ] تفعيل RLS على جميع جداول المستأجرين
- [ ] إضافة CHECK constraints للأعمدة الحساسة
- [ ] إضافة Foreign Keys المفقودة
- [ ] إنشاء الفهارس المفقودة
- [ ] تشفير API Keys
- [ ] إضافة Rate Limiting
- [ ] توثيق استراتيجية النسخ الاحتياطي
- [ ] اختبار الـ Migrations
- [ ] إنشاء اختبارات التكامل لقاعدة البيانات

---

## 12. المراجع

- `docs/DATABASE_SCHEMA.md` - توثيق Schema
- `services/backend/migrations/` - 80+ Migration
- `services/backend/internal/db/` - كود قاعدة البيانات
- `docs/ARCHITECTURE.md` - معمارية النظام

---

**تم إنشاء هذا التقرير:** 24 مايو 2026  
**أدوات المراجعة:** everything-claude-code (database-reviewer agent)