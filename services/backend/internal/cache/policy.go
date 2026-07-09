package cache

import "time"

// Global Cache TTL Policy
// Requirement 5.2 & 5.7: Unified TTL values across the system
const (
	// Default TTL for general data
	DefaultTTL = 5 * time.Minute

	// Routing & Channels TTL (High consistency required)
	RoutingTTL = 5 * time.Minute

	// Idempotency Keys TTL (Prevent duplicates)
	// Reduced from 24h to 1h to optimize Redis memory as per audit
	IdempotencyTTL = 1 * time.Hour

	// Balance & Accounting TTL (Very short, high volatility)
	AccountingTTL = 30 * time.Second

	// Statistics & Metrics TTL
	StatsTTL = 10 * time.Minute

	// Phase 3.2: Cache Invalidation Versioning
	// Increment this when changing the structure of cached data
	CacheVersion = "v1"
)

// GetVersionedKey prefixes the key with the current cache version
func GetVersionedKey(key string) string {
	return CacheVersion + ":" + key
}

// Policy defines various cache behaviors
type Policy struct {
	TTL            time.Duration
	SecureKeys     bool
	CircuitBreaker bool
}

// GetRoutingPolicy returns the standard policy for routing data
func GetRoutingPolicy() Policy {
	return Policy{
		TTL:            RoutingTTL,
		SecureKeys:     true,
		CircuitBreaker: true,
	}
}
