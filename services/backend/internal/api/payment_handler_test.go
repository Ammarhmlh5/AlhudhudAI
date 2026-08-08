package api

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/http/httptest"
    "os"
    "testing"

    "github.com/al-hudhud/backend/internal/payment"
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/stretchr/testify/require"
)

func TestPayPalWebhook_Handler_SkipsVerify_EventNotHandled(t *testing.T) {
    // Enable skip verification so test does not call PayPal
    os.Setenv("PAYPAL_SKIP_WEBHOOK_VERIFY", "true")
    defer os.Unsetenv("PAYPAL_SKIP_WEBHOOK_VERIFY")

    // Provide webhook ID so handler does not fail before verification check
    os.Setenv("PAYPAL_WEBHOOK_ID", "test-webhook-id")
    defer os.Unsetenv("PAYPAL_WEBHOOK_ID")

    // Create a PaymentHandler with nil clients — skip verification prevents nil deref
    ph := NewPaymentHandler(nil, nil, nil)

    r := gin.New()
    r.POST("/api/v1/webhooks/paypal", ph.PayPalWebhook)

    payload := map[string]interface{}{
        "event_type": "OTHER.EVENT",
    }
    body, _ := json.Marshal(payload)

    req := httptest.NewRequest("POST", "/api/v1/webhooks/paypal", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    require.Equal(t, http.StatusOK, w.Code)
}

func TestPayPalWebhook_Handler_SkipsVerify_WithoutWebhookID(t *testing.T) {
    // Enable skip verification so handler can accept requests without a configured webhook ID.
    os.Setenv("PAYPAL_SKIP_WEBHOOK_VERIFY", "true")
    defer os.Unsetenv("PAYPAL_SKIP_WEBHOOK_VERIFY")

    // Ensure the webhook ID is not configured in this mode.
    os.Unsetenv("PAYPAL_WEBHOOK_ID")

    ph := NewPaymentHandler(nil, nil, nil)

    r := gin.New()
    r.POST("/api/v1/webhooks/paypal", ph.PayPalWebhook)

    payload := map[string]interface{}{
        "event_type": "OTHER.EVENT",
    }
    body, _ := json.Marshal(payload)

    req := httptest.NewRequest("POST", "/api/v1/webhooks/paypal", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    require.Equal(t, http.StatusOK, w.Code)
}


type mockPaymentManager struct{
    handleWebhookFunc func(ctx context.Context, event *payment.WebhookEvent) error
}

func (m *mockPaymentManager) ProcessPayment(ctx context.Context, req *payment.CreatePaymentRequest) (*payment.PaymentIntentResponse, error) {
    return nil, nil
}

func (m *mockPaymentManager) HandleWebhookEvent(ctx context.Context, event *payment.WebhookEvent) error {
    if m.handleWebhookFunc != nil {
        return m.handleWebhookFunc(ctx, event)
    }
    return nil
}

func (m *mockPaymentManager) GetPaymentStatus(ctx context.Context, tenantID uuid.UUID, isSuperAdmin bool, transactionID string) (*payment.PaymentTransaction, error) {
    return nil, nil
}

func (m *mockPaymentManager) ListPaymentTransactions(ctx context.Context, tenantID uuid.UUID, isSuperAdmin bool, limit, offset int) ([]*payment.PaymentTransaction, error) {
    return nil, nil
}

type mockPayPalClient struct{}

func (m *mockPayPalClient) CapturePayment(ctx context.Context, orderID string) (*payment.WebhookEvent, error) {
    return &payment.WebhookEvent{
        Gateway:       payment.GatewayPayPal,
        EventType:     "order.captured",
        TransactionID: "TXN-123",
        Status:        payment.StatusCompleted,
        Amount:        10.0,
        Currency:      "USD",
        Metadata: map[string]interface{}{
            "paypal_order_id": orderID,
        },
    }, nil
}

func (m *mockPayPalClient) GetOrderDetails(ctx context.Context, orderID string) (*payment.PayPalOrder, error) {
    return &payment.PayPalOrder{
        ID:     orderID,
        Status: "COMPLETED",
        PurchaseUnits: []payment.PayPalPurchaseUnit{
            {
                ReferenceID: "TXN-123",
                Amount: payment.PayPalAmount{
                    CurrencyCode: "USD",
                    Value:        "10.00",
                },
            },
        },
    }, nil
}

func (m *mockPayPalClient) VerifyWebhookSignature(ctx context.Context, transmissionID, transmissionTime, transmissionSig, certURL, authAlgo, webhookID string, body []byte) (bool, error) {
    return true, nil
}

func TestPayPalWebhook_Handler_PaymentCaptureCompleted(t *testing.T) {
    os.Setenv("PAYPAL_SKIP_WEBHOOK_VERIFY", "true")
    defer os.Unsetenv("PAYPAL_SKIP_WEBHOOK_VERIFY")

    called := false
    mgr := &mockPaymentManager{
        handleWebhookFunc: func(ctx context.Context, event *payment.WebhookEvent) error {
            called = true
            if event.TransactionID != "TXN-123" {
                return fmt.Errorf("unexpected transaction id: %s", event.TransactionID)
            }
            if event.Status != payment.StatusCompleted {
                return fmt.Errorf("unexpected status: %v", event.Status)
            }
            return nil
        },
    }

    ph := NewPaymentHandler(mgr, nil, &mockPayPalClient{})

    r := gin.New()
    r.POST("/api/v1/webhooks/paypal", ph.PayPalWebhook)

    payload := map[string]interface{}{
        "event_type": "PAYMENT.CAPTURE.COMPLETED",
        "resource": map[string]interface{}{
            "supplementary_data": map[string]interface{}{
                "related_ids": map[string]interface{}{
                    "order_id": "ORDER-123",
                },
            },
        },
    }
    body, _ := json.Marshal(payload)

    req := httptest.NewRequest("POST", "/api/v1/webhooks/paypal", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if !called {
        t.Fatalf("expected HandleWebhookEvent to be called")
    }
    require.Equal(t, http.StatusOK, w.Code)
}

func TestPayPalCapturePayment_Success(t *testing.T) {
    ph := NewPaymentHandler(&mockPaymentManager{}, nil, &mockPayPalClient{})

    r := gin.New()
    r.POST("/api/v1/payments/paypal/capture/:order_id", ph.PayPalCapturePayment)

    req := httptest.NewRequest("POST", "/api/v1/payments/paypal/capture/ORDER-123", nil)
    req.Header.Set("Content-Type", "application/json")

    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    require.Equal(t, http.StatusOK, w.Code)
    require.Contains(t, w.Body.String(), "Payment captured successfully")
}
