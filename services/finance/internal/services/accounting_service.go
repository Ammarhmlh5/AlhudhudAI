package services

import (
	"context"
	"errors"
	"finance/internal/models"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type AccountingService struct {
	db *gorm.DB
}

func NewAccountingService(db *gorm.DB) *AccountingService {
	return &AccountingService{db: db}
}

// CreateJournalEntry creates a balanced double-entry record.
// It ensures that the sum of debits equals the sum of credits.
func (s *AccountingService) CreateJournalEntry(ctx context.Context, description string, lines []models.JournalEntryLine, createdBy *uuid.UUID) (*models.JournalEntry, error) {
	if len(lines) < 2 {
		return nil, errors.New("القيد يجب أن يحتوي على سطرين على الأقل (مدين ودائن)")
	}

	totalDebit := decimal.NewFromInt(0)
	totalCredit := decimal.NewFromInt(0)

	for _, line := range lines {
		if line.Debit.IsNegative() || line.Credit.IsNegative() {
			return nil, errors.New("لا يمكن أن تحتوي الأرصدة على قيم سالبة في أسطر القيد")
		}
		if line.Debit.IsPositive() && line.Credit.IsPositive() {
			return nil, errors.New("السطر الواحد لا يمكن أن يكون مديناً ودائناً في نفس الوقت")
		}
		totalDebit = totalDebit.Add(line.Debit)
		totalCredit = totalCredit.Add(line.Credit)
	}

	if !totalDebit.Equal(totalCredit) {
		return nil, fmt.Errorf("القيد غير متزن: إجمالي المدين (%s) لا يساوي إجمالي الدائن (%s)", totalDebit.String(), totalCredit.String())
	}

	if totalDebit.IsZero() {
		return nil, errors.New("القيد صفري ولا يحتوي على قيمة مالية")
	}

	// Create the transaction
	entry := &models.JournalEntry{
		// In production, entry number should be generated safely using a sequence or locking
		EntryNumber: fmt.Sprintf("JE-%d", time.Now().UnixNano()),
		EntryDate:   time.Now(),
		Description: description,
		Status:      models.Posted,
		CreatedBy:   createdBy,
		Lines:       lines,
	}

	// Save everything atomically
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(entry).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("فشل حفظ القيد في قاعدة البيانات: %v", err)
	}

	return entry, nil
}

// GetAccountBalance calculates the real-time balance of an account from the ledger
func (s *AccountingService) GetAccountBalance(ctx context.Context, accountID uuid.UUID) (decimal.Decimal, error) {
	var account models.Account
	if err := s.db.WithContext(ctx).First(&account, "id = ?", accountID).Error; err != nil {
		return decimal.Zero, fmt.Errorf("الحساب غير موجود: %v", err)
	}

	type Result struct {
		TotalDebit  decimal.Decimal
		TotalCredit decimal.Decimal
	}
	var res Result

	// Sum all debits and credits for this account from POSTED entries only
	err := s.db.WithContext(ctx).Table("journal_entry_lines").
		Select("COALESCE(SUM(debit), 0) as total_debit, COALESCE(SUM(credit), 0) as total_credit").
		Joins("JOIN journal_entries on journal_entries.id = journal_entry_lines.journal_entry_id").
		Where("journal_entry_lines.account_id = ? AND journal_entries.status = ?", accountID, models.Posted).
		Scan(&res).Error

	if err != nil {
		return decimal.Zero, fmt.Errorf("فشل حساب الرصيد من دفتر الأستاذ: %v", err)
	}

	balance := decimal.Zero

	// Normal Balance Logic:
	// ASSET, EXPENSE increase with Debit, decrease with Credit -> Balance = Debit - Credit
	// LIABILITY, REVENUE, EQUITY increase with Credit, decrease with Debit -> Balance = Credit - Debit
	if account.NormalBalance == models.Debit {
		balance = res.TotalDebit.Sub(res.TotalCredit)
	} else if account.NormalBalance == models.Credit {
		balance = res.TotalCredit.Sub(res.TotalDebit)
	}

	return balance, nil
}
