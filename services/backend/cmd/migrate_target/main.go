package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	ctx := context.Background()
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_SSLMODE"),
	)

	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		log.Fatal("Failed to connect: ", err)
	}
	defer conn.Close(ctx)

	filename := "migrations/075_admin_reporting_and_events.sql"
	log.Printf("Executing target migration: %s", filename)
	content, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Failed to read migration: %v", err)
	}

	_, err = conn.Exec(ctx, string(content))
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Target migration applied successfully.")
}
