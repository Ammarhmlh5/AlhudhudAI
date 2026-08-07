package accounting

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

// recordMovement applies a signed movement to a cashbox and records the ledger
// line. balanceBefore must already be the locked cashbox balance.
func (m *Manager) recordMovement(
	ctx context.Context,
	tx *sqlx.Tx,
	voucherType string,
	voucherID uuid.UUID,
	voucherNumber string,
	cashboxID uuid.UUID,
	amount decimal.Decimal,
	balanceBefore decimal.Decimal,
	description *string,
	createdBy *uuid.UUID,
) error {
	balanceAfter := balanceBefore.Add(amount)

	if _, err := tx.ExecContext(ctx,
		`UPDATE cashboxes SET current_balance = $2, updated_at = NOW() WHERE id = $1`,
		cashboxID, balanceAfter,
	); err != nil {
		return fmt.Errorf("failed to update cashbox balance: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO cashbox_movements (
			cashbox_id, voucher_type, voucher_id, voucher_number, movement_date,
			amount, balance_after, description, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, cashboxID, voucherType, voucherID, voucherNumber, time.Now(),
		amount, balanceAfter, strArg(description), uuidArg(createdBy),
	); err != nil {
		return fmt.Errorf("failed to record cashbox movement: %w", err)
	}

	return nil
}

// recordReversalMovement records the opposite movement for a reversal voucher.
func (m *Manager) recordReversalMovement(
	ctx context.Context,
	tx *sqlx.Tx,
	voucherType string,
	voucherID uuid.UUID,
	voucherNumber string,
	cashboxID uuid.UUID,
	amount decimal.Decimal,
	balanceBefore decimal.Decimal,
	createdBy *uuid.UUID,
) error {
	return m.recordMovement(ctx, tx, voucherType, voucherID, voucherNumber, cashboxID, amount, balanceBefore, nil, createdBy)
}

func receiptToView(v *ReceiptVoucher) *VoucherView {
	return &VoucherView{
		ID:              v.ID,
		VoucherNumber:   v.VoucherNumber,
		VoucherType:     VoucherTypeReceipt,
		VoucherDate:     v.VoucherDate,
		CashboxID:       v.CashboxID,
		Amount:          v.Amount,
		Currency:        v.Currency,
		PartyName:       v.PayerName,
		PartyType:       v.PayerType,
		TenantID:        v.TenantID,
		ResellerID:      v.ResellerID,
		PaymentMethod:   v.PaymentMethod,
		ReferenceNumber: v.ReferenceNumber,
		Description:     v.Description,
		Status:          v.Status,
		ReversalOf:      v.ReversalOf,
		CreatedBy:       v.CreatedBy,
		CreatedAt:       v.CreatedAt,
		PostedAt:        v.PostedAt,
		PostedBy:        v.PostedBy,
	}
}

func paymentToView(v *PaymentVoucher) *VoucherView {
	return &VoucherView{
		ID:              v.ID,
		VoucherNumber:   v.VoucherNumber,
		VoucherType:     VoucherTypePayment,
		VoucherDate:     v.VoucherDate,
		CashboxID:       v.CashboxID,
		Amount:          v.Amount,
		Currency:        v.Currency,
		PartyName:       v.PayeeName,
		PartyType:       v.PayeeType,
		TenantID:        v.TenantID,
		ResellerID:      v.ResellerID,
		PaymentMethod:   v.PaymentMethod,
		ReferenceNumber: v.ReferenceNumber,
		Description:     v.Description,
		Status:          v.Status,
		ReversalOf:      v.ReversalOf,
		CreatedBy:       v.CreatedBy,
		CreatedAt:       v.CreatedAt,
		PostedAt:        v.PostedAt,
		PostedBy:        v.PostedBy,
	}
}

// parseVoucherDate parses a date string (YYYY-MM-DD) or returns today.
func parseVoucherDate(s string) (time.Time, error) {
	if s == "" {
		return time.Now(), nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid voucher_date, expected YYYY-MM-DD: %w", err)
	}
	return t, nil
}

// parseUUIDPtr parses an optional UUID string pointer.
func parseUUIDPtr(s *string) *uuid.UUID {
	if s == nil || *s == "" {
		return nil
	}
	id, err := uuid.Parse(*s)
	if err != nil {
		return nil
	}
	return &id
}

// defaultString returns val when non-empty, otherwise fallback.
func defaultString(val, fallback string) string {
	if val == "" {
		return fallback
	}
	return val
}

// reversalDescription builds the description for a reversal voucher.
func reversalDescription(prefix, originalNumber string, originalDesc *string) *string {
	s := prefix + " " + originalNumber
	if originalDesc != nil && *originalDesc != "" {
		s += " - " + *originalDesc
	}
	return &s
}

// listVouchers lists receipts or payments with filtering and pagination.
func (m *Manager) listVouchers(ctx context.Context, voucherType string, filter *VoucherFilter) ([]*VoucherView, int64, error) {
	table := "receipt_vouchers"
	columns := receiptColumns
	if voucherType == VoucherTypePayment {
		table = "payment_vouchers"
		columns = paymentColumns
	}

	where := " WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if filter.CashboxID != nil {
		where += fmt.Sprintf(" AND cashbox_id = $%d", argIndex)
		args = append(args, *filter.CashboxID)
		argIndex++
	}
	if filter.Status != nil {
		where += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *filter.Status)
		argIndex++
	}
	if filter.From != nil {
		where += fmt.Sprintf(" AND voucher_date >= $%d", argIndex)
		args = append(args, *filter.From)
		argIndex++
	}
	if filter.To != nil {
		where += fmt.Sprintf(" AND voucher_date <= $%d", argIndex)
		args = append(args, *filter.To)
		argIndex++
	}

	var total int64
	if err := m.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM `+table+where, args...); err != nil {
		return nil, 0, fmt.Errorf("failed to count vouchers: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(`SELECT %s FROM %s%s ORDER BY voucher_date DESC, created_at DESC LIMIT $%d OFFSET $%d`,
		columns, table, where, argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := m.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query vouchers: %w", err)
	}
	defer rows.Close()

	vouchers := []*VoucherView{}
	for rows.Next() {
		var v *VoucherView
		if voucherType == VoucherTypePayment {
			pv, err := scanPayment(rows)
			if err != nil {
				return nil, 0, err
			}
			v = paymentToView(pv)
		} else {
			rv, err := scanReceipt(rows)
			if err != nil {
				return nil, 0, err
			}
			v = receiptToView(rv)
		}
		vouchers = append(vouchers, v)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating vouchers: %w", err)
	}

	return vouchers, total, nil
}
