-- Test script for packages table migration
-- This script verifies the migration was applied correctly

-- 1. Check if packages table exists
SELECT EXISTS (
    SELECT FROM information_schema.tables 
    WHERE table_schema = 'public' 
    AND table_name = 'packages'
) AS packages_table_exists;

-- 2. Check table structure
SELECT 
    column_name, 
    data_type, 
    is_nullable,
    column_default
FROM information_schema.columns
WHERE table_name = 'packages'
ORDER BY ordinal_position;

-- 3. Check indexes
SELECT 
    indexname,
    indexdef
FROM pg_indexes
WHERE tablename = 'packages'
ORDER BY indexname;

-- 4. Check RLS policies
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
WHERE tablename = 'packages'
ORDER BY policyname;

-- 5. Check if RLS is enabled
SELECT 
    tablename,
    rowsecurity
FROM pg_tables
WHERE tablename = 'packages';

-- 6. Check constraints
SELECT
    conname AS constraint_name,
    contype AS constraint_type,
    pg_get_constraintdef(oid) AS constraint_definition
FROM pg_constraint
WHERE conrelid = 'packages'::regclass
ORDER BY conname;

-- 7. Check if the helper function exists
SELECT EXISTS (
    SELECT FROM pg_proc 
    WHERE proname = 'insert_default_packages_for_tenant'
) AS function_exists;

-- 8. Check if the trigger exists
SELECT EXISTS (
    SELECT FROM pg_trigger 
    WHERE tgname = 'trigger_packages_updated_at'
) AS trigger_exists;

-- Example: Insert default packages for a test tenant
-- Note: Replace '<tenant-uuid>' with an actual tenant UUID
-- Note: Replace <service-id> with the actual service ID (typically 2 for MOBILE_GATEWAY)
-- SELECT insert_default_packages_for_tenant('<tenant-uuid>', 2);

-- Example: Query packages (requires setting app.current_tenant)
-- SET app.current_tenant = '<tenant-uuid>';
-- SELECT * FROM packages;
