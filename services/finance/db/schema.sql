-- db/schema.sql

-- إنشاء الامتداد الخاص بالمعرفات الفريدة
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 1. جدول الفترات المحاسبية
CREATE TABLE IF NOT EXISTS accounting_periods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    period_name VARCHAR(50) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'CLOSED')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 2. دليل الحسابات
CREATE TABLE IF NOT EXISTS accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(20) UNIQUE NOT NULL,
    name VARCHAR(150) NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('ASSET', 'LIABILITY', 'EQUITY', 'REVENUE', 'EXPENSE')),
    normal_balance VARCHAR(10) NOT NULL CHECK (normal_balance IN ('DEBIT', 'CREDIT')),
    parent_id UUID REFERENCES accounts(id),
    reference_type VARCHAR(50), -- 'TENANT', 'RESELLER'
    reference_id UUID,          -- ID from the main database
    is_active BOOLEAN DEFAULT TRUE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX idx_accounts_ref ON accounts(reference_type, reference_id);

-- 3. رؤوس قيود اليومية
CREATE TABLE IF NOT EXISTS journal_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_number VARCHAR(40) UNIQUE NOT NULL,
    entry_date TIMESTAMPTZ NOT NULL,
    description TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'POSTED' CHECK (status IN ('DRAFT', 'POSTED', 'REVERSED')),
    reversal_of UUID REFERENCES journal_entries(id),
    created_by UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- 4. سطور قيود اليومية
CREATE TABLE IF NOT EXISTS journal_entry_lines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    journal_entry_id UUID NOT NULL REFERENCES journal_entries(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES accounts(id),
    debit DECIMAL(19,4) DEFAULT 0 NOT NULL,
    credit DECIMAL(19,4) DEFAULT 0 NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_journal_entry_lines_je_id ON journal_entry_lines(journal_entry_id);
CREATE INDEX idx_journal_entry_lines_acc_id ON journal_entry_lines(account_id);

-- 5. جدول الفواتير
CREATE TABLE IF NOT EXISTS invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_number VARCHAR(50) UNIQUE NOT NULL,
    journal_entry_id UUID NOT NULL REFERENCES journal_entries(id),
    customer_name VARCHAR(150) NOT NULL,
    issue_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sub_total DECIMAL(19,4) NOT NULL,
    vat_amount DECIMAL(19,4) NOT NULL,
    total_amount DECIMAL(19,4) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
