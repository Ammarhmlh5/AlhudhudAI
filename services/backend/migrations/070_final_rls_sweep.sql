-- 070_final_rls_sweep.sql
-- Addressing Audit Point 4.2: Comprehensive RLS for all tenant-owned tables.

-- [1] Accounting Transactions
ALTER TABLE accounting_transactions ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_policy ON accounting_transactions;
CREATE POLICY tenant_isolation_policy ON accounting_transactions
USING (tenant_id = current_setting('app.current_tenant', true)::uuid OR current_setting('app.is_super_admin', true) = 'true');

-- [2] Billing Invoices
ALTER TABLE billing_invoices ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_policy ON billing_invoices;
CREATE POLICY tenant_isolation_policy ON billing_invoices
USING (tenant_id = current_setting('app.current_tenant', true)::uuid OR current_setting('app.is_super_admin', true) = 'true');

-- [3] Staff Activities & Profiles
ALTER TABLE staff_profiles ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_policy ON staff_profiles;
CREATE POLICY tenant_isolation_policy ON staff_profiles
USING (tenant_id = current_setting('app.current_tenant', true)::uuid OR current_setting('app.is_super_admin', true) = 'true');

ALTER TABLE staff_activities ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_policy ON staff_activities;
CREATE POLICY tenant_isolation_policy ON staff_activities
USING (tenant_id = current_setting('app.current_tenant', true)::uuid OR current_setting('app.is_super_admin', true) = 'true');

-- [4] Roles (Tenant-specific)
ALTER TABLE roles ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_policy ON roles;
CREATE POLICY tenant_isolation_policy ON roles
USING (tenant_id IS NULL OR tenant_id = current_setting('app.current_tenant', true)::uuid OR current_setting('app.is_super_admin', true) = 'true');

-- [5] Dispatch Channels
ALTER TABLE dispatch_channels ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_policy ON dispatch_channels;
CREATE POLICY tenant_isolation_policy ON dispatch_channels
USING (tenant_id = current_setting('app.current_tenant', true)::uuid OR current_setting('app.is_super_admin', true) = 'true');

-- [6] Segments (Note: segmentation_rules table does not exist in this version - Skipping)
