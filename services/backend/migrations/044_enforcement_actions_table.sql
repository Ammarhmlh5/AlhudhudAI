-- Migration: 044_enforcement_actions_table.sql
-- Description: Create enforcement_actions table for tracking automatic enforcement actions
-- This table records when the system automatically switches tenants to Push mode or takes other actions

-- Create enforcement_actions table
CREATE TABLE IF NOT EXISTS enforcement_actions (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Action information
    action_type VARCHAR(50) NOT NULL CHECK (action_type IN ('warning', 'force_push', 'restore_signal', 'throttle', 'block')),
    reason TEXT NOT NULL,
    anomaly_level INTEGER NOT NULL CHECK (anomaly_level >= 0 AND anomaly_level <= 3),
    
    -- Metrics at time of action
    violation_rate DECIMAL(5,2) CHECK (violation_rate >= 0 AND violation_rate <= 100),
    z_score DECIMAL(5,2),
    peak_ratio DECIMAL(5,2) CHECK (peak_ratio >= 0),
    
    -- Action details
    previous_mode VARCHAR(10) CHECK (previous_mode IN ('signal', 'push')),
    new_mode VARCHAR(10) CHECK (new_mode IN ('signal', 'push')),
    
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP,
    resolved_at TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX idx_enforcement_actions_tenant ON enforcement_actions(tenant_id, created_at DESC);
CREATE INDEX idx_enforcement_actions_type ON enforcement_actions(action_type, created_at DESC);
CREATE INDEX idx_enforcement_actions_active ON enforcement_actions(tenant_id, expires_at) WHERE expires_at IS NOT NULL AND resolved_at IS NULL;

-- Enable Row Level Security for multi-tenancy
ALTER TABLE enforcement_actions ENABLE ROW LEVEL SECURITY;

-- Create RLS policy for tenant isolation
CREATE POLICY tenant_isolation_policy ON enforcement_actions
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- Create policy for service role (bypass RLS)
CREATE POLICY service_role_policy ON enforcement_actions
    USING (current_setting('app.bypass_rls', true)::text = 'on')
    WITH CHECK (current_setting('app.bypass_rls', true)::text = 'on');

-- Add comments for documentation
COMMENT ON TABLE enforcement_actions IS 'Records automatic enforcement actions taken by the system';
COMMENT ON COLUMN enforcement_actions.id IS 'Unique identifier for the enforcement action';
COMMENT ON COLUMN enforcement_actions.tenant_id IS 'Reference to the tenant (company) affected by the action';
COMMENT ON COLUMN enforcement_actions.action_type IS 'Type of action: warning, force_push, restore_signal, throttle, or block';
COMMENT ON COLUMN enforcement_actions.reason IS 'Human-readable explanation of why the action was taken';
COMMENT ON COLUMN enforcement_actions.anomaly_level IS 'Anomaly severity that triggered the action: 0=none, 1=minor, 2=moderate, 3=severe';
COMMENT ON COLUMN enforcement_actions.violation_rate IS 'Percentage of messages violating timing constraints';
COMMENT ON COLUMN enforcement_actions.z_score IS 'Statistical Z-score at time of action';
COMMENT ON COLUMN enforcement_actions.peak_ratio IS 'Ratio of current rate to peak rate';
COMMENT ON COLUMN enforcement_actions.previous_mode IS 'Messaging mode before the action (signal or push)';
COMMENT ON COLUMN enforcement_actions.new_mode IS 'Messaging mode after the action (signal or push)';
COMMENT ON COLUMN enforcement_actions.created_at IS 'Timestamp when action was taken';
COMMENT ON COLUMN enforcement_actions.expires_at IS 'Timestamp when action expires (for temporary actions)';
COMMENT ON COLUMN enforcement_actions.resolved_at IS 'Timestamp when action was resolved or reverted';
