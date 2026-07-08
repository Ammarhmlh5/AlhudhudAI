-- Migration: 046_fix_missing_indexes.sql
-- Description: Add missing indexes for optimized message dispatching and status tracking

-- Index for the dispatcher to quickly find pending messages for active tenants
-- Requirement: Optimize fetchAndQueueMessages in tenant_dispatcher.go
CREATE INDEX IF NOT EXISTS idx_messages_tenant_status_scheduled 
ON messages (tenant_id, status, scheduled_at) 
WHERE deleted_at IS NULL;

-- Index for tracking active processing load on channels
-- Requirement: Optimize SelectChannel and status reporting
CREATE INDEX IF NOT EXISTS idx_messages_channel_status 
ON messages (channel_id, status) 
WHERE status IN ('PROCESSING', 'PENDING') AND deleted_at IS NULL;

-- Index for idempotency checks
CREATE INDEX IF NOT EXISTS idx_balance_logs_idempotency 
ON balance_logs (tenant_id, message_id) 
WHERE message_id IS NOT NULL;
