package balance

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/al-hudhud/backend/internal/audit"
	"github.com/al-hudhud/backend/internal/metrics"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type Manager interface {
	// DeductBalance deducts balance atomically with race condition protection
	// Returns error if balance insufficient
	DeductBalance(ctx context.Context, op BalanceOperation) (*BalanceResult, error)

	// DeductBalanceTx deducts balance within an existing transaction
	DeductBalanceTx(ctx context.Context, tx pgx.Tx, op BalanceOperation) (*BalanceResult, error)

	// DeductBalanceWithTimeout deducts balance with a timeout context
	// Implements Requirements 27.1, 27.2
	DeductBalanceWithTimeout(ctx context.Context, op BalanceOperation, timeout time.Duration) (*BalanceResult, error)

	// RefundBalance refunds balance atomically
	RefundBalance(ctx context.Context, op BalanceOperation) (*BalanceResult, error)

	// RefundBalanceTx refunds balance within an existing transaction
	RefundBalanceTx(ctx context.Context, tx pgx.Tx, op BalanceOperation) (*BalanceResult, error)

	// GetBalance returns current balance for tenant
	GetBalance(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error)

	// CheckSufficient checks if balance is sufficient for the amount
	CheckSufficient(ctx context.Context, tenantID uuid.UUID, amount decimal.Decimal) (bool, error)

	// CreditBalance credits balance to a tenant atomically (e.g. admin funding a wallet)
	CreditBalance(ctx context.Context, op BalanceOperation) (*BalanceResult, error)

	// TransferBalance moves funds from one tenant wallet to another atomically
	TransferBalance(ctx context.Context, from, to uuid.UUID, amount decimal.Decimal, reason, createdBy, ipAddress string) (*TransferResult, error)
}

// manager implements the Manager interface
type manager struct {
	pool                *pgxpool.Pool
	logger              *zap.Logger
	historyLogger       HistoryLogger
	notificationService NotificationService
	auditLogger         audit.AuditLogger
}

// NewManager creates a new balance manager
func NewManager(pool *pgxpool.Pool, logger *zap.Logger) Manager {
	// Get standard database connection for audit logger
	db := pool.Config().ConnConfig.Database
	// Build the URL via net/url so the password (which may contain reserved
	// characters like # or @) is properly percent-encoded.
	dsnURL := &url.URL{
		Scheme: "postgres",
		User: url.UserPassword(
			pool.Config().ConnConfig.User,
			pool.Config().ConnConfig.Password,
		),
		Host: net.JoinHostPort(
			pool.Config().ConnConfig.Host,
			strconv.Itoa(int(pool.Config().ConnConfig.Port)),
		),
		Path: db,
	}
	q := dsnURL.Query()
	q.Set("sslmode", "disable")
	dsnURL.RawQuery = q.Encode()
	connStr := dsnURL.String()
	
	sqlDB, err := sql.Open("postgres", connStr)
	if err != nil {
		logger.Error("Failed to create audit logger database connection", zap.Error(err))
		// Continue without audit logger - it's not critical for operation
		return &manager{
			pool:                pool,
			logger:              logger,
			historyLogger:       NewHistoryLogger(pool, logger),
			notificationService: NewNotificationService(logger),
			auditLogger:         nil,
		}
	}
	
	return &manager{
		pool:                pool,
		logger:              logger,
		historyLogger:       NewHistoryLogger(pool, logger),
		notificationService: NewNotificationService(logger),
		auditLogger:         audit.NewLogger(sqlDB),
	}
}

// DeductBalance deducts balance atomically with WHERE condition to prevent negative balance
// Implements Requirements 1.1, 1.2, 1.3, 1.5
func (m *manager) DeductBalance(ctx context.Context, op BalanceOperation) (*BalanceResult, error) {
	start := time.Now()
	tenantIDStr := op.TenantID.String()
	
	// Validate operation
	if op.TenantID == uuid.Nil {
		metrics.RecordBalanceOperation(tenantIDStr, "DEDUCT", "failure")
		return nil, NewTenantNotFoundError(op.TenantID)
	}
	if op.Amount.LessThanOrEqual(decimal.Zero) {
		metrics.RecordBalanceOperation(tenantIDStr, "DEDUCT", "failure")
		return nil, NewDatabaseError(fmt.Errorf("amount must be positive"))
	}

	// Retry logic with exponential backoff for concurrent updates
	maxRetries := 3
	baseDelay := 10 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			delay := baseDelay * time.Duration(1<<uint(attempt-1))
			m.logger.Debug("Retrying balance deduction after concurrent update",
				zap.String("tenant_id", op.TenantID.String()),
				zap.Int("attempt", attempt+1),
				zap.Duration("delay", delay),
			)
			time.Sleep(delay)
		}

		result, err := m.attemptDeduct(ctx, op)
		if err != nil {
			// Check if it's a concurrent update error
			if balErr, ok := err.(*BalanceError); ok && balErr.Type == ErrorConcurrentUpdate {
				if attempt < maxRetries-1 {
					continue // Retry
				}
			}
			
			// Record failure
			metrics.RecordBalanceOperation(tenantIDStr, "DEDUCT", "failure")
			metrics.ObserveBalanceOperationDuration("DEDUCT", time.Since(start).Seconds())
			
			return nil, err
		}

		// Record success
		metrics.RecordBalanceOperation(tenantIDStr, "DEDUCT", "success")
		metrics.ObserveBalanceOperationDuration("DEDUCT", time.Since(start).Seconds())
		
		// Update tenant balance gauge
		balanceFloat, _ := result.NewBalance.Float64()
		metrics.SetTenantBalance(tenantIDStr, balanceFloat)

		return result, nil
	}

	metrics.RecordBalanceOperation(tenantIDStr, "DEDUCT", "failure")
	return nil, NewConcurrentUpdateError(fmt.Errorf("max retries exceeded"))
}

// DeductBalanceTx implements the atomic deduction within an existing transaction
// Audit Fix: Allows shared transaction with other DB operations (e.g. Message updates)
func (m *manager) DeductBalanceTx(ctx context.Context, tx pgx.Tx, op BalanceOperation) (*BalanceResult, error) {
	// Validate operation
	if op.TenantID == uuid.Nil {
		return nil, NewTenantNotFoundError(op.TenantID)
	}
	if op.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, NewDatabaseError(fmt.Errorf("amount must be positive"))
	}

	// Atomic balance deduction with WHERE condition
	query := `
		UPDATE tenants 
		SET 
			balance = balance - $1,
			updated_at = NOW()
		WHERE 
			id = $2 
			AND balance >= $1
			AND deleted_at IS NULL
		RETURNING 
			id,
			balance AS new_balance,
			(balance + $1) AS old_balance
	`

	var tID uuid.UUID
	var nBalance, oBalance decimal.Decimal

	err := tx.QueryRow(ctx, query, op.Amount, op.TenantID).Scan(&tID, &nBalance, &oBalance)
	if err != nil {
		// Handle trigger exception for negative balance
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Message == "Insufficient balance: balance cannot be negative" {
			return nil, NewInsufficientBalanceError(op.TenantID, op.Amount, decimal.Zero)
		}

		if err == pgx.ErrNoRows {
			// Check if tenant exists
			var exists bool
			var currentBalance decimal.Decimal
			checkQuery := `SELECT EXISTS(SELECT 1 FROM tenants WHERE id = $1 AND deleted_at IS NULL), 
			               COALESCE((SELECT balance FROM tenants WHERE id = $1 AND deleted_at IS NULL), 0)`
			if err := tx.QueryRow(ctx, checkQuery, op.TenantID).Scan(&exists, &currentBalance); err != nil {
				return nil, NewDatabaseError(fmt.Errorf("failed to check tenant existence: %w", err))
			}

			if !exists {
				return nil, NewTenantNotFoundError(op.TenantID)
			}
			return nil, NewInsufficientBalanceError(op.TenantID, op.Amount, currentBalance)
		}
		return nil, NewDatabaseError(fmt.Errorf("failed to deduct balance in tx: %w", err))
	}

	// Log balance history using HistoryLogger
	err = m.historyLogger.LogDeduction(ctx, tx, HistoryEntry{
		TenantID:   op.TenantID,
		MessageID:  op.MessageID,
		GatewayID:  op.GatewayID,
		OldBalance: oBalance,
		NewBalance: nBalance,
		Amount:     op.Amount,
		Reason:     op.Reason,
		CreatedBy:  op.CreatedBy,
		IPAddress:  op.IPAddress,
	})
	if err != nil {
		return nil, NewDatabaseError(fmt.Errorf("failed to log balance history in tx: %w", err))
	}

	// Create and log accounting transaction record directly in same TX
	// This ensures consistency between balance and accounting
	accountingQuery := `
		INSERT INTO accounting_transactions (
			tenant_id, message_id, amount, status, type, created_at, updated_at, gateway_id
		) VALUES ($1, $2, $3, 'COMMITTED', 'DEDUCT', NOW(), NOW(), $4)
	`
	_, err = tx.Exec(ctx, accountingQuery, op.TenantID, op.MessageID, op.Amount, op.GatewayID)
	if err != nil {
		return nil, NewDatabaseError(fmt.Errorf("failed to insert accounting transaction in tx: %w", err))
	}

	return &BalanceResult{
		Success:    true,
		OldBalance: oBalance,
		NewBalance: nBalance,
	}, nil
}

// DeductBalanceWithTimeout deducts balance with a timeout context
// Implements Requirements 27.1, 27.2
func (m *manager) DeductBalanceWithTimeout(ctx context.Context, op BalanceOperation, timeout time.Duration) (*BalanceResult, error) {
	// Create context with timeout (default 5 seconds if not specified)
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Use a channel to handle the result
	resultChan := make(chan *BalanceResult, 1)
	errChan := make(chan error, 1)

	// Execute deduction in goroutine
	go func() {
		result, err := m.DeductBalance(timeoutCtx, op)
		if err != nil {
			errChan <- err
			return
		}
		resultChan <- result
	}()

	// Wait for result or timeout
	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errChan:
		return nil, err
	case <-timeoutCtx.Done():
		// Context cancelled or timed out
		if timeoutCtx.Err() == context.DeadlineExceeded {
			m.logger.Warn("Balance deduction timed out",
				zap.String("tenant_id", op.TenantID.String()),
				zap.Any("message_id", op.MessageID),
				zap.Duration("timeout", timeout),
			)
			return nil, NewDatabaseError(fmt.Errorf("operation timed out after %v", timeout))
		}
		return nil, NewDatabaseError(fmt.Errorf("operation cancelled: %w", timeoutCtx.Err()))
	}
}

// attemptDeduct attempts a single balance deduction
func (m *manager) attemptDeduct(ctx context.Context, op BalanceOperation) (*BalanceResult, error) {
	// Start transaction with ReadCommitted isolation level
	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.ReadCommitted,
	})
	if err != nil {
		return nil, NewDatabaseError(fmt.Errorf("failed to begin transaction: %w", err))
	}
	defer tx.Rollback(ctx)

	// Atomic balance deduction with WHERE condition (Requirements 1.1, 1.2)
	// This prevents negative balance and race conditions
	query := `
		UPDATE tenants 
		SET 
			balance = balance - $1,
			updated_at = NOW()
		WHERE 
			id = $2 
			AND balance >= $1
			AND deleted_at IS NULL
		RETURNING 
			id,
			balance AS new_balance,
			(balance + $1) AS old_balance
	`

	var tID uuid.UUID
	var nBalance, oBalance decimal.Decimal

	err = tx.QueryRow(ctx, query, op.Amount, op.TenantID).Scan(&tID, &nBalance, &oBalance)
	if err != nil {
		// Handle trigger exception for negative balance (Requirement: Database Integrity)
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Message == "Insufficient balance: balance cannot be negative" {
			return nil, NewInsufficientBalanceError(op.TenantID, op.Amount, decimal.Zero) // currentBalance unknown here but will be fetched below if needed
		}

		if err == pgx.ErrNoRows {
			// No rows updated - either tenant not found or insufficient balance
			// Check if tenant exists
			var exists bool
			var currentBalance decimal.Decimal
			checkQuery := `SELECT EXISTS(SELECT 1 FROM tenants WHERE id = $1 AND deleted_at IS NULL), 
			               COALESCE((SELECT balance FROM tenants WHERE id = $1 AND deleted_at IS NULL), 0)`
			if err := tx.QueryRow(ctx, checkQuery, op.TenantID).Scan(&exists, &currentBalance); err != nil {
				return nil, NewDatabaseError(fmt.Errorf("failed to check tenant existence: %w", err))
			}

			if !exists {
				return nil, NewTenantNotFoundError(op.TenantID)
			}

			// Tenant exists but balance insufficient
			// Log insufficient balance attempt (Requirement 11.6)
			if logErr := m.notificationService.LogInsufficientBalanceAttempt(ctx, op.TenantID, op.MessageID, op.Amount, currentBalance); logErr != nil {
				m.logger.Warn("Failed to log insufficient balance attempt",
					zap.String("tenant_id", op.TenantID.String()),
					zap.Any("message_id", op.MessageID),
					zap.Error(logErr),
				)
			}

			// Log to audit trail (Requirement 39.1, 39.2)
			if m.auditLogger != nil {
				auditDetails := map[string]interface{}{
					"requested_amount": op.Amount.String(),
					"current_balance":  currentBalance.String(),
					"message_id":       op.MessageID,
					"reason":           op.Reason,
				}
				
				if auditErr := m.auditLogger.LogFailure(
					audit.ActionBalanceDeduct,
					audit.ResourceBalance,
					op.MessageID.String(),
					nil, // User ID (system operation)
					&op.TenantID,
					op.IPAddress,
					"Insufficient balance",
					auditDetails,
				); auditErr != nil {
					m.logger.Warn("Failed to log audit entry",
						zap.String("tenant_id", op.TenantID.String()),
						zap.Error(auditErr),
					)
				}
			}

			return nil, NewInsufficientBalanceError(op.TenantID, op.Amount, currentBalance)
		}
		return nil, NewDatabaseError(fmt.Errorf("failed to deduct balance: %w", err))
	}

	// Log balance history using HistoryLogger (Requirements 10.1, 10.2, 20.2)
	err = m.historyLogger.LogDeduction(ctx, tx, HistoryEntry{
		TenantID:   op.TenantID,
		MessageID:  op.MessageID,
		GatewayID:  op.GatewayID,
		OldBalance: oBalance,
		NewBalance: nBalance,
		Amount:     op.Amount,
		Reason:     op.Reason,
		CreatedBy:  op.CreatedBy,
		IPAddress:  op.IPAddress,
	})
	if err != nil {
		return nil, NewDatabaseError(fmt.Errorf("failed to log balance history: %w", err))
	}

	// Insert into accounting_transactions - ensures every deduction has a record
	// Implements Requirement 16.1, 16.4
	accountingQuery := `
		INSERT INTO accounting_transactions (
			tenant_id, message_id, amount, status, type, created_at, updated_at, gateway_id
		) VALUES ($1, $2, $3, 'COMMITTED', 'DEDUCT', NOW(), NOW(), $4)
	`
	_, err = tx.Exec(ctx, accountingQuery, op.TenantID, op.MessageID, op.Amount, op.GatewayID)
	if err != nil {
		return nil, NewDatabaseError(fmt.Errorf("failed to insert accounting transaction: %w", err))
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, NewDatabaseError(fmt.Errorf("failed to commit transaction: %w", err))
	}

	// Log to audit trail (Requirement 39.1, 39.2)
	if m.auditLogger != nil {
		auditDetails := map[string]interface{}{
			"amount":      op.Amount.String(),
			"old_balance": oBalance.String(),
			"new_balance": nBalance.String(),
			"message_id":  op.MessageID,
			"reason":      op.Reason,
		}
		
		if auditErr := m.auditLogger.LogSuccess(
			audit.ActionBalanceDeduct,
			audit.ResourceBalance,
			op.MessageID.String(),
			nil, // User ID (system operation)
			&op.TenantID,
			op.IPAddress,
			auditDetails,
		); auditErr != nil {
			m.logger.Warn("Failed to log audit entry",
				zap.String("tenant_id", op.TenantID.String()),
				zap.Error(auditErr),
			)
		}
	}

	// Check and notify for low balance after successful deduction (Requirements 11.4, 11.5)
	if notifyErr := m.notificationService.CheckAndNotifyLowBalance(ctx, op.TenantID, nBalance); notifyErr != nil {
		// Log error but don't fail the operation since deduction was successful
		m.logger.Warn("Failed to send low balance notification",
			zap.String("tenant_id", op.TenantID.String()),
			zap.String("new_balance", nBalance.String()),
			zap.Error(notifyErr),
		)
	}

	m.logger.Info("Balance deducted successfully",
		zap.String("tenant_id", op.TenantID.String()),
		zap.Any("message_id", op.MessageID),
		zap.String("amount", op.Amount.String()),
		zap.String("old_balance", oBalance.String()),
		zap.String("new_balance", nBalance.String()),
	)

	return &BalanceResult{
		Success:    true,
		OldBalance: oBalance,
		NewBalance: nBalance,
		Error:      nil,
	}, nil
}

// RefundBalance refunds balance atomically using INCREMENT with idempotency check
// Implements Requirements 2.2, 2.5, 16.8, 16.9
func (m *manager) RefundBalance(ctx context.Context, op BalanceOperation) (*BalanceResult, error) {
	start := time.Now()
	tenantIDStr := op.TenantID.String()
	
	// Validate operation
	if op.TenantID == uuid.Nil {
		metrics.RecordBalanceOperation(tenantIDStr, "REFUND", "failure")
		return nil, NewTenantNotFoundError(op.TenantID)
	}
	if op.Amount.LessThanOrEqual(decimal.Zero) {
		metrics.RecordBalanceOperation(tenantIDStr, "REFUND", "failure")
		return nil, NewDatabaseError(fmt.Errorf("amount must be positive"))
	}
	if op.MessageID == nil {
		metrics.RecordBalanceOperation(tenantIDStr, "REFUND", "failure")
		return nil, NewDatabaseError(fmt.Errorf("message_id is required for refund"))
	}

	// Start transaction
	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.ReadCommitted,
	})
	if err != nil {
		metrics.RecordBalanceOperation(tenantIDStr, "REFUND", "failure")
		return nil, NewDatabaseError(fmt.Errorf("failed to begin transaction: %w", err))
	}
	defer tx.Rollback(ctx)

	// Check if refund already processed or if deduction exists (Requirement 16.9 + Audit Fix)
	checkQuery := `
		SELECT status, amount 
		FROM accounting_transactions 
		WHERE tenant_id = $1 AND message_id = $2
		  AND type IN ('DEDUCT', 'DEDUCTION')
	`
	
	var currentStatus string
	var originalAmount float64
	err = tx.QueryRow(ctx, checkQuery, op.TenantID, op.MessageID).Scan(&currentStatus, &originalAmount)
	
	if err == pgx.ErrNoRows {
		metrics.RecordBalanceOperation(tenantIDStr, "REFUND", "failure")
		return nil, NewDatabaseError(fmt.Errorf("no original transaction found for message %s", op.MessageID))
	}
	
	if err != nil && err != pgx.ErrNoRows {
		metrics.RecordBalanceOperation(tenantIDStr, "REFUND", "failure")
		return nil, NewDatabaseError(fmt.Errorf("failed to check transaction status: %w", err))
	}
	
	// If already refunded, return success without processing again (Idempotency)
	if currentStatus == "REFUNDED" {
		m.logger.Info("Refund already processed (idempotent)",
			zap.String("tenant_id", op.TenantID.String()),
			zap.Any("message_id", op.MessageID),
		)
		
		// Get current balance for response
		var currentBalance decimal.Decimal
		balanceQuery := `SELECT balance FROM tenants WHERE id = $1 AND deleted_at IS NULL`
		err = tx.QueryRow(ctx, balanceQuery, op.TenantID).Scan(&currentBalance)
		if err != nil {
			metrics.RecordBalanceOperation(tenantIDStr, "REFUND", "failure")
			return nil, NewDatabaseError(fmt.Errorf("failed to get current balance: %w", err))
		}
		
		metrics.RecordBalanceOperation(tenantIDStr, "REFUND", "success")
		return &BalanceResult{
			Success:    true,
			OldBalance: currentBalance,
			NewBalance: currentBalance,
		}, nil
	}

	// Audit Fix: Ensure original transaction was COMMITTED and amount matches
	if currentStatus != "COMMITTED" {
		return nil, NewDatabaseError(fmt.Errorf("transaction status is %s, only COMMITTED transactions can be refunded", currentStatus))
	}
	
	// Comparison using decimal for precision
	if !op.Amount.Equal(decimal.NewFromFloat(originalAmount)) {
		return nil, NewDatabaseError(fmt.Errorf("refund amount mismatch: expected %v, got %s", originalAmount, op.Amount.String()))
	}

	// Claim the refund atomically FIRST: mark the original DEDUCT/DEDUCTION
	// transaction as REFUNDED. The row lock serializes concurrent refunds for
	// the same message, so exactly one caller matches the guard and proceeds
	// to increment the balance. The REFUND entry written by LogRefund (and any
	// non-deduction type) is excluded so it can never be claimed.
	updateTxQuery := `
		UPDATE accounting_transactions
		SET 
			status = 'REFUNDED',
			updated_at = NOW(),
			settled_at = NOW()
		WHERE 
			tenant_id = $1 
			AND message_id = $2
			AND status != 'REFUNDED'
			AND type IN ('DEDUCT', 'DEDUCTION')
		RETURNING id
	`

	var txID uuid.UUID
	err = tx.QueryRow(ctx, updateTxQuery,
		op.TenantID,
		op.MessageID,
	).Scan(&txID)

	if err == pgx.ErrNoRows {
		// A concurrent refund already claimed this transaction (or it was
		// marked REFUNDED between the status check and this update). Nothing
		// has been written in this transaction yet, so return idempotent
		// success with the committed balance.
		var currentBalance decimal.Decimal
		balanceQuery := `SELECT balance FROM tenants WHERE id = $1 AND deleted_at IS NULL`
		err = tx.QueryRow(ctx, balanceQuery, op.TenantID).Scan(&currentBalance)
		if err != nil {
			metrics.RecordBalanceOperation(tenantIDStr, "REFUND", "failure")
			return nil, NewDatabaseError(fmt.Errorf("failed to get current balance: %w", err))
		}
		m.logger.Info("Refund already processed by concurrent request (idempotent)",
			zap.String("tenant_id", op.TenantID.String()),
			zap.Any("message_id", op.MessageID),
		)
		metrics.RecordBalanceOperation(tenantIDStr, "REFUND", "success")
		return &BalanceResult{
			Success:    true,
			OldBalance: currentBalance,
			NewBalance: currentBalance,
		}, nil
	}

	if err != nil {
		metrics.RecordBalanceOperation(tenantIDStr, "REFUND", "failure")
		return nil, NewDatabaseError(fmt.Errorf("failed to update transaction status: %w", err))
	}

	// Log transaction state update (Requirement 16.8)
	m.logger.Info("Transaction state updated to REFUNDED",
		zap.String("tenant_id", op.TenantID.String()),
		zap.Any("message_id", op.MessageID),
		zap.String("transaction_id", txID.String()),
	)

	// Atomic balance refund using INCREMENT (Requirement 2.5)
	query := `
		UPDATE tenants 
		SET 
			balance = balance + $1,
			updated_at = NOW()
		WHERE 
			id = $2 
			AND deleted_at IS NULL
		RETURNING 
			id,
			balance AS new_balance,
			(balance - $1) AS old_balance
	`

	var tID uuid.UUID
	var nBalance, oBalance decimal.Decimal

	err = tx.QueryRow(ctx, query, op.Amount, op.TenantID).Scan(&tID, &nBalance, &oBalance)
	if err != nil {
		metrics.RecordBalanceOperation(tenantIDStr, "REFUND", "failure")
		if err == pgx.ErrNoRows {
			return nil, NewTenantNotFoundError(op.TenantID)
		}
		return nil, NewDatabaseError(fmt.Errorf("failed to refund balance: %w", err))
	}

	// Log balance history using HistoryLogger (Requirements 10.3, 20.2, 16.8)
	err = m.historyLogger.LogRefund(ctx, tx, HistoryEntry{
		TenantID:   op.TenantID,
		MessageID:  op.MessageID,
		OldBalance: oBalance,
		NewBalance: nBalance,
		Amount:     op.Amount,
		Reason:     op.Reason,
		CreatedBy:  op.CreatedBy,
		IPAddress:  op.IPAddress,
	})
	if err != nil {
		metrics.RecordBalanceOperation(tenantIDStr, "REFUND", "failure")
		return nil, NewDatabaseError(fmt.Errorf("failed to log balance history: %w", err))
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		metrics.RecordBalanceOperation(tenantIDStr, "REFUND", "failure")
		return nil, NewDatabaseError(fmt.Errorf("failed to commit transaction: %w", err))
	}

	// Log to audit trail (Requirement 39.1, 39.2)
	if m.auditLogger != nil {
		auditDetails := map[string]interface{}{
			"amount":      op.Amount.String(),
			"old_balance": oBalance.String(),
			"new_balance": nBalance.String(),
			"message_id":  op.MessageID,
			"reason":      op.Reason,
		}
		
		if auditErr := m.auditLogger.LogSuccess(
			audit.ActionBalanceRefund,
			audit.ResourceBalance,
			op.MessageID.String(),
			nil, // User ID (system operation)
			&op.TenantID,
			op.IPAddress,
			auditDetails,
		); auditErr != nil {
			m.logger.Warn("Failed to log audit entry",
				zap.String("tenant_id", op.TenantID.String()),
				zap.Error(auditErr),
			)
		}
	}

	// Record success metrics
	metrics.RecordBalanceOperation(tenantIDStr, "REFUND", "success")
	metrics.ObserveBalanceOperationDuration("REFUND", time.Since(start).Seconds())
	
	// Update tenant balance gauge
	balanceFloat, _ := nBalance.Float64()
	metrics.SetTenantBalance(tenantIDStr, balanceFloat)

	m.logger.Info("Balance refunded successfully",
		zap.String("tenant_id", op.TenantID.String()),
		zap.Any("message_id", op.MessageID),
		zap.String("amount", op.Amount.String()),
		zap.String("old_balance", oBalance.String()),
		zap.String("new_balance", nBalance.String()),
	)

	return &BalanceResult{
		Success:    true,
		OldBalance: oBalance,
		NewBalance: nBalance,
		Error:      nil,
	}, nil
}

// GetBalance returns the current balance for a tenant
func (m *manager) GetBalance(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
	if tenantID == uuid.Nil {
		return decimal.Zero, NewTenantNotFoundError(tenantID)
	}

	query := `
		SELECT balance 
		FROM tenants 
		WHERE id = $1 AND deleted_at IS NULL
	`

	var balance decimal.Decimal
	err := m.pool.QueryRow(ctx, query, tenantID).Scan(&balance)
	if err != nil {
		if err == pgx.ErrNoRows || err == sql.ErrNoRows {
			return decimal.Zero, NewTenantNotFoundError(tenantID)
		}
		return decimal.Zero, NewDatabaseError(fmt.Errorf("failed to get balance: %w", err))
	}

	return balance, nil
}

// CheckSufficient checks if the tenant has sufficient balance
// Implements Requirement 1.2
func (m *manager) CheckSufficient(ctx context.Context, tenantID uuid.UUID, amount decimal.Decimal) (bool, error) {
	if tenantID == uuid.Nil {
		return false, NewTenantNotFoundError(tenantID)
	}

	if amount.LessThanOrEqual(decimal.Zero) {
		return true, nil // Zero or negative amount is always sufficient
	}

	balance, err := m.GetBalance(ctx, tenantID)
	if err != nil {
		return false, err
	}

	return balance.GreaterThanOrEqual(amount), nil
}

// RefundBalanceTx implements atomic refund within an existing transaction
// Audit Fix: Allows shared transaction with other recovery operations
func (m *manager) RefundBalanceTx(ctx context.Context, tx pgx.Tx, op BalanceOperation) (*BalanceResult, error) {
	// Validate operation
	if op.TenantID == uuid.Nil || op.MessageID == nil {
		return nil, NewDatabaseError(fmt.Errorf("tenant_id and message_id are required for refund"))
	}
	if op.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, NewDatabaseError(fmt.Errorf("amount must be positive"))
	}

	// Check idempotency and original transaction
	checkQuery := `
		SELECT status, amount 
		FROM accounting_transactions 
		WHERE tenant_id = $1 AND message_id = $2
		  AND type IN ('DEDUCT', 'DEDUCTION')
	`
	var currentStatus string
	var originalAmount float64
	err := tx.QueryRow(ctx, checkQuery, op.TenantID, op.MessageID).Scan(&currentStatus, &originalAmount)
	
	if err == pgx.ErrNoRows {
		return nil, NewDatabaseError(fmt.Errorf("no original transaction found for message %s", op.MessageID))
	}
	if err != nil {
		return nil, NewDatabaseError(fmt.Errorf("failed to check transaction status in tx: %w", err))
	}

	if currentStatus == "REFUNDED" {
		return &BalanceResult{Success: true}, nil
	}

	if currentStatus != "COMMITTED" {
		return nil, NewDatabaseError(fmt.Errorf("transaction status is %s, cannot refund", currentStatus))
	}

	if !op.Amount.Equal(decimal.NewFromFloat(originalAmount)) {
		return nil, NewDatabaseError(fmt.Errorf("refund amount mismatch: expected %v, got %s", originalAmount, op.Amount.String()))
	}

	// Atomic update
	query := `
		UPDATE tenants SET balance = balance + $1, updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING balance, (balance - $1)
	`
	var nBalance, oBalance decimal.Decimal
	err = tx.QueryRow(ctx, query, op.Amount, op.TenantID).Scan(&nBalance, &oBalance)
	if err != nil {
		return nil, NewDatabaseError(fmt.Errorf("failed to update balance in tx: %w", err))
	}

	// Update transaction status
	_, err = tx.Exec(ctx, `
		UPDATE accounting_transactions 
		SET status = 'REFUNDED', updated_at = NOW(), settled_at = NOW()
		WHERE tenant_id = $1 AND message_id = $2
		  AND type IN ('DEDUCT', 'DEDUCTION')
	`, op.TenantID, op.MessageID)
	if err != nil {
		return nil, NewDatabaseError(fmt.Errorf("failed to update tx status in tx: %w", err))
	}

	// Log history
	err = m.historyLogger.LogRefund(ctx, tx, HistoryEntry{
		TenantID:   op.TenantID,
		MessageID:  op.MessageID,
		OldBalance: oBalance,
		NewBalance: nBalance,
		Amount:     op.Amount,
		Reason:     op.Reason,
		CreatedBy:  op.CreatedBy,
		IPAddress:  op.IPAddress,
	})
	if err != nil {
		return nil, NewDatabaseError(fmt.Errorf("failed to log refund history in tx: %w", err))
	}

	return &BalanceResult{
		Success:    true,
		OldBalance: oBalance,
		NewBalance: nBalance,
	}, nil
}
