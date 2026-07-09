-- Phase 4: Admin Reporting and Message Event Architecture
-- Date: 2026-03-22

BEGIN;

-- 1. Admin Statistics Table (Global)
CREATE TABLE IF NOT EXISTS admin_statistics (
    id INTEGER PRIMARY KEY DEFAULT 1,
    total_tenants INTEGER DEFAULT 0,
    active_tenants INTEGER DEFAULT 0,
    total_devices INTEGER DEFAULT 0,
    active_devices INTEGER DEFAULT 0,
    pending_messages INTEGER DEFAULT 0,
    queued_messages INTEGER DEFAULT 0,
    processing_messages INTEGER DEFAULT 0,
    total_messages_today INTEGER DEFAULT 0,
    sent_messages_today INTEGER DEFAULT 0,
    failed_messages_today INTEGER DEFAULT 0,
    total_balance DECIMAL(14,4) DEFAULT 0,
    total_events_today INTEGER DEFAULT 0,
    parsed_events_today INTEGER DEFAULT 0,
    failed_events_today INTEGER DEFAULT 0,
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Insert initial row if not exists
INSERT INTO admin_statistics (id) VALUES (1) ON CONFLICT (id) DO NOTHING;

-- 2. Message Events Table (Tenant Specific)
-- Note: This will be created in tenant schemas via GORM MigrateTenantSchema
-- but we define it here for global reference if needed or for manual creation.

CREATE TABLE IF NOT EXISTS message_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    name TEXT,
    source TEXT,
    status TEXT DEFAULT 'RECEIVED',
    message_type TEXT,
    total_items INTEGER DEFAULT 0,
    processed_items INTEGER DEFAULT 0,
    failed_items INTEGER DEFAULT 0,
    started_at TIMESTAMP WITH TIME ZONE,
    finished_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_message_events_tenant_status ON message_events(tenant_id, status);

-- 3. Update Message Table (Add EventID)
ALTER TABLE messages ADD COLUMN IF NOT EXISTS event_id UUID;
CREATE INDEX IF NOT EXISTS idx_messages_event_id ON messages(event_id);

COMMIT;
