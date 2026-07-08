-- Migration: Test Mode Operations Logging
-- Description: Creates table for logging test mode operations
-- Requirements: 38.10

-- Test Operations Table
CREATE TABLE IF NOT EXISTS test_operations (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    api_key VARCHAR(100) NOT NULL,
    operation VARCHAR(50) NOT NULL,
    test_number VARCHAR(20),
    result VARCHAR(50),
    request_data TEXT,
    response_data JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Enable Row Level Security
ALTER TABLE test_operations ENABLE ROW LEVEL SECURITY;

-- RLS Policies for test_operations
CREATE POLICY test_operations_tenant_isolation ON test_operations
    USING (tenant_id = current_setting('app.current_tenant')::bigint);

CREATE POLICY test_operations_tenant_insert ON test_operations
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::bigint);

-- Indexes for test_operations
CREATE INDEX idx_test_operations_tenant ON test_operations(tenant_id);
CREATE INDEX idx_test_operations_created ON test_operations(created_at DESC);
CREATE INDEX idx_test_operations_api_key ON test_operations(api_key);
CREATE INDEX idx_test_operations_result ON test_operations(result);
CREATE INDEX idx_test_operations_operation ON test_operations(operation);

-- Function to clean up old test operations (older than 30 days)
CREATE OR REPLACE FUNCTION cleanup_old_test_operations()
RETURNS void AS $$
BEGIN
    DELETE FROM test_operations
    WHERE created_at < NOW() - INTERVAL '30 days';
END;
$$ LANGUAGE plpgsql;

-- Comments
COMMENT ON TABLE test_operations IS 'Logs all test mode operations for debugging and monitoring';
COMMENT ON COLUMN test_operations.api_key IS 'Test API key used for the operation';
COMMENT ON COLUMN test_operations.operation IS 'Type of operation (send_message, check_balance, etc.)';
COMMENT ON COLUMN test_operations.test_number IS 'Test phone number used';
COMMENT ON COLUMN test_operations.result IS 'Result of the test operation';
COMMENT ON COLUMN test_operations.request_data IS 'Request parameters';
COMMENT ON COLUMN test_operations.response_data IS 'Response data in JSON format';
