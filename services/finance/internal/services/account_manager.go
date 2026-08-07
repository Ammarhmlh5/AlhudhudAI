package services

import (
	"context"
	"finance/internal/models"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AccountManager struct {
	db *gorm.DB
}

func NewAccountManager(db *gorm.DB) *AccountManager {
	return &AccountManager{db: db}
}

// defaultAccount defines the structure of a system-required account
type defaultAccount struct {
	Code          string
	Name          string
	Type          models.AccountType
	NormalBalance models.NormalBalance
	ParentCode    *string
}

// SeedDefaultAccounts creates the baseline Chart of Accounts required by the system
func (m *AccountManager) SeedDefaultAccounts(ctx context.Context) error {
	rootAssets := "1000"
	rootLiabilities := "2000"
	rootRevenue := "4000"
	rootExpense := "5000"

	defaults := []defaultAccount{
		// Roots
		{Code: rootAssets, Name: "الأصول", Type: models.Asset, NormalBalance: models.Debit, ParentCode: nil},
		{Code: rootLiabilities, Name: "الخصوم (الالتزامات)", Type: models.Liability, NormalBalance: models.Credit, ParentCode: nil},
		{Code: rootRevenue, Name: "الإيرادات", Type: models.Revenue, NormalBalance: models.Credit, ParentCode: nil},
		{Code: rootExpense, Name: "المصروفات", Type: models.Expense, NormalBalance: models.Debit, ParentCode: nil},

		// Asset Sub-accounts
		{Code: "1100", Name: "الصندوق / الحسابات البنكية للمنصة", Type: models.Asset, NormalBalance: models.Debit, ParentCode: &rootAssets},
		{Code: "1200", Name: "الذمم المدينة", Type: models.Asset, NormalBalance: models.Debit, ParentCode: &rootAssets},

		// Liability Sub-accounts
		{Code: "2100", Name: "محافظ المستخدمين (دائنون)", Type: models.Liability, NormalBalance: models.Credit, ParentCode: &rootLiabilities},
		{Code: "2200", Name: "محافظ الوكلاء (دائنون)", Type: models.Liability, NormalBalance: models.Credit, ParentCode: &rootLiabilities},
		{Code: "2300", Name: "ضريبة القيمة المضافة المستحقة (VAT)", Type: models.Liability, NormalBalance: models.Credit, ParentCode: &rootLiabilities},

		// Revenue Sub-accounts
		{Code: "4100", Name: "إيرادات استهلاك الرسائل", Type: models.Revenue, NormalBalance: models.Credit, ParentCode: &rootRevenue},

		// Expense Sub-accounts
		{Code: "5100", Name: "رسوم بوابات الدفع (عمولات)", Type: models.Expense, NormalBalance: models.Debit, ParentCode: &rootExpense},
	}

	for _, acc := range defaults {
		var existing models.Account
		err := m.db.WithContext(ctx).Where("code = ?", acc.Code).First(&existing).Error
		if errorsIsRecordNotFound(err) {
			newAcc := models.Account{
				Code:          acc.Code,
				Name:          acc.Name,
				Type:          acc.Type,
				NormalBalance: acc.NormalBalance,
				IsActive:      true,
			}

			if acc.ParentCode != nil {
				var parent models.Account
				if err := m.db.WithContext(ctx).Where("code = ?", *acc.ParentCode).First(&parent).Error; err == nil {
					newAcc.ParentID = &parent.ID
				}
			}

			if err := m.db.WithContext(ctx).Create(&newAcc).Error; err != nil {
				return fmt.Errorf("فشل في إنشاء الحساب الأساسي %s: %v", acc.Code, err)
			}
		} else if err != nil {
			return fmt.Errorf("حدث خطأ أثناء البحث عن الحساب %s: %v", acc.Code, err)
		}
	}

	return nil
}

// CreateUserWalletAccount automatically generates a sub-account for a new user/tenant under Liabilities
func (m *AccountManager) CreateUserWalletAccount(ctx context.Context, tenantID uuid.UUID, userName string) (*models.Account, error) {
	return m.createDynamicWallet(ctx, "2100", "TENANT", tenantID, userName)
}

// CreateResellerWalletAccount automatically generates a sub-account for a new reseller under Liabilities
func (m *AccountManager) CreateResellerWalletAccount(ctx context.Context, resellerID uuid.UUID, resellerName string) (*models.Account, error) {
	return m.createDynamicWallet(ctx, "2200", "RESELLER", resellerID, resellerName)
}

// createDynamicWallet is an internal helper to generate wallets for users or resellers
func (m *AccountManager) createDynamicWallet(ctx context.Context, parentCode string, refType string, refID uuid.UUID, name string) (*models.Account, error) {
	// Find the parent account
	var parent models.Account
	if err := m.db.WithContext(ctx).Where("code = ?", parentCode).First(&parent).Error; err != nil {
		return nil, fmt.Errorf("تعذر العثور على الحساب الرئيسي للمحافظ (%s): %v", parentCode, err)
	}

	// Check if this wallet already exists
	var existing models.Account
	err := m.db.WithContext(ctx).Where("reference_type = ? AND reference_id = ?", refType, refID).First(&existing).Error
	if err == nil {
		return &existing, nil // Already exists
	} else if !errorsIsRecordNotFound(err) {
		return nil, err
	}

	// Generate a short deterministic or random unique code suffix based on UUID
	shortUUID := strings.Split(refID.String(), "-")[0]
	code := fmt.Sprintf("%s-%s-%s", parentCode, refType[:3], strings.ToUpper(shortUUID))

	walletTitle := fmt.Sprintf("محفظة: %s", name)

	newWallet := &models.Account{
		Code:          code,
		Name:          walletTitle,
		Type:          models.Liability,
		NormalBalance: models.Credit,
		ParentID:      &parent.ID,
		ReferenceType: &refType,
		ReferenceID:   &refID,
		IsActive:      true,
	}

	if err := m.db.WithContext(ctx).Create(newWallet).Error; err != nil {
		return nil, fmt.Errorf("فشل في إنشاء حساب المحفظة: %v", err)
	}

	return newWallet, nil
}

func errorsIsRecordNotFound(err error) bool {
	return err == gorm.ErrRecordNotFound
}
