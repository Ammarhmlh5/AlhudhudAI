# Database Schema Documentation

## نظرة عامة

هذا المستند يوثق schema قاعدة البيانات الكاملة لنظام إدارة الرسائل SMS ونظام التعرفات والتسعير.

**Database**: PostgreSQL 15  
**Encoding**: UTF-8  
**Timezone**: UTC

## الأنظمة المغطاة

1. **نظام إدارة الرسائل الأساسي** - إدارة الرسائل والقنوات والأجهزة
2. **نظام التعرفات والتسعير** - إدارة التعرفات والباقات والاشتراكات
3. **نظام البوابات المخصصة** - إدارة البوابات المخصصة للشركات المؤسسية
4. **نظام الدفع والفوترة** - إدارة المدفوعات والفواتير
5. **نظام الخصومات والإحالة** - إدارة العروض والإحالات

## Entity Relationship Diagram

### نظام إدارة الرسائل الأساسي
```
┌─────────────┐       ┌──────────────┐       ┌─────────────┐
│   tenants   │───────│   messages   │───────│  channels   │
└─────────────┘   1:N └──────────────┘   N:1 └─────────────┘
      │                      │                       │
      │ 1:N                  │ 1:N                   │ 1:N
      │                      │                       │
┌─────────────┐       ┌──────────────┐       ┌─────────────┐
│    users    │       │   devices    │───────│  channels   │
└─────────────┘       └──────────────┘       └─────────────┘
      │
      │ 1:N
      │
┌─────────────────────┐
│  balance_history    │
└─────────────────────┘
      │
      │ 1:1
      │
┌─────────────────────────┐
│ accounting_transactions │
└─────────────────────────┘
```

### نظام التعرفات والباقات
```
┌─────────────┐       ┌──────────────┐       ┌─────────────────┐
│   tenants   │───────│   services   │───────│    tariffs      │
└─────────────┘   1:N └──────────────┘   1:N └─────────────────┘
      │
      │ 1:N
      │
┌─────────────────┐       ┌──────────────────┐       ┌──────────────────┐
│    packages     │───────│  subscriptions   │───────│ rollover_messages│
└─────────────────┘   1:N └──────────────────┘   1:N └──────────────────┘
```

### نظام البوابات المخصصة
```
┌─────────────┐       ┌────────────────────┐       ┌──────────────────┐
│   tenants   │───────│ dedicated_gateways │───────│ resource_quotas  │
└─────────────┘   1:N └────────────────────┘   1:1 └──────────────────┘
                              │                              │
                              │ 1:N                          │ 1:N
                              │                              │
                    ┌─────────────────┐           ┌──────────────────┐
                    │ gateway_users   │           │ resource_usage   │
                    └─────────────────┘           └──────────────────┘
                              │
                              │ 1:N
                              │
                    ┌─────────────────────┐
                    │ gateway_api_keys    │
                    └─────────────────────┘
```

### نظام الدفع والفوترة
```
┌─────────────┐       ┌──────────────────────┐       ┌──────────────────┐
│   tenants   │───────│ payment_transactions │───────│ billing_invoices │
└─────────────┘   1:N └──────────────────────┘   1:N └──────────────────┘
```

### نظام الخصومات والإحالة
```
┌─────────────┐       ┌──────────────┐       ┌──────────────┐
│   tenants   │───────│ promo_codes  │───────│  referrals   │
└─────────────┘   1:N └──────────────┘   1:N └──────────────┘
```

## الجداول (Tables)

### 1. tenants

جدول الشركات/العملاء الرئيسيين.

```sql
CREATE TABLE tenants (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    balance DECIMAL(10, 2) NOT NULL DEFAULT 0.00 CHECK (balance >= 0),
    status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tenants_email ON tenants(email);
CREATE INDEX idx_tenants_status ON tenants(status);
```

**Columns:**
- `id`: معرف فريد للشركة
- `name`: اسم الشركة
- `email`: البريد الإلكتروني (فريد)
- `balance`: الرصيد المالي الحالي (لا يمكن أن يكون سالب)
- `status`: حالة الحساب (`ACTIVE`, `SUSPENDED`, `CLOSED`)
- `created_at`: تاريخ الإنشاء
- `updated_at`: تاريخ آخر تحديث

**Constraints:**
- `balance >= 0`: لا يمكن أن يكون الرصيد سالب
- `email UNIQUE`: البريد الإلكتروني فريد

---

### 2. users

جدول المستخدمين.

```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'client',
    status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',
    last_login TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_tenant ON users(tenant_id);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);
```

**Columns:**
- `id`: معرف فريد للمستخدم
- `tenant_id`: معرف الشركة (foreign key)
- `email`: البريد الإلكتروني (فريد)
- `password_hash`: كلمة المرور المشفرة
- `name`: اسم المستخدم
- `role`: الدور (`admin`, `client`, `staff`)
- `status`: حالة الحساب (`ACTIVE`, `SUSPENDED`)
- `last_login`: آخر تسجيل دخول
- `created_at`: تاريخ الإنشاء
- `updated_at`: تاريخ آخر تحديث

---

### 3. channels

جدول القنوات.

```sql
CREATE TABLE channels (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'OFFLINE',
    priority INTEGER NOT NULL DEFAULT 5,
    current_load INTEGER NOT NULL DEFAULT 0,
    max_load INTEGER NOT NULL DEFAULT 100,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_channels_tenant ON channels(tenant_id);
CREATE INDEX idx_channels_status ON channels(status);
CREATE INDEX idx_channels_priority ON channels(tenant_id, status, priority DESC, current_load ASC);
```

**Columns:**
- `id`: معرف فريد للقناة
- `tenant_id`: معرف الشركة (foreign key)
- `name`: اسم القناة
- `status`: حالة القناة (`ONLINE`, `OFFLINE`, `MAINTENANCE`)
- `priority`: أولوية القناة (أعلى = أفضل)
- `current_load`: الحمل الحالي
- `max_load`: الحد الأقصى للحمل
- `created_at`: تاريخ الإنشاء
- `updated_at`: تاريخ آخر تحديث

**Performance Index:**
- `idx_channels_priority`: مُحسّن لاختيار القناة (tenant + status + priority + load)

---

### 4. devices

جدول الأجهزة (Mobile Gateways).

```sql
CREATE TABLE devices (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    channel_id INTEGER REFERENCES channels(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    phone_number VARCHAR(20) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'OFFLINE',
    api_key VARCHAR(255) UNIQUE NOT NULL,
    device_token VARCHAR(255),
    last_seen TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_devices_tenant ON devices(tenant_id);
CREATE INDEX idx_devices_channel ON devices(channel_id);
CREATE INDEX idx_devices_status ON devices(status);
CREATE INDEX idx_devices_api_key ON devices(api_key);
```

**Columns:**
- `id`: معرف فريد للجهاز
- `tenant_id`: معرف الشركة (foreign key)
- `channel_id`: معرف القناة (foreign key, nullable)
- `name`: اسم الجهاز
- `phone_number`: رقم الهاتف
- `status`: حالة الجهاز (`ONLINE`, `OFFLINE`)
- `api_key`: مفتاح API للمصادقة (فريد)
- `device_token`: FCM token للإشعارات
- `last_seen`: آخر ظهور
- `created_at`: تاريخ الإنشاء
- `updated_at`: تاريخ آخر تحديث

---

### 5. messages

جدول الرسائل الرئيسي.

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
    cost DECIMAL(10, 4),
    attempts INTEGER NOT NULL DEFAULT 0,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    sent_at TIMESTAMP,
    delivered_at TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_messages_tenant ON messages(tenant_id, created_at DESC);
CREATE INDEX idx_messages_status ON messages(status);
CREATE INDEX idx_messages_message_id ON messages(message_id);
CREATE INDEX idx_messages_phone ON messages(phone_number);
CREATE INDEX idx_messages_channel ON messages(channel_id);
CREATE INDEX idx_messages_device ON messages(device_id);
```

**Columns:**
- `id`: معرف فريد للرسالة (internal)
- `message_id`: معرف الرسالة (external, فريد)
- `tenant_id`: معرف الشركة (foreign key)
- `channel_id`: معرف القناة (foreign key, nullable)
- `device_id`: معرف الجهاز (foreign key, nullable)
- `phone_number`: رقم الهاتف المستلم
- `content`: محتوى الرسالة
- `status`: حالة الرسالة
- `priority`: أولوية الرسالة (`low`, `normal`, `high`)
- `cost`: تكلفة الرسالة
- `attempts`: عدد المحاولات
- `error_message`: رسالة الخطأ (إن وجدت)
- `created_at`: تاريخ الإنشاء
- `sent_at`: تاريخ الإرسال
- `delivered_at`: تاريخ التسليم
- `updated_at`: تاريخ آخر تحديث

**Message Statuses:**
- `QUEUED`: في قائمة الانتظار
- `PROCESSING`: قيد المعالجة
- `BUFFERED`: في ذاكرة الجهاز
- `SENT`: تم الإرسال
- `DELIVERED`: تم التسليم
- `FAILED`: فشل الإرسال
- `WAITING`: في انتظار قناة متاحة

---

### 6. balance_history

جدول سجل الرصيد (جديد).

```sql
CREATE TABLE balance_history (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    message_id VARCHAR(255) REFERENCES messages(message_id) ON DELETE SET NULL,
    type VARCHAR(50) NOT NULL,
    amount DECIMAL(10, 4) NOT NULL,
    old_balance DECIMAL(10, 2) NOT NULL,
    new_balance DECIMAL(10, 2) NOT NULL,
    reason TEXT,
    created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    ip_address VARCHAR(45),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_balance_history_tenant ON balance_history(tenant_id, created_at DESC);
CREATE INDEX idx_balance_history_message ON balance_history(message_id);
CREATE INDEX idx_balance_history_type ON balance_history(type);
```

**Columns:**
- `id`: معرف فريد للسجل
- `tenant_id`: معرف الشركة (foreign key)
- `message_id`: معرف الرسالة (foreign key, nullable)
- `type`: نوع العملية (`DEDUCTION`, `REFUND`, `RECHARGE`, `ADJUSTMENT`)
- `amount`: المبلغ (موجب للإضافة، سالب للخصم)
- `old_balance`: الرصيد القديم
- `new_balance`: الرصيد الجديد
- `reason`: سبب العملية
- `created_by`: المستخدم الذي أجرى العملية (nullable)
- `ip_address`: عنوان IP
- `created_at`: تاريخ العملية

**Invariant:**
```
new_balance = old_balance + amount
```

---

### 7. accounting_transactions

جدول المعاملات المحاسبية (جديد).

```sql
CREATE TABLE accounting_transactions (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    message_id VARCHAR(255) NOT NULL REFERENCES messages(message_id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    amount DECIMAL(10, 4) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_accounting_tenant ON accounting_transactions(tenant_id, created_at DESC);
CREATE INDEX idx_accounting_message ON accounting_transactions(message_id);
CREATE INDEX idx_accounting_status ON accounting_transactions(status);
CREATE UNIQUE INDEX idx_accounting_unique ON accounting_transactions(tenant_id, message_id);
```

**Columns:**
- `id`: معرف فريد للمعاملة
- `tenant_id`: معرف الشركة (foreign key)
- `message_id`: معرف الرسالة (foreign key)
- `type`: نوع المعاملة (`DEDUCTION`, `REFUND`)
- `amount`: المبلغ
- `status`: حالة المعاملة (`PENDING`, `COMPLETED`, `REFUNDED`)
- `created_at`: تاريخ الإنشاء
- `completed_at`: تاريخ الإكمال
- `updated_at`: تاريخ آخر تحديث

**Unique Constraint:**
- `(tenant_id, message_id)`: معاملة واحدة فقط لكل رسالة

---

### 8. dead_letter_queue

جدول الرسائل الفاشلة (جديد).

```sql
CREATE TABLE dead_letter_queue (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    message_id VARCHAR(255) NOT NULL,
    phone_number VARCHAR(20) NOT NULL,
    content TEXT NOT NULL,
    attempts INTEGER NOT NULL,
    last_error TEXT,
    original_created_at TIMESTAMP NOT NULL,
    moved_to_dlq_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reprocessed BOOLEAN NOT NULL DEFAULT FALSE,
    reprocessed_at TIMESTAMP
);

CREATE INDEX idx_dlq_tenant ON dead_letter_queue(tenant_id, moved_to_dlq_at DESC);
CREATE INDEX idx_dlq_message ON dead_letter_queue(message_id);
CREATE INDEX idx_dlq_reprocessed ON dead_letter_queue(reprocessed);
```

**Columns:**
- `id`: معرف فريد للسجل
- `tenant_id`: معرف الشركة (foreign key)
- `message_id`: معرف الرسالة الأصلية
- `phone_number`: رقم الهاتف
- `content`: محتوى الرسالة
- `attempts`: عدد المحاولات
- `last_error`: آخر رسالة خطأ
- `original_created_at`: تاريخ إنشاء الرسالة الأصلية
- `moved_to_dlq_at`: تاريخ النقل إلى DLQ
- `reprocessed`: هل تمت إعادة المعالجة
- `reprocessed_at`: تاريخ إعادة المعالجة

**Retention Policy:**
- الاحتفاظ بالرسائل لمدة 7 أيام
- حذف تلقائي بعد 7 أيام

---

### 9. audit_trail

جدول سجل التدقيق (جديد).

```sql
CREATE TABLE audit_trail (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER REFERENCES tenants(id) ON DELETE SET NULL,
    user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(255) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    resource_id VARCHAR(255),
    details JSONB,
    ip_address VARCHAR(45),
    user_agent TEXT,
    result VARCHAR(50) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_tenant ON audit_trail(tenant_id, created_at DESC);
CREATE INDEX idx_audit_user ON audit_trail(user_id, created_at DESC);
CREATE INDEX idx_audit_action ON audit_trail(action);
CREATE INDEX idx_audit_resource ON audit_trail(resource_type, resource_id);
```

**Columns:**
- `id`: معرف فريد للسجل
- `tenant_id`: معرف الشركة (nullable)
- `user_id`: معرف المستخدم (nullable)
- `action`: الإجراء المُنفذ
- `resource_type`: نوع المورد (`message`, `balance`, `channel`, etc.)
- `resource_id`: معرف المورد
- `details`: تفاصيل إضافية (JSON)
- `ip_address`: عنوان IP
- `user_agent`: User agent
- `result`: نتيجة الإجراء (`SUCCESS`, `FAILURE`)
- `created_at`: تاريخ الإجراء

**Immutability:**
- هذا الجدول immutable (لا يتم تحديث أو حذف السجلات)

---

### 10. tariffs

جدول التعرفات.

```sql
CREATE TABLE tariffs (
    id SERIAL PRIMARY KEY,
    country_code VARCHAR(2) NOT NULL UNIQUE,
    country_name VARCHAR(100) NOT NULL,
    cost_per_message DECIMAL(10, 4) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'SAR',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tariffs_country ON tariffs(country_code);
```

**Columns:**
- `id`: معرف فريد للتعرفة
- `country_code`: رمز الدولة (ISO 3166-1 alpha-2)
- `country_name`: اسم الدولة
- `cost_per_message`: تكلفة الرسالة الواحدة
- `currency`: العملة
- `created_at`: تاريخ الإنشاء
- `updated_at`: تاريخ آخر تحديث

---

## نظام التعرفات والتسعير (Tariff Pricing System)

نظام التعرفات والتسعير يتضمن 18 جدول إضافي لإدارة:
- التعرفات الديناميكية
- الباقات الشهرية والاشتراكات
- نظام الترحيل (Rollover) مع سياسة الترحيل المشروط
- البوابات المخصصة للشركات المؤسسية
- إدارة الموارد والحصص
- نظام الدفع والفوترة
- أكواد الخصم والإحالات

**للاطلاع على التوثيق الكامل لنظام التعرفات والتسعير، راجع:**
📄 [DATABASE_SCHEMA_TARIFF_SYSTEM.md](./DATABASE_SCHEMA_TARIFF_SYSTEM.md)

### ملخص الجداول الرئيسية

#### جداول التعرفات والباقات
- `services` - الخدمات الثلاثة (بوابة الهدهد، الهاتف، البوابات المخصصة)
- `tariffs` - التعرفات الديناميكية مع RLS
- `segmentation_rules` - قواعد تقسيم الرسائل
- `currencies` - العملات المدعومة
- `exchange_rates` - أسعار الصرف
- `packages` - الباقات الشهرية مع RLS
- `subscriptions` - اشتراكات العملاء مع RLS
- `rollover_messages` - الرسائل المرحّلة مع سياسة الترحيل المشروط

#### جداول البوابات المخصصة
- `dedicated_gateways` - البوابات المخصصة للشركات المؤسسية
- `resource_quotas` - الحصص الموردية لكل بوابة
- `resource_usage` - استخدام الموارد الفعلي (Partitioned)
- `gateway_users` - مستخدمو البوابات
- `gateway_api_keys` - مفاتيح API للبوابات
- `gateway_statistics` - إحصائيات البوابات اليومية

#### جداول الدفع والعروض
- `billing_invoices` - الفواتير المولدة تلقائياً
- `payment_transactions` - معاملات الدفع عبر بوابات الدفع
- `promo_codes` - أكواد الخصم والعروض
- `referrals` - نظام الإحالة

### Row Level Security (RLS)

جميع جداول نظام التعرفات محمية بـ RLS لضمان العزل التام بين الشركات:

```sql
-- مثال على RLS Policy
ALTER TABLE tariffs ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON tariffs
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

**الجداول المحمية بـ RLS:**
- tariffs
- packages
- subscriptions
- rollover_messages
- dedicated_gateways
- billing_invoices
- payment_transactions

---

## River Queue Tables

River Queue يستخدم جداول خاصة به:

```sql
-- River jobs table
CREATE TABLE river_job (
    id BIGSERIAL PRIMARY KEY,
    args JSONB NOT NULL,
    attempt INTEGER NOT NULL DEFAULT 0,
    attempted_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    errors TEXT[],
    finalized_at TIMESTAMP,
    kind VARCHAR(255) NOT NULL,
    max_attempts INTEGER NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    priority INTEGER NOT NULL DEFAULT 1,
    queue VARCHAR(255) NOT NULL DEFAULT 'default',
    state VARCHAR(255) NOT NULL,
    scheduled_at TIMESTAMP NOT NULL,
    attempted_by VARCHAR(255)[]
);

CREATE INDEX idx_river_job_state ON river_job(state);
CREATE INDEX idx_river_job_kind ON river_job(kind);
CREATE INDEX idx_river_job_queue ON river_job(queue);
```

---

## Views

### messages_summary

عرض ملخص الرسائل.

```sql
CREATE VIEW messages_summary AS
SELECT 
    m.tenant_id,
    DATE(m.created_at) as date,
    COUNT(*) as total_messages,
    COUNT(CASE WHEN m.status = 'DELIVERED' THEN 1 END) as delivered,
    COUNT(CASE WHEN m.status = 'FAILED' THEN 1 END) as failed,
    SUM(m.cost) as total_cost
FROM messages m
GROUP BY m.tenant_id, DATE(m.created_at);
```

---

## Functions

### update_updated_at_column()

دالة لتحديث عمود `updated_at` تلقائياً.

```sql
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
```

**Triggers:**
```sql
CREATE TRIGGER update_tenants_updated_at
    BEFORE UPDATE ON tenants
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ... (similar triggers for other tables)
```

---

## Indexes Summary

### Performance Indexes

| Table | Index | Purpose |
|-------|-------|---------|
| tenants | idx_tenants_email | البحث بالبريد الإلكتروني |
| users | idx_users_tenant | فلترة المستخدمين حسب الشركة |
| channels | idx_channels_priority | اختيار القناة الأمثل |
| devices | idx_devices_api_key | المصادقة بـ API key |
| messages | idx_messages_tenant | فلترة الرسائل حسب الشركة |
| balance_history | idx_balance_history_tenant | سجل الرصيد حسب الشركة |
| accounting_transactions | idx_accounting_unique | منع التكرار |

---

## Data Retention Policies

| Table | Retention Period | Action |
|-------|------------------|--------|
| messages | 6 months | Archive to cold storage |
| balance_history | 2 years | Archive to cold storage |
| accounting_transactions | 7 years | Archive to cold storage |
| audit_trail | 1 year | Archive to cold storage |
| dead_letter_queue | 7 days | Delete |

---

## Backup Strategy

### Daily Backups
```bash
pg_dump -h localhost -U postgres -d sms_gateway > backup_$(date +%Y%m%d).sql
```

### Point-in-Time Recovery
- WAL archiving enabled
- Retention: 7 days

---

## Migration Files

الترتيب الصحيح لتنفيذ migrations:

### نظام إدارة الرسائل الأساسي
1. `001_initial_schema.sql` - Schema الأساسي
2. `002_balance_history_table.sql` - جدول سجل الرصيد
3. `003_accounting_transactions_table.sql` - جدول المعاملات المحاسبية
4. `004_dead_letter_queue_table.sql` - جدول DLQ
5. `005_river_queue_tables.sql` - جداول River Queue
6. `006_indexes_optimization.sql` - تحسين Indexes
7. `007_audit_trail_table.sql` - جدول التدقيق

### نظام التعرفات والتسعير
8. `008_tariff_services_table.sql` - جدول الخدمات
9. `009_insert_default_services.sql` - البيانات الأولية للخدمات
10. `010_tariffs_table.sql` - جدول التعرفات مع RLS
11. `011_segmentation_rules_table.sql` - جدول قواعد التقسيم
12. `012_currencies_table.sql` - جدول العملات
13. `013_exchange_rates_table.sql` - جدول أسعار الصرف
14. `014_balance_history_rls.sql` - تحديث جدول سجل الرصيد مع RLS
15. `016_packages_table.sql` - جدول الباقات مع RLS
16. `017_subscriptions_table.sql` - جدول الاشتراكات مع RLS
17. `018_rollover_messages_table.sql` - جدول الرسائل المرحّلة مع RLS
18. `019_billing_invoices_table.sql` - جدول الفواتير مع RLS
19. `020_dedicated_gateways_table.sql` - جدول البوابات المخصصة مع RLS
20. `021_resource_quotas_table.sql` - جدول الحصص الموردية
21. `022_resource_usage_table.sql` - جدول استخدام الموارد (Partitioned)
22. `023_gateway_users_table.sql` - جدول مستخدمي البوابات
23. `024_gateway_api_keys_table.sql` - جدول مفاتيح API
24. `025_gateway_statistics_table.sql` - جدول إحصائيات البوابات
25. `026_promo_codes_table.sql` - جدول أكواد الخصم
26. `027_referrals_table.sql` - جدول الإحالات
27. `028_payment_transactions_table.sql` - جدول معاملات الدفع مع RLS

**للمزيد من التفاصيل حول migrations نظام التعرفات، راجع:**
📄 [DATABASE_SCHEMA_TARIFF_SYSTEM.md](./DATABASE_SCHEMA_TARIFF_SYSTEM.md#migration-files)

---

## مراجع إضافية

- 📄 [DATABASE_SCHEMA_TARIFF_SYSTEM.md](./DATABASE_SCHEMA_TARIFF_SYSTEM.md) - توثيق شامل لنظام التعرفات والتسعير
- 📄 [ARCHITECTURE.md](./ARCHITECTURE.md) - معمارية النظام الكاملة
- 📄 [API_REFERENCE.md](./API_REFERENCE.md) - توثيق APIs
- 📄 [DEVELOPER_GUIDE_TARIFF.md](./DEVELOPER_GUIDE_TARIFF.md) - دليل المطورين لنظام التعرفات

---

**تاريخ الإنشاء**: 2026-02-26  
**الإصدار**: 2.0  
**آخر تحديث**: 2026-03-02  
**التحديثات الرئيسية**:
- إضافة نظام التعرفات والتسعير الشامل (18 جدول جديد)
- إضافة Row Level Security (RLS) للعزل بين الشركات
- إضافة نظام البوابات المخصصة للشركات المؤسسية
- إضافة نظام الدفع والفوترة
- إضافة نظام الخصومات والإحالات
