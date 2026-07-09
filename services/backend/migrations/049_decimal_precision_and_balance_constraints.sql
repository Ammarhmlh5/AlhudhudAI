-- Migration: 049_decimal_precision_and_balance_constraints.sql
-- Description: ضبط دقة الفواصل العشرية في الأعمدة المالية ومنع الأرصدة السالبة
-- Point 4.4: توثيق وضبط دقة الفواصل العشرية وتضمين Constraint لمنع أي أرصدة بالسالب

BEGIN;

-- ================================================================
-- 1. تحسين دقة عمود balance في جدول tenants إلى DECIMAL(12,4)
-- ================================================================
ALTER TABLE tenants
    ALTER COLUMN balance TYPE DECIMAL(12, 4);

-- ================================================================
-- 2. تحسين دقة الأعمدة المالية في جدول balance_history
-- ================================================================
ALTER TABLE balance_history
    ALTER COLUMN old_balance TYPE DECIMAL(12, 4),
    ALTER COLUMN new_balance TYPE DECIMAL(12, 4),
    ALTER COLUMN amount      TYPE DECIMAL(12, 4);

-- إضافة Constraint لمنع وجود رصيد سالب بعد أي عملية (idempotent)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'balance_history_new_balance_non_negative'
    ) THEN
        ALTER TABLE balance_history
            ADD CONSTRAINT balance_history_new_balance_non_negative
                CHECK (new_balance >= 0);
    END IF;
END $$;

-- ================================================================
-- 3. تحسين دقة تكلفة الرسائل في جدول messages (إن وُجد)
-- ================================================================
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'messages' AND column_name = 'cost'
    ) THEN
        ALTER TABLE messages ALTER COLUMN cost TYPE DECIMAL(12, 4);

        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint
            WHERE conname = 'messages_cost_non_negative'
        ) THEN
            ALTER TABLE messages
                ADD CONSTRAINT messages_cost_non_negative CHECK (cost >= 0);
        END IF;
    END IF;
END $$;

-- ================================================================
-- 4. تحسين دقة جدول archived_messages (إن وُجد)
-- ================================================================
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'archived_messages' AND column_name = 'cost'
    ) THEN
        ALTER TABLE archived_messages ALTER COLUMN cost TYPE DECIMAL(12, 4);
    END IF;
END $$;

-- ================================================================
-- 5. منع مبالغ سالبة في payment_transactions
-- ================================================================
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'payment_transactions_refund_non_negative'
    ) THEN
        ALTER TABLE payment_transactions
            ADD CONSTRAINT payment_transactions_refund_non_negative
                CHECK (refund_amount IS NULL OR refund_amount >= 0);
    END IF;
END $$;

-- ================================================================
-- 6. تحسين دقة أرقام الباقات المالية
-- ================================================================
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'packages' AND column_name = 'price'
    ) THEN
        ALTER TABLE packages ALTER COLUMN price TYPE DECIMAL(12, 4);

        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint
            WHERE conname = 'packages_price_non_negative'
        ) THEN
            ALTER TABLE packages
                ADD CONSTRAINT packages_price_non_negative CHECK (price >= 0);
        END IF;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'packages' AND column_name = 'cost_per_message'
    ) THEN
        ALTER TABLE packages ALTER COLUMN cost_per_message TYPE DECIMAL(12, 6);

        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint
            WHERE conname = 'packages_cost_per_message_non_negative'
        ) THEN
            ALTER TABLE packages
                ADD CONSTRAINT packages_cost_per_message_non_negative
                    CHECK (cost_per_message >= 0);
        END IF;
    END IF;
END $$;

-- ================================================================
-- 7. توثيق التغييرات
-- ================================================================
COMMENT ON COLUMN tenants.balance IS
    'رصيد المستأجر بدقة 4 فواصل عشرية - DECIMAL(12,4) - محمي من السالب عبر constraint';

COMMENT ON COLUMN balance_history.amount IS
    'قيمة العملية المالية بدقة 4 فواصل عشرية - DECIMAL(12,4)';

COMMENT ON COLUMN balance_history.new_balance IS
    'الرصيد بعد العملية بدقة 4 فواصل عشرية - DECIMAL(12,4) - لا يمكن أن يكون سالباً';

COMMIT;
