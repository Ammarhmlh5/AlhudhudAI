# تقرير تحليل النقص في جداول الإدارة

**التاريخ:** 2026-03-21  
**النظام:** al_hudhud SMS Platform  
**التركيز:** Admin & Management Tables

---

## ملخص تنفيذي

تم إجراء فحص شامل لجداول الإدارة والصلاحيات في النظام. تم اكتشاف **15 نقص حرج** في البنية الإدارية.

### النتيجة الإجمالية: **⚠️ نقص كبير في جداول الإدارة**

**الجداول الموجودة:**
- ✅ users (المستخدمين)
- ✅ tenants (العملاء)
- ✅ roles (الأدوار)
- ✅ permissions (الصلاحيات)
- ✅ staff_profiles (الموظفين)
- ✅ audit_trail (سجل التدقيق)
- ✅ staff_activity (نشاط الموظفين)

**الجداول المفقودة:** 15 جدول

---

## 1. الجداول المفقودة الحرجة 🔴

### 1.1 جدول role_permissions (ربط الأدوار بالصلاحيات)

#### المشكلة:
**الحالة:** ❌ غير موجود

```go
// في models.go موجود:
type Role struct {
    Permissions []Permission `gorm:"many2many:role_permissions;"`
}

// لكن الجدول غير موجود في migrations!
```

**التأثير:**
- لا يمكن ربط الأدوار بالصلاحيات
- نظام الصلاحيات غير فعال
- **RBAC معطل تماماً** ❌

**الحل المقترح:**
```sql
CREATE TABLE IF NOT EXISTS role_permissions (
    id BIGSERIAL PRIMARY KEY,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(role_id, permission_id)
);

CREATE INDEX idx_role_permissions_role ON role_permissions(role_id);
CREATE INDEX idx_role_permissions_permission ON role_permissions(permission_id);
```

**الأولوية:** 🔴 حرجة جداً

---

### 1.2 جدول user_roles (ربط المستخدمين بالأدوار)

#### المشكلة:
**الحالة:** ❌ غير موجود

**التأثير:**
- لا يمكن تعيين أدوار للمستخدمين
- كل مستخدم يحتاج دور واحد على الأقل
- **لا يمكن إدارة الصلاحيات** ❌

**الحل المقترح:**
```sql
CREATE TABLE IF NOT EXISTS user_roles (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    assigned_by UUID REFERENCES users(id) ON DELETE SET NULL,
    assigned_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(user_id, role_id, tenant_id)
);

CREATE INDEX idx_user_roles_user ON user_roles(user_id);
CREATE INDEX idx_user_roles_role ON user_roles(role_id);
CREATE INDEX idx_user_roles_tenant ON user_roles(tenant_id);
CREATE INDEX idx_user_roles_active ON user_roles(is_active, expires_at);
```

**الأولوية:** 🔴 حرجة جداً


---

### 1.3 جدول admin_settings (إعدادات النظام)

#### المشكلة:
**الحالة:** ❌ غير موجود

**التأثير:**
- لا يمكن تخزين إعدادات النظام العامة
- كل إعداد يحتاج hardcoded في الكود
- **صعوبة في التكوين** ❌

**الحل المقترح:**
```sql
CREATE TABLE IF NOT EXISTS admin_settings (
    id BIGSERIAL PRIMARY KEY,
    setting_key VARCHAR(255) NOT NULL UNIQUE,
    setting_value TEXT,
    value_type VARCHAR(50) DEFAULT 'STRING', -- STRING, INTEGER, BOOLEAN, JSON
    category VARCHAR(100), -- SYSTEM, BILLING, MESSAGING, SECURITY
    description TEXT,
    is_public BOOLEAN DEFAULT false, -- Can be accessed by tenants
    is_encrypted BOOLEAN DEFAULT false,
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_admin_settings_category ON admin_settings(category);
CREATE INDEX idx_admin_settings_public ON admin_settings(is_public);
```

**أمثلة الإعدادات:**
- `system.maintenance_mode` = false
- `billing.default_currency` = USD
- `messaging.max_retry_count` = 3
- `security.session_timeout` = 3600

**الأولوية:** 🔴 حرجة

---

### 1.4 جدول tenant_settings (إعدادات العملاء)

#### المشكلة:
**الحالة:** ❌ غير موجود

**التأثير:**
- لا يمكن تخصيص إعدادات لكل عميل
- كل العملاء لهم نفس الإعدادات
- **لا مرونة في التخصيص** ❌

**الحل المقترح:**
```sql
CREATE TABLE IF NOT EXISTS tenant_settings (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    setting_key VARCHAR(255) NOT NULL,
    setting_value TEXT,
    value_type VARCHAR(50) DEFAULT 'STRING',
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(tenant_id, setting_key)
);

CREATE INDEX idx_tenant_settings_tenant ON tenant_settings(tenant_id);
CREATE INDEX idx_tenant_settings_key ON tenant_settings(setting_key);
```

**أمثلة الإعدادات:**
- `messaging.sender_name` = "شركة ABC"
- `notifications.email_enabled` = true
- `billing.auto_recharge` = false
- `limits.daily_message_limit` = 10000

**الأولوية:** 🔴 حرجة

---

### 1.5 جدول notifications (الإشعارات)

#### المشكلة:
**الحالة:** ❌ غير موجود

**التأثير:**
- لا يمكن إرسال إشعارات للمستخدمين
- لا يوجد inbox للإشعارات
- **تجربة مستخدم ضعيفة** ❌

**الحل المقترح:**
```sql
CREATE TABLE IF NOT EXISTS notifications (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL, -- INFO, WARNING, ERROR, SUCCESS
    category VARCHAR(100), -- BALANCE, MESSAGE, SYSTEM, BILLING
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    action_url VARCHAR(500),
    action_label VARCHAR(100),
    is_read BOOLEAN DEFAULT false,
    read_at TIMESTAMP,
    priority INTEGER DEFAULT 0, -- 0=Normal, 1=High, 2=Urgent
    expires_at TIMESTAMP,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_notifications_user ON notifications(user_id, is_read, created_at DESC);
CREATE INDEX idx_notifications_tenant ON notifications(tenant_id, created_at DESC);
CREATE INDEX idx_notifications_unread ON notifications(user_id, is_read) WHERE is_read = false;
CREATE INDEX idx_notifications_priority ON notifications(priority DESC, created_at DESC);
```

**الأولوية:** 🔴 حرجة


---

### 1.6 جدول email_templates (قوالب البريد الإلكتروني)

#### المشكلة:
**الحالة:** ❌ غير موجود

**التأثير:**
- لا يمكن إدارة قوالب البريد
- كل email hardcoded في الكود
- **صعوبة في التخصيص** ❌

**الحل المقترح:**
```sql
CREATE TABLE IF NOT EXISTS email_templates (
    id BIGSERIAL PRIMARY KEY,
    template_key VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    subject VARCHAR(500) NOT NULL,
    body_html TEXT NOT NULL,
    body_text TEXT,
    category VARCHAR(100), -- AUTHENTICATION, BILLING, NOTIFICATIONS
    variables JSONB, -- Available variables for template
    is_active BOOLEAN DEFAULT true,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_email_templates_category ON email_templates(category);
CREATE INDEX idx_email_templates_active ON email_templates(is_active);
```

**أمثلة القوالب:**
- `welcome_email` - رسالة الترحيب
- `password_reset` - إعادة تعيين كلمة المرور
- `low_balance_warning` - تحذير انخفاض الرصيد
- `invoice_generated` - فاتورة جديدة

**الأولوية:** 🟠 عالية

---

### 1.7 جدول system_logs (سجلات النظام)

#### المشكلة:
**الحالة:** ❌ غير موجود (يوجد audit_trail فقط)

**التأثير:**
- لا يمكن تتبع أخطاء النظام
- صعوبة في debugging
- **لا يوجد centralized logging** ❌

**الحل المقترح:**
```sql
CREATE TABLE IF NOT EXISTS system_logs (
    id BIGSERIAL PRIMARY KEY,
    level VARCHAR(20) NOT NULL, -- DEBUG, INFO, WARNING, ERROR, CRITICAL
    source VARCHAR(255) NOT NULL, -- Service/Component name
    message TEXT NOT NULL,
    error_code VARCHAR(50),
    stack_trace TEXT,
    context JSONB, -- Additional context
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL,
    ip_address INET,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_system_logs_level ON system_logs(level, created_at DESC);
CREATE INDEX idx_system_logs_source ON system_logs(source, created_at DESC);
CREATE INDEX idx_system_logs_tenant ON system_logs(tenant_id, created_at DESC);
CREATE INDEX idx_system_logs_created ON system_logs(created_at DESC);

-- Partition by month for better performance
CREATE TABLE system_logs_2026_03 PARTITION OF system_logs
FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');
```

**الأولوية:** 🟠 عالية

---

### 1.8 جدول scheduled_tasks (المهام المجدولة)

#### المشكلة:
**الحالة:** ❌ غير موجود

**التأثير:**
- لا يمكن جدولة مهام دورية
- كل cron job يحتاج external scheduler
- **لا يوجد task management** ❌

**الحل المقترح:**
```sql
CREATE TABLE IF NOT EXISTS scheduled_tasks (
    id BIGSERIAL PRIMARY KEY,
    task_name VARCHAR(255) NOT NULL UNIQUE,
    task_type VARCHAR(100) NOT NULL, -- CLEANUP, REPORT, NOTIFICATION, BILLING
    schedule_expression VARCHAR(255) NOT NULL, -- Cron expression
    handler_function VARCHAR(255) NOT NULL,
    parameters JSONB,
    is_active BOOLEAN DEFAULT true,
    last_run_at TIMESTAMP,
    last_run_status VARCHAR(50), -- SUCCESS, FAILURE, RUNNING
    last_run_duration_ms INTEGER,
    last_error TEXT,
    next_run_at TIMESTAMP,
    run_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_scheduled_tasks_active ON scheduled_tasks(is_active, next_run_at);
CREATE INDEX idx_scheduled_tasks_type ON scheduled_tasks(task_type);
```

**أمثلة المهام:**
- `cleanup_old_messages` - حذف الرسائل القديمة
- `generate_daily_reports` - تقارير يومية
- `check_low_balances` - فحص الأرصدة المنخفضة
- `renew_subscriptions` - تجديد الاشتراكات

**الأولوية:** 🟠 عالية

---

### 1.9 جدول api_rate_limits (حدود API)

#### المشكلة:**
**الحالة:** ❌ غير موجود

**التأثير:**
- لا يمكن تخصيص rate limits لكل tenant
- كل العملاء لهم نفس الحدود
- **لا مرونة في API limits** ❌

**الحل المقترح:**
```sql
CREATE TABLE IF NOT EXISTS api_rate_limits (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    endpoint_pattern VARCHAR(255) NOT NULL,
    limit_per_minute INTEGER NOT NULL DEFAULT 60,
    limit_per_hour INTEGER NOT NULL DEFAULT 1000,
    limit_per_day INTEGER NOT NULL DEFAULT 10000,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(tenant_id, endpoint_pattern)
);

CREATE INDEX idx_api_rate_limits_tenant ON api_rate_limits(tenant_id);
CREATE INDEX idx_api_rate_limits_active ON api_rate_limits(is_active);
```

**الأولوية:** 🟡 متوسطة


---

### 1.10 جدول feature_flags (تفعيل الميزات)

#### المشكلة:
**الحالة:** ❌ غير موجود

**التأثير:**
- لا يمكن تفعيل/تعطيل ميزات بدون deployment
- صعوبة في A/B testing
- **لا يوجد feature toggles** ❌

**الحل المقترح:**
```sql
CREATE TABLE IF NOT EXISTS feature_flags (
    id BIGSERIAL PRIMARY KEY,
    flag_key VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_enabled BOOLEAN DEFAULT false,
    rollout_percentage INTEGER DEFAULT 0, -- 0-100 for gradual rollout
    target_tenants UUID[], -- Specific tenants
    target_user_types VARCHAR(50)[], -- SUPER_ADMIN, TENANT, STAFF
    conditions JSONB, -- Complex conditions
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_feature_flags_enabled ON feature_flags(is_enabled);
CREATE INDEX idx_feature_flags_rollout ON feature_flags(rollout_percentage);
```

**أمثلة الميزات:**
- `new_dashboard_ui` - واجهة جديدة
- `advanced_analytics` - تحليلات متقدمة
- `bulk_messaging_v2` - إرسال جماعي محسّن
- `ai_message_optimization` - تحسين الرسائل بالذكاء الاصطناعي

**الأولوية:** 🟡 متوسطة

---

### 1.11 جدول admin_announcements (الإعلانات الإدارية)

#### المشكلة:
**الحالة:** ❌ غير موجود

**التأثير:**
- لا يمكن إرسال إعلانات للمستخدمين
- صعوبة في التواصل مع العملاء
- **لا يوجد broadcast messaging** ❌

**الحل المقترح:**
```sql
CREATE TABLE IF NOT EXISTS admin_announcements (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    type VARCHAR(50) NOT NULL, -- MAINTENANCE, UPDATE, PROMOTION, WARNING
    priority VARCHAR(20) DEFAULT 'NORMAL', -- LOW, NORMAL, HIGH, URGENT
    target_audience VARCHAR(50) DEFAULT 'ALL', -- ALL, TENANTS, STAFF, SPECIFIC
    target_tenant_ids UUID[],
    start_date TIMESTAMP NOT NULL,
    end_date TIMESTAMP,
    is_dismissible BOOLEAN DEFAULT true,
    is_active BOOLEAN DEFAULT true,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_announcements_active ON admin_announcements(is_active, start_date, end_date);
CREATE INDEX idx_announcements_type ON admin_announcements(type);
CREATE INDEX idx_announcements_priority ON admin_announcements(priority);
```

**الأولوية:** 🟡 متوسطة

---

### 1.12 جدول backup_logs (سجلات النسخ الاحتياطي)

#### المشكلة:
**الحالة:** ❌ غير موجود

**التأثير:**
- لا يمكن تتبع النسخ الاحتياطية
- صعوبة في disaster recovery
- **لا يوجد backup tracking** ❌

**الحل المقترح:**
```sql
CREATE TABLE IF NOT EXISTS backup_logs (
    id BIGSERIAL PRIMARY KEY,
    backup_type VARCHAR(50) NOT NULL, -- FULL, INCREMENTAL, DIFFERENTIAL
    backup_target VARCHAR(100) NOT NULL, -- DATABASE, FILES, REDIS
    backup_location VARCHAR(500) NOT NULL,
    backup_size_bytes BIGINT,
    status VARCHAR(50) NOT NULL, -- STARTED, COMPLETED, FAILED
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    duration_seconds INTEGER,
    error_message TEXT,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_backup_logs_type ON backup_logs(backup_type, created_at DESC);
CREATE INDEX idx_backup_logs_status ON backup_logs(status, created_at DESC);
CREATE INDEX idx_backup_logs_target ON backup_logs(backup_target, created_at DESC);
```

**الأولوية:** 🟡 متوسطة

---

### 1.13 جدول tenant_quotas (حصص العملاء)

#### المشكلة:
**الحالة:** ❌ غير موجود (يوجد resource_quotas لكن غير كافٍ)

**التأثير:**
- لا يمكن تحديد حصص مفصلة لكل عميل
- صعوبة في fair usage policy
- **لا يوجد quota management** ❌

**الحل المقترح:**
```sql
CREATE TABLE IF NOT EXISTS tenant_quotas (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    quota_type VARCHAR(100) NOT NULL, -- MESSAGES, STORAGE, API_CALLS, USERS
    quota_limit BIGINT NOT NULL,
    quota_used BIGINT DEFAULT 0,
    quota_period VARCHAR(50) DEFAULT 'MONTHLY', -- DAILY, WEEKLY, MONTHLY, YEARLY
    reset_at TIMESTAMP,
    is_hard_limit BOOLEAN DEFAULT true, -- Hard limit vs soft limit
    warning_threshold INTEGER DEFAULT 80, -- Percentage
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(tenant_id, quota_type, quota_period)
);

CREATE INDEX idx_tenant_quotas_tenant ON tenant_quotas(tenant_id);
CREATE INDEX idx_tenant_quotas_type ON tenant_quotas(quota_type);
CREATE INDEX idx_tenant_quotas_exceeded ON tenant_quotas(tenant_id) 
WHERE quota_used >= quota_limit;
```

**الأولوية:** 🟠 عالية

---

### 1.14 جدول admin_dashboard_widgets (ودجات لوحة التحكم)

#### المشكلة:
**الحالة:** ❌ غير موجود

**التأثير:**
- لا يمكن تخصيص لوحة التحكم
- كل المستخدمين يرون نفس الودجات
- **لا مرونة في Dashboard** ❌

**الحل المقترح:**
```sql
CREATE TABLE IF NOT EXISTS admin_dashboard_widgets (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    widget_type VARCHAR(100) NOT NULL, -- CHART, TABLE, METRIC, ALERT
    widget_config JSONB NOT NULL,
    position_x INTEGER DEFAULT 0,
    position_y INTEGER DEFAULT 0,
    width INTEGER DEFAULT 4,
    height INTEGER DEFAULT 3,
    is_visible BOOLEAN DEFAULT true,
    refresh_interval_seconds INTEGER DEFAULT 60,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_dashboard_widgets_user ON admin_dashboard_widgets(user_id, position_y, position_x);
CREATE INDEX idx_dashboard_widgets_visible ON admin_dashboard_widgets(user_id, is_visible);
```

**الأولوية:** 🟢 منخفضة

---

### 1.15 جدول tenant_billing_history (تاريخ الفواتير)

#### المشكلة:
**الحالة:** ⚠️ يوجد billing_invoices لكن ناقص

**التأثير:**
- لا يوجد تتبع لتاريخ الدفعات
- صعوبة في financial reporting
- **ناقص payment history** ⚠️

**الحل المقترح:**
```sql
CREATE TABLE IF NOT EXISTS tenant_payment_history (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    invoice_id UUID REFERENCES billing_invoices(id) ON DELETE SET NULL,
    payment_method VARCHAR(50) NOT NULL, -- CREDIT_CARD, BANK_TRANSFER, WALLET
    payment_gateway VARCHAR(100), -- STRIPE, PAYPAL, LOCAL_BANK
    amount DECIMAL(12,4) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    status VARCHAR(50) NOT NULL, -- PENDING, COMPLETED, FAILED, REFUNDED
    transaction_id VARCHAR(255) UNIQUE,
    payment_date TIMESTAMP,
    notes TEXT,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_payment_history_tenant ON tenant_payment_history(tenant_id, payment_date DESC);
CREATE INDEX idx_payment_history_invoice ON tenant_payment_history(invoice_id);
CREATE INDEX idx_payment_history_status ON tenant_payment_history(status, payment_date DESC);
CREATE INDEX idx_payment_history_transaction ON tenant_payment_history(transaction_id);
```

**الأولوية:** 🟠 عالية


---

## 2. ملخص الجداول المفقودة

| # | اسم الجدول | الأولوية | التأثير | الحالة |
|---|------------|----------|---------|--------|
| 1 | role_permissions | 🔴 حرجة | RBAC معطل | ❌ مفقود |
| 2 | user_roles | 🔴 حرجة | لا يمكن تعيين أدوار | ❌ مفقود |
| 3 | admin_settings | 🔴 حرجة | لا إعدادات نظام | ❌ مفقود |
| 4 | tenant_settings | 🔴 حرجة | لا تخصيص للعملاء | ❌ مفقود |
| 5 | notifications | 🔴 حرجة | لا إشعارات | ❌ مفقود |
| 6 | email_templates | 🟠 عالية | emails hardcoded | ❌ مفقود |
| 7 | system_logs | 🟠 عالية | لا centralized logging | ❌ مفقود |
| 8 | scheduled_tasks | 🟠 عالية | لا task management | ❌ مفقود |
| 9 | tenant_quotas | 🟠 عالية | لا quota management | ❌ مفقود |
| 10 | tenant_payment_history | 🟠 عالية | لا payment tracking | ❌ مفقود |
| 11 | api_rate_limits | 🟡 متوسطة | rate limits ثابتة | ❌ مفقود |
| 12 | feature_flags | 🟡 متوسطة | لا feature toggles | ❌ مفقود |
| 13 | admin_announcements | 🟡 متوسطة | لا broadcast messaging | ❌ مفقود |
| 14 | backup_logs | 🟡 متوسطة | لا backup tracking | ❌ مفقود |
| 15 | admin_dashboard_widgets | 🟢 منخفضة | dashboard ثابت | ❌ مفقود |

---

## 3. التوصيات حسب الأولوية

### المرحلة 1: الجداول الحرجة (أسبوع واحد)

```sql
-- ملف: 050_critical_admin_tables.sql

BEGIN;

-- 1. role_permissions
CREATE TABLE IF NOT EXISTS role_permissions (
    id BIGSERIAL PRIMARY KEY,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(role_id, permission_id)
);

-- 2. user_roles
CREATE TABLE IF NOT EXISTS user_roles (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    assigned_by UUID REFERENCES users(id) ON DELETE SET NULL,
    assigned_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, role_id, tenant_id)
);

-- 3. admin_settings
CREATE TABLE IF NOT EXISTS admin_settings (
    id BIGSERIAL PRIMARY KEY,
    setting_key VARCHAR(255) NOT NULL UNIQUE,
    setting_value TEXT,
    value_type VARCHAR(50) DEFAULT 'STRING',
    category VARCHAR(100),
    description TEXT,
    is_public BOOLEAN DEFAULT false,
    is_encrypted BOOLEAN DEFAULT false,
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- 4. tenant_settings
CREATE TABLE IF NOT EXISTS tenant_settings (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    setting_key VARCHAR(255) NOT NULL,
    setting_value TEXT,
    value_type VARCHAR(50) DEFAULT 'STRING',
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(tenant_id, setting_key)
);

-- 5. notifications
CREATE TABLE IF NOT EXISTS notifications (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    category VARCHAR(100),
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    action_url VARCHAR(500),
    action_label VARCHAR(100),
    is_read BOOLEAN DEFAULT false,
    read_at TIMESTAMP,
    priority INTEGER DEFAULT 0,
    expires_at TIMESTAMP,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_role_permissions_role ON role_permissions(role_id);
CREATE INDEX idx_role_permissions_permission ON role_permissions(permission_id);
CREATE INDEX idx_user_roles_user ON user_roles(user_id);
CREATE INDEX idx_user_roles_role ON user_roles(role_id);
CREATE INDEX idx_user_roles_tenant ON user_roles(tenant_id);
CREATE INDEX idx_admin_settings_category ON admin_settings(category);
CREATE INDEX idx_tenant_settings_tenant ON tenant_settings(tenant_id);
CREATE INDEX idx_notifications_user ON notifications(user_id, is_read, created_at DESC);
CREATE INDEX idx_notifications_unread ON notifications(user_id, is_read) WHERE is_read = false;

COMMIT;
```

### المرحلة 2: الجداول العالية (أسبوعان)

```sql
-- ملف: 051_high_priority_admin_tables.sql

BEGIN;

-- 6. email_templates
CREATE TABLE IF NOT EXISTS email_templates (
    id BIGSERIAL PRIMARY KEY,
    template_key VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    subject VARCHAR(500) NOT NULL,
    body_html TEXT NOT NULL,
    body_text TEXT,
    category VARCHAR(100),
    variables JSONB,
    is_active BOOLEAN DEFAULT true,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- 7. system_logs
CREATE TABLE IF NOT EXISTS system_logs (
    id BIGSERIAL PRIMARY KEY,
    level VARCHAR(20) NOT NULL,
    source VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    error_code VARCHAR(50),
    stack_trace TEXT,
    context JSONB,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL,
    ip_address INET,
    created_at TIMESTAMP DEFAULT NOW()
) PARTITION BY RANGE (created_at);

-- 8. scheduled_tasks
CREATE TABLE IF NOT EXISTS scheduled_tasks (
    id BIGSERIAL PRIMARY KEY,
    task_name VARCHAR(255) NOT NULL UNIQUE,
    task_type VARCHAR(100) NOT NULL,
    schedule_expression VARCHAR(255) NOT NULL,
    handler_function VARCHAR(255) NOT NULL,
    parameters JSONB,
    is_active BOOLEAN DEFAULT true,
    last_run_at TIMESTAMP,
    last_run_status VARCHAR(50),
    last_run_duration_ms INTEGER,
    last_error TEXT,
    next_run_at TIMESTAMP,
    run_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- 9. tenant_quotas
CREATE TABLE IF NOT EXISTS tenant_quotas (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    quota_type VARCHAR(100) NOT NULL,
    quota_limit BIGINT NOT NULL,
    quota_used BIGINT DEFAULT 0,
    quota_period VARCHAR(50) DEFAULT 'MONTHLY',
    reset_at TIMESTAMP,
    is_hard_limit BOOLEAN DEFAULT true,
    warning_threshold INTEGER DEFAULT 80,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(tenant_id, quota_type, quota_period)
);

-- 10. tenant_payment_history
CREATE TABLE IF NOT EXISTS tenant_payment_history (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    invoice_id UUID REFERENCES billing_invoices(id) ON DELETE SET NULL,
    payment_method VARCHAR(50) NOT NULL,
    payment_gateway VARCHAR(100),
    amount DECIMAL(12,4) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    status VARCHAR(50) NOT NULL,
    transaction_id VARCHAR(255) UNIQUE,
    payment_date TIMESTAMP,
    notes TEXT,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_email_templates_category ON email_templates(category);
CREATE INDEX idx_system_logs_level ON system_logs(level, created_at DESC);
CREATE INDEX idx_system_logs_source ON system_logs(source, created_at DESC);
CREATE INDEX idx_scheduled_tasks_active ON scheduled_tasks(is_active, next_run_at);
CREATE INDEX idx_tenant_quotas_tenant ON tenant_quotas(tenant_id);
CREATE INDEX idx_payment_history_tenant ON tenant_payment_history(tenant_id, payment_date DESC);

COMMIT;
```

---

## 4. البيانات الأولية المطلوبة

### 4.1 الأدوار الافتراضية

```sql
-- Insert default roles
INSERT INTO roles (id, name, description) VALUES
(gen_random_uuid(), 'SUPER_ADMIN', 'Super Administrator with full access'),
(gen_random_uuid(), 'TENANT_ADMIN', 'Tenant Administrator'),
(gen_random_uuid(), 'TENANT_USER', 'Regular Tenant User'),
(gen_random_uuid(), 'STAFF_ADMIN', 'Staff Administrator'),
(gen_random_uuid(), 'STAFF_SUPPORT', 'Support Staff'),
(gen_random_uuid(), 'STAFF_BILLING', 'Billing Staff'),
(gen_random_uuid(), 'STAFF_VIEWER', 'Read-only Staff')
ON CONFLICT (name) DO NOTHING;
```

### 4.2 الصلاحيات الافتراضية

```sql
-- Insert default permissions
INSERT INTO permissions (id, name, resource, action, description) VALUES
-- User Management
(gen_random_uuid(), 'users.view', 'users', 'view', 'View users'),
(gen_random_uuid(), 'users.create', 'users', 'create', 'Create users'),
(gen_random_uuid(), 'users.update', 'users', 'update', 'Update users'),
(gen_random_uuid(), 'users.delete', 'users', 'delete', 'Delete users'),

-- Tenant Management
(gen_random_uuid(), 'tenants.view', 'tenants', 'view', 'View tenants'),
(gen_random_uuid(), 'tenants.create', 'tenants', 'create', 'Create tenants'),
(gen_random_uuid(), 'tenants.update', 'tenants', 'update', 'Update tenants'),
(gen_random_uuid(), 'tenants.delete', 'tenants', 'delete', 'Delete tenants'),

-- Message Management
(gen_random_uuid(), 'messages.view', 'messages', 'view', 'View messages'),
(gen_random_uuid(), 'messages.send', 'messages', 'send', 'Send messages'),
(gen_random_uuid(), 'messages.delete', 'messages', 'delete', 'Delete messages'),

-- Balance Management
(gen_random_uuid(), 'balance.view', 'balance', 'view', 'View balance'),
(gen_random_uuid(), 'balance.recharge', 'balance', 'recharge', 'Recharge balance'),
(gen_random_uuid(), 'balance.deduct', 'balance', 'deduct', 'Deduct balance'),

-- Settings Management
(gen_random_uuid(), 'settings.view', 'settings', 'view', 'View settings'),
(gen_random_uuid(), 'settings.update', 'settings', 'update', 'Update settings'),

-- Reports
(gen_random_uuid(), 'reports.view', 'reports', 'view', 'View reports'),
(gen_random_uuid(), 'reports.export', 'reports', 'export', 'Export reports')
ON CONFLICT (name) DO NOTHING;
```

### 4.3 الإعدادات الافتراضية

```sql
-- Insert default admin settings
INSERT INTO admin_settings (setting_key, setting_value, value_type, category, description) VALUES
('system.maintenance_mode', 'false', 'BOOLEAN', 'SYSTEM', 'Enable maintenance mode'),
('system.registration_enabled', 'true', 'BOOLEAN', 'SYSTEM', 'Allow new registrations'),
('billing.default_currency', 'USD', 'STRING', 'BILLING', 'Default currency'),
('billing.tax_rate', '0.15', 'DECIMAL', 'BILLING', 'Default tax rate (15%)'),
('messaging.max_retry_count', '3', 'INTEGER', 'MESSAGING', 'Maximum retry attempts'),
('messaging.default_sender_name', 'ALHudhud', 'STRING', 'MESSAGING', 'Default sender name'),
('security.session_timeout', '3600', 'INTEGER', 'SECURITY', 'Session timeout in seconds'),
('security.password_min_length', '8', 'INTEGER', 'SECURITY', 'Minimum password length'),
('security.require_2fa', 'false', 'BOOLEAN', 'SECURITY', 'Require two-factor authentication'),
('limits.default_message_limit', '1000', 'INTEGER', 'LIMITS', 'Default monthly message limit')
ON CONFLICT (setting_key) DO NOTHING;
```

---

## 5. الخلاصة

### المشاكل الرئيسية:
1. ❌ **RBAC غير مكتمل** - لا يوجد role_permissions و user_roles
2. ❌ **لا إعدادات** - لا admin_settings و tenant_settings
3. ❌ **لا إشعارات** - لا notifications system
4. ❌ **لا email templates** - كل emails hardcoded
5. ❌ **لا system logs** - صعوبة في debugging
6. ❌ **لا scheduled tasks** - لا task management
7. ❌ **لا quotas** - لا fair usage policy
8. ❌ **لا payment history** - ناقص financial tracking

### التأثير على النظام:
- 🔴 **نظام الصلاحيات معطل تماماً**
- 🔴 **لا يمكن تخصيص الإعدادات**
- 🔴 **تجربة مستخدم ضعيفة** (لا إشعارات)
- 🟠 **صعوبة في الصيانة** (لا logs مركزية)
- 🟠 **لا مرونة في الإدارة**

### الأولوية القصوى:
1. إنشاء role_permissions و user_roles
2. إنشاء admin_settings و tenant_settings
3. إنشاء notifications
4. إنشاء email_templates
5. إنشاء system_logs

**هذه الجداول ضرورية لنظام إدارة فعال!**

---

**تم إنشاء التقرير بواسطة:** Kiro AI  
**التاريخ:** 2026-03-21  
**الإصدار:** 1.0
