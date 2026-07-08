package payment

import (
	"time"

	"github.com/google/uuid"
)

// PaymentGateway represents supported payment gateways
type PaymentGateway string

const (
	GatewayStripe PaymentGateway = "stripe"
	GatewayPayPal PaymentGateway = "paypal"
	GatewayLocal  PaymentGateway = "local"
)

// PaymentStatus represents payment transaction status
type PaymentStatus string

const (
	StatusPending    PaymentStatus = "pending"
	StatusProcessing PaymentStatus = "processing"
	StatusCompleted  PaymentStatus = "completed"
	StatusFailed     PaymentStatus = "failed"
	StatusRefunded   PaymentStatus = "refunded"
)

// PaymentMethod represents payment methods
type PaymentMethod string

const (
	MethodCard         PaymentMethod = "card"
	MethodBankTransfer PaymentMethod = "bank_transfer"
	MethodWallet       PaymentMethod = "wallet"
	MethodPayPal       PaymentMethod = "paypal_account"
)

// PaymentTransaction represents a payment transaction
type PaymentTransaction struct {
	ID                    int               `json:"id" db:"id"`
	TenantID              uuid.UUID         `json:"tenant_id" db:"tenant_id"`
	TransactionID         string            `json:"transaction_id" db:"transaction_id"`
	ExternalTransactionID *string           `json:"external_transaction_id,omitempty" db:"external_transaction_id"`
	PaymentGateway        PaymentGateway    `json:"payment_gateway" db:"payment_gateway"`
	PaymentMethod         *PaymentMethod    `json:"payment_method,omitempty" db:"payment_method"`
	Amount                float64           `json:"amount" db:"amount"`
	Currency              string            `json:"currency" db:"currency"`
	Status                PaymentStatus     `json:"status" db:"status"`
	Description           *string           `json:"description,omitempty" db:"description"`
	Metadata              map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	RefundAmount          *float64          `json:"refund_amount,omitempty" db:"refund_amount"`
	RefundReason          *string           `json:"refund_reason,omitempty" db:"refund_reason"`
	RefundedAt            *time.Time        `json:"refunded_at,omitempty" db:"refunded_at"`
	ErrorCode             *string           `json:"error_code,omitempty" db:"error_code"`
	ErrorMessage          *string           `json:"error_message,omitempty" db:"error_message"`
	InitiatedAt           time.Time         `json:"initiated_at" db:"initiated_at"`
	CompletedAt           *time.Time        `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt             time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time         `json:"updated_at" db:"updated_at"`
}

// CreatePaymentRequest represents a request to create a payment
type CreatePaymentRequest struct {
	TenantID       uuid.UUID      `json:"tenant_id"`
	Amount         float64        `json:"amount"`
	Currency       string         `json:"currency"`
	PaymentGateway PaymentGateway `json:"payment_gateway"`
	Description    string         `json:"description,omitempty"`
	IdempotencyKey string         `json:"idempotency_key,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// PaymentIntentResponse represents a payment intent response
type PaymentIntentResponse struct {
	TransactionID string                 `json:"transaction_id"`
	ClientSecret  string                 `json:"client_secret,omitempty"`
	PaymentURL    string                 `json:"payment_url,omitempty"`
	Status        PaymentStatus          `json:"status"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// RefundRequest represents a refund request
type RefundRequest struct {
	TransactionID string  `json:"transaction_id"`
	Amount        float64 `json:"amount"`
	Reason        string  `json:"reason"`
}

// RefundResponse represents a refund response
type RefundResponse struct {
	TransactionID string        `json:"transaction_id"`
	RefundAmount  float64       `json:"refund_amount"`
	Status        PaymentStatus `json:"status"`
	RefundedAt    time.Time     `json:"refunded_at"`
}

// WebhookEvent represents a webhook event from payment gateway
type WebhookEvent struct {
	Gateway       PaymentGateway         `json:"gateway"`
	EventType     string                 `json:"event_type"`
	TransactionID string                 `json:"transaction_id"`
	Status        PaymentStatus          `json:"status"`
	Amount        float64                `json:"amount"`
	Currency      string                 `json:"currency"`
	Metadata      map[string]interface{} `json:"metadata"`
	ErrorCode     *string                `json:"error_code,omitempty"`
	ErrorMessage  *string                `json:"error_message,omitempty"`
}
