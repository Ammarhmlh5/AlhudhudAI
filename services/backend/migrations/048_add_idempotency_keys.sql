-- Migration: 048_add_idempotency_keys.sql
-- Description: Add idempotency_key to subscriptions and payment_transactions
-- Requirements: 2.4

BEGIN;

-- Add idempotency_key to subscriptions
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(255);
-- Add unique constraint (tenant_id, idempotency_key)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'subscriptions_idempotency_key_unique') THEN
        ALTER TABLE subscriptions ADD CONSTRAINT subscriptions_idempotency_key_unique UNIQUE (tenant_id, idempotency_key);
    END IF;
END $$;

-- Add idempotency_key to payment_transactions
ALTER TABLE payment_transactions ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(255);
-- Add unique constraint (tenant_id, idempotency_key)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'payment_transactions_idempotency_key_unique') THEN
        ALTER TABLE payment_transactions ADD CONSTRAINT payment_transactions_idempotency_key_unique UNIQUE (tenant_id, idempotency_key);
    END IF;
END $$;

COMMIT;
