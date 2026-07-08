/**
 * Bug 28: No Message List Pagination - Preservation Property Tests
 * 
 * **Validates: Requirements 3.16**
 * 
 * These tests verify PRESERVATION properties:
 * - Message display continues to work correctly
 * - Message ordering is preserved (newest first)
 * - Filtering by status still works
 * - Message content is displayed correctly
 * - Initial load performance is maintained or improved
 * 
 * EXPECTED OUTCOME: PASS on both UNFIXED and FIXED code
 * These tests ensure pagination doesn't break existing functionality.
 */

import axios from 'axios';
import { performance } from 'perf_hooks';

// Configuration
const API_BASE_URL = process.env.VITE_API_BASE_URL || 'http://localhost:8081';

let testsPassed = 0;
let testsFailed = 0;
const failures = [];
let authToken = null;

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
    },
    toContain(substring) {
      if (!value || !value.includes(substring)) {
        throw new Error(message || `Expected value to contain "${substring}"`);
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

// Helper: Create a few test messages for preservation tests
async function createTestMessages(count = 10) {
  console.log(`\nCreating ${count} test messages for preservation tests...`);
  
  const messages = [];
  for (let i = 0; i < count; i++) {
    messages.push({
      phone_number: `+96650${String(i).padStart(7, '0')}`,
      content: `Preservation test message ${i} - ${new Date().toISOString()}`,
      sender_name: 'Bug28Preservation'
    });
  }
  
  try {
    await axios.post(
      `${API_BASE_URL}/api/v1/messaging/ingest`,
      { messages },
      { headers: { Authorization: `Bearer ${authToken}` } }
    );
    
    console.log(`✓ Created ${count} test messages`);
    return count;
  } catch (error) {
    console.error(`Failed to create test messages:`, error.message);
    throw error;
  }
}

console.log('\n=== Bug 28: Preservation Property Tests ===\n');
console.log(`API Base URL: ${API_BASE_URL}`);
console.log('Testing that message display functionality is preserved\n');

// Main test execution
(async () => {
  try {
    // Setup: Login
    await login();
    
    // Setup: Create some test messages
    await createTestMessages(10);
    
    // Wait a moment for messages to be processed
    await new Promise(resolve => setTimeout(resolve, 1000));
    
    console.log('\n=== Running Preservation Tests ===\n');
    
    // Property 1: Message Display Works
    await test('PRESERVATION: Message list API returns messages successfully', async () => {
      const response = await axios.get(
        `${API_BASE_URL}/api/v1/messaging/list`,
        { headers: { Authorization: `Bearer ${authToken}` } }
      );
      
      expect(response.status, 'API should return 200 OK').toBe(200);
      expect(response.data, 'Response should have data').toBeDefined();
      expect(response.data.messages, 'Response should have messages array').toBeDefined();
      
      console.log(`  ✓ API returned ${response.data.messages?.length || 0} messages`);
    })();
    
    // Property 2: Message Ordering Preserved (Newest First)
    await test('PRESERVATION: Messages are ordered by created_at DESC (newest first)', async () => {
      const response = await axios.get(
        `${API_BASE_URL}/api/v1/messaging/list?page=1&page_size=10`,
        { headers: { Authorization: `Bearer ${authToken}` } }
      );
      
      const messages = response.data.messages || [];
      
      if (messages.length < 2) {
        console.log('  ℹ Not enough messages to verify ordering (need at least 2)');
        return;
      }
      
      // Verify messages are ordered newest first
      for (let i = 0; i < messages.length - 1; i++) {
        const currentDate = new Date(messages[i].created_at);
        const nextDate = new Date(messages[i + 1].created_at);
        
        if (currentDate < nextDate) {
          throw new Error(
            `Message ordering broken: message at index ${i} (${messages[i].created_at}) ` +
            `is older than message at index ${i + 1} (${messages[i + 1].created_at})`
          );
        }
      }
      
      console.log(`  ✓ Verified ${messages.length} messages are ordered newest first`);
    })();
    
    // Property 3: Message Content Displayed Correctly
    await test('PRESERVATION: Message content fields are present and valid', async () => {
      const response = await axios.get(
        `${API_BASE_URL}/api/v1/messaging/list?page=1&page_size=5`,
        { headers: { Authorization: `Bearer ${authToken}` } }
      );
      
      const messages = response.data.messages || [];
      
      if (messages.length === 0) {
        console.log('  ℹ No messages to verify');
        return;
      }
      
      // Verify each message has required fields
      messages.forEach((msg, index) => {
        expect(msg.id, `Message ${index} should have id`).toBeDefined();
        expect(msg.phone_number, `Message ${index} should have phone_number`).toBeDefined();
        expect(msg.content, `Message ${index} should have content`).toBeDefined();
        expect(msg.status, `Message ${index} should have status`).toBeDefined();
        expect(msg.created_at, `Message ${index} should have created_at`).toBeDefined();
      });
      
      console.log(`  ✓ Verified ${messages.length} messages have all required fields`);
    })();
    
    // Property 4: Filtering by Status Works
    await test('PRESERVATION: Filtering by status continues to work', async () => {
      // Get all messages first
      const allResponse = await axios.get(
        `${API_BASE_URL}/api/v1/messaging/list?page=1&page_size=100`,
        { headers: { Authorization: `Bearer ${authToken}` } }
      );
      
      const allMessages = allResponse.data.messages || [];
      
      if (allMessages.length === 0) {
        console.log('  ℹ No messages to test filtering');
        return;
      }
      
      // Get unique statuses
      const statuses = [...new Set(allMessages.map(m => m.status))];
      
      if (statuses.length === 0) {
        console.log('  ℹ No status values to filter by');
        return;
      }
      
      // Test filtering by first status
      const testStatus = statuses[0];
      const filteredResponse = await axios.get(
        `${API_BASE_URL}/api/v1/messaging/list?status=${testStatus}&page=1&page_size=100`,
        { headers: { Authorization: `Bearer ${authToken}` } }
      );
      
      const filteredMessages = filteredResponse.data.messages || [];
      
      // Verify all returned messages have the filtered status
      filteredMessages.forEach((msg, index) => {
        if (msg.status !== testStatus) {
          throw new Error(
            `Filtering broken: message ${index} has status "${msg.status}" ` +
            `but filter was "${testStatus}"`
          );
        }
      });
      
      console.log(`  ✓ Filtering by status="${testStatus}" works (${filteredMessages.length} messages)`);
    })();
    
    // Property 5: Initial Load Performance Maintained
    await test('PRESERVATION: Initial load performance is acceptable (<2 seconds)', async () => {
      const startTime = performance.now();
      
      const response = await axios.get(
        `${API_BASE_URL}/api/v1/messaging/list?page=1&page_size=50`,
        { headers: { Authorization: `Bearer ${authToken}` } }
      );
      
      const endTime = performance.now();
      const responseTime = Math.round(endTime - startTime);
      
      console.log(`  Response time: ${responseTime}ms`);
      console.log(`  Messages returned: ${response.data.messages?.length || 0}`);
      
      // Performance should be good (< 2 seconds)
      expect(responseTime, 
        `Initial load should be fast (${responseTime}ms), but took more than 2 seconds`)
        .toBeLessThan(2000);
      
      console.log(`  ✓ Initial load completed in ${responseTime}ms`);
    })();
    
    // Property 6: Pagination Metadata Present
    await test('PRESERVATION: Response includes pagination metadata', async () => {
      const response = await axios.get(
        `${API_BASE_URL}/api/v1/messaging/list?page=1&page_size=25`,
        { headers: { Authorization: `Bearer ${authToken}` } }
      );
      
      const data = response.data;
      
      // Check for pagination metadata
      expect(data.page, 'Response should include page number').toBeDefined();
      expect(data.page_size, 'Response should include page_size').toBeDefined();
      expect(data.total_count, 'Response should include total_count').toBeDefined();
      
      console.log(`  ✓ Pagination metadata present:`);
      console.log(`    - page: ${data.page}`);
      console.log(`    - page_size: ${data.page_size}`);
      console.log(`    - total_count: ${data.total_count}`);
      console.log(`    - total_pages: ${data.total_pages || 'N/A'}`);
    })();
    
    // Property 7: Multiple Pages Work Correctly
    await test('PRESERVATION: Pagination across multiple pages works', async () => {
      // Get first page
      const page1Response = await axios.get(
        `${API_BASE_URL}/api/v1/messaging/list?page=1&page_size=5`,
        { headers: { Authorization: `Bearer ${authToken}` } }
      );
      
      const page1Messages = page1Response.data.messages || [];
      const totalCount = page1Response.data.total_count || 0;
      
      if (totalCount <= 5) {
        console.log(`  ℹ Only ${totalCount} messages, not enough to test multiple pages`);
        return;
      }
      
      // Get second page
      const page2Response = await axios.get(
        `${API_BASE_URL}/api/v1/messaging/list?page=2&page_size=5`,
        { headers: { Authorization: `Bearer ${authToken}` } }
      );
      
      const page2Messages = page2Response.data.messages || [];
      
      // Verify pages have different messages
      if (page1Messages.length > 0 && page2Messages.length > 0) {
        const page1Ids = new Set(page1Messages.map(m => m.id));
        const page2Ids = new Set(page2Messages.map(m => m.id));
        
        // Check for overlap (should be none)
        const overlap = [...page1Ids].filter(id => page2Ids.has(id));
        
        if (overlap.length > 0) {
          throw new Error(
            `Pagination broken: ${overlap.length} messages appear in both page 1 and page 2`
          );
        }
        
        console.log(`  ✓ Page 1 and Page 2 have different messages (no overlap)`);
        console.log(`    - Page 1: ${page1Messages.length} messages`);
        console.log(`    - Page 2: ${page2Messages.length} messages`);
      }
    })();
    
    // Property 8: Empty Results Handled Gracefully
    await test('PRESERVATION: Empty results handled gracefully', async () => {
      // Request a very high page number that should have no results
      const response = await axios.get(
        `${API_BASE_URL}/api/v1/messaging/list?page=99999&page_size=50`,
        { headers: { Authorization: `Bearer ${authToken}` } }
      );
      
      expect(response.status, 'API should return 200 OK even for empty results').toBe(200);
      expect(response.data.messages, 'Response should have messages array').toBeDefined();
      
      const messages = response.data.messages || [];
      expect(messages.length, 'High page number should return empty array').toBe(0);
      
      console.log(`  ✓ Empty results handled gracefully (page 99999 returned 0 messages)`);
    })();
    
    // Property 9: Default Pagination Applied
    await test('PRESERVATION: Default pagination applied when no parameters provided', async () => {
      const response = await axios.get(
        `${API_BASE_URL}/api/v1/messaging/list`,
        { headers: { Authorization: `Bearer ${authToken}` } }
      );
      
      const messages = response.data.messages || [];
      const pageSize = response.data.page_size;
      
      // Should have default page size (typically 50)
      expect(pageSize, 'Should have default page_size').toBeDefined();
      
      // Should not return excessive number of messages
      if (messages.length > 100) {
        throw new Error(
          `Default pagination not applied: returned ${messages.length} messages ` +
          `without explicit page_size parameter`
        );
      }
      
      console.log(`  ✓ Default pagination applied (page_size: ${pageSize}, returned: ${messages.length})`);
    })();
    
    // Property 10: Response Size Reasonable
    await test('PRESERVATION: Response size is reasonable (<5MB)', async () => {
      const response = await axios.get(
        `${API_BASE_URL}/api/v1/messaging/list?page=1&page_size=50`,
        { headers: { Authorization: `Bearer ${authToken}` } }
      );
      
      const dataStr = JSON.stringify(response.data);
      const sizeBytes = new Blob([dataStr]).size;
      const sizeMB = Math.round(sizeBytes / 1024 / 1024 * 100) / 100;
      
      console.log(`  Response size: ${sizeMB}MB`);
      console.log(`  Messages: ${response.data.messages?.length || 0}`);
      
      // Response should be reasonable size
      expect(sizeMB, 
        `Response size (${sizeMB}MB) should be less than 5MB`)
        .toBeLessThan(5);
      
      console.log(`  ✓ Response size is reasonable (${sizeMB}MB)`);
    })();
    
  } catch (error) {
    console.error('\n✗ Test execution failed:', error.message);
    testsFailed++;
  } finally {
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
      
      console.log('\n=== Preservation Test Result ===');
      console.log('✗ PRESERVATION BROKEN: Some functionality is not working correctly');
      console.log('  Message display or pagination has issues that need to be fixed.');
      
      process.exit(1);
    } else {
      console.log('\n=== Preservation Test Result ===');
      console.log('✓ PRESERVATION VERIFIED: All message display functionality works correctly');
      console.log('  - Message listing works');
      console.log('  - Message ordering preserved (newest first)');
      console.log('  - Filtering by status works');
      console.log('  - Pagination metadata present');
      console.log('  - Performance is acceptable');
      console.log('  - Multiple pages work correctly');
      console.log('\nPagination implementation preserves all existing functionality.');
      
      process.exit(0);
    }
  }
})();
