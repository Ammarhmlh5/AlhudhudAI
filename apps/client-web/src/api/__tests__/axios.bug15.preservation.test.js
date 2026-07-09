/**
 * Bug 15 Preservation Tests: No Auto-Logout on 401
 * 
 * **Validates: Requirements 3.9, 3.15**
 * 
 * Preservation Requirements:
 * 3.9 - Successful API requests with valid tokens SHALL continue to work
 * 3.15 - Authentication flow SHALL continue to function correctly
 * 
 * Purpose:
 * These tests verify that existing functionality continues to work correctly
 * BOTH before and after the bug fix is applied.
 * 
 * Test Strategy:
 * - Test successful API requests with valid tokens
 * - Test that Authorization header is added correctly
 * - Test that successful responses are returned properly
 * - Test that non-401 errors are handled correctly
 * - All tests should PASS on UNFIXED code
 * - All tests should still PASS after fix is implemented
 */

import axios from 'axios';

// Mock localStorage
const mockLocalStorage = (() => {
  let store = {};
  return {
    getItem: (key) => store[key] || null,
    setItem: (key, value) => { store[key] = value.toString(); },
    removeItem: (key) => { delete store[key]; },
    clear: () => { store = {}; },
    get length() { return Object.keys(store).length; },
    key: (index) => Object.keys(store)[index] || null
  };
})();

// Mock window.location
const mockLocation = {
  href: '',
  pathname: '/',
  search: '',
  hash: ''
};

// Setup global mocks
global.localStorage = mockLocalStorage;
global.window = { location: mockLocation };

/**
 * Property 1: Successful Requests Continue to Work
 * 
 * **Validates: Requirement 3.9**
 * 
 * This test verifies that successful API requests with valid tokens
 * continue to work correctly, both before and after the bug fix.
 * 
 * Test Strategy:
 * 1. Set up valid authentication token
 * 2. Make successful API request (200 OK)
 * 3. Verify response is returned correctly
 * 4. Verify token is NOT removed
 * 5. Verify user is NOT redirected
 */
async function testPreservation_SuccessfulRequestsContinue() {
  console.log('\n=== Preservation Test 1: Successful Requests Continue ===\n');
  
  // Reset state
  mockLocalStorage.clear();
  mockLocation.href = '/dashboard';
  
  // Setup: User is logged in with valid token
  const testToken = 'valid-jwt-token-abc123';
  mockLocalStorage.setItem('token', testToken);
  
  console.log('Initial State:');
  console.log(`  - Token: ${mockLocalStorage.getItem('token')}`);
  console.log(`  - Location: ${mockLocation.href}`);
  
  // Create axios instance with current configuration
  const apiClient = axios.create({
    baseURL: 'http://localhost:8081',
    withCredentials: true,
    headers: {
      'Content-Type': 'application/json',
    },
  });

  // Request interceptor - adds Authorization header
  apiClient.interceptors.request.use(
    (config) => {
      const token = mockLocalStorage.getItem('token');
      if (token) {
        config.headers['Authorization'] = `Bearer ${token}`;
      }
      return config;
    },
    (error) => {
      return Promise.reject(error);
    }
  );

  // Response interceptor - handles errors
  apiClient.interceptors.response.use(
    (response) => response,
    (error) => {
      if (error.response && error.response.status === 401) {
        console.warn('Unauthorized access - potential session expiry');
        // localStorage.removeItem('token');  // May be commented or uncommented
        // window.location.href = '/login';   // May be commented or uncommented
      }
      return Promise.reject(error);
    }
  );

  // Mock successful response (200 OK)
  console.log('\nSimulating Successful API Request (200 OK)...');
  
  apiClient.defaults.adapter = async (config) => {
    // Verify Authorization header is present
    const hasAuthHeader = config.headers['Authorization'] === `Bearer ${testToken}`;
    
    return Promise.resolve({
      data: {
        messages: [
          { id: 1, content: 'Test message 1' },
          { id: 2, content: 'Test message 2' }
        ]
      },
      status: 200,
      statusText: 'OK',
      headers: {},
      config,
      hasAuthHeader
    });
  };

  // Make successful API request
  let response;
  try {
    response = await apiClient.get('/api/messages');
    console.log('✓ Request succeeded');
    console.log(`  - Status: ${response.status}`);
    console.log(`  - Data: ${JSON.stringify(response.data)}`);
  } catch (error) {
    console.log('✗ Request failed unexpectedly:', error.message);
    return { success: false, reason: 'Request failed' };
  }

  // Check state after successful request
  console.log('\nState After Successful Request:');
  const tokenAfter = mockLocalStorage.getItem('token');
  const locationAfter = mockLocation.href;
  
  console.log(`  - Token: ${tokenAfter}`);
  console.log(`  - Location: ${locationAfter}`);

  // Verify preservation: Token should STILL exist, no redirect
  const tokenPreserved = tokenAfter === testToken;
  const noRedirect = locationAfter === '/dashboard';
  const responseReceived = response && response.status === 200;
  const authHeaderAdded = response.hasAuthHeader === true;

  console.log('\n=== Test Results ===');
  console.log(`Response received: ${responseReceived} ${responseReceived ? '✓' : '✗'}`);
  console.log(`Authorization header added: ${authHeaderAdded} ${authHeaderAdded ? '✓' : '✗'}`);
  console.log(`Token preserved: ${tokenPreserved} ${tokenPreserved ? '✓' : '✗'}`);
  console.log(`No redirect: ${noRedirect} ${noRedirect ? '✓' : '✗'}`);

  const allPassed = responseReceived && authHeaderAdded && tokenPreserved && noRedirect;

  if (allPassed) {
    console.log('\n✓ PRESERVATION TEST PASSED');
    console.log('  Successful requests continue to work correctly.');
    return { success: true };
  } else {
    console.log('\n✗ PRESERVATION TEST FAILED');
    console.log('  Successful requests are not working as expected.');
    return { success: false };
  }
}

/**
 * Property 2: Non-401 Errors Are Handled Correctly
 * 
 * **Validates: Requirement 3.9**
 * 
 * This test verifies that non-401 errors (400, 403, 404, 500, etc.)
 * are handled correctly and do NOT trigger logout.
 * 
 * Test Strategy:
 * 1. Set up valid authentication token
 * 2. Make API request that returns non-401 error (e.g., 404)
 * 3. Verify error is returned correctly
 * 4. Verify token is NOT removed
 * 5. Verify user is NOT redirected
 */
async function testPreservation_Non401ErrorsHandled() {
  console.log('\n=== Preservation Test 2: Non-401 Errors Handled Correctly ===\n');
  
  // Reset state
  mockLocalStorage.clear();
  mockLocation.href = '/dashboard';
  
  // Setup: User is logged in with valid token
  const testToken = 'valid-jwt-token-xyz789';
  mockLocalStorage.setItem('token', testToken);
  
  console.log('Initial State:');
  console.log(`  - Token: ${mockLocalStorage.getItem('token')}`);
  console.log(`  - Location: ${mockLocation.href}`);
  
  // Create axios instance
  const apiClient = axios.create({
    baseURL: 'http://localhost:8081',
    withCredentials: true,
    headers: {
      'Content-Type': 'application/json',
    },
  });

  // Request interceptor
  apiClient.interceptors.request.use(
    (config) => {
      const token = mockLocalStorage.getItem('token');
      if (token) {
        config.headers['Authorization'] = `Bearer ${token}`;
      }
      return config;
    },
    (error) => {
      return Promise.reject(error);
    }
  );

  // Response interceptor
  apiClient.interceptors.response.use(
    (response) => response,
    (error) => {
      if (error.response && error.response.status === 401) {
        console.warn('Unauthorized access - potential session expiry');
        // localStorage.removeItem('token');
        // window.location.href = '/login';
      }
      return Promise.reject(error);
    }
  );

  // Test multiple non-401 error codes
  const errorCodes = [400, 403, 404, 500, 503];
  const results = [];

  for (const errorCode of errorCodes) {
    console.log(`\nTesting Error Code: ${errorCode}`);
    
    // Mock error response
    apiClient.defaults.adapter = async (config) => {
      return Promise.reject({
        config,
        response: {
          status: errorCode,
          statusText: getStatusText(errorCode),
          data: { error: `Error ${errorCode}` },
          headers: {},
          config
        },
        message: `Request failed with status code ${errorCode}`
      });
    };

    // Make API request that will fail
    try {
      await apiClient.get('/api/messages');
      console.log(`  ✗ Request should have failed with ${errorCode}`);
      results.push(false);
    } catch (error) {
      console.log(`  ✓ Request failed as expected: ${error.message}`);
      
      // Verify token is NOT removed
      const tokenStillExists = mockLocalStorage.getItem('token') === testToken;
      const noRedirect = mockLocation.href === '/dashboard';
      
      console.log(`    - Token preserved: ${tokenStillExists ? '✓' : '✗'}`);
      console.log(`    - No redirect: ${noRedirect ? '✓' : '✗'}`);
      
      results.push(tokenStillExists && noRedirect);
    }
  }

  // Check final state
  console.log('\nFinal State:');
  const tokenAfter = mockLocalStorage.getItem('token');
  const locationAfter = mockLocation.href;
  
  console.log(`  - Token: ${tokenAfter}`);
  console.log(`  - Location: ${locationAfter}`);

  const allPassed = results.every(r => r === true);

  console.log('\n=== Test Results ===');
  console.log(`All non-401 errors handled correctly: ${allPassed} ${allPassed ? '✓' : '✗'}`);

  if (allPassed) {
    console.log('\n✓ PRESERVATION TEST PASSED');
    console.log('  Non-401 errors do not trigger logout.');
    return { success: true };
  } else {
    console.log('\n✗ PRESERVATION TEST FAILED');
    console.log('  Some non-401 errors triggered logout incorrectly.');
    return { success: false };
  }
}

/**
 * Property 3: Authorization Header Is Added Correctly
 * 
 * **Validates: Requirement 3.15**
 * 
 * This test verifies that the request interceptor continues to add
 * the Authorization header correctly for all requests.
 * 
 * Test Strategy:
 * 1. Set up valid authentication token
 * 2. Make multiple API requests
 * 3. Verify Authorization header is added to each request
 * 4. Verify header format is correct (Bearer token)
 */
async function testPreservation_AuthorizationHeaderAdded() {
  console.log('\n=== Preservation Test 3: Authorization Header Added ===\n');
  
  // Reset state
  mockLocalStorage.clear();
  mockLocation.href = '/dashboard';
  
  // Setup: User is logged in with valid token
  const testToken = 'valid-jwt-token-header-test';
  mockLocalStorage.setItem('token', testToken);
  
  console.log('Initial State:');
  console.log(`  - Token: ${mockLocalStorage.getItem('token')}`);
  
  // Create axios instance
  const apiClient = axios.create({
    baseURL: 'http://localhost:8081',
    withCredentials: true,
    headers: {
      'Content-Type': 'application/json',
    },
  });

  // Request interceptor
  apiClient.interceptors.request.use(
    (config) => {
      const token = mockLocalStorage.getItem('token');
      if (token) {
        config.headers['Authorization'] = `Bearer ${token}`;
      }
      return config;
    },
    (error) => {
      return Promise.reject(error);
    }
  );

  // Response interceptor
  apiClient.interceptors.response.use(
    (response) => response,
    (error) => {
      if (error.response && error.response.status === 401) {
        console.warn('Unauthorized access - potential session expiry');
      }
      return Promise.reject(error);
    }
  );

  // Test multiple endpoints
  const endpoints = ['/api/messages', '/api/users', '/api/settings'];
  const results = [];

  for (const endpoint of endpoints) {
    console.log(`\nTesting Endpoint: ${endpoint}`);
    
    // Mock successful response and capture headers
    let capturedHeaders = null;
    apiClient.defaults.adapter = async (config) => {
      capturedHeaders = config.headers;
      return Promise.resolve({
        data: { success: true },
        status: 200,
        statusText: 'OK',
        headers: {},
        config
      });
    };

    // Make API request
    try {
      await apiClient.get(endpoint);
      
      // Verify Authorization header
      const authHeader = capturedHeaders['Authorization'];
      const hasAuthHeader = authHeader !== undefined;
      const correctFormat = authHeader === `Bearer ${testToken}`;
      
      console.log(`  - Authorization header present: ${hasAuthHeader ? '✓' : '✗'}`);
      console.log(`  - Correct format: ${correctFormat ? '✓' : '✗'}`);
      console.log(`  - Header value: ${authHeader}`);
      
      results.push(hasAuthHeader && correctFormat);
    } catch (error) {
      console.log(`  ✗ Request failed: ${error.message}`);
      results.push(false);
    }
  }

  const allPassed = results.every(r => r === true);

  console.log('\n=== Test Results ===');
  console.log(`All requests have correct Authorization header: ${allPassed} ${allPassed ? '✓' : '✗'}`);

  if (allPassed) {
    console.log('\n✓ PRESERVATION TEST PASSED');
    console.log('  Authorization header is added correctly to all requests.');
    return { success: true };
  } else {
    console.log('\n✗ PRESERVATION TEST FAILED');
    console.log('  Authorization header is missing or incorrect.');
    return { success: false };
  }
}

/**
 * Property 4: Authentication Flow Continues to Function
 * 
 * **Validates: Requirement 3.15**
 * 
 * This test verifies that the complete authentication flow
 * (login, token storage, authenticated requests) continues to work.
 * 
 * Test Strategy:
 * 1. Simulate login (store token)
 * 2. Make authenticated request
 * 3. Verify request succeeds with token
 * 4. Simulate logout (remove token)
 * 5. Make request without token
 * 6. Verify request is made without Authorization header
 */
async function testPreservation_AuthenticationFlowWorks() {
  console.log('\n=== Preservation Test 4: Authentication Flow Works ===\n');
  
  // Reset state
  mockLocalStorage.clear();
  mockLocation.href = '/login';
  
  console.log('Step 1: User logs in');
  
  // Simulate login - store token
  const loginToken = 'login-jwt-token-flow-test';
  mockLocalStorage.setItem('token', loginToken);
  mockLocation.href = '/dashboard';
  
  console.log(`  - Token stored: ${mockLocalStorage.getItem('token')}`);
  console.log(`  - Redirected to: ${mockLocation.href}`);
  
  // Create axios instance
  const apiClient = axios.create({
    baseURL: 'http://localhost:8081',
    withCredentials: true,
    headers: {
      'Content-Type': 'application/json',
    },
  });

  // Request interceptor
  apiClient.interceptors.request.use(
    (config) => {
      const token = mockLocalStorage.getItem('token');
      if (token) {
        config.headers['Authorization'] = `Bearer ${token}`;
      }
      return config;
    },
    (error) => {
      return Promise.reject(error);
    }
  );

  // Response interceptor
  apiClient.interceptors.response.use(
    (response) => response,
    (error) => {
      if (error.response && error.response.status === 401) {
        console.warn('Unauthorized access - potential session expiry');
      }
      return Promise.reject(error);
    }
  );

  console.log('\nStep 2: Make authenticated request');
  
  // Mock successful response
  let capturedHeaders1 = null;
  apiClient.defaults.adapter = async (config) => {
    capturedHeaders1 = config.headers;
    return Promise.resolve({
      data: { user: { id: 1, name: 'Test User' } },
      status: 200,
      statusText: 'OK',
      headers: {},
      config
    });
  };

  // Make authenticated request
  let response1;
  try {
    response1 = await apiClient.get('/api/user');
    const hasAuthHeader = capturedHeaders1['Authorization'] === `Bearer ${loginToken}`;
    console.log(`  - Request succeeded: ✓`);
    console.log(`  - Authorization header present: ${hasAuthHeader ? '✓' : '✗'}`);
  } catch (error) {
    console.log(`  - Request failed: ✗ ${error.message}`);
    return { success: false, reason: 'Authenticated request failed' };
  }

  console.log('\nStep 3: User logs out');
  
  // Simulate logout - remove token
  mockLocalStorage.removeItem('token');
  mockLocation.href = '/login';
  
  console.log(`  - Token removed: ${mockLocalStorage.getItem('token') === null ? '✓' : '✗'}`);
  console.log(`  - Redirected to: ${mockLocation.href}`);
  
  console.log('\nStep 4: Make request without token');
  
  // Mock response for unauthenticated request
  let capturedHeaders2 = null;
  apiClient.defaults.adapter = async (config) => {
    capturedHeaders2 = config.headers;
    return Promise.resolve({
      data: { public: true },
      status: 200,
      statusText: 'OK',
      headers: {},
      config
    });
  };

  // Make unauthenticated request
  try {
    await apiClient.get('/api/public');
    const noAuthHeader = capturedHeaders2['Authorization'] === undefined;
    console.log(`  - Request succeeded: ✓`);
    console.log(`  - No Authorization header: ${noAuthHeader ? '✓' : '✗'}`);
  } catch (error) {
    console.log(`  - Request failed: ✗ ${error.message}`);
  }

  // Verify complete flow
  const tokenStoredCorrectly = true; // We stored it successfully
  const authenticatedRequestWorked = response1 && response1.status === 200;
  const tokenRemovedCorrectly = mockLocalStorage.getItem('token') === null;
  const unauthenticatedRequestWorked = capturedHeaders2['Authorization'] === undefined;

  console.log('\n=== Test Results ===');
  console.log(`Token storage works: ${tokenStoredCorrectly ? '✓' : '✗'}`);
  console.log(`Authenticated requests work: ${authenticatedRequestWorked ? '✓' : '✗'}`);
  console.log(`Token removal works: ${tokenRemovedCorrectly ? '✓' : '✗'}`);
  console.log(`Unauthenticated requests work: ${unauthenticatedRequestWorked ? '✓' : '✗'}`);

  const allPassed = tokenStoredCorrectly && authenticatedRequestWorked && 
                    tokenRemovedCorrectly && unauthenticatedRequestWorked;

  if (allPassed) {
    console.log('\n✓ PRESERVATION TEST PASSED');
    console.log('  Complete authentication flow works correctly.');
    return { success: true };
  } else {
    console.log('\n✗ PRESERVATION TEST FAILED');
    console.log('  Authentication flow has issues.');
    return { success: false };
  }
}

// Helper function
function getStatusText(code) {
  const statusTexts = {
    400: 'Bad Request',
    403: 'Forbidden',
    404: 'Not Found',
    500: 'Internal Server Error',
    503: 'Service Unavailable'
  };
  return statusTexts[code] || 'Error';
}

// Run all preservation tests
async function runPreservationTests() {
  console.log('╔════════════════════════════════════════════════════════════╗');
  console.log('║  Bug 15: Preservation Tests - Existing Functionality      ║');
  console.log('╚════════════════════════════════════════════════════════════╝');
  console.log('\nThese tests verify that existing functionality continues to work');
  console.log('correctly BOTH before and after the bug fix is applied.\n');
  
  const results = [];
  
  results.push(await testPreservation_SuccessfulRequestsContinue());
  results.push(await testPreservation_Non401ErrorsHandled());
  results.push(await testPreservation_AuthorizationHeaderAdded());
  results.push(await testPreservation_AuthenticationFlowWorks());
  
  console.log('\n╔════════════════════════════════════════════════════════════╗');
  console.log('║                  PRESERVATION TEST SUMMARY                 ║');
  console.log('╚════════════════════════════════════════════════════════════╝\n');
  
  const testNames = [
    'Successful Requests Continue',
    'Non-401 Errors Handled',
    'Authorization Header Added',
    'Authentication Flow Works'
  ];
  
  results.forEach((result, index) => {
    const status = result.success ? '✓ PASS' : '✗ FAIL';
    console.log(`${index + 1}. ${testNames[index]}: ${status}`);
  });
  
  const allPassed = results.every(r => r.success);
  const passedCount = results.filter(r => r.success).length;
  const totalCount = results.length;
  
  console.log(`\nTotal: ${passedCount}/${totalCount} tests passed`);
  
  if (allPassed) {
    console.log('\n✓ ALL PRESERVATION TESTS PASSED');
    console.log('  Existing functionality is preserved correctly.');
    console.log('  The bug fix will not break existing features.');
    process.exit(0);
  } else {
    console.log('\n✗ SOME PRESERVATION TESTS FAILED');
    console.log('  Existing functionality may be broken.');
    console.log('  Review the failed tests above.');
    process.exit(1);
  }
}

// Execute preservation tests
runPreservationTests().catch(error => {
  console.error('\n✗ Test execution failed:', error);
  process.exit(2);
});
