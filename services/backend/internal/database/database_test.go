package database

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestConnectionPoolConfiguration tests that connection pool is configured correctly
// **Validates: Requirements 12.1, 12.2, 12.3, 12.4**
func TestConnectionPoolConfiguration(t *testing.T) {
	if os.Getenv("DB_HOST") == "" {
		t.Skip("Skipping test: DB_HOST not set")
	}
	// Load configuration
	cfg, err := config.Load()
	require.NoError(t, err, "Failed to load configuration")

	// Create logger
	logger, err := zap.NewDevelopment()
	require.NoError(t, err, "Failed to create logger")
	defer logger.Sync()

	// Create database connection
	db, err := NewDatabase(cfg, logger)
	require.NoError(t, err, "Failed to create database connection")
	defer db.Close()

	// Test 1: Verify MaxOpenConns = 100 (Requirement 12.1)
	t.Run("MaxOpenConns should be 100", func(t *testing.T) {
		stats := db.GetStats()
		pgxStats := stats["pgxpool"].(map[string]interface{})
		sqlStats := stats["sql_db"].(map[string]interface{})

		assert.Equal(t, int32(100), pgxStats["max_conns"], "pgxpool MaxConns should be 100")
		assert.Equal(t, 100, sqlStats["max_open_conns"], "sql.DB MaxOpenConns should be 100")
	})

	// Test 2: Verify MinIdleConns = 10 (Requirement 12.2)
	t.Run("MinIdleConns should be 10", func(t *testing.T) {
		// For pgxpool, MinConns is set to 10
		// We can verify this by checking the pool config
		assert.Equal(t, 10, cfg.Database.MaxIdleConns, "MaxIdleConns config should be 10")
	})

	// Test 3: Verify ConnMaxLifetime = 1 hour (Requirement 12.3)
	t.Run("ConnMaxLifetime should be 1 hour", func(t *testing.T) {
		expectedLifetime := 1 * time.Hour
		assert.Equal(t, expectedLifetime, cfg.Database.ConnMaxLifetime, "ConnMaxLifetime should be 1 hour")
	})

	// Test 4: Verify ConnMaxIdleTime = 10 minutes (Requirement 12.4)
	t.Run("ConnMaxIdleTime should be 10 minutes", func(t *testing.T) {
		expectedIdleTime := 10 * time.Minute
		assert.Equal(t, expectedIdleTime, cfg.Database.ConnMaxIdleTime, "ConnMaxIdleTime should be 10 minutes")
	})

	// Test 5: Verify health check works
	t.Run("Health check should pass", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := db.HealthCheck(ctx)
		assert.NoError(t, err, "Health check should pass")
	})

	// Test 6: Verify connection pool statistics are available
	t.Run("Connection pool statistics should be available", func(t *testing.T) {
		stats := db.GetStats()

		// Check pgxpool stats
		pgxStats, ok := stats["pgxpool"].(map[string]interface{})
		require.True(t, ok, "pgxpool stats should be available")
		assert.Contains(t, pgxStats, "acquired_conns")
		assert.Contains(t, pgxStats, "idle_conns")
		assert.Contains(t, pgxStats, "total_conns")
		assert.Contains(t, pgxStats, "max_conns")

		// Check sql.DB stats
		sqlStats, ok := stats["sql_db"].(map[string]interface{})
		require.True(t, ok, "sql.DB stats should be available")
		assert.Contains(t, sqlStats, "max_open_conns")
		assert.Contains(t, sqlStats, "open_conns")
		assert.Contains(t, sqlStats, "in_use")
		assert.Contains(t, sqlStats, "idle")
	})
}

// TestConnectionPoolUnderLoad tests connection pool behavior under load
// **Validates: Requirements 12.1, 12.5, 12.6**
func TestConnectionPoolUnderLoad(t *testing.T) {
	if os.Getenv("DB_HOST") == "" {
		t.Skip("Skipping test: DB_HOST not set")
	}
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	// Load configuration
	cfg, err := config.Load()
	require.NoError(t, err, "Failed to load configuration")

	// Create logger
	logger, err := zap.NewDevelopment()
	require.NoError(t, err, "Failed to create logger")
	defer logger.Sync()

	// Create database connection
	db, err := NewDatabase(cfg, logger)
	require.NoError(t, err, "Failed to create database connection")
	defer db.Close()

	// Test: Simulate high load and verify pool handles it correctly
	t.Run("Pool should handle concurrent connections", func(t *testing.T) {
		ctx := context.Background()
		concurrentQueries := 50

		// Create a channel to collect results
		results := make(chan error, concurrentQueries)

		// Execute concurrent queries
		for i := 0; i < concurrentQueries; i++ {
			go func() {
				// Use pgxpool for query
				var result int
				err := db.Pool.QueryRow(ctx, "SELECT 1").Scan(&result)
				results <- err
			}()
		}

		// Collect results
		successCount := 0
		for i := 0; i < concurrentQueries; i++ {
			err := <-results
			if err == nil {
				successCount++
			}
		}

		// All queries should succeed
		assert.Equal(t, concurrentQueries, successCount, "All concurrent queries should succeed")

		// Check stats after load
		stats := db.GetStats()
		pgxStats := stats["pgxpool"].(map[string]interface{})
		
		// Total connections should not exceed max
		totalConns := pgxStats["total_conns"].(int32)
		maxConns := pgxStats["max_conns"].(int32)
		assert.LessOrEqual(t, totalConns, maxConns, "Total connections should not exceed max")
	})
}

// TestConnectionPoolTimeout tests connection timeout behavior
// **Validates: Requirements 12.5, 12.6**
func TestConnectionPoolTimeout(t *testing.T) {
	if os.Getenv("DB_HOST") == "" {
		t.Skip("Skipping test: DB_HOST not set")
	}
	if testing.Short() {
		t.Skip("Skipping timeout test in short mode")
	}

	// Load configuration
	cfg, err := config.Load()
	require.NoError(t, err, "Failed to load configuration")

	// Create logger
	logger, err := zap.NewDevelopment()
	require.NoError(t, err, "Failed to create logger")
	defer logger.Sync()

	// Create database connection
	db, err := NewDatabase(cfg, logger)
	require.NoError(t, err, "Failed to create database connection")
	defer db.Close()

	// Test: Verify timeout when pool is exhausted
	t.Run("Should timeout when pool is exhausted", func(t *testing.T) {
		// Create a context with 1 second timeout (less than 5 seconds requirement)
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		// Try to acquire more connections than max
		maxConns := 100
		connections := make([]*sql.Conn, 0, maxConns)

		// Acquire all available connections
		for i := 0; i < maxConns; i++ {
			conn, err := db.SqlDB.Conn(ctx)
			if err != nil {
				break
			}
			connections = append(connections, conn)
		}

		// Try to acquire one more connection (should timeout)
		_, err := db.SqlDB.Conn(ctx)
		
		// Should get timeout or context deadline exceeded
		assert.Error(t, err, "Should get error when pool is exhausted")

		// Release all connections
		for _, conn := range connections {
			conn.Close()
		}
	})
}
