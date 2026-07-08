-- Migration: 042_timing_violations_table.sql
-- Description: Create timing_violations table for tracking inter-message timing violations
-- This table records when messages are sent too quickly, violating minimum gap requirements

-- Create timing_violations table
CREATE TABLE IF NOT EXISTS timing_violations (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    message_id VARCHAR(255) NOT NULL,
    message_type VARCHAR(10) NOT NULL CHECK (message_type IN ('signal', 'push')),
    
    -- Timing information
    expected_gap_ms INTEGER NOT NULL CHECK (expected_gap_ms > 0),
    actual_gap_ms INTEGER NOT NULL CHECK (actual_gap_ms >= 0),
    violation_severity VARCHAR(20) NOT NULL CHECK (violation_severity IN ('minor', 'moderate', 'severe')),
    
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX idx_timing_violations_tenant ON timing_violations(tenant_id, created_at DESC);
CREATE INDEX idx_timing_violations_severity ON timing_violations(violation_severity, created_at DESC);
CREATE INDEX idx_timing_violations_message_type ON timing_violations(tenant_id, message_type, created_at DESC);

-- Enable Row Level Security for multi-tenancy
ALTER TABLE timing_violations ENABLE ROW LEVEL SECURITY;

-- Create RLS policy for tenant isolation
CREATE POLICY tenant_isolation_policy ON timing_violations
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- Create policy for service role (bypass RLS)
CREATE POLICY service_role_policy ON timing_violations
    USING (current_setting('app.bypass_rls', true)::text = 'on')
    WITH CHECK (current_setting('app.bypass_rls', true)::text = 'on');

-- Add comments for documentation
COMMENT ON TABLE timing_violations IS 'Records timing violations when messages are sent too quickly';
COMMENT ON COLUMN timing_violations.id IS 'Unique identifier for the violation record';
COMMENT ON COLUMN timing_violations.tenant_id IS 'Reference to the tenant (company) that caused the violation';
COMMENT ON COLUMN timing_violations.message_id IS 'Identifier of the message that violated timing constraints';
COMMENT ON COLUMN timing_violations.message_type IS 'Type of message: signal or push';
COMMENT ON COLUMN timing_violations.expected_gap_ms IS 'Expected minimum gap in milliseconds between messages';
COMMENT ON COLUMN timing_violations.actual_gap_ms IS 'Actual gap in milliseconds that was observed';
COMMENT ON COLUMN timing_violations.violation_severity IS 'Severity level: minor, moderate, or severe';
COMMENT ON COLUMN timing_violations.created_at IS 'Timestamp when violation was recorded';
