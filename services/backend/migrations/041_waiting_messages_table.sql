-- Migration: 041_waiting_messages_table.sql
-- Description: Create waiting_messages table for tracking messages awaiting delivery reports
-- This table stores messages that have been sent to devices and are waiting for delivery confirmation

-- Create waiting_messages table
CREATE TABLE IF NOT EXISTS waiting_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    message_id VARCHAR(255) NOT NULL UNIQUE,
    
    -- Status
    status VARCHAR(30) NOT NULL DEFAULT 'waiting_delivery_report' CHECK (status IN ('waiting_delivery_report', 'delivered', 'failed', 'expired')),
    
    -- Timestamps
    sent_to_device_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX idx_waiting_messages_tenant ON waiting_messages(tenant_id, created_at DESC);
CREATE INDEX idx_waiting_messages_message_id ON waiting_messages(message_id);
CREATE INDEX idx_waiting_messages_status ON waiting_messages(status, sent_to_device_at);

-- Enable Row Level Security for multi-tenancy
ALTER TABLE waiting_messages ENABLE ROW LEVEL SECURITY;

-- Create RLS policy for tenant isolation
CREATE POLICY tenant_isolation_policy ON waiting_messages
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- Create policy for service role (bypass RLS)
CREATE POLICY service_role_policy ON waiting_messages
    USING (current_setting('app.bypass_rls', true)::text = 'on')
    WITH CHECK (current_setting('app.bypass_rls', true)::text = 'on');

-- Add comments for documentation
COMMENT ON TABLE waiting_messages IS 'Tracks messages sent to devices awaiting delivery reports';
COMMENT ON COLUMN waiting_messages.id IS 'Unique identifier for the waiting message record';
COMMENT ON COLUMN waiting_messages.tenant_id IS 'Reference to the tenant (company) that owns this message';
COMMENT ON COLUMN waiting_messages.message_id IS 'Unique message identifier (must be unique across all messages)';
COMMENT ON COLUMN waiting_messages.status IS 'Current status: waiting_delivery_report, delivered, failed, or expired';
COMMENT ON COLUMN waiting_messages.sent_to_device_at IS 'Timestamp when message was sent to the device';
COMMENT ON COLUMN waiting_messages.created_at IS 'Timestamp when record was created';
COMMENT ON COLUMN waiting_messages.updated_at IS 'Timestamp when record was last updated';
