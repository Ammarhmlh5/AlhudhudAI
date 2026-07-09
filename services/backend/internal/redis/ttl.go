package redis

import "time"

// TTLConfig holds TTL configuration for different data types
type TTLConfig struct {
	Tariff       time.Duration
	Package      time.Duration
	Subscription time.Duration
	Balance      time.Duration
	Usage        time.Duration
	Stats        time.Duration
	Gateway      time.Duration
	Rollover     time.Duration
	Staff        time.Duration
	Tenant       time.Duration
	RateLimit    time.Duration
	Idempotency  time.Duration
	Session      time.Duration
}

// DefaultTTLConfig returns default TTL configuration
func DefaultTTLConfig() *TTLConfig {
	return &TTLConfig{
		Tariff:       1 * time.Hour,    // Tariffs rarely change
		Package:      30 * time.Minute, // Packages may change
		Subscription: 5 * time.Minute,  // Subscriptions change frequently
		Balance:      1 * time.Minute,  // Balance changes constantly
		Usage:        1 * time.Minute,  // Usage changes rapidly - consistent with Balance TTL
		Stats:        10 * time.Minute, // Statistics
		Gateway:      15 * time.Minute, // Dedicated gateways
		Rollover:     5 * time.Minute,  // Rollover messages
		Staff:        15 * time.Minute, // Staff profiles
		Tenant:       30 * time.Minute, // Tenant data
		RateLimit:    1 * time.Minute,  // Rate limit counters
		Idempotency:  1 * time.Hour,    // Idempotency keys
		Session:      24 * time.Hour,   // User sessions
	}
}

// GetTTL returns TTL for a specific data type
func (c *TTLConfig) GetTTL(dataType string) time.Duration {
	switch dataType {
	case "tariff":
		return c.Tariff
	case "package":
		return c.Package
	case "subscription":
		return c.Subscription
	case "balance":
		return c.Balance
	case "usage":
		return c.Usage
	case "stats":
		return c.Stats
	case "gateway":
		return c.Gateway
	case "rollover":
		return c.Rollover
	case "staff":
		return c.Staff
	case "tenant":
		return c.Tenant
	case "rate_limit":
		return c.RateLimit
	case "idempotency":
		return c.Idempotency
	case "session":
		return c.Session
	default:
		return 5 * time.Minute // Default TTL
	}
}
