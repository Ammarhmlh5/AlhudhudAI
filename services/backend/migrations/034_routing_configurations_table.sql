-- Migration: 034_routing_configurations_table.sql
-- Description: Create routing_configurations table for managing dedicated server routing
-- Requirements: 9.3
-- Feature: admin-interface-enhancements

-- Create routing_configurations table
CREATE TABLE IF NOT EXISTS routing_configurations (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Routing type
    routing_type VARCHAR(20) NOT NULL DEFAULT 'shared' CHECK (routing_type IN ('shared', 'dedicated')),
    
    -- Dedicated server configuration
    dedicated_server_id UUID REFERENCES dedicated_gateways(id) ON DELETE SET NULL,
    dedicated_server_url VARCHAR(500),
    dedicated_server_api_key VARCHAR(255),
    
    -- Routing metrics
    total_requests BIGINT NOT NULL DEFAULT 0 CHECK (total_requests >= 0),
    successful_requests BIGINT NOT NULL DEFAULT 0 CHECK (successful_requests >= 0),
    failed_requests BIGINT NOT NULL DEFAULT 0 CHECK (failed_requests >= 0),
    average_latency_ms INTEGER CHECK (average_latency_ms >= 0),
    
    -- Status
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_health_check TIMESTAMP,
    health_status VARCHAR(20) CHECK (health_status IN ('healthy', 'degraded', 'unhealthy')),
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Ensure successful + failed = total
    CONSTRAINT check_request_totals CHECK (successful_requests + failed_requests <= total_requests),
    -- Ensure dedicated server config is complete when routing_type is dedicated
    CONSTRAINT check_dedicated_config CHECK (
        routing_type = 'shared' OR 
        (routing_type = 'dedicated' AND dedicated_server_id IS NOT NULL AND dedicated_server_url IS NOT NULL)
    )
);

-- Create indexes for performance
CREATE INDEX idx_routing_config_tenant ON routing_configurations(tenant_id);
CREATE INDEX idx_routing_config_type ON routing_configurations(routing_type);
CREATE INDEX idx_routing_config_server ON routing_configurations(dedicated_server_id) WHERE dedicated_server_id IS NOT NULL;
CREATE INDEX idx_routing_config_health ON routing_configurations(health_status) WHERE health_status IS NOT NULL;
CREATE INDEX idx_routing_config_active ON routing_configurations(is_active) WHERE is_active = true;

-- Enable Row Level Security for tenant isolation
ALTER TABLE routing_configurations ENABLE ROW LEVEL SECURITY;

-- Create RLS policy for tenant isolation
CREATE POLICY tenant_isolation_policy ON routing_configurations
    USING (tenant_id = current_setting('app.current_tenant', true)::uuid);

-- Create policy for admins to view all configurations
CREATE POLICY admin_view_all_policy ON routing_configurations
    FOR SELECT
    USING (current_setting('app.user_role', true) = 'admin');

-- Create policy for admins to modify configurations
CREATE POLICY admin_modify_policy ON routing_configurations
    FOR ALL
    USING (current_setting('app.user_role', true) = 'admin');

-- Add comments for documentation
COMMENT ON TABLE routing_configurations IS 'Stores routing configurations for companies with dedicated or shared infrastructure';
COMMENT ON COLUMN routing_configurations.routing_type IS 'Type of routing: shared (default) or dedicated';
COMMENT ON COLUMN routing_configurations.dedicated_server_id IS 'Foreign key to dedicated_gateways table';
COMMENT ON COLUMN routing_configurations.dedicated_server_url IS 'URL endpoint for the dedicated server';
COMMENT ON COLUMN routing_configurations.dedicated_server_api_key IS 'API key for authenticating with dedicated server';
COMMENT ON COLUMN routing_configurations.health_status IS 'Current health status: healthy, degraded, or unhealthy';
COMMENT ON COLUMN routing_configurations.average_latency_ms IS 'Average request latency in milliseconds';
