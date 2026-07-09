package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go <email>")
	}
	email := os.Args[1]

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

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var code string
	// Get latest OTP
	err = db.QueryRow("SELECT code FROM verification_codes WHERE target = $1 ORDER BY created_at DESC LIMIT 1", email).Scan(&code)
	if err != nil {
		log.Fatal("Error fetching OTP:", err)
	}

	fmt.Print(code)
}
