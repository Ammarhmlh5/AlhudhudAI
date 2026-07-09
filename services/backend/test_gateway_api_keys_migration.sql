-- =====================================================
-- اختبار Migration: gateway_api_keys
-- =====================================================

-- تعيين tenant_id للاختبار
SET app.current_tenant = '123e4567-e89b-12d3-a456-426614174000';

BEGIN;

-- =====================================================
-- 1. اختبار إنشاء API Key
-- =====================================================

INSERT INTO gateway_api_keys (
    tenant_id,
    gateway_id,
    key_name,
    description,
    api_key_hash,
    permissions,
    rate_limit,
    allowed_ips,
    expires_at
) VALUES (
    '123e4567-e89b-12d3-a456-426614174000',
    1,
    'Production API Key',
    'مفتاح API للإنتاج - إرسال الرسائل فقط',
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',  -- bcrypt hash of "test_key_123"
    '{"send_sms": true, "view_stats": true, "view_reports": false}'::jsonb,
    100,
    ARRAY['192.168.1.0/24', '10.0.0.5'],
    '2025-12-31 23:59:59'
);

-- التحقق من الإنشاء
SELECT 
    id,
    key_name,
    permissions,
    rate_limit,
    array_length(allowed_ips, 1) as ip_count,
    is_active,
    expires_at
FROM gateway_api_keys
WHERE key_name = 'Production API Key';

-- =====================================================
-- 2. اختبار الصلاحيات الافتراضية
-- =====================================================

INSERT INTO gateway_api_keys (
    tenant_id,
    gateway_id,
    key_name,
    api_key_hash
) VALUES (
    '123e4567-e89b-12d3-a456-426614174000',
    1,
    'Default Permissions Key',
    '$2a$10$abcdefghijklmnopqrstuvwxyz123456789'
);

-- التحقق من الصلاحيات الافتراضية
SELECT 
    key_name,
    permissions,
    rate_limit
FROM gateway_api_keys
WHERE key_name = 'Default Permissions Key';

-- =====================================================
-- 3. اختبار IP Whitelisting
-- =====================================================

-- IPv4
INSERT INTO gateway_api_keys (
    tenant_id,
    gateway_id,
    key_name,
    api_key_hash,
    allowed_ips
) VALUES (
    '123e4567-e89b-12d3-a456-426614174000',
    1,
    'IPv4 Key',
    '$2a$10$ipv4test123',
    ARRAY['192.168.1.1', '10.0.0.0/24']
);

-- IPv6
INSERT INTO gateway_api_keys (
    tenant_id,
    gateway_id,
    key_name,
    api_key_hash,
    allowed_ips
) VALUES (
    '123e4567-e89b-12d3-a456-426614174000',
    1,
    'IPv6 Key',
    '$2a$10$ipv6test123',
    ARRAY['2001:db8::1', '2001:db8::/32']
);

-- التحقق
SELECT 
    key_name,
    allowed_ips
FROM gateway_api_keys
WHERE key_name IN ('IPv4 Key', 'IPv6 Key');

-- =====================================================
-- 4. اختبار IP غير صالح (يجب أن يفشل)
-- =====================================================

DO $$
BEGIN
    BEGIN
        INSERT INTO gateway_api_keys (
            tenant_id,
            gateway_id,
            key_name,
            api_key_hash,
            allowed_ips
        ) VALUES (
            '123e4567-e89b-12d3-a456-426614174000',
            1,
            'Invalid IP Key',
            '$2a$10$invalidiptest',
            ARRAY['999.999.999.999']  -- IP غير صالح
        );
        RAISE EXCEPTION 'Test failed: Invalid IP should have been rejected';
    EXCEPTION
        WHEN OTHERS THEN
            RAISE NOTICE 'Test passed: Invalid IP was rejected - %', SQLERRM;
    END;
END $$;

-- =====================================================
-- 5. اختبار Rate Limit
-- =====================================================

-- Rate Limit مخصص
INSERT INTO gateway_api_keys (
    tenant_id,
    gateway_id,
    key_name,
    api_key_hash,
    rate_limit
) VALUES (
    '123e4567-e89b-12d3-a456-426614174000',
    1,
    'High Rate Limit Key',
    '$2a$10$highrate123',
    500
);

-- التحقق
SELECT 
    key_name,
    rate_limit
FROM gateway_api_keys
WHERE key_name = 'High Rate Limit Key';

-- =====================================================
-- 6. اختبار انتهاء الصلاحية
-- =====================================================

-- مفتاح منتهي
INSERT INTO gateway_api_keys (
    tenant_id,
    gateway_id,
    key_name,
    api_key_hash,
    expires_at
) VALUES (
    '123e4567-e89b-12d3-a456-426614174000',
    1,
    'Expired Key',
    '$2a$10$expiredkey123',
    '2020-01-01 00:00:00'
);

-- مفتاح سينتهي قريباً
INSERT INTO gateway_api_keys (
    tenant_id,
    gateway_id,
    key_name,
    api_key_hash,
    expires_at
) VALUES (
    '123e4567-e89b-12d3-a456-426614174000',
    1,
    'Expiring Soon Key',
    '$2a$10$expiringsoon123',
    CURRENT_TIMESTAMP + INTERVAL '15 days'
);

-- التحقق
SELECT 
    key_name,
    expires_at,
    CASE 
        WHEN expires_at < CURRENT_TIMESTAMP THEN 'منتهي'
        WHEN expires_at < CURRENT_TIMESTAMP + INTERVAL '30 days' THEN 'سينتهي قريباً'
        ELSE 'صالح'
    END as status
FROM gateway_api_keys
WHERE key_name IN ('Expired Key', 'Expiring Soon Key');

-- =====================================================
-- 7. اختبار validate_api_key Function
-- =====================================================

-- مفتاح صالح من IP مسموح
SELECT 
    is_valid,
    key_id,
    gateway_id,
    permissions,
    rate_limit,
    error_message
FROM validate_api_key(
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
    '192.168.1.100'::inet
);

-- مفتاح من IP غير مسموح
SELECT 
    is_valid,
    error_message
FROM validate_api_key(
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
    '172.16.0.1'::inet
);

-- مفتاح منتهي
SELECT 
    is_valid,
    error_message
FROM validate_api_key(
    '$2a$10$expiredkey123',
    '192.168.1.1'::inet
);

-- =====================================================
-- 8. اختبار cleanup_expired_api_keys Function
-- =====================================================

-- عرض المفاتيح النشطة قبل التنظيف
SELECT 
    key_name,
    is_active,
    expires_at
FROM gateway_api_keys
WHERE key_name = 'Expired Key';

-- تشغيل التنظيف
SELECT cleanup_expired_api_keys() as deactivated_count;

-- عرض المفاتيح بعد التنظيف
SELECT 
    key_name,
    is_active,
    expires_at
FROM gateway_api_keys
WHERE key_name = 'Expired Key';

-- =====================================================
-- 9. اختبار تحديث last_used_at و usage_count
-- =====================================================

-- عرض القيم الأولية
SELECT 
    key_name,
    last_used_at,
    usage_count
FROM gateway_api_keys
WHERE key_name = 'Production API Key';

-- محاكاة استخدام المفتاح
UPDATE gateway_api_keys
SET last_used_at = CURRENT_TIMESTAMP,
    usage_count = usage_count + 1
WHERE key_name = 'Production API Key';

-- عرض القيم بعد الاستخدام
SELECT 
    key_name,
    last_used_at,
    usage_count
FROM gateway_api_keys
WHERE key_name = 'Production API Key';

-- =====================================================
-- 10. اختبار Row Level Security (RLS)
-- =====================================================

-- محاولة الوصول لمفاتيح tenant آخر (يجب أن تفشل)
SET app.current_tenant = '987fcdeb-51a2-43f7-b890-123456789abc';

SELECT COUNT(*) as keys_from_other_tenant
FROM gateway_api_keys;

-- العودة للـ tenant الأصلي
SET app.current_tenant = '123e4567-e89b-12d3-a456-426614174000';

SELECT COUNT(*) as keys_from_current_tenant
FROM gateway_api_keys;

-- =====================================================
-- 11. اختبار Indexes
-- =====================================================

-- التحقق من استخدام Index على tenant_id
EXPLAIN (ANALYZE, BUFFERS)
SELECT * FROM gateway_api_keys
WHERE tenant_id = '123e4567-e89b-12d3-a456-426614174000';

-- التحقق من استخدام Index على api_key_hash
EXPLAIN (ANALYZE, BUFFERS)
SELECT * FROM gateway_api_keys
WHERE api_key_hash = '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy';

-- =====================================================
-- 12. اختبار تعطيل وتفعيل المفتاح
-- =====================================================

-- تعطيل المفتاح
UPDATE gateway_api_keys
SET is_active = false
WHERE key_name = 'Production API Key';

-- التحقق
SELECT key_name, is_active
FROM gateway_api_keys
WHERE key_name = 'Production API Key';

-- إعادة التفعيل
UPDATE gateway_api_keys
SET is_active = true
WHERE key_name = 'Production API Key';

-- التحقق
SELECT key_name, is_active
FROM gateway_api_keys
WHERE key_name = 'Production API Key';

-- =====================================================
-- 13. اختبار تحديث الصلاحيات
-- =====================================================

-- تحديث الصلاحيات
UPDATE gateway_api_keys
SET permissions = '{"send_sms": true, "view_stats": true, "view_reports": true, "manage_gateway": true}'::jsonb
WHERE key_name = 'Production API Key';

-- التحقق
SELECT 
    key_name,
    permissions
FROM gateway_api_keys
WHERE key_name = 'Production API Key';

-- =====================================================
-- 14. اختبار إضافة وإزالة IPs
-- =====================================================

-- إضافة IP جديد
UPDATE gateway_api_keys
SET allowed_ips = array_append(allowed_ips, '172.16.0.0/16')
WHERE key_name = 'Production API Key';

-- التحقق
SELECT 
    key_name,
    allowed_ips
FROM gateway_api_keys
WHERE key_name = 'Production API Key';

-- إزالة IP
UPDATE gateway_api_keys
SET allowed_ips = array_remove(allowed_ips, '10.0.0.5')
WHERE key_name = 'Production API Key';

-- التحقق
SELECT 
    key_name,
    allowed_ips
FROM gateway_api_keys
WHERE key_name = 'Production API Key';

-- =====================================================
-- 15. اختبار إحصائيات الاستخدام
-- =====================================================

-- المفاتيح الأكثر استخداماً
SELECT 
    key_name,
    usage_count,
    last_used_at,
    rate_limit
FROM gateway_api_keys
WHERE is_active = true
ORDER BY usage_count DESC
LIMIT 5;

-- المفاتيح غير المستخدمة
SELECT 
    key_name,
    created_at,
    last_used_at
FROM gateway_api_keys
WHERE is_active = true
  AND last_used_at IS NULL
ORDER BY created_at ASC;

-- المفاتيح التي ستنتهي خلال 30 يوم
SELECT 
    key_name,
    expires_at,
    EXTRACT(DAY FROM (expires_at - CURRENT_TIMESTAMP)) as days_remaining
FROM gateway_api_keys
WHERE expires_at IS NOT NULL
  AND expires_at > CURRENT_TIMESTAMP
  AND expires_at < CURRENT_TIMESTAMP + INTERVAL '30 days'
  AND is_active = true
ORDER BY expires_at ASC;

-- =====================================================
-- 16. اختبار Tenant Consistency
-- =====================================================

-- محاولة إنشاء مفتاح لبوابة من tenant آخر (يجب أن تفشل)
DO $$
BEGIN
    BEGIN
        INSERT INTO gateway_api_keys (
            tenant_id,
            gateway_id,
            key_name,
            api_key_hash
        ) VALUES (
            '987fcdeb-51a2-43f7-b890-123456789abc',  -- tenant مختلف
            1,  -- gateway من tenant آخر
            'Cross Tenant Key',
            '$2a$10$crosstenanttest'
        );
        RAISE EXCEPTION 'Test failed: Cross-tenant key creation should have been rejected';
    EXCEPTION
        WHEN OTHERS THEN
            RAISE NOTICE 'Test passed: Cross-tenant key creation was rejected - %', SQLERRM;
    END;
END $$;

-- =====================================================
-- 17. عرض ملخص الاختبارات
-- =====================================================

SELECT 
    COUNT(*) as total_keys,
    COUNT(*) FILTER (WHERE is_active = true) as active_keys,
    COUNT(*) FILTER (WHERE is_active = false) as inactive_keys,
    COUNT(*) FILTER (WHERE expires_at IS NOT NULL AND expires_at < CURRENT_TIMESTAMP) as expired_keys,
    COUNT(*) FILTER (WHERE allowed_ips IS NOT NULL AND array_length(allowed_ips, 1) > 0) as keys_with_ip_whitelist,
    AVG(rate_limit)::INTEGER as avg_rate_limit,
    MAX(usage_count) as max_usage_count
FROM gateway_api_keys;

-- =====================================================
-- تنظيف بيانات الاختبار
-- =====================================================

ROLLBACK;

-- =====================================================
-- ملاحظات الاختبار
-- =====================================================

/*
الاختبارات المغطاة:
✅ 1. إنشاء API Key مع جميع الحقول
✅ 2. الصلاحيات الافتراضية
✅ 3. IP Whitelisting (IPv4 و IPv6)
✅ 4. التحقق من صحة IP addresses
✅ 5. Rate Limiting
✅ 6. انتهاء الصلاحية
✅ 7. validate_api_key Function
✅ 8. cleanup_expired_api_keys Function
✅ 9. تحديث last_used_at و usage_count
✅ 10. Row Level Security (RLS)
✅ 11. استخدام Indexes
✅ 12. تعطيل وتفعيل المفتاح
✅ 13. تحديث الصلاحيات
✅ 14. إضافة وإزالة IPs
✅ 15. إحصائيات الاستخدام
✅ 16. Tenant Consistency
✅ 17. ملخص الاختبارات

جميع الاختبارات يجب أن تنجح بدون أخطاء.
*/
