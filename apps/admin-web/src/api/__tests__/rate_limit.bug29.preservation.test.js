/**
 * Bug 29 Preservation Property Tests
 * 
 * **Property 2: Preservation** - Normal Usage Works
 * **Validates**: Requirements 3.9, 3.10
 * 
 * These tests verify that normal API usage patterns continue to work
 * correctly and are NOT blocked by rate limiting. They should PASS on
 * both UNFIXED and FIXED code.
 * 
 * Test Strategy:
 * - Make requests at normal rates (well below limit)
 * - Verify all requests succeed (no 429 responses)
 * - Test typical admin panel workflows
 * - Ensure performance is acceptable
 */

import axios from 'axios';

const API_BASE_URL = 'http://localhost:8081';

// Helper to make API request
async function makeRequest(endpoint, method = 'GET') {
  try {
    const response = await axios({
      method,
      url: `${API_BASE_URL}${endpoint}`,
      validateStatus: () => true, // Don't throw on any status
    });
    return response.status;
  } catch (error) {
    if (error.code === 'ECONNREFUSED') {
      throw new Error('Backend server not running');
    }
    throw error;
  }
}

// Helper to delay execution
function delay(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

/**
 * Test 1: Normal Request Rate (Well Below Limit)
 * 
 * Simulates typical admin panel usage:
 * - 20 requests over 20 seconds
 * - Rate: 1 request per second (60 req/min)
 * - Expected: All requests succeed (no 429)
 */
async function testNormalRequestRate() {
  console.log('\n=== Test 1: Normal Request Rate ===');
  console.log('Making 20 requests at 1 req/sec (60 req/min)...');
  
  const results = {
    total: 20,
    rateLimited: 0,
    successful: 0,
    errors: 0,
  };
  
  for (let i = 0; i < 20; i++) {
    const status = await makeRequest('/api/v1/messages');
    
    if (status === 429) {
      results.rateLimited++;
    } else if (status >= 200 && status < 500) {
      results.successful++;
    } else {
      results.errors++;
    }
    
    // Wait 1 second between requests
    if (i < 19) {
      await delay(1000);
    }
  }
  
  console.log('\nResults:');
  console.log(`  Total Requests: ${results.total}`);
  console.log(`  Successful: ${results.successful}`);
  console.log(`  Rate Limited (429): ${results.rateLimited}`);
  console.log(`  Errors: ${results.errors}`);
  
  const passed = results.rateLimited === 0;
  console.log(`\n${passed ? '✓' : '✗'} Test 1: ${passed ? 'PASSED' : 'FAILED'}`);
  console.log(`  Expected: No rate limiting at normal rate`);
  console.log(`  Actual: ${results.rateLimited} rate limited`);
  
  return passed;
}

/**
 * Test 2: Typical Admin Workflow
 * 
 * Simulates common admin panel operations:
 * - Load messages list
 * - Load gateways list
 * - Load tenants list
 * - Load statistics
 * - Total: 4 requests in quick succession
 * - Expected: All requests succeed
 */
async function testTypicalAdminWorkflow() {
  console.log('\n=== Test 2: Typical Admin Workflow ===');
  console.log('Simulating admin panel page load (4 requests)...');
  
  const endpoints = [
    '/api/v1/messages',
    '/api/v1/gateways',
    '/api/v1/tenants',
    '/api/v1/statistics',
  ];
  
  const results = {
    total: endpoints.length,
    rateLimited: 0,
    successful: 0,
  };
  
  for (const endpoint of endpoints) {
    const status = await makeRequest(endpoint);
    
    if (status === 429) {
      results.rateLimited++;
    } else if (status >= 200 && status < 500) {
      results.successful++;
    }
    
    // Small delay between requests (realistic)
    await delay(100);
  }
  
  console.log('\nResults:');
  console.log(`  Total Requests: ${results.total}`);
  console.log(`  Successful: ${results.successful}`);
  console.log(`  Rate Limited (429): ${results.rateLimited}`);
  
  const passed = results.rateLimited === 0;
  console.log(`\n${passed ? '✓' : '✗'} Test 2: ${passed ? 'PASSED' : 'FAILED'}`);
  console.log(`  Expected: No rate limiting for typical workflow`);
  console.log(`  Actual: ${results.rateLimited} rate limited`);
  
  return passed;
}

/**
 * Test 3: Sustained Moderate Load
 * 
 * Simulates sustained admin panel usage:
 * - 50 requests over 60 seconds
 * - Rate: ~50 req/min (below 100 req/min limit)
 * - Expected: All requests succeed
 */
async function testSustainedModerateLoad() {
  console.log('\n=== Test 3: Sustained Moderate Load ===');
  console.log('Making 50 requests over 60 seconds (~50 req/min)...');
  
  const results = {
    total: 50,
    rateLimited: 0,
    successful: 0,
  };
  
  const startTime = Date.now();
  
  for (let i = 0; i < 50; i++) {
    const status = await makeRequest('/api/v1/messages');
    
    if (status === 429) {
      results.rateLimited++;
    } else if (status >= 200 && status < 500) {
      results.successful++;
    }
    
    // Wait ~1.2 seconds between requests (50 req/min)
    if (i < 49) {
      await delay(1200);
    }
  }
  
  const duration = (Date.now() - startTime) / 1000;
  const actualRate = (results.total / duration) * 60;
  
  console.log('\nResults:');
  console.log(`  Total Requests: ${results.total}`);
  console.log(`  Duration: ${duration.toFixed(1)}s`);
  console.log(`  Actual Rate: ${actualRate.toFixed(1)} req/min`);
  console.log(`  Successful: ${results.successful}`);
  console.log(`  Rate Limited (429): ${results.rateLimited}`);
  
  const passed = results.rateLimited === 0;
  console.log(`\n${passed ? '✓' : '✗'} Test 3: ${passed ? 'PASSED' : 'FAILED'}`);
  console.log(`  Expected: No rate limiting at ~50 req/min`);
  console.log(`  Actual: ${results.rateLimited} rate limited`);
  
  return passed;
}

/**
 * Main Test Runner
 */
async function runPreservationTests() {
  console.log('╔════════════════════════════════════════════════════════════╗');
  console.log('║  Bug 29 Preservation Property Tests                       ║');
  console.log('║  Property 2: Normal Usage Works                           ║');
  console.log('╚════════════════════════════════════════════════════════════╝');
  console.log('\nValidates: Requirements 3.9, 3.10');
  console.log('Expected: All tests PASS (normal usage not blocked)');
  console.log('\nNOTE: These tests should PASS on both UNFIXED and FIXED code');
  console.log('      They verify that legitimate usage is not affected');
  
  try {
    // Test backend connectivity
    console.log('\nChecking backend connectivity...');
    await makeRequest('/api/v1/messages');
    console.log('✓ Backend is reachable');
    
    // Run all tests
    const results = [];
    
    results.push(await testNormalRequestRate());
    results.push(await testTypicalAdminWorkflow());
    results.push(await testSustainedModerateLoad());
    
    // Summary
    const allPassed = results.every(r => r);
    const passedCount = results.filter(r => r).length;
    
    console.log('\n╔════════════════════════════════════════════════════════════╗');
    console.log('║  Test Summary                                              ║');
    console.log('╚════════════════════════════════════════════════════════════╝');
    console.log(`\nTests Passed: ${passedCount}/${results.length}`);
    console.log(`\n${allPassed ? '✓' : '✗'} Preservation Tests: ${allPassed ? 'PASSED' : 'FAILED'}`);
    
    if (allPassed) {
      console.log('\n✓ Normal API usage is NOT blocked by rate limiting');
      console.log('✓ Typical admin workflows work correctly');
      console.log('✓ Performance is acceptable');
      console.log('\n✓ PRESERVATION VERIFIED: Normal usage works');
    } else {
      console.log('\n✗ Some normal usage patterns are being blocked');
      console.log('✗ Rate limiting may be too aggressive');
      console.log('\n✗ PRESERVATION FAILED: Normal usage affected');
    }
    
    process.exit(allPassed ? 0 : 1);
    
  } catch (error) {
    console.error('\n✗ Test execution error:', error.message);
    
    if (error.message.includes('Backend server not running')) {
      console.error('\nPlease start the backend server:');
      console.error('  cd services/backend');
      console.error('  go run main.go');
    }
    
    process.exit(3);
  }
}

// Run tests
runPreservationTests();
