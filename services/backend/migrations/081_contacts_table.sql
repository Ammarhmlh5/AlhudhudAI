-- Migration: 081_contacts_table.sql
-- Description: Create contacts table for the phone directory sync feature
--              (mobile SMS gateway app -> server). Mirrors paired_devices in
--              being a public schema table isolated by tenant_id + RLS.
-- Feature: android-sms-gateway / contacts-sync

CREATE TABLE IF NOT EXISTS contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    phone_number VARCHAR(32) NOT NULL,
    "group" VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_contacts_tenant_phone
    ON contacts(tenant_id, phone_number) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_contacts_tenant ON contacts(tenant_id);
CREATE INDEX IF NOT EXISTS idx_contacts_deleted_at ON contacts(deleted_at);

-- Row Level Security for tenant isolation (consistent with other tenant tables)
ALTER TABLE contacts ENABLE ROW LEVEL SECURITY;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies WHERE schemaname = 'public' AND tablename = 'contacts' AND policyname = 'contacts_tenant_isolation'
    ) THEN
        CREATE POLICY contacts_tenant_isolation ON contacts
            FOR ALL
            USING (tenant_id = current_setting('app.current_tenant', true)::UUID)
            WITH CHECK (tenant_id = current_setting('app.current_tenant', true)::UUID);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_policies WHERE schemaname = 'public' AND tablename = 'contacts' AND policyname = 'contacts_admin_all'
    ) THEN
        CREATE POLICY contacts_admin_all ON contacts
            FOR ALL
            USING (current_setting('app.is_super_admin', true) = 'true');
    END IF;
END
$$;
