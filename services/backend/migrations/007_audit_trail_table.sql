-- Migration: Create Audit Trail Table
-- Purpose: Comprehensive audit logging for all critical operations
-- Requirements: 39.1, 39.2, 39.3, 39.4, 39.5

-- Create audit_trail table in public schema (shared across all tenants)
CREATE TABLE IF NOT EXISTS audit_trail (
    id BIGSERIAL PRIMARY KEY,
    
    -- User Information
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Action Details
    action VARCHAR(100) NOT NULL,
    resource VARCHAR(100) NOT NULL,
    resource_id VARCHAR(255),
    
    -- Request Information
    ip_address INET NOT NULL,
    user_agent TEXT,
    
    -- Result
    result VARCHAR(20) NOT NULL CHECK (result IN ('SUCCESS', 'FAILURE', 'PARTIAL')),
    error_message TEXT,
    
    -- Additional Context
    details JSONB,
    
    -- Timestamp (immutable - no updated_at)
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT audit_trail_action_check CHECK (action != ''),
    CONSTRAINT audit_trail_resource_check CHECK (resource != '')
);

-- Indexes for performance and querying
CREATE INDEX idx_audit_trail_tenant ON audit_trail(tenant_id, created_at DESC);
CREATE INDEX idx_audit_trail_user ON audit_trail(user_id, created_at DESC);
CREATE INDEX idx_audit_trail_action ON audit_trail(action, created_at DESC);
CREATE INDEX idx_audit_trail_resource ON audit_trail(resource, resource_id);
CREATE INDEX idx_audit_trail_result ON audit_trail(result, created_at DESC);
CREATE INDEX idx_audit_trail_created ON audit_trail(created_at DESC);

-- GIN index for JSONB details for efficient querying
CREATE INDEX idx_audit_trail_details ON audit_trail USING GIN (details);

-- Prevent updates and deletes (immutable audit log)
-- Create a trigger function to prevent modifications
CREATE OR REPLACE FUNCTION prevent_audit_trail_modification()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        RAISE EXCEPTION 'Audit trail records cannot be modified';
    ELSIF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'Audit trail records cannot be deleted';
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Create triggers to enforce immutability
CREATE TRIGGER prevent_audit_trail_update
    BEFORE UPDATE ON audit_trail
    FOR EACH ROW
    EXECUTE FUNCTION prevent_audit_trail_modification();

CREATE TRIGGER prevent_audit_trail_delete
    BEFORE DELETE ON audit_trail
    FOR EACH ROW
    EXECUTE FUNCTION prevent_audit_trail_modification();

-- Comment on table
COMMENT ON TABLE audit_trail IS 'Immutable audit log for all critical operations. Records cannot be modified or deleted.';
COMMENT ON COLUMN audit_trail.user_id IS 'User who performed the action (NULL for system actions)';
COMMENT ON COLUMN audit_trail.tenant_id IS 'Tenant context for the action';
COMMENT ON COLUMN audit_trail.action IS 'Action performed (e.g., BALANCE_DEDUCT, MESSAGE_SEND, USER_LOGIN)';
COMMENT ON COLUMN audit_trail.resource IS 'Resource type affected (e.g., BALANCE, MESSAGE, USER)';
COMMENT ON COLUMN audit_trail.resource_id IS 'ID of the affected resource';
COMMENT ON COLUMN audit_trail.result IS 'Result of the action: SUCCESS, FAILURE, or PARTIAL';
COMMENT ON COLUMN audit_trail.details IS 'Additional context in JSON format';
