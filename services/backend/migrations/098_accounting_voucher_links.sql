-- ============================================================================
-- Migration: 098 - إضافة روابط القيود المالية لسندات القبض والصرف
-- Description: إضافة journal_entry_id لربط السندات بالقيود المزدوجة
-- Date: 2026-08-08
-- ============================================================================

-- Add journal_entry_id to receipt_vouchers if not exists
ALTER TABLE receipt_vouchers ADD COLUMN IF NOT EXISTS journal_entry_id UUID REFERENCES journal_entries(id);

-- Add journal_entry_id to payment_vouchers if not exists
ALTER TABLE payment_vouchers ADD COLUMN IF NOT EXISTS journal_entry_id UUID REFERENCES journal_entries(id);

-- Add indexes for faster lookups
CREATE INDEX IF NOT EXISTS idx_receipt_vouchers_journal ON receipt_vouchers(journal_entry_id) WHERE journal_entry_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_payment_vouchers_journal ON payment_vouchers(journal_entry_id) WHERE journal_entry_id IS NOT NULL;