-- Migration: 025_gateway_statistics_table.sql
-- Description: إنشاء جدول gateway_statistics لتسجيل الإحصائيات اليومية للبوابات المخصصة
-- Requirements: 13.1, 13.2, 13.3, 13.4, 13.5, 13.6, 13.7
-- Date: 2024

-- =====================================================
-- جدول إحصائيات البوابات المخصصة
-- =====================================================

CREATE TABLE IF NOT EXISTS gateway_statistics (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    gateway_id INTEGER NOT NULL REFERENCES dedicated_gateways(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    
    -- إحصائيات الرسائل
    messages_sent INTEGER DEFAULT 0,
    messages_delivered INTEGER DEFAULT 0,
    messages_failed INTEGER DEFAULT 0,
    messages_pending INTEGER DEFAULT 0,
    
    -- إحصائيات الأداء
    avg_response_time_ms INTEGER,
    max_response_time_ms INTEGER,
    min_response_time_ms INTEGER,
    uptime_percentage DECIMAL(5, 2),
    downtime_minutes INTEGER DEFAULT 0,
    
    -- إحصائيات مالية
    total_cost DECIMAL(10, 2) DEFAULT 0.00,
    total_revenue DECIMAL(10, 2) DEFAULT 0.00,
    profit_margin DECIMAL(10, 2) DEFAULT 0.00,
    
    -- التواريخ
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- قيود
    CONSTRAINT fk_gateway_statistics_tenant_gateway FOREIGN KEY (tenant_id, gateway_id) 
        REFERENCES dedicated_gateways(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT uq_gateway_statistics_gateway_date UNIQUE(gateway_id, date),
    CONSTRAINT check_messages_non_negative CHECK (
        messages_sent >= 0 AND 
        messages_delivered >= 0 AND 
        messages_failed >= 0 AND 
        messages_pending >= 0
    ),
    CONSTRAINT check_response_times_positive CHECK (
        (avg_response_time_ms IS NULL OR avg_response_time_ms >= 0) AND
        (max_response_time_ms IS NULL OR max_response_time_ms >= 0) AND
        (min_response_time_ms IS NULL OR min_response_time_ms >= 0)
    ),
    CONSTRAINT check_uptime_percentage CHECK (
        uptime_percentage IS NULL OR (uptime_percentage >= 0 AND uptime_percentage <= 100)
    ),
    CONSTRAINT check_downtime_non_negative CHECK (downtime_minutes >= 0),
    CONSTRAINT check_messages_consistency CHECK (
        messages_sent >= (messages_delivered + messages_failed)
    )
);

-- =====================================================
-- Indexes للأداء
-- =====================================================

-- Index مركب يبدأ بـ tenant_id للعزل
CREATE INDEX idx_gateway_statistics_tenant ON gateway_statistics(tenant_id);
CREATE INDEX idx_gateway_statistics_tenant_gateway ON gateway_statistics(tenant_id, gateway_id);
CREATE INDEX idx_gateway_statistics_tenant_date ON gateway_statistics(tenant_id, date DESC);
CREATE INDEX idx_gateway_statistics_tenant_gateway_date ON gateway_statistics(tenant_id, gateway_id, date DESC);

-- Indexes إضافية للبحث السريع
CREATE INDEX idx_gateway_statistics_gateway_date ON gateway_statistics(gateway_id, date DESC);
CREATE INDEX idx_gateway_statistics_date ON gateway_statistics(date DESC);
CREATE INDEX idx_gateway_statistics_gateway ON gateway_statistics(gateway_id);

-- =====================================================
-- Row Level Security (RLS)
-- =====================================================

-- تفعيل RLS
ALTER TABLE gateway_statistics ENABLE ROW LEVEL SECURITY;

-- Policy للقراءة
CREATE POLICY tenant_isolation_policy ON gateway_statistics
    FOR SELECT
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy للإدراج
CREATE POLICY tenant_isolation_insert_policy ON gateway_statistics
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy للتحديث
CREATE POLICY tenant_isolation_update_policy ON gateway_statistics
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy للحذف
CREATE POLICY tenant_isolation_delete_policy ON gateway_statistics
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- =====================================================
-- Trigger لتحديث updated_at تلقائياً
-- =====================================================

CREATE OR REPLACE FUNCTION update_gateway_statistics_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_gateway_statistics_updated_at
    BEFORE UPDATE ON gateway_statistics
    FOR EACH ROW
    EXECUTE FUNCTION update_gateway_statistics_updated_at();

-- =====================================================
-- Trigger للتحقق من tenant_id consistency
-- =====================================================

CREATE OR REPLACE FUNCTION check_gateway_statistics_tenant_consistency()
RETURNS TRIGGER AS $$
DECLARE
    gateway_tenant_id UUID;
BEGIN
    -- التحقق من أن gateway_id ينتمي لنفس tenant_id
    SELECT tenant_id INTO gateway_tenant_id
    FROM dedicated_gateways
    WHERE id = NEW.gateway_id;
    
    IF gateway_tenant_id IS NULL THEN
        RAISE EXCEPTION 'Gateway % does not exist', NEW.gateway_id;
    END IF;
    
    IF gateway_tenant_id != NEW.tenant_id THEN
        RAISE EXCEPTION 'Gateway % does not belong to tenant %', NEW.gateway_id, NEW.tenant_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_check_gateway_statistics_tenant_consistency
    BEFORE INSERT OR UPDATE ON gateway_statistics
    FOR EACH ROW
    EXECUTE FUNCTION check_gateway_statistics_tenant_consistency();

-- =====================================================
-- Trigger لحساب profit_margin تلقائياً
-- =====================================================

CREATE OR REPLACE FUNCTION calculate_gateway_statistics_profit_margin()
RETURNS TRIGGER AS $$
BEGIN
    -- حساب هامش الربح: (الإيرادات - التكلفة)
    IF NEW.total_revenue IS NOT NULL AND NEW.total_cost IS NOT NULL THEN
        NEW.profit_margin = NEW.total_revenue - NEW.total_cost;
    ELSE
        NEW.profit_margin = 0.00;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_calculate_gateway_statistics_profit_margin
    BEFORE INSERT OR UPDATE ON gateway_statistics
    FOR EACH ROW
    EXECUTE FUNCTION calculate_gateway_statistics_profit_margin();

-- =====================================================
-- Trigger لحساب معدل النجاح تلقائياً
-- =====================================================

CREATE OR REPLACE FUNCTION validate_gateway_statistics_consistency()
RETURNS TRIGGER AS $$
DECLARE
    success_rate DECIMAL(5, 2);
BEGIN
    -- التحقق من أن messages_sent >= messages_delivered + messages_failed
    IF NEW.messages_sent < (NEW.messages_delivered + NEW.messages_failed) THEN
        RAISE EXCEPTION 'Invalid message counts: sent=%, delivered=%, failed=%', 
            NEW.messages_sent, NEW.messages_delivered, NEW.messages_failed;
    END IF;
    
    -- حساب معدل النجاح
    IF NEW.messages_sent > 0 THEN
        success_rate = (NEW.messages_delivered::DECIMAL / NEW.messages_sent::DECIMAL) * 100;
        
        -- تحذير إذا كان معدل النجاح منخفض جداً
        IF success_rate < 50 THEN
            RAISE WARNING 'Low success rate for gateway % on %: %.2f%%', 
                NEW.gateway_id, NEW.date, success_rate;
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_validate_gateway_statistics_consistency
    BEFORE INSERT OR UPDATE ON gateway_statistics
    FOR EACH ROW
    EXECUTE FUNCTION validate_gateway_statistics_consistency();

-- =====================================================
-- Function لحساب الإحصائيات اليومية
-- =====================================================

CREATE OR REPLACE FUNCTION aggregate_daily_gateway_statistics(
    p_gateway_id INTEGER,
    p_date DATE
) RETURNS VOID AS $$
DECLARE
    v_tenant_id UUID;
    v_messages_sent INTEGER;
    v_messages_delivered INTEGER;
    v_messages_failed INTEGER;
    v_messages_pending INTEGER;
    v_avg_response_time INTEGER;
    v_max_response_time INTEGER;
    v_min_response_time INTEGER;
    v_uptime_percentage DECIMAL(5, 2);
    v_downtime_minutes INTEGER;
    v_total_cost DECIMAL(10, 2);
    v_total_revenue DECIMAL(10, 2);
BEGIN
    -- الحصول على tenant_id
    SELECT tenant_id INTO v_tenant_id
    FROM dedicated_gateways
    WHERE id = p_gateway_id;
    
    IF v_tenant_id IS NULL THEN
        RAISE EXCEPTION 'Gateway % does not exist', p_gateway_id;
    END IF;
    
    -- حساب إحصائيات الرسائل
    SELECT 
        COUNT(*) FILTER (WHERE status IN ('sent', 'delivered', 'failed', 'pending')),
        COUNT(*) FILTER (WHERE status = 'delivered'),
        COUNT(*) FILTER (WHERE status = 'failed'),
        COUNT(*) FILTER (WHERE status = 'pending')
    INTO 
        v_messages_sent,
        v_messages_delivered,
        v_messages_failed,
        v_messages_pending
    FROM messages
    WHERE gateway_id = p_gateway_id
      AND DATE(created_at) = p_date;
    
    -- حساب إحصائيات الأداء (من جدول message_logs أو performance_metrics)
    SELECT 
        AVG(response_time_ms)::INTEGER,
        MAX(response_time_ms)::INTEGER,
        MIN(response_time_ms)::INTEGER
    INTO 
        v_avg_response_time,
        v_max_response_time,
        v_min_response_time
    FROM message_logs
    WHERE gateway_id = p_gateway_id
      AND DATE(created_at) = p_date
      AND response_time_ms IS NOT NULL;
    
    -- حساب وقت التشغيل (من جدول gateway_health_checks)
    SELECT 
        (COUNT(*) FILTER (WHERE is_healthy = true)::DECIMAL / COUNT(*)::DECIMAL * 100)::DECIMAL(5,2),
        SUM(CASE WHEN is_healthy = false THEN check_interval_minutes ELSE 0 END)::INTEGER
    INTO 
        v_uptime_percentage,
        v_downtime_minutes
    FROM gateway_health_checks
    WHERE gateway_id = p_gateway_id
      AND DATE(checked_at) = p_date;
    
    -- حساب الإحصائيات المالية
    SELECT 
        COALESCE(SUM(amount) FILTER (WHERE transaction_type = 'cost'), 0),
        COALESCE(SUM(amount) FILTER (WHERE transaction_type = 'revenue'), 0)
    INTO 
        v_total_cost,
        v_total_revenue
    FROM accounting_transactions
    WHERE gateway_id = p_gateway_id
      AND DATE(created_at) = p_date;
    
    -- إدراج أو تحديث الإحصائيات
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
        v_tenant_id,
        p_gateway_id,
        p_date,
        COALESCE(v_messages_sent, 0),
        COALESCE(v_messages_delivered, 0),
        COALESCE(v_messages_failed, 0),
        COALESCE(v_messages_pending, 0),
        v_avg_response_time,
        v_max_response_time,
        v_min_response_time,
        COALESCE(v_uptime_percentage, 100.00),
        COALESCE(v_downtime_minutes, 0),
        COALESCE(v_total_cost, 0.00),
        COALESCE(v_total_revenue, 0.00)
    )
    ON CONFLICT (gateway_id, date)
    DO UPDATE SET
        messages_sent = EXCLUDED.messages_sent,
        messages_delivered = EXCLUDED.messages_delivered,
        messages_failed = EXCLUDED.messages_failed,
        messages_pending = EXCLUDED.messages_pending,
        avg_response_time_ms = EXCLUDED.avg_response_time_ms,
        max_response_time_ms = EXCLUDED.max_response_time_ms,
        min_response_time_ms = EXCLUDED.min_response_time_ms,
        uptime_percentage = EXCLUDED.uptime_percentage,
        downtime_minutes = EXCLUDED.downtime_minutes,
        total_cost = EXCLUDED.total_cost,
        total_revenue = EXCLUDED.total_revenue,
        updated_at = CURRENT_TIMESTAMP;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- Function لحساب معدل النجاح
-- =====================================================

CREATE OR REPLACE FUNCTION get_gateway_success_rate(
    p_gateway_id INTEGER,
    p_date DATE
) RETURNS DECIMAL(5, 2) AS $$
DECLARE
    v_success_rate DECIMAL(5, 2);
BEGIN
    SELECT 
        CASE 
            WHEN messages_sent > 0 THEN 
                (messages_delivered::DECIMAL / messages_sent::DECIMAL * 100)::DECIMAL(5,2)
            ELSE 
                0.00
        END
    INTO v_success_rate
    FROM gateway_statistics
    WHERE gateway_id = p_gateway_id
      AND date = p_date;
    
    RETURN COALESCE(v_success_rate, 0.00);
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- Function للحصول على إحصائيات فترة زمنية
-- =====================================================

CREATE OR REPLACE FUNCTION get_gateway_statistics_range(
    p_gateway_id INTEGER,
    p_start_date DATE,
    p_end_date DATE
) RETURNS TABLE (
    total_messages_sent BIGINT,
    total_messages_delivered BIGINT,
    total_messages_failed BIGINT,
    avg_success_rate DECIMAL(5, 2),
    avg_response_time INTEGER,
    avg_uptime_percentage DECIMAL(5, 2),
    total_cost DECIMAL(10, 2),
    total_revenue DECIMAL(10, 2),
    total_profit DECIMAL(10, 2)
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        SUM(gs.messages_sent)::BIGINT,
        SUM(gs.messages_delivered)::BIGINT,
        SUM(gs.messages_failed)::BIGINT,
        CASE 
            WHEN SUM(gs.messages_sent) > 0 THEN
                (SUM(gs.messages_delivered)::DECIMAL / SUM(gs.messages_sent)::DECIMAL * 100)::DECIMAL(5,2)
            ELSE
                0.00
        END,
        AVG(gs.avg_response_time_ms)::INTEGER,
        AVG(gs.uptime_percentage)::DECIMAL(5,2),
        SUM(gs.total_cost)::DECIMAL(10,2),
        SUM(gs.total_revenue)::DECIMAL(10,2),
        SUM(gs.profit_margin)::DECIMAL(10,2)
    FROM gateway_statistics gs
    WHERE gs.gateway_id = p_gateway_id
      AND gs.date BETWEEN p_start_date AND p_end_date;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- Function لتنظيف الإحصائيات القديمة
-- =====================================================

CREATE OR REPLACE FUNCTION cleanup_old_gateway_statistics(
    p_retention_days INTEGER DEFAULT 365
) RETURNS INTEGER AS $$
DECLARE
    v_deleted_count INTEGER;
BEGIN
    -- حذف الإحصائيات الأقدم من فترة الاحتفاظ
    DELETE FROM gateway_statistics
    WHERE date < CURRENT_DATE - p_retention_days;
    
    GET DIAGNOSTICS v_deleted_count = ROW_COUNT;
    
    RETURN v_deleted_count;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- Comments للتوثيق
-- =====================================================

COMMENT ON TABLE gateway_statistics IS 'جدول إحصائيات البوابات المخصصة - يسجل الإحصائيات اليومية لكل بوابة';
COMMENT ON COLUMN gateway_statistics.id IS 'المعرف الفريد للإحصائية';
COMMENT ON COLUMN gateway_statistics.tenant_id IS 'معرف الشركة (للعزل بين الشركات)';
COMMENT ON COLUMN gateway_statistics.gateway_id IS 'معرف البوابة المخصصة';
COMMENT ON COLUMN gateway_statistics.date IS 'تاريخ الإحصائية';
COMMENT ON COLUMN gateway_statistics.messages_sent IS 'عدد الرسائل المرسلة';
COMMENT ON COLUMN gateway_statistics.messages_delivered IS 'عدد الرسائل المسلّمة';
COMMENT ON COLUMN gateway_statistics.messages_failed IS 'عدد الرسائل الفاشلة';
COMMENT ON COLUMN gateway_statistics.messages_pending IS 'عدد الرسائل المعلقة';
COMMENT ON COLUMN gateway_statistics.avg_response_time_ms IS 'متوسط وقت الاستجابة بالميلي ثانية';
COMMENT ON COLUMN gateway_statistics.max_response_time_ms IS 'أقصى وقت استجابة بالميلي ثانية';
COMMENT ON COLUMN gateway_statistics.min_response_time_ms IS 'أدنى وقت استجابة بالميلي ثانية';
COMMENT ON COLUMN gateway_statistics.uptime_percentage IS 'نسبة وقت التشغيل (%)';
COMMENT ON COLUMN gateway_statistics.downtime_minutes IS 'وقت التوقف بالدقائق';
COMMENT ON COLUMN gateway_statistics.total_cost IS 'التكلفة الإجمالية';
COMMENT ON COLUMN gateway_statistics.total_revenue IS 'الإيرادات الإجمالية';
COMMENT ON COLUMN gateway_statistics.profit_margin IS 'هامش الربح (الإيرادات - التكلفة)';
COMMENT ON COLUMN gateway_statistics.created_at IS 'تاريخ إنشاء السجل';
COMMENT ON COLUMN gateway_statistics.updated_at IS 'تاريخ آخر تحديث';

COMMENT ON FUNCTION aggregate_daily_gateway_statistics(INTEGER, DATE) IS 'حساب وتجميع الإحصائيات اليومية لبوابة محددة';
COMMENT ON FUNCTION get_gateway_success_rate(INTEGER, DATE) IS 'حساب معدل النجاح لبوابة في تاريخ محدد';
COMMENT ON FUNCTION get_gateway_statistics_range(INTEGER, DATE, DATE) IS 'الحصول على إحصائيات مجمعة لفترة زمنية';
COMMENT ON FUNCTION cleanup_old_gateway_statistics(INTEGER) IS 'تنظيف الإحصائيات القديمة (افتراضياً: أقدم من سنة)';

-- =====================================================
-- Rollback Script
-- =====================================================

-- لإلغاء هذا الـ Migration، قم بتشغيل:
-- DROP FUNCTION IF EXISTS cleanup_old_gateway_statistics(INTEGER);
-- DROP FUNCTION IF EXISTS get_gateway_statistics_range(INTEGER, DATE, DATE);
-- DROP FUNCTION IF EXISTS get_gateway_success_rate(INTEGER, DATE);
-- DROP FUNCTION IF EXISTS aggregate_daily_gateway_statistics(INTEGER, DATE);
-- DROP TRIGGER IF EXISTS trigger_validate_gateway_statistics_consistency ON gateway_statistics;
-- DROP TRIGGER IF EXISTS trigger_calculate_gateway_statistics_profit_margin ON gateway_statistics;
-- DROP TRIGGER IF EXISTS trigger_check_gateway_statistics_tenant_consistency ON gateway_statistics;
-- DROP TRIGGER IF EXISTS trigger_update_gateway_statistics_updated_at ON gateway_statistics;
-- DROP FUNCTION IF EXISTS validate_gateway_statistics_consistency();
-- DROP FUNCTION IF EXISTS calculate_gateway_statistics_profit_margin();
-- DROP FUNCTION IF EXISTS check_gateway_statistics_tenant_consistency();
-- DROP FUNCTION IF EXISTS update_gateway_statistics_updated_at();
-- DROP TABLE IF EXISTS gateway_statistics CASCADE;
