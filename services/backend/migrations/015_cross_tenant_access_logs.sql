-- Migration: Cross-Tenant Access Logs Table
-- Description: جدول لتسجيل محاولات الوصول عبر الشركات (Cross-Tenant Access Attempts)
-- Created: 2026-02-27

-- إنشاء جدول cross_tenant_access_logs
CREATE TABLE IF NOT EXISTS cross_tenant_access_logs (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,
    requested_tenant_id UUID NOT NULL,
    user_id UUID,
    user_type VARCHAR(50),
    ip_address VARCHAR(45),
    request_path TEXT,
    request_method VARCHAR(10),
    blocked BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Foreign Keys
    CONSTRAINT fk_cross_tenant_logs_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT fk_cross_tenant_logs_requested_tenant FOREIGN KEY (requested_tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT fk_cross_tenant_logs_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

-- Indexes للأداء
CREATE INDEX idx_cross_tenant_logs_tenant ON cross_tenant_access_logs(tenant_id);
CREATE INDEX idx_cross_tenant_logs_requested_tenant ON cross_tenant_access_logs(requested_tenant_id);
CREATE INDEX idx_cross_tenant_logs_created_at ON cross_tenant_access_logs(created_at DESC);
CREATE INDEX idx_cross_tenant_logs_blocked ON cross_tenant_access_logs(blocked);
CREATE INDEX idx_cross_tenant_logs_user ON cross_tenant_access_logs(user_id);

-- تعليقات على الجدول والأعمدة
COMMENT ON TABLE cross_tenant_access_logs IS 'سجل محاولات الوصول عبر الشركات (Cross-Tenant Access Attempts)';
COMMENT ON COLUMN cross_tenant_access_logs.tenant_id IS 'معرّف الشركة المصادق عليها';
COMMENT ON COLUMN cross_tenant_access_logs.requested_tenant_id IS 'معرّف الشركة المطلوب الوصول إليها';
COMMENT ON COLUMN cross_tenant_access_logs.user_id IS 'معرّف المستخدم الذي حاول الوصول';
COMMENT ON COLUMN cross_tenant_access_logs.user_type IS 'نوع المستخدم (TENANT، STAFF، SUPER_ADMIN)';
COMMENT ON COLUMN cross_tenant_access_logs.ip_address IS 'عنوان IP للطلب';
COMMENT ON COLUMN cross_tenant_access_logs.request_path IS 'مسار الطلب (URL Path)';
COMMENT ON COLUMN cross_tenant_access_logs.request_method IS 'طريقة الطلب (GET، POST، PUT، DELETE)';
COMMENT ON COLUMN cross_tenant_access_logs.blocked IS 'هل تم حظر الطلب؟';
COMMENT ON COLUMN cross_tenant_access_logs.created_at IS 'تاريخ ووقت المحاولة';
