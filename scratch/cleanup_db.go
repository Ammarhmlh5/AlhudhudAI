package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
)

func main() {
	connStr := "postgres://postgres:Ammar1983hmlh%23%23@localhost:5433/al_hudhud_new?sslmode=disable"
	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer conn.Close(context.Background())

	queries := []string{
		"DROP TABLE IF EXISTS public.companies CASCADE",
		"DROP TABLE IF EXISTS public.company_rating_history CASCADE",
		"DROP TABLE IF EXISTS public.unmatched_payments CASCADE",
		"DROP TABLE IF EXISTS public.system_translations CASCADE",
		"DROP TABLE IF EXISTS public.payment_gateways CASCADE",
		"DROP TABLE IF EXISTS public.bank_accounts CASCADE", // Migration 081
		"DROP FUNCTION IF EXISTS public.provision_company_schema(UUID) CASCADE",
		"ALTER TABLE public.tenants DROP COLUMN IF EXISTS company_id",
		"ALTER TABLE public.audit_logs DROP COLUMN IF EXISTS company_id", // Migration 079 update
		// Check for other potential 4.0.0 tables
		"DROP TABLE IF EXISTS public.public_servers CASCADE", // Migration 083
	}

	for _, q := range queries {
		fmt.Printf("Executing: %s\n", q)
		_, err := conn.Exec(context.Background(), q)
		if err != nil {
			fmt.Printf("Warning: failed to execute %s: %v\n", q, err)
		}
	}

	// Drop any company_* schemas if they exist
	rows, err := conn.Query(context.Background(), "SELECT schema_name FROM information_schema.schemata WHERE schema_name LIKE 'company_%'")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var schemaName string
			if err := rows.Scan(&schemaName); err == nil {
				fmt.Printf("Dropping schema: %s\n", schemaName)
				_, _ = conn.Exec(context.Background(), fmt.Sprintf("DROP SCHEMA %s CASCADE", schemaName))
			}
		}
	}

	fmt.Println("\nDatabase cleanup completed.")
}
