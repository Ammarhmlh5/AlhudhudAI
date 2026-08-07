package billing

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestGenerateInvoice tests invoice generation
func TestGenerateInvoice(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := zap.NewNop()
	generator := NewInvoiceGenerator(db, logger)

	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("Generate invoice with all details", func(t *testing.T) {
		params := GenerateInvoiceParams{
			TenantID:    tenantID,
			Subtotal:    decimal.NewFromFloat(50.00),
			Description: "Monthly subscription payment",
			ServiceName: "Mobile Gateway Service",
			CountryCode: "YE",
			Currency:    "USD",
		}

		// Expect transaction 1 for CREATE
		mock.ExpectBegin()
		mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
			WithArgs(tenantID.String()).
			WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectExec("SELECT set_config\\('app.is_super_admin', \\$1, true\\)").
				WithArgs("false").
				WillReturnResult(sqlmock.NewResult(0, 0))

		mock.ExpectQuery("SELECT create_invoice_for_payment").
			WithArgs(
				tenantID,
				sqlmock.AnyArg(), // subscription_id
				sqlmock.AnyArg(), // transaction_id
				params.Subtotal,
				params.Description,
				params.ServiceName,
				params.CountryCode,
			).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
		
		mock.ExpectCommit()

		// Expect transaction 2 for GET
		mock.ExpectBegin()
		mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
			WithArgs(tenantID.String()).
			WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectExec("SELECT set_config\\('app.is_super_admin', \\$1, true\\)").
				WithArgs("false").
				WillReturnResult(sqlmock.NewResult(0, 0))

		rows := sqlmock.NewRows([]string{
			"id", "tenant_id", "invoice_number", "subscription_id", "transaction_id",
			"subtotal", "tax_rate", "tax_amount", "total_amount", "currency",
			"description", "service_name", "issue_date", "due_date", "paid_date",
			"status", "pdf_generated", "pdf_path", "email_sent", "email_sent_at",
			"created_at", "updated_at",
		}).AddRow(
			int64(1), tenantID, "INV-202401-001", nil, nil,
			50.00, 5.00, 2.50, 52.50, "USD",
			"Monthly subscription payment", "Mobile Gateway Service", time.Now(), nil, nil,
			"paid", false, "", false, nil,
			time.Now(), time.Now(),
		)

		mock.ExpectQuery("SELECT (.+) FROM billing_invoices WHERE id = \\$1 AND tenant_id = \\$2").
			WithArgs(int64(1), tenantID).
			WillReturnRows(rows)
		
		mock.ExpectCommit()

		invoice, err := generator.GenerateInvoice(ctx, params)
		require.NoError(t, err)
		assert.NotNil(t, invoice)
		assert.Equal(t, int64(1), invoice.ID)

		// Verify invoice details
		assert.Equal(t, tenantID, invoice.TenantID)
		assert.NotEmpty(t, invoice.InvoiceNumber)
		assert.True(t, invoice.InvoiceNumber[:4] == "INV-")
		assert.Equal(t, params.Subtotal, invoice.Subtotal)
		assert.Equal(t, params.Description, invoice.Description)
		assert.Equal(t, params.ServiceName, invoice.ServiceName)
		assert.Equal(t, params.Currency, invoice.Currency)
		assert.Equal(t, "paid", invoice.Status)

		// Verify tax calculation (5% for Yemen)
		expectedTaxRate := decimal.NewFromFloat(5.00)
		expectedTaxAmount := params.Subtotal.Mul(expectedTaxRate).Div(decimal.NewFromInt(100))
		expectedTotal := params.Subtotal.Add(expectedTaxAmount)

		assert.Equal(t, expectedTaxRate, invoice.TaxRate)
		assert.True(t, invoice.TaxAmount.Equal(expectedTaxAmount))
		assert.True(t, invoice.TotalAmount.Equal(expectedTotal))

		// Verify dates
		assert.Equal(t, time.Now().Format("2006-01-02"), invoice.IssueDate.Format("2006-01-02"))
		assert.False(t, invoice.PDFGenerated)
		assert.False(t, invoice.EmailSent)
	})

	t.Run("Generate invoice with subscription ID", func(t *testing.T) {
		subscriptionID := int64(123)
		params := GenerateInvoiceParams{
			TenantID:       tenantID,
			SubscriptionID: &subscriptionID,
			Subtotal:       decimal.NewFromFloat(20.00),
			Description:    "Standard package subscription",
			ServiceName:    "Mobile Gateway Service",
			CountryCode:    "YE",
			Currency:       "USD",
		}

		// Expect transaction 1 for CREATE
		mock.ExpectBegin()
		mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
			WithArgs(tenantID.String()).
			WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectExec("SELECT set_config\\('app.is_super_admin', \\$1, true\\)").
				WithArgs("false").
				WillReturnResult(sqlmock.NewResult(0, 0))

		mock.ExpectQuery("SELECT create_invoice_for_payment").
			WithArgs(tenantID, subscriptionID, sqlmock.AnyArg(), params.Subtotal, params.Description, params.ServiceName, params.CountryCode).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(2)))
		
		mock.ExpectCommit()

		// Expect transaction 2 for GET
		mock.ExpectBegin()
		mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
			WithArgs(tenantID.String()).
			WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectExec("SELECT set_config\\('app.is_super_admin', \\$1, true\\)").
				WithArgs("false").
				WillReturnResult(sqlmock.NewResult(0, 0))

		rows := sqlmock.NewRows([]string{
			"id", "tenant_id", "invoice_number", "subscription_id", "transaction_id",
			"subtotal", "tax_rate", "tax_amount", "total_amount", "currency",
			"description", "service_name", "issue_date", "due_date", "paid_date",
			"status", "pdf_generated", "pdf_path", "email_sent", "email_sent_at",
			"created_at", "updated_at",
		}).AddRow(
			int64(2), tenantID, "INV-202401-002", subscriptionID, nil,
			20.00, 5.00, 1.00, 21.00, "USD",
			params.Description, params.ServiceName, time.Now(), nil, nil,
			"paid", false, "", false, nil,
			time.Now(), time.Now(),
		)

		mock.ExpectQuery("SELECT (.+) FROM billing_invoices WHERE id = \\$1 AND tenant_id = \\$2").
			WithArgs(int64(2), tenantID).
			WillReturnRows(rows)
		
		mock.ExpectCommit()

		invoice, err := generator.GenerateInvoice(ctx, params)
		require.NoError(t, err)
		assert.NotNil(t, invoice)
		assert.NotNil(t, invoice.SubscriptionID)
		assert.Equal(t, subscriptionID, *invoice.SubscriptionID)
	})

	t.Run("Generate invoice with transaction ID", func(t *testing.T) {
		transactionID := int64(456)
		params := GenerateInvoiceParams{
			TenantID:      tenantID,
			TransactionID: &transactionID,
			Subtotal:      decimal.NewFromFloat(100.00),
			Description:   "Balance top-up",
			ServiceName:   "Top-up Service",
			CountryCode:   "YE",
			Currency:      "USD",
		}

		// Expect transaction 1 for CREATE
		mock.ExpectBegin()
		mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
			WithArgs(tenantID.String()).
			WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectExec("SELECT set_config\\('app.is_super_admin', \\$1, true\\)").
				WithArgs("false").
				WillReturnResult(sqlmock.NewResult(0, 0))

		mock.ExpectQuery("SELECT create_invoice_for_payment").
			WithArgs(tenantID, sqlmock.AnyArg(), transactionID, params.Subtotal, params.Description, params.ServiceName, params.CountryCode).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(3)))
		
		mock.ExpectCommit()

		// Expect transaction 2 for GET
		mock.ExpectBegin()
		mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
			WithArgs(tenantID.String()).
			WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectExec("SELECT set_config\\('app.is_super_admin', \\$1, true\\)").
				WithArgs("false").
				WillReturnResult(sqlmock.NewResult(0, 0))

		rows := sqlmock.NewRows([]string{
			"id", "tenant_id", "invoice_number", "subscription_id", "transaction_id",
			"subtotal", "tax_rate", "tax_amount", "total_amount", "currency",
			"description", "service_name", "issue_date", "due_date", "paid_date",
			"status", "pdf_generated", "pdf_path", "email_sent", "email_sent_at",
			"created_at", "updated_at",
		}).AddRow(
			int64(3), tenantID, "INV-202401-003", nil, transactionID,
			100.00, 5.00, 5.00, 105.00, "USD",
			params.Description, params.ServiceName, time.Now(), nil, nil,
			"paid", false, "", false, nil,
			time.Now(), time.Now(),
		)

		mock.ExpectQuery("SELECT (.+) FROM billing_invoices WHERE id = \\$1 AND tenant_id = \\$2").
			WithArgs(int64(3), tenantID).
			WillReturnRows(rows)
		
		mock.ExpectCommit()

		invoice, err := generator.GenerateInvoice(ctx, params)
		require.NoError(t, err)
		assert.NotNil(t, invoice)
		assert.NotNil(t, invoice.TransactionID)
		assert.Equal(t, transactionID, *invoice.TransactionID)
	})

	t.Run("Generate invoice with different country (Saudi Arabia)", func(t *testing.T) {
		params := GenerateInvoiceParams{
			TenantID:    tenantID,
			Subtotal:    decimal.NewFromFloat(100.00),
			Description: "Test invoice for Saudi Arabia",
			ServiceName: "Test Service",
			CountryCode: "SA", // Saudi Arabia has 15% VAT
			Currency:    "SAR",
		}

		// Expect transaction 1 for CREATE
		mock.ExpectBegin()
		mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
			WithArgs(tenantID.String()).
			WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectExec("SELECT set_config\\('app.is_super_admin', \\$1, true\\)").
				WithArgs("false").
				WillReturnResult(sqlmock.NewResult(0, 0))

		mock.ExpectQuery("SELECT create_invoice_for_payment").
			WithArgs(tenantID, sqlmock.AnyArg(), sqlmock.AnyArg(), params.Subtotal, params.Description, params.ServiceName, params.CountryCode).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(4)))
		
		mock.ExpectCommit()

		// Expect transaction 2 for GET
		mock.ExpectBegin()
		mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
			WithArgs(tenantID.String()).
			WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectExec("SELECT set_config\\('app.is_super_admin', \\$1, true\\)").
				WithArgs("false").
				WillReturnResult(sqlmock.NewResult(0, 0))

		rows := sqlmock.NewRows([]string{
			"id", "tenant_id", "invoice_number", "subscription_id", "transaction_id",
			"subtotal", "tax_rate", "tax_amount", "total_amount", "currency",
			"description", "service_name", "issue_date", "due_date", "paid_date",
			"status", "pdf_generated", "pdf_path", "email_sent", "email_sent_at",
			"created_at", "updated_at",
		}).AddRow(
			int64(4), tenantID, "INV-202401-004", nil, nil,
			100.00, 15.00, 15.00, 115.00, "SAR",
			params.Description, params.ServiceName, time.Now(), nil, nil,
			"paid", false, "", false, nil,
			time.Now(), time.Now(),
		)

		mock.ExpectQuery("SELECT (.+) FROM billing_invoices WHERE id = \\$1 AND tenant_id = \\$2").
			WithArgs(int64(4), tenantID).
			WillReturnRows(rows)
		
		mock.ExpectCommit()

		invoice, err := generator.GenerateInvoice(ctx, params)
		require.NoError(t, err)
		assert.NotNil(t, invoice)

		// Verify tax calculation (15% for Saudi Arabia)
		expectedTaxRate := decimal.NewFromFloat(15.00)
		assert.Equal(t, expectedTaxRate, invoice.TaxRate)
	})
}

// TestGetInvoiceByID tests retrieving invoice by ID
func TestGetInvoiceByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := zap.NewNop()
	generator := NewInvoiceGenerator(db, logger)

	ctx := context.Background()
	tenantID := uuid.New()
	invoiceID := int64(1)

	t.Run("Get existing invoice", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "tenant_id", "invoice_number", "subscription_id", "transaction_id",
			"subtotal", "tax_rate", "tax_amount", "total_amount", "currency",
			"description", "service_name", "issue_date", "due_date", "paid_date",
			"status", "pdf_generated", "pdf_path", "email_sent", "email_sent_at",
			"created_at", "updated_at",
		}).AddRow(
			invoiceID, tenantID, "INV-202401-001", nil, nil,
			50.00, 5.00, 2.50, 52.50, "USD",
			"Test invoice", "Test Service", time.Now(), nil, nil,
			"paid", false, "", false, nil,
			time.Now(), time.Now(),
		)

		// Expect transaction for SET app.current_tenant
		mock.ExpectBegin()
		mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
			WithArgs(tenantID.String()).
			WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectExec("SELECT set_config\\('app.is_super_admin', \\$1, true\\)").
				WithArgs("false").
				WillReturnResult(sqlmock.NewResult(0, 0))

		mock.ExpectQuery("SELECT (.+) FROM billing_invoices WHERE id = \\$1 AND tenant_id = \\$2").
			WithArgs(invoiceID, tenantID).
			WillReturnRows(rows)
		
		mock.ExpectCommit()

		invoice, err := generator.GetInvoiceByID(ctx, tenantID, invoiceID)
		require.NoError(t, err)
		assert.NotNil(t, invoice)
		assert.Equal(t, invoiceID, invoice.ID)
	})

	t.Run("Get non-existent invoice", func(t *testing.T) {
		// Expect transaction for SET app.current_tenant
		mock.ExpectBegin()
		mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
			WithArgs(tenantID.String()).
			WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectExec("SELECT set_config\\('app.is_super_admin', \\$1, true\\)").
				WithArgs("false").
				WillReturnResult(sqlmock.NewResult(0, 0))

		mock.ExpectQuery("SELECT (.+) FROM billing_invoices WHERE id = \\$1 AND tenant_id = \\$2").
			WithArgs(int64(99999), tenantID).
			WillReturnError(sql.ErrNoRows)
		
		mock.ExpectRollback()

		_, err := generator.GetInvoiceByID(ctx, tenantID, 99999)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invoice not found")
	})

	t.Run("Get invoice with wrong tenant ID", func(t *testing.T) {
		wrongTenantID := uuid.New()
		
		// Expect transaction for SET app.current_tenant (even if it's the wrong tenant ID, the function uses it)
		mock.ExpectBegin()
		mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
			WithArgs(wrongTenantID.String()).
			WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectExec("SELECT set_config\\('app.is_super_admin', \\$1, true\\)").
				WithArgs("false").
				WillReturnResult(sqlmock.NewResult(0, 0))

		mock.ExpectQuery("SELECT (.+) FROM billing_invoices WHERE id = \\$1 AND tenant_id = \\$2").
			WithArgs(invoiceID, wrongTenantID).
			WillReturnError(sql.ErrNoRows)
		
		mock.ExpectRollback()

		_, err := generator.GetInvoiceByID(ctx, wrongTenantID, invoiceID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invoice not found")
	})
}

// TestGetInvoiceByNumber tests retrieving invoice by number
func TestGetInvoiceByNumber(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := zap.NewNop()
	generator := NewInvoiceGenerator(db, logger)

	ctx := context.Background()
	tenantID := uuid.New()
	invoiceNumber := "INV-202401-001"

	t.Run("Get existing invoice by number", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "tenant_id", "invoice_number", "subscription_id", "transaction_id",
			"subtotal", "tax_rate", "tax_amount", "total_amount", "currency",
			"description", "service_name", "issue_date", "due_date", "paid_date",
			"status", "pdf_generated", "pdf_path", "email_sent", "email_sent_at",
			"created_at", "updated_at",
		}).AddRow(
			int64(1), tenantID, invoiceNumber, nil, nil,
			50.00, 5.00, 2.50, 52.50, "USD",
			"Test invoice", "Test Service", time.Now(), nil, nil,
			"paid", false, "", false, nil,
			time.Now(), time.Now(),
		)

		// Expect transaction for SET app.current_tenant
		mock.ExpectBegin()
		mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
			WithArgs(tenantID.String()).
			WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectExec("SELECT set_config\\('app.is_super_admin', \\$1, true\\)").
				WithArgs("false").
				WillReturnResult(sqlmock.NewResult(0, 0))

		mock.ExpectQuery("SELECT (.+) FROM billing_invoices WHERE invoice_number = \\$1 AND tenant_id = \\$2").
			WithArgs(invoiceNumber, tenantID).
			WillReturnRows(rows)
		
		mock.ExpectCommit()

		invoice, err := generator.GetInvoiceByNumber(ctx, tenantID, invoiceNumber)
		require.NoError(t, err)
		assert.NotNil(t, invoice)
		assert.Equal(t, invoiceNumber, invoice.InvoiceNumber)
	})

	t.Run("Get non-existent invoice by number", func(t *testing.T) {
		// Expect transaction for SET app.current_tenant
		mock.ExpectBegin()
		mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
			WithArgs(tenantID.String()).
			WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectExec("SELECT set_config\\('app.is_super_admin', \\$1, true\\)").
				WithArgs("false").
				WillReturnResult(sqlmock.NewResult(0, 0))

		mock.ExpectQuery("SELECT (.+) FROM billing_invoices WHERE invoice_number = \\$1 AND tenant_id = \\$2").
			WithArgs("INV-NONE", tenantID).
			WillReturnError(sql.ErrNoRows)
		
		mock.ExpectRollback()

		_, err := generator.GetInvoiceByNumber(ctx, tenantID, "INV-NONE")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invoice not found")
	})
}

// TestMarkPDFGenerated tests marking invoice PDF as generated
func TestMarkPDFGenerated(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := zap.NewNop()
	generator := NewInvoiceGenerator(db, logger)

	ctx := context.Background()
	tenantID := uuid.New()
	invoiceID := int64(1)

	t.Run("Mark PDF as generated", func(t *testing.T) {
		pdfPath := "/invoices/INV-202602-0001.pdf"

		mock.ExpectBegin()
		mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
			WithArgs(tenantID.String()).
			WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectExec("SELECT set_config\\('app.is_super_admin', \\$1, true\\)").
				WithArgs("false").
				WillReturnResult(sqlmock.NewResult(0, 0))

		mock.ExpectExec("UPDATE billing_invoices SET pdf_generated = true").
			WithArgs(pdfPath, invoiceID, tenantID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		
		mock.ExpectCommit()

		err := generator.MarkPDFGenerated(ctx, tenantID, invoiceID, pdfPath)
		require.NoError(t, err)
	})
}

// TestMarkEmailSent tests marking invoice email as sent
func TestMarkEmailSent(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := zap.NewNop()
	generator := NewInvoiceGenerator(db, logger)

	ctx := context.Background()
	tenantID := uuid.New()
	invoiceID := int64(1)

	t.Run("Mark email as sent", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
			WithArgs(tenantID.String()).
			WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectExec("SELECT set_config\\('app.is_super_admin', \\$1, true\\)").
				WithArgs("false").
				WillReturnResult(sqlmock.NewResult(0, 0))

		mock.ExpectExec("UPDATE billing_invoices SET email_sent = true").
			WithArgs(invoiceID, tenantID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		
		mock.ExpectCommit()

		err := generator.MarkEmailSent(ctx, tenantID, invoiceID)
		require.NoError(t, err)
	})
}

// Helper functions

func setupTestTenant(t *testing.T, db *sql.DB, tenantID uuid.UUID) {
	// TODO: Setup test tenant in database
}

func setupInvoiceTestDB(t *testing.T) (*sql.DB, error) {
	return nil, nil // Tests skip when short=true
}
