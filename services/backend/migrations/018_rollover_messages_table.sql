-- Migration: Add Rollover Messages Table
-- Version: 1.0
-- Date: 2026-02-20
-- Description: Creates rollover_messages table for tracking rolled-over messages from one subscription period to the next with tenant isolation

-- This migration creates the rollover_messages table that tracks:
-- - Messages rolled over from previous subscription periods
-- - Source subscription and period tracking
-- - Expiry dates for rolled-over messages
-- - Usage tracking (messages_used, messages_remaining)
-- - FIFO ordering (created_at for oldest-first usage)

BEGIN;

-- 1. Create rollover_messages table
CREATE TABLE IF NOT EXISTS rollover_messages (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,  -- CRITICAL: Required for tenant isolation
    subscription_id INTEGER NOT NULL,  -- References subscriptions(id)
    
    -- تفاصيل الترحيل (Rollover Details)
    messages_count INTEGER NOT NULL,
    source_month DATE NOT NULL,
    expiry_date DATE,
    
    -- الاستخدام (Usage Tracking)
    messages_used INTEGER DEFAULT 0,
    messages_remaining INTEGER NOT NULL,
    
    -- الحالة (Status)
    status VARCHAR(20) DEFAULT 'active', -- 'active', 'expired', 'consumed'
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT check_rollover_positive CHECK (messages_count > 0),
    CONSTRAINT check_rollover_remaining CHECK (messages_remaining >= 0),
    CONSTRAINT check_rollover_used_valid CHECK (messages_used >= 0 AND messages_used <= messages_count),
    CONSTRAINT check_rollover_status_valid CHECK (status IN ('active', 'expired', 'consumed'))
);

-- 2. Create indexes for performance and isolation
-- All indexes start with tenant_id for optimal performance
CREATE INDEX idx_rollover_tenant ON rollover_messages(tenant_id);
CREATE INDEX idx_rollover_tenant_subscription ON rollover_messages(tenant_id, subscription_id);
CREATE INDEX idx_rollover_tenant_status ON rollover_messages(tenant_id, status);
CREATE INDEX idx_rollover_tenant_expiry ON rollover_messages(tenant_id, expiry_date);

-- Index for FIFO ordering (oldest first)
CREATE INDEX idx_rollover_tenant_created ON rollover_messages(tenant_id, subscription_id, created_at)
WHERE status = 'active';

-- Index for finding expired rollover messages (used by cleanup cron job)
CREATE INDEX idx_rollover_expired ON rollover_messages(expiry_date, status)
WHERE status = 'active' AND expiry_date IS NOT NULL;

-- 3. Enable Row Level Security (RLS)
ALTER TABLE rollover_messages ENABLE ROW LEVEL SECURITY;

-- 4. Create RLS Policies for tenant isolation

-- Policy for SELECT: Users can only see their tenant's rollover messages
CREATE POLICY tenant_isolation_select_policy ON rollover_messages
    FOR SELECT
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy for INSERT: Users can only insert rollover messages for their tenant
CREATE POLICY tenant_isolation_insert_policy ON rollover_messages
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy for UPDATE: Users can only update their tenant's rollover messages
CREATE POLICY tenant_isolation_update_policy ON rollover_messages
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant')::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy for DELETE: Users can only delete their tenant's rollover messages
CREATE POLICY tenant_isolation_delete_policy ON rollover_messages
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- 5. Create trigger for updated_at
CREATE OR REPLACE FUNCTION update_rollover_messages_updated_at()
RETURNS TRIGGER AS $
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_rollover_messages_updated_at
    BEFORE UPDATE ON rollover_messages
    FOR EACH ROW
    EXECUTE FUNCTION update_rollover_messages_updated_at();

-- 6. Create helper functions

-- Function to get available rollover messages for a subscription (FIFO order)
CREATE OR REPLACE FUNCTION get_available_rollover_messages(
    p_tenant_id UUID,
    p_subscription_id INTEGER
)
RETURNS TABLE (
    id INTEGER,
    messages_remaining INTEGER,
    created_at TIMESTAMP
) AS $
BEGIN
    RETURN QUERY
    SELECT rm.id, rm.messages_remaining, rm.created_at
    FROM rollover_messages rm
    WHERE rm.tenant_id = p_tenant_id
      AND rm.subscription_id = p_subscription_id
      AND rm.status = 'active'
      AND rm.messages_remaining > 0
      AND (rm.expiry_date IS NULL OR rm.expiry_date >= CURRENT_DATE)
    ORDER BY rm.created_at ASC;  -- FIFO: oldest first
END;
$ LANGUAGE plpgsql;

-- Function to use rollover messages (FIFO)
CREATE OR REPLACE FUNCTION use_rollover_messages(
    p_tenant_id UUID,
    p_subscription_id INTEGER,
    p_messages_needed INTEGER
)
RETURNS INTEGER AS $
DECLARE
    v_rollover_record RECORD;
    v_messages_to_use INTEGER;
    v_total_used INTEGER := 0;
BEGIN
    -- Loop through available rollover messages in FIFO order
    FOR v_rollover_record IN 
        SELECT id, messages_remaining
        FROM rollover_messages
        WHERE tenant_id = p_tenant_id
          AND subscription_id = p_subscription_id
          AND status = 'active'
          AND messages_remaining > 0
          AND (expiry_date IS NULL OR expiry_date >= CURRENT_DATE)
        ORDER BY created_at ASC  -- FIFO: oldest first
        FOR UPDATE  -- Lock rows for update
    LOOP
        -- Calculate how many messages to use from this rollover record
        v_messages_to_use := LEAST(v_rollover_record.messages_remaining, p_messages_needed - v_total_used);
        
        -- Update the rollover record
        UPDATE rollover_messages
        SET messages_used = messages_used + v_messages_to_use,
            messages_remaining = messages_remaining - v_messages_to_use,
            status = CASE 
                WHEN messages_remaining - v_messages_to_use = 0 THEN 'consumed'
                ELSE 'active'
            END,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = v_rollover_record.id;
        
        -- Track total used
        v_total_used := v_total_used + v_messages_to_use;
        
        -- Exit if we've used enough messages
        EXIT WHEN v_total_used >= p_messages_needed;
    END LOOP;
    
    RETURN v_total_used;
END;
$ LANGUAGE plpgsql;

-- Function to get total available rollover messages for a subscription
CREATE OR REPLACE FUNCTION get_total_rollover_messages(
    p_tenant_id UUID,
    p_subscription_id INTEGER
)
RETURNS INTEGER AS $
DECLARE
    v_total INTEGER;
BEGIN
    SELECT COALESCE(SUM(messages_remaining), 0)
    INTO v_total
    FROM rollover_messages
    WHERE tenant_id = p_tenant_id
      AND subscription_id = p_subscription_id
      AND status = 'active'
      AND (expiry_date IS NULL OR expiry_date >= CURRENT_DATE);
    
    RETURN v_total;
END;
$ LANGUAGE plpgsql;

-- Function to expire old rollover messages
CREATE OR REPLACE FUNCTION expire_old_rollover_messages()
RETURNS INTEGER AS $
DECLARE
    v_expired_count INTEGER;
BEGIN
    -- Update expired rollover messages
    UPDATE rollover_messages
    SET status = 'expired',
        updated_at = CURRENT_TIMESTAMP
    WHERE status = 'active'
      AND expiry_date IS NOT NULL
      AND expiry_date < CURRENT_DATE;
    
    GET DIAGNOSTICS v_expired_count = ROW_COUNT;
    
    RETURN v_expired_count;
END;
$ LANGUAGE plpgsql;

-- Function to create rollover messages for a subscription
CREATE OR REPLACE FUNCTION create_rollover_messages(
    p_tenant_id UUID,
    p_subscription_id INTEGER,
    p_messages_count INTEGER,
    p_source_month DATE,
    p_expiry_months INTEGER DEFAULT NULL
)
RETURNS INTEGER AS $
DECLARE
    v_expiry_date DATE;
    v_rollover_id INTEGER;
BEGIN
    -- Calculate expiry date if expiry_months is provided
    IF p_expiry_months IS NOT NULL THEN
        v_expiry_date := p_source_month + (p_expiry_months || ' months')::INTERVAL;
    ELSE
        v_expiry_date := NULL;  -- No expiry
    END IF;
    
    -- Insert rollover record
    INSERT INTO rollover_messages (
        tenant_id,
        subscription_id,
        messages_count,
        source_month,
        expiry_date,
        messages_remaining,
        status
    ) VALUES (
        p_tenant_id,
        p_subscription_id,
        p_messages_count,
        p_source_month,
        v_expiry_date,
        p_messages_count,
        'active'
    )
    RETURNING id INTO v_rollover_id;
    
    RETURN v_rollover_id;
END;
$ LANGUAGE plpgsql;

-- 7. Comments for documentation
COMMENT ON TABLE rollover_messages IS 'Stores rolled-over messages from previous subscription periods with FIFO usage tracking';
COMMENT ON COLUMN rollover_messages.tenant_id IS 'Tenant ID for multi-tenant isolation - CRITICAL for RLS';
COMMENT ON COLUMN rollover_messages.subscription_id IS 'References the subscription that owns these rollover messages';
COMMENT ON COLUMN rollover_messages.messages_count IS 'Total number of messages rolled over from the source period';
COMMENT ON COLUMN rollover_messages.source_month IS 'The month/period from which these messages were rolled over';
COMMENT ON COLUMN rollover_messages.expiry_date IS 'Date when these rollover messages expire (NULL = never expire)';
COMMENT ON COLUMN rollover_messages.messages_used IS 'Number of messages used from this rollover batch';
COMMENT ON COLUMN rollover_messages.messages_remaining IS 'Number of messages remaining in this rollover batch';
COMMENT ON COLUMN rollover_messages.status IS 'Status: active (available), expired (past expiry_date), consumed (all used)';
COMMENT ON COLUMN rollover_messages.created_at IS 'Creation timestamp - used for FIFO ordering (oldest first)';

COMMIT;

-- Usage Instructions:
-- 
-- 1. Create rollover messages when subscription renews:
-- SELECT create_rollover_messages(
--     '<tenant-uuid>'::uuid,
--     <subscription-id>,
--     500,  -- messages to rollover
--     '2026-02-01'::date,  -- source month
--     3  -- expire after 3 months
-- );
--
-- 2. Get total available rollover messages:
-- SELECT get_total_rollover_messages('<tenant-uuid>'::uuid, <subscription-id>);
--
-- 3. Use rollover messages (FIFO):
-- SELECT use_rollover_messages('<tenant-uuid>'::uuid, <subscription-id>, 10);
--
-- 4. Expire old rollover messages (run daily via cron):
-- SELECT expire_old_rollover_messages();

-- Rollback script (if needed):
-- BEGIN;
-- DROP TRIGGER IF EXISTS trigger_rollover_messages_updated_at ON rollover_messages;
-- DROP FUNCTION IF EXISTS update_rollover_messages_updated_at();
-- DROP FUNCTION IF EXISTS get_available_rollover_messages(UUID, INTEGER);
-- DROP FUNCTION IF EXISTS use_rollover_messages(UUID, INTEGER, INTEGER);
-- DROP FUNCTION IF EXISTS get_total_rollover_messages(UUID, INTEGER);
-- DROP FUNCTION IF EXISTS expire_old_rollover_messages();
-- DROP FUNCTION IF EXISTS create_rollover_messages(UUID, INTEGER, INTEGER, DATE, INTEGER);
-- DROP TABLE IF EXISTS rollover_messages CASCADE;
-- COMMIT;
