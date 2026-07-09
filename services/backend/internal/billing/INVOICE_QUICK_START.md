# Invoice Generator - Quick Start Guide

## 🚀 Quick Start

Get started with invoice generation in 5 minutes!

## Prerequisites

- Go 1.21 or higher
- PostgreSQL database with billing_invoices table
- Migration 019 applied

## Installation

```bash
# Install required packages
go get github.com/jung-kurt/gofpdf
go get github.com/shopspring/decimal
go get github.com/google/uuid
```

## Basic Usage

### 1. Generate Your First Invoice

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "log"

    "github.com/google/uuid"
    "github.com/shopspring/decimal"
    _ "github.com/lib/pq"
    "go.uber.org/zap"

    "sms-gateway/internal/billing"
)

func main() {
    // Connect to database
    db, err := sql.Open("postgres", "postgres://user:pass@localhost/sms_gateway?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Create logger
    logger, _ := zap.NewProduction()
    defer logger.Sync()

    // Create invoice generator
    generator := billing.NewInvoiceGenerator(db, logger)

    // Generate invoice
    params := billing.GenerateInvoiceParams{
        TenantID:    uuid.MustParse("your-tenant-uuid"),
        Subtotal:    decimal.NewFromFloat(50.00),
        Description: "Monthly subscription - Standard Package",
        ServiceName: "Mobile Gateway Service",
        CountryCode: "YE", // Yemen - 5% VAT
        Currency:    "USD",
    }

    invoice, err := generator.GenerateInvoice(context.Background(), params)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("✅ Invoice generated successfully!\n")
    fmt.Printf("   Invoice Number: %s\n", invoice.InvoiceNumber)
    fmt.Printf("   Subtotal: %s %s\n", invoice.Subtotal.StringFixed(2), invoice.Currency)
    fmt.Printf("   Tax (%s%%): %s %s\n", invoice.TaxRate.StringFixed(2), invoice.TaxAmount.StringFixed(2), invoice.Currency)
    fmt.Printf("   Total: %s %s\n", invoice.TotalAmount.StringFixed(2), invoice.Currency)
}
```

### 2. Generate PDF

```go
// Create PDF generator
pdfGenerator := billing.NewPDFGenerator(logger, "/var/invoices")

// Generate PDF
pdfPath, err := pdfGenerator.GeneratePDF(context.Background(), invoice)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("📄 PDF generated: %s\n", pdfPath)
```

### 3. Send Invoice via Email

```go
// Create invoice service (combines generation, PDF, and email)
service := billing.NewInvoiceService(db, logger, "/var/invoices")

// Generate and send invoice
invoice, err := service.GenerateAndSendInvoice(
    context.Background(),
    params,
    "customer@example.com",
    "John Doe",
)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("📧 Invoice sent to customer!\n")
```

## Common Scenarios

### Scenario 1: Subscription Payment Invoice

```go
subscriptionID := int64(123)

params := billing.GenerateInvoiceParams{
    TenantID:       tenantID,
    SubscriptionID: &subscriptionID,
    Subtotal:       decimal.NewFromFloat(20.00),
    Description:    "Monthly subscription renewal - Standard Package",
    ServiceName:    "Mobile Gateway Service",
    CountryCode:    "YE",
    Currency:       "USD",
}

invoice, err := generator.GenerateInvoice(ctx, params)
```

### Scenario 2: Balance Top-up Invoice

```go
transactionID := int64(456)

params := billing.GenerateInvoiceParams{
    TenantID:      tenantID,
    TransactionID: &transactionID,
    Subtotal:      decimal.NewFromFloat(100.00),
    Description:   "Balance top-up",
    ServiceName:   "Top-up Service",
    CountryCode:   "YE",
    Currency:      "USD",
}

invoice, err := generator.GenerateInvoice(ctx, params)
```

### Scenario 3: Different Country Tax Rates

```go
// Saudi Arabia - 15% VAT
params := billing.GenerateInvoiceParams{
    TenantID:    tenantID,
    Subtotal:    decimal.NewFromFloat(100.00),
    Description: "Service payment",
    ServiceName: "SMS Service",
    CountryCode: "SA", // Saudi Arabia
    Currency:    "SAR",
}

invoice, err := generator.GenerateInvoice(ctx, params)
// Tax will be calculated at 15%
```

## API Usage

### List Invoices

```bash
# Get all invoices
curl -X GET "http://localhost:8080/api/invoices" \
  -H "Authorization: Bearer YOUR_TOKEN"

# Filter by status
curl -X GET "http://localhost:8080/api/invoices?status=paid" \
  -H "Authorization: Bearer YOUR_TOKEN"

# Filter by date range
curl -X GET "http://localhost:8080/api/invoices?from_date=2026-01-01&to_date=2026-12-31" \
  -H "Authorization: Bearer YOUR_TOKEN"

# With pagination
curl -X GET "http://localhost:8080/api/invoices?limit=20&offset=0" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Get Invoice Details

```bash
curl -X GET "http://localhost:8080/api/invoices/123" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Download Invoice PDF

```bash
curl -X GET "http://localhost:8080/api/invoices/123/pdf" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  --output invoice.pdf
```

## Testing

### Run Unit Tests

```bash
# Run all tests
go test ./internal/billing/...

# Run with verbose output
go test -v ./internal/billing/...

# Run specific test
go test -v ./internal/billing/... -run TestGenerateInvoice
```

### Run Integration Tests

```bash
# Requires database connection
go test -v ./internal/billing/... -short=false
```

## Database Helper Functions

### Generate Invoice Number

```sql
-- Generate unique invoice number for a tenant
SELECT generate_invoice_number('tenant-uuid'::uuid);
-- Returns: INV-202602-0001
```

### Calculate Tax

```sql
-- Calculate tax for Yemen (5%)
SELECT * FROM calculate_invoice_tax(100.00, 'YE');
-- Returns: tax_rate=5.00, tax_amount=5.00, total_amount=105.00

-- Calculate tax for Saudi Arabia (15%)
SELECT * FROM calculate_invoice_tax(100.00, 'SA');
-- Returns: tax_rate=15.00, tax_amount=15.00, total_amount=115.00
```

### Create Invoice

```sql
-- Create invoice using database function
SELECT create_invoice_for_payment(
    'tenant-uuid'::uuid,
    123,  -- subscription_id
    456,  -- transaction_id
    50.00,  -- subtotal
    'Monthly subscription payment',
    'Mobile Gateway Service',
    'YE'  -- country_code
);
-- Returns: invoice_id
```

### List Invoices

```sql
-- Get invoices for a tenant
SELECT * FROM get_tenant_invoices(
    'tenant-uuid'::uuid,
    'paid',  -- status filter (NULL for all)
    '2026-01-01'::date,  -- from_date
    '2026-12-31'::date,  -- to_date
    50,  -- limit
    0    -- offset
);
```

## Troubleshooting

### Issue: "tenant_id not found in context"

**Solution:** Ensure the auth middleware sets `tenant_id` in the context:

```go
c.Set("tenant_id", tenantID.String())
```

### Issue: "failed to set tenant context"

**Solution:** Check database connection and RLS configuration:

```sql
-- Verify RLS is enabled
SELECT tablename, rowsecurity 
FROM pg_tables 
WHERE tablename = 'billing_invoices';

-- Check RLS policies
\d+ billing_invoices
```

### Issue: "invoice not found"

**Solution:** Verify tenant isolation - you can only access invoices for your tenant:

```go
// Correct
invoice, err := generator.GetInvoiceByID(ctx, yourTenantID, invoiceID)

// Wrong - will fail if invoice belongs to different tenant
invoice, err := generator.GetInvoiceByID(ctx, wrongTenantID, invoiceID)
```

### Issue: PDF generation fails

**Solution:** Ensure output directory exists and has write permissions:

```bash
mkdir -p /var/invoices
chmod 755 /var/invoices
```

## Configuration

### Environment Variables

```env
# Database
DATABASE_URL=postgres://user:pass@localhost/sms_gateway?sslmode=disable

# PDF Output
INVOICE_PDF_DIR=/var/invoices

# Email (for future implementation)
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=noreply@smsgateway.com
SMTP_PASSWORD=your-password
```

### Tax Rates by Country

Current tax rates (can be customized in `calculate_invoice_tax` function):

| Country | Code | Tax Rate |
|---------|------|----------|
| Yemen | YE | 5% |
| Saudi Arabia | SA | 15% |
| UAE | AE | 5% |
| Default | * | 0% |

## Next Steps

1. ✅ Generate your first invoice
2. ✅ Test PDF generation
3. ✅ Integrate with your payment flow
4. ✅ Set up email delivery
5. ✅ Configure tax rates for your countries
6. ✅ Test API endpoints
7. ✅ Deploy to production

## Resources

- [Full README](./README.md) - Complete documentation
- [Migration README](../../migrations/README_BILLING_INVOICES.md) - Database schema
- [API Documentation](../api/README.md) - API reference
- [Design Document](../../../.kiro/specs/tariff-pricing-system/design.md) - Architecture

## Support

Need help? Check:
1. This Quick Start guide
2. Full README documentation
3. Migration README for database issues
4. Design document for architecture questions

Happy invoicing! 🎉
