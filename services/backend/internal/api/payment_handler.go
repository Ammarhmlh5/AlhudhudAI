package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/al-hudhud/backend/internal/payment"
)

// PaymentManager defines the subset of Payment.Manager needed by the API handler.
type PaymentManager interface {
	ProcessPayment(ctx context.Context, req *payment.CreatePaymentRequest) (*payment.PaymentIntentResponse, error)
	HandleWebhookEvent(ctx context.Context, event *payment.WebhookEvent) error
	GetPaymentStatus(ctx context.Context, tenantID uuid.UUID, isSuperAdmin bool, transactionID string) (*payment.PaymentTransaction, error)
	ListPaymentTransactions(ctx context.Context, tenantID uuid.UUID, isSuperAdmin bool, limit, offset int) ([]*payment.PaymentTransaction, error)
}

// PayPalClient defines the subset of PayPal client methods used by the API handler.
type PayPalClient interface {
	CapturePayment(ctx context.Context, orderID string) (*payment.WebhookEvent, error)
	GetOrderDetails(ctx context.Context, orderID string) (*payment.PayPalOrder, error)
	VerifyWebhookSignature(ctx context.Context, transmissionID, transmissionTime, transmissionSig, certURL, authAlgo, webhookID string, body []byte) (bool, error)
}

// PaymentHandler handles payment-related HTTP requests
type PaymentHandler struct {
	paymentManager PaymentManager
	stripeClient   *payment.StripeClient
	paypalClient   PayPalClient
}

// NewPaymentHandler creates a new payment handler
func NewPaymentHandler(
	paymentManager PaymentManager,
	stripeClient *payment.StripeClient,
	paypalClient PayPalClient,
) *PaymentHandler {
	return &PaymentHandler{
		paymentManager: paymentManager,
		stripeClient:   stripeClient,
		paypalClient:   paypalClient,
	}
}

// InitiatePaymentRequest represents a request to initiate a payment
type InitiatePaymentRequest struct {
	Amount         float64                `json:"amount" binding:"required,gt=0"`
	Currency       string                 `json:"currency" binding:"required"`
	PaymentGateway payment.PaymentGateway `json:"payment_gateway" binding:"required"`
	Description    string                 `json:"description"`
}

// InitiatePayment initiates a payment
// Requirements: 39.4
// @Summary Initiate a payment
// @Description Initiates a payment with the specified gateway
// @Tags Payment
// @Accept json
// @Produce json
// @Param request body InitiatePaymentRequest true "Payment request"
// @Success 200 {object} payment.PaymentIntentResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/payments/initiate [post]
func (ph *PaymentHandler) InitiatePayment(c *gin.Context) {
	// Get tenant ID from context (stored as a string by AuthMiddleware)
	tenantID, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil || tenantID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "UNAUTHORIZED",
				Message: "Tenant ID not found in context",
			},
		})
		return
	}

	// Parse request
	var req InitiatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_REQUEST",
				Message: err.Error(),
			},
		})
		return
	}

	// Validate payment gateway
	if req.PaymentGateway != payment.GatewayStripe && req.PaymentGateway != payment.GatewayPayPal {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_GATEWAY",
				Message: "Unsupported payment gateway",
			},
		})
		return
	}

	// Create payment request
	paymentReq := &payment.CreatePaymentRequest{
		TenantID:       tenantID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		PaymentGateway: req.PaymentGateway,
		Description:    req.Description,
	}

	// Process payment
	response, err := ph.paymentManager.ProcessPayment(c.Request.Context(), paymentReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "PAYMENT_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// GetPaymentStatus retrieves the status of a payment
// Requirements: 39.4
// @Summary Get payment status
// @Description Retrieves the status of a payment transaction
// @Tags Payment
// @Produce json
// @Param transaction_id path string true "Transaction ID"
// @Success 200 {object} payment.PaymentTransaction
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/payments/{transaction_id} [get]
func (ph *PaymentHandler) GetPaymentStatus(c *gin.Context) {
	// Get tenant ID from context (stored as a string by AuthMiddleware)
	tenantID, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil || tenantID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "UNAUTHORIZED",
				Message: "Tenant ID not found in context",
			},
		})
		return
	}

	// Get transaction ID from URL
	transactionID := c.Param("transaction_id")
	if transactionID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_REQUEST",
				Message: "Transaction ID is required",
			},
		})
		return
	}

	// Get payment status
	isSuperAdmin := c.GetString("user_type") == "SUPER_ADMIN"
	transaction, err := ph.paymentManager.GetPaymentStatus(c.Request.Context(), tenantID, isSuperAdmin, transactionID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "NOT_FOUND",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    transaction,
	})
}

// ListPaymentTransactions lists payment transactions
// @Summary List payment transactions
// @Description Lists payment transactions for the authenticated tenant
// @Tags Payment
// @Produce json
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} payment.PaymentTransaction
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/payments [get]
func (ph *PaymentHandler) ListPaymentTransactions(c *gin.Context) {
	// Get tenant ID from context (stored as a string by AuthMiddleware)
	tenantID, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil || tenantID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "UNAUTHORIZED",
				Message: "Tenant ID not found in context",
			},
		})
		return
	}

	// Parse query parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// List transactions
	isSuperAdmin := c.GetString("user_type") == "SUPER_ADMIN"
	transactions, err := ph.paymentManager.ListPaymentTransactions(c.Request.Context(), tenantID, isSuperAdmin, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "QUERY_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    transactions,
		"meta": gin.H{
			"limit":  limit,
			"offset": offset,
			"count":  len(transactions),
		},
	})
}

// StripeWebhook handles Stripe webhook events
// Requirements: 39.5
// @Summary Stripe webhook
// @Description Handles webhook events from Stripe
// @Tags Payment
// @Accept json
// @Produce json
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/webhooks/stripe [post]
func (ph *PaymentHandler) StripeWebhook(c *gin.Context) {
	// Read request body
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_PAYLOAD",
				Message: "Failed to read request body",
			},
		})
		return
	}

	// Get Stripe signature
	signature := c.GetHeader("Stripe-Signature")
	if signature == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "MISSING_SIGNATURE",
				Message: "Stripe signature is required",
			},
		})
		return
	}

	// Handle webhook
	event, err := ph.stripeClient.HandleWebhook(c.Request.Context(), payload, signature)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "WEBHOOK_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	// Process webhook event
	if err := ph.paymentManager.HandleWebhookEvent(c.Request.Context(), event); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "PROCESSING_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Webhook processed successfully",
	})
}

// PayPalWebhook handles PayPal webhook events
// Requirements: 39.5
// @Summary PayPal webhook
// @Description Handles webhook events from PayPal
// @Tags Payment
// @Accept json
// @Produce json
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/webhooks/paypal [post]
func (ph *PaymentHandler) PayPalWebhook(c *gin.Context) {
	// Read raw body for verification
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_PAYLOAD",
				Message: "Failed to read request body",
			},
		})
		return
	}

	// Extract PayPal verification headers
	transmissionID := c.GetHeader("PayPal-Transmission-Id")
	transmissionTime := c.GetHeader("PayPal-Transmission-Time")
	transmissionSig := c.GetHeader("PayPal-Transmission-Sig")
	certURL := c.GetHeader("PayPal-Cert-Url")
	authAlgo := c.GetHeader("PayPal-Auth-Algo")

	// Allow skipping signature verification in development for local testing
	// and integration test scenarios where a webhook ID is not available.
	skipVerify := os.Getenv("PAYPAL_SKIP_WEBHOOK_VERIFY") == "true"

	webhookID := os.Getenv("PAYPAL_WEBHOOK_ID")
	if webhookID == "" && !skipVerify {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "MISSING_CONFIGURATION",
				Message: "PAYPAL_WEBHOOK_ID is not configured",
			},
		})
		return
	}

	var verified bool
	if skipVerify {
		verified = true
	} else {
		var verr error
		verified, verr = ph.paypalClient.VerifyWebhookSignature(c.Request.Context(), transmissionID, transmissionTime, transmissionSig, certURL, authAlgo, webhookID, body)
		if verr != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "WEBHOOK_VERIFY_FAILED",
					Message: verr.Error(),
				},
			})
			return
		}
	}
	
	if !verified {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "WEBHOOK_NOT_VERIFIED",
				Message: "Webhook signature could not be verified",
			},
		})
		return
	}

	// Parse webhook payload
	var webhookPayload map[string]interface{}
	if err := json.Unmarshal(body, &webhookPayload); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_PAYLOAD",
				Message: err.Error(),
			},
		})
		return
	}

	// Extract event type
	eventType, ok := webhookPayload["event_type"].(string)
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "MISSING_EVENT_TYPE",
				Message: "Event type is required",
			},
		})
		return
	}

	// Handle different event types
	switch eventType {
	case "PAYMENT.CAPTURE.COMPLETED", "CHECKOUT.ORDER.COMPLETED":
		resource, ok := webhookPayload["resource"].(map[string]interface{})
		if !ok {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "INVALID_RESOURCE",
					Message: "Resource is required",
				},
			})
			return
		}

		orderID, ok := resource["id"].(string)
		if !ok || orderID == "" {
			supplementaryData, _ := resource["supplementary_data"].(map[string]interface{})
			relatedIDs, _ := supplementaryData["related_ids"].(map[string]interface{})
			orderID, ok = relatedIDs["order_id"].(string)
		}
		if !ok || orderID == "" {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "MISSING_ORDER_ID",
					Message: "Order ID is required",
				},
			})
			return
		}

		order, err := ph.paypalClient.GetOrderDetails(c.Request.Context(), orderID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "ORDER_LOOKUP_FAILED",
					Message: err.Error(),
				},
			})
			return
		}

		transactionID := ""
		amount := 0.0
		currency := ""
		if len(order.PurchaseUnits) > 0 {
			transactionID = order.PurchaseUnits[0].ReferenceID
			currency = order.PurchaseUnits[0].Amount.CurrencyCode
			amount, _ = strconv.ParseFloat(order.PurchaseUnits[0].Amount.Value, 64)
		}
		if transactionID == "" {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "MISSING_TRANSACTION_ID",
					Message: "Transaction reference is required",
				},
			})
			return
		}

		status := payment.StatusFailed
		if eventType == "PAYMENT.CAPTURE.COMPLETED" || order.Status == "COMPLETED" {
			status = payment.StatusCompleted
		}

		event := &payment.WebhookEvent{
			Gateway:       payment.GatewayPayPal,
			EventType:     eventType,
			TransactionID: transactionID,
			Status:        status,
			Amount:        amount,
			Currency:      currency,
			Metadata: map[string]interface{}{
				"paypal_order_id": order.ID,
				"paypal_status":   order.Status,
			},
		}

		if err := ph.paymentManager.HandleWebhookEvent(c.Request.Context(), event); err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "PROCESSING_FAILED",
					Message: err.Error(),
				},
			})
			return
		}

	default:
		// Ignore other event types
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Event type not handled",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Webhook processed successfully",
	})
}

// PayPalCapturePayment captures a PayPal payment after user approval
// @Summary Capture PayPal payment
// @Description Captures a PayPal payment after user approval
// @Tags Payment
// @Produce json
// @Param order_id path string true "PayPal Order ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/payments/paypal/capture/{order_id} [post]
func (ph *PaymentHandler) PayPalCapturePayment(c *gin.Context) {
	// Get order ID from URL
	orderID := c.Param("order_id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_REQUEST",
				Message: "Order ID is required",
			},
		})
		return
	}

	// Capture payment
	event, err := ph.paypalClient.CapturePayment(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "CAPTURE_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	// Process webhook event
	if err := ph.paymentManager.HandleWebhookEvent(c.Request.Context(), event); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "PROCESSING_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Payment captured successfully",
		"data":    event,
	})
}

// SuccessResponse represents a successful API response
type SuccessResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ErrorResponse represents an error API response
type ErrorResponse struct {
	Success bool        `json:"success"`
	Error   ErrorDetail `json:"error"`
}

// ErrorDetail represents error details
type ErrorDetail struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}
