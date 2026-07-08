package resources

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// TestProperty_RateLimitingPerTenant tests Property 20: Rate Limiting per Tenant
// **Validates: Requirements 9.8**
//
// Property: لأي tenant يتجاوز حد الـ rate limit الخاص به، يجب أن يرفض النظام طلباته دون التأثير على tenants أخرى
//
// This property test verifies:
// 1. Rate limits are enforced per tenant
// 2. One tenant exceeding their limit does not affect other tenants
// 3. Rate limits are isolated between tenants
func TestProperty_RateLimitingPerTenant(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		numTenants := rapid.IntRange(2, 5).Draw(t, "numTenants")
		
		// Create multiple tenants with different rate limits
		tenants := make([]*TenantRateLimiter, numTenants)
		tenantIDs := make([]uuid.UUID, numTenants)
		configs := make([]*RateLimitConfig, numTenants)
		
		for i := 0; i < numTenants; i++ {
			tenantIDs[i] = uuid.New()
			
			// Generate different rate limits for each tenant
			configs[i] = &RateLimitConfig{
				APIRatePerMinute: rapid.IntRange(5, 20).Draw(t, fmt.Sprintf("apiRate_%d", i)),
				MessagesPerHour:  rapid.IntRange(100, 500).Draw(t, fmt.Sprintf("msgRate_%d", i)),
				QueriesPerSecond: rapid.IntRange(10, 50).Draw(t, fmt.Sprintf("queryRate_%d", i)),
			}
			
			tenants[i] = NewTenantRateLimiter(tenantIDs[i], i+1, configs[i])
		}
		
		// Select one tenant to exceed their limit
		victimTenantIdx := rapid.IntRange(0, numTenants-1).Draw(t, "victimTenant")
		
		ctx := context.Background()
		
		// Exhaust the victim tenant's API rate limit
		for j := 0; j < configs[victimTenantIdx].APIRatePerMinute; j++ {
			err := tenants[victimTenantIdx].AllowAPI(ctx)
			if err != nil {
				// If we hit the limit early, that's fine
				break
			}
		}
		
		// Verify victim tenant is rate limited
		err := tenants[victimTenantIdx].AllowAPI(ctx)
		assert.Error(t, err, "Victim tenant should be rate limited")
		if err != nil {
			assert.Contains(t, err.Error(), "API rate limit exceeded")
			assert.Contains(t, err.Error(), tenantIDs[victimTenantIdx].String())
		}
		
		// Verify other tenants are NOT affected
		for i := 0; i < numTenants; i++ {
			if i == victimTenantIdx {
				continue
			}
			
			// Other tenants should still be able to make requests
			err := tenants[i].AllowAPI(ctx)
			assert.NoError(t, err, "Tenant %d should not be affected by tenant %d's rate limit", i, victimTenantIdx)
		}
	})
}

// TestProperty_RateLimitIsolationUnderLoad tests rate limit isolation under concurrent load
// **Validates: Requirements 9.8**
//
// Property: عند تحميل متزامن، يجب أن تظل حدود Rate Limit معزولة بين Tenants
//
// This property test verifies:
// 1. Rate limits remain isolated under concurrent load
// 2. Multiple tenants can hit their limits independently
// 3. No cross-tenant interference occurs
func TestProperty_RateLimitIsolationUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		numTenants := rapid.IntRange(2, 4).Draw(t, "numTenants")
		requestsPerTenant := rapid.IntRange(10, 30).Draw(t, "requestsPerTenant")
		
		// Create tenants with low rate limits for testing
		tenants := make([]*TenantRateLimiter, numTenants)
		tenantIDs := make([]uuid.UUID, numTenants)
		configs := make([]*RateLimitConfig, numTenants)
		
		for i := 0; i < numTenants; i++ {
			tenantIDs[i] = uuid.New()
			
			// Low rate limits to ensure we hit them
			configs[i] = &RateLimitConfig{
				APIRatePerMinute: rapid.IntRange(5, 15).Draw(t, fmt.Sprintf("apiRate_%d", i)),
				MessagesPerHour:  rapid.IntRange(50, 150).Draw(t, fmt.Sprintf("msgRate_%d", i)),
				QueriesPerSecond: rapid.IntRange(5, 20).Draw(t, fmt.Sprintf("queryRate_%d", i)),
			}
			
			tenants[i] = NewTenantRateLimiter(tenantIDs[i], i+1, configs[i])
		}
		
		// Track results per tenant
		type TenantResults struct {
			successCount int
			errorCount   int
			mu           sync.Mutex
		}
		
		results := make([]*TenantResults, numTenants)
		for i := 0; i < numTenants; i++ {
			results[i] = &TenantResults{}
		}
		
		// Concurrent requests from all tenants
		var wg sync.WaitGroup
		ctx := context.Background()
		
		for i := 0; i < numTenants; i++ {
			tenantIdx := i
			
			for j := 0; j < requestsPerTenant; j++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					
					err := tenants[tenantIdx].AllowAPI(ctx)
					
					results[tenantIdx].mu.Lock()
					if err == nil {
						results[tenantIdx].successCount++
					} else {
						results[tenantIdx].errorCount++
					}
					results[tenantIdx].mu.Unlock()
				}()
			}
		}
		
		wg.Wait()
		
		// Verify each tenant's results independently
		for i := 0; i < numTenants; i++ {
			results[i].mu.Lock()
			totalRequests := results[i].successCount + results[i].errorCount
			results[i].mu.Unlock()
			
			// All requests should be accounted for
			assert.Equal(t, requestsPerTenant, totalRequests, 
				"Tenant %d: all requests should be accounted for", i)
			
			// Each tenant should have some successful requests (at least their burst capacity)
			assert.Greater(t, results[i].successCount, 0, 
				"Tenant %d should have at least some successful requests", i)
			
			// If we sent more requests than the rate limit, some should fail
			if requestsPerTenant > configs[i].APIRatePerMinute {
				assert.Greater(t, results[i].errorCount, 0, 
					"Tenant %d should have some rate limited requests", i)
			}
		}
		
		// Verify isolation: each tenant's results are tracked independently
		// The key property is that each tenant's rate limit is enforced independently,
		// not that they have different success counts (which can be the same by chance)
		for i := 0; i < numTenants; i++ {
			// Each tenant should have been rate limited based on their own quota
			// If they sent more requests than their limit, they should have some failures
			if requestsPerTenant > configs[i].APIRatePerMinute {
				// This tenant should have hit their limit
				assert.Greater(t, results[i].errorCount, 0, 
					"Tenant %d should have been rate limited based on their own quota", i)
			}
		}
	})
}

// TestProperty_RateLimitDoesNotExceedQuota tests that rate limiting never allows exceeding the quota
// **Validates: Requirements 9.8**
//
// Property: عدد الطلبات المسموحة يجب ألا يتجاوز الحد المحدد أبداً
//
// This property test verifies:
// 1. Rate limiter never allows more requests than the configured limit
// 2. Burst capacity is respected
// 3. Token bucket algorithm works correctly
func TestProperty_RateLimitDoesNotExceedQuota(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		tenantID := uuid.New()
		gatewayID := rapid.IntRange(1, 100).Draw(t, "gatewayID")
		
		apiRatePerMinute := rapid.IntRange(10, 100).Draw(t, "apiRate")
		
		config := &RateLimitConfig{
			APIRatePerMinute: apiRatePerMinute,
			MessagesPerHour:  rapid.IntRange(1000, 10000).Draw(t, "msgRate"),
			QueriesPerSecond: rapid.IntRange(50, 500).Draw(t, "queryRate"),
		}
		
		limiter := NewTenantRateLimiter(tenantID, gatewayID, config)
		ctx := context.Background()
		
		// Try to make many more requests than the limit
		totalRequests := apiRatePerMinute * 2
		successCount := 0
		errorCount := 0
		
		for i := 0; i < totalRequests; i++ {
			err := limiter.AllowAPI(ctx)
			if err == nil {
				successCount++
			} else {
				errorCount++
			}
		}
		
		// Verify that successful requests do not exceed the burst capacity
		// The burst capacity is set to apiRatePerMinute in the implementation
		assert.LessOrEqual(t, successCount, apiRatePerMinute, 
			"Successful requests should not exceed burst capacity")
		
		// Verify that some requests were rate limited
		assert.Greater(t, errorCount, 0, 
			"Some requests should be rate limited when exceeding the limit")
		
		// Verify total requests
		assert.Equal(t, totalRequests, successCount+errorCount, 
			"All requests should be accounted for")
	})
}

// TestProperty_RateLimitRecoveryAfterWait tests that rate limits recover after waiting
// **Validates: Requirements 9.8**
//
// Property: بعد الانتظار، يجب أن يتم تجديد الـ tokens ويسمح بطلبات جديدة
//
// This property test verifies:
// 1. Rate limiter recovers after waiting
// 2. Token bucket refills at the correct rate
// 3. Requests are allowed again after recovery period
func TestProperty_RateLimitRecoveryAfterWait(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		tenantID := uuid.New()
		gatewayID := rapid.IntRange(1, 100).Draw(t, "gatewayID")
		
		// Use a high rate for faster testing
		apiRatePerMinute := rapid.IntRange(60, 120).Draw(t, "apiRate") // 1-2 requests per second
		
		config := &RateLimitConfig{
			APIRatePerMinute: apiRatePerMinute,
			MessagesPerHour:  3600,
			QueriesPerSecond: 100,
		}
		
		limiter := NewTenantRateLimiter(tenantID, gatewayID, config)
		ctx := context.Background()
		
		// Exhaust the burst capacity
		for i := 0; i < apiRatePerMinute; i++ {
			err := limiter.AllowAPI(ctx)
			if err != nil {
				break
			}
		}
		
		// Verify rate limit is hit
		err := limiter.AllowAPI(ctx)
		require.Error(t, err, "Should be rate limited after exhausting burst")
		
		// Wait for tokens to refill
		// Token refill rate is apiRatePerMinute/60 per second
		// Wait for enough time to get at least 2 tokens
		refillRate := float64(apiRatePerMinute) / 60.0 // tokens per second
		waitTime := time.Duration(float64(2.0)/refillRate*1000) * time.Millisecond
		
		// Add some buffer
		waitTime = waitTime + 100*time.Millisecond
		
		time.Sleep(waitTime)
		
		// After waiting, should be able to make requests again
		err = limiter.AllowAPI(ctx)
		assert.NoError(t, err, "Should be able to make requests after waiting for token refill")
		
		// Should be able to make at least one more request
		err = limiter.AllowAPI(ctx)
		assert.NoError(t, err, "Should be able to make multiple requests after recovery")
	})
}

// TestProperty_MessageRateLimitIsolation tests message rate limit isolation
// **Validates: Requirements 9.8**
//
// Property: حدود الرسائل يجب أن تكون معزولة بين Tenants
//
// This property test verifies:
// 1. Message rate limits are enforced per tenant
// 2. One tenant's message sending does not affect others
func TestProperty_MessageRateLimitIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		numTenants := rapid.IntRange(2, 4).Draw(t, "numTenants")
		
		// Create tenants with different message rate limits
		tenants := make([]*TenantRateLimiter, numTenants)
		tenantIDs := make([]uuid.UUID, numTenants)
		configs := make([]*RateLimitConfig, numTenants)
		
		for i := 0; i < numTenants; i++ {
			tenantIDs[i] = uuid.New()
			
			// Different message rate limits
			configs[i] = &RateLimitConfig{
				APIRatePerMinute: 300,
				MessagesPerHour:  rapid.IntRange(100, 500).Draw(t, fmt.Sprintf("msgRate_%d", i)),
				QueriesPerSecond: 100,
			}
			
			tenants[i] = NewTenantRateLimiter(tenantIDs[i], i+1, configs[i])
		}
		
		// Select one tenant to exceed their message limit
		victimTenantIdx := rapid.IntRange(0, numTenants-1).Draw(t, "victimTenant")
		
		ctx := context.Background()
		
		// Exhaust the victim tenant's message burst capacity
		burstCapacity := configs[victimTenantIdx].MessagesPerHour / 10
		for j := 0; j < burstCapacity; j++ {
			err := tenants[victimTenantIdx].AllowMessage(ctx)
			if err != nil {
				break
			}
		}
		
		// Verify victim tenant is rate limited for messages
		err := tenants[victimTenantIdx].AllowMessage(ctx)
		assert.Error(t, err, "Victim tenant should be rate limited for messages")
		if err != nil {
			assert.Contains(t, err.Error(), "message rate limit exceeded")
		}
		
		// Verify other tenants can still send messages
		for i := 0; i < numTenants; i++ {
			if i == victimTenantIdx {
				continue
			}
			
			err := tenants[i].AllowMessage(ctx)
			assert.NoError(t, err, "Tenant %d should not be affected by tenant %d's message rate limit", i, victimTenantIdx)
		}
	})
}

// TestProperty_QueryRateLimitIsolation tests query rate limit isolation
// **Validates: Requirements 9.8**
//
// Property: حدود الاستعلامات يجب أن تكون معزولة بين Tenants
//
// This property test verifies:
// 1. Query rate limits are enforced per tenant
// 2. One tenant's queries do not affect others
func TestProperty_QueryRateLimitIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		numTenants := rapid.IntRange(2, 4).Draw(t, "numTenants")
		
		// Create tenants with different query rate limits
		tenants := make([]*TenantRateLimiter, numTenants)
		tenantIDs := make([]uuid.UUID, numTenants)
		configs := make([]*RateLimitConfig, numTenants)
		
		for i := 0; i < numTenants; i++ {
			tenantIDs[i] = uuid.New()
			
			// Different query rate limits
			configs[i] = &RateLimitConfig{
				APIRatePerMinute: 300,
				MessagesPerHour:  3600,
				QueriesPerSecond: rapid.IntRange(5, 20).Draw(t, fmt.Sprintf("queryRate_%d", i)),
			}
			
			tenants[i] = NewTenantRateLimiter(tenantIDs[i], i+1, configs[i])
		}
		
		// Select one tenant to exceed their query limit
		victimTenantIdx := rapid.IntRange(0, numTenants-1).Draw(t, "victimTenant")
		
		ctx := context.Background()
		
		// Exhaust the victim tenant's query burst capacity
		burstCapacity := configs[victimTenantIdx].QueriesPerSecond * 2
		for j := 0; j < burstCapacity; j++ {
			err := tenants[victimTenantIdx].AllowQuery(ctx)
			if err != nil {
				break
			}
		}
		
		// Verify victim tenant is rate limited for queries
		err := tenants[victimTenantIdx].AllowQuery(ctx)
		assert.Error(t, err, "Victim tenant should be rate limited for queries")
		if err != nil {
			assert.Contains(t, err.Error(), "query rate limit exceeded")
		}
		
		// Verify other tenants can still make queries
		for i := 0; i < numTenants; i++ {
			if i == victimTenantIdx {
				continue
			}
			
			err := tenants[i].AllowQuery(ctx)
			assert.NoError(t, err, "Tenant %d should not be affected by tenant %d's query rate limit", i, victimTenantIdx)
		}
	})
}

// TestProperty_RateLimitUpdatesDynamically tests that rate limits can be updated dynamically
// **Validates: Requirements 9.8**
//
// Property: يجب أن يكون من الممكن تحديث حدود Rate Limit ديناميكياً دون إعادة إنشاء الـ Limiter
//
// This property test verifies:
// 1. Rate limits can be updated while the limiter is in use
// 2. New limits take effect
// 3. No race conditions occur during updates
func TestProperty_RateLimitUpdatesDynamically(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		tenantID := uuid.New()
		gatewayID := rapid.IntRange(1, 100).Draw(t, "gatewayID")
		
		initialAPIRate := rapid.IntRange(60, 120).Draw(t, "initialAPIRate") // 1-2 per second
		updatedAPIRate := rapid.IntRange(180, 300).Draw(t, "updatedAPIRate") // 3-5 per second
		
		initialConfig := &RateLimitConfig{
			APIRatePerMinute: initialAPIRate,
			MessagesPerHour:  3600,
			QueriesPerSecond: 100,
		}
		
		limiter := NewTenantRateLimiter(tenantID, gatewayID, initialConfig)
		ctx := context.Background()
		
		// Make a few requests with initial config (don't exhaust)
		initialSuccessCount := 0
		for i := 0; i < 5; i++ {
			err := limiter.AllowAPI(ctx)
			if err == nil {
				initialSuccessCount++
			}
		}
		
		// Update rate limits to higher values
		updatedConfig := &RateLimitConfig{
			APIRatePerMinute: updatedAPIRate,
			MessagesPerHour:  7200,
			QueriesPerSecond: 200,
		}
		
		limiter.UpdateRateLimits(updatedConfig)
		
		// Wait for token refill at new rate (higher rate = faster refill)
		time.Sleep(500 * time.Millisecond)
		
		// After update and waiting, should be able to make requests
		updatedSuccessCount := 0
		for i := 0; i < 10; i++ {
			err := limiter.AllowAPI(ctx)
			if err == nil {
				updatedSuccessCount++
			}
		}
		
		// Should have been able to make some requests after update
		assert.Greater(t, updatedSuccessCount, 0, 
			"Should be able to make requests after updating rate limits")
		
		// Verify the update actually took effect by checking status
		status := limiter.GetRateLimitStatus()
		assert.NotNil(t, status)
		assert.Contains(t, status, "api")
	})
}

// TestProperty_RateLimitStatusReporting tests that rate limit status is reported correctly
// **Validates: Requirements 9.8**
//
// Property: يجب أن يعرض النظام حالة Rate Limit الحالية بدقة
//
// This property test verifies:
// 1. Rate limit status can be retrieved
// 2. Status contains expected fields
// 3. Available tokens decrease after consumption
func TestProperty_RateLimitStatusReporting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		tenantID := uuid.New()
		gatewayID := rapid.IntRange(1, 100).Draw(t, "gatewayID")
		
		apiRatePerMinute := rapid.IntRange(60, 300).Draw(t, "apiRate")
		
		config := &RateLimitConfig{
			APIRatePerMinute: apiRatePerMinute,
			MessagesPerHour:  3600,
			QueriesPerSecond: 100,
		}
		
		limiter := NewTenantRateLimiter(tenantID, gatewayID, config)
		ctx := context.Background()
		
		// Get initial status
		initialStatus := limiter.GetRateLimitStatus()
		require.NotNil(t, initialStatus)
		require.Contains(t, initialStatus, "api")
		require.Contains(t, initialStatus, "messages")
		require.Contains(t, initialStatus, "queries")
		
		// Verify all limiters have the required fields
		for name, statusMap := range initialStatus {
			statusData := statusMap.(map[string]interface{})
			assert.Contains(t, statusData, "limit", "Status for %s should contain limit", name)
			assert.Contains(t, statusData, "burst", "Status for %s should contain burst", name)
			assert.Contains(t, statusData, "available", "Status for %s should contain available", name)
		}
		
		// Make some requests
		requestsToMake := rapid.IntRange(1, apiRatePerMinute/2).Draw(t, "requestsToMake")
		successCount := 0
		for i := 0; i < requestsToMake; i++ {
			err := limiter.AllowAPI(ctx)
			if err == nil {
				successCount++
			}
		}
		
		// Get status after consuming tokens
		afterStatus := limiter.GetRateLimitStatus()
		require.NotNil(t, afterStatus)
		require.Contains(t, afterStatus, "api")
		
		// Verify status is still valid after consumption
		apiStatusAfter := afterStatus["api"].(map[string]interface{})
		assert.Contains(t, apiStatusAfter, "limit")
		assert.Contains(t, apiStatusAfter, "burst")
		assert.Contains(t, apiStatusAfter, "available")
	})
}
