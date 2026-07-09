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

	rows, err := conn.Query(context.Background(), "SELECT table_name FROM information_schema.tables WHERE table_schema = 'public'")
	if err != nil {
		log.Fatalf("Query failed: %v\n", err)
	}
	defer rows.Close()

	fmt.Println("Tables in public schema:")
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("- %s\n", tableName)
	}

	fmt.Println("\nChecking for company_id in tenants table:")
	var hasCompanyID bool
	err = conn.QueryRow(context.Background(), "SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tenants' AND column_name = 'company_id')").Scan(&hasCompanyID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("tenants.company_id exists: %v\n", hasCompanyID)

	fmt.Println("\nChecking for provision_company_schema function:")
	var hasFunc bool
	err = conn.QueryRow(context.Background(), "SELECT EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON p.pronamespace = n.oid WHERE n.nspname = 'public' AND p.proname = 'provision_company_schema')").Scan(&hasFunc)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("public.provision_company_schema exists: %v\n", hasFunc)
}
