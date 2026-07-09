-- Migration: 020_dedicated_gateways_table.sql
-- Description: إنشاء جدول البوابات المخصصة للشركات المؤسسية مع نموذج الاشتراك
-- Requirements: 8.1, 8.2, 8.3, 8.4, 8.10, 8.11, 8.13, 8.14
-- Date: 2024

-- ============================================================================
-- جدول البوابات المخصصة (Dedicated Gateways)
-- ============================================================================
-- نموذج العمل المالي:
-- - الشركة المؤسسية تدفع لشركة الاتصالات مباشرة لتكلفة الرسائل
-- - المنصة تتقاضى فقط:
--   1. رسوم الاشتراك (subscription_fee) - شهري أو سنوي
--   2. رسوم إيجار السيرفر (server_rental_fee) - شهري
-- - المنصة تراقب رصيد الشركة لدى شركة الاتصالات وترسل تنبيهات فقط
-- ============================================================================

CREATE TABLE IF NOT EXISTS dedicated_gateways (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- ========================================================================
    -- معلومات البوابة الأساسية
    -- ========================================================================
    gateway_name VARCHAR(255) NOT NULL,
    gateway_code VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    
    -- ========================================================================
    -- معلومات الاتصال بشركة الاتصالات
    -- ========================================================================
    telecom_provider VARCHAR(100) NOT NULL,
    telecom_api_endpoint TEXT NOT NULL,
    telecom_username VARCHAR(255),
    telecom_password_encrypted TEXT, -- مشفر باستخدام AES-256
    telecom_api_key_encrypted TEXT,  -- مشفر باستخدام AES-256
    
    -- ========================================================================
    -- معلومات السيرفر المخصص
    -- ========================================================================
    server_type VARCHAR(50) NOT NULL DEFAULT 'shared', -- 'dedicated', 'vps', 'container', 'shared'
    server_ip VARCHAR(45),
    server_hostname VARCHAR(255),
    server_specs JSONB, -- مثال: {"cpu": 4, "ram": "8GB", "storage": "100GB", "bandwidth": "1Gbps"}
    
    -- ========================================================================
    -- حدود الاستخدام
    -- ========================================================================
    max_throughput INTEGER DEFAULT 100, -- رسائل/ثانية
    daily_limit INTEGER,                -- الحد اليومي للرسائل
    monthly_limit INTEGER,              -- الحد الشهري للرسائل
    
    -- ========================================================================
    -- نموذج الاشتراك والفوترة
    -- ========================================================================
    -- ملاحظة: الشركة تدفع لشركة الاتصالات مباشرة لتكلفة الرسائل
    -- المنصة تتقاضى فقط رسوم الاشتراك والإيجار
    subscription_type VARCHAR(20) NOT NULL DEFAULT 'monthly', -- 'monthly', 'yearly'
    subscription_fee DECIMAL(10, 2) NOT NULL DEFAULT 0.00,    -- رسوم الاشتراك الشهري/السنوي للمنصة
    server_rental_fee DECIMAL(10, 2) NOT NULL DEFAULT 0.00,   -- رسوم إيجار السيرفر الشهرية
    setup_fee DECIMAL(10, 2) DEFAULT 0.00,                    -- رسوم الإعداد الأولية (مرة واحدة)
    billing_cycle VARCHAR(20) NOT NULL DEFAULT 'monthly',     -- 'monthly', 'yearly'
    next_billing_date DATE,                                   -- تاريخ الفوترة القادم
    
    -- ========================================================================
    -- مراقبة رصيد الشركة لدى شركة الاتصالات (للقراءة فقط)
    -- ========================================================================
    -- ملاحظة: هذه الحقول للمراقبة فقط، المنصة لا تعالج دفعات الرسائل
    telecom_balance DECIMAL(10, 2),              -- رصيد الشركة لدى شركة الاتصالات (للمراقبة)
    balance_alert_threshold DECIMAL(10, 2),      -- عتبة التنبيه عند انخفاض الرصيد
    last_balance_check TIMESTAMP,                -- آخر فحص للرصيد
    balance_check_enabled BOOLEAN DEFAULT true,  -- تفعيل/تعطيل مراقبة الرصيد
    
    -- ========================================================================
    -- معلومات العقد
    -- ========================================================================
    contract_start_date DATE NOT NULL,
    contract_end_date DATE,
    auto_renew BOOLEAN DEFAULT false,
    
    -- ========================================================================
    -- الحالة والصحة
    -- ========================================================================
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- 'active', 'suspended', 'maintenance', 'terminated'
    last_health_check TIMESTAMP,
    health_status VARCHAR(20) DEFAULT 'unknown',  -- 'healthy', 'degraded', 'unhealthy', 'unknown'
    
    -- ========================================================================
    -- التوقيتات
    -- ========================================================================
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- ========================================================================
    -- القيود (Constraints)
    -- ========================================================================
    CONSTRAINT check_subscription_type CHECK (subscription_type IN ('monthly', 'yearly')),
    CONSTRAINT check_billing_cycle CHECK (billing_cycle IN ('monthly', 'yearly')),
    CONSTRAINT check_status CHECK (status IN ('active', 'suspended', 'maintenance', 'terminated')),
    CONSTRAINT check_health_status CHECK (health_status IN ('healthy', 'degraded', 'unhealthy', 'unknown')),
    CONSTRAINT check_server_type CHECK (server_type IN ('dedicated', 'vps', 'container', 'shared')),
    CONSTRAINT check_subscription_fee CHECK (subscription_fee >= 0),
    CONSTRAINT check_server_rental_fee CHECK (server_rental_fee >= 0),
    CONSTRAINT check_setup_fee CHECK (setup_fee >= 0),
    CONSTRAINT check_max_throughput CHECK (max_throughput > 0),
    CONSTRAINT check_daily_limit CHECK (daily_limit IS NULL OR daily_limit > 0),
    CONSTRAINT check_monthly_limit CHECK (monthly_limit IS NULL OR monthly_limit > 0),
    CONSTRAINT check_balance_threshold CHECK (balance_alert_threshold IS NULL OR balance_alert_threshold >= 0),
    CONSTRAINT check_contract_dates CHECK (contract_end_date IS NULL OR contract_end_date >= contract_start_date)
);

-- ============================================================================
-- Indexes للأداء
-- ============================================================================

-- Index رئيسي للعزل بين الشركات (tenant isolation)
CREATE INDEX idx_dedicated_gateways_tenant ON dedicated_gateways(tenant_id);

-- Index لرمز البوابة (للبحث السريع)
CREATE INDEX idx_dedicated_gateways_code ON dedicated_gateways(gateway_code);

-- Index للحالة (لفلترة البوابات النشطة)
CREATE INDEX idx_dedicated_gateways_status ON dedicated_gateways(tenant_id, status);

-- Index لتاريخ الفوترة القادم (للفحص اليومي)
CREATE INDEX idx_dedicated_gateways_billing ON dedicated_gateways(next_billing_date) 
WHERE status = 'active';

-- Index لمراقبة الرصيد (للفحص الدوري)
CREATE INDEX idx_dedicated_gateways_balance_check ON dedicated_gateways(last_balance_check) 
WHERE balance_check_enabled = true AND status = 'active';

-- Index مركب للبحث والفلترة
CREATE INDEX idx_dedicated_gateways_tenant_status_type ON dedicated_gateways(tenant_id, status, subscription_type);

-- ============================================================================
-- Row Level Security (RLS) للعزل التام بين الشركات
-- ============================================================================

-- تفعيل RLS على الجدول
ALTER TABLE dedicated_gateways ENABLE ROW LEVEL SECURITY;

-- Policy للقراءة: كل شركة ترى بواباتها فقط
CREATE POLICY tenant_isolation_policy ON dedicated_gateways
    FOR SELECT
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy للإدراج: يجب أن يكون tenant_id مطابق للشركة الحالية
CREATE POLICY tenant_isolation_insert_policy ON dedicated_gateways
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy للتحديث: يمكن تحديث بوابات الشركة فقط
CREATE POLICY tenant_isolation_update_policy ON dedicated_gateways
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy للحذف: يمكن حذف بوابات الشركة فقط
CREATE POLICY tenant_isolation_delete_policy ON dedicated_gateways
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- ============================================================================
-- Trigger لتحديث updated_at تلقائياً
-- ============================================================================

CREATE OR REPLACE FUNCTION update_dedicated_gateways_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_dedicated_gateways_updated_at
    BEFORE UPDATE ON dedicated_gateways
    FOR EACH ROW
    EXECUTE FUNCTION update_dedicated_gateways_updated_at();

-- ============================================================================
-- Comments للتوثيق
-- ============================================================================

COMMENT ON TABLE dedicated_gateways IS 'جدول البوابات المخصصة للشركات المؤسسية - نموذج Gateway-as-a-Service';
COMMENT ON COLUMN dedicated_gateways.tenant_id IS 'معرف الشركة المؤسسية المالكة للبوابة';
COMMENT ON COLUMN dedicated_gateways.gateway_code IS 'رمز فريد للبوابة';
COMMENT ON COLUMN dedicated_gateways.telecom_password_encrypted IS 'كلمة مرور شركة الاتصالات مشفرة بـ AES-256';
COMMENT ON COLUMN dedicated_gateways.telecom_api_key_encrypted IS 'مفتاح API لشركة الاتصالات مشفر بـ AES-256';
COMMENT ON COLUMN dedicated_gateways.subscription_fee IS 'رسوم الاشتراك الشهري/السنوي للمنصة (لا تشمل تكلفة الرسائل)';
COMMENT ON COLUMN dedicated_gateways.server_rental_fee IS 'رسوم إيجار السيرفر الشهرية';
COMMENT ON COLUMN dedicated_gateways.telecom_balance IS 'رصيد الشركة لدى شركة الاتصالات - للمراقبة فقط، الشركة تدفع مباشرة';
COMMENT ON COLUMN dedicated_gateways.balance_alert_threshold IS 'عتبة التنبيه - يتم إرسال تنبيه عند انخفاض الرصيد عن هذا الحد';

-- ============================================================================
-- نهاية Migration
-- ============================================================================
