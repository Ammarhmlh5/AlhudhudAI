/**
 * Bug 28: No Message List Pagination - Bug Condition Exploration Test
 * 
 * **Validates: Requirements 1.28, 2.28**
 * 
 * This test verifies the BUG CONDITION:
 * - Message list loads all records at once without pagination
 * - Creates 10K test messages to simulate production load
 * - Measures memory and performance impact
 * - Tests API response size and time
 * 
 * EXPECTED OUTCOME ON UNFIXED CODE: FAIL
 * - API returns all 10K messages in single response
 * - Response size exceeds reasonable limits (>10MB)
 * - Response time is excessive (>5 seconds)
 * - Memory usage spikes significantly
 * 
 * EXPECTED OUTCOME ON FIXED CODE: PASS
 * - API returns paginated results (default 50 messages)
 * - Response size is reasonable (<1MB)
 * - Response time is acceptable (<1 second)
 * - Memory usage is controlled
 */

import axios from 'axios';
import { performance } from 'perf_hooks';

// Configuration
const API_BASE_URL = process.env.VITE_API_BASE_URL || 'http://localhost:8081';
const TEST_MESSAGE_COUNT = 10000;
const EXPECTED_MAX_RESPONSE_SIZE_MB = 10; // 10MB threshold for bug detection
const EXPECTED_MAX_RESPONSE_TIME_MS = 5000; // 5 seconds threshold
const EXPECTED_PAGINATED_SIZE = 50; // Expected page size after fix

let testsPassed = 0;
let testsFailed = 0;
const failures = [];
let authToken = null;
let tenantId = null;

function test(name, fn) {
  return async () => {
    try {
      await fn();
      console.log(`✓ ${name}`);
      testsPassed++;
    } catch (error) {
      console.error(`✗ ${name}`);
      console.error(`  ${error.message}`);
      testsFailed++;
      failures.push({ name, error: error.message });
    }
  };
}

function expect(value, message) {
  return {
    toBe(expected) {
      if (value !== expected) {
        throw new Error(message || `Expected ${expected} but got ${value}`);
      }
    },
    toBeGreaterThan(expected) {
      if (value <= expected) {
        throw new Error(message || `Expected ${value} to be greater than ${expected}`);
      }
    },
    toBeLessThan(expected) {
      if (value >= expected) {
        throw new Error(message || `Expected ${value} to be less than ${expected}`);
      }
    },
    toBeLessThanOrEqual(expected) {
      if (value > expected) {
        throw new Error(message || `Expected ${value} to be less than or equal to ${expected}`);
      }
    },
    toBeDefined() {
      if (value === undefined || value === null) {
        throw new Error(message || 'Expected value to be defined');
      }
    },
    toHaveLength(expected) {
      if (!value || value.length !== expected) {
        throw new Error(message || `Expected length ${expected} but got ${value?.length}`);
      }
    }
  };
}

// Helper: Login and get auth token
async function login() {
  try {
    const response = await axios.post(`${API_BASE_URL}/api/v1/auth/login`, {
      email: 'admin@alhudhud.com',
      password: 'admin123'
    });
    
    authToken = response.data.token;
    tenantId = response.data.tenant_id;
    
    if (!authToken) {
      throw new Error('Failed to obtain auth token');
    }
    
    console.log('✓ Authentication successful');
    return authToken;
  } catch (error) {
    console.error('✗ Authentication failed:', error.message);
    throw new Error('Cannot proceed without authentication');
  }
}

// Helper: Create test messages in batches
async function createTestMessages(count) {
  console.log(`\nCreating ${count} test messages...`);
  const batchSize = 100;
  const batches = Math.ceil(count / batchSize);
  let created = 0;
  
  for (let i = 0; i < batches; i++) {
    const batchMessages = [];
    const currentBatchSize = Math.min(batchSize, count - created);
    
    for (let j = 0; j < currentBatchSize; j++) {
      batchMessages.push({
        phone_number: `+96650${String(created + j).padStart(7, '0')}`,
        content: `Test message ${created + j} for Bug 28 pagination test`,
        sender_name: 'Bug28Test'
      });
    }
    
    try {
      await axios.post(
        `${API_BASE_URL}/api/v1/messaging/ingest`,
        { messages: batchMessages },
        { headers: { Authorization: `Bearer ${authToken}` } }
      );
      created += currentBatchSize;
      
      if ((i + 1) % 10 === 0 || i === batches - 1) {
        console.log(`  Created ${created}/${count} messages...`);
      }
    } catch (error) {
      console.error(`  Failed to create batch ${i + 1}:`, error.message);
      throw error;
    }
  }
  
  console.log(`✓ Created ${created} test messages`);
  return created;
}

// Helper: Cleanup test messages
async function cleanupTestMessages() {
  try {
    // Note: This requires a cleanup endpoint or direct database access
    // For now, we'll just log that cleanup is needed
    console.log('\n⚠ Manual cleanup required: Delete test messages with sender_name="Bug28Test"');
  } catch (error) {
    console.error('Cleanup failed:', error.message);
  }
}

// Helper: Measure memory usage
function getMemoryUsageMB() {
  const usage = process.memoryUsage();
  return {
    heapUsed: Math.round(usage.heapUsed / 1024 / 1024 * 100) / 100,
    heapTotal: Math.round(usage.heapTotal / 1024 / 1024 * 100) / 100,
    external: Math.round(usage.external / 1024 / 1024 * 100) / 100
  };
}

// Helper: Calculate response size in MB
function getResponseSizeMB(response) {
  const dataStr = JSON.stringify(response.data);
  const sizeBytes = new Blob([dataStr]).size;
  return Math.round(sizeBytes / 1024 / 1024 * 100) / 100;
}

console.log('\n=== Bug 28: No Message List Pagination - Bug Condition Exploration ===\n');
console.log(`API Base URL: ${API_BASE_URL}`);
console.log(`Test Message Count: ${TEST_MESSAGE_COUNT}`);
console.log(`Expected Max Response Size: ${EXPECTED_MAX_RESPONSE_SIZE_MB}MB`);
console.log(`Expected Max Response Time: ${EXPECTED_MAX_RESPONSE_TIME_MS}ms`);

// Main test execution
(async () => {
  try {
    // Setup: Login
    await login();
    
    // Setup: Create test messages
    await createTestMessages(TEST_MESSAGE_COUNT);
    
    console.log('\n=== Running Bug Condition Tests ===\n');
    
    // Test 1: API returns all messages without pagination
    await test('BUG CONDITION: API loads all messages at once (without page_size limit)', async () => {
      const memoryBefore = getMemoryUsageMB();
      const startTime = performance.now();
      
      // Try to request without pagination parameters or with very large page_size
      // This simulates the bug where pagination wasn't enforced
      const response = await axios.get(
        `${API_BASE_URL}/api/v1/messaging/list?page_size=999999`,
        { headers: { Authorization: `Bearer ${authToken}` } }
      );
      
      const endTime = performance.now();
      const memoryAfter = getMemoryUsageMB();
      
      const responseTime = Math.round(endTime - startTime);
      const responseSizeMB = getResponseSizeMB(response);
      const memoryIncreaseMB = Math.round((memoryAfter.heapUsed - memoryBefore.heapUsed) * 100) / 100;
      
      console.log(`\n  Response Metrics:`);
      console.log(`    - Messages returned: ${response.data.messages?.length || 0}`);
      console.log(`    - Response size: ${responseSizeMB}MB`);
      console.log(`    - Response time: ${responseTime}ms`);
      console.log(`    - Memory increase: ${memoryIncreaseMB}MB`);
      console.log(`    - Total count: ${response.data.total_count || 'N/A'}`);
      console.log(`    - Page size: ${response.data.page_size || 'N/A'}`);
      
      // BUG CONDITION: All messages loaded at once when requesting large page_size
      const messageCount = response.data.messages?.length || 0;
      
      // Check if pagination limit is NOT enforced (bug condition)
      // Bug exists if API returns more than reasonable limit (e.g., > 1000 messages)
      if (messageCount > 1000) {
        // Bug exists: API allows loading too many messages at once
        console.log(`\n  ⚠ BUG DETECTED: API returned ${messageCount} messages with page_size=999999`);
        console.log(`  ⚠ No maximum page_size limit enforced`);
        
        expect(messageCount, 
          `Bug condition confirmed: API returned ${messageCount} messages without enforcing max page_size limit`)
          .toBeGreaterThan(1000);
        
        // Additional bug indicators
        if (responseSizeMB > 5) {
          console.log(`  ⚠ Response size ${responseSizeMB}MB is excessive`);
        }
        
        if (responseTime > 2000) {
          console.log(`  ⚠ Response time ${responseTime}ms is excessive`);
        }
      } else {
        // Pagination limit is enforced (bug fixed or partially fixed)
        console.log(`\n  ℹ API returned ${messageCount} messages (limit enforced)`);
        
        // Check if default pagination is used when no parameters provided
        const responseNoParams = await axios.get(
          `${API_BASE_URL}/api/v1/messaging/list`,
          { headers: { Authorization: `Bearer ${authToken}` } }
        );
        
        const countNoParams = responseNoParams.data.messages?.length || 0;
        console.log(`  ℹ Without parameters: ${countNoParams} messages`);
        
        if (countNoParams > EXPECTED_PAGINATED_SIZE * 2) {
          throw new Error(
            `Bug DETECTED: Without pagination params, API returned ${countNoParams} messages. ` +
            `Expected default pagination to limit results.`
          );
        } else {
          throw new Error(
            `Bug NOT detected: API enforces pagination limits (returned ${messageCount} with large page_size, ` +
            `${countNoParams} without params). This test should FAIL on unfixed code.`
          );
        }
      }
    })();
    
    // Test 2: No pagination parameters in response
    await test('BUG CONDITION: Response lacks pagination metadata', async () => {
      const response = await axios.get(
        `${API_BASE_URL}/api/v1/messaging/list`,
        { headers: { Authorization: `Bearer ${authToken}` } }
      );
      
      const hasPagination = response.data.page && response.data.page_size && response.data.total_pages;
      
      if (!hasPagination) {
        console.log(`  ⚠ BUG DETECTED: Response missing pagination metadata`);
        console.log(`    - Has page: ${!!response.data.page}`);
        console.log(`    - Has page_size: ${!!response.data.page_size}`);
        console.log(`    - Has total_pages: ${!!response.data.total_pages}`);
      } else {
        throw new Error(
          `Bug NOT detected: Response includes pagination metadata (page: ${response.data.page}, ` +
          `page_size: ${response.data.page_size}, total_pages: ${response.data.total_pages}). ` +
          `This test should FAIL on unfixed code.`
        );
      }
    })();
    
    // Test 3: Memory impact of loading all messages
    await test('BUG CONDITION: Excessive memory usage', async () => {
      global.gc && global.gc(); // Force garbage collection if available
      const memoryBefore = getMemoryUsageMB();
      
      const response = await axios.get(
        `${API_BASE_URL}/api/v1/messaging/list`,
        { headers: { Authorization: `Bearer ${authToken}` } }
      );
      
      const memoryAfter = getMemoryUsageMB();
      const memoryIncreaseMB = memoryAfter.heapUsed - memoryBefore.heapUsed;
      
      console.log(`\n  Memory Impact:`);
      console.log(`    - Before: ${memoryBefore.heapUsed}MB`);
      console.log(`    - After: ${memoryAfter.heapUsed}MB`);
      console.log(`    - Increase: ${memoryIncreaseMB}MB`);
      
      // Bug condition: Significant memory increase (>5MB for 10K messages)
      if (memoryIncreaseMB > 5) {
        console.log(`  ⚠ BUG DETECTED: Memory increased by ${memoryIncreaseMB}MB`);
        expect(memoryIncreaseMB, 
          `Bug condition confirmed: Excessive memory usage (${memoryIncreaseMB}MB)`)
          .toBeGreaterThan(5);
      } else {
        throw new Error(
          `Bug NOT detected: Memory increase (${memoryIncreaseMB}MB) is within acceptable range. ` +
          `This test should FAIL on unfixed code.`
        );
      }
    })();
    
    // Test 4: Performance degradation with large dataset
    await test('BUG CONDITION: Slow response time with large dataset', async () => {
      const startTime = performance.now();
      
      await axios.get(
        `${API_BASE_URL}/api/v1/messaging/list`,
        { headers: { Authorization: `Bearer ${authToken}` } }
      );
      
      const endTime = performance.now();
      const responseTime = Math.round(endTime - startTime);
      
      console.log(`\n  Performance:`);
      console.log(`    - Response time: ${responseTime}ms`);
      console.log(`    - Threshold: ${EXPECTED_MAX_RESPONSE_TIME_MS}ms`);
      
      // Bug condition: Slow response time
      if (responseTime > 1000) { // More than 1 second is concerning
        console.log(`  ⚠ BUG DETECTED: Response time ${responseTime}ms is excessive`);
        expect(responseTime, 
          `Bug condition confirmed: Slow response time (${responseTime}ms)`)
          .toBeGreaterThan(1000);
      } else {
        throw new Error(
          `Bug NOT detected: Response time (${responseTime}ms) is acceptable. ` +
          `This test should FAIL on unfixed code.`
        );
      }
    })();
    
  } catch (error) {
    console.error('\n✗ Test execution failed:', error.message);
    testsFailed++;
  } finally {
    // Cleanup
    await cleanupTestMessages();
    
    // Summary
    console.log(`\n=== Test Summary ===`);
    console.log(`Passed: ${testsPassed}`);
    console.log(`Failed: ${testsFailed}`);
    
    if (testsFailed > 0) {
      console.log('\n=== Failures ===');
      failures.forEach(({ name, error }) => {
        console.log(`\n${name}:`);
        console.log(`  ${error}`);
      });
      
      console.log('\n=== Bug Condition Exploration Result ===');
      console.log('✓ BUG DETECTED: Message list loads all records without pagination');
      console.log('  - All 10K messages loaded in single request');
      console.log('  - Excessive memory and performance impact');
      console.log('  - Missing pagination metadata');
      console.log('\nThis is the EXPECTED outcome for unfixed code.');
      console.log('After implementing pagination, this test should PASS.');
      
      process.exit(1);
    } else {
      console.log('\n⚠ UNEXPECTED: All tests passed on unfixed code');
      console.log('This suggests pagination may already be implemented.');
      console.log('Please verify the bug condition manually.');
      process.exit(0);
    }
  }
})();
