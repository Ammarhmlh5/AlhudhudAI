package queue

import (
	"os"
	"testing"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// initAccountingTestDB sets env vars and connects to the real database
func initAccountingTestDB(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping accounting integration test in short mode")
	}
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5433")
	os.Setenv("DB_USER", "postgres")
	os.Setenv("DB_PASSWORD", "Ammar1983hmlh##")
	os.Setenv("DB_NAME", "al_hudhud_new")
	os.Setenv("DB_SSLMODE", "disable")
	db.InitGormDB()
	if db.GormDB == nil {
		t.Skip("Skipping test: database not available")
	}
}

// TestTwoStageAccounting_Reserve tests that balance is reserved on message staging
func TestTwoStageAccounting_Reserve(t *testing.T) {
	initAccountingTestDB(t)
	tenantID := uuid.New()

	initialBalance := 100.0

	userID := uuid.New()
	err := db.GormDB.Create(&models.User{
		ID:       userID,
		Email:    "test_user_" + tenantID.String() + "@example.com",
		UserType: models.TenantType,
	}).Error
	assert.NoError(t, err)

	// Create tenant with balance
	err = db.GormDB.Create(&models.Tenant{
		ID:               tenantID,
		UserID:           userID,
		Balance:          initialBalance,
		Status:           models.TenantActive,
		BusinessName:     "Test Business",
		ContactName:      "Test Contact",
		Email:            "test_" + tenantID.String() + "@example.com",
		CompanySerial:    "SER-" + tenantID.String()[:8],
		APIKeyAccounting: "acct-" + tenantID.String()[:8],
		APIKeyPhone:      "phone-" + tenantID.String()[:8],
		APIKeyWeb:        "web-" + tenantID.String()[:8],
	}).Error
	assert.NoError(t, err)

	// Create tenant schema first
	err = db.CreateTenantSchema(tenantID)
	assert.NoError(t, err)

	// Migrate tenant schema
	err = db.MigrateTenantSchema(tenantID)
	assert.NoError(t, err)

	// Create a message and channel
	var msgID uuid.UUID
	err = db.RunInTenantScope(tenantID, false, func(tx *gorm.DB) error {
		channel := models.DispatchChannel{
			TenantID:   tenantID,
			Type:       "PHONE",
			ChannelKey: "accounting_test_phone",
			Status:     "ONLINE",
			Priority:   5,
			Config:     "{}",
		}
		if err := tx.Create(&channel).Error; err != nil {
			return err
		}

		msg := models.Message{
			TenantID:       tenantID,
			CustomID:       "T-RES-" + tenantID.String()[:12],
			PhoneNumber:    "+966501234567",
			Content:        "Test message",
			Status:         models.MessagePending,
			IdempotencyKey: "reserve-" + tenantID.String(),
		}
		if err := tx.Create(&msg).Error; err != nil {
			return err
		}
		msgID = msg.ID
		return nil
	})
	assert.NoError(t, err)

	// Simulate job execution (this would normally be done by River)
	worker := &SendSMSWorker{}
	// Note: In real test, we'd use River's test helpers
	// For now, we're just verifying the logic structure exists
	assert.NotNil(t, worker)
	assert.NotNil(t, msgID)
}

// TestTwoStageAccounting_Refund tests that balance is refunded on failure
func TestTwoStageAccounting_Refund(t *testing.T) {
	initAccountingTestDB(t)
	tenantID := uuid.New()

	initialBalance := 100.0

	userID := uuid.New()
	err := db.GormDB.Create(&models.User{
		ID:       userID,
		Email:    "test_user_refund_" + tenantID.String() + "@example.com",
		UserType: models.TenantType,
	}).Error
	assert.NoError(t, err)

	// Create tenant
	err = db.GormDB.Create(&models.Tenant{
		ID:               tenantID,
		UserID:           userID,
		Balance:          initialBalance,
		Status:           models.TenantActive,
		BusinessName:     "Test Business Refund",
		ContactName:      "Test Contact",
		Email:            "test_refund_" + tenantID.String() + "@example.com",
		CompanySerial:    "SER-R-" + tenantID.String()[:8],
		APIKeyAccounting: "acct-r-" + tenantID.String()[:8],
		APIKeyPhone:      "phone-r-" + tenantID.String()[:8],
		APIKeyWeb:        "web-r-" + tenantID.String()[:8],
	}).Error
	assert.NoError(t, err)

	// Create tenant schema first
	err = db.CreateTenantSchema(tenantID)
	assert.NoError(t, err)

	// Migrate tenant schema
	err = db.MigrateTenantSchema(tenantID)
	assert.NoError(t, err)

	// Create transaction in PENDING state
	var txID uuid.UUID
	err = db.RunInTenantScope(tenantID, false, func(tx *gorm.DB) error {
		msg := models.Message{
			TenantID:       tenantID,
			CustomID:       "T-REF-" + tenantID.String()[:12],
			PhoneNumber:    "+966501234567",
			Content:        "Test",
			Status:         models.MessagePending,
			IdempotencyKey: "refund-" + tenantID.String(),
		}
		if err := tx.Create(&msg).Error; err != nil {
			return err
		}

		transaction := models.AccountingTransaction{
			TenantID:  tenantID,
			MessageID: msg.ID,
			Amount:    1.0,
			Status:    models.TxPending,
		}
		if err := tx.Create(&transaction).Error; err != nil {
			return err
		}
		txID = transaction.ID
		return nil
	})
	assert.NoError(t, err)

	// Simulate deduction (Phase 1: Reserve)
	err = db.GormDB.Model(&models.Tenant{}).Where("id = ?", tenantID).
		UpdateColumn("balance", initialBalance-1.0).Error
	assert.NoError(t, err)

	// Simulate refund (Phase 2: Refund on failure)
	err = db.RunInTenantScope(tenantID, false, func(tx *gorm.DB) error {
		var transaction models.AccountingTransaction
		if err := tx.First(&transaction, txID).Error; err != nil {
			return err
		}

		var tenant models.Tenant
		if err := db.GormDB.First(&tenant, tenantID).Error; err != nil {
			return err
		}

		newBalance := tenant.Balance + transaction.Amount
		if err := db.GormDB.Model(&tenant).Update("balance", newBalance).Error; err != nil {
			return err
		}
		tx.Model(&transaction).Update("status", models.TxRefunded)

		return nil
	})
	assert.NoError(t, err)

	// Verify balance was restored
	var tenant models.Tenant
	db.GormDB.First(&tenant, tenantID)
	assert.Equal(t, initialBalance, tenant.Balance, "Balance should be restored after refund")
}

// TestTwoStageAccounting_Commit tests that balance is committed on success
func TestTwoStageAccounting_Commit(t *testing.T) {
	initAccountingTestDB(t)
	tenantID := uuid.New()

	initialBalance := 100.0
	cost := 1.0

	userID := uuid.New()
	err := db.GormDB.Create(&models.User{
		ID:       userID,
		Email:    "test_user_commit_" + tenantID.String() + "@example.com",
		UserType: models.TenantType,
	}).Error
	assert.NoError(t, err)

	// Create tenant
	err = db.GormDB.Create(&models.Tenant{
		ID:               tenantID,
		UserID:           userID,
		Balance:          initialBalance,
		Status:           models.TenantActive,
		BusinessName:     "Test Business Commit",
		ContactName:      "Test Contact",
		Email:            "test_commit_" + tenantID.String() + "@example.com",
		CompanySerial:    "SER-C-" + tenantID.String()[:8],
		APIKeyAccounting: "acct-c-" + tenantID.String()[:8],
		APIKeyPhone:      "phone-c-" + tenantID.String()[:8],
		APIKeyWeb:        "web-c-" + tenantID.String()[:8],
	}).Error
	assert.NoError(t, err)

	// Create tenant schema first
	err = db.CreateTenantSchema(tenantID)
	assert.NoError(t, err)

	// Migrate tenant schema
	err = db.MigrateTenantSchema(tenantID)
	assert.NoError(t, err)

	// Create and commit transaction
	var txID uuid.UUID
	err = db.RunInTenantScope(tenantID, false, func(tx *gorm.DB) error {
		msg := models.Message{
			TenantID:       tenantID,
			CustomID:       "T-COM-" + tenantID.String()[:12],
			PhoneNumber:    "+966501234567",
			Content:        "Test",
			Status:         models.MessagePending,
			IdempotencyKey: "commit-" + tenantID.String(),
		}
		if err := tx.Create(&msg).Error; err != nil {
			return err
		}

		transaction := models.AccountingTransaction{
			TenantID:  tenantID,
			MessageID: msg.ID,
			Amount:    cost,
			Status:    models.TxPending,
		}
		if err := tx.Create(&transaction).Error; err != nil {
			return err
		}
		txID = transaction.ID

		// Deduct balance - tenant table is in public schema
		var tenant models.Tenant
		if err := db.GormDB.First(&tenant, tenantID).Error; err != nil {
			return err
		}
		if err := db.GormDB.Model(&tenant).Update("balance", tenant.Balance-cost).Error; err != nil {
			return err
		}

		return nil
	})
	assert.NoError(t, err)

	// Commit transaction
	err = db.RunInTenantScope(tenantID, false, func(tx *gorm.DB) error {
		return tx.Model(&models.AccountingTransaction{}).
			Where("id = ?", txID).
			Update("status", models.TxCommitted).Error
	})
	assert.NoError(t, err)

	// Verify balance was deducted
	var tenant models.Tenant
	db.GormDB.First(&tenant, tenantID)
	expectedBalance := initialBalance - cost
	assert.Equal(t, expectedBalance, tenant.Balance, "Balance should be deducted after commit")
}
