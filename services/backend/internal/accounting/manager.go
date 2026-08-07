package accounting

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/al-hudhud/backend/internal/balance"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// Sentinel errors returned by the accounting manager.
var (
	ErrCashboxNotFound           = errors.New("cashbox not found")
	ErrCashboxInactive           = errors.New("cashbox is inactive")
	ErrCashboxCodeExists         = errors.New("cashbox code already exists")
	ErrVoucherNotFound           = errors.New("voucher not found")
	ErrVoucherNotPostable        = errors.New("voucher is not in DRAFT status")
	ErrVoucherNotReversable      = errors.New("only POSTED vouchers can be reversed")
	ErrVoucherAlreadyReversed    = errors.New("voucher has already been reversed")
	ErrInsufficientBalance       = errors.New("insufficient cashbox balance")
	ErrInsufficientWalletBalance = errors.New("insufficient wallet balance")
	ErrInvalidVoucherType        = errors.New("invalid voucher type")
)

// Manager handles cashboxes and voucher operations atomically.
type Manager struct {
	db      *sqlx.DB
	balance balance.Manager
	logger  *zap.Logger
}

// NewManager creates a new accounting manager.
func NewManager(db *sqlx.DB, balance balance.Manager, logger *zap.Logger) *Manager {
	return &Manager{
		db:      db,
		balance: balance,
		logger:  logger,
	}
}

// runInTx executes fn inside a database transaction, rolling back on error.
func (m *Manager) runInTx(ctx context.Context, fn func(tx *sqlx.Tx) error) error {
	tx, err := m.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			m.logger.Warn("Failed to rollback accounting transaction", zap.Error(err))
		}
	}()

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// lockCashbox locks the cashbox row FOR UPDATE and returns its current balance,
// validating that the cashbox exists and is active.
func (m *Manager) lockCashbox(ctx context.Context, tx *sqlx.Tx, cashboxID uuid.UUID) (decimal.Decimal, error) {
	var balance decimal.Decimal
	var status string
	err := tx.QueryRowxContext(ctx,
		`SELECT current_balance, status FROM cashboxes WHERE id = $1 FOR UPDATE`,
		cashboxID,
	).Scan(&balance, &status)
	if err == sql.ErrNoRows {
		return decimal.Zero, ErrCashboxNotFound
	}
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to lock cashbox: %w", err)
	}
	if status != CashboxStatusActive {
		return decimal.Zero, ErrCashboxInactive
	}
	return balance, nil
}

// voucherYear returns the fiscal year for a voucher (currently the calendar year).
func voucherYear(t time.Time) int {
	return t.Year()
}

// nextVoucherNumber atomically allocates the next voucher number for the given
// voucher type and fiscal year using an INSERT ... ON CONFLICT upsert.
func (m *Manager) nextVoucherNumber(ctx context.Context, tx *sqlx.Tx, voucherType string, year int) (string, error) {
	prefix := ""
	switch voucherType {
	case VoucherTypeReceipt:
		prefix = "QABD"
	case VoucherTypePayment:
		prefix = "SARF"
	default:
		return "", ErrInvalidVoucherType
	}

	var lastNumber int
	err := tx.QueryRowxContext(ctx, `
		INSERT INTO voucher_counters (voucher_type, fiscal_year, last_number)
		VALUES ($1, $2, 1)
		ON CONFLICT (voucher_type, fiscal_year)
		DO UPDATE SET last_number = voucher_counters.last_number + 1
		RETURNING last_number
	`, voucherType, year).Scan(&lastNumber)
	if err != nil {
		return "", fmt.Errorf("failed to allocate voucher number: %w", err)
	}

	return fmt.Sprintf("%s-%d-%06d", prefix, year, lastNumber), nil
}
