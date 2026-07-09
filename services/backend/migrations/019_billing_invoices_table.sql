-- Migration: Add Billing Invoices Table
-- Version: 1.0
-- Date: 2026-02-21
-- Description: Creates billing_invoices table for automatic invoice generation with tenant isolation

-- This migration creates the billing_invoices table that tracks:
-- - Automatic invoice generation for all payment operations
-- - Unique invoice numbers for each invoice
-- - Invoice details (service, amount, tax, total)
-- - Invoice status (pending, paid, failed, cancelled)
-- - PDF generation and email delivery tracking

BEGIN;

-- 1. Create billing_invoices table
CREATE TABLE IF NOT EXISTS billing_invoices (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,  -- CRITICAL: Required for tenant isolation
    
    -- رقم الفاتورة (Invoice Number)
    invoice_number VARCHAR(50) NOT NULL,
    
    -- التفاصيل (Details)
    subscription_id INTEGER,  -- References subscriptions(id) - NULL for non-subscription payments
    transaction_id INTEGER,  -- References accounting_transactions(id)
    
    -- المبالغ (Amounts)
    subtotal DECIMAL(10, 2) NOT NULL,
    tax_rate DECIMAL(5, 2) DEFAULT 0.00,
    tax_amount DECIMAL(10, 2) DEFAULT 0.00,
    total_amount DECIMAL(10, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    
    -- الوصف (Description)
    description TEXT,
    service_name VARCHAR(255),
    
    -- التواريخ (Dates)
    issue_date DATE NOT NULL DEFAULT CURRENT_DATE,
    due_date DATE,
    paid_date DATE,
    
    -- الحالة (Status)
    status VARCHAR(20) DEFAULT 'pending', -- 'pending', 'paid', 'failed', 'cancelled'
    
    -- PDF والإرسال (PDF & Delivery)
    pdf_generated BOOLEAN DEFAULT false,
    pdf_path VARCHAR(500),
    email_sent BOOLEAN DEFAULT false,
    email_sent_at TIMESTAMP,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT check_invoice_subtotal_positive CHECK (subtotal > 0),
    CONSTRAINT check_invoice_tax_rate_valid CHECK (tax_rate >= 0 AND tax_rate <= 100),
    CONSTRAINT check_invoice_tax_amount_non_negative CHECK (tax_amount >= 0),
    CONSTRAINT check_invoice_total_positive CHECK (total_amount > 0),
    CONSTRAINT check_invoice_status_valid CHECK (status IN ('pending', 'paid', 'failed', 'cancelled')),
    CONSTRAINT check_invoice_dates_valid CHECK (due_date IS NULL OR due_date >= issue_date)
);

-- 2. Create indexes for performance and isolation
-- All indexes start with tenant_id for optimal performance
CREATE INDEX idx_invoices_tenant ON billing_invoices(tenant_id);
CREATE INDEX idx_invoices_tenant_status ON billing_invoices(tenant_id, status);
CREATE INDEX idx_invoices_tenant_subscription ON billing_invoices(tenant_id, subscription_id);
CREATE INDEX idx_invoices_tenant_transaction ON billing_invoices(tenant_id, transaction_id);
CREATE INDEX idx_invoices_tenant_issue_date ON billing_invoices(tenant_id, issue_date DESC);
CREATE INDEX idx_invoices_tenant_due_date ON billing_invoices(tenant_id, due_date);

-- UNIQUE constraint: invoice_number must be unique per tenant
CREATE UNIQUE INDEX idx_invoices_tenant_number_unique 
ON billing_invoices(tenant_id, invoice_number);

-- Index for finding unpaid invoices
CREATE INDEX idx_invoices_unpaid ON billing_invoices(tenant_id, status, due_date)
WHERE status IN ('pending', 'failed');

-- 3. Enable Row Level Security (RLS)
ALTER TABLE billing_invoices ENABLE ROW LEVEL SECURITY;

-- 4. Create RLS Policies for tenant isolation

-- Policy for SELECT: Users can only see their tenant's invoices
CREATE POLICY tenant_isolation_select_policy ON billing_invoices
    FOR SELECT
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy for INSERT: Users can only insert invoices for their tenant
CREATE POLICY tenant_isolation_insert_policy ON billing_invoices
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy for UPDATE: Users can only update their tenant's invoices
CREATE POLICY tenant_isolation_update_policy ON billing_invoices
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant')::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy for DELETE: Users can only delete their tenant's invoices
CREATE POLICY tenant_isolation_delete_policy ON billing_invoices
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- 5. Create trigger for updated_at
CREATE OR REPLACE FUNCTION update_billing_invoices_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_billing_invoices_updated_at
    BEFORE UPDATE ON billing_invoices
    FOR EACH ROW
    EXECUTE FUNCTION update_billing_invoices_updated_at();

-- 6. Create helper functions

-- Function to generate unique invoice number
CREATE OR REPLACE FUNCTION generate_invoice_number(p_tenant_id UUID)
RETURNS VARCHAR(50) AS $$
DECLARE
    v_year VARCHAR(4);
    v_month VARCHAR(2);
    v_sequence INTEGER;
    v_invoice_number VARCHAR(50);
BEGIN
    -- Get current year and month
    v_year := TO_CHAR(CURRENT_DATE, 'YYYY');
    v_month := TO_CHAR(CURRENT_DATE, 'MM');
    
    -- Get next sequence number for this tenant, year, and month
    SELECT COALESCE(MAX(CAST(SUBSTRING(invoice_number FROM 12) AS INTEGER)), 0) + 1
    INTO v_sequence
    FROM billing_invoices
    WHERE tenant_id = p_tenant_id
      AND invoice_number LIKE 'INV-' || v_year || v_month || '-%';
    
    -- Format: INV-YYYYMM-NNNN (e.g., INV-202602-0001)
    v_invoice_number := 'INV-' || v_year || v_month || '-' || LPAD(v_sequence::TEXT, 4, '0');
    
    RETURN v_invoice_number;
END;
$$ LANGUAGE plpgsql;

-- Function to calculate tax amount based on country
CREATE OR REPLACE FUNCTION calculate_invoice_tax(
    p_subtotal DECIMAL(10, 2),
    p_country_code VARCHAR(2) DEFAULT 'YE'
)
RETURNS TABLE (
    tax_rate DECIMAL(5, 2),
    tax_amount DECIMAL(10, 2),
    total_amount DECIMAL(10, 2)
) AS $$
DECLARE
    v_tax_rate DECIMAL(5, 2);
BEGIN
    -- Default tax rates by country (can be customized)
    v_tax_rate := CASE p_country_code
        WHEN 'YE' THEN 5.00  -- Yemen: 5% VAT
        WHEN 'SA' THEN 15.00 -- Saudi Arabia: 15% VAT
        WHEN 'AE' THEN 5.00  -- UAE: 5% VAT
        ELSE 0.00            -- Default: No tax
    END;
    
    RETURN QUERY
    SELECT 
        v_tax_rate,
        ROUND(p_subtotal * v_tax_rate / 100, 2),
        p_subtotal + ROUND(p_subtotal * v_tax_rate / 100, 2);
END;
$$ LANGUAGE plpgsql;

-- Function to create invoice for a payment
CREATE OR REPLACE FUNCTION create_invoice_for_payment(
    p_tenant_id UUID,
    p_subscription_id INTEGER DEFAULT NULL,
    p_transaction_id INTEGER DEFAULT NULL,
    p_subtotal DECIMAL(10, 2),
    p_description TEXT DEFAULT NULL,
    p_service_name VARCHAR(255) DEFAULT NULL,
    p_country_code VARCHAR(2) DEFAULT 'YE'
)
RETURNS INTEGER AS $$
DECLARE
    v_invoice_id INTEGER;
    v_invoice_number VARCHAR(50);
    v_tax_rate DECIMAL(5, 2);
    v_tax_amount DECIMAL(10, 2);
    v_total_amount DECIMAL(10, 2);
BEGIN
    -- Generate unique invoice number
    v_invoice_number := generate_invoice_number(p_tenant_id);
    
    -- Calculate tax
    SELECT * INTO v_tax_rate, v_tax_amount, v_total_amount
    FROM calculate_invoice_tax(p_subtotal, p_country_code);
    
    -- Insert invoice
    INSERT INTO billing_invoices (
        tenant_id,
        invoice_number,
        subscription_id,
        transaction_id,
        subtotal,
        tax_rate,
        tax_amount,
        total_amount,
        description,
        service_name,
        issue_date,
        status
    ) VALUES (
        p_tenant_id,
        v_invoice_number,
        p_subscription_id,
        p_transaction_id,
        p_subtotal,
        v_tax_rate,
        v_tax_amount,
        v_total_amount,
        p_description,
        p_service_name,
        CURRENT_DATE,
        'paid'  -- Assuming payment is already made
    )
    RETURNING id INTO v_invoice_id;
    
    RETURN v_invoice_id;
END;
$$ LANGUAGE plpgsql;

-- Function to mark invoice as paid
CREATE OR REPLACE FUNCTION mark_invoice_paid(p_invoice_id INTEGER)
RETURNS BOOLEAN AS $$
BEGIN
    UPDATE billing_invoices
    SET status = 'paid',
        paid_date = CURRENT_DATE,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = p_invoice_id
      AND status = 'pending';
    
    RETURN FOUND;
END;
$$ LANGUAGE plpgsql;

-- Function to get invoices for a tenant with filtering
CREATE OR REPLACE FUNCTION get_tenant_invoices(
    p_tenant_id UUID,
    p_status VARCHAR(20) DEFAULT NULL,
    p_from_date DATE DEFAULT NULL,
    p_to_date DATE DEFAULT NULL,
    p_limit INTEGER DEFAULT 50,
    p_offset INTEGER DEFAULT 0
)
RETURNS TABLE (
    id INTEGER,
    invoice_number VARCHAR(50),
    issue_date DATE,
    total_amount DECIMAL(10, 2),
    status VARCHAR(20),
    service_name VARCHAR(255)
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        bi.id,
        bi.invoice_number,
        bi.issue_date,
        bi.total_amount,
        bi.status,
        bi.service_name
    FROM billing_invoices bi
    WHERE bi.tenant_id = p_tenant_id
      AND (p_status IS NULL OR bi.status = p_status)
      AND (p_from_date IS NULL OR bi.issue_date >= p_from_date)
      AND (p_to_date IS NULL OR bi.issue_date <= p_to_date)
    ORDER BY bi.issue_date DESC, bi.id DESC
    LIMIT p_limit
    OFFSET p_offset;
END;
$$ LANGUAGE plpgsql;

-- 7. Comments for documentation
COMMENT ON TABLE billing_invoices IS 'Stores invoices generated automatically for all payment operations with tenant isolation';
COMMENT ON COLUMN billing_invoices.tenant_id IS 'Tenant ID for multi-tenant isolation - CRITICAL for RLS';
COMMENT ON COLUMN billing_invoices.invoice_number IS 'Unique invoice number in format INV-YYYYMM-NNNN';
COMMENT ON COLUMN billing_invoices.subscription_id IS 'References the subscription if invoice is for subscription payment';
COMMENT ON COLUMN billing_invoices.transaction_id IS 'References the accounting transaction';
COMMENT ON COLUMN billing_invoices.subtotal IS 'Amount before tax';
COMMENT ON COLUMN billing_invoices.tax_rate IS 'Tax rate percentage (e.g., 5.00 for 5%)';
COMMENT ON COLUMN billing_invoices.tax_amount IS 'Calculated tax amount';
COMMENT ON COLUMN billing_invoices.total_amount IS 'Total amount including tax (subtotal + tax_amount)';
COMMENT ON COLUMN billing_invoices.issue_date IS 'Date when the invoice was issued';
COMMENT ON COLUMN billing_invoices.due_date IS 'Payment due date (NULL for already paid invoices)';
COMMENT ON COLUMN billing_invoices.paid_date IS 'Date when the invoice was paid';
COMMENT ON COLUMN billing_invoices.status IS 'Invoice status: pending, paid, failed, cancelled';
COMMENT ON COLUMN billing_invoices.pdf_generated IS 'Whether PDF has been generated for this invoice';
COMMENT ON COLUMN billing_invoices.pdf_path IS 'Path to the generated PDF file';
COMMENT ON COLUMN billing_invoices.email_sent IS 'Whether invoice email has been sent';
COMMENT ON COLUMN billing_invoices.email_sent_at IS 'Timestamp when invoice email was sent';

COMMIT;

-- Usage Instructions:
-- 
-- 1. Create invoice for a payment:
-- SELECT create_invoice_for_payment(
--     '<tenant-uuid>'::uuid,
--     <subscription-id>,
--     <transaction-id>,
--     50.00,  -- subtotal
--     'Monthly subscription payment',
--     'Mobile Gateway Service',
--     'YE'  -- country code for tax calculation
-- );
--
-- 2. Get invoices for a tenant:
-- SELECT * FROM get_tenant_invoices(
--     '<tenant-uuid>'::uuid,
--     'paid',  -- status filter (NULL for all)
--     '2026-01-01'::date,  -- from_date
--     '2026-12-31'::date,  -- to_date
--     50,  -- limit
--     0    -- offset
-- );
--
-- 3. Mark invoice as paid:
-- SELECT mark_invoice_paid(<invoice-id>);
--
-- 4. Generate invoice number:
-- SELECT generate_invoice_number('<tenant-uuid>'::uuid);

-- Rollback script (if needed):
-- BEGIN;
-- DROP TRIGGER IF EXISTS trigger_billing_invoices_updated_at ON billing_invoices;
-- DROP FUNCTION IF EXISTS update_billing_invoices_updated_at();
-- DROP FUNCTION IF EXISTS generate_invoice_number(UUID);
-- DROP FUNCTION IF EXISTS calculate_invoice_tax(DECIMAL, VARCHAR);
-- DROP FUNCTION IF EXISTS create_invoice_for_payment(UUID, INTEGER, INTEGER, DECIMAL, TEXT, VARCHAR, VARCHAR);
-- DROP FUNCTION IF EXISTS mark_invoice_paid(INTEGER);
-- DROP FUNCTION IF EXISTS get_tenant_invoices(UUID, VARCHAR, DATE, DATE, INTEGER, INTEGER);
-- DROP TABLE IF EXISTS billing_invoices CASCADE;
-- COMMIT;
