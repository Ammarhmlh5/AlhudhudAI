-- Migration 066: Compliance and Consistency Fixes
-- Description: Adds updated_at triggers to roles/permissions and missing NOT NULLs.

BEGIN;

-- 1. Add UpdatedAt column if missing (already added in models.go but just in case)
ALTER TABLE permissions ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();
ALTER TABLE roles ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();

-- 2. Add triggers for automatic updated_at
-- This uses the function created in Migration 061
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_tables WHERE tablename = 'permissions') THEN
        IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_permissions_updated_at') THEN
            CREATE TRIGGER update_permissions_updated_at
            BEFORE UPDATE ON permissions
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
        END IF;
    END IF;

    IF EXISTS (SELECT 1 FROM pg_tables WHERE tablename = 'roles') THEN
        IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_roles_updated_at') THEN
            CREATE TRIGGER update_roles_updated_at
            BEFORE UPDATE ON roles
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
        END IF;
    END IF;
END $$;

-- 3. Hardening NOT NULLs as per Audit Issue #1
-- These were partially done, but ensuring all mentioned are covered.
ALTER TABLE messages ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE messages ALTER COLUMN phone_number SET NOT NULL;
ALTER TABLE messages ALTER COLUMN content SET NOT NULL;
ALTER TABLE messages ALTER COLUMN status SET NOT NULL;

ALTER TABLE dispatch_channels ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE dispatch_channels ALTER COLUMN type SET NOT NULL;
ALTER TABLE dispatch_channels ALTER COLUMN channel_key SET NOT NULL;

-- 4. Financial Precision Fix as per Audit Issue #2
-- Drop dependent views before altering type
DROP VIEW IF EXISTS v_messages_with_custom_ids;
DROP VIEW IF EXISTS orphaned_message_deductions;

ALTER TABLE messages ALTER COLUMN cost TYPE DECIMAL(12,4);
ALTER TABLE accounting_transactions ALTER COLUMN amount TYPE DECIMAL(12,4);

-- Recreate views
CREATE OR REPLACE VIEW v_messages_with_custom_ids AS
 SELECT m.custom_id AS message_id,
    m.event_custom_id AS event_id,
    m.tenant_id,
    m.phone_number,
    m.content,
    m.status,
    m.cost,
    m.created_at,
    m.updated_at,
    ge.custom_id AS event_custom_id_full,
    ge.sender_ip AS event_sender_ip,
    ge.created_at AS event_created_at
   FROM messages m
     LEFT JOIN global_incoming_events ge ON m.event_custom_id::text = ge.custom_id::text
  WHERE m.custom_id IS NOT NULL;

CREATE OR REPLACE VIEW orphaned_message_deductions AS
 SELECT m.id AS message_id,
    m.tenant_id,
    m.status,
    m.cost,
    m.created_at
   FROM messages m
     LEFT JOIN accounting_transactions at ON m.id = at.message_id
  WHERE (m.status <> ALL (ARRAY['PENDING'::text, 'QUEUED'::text, 'WAITING'::text])) AND at.id IS NULL AND m.created_at < (now() - '00:05:00'::interval);

-- 5. RLS for Invoices (already enabled in 053/051, but ensuring it)
ALTER TABLE billing_invoices ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_policy ON billing_invoices;
CREATE POLICY tenant_isolation_policy ON billing_invoices
USING (
    tenant_id = current_setting('app.current_tenant', true)::uuid 
    OR current_setting('app.is_super_admin', true) = 'true'
);

COMMIT;
