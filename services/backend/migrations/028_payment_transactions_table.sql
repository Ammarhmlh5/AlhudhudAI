-- Migration: 028_payment_transactions_table.sql
-- Description: إنشاء جدول payment_transactions لتسجيل جميع عمليات الدفع
-- Requirements: 39.8, 39.9

-- إنشاء جدول payment_transactions
CREATE TABLE IF NOT EXISTS payment_transactions (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    
    -- معلومات المعاملة
    transaction_id VARCHAR(255) UNIQUE NOT NULL, -- معرف فريد للمعاملة
    external_transaction_id VARCHAR(255), -- معرف المعاملة من بوابة الدفع
    
    -- معلومات الدفع
    payment_gateway VARCHAR(50) NOT NULL, -- 'stripe', 'paypal', 'local'
    payment_method VARCHAR(50), -- 'card', 'bank_transfer', 'wallet'
    amount DECIMAL(10, 2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    
    -- الحالة
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- 'pending', 'processing', 'completed', 'failed', 'refunded'
    
    -- معلومات إضافية
    description TEXT,
    metadata JSONB, -- بيانات إضافية من بوابة الدفع
    
    -- معلومات الاسترجاع
    refund_amount DECIMAL(10, 2),
    refund_reason TEXT,
    refunded_at TIMESTAMP,
    
    -- معلومات الخطأ
    error_code VARCHAR(100),
    error_message TEXT,
    
    -- التوقيتات
    initiated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- القيود
    CONSTRAINT check_amount_positive CHECK (amount > 0),
    CONSTRAINT check_refund_amount CHECK (refund_amount IS NULL OR refund_amount <= amount),
    CONSTRAINT check_status_valid CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'refunded'))
);

-- إنشاء Indexes مركبة تبدأ بـ tenant_id
CREATE INDEX idx_payment_transactions_tenant ON payment_transactions(tenant_id);
CREATE INDEX idx_payment_transactions_tenant_status ON payment_transactions(tenant_id, status);
CREATE INDEX idx_payment_transactions_tenant_created ON payment_transactions(tenant_id, created_at DESC);
CREATE INDEX idx_payment_transactions_tenant_gateway ON payment_transactions(tenant_id, payment_gateway);
CREATE INDEX idx_payment_transactions_external_id ON payment_transactions(external_transaction_id);
CREATE INDEX idx_payment_transactions_transaction_id ON payment_transactions(transaction_id);

-- تفعيل Row Level Security
ALTER TABLE payment_transactions ENABLE ROW LEVEL SECURITY;

-- Policy للقراءة
CREATE POLICY tenant_isolation_policy ON payment_transactions
    FOR SELECT
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy للإدراج
CREATE POLICY tenant_isolation_insert_policy ON payment_transactions
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy للتحديث
CREATE POLICY tenant_isolation_update_policy ON payment_transactions
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy للحذف (محظور - للحفاظ على سجل المعاملات)
CREATE POLICY tenant_isolation_delete_policy ON payment_transactions
    FOR DELETE
    USING (false); -- لا يسمح بالحذف أبداً

-- إنشاء Trigger لتحديث updated_at
CREATE OR REPLACE FUNCTION update_payment_transactions_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER payment_transactions_updated_at
    BEFORE UPDATE ON payment_transactions
    FOR EACH ROW
    EXECUTE FUNCTION update_payment_transactions_updated_at();

-- إضافة تعليقات
COMMENT ON TABLE payment_transactions IS 'جدول لتسجيل جميع عمليات الدفع من بوابات الدفع المختلفة';
COMMENT ON COLUMN payment_transactions.tenant_id IS 'معرف الشركة (للعزل)';
COMMENT ON COLUMN payment_transactions.transaction_id IS 'معرف فريد للمعاملة في النظام';
COMMENT ON COLUMN payment_transactions.external_transaction_id IS 'معرف المعاملة من بوابة الدفع الخارجية';
COMMENT ON COLUMN payment_transactions.payment_gateway IS 'بوابة الدفع المستخدمة (stripe, paypal, local)';
COMMENT ON COLUMN payment_transactions.status IS 'حالة المعاملة (pending, processing, completed, failed, refunded)';
COMMENT ON COLUMN payment_transactions.metadata IS 'بيانات إضافية من بوابة الدفع بصيغة JSON';
