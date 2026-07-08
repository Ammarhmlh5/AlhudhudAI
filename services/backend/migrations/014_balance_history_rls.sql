-- Migration: Add Row Level Security to balance_history table
-- Purpose: Enable tenant isolation for balance audit trail
-- Related Requirements: 16.4, 16.5, 16.10, 26.5

-- Enable Row Level Security on balance_history
ALTER TABLE balance_history ENABLE ROW LEVEL SECURITY;

-- Policy for SELECT: Users can only see their own tenant's balance history
CREATE POLICY tenant_isolation_policy ON balance_history
    FOR SELECT
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy for INSERT: Users can only insert balance history for their own tenant
CREATE POLICY tenant_isolation_insert_policy ON balance_history
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy for UPDATE: Prevent updates to balance history (audit trail should be immutable)
-- This policy will deny all updates by design (requirement 16.11)
CREATE POLICY tenant_isolation_update_policy ON balance_history
    FOR UPDATE
    USING (false);

-- Policy for DELETE: Prevent deletions from balance history (audit trail should be immutable)
-- This policy will deny all deletes by design (requirement 16.11)
CREATE POLICY tenant_isolation_delete_policy ON balance_history
    FOR DELETE
    USING (false);

-- Add comments for documentation
COMMENT ON POLICY tenant_isolation_policy ON balance_history IS 'Ensures tenants can only view their own balance history records';
COMMENT ON POLICY tenant_isolation_insert_policy ON balance_history IS 'Ensures balance history records are created with correct tenant_id';
COMMENT ON POLICY tenant_isolation_update_policy ON balance_history IS 'Prevents any updates to balance history (immutable audit trail)';
COMMENT ON POLICY tenant_isolation_delete_policy ON balance_history IS 'Prevents any deletions from balance history (immutable audit trail)';
