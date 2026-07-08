# Bug 17: Missing CSRF Protection - Exploration Test Instructions

## Overview

This exploration test demonstrates the CSRF (Cross-Site Request Forgery) vulnerability in the SMS Gateway API. The system currently uses `withCredentials: true` in axios configuration but has no CSRF token validation, making it vulnerable to CSRF attacks.

## Bug Condition

**Current Behavior (Defect):**
- API accepts POST/PUT/DELETE requests with valid session cookies
- No CSRF token validation is performed
- Attacker can craft malicious requests that execute with user's credentials

**Formal Specification:**
```
WHEN API request is made with withCredentials: true
AND request is a state-changing operation (POST/PUT/DELETE)
AND no CSRF token is included in request
THEN request is ACCEPTED (BUG - should be rejected)
```

## CSRF Attack Scenario

### How the Attack Works:

1. **User logs into legitimate app** → Browser stores session cookie
2. **User visits attacker's malicious website** (while still logged in)
3. **Attacker's site makes hidden request** to API endpoint
4. **Browser automatically sends cookies** with the request
5. **API accepts request** because it has valid session cookie
6. **Unauthorized action is performed** using user's credentials

### Example Attack:

```html
<!-- Attacker's malicious website -->
<form action="https://api.smsgateway.com/api/messages" method="POST">
  <input type="hidden" name="phone_number" value="+1234567890" />
  <input type="hidden" name="content" value="Spam message" />
</form>
<script>
  // Auto-submit form when page loads
  document.forms[0].submit();
</script>
```

When a logged-in user visits this page, the form submits automatically with their session cookie, and the API accepts it because there's no CSRF protection.

## Test Structure

### Property 1: Bug Condition - No CSRF Validation

**Test File:** `axios.bug17.test.js`

**What it tests:**
1. Makes POST request without CSRF token
2. Verifies request is accepted (proves bug exists)
3. Documents the security vulnerability

**Expected Outcomes:**

**On UNFIXED code (current state):**
- ❌ Test FAILS: Request is accepted without CSRF token
- This proves the bug exists and system is vulnerable

**On FIXED code (after implementation):**
- ✅ Test PASSES: Request is rejected with 403 Forbidden
- This proves CSRF protection is working

## Running the Test

```bash
cd apps/client-web
npm test -- axios.bug17.test.js
```

## Expected Test Results

### On Unfixed Code:

```
FAIL  src/api/__tests__/axios.bug17.test.js
  Bug 17: CSRF Protection Exploration
    ✕ Property 1: API accepts POST request without CSRF token (Bug Condition)
    ✓ Property 1 (Alternative): Current axios client does not send CSRF token
    ✓ Property 1 (Documentation): CSRF attack scenario

  ● Bug 17: CSRF Protection Exploration › Property 1: API accepts POST request without CSRF token

    SECURITY VULNERABILITY: API accepted POST request without CSRF token.
    This proves Bug 17 exists - the system is vulnerable to CSRF attacks.
```

### On Fixed Code:

```
PASS  src/api/__tests__/axios.bug17.test.js
  Bug 17: CSRF Protection Exploration
    ✓ Property 1: API accepts POST request without CSRF token (Bug Condition)
    ✓ Property 1 (Alternative): Current axios client does not send CSRF token
    ✓ Property 1 (Documentation): CSRF attack scenario
```

## Security Impact

**Severity:** HIGH

**Affected Operations:**
- POST /api/messages - Send unauthorized messages
- PUT /api/settings - Modify user settings
- DELETE /api/resources - Delete user data
- Any state-changing operation

**Risk:**
- Attacker can perform any action the user is authorized to do
- No user interaction required beyond visiting malicious site
- Attack is invisible to the user

## Required Fix

Implement CSRF protection using **Double-Submit Cookie Pattern**:

1. **Backend:**
   - Generate CSRF token on login/session creation
   - Set token in cookie (not httpOnly, so JS can read it)
   - Validate token on all POST/PUT/DELETE requests
   - Reject requests without valid token (403 Forbidden)

2. **Frontend:**
   - Fetch CSRF token from cookie
   - Include token in X-CSRF-Token header for state-changing requests
   - Refresh token when needed

## References

- **Requirements:** 1.17, 2.17
- **Design Document:** `.kiro/specs/system-critical-bugfixes/design.md` (Bug 17)
- **OWASP CSRF Prevention:** https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html

## Next Steps

1. ✅ Run exploration test to confirm bug exists
2. ⏳ Write preservation tests (legitimate requests should still work)
3. ⏳ Implement CSRF middleware (backend)
4. ⏳ Update axios client to send CSRF token (frontend)
5. ⏳ Verify exploration test passes after fix
6. ⏳ Verify preservation tests still pass

---

**Status:** Exploration Phase
**Last Updated:** 2024
**Bug Priority:** Security (High)
