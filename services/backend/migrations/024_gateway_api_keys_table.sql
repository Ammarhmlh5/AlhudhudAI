-- Migration: 024_gateway_api_keys_table.sql
-- Description: إنشاء جدول gateway_api_keys لإدارة مفاتيح API للبوابات المخصصة
-- Requirements: 12.1, 12.2, 12.3, 12.4, 12.5, 12.6
-- Date: 2024

-- =====================================================
-- جدول مفاتيح API للبوابات المخصصة
-- =====================================================

CREATE TABLE IF NOT EXISTS gateway_api_keys (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    gateway_id INTEGER NOT NULL REFERENCES dedicated_gateways(id) ON DELETE CASCADE,
    
    -- معلومات المفتاح
    key_name VARCHAR(255) NOT NULL,
    description TEXT,
    api_key_hash VARCHAR(255) NOT NULL UNIQUE, -- bcrypt hash of the API key
    api_secret_hash VARCHAR(255), -- optional secret for additional security
    
    -- الصلاحيات
    permissions JSONB DEFAULT '{}'::jsonb, -- {send_sms: true, view_stats: true, manage_gateway: false}
    rate_limit INTEGER DEFAULT 60, -- requests per minute (default: 60)
    
    -- IP Whitelisting
    allowed_ips TEXT[], -- ['192.168.1.1', '10.0.0.0/24', '2001:db8::/32']
    
    -- الحالة والاستخدام
    is_active BOOLEAN DEFAULT true,
    expires_at TIMESTAMP, -- NULL means no expiration
    last_used_at TIMESTAMP,
    usage_count INTEGER DEFAULT 0,
    
    -- التواريخ
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- قيود
    CONSTRAINT fk_gateway_api_keys_tenant_gateway FOREIGN KEY (tenant_id, gateway_id) 
        REFERENCES dedicated_gateways(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT check_rate_limit_positive CHECK (rate_limit > 0)
);

-- =====================================================
-- Indexes للأداء
-- =====================================================

-- Index مركب يبدأ بـ tenant_id للعزل
CREATE INDEX idx_gateway_api_keys_tenant ON gateway_api_keys(tenant_id);
CREATE INDEX idx_gateway_api_keys_tenant_gateway ON gateway_api_keys(tenant_id, gateway_id);
CREATE INDEX idx_gateway_api_keys_tenant_active ON gateway_api_keys(tenant_id, is_active);
CREATE INDEX idx_gateway_api_keys_tenant_expires ON gateway_api_keys(tenant_id, expires_at) WHERE expires_at IS NOT NULL;

-- Indexes إضافية للبحث السريع
CREATE INDEX idx_gateway_api_keys_gateway ON gateway_api_keys(gateway_id);
CREATE INDEX idx_gateway_api_keys_hash ON gateway_api_keys(api_key_hash);
CREATE INDEX idx_gateway_api_keys_active ON gateway_api_keys(is_active);
CREATE INDEX idx_gateway_api_keys_expires ON gateway_api_keys(expires_at) WHERE expires_at IS NOT NULL;

-- =====================================================
-- Row Level Security (RLS)
-- =====================================================

-- تفعيل RLS
ALTER TABLE gateway_api_keys ENABLE ROW LEVEL SECURITY;

-- Policy للقراءة
CREATE POLICY tenant_isolation_policy ON gateway_api_keys
    FOR SELECT
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy للإدراج
CREATE POLICY tenant_isolation_insert_policy ON gateway_api_keys
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy للتحديث
CREATE POLICY tenant_isolation_update_policy ON gateway_api_keys
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy للحذف
CREATE POLICY tenant_isolation_delete_policy ON gateway_api_keys
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- =====================================================
-- Trigger لتحديث updated_at تلقائياً
-- =====================================================

CREATE OR REPLACE FUNCTION update_gateway_api_keys_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_gateway_api_keys_updated_at
    BEFORE UPDATE ON gateway_api_keys
    FOR EACH ROW
    EXECUTE FUNCTION update_gateway_api_keys_updated_at();

-- =====================================================
-- Trigger للتحقق من tenant_id consistency
-- =====================================================

CREATE OR REPLACE FUNCTION check_gateway_api_key_tenant_consistency()
RETURNS TRIGGER AS $$
DECLARE
    gateway_tenant_id UUID;
BEGIN
    -- التحقق من أن gateway_id ينتمي لنفس tenant_id
    SELECT tenant_id INTO gateway_tenant_id
    FROM dedicated_gateways
    WHERE id = NEW.gateway_id;
    
    IF gateway_tenant_id IS NULL THEN
        RAISE EXCEPTION 'Gateway % does not exist', NEW.gateway_id;
    END IF;
    
    IF gateway_tenant_id != NEW.tenant_id THEN
        RAISE EXCEPTION 'Gateway % does not belong to tenant %', NEW.gateway_id, NEW.tenant_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_check_gateway_api_key_tenant_consistency
    BEFORE INSERT OR UPDATE ON gateway_api_keys
    FOR EACH ROW
    EXECUTE FUNCTION check_gateway_api_key_tenant_consistency();

-- =====================================================
-- Trigger للتحقق من صلاحية IP Addresses
-- =====================================================

CREATE OR REPLACE FUNCTION validate_gateway_api_key_ips()
RETURNS TRIGGER AS $$
DECLARE
    ip TEXT;
BEGIN
    -- التحقق من صحة IP addresses إذا تم تحديدها
    IF NEW.allowed_ips IS NOT NULL AND array_length(NEW.allowed_ips, 1) > 0 THEN
        FOREACH ip IN ARRAY NEW.allowed_ips
        LOOP
            -- التحقق من صحة IPv4 أو IPv6 أو CIDR notation
            IF ip !~ '^([0-9]{1,3}\.){3}[0-9]{1,3}(/[0-9]{1,2})?$' AND 
               ip !~ '^([0-9a-fA-F]{0,4}:){2,7}[0-9a-fA-F]{0,4}(/[0-9]{1,3})?$' THEN
                RAISE EXCEPTION 'Invalid IP address or CIDR notation: %', ip;
            END IF;
        END LOOP;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_validate_gateway_api_key_ips
    BEFORE INSERT OR UPDATE ON gateway_api_keys
    FOR EACH ROW
    EXECUTE FUNCTION validate_gateway_api_key_ips();

-- =====================================================
-- Trigger لتعيين الصلاحيات الافتراضية
-- =====================================================

CREATE OR REPLACE FUNCTION set_default_gateway_api_key_permissions()
RETURNS TRIGGER AS $$
BEGIN
    -- تعيين الصلاحيات الافتراضية إذا لم يتم تحديدها
    IF NEW.permissions IS NULL OR NEW.permissions = '{}'::jsonb THEN
        NEW.permissions = jsonb_build_object(
            'send_sms', true,
            'view_stats', true,
            'view_reports', true,
            'manage_gateway', false,
            'manage_users', false,
            'manage_api_keys', false
        );
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_set_default_gateway_api_key_permissions
    BEFORE INSERT ON gateway_api_keys
    FOR EACH ROW
    EXECUTE FUNCTION set_default_gateway_api_key_permissions();

-- =====================================================
-- Function للتحقق من صلاحية API Key
-- =====================================================

CREATE OR REPLACE FUNCTION validate_api_key(
    p_api_key_hash VARCHAR(255),
    p_client_ip INET
) RETURNS TABLE (
    is_valid BOOLEAN,
    key_id INTEGER,
    gateway_id INTEGER,
    tenant_id UUID,
    permissions JSONB,
    rate_limit INTEGER,
    error_message TEXT
) AS $$
DECLARE
    v_key RECORD;
    v_ip_allowed BOOLEAN;
BEGIN
    -- البحث عن المفتاح
    SELECT * INTO v_key
    FROM gateway_api_keys
    WHERE api_key_hash = p_api_key_hash;
    
    -- التحقق من وجود المفتاح
    IF NOT FOUND THEN
        RETURN QUERY SELECT false, NULL::INTEGER, NULL::INTEGER, NULL::UUID, NULL::JSONB, NULL::INTEGER, 'Invalid API key'::TEXT;
        RETURN;
    END IF;
    
    -- التحقق من حالة المفتاح
    IF NOT v_key.is_active THEN
        RETURN QUERY SELECT false, v_key.id, v_key.gateway_id, v_key.tenant_id, v_key.permissions, v_key.rate_limit, 'API key is not active'::TEXT;
        RETURN;
    END IF;
    
    -- التحقق من تاريخ الانتهاء
    IF v_key.expires_at IS NOT NULL AND v_key.expires_at < CURRENT_TIMESTAMP THEN
        RETURN QUERY SELECT false, v_key.id, v_key.gateway_id, v_key.tenant_id, v_key.permissions, v_key.rate_limit, 'API key has expired'::TEXT;
        RETURN;
    END IF;
    
    -- التحقق من IP Whitelisting
    IF v_key.allowed_ips IS NOT NULL AND array_length(v_key.allowed_ips, 1) > 0 THEN
        v_ip_allowed := false;
        
        -- التحقق من تطابق IP مع القائمة المسموحة
        FOR i IN 1..array_length(v_key.allowed_ips, 1) LOOP
            IF p_client_ip <<= v_key.allowed_ips[i]::inet THEN
                v_ip_allowed := true;
                EXIT;
            END IF;
        END LOOP;
        
        IF NOT v_ip_allowed THEN
            -- تسجيل محاولة الوصول غير المصرح بها
            INSERT INTO security_logs (
                tenant_id,
                event_type,
                resource_type,
                resource_id,
                ip_address,
                details,
                created_at
            ) VALUES (
                v_key.tenant_id,
                'UNAUTHORIZED_API_KEY_ACCESS',
                'gateway_api_key',
                v_key.id::TEXT,
                host(p_client_ip),
                jsonb_build_object(
                    'api_key_id', v_key.id,
                    'gateway_id', v_key.gateway_id,
                    'client_ip', host(p_client_ip),
                    'allowed_ips', v_key.allowed_ips
                ),
                CURRENT_TIMESTAMP
            );
            
            RETURN QUERY SELECT false, v_key.id, v_key.gateway_id, v_key.tenant_id, v_key.permissions, v_key.rate_limit, 'IP address not whitelisted'::TEXT;
            RETURN;
        END IF;
    END IF;
    
    -- تحديث آخر استخدام وعداد الاستخدام
    UPDATE gateway_api_keys
    SET last_used_at = CURRENT_TIMESTAMP,
        usage_count = usage_count + 1
    WHERE id = v_key.id;
    
    -- المفتاح صالح
    RETURN QUERY SELECT true, v_key.id, v_key.gateway_id, v_key.tenant_id, v_key.permissions, v_key.rate_limit, NULL::TEXT;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- Function لتنظيف المفاتيح المنتهية
-- =====================================================

CREATE OR REPLACE FUNCTION cleanup_expired_api_keys()
RETURNS INTEGER AS $$
DECLARE
    v_deleted_count INTEGER;
BEGIN
    -- تعطيل المفاتيح المنتهية (لا نحذفها للاحتفاظ بالسجل)
    UPDATE gateway_api_keys
    SET is_active = false,
        updated_at = CURRENT_TIMESTAMP
    WHERE expires_at IS NOT NULL
      AND expires_at < CURRENT_TIMESTAMP
      AND is_active = true;
    
    GET DIAGNOSTICS v_deleted_count = ROW_COUNT;
    
    RETURN v_deleted_count;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- Comments للتوثيق
-- =====================================================

COMMENT ON TABLE gateway_api_keys IS 'جدول مفاتيح API للبوابات المخصصة - يسجل مفاتيح API المصرح بها لكل بوابة مخصصة';
COMMENT ON COLUMN gateway_api_keys.id IS 'المعرف الفريد لمفتاح API';
COMMENT ON COLUMN gateway_api_keys.tenant_id IS 'معرف الشركة (للعزل بين الشركات)';
COMMENT ON COLUMN gateway_api_keys.gateway_id IS 'معرف البوابة المخصصة';
COMMENT ON COLUMN gateway_api_keys.key_name IS 'اسم المفتاح (للتعريف)';
COMMENT ON COLUMN gateway_api_keys.description IS 'وصف المفتاح والغرض منه';
COMMENT ON COLUMN gateway_api_keys.api_key_hash IS 'Hash آمن لمفتاح API (bcrypt)';
COMMENT ON COLUMN gateway_api_keys.api_secret_hash IS 'Hash آمن للسر الإضافي (اختياري)';
COMMENT ON COLUMN gateway_api_keys.permissions IS 'الصلاحيات المخصصة بصيغة JSON';
COMMENT ON COLUMN gateway_api_keys.rate_limit IS 'الحد الأقصى للطلبات في الدقيقة';
COMMENT ON COLUMN gateway_api_keys.allowed_ips IS 'قائمة IP addresses المسموح بها (IPv4/IPv6/CIDR)';
COMMENT ON COLUMN gateway_api_keys.is_active IS 'حالة المفتاح (نشط/معطل)';
COMMENT ON COLUMN gateway_api_keys.expires_at IS 'تاريخ انتهاء صلاحية المفتاح (NULL = لا ينتهي)';
COMMENT ON COLUMN gateway_api_keys.last_used_at IS 'تاريخ آخر استخدام للمفتاح';
COMMENT ON COLUMN gateway_api_keys.usage_count IS 'عدد مرات استخدام المفتاح';
COMMENT ON COLUMN gateway_api_keys.created_at IS 'تاريخ إنشاء المفتاح';
COMMENT ON COLUMN gateway_api_keys.updated_at IS 'تاريخ آخر تحديث';

COMMENT ON FUNCTION validate_api_key(VARCHAR, INET) IS 'التحقق من صلاحية API Key مع فحص IP Whitelisting وتاريخ الانتهاء';
COMMENT ON FUNCTION cleanup_expired_api_keys() IS 'تعطيل المفاتيح المنتهية تلقائياً';

-- =====================================================
-- Rollback Script
-- =====================================================

-- لإلغاء هذا الـ Migration، قم بتشغيل:
-- DROP FUNCTION IF EXISTS cleanup_expired_api_keys();
-- DROP FUNCTION IF EXISTS validate_api_key(VARCHAR, INET);
-- DROP TRIGGER IF EXISTS trigger_set_default_gateway_api_key_permissions ON gateway_api_keys;
-- DROP TRIGGER IF EXISTS trigger_validate_gateway_api_key_ips ON gateway_api_keys;
-- DROP TRIGGER IF EXISTS trigger_check_gateway_api_key_tenant_consistency ON gateway_api_keys;
-- DROP TRIGGER IF EXISTS trigger_update_gateway_api_keys_updated_at ON gateway_api_keys;
-- DROP FUNCTION IF EXISTS set_default_gateway_api_key_permissions();
-- DROP FUNCTION IF EXISTS validate_gateway_api_key_ips();
-- DROP FUNCTION IF EXISTS check_gateway_api_key_tenant_consistency();
-- DROP FUNCTION IF EXISTS update_gateway_api_keys_updated_at();
-- DROP TABLE IF EXISTS gateway_api_keys CASCADE;
