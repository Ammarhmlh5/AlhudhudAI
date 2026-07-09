-- Test Script for Subscriptions Table Migration
-- This script tests the subscriptions table creation and RLS policies

BEGIN;

-- 1. Test table creation
SELECT 'Testing subscriptions table exists...' AS test;
SELECT EXISTS (
    SELECT FROM information_schema.tables 
    WHERE table_schema = 'public' 
    AND table_name = 'subscriptions'
) AS table_exists;

-- 2. Test indexes exist
SELECT 'Testing indexes exist...' AS test;
SELECT 
    indexname,
    indexdef
FROM pg_indexes
WHERE tablename = 'subscriptions'
ORDER BY indexname;

-- 3. Test RLS is enabled
SELECT 'Testing RLS is enabled...' AS test;
SELECT 
    tablename,
    rowsecurity
FROM pg_tables
WHERE tablename = 'subscriptions';

-- 4. Test RLS policies exist
SELECT 'Testing RLS policies exist...' AS test;
SELECT 
    policyname,
    cmd,
    qual,
    with_check
FROM pg_policies
WHERE tablename = 'subscriptions'
ORDER BY policyname;

-- 5. Test constraints
SELECT 'Testing constraints exist...' AS test;
SELECT
    conname AS constraint_name,
    contype AS constraint_type,
    pg_get_constraintdef(oid) AS constraint_definition
FROM pg_constraint
WHERE conrelid = 'subscriptions'::regclass
ORDER BY conname;

-- 6. Test helper functions exist
SELECT 'Testing helper functions exist...' AS test;
SELECT 
    proname AS function_name,
    pg_get_function_arguments(oid) AS arguments,
    pg_get_functiondef(oid) AS definition
FROM pg_proc
WHERE proname IN (
    'update_subscriptions_updated_at',
    'calculate_messages_remaining',
    'is_subscription_active',
    'get_active_subscription'
)
ORDER BY proname;

-- 7. Test trigger exists
SELECT 'Testing trigger exists...' AS test;
SELECT 
    trigger_name,
    event_manipulation,
    event_object_table,
    action_statement
FROM information_schema.triggers
WHERE event_object_table = 'subscriptions'
ORDER BY trigger_name;

-- 8. Test column comments
SELECT 'Testing column comments...' AS test;
SELECT 
    cols.column_name,
    pg_catalog.col_description(c.oid, cols.ordinal_position::int) AS column_comment
FROM information_schema.columns cols
JOIN pg_catalog.pg_class c ON c.relname = cols.table_name
WHERE cols.table_name = 'subscriptions'
  AND pg_catalog.col_description(c.oid, cols.ordinal_position::int) IS NOT NULL
ORDER BY cols.ordinal_position;

ROLLBACK;

-- To actually run the migration, execute:
-- psql -U your_user -d your_database -f services/backend/migrations/017_subscriptions_table.sql
