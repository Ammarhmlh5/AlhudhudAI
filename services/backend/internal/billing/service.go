package billing

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type BillingRepository interface {
	CreateInvoice(ctx context.Context, invoice *Invoice) error
	GetInvoiceByID(ctx context.Context, invoiceID uuid.UUID) (*Invoice, error)
	GetInvoicesByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]Invoice, error)
	UpdateInvoiceStatus(ctx context.Context, invoiceID uuid.UUID, status string) error
	CreatePaymentTransaction(ctx context.Context, payment *PaymentTransaction) error
	GetPaymentByID(ctx context.Context, paymentID uuid.UUID) (*PaymentTransaction, error)
	ValidatePromoCode(ctx context.Context, code string) (*PromoCode, error)
}

type TenantRepository interface {
	GetTenantByID(ctx context.Context, tenantID uuid.UUID) (*Tenant, error)
}

type Service struct {
	repo     BillingRepository
	tenantRepo TenantRepository
	invoiceNum int
}

func NewService(repo BillingRepository) *Service {
	return &Service{
		repo:     repo,
		invoiceNum: 0,
	}
}

func (s *Service) SetTenantRepository(repo TenantRepository) {
	s.tenantRepo = repo
}

func (s *Service) CreateInvoice(ctx context.Context, invoice *Invoice) (*Invoice, error) {
	if invoice.ID == uuid.Nil {
		invoice.ID = uuid.New()
	}
	if err := s.repo.CreateInvoice(ctx, invoice); err != nil {
		return nil, err
	}
	return invoice, nil
}

func (s *Service) GenerateMonthlyInvoice(ctx context.Context, tenantID uuid.UUID, billingDate time.Time, usage []UsageRecord) (*Invoice, error) {
	var totalAmount decimal.Decimal
	for _, u := range usage {
		amount := u.UnitPrice.Mul(decimal.NewFromInt(int64(u.Quantity)))
		totalAmount = totalAmount.Add(amount)
	}

	invoice := &Invoice{
		ID:            uuid.New(),
		TenantID:      tenantID,
		InvoiceNumber: fmt.Sprintf("INV-%s-%d", billingDate.Format("200601"), s.invoiceNum),
		TotalAmount:   totalAmount,
		Status:        "DRAFT",
		DueDate:       billingDate.Add(30 * 24 * time.Hour),
	}
	s.invoiceNum++

	if err := s.repo.CreateInvoice(ctx, invoice); err != nil {
		return nil, err
	}
	return invoice, nil
}

func (s *Service) ProcessPayment(ctx context.Context, payment *PaymentTransaction) (*PaymentTransaction, error) {
	if payment.ID == uuid.Nil {
		payment.ID = uuid.New()
	}
	payment.Status = "COMPLETED"
	if err := s.repo.CreatePaymentTransaction(ctx, payment); err != nil {
		return nil, err
	}
	return payment, nil
}

func (s *Service) GetInvoiceHistory(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]Invoice, error) {
	return s.repo.GetInvoicesByTenant(ctx, tenantID, limit, offset)
}

func (s *Service) ApplyPromoCode(ctx context.Context, code string, originalAmount decimal.Decimal) (decimal.Decimal, error) {
	promo, err := s.repo.ValidatePromoCode(ctx, code)
	if err != nil {
		return originalAmount, err
	}

	if time.Now().After(promo.ExpiresAt) {
		return originalAmount, ErrPromoCodeExpired
	}

	discountFactor := decimal.NewFromInt(100).Sub(decimal.NewFromInt(int64(promo.Discount))).Div(decimal.NewFromInt(100))
	return originalAmount.Mul(discountFactor), nil
}

func (s *Service) GenerateInvoiceNumber() string {
	s.invoiceNum++
	return fmt.Sprintf("INV-%d", s.invoiceNum)
}

func (s *Service) CalculateLateFee(amount decimal.Decimal, daysOverdue int) (decimal.Decimal, error) {
	rate := decimal.NewFromFloat(0.01)
	fee := amount.Mul(rate).Mul(decimal.NewFromInt(int64(daysOverdue)))
	return fee, nil
}

func (s *Service) RefundPayment(ctx context.Context, paymentID uuid.UUID, amount decimal.Decimal, reason string) error {
	payment, err := s.repo.GetPaymentByID(ctx, paymentID)
	if err != nil {
		return err
	}
	_ = payment
	return s.repo.UpdateInvoiceStatus(ctx, paymentID, "REFUNDED")
}

// Tenant is a minimal tenant type used by the billing service.
type Tenant struct {
	ID      uuid.UUID       `json:"id"`
	Name    string          `json:"name"`
	Email   string          `json:"email"`
	Balance decimal.Decimal `json:"balance"`
}

