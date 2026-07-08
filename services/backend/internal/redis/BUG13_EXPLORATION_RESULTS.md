# Bug 13 Exploration Test Results

## Test Execution Summary

**Date**: 2025-01-XX  
**Test File**: `services/backend/internal/redis/ttl_bug13_test.go`  
**Status**: ✅ **BUG CONFIRMED** - Test failed as expected, proving Bug 13 exists

---

## Bug Condition Verified

**Bug 13**: Inconsistent Cache TTL Values

The exploration test successfully confirmed that the system uses inconsistent TTL values for related cache data:

- **Balance TTL**: 60 seconds (1 minute)
- **Usage TTL**: 30 seconds
- **Result**: Cache coherence issues at T=35s

---

## Test Results

### Test 1: Main Exploration Test
**Test**: `TestBug13_InconsistentCacheTTL_ExplorationTest`  
**Result**: ❌ FAILED (as expected)

**Scenario**:
1. Cache balance and usage at T=0
2. Fast-forward 35 seconds
3. Check cache state

**Findings**:
- Balance: STILL CACHED (60s TTL, only 35s passed)
- Usage: EXPIRED (30s TTL, 35s passed)
- **Cache Coherence Issue**: Balance shows old value, Usage would be recalculated from DB

### Test 2: TTL Configuration Verification
**Test**: `TestBug13_VerifyTTLConfiguration`  
**Result**: ❌ FAILED (as expected)

**Current Configuration** (from `ttl.go`):
```go
Balance:      1 * time.Minute,  // 60 seconds
Usage:        30 * time.Second, // 30 seconds
```

**Inconsistency Detected**: Balance and Usage have different TTLs (60s vs 30s)

### Test 3: Real-World Scenario
**Test**: `TestBug13_CacheCoherenceScenario`  
**Result**: ❌ FAILED (as expected)

**User Impact Scenario**:
- T=0: User makes purchase, balance and usage cached
- T=35s: User checks dashboard
- **Problem**: Dashboard shows balance from cache (old) but usage from DB (new)
- **Impact**: User sees inconsistent financial data

---

## Counterexample Documentation

### Primary Counterexample

```
Tenant ID: 3729b711-5d60-4be3-9f36-367f5492ae45
Balance: 100.50 SAR (TTL: 60s)
Usage: {"messages_sent":1000,"messages_used":500} (TTL: 30s)

Timeline:
- T=0s:  Both cached
- T=35s: Balance exists=true, Usage exists=false
```

### Impact Analysis

**Technical Impact**:
- Application reads balance from cache (potentially stale)
- Application reads usage from database (fresh data)
- Creates inconsistent view of tenant data

**Business Impact**:
- User sees old balance with new usage → confusion
- May cause incorrect billing calculations
- Dashboard displays inconsistent financial data
- Potential customer support issues

---

## Root Cause Analysis

**Location**: `services/backend/internal/redis/ttl.go`

**Current Implementation**:
```go
func DefaultTTLConfig() *TTLConfig {
    return &TTLConfig{
        // ... other fields ...
        Balance:      1 * time.Minute,  // 60 seconds
        Usage:        30 * time.Second, // 30 seconds - INCONSISTENT!
        // ... other fields ...
    }
}
```

**Problem**: Balance and Usage are tightly coupled financial data but use different TTLs:
- Balance: 60 seconds
- Usage: 30 seconds

**Why This Causes Issues**:
1. At T=0, both are cached
2. At T=35s, Usage expires but Balance remains
3. Application reads Balance from cache (old) and Usage from DB (new)
4. User sees inconsistent financial state

---

## Expected Behavior (After Fix)

**Requirement 2.13**: The system SHALL use consistent TTL values (e.g., both 1 minute) to maintain cache coherence

**Proposed Fix**:
```go
func DefaultTTLConfig() *TTLConfig {
    return &TTLConfig{
        // ... other fields ...
        Balance:      1 * time.Minute,  // 60 seconds
        Usage:        1 * time.Minute,  // 60 seconds - CONSISTENT!
        // ... other fields ...
    }
}
```

**Benefits**:
- Balance and Usage expire at the same time
- No cache coherence issues
- User always sees consistent financial data
- Either both from cache or both from DB

---

## Test Methodology

**Approach**: Bug Condition Exploration (Property-Based Testing)

**Property Tested**:
```
FUNCTION isBugCondition_13(input)
  INPUT: cache operations for balance and usage
  OUTPUT: boolean
  
  RETURN balance cache uses TTL = 1 minute
         AND usage cache uses TTL = 30 seconds
         AND inconsistent TTLs cause cache coherence issues
END FUNCTION
```

**Test Strategy**:
1. ✅ Cache both balance and usage at T=0
2. ✅ Wait 35 seconds (after usage expires, before balance expires)
3. ✅ Check cache state
4. ✅ Verify inconsistency (balance cached, usage expired)
5. ✅ Document counterexample

**Tools Used**:
- `miniredis` for in-memory Redis testing
- `FastForward()` to simulate time passage
- Property-based testing methodology

---

## Preservation Requirements

**What Must NOT Break** (Requirements 3.6, 3.7, 3.8):

1. **Cache Performance**: Cache hit rates must remain high
2. **Pricing Cache**: Pricing cache hits for common message lengths must continue
3. **Channel Cache**: Channel cache hits for active tenants must continue
4. **Redis Operations**: Redis operations must continue to provide fast queue operations

**Preservation Tests**: Will be written in Task 13.2

---

## Next Steps

1. ✅ **Task 13.1 Complete**: Bug condition exploration test written and executed
2. ⏭️ **Task 13.2**: Write preservation property tests (BEFORE implementing fix)
3. ⏭️ **Task 13.3**: Implement fix for inconsistent TTL values
4. ⏭️ **Task 13.4**: Verify bug condition test passes after fix
5. ⏭️ **Task 13.5**: Verify preservation tests still pass

---

## Conclusion

✅ **Bug 13 EXISTS and has been successfully reproduced**

The exploration test confirmed that:
- Balance uses 60s TTL
- Usage uses 30s TTL
- This inconsistency causes cache coherence issues at T=35s
- Users see old balance with new usage, creating confusion

The test is working correctly and will validate the fix when implemented in Task 13.3.

**Test Status**: PASSED (bug detection successful)  
**Bug Status**: CONFIRMED  
**Ready for**: Task 13.2 (Preservation Tests)
