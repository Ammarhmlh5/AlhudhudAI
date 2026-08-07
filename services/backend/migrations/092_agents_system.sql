-- ============================================================================
-- Migration: 092 - نظام الوكلاء والموظفين (Agents & Employees System)
-- Description: إنشاء جداول نظام إدارة الوكلاء والموظفين
-- Date: 2026-08-06
-- ============================================================================

-- Enable UUID extension if not exists
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================================
-- جدول الوكلاء (شركات البرمجة)
-- ============================================================================
CREATE TABLE IF NOT EXISTS agents (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_name        VARCHAR(255) NOT NULL,
    email               VARCHAR(255) UNIQUE NOT NULL,
    password_hash       VARCHAR(255) NOT NULL,
    phone               VARCHAR(50),
    api_key             VARCHAR(255) UNIQUE,
    api_secret          VARCHAR(255),  -- مشفر
    is_active           BOOLEAN DEFAULT true,
    created_at          TIMESTAMPTZ DEFAULT NOW(),
    updated_at          TIMESTAMPTZ DEFAULT NOW()
);

-- فهارس
CREATE INDEX idx_agents_email ON agents(email);
CREATE INDEX idx_agents_api_key ON agents(api_key) WHERE api_key IS NOT NULL;

-- ============================================================================
-- جدول موظفي الوكلاء
-- ============================================================================
CREATE TABLE IF NOT EXISTS agent_employees (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agent_id            UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    name                VARCHAR(255) NOT NULL,
    email               VARCHAR(255) UNIQUE NOT NULL,
    password_hash       VARCHAR(255) NOT NULL,
    role                VARCHAR(20) DEFAULT 'staff',  -- 'admin', 'staff'
    is_active           BOOLEAN DEFAULT true,
    last_login          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ DEFAULT NOW(),
    updated_at          TIMESTAMPTZ DEFAULT NOW()
);

-- فهارس
CREATE INDEX idx_agent_employees_agent_id ON agent_employees(agent_id);
CREATE INDEX idx_agent_employees_email ON agent_employees(email);

-- ============================================================================
-- جدول ربط الوكلاء بمستأجرينهم (العملاء)
-- ============================================================================
CREATE TABLE IF NOT EXISTS agent_tenants (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agent_id            UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    tenant_id           UUID NOT NULL,  -- مرجع للمستأجر (سيتم الربط لاحقاً)
    created_by          UUID,  -- employee_id who created this tenant
    created_at          TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(agent_id, tenant_id)
);

-- فهارس
CREATE INDEX idx_agent_tenants_agent_id ON agent_tenants(agent_id);
CREATE INDEX idx_agent_tenants_tenant_id ON agent_tenants(tenant_id);

-- ============================================================================
-- جدول المدفوعات
-- ============================================================================
CREATE TABLE IF NOT EXISTS agent_payments (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    invoice_id          UUID,  -- مرجع للفاتورة
    employee_id         UUID REFERENCES agent_employees(id) ON DELETE SET NULL,
    agent_id            UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    tenant_id           UUID,  -- المستأجر المرتبط بالفاتورة
    amount              DECIMAL(10,2) NOT NULL,
    currency            VARCHAR(3) DEFAULT 'SAR',
    payment_method      VARCHAR(50),  -- credit_card, bank_transfer, apple_pay
    transaction_id      VARCHAR(255),  -- من بوابة الدفع
    status              VARCHAR(20) DEFAULT 'pending',  -- pending, completed, failed
    notes               TEXT,
    created_at          TIMESTAMPTZ DEFAULT NOW(),
    updated_at          TIMESTAMPTZ DEFAULT NOW()
);

-- فهارس
CREATE INDEX idx_agent_payments_agent_id ON agent_payments(agent_id);
CREATE INDEX idx_agent_payments_employee_id ON agent_payments(employee_id);
CREATE INDEX idx_agent_payments_tenant_id ON agent_payments(tenant_id);
CREATE INDEX idx_agent_payments_invoice_id ON agent_payments(invoice_id) WHERE invoice_id IS NOT NULL;
CREATE INDEX idx_agent_payments_created_at ON agent_payments(created_at);

-- ============================================================================
-- جدول جلسات تسجيل الدخول
-- ============================================================================
CREATE TABLE IF NOT EXISTS agent_sessions (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id             UUID NOT NULL,  -- agent_id or employee_id
    user_type           VARCHAR(20) NOT NULL,  -- 'agent' or 'employee'
    token               VARCHAR(500) NOT NULL,
    refresh_token       VARCHAR(500),
    expires_at          TIMESTAMPTZ NOT NULL,
    ip_address          VARCHAR(45),
    user_agent          TEXT,
    created_at          TIMESTAMPTZ DEFAULT NOW()
);

-- فهارس
CREATE INDEX idx_agent_sessions_token ON agent_sessions(token);
CREATE INDEX idx_agent_sessions_user ON agent_sessions(user_id, user_type);
CREATE INDEX idx_agent_sessions_expires_at ON agent_sessions(expires_at);

-- ============================================================================
-- جدول سجل الأنشطة
-- ============================================================================
CREATE TABLE IF NOT EXISTS agent_activity_log (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id             UUID NOT NULL,
    user_type           VARCHAR(20) NOT NULL,  -- 'agent' or 'employee'
    agent_id            UUID,  -- مرتبط بالوكيل
    action              VARCHAR(100) NOT NULL,
    entity_type         VARCHAR(50),  -- tenant, payment, employee
    entity_id           UUID,
    details             JSONB,
    ip_address          VARCHAR(45),
    created_at          TIMESTAMPTZ DEFAULT NOW()
);

-- فهارس
CREATE INDEX idx_agent_activity_log_user ON agent_activity_log(user_id, user_type);
CREATE INDEX idx_agent_activity_log_agent_id ON agent_activity_log(agent_id);
CREATE INDEX idx_agent_activity_log_created_at ON agent_activity_log(created_at);

-- ============================================================================
-- دالة لتحديث updated_at تلقائياً
-- ============================================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- إضافة triggers للتحديث التلقائي
CREATE TRIGGER update_agents_updated_at 
    BEFORE UPDATE ON agents 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_agent_employees_updated_at 
    BEFORE UPDATE ON agent_employees 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_agent_payments_updated_at 
    BEFORE UPDATE ON agent_payments 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
--注释: إنشاء مستخدم للقراءة فقط (للاستخدام المستقبلي)
-- ============================================================================
-- CREATE ROLE agent_readonly WITH LOGIN PASSWORD 'change_this_password';
-- GRANT USAGE ON SCHEMA public TO agent_readonly;
-- GRANT SELECT ON agents, agent_employees, agent_tenants, agent_payments, agent_sessions, agent_activity_log TO agent_readonly;

-- ============================================================================
--注释: RLS policies للعزل بين الوكلاء (مستقبلاً)
-- ============================================================================
-- ALTER TABLE agents ENABLE ROW LEVEL SECURITY;
-- ALTER TABLE agent_employees ENABLE ROW LEVEL SECURITY;
-- ALTER TABLE agent_tenants ENABLE ROW LEVEL SECURITY;
-- ALTER TABLE agent_payments ENABLE ROW LEVEL SECURITY;

-- ============================================================================
-- ملاحظة: يجب إضافة constraint للربط مع جدول tenants
-- ============================================================================
-- سيتم إضافةforeign key لاحقاً عند التأكد من وجود جدول tenants
-- ALTER TABLE agent_tenants ADD CONSTRAINT fk_agent_tenants_tenant 
--     FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;