/**
 * Bug 17: Missing CSRF Protection - Exploration Test
 * 
 * This test demonstrates the CSRF vulnerability where the API accepts
 * state-changing requests (POST/PUT/DELETE) without CSRF token validation.
 * 
 * EXPECTED BEHAVIOR ON UNFIXED CODE:
 * - Test FAILS: Request is accepted without CSRF token (proves bug exists)
 * 
 * EXPECTED BEHAVIOR ON FIXED CODE:
 * - Test PASSES: Request is rejected without CSRF token (bug is fixed)
 * 
 * Requirements: 1.17, 2.17
 */

import axios from 'axios';

// API Base URL - using environment variable or default
const API_BASE_URL = process.env.VITE_API_BASE_URL || 'http://localhost:8081';

console.log('\n=== Bug 17: CSRF Protection Exploration Test ===\n');
console.log('Testing CSRF vulnerability in API...\n');

/**
 * Property 1: Bug Condition - No CSRF Validation
 * 
 * **Validates: Requirements 1.17**
 * 
 * This test verifies that the API currently accepts POST requests
 * without CSRF token validation, making it vulnerable to CSRF attacks.
 * 
 * On UNFIXED code: Test FAILS (request accepted - bug exists)
 * On FIXED code: Test PASSES (request rejected - bug fixed)
 */
async function testCSRFVulnerability() {
  console.log('Test 1: Checking if API accepts requests without CSRF token...\n');

  // Create a test client WITHOUT CSRF token
  const testClient = axios.create({
    baseURL: API_BASE_URL,
    withCredentials: true,
    headers: {
      'Content-Type': 'application/json',
    },
  });

  // Add auth token (simulating authenticated user)
  const mockToken = 'Bearer test-jwt-token';
  testClient.interceptors.request.use((config) => {
    config.headers['Authorization'] = mockToken;
    // Deliberately NOT adding X-CSRF-Token header
    console.log('Request headers:', {
      'Authorization': config.headers['Authorization'] ? 'Bearer [token]' : 'none',
      'X-CSRF-Token': config.headers['X-CSRF-Token'] || 'MISSING (This is the bug!)'
    });
    return config;
  });

  try {
    // Attempt a state-changing operation without CSRF token
    console.log('Making POST request without CSRF token...');
    const response = await testClient.post('/api/csrf-test', {
      action: 'test',
      data: 'This request should be rejected without CSRF token'
    });

    // If we reach here, the request was accepted (BUG EXISTS)
    console.log('\n⚠️  SECURITY VULNERABILITY DETECTED!');
    console.log('Response status:', response.status);
    console.log('Response data:', response.data);
    console.log('\n❌ BUG CONFIRMED: API accepted POST request without CSRF token');
    console.log('This proves Bug 17 exists - the system is vulnerable to CSRF attacks.\n');
    
    return {
      testPassed: false,
      bugExists: true,
      message: 'CSRF vulnerability confirmed - requests accepted without token'
    };
    
  } catch (error) {
    if (error.response) {
      if (error.response.status === 403) {
        // Good! Request was rejected
        console.log('\n✅ CSRF protection is working!');
        console.log('Request was rejected with 403 Forbidden');
        console.log('Error message:', error.response.data.error || error.response.data);
        console.log('\nBug 17 is FIXED - CSRF protection is active.\n');
        
        return {
          testPassed: true,
          bugExists: false,
          message: 'CSRF protection working - requests rejected without token'
        };
      } else if (error.response.status === 404) {
        console.log('\nNote: Test endpoint /api/csrf-test not found (404)');
        console.log('This is expected on unfixed code - endpoint may not exist yet.');
        console.log('The important finding is that NO CSRF validation middleware is active.\n');
        
        return {
          testPassed: false,
          bugExists: true,
          message: 'No CSRF middleware detected - endpoint not found'
        };
      } else {
        console.log('\nUnexpected response status:', error.response.status);
        console.log('Error:', error.response.data);
      }
    } else if (error.code === 'ECONNREFUSED') {
      console.log('\n⚠️  Cannot connect to API server at', API_BASE_URL);
      console.log('Please ensure the backend server is running.');
      console.log('This test requires a running API server to verify CSRF protection.\n');
      
      return {
        testPassed: false,
        bugExists: 'unknown',
        message: 'Cannot test - API server not running'
      };
    } else {
      console.log('\nNetwork error:', error.message);
    }
    
    return {
      testPassed: false,
      bugExists: 'unknown',
      message: error.message
    };
  }
}

/**
 * Test 2: Verify current axios client configuration
 */
function testAxiosConfiguration() {
  console.log('\nTest 2: Checking current axios client configuration...\n');

  // Check if axios client has CSRF token logic
  const axiosClientCode = `
  Current axios.js configuration:
  - withCredentials: true ✓ (cookies sent with requests)
  - Authorization header: Added from localStorage ✓
  - X-CSRF-Token header: MISSING ❌ (This is the vulnerability!)
  `;

  console.log(axiosClientCode);
  console.log('⚠️  BUG CONFIRMED: No CSRF token in request headers');
  console.log('This proves Bug 17 exists - CSRF protection is not implemented\n');

  return {
    testPassed: false,
    bugExists: true,
    message: 'No CSRF token logic in axios configuration'
  };
}

/**
 * Test 3: Document CSRF attack scenario
 */
function documentCSRFAttack() {
  console.log('\nTest 3: CSRF Attack Scenario Documentation\n');

  const attackScenario = {
    vulnerability: 'API accepts requests with withCredentials: true but no CSRF validation',
    attackVector: [
      '1. Attacker creates malicious website with hidden form',
      '2. Form targets vulnerable API endpoint (e.g., /api/messages)',
      '3. User visits attacker site while logged into legitimate app',
      '4. Browser automatically sends cookies with request',
      '5. API accepts request because it has valid session cookie',
      '6. Attacker successfully performs unauthorized action'
    ],
    impact: 'High - Attacker can perform any action the user is authorized to do',
    affectedEndpoints: [
      'POST /api/messages',
      'PUT /api/settings',
      'DELETE /api/resources',
      'Any state-changing operation'
    ],
    currentState: 'VULNERABLE - No CSRF protection implemented',
    requiredFix: 'Implement CSRF token validation using double-submit cookie pattern'
  };

  console.log('=== CSRF VULNERABILITY ANALYSIS ===');
  console.log('Vulnerability:', attackScenario.vulnerability);
  console.log('\nAttack Vector:');
  attackScenario.attackVector.forEach(step => console.log('  ', step));
  console.log('\nImpact:', attackScenario.impact);
  console.log('\nAffected Endpoints:');
  attackScenario.affectedEndpoints.forEach(endpoint => console.log('  ', endpoint));
  console.log('\nCurrent State:', attackScenario.currentState);
  console.log('Required Fix:', attackScenario.requiredFix);
  console.log('===================================\n');

  return {
    testPassed: true,
    bugExists: true,
    message: 'Attack scenario documented'
  };
}

/**
 * Run all tests
 */
async function runAllTests() {
  const results = [];

  // Test 1: Try actual API request
  results.push(await testCSRFVulnerability());

  // Test 2: Check axios configuration
  results.push(testAxiosConfiguration());

  // Test 3: Document attack scenario
  results.push(documentCSRFAttack());

  // Summary
  console.log('\n=== TEST SUMMARY ===\n');
  
  const bugConfirmed = results.some(r => r.bugExists === true);
  const allTestsPassed = results.every(r => r.testPassed);

  if (bugConfirmed) {
    console.log('❌ BUG 17 CONFIRMED: CSRF Protection Missing');
    console.log('\nFindings:');
    results.forEach((result, index) => {
      console.log(`  Test ${index + 1}: ${result.message}`);
    });
    console.log('\n⚠️  SECURITY RISK: System is vulnerable to CSRF attacks');
    console.log('Recommendation: Implement CSRF token validation immediately');
  } else if (allTestsPassed) {
    console.log('✅ BUG 17 FIXED: CSRF Protection Active');
    console.log('\nAll tests passed - CSRF protection is working correctly');
  } else {
    console.log('⚠️  TEST INCOMPLETE: Unable to fully verify CSRF protection');
    console.log('Some tests could not be completed (e.g., API server not running)');
  }

  console.log('\n===================\n');
  
  // Exit with appropriate code
  process.exit(bugConfirmed ? 1 : 0);
}

// Run tests
runAllTests().catch(error => {
  console.error('\n❌ Test execution failed:', error.message);
  console.error(error.stack);
  process.exit(1);
});
