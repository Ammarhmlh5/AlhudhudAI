-- Migration: Create accounting_transactions table
-- Purpose: Track accounting state for each message (pending, completed, refunded)
-- Related Requirements: 2.1, 2.2, 2.5, 2.6

-- Drop old accounting_transactions table if exists (from migration 001)
DROP TABLE IF EXISTS accounting_transactions CASCADE;

-- Create accounting_transactions table with correct schema
CREATE TABLE accounting_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    amount DECIMAL(12, 2) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    transaction_type VARCHAR(20) NOT NULL DEFAULT 'DEDUCT',
    discount_applied DECIMAL(12, 2) DEFAULT 0,
    original_amount DECIMAL(12, 2),
    settled_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT accounting_transactions_status_check CHECK (status IN ('PENDING', 'COMPLETED', 'REFUNDED', 'FAILED')),
    CONSTRAINT accounting_transactions_type_check CHECK (transaction_type IN ('DEDUCT', 'REFUND', 'CHARGE')),
    CONSTRAINT accounting_transactions_amount_positive CHECK (amount >= 0),
    CONSTRAINT accounting_transactions_tenant_message_unique UNIQUE(tenant_id, message_id)
);

-- Indexes for performance and tenant isolation
CREATE INDEX idx_accounting_transactions_tenant ON accounting_transactions(tenant_id, status, created_at DESC);
CREATE INDEX idx_accounting_transactions_message ON accounting_transactions(message_id);
CREATE INDEX idx_accounting_transactions_status ON accounting_transactions(status, created_at DESC);
CREATE INDEX idx_accounting_transactions_settled ON accounting_transactions(settled_at) WHERE settled_at IS NOT NULL;

-- Add comments for documentation
COMMENT ON TABLE accounting_transactions IS 'Tracks accounting state for each message - links messages to balance operations';
COMMENT ON COLUMN accounting_transactions.status IS 'Transaction status: PENDING (awaiting settlement), COMPLETED (message sent successfully), REFUNDED (message failed, balance returned), FAILED (transaction failed)';
COMMENT ON COLUMN accounting_transactions.transaction_type IS 'Type of transaction: DEDUCT (charge for sending), REFUND (return balance), CHARGE (add balance)';
COMMENT ON COLUMN accounting_transactions.discount_applied IS 'Volume discount applied to this transaction';
COMMENT ON COLUMN accounting_transactions.original_amount IS 'Original amount before discount';
COMMENT ON COLUMN accounting_transactions.settled_at IS 'Timestamp when transaction was settled (completed or refunded)';
