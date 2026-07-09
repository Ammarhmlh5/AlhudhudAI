-- Migration 062: Audit and Consistency Tools
-- Description: Adds tools for verifying financial data and auditing.

-- 1. Balance Consistency Check Function
CREATE OR REPLACE FUNCTION check_balance_consistency()
RETURNS TABLE(
    tenant_id UUID,
    current_balance DECIMAL(12,4),
    calculated_balance DECIMAL(12,4),
    difference DECIMAL(12,4)
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        t.id,
        t.balance,
        COALESCE(bh.balance_after, 0),
        t.balance - COALESCE(bh.balance_after, 0)
    FROM tenants t
    LEFT JOIN LATERAL (
        SELECT balance_after
        FROM balance_history
        WHERE balance_history.tenant_id = t.id
        ORDER BY created_at DESC
        LIMIT 1
    ) bh ON true
    WHERE ABS(t.balance - COALESCE(bh.balance_after, 0)) > 0.01;
END;
$$ LANGUAGE plpgsql;

-- 2. Audit Log Indexing for performance
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_id ON audit_logs(resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_created ON audit_logs(tenant_id, created_at DESC);
