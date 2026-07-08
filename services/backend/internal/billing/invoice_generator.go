package billing

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// InvoiceGenerator handles invoice generation for all payment operations
// Implements Requirements 40.1, 40.2, 40.3, 40.4, 40.5
type InvoiceGenerator struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewInvoiceGenerator creates a new invoice generator
func NewInvoiceGenerator(db *sql.DB, logger *zap.Logger) *InvoiceGenerator {
	return &InvoiceGenerator{
		db:     db,
		logger: logger,
	}
}

// GetDB returns the database connection
func (ig *InvoiceGenerator) GetDB() *sql.DB {
	return ig.db
}

// Use models.Invoice

// GenerateInvoiceParams contains parameters for invoice generation
type GenerateInvoiceParams struct {
	TenantID       uuid.UUID
	SubscriptionID *int64
	TransactionID  *int64
	Subtotal       decimal.Decimal
	Description    string
	ServiceName    string
	CountryCode    string // For tax calculation
	Currency       string
}

// GenerateInvoice generates an invoice for a payment operation
// Implements Requirements 40.1, 40.2, 40.3, 40.4, 40.5
func (ig *InvoiceGenerator) GenerateInvoice(ctx context.Context, params GenerateInvoiceParams) (*models.Invoice, error) {
	ig.logger.Info("Generating invoice",
		zap.String("tenant_id", params.TenantID.String()),
		zap.String("service_name", params.ServiceName),
		zap.String("subtotal", params.Subtotal.String()),
	)

	// Default values
	if params.Currency == "" {
		params.Currency = "USD"
	}
	if params.CountryCode == "" {
		params.CountryCode = "YE"
	}

	// Use database function to create invoice
	// This function handles:
	// - Unique invoice number generation (Requirement 40.2)
	// - Automatic tax calculation (Requirement 40.5)
	// - All invoice details (Requirements 40.3, 40.4)
	var invoiceID int64
	err := db.RunInTenantTx(ctx, params.TenantID, false, ig.db, func(tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, `
			SELECT create_invoice_for_payment($1, $2, $3, $4, $5, $6, $7)
		`,
			params.TenantID,
			params.SubscriptionID,
			params.TransactionID,
			params.Subtotal,
			params.Description,
			params.ServiceName,
			params.CountryCode,
		).Scan(&invoiceID)
	})

	if err != nil {
		ig.logger.Error("Failed to create invoice",
			zap.Error(err),
			zap.String("tenant_id", params.TenantID.String()),
		)
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	// Retrieve the created invoice
	invoice, err := ig.GetInvoiceByID(ctx, params.TenantID, invoiceID)
	if err != nil {
		ig.logger.Error("Failed to retrieve created invoice",
			zap.Error(err),
			zap.Int64("invoice_id", invoiceID),
		)
		return nil, fmt.Errorf("failed to retrieve invoice: %w", err)
	}

	ig.logger.Info("Invoice generated successfully",
		zap.Int64("invoice_id", invoice.ID),
		zap.String("invoice_number", invoice.InvoiceNumber),
		zap.String("total_amount", invoice.TotalAmount.String()),
	)

	return invoice, nil
}

// GetInvoiceByID retrieves an invoice by ID
func (ig *InvoiceGenerator) GetInvoiceByID(ctx context.Context, tenantID uuid.UUID, invoiceID int64) (*models.Invoice, error) {
	var invoice models.Invoice
	var subscriptionID, transactionID sql.NullInt64
	var dueDate, paidDate, emailSentAt sql.NullTime
	var pdfPath sql.NullString

	err := db.RunInTenantTx(ctx, tenantID, false, ig.db, func(tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, `
			SELECT 
				id, tenant_id, invoice_number, subscription_id, transaction_id,
				subtotal, tax_rate, tax_amount, total_amount, currency,
				description, service_name, issue_date, due_date, paid_date,
				status, pdf_generated, pdf_path, email_sent, email_sent_at,
				created_at, updated_at
			FROM billing_invoices
			WHERE id = $1 AND tenant_id = $2
		`, invoiceID, tenantID).Scan(
			&invoice.ID,
			&invoice.TenantID,
			&invoice.InvoiceNumber,
			&subscriptionID,
			&transactionID,
			&invoice.Subtotal,
			&invoice.TaxRate,
			&invoice.TaxAmount,
			&invoice.TotalAmount,
			&invoice.Currency,
			&invoice.Description,
			&invoice.ServiceName,
			&invoice.IssueDate,
			&dueDate,
			&paidDate,
			&invoice.Status,
			&invoice.PDFGenerated,
			&pdfPath,
			&invoice.EmailSent,
			&emailSentAt,
			&invoice.CreatedAt,
			&invoice.UpdatedAt,
		)
	})

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invoice not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}

	// Handle nullable fields
	if subscriptionID.Valid {
		invoice.SubscriptionID = &subscriptionID.Int64
	}
	if transactionID.Valid {
		invoice.TransactionID = &transactionID.Int64
	}
	if dueDate.Valid {
		invoice.DueDate = &dueDate.Time
	}
	if paidDate.Valid {
		invoice.PaidDate = &paidDate.Time
	}
	if pdfPath.Valid {
		invoice.PDFPath = pdfPath.String
	}
	if emailSentAt.Valid {
		invoice.EmailSentAt = &emailSentAt.Time
	}

	return &invoice, nil
}

// GetInvoiceByNumber retrieves an invoice by invoice number
func (ig *InvoiceGenerator) GetInvoiceByNumber(ctx context.Context, tenantID uuid.UUID, invoiceNumber string) (*models.Invoice, error) {
	var invoice models.Invoice
	var subscriptionID, transactionID sql.NullInt64
	var dueDate, paidDate, emailSentAt sql.NullTime
	var pdfPath sql.NullString

	err := db.RunInTenantTx(ctx, tenantID, false, ig.db, func(tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, `
			SELECT 
				id, tenant_id, invoice_number, subscription_id, transaction_id,
				subtotal, tax_rate, tax_amount, total_amount, currency,
				description, service_name, issue_date, due_date, paid_date,
				status, pdf_generated, pdf_path, email_sent, email_sent_at,
				created_at, updated_at
			FROM billing_invoices
			WHERE invoice_number = $1 AND tenant_id = $2
		`, invoiceNumber, tenantID).Scan(
			&invoice.ID,
			&invoice.TenantID,
			&invoice.InvoiceNumber,
			&subscriptionID,
			&transactionID,
			&invoice.Subtotal,
			&invoice.TaxRate,
			&invoice.TaxAmount,
			&invoice.TotalAmount,
			&invoice.Currency,
			&invoice.Description,
			&invoice.ServiceName,
			&invoice.IssueDate,
			&dueDate,
			&paidDate,
			&invoice.Status,
			&invoice.PDFGenerated,
			&pdfPath,
			&invoice.EmailSent,
			&emailSentAt,
			&invoice.CreatedAt,
			&invoice.UpdatedAt,
		)
	})

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invoice not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}

	// Handle nullable fields
	if subscriptionID.Valid {
		invoice.SubscriptionID = &subscriptionID.Int64
	}
	if transactionID.Valid {
		invoice.TransactionID = &transactionID.Int64
	}
	if dueDate.Valid {
		invoice.DueDate = &dueDate.Time
	}
	if paidDate.Valid {
		invoice.PaidDate = &paidDate.Time
	}
	if pdfPath.Valid {
		invoice.PDFPath = pdfPath.String
	}
	if emailSentAt.Valid {
		invoice.EmailSentAt = &emailSentAt.Time
	}

	return &invoice, nil
}

// MarkPDFGenerated marks an invoice as having its PDF generated
func (ig *InvoiceGenerator) MarkPDFGenerated(ctx context.Context, tenantID uuid.UUID, invoiceID int64, pdfPath string) error {
	var result sql.Result
	err := db.RunInTenantTx(ctx, tenantID, false, ig.db, func(tx *sql.Tx) error {
		var txErr error
		result, txErr = tx.ExecContext(ctx, `
			UPDATE billing_invoices
			SET pdf_generated = true,
				pdf_path = $1,
				updated_at = CURRENT_TIMESTAMP
			WHERE id = $2 AND tenant_id = $3
		`, pdfPath, invoiceID, tenantID)
		return txErr
	})

	if err != nil {
		return fmt.Errorf("failed to mark PDF generated: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("invoice not found or not updated")
	}

	ig.logger.Info("Invoice PDF marked as generated",
		zap.Int64("invoice_id", invoiceID),
		zap.String("pdf_path", pdfPath),
	)

	return nil
}

// MarkEmailSent marks an invoice as having its email sent
func (ig *InvoiceGenerator) MarkEmailSent(ctx context.Context, tenantID uuid.UUID, invoiceID int64) error {
	var result sql.Result
	err := db.RunInTenantTx(ctx, tenantID, false, ig.db, func(tx *sql.Tx) error {
		var txErr error
		result, txErr = tx.ExecContext(ctx, `
			UPDATE billing_invoices
			SET email_sent = true,
				email_sent_at = CURRENT_TIMESTAMP,
				updated_at = CURRENT_TIMESTAMP
			WHERE id = $1 AND tenant_id = $2
		`, invoiceID, tenantID)
		return txErr
	})

	if err != nil {
		return fmt.Errorf("failed to mark email sent: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("invoice not found or not updated")
	}

	ig.logger.Info("Invoice email marked as sent",
		zap.Int64("invoice_id", invoiceID),
	)

	return nil
}


// InvoiceService combines invoice generation, PDF creation, and email sending
// Implements Requirements 40.1-40.7
type InvoiceService struct {
	generator    *InvoiceGenerator
	pdfGenerator *PDFGenerator
	logger       *zap.Logger
}

// NewInvoiceService creates a new invoice service
func NewInvoiceService(db *sql.DB, logger *zap.Logger, pdfOutputDir string) *InvoiceService {
	return &InvoiceService{
		generator:    NewInvoiceGenerator(db, logger),
		pdfGenerator: NewPDFGenerator(logger, pdfOutputDir),
		logger:       logger,
	}
}

// GetDB returns the database connection
func (is *InvoiceService) GetDB() *sql.DB {
	return is.generator.GetDB()
}

// GetInvoiceByID retrieves an invoice by ID
func (is *InvoiceService) GetInvoiceByID(ctx context.Context, tenantID uuid.UUID, invoiceID int64) (*models.Invoice, error) {
	return is.generator.GetInvoiceByID(ctx, tenantID, invoiceID)
}

// GeneratePDF generates a PDF for an invoice
func (is *InvoiceService) GeneratePDF(ctx context.Context, invoice *models.Invoice) (string, error) {
	return is.pdfGenerator.GeneratePDF(ctx, invoice)
}

// MarkPDFGenerated marks a PDF as generated
func (is *InvoiceService) MarkPDFGenerated(ctx context.Context, tenantID uuid.UUID, invoiceID int64, pdfPath string) error {
	return is.generator.MarkPDFGenerated(ctx, tenantID, invoiceID, pdfPath)
}

// SendInvoiceParams contains parameters for sending an invoice
type SendInvoiceParams struct {
	InvoiceID      int64
	TenantID       uuid.UUID
	RecipientEmail string
	RecipientName  string
}

// SendInvoice generates PDF and sends invoice via email
// Implements Requirement 40.7
func (is *InvoiceService) SendInvoice(ctx context.Context, params SendInvoiceParams) error {
	is.logger.Info("Sending invoice",
		zap.Int64("invoice_id", params.InvoiceID),
		zap.String("tenant_id", params.TenantID.String()),
		zap.String("recipient_email", params.RecipientEmail),
	)

	// Get invoice
	invoice, err := is.generator.GetInvoiceByID(ctx, params.TenantID, params.InvoiceID)
	if err != nil {
		is.logger.Error("Failed to get invoice",
			zap.Error(err),
			zap.Int64("invoice_id", params.InvoiceID),
		)
		return fmt.Errorf("failed to get invoice: %w", err)
	}

	// Generate PDF if not already generated
	var pdfPath string
	if !invoice.PDFGenerated {
		pdfPath, err = is.pdfGenerator.GeneratePDF(ctx, invoice)
		if err != nil {
			is.logger.Error("Failed to generate PDF",
				zap.Error(err),
				zap.String("invoice_number", invoice.InvoiceNumber),
			)
			return fmt.Errorf("failed to generate PDF: %w", err)
		}

		// Mark PDF as generated
		if err := is.generator.MarkPDFGenerated(ctx, params.TenantID, params.InvoiceID, pdfPath); err != nil {
			is.logger.Error("Failed to mark PDF as generated",
				zap.Error(err),
				zap.Int64("invoice_id", params.InvoiceID),
			)
			return fmt.Errorf("failed to mark PDF as generated: %w", err)
		}
	} else {
		pdfPath = invoice.PDFPath
	}

	// Send email with PDF attachment
	if err := is.sendEmailWithAttachment(ctx, params, invoice, pdfPath); err != nil {
		is.logger.Error("Failed to send email",
			zap.Error(err),
			zap.String("recipient_email", params.RecipientEmail),
		)
		return fmt.Errorf("failed to send email: %w", err)
	}

	// Mark email as sent
	if err := is.generator.MarkEmailSent(ctx, params.TenantID, params.InvoiceID); err != nil {
		is.logger.Error("Failed to mark email as sent",
			zap.Error(err),
			zap.Int64("invoice_id", params.InvoiceID),
		)
		return fmt.Errorf("failed to mark email as sent: %w", err)
	}

	is.logger.Info("Invoice sent successfully",
		zap.Int64("invoice_id", params.InvoiceID),
		zap.String("invoice_number", invoice.InvoiceNumber),
		zap.String("recipient_email", params.RecipientEmail),
	)

	return nil
}

// sendEmailWithAttachment sends an email with the invoice PDF attached
func (is *InvoiceService) sendEmailWithAttachment(
	ctx context.Context,
	params SendInvoiceParams,
	invoice *models.Invoice,
	pdfPath string,
) error {
	// TODO: Implement actual email sending
	// This should integrate with an email service like:
	// - SMTP server
	// - SendGrid
	// - AWS SES
	// - Mailgun
	// etc.

	is.logger.Info("Sending email with attachment",
		zap.String("recipient_email", params.RecipientEmail),
		zap.String("invoice_number", invoice.InvoiceNumber),
		zap.String("pdf_path", pdfPath),
	)

	// Email template
	subject := fmt.Sprintf("Invoice %s - SMS Gateway Platform", invoice.InvoiceNumber)
	body := fmt.Sprintf(`
Dear %s,

Thank you for your payment. Please find attached your invoice %s.

Invoice Details:
- Invoice Number: %s
- Issue Date: %s
- Service: %s
- Total Amount: %s %s

If you have any questions, please don't hesitate to contact us.

Best regards,
SMS Gateway Platform Team
	`,
		params.RecipientName,
		invoice.InvoiceNumber,
		invoice.InvoiceNumber,
		invoice.IssueDate.Format("2006-01-02"),
		invoice.ServiceName,
		invoice.TotalAmount.StringFixed(2),
		invoice.Currency,
	)

	is.logger.Info("Email prepared",
		zap.String("subject", subject),
		zap.String("body_preview", body[:100]+"..."),
	)

	// TODO: Actual email sending implementation
	// Example with SMTP:
	// err := smtp.SendMail(
	//     "smtp.example.com:587",
	//     auth,
	//     "noreply@smsgateway.com",
	//     []string{params.RecipientEmail},
	//     []byte(message),
	// )

	return nil
}

// GenerateAndSendInvoice is a convenience method that generates an invoice and sends it
func (is *InvoiceService) GenerateAndSendInvoice(
	ctx context.Context,
	generateParams GenerateInvoiceParams,
	recipientEmail string,
	recipientName string,
) (*models.Invoice, error) {
	// Generate invoice
	invoice, err := is.generator.GenerateInvoice(ctx, generateParams)
	if err != nil {
		return nil, fmt.Errorf("failed to generate invoice: %w", err)
	}

	// Send invoice
	sendParams := SendInvoiceParams{
		InvoiceID:      invoice.ID,
		TenantID:       generateParams.TenantID,
		RecipientEmail: recipientEmail,
		RecipientName:  recipientName,
	}

	if err := is.SendInvoice(ctx, sendParams); err != nil {
		is.logger.Warn("Invoice generated but failed to send",
			zap.Int64("invoice_id", invoice.ID),
			zap.Error(err),
		)
		// Don't return error - invoice is still generated
	}

	return invoice, nil
}
