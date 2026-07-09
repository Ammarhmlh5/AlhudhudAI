-- Migration: Add Row Level Security to core messaging tables
-- Purpose: Complete tenant isolation for the messaging engine
-- Created: 2026-03-20

-- 1. messages
ALTER TABLE messages ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_policy ON messages;
CREATE POLICY tenant_isolation_policy ON messages
    USING (
        tenant_id = current_setting('app.current_tenant', true)::uuid 
        OR current_setting('app.is_super_admin', true) = 'true'
    );

-- 2. dispatch_channels
ALTER TABLE dispatch_channels ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_policy ON dispatch_channels;
CREATE POLICY tenant_isolation_policy ON dispatch_channels
    USING (
        tenant_id = current_setting('app.current_tenant', true)::uuid 
        OR current_setting('app.is_super_admin', true) = 'true'
    );

-- 3. incoming_messages
ALTER TABLE incoming_messages ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_policy ON incoming_messages;
CREATE POLICY tenant_isolation_policy ON incoming_messages
    USING (
        tenant_id = current_setting('app.current_tenant', true)::uuid 
        OR current_setting('app.is_super_admin', true) = 'true'
    );

-- 4. accounting_transactions
ALTER TABLE accounting_transactions ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_policy ON accounting_transactions;
CREATE POLICY tenant_isolation_policy ON accounting_transactions
    USING (
        tenant_id = current_setting('app.current_tenant', true)::uuid 
        OR current_setting('app.is_super_admin', true) = 'true'
    );

-- 5. balance_history (Ensure super_admin bypass)
ALTER TABLE balance_history ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_policy ON balance_history;
CREATE POLICY tenant_isolation_policy ON balance_history
    USING (
        tenant_id = current_setting('app.current_tenant', true)::uuid 
        OR current_setting('app.is_super_admin', true) = 'true'
    );
