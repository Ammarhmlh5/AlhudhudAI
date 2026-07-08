# Bug 29 Exploration Test Instructions

## Overview

This document provides instructions for running the Bug 29 exploration test, which verifies that the admin panel API has no rate limiting implemented.

## Bug Description

**Bug 29: No API Rate Limiting**

- **Bug Condition (1.29)**: When admin panel makes API requests, the system has no rate limiting, allowing potential abuse
- **Expected Behavior (2.29)**: System should implement rate limiting (e.g., 100 requests per minute per IP)

## Test File

- **Location**: `apps/admin-web/src/api/__tests__/rate_limit.bug29.test.js`
- **Type**: Bug Condition Exploration Test
- **Framework**: Node.js with axios

## Prerequisites

1. **Backend Server Running**: The backend API must be running on `http://localhost:8081`
2. **Node.js Installed**: Node.js v16+ required
3. **Dependencies**: axios package (already in project)

## Running the Test

### Option 1: Direct Execution

```bash
# From project root
node apps/admin-web/src/api/__tests__/rate_limit.bug29.test.js
```

### Option 2: Using npm/yarn (if test script is configured)

```bash
# From admin-web directory
cd apps/admin-web
npm test rate_limit.bug29.test.js
```

## Expected Results

### On UNFIXED Code (Current State)

The test should **FAIL** with output similar to:

```
=== Bug 29 Exploration Test: No API Rate Limiting ===

Test Configuration:
  - API Base URL: http://localhost:8081
  - Test Endpoint: /api/v1/messages
  - Total Requests: 1000
  - Expected Rate Limit: 100 req/min

=== Making Rapid Requests ===
Sending 1000 requests as fast as possible...

Progress: 100% (1000/1000)

=== Test Results ===
Duration: 2500ms (400 req/sec)

Response Breakdown:
  - Total Requests: 1000
  - Accepted (2xx): 0
  - Rate Limited (429): 0 ✗
  - Unauthorized (401): 1000
  - Forbidden (403): 0
  - Server Error (5xx): 0
  - Network Error: 0
  - Other: 0

=== Analysis ===
Rate Limiting Active: NO ✗
Rate Limited Percentage: 0.0%

=== Bug Status ===
✗ BUG CONFIRMED: No rate limiting detected!
  All requests were processed without rate limiting.
  The system is vulnerable to abuse.

  Expected Behavior:
    - First ~100 requests: Accepted
    - Remaining requests: 429 Too Many Requests

  Actual Behavior:
    - All 1000 requests: Processed
    - Zero 429 responses

✗ EXPLORATION TEST FAILED (Bug Exists)
  This test will PASS after the bug is fixed.

Status: BUG EXISTS - No rate limiting implemented
Action: Implement rate limiting middleware
        - Install rate limiting library
        - Configure 100 requests per minute per IP
        - Apply to /api routes
        - Return 429 on limit exceeded

Exit Code: 1
```

**Key Indicators of Bug**:
- ✗ Rate Limited (429): 0
- ✗ Rate Limiting Active: NO
- ✗ BUG CONFIRMED: No rate limiting detected!
- Exit Code: 1 (failure)

### After FIX is Implemented

The test should **PASS** with output similar to:

```
=== Bug 29 Exploration Test: No API Rate Limiting ===

Test Configuration:
  - API Base URL: http://localhost:8081
  - Test Endpoint: /api/v1/messages
  - Total Requests: 1000
  - Expected Rate Limit: 100 req/min

=== Making Rapid Requests ===
Sending 1000 requests as fast as possible...

Progress: 100% (1000/1000)

=== Test Results ===
Duration: 3200ms (312 req/sec)

Response Breakdown:
  - Total Requests: 1000
  - Accepted (2xx): 0
  - Rate Limited (429): 900 ✓
  - Unauthorized (401): 100
  - Forbidden (403): 0
  - Server Error (5xx): 0
  - Network Error: 0
  - Other: 0

=== Analysis ===
Rate Limiting Active: YES ✓
Rate Limited Percentage: 90.0%
Requests Before Limit: 100
Requests After Limit: 900

=== Bug Status ===
✓ Rate limiting is ACTIVE!
  900 requests were rate limited (429).
  The system is protected against abuse.

✓ EXPLORATION TEST PASSED (Bug Fixed)
  Rate limiting is working correctly.

Status: BUG FIXED - Rate limiting is active
Action: No action needed, bug has been resolved

Exit Code: 0
```

**Key Indicators of Fix**:
- ✓ Rate Limited (429): 900
- ✓ Rate Limiting Active: YES
- ✓ EXPLORATION TEST PASSED (Bug Fixed)
- Exit Code: 0 (success)

## Test Properties

### Property 1: Bug Condition - No Rate Limiting

**Validates**: Requirements 1.29, 2.29

**Test Logic**:
1. Make 1000 rapid requests to `/api/v1/messages`
2. Count responses by status code
3. Verify that ZERO 429 (Too Many Requests) responses are received
4. If no 429 responses → Bug confirmed (test FAILS)
5. If 429 responses present → Bug fixed (test PASSES)

**Bug Condition Function**:
```
FUNCTION isBugCondition_29(input)
  INPUT: API requests from admin panel
  OUTPUT: boolean
  
  RETURN admin panel makes API requests
         AND no rate limiting is implemented
         AND allows potential abuse
END FUNCTION
```

### Property 2: Expected Behavior - Rate Limiting Active

**Validates**: Requirements 2.29

**Test Logic**:
1. **Phase 1**: Make 50 controlled requests (within limit)
   - Expected: All accepted, no 429 responses
2. **Phase 2**: Make 150 burst requests (exceeding limit)
   - Expected: Some accepted, many 429 responses
3. Verify rate limiting is working correctly

## Troubleshooting

### Backend Not Running

If you see network errors:
```
Network Error: 1000
```

**Solution**: Start the backend server:
```bash
cd services/backend
go run main.go
```

### Import Error (ES Modules)

If you see:
```
SyntaxError: Cannot use import statement outside a module
```

**Solution**: Run with Node.js ES module support:
```bash
node --input-type=module apps/admin-web/src/api/__tests__/rate_limit.bug29.test.js
```

Or convert to CommonJS:
```javascript
const axios = require('axios');
```

### Test Passes Unexpectedly

If the test passes on unfixed code, it means:
1. Rate limiting is already implemented (bug doesn't exist)
2. Test endpoint is incorrect
3. Backend configuration has changed

**Action**: Review the test output and verify the bug still exists manually.

## Manual Verification

To manually verify the bug exists:

1. **Start Backend**: `cd services/backend && go run main.go`
2. **Open Browser Console**: Navigate to admin panel
3. **Run Script**:
```javascript
// Make 200 rapid requests
for (let i = 0; i < 200; i++) {
  fetch('http://localhost:8081/api/v1/messages', {
    credentials: 'include'
  }).then(r => console.log(i, r.status));
}
```
4. **Check Results**: If all requests return 200/401 (no 429), bug exists

## Next Steps

After confirming the bug:

1. **Task 29.2**: Write preservation property tests
2. **Task 29.3**: Implement rate limiting fix
   - Install rate limiting library
   - Create rate limiter middleware
   - Apply to API routes
   - Configure 100 req/min per IP
3. **Task 29.3.6**: Re-run this test (should PASS)
4. **Task 29.3.7**: Run preservation tests (should PASS)

## Related Files

- **Test File**: `apps/admin-web/src/api/__tests__/rate_limit.bug29.test.js`
- **Spec**: `.kiro/specs/system-critical-bugfixes/`
- **Design**: `.kiro/specs/system-critical-bugfixes/design.md` (Bug 29)
- **Tasks**: `.kiro/specs/system-critical-bugfixes/tasks.md` (Task 29.1)

## Notes

- This is a **bug exploration test** - it's EXPECTED to FAIL on unfixed code
- The test confirms the bug exists by detecting the ABSENCE of rate limiting
- After the fix is implemented, this test should PASS
- The test makes 1000 requests to thoroughly verify rate limiting behavior
- Network latency and server performance may affect exact numbers
