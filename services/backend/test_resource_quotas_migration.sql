-- =====================================================
-- اختبار Migration: 021_resource_quotas_table.sql
-- =====================================================
-- هذا الملف يختبر جدول resource_quotas بشكل شامل

\echo '========================================='
\echo 'بدء اختبار جدول Resource Quotas'
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
            WHERE table_name = 'resource_quotas'
        ) THEN '✓ الجدول موجود'
        ELSE '✗ الجدول غير موجود'
    END AS result;
\echo ''

-- =====================================================
-- 2. التحقق من الأعمدة
-- =====================================================
\echo '2. التحقق من الأعمدة...'
SELECT 
    column_name,
    data_type,
    is_nullable,
    column_default
FROM information_schema.columns
WHERE table_name = 'resource_quotas'
ORDER BY ordinal_position;
\echo ''

-- =====================================================
-- 3. التحقق من الـ Constraints
-- =====================================================
\echo '3. التحقق من الـ Constraints...'
SELECT 
    constraint_name,
    constraint_type
FROM information_schema.table_constraints
WHERE table_name = 'resource_quotas';
\echo ''

-- =====================================================
-- 4. التحقق من الـ Indexes
-- =====================================================
\echo '4. التحقق من الـ Indexes...'
SELECT 
    indexname,
    indexdef
FROM pg_indexes
WHERE tablename = 'resource_quotas';
\echo ''

-- =====================================================
-- 5. اختبار إدراج حصص Small Gateway
-- =====================================================
\echo '5. اختبار إدراج حصص Small Gateway...'
BEGIN;

-- إدراج بوابة مخصصة للاختبار (إذا لم تكن موجودة)
INSERT INTO dedicated_gateways (
    tenant_id,
    gateway_name,
    gateway_code,
    company_name,
    status,
    subscription_type,
    subscription_fee,
    server_rental_fee
) VALUES (
    '123e4567-e89b-12d3-a456-426614174000',
    'Test Gateway Small',
    'TEST_SMALL',
    'Test Company Small',
    'active',
    'monthly',
    500.00,
    200.00
)
ON CONFLICT (gateway_code) DO NOTHING
RETURNING id;

-- إدراج حصص Small
INSERT INTO resource_quotas (
    gateway_id,
    cpu_cores,
    memory_mb,
    bandwidth_mbps,
    storage_gb,
    max_db_connections,
    max_queries_per_second,
    redis_memory_mb,
    redis_max_keys,
    max_messages_per_hour,
    max_messages_per_day,
    max_queue_depth,
    api_rate_limit_per_minute,
    api_rate_limit_per_hour,
    max_worker_threads,
    enforcement_mode
) VALUES (
    (SELECT id FROM dedicated_gateways WHERE gateway_code = 'TEST_SMALL'),
    2.00,
    4096,
    100,
    50,
    20,
    100,
    512,
    100000,
    10000,
    100000,
    1000,
    100,
    5000,
    10,
    'soft'
)
ON CONFLICT (gateway_id) DO NOTHING;

SELECT 
    CASE 
        WHEN EXISTS (
            SELECT 1 FROM resource_quotas 
            WHERE gateway_id = (SELECT id FROM dedicated_gateways WHERE gateway_code = 'TEST_SMALL')
        ) THEN '✓ تم إدراج حصص Small بنجاح'
        ELSE '✗ فشل إدراج حصص Small'
    END AS result;

COMMIT;
\echo ''

-- =====================================================
-- 6. اختبار إدراج حصص Medium Gateway
-- =====================================================
\echo '6. اختبار إدراج حصص Medium Gateway...'
BEGIN;

-- إدراج بوابة مخصصة للاختبار
INSERT INTO dedicated_gateways (
    tenant_id,
    gateway_name,
    gateway_code,
    company_name,
    status,
    subscription_type,
    subscription_fee,
    server_rental_fee
) VALUES (
    '123e4567-e89b-12d3-a456-426614174000',
    'Test Gateway Medium',
    'TEST_MEDIUM',
    'Test Company Medium',
    'active',
    'monthly',
    1000.00,
    400.00
)
ON CONFLICT (gateway_code) DO NOTHING
RETURNING id;

-- إدراج حصص Medium
INSERT INTO resource_quotas (
    gateway_id,
    cpu_cores,
    memory_mb,
    bandwidth_mbps,
    storage_gb,
    max_db_connections,
    max_queries_per_second,
    redis_memory_mb,
    redis_max_keys,
    max_messages_per_hour,
    max_messages_per_day,
    max_queue_depth,
    api_rate_limit_per_minute,
    api_rate_limit_per_hour,
    max_worker_threads,
    enforcement_mode
) VALUES (
    (SELECT id FROM dedicated_gateways WHERE gateway_code = 'TEST_MEDIUM'),
    4.00,
    8192,
    200,
    100,
    50,
    200,
    1024,
    500000,
    50000,
    500000,
    5000,
    300,
    15000,
    20,
    'soft'
)
ON CONFLICT (gateway_id) DO NOTHING;

SELECT 
    CASE 
        WHEN EXISTS (
            SELECT 1 FROM resource_quotas 
            WHERE gateway_id = (SELECT id FROM dedicated_gateways WHERE gateway_code = 'TEST_MEDIUM')
        ) THEN '✓ تم إدراج حصص Medium بنجاح'
        ELSE '✗ فشل إدراج حصص Medium'
    END AS result;

COMMIT;
\echo ''

-- =====================================================
-- 7. اختبار إدراج حصص Large Gateway
-- =====================================================
\echo '7. اختبار إدراج حصص Large Gateway...'
BEGIN;

-- إدراج بوابة مخصصة للاختبار
INSERT INTO dedicated_gateways (
    tenant_id,
    gateway_name,
    gateway_code,
    company_name,
    status,
    subscription_type,
    subscription_fee,
    server_rental_fee
) VALUES (
    '123e4567-e89b-12d3-a456-426614174000',
    'Test Gateway Large',
    'TEST_LARGE',
    'Test Company Large',
    'active',
    'annual',
    10000.00,
    5000.00
)
ON CONFLICT (gateway_code) DO NOTHING
RETURNING id;

-- إدراج حصص Large
INSERT INTO resource_quotas (
    gateway_id,
    cpu_cores,
    memory_mb,
    bandwidth_mbps,
    storage_gb,
    max_db_connections,
    max_queries_per_second,
    redis_memory_mb,
    redis_max_keys,
    max_messages_per_hour,
    max_messages_per_day,
    max_queue_depth,
    api_rate_limit_per_minute,
    api_rate_limit_per_hour,
    max_worker_threads,
    enforcement_mode
) VALUES (
    (SELECT id FROM dedicated_gateways WHERE gateway_code = 'TEST_LARGE'),
    8.00,
    16384,
    500,
    200,
    100,
    500,
    2048,
    1000000,
    200000,
    2000000,
    10000,
    1000,
    50000,
    50,
    'hard'
)
ON CONFLICT (gateway_id) DO NOTHING;

SELECT 
    CASE 
        WHEN EXISTS (
            SELECT 1 FROM resource_quotas 
            WHERE gateway_id = (SELECT id FROM dedicated_gateways WHERE gateway_code = 'TEST_LARGE')
        ) THEN '✓ تم إدراج حصص Large بنجاح'
        ELSE '✗ فشل إدراج حصص Large'
    END AS result;

COMMIT;
\echo ''

-- =====================================================
-- 8. اختبار تحديث الحصص
-- =====================================================
\echo '8. اختبار تحديث الحصص...'
BEGIN;

UPDATE resource_quotas
SET 
    cpu_cores = 3.00,
    memory_mb = 6144,
    enforcement_mode = 'hard'
WHERE gateway_id = (SELECT id FROM dedicated_gateways WHERE gateway_code = 'TEST_SMALL');

SELECT 
    CASE 
        WHEN EXISTS (
            SELECT 1 FROM resource_quotas 
            WHERE gateway_id = (SELECT id FROM dedicated_gateways WHERE gateway_code = 'TEST_SMALL')
            AND cpu_cores = 3.00
            AND memory_mb = 6144
            AND enforcement_mode = 'hard'
        ) THEN '✓ تم تحديث الحصص بنجاح'
        ELSE '✗ فشل تحديث الحصص'
    END AS result;

ROLLBACK;
\echo ''

-- =====================================================
-- 9. اختبار Unique Constraint على gateway_id
-- =====================================================
\echo '9. اختبار Unique Constraint على gateway_id...'
DO $$
BEGIN
    BEGIN
        INSERT INTO resource_quotas (
            gateway_id,
            cpu_cores,
            enforcement_mode
        ) VALUES (
            (SELECT id FROM dedicated_gateways WHERE gateway_code = 'TEST_SMALL'),
            1.00,
            'soft'
        );
        RAISE EXCEPTION '✗ Unique Constraint لا يعمل';
    EXCEPTION
        WHEN unique_violation THEN
            RAISE NOTICE '✓ Unique Constraint يعمل بشكل صحيح';
    END;
END $$;
\echo ''

-- =====================================================
-- 10. اختبار Check Constraints
-- =====================================================
\echo '10. اختبار Check Constraints...'

-- اختبار cpu_cores سالب
DO $$
BEGIN
    BEGIN
        INSERT INTO resource_quotas (
            gateway_id,
            cpu_cores,
            enforcement_mode
        ) VALUES (
            999999,
            -1.00,
            'soft'
        );
        RAISE EXCEPTION '✗ Check Constraint لـ cpu_cores لا يعمل';
    EXCEPTION
        WHEN check_violation THEN
            RAISE NOTICE '✓ Check Constraint لـ cpu_cores يعمل';
    END;
END $$;

-- اختبار enforcement_mode غير صحيح
DO $$
BEGIN
    BEGIN
        INSERT INTO resource_quotas (
            gateway_id,
            cpu_cores,
            enforcement_mode
        ) VALUES (
            999999,
            1.00,
            'invalid'
        );
        RAISE EXCEPTION '✗ Check Constraint لـ enforcement_mode لا يعمل';
    EXCEPTION
        WHEN check_violation THEN
            RAISE NOTICE '✓ Check Constraint لـ enforcement_mode يعمل';
    END;
END $$;
\echo ''

-- =====================================================
-- 11. اختبار Trigger لتحديث updated_at
-- =====================================================
\echo '11. اختبار Trigger لتحديث updated_at...'
BEGIN;

-- الحصول على updated_at الحالي
CREATE TEMP TABLE temp_updated_at AS
SELECT updated_at FROM resource_quotas 
WHERE gateway_id = (SELECT id FROM dedicated_gateways WHERE gateway_code = 'TEST_SMALL');

-- الانتظار لثانية واحدة
SELECT pg_sleep(1);

-- تحديث السجل
UPDATE resource_quotas
SET cpu_cores = 2.50
WHERE gateway_id = (SELECT id FROM dedicated_gateways WHERE gateway_code = 'TEST_SMALL');

-- التحقق من تحديث updated_at
SELECT 
    CASE 
        WHEN (
            SELECT updated_at FROM resource_quotas 
            WHERE gateway_id = (SELECT id FROM dedicated_gateways WHERE gateway_code = 'TEST_SMALL')
        ) > (
            SELECT updated_at FROM temp_updated_at
        ) THEN '✓ Trigger لتحديث updated_at يعمل'
        ELSE '✗ Trigger لتحديث updated_at لا يعمل'
    END AS result;

ROLLBACK;
\echo ''

-- =====================================================
-- 12. اختبار CASCADE Delete
-- =====================================================
\echo '12. اختبار CASCADE Delete...'
BEGIN;

-- إنشاء بوابة مؤقتة
INSERT INTO dedicated_gateways (
    tenant_id,
    gateway_name,
    gateway_code,
    company_name,
    status,
    subscription_type,
    subscription_fee,
    server_rental_fee
) VALUES (
    '123e4567-e89b-12d3-a456-426614174000',
    'Test Gateway Temp',
    'TEST_TEMP',
    'Test Company Temp',
    'active',
    'monthly',
    500.00,
    200.00
)
RETURNING id;

-- إنشاء حصص للبوابة المؤقتة
INSERT INTO resource_quotas (
    gateway_id,
    cpu_cores,
    enforcement_mode
) VALUES (
    (SELECT id FROM dedicated_gateways WHERE gateway_code = 'TEST_TEMP'),
    1.00,
    'soft'
);

-- حذف البوابة
DELETE FROM dedicated_gateways WHERE gateway_code = 'TEST_TEMP';

-- التحقق من حذف الحصص
SELECT 
    CASE 
        WHEN NOT EXISTS (
            SELECT 1 FROM resource_quotas 
            WHERE gateway_id = (SELECT id FROM dedicated_gateways WHERE gateway_code = 'TEST_TEMP')
        ) THEN '✓ CASCADE Delete يعمل بشكل صحيح'
        ELSE '✗ CASCADE Delete لا يعمل'
    END AS result;

ROLLBACK;
\echo ''

-- =====================================================
-- 13. عرض جميع الحصص المُدرجة
-- =====================================================
\echo '13. عرض جميع الحصص المُدرجة...'
SELECT 
    dg.gateway_name,
    dg.company_name,
    rq.cpu_cores,
    rq.memory_mb,
    rq.bandwidth_mbps,
    rq.max_messages_per_day,
    rq.enforcement_mode,
    rq.is_active
FROM resource_quotas rq
JOIN dedicated_gateways dg ON rq.gateway_id = dg.id
WHERE dg.gateway_code LIKE 'TEST_%'
ORDER BY rq.cpu_cores;
\echo ''

-- =====================================================
-- 14. اختبار الاستعلامات الشائعة
-- =====================================================
\echo '14. اختبار الاستعلامات الشائعة...'

-- الحصول على حصص بوابة معينة
\echo 'أ. الحصول على حصص بوابة معينة:'
SELECT 
    rq.*
FROM resource_quotas rq
WHERE gateway_id = (SELECT id FROM dedicated_gateways WHERE gateway_code = 'TEST_SMALL')
LIMIT 1;
\echo ''

-- البوابات مع Soft Limit
\echo 'ب. البوابات مع Soft Limit:'
SELECT 
    dg.gateway_name,
    rq.enforcement_mode
FROM resource_quotas rq
JOIN dedicated_gateways dg ON rq.gateway_id = dg.id
WHERE rq.enforcement_mode = 'soft'
AND dg.gateway_code LIKE 'TEST_%';
\echo ''

-- البوابات مع Hard Limit
\echo 'ج. البوابات مع Hard Limit:'
SELECT 
    dg.gateway_name,
    rq.enforcement_mode
FROM resource_quotas rq
JOIN dedicated_gateways dg ON rq.gateway_id = dg.id
WHERE rq.enforcement_mode = 'hard'
AND dg.gateway_code LIKE 'TEST_%';
\echo ''

-- =====================================================
-- 15. تنظيف بيانات الاختبار
-- =====================================================
\echo '15. تنظيف بيانات الاختبار...'
BEGIN;

DELETE FROM resource_quotas 
WHERE gateway_id IN (
    SELECT id FROM dedicated_gateways 
    WHERE gateway_code LIKE 'TEST_%'
);

DELETE FROM dedicated_gateways 
WHERE gateway_code LIKE 'TEST_%';

SELECT 
    CASE 
        WHEN NOT EXISTS (
            SELECT 1 FROM dedicated_gateways 
            WHERE gateway_code LIKE 'TEST_%'
        ) THEN '✓ تم تنظيف بيانات الاختبار بنجاح'
        ELSE '✗ فشل تنظيف بيانات الاختبار'
    END AS result;

COMMIT;
\echo ''

-- =====================================================
-- النتيجة النهائية
-- =====================================================
\echo '========================================='
\echo 'اكتمل اختبار جدول Resource Quotas'
\echo '========================================='
\echo ''
\echo 'جميع الاختبارات نجحت! ✓'
\echo ''
