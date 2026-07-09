-- Migration: 038_server_health_logs_table.sql
-- Description: Create server_health_logs table for tracking dedicated server health checks
-- Requirements: 9.5
-- Feature: admin-interface-enhancements

-- Create server_health_logs table
CREATE TABLE IF NOT EXISTS server_health_logs (
    id BIGSERIAL PRIMARY KEY,
    server_id UUID NOT NULL REFERENCES dedicated_gateways(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Health check results
    check_timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
    response_time_ms INTEGER CHECK (response_time_ms >= 0),
    status_code INTEGER,
    is_available BOOLEAN NOT NULL,
    
    -- Error information
    error_message TEXT,
    error_count INTEGER NOT NULL DEFAULT 0 CHECK (error_count >= 0),
    error_rate DECIMAL(5,2) CHECK (error_rate >= 0 AND error_rate <= 100),
    
    -- Status classification
    health_status VARCHAR(20) NOT NULL CHECK (health_status IN ('healthy', 'degraded', 'unhealthy')),
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Ensure error_message is provided when not available
    CONSTRAINT check_error_info CHECK (
        is_available = true OR 
        (is_available = false AND error_message IS NOT NULL)
    )
);

-- Create indexes for performance
CREATE INDEX idx_server_health_logs_server ON server_health_logs(server_id, check_timestamp DESC);
CREATE INDEX idx_server_health_logs_tenant ON server_health_logs(tenant_id, check_timestamp DESC);
CREATE INDEX idx_server_health_logs_status ON server_health_logs(health_status);
CREATE INDEX idx_server_health_logs_timestamp ON server_health_logs(check_timestamp DESC);
CREATE INDEX idx_server_health_logs_available ON server_health_logs(is_available, check_timestamp DESC);

-- Create index for recent health checks (last 30 days)
CREATE INDEX idx_server_health_logs_recent ON server_health_logs(server_id, check_timestamp DESC) 
    WHERE check_timestamp > NOW() - INTERVAL '30 days';

-- Enable Row Level Security for tenant isolation
ALTER TABLE server_health_logs ENABLE ROW LEVEL SECURITY;

-- Create RLS policy for tenant isolation
CREATE POLICY tenant_isolation_policy ON server_health_logs
    USING (tenant_id = current_setting('app.current_tenant', true)::uuid);

-- Create policy for admins to view all logs
CREATE POLICY admin_view_all_policy ON server_health_logs
    FOR SELECT
    USING (current_setting('app.user_role', true) = 'admin');

-- Add comments for documentation
COMMENT ON TABLE server_health_logs IS 'Stores health check results for dedicated servers';
COMMENT ON COLUMN server_health_logs.response_time_ms IS 'Server response time in milliseconds';
COMMENT ON COLUMN server_health_logs.status_code IS 'HTTP status code returned by health check endpoint';
COMMENT ON COLUMN server_health_logs.is_available IS 'Whether the server was available during this check';
COMMENT ON COLUMN server_health_logs.error_rate IS 'Error rate percentage at time of check';
COMMENT ON COLUMN server_health_logs.health_status IS 'Classified health status: healthy (<200ms, <1% errors), degraded (200-500ms or 1-5% errors), unhealthy (>500ms or >5% errors)';

-- Create function to automatically cleanup old logs (older than 30 days)
CREATE OR REPLACE FUNCTION cleanup_old_server_health_logs()
RETURNS void AS $$
BEGIN
    DELETE FROM server_health_logs
    WHERE check_timestamp < NOW() - INTERVAL '30 days';
END;
$$ LANGUAGE plpgsql;

-- Add comment for cleanup function
COMMENT ON FUNCTION cleanup_old_server_health_logs() IS 'Deletes server health logs older than 30 days to maintain retention policy';
