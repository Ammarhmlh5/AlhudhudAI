package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/al-hudhud/backend/internal/config"
	"github.com/al-hudhud/backend/internal/errors"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

// DB holds database connections
type DB struct {
	Pool           *pgxpool.Pool
	SqlDB          *sql.DB
	Logger         *zap.Logger
	circuitBreaker *errors.CircuitBreaker // Circuit breaker for database operations
}

// NewDatabase creates a new database connection with connection pool
func NewDatabase(cfg *config.Config, logger *zap.Logger) (*DB, error) {
	// Create pgxpool config
	poolConfig, err := pgxpool.ParseConfig(cfg.GetDatabaseDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Configure connection pool (Requirements 12.1-12.7)
	poolConfig.MaxConns = int32(cfg.Database.MaxOpenConns)
	poolConfig.MinConns = int32(cfg.Database.MaxIdleConns)
	poolConfig.MaxConnLifetime = cfg.Database.ConnMaxLifetime
	poolConfig.MaxConnIdleTime = cfg.Database.ConnMaxIdleTime
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	// Create connection pool
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create sql.DB for compatibility with existing code
	sqlDB, err := sql.Open("postgres", cfg.GetDatabaseDSN())
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to open sql.DB: %w", err)
	}

	// Configure sql.DB connection pool
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.Database.ConnMaxIdleTime)

	logger.Info("Database connection established",
		zap.String("host", cfg.Database.Host),
		zap.String("port", cfg.Database.Port),
		zap.String("database", cfg.Database.Name),
		zap.Int("max_open_conns", cfg.Database.MaxOpenConns),
		zap.Int("max_idle_conns", cfg.Database.MaxIdleConns),
		zap.Duration("conn_max_lifetime", cfg.Database.ConnMaxLifetime),
		zap.Duration("conn_max_idle_time", cfg.Database.ConnMaxIdleTime),
	)

	// Initialize circuit breaker for database operations
	// Requirement 19.3: Circuit Breaker للحماية من فشل الخدمات الخارجية
	// Max failures: 5, Timeout: 30 seconds
	circuitBreaker := errors.NewCircuitBreaker(5, 30*time.Second)

	return &DB{
		Pool:           pool,
		SqlDB:          sqlDB,
		Logger:         logger,
		circuitBreaker: circuitBreaker,
	}, nil
}

// NewFromPool creates a DB from an existing pgxpool.Pool (useful for testing).
func NewFromPool(pool *pgxpool.Pool) *DB {
	return &DB{
		Pool:           pool,
		circuitBreaker: errors.NewCircuitBreaker(5, 30*time.Second),
	}
}

// Close closes all database connections
func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
	if db.SqlDB != nil {
		db.SqlDB.Close()
	}
	if db.Logger != nil {
		db.Logger.Info("Database connections closed")
	}
}

// GetStats returns connection pool statistics
func (db *DB) GetStats() map[string]interface{} {
	stats := db.Pool.Stat()
	sqlStats := db.SqlDB.Stats()

	return map[string]interface{}{
		"pgxpool": map[string]interface{}{
			"acquired_conns":     stats.AcquiredConns(),
			"canceled_acquire":   stats.CanceledAcquireCount(),
			"constructing_conns": stats.ConstructingConns(),
			"empty_acquire":      stats.EmptyAcquireCount(),
			"idle_conns":         stats.IdleConns(),
			"max_conns":          stats.MaxConns(),
			"total_conns":        stats.TotalConns(),
		},
		"sql_db": map[string]interface{}{
			"max_open_conns":    sqlStats.MaxOpenConnections,
			"open_conns":        sqlStats.OpenConnections,
			"in_use":            sqlStats.InUse,
			"idle":              sqlStats.Idle,
			"wait_count":        sqlStats.WaitCount,
			"wait_duration_ms":  sqlStats.WaitDuration.Milliseconds(),
			"max_idle_closed":   sqlStats.MaxIdleClosed,
			"max_lifetime_closed": sqlStats.MaxLifetimeClosed,
		},
	}
}

// HealthCheck performs a health check on the database
func (db *DB) HealthCheck(ctx context.Context) error {
	// Check pgxpool
	if err := db.Pool.Ping(ctx); err != nil {
		return fmt.Errorf("pgxpool ping failed: %w", err)
	}

	// Check sql.DB
	if err := db.SqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("sql.DB ping failed: %w", err)
	}

	return nil
}

// ExecuteWithCircuitBreaker executes a database operation with circuit breaker protection
// Requirement 19.3: استخدام Circuit Breaker لحماية من فشل قاعدة البيانات
func (db *DB) ExecuteWithCircuitBreaker(ctx context.Context, operation func() error) error {
	return db.circuitBreaker.Call(ctx, operation)
}

// QueryWithCircuitBreaker executes a query with circuit breaker protection
// Requirement 19.3: استخدام Circuit Breaker لحماية من فشل قاعدة البيانات
func (db *DB) QueryWithCircuitBreaker(ctx context.Context, query func() (interface{}, error)) (interface{}, error) {
	return db.circuitBreaker.CallWithResult(ctx, query)
}

// GetCircuitBreakerState returns the current state of the circuit breaker
func (db *DB) GetCircuitBreakerState() errors.CircuitState {
	return db.circuitBreaker.GetState()
}

// IsCircuitBreakerOpen checks if the circuit breaker is open
func (db *DB) IsCircuitBreakerOpen() bool {
	return db.circuitBreaker.IsOpen()
}

// ResetCircuitBreaker resets the circuit breaker to closed state
func (db *DB) ResetCircuitBreaker() {
	db.circuitBreaker.Reset()
	if db.Logger != nil {
		db.Logger.Info("Database circuit breaker reset")
	}
}
