package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println(".env file not found")
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_SSLMODE"),
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Aggressively drop tables that GORM will now manage
	tables := []string{
		"sessions",
		"staff_activities",
		"messages",
		"paired_devices",
		"staff_profiles",
		"role_permissions",
		"permissions",
		"roles",
		"tenants",
		"users",
	}

	for _, t := range tables {
		log.Printf("Dropping table: %s", t)
		_, err := db.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS %s CASCADE;`, t))
		if err != nil {
			log.Printf("Error dropping table %s: %v", t, err)
		}
	}

	log.Println("Database reset completed. Path cleared for GORM.")
}
