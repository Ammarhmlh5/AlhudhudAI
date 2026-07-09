-- Test Script for Referrals Table Migration
-- Usage: psql -U postgres -d sms_gateway -f test_referrals_migration.sql

\echo '========================================'
\echo '  Testing Referrals Table Migration'
\echo '========================================'
\echo ''

-- =====================================================
-- 1. Verify Table Structure
-- =====================================================

\echo '[TEST 1] Verifying table structure...'
\echo ''

SELECT 
    column_name,
    data_type,
    is_nullable,
    column_default
FROM information_schema.columns
WHERE table_name = 'referrals'
ORDER BY ordinal_position;

\echo ''

-- =====================================================
-- 2. Verify Indexes
-- =====================================================

\echo '[TEST 2] Verifying indexes...'
\echo ''

SELECT 
    indexname,
    indexdef
FROM pg_indexes
WHERE tablename = 'referrals'
ORDER BY indexname;

\echo ''

-- =====================================================
-- 3. Verify Constraints
-- =====================================================

\echo '[TEST 3] Verifying constraints...'
\echo ''

SELECT 
    conname as constraint_name,
    contype as constraint_type,
    pg_get_constraintdef(oid) as definition
FROM pg_constraint
WHERE conrelid = 'referrals'::regclass
ORDER BY conname;

\echo ''

-- =====================================================
-- 4. Verify RLS Policies
-- =====================================================

\echo '[TEST 4] Verifying Row Level Security policies...'
\echo ''

SELECT 
    schemaname,
    tablename,
    policyname,
    permissive,
    roles,
    cmd,
    qual,
    with_check
FROM pg_policies
WHERE tablename = 'referrals'
ORDER BY policyname;

\echo ''

-- =====================================================
-- 5. Verify Triggers
-- =====================================================

\echo '[TEST 5] Verifying triggers...'
\echo ''

SELECT 
    trigger_name,
    event_manipulation,
    event_object_table,
    action_statement
FROM information_schema.triggers
WHERE event_object_table = 'referrals'
ORDER BY trigger_name;

\echo ''

-- =====================================================
-- 6. Verify Views
-- =====================================================

\echo '[TEST 6] Verifying views...'
\echo ''

SELECT 
    table_name,
    view_definition
FROM information_schema.views
WHERE table_name = 'referral_stats';

\echo ''

-- =====================================================
-- 7. Test Data Insertion
-- =====================================================

\echo '[TEST 7] Testing data insertion...'
\echo ''

-- Create test tenants if they don't exist
DO $$
BEGIN
    -- Test tenant 1 (referrer)
    IF NOT EXISTS (SELECT 1 FROM tenants WHERE id = '11111111-1111-1111-1111-111111111111') THEN
        INSERT INTO tenants (id, name, email, balance, currency)
        VALUES ('11111111-1111-1111-1111-111111111111', 'Test Referrer', 'referrer@test.com', 100.00, 'USD');
    END IF;
    
    -- Test tenant 2 (referee)
    IF NOT EXISTS (SELECT 1 FROM tenants WHERE id = '22222222-2222-2222-2222-222222222222') THEN
        INSERT INTO tenants (id, name, email, balance, currency)
        VALUES ('22222222-2222-2222-2222-222222222222', 'Test Referee', 'referee@test.com', 50.00, 'USD');
    END IF;
END $$;

-- Insert test referral
INSERT INTO referrals (
    referrer_tenant_id,
    referee_tenant_id,
    referral_code,
    reward_type,
    reward_amount,
    reward_condition
) VALUES (
    '11111111-1111-1111-1111-111111111111',
    '22222222-2222-2222-2222-222222222222',
    'TEST-REF-001',
    'balance',
    10.00,
    'first_payment'
) ON CONFLICT DO NOTHING;

\echo 'Test referral inserted successfully'
\echo ''

-- =====================================================
-- 8. Test Constraint: Different Referrer and Referee
-- =====================================================

\echo '[TEST 8] Testing constraint: referrer != referee...'
\echo ''

-- This should fail
DO $$
BEGIN
    INSERT INTO referrals (
        referrer_tenant_id,
        referee_tenant_id,
        referral_code
    ) VALUES (
        '11111111-1111-1111-1111-111111111111',
        '11111111-1111-1111-1111-111111111111',  -- Same as referrer
        'TEST-FAIL-001'
    );
    
    RAISE EXCEPTION 'TEST FAILED: Should not allow same referrer and referee';
EXCEPTION
    WHEN check_violation THEN
        RAISE NOTICE 'TEST PASSED: Constraint check_referrer_referee_different works correctly';
END $$;

\echo ''

-- =====================================================
-- 9. Test Constraint: Positive Reward Amount
-- =====================================================

\echo '[TEST 9] Testing constraint: reward_amount > 0...'
\echo ''

-- This should fail
DO $$
BEGIN
    INSERT INTO referrals (
        referrer_tenant_id,
        referee_tenant_id,
        referral_code,
        reward_amount
    ) VALUES (
        '11111111-1111-1111-1111-111111111111',
        '22222222-2222-2222-2222-222222222222',
        'TEST-FAIL-002',
        -10.00  -- Negative amount
    );
    
    RAISE EXCEPTION 'TEST FAILED: Should not allow negative reward amount';
EXCEPTION
    WHEN check_violation THEN
        RAISE NOTICE 'TEST PASSED: Constraint check_reward_amount_positive works correctly';
END $$;

\echo ''

-- =====================================================
-- 10. Test Constraint: Valid Status
-- =====================================================

\echo '[TEST 10] Testing constraint: valid status...'
\echo ''

-- This should fail
DO $$
BEGIN
    INSERT INTO referrals (
        referrer_tenant_id,
        referee_tenant_id,
        referral_code,
        status
    ) VALUES (
        '11111111-1111-1111-1111-111111111111',
        '22222222-2222-2222-2222-222222222222',
        'TEST-FAIL-003',
        'invalid_status'  -- Invalid status
    );
    
    RAISE EXCEPTION 'TEST FAILED: Should not allow invalid status';
EXCEPTION
    WHEN check_violation THEN
        RAISE NOTICE 'TEST PASSED: Constraint check_status_valid works correctly';
END $$;

\echo ''

-- =====================================================
-- 11. Test Constraint: Valid Reward Type
-- =====================================================

\echo '[TEST 11] Testing constraint: valid reward type...'
\echo ''

-- This should fail
DO $$
BEGIN
    INSERT INTO referrals (
        referrer_tenant_id,
        referee_tenant_id,
        referral_code,
        reward_type
    ) VALUES (
        '11111111-1111-1111-1111-111111111111',
        '22222222-2222-2222-2222-222222222222',
        'TEST-FAIL-004',
        'invalid_type'  -- Invalid reward type
    );
    
    RAISE EXCEPTION 'TEST FAILED: Should not allow invalid reward type';
EXCEPTION
    WHEN check_violation THEN
        RAISE NOTICE 'TEST PASSED: Constraint check_reward_type_valid works correctly';
END $$;

\echo ''

-- =====================================================
-- 12. Test Trigger: Updated At
-- =====================================================

\echo '[TEST 12] Testing trigger: update updated_at...'
\echo ''

-- Get current updated_at
DO $$
DECLARE
    old_updated_at TIMESTAMP;
    new_updated_at TIMESTAMP;
BEGIN
    SELECT updated_at INTO old_updated_at
    FROM referrals
    WHERE referral_code = 'TEST-REF-001';
    
    -- Wait a moment
    PERFORM pg_sleep(1);
    
    -- Update the record
    UPDATE referrals
    SET notes = 'Test update'
    WHERE referral_code = 'TEST-REF-001';
    
    -- Get new updated_at
    SELECT updated_at INTO new_updated_at
    FROM referrals
    WHERE referral_code = 'TEST-REF-001';
    
    IF new_updated_at > old_updated_at THEN
        RAISE NOTICE 'TEST PASSED: Trigger update_referrals_updated_at works correctly';
    ELSE
        RAISE EXCEPTION 'TEST FAILED: updated_at was not updated';
    END IF;
END $$;

\echo ''

-- =====================================================
-- 13. Test View: Referral Stats
-- =====================================================

\echo '[TEST 13] Testing view: referral_stats...'
\echo ''

SELECT * FROM referral_stats
WHERE referrer_tenant_id = '11111111-1111-1111-1111-111111111111';

\echo ''

-- =====================================================
-- 14. Test Index Performance
-- =====================================================

\echo '[TEST 14] Testing index performance...'
\echo ''

EXPLAIN ANALYZE
SELECT * FROM referrals
WHERE referrer_tenant_id = '11111111-1111-1111-1111-111111111111'
ORDER BY created_at DESC;

\echo ''

-- =====================================================
-- 15. Test RLS: Set Current Tenant
-- =====================================================

\echo '[TEST 15] Testing Row Level Security...'
\echo ''

-- Set current tenant to referrer
SET app.current_tenant = '11111111-1111-1111-1111-111111111111';

-- Should see the referral
SELECT 
    id,
    referrer_tenant_id,
    referee_tenant_id,
    referral_code,
    status
FROM referrals
WHERE referral_code = 'TEST-REF-001';

\echo ''

-- Reset current tenant
RESET app.current_tenant;

-- =====================================================
-- 16. Test Complete Workflow
-- =====================================================

\echo '[TEST 16] Testing complete workflow...'
\echo ''

-- Create a new referral
INSERT INTO referrals (
    referrer_tenant_id,
    referee_tenant_id,
    referral_code,
    reward_type,
    reward_amount,
    reward_condition,
    status
) VALUES (
    '11111111-1111-1111-1111-111111111111',
    '22222222-2222-2222-2222-222222222222',
    'TEST-WORKFLOW-001',
    'messages',
    100,
    'first_subscription',
    'pending'
) RETURNING id;

-- Update to completed
UPDATE referrals
SET 
    condition_met = true,
    condition_met_at = CURRENT_TIMESTAMP,
    status = 'completed'
WHERE referral_code = 'TEST-WORKFLOW-001';

-- Grant reward
UPDATE referrals
SET 
    reward_granted = true,
    reward_granted_at = CURRENT_TIMESTAMP,
    status = 'rewarded'
WHERE referral_code = 'TEST-WORKFLOW-001';

-- Verify final state
SELECT 
    referral_code,
    status,
    condition_met,
    reward_granted,
    reward_type,
    reward_amount
FROM referrals
WHERE referral_code = 'TEST-WORKFLOW-001';

\echo ''

-- =====================================================
-- 17. Summary
-- =====================================================

\echo '========================================'
\echo '  Test Summary'
\echo '========================================'
\echo ''

SELECT 
    'Total Referrals' as metric,
    COUNT(*)::text as value
FROM referrals
UNION ALL
SELECT 
    'Pending Referrals',
    COUNT(*)::text
FROM referrals
WHERE status = 'pending'
UNION ALL
SELECT 
    'Completed Referrals',
    COUNT(*)::text
FROM referrals
WHERE status = 'completed'
UNION ALL
SELECT 
    'Rewarded Referrals',
    COUNT(*)::text
FROM referrals
WHERE status = 'rewarded'
UNION ALL
SELECT 
    'Total Indexes',
    COUNT(*)::text
FROM pg_indexes
WHERE tablename = 'referrals'
UNION ALL
SELECT 
    'Total Policies',
    COUNT(*)::text
FROM pg_policies
WHERE tablename = 'referrals';

\echo ''
\echo '[INFO] All tests completed!'
\echo ''
\echo 'Next steps:'
\echo '  1. Review test results above'
\echo '  2. Implement Referral Manager (internal/referral/manager.go)'
\echo '  3. Implement Referral APIs'
\echo ''
