//go:build ignore

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	// Build connection string from environment variables
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	sslmode := os.Getenv("DB_SSLMODE")
	
	// Validate required environment variables
	if host == "" || port == "" || user == "" || password == "" || dbname == "" {
		log.Fatal("Error: Required environment variables not set (DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME)")
	}
	
	if sslmode == "" {
		sslmode = "disable"
	}
	
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	// Connect to database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test connection
	err = db.Ping()
	if err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	fmt.Println("✓ Connected to database successfully")

	// Read migration file
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run apply_migration.go <migration_file>")
	}

	migrationFile := os.Args[1]
	content, err := os.ReadFile(migrationFile)
	if err != nil {
		log.Fatal("Failed to read migration file:", err)
	}

	fmt.Printf("Applying migration: %s\n", migrationFile)

	// Execute migration
	_, err = db.Exec(string(content))
	if err != nil {
		log.Fatal("Failed to execute migration:", err)
	}

	fmt.Println("✓ Migration applied successfully")
}
