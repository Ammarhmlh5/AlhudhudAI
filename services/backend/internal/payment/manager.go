package payment

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// Manager handles payment operations
type Manager struct {
	db            *sqlx.DB
	logger        *zap.Logger
	stripeClient  *StripeClient
	paypalClient  *PayPalClient
	balanceTopup  BalanceTopupFunc
	notifier      NotificationFunc
}

// BalanceTopupFunc is a function type for adding balance
type BalanceTopupFunc func(ctx context.Context, tenantID uuid.UUID, amount float64, currency, reason string) error

// NotificationFunc is a function type for sending notifications
type NotificationFunc func(ctx context.Context, tenantID uuid.UUID, notificationType, message string) error

// NewManager creates a new payment manager
func NewManager(
	db *sqlx.DB,
	logger *zap.Logger,
	stripeClient *StripeClient,
	paypalClient *PayPalClient,
	balanceTopup BalanceTopupFunc,
	notifier NotificationFunc,
) *Manager {
	return &Manager{
		db:           db,
		logger:       logger,
		stripeClient: stripeClient,
		paypalClient: paypalClient,
		balanceTopup: balanceTopup,
		notifier:     notifier,
	}
}

// ProcessPayment processes a payment request
// Requirements: 39.3, 39.5
func (m *Manager) ProcessPayment(ctx context.Context, req *CreatePaymentRequest) (*PaymentIntentResponse, error) {
	// Validate request
	if req.Amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}

	if req.TenantID == uuid.Nil {
		return nil, fmt.Errorf("tenant_id is required")
	}

	// Check for existing transaction with same IdempotencyKey (Point 2.4)
	if req.IdempotencyKey != "" {
		var existingResponse PaymentIntentResponse
		var metadataJSON []byte
		var status string
		query := `
			SELECT transaction_id, status, metadata
			FROM payment_transactions
			WHERE tenant_id = $1 AND idempotency_key = $2
			LIMIT 1
		`
		err := m.db.QueryRowContext(ctx, query, req.TenantID.String(), req.IdempotencyKey).Scan(
			&existingResponse.TransactionID,
			&status,
			&metadataJSON,
		)
		if err == nil {
			existingResponse.Status = PaymentStatus(status)
			_ = json.Unmarshal(metadataJSON, &existingResponse.Metadata)
			m.logger.Info("Duplicate payment request detected (idempotency)",
				zap.String("tenant_id", req.TenantID.String()),
				zap.String("idempotency_key", req.IdempotencyKey),
				zap.String("transaction_id", existingResponse.TransactionID),
			)
			return &existingResponse, nil
		}
	}

	// Process payment based on gateway
	var response *PaymentIntentResponse
	var err error

	switch req.PaymentGateway {
	case GatewayStripe:
		response, err = m.stripeClient.CreatePaymentIntent(ctx, req)
	case GatewayPayPal:
		response, err = m.paypalClient.CreateOrder(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported payment gateway: %s", req.PaymentGateway)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to process payment: %w", err)
	}

	// Store transaction in database inside an RLS-enforced transaction
	// We pass isSuperAdmin=false here as this is a tenant-initiated action
	err = db.RunInTenantSqlxTx(ctx, req.TenantID, false, m.db, func(tx *sqlx.Tx) error {
		return m.storeTransactionTx(ctx, tx, req, response)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store transaction: %w", err)
	}

	return response, nil
}

// storeTransactionTx stores a payment transaction in the database
func (m *Manager) storeTransactionTx(ctx context.Context, tx *sqlx.Tx, req *CreatePaymentRequest, response *PaymentIntentResponse) error {
	// Prepare metadata
	metadataJSON, err := json.Marshal(response.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Insert transaction
	query := `
		INSERT INTO payment_transactions (
			tenant_id,
			transaction_id,
			payment_gateway,
			amount,
			currency,
			status,
			description,
			metadata,
			initiated_at,
			idempotency_key
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = tx.ExecContext(
		ctx,
		query,
		req.TenantID.String(),
		response.TransactionID,
		req.PaymentGateway,
		req.Amount,
		req.Currency,
		string(response.Status),
		req.Description,
		metadataJSON,
		time.Now(),
		req.IdempotencyKey,
	)

	if err != nil {
		return fmt.Errorf("failed to insert transaction: %w", err)
	}

	return nil
}

// HandleWebhookEvent handles webhook events from payment gateways
// Requirements: 39.5, 39.6
func (m *Manager) HandleWebhookEvent(ctx context.Context, event *WebhookEvent) error {
	// Guard against duplicate webhook processing (Point 2.4)
	var currentStatus string
	statusQuery := `SELECT status FROM payment_transactions WHERE transaction_id = $1`
	err := m.db.GetContext(ctx, &currentStatus, statusQuery, event.TransactionID)
	if err == nil && PaymentStatus(currentStatus) == StatusCompleted {
		m.logger.Info("Transaction already completed, skipping webhook processing",
			zap.String("transaction_id", event.TransactionID),
		)
		return nil
	}

	// Set tenant context (extract from transaction)
	tenantID, err := m.getTenantIDFromTransaction(ctx, event.TransactionID)
	if err != nil {
		return fmt.Errorf("failed to get tenant ID: %w", err)
	}

	// Update transaction status inside an RLS-enforced transaction
	// Webhooks operate on behalf of the tenant, so isSuperAdmin=false is appropriate
	err = db.RunInTenantSqlxTx(ctx, tenantID, false, m.db, func(tx *sqlx.Tx) error {
		return m.updateTransactionStatusTx(ctx, tx, event)
	})
	if err != nil {
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	// Handle completed payments
	if event.Status == StatusCompleted {
		if err := m.handleCompletedPayment(ctx, tenantID, event); err != nil {
			return fmt.Errorf("failed to handle completed payment: %w", err)
		}
	}

	// Handle failed payments
	if event.Status == StatusFailed {
		if err := m.handleFailedPayment(ctx, tenantID, event); err != nil {
			return fmt.Errorf("failed to handle failed payment: %w", err)
		}
	}

	return nil
}

// getTenantIDFromTransaction retrieves tenant ID from transaction
func (m *Manager) getTenantIDFromTransaction(ctx context.Context, transactionID string) (uuid.UUID, error) {
	var tenantID uuid.UUID
	query := `SELECT tenant_id FROM payment_transactions WHERE transaction_id = $1`
	
	err := m.db.GetContext(ctx, &tenantID, query, transactionID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get tenant ID: %w", err)
	}

	return tenantID, nil
}

// updateTransactionStatusTx updates the status of a payment transaction
func (m *Manager) updateTransactionStatusTx(ctx context.Context, tx *sqlx.Tx, event *WebhookEvent) error {
	// Prepare metadata
	metadataJSON, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Update query based on status
	var query string
	var args []interface{}

	if event.Status == StatusCompleted {
		query = `
			UPDATE payment_transactions
			SET 
				status = $1,
				completed_at = $2,
				metadata = $3,
				updated_at = $4
			WHERE transaction_id = $5
		`
		args = []interface{}{
			event.Status,
			time.Now(),
			metadataJSON,
			time.Now(),
			event.TransactionID,
		}
	} else if event.Status == StatusFailed {
		query = `
			UPDATE payment_transactions
			SET 
				status = $1,
				error_code = $2,
				error_message = $3,
				metadata = $4,
				updated_at = $5
			WHERE transaction_id = $6
		`
		args = []interface{}{
			event.Status,
			event.ErrorCode,
			event.ErrorMessage,
			metadataJSON,
			time.Now(),
			event.TransactionID,
		}
	} else {
		query = `
			UPDATE payment_transactions
			SET 
				status = $1,
				metadata = $2,
				updated_at = $3
			WHERE transaction_id = $4
		`
		args = []interface{}{
			event.Status,
			metadataJSON,
			time.Now(),
			event.TransactionID,
		}
	}

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	return nil
}

// handleCompletedPayment handles a completed payment
// Requirements: 39.5, 39.6, 39.7
func (m *Manager) handleCompletedPayment(ctx context.Context, tenantID uuid.UUID, event *WebhookEvent) error {
	// Auto top-up balance
	if err := m.AutoTopup(ctx, tenantID, event.Amount, event.Currency, event.TransactionID); err != nil {
		return fmt.Errorf("failed to auto top-up: %w", err)
	}

	// Send notification to customer
	message := fmt.Sprintf("تم شحن رصيدك بنجاح بمبلغ %.2f %s", event.Amount, event.Currency)
	if err := m.notifier(ctx, tenantID, "payment_success", message); err != nil {
		// Log error but don't fail the transaction
		logger.GetLogger().Error("failed to send notification", zap.Error(err))

	}

	return nil
}

// handleFailedPayment handles a failed payment
// Requirements: 39.7
func (m *Manager) handleFailedPayment(ctx context.Context, tenantID uuid.UUID, event *WebhookEvent) error {
	// Send notification to customer
	errorMsg := "فشلت عملية الدفع"
	if event.ErrorMessage != nil {
		errorMsg = *event.ErrorMessage
	}

	message := fmt.Sprintf("فشلت عملية الدفع: %s", errorMsg)
	if err := m.notifier(ctx, tenantID, "payment_failed", message); err != nil {
		// Log error but don't fail
		logger.GetLogger().Error("failed to send notification", zap.Error(err))

	}

	return nil
}

// AutoTopup automatically adds balance after successful payment
// Requirements: 39.5, 39.6
func (m *Manager) AutoTopup(ctx context.Context, tenantID uuid.UUID, amount float64, currency, transactionID string) error {
	// Call balance top-up function
	reason := fmt.Sprintf("شحن تلقائي من الدفع الإلكتروني - معاملة: %s", transactionID)
	
	if err := m.balanceTopup(ctx, tenantID, amount, currency, reason); err != nil {
		return fmt.Errorf("failed to top-up balance: %w", err)
	}

	return nil
}

// GetPaymentStatus retrieves the status of a payment
// Requirements: 39.4
func (m *Manager) GetPaymentStatus(ctx context.Context, tenantID uuid.UUID, isSuperAdmin bool, transactionID string) (*PaymentTransaction, error) {
	var txRecord PaymentTransaction

	errTx := db.RunInTenantSqlxTx(ctx, tenantID, isSuperAdmin, m.db, func(tx *sqlx.Tx) error {
		query := `
			SELECT 
				id, tenant_id, transaction_id, external_transaction_id,
				payment_gateway, payment_method, amount, currency, status,
				description, metadata, refund_amount, refund_reason, refunded_at,
				error_code, error_message, initiated_at, completed_at,
				created_at, updated_at
			FROM payment_transactions
			WHERE transaction_id = $1
		`
		return tx.GetContext(ctx, &txRecord, query, transactionID)
	})

	if errTx == sql.ErrNoRows {
		return nil, fmt.Errorf("transaction not found")
	}
	if errTx != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", errTx)
	}

	return &txRecord, nil
}

// ListPaymentTransactions lists payment transactions for a tenant
func (m *Manager) ListPaymentTransactions(ctx context.Context, tenantID uuid.UUID, isSuperAdmin bool, limit, offset int) ([]*PaymentTransaction, error) {
	var transactions []*PaymentTransaction

	err := db.RunInTenantSqlxTx(ctx, tenantID, isSuperAdmin, m.db, func(tx *sqlx.Tx) error {
		query := `
			SELECT 
				id, tenant_id, transaction_id, external_transaction_id,
				payment_gateway, payment_method, amount, currency, status,
				description, metadata, refund_amount, refund_reason, refunded_at,
				error_code, error_message, initiated_at, completed_at,
				created_at, updated_at
			FROM payment_transactions
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2
		`
		return tx.SelectContext(ctx, &transactions, query, limit, offset)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}

	return transactions, nil
}
