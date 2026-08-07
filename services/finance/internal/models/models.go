package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// AccountType represents the main 5 accounting types
type AccountType string

const (
	Asset     AccountType = "ASSET"
	Liability AccountType = "LIABILITY"
	Equity    AccountType = "EQUITY"
	Revenue   AccountType = "REVENUE"
	Expense   AccountType = "EXPENSE"
)

// NormalBalance represents the normal state of the account balance
type NormalBalance string

const (
	Debit  NormalBalance = "DEBIT"
	Credit NormalBalance = "CREDIT"
)

// Account represents a node in the Chart of Accounts
type Account struct {
	ID            uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Code          string         `gorm:"type:varchar(20);uniqueIndex;not null"`
	Name          string         `gorm:"type:varchar(150);not null"`
	Type          AccountType    `gorm:"type:varchar(20);not null"`
	NormalBalance NormalBalance  `gorm:"type:varchar(10);not null"`
	ParentID      *uuid.UUID     `gorm:"type:uuid"`
	ReferenceType *string        `gorm:"type:varchar(50)"` // e.g., 'TENANT', 'RESELLER'
	ReferenceID   *uuid.UUID     `gorm:"type:uuid"`        // The external ID in the main system
	IsActive      bool           `gorm:"default:true;not null"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

// EntryStatus represents the state of a journal entry
type EntryStatus string

const (
	Draft    EntryStatus = "DRAFT"
	Posted   EntryStatus = "POSTED"
	Reversed EntryStatus = "REVERSED"
)

// JournalEntry represents a double-entry transaction header
type JournalEntry struct {
	ID          uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	EntryNumber string         `gorm:"type:varchar(40);uniqueIndex;not null"`
	EntryDate   time.Time      `gorm:"not null"`
	Description string         `gorm:"type:text"`
	Status      EntryStatus    `gorm:"type:varchar(20);default:'POSTED';not null"`
	ReversalOf  *uuid.UUID     `gorm:"type:uuid"` // ID of the entry being reversed (if any)
	CreatedBy   *uuid.UUID     `gorm:"type:uuid"` // Admin or system user ID who triggered it
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`

	Lines []JournalEntryLine `gorm:"foreignKey:JournalEntryID"`
}

// JournalEntryLine represents a single debit or credit line within a JournalEntry
type JournalEntryLine struct {
	ID             uuid.UUID       `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	JournalEntryID uuid.UUID       `gorm:"type:uuid;not null;index"`
	AccountID      uuid.UUID       `gorm:"type:uuid;not null;index"`
	Debit          decimal.Decimal `gorm:"type:decimal(19,4);default:0;not null"`
	Credit         decimal.Decimal `gorm:"type:decimal(19,4);default:0;not null"`
	Description    string          `gorm:"type:text"`
	CreatedAt      time.Time
}

// PeriodStatus represents if a financial period is still modifiable
type PeriodStatus string

const (
	Open   PeriodStatus = "OPEN"
	Closed PeriodStatus = "CLOSED"
)

// AccountingPeriod represents a fiscal month or year to prevent back-dating entries
type AccountingPeriod struct {
	ID         uuid.UUID    `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	PeriodName string       `gorm:"type:varchar(50);not null"`
	StartDate  time.Time    `gorm:"type:date;not null"`
	EndDate    time.Time    `gorm:"type:date;not null"`
	Status     PeriodStatus `gorm:"type:varchar(20);default:'OPEN';not null"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Invoice represents a tax invoice generated for a specific transaction
type Invoice struct {
	ID             uuid.UUID       `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	InvoiceNumber  string          `gorm:"type:varchar(50);uniqueIndex;not null"`
	JournalEntryID uuid.UUID       `gorm:"type:uuid;not null;index"`
	CustomerName   string          `gorm:"type:varchar(150);not null"`
	IssueDate      time.Time       `gorm:"not null"`
	SubTotal       decimal.Decimal `gorm:"type:decimal(19,4);not null"`
	VATAmount      decimal.Decimal `gorm:"type:decimal(19,4);not null"`
	TotalAmount    decimal.Decimal `gorm:"type:decimal(19,4);not null"`
	CreatedAt      time.Time
}
