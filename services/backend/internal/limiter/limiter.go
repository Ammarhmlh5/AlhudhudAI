package limiter

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// TenantLimiter manages rate limits for each tenant
type TenantLimiter struct {
	mu          sync.Mutex
	limits      map[uuid.UUID]*rateLimit
	globalLimit int // Global messages per minute
}

type rateLimit struct {
	count     int
	lastReset time.Time
}

func NewTenantLimiter(globalLimit int) *TenantLimiter {
	return &TenantLimiter{
		limits:      make(map[uuid.UUID]*rateLimit),
		globalLimit: globalLimit,
	}
}

// Allow checks if a tenant is within their rate limit
func (tl *TenantLimiter) Allow(tenantID uuid.UUID, tenantLimit int) bool {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	now := time.Now()
	limit, exists := tl.limits[tenantID]

	if !exists || now.Sub(limit.lastReset) >= time.Minute {
		tl.limits[tenantID] = &rateLimit{
			count:     1,
			lastReset: now,
		}
		return true
	}

	if limit.count >= tenantLimit {
		return false
	}

	limit.count++
	return true
}

// GetGlobalLimit returns the configured global throughput limit
func (tl *TenantLimiter) GetGlobalLimit() int {
	return tl.globalLimit
}
