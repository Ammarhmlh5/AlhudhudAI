-- Migration: 035_distribution_strategies_table.sql
-- Description: Create distribution_strategies table for managing message distribution across devices
-- Requirements: 9.4
-- Feature: admin-interface-enhancements

-- Create distribution_strategies table
CREATE TABLE IF NOT EXISTS distribution_strategies (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Strategy configuration
    strategy_type VARCHAR(50) NOT NULL CHECK (strategy_type IN ('round_robin', 'load_balancing', 'priority', 'custom')),
    strategy_config JSONB,
    
    -- Custom rules (for custom strategy)
    custom_rules JSONB,
    
    -- Status
    is_active BOOLEAN NOT NULL DEFAULT true,
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Ensure custom_rules is provided when strategy_type is custom
    CONSTRAINT check_custom_rules CHECK (
        strategy_type != 'custom' OR 
        (strategy_type = 'custom' AND custom_rules IS NOT NULL)
    )
);

-- Create indexes for performance
CREATE INDEX idx_distribution_strategies_tenant ON distribution_strategies(tenant_id);
CREATE INDEX idx_distribution_strategies_type ON distribution_strategies(strategy_type);
CREATE INDEX idx_distribution_strategies_active ON distribution_strategies(is_active) WHERE is_active = true;

-- Enable Row Level Security for tenant isolation
ALTER TABLE distribution_strategies ENABLE ROW LEVEL SECURITY;

-- Create RLS policy for tenant isolation
CREATE POLICY tenant_isolation_policy ON distribution_strategies
    USING (tenant_id = current_setting('app.current_tenant', true)::uuid);

-- Create policy for admins to view all strategies
CREATE POLICY admin_view_all_policy ON distribution_strategies
    FOR SELECT
    USING (current_setting('app.user_role', true) = 'admin');

-- Create policy for admins to modify strategies
CREATE POLICY admin_modify_policy ON distribution_strategies
    FOR ALL
    USING (current_setting('app.user_role', true) = 'admin');

-- Add comments for documentation
COMMENT ON TABLE distribution_strategies IS 'Stores message distribution strategies for companies with multiple devices';
COMMENT ON COLUMN distribution_strategies.strategy_type IS 'Distribution strategy: round_robin, load_balancing, priority, or custom';
COMMENT ON COLUMN distribution_strategies.strategy_config IS 'JSON configuration specific to the strategy type';
COMMENT ON COLUMN distribution_strategies.custom_rules IS 'JSON rules for custom distribution strategy';

-- Example strategy_config structures:
-- round_robin: {"start_index": 0}
-- load_balancing: {"load_threshold": 80, "rebalance_interval": 60}
-- priority: {"priority_levels": [1, 2, 3, 4, 5], "fallback_to_lower": true}
-- custom: stored in custom_rules field

-- Example custom_rules structure:
-- {
--   "rules": [
--     {"condition": "time_of_day", "operator": "between", "value": ["09:00", "17:00"], "device_ids": ["uuid1", "uuid2"]},
--     {"condition": "message_type", "operator": "equals", "value": "urgent", "device_ids": ["uuid3"]},
--     {"condition": "default", "device_ids": ["uuid1", "uuid2", "uuid3", "uuid4"]}
--   ]
-- }
