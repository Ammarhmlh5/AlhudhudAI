package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/al-hudhud/backend/internal/subscription"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

// Config holds runtime configuration loaded from the environment.
type Config struct {
	Port             string
	DBHost           string
	DBPort           string
	DBUser           string
	DBPassword       string
	DBName           string
	DBSSLMode        string
	JWTSecret        string
	JWTExpiration    time.Duration
	AllowedOrigins   []string
	RegistrationCode string
}

func loadConfig() (*Config, error) {
	_ = godotenv.Load(".env", ".env.development")

	cfg := &Config{
		Port:             getEnv("PORT", "8085"),
		DBHost:           getEnv("DB_HOST", "localhost"),
		DBPort:           getEnv("DB_PORT", "5432"),
		DBUser:           getEnv("DB_USER", "postgres"),
		DBPassword:       getEnv("DB_PASSWORD", ""),
		DBName:           getEnv("DB_NAME", "al_hudhud"),
		DBSSLMode:        getEnv("DB_SSLMODE", "disable"),
		JWTSecret:        getEnv("JWT_SECRET", ""),
		JWTExpiration:    24 * time.Hour,
		RegistrationCode: getEnv("REGISTRATION_CODE", ""),
	}

	if exp, err := time.ParseDuration(getEnv("JWT_EXPIRATION", "24h")); err == nil {
		cfg.JWTExpiration = exp
	}

	origins := strings.Split(getEnv("ALLOWED_ORIGINS", "http://localhost:5175"), ",")
	for _, o := range origins {
		o = strings.TrimSpace(o)
		if o != "" {
			cfg.AllowedOrigins = append(cfg.AllowedOrigins, o)
		}
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	defer logger.Sync()

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
	)

	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		logger.Fatal("failed to parse db config", zap.Error(err))
	}
	if maxConns, err := strconv.Atoi(getEnv("DB_MAX_OPEN_CONNS", "50")); err == nil {
		poolCfg.MaxConns = int32(maxConns)
	}
	if maxIdle, err := strconv.Atoi(getEnv("DB_MAX_IDLE_CONNS", "10")); err == nil {
		poolCfg.MinConns = 0
		_ = maxIdle
	}
	if lifetime, err := strconv.Atoi(getEnv("DB_CONN_MAX_LIFETIME", "3600")); err == nil {
		poolCfg.MaxConnLifetime = time.Duration(lifetime) * time.Second
	}

	ctx := context.Background()
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		logger.Fatal("failed to create pool", zap.Error(err))
	}
	if err := pool.Ping(ctx); err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	logger.Info("connected to database")

	// Ensure the company_users table exists (bootstrap).
	if err := ensureSchema(ctx, pool); err != nil {
		logger.Fatal("failed to ensure schema", zap.Error(err))
	}

	subManager := subscription.NewManager(pool, logger)

	srv := &server{
		cfg:         cfg,
		pool:        pool,
		logger:      logger,
		subManager:  subManager,
		jwtSecret:   []byte(cfg.JWTSecret),
		jwtLifetime: cfg.JWTExpiration,
	}

	router := gin.New()
	router.Use(gin.Recovery(), srv.corsMiddleware())

	api := router.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", srv.handleRegister)
			auth.POST("/login", srv.handleLogin)
			auth.GET("/me", srv.authMiddleware(), srv.handleMe)
		}

		protected := api.Group("")
		protected.Use(srv.authMiddleware())
		{
			protected.GET("/packages", srv.handleListPackages)
			protected.POST("/subscriptions", srv.handleSubscribe)
			protected.GET("/subscriptions/active", srv.handleActiveSubscription)
			protected.GET("/subscriptions/usage", srv.handleUsage)
			protected.PUT("/subscriptions/:id/cancel", srv.handleCancelAutoRenew)
			protected.PUT("/subscriptions/:id/enable-auto-renew", srv.handleEnableAutoRenew)
			protected.GET("/balance", srv.handleBalance)
			protected.GET("/transactions", srv.handleTransactions)
		}
	}

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	logger.Info("company-api listening", zap.String("port", cfg.Port))
	if err := router.Run(":" + cfg.Port); err != nil {
		logger.Fatal("server failed", zap.Error(err))
	}
}

func ensureSchema(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS company_users (
			id BIGSERIAL PRIMARY KEY,
			tenant_id UUID NOT NULL,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			full_name TEXT,
			is_active BOOLEAN NOT NULL DEFAULT true,
			last_login_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create company_users: %w", err)
	}
	_, err = pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_company_users_tenant ON company_users(tenant_id)
	`)
	return err
}

// withTenantTx runs fn inside a transaction with the company's tenant context
// set for Row Level Security (same pattern as the main backend).
func (s *server) withTenantTx(ctx context.Context, tenantID uuid.UUID, fn func(tx pgx.Tx) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID.String()); err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
