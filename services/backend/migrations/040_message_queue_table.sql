-- Migration: 040_message_queue_table.sql
-- Description: Create message_queue table for Signal/Push messaging routing system
-- This table stores messages in a persistent queue for batch processing in Push mode

-- Create message_queue table
CREATE TABLE IF NOT EXISTS message_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Message information
    message_id VARCHAR(255) NOT NULL,
    message_type VARCHAR(10) NOT NULL CHECK (message_type IN ('signal', 'push')),
    message_content TEXT NOT NULL,
    recipient VARCHAR(50) NOT NULL,
    
    -- Routing information
    device_id UUID REFERENCES phone_devices(id) ON DELETE SET NULL,
    
    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'processing', 'sent', 'failed')),
    priority INTEGER NOT NULL DEFAULT 1 CHECK (priority >= 1 AND priority <= 10),
    
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    sent_at TIMESTAMP,
    delivered_at TIMESTAMP,
    
    -- Retry information
    retry_count INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 3,
    last_error TEXT,
    
    -- Metadata
    metadata JSONB
);

-- Create indexes for performance
CREATE INDEX idx_message_queue_tenant_status ON message_queue(tenant_id, status, created_at);
CREATE INDEX idx_message_queue_device ON message_queue(device_id, status) WHERE device_id IS NOT NULL;
CREATE INDEX idx_message_queue_priority ON message_queue(priority DESC, created_at ASC) WHERE status = 'queued';

-- Enable Row Level Security for multi-tenancy
ALTER TABLE message_queue ENABLE ROW LEVEL SECURITY;

-- Create RLS policy for tenant isolation
CREATE POLICY tenant_isolation_policy ON message_queue
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- Create policy for service role (bypass RLS)
CREATE POLICY service_role_policy ON message_queue
    USING (current_setting('app.bypass_rls', true)::text = 'on')
    WITH CHECK (current_setting('app.bypass_rls', true)::text = 'on');

-- Add comments for documentation
COMMENT ON TABLE message_queue IS 'Persistent message queue for Signal/Push routing system';
COMMENT ON COLUMN message_queue.id IS 'Unique identifier for the queued message';
COMMENT ON COLUMN message_queue.tenant_id IS 'Reference to the tenant (company) that owns this message';
COMMENT ON COLUMN message_queue.message_id IS 'External message identifier';
COMMENT ON COLUMN message_queue.message_type IS 'Type of message: signal (direct) or push (batched)';
COMMENT ON COLUMN message_queue.message_content IS 'The actual message content to be sent';
COMMENT ON COLUMN message_queue.recipient IS 'Phone number or recipient identifier';
COMMENT ON COLUMN message_queue.device_id IS 'Target device for message delivery';
COMMENT ON COLUMN message_queue.status IS 'Current status: queued, processing, sent, or failed';
COMMENT ON COLUMN message_queue.priority IS 'Message priority (1-10, higher = more urgent)';
COMMENT ON COLUMN message_queue.retry_count IS 'Number of retry attempts made';
COMMENT ON COLUMN message_queue.max_retries IS 'Maximum number of retry attempts allowed';
COMMENT ON COLUMN message_queue.metadata IS 'Additional metadata in JSON format';
