-- ================================================================
-- Verification Script for Phase 1 Critical Fixes
-- ================================================================
-- Purpose: Verify that all Phase 1 fixes were applied successfully
-- Date: 2026-03-22
-- ================================================================

\echo '========================================'
\echo 'Phase 1 Verification Script'
\echo '========================================'
\echo ''

-- ================================================================
-- 1. Verify RBAC Tables
-- ================================================================
\echo '1. Checking RBAC Tables...'
\echo '----------------------------'

SELECT 
    CASE 
        WHEN EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'role_permissions')
        THEN '✓ role_permissions table exists'
        ELSE '✗ role_permissions table MISSING'
    END as status;

SELECT 
    CASE 
        WHEN EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'user_roles')
        THEN '✓ user_roles table exists'
        ELSE '✗ user_roles table MISSING'
    END as status;

\echo ''

-- ================================================================
-- 2. Verify Foreign Keys on RBAC Tables
-- ================================================================
\echo '2. Checking Foreign Keys...'
\echo '----------------------------'

SELECT 
    tc.table_name,
    kcu.column_name,
    ccu.table_name AS foreign_table_name,
    ccu.column_name AS foreign_column_name
FROM information_schema.table_constraints AS tc 
JOIN information_schema.key_column_usage AS kcu
    ON tc.constraint_name = kcu.constraint_name
JOIN information_schema.constraint_column_usage AS ccu
    ON ccu.constraint_name = tc.constraint_name
WHERE tc.constraint_type = 'FOREIGN KEY' 
    AND tc.table_name IN ('role_permissions', 'user_roles')
ORDER BY tc.table_name, kcu.column_name;

\echo ''

-- ================================================================
-- 3. Verify Indexes on RBAC Tables
-- ================================================================
\echo '3. Checking Indexes...'
\echo '----------------------------'

SELECT 
    schemaname,
    tablename,
    indexname,
    indexdef
FROM pg_indexes
WHERE tablename IN ('role_permissions', 'user_roles')
ORDER BY tablename, indexname;

\echo ''

-- ================================================================
-- 4. Count Records in RBAC Tables
-- ================================================================
\echo '4. Counting Records...'
\echo '----------------------------'

SELECT 
    'roles' as table_name,
    COUNT(*) as record_count
FROM roles
UNION ALL
SELECT 
    'permissions' as table_name,
    COUNT(*) as record_count
FROM permissions
UNION ALL
SELECT 
    'role_permissions' as table_name,
    COUNT(*) as record_count
FROM role_permissions
UNION ALL
SELECT 
    'user_roles' as table_name,
    COUNT(*) as record_count
FROM user_roles;

\echo ''

-- ================================================================
-- 5. Verify Table Comments
-- ================================================================
\echo '5. Checking Table Comments...'
\echo '----------------------------'

SELECT 
    c.relname as table_name,
    COALESCE(d.description, 'No comment') as comment
FROM pg_class c
LEFT JOIN pg_description d ON c.oid = d.objoid AND d.objsubid = 0
WHERE c.relname IN ('role_permissions', 'user_roles')
ORDER BY c.relname;

\echo ''

-- ================================================================
-- 6. Test RBAC Relationships
-- ================================================================
\echo '6. Testing RBAC Relationships...'
\echo '----------------------------'

-- Test if we can join roles with permissions
SELECT 
    COUNT(*) as total_role_permission_combinations
FROM roles r
CROSS JOIN permissions p;

-- Test if we can join users with roles
SELECT 
    COUNT(*) as total_user_role_combinations
FROM users u
CROSS JOIN roles r;

\echo ''

-- ================================================================
-- 7. Verify Decimal Precision (from previous migration)
-- ================================================================
\echo '7. Checking Decimal Precision...'
\echo '----------------------------'

SELECT 
    table_name,
    column_name,
    data_type,
    numeric_precision,
    numeric_scale
FROM information_schema.columns
WHERE table_name IN ('tenants', 'messages', 'balance_history', 'packages')
    AND data_type = 'numeric'
ORDER BY table_name, column_name;

\echo ''

-- ================================================================
-- 8. Summary
-- ================================================================
\echo '========================================'
\echo 'Verification Summary'
\echo '========================================'

DO $$
DECLARE
    rbac_tables_ok BOOLEAN;
    fk_count INT;
    idx_count INT;
BEGIN
    -- Check RBAC tables
    SELECT 
        EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'role_permissions')
        AND EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'user_roles')
    INTO rbac_tables_ok;
    
    -- Count foreign keys
    SELECT COUNT(*) INTO fk_count
    FROM information_schema.table_constraints
    WHERE constraint_type = 'FOREIGN KEY'
        AND table_name IN ('role_permissions', 'user_roles');
    
    -- Count indexes
    SELECT COUNT(*) INTO idx_count
    FROM pg_indexes
    WHERE tablename IN ('role_permissions', 'user_roles');
    
    RAISE NOTICE '';
    RAISE NOTICE 'RBAC Tables: %', CASE WHEN rbac_tables_ok THEN '✓ OK' ELSE '✗ FAILED' END;
    RAISE NOTICE 'Foreign Keys: % found', fk_count;
    RAISE NOTICE 'Indexes: % found', idx_count;
    RAISE NOTICE '';
    
    IF rbac_tables_ok AND fk_count >= 4 AND idx_count >= 6 THEN
        RAISE NOTICE '========================================';
        RAISE NOTICE '✓ Phase 1 PostgreSQL: SUCCESS';
        RAISE NOTICE '========================================';
    ELSE
        RAISE WARNING '========================================';
        RAISE WARNING '✗ Phase 1 PostgreSQL: INCOMPLETE';
        RAISE WARNING '========================================';
    END IF;
END $$;

\echo ''
\echo 'Next Steps:'
\echo '1. Verify Redis security (run: redis-cli -a PASSWORD ping)'
\echo '2. Verify SQLite fixes (check mobile app)'
\echo '3. Review QUICK_START_GUIDE.md for next phase'
\echo ''
