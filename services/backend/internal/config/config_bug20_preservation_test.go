package config

import (
	"os"
	"testing"
	"time"
)

// Bug 20 Preservation Test: Normal Startup Works
// **Validates: Requirements 3.13**
//
// Property 2: Preservation - Normal Startup Works
// This test verifies that when all required environment variables are properly set,
// the configuration loads successfully and the application can start normally.
//
// EXPECTED OUTCOME: Test PASSES on both UNFIXED and FIXED code

func TestBug20_Preservation_NormalStartupWorks(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	allVars := []string{
		"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSLMODE",
		"REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD", "REDIS_DB",
		"JWT_SECRET", "JWT_EXPIRY_HOURS",
		"PORT", "APP_ENV",
	}

	for _, key := range allVars {
		originalEnv[key] = os.Getenv(key)
	}

	// Restore environment after test
	defer func() {
		for key, value := range originalEnv {
			if value != "" {
				os.Setenv(key, value)
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	// Test Case 1: Valid configuration with all required variables
	t.Run("Valid_Configuration_All_Required", func(t *testing.T) {
		// Set all required environment variables
		os.Setenv("DB_HOST", "localhost")
		os.Setenv("DB_PORT", "5432")
		os.Setenv("DB_USER", "test_user")
		os.Setenv("DB_PASSWORD", "test_password")
		os.Setenv("DB_NAME", "test_database")
		os.Setenv("DB_SSLMODE", "disable")
		os.Setenv("REDIS_HOST", "localhost")
		os.Setenv("REDIS_PORT", "6379")
		os.Setenv("JWT_SECRET", "this-is-a-secure-jwt-secret-key-with-minimum-32-characters")
		os.Setenv("PORT", "8080")
		os.Setenv("APP_ENV", "test")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("PRESERVATION FAILED: Load() failed with valid config: %v", err)
		}

		// Verify configuration values are loaded correctly
		if cfg.Database.Host != "localhost" {
			t.Errorf("Expected DB_HOST=localhost, got %s", cfg.Database.Host)
		}
		if cfg.Database.Port != "5432" {
			t.Errorf("Expected DB_PORT=5432, got %s", cfg.Database.Port)
		}
		if cfg.Database.User != "test_user" {
			t.Errorf("Expected DB_USER=test_user, got %s", cfg.Database.User)
		}
		if cfg.Redis.Host != "localhost" {
			t.Errorf("Expected REDIS_HOST=localhost, got %s", cfg.Redis.Host)
		}
		if cfg.JWT.Secret != "this-is-a-secure-jwt-secret-key-with-minimum-32-characters" {
			t.Errorf("Expected JWT_SECRET to match, got %s", cfg.JWT.Secret)
		}

		t.Log("✓ Configuration loaded successfully with all required variables")
	})

	// Test Case 2: Valid configuration with optional variables
	t.Run("Valid_Configuration_With_Optionals", func(t *testing.T) {
		// Set required variables
		os.Setenv("DB_HOST", "db.example.com")
		os.Setenv("DB_USER", "prod_user")
		os.Setenv("DB_PASSWORD", "secure_password_123")
		os.Setenv("DB_NAME", "production_db")
		os.Setenv("REDIS_HOST", "redis.example.com")
		os.Setenv("JWT_SECRET", "production-jwt-secret-key-very-secure-and-long-enough")

		// Set optional variables
		os.Setenv("DB_PORT", "5433")
		os.Setenv("REDIS_PORT", "6380")
		os.Setenv("REDIS_PASSWORD", "redis_password")
		os.Setenv("JWT_EXPIRY_HOURS", "48")
		os.Setenv("BATCH_SIZE", "200")
		os.Setenv("WORKER_COUNT", "100")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("PRESERVATION FAILED: Load() failed with valid config: %v", err)
		}

		// Verify optional values are loaded
		if cfg.Database.Port != "5433" {
			t.Errorf("Expected DB_PORT=5433, got %s", cfg.Database.Port)
		}
		if cfg.Redis.Port != "6380" {
			t.Errorf("Expected REDIS_PORT=6380, got %s", cfg.Redis.Port)
		}
		if cfg.JWT.ExpiryHours != 48 {
			t.Errorf("Expected JWT_EXPIRY_HOURS=48, got %d", cfg.JWT.ExpiryHours)
		}
		if cfg.BatchProcess.BatchSize != 200 {
			t.Errorf("Expected BATCH_SIZE=200, got %d", cfg.BatchProcess.BatchSize)
		}

		t.Log("✓ Configuration loaded successfully with optional variables")
	})

	// Test Case 3: Default values work correctly
	t.Run("Default_Values_Applied", func(t *testing.T) {
		// Set only required variables, let others use defaults
		os.Setenv("DB_HOST", "localhost")
		os.Setenv("DB_USER", "user")
		os.Setenv("DB_PASSWORD", "pass")
		os.Setenv("DB_NAME", "db")
		os.Setenv("REDIS_HOST", "localhost")
		os.Setenv("JWT_SECRET", "minimum-32-character-secret-key-for-jwt-tokens")

		// Unset optional variables to test defaults
		os.Unsetenv("DB_PORT")
		os.Unsetenv("REDIS_PORT")
		os.Unsetenv("JWT_EXPIRY_HOURS")
		os.Unsetenv("BATCH_SIZE")
		os.Unsetenv("WORKER_COUNT")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("PRESERVATION FAILED: Load() failed with defaults: %v", err)
		}

		// Verify default values
		if cfg.Database.Port != "5432" {
			t.Errorf("Expected default DB_PORT=5432, got %s", cfg.Database.Port)
		}
		if cfg.Redis.Port != "6379" {
			t.Errorf("Expected default REDIS_PORT=6379, got %s", cfg.Redis.Port)
		}
		if cfg.JWT.ExpiryHours != 24 {
			t.Errorf("Expected default JWT_EXPIRY_HOURS=24, got %d", cfg.JWT.ExpiryHours)
		}
		if cfg.BatchProcess.BatchSize != 100 {
			t.Errorf("Expected default BATCH_SIZE=100, got %d", cfg.BatchProcess.BatchSize)
		}
		if cfg.WorkerPool.WorkerCount != 50 {
			t.Errorf("Expected default WORKER_COUNT=50, got %d", cfg.WorkerPool.WorkerCount)
		}

		t.Log("✓ Default values applied correctly for optional variables")
	})

	// Test Case 4: Database DSN generation works
	t.Run("Database_DSN_Generation", func(t *testing.T) {
		os.Setenv("DB_HOST", "db.example.com")
		os.Setenv("DB_PORT", "5432")
		os.Setenv("DB_USER", "myuser")
		os.Setenv("DB_PASSWORD", "mypassword")
		os.Setenv("DB_NAME", "mydb")
		os.Setenv("DB_SSLMODE", "require")
		os.Setenv("REDIS_HOST", "localhost")
		os.Setenv("JWT_SECRET", "secure-jwt-secret-key-with-32-plus-characters")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("PRESERVATION FAILED: Load() failed: %v", err)
		}

		dsn := cfg.GetDatabaseDSN()
		expectedDSN := "host=db.example.com port=5432 user=myuser password=mypassword dbname=mydb sslmode=require"
		if dsn != expectedDSN {
			t.Errorf("DSN mismatch:\nExpected: %s\nGot:      %s", expectedDSN, dsn)
		}

		t.Log("✓ Database DSN generated correctly")
	})

	// Test Case 5: Redis address generation works
	t.Run("Redis_Address_Generation", func(t *testing.T) {
		os.Setenv("DB_HOST", "localhost")
		os.Setenv("DB_USER", "user")
		os.Setenv("DB_PASSWORD", "pass")
		os.Setenv("DB_NAME", "db")
		os.Setenv("REDIS_HOST", "redis.example.com")
		os.Setenv("REDIS_PORT", "6380")
		os.Setenv("JWT_SECRET", "jwt-secret-key-must-be-at-least-32-characters-long")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("PRESERVATION FAILED: Load() failed: %v", err)
		}

		addr := cfg.GetRedisAddr()
		expectedAddr := "redis.example.com:6380"
		if addr != expectedAddr {
			t.Errorf("Redis address mismatch: expected %s, got %s", expectedAddr, addr)
		}

		t.Log("✓ Redis address generated correctly")
	})

	// Test Case 6: Complex configuration with all features
	t.Run("Complex_Configuration", func(t *testing.T) {
		// Set comprehensive configuration
		os.Setenv("DB_HOST", "prod-db.example.com")
		os.Setenv("DB_PORT", "5432")
		os.Setenv("DB_USER", "prod_admin")
		os.Setenv("DB_PASSWORD", "very_secure_password_123")
		os.Setenv("DB_NAME", "production")
		os.Setenv("DB_SSLMODE", "verify-full")
		os.Setenv("DB_MAX_OPEN_CONNS", "200")
		os.Setenv("DB_MAX_IDLE_CONNS", "20")
		os.Setenv("DB_CONN_MAX_LIFETIME", "7200")
		os.Setenv("DB_CONN_MAX_IDLE_TIME", "1200")

		os.Setenv("REDIS_HOST", "prod-redis.example.com")
		os.Setenv("REDIS_PORT", "6379")
		os.Setenv("REDIS_PASSWORD", "redis_secure_pass")
		os.Setenv("REDIS_DB", "1")

		os.Setenv("JWT_SECRET", "production-grade-jwt-secret-key-with-high-entropy-and-length")
		os.Setenv("JWT_EXPIRY_HOURS", "168") // 7 days

		os.Setenv("PORT", "8443")
		os.Setenv("APP_ENV", "production")

		os.Setenv("BATCH_SIZE", "50")
		os.Setenv("WORKER_COUNT", "100")
		os.Setenv("RATE_LIMIT_REQUESTS_PER_MINUTE", "1000")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("PRESERVATION FAILED: Load() failed with complex config: %v", err)
		}

		// Verify all values
		if cfg.Database.Host != "prod-db.example.com" {
			t.Errorf("DB_HOST mismatch")
		}
		if cfg.Database.MaxOpenConns != 200 {
			t.Errorf("Expected MaxOpenConns=200, got %d", cfg.Database.MaxOpenConns)
		}
		if cfg.Database.ConnMaxLifetime != 7200*time.Second {
			t.Errorf("ConnMaxLifetime mismatch")
		}
		if cfg.Redis.Password != "redis_secure_pass" {
			t.Errorf("Redis password mismatch")
		}
		if cfg.JWT.ExpiryHours != 168 {
			t.Errorf("JWT expiry mismatch")
		}
		if cfg.Server.Port != "8443" {
			t.Errorf("Server port mismatch")
		}
		if cfg.BatchProcess.BatchSize != 50 {
			t.Errorf("Batch size mismatch")
		}
		if cfg.WorkerPool.WorkerCount != 100 {
			t.Errorf("Worker count mismatch")
		}

		t.Log("✓ Complex configuration loaded successfully")
	})
}
