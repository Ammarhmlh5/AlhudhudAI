-- Migration: 050_fix_rbac_tables.sql
-- Description: إصلاح نظام RBAC بإضافة الجداول المفقودة
-- Date: 2026-03-22
-- Priority: CRITICAL

BEGIN;

-- ================================================================
-- 1. إنشاء جدول role_permissions (many-to-many)
-- ================================================================
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (role_id, permission_id)
);

-- Indexes للأداء
CREATE INDEX IF NOT EXISTS idx_role_permissions_role 
    ON role_permissions(role_id);
CREATE INDEX IF NOT EXISTS idx_role_permissions_permission 
    ON role_permissions(permission_id);

COMMENT ON TABLE role_permissions IS 
    'جدول ربط الأدوار بالصلاحيات (many-to-many) - يربط كل دور بمجموعة من الصلاحيات';
COMMENT ON COLUMN role_permissions.role_id IS 
    'معرف الدور - مرتبط بجدول roles';
COMMENT ON COLUMN role_permissions.permission_id IS 
    'معرف الصلاحية - مرتبط بجدول permissions';

-- ================================================================
-- 2. إنشاء جدول user_roles (many-to-many)
-- ================================================================
CREATE TABLE IF NOT EXISTS user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    assigned_by UUID REFERENCES users(id) ON DELETE SET NULL,
    PRIMARY KEY (user_id, role_id)
);

-- Indexes للأداء
CREATE INDEX IF NOT EXISTS idx_user_roles_user 
    ON user_roles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role 
    ON user_roles(role_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_assigned_by 
    ON user_roles(assigned_by);
CREATE INDEX IF NOT EXISTS idx_user_roles_assigned_at 
    ON user_roles(assigned_at);

COMMENT ON TABLE user_roles IS 
    'جدول ربط المستخدمين بالأدوار (many-to-many) - يربط كل مستخدم بمجموعة من الأدوار';
COMMENT ON COLUMN user_roles.user_id IS 
    'معرف المستخدم - مرتبط بجدول users';
COMMENT ON COLUMN user_roles.role_id IS 
    'معرف الدور - مرتبط بجدول roles';
COMMENT ON COLUMN user_roles.assigned_at IS 
    'تاريخ ووقت تعيين الدور للمستخدم';
COMMENT ON COLUMN user_roles.assigned_by IS 
    'معرف المستخدم الذي قام بتعيين الدور (للتدقيق)';

-- ================================================================
-- 3. التحقق من البيانات وعرض إحصائيات
-- ================================================================
DO $$
DECLARE
    role_count INT;
    permission_count INT;
    user_count INT;
BEGIN
    SELECT COUNT(*) INTO role_count FROM roles;
    SELECT COUNT(*) INTO permission_count FROM permissions;
    SELECT COUNT(*) INTO user_count FROM users;
    
    RAISE NOTICE '========================================';
    RAISE NOTICE 'RBAC Tables Created Successfully';
    RAISE NOTICE '========================================';
    RAISE NOTICE 'عدد الأدوار الموجودة: %', role_count;
    RAISE NOTICE 'عدد الصلاحيات الموجودة: %', permission_count;
    RAISE NOTICE 'عدد المستخدمين: %', user_count;
    RAISE NOTICE '========================================';
    
    IF role_count = 0 THEN
        RAISE WARNING 'تحذير: لا توجد أدوار في النظام - يجب إنشاء أدوار أساسية';
    END IF;
    
    IF permission_count = 0 THEN
        RAISE WARNING 'تحذير: لا توجد صلاحيات في النظام - يجب إنشاء صلاحيات أساسية';
    END IF;
END $$;

COMMIT;
