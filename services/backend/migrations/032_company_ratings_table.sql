-- Migration: 032_company_ratings_table.sql
-- Description: Create company_ratings table for storing company performance ratings
-- Requirements: 9.1
-- Feature: admin-interface-enhancements

-- Create company_ratings table
CREATE TABLE IF NOT EXISTS company_ratings (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Rating components
    success_rate DECIMAL(5,2) NOT NULL CHECK (success_rate >= 0 AND success_rate <= 100),
    sending_volume BIGINT NOT NULL CHECK (sending_volume >= 0),
    payment_compliance DECIMAL(5,2) NOT NULL CHECK (payment_compliance >= 0 AND payment_compliance <= 100),
    service_quality_score DECIMAL(5,2) NOT NULL CHECK (service_quality_score >= 0 AND service_quality_score <= 100),
    
    -- Calculated rating
    overall_rating DECIMAL(5,2) NOT NULL CHECK (overall_rating >= 0 AND overall_rating <= 100),
    rating_category VARCHAR(20) NOT NULL CHECK (rating_category IN ('excellent', 'good', 'average', 'poor')),
    
    -- Metadata
    calculation_timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
    valid_from TIMESTAMP NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMP,
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX idx_company_ratings_tenant ON company_ratings(tenant_id, calculation_timestamp DESC);
CREATE INDEX idx_company_ratings_category ON company_ratings(rating_category);
CREATE INDEX idx_company_ratings_valid ON company_ratings(valid_from, valid_to);
CREATE INDEX idx_company_ratings_overall ON company_ratings(overall_rating DESC);

-- Enable Row Level Security for tenant isolation
ALTER TABLE company_ratings ENABLE ROW LEVEL SECURITY;

-- Create RLS policy for tenant isolation
CREATE POLICY tenant_isolation_policy ON company_ratings
    USING (tenant_id = current_setting('app.current_tenant', true)::uuid);

-- Create policy for admins to view all ratings
CREATE POLICY admin_view_all_policy ON company_ratings
    FOR SELECT
    USING (current_setting('app.user_role', true) = 'admin');

-- Add comments for documentation
COMMENT ON TABLE company_ratings IS 'Stores company performance ratings based on multiple metrics';
COMMENT ON COLUMN company_ratings.success_rate IS 'Message success rate percentage (0-100)';
COMMENT ON COLUMN company_ratings.sending_volume IS 'Total number of messages sent';
COMMENT ON COLUMN company_ratings.payment_compliance IS 'Payment compliance percentage (0-100)';
COMMENT ON COLUMN company_ratings.service_quality_score IS 'Service quality score (0-100)';
COMMENT ON COLUMN company_ratings.overall_rating IS 'Overall calculated rating (0-100)';
COMMENT ON COLUMN company_ratings.rating_category IS 'Rating category: excellent (>=90), good (75-89), average (60-74), poor (<60)';
COMMENT ON COLUMN company_ratings.valid_from IS 'Start of validity period for this rating';
COMMENT ON COLUMN company_ratings.valid_to IS 'End of validity period for this rating (NULL for current rating)';
