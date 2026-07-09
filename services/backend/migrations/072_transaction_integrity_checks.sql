-- Migration 072: Transaction Integrity Checks
-- Purpose: Provide tools to identify and fix data inconsistencies in accounting

-- View to identify messages that have a status but no corresponding accounting transaction
CREATE OR REPLACE VIEW orphaned_message_deductions AS
SELECT 
    m.id AS message_id,
    m.tenant_id,
    m.status,
    m.cost,
    m.created_at
FROM messages m
LEFT JOIN accounting_transactions at ON m.id = at.message_id
WHERE m.status NOT IN ('PENDING', 'QUEUED', 'WAITING')
  AND at.id IS NULL
  AND m.created_at < NOW() - INTERVAL '5 minutes';

-- Function to check for accounting consistency
CREATE OR REPLACE FUNCTION check_transaction_integrity()
RETURNS TABLE (
    issue_type TEXT,
    tenant_id UUID,
    message_id UUID,
    details JSONB
) AS $$
BEGIN
    -- 1. Find orphaned message deductions (Message exists, no Accounting entry)
    RETURN QUERY
    SELECT 
        'ORPHANED_MESSAGE_DEDUCTION'::TEXT,
        m.tenant_id,
        m.id,
        jsonb_build_object(
            'status', m.status,
            'cost', m.cost,
            'created_at', m.created_at
        )
    FROM messages m
    LEFT JOIN accounting_transactions at ON m.id = at.message_id
    WHERE m.status IN ('PROCESSING', 'SENT', 'FAILED')
      AND at.id IS NULL
      AND m.created_at < NOW() - INTERVAL '5 minutes';

    -- 2. Find amount mismatches between Balance History and Accounting
    RETURN QUERY
    SELECT 
        'AMOUNT_MISMATCH'::TEXT,
        bh.tenant_id,
        bh.message_id,
        jsonb_build_object(
            'history_amount', bh.amount,
            'accounting_amount', at.amount
        )
    FROM balance_history bh
    JOIN accounting_transactions at ON bh.message_id = at.message_id
    WHERE bh.amount != at.amount;

    -- 3. Find double deductions (Multiple Accounting entries for same message)
    RETURN QUERY
    SELECT 
        'DOUBLE_DEDUCTION'::TEXT,
        at.tenant_id,
        at.message_id,
        jsonb_build_object(
            'count', COUNT(*),
            'total_amount', SUM(at.amount)
        )
    FROM accounting_transactions at
    WHERE at.message_id IS NOT NULL
    GROUP BY at.tenant_id, at.message_id
    HAVING COUNT(*) > 1;
END;
$$ LANGUAGE plpgsql;

-- COMMENT on the tools
COMMENT ON VIEW orphaned_message_deductions IS 'Identifies messages that are being processed but missing a financial record.';
COMMENT ON FUNCTION check_transaction_integrity IS 'Core auditor for system financial integrity.';
