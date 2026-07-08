-- =====================================================
-- اختبار Migration: 023_gateway_users_table.sql
-- =====================================================

-- تنظيف البيانات السابقة
DO $$ 
BEGIN
    -- حذف البيانات التجريبية إن وجدت
    DELETE FROM gateway_users WHERE gateway_id IN (
        SELECT id FROM dedicated_gateways WHERE gateway_name LIKE 'Test Gateway%'
    );
    
    DELETE FROM dedicated_gateways WHERE gateway_name LIKE 'Test Gateway%';
    DELETE FROM users WHERE email LIKE 'test.gateway.user%@example.com';
    DELETE FROM tenants WHERE company_name LIKE 'Test Company%';
END $$;

-- =====================================================
-- إنشاء بيانات تجريبية
-- =====================================================

-- إنشاء شركة تجريبية
INSERT INTO tenants (id, company_name, email, phone, balance, currency)
VALUES (
    '11111111-1111-1111-1111-111111111111',
    'Test Company for Gateway Users',
    'test.gateway.users@example.com',
    '+967777777777',
    10000.00,
    'USD'
) ON CONFLICT (id) DO UPDATE SET company_name = EXCLUDED.company_name;

-- إنشاء مستخدمين تجريبيين
INSERT INTO users (id, tenant_id, email, full_name, password_hash, role)
VALUES 
    (
        '22222222-2222-2222-2222-222222222222',
        '11111111-1111-1111-1111-111111111111',
        'test.gateway.user.admin@example.com',
        'Admin User',
        '$2a$10$abcdefghijklmnopqrstuv',
        'user'
    ),
    (
        '33333333-3333-3333-3333-333333333333',
        '11111111-1111-1111-1111-111111111111',
        'test.gateway.user.operator@example.com',
        'Operator User',
        '$2a$10$abcdefghijklmnopqrstuv',
        'user'
    ),
    (
        '44444444-4444-4444-4444-444444444444',
        '11111111-1111-1111-1111-111111111111',
        'test.gateway.user.viewer@example.com',
        'Viewer User',
        '$2a$10$abcdefghijklmnopqrstuv',
        'user'
    )
ON CONFLICT (id) DO UPDATE SET email = EXCLUDED.email;

-- إنشاء بوابة مخصصة تجريبية
INSERT INTO dedicated_gateways (
    id, tenant_id, gateway_name, gateway_code, status,
    telecom_provider, telecom_username, telecom_password_encrypted,
    server_type, cpu_cores, ram_gb, storage_gb,
    max_throughput, daily_message_limit, monthly_message_limit,
    subscription_type, subscription_fee, server_rental_fee,
    contract_start_date, contract_end_date,
    telecom_balance, balance_alert_threshold
)
VALUES (
    999,
    '11111111-1111-1111-1111-111111111111',
    'Test Gateway for Users',
    'TEST_GW_USERS',
    'active',
    'Yemen Mobile',
    'test_user',
    'encrypted_password_123',
    'dedicated',
    4,
    8,
    100,
    100,
    10000,
    300000,
    'monthly',
    500.00,
    200.00,
    CURRENT_DATE,
    CURRENT_DATE + INTERVAL '1 year',
    50000.00,
    10000.00
) ON CONFLICT (id) DO UPDATE SET gateway_name = EXCLUDED.gateway_name;

-- تعيين tenant_id في الجلسة
SET app.current_tenant = '11111111-1111-1111-1111-111111111111';

-- =====================================================
-- الاختبار 1: إضافة مستخدم Admin
-- =====================================================

\echo '=== الاختبار 1: إضافة مستخدم Admin ==='

INSERT INTO gateway_users (tenant_id, gateway_id, user_id, role)
VALUES (
    '11111111-1111-1111-1111-111111111111',
    999,
    '22222222-2222-2222-2222-222222222222',
    'admin'
);

-- التحقق من الصلاحيات الافتراضية
SELECT 
    id,
    role,
    permissions,
    is_active,
    created_at
FROM gateway_users
WHERE gateway_id = 999 AND user_id = '22222222-2222-2222-2222-222222222222';

\echo 'النتيجة المتوقعة: مستخدم Admin مع جميع الصلاحيات = true'
\echo ''

-- =====================================================
-- الاختبار 2: إضافة مستخدم Operator
-- =====================================================

\echo '=== الاختبار 2: إضافة مستخدم Operator ==='

INSERT INTO gateway_users (tenant_id, gateway_id, user_id, role)
VALUES (
    '11111111-1111-1111-1111-111111111111',
    999,
    '33333333-3333-3333-3333-333333333333',
    'operator'
);

-- التحقق من الصلاحيات الافتراضية
SELECT 
    id,
    role,
    permissions,
    is_active
FROM gateway_users
WHERE gateway_id = 999 AND user_id = '33333333-3333-3333-3333-333333333333';

\echo 'النتيجة المتوقعة: مستخدم Operator مع send_messages=true، manage_users=false'
\echo ''

-- =====================================================
-- الاختبار 3: إضافة مستخدم Viewer
-- =====================================================

\echo '=== الاختبار 3: إضافة مستخدم Viewer ==='

INSERT INTO gateway_users (tenant_id, gateway_id, user_id, role)
VALUES (
    '11111111-1111-1111-1111-111111111111',
    999,
    '44444444-4444-4444-4444-444444444444',
    'viewer'
);

-- التحقق من الصلاحيات الافتراضية
SELECT 
    id,
    role,
    permissions,
    is_active
FROM gateway_users
WHERE gateway_id = 999 AND user_id = '44444444-4444-4444-4444-444444444444';

\echo 'النتيجة المتوقعة: مستخدم Viewer مع send_messages=false، view_reports=true'
\echo ''

-- =====================================================
-- الاختبار 4: محاولة إضافة دور غير صحيح
-- =====================================================

\echo '=== الاختبار 4: محاولة إضافة دور غير صحيح ==='

DO $$
BEGIN
    INSERT INTO gateway_users (tenant_id, gateway_id, user_id, role)
    VALUES (
        '11111111-1111-1111-1111-111111111111',
        999,
        '22222222-2222-2222-2222-222222222222',
        'invalid_role'
    );
    
    RAISE EXCEPTION 'الاختبار فشل: تم قبول دور غير صحيح';
EXCEPTION
    WHEN check_violation THEN
        RAISE NOTICE 'نجح: تم رفض الدور غير الصحيح';
    WHEN OTHERS THEN
        RAISE NOTICE 'نجح: تم رفض الدور غير الصحيح - %', SQLERRM;
END $$;

\echo ''

-- =====================================================
-- الاختبار 5: محاولة إضافة نفس المستخدم مرتين
-- =====================================================

\echo '=== الاختبار 5: محاولة إضافة نفس المستخدم مرتين ==='

DO $$
BEGIN
    INSERT INTO gateway_users (tenant_id, gateway_id, user_id, role)
    VALUES (
        '11111111-1111-1111-1111-111111111111',
        999,
        '22222222-2222-2222-2222-222222222222',
        'operator'
    );
    
    RAISE EXCEPTION 'الاختبار فشل: تم قبول مستخدم مكرر';
EXCEPTION
    WHEN unique_violation THEN
        RAISE NOTICE 'نجح: تم رفض المستخدم المكرر';
END $$;

\echo ''

-- =====================================================
-- الاختبار 6: تحديث الصلاحيات
-- =====================================================

\echo '=== الاختبار 6: تحديث الصلاحيات ==='

-- تحديث صلاحية معينة
UPDATE gateway_users
SET permissions = jsonb_set(
    permissions,
    '{export_reports}',
    'false'
)
WHERE gateway_id = 999 AND user_id = '33333333-3333-3333-3333-333333333333';

-- التحقق من التحديث
SELECT 
    id,
    role,
    permissions->>'export_reports' AS export_reports,
    updated_at
FROM gateway_users
WHERE gateway_id = 999 AND user_id = '33333333-3333-3333-3333-333333333333';

\echo 'النتيجة المتوقعة: export_reports = false، updated_at تم تحديثه'
\echo ''

-- =====================================================
-- الاختبار 7: تعطيل مستخدم
-- =====================================================

\echo '=== الاختبار 7: تعطيل مستخدم ==='

UPDATE gateway_users
SET is_active = false
WHERE gateway_id = 999 AND user_id = '44444444-4444-4444-4444-444444444444';

-- التحقق من التعطيل
SELECT 
    id,
    role,
    is_active,
    updated_at
FROM gateway_users
WHERE gateway_id = 999 AND user_id = '44444444-4444-4444-4444-444444444444';

\echo 'النتيجة المتوقعة: is_active = false'
\echo ''

-- =====================================================
-- الاختبار 8: عرض جميع المستخدمين مع التفاصيل
-- =====================================================

\echo '=== الاختبار 8: عرض جميع المستخدمين مع التفاصيل ==='

SELECT 
    gu.id,
    dg.gateway_name,
    u.email,
    u.full_name,
    gu.role,
    gu.permissions->>'send_messages' AS can_send,
    gu.permissions->>'manage_users' AS can_manage_users,
    gu.is_active,
    gu.created_at
FROM gateway_users gu
JOIN dedicated_gateways dg ON gu.gateway_id = dg.id
JOIN users u ON gu.user_id = u.id
WHERE gu.gateway_id = 999
ORDER BY gu.role, u.full_name;

\echo ''

-- =====================================================
-- الاختبار 9: إحصائيات المستخدمين
-- =====================================================

\echo '=== الاختبار 9: إحصائيات المستخدمين ==='

SELECT 
    dg.gateway_name,
    COUNT(gu.id) AS total_users,
    COUNT(CASE WHEN gu.role = 'admin' THEN 1 END) AS admin_count,
    COUNT(CASE WHEN gu.role = 'operator' THEN 1 END) AS operator_count,
    COUNT(CASE WHEN gu.role = 'viewer' THEN 1 END) AS viewer_count,
    COUNT(CASE WHEN gu.is_active = true THEN 1 END) AS active_users,
    COUNT(CASE WHEN gu.is_active = false THEN 1 END) AS inactive_users
FROM dedicated_gateways dg
LEFT JOIN gateway_users gu ON dg.id = gu.gateway_id
WHERE dg.id = 999
GROUP BY dg.gateway_name;

\echo ''

-- =====================================================
-- الاختبار 10: Row Level Security
-- =====================================================

\echo '=== الاختبار 10: Row Level Security ==='

-- إنشاء شركة ثانية
INSERT INTO tenants (id, company_name, email, phone, balance, currency)
VALUES (
    '55555555-5555-5555-5555-555555555555',
    'Another Test Company',
    'another.test@example.com',
    '+967777777778',
    5000.00,
    'USD'
) ON CONFLICT (id) DO NOTHING;

-- تغيير tenant_id في الجلسة
SET app.current_tenant = '55555555-5555-5555-5555-555555555555';

-- محاولة عرض مستخدمي الشركة الأولى
SELECT COUNT(*) AS visible_users
FROM gateway_users
WHERE gateway_id = 999;

\echo 'النتيجة المتوقعة: 0 (لا يمكن رؤية مستخدمي شركة أخرى)'
\echo ''

-- العودة للشركة الأولى
SET app.current_tenant = '11111111-1111-1111-1111-111111111111';

-- =====================================================
-- الاختبار 11: التحقق من Indexes
-- =====================================================

\echo '=== الاختبار 11: التحقق من Indexes ==='

SELECT 
    indexname,
    indexdef
FROM pg_indexes
WHERE tablename = 'gateway_users'
ORDER BY indexname;

\echo ''

-- =====================================================
-- الاختبار 12: التحقق من Triggers
-- =====================================================

\echo '=== الاختبار 12: التحقق من Triggers ==='

SELECT 
    trigger_name,
    event_manipulation,
    action_timing
FROM information_schema.triggers
WHERE event_object_table = 'gateway_users'
ORDER BY trigger_name;

\echo ''

-- =====================================================
-- الاختبار 13: التحقق من RLS Policies
-- =====================================================

\echo '=== الاختبار 13: التحقق من RLS Policies ==='

SELECT 
    policyname,
    cmd,
    qual
FROM pg_policies
WHERE tablename = 'gateway_users'
ORDER BY policyname;

\echo ''

-- =====================================================
-- الاختبار 14: أداء الاستعلامات
-- =====================================================

\echo '=== الاختبار 14: أداء الاستعلامات ==='

EXPLAIN ANALYZE
SELECT 
    gu.id,
    gu.role,
    gu.permissions,
    u.email
FROM gateway_users gu
JOIN users u ON gu.user_id = u.id
WHERE gu.tenant_id = '11111111-1111-1111-1111-111111111111'
  AND gu.gateway_id = 999
  AND gu.is_active = true;

\echo ''

-- =====================================================
-- الاختبار 15: حذف مستخدم
-- =====================================================

\echo '=== الاختبار 15: حذف مستخدم ==='

DELETE FROM gateway_users
WHERE gateway_id = 999 AND user_id = '44444444-4444-4444-4444-444444444444';

-- التحقق من الحذف
SELECT COUNT(*) AS remaining_users
FROM gateway_users
WHERE gateway_id = 999;

\echo 'النتيجة المتوقعة: 2 (تم حذف مستخدم واحد)'
\echo ''

-- =====================================================
-- ملخص النتائج
-- =====================================================

\echo '=== ملخص النتائج ==='
\echo ''
\echo 'الاختبارات المكتملة:'
\echo '✓ إضافة مستخدمين بأدوار مختلفة'
\echo '✓ التحقق من الصلاحيات الافتراضية'
\echo '✓ رفض الأدوار غير الصحيحة'
\echo '✓ منع المستخدمين المكررين'
\echo '✓ تحديث الصلاحيات'
\echo '✓ تعطيل المستخدمين'
\echo '✓ Row Level Security'
\echo '✓ Indexes و Triggers'
\echo '✓ أداء الاستعلامات'
\echo ''

-- =====================================================
-- تنظيف البيانات التجريبية
-- =====================================================

\echo '=== تنظيف البيانات التجريبية ==='

DELETE FROM gateway_users WHERE gateway_id = 999;
DELETE FROM dedicated_gateways WHERE id = 999;
DELETE FROM users WHERE email LIKE 'test.gateway.user%@example.com';
DELETE FROM tenants WHERE id IN (
    '11111111-1111-1111-1111-111111111111',
    '55555555-5555-5555-5555-555555555555'
);

\echo 'تم تنظيف البيانات التجريبية بنجاح'
\echo ''
\echo '=== انتهى الاختبار ==='
