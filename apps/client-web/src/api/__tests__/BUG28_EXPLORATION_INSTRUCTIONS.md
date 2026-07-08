# Bug 28: No Message List Pagination - Exploration Test Instructions

## Test Overview

**Bug Condition (1.28):**
WHEN message lists are loaded THEN the system loads all messages at once without pagination, causing memory and performance issues.

**Expected Behavior (2.28):**
WHEN message lists are loaded THEN the system SHALL implement pagination with configurable page size (default 50) and load-more functionality.

## Test File

`messages.bug28.test.js` - Bug condition exploration test

## Prerequisites

1. **Backend server must be running:**
   ```bash
   cd services/backend
   go run main.go
   ```

2. **Valid test credentials:**
   - Email: `admin@alhudhud.com`
   - Password: `admin123`
   - Or update credentials in the test file

3. **Database access:**
   - PostgreSQL database must be accessible
   - Test will create 10,000 test messages

## Running the Test

### Option 1: Using npm script (Recommended)

```bash
cd apps/client-web
npm run test:bug28
```

### Option 2: Direct execution

```bash
cd apps/client-web
node src/api/__tests__/messages.bug28.test.js
```

## Expected Outcomes

### On UNFIXED Code (Bug Exists)

The test should **FAIL** with the following indicators:

1. **All messages loaded at once:**
   - API returns 10,000+ messages in single response
   - Response size exceeds 10MB
   - Response time exceeds 5 seconds

2. **Missing pagination metadata:**
   - No `page` field in response
   - No `page_size` field in response
   - No `total_pages` field in response

3. **Excessive memory usage:**
   - Memory increase > 5MB
   - Heap usage spikes significantly

4. **Performance degradation:**
   - Response time > 1 second
   - Browser/client becomes unresponsive

**Expected Test Output:**
```
✗ BUG CONDITION: API loads all messages at once
✗ BUG CONDITION: Response lacks pagination metadata
✗ BUG CONDITION: Excessive memory usage
✗ BUG CONDITION: Slow response time with large dataset

=== Bug Condition Exploration Result ===
✓ BUG DETECTED: Message list loads all records without pagination
  - All 10K messages loaded in single request
  - Excessive memory and performance impact
  - Missing pagination metadata
```

### On FIXED Code (Bug Resolved)

The test should **PASS** with the following indicators:

1. **Pagination implemented:**
   - API returns only 50 messages (default page size)
   - Response includes pagination metadata
   - Response size < 1MB
   - Response time < 1 second

2. **Controlled memory usage:**
   - Memory increase < 5MB
   - No significant heap spikes

**Expected Test Output:**
```
✓ All tests passed
⚠ UNEXPECTED: All tests passed on unfixed code
This suggests pagination may already be implemented.
```

## Test Metrics

The test measures:

1. **Response Size:**
   - Calculates JSON response size in MB
   - Threshold: 10MB (bug indicator)

2. **Response Time:**
   - Measures API response time in milliseconds
   - Threshold: 5000ms (bug indicator)

3. **Memory Usage:**
   - Tracks heap memory before/after request
   - Threshold: 5MB increase (bug indicator)

4. **Message Count:**
   - Counts messages in response
   - Expected: 50 (paginated) vs 10,000+ (bug)

## Cleanup

After running the test, cleanup is required:

### Manual Cleanup (SQL)

```sql
-- Delete test messages created by Bug 28 test
DELETE FROM messages 
WHERE sender_name = 'Bug28Test';
```

### Verify Cleanup

```sql
-- Check remaining test messages
SELECT COUNT(*) 
FROM messages 
WHERE sender_name = 'Bug28Test';
```

## Troubleshooting

### Authentication Failed

**Error:** `✗ Authentication failed`

**Solutions:**
1. Verify backend server is running on `http://localhost:8081`
2. Check credentials in test file match your setup
3. Verify database has admin user with correct password
4. Update `API_BASE_URL` if using different port

### Cannot Create Messages

**Error:** `Failed to create batch`

**Solutions:**
1. Check database connection
2. Verify tenant has sufficient balance
3. Check message ingestion endpoint is working
4. Review backend logs for errors

### Test Passes Unexpectedly

**Issue:** Test passes on unfixed code

**Explanation:**
This means pagination is already implemented in the backend. The bug may have been fixed, or the test is running against fixed code.

**Action:**
1. Verify backend implementation in `internal/receiver/service.go`
2. Check if `ListMessages` function uses pagination
3. Review API response structure
4. Update bug status if already fixed

## Test Validation

To manually verify the bug condition:

1. **Check API Response:**
   ```bash
   curl -H "Authorization: Bearer YOUR_TOKEN" \
        http://localhost:8081/api/v1/messaging/list
   ```

2. **Verify Response Structure:**
   - Count messages in response
   - Check for pagination fields
   - Measure response size

3. **Test with Query Parameters:**
   ```bash
   # Test pagination parameters
   curl -H "Authorization: Bearer YOUR_TOKEN" \
        "http://localhost:8081/api/v1/messaging/list?page=1&page_size=50"
   ```

## Notes

- Test creates 10,000 messages to simulate production load
- Test is designed to FAIL on unfixed code (confirms bug exists)
- Test should PASS after pagination is implemented
- Manual cleanup is required after test execution
- Test requires ~30-60 seconds to complete (message creation time)

## Related Files

- Test: `apps/client-web/src/api/__tests__/messages.bug28.test.js`
- Backend Handler: `services/backend/internal/receiver/handler.go`
- Backend Service: `services/backend/internal/receiver/service.go`
- Frontend Component: `apps/client-web/src/components/MessageHistory.jsx`
- Design Doc: `.kiro/specs/system-critical-bugfixes/design.md` (Bug 28)
- Requirements: `.kiro/specs/system-critical-bugfixes/bugfix.md` (1.28, 2.28)
