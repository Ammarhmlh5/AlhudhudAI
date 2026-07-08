-- Migration 064: GIN Indexes for JSONB
-- Description: Optimizes queries on JSONB columns using GIN indexes.

-- 1. Index for dispatch_channels config
CREATE INDEX IF NOT EXISTS idx_dispatch_channels_config_gin ON dispatch_channels USING GIN (config);

-- 2. Index for incoming_messages raw_payload
CREATE INDEX IF NOT EXISTS idx_incoming_messages_payload_gin ON incoming_messages USING GIN (raw_payload);

-- 3. Index for audit_logs JSONB columns
CREATE INDEX IF NOT EXISTS idx_audit_logs_old_values_gin ON audit_logs USING GIN (old_values);
CREATE INDEX IF NOT EXISTS idx_audit_logs_new_values_gin ON audit_logs USING GIN (new_values);

-- 4. Index for dedicated_gateways config (if exists)
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'dedicated_gateways' AND column_name = 'config') THEN
        CREATE INDEX IF NOT EXISTS idx_dedicated_gateways_config_gin ON dedicated_gateways USING GIN (config);
    END IF;
END $$;
