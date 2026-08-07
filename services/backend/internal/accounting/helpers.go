package accounting

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// scanner is satisfied by both *sqlx.Row and *sqlx.Rows.
type scanner interface {
	Scan(dest ...interface{}) error
}

// scanCashbox scans one row into a Cashbox.
func scanCashbox(row scanner) (*Cashbox, error) {
	var c Cashbox
	var nameEN, bankName, accountHolder, accountNumber, iban sql.NullString
	var description sql.NullString
	var createdBy sql.NullString

	err := row.Scan(
		&c.ID, &c.Code, &c.Name, &nameEN, &c.Type, &c.Currency, &bankName,
		&accountHolder, &accountNumber, &iban, &c.OpeningBalance, &c.CurrentBalance,
		&c.MinBalance, &c.Status, &description, &createdBy, &c.CreatedAt, &c.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrCashboxNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan cashbox: %w", err)
	}

	c.NameEN = strOrNil(nameEN)
	c.BankName = strOrNil(bankName)
	c.AccountHolder = strOrNil(accountHolder)
	c.AccountNumber = strOrNil(accountNumber)
	c.IBAN = strOrNil(iban)
	c.Description = strOrNil(description)
	c.CreatedBy = uuidOrNil(createdBy)
	return &c, nil
}

// parseDecimal parses a numeric string into a decimal, treating "" as zero.
func parseDecimal(s, field string) (decimal.Decimal, error) {
	if s == "" {
		return decimal.Zero, nil
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero, fmt.Errorf("invalid %s: %w", field, err)
	}
	return d, nil
}

// uuidArg converts a *uuid.UUID to a driver-safe arg (nil -> NULL).
func uuidArg(v *uuid.UUID) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

// strArg converts a *string to a driver-safe arg (nil -> NULL).
func strArg(v *string) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

func zapString(key, val string) zap.Field {
	return zap.String(key, val)
}
