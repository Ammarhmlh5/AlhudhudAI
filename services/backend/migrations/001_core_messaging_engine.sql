-- Migration: Add Core Messaging Engine Tables
-- Version: 1.0
-- Date: 2026-02-18

-- This migration should be run in each tenant schema

BEGIN;

-- 1. Create DispatchChannel table
CREATE TABLE IF NOT EXISTS dispatch_channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,  -- CRITICAL: Required for tenant isolation
    type VARCHAR(50) NOT NULL, -- PHONE, GATEWAY_HUDHUD, GATEWAY_PRIVATE
    channel_key VARCHAR(255) NOT NULL,  -- Removed UNIQUE - will be unique per tenant
    config JSONB DEFAULT '{}',
    status VARCHAR(20) DEFAULT 'OFFLINE', -- ONLINE, OFFLINE, WAIT_MODE
    priority INTEGER DEFAULT 5,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for performance and isolation
CREATE INDEX idx_dispatch_channels_tenant_id ON dispatch_channels(tenant_id);
CREATE INDEX idx_dispatch_channels_status ON dispatch_channels(status);
CREATE INDEX idx_dispatch_channels_priority ON dispatch_channels(priority DESC);

-- UNIQUE constraint: channel_key must be unique per tenant only
CREATE UNIQUE INDEX idx_dispatch_channels_tenant_key 
ON dispatch_channels(tenant_id, channel_key);

-- 2. Create AccountingTransaction table
-- NOTE: This table definition has been moved to migration 003_accounting_transactions_table.sql
-- The definition below is kept for reference only and should not be executed
-- CREATE TABLE IF NOT EXISTS accounting_transactions (
--     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
--     tenant_id UUID NOT NULL,
--     message_id UUID NOT NULL,
--     amount DECIMAL(10, 2) NOT NULL,
--     status VARCHAR(20) DEFAULT 'PENDING', -- PENDING, COMMITTED, REFUNDED
--     created_at TIMESTAMP DEFAULT NOW(),
--     updated_at TIMESTAMP DEFAULT NOW()
-- );
-- 
-- CREATE INDEX idx_accounting_transactions_message_id ON accounting_transactions(message_id);
-- CREATE INDEX idx_accounting_transactions_status ON accounting_transactions(status);

-- 3. Create AccountingSystem table (global, not per-tenant)
-- This should be run in the main schema, not tenant schemas
-- CREATE TABLE IF NOT EXISTS accounting_systems (
--     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
--     name VARCHAR(255) NOT NULL,
--     type VARCHAR(50) NOT NULL,
--     config JSONB DEFAULT '{}',
--     is_active BOOLEAN DEFAULT true,
--     created_at TIMESTAMP DEFAULT NOW(),
--     updated_at TIMESTAMP DEFAULT NOW()
-- );

-- 4. Modify Messages table to add new columns
ALTER TABLE messages 
ADD COLUMN IF NOT EXISTS channel_id UUID REFERENCES dispatch_channels(id),
ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(255),
ADD COLUMN IF NOT EXISTS wait_reason TEXT,
ADD COLUMN IF NOT EXISTS provider_job_id VARCHAR(255),
ADD COLUMN IF NOT EXISTS cost DECIMAL(10, 2) DEFAULT 0.0;

CREATE INDEX IF NOT EXISTS idx_messages_channel_id ON messages(channel_id);
CREATE INDEX IF NOT EXISTS idx_messages_idempotency_key ON messages(idempotency_key);
CREATE INDEX IF NOT EXISTS idx_messages_status ON messages(status);

-- 5. Modify Tenants table to add new columns
-- This should be run in the main schema
-- ALTER TABLE tenants
-- ADD COLUMN IF NOT EXISTS allow_gateways BOOLEAN DEFAULT false;

-- 6. Create IncomingMessage table
CREATE TABLE IF NOT EXISTS incoming_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    phone_number VARCHAR(20),
    message_content TEXT,
    raw_payload JSONB,
    status VARCHAR(20) DEFAULT 'PENDING',
    processed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_incoming_messages_status ON incoming_messages(status);
CREATE INDEX idx_incoming_messages_tenant_id ON incoming_messages(tenant_id);

-- 7. Insert default dispatch channel (ALHudhudAI Global Gateway)
-- NOTE: This should be inserted for each tenant schema separately
-- You may want to do this programmatically during tenant creation
-- Example for a specific tenant:
-- INSERT INTO dispatch_channels (tenant_id, type, channel_key, status, priority, config)
-- VALUES (
--     '<tenant_uuid>',
--     'GATEWAY_HUDHUD',
--     'global_default',
--     'ONLINE',
--     10,
--     '{"endpoint": "https://api.alhudhud.com/v1/sms", "description": "ALHudhudAI Global Gateway"}'::jsonb
-- )
-- ON CONFLICT (tenant_id, channel_key) DO NOTHING;

COMMIT;

-- Rollback script (if needed)
-- BEGIN;
-- DROP TABLE IF EXISTS accounting_transactions CASCADE;
-- DROP TABLE IF EXISTS dispatch_channels CASCADE;
-- DROP TABLE IF EXISTS incoming_messages CASCADE;
-- ALTER TABLE messages DROP COLUMN IF EXISTS channel_id;
-- ALTER TABLE messages DROP COLUMN IF EXISTS idempotency_key;
-- ALTER TABLE messages DROP COLUMN IF EXISTS wait_reason;
-- ALTER TABLE messages DROP COLUMN IF EXISTS provider_job_id;
-- ALTER TABLE messages DROP COLUMN IF EXISTS cost;
-- COMMIT;
