# Bug 29 Preservation Property Tests - Instructions

## Test Information

- **Bug ID**: Bug 29
- **Bug Title**: No API Rate Limiting
- **Test Type**: Preservation Property Tests
- **Task**: 29.2 - Write preservation property tests
- **Property**: Property 2 - Normal Usage Works
- **Validates**: Requirements 3.9, 3.10

## Purpose

These tests verify that **normal API usage patterns continue to work correctly** and are NOT blocked by rate limiting. They should **PASS on both UNFIXED and FIXED code**.

The goal is to ensure that legitimate admin panel usage is not affected by the rate limiting implementation.

## Test Files

1. **Full Test Suite**: `apps/admin-web/src/api/__tests__/rate_limit.bug29.preservation.test.js`
   - ES6 module format with axios
   - 3 comprehensive tests
   - Detailed output

2. **Standalone Test**: `test-bug29-preservation.js` (project root)
   - CommonJS format with native http module
   - No external dependencies
   - Can be run directly: `node test-bug29-preservation.js`

## Test Strategy

### Test 1: Normal Request Rate
- **Scenario**: Typical admin panel usage
- **Requests**: 20 requests over 20 seconds
- **Rate**: 1 request per second (60 req/min)
- **Expected**: All requests succeed (no 429 responses)
- **Validates**: Normal browsing patterns work

### Test 2: Typical Admin Workflow
- **Scenario**: Admin panel page load
- **Requests**: 4 requests in quick succession
  - Load messages list
  - Load gateways list
  - Load tenants list
  - Load statistics
- **Expected**: All requests succeed
- **Validates**: Page load workflows work

### Test 3: Sustained Moderate Load (Full test only)
- **Scenario**: Sustained admin panel usage
- **Requests**: 50 requests over 60 seconds
- **Rate**: ~50 req/min (below 100 req/min limit)
- **Expected**: All requests succeed
- **Validates**: Sustained usage works

## How to Run

### Prerequisites

1. **Backend Server Running**:
   ```bash
   cd services/backend
   go run main.go
   ```

2. **Node.js Installed**: v16+ required

### Execution Options

#### Option 1: Standalone Test (Recommended)

```bash
# From project root
node test-bug29-preservation.js
```

**Advantages**:
- No dependencies required
- Uses native Node.js http module
- Simple and fast
- 2 core tests

#### Option 2: Full Test Suite

```bash
# From project root
node apps/admin-web/src/api/__tests__/rate_limit.bug29.preservation.test.js
```

**Advantages**:
- Uses axios (more realistic)
- 3 comprehensive tests
- Detailed output

## Expected Results

### On UNFIXED Code (Current State)

```
╔════════════════════════════════════════════════════════════╗
║  Bug 29 Preservation Property Tests                       ║
║  Property 2: Normal Usage Works                           ║
╚════════════════════════════════════════════════════════════╝

Validates: Requirements 3.9, 3.10
Expected: All tests PASS (normal usage not blocked)

Checking backend connectivity...
✓ Backend is reachable

=== Test 1: Normal Request Rate ===
Making 20 requests at 1 req/sec (60 req/min)...

Results: 20 successful, 0 rate limited
✓ Test 1: PASSED

=== Test 2: Typical Admin Workflow ===
Simulating admin panel page load (4 requests)...

Results: 4 successful, 0 rate limited
✓ Test 2: PASSED

╔════════════════════════════════════════════════════════════╗
║  Test Summary                                              ║
╚════════════════════════════════════════════════════════════╝

Tests Passed: 2/2

✓ Preservation Tests: PASSED

✓ Normal API usage is NOT blocked
✓ Typical admin workflows work correctly

✓ PRESERVATION VERIFIED: Normal usage works

Exit Code: 0
```

### After Fix Implementation

**Expected**: Same result - all tests should still PASS

```
Tests Passed: 2/2
✓ Preservation Tests: PASSED
✓ PRESERVATION VERIFIED: Normal usage works
Exit Code: 0
```

**Important**: If tests FAIL after fix, it means the rate limiting is too aggressive and needs adjustment.

## Preservation Requirements

From bugfix.md Requirements 3.9, 3.10:

**3.9**: WHEN valid JWT tokens are provided THEN the system SHALL CONTINUE TO authenticate users and extract tenant context correctly

**3.10**: WHEN API keys are used for authentication THEN the system SHALL CONTINUE TO validate them and allow programmatic access

## Success Criteria

✓ All tests PASS on UNFIXED code (current state)
✓ All tests PASS after rate limiting is implemented
✓ Normal admin panel usage is not blocked
✓ Typical workflows complete successfully
✓ Performance is acceptable (no noticeable delays)

## Troubleshooting

### Backend Not Running

```
✗ Test execution error: Backend server not running

Please start the backend server:
  cd services/backend
  go run main.go
```

**Solution**: Start the backend server before running tests

### Tests Fail After Fix

If tests fail after implementing rate limiting:

1. **Check Rate Limit Configuration**:
   - Ensure limit is 100 req/min (not lower)
   - Verify window is 1 minute

2. **Check IP Detection**:
   - Ensure localhost requests are handled correctly
   - Verify IP extraction logic

3. **Check Endpoint Coverage**:
   - Ensure rate limiting is applied to correct endpoints
   - Verify auth endpoints have separate limits

4. **Adjust Rate Limit**:
   - If legitimate usage is blocked, increase limit
   - Consider tiered limits (admin vs user)

### Connection Timeout

If requests timeout:
- Check backend is running and accessible
- Verify port 8081 is correct
- Check firewall settings

## Next Steps

After preservation tests pass:

1. **Task 29.3.1**: Install rate limiting library
2. **Task 29.3.2**: Create rate limiter middleware
3. **Task 29.3.3**: Apply middleware to API routes
4. **Task 29.3.5**: Update frontend error handling
5. **Task 29.3.6**: Verify bug condition test passes
6. **Task 29.3.7**: Re-run these preservation tests (should still pass)

## Notes

1. **These tests should ALWAYS pass**: They verify normal usage, not bug conditions
2. **Run on both unfixed and fixed code**: Ensures no regressions
3. **Realistic scenarios**: Tests simulate actual admin panel usage
4. **Performance sensitive**: Tests measure actual request timing
5. **Network dependent**: Requires backend server running

## Test Metadata

- **Property-Based Test**: No (specific scenario tests)
- **Framework**: Node.js with native http / axios
- **Test Type**: Preservation Property Tests
- **Expected Outcome**: PASS (on both unfixed and fixed code)
- **Requirements Validated**: 3.9, 3.10
- **Exit Codes**:
  - 0: All tests passed (preservation verified)
  - 1: Some tests failed (normal usage affected)
  - 3: Test execution error (backend not running)
