-- Migration 051: Security and Indexing Fixes (Corrected)
-- This script enables RLS on remaining tables and adds missing indexes.

BEGIN;

-- 1. Enable RLS on target tables
DO $$
DECLARE
    t text;
    tables_to_enable text[] := ARRAY[
        'accounting_keys',
        'paired_devices',
        'accounting_systems',
        'incoming_messages',
        'dispatch_channels',
        'messages',
        'dead_letter_queue',
        'audit_trail',
        'cross_tenant_access_logs',
        'message_batches',
        'message_refs',
        'test_operations',
        'payment_methods',
        'tenant_stats_summaries',
        'accounting_transactions',
        'tenant_audit_logs'
    ];
BEGIN
    FOREACH t IN ARRAY tables_to_enable LOOP
        -- Check if table exists before enabling RLS
        IF EXISTS (SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = t) THEN
            EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
            
            -- Dropping policy if exists to make it idempotent
            EXECUTE format('DROP POLICY IF EXISTS tenant_isolation_policy ON %I', t);
            
            -- Creating the policy
            -- Requirement: tenant_id isolation with super_admin bypass
            EXECUTE format('
                CREATE POLICY tenant_isolation_policy ON %I
                USING (
                    tenant_id = current_setting(''app.current_tenant'', true)::uuid 
                    OR current_setting(''app.is_super_admin'', true) = ''true''
                )', t);
        END IF;
    END LOOP;
END $$;

-- 2. Add Missing Index on tenant_id
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = 'accounting_keys') THEN
        IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_accounting_keys_tenant_id') THEN
            CREATE INDEX idx_accounting_keys_tenant_id ON accounting_keys(tenant_id);
        END IF;
    END IF;
END $$;

-- 3. Add Composite Indexes for Performance
-- لتحسين استعلامات الرسائل حسب الحالة والتاريخ
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_messages_status_created') THEN
        -- Removed deleted_at filter as column does not exist in standard models.go for Messages
        CREATE INDEX idx_messages_status_created 
        ON messages(status, created_at DESC);
    END IF;
END $$;

-- لتحسين استعلامات القنوات النشطة
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_dispatch_channels_tenant_status_priority') THEN
        CREATE INDEX idx_dispatch_channels_tenant_status_priority 
        ON dispatch_channels(tenant_id, status, priority DESC) 
        WHERE status = 'ONLINE';
    END IF;
END $$;

-- لتحسين استعلامات المعاملات المالية
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_accounting_transactions_tenant_status_created') THEN
        CREATE INDEX idx_accounting_transactions_tenant_status_created 
        ON accounting_transactions(tenant_id, status, created_at DESC);
    END IF;
END $$;

COMMIT;
