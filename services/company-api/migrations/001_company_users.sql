-- Company Portal authentication + grants
-- Run as a superuser / migration owner once on the shared al_hudhud database.

CREATE TABLE IF NOT EXISTS company_users (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    full_name TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_company_users_tenant ON company_users(tenant_id);
CREATE INDEX IF NOT EXISTS idx_company_users_email ON company_users(LOWER(email));

-- Grants for the company-api database role.
-- The company-api role is NOT a table owner, so it needs explicit grants plus
-- RLS-compliant access (tenant rows are reached via SET app.current_tenant).
DO $$
DECLARE r TEXT;
BEGIN
    SELECT rolname INTO r FROM pg_roles WHERE rolname = 'company_api_user';
    IF r IS NOT NULL THEN
        GRANT USAGE ON SCHEMA public TO company_api_user;
        GRANT SELECT, INSERT, UPDATE, DELETE ON company_users TO company_api_user;
        GRANT USAGE, SELECT ON SEQUENCE company_users_id_seq TO company_api_user;
        GRANT SELECT, INSERT, UPDATE ON tenants TO company_api_user;
        GRANT SELECT ON packages TO company_api_user;
        GRANT SELECT, UPDATE ON subscriptions TO company_api_user;
        GRANT SELECT ON rollover_messages TO company_api_user;
        GRANT SELECT, INSERT ON accounting_transactions TO company_api_user;
    END IF;
END $$;
