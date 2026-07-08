//go:build ignore

package main

import (
	"fmt"
	"log"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load(".env")
	db.InitDB()
	db.InitGormDB()

	var tenantIDs []uuid.UUID
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

	// Tables that should ONLY exist in public schema
	globalTables := []string{
		"users", "tenants", "roles", "permissions", 
		"staff_profiles", "staff_activities", "sessions", 
		"verification_codes", "pending_registrations", "invoices", 
		"admin_statistics", "audit_trail", "data_deletion_requests",
		"cross_tenant_access_logs", "fraud_alerts", "promo_code_usage",
		"promo_codes", "support_tickets", "ticket_attachments",
		"ticket_comments", "user_consents", "user_roles",
	}

	fmt.Printf("Cleaning up stray global tables from %d tenant schemas...\n", len(tenantIDs))

	for _, id := range tenantIDs {
		schemaName := db.GetSchemaName(id)
		fmt.Printf("\nProcessing schema: %s\n", schemaName)

		for _, table := range globalTables {
			// Check if table exists in this schema (and is NOT public)
			var exists bool
			err := db.DB.QueryRow(`
				SELECT EXISTS (
					SELECT 1 FROM information_schema.tables 
					WHERE table_schema = $1 AND table_name = $2
				)`, schemaName, table).Scan(&exists)
			
			if err != nil {
				fmt.Printf("    Error checking %s: %v\n", table, err)
				continue
			}

			if exists {
				fmt.Printf("    Found stray table: %s. Dropping...\n", table)
				// Drop with CASCADE because there might be foreign keys between stray tables
				_, err := db.DB.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s.%s CASCADE", schemaName, table))
				if err != nil {
					fmt.Printf("      Error dropping %s: %v\n", table, err)
				} else {
					fmt.Printf("      Dropped %s successfully\n", table)
				}
			}
		}
	}

	fmt.Println("\nCleanup complete.")
}
