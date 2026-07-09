-- Migration: Recreate dedicated_gateways with UUID
-- Purpose: Match Go struct and standardize on UUID PKs

-- DROP IF EXISTS (be careful! but we checked and it has 0 rows)
DROP TABLE IF EXISTS dedicated_gateways CASCADE;

CREATE TABLE dedicated_gateways (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    gateway_name VARCHAR(255) NOT NULL,
    gateway_code VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    telecom_provider VARCHAR(100) NOT NULL,
    telecom_api_endpoint TEXT NOT NULL,
    telecom_username VARCHAR(255),
    telecom_password_encrypted TEXT,
    telecom_api_key_encrypted TEXT,
    server_type VARCHAR(50) DEFAULT 'shared',
    server_ip VARCHAR(45),
    server_hostname VARCHAR(255),
    server_specs JSONB,
    max_throughput INTEGER DEFAULT 100,
    daily_limit INTEGER,
    monthly_limit INTEGER,
    subscription_type VARCHAR(20) DEFAULT 'monthly',
    subscription_fee DECIMAL(12, 4) DEFAULT 0.00,
    server_rental_fee DECIMAL(12, 4) DEFAULT 0.00,
    setup_fee DECIMAL(12, 4) DEFAULT 0.00,
    billing_cycle VARCHAR(20) DEFAULT 'monthly',
    next_billing_date TIMESTAMP,
    telecom_balance DECIMAL(12, 4),
    balance_alert_threshold DECIMAL(12, 4),
    last_balance_check TIMESTAMP,
    balance_check_enabled BOOLEAN DEFAULT true,
    contract_start_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    contract_end_date TIMESTAMP,
    auto_renew BOOLEAN DEFAULT false,
    status VARCHAR(20) DEFAULT 'active',
    last_health_check TIMESTAMP,
    health_status VARCHAR(20) DEFAULT 'unknown',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_dedicated_gateways_tenant ON dedicated_gateways(tenant_id);
CREATE INDEX idx_dedicated_gateways_code ON dedicated_gateways(gateway_code);

-- Enable RLS
ALTER TABLE dedicated_gateways ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_policy ON dedicated_gateways
    USING (tenant_id = (current_setting('app.current_tenant', true))::uuid);
