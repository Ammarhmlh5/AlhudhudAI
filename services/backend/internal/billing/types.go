package billing

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Invoice represents a billing invoice.
type Invoice struct {
	ID            uuid.UUID       `json:"id"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	InvoiceNumber string          `json:"invoice_number"`
	TotalAmount   decimal.Decimal `json:"total_amount"`
	Status        string          `json:"status"`
	DueDate       time.Time       `json:"due_date,omitempty"`
}

// PaymentTransaction represents a payment transaction.
type PaymentTransaction struct {
	ID            uuid.UUID       `json:"id"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	Amount        decimal.Decimal `json:"amount"`
	PaymentMethod string          `json:"payment_method"`
	Status        string          `json:"status"`
}

// PromoCode represents a promotional code.
type PromoCode struct {
	Code      string          `json:"code"`
	Discount  int             `json:"discount"`
	Type      string          `json:"type"`
	ExpiresAt time.Time       `json:"expires_at"`
}

// UsageRecord represents a usage record for billing.
type UsageRecord struct {
	Service   string          `json:"service"`
	Quantity  int             `json:"quantity"`
	UnitPrice decimal.Decimal `json:"unit_price"`
}

var (
	ErrPromoCodeExpired = errors.New("promo code has expired")
)
