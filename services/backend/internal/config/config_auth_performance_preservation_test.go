package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/al-hudhud/backend/pkg/jwt"
)

// TestAuthenticationPerformance_BaselineMeasurement establishes baseline performance
// for JWT authentication operations before implementing security fixes.
// Property 2: Preservation - JWT Authentication Performance
// Requirement 3.1: Authentication SHALL maintain performance < 100ms
func TestAuthenticationPerformance_BaselineMeasurement(t *testing.T) {
	// Observation-first methodology: measure current performance on unfixed code
	
	// Setup: Use current (insecure) JWT configuration
	jwt.SetJWTSecret("change-me-in-production-to-a-secure-random-string")
	
	// Test data
	userID := "test-user-123"
	email := "test@example.com"
	userType := "tenant"
	tenantID := "test-tenant-456"
	staffID := ""
	permissions := []string{"read:messages", "write:messages", "admin:users"}
	
	const iterations = 100
	var totalGenerationTime time.Duration
	var totalValidationTime time.Duration
	
	tokens := make([]string, iterations)
	
	// Measure JWT generation performance
	for i := 0; i < iterations; i++ {
		start := time.Now()
		token, err := jwt.GenerateToken(userID, email, userType, tenantID, staffID, permissions)
		generationTime := time.Since(start)
		
		require.NoError(t, err, "JWT generation should not fail")
		require.NotEmpty(t, token, "Generated token should not be empty")
		
		tokens[i] = token
		totalGenerationTime += generationTime
	}
	
	avgGenerationTime := totalGenerationTime / iterations
	t.Logf("Average JWT generation time: %v", avgGenerationTime)
	
	// Measure JWT validation performance
	for i := 0; i < iterations; i++ {
		start := time.Now()
		claims, err := jwt.ValidateToken(tokens[i])
		validationTime := time.Since(start)
		
		require.NoError(t, err, "JWT validation should not fail")
		require.NotNil(t, claims, "Claims should not be nil")
		require.Equal(t, userID, claims.UserID, "UserID should match")
		require.Equal(t, tenantID, claims.TenantID, "TenantID should match")
		
		totalValidationTime += validationTime
	}
	
	avgValidationTime := totalValidationTime / iterations
	t.Logf("Average JWT validation time: %v", avgValidationTime)
	
	// Calculate total authentication time (generation + validation)
	avgTotalTime := avgGenerationTime + avgValidationTime
	t.Logf("Average total authentication time: %v", avgTotalTime)
	
	// Property 2: Preservation - Authentication performance must remain < 100ms
	assert.Less(t, avgTotalTime.Milliseconds(), int64(100),
		"Total authentication time (generation + validation) must be < 100ms")
	
	// Additional assertion: Individual operations should be very fast (< 10ms each)
	assert.Less(t, avgGenerationTime.Milliseconds(), int64(10),
		"JWT generation should be < 10ms")
	assert.Less(t, avgValidationTime.Milliseconds(), int64(10),
		"JWT validation should be < 10ms")
}

// TestJWTGeneration_PerformanceUnderLoad tests JWT generation performance
// with varying payload sizes (different permission counts).
// Property 2: Preservation - Performance should scale linearly with payload size
func TestJWTGeneration_PerformanceUnderLoad(t *testing.T) {
	jwt.SetJWTSecret("change-me-in-production-to-a-secure-random-string")
	
	userID := "test-user-123"
	email := "test@example.com"
	userType := "tenant"
	tenantID := "test-tenant-456"
	staffID := ""
	
	testCases := []struct {
		name            string
		permissionCount int
	}{
		{"No permissions", 0},
		{"Few permissions", 5},
		{"Medium permissions", 20},
		{"Many permissions", 50},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Generate permissions array
			permissions := make([]string, tc.permissionCount)
			for i := 0; i < tc.permissionCount; i++ {
				permissions[i] = "permission:" + string(rune('a'+i%26))
			}
			
			const iterations = 50
			var totalTime time.Duration
			
			for i := 0; i < iterations; i++ {
				start := time.Now()
				token, err := jwt.GenerateToken(userID, email, userType, tenantID, staffID, permissions)
				elapsed := time.Since(start)
				
				require.NoError(t, err)
				require.NotEmpty(t, token)
				totalTime += elapsed
			}
			
			avgTime := totalTime / iterations
			t.Logf("Average time with %d permissions: %v", tc.permissionCount, avgTime)
			
			// Property 2: Even with many permissions, generation should be < 50ms
			assert.Less(t, avgTime.Milliseconds(), int64(50),
				"JWT generation with %d permissions should be < 50ms", tc.permissionCount)
		})
	}
}

// TestAuthentication_ConcurrentPerformance tests authentication performance
// under concurrent load to ensure thread-safety doesn't impact performance.
// Property 2: Preservation - Concurrent operations should maintain performance
func TestAuthentication_ConcurrentPerformance(t *testing.T) {
	jwt.SetJWTSecret("change-me-in-production-to-a-secure-random-string")
	
	userID := "test-user-123"
	email := "test@example.com"
	userType := "tenant"
	tenantID := "test-tenant-456"
	staffID := ""
	permissions := []string{"read:messages", "write:messages"}
	
	const numWorkers = 10
	const iterationsPerWorker = 10
	
	results := make(chan time.Duration, numWorkers*iterationsPerWorker)
	
	// Launch concurrent workers
	for w := 0; w < numWorkers; w++ {
		go func(workerID int) {
			for i := 0; i < iterationsPerWorker; i++ {
				start := time.Now()
				
				// Generate token
				token, err := jwt.GenerateToken(userID, email, userType, tenantID, staffID, permissions)
				if err != nil {
					t.Errorf("Worker %d: JWT generation failed: %v", workerID, err)
					return
				}
				
				// Validate token
				claims, err := jwt.ValidateToken(token)
				if err != nil {
					t.Errorf("Worker %d: JWT validation failed: %v", workerID, err)
					return
				}
				
				if claims.UserID != userID {
					t.Errorf("Worker %d: UserID mismatch", workerID)
					return
				}
				
				elapsed := time.Since(start)
				results <- elapsed
			}
		}(w)
	}
	
	// Collect results
	var totalTime time.Duration
	for i := 0; i < numWorkers*iterationsPerWorker; i++ {
		totalTime += <-results
	}
	
	avgTime := totalTime / (numWorkers * iterationsPerWorker)
	t.Logf("Average concurrent authentication time: %v", avgTime)
	
	// Property 2: Concurrent operations should maintain < 100ms performance
	assert.Less(t, avgTime.Milliseconds(), int64(100),
		"Concurrent authentication should maintain < 100ms performance")
}
