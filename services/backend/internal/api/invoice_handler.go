package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/al-hudhud/backend/internal/billing"
	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
)

// InvoiceHandler handles invoice-related API requests
// Implements Requirement 40.8
type InvoiceHandler struct {
	service *billing.InvoiceService
	logger  *zap.Logger
}

// NewInvoiceHandler creates a new invoice handler
func NewInvoiceHandler(db *sql.DB, logger *zap.Logger, pdfOutputDir string) *InvoiceHandler {
	return &InvoiceHandler{
		service: billing.NewInvoiceService(db, logger, pdfOutputDir),
		logger:  logger,
	}
}

// ListInvoicesRequest represents the request for listing invoices
type ListInvoicesRequest struct {
	Status    string `form:"status"`     // Filter by status
	FromDate  string `form:"from_date"`  // Filter from date (YYYY-MM-DD)
	ToDate    string `form:"to_date"`    // Filter to date (YYYY-MM-DD)
	Limit     int    `form:"limit"`      // Pagination limit
	Offset    int    `form:"offset"`     // Pagination offset
}

// ListInvoicesResponse represents the response for listing invoices
type ListInvoicesResponse struct {
	Success  bool                `json:"success"`
	Invoices []InvoiceSummary    `json:"invoices"`
	Total    int                 `json:"total"`
	Limit    int                 `json:"limit"`
	Offset   int                 `json:"offset"`
}

// InvoiceSummary represents a summary of an invoice
type InvoiceSummary struct {
	ID            int64   `json:"id"`
	InvoiceNumber string  `json:"invoice_number"`
	IssueDate     string  `json:"issue_date"`
	TotalAmount   string  `json:"total_amount"`
	Currency      string  `json:"currency"`
	Status        string  `json:"status"`
	ServiceName   string  `json:"service_name"`
	PDFGenerated  bool    `json:"pdf_generated"`
	EmailSent     bool    `json:"email_sent"`
}

// ListInvoices handles GET /api/invoices
// Implements Requirement 40.8 - List invoices with tenant isolation and filtering
func (ih *InvoiceHandler) ListInvoices(c *gin.Context) {
	// Extract tenant ID from context (set by auth middleware)
	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "tenant_id not found in context",
		})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid tenant_id",
		})
		return
	}

	// Parse request
	var req ListInvoicesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid request parameters",
		})
		return
	}

	// Set defaults
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 50
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	// Parse dates
	var fromDate, toDate *time.Time
	if req.FromDate != "" {
		parsed, err := time.Parse("2006-01-02", req.FromDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "invalid from_date format (use YYYY-MM-DD)",
			})
			return
		}
		fromDate = &parsed
	}
	if req.ToDate != "" {
		parsed, err := time.Parse("2006-01-02", req.ToDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "invalid to_date format (use YYYY-MM-DD)",
			})
			return
		}
		toDate = &parsed
	}

	// Get invoices from database
	isSuperAdmin := c.GetString("user_type") == "SUPER_ADMIN"
	invoices, total, err := ih.getInvoices(c.Request.Context(), tenantID, isSuperAdmin, req.Status, fromDate, toDate, req.Limit, req.Offset)
	if err != nil {
		ih.logger.Error("Failed to get invoices",
			zap.Error(err),
			zap.String("tenant_id", tenantID.String()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to retrieve invoices",
		})
		return
	}

	c.JSON(http.StatusOK, ListInvoicesResponse{
		Success:  true,
		Invoices: invoices,
		Total:    total,
		Limit:    req.Limit,
		Offset:   req.Offset,
	})
}


// GetInvoiceResponse represents the response for getting a single invoice
type GetInvoiceResponse struct {
	Success bool              `json:"success"`
	Invoice *InvoiceDetail    `json:"invoice,omitempty"`
	Error   string            `json:"error,omitempty"`
}

// InvoiceDetail represents detailed invoice information
type InvoiceDetail struct {
	ID              int64   `json:"id"`
	InvoiceNumber   string  `json:"invoice_number"`
	SubscriptionID  *int64  `json:"subscription_id,omitempty"`
	TransactionID   *int64  `json:"transaction_id,omitempty"`
	Subtotal        string  `json:"subtotal"`
	TaxRate         string  `json:"tax_rate"`
	TaxAmount       string  `json:"tax_amount"`
	TotalAmount     string  `json:"total_amount"`
	Currency        string  `json:"currency"`
	Description     string  `json:"description"`
	ServiceName     string  `json:"service_name"`
	IssueDate       string  `json:"issue_date"`
	DueDate         *string `json:"due_date,omitempty"`
	PaidDate        *string `json:"paid_date,omitempty"`
	Status          string  `json:"status"`
	PDFGenerated    bool    `json:"pdf_generated"`
	PDFPath         string  `json:"pdf_path,omitempty"`
	EmailSent       bool    `json:"email_sent"`
	EmailSentAt     *string `json:"email_sent_at,omitempty"`
}

// GetInvoice handles GET /api/invoices/:id
func (ih *InvoiceHandler) GetInvoice(c *gin.Context) {
	// Extract tenant ID
	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "tenant_id not found in context",
		})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid tenant_id",
		})
		return
	}

	// Parse invoice ID
	invoiceIDStr := c.Param("id")
	invoiceID, err := strconv.ParseInt(invoiceIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid invoice ID",
		})
		return
	}

	// Get invoice
	invoice, err := ih.service.GetInvoiceByID(c.Request.Context(), tenantID, invoiceID)
	if err != nil {
		ih.logger.Error("Failed to get invoice",
			zap.Error(err),
			zap.Int64("invoice_id", invoiceID),
		)
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "invoice not found",
		})
		return
	}

	// Convert to response format
	detail := ih.invoiceToDetail(invoice)

	c.JSON(http.StatusOK, GetInvoiceResponse{
		Success: true,
		Invoice: detail,
	})
}

// DownloadInvoicePDF handles GET /api/invoices/:id/pdf
func (ih *InvoiceHandler) DownloadInvoicePDF(c *gin.Context) {
	// Extract tenant ID
	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "tenant_id not found in context",
		})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid tenant_id",
		})
		return
	}

	// Parse invoice ID
	invoiceIDStr := c.Param("id")
	invoiceID, err := strconv.ParseInt(invoiceIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid invoice ID",
		})
		return
	}

	// Get invoice
	invoice, err := ih.service.GetInvoiceByID(c.Request.Context(), tenantID, invoiceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "invoice not found",
		})
		return
	}

	// Check if PDF exists, generate if not
	if !invoice.PDFGenerated {
		// Generate PDF
		pdfPath, err := ih.service.GeneratePDF(c.Request.Context(), invoice)
		if err != nil {
			ih.logger.Error("Failed to generate PDF",
				zap.Error(err),
				zap.Int64("invoice_id", invoiceID),
			)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "failed to generate PDF",
			})
			return
		}

		// Mark as generated
		if err := ih.service.MarkPDFGenerated(c.Request.Context(), tenantID, invoiceID, pdfPath); err != nil {
			ih.logger.Error("Failed to mark PDF as generated", zap.Error(err))
		}

		invoice.PDFPath = pdfPath
	}

	// Serve the PDF file
	c.FileAttachment(invoice.PDFPath, invoice.InvoiceNumber+".pdf")
}

// Helper methods

func (ih *InvoiceHandler) getInvoices(
	ctx context.Context,
	tenantID uuid.UUID,
	isSuperAdmin bool,
	status string,
	fromDate, toDate *time.Time,
	limit, offset int,
) ([]InvoiceSummary, int, error) {
	var invoices []InvoiceSummary
	var total int

	err := db.RunInTenantTx(ctx, tenantID, isSuperAdmin, ih.service.GetDB(), func(tx *sql.Tx) error {
		// Get total count
		countQuery := "SELECT count(*) FROM billing_invoices WHERE tenant_id = $1"
		args := []interface{}{tenantID}
		argIdx := 2

		if status != "" {
			countQuery += fmt.Sprintf(" AND status = $%d", argIdx)
			args = append(args, status)
			argIdx++
		}
		if fromDate != nil {
			countQuery += fmt.Sprintf(" AND issue_date >= $%d", argIdx)
			args = append(args, *fromDate)
			argIdx++
		}
		if toDate != nil {
			countQuery += fmt.Sprintf(" AND issue_date <= $%d", argIdx)
			args = append(args, *toDate)
			argIdx++
		}

		err := tx.QueryRowContext(ctx, countQuery, args...).Scan(&total)
		if err != nil {
			return err
		}

		// Get invoices
		query := `
			SELECT id, invoice_number, issue_date, total_amount, currency, status, service_name, pdf_generated, email_sent
			FROM billing_invoices
			WHERE tenant_id = $1
		`
		qArgs := []interface{}{tenantID}
		qIdx := 2

		if status != "" {
			query += fmt.Sprintf(" AND status = $%d", qIdx)
			qArgs = append(qArgs, status)
			qIdx++
		}
		if fromDate != nil {
			query += fmt.Sprintf(" AND issue_date >= $%d", qIdx)
			qArgs = append(qArgs, *fromDate)
			qIdx++
		}
		if toDate != nil {
			query += fmt.Sprintf(" AND issue_date <= $%d", qIdx)
			qArgs = append(qArgs, *toDate)
			qIdx++
		}

		query += fmt.Sprintf(" ORDER BY issue_date DESC LIMIT $%d OFFSET $%d", qIdx, qIdx+1)
		qArgs = append(qArgs, limit, offset)

		rows, err := tx.QueryContext(ctx, query, qArgs...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var inv InvoiceSummary
			var totalAmount decimal.Decimal
			var issueDate time.Time
			err := rows.Scan(
				&inv.ID, &inv.InvoiceNumber, &issueDate, &totalAmount,
				&inv.Currency, &inv.Status, &inv.ServiceName,
				&inv.PDFGenerated, &inv.EmailSent,
			)
			if err != nil {
				return err
			}
			inv.IssueDate = issueDate.Format("2006-01-02")
			inv.TotalAmount = totalAmount.StringFixed(2)
			invoices = append(invoices, inv)
		}
		return nil
	})

	return invoices, total, err
}

func (ih *InvoiceHandler) invoiceToDetail(invoice *models.Invoice) *InvoiceDetail {
	detail := &InvoiceDetail{
		ID:            invoice.ID,
		InvoiceNumber: invoice.InvoiceNumber,
		Subtotal:      invoice.Subtotal.StringFixed(2),
		TaxRate:       invoice.TaxRate.StringFixed(2),
		TaxAmount:     invoice.TaxAmount.StringFixed(2),
		TotalAmount:   invoice.TotalAmount.StringFixed(2),
		Currency:      invoice.Currency,
		Description:   invoice.Description,
		ServiceName:   invoice.ServiceName,
		IssueDate:     invoice.IssueDate.Format("2006-01-02"),
		Status:        invoice.Status,
		PDFGenerated:  invoice.PDFGenerated,
		PDFPath:       invoice.PDFPath,
		EmailSent:     invoice.EmailSent,
	}

	if invoice.SubscriptionID != nil {
		detail.SubscriptionID = invoice.SubscriptionID
	}
	if invoice.TransactionID != nil {
		detail.TransactionID = invoice.TransactionID
	}
	if invoice.DueDate != nil {
		dueDate := invoice.DueDate.Format("2006-01-02")
		detail.DueDate = &dueDate
	}
	if invoice.PaidDate != nil {
		paidDate := invoice.PaidDate.Format("2006-01-02")
		detail.PaidDate = &paidDate
	}
	if invoice.EmailSentAt != nil {
		emailSentAt := invoice.EmailSentAt.Format("2006-01-02 15:04:05")
		detail.EmailSentAt = &emailSentAt
	}

	return detail
}
