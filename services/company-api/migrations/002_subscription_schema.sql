-- Migration: 002_subscription_schema.sql
-- Description: Additive subscription schema for the Company Portal feature.
-- Safe to re-run. Creates NEW tables only (subscriptions, rollover_messages)
-- and ADDS columns/indexes to the existing `packages` table. No drops, no
-- changes to existing data, and no RLS is enabled (matches the live DB state
-- where only `referrals` uses RLS; table owners bypass RLS regardless).

BEGIN;

-- 1. subscriptions (from migration 017 + 048 idempotency_key)
CREATE TABLE IF NOT EXISTS subscriptions (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,
    package_id INTEGER NOT NULL,

    start_date DATE NOT NULL,
    end_date DATE NOT NULL,

    messages_used INTEGER DEFAULT 0,
    messages_remaining INTEGER,
    overage_messages INTEGER DEFAULT 0,
    overage_cost DECIMAL(10, 2) DEFAULT 0.00,

    status VARCHAR(20) DEFAULT 'active',
    auto_renew BOOLEAN DEFAULT true,
    grace_period_days INTEGER DEFAULT 3,

    last_renewed_at TIMESTAMP,
    next_renewal_date DATE,
    renewal_attempts INTEGER DEFAULT 0,
    last_renewal_attempt TIMESTAMP,

    idempotency_key VARCHAR(255),

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT check_messages_used_non_negative CHECK (messages_used >= 0),
    CONSTRAINT check_overage_non_negative CHECK (overage_messages >= 0),
    CONSTRAINT check_overage_cost_non_negative CHECK (overage_cost >= 0),
    CONSTRAINT check_grace_period_positive CHECK (grace_period_days > 0),
    CONSTRAINT check_status_valid CHECK (status IN ('active', 'grace_period', 'suspended', 'cancelled', 'expired')),
    CONSTRAINT check_dates_valid CHECK (end_date >= start_date)
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_tenant ON subscriptions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_tenant_package ON subscriptions(tenant_id, package_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_tenant_status ON subscriptions(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_subscriptions_tenant_renewal ON subscriptions(tenant_id, next_renewal_date);
CREATE INDEX IF NOT EXISTS idx_subscriptions_tenant_dates ON subscriptions(tenant_id, start_date, end_date);
CREATE INDEX IF NOT EXISTS idx_subscriptions_tenant_auto_renew ON subscriptions(tenant_id, auto_renew);
CREATE INDEX IF NOT EXISTS idx_subscriptions_renewal_due ON subscriptions(next_renewal_date, status, auto_renew)
    WHERE status IN ('active', 'grace_period') AND auto_renew = true;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'subscriptions_idempotency_key_unique') THEN
        ALTER TABLE subscriptions ADD CONSTRAINT subscriptions_idempotency_key_unique UNIQUE (tenant_id, idempotency_key);
    END IF;
END $$;

-- 2. rollover_messages (from migration 018, extended with forfeiture columns
--    used by the subscription manager's upgrade path)
CREATE TABLE IF NOT EXISTS rollover_messages (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,
    subscription_id INTEGER NOT NULL,

    messages_count INTEGER NOT NULL,
    source_month DATE NOT NULL,
    expiry_date DATE,

    messages_used INTEGER DEFAULT 0,
    messages_remaining INTEGER NOT NULL,

    status VARCHAR(20) DEFAULT 'active',
    payment_status VARCHAR(20) DEFAULT 'pending',
    forfeited_reason TEXT,
    forfeited_at TIMESTAMP,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT check_rollover_positive CHECK (messages_count > 0),
    CONSTRAINT check_rollover_remaining CHECK (messages_remaining >= 0),
    CONSTRAINT check_rollover_used_valid CHECK (messages_used >= 0 AND messages_used <= messages_count),
    CONSTRAINT check_rollover_status_valid CHECK (status IN ('active', 'expired', 'consumed', 'pending', 'forfeited'))
);

CREATE INDEX IF NOT EXISTS idx_rollover_tenant ON rollover_messages(tenant_id);
CREATE INDEX IF NOT EXISTS idx_rollover_tenant_subscription ON rollover_messages(tenant_id, subscription_id);
CREATE INDEX IF NOT EXISTS idx_rollover_tenant_status ON rollover_messages(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_rollover_tenant_expiry ON rollover_messages(tenant_id, expiry_date);
CREATE INDEX IF NOT EXISTS idx_rollover_tenant_created ON rollover_messages(tenant_id, subscription_id, created_at)
    WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_rollover_expired ON rollover_messages(expiry_date, status)
    WHERE status = 'active' AND expiry_date IS NOT NULL;

-- 3. packages: add the subscription feature columns (live table is empty;
--    defaults keep legacy inserts working)
ALTER TABLE packages ADD COLUMN IF NOT EXISTS service_id BIGINT;
ALTER TABLE packages ADD COLUMN IF NOT EXISTS overage_rate NUMERIC(12, 6) DEFAULT 0;
ALTER TABLE packages ADD COLUMN IF NOT EXISTS is_unlimited BOOLEAN DEFAULT false;
ALTER TABLE packages ADD COLUMN IF NOT EXISTS billing_cycle VARCHAR(20) DEFAULT 'monthly';
ALTER TABLE packages ADD COLUMN IF NOT EXISTS rollover_enabled BOOLEAN DEFAULT false;
ALTER TABLE packages ADD COLUMN IF NOT EXISTS rollover_limit BIGINT;
ALTER TABLE packages ADD COLUMN IF NOT EXISTS rollover_expiry_months INTEGER;
ALTER TABLE packages ADD COLUMN IF NOT EXISTS display_order INTEGER DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_packages_tenant_service ON packages(tenant_id, service_id);
CREATE INDEX IF NOT EXISTS idx_packages_tenant_active ON packages(tenant_id, is_active);
CREATE INDEX IF NOT EXISTS idx_packages_tenant_order ON packages(tenant_id, display_order);
CREATE UNIQUE INDEX IF NOT EXISTS idx_packages_tenant_code_unique ON packages(tenant_id, code);

COMMIT;
