-- Migration: Add Subscriptions Table
-- Version: 1.0
-- Date: 2026-02-20
-- Description: Creates subscriptions table for customer subscriptions to monthly packages with tenant isolation

-- This migration creates the subscriptions table that tracks:
-- - Customer subscriptions to monthly packages
-- - Message usage tracking (used, remaining)
-- - Overage tracking (messages, cost)
-- - Renewal settings (auto_renew, next_renewal_date)
-- - Status tracking (active, grace_period, suspended, cancelled)

BEGIN;

-- 1. Create subscriptions table
CREATE TABLE IF NOT EXISTS subscriptions (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,  -- CRITICAL: Required for tenant isolation
    package_id INTEGER NOT NULL,  -- References packages(id)
    
    -- فترة الاشتراك (Subscription Period)
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    
    -- الاستخدام (Usage Tracking)
    messages_used INTEGER DEFAULT 0,
    messages_remaining INTEGER,
    overage_messages INTEGER DEFAULT 0,
    overage_cost DECIMAL(10, 2) DEFAULT 0.00,
    
    -- الحالة (Status)
    status VARCHAR(20) DEFAULT 'active', -- 'active', 'grace_period', 'suspended', 'cancelled', 'expired'
    auto_renew BOOLEAN DEFAULT true,
    grace_period_days INTEGER DEFAULT 3,
    
    -- التجديد (Renewal)
    last_renewed_at TIMESTAMP,
    next_renewal_date DATE,
    renewal_attempts INTEGER DEFAULT 0,
    last_renewal_attempt TIMESTAMP,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT check_messages_used_non_negative CHECK (messages_used >= 0),
    CONSTRAINT check_overage_non_negative CHECK (overage_messages >= 0),
    CONSTRAINT check_overage_cost_non_negative CHECK (overage_cost >= 0),
    CONSTRAINT check_grace_period_positive CHECK (grace_period_days > 0),
    CONSTRAINT check_status_valid CHECK (status IN ('active', 'grace_period', 'suspended', 'cancelled', 'expired')),
    CONSTRAINT check_dates_valid CHECK (end_date >= start_date)
);

-- 2. Create indexes for performance and isolation
-- All indexes start with tenant_id for optimal performance
CREATE INDEX idx_subscriptions_tenant ON subscriptions(tenant_id);
CREATE INDEX idx_subscriptions_tenant_package ON subscriptions(tenant_id, package_id);
CREATE INDEX idx_subscriptions_tenant_status ON subscriptions(tenant_id, status);
CREATE INDEX idx_subscriptions_tenant_renewal ON subscriptions(tenant_id, next_renewal_date);
CREATE INDEX idx_subscriptions_tenant_dates ON subscriptions(tenant_id, start_date, end_date);
CREATE INDEX idx_subscriptions_tenant_auto_renew ON subscriptions(tenant_id, auto_renew);

-- Index for finding subscriptions that need renewal (used by auto-renewal cron job)
CREATE INDEX idx_subscriptions_renewal_due ON subscriptions(next_renewal_date, status, auto_renew)
WHERE status IN ('active', 'grace_period') AND auto_renew = true;

-- 3. Enable Row Level Security (RLS)
ALTER TABLE subscriptions ENABLE ROW LEVEL SECURITY;

-- 4. Create RLS Policies for tenant isolation

-- Policy for SELECT: Users can only see their tenant's subscriptions
CREATE POLICY tenant_isolation_select_policy ON subscriptions
    FOR SELECT
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy for INSERT: Users can only insert subscriptions for their tenant
CREATE POLICY tenant_isolation_insert_policy ON subscriptions
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy for UPDATE: Users can only update their tenant's subscriptions
CREATE POLICY tenant_isolation_update_policy ON subscriptions
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant')::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy for DELETE: Users can only delete their tenant's subscriptions
CREATE POLICY tenant_isolation_delete_policy ON subscriptions
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- 5. Create trigger for updated_at
CREATE OR REPLACE FUNCTION update_subscriptions_updated_at()
RETURNS TRIGGER AS $
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_subscriptions_updated_at
    BEFORE UPDATE ON subscriptions
    FOR EACH ROW
    EXECUTE FUNCTION update_subscriptions_updated_at();

-- 6. Create helper functions

-- Function to calculate messages_remaining based on package
CREATE OR REPLACE FUNCTION calculate_messages_remaining(
    p_package_id INTEGER,
    p_messages_used INTEGER
)
RETURNS INTEGER AS $
DECLARE
    v_included_messages INTEGER;
    v_is_unlimited BOOLEAN;
BEGIN
    -- Get package details
    SELECT included_messages, is_unlimited
    INTO v_included_messages, v_is_unlimited
    FROM packages
    WHERE id = p_package_id;
    
    -- If unlimited package, return NULL (unlimited)
    IF v_is_unlimited THEN
        RETURN NULL;
    END IF;
    
    -- Calculate remaining messages
    RETURN GREATEST(0, v_included_messages - p_messages_used);
END;
$ LANGUAGE plpgsql;

-- Function to check if subscription is active
CREATE OR REPLACE FUNCTION is_subscription_active(p_subscription_id INTEGER)
RETURNS BOOLEAN AS $
DECLARE
    v_status VARCHAR(20);
    v_end_date DATE;
BEGIN
    SELECT status, end_date
    INTO v_status, v_end_date
    FROM subscriptions
    WHERE id = p_subscription_id;
    
    RETURN v_status = 'active' AND v_end_date >= CURRENT_DATE;
END;
$ LANGUAGE plpgsql;

-- Function to get active subscription for a tenant
CREATE OR REPLACE FUNCTION get_active_subscription(p_tenant_id UUID)
RETURNS INTEGER AS $
DECLARE
    v_subscription_id INTEGER;
BEGIN
    SELECT id
    INTO v_subscription_id
    FROM subscriptions
    WHERE tenant_id = p_tenant_id
      AND status = 'active'
      AND end_date >= CURRENT_DATE
    ORDER BY created_at DESC
    LIMIT 1;
    
    RETURN v_subscription_id;
END;
$ LANGUAGE plpgsql;

-- 7. Comments for documentation
COMMENT ON TABLE subscriptions IS 'Stores customer subscriptions to monthly packages with usage tracking and renewal management';
COMMENT ON COLUMN subscriptions.tenant_id IS 'Tenant ID for multi-tenant isolation - CRITICAL for RLS';
COMMENT ON COLUMN subscriptions.package_id IS 'References the subscribed package';
COMMENT ON COLUMN subscriptions.start_date IS 'Subscription start date';
COMMENT ON COLUMN subscriptions.end_date IS 'Subscription end date';
COMMENT ON COLUMN subscriptions.messages_used IS 'Number of messages used from the included quota';
COMMENT ON COLUMN subscriptions.messages_remaining IS 'Number of messages remaining in the included quota (NULL = unlimited)';
COMMENT ON COLUMN subscriptions.overage_messages IS 'Number of messages sent beyond the included quota';
COMMENT ON COLUMN subscriptions.overage_cost IS 'Total cost of overage messages';
COMMENT ON COLUMN subscriptions.status IS 'Subscription status: active, grace_period, suspended, cancelled, expired';
COMMENT ON COLUMN subscriptions.auto_renew IS 'Whether the subscription should auto-renew at the end of the period';
COMMENT ON COLUMN subscriptions.grace_period_days IS 'Number of days grace period after failed renewal';
COMMENT ON COLUMN subscriptions.last_renewed_at IS 'Timestamp of last successful renewal';
COMMENT ON COLUMN subscriptions.next_renewal_date IS 'Date when the subscription should be renewed';
COMMENT ON COLUMN subscriptions.renewal_attempts IS 'Number of renewal attempts (for tracking failed renewals)';
COMMENT ON COLUMN subscriptions.last_renewal_attempt IS 'Timestamp of last renewal attempt';

COMMIT;

-- Usage Instructions:
-- To create a new subscription:
-- INSERT INTO subscriptions (tenant_id, package_id, start_date, end_date, messages_remaining, next_renewal_date)
-- SELECT 
--     '<tenant-uuid>'::uuid,
--     p.id,
--     CURRENT_DATE,
--     CURRENT_DATE + INTERVAL '1 month',
--     p.included_messages,
--     CURRENT_DATE + INTERVAL '1 month'
-- FROM packages p
-- WHERE p.code = 'PKG_BASIC' AND p.tenant_id = '<tenant-uuid>'::uuid;

-- Rollback script (if needed):
-- BEGIN;
-- DROP TRIGGER IF EXISTS trigger_subscriptions_updated_at ON subscriptions;
-- DROP FUNCTION IF EXISTS update_subscriptions_updated_at();
-- DROP FUNCTION IF EXISTS calculate_messages_remaining(INTEGER, INTEGER);
-- DROP FUNCTION IF EXISTS is_subscription_active(INTEGER);
-- DROP FUNCTION IF EXISTS get_active_subscription(UUID);
-- DROP TABLE IF EXISTS subscriptions CASCADE;
-- COMMIT;
