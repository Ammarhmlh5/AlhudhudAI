package accounting

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Cashbox types
const (
	CashboxTypeCash   = "CASH"
	CashboxTypeBank   = "BANK"
	CashboxTypeWallet = "WALLET"
	CashboxTypeOther  = "OTHER"
)

// Cashbox statuses
const (
	CashboxStatusActive   = "ACTIVE"
	CashboxStatusInactive = "INACTIVE"
)

// Voucher types
const (
	VoucherTypeReceipt = "RECEIPT"
	VoucherTypePayment = "PAYMENT"
)

// Voucher statuses
const (
	VoucherStatusDraft    = "DRAFT"
	VoucherStatusPosted   = "POSTED"
	VoucherStatusReversed = "REVERSED"
)

// Cashbox represents a fund location (cash, bank account, wallet).
type Cashbox struct {
	ID             uuid.UUID       `json:"id"`
	Code           string          `json:"code"`
	Name           string          `json:"name"`
	NameEN         *string         `json:"name_en,omitempty"`
	Type           string          `json:"type"`
	Currency       string          `json:"currency"`
	BankName       *string         `json:"bank_name,omitempty"`
	AccountHolder  *string         `json:"account_holder,omitempty"`
	AccountNumber  *string         `json:"account_number,omitempty"`
	IBAN           *string         `json:"iban,omitempty"`
	OpeningBalance decimal.Decimal `json:"opening_balance"`
	CurrentBalance decimal.Decimal `json:"current_balance"`
	MinBalance     decimal.Decimal `json:"min_balance"`
	Status         string          `json:"status"`
	Description    *string         `json:"description,omitempty"`
	CreatedBy      *uuid.UUID      `json:"created_by,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// ReceiptVoucher documents money entering a cashbox.
type ReceiptVoucher struct {
	ID              uuid.UUID       `json:"id"`
	VoucherNumber   string          `json:"voucher_number"`
	VoucherDate     time.Time       `json:"voucher_date"`
	CashboxID       uuid.UUID       `json:"cashbox_id"`
	Amount          decimal.Decimal `json:"amount"`
	Currency        string          `json:"currency"`
	PayerName       string          `json:"payer_name"`
	PayerType       string          `json:"payer_type"`
	TenantID        *uuid.UUID      `json:"tenant_id,omitempty"`
	ResellerID      *uuid.UUID      `json:"reseller_id,omitempty"`
	PaymentMethod   string          `json:"payment_method"`
	ReferenceNumber *string         `json:"reference_number,omitempty"`
	Description     *string         `json:"description,omitempty"`
	Status          string          `json:"status"`
	ReversalOf      *uuid.UUID      `json:"reversal_of,omitempty"`
	CreditedWallet  bool            `json:"credited_wallet"`
	CreatedBy       *uuid.UUID      `json:"created_by,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	PostedAt        *time.Time      `json:"posted_at,omitempty"`
	PostedBy        *uuid.UUID      `json:"posted_by,omitempty"`
	ReversedAt      *time.Time      `json:"reversed_at,omitempty"`
	ReversedBy      *uuid.UUID      `json:"reversed_by,omitempty"`
}

// PaymentVoucher documents money leaving a cashbox.
type PaymentVoucher struct {
	ID              uuid.UUID       `json:"id"`
	VoucherNumber   string          `json:"voucher_number"`
	VoucherDate     time.Time       `json:"voucher_date"`
	CashboxID       uuid.UUID       `json:"cashbox_id"`
	Amount          decimal.Decimal `json:"amount"`
	Currency        string          `json:"currency"`
	PayeeName       string          `json:"payee_name"`
	PayeeType       string          `json:"payee_type"`
	TenantID        *uuid.UUID      `json:"tenant_id,omitempty"`
	ResellerID      *uuid.UUID      `json:"reseller_id,omitempty"`
	PaymentMethod   string          `json:"payment_method"`
	ReferenceNumber *string         `json:"reference_number,omitempty"`
	Description     *string         `json:"description,omitempty"`
	Status          string          `json:"status"`
	ReversalOf      *uuid.UUID      `json:"reversal_of,omitempty"`
	DebitedWallet   bool            `json:"debited_wallet"`
	CreatedBy       *uuid.UUID      `json:"created_by,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	PostedAt        *time.Time      `json:"posted_at,omitempty"`
	PostedBy        *uuid.UUID      `json:"posted_by,omitempty"`
	ReversedAt      *time.Time      `json:"reversed_at,omitempty"`
	ReversedBy      *uuid.UUID      `json:"reversed_by,omitempty"`
}

// CashboxMovement is a single ledger line on a cashbox.
type CashboxMovement struct {
	ID            uuid.UUID       `json:"id"`
	CashboxID     uuid.UUID       `json:"cashbox_id"`
	VoucherType   string          `json:"voucher_type"`
	VoucherID     uuid.UUID       `json:"voucher_id"`
	VoucherNumber string          `json:"voucher_number"`
	MovementDate  time.Time       `json:"movement_date"`
	Amount        decimal.Decimal `json:"amount"`
	BalanceAfter  decimal.Decimal `json:"balance_after"`
	Description   *string         `json:"description,omitempty"`
	CreatedBy     *uuid.UUID      `json:"created_by,omitempty"`
}

// StatementTotals summarizes a cashbox statement.
type StatementTotals struct {
	TotalReceipts decimal.Decimal `json:"total_receipts"`
	TotalPayments decimal.Decimal `json:"total_payments"`
	Net           decimal.Decimal `json:"net"`
}

// CashboxStatement is the cashbox detail plus its movements.
type CashboxStatement struct {
	Cashbox   Cashbox           `json:"cashbox"`
	Movements []CashboxMovement `json:"movements"`
	Totals    StatementTotals   `json:"totals"`
}

// VoucherView is a unified view of a receipt or payment voucher for lists.
type VoucherView struct {
	ID              uuid.UUID       `json:"id"`
	VoucherNumber   string          `json:"voucher_number"`
	VoucherType     string          `json:"voucher_type"`
	VoucherDate     time.Time       `json:"voucher_date"`
	CashboxID       uuid.UUID       `json:"cashbox_id"`
	Amount          decimal.Decimal `json:"amount"`
	Currency        string          `json:"currency"`
	PartyName       string          `json:"party_name"`
	PartyType       string          `json:"party_type"`
	TenantID        *uuid.UUID      `json:"tenant_id,omitempty"`
	ResellerID      *uuid.UUID      `json:"reseller_id,omitempty"`
	PaymentMethod   string          `json:"payment_method"`
	ReferenceNumber *string         `json:"reference_number,omitempty"`
	Description     *string         `json:"description,omitempty"`
	Status          string          `json:"status"`
	ReversalOf      *uuid.UUID      `json:"reversal_of,omitempty"`
	CreatedBy       *uuid.UUID      `json:"created_by,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	PostedAt        *time.Time      `json:"posted_at,omitempty"`
	PostedBy        *uuid.UUID      `json:"posted_by,omitempty"`
}

// uuidOrNil converts a sql.NullString into *uuid.UUID (nil when invalid).
func uuidOrNil(ns sql.NullString) *uuid.UUID {
	if !ns.Valid || ns.String == "" {
		return nil
	}
	id, err := uuid.Parse(ns.String)
	if err != nil {
		return nil
	}
	return &id
}

// strOrNil converts a sql.NullString into *string.
func strOrNil(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	s := ns.String
	return &s
}

// Account types (Chart of Accounts)
const (
	AccountTypeAsset     = "ASSET"
	AccountTypeLiability = "LIABILITY"
	AccountTypeEquity    = "EQUITY"
	AccountTypeIncome    = "INCOME"
	AccountTypeExpense   = "EXPENSE"
)

// Journal entry statuses
const (
	JournalEntryStatusDraft     = "DRAFT"
	JournalEntryStatusPosted    = "POSTED"
	JournalEntryStatusCancelled = "CANCELLED"
)

// Account represents a chart of accounts entry.
type Account struct {
	ID           uuid.UUID  `json:"id"`
	Code         string     `json:"code"`
	Name         string     `json:"name"`
	NameEN       *string    `json:"name_en,omitempty"`
	Type         string     `json:"type"`
	ParentID     *uuid.UUID `json:"parent_id,omitempty"`
	IsActive     bool       `json:"is_active"`
	AllowInvoice bool       `json:"allow_invoice"`
	Description  *string    `json:"description,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// JournalEntry represents a double-entry journal entry.
type JournalEntry struct {
	ID          uuid.UUID          `json:"id"`
	EntryNumber string             `json:"entry_number"`
	VoucherType *string            `json:"voucher_type,omitempty"`
	VoucherID   *uuid.UUID         `json:"voucher_id,omitempty"`
	EntryDate   time.Time          `json:"entry_date"`
	Description *string            `json:"description,omitempty"`
	TotalDebit  decimal.Decimal    `json:"total_debit"`
	TotalCredit decimal.Decimal    `json:"total_credit"`
	Status      string             `json:"status"`
	CreatedBy   *uuid.UUID         `json:"created_by,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
	PostedAt    *time.Time         `json:"posted_at,omitempty"`
	PostedBy    *uuid.UUID         `json:"posted_by,omitempty"`
	Lines       []JournalEntryLine `json:"lines,omitempty"`
}

// JournalEntryLine represents a single line in a journal entry.
type JournalEntryLine struct {
	ID             uuid.UUID       `json:"id"`
	JournalEntryID uuid.UUID       `json:"journal_entry_id"`
	AccountID      uuid.UUID       `json:"account_id"`
	Debit          decimal.Decimal `json:"debit"`
	Credit         decimal.Decimal `json:"credit"`
	Notes          *string         `json:"notes,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

// LedgerEntry represents a ledger posting.
type LedgerEntry struct {
	ID             uuid.UUID       `json:"id"`
	AccountID      uuid.UUID       `json:"account_id"`
	JournalEntryID *uuid.UUID      `json:"journal_entry_id,omitempty"`
	VoucherType    *string         `json:"voucher_type,omitempty"`
	VoucherID      *uuid.UUID      `json:"voucher_id,omitempty"`
	EntryDate      time.Time       `json:"entry_date"`
	Description    *string         `json:"description,omitempty"`
	Debit          decimal.Decimal `json:"debit"`
	Credit         decimal.Decimal `json:"credit"`
	Balance        decimal.Decimal `json:"balance"`
	CreatedAt      time.Time       `json:"created_at"`
}

// TrialBalance represents a trial balance report row.
type TrialBalance struct {
	AccountID   uuid.UUID       `json:"account_id"`
	Code        string          `json:"code"`
	Name        string          `json:"name"`
	Type        string          `json:"type"`
	Balance     decimal.Decimal `json:"balance"`
	TotalDebit  decimal.Decimal `json:"total_debit"`
	TotalCredit decimal.Decimal `json:"total_credit"`
}

// LedgerReport represents ledger entries for an account.
type LedgerReport struct {
	Account     Account         `json:"account"`
	Entries     []LedgerEntry   `json:"entries"`
	TotalDebit  decimal.Decimal `json:"total_debit"`
	TotalCredit decimal.Decimal `json:"total_credit"`
	Balance     decimal.Decimal `json:"balance"`
}

// CreateAccountRequest for creating a new account
type CreateAccountRequest struct {
	Code        string     `json:"code" binding:"required"`
	Name        string     `json:"name" binding:"required"`
	NameEN      *string    `json:"name_en"`
	Type        string     `json:"type" binding:"required,oneof=ASSET LIABILITY EQUITY INCOME EXPENSE"`
	ParentID    *uuid.UUID `json:"parent_id"`
	Description *string    `json:"description"`
}

// UpdateAccountRequest for updating an account
type UpdateAccountRequest struct {
	Code        string     `json:"code"`
	Name        string     `json:"name"`
	NameEN      *string    `json:"name_en"`
	Type        string     `json:"type"`
	ParentID    *uuid.UUID `json:"parent_id"`
	IsActive    *bool      `json:"is_active"`
	Description *string    `json:"description"`
}

// JournalEntryRequest for creating a journal entry
type JournalEntryRequest struct {
	EntryDate   string               `json:"entry_date" binding:"required"`
	Description string               `json:"description"`
	Lines       []JournalLineRequest `json:"lines" binding:"required,min=2,dive"`
}

// JournalLineRequest for journal entry line
type JournalLineRequest struct {
	AccountID uuid.UUID       `json:"account_id" binding:"required"`
	Debit     decimal.Decimal `json:"debit"`
	Credit    decimal.Decimal `json:"credit"`
	Notes     *string         `json:"notes"`
}

// LedgerFilter for ledger queries
type LedgerFilter struct {
	AccountID *uuid.UUID `json:"account_id"`
	FromDate  *string    `json:"from_date"`
	ToDate    *string    `json:"to_date"`
	Page      int        `json:"page"`
	PageSize  int        `json:"page_size"`
}

// TrialBalanceFilter for trial balance queries
type TrialBalanceFilter struct {
	FromDate    *string `json:"from_date"`
	ToDate      *string `json:"to_date"`
	IncludeZero bool    `json:"include_zero"`
}
