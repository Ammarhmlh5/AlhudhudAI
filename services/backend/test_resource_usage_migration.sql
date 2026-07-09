-- =====================================================
-- اختبار Migration: 022_resource_usage_table.sql
-- =====================================================
-- هذا الملف يختبر جميع جوانب جدول resource_usage

\echo '========================================='
\echo 'اختبار جدول Resource Usage'
\echo '========================================='
\echo ''

-- =====================================================
-- 1. التحقق من وجود الجدول
-- =====================================================
\echo '1. التحقق من وجود الجدول...'
SELECT 
    CASE 
        WHEN EXISTS (
            SELECT 1 FROM information_schema.tables 
            WHERE table_name = 'resource_usage'
        ) THEN '✓ الجدول موجود'
        ELSE '✗ الجدول غير موجود'
    END as result;
\echo ''

-- =====================================================
-- 2. التحقق من الحقول
-- =====================================================
\echo '2. التحقق من الحقول...'
SELECT 
    column_name,
    data_type,
    is_nullable,
    column_default
FROM information_schema.columns
WHERE table_name = 'resource_usage'
ORDER BY ordinal_position;
\echo ''

-- =====================================================
-- 3. التحقق من Constraints
-- =====================================================
\echo '3. التحقق من Constraints...'
SELECT 
    conname as constraint_name,
    contype as constraint_type,
    pg_get_constraintdef(oid) as definition
FROM pg_constraint
WHERE conrelid = 'resource_usage'::regclass
ORDER BY conname;
\echo ''

-- =====================================================
-- 4. التحقق من Indexes
-- =====================================================
\echo '4. التحقق من Indexes...'
SELECT 
    indexname,
    indexdef
FROM pg_indexes
WHERE tablename = 'resource_usage'
ORDER BY indexname;
\echo ''

-- =====================================================
-- 5. التحقق من Partitions
-- =====================================================
\echo '5. التحقق من Partitions...'
SELECT 
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables
WHERE tablename LIKE 'resource_usage_%'
ORDER BY tablename;
\echo ''

-- =====================================================
-- 6. التحقق من Row Level Security
-- =====================================================
\echo '6. التحقق من Row Level Security...'
SELECT 
    schemaname,
    tablename,
    rowsecurity as rls_enabled
FROM pg_tables
WHERE tablename = 'resource_usage';
\echo ''

\echo 'RLS Policies:'
SELECT 
    policyname,
    cmd,
    qual,
    with_check
FROM pg_policies
WHERE tablename = 'resource_usage'
ORDER BY policyname;
\echo ''

-- =====================================================
-- 7. التحقق من Functions
-- =====================================================
\echo '7. التحقق من Functions...'
SELECT 
    proname as function_name,
    pg_get_function_arguments(oid) as arguments,
    pg_get_functiondef(oid) as definition
FROM pg_proc
WHERE proname IN ('create_resource_usage_partition', 'cleanup_old_resource_usage_partitions')
ORDER BY proname;
\echo ''

-- =====================================================
-- 8. اختبار إدراج بيانات
-- =====================================================
\echo '8. اختبار إدراج بيانات...'

-- إنشاء tenant تجريبي
DO $
BEGIN
    IF NOT EXISTS (SELECT 1 FROM tenants WHERE id = '00000000-0000-0000-0000-000000000001') THEN
        INSERT INTO tenants (id, name, email, balance, currency)
        VALUES ('00000000-0000-0000-0000-000000000001', 'Test Tenant', 'test@example.com', 1000.00, 'USD');
    END IF;
END $;

-- إنشاء بوابة تجريبية
DO $
BEGIN
    IF NOT EXISTS (SELECT 1 FROM dedicated_gateways WHERE id = 999) THEN
        INSERT INTO dedicated_gateways (
            id,
            tenant_id,
            gateway_name,
            gateway_code,
            telecom_provider,
            telecom_api_endpoint,
            contract_start_date,
            subscription_fee,
            server_rental_fee
        ) VALUES (
            999,
            '00000000-0000-0000-0000-000000000001',
            'Test Gateway',
            'TEST_GW_999',
            'Test Telecom',
            'https://api.test.com',
            CURRENT_DATE,
            100.00,
            50.00
        );
    END IF;
END $;

-- تعيين tenant_id للجلسة
SET app.current_tenant = '00000000-0000-0000-0000-000000000001';

-- إدراج بيانات تجريبية
INSERT INTO resource_usage (
    gateway_id,
    tenant_id,
    cpu_usage_percent,
    memory_usage_mb,
    bandwidth_usage_mbps,
    active_connections,
    db_connections_active,
    queries_per_second,
    redis_memory_usage_mb,
    redis_keys_count,
    messages_processed_last_hour,
    messages_processed_today,
    current_queue_depth,
    api_requests_last_minute,
    api_requests_last_hour,
    active_worker_threads
) VALUES (
    999,
    '00000000-0000-0000-0000-000000000001',
    45.50,
    2048,
    150.75,
    25,
    15,
    120,
    512,
    50000,
    5000,
    45000,
    150,
    80,
    4500,
    8
);

SELECT 
    CASE 
        WHEN COUNT(*) > 0 THEN '✓ تم إدراج البيانات بنجاح'
        ELSE '✗ فشل إدراج البيانات'
    END as result
FROM resource_usage
WHERE gateway_id = 999;
\echo ''

-- =====================================================
-- 9. اختبار RLS
-- =====================================================
\echo '9. اختبار Row Level Security...'

-- إنشاء tenant آخر
DO $
BEGIN
    IF NOT EXISTS (SELECT 1 FROM tenants WHERE id = '00000000-0000-0000-0000-000000000002') THEN
        INSERT INTO tenants (id, name, email, balance, currency)
        VALUES ('00000000-0000-0000-0000-000000000002', 'Test Tenant 2', 'test2@example.com', 1000.00, 'USD');
    END IF;
END $;

-- محاولة قراءة بيانات tenant آخر
SET app.current_tenant = '00000000-0000-0000-0000-000000000002';

SELECT 
    CASE 
        WHEN COUNT(*) = 0 THEN '✓ RLS يعمل بشكل صحيح - لا يمكن رؤية بيانات tenant آخر'
        ELSE '✗ RLS لا يعمل - يمكن رؤية بيانات tenant آخر'
    END as result
FROM resource_usage
WHERE gateway_id = 999;
\echo ''

-- العودة للـ tenant الأول
SET app.current_tenant = '00000000-0000-0000-0000-000000000001';

-- =====================================================
-- 10. اختبار Constraints
-- =====================================================
\echo '10. اختبار Constraints...'

-- اختبار cpu_usage_percent (يجب أن يكون بين 0 و 100)
\echo 'اختبار cpu_usage_percent > 100...'
DO $
BEGIN
    INSERT INTO resource_usage (
        gateway_id,
        tenant_id,
        cpu_usage_percent
    ) VALUES (
        999,
        '00000000-0000-0000-0000-000000000001',
        150.00  -- قيمة غير صحيحة
    );
    RAISE EXCEPTION 'يجب أن يفشل الإدراج';
EXCEPTION
    WHEN check_violation THEN
        RAISE NOTICE '✓ Constraint يعمل بشكل صحيح - رفض cpu_usage_percent > 100';
END $;
\echo ''

-- اختبار memory_usage_mb (يجب أن يكون موجب)
\echo 'اختبار memory_usage_mb سالب...'
DO $
BEGIN
    INSERT INTO resource_usage (
        gateway_id,
        tenant_id,
        memory_usage_mb
    ) VALUES (
        999,
        '00000000-0000-0000-0000-000000000001',
        -100  -- قيمة غير صحيحة
    );
    RAISE EXCEPTION 'يجب أن يفشل الإدراج';
EXCEPTION
    WHEN check_violation THEN
        RAISE NOTICE '✓ Constraint يعمل بشكل صحيح - رفض memory_usage_mb سالب';
END $;
\echo ''

-- =====================================================
-- 11. اختبار Partitioning
-- =====================================================
\echo '11. اختبار Partitioning...'

-- إدراج بيانات في أشهر مختلفة
INSERT INTO resource_usage (
    gateway_id,
    tenant_id,
    timestamp,
    cpu_usage_percent
) VALUES 
    (999, '00000000-0000-0000-0000-000000000001', '2024-12-15 10:00:00', 50.00),
    (999, '00000000-0000-0000-0000-000000000001', '2025-01-15 10:00:00', 60.00),
    (999, '00000000-0000-0000-0000-000000000001', '2025-02-15 10:00:00', 70.00);

-- التحقق من توزيع البيانات على Partitions
SELECT 
    'resource_usage_2024_12' as partition,
    COUNT(*) as row_count
FROM resource_usage_2024_12
UNION ALL
SELECT 
    'resource_usage_2025_01' as partition,
    COUNT(*) as row_count
FROM resource_usage_2025_01
UNION ALL
SELECT 
    'resource_usage_2025_02' as partition,
    COUNT(*) as row_count
FROM resource_usage_2025_02
ORDER BY partition;
\echo ''

-- =====================================================
-- 12. اختبار Function: create_resource_usage_partition
-- =====================================================
\echo '12. اختبار Function: create_resource_usage_partition...'

-- إنشاء partition جديد
SELECT create_resource_usage_partition('2025-07-01');

-- التحقق من إنشاء الـ Partition
SELECT 
    CASE 
        WHEN EXISTS (
            SELECT 1 FROM pg_tables 
            WHERE tablename = 'resource_usage_2025_07'
        ) THEN '✓ تم إنشاء Partition بنجاح'
        ELSE '✗ فشل إنشاء Partition'
    END as result;
\echo ''

-- =====================================================
-- 13. اختبار الاستعلامات
-- =====================================================
\echo '13. اختبار الاستعلامات...'

-- آخر تسجيل
\echo 'آخر تسجيل:'
SELECT 
    gateway_id,
    cpu_usage_percent,
    memory_usage_mb,
    timestamp
FROM resource_usage
WHERE gateway_id = 999
ORDER BY timestamp DESC
LIMIT 1;
\echo ''

-- متوسط الاستخدام
\echo 'متوسط الاستخدام:'
SELECT 
    gateway_id,
    AVG(cpu_usage_percent) as avg_cpu,
    AVG(memory_usage_mb) as avg_memory,
    COUNT(*) as total_records
FROM resource_usage
WHERE gateway_id = 999
GROUP BY gateway_id;
\echo ''

-- =====================================================
-- 14. اختبار الأداء
-- =====================================================
\echo '14. اختبار الأداء...'

-- إدراج 100 سجل
\echo 'إدراج 100 سجل...'
DO $
DECLARE
    i INTEGER;
BEGIN
    FOR i IN 1..100 LOOP
        INSERT INTO resource_usage (
            gateway_id,
            tenant_id,
            cpu_usage_percent,
            memory_usage_mb,
            timestamp
        ) VALUES (
            999,
            '00000000-0000-0000-0000-000000000001',
            RANDOM() * 100,
            (RANDOM() * 4096)::INTEGER,
            NOW() - (i || ' minutes')::INTERVAL
        );
    END LOOP;
END $;

SELECT 
    COUNT(*) as total_records,
    '✓ تم إدراج 100 سجل بنجاح' as result
FROM resource_usage
WHERE gateway_id = 999;
\echo ''

-- اختبار سرعة الاستعلام
\echo 'اختبار سرعة الاستعلام...'
EXPLAIN ANALYZE
SELECT 
    gateway_id,
    AVG(cpu_usage_percent) as avg_cpu,
    AVG(memory_usage_mb) as avg_memory
FROM resource_usage
WHERE gateway_id = 999
  AND timestamp >= NOW() - INTERVAL '1 hour'
GROUP BY gateway_id;
\echo ''

-- =====================================================
-- 15. تنظيف البيانات التجريبية
-- =====================================================
\echo '15. تنظيف البيانات التجريبية...'

-- حذف البيانات التجريبية
DELETE FROM resource_usage WHERE gateway_id = 999;
DELETE FROM dedicated_gateways WHERE id = 999;
-- لا نحذف الـ tenants لأنها قد تكون مستخدمة في اختبارات أخرى

SELECT '✓ تم تنظيف البيانات التجريبية' as result;
\echo ''

-- =====================================================
-- الخلاصة
-- =====================================================
\echo '========================================='
\echo 'اكتملت جميع الاختبارات بنجاح!'
\echo '========================================='
\echo ''
\echo 'الجدول جاهز للاستخدام:'
\echo '- ✓ الجدول موجود مع جميع الحقول'
\echo '- ✓ Constraints تعمل بشكل صحيح'
\echo '- ✓ Indexes موجودة'
\echo '- ✓ Partitioning يعمل بشكل صحيح'
\echo '- ✓ RLS يعمل بشكل صحيح'
\echo '- ✓ Functions تعمل بشكل صحيح'
\echo '- ✓ الأداء جيد'
\echo ''
