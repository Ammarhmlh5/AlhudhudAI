# Bug 17: Missing CSRF Protection - Preservation Test Instructions

## Overview

**Bug ID:** Bug 17  
**Priority:** Security (High)  
**Task:** 17.2 Write preservation property tests  
**Status:** ✅ READY TO RUN

---

## Purpose

These preservation tests verify that existing authentication and API functionality continues to work correctly **BOTH before and after** CSRF protection is implemented.

**Critical Requirement:** All tests MUST PASS on UNFIXED code (before CSRF implementation).

---

## Preservation Requirements

### Requirement 3.9: JWT Token Authentication
**WHEN** valid JWT tokens are provided  
**THEN** the system SHALL CONTINUE TO authenticate users and extract tenant context correctly

### Requirement 3.10: API Key Authentication
**WHEN** API keys are used for authentication  
**THEN** the system SHALL CONTINUE TO validate them and allow programmatic access

---

## Test Properties

### Property 1: JWT Token Authentication Continues to Work
**Validates:** Requirement 3.9

**Test Strategy:**
- Set up valid JWT authentication token
- Make API requests with JWT token (GET, POST, PUT, DELETE)
- Verify Authorization header is added correctly
- Verify requests succeed with 200 OK
- Verify tenant context is extracted correctly
- Verify withCredentials is set correctly

**Expected Outcome on UNFIXED Code:**
- ✅ Test PASSES: All JWT authenticated requests work correctly
- ✅ Authorization header is added: `Bearer [token]`
- ✅ Tenant context is extracted from JWT
- ✅ No errors or failures

**Expected Outcome After CSRF Implementation:**
- ✅ Test STILL PASSES: JWT authentication continues to work
- ✅ CSRF token is transparently added to requests
- ✅ No changes required to existing JWT authentication flow

---

### Property 2: API Key Authentication Continues to Work
**Validates:** Requirement 3.10

**Test Strategy:**
- Set up valid API key for programmatic access
- Make API requests with API key (POST, GET)
- Verify X-API-Key header is added correctly
- Verify requests succeed with 200 OK
- Verify API key is validated correctly
- Verify programmatic access is allowed

**Expected Outcome on UNFIXED Code:**
- ✅ Test PASSES: All API key authenticated requests work correctly
- ✅ X-API-Key header is added correctly
- ✅ API key is validated by backend
- ✅ Programmatic access is allowed

**Expected Outcome After CSRF Implementation:**
- ✅ Test STILL PASSES: API key authentication continues to work
- ✅ CSRF protection may be bypassed for API key requests (common pattern)
- ✅ No changes required to existing API key authentication flow

---

### Property 3: withCredentials Configuration Preserved
**Validates:** Requirement 3.9

**Test Strategy:**
- Create axios instance with withCredentials: true
- Make API requests (GET, POST, PUT, DELETE)
- Verify withCredentials is set on all requests
- Verify cookies would be sent with requests
- Verify CORS headers are handled correctly

**Expected Outcome on UNFIXED Code:**
- ✅ Test PASSES: withCredentials is set correctly
- ✅ CORS headers are present (Access-Control-Allow-Credentials: true)
- ✅ Cookies are sent with requests

**Expected Outcome After CSRF Implementation:**
- ✅ Test STILL PASSES: withCredentials continues to work
- ✅ CSRF cookies are sent with requests (required for CSRF protection)
- ✅ No changes to withCredentials configuration

---

### Property 4: Normal API Operations Not Affected
**Validates:** Requirements 3.9, 3.10

**Test Strategy:**
- Test complete CRUD workflow
- CREATE resource (POST)
- READ resource (GET)
- UPDATE resource (PUT)
- DELETE resource (DELETE)
- Verify all operations succeed

**Expected Outcome on UNFIXED Code:**
- ✅ Test PASSES: All CRUD operations work correctly
- ✅ CREATE: 201 Created
- ✅ READ: 200 OK
- ✅ UPDATE: 200 OK
- ✅ DELETE: 200 OK

**Expected Outcome After CSRF Implementation:**
- ✅ Test STILL PASSES: All CRUD operations continue to work
- ✅ CSRF tokens are transparently added to state-changing requests (POST, PUT, DELETE)
- ✅ GET requests work without CSRF tokens (safe methods)
- ✅ No changes required to existing API calls

---

## How to Run Tests

### Prerequisites
- Node.js installed
- axios package installed (`npm install axios`)
- Test file: `axios.bug17.preservation.test.js`

### Run Command
```bash
cd apps/client-web
node src/api/__tests__/axios.bug17.preservation.test.js
```

### Expected Output (UNFIXED Code)
```
╔════════════════════════════════════════════════════════════╗
║  Bug 17: Preservation Tests - Existing Functionality      ║
╚════════════════════════════════════════════════════════════╝

=== Preservation Test 1: JWT Token Authentication Works ===
✓ All JWT authenticated requests succeeded
✓ PRESERVATION TEST PASSED

=== Preservation Test 2: API Key Authentication Works ===
✓ All API key authenticated requests succeeded
✓ PRESERVATION TEST PASSED

=== Preservation Test 3: withCredentials Configuration Preserved ===
✓ All requests have withCredentials set
✓ PRESERVATION TEST PASSED

=== Preservation Test 4: Normal API Operations Work ===
✓ All CRUD operations succeeded
✓ PRESERVATION TEST PASSED

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
  CSRF protection can be implemented without breaking existing features.
```

---

## Test Validation Checklist

### Before Running Tests
- [ ] Ensure axios.js is in current state (UNFIXED)
- [ ] Ensure no CSRF middleware exists in backend
- [ ] Ensure test file is in correct location
- [ ] Ensure Node.js and axios are installed

### After Running Tests (UNFIXED Code)
- [ ] All 4 tests PASS ✅
- [ ] JWT authentication works correctly
- [ ] API key authentication works correctly
- [ ] withCredentials is set correctly
- [ ] CRUD operations work correctly
- [ ] No errors or failures

### After CSRF Implementation
- [ ] Re-run preservation tests
- [ ] All 4 tests STILL PASS ✅
- [ ] JWT authentication still works
- [ ] API key authentication still works
- [ ] withCredentials still works
- [ ] CRUD operations still work
- [ ] CSRF tokens are added transparently
- [ ] No regressions introduced

---

## What These Tests Prove

### On UNFIXED Code (Before CSRF)
These tests establish the **baseline behavior** that must be preserved:

1. ✅ **JWT authentication works** - Users can authenticate with JWT tokens
2. ✅ **API key authentication works** - Programmatic access is allowed
3. ✅ **withCredentials is set** - Cookies are sent with requests
4. ✅ **CRUD operations work** - All API operations function correctly

### After CSRF Implementation
These tests verify **no regressions** were introduced:

1. ✅ **JWT authentication still works** - No changes to authentication flow
2. ✅ **API key authentication still works** - Programmatic access still allowed
3. ✅ **withCredentials still works** - CSRF cookies are sent correctly
4. ✅ **CRUD operations still work** - CSRF tokens added transparently

---

## CSRF Implementation Guidelines

When implementing CSRF protection, ensure:

### Backend
- ✅ CSRF middleware validates tokens on POST/PUT/DELETE/PATCH
- ✅ CSRF middleware skips validation on GET/HEAD/OPTIONS (safe methods)
- ✅ CSRF middleware may skip validation for API key authentication
- ✅ CSRF tokens are generated and set in cookies
- ✅ CSRF endpoint exists: `/api/csrf-token`

### Frontend
- ✅ Fetch CSRF token on app initialization
- ✅ Add CSRF token to request headers: `X-CSRF-Token`
- ✅ Refresh CSRF token on 403 CSRF errors
- ✅ No changes to existing authentication logic
- ✅ No changes to existing API calls

### Transparency
- ✅ CSRF protection is transparent to application code
- ✅ Existing API calls work without modification
- ✅ JWT authentication flow unchanged
- ✅ API key authentication flow unchanged

---

## Success Criteria

### Task 17.2 Complete When:
1. ✅ Preservation test file created: `axios.bug17.preservation.test.js`
2. ✅ Preservation instructions created: `BUG17_PRESERVATION_INSTRUCTIONS.md`
3. ✅ Tests run successfully on UNFIXED code
4. ✅ All 4 preservation tests PASS
5. ✅ Results documented in: `BUG17_PRESERVATION_RESULTS.md`

### CSRF Implementation Ready When:
1. ✅ All preservation tests PASS on UNFIXED code (baseline established)
2. ✅ Baseline behavior is documented
3. ✅ Implementation can proceed without breaking existing features

---

## Next Steps

1. ✅ **Run preservation tests** on UNFIXED code
2. ✅ **Document results** in `BUG17_PRESERVATION_RESULTS.md`
3. ⏳ **Implement CSRF middleware** (Task 17.3)
4. ⏳ **Update axios client** to fetch and send CSRF tokens
5. ⏳ **Re-run preservation tests** to verify no regressions
6. ⏳ **Re-run exploration test** to verify CSRF protection works

---

## References

- **Requirements:** 3.9, 3.10
- **Design Document:** `.kiro/specs/system-critical-bugfixes/design.md` (Bug 17)
- **Task List:** `.kiro/specs/system-critical-bugfixes/tasks.md` (Task 17.2)
- **Exploration Test:** `axios.bug17.test.js`
- **Exploration Results:** `BUG17_EXPLORATION_RESULTS.md`

---

**Test Created By:** Kiro AI Agent  
**Test Type:** Preservation Tests (Regression Prevention)  
**Expected Outcome:** All tests PASS on UNFIXED code  
**Purpose:** Establish baseline behavior before CSRF implementation
