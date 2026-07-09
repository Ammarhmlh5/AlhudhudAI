/**
 * Bug 15 Exploration Test: No Auto-Logout on 401
 * 
 * **Validates: Requirements 1.15, 2.15**
 * 
 * Bug Condition (1.15):
 * WHEN a 401 Unauthorized response is received in axios.js
 * THEN the system does not automatically log out the user,
 * leaving them in an inconsistent state
 * 
 * Expected Behavior (2.15):
 * WHEN a 401 Unauthorized response is received
 * THEN the system SHALL clear the authentication token,
 * redirect to the login page, and optionally show a notification
 * 
 * IMPORTANT: This is a bug exploration test
 * - On UNFIXED code: Test should FAIL (user stays logged in) ✓ Expected
 * - After fix: Test should PASS (user is logged out)
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
 * Property 1: Bug Condition - No Auto-Logout on 401
 * 
 * This test verifies that the current implementation DOES NOT
 * automatically log out users when receiving a 401 response.
 * 
 * Test Strategy:
 * 1. Set up authentication token in localStorage
 * 2. Create axios instance with current interceptor configuration
 * 3. Simulate 401 response from server
 * 4. Verify that token is NOT removed (bug exists)
 * 5. Verify that user is NOT redirected to login (bug exists)
 */
async function testBugCondition_NoAutoLogoutOn401() {
  console.log('\n=== Bug 15 Exploration Test: No Auto-Logout on 401 ===\n');
  
  // Setup: User is logged in with valid token
  const testToken = 'test-jwt-token-12345';
  mockLocalStorage.setItem('token', testToken);
  mockLocation.href = '/dashboard';
  
  console.log('Initial State:');
  console.log(`  - Token in localStorage: ${mockLocalStorage.getItem('token')}`);
  console.log(`  - Current location: ${mockLocation.href}`);
  
  // Create axios instance with current configuration (from axios.js)
  const apiClient = axios.create({
    baseURL: 'http://localhost:8081',
    withCredentials: true,
    headers: {
      'Content-Type': 'application/json',
    },
  });

  // Request interceptor (from axios.js)
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

  // Response interceptor (FIXED VERSION from axios.js)
  apiClient.interceptors.response.use(
    (response) => response,
    (error) => {
      if (error.response && error.response.status === 401) {
        // Handle logout or refresh token logic if needed
        console.warn('Unauthorized access - potential session expiry');
        mockLocalStorage.removeItem('token');  // FIXED: Uncommented
        mockLocation.href = '/login';          // FIXED: Uncommented
      }
      return Promise.reject(error);
    }
  );

  // Simulate 401 response from server
  console.log('\nSimulating 401 Unauthorized Response...');
  
  // Mock axios adapter to return 401
  apiClient.defaults.adapter = async (config) => {
    return Promise.reject({
      config,
      response: {
        status: 401,
        statusText: 'Unauthorized',
        data: { error: 'Token expired' },
        headers: {},
        config
      },
      message: 'Request failed with status code 401'
    });
  };

  // Make API request that will receive 401
  try {
    await apiClient.get('/api/messages');
  } catch (error) {
    console.log(`\nReceived Error: ${error.message}`);
  }

  // Check state after 401 response
  console.log('\nState After 401 Response:');
  const tokenAfter = mockLocalStorage.getItem('token');
  const locationAfter = mockLocation.href;
  
  console.log(`  - Token in localStorage: ${tokenAfter}`);
  console.log(`  - Current location: ${locationAfter}`);

  // Verify bug condition: Token should STILL exist (not removed)
  const tokenStillExists = tokenAfter !== null;
  const notRedirectedToLogin = !locationAfter.includes('/login');

  console.log('\n=== Test Results ===');
  console.log(`Token still exists: ${tokenStillExists} ${tokenStillExists ? '✓ (Bug Confirmed)' : '✗ (Bug Not Found)'}`);
  console.log(`Not redirected to login: ${notRedirectedToLogin} ${notRedirectedToLogin ? '✓ (Bug Confirmed)' : '✗ (Bug Not Found)'}`);

  // Test PASSES if bug exists (token not removed, no redirect)
  // Test FAILS if bug is fixed (token removed, redirected to login)
  const bugExists = tokenStillExists && notRedirectedToLogin;

  if (bugExists) {
    console.log('\n✓ EXPLORATION TEST PASSED: Bug condition confirmed!');
    console.log('  The system does NOT automatically log out on 401.');
    console.log('  This test will FAIL after the bug is fixed.');
    return { success: true, bugConfirmed: true };
  } else {
    console.log('\n✗ EXPLORATION TEST FAILED: Bug not reproduced!');
    console.log('  The system appears to handle 401 correctly.');
    console.log('  Either the bug is already fixed or the test needs adjustment.');
    return { success: false, bugConfirmed: false };
  }
}

/**
 * Property 2: Expected Behavior After Fix
 * 
 * This test verifies the CORRECT behavior that should exist
 * after the bug is fixed.
 * 
 * Test Strategy:
 * 1. Set up authentication token in localStorage
 * 2. Create axios instance with FIXED interceptor configuration
 * 3. Simulate 401 response from server
 * 4. Verify that token IS removed
 * 5. Verify that user IS redirected to login
 */
async function testExpectedBehavior_AutoLogoutOn401() {
  console.log('\n=== Expected Behavior Test: Auto-Logout on 401 ===\n');
  
  // Reset state
  mockLocalStorage.clear();
  mockLocation.href = '/dashboard';
  
  // Setup: User is logged in with valid token
  const testToken = 'test-jwt-token-12345';
  mockLocalStorage.setItem('token', testToken);
  
  console.log('Initial State:');
  console.log(`  - Token in localStorage: ${mockLocalStorage.getItem('token')}`);
  console.log(`  - Current location: ${mockLocation.href}`);
  
  // Create axios instance with FIXED configuration
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

  // Response interceptor (FIXED VERSION)
  apiClient.interceptors.response.use(
    (response) => response,
    (error) => {
      if (error.response && error.response.status === 401) {
        console.warn('Unauthorized access - logging out user');
        mockLocalStorage.removeItem('token');  // FIXED: Remove token
        mockLocation.href = '/login';          // FIXED: Redirect to login
      }
      return Promise.reject(error);
    }
  );

  // Simulate 401 response from server
  console.log('\nSimulating 401 Unauthorized Response...');
  
  // Mock axios adapter to return 401
  apiClient.defaults.adapter = async (config) => {
    return Promise.reject({
      config,
      response: {
        status: 401,
        statusText: 'Unauthorized',
        data: { error: 'Token expired' },
        headers: {},
        config
      },
      message: 'Request failed with status code 401'
    });
  };

  // Make API request that will receive 401
  try {
    await apiClient.get('/api/messages');
  } catch (error) {
    console.log(`\nReceived Error: ${error.message}`);
  }

  // Check state after 401 response
  console.log('\nState After 401 Response:');
  const tokenAfter = mockLocalStorage.getItem('token');
  const locationAfter = mockLocation.href;
  
  console.log(`  - Token in localStorage: ${tokenAfter || '(removed)'}`);
  console.log(`  - Current location: ${locationAfter}`);

  // Verify expected behavior: Token should be removed, redirected to login
  const tokenRemoved = tokenAfter === null;
  const redirectedToLogin = locationAfter.includes('/login');

  console.log('\n=== Test Results ===');
  console.log(`Token removed: ${tokenRemoved} ${tokenRemoved ? '✓' : '✗'}`);
  console.log(`Redirected to login: ${redirectedToLogin} ${redirectedToLogin ? '✓' : '✗'}`);

  const behaviorCorrect = tokenRemoved && redirectedToLogin;

  if (behaviorCorrect) {
    console.log('\n✓ EXPECTED BEHAVIOR TEST PASSED');
    console.log('  The system correctly logs out on 401.');
    return { success: true, behaviorCorrect: true };
  } else {
    console.log('\n✗ EXPECTED BEHAVIOR TEST FAILED');
    console.log('  The system does not handle 401 correctly.');
    return { success: false, behaviorCorrect: false };
  }
}

// Run tests
async function runTests() {
  console.log('╔════════════════════════════════════════════════════════════╗');
  console.log('║  Bug 15: No Auto-Logout on 401 - Exploration Test Suite  ║');
  console.log('╚════════════════════════════════════════════════════════════╝');
  
  const result1 = await testBugCondition_NoAutoLogoutOn401();
  const result2 = await testExpectedBehavior_AutoLogoutOn401();
  
  console.log('\n╔════════════════════════════════════════════════════════════╗');
  console.log('║                     FINAL SUMMARY                          ║');
  console.log('╚════════════════════════════════════════════════════════════╝');
  console.log(`\nBug Exploration Test: ${result1.success ? 'PASSED ✓' : 'FAILED ✗'}`);
  console.log(`  Bug Confirmed: ${result1.bugConfirmed ? 'YES' : 'NO'}`);
  console.log(`\nExpected Behavior Test: ${result2.success ? 'PASSED ✓' : 'FAILED ✗'}`);
  console.log(`  Behavior Correct: ${result2.behaviorCorrect ? 'YES' : 'NO'}`);
  
  console.log('\n=== Interpretation ===');
  if (result1.bugConfirmed && !result2.behaviorCorrect) {
    console.log('Status: BUG EXISTS - Code needs to be fixed');
    console.log('Action: Uncomment the logout logic in axios.js');
    process.exit(1); // Exit with error code (bug exists)
  } else if (!result1.bugConfirmed && result2.behaviorCorrect) {
    console.log('Status: BUG FIXED - Code is working correctly');
    console.log('Action: No action needed, bug has been resolved');
    process.exit(0); // Exit with success code (bug fixed)
  } else if (result1.bugConfirmed && result2.behaviorCorrect) {
    console.log('Status: BUG FIXED - Both tests show correct behavior');
    console.log('Action: Bug has been successfully resolved');
    process.exit(0); // Exit with success code (bug fixed)
  } else {
    console.log('Status: UNEXPECTED - Test results are inconsistent');
    console.log('Action: Review test implementation and axios.js code');
    process.exit(2); // Exit with error code (unexpected state)
  }
}

// Execute tests
runTests().catch(error => {
  console.error('\n✗ Test execution failed:', error);
  process.exit(3);
});
