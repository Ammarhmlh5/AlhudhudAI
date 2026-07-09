-- Migration: Add performance indexes for messages table
-- Purpose: Optimize queries for tenant isolation and message processing
-- Related Requirements: 3.1, 4.1, 18.4

-- Add indexes for tenant isolation and performance
CREATE INDEX IF NOT EXISTS idx_messages_tenant_status ON messages(tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_messages_tenant_created ON messages(tenant_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_messages_message_id ON messages(message_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_messages_scheduled ON messages(scheduled_at) WHERE status = 'SCHEDULED' AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_messages_queued_at ON messages(queued_at DESC) WHERE status = 'QUEUED' AND deleted_at IS NULL;

-- Add comments
COMMENT ON INDEX idx_messages_tenant_status IS 'Optimizes tenant-specific status queries with isolation';
COMMENT ON INDEX idx_messages_tenant_created IS 'Optimizes tenant message history queries';
COMMENT ON INDEX idx_messages_message_id IS 'Fast lookup by message_id for status reports';
COMMENT ON INDEX idx_messages_scheduled IS 'Optimizes scheduled message processing';
COMMENT ON INDEX idx_messages_queued_at IS 'Optimizes batch processor queue extraction';
