package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	logger.InitLogger()
	
	log.Println("--- Starting Migrations ---")

	// 1. Run GORM Auto-Migrations
	log.Println("1. Initializing GORM and running auto-migrations...")
	db.InitGormDB()
	
	// 2. Run Manual SQL Migrations
	log.Println("2. Running manual SQL migrations from migrations/ directory...")
	runSqlMigrations()

	log.Println("--- Migrations completed successfully ---")
}

func runSqlMigrations() {
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
		log.Fatal("Failed to connect for SQL migrations: ", err)
	}
	defer conn.Close(ctx)

	// Get all .sql files from migrations directory
	files, err := os.ReadDir("migrations")
	if err != nil {
		log.Printf("Warning: failed to read migrations directory: %v", err)
		return
	}

	var sqlFiles []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".sql") {
			sqlFiles = append(sqlFiles, f.Name())
		}
	}
	sort.Strings(sqlFiles)

	for _, filename := range sqlFiles {
		log.Printf("Executing migration: %s", filename)
		content, err := os.ReadFile(filepath.Join("migrations", filename))
		if err != nil {
			log.Fatalf("Failed to read migration %s: %v", filename, err)
		}

		_, err = conn.Exec(ctx, string(content))
		if err != nil {
			// If it fails, log and stop (atomic block inside SQL usually handles rollback)
			log.Fatalf("Migration %s failed: %v", filename, err)
		}
	}
}
