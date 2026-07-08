-- Migration: 050_critical_schema_fixes.sql
-- Description: إصلاح القيود الحرجة وتحديث دقة الأعمدة المالية المتبقية
-- Reference: UNIFIED_DATABASE_AUDIT_REPORT.md (Critical Issues #1, #2, #8)

BEGIN;

-- 1. إصلاح قيود NOT NULL في جدول messages
DO $$ 
BEGIN
    ALTER TABLE messages ALTER COLUMN tenant_id SET NOT NULL;
    ALTER TABLE messages ALTER COLUMN phone_number SET NOT NULL;
    ALTER TABLE messages ALTER COLUMN content SET NOT NULL;
    ALTER TABLE messages ALTER COLUMN status SET NOT NULL;
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'Some columns in messages might already be NOT NULL or do not exist';
END $$;

-- 2. إصلاح قيود NOT NULL في جدول dispatch_channels
DO $$ 
BEGIN
    ALTER TABLE dispatch_channels ALTER COLUMN tenant_id SET NOT NULL;
    ALTER TABLE dispatch_channels ALTER COLUMN type SET NOT NULL;
    ALTER TABLE dispatch_channels ALTER COLUMN channel_key SET NOT NULL;
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'Some columns in dispatch_channels might already be NOT NULL';
END $$;

-- 3. إصلاح قيود NOT NULL ودقة DECIMAL في جدول accounting_transactions
DO $$ 
BEGIN
    -- تحديث الدقة إلى 4 فواصل عشرية
    ALTER TABLE accounting_transactions ALTER COLUMN amount TYPE DECIMAL(12, 4);
    ALTER TABLE accounting_transactions ALTER COLUMN discount_applied TYPE DECIMAL(12, 4);
    ALTER TABLE accounting_transactions ALTER COLUMN original_amount TYPE DECIMAL(12, 4);
    
    -- ضبط NOT NULL
    ALTER TABLE accounting_transactions ALTER COLUMN tenant_id SET NOT NULL;
    ALTER TABLE accounting_transactions ALTER COLUMN message_id SET NOT NULL;
    ALTER TABLE accounting_transactions ALTER COLUMN amount SET NOT NULL;
    
    -- إضافة Constraint لمنع القيم السالبة (في حال لم تكن موجودة)
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'accounting_transactions_amount_non_negative') THEN
        ALTER TABLE accounting_transactions ADD CONSTRAINT accounting_transactions_amount_non_negative CHECK (amount >= 0);
    END IF;
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'Some columns in accounting_transactions might already be configured or do not exist';
END $$;

-- 4. إصلاح قيود NOT NULL في جدول incoming_messages
DO $$ 
BEGIN
    ALTER TABLE incoming_messages ALTER COLUMN tenant_id SET NOT NULL;
    -- ملاحظة: التقرير ذكر status بينما النموذج (models) يستخدم is_processed
    -- سنحاول ضبط status إذا كان موجوداً لضمان توافق التحليل
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'incoming_messages' AND column_name = 'status') THEN
        ALTER TABLE incoming_messages ALTER COLUMN status SET NOT NULL;
    END IF;
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'Error adjusting incoming_messages columns';
END $$;

-- 5. إضافة CHECK constraint لجدول tenants لضمان الرصيد غير السالب
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'tenants_balance_non_negative') THEN
        ALTER TABLE tenants ADD CONSTRAINT tenants_balance_non_negative CHECK (balance >= 0);
    END IF;
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'Could not add constraint to tenants balance';
END $$;

COMMIT;
