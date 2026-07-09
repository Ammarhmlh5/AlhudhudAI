-- Migration 069: Phase 2 - Extend messages table for Parsing & Pricing
-- This migration adds the fields needed after JSON parsing in Phase 2.
-- BEGIN;

-- 1. Add event tracking link (to global_incoming_events)
ALTER TABLE messages
    ADD COLUMN IF NOT EXISTS global_event_id BIGINT REFERENCES global_incoming_events(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_messages_global_event_id ON messages(global_event_id);

-- 2. Add message type column (INVOICE, PAYMENT, ALERT, REVERSAL, PROMO, OTHER)
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'message_category') THEN
        CREATE TYPE message_category AS ENUM (
            'INVOICE',      -- فاتورة
            'PAYMENT',      -- سداد / دفع
            'ALERT',        -- تنبيه / إشعار
            'REVERSAL',     -- عكس تسديد
            'PROMO',        -- إعلان / عرض ترويجي
            'OTP',          -- رقم تحقق
            'REMINDER',     -- تذكير
            'OTHER'         -- أخرى
        );
    END IF;
END $$;

ALTER TABLE messages
    ADD COLUMN IF NOT EXISTS message_type message_category DEFAULT 'OTHER';

CREATE INDEX IF NOT EXISTS idx_messages_message_type ON messages(message_type);

-- 3. Add character count and pricing columns
ALTER TABLE messages
    ADD COLUMN IF NOT EXISTS char_count      INT          NOT NULL DEFAULT 0,   -- عدد أحرف الرسالة
    ADD COLUMN IF NOT EXISTS parts_count     INT          NOT NULL DEFAULT 1,   -- عدد الأجزاء المخصومة
    ADD COLUMN IF NOT EXISTS encoding        VARCHAR(10)  NOT NULL DEFAULT 'GSM7'; -- GSM7 or UNICODE

-- 4. Add sender context columns
ALTER TABLE messages
    ADD COLUMN IF NOT EXISTS sender_user_id  UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS sender_device_id UUID REFERENCES paired_devices(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_messages_sender_user   ON messages(sender_user_id);
CREATE INDEX IF NOT EXISTS idx_messages_sender_device ON messages(sender_device_id);

-- 5. cost column precision upgrade was already handled in Migration 066.

-- 6. Add performance index for status+scheduled combined for the Dispatcher
-- (the existing idx_messages_tenant_status_scheduled covers per-tenant, this covers global pull)
CREATE INDEX IF NOT EXISTS idx_messages_status_scheduled_global
    ON messages (status, scheduled_at)
    WHERE status = 'PENDING';

-- 7. Add missing composite index for accounting lookups by message
CREATE INDEX IF NOT EXISTS idx_accounting_transactions_tenant_message
    ON accounting_transactions (tenant_id, message_id);

-- 8. Add idempotency partial unique index on messages (prevents duplicate submits)
CREATE UNIQUE INDEX IF NOT EXISTS idx_messages_idempotency_unique
    ON messages (tenant_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL AND idempotency_key <> '';
