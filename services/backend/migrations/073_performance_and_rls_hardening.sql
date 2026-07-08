-- Phase 2: Project Hardening - Performance and Security (Corrected)
-- Date: 2026-03-22
-- Includes: Critical Indexes, Idempotency Unique Constraints, RLS Completion

BEGIN;

-- 1. Performance: Dispatcher Optimized Index
-- Removed deleted_at as it doesn't exist in the messages table.
CREATE INDEX IF NOT EXISTS idx_messages_status_scheduled_for_dispatcher 
ON messages (status, scheduled_at) 
WHERE status = 'PENDING';

-- 2. Performance: Accounting Lookup
CREATE INDEX IF NOT EXISTS idx_accounting_transactions_tenant_message_lookup
ON accounting_transactions (tenant_id, message_id);

-- 3. Security: Idempotency Key Scoping
-- Make idempotency keys unique per tenant.
ALTER TABLE messages DROP CONSTRAINT IF EXISTS messages_idempotency_key_key;
ALTER TABLE messages 
ADD CONSTRAINT messages_tenant_idempotency_unique 
UNIQUE (tenant_id, idempotency_key);

-- 4. Security: RLS Completion on existing tenant-scoped tables
-- staff_profiles and staff_activities already have tenant_id.

DO $$
DECLARE
    t_name TEXT;
    target_tables TEXT[] := ARRAY[
        'staff_activities', 
        'staff_profiles'
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

-- 5. Security: RLS for ticket_replies (via join with support_tickets)
-- Since ticket_replies doesn't have tenant_id directly.
ALTER TABLE ticket_replies ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_policy ON ticket_replies;
CREATE POLICY tenant_isolation_policy ON ticket_replies USING (
    EXISTS (
        SELECT 1 FROM support_tickets st 
        WHERE st.id = ticket_replies.ticket_id 
        AND (st.tenant_id = current_setting('app.current_tenant', true)::uuid 
             OR current_setting('app.is_super_admin', true) = 'true')
    )
);

-- 6. Performance: Balance History Recently Used Index
CREATE INDEX IF NOT EXISTS idx_balance_history_tenant_recent 
ON balance_history (tenant_id, created_at DESC);

COMMIT;
