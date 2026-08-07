package accounting

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

// PaymentRequest defines the fields for creating a payment voucher.
type PaymentRequest struct {
	VoucherDate     string  `json:"voucher_date"`
	CashboxID       string  `json:"cashbox_id"`
	Amount          string  `json:"amount"`
	Currency        string  `json:"currency"`
	PayeeName       string  `json:"payee_name"`
	PayeeType       string  `json:"payee_type"`
	TenantID        *string `json:"tenant_id"`
	ResellerID      *string `json:"reseller_id"`
	PaymentMethod   string  `json:"payment_method"`
	ReferenceNumber *string `json:"reference_number"`
	Description     *string `json:"description"`
	PostNow         bool    `json:"post_now"`
}

// CreatePayment creates a payment voucher. When req.PostNow is true the voucher
// is posted immediately: the cashbox is debited and a movement is recorded.
func (m *Manager) CreatePayment(ctx context.Context, req *PaymentRequest, createdBy *uuid.UUID) (*PaymentVoucher, error) {
	cashboxID, err := uuid.Parse(req.CashboxID)
	if err != nil {
		return nil, fmt.Errorf("invalid cashbox_id: %w", err)
	}
	amount, err := parseDecimal(req.Amount, "amount")
	if err != nil {
		return nil, err
	}
	if !amount.IsPositive() {
		return nil, fmt.Errorf("amount must be greater than zero")
	}
	if req.PayeeName == "" {
		return nil, fmt.Errorf("payee_name is required")
	}

	var voucher *PaymentVoucher
	err = m.runInTx(ctx, func(tx *sqlx.Tx) error {
		v, err := m.createPaymentInTx(ctx, tx, req, cashboxID, amount, createdBy)
		voucher = v
		return err
	})
	if err != nil {
		return nil, err
	}
	return voucher, nil
}

func (m *Manager) createPaymentInTx(ctx context.Context, tx *sqlx.Tx, req *PaymentRequest, cashboxID uuid.UUID, amount decimal.Decimal, createdBy *uuid.UUID) (*PaymentVoucher, error) {
	// If posting immediately, lock the cashbox first and verify sufficient balance.
	var balance decimal.Decimal
	if req.PostNow {
		b, err := m.lockCashbox(ctx, tx, cashboxID)
		if err != nil {
			return nil, err
		}
		if b.LessThan(amount) {
			return nil, ErrInsufficientBalance
		}
		balance = b
	}

	number, err := m.nextVoucherNumber(ctx, tx, VoucherTypePayment, voucherYear(time.Now()))
	if err != nil {
		return nil, err
	}

	voucherDate, err := parseVoucherDate(req.VoucherDate)
	if err != nil {
		return nil, err
	}

	debitWallet := req.PostNow && req.TenantID != nil && parseUUIDPtr(req.TenantID) != nil

	status := VoucherStatusDraft
	var postedAt *time.Time
	var postedBy *uuid.UUID
	if req.PostNow {
		status = VoucherStatusPosted
		now := time.Now()
		postedAt = &now
		postedBy = createdBy
	}

	query := `
		INSERT INTO payment_vouchers (
			voucher_number, voucher_date, cashbox_id, amount, currency, payee_name,
			payee_type, tenant_id, reseller_id, payment_method, reference_number,
			description, status, debited_wallet, created_by, posted_at, posted_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id
	`

	var voucherID uuid.UUID
	err = tx.QueryRowxContext(ctx, query,
		number, voucherDate, cashboxID, amount, defaultString(req.Currency, "SAR"),
		req.PayeeName, defaultString(req.PayeeType, "OTHER"),
		parseUUIDPtr(req.TenantID), parseUUIDPtr(req.ResellerID),
		defaultString(req.PaymentMethod, "CASH"),
		strArg(req.ReferenceNumber), strArg(req.Description),
		status, debitWallet, uuidArg(createdBy), postedAt, uuidArg(postedBy),
	).Scan(&voucherID)
	if err != nil {
		return nil, fmt.Errorf("failed to insert payment voucher: %w", err)
	}

	voucher := &PaymentVoucher{
		ID:              voucherID,
		VoucherNumber:   number,
		VoucherDate:     voucherDate,
		CashboxID:       cashboxID,
		Amount:          amount,
		Currency:        defaultString(req.Currency, "SAR"),
		PayeeName:       req.PayeeName,
		PayeeType:       defaultString(req.PayeeType, "OTHER"),
		TenantID:        parseUUIDPtr(req.TenantID),
		ResellerID:      parseUUIDPtr(req.ResellerID),
		PaymentMethod:   defaultString(req.PaymentMethod, "CASH"),
		ReferenceNumber: req.ReferenceNumber,
		Description:     req.Description,
		Status:          status,
		DebitedWallet:   debitWallet,
		CreatedBy:       createdBy,
		CreatedAt:       time.Now(),
		PostedAt:        postedAt,
		PostedBy:        postedBy,
	}

	if req.PostNow {
		if err := m.recordMovement(ctx, tx, VoucherTypePayment, voucherID, number, cashboxID, amount.Neg(), balance, req.Description, createdBy); err != nil {
			return nil, err
		}
		if debitWallet {
			if err := m.debitTenantWallet(ctx, *parseUUIDPtr(req.TenantID), amount, walletReason(VoucherTypePayment, number), createdBy); err != nil {
				return nil, err
			}
		}
	}

	return voucher, nil
}

// PostPayment transitions a DRAFT payment voucher to POSTED, debiting the cashbox.
func (m *Manager) PostPayment(ctx context.Context, voucherID uuid.UUID, userID *uuid.UUID) (*PaymentVoucher, error) {
	var voucher *PaymentVoucher
	err := m.runInTx(ctx, func(tx *sqlx.Tx) error {
		// Lock the voucher row to prevent double-posting.
		var orig PaymentVoucher
		if err := lockPaymentForUpdate(ctx, tx, voucherID, &orig); err != nil {
			return err
		}
		if orig.Status != VoucherStatusDraft {
			return ErrVoucherNotPostable
		}

		balance, err := m.lockCashbox(ctx, tx, orig.CashboxID)
		if err != nil {
			return err
		}
		if balance.LessThan(orig.Amount) {
			return ErrInsufficientBalance
		}

		now := time.Now()
		_, err = tx.ExecContext(ctx, `
			UPDATE payment_vouchers
			SET status = $2, posted_at = $3, posted_by = $4, debited_wallet = $5
			WHERE id = $1
		`, voucherID, VoucherStatusPosted, now, uuidArg(userID), orig.TenantID != nil)
		if err != nil {
			return fmt.Errorf("failed to post payment voucher: %w", err)
		}

		if err := m.recordMovement(ctx, tx, VoucherTypePayment, voucherID, orig.VoucherNumber, orig.CashboxID, orig.Amount.Neg(), balance, orig.Description, userID); err != nil {
			return err
		}

		if orig.TenantID != nil {
			if err := m.debitTenantWallet(ctx, *orig.TenantID, orig.Amount, walletReason(VoucherTypePayment, orig.VoucherNumber), userID); err != nil {
				return err
			}
		}

		voucher, err = m.GetPayment(ctx, voucherID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return voucher, nil
}

// ReversePayment reverses a POSTED payment voucher: it credits the amount back
// to the cashbox and creates a reversal voucher linked to the original.
func (m *Manager) ReversePayment(ctx context.Context, voucherID uuid.UUID, userID *uuid.UUID) (*PaymentVoucher, error) {
	var reversal *PaymentVoucher
	err := m.runInTx(ctx, func(tx *sqlx.Tx) error {
		var orig PaymentVoucher
		if err := lockPaymentForUpdate(ctx, tx, voucherID, &orig); err != nil {
			return err
		}
		if orig.Status != VoucherStatusPosted {
			return ErrVoucherNotReversable
		}
		if orig.ReversalOf != nil {
			return ErrVoucherAlreadyReversed
		}

		balance, err := m.lockCashbox(ctx, tx, orig.CashboxID)
		if err != nil {
			return err
		}

		number, err := m.nextVoucherNumber(ctx, tx, VoucherTypePayment, voucherYear(time.Now()))
		if err != nil {
			return err
		}

		now := time.Now()
		var reversalID uuid.UUID
		err = tx.QueryRowxContext(ctx, `
			INSERT INTO payment_vouchers (
				voucher_number, voucher_date, cashbox_id, amount, currency, payee_name,
				payee_type, tenant_id, reseller_id, payment_method, reference_number,
				description, status, reversal_of, created_by, posted_at, posted_by
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
			RETURNING id
		`,
			number, orig.VoucherDate, orig.CashboxID, orig.Amount, orig.Currency,
			orig.PayeeName, orig.PayeeType, uuidArg(orig.TenantID), uuidArg(orig.ResellerID),
			orig.PaymentMethod, strArg(orig.ReferenceNumber),
			reversalDescription("إلغاء سند صرف", orig.VoucherNumber, orig.Description),
			VoucherStatusPosted, voucherID, uuidArg(userID), now, uuidArg(userID),
		).Scan(&reversalID)
		if err != nil {
			return fmt.Errorf("failed to insert payment reversal: %w", err)
		}

		// Credit the amount back (positive movement).
		if err := m.recordReversalMovement(ctx, tx, VoucherTypePayment, reversalID, number, orig.CashboxID, orig.Amount, balance, userID); err != nil {
			return err
		}

		// Compensate the wallet deduction made when the payment was posted.
		if orig.TenantID != nil {
			if err := m.creditTenantWallet(ctx, *orig.TenantID, orig.Amount, walletReversalReason(VoucherTypePayment, orig.VoucherNumber), userID); err != nil {
				return err
			}
		}

		_, err = tx.ExecContext(ctx, `
			UPDATE payment_vouchers
			SET status = $2, reversed_at = $3, reversed_by = $4
			WHERE id = $1
		`, voucherID, VoucherStatusReversed, now, uuidArg(userID))
		if err != nil {
			return fmt.Errorf("failed to mark original payment reversed: %w", err)
		}

		reversal, err = m.GetPayment(ctx, reversalID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return reversal, nil
}

// GetPayment returns a payment voucher by ID.
func (m *Manager) GetPayment(ctx context.Context, voucherID uuid.UUID) (*PaymentVoucher, error) {
	v, err := scanPayment(m.db.QueryRowxContext(ctx,
		`SELECT `+paymentColumns+` FROM payment_vouchers WHERE id = $1`, voucherID))
	if err != nil {
		return nil, err
	}
	return v, nil
}

// ListPayments returns payment vouchers matching the filter.
func (m *Manager) ListPayments(ctx context.Context, filter *VoucherFilter) ([]*VoucherView, int64, error) {
	return m.listVouchers(ctx, VoucherTypePayment, filter)
}

// PreviewNextNumber returns the next voucher number without consuming it.
func (m *Manager) PreviewNextNumber(ctx context.Context, voucherType string) (string, error) {
	prefix := ""
	switch voucherType {
	case VoucherTypeReceipt:
		prefix = "QABD"
	case VoucherTypePayment:
		prefix = "SARF"
	default:
		return "", ErrInvalidVoucherType
	}

	year := voucherYear(time.Now())
	var lastNumber int
	err := m.db.GetContext(ctx, &lastNumber,
		`SELECT last_number FROM voucher_counters WHERE voucher_type = $1 AND fiscal_year = $2`,
		voucherType, year,
	)
	if err != nil && err != sql.ErrNoRows {
		return "", fmt.Errorf("failed to read voucher counter: %w", err)
	}

	return fmt.Sprintf("%s-%d-%06d", prefix, year, lastNumber+1), nil
}

const paymentColumns = `
	id, voucher_number, voucher_date, cashbox_id, amount, currency, payee_name,
	payee_type, tenant_id, reseller_id, payment_method, reference_number,
	description, status, reversal_of, debited_wallet, created_by, created_at,
	posted_at, posted_by, reversed_at, reversed_by
`

// scanPayment scans one payment voucher row.
func scanPayment(row scanner) (*PaymentVoucher, error) {
	var v PaymentVoucher
	var tenantID, resellerID, referenceNumber, description, reversalOf, createdBy, postedBy, reversedBy sql.NullString
	var postedAt, reversedAt sql.NullTime

	err := row.Scan(
		&v.ID, &v.VoucherNumber, &v.VoucherDate, &v.CashboxID, &v.Amount, &v.Currency,
		&v.PayeeName, &v.PayeeType, &tenantID, &resellerID, &v.PaymentMethod,
		&referenceNumber, &description, &v.Status, &reversalOf, &v.DebitedWallet,
		&createdBy, &v.CreatedAt, &postedAt, &postedBy, &reversedAt, &reversedBy,
	)
	if err == sql.ErrNoRows {
		return nil, ErrVoucherNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan payment voucher: %w", err)
	}

	v.TenantID = uuidOrNil(tenantID)
	v.ResellerID = uuidOrNil(resellerID)
	v.ReferenceNumber = strOrNil(referenceNumber)
	v.Description = strOrNil(description)
	v.ReversalOf = uuidOrNil(reversalOf)
	v.CreatedBy = uuidOrNil(createdBy)
	v.PostedBy = uuidOrNil(postedBy)
	v.ReversedBy = uuidOrNil(reversedBy)
	if postedAt.Valid {
		v.PostedAt = &postedAt.Time
	}
	if reversedAt.Valid {
		v.ReversedAt = &reversedAt.Time
	}
	return &v, nil
}

func lockPaymentForUpdate(ctx context.Context, tx *sqlx.Tx, id uuid.UUID, v *PaymentVoucher) error {
	row := tx.QueryRowxContext(ctx, `SELECT `+paymentColumns+` FROM payment_vouchers WHERE id = $1 FOR UPDATE`, id)
	scanned, err := scanPayment(row)
	if err != nil {
		return err
	}
	*v = *scanned
	return nil
}
