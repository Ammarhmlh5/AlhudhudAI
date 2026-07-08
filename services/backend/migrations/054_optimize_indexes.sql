-- Migration: 054_optimize_indexes.sql
-- Description: إضافة composite indexes لتحسين أداء الاستعلامات الشائعة
-- Date: 2026-03-22
-- Priority: MEDIUM

BEGIN;

-- ================================================================
-- 1. Indexes على جدول messages
-- ================================================================

-- Index للاستعلامات حسب tenant + status + created_at
CREATE INDEX IF NOT EXISTS idx_messages_tenant_status_created
    ON messages(tenant_id, status, created_at DESC);

COMMENT ON INDEX idx_messages_tenant_status_created IS
    'تحسين استعلامات الرسائل حسب المستأجر والحالة والتاريخ';

-- Index للاستعلامات حسب tenant + created_at فقط
CREATE INDEX IF NOT EXISTS idx_messages_tenant_created
    ON messages(tenant_id, created_at DESC);

COMMENT ON INDEX idx_messages_tenant_created IS
    'تحسين استعلامات الرسائل حسب المستأجر والتاريخ';

-- Index للاستعلامات حسب channel_id + status
CREATE INDEX IF NOT EXISTS idx_messages_channel_status
    ON messages(channel_id, status)
    WHERE channel_id IS NOT NULL;

COMMENT ON INDEX idx_messages_channel_status IS
    'تحسين استعلامات الرسائل حسب القناة والحالة';

-- ================================================================
-- 2. Indexes على جدول staff_activities
-- ================================================================

-- Index للاستعلامات حسب tenant + created_at
CREATE INDEX IF NOT EXISTS idx_staff_activities_tenant_created
    ON staff_activities(tenant_id, created_at DESC);

COMMENT ON INDEX idx_staff_activities_tenant_created IS
    'تحسين استعلامات نشاطات الموظفين حسب المستأجر والتاريخ';

-- Index للاستعلامات حسب staff_id + created_at
CREATE INDEX IF NOT EXISTS idx_staff_activities_staff_created
    ON staff_activities(staff_id, created_at DESC)
    WHERE staff_id IS NOT NULL;

COMMENT ON INDEX idx_staff_activities_staff_created IS
    'تحسين استعلامات نشاطات موظف معين حسب التاريخ';

-- ================================================================
-- 3. Indexes على جدول sessions
-- ================================================================

-- Index للاستعلامات حسب user_id + expires_at
CREATE INDEX IF NOT EXISTS idx_sessions_user_expires
    ON sessions(user_id, expires_at DESC);

COMMENT ON INDEX idx_sessions_user_expires IS
    'تحسين استعلامات الجلسات النشطة للمستخدم';

-- Index للاستعلامات حسب expires_at فقط (لتنظيف الجلسات المنتهية)
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);

COMMENT ON INDEX idx_sessions_expires IS
    'تحسين عملية حذف الجلسات المنتهية';

-- ================================================================
-- 4. Indexes على جدول accounting_transactions
-- ================================================================

-- Index للاستعلامات حسب tenant_id + created_at
CREATE INDEX IF NOT EXISTS idx_accounting_transactions_tenant_created
    ON accounting_transactions(tenant_id, created_at DESC);

COMMENT ON INDEX idx_accounting_transactions_tenant_created IS
    'تحسين استعلامات المعاملات المالية حسب المستأجر والتاريخ';

-- Index للاستعلامات حسب message_id
CREATE INDEX IF NOT EXISTS idx_accounting_transactions_message
    ON accounting_transactions(message_id);

COMMENT ON INDEX idx_accounting_transactions_message IS
    'تحسين البحث عن المعاملات المرتبطة برسالة معينة';

-- Index للاستعلامات حسب status
CREATE INDEX IF NOT EXISTS idx_accounting_transactions_status
    ON accounting_transactions(status, created_at DESC);

COMMENT ON INDEX idx_accounting_transactions_status IS
    'تحسين استعلامات المعاملات حسب الحالة';

-- ================================================================
-- 5. Indexes على جدول balance_history (إن وجد)
-- ================================================================
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_name = 'balance_history'
    ) THEN
        -- Index للاستعلامات حسب tenant_id + created_at
        CREATE INDEX IF NOT EXISTS idx_balance_history_tenant_created
            ON balance_history(tenant_id, created_at DESC);
        
        RAISE NOTICE 'Created index: idx_balance_history_tenant_created';
    END IF;
END $$;

-- ================================================================
-- 6. Indexes على جدول paired_devices
-- ================================================================

-- Index للاستعلامات حسب tenant_id + status
CREATE INDEX IF NOT EXISTS idx_paired_devices_tenant_status
    ON paired_devices(tenant_id, status);

COMMENT ON INDEX idx_paired_devices_tenant_status IS
    'تحسين استعلامات الأجهزة المقترنة حسب المستأجر والحالة';

-- Index للاستعلامات حسب last_seen
CREATE INDEX IF NOT EXISTS idx_paired_devices_last_seen
    ON paired_devices(last_seen DESC)
    WHERE last_seen IS NOT NULL;

COMMENT ON INDEX idx_paired_devices_last_seen IS
    'تحسين استعلامات الأجهزة النشطة مؤخراً';

-- ================================================================
-- 7. Indexes على جدول dispatch_channels
-- ================================================================

-- Index للاستعلامات حسب tenant_id + status + priority
-- (موجود بالفعل كـ idx_channel_routing)
-- نتحقق فقط من وجوده

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE indexname = 'idx_channel_routing'
    ) THEN
        CREATE INDEX idx_channel_routing
            ON dispatch_channels(tenant_id, status, priority DESC);
        
        RAISE NOTICE 'Created index: idx_channel_routing';
    ELSE
        RAISE NOTICE 'Index idx_channel_routing already exists';
    END IF;
END $$;

-- ================================================================
-- 8. عرض ملخص جميع الـ Indexes
-- ================================================================
DO $$
DECLARE
    index_record RECORD;
    total_count INT := 0;
BEGIN
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Indexes Summary';
    RAISE NOTICE '========================================';
    
    FOR index_record IN
        SELECT 
            schemaname,
            tablename,
            indexname,
            pg_size_pretty(pg_relation_size(indexname::regclass)) as size
        FROM pg_indexes
        WHERE schemaname = 'public'
            AND indexname LIKE 'idx_%'
        ORDER BY tablename, indexname
    LOOP
        RAISE NOTICE '% . % (size: %)', 
            index_record.tablename,
            index_record.indexname,
            index_record.size;
        total_count := total_count + 1;
    END LOOP;
    
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Total indexes: %', total_count;
    RAISE NOTICE '========================================';
END $$;

-- ================================================================
-- 9. تحليل الجداول لتحديث إحصائيات الـ query planner
-- ================================================================
ANALYZE messages;
ANALYZE staff_activities;
ANALYZE sessions;
ANALYZE accounting_transactions;
ANALYZE paired_devices;
ANALYZE dispatch_channels;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_name = 'balance_history'
    ) THEN
        EXECUTE 'ANALYZE balance_history';
    END IF;
END $$;



COMMIT;
