-- Migration: 023_gateway_users_table.sql
-- Description: إنشاء جدول gateway_users لإدارة مستخدمي البوابات المخصصة
-- Requirements: 11.1, 11.2, 11.3
-- Date: 2024

-- =====================================================
-- جدول مستخدمي البوابات المخصصة
-- =====================================================

CREATE TABLE IF NOT EXISTS gateway_users (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    gateway_id INTEGER NOT NULL REFERENCES dedicated_gateways(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- الصلاحيات
    role VARCHAR(50) NOT NULL CHECK (role IN ('admin', 'operator', 'viewer')),
    permissions JSONB DEFAULT '{}'::jsonb, -- {send_messages: true, view_reports: true, manage_users: false}
    
    -- الحالة
    is_active BOOLEAN DEFAULT true,
    
    -- التواريخ
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- قيود
    CONSTRAINT uq_gateway_users_gateway_user UNIQUE(gateway_id, user_id),
    CONSTRAINT fk_gateway_users_tenant_gateway FOREIGN KEY (tenant_id, gateway_id) 
        REFERENCES dedicated_gateways(tenant_id, id) ON DELETE CASCADE
);

-- =====================================================
-- Indexes للأداء
-- =====================================================

-- Index مركب يبدأ بـ tenant_id للعزل
CREATE INDEX idx_gateway_users_tenant ON gateway_users(tenant_id);
CREATE INDEX idx_gateway_users_tenant_gateway ON gateway_users(tenant_id, gateway_id);
CREATE INDEX idx_gateway_users_tenant_user ON gateway_users(tenant_id, user_id);
CREATE INDEX idx_gateway_users_tenant_role ON gateway_users(tenant_id, role);
CREATE INDEX idx_gateway_users_tenant_active ON gateway_users(tenant_id, is_active);

-- Indexes إضافية
CREATE INDEX idx_gateway_users_gateway ON gateway_users(gateway_id);
CREATE INDEX idx_gateway_users_user ON gateway_users(user_id);
CREATE INDEX idx_gateway_users_role ON gateway_users(role);

-- =====================================================
-- Row Level Security (RLS)
-- =====================================================

-- تفعيل RLS
ALTER TABLE gateway_users ENABLE ROW LEVEL SECURITY;

-- Policy للقراءة
CREATE POLICY tenant_isolation_policy ON gateway_users
    FOR SELECT
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy للإدراج
CREATE POLICY tenant_isolation_insert_policy ON gateway_users
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy للتحديث
CREATE POLICY tenant_isolation_update_policy ON gateway_users
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy للحذف
CREATE POLICY tenant_isolation_delete_policy ON gateway_users
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- =====================================================
-- Trigger لتحديث updated_at تلقائياً
-- =====================================================

CREATE OR REPLACE FUNCTION update_gateway_users_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_gateway_users_updated_at
    BEFORE UPDATE ON gateway_users
    FOR EACH ROW
    EXECUTE FUNCTION update_gateway_users_updated_at();

-- =====================================================
-- Trigger للتحقق من صلاحيات الأدوار
-- =====================================================

CREATE OR REPLACE FUNCTION validate_gateway_user_role()
RETURNS TRIGGER AS $$
BEGIN
    -- التحقق من الدور
    IF NEW.role NOT IN ('admin', 'operator', 'viewer') THEN
        RAISE EXCEPTION 'Invalid role: %. Must be admin, operator, or viewer', NEW.role;
    END IF;
    
    -- تعيين الصلاحيات الافتراضية حسب الدور
    IF NEW.permissions IS NULL OR NEW.permissions = '{}'::jsonb THEN
        CASE NEW.role
            WHEN 'admin' THEN
                NEW.permissions = jsonb_build_object(
                    'send_messages', true,
                    'view_reports', true,
                    'manage_users', true,
                    'manage_gateway', true,
                    'manage_api_keys', true,
                    'view_statistics', true,
                    'export_reports', true
                );
            WHEN 'operator' THEN
                NEW.permissions = jsonb_build_object(
                    'send_messages', true,
                    'view_reports', true,
                    'manage_users', false,
                    'manage_gateway', false,
                    'manage_api_keys', false,
                    'view_statistics', true,
                    'export_reports', true
                );
            WHEN 'viewer' THEN
                NEW.permissions = jsonb_build_object(
                    'send_messages', false,
                    'view_reports', true,
                    'manage_users', false,
                    'manage_gateway', false,
                    'manage_api_keys', false,
                    'view_statistics', true,
                    'export_reports', false
                );
        END CASE;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_validate_gateway_user_role
    BEFORE INSERT OR UPDATE ON gateway_users
    FOR EACH ROW
    EXECUTE FUNCTION validate_gateway_user_role();

-- =====================================================
-- Trigger للتحقق من tenant_id consistency
-- =====================================================

CREATE OR REPLACE FUNCTION check_gateway_user_tenant_consistency()
RETURNS TRIGGER AS $$
DECLARE
    gateway_tenant_id UUID;
    user_tenant_id UUID;
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
    
    -- التحقق من أن user_id ينتمي لنفس tenant_id
    SELECT tenant_id INTO user_tenant_id
    FROM users
    WHERE id = NEW.user_id;
    
    IF user_tenant_id IS NULL THEN
        RAISE EXCEPTION 'User % does not exist', NEW.user_id;
    END IF;
    
    IF user_tenant_id != NEW.tenant_id THEN
        RAISE EXCEPTION 'User % does not belong to tenant %', NEW.user_id, NEW.tenant_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_check_gateway_user_tenant_consistency
    BEFORE INSERT OR UPDATE ON gateway_users
    FOR EACH ROW
    EXECUTE FUNCTION check_gateway_user_tenant_consistency();

-- =====================================================
-- Comments للتوثيق
-- =====================================================

COMMENT ON TABLE gateway_users IS 'جدول مستخدمي البوابات المخصصة - يسجل المستخدمين المصرح لهم بالوصول لكل بوابة مخصصة';
COMMENT ON COLUMN gateway_users.id IS 'المعرف الفريد للمستخدم في البوابة';
COMMENT ON COLUMN gateway_users.tenant_id IS 'معرف الشركة (للعزل بين الشركات)';
COMMENT ON COLUMN gateway_users.gateway_id IS 'معرف البوابة المخصصة';
COMMENT ON COLUMN gateway_users.user_id IS 'معرف المستخدم';
COMMENT ON COLUMN gateway_users.role IS 'دور المستخدم: admin (كامل الصلاحيات)، operator (إرسال ومراقبة)، viewer (عرض فقط)';
COMMENT ON COLUMN gateway_users.permissions IS 'الصلاحيات التفصيلية بصيغة JSON';
COMMENT ON COLUMN gateway_users.is_active IS 'حالة المستخدم (نشط/معطل)';
COMMENT ON COLUMN gateway_users.created_at IS 'تاريخ إضافة المستخدم';
COMMENT ON COLUMN gateway_users.updated_at IS 'تاريخ آخر تحديث';

-- =====================================================
-- Rollback Script
-- =====================================================

-- لإلغاء هذا الـ Migration، قم بتشغيل:
-- DROP TRIGGER IF EXISTS trigger_check_gateway_user_tenant_consistency ON gateway_users;
-- DROP TRIGGER IF EXISTS trigger_validate_gateway_user_role ON gateway_users;
-- DROP TRIGGER IF EXISTS trigger_update_gateway_users_updated_at ON gateway_users;
-- DROP FUNCTION IF EXISTS check_gateway_user_tenant_consistency();
-- DROP FUNCTION IF EXISTS validate_gateway_user_role();
-- DROP FUNCTION IF EXISTS update_gateway_users_updated_at();
-- DROP TABLE IF EXISTS gateway_users CASCADE;
