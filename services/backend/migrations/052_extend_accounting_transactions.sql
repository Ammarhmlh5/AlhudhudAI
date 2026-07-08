-- Migration: Extend accounting_transactions table
-- Purpose: Support non-message transactions (subscriptions, rentals, adjustments)
-- Implements Requirements for unified accounting and reporting

ALTER TABLE accounting_transactions 
    ALTER COLUMN message_id DROP NOT NULL,
    ADD COLUMN description TEXT,
    ADD COLUMN reference_type VARCHAR(50),
    ADD COLUMN reference_id VARCHAR(100),
    ADD COLUMN currency VARCHAR(10) DEFAULT 'USD',
    ADD COLUMN balance_before DECIMAL(12, 4),
    ADD COLUMN balance_after DECIMAL(12, 4);

-- Update constraints to allow more transaction types and statuses
ALTER TABLE accounting_transactions DROP CONSTRAINT IF EXISTS accounting_transactions_type_check;
ALTER TABLE accounting_transactions ADD CONSTRAINT accounting_transactions_type_check 
    CHECK (transaction_type IN ('DEDUCT', 'REFUND', 'CHARGE', 'ADJUSTMENT', 'GATEWAY_SUBSCRIPTION', 'SERVER_RENTAL'));

ALTER TABLE accounting_transactions DROP CONSTRAINT IF EXISTS accounting_transactions_status_check;
ALTER TABLE accounting_transactions ADD CONSTRAINT accounting_transactions_status_check 
    CHECK (status IN ('PENDING', 'COMPLETED', 'REFUNDED', 'FAILED', 'PAID', 'SUCCESS'));

-- Update unique constraint to allow multiple non-message transactions for the same tenant
ALTER TABLE accounting_transactions DROP CONSTRAINT IF EXISTS accounting_transactions_tenant_message_unique;
ALTER TABLE accounting_transactions DROP CONSTRAINT IF EXISTS accounting_transactions_tenant_ref_unique;
CREATE UNIQUE INDEX idx_accounting_transactions_tenant_ref_unique 
    ON accounting_transactions (tenant_id, reference_type, reference_id) 
    WHERE reference_id IS NOT NULL;

-- Index for reporting performance
CREATE INDEX idx_accounting_transactions_ref ON accounting_transactions(reference_type, reference_id);
CREATE INDEX idx_accounting_transactions_created_br ON accounting_transactions(created_at DESC);

-- Comments
COMMENT ON COLUMN accounting_transactions.description IS 'Human readable description of the transaction';
COMMENT ON COLUMN accounting_transactions.reference_type IS 'Source of the transaction: dedicated_gateway, subscription, etc.';
COMMENT ON COLUMN accounting_transactions.reference_id IS 'ID of the reference object';
