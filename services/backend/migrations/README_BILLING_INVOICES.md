# Billing Invoices Migration Guide

## Overview

This migration creates the `billing_invoices` table for automatic invoice generation with complete tenant isolation. The system automatically generates invoices for all payment operations including subscription payments, top-ups, and other transactions.

## Requirements Implemented

This migration implements the following requirements from the tariff pricing system spec:

- **Requirement 40.1**: THE النظام SHALL يولّد فاتورة تلقائياً لكل عملية دفع
- **Requirement 40.2**: THE النظام SHALL يحفظ رقم فاتورة فريد لكل فاتورة
- **Requirement 40.3**: THE النظام SHALL يحفظ تاريخ إصدار الفاتورة

## Table Structure

### billing_invoices

| Column | Type | Description |
|--------|------|-------------|
| id | SERIAL | Primary key |
| tenant_id | UUID | Tenant ID for multi-tenant isolation (CRITICAL) |
| invoice_number | VARCHAR(50) | Unique invoice number (format: INV-YYYYMM-NNNN) |
| subscription_id | INTEGER | References subscriptions(id) - NULL for non-subscription payments |
| transaction_id | INTEGER | References accounting_transactions(id) |
| subtotal | DECIMAL(10,2) | Amount before tax |
| tax_rate | DECIMAL(5,2) | Tax rate percentage (e.g., 5.00 for 5%) |
| tax_amount | DECIMAL(10,2) | Calculated tax amount |
| total_amount | DECIMAL(10,2) | Total amount including tax |
| currency | VARCHAR(3) | Currency code (default: USD) |
| description | TEXT | Invoice description |
| service_name | VARCHAR(255) | Name of the service |
| issue_date | DATE | Date when invoice was issued |
| due_date | DATE | Payment due date (NULL for already paid) |
| paid_date | DATE | Date when invoice was paid |
| status | VARCHAR(20) | Invoice status: pending, paid, failed, cancelled |
| pdf_generated | BOOLEAN | Whether PDF has been generated |
| pdf_path | VARCHAR(500) | Path to the generated PDF file |
| email_sent | BOOLEAN | Whether invoice email has been sent |
| email_sent_at | TIMESTAMP | Timestamp when invoice email was sent |
| created_at | TIMESTAMP | Creation timestamp |
| updated_at | TIMESTAMP | Last update timestamp |

## Indexes

All indexes are optimized for tenant isolation and start with `tenant_id`:

1. **idx_invoices_tenant**: Basic tenant lookup
2. **idx_invoices_tenant_status**: Filter by status per tenant
3. **idx_invoices_tenant_subscription**: Find invoices for a subscription
4. **idx_invoices_tenant_transaction**: Find invoice for a transaction
5. **idx_invoices_tenant_issue_date**: Sort by issue date (DESC)
6. **idx_invoices_tenant_due_date**: Find invoices by due date
7. **idx_invoices_tenant_number_unique**: Ensure unique invoice numbers per tenant
8. **idx_invoices_unpaid**: Find unpaid invoices efficiently

## Row Level Security (RLS)

The table has RLS enabled with policies for complete tenant isolation:

- **SELECT Policy**: Users can only see their tenant's invoices
- **INSERT Policy**: Users can only create invoices for their tenant
- **UPDATE Policy**: Users can only update their tenant's invoices
- **DELETE Policy**: Users can only delete their tenant's invoices

## Helper Functions

### 1. generate_invoice_number(p_tenant_id UUID)

Generates a unique invoice number in the format `INV-YYYYMM-NNNN`.

**Example:**
```sql
SELECT generate_invoice_number('123e4567-e89b-12d3-a456-426614174000'::uuid);
-- Returns: INV-202602-0001
```

### 2. calculate_invoice_tax(p_subtotal DECIMAL, p_country_code VARCHAR)

Calculates tax based on country code.

**Tax Rates:**
- Yemen (YE): 5%
- Saudi Arabia (SA): 15%
- UAE (AE): 5%
- Default: 0%

**Example:**
```sql
SELECT * FROM calculate_invoice_tax(100.00, 'YE');
-- Returns: tax_rate=5.00, tax_amount=5.00, total_amount=105.00
```

### 3. create_invoice_for_payment(...)

Creates an invoice for a payment operation.

**Parameters:**
- `p_tenant_id`: Tenant UUID
- `p_subscription_id`: Subscription ID (optional)
- `p_transaction_id`: Transaction ID (optional)
- `p_subtotal`: Amount before tax
- `p_description`: Invoice description (optional)
- `p_service_name`: Service name (optional)
- `p_country_code`: Country code for tax calculation (default: 'YE')

**Example:**
```sql
SELECT create_invoice_for_payment(
    '123e4567-e89b-12d3-a456-426614174000'::uuid,
    1,  -- subscription_id
    100,  -- transaction_id
    50.00,  -- subtotal
    'Monthly subscription payment for Standard package',
    'Mobile Gateway Service',
    'YE'
);
-- Returns: invoice_id
```

### 4. mark_invoice_paid(p_invoice_id INTEGER)

Marks an invoice as paid.

**Example:**
```sql
SELECT mark_invoice_paid(1);
-- Returns: true if successful
```

### 5. get_tenant_invoices(...)

Retrieves invoices for a tenant with filtering and pagination.

**Parameters:**
- `p_tenant_id`: Tenant UUID
- `p_status`: Status filter (optional, NULL for all)
- `p_from_date`: Start date filter (optional)
- `p_to_date`: End date filter (optional)
- `p_limit`: Number of records to return (default: 50)
- `p_offset`: Offset for pagination (default: 0)

**Example:**
```sql
-- Get all paid invoices for a tenant
SELECT * FROM get_tenant_invoices(
    '123e4567-e89b-12d3-a456-426614174000'::uuid,
    'paid',
    NULL,
    NULL,
    50,
    0
);

-- Get invoices for a specific date range
SELECT * FROM get_tenant_invoices(
    '123e4567-e89b-12d3-a456-426614174000'::uuid,
    NULL,
    '2026-01-01'::date,
    '2026-12-31'::date,
    100,
    0
);
```

## Running the Migration

### Using PowerShell Script

```powershell
.\run-billing-invoices-migration.ps1
```

### Manual Execution

```bash
psql -h localhost -U postgres -d sms_gateway -f migrations/019_billing_invoices_table.sql
```

## Testing the Migration

A test script is provided to verify the migration:

```powershell
.\test_billing_invoices_migration.sql
```

The test script will:
1. Create a test tenant
2. Generate invoice numbers
3. Calculate tax for different countries
4. Create sample invoices
5. Test filtering and pagination
6. Verify RLS policies
7. Clean up test data

## Invoice Number Format

Invoice numbers follow the format: `INV-YYYYMM-NNNN`

- **INV**: Prefix
- **YYYY**: Year (4 digits)
- **MM**: Month (2 digits)
- **NNNN**: Sequential number (4 digits, padded with zeros)

**Examples:**
- `INV-202602-0001`: First invoice in February 2026
- `INV-202602-0042`: 42nd invoice in February 2026
- `INV-202603-0001`: First invoice in March 2026

## Integration with Other Systems

### Subscription Manager

When a subscription is created or renewed, automatically create an invoice:

```go
// After successful subscription payment
invoiceID, err := db.CreateInvoiceForPayment(
    ctx,
    tenantID,
    subscriptionID,
    transactionID,
    packagePrice,
    fmt.Sprintf("Monthly subscription payment for %s package", packageName),
    "Mobile Gateway Service",
    "YE",
)
```

### Balance Manager

When balance is topped up, create an invoice:

```go
// After successful top-up
invoiceID, err := db.CreateInvoiceForPayment(
    ctx,
    tenantID,
    nil,  // no subscription
    transactionID,
    topUpAmount,
    "Balance top-up",
    "Balance Service",
    countryCode,
)
```

## Status Flow

```
pending → paid (successful payment)
pending → failed (payment failed)
pending → cancelled (cancelled by user/admin)
```

## PDF Generation

The system tracks PDF generation status:

1. Invoice is created with `pdf_generated = false`
2. Background job generates PDF and updates:
   - `pdf_generated = true`
   - `pdf_path = '/path/to/invoice.pdf'`
3. Email service sends invoice with PDF attachment
4. Updates `email_sent = true` and `email_sent_at`

## Email Delivery

Invoice emails are sent automatically:

1. After invoice creation
2. After PDF generation
3. Tracks delivery status in `email_sent` and `email_sent_at`

## Rollback

If you need to rollback this migration:

```sql
BEGIN;
DROP TRIGGER IF EXISTS trigger_billing_invoices_updated_at ON billing_invoices;
DROP FUNCTION IF EXISTS update_billing_invoices_updated_at();
DROP FUNCTION IF EXISTS generate_invoice_number(UUID);
DROP FUNCTION IF EXISTS calculate_invoice_tax(DECIMAL, VARCHAR);
DROP FUNCTION IF EXISTS create_invoice_for_payment(UUID, INTEGER, INTEGER, DECIMAL, TEXT, VARCHAR, VARCHAR);
DROP FUNCTION IF EXISTS mark_invoice_paid(INTEGER);
DROP FUNCTION IF EXISTS get_tenant_invoices(UUID, VARCHAR, DATE, DATE, INTEGER, INTEGER);
DROP TABLE IF EXISTS billing_invoices CASCADE;
COMMIT;
```

## Security Considerations

1. **Tenant Isolation**: All queries must set `app.current_tenant` before accessing data
2. **RLS Enforcement**: Row Level Security prevents cross-tenant data access
3. **Unique Invoice Numbers**: Per-tenant uniqueness ensures no conflicts
4. **Audit Trail**: All changes are tracked with timestamps
5. **Immutable Records**: Invoices should not be deleted, only cancelled

## Performance Considerations

1. All indexes start with `tenant_id` for optimal query performance
2. Composite indexes support common query patterns
3. Partial indexes for unpaid invoices reduce index size
4. Pagination support prevents large result sets

## Monitoring

Monitor the following metrics:

1. Invoice generation rate
2. Failed invoice creations
3. Unpaid invoices count
4. PDF generation success rate
5. Email delivery success rate

## Next Steps

After running this migration:

1. Implement invoice generation in subscription manager
2. Implement invoice generation in balance manager
3. Create PDF generation service
4. Create email delivery service
5. Add invoice viewing API endpoints
6. Add invoice download functionality to web interfaces

## Related Migrations

- `016_packages_table.sql`: Package definitions
- `017_subscriptions_table.sql`: Subscription tracking
- `003_accounting_transactions_table.sql`: Transaction records

## Support

For issues or questions about this migration, refer to:
- Design document: `.kiro/specs/tariff-pricing-system/design.md`
- Requirements document: `.kiro/specs/tariff-pricing-system/requirements.md`
- Task list: `.kiro/specs/tariff-pricing-system/tasks.md`
