-- ============================================================================
-- Migration: 093 - نظام الحسابات: الصناديق ومولّد أرقام السندات
-- Description: إنشاء جدولي الصناديق (Cashboxes) ومولّد الأرقام التسلسلية للسندات
-- Date: 2026-08-07
-- ============================================================================

-- Enable UUID extension if not exists
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================================
-- جدول الصناديق
-- يمثل مكاناً مالياً: صندوق نقدي، حساب بنكي، أو محفظة دفع
-- ============================================================================
CREATE TABLE IF NOT EXISTS cashboxes (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code             VARCHAR(20)  NOT NULL UNIQUE,
    name             VARCHAR(100) NOT NULL,
    name_en          VARCHAR(100),
    type             VARCHAR(20)  NOT NULL DEFAULT 'CASH'
                     CHECK (type IN ('CASH','BANK','WALLET','OTHER')),
    currency         VARCHAR(3)   NOT NULL DEFAULT 'SAR',
    bank_name        VARCHAR(100),
    account_holder   VARCHAR(100),
    account_number   VARCHAR(50),
    iban             VARCHAR(34),
    opening_balance  DECIMAL(15,4) NOT NULL DEFAULT 0,
    current_balance  DECIMAL(15,4) NOT NULL DEFAULT 0,
    min_balance      DECIMAL(15,4) NOT NULL DEFAULT 0,
    status           VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE'
                     CHECK (status IN ('ACTIVE','INACTIVE')),
    description      TEXT,
    created_by       UUID,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- فهارس
CREATE INDEX IF NOT EXISTS idx_cashboxes_type ON cashboxes(type);
CREATE INDEX IF NOT EXISTS idx_cashboxes_status ON cashboxes(status);
CREATE INDEX IF NOT EXISTS idx_cashboxes_created_at ON cashboxes(created_at);

-- ============================================================================
-- جدول مولّد أرقام السندات التسلسلية (ذرّي، بلا تكرار)
-- الصيغة: QABD-2026-000001 للسند قبض، SARF-2026-000001 لسند صرف
-- ============================================================================
CREATE TABLE IF NOT EXISTS voucher_counters (
    voucher_type VARCHAR(20) NOT NULL CHECK (voucher_type IN ('RECEIPT','PAYMENT')),
    fiscal_year  INT          NOT NULL,
    last_number  INT          NOT NULL DEFAULT 0,
    PRIMARY KEY (voucher_type, fiscal_year)
);
