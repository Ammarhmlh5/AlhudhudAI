-- Test script for balance_history RLS policies
-- This script verifies that Row Level Security is working correctly

-- First, check if RLS is enabled
SELECT 
    relname as table_name,
    relrowsecurity as rls_enabled
FROM pg_class 
WHERE relname = 'balance_history';

-- List all policies on balance_history
SELECT 
    schemaname,
    tablename,
    policyname,
    permissive,
    cmd as command,
    qual as using_expression,
    with_check as with_check_expression
FROM pg_policies
WHERE tablename = 'balance_history'
ORDER BY policyname;

-- Test 1: Verify UPDATE policy denies all updates
SELECT 
    policyname,
    cmd,
    qual
FROM pg_policies
WHERE tablename = 'balance_history' AND cmd = 'UPDATE';
-- Expected: qual should be 'false'

-- Test 2: Verify DELETE policy denies all deletions
SELECT 
    policyname,
    cmd,
    qual
FROM pg_policies
WHERE tablename = 'balance_history' AND cmd = 'DELETE';
-- Expected: qual should be 'false'

-- Test 3: Check indexes
SELECT
    indexname,
    indexdef
FROM pg_indexes
WHERE tablename = 'balance_history'
ORDER BY indexname;

-- Expected indexes:
-- 1. idx_balance_history_tenant (tenant_id, created_at DESC)
-- 2. idx_balance_history_message (message_id) WHERE message_id IS NOT NULL
-- 3. idx_balance_history_type (tenant_id, type, created_at DESC)
-- 4. idx_balance_history_created_at (created_at DESC)

-- Test 4: Verify table structure
SELECT 
    column_name,
    data_type,
    is_nullable,
    column_default
FROM information_schema.columns
WHERE table_name = 'balance_history'
ORDER BY ordinal_position;

-- Test 5: Check constraints
SELECT
    conname as constraint_name,
    contype as constraint_type,
    pg_get_constraintdef(oid) as constraint_definition
FROM pg_constraint
WHERE conrelid = 'balance_history'::regclass
ORDER BY conname;

-- Summary
SELECT 
    'RLS Policies Verification' as test_category,
    COUNT(*) as policy_count
FROM pg_policies
WHERE tablename = 'balance_history'
UNION ALL
SELECT 
    'Indexes Verification' as test_category,
    COUNT(*) as index_count
FROM pg_indexes
WHERE tablename = 'balance_history' AND indexname LIKE 'idx_%'
UNION ALL
SELECT
    'Constraints Verification' as test_category,
    COUNT(*) as constraint_count
FROM pg_constraint
WHERE conrelid = 'balance_history'::regclass;

-- Expected results:
-- RLS Policies: 4 (SELECT, INSERT, UPDATE, DELETE)
-- Indexes: 4 (tenant, message, type, created_at)
-- Constraints: 2+ (primary key, foreign keys, check constraint)
