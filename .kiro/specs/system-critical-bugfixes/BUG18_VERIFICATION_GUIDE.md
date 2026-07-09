# Bug 18: Verification Guide

## Quick Verification Steps

### 1. Backend Verification

#### Check Login Response
```bash
# Login and check response
curl -X POST http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password"}' \
  -c cookies.txt -v

# Expected:
# - Response body should NOT contain "token" or "refresh_token" fields
# - Set-Cookie headers should include "access_token" and "refresh_token"
# - Cookies should have HttpOnly flag
```

#### Check Cookie Attributes
```bash
# Check cookies file
cat cookies.txt

# Expected attributes:
# - HttpOnly
# - Secure (in production)
# - SameSite=Strict
# - Path=/
```

#### Test Authenticated Request
```bash
# Make authenticated request using cookies
curl http://localhost:8081/api/v1/auth/me \
  -b cookies.txt

# Expected:
# - Request succeeds with user data
# - No Authorization header needed
```

### 2. Frontend Verification (Browser DevTools)

#### Check localStorage
```javascript
// Open browser console
localStorage.getItem('token')
// Expected: null (no token in localStorage)
```

#### Check Cookies
```javascript
// Open DevTools → Application → Cookies
// Expected:
// - access_token cookie exists
// - HttpOnly flag is checked ✓
// - Secure flag is checked (in production) ✓
// - SameSite: Strict ✓
```

#### Check Network Requests
```javascript
// Open DevTools → Network → Select any API request
// Headers tab:
// - Cookie header should include access_token
// - No Authorization header (unless API client)
```

#### Test XSS Protection
```javascript
// Try to access token via JavaScript
document.cookie
// Expected: access_token should NOT be visible in the output
// (HttpOnly cookies are not accessible to JavaScript)
```

### 3. Authentication Flow Testing

#### Test Login
1. Open login page
2. Enter credentials
3. Submit form
4. **Expected:**
   - Redirect to dashboard
   - No token in localStorage
   - access_token cookie set
   - User authenticated

#### Test Page Reload
1. Login successfully
2. Reload page
3. **Expected:**
   - User remains authenticated
   - No re-login required
   - Cookie persists

#### Test Logout
1. Login successfully
2. Click logout
3. **Expected:**
   - Redirect to login page
   - Cookies cleared
   - Cannot access protected routes

#### Test 401 Handling
1. Login successfully
2. Wait for token expiration (or manually delete cookie)
3. Make API request
4. **Expected:**
   - 401 response
   - Automatic redirect to login
   - User prompted to re-authenticate

### 4. Security Testing

#### XSS Protection Test
```javascript
// Attempt to steal token via XSS
console.log(localStorage.getItem('token')); // null
console.log(document.cookie); // access_token not visible

// Try to access via various methods
Object.keys(localStorage); // no 'token' key
document.cookie.split(';').find(c => c.includes('access_token')); // undefined
```

#### CSRF Protection Test
```bash
# Verify CSRF protection still works (Bug 17)
curl -X POST http://localhost:8081/api/v1/some-endpoint \
  -b cookies.txt \
  -H "Content-Type: application/json" \
  -d '{"data":"test"}'

# Expected:
# - 403 Forbidden (missing CSRF token)
# - CSRF protection still active
```

### 5. Cross-Browser Testing

Test in each browser:
- [ ] Chrome/Chromium
- [ ] Firefox
- [ ] Safari
- [ ] Edge
- [ ] Mobile Safari (iOS)
- [ ] Mobile Chrome (Android)

For each browser:
1. Login works
2. Token in httpOnly cookie
3. Authenticated requests work
4. Logout works
5. Session persists across reloads

### 6. API Client Testing

#### Test Authorization Header Fallback
```bash
# API clients can still use Authorization header
curl http://localhost:8081/api/v1/auth/me \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"

# Expected:
# - Request succeeds
# - Backward compatibility maintained
```

### 7. Production Verification

#### Check Secure Flag
```bash
# In production (HTTPS)
curl -X POST https://production.example.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password"}' \
  -c cookies.txt -v

# Expected:
# - Secure flag is set
# - Cookies only sent over HTTPS
```

## Common Issues & Solutions

### Issue: Token not sent in requests
**Solution:** Ensure `withCredentials: true` in axios config

### Issue: CORS error
**Solution:** Verify `AllowCredentials: true` in backend CORS config

### Issue: Cookie not set
**Solution:** Check domain and path settings in cookie configuration

### Issue: Token visible in JavaScript
**Solution:** Verify HttpOnly flag is set correctly

## Success Criteria

✅ All checks pass:
- [ ] No token in localStorage
- [ ] Token in httpOnly cookie
- [ ] JavaScript cannot access token
- [ ] Authentication flow works
- [ ] Logout clears cookies
- [ ] CSRF protection works
- [ ] Cross-browser compatible
- [ ] API clients still work

## Rollback Plan

If issues are found:

1. **Immediate Rollback:**
   ```bash
   git revert <commit-hash>
   ```

2. **Restore Previous Behavior:**
   - Backend: Send token in response body
   - Frontend: Store token in localStorage
   - Frontend: Add Authorization header

3. **Investigate Issues:**
   - Check error logs
   - Review browser console
   - Test in different environments

4. **Fix and Redeploy:**
   - Address identified issues
   - Re-test thoroughly
   - Deploy again

---

**Last Updated:** 2024
**Status:** Ready for Verification
**Priority:** Security (Critical)
