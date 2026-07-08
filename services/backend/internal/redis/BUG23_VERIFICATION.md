# Bug 23: Stale Channel Cache - Verification

## Status: ⚠️ PARTIALLY COVERED BY BUG 13

## Bug 23 Requirements

### Current Behavior (Bug 23 - Requirement 1.23)
WHEN channel cache uses 5-minute TTL THEN the system may serve stale channel data, causing routing to offline channels

### Expected Behavior (Bug 23 - Requirement 2.23)
WHEN channel cache is configured THEN the system SHALL use shorter TTL (e.g., 1 minute) to reduce stale data while maintaining performance

## Verification Against Bug 13 Fix

### Bug 13 Analysis
Bug 13 addressed inconsistent TTL values for Balance and Usage caches:
- **Original Bug**: Balance (60s) vs Usage (30s) inconsistency
- **Fix Applied**: Changed Usage TTL to 60s for consistency
- **File**: `services/backend/internal/redis/ttl.go`

### Channel Cache Current State

**Current Implementation in `channel_cache.go`**:
```go
func NewChannelCache(client *Client) *ChannelCache {
    return &ChannelCache{
        client: client,
        ttl:    5 * time.Minute, // Cache for 5 minutes
    }
}
```

### Requirements Mapping

| Bug 23 Requirement | Current State | Status |
|-------------------|---------------|--------|
| Reduce TTL from 5 min to 1 min | ❌ Still 5 minutes | ❌ NOT COVERED |
| Active invalidation on offline | ❌ Not implemented | ❌ NOT COVERED |
| Health check verification | ❌ Not implemented | ❌ NOT COVERED |

## Gap Analysis

### What Bug 13 Fixed
✅ Balance and Usage cache TTL consistency (both 1 minute)
✅ Cache coherence for financial data
✅ Comprehensive preservation tests

### What Bug 23 Requires (NOT covered by Bug 13)
❌ **Channel cache TTL reduction**: Still 5 minutes, needs to be 1 minute
❌ **Active invalidation**: No mechanism to invalidate when channel goes offline
❌ **Health check verification**: No periodic verification of cached channel status

## Additional Work Required

### Task 23.2.1: Add Active Invalidation
**Requirement**: Invalidate cache when channel goes offline

**Implementation Needed**:
1. Add `InvalidateChannel()` method to `ChannelCache`
2. Call invalidation when channel status changes to offline
3. Integrate with channel status monitoring

### Task 23.2.2: Add Health Check Verification
**Requirement**: Periodically verify cached channel is online

**Implementation Needed**:
1. Add periodic health check for cached channels
2. Invalidate cache if channel is found offline
3. Implement background verification routine

### Proposed Fix for TTL
**File**: `services/backend/internal/redis/channel_cache.go`
**Line**: 28

**Change**:
```go
// Before
ttl:    5 * time.Minute, // Cache for 5 minutes

// After
ttl:    1 * time.Minute, // Cache for 1 minute - reduce stale data
```

## Conclusion

⚠️ **Bug 23 is PARTIALLY COVERED by Bug 13**

**What's Covered**:
- ✅ Bug 13 established pattern for consistent TTL values
- ✅ Bug 13 provided comprehensive test framework

**What's NOT Covered**:
- ❌ Channel cache TTL still 5 minutes (needs reduction to 1 minute)
- ❌ No active invalidation mechanism
- ❌ No health check verification

**Next Steps**:
1. ✅ Task 23.1: Verify Bug 13 covers partial requirements
2. ⏭️ Task 23.2.1: Implement active invalidation
3. ⏭️ Task 23.2.2: Implement health check verification

## References
- Bug 13 Fix Summary: `services/backend/internal/redis/BUG13_FIX_SUMMARY.md`
- Channel Cache Implementation: `services/backend/internal/redis/channel_cache.go`
- Bug 23 Requirements: `.kiro/specs/system-critical-bugfixes/bugfix.md` (Requirement 2.23)
