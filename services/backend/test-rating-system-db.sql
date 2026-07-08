-- Test Script for Company Rating System Database Schema
-- This script verifies that all required tables and indexes exist

\echo '========================================'
\echo 'Company Rating System - Database Test'
\echo '========================================'
\echo ''

-- Test 1: Check if company_ratings table exists
\echo 'Test 1: Checking company_ratings table...'
SELECT 
    CASE 
        WHEN EXISTS (
            SELECT 1 FROM information_schema.tables 
            WHERE table_name = 'company_ratings'
        ) THEN '✓ company_ratings table exists'
        ELSE '✗ company_ratings table NOT FOUND'
    END AS result;

-- Test 2: Check company_ratings columns
\echo ''
\echo 'Test 2: Checking company_ratings columns...'
SELECT 
    column_name, 
    data_type,
    is_nullable
FROM information_schema.columns
WHERE table_name = 'company_ratings'
ORDER BY ordinal_position;

-- Test 3: Check company_ratings indexes
\echo ''
\echo 'Test 3: Checking company_ratings indexes...'
SELECT 
    indexname,
    indexdef
FROM pg_indexes
WHERE tablename = 'company_ratings';

-- Test 4: Check if performance_metrics table exists
\echo ''
\echo 'Test 4: Checking performance_metrics table...'
SELECT 
    CASE 
        WHEN EXISTS (
            SELECT 1 FROM information_schema.tables 
            WHERE table_name = 'performance_metrics'
        ) THEN '✓ performance_metrics table exists'
        ELSE '✗ performance_metrics table NOT FOUND'
    END AS result;

-- Test 5: Check performance_metrics columns
\echo ''
\echo 'Test 5: Checking performance_metrics columns...'
SELECT 
    column_name, 
    data_type,
    is_nullable
FROM information_schema.columns
WHERE table_name = 'performance_metrics'
ORDER BY ordinal_position;

-- Test 6: Check performance_metrics indexes
\echo ''
\echo 'Test 6: Checking performance_metrics indexes...'
SELECT 
    indexname,
    indexdef
FROM pg_indexes
WHERE tablename = 'performance_metrics';

-- Test 7: Test inserting sample data into company_ratings
\echo ''
\echo 'Test 7: Testing insert into company_ratings...'
BEGIN;

-- Create a test tenant if not exists
INSERT INTO tenants (id, name, email, phone, password_hash, role, created_at)
VALUES (
    'test-rating-tenant-001',
    'Test Rating Company',
    'test-rating@example.com',
    '+1234567890',
    'test-hash',
    'client',
    NOW()
)
ON CONFLICT (id) DO NOTHING;

-- Insert test rating
INSERT INTO company_ratings (
    tenant_id,
    success_rate,
    sending_volume,
    payment_compliance,
    service_quality_score,
    overall_rating,
    rating_category,
    calculation_timestamp
) VALUES (
    'test-rating-tenant-001',
    95.00,
    10000,
    100.00,
    85.00,
    91.25,
    'excellent',
    NOW()
);

SELECT '✓ Successfully inserted test rating' AS result;

-- Query the inserted data
SELECT 
    tenant_id,
    success_rate,
    overall_rating,
    rating_category,
    calculation_timestamp
FROM company_ratings
WHERE tenant_id = 'test-rating-tenant-001'
ORDER BY calculation_timestamp DESC
LIMIT 1;

ROLLBACK;
\echo '✓ Test data rolled back (no permanent changes)'

-- Test 8: Test inserting sample data into performance_metrics
\echo ''
\echo 'Test 8: Testing insert into performance_metrics...'
BEGIN;

INSERT INTO performance_metrics (
    tenant_id,
    period_start,
    period_end,
    period_type,
    total_messages,
    successful_messages,
    failed_messages,
    average_delivery_time_ms,
    error_count,
    total_payments,
    on_time_payments
) VALUES (
    'test-rating-tenant-001',
    NOW() - INTERVAL '1 hour',
    NOW(),
    'hourly',
    1000,
    950,
    50,
    150,
    50,
    10,
    10
);

SELECT '✓ Successfully inserted test metrics' AS result;

-- Query the inserted data with computed columns
SELECT 
    tenant_id,
    period_type,
    total_messages,
    successful_messages,
    success_rate,
    error_rate,
    payment_compliance
FROM performance_metrics
WHERE tenant_id = 'test-rating-tenant-001'
ORDER BY period_start DESC
LIMIT 1;

ROLLBACK;
\echo '✓ Test data rolled back (no permanent changes)'

-- Test 9: Check Row Level Security
\echo ''
\echo 'Test 9: Checking Row Level Security...'
SELECT 
    schemaname,
    tablename,
    rowsecurity
FROM pg_tables
WHERE tablename IN ('company_ratings', 'performance_metrics');

-- Test 10: Summary
\echo ''
\echo '========================================'
\echo 'Test Summary'
\echo '========================================'
\echo ''
\echo 'Database Schema Status:'
SELECT 
    COUNT(*) FILTER (WHERE table_name = 'company_ratings') AS company_ratings_exists,
    COUNT(*) FILTER (WHERE table_name = 'performance_metrics') AS performance_metrics_exists
FROM information_schema.tables
WHERE table_name IN ('company_ratings', 'performance_metrics');

\echo ''
\echo 'Indexes Status:'
SELECT 
    tablename,
    COUNT(*) AS index_count
FROM pg_indexes
WHERE tablename IN ('company_ratings', 'performance_metrics')
GROUP BY tablename;

\echo ''
\echo '========================================'
\echo 'All Tests Complete!'
\echo '========================================'
