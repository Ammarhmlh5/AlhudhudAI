-- Migration: Create billing_invoices table (Fixed version)
-- Purpose: Support automatic invoice generation as expected by gateway_billing.go

CREATE TABLE IF NOT EXISTS billing_invoices (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    invoice_number VARCHAR(50) NOT NULL,
    invoice_type VARCHAR(50), -- Type of invoice (e.g., gateway_subscription)
    amount DECIMAL(12, 4) NOT NULL,
    currency VARCHAR(10) DEFAULT 'USD',
    tax_amount DECIMAL(12, 4) DEFAULT 0.00,
    total_amount DECIMAL(12, 4) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending', -- 'pending', 'paid', 'failed', 'cancelled'
    issue_date DATE NOT NULL DEFAULT CURRENT_DATE,
    due_date DATE,
    paid_date DATE,
    description TEXT,
    reference_type VARCHAR(50),
    reference_id VARCHAR(100),
    pdf_generated BOOLEAN DEFAULT false,
    pdf_path VARCHAR(500),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance and isolation
CREATE INDEX idx_invoices_tenant ON billing_invoices(tenant_id);
CREATE INDEX idx_invoices_tenant_status ON billing_invoices(tenant_id, status);
CREATE INDEX idx_invoices_ref ON billing_invoices(reference_type, reference_id);

-- UNIQUE constraint: invoice_number must be unique per tenant
CREATE UNIQUE INDEX idx_invoices_tenant_number_unique ON billing_invoices(tenant_id, invoice_number);

-- RLS
ALTER TABLE billing_invoices ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON billing_invoices
    USING (tenant_id = (current_setting('app.current_tenant', true))::uuid OR (current_setting('app.is_super_admin', true) = 'true'));
