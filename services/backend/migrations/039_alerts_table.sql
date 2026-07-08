-- Migration: 039_alerts_table.sql
-- Description: Create alerts table for managing system and company alerts
-- Requirements: 9.6
-- Feature: admin-interface-enhancements

-- Create alerts table
CREATE TABLE IF NOT EXISTS alerts (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE, -- NULL for system-wide alerts
    
    -- Alert information
    alert_type VARCHAR(50) NOT NULL CHECK (alert_type IN ('rating_change', 'server_failure', 'threshold_violation', 'device_failure', 'timing_violation', 'system_alert')),
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('critical', 'warning', 'info')),
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    
    -- Alert context (additional data in JSON format)
    context JSONB,
    
    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'acknowledged', 'resolved')),
    acknowledged_at TIMESTAMP,
    acknowledged_by UUID REFERENCES users(id) ON DELETE SET NULL,
    resolved_at TIMESTAMP,
    
    -- Deduplication (prevents duplicate alerts within 5 minutes)
    dedup_key VARCHAR(255),
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Ensure acknowledged_at is set when status is acknowledged
    CONSTRAINT check_acknowledged CHECK (
        status != 'acknowledged' OR 
        (status = 'acknowledged' AND acknowledged_at IS NOT NULL AND acknowledged_by IS NOT NULL)
    ),
    -- Ensure resolved_at is set when status is resolved
    CONSTRAINT check_resolved CHECK (
        status != 'resolved' OR 
        (status = 'resolved' AND resolved_at IS NOT NULL)
    ),
    -- Ensure resolved_at >= acknowledged_at if both are set
    CONSTRAINT check_resolution_order CHECK (
        acknowledged_at IS NULL OR resolved_at IS NULL OR resolved_at >= acknowledged_at
    )
);

-- Create indexes for performance
CREATE INDEX idx_alerts_tenant ON alerts(tenant_id, created_at DESC) WHERE tenant_id IS NOT NULL;
CREATE INDEX idx_alerts_type ON alerts(alert_type);
CREATE INDEX idx_alerts_severity ON alerts(severity);
CREATE INDEX idx_alerts_status ON alerts(status);
CREATE INDEX idx_alerts_dedup ON alerts(dedup_key, created_at) WHERE dedup_key IS NOT NULL;
CREATE INDEX idx_alerts_created ON alerts(created_at DESC);

-- Partial index for active alerts only (most frequently queried)
CREATE INDEX idx_alerts_active ON alerts(tenant_id, created_at DESC) WHERE status = 'active';

-- Partial index for critical alerts
CREATE INDEX idx_alerts_critical ON alerts(created_at DESC) WHERE severity = 'critical' AND status = 'active';

-- Partial index for system-wide alerts
CREATE INDEX idx_alerts_system ON alerts(created_at DESC) WHERE tenant_id IS NULL;

-- Enable Row Level Security for tenant isolation
ALTER TABLE alerts ENABLE ROW LEVEL SECURITY;

-- Create RLS policy for tenant isolation (users can only see their own alerts)
CREATE POLICY tenant_isolation_policy ON alerts
    FOR SELECT
    USING (
        tenant_id = current_setting('app.current_tenant', true)::uuid OR
        tenant_id IS NULL -- System-wide alerts visible to all
    );

-- Create policy for admins to view all alerts
CREATE POLICY admin_view_all_policy ON alerts
    FOR SELECT
    USING (current_setting('app.user_role', true) = 'admin');

-- Create policy for admins to modify alerts
CREATE POLICY admin_modify_policy ON alerts
    FOR ALL
    USING (current_setting('app.user_role', true) = 'admin');

-- Add comments for documentation
COMMENT ON TABLE alerts IS 'Stores system and company alerts with deduplication support';
COMMENT ON COLUMN alerts.tenant_id IS 'Company ID (NULL for system-wide alerts)';
COMMENT ON COLUMN alerts.alert_type IS 'Type of alert: rating_change, server_failure, threshold_violation, device_failure, timing_violation, system_alert';
COMMENT ON COLUMN alerts.severity IS 'Alert severity: critical, warning, or info';
COMMENT ON COLUMN alerts.context IS 'Additional context data in JSON format (e.g., threshold values, server details)';
COMMENT ON COLUMN alerts.dedup_key IS 'Deduplication key to prevent duplicate alerts within 5 minutes';
COMMENT ON COLUMN alerts.status IS 'Alert status: active, acknowledged, or resolved';

-- Create function to check for duplicate alerts
CREATE OR REPLACE FUNCTION check_duplicate_alert(
    p_dedup_key VARCHAR(255),
    p_minutes INTEGER DEFAULT 5
)
RETURNS BOOLEAN AS $$
DECLARE
    v_exists BOOLEAN;
BEGIN
    SELECT EXISTS(
        SELECT 1 FROM alerts
        WHERE dedup_key = p_dedup_key
        AND created_at > NOW() - (p_minutes || ' minutes')::INTERVAL
        AND status = 'active'
    ) INTO v_exists;
    
    RETURN v_exists;
END;
$$ LANGUAGE plpgsql;

-- Add comment for deduplication function
COMMENT ON FUNCTION check_duplicate_alert(VARCHAR, INTEGER) IS 'Checks if a duplicate alert exists within the specified time window (default 5 minutes)';

-- Example context structures:
-- rating_change: {"previous_category": "good", "new_category": "average", "rating_change": -10.5}
-- server_failure: {"server_id": "uuid", "server_url": "https://...", "error": "Connection timeout"}
-- threshold_violation: {"metric": "success_rate", "threshold": 90, "actual": 85.5}
-- device_failure: {"device_id": "uuid", "device_name": "Device 1", "error": "Connection lost"}
-- timing_violation: {"violation_count": 50, "violation_rate": 25.5, "z_score": 3.2}
