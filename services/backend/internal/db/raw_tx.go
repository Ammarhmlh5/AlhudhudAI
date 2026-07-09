package db

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// RawTx is a thin interface over pgx.Tx that exposes only what we need,
// making it trivial to mock in tests.
type RawTx interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// RunInRawTx begins a ReadCommitted pgx transaction, runs fn inside it,
// and commits or rolls back automatically. It does NOT set RLS context.
// Use RunInScopedRawTx if you need tenant isolation.
func RunInRawTx(ctx context.Context, fn func(tx RawTx) error) error {
	return RunInScopedRawTx(ctx, uuid.Nil, false, fn)
}

// RunInScopedRawTx begins a transaction and sets the app.current_tenant 
// and app.is_super_admin session variables for RLS support.
func RunInScopedRawTx(ctx context.Context, tenantID uuid.UUID, isSuperAdmin bool, fn func(tx RawTx) error) error {
	tx, err := PgxPool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.ReadCommitted,
	})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Set RLS Context if tenantID is provided
	if tenantID != uuid.Nil {
		_, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID.String())
		if err != nil {
			return err
		}
	}

	adminVal := "false"
	if isSuperAdmin {
		adminVal = "true"
	}
	_, err = tx.Exec(ctx, "SELECT set_config('app.is_super_admin', $1, true)", adminVal)
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
