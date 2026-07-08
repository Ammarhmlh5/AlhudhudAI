package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	// Database connection string
	connStr := "host=localhost port=5432 user=postgres password=Ammar1983hmlh## dbname=al_hudhud_new sslmode=disable"

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Read migration file
	migrationFile, err := os.ReadFile("migrations/078_message_delivery_logs.sql")
	if err != nil {
		log.Fatal("Failed to read migration file:", err)
	}

	// Execute migration
	_, err = db.Exec(string(migrationFile))
	if err != nil {
		log.Fatal("Migration failed:", err)
	}

	fmt.Println("✅ Migration 078_message_delivery_logs.sql executed successfully!")
}