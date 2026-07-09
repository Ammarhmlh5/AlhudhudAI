-- ============================================================================
-- Admin Interface Migrations Verification Script
-- ============================================================================
-- This script verifies that all admin interface enhancement migrations
-- have been applied successfully and all database objects are in place.
-- ============================================================================

\echo '=== Admin Interface Migrations Verification ==='
\echo ''

-- ============================================================================
-- 1. Check if all tables exist
-- ============================================================================
\echo '1. Checking if all tables exist...'
\echo ''

SELECT 
    CASE 
        WHEN COUNT(*) = 14 THEN '✓ All 14 tables exist'
        ELSE '✗ Missing ' || (14 - COUNT(*))::text || ' tables'
    END AS table_check,
    COUNT(*) AS tables_found
FROM information_schema.tables 
WHERE table_schema = 'public' 
AND table_name IN (
    'company_ratings',
    'performance_metrics',
    'routing_configurations',
    'distribution_strategies',
    'phone_devices',
    'distribution_metrics',
    'server_health_logs',
    'alerts',
    'message_queue',
    'waiting_messages',
    'timing_violations',
    'tenant_statistics',
    'enforcement_actions',
    'tenant_timing_config'
);

\echo ''
\echo 'Detailed table list:'
SELECT 
    table_name,
    '✓' AS status
FROM information_schema.tables 
WHERE table_schema = 'public' 
AND table_name IN (
    'company_ratings',
    'performance_metrics',
    'routing_configurations',
    'distribution_strategies',
    'phone_devices',
    'distribution_metrics',
    'server_health_logs',
    'alerts',
    'message_queue',
    'waiting_messages',
    'timing_violations',
    'tenant_statistics',
    'enforcement_actions',
    'tenant_timing_config'
)
ORDER BY table_name;

-- ============================================================================
-- 2. Check indexes
-- ============================================================================
\echo ''
\echo '2. Checking indexes...'
\echo ''

SELECT 
    tablename,
    COUNT(*) AS index_count
FROM pg_indexes
WHERE schemaname = 'public'
AND tablename IN (
    'company_ratings',
    'performance_metrics',
    'routing_configurations',
    'distribution_strategies',
    'phone_devices',
    'distribution_metrics',
    'server_health_logs',
    'alerts',
    'message_queue',
    'waiting_messages',
    'timing_violations',
    'tenant_statistics',
    'enforcement_actions',
    'tenant_timing_config'
)
GROUP BY tablename
ORDER BY tablename;

-- ============================================================================
-- 3. Check RLS is enabled
-- ============================================================================
\echo ''
\echo '3. Checking Row Level Security (RLS)...'
\echo ''

SELECT 
    schemaname,
    tablename,
    CASE 
        WHEN rowsecurity THEN '✓ Enabled'
        ELSE '✗ Disabled'
    END AS rls_status
FROM pg_tables
WHERE schemaname = 'public'
AND tablename IN (
    'company_ratings',
    'performance_metrics',
    'routing_configurations',
    'distribution_strategies',
    'phone_devices',
    'distribution_metrics',
    'server_health_logs',
    'alerts',
    'message_queue',
    'waiting_messages',
    'timing_violations',
    'tenant_statistics',
    'enforcement_actions',
    'tenant_timing_config'
)
ORDER BY tablename;

-- ============================================================================
-- 4. Check RLS policies
-- ============================================================================
\echo ''
\echo '4. Checking RLS policies...'
\echo ''

SELECT 
    tablename,
    COUNT(*) AS policy_count
FROM pg_policies
WHERE schemaname = 'public'
AND tablename IN (
    'company_ratings',
    'performance_metrics',
    'routing_configurations',
    'distribution_strategies',
    'phone_devices',
    'distribution_metrics',
    'server_health_logs',
    'alerts',
    'message_queue',
    'waiting_messages',
    'timing_violations',
    'tenant_statistics',
    'enforcement_actions',
    'tenant_timing_config'
)
GROUP BY tablename
ORDER BY tablename;

-- ============================================================================
-- 5. Check constraints
-- ============================================================================
\echo ''
\echo '5. Checking constraints...'
\echo ''

SELECT 
    tc.table_name,
    tc.constraint_type,
    COUNT(*) AS constraint_count
FROM information_schema.table_constraints tc
WHERE tc.table_schema = 'public'
AND tc.table_name IN (
    'company_ratings',
    'performance_metrics',
    'routing_configurations',
    'distribution_strategies',
    'phone_devices',
    'distribution_metrics',
    'server_health_logs',
    'alerts',
    'message_queue',
    'waiting_messages',
    'timing_violations',
    'tenant_statistics',
    'enforcement_actions',
    'tenant_timing_config'
)
GROUP BY tc.table_name, tc.constraint_type
ORDER BY tc.table_name, tc.constraint_type;

-- ============================================================================
-- 6. Check foreign key relationships
-- ============================================================================
\echo ''
\echo '6. Checking foreign key relationships...'
\echo ''

SELECT 
    tc.table_name AS from_table,
    kcu.column_name AS from_column,
    ccu.table_name AS to_table,
    ccu.column_name AS to_column
FROM information_schema.table_constraints AS tc 
JOIN information_schema.key_column_usage AS kcu
    ON tc.constraint_name = kcu.constraint_name
    AND tc.table_schema = kcu.table_schema
JOIN information_schema.constraint_column_usage AS ccu
    ON ccu.constraint_name = tc.constraint_name
    AND ccu.table_schema = tc.table_schema
WHERE tc.constraint_type = 'FOREIGN KEY'
AND tc.table_schema = 'public'
AND tc.table_name IN (
    'company_ratings',
    'performance_metrics',
    'routing_configurations',
    'distribution_strategies',
    'phone_devices',
    'distribution_metrics',
    'server_health_logs',
    'alerts',
    'message_queue',
    'waiting_messages',
    'timing_violations',
    'tenant_statistics',
    'enforcement_actions',
    'tenant_timing_config'
)
ORDER BY tc.table_name, kcu.column_name;

-- ============================================================================
-- 7. Check table comments (documentation)
-- ============================================================================
\echo ''
\echo '7. Checking table documentation...'
\echo ''

SELECT 
    c.relname AS table_name,
    CASE 
        WHEN d.description IS NOT NULL THEN '✓ Documented'
        ELSE '✗ No comment'
    END AS documentation_status,
    COALESCE(LEFT(d.description, 60), 'N/A') AS description_preview
FROM pg_class c
LEFT JOIN pg_description d ON c.oid = d.objoid AND d.objsubid = 0
WHERE c.relnamespace = 'public'::regnamespace
AND c.relname IN (
    'company_ratings',
    'performance_metrics',
    'routing_configurations',
    'distribution_strategies',
    'phone_devices',
    'distribution_metrics',
    'server_health_logs',
    'alerts',
    'message_queue',
    'waiting_messages',
    'timing_violations',
    'tenant_statistics',
    'enforcement_actions',
    'tenant_timing_config'
)
ORDER BY c.relname;

-- ============================================================================
-- 8. Check column counts
-- ============================================================================
\echo ''
\echo '8. Checking column counts per table...'
\echo ''

SELECT 
    table_name,
    COUNT(*) AS column_count
FROM information_schema.columns
WHERE table_schema = 'public'
AND table_name IN (
    'company_ratings',
    'performance_metrics',
    'routing_configurations',
    'distribution_strategies',
    'phone_devices',
    'distribution_metrics',
    'server_health_logs',
    'alerts',
    'message_queue',
    'waiting_messages',
    'timing_violations',
    'tenant_statistics',
    'enforcement_actions',
    'tenant_timing_config'
)
GROUP BY table_name
ORDER BY table_name;

-- ============================================================================
-- 9. Summary statistics
-- ============================================================================
\echo ''
\echo '=== Summary Statistics ==='
\echo ''

WITH stats AS (
    SELECT 
        (SELECT COUNT(*) FROM information_schema.tables 
         WHERE table_schema = 'public' 
         AND table_name IN (
             'company_ratings', 'performance_metrics', 'routing_configurations',
             'distribution_strategies', 'phone_devices', 'distribution_metrics',
             'server_health_logs', 'alerts', 'message_queue', 'waiting_messages',
             'timing_violations', 'tenant_statistics', 'enforcement_actions',
             'tenant_timing_config'
         )) AS total_tables,
        (SELECT COUNT(*) FROM pg_indexes 
         WHERE schemaname = 'public' 
         AND tablename IN (
             'company_ratings', 'performance_metrics', 'routing_configurations',
             'distribution_strategies', 'phone_devices', 'distribution_metrics',
             'server_health_logs', 'alerts', 'message_queue', 'waiting_messages',
             'timing_violations', 'tenant_statistics', 'enforcement_actions',
             'tenant_timing_config'
         )) AS total_indexes,
        (SELECT COUNT(*) FROM pg_policies 
         WHERE schemaname = 'public' 
         AND tablename IN (
             'company_ratings', 'performance_metrics', 'routing_configurations',
             'distribution_strategies', 'phone_devices', 'distribution_metrics',
             'server_health_logs', 'alerts', 'message_queue', 'waiting_messages',
             'timing_violations', 'tenant_statistics', 'enforcement_actions',
             'tenant_timing_config'
         )) AS total_policies,
        (SELECT COUNT(*) FROM information_schema.table_constraints 
         WHERE table_schema = 'public' 
         AND table_name IN (
             'company_ratings', 'performance_metrics', 'routing_configurations',
             'distribution_strategies', 'phone_devices', 'distribution_metrics',
             'server_health_logs', 'alerts', 'message_queue', 'waiting_messages',
             'timing_violations', 'tenant_statistics', 'enforcement_actions',
             'tenant_timing_config'
         )) AS total_constraints
)
SELECT 
    'Tables Created' AS metric,
    total_tables || ' / 14' AS value,
    CASE WHEN total_tables = 14 THEN '✓' ELSE '✗' END AS status
FROM stats
UNION ALL
SELECT 
    'Indexes Created' AS metric,
    total_indexes::text AS value,
    CASE WHEN total_indexes > 0 THEN '✓' ELSE '✗' END AS status
FROM stats
UNION ALL
SELECT 
    'RLS Policies' AS metric,
    total_policies::text AS value,
    CASE WHEN total_policies > 0 THEN '✓' ELSE '✗' END AS status
FROM stats
UNION ALL
SELECT 
    'Constraints' AS metric,
    total_constraints::text AS value,
    CASE WHEN total_constraints > 0 THEN '✓' ELSE '✗' END AS status
FROM stats;

\echo ''
\echo '=== Verification Complete ==='
\echo ''
\echo 'If all checks show ✓, the migrations have been applied successfully!'
\echo 'If any checks show ✗, please review the migration files and reapply them.'
\echo ''
