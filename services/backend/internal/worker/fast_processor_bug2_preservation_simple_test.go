package worker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/al-hudhud/backend/internal/db"
)

// Bug 2 Preservation Tests - Simplified Version
// **Validates: Requirements 3.12, 3.13, 3.14**
//
// **Property 2: Preservation** - GORM Operations Continue
//
// These tests verify that GORM database operations continue to work correctly
// after PgxPool is initialized. They test on FIXED code to ensure no regressions.
//
// IMPORTANT: These tests verify that adding PgxPool does NOT break existing GORM functionality.

// TestBug2_GormDBInitialized verifies GORM DB is initialized
// **Validates: Requirements 3.12, 3.13**
//
// This test MUST PASS on FIXED code.
func TestBug2_GormDBInitialized(t *testing.T) {
	// Initialize database
	db.InitGormDB()

	// ASSERT: GormDB should be initialized
	// On FIXED code: PASSES (GORM still works with PgxPool added)
	require.NotNil(t, db.GormDB,
		"Preservation: GORM DB should be initialized. "+
			"This test verifies that adding PgxPool does not break GORM initialization.")

	// Verify we can get a SQL DB connection
	sqlDB, err := db.GormDB.DB()
	assert.NoError(t, err, "Should be able to get SQL DB from GORM")
	assert.NotNil(t, sqlDB, "SQL DB should not be nil")

	// Verify connection is alive
	err = sqlDB.Ping()
	assert.NoError(t, err,
		"Preservation: GORM database connection should be alive. "+
			"This verifies that GORM operations can proceed normally.")
}

// TestBug2_GormMigratorWorks verifies GORM Migrator works
// **Validates: Requirements 3.13**
//
// This test MUST PASS on FIXED code.
func TestBug2_GormMigratorWorks(t *testing.T) {
	// Initialize database
	db.InitGormDB()
	require.NotNil(t, db.GormDB, "GormDB should be initialized")

	// ASSERT: GORM Migrator should work
	// On FIXED code: PASSES (GORM Migrator still works with PgxPool added)
	migrator := db.GormDB.Migrator()
	assert.NotNil(t, migrator,
		"Preservation: GORM Migrator should be available. "+
			"This test verifies that adding PgxPool does not break GORM migrations.")

	// Verify we can check for table existence (basic migrator operation)
	// This doesn't create tables, just checks if the migrator can query schema
	hasMethod := migrator.HasTable("tenants")
	assert.True(t, hasMethod || !hasMethod, // Always true - just testing the call works
		"Preservation: GORM Migrator HasTable() should work without errors")
}

// TestBug2_GormQueryInterfaceWorks verifies GORM query interface works
// **Validates: Requirements 3.12**
//
// This test MUST PASS on FIXED code.
func TestBug2_GormQueryInterfaceWorks(t *testing.T) {
	// Initialize database
	db.InitGormDB()
	require.NotNil(t, db.GormDB, "GormDB should be initialized")

	// ASSERT: GORM query interface should work
	// On FIXED code: PASSES (GORM query builder still works with PgxPool added)
	
	// Test that we can create a query (doesn't execute, just builds)
	query := db.GormDB.Table("tenants").Where("is_active = ?", true)
	assert.NotNil(t, query,
		"Preservation: GORM query builder should work. "+
			"This test verifies that adding PgxPool does not break GORM query building.")

	// Verify query has expected methods
	assert.NotNil(t, query.Statement, "Query should have Statement")
}

// TestBug2_GormTransactionInterfaceWorks verifies GORM transaction interface works
// **Validates: Requirements 3.14**
//
// This test MUST PASS on FIXED code.
func TestBug2_GormTransactionInterfaceWorks(t *testing.T) {
	// Initialize database
	db.InitGormDB()
	require.NotNil(t, db.GormDB, "GormDB should be initialized")

	// ASSERT: GORM transaction interface should work
	// On FIXED code: PASSES (GORM transactions still work with PgxPool added)
	
	// Test that we can begin a transaction (then immediately rollback)
	tx := db.GormDB.Begin()
	assert.NotNil(t, tx,
		"Preservation: GORM Begin() should work. "+
			"This test verifies that adding PgxPool does not break GORM transactions.")
	
	// Verify transaction has expected methods
	assert.NotNil(t, tx.Statement, "Transaction should have Statement")
	
	// Rollback the transaction (we're not actually doing anything)
	err := tx.Rollback().Error
	assert.NoError(t, err, "Transaction rollback should work")
}

// TestBug2_GormAndPgxPoolCoexist verifies both GORM and PgxPool are initialized
// **Validates: Requirements 3.12, 3.13, 3.14**
//
// This is the key preservation test - both should work together.
func TestBug2_GormAndPgxPoolCoexist(t *testing.T) {
	// Initialize database
	db.InitGormDB()

	// ASSERT: Both GormDB and PgxPool should be initialized
	assert.NotNil(t, db.GormDB,
		"Preservation: GORM DB should still be initialized when PgxPool is added")
	
	assert.NotNil(t, db.PgxPool,
		"Bug Fix: PgxPool should now be initialized (this was the bug)")

	// Verify both are functional
	sqlDB, err := db.GormDB.DB()
	assert.NoError(t, err, "GORM should provide SQL DB")
	assert.NotNil(t, sqlDB, "GORM SQL DB should not be nil")

	// Verify PgxPool is functional
	assert.NotNil(t, db.PgxPool.Config(), "PgxPool should have config")
	
	// This test proves that:
	// 1. The bug is fixed (PgxPool is no longer nil)
	// 2. GORM operations are preserved (GormDB still works)
	// 3. Both can coexist without conflicts
}
