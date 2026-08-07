package services

import (
	"context"
	"finance/internal/models"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type InvoiceService struct {
	db *gorm.DB
}

func NewInvoiceService(db *gorm.DB) *InvoiceService {
	return &InvoiceService{db: db}
}

// GenerateInvoice generates an invoice for a given journal entry, calculating VAT automatically
func (s *InvoiceService) GenerateInvoice(ctx context.Context, entryID uuid.UUID, customerName string, amount decimal.Decimal, vatRate decimal.Decimal) (*models.Invoice, error) {
	
	// Example calculation: SubTotal = Amount / (1 + VAT Rate)
	// VAT = Amount - SubTotal
	// If amount is 1150 and VAT is 0.15 (15%), Subtotal = 1000, VAT = 150
	
	onePlusVat := decimal.NewFromInt(1).Add(vatRate)
	subTotal := amount.Div(onePlusVat).Round(4)
	vatAmount := amount.Sub(subTotal).Round(4)

	invoice := &models.Invoice{
		InvoiceNumber:  fmt.Sprintf("INV-%d", time.Now().UnixNano()),
		JournalEntryID: entryID,
		CustomerName:   customerName,
		IssueDate:      time.Now(),
		SubTotal:       subTotal,
		VATAmount:      vatAmount,
		TotalAmount:    amount,
	}

	if err := s.db.WithContext(ctx).Create(invoice).Error; err != nil {
		return nil, fmt.Errorf("فشل توليد وحفظ الفاتورة: %v", err)
	}

	return invoice, nil
}

// GetInvoiceHTML returns a simple HTML string representing the invoice for frontend display/printing
func (s *InvoiceService) GetInvoiceHTML(ctx context.Context, invoiceID uuid.UUID) (string, error) {
	var inv models.Invoice
	if err := s.db.WithContext(ctx).Where("id = ?", invoiceID).First(&inv).Error; err != nil {
		return "", fmt.Errorf("الفاتورة غير موجودة: %v", err)
	}

	html := fmt.Sprintf(`
	<!DOCTYPE html>
	<html lang="ar" dir="rtl">
	<head>
		<meta charset="UTF-8">
		<title>فاتورة ضريبية</title>
		<style>
			body { font-family: Tahoma, Arial, sans-serif; padding: 20px; }
			.invoice-box { border: 1px solid #eee; padding: 30px; max-width: 800px; margin: auto; }
			table { width: 100%%; line-height: inherit; text-align: right; border-collapse: collapse; }
			table td { padding: 5px; vertical-align: top; }
			table tr.heading td { background: #eee; border-bottom: 1px solid #ddd; font-weight: bold; }
			table tr.item td { border-bottom: 1px solid #eee; }
			table tr.total td:nth-child(2) { border-top: 2px solid #eee; font-weight: bold; }
		</style>
	</head>
	<body>
		<div class="invoice-box">
			<h2>فاتورة ضريبية (Tax Invoice)</h2>
			<p>رقم الفاتورة: %s</p>
			<p>التاريخ: %s</p>
			<p>العميل: %s</p>
			<hr>
			<table>
				<tr class="heading">
					<td>الوصف</td>
					<td>القيمة (SAR)</td>
				</tr>
				<tr class="item">
					<td>إجمالي المبلغ قبل الضريبة</td>
					<td>%s</td>
				</tr>
				<tr class="item">
					<td>ضريبة القيمة المضافة (15%%)</td>
					<td>%s</td>
				</tr>
				<tr class="total">
					<td>الإجمالي المستحق (شامل الضريبة)</td>
					<td>%s</td>
				</tr>
			</table>
		</div>
	</body>
	</html>
	`, inv.InvoiceNumber, inv.IssueDate.Format("2006-01-02"), inv.CustomerName, inv.SubTotal.StringFixed(2), inv.VATAmount.StringFixed(2), inv.TotalAmount.StringFixed(2))

	return html, nil
}
