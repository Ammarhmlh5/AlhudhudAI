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

// ReceiptRequest defines the fields for creating a receipt voucher.
type ReceiptRequest struct {
	VoucherDate     string  `json:"voucher_date"`
	CashboxID       string  `json:"cashbox_id"`
	Amount          string  `json:"amount"`
	Currency        string  `json:"currency"`
	PayerName       string  `json:"payer_name"`
	PayerType       string  `json:"payer_type"`
	TenantID        *string `json:"tenant_id"`
	ResellerID      *string `json:"reseller_id"`
	PaymentMethod   string  `json:"payment_method"`
	ReferenceNumber *string `json:"reference_number"`
	Description     *string `json:"description"`
	PostNow         bool    `json:"post_now"`
}

// VoucherFilter is used to list vouchers.
type VoucherFilter struct {
	CashboxID *uuid.UUID `json:"cashbox_id"`
	Status    *string    `json:"status"`
	From      *time.Time `json:"from"`
	To        *time.Time `json:"to"`
	Limit     int        `json:"limit"`
	Offset    int        `json:"offset"`
}

// CreateReceipt creates a receipt voucher. When req.PostNow is true the voucher
// is posted immediately: number allocated, balance credited, movement recorded.
func (m *Manager) CreateReceipt(ctx context.Context, req *ReceiptRequest, createdBy *uuid.UUID) (*ReceiptVoucher, error) {
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
	if req.PayerName == "" {
		return nil, fmt.Errorf("payer_name is required")
	}

	var voucher *ReceiptVoucher
	err = m.runInTx(ctx, func(tx *sqlx.Tx) error {
		v, err := m.createReceiptInTx(ctx, tx, req, cashboxID, amount, createdBy)
		voucher = v
		return err
	})
	if err != nil {
		return nil, err
	}
	return voucher, nil
}

func (m *Manager) createReceiptInTx(ctx context.Context, tx *sqlx.Tx, req *ReceiptRequest, cashboxID uuid.UUID, amount decimal.Decimal, createdBy *uuid.UUID) (*ReceiptVoucher, error) {
	// If posting immediately, lock the cashbox first to get a valid balance.
	var balance decimal.Decimal
	if req.PostNow {
		b, err := m.lockCashbox(ctx, tx, cashboxID)
		if err != nil {
			return nil, err
		}
		balance = b
	}

	number, err := m.nextVoucherNumber(ctx, tx, VoucherTypeReceipt, voucherYear(time.Now()))
	if err != nil {
		return nil, err
	}

	voucherDate, err := parseVoucherDate(req.VoucherDate)
	if err != nil {
		return nil, err
	}

	creditWallet := req.PostNow && req.TenantID != nil && parseUUIDPtr(req.TenantID) != nil

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
		INSERT INTO receipt_vouchers (
			voucher_number, voucher_date, cashbox_id, amount, currency, payer_name,
			payer_type, tenant_id, reseller_id, payment_method, reference_number,
			description, status, credited_wallet, created_by, posted_at, posted_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id
	`

	var voucherID uuid.UUID
	err = tx.QueryRowxContext(ctx, query,
		number, voucherDate, cashboxID, amount, defaultString(req.Currency, "SAR"),
		req.PayerName, defaultString(req.PayerType, "OTHER"),
		parseUUIDPtr(req.TenantID), parseUUIDPtr(req.ResellerID),
		defaultString(req.PaymentMethod, "CASH"),
		strArg(req.ReferenceNumber), strArg(req.Description),
		status, creditWallet, uuidArg(createdBy), postedAt, uuidArg(postedBy),
	).Scan(&voucherID)
	if err != nil {
		return nil, fmt.Errorf("failed to insert receipt voucher: %w", err)
	}

	voucher := &ReceiptVoucher{
		ID:              voucherID,
		VoucherNumber:   number,
		VoucherDate:     voucherDate,
		CashboxID:       cashboxID,
		Amount:          amount,
		Currency:        defaultString(req.Currency, "SAR"),
		PayerName:       req.PayerName,
		PayerType:       defaultString(req.PayerType, "OTHER"),
		TenantID:        parseUUIDPtr(req.TenantID),
		ResellerID:      parseUUIDPtr(req.ResellerID),
		PaymentMethod:   defaultString(req.PaymentMethod, "CASH"),
		ReferenceNumber: req.ReferenceNumber,
		Description:     req.Description,
		Status:          status,
		CreditedWallet:  creditWallet,
		CreatedBy:       createdBy,
		CreatedAt:       time.Now(),
		PostedAt:        postedAt,
		PostedBy:        postedBy,
	}

	if req.PostNow {
		if err := m.recordMovement(ctx, tx, VoucherTypeReceipt, voucherID, number, cashboxID, amount, balance, req.Description, createdBy); err != nil {
			return nil, err
		}
		if creditWallet {
			if err := m.creditTenantWallet(ctx, *parseUUIDPtr(req.TenantID), amount, walletReason(VoucherTypeReceipt, number), createdBy); err != nil {
				return nil, err
			}
		}
		// Generate double-entry journal entry
		desc := walletReason(VoucherTypeReceipt, number)
		if req.Description != nil {
			desc = *req.Description
		}
		if err := m.GenerateJournalEntryFromVoucher(ctx, "RECEIPT", voucherID, number, amount, cashboxID, parseUUIDPtr(req.TenantID), desc); err != nil {
			fmt.Printf("WARN: failed to generate journal entry for receipt %s: %v\n", number, err)
		}
	}

	return voucher, nil
}

// PostReceipt transitions a DRAFT receipt voucher to POSTED, crediting the cashbox.
func (m *Manager) PostReceipt(ctx context.Context, voucherID uuid.UUID, userID *uuid.UUID) (*ReceiptVoucher, error) {
	var voucher *ReceiptVoucher
	err := m.runInTx(ctx, func(tx *sqlx.Tx) error {
		// Lock the voucher row to prevent double-posting.
		var orig ReceiptVoucher
		if err := lockReceiptForUpdate(ctx, tx, voucherID, &orig); err != nil {
			return err
		}
		if orig.Status != VoucherStatusDraft {
			return ErrVoucherNotPostable
		}

		balance, err := m.lockCashbox(ctx, tx, orig.CashboxID)
		if err != nil {
			return err
		}

		now := time.Now()
		_, err = tx.ExecContext(ctx, `
			UPDATE receipt_vouchers
			SET status = $2, posted_at = $3, posted_by = $4, credited_wallet = $5
			WHERE id = $1
		`, voucherID, VoucherStatusPosted, now, uuidArg(userID), orig.TenantID != nil)
		if err != nil {
			return fmt.Errorf("failed to post receipt voucher: %w", err)
		}

		if err := m.recordMovement(ctx, tx, VoucherTypeReceipt, voucherID, orig.VoucherNumber, orig.CashboxID, orig.Amount, balance, orig.Description, userID); err != nil {
			return err
		}

		if orig.TenantID != nil {
			if err := m.creditTenantWallet(ctx, *orig.TenantID, orig.Amount, walletReason(VoucherTypeReceipt, orig.VoucherNumber), userID); err != nil {
				return err
			}
		}

		// Generate double-entry journal entry
		desc := walletReason(VoucherTypeReceipt, orig.VoucherNumber)
		if orig.Description != nil {
			desc = *orig.Description
		}
		if err := m.GenerateJournalEntryFromVoucher(ctx, "RECEIPT", voucherID, orig.VoucherNumber, orig.Amount, orig.CashboxID, orig.TenantID, desc); err != nil {
			// Log but don't fail - journal entry is not critical for voucher posting
			// In production, you might want to handle this differently
			fmt.Printf("WARN: failed to generate journal entry for receipt %s: %v\n", orig.VoucherNumber, err)
		}

		voucher, err = m.GetReceipt(ctx, voucherID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return voucher, nil
}

// ReverseReceipt reverses a POSTED receipt voucher: it deducts the amount back
// from the cashbox and creates a reversal voucher linked to the original.
func (m *Manager) ReverseReceipt(ctx context.Context, voucherID uuid.UUID, userID *uuid.UUID) (*ReceiptVoucher, error) {
	var reversal *ReceiptVoucher
	err := m.runInTx(ctx, func(tx *sqlx.Tx) error {
		var orig ReceiptVoucher
		err := lockReceiptForUpdate(ctx, tx, voucherID, &orig)
		if err != nil {
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
		if balance.LessThan(orig.Amount) {
			return ErrInsufficientBalance
		}

		number, err := m.nextVoucherNumber(ctx, tx, VoucherTypeReceipt, voucherYear(time.Now()))
		if err != nil {
			return err
		}

		now := time.Now()
		var reversalID uuid.UUID
		err = tx.QueryRowxContext(ctx, `
			INSERT INTO receipt_vouchers (
				voucher_number, voucher_date, cashbox_id, amount, currency, payer_name,
				payer_type, tenant_id, reseller_id, payment_method, reference_number,
				description, status, reversal_of, created_by, posted_at, posted_by
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
			RETURNING id
		`,
			number, orig.VoucherDate, orig.CashboxID, orig.Amount, orig.Currency,
			orig.PayerName, orig.PayerType, uuidArg(orig.TenantID), uuidArg(orig.ResellerID),
			orig.PaymentMethod, strArg(orig.ReferenceNumber),
			reversalDescription("إلغاء سند قبض", orig.VoucherNumber, orig.Description),
			VoucherStatusPosted, voucherID, uuidArg(userID), now, uuidArg(userID),
		).Scan(&reversalID)
		if err != nil {
			return fmt.Errorf("failed to insert receipt reversal: %w", err)
		}

		// Deduct the amount back (negative movement).
		if err := m.recordReversalMovement(ctx, tx, VoucherTypeReceipt, reversalID, number, orig.CashboxID, orig.Amount.Neg(), balance, userID); err != nil {
			return err
		}

		// Compensate the wallet credit made when the receipt was posted.
		if orig.TenantID != nil {
			if err := m.debitTenantWallet(ctx, *orig.TenantID, orig.Amount, walletReversalReason(VoucherTypeReceipt, orig.VoucherNumber), userID); err != nil {
				return err
			}
		}

		_, err = tx.ExecContext(ctx, `
			UPDATE receipt_vouchers
			SET status = $2, reversed_at = $3, reversed_by = $4
			WHERE id = $1
		`, voucherID, VoucherStatusReversed, now, uuidArg(userID))
		if err != nil {
			return fmt.Errorf("failed to mark original receipt reversed: %w", err)
		}

		reversal, err = m.GetReceipt(ctx, reversalID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return reversal, nil
}

// GetReceipt returns a receipt voucher by ID.
func (m *Manager) GetReceipt(ctx context.Context, voucherID uuid.UUID) (*ReceiptVoucher, error) {
	v, err := scanReceipt(m.db.QueryRowxContext(ctx,
		receiptColumns+` FROM receipt_vouchers WHERE id = $1`, voucherID))
	if err != nil {
		return nil, err
	}
	return v, nil
}

// ListReceipts returns receipt vouchers matching the filter.
func (m *Manager) ListReceipts(ctx context.Context, filter *VoucherFilter) ([]*VoucherView, int64, error) {
	return m.listVouchers(ctx, VoucherTypeReceipt, filter)
}

const receiptColumns = `
	id, voucher_number, voucher_date, cashbox_id, amount, currency, payer_name,
	payer_type, tenant_id, reseller_id, payment_method, reference_number,
	description, status, reversal_of, credited_wallet, created_by, created_at,
	posted_at, posted_by, reversed_at, reversed_by
`

// scanReceipt scans one receipt voucher row.
func scanReceipt(row scanner) (*ReceiptVoucher, error) {
	var v ReceiptVoucher
	var tenantID, resellerID, referenceNumber, description, reversalOf, createdBy, postedBy, reversedBy sql.NullString
	var postedAt, reversedAt sql.NullTime

	err := row.Scan(
		&v.ID, &v.VoucherNumber, &v.VoucherDate, &v.CashboxID, &v.Amount, &v.Currency,
		&v.PayerName, &v.PayerType, &tenantID, &resellerID, &v.PaymentMethod,
		&referenceNumber, &description, &v.Status, &reversalOf, &v.CreditedWallet,
		&createdBy, &v.CreatedAt, &postedAt, &postedBy, &reversedAt, &reversedBy,
	)
	if err == sql.ErrNoRows {
		return nil, ErrVoucherNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan receipt voucher: %w", err)
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

func lockReceiptForUpdate(ctx context.Context, tx *sqlx.Tx, id uuid.UUID, v *ReceiptVoucher) error {
	row := tx.QueryRowxContext(ctx, `SELECT `+receiptColumns+` FROM receipt_vouchers WHERE id = $1 FOR UPDATE`, id)
	return scanRowIntoReceipt(row, v)
}

func scanRowIntoReceipt(row scanner, v *ReceiptVoucher) error {
	scanned, err := scanReceipt(row)
	if err != nil {
		return err
	}
	*v = *scanned
	return nil
}
