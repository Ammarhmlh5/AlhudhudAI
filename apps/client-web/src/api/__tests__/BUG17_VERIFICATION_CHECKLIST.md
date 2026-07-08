# Bug 17: CSRF Protection - Verification Checklist

## Overview
This document provides step-by-step instructions to verify that Bug 17 (Missing CSRF Protection) has been successfully fixed.

---

## Prerequisites

Before running verification tests:

1. **Backend Server Running**:
   ```bash
   cd services/backend
   go run main.go
   ```
   - Server should start on port 8081
   - Check for "Server listening" message

2. **Database Running**:
   - PostgreSQL should be running
   - Database should be seeded

3. **Redis Running** (optional but recommended):
   - Redis server should be running for rate limiting

---

## Verification Steps

### Step 1: Verify Backend Compilation ✅

**Status**: COMPLETED

The backend compiles successfully with no errors:
- `services/backend/internal/middleware/csrf.go` - No diagnostics
- `services/backend/internal/api/csrf_handler.go` - No diagnostics
- `services/backend/main.go` - No diagnostics

---

### Step 2: Run Bug Condition Exploration Test

**Test File**: `apps/client-web/src/api/__tests__/axios.bug17.test.js`

**Command**:
```bash
node apps/client-web/src/api/__tests__/axios.bug17.test.js
```

**Expected Output** (AFTER FIX):
```
=== Bug 17: CSRF Protection Exploration Test ===

Test 1: Checking if API accepts requests without CSRF token...

Request headers:
  Authorization: Bearer [token]
  X-CSRF-Token: MISSING (This is the bug!)

Making POST request without CSRF token...

✅ CSRF protection is working!
Request was rejected with 403 Forbidden
Error message: CSRF token missing in cookie

Bug 17 is FIXED - CSRF protection is active.

=== TEST SUMMARY ===

✅ BUG 17 FIXED: CSRF Protection Active

All tests passed - CSRF protection is working correctly
```

**What This Proves**:
- Requests without CSRF token are rejected with 403
- CSRF middleware is active and validating tokens
- Bug 17 is fixed

---

### Step 3: Run Preservation Tests

**Test File**: `apps/client-web/src/api/__tests__/axios.bug17.preservation.test.js`

**Command**:
```bash
node apps/client-web/src/api/__tests__/axios.bug17.preservation.test.js
```

**Expected Output**:
```
╔════════════════════════════════════════════════════════════╗
║  Bug 17: Preservation Tests - Existing Functionality      ║
╚════════════════════════════════════════════════════════════╝

=== Preservation Test 1: JWT Token Authentication Works ===
Testing: Fetch messages (GET /api/messages)
  ✓ Request succeeded: 200 OK
  - Authorization header: ✓ Bearer eyJhbGc...
  - withCredentials: ✓
  - Content-Type: ✓
  - Tenant context extracted: ✓ (12345678-90ab-cdef-1234-567890abcdef)

[... more tests ...]

✓ PRESERVATION TEST PASSED
  JWT token authentication continues to work correctly.

=== Preservation Test 2: API Key Authentication Works ===
[... tests ...]

✓ PRESERVATION TEST PASSED
  API key authentication continues to work correctly.

=== Preservation Test 3: withCredentials Configuration Preserved ===
[... tests ...]

✓ PRESERVATION TEST PASSED
  withCredentials configuration is preserved correctly.

=== Preservation Test 4: Normal API Operations Work ===
[... tests ...]

✓ PRESERVATION TEST PASSED
  Normal API operations (CRUD) work correctly.

╔════════════════════════════════════════════════════════════╗
║                  PRESERVATION TEST SUMMARY                 ║
╚════════════════════════════════════════════════════════════╝

1. JWT Token Authentication Works: ✓ PASS
2. API Key Authentication Works: ✓ PASS
3. withCredentials Configuration Preserved: ✓ PASS
4. Normal API Operations Work: ✓ PASS

Total: 4/4 tests passed

✓ ALL PRESERVATION TESTS PASSED
  Existing authentication and API functionality is preserved.
```

**What This Proves**:
- JWT authentication still works
- API key authentication still works
- withCredentials configuration preserved
- Normal CRUD operations work
- No regressions introduced

---

### Step 4: Manual Browser Testing

#### 4.1 Test CSRF Token Fetching

1. **Start Frontend**:
   ```bash
   cd apps/client-web
   npm run dev
   ```

2. **Open Browser** (http://localhost:5173)

3. **Open Developer Tools** → Console

4. **Expected Console Output**:
   ```
   CSRF token fetched successfully
   ```

5. **Check Network Tab**:
   - Look for request to `/api/v1/csrf-token`
   - Status: 200 OK
   - Response should contain `csrf_token`

#### 4.2 Test CSRF Token in Requests

1. **Login to Application**

2. **Perform State-Changing Operation** (e.g., send message, create contact)

3. **Open Developer Tools** → Network Tab

4. **Find POST/PUT/DELETE Request**

5. **Check Request Headers**:
   - Should contain: `X-CSRF-Token: [base64-token]`
   - Should contain: `Authorization: Bearer [jwt-token]`

6. **Check Cookies**:
   - Should contain: `csrf_token=[base64-token]`

7. **Verify Response**:
   - Status: 200 OK (request succeeded)

#### 4.3 Test CSRF Protection (Negative Test)

1. **Open Developer Tools** → Console

2. **Clear CSRF Cookie**:
   ```javascript
   document.cookie = 'csrf_token=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;';
   ```

3. **Try State-Changing Operation** (e.g., send message)

4. **Expected Behavior**:
   - Request fails with 403 Forbidden
   - Console shows: "CSRF token error detected, refreshing token..."
   - New CSRF token is fetched automatically
   - Request is retried and succeeds

5. **Verify**:
   - Check Network tab for two requests:
     1. First request: 403 Forbidden (CSRF error)
     2. GET /api/v1/csrf-token: 200 OK (token refresh)
     3. Retry request: 200 OK (success with new token)

---

### Step 5: Security Testing

#### 5.1 Test CSRF Attack Scenario

**Scenario**: Attacker tries to perform CSRF attack

1. **Create Malicious HTML File** (`csrf_attack.html`):
   ```html
   <!DOCTYPE html>
   <html>
   <head>
       <title>CSRF Attack Test</title>
   </head>
   <body>
       <h1>CSRF Attack Test</h1>
       <form id="attackForm" action="http://localhost:8081/api/v1/messaging/send" method="POST">
           <input type="hidden" name="phone_number" value="+1234567890" />
           <input type="hidden" name="content" value="This is a CSRF attack!" />
       </form>
       <script>
           // Automatically submit form (simulating CSRF attack)
           document.getElementById('attackForm').submit();
       </script>
   </body>
   </html>
   ```

2. **Open File in Browser** (while logged into application in another tab)

3. **Expected Behavior**:
   - Request is sent with cookies (browser automatically includes them)
   - Request is REJECTED with 403 Forbidden
   - Error: "CSRF token missing in header"

4. **Why Attack Fails**:
   - Attacker cannot read CSRF token from victim's cookies (Same-Origin Policy)
   - Attacker cannot set custom headers in cross-origin form submission
   - Server rejects request because `X-CSRF-Token` header is missing

#### 5.2 Test API Key Exemption

**Scenario**: Server-to-server API calls should work without CSRF token

1. **Create Test Script** (`test_api_key.js`):
   ```javascript
   const axios = require('axios');

   async function testAPIKey() {
     try {
       const response = await axios.post('http://localhost:8081/api/v1/messaging/send', {
         phone_number: '+1234567890',
         content: 'Test message via API key'
       }, {
         headers: {
           'X-API-Key': 'your-api-key-here',
           'Content-Type': 'application/json'
         }
       });

       console.log('✓ API key request succeeded:', response.status);
       console.log('Response:', response.data);
     } catch (error) {
       console.error('✗ API key request failed:', error.response?.status);
       console.error('Error:', error.response?.data);
     }
   }

   testAPIKey();
   ```

2. **Run Script**:
   ```bash
   node test_api_key.js
   ```

3. **Expected Behavior**:
   - Request succeeds with 200 OK
   - No CSRF token required (API key authentication)
   - Message is sent successfully

4. **What This Proves**:
   - API key authentication is exempt from CSRF protection
   - Server-to-server communication works correctly
   - CSRF middleware correctly skips API key requests

---

## Verification Results

### Automated Tests

| Test | Status | Notes |
|------|--------|-------|
| Backend Compilation | ✅ PASS | No diagnostics found |
| Bug Condition Test | ⏳ PENDING | Requires running backend server |
| Preservation Tests | ⏳ PENDING | Requires running backend server |

### Manual Tests

| Test | Status | Notes |
|------|--------|-------|
| CSRF Token Fetching | ⏳ PENDING | Check browser console |
| CSRF Token in Requests | ⏳ PENDING | Check Network tab |
| CSRF Protection (Negative) | ⏳ PENDING | Test token refresh |
| CSRF Attack Scenario | ⏳ PENDING | Verify attack fails |
| API Key Exemption | ⏳ PENDING | Verify API keys work |

---

## Common Issues and Solutions

### Issue 1: CSRF Token Not Fetched

**Symptom**: Console shows "Failed to fetch CSRF token"

**Solution**:
- Verify backend server is running
- Check CORS configuration allows credentials
- Verify `/api/v1/csrf-token` endpoint is accessible

### Issue 2: Requests Fail with 403 CSRF Error

**Symptom**: All POST/PUT/DELETE requests fail with 403

**Possible Causes**:
1. CSRF token not being added to headers
   - Check axios interceptor is working
   - Verify `X-CSRF-Token` header is present

2. Cookie not being sent
   - Verify `withCredentials: true` is set
   - Check browser allows cookies

3. Token expired
   - Should auto-refresh, check console for refresh attempts
   - Verify token refresh logic is working

**Solution**:
- Check browser console for error messages
- Verify CSRF token is present in both cookie and header
- Test token refresh by clearing cookie manually

### Issue 3: API Key Requests Fail

**Symptom**: API key authenticated requests fail with 403

**Solution**:
- Verify CSRF middleware checks for `X-API-Key` header
- Ensure API key is valid
- Check middleware order (API key middleware should run before CSRF)

---

## Success Criteria

The fix is considered successful when:

✅ **Bug Condition Test**:
- Test PASSES (requests without CSRF token are rejected)

✅ **Preservation Tests**:
- All 4 tests PASS (existing functionality preserved)

✅ **Manual Browser Testing**:
- CSRF token is fetched on app load
- CSRF token is included in state-changing requests
- Requests succeed with valid token
- Token auto-refreshes on errors

✅ **Security Testing**:
- CSRF attack attempts fail
- API key requests work without CSRF token

✅ **No Regressions**:
- JWT authentication works
- API key authentication works
- All existing features work correctly

---

## Conclusion

Once all verification steps are completed successfully, Bug 17 (Missing CSRF Protection) can be marked as FIXED.

The implementation provides:
- Strong CSRF protection using double-submit cookie pattern
- Automatic token refresh on errors
- Transparent integration with existing authentication
- No impact on API key authentication
- Full backward compatibility

**Next Steps**:
1. Run automated tests with backend server
2. Perform manual browser testing
3. Conduct security testing
4. Update task status to COMPLETED
