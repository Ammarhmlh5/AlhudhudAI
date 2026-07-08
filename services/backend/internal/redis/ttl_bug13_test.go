package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBug13_InconsistentCacheTTL_ExplorationTest tests the bug condition for Bug 13
//
// **Validates: Requirements 1.13, 2.13**
//
// Bug Condition: WHEN caching balance and usage data THEN the system uses inconsistent TTL values
// (Balance: 1 minute, Usage: 30 seconds), causing cache coherence issues
//
// Expected Behavior: The system SHALL use consistent TTL values (e.g., both 1 minute) to maintain cache coherence
//
// **CRITICAL**: This test MUST FAIL on unfixed code - failure confirms the bug exists
//
// Test Strategy:
// 1. Cache both balance and usage data at T=0
// 2. Wait 35 seconds (after usage expires, before balance expires)
// 3. Check cache state - usage should be expired, balance should still be cached
// 4. Verify this causes inconsistency (balance shows old value, usage is recalculated)
// 5. **EXPECTED OUTCOME**: Test FAILS (inconsistent TTL causes stale data)
func TestBug13_InconsistentCacheTTL_ExplorationTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Bug 13 exploration test in short mode")
	}

	// Setup
	ctx := context.Background()
	client, mr := setupBug13TestRedis(t)
	defer mr.Close()

	// Test tenant
	tenantID := uuid.New()

	// **STEP 1**: Cache balance and usage at T=0
	t.Log("=== STEP 1: Caching balance and usage at T=0 ===")
	
	// Get current TTL configuration
	ttlConfig := DefaultTTLConfig()
	balanceTTL := ttlConfig.Balance
	usageTTL := ttlConfig.Usage
	
	t.Logf("Current TTL configuration:")
	t.Logf("  Balance TTL: %v", balanceTTL)
	t.Logf("  Usage TTL: %v", usageTTL)

	// Cache balance (should use 1 minute TTL)
	balanceKey := fmt.Sprintf("%s:cache:balance:%s", AppNamespace, tenantID.String())
	balanceValue := "100.50" // 100.50 SAR
	err := client.Client.Set(ctx, balanceKey, balanceValue, balanceTTL).Err()
	require.NoError(t, err, "Failed to cache balance")
	t.Logf("Cached balance: %s with TTL %v", balanceValue, balanceTTL)

	// Cache usage (should use 30 seconds TTL)
	usageKey := fmt.Sprintf("%s:cache:usage:%s", AppNamespace, tenantID.String())
	usageData := map[string]interface{}{
		"messages_sent": 1000,
		"messages_used": 500,
		"last_updated":  time.Now().Unix(),
	}
	usageJSON, _ := json.Marshal(usageData)
	err = client.Client.Set(ctx, usageKey, usageJSON, usageTTL).Err()
	require.NoError(t, err, "Failed to cache usage")
	t.Logf("Cached usage: %s with TTL %v", string(usageJSON), usageTTL)

	// Verify both are cached
	balanceExists := client.Client.Exists(ctx, balanceKey).Val()
	usageExists := client.Client.Exists(ctx, usageKey).Val()
	assert.Equal(t, int64(1), balanceExists, "Balance should be cached")
	assert.Equal(t, int64(1), usageExists, "Usage should be cached")

	// Get actual TTLs from Redis
	balanceTTLActual := client.Client.TTL(ctx, balanceKey).Val()
	usageTTLActual := client.Client.TTL(ctx, usageKey).Val()
	t.Logf("Actual TTLs from Redis:")
	t.Logf("  Balance TTL: %v", balanceTTLActual)
	t.Logf("  Usage TTL: %v", usageTTLActual)

	// **BUG CONDITION VERIFICATION**: Check if TTLs are inconsistent
	t.Log("\n=== BUG CONDITION VERIFICATION ===")
	
	// The bug exists if:
	// 1. Balance TTL is 60 seconds (1 minute)
	// 2. Usage TTL is 30 seconds
	// 3. These are different values
	
	balanceTTLSeconds := int(balanceTTL.Seconds())
	usageTTLSeconds := int(usageTTL.Seconds())
	
	t.Logf("Balance TTL: %d seconds", balanceTTLSeconds)
	t.Logf("Usage TTL: %d seconds", usageTTLSeconds)
	
	if balanceTTLSeconds == 60 && usageTTLSeconds == 30 {
		t.Log("✅ Bug condition CONFIRMED: Balance uses 60s TTL, Usage uses 30s TTL")
		t.Log("This inconsistency will cause cache coherence issues")
	} else if balanceTTLSeconds == usageTTLSeconds {
		t.Log("❌ Bug condition NOT FOUND: TTLs are consistent")
		t.Logf("Both Balance and Usage use %d seconds TTL", balanceTTLSeconds)
		t.Log("This indicates the bug has already been fixed")
	} else {
		t.Logf("⚠️  Unexpected TTL configuration: Balance=%ds, Usage=%ds", balanceTTLSeconds, usageTTLSeconds)
	}

	// **STEP 2**: Wait 35 seconds (after usage expires, before balance expires)
	// For testing purposes, we'll use miniredis FastForward to simulate time passage
	t.Log("\n=== STEP 2: Simulating 35 seconds time passage ===")
	
	// Fast forward 35 seconds in miniredis
	mr.FastForward(35 * time.Second)
	t.Log("Fast-forwarded 35 seconds")

	// **STEP 3**: Check cache state
	t.Log("\n=== STEP 3: Checking cache state after 35 seconds ===")
	
	balanceExistsAfter := client.Client.Exists(ctx, balanceKey).Val()
	usageExistsAfter := client.Client.Exists(ctx, usageKey).Val()
	
	t.Logf("Balance exists after 35s: %v", balanceExistsAfter == 1)
	t.Logf("Usage exists after 35s: %v", usageExistsAfter == 1)

	// **STEP 4**: Verify inconsistency
	t.Log("\n=== STEP 4: Verifying cache coherence issue ===")
	
	// Expected behavior with bug:
	// - Balance: STILL CACHED (60s TTL, only 35s passed)
	// - Usage: EXPIRED (30s TTL, 35s passed)
	
	if balanceExistsAfter == 1 && usageExistsAfter == 0 {
		t.Log("✅ CACHE COHERENCE ISSUE DETECTED:")
		t.Log("   - Balance is still cached (old data)")
		t.Log("   - Usage has expired (would be recalculated from DB)")
		t.Log("   - This creates an inconsistent view of tenant data")
		t.Log("   - User sees old balance with new usage → CONFUSION")
		
		// Get the cached balance value
		cachedBalance, err := client.Client.Get(ctx, balanceKey).Result()
		require.NoError(t, err)
		t.Logf("   - Cached balance value: %s (potentially stale)", cachedBalance)
		
		t.Log("\n=== BUG 13 EXPLORATION TEST RESULT ===")
		t.Log("❌ TEST FAILED (as expected) - Bug 13 EXISTS")
		t.Log("The inconsistent TTL values cause cache coherence issues:")
		t.Logf("  - Balance TTL: %d seconds", balanceTTLSeconds)
		t.Logf("  - Usage TTL: %d seconds", usageTTLSeconds)
		t.Log("  - At T=35s: Balance cached, Usage expired → INCONSISTENT STATE")
		
		// This assertion SHOULD FAIL on unfixed code
		assert.Fail(t, "Cache coherence issue detected",
			"Balance and Usage have inconsistent TTLs (%ds vs %ds), causing cache coherence issues at T=35s",
			balanceTTLSeconds, usageTTLSeconds)
			
	} else if balanceExistsAfter == 1 && usageExistsAfter == 1 {
		t.Log("✅ BOTH CACHED: No coherence issue at T=35s")
		t.Log("This indicates consistent TTLs (both > 35s)")
		t.Log("\n=== BUG 13 EXPLORATION TEST RESULT ===")
		t.Log("✅ TEST PASSED - Bug 13 does NOT exist")
		t.Log("TTLs are consistent, no cache coherence issues detected")
		
	} else if balanceExistsAfter == 0 && usageExistsAfter == 0 {
		t.Log("✅ BOTH EXPIRED: No coherence issue (both expired)")
		t.Log("This indicates consistent TTLs (both < 35s)")
		t.Log("\n=== BUG 13 EXPLORATION TEST RESULT ===")
		t.Log("✅ TEST PASSED - Bug 13 does NOT exist")
		t.Log("TTLs are consistent, no cache coherence issues detected")
		
	} else {
		// balanceExistsAfter == 0 && usageExistsAfter == 1
		t.Log("⚠️  UNEXPECTED STATE: Usage cached, Balance expired")
		t.Log("This indicates Usage TTL > Balance TTL (opposite of bug)")
		t.Log("\n=== BUG 13 EXPLORATION TEST RESULT ===")
		t.Log("⚠️  UNEXPECTED CONFIGURATION")
		t.Logf("Usage TTL (%ds) > Balance TTL (%ds)", usageTTLSeconds, balanceTTLSeconds)
	}

	// **STEP 5**: Document the counterexample
	t.Log("\n=== COUNTEREXAMPLE DOCUMENTATION ===")
	t.Log("Scenario: Tenant caches balance and usage at same time")
	t.Logf("  - Tenant ID: %s", tenantID)
	t.Logf("  - Balance: %s SAR (TTL: %ds)", balanceValue, balanceTTLSeconds)
	t.Logf("  - Usage: %s (TTL: %ds)", string(usageJSON), usageTTLSeconds)
	t.Log("  - Time: T=0 (both cached)")
	t.Log("  - Time: T=35s (check cache state)")
	t.Logf("  - Result: Balance exists=%v, Usage exists=%v", balanceExistsAfter == 1, usageExistsAfter == 1)
	
	if balanceExistsAfter == 1 && usageExistsAfter == 0 {
		t.Log("\n💥 IMPACT:")
		t.Log("  - Application reads balance from cache (old value)")
		t.Log("  - Application reads usage from database (new value)")
		t.Log("  - User sees inconsistent financial data")
		t.Log("  - May cause incorrect billing or display issues")
	}
}

// TestBug13_VerifyTTLConfiguration verifies the current TTL configuration
//
// This test documents the current TTL values in the system
func TestBug13_VerifyTTLConfiguration(t *testing.T) {
	ttlConfig := DefaultTTLConfig()
	
	t.Log("=== Current TTL Configuration ===")
	t.Logf("Balance TTL: %v (%d seconds)", ttlConfig.Balance, int(ttlConfig.Balance.Seconds()))
	t.Logf("Usage TTL: %v (%d seconds)", ttlConfig.Usage, int(ttlConfig.Usage.Seconds()))
	t.Logf("Tariff TTL: %v", ttlConfig.Tariff)
	t.Logf("Package TTL: %v", ttlConfig.Package)
	t.Logf("Subscription TTL: %v", ttlConfig.Subscription)
	t.Logf("Stats TTL: %v", ttlConfig.Stats)
	t.Logf("Gateway TTL: %v", ttlConfig.Gateway)
	
	// Check for inconsistency
	balanceSeconds := int(ttlConfig.Balance.Seconds())
	usageSeconds := int(ttlConfig.Usage.Seconds())
	
	if balanceSeconds == 60 && usageSeconds == 30 {
		t.Log("\n❌ INCONSISTENCY DETECTED:")
		t.Log("Balance uses 60s TTL, Usage uses 30s TTL")
		t.Log("This will cause cache coherence issues")
		assert.Fail(t, "Inconsistent TTL configuration",
			"Balance (%ds) and Usage (%ds) have different TTLs",
			balanceSeconds, usageSeconds)
	} else if balanceSeconds == usageSeconds {
		t.Log("\n✅ CONSISTENT CONFIGURATION:")
		t.Logf("Both Balance and Usage use %d seconds TTL", balanceSeconds)
	} else {
		t.Log("\n⚠️  DIFFERENT TTLs:")
		t.Logf("Balance: %ds, Usage: %ds", balanceSeconds, usageSeconds)
	}
}

// TestBug13_CacheCoherenceScenario simulates a real-world scenario
//
// This test demonstrates how inconsistent TTLs affect user experience
func TestBug13_CacheCoherenceScenario(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping scenario test in short mode")
	}

	ctx := context.Background()
	client, mr := setupBug13TestRedis(t)
	defer mr.Close()

	tenantID := uuid.New()
	ttlConfig := DefaultTTLConfig()

	t.Log("=== SCENARIO: User checks balance and usage ===")
	
	// T=0: User makes a purchase, balance and usage are cached
	t.Log("\nT=0: User makes purchase")
	balanceKey := fmt.Sprintf("%s:cache:balance:%s", AppNamespace, tenantID.String())
	usageKey := fmt.Sprintf("%s:cache:usage:%s", AppNamespace, tenantID.String())
	
	client.Client.Set(ctx, balanceKey, "100.00", ttlConfig.Balance)
	usageData := map[string]int{"messages_sent": 1000}
	usageJSON, _ := json.Marshal(usageData)
	client.Client.Set(ctx, usageKey, usageJSON, ttlConfig.Usage)
	
	t.Log("  Balance: 100.00 SAR (cached)")
	t.Log("  Usage: 1000 messages (cached)")

	// T=35s: User checks dashboard
	t.Log("\nT=35s: User checks dashboard")
	mr.FastForward(35 * time.Second)
	
	balanceExists := client.Client.Exists(ctx, balanceKey).Val() == 1
	usageExists := client.Client.Exists(ctx, usageKey).Val() == 1
	
	t.Logf("  Balance in cache: %v", balanceExists)
	t.Logf("  Usage in cache: %v", usageExists)
	
	if balanceExists && !usageExists {
		t.Log("\n❌ PROBLEM:")
		t.Log("  - Dashboard shows balance from cache (100.00 SAR)")
		t.Log("  - Dashboard fetches usage from DB (might be 1500 messages now)")
		t.Log("  - User sees OLD balance with NEW usage")
		t.Log("  - This creates confusion and potential billing issues")
		
		assert.Fail(t, "Cache coherence issue in user scenario",
			"User sees inconsistent data due to different TTLs")
	} else {
		t.Log("\n✅ NO PROBLEM:")
		t.Log("  - Both values from same source (both cached or both from DB)")
		t.Log("  - User sees consistent data")
	}
}

// Helper function to setup test Redis client with miniredis
func setupBug13TestRedis(t *testing.T) (*Client, *miniredis.Miniredis) {
	mr := miniredis.RunT(t)
	
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return &Client{
		Client: client,
	}, mr
}
