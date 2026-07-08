package balance

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// HistoryLogger interface defines balance history logging operations
// Implements Requirements 10.1, 10.2, 10.3, 10.4, 10.5, 20.2
type HistoryLogger interface {
	// LogDeduction logs a balance deduction operation
	LogDeduction(ctx context.Context, tx pgx.Tx, entry HistoryEntry) error

	// LogRefund logs a balance refund operation
	LogRefund(ctx context.Context, tx pgx.Tx, entry HistoryEntry) error

	// LogCharge logs a balance charge (add balance) operation
	LogCharge(ctx context.Context, tx pgx.Tx, entry HistoryEntry) error

	// LogAdjustment logs a manual balance adjustment operation
	LogAdjustment(ctx context.Context, tx pgx.Tx, entry HistoryEntry) error

	// GetHistory retrieves balance history for a tenant
	GetHistory(ctx context.Context, tenantID uuid.UUID, limit int) ([]BalanceHistory, error)
}

// HistoryEntry represents a balance history entry to be logged
type HistoryEntry struct {
	TenantID   uuid.UUID       `json:"tenant_id"`
	MessageID  *uuid.UUID      `json:"message_id,omitempty"`
	GatewayID  *uuid.UUID      `json:"gateway_id,omitempty"` // Link to dedicated gateway
	OldBalance decimal.Decimal `json:"old_balance"`
	NewBalance decimal.Decimal `json:"new_balance"`
	Amount     decimal.Decimal `json:"amount"`
	Type       OperationType   `json:"type"`
	Reason     string          `json:"reason"`
	CreatedBy  string          `json:"created_by"`
	IPAddress  string          `json:"ip_address,omitempty"` // Requirement 20.2
}

// historyLogger implements the HistoryLogger interface
type historyLogger struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewHistoryLogger creates a new balance history logger
func NewHistoryLogger(pool *pgxpool.Pool, logger *zap.Logger) HistoryLogger {
	return &historyLogger{
		pool:   pool,
		logger: logger,
	}
}

// LogDeduction logs a balance deduction operation
// Implements Requirements 10.1, 10.2, 20.2
func (hl *historyLogger) LogDeduction(ctx context.Context, tx pgx.Tx, entry HistoryEntry) error {
	entry.Type = OperationTypeDeduct
	return hl.logEntry(ctx, tx, entry)
}

// LogRefund logs a balance refund operation
// Implements Requirements 10.3, 20.2
func (hl *historyLogger) LogRefund(ctx context.Context, tx pgx.Tx, entry HistoryEntry) error {
	entry.Type = OperationTypeRefund
	return hl.logEntry(ctx, tx, entry)
}

// LogCharge logs a balance charge operation
// Implements Requirements 10.1, 10.2, 20.2
func (hl *historyLogger) LogCharge(ctx context.Context, tx pgx.Tx, entry HistoryEntry) error {
	entry.Type = OperationTypeCharge
	return hl.logEntry(ctx, tx, entry)
}

// LogAdjustment logs a manual balance adjustment operation
// Implements Requirements 10.1, 10.2, 20.2
func (hl *historyLogger) LogAdjustment(ctx context.Context, tx pgx.Tx, entry HistoryEntry) error {
	entry.Type = OperationTypeAdjustment
	return hl.logEntry(ctx, tx, entry)
}

// logEntry logs a balance history entry
// Implements Requirements 10.1, 10.2, 10.3, 10.4, 10.5, 20.2
func (hl *historyLogger) logEntry(ctx context.Context, tx pgx.Tx, entry HistoryEntry) error {
	// Validate entry
	if entry.TenantID == uuid.Nil {
		return fmt.Errorf("invalid tenant_id: %s", entry.TenantID)
	}

	// 1. Insert into balance_history
	balanceQuery := `
		INSERT INTO balance_history (
			tenant_id, message_id, gateway_id,
			old_balance, new_balance, amount,
			type, reason, created_by, ip_address, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
	`

	var ipAddress *string
	if entry.IPAddress != "" {
		ipAddress = &entry.IPAddress
	}

	_, err := tx.Exec(ctx, balanceQuery,
		entry.TenantID, entry.MessageID, entry.GatewayID,
		entry.OldBalance, entry.NewBalance, entry.Amount,
		entry.Type, entry.Reason, entry.CreatedBy, ipAddress,
	)
	if err != nil {
		return fmt.Errorf("failed to log balance history: %w", err)
	}

	// 2. Insert into accounting_transactions for unified reporting
	// Requirement: Fast aggregation per gateway
	accountingType := "DEDUCTION"
	if entry.Type == OperationTypeRefund {
		accountingType = "REFUND"
	} else if entry.Type == OperationTypeCharge {
		accountingType = "DEPOSIT"
	}

	accountingQuery := `
		INSERT INTO accounting_transactions (
			tenant_id, message_id, gateway_id,
			type, amount, status,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	`

	accountingStatus := "PENDING"
	if entry.Type == OperationTypeDeduct {
		accountingStatus = "PENDING" // Will be set to COMPLETED by status handler
	} else {
		accountingStatus = "COMPLETED"
	}

	_, err = tx.Exec(ctx, accountingQuery,
		entry.TenantID, entry.MessageID, entry.GatewayID,
		accountingType, entry.Amount, accountingStatus,
	)
	if err != nil {
		// Log warning but don't fail the whole transaction if accounting insert fails
		// (though in message processing, consistency is important)
		hl.logger.Warn("Failed to log accounting transaction", zap.Error(err))
	}

	return nil
}

// GetHistory retrieves balance history for a tenant
// Implements Requirement 10.7
func (hl *historyLogger) GetHistory(ctx context.Context, tenantID uuid.UUID, limit int) ([]BalanceHistory, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("invalid tenant_id: %s", tenantID)
	}

	if limit <= 0 {
		limit = 100 // Default limit
	}

	query := `
		SELECT 
			id,
			tenant_id,
			message_id,
			old_balance,
			new_balance,
			amount,
			type,
			reason,
			created_by,
			ip_address,
			created_at
		FROM balance_history
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := hl.pool.Query(ctx, query, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query balance history: %w", err)
	}
	defer rows.Close()

	var history []BalanceHistory
	for rows.Next() {
		var h BalanceHistory
		err := rows.Scan(
			&h.ID,
			&h.TenantID,
			&h.MessageID,
			&h.OldBalance,
			&h.NewBalance,
			&h.Amount,
			&h.Type,
			&h.Reason,
			&h.CreatedBy,
			&h.IPAddress,
			&h.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan balance history: %w", err)
		}
		history = append(history, h)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating balance history: %w", err)
	}

	return history, nil
}
