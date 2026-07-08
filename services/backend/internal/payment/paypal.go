package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// PayPalClient handles PayPal payment integration
type PayPalClient struct {
	clientID     string
	clientSecret string
	baseURL      string
	httpClient   *http.Client
}

// PayPalAccessToken represents PayPal OAuth token
type PayPalAccessToken struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// PayPalOrder represents a PayPal order
type PayPalOrder struct {
	ID            string                 `json:"id"`
	Status        string                 `json:"status"`
	PurchaseUnits []PayPalPurchaseUnit   `json:"purchase_units"`
	Links         []PayPalLink           `json:"links"`
	CreateTime    string                 `json:"create_time"`
	UpdateTime    string                 `json:"update_time"`
}

// PayPalPurchaseUnit represents a purchase unit in PayPal order
type PayPalPurchaseUnit struct {
	ReferenceID string          `json:"reference_id"`
	Amount      PayPalAmount    `json:"amount"`
	Description string          `json:"description,omitempty"`
	CustomID    string          `json:"custom_id,omitempty"`
}

// PayPalAmount represents amount in PayPal
type PayPalAmount struct {
	CurrencyCode string `json:"currency_code"`
	Value        string `json:"value"`
}

// PayPalLink represents a HATEOAS link
type PayPalLink struct {
	Href   string `json:"href"`
	Rel    string `json:"rel"`
	Method string `json:"method"`
}

// NewPayPalClient creates a new PayPal client
func NewPayPalClient(clientID, clientSecret string, sandbox bool) *PayPalClient {
	baseURL := "https://api-m.paypal.com"
	if sandbox {
		baseURL = "https://api-m.sandbox.paypal.com"
	}

	return &PayPalClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		baseURL:      baseURL,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

// getAccessToken retrieves an OAuth access token from PayPal
func (pc *PayPalClient) getAccessToken(ctx context.Context) (*PayPalAccessToken, error) {
	url := fmt.Sprintf("%s/v1/oauth2/token", pc.baseURL)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBufferString("grant_type=client_credentials"))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(pc.clientID, pc.clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := pc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get access token: status %d, body: %s", resp.StatusCode, string(body))
	}

	var token PayPalAccessToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("failed to decode access token: %w", err)
	}

	return &token, nil
}

// CreateOrder creates a PayPal order
// Requirements: 39.2
func (pc *PayPalClient) CreateOrder(ctx context.Context, req *CreatePaymentRequest) (*PaymentIntentResponse, error) {
	// Get access token
	token, err := pc.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	// Generate transaction ID
	transactionID := fmt.Sprintf("TXN-%s", uuid.New().String())

	// Prepare order request
	orderReq := map[string]interface{}{
		"intent": "CAPTURE",
		"purchase_units": []map[string]interface{}{
			{
				"reference_id": transactionID,
				"description":  req.Description,
				"custom_id":    req.TenantID.String(),
				"amount": map[string]string{
					"currency_code": req.Currency,
					"value":         fmt.Sprintf("%.2f", req.Amount),
				},
			},
		},
		"application_context": map[string]interface{}{
			"brand_name":          "SMS Gateway",
			"landing_page":        "BILLING",
			"user_action":         "PAY_NOW",
			"return_url":          "https://your-domain.com/payment/success",
			"cancel_url":          "https://your-domain.com/payment/cancel",
		},
	}

	// Marshal request
	body, err := json.Marshal(orderReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal order request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/v2/checkout/orders", pc.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))

	// Add Idempotency Key if provided (Requirement 2.4)
	if req.IdempotencyKey != "" {
		httpReq.Header.Set("PayPal-Request-Id", req.IdempotencyKey)
	}

	// Send request
	resp, err := pc.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create order: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var order PayPalOrder
	if err := json.NewDecoder(resp.Body).Decode(&order); err != nil {
		return nil, fmt.Errorf("failed to decode order response: %w", err)
	}

	// Find approval URL
	var approvalURL string
	for _, link := range order.Links {
		if link.Rel == "approve" {
			approvalURL = link.Href
			break
		}
	}

	// Prepare response
	response := &PaymentIntentResponse{
		TransactionID: transactionID,
		PaymentURL:    approvalURL,
		Status:        StatusPending,
		Metadata: map[string]interface{}{
			"paypal_order_id": order.ID,
			"paypal_status":   order.Status,
		},
	}

	return response, nil
}

// CapturePayment captures a PayPal payment after user approval
// Requirements: 39.2
func (pc *PayPalClient) CapturePayment(ctx context.Context, orderID string) (*WebhookEvent, error) {
	// Get access token
	token, err := pc.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/v2/checkout/orders/%s/capture", pc.baseURL, orderID)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))

	// Send request
	resp, err := pc.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to capture payment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to capture payment: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var order PayPalOrder
	if err := json.NewDecoder(resp.Body).Decode(&order); err != nil {
		return nil, fmt.Errorf("failed to decode capture response: %w", err)
	}

	// Extract transaction details
	var transactionID string
	var amount float64
	var currency string

	if len(order.PurchaseUnits) > 0 {
		unit := order.PurchaseUnits[0]
		transactionID = unit.ReferenceID
		currency = unit.Amount.CurrencyCode
		
		// Parse amount
		fmt.Sscanf(unit.Amount.Value, "%f", &amount)
	}

	// Determine status
	var status PaymentStatus
	switch order.Status {
	case "COMPLETED":
		status = StatusCompleted
	case "APPROVED":
		status = StatusProcessing
	default:
		status = StatusFailed
	}

	// Prepare webhook event
	webhookEvent := &WebhookEvent{
		Gateway:       GatewayPayPal,
		EventType:     "order.captured",
		TransactionID: transactionID,
		Status:        status,
		Amount:        amount,
		Currency:      currency,
		Metadata: map[string]interface{}{
			"paypal_order_id": order.ID,
			"paypal_status":   order.Status,
		},
	}

	return webhookEvent, nil
}

// GetOrderDetails retrieves PayPal order details
func (pc *PayPalClient) GetOrderDetails(ctx context.Context, orderID string) (*PayPalOrder, error) {
	// Get access token
	token, err := pc.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/v2/checkout/orders/%s", pc.baseURL, orderID)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))

	// Send request
	resp, err := pc.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get order details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get order details: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var order PayPalOrder
	if err := json.NewDecoder(resp.Body).Decode(&order); err != nil {
		return nil, fmt.Errorf("failed to decode order response: %w", err)
	}

	return &order, nil
}
