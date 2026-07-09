-- Migration: Add gateway_id to accounting_transactions
-- Purpose: Fast financial reporting per gateway

ALTER TABLE accounting_transactions ADD COLUMN IF NOT EXISTS gateway_id UUID REFERENCES dedicated_gateways(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_accounting_transactions_gateway ON accounting_transactions(gateway_id);

COMMENT ON COLUMN accounting_transactions.gateway_id IS 'Link to dedicated gateway for financial reporting and profit margin calculation';
