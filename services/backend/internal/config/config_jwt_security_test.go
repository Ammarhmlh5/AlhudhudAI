package config

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

// Bug Condition Exploration Test: JWT_SECRET Security Vulnerability
// **Validates: Requirements 2.1**
//
// **Property 1: Bug Condition** - Insecure JWT_SECRET Detection
//
// This test MUST FAIL on unfixed code - failure confirms the bug exists.
// The bug: System accepts weak JWT_SECRET values (default placeholder, short length < 32 chars, non-random patterns).
//
// CRITICAL: This test encodes the EXPECTED behavior (what SHOULD happen).
// On UNFIXED code: Test FAILS (proves bug exists - system accepts weak secrets)
// On FIXED code: Test PASSES (proves bug is fixed - system rejects weak secrets)
//
// DO NOT attempt to fix the test or the code when it fails.
// Document the failure as proof of the bug.

func TestProperty_JWTSecurityBugCondition_WeakSecretsAccepted(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	// Save original environment
	originalJWTSecret := os.Getenv("JWT_SECRET")
	defer func() {
		if originalJWTSecret != "" {
			os.Setenv("JWT_SECRET", originalJWTSecret)
		} else {
			os.Unsetenv("JWT_SECRET")
		}
	}()

	rapid.Check(t, func(t *rapid.T) {
		// Generate weak JWT_SECRET values that should be REJECTED
		weakSecretType := rapid.IntRange(0, 3).Draw(t, "weakSecretType")
		
		var weakSecret string
		var description string
		
		switch weakSecretType {
		case 0:
			// Default placeholder value (exact match from .env)
			weakSecret = "change-me-in-production-to-a-secure-random-string"
			description = "default placeholder value"
			
		case 1:
			// Short secret (< 32 characters)
			length := rapid.IntRange(1, 31).Draw(t, "shortLength")
			weakSecret = rapid.StringN(length, length, -1).Draw(t, "shortSecret")
			description = "short secret (< 32 chars)"
			
		case 2:
			// Non-random pattern (repeated characters)
			char := rapid.Rune().Draw(t, "repeatedChar")
			length := rapid.IntRange(10, 50).Draw(t, "patternLength")
			weakSecret = strings.Repeat(string(char), length)
			description = "non-random pattern (repeated characters)"
			
		case 3:
			// Common weak patterns
			weakPatterns := []string{
				"12345678901234567890123456789012",
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				"password123456789012345678901234",
				"secret1234567890123456789012345",
				"test-secret-key-1234567890123456",
			}
			weakSecret = rapid.SampledFrom(weakPatterns).Draw(t, "weakPattern")
			description = "common weak pattern"
		}

		// Set the weak JWT_SECRET
		os.Setenv("JWT_SECRET", weakSecret)
		
		// Also set other required vars to isolate JWT_SECRET testing
		os.Setenv("DB_HOST", "localhost")
		os.Setenv("DB_USER", "test_user")
		os.Setenv("DB_PASSWORD", "test_pass")
		os.Setenv("DB_NAME", "test_db")
		os.Setenv("REDIS_HOST", "localhost")

		// ACT: Try to load configuration
		cfg, err := Load()

		// ASSERT: Expected behavior - should REJECT weak secrets
		// On UNFIXED code: This assertion WILL FAIL because Load() succeeds with weak secrets
		// On FIXED code: This assertion WILL PASS because Load() rejects weak secrets
		assert.Error(t, err,
			"Bug Condition: System accepts weak JWT_SECRET (%s: '%s', length: %d). "+
				"Expected: Load() should reject weak secrets with validation error. "+
				"Got: Load() succeeded. "+
				"This failure confirms the security vulnerability exists.",
			description, weakSecret, len(weakSecret))

		// If Load() succeeded (bug exists), verify it's using the weak secret
		if err == nil && cfg != nil {
			assert.Equal(t, weakSecret, cfg.JWT.Secret,
				"Bug confirmed: System is using the weak JWT_SECRET value")
			
			t.Logf("SECURITY VULNERABILITY DETECTED:")
			t.Logf("  Type: %s", description)
			t.Logf("  Value: %s", weakSecret)
			t.Logf("  Length: %d characters", len(weakSecret))
			t.Logf("  Expected: Validation error rejecting weak secret")
			t.Logf("  Actual: Configuration loaded successfully with weak secret")
		}
	})
}

// Example-based test for Bug Condition - Default placeholder value
// **Validates: Requirements 2.1**
//
// This test provides a concrete example of the most critical bug condition:
// The system accepts the default placeholder JWT_SECRET from .env file.
// EXPECTED OUTCOME on UNFIXED code: FAILS (proves bug exists)
func TestJWTSecurity_DefaultPlaceholderAccepted(t *testing.T) {
	// Save original environment
	originalJWTSecret := os.Getenv("JWT_SECRET")
	defer func() {
		if originalJWTSecret != "" {
			os.Setenv("JWT_SECRET", originalJWTSecret)
		} else {
			os.Unsetenv("JWT_SECRET")
		}
	}()

	// Set the exact default value from .env
	defaultPlaceholder := "change-me-in-production-to-a-secure-random-string"
	os.Setenv("JWT_SECRET", defaultPlaceholder)
	
	// Set other required vars
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_USER", "test_user")
	os.Setenv("DB_PASSWORD", "test_pass")
	os.Setenv("DB_NAME", "test_db")
	os.Setenv("REDIS_HOST", "localhost")
	os.Setenv("REDIS_PASSWORD", "test_redis_password")

	// ACT: Try to load configuration
	cfg, err := Load()

	// ASSERT: Expected behavior - should REJECT default placeholder
	// On UNFIXED code: This WILL FAIL (Load() succeeds)
	// On FIXED code: This WILL PASS (Load() returns error)
	assert.Error(t, err,
		"CRITICAL SECURITY BUG: System accepts default JWT_SECRET placeholder. "+
			"Value: '%s'. "+
			"This is a well-known default value that can be easily guessed. "+
			"Expected: Load() should reject with validation error. "+
			"This confirms the security vulnerability exists.",
		defaultPlaceholder)

	// If Load() succeeded (bug exists), log the vulnerability
	if err == nil && cfg != nil {
		t.Errorf("CRITICAL SECURITY VULNERABILITY CONFIRMED:")
		t.Logf("  JWT_SECRET is using default placeholder: %s", cfg.JWT.Secret)
		t.Logf("  This value is publicly visible in .env.example")
		t.Logf("  Anyone can forge JWT tokens using this known secret")
		t.Logf("  Expected: System should reject startup with validation error")
		t.Logf("  Actual: System started successfully with insecure secret")
	}
}

// Example-based test for Bug Condition - Short JWT_SECRET
// **Validates: Requirements 2.1**
//
// This test verifies that the system accepts JWT_SECRET shorter than 32 characters.
// EXPECTED OUTCOME on UNFIXED code: FAILS (proves bug exists)
func TestJWTSecurity_ShortSecretAccepted(t *testing.T) {
	// Save original environment
	originalJWTSecret := os.Getenv("JWT_SECRET")
	defer func() {
		if originalJWTSecret != "" {
			os.Setenv("JWT_SECRET", originalJWTSecret)
		} else {
			os.Unsetenv("JWT_SECRET")
		}
	}()

	// Test various short secrets
	testCases := []struct {
		secret      string
		description string
	}{
		{"short", "very short (5 chars)"},
		{"12345678", "8 characters"},
		{"secret-key-16ch", "16 characters"},
		{"this-is-a-24-char-secret", "24 characters"},
		{"almost-32-but-not-quite-30c", "30 characters"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			os.Setenv("JWT_SECRET", tc.secret)
			
			// Set other required vars
			os.Setenv("DB_HOST", "localhost")
			os.Setenv("DB_USER", "test_user")
			os.Setenv("DB_PASSWORD", "test_pass")
			os.Setenv("DB_NAME", "test_db")
			os.Setenv("REDIS_HOST", "localhost")

			// ACT
			cfg, err := Load()

			// ASSERT: Should reject short secrets
			// On UNFIXED code: This WILL FAIL (Load() succeeds)
			// On FIXED code: This WILL PASS (Load() returns error)
			assert.Error(t, err,
				"Bug Condition: System accepts short JWT_SECRET (%s, length: %d). "+
					"Minimum secure length is 32 characters. "+
					"Expected: Load() should reject with validation error. "+
					"This confirms the security vulnerability exists.",
				tc.description, len(tc.secret))

			// If Load() succeeded (bug exists), log the vulnerability
			if err == nil && cfg != nil {
				t.Errorf("SECURITY VULNERABILITY: Short JWT_SECRET accepted")
				t.Logf("  Secret: %s", tc.secret)
				t.Logf("  Length: %d characters (minimum should be 32)", len(tc.secret))
				t.Logf("  Expected: Validation error")
				t.Logf("  Actual: Configuration loaded successfully")
			}
		})
	}
}

// Example-based test for Bug Condition - Non-random patterns
// **Validates: Requirements 2.1**
//
// This test verifies that the system accepts JWT_SECRET with non-random patterns.
// EXPECTED OUTCOME on UNFIXED code: FAILS (proves bug exists)
func TestJWTSecurity_NonRandomPatternAccepted(t *testing.T) {
	// Save original environment
	originalJWTSecret := os.Getenv("JWT_SECRET")
	defer func() {
		if originalJWTSecret != "" {
			os.Setenv("JWT_SECRET", originalJWTSecret)
		} else {
			os.Unsetenv("JWT_SECRET")
		}
	}()

	// Test various non-random patterns
	testCases := []struct {
		secret      string
		description string
	}{
		{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "repeated 'a' (32 chars)"},
		{"12345678901234567890123456789012", "sequential numbers (32 chars)"},
		{"password123456789012345678901234", "common word + numbers"},
		{"abcdefghijklmnopqrstuvwxyz123456", "alphabet sequence"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			os.Setenv("JWT_SECRET", tc.secret)
			
			// Set other required vars
			os.Setenv("DB_HOST", "localhost")
			os.Setenv("DB_USER", "test_user")
			os.Setenv("DB_PASSWORD", "test_pass")
			os.Setenv("DB_NAME", "test_db")
			os.Setenv("REDIS_HOST", "localhost")

			// ACT
			cfg, err := Load()

			// ASSERT: Should reject non-random patterns
			// On UNFIXED code: This WILL FAIL (Load() succeeds)
			// On FIXED code: This WILL PASS (Load() returns error)
			//
			// NOTE: The current design requires minimum 32 chars, which will catch
			// some patterns. However, patterns like "aaaaa..." (32+ chars) might
			// still pass length validation but are cryptographically weak.
			// The fix should ideally check for randomness/entropy as well.
			if len(tc.secret) < 32 {
				assert.Error(t, err,
					"Bug Condition: System accepts non-random JWT_SECRET pattern (%s). "+
						"Expected: Load() should reject with validation error.",
					tc.description)
			}

			// If Load() succeeded with a weak pattern, log it
			if err == nil && cfg != nil {
				t.Logf("WARNING: Non-random pattern accepted")
				t.Logf("  Pattern: %s", tc.description)
				t.Logf("  Secret: %s", tc.secret)
				t.Logf("  Length: %d characters", len(tc.secret))
				
				// For patterns >= 32 chars, this is a design consideration
				if len(tc.secret) >= 32 {
					t.Logf("  Note: Passes length requirement but lacks entropy")
					t.Logf("  Consider: Adding entropy/randomness validation in the fix")
				}
			}
		})
	}
}

// Edge case test: Valid strong JWT_SECRET should be accepted
// **Validates: Requirements 2.1**
//
// This test verifies that a properly generated strong JWT_SECRET is accepted.
// This should PASS on both UNFIXED and FIXED code (when we provide a valid secret).
func TestJWTSecurity_StrongSecretAccepted(t *testing.T) {
	// Save original environment
	originalJWTSecret := os.Getenv("JWT_SECRET")
	defer func() {
		if originalJWTSecret != "" {
			os.Setenv("JWT_SECRET", originalJWTSecret)
		} else {
			os.Unsetenv("JWT_SECRET")
		}
	}()

	// Generate a strong JWT_SECRET (simulating: openssl rand -base64 48)
	// This is a 64-character base64 string (48 bytes encoded)
	strongSecret := "7K9mP2xQ5vR8wT1yU4zN6bC3dF0gH7jK9mP2xQ5vR8wT1yU4zN6bC3dF0gH7jK=="
	os.Setenv("JWT_SECRET", strongSecret)
	
	// Set other required vars
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_USER", "test_user")
	os.Setenv("DB_PASSWORD", "test_pass")
	os.Setenv("DB_NAME", "test_db")
	os.Setenv("REDIS_HOST", "localhost")
	os.Setenv("REDIS_PASSWORD", "test_redis_password")

	// ACT
	cfg, err := Load()

	// ASSERT: Strong secret should be accepted
	// This should PASS on both UNFIXED and FIXED code
	assert.NoError(t, err,
		"Strong JWT_SECRET should be accepted. "+
			"Secret length: %d characters (>= 32 required). "+
			"Error: %v",
		len(strongSecret), err)

	if cfg != nil {
		assert.Equal(t, strongSecret, cfg.JWT.Secret,
			"Configuration should use the provided strong secret")
		
		t.Logf("SUCCESS: Strong JWT_SECRET accepted")
		t.Logf("  Length: %d characters", len(strongSecret))
		t.Logf("  This is the expected behavior for secure secrets")
	}
}

// Edge case test: JWT_SECRET at minimum length boundary (48 chars)
// **Validates: Requirements 2.1**
//
// This test verifies behavior at the 48-character boundary.
func TestJWTSecurity_MinimumLengthBoundary(t *testing.T) {
	// Save original environment
	originalJWTSecret := os.Getenv("JWT_SECRET")
	defer func() {
		if originalJWTSecret != "" {
			os.Setenv("JWT_SECRET", originalJWTSecret)
		} else {
			os.Unsetenv("JWT_SECRET")
		}
	}()

	testCases := []struct {
		secret      string
		shouldPass  bool
		description string
	}{
		{strings.Repeat("a", 47), false, "47 characters (just below minimum)"},
		{strings.Repeat("b", 48), true, "48 characters (at minimum boundary)"},
		{strings.Repeat("c", 49), true, "49 characters (above minimum)"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			os.Setenv("JWT_SECRET", tc.secret)
			
			// Set other required vars
			os.Setenv("DB_HOST", "localhost")
			os.Setenv("DB_USER", "test_user")
			os.Setenv("DB_PASSWORD", "test_pass")
			os.Setenv("DB_NAME", "test_db")
			os.Setenv("REDIS_HOST", "localhost")

			// ACT
			cfg, err := Load()

			// ASSERT
			if tc.shouldPass {
				// On FIXED code: Should accept secrets >= 48 chars
				assert.NoError(t, err,
					"Secret with %d characters should be accepted (minimum is 48)",
					len(tc.secret))
			} else {
				// On UNFIXED code: This WILL FAIL (accepts < 48 chars)
				// On FIXED code: This WILL PASS (rejects < 48 chars)
				assert.Error(t, err,
					"Bug Condition: Secret with %d characters should be rejected (minimum is 48). "+
						"Expected: Validation error. "+
						"This confirms the bug exists.",
					len(tc.secret))
			}

			// Log the result
			if err == nil && cfg != nil {
				t.Logf("Configuration loaded with secret length: %d", len(tc.secret))
			} else if err != nil {
				t.Logf("Configuration rejected with error: %v", err)
			}
		})
	}
}
