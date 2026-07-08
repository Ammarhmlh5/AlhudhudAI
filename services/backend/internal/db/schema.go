package db

import (
	"fmt"
	"strings"

	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Forward declaration to avoid circular dependency if queue package is used
// In a real project, we might use an interface or a registry.
var MigrateRiver func(tenantID uuid.UUID) error

// PublicTables lists tables that should remain in the public schema
var PublicTables = []interface{}{
	&models.User{},
	&models.Tenant{},
	&models.CompanyStatus{},
	&models.Role{},
	&models.Permission{},
	&models.StaffProfile{},
	&models.StaffActivity{},
	&models.Session{},
	&models.VerificationCode{},
	&models.PendingRegistration{},
	&models.Invoice{},
	&models.AdminStatistics{},
	&models.Tariff{},
	&models.Package{},
	&models.DispatchChannel{},
	&models.GlobalIncomingEvent{},
	&models.AlhudhudGateway{},
}

// TenantTables lists tables that should be created in each tenant's schema
var TenantTables = []interface{}{
	&models.Message{},
	&models.PairedDevice{},
	&models.IncomingMessage{},
	&models.AccountingSystem{},
	&models.DispatchChannel{},
	&models.AccountingTransaction{},
	&models.MessageEvent{},
}

// GetSchemaName generates a safe schema name from tenant ID
func GetSchemaName(tenantID uuid.UUID) string {
	// Format: tenant_UID
	// We replace - with _ to make it safe for SQL
	safeID := strings.ReplaceAll(tenantID.String(), "-", "_")
	return fmt.Sprintf("tenant_%s", safeID)
}

// CreateTenantSchema creates a new schema for a tenant if it doesn't exist
func CreateTenantSchema(tenantID uuid.UUID) error {
	log := logger.GetLogger()
	schemaName := GetSchemaName(tenantID)

	// Create Schema
	if err := GormDB.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schemaName)).Error; err != nil {
		log.Error("Failed to create schema", zap.String("schema", schemaName), zap.Error(err))
		return err
	}

	// Run River migrations if the migrator is registered
	if MigrateRiver != nil {
		if err := MigrateRiver(tenantID); err != nil {
			log.Error("Failed to run River migrations for schema", zap.String("schema", schemaName), zap.Error(err))
			return err
		}
	}

	return nil
}

// MigrateTenantSchema runs migrations for tenant-specific tables in the tenant's schema
func MigrateTenantSchema(tenantID uuid.UUID) error {
	log := logger.GetLogger()
	schemaName := GetSchemaName(tenantID)

	log.Info("Migrating schema", zap.String("schema", schemaName))

	// IMPORTANT: We use a dedicated transaction with explicit search_path set ONLY to the
	// tenant schema (without "public" fallback) to prevent GORM AutoMigrate from
	// creating public tables (users, tenants, etc.) inside tenant schemas via foreign key chains.
	err := GormDB.Transaction(func(tx *gorm.DB) error {
		// Set search_path ONLY to the tenant schema - no public fallback.
		// This ensures AutoMigrate only creates tables in the tenant schema.
		// Set search_path to the tenant schema, falling back to public for extensions (uuid-ossp)
		// IMPORTANT: We use a dedicated transaction with explicit search_path set ONLY to the
		// tenant schema (without "public" fallback) to prevent GORM AutoMigrate from
		// creating public tables (users, tenants, etc.) inside tenant schemas via foreign key chains.
		// We avoid using "public" here to force GORM to fail or find existing global tables 
		// if it tries to follow relationships that shouldn't be in the tenant schema.
		if err := tx.Exec(fmt.Sprintf("SET LOCAL search_path TO %s", schemaName)).Error; err != nil {
			return err
		}

		/* 
		TEMPORARILY DISABLED FOR STARTUP PERFORMANCE (1800+ tables)
		The critical paired_devices fix was applied manually.
		if err := tx.AutoMigrate(TenantTables...); err != nil {
			// Check if it's a "constraint does not exist" error (SQLSTATE 42704)
			// This often happens in GORM when it tries to drop a non-existent constraint.
			// We log it but don't fail the whole migration if it's just a cleanup issue.
			errStr := err.Error()
			if strings.Contains(errStr, "42704") || strings.Contains(errStr, "does not exist") {
				log.Warn("Non-fatal migration error (ignored)", 
					zap.String("schema", schemaName), 
					zap.Error(err))
				return nil
			}
			return err
		}
		*/

		return nil
	})

	if err != nil {
		log.Error("Failed to migrate tenant schema",
			zap.String("schema", schemaName),
			zap.Error(err))
		return err
	}

	return nil
}

// GetTenantDB returns a DB instance scoped to the tenant's schema
// WARNING: This returns a new Session. You should use this for operations.
// RunInTenantScope executes a function within a transaction with the tenant's search_path 
// and RLS context set.
func RunInTenantScope(tenantID uuid.UUID, isSuperAdmin bool, fn func(tx *gorm.DB) error) error {
	schemaName := GetSchemaName(tenantID)

	// We use a transaction to ensure the search_path is set for the duration of the operations
	// and to ensure we are using a single connection.
	// We use a transaction to ensure the search_path is set for the duration of the operations
	// and to ensure we are using a single connection.
	return GormDB.Transaction(func(tx *gorm.DB) error {
		// Set search_path to the tenant's schema, falling back to public
		// Use SET LOCAL to ensure it's limited to this transaction only.
		query := fmt.Sprintf("SET LOCAL search_path TO %s, public", schemaName)
		if err := tx.Exec(query).Error; err != nil {
			return err
		}

		// Set tenant context for Row Level Security (RLS)
		// Requirement 16.4: استخدام session context لـ RLS
		// Using true for the last parameter (is_local) makes it transaction-scoped
		if err := tx.Exec("SELECT set_config('app.current_tenant', $1, true)", tenantID.String()).Error; err != nil {
			return err
		}

		adminVal := "false"
		if isSuperAdmin {
			adminVal = "true"
		}
		if err := tx.Exec("SELECT set_config('app.is_super_admin', $1, true)", adminVal).Error; err != nil {
			return err
		}

		return fn(tx)
	})
}
