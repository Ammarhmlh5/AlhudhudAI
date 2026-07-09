-- Phase 3: Final Hardening - Full RLS and Consistency Constraints (Corrected)
-- Date: 2026-03-22

BEGIN;

-- 1. Add tenant_id to linked tables for direct RLS enforcement (Requirement 513)
ALTER TABLE resource_quotas ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE resource_quotas rq
SET tenant_id = dg.tenant_id
FROM dedicated_gateways dg
WHERE rq.gateway_id = dg.id AND rq.tenant_id IS NULL;

ALTER TABLE ticket_replies ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE ticket_replies tr
SET tenant_id = st.tenant_id
FROM support_tickets st
WHERE tr.ticket_id = st.id AND tr.tenant_id IS NULL;

ALTER TABLE bank_accounts ADD COLUMN IF NOT EXISTS tenant_id UUID;
ALTER TABLE fraud_rules ADD COLUMN IF NOT EXISTS tenant_id UUID;
ALTER TABLE gateway_health_checks ADD COLUMN IF NOT EXISTS tenant_id UUID;
ALTER TABLE verification_codes ADD COLUMN IF NOT EXISTS tenant_id UUID;

-- 2. Complete RLS Rollout
DO $$
DECLARE
    t_name TEXT;
    target_tables TEXT[] := ARRAY[
        'resource_quotas', 
        'bank_accounts', 
        'fraud_rules', 
        'gateway_health_checks', 
        'ticket_replies',
        'verification_codes'
    ];
BEGIN
    FOREACH t_name IN ARRAY target_tables
    LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t_name);
        EXECUTE format('DROP POLICY IF EXISTS tenant_isolation_policy ON %I', t_name);
        EXECUTE format('CREATE POLICY tenant_isolation_policy ON %I USING (
            tenant_id = current_setting(''app.current_tenant'', true)::uuid 
            OR current_setting(''app.is_super_admin'', true) = ''true''
        )', t_name);
    END LOOP;
END $$;

-- 3. Financial Integrity: Non-negative Constraints (Requirement 4.4)
-- Corrected column names: price in catalog_packages, total_amount in billing_invoices.
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_balance_check;
ALTER TABLE tenants ADD CONSTRAINT tenants_balance_check CHECK (balance >= 0);

ALTER TABLE catalog_packages DROP CONSTRAINT IF EXISTS catalog_packages_price_check;
ALTER TABLE catalog_packages ADD CONSTRAINT catalog_packages_price_check CHECK (price >= 0);

ALTER TABLE billing_invoices DROP CONSTRAINT IF EXISTS billing_invoices_amount_check;
ALTER TABLE billing_invoices ADD CONSTRAINT billing_invoices_amount_check CHECK (total_amount >= 0);

-- 4. Unified RLS Policy for messages
ALTER TABLE messages ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_policy ON messages;
CREATE POLICY tenant_isolation_policy ON messages USING (
    tenant_id = current_setting('app.current_tenant', true)::uuid 
    OR current_setting('app.is_super_admin', true) = 'true'
);

COMMIT;
