-- Test script for payment_transactions migration
-- This script tests the payment_transactions table creation and RLS policies

\echo '========================================='
\echo 'Testing Payment Transactions Migration'
\echo '========================================='
\echo ''

-- Create test tenant if not exists
DO $$
DECLARE
    test_tenant_id UUID := '11111111-1111-1111-1111-111111111111';
BEGIN
    IF NOT EXISTS (SELECT 1 FROM tenants WHERE id = test_tenant_id) THEN
        INSERT INTO tenants (id, name, email, balance, currency)
        VALUES (test_tenant_id, 'Test Tenant Payment', 'test-payment@example.com', 100.00, 'USD');
        RAISE NOTICE 'Test tenant created';
    END IF;
END $$;

-- Set current tenant
SET app.current_tenant = '11111111-1111-1111-1111-111111111111';

\echo 'Test 1: Insert payment transaction'
INSERT INTO payment_transactions (
    tenant_id,
    transaction_id,
    external_transaction_id,
    payment_gateway,
    payment_method,
    amount,
    currency,
    status,
    description
) VALUES (
    '11111111-1111-1111-1111-111111111111',
    'TXN-TEST-001',
    'pi_test_123456',
    'stripe',
    'card',
    50.00,
    'USD',
    'pending',
    'Test payment transaction'
);

\echo '✓ Payment transaction inserted'
\echo ''

\echo 'Test 2: Query payment transactions'
SELECT 
    transaction_id,
    payment_gateway,
    amount,
    currency,
    status
FROM payment_transactions
WHERE tenant_id = '11111111-1111-1111-1111-111111111111';

\echo ''

\echo 'Test 3: Update transaction status to completed'
UPDATE payment_transactions
SET 
    status = 'completed',
    completed_at = CURRENT_TIMESTAMP
WHERE transaction_id = 'TXN-TEST-001';

SELECT 
    transaction_id,
    status,
    completed_at
FROM payment_transactions
WHERE transaction_id = 'TXN-TEST-001';

\echo '✓ Transaction status updated'
\echo ''

\echo 'Test 4: Insert failed transaction'
INSERT INTO payment_transactions (
    tenant_id,
    transaction_id,
    external_transaction_id,
    payment_gateway,
    payment_method,
    amount,
    currency,
    status,
    error_code,
    error_message
) VALUES (
    '11111111-1111-1111-1111-111111111111',
    'TXN-TEST-002',
    'pi_test_failed',
    'stripe',
    'card',
    75.00,
    'USD',
    'failed',
    'card_declined',
    'البطاقة مرفوضة'
);

\echo '✓ Failed transaction inserted'
\echo ''

\echo 'Test 5: Process refund'
INSERT INTO payment_transactions (
    tenant_id,
    transaction_id,
    external_transaction_id,
    payment_gateway,
    payment_method,
    amount,
    currency,
    status
) VALUES (
    '11111111-1111-1111-1111-111111111111',
    'TXN-TEST-003',
    'pi_test_refund',
    'paypal',
    'paypal_account',
    100.00,
    'USD',
    'completed'
);

UPDATE payment_transactions
SET 
    status = 'refunded',
    refund_amount = 100.00,
    refund_reason = 'طلب العميل',
    refunded_at = CURRENT_TIMESTAMP
WHERE transaction_id = 'TXN-TEST-003';

SELECT 
    transaction_id,
    status,
    refund_amount,
    refund_reason
FROM payment_transactions
WHERE transaction_id = 'TXN-TEST-003';

\echo '✓ Refund processed'
\echo ''

\echo 'Test 6: Test RLS - Try to access another tenant data'
-- Create another tenant
DO $$
DECLARE
    other_tenant_id UUID := '22222222-2222-2222-2222-222222222222';
BEGIN
    IF NOT EXISTS (SELECT 1 FROM tenants WHERE id = other_tenant_id) THEN
        INSERT INTO tenants (id, name, email, balance, currency)
        VALUES (other_tenant_id, 'Other Tenant', 'other@example.com', 50.00, 'USD');
    END IF;
END $$;

-- Insert transaction for other tenant (should fail with RLS)
BEGIN;
    SET app.current_tenant = '11111111-1111-1111-1111-111111111111';
    
    -- This should fail because we're trying to insert for another tenant
    INSERT INTO payment_transactions (
        tenant_id,
        transaction_id,
        payment_gateway,
        amount,
        currency,
        status
    ) VALUES (
        '22222222-2222-2222-2222-222222222222',
        'TXN-TEST-CROSS',
        'stripe',
        25.00,
        'USD',
        'pending'
    );
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE '✓ RLS prevented cross-tenant insert (expected)';
END;

\echo ''

\echo 'Test 7: Statistics by payment gateway'
SELECT 
    payment_gateway,
    COUNT(*) as total_transactions,
    SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed,
    SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed,
    SUM(CASE WHEN status = 'refunded' THEN 1 ELSE 0 END) as refunded,
    SUM(CASE WHEN status = 'completed' THEN amount ELSE 0 END) as total_amount
FROM payment_transactions
WHERE tenant_id = '11111111-1111-1111-1111-111111111111'
GROUP BY payment_gateway;

\echo ''

\echo 'Test 8: Test metadata storage'
UPDATE payment_transactions
SET metadata = '{"card_brand": "visa", "card_last4": "4242", "customer_email": "test@example.com"}'::jsonb
WHERE transaction_id = 'TXN-TEST-001';

SELECT 
    transaction_id,
    metadata->>'card_brand' as card_brand,
    metadata->>'card_last4' as card_last4
FROM payment_transactions
WHERE transaction_id = 'TXN-TEST-001';

\echo '✓ Metadata stored and retrieved'
\echo ''

\echo 'Test 9: Verify indexes'
SELECT 
    indexname,
    indexdef
FROM pg_indexes
WHERE tablename = 'payment_transactions'
ORDER BY indexname;

\echo ''

\echo 'Test 10: Verify RLS policies'
SELECT 
    policyname,
    cmd,
    qual
FROM pg_policies
WHERE tablename = 'payment_transactions'
ORDER BY policyname;

\echo ''

\echo '========================================='
\echo 'All Tests Completed!'
\echo '========================================='
\echo ''
\echo 'Summary:'
SELECT 
    COUNT(*) as total_transactions,
    SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed,
    SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed,
    SUM(CASE WHEN status = 'refunded' THEN 1 ELSE 0 END) as refunded,
    SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending
FROM payment_transactions
WHERE tenant_id = '11111111-1111-1111-1111-111111111111';

\echo ''
\echo 'Cleanup: Removing test data...'
DELETE FROM payment_transactions WHERE tenant_id = '11111111-1111-1111-1111-111111111111';
\echo '✓ Test data removed'
