-- Migration: User Consents Table
-- Description: Stores user consent records for GDPR/CCPA compliance
-- Requirements: 36.3, 36.4

-- Create user_consents table
CREATE TABLE IF NOT EXISTS user_consents (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    consent_type VARCHAR(50) NOT NULL CHECK (consent_type IN ('terms_of_service', 'privacy_policy', 'marketing', 'data_processing')),
    version VARCHAR(20) NOT NULL,
    granted BOOLEAN NOT NULL DEFAULT false,
    granted_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMP WITH TIME ZONE,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Enable Row Level Security
ALTER TABLE user_consents ENABLE ROW LEVEL SECURITY;

-- RLS Policy for SELECT
CREATE POLICY user_consents_tenant_isolation ON user_consents
    FOR SELECT
    USING (tenant_id = current_setting('app.current_tenant', true)::bigint);

-- RLS Policy for INSERT
CREATE POLICY user_consents_tenant_isolation_insert ON user_consents
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant', true)::bigint);

-- RLS Policy for UPDATE
CREATE POLICY user_consents_tenant_isolation_update ON user_consents
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant', true)::bigint);

-- RLS Policy for DELETE
CREATE POLICY user_consents_tenant_isolation_delete ON user_consents
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant', true)::bigint);

-- Indexes for performance
CREATE INDEX idx_user_consents_tenant_user ON user_consents(tenant_id, user_id);
CREATE INDEX idx_user_consents_type ON user_consents(consent_type);
CREATE INDEX idx_user_consents_granted_at ON user_consents(granted_at DESC);

-- Trigger to update updated_at
CREATE OR REPLACE FUNCTION update_user_consents_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER user_consents_updated_at
    BEFORE UPDATE ON user_consents
    FOR EACH ROW
    EXECUTE FUNCTION update_user_consents_updated_at();

-- Comments
COMMENT ON TABLE user_consents IS 'Stores user consent records for GDPR/CCPA compliance';
COMMENT ON COLUMN user_consents.consent_type IS 'Type of consent: terms_of_service, privacy_policy, marketing, data_processing';
COMMENT ON COLUMN user_consents.version IS 'Version of the document user consented to';
COMMENT ON COLUMN user_consents.granted IS 'Whether consent was granted or denied';
COMMENT ON COLUMN user_consents.ip_address IS 'IP address from which consent was given';
COMMENT ON COLUMN user_consents.user_agent IS 'Browser user agent string';
