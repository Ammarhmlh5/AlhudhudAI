package accounting

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/jmoiron/sqlx"
)

// ErrInvalidJournalEntry returned when journal entry debits != credits
var ErrInvalidJournalEntry = fmt.Errorf("debit and credit must be equal")

// ErrAccountNotFound returned when account doesn't exist
var ErrAccountNotFound = fmt.Errorf("account not found")

// ErrInsufficientAccountBalance returned for asset/expense accounts going negative
var ErrInsufficientAccountBalance = fmt.Errorf("insufficient account balance")

// GetAccount retrieves an account by ID
func (m *Manager) GetAccount(ctx context.Context, id uuid.UUID) (*Account, error) {
	var a Account
	var nameEN, desc sql.NullString
	var parentID sql.NullString

	err := m.db.GetContext(ctx, &a, `
		SELECT id, code, name, name_en, type, parent_id, is_active, allow_invoice, description, created_at, updated_at
		FROM accounts WHERE id = $1`, id)
	if err == sql.ErrNoRows {
		return nil, ErrAccountNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	a.NameEN = nullStringPtr(nameEN)
	a.Description = nullStringPtr(desc)
	if parentID.Valid {
		pid, _ := uuid.Parse(parentID.String)
		a.ParentID = &pid
	}

	return &a, nil
}

// ListAccounts returns all active accounts
func (m *Manager) ListAccounts(ctx context.Context, accountType string) ([]Account, error) {
	query := `SELECT id, code, name, name_en, type, parent_id, is_active, allow_invoice, description, created_at, updated_at
		FROM accounts WHERE is_active = TRUE`
	args := []interface{}{}

	if accountType != "" {
		query += " AND type = $1"
		args = append(args, accountType)
	}
	query += " ORDER BY code"

	rows, err := m.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var a Account
		var nameEN, desc sql.NullString
		var parentID sql.NullString

		err := rows.Scan(&a.ID, &a.Code, &a.Name, &nameEN, &a.Type, &parentID, &a.IsActive, &a.AllowInvoice, &desc, &a.CreatedAt, &a.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan account: %w", err)
		}

		a.NameEN = nullStringPtr(nameEN)
		a.Description = nullStringPtr(desc)
		if parentID.Valid {
			pid, _ := uuid.Parse(parentID.String)
			a.ParentID = &pid
		}

		accounts = append(accounts, a)
	}

	return accounts, nil
}

// CreateAccount creates a new account
func (m *Manager) CreateAccount(ctx context.Context, req *CreateAccountRequest, createdBy *uuid.UUID) (*Account, error) {
	// Check for duplicate code
	var exists bool
	err := m.db.GetContext(ctx, &exists, "SELECT EXISTS(SELECT 1 FROM accounts WHERE code = $1)", req.Code)
	if err != nil {
		return nil, fmt.Errorf("failed to check account code: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("account code %s already exists", req.Code)
	}

	var account Account
	err = m.db.GetContext(ctx, &account, `
		INSERT INTO accounts (code, name, name_en, type, parent_id, description, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, code, name, name_en, type, parent_id, is_active, allow_invoice, description, created_at, updated_at`,
		req.Code, req.Name, req.NameEN, req.Type, req.ParentID, req.Description, createdBy)
	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	return &account, nil
}

// UpdateAccount updates an existing account
func (m *Manager) UpdateAccount(ctx context.Context, id uuid.UUID, req *UpdateAccountRequest) (*Account, error) {
	// Verify account exists
	_, err := m.GetAccount(ctx, id)
	if err != nil {
		return nil, err
	}

	var account Account
	err = m.db.GetContext(ctx, &account, `
		UPDATE accounts SET 
			coalesce(NULLIF($2, ''), code),
			name = COALESCE(NULLIF($3, ''), name),
			name_en = $4,
			type = COALESCE(NULLIF($5, ''), type),
			parent_id = $6,
			is_active = COALESCE($7, is_active),
			description = $8,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, code, name, name_en, type, parent_id, is_active, allow_invoice, description, created_at, updated_at`,
		id, req.Code, req.Name, req.NameEN, req.Type, req.ParentID, req.IsActive, req.Description)
	if err != nil {
		return nil, fmt.Errorf("failed to update account: %w", err)
	}

	return &account, nil
}

// nextJournalEntryNumber generates the next journal entry number
func (m *Manager) nextJournalEntryNumber(ctx context.Context, tx *sqlx.Tx, year int) (string, error) {
	var lastNumber int
	err := tx.GetContext(ctx, &lastNumber, `
		INSERT INTO journal_entry_counters (fiscal_year, last_number)
		VALUES ($1, 0)
		ON CONFLICT (fiscal_year) DO UPDATE SET last_number = journal_entry_counters.last_number
		RETURNING last_number`, year)
	if err != nil {
		return "", fmt.Errorf("failed to get journal entry counter: %w", err)
	}

	newNumber := lastNumber + 1
	_, err = tx.ExecContext(ctx, `
		UPDATE journal_entry_counters SET last_number = $1 WHERE fiscal_year = $2`,
		newNumber, year)
	if err != nil {
		return "", fmt.Errorf("failed to update journal entry counter: %w", err)
	}

	return fmt.Sprintf("JE-%d-%06d", year, newNumber), nil
}

// CreateJournalEntry creates a new journal entry with lines
func (m *Manager) CreateJournalEntry(ctx context.Context, req *JournalEntryRequest, createdBy *uuid.UUID) (*JournalEntry, error) {
	// Validate debits equal credits
	var totalDebit, totalCredit decimal.Decimal
	for _, line := range req.Lines {
		totalDebit = totalDebit.Add(line.Debit)
		totalCredit = totalCredit.Add(line.Credit)
	}

	if !totalDebit.Equal(totalCredit) {
		return nil, ErrInvalidJournalEntry
	}

	if totalDebit.IsZero() {
		return nil, fmt.Errorf("journal entry must have non-zero amounts")
	}

	entryDate, err := time.Parse("2006-01-02", req.EntryDate)
	if err != nil {
		entryDate = time.Now()
	}

	return m.createJournalEntryInTx(ctx, nil, req, entryDate, totalDebit, createdBy)
}

func (m *Manager) createJournalEntryInTx(ctx context.Context, tx *sqlx.Tx, req *JournalEntryRequest, entryDate time.Time, totalAmount decimal.Decimal, createdBy *uuid.UUID) (*JournalEntry, error) {
	runInTx := tx == nil

	if runInTx {
		tx, err := m.db.BeginTxx(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer tx.Rollback()

		result, err := m.createJournalEntryInTx(ctx, tx, req, entryDate, totalAmount, createdBy)
		if err != nil {
			return nil, err
		}

		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("failed to commit journal entry: %w", err)
		}

		return result, nil
	}

	// Generate entry number
	entryNumber, err := m.nextJournalEntryNumber(ctx, tx, entryDate.Year())
	if err != nil {
		return nil, err
	}

	// Insert journal entry
	var entry JournalEntry
	err = tx.GetContext(ctx, &entry, `
		INSERT INTO journal_entries (entry_number, entry_date, description, total_debit, total_credit, status, created_by)
		VALUES ($1, $2, $3, $4, $5, 'POSTED', $6)
		RETURNING id, entry_number, voucher_type, voucher_id, entry_date, description, total_debit, total_credit, status, created_by, created_at, posted_at, posted_by`,
		entryNumber, entryDate, req.Description, totalAmount, totalAmount, createdBy)
	if err != nil {
		return nil, fmt.Errorf("failed to create journal entry: %w", err)
	}

	now := time.Now()
	entry.PostedAt = &now
	if createdBy != nil {
		entry.PostedBy = createdBy
	}

	// Insert lines
	for _, line := range req.Lines {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO journal_entry_lines (journal_entry_id, account_id, debit, credit, notes)
			VALUES ($1, $2, $3, $4, $5)`,
			entry.ID, line.AccountID, line.Debit, line.Credit, line.Notes)
		if err != nil {
			return nil, fmt.Errorf("failed to insert journal entry line: %w", err)
		}

		// Post to ledger
		if err := m.postToLedger(ctx, tx, line.AccountID, entry.ID, entryDate, req.Description, line.Debit, line.Credit); err != nil {
			return nil, err
		}
	}

	// Map request lines to JournalEntryLine for the response
	var jlines []JournalEntryLine
	for _, rl := range req.Lines {
		jlines = append(jlines, JournalEntryLine{
			JournalEntryID: entry.ID,
			AccountID:      rl.AccountID,
			Debit:          rl.Debit,
			Credit:         rl.Credit,
			Notes:          rl.Notes,
		})
	}
	entry.Lines = jlines
	return &entry, nil
}

// postToLedger posts a line to the ledger
func (m *Manager) postToLedger(ctx context.Context, tx *sqlx.Tx, accountID uuid.UUID, journalEntryID uuid.UUID, entryDate time.Time, description string, debit, credit decimal.Decimal) error {
	if debit.IsZero() && credit.IsZero() {
		return nil
	}

	var balance decimal.Decimal
	err := tx.GetContext(ctx, &balance, `
		SELECT COALESCE(SUM(credit) - SUM(debit), 0)
		FROM ledger_entries
		WHERE account_id = $1 AND entry_date <= $2`,
		accountID, entryDate)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to calculate ledger balance: %w", err)
	}

	// For asset/expense accounts, balance increases with debit
	// For liability/equity/income accounts, balance increases with credit
	var accountType string
	err = tx.GetContext(ctx, &accountType, "SELECT type FROM accounts WHERE id = $1", accountID)
	if err != nil {
		return fmt.Errorf("failed to get account type: %w", err)
	}

	if accountType == AccountTypeAsset || accountType == AccountTypeExpense {
		balance = balance.Add(debit).Sub(credit)
	} else {
		balance = balance.Add(credit).Sub(debit)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO ledger_entries (account_id, journal_entry_id, entry_date, description, debit, credit, balance)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		accountID, journalEntryID, entryDate, description, debit, credit, balance)
	if err != nil {
		return fmt.Errorf("failed to post to ledger: %w", err)
	}

	return nil
}

// GetJournalEntries returns journal entries with optional filters
func (m *Manager) GetJournalEntries(ctx context.Context, fromDate, toDate *string, page, pageSize int) ([]JournalEntry, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// Build query
	query := `SELECT id, entry_number, voucher_type, voucher_id, entry_date, description, total_debit, total_credit, status, created_by, created_at, posted_at, posted_by
		FROM journal_entries WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM journal_entries WHERE 1=1`
	args := []interface{}{}
	argNum := 1

	if fromDate != nil && *fromDate != "" {
		query += fmt.Sprintf(" AND entry_date >= $%d", argNum)
		countQuery += fmt.Sprintf(" AND entry_date >= $%d", argNum)
		args = append(args, *fromDate)
		argNum++
	}

	if toDate != nil && *toDate != "" {
		query += fmt.Sprintf(" AND entry_date <= $%d", argNum)
		countQuery += fmt.Sprintf(" AND entry_date <= $%d", argNum)
		args = append(args, *toDate)
		argNum++
	}

	// Get total count
	var total int
	err := m.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count journal entries: %w", err)
	}

	// Get entries
	query += fmt.Sprintf(" ORDER BY entry_date DESC, id DESC LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, pageSize, offset)

	rows, err := m.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get journal entries: %w", err)
	}
	defer rows.Close()

	var entries []JournalEntry
	for rows.Next() {
		var e JournalEntry
		err := rows.Scan(&e.ID, &e.EntryNumber, &e.VoucherType, &e.VoucherID, &e.EntryDate, &e.Description,
			&e.TotalDebit, &e.TotalCredit, &e.Status, &e.CreatedBy, &e.CreatedAt, &e.PostedAt, &e.PostedBy)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan journal entry: %w", err)
		}
		entries = append(entries, e)
	}

	return entries, total, nil
}

// GetJournalEntry returns a single journal entry with lines
func (m *Manager) GetJournalEntry(ctx context.Context, id uuid.UUID) (*JournalEntry, error) {
	var e JournalEntry
	err := m.db.GetContext(ctx, &e, `
		SELECT id, entry_number, voucher_type, voucher_id, entry_date, description, total_debit, total_credit, status, created_by, created_at, posted_at, posted_by
		FROM journal_entries WHERE id = $1`, id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("journal entry not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get journal entry: %w", err)
	}

	// Get lines
	lines, err := m.GetJournalEntryLines(ctx, id)
	if err != nil {
		return nil, err
	}
	e.Lines = lines

	return &e, nil
}

// GetJournalEntryLines returns lines for a journal entry
func (m *Manager) GetJournalEntryLines(ctx context.Context, entryID uuid.UUID) ([]JournalEntryLine, error) {
	rows, err := m.db.QueryxContext(ctx, `
		SELECT id, journal_entry_id, account_id, debit, credit, notes, created_at
		FROM journal_entry_lines WHERE journal_entry_id = $1
		ORDER BY id`, entryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get journal entry lines: %w", err)
	}
	defer rows.Close()

	var lines []JournalEntryLine
	for rows.Next() {
		var l JournalEntryLine
		err := rows.Scan(&l.ID, &l.JournalEntryID, &l.AccountID, &l.Debit, &l.Credit, &l.Notes, &l.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan journal entry line: %w", err)
		}
		lines = append(lines, l)
	}

	return lines, nil
}

// GetLedger returns ledger entries for an account
func (m *Manager) GetLedger(ctx context.Context, filter LedgerFilter) (*LedgerReport, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	// Get account
	account, err := m.GetAccount(ctx, *filter.AccountID)
	if err != nil {
		return nil, err
	}

	offset := (filter.Page - 1) * filter.PageSize

	// Build query
	query := `SELECT id, account_id, journal_entry_id, voucher_type, voucher_id, entry_date, description, debit, credit, balance, created_at
		FROM ledger_entries WHERE account_id = $1`
	args := []interface{}{filter.AccountID}
	argNum := 2

	if filter.FromDate != nil && *filter.FromDate != "" {
		query += fmt.Sprintf(" AND entry_date >= $%d", argNum)
		args = append(args, *filter.FromDate)
		argNum++
	}

	if filter.ToDate != nil && *filter.ToDate != "" {
		query += fmt.Sprintf(" AND entry_date <= $%d", argNum)
		args = append(args, *filter.ToDate)
		argNum++
	}

	query += fmt.Sprintf(" ORDER BY entry_date ASC, id ASC LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, filter.PageSize, offset)

	rows, err := m.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get ledger entries: %w", err)
	}
	defer rows.Close()

	var entries []LedgerEntry
	var totalDebit, totalCredit, balance decimal.Decimal
	for rows.Next() {
		var le LedgerEntry
		err := rows.Scan(&le.ID, &le.AccountID, &le.JournalEntryID, &le.VoucherType, &le.VoucherID,
			&le.EntryDate, &le.Description, &le.Debit, &le.Credit, &le.Balance, &le.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ledger entry: %w", err)
		}
		entries = append(entries, le)
		totalDebit = totalDebit.Add(le.Debit)
		totalCredit = totalCredit.Add(le.Credit)
	}

	// Get running balance
	if len(entries) > 0 {
		balance = entries[len(entries)-1].Balance
	}

	return &LedgerReport{
		Account:     *account,
		Entries:     entries,
		TotalDebit:  totalDebit,
		TotalCredit: totalCredit,
		Balance:     balance,
	}, nil
}

// GetTrialBalance returns trial balance
func (m *Manager) GetTrialBalance(ctx context.Context, filter TrialBalanceFilter) ([]TrialBalance, error) {
	query := `
		SELECT 
			a.id as account_id,
			a.code,
			a.name,
			a.type,
			COALESCE(SUM(le.credit), 0) - COALESCE(SUM(le.debit), 0) as balance,
			COALESCE(SUM(le.debit), 0) as total_debit,
			COALESCE(SUM(le.credit), 0) as total_credit
		FROM accounts a
		LEFT JOIN ledger_entries le ON a.id = le.account_id
		WHERE a.is_active = TRUE`

	args := []interface{}{}
	argNum := 1

	if filter.FromDate != nil && *filter.FromDate != "" {
		query += fmt.Sprintf(" AND le.entry_date >= $%d", argNum)
		args = append(args, *filter.FromDate)
		argNum++
	}

	if filter.ToDate != nil && *filter.ToDate != "" {
		query += fmt.Sprintf(" AND le.entry_date <= $%d", argNum)
		args = append(args, *filter.ToDate)
		argNum++
	}

	if !filter.IncludeZero {
		query += " HAVING COALESCE(SUM(le.credit), 0) != 0 OR COALESCE(SUM(le.debit), 0) != 0"
	}

	query += " GROUP BY a.id, a.code, a.name, a.type ORDER BY a.type, a.code"

	rows, err := m.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get trial balance: %w", err)
	}
	defer rows.Close()

	var tb []TrialBalance
	for rows.Next() {
		var t TrialBalance
		err := rows.Scan(&t.AccountID, &t.Code, &t.Name, &t.Type, &t.Balance, &t.TotalDebit, &t.TotalCredit)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trial balance: %w", err)
		}
		tb = append(tb, t)
	}

	return tb, nil
}

// Helper function to convert NullString to *string
func nullStringPtr(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

// GenerateJournalEntryFromVoucher creates a journal entry from a posted voucher
func (m *Manager) GenerateJournalEntryFromVoucher(ctx context.Context, voucherType string, voucherID uuid.UUID, voucherNumber string, amount decimal.Decimal, cashboxID uuid.UUID, tenantID *uuid.UUID, description string) error {
	// Get cashbox account (asset)
	var cashboxAccountID uuid.UUID
	err := m.db.GetContext(ctx, &cashboxAccountID, `
		SELECT id FROM accounts WHERE code = '1000' AND is_active = TRUE LIMIT 1`)
	if err != nil {
		return fmt.Errorf("cashbox account not found: %w", err)
	}

	// Get AR or Income account based on tenant
	var counterAccountID uuid.UUID
	if tenantID != nil {
		// Customer receivable
		err = m.db.GetContext(ctx, &counterAccountID, `
			SELECT id FROM accounts WHERE code = '1100' AND is_active = TRUE LIMIT 1`)
	} else {
		// Revenue
		err = m.db.GetContext(ctx, &counterAccountID, `
			SELECT id FROM accounts WHERE code = '4000' AND is_active = TRUE LIMIT 1`)
	}
	if err != nil {
		return fmt.Errorf("counter account not found: %w", err)
	}

	req := &JournalEntryRequest{
		EntryDate: time.Now().Format("2006-01-02"),
		Description: fmt.Sprintf("%s - %s", voucherType, voucherNumber),
		Lines: []JournalLineRequest{
			{
				AccountID: cashboxAccountID,
				Debit:     amount,
				Credit:    decimal.Zero,
			},
			{
				AccountID: counterAccountID,
				Debit:     decimal.Zero,
				Credit:    amount,
			},
		},
	}

	entry, err := m.CreateJournalEntry(ctx, req, nil)
	if err != nil {
		return fmt.Errorf("failed to create journal entry: %w", err)
	}

	// Update voucher with journal entry reference
	if voucherType == "RECEIPT" {
		_, err = m.db.ExecContext(ctx, `
			UPDATE receipt_vouchers SET journal_entry_id = $1 WHERE id = $2`,
			entry.ID, voucherID)
	} else {
		_, err = m.db.ExecContext(ctx, `
			UPDATE payment_vouchers SET journal_entry_id = $1 WHERE id = $2`,
			entry.ID, voucherID)
	}

	return err
}