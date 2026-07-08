-- Migration: 033_performance_metrics_table.sql
-- Description: Create performance_metrics table for tracking company performance over time
-- Requirements: 9.2
-- Feature: admin-interface-enhancements

-- Create performance_metrics table
CREATE TABLE IF NOT EXISTS performance_metrics (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Time period
    period_start TIMESTAMP NOT NULL,
    period_end TIMESTAMP NOT NULL,
    period_type VARCHAR(20) NOT NULL CHECK (period_type IN ('hourly', 'daily', 'monthly')),
    
    -- Message metrics
    total_messages BIGINT NOT NULL DEFAULT 0 CHECK (total_messages >= 0),
    successful_messages BIGINT NOT NULL DEFAULT 0 CHECK (successful_messages >= 0),
    failed_messages BIGINT NOT NULL DEFAULT 0 CHECK (failed_messages >= 0),
    
    -- Computed success rate
    success_rate DECIMAL(5,2) GENERATED ALWAYS AS (
        CASE WHEN total_messages > 0 
        THEN (successful_messages::DECIMAL / total_messages) * 100 
        ELSE 0 END
    ) STORED,
    
    -- Delivery metrics
    average_delivery_time_ms INTEGER CHECK (average_delivery_time_ms >= 0),
    p95_delivery_time_ms INTEGER CHECK (p95_delivery_time_ms >= 0),
    p99_delivery_time_ms INTEGER CHECK (p99_delivery_time_ms >= 0),
    
    -- Error metrics
    error_count INTEGER NOT NULL DEFAULT 0 CHECK (error_count >= 0),
    
    -- Computed error rate
    error_rate DECIMAL(5,2) GENERATED ALWAYS AS (
        CASE WHEN total_messages > 0 
        THEN (error_count::DECIMAL / total_messages) * 100 
        ELSE 0 END
    ) STORED,
    
    -- Payment metrics
    total_payments INTEGER NOT NULL DEFAULT 0 CHECK (total_payments >= 0),
    on_time_payments INTEGER NOT NULL DEFAULT 0 CHECK (on_time_payments >= 0),
    
    -- Computed payment compliance
    payment_compliance DECIMAL(5,2) GENERATED ALWAYS AS (
        CASE WHEN total_payments > 0 
        THEN (on_time_payments::DECIMAL / total_payments) * 100 
        ELSE 0 END
    ) STORED,
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Ensure successful + failed = total
    CONSTRAINT check_message_totals CHECK (successful_messages + failed_messages <= total_messages),
    -- Ensure on_time <= total payments
    CONSTRAINT check_payment_totals CHECK (on_time_payments <= total_payments),
    -- Ensure period_end > period_start
    CONSTRAINT check_period_order CHECK (period_end > period_start)
);

-- Create indexes for performance
CREATE INDEX idx_performance_metrics_tenant_period ON performance_metrics(tenant_id, period_start DESC);
CREATE INDEX idx_performance_metrics_type ON performance_metrics(period_type);
CREATE INDEX idx_performance_metrics_success_rate ON performance_metrics(success_rate);

-- Create unique constraint to prevent duplicate metrics for same period
CREATE UNIQUE INDEX idx_performance_metrics_unique ON performance_metrics(tenant_id, period_start, period_type);

-- Enable Row Level Security for tenant isolation
ALTER TABLE performance_metrics ENABLE ROW LEVEL SECURITY;

-- Create RLS policy for tenant isolation
CREATE POLICY tenant_isolation_policy ON performance_metrics
    USING (tenant_id = current_setting('app.current_tenant', true)::uuid);

-- Create policy for admins to view all metrics
CREATE POLICY admin_view_all_policy ON performance_metrics
    FOR SELECT
    USING (current_setting('app.user_role', true) = 'admin');

-- Add comments for documentation
COMMENT ON TABLE performance_metrics IS 'Stores aggregated performance metrics for companies over different time periods';
COMMENT ON COLUMN performance_metrics.period_type IS 'Type of aggregation period: hourly, daily, or monthly';
COMMENT ON COLUMN performance_metrics.success_rate IS 'Computed success rate: (successful_messages / total_messages) * 100';
COMMENT ON COLUMN performance_metrics.error_rate IS 'Computed error rate: (error_count / total_messages) * 100';
COMMENT ON COLUMN performance_metrics.payment_compliance IS 'Computed payment compliance: (on_time_payments / total_payments) * 100';
COMMENT ON COLUMN performance_metrics.p95_delivery_time_ms IS '95th percentile delivery time in milliseconds';
COMMENT ON COLUMN performance_metrics.p99_delivery_time_ms IS '99th percentile delivery time in milliseconds';
