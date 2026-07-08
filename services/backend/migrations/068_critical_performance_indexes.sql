-- 068_critical_performance_indexes.sql
-- Addressing Audit Points 5 & 1065: Missing Dispatcher and Accounting Indexes

-- 1. Optimized Dispatcher Index
-- Current index idx_messages_tenant_status_scheduled starts with tenant_id.
-- Multi-node dispatcher logic often filters by status/scheduled_at globally or across many tenants.
-- This partial index specifically targets PENDING messages for lightning-fast fetching.
CREATE INDEX IF NOT EXISTS idx_messages_status_scheduled_for_dispatcher 
ON messages (status, scheduled_at) 
WHERE status = 'PENDING';

-- 2. Accounting Transaction Lookup Index
-- Audit Point: Full scan on transactions when looking up by message_id.
-- This composite index makes balance/refund lookups O(1).
CREATE INDEX IF NOT EXISTS idx_accounting_transactions_tenant_message 
ON accounting_transactions (tenant_id, message_id);

-- 3. Idempotency Check Index (Audit Point 697)
-- Speeds up duplicate detection for messages with idempotent keys.
CREATE INDEX IF NOT EXISTS idx_messages_idempotency 
ON messages (tenant_id, idempotency_key) 
WHERE idempotency_key IS NOT NULL;
