-- ============================================================================
-- Migration: 094 - نظام الحسابات: سندات القبض والصرف وسجل حركة الصندوق
-- Description: إنشاء جدولي سندات القبض والصرف وجدول سجل حركة الصناديق
-- Date: 2026-08-07
-- ============================================================================

-- Enable UUID extension if not exists
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================================
-- جدول سندات القبض (المبالغ الداخلة)
-- ============================================================================
CREATE TABLE IF NOT EXISTS receipt_vouchers (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    voucher_number   VARCHAR(40) NOT NULL UNIQUE,
    voucher_date     DATE        NOT NULL DEFAULT CURRENT_DATE,
    cashbox_id       UUID        NOT NULL REFERENCES cashboxes(id),
    amount           DECIMAL(15,4) NOT NULL CHECK (amount > 0),
    currency         VARCHAR(3)  NOT NULL DEFAULT 'SAR',
    payer_name       VARCHAR(200) NOT NULL,
    payer_type       VARCHAR(20) NOT NULL DEFAULT 'OTHER'
                     CHECK (payer_type IN ('TENANT','RESELLER','CUSTOMER','OTHER')),
    tenant_id        UUID,
    reseller_id      UUID,
    payment_method   VARCHAR(20) NOT NULL DEFAULT 'CASH'
                     CHECK (payment_method IN ('CASH','BANK_TRANSFER','CHEQUE','CARD','WALLET')),
    reference_number VARCHAR(100),
    description      TEXT,
    status           VARCHAR(20) NOT NULL DEFAULT 'POSTED'
                     CHECK (status IN ('DRAFT','POSTED','REVERSED')),
    reversal_of      UUID,
    credited_wallet  BOOLEAN     NOT NULL DEFAULT FALSE,
    created_by       UUID,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    posted_at        TIMESTAMPTZ,
    posted_by        UUID,
    reversed_at      TIMESTAMPTZ,
    reversed_by      UUID
);

-- فهارس
CREATE INDEX IF NOT EXISTS idx_receipt_vouchers_cashbox ON receipt_vouchers(cashbox_id);
CREATE INDEX IF NOT EXISTS idx_receipt_vouchers_date ON receipt_vouchers(voucher_date);
CREATE INDEX IF NOT EXISTS idx_receipt_vouchers_status ON receipt_vouchers(status);
CREATE INDEX IF NOT EXISTS idx_receipt_vouchers_tenant ON receipt_vouchers(tenant_id) WHERE tenant_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_receipt_vouchers_created_at ON receipt_vouchers(created_at);

-- ============================================================================
-- جدول سندات الصرف (المبالغ الخارجة)
-- ============================================================================
CREATE TABLE IF NOT EXISTS payment_vouchers (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    voucher_number   VARCHAR(40) NOT NULL UNIQUE,
    voucher_date     DATE        NOT NULL DEFAULT CURRENT_DATE,
    cashbox_id       UUID        NOT NULL REFERENCES cashboxes(id),
    amount           DECIMAL(15,4) NOT NULL CHECK (amount > 0),
    currency         VARCHAR(3)  NOT NULL DEFAULT 'SAR',
    payee_name       VARCHAR(200) NOT NULL,
    payee_type       VARCHAR(20) NOT NULL DEFAULT 'OTHER'
                     CHECK (payee_type IN ('TENANT','RESELLER','SUPPLIER','OTHER')),
    tenant_id        UUID,
    reseller_id      UUID,
    payment_method   VARCHAR(20) NOT NULL DEFAULT 'CASH'
                     CHECK (payment_method IN ('CASH','BANK_TRANSFER','CHEQUE','CARD','WALLET')),
    reference_number VARCHAR(100),
    description      TEXT,
    status           VARCHAR(20) NOT NULL DEFAULT 'POSTED'
                     CHECK (status IN ('DRAFT','POSTED','REVERSED')),
    reversal_of      UUID,
    debited_wallet   BOOLEAN     NOT NULL DEFAULT FALSE,
    created_by       UUID,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    posted_at        TIMESTAMPTZ,
    posted_by        UUID,
    reversed_at      TIMESTAMPTZ,
    reversed_by      UUID
);

-- فهارس
CREATE INDEX IF NOT EXISTS idx_payment_vouchers_cashbox ON payment_vouchers(cashbox_id);
CREATE INDEX IF NOT EXISTS idx_payment_vouchers_date ON payment_vouchers(voucher_date);
CREATE INDEX IF NOT EXISTS idx_payment_vouchers_status ON payment_vouchers(status);
CREATE INDEX IF NOT EXISTS idx_payment_vouchers_tenant ON payment_vouchers(tenant_id) WHERE tenant_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_payment_vouchers_created_at ON payment_vouchers(created_at);

-- ============================================================================
-- جدول سجل حركة الصندوق (أساس كشف حساب الصندوق)
-- ============================================================================
CREATE TABLE IF NOT EXISTS cashbox_movements (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    cashbox_id     UUID NOT NULL REFERENCES cashboxes(id),
    voucher_type   VARCHAR(20) NOT NULL CHECK (voucher_type IN ('RECEIPT','PAYMENT')),
    voucher_id     UUID NOT NULL,
    voucher_number VARCHAR(40) NOT NULL,
    movement_date  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    amount         DECIMAL(15,4) NOT NULL CHECK (amount <> 0),
    balance_after  DECIMAL(15,4) NOT NULL,
    description    TEXT,
    created_by     UUID
);

-- فهارس
CREATE INDEX IF NOT EXISTS idx_cashbox_movements_box_date ON cashbox_movements(cashbox_id, movement_date DESC);
CREATE INDEX IF NOT EXISTS idx_cashbox_movements_voucher ON cashbox_movements(voucher_id);
