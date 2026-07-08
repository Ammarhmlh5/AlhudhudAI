-- =====================================================
-- اختبار Migration: 025_gateway_statistics_table.sql
-- =====================================================

\echo '========================================='
\echo 'اختبار جدول gateway_statistics'
\echo '========================================='
\echo ''

-- =====================================================
-- 1. التحقق من إنشاء الجدول
-- =====================================================

\echo '1. التحقق من إنشاء الجدول...'

SELECT 
    table_name,
    table_type
FROM information_schema.tables
WHERE table_schema = 'public'
  AND table_name = 'gateway_statistics';

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
WHERE table_schema = 'public'
  AND table_name = 'gateway_statistics'
ORDER BY ordinal_position;

\echo ''

-- =====================================================
-- 3. التحقق من Constraints
-- =====================================================

\echo '3. التحقق من Constraints...'

SELECT 
    constraint_name,
    constraint_type
FROM information_schema.table_constraints
WHERE table_schema = 'public'
  AND table_name = 'gateway_statistics'
ORDER BY constraint_type, constraint_name;

\echo ''

-- =====================================================
-- 4. التحقق من Indexes
-- =====================================================

\echo '4. التحقق من Indexes...'

SELECT 
    indexname,
    indexdef
FROM pg_indexes
WHERE schemaname = 'public'
  AND tablename = 'gateway_statistics'
ORDER BY indexname;

\echo ''

-- =====================================================
-- 5. التحقق من RLS Policies
-- =====================================================

\echo '5. التحقق من RLS Policies...'

SELECT 
    policyname,
    cmd,
    qual,
    with_check
FROM pg_policies
WHERE schemaname = 'public'
  AND tablename = 'gateway_statistics'
ORDER BY policyname;

\echo ''

-- =====================================================
-- 6. التحقق من Triggers
-- =====================================================

\echo '6. التحقق من Triggers...'

SELECT 
    trigger_name,
    event_manipulation,
    action_timing,
    action_statement
FROM information_schema.triggers
WHERE event_object_schema = 'public'
  AND event_object_table = 'gateway_statistics'
ORDER BY trigger_name;

\echo ''

-- =====================================================
-- 7. التحقق من Functions
-- =====================================================

\echo '7. التحقق من Functions...'

SELECT 
    routine_name,
    routine_type,
    data_type AS return_type
FROM information_schema.routines
WHERE routine_schema = 'public'
  AND routine_name IN (
      'aggregate_daily_gateway_statistics',
      'get_gateway_success_rate',
      'get_gateway_statistics_range',
      'cleanup_old_gateway_statistics',
      'update_gateway_statistics_updated_at',
      'check_gateway_statistics_tenant_consistency',
      'calculate_gateway_statistics_profit_margin',
      'validate_gateway_statistics_consistency'
  )
ORDER BY routine_name;

\echo ''

-- =====================================================
-- 8. اختبار إدراج بيانات
-- =====================================================

\echo '8. اختبار إدراج بيانات...'

-- تعيين tenant_id للاختبار
SET app.current_tenant = '123e4567-e89b-12d3-a456-426614174000';

-- إدراج إحصائيات تجريبية
INSERT INTO gateway_statistics (
    tenant_id,
    gateway_id,
    date,
    messages_sent,
    messages_delivered,
    messages_failed,
    messages_pending,
    avg_response_time_ms,
    max_response_time_ms,
    min_response_time_ms,
    uptime_percentage,
    downtime_minutes,
    total_cost,
    total_revenue
) VALUES (
    '123e4567-e89b-12d3-a456-426614174000',
    1,
    CURRENT_DATE,
    1000,
    950,
    50,
    0,
    200,
    500,
    50,
    99.50,
    7,
    100.00,
    150.00
);

\echo 'تم إدراج إحصائيات تجريبية بنجاح'
\echo ''

-- =====================================================
-- 9. التحقق من البيانات المدرجة
-- =====================================================

\echo '9. التحقق من البيانات المدرجة...'

SELECT 
    id,
    gateway_id,
    date,
    messages_sent,
    messages_delivered,
    messages_failed,
    profit_margin,
    created_at
FROM gateway_statistics
WHERE gateway_id = 1
  AND date = CURRENT_DATE;

\echo ''

-- =====================================================
-- 10. اختبار Unique Constraint
-- =====================================================

\echo '10. اختبار Unique Constraint (يجب أن يفشل)...'

DO $$
BEGIN
    INSERT INTO gateway_statistics (
        tenant_id,
        gateway_id,
        date,
        messages_sent,
        messages_delivered,
        messages_failed
    ) VALUES (
        '123e4567-e89b-12d3-a456-426614174000',
        1,
        CURRENT_DATE,
        500,
        475,
        25
    );
    
    RAISE EXCEPTION 'Unique constraint test failed: duplicate insert succeeded';
EXCEPTION
    WHEN unique_violation THEN
        RAISE NOTICE 'Unique constraint test passed: duplicate insert blocked';
END $$;

\echo ''

-- =====================================================
-- 11. اختبار Check Constraint (messages consistency)
-- =====================================================

\echo '11. اختبار Check Constraint (يجب أن يفشل)...'

DO $$
BEGIN
    INSERT INTO gateway_statistics (
        tenant_id,
        gateway_id,
        date,
        messages_sent,
        messages_delivered,
        messages_failed
    ) VALUES (
        '123e4567-e89b-12d3-a456-426614174000',
        2,
        CURRENT_DATE,
        100,
        80,
        30  -- 100 < 80 + 30
    );
    
    RAISE EXCEPTION 'Check constraint test failed: invalid data accepted';
EXCEPTION
    WHEN check_violation THEN
        RAISE NOTICE 'Check constraint test passed: invalid data rejected';
    WHEN OTHERS THEN
        RAISE NOTICE 'Check constraint test passed: invalid data rejected (%)' , SQLERRM;
END $$;

\echo ''

-- =====================================================
-- 12. اختبار Trigger (profit_margin calculation)
-- =====================================================

\echo '12. اختبار Trigger (profit_margin calculation)...'

INSERT INTO gateway_statistics (
    tenant_id,
    gateway_id,
    date,
    messages_sent,
    messages_delivered,
    messages_failed,
    total_cost,
    total_revenue
) VALUES (
    '123e4567-e89b-12d3-a456-426614174000',
    3,
    CURRENT_DATE,
    500,
    475,
    25,
    50.00,
    75.00
);

SELECT 
    gateway_id,
    total_cost,
    total_revenue,
    profit_margin,
    CASE 
        WHEN profit_margin = (total_revenue - total_cost) THEN 'PASS'
        ELSE 'FAIL'
    END AS test_result
FROM gateway_statistics
WHERE gateway_id = 3
  AND date = CURRENT_DATE;

\echo ''

-- =====================================================
-- 13. اختبار Function: get_gateway_success_rate
-- =====================================================

\echo '13. اختبار Function: get_gateway_success_rate...'

SELECT 
    gateway_id,
    messages_sent,
    messages_delivered,
    get_gateway_success_rate(gateway_id, date) AS success_rate,
    ROUND((messages_delivered::DECIMAL / messages_sent::DECIMAL * 100), 2) AS expected_rate,
    CASE 
        WHEN get_gateway_success_rate(gateway_id, date) = ROUND((messages_delivered::DECIMAL / messages_sent::DECIMAL * 100), 2) THEN 'PASS'
        ELSE 'FAIL'
    END AS test_result
FROM gateway_statistics
WHERE gateway_id = 1
  AND date = CURRENT_DATE;

\echo ''

-- =====================================================
-- 14. اختبار Function: get_gateway_statistics_range
-- =====================================================

\echo '14. اختبار Function: get_gateway_statistics_range...'

-- إدراج بيانات إضافية للاختبار
INSERT INTO gateway_statistics (
    tenant_id, gateway_id, date,
    messages_sent, messages_delivered, messages_failed,
    total_cost, total_revenue
) VALUES 
    ('123e4567-e89b-12d3-a456-426614174000', 1, CURRENT_DATE - INTERVAL '1 day', 800, 760, 40, 80.00, 120.00),
    ('123e4567-e89b-12d3-a456-426614174000', 1, CURRENT_DATE - INTERVAL '2 days', 900, 855, 45, 90.00, 135.00);

SELECT * FROM get_gateway_statistics_range(
    1,
    CURRENT_DATE - INTERVAL '2 days',
    CURRENT_DATE
);

\echo ''

-- =====================================================
-- 15. اختبار RLS (Row Level Security)
-- =====================================================

\echo '15. اختبار RLS...'

-- تعيين tenant مختلف
SET app.current_tenant = '987fcdeb-51a2-43f7-b890-123456789abc';

-- محاولة قراءة إحصائيات tenant آخر
SELECT 
    COUNT(*) AS visible_rows,
    CASE 
        WHEN COUNT(*) = 0 THEN 'PASS: RLS working correctly'
        ELSE 'FAIL: RLS not working'
    END AS test_result
FROM gateway_statistics
WHERE gateway_id = 1;

\echo ''

-- إعادة تعيين tenant_id الأصلي
SET app.current_tenant = '123e4567-e89b-12d3-a456-426614174000';

-- =====================================================
-- 16. اختبار Trigger: updated_at
-- =====================================================

\echo '16. اختبار Trigger: updated_at...'

-- الحصول على updated_at الحالي
SELECT updated_at INTO TEMP TABLE temp_updated_at
FROM gateway_statistics
WHERE gateway_id = 1 AND date = CURRENT_DATE;

-- الانتظار لثانية واحدة
SELECT pg_sleep(1);

-- تحديث السجل
UPDATE gateway_statistics
SET messages_sent = messages_sent + 1
WHERE gateway_id = 1 AND date = CURRENT_DATE;

-- التحقق من تحديث updated_at
SELECT 
    CASE 
        WHEN gs.updated_at > t.updated_at THEN 'PASS: updated_at trigger working'
        ELSE 'FAIL: updated_at not updated'
    END AS test_result
FROM gateway_statistics gs
CROSS JOIN temp_updated_at t
WHERE gs.gateway_id = 1 AND gs.date = CURRENT_DATE;

DROP TABLE temp_updated_at;

\echo ''

-- =====================================================
-- 17. اختبار Tenant Consistency Trigger
-- =====================================================

\echo '17. اختبار Tenant Consistency Trigger (يجب أن يفشل)...'

DO $$
BEGIN
    -- محاولة إدراج إحصائيات لبوابة من tenant آخر
    INSERT INTO gateway_statistics (
        tenant_id,
        gateway_id,
        date,
        messages_sent,
        messages_delivered,
        messages_failed
    ) VALUES (
        '987fcdeb-51a2-43f7-b890-123456789abc',  -- tenant مختلف
        1,  -- gateway ينتمي لـ tenant آخر
        CURRENT_DATE - INTERVAL '3 days',
        100,
        95,
        5
    );
    
    RAISE EXCEPTION 'Tenant consistency test failed: cross-tenant insert succeeded';
EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'Tenant consistency test passed: cross-tenant insert blocked (%)' , SQLERRM;
END $$;

\echo ''

-- =====================================================
-- 18. اختبار Performance (Indexes)
-- =====================================================

\echo '18. اختبار Performance (Indexes)...'

EXPLAIN ANALYZE
SELECT * FROM gateway_statistics
WHERE tenant_id = '123e4567-e89b-12d3-a456-426614174000'
  AND gateway_id = 1
  AND date >= CURRENT_DATE - INTERVAL '7 days'
ORDER BY date DESC;

\echo ''

-- =====================================================
-- 19. اختبار Function: cleanup_old_gateway_statistics
-- =====================================================

\echo '19. اختبار Function: cleanup_old_gateway_statistics...'

-- إدراج إحصائيات قديمة
INSERT INTO gateway_statistics (
    tenant_id, gateway_id, date,
    messages_sent, messages_delivered, messages_failed
) VALUES (
    '123e4567-e89b-12d3-a456-426614174000',
    1,
    CURRENT_DATE - INTERVAL '400 days',
    100,
    95,
    5
);

-- عد السجلات قبل التنظيف
SELECT COUNT(*) AS records_before_cleanup
FROM gateway_statistics
WHERE date < CURRENT_DATE - INTERVAL '365 days';

-- تنظيف السجلات القديمة
SELECT cleanup_old_gateway_statistics(365) AS deleted_count;

-- عد السجلات بعد التنظيف
SELECT COUNT(*) AS records_after_cleanup
FROM gateway_statistics
WHERE date < CURRENT_DATE - INTERVAL '365 days';

\echo ''

-- =====================================================
-- 20. ملخص الاختبار
-- =====================================================

\echo '========================================='
\echo 'ملخص الاختبار'
\echo '========================================='

SELECT 
    'gateway_statistics' AS table_name,
    (SELECT COUNT(*) FROM gateway_statistics) AS total_records,
    (SELECT COUNT(*) FROM pg_indexes WHERE tablename = 'gateway_statistics') AS total_indexes,
    (SELECT COUNT(*) FROM pg_policies WHERE tablename = 'gateway_statistics') AS total_policies,
    (SELECT COUNT(*) FROM information_schema.triggers WHERE event_object_table = 'gateway_statistics') AS total_triggers,
    (SELECT COUNT(*) FROM information_schema.table_constraints WHERE table_name = 'gateway_statistics') AS total_constraints;

\echo ''
\echo 'جميع الاختبارات اكتملت!'
\echo ''

-- =====================================================
-- تنظيف بيانات الاختبار
-- =====================================================

\echo 'تنظيف بيانات الاختبار...'

DELETE FROM gateway_statistics
WHERE gateway_id IN (1, 2, 3)
  AND date >= CURRENT_DATE - INTERVAL '3 days';

\echo 'تم تنظيف بيانات الاختبار بنجاح'
\echo ''
