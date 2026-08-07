package accounting

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CreateCashboxRequest defines the fields needed to create/update a cashbox.
type CreateCashboxRequest struct {
	Code           string  `json:"code"`
	Name           string  `json:"name"`
	NameEN         *string `json:"name_en"`
	Type           string  `json:"type"`
	Currency       string  `json:"currency"`
	BankName       *string `json:"bank_name"`
	AccountHolder  *string `json:"account_holder"`
	AccountNumber  *string `json:"account_number"`
	IBAN           *string `json:"iban"`
	OpeningBalance string  `json:"opening_balance"`
	MinBalance     string  `json:"min_balance"`
	Description    *string `json:"description"`
}

const cashboxColumns = `
	id, code, name, name_en, type, currency, bank_name, account_holder,
	account_number, iban, opening_balance, current_balance, min_balance,
	status, description, created_by, created_at, updated_at
`

// CreateCashbox creates a new cashbox.
func (m *Manager) CreateCashbox(ctx context.Context, req *CreateCashboxRequest, createdBy *uuid.UUID) (*Cashbox, error) {
	if err := m.validateCashboxRequest(req); err != nil {
		return nil, err
	}

	opening, err := parseDecimal(req.OpeningBalance, "opening_balance")
	if err != nil {
		return nil, err
	}
	minBal, err := parseDecimal(req.MinBalance, "min_balance")
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
		INSERT INTO cashboxes (
			code, name, name_en, type, currency, bank_name, account_holder,
			account_number, iban, opening_balance, current_balance, min_balance,
			status, description, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING %s
	`, cashboxColumns)

	row := m.db.QueryRowxContext(ctx, query,
		req.Code, req.Name, strArg(req.NameEN), req.Type, req.Currency, strArg(req.BankName),
		strArg(req.AccountHolder), strArg(req.AccountNumber), strArg(req.IBAN), opening, opening, minBal,
		CashboxStatusActive, strArg(req.Description), uuidArg(createdBy),
	)

	cashbox, err := scanCashbox(row)
	if err != nil {
		return nil, err
	}
	m.logger.Info("Cashbox created", zapString("code", cashbox.Code))
	return cashbox, nil
}

// ListCashboxes returns all cashboxes.
func (m *Manager) ListCashboxes(ctx context.Context) ([]Cashbox, error) {
	query := fmt.Sprintf(`SELECT %s FROM cashboxes ORDER BY created_at ASC`, cashboxColumns)

	rows, err := m.db.QueryxContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query cashboxes: %w", err)
	}
	defer rows.Close()

	cashboxes := []Cashbox{}
	for rows.Next() {
		c, err := scanCashbox(rows)
		if err != nil {
			return nil, err
		}
		cashboxes = append(cashboxes, *c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating cashboxes: %w", err)
	}
	return cashboxes, nil
}

// GetCashbox returns a single cashbox by ID.
func (m *Manager) GetCashbox(ctx context.Context, id uuid.UUID) (*Cashbox, error) {
	query := fmt.Sprintf(`SELECT %s FROM cashboxes WHERE id = $1`, cashboxColumns)
	c, err := scanCashbox(m.db.QueryRowxContext(ctx, query, id))
	if err != nil {
		return nil, err
	}
	return c, nil
}

// UpdateCashbox updates mutable fields of a cashbox.
func (m *Manager) UpdateCashbox(ctx context.Context, id uuid.UUID, req *CreateCashboxRequest) (*Cashbox, error) {
	if req.Name != "" || req.Code != "" {
		// Only re-validate when identity fields are being changed.
		if err := m.validateCashboxRequest(req); err != nil {
			return nil, err
		}
	}

	minBal, err := parseDecimal(req.MinBalance, "min_balance")
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
		UPDATE cashboxes SET
			code = COALESCE($2, code),
			name = COALESCE($3, name),
			name_en = $4,
			type = COALESCE($5, type),
			currency = COALESCE($6, currency),
			bank_name = $7,
			account_holder = $8,
			account_number = $9,
			iban = $10,
			min_balance = $11,
			description = $12,
			updated_at = NOW()
		WHERE id = $1
		RETURNING %s
	`, cashboxColumns)

	row := m.db.QueryRowxContext(ctx, query,
		id, req.Code, req.Name, strArg(req.NameEN), req.Type, req.Currency, strArg(req.BankName),
		strArg(req.AccountHolder), strArg(req.AccountNumber), strArg(req.IBAN), minBal, strArg(req.Description),
	)

	c, err := scanCashbox(row)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// SetCashboxStatus activates or deactivates a cashbox.
func (m *Manager) SetCashboxStatus(ctx context.Context, id uuid.UUID, status string) (*Cashbox, error) {
	if status != CashboxStatusActive && status != CashboxStatusInactive {
		return nil, fmt.Errorf("invalid status: %s", status)
	}

	query := fmt.Sprintf(`
		UPDATE cashboxes SET status = $2, updated_at = NOW() WHERE id = $1 RETURNING %s
	`, cashboxColumns)

	c, err := scanCashbox(m.db.QueryRowxContext(ctx, query, id, status))
	if err != nil {
		return nil, err
	}
	return c, nil
}

// GetStatement returns the cashbox detail and its movements in the given range.
func (m *Manager) GetStatement(ctx context.Context, id uuid.UUID, from, to *time.Time) (*CashboxStatement, error) {
	cashbox, err := m.GetCashbox(ctx, id)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, cashbox_id, voucher_type, voucher_id, voucher_number,
			movement_date, amount, balance_after, description, created_by
		FROM cashbox_movements
		WHERE cashbox_id = $1
	`
	args := []interface{}{id}
	argIndex := 2

	if from != nil {
		query += fmt.Sprintf(" AND movement_date >= $%d", argIndex)
		args = append(args, *from)
		argIndex++
	}
	if to != nil {
		query += fmt.Sprintf(" AND movement_date <= $%d", argIndex)
		args = append(args, *to)
		argIndex++
	}
	query += " ORDER BY movement_date ASC, id ASC"

	rows, err := m.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query movements: %w", err)
	}
	defer rows.Close()

	movements := []CashboxMovement{}
	totals := StatementTotals{}
	for rows.Next() {
		var mv CashboxMovement
		var desc sql.NullString
		var createdBy sql.NullString
		if err := rows.Scan(
			&mv.ID, &mv.CashboxID, &mv.VoucherType, &mv.VoucherID, &mv.VoucherNumber,
			&mv.MovementDate, &mv.Amount, &mv.BalanceAfter, &desc, &createdBy,
		); err != nil {
			return nil, fmt.Errorf("failed to scan movement: %w", err)
		}
		mv.Description = strOrNil(desc)
		mv.CreatedBy = uuidOrNil(createdBy)

		if mv.Amount.IsPositive() {
			totals.TotalReceipts = totals.TotalReceipts.Add(mv.Amount)
		} else {
			totals.TotalPayments = totals.TotalPayments.Add(mv.Amount.Abs())
		}
		movements = append(movements, mv)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating movements: %w", err)
	}

	totals.Net = totals.TotalReceipts.Sub(totals.TotalPayments)

	return &CashboxStatement{
		Cashbox:   *cashbox,
		Movements: movements,
		Totals:    totals,
	}, nil
}

func (m *Manager) validateCashboxRequest(req *CreateCashboxRequest) error {
	if req.Code == "" {
		return fmt.Errorf("code is required")
	}
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	validTypes := map[string]bool{
		CashboxTypeCash:   true,
		CashboxTypeBank:   true,
		CashboxTypeWallet: true,
		CashboxTypeOther:  true,
	}
	if req.Type == "" {
		req.Type = CashboxTypeCash
	}
	if !validTypes[req.Type] {
		return fmt.Errorf("invalid cashbox type: %s", req.Type)
	}
	if req.Currency == "" {
		req.Currency = "SAR"
	}
	return nil
}
