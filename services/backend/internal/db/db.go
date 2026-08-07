package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDB() {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_SSLMODE"),
	)

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Error opening database: ", err)
	}

	// Cap the pool so leaked/unreleased connections cannot exhaust the server's
	// max_connections budget (was previously unbounded and caused 53300 storms).
	DB.SetMaxOpenConns(50)
	DB.SetMaxIdleConns(10)
	DB.SetConnMaxLifetime(30 * time.Minute)

	if err = DB.Ping(); err != nil {
		log.Fatal("Error connecting to the database: ", err)
	}

	log.Println("Successfully connected to PostgreSQL via sql.DB")

	// Create essential extensions
	_, err = DB.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`)
	if err != nil {
		log.Printf("Error creating uuid-ossp extension: %v", err)
	}
}

// CloseDB closes the database connections
func CloseDB() {
	if DB != nil {
		DB.Close()
	}
}
