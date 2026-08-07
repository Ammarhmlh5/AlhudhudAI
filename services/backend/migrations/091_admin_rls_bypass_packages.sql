-- 091_admin_rls_bypass_packages.sql
-- Align packages / subscriptions / rollover_messages RLS with the established pattern
-- (missing_ok=true + super-admin bypass) used by migrations 069/070/073/074.
--
-- Why:
--   1. The original policies (016/017/018) called current_setting('app.current_tenant')
--      without missing_ok=true, which raises "unrecognized configuration parameter"
--      whenever the middleware has not set the tenant context (e.g. SUPER_ADMIN routes).
--   2. There was no app.is_super_admin bypass, so SUPER_ADMIN cross-tenant management
--      (GET/POST/PUT/DELETE /api/v1/admin/packages) was blocked by RLS.

BEGIN;

-- [1] packages
ALTER TABLE packages ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_select_policy ON packages;
CREATE POLICY tenant_isolation_select_policy ON packages
    FOR SELECT
    USING (tenant_id = current_setting('app.current_tenant', true)::uuid OR current_setting('app.is_super_admin', true) = 'true');

DROP POLICY IF EXISTS tenant_isolation_insert_policy ON packages;
CREATE POLICY tenant_isolation_insert_policy ON packages
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant', true)::uuid OR current_setting('app.is_super_admin', true) = 'true');

DROP POLICY IF EXISTS tenant_isolation_update_policy ON packages;
CREATE POLICY tenant_isolation_update_policy ON packages
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant', true)::uuid OR current_setting('app.is_super_admin', true) = 'true')
    WITH CHECK (tenant_id = current_setting('app.current_tenant', true)::uuid OR current_setting('app.is_super_admin', true) = 'true');

DROP POLICY IF EXISTS tenant_isolation_delete_policy ON packages;
CREATE POLICY tenant_isolation_delete_policy ON packages
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant', true)::uuid OR current_setting('app.is_super_admin', true) = 'true');

-- [2] subscriptions
ALTER TABLE subscriptions ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_select_policy ON subscriptions;
CREATE POLICY tenant_isolation_select_policy ON subscriptions
    FOR SELECT
    USING (tenant_id = current_setting('app.current_tenant', true)::uuid OR current_setting('app.is_super_admin', true) = 'true');

DROP POLICY IF EXISTS tenant_isolation_insert_policy ON subscriptions;
CREATE POLICY tenant_isolation_insert_policy ON subscriptions
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant', true)::uuid OR current_setting('app.is_super_admin', true) = 'true');

DROP POLICY IF EXISTS tenant_isolation_update_policy ON subscriptions;
CREATE POLICY tenant_isolation_update_policy ON subscriptions
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant', true)::uuid OR current_setting('app.is_super_admin', true) = 'true')
    WITH CHECK (tenant_id = current_setting('app.current_tenant', true)::uuid OR current_setting('app.is_super_admin', true) = 'true');

DROP POLICY IF EXISTS tenant_isolation_delete_policy ON subscriptions;
CREATE POLICY tenant_isolation_delete_policy ON subscriptions
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant', true)::uuid OR current_setting('app.is_super_admin', true) = 'true');

-- [3] rollover_messages
ALTER TABLE rollover_messages ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_select_policy ON rollover_messages;
CREATE POLICY tenant_isolation_select_policy ON rollover_messages
    FOR SELECT
    USING (tenant_id = current_setting('app.current_tenant', true)::uuid OR current_setting('app.is_super_admin', true) = 'true');

DROP POLICY IF EXISTS tenant_isolation_insert_policy ON rollover_messages;
CREATE POLICY tenant_isolation_insert_policy ON rollover_messages
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant', true)::uuid OR current_setting('app.is_super_admin', true) = 'true');

DROP POLICY IF EXISTS tenant_isolation_update_policy ON rollover_messages;
CREATE POLICY tenant_isolation_update_policy ON rollover_messages
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant', true)::uuid OR current_setting('app.is_super_admin', true) = 'true')
    WITH CHECK (tenant_id = current_setting('app.current_tenant', true)::uuid OR current_setting('app.is_super_admin', true) = 'true');

DROP POLICY IF EXISTS tenant_isolation_delete_policy ON rollover_messages;
CREATE POLICY tenant_isolation_delete_policy ON rollover_messages
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant', true)::uuid OR current_setting('app.is_super_admin', true) = 'true');

COMMIT;
