//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/queue"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load(".env")
	db.InitDB()
	db.InitGormDB()

	// Register the migrator
	db.MigrateRiver = func(tenantID uuid.UUID) error {
		schemaName := db.GetSchemaName(tenantID)
		return queue.MigrateSchema(context.Background(), schemaName)
	}

	var tenantIDs []uuid.UUID
	
	// Query current tenants
	rows, err := db.DB.Query("SELECT id FROM tenants")
	if err != nil {
		log.Fatalf("Failed to query tenants: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			log.Printf("Error scanning tenant ID: %v", err)
			continue
		}
		tenantIDs = append(tenantIDs, id)
	}

	fmt.Printf("Full Synchronization for %d tenants...\n", len(tenantIDs))
	
	// 0. Fix roles and permissions constraints in public
	fmt.Println("Ensuring constraints in public schema...")
	_, _ = db.DB.Exec(`
		DO $$ 
		BEGIN 
			-- Roles
			IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'roles_name_unique') THEN
				ALTER TABLE roles RENAME CONSTRAINT roles_name_unique TO uni_roles_name;
			ELSIF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uni_roles_name') THEN
				IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_roles_name') THEN
					DROP INDEX IF EXISTS idx_roles_name;
				END IF;
				ALTER TABLE roles ADD CONSTRAINT uni_roles_name UNIQUE (name);
			END IF;

			-- Permissions
			IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'permissions_name_unique') THEN
				ALTER TABLE permissions RENAME CONSTRAINT permissions_name_unique TO uni_permissions_name;
			ELSIF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uni_permissions_name') THEN
				IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_permissions_name') THEN
					DROP INDEX IF EXISTS idx_permissions_name;
				END IF;
				ALTER TABLE permissions ADD CONSTRAINT uni_permissions_name UNIQUE (name);
			END IF;
		END $$;`)

	tables := []string{"messages", "paired_devices", "incoming_messages", "accounting_systems", "dispatch_channels", "accounting_transactions", "message_events"}

	for _, id := range tenantIDs {
		schemaName := db.GetSchemaName(id)
		fmt.Printf("\nProcessing schema: %s\n", schemaName)
		
		// 1. Ensure schema exists
		_, _ = db.DB.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schemaName))

		// 2. Clone tables from public if they don't exist OR add missing columns
		for _, table := range tables {
			fmt.Printf("  Unifying %s...\n", table)
			
			// Use CREATE TABLE LIKE as a base for missing tables
			query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (LIKE public.%s INCLUDING ALL)", schemaName, table, table)
			_, err := db.DB.Exec(query)
			if err != nil {
				fmt.Printf("    Notice: could not use LIKE for %s (already exists?): %v\n", table, err)
			}
			
			// Always sync columns just in case table existed but was old
			addMissingColumns(schemaName, table)
		}

		// 3. Run River migrations
		fmt.Printf("  Step 4: Running River migrations...\n")
		_ = queue.MigrateSchema(context.Background(), schemaName)
	}
	fmt.Println("\nSynchronization complete.")
}

func addMissingColumns(targetSchema, tableName string) {
	query := fmt.Sprintf(`
		SELECT column_name, udt_name, character_maximum_length, column_default, is_nullable
		FROM information_schema.columns 
		WHERE table_schema = 'public' AND table_name = '%s'
		AND column_name NOT IN (
			SELECT column_name FROM information_schema.columns 
			WHERE table_schema = '%s' AND table_name = '%s'
		)`, tableName, targetSchema, tableName)

	rows, err := db.DB.Query(query)
	if err != nil {
		fmt.Printf("    Error querying missing columns for %s.%s: %v\n", targetSchema, tableName, err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var col, dtype, isNullable string
		var maxLen *int
		var def *string
		if err := rows.Scan(&col, &dtype, &maxLen, &def, &isNullable); err != nil {
			fmt.Printf("    Error scanning column info: %v\n", err)
			continue
		}

		fmt.Printf("    Adding missing column: %s (%s)\n", col, dtype)
		
		alterQuery := fmt.Sprintf("ALTER TABLE %s.%s ADD COLUMN %s %s", targetSchema, tableName, col, dtype)
		if dtype == "varchar" && maxLen != nil {
			alterQuery = fmt.Sprintf("ALTER TABLE %s.%s ADD COLUMN %s varchar(%d)", targetSchema, tableName, col, *maxLen)
		}
		
		if def != nil {
			alterQuery += fmt.Sprintf(" DEFAULT %s", *def)
		}

		if isNullable == "NO" {
			// If not nullable, we use the default for existing rows if available
		}

		_, err = db.DB.Exec(alterQuery)
		if err != nil {
			fmt.Printf("      Error adding %s: %v\n", col, err)
		} else {
			fmt.Printf("      Fixed %s successfully\n", col)
		}
	}
}
