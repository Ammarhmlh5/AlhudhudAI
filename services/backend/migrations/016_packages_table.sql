-- Migration: Add Packages Table
-- Version: 1.0
-- Date: 2026-02-20
-- Description: Creates packages table for monthly subscription packages with tenant isolation

-- This migration creates the packages table that supports monthly subscription packages:
-- - Basic: $10 - 1,000 messages/month
-- - Standard: $20 - 3,000 messages/month
-- - Professional: $40 - 8,000 messages/month
-- - Business: $50 - 15,000 messages/month
-- - Enterprise: $100 - Unlimited messages

BEGIN;

-- 1. Create packages table
CREATE TABLE IF NOT EXISTS packages (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,  -- CRITICAL: Required for tenant isolation
    service_id INTEGER NOT NULL,  -- References services(id)
    name VARCHAR(255) NOT NULL,
    code VARCHAR(50) NOT NULL,
    price_usd DECIMAL(10, 2) NOT NULL,
    included_messages INTEGER, -- NULL = unlimited
    overage_rate DECIMAL(10, 4), -- السعر لكل رسالة إضافية
    is_unlimited BOOLEAN DEFAULT false,
    billing_cycle VARCHAR(20) DEFAULT 'monthly', -- 'monthly', 'yearly'
    
    -- إعدادات Rollover
    rollover_enabled BOOLEAN DEFAULT false,
    rollover_limit INTEGER, -- NULL = غير محدود
    rollover_expiry_months INTEGER, -- NULL = لا تنتهي
    
    -- الحالة
    is_active BOOLEAN DEFAULT true,
    display_order INTEGER DEFAULT 0,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT check_price_positive CHECK (price_usd > 0),
    CONSTRAINT check_messages_positive CHECK (included_messages IS NULL OR included_messages > 0),
    CONSTRAINT check_billing_cycle CHECK (billing_cycle IN ('monthly', 'yearly'))
);

-- 2. Create indexes for performance and isolation
-- All indexes start with tenant_id for optimal performance
CREATE INDEX idx_packages_tenant_id ON packages(tenant_id);
CREATE INDEX idx_packages_tenant_service ON packages(tenant_id, service_id);
CREATE INDEX idx_packages_tenant_active ON packages(tenant_id, is_active);
CREATE INDEX idx_packages_tenant_order ON packages(tenant_id, display_order);

-- UNIQUE constraint: code must be unique per tenant only
CREATE UNIQUE INDEX idx_packages_tenant_code_unique 
ON packages(tenant_id, code);

-- 3. Enable Row Level Security (RLS)
ALTER TABLE packages ENABLE ROW LEVEL SECURITY;

-- 4. Create RLS Policies for tenant isolation

-- Policy for SELECT: Users can only see their tenant's packages
CREATE POLICY tenant_isolation_select_policy ON packages
    FOR SELECT
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy for INSERT: Users can only insert packages for their tenant
CREATE POLICY tenant_isolation_insert_policy ON packages
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy for UPDATE: Users can only update their tenant's packages
CREATE POLICY tenant_isolation_update_policy ON packages
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant')::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy for DELETE: Users can only delete their tenant's packages
CREATE POLICY tenant_isolation_delete_policy ON packages
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- 5. Create trigger for updated_at
CREATE OR REPLACE FUNCTION update_packages_updated_at()
RETURNS TRIGGER AS $
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_packages_updated_at
    BEFORE UPDATE ON packages
    FOR EACH ROW
    EXECUTE FUNCTION update_packages_updated_at();

-- 6. Create function to insert default packages for a tenant
CREATE OR REPLACE FUNCTION insert_default_packages_for_tenant(p_tenant_id UUID, p_service_id INTEGER)
RETURNS void AS $
BEGIN
    -- Insert the five default packages for the tenant
    -- Note: service_id should be the ID of 'MOBILE_GATEWAY' service (typically 2)
    INSERT INTO packages (
        tenant_id, 
        service_id, 
        name, 
        code, 
        price_usd, 
        included_messages, 
        overage_rate, 
        is_unlimited, 
        rollover_enabled, 
        display_order
    ) VALUES
    (p_tenant_id, p_service_id, 'Basic', 'PKG_BASIC', 10.00, 1000, 0.0100, false, true, 1),
    (p_tenant_id, p_service_id, 'Standard', 'PKG_STANDARD', 20.00, 3000, 0.0067, false, true, 2),
    (p_tenant_id, p_service_id, 'Professional', 'PKG_PROFESSIONAL', 40.00, 8000, 0.0050, false, true, 3),
    (p_tenant_id, p_service_id, 'Business', 'PKG_BUSINESS', 50.00, 15000, 0.0033, false, true, 4),
    (p_tenant_id, p_service_id, 'Enterprise', 'PKG_ENTERPRISE', 100.00, NULL, 0.0000, true, false, 5)
    ON CONFLICT (tenant_id, code) DO NOTHING;
END;
$ LANGUAGE plpgsql;

-- 7. Comments for documentation
COMMENT ON TABLE packages IS 'Stores monthly subscription packages with different pricing tiers and message quotas';
COMMENT ON COLUMN packages.tenant_id IS 'Tenant ID for multi-tenant isolation - CRITICAL for RLS';
COMMENT ON COLUMN packages.service_id IS 'References the service (typically MOBILE_GATEWAY service)';
COMMENT ON COLUMN packages.included_messages IS 'Number of messages included in the package (NULL = unlimited)';
COMMENT ON COLUMN packages.overage_rate IS 'Price per additional message beyond the included quota';
COMMENT ON COLUMN packages.is_unlimited IS 'Whether the package has unlimited messages (Enterprise tier)';
COMMENT ON COLUMN packages.rollover_enabled IS 'Whether unused messages can be rolled over to next month';
COMMENT ON COLUMN packages.rollover_limit IS 'Maximum number of messages that can be rolled over (NULL = unlimited)';
COMMENT ON COLUMN packages.rollover_expiry_months IS 'Number of months before rolled over messages expire (NULL = never expire)';

COMMIT;

-- Usage Instructions:
-- To insert default packages for a specific tenant, run:
-- SELECT insert_default_packages_for_tenant('<your-tenant-uuid>', <mobile-gateway-service-id>);
-- 
-- Example:
-- SELECT insert_default_packages_for_tenant('123e4567-e89b-12d3-a456-426614174000', 2);

-- Rollback script (if needed):
-- BEGIN;
-- DROP TRIGGER IF EXISTS trigger_packages_updated_at ON packages;
-- DROP FUNCTION IF EXISTS update_packages_updated_at();
-- DROP FUNCTION IF EXISTS insert_default_packages_for_tenant(UUID, INTEGER);
-- DROP TABLE IF EXISTS packages CASCADE;
-- COMMIT;
