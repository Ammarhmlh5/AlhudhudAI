package billing

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/al-hudhud/backend/internal/models"
	"github.com/jung-kurt/gofpdf"
	"go.uber.org/zap"
)

// PDFGenerator handles PDF generation for invoices
// Implements Requirements 40.6, 40.9, 40.11
type PDFGenerator struct {
	logger      *zap.Logger
	outputDir   string
	companyName string
	companyLogo string
}

// NewPDFGenerator creates a new PDF generator
func NewPDFGenerator(logger *zap.Logger, outputDir string) *PDFGenerator {
	return &PDFGenerator{
		logger:      logger,
		outputDir:   outputDir,
		companyName: "SMS Gateway Platform",
		companyLogo: "", // Path to company logo (optional)
	}
}

// GeneratePDF generates a PDF for an invoice
// Implements Requirements 40.6, 40.9, 40.11
func (pg *PDFGenerator) GeneratePDF(ctx context.Context, invoice *models.Invoice) (string, error) {
	pg.logger.Info("Generating PDF for invoice",
		zap.String("invoice_number", invoice.InvoiceNumber),
		zap.Int64("invoice_id", invoice.ID),
	)

	// Create PDF
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Add company header
	pg.addHeader(pdf)

	// Add invoice details
	pg.addInvoiceDetails(pdf, invoice)

	// Add line items table
	pg.addLineItems(pdf, invoice)

	// Add totals
	pg.addTotals(pdf, invoice)

	// Add footer
	pg.addFooter(pdf)

	// Ensure output directory exists
	if err := os.MkdirAll(pg.outputDir, 0755); err != nil {
		pg.logger.Error("Failed to create output directory",
			zap.Error(err),
			zap.String("dir", pg.outputDir),
		)
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate filename
	filename := fmt.Sprintf("%s.pdf", invoice.InvoiceNumber)
	filepath := filepath.Join(pg.outputDir, filename)

	// Save PDF
	if err := pdf.OutputFileAndClose(filepath); err != nil {
		pg.logger.Error("Failed to save PDF",
			zap.Error(err),
			zap.String("filepath", filepath),
		)
		return "", fmt.Errorf("failed to save PDF: %w", err)
	}

	pg.logger.Info("PDF generated successfully",
		zap.String("invoice_number", invoice.InvoiceNumber),
		zap.String("filepath", filepath),
	)

	return filepath, nil
}


// addHeader adds company header with logo and colors
func (pg *PDFGenerator) addHeader(pdf *gofpdf.Fpdf) {
	// Company colors (blue theme)
	pdf.SetFillColor(41, 128, 185) // Blue
	pdf.Rect(0, 0, 210, 40, "F")

	// Company logo (if available)
	if pg.companyLogo != "" && fileExists(pg.companyLogo) {
		pdf.Image(pg.companyLogo, 15, 10, 30, 0, false, "", 0, "")
	}

	// Company name
	pdf.SetTextColor(255, 255, 255) // White
	pdf.SetFont("Arial", "B", 24)
	pdf.SetXY(50, 15)
	pdf.Cell(0, 10, pg.companyName)

	// Reset colors
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(30)
}

// addInvoiceDetails adds invoice header information
func (pg *PDFGenerator) addInvoiceDetails(pdf *gofpdf.Fpdf, invoice *models.Invoice) {
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 10, "INVOICE / فاتورة")
	pdf.Ln(12)

	// Invoice details in two columns
	pdf.SetFont("Arial", "", 10)

	// Left column
	pdf.SetX(15)
	pdf.SetFont("Arial", "B", 10)
	pdf.Cell(40, 6, "Invoice Number:")
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(60, 6, invoice.InvoiceNumber)
	pdf.Ln(7)

	pdf.SetX(15)
	pdf.SetFont("Arial", "B", 10)
	pdf.Cell(40, 6, "Issue Date:")
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(60, 6, invoice.IssueDate.Format("2006-01-02"))
	pdf.Ln(7)

	pdf.SetX(15)
	pdf.SetFont("Arial", "B", 10)
	pdf.Cell(40, 6, "Status:")
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(60, 6, invoice.Status)
	pdf.Ln(7)

	// Service name
	if invoice.ServiceName != "" {
		pdf.SetX(15)
		pdf.SetFont("Arial", "B", 10)
		pdf.Cell(40, 6, "Service:")
		pdf.SetFont("Arial", "", 10)
		pdf.Cell(60, 6, invoice.ServiceName)
		pdf.Ln(7)
	}

	pdf.Ln(5)
}

// addLineItems adds the invoice line items table
func (pg *PDFGenerator) addLineItems(pdf *gofpdf.Fpdf, invoice *models.Invoice) {
	// Table header
	pdf.SetFillColor(240, 240, 240)
	pdf.SetFont("Arial", "B", 10)

	pdf.SetX(15)
	pdf.CellFormat(100, 8, "Description", "1", 0, "L", true, 0, "")
	pdf.CellFormat(40, 8, "Amount", "1", 0, "R", true, 0, "")
	pdf.CellFormat(40, 8, "Currency", "1", 1, "C", true, 0, "")

	// Table content
	pdf.SetFont("Arial", "", 10)
	pdf.SetX(15)
	pdf.CellFormat(100, 8, invoice.Description, "1", 0, "L", false, 0, "")
	pdf.CellFormat(40, 8, invoice.Subtotal.StringFixed(2), "1", 0, "R", false, 0, "")
	pdf.CellFormat(40, 8, invoice.Currency, "1", 1, "C", false, 0, "")

	pdf.Ln(5)
}

// addTotals adds the totals section
func (pg *PDFGenerator) addTotals(pdf *gofpdf.Fpdf, invoice *models.Invoice) {
	// Subtotal
	pdf.SetX(115)
	pdf.SetFont("Arial", "B", 10)
	pdf.Cell(40, 7, "Subtotal:")
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(40, 7, fmt.Sprintf("%s %s", invoice.Subtotal.StringFixed(2), invoice.Currency))
	pdf.Ln(8)

	// Tax
	pdf.SetX(115)
	pdf.SetFont("Arial", "B", 10)
	pdf.Cell(40, 7, fmt.Sprintf("Tax (%s%%):", invoice.TaxRate.StringFixed(2)))
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(40, 7, fmt.Sprintf("%s %s", invoice.TaxAmount.StringFixed(2), invoice.Currency))
	pdf.Ln(8)

	// Total (highlighted)
	pdf.SetFillColor(41, 128, 185)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetX(115)
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(40, 10, "TOTAL:", "1", 0, "L", true, 0, "")
	pdf.CellFormat(40, 10, fmt.Sprintf("%s %s", invoice.TotalAmount.StringFixed(2), invoice.Currency), "1", 1, "R", true, 0, "")

	// Reset colors
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(10)
}

// addFooter adds footer with payment information
func (pg *PDFGenerator) addFooter(pdf *gofpdf.Fpdf) {
	pdf.SetY(-30)
	pdf.SetFont("Arial", "I", 8)
	pdf.SetTextColor(128, 128, 128)

	pdf.Cell(0, 5, "Thank you for your business!")
	pdf.Ln(5)
	pdf.Cell(0, 5, fmt.Sprintf("Generated on: %s", time.Now().Format("2006-01-02 15:04:05")))
	pdf.Ln(5)
	pdf.Cell(0, 5, "For questions, please contact support@smsgateway.com")
}

// fileExists checks if a file exists
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
