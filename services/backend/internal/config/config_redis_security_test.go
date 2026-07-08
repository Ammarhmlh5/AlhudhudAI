package config

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

// Bug Condition Exploration Test: REDIS_PASSWORD Security Vulnerability
// **Validates: Requirements 2.2**
//
// **Property 1: Bug Condition** - Empty REDIS_PASSWORD Detection
//
// This test MUST FAIL on unfixed code - failure confirms the bug exists.
// The bug: System accepts empty REDIS_PASSWORD and connects to Redis without authentication.
//
// CRITICAL: This test encodes the EXPECTED behavior (what SHOULD happen).
// On UNFIXED code: Test FAILS (proves bug exists - system accepts empty password)
// On FIXED code: Test PASSES (proves bug is fixed - system rejects empty password)
//
// DO NOT attempt to fix the test or the code when it fails.
// Document the failure as proof of the bug.

func TestProperty_RedisSecurityBugCondition_EmptyPasswordAccepted(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	// Save original environment
	originalRedisPassword := os.Getenv("REDIS_PASSWORD")
	defer func() {
		if originalRedisPassword != "" {
			os.Setenv("REDIS_PASSWORD", originalRedisPassword)
		} else {
			os.Unsetenv("REDIS_PASSWORD")
		}
	}()

	rapid.Check(t, func(t *rapid.T) {
		// Generate empty or weak REDIS_PASSWORD values that should be REJECTED
		weakPasswordType := rapid.IntRange(0, 2).Draw(t, "weakPasswordType")
		
		var weakPassword string
		var description string
		
		switch weakPasswordType {
		case 0:
			// Empty password (exact match from .env)
			weakPassword = ""
			description = "empty password"
			
		case 1:
			// Whitespace only
			spaceCount := rapid.IntRange(1, 10).Draw(t, "spaceCount")
			weakPassword = strings.Repeat(" ", spaceCount)
			description = "whitespace only"
			
		case 2:
			// Very short password (< 8 characters)
			length := rapid.IntRange(1, 7).Draw(t, "shortLength")
			weakPassword = rapid.StringN(length, length, -1).Draw(t, "shortPassword")
			description = "very short password (< 8 chars)"
		}

		// Set the weak REDIS_PASSWORD
		os.Setenv("REDIS_PASSWORD", weakPassword)
		
		// Also set other required vars to isolate REDIS_PASSWORD testing
		os.Setenv("DB_HOST", "localhost")
		os.Setenv("DB_USER", "test_user")
		os.Setenv("DB_PASSWORD", "test_pass")
		os.Setenv("DB_NAME", "test_db")
		os.Setenv("REDIS_HOST", "localhost")
		os.Setenv("JWT_SECRET", "secure-jwt-secret-with-at-least-32-characters-long")

		// ACT: Try to load configuration
		cfg, err := Load()

		// ASSERT: Expected behavior - should REJECT empty/weak passwords
		// On UNFIXED code: This assertion WILL FAIL because Load() succeeds with empty password
		// On FIXED code: This assertion WILL PASS because Load() rejects empty password
		assert.Error(t, err,
			"Bug Condition: System accepts weak REDIS_PASSWORD (%s: '%s', length: %d). "+
				"Expected: Load() should reject empty/weak passwords with validation error. "+
				"Got: Load() succeeded. "+
				"This failure confirms the security vulnerability exists.",
			description, weakPassword, len(weakPassword))

		// If Load() succeeded (bug exists), verify it's using the weak password
		if err == nil && cfg != nil {
			assert.Equal(t, weakPassword, cfg.Redis.Password,
				"Bug confirmed: System is using the weak REDIS_PASSWORD value")
			
			t.Logf("SECURITY VULNERABILITY DETECTED:")
			t.Logf("  Type: %s", description)
			t.Logf("  Value: '%s'", weakPassword)
			t.Logf("  Length: %d characters", len(weakPassword))
			t.Logf("  Expected: Validation error rejecting weak password")
			t.Logf("  Actual: Configuration loaded successfully with weak password")
			t.Logf("  Impact: Redis is accessible without authentication on localhost")
		}
	})
}

// Example-based test for Bug Condition - Empty REDIS_PASSWORD
// **Validates: Requirements 2.2**
//
// This test provides a concrete example of the most critical bug condition:
// The system accepts empty REDIS_PASSWORD from .env file.
// EXPECTED OUTCOME on UNFIXED code: FAILS (proves bug exists)
func TestRedisSecurity_EmptyPasswordAccepted(t *testing.T) {
	// Save original environment
	originalRedisPassword := os.Getenv("REDIS_PASSWORD")
	defer func() {
		if originalRedisPassword != "" {
			os.Setenv("REDIS_PASSWORD", originalRedisPassword)
		} else {
			os.Unsetenv("REDIS_PASSWORD")
		}
	}()

	// Set empty REDIS_PASSWORD (exact value from .env)
	os.Setenv("REDIS_PASSWORD", "")
	
	// Set other required vars
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_USER", "test_user")
	os.Setenv("DB_PASSWORD", "test_pass")
	os.Setenv("DB_NAME", "test_db")
	os.Setenv("REDIS_HOST", "localhost")
	os.Setenv("JWT_SECRET", "secure-jwt-secret-with-at-least-32-characters-long")

	// ACT: Try to load configuration
	cfg, err := Load()

	// ASSERT: Expected behavior - should REJECT empty password
	// On UNFIXED code: This WILL FAIL (Load() succeeds)
	// On FIXED code: This WILL PASS (Load() returns error)
	assert.Error(t, err,
		"CRITICAL SECURITY BUG: System accepts empty REDIS_PASSWORD. "+
			"This means Redis is running without authentication. "+
			"Expected: Load() should reject with validation error. "+
			"This confirms the security vulnerability exists.")

	// If Load() succeeded (bug exists), log the vulnerability
	if err == nil && cfg != nil {
		t.Errorf("CRITICAL SECURITY VULNERABILITY CONFIRMED:")
		t.Logf("  REDIS_PASSWORD is empty: '%s'", cfg.Redis.Password)
		t.Logf("  Redis is accessible without authentication on %s", cfg.GetRedisAddr())
		t.Logf("  Anyone with network access can read/write Redis data")
		t.Logf("  Expected: System should reject startup with validation error")
		t.Logf("  Actual: System started successfully with no Redis authentication")
	}
}

// Example-based test for Bug Condition - Whitespace-only REDIS_PASSWORD
// **Validates: Requirements 2.2**
//
// This test verifies that the system accepts REDIS_PASSWORD with only whitespace.
// EXPECTED OUTCOME on UNFIXED code: FAILS (proves bug exists)
func TestRedisSecurity_WhitespacePasswordAccepted(t *testing.T) {
	// Save original environment
	originalRedisPassword := os.Getenv("REDIS_PASSWORD")
	defer func() {
		if originalRedisPassword != "" {
			os.Setenv("REDIS_PASSWORD", originalRedisPassword)
		} else {
			os.Unsetenv("REDIS_PASSWORD")
		}
	}()

	testCases := []struct {
		password    string
		description string
	}{
		{" ", "single space"},
		{"   ", "three spaces"},
		{"\t", "tab character"},
		{"\n", "newline character"},
		{" \t\n ", "mixed whitespace"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			os.Setenv("REDIS_PASSWORD", tc.password)
			
			// Set other required vars
			os.Setenv("DB_HOST", "localhost")
			os.Setenv("DB_USER", "test_user")
			os.Setenv("DB_PASSWORD", "test_pass")
			os.Setenv("DB_NAME", "test_db")
			os.Setenv("REDIS_HOST", "localhost")
			os.Setenv("JWT_SECRET", "secure-jwt-secret-with-at-least-32-characters-long")

			// ACT
			cfg, err := Load()

			// ASSERT: Should reject whitespace-only passwords
			// On UNFIXED code: This WILL FAIL (Load() succeeds)
			// On FIXED code: This WILL PASS (Load() returns error)
			assert.Error(t, err,
				"Bug Condition: System accepts whitespace-only REDIS_PASSWORD (%s). "+
					"Expected: Load() should reject with validation error. "+
					"This confirms the security vulnerability exists.",
				tc.description)

			// If Load() succeeded (bug exists), log the vulnerability
			if err == nil && cfg != nil {
				t.Errorf("SECURITY VULNERABILITY: Whitespace-only REDIS_PASSWORD accepted")
				t.Logf("  Password: '%s' (%s)", tc.password, tc.description)
				t.Logf("  Length: %d characters", len(tc.password))
				t.Logf("  Expected: Validation error")
				t.Logf("  Actual: Configuration loaded successfully")
			}
		})
	}
}

// Example-based test for Bug Condition - Very short REDIS_PASSWORD
// **Validates: Requirements 2.2**
//
// This test verifies that the system accepts REDIS_PASSWORD shorter than 8 characters.
// EXPECTED OUTCOME on UNFIXED code: FAILS (proves bug exists)
func TestRedisSecurity_ShortPasswordAccepted(t *testing.T) {
	// Save original environment
	originalRedisPassword := os.Getenv("REDIS_PASSWORD")
	defer func() {
		if originalRedisPassword != "" {
			os.Setenv("REDIS_PASSWORD", originalRedisPassword)
		} else {
			os.Unsetenv("REDIS_PASSWORD")
		}
	}()

	testCases := []struct {
		password    string
		description string
	}{
		{"a", "1 character"},
		{"ab", "2 characters"},
		{"abc", "3 characters"},
		{"1234", "4 characters"},
		{"12345", "5 characters"},
		{"123456", "6 characters"},
		{"1234567", "7 characters"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			os.Setenv("REDIS_PASSWORD", tc.password)
			
			// Set other required vars
			os.Setenv("DB_HOST", "localhost")
			os.Setenv("DB_USER", "test_user")
			os.Setenv("DB_PASSWORD", "test_pass")
			os.Setenv("DB_NAME", "test_db")
			os.Setenv("REDIS_HOST", "localhost")
			os.Setenv("JWT_SECRET", "secure-jwt-secret-with-at-least-32-characters-long")

			// ACT
			cfg, err := Load()

			// ASSERT: Should reject very short passwords
			// On UNFIXED code: This WILL FAIL (Load() succeeds)
			// On FIXED code: This WILL PASS (Load() returns error)
			assert.Error(t, err,
				"Bug Condition: System accepts very short REDIS_PASSWORD (%s, length: %d). "+
					"Minimum secure length should be at least 8 characters. "+
					"Expected: Load() should reject with validation error. "+
					"This confirms the security vulnerability exists.",
				tc.description, len(tc.password))

			// If Load() succeeded (bug exists), log the vulnerability
			if err == nil && cfg != nil {
				t.Errorf("SECURITY VULNERABILITY: Very short REDIS_PASSWORD accepted")
				t.Logf("  Password: %s", tc.password)
				t.Logf("  Length: %d characters (minimum should be 8+)", len(tc.password))
				t.Logf("  Expected: Validation error")
				t.Logf("  Actual: Configuration loaded successfully")
			}
		})
	}
}

// Edge case test: Valid strong REDIS_PASSWORD should be accepted
// **Validates: Requirements 2.2**
//
// This test verifies that a properly generated strong REDIS_PASSWORD is accepted.
// This should PASS on both UNFIXED and FIXED code (when we provide a valid password).
func TestRedisSecurity_StrongPasswordAccepted(t *testing.T) {
	// Save original environment
	originalRedisPassword := os.Getenv("REDIS_PASSWORD")
	defer func() {
		if originalRedisPassword != "" {
			os.Setenv("REDIS_PASSWORD", originalRedisPassword)
		} else {
			os.Unsetenv("REDIS_PASSWORD")
		}
	}()

	// Generate a strong REDIS_PASSWORD (simulating: openssl rand -base64 32)
	// This is a 44-character base64 string (32 bytes encoded)
	strongPassword := "7K9mP2xQ5vR8wT1yU4zN6bC3dF0gH7jK9mP2xQ5vR8w="
	os.Setenv("REDIS_PASSWORD", strongPassword)
	
	// Set other required vars
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_USER", "test_user")
	os.Setenv("DB_PASSWORD", "test_pass")
	os.Setenv("DB_NAME", "test_db")
	os.Setenv("REDIS_HOST", "localhost")
	os.Setenv("JWT_SECRET", "secure-jwt-secret-with-at-least-32-characters-long")

	// ACT
	cfg, err := Load()

	// ASSERT: Strong password should be accepted
	// This should PASS on both UNFIXED and FIXED code
	assert.NoError(t, err,
		"Strong REDIS_PASSWORD should be accepted. "+
			"Password length: %d characters (>= 8 required). "+
			"Error: %v",
		len(strongPassword), err)

	if cfg != nil {
		assert.Equal(t, strongPassword, cfg.Redis.Password,
			"Configuration should use the provided strong password")
		
		t.Logf("SUCCESS: Strong REDIS_PASSWORD accepted")
		t.Logf("  Length: %d characters", len(strongPassword))
		t.Logf("  This is the expected behavior for secure passwords")
	}
}

// Edge case test: REDIS_PASSWORD at minimum length boundary (8 chars)
// **Validates: Requirements 2.2**
//
// This test verifies behavior at the 8-character boundary.
func TestRedisSecurity_MinimumLengthBoundary(t *testing.T) {
	// Save original environment
	originalRedisPassword := os.Getenv("REDIS_PASSWORD")
	defer func() {
		if originalRedisPassword != "" {
			os.Setenv("REDIS_PASSWORD", originalRedisPassword)
		} else {
			os.Unsetenv("REDIS_PASSWORD")
		}
	}()

	testCases := []struct {
		password    string
		shouldPass  bool
		description string
	}{
		{"7chars!", false, "7 characters (just below minimum)"},
		{"8chars!!", true, "8 characters (at minimum boundary)"},
		{"9chars!!!", true, "9 characters (above minimum)"},
		{"secure-redis-password-32-ch", true, "28 characters (well above minimum)"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			os.Setenv("REDIS_PASSWORD", tc.password)
			
			// Set other required vars
			os.Setenv("DB_HOST", "localhost")
			os.Setenv("DB_USER", "test_user")
			os.Setenv("DB_PASSWORD", "test_pass")
			os.Setenv("DB_NAME", "test_db")
			os.Setenv("REDIS_HOST", "localhost")
			os.Setenv("JWT_SECRET", "secure-jwt-secret-with-at-least-32-characters-long")

			// ACT
			cfg, err := Load()

			// ASSERT
			if tc.shouldPass {
				// On FIXED code: Should accept passwords >= 8 chars
				assert.NoError(t, err,
					"Password with %d characters should be accepted (minimum is 8)",
					len(tc.password))
			} else {
				// On UNFIXED code: This WILL FAIL (accepts < 8 chars)
				// On FIXED code: This WILL PASS (rejects < 8 chars)
				assert.Error(t, err,
					"Bug Condition: Password with %d characters should be rejected (minimum is 8). "+
						"Expected: Validation error. "+
						"This confirms the bug exists.",
					len(tc.password))
			}

			// Log the result
			if err == nil && cfg != nil {
				t.Logf("Configuration loaded with password length: %d", len(tc.password))
			} else if err != nil {
				t.Logf("Configuration rejected with error: %v", err)
			}
		})
	}
}

// Edge case test: Production environment should enforce REDIS_PASSWORD
// **Validates: Requirements 2.2**
//
// This test verifies that production environment specifically rejects empty REDIS_PASSWORD.
// EXPECTED OUTCOME on UNFIXED code: FAILS (proves bug exists)
func TestRedisSecurity_ProductionEnvironmentEmptyPassword(t *testing.T) {
	// Save original environment
	originalRedisPassword := os.Getenv("REDIS_PASSWORD")
	originalAppEnv := os.Getenv("APP_ENV")
	defer func() {
		if originalRedisPassword != "" {
			os.Setenv("REDIS_PASSWORD", originalRedisPassword)
		} else {
			os.Unsetenv("REDIS_PASSWORD")
		}
		if originalAppEnv != "" {
			os.Setenv("APP_ENV", originalAppEnv)
		} else {
			os.Unsetenv("APP_ENV")
		}
	}()

	// Set production environment
	os.Setenv("APP_ENV", "production")
	os.Setenv("REDIS_PASSWORD", "")
	
	// Set other required vars
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_USER", "test_user")
	os.Setenv("DB_PASSWORD", "test_pass")
	os.Setenv("DB_NAME", "test_db")
	os.Setenv("REDIS_HOST", "localhost")
	os.Setenv("JWT_SECRET", "secure-jwt-secret-with-at-least-32-characters-long")

	// ACT
	cfg, err := Load()

	// ASSERT: Production environment should STRICTLY reject empty password
	// On UNFIXED code: This WILL FAIL (Load() succeeds)
	// On FIXED code: This WILL PASS (Load() returns error)
	assert.Error(t, err,
		"CRITICAL: Production environment accepts empty REDIS_PASSWORD. "+
			"Production systems MUST have Redis authentication enabled. "+
			"Expected: Load() should reject with validation error. "+
			"This confirms the security vulnerability exists.")

	// If Load() succeeded (bug exists), log the critical vulnerability
	if err == nil && cfg != nil {
		t.Errorf("CRITICAL PRODUCTION SECURITY VULNERABILITY:")
		t.Logf("  Environment: %s", cfg.Server.AppEnv)
		t.Logf("  REDIS_PASSWORD is empty in PRODUCTION")
		t.Logf("  This is a critical security risk in production environment")
		t.Logf("  Expected: System should refuse to start")
		t.Logf("  Actual: System started successfully")
	}
}
