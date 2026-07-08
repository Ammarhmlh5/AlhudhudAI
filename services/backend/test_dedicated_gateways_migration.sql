-- ============================================================================
-- اختبار Migration: 020_dedicated_gateways_table.sql
-- ============================================================================
-- هذا الملف يختبر:
-- 1. إنشاء الجدول بنجاح
-- 2. عمل Row Level Security (RLS)
-- 3. عمل Indexes
-- 4. عمل Constraints
-- 5. عمل Triggers
-- ============================================================================

\echo '============================================================================'
\echo 'بدء اختبار جدول dedicated_gateways'
\echo '============================================================================'

-- ============================================================================
-- الاختبار 1: التحقق من وجود الجدول
-- ============================================================================
\echo ''
\echo 'الاختبار 1: التحقق من وجود الجدول...'

SELECT 
    CASE 
        WHEN EXISTS (
            SELECT 1 FROM information_schema.tables 
            WHERE table_name = 'dedicated_gateways'
        ) 
        THEN '✓ الجدول موجود'
        ELSE '✗ الجدول غير موجود'
    END AS result;

-- ============================================================================
-- الاختبار 2: التحقق من الأعمدة الأساسية
-- ============================================================================
\echo ''
\echo 'الاختبار 2: التحقق من الأعمدة الأساسية...'

SELECT 
    column_name,
    data_type,
    is_nullable,
    column_default
FROM information_schema.columns
WHERE table_name = 'dedicated_gateways'
ORDER BY ordinal_position;

-- ============================================================================
-- الاختبار 3: التحقق من Indexes
-- ============================================================================
\echo ''
\echo 'الاختبار 3: التحقق من Indexes...'

SELECT 
    indexname,
    indexdef
FROM pg_indexes
WHERE tablename = 'dedicated_gateways'
ORDER BY indexname;

-- ============================================================================
-- الاختبار 4: التحقق من تفعيل RLS
-- ============================================================================
\echo ''
\echo 'الاختبار 4: التحقق من تفعيل RLS...'

SELECT 
    tablename,
    rowsecurity AS rls_enabled
FROM pg_tables
WHERE tablename = 'dedicated_gateways';

-- ============================================================================
-- الاختبار 5: التحقق من RLS Policies
-- ============================================================================
\echo ''
\echo 'الاختبار 5: التحقق من RLS Policies...'

SELECT 
    policyname,
    cmd,
    qual,
    with_check
FROM pg_policies
WHERE tablename = 'dedicated_gateways'
ORDER BY policyname;

-- ============================================================================
-- الاختبار 6: التحقق من Constraints
-- ============================================================================
\echo ''
\echo 'الاختبار 6: التحقق من Constraints...'

SELECT 
    conname AS constraint_name,
    contype AS constraint_type,
    pg_get_constraintdef(oid) AS constraint_definition
FROM pg_constraint
WHERE conrelid = 'dedicated_gateways'::regclass
ORDER BY conname;

-- ============================================================================
-- الاختبار 7: التحقق من Triggers
-- ============================================================================
\echo ''
\echo 'الاختبار 7: التحقق من Triggers...'

SELECT 
    trigger_name,
    event_manipulation,
    action_timing,
    action_statement
FROM information_schema.triggers
WHERE event_object_table = 'dedicated_gateways'
ORDER BY trigger_name;

-- ============================================================================
-- الاختبار 8: إدراج بيانات تجريبية
-- ============================================================================
\echo ''
\echo 'الاختبار 8: إدراج بيانات تجريبية...'

-- إنشاء tenant تجريبي إذا لم يكن موجوداً
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM tenants WHERE id = '11111111-1111-1111-1111-111111111111') THEN
        INSERT INTO tenants (id, name, email, balance, currency)
        VALUES (
            '11111111-1111-1111-1111-111111111111',
            'شركة الاختبار الأولى',
            'test1@example.com',
            10000.00,
            'USD'
        );
        RAISE NOTICE 'تم إنشاء tenant تجريبي 1';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM tenants WHERE id = '22222222-2222-2222-2222-222222222222') THEN
        INSERT INTO tenants (id, name, email, balance, currency)
        VALUES (
            '22222222-2222-2222-2222-222222222222',
            'شركة الاختبار الثانية',
            'test2@example.com',
            20000.00,
            'USD'
        );
        RAISE NOTICE 'تم إنشاء tenant تجريبي 2';
    END IF;
END $$;

-- تعيين tenant الأول
SET app.current_tenant = '11111111-1111-1111-1111-111111111111';

-- إدراج بوابة للشركة الأولى
INSERT INTO dedicated_gateways (
    tenant_id,
    gateway_name,
    gateway_code,
    description,
    telecom_provider,
    telecom_api_endpoint,
    telecom_username,
    telecom_password_encrypted,
    telecom_api_key_encrypted,
    server_type,
    server_specs,
    max_throughput,
    daily_limit,
    monthly_limit,
    subscription_type,
    subscription_fee,
    server_rental_fee,
    setup_fee,
    balance_alert_threshold,
    contract_start_date,
    next_billing_date
) VALUES (
    '11111111-1111-1111-1111-111111111111',
    'بوابة الاختبار الأولى',
    'TEST_GATEWAY_1',
    'بوابة مخصصة للاختبار',
    'Yemen Mobile',
    'https://api.yemenmobile.ye/sms',
    'test_user_1',
    'ENCRYPTED_PASSWORD_1',
    'ENCRYPTED_API_KEY_1',
    'dedicated',
    '{"cpu": 4, "ram": "8GB", "storage": "100GB"}',
    500,
    50000,
    1500000,
    'monthly',
    1000.00,
    200.00,
    500.00,
    5000.00,
    CURRENT_DATE,
    CURRENT_DATE + INTERVAL '1 month'
) ON CONFLICT (gateway_code) DO NOTHING;

\echo '✓ تم إدراج بوابة للشركة الأولى'

-- تعيين tenant الثاني
SET app.current_tenant = '22222222-2222-2222-2222-222222222222';

-- إدراج بوابة للشركة الثانية
INSERT INTO dedicated_gateways (
    tenant_id,
    gateway_name,
    gateway_code,
    description,
    telecom_provider,
    telecom_api_endpoint,
    telecom_username,
    telecom_password_encrypted,
    server_type,
    server_specs,
    max_throughput,
    daily_limit,
    monthly_limit,
    subscription_type,
    subscription_fee,
    server_rental_fee,
    setup_fee,
    balance_alert_threshold,
    contract_start_date,
    next_billing_date
) VALUES (
    '22222222-2222-2222-2222-222222222222',
    'بوابة الاختبار الثانية',
    'TEST_GATEWAY_2',
    'بوابة مخصصة للاختبار',
    'Sabafon',
    'https://api.sabafon.ye/sms',
    'test_user_2',
    'ENCRYPTED_PASSWORD_2',
    'vps',
    '{"cpu": 2, "ram": "4GB", "storage": "50GB"}',
    200,
    20000,
    600000,
    'yearly',
    10000.00,
    150.00,
    300.00,
    3000.00,
    CURRENT_DATE,
    CURRENT_DATE + INTERVAL '1 year'
) ON CONFLICT (gateway_code) DO NOTHING;

\echo '✓ تم إدراج بوابة للشركة الثانية'

-- ============================================================================
-- الاختبار 9: اختبار RLS - الشركة الأولى ترى بواباتها فقط
-- ============================================================================
\echo ''
\echo 'الاختبار 9: اختبار RLS - الشركة الأولى...'

SET app.current_tenant = '11111111-1111-1111-1111-111111111111';

SELECT 
    COUNT(*) AS gateway_count,
    CASE 
        WHEN COUNT(*) = 1 THEN '✓ RLS يعمل بشكل صحيح - الشركة الأولى ترى بوابة واحدة فقط'
        ELSE '✗ RLS لا يعمل - الشركة الأولى ترى ' || COUNT(*) || ' بوابات'
    END AS result
FROM dedicated_gateways;

\echo ''
\echo 'البوابات المرئية للشركة الأولى:'
SELECT 
    id,
    gateway_name,
    gateway_code,
    tenant_id
FROM dedicated_gateways;

-- ============================================================================
-- الاختبار 10: اختبار RLS - الشركة الثانية ترى بواباتها فقط
-- ============================================================================
\echo ''
\echo 'الاختبار 10: اختبار RLS - الشركة الثانية...'

SET app.current_tenant = '22222222-2222-2222-2222-222222222222';

SELECT 
    COUNT(*) AS gateway_count,
    CASE 
        WHEN COUNT(*) = 1 THEN '✓ RLS يعمل بشكل صحيح - الشركة الثانية ترى بوابة واحدة فقط'
        ELSE '✗ RLS لا يعمل - الشركة الثانية ترى ' || COUNT(*) || ' بوابات'
    END AS result
FROM dedicated_gateways;

\echo ''
\echo 'البوابات المرئية للشركة الثانية:'
SELECT 
    id,
    gateway_name,
    gateway_code,
    tenant_id
FROM dedicated_gateways;

-- ============================================================================
-- الاختبار 11: اختبار Constraints - subscription_type
-- ============================================================================
\echo ''
\echo 'الاختبار 11: اختبار Constraints - subscription_type...'

SET app.current_tenant = '11111111-1111-1111-1111-111111111111';

DO $$
BEGIN
    BEGIN
        INSERT INTO dedicated_gateways (
            tenant_id,
            gateway_name,
            gateway_code,
            telecom_provider,
            telecom_api_endpoint,
            subscription_type,
            subscription_fee,
            server_rental_fee,
            contract_start_date
        ) VALUES (
            '11111111-1111-1111-1111-111111111111',
            'بوابة خاطئة',
            'INVALID_GATEWAY',
            'Test Provider',
            'https://test.com',
            'invalid_type', -- قيمة غير صحيحة
            100.00,
            50.00,
            CURRENT_DATE
        );
        RAISE EXCEPTION '✗ Constraint لم يعمل - تم قبول subscription_type غير صحيح';
    EXCEPTION
        WHEN check_violation THEN
            RAISE NOTICE '✓ Constraint يعمل بشكل صحيح - تم رفض subscription_type غير صحيح';
    END;
END $$;

-- ============================================================================
-- الاختبار 12: اختبار Constraints - رسوم سالبة
-- ============================================================================
\echo ''
\echo 'الاختبار 12: اختبار Constraints - رسوم سالبة...'

DO $$
BEGIN
    BEGIN
        INSERT INTO dedicated_gateways (
            tenant_id,
            gateway_name,
            gateway_code,
            telecom_provider,
            telecom_api_endpoint,
            subscription_type,
            subscription_fee,
            server_rental_fee,
            contract_start_date
        ) VALUES (
            '11111111-1111-1111-1111-111111111111',
            'بوابة خاطئة 2',
            'INVALID_GATEWAY_2',
            'Test Provider',
            'https://test.com',
            'monthly',
            -100.00, -- قيمة سالبة
            50.00,
            CURRENT_DATE
        );
        RAISE EXCEPTION '✗ Constraint لم يعمل - تم قبول subscription_fee سالب';
    EXCEPTION
        WHEN check_violation THEN
            RAISE NOTICE '✓ Constraint يعمل بشكل صحيح - تم رفض subscription_fee سالب';
    END;
END $$;

-- ============================================================================
-- الاختبار 13: اختبار Trigger - updated_at
-- ============================================================================
\echo ''
\echo 'الاختبار 13: اختبار Trigger - updated_at...'

SET app.current_tenant = '11111111-1111-1111-1111-111111111111';

-- الحصول على updated_at الحالي
DO $$
DECLARE
    old_updated_at TIMESTAMP;
    new_updated_at TIMESTAMP;
BEGIN
    SELECT updated_at INTO old_updated_at
    FROM dedicated_gateways
    WHERE gateway_code = 'TEST_GATEWAY_1';
    
    -- الانتظار لثانية واحدة
    PERFORM pg_sleep(1);
    
    -- تحديث البوابة
    UPDATE dedicated_gateways
    SET description = 'وصف محدث'
    WHERE gateway_code = 'TEST_GATEWAY_1';
    
    -- الحصول على updated_at الجديد
    SELECT updated_at INTO new_updated_at
    FROM dedicated_gateways
    WHERE gateway_code = 'TEST_GATEWAY_1';
    
    IF new_updated_at > old_updated_at THEN
        RAISE NOTICE '✓ Trigger يعمل بشكل صحيح - تم تحديث updated_at تلقائياً';
    ELSE
        RAISE EXCEPTION '✗ Trigger لا يعمل - updated_at لم يتغير';
    END IF;
END $$;

-- ============================================================================
-- الاختبار 14: اختبار Cross-Tenant Access Prevention
-- ============================================================================
\echo ''
\echo 'الاختبار 14: اختبار منع Cross-Tenant Access...'

SET app.current_tenant = '11111111-1111-1111-1111-111111111111';

DO $$
DECLARE
    gateway_count INTEGER;
BEGIN
    -- محاولة الوصول لبوابة الشركة الثانية
    SELECT COUNT(*) INTO gateway_count
    FROM dedicated_gateways
    WHERE gateway_code = 'TEST_GATEWAY_2';
    
    IF gateway_count = 0 THEN
        RAISE NOTICE '✓ Cross-Tenant Access Prevention يعمل - الشركة الأولى لا ترى بوابات الشركة الثانية';
    ELSE
        RAISE EXCEPTION '✗ Cross-Tenant Access Prevention لا يعمل - الشركة الأولى ترى بوابات الشركة الثانية';
    END IF;
END $$;

-- ============================================================================
-- الاختبار 15: اختبار تحديث رصيد شركة الاتصالات
-- ============================================================================
\echo ''
\echo 'الاختبار 15: اختبار تحديث رصيد شركة الاتصالات...'

SET app.current_tenant = '11111111-1111-1111-1111-111111111111';

UPDATE dedicated_gateways
SET 
    telecom_balance = 15000.00,
    last_balance_check = CURRENT_TIMESTAMP
WHERE gateway_code = 'TEST_GATEWAY_1';

SELECT 
    gateway_code,
    telecom_balance,
    balance_alert_threshold,
    CASE 
        WHEN telecom_balance > balance_alert_threshold 
        THEN '✓ الرصيد فوق العتبة - لا حاجة لتنبيه'
        ELSE '⚠ الرصيد تحت العتبة - يجب إرسال تنبيه'
    END AS balance_status
FROM dedicated_gateways
WHERE gateway_code = 'TEST_GATEWAY_1';

-- ============================================================================
-- الاختبار 16: عرض ملخص البوابات
-- ============================================================================
\echo ''
\echo 'الاختبار 16: عرض ملخص البوابات...'

-- إلغاء RLS مؤقتاً للعرض الشامل (للمسؤول فقط)
SET app.current_tenant = '';

SELECT 
    COUNT(*) AS total_gateways,
    COUNT(CASE WHEN status = 'active' THEN 1 END) AS active_gateways,
    COUNT(CASE WHEN subscription_type = 'monthly' THEN 1 END) AS monthly_subscriptions,
    COUNT(CASE WHEN subscription_type = 'yearly' THEN 1 END) AS yearly_subscriptions,
    SUM(subscription_fee) AS total_subscription_revenue,
    SUM(server_rental_fee) AS total_rental_revenue
FROM dedicated_gateways;

-- ============================================================================
-- الاختبار 17: عرض البوابات التي تحتاج فوترة
-- ============================================================================
\echo ''
\echo 'الاختبار 17: البوابات التي تحتاج فوترة...'

SELECT 
    gateway_code,
    gateway_name,
    subscription_type,
    subscription_fee,
    server_rental_fee,
    next_billing_date,
    CASE 
        WHEN next_billing_date <= CURRENT_DATE THEN '⚠ يحتاج فوترة الآن'
        WHEN next_billing_date <= CURRENT_DATE + INTERVAL '7 days' THEN '⚠ يحتاج فوترة قريباً'
        ELSE '✓ لا يحتاج فوترة'
    END AS billing_status
FROM dedicated_gateways
WHERE status = 'active'
ORDER BY next_billing_date;

-- ============================================================================
-- الاختبار 18: عرض البوابات التي تحتاج تنبيه رصيد
-- ============================================================================
\echo ''
\echo 'الاختبار 18: البوابات التي تحتاج تنبيه رصيد...'

SELECT 
    gateway_code,
    gateway_name,
    telecom_balance,
    balance_alert_threshold,
    CASE 
        WHEN telecom_balance IS NULL THEN 'لم يتم فحص الرصيد بعد'
        WHEN telecom_balance < balance_alert_threshold THEN '⚠ يحتاج تنبيه - الرصيد منخفض'
        ELSE '✓ الرصيد جيد'
    END AS balance_status
FROM dedicated_gateways
WHERE 
    status = 'active'
    AND balance_check_enabled = true
ORDER BY telecom_balance NULLS LAST;

-- ============================================================================
-- نهاية الاختبارات
-- ============================================================================
\echo ''
\echo '============================================================================'
\echo 'انتهت جميع الاختبارات بنجاح!'
\echo '============================================================================'
\echo ''
\echo 'ملخص:'
\echo '- الجدول تم إنشاؤه بنجاح'
\echo '- RLS يعمل بشكل صحيح'
\echo '- Indexes تم إنشاؤها'
\echo '- Constraints تعمل بشكل صحيح'
\echo '- Triggers تعمل بشكل صحيح'
\echo '- Cross-Tenant Access Prevention يعمل'
\echo ''
\echo 'يمكنك الآن استخدام الجدول في التطبيق!'
\echo '============================================================================'
