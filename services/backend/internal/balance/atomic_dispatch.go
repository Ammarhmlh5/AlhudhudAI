package balance

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
)

// RawTx is a thin interface over pgx.Tx that exposes only what we need,
// making it trivial to mock in tests.
type RawTx interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// DeductAndActivateMessage performs an atomic operation that:
//  1. Deducts the message cost from the tenant's balance.
//  2. Updates the message status from PENDING → QUEUED in the same DB row.
//  3. Records the accounting_transaction as COMMITTED.
//
// If the deduction fails (e.g., insufficient balance), the message stays PENDING
// and NO side effects occur. This is the Phase 3 core atomic transaction.
func DeductAndActivateMessage(
	ctx context.Context,
	tx RawTx,
	tenantID uuid.UUID,
	messageID uuid.UUID,
	totalCost decimal.Decimal,
) error {

	// 1. Deduct balance atomically (prevents negative balance via WHERE clause)
	var newBalance decimal.Decimal
	err := tx.QueryRow(ctx, `
		UPDATE tenants
		SET   balance    = balance - $1,
		      updated_at = NOW()
		WHERE id             = $2
		  AND balance       >= $1
		  AND deleted_at IS NULL
		RETURNING balance
	`, totalCost, tenantID).Scan(&newBalance)
	if err != nil {
		// pgx.ErrNoRows → insufficient balance or tenant not found
		return fmt.Errorf("insufficient balance or tenant not found for tenant %s: %w", tenantID, err)
	}

	// 2. Activate message: PENDING → QUEUED,  set cost = totalCost
	result, err := tx.Exec(ctx, `
		UPDATE messages
		SET   status     = 'QUEUED',
		      cost       = $1,
		      updated_at = NOW()
		WHERE id        = $2
		  AND tenant_id = $3
		  AND status    = 'PENDING'
	`, totalCost, messageID, tenantID)
	if err != nil {
		return fmt.Errorf("activate message: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("message %s not found or already activated", messageID)
	}

	// 3. Record the accounting transaction as COMMITTED in one step
	_, err = tx.Exec(ctx, `
		INSERT INTO accounting_transactions (tenant_id, message_id, amount, status, type, created_at)
		VALUES ($1, $2, $3, 'COMMITTED', 'DEDUCTION', NOW())
		ON CONFLICT DO NOTHING
	`, tenantID, messageID, totalCost)
	if err != nil {
		return fmt.Errorf("record accounting transaction: %w", err)
	}

	return nil
}
