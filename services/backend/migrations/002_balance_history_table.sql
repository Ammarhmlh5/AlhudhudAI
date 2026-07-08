-- Migration: Create balance_history table for audit trail
-- Purpose: Track all balance operations (deduct, refund, charge) for financial audit
-- Related Requirements: 10.1, 10.2, 10.3, 10.4, 10.5, 20.2

-- Create balance_history table
CREATE TABLE IF NOT EXISTS balance_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    old_balance DECIMAL(12, 2) NOT NULL,
    new_balance DECIMAL(12, 2) NOT NULL,
    amount DECIMAL(12, 2) NOT NULL,
    type VARCHAR(20) NOT NULL,
    reason TEXT,
    created_by VARCHAR(100),
    ip_address INET,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT balance_history_type_check CHECK (type IN ('DEDUCT', 'REFUND', 'CHARGE', 'ADJUSTMENT'))
);

-- Indexes for performance and audit queries
CREATE INDEX idx_balance_history_tenant ON balance_history(tenant_id, created_at DESC);
CREATE INDEX idx_balance_history_message ON balance_history(message_id) WHERE message_id IS NOT NULL;
CREATE INDEX idx_balance_history_type ON balance_history(tenant_id, type, created_at DESC);
CREATE INDEX idx_balance_history_created_at ON balance_history(created_at DESC);

-- Add comment for documentation
COMMENT ON TABLE balance_history IS 'Audit trail for all balance operations - tracks every deduct, refund, charge, and adjustment';
COMMENT ON COLUMN balance_history.type IS 'Operation type: DEDUCT (charge for message), REFUND (failed message), CHARGE (add balance), ADJUSTMENT (manual correction)';
COMMENT ON COLUMN balance_history.reason IS 'Human-readable reason for the operation (e.g., MESSAGE_FAILED, MANUAL_ADJUSTMENT)';
COMMENT ON COLUMN balance_history.created_by IS 'User ID or system identifier that performed the operation';
COMMENT ON COLUMN balance_history.ip_address IS 'IP address of the request for security audit';
