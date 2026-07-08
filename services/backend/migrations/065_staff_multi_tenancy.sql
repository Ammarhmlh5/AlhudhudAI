-- Migration 065: Staff Multi-tenancy
-- Description: Adds tenant_id to staff tables and enables RLS.

BEGIN;

-- 1. Add tenant_id to staff_profiles
ALTER TABLE staff_profiles ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_staff_profiles_tenant_id ON staff_profiles(tenant_id);

-- 2. Add tenant_id to roles (Optional: allowing tenant-specific roles)
ALTER TABLE roles ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_roles_tenant_id ON roles(tenant_id);

-- 3. Add tenant_id to staff_activities
ALTER TABLE staff_activities ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_staff_activities_tenant_id ON staff_activities(tenant_id);

-- 4. Enable RLS on these tables
ALTER TABLE staff_profiles ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_policy ON staff_profiles;
CREATE POLICY tenant_isolation_policy ON staff_profiles
USING (
    tenant_id = current_setting('app.current_tenant', true)::uuid 
    OR current_setting('app.is_super_admin', true) = 'true'
);

ALTER TABLE roles ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_policy ON roles;
CREATE POLICY tenant_isolation_policy ON roles
USING (
    tenant_id = current_setting('app.current_tenant', true)::uuid 
    OR tenant_id IS NULL -- System roles
    OR current_setting('app.is_super_admin', true) = 'true'
);

ALTER TABLE staff_activities ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_policy ON staff_activities;
CREATE POLICY tenant_isolation_policy ON staff_activities
USING (
    tenant_id = current_setting('app.current_tenant', true)::uuid 
    OR current_setting('app.is_super_admin', true) = 'true'
);

COMMIT;
