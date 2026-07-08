-- Migration 067: Performance Indexes and GIN Indexes
-- Requirements: Performance Optimization for JSONB and High-Volume Queries

-- 1. GIN Index for Dispatch Channels Config (JSONB)
-- This allows fast searching within the JSON configuration (e.g., DeviceID, API Keys)
CREATE INDEX IF NOT EXISTS idx_dispatch_channels_config_gin ON dispatch_channels USING gin (config);

-- 2. Composite Index for Messages Status and Creation Date
-- Optimizes queries filtering by status and ordering by date (common in dispatcher and API)
CREATE INDEX IF NOT EXISTS idx_messages_tenant_status_created ON messages (tenant_id, status, created_at DESC);

-- 3. Composite Index for Accounting Transactions
-- Optimizes financial reporting and settlement checks
CREATE INDEX IF NOT EXISTS idx_accounting_transactions_tenant_status_created ON accounting_transactions (tenant_id, status, created_at DESC);

-- 4. Composite Index for Staff Activity
-- Optimizes audit log retrieval
CREATE INDEX IF NOT EXISTS idx_staff_activity_tenant_action_created ON staff_activities (tenant_id, action, created_at DESC);

-- Rollback Script
-- DROP INDEX IF EXISTS idx_dispatch_channels_config_gin;
-- DROP INDEX IF EXISTS idx_messages_tenant_status_created;
-- DROP INDEX IF EXISTS idx_accounting_transactions_tenant_status_created;
-- DROP INDEX IF EXISTS idx_staff_activity_tenant_action_created;
