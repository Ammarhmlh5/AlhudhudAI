-- Migration: 027_referrals_table.sql
-- Description: إنشاء جدول referrals لنظام الإحالة
-- Requirements: 33.1, 33.3, 33.8
-- Date: 2026-02-27

-- =====================================================
-- جدول الإحالات (Referrals)
-- =====================================================
-- يسجل العلاقة بين المُحيل (Referrer) والمُحال (Referee)
-- ويتتبع حالة الإحالة والمكافآت الممنوحة

CREATE TABLE IF NOT EXISTS referrals (
    id SERIAL PRIMARY KEY,
    
    -- معلومات الإحالة
    referrer_tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,  -- المُحيل
    referee_tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,   -- المُحال
    referral_code VARCHAR(50) NOT NULL,  -- كود الإحالة المستخدم
    
    -- حالة الإحالة
    status VARCHAR(50) NOT NULL DEFAULT 'pending',  -- pending, completed, rewarded, cancelled
    
    -- معلومات المكافأة
    reward_type VARCHAR(50),  -- balance, messages, discount
    reward_amount DECIMAL(10, 2),  -- المبلغ أو عدد الرسائل أو نسبة الخصم
    reward_granted BOOLEAN DEFAULT false,
    reward_granted_at TIMESTAMP,
    
    -- شروط المكافأة
    reward_condition VARCHAR(100),  -- first_payment, first_subscription, immediate
    condition_met BOOLEAN DEFAULT false,
    condition_met_at TIMESTAMP,
    
    -- معلومات إضافية
    notes TEXT,
    metadata JSONB,  -- بيانات إضافية مرنة
    
    -- التوقيتات
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- القيود
    CONSTRAINT check_referrer_referee_different CHECK (referrer_tenant_id != referee_tenant_id),
    CONSTRAINT check_reward_amount_positive CHECK (reward_amount IS NULL OR reward_amount > 0),
    CONSTRAINT check_status_valid CHECK (status IN ('pending', 'completed', 'rewarded', 'cancelled')),
    CONSTRAINT check_reward_type_valid CHECK (reward_type IS NULL OR reward_type IN ('balance', 'messages', 'discount'))
);

-- =====================================================
-- Indexes للأداء
-- =====================================================

-- Index مركب للمُحيل (للبحث عن إحالات المُحيل)
CREATE INDEX idx_referrals_referrer ON referrals(referrer_tenant_id, created_at DESC);

-- Index مركب للمُحال (للتحقق من عدم استخدام كود سابقاً)
CREATE INDEX idx_referrals_referee ON referrals(referee_tenant_id);

-- Index لكود الإحالة (للبحث السريع)
CREATE INDEX idx_referrals_code ON referrals(referral_code);

-- Index للحالة (لمعالجة الإحالات المعلقة)
CREATE INDEX idx_referrals_status ON referrals(status, created_at DESC);

-- Index للمكافآت غير الممنوحة (لمعالجة المكافآت)
CREATE INDEX idx_referrals_reward_pending ON referrals(reward_granted, condition_met) 
WHERE reward_granted = false AND condition_met = true;

-- =====================================================
-- Row Level Security (RLS)
-- =====================================================

-- تفعيل RLS
ALTER TABLE referrals ENABLE ROW LEVEL SECURITY;

-- Policy للقراءة: يمكن للمُحيل رؤية إحالاته فقط
CREATE POLICY referrals_select_policy ON referrals
    FOR SELECT
    USING (
        referrer_tenant_id = current_setting('app.current_tenant', true)::uuid
        OR referee_tenant_id = current_setting('app.current_tenant', true)::uuid
    );

-- Policy للإدراج: يمكن إنشاء إحالة للمُحال فقط
CREATE POLICY referrals_insert_policy ON referrals
    FOR INSERT
    WITH CHECK (referee_tenant_id = current_setting('app.current_tenant', true)::uuid);

-- Policy للتحديث: يمكن للنظام فقط تحديث الإحالات (عبر service account)
CREATE POLICY referrals_update_policy ON referrals
    FOR UPDATE
    USING (true)  -- سيتم التحكم عبر application logic
    WITH CHECK (true);

-- Policy للحذف: لا يُسمح بالحذف (للحفاظ على سجل التدقيق)
CREATE POLICY referrals_delete_policy ON referrals
    FOR DELETE
    USING (false);

-- =====================================================
-- Trigger لتحديث updated_at تلقائياً
-- =====================================================

CREATE OR REPLACE FUNCTION update_referrals_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_referrals_updated_at
    BEFORE UPDATE ON referrals
    FOR EACH ROW
    EXECUTE FUNCTION update_referrals_updated_at();

-- =====================================================
-- Comments للتوثيق
-- =====================================================

COMMENT ON TABLE referrals IS 'جدول الإحالات - يسجل العلاقة بين المُحيل والمُحال ويتتبع المكافآت';
COMMENT ON COLUMN referrals.referrer_tenant_id IS 'معرف المُحيل (الذي دعا صديقه)';
COMMENT ON COLUMN referrals.referee_tenant_id IS 'معرف المُحال (الذي تم دعوته)';
COMMENT ON COLUMN referrals.referral_code IS 'كود الإحالة المستخدم';
COMMENT ON COLUMN referrals.status IS 'حالة الإحالة: pending (معلقة), completed (مكتملة), rewarded (تم منح المكافأة), cancelled (ملغاة)';
COMMENT ON COLUMN referrals.reward_type IS 'نوع المكافأة: balance (رصيد), messages (رسائل مجانية), discount (خصم)';
COMMENT ON COLUMN referrals.reward_amount IS 'قيمة المكافأة (مبلغ، عدد رسائل، أو نسبة خصم)';
COMMENT ON COLUMN referrals.reward_condition IS 'شرط منح المكافأة: first_payment (أول دفعة), first_subscription (أول اشتراك), immediate (فوري)';
COMMENT ON COLUMN referrals.condition_met IS 'هل تم استيفاء شرط المكافأة؟';
COMMENT ON COLUMN referrals.metadata IS 'بيانات إضافية مرنة بصيغة JSON';

-- =====================================================
-- إحصائيات أولية
-- =====================================================

-- عرض لإحصائيات الإحالات لكل مُحيل
CREATE OR REPLACE VIEW referral_stats AS
SELECT 
    referrer_tenant_id,
    COUNT(*) as total_referrals,
    COUNT(*) FILTER (WHERE status = 'completed') as completed_referrals,
    COUNT(*) FILTER (WHERE status = 'rewarded') as rewarded_referrals,
    COUNT(*) FILTER (WHERE status = 'pending') as pending_referrals,
    SUM(CASE WHEN reward_granted THEN reward_amount ELSE 0 END) as total_rewards_granted,
    MAX(created_at) as last_referral_date
FROM referrals
GROUP BY referrer_tenant_id;

COMMENT ON VIEW referral_stats IS 'إحصائيات الإحالات لكل مُحيل';

-- =====================================================
-- Migration Complete
-- =====================================================

-- تسجيل في جدول الـ migrations (إذا كان موجوداً)
-- INSERT INTO schema_migrations (version, name, applied_at) 
-- VALUES (27, '027_referrals_table', CURRENT_TIMESTAMP);
