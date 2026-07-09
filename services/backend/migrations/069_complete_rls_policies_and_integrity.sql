-- 069_complete_rls_policies_and_integrity.sql
-- Addressing Audit Points 492 & 639: RLS Expansion and Data Integrity

-- [1] RLS EXPANSION
-- Enabling RLS on remaining tenant-owned tables to prevent data leakage.

-- Dedicated Gateways
ALTER TABLE dedicated_gateways ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_policy ON dedicated_gateways;
CREATE POLICY tenant_isolation_policy ON dedicated_gateways
USING (tenant_id = current_setting('app.current_tenant', true)::uuid OR current_setting('app.is_super_admin', true) = 'true');

-- [2] DATA INTEGRITY PROCEDURE
-- NOTE: check_transaction_integrity() is replaced by a more comprehensive version in 072_transaction_integrity_checks.sql.
-- Skipping redundant definition here to avoid return type mismatch if the enhanced version already exists.

-- [3] BALANCE SAFETY TRIGGER (Audit Point 573)
-- Final layer of defense: Prevent negative balance at the DB level regardless of application logic.
CREATE OR REPLACE FUNCTION check_balance_non_negative()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.balance < 0 THEN
        RAISE EXCEPTION 'Database Integrity Constraint: Balance cannot be negative for tenant %', NEW.id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_check_tenants_balance ON tenants;
CREATE TRIGGER trg_check_tenants_balance
BEFORE UPDATE ON tenants
FOR EACH ROW
EXECUTE FUNCTION check_balance_non_negative();
