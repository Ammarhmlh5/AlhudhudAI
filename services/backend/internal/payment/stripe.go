package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/paymentintent"
	"github.com/stripe/stripe-go/v76/refund"
	"github.com/stripe/stripe-go/v76/webhook"
)

// StripeClient handles Stripe payment integration
type StripeClient struct {
	apiKey        string
	webhookSecret string
}

// NewStripeClient creates a new Stripe client
func NewStripeClient(apiKey, webhookSecret string) *StripeClient {
	stripe.Key = apiKey
	return &StripeClient{
		apiKey:        apiKey,
		webhookSecret: webhookSecret,
	}
}

// CreatePaymentIntent creates a Stripe payment intent
// Requirements: 39.2, 39.4
func (sc *StripeClient) CreatePaymentIntent(ctx context.Context, req *CreatePaymentRequest) (*PaymentIntentResponse, error) {
	// Generate transaction ID
	transactionID := fmt.Sprintf("TXN-%s", uuid.New().String())

	// Convert amount to cents (Stripe uses smallest currency unit)
	amountInCents := int64(req.Amount * 100)

	// Prepare metadata
	metadata := map[string]string{
		"tenant_id":      req.TenantID.String(),
		"transaction_id": transactionID,
	}
	
	// Add custom metadata
	for k, v := range req.Metadata {
		if str, ok := v.(string); ok {
			metadata[k] = str
		}
	}

	// Create payment intent parameters
	params := &stripe.PaymentIntentParams{
		Amount:      stripe.Int64(amountInCents),
		Currency:    stripe.String(req.Currency),
		Description: stripe.String(req.Description),
		Metadata:    metadata,
		// Enable automatic payment methods
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
	}

	// Add Idempotency Key if provided (Requirement 2.4)
	if req.IdempotencyKey != "" {
		params.IdempotencyKey = stripe.String(req.IdempotencyKey)
	}

	// Create payment intent
	pi, err := paymentintent.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment intent: %w", err)
	}

	// Prepare response
	response := &PaymentIntentResponse{
		TransactionID: transactionID,
		ClientSecret:  pi.ClientSecret,
		Status:        StatusPending,
		Metadata: map[string]interface{}{
			"payment_intent_id": pi.ID,
			"stripe_status":     pi.Status,
		},
	}

	return response, nil
}

// HandleWebhook processes Stripe webhook events
// Requirements: 39.5
func (sc *StripeClient) HandleWebhook(ctx context.Context, payload []byte, signature string) (*WebhookEvent, error) {
	// Verify webhook signature
	event, err := webhook.ConstructEvent(payload, signature, sc.webhookSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to verify webhook signature: %w", err)
	}

	// Parse webhook event
	webhookEvent := &WebhookEvent{
		Gateway:   GatewayStripe,
		EventType: string(event.Type),
	}

	// Handle different event types
	switch event.Type {
	case "payment_intent.succeeded":
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			return nil, fmt.Errorf("failed to parse payment intent: %w", err)
		}

		webhookEvent.TransactionID = pi.Metadata["transaction_id"]
		webhookEvent.Status = StatusCompleted
		webhookEvent.Amount = float64(pi.Amount) / 100
		webhookEvent.Currency = string(pi.Currency)
		webhookEvent.Metadata = map[string]interface{}{
			"payment_intent_id": pi.ID,
			"payment_method":    pi.PaymentMethod,
		}

	case "payment_intent.payment_failed":
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			return nil, fmt.Errorf("failed to parse payment intent: %w", err)
		}

		webhookEvent.TransactionID = pi.Metadata["transaction_id"]
		webhookEvent.Status = StatusFailed
		webhookEvent.Amount = float64(pi.Amount) / 100
		webhookEvent.Currency = string(pi.Currency)

		// Extract error information
		if pi.LastPaymentError != nil {
			errorCode := string(pi.LastPaymentError.Code)
			errorMessage := pi.LastPaymentError.Msg
			webhookEvent.ErrorCode = &errorCode
			webhookEvent.ErrorMessage = &errorMessage
		}

	case "payment_intent.processing":
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			return nil, fmt.Errorf("failed to parse payment intent: %w", err)
		}

		webhookEvent.TransactionID = pi.Metadata["transaction_id"]
		webhookEvent.Status = StatusProcessing
		webhookEvent.Amount = float64(pi.Amount) / 100
		webhookEvent.Currency = string(pi.Currency)

	case "charge.refunded":
		var charge stripe.Charge
		if err := json.Unmarshal(event.Data.Raw, &charge); err != nil {
			return nil, fmt.Errorf("failed to parse charge: %w", err)
		}

		webhookEvent.TransactionID = charge.Metadata["transaction_id"]
		webhookEvent.Status = StatusRefunded
		webhookEvent.Amount = float64(charge.AmountRefunded) / 100
		webhookEvent.Currency = string(charge.Currency)

	default:
		// Ignore other event types
		return nil, fmt.Errorf("unhandled event type: %s", event.Type)
	}

	return webhookEvent, nil
}

// ProcessRefund processes a refund for a Stripe payment
// Requirements: 39.6, 39.10
func (sc *StripeClient) ProcessRefund(ctx context.Context, req *RefundRequest) (*RefundResponse, error) {
	// Get payment intent ID from metadata
	// In a real implementation, you would retrieve this from the database
	// For now, we'll assume it's passed in the metadata
	
	// Convert amount to cents
	amountInCents := int64(req.Amount * 100)

	// Create refund parameters
	params := &stripe.RefundParams{
		Amount: stripe.Int64(amountInCents),
		Reason: stripe.String(req.Reason),
		Metadata: map[string]string{
			"transaction_id": req.TransactionID,
			"refund_reason":  req.Reason,
		},
	}

	// Note: In production, you need to get the payment_intent_id from the database
	// params.PaymentIntent = stripe.String(paymentIntentID)

	// Create refund
	r, err := refund.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create refund: %w", err)
	}

	// Prepare response
	response := &RefundResponse{
		TransactionID: req.TransactionID,
		RefundAmount:  float64(r.Amount) / 100,
		Status:        StatusRefunded,
		RefundedAt:    time.Now(),
	}

	return response, nil
}

// GetPaymentStatus retrieves the status of a payment from Stripe
func (sc *StripeClient) GetPaymentStatus(ctx context.Context, paymentIntentID string) (PaymentStatus, error) {
	pi, err := paymentintent.Get(paymentIntentID, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get payment intent: %w", err)
	}

	// Map Stripe status to our status
	switch pi.Status {
	case stripe.PaymentIntentStatusSucceeded:
		return StatusCompleted, nil
	case stripe.PaymentIntentStatusProcessing:
		return StatusProcessing, nil
	case stripe.PaymentIntentStatusRequiresPaymentMethod,
		stripe.PaymentIntentStatusRequiresConfirmation,
		stripe.PaymentIntentStatusRequiresAction:
		return StatusPending, nil
	case stripe.PaymentIntentStatusCanceled:
		return StatusFailed, nil
	default:
		return StatusPending, nil
	}
}
