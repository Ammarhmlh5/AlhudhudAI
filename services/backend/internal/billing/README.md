# Billing Module - Invoice Generator

## Overview

The Billing module provides comprehensive invoice generation, PDF creation, and email delivery functionality for the SMS Gateway platform. It automatically generates invoices for all payment operations including subscription payments, top-ups, and other transactions.

## Features

### 1. Automatic Invoice Generation (Requirement 40.1)
- Generates invoices automatically for every payment operation
- Supports subscription payments, top-ups, and custom transactions
- Atomic operations with full transaction support

### 2. Unique Invoice Numbers (Requirement 40.2)
- Format: `INV-YYYYMM-NNNN` (e.g., `INV-202602-0001`)
- Sequential numbering per tenant per month
- Guaranteed uniqueness through database constraints

### 3. Invoice Details (Requirements 40.3, 40.4)
- Issue date, due date, paid date
- Service name and description
- Subscription and transaction references
- Complete audit trail

### 4. Automatic Tax Calculation (Requirement 40.5)
- Country-specific tax rates:
  - Yemen (YE): 5% VAT
  - Saudi Arabia (SA): 15% VAT
  - UAE (AE): 5% VAT
  - Default: 0% (no tax)
- Automatic calculation of tax amount and total

### 5. PDF Generation (Requirement 40.6)
- Professional invoice template
- Company branding (logo and colors)
- Detailed line items and totals
- Automatic generation on demand

### 6. Email Delivery (Requirement 40.7)
- Send invoices via email with PDF attachment
- Customizable email templates
- Delivery tracking and status

### 7. Invoice Listing API (Requirement 40.8)
- List all invoices with tenant isolation
- Filter by status, date range
- Pagination support
- Download PDF functionality

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   Invoice Service                        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │   Invoice    │  │     PDF      │  │    Email     │  │
│  │  Generator   │  │  Generator   │  │   Sender     │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│              Database (billing_invoices)                 │
│  - Tenant isolation with RLS                             │
│  - Unique invoice numbers                                │
│  - Automatic tax calculation                             │
│  - PDF and email tracking                                │
└─────────────────────────────────────────────────────────┘
```

## Components

### InvoiceGenerator
Handles invoice creation and retrieval with tenant isolation.

**Methods:**
- `GenerateInvoice(ctx, params)` - Generate new invoice
- `GetInvoiceByID(ctx, tenantID, invoiceID)` - Retrieve by ID
- `GetInvoiceByNumber(ctx, tenantID, invoiceNumber)` - Retrieve by number
- `MarkPDFGenerated(ctx, tenantID, invoiceID, pdfPath)` - Mark PDF as generated
- `MarkEmailSent(ctx, tenantID, invoiceID)` - Mark email as sent

### PDFGenerator
Generates professional PDF invoices.

**Methods:**
- `GeneratePDF(ctx, invoice)` - Generate PDF file

**Features:**
- Company header with logo and branding
- Invoice details and line items
- Tax calculation and totals
- Professional formatting

### InvoiceService
Combines invoice generation, PDF creation, and email sending.

**Methods:**
- `SendInvoice(ctx, params)` - Generate PDF and send via email
- `GenerateAndSendInvoice(ctx, generateParams, email, name)` - All-in-one method

## Usage Examples

### 1. Generate Invoice

```go
import (
    "context"
    "github.com/google/uuid"
    "github.com/shopspring/decimal"
    "sms-gateway/internal/billing"
)

// Create generator
generator := billing.NewInvoiceGenerator(db, logger)

// Generate invoice
params := billing.GenerateInvoiceParams{
    TenantID:    uuid.MustParse("tenant-uuid"),
    Subtotal:    decimal.NewFromFloat(50.00),
    Description: "Monthly subscription payment",
    ServiceName: "Mobile Gateway Service",
    CountryCode: "YE",
    Currency:    "USD",
}

invoice, err := generator.GenerateInvoice(context.Background(), params)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Invoice generated: %s\n", invoice.InvoiceNumber)
```

### 2. Generate and Send Invoice

```go
// Create service
service := billing.NewInvoiceService(db, logger, "/var/invoices")

// Generate and send
invoice, err := service.GenerateAndSendInvoice(
    context.Background(),
    params,
    "customer@example.com",
    "Customer Name",
)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Invoice sent: %s\n", invoice.InvoiceNumber)
```

### 3. List Invoices via API

```bash
# List all invoices
GET /api/invoices?limit=50&offset=0

# Filter by status
GET /api/invoices?status=paid

# Filter by date range
GET /api/invoices?from_date=2026-01-01&to_date=2026-12-31

# Combined filters
GET /api/invoices?status=paid&from_date=2026-02-01&limit=20
```

### 4. Get Invoice Details

```bash
# Get invoice by ID
GET /api/invoices/123

# Download PDF
GET /api/invoices/123/pdf
```

## Database Schema

The `billing_invoices` table stores all invoice data:

```sql
CREATE TABLE billing_invoices (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,
    invoice_number VARCHAR(50) NOT NULL,
    subscription_id INTEGER,
    transaction_id INTEGER,
    subtotal DECIMAL(10, 2) NOT NULL,
    tax_rate DECIMAL(5, 2) DEFAULT 0.00,
    tax_amount DECIMAL(10, 2) DEFAULT 0.00,
    total_amount DECIMAL(10, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    description TEXT,
    service_name VARCHAR(255),
    issue_date DATE NOT NULL DEFAULT CURRENT_DATE,
    due_date DATE,
    paid_date DATE,
    status VARCHAR(20) DEFAULT 'pending',
    pdf_generated BOOLEAN DEFAULT false,
    pdf_path VARCHAR(500),
    email_sent BOOLEAN DEFAULT false,
    email_sent_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Helper Functions

The migration includes several helper functions:

### generate_invoice_number(tenant_id)
Generates unique invoice numbers in format `INV-YYYYMM-NNNN`.

### calculate_invoice_tax(subtotal, country_code)
Calculates tax based on country-specific rates.

### create_invoice_for_payment(...)
Creates a complete invoice with all calculations.

### get_tenant_invoices(...)
Retrieves invoices with filtering and pagination.

## Security

### Tenant Isolation
- Row Level Security (RLS) enabled on all tables
- All queries filtered by `tenant_id`
- Cross-tenant access prevented at database level

### Data Protection
- Invoice numbers unique per tenant
- PDF files stored securely
- Email delivery tracked

## Testing

Run tests with:

```bash
# Unit tests
go test ./internal/billing/...

# Integration tests
go test -v ./internal/billing/... -short=false

# API tests
go test ./internal/api/... -run TestInvoice
```

## Configuration

### PDF Output Directory
Set the PDF output directory when creating the service:

```go
service := billing.NewInvoiceService(db, logger, "/var/invoices")
```

### Email Configuration
Configure email settings in your environment:

```env
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=noreply@smsgateway.com
SMTP_PASSWORD=your-password
```

## Error Handling

All methods return descriptive errors:

```go
invoice, err := generator.GenerateInvoice(ctx, params)
if err != nil {
    switch {
    case errors.Is(err, ErrInsufficientBalance):
        // Handle insufficient balance
    case errors.Is(err, ErrInvalidTenant):
        // Handle invalid tenant
    default:
        // Handle other errors
    }
}
```

## Performance Considerations

1. **Database Indexes**: All queries use indexes starting with `tenant_id`
2. **PDF Generation**: PDFs are generated once and cached
3. **Email Delivery**: Asynchronous delivery recommended for production
4. **Pagination**: Use limit/offset for large result sets

## Future Enhancements

- [ ] Bulk invoice generation
- [ ] Invoice templates per tenant
- [ ] Multi-language support
- [ ] Invoice reminders for unpaid invoices
- [ ] Payment gateway integration
- [ ] Invoice disputes and adjustments

## Related Documentation

- [Migration README](../../migrations/README_BILLING_INVOICES.md)
- [Quick Start Guide](./QUICK_START.md)
- [API Documentation](../api/README.md)
- [Design Document](../../../.kiro/specs/tariff-pricing-system/design.md)

## Support

For questions or issues:
1. Check the Quick Start guide
2. Review the migration README
3. Consult the design document
4. Contact the development team
