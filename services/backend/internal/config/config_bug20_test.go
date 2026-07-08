package config

import (
	"os"
	"testing"
)

// Bug 20 Exploration Test: App Runs Without Required Vars
// **Validates: Requirements 1.20, 2.20**
//
// Property 1: Bug Condition - App Runs Without Required Vars
// This test verifies that the application currently starts without validating
// required environment variables, which is the bug we're fixing.
//
// EXPECTED OUTCOME ON UNFIXED CODE: Test FAILS (app starts without validation)
// EXPECTED OUTCOME ON FIXED CODE: Test PASSES (validation prevents startup)

func TestBug20_Exploration_AppStartsWithoutRequiredVars(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	requiredVars := []string{
		"DB_HOST",
		"DB_USER",
		"DB_PASSWORD",
		"DB_NAME",
		"REDIS_HOST",
		"JWT_SECRET",
	}

	for _, key := range requiredVars {
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

	// Test Case 1: Missing DB_HOST
	t.Run("Missing_DB_HOST", func(t *testing.T) {
		os.Unsetenv("DB_HOST")
		
		cfg, err := Load()
		
		// BUG CONDITION: On unfixed code, Load() succeeds with default value
		// EXPECTED BEHAVIOR: On fixed code, Load() should return error
		if err == nil {
			t.Errorf("UNFIXED CODE DETECTED: Load() succeeded without DB_HOST (got default: %s)", cfg.Database.Host)
			t.Log("Expected: Load() should fail with validation error")
			t.Log("This test will PASS once the fix is implemented")
		} else {
			t.Logf("FIXED CODE: Load() correctly failed with error: %v", err)
		}
	})

	// Test Case 2: Missing JWT_SECRET
	t.Run("Missing_JWT_SECRET", func(t *testing.T) {
		// Restore DB_HOST for this test
		os.Setenv("DB_HOST", "localhost")
		os.Unsetenv("JWT_SECRET")
		
		cfg, err := Load()
		
		// BUG CONDITION: On unfixed code, Load() succeeds with default value
		// EXPECTED BEHAVIOR: On fixed code, Load() should return error
		if err == nil {
			t.Errorf("UNFIXED CODE DETECTED: Load() succeeded without JWT_SECRET (got default: %s)", cfg.JWT.Secret)
			t.Log("Expected: Load() should fail with validation error")
			t.Log("This test will PASS once the fix is implemented")
		} else {
			t.Logf("FIXED CODE: Load() correctly failed with error: %v", err)
		}
	})

	// Test Case 3: Missing REDIS_HOST
	t.Run("Missing_REDIS_HOST", func(t *testing.T) {
		// Restore previous vars
		os.Setenv("DB_HOST", "localhost")
		os.Setenv("JWT_SECRET", "test-secret-key-minimum-32-chars-long")
		os.Unsetenv("REDIS_HOST")
		
		cfg, err := Load()
		
		// BUG CONDITION: On unfixed code, Load() succeeds with default value
		// EXPECTED BEHAVIOR: On fixed code, Load() should return error
		if err == nil {
			t.Errorf("UNFIXED CODE DETECTED: Load() succeeded without REDIS_HOST (got default: %s)", cfg.Redis.Host)
			t.Log("Expected: Load() should fail with validation error")
			t.Log("This test will PASS once the fix is implemented")
		} else {
			t.Logf("FIXED CODE: Load() correctly failed with error: %v", err)
		}
	})

	// Test Case 4: Missing multiple required variables
	t.Run("Missing_Multiple_Required_Vars", func(t *testing.T) {
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("REDIS_HOST")
		os.Unsetenv("JWT_SECRET")
		
		cfg, err := Load()
		
		// BUG CONDITION: On unfixed code, Load() succeeds with all defaults
		// EXPECTED BEHAVIOR: On fixed code, Load() should return error listing all missing vars
		if err == nil {
			t.Errorf("UNFIXED CODE DETECTED: Load() succeeded without multiple required vars")
			t.Logf("  DB_HOST: %s (default)", cfg.Database.Host)
			t.Logf("  DB_USER: %s (default)", cfg.Database.User)
			t.Logf("  REDIS_HOST: %s (default)", cfg.Redis.Host)
			t.Logf("  JWT_SECRET: %s (default)", cfg.JWT.Secret)
			t.Log("Expected: Load() should fail with validation error listing all missing vars")
			t.Log("This test will PASS once the fix is implemented")
		} else {
			t.Logf("FIXED CODE: Load() correctly failed with error: %v", err)
		}
	})

	// Test Case 5: JWT_SECRET too short
	t.Run("JWT_SECRET_Too_Short", func(t *testing.T) {
		// Restore all required vars
		os.Setenv("DB_HOST", "localhost")
		os.Setenv("DB_USER", "test_user")
		os.Setenv("DB_PASSWORD", "test_pass")
		os.Setenv("DB_NAME", "test_db")
		os.Setenv("REDIS_HOST", "localhost")
		os.Setenv("JWT_SECRET", "short")  // Only 5 characters
		
		cfg, err := Load()
		
		// BUG CONDITION: On unfixed code, Load() succeeds with short secret
		// EXPECTED BEHAVIOR: On fixed code, Load() should return error about secret length
		if err == nil {
			t.Errorf("UNFIXED CODE DETECTED: Load() succeeded with JWT_SECRET too short (length: %d)", len(cfg.JWT.Secret))
			t.Log("Expected: Load() should fail with validation error about minimum length")
			t.Log("This test will PASS once the fix is implemented")
		} else {
			t.Logf("FIXED CODE: Load() correctly failed with error: %v", err)
		}
	})
}
