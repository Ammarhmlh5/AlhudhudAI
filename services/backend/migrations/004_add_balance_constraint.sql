-- Migration: Add balance constraint to prevent negative balance
-- Purpose: Database-level protection against negative balance (defense in depth)
-- Related Requirements: 11.2, 11.7

-- Add check constraint to tenants table to prevent negative balance
ALTER TABLE tenants 
ADD CONSTRAINT tenants_balance_non_negative CHECK (balance >= 0);

-- Add comment
COMMENT ON CONSTRAINT tenants_balance_non_negative ON tenants IS 'Prevents negative balance at database level - critical financial protection';
