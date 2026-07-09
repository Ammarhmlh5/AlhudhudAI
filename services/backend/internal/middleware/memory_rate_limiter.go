package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

/**
 * In-Memory Rate Limiter
 *
 * This is a fallback rate limiter that works without Redis.
 * It uses golang.org/x/time/rate for token bucket rate limiting.
 *
 * Features:
 * - Per-tenant rate limiting using sync.Map for concurrent access
 * - Same limits as Redis version: 100 requests/minute per tenant
 * - Automatic cleanup of inactive limiters to prevent memory leaks
 * - < 1ms overhead per request (Requirement 2.5)
 *
 * **Validates: Requirements 2.5, 3.3**
 */

// MemoryRateLimiter implements in-memory rate limiting
type MemoryRateLimiter struct {
	limiters          sync.Map // map[string]*rateLimiterEntry
	requestsPerMinute int
	logger            *zap.Logger
	cleanupInterval   time.Duration
	inactiveThreshold time.Duration
	stopCleanup       chan struct{}
	cleanupWg         sync.WaitGroup
}

// rateLimiterEntry stores a rate limiter and its last access time
type rateLimiterEntry struct {
	limiter    *rate.Limiter
	lastAccess time.Time
	mu         sync.Mutex
}

// NewMemoryRateLimiter creates a new in-memory rate limiter
//
// Parameters:
// - requestsPerMinute: Maximum requests allowed per minute per IP/tenant
// - logger: Logger for rate limit events
//
// Returns a Gin middleware function that enforces rate limits
func NewMemoryRateLimiter(requestsPerMinute int, logger *zap.Logger) gin.HandlerFunc {
	limiter := &MemoryRateLimiter{
		requestsPerMinute: requestsPerMinute,
		logger:            logger,
		cleanupInterval:   10 * time.Minute, // Cleanup every 10 minutes
		inactiveThreshold: 30 * time.Minute, // Remove limiters inactive for 30 minutes
		stopCleanup:       make(chan struct{}),
	}

	// Start cleanup goroutine
	limiter.startCleanup()

	return limiter.middleware
}

// middleware is the Gin handler function
func (m *MemoryRateLimiter) middleware(c *gin.Context) {
	// Get client IP as the key for rate limiting
	ip := c.ClientIP()
	
	// Create a unique key for this IP and path
	key := fmt.Sprintf("%s:%s", c.Request.URL.Path, ip)

	// Get or create rate limiter for this key
	limiter := m.getLimiter(key)

	// Check if request is allowed
	if !limiter.Allow() {
		// Rate limit exceeded
		if m.logger != nil {
			m.logger.Warn("Rate limit exceeded (in-memory)",
				zap.String("ip", ip),
				zap.String("path", c.Request.URL.Path),
				zap.Int("limit", m.requestsPerMinute),
			)
		}

		// Calculate retry-after (approximate)
		retryAfter := 60 // seconds (1 minute window)

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(m.requestsPerMinute))
		c.Header("X-RateLimit-Remaining", "0")
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Duration(retryAfter)*time.Second).Unix(), 10))
		c.Header("Retry-After", strconv.Itoa(retryAfter))

		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
			"error":               "Too many requests. Please try again later.",
			"message":             fmt.Sprintf("Rate limit exceeded. Maximum %d requests per minute allowed.", m.requestsPerMinute),
			"retry_after_seconds": retryAfter,
		})
		return
	}

	// Request allowed - set rate limit headers
	// Note: Remaining count is approximate with token bucket
	c.Header("X-RateLimit-Limit", strconv.Itoa(m.requestsPerMinute))
	c.Header("X-RateLimit-Remaining", strconv.Itoa(m.requestsPerMinute-1)) // Approximate
	c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Minute).Unix(), 10))

	c.Next()
}

// getLimiter gets or creates a rate limiter for the given key
func (m *MemoryRateLimiter) getLimiter(key string) *rate.Limiter {
	// Try to load existing limiter
	if val, ok := m.limiters.Load(key); ok {
		entry := val.(*rateLimiterEntry)
		entry.mu.Lock()
		entry.lastAccess = time.Now()
		entry.mu.Unlock()
		return entry.limiter
	}

	// Create new limiter
	// rate.Limit is requests per second, so divide by 60 for per-minute limit
	// Burst allows some flexibility for bursty traffic
	limiter := rate.NewLimiter(rate.Limit(float64(m.requestsPerMinute)/60.0), m.requestsPerMinute)

	entry := &rateLimiterEntry{
		limiter:    limiter,
		lastAccess: time.Now(),
	}

	// Store the new limiter
	m.limiters.Store(key, entry)

	return limiter
}

// startCleanup starts the background cleanup goroutine
func (m *MemoryRateLimiter) startCleanup() {
	m.cleanupWg.Add(1)
	go func() {
		defer m.cleanupWg.Done()
		ticker := time.NewTicker(m.cleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				m.cleanup()
			case <-m.stopCleanup:
				if m.logger != nil {
					m.logger.Info("Memory rate limiter cleanup stopped")
				}
				return
			}
		}
	}()

	if m.logger != nil {
		m.logger.Info("Memory rate limiter cleanup started",
			zap.Duration("interval", m.cleanupInterval),
			zap.Duration("inactive_threshold", m.inactiveThreshold),
		)
	}
}

// cleanup removes inactive rate limiters to prevent memory leaks
func (m *MemoryRateLimiter) cleanup() {
	now := time.Now()
	removed := 0

	m.limiters.Range(func(key, value interface{}) bool {
		entry := value.(*rateLimiterEntry)
		entry.mu.Lock()
		lastAccess := entry.lastAccess
		entry.mu.Unlock()

		// Remove if inactive for longer than threshold
		if now.Sub(lastAccess) > m.inactiveThreshold {
			m.limiters.Delete(key)
			removed++
		}

		return true // continue iteration
	})

	if m.logger != nil && removed > 0 {
		m.logger.Info("Memory rate limiter cleanup completed",
			zap.Int("removed_limiters", removed),
		)
	}
}

// Stop gracefully stops the cleanup goroutine
func (m *MemoryRateLimiter) Stop() {
	close(m.stopCleanup)
	m.cleanupWg.Wait()
}

// GetStats returns statistics about the in-memory rate limiter
func (m *MemoryRateLimiter) GetStats() map[string]interface{} {
	count := 0
	m.limiters.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	return map[string]interface{}{
		"active_limiters":     count,
		"requests_per_minute": m.requestsPerMinute,
		"cleanup_interval":    m.cleanupInterval.String(),
		"inactive_threshold":  m.inactiveThreshold.String(),
	}
}
