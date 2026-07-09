package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

/**
 * Bug 7 Exploration Test: Rate Limiting Disabled When Redis Fails
 *
 * **Validates: Requirements 2.5**
 *
 * Bug Condition (from design.md):
 * WHEN Redis is unavailable (connection fails)
 * THEN the system creates a no-op middleware that allows unlimited requests
 * AND rate limiting is completely disabled
 * AND the system is vulnerable to DDoS attacks
 *
 * Expected Behavior (from design.md - Property 4):
 * WHEN Redis is unavailable
 * THEN the system SHALL activate an in-memory rate limiter
 * AND enforce the same limits (100 requests/minute per tenant)
 * AND maintain < 1ms overhead per request
 *
 * IMPORTANT: This is a bug exploration test
 * - On UNFIXED code: Test should FAIL (unlimited requests accepted) ✓ Expected
 * - After fix: Test should PASS (in-memory rate limiter enforces limits)
 */

/**
 * Property 1: Bug Condition - Rate Limiting Disabled When Redis Fails
 *
 * This test verifies that the current implementation DOES NOT
 * implement rate limiting when Redis is unavailable.
 *
 * Test Strategy:
 * 1. Simulate Redis failure (nil Redis client)
 * 2. Create rate limiter middleware with nil Redis
 * 3. Make > 100 requests/minute (exceeding expected limit)
 * 4. Verify that ALL requests are accepted (no 429 responses)
 * 5. Confirm bug exists: rate limiting is disabled
 *
 * Expected on UNFIXED code:
 * - All requests accepted (200 OK)
 * - Zero 429 (Too Many Requests) responses
 * - Test FAILS (bug confirmed)
 *
 * Expected after FIX:
 * - First ~100 requests accepted
 * - Remaining requests get 429 responses
 * - Test PASSES (bug fixed)
 */
func TestBugCondition_RateLimitingDisabledWhenRedisFails(t *testing.T) {
	fmt.Println("\n=== Bug 7 Exploration Test: Rate Limiting Disabled When Redis Fails ===")
	fmt.Println()

	// Setup: Create a nil Redis client to simulate Redis failure
	var nilRedisClient *redis.Client = nil

	// Create logger
	logger, _ := zap.NewDevelopment()

	// Create rate limiter with nil Redis (simulating Redis failure)
	rateLimiter := NewRateLimiter(nilRedisClient, 100, time.Minute, logger)

	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(rateLimiter)

	// Test endpoint
	router.GET("/api/v1/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	fmt.Println("Test Configuration:")
	fmt.Println("  - Redis Client: nil (simulating failure)")
	fmt.Println("  - Expected Rate Limit: 100 requests/minute")
	fmt.Println("  - Test Requests: 150 (exceeding limit)")
	fmt.Println("  - Expected Bug: All requests accepted (no rate limiting)")

	// Make 150 requests (exceeding the 100 req/min limit)
	const totalRequests = 150
	results := struct {
		accepted     int
		rateLimited  int
		other        int
	}{}

	fmt.Println("\n=== Making Rapid Requests ===")
	fmt.Printf("Sending %d requests to test rate limiting...\n\n", totalRequests)

	startTime := time.Now()

	for i := 0; i < totalRequests; i++ {
		req, _ := http.NewRequest("GET", "/api/v1/test", nil)
		req.Header.Set("X-Forwarded-For", "192.168.1.100") // Same IP for all requests

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code == http.StatusOK {
			results.accepted++
		} else if w.Code == http.StatusTooManyRequests {
			results.rateLimited++
		} else {
			results.other++
		}

		// Progress indicator
		if (i+1)%30 == 0 {
			fmt.Printf("Progress: %d/%d requests\n", i+1, totalRequests)
		}
	}

	duration := time.Since(startTime)
	requestsPerSecond := float64(totalRequests) / duration.Seconds()

	fmt.Println("\n=== Test Results ===")
	fmt.Printf("Duration: %v (%.0f req/sec)\n", duration, requestsPerSecond)
	fmt.Println("\nResponse Breakdown:")
	fmt.Printf("  - Total Requests: %d\n", totalRequests)
	fmt.Printf("  - Accepted (200): %d\n", results.accepted)
	fmt.Printf("  - Rate Limited (429): %d %s\n", results.rateLimited, 
		map[bool]string{true: "✓", false: "✗"}[results.rateLimited > 0])
	fmt.Printf("  - Other: %d\n", results.other)

	// Calculate rate limiting effectiveness
	rateLimitingActive := results.rateLimited > 0
	rateLimitPercentage := float64(results.rateLimited) / float64(totalRequests) * 100

	fmt.Println("\n=== Analysis ===")
	fmt.Printf("Rate Limiting Active: %s\n", 
		map[bool]string{true: "YES ✓", false: "NO ✗"}[rateLimitingActive])
	fmt.Printf("Rate Limited Percentage: %.1f%%\n", rateLimitPercentage)

	if rateLimitingActive {
		fmt.Printf("Requests Before Limit: %d\n", results.accepted)
		fmt.Printf("Requests After Limit: %d\n", results.rateLimited)
	}

	// Determine bug status
	// Bug exists if NO rate limiting is detected when Redis is unavailable
	bugExists := !rateLimitingActive

	fmt.Println("\n=== Bug Status ===")
	if bugExists {
		fmt.Println("✗ BUG CONFIRMED: Rate limiting is disabled when Redis fails!")
		fmt.Println("  All requests were processed without rate limiting.")
		fmt.Println("  The system is vulnerable to DDoS attacks.")
		fmt.Println("\n  Expected Behavior:")
		fmt.Println("    - First ~100 requests: Accepted (200 OK)")
		fmt.Println("    - Remaining 50 requests: Rate Limited (429)")
		fmt.Println("\n  Actual Behavior:")
		fmt.Printf("    - All %d requests: Accepted (200 OK)\n", results.accepted)
		fmt.Println("    - Zero 429 responses")
		fmt.Println("    - No-op middleware allows unlimited requests")
		fmt.Println("\n  Root Cause (from main.go:416-419):")
		fmt.Println("    if redisClient != nil {")
		fmt.Println("      rateLimiter = middleware.NewRateLimiter(...)")
		fmt.Println("    } else {")
		fmt.Println("      rateLimiter = func(c *gin.Context) { c.Next() } // NO-OP!")
		fmt.Println("    }")
		fmt.Println("\n✗ EXPLORATION TEST FAILED (Bug Exists)")
		fmt.Println("  This test will PASS after implementing in-memory rate limiter fallback.")

		// Test assertions
		assert.Equal(t, totalRequests, results.accepted, 
			"Expected all requests to be accepted (bug condition)")
		assert.Equal(t, 0, results.rateLimited, 
			"Expected zero rate limited responses (bug condition)")
		assert.False(t, rateLimitingActive, 
			"Expected rate limiting to be inactive (bug condition)")

		t.Log("Bug Condition Confirmed: Rate limiting is disabled when Redis fails")
		t.Log("Counterexample: Made 150 requests, all accepted, zero 429 responses")
		t.Log("Expected: Should have ~50 requests rate limited (429)")
		
		// Mark test as failed (expected on unfixed code)
		t.Fail()
	} else {
		fmt.Println("✓ Rate limiting is ACTIVE even without Redis!")
		fmt.Printf("  %d requests were rate limited (429).\n", results.rateLimited)
		fmt.Println("  The system has in-memory rate limiter fallback.")
		fmt.Println("\n✓ EXPLORATION TEST PASSED (Bug Fixed)")
		fmt.Println("  In-memory rate limiter is working correctly.")

		// Test assertions
		assert.Greater(t, results.rateLimited, 0, 
			"Expected some requests to be rate limited (bug fixed)")
		assert.Less(t, results.accepted, totalRequests, 
			"Expected some requests to be rejected (bug fixed)")
		assert.True(t, rateLimitingActive, 
			"Expected rate limiting to be active (bug fixed)")

		t.Log("Bug Fixed: In-memory rate limiter is active when Redis fails")
		t.Log(fmt.Sprintf("Accepted: %d, Rate Limited: %d", results.accepted, results.rateLimited))
	}
}

/**
 * Property 2: Bug Condition - Implementation Details Verification
 *
 * This test verifies the specific implementation details from the bug condition:
 * - isBugCondition where RedisAvailable == false AND RateLimiterActive == false
 *
 * Test Strategy:
 * 1. Verify that NewRateLimiter with nil Redis returns a middleware
 * 2. Verify that the middleware does NOT enforce rate limits
 * 3. Verify that the middleware acts as a no-op (passes all requests through)
 */
func TestBugCondition_NoOpMiddlewareWhenRedisUnavailable(t *testing.T) {
	fmt.Println("\n=== Bug 7 Implementation Details Test: No-Op Middleware ===")
	fmt.Println()

	// Setup: Create a nil Redis client
	var nilRedisClient *redis.Client = nil
	logger, _ := zap.NewDevelopment()

	// Create rate limiter with nil Redis
	rateLimiter := NewRateLimiter(nilRedisClient, 100, time.Minute, logger)

	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(rateLimiter)

	router.GET("/api/v1/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	fmt.Println("Test: Verify middleware behavior with nil Redis")
	fmt.Println("  - Redis Client: nil")
	fmt.Println("  - Expected: Middleware acts as no-op (all requests pass)")

	// Test 1: Single request should pass
	req1, _ := http.NewRequest("GET", "/api/v1/test", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code, "First request should be accepted")

	// Test 2: 200 rapid requests should all pass (no rate limiting)
	successCount := 0
	for i := 0; i < 200; i++ {
		req, _ := http.NewRequest("GET", "/api/v1/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code == http.StatusOK {
			successCount++
		}
	}

	fmt.Println("\n=== Results ===")
	fmt.Printf("Requests Made: 200\n")
	fmt.Printf("Requests Accepted: %d\n", successCount)
	fmt.Printf("Requests Rate Limited: %d\n", 200-successCount)

	// Bug condition: All requests should pass (no rate limiting)
	if successCount == 200 {
		fmt.Println("\n✗ BUG CONFIRMED: Middleware is a no-op!")
		fmt.Println("  All 200 requests passed without rate limiting.")
		fmt.Println("  Expected: ~100 accepted, ~100 rate limited")
		fmt.Println("  Actual: 200 accepted, 0 rate limited")
		
		assert.Equal(t, 200, successCount, "Bug condition: all requests should pass")
		t.Log("Bug Condition: Middleware does not enforce rate limits when Redis is nil")
		t.Fail() // Expected on unfixed code
	} else {
		fmt.Println("\n✓ Rate limiting is active!")
		fmt.Printf("  %d requests were rate limited.\n", 200-successCount)
		fmt.Println("  In-memory fallback is working.")
		
		assert.Less(t, successCount, 200, "Bug fixed: some requests should be rate limited")
		t.Log("Bug Fixed: Middleware enforces rate limits even without Redis")
	}
}

/**
 * Property 3: Bug Condition - Performance Impact Verification
 *
 * This test verifies that when Redis fails, the no-op middleware
 * has minimal overhead (since it does nothing).
 *
 * After fix, the in-memory rate limiter should maintain < 1ms overhead
 * as specified in Property 4 of the design document.
 */
func TestBugCondition_PerformanceWithoutRedis(t *testing.T) {
	fmt.Println("\n=== Bug 7 Performance Test: Middleware Overhead ===")
	fmt.Println()

	var nilRedisClient *redis.Client = nil
	logger, _ := zap.NewDevelopment()

	rateLimiter := NewRateLimiter(nilRedisClient, 100, time.Minute, logger)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(rateLimiter)

	router.GET("/api/v1/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	fmt.Println("Test: Measure middleware overhead")
	fmt.Println("  - Expected (unfixed): Near-zero overhead (no-op)")
	fmt.Println("  - Expected (fixed): < 1ms overhead (in-memory limiter)")

	// Measure time for 1000 requests
	const testRequests = 1000
	startTime := time.Now()

	for i := 0; i < testRequests; i++ {
		req, _ := http.NewRequest("GET", "/api/v1/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	duration := time.Since(startTime)
	avgOverhead := duration / testRequests

	fmt.Println("\n=== Performance Results ===")
	fmt.Printf("Total Requests: %d\n", testRequests)
	fmt.Printf("Total Duration: %v\n", duration)
	fmt.Printf("Average Overhead: %v per request\n", avgOverhead)

	// On unfixed code: overhead should be very low (no-op)
	// On fixed code: overhead should be < 1ms (in-memory limiter)
	if avgOverhead < 100*time.Microsecond {
		fmt.Println("\n✗ Very low overhead detected (likely no-op middleware)")
		fmt.Printf("  Average: %v (< 100µs suggests no rate limiting logic)\n", avgOverhead)
		fmt.Println("  This is consistent with the bug condition.")
		t.Log("Bug Condition: Near-zero overhead suggests no-op middleware")
	} else if avgOverhead < time.Millisecond {
		fmt.Println("\n✓ Acceptable overhead detected (likely in-memory limiter)")
		fmt.Printf("  Average: %v (< 1ms as specified in Property 4)\n", avgOverhead)
		fmt.Println("  In-memory rate limiter is active and performant.")
		t.Log("Bug Fixed: Overhead within acceptable range for in-memory limiter")
	} else {
		fmt.Println("\n⚠ High overhead detected")
		fmt.Printf("  Average: %v (> 1ms exceeds Property 4 requirement)\n", avgOverhead)
		fmt.Println("  Rate limiter may need optimization.")
		t.Log("Warning: Overhead exceeds 1ms requirement")
	}

	// This test is informational, not a pass/fail assertion
	t.Logf("Middleware overhead: %v per request", avgOverhead)
}
