-- Migration 057: Unify UUID Generation
-- This migration standardizes ID generation to use UUID instead of SERIAL or VARCHAR
-- Created: May 2026

-- WARNING: This is a complex migration that modifies primary keys
-- Always backup your database before running this migration
-- Test on staging environment first

-- Step 1: Add new UUID columns to tenants table (without dropping old columns yet)
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS id_uuid UUID DEFAULT gen_random_uuid();
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS message_id_uuid UUID DEFAULT gen_random_uuid();

-- Step 2: Update tenants table to use UUID as primary key
-- First, update the new UUID column with existing values
UPDATE tenants SET id_uuid = id::UUID WHERE id IS NOT NULL AND id ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$';

-- For tenants with numeric IDs, generate new UUIDs
UPDATE tenants SET id_uuid = gen_random_uuid() WHERE id_uuid IS NULL;

-- Step 3: Create a mapping table for old ID to new UUID (for foreign key updates)
CREATE TABLE IF NOT EXISTS tenants_id_mapping (
    old_id INTEGER PRIMARY KEY,
    new_uuid UUID NOT NULL
);

-- Populate mapping table
INSERT INTO tenants_id_mapping (old_id, new_uuid)
SELECT id, id_uuid FROM tenants
ON CONFLICT (old_id) DO NOTHING;

-- Step 4: Add UUID columns to messages table
ALTER TABLE messages ADD COLUMN IF NOT EXISTS id_uuid UUID DEFAULT gen_random_uuid();
ALTER TABLE messages ADD COLUMN IF NOT EXISTS tenant_id_uuid UUID;

-- Step 5: Update messages with new UUIDs
UPDATE messages m
SET 
    id_uuid = gen_random_uuid(),
    tenant_id_uuid = t.id_uuid
FROM tenants t
WHERE m.tenant_id = t.id::UUID::TEXT;

-- For messages with numeric tenant IDs, use the mapping
UPDATE messages m
SET tenant_id_uuid = tm.new_uuid
FROM tenants_id_mapping tm
WHERE m.tenant_id = tm.old_id::TEXT
AND m.tenant_id_uuid IS NULL;

-- Step 6: Update foreign key references in other tables
-- Update messages.tenant_id to reference UUID
ALTER TABLE messages DROP CONSTRAINT IF EXISTS fk_messages_tenant;
ALTER TABLE messages ADD CONSTRAINT fk_messages_tenant_uuid 
    FOREIGN KEY (tenant_id_uuid) REFERENCES tenants(id_uuid) ON DELETE CASCADE;

-- Step 7: Create new tables with UUID primary keys (for future use)
CREATE TABLE IF NOT EXISTS tenants_v2 (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    phone VARCHAR(50),
    balance DECIMAL(12,4) DEFAULT 0,
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS messages_v2 (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants_v2(id) ON DELETE CASCADE,
    channel_id UUID NOT NULL,
    device_id UUID NOT NULL,
    phone_number VARCHAR(50) NOT NULL,
    content TEXT,
    status VARCHAR(50) DEFAULT 'pending',
    cost DECIMAL(12,4),
    sent_at TIMESTAMP,
    delivered_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP
);

-- Step 8: Create helper functions for backward compatibility
CREATE OR REPLACE FUNCTION get_tenant_id_by_old_id(p_old_id INTEGER)
RETURNS UUID AS $$
BEGIN
    RETURN (SELECT new_uuid FROM tenants_id_mapping WHERE old_id = p_old_id);
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION get_tenant_old_id_by_uuid(p_uuid UUID)
RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT old_id FROM tenants_id_mapping WHERE new_uuid = p_uuid);
END;
$$ LANGUAGE plpgsql;

-- Step 9: Create indexes on new UUID columns
CREATE INDEX IF NOT EXISTS idx_messages_tenant_id_uuid ON messages(tenant_id_uuid);
CREATE INDEX IF NOT EXISTS idx_tenants_id_uuid ON tenants(id_uuid);

-- Step 10: Add comments for documentation
COMMENT ON COLUMN tenants.id IS 'Old numeric ID (deprecated, use id_uuid)';
COMMENT ON COLUMN tenants.id_uuid IS 'New UUID primary key (use this for new records)';
COMMENT ON COLUMN messages.tenant_id IS 'Old numeric tenant ID (deprecated, use tenant_id_uuid)';
COMMENT ON COLUMN messages.tenant_id_uuid IS 'New UUID tenant reference (use this for new records)';

-- Verification queries (run these after migration)
-- SELECT 'tenants' as table_name, COUNT(*) as total, COUNT(id_uuid) as uuid_count FROM tenants;
-- SELECT 'messages' as table_name, COUNT(*) as total, COUNT(id_uuid) as uuid_count FROM messages;

-- Rollback procedure (if needed):
-- ALTER TABLE messages DROP CONSTRAINT fk_messages_tenant_uuid;
-- ALTER TABLE messages DROP COLUMN IF EXISTS tenant_id_uuid;
-- ALTER TABLE messages DROP COLUMN IF EXISTS id_uuid;
-- ALTER TABLE tenants DROP COLUMN IF EXISTS id_uuid;
-- DROP TABLE IF EXISTS tenants_id_mapping;
-- DROP FUNCTION IF EXISTS get_tenant_id_by_old_id;
-- DROP FUNCTION IF EXISTS get_tenant_old_id_by_uuid;
-- DROP TABLE IF EXISTS tenants_v2;
-- DROP TABLE IF EXISTS messages_v2;