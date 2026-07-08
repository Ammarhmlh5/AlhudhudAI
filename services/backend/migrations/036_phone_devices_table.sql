-- Migration: 036_phone_devices_table.sql
-- Description: Create phone_devices table for managing devices used for message distribution
-- Requirements: 9.7
-- Feature: admin-interface-enhancements

-- Create phone_devices table
CREATE TABLE IF NOT EXISTS phone_devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Device information
    device_name VARCHAR(255) NOT NULL,
    device_model VARCHAR(255),
    os_version VARCHAR(50),
    app_version VARCHAR(50),
    
    -- Device configuration
    priority_level INTEGER NOT NULL DEFAULT 1 CHECK (priority_level >= 1 AND priority_level <= 5),
    max_concurrent_messages INTEGER NOT NULL DEFAULT 10 CHECK (max_concurrent_messages > 0),
    
    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'offline' CHECK (status IN ('online', 'offline', 'failed', 'excluded')),
    last_seen TIMESTAMP,
    last_failure TIMESTAMP,
    excluded_until TIMESTAMP,
    
    -- Metrics
    total_messages_sent BIGINT NOT NULL DEFAULT 0 CHECK (total_messages_sent >= 0),
    successful_messages BIGINT NOT NULL DEFAULT 0 CHECK (successful_messages >= 0),
    failed_messages BIGINT NOT NULL DEFAULT 0 CHECK (failed_messages >= 0),
    current_load INTEGER NOT NULL DEFAULT 0 CHECK (current_load >= 0),
    average_processing_time_ms INTEGER CHECK (average_processing_time_ms >= 0),
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Ensure successful + failed = total
    CONSTRAINT check_message_totals CHECK (successful_messages + failed_messages <= total_messages_sent),
    -- Ensure current_load doesn't exceed max
    CONSTRAINT check_load_limit CHECK (current_load <= max_concurrent_messages)
);

-- Create indexes for performance
CREATE INDEX idx_phone_devices_tenant ON phone_devices(tenant_id);
CREATE INDEX idx_phone_devices_status ON phone_devices(status);
CREATE INDEX idx_phone_devices_priority ON phone_devices(priority_level DESC);
CREATE INDEX idx_phone_devices_load ON phone_devices(current_load ASC);
CREATE INDEX idx_phone_devices_tenant_status ON phone_devices(tenant_id, status);
CREATE INDEX idx_phone_devices_excluded ON phone_devices(excluded_until) WHERE excluded_until IS NOT NULL;

-- Create index for available devices (online and not excluded)
CREATE INDEX idx_phone_devices_available ON phone_devices(tenant_id, priority_level DESC, current_load ASC) 
    WHERE status = 'online' AND (excluded_until IS NULL OR excluded_until < NOW());

-- Enable Row Level Security for tenant isolation
ALTER TABLE phone_devices ENABLE ROW LEVEL SECURITY;

-- Create RLS policy for tenant isolation
CREATE POLICY tenant_isolation_policy ON phone_devices
    USING (tenant_id = current_setting('app.current_tenant', true)::uuid);

-- Create policy for admins to view all devices
CREATE POLICY admin_view_all_policy ON phone_devices
    FOR SELECT
    USING (current_setting('app.user_role', true) = 'admin');

-- Create policy for admins to modify devices
CREATE POLICY admin_modify_policy ON phone_devices
    FOR ALL
    USING (current_setting('app.user_role', true) = 'admin');

-- Add comments for documentation
COMMENT ON TABLE phone_devices IS 'Stores information about phone devices used for message distribution';
COMMENT ON COLUMN phone_devices.priority_level IS 'Device priority level (1-5, where 5 is highest priority)';
COMMENT ON COLUMN phone_devices.max_concurrent_messages IS 'Maximum number of concurrent messages this device can handle';
COMMENT ON COLUMN phone_devices.status IS 'Device status: online, offline, failed, or excluded';
COMMENT ON COLUMN phone_devices.excluded_until IS 'Timestamp until which device is excluded from distribution (NULL if not excluded)';
COMMENT ON COLUMN phone_devices.current_load IS 'Current number of messages being processed by this device';
COMMENT ON COLUMN phone_devices.average_processing_time_ms IS 'Average message processing time in milliseconds';
