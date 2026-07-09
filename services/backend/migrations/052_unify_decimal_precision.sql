-- Migration: 052_unify_decimal_precision.sql
-- Description: توحيد دقة جميع الأعمدة المالية إلى DECIMAL(12,4)
-- Date: 2026-03-22
-- Priority: HIGH

BEGIN;

-- ================================================================
-- 1. توحيد دقة accounting_transactions
-- ================================================================
DO $$
BEGIN
    -- amount
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'accounting_transactions' 
            AND column_name = 'amount'
    ) THEN
        ALTER TABLE accounting_transactions 
            ALTER COLUMN amount TYPE DECIMAL(12, 4);
        
        -- إضافة constraint لمنع القيم السالبة
        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint
            WHERE conname = 'accounting_transactions_amount_non_negative'
        ) THEN
            ALTER TABLE accounting_transactions
                ADD CONSTRAINT accounting_transactions_amount_non_negative
                    CHECK (amount >= 0);
        END IF;
        
        RAISE NOTICE 'Updated accounting_transactions.amount to DECIMAL(12,4)';
    END IF;
END $$;

COMMENT ON COLUMN accounting_transactions.amount IS
    'قيمة المعاملة المالية - DECIMAL(12,4) - لا يمكن أن تكون سالبة';

-- ================================================================
-- 2. توحيد دقة payment_transactions
-- ================================================================
DO $$
BEGIN
    -- amount
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'payment_transactions' 
            AND column_name = 'amount'
    ) THEN
        ALTER TABLE payment_transactions 
            ALTER COLUMN amount TYPE DECIMAL(12, 4);
        
        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint
            WHERE conname = 'payment_transactions_amount_non_negative'
        ) THEN
            ALTER TABLE payment_transactions
                ADD CONSTRAINT payment_transactions_amount_non_negative
                    CHECK (amount >= 0);
        END IF;
        
        RAISE NOTICE 'Updated payment_transactions.amount to DECIMAL(12,4)';
    END IF;

    -- refund_amount
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'payment_transactions' 
            AND column_name = 'refund_amount'
    ) THEN
        ALTER TABLE payment_transactions 
            ALTER COLUMN refund_amount TYPE DECIMAL(12, 4);
        
        -- constraint موجود بالفعل من migration 049
        RAISE NOTICE 'Updated payment_transactions.refund_amount to DECIMAL(12,4)';
    END IF;
END $$;

COMMENT ON COLUMN payment_transactions.amount IS
    'قيمة الدفعة - DECIMAL(12,4) - لا يمكن أن تكون سالبة';
COMMENT ON COLUMN payment_transactions.refund_amount IS
    'قيمة الاسترداد - DECIMAL(12,4) - لا يمكن أن تكون سالبة';

-- ================================================================
-- 3. التحقق من جداول أخرى قد تحتوي على أعمدة مالية
-- ================================================================
DO $$
BEGIN
    -- subscriptions (إن وجد)
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'subscriptions' 
            AND column_name = 'price'
    ) THEN
        ALTER TABLE subscriptions 
            ALTER COLUMN price TYPE DECIMAL(12, 4);
        
        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint
            WHERE conname = 'subscriptions_price_non_negative'
        ) THEN
            ALTER TABLE subscriptions
                ADD CONSTRAINT subscriptions_price_non_negative
                    CHECK (price >= 0);
        END IF;
        
        RAISE NOTICE 'Updated subscriptions.price to DECIMAL(12,4)';
    END IF;

    -- gateway_transactions (إن وجد)
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'gateway_transactions' 
            AND column_name = 'amount'
    ) THEN
        ALTER TABLE gateway_transactions 
            ALTER COLUMN amount TYPE DECIMAL(12, 4);
        
        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint
            WHERE conname = 'gateway_transactions_amount_non_negative'
        ) THEN
            ALTER TABLE gateway_transactions
                ADD CONSTRAINT gateway_transactions_amount_non_negative
                    CHECK (amount >= 0);
        END IF;
        
        RAISE NOTICE 'Updated gateway_transactions.amount to DECIMAL(12,4)';
    END IF;

    -- referral_rewards (إن وجد)
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'referral_rewards' 
            AND column_name = 'reward_amount'
    ) THEN
        ALTER TABLE referral_rewards 
            ALTER COLUMN reward_amount TYPE DECIMAL(12, 4);
        
        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint
            WHERE conname = 'referral_rewards_amount_non_negative'
        ) THEN
            ALTER TABLE referral_rewards
                ADD CONSTRAINT referral_rewards_amount_non_negative
                    CHECK (reward_amount >= 0);
        END IF;
        
        RAISE NOTICE 'Updated referral_rewards.reward_amount to DECIMAL(12,4)';
    END IF;
END $$;

-- ================================================================
-- 4. عرض ملخص جميع الأعمدة المالية
-- ================================================================
DO $$
DECLARE
    decimal_columns RECORD;
    total_count INT := 0;
BEGIN
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Financial Columns Summary';
    RAISE NOTICE '========================================';
    
    FOR decimal_columns IN
        SELECT 
            table_name,
            column_name,
            data_type,
            numeric_precision,
            numeric_scale
        FROM information_schema.columns
        WHERE data_type = 'numeric'
            AND table_schema = 'public'
        ORDER BY table_name, column_name
    LOOP
        RAISE NOTICE '% . % : DECIMAL(%, %)', 
            decimal_columns.table_name,
            decimal_columns.column_name,
            decimal_columns.numeric_precision,
            decimal_columns.numeric_scale;
        total_count := total_count + 1;
    END LOOP;
    
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Total financial columns: %', total_count;
    RAISE NOTICE '========================================';
END $$;

-- ================================================================
-- 5. التحقق من الـ constraints
-- ================================================================
DO $$
DECLARE
    constraint_count INT;
BEGIN
    SELECT COUNT(*) INTO constraint_count
    FROM pg_constraint
    WHERE conname LIKE '%_non_negative%'
        AND contype = 'c';
    
    RAISE NOTICE 'Total non-negative constraints: %', constraint_count;
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Decimal Precision Migration Complete';
    RAISE NOTICE '========================================';
END $$;

COMMIT;
