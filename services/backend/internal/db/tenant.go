package db

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jmoiron/sqlx"
)

// RunInTenantTx runs a function in a transaction with tenant context set
func RunInTenantTx(ctx context.Context, tenantID uuid.UUID, isSuperAdmin bool, db *sql.DB, fn func(tx *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// true means local to the transaction
	_, err = tx.ExecContext(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID.String())
	if err != nil {
		return err
	}

	adminVal := "false"
	if isSuperAdmin {
		adminVal = "true"
	}
	_, err = tx.ExecContext(ctx, "SELECT set_config('app.is_super_admin', $1, true)", adminVal)
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// RunInTenantSqlxTx runs a function in an sqlx transaction with tenant context set
func RunInTenantSqlxTx(ctx context.Context, tenantID uuid.UUID, isSuperAdmin bool, db *sqlx.DB, fn func(tx *sqlx.Tx) error) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID.String())
	if err != nil {
		return err
	}

	adminVal := "false"
	if isSuperAdmin {
		adminVal = "true"
	}
	_, err = tx.ExecContext(ctx, "SELECT set_config('app.is_super_admin', $1, true)", adminVal)
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// RunInTenantPgxTx runs a function in a pgx transaction with tenant context set
func RunInTenantPgxTx(ctx context.Context, tenantID uuid.UUID, isSuperAdmin bool, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID.String())
	if err != nil {
		return err
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
