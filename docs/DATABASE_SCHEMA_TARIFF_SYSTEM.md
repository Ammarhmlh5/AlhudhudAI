# Database Schema - Tariff Pricing System

## نظرة عامة

هذا المستند يوثق جداول نظام التعرفات والتسعير بالتفصيل.

**النظام الأساسي**: docs/DATABASE_SCHEMA.md  
**التاريخ**: 2026-03-02  
**الإصدار**: 1.0

---

## جداول نظام التعرفات والتسعير

### 1. services (الخدمات)

جدول الخدمات الثلاثة المدعومة في النظام.

```sql
CREATE TABLE services (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(50) UNIQUE NOT NULL,
    type VARCHAR(50) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_services_type ON services(type);
CREATE INDEX idx_services_active ON services(is_active);
```

**Columns:**
- `id`: معرف فريد للخدمة
- `name`: اسم الخدمة (عربي)
- `code`: رمز الخدمة (فريد)
- `type`: نوع الخدمة (`gateway`, `mobile`, `dedicated`)
- `description`: وصف الخدمة
- `is_active`: هل الخدمة نشطة
- `created_at`: تاريخ الإنشاء
- `updated_at`: تاريخ آخر تحديث

**Service Types:**
- `gateway`: خدمة بوابة الهدهد (Pay-as-you-go)
- `mobile`: خدمة الإرسال عبر الهاتف (Subscription)
- `dedicated`: خدمة إدارة البوابات المخصصة (Enterprise)

**Initial Data:**
```sql
INSERT INTO services (name, code, type, description) VALUES
('خدمة بوابة الهدهد', 'HUDUD_GATEWAY', 'gateway', 'إرسال الرسائل عبر بوابة الهدهد بنموذج الدفع حسب الاستخدام'),
('خدمة الإرسال عبر الهاتف', 'MOBILE_GATEWAY', 'mobile', 'إرسال الرسائل عبر الهاتف بنموذج الاشتراك الشهري'),
('خدمة إدارة البوابات', 'DEDICATED_GATEWAY', 'dedicated', 'بوابات مخصصة للشركات المؤسسية');
```

---

### 2. tariffs (التعرفات)

جدول التعرفات الديناميكية مع دعم Multi-Tenancy.

```sql
CREATE TABLE tariffs (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    service_id INTEGER NOT NULL REFERENCES services(id),
    country_code VARCHAR(2) NOT NULL,
    network_operator VARCHAR(100),
    message_type VARCHAR(50) DEFAULT 'standard',
    cost_per_message DECIMAL(10, 4) NOT NULL,
    currency VARCHAR(3) DEFAULT 'YER',
    effective_from DATE NOT NULL,
    effective_to DATE,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT check_cost_positive CHECK (cost_per_message > 0)
);

-- Indexes مركبة تبدأ بـ tenant_id للعزل
CREATE INDEX idx_tariffs_tenant_service ON tariffs(tenant_id, service_id);
CREATE INDEX idx_tariffs_tenant_country ON tariffs(tenant_id, country_code);
CREATE INDEX idx_tariffs_tenant_operator ON tariffs(tenant_id, network_operator);
CREATE INDEX idx_tariffs_tenant_active ON tariffs(tenant_id, is_active);
CREATE INDEX idx_tariffs_tenant_effective ON tariffs(tenant_id, effective_from, effective_to);

-- Row Level Security
ALTER TABLE tariffs ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON tariffs
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

**Columns:**
- `id`: معرف فريد للتعرفة
- `tenant_id`: معرف الشركة (للعزل)
- `service_id`: معرف الخدمة
- `country_code`: رمز الدولة (ISO 3166-1)
- `network_operator`: مشغل الشبكة (مثل: Yemen Mobile)
- `message_type`: نوع الرسالة (`standard`, `unicode`)
- `cost_per_message`: تكلفة الرسالة الواحدة
- `currency`: العملة (YER, USD, SAR)
- `effective_from`: تاريخ بداية السريان
- `effective_to`: تاريخ نهاية السريان (nullable)
- `is_active`: هل التعرفة نشطة
- `created_at`: تاريخ الإنشاء
- `updated_at`: تاريخ آخر تحديث

**RLS Policies:**
- `tenant_isolation_policy`: عزل التعرفات بين الشركات

---

### 3. segmentation_rules (قواعد تقسيم الرسائل)

جدول قواعد تقسيم الرسائل الطويلة.

```sql
CREATE TABLE segmentation_rules (
    id SERIAL PRIMARY KEY,
    service_id INTEGER NOT NULL REFERENCES services(id),
    encoding VARCHAR(50) NOT NULL,
    segment_number INTEGER NOT NULL,
    max_length INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(service_id, encoding, segment_number)
);

CREATE INDEX idx_segmentation_service ON segmentation_rules(service_id);
```

**Columns:**
- `id`: معرف فريد للقاعدة
- `service_id`: معرف الخدمة
- `encoding`: نوع الترميز (`GSM-7`, `Unicode`)
- `segment_number`: رقم الجزء (1, 2, 3, ...)
- `max_length`: الحد الأقصى للطول
- `created_at`: تاريخ الإنشاء
- `updated_at`: تاريخ آخر تحديث

**Default Rules:**
```sql
INSERT INTO segmentation_rules (service_id, encoding, segment_number, max_length) VALUES
(1, 'GSM-7', 1, 70),
(1, 'GSM-7', 2, 65),
(1, 'GSM-7', 3, 62),
(1, 'Unicode', 1, 70),
(1, 'Unicode', 2, 65),
(1, 'Unicode', 3, 62);
```

---

### 4. currencies (العملات)

جدول العملات المدعومة.

```sql
CREATE TABLE currencies (
    id SERIAL PRIMARY KEY,
    code VARCHAR(3) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    symbol VARCHAR(10),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_currencies_code ON currencies(code);
```

**Columns:**
- `id`: معرف فريد للعملة
- `code`: رمز العملة (ISO 4217)
- `name`: اسم العملة
- `symbol`: رمز العملة
- `is_active`: هل العملة نشطة
- `created_at`: تاريخ الإنشاء
- `updated_at`: تاريخ آخر تحديث

---

### 5. exchange_rates (أسعار الصرف)

جدول أسعار الصرف بين العملات.

```sql
CREATE TABLE exchange_rates (
    id SERIAL PRIMARY KEY,
    from_currency VARCHAR(3) NOT NULL REFERENCES currencies(code),
    to_currency VARCHAR(3) NOT NULL REFERENCES currencies(code),
    rate DECIMAL(10, 6) NOT NULL,
    effective_from DATE NOT NULL,
    effective_to DATE,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT check_rate_positive CHECK (rate > 0),
    UNIQUE(from_currency, to_currency, effective_from)
);

CREATE INDEX idx_exchange_rates_currencies ON exchange_rates(from_currency, to_currency);
CREATE INDEX idx_exchange_rates_effective ON exchange_rates(effective_from, effective_to);
```

**Columns:**
- `id`: معرف فريد لسعر الصرف
- `from_currency`: العملة المصدر
- `to_currency`: العملة الهدف
- `rate`: سعر الصرف
- `effective_from`: تاريخ بداية السريان
- `effective_to`: تاريخ نهاية السريان (nullable)
- `is_active`: هل سعر الصرف نشط
- `created_at`: تاريخ الإنشاء
- `updated_at`: تاريخ آخر تحديث

---

### 6. packages (الباقات الشهرية)

جدول الباقات الشهرية مع دعم Multi-Tenancy.

```sql
CREATE TABLE packages (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    service_id INTEGER NOT NULL REFERENCES services(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    included_messages INTEGER,
    billing_cycle VARCHAR(50) DEFAULT 'monthly',
    rollover_enabled BOOLEAN DEFAULT false,
    rollover_limit INTEGER,
    rollover_expiry_months INTEGER DEFAULT 1,
    overage_rate DECIMAL(10, 4),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT check_price_positive CHECK (price > 0)
);

CREATE INDEX idx_packages_tenant_service ON packages(tenant_id, service_id);
CREATE INDEX idx_packages_tenant_active ON packages(tenant_id, is_active);

ALTER TABLE packages ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON packages
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

**Columns:**
- `id`: معرف فريد للباقة
- `tenant_id`: معرف الشركة (للعزل)
- `service_id`: معرف الخدمة
- `name`: اسم الباقة
- `description`: وصف الباقة
- `price`: سعر الباقة
- `currency`: العملة
- `included_messages`: عدد الرسائل المضمنة (NULL = unlimited)
- `billing_cycle`: دورة الفوترة (`monthly`, `yearly`)
- `rollover_enabled`: هل الترحيل مفعّل
- `rollover_limit`: الحد الأقصى للترحيل
- `rollover_expiry_months`: مدة صلاحية الرسائل المرحّلة (بالأشهر)
- `overage_rate`: سعر الرسائل الإضافية
- `is_active`: هل الباقة نشطة
- `created_at`: تاريخ الإنشاء
- `updated_at`: تاريخ آخر تحديث

**Default Packages:**
```sql
INSERT INTO packages (name, price, included_messages, overage_rate) VALUES
('Basic', 10.00, 1000, 0.010),
('Standard', 20.00, 3000, 0.007),
('Professional', 40.00, 8000, 0.005),
('Business', 50.00, 15000, 0.003),
('Enterprise', 100.00, NULL, 0.000);
```

---

### 7. subscriptions (الاشتراكات)

جدول اشتراكات العملاء في الباقات.

```sql
CREATE TABLE subscriptions (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    package_id INTEGER NOT NULL REFERENCES packages(id),
    status VARCHAR(50) DEFAULT 'active',
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    next_renewal_date DATE NOT NULL,
    messages_included INTEGER NOT NULL,
    messages_used INTEGER DEFAULT 0,
    messages_remaining INTEGER NOT NULL,
    overage_messages INTEGER DEFAULT 0,
    overage_cost DECIMAL(10, 2) DEFAULT 0.00,
    auto_renew BOOLEAN DEFAULT true,
    grace_period_days INTEGER DEFAULT 3,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT check_messages_valid CHECK (messages_used >= 0 AND messages_remaining >= 0)
);

CREATE INDEX idx_subscriptions_tenant ON subscriptions(tenant_id);
CREATE INDEX idx_subscriptions_tenant_status ON subscriptions(tenant_id, status);
CREATE INDEX idx_subscriptions_tenant_renewal ON subscriptions(tenant_id, next_renewal_date);

ALTER TABLE subscriptions ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON subscriptions
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

**Columns:**
- `id`: معرف فريد للاشتراك
- `tenant_id`: معرف الشركة (للعزل)
- `package_id`: معرف الباقة
- `status`: حالة الاشتراك (`active`, `grace_period`, `suspended`, `cancelled`)
- `start_date`: تاريخ بداية الاشتراك
- `end_date`: تاريخ نهاية الاشتراك
- `next_renewal_date`: تاريخ التجديد القادم
- `messages_included`: عدد الرسائل المضمنة
- `messages_used`: عدد الرسائل المستخدمة
- `messages_remaining`: عدد الرسائل المتبقية
- `overage_messages`: عدد الرسائل الإضافية
- `overage_cost`: تكلفة الرسائل الإضافية
- `auto_renew`: هل التجديد التلقائي مفعّل
- `grace_period_days`: عدد أيام فترة السماح
- `created_at`: تاريخ الإنشاء
- `updated_at`: تاريخ آخر تحديث

**Subscription Statuses:**
- `active`: نشط
- `grace_period`: في فترة السماح
- `suspended`: معلق
- `cancelled`: ملغي

---

### 8. rollover_messages (الرسائل المرحّلة)

جدول الرسائل المرحّلة من شهر لآخر مع دعم سياسة الترحيل المشروط.

```sql
CREATE TABLE rollover_messages (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    subscription_id INTEGER NOT NULL REFERENCES subscriptions(id),
    source_month DATE NOT NULL,
    messages_count INTEGER NOT NULL,
    messages_used INTEGER DEFAULT 0,
    messages_remaining INTEGER NOT NULL,
    expiry_date DATE NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    payment_status VARCHAR(50) DEFAULT 'pending',
    payment_verified_at TIMESTAMP,
    forfeited_reason TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT check_messages_valid CHECK (messages_count > 0 AND messages_remaining >= 0)
);

CREATE INDEX idx_rollover_tenant_subscription ON rollover_messages(tenant_id, subscription_id);
CREATE INDEX idx_rollover_tenant_expiry ON rollover_messages(tenant_id, expiry_date);
CREATE INDEX idx_rollover_status ON rollover_messages(status);

ALTER TABLE rollover_messages ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON rollover_messages
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

**Columns:**
- `id`: معرف فريد للسجل
- `tenant_id`: معرف الشركة (للعزل)
- `subscription_id`: معرف الاشتراك
- `source_month`: الشهر المصدر
- `messages_count`: عدد الرسائل المرحّلة
- `messages_used`: عدد الرسائل المستخدمة
- `messages_remaining`: عدد الرسائل المتبقية
- `expiry_date`: تاريخ انتهاء الصلاحية
- `status`: حالة الترحيل (`pending`, `active`, `expired`, `forfeited`)
- `payment_status`: حالة الدفع (`pending`, `success`, `failed`)
- `payment_verified_at`: تاريخ تأكيد الدفع
- `forfeited_reason`: سبب المصادرة
- `created_at`: تاريخ الإنشاء
- `updated_at`: تاريخ آخر تحديث

**Rollover Statuses:**
- `pending`: في انتظار تأكيد الدفع
- `active`: نشط (تم تأكيد الدفع)
- `expired`: منتهي الصلاحية
- `forfeited`: تمت مصادرته (فشل الدفع)

**سياسة الترحيل المشروط:**
1. عند نهاية الشهر، يتم حفظ الرسائل المتبقية بحالة `pending`
2. عند نجاح الدفع التلقائي، يتم تحديث الحالة إلى `active`
3. عند فشل الدفع التلقائي، يتم تحديث الحالة إلى `forfeited`

---

### 9. billing_invoices (الفواتير)

جدول الفواتير المولدة تلقائياً.

```sql
CREATE TABLE billing_invoices (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    invoice_number VARCHAR(50) UNIQUE NOT NULL,
    invoice_type VARCHAR(50) NOT NULL,
    amount DECIMAL(10, 2) NOT NULL,
    tax_amount DECIMAL(10, 2) DEFAULT 0.00,
    total_amount DECIMAL(10, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    status VARCHAR(50) DEFAULT 'pending',
    due_date DATE NOT NULL,
    paid_at TIMESTAMP,
    pdf_path VARCHAR(500),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT check_amounts_positive CHECK (amount > 0 AND total_amount > 0)
);

CREATE INDEX idx_invoices_tenant ON billing_invoices(tenant_id);
CREATE INDEX idx_invoices_status ON billing_invoices(status);
CREATE INDEX idx_invoices_number ON billing_invoices(invoice_number);

ALTER TABLE billing_invoices ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON billing_invoices
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

**Columns:**
- `id`: معرف فريد للفاتورة
- `tenant_id`: معرف الشركة (للعزل)
- `invoice_number`: رقم الفاتورة (فريد)
- `invoice_type`: نوع الفاتورة (`subscription`, `topup`, `overage`, `gateway_rental`)
- `amount`: المبلغ الأساسي
- `tax_amount`: مبلغ الضريبة
- `total_amount`: المبلغ الإجمالي
- `currency`: العملة
- `status`: حالة الفاتورة (`pending`, `paid`, `overdue`, `cancelled`)
- `due_date`: تاريخ الاستحقاق
- `paid_at`: تاريخ الدفع
- `pdf_path`: مسار ملف PDF
- `created_at`: تاريخ الإنشاء
- `updated_at`: تاريخ آخر تحديث

---

## جداول البوابات المخصصة (Dedicated Gateways)

### 10. dedicated_gateways (البوابات المخصصة)

جدول البوابات المخصصة للشركات المؤسسية.

```sql
CREATE TABLE dedicated_gateways (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    code VARCHAR(50) UNIQUE NOT NULL,
    status VARCHAR(50) DEFAULT 'active',
    telecom_provider VARCHAR(100) NOT NULL,
    telecom_username VARCHAR(255),
    telecom_password_encrypted TEXT,
    telecom_api_key_encrypted TEXT,
    telecom_balance DECIMAL(10, 2),
    balance_alert_threshold DECIMAL(10, 2),
    last_balance_check TIMESTAMP,
    subscription_type VARCHAR(50) DEFAULT 'monthly',
    subscription_fee DECIMAL(10, 2) NOT NULL,
    server_rental_fee DECIMAL(10, 2) NOT NULL,
    contract_start_date DATE NOT NULL,
    contract_end_date DATE,
    max_throughput INTEGER DEFAULT 100,
    daily_message_limit INTEGER,
    monthly_message_limit INTEGER,
    last_health_check TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_dedicated_gateways_tenant ON dedicated_gateways(tenant_id);
CREATE INDEX idx_dedicated_gateways_status ON dedicated_gateways(status);
CREATE INDEX idx_dedicated_gateways_code ON dedicated_gateways(code);

ALTER TABLE dedicated_gateways ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON dedicated_gateways
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

**Columns:**
- `id`: معرف فريد للبوابة
- `tenant_id`: معرف الشركة (للعزل)
- `name`: اسم البوابة
- `code`: رمز البوابة (فريد)
- `status`: حالة البوابة (`active`, `suspended`, `maintenance`, `terminated`)
- `telecom_provider`: مزود خدمة الاتصالات
- `telecom_username`: اسم المستخدم لدى شركة الاتصالات
- `telecom_password_encrypted`: كلمة المرور المشفرة
- `telecom_api_key_encrypted`: مفتاح API المشفر
- `telecom_balance`: رصيد الشركة لدى شركة الاتصالات (للقراءة فقط)
- `balance_alert_threshold`: عتبة تنبيه الرصيد
- `last_balance_check`: آخر فحص للرصيد
- `subscription_type`: نوع الاشتراك (`monthly`, `yearly`)
- `subscription_fee`: رسوم الاشتراك
- `server_rental_fee`: رسوم إيجار السيرفر
- `contract_start_date`: تاريخ بداية العقد
- `contract_end_date`: تاريخ نهاية العقد
- `max_throughput`: الحد الأقصى للإنتاجية (رسائل/ثانية)
- `daily_message_limit`: الحد اليومي للرسائل
- `monthly_message_limit`: الحد الشهري للرسائل
- `last_health_check`: آخر فحص صحة
- `created_at`: تاريخ الإنشاء
- `updated_at`: تاريخ آخر تحديث

**نموذج العمل المالي:**
- الشركة المؤسسية تدفع لشركة الاتصالات مباشرة لتكلفة الرسائل
- المنصة تتقاضى فقط: رسوم الاشتراك + رسوم إيجار السيرفر
- المنصة تراقب رصيد الشركة لدى شركة الاتصالات وترسل تنبيهات

---

### 11. resource_quotas (الحصص الموردية)

جدول الحصص الموردية لكل بوابة مخصصة.

```sql
CREATE TABLE resource_quotas (
    id SERIAL PRIMARY KEY,
    gateway_id INTEGER NOT NULL REFERENCES dedicated_gateways(id) ON DELETE CASCADE,
    cpu_cores DECIMAL(4, 2) NOT NULL,
    memory_mb INTEGER NOT NULL,
    bandwidth_mbps INTEGER NOT NULL,
    storage_gb INTEGER NOT NULL,
    max_db_connections INTEGER NOT NULL,
    redis_memory_mb INTEGER NOT NULL,
    max_messages_per_hour INTEGER NOT NULL,
    max_messages_per_day INTEGER NOT NULL,
    api_rate_limit_per_minute INTEGER NOT NULL,
    max_worker_threads INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(gateway_id)
);

CREATE INDEX idx_resource_quotas_gateway ON resource_quotas(gateway_id);
```

**Columns:**
- `id`: معرف فريد للحصة
- `gateway_id`: معرف البوابة
- `cpu_cores`: عدد أنوية CPU
- `memory_mb`: الذاكرة (MB)
- `bandwidth_mbps`: عرض النطاق (Mbps)
- `storage_gb`: التخزين (GB)
- `max_db_connections`: الحد الأقصى لاتصالات قاعدة البيانات
- `redis_memory_mb`: ذاكرة Redis (MB)
- `max_messages_per_hour`: الحد الأقصى للرسائل في الساعة
- `max_messages_per_day`: الحد الأقصى للرسائل في اليوم
- `api_rate_limit_per_minute`: حد API في الدقيقة
- `max_worker_threads`: الحد الأقصى لـ Worker Threads
- `created_at`: تاريخ الإنشاء
- `updated_at`: تاريخ آخر تحديث

---

### 12. resource_usage (استخدام الموارد)

جدول استخدام الموارد الفعلي (مع Partitioning حسب الشهر).

```sql
CREATE TABLE resource_usage (
    id BIGSERIAL PRIMARY KEY,
    gateway_id INTEGER NOT NULL REFERENCES dedicated_gateways(id) ON DELETE CASCADE,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    cpu_usage DECIMAL(5, 2) NOT NULL,
    memory_usage_mb INTEGER NOT NULL,
    bandwidth_usage_mbps DECIMAL(10, 2) NOT NULL,
    active_db_connections INTEGER NOT NULL,
    redis_memory_mb INTEGER NOT NULL,
    redis_keys_count INTEGER NOT NULL,
    messages_processed_hour INTEGER NOT NULL,
    queue_depth INTEGER NOT NULL,
    api_requests_minute INTEGER NOT NULL,
    active_worker_threads INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) PARTITION BY RANGE (timestamp);

CREATE INDEX idx_resource_usage_gateway_time ON resource_usage(gateway_id, timestamp DESC);

-- إنشاء Partitions شهرية
CREATE TABLE resource_usage_2026_03 PARTITION OF resource_usage
    FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');
```

**Columns:**
- `id`: معرف فريد للسجل
- `gateway_id`: معرف البوابة
- `timestamp`: وقت القياس
- `cpu_usage`: استخدام CPU (%)
- `memory_usage_mb`: استخدام الذاكرة (MB)
- `bandwidth_usage_mbps`: استخدام عرض النطاق (Mbps)
- `active_db_connections`: اتصالات قاعدة البيانات النشطة
- `redis_memory_mb`: استخدام ذاكرة Redis (MB)
- `redis_keys_count`: عدد مفاتيح Redis
- `messages_processed_hour`: الرسائل المعالجة في الساعة الأخيرة
- `queue_depth`: عمق قائمة الانتظار
- `api_requests_minute`: طلبات API في الدقيقة الأخيرة
- `active_worker_threads`: Worker Threads النشطة
- `created_at`: تاريخ الإنشاء

**Retention Policy:**
- الاحتفاظ بالبيانات لمدة 90 يوم
- حذف تلقائي للبيانات الأقدم من 90 يوم

---

### 13. gateway_users (مستخدمو البوابات)

جدول مستخدمي البوابات المخصصة.

```sql
CREATE TABLE gateway_users (
    id SERIAL PRIMARY KEY,
    gateway_id INTEGER NOT NULL REFERENCES dedicated_gateways(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL,
    permissions JSONB DEFAULT '[]',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(gateway_id, user_id)
);

CREATE INDEX idx_gateway_users_gateway ON gateway_users(gateway_id);
CREATE INDEX idx_gateway_users_user ON gateway_users(user_id);
CREATE INDEX idx_gateway_users_role ON gateway_users(role);
```

**Columns:**
- `id`: معرف فريد للسجل
- `gateway_id`: معرف البوابة
- `user_id`: معرف المستخدم
- `role`: الدور (`admin`, `operator`, `viewer`)
- `permissions`: الصلاحيات الدقيقة (JSON)
- `is_active`: هل المستخدم نشط
- `created_at`: تاريخ الإنشاء
- `updated_at`: تاريخ آخر تحديث

**Roles:**
- `admin`: مدير (إدارة كاملة)
- `operator`: مشغّل (إرسال رسائل وعرض تقارير)
- `viewer`: مشاهد (عرض تقارير فقط)

---

### 14. gateway_api_keys (مفاتيح API للبوابات)

جدول مفاتيح API للبوابات المخصصة.

```sql
CREATE TABLE gateway_api_keys (
    id SERIAL PRIMARY KEY,
    gateway_id INTEGER NOT NULL REFERENCES dedicated_gateways(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    key_hash VARCHAR(255) UNIQUE NOT NULL,
    permissions JSONB DEFAULT '[]',
    rate_limit_per_minute INTEGER DEFAULT 60,
    allowed_ips TEXT[],
    expires_at TIMESTAMP,
    is_active BOOLEAN DEFAULT true,
    last_used_at TIMESTAMP,
    usage_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_gateway_api_keys_gateway ON gateway_api_keys(gateway_id);
CREATE INDEX idx_gateway_api_keys_hash ON gateway_api_keys(key_hash);
CREATE INDEX idx_gateway_api_keys_active ON gateway_api_keys(is_active);
```

**Columns:**
- `id`: معرف فريد للمفتاح
- `gateway_id`: معرف البوابة
- `name`: اسم المفتاح
- `description`: وصف المفتاح
- `key_hash`: hash المفتاح (bcrypt)
- `permissions`: الصلاحيات (JSON)
- `rate_limit_per_minute`: حد الطلبات في الدقيقة
- `allowed_ips`: قائمة IPs المسموحة
- `expires_at`: تاريخ انتهاء الصلاحية
- `is_active`: هل المفتاح نشط
- `last_used_at`: آخر استخدام
- `usage_count`: عدد مرات الاستخدام
- `created_at`: تاريخ الإنشاء
- `updated_at`: تاريخ آخر تحديث

---

### 15. gateway_statistics (إحصائيات البوابات)

جدول الإحصائيات اليومية للبوابات المخصصة.

```sql
CREATE TABLE gateway_statistics (
    id SERIAL PRIMARY KEY,
    gateway_id INTEGER NOT NULL REFERENCES dedicated_gateways(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    messages_sent INTEGER DEFAULT 0,
    messages_delivered INTEGER DEFAULT 0,
    messages_failed INTEGER DEFAULT 0,
    messages_pending INTEGER DEFAULT 0,
    avg_response_time_ms INTEGER DEFAULT 0,
    max_response_time_ms INTEGER DEFAULT 0,
    uptime_percentage DECIMAL(5, 2) DEFAULT 100.00,
    total_cost DECIMAL(10, 2) DEFAULT 0.00,
    total_revenue DECIMAL(10, 2) DEFAULT 0.00,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(gateway_id, date)
);

CREATE INDEX idx_gateway_statistics_gateway_date ON gateway_statistics(gateway_id, date DESC);
```

**Columns:**
- `id`: معرف فريد للسجل
- `gateway_id`: معرف البوابة
- `date`: التاريخ
- `messages_sent`: عدد الرسائل المرسلة
- `messages_delivered`: عدد الرسائل المسلّمة
- `messages_failed`: عدد الرسائل الفاشلة
- `messages_pending`: عدد الرسائل المعلقة
- `avg_response_time_ms`: متوسط وقت الاستجابة (ms)
- `max_response_time_ms`: أقصى وقت استجابة (ms)
- `uptime_percentage`: نسبة وقت التشغيل (%)
- `total_cost`: التكلفة الإجمالية
- `total_revenue`: الإيرادات الإجمالية
- `created_at`: تاريخ الإنشاء
- `updated_at`: تاريخ آخر تحديث

---

## جداول الدفع والعروض

### 16. payment_transactions (معاملات الدفع)

جدول معاملات الدفع عبر بوابات الدفع.

```sql
CREATE TABLE payment_transactions (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    transaction_id VARCHAR(255) UNIQUE NOT NULL,
    payment_gateway VARCHAR(50) NOT NULL,
    amount DECIMAL(10, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    status VARCHAR(50) DEFAULT 'pending',
    payment_method VARCHAR(50),
    gateway_transaction_id VARCHAR(255),
    gateway_response JSONB,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_payment_transactions_tenant ON payment_transactions(tenant_id);
CREATE INDEX idx_payment_transactions_status ON payment_transactions(status);
CREATE INDEX idx_payment_transactions_gateway ON payment_transactions(payment_gateway);

ALTER TABLE payment_transactions ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON payment_transactions
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

**Columns:**
- `id`: معرف فريد للمعاملة
- `tenant_id`: معرف الشركة (للعزل)
- `transaction_id`: معرف المعاملة (فريد)
- `payment_gateway`: بوابة الدفع (`stripe`, `paypal`)
- `amount`: المبلغ
- `currency`: العملة
- `status`: حالة المعاملة (`pending`, `completed`, `failed`, `refunded`)
- `payment_method`: طريقة الدفع (`card`, `paypal`, `bank_transfer`)
- `gateway_transaction_id`: معرف المعاملة في بوابة الدفع
- `gateway_response`: استجابة بوابة الدفع (JSON)
- `error_message`: رسالة الخطأ
- `created_at`: تاريخ الإنشاء
- `completed_at`: تاريخ الإكمال
- `updated_at`: تاريخ آخر تحديث

---

### 17. promo_codes (أكواد الخصم)

جدول أكواد الخصم والعروض.

```sql
CREATE TABLE promo_codes (
    id SERIAL PRIMARY KEY,
    tenant_id UUID REFERENCES tenants(id),
    code VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    discount_type VARCHAR(50) NOT NULL,
    discount_value DECIMAL(10, 2) NOT NULL,
    min_purchase_amount DECIMAL(10, 2),
    max_discount_amount DECIMAL(10, 2),
    usage_limit INTEGER,
    usage_count INTEGER DEFAULT 0,
    valid_from DATE NOT NULL,
    valid_to DATE NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT check_discount_positive CHECK (discount_value > 0)
);

CREATE INDEX idx_promo_codes_code ON promo_codes(code);
CREATE INDEX idx_promo_codes_tenant ON promo_codes(tenant_id);
CREATE INDEX idx_promo_codes_active ON promo_codes(is_active);
```

**Columns:**
- `id`: معرف فريد للكود
- `tenant_id`: معرف الشركة (nullable للأكواد العامة)
- `code`: كود الخصم (فريد)
- `description`: وصف العرض
- `discount_type`: نوع الخصم (`percentage`, `fixed_amount`)
- `discount_value`: قيمة الخصم
- `min_purchase_amount`: الحد الأدنى للشراء
- `max_discount_amount`: الحد الأقصى للخصم
- `usage_limit`: الحد الأقصى للاستخدام
- `usage_count`: عدد مرات الاستخدام
- `valid_from`: تاريخ بداية الصلاحية
- `valid_to`: تاريخ نهاية الصلاحية
- `is_active`: هل الكود نشط
- `created_at`: تاريخ الإنشاء
- `updated_at`: تاريخ آخر تحديث

---

### 18. referrals (الإحالات)

جدول نظام الإحالة.

```sql
CREATE TABLE referrals (
    id SERIAL PRIMARY KEY,
    referrer_tenant_id UUID NOT NULL REFERENCES tenants(id),
    referred_tenant_id UUID NOT NULL REFERENCES tenants(id),
    referral_code VARCHAR(50) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    referrer_reward_amount DECIMAL(10, 2) DEFAULT 0.00,
    referred_reward_amount DECIMAL(10, 2) DEFAULT 0.00,
    referrer_reward_granted BOOLEAN DEFAULT false,
    referred_reward_granted BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(referrer_tenant_id, referred_tenant_id)
);

CREATE INDEX idx_referrals_referrer ON referrals(referrer_tenant_id);
CREATE INDEX idx_referrals_referred ON referrals(referred_tenant_id);
CREATE INDEX idx_referrals_code ON referrals(referral_code);
CREATE INDEX idx_referrals_status ON referrals(status);
```

**Columns:**
- `id`: معرف فريد للإحالة
- `referrer_tenant_id`: معرف المُحيل
- `referred_tenant_id`: معرف المُحال
- `referral_code`: كود الإحالة
- `status`: حالة الإحالة (`pending`, `completed`, `cancelled`)
- `referrer_reward_amount`: مكافأة المُحيل
- `referred_reward_amount`: مكافأة المُحال
- `referrer_reward_granted`: هل تم منح مكافأة المُحيل
- `referred_reward_granted`: هل تم منح مكافأة المُحال
- `created_at`: تاريخ الإنشاء
- `completed_at`: تاريخ الإكمال
- `updated_at`: تاريخ آخر تحديث

---

## Row Level Security (RLS) Policies

### نظرة عامة على RLS

Row Level Security (RLS) هو آلية أمان في PostgreSQL تسمح بتطبيق سياسات على مستوى الصف لعزل البيانات بين الشركات (Multi-Tenancy Isolation).

### الجداول المحمية بـ RLS

جميع الجداول التالية محمية بـ RLS:

1. `tariffs`
2. `packages`
3. `subscriptions`
4. `rollover_messages`
5. `dedicated_gateways`
6. `billing_invoices`
7. `payment_transactions`

### سياسات RLS القياسية

لكل جدول محمي، يتم تطبيق السياسات التالية:

#### 1. سياسة القراءة (SELECT)
```sql
CREATE POLICY tenant_isolation_policy ON <table_name>
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

#### 2. سياسة الإدراج (INSERT)
```sql
CREATE POLICY tenant_isolation_insert_policy ON <table_name>
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);
```

#### 3. سياسة التحديث (UPDATE)
```sql
CREATE POLICY tenant_isolation_update_policy ON <table_name>
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

#### 4. سياسة الحذف (DELETE)
```sql
CREATE POLICY tenant_isolation_delete_policy ON <table_name>
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

### كيفية تعيين tenant_id في الجلسة

قبل تنفيذ أي استعلام، يجب تعيين `app.current_tenant`:

```sql
SET app.current_tenant = '123e4567-e89b-12d3-a456-426614174000';
```

أو في Go:

```go
db.Exec("SET app.current_tenant = $1", tenantID.String())
```

### فوائد RLS

1. **عزل تام**: لا يمكن لشركة الوصول لبيانات شركة أخرى
2. **أمان على مستوى قاعدة البيانات**: الحماية في قاعدة البيانات نفسها
3. **منع SQL Injection**: حتى مع SQL Injection، لا يمكن الوصول لبيانات شركة أخرى
4. **تدقيق سهل**: جميع الوصولات مسجلة

---

## Indexes Strategy

### مبادئ Indexing

1. **جميع Indexes تبدأ بـ tenant_id**: لتحسين أداء الاستعلامات المعزولة
2. **Composite Indexes**: استخدام indexes مركبة للاستعلامات الشائعة
3. **Covering Indexes**: تضمين الأعمدة المستخدمة في SELECT

### Indexes الرئيسية

#### Tariffs Table
```sql
CREATE INDEX idx_tariffs_tenant_service ON tariffs(tenant_id, service_id);
CREATE INDEX idx_tariffs_tenant_country ON tariffs(tenant_id, country_code);
CREATE INDEX idx_tariffs_tenant_operator ON tariffs(tenant_id, network_operator);
CREATE INDEX idx_tariffs_tenant_active ON tariffs(tenant_id, is_active);
CREATE INDEX idx_tariffs_tenant_effective ON tariffs(tenant_id, effective_from, effective_to);
```

#### Subscriptions Table
```sql
CREATE INDEX idx_subscriptions_tenant ON subscriptions(tenant_id);
CREATE INDEX idx_subscriptions_tenant_status ON subscriptions(tenant_id, status);
CREATE INDEX idx_subscriptions_tenant_renewal ON subscriptions(tenant_id, next_renewal_date);
```

#### Dedicated Gateways Table
```sql
CREATE INDEX idx_dedicated_gateways_tenant ON dedicated_gateways(tenant_id);
CREATE INDEX idx_dedicated_gateways_status ON dedicated_gateways(status);
CREATE INDEX idx_dedicated_gateways_code ON dedicated_gateways(code);
```

#### Resource Usage Table (Partitioned)
```sql
CREATE INDEX idx_resource_usage_gateway_time ON resource_usage(gateway_id, timestamp DESC);
```

### Index Maintenance

```sql
-- إعادة بناء Indexes دورياً
REINDEX TABLE tariffs;
REINDEX TABLE subscriptions;

-- تحليل الجداول لتحديث الإحصائيات
ANALYZE tariffs;
ANALYZE subscriptions;
```

---

## العلاقات بين الجداول (Relationships)

### علاقات نظام التعرفات

```
tenants (1) ──→ (N) tariffs
tenants (1) ──→ (N) packages
tenants (1) ──→ (N) subscriptions
subscriptions (1) ──→ (N) rollover_messages
services (1) ──→ (N) tariffs
services (1) ──→ (N) packages
```

### علاقات البوابات المخصصة

```
tenants (1) ──→ (N) dedicated_gateways
dedicated_gateways (1) ──→ (1) resource_quotas
dedicated_gateways (1) ──→ (N) resource_usage
dedicated_gateways (1) ──→ (N) gateway_users
dedicated_gateways (1) ──→ (N) gateway_api_keys
dedicated_gateways (1) ──→ (N) gateway_statistics
```

### علاقات الدفع والعروض

```
tenants (1) ──→ (N) payment_transactions
tenants (1) ──→ (N) billing_invoices
tenants (1) ──→ (N) promo_codes
tenants (1) ──→ (N) referrals (as referrer)
tenants (1) ──→ (N) referrals (as referred)
```

---

## Migration Files

### ترتيب تنفيذ Migrations

يجب تنفيذ migrations بالترتيب التالي:

#### نظام التعرفات الأساسي
1. `008_tariff_services_table.sql` - جدول الخدمات
2. `009_insert_default_services.sql` - البيانات الأولية للخدمات
3. `010_tariffs_table.sql` - جدول التعرفات مع RLS
4. `011_segmentation_rules_table.sql` - جدول قواعد التقسيم
5. `012_currencies_table.sql` - جدول العملات
6. `013_exchange_rates_table.sql` - جدول أسعار الصرف
7. `014_balance_history_rls.sql` - تحديث جدول سجل الرصيد مع RLS

#### نظام الباقات والاشتراكات
8. `016_packages_table.sql` - جدول الباقات مع RLS
9. `017_subscriptions_table.sql` - جدول الاشتراكات مع RLS
10. `018_rollover_messages_table.sql` - جدول الرسائل المرحّلة مع RLS
11. `019_billing_invoices_table.sql` - جدول الفواتير مع RLS

#### نظام البوابات المخصصة
12. `020_dedicated_gateways_table.sql` - جدول البوابات المخصصة مع RLS
13. `021_resource_quotas_table.sql` - جدول الحصص الموردية
14. `022_resource_usage_table.sql` - جدول استخدام الموارد (Partitioned)
15. `023_gateway_users_table.sql` - جدول مستخدمي البوابات
16. `024_gateway_api_keys_table.sql` - جدول مفاتيح API
17. `025_gateway_statistics_table.sql` - جدول إحصائيات البوابات

#### نظام الدفع والعروض
18. `026_promo_codes_table.sql` - جدول أكواد الخصم
19. `027_referrals_table.sql` - جدول الإحالات
20. `028_payment_transactions_table.sql` - جدول معاملات الدفع مع RLS

### تشغيل Migrations

#### باستخدام Go Migrate
```bash
cd services/backend
go run cmd/migrate/main.go
```

#### باستخدام PowerShell Scripts
```powershell
# تشغيل migration محدد
.\run-tariff-migration.ps1

# تشغيل جميع migrations
.\run-all-migrations.ps1
```

#### باستخدام psql مباشرة
```bash
psql -h localhost -U postgres -d sms_gateway -f migrations/008_tariff_services_table.sql
```

---

## Data Retention Policies

### سياسات الاحتفاظ بالبيانات

| Table | Retention Period | Action | Notes |
|-------|------------------|--------|-------|
| tariffs | Permanent | Archive old versions | الاحتفاظ بالسجل التاريخي |
| packages | Permanent | Archive inactive | الاحتفاظ بالباقات القديمة |
| subscriptions | 2 years | Archive to cold storage | بعد انتهاء الاشتراك |
| rollover_messages | 6 months | Delete | بعد انتهاء الصلاحية |
| billing_invoices | 7 years | Archive to cold storage | متطلبات قانونية |
| payment_transactions | 7 years | Archive to cold storage | متطلبات قانونية |
| resource_usage | 90 days | Delete | بيانات مراقبة |
| gateway_statistics | 2 years | Archive to cold storage | للتحليل |
| promo_codes | 1 year | Archive inactive | بعد انتهاء الصلاحية |
| referrals | Permanent | Keep all | للتدقيق |

### تنفيذ سياسات الاحتفاظ

```sql
-- حذف resource_usage الأقدم من 90 يوم
DELETE FROM resource_usage 
WHERE timestamp < CURRENT_DATE - INTERVAL '90 days';

-- أرشفة rollover_messages المنتهية
INSERT INTO rollover_messages_archive 
SELECT * FROM rollover_messages 
WHERE expiry_date < CURRENT_DATE - INTERVAL '6 months';

DELETE FROM rollover_messages 
WHERE expiry_date < CURRENT_DATE - INTERVAL '6 months';
```

---

## Backup and Recovery

### استراتيجية النسخ الاحتياطي

#### 1. النسخ الاحتياطي اليومي
```bash
# نسخ احتياطي كامل
pg_dump -h localhost -U postgres -d sms_gateway \
  --format=custom \
  --file=backup_tariff_$(date +%Y%m%d).dump

# نسخ احتياطي للجداول الحرجة فقط
pg_dump -h localhost -U postgres -d sms_gateway \
  --table=tariffs \
  --table=packages \
  --table=subscriptions \
  --table=dedicated_gateways \
  --format=custom \
  --file=backup_tariff_critical_$(date +%Y%m%d).dump
```

#### 2. النسخ الاحتياطي الأسبوعي
```bash
# نسخ احتياطي مع ضغط
pg_dump -h localhost -U postgres -d sms_gateway \
  --format=custom \
  --compress=9 \
  --file=backup_tariff_weekly_$(date +%Y%m%d).dump
```

#### 3. Point-in-Time Recovery (PITR)
```bash
# تفعيل WAL archiving
archive_mode = on
archive_command = 'cp %p /backup/wal/%f'
wal_level = replica
```

### الاسترجاع (Recovery)

#### استرجاع كامل
```bash
pg_restore -h localhost -U postgres -d sms_gateway \
  --clean \
  --if-exists \
  backup_tariff_20260302.dump
```

#### استرجاع جداول محددة
```bash
pg_restore -h localhost -U postgres -d sms_gateway \
  --table=tariffs \
  --table=packages \
  backup_tariff_20260302.dump
```

#### استرجاع Point-in-Time
```bash
# في postgresql.conf
restore_command = 'cp /backup/wal/%f %p'
recovery_target_time = '2026-03-02 14:30:00'
```

---

## Performance Optimization

### Query Optimization Tips

#### 1. استخدام Prepared Statements
```go
stmt, err := db.Prepare(`
    SELECT * FROM tariffs 
    WHERE tenant_id = $1 AND service_id = $2 AND is_active = true
`)
defer stmt.Close()

rows, err := stmt.Query(tenantID, serviceID)
```

#### 2. استخدام EXPLAIN ANALYZE
```sql
EXPLAIN ANALYZE
SELECT * FROM subscriptions 
WHERE tenant_id = '123e4567-e89b-12d3-a456-426614174000' 
  AND status = 'active';
```

#### 3. Batch Operations
```go
// بدلاً من إدراج صف واحد في كل مرة
for _, msg := range messages {
    db.Exec("INSERT INTO rollover_messages ...")
}

// استخدم Batch Insert
stmt := `INSERT INTO rollover_messages (tenant_id, subscription_id, ...) VALUES `
values := []interface{}{}
for i, msg := range messages {
    stmt += fmt.Sprintf("($%d, $%d, ...),", i*5+1, i*5+2)
    values = append(values, msg.TenantID, msg.SubscriptionID, ...)
}
stmt = strings.TrimSuffix(stmt, ",")
db.Exec(stmt, values...)
```

#### 4. Connection Pooling
```go
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

---

## Monitoring and Maintenance

### مراقبة الأداء

#### 1. مراقبة حجم الجداول
```sql
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
  AND tablename IN ('tariffs', 'packages', 'subscriptions', 'dedicated_gateways')
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

#### 2. مراقبة Indexes
```sql
SELECT 
    schemaname,
    tablename,
    indexname,
    pg_size_pretty(pg_relation_size(indexrelid)) AS index_size
FROM pg_indexes
JOIN pg_class ON pg_class.relname = indexname
WHERE schemaname = 'public'
ORDER BY pg_relation_size(indexrelid) DESC;
```

#### 3. مراقبة الاستعلامات البطيئة
```sql
-- تفعيل slow query log
ALTER DATABASE sms_gateway SET log_min_duration_statement = 1000;

-- عرض الاستعلامات النشطة
SELECT 
    pid,
    now() - query_start AS duration,
    query
FROM pg_stat_activity
WHERE state = 'active'
  AND now() - query_start > interval '5 seconds'
ORDER BY duration DESC;
```

#### 4. مراقبة RLS Performance
```sql
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size,
    n_tup_ins AS inserts,
    n_tup_upd AS updates,
    n_tup_del AS deletes
FROM pg_stat_user_tables
WHERE tablename IN ('tariffs', 'packages', 'subscriptions')
ORDER BY n_tup_ins + n_tup_upd + n_tup_del DESC;
```

### صيانة دورية

#### 1. VACUUM و ANALYZE
```sql
-- يومياً
VACUUM ANALYZE tariffs;
VACUUM ANALYZE subscriptions;
VACUUM ANALYZE rollover_messages;

-- أسبوعياً
VACUUM FULL ANALYZE dedicated_gateways;
VACUUM FULL ANALYZE resource_usage;
```

#### 2. REINDEX
```sql
-- شهرياً
REINDEX TABLE tariffs;
REINDEX TABLE subscriptions;
REINDEX TABLE dedicated_gateways;
```

#### 3. تحديث الإحصائيات
```sql
ANALYZE tariffs;
ANALYZE subscriptions;
ANALYZE packages;
```

---

## Security Best Practices

### 1. تشفير البيانات الحساسة

```go
// تشفير بيانات الاعتماد
func EncryptCredentials(plaintext string) (string, error) {
    key := []byte(os.Getenv("ENCRYPTION_KEY"))
    block, err := aes.NewCipher(key)
    if err != nil {
        return "", err
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    
    nonce := make([]byte, gcm.NonceSize())
    if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
        return "", err
    }
    
    ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}
```

### 2. Hash API Keys

```go
// استخدام bcrypt لـ hash API keys
func HashAPIKey(key string) (string, error) {
    hash, err := bcrypt.GenerateFromPassword([]byte(key), bcrypt.DefaultCost)
    if err != nil {
        return "", err
    }
    return string(hash), nil
}

func VerifyAPIKey(key, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(key))
    return err == nil
}
```

### 3. Audit Logging

```sql
-- تسجيل جميع العمليات الحساسة
CREATE TABLE security_audit_log (
    id SERIAL PRIMARY KEY,
    tenant_id UUID,
    user_id INTEGER,
    action VARCHAR(255) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    resource_id VARCHAR(255),
    ip_address VARCHAR(45),
    user_agent TEXT,
    success BOOLEAN NOT NULL,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_security_audit_tenant ON security_audit_log(tenant_id, created_at DESC);
CREATE INDEX idx_security_audit_action ON security_audit_log(action);
```

---

## Troubleshooting

### مشاكل شائعة وحلولها

#### 1. RLS Policy لا يعمل

**المشكلة**: البيانات تظهر من جميع الشركات

**الحل**:
```sql
-- التحقق من تفعيل RLS
SELECT tablename, rowsecurity 
FROM pg_tables 
WHERE schemaname = 'public' 
  AND tablename = 'tariffs';

-- التحقق من Policies
SELECT * FROM pg_policies WHERE tablename = 'tariffs';

-- التحقق من tenant_id في الجلسة
SHOW app.current_tenant;
```

#### 2. Indexes لا تُستخدم

**المشكلة**: الاستعلامات بطيئة رغم وجود Indexes

**الحل**:
```sql
-- التحقق من استخدام Index
EXPLAIN ANALYZE
SELECT * FROM tariffs 
WHERE tenant_id = '123e4567-e89b-12d3-a456-426614174000';

-- إعادة بناء Index
REINDEX INDEX idx_tariffs_tenant_service;

-- تحديث الإحصائيات
ANALYZE tariffs;
```

#### 3. Partitioning لا يعمل

**المشكلة**: البيانات لا تُدرج في Partitions الصحيحة

**الحل**:
```sql
-- التحقق من Partitions الموجودة
SELECT 
    parent.relname AS parent_table,
    child.relname AS partition_name,
    pg_get_expr(child.relpartbound, child.oid) AS partition_expression
FROM pg_inherits
JOIN pg_class parent ON pg_inherits.inhparent = parent.oid
JOIN pg_class child ON pg_inherits.inhrelid = child.oid
WHERE parent.relname = 'resource_usage';

-- إنشاء Partition جديد
CREATE TABLE resource_usage_2026_04 PARTITION OF resource_usage
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');
```

---

**تاريخ الإنشاء**: 2026-03-02  
**الإصدار**: 1.0  
**آخر تحديث**: 2026-03-02  
**المؤلف**: Tariff Pricing System Team

