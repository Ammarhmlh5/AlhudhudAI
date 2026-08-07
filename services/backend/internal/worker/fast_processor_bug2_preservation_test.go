package worker

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"pgregory.net/rapid"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
)

// Bug 2 Preservation Tests
// **Validates: Requirements 3.12, 3.13, 3.14**
//
// **Property 2: Preservation** - GORM Operations Continue
//
// These tests observe and verify that existing GORM operations continue to work
// correctly after PgxPool is initialized. They test on UNFIXED code to establish
// baseline behavior, and must PASS on both UNFIXED and FIXED code.
//
// IMPORTANT: These tests verify that adding PgxPool does NOT break existing GORM functionality.

// TestProperty_Bug2_GormQueriesContinueToWork verifies GORM queries work
// **Validates: Requirements 3.12**
//
// This test MUST PASS on both UNFIXED and FIXED code.
func TestProperty_Bug2_GormQueriesContinueToWork(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	// Initialize database
	db.InitGormDB()
	require.NotNil(t, db.GormDB, "GormDB should be initialized")

	ctx := context.Background()

	rapid.Check(t, func(t *rapid.T) {
		// Generate test tenant data
		tenantName := rapid.StringN(5, 20, -1).Draw(t, "tenantName")
		// Strip control characters (e.g. 0x00) which PostgreSQL rejects in UTF8
		tenantName = strings.Map(func(r rune) rune {
			if r < 0x20 || r == 0x7f {
				return -1
			}
			return r
		}, tenantName)
		if tenantName == "" {
			tenantName = "Test Tenant"
		}

		// Create a valid test user to satisfy fk_tenants_user
		userID := uuid.New()
		err := db.GormDB.Exec(`
			INSERT INTO users (id, email, password_hash, user_type, is_active, is_verified, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, userID, "worker-"+userID.String()+"@test.com", "hash", "ADMIN", true, true, time.Now(), time.Now()).Error
		if err != nil {
			t.Fatalf("Failed to create test user: %v", err)
			return
		}
		defer db.GormDB.Exec("DELETE FROM users WHERE id = ?", userID)

		// Create a test tenant using GORM with unique company_serial
		tenant := &models.Tenant{
			ID:            uuid.New(),
			UserID:        userID,
			BusinessName:  tenantName,
			ContactName:   "Test Contact",
			Email:         "test@example.com",
			CompanySerial: uuid.New().String()[:8], // Unique serial
			IsActive:      true,
		}

		// ACT: Create tenant using GORM
		err = db.GormDB.WithContext(ctx).Create(tenant).Error
		
		// ASSERT: GORM create should work
		// On UNFIXED code: PASSES (GORM works independently)
		// On FIXED code: PASSES (GORM still works with PgxPool added)
		assert.NoError(t, err,
			"Preservation: GORM Create operation should continue to work. "+
				"This test verifies that adding PgxPool does not break GORM.")

		if err == nil {
			// Verify we can query it back
			var retrieved models.Tenant
			err = db.GormDB.WithContext(ctx).First(&retrieved, "id = ?", tenant.ID).Error
			
			assert.NoError(t, err, "GORM query should work")
			assert.Equal(t, tenant.BusinessName, retrieved.BusinessName, "Retrieved tenant should match")

			// Cleanup
			db.GormDB.Unscoped().Delete(&tenant)
		}
	})
}

// TestBug2_GormTransactionsContinueToWork verifies GORM transactions work
// **Validates: Requirements 3.14**
//
// This test MUST PASS on both UNFIXED and FIXED code.
func TestBug2_GormTransactionsContinueToWork(t *testing.T) {
	// Initialize database
	db.InitGormDB()
	require.NotNil(t, db.GormDB, "GormDB should be initialized")

	ctx := context.Background()

	// Create test data with unique company_serial
	tenant := &models.Tenant{
		ID:            uuid.New(),
		UserID:        createTestUser(t),
		BusinessName:  "Test Company",
		ContactName:   "Test Contact",
		Email:         "test@example.com",
		CompanySerial: uuid.New().String()[:8], // Unique serial
		IsActive:      true,
	}

	// ACT: Use GORM transaction
	err := db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(tenant).Error; err != nil {
			return err
		}
		
		// Verify within transaction
		var count int64
		if err := tx.Model(&models.Tenant{}).Where("id = ?", tenant.ID).Count(&count).Error; err != nil {
			return err
		}
		
		if count != 1 {
			t.Errorf("Expected 1 tenant, got %d", count)
		}
		
		return nil
	})

	// ASSERT: GORM transaction should work
	// On UNFIXED code: PASSES (GORM transactions work independently)
	// On FIXED code: PASSES (GORM transactions still work with PgxPool added)
	assert.NoError(t, err,
		"Preservation: GORM Transaction operations should continue to work. "+
			"This test verifies that adding PgxPool does not break GORM transactions.")

	// Cleanup
	if err == nil {
		db.GormDB.Unscoped().Delete(&tenant)
	}
}

// TestBug2_GormMigrationsContinueToWork verifies GORM migrations work
// **Validates: Requirements 3.13**
//
// This test MUST PASS on both UNFIXED and FIXED code.
func TestBug2_GormMigrationsContinueToWork(t *testing.T) {
	// Initialize database
	db.InitGormDB()
	require.NotNil(t, db.GormDB, "GormDB should be initialized")

	// ACT: Verify migrations ran successfully
	// Check if key tables exist
	hasTenants := db.GormDB.Migrator().HasTable(&models.Tenant{})
	hasUsers := db.GormDB.Migrator().HasTable(&models.User{})

	// ASSERT: GORM migrations should work
	// On UNFIXED code: PASSES (GORM migrations work independently)
	// On FIXED code: PASSES (GORM migrations still work with PgxPool added)
	assert.True(t, hasTenants,
		"Preservation: GORM migrations should continue to work. "+
			"Tenants table should exist after migrations.")
	assert.True(t, hasUsers,
		"Preservation: GORM migrations should continue to work. "+
			"Users table should exist after migrations.")
}

// TestBug2_GormQueryBuilderContinuesToWork verifies GORM query builder works
// **Validates: Requirements 3.12**
//
// This test MUST PASS on both UNFIXED and FIXED code.
func TestBug2_GormQueryBuilderContinuesToWork(t *testing.T) {
	// Initialize database
	db.InitGormDB()
	require.NotNil(t, db.GormDB, "GormDB should be initialized")

	ctx := context.Background()

	// Create test tenants with unique company_serial
	tenant1 := &models.Tenant{
		ID:            uuid.New(),
		UserID:        createTestUser(t),
		BusinessName:  "Active Company",
		ContactName:   "Test Contact",
		Email:         "active@example.com",
		CompanySerial: uuid.New().String()[:8], // Unique serial
		IsActive:      true,
	}
	tenant2 := &models.Tenant{
		ID:            uuid.New(),
		UserID:        createTestUser(t),
		BusinessName:  "Inactive Company",
		ContactName:   "Test Contact",
		Email:         "inactive@example.com",
		CompanySerial: uuid.New().String()[:8], // Unique serial
		IsActive:      false,
	}

	// Create tenants
	db.GormDB.WithContext(ctx).Create(tenant1)
	db.GormDB.WithContext(ctx).Create(tenant2)
	defer func() {
		db.GormDB.Unscoped().Delete(tenant1)
		db.GormDB.Unscoped().Delete(tenant2)
	}()

	// ACT: Use GORM query builder with WHERE clause
	var activeTenants []models.Tenant
	err := db.GormDB.WithContext(ctx).
		Where("is_active = ?", true).
		Where("business_name LIKE ?", "%Active%").
		Find(&activeTenants).Error

	// ASSERT: GORM query builder should work
	// On UNFIXED code: PASSES (GORM query builder works independently)
	// On FIXED code: PASSES (GORM query builder still works with PgxPool added)
	assert.NoError(t, err,
		"Preservation: GORM query builder should continue to work. "+
			"This test verifies that adding PgxPool does not break GORM query building.")

	// Verify we got the right tenant
	if err == nil {
		assert.GreaterOrEqual(t, len(activeTenants), 1,
			"Should find at least one active tenant")
		
		found := false
		for _, tenant := range activeTenants {
			if tenant.ID == tenant1.ID {
				found = true
				break
			}
		}
		assert.True(t, found, "Should find our test active tenant")
	}
}

// TestBug2_GormUpdatesContinueToWork verifies GORM updates work
// **Validates: Requirements 3.12**
//
// This test MUST PASS on both UNFIXED and FIXED code.
func TestBug2_GormUpdatesContinueToWork(t *testing.T) {
	// Initialize database
	db.InitGormDB()
	require.NotNil(t, db.GormDB, "GormDB should be initialized")

	ctx := context.Background()

	// Create test tenant with unique company_serial
	tenant := &models.Tenant{
		ID:            uuid.New(),
		UserID:        createTestUser(t),
		BusinessName:  "Original Name",
		ContactName:   "Test Contact",
		Email:         "test@example.com",
		CompanySerial: uuid.New().String()[:8], // Unique serial
		IsActive:      true,
	}

	err := db.GormDB.WithContext(ctx).Create(tenant).Error
	require.NoError(t, err)
	defer db.GormDB.Unscoped().Delete(tenant)

	// ACT: Update using GORM
	err = db.GormDB.WithContext(ctx).
		Model(&models.Tenant{}).
		Where("id = ?", tenant.ID).
		Update("business_name", "Updated Name").Error

	// ASSERT: GORM update should work
	// On UNFIXED code: PASSES (GORM updates work independently)
	// On FIXED code: PASSES (GORM updates still work with PgxPool added)
	assert.NoError(t, err,
		"Preservation: GORM Update operations should continue to work. "+
			"This test verifies that adding PgxPool does not break GORM updates.")

	// Verify update worked
	if err == nil {
		var updated models.Tenant
		db.GormDB.WithContext(ctx).First(&updated, "id = ?", tenant.ID)
		assert.Equal(t, "Updated Name", updated.BusinessName,
			"Tenant name should be updated")
	}
}

// TestBug2_GormDeletesContinueToWork verifies GORM deletes work
// **Validates: Requirements 3.12**
//
// This test MUST PASS on both UNFIXED and FIXED code.
func TestBug2_GormDeletesContinueToWork(t *testing.T) {
	// Initialize database
	db.InitGormDB()
	require.NotNil(t, db.GormDB, "GormDB should be initialized")

	ctx := context.Background()

	// Create test tenant with unique company_serial
	tenant := &models.Tenant{
		ID:            uuid.New(),
		UserID:        createTestUser(t),
		BusinessName:  "To Be Deleted",
		ContactName:   "Test Contact",
		Email:         "delete@example.com",
		CompanySerial: uuid.New().String()[:8], // Unique serial
		IsActive:      true,
	}

	err := db.GormDB.WithContext(ctx).Create(tenant).Error
	require.NoError(t, err)

	// ACT: Delete using GORM (soft delete)
	err = db.GormDB.WithContext(ctx).Delete(tenant).Error

	// ASSERT: GORM delete should work
	// On UNFIXED code: PASSES (GORM deletes work independently)
	// On FIXED code: PASSES (GORM deletes still work with PgxPool added)
	assert.NoError(t, err,
		"Preservation: GORM Delete operations should continue to work. "+
			"This test verifies that adding PgxPool does not break GORM deletes.")

	// Verify soft delete worked
	if err == nil {
		var count int64
		db.GormDB.WithContext(ctx).Model(&models.Tenant{}).
			Where("id = ?", tenant.ID).
			Count(&count)
		assert.Equal(t, int64(0), count,
			"Tenant should be soft deleted (not visible in normal queries)")

		// Verify it exists with Unscoped
		db.GormDB.Unscoped().Model(&models.Tenant{}).
			Where("id = ?", tenant.ID).
			Count(&count)
		assert.Equal(t, int64(1), count,
			"Tenant should still exist in database (soft deleted)")

		// Cleanup
		db.GormDB.Unscoped().Delete(tenant)
	}
}
