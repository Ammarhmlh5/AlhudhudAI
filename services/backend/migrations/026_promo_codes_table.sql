-- Migration: 026_promo_codes_table.sql
-- Description: إنشاء جدول promo_codes لنظام الخصومات والعروض
-- Requirements: 32.1, 32.2
-- Date: 2026-03-02

-- =====================================================
-- جدول promo_codes: أكواد الخصم والعروض الترويجية
-- =====================================================

CREATE TABLE IF NOT EXISTS promo_codes (
    -- المعرفات الأساسية
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- معلومات الكود
    code VARCHAR(50) NOT NULL,                    -- كود الخصم (مثل: SUMMER2026)
    description TEXT,                             -- وصف العرض
    
    -- نوع الخصم
    discount_type VARCHAR(20) NOT NULL CHECK (discount_type IN ('percentage', 'fixed_amount')),
    discount_value DECIMAL(10, 2) NOT NULL CHECK (discount_value > 0),
    
    -- الحد الأقصى للخصم (للنسبة المئوية)
    max_discount_amount DECIMAL(10, 2),          -- الحد الأقصى للخصم بالمبلغ
    
    -- الحد الأدنى للطلب
    min_purchase_amount DECIMAL(10, 2),          -- الحد الأدنى لقيمة الطلب
    
    -- نطاق الصلاحية
    valid_from TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    valid_until TIMESTAMP NOT NULL,
    
    -- حدود الاستخدام
    usage_limit INT,                              -- الحد الأقصى لعدد الاستخدامات (NULL = غير محدود)
    usage_count INT NOT NULL DEFAULT 0,           -- عدد المرات المستخدمة
    per_user_limit INT DEFAULT 1,                 -- الحد الأقصى للاستخدام لكل مستخدم
    
    -- نطاق التطبيق
    applicable_to VARCHAR(20) NOT NULL DEFAULT 'all' CHECK (applicable_to IN ('all', 'packages', 'topup', 'gateway')),
    applicable_package_ids BIGINT[],              -- معرفات الباقات المحددة (NULL = جميع الباقات)
    
    -- الحالة
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    
    -- التوقيتات
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT REFERENCES users(id),
    
    -- القيود
    CONSTRAINT unique_promo_code_per_tenant UNIQUE (tenant_id, code),
    CONSTRAINT valid_date_range CHECK (valid_until > valid_from),
    CONSTRAINT valid_usage_limit CHECK (usage_limit IS NULL OR usage_limit > 0),
    CONSTRAINT valid_per_user_limit CHECK (per_user_limit > 0)
);

-- =====================================================
-- جدول promo_code_usage: سجل استخدام أكواد الخصم
-- =====================================================

CREATE TABLE IF NOT EXISTS promo_code_usage (
    -- المعرفات الأساسية
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    promo_code_id BIGINT NOT NULL REFERENCES promo_codes(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- معلومات الاستخدام
    order_type VARCHAR(20) NOT NULL CHECK (order_type IN ('subscription', 'topup', 'gateway')),
    order_id BIGINT,                              -- معرف الطلب (subscription_id أو transaction_id)
    
    -- قيمة الخصم المطبقة
    original_amount DECIMAL(10, 2) NOT NULL,
    discount_amount DECIMAL(10, 2) NOT NULL,
    final_amount DECIMAL(10, 2) NOT NULL,
    
    -- التوقيت
    used_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- القيود
    CONSTRAINT unique_promo_usage_per_order UNIQUE (tenant_id, promo_code_id, order_type, order_id),
    CONSTRAINT valid_discount_amount CHECK (discount_amount >= 0 AND discount_amount <= original_amount),
    CONSTRAINT valid_final_amount CHECK (final_amount = original_amount - discount_amount)
);

-- =====================================================
-- Indexes لتحسين الأداء
-- =====================================================

-- Indexes لجدول promo_codes
CREATE INDEX idx_promo_codes_tenant ON promo_codes(tenant_id);
CREATE INDEX idx_promo_codes_tenant_code ON promo_codes(tenant_id, code);
CREATE INDEX idx_promo_codes_tenant_active ON promo_codes(tenant_id, is_active) WHERE is_active = TRUE;
CREATE INDEX idx_promo_codes_tenant_valid ON promo_codes(tenant_id, valid_from, valid_until) WHERE is_active = TRUE;
CREATE INDEX idx_promo_codes_applicable_to ON promo_codes(tenant_id, applicable_to) WHERE is_active = TRUE;

-- Indexes لجدول promo_code_usage
CREATE INDEX idx_promo_usage_tenant ON promo_code_usage(tenant_id);
CREATE INDEX idx_promo_usage_promo_code ON promo_code_usage(tenant_id, promo_code_id);
CREATE INDEX idx_promo_usage_user ON promo_code_usage(tenant_id, user_id);
CREATE INDEX idx_promo_usage_order ON promo_code_usage(tenant_id, order_type, order_id);
CREATE INDEX idx_promo_usage_used_at ON promo_code_usage(tenant_id, used_at);

-- =====================================================
-- Row Level Security (RLS) Policies
-- =====================================================

-- تفعيل RLS على جدول promo_codes
ALTER TABLE promo_codes ENABLE ROW LEVEL SECURITY;

-- Policy للقراءة: يمكن للمستخدم قراءة أكواد الخصم الخاصة بشركته فقط
CREATE POLICY promo_codes_select_policy ON promo_codes
    FOR SELECT
    USING (tenant_id = current_setting('app.current_tenant')::BIGINT);

-- Policy للإدراج: يمكن للمستخدم إنشاء أكواد خصم لشركته فقط
CREATE POLICY promo_codes_insert_policy ON promo_codes
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::BIGINT);

-- Policy للتحديث: يمكن للمستخدم تحديث أكواد الخصم الخاصة بشركته فقط
CREATE POLICY promo_codes_update_policy ON promo_codes
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant')::BIGINT)
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::BIGINT);

-- Policy للحذف: يمكن للمستخدم حذف أكواد الخصم الخاصة بشركته فقط
CREATE POLICY promo_codes_delete_policy ON promo_codes
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant')::BIGINT);

-- تفعيل RLS على جدول promo_code_usage
ALTER TABLE promo_code_usage ENABLE ROW LEVEL SECURITY;

-- Policy للقراءة: يمكن للمستخدم قراءة سجل استخدام أكواد الخصم الخاصة بشركته فقط
CREATE POLICY promo_usage_select_policy ON promo_code_usage
    FOR SELECT
    USING (tenant_id = current_setting('app.current_tenant')::BIGINT);

-- Policy للإدراج: يمكن للمستخدم تسجيل استخدام أكواد الخصم لشركته فقط
CREATE POLICY promo_usage_insert_policy ON promo_code_usage
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::BIGINT);

-- Policy للتحديث: لا يُسمح بتحديث سجل الاستخدام (للحفاظ على سلامة البيانات)
-- (لا توجد policy للتحديث)

-- Policy للحذف: لا يُسمح بحذف سجل الاستخدام (للحفاظ على سلامة البيانات)
-- (لا توجد policy للحذف)

-- =====================================================
-- Triggers
-- =====================================================

-- Trigger لتحديث updated_at تلقائياً
CREATE OR REPLACE FUNCTION update_promo_codes_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER promo_codes_updated_at_trigger
    BEFORE UPDATE ON promo_codes
    FOR EACH ROW
    EXECUTE FUNCTION update_promo_codes_updated_at();

-- Trigger لزيادة usage_count عند الاستخدام
CREATE OR REPLACE FUNCTION increment_promo_code_usage()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE promo_codes
    SET usage_count = usage_count + 1
    WHERE id = NEW.promo_code_id AND tenant_id = NEW.tenant_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER increment_promo_usage_trigger
    AFTER INSERT ON promo_code_usage
    FOR EACH ROW
    EXECUTE FUNCTION increment_promo_code_usage();

-- =====================================================
-- Comments للتوثيق
-- =====================================================

COMMENT ON TABLE promo_codes IS 'جدول أكواد الخصم والعروض الترويجية مع عزل كامل بين الشركات';
COMMENT ON COLUMN promo_codes.code IS 'كود الخصم الفريد لكل شركة';
COMMENT ON COLUMN promo_codes.discount_type IS 'نوع الخصم: percentage (نسبة مئوية) أو fixed_amount (مبلغ ثابت)';
COMMENT ON COLUMN promo_codes.discount_value IS 'قيمة الخصم (نسبة مئوية أو مبلغ ثابت)';
COMMENT ON COLUMN promo_codes.max_discount_amount IS 'الحد الأقصى للخصم بالمبلغ (للنسبة المئوية فقط)';
COMMENT ON COLUMN promo_codes.usage_limit IS 'الحد الأقصى لعدد الاستخدامات (NULL = غير محدود)';
COMMENT ON COLUMN promo_codes.per_user_limit IS 'الحد الأقصى للاستخدام لكل مستخدم';
COMMENT ON COLUMN promo_codes.applicable_to IS 'نطاق التطبيق: all, packages, topup, gateway';

COMMENT ON TABLE promo_code_usage IS 'سجل استخدام أكواد الخصم مع عزل كامل بين الشركات';
COMMENT ON COLUMN promo_code_usage.order_type IS 'نوع الطلب: subscription, topup, gateway';
COMMENT ON COLUMN promo_code_usage.discount_amount IS 'قيمة الخصم المطبقة فعلياً';
