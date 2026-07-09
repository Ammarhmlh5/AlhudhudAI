-- Migration: Create Monitoring and Statistics Tables
-- Purpose: Support daily aggregation and health monitoring for dedicated gateways

-- 1. Create gateway_statistics table
CREATE TABLE IF NOT EXISTS gateway_statistics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    gateway_id UUID NOT NULL REFERENCES dedicated_gateways(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    
    -- Message Stats
    messages_sent INTEGER DEFAULT 0,
    messages_delivered INTEGER DEFAULT 0,
    messages_failed INTEGER DEFAULT 0,
    messages_pending INTEGER DEFAULT 0,
    
    -- Performance Stats
    avg_response_time_ms INTEGER,
    max_response_time_ms INTEGER,
    min_response_time_ms INTEGER,
    
    -- Availability Stats
    uptime_percentage DECIMAL(5, 2) DEFAULT 100.00,
    downtime_minutes INTEGER DEFAULT 0,
    
    -- Financial Stats
    total_cost DECIMAL(12, 4) DEFAULT 0.0000,
    total_revenue DECIMAL(12, 4) DEFAULT 0.0000,
    profit_margin DECIMAL(12, 4) DEFAULT 0.0000,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT gateway_stats_date_unique UNIQUE (gateway_id, date)
);

-- 2. Create gateway_health_checks table
CREATE TABLE IF NOT EXISTS gateway_health_checks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    gateway_id UUID NOT NULL REFERENCES dedicated_gateways(id) ON DELETE CASCADE,
    checked_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_healthy BOOLEAN NOT NULL,
    response_time_ms INTEGER,
    status_code INTEGER,
    error_message TEXT,
    check_interval_minutes INTEGER DEFAULT 5
);

-- 3. Create message_logs table (Optional but good for detailed performance)
CREATE TABLE IF NOT EXISTS message_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    gateway_id UUID REFERENCES dedicated_gateways(id) ON DELETE SET NULL,
    action VARCHAR(50) NOT NULL, -- 'SEND', 'RETRY', 'STATUS_UPDATE'
    status VARCHAR(20),
    response_time_ms INTEGER,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_gateway_stats_tenant_date ON gateway_statistics(tenant_id, date DESC);
CREATE INDEX idx_health_checks_gateway_date ON gateway_health_checks(gateway_id, checked_at DESC);
CREATE INDEX idx_message_logs_message ON message_logs(message_id);

-- RLS
ALTER TABLE gateway_statistics ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_policy ON gateway_statistics
    USING (tenant_id = (current_setting('app.current_tenant', true))::uuid);

-- Comments
COMMENT ON TABLE gateway_statistics IS 'Pre-aggregated daily statistics for dedicated gateways to avoid N+1 queries during report generation';
