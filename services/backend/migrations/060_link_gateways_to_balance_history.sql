-- Migration: Add gateway_id to balance_history
-- Purpose: Complete financial audit trail per gateway

ALTER TABLE balance_history ADD COLUMN IF NOT EXISTS gateway_id UUID REFERENCES dedicated_gateways(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_balance_history_gateway ON balance_history(gateway_id);

COMMENT ON COLUMN balance_history.gateway_id IS 'Link to dedicated gateway for comprehensive financial audit trails';
