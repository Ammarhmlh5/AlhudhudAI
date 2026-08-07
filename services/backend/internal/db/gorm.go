package db

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var GormDB *gorm.DB
var PgxPool *pgxpool.Pool

// Get returns the GORM DB instance from context if available (transaction/schema scoped),
// otherwise returns the global instance
func Get(ctx context.Context) *gorm.DB {
	// Check if there is a db in context
	val := ctx.Value("db")
	if val == nil {
		return GormDB
	}

	if tx, ok := val.(*gorm.DB); ok {
		return tx
	}
	return GormDB
}

func InitGormDB() {
	if GormDB != nil {
		return
	}
	log := logger.GetLogger()

	// Try to load .env if DB_HOST is not set (likely running tests)
	if os.Getenv("DB_HOST") == "" {
		// Try loading from current directory or parents
		_ = godotenv.Load(".env")
		_ = godotenv.Load("../.env")
		_ = godotenv.Load("../../.env")
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s timezone=UTC",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_SSLMODE"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database", zap.Error(err))
	}

	GormDB = db
	log.Info("Connection to PostgreSQL established successfully via GORM")

	// Initialize PgxPool for pgx-based operations
	ctx := context.Background()
	pgxConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatal("Failed to parse pgxpool config", zap.Error(err))
	}

	// Allow tests (and deployments) to cap the pool so parallel test packages
	// do not exhaust the server's max_connections budget.
	maxConns := int32(50)
	if v := os.Getenv("DB_MAX_CONNS"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 32); err == nil && n > 0 {
			maxConns = int32(n)
		}
	}
	minConns := maxConns / 5
	if v := os.Getenv("DB_MIN_CONNS"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 32); err == nil && n >= 0 {
			minConns = int32(n)
		}
	}
	if minConns > maxConns {
		minConns = maxConns
	}

	pgxConfig.MaxConns = maxConns
	pgxConfig.MinConns = minConns

	PgxPool, err = pgxpool.NewWithConfig(ctx, pgxConfig)
	if err != nil {
		log.Fatal("Failed to create pgxpool", zap.Error(err))
	}

	if err = PgxPool.Ping(ctx); err != nil {
		log.Fatal("Failed to ping pgxpool", zap.Error(err))
	}

	log.Info("PgxPool initialized successfully",
		zap.Int32("max_conns", pgxConfig.MaxConns),
		zap.Int32("min_conns", pgxConfig.MinConns))

	// Cap the GORM connection pool too, so parallel test packages cannot
	// exhaust the server's max_connections budget.
	if sqlDB, err := GormDB.DB(); err == nil {
		sqlDB.SetMaxOpenConns(int(maxConns))
		sqlDB.SetMaxIdleConns(int(minConns))
		sqlDB.SetConnMaxLifetime(time.Hour)
	}

	// Run Migrations
	migrate()
}

func migrate() {
	log := logger.GetLogger()
	log.Info("Running auto-migrations for Global Standard models...")

	for _, table := range PublicTables {
		if err := GormDB.AutoMigrate(table); err != nil {
			errStr := err.Error()
			if strings.Contains(errStr, "42704") || strings.Contains(errStr, "does not exist") || strings.Contains(errStr, "0A000") {
				log.Warn("Ignored non-fatal migration error", zap.Error(err))
			} else {
				log.Error("Auto-migration encountered an error", zap.Error(err))
			}
		}
	}

	log.Info("Running auto-migrations for Tenant models in public schema...")
	for _, table := range TenantTables {
		if err := GormDB.AutoMigrate(table); err != nil {
			errStr := err.Error()
			if strings.Contains(errStr, "42704") || strings.Contains(errStr, "does not exist") || strings.Contains(errStr, "0A000") {
				log.Warn("Ignored non-fatal migration error", zap.Error(err))
			} else {
				log.Error("Auto-migration encountered an error", zap.Error(err))
			}
		}
	}

	log.Info("Auto-migrations completed")
}

// Close closes the database connections
func Close() {
	if PgxPool != nil {
		PgxPool.Close()
		PgxPool = nil
	}
	if GormDB != nil {
		sqlDB, err := GormDB.DB()
		if err == nil {
			sqlDB.Close()
		}
		GormDB = nil
	}
}
