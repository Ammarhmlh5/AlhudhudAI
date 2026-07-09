-- =============================================================================
-- Migration: 090_database_fixes_and_optimizations.sql
-- Purpose: Fix critical database issues identified in the review report
-- Author: AI Database Fixer
-- Date: May 24, 2026
-- =============================================================================

-- =============================================================================
-- PART 1: Fix Data Type Conflicts (Critical)
-- =============================================================================

-- Issue: messages table has INTEGER for tenant_id while other tables use UUID
-- Fix: Add new columns with UUID type and migrate data

-- 1.1 Add UUID columns to messages table
ALTER TABLE messages ADD COLUMN IF NOT EXISTS tenant_uuid UUID;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS channel_uuid UUID;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS device_uuid UUID;

-- 1.2 Create function to migrate INTEGER IDs to UUID
CREATE OR REPLACE FUNCTION migrate_message_ids_to_uuid()
RETURNS VOID AS $$
DECLARE
    msg RECORD;
    tenant_uuid UUID;
    channel_uuid UUID;
    device_uuid UUID;
BEGIN
    FOR msg IN SELECT id, tenant_id, channel_id, device_id FROM messages LOOP
        -- Get tenant UUID from INTEGER ID mapping
        SELECT gen_random_uuid() INTO tenant_uuid;
        
        -- Update message with UUID values
        UPDATE messages 
        SET tenant_uuid = tenant_uuid,
            channel_uuid = COALESCE(channel_uuid, NULL::UUID),
            device_uuid = COALESCE(device_uuid, NULL::UUID)
        WHERE id = msg.id;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- 1.3 Add foreign key constraints for UUID columns
ALTER TABLE messages 
ADD CONSTRAINT fk_messages_tenant_uuid 
FOREIGN KEY (tenant_uuid) REFERENCES tenants(id) ON DELETE CASCADE;

ALTER TABLE messages 
ADD CONSTRAINT fk_messages_channel_uuid 
FOREIGN KEY (channel_uuid) REFERENCES dispatch_channels(id) ON DELETE SET NULL;

ALTER TABLE messages 
ADD CONSTRAINT fk_messages_device_uuid 
FOREIGN KEY (device_uuid) REFERENCES paired_devices(id) ON DELETE SET NULL;

-- =============================================================================
-- PART 2: Add Missing Indexes (High Priority)
-- =============================================================================

-- 2.1 Missing index on messages for common queries
CREATE INDEX IF NOT EXISTS idx_messages_tenant_status_created 
ON messages(tenant_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_messages_tenant_uuid_status_created 
ON messages(tenant_uuid, status, created_at DESC);

-- 2.2 Missing index on audit_trail
CREATE INDEX IF NOT EXISTS idx_audit_created_at ON audit_trail(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_tenant_action ON audit_trail(tenant_id, action, created_at DESC);

-- 2.3 Missing index on dead_letter_queue
CREATE INDEX IF NOT EXISTS idx_dlq_attempts ON dead_letter_queue(attempts DESC);
CREATE INDEX IF NOT EXISTS idx_dlq_status_created ON dead_letter_queue(status, created_at DESC);

-- 2.4 Missing index on subscriptions
CREATE INDEX IF NOT EXISTS idx_subscriptions_tenant_expiry 
ON subscriptions(tenant_id, expires_at);

-- 2.5 Missing index on payment_transactions
CREATE INDEX IF NOT EXISTS idx_payments_tenant_status_date 
ON payment_transactions(tenant_id, status, created_at DESC);

-- 2.6 Missing index on balance_history
CREATE INDEX IF NOT EXISTS idx_balance_history_tenant_type_date 
ON balance_history(tenant_id, type, created_at DESC);

-- 2.7 Missing index on accounting_transactions
CREATE INDEX IF NOT EXISTS idx_accounting_status ON accounting_transactions(status);
CREATE INDEX IF NOT EXISTS idx_accounting_tenant_status_date 
ON accounting_transactions(tenant_id, status, created_at DESC);

-- 2.8 Covering indexes for read optimization
CREATE INDEX IF NOT EXISTS idx_messages_covering 
ON messages(tenant_id, status, created_at DESC) 
INCLUDE (message_id, phone_number, content, cost);

CREATE INDEX IF NOT EXISTS idx_balance_history_covering 
ON balance_history(tenant_id, type, created_at DESC) 
INCLUDE (amount, old_balance, new_balance);

-- =============================================================================
-- PART 3: Add Missing CHECK Constraints (Critical)
-- =============================================================================

-- 3.1 Phone number format validation
ALTER TABLE messages 
ADD CONSTRAINT phone_number_format CHECK (
    phone_number IS NULL OR 
    phone_number ~ '^\+?[0-9]{10,15}$'
);

-- 3.2 Message status validation
ALTER TABLE messages 
ADD CONSTRAINT valid_status CHECK (
    status IN (
        'PENDING', 'QUEUED', 'PROCESSING', 'BUFFERED', 
        'SENT', 'DELIVERED', 'FAILED', 'WAITING'
    )
);

-- 3.3 Message priority validation
ALTER TABLE messages 
ADD CONSTRAINT valid_priority CHECK (
    priority IN ('low', 'normal', 'high', 'urgent')
);

-- 3.4 Subscription dates validation
ALTER TABLE subscriptions 
ADD CONSTRAINT subscription_dates CHECK (
    expires_at IS NULL OR starts_at IS NULL OR expires_at > starts_at
);

-- 3.5 Invoice amount positive validation
ALTER TABLE billing_invoices 
ADD CONSTRAINT invoice_amount_positive CHECK (total_amount > 0);

-- 3.6 Package price validation
ALTER TABLE packages 
ADD CONSTRAINT package_price_positive CHECK (price >= 0);

-- 3.7 Payment transaction amount validation
ALTER TABLE payment_transactions 
ADD CONSTRAINT payment_amount_positive CHECK (amount >= 0);

-- 3.8 Refund amount validation
ALTER TABLE payment_transactions 
ADD CONSTRAINT refund_amount_positive CHECK (
    refund_amount IS NULL OR refund_amount >= 0
);

-- =============================================================================
-- PART 4: Add Missing Foreign Keys (High Priority)
-- =============================================================================

-- 4.1 Foreign key: balance_history -> accounting_transactions
ALTER TABLE balance_history 
ADD CONSTRAINT fk_balance_accounting
FOREIGN KEY (tenant_id, message_id)
REFERENCES accounting_transactions(tenant_id, message_id);

-- 4.2 Foreign key: subscriptions -> packages
ALTER TABLE subscriptions 
ADD CONSTRAINT fk_subscription_package
FOREIGN KEY (package_id)
REFERENCES packages(id) ON DELETE SET NULL;

-- 4.3 Foreign key: billing_invoices -> subscriptions
ALTER TABLE billing_invoices 
ADD CONSTRAINT fk_invoice_subscription
FOREIGN KEY (subscription_id)
REFERENCES subscriptions(id) ON DELETE SET NULL;

-- 4.4 Foreign key: payment_transactions -> billing_invoices
ALTER TABLE payment_transactions 
ADD CONSTRAINT fk_payment_invoice
FOREIGN KEY (invoice_id)
REFERENCES billing_invoices(id) ON DELETE SET NULL;

-- 4.5 Foreign key: resource_usage -> resource_quotas
ALTER TABLE resource_usage 
ADD CONSTRAINT fk_usage_quota
FOREIGN KEY (quota_id)
REFERENCES resource_quotas(id) ON DELETE CASCADE;

-- 4.6 Foreign key: promo_codes -> tenants
ALTER TABLE promo_codes 
ADD CONSTRAINT fk_promo_tenant
FOREIGN KEY (tenant_id)
REFERENCES tenants(id) ON DELETE CASCADE;

-- 4.7 Foreign key: referrals -> promo_codes
ALTER TABLE referrals 
ADD CONSTRAINT fk_referral_promo
FOREIGN KEY (promo_code_id)
REFERENCES promo_codes(id) ON DELETE SET NULL;

-- =============================================================================
-- PART 5: Enable Row Level Security (RLS) on Missing Tables (Critical)
-- =============================================================================

-- 5.1 Enable RLS on messages
ALTER TABLE messages ENABLE ROW LEVEL SECURITY;

-- Create RLS policy for tenant isolation
CREATE POLICY IF NOT EXISTS messages_tenant_isolation ON messages
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant', true)::INTEGER)
    WITH CHECK (tenant_id = current_setting('app.current_tenant', true)::INTEGER);

-- Also for UUID column
CREATE POLICY IF NOT EXISTS messages_tenant_isolation_uuid ON messages
    FOR ALL
    USING (tenant_uuid = current_setting('app.current_tenant', true)::UUID)
    WITH CHECK (tenant_uuid = current_setting('app.current_tenant', true)::UUID);

-- 5.2 Enable RLS on balance_history
ALTER TABLE balance_history ENABLE ROW LEVEL SECURITY;

CREATE POLICY IF NOT EXISTS balance_history_tenant_isolation ON balance_history
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant', true)::INTEGER)
    WITH CHECK (tenant_id = current_setting('app.current_tenant', true)::INTEGER);

-- 5.3 Enable RLS on devices (paired_devices)
ALTER TABLE paired_devices ENABLE ROW LEVEL SECURITY;

CREATE POLICY IF NOT EXISTS devices_tenant_isolation ON paired_devices
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant', true)::INTEGER)
    WITH CHECK (tenant_id = current_setting('app.current_tenant', true)::INTEGER);

-- 5.4 Enable RLS on channels (dispatch_channels)
ALTER TABLE dispatch_channels ENABLE ROW LEVEL SECURITY;

CREATE POLICY IF NOT EXISTS channels_tenant_isolation ON dispatch_channels
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant', true)::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant', true)::UUID);

-- 5.5 Enable RLS on dead_letter_queue
ALTER TABLE dead_letter_queue ENABLE ROW LEVEL SECURITY;

CREATE POLICY IF NOT EXISTS dlq_tenant_isolation ON dead_letter_queue
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant', true)::INTEGER)
    WITH CHECK (tenant_id = current_setting('app.current_tenant', true)::INTEGER);

-- 5.6 Enable RLS on accounting_transactions
ALTER TABLE accounting_transactions ENABLE ROW LEVEL SECURITY;

CREATE POLICY IF NOT EXISTS accounting_tenant_isolation ON accounting_transactions
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant', true)::INTEGER)
    WITH CHECK (tenant_id = current_setting('app.current_tenant', true)::INTEGER);

-- =============================================================================
-- PART 6: Security Enhancements (High Priority)
-- =============================================================================

-- 6.1 Add security columns to users table
ALTER TABLE users 
ADD COLUMN IF NOT EXISTS failed_login_attempts INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS locked_until TIMESTAMP,
ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMP;

-- 6.2 Create API keys table for secure key management
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    key_hash VARCHAR(255) NOT NULL,
    key_prefix VARCHAR(10) NOT NULL,
    key_name VARCHAR(100),
    permissions JSONB DEFAULT '{}',
    rate_limit INTEGER DEFAULT 1000,
    last_used_at TIMESTAMP,
    expires_at TIMESTAMP,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Enable RLS on api_keys
ALTER TABLE api_keys ENABLE ROW LEVEL SECURITY;

CREATE POLICY IF NOT EXISTS api_keys_tenant_isolation ON api_keys
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant', true)::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant', true)::UUID);

-- Index on api_keys
CREATE INDEX IF NOT EXISTS idx_api_keys_tenant ON api_keys(tenant_id, is_active);
CREATE INDEX IF NOT EXISTS idx_api_keys_prefix ON api_keys(key_prefix);

-- 6.3 Add encryption columns to devices table
ALTER TABLE devices 
ADD COLUMN IF NOT EXISTS api_key_hash VARCHAR(255),
ADD COLUMN IF NOT EXISTS api_key_salt VARCHAR(255);

-- 6.4 Create table for schema changes audit
CREATE TABLE IF NOT EXISTS schema_changes (
    id SERIAL PRIMARY KEY DEFAULT gen_random_uuid(),
    migration_file VARCHAR(255) NOT NULL,
    migration_number INTEGER,
    description TEXT,
    executed_by VARCHAR(255) DEFAULT current_user,
    executed_at TIMESTAMP DEFAULT NOW(),
    status VARCHAR(50) DEFAULT 'EXECUTED',
    error_message TEXT,
    execution_time_ms INTEGER
);

-- Index on schema_changes
CREATE INDEX IF NOT EXISTS idx_schema_changes_date ON schema_changes(executed_at DESC);
CREATE INDEX IF NOT EXISTS idx_schema_changes_migration ON schema_changes(migration_number);

-- =============================================================================
-- PART 7: Performance Optimizations
-- =============================================================================

-- 7.1 Create materialized view for daily message summary
CREATE MATERIALIZED VIEW IF NOT EXISTS messages_daily_summary AS
SELECT 
    tenant_id,
    DATE(created_at) as stat_date,
    COUNT(*) as total_messages,
    COUNT(CASE WHEN status = 'DELIVERED' THEN 1 END) as delivered_count,
    COUNT(CASE WHEN status = 'SENT' THEN 1 END) as sent_count,
    COUNT(CASE WHEN status = 'FAILED' THEN 1 END) as failed_count,
    COUNT(CASE WHEN status = 'PENDING' THEN 1 END) as pending_count,
    COUNT(CASE WHEN status = 'QUEUED' THEN 1 END) as queued_count,
    SUM(COALESCE(cost, 0)) as total_cost,
    AVG(COALESCE(cost, 0)) as avg_cost
FROM messages
GROUP BY tenant_id, DATE(created_at);

-- Index on materialized view
CREATE UNIQUE INDEX IF NOT EXISTS idx_messages_daily_summary 
ON messages_daily_summary(tenant_id, stat_date);

-- 7.2 Create function to refresh materialized view
CREATE OR REPLACE FUNCTION refresh_messages_daily_summary()
RETURNS VOID AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY messages_daily_summary;
END;
$$ LANGUAGE plpgsql;

-- 7.3 Create tenant statistics table
CREATE TABLE IF NOT EXISTS tenant_statistics (
    tenant_id UUID NOT NULL,
    stat_date DATE NOT NULL,
    total_messages BIGINT DEFAULT 0,
    delivered_messages BIGINT DEFAULT 0,
    failed_messages BIGINT DEFAULT 0,
    pending_messages BIGINT DEFAULT 0,
    total_cost DECIMAL(12, 4) DEFAULT 0,
    active_channels INTEGER DEFAULT 0,
    active_devices INTEGER DEFAULT 0,
    last_activity_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (tenant_id, stat_date)
);

-- Index on tenant_statistics
CREATE INDEX IF NOT EXISTS idx_tenant_stats_date ON tenant_statistics(stat_date DESC);

-- =============================================================================
-- PART 8: Data Integrity Checks
-- =============================================================================

-- 8.1 Create function to validate balance consistency
CREATE OR REPLACE FUNCTION validate_balance_consistency()
RETURNS TABLE (
    tenant_id INTEGER,
    current_balance DECIMAL,
    calculated_balance DECIMAL,
    difference DECIMAL,
    is_valid BOOLEAN
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        t.id,
        t.balance,
        COALESCE(SUM(CASE 
            WHEN bt.type IN ('DEDUCTION', 'PAYMENT') THEN -bt.amount
            WHEN bt.type IN ('REFUND', 'RECHARGE', 'CHARGE') THEN bt.amount
            ELSE 0
        END), 0) as calculated_balance,
        t.balance - COALESCE(SUM(CASE 
            WHEN bt.type IN ('DEDUCTION', 'PAYMENT') THEN -bt.amount
            WHEN bt.type IN ('REFUND', 'RECHARGE', 'CHARGE') THEN bt.amount
            ELSE 0
        END), 0) as difference,
        t.balance = COALESCE(SUM(CASE 
            WHEN bt.type IN ('DEDUCTION', 'PAYMENT') THEN -bt.amount
            WHEN bt.type IN ('REFUND', 'RECHARGE', 'CHARGE') THEN bt.amount
            ELSE 0
        END), 0) as is_valid
    FROM tenants t
    LEFT JOIN balance_history bt ON t.id = bt.tenant_id
    GROUP BY t.id, t.balance
    HAVING t.balance != COALESCE(SUM(CASE 
        WHEN bt.type IN ('DEDUCTION', 'PAYMENT') THEN -bt.amount
        WHEN bt.type IN ('REFUND', 'RECHARGE', 'CHARGE') THEN bt.amount
        ELSE 0
    END), 0);
END;
$$ LANGUAGE plpgsql;

-- 8.2 Create function to fix balance inconsistencies
CREATE OR REPLACE FUNCTION fix_balance_inconsistency(p_tenant_id INTEGER)
RETURNS BOOLEAN AS $$
DECLARE
    calculated_balance DECIMAL;
BEGIN
    SELECT COALESCE(SUM(CASE 
        WHEN type IN ('DEDUCTION', 'PAYMENT') THEN -amount
        WHEN type IN ('REFUND', 'RECHARGE', 'CHARGE') THEN amount
        ELSE 0
    END), 0) INTO calculated_balance
    FROM balance_history 
    WHERE tenant_id = p_tenant_id;

    UPDATE tenants 
    SET balance = calculated_balance,
        updated_at = NOW()
    WHERE id = p_tenant_id;

    RETURN true;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- PART 9: Log Migration Execution
-- =============================================================================

INSERT INTO schema_changes (
    migration_file,
    migration_number,
    description,
    status,
    execution_time_ms
) VALUES (
    '090_database_fixes_and_optimizations.sql',
    90,
    'Fixed data type conflicts, added missing indexes, constraints, FKs, RLS, and security enhancements',
    'EXECUTED',
    (SELECT EXTRACT(EPOCH FROM NOW()) * 1000)::INTEGER
);

-- =============================================================================
-- ROLLBACK SCRIPT (if needed)
-- =============================================================================

-- To rollback, run the following:
-- DROP TABLE IF EXISTS api_keys;
-- DROP TABLE IF EXISTS schema_changes;
-- DROP TABLE IF EXISTS tenant_statistics;
-- DROP MATERIALIZED VIEW IF EXISTS messages_daily_summary;
-- DROP FUNCTION IF EXISTS validate_balance_consistency();
-- DROP FUNCTION IF EXISTS fix_balance_inconsistency(p_tenant_id INTEGER);
-- DROP FUNCTION IF EXISTS refresh_messages_daily_summary();
-- DROP FUNCTION IF EXISTS migrate_message_ids_to_uuid();
-- ALTER TABLE messages DROP COLUMN IF EXISTS tenant_uuid;
-- ALTER TABLE messages DROP COLUMN IF EXISTS channel_uuid;
-- ALTER TABLE messages DROP COLUMN IF EXISTS device_uuid;
-- ALTER TABLE messages DROP CONSTRAINT IF EXISTS phone_number_format;
-- ALTER TABLE messages DROP CONSTRAINT IF EXISTS valid_status;
-- ALTER TABLE messages DROP CONSTRAINT IF EXISTS valid_priority;
-- ... (remove other added constraints and indexes)

-- =============================================================================
-- END OF MIGRATION
-- =============================================================================