-- Migration: 095_tenant_sms_quotas
-- Description: Create isolated table in the main database for tracking purchased SMS quotas.
-- This table is decoupled from the financial ledger and resets at the end of the month based on the subscription rules.

CREATE TABLE IF NOT EXISTS tenant_sms_quotas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- The actual SMS balance available for the current month
    available_sms BIGINT DEFAULT 0 NOT NULL CHECK (available_sms >= 0),
    
    -- Track total consumed for reporting purposes
    total_consumed_this_month BIGINT DEFAULT 0 NOT NULL,
    
    -- Tracking the expiration of the current quota
    quota_expires_at TIMESTAMPTZ,
    
    -- When the quota was last topped up or reset
    last_reset_at TIMESTAMPTZ DEFAULT NOW(),

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for fast lookup by tenant
CREATE INDEX IF NOT EXISTS idx_tenant_sms_quotas_tenant_id ON tenant_sms_quotas(tenant_id);

-- Trigger to auto-update 'updated_at'
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'set_tenant_sms_quotas_updated_at') THEN
        CREATE TRIGGER set_tenant_sms_quotas_updated_at
        BEFORE UPDATE ON tenant_sms_quotas
        FOR EACH ROW
        EXECUTE FUNCTION update_modified_column();
    END IF;
END $$;

-- Enable Row Level Security
ALTER TABLE tenant_sms_quotas ENABLE ROW LEVEL SECURITY;

-- Allow tenants to view only their own quotas
CREATE POLICY tenant_sms_quotas_select_policy ON tenant_sms_quotas
    FOR SELECT
    USING (tenant_id = auth.uid());

-- Allow system services to update quotas
-- In a microservices environment or server-side execution, a service role is used.
CREATE POLICY tenant_sms_quotas_service_role_policy ON tenant_sms_quotas
    FOR ALL
    USING (current_setting('request.jwt.claims', true)::json->>'role' = 'service_role');
