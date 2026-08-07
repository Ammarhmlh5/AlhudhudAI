package services

import (
	"context"
	"finance/internal/models"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ─── Operation 1: Charge Wallet (شحن رصيد شركة) ───────────────────────────────
// Creates a receipt voucher, posts a double-entry (Cashbox DR / Wallet CR), and generates invoice.
func (s *AccountingService) ChargeWallet(ctx context.Context, cashboxCode string, walletAccountID uuid.UUID, amount decimal.Decimal, payerName string, description string, createdBy *uuid.UUID) (*models.JournalEntry, error) {
	var cashbox models.Account
	if err := s.db.WithContext(ctx).Where("code = ?", cashboxCode).First(&cashbox).Error; err != nil {
		return nil, fmt.Errorf("الصندوق %s غير موجود", cashboxCode)
	}
	lines := []models.JournalEntryLine{
		{AccountID: cashbox.ID, Debit: amount, Credit: decimal.Zero, Description: "إيداع في الصندوق - " + payerName},
		{AccountID: walletAccountID, Debit: decimal.Zero, Credit: amount, Description: "شحن محفظة: " + description},
	}
	return s.CreateJournalEntry(ctx, "شحن رصيد: "+payerName, lines, createdBy)
}

// ─── Operation 2: Debit Cashbox (سند صرف) ────────────────────────────────────
// Creates a payment voucher: Expense/Payee DR / Cashbox CR
func (s *AccountingService) DebitCashbox(ctx context.Context, cashboxCode string, expenseAccountCode string, amount decimal.Decimal, payeeName string, description string, createdBy *uuid.UUID) (*models.JournalEntry, error) {
	var cashbox, expenseAcc models.Account
	if err := s.db.WithContext(ctx).Where("code = ?", cashboxCode).First(&cashbox).Error; err != nil {
		return nil, fmt.Errorf("الصندوق %s غير موجود", cashboxCode)
	}
	if err := s.db.WithContext(ctx).Where("code = ?", expenseAccountCode).First(&expenseAcc).Error; err != nil {
		return nil, fmt.Errorf("حساب المصروف %s غير موجود", expenseAccountCode)
	}
	// Check cashbox has enough balance
	balance, err := s.GetAccountBalance(ctx, cashbox.ID)
	if err != nil || balance.LessThan(amount) {
		return nil, fmt.Errorf("رصيد الصندوق غير كافٍ للصرف (متاح: %s)", balance.StringFixed(2))
	}
	lines := []models.JournalEntryLine{
		{AccountID: expenseAcc.ID, Debit: amount, Credit: decimal.Zero, Description: "مصروف: " + payeeName},
		{AccountID: cashbox.ID, Debit: decimal.Zero, Credit: amount, Description: "صرف من الصندوق: " + description},
	}
	return s.CreateJournalEntry(ctx, "سند صرف: "+payeeName, lines, createdBy)
}

// ─── Operation 3: Create Cashbox (إضافة صندوق) ───────────────────────────────
func (m *AccountManager) CreateCashbox(ctx context.Context, code, name string, createdBy *uuid.UUID) (*models.Account, error) {
	var parentAsset models.Account
	if err := m.db.WithContext(ctx).Where("code = ?", "1100").First(&parentAsset).Error; err != nil {
		return nil, fmt.Errorf("الحساب الأب للأصول (1100) غير موجود")
	}
	cashbox := &models.Account{
		Code:          code,
		Name:          name,
		Type:          models.Asset,
		NormalBalance: models.Debit,
		ParentID:      &parentAsset.ID,
		IsActive:      true,
	}
	if err := m.db.WithContext(ctx).Create(cashbox).Error; err != nil {
		return nil, fmt.Errorf("فشل إنشاء الصندوق: %v", err)
	}
	return cashbox, nil
}

// ─── Operation 5: Cashbox Statement (كشف حساب صندوق) ─────────────────────────
type MovementLine struct {
	EntryDate   time.Time       `json:"entry_date"`
	EntryNumber string          `json:"entry_number"`
	Description string          `json:"description"`
	Debit       decimal.Decimal `json:"debit"`
	Credit      decimal.Decimal `json:"credit"`
	Balance     decimal.Decimal `json:"balance"`
}

func (s *AccountingService) GetCashboxStatement(ctx context.Context, cashboxAccountID uuid.UUID, from, to time.Time) ([]MovementLine, error) {
	type rawLine struct {
		EntryDate   time.Time
		EntryNumber string
		Description string
		Debit       decimal.Decimal
		Credit      decimal.Decimal
	}
	var rows []rawLine
	err := s.db.WithContext(ctx).Table("journal_entry_lines jel").
		Select("je.entry_date, je.entry_number, jel.description, jel.debit, jel.credit").
		Joins("JOIN journal_entries je ON je.id = jel.journal_entry_id").
		Where("jel.account_id = ? AND je.entry_date BETWEEN ? AND ? AND je.status = ?", cashboxAccountID, from, to, models.Posted).
		Order("je.entry_date ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	var account models.Account
	s.db.WithContext(ctx).First(&account, "id = ?", cashboxAccountID)

	runningBalance := decimal.Zero
	var lines []MovementLine
	for _, r := range rows {
		if account.NormalBalance == models.Debit {
			runningBalance = runningBalance.Add(r.Debit).Sub(r.Credit)
		} else {
			runningBalance = runningBalance.Add(r.Credit).Sub(r.Debit)
		}
		lines = append(lines, MovementLine{
			EntryDate:   r.EntryDate,
			EntryNumber: r.EntryNumber,
			Description: r.Description,
			Debit:       r.Debit,
			Credit:      r.Credit,
			Balance:     runningBalance,
		})
	}
	return lines, nil
}

// ─── Operation 6: Reverse Entry (عكس سند) ────────────────────────────────────
func (s *AccountingService) ReverseJournalEntry(ctx context.Context, entryID uuid.UUID, reversedBy *uuid.UUID) (*models.JournalEntry, error) {
	var original models.JournalEntry
	if err := s.db.WithContext(ctx).Preload("Lines").First(&original, "id = ?", entryID).Error; err != nil {
		return nil, fmt.Errorf("القيد غير موجود")
	}
	if original.Status == models.Reversed {
		return nil, fmt.Errorf("القيد معكوس مسبقاً")
	}
	if original.Status != models.Posted {
		return nil, fmt.Errorf("لا يمكن عكس قيد غير مرحّل")
	}
	// Build reversal lines (swap debit/credit)
	var reversalLines []models.JournalEntryLine
	for _, l := range original.Lines {
		reversalLines = append(reversalLines, models.JournalEntryLine{
			AccountID:   l.AccountID,
			Debit:       l.Credit,
			Credit:      l.Debit,
			Description: "عكس: " + l.Description,
		})
	}
	reversal, err := s.CreateJournalEntry(ctx, "عكس قيد: "+original.Description, reversalLines, reversedBy)
	if err != nil {
		return nil, err
	}
	// Mark original as reversed
	s.db.WithContext(ctx).Model(&original).Updates(map[string]interface{}{
		"status":      models.Reversed,
		"reversal_of": reversal.ID,
	})
	reversal.ReversalOf = &original.ID
	s.db.WithContext(ctx).Save(reversal)
	return reversal, nil
}
