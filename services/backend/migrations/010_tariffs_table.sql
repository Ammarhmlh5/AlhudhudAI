-- Migration: Add Tariffs Table
-- Version: 1.0
-- Date: 2026-02-19
-- Description: Creates tariffs table for dynamic pricing with full tenant isolation

-- This migration creates the tariffs table that supports:
-- - Dynamic tariffs based on network operator and country
-- - Effective date ranges for tariff changes
-- - Full tenant isolation with Row Level Security
-- - Initial tariffs for Hudud Gateway (9 YER for Yemen Mobile, 13 YER for others)

BEGIN;

-- 1. Create tariffs table
CREATE TABLE IF NOT EXISTS tariffs (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,  -- CRITICAL: Required for tenant isolation
    service_id INTEGER NOT NULL,
    country_code VARCHAR(2) NOT NULL, -- 'YE', 'SA', etc.
    network_operator VARCHAR(100), -- 'Yemen Mobile', 'Sabafon', 'Other', etc.
    message_type VARCHAR(50) DEFAULT 'standard', -- 'standard', 'unicode'
    cost_per_message DECIMAL(10, 4) NOT NULL,
    currency VARCHAR(3) DEFAULT 'YER', -- 'YER', 'USD', 'SAR'
    effective_from DATE NOT NULL,
    effective_to DATE,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT check_cost_positive CHECK (cost_per_message > 0),
    CONSTRAINT check_effective_dates CHECK (effective_to IS NULL OR effective_to >= effective_from),
    CONSTRAINT check_currency CHECK (currency IN ('YER', 'USD', 'SAR', 'EUR', 'GBP'))
);

-- 2. Create indexes for performance and isolation
-- All indexes start with tenant_id for optimal performance with RLS
CREATE INDEX idx_tariffs_tenant_id ON tariffs(tenant_id);
CREATE INDEX idx_tariffs_tenant_service ON tariffs(tenant_id, service_id);
CREATE INDEX idx_tariffs_tenant_country ON tariffs(tenant_id, country_code);
CREATE INDEX idx_tariffs_tenant_operator ON tariffs(tenant_id, network_operator);
CREATE INDEX idx_tariffs_tenant_active ON tariffs(tenant_id, is_active);
CREATE INDEX idx_tariffs_tenant_effective ON tariffs(tenant_id, effective_from, effective_to);

-- Composite index for common query pattern: finding active tariff for a specific service/country/operator
CREATE INDEX idx_tariffs_lookup ON tariffs(tenant_id, service_id, country_code, network_operator, is_active, effective_from, effective_to);

-- 3. Enable Row Level Security (RLS)
ALTER TABLE tariffs ENABLE ROW LEVEL SECURITY;

-- 4. Create RLS Policies for tenant isolation

-- Policy for SELECT: Users can only see their tenant's tariffs
CREATE POLICY tenant_isolation_select_policy ON tariffs
    FOR SELECT
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy for INSERT: Users can only insert tariffs for their tenant
CREATE POLICY tenant_isolation_insert_policy ON tariffs
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy for UPDATE: Users can only update their tenant's tariffs
CREATE POLICY tenant_isolation_update_policy ON tariffs
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant')::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy for DELETE: Users can only delete their tenant's tariffs
CREATE POLICY tenant_isolation_delete_policy ON tariffs
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- 5. Create trigger for updated_at
CREATE OR REPLACE FUNCTION update_tariffs_updated_at()
RETURNS TRIGGER AS $
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_tariffs_updated_at
    BEFORE UPDATE ON tariffs
    FOR EACH ROW
    EXECUTE FUNCTION update_tariffs_updated_at();

-- 6. Create function to insert default Hudud Gateway tariffs for a tenant
CREATE OR REPLACE FUNCTION insert_default_tariffs_for_tenant(p_tenant_id UUID)
RETURNS void AS $
DECLARE
    v_hudud_service_id INTEGER;
BEGIN
    -- Get the Hudud Gateway service ID for this tenant
    SELECT id INTO v_hudud_service_id
    FROM services
    WHERE tenant_id = p_tenant_id 
    AND code = 'HUDUD_GATEWAY'
    LIMIT 1;
    
    -- If service doesn't exist, raise notice and exit
    IF v_hudud_service_id IS NULL THEN
        RAISE NOTICE 'Hudud Gateway service not found for tenant %. Skipping tariff insertion.', p_tenant_id;
        RETURN;
    END IF;
    
    -- Insert default tariffs for Hudud Gateway
    -- Yemen Mobile: 9 YER per message
    INSERT INTO tariffs (
        tenant_id, 
        service_id, 
        country_code, 
        network_operator, 
        message_type,
        cost_per_message, 
        currency, 
        effective_from,
        is_active
    ) VALUES
    (
        p_tenant_id,
        v_hudud_service_id,
        'YE',
        'Yemen Mobile',
        'standard',
        9.0000,
        'YER',
        CURRENT_DATE,
        true
    ),
    -- Other networks: 13 YER per message
    (
        p_tenant_id,
        v_hudud_service_id,
        'YE',
        'Other',
        'standard',
        13.0000,
        'YER',
        CURRENT_DATE,
        true
    )
    ON CONFLICT DO NOTHING;
    
    RAISE NOTICE 'Inserted default tariffs for tenant %', p_tenant_id;
END;
$ LANGUAGE plpgsql;

-- 7. Create function to get active tariff for a specific service/country/operator
CREATE OR REPLACE FUNCTION get_active_tariff(
    p_tenant_id UUID,
    p_service_id INTEGER,
    p_country_code VARCHAR(2),
    p_network_operator VARCHAR(100),
    p_message_type VARCHAR(50) DEFAULT 'standard',
    p_date DATE DEFAULT CURRENT_DATE
)
RETURNS TABLE (
    id INTEGER,
    cost_per_message DECIMAL(10, 4),
    currency VARCHAR(3)
) AS $
BEGIN
    RETURN QUERY
    SELECT 
        t.id,
        t.cost_per_message,
        t.currency
    FROM tariffs t
    WHERE t.tenant_id = p_tenant_id
    AND t.service_id = p_service_id
    AND t.country_code = p_country_code
    AND (t.network_operator = p_network_operator OR t.network_operator = 'Other')
    AND t.message_type = p_message_type
    AND t.is_active = true
    AND t.effective_from <= p_date
    AND (t.effective_to IS NULL OR t.effective_to >= p_date)
    ORDER BY 
        -- Prefer specific operator over 'Other'
        CASE WHEN t.network_operator = p_network_operator THEN 0 ELSE 1 END,
        t.effective_from DESC
    LIMIT 1;
END;
$ LANGUAGE plpgsql;

-- 8. Comments for documentation
COMMENT ON TABLE tariffs IS 'Stores dynamic tariffs for SMS services based on network operator, country, and date ranges';
COMMENT ON COLUMN tariffs.tenant_id IS 'Tenant ID for multi-tenant isolation - CRITICAL for RLS';
COMMENT ON COLUMN tariffs.service_id IS 'Reference to the service (e.g., Hudud Gateway)';
COMMENT ON COLUMN tariffs.country_code IS 'ISO 3166-1 alpha-2 country code (e.g., YE for Yemen)';
COMMENT ON COLUMN tariffs.network_operator IS 'Network operator name (e.g., Yemen Mobile, Sabafon, or Other for default)';
COMMENT ON COLUMN tariffs.message_type IS 'Message type: standard (GSM-7) or unicode';
COMMENT ON COLUMN tariffs.cost_per_message IS 'Cost per message in the specified currency';
COMMENT ON COLUMN tariffs.effective_from IS 'Date when this tariff becomes effective';
COMMENT ON COLUMN tariffs.effective_to IS 'Date when this tariff expires (NULL = no expiry)';
COMMENT ON COLUMN tariffs.is_active IS 'Whether this tariff is currently active';

COMMENT ON FUNCTION insert_default_tariffs_for_tenant(UUID) IS 'Inserts default Hudud Gateway tariffs (9 YER for Yemen Mobile, 13 YER for others) for a tenant';
COMMENT ON FUNCTION get_active_tariff(UUID, INTEGER, VARCHAR, VARCHAR, VARCHAR, DATE) IS 'Returns the active tariff for a specific service/country/operator combination at a given date';

COMMIT;

-- Usage Instructions:
-- To insert default tariffs for a specific tenant, run:
-- SELECT insert_default_tariffs_for_tenant('<your-tenant-uuid>');
--
-- To get active tariff for a tenant:
-- SELECT * FROM get_active_tariff(
--     '<tenant-uuid>',
--     <service-id>,
--     'YE',
--     'Yemen Mobile',
--     'standard',
--     CURRENT_DATE
-- );
