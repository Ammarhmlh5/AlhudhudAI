-- Migration: 045_tenant_timing_config_table.sql
-- Description: Create tenant_timing_config table for per-tenant timing configuration
-- This table stores timing constraints and enforcement thresholds for each tenant

-- Create tenant_timing_config table
CREATE TABLE IF NOT EXISTS tenant_timing_config (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Timing constraints (in milliseconds)
    signal_min_gap_ms INTEGER NOT NULL DEFAULT 2000 CHECK (signal_min_gap_ms > 0),
    push_min_gap_ms INTEGER NOT NULL DEFAULT 500 CHECK (push_min_gap_ms > 0),
    
    -- Enforcement thresholds
    violation_rate_threshold DECIMAL(5,2) NOT NULL DEFAULT 50.0 CHECK (violation_rate_threshold >= 0 AND violation_rate_threshold <= 100),
    z_score_threshold DECIMAL(5,2) NOT NULL DEFAULT 3.0 CHECK (z_score_threshold > 0),
    peak_ratio_threshold DECIMAL(5,2) NOT NULL DEFAULT 2.0 CHECK (peak_ratio_threshold > 0),
    anomaly_duration_minutes INTEGER NOT NULL DEFAULT 10 CHECK (anomaly_duration_minutes > 0),
    
    -- Current mode
    current_mode VARCHAR(10) NOT NULL DEFAULT 'signal' CHECK (current_mode IN ('signal', 'push')),
    forced_mode BOOLEAN NOT NULL DEFAULT FALSE,
    
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX idx_tenant_timing_config_tenant ON tenant_timing_config(tenant_id);
CREATE INDEX idx_tenant_timing_config_mode ON tenant_timing_config(current_mode);
CREATE INDEX idx_tenant_timing_config_forced ON tenant_timing_config(forced_mode) WHERE forced_mode = TRUE;

-- Enable Row Level Security for multi-tenancy
ALTER TABLE tenant_timing_config ENABLE ROW LEVEL SECURITY;

-- Create RLS policy for tenant isolation
CREATE POLICY tenant_isolation_policy ON tenant_timing_config
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- Create policy for service role (bypass RLS)
CREATE POLICY service_role_policy ON tenant_timing_config
    USING (current_setting('app.bypass_rls', true)::text = 'on')
    WITH CHECK (current_setting('app.bypass_rls', true)::text = 'on');

-- Add comments for documentation
COMMENT ON TABLE tenant_timing_config IS 'Per-tenant timing configuration and enforcement thresholds';
COMMENT ON COLUMN tenant_timing_config.id IS 'Unique identifier for the configuration record';
COMMENT ON COLUMN tenant_timing_config.tenant_id IS 'Reference to the tenant (company) - unique per tenant';
COMMENT ON COLUMN tenant_timing_config.signal_min_gap_ms IS 'Minimum gap in milliseconds between Signal messages (default: 2000ms)';
COMMENT ON COLUMN tenant_timing_config.push_min_gap_ms IS 'Minimum gap in milliseconds between Push messages (default: 500ms)';
COMMENT ON COLUMN tenant_timing_config.violation_rate_threshold IS 'Violation rate percentage that triggers enforcement (default: 50%)';
COMMENT ON COLUMN tenant_timing_config.z_score_threshold IS 'Z-score threshold for anomaly detection (default: 3.0)';
COMMENT ON COLUMN tenant_timing_config.peak_ratio_threshold IS 'Peak ratio threshold for anomaly detection (default: 2.0)';
COMMENT ON COLUMN tenant_timing_config.anomaly_duration_minutes IS 'Duration in minutes that anomaly must persist before enforcement (default: 10)';
COMMENT ON COLUMN tenant_timing_config.current_mode IS 'Current messaging mode: signal (direct) or push (batched)';
COMMENT ON COLUMN tenant_timing_config.forced_mode IS 'Whether mode was forced by system (true) or set by user (false)';
COMMENT ON COLUMN tenant_timing_config.created_at IS 'Timestamp when configuration was created';
COMMENT ON COLUMN tenant_timing_config.updated_at IS 'Timestamp when configuration was last updated';

-- Insert default configuration for existing tenants
INSERT INTO tenant_timing_config (tenant_id, signal_min_gap_ms, push_min_gap_ms, current_mode, forced_mode)
SELECT id, 2000, 500, 'signal', FALSE
FROM tenants
ON CONFLICT (tenant_id) DO NOTHING;
