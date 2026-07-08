-- Migration: Add Tariff Services Table
-- Version: 1.0
-- Date: 2026-02-19
-- Description: Creates services table for tariff and pricing system with tenant isolation

-- This migration creates the services table that supports three service models:
-- 1. Hudud Gateway Service (pay-as-you-go)
-- 2. Mobile Gateway Service (monthly subscription)
-- 3. Dedicated Gateway Service (gateway-as-a-service for enterprises)

BEGIN;

-- 1. Create services table
CREATE TABLE IF NOT EXISTS services (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,  -- CRITICAL: Required for tenant isolation
    name VARCHAR(255) NOT NULL,
    code VARCHAR(50) NOT NULL,
    type VARCHAR(50) NOT NULL, -- 'gateway', 'mobile', 'dedicated'
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT check_service_type CHECK (type IN ('gateway', 'mobile', 'dedicated'))
);

-- 2. Create indexes for performance and isolation
-- All indexes start with tenant_id for optimal performance
CREATE INDEX idx_services_tenant_id ON services(tenant_id);
CREATE INDEX idx_services_tenant_type ON services(tenant_id, type);
CREATE INDEX idx_services_tenant_active ON services(tenant_id, is_active);
CREATE INDEX idx_services_tenant_code ON services(tenant_id, code);

-- UNIQUE constraint: code must be unique per tenant only
CREATE UNIQUE INDEX idx_services_tenant_code_unique 
ON services(tenant_id, code);

-- 3. Enable Row Level Security (RLS)
ALTER TABLE services ENABLE ROW LEVEL SECURITY;

-- 4. Create RLS Policies for tenant isolation

-- Policy for SELECT: Users can only see their tenant's services
CREATE POLICY tenant_isolation_select_policy ON services
    FOR SELECT
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy for INSERT: Users can only insert services for their tenant
CREATE POLICY tenant_isolation_insert_policy ON services
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy for UPDATE: Users can only update their tenant's services
CREATE POLICY tenant_isolation_update_policy ON services
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant')::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy for DELETE: Users can only delete their tenant's services
CREATE POLICY tenant_isolation_delete_policy ON services
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- 5. Create trigger for updated_at
CREATE OR REPLACE FUNCTION update_services_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_services_updated_at
    BEFORE UPDATE ON services
    FOR EACH ROW
    EXECUTE FUNCTION update_services_updated_at();

-- 6. Insert initial service data
-- Note: These are global service definitions that should be available to all tenants
-- In a real multi-tenant setup, you would insert these for each tenant during tenant creation
-- For now, we'll insert them with a placeholder tenant_id that should be updated

-- Example for a specific tenant (replace with actual tenant_id):
-- INSERT INTO services (tenant_id, name, code, type, description, is_active) VALUES
-- ('<tenant_uuid>', 'خدمة بوابة الهدهد', 'HUDUD_GATEWAY', 'gateway', 'إرسال الرسائل عبر بوابة الهدهد بنموذج الدفع حسب الاستخدام', true),
-- ('<tenant_uuid>', 'خدمة الإرسال عبر الهاتف', 'MOBILE_GATEWAY', 'mobile', 'إرسال الرسائل عبر الهاتف بنموذج الاشتراك الشهري', true),
-- ('<tenant_uuid>', 'خدمة إدارة البوابات', 'DEDICATED_GATEWAY', 'dedicated', 'بوابات مخصصة للشركات المؤسسية', true);

-- 7. Create function to insert default services for a tenant
CREATE OR REPLACE FUNCTION insert_default_services_for_tenant(p_tenant_id UUID)
RETURNS void AS $$
BEGIN
    -- Insert the three default services for the tenant
    INSERT INTO services (tenant_id, name, code, type, description, is_active) VALUES
    (p_tenant_id, 'خدمة بوابة الهدهد', 'HUDUD_GATEWAY', 'gateway', 'إرسال الرسائل عبر بوابة الهدهد بنموذج الدفع حسب الاستخدام', true),
    (p_tenant_id, 'خدمة الإرسال عبر الهاتف', 'MOBILE_GATEWAY', 'mobile', 'إرسال الرسائل عبر الهاتف بنموذج الاشتراك الشهري', true),
    (p_tenant_id, 'خدمة إدارة البوابات', 'DEDICATED_GATEWAY', 'dedicated', 'بوابات مخصصة للشركات المؤسسية', true)
    ON CONFLICT (tenant_id, code) DO NOTHING;
END;
$$ LANGUAGE plpgsql;

-- 8. Comments for documentation
COMMENT ON TABLE services IS 'Stores the three service models: Hudud Gateway (pay-as-you-go), Mobile Gateway (subscription), and Dedicated Gateway (enterprise)';
COMMENT ON COLUMN services.tenant_id IS 'Tenant ID for multi-tenant isolation - CRITICAL for RLS';
COMMENT ON COLUMN services.type IS 'Service type: gateway (pay-as-you-go), mobile (subscription), dedicated (enterprise)';
COMMENT ON COLUMN services.code IS 'Unique service code per tenant (HUDUD_GATEWAY, MOBILE_GATEWAY, DEDICATED_GATEWAY)';

COMMIT;

-- Usage Instructions:
-- To insert default services for a specific tenant, run:
-- SELECT insert_default_services_for_tenant('<your-tenant-uuid>');
