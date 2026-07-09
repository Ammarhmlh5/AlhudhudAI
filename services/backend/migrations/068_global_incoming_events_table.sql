-- Migration: Create global_incoming_events table for raw message ingestion
-- Up Migration
BEGIN;

CREATE TABLE IF NOT EXISTS global_incoming_events (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID,                     -- Nullable if unknown initially, but usually set via API Key later
    sender_ip INET,
    raw_payload JSONB NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'RECEIVED', -- RECEIVED, PARSED, FAILED, COMPLETED
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Index for searching status and creation
CREATE INDEX idx_global_incoming_events_status ON global_incoming_events(status);
CREATE INDEX idx_global_incoming_events_created_at ON global_incoming_events(created_at DESC);
CREATE INDEX idx_global_incoming_events_tenant ON global_incoming_events(tenant_id) WHERE tenant_id IS NOT NULL;

-- Automatically update `updated_at` triggers
CREATE TRIGGER update_global_incoming_events_updated_at
BEFORE UPDATE ON global_incoming_events
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

COMMIT;

-- Down Migration removed for automated execution compatibility.
