/**
 * Bug 29 Exploration Test: No API Rate Limiting
 * 
 * **Validates: Requirements 1.29, 2.29**
 * 
 * Bug Condition (1.29):
 * WHEN admin panel makes API requests
 * THEN the system has no rate limiting,
 * allowing potential abuse (e.g., 10,000 requests/second)
 * 
 * Expected Behavior (2.29):
 * WHEN admin panel makes API requests
 * THEN the system SHALL implement rate limiting
 * (e.g., 100 requests per minute per IP)
 * 
 * IMPORTANT: This is a bug exploration test
 * - On UNFIXED code: Test should FAIL (all requests accepted) ✓ Expected
 * - After fix: Test should PASS (requests limited with 429 responses)
 */

import axios from 'axios';

/**
 * Property 1: Bug Condition - No Rate Limiting
 * 
 * This test verifies that the current implementation DOES NOT
 * implement rate limiting on API endpoints.
 * 
 * Test Strategy:
 * 1. Make 1000 rapid requests to an API endpoint
 * 2. Count how many requests are accepted (200/201/etc)
 * 3. Count how many requests are rate limited (429)
 * 4. Verify that NO requests are rate limited (bug exists)
 * 
 * Expected on UNFIXED code:
 * - All 1000 requests accepted (or fail for other reasons like auth)
 * - Zero 429 (Too Many Requests) responses
 * - Test FAILS (bug confirmed)
 * 
 * Expected after FIX:
 * - Some requests accepted (up to rate limit)
 * - Many 429 responses after limit exceeded
 * - Test PASSES (bug fixed)
 */
async function testBugCondition_NoRateLimiting() {
  console.log('\n=== Bug 29 Exploration Test: No API Rate Limiting ===\n');
  
  const API_BASE_URL = 'http://localhost:8081';
  const TEST_ENDPOINT = '/api/v1/messages'; // Common admin panel endpoint
  const TOTAL_REQUESTS = 1000;
  const EXPECTED_RATE_LIMIT = 100; // Expected: 100 requests per minute
  
  console.log('Test Configuration:');
  console.log(`  - API Base URL: ${API_BASE_URL}`);
  console.log(`  - Test Endpoint: ${TEST_ENDPOINT}`);
  console.log(`  - Total Requests: ${TOTAL_REQUESTS}`);
  console.log(`  - Expected Rate Limit: ${EXPECTED_RATE_LIMIT} req/min`);
  
  // Create axios instance (no authentication for this test)
  const apiClient = axios.create({
    baseURL: API_BASE_URL,
    timeout: 5000,
    validateStatus: () => true, // Accept all status codes
  });

  console.log('\n=== Making Rapid Requests ===');
  console.log('Sending 1000 requests as fast as possible...\n');
  
  const startTime = Date.now();
  const results = {
    total: 0,
    accepted: 0,        // 2xx responses
    rateLimited: 0,     // 429 responses
    unauthorized: 0,    // 401 responses
    forbidden: 0,       // 403 responses
    serverError: 0,     // 5xx responses
    networkError: 0,    // Network failures
    other: 0,
  };

  // Make requests in parallel batches for speed
  const BATCH_SIZE = 100;
  const batches = Math.ceil(TOTAL_REQUESTS / BATCH_SIZE);
  
  for (let batch = 0; batch < batches; batch++) {
    const batchStart = batch * BATCH_SIZE;
    const batchEnd = Math.min(batchStart + BATCH_SIZE, TOTAL_REQUESTS);
    const batchSize = batchEnd - batchStart;
    
    const promises = [];
    for (let i = 0; i < batchSize; i++) {
      promises.push(
        apiClient.get(TEST_ENDPOINT)
          .then(response => {
            results.total++;
            
            if (response.status >= 200 && response.status < 300) {
              results.accepted++;
            } else if (response.status === 429) {
              results.rateLimited++;
            } else if (response.status === 401) {
              results.unauthorized++;
            } else if (response.status === 403) {
              results.forbidden++;
            } else if (response.status >= 500) {
              results.serverError++;
            } else {
              results.other++;
            }
          })
          .catch(error => {
            results.total++;
            
            if (error.response) {
              if (error.response.status === 429) {
                results.rateLimited++;
              } else if (error.response.status === 401) {
                results.unauthorized++;
              } else if (error.response.status === 403) {
                results.forbidden++;
              } else if (error.response.status >= 500) {
                results.serverError++;
              } else {
                results.other++;
              }
            } else {
              results.networkError++;
            }
          })
      );
    }
    
    await Promise.all(promises);
    
    // Progress indicator
    const progress = Math.round((results.total / TOTAL_REQUESTS) * 100);
    process.stdout.write(`\rProgress: ${progress}% (${results.total}/${TOTAL_REQUESTS})`);
  }
  
  const duration = Date.now() - startTime;
  const requestsPerSecond = Math.round((TOTAL_REQUESTS / duration) * 1000);
  
  console.log('\n\n=== Test Results ===');
  console.log(`Duration: ${duration}ms (${requestsPerSecond} req/sec)`);
  console.log(`\nResponse Breakdown:`);
  console.log(`  - Total Requests: ${results.total}`);
  console.log(`  - Accepted (2xx): ${results.accepted}`);
  console.log(`  - Rate Limited (429): ${results.rateLimited} ${results.rateLimited > 0 ? '✓' : '✗'}`);
  console.log(`  - Unauthorized (401): ${results.unauthorized}`);
  console.log(`  - Forbidden (403): ${results.forbidden}`);
  console.log(`  - Server Error (5xx): ${results.serverError}`);
  console.log(`  - Network Error: ${results.networkError}`);
  console.log(`  - Other: ${results.other}`);

  // Calculate rate limiting effectiveness
  const totalResponses = results.accepted + results.unauthorized + results.forbidden + results.other;
  const rateLimitingActive = results.rateLimited > 0;
  const rateLimitPercentage = (results.rateLimited / results.total) * 100;

  console.log('\n=== Analysis ===');
  console.log(`Rate Limiting Active: ${rateLimitingActive ? 'YES ✓' : 'NO ✗'}`);
  console.log(`Rate Limited Percentage: ${rateLimitPercentage.toFixed(1)}%`);
  
  if (rateLimitingActive) {
    console.log(`Requests Before Limit: ${totalResponses}`);
    console.log(`Requests After Limit: ${results.rateLimited}`);
  }

  // Determine bug status
  // Bug exists if NO rate limiting is detected
  const bugExists = !rateLimitingActive;

  console.log('\n=== Bug Status ===');
  if (bugExists) {
    console.log('✗ BUG CONFIRMED: No rate limiting detected!');
    console.log('  All requests were processed without rate limiting.');
    console.log('  The system is vulnerable to abuse.');
    console.log('\n  Expected Behavior:');
    console.log(`    - First ~${EXPECTED_RATE_LIMIT} requests: Accepted`);
    console.log(`    - Remaining requests: 429 Too Many Requests`);
    console.log('\n  Actual Behavior:');
    console.log(`    - All ${totalResponses} requests: Processed`);
    console.log(`    - Zero 429 responses`);
    console.log('\n✗ EXPLORATION TEST FAILED (Bug Exists)');
    console.log('  This test will PASS after the bug is fixed.');
    return { success: false, bugConfirmed: true, results };
  } else {
    console.log('✓ Rate limiting is ACTIVE!');
    console.log(`  ${results.rateLimited} requests were rate limited (429).`);
    console.log('  The system is protected against abuse.');
    console.log('\n✓ EXPLORATION TEST PASSED (Bug Fixed)');
    console.log('  Rate limiting is working correctly.');
    return { success: true, bugConfirmed: false, results };
  }
}

/**
 * Property 2: Expected Behavior After Fix
 * 
 * This test verifies the CORRECT behavior that should exist
 * after rate limiting is implemented.
 * 
 * Test Strategy:
 * 1. Make requests at a controlled rate (within limit)
 * 2. Verify all requests are accepted
 * 3. Make burst of requests exceeding limit
 * 4. Verify 429 responses are returned
 */
async function testExpectedBehavior_RateLimitingActive() {
  console.log('\n=== Expected Behavior Test: Rate Limiting Active ===\n');
  
  const API_BASE_URL = 'http://localhost:8081';
  const TEST_ENDPOINT = '/api/v1/messages';
  const RATE_LIMIT = 100; // Expected: 100 requests per minute
  const BURST_SIZE = 150; // Exceed the limit
  
  console.log('Test Configuration:');
  console.log(`  - API Base URL: ${API_BASE_URL}`);
  console.log(`  - Test Endpoint: ${TEST_ENDPOINT}`);
  console.log(`  - Expected Rate Limit: ${RATE_LIMIT} req/min`);
  console.log(`  - Burst Size: ${BURST_SIZE} requests`);
  
  const apiClient = axios.create({
    baseURL: API_BASE_URL,
    timeout: 5000,
    validateStatus: () => true,
  });

  console.log('\n=== Phase 1: Controlled Rate (Within Limit) ===');
  console.log('Making 50 requests at controlled pace...\n');
  
  let phase1Results = { accepted: 0, rateLimited: 0, other: 0 };
  
  for (let i = 0; i < 50; i++) {
    try {
      const response = await apiClient.get(TEST_ENDPOINT);
      if (response.status >= 200 && response.status < 300) {
        phase1Results.accepted++;
      } else if (response.status === 429) {
        phase1Results.rateLimited++;
      } else {
        phase1Results.other++;
      }
    } catch (error) {
      if (error.response?.status === 429) {
        phase1Results.rateLimited++;
      } else {
        phase1Results.other++;
      }
    }
    
    // Small delay to stay within rate limit
    await new Promise(resolve => setTimeout(resolve, 50));
  }
  
  console.log('Phase 1 Results:');
  console.log(`  - Accepted: ${phase1Results.accepted}`);
  console.log(`  - Rate Limited: ${phase1Results.rateLimited}`);
  console.log(`  - Other: ${phase1Results.other}`);

  console.log('\n=== Phase 2: Burst (Exceeding Limit) ===');
  console.log(`Making ${BURST_SIZE} rapid requests...\n`);
  
  let phase2Results = { accepted: 0, rateLimited: 0, other: 0 };
  
  const promises = [];
  for (let i = 0; i < BURST_SIZE; i++) {
    promises.push(
      apiClient.get(TEST_ENDPOINT)
        .then(response => {
          if (response.status >= 200 && response.status < 300) {
            phase2Results.accepted++;
          } else if (response.status === 429) {
            phase2Results.rateLimited++;
          } else {
            phase2Results.other++;
          }
        })
        .catch(error => {
          if (error.response?.status === 429) {
            phase2Results.rateLimited++;
          } else {
            phase2Results.other++;
          }
        })
    );
  }
  
  await Promise.all(promises);
  
  console.log('Phase 2 Results:');
  console.log(`  - Accepted: ${phase2Results.accepted}`);
  console.log(`  - Rate Limited: ${phase2Results.rateLimited} ${phase2Results.rateLimited > 0 ? '✓' : '✗'}`);
  console.log(`  - Other: ${phase2Results.other}`);

  // Verify expected behavior
  const phase1Successful = phase1Results.rateLimited === 0;
  const phase2RateLimited = phase2Results.rateLimited > 0;
  const behaviorCorrect = phase1Successful && phase2RateLimited;

  console.log('\n=== Test Results ===');
  console.log(`Phase 1 (Controlled): ${phase1Successful ? 'PASS ✓' : 'FAIL ✗'}`);
  console.log(`  Expected: No rate limiting`);
  console.log(`  Actual: ${phase1Results.rateLimited} rate limited`);
  console.log(`\nPhase 2 (Burst): ${phase2RateLimited ? 'PASS ✓' : 'FAIL ✗'}`);
  console.log(`  Expected: Rate limiting active`);
  console.log(`  Actual: ${phase2Results.rateLimited} rate limited`);

  if (behaviorCorrect) {
    console.log('\n✓ EXPECTED BEHAVIOR TEST PASSED');
    console.log('  Rate limiting is working correctly.');
    return { success: true, behaviorCorrect: true };
  } else {
    console.log('\n✗ EXPECTED BEHAVIOR TEST FAILED');
    console.log('  Rate limiting is not working as expected.');
    return { success: false, behaviorCorrect: false };
  }
}

// Run tests
async function runTests() {
  console.log('╔════════════════════════════════════════════════════════════╗');
  console.log('║  Bug 29: No API Rate Limiting - Exploration Test Suite   ║');
  console.log('╚════════════════════════════════════════════════════════════╝');
  
  const result1 = await testBugCondition_NoRateLimiting();
  
  console.log('\n' + '═'.repeat(64) + '\n');
  
  const result2 = await testExpectedBehavior_RateLimitingActive();
  
  console.log('\n╔════════════════════════════════════════════════════════════╗');
  console.log('║                     FINAL SUMMARY                          ║');
  console.log('╚════════════════════════════════════════════════════════════╝');
  console.log(`\nBug Exploration Test: ${result1.success ? 'PASSED ✓' : 'FAILED ✗'}`);
  console.log(`  Bug Confirmed: ${result1.bugConfirmed ? 'YES' : 'NO'}`);
  console.log(`  Rate Limited Responses: ${result1.results.rateLimited}`);
  console.log(`\nExpected Behavior Test: ${result2.success ? 'PASSED ✓' : 'FAILED ✗'}`);
  console.log(`  Behavior Correct: ${result2.behaviorCorrect ? 'YES' : 'NO'}`);
  
  console.log('\n=== Interpretation ===');
  if (result1.bugConfirmed && !result2.behaviorCorrect) {
    console.log('Status: BUG EXISTS - No rate limiting implemented');
    console.log('Action: Implement rate limiting middleware');
    console.log('        - Install rate limiting library');
    console.log('        - Configure 100 requests per minute per IP');
    console.log('        - Apply to /api routes');
    console.log('        - Return 429 on limit exceeded');
    process.exit(1); // Exit with error code (bug exists)
  } else if (!result1.bugConfirmed && result2.behaviorCorrect) {
    console.log('Status: BUG FIXED - Rate limiting is active');
    console.log('Action: No action needed, bug has been resolved');
    process.exit(0); // Exit with success code (bug fixed)
  } else if (!result1.bugConfirmed && !result2.behaviorCorrect) {
    console.log('Status: PARTIAL FIX - Rate limiting exists but not working correctly');
    console.log('Action: Review rate limiting configuration');
    process.exit(1); // Exit with error code (needs adjustment)
  } else {
    console.log('Status: UNEXPECTED - Test results are inconsistent');
    console.log('Action: Review test implementation and rate limiting code');
    process.exit(2); // Exit with error code (unexpected state)
  }
}

// Execute tests
runTests().catch(error => {
  console.error('\n✗ Test execution failed:', error);
  process.exit(3);
});
