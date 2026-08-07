-- Migration 096: Finance Wallet Accounts
-- Description: Create a backend mapping table for finance wallet account IDs linked to tenants and resellers.

CREATE TABLE IF NOT EXISTS finance_wallet_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reference_id UUID NOT NULL,
    reference_type VARCHAR(50) NOT NULL CHECK (reference_type IN ('TENANT', 'RESELLER')),
    wallet_account_id UUID NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_finance_wallet_accounts_reference
    ON finance_wallet_accounts(reference_type, reference_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_finance_wallet_accounts_wallet_id
    ON finance_wallet_accounts(wallet_account_id);

-- If the database uses row-level security, restrict direct access to service role only.
ALTER TABLE finance_wallet_accounts ENABLE ROW LEVEL SECURITY;

CREATE POLICY IF NOT EXISTS finance_wallet_accounts_service_role_policy
    ON finance_wallet_accounts
    FOR ALL
    USING (current_setting('request.jwt.claims', true)::json->>'role' = 'service_role');
