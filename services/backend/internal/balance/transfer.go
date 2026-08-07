package balance

import (
	"context"
	"fmt"
	"time"

	"github.com/al-hudhud/backend/internal/audit"
	"github.com/al-hudhud/backend/internal/metrics"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// TransferResult represents the outcome of a balance transfer between two tenants.
type TransferResult struct {
	Success        bool            `json:"success"`
	Amount         decimal.Decimal `json:"amount"`
	FromOldBalance decimal.Decimal `json:"from_old_balance"`
	FromNewBalance decimal.Decimal `json:"from_new_balance"`
	ToOldBalance   decimal.Decimal `json:"to_old_balance"`
	ToNewBalance   decimal.Decimal `json:"to_new_balance"`
}

// CreditBalance credits balance to a tenant atomically.
// Used by admins to fund a reseller wallet (OperationTypeCharge).
func (m *manager) CreditBalance(ctx context.Context, op BalanceOperation) (*BalanceResult, error) {
	start := time.Now()
	tenantIDStr := op.TenantID.String()

	if op.TenantID == uuid.Nil {
		metrics.RecordBalanceOperation(tenantIDStr, "CHARGE", "failure")
		return nil, NewTenantNotFoundError(op.TenantID)
	}
	if op.Amount.LessThanOrEqual(decimal.Zero) {
		metrics.RecordBalanceOperation(tenantIDStr, "CHARGE", "failure")
		return nil, NewDatabaseError(fmt.Errorf("amount must be positive"))
	}

	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		metrics.RecordBalanceOperation(tenantIDStr, "CHARGE", "failure")
		return nil, NewDatabaseError(fmt.Errorf("failed to begin transaction: %w", err))
	}
	defer tx.Rollback(ctx)

	// Scope RLS to the credited tenant so the history/accounting inserts pass.
	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", op.TenantID.String()); err != nil {
		metrics.RecordBalanceOperation(tenantIDStr, "CHARGE", "failure")
		return nil, NewDatabaseError(fmt.Errorf("failed to set tenant context: %w", err))
	}

	query := `
		UPDATE tenants
		SET balance = balance + $1, updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING id, balance AS new_balance, (balance - $1) AS old_balance
	`

	var tID uuid.UUID
	var nBalance, oBalance decimal.Decimal
	err = tx.QueryRow(ctx, query, op.Amount, op.TenantID).Scan(&tID, &nBalance, &oBalance)
	if err != nil {
		metrics.RecordBalanceOperation(tenantIDStr, "CHARGE", "failure")
		if err == pgx.ErrNoRows {
			return nil, NewTenantNotFoundError(op.TenantID)
		}
		return nil, NewDatabaseError(fmt.Errorf("failed to credit balance: %w", err))
	}

	if err := m.historyLogger.LogCharge(ctx, tx, HistoryEntry{
		TenantID:   op.TenantID,
		MessageID:  op.MessageID,
		GatewayID:  op.GatewayID,
		OldBalance: oBalance,
		NewBalance: nBalance,
		Amount:     op.Amount,
		Reason:     op.Reason,
		CreatedBy:  op.CreatedBy,
		IPAddress:  op.IPAddress,
	}); err != nil {
		metrics.RecordBalanceOperation(tenantIDStr, "CHARGE", "failure")
		return nil, NewDatabaseError(fmt.Errorf("failed to log balance history: %w", err))
	}

	if err := tx.Commit(ctx); err != nil {
		metrics.RecordBalanceOperation(tenantIDStr, "CHARGE", "failure")
		return nil, NewDatabaseError(fmt.Errorf("failed to commit transaction: %w", err))
	}

	if m.auditLogger != nil {
		auditDetails := map[string]interface{}{
			"amount":      op.Amount.String(),
			"old_balance": oBalance.String(),
			"new_balance": nBalance.String(),
			"reason":      op.Reason,
		}
		if auditErr := m.auditLogger.LogSuccess(
			audit.ActionBalanceCharge,
			audit.ResourceBalance,
			op.TenantID.String(),
			nil,
			&op.TenantID,
			op.IPAddress,
			auditDetails,
		); auditErr != nil {
			m.logger.Warn("Failed to log audit entry", zap.Error(auditErr))
		}
	}

	metrics.RecordBalanceOperation(tenantIDStr, "CHARGE", "success")
	metrics.ObserveBalanceOperationDuration("CHARGE", time.Since(start).Seconds())
	balanceFloat, _ := nBalance.Float64()
	metrics.SetTenantBalance(tenantIDStr, balanceFloat)

	m.logger.Info("Balance credited successfully",
		zap.String("tenant_id", op.TenantID.String()),
		zap.String("amount", op.Amount.String()),
		zap.String("old_balance", oBalance.String()),
		zap.String("new_balance", nBalance.String()),
	)

	return &BalanceResult{
		Success:    true,
		OldBalance: oBalance,
		NewBalance: nBalance,
	}, nil
}

// TransferBalance moves funds from one tenant wallet to another atomically.
// The whole transfer succeeds or fails as a single unit: debit and credit use
// row-level atomic UPDATEs, and the history for both sides is written in the
// same transaction with the tenant context switched per side for RLS.
func (m *manager) TransferBalance(ctx context.Context, from, to uuid.UUID, amount decimal.Decimal, reason, createdBy, ipAddress string) (*TransferResult, error) {
	start := time.Now()

	if from == uuid.Nil || to == uuid.Nil {
		metrics.RecordBalanceOperation(from.String(), "TRANSFER", "failure")
		return nil, NewTenantNotFoundError(from)
	}
	if from == to {
		metrics.RecordBalanceOperation(from.String(), "TRANSFER", "failure")
		return nil, NewDatabaseError(fmt.Errorf("cannot transfer to the same tenant"))
	}
	if amount.LessThanOrEqual(decimal.Zero) {
		metrics.RecordBalanceOperation(from.String(), "TRANSFER", "failure")
		return nil, NewDatabaseError(fmt.Errorf("amount must be positive"))
	}

	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		metrics.RecordBalanceOperation(from.String(), "TRANSFER", "failure")
		return nil, NewDatabaseError(fmt.Errorf("failed to begin transaction: %w", err))
	}
	defer tx.Rollback(ctx)

	// 1. Deduct from the source atomically (prevents negative balance).
	deductQuery := `
		UPDATE tenants
		SET balance = balance - $1, updated_at = NOW()
		WHERE id = $2 AND balance >= $1 AND deleted_at IS NULL
		RETURNING id, balance AS new_balance, (balance + $1) AS old_balance
	`
	var fromID uuid.UUID
	var fromNew, fromOld decimal.Decimal
	err = tx.QueryRow(ctx, deductQuery, amount, from).Scan(&fromID, &fromNew, &fromOld)
	if err != nil {
		if err == pgx.ErrNoRows {
			var exists bool
			if qerr := tx.QueryRow(ctx,
				`SELECT EXISTS(SELECT 1 FROM tenants WHERE id = $1 AND deleted_at IS NULL)`, from,
			).Scan(&exists); qerr != nil {
				metrics.RecordBalanceOperation(from.String(), "TRANSFER", "failure")
				return nil, NewDatabaseError(fmt.Errorf("failed to check tenant existence: %w", qerr))
			}
			if !exists {
				metrics.RecordBalanceOperation(from.String(), "TRANSFER", "failure")
				return nil, NewTenantNotFoundError(from)
			}
			metrics.RecordBalanceOperation(from.String(), "TRANSFER", "failure")
			return nil, NewInsufficientBalanceError(from, amount, decimal.Zero)
		}
		metrics.RecordBalanceOperation(from.String(), "TRANSFER", "failure")
		return nil, NewDatabaseError(fmt.Errorf("failed to deduct balance in transfer: %w", err))
	}

	// 2. Credit the destination.
	creditQuery := `
		UPDATE tenants
		SET balance = balance + $1, updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING id, balance AS new_balance, (balance - $1) AS old_balance
	`
	var toID uuid.UUID
	var toNew, toOld decimal.Decimal
	err = tx.QueryRow(ctx, creditQuery, amount, to).Scan(&toID, &toNew, &toOld)
	if err != nil {
		metrics.RecordBalanceOperation(from.String(), "TRANSFER", "failure")
		if err == pgx.ErrNoRows {
			return nil, NewTenantNotFoundError(to)
		}
		return nil, NewDatabaseError(fmt.Errorf("failed to credit balance in transfer: %w", err))
	}

	// 3. Log history for the source tenant.
	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", from.String()); err != nil {
		metrics.RecordBalanceOperation(from.String(), "TRANSFER", "failure")
		return nil, NewDatabaseError(fmt.Errorf("failed to set tenant context: %w", err))
	}
	if err := m.historyLogger.LogDeduction(ctx, tx, HistoryEntry{
		TenantID:   from,
		OldBalance: fromOld,
		NewBalance: fromNew,
		Amount:     amount,
		Reason:     fmt.Sprintf("TRANSFER_OUT to %s | %s", to, reason),
		CreatedBy:  createdBy,
		IPAddress:  ipAddress,
	}); err != nil {
		metrics.RecordBalanceOperation(from.String(), "TRANSFER", "failure")
		return nil, NewDatabaseError(fmt.Errorf("failed to log transfer-out history: %w", err))
	}

	// 4. Log history for the destination tenant.
	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", to.String()); err != nil {
		metrics.RecordBalanceOperation(from.String(), "TRANSFER", "failure")
		return nil, NewDatabaseError(fmt.Errorf("failed to set tenant context: %w", err))
	}
	if err := m.historyLogger.LogCharge(ctx, tx, HistoryEntry{
		TenantID:   to,
		OldBalance: toOld,
		NewBalance: toNew,
		Amount:     amount,
		Reason:     fmt.Sprintf("TRANSFER_IN from %s | %s", from, reason),
		CreatedBy:  createdBy,
		IPAddress:  ipAddress,
	}); err != nil {
		metrics.RecordBalanceOperation(from.String(), "TRANSFER", "failure")
		return nil, NewDatabaseError(fmt.Errorf("failed to log transfer-in history: %w", err))
	}

	if err := tx.Commit(ctx); err != nil {
		metrics.RecordBalanceOperation(from.String(), "TRANSFER", "failure")
		return nil, NewDatabaseError(fmt.Errorf("failed to commit transfer: %w", err))
	}

	// Audit trail for both sides.
	if m.auditLogger != nil {
		outDetails := map[string]interface{}{
			"amount":       amount.String(),
			"old_balance":  fromOld.String(),
			"new_balance":  fromNew.String(),
			"to_tenant_id": to.String(),
			"reason":       reason,
		}
		if auditErr := m.auditLogger.LogSuccess(
			audit.ActionBalanceDeduct,
			audit.ResourceBalance,
			from.String(),
			nil,
			&from,
			ipAddress,
			outDetails,
		); auditErr != nil {
			m.logger.Warn("Failed to log transfer-out audit entry", zap.Error(auditErr))
		}

		inDetails := map[string]interface{}{
			"amount":       amount.String(),
			"old_balance":  toOld.String(),
			"new_balance":  toNew.String(),
			"from_tenant_id": from.String(),
			"reason":       reason,
		}
		if auditErr := m.auditLogger.LogSuccess(
			audit.ActionBalanceCharge,
			audit.ResourceBalance,
			to.String(),
			nil,
			&to,
			ipAddress,
			inDetails,
		); auditErr != nil {
			m.logger.Warn("Failed to log transfer-in audit entry", zap.Error(auditErr))
		}
	}

	metrics.RecordBalanceOperation(from.String(), "TRANSFER", "success")
	metrics.ObserveBalanceOperationDuration("TRANSFER", time.Since(start).Seconds())
	fromFloat, _ := fromNew.Float64()
	toFloat, _ := toNew.Float64()
	metrics.SetTenantBalance(from.String(), fromFloat)
	metrics.SetTenantBalance(to.String(), toFloat)

	m.logger.Info("Balance transferred successfully",
		zap.String("from_tenant_id", from.String()),
		zap.String("to_tenant_id", to.String()),
		zap.String("amount", amount.String()),
		zap.String("from_old_balance", fromOld.String()),
		zap.String("from_new_balance", fromNew.String()),
		zap.String("to_old_balance", toOld.String()),
		zap.String("to_new_balance", toNew.String()),
	)

	return &TransferResult{
		Success:        true,
		Amount:         amount,
		FromOldBalance: fromOld,
		FromNewBalance: fromNew,
		ToOldBalance:   toOld,
		ToNewBalance:   toNew,
	}, nil
}
