package resources

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/time/rate"
)

// ResourceQuota represents the resource quotas for a dedicated gateway
type ResourceQuota struct {
	ID                      int
	GatewayID               int
	CPUCores                *float64
	MemoryMB                *int
	BandwidthMbps           *int
	MaxConcurrentConnections *int
	StorageGB               *int
	MaxDBConnections        *int
	MaxQueriesPerSecond     *int
	RedisMemoryMB           *int
	RedisMaxKeys            *int
	MaxMessagesPerHour      *int
	MaxMessagesPerDay       *int
	MaxQueueDepth           *int
	APIRateLimitPerMinute   *int
	APIRateLimitPerHour     *int
	MaxWorkerThreads        *int
	IsActive                bool
	EnforcementMode         string // 'soft' or 'hard'
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// ResourceUsage represents the current resource usage for a gateway
type ResourceUsage struct {
	ID                        int64
	GatewayID                 int
	TenantID                  uuid.UUID
	Timestamp                 time.Time
	CPUUsagePercent           *float64
	MemoryUsageMB             *int
	BandwidthUsageMbps        *float64
	ActiveConnections         *int
	DBConnectionsActive       *int
	QueriesPerSecond          *int
	RedisMemoryUsageMB        *int
	RedisKeysCount            *int
	MessagesProcessedLastHour *int
	MessagesProcessedToday    *int
	CurrentQueueDepth         *int
	APIRequestsLastMinute     *int
	APIRequestsLastHour       *int
	ActiveWorkerThreads       *int
	CreatedAt                 time.Time
}

// QuotaCheckResult represents the result of a quota check
type QuotaCheckResult struct {
	Exceeded      bool
	CurrentUsage  interface{}
	Quota         interface{}
	UsagePercent  float64
	ResourceType  string
	EnforcementMode string
}

// ResourceLimiter manages resource quotas and usage for dedicated gateways
type ResourceLimiter struct {
	db        *sql.DB
	gatewayID int
	tenantID  uuid.UUID
}

// NewResourceLimiter creates a new ResourceLimiter instance
func NewResourceLimiter(db *sql.DB, gatewayID int, tenantID uuid.UUID) *ResourceLimiter {
	return &ResourceLimiter{
		db:        db,
		gatewayID: gatewayID,
		tenantID:  tenantID,
	}
}

// GetResourceQuota retrieves the resource quota for the gateway
func (rl *ResourceLimiter) GetResourceQuota(ctx context.Context) (*ResourceQuota, error) {
	query := `
		SELECT 
			id, gateway_id, cpu_cores, memory_mb, bandwidth_mbps,
			max_concurrent_connections, storage_gb, max_db_connections,
			max_queries_per_second, redis_memory_mb, redis_max_keys,
			max_messages_per_hour, max_messages_per_day, max_queue_depth,
			api_rate_limit_per_minute, api_rate_limit_per_hour,
			max_worker_threads, is_active, enforcement_mode,
			created_at, updated_at
		FROM resource_quotas
		WHERE gateway_id = $1 AND is_active = true
	`

	quota := &ResourceQuota{}
	err := rl.db.QueryRowContext(ctx, query, rl.gatewayID).Scan(
		&quota.ID,
		&quota.GatewayID,
		&quota.CPUCores,
		&quota.MemoryMB,
		&quota.BandwidthMbps,
		&quota.MaxConcurrentConnections,
		&quota.StorageGB,
		&quota.MaxDBConnections,
		&quota.MaxQueriesPerSecond,
		&quota.RedisMemoryMB,
		&quota.RedisMaxKeys,
		&quota.MaxMessagesPerHour,
		&quota.MaxMessagesPerDay,
		&quota.MaxQueueDepth,
		&quota.APIRateLimitPerMinute,
		&quota.APIRateLimitPerHour,
		&quota.MaxWorkerThreads,
		&quota.IsActive,
		&quota.EnforcementMode,
		&quota.CreatedAt,
		&quota.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no active resource quota found for gateway %d", rl.gatewayID)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get resource quota: %w", err)
	}

	return quota, nil
}

// GetLatestResourceUsage retrieves the latest resource usage for the gateway
func (rl *ResourceLimiter) GetLatestResourceUsage(ctx context.Context) (*ResourceUsage, error) {
	query := `
		SELECT 
			id, gateway_id, tenant_id, timestamp,
			cpu_usage_percent, memory_usage_mb, bandwidth_usage_mbps,
			active_connections, db_connections_active, queries_per_second,
			redis_memory_usage_mb, redis_keys_count,
			messages_processed_last_hour, messages_processed_today,
			current_queue_depth, api_requests_last_minute,
			api_requests_last_hour, active_worker_threads, created_at
		FROM resource_usage
		WHERE gateway_id = $1 AND tenant_id = $2
		ORDER BY timestamp DESC
		LIMIT 1
	`

	usage := &ResourceUsage{}
	err := rl.db.QueryRowContext(ctx, query, rl.gatewayID, rl.tenantID).Scan(
		&usage.ID,
		&usage.GatewayID,
		&usage.TenantID,
		&usage.Timestamp,
		&usage.CPUUsagePercent,
		&usage.MemoryUsageMB,
		&usage.BandwidthUsageMbps,
		&usage.ActiveConnections,
		&usage.DBConnectionsActive,
		&usage.QueriesPerSecond,
		&usage.RedisMemoryUsageMB,
		&usage.RedisKeysCount,
		&usage.MessagesProcessedLastHour,
		&usage.MessagesProcessedToday,
		&usage.CurrentQueueDepth,
		&usage.APIRequestsLastMinute,
		&usage.APIRequestsLastHour,
		&usage.ActiveWorkerThreads,
		&usage.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no resource usage found for gateway %d", rl.gatewayID)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get resource usage: %w", err)
	}

	return usage, nil
}

// CheckCPUQuota checks if the CPU usage exceeds the quota
// Requirement 9.1: THE النظام SHALL يدعم تحديد حصة CPU لكل بوابة مخصصة
func (rl *ResourceLimiter) CheckCPUQuota(ctx context.Context) (*QuotaCheckResult, error) {
	quota, err := rl.GetResourceQuota(ctx)
	if err != nil {
		return nil, err
	}

	usage, err := rl.GetLatestResourceUsage(ctx)
	if err != nil {
		return nil, err
	}

	// If no CPU quota is set, no limit
	if quota.CPUCores == nil {
		return &QuotaCheckResult{
			Exceeded:        false,
			CurrentUsage:    nil,
			Quota:           nil,
			UsagePercent:    0,
			ResourceType:    "CPU",
			EnforcementMode: quota.EnforcementMode,
		}, nil
	}

	// If no usage data, assume 0
	if usage.CPUUsagePercent == nil {
		return &QuotaCheckResult{
			Exceeded:        false,
			CurrentUsage:    0.0,
			Quota:           *quota.CPUCores,
			UsagePercent:    0,
			ResourceType:    "CPU",
			EnforcementMode: quota.EnforcementMode,
		}, nil
	}

	// Calculate usage percentage
	usagePercent := *usage.CPUUsagePercent
	exceeded := usagePercent >= 100.0

	return &QuotaCheckResult{
		Exceeded:        exceeded,
		CurrentUsage:    usagePercent,
		Quota:           *quota.CPUCores,
		UsagePercent:    usagePercent,
		ResourceType:    "CPU",
		EnforcementMode: quota.EnforcementMode,
	}, nil
}

// CheckMemoryQuota checks if the memory usage exceeds the quota
// Requirement 9.2: THE النظام SHALL يدعم تحديد حصة RAM لكل بوابة مخصصة
func (rl *ResourceLimiter) CheckMemoryQuota(ctx context.Context) (*QuotaCheckResult, error) {
	quota, err := rl.GetResourceQuota(ctx)
	if err != nil {
		return nil, err
	}

	usage, err := rl.GetLatestResourceUsage(ctx)
	if err != nil {
		return nil, err
	}

	// If no memory quota is set, no limit
	if quota.MemoryMB == nil {
		return &QuotaCheckResult{
			Exceeded:        false,
			CurrentUsage:    nil,
			Quota:           nil,
			UsagePercent:    0,
			ResourceType:    "Memory",
			EnforcementMode: quota.EnforcementMode,
		}, nil
	}

	// If no usage data, assume 0
	if usage.MemoryUsageMB == nil {
		return &QuotaCheckResult{
			Exceeded:        false,
			CurrentUsage:    0,
			Quota:           *quota.MemoryMB,
			UsagePercent:    0,
			ResourceType:    "Memory",
			EnforcementMode: quota.EnforcementMode,
		}, nil
	}

	// Calculate usage percentage
	usagePercent := (float64(*usage.MemoryUsageMB) / float64(*quota.MemoryMB)) * 100.0
	exceeded := *usage.MemoryUsageMB >= *quota.MemoryMB

	return &QuotaCheckResult{
		Exceeded:        exceeded,
		CurrentUsage:    *usage.MemoryUsageMB,
		Quota:           *quota.MemoryMB,
		UsagePercent:    usagePercent,
		ResourceType:    "Memory",
		EnforcementMode: quota.EnforcementMode,
	}, nil
}

// CheckBandwidthQuota checks if the bandwidth usage exceeds the quota
// Requirement 9.3: THE النظام SHALL يدعم تحديد حصة Bandwidth لكل بوابة مخصصة
func (rl *ResourceLimiter) CheckBandwidthQuota(ctx context.Context) (*QuotaCheckResult, error) {
	quota, err := rl.GetResourceQuota(ctx)
	if err != nil {
		return nil, err
	}

	usage, err := rl.GetLatestResourceUsage(ctx)
	if err != nil {
		return nil, err
	}

	// If no bandwidth quota is set, no limit
	if quota.BandwidthMbps == nil {
		return &QuotaCheckResult{
			Exceeded:        false,
			CurrentUsage:    nil,
			Quota:           nil,
			UsagePercent:    0,
			ResourceType:    "Bandwidth",
			EnforcementMode: quota.EnforcementMode,
		}, nil
	}

	// If no usage data, assume 0
	if usage.BandwidthUsageMbps == nil {
		return &QuotaCheckResult{
			Exceeded:        false,
			CurrentUsage:    0.0,
			Quota:           *quota.BandwidthMbps,
			UsagePercent:    0,
			ResourceType:    "Bandwidth",
			EnforcementMode: quota.EnforcementMode,
		}, nil
	}

	// Calculate usage percentage
	usagePercent := (*usage.BandwidthUsageMbps / float64(*quota.BandwidthMbps)) * 100.0
	exceeded := *usage.BandwidthUsageMbps >= float64(*quota.BandwidthMbps)

	return &QuotaCheckResult{
		Exceeded:        exceeded,
		CurrentUsage:    *usage.BandwidthUsageMbps,
		Quota:           *quota.BandwidthMbps,
		UsagePercent:    usagePercent,
		ResourceType:    "Bandwidth",
		EnforcementMode: quota.EnforcementMode,
	}, nil
}

// CheckAllQuotas checks all resource quotas and returns results
func (rl *ResourceLimiter) CheckAllQuotas(ctx context.Context) (map[string]*QuotaCheckResult, error) {
	results := make(map[string]*QuotaCheckResult)

	// Check CPU
	cpuResult, err := rl.CheckCPUQuota(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check CPU quota: %w", err)
	}
	results["cpu"] = cpuResult

	// Check Memory
	memoryResult, err := rl.CheckMemoryQuota(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check memory quota: %w", err)
	}
	results["memory"] = memoryResult

	// Check Bandwidth
	bandwidthResult, err := rl.CheckBandwidthQuota(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check bandwidth quota: %w", err)
	}
	results["bandwidth"] = bandwidthResult

	return results, nil
}

// IsQuotaExceeded checks if any quota is exceeded
func (rl *ResourceLimiter) IsQuotaExceeded(ctx context.Context) (bool, []string, error) {
	results, err := rl.CheckAllQuotas(ctx)
	if err != nil {
		return false, nil, err
	}

	var exceededResources []string
	for resourceType, result := range results {
		if result.Exceeded {
			exceededResources = append(exceededResources, resourceType)
		}
	}

	return len(exceededResources) > 0, exceededResources, nil
}

// RateLimitConfig holds the rate limit configuration for a gateway
type RateLimitConfig struct {
	APIRatePerMinute   int // API requests per minute
	MessagesPerHour    int // Messages per hour
	QueriesPerSecond   int // Database queries per second
}

// TenantRateLimiter manages rate limiting for a dedicated gateway using Token Bucket Algorithm
// Requirement 9.8: THE النظام SHALL يدعم تحديد حد API Rate Limit لكل بوابة
// Requirement 27.10: THE النظام SHALL يحسّن الاستعلامات باستخدام Indexes مناسبة
type TenantRateLimiter struct {
	tenantID  uuid.UUID
	gatewayID int
	limiters  map[string]*rate.Limiter
	mu        sync.RWMutex
}

// NewTenantRateLimiter creates a new TenantRateLimiter instance with Token Bucket Algorithm
// The Token Bucket Algorithm allows for burst traffic while maintaining average rate limits
func NewTenantRateLimiter(tenantID uuid.UUID, gatewayID int, config *RateLimitConfig) *TenantRateLimiter {
	return &TenantRateLimiter{
		tenantID:  tenantID,
		gatewayID: gatewayID,
		limiters: map[string]*rate.Limiter{
			// API Rate Limiter: allows APIRatePerMinute requests per minute
			// Token Bucket: refills at rate of APIRatePerMinute/60 tokens per second
			// Burst: allows up to APIRatePerMinute tokens to be consumed at once
			"api": rate.NewLimiter(rate.Limit(float64(config.APIRatePerMinute)/60.0), config.APIRatePerMinute),
			
			// Messages Rate Limiter: allows MessagesPerHour messages per hour
			// Token Bucket: refills at rate of MessagesPerHour/3600 tokens per second
			// Burst: allows up to MessagesPerHour/10 tokens to be consumed at once (10% burst)
			"messages": rate.NewLimiter(rate.Limit(float64(config.MessagesPerHour)/3600.0), config.MessagesPerHour/10),
			
			// Queries Rate Limiter: allows QueriesPerSecond queries per second
			// Token Bucket: refills at rate of QueriesPerSecond tokens per second
			// Burst: allows up to QueriesPerSecond*2 tokens to be consumed at once (2x burst)
			"queries": rate.NewLimiter(rate.Limit(config.QueriesPerSecond), config.QueriesPerSecond*2),
		},
	}
}

// AllowAPI checks if an API request is allowed under the rate limit
// Returns error if rate limit is exceeded
// Requirement 9.8: دعم تحديد حد API Rate Limit لكل بوابة
func (trl *TenantRateLimiter) AllowAPI(ctx context.Context) error {
	trl.mu.RLock()
	limiter := trl.limiters["api"]
	trl.mu.RUnlock()

	if !limiter.Allow() {
		return fmt.Errorf("API rate limit exceeded for tenant %s gateway %d", trl.tenantID, trl.gatewayID)
	}
	return nil
}

// AllowMessage checks if a message send is allowed under the rate limit
// Returns error if rate limit is exceeded
// Requirement 9.8: دعم تحديد حد API Rate Limit لكل بوابة
func (trl *TenantRateLimiter) AllowMessage(ctx context.Context) error {
	trl.mu.RLock()
	limiter := trl.limiters["messages"]
	trl.mu.RUnlock()

	if !limiter.Allow() {
		return fmt.Errorf("message rate limit exceeded for tenant %s gateway %d", trl.tenantID, trl.gatewayID)
	}
	return nil
}

// AllowQuery checks if a database query is allowed under the rate limit
// Returns error if rate limit is exceeded
// Requirement 9.8: دعم تحديد حد API Rate Limit لكل بوابة
func (trl *TenantRateLimiter) AllowQuery(ctx context.Context) error {
	trl.mu.RLock()
	limiter := trl.limiters["queries"]
	trl.mu.RUnlock()

	if !limiter.Allow() {
		return fmt.Errorf("query rate limit exceeded for tenant %s gateway %d", trl.tenantID, trl.gatewayID)
	}
	return nil
}

// AllowAPIWithWait waits until an API request is allowed or context is cancelled
// This is useful for graceful degradation instead of immediate rejection
func (trl *TenantRateLimiter) AllowAPIWithWait(ctx context.Context) error {
	trl.mu.RLock()
	limiter := trl.limiters["api"]
	trl.mu.RUnlock()

	if err := limiter.Wait(ctx); err != nil {
		return fmt.Errorf("API rate limit wait failed for tenant %s gateway %d: %w", trl.tenantID, trl.gatewayID, err)
	}
	return nil
}

// AllowMessageWithWait waits until a message send is allowed or context is cancelled
func (trl *TenantRateLimiter) AllowMessageWithWait(ctx context.Context) error {
	trl.mu.RLock()
	limiter := trl.limiters["messages"]
	trl.mu.RUnlock()

	if err := limiter.Wait(ctx); err != nil {
		return fmt.Errorf("message rate limit wait failed for tenant %s gateway %d: %w", trl.tenantID, trl.gatewayID, err)
	}
	return nil
}

// AllowQueryWithWait waits until a database query is allowed or context is cancelled
func (trl *TenantRateLimiter) AllowQueryWithWait(ctx context.Context) error {
	trl.mu.RLock()
	limiter := trl.limiters["queries"]
	trl.mu.RUnlock()

	if err := limiter.Wait(ctx); err != nil {
		return fmt.Errorf("query rate limit wait failed for tenant %s gateway %d: %w", trl.tenantID, trl.gatewayID, err)
	}
	return nil
}

// GetRateLimitStatus returns the current status of all rate limiters
func (trl *TenantRateLimiter) GetRateLimitStatus() map[string]interface{} {
	trl.mu.RLock()
	defer trl.mu.RUnlock()

	status := make(map[string]interface{})
	
	for name, limiter := range trl.limiters {
		status[name] = map[string]interface{}{
			"limit":     limiter.Limit(),
			"burst":     limiter.Burst(),
			"available": limiter.Tokens(),
		}
	}
	
	return status
}

// UpdateRateLimits updates the rate limits dynamically
// This allows changing rate limits without recreating the limiter
func (trl *TenantRateLimiter) UpdateRateLimits(config *RateLimitConfig) {
	trl.mu.Lock()
	defer trl.mu.Unlock()

	// Update API rate limiter
	trl.limiters["api"].SetLimit(rate.Limit(float64(config.APIRatePerMinute) / 60.0))
	trl.limiters["api"].SetBurst(config.APIRatePerMinute)

	// Update Messages rate limiter
	trl.limiters["messages"].SetLimit(rate.Limit(float64(config.MessagesPerHour) / 3600.0))
	trl.limiters["messages"].SetBurst(config.MessagesPerHour / 10)

	// Update Queries rate limiter
	trl.limiters["queries"].SetLimit(rate.Limit(config.QueriesPerSecond))
	trl.limiters["queries"].SetBurst(config.QueriesPerSecond * 2)
}

// LoadRateLimitConfigFromDB loads the rate limit configuration from the database
func (rl *ResourceLimiter) LoadRateLimitConfigFromDB(ctx context.Context) (*RateLimitConfig, error) {
	quota, err := rl.GetResourceQuota(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource quota: %w", err)
	}

	config := &RateLimitConfig{
		APIRatePerMinute: 300,    // Default: 300 requests per minute
		MessagesPerHour:  50000,  // Default: 50,000 messages per hour
		QueriesPerSecond: 200,    // Default: 200 queries per second
	}

	// Override with database values if set
	if quota.APIRateLimitPerMinute != nil {
		config.APIRatePerMinute = *quota.APIRateLimitPerMinute
	}
	if quota.MaxMessagesPerHour != nil {
		config.MessagesPerHour = *quota.MaxMessagesPerHour
	}
	if quota.MaxQueriesPerSecond != nil {
		config.QueriesPerSecond = *quota.MaxQueriesPerSecond
	}

	return config, nil
}

// CreateTenantRateLimiter creates a new TenantRateLimiter with configuration from database
func (rl *ResourceLimiter) CreateTenantRateLimiter(ctx context.Context) (*TenantRateLimiter, error) {
	config, err := rl.LoadRateLimitConfigFromDB(ctx)
	if err != nil {
		return nil, err
	}

	return NewTenantRateLimiter(rl.tenantID, rl.gatewayID, config), nil
}

// EnforcementPolicy represents the enforcement policy for resource quotas
type EnforcementPolicy string

const (
	// EnforcementSoft sends warnings only when quota is exceeded
	// Requirement 9.11: IF تجاوزت بوابة حدها الموردي، THEN THE النظام SHALL يطبق سياسة التحكم (Soft أو Hard Limit)
	EnforcementSoft EnforcementPolicy = "soft"
	
	// EnforcementHard rejects requests when quota is exceeded
	// Requirement 9.12: WHERE السياسة Hard، THE النظام SHALL يرفض الطلبات الإضافية حتى تنخفض الموارد
	EnforcementHard EnforcementPolicy = "hard"
)

// EnforcementResult represents the result of enforcing a policy
type EnforcementResult struct {
	Allowed         bool
	Policy          EnforcementPolicy
	ResourceType    string
	CurrentUsage    interface{}
	Quota           interface{}
	UsagePercent    float64
	WarningMessage  string
	RejectionReason string
}

// EnforceQuota enforces the resource quota based on the enforcement policy
// Requirement 9.11: تطبيق Soft Limit (تحذير فقط)
// Requirement 9.12: تطبيق Hard Limit (رفض الطلبات)
func (rl *ResourceLimiter) EnforceQuota(ctx context.Context, resourceType string) (*EnforcementResult, error) {
	var result *QuotaCheckResult
	var err error

	// Check the specific resource quota
	switch resourceType {
	case "cpu":
		result, err = rl.CheckCPUQuota(ctx)
	case "memory":
		result, err = rl.CheckMemoryQuota(ctx)
	case "bandwidth":
		result, err = rl.CheckBandwidthQuota(ctx)
	default:
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to check %s quota: %w", resourceType, err)
	}

	enforcement := &EnforcementResult{
		Policy:       EnforcementPolicy(result.EnforcementMode),
		ResourceType: result.ResourceType,
		CurrentUsage: result.CurrentUsage,
		Quota:        result.Quota,
		UsagePercent: result.UsagePercent,
	}

	// If quota is not exceeded, allow the request
	if !result.Exceeded {
		enforcement.Allowed = true
		
		// Send warning if approaching limit (90% or more)
		if result.UsagePercent >= 90.0 {
			enforcement.WarningMessage = fmt.Sprintf(
				"%s usage is at %.2f%% (approaching limit)",
				result.ResourceType,
				result.UsagePercent,
			)
		}
		
		return enforcement, nil
	}

	// Quota is exceeded - apply enforcement policy
	if enforcement.Policy == EnforcementSoft {
		// Soft limit: Allow request but send warning
		// Requirement 9.11: Soft Limit (تحذير فقط)
		enforcement.Allowed = true
		enforcement.WarningMessage = fmt.Sprintf(
			"WARNING: %s quota exceeded - Current: %v, Quota: %v (%.2f%%)",
			result.ResourceType,
			result.CurrentUsage,
			result.Quota,
			result.UsagePercent,
		)
	} else {
		// Hard limit: Reject request
		// Requirement 9.12: Hard Limit (رفض الطلبات)
		enforcement.Allowed = false
		enforcement.RejectionReason = fmt.Sprintf(
			"%s quota exceeded - Current: %v, Quota: %v (%.2f%%). Request rejected.",
			result.ResourceType,
			result.CurrentUsage,
			result.Quota,
			result.UsagePercent,
		)
	}

	return enforcement, nil
}

// EnforceAllQuotas enforces all resource quotas and returns the results
// Returns the most restrictive result (if any hard limit is exceeded, the request is rejected)
func (rl *ResourceLimiter) EnforceAllQuotas(ctx context.Context) (*EnforcementResult, error) {
	resourceTypes := []string{"cpu", "memory", "bandwidth"}
	
	var warnings []string
	var rejections []string
	var mostRestrictive *EnforcementResult

	for _, resourceType := range resourceTypes {
		result, err := rl.EnforceQuota(ctx, resourceType)
		if err != nil {
			return nil, fmt.Errorf("failed to enforce %s quota: %w", resourceType, err)
		}

		// Collect warnings
		if result.WarningMessage != "" {
			warnings = append(warnings, result.WarningMessage)
		}

		// If any hard limit is exceeded, reject the request
		if !result.Allowed {
			rejections = append(rejections, result.RejectionReason)
			if mostRestrictive == nil || mostRestrictive.Allowed {
				mostRestrictive = result
			}
		}
	}

	// If any resource has a hard limit exceeded, return rejection
	if len(rejections) > 0 {
		return &EnforcementResult{
			Allowed:         false,
			Policy:          EnforcementHard,
			ResourceType:    "multiple",
			RejectionReason: fmt.Sprintf("Multiple quotas exceeded: %v", rejections),
		}, nil
	}

	// All checks passed, but may have warnings
	return &EnforcementResult{
		Allowed:        true,
		Policy:         EnforcementSoft,
		ResourceType:   "all",
		WarningMessage: fmt.Sprintf("Warnings: %v", warnings),
	}, nil
}

// ShouldAllowRequest checks if a request should be allowed based on all resource quotas
// This is a convenience method that returns a simple boolean and error message
func (rl *ResourceLimiter) ShouldAllowRequest(ctx context.Context) (bool, string, error) {
	result, err := rl.EnforceAllQuotas(ctx)
	if err != nil {
		return false, "", err
	}

	if !result.Allowed {
		return false, result.RejectionReason, nil
	}

	return true, result.WarningMessage, nil
}
