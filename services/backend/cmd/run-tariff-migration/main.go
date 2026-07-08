package main

import (
	"context"
	"fmt"
	"os"

	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger.InitLogger()
	log := logger.GetLogger()
	defer logger.Sync()

	log.Info("Starting Tariff Table Migration")

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Warn(".env file not found, using system environment variables")
	}

	// Build connection string
	connString := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_SSLMODE"),
	)

	// Connect to database
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		log.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer pool.Close()

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		log.Fatal("Failed to ping database", zap.Error(err))
	}
	log.Info("Connected to database successfully")

	// Read the tariff migration file
	migrationFile := "migrations/010_tariffs_table.sql"
	log.Info("Reading migration file", zap.String("file", migrationFile))

	content, err := os.ReadFile(migrationFile)
	if err != nil {
		log.Fatal("Failed to read migration file", zap.Error(err))
	}

	sql := string(content)

	// Execute the migration
	log.Info("Executing tariff table migration...")
	_, err = pool.Exec(ctx, sql)
	if err != nil {
		log.Fatal("Failed to execute migration", zap.Error(err))
	}

	log.Info("✓ Tariff table migration completed successfully!")
	
	// Verify the table was created
	var exists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'tariffs'
		)
	`).Scan(&exists)
	
	if err != nil {
		log.Error("Failed to verify table creation", zap.Error(err))
	} else if exists {
		log.Info("✓ Tariff table verified in database")
	} else {
		log.Warn("⚠ Tariff table not found after migration")
	}
}
