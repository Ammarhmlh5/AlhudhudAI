-- Migration: 053_add_admin_tables.sql
-- Description: إضافة الجداول الإدارية المفقودة (فقط إذا كانت الميزات موجودة)
-- Date: 2026-03-22
-- Priority: MEDIUM
-- Note: هذا Migration اختياري - يضيف جداول للميزات الإدارية

BEGIN;

-- ================================================================
-- 1. جدول admin_settings - إعدادات النظام العامة
-- ================================================================
CREATE TABLE IF NOT EXISTS admin_settings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    key VARCHAR(255) UNIQUE NOT NULL,
    value TEXT,
    value_type VARCHAR(50) DEFAULT 'string', -- string, number, boolean, json
    category VARCHAR(100), -- system, email, sms, payment, etc.
    description TEXT,
    is_public BOOLEAN DEFAULT false, -- هل يمكن للمستخدمين رؤيتها
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_admin_settings_category 
    ON admin_settings(category);
CREATE INDEX IF NOT EXISTS idx_admin_settings_public 
    ON admin_settings(is_public) WHERE is_public = true;

COMMENT ON TABLE admin_settings IS
    'إعدادات النظام العامة - مثل SMTP، SMS Gateway، إلخ';

-- ================================================================
-- 2. جدول tenant_settings - إعدادات خاصة بكل tenant
-- ================================================================
CREATE TABLE IF NOT EXISTS tenant_settings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    key VARCHAR(255) NOT NULL,
    value TEXT,
    value_type VARCHAR(50) DEFAULT 'string',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, key)
);

CREATE INDEX IF NOT EXISTS idx_tenant_settings_tenant 
    ON tenant_settings(tenant_id);

COMMENT ON TABLE tenant_settings IS
    'إعدادات خاصة بكل مستأجر - تخصيص السلوك لكل tenant';

-- ================================================================
-- 3. جدول notifications - إشعارات النظام
-- ================================================================
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL, -- info, warning, error, success
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    link VARCHAR(500), -- رابط للإجراء
    is_read BOOLEAN DEFAULT false,
    read_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_notifications_user 
    ON notifications(user_id, is_read, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_tenant 
    ON notifications(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_unread 
    ON notifications(user_id) WHERE is_read = false;

COMMENT ON TABLE notifications IS
    'إشعارات النظام للمستخدمين - تنبيهات، تحذيرات، إلخ';

-- ================================================================
-- 4. جدول notification_preferences - تفضيلات الإشعارات
-- ================================================================
CREATE TABLE IF NOT EXISTS notification_preferences (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    notification_type VARCHAR(50) NOT NULL,
    email_enabled BOOLEAN DEFAULT true,
    sms_enabled BOOLEAN DEFAULT false,
    push_enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, notification_type)
);

CREATE INDEX IF NOT EXISTS idx_notification_preferences_user 
    ON notification_preferences(user_id);

COMMENT ON TABLE notification_preferences IS
    'تفضيلات المستخدم لتلقي الإشعارات عبر القنوات المختلفة';

-- ================================================================
-- 5. جدول system_health_checks - فحوصات صحة النظام
-- ================================================================
CREATE TABLE IF NOT EXISTS system_health_checks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    check_name VARCHAR(100) NOT NULL,
    check_type VARCHAR(50) NOT NULL, -- database, redis, api, service
    status VARCHAR(20) NOT NULL, -- healthy, degraded, unhealthy
    response_time_ms INTEGER,
    error_message TEXT,
    details JSONB,
    checked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_health_checks_name_time 
    ON system_health_checks(check_name, checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_health_checks_status 
    ON system_health_checks(status, checked_at DESC);

COMMENT ON TABLE system_health_checks IS
    'سجل فحوصات صحة النظام - قاعدة البيانات، Redis، APIs، إلخ';

-- ================================================================
-- 6. جدول api_rate_limits - حدود معدل استخدام API
-- ================================================================
CREATE TABLE IF NOT EXISTS api_rate_limits (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    endpoint VARCHAR(255) NOT NULL,
    requests_per_minute INTEGER DEFAULT 60,
    requests_per_hour INTEGER DEFAULT 1000,
    requests_per_day INTEGER DEFAULT 10000,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, endpoint)
);

CREATE INDEX IF NOT EXISTS idx_api_rate_limits_tenant 
    ON api_rate_limits(tenant_id) WHERE is_active = true;

COMMENT ON TABLE api_rate_limits IS
    'حدود معدل استخدام API لكل مستأجر ونقطة نهاية';

-- ================================================================
-- 7. جدول scheduled_tasks - المهام المجدولة
-- ================================================================
CREATE TABLE IF NOT EXISTS scheduled_tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    task_type VARCHAR(100) NOT NULL, -- cleanup, report, backup, etc.
    schedule VARCHAR(100) NOT NULL, -- cron expression
    is_active BOOLEAN DEFAULT true,
    last_run_at TIMESTAMP,
    last_run_status VARCHAR(50), -- success, failed, running
    last_run_duration_ms INTEGER,
    last_error TEXT,
    next_run_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_scheduled_tasks_active 
    ON scheduled_tasks(is_active, next_run_at);

COMMENT ON TABLE scheduled_tasks IS
    'المهام المجدولة - تنظيف، تقارير، نسخ احتياطي، إلخ';

-- ================================================================
-- 8. جدول feature_flags - أعلام الميزات
-- ================================================================
CREATE TABLE IF NOT EXISTS feature_flags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    is_enabled BOOLEAN DEFAULT false,
    rollout_percentage INTEGER DEFAULT 0 CHECK (rollout_percentage >= 0 AND rollout_percentage <= 100),
    tenant_whitelist UUID[], -- قائمة tenants المسموح لهم
    tenant_blacklist UUID[], -- قائمة tenants الممنوعين
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_feature_flags_enabled 
    ON feature_flags(is_enabled);

COMMENT ON TABLE feature_flags IS
    'أعلام الميزات - تفعيل/تعطيل ميزات معينة ديناميكياً';

-- ================================================================
-- 9. جدول audit_logs - سجلات التدقيق الشاملة
-- ================================================================
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL, -- CREATE, UPDATE, DELETE, LOGIN, etc.
    resource_type VARCHAR(100) NOT NULL, -- user, message, tenant, etc.
    resource_id UUID,
    old_values JSONB,
    new_values JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_time 
    ON audit_logs(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_time 
    ON audit_logs(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource 
    ON audit_logs(resource_type, resource_id, created_at DESC);

COMMENT ON TABLE audit_logs IS
    'سجلات التدقيق الشاملة - تتبع جميع التغييرات في النظام';

-- ================================================================
-- 10. جدول email_templates - قوالب البريد الإلكتروني
-- ================================================================
CREATE TABLE IF NOT EXISTS email_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) UNIQUE NOT NULL,
    subject VARCHAR(500) NOT NULL,
    body_html TEXT NOT NULL,
    body_text TEXT,
    variables JSONB, -- قائمة المتغيرات المتاحة
    category VARCHAR(100), -- welcome, notification, invoice, etc.
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_email_templates_category 
    ON email_templates(category) WHERE is_active = true;

COMMENT ON TABLE email_templates IS
    'قوالب البريد الإلكتروني - رسائل ترحيب، إشعارات، فواتير، إلخ';

-- ================================================================
-- 11. جدول sms_templates - قوالب الرسائل النصية
-- ================================================================
CREATE TABLE IF NOT EXISTS sms_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) UNIQUE NOT NULL,
    content TEXT NOT NULL,
    variables JSONB,
    category VARCHAR(100),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sms_templates_category 
    ON sms_templates(category) WHERE is_active = true;

COMMENT ON TABLE sms_templates IS
    'قوالب الرسائل النصية - OTP، تنبيهات، إلخ';

-- ================================================================
-- 12. جدول backup_logs - سجلات النسخ الاحتياطي
-- ================================================================
CREATE TABLE IF NOT EXISTS backup_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    backup_type VARCHAR(50) NOT NULL, -- full, incremental, differential
    database_name VARCHAR(255) NOT NULL,
    file_path VARCHAR(1000),
    file_size_bytes BIGINT,
    status VARCHAR(50) NOT NULL, -- started, completed, failed
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    duration_seconds INTEGER,
    error_message TEXT,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_backup_logs_time 
    ON backup_logs(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_backup_logs_status 
    ON backup_logs(status, started_at DESC);

COMMENT ON TABLE backup_logs IS
    'سجلات النسخ الاحتياطي - تتبع عمليات النسخ الاحتياطي';

-- ================================================================
-- 13. جدول system_metrics - مقاييس النظام
-- ================================================================
CREATE TABLE IF NOT EXISTS system_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    metric_name VARCHAR(100) NOT NULL,
    metric_value DECIMAL(20, 4) NOT NULL,
    metric_unit VARCHAR(50), -- requests, ms, bytes, percentage, etc.
    tags JSONB, -- إضافة tags للتصنيف
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_system_metrics_name_time 
    ON system_metrics(metric_name, recorded_at DESC);

COMMENT ON TABLE system_metrics IS
    'مقاييس النظام - CPU، Memory، Requests، Response Time، إلخ';

-- ================================================================
-- 14. عرض ملخص الجداول المُنشأة
-- ================================================================
DO $$
DECLARE
    table_record RECORD;
    total_count INT := 0;
BEGIN
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Admin Tables Created';
    RAISE NOTICE '========================================';
    
    FOR table_record IN
        SELECT 
            table_name,
            pg_size_pretty(pg_total_relation_size(table_name::regclass)) as size
        FROM information_schema.tables
        WHERE table_schema = 'public'
            AND table_name IN (
                'admin_settings', 'tenant_settings', 'notifications',
                'notification_preferences', 'system_health_checks',
                'api_rate_limits', 'scheduled_tasks', 'feature_flags',
                'audit_logs', 'email_templates', 'sms_templates',
                'backup_logs', 'system_metrics'
            )
        ORDER BY table_name
    LOOP
        RAISE NOTICE '% (size: %)', 
            table_record.table_name,
            table_record.size;
        total_count := total_count + 1;
    END LOOP;
    
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Total admin tables: %', total_count;
    RAISE NOTICE '========================================';
END $$;




COMMIT;
