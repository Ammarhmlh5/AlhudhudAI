-- Migration: 022_resource_usage_table.sql
-- Description: إنشاء جدول resource_usage لتسجيل استخدام الموارد للبوابات المخصصة
-- Requirements: 10.1, 10.2, 10.3, 10.4, 10.5, 10.6, 10.7, 10.8, 10.9, 10.10
-- Date: 2024

-- =====================================================
-- جدول الاستخدام الفعلي للموارد (Resource Usage)
-- =====================================================
-- يسجل استخدام الموارد لكل بوابة مخصصة كل دقيقة
-- يستخدم Partitioning حسب الشهر لتحسين الأداء
-- يحتفظ بالبيانات لمدة 90 يوم على الأقل

CREATE TABLE IF NOT EXISTS resource_usage (
    id BIGSERIAL,
    gateway_id INTEGER NOT NULL REFERENCES dedicated_gateways(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- =====================================================
    -- CPU & Memory (Requirements 10.1, 10.2)
    -- =====================================================
    cpu_usage_percent DECIMAL(5, 2), -- نسبة استخدام المعالج (0.00 - 100.00)
    memory_usage_mb INTEGER,          -- استخدام الذاكرة بالميجابايت
    
    -- =====================================================
    -- Network (Requirement 10.3)
    -- =====================================================
    bandwidth_usage_mbps DECIMAL(10, 2), -- استخدام عرض النطاق الترددي بالميجابت/ثانية
    active_connections INTEGER,          -- عدد الاتصالات النشطة
    
    -- =====================================================
    -- Database (Requirements 10.4, 10.5)
    -- =====================================================
    db_connections_active INTEGER,  -- عدد اتصالات قاعدة البيانات النشطة
    queries_per_second INTEGER,     -- عدد الاستعلامات في الثانية
    
    -- =====================================================
    -- Redis/Cache (Requirement 10.6)
    -- =====================================================
    redis_memory_usage_mb INTEGER, -- استخدام ذاكرة Redis بالميجابايت
    redis_keys_count INTEGER,      -- عدد المفاتيح في Redis
    
    -- =====================================================
    -- Message Processing (Requirements 10.7, 10.8)
    -- =====================================================
    messages_processed_last_hour INTEGER, -- عدد الرسائل المعالجة في الساعة الأخيرة
    messages_processed_today INTEGER,     -- عدد الرسائل المعالجة اليوم
    current_queue_depth INTEGER,          -- عمق قائمة الانتظار الحالي
    
    -- =====================================================
    -- API (Requirement 10.9)
    -- =====================================================
    api_requests_last_minute INTEGER, -- عدد طلبات API في الدقيقة الأخيرة
    api_requests_last_hour INTEGER,   -- عدد طلبات API في الساعة الأخيرة
    
    -- =====================================================
    -- Workers (Requirement 10.10)
    -- =====================================================
    active_worker_threads INTEGER, -- عدد Worker Threads النشطة
    
    -- =====================================================
    -- Timestamps
    -- =====================================================
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- =====================================================
    -- Constraints
    -- =====================================================
    CONSTRAINT check_cpu_usage_range CHECK (cpu_usage_percent IS NULL OR (cpu_usage_percent >= 0 AND cpu_usage_percent <= 100)),
    CONSTRAINT check_memory_usage_positive CHECK (memory_usage_mb IS NULL OR memory_usage_mb >= 0),
    CONSTRAINT check_bandwidth_positive CHECK (bandwidth_usage_mbps IS NULL OR bandwidth_usage_mbps >= 0),
    CONSTRAINT check_connections_positive CHECK (active_connections IS NULL OR active_connections >= 0),
    CONSTRAINT check_db_connections_positive CHECK (db_connections_active IS NULL OR db_connections_active >= 0),
    CONSTRAINT check_queries_positive CHECK (queries_per_second IS NULL OR queries_per_second >= 0),
    CONSTRAINT check_redis_memory_positive CHECK (redis_memory_usage_mb IS NULL OR redis_memory_usage_mb >= 0),
    CONSTRAINT check_redis_keys_positive CHECK (redis_keys_count IS NULL OR redis_keys_count >= 0),
    CONSTRAINT check_messages_hour_positive CHECK (messages_processed_last_hour IS NULL OR messages_processed_last_hour >= 0),
    CONSTRAINT check_messages_today_positive CHECK (messages_processed_today IS NULL OR messages_processed_today >= 0),
    CONSTRAINT check_queue_depth_positive CHECK (current_queue_depth IS NULL OR current_queue_depth >= 0),
    CONSTRAINT check_api_minute_positive CHECK (api_requests_last_minute IS NULL OR api_requests_last_minute >= 0),
    CONSTRAINT check_api_hour_positive CHECK (api_requests_last_hour IS NULL OR api_requests_last_hour >= 0),
    CONSTRAINT check_worker_threads_positive CHECK (active_worker_threads IS NULL OR active_worker_threads >= 0),
    
    -- Primary Key يتضمن timestamp للـ Partitioning
    PRIMARY KEY (id, timestamp)
) PARTITION BY RANGE (timestamp);

-- =====================================================
-- Row Level Security (RLS)
-- =====================================================
-- تطبيق العزل التام بين الشركات

ALTER TABLE resource_usage ENABLE ROW LEVEL SECURITY;

-- Policy للقراءة
CREATE POLICY tenant_isolation_policy ON resource_usage
    FOR SELECT
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy للإدراج
CREATE POLICY tenant_isolation_insert_policy ON resource_usage
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy للتحديث (نادر الاستخدام)
CREATE POLICY tenant_isolation_update_policy ON resource_usage
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy للحذف (للصيانة فقط)
CREATE POLICY tenant_isolation_delete_policy ON resource_usage
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- =====================================================
-- Indexes
-- =====================================================
-- Index رئيسي للاستعلامات حسب البوابة والوقت
CREATE INDEX idx_resource_usage_gateway_time ON resource_usage(gateway_id, timestamp DESC);

-- Index للاستعلامات حسب tenant_id
CREATE INDEX idx_resource_usage_tenant ON resource_usage(tenant_id, timestamp DESC);

-- Index للاستعلامات حسب الوقت فقط
CREATE INDEX idx_resource_usage_timestamp ON resource_usage(timestamp DESC);

-- Index مركب لتحليل الأداء
CREATE INDEX idx_resource_usage_tenant_gateway ON resource_usage(tenant_id, gateway_id, timestamp DESC);

-- =====================================================
-- Partitioning حسب الشهر (Requirement 10.11)
-- =====================================================
-- إنشاء Partitions للأشهر الستة القادمة
-- يجب إنشاء Partitions جديدة شهرياً (يمكن أتمتتها)

-- الشهر الحالي
CREATE TABLE resource_usage_2024_12 PARTITION OF resource_usage
    FOR VALUES FROM ('2024-12-01') TO ('2025-01-01');

-- الأشهر القادمة
CREATE TABLE resource_usage_2025_01 PARTITION OF resource_usage
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');

CREATE TABLE resource_usage_2025_02 PARTITION OF resource_usage
    FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');

CREATE TABLE resource_usage_2025_03 PARTITION OF resource_usage
    FOR VALUES FROM ('2025-03-01') TO ('2025-04-01');

CREATE TABLE resource_usage_2025_04 PARTITION OF resource_usage
    FOR VALUES FROM ('2025-04-01') TO ('2025-05-01');

CREATE TABLE resource_usage_2025_05 PARTITION OF resource_usage
    FOR VALUES FROM ('2025-05-01') TO ('2025-06-01');

CREATE TABLE resource_usage_2025_06 PARTITION OF resource_usage
    FOR VALUES FROM ('2025-06-01') TO ('2025-07-01');

-- =====================================================
-- Function لإنشاء Partitions تلقائياً
-- =====================================================
CREATE OR REPLACE FUNCTION create_resource_usage_partition(partition_date DATE)
RETURNS VOID AS $
DECLARE
    partition_name TEXT;
    start_date DATE;
    end_date DATE;
BEGIN
    -- حساب بداية ونهاية الشهر
    start_date := DATE_TRUNC('month', partition_date);
    end_date := start_date + INTERVAL '1 month';
    
    -- اسم الـ Partition
    partition_name := 'resource_usage_' || TO_CHAR(start_date, 'YYYY_MM');
    
    -- التحقق من عدم وجود الـ Partition
    IF NOT EXISTS (
        SELECT 1 FROM pg_class WHERE relname = partition_name
    ) THEN
        -- إنشاء الـ Partition
        EXECUTE format(
            'CREATE TABLE %I PARTITION OF resource_usage FOR VALUES FROM (%L) TO (%L)',
            partition_name,
            start_date,
            end_date
        );
        
        RAISE NOTICE 'Created partition: %', partition_name;
    ELSE
        RAISE NOTICE 'Partition already exists: %', partition_name;
    END IF;
END;
$ LANGUAGE plpgsql;

-- =====================================================
-- Function لحذف Partitions القديمة (أكثر من 90 يوم)
-- =====================================================
CREATE OR REPLACE FUNCTION cleanup_old_resource_usage_partitions()
RETURNS VOID AS $
DECLARE
    partition_record RECORD;
    cutoff_date DATE;
BEGIN
    -- تاريخ القطع (90 يوم مضت)
    cutoff_date := CURRENT_DATE - INTERVAL '90 days';
    
    -- البحث عن Partitions القديمة
    FOR partition_record IN
        SELECT tablename
        FROM pg_tables
        WHERE schemaname = 'public'
        AND tablename LIKE 'resource_usage_%'
        AND tablename ~ '^\d{4}_\d{2}$'
    LOOP
        -- استخراج التاريخ من اسم الـ Partition
        DECLARE
            partition_date DATE;
        BEGIN
            partition_date := TO_DATE(
                SUBSTRING(partition_record.tablename FROM 'resource_usage_(\d{4}_\d{2})'),
                'YYYY_MM'
            );
            
            -- حذف إذا كان أقدم من 90 يوم
            IF partition_date < cutoff_date THEN
                EXECUTE format('DROP TABLE IF EXISTS %I', partition_record.tablename);
                RAISE NOTICE 'Dropped old partition: %', partition_record.tablename;
            END IF;
        EXCEPTION
            WHEN OTHERS THEN
                RAISE WARNING 'Error processing partition %: %', partition_record.tablename, SQLERRM;
        END;
    END LOOP;
END;
$ LANGUAGE plpgsql;

-- =====================================================
-- Comments للتوثيق
-- =====================================================
COMMENT ON TABLE resource_usage IS 'جدول تسجيل استخدام الموارد للبوابات المخصصة - يسجل كل دقيقة';
COMMENT ON COLUMN resource_usage.gateway_id IS 'معرّف البوابة المخصصة';
COMMENT ON COLUMN resource_usage.tenant_id IS 'معرّف الشركة (للعزل)';
COMMENT ON COLUMN resource_usage.timestamp IS 'وقت التسجيل';
COMMENT ON COLUMN resource_usage.cpu_usage_percent IS 'نسبة استخدام المعالج (0-100)';
COMMENT ON COLUMN resource_usage.memory_usage_mb IS 'استخدام الذاكرة بالميجابايت';
COMMENT ON COLUMN resource_usage.bandwidth_usage_mbps IS 'استخدام عرض النطاق الترددي بالميجابت/ثانية';
COMMENT ON COLUMN resource_usage.active_connections IS 'عدد الاتصالات النشطة';
COMMENT ON COLUMN resource_usage.db_connections_active IS 'عدد اتصالات قاعدة البيانات النشطة';
COMMENT ON COLUMN resource_usage.queries_per_second IS 'عدد الاستعلامات في الثانية';
COMMENT ON COLUMN resource_usage.redis_memory_usage_mb IS 'استخدام ذاكرة Redis بالميجابايت';
COMMENT ON COLUMN resource_usage.redis_keys_count IS 'عدد المفاتيح في Redis';
COMMENT ON COLUMN resource_usage.messages_processed_last_hour IS 'عدد الرسائل المعالجة في الساعة الأخيرة';
COMMENT ON COLUMN resource_usage.messages_processed_today IS 'عدد الرسائل المعالجة اليوم';
COMMENT ON COLUMN resource_usage.current_queue_depth IS 'عمق قائمة الانتظار الحالي';
COMMENT ON COLUMN resource_usage.api_requests_last_minute IS 'عدد طلبات API في الدقيقة الأخيرة';
COMMENT ON COLUMN resource_usage.api_requests_last_hour IS 'عدد طلبات API في الساعة الأخيرة';
COMMENT ON COLUMN resource_usage.active_worker_threads IS 'عدد Worker Threads النشطة';

-- =====================================================
-- Rollback Script
-- =====================================================
-- للتراجع عن هذا الـ Migration:
-- DROP FUNCTION IF EXISTS cleanup_old_resource_usage_partitions();
-- DROP FUNCTION IF EXISTS create_resource_usage_partition(DATE);
-- DROP TABLE IF EXISTS resource_usage CASCADE;
