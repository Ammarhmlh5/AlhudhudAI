-- Migration 070: Phase 3 - Admin Statistics Table & Tenant Aggregation
-- This creates the admin aggregation tables so the Admin dashboard never does slow COUNTs
-- BEGIN;

-- 1. Create the global admin statistics table (single-row, constantly refreshed)
CREATE TABLE IF NOT EXISTS admin_statistics (
    id                      INT PRIMARY KEY DEFAULT 1,

    -- Tenant metrics
    total_tenants           INT NOT NULL DEFAULT 0,
    active_tenants          INT NOT NULL DEFAULT 0,

    -- Device metrics
    total_devices           INT NOT NULL DEFAULT 0,
    active_devices          INT NOT NULL DEFAULT 0,

    -- Message metrics (reset-able daily)
    pending_messages        INT NOT NULL DEFAULT 0,
    queued_messages         INT NOT NULL DEFAULT 0,
    processing_messages     INT NOT NULL DEFAULT 0,
    total_messages_today    INT NOT NULL DEFAULT 0,
    sent_messages_today     INT NOT NULL DEFAULT 0,
    failed_messages_today   INT NOT NULL DEFAULT 0,

    -- Balance aggregates across all tenants
    total_balance           DECIMAL(14, 4) NOT NULL DEFAULT 0,

    -- Raw ingestion metrics
    total_events_today      INT NOT NULL DEFAULT 0,
    parsed_events_today     INT NOT NULL DEFAULT 0,
    failed_events_today     INT NOT NULL DEFAULT 0,

    last_updated            TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Enforce single-row via CHECK
    CONSTRAINT single_row CHECK (id = 1)
);

-- Insert the single row if it doesn't exist
INSERT INTO admin_statistics (id) VALUES (1) ON CONFLICT (id) DO NOTHING;

-- 2. Create the stored function to refresh admin_statistics atomically
CREATE OR REPLACE FUNCTION refresh_admin_statistics()
RETURNS void AS $$
BEGIN
    UPDATE admin_statistics SET
        total_tenants         = (SELECT COUNT(*) FROM tenants WHERE deleted_at IS NULL),
        active_tenants        = (SELECT COUNT(*) FROM tenants WHERE is_active = true AND deleted_at IS NULL),
        total_devices         = (SELECT COUNT(*) FROM paired_devices),
        active_devices        = (SELECT COUNT(*) FROM paired_devices WHERE status = 'ONLINE'),
        pending_messages      = (SELECT COUNT(*) FROM messages WHERE status = 'PENDING'),
        queued_messages       = (SELECT COUNT(*) FROM messages WHERE status = 'QUEUED'),
        processing_messages   = (SELECT COUNT(*) FROM messages WHERE status = 'PROCESSING'),
        total_messages_today  = (SELECT COUNT(*) FROM messages WHERE created_at >= CURRENT_DATE),
        sent_messages_today   = (SELECT COUNT(*) FROM messages WHERE status IN ('SENT','DELIVERED') AND created_at >= CURRENT_DATE),
        failed_messages_today = (SELECT COUNT(*) FROM messages WHERE status = 'FAILED' AND created_at >= CURRENT_DATE),
        total_balance         = (SELECT COALESCE(SUM(balance), 0) FROM tenants WHERE deleted_at IS NULL),
        total_events_today    = (SELECT COUNT(*) FROM global_incoming_events WHERE created_at >= CURRENT_DATE),
        parsed_events_today   = (SELECT COUNT(*) FROM global_incoming_events WHERE status = 'PARSED' AND created_at >= CURRENT_DATE),
        failed_events_today   = (SELECT COUNT(*) FROM global_incoming_events WHERE status = 'FAILED' AND created_at >= CURRENT_DATE),
        last_updated          = NOW()
    WHERE id = 1;
END;
$$ LANGUAGE plpgsql;

-- 3. Optional: create an index to speed up queries on global_incoming_events status + date
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_global_events_status_date
    ON global_incoming_events(status, created_at DESC);
