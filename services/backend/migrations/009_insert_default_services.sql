-- Migration: Insert Default Services for Existing Tenants
-- Version: 1.0
-- Date: 2026-02-19
-- Description: Inserts the three default services for all existing tenants

-- This migration should be run after 008_tariff_services_table.sql
-- It inserts the three default services for all existing tenants in the system

BEGIN;

-- Insert default services for all existing tenants
-- This uses the insert_default_services_for_tenant function created in migration 008

DO $$
DECLARE
    tenant_record RECORD;
    inserted_count INTEGER := 0;
BEGIN
    -- Loop through all active tenants
    FOR tenant_record IN 
        SELECT id, business_name 
        FROM tenants 
        WHERE is_active = true
    LOOP
        -- Insert default services for this tenant
        PERFORM insert_default_services_for_tenant(tenant_record.id);
        inserted_count := inserted_count + 1;
        
        RAISE NOTICE 'Inserted default services for tenant: % (ID: %)', 
            tenant_record.business_name, tenant_record.id;
    END LOOP;
    
    RAISE NOTICE 'Total tenants processed: %', inserted_count;
END $$;

COMMIT;

-- Verification: Show count of services per tenant
DO $$
DECLARE
    tenant_record RECORD;
    service_count INTEGER;
BEGIN
    RAISE NOTICE '=== Services Count per Tenant ===';
    
    FOR tenant_record IN 
        SELECT id, business_name 
        FROM tenants 
        WHERE is_active = true
        ORDER BY business_name
    LOOP
        SELECT COUNT(*) INTO service_count
        FROM services
        WHERE tenant_id = tenant_record.id;
        
        RAISE NOTICE 'Tenant: % - Services: %', 
            tenant_record.business_name, service_count;
    END LOOP;
END $$;
