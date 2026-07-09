-- Migration: Data Deletion Requests Table
-- Description: Stores GDPR data deletion requests
-- Requirements: 36.5, 36.6, 36.7

-- Create data_deletion_requests table
CREATE TABLE IF NOT EXISTS data_deletion_requests (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    requested_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    reason TEXT,
    error_msg TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Enable Row Level Security
ALTER TABLE data_deletion_requests ENABLE ROW LEVEL SECURITY;

-- RLS Policy for SELECT
CREATE POLICY data_deletion_requests_tenant_isolation ON data_deletion_requests
    FOR SELECT
    USING (tenant_id = current_setting('app.current_tenant', true)::bigint);

-- RLS Policy for INSERT
CREATE POLICY data_deletion_requests_tenant_isolation_insert ON data_deletion_requests
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant', true)::bigint);

-- RLS Policy for UPDATE
CREATE POLICY data_deletion_requests_tenant_isolation_update ON data_deletion_requests
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant', true)::bigint);

-- Indexes for performance
CREATE INDEX idx_data_deletion_requests_tenant ON data_deletion_requests(tenant_id);
CREATE INDEX idx_data_deletion_requests_user ON data_deletion_requests(tenant_id, user_id);
CREATE INDEX idx_data_deletion_requests_status ON data_deletion_requests(status);
CREATE INDEX idx_data_deletion_requests_requested_at ON data_deletion_requests(requested_at DESC);

-- Trigger to update updated_at
CREATE OR REPLACE FUNCTION update_data_deletion_requests_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER data_deletion_requests_updated_at
    BEFORE UPDATE ON data_deletion_requests
    FOR EACH ROW
    EXECUTE FUNCTION update_data_deletion_requests_updated_at();

-- Comments
COMMENT ON TABLE data_deletion_requests IS 'Stores GDPR data deletion requests';
COMMENT ON COLUMN data_deletion_requests.status IS 'Status: pending, processing, completed, failed';
COMMENT ON COLUMN data_deletion_requests.reason IS 'User-provided reason for deletion';
COMMENT ON COLUMN data_deletion_requests.error_msg IS 'Error message if deletion failed';
