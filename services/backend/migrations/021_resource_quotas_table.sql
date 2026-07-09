-- Migration: 021_resource_quotas_table.sql
-- Description: إنشاء جدول resource_quotas لإدارة الحصص الموردية للبوابات المخصصة
-- Requirements: 9.1, 9.2, 9.3, 9.4, 9.5, 9.6, 9.7, 9.8, 9.9
-- Date: 2024

-- =====================================================
-- جدول الحصص الموردية (Resource Quotas)
-- =====================================================
-- يحدد الحصص الموردية المخصصة لكل بوابة مخصصة
-- يضمن عدم تأثير أي شركة على أداء النظام أو الشركات الأخرى

CREATE TABLE IF NOT EXISTS resource_quotas (
    id SERIAL PRIMARY KEY,
    gateway_id INTEGER NOT NULL REFERENCES dedicated_gateways(id) ON DELETE CASCADE,
    
    -- =====================================================
    -- CPU & Memory (Requirements 9.1, 9.2)
    -- =====================================================
    cpu_cores DECIMAL(4, 2), -- عدد أنوية المعالج (مثال: 2.00 = 2 vCPU)
    memory_mb INTEGER, -- الذاكرة بالميجابايت (مثال: 4096 = 4 GB)
    
    -- =====================================================
    -- Network (Requirement 9.3)
    -- =====================================================
    bandwidth_mbps INTEGER, -- عرض النطاق الترددي بالميجابت/ثانية
    max_concurrent_connections INTEGER, -- الحد الأقصى للاتصالات المتزامنة
    
    -- =====================================================
    -- Storage (Requirement 9.4)
    -- =====================================================
    storage_gb INTEGER, -- مساحة التخزين بالجيجابايت
    
    -- =====================================================
    -- Database (Requirement 9.5)
    -- =====================================================
    max_db_connections INTEGER, -- الحد الأقصى لاتصالات قاعدة البيانات
    max_queries_per_second INTEGER, -- الحد الأقصى للاستعلامات في الثانية
    
    -- =====================================================
    -- Redis/Cache (Requirement 9.6)
    -- =====================================================
    redis_memory_mb INTEGER, -- ذاكرة Redis بالميجابايت
    redis_max_keys INTEGER, -- الحد الأقصى لعدد المفاتيح في Redis
    
    -- =====================================================
    -- Message Processing (Requirement 9.7)
    -- =====================================================
    max_messages_per_hour INTEGER, -- الحد الأقصى للرسائل في الساعة
    max_messages_per_day INTEGER, -- الحد الأقصى للرسائل في اليوم
    max_queue_depth INTEGER, -- الحد الأقصى لعمق قائمة الانتظار
    
    -- =====================================================
    -- API Rate Limiting (Requirement 9.8)
    -- =====================================================
    api_rate_limit_per_minute INTEGER, -- الحد الأقصى لطلبات API في الدقيقة
    api_rate_limit_per_hour INTEGER, -- الحد الأقصى لطلبات API في الساعة
    
    -- =====================================================
    -- Workers (Requirement 9.9)
    -- =====================================================
    max_worker_threads INTEGER, -- الحد الأقصى لعدد Worker Threads
    
    -- =====================================================
    -- الحالة والسياسات
    -- =====================================================
    is_active BOOLEAN DEFAULT true, -- هل الحصص نشطة
    enforcement_mode VARCHAR(20) DEFAULT 'soft', -- 'soft' (تحذير فقط) أو 'hard' (رفض الطلبات)
    
    -- =====================================================
    -- Timestamps
    -- =====================================================
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- =====================================================
    -- Constraints
    -- =====================================================
    CONSTRAINT uq_resource_quotas_gateway UNIQUE(gateway_id),
    CONSTRAINT check_cpu_cores_positive CHECK (cpu_cores IS NULL OR cpu_cores > 0),
    CONSTRAINT check_memory_positive CHECK (memory_mb IS NULL OR memory_mb > 0),
    CONSTRAINT check_bandwidth_positive CHECK (bandwidth_mbps IS NULL OR bandwidth_mbps > 0),
    CONSTRAINT check_storage_positive CHECK (storage_gb IS NULL OR storage_gb > 0),
    CONSTRAINT check_db_connections_positive CHECK (max_db_connections IS NULL OR max_db_connections > 0),
    CONSTRAINT check_queries_positive CHECK (max_queries_per_second IS NULL OR max_queries_per_second > 0),
    CONSTRAINT check_redis_memory_positive CHECK (redis_memory_mb IS NULL OR redis_memory_mb > 0),
    CONSTRAINT check_redis_keys_positive CHECK (redis_max_keys IS NULL OR redis_max_keys > 0),
    CONSTRAINT check_messages_hour_positive CHECK (max_messages_per_hour IS NULL OR max_messages_per_hour > 0),
    CONSTRAINT check_messages_day_positive CHECK (max_messages_per_day IS NULL OR max_messages_per_day > 0),
    CONSTRAINT check_queue_depth_positive CHECK (max_queue_depth IS NULL OR max_queue_depth > 0),
    CONSTRAINT check_api_rate_minute_positive CHECK (api_rate_limit_per_minute IS NULL OR api_rate_limit_per_minute > 0),
    CONSTRAINT check_api_rate_hour_positive CHECK (api_rate_limit_per_hour IS NULL OR api_rate_limit_per_hour > 0),
    CONSTRAINT check_worker_threads_positive CHECK (max_worker_threads IS NULL OR max_worker_threads > 0),
    CONSTRAINT check_enforcement_mode CHECK (enforcement_mode IN ('soft', 'hard'))
);

-- =====================================================
-- Indexes
-- =====================================================
CREATE INDEX idx_resource_quotas_gateway ON resource_quotas(gateway_id);
CREATE INDEX idx_resource_quotas_active ON resource_quotas(is_active);
CREATE INDEX idx_resource_quotas_enforcement ON resource_quotas(enforcement_mode);

-- =====================================================
-- Trigger لتحديث updated_at تلقائياً
-- =====================================================
CREATE OR REPLACE FUNCTION update_resource_quotas_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_resource_quotas_updated_at
    BEFORE UPDATE ON resource_quotas
    FOR EACH ROW
    EXECUTE FUNCTION update_resource_quotas_updated_at();

-- =====================================================
-- Comments للتوثيق
-- =====================================================
COMMENT ON TABLE resource_quotas IS 'جدول الحصص الموردية للبوابات المخصصة - يحدد الحدود القصوى للموارد لكل بوابة';
COMMENT ON COLUMN resource_quotas.gateway_id IS 'معرّف البوابة المخصصة';
COMMENT ON COLUMN resource_quotas.cpu_cores IS 'عدد أنوية المعالج المخصصة (vCPU)';
COMMENT ON COLUMN resource_quotas.memory_mb IS 'الذاكرة المخصصة بالميجابايت';
COMMENT ON COLUMN resource_quotas.bandwidth_mbps IS 'عرض النطاق الترددي بالميجابت/ثانية';
COMMENT ON COLUMN resource_quotas.storage_gb IS 'مساحة التخزين المخصصة بالجيجابايت';
COMMENT ON COLUMN resource_quotas.max_db_connections IS 'الحد الأقصى لاتصالات قاعدة البيانات';
COMMENT ON COLUMN resource_quotas.redis_memory_mb IS 'ذاكرة Redis المخصصة بالميجابايت';
COMMENT ON COLUMN resource_quotas.max_messages_per_hour IS 'الحد الأقصى للرسائل في الساعة';
COMMENT ON COLUMN resource_quotas.max_messages_per_day IS 'الحد الأقصى للرسائل في اليوم';
COMMENT ON COLUMN resource_quotas.api_rate_limit_per_minute IS 'الحد الأقصى لطلبات API في الدقيقة';
COMMENT ON COLUMN resource_quotas.max_worker_threads IS 'الحد الأقصى لعدد Worker Threads';
COMMENT ON COLUMN resource_quotas.enforcement_mode IS 'سياسة التطبيق: soft (تحذير) أو hard (رفض)';

-- =====================================================
-- البيانات الافتراضية (Default Quotas)
-- =====================================================
-- حصص افتراضية للبوابات الجديدة

-- ملاحظة: سيتم إدراج الحصص الافتراضية عند إنشاء بوابة جديدة
-- هذه مجرد أمثلة للحصص الافتراضية:

-- Small Gateway (صغيرة)
-- cpu_cores: 2.00
-- memory_mb: 4096 (4 GB)
-- bandwidth_mbps: 100
-- storage_gb: 50
-- max_db_connections: 20
-- max_queries_per_second: 100
-- redis_memory_mb: 512
-- redis_max_keys: 100000
-- max_messages_per_hour: 10000
-- max_messages_per_day: 100000
-- max_queue_depth: 1000
-- api_rate_limit_per_minute: 100
-- api_rate_limit_per_hour: 5000
-- max_worker_threads: 10

-- Medium Gateway (متوسطة)
-- cpu_cores: 4.00
-- memory_mb: 8192 (8 GB)
-- bandwidth_mbps: 200
-- storage_gb: 100
-- max_db_connections: 50
-- max_queries_per_second: 200
-- redis_memory_mb: 1024
-- redis_max_keys: 500000
-- max_messages_per_hour: 50000
-- max_messages_per_day: 500000
-- max_queue_depth: 5000
-- api_rate_limit_per_minute: 300
-- api_rate_limit_per_hour: 15000
-- max_worker_threads: 20

-- Large Gateway (كبيرة)
-- cpu_cores: 8.00
-- memory_mb: 16384 (16 GB)
-- bandwidth_mbps: 500
-- storage_gb: 200
-- max_db_connections: 100
-- max_queries_per_second: 500
-- redis_memory_mb: 2048
-- redis_max_keys: 1000000
-- max_messages_per_hour: 200000
-- max_messages_per_day: 2000000
-- max_queue_depth: 10000
-- api_rate_limit_per_minute: 1000
-- api_rate_limit_per_hour: 50000
-- max_worker_threads: 50

-- =====================================================
-- Rollback Script
-- =====================================================
-- للتراجع عن هذا الـ Migration:
-- DROP TRIGGER IF EXISTS trigger_update_resource_quotas_updated_at ON resource_quotas;
-- DROP FUNCTION IF EXISTS update_resource_quotas_updated_at();
-- DROP TABLE IF EXISTS resource_quotas CASCADE;
