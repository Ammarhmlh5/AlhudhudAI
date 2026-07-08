# Bug 4 Exploration Test Results

## Test Status: ✅ PASSED (Bug Confirmed)

**Date**: 2026-04-03  
**Task**: 4.1 Write bug condition exploration test  
**Requirements**: 1.4, 2.4

## Summary

The bug exploration test **successfully detected the bug** by failing on unfixed code. This confirms that Bug 4 exists as described in the bugfix requirements.

## Bug Description

**Bug**: GetCurrentLoad() returns MessageCount: 0 despite messages existing in the system

**Location**: `services/backend/internal/redis/analytics.go` - `GetCurrentLoad()` function

## Test Results

### Simple Unit Test
```
Created: 10 messages with event metadata
Event metadata: map[msg_count:10 ...]
GetCurrentLoad() returned:
  - EventCount: 1 ✅ (correct)
  - MessageCount: 0 ❌ (BUG - should be 10)
  - AvgMsgPerEvent: 0.00 ❌ (should be 10.0)
```

### Property-Based Test
```
Counterexample:
  - Created: 35 messages
  - Event ID: EV260403165432170001
  - EventCount: 4 ✅ (correct)
  - MessageCount: 0 ❌ (BUG - should be at least 35)
```

## Root Cause Analysis

Looking at `analytics.go` line 30-60, the `GetCurrentLoad()` function:

1. ✅ **Correctly counts events**: Scans `alhudhud:event:counter:*` keys
2. ❌ **Does NOT count messages**: Has a comment "We need to check all events created with this counter" but doesn't implement it
3. ❌ **Missing implementation**: 
   - Does not scan event metadata keys to extract `message_count`
   - Does not scan queue keys for pending messages
   - Variables `totalMessages`, `maxMessages`, `nearLimit` remain at 0

## Evidence

### Event Metadata Exists
```go
// Event metadata is correctly stored with msg_count
map[created_at:1775224423 msg_count:10 sender_ip:192.168.1.1 tenant_id:...]
```

### Queue Has Messages
```go
// Queue size verification passed
queueSize: 10 messages in alhudhud:queue:messages:normal:{tenant_id}
```

### GetCurrentLoad() Returns Zero
```go
// But GetCurrentLoad() returns 0 for MessageCount
MessageCount: 0 (should be 10+)
```

## Expected Fix

According to design.md, the fix should:

1. Scan Redis keys for event metadata: `alhudhud:event:metadata:EV*`
2. Extract `message_count` from each event's metadata hash
3. Scan queue keys: `alhudhud:queue:messages:*`
4. Count pending messages in queues using LLEN
5. Aggregate all counts and return accurate metrics

## Test Files Created

- `services/backend/internal/redis/analytics_bug4_test.go`
  - `TestBug4_Simple_GetCurrentLoad_ReturnsZeroMessages()` - Simple unit test
  - `TestBug4_GetCurrentLoad_ReturnsZeroMessages()` - Property-based test

## Next Steps

1. ✅ Task 4.1 Complete - Bug exploration test written and confirmed bug exists
2. ⏭️ Task 4.2 - Write preservation property tests
3. ⏭️ Task 4.3 - Implement fix for incomplete analytics calculation
4. ⏭️ Task 4.3.3 - Verify bug condition test passes after fix
5. ⏭️ Task 4.3.4 - Verify preservation tests still pass

## Validation

The test correctly implements the bug exploration methodology:
- ✅ Creates test data (messages and events)
- ✅ Calls the buggy function (GetCurrentLoad)
- ✅ Asserts expected behavior (MessageCount > 0)
- ✅ FAILS on unfixed code (confirming bug exists)
- ✅ Will PASS on fixed code (validating the fix)

This test serves dual purpose:
1. **Exploration**: Confirms the bug exists (FAILS on unfixed code)
2. **Validation**: Will verify the fix works (PASSES on fixed code)
