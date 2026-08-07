-- ============================================================================
-- Migration: 097 - نظام المحاسبة المزدوجة (Double-Entry Accounting)
-- Description: إنشاء جداول دليل الحسابات والقيود اليومية
-- Date: 2026-08-08
-- ============================================================================

-- Enable UUID extension if not exists
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================================
-- جدول دليل الحسابات (Chart of Accounts)
-- ============================================================================
CREATE TABLE IF NOT EXISTS accounts (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code            VARCHAR(20)  NOT NULL UNIQUE,
    name            VARCHAR(150) NOT NULL,
    name_en         VARCHAR(150),
    type            VARCHAR(20)  NOT NULL 
                     CHECK (type IN ('ASSET','LIABILITY','EQUITY','INCOME','EXPENSE')),
    parent_id       UUID REFERENCES accounts(id) ON DELETE SET NULL,
    is_active       BOOLEAN      NOT NULL DEFAULT TRUE,
    allow_invoice   BOOLEAN      NOT NULL DEFAULT FALSE,  -- للربط بسندات القبض/الصرف
    description     TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- فهارس
CREATE INDEX IF NOT EXISTS idx_accounts_type ON accounts(type);
CREATE INDEX IF NOT EXISTS idx_accounts_parent ON accounts(parent_id);
CREATE INDEX IF NOT EXISTS idx_accounts_active ON accounts(is_active);

-- ============================================================================
-- جدول قيود اليومية (Journal Entries)
-- ============================================================================
CREATE TABLE IF NOT EXISTS journal_entries (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    entry_number    VARCHAR(40)  NOT NULL UNIQUE,
    voucher_type    VARCHAR(20),  -- REVISION, PAYMENT - للربط بسند
    voucher_id      UUID,
    entry_date      DATE        NOT NULL DEFAULT CURRENT_DATE,
    description     TEXT,
    total_debit     DECIMAL(15,4) NOT NULL DEFAULT 0,
    total_credit    DECIMAL(15,4) NOT NULL DEFAULT 0,
    status          VARCHAR(20) NOT NULL DEFAULT 'POSTED'
                     CHECK (status IN ('DRAFT','POSTED','CANCELLED')),
    created_by      UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    posted_at       TIMESTAMPTZ,
    posted_by       UUID
);

-- فهارس
CREATE INDEX IF NOT EXISTS idx_journal_entries_date ON journal_entries(entry_date);
CREATE INDEX IF NOT EXISTS idx_journal_entries_voucher ON journal_entries(voucher_type, voucher_id);
CREATE INDEX IF NOT EXISTS idx_journal_entries_status ON journal_entries(status);

-- ============================================================================
-- جدول بنود قيود اليومية (Journal Entry Lines)
-- ============================================================================
CREATE TABLE IF NOT EXISTS journal_entry_lines (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    journal_entry_id UUID NOT NULL REFERENCES journal_entries(id) ON DELETE CASCADE,
    account_id       UUID NOT NULL REFERENCES accounts(id),
    debit            DECIMAL(15,4) NOT NULL DEFAULT 0,
    credit           DECIMAL(15,4) NOT NULL DEFAULT 0,
    notes            TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- فهارس
CREATE INDEX IF NOT EXISTS idx_journal_entry_lines_entry ON journal_entry_lines(journal_entry_id);
CREATE INDEX IF NOT EXISTS idx_journal_entry_lines_account ON journal_entry_lines(account_id);

-- ============================================================================
-- جدول مولّد أرقام قيود اليومية
-- ============================================================================
CREATE TABLE IF NOT EXISTS journal_entry_counters (
    fiscal_year  INT          NOT NULL PRIMARY KEY,
    last_number  INT          NOT NULL DEFAULT 0
);

-- ============================================================================
-- دفاتر الأستاذ (Ledger Accounts) - للعرض
-- ============================================================================
CREATE TABLE IF NOT EXISTS ledger_entries (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    account_id      UUID NOT NULL REFERENCES accounts(id),
    journal_entry_id UUID REFERENCES journal_entries(id) ON DELETE SET NULL,
    voucher_type    VARCHAR(20),  -- للربط بسند خارجي
    voucher_id      UUID,
    entry_date      DATE        NOT NULL DEFAULT CURRENT_DATE,
    description     TEXT,
    debit           DECIMAL(15,4) NOT NULL DEFAULT 0,
    credit          DECIMAL(15,4) NOT NULL DEFAULT 0,
    balance         DECIMAL(15,4) NOT NULL DEFAULT 0,  -- الرصيد التراكمي
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ledger_entries_account_date ON ledger_entries(account_id, entry_date);
CREATE INDEX IF NOT EXISTS idx_ledger_entries_voucher ON ledger_entries(voucher_type, voucher_id);

-- ============================================================================
-- إعداد دليل حسابات افتراضي
-- ============================================================================
INSERT INTO accounts (code, name, name_en, type, allow_invoice, description) VALUES
-- الأصول (Assets)
('1000', 'النقدية والبنوك', 'Cash and Banks', 'ASSET', TRUE, 'الصناديق والحسابات البنكية'),
('1100', 'الذمم المدينة', 'Accounts Receivable', 'ASSET', TRUE, 'أرصدة العملاء والجهات'),
('1200', 'مInventories', 'المخزون', 'ASSET', FALSE, 'المخزون السلعي'),

-- обязательства (Liabilities)
('2000', 'الذمم الدائنة', 'Accounts Payable', 'LIABILITY', TRUE, 'الالتزامات تجاه الموردين'),
('2100', 'البنوك', 'Bank Overdraft', 'LIABILITY', FALSE, 'السحب على المكشوف'),

-- حقوق الملكية (Equity)
('3000', 'رأس المال', 'Capital', 'EQUITY', FALSE, 'رأس المال المدفوع'),
('3100', 'الأرباح المحتجزة', 'Retained Earnings', 'EQUITY', FALSE, 'الأرباح المرحلة'),

-- الإيرادات (Income)
('4000', 'إيرادات الخدمات', 'Service Revenue', 'INCOME', TRUE, 'إيرادات الرسائل والخدمات'),
('4100', 'إيرادات أخرى', 'Other Income', 'INCOME', FALSE, 'إيرادات متنوعة'),

-- المصروفات (Expense)
('5000', 'تكلفة الخدمات', 'Cost of Services', 'EXPENSE', FALSE, 'تكلفة الرسائل المرسلة'),
('5100', 'مصروفات الرواتب', 'Salaries Expense', 'EXPENSE', FALSE, 'مصروفات الموظفين'),
('5200', 'المصروفات الإدارية', 'Administrative Expenses', 'EXPENSE', FALSE, 'المصروفات التشغيلية'),
('5300', 'مصروفات بنكية', 'Bank Charges', 'EXPENSE', FALSE, 'الرسوم والمصروفات البنكية')
ON CONFLICT (code) DO NOTHING;

-- ============================================================================
-- Function لتحديث الرصيد في دفتر الأستاذ
-- ============================================================================
CREATE OR REPLACE FUNCTION update_ledger_balance()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE ledger_entries 
    SET balance = (
        SELECT COALESCE(SUM(credit) - SUM(debit), 0)
        FROM ledger_entries 
        WHERE account_id = NEW.account_id 
          AND entry_date <= NEW.entry_date
          AND id <= NEW.id
    )
    WHERE id = NEW.id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- View لميزان المراجعة (Trial Balance)
-- ============================================================================
CREATE OR REPLACE VIEW trial_balance_view AS
SELECT 
    a.id,
    a.code,
    a.name,
    a.type,
    COALESCE(SUM(le.credit), 0) - COALESCE(SUM(le.debit), 0) AS balance,
    COALESCE(SUM(le.debit), 0) AS total_debit,
    COALESCE(SUM(le.credit), 0) AS total_credit
FROM accounts a
LEFT JOIN ledger_entries le ON a.id = le.account_id
WHERE a.is_active = TRUE
GROUP BY a.id, a.code, a.name, a.type
HAVING COALESCE(SUM(le.credit), 0) != COALESCE(SUM(le.debit), 0)
   OR a.type IN ('ASSET', 'EXPENSE');