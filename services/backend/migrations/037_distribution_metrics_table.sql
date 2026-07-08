-- Migration: 037_distribution_metrics_table.sql
-- Description: Create distribution_metrics table for tracking per-device distribution statistics
-- Requirements: 9.8
-- Feature: admin-interface-enhancements

-- Create distribution_metrics table
CREATE TABLE IF NOT EXISTS distribution_metrics (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    device_id UUID NOT NULL REFERENCES phone_devices(id) ON DELETE CASCADE,
    
    -- Time period
    period_start TIMESTAMP NOT NULL,
    period_end TIMESTAMP NOT NULL,
    
    -- Distribution metrics
    messages_assigned BIGINT NOT NULL DEFAULT 0 CHECK (messages_assigned >= 0),
    messages_sent BIGINT NOT NULL DEFAULT 0 CHECK (messages_sent >= 0),
    messages_failed BIGINT NOT NULL DEFAULT 0 CHECK (messages_failed >= 0),
    
    -- Performance metrics
    average_processing_time_ms INTEGER CHECK (average_processing_time_ms >= 0),
    min_processing_time_ms INTEGER CHECK (min_processing_time_ms >= 0),
    max_processing_time_ms INTEGER CHECK (max_processing_time_ms >= 0),
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Ensure sent + failed <= assigned
    CONSTRAINT check_message_counts CHECK (messages_sent + messages_failed <= messages_assigned),
    -- Ensure period_end > period_start
    CONSTRAINT check_period_order CHECK (period_end > period_start),
    -- Ensure min <= avg <= max
    CONSTRAINT check_processing_times CHECK (
        (min_processing_time_ms IS NULL OR average_processing_time_ms IS NULL OR max_processing_time_ms IS NULL) OR
        (min_processing_time_ms <= average_processing_time_ms AND average_processing_time_ms <= max_processing_time_ms)
    )
);

-- Create indexes for performance
CREATE INDEX idx_distribution_metrics_tenant ON distribution_metrics(tenant_id, period_start DESC);
CREATE INDEX idx_distribution_metrics_device ON distribution_metrics(device_id, period_start DESC);
CREATE INDEX idx_distribution_metrics_period ON distribution_metrics(period_start DESC);

-- Create unique constraint to prevent duplicate metrics for same device and period
CREATE UNIQUE INDEX idx_distribution_metrics_unique ON distribution_metrics(tenant_id, device_id, period_start);

-- Enable Row Level Security for tenant isolation
ALTER TABLE distribution_metrics ENABLE ROW LEVEL SECURITY;

-- Create RLS policy for tenant isolation
CREATE POLICY tenant_isolation_policy ON distribution_metrics
    USING (tenant_id = current_setting('app.current_tenant', true)::uuid);

-- Create policy for admins to view all metrics
CREATE POLICY admin_view_all_policy ON distribution_metrics
    FOR SELECT
    USING (current_setting('app.user_role', true) = 'admin');

-- Add comments for documentation
COMMENT ON TABLE distribution_metrics IS 'Stores distribution statistics per device over time periods';
COMMENT ON COLUMN distribution_metrics.messages_assigned IS 'Number of messages assigned to this device';
COMMENT ON COLUMN distribution_metrics.messages_sent IS 'Number of messages successfully sent by this device';
COMMENT ON COLUMN distribution_metrics.messages_failed IS 'Number of messages that failed on this device';
COMMENT ON COLUMN distribution_metrics.average_processing_time_ms IS 'Average message processing time in milliseconds';
COMMENT ON COLUMN distribution_metrics.min_processing_time_ms IS 'Minimum message processing time in milliseconds';
COMMENT ON COLUMN distribution_metrics.max_processing_time_ms IS 'Maximum message processing time in milliseconds';
