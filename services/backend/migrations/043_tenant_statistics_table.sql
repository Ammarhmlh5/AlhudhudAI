-- Migration: 043_tenant_statistics_table.sql
-- Description: Create tenant_statistics table for statistical analysis and anomaly detection
-- This table stores statistical profiles and anomaly detection results for each tenant

-- Create tenant_statistics table
CREATE TABLE IF NOT EXISTS tenant_statistics (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    message_type VARCHAR(10) NOT NULL CHECK (message_type IN ('signal', 'push')),
    
    -- Current metrics
    current_mpm DECIMAL(10,2) NOT NULL DEFAULT 0 CHECK (current_mpm >= 0),
    current_stddev DECIMAL(10,2) NOT NULL DEFAULT 0 CHECK (current_stddev >= 0),
    
    -- Historical baselines (daily)
    daily_avg_mpm DECIMAL(10,2) NOT NULL DEFAULT 0 CHECK (daily_avg_mpm >= 0),
    daily_peak_mpm DECIMAL(10,2) NOT NULL DEFAULT 0 CHECK (daily_peak_mpm >= 0),
    
    -- Historical baselines (weekly)
    weekly_avg_mpm DECIMAL(10,2) NOT NULL DEFAULT 0 CHECK (weekly_avg_mpm >= 0),
    weekly_peak_mpm DECIMAL(10,2) NOT NULL DEFAULT 0 CHECK (weekly_peak_mpm >= 0),
    
    -- Anomaly detection results
    anomaly_score DECIMAL(5,2) NOT NULL DEFAULT 0,
    anomaly_level INTEGER NOT NULL DEFAULT 0 CHECK (anomaly_level >= 0 AND anomaly_level <= 3),
    is_anomaly BOOLEAN NOT NULL DEFAULT FALSE,
    anomaly_reason TEXT,
    
    -- Timestamps
    last_updated TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Unique constraint: one record per tenant per message type
    CONSTRAINT unique_tenant_message_type UNIQUE (tenant_id, message_type)
);

-- Create indexes for performance
CREATE INDEX idx_tenant_statistics_tenant ON tenant_statistics(tenant_id);
CREATE INDEX idx_tenant_statistics_anomaly ON tenant_statistics(is_anomaly, anomaly_level) WHERE is_anomaly = TRUE;
CREATE INDEX idx_tenant_statistics_updated ON tenant_statistics(last_updated DESC);

-- Enable Row Level Security for multi-tenancy
ALTER TABLE tenant_statistics ENABLE ROW LEVEL SECURITY;

-- Create RLS policy for tenant isolation
CREATE POLICY tenant_isolation_policy ON tenant_statistics
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- Create policy for service role (bypass RLS)
CREATE POLICY service_role_policy ON tenant_statistics
    USING (current_setting('app.bypass_rls', true)::text = 'on')
    WITH CHECK (current_setting('app.bypass_rls', true)::text = 'on');

-- Add comments for documentation
COMMENT ON TABLE tenant_statistics IS 'Statistical profiles and anomaly detection for tenant messaging patterns';
COMMENT ON COLUMN tenant_statistics.id IS 'Unique identifier for the statistics record';
COMMENT ON COLUMN tenant_statistics.tenant_id IS 'Reference to the tenant (company)';
COMMENT ON COLUMN tenant_statistics.message_type IS 'Type of message: signal or push';
COMMENT ON COLUMN tenant_statistics.current_mpm IS 'Current messages per minute rate';
COMMENT ON COLUMN tenant_statistics.current_stddev IS 'Current standard deviation of message rate';
COMMENT ON COLUMN tenant_statistics.daily_avg_mpm IS 'Average messages per minute over last 24 hours';
COMMENT ON COLUMN tenant_statistics.daily_peak_mpm IS 'Peak messages per minute over last 24 hours';
COMMENT ON COLUMN tenant_statistics.weekly_avg_mpm IS 'Average messages per minute over last 7 days';
COMMENT ON COLUMN tenant_statistics.weekly_peak_mpm IS 'Peak messages per minute over last 7 days';
COMMENT ON COLUMN tenant_statistics.anomaly_score IS 'Calculated anomaly score (Z-score or similar)';
COMMENT ON COLUMN tenant_statistics.anomaly_level IS 'Anomaly severity: 0=none, 1=minor, 2=moderate, 3=severe';
COMMENT ON COLUMN tenant_statistics.is_anomaly IS 'Whether current behavior is considered anomalous';
COMMENT ON COLUMN tenant_statistics.anomaly_reason IS 'Human-readable explanation of the anomaly';
COMMENT ON COLUMN tenant_statistics.last_updated IS 'Timestamp when statistics were last updated';
