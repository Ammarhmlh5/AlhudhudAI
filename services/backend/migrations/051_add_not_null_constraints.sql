-- Migration: 051_add_not_null_constraints.sql
-- Description: إضافة NOT NULL constraints على الأعمدة المهمة
-- Date: 2026-03-22
-- Priority: HIGH

BEGIN;

-- ================================================================
-- 1. إضافة NOT NULL على tenants.balance
-- ================================================================
-- التحقق من عدم وجود قيم NULL أولاً
UPDATE tenants SET balance = 0 WHERE balance IS NULL;

-- إضافة NOT NULL constraint
ALTER TABLE tenants 
    ALTER COLUMN balance SET NOT NULL,
    ALTER COLUMN balance SET DEFAULT 0;

COMMENT ON COLUMN tenants.balance IS 
    'رصيد المستأجر - DECIMAL(12,4) - NOT NULL - محمي من السالب';

-- ================================================================
-- 2. التحقق من messages - الأعمدة يجب أن تكون NOT NULL بالفعل
-- ================================================================
-- هذه الأعمدة محددة في models.go كـ NOT NULL
-- نتحقق فقط من أنها مطبقة في قاعدة البيانات

DO $$
BEGIN
    -- tenant_id
    -- تحديث القيم NULL إن وجدت
    DELETE FROM messages WHERE tenant_id IS NULL;
    
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'messages' 
            AND column_name = 'tenant_id'
            AND is_nullable = 'YES'
    ) THEN
        ALTER TABLE messages 
            ALTER COLUMN tenant_id SET NOT NULL;
        
        RAISE NOTICE 'Added NOT NULL to messages.tenant_id';
    ELSE
        RAISE NOTICE 'messages.tenant_id already NOT NULL';
    END IF;

    -- phone_number
    DELETE FROM messages WHERE phone_number IS NULL OR phone_number = '';
    
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'messages' 
            AND column_name = 'phone_number'
            AND is_nullable = 'YES'
    ) THEN
        ALTER TABLE messages 
            ALTER COLUMN phone_number SET NOT NULL;
        
        RAISE NOTICE 'Added NOT NULL to messages.phone_number';
    ELSE
        RAISE NOTICE 'messages.phone_number already NOT NULL';
    END IF;

    -- content
    DELETE FROM messages WHERE content IS NULL OR content = '';
    
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'messages' 
            AND column_name = 'content'
            AND is_nullable = 'YES'
    ) THEN
        ALTER TABLE messages 
            ALTER COLUMN content SET NOT NULL;
        
        RAISE NOTICE 'Added NOT NULL to messages.content';
    ELSE
        RAISE NOTICE 'messages.content already NOT NULL';
    END IF;
END $$;

-- ================================================================
-- 3. التحقق من dispatch_channels
-- ================================================================
DO $$
BEGIN
    -- type
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'dispatch_channels' 
            AND column_name = 'type'
            AND is_nullable = 'YES'
    ) THEN
        DELETE FROM dispatch_channels WHERE type IS NULL;
        
        ALTER TABLE dispatch_channels 
            ALTER COLUMN type SET NOT NULL;
        
        RAISE NOTICE 'Added NOT NULL to dispatch_channels.type';
    ELSE
        RAISE NOTICE 'dispatch_channels.type already NOT NULL';
    END IF;

    -- channel_key
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'dispatch_channels' 
            AND column_name = 'channel_key'
            AND is_nullable = 'YES'
    ) THEN
        DELETE FROM dispatch_channels WHERE channel_key IS NULL OR channel_key = '';
        
        ALTER TABLE dispatch_channels 
            ALTER COLUMN channel_key SET NOT NULL;
        
        RAISE NOTICE 'Added NOT NULL to dispatch_channels.channel_key';
    ELSE
        RAISE NOTICE 'dispatch_channels.channel_key already NOT NULL';
    END IF;
END $$;

-- ================================================================
-- 4. إضافة CHECK constraints إضافية
-- ================================================================

-- tenants.balance >= 0
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'tenants_balance_non_negative'
    ) THEN
        ALTER TABLE tenants
            ADD CONSTRAINT tenants_balance_non_negative
                CHECK (balance >= 0);
        RAISE NOTICE 'Added CHECK constraint: tenants_balance_non_negative';
    END IF;
END $$;

-- messages.phone_number not empty
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'messages_phone_number_not_empty'
    ) THEN
        ALTER TABLE messages
            ADD CONSTRAINT messages_phone_number_not_empty
                CHECK (length(phone_number) > 0);
        RAISE NOTICE 'Added CHECK constraint: messages_phone_number_not_empty';
    END IF;
END $$;

-- messages.content not empty
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'messages_content_not_empty'
    ) THEN
        ALTER TABLE messages
            ADD CONSTRAINT messages_content_not_empty
                CHECK (length(content) > 0);
        RAISE NOTICE 'Added CHECK constraint: messages_content_not_empty';
    END IF;
END $$;

-- ================================================================
-- 5. التحقق والإحصائيات
-- ================================================================
DO $$
DECLARE
    constraint_count INT;
BEGIN
    SELECT COUNT(*) INTO constraint_count
    FROM pg_constraint
    WHERE conname LIKE '%non_negative%' 
        OR conname LIKE '%not_empty%';
    
    RAISE NOTICE '========================================';
    RAISE NOTICE 'NOT NULL Constraints Migration Complete';
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Total CHECK constraints added: %', constraint_count;
    RAISE NOTICE '========================================';
END $$;

COMMIT;
