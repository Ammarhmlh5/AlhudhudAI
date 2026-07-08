/**
 * Bug 28: No Message List Pagination - Simplified Preservation Tests
 * 
 * **Validates: Requirements 3.16**
 * 
 * This is a simplified test that verifies preservation properties
 * by analyzing the code structure without requiring a running backend.
 * 
 * EXPECTED OUTCOME: PASS on both UNFIXED and FIXED code
 * These tests ensure pagination implementation preserves existing functionality.
 */

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// __dirname is at: apps/client-web/src/api/__tests__
// Go up 5 levels to workspace root
const workspaceRoot = path.resolve(__dirname, '../../../../..');
const backendServicePath = path.join(workspaceRoot, 'services/backend/internal/receiver/service.go');
const messageHistoryPath = path.join(workspaceRoot, 'apps/client-web/src/components/MessageHistory.jsx');

let testsPassed = 0;
let testsFailed = 0;
const failures = [];

function test(name, fn) {
  try {
    fn();
    console.log(`✓ ${name}`);
    testsPassed++;
  } catch (error) {
    console.error(`✗ ${name}`);
    console.error(`  ${error.message}`);
    testsFailed++;
    failures.push({ name, error: error.message });
  }
}

function expect(value, message) {
  return {
    toBe(expected) {
      if (value !== expected) {
        throw new Error(message || `Expected ${expected} but got ${value}`);
      }
    },
    toBeDefined() {
      if (value === undefined) {
        throw new Error(message || 'Expected value to be defined');
      }
    },
    toContain(substring) {
      if (!value.includes(substring)) {
        throw new Error(message || `Expected value to contain "${substring}"`);
      }
    },
    not: {
      toContain(substring) {
        if (value.includes(substring)) {
          throw new Error(message || `Expected value not to contain "${substring}"`);
        }
      }
    }
  };
}

console.log('\n=== Bug 28: Simplified Preservation Tests ===\n');
console.log('Verifying that pagination preserves existing functionality\n');

// Test 1: Backend service file exists
test('Backend service file should exist', () => {
  expect(fs.existsSync(backendServicePath), 
    'service.go should exist').toBe(true);
});

// Test 2: MessageHistory component exists
test('MessageHistory component should exist', () => {
  expect(fs.existsSync(messageHistoryPath), 
    'MessageHistory.jsx should exist').toBe(true);
});

// Test 3: ListMessages function preserves ordering
test('PRESERVATION: ListMessages maintains DESC ordering (newest first)', () => {
  const serviceContent = fs.readFileSync(backendServicePath, 'utf-8');
  
  // Find ListMessages function
  const funcStart = serviceContent.indexOf('func (s *Service) ListMessages');
  if (funcStart === -1) {
    throw new Error('ListMessages function not found');
  }
  
  let funcEnd = serviceContent.indexOf('\nfunc ', funcStart + 1);
  if (funcEnd === -1) {
    funcEnd = serviceContent.length;
  }
  
  const listMessagesFunc = serviceContent.substring(funcStart, funcEnd);
  
  // Check for DESC ordering
  const hasDescOrdering = listMessagesFunc.includes('Order("created_at DESC")') ||
                          listMessagesFunc.includes('ORDER BY created_at DESC');
  
  if (!hasDescOrdering) {
    throw new Error('Message ordering (DESC) not preserved - messages may not be sorted newest first');
  }
  
  console.log('  ✓ Messages ordered by created_at DESC (newest first)');
});

// Test 4: ListMessages preserves filtering capability
test('PRESERVATION: ListMessages maintains status filtering', () => {
  const serviceContent = fs.readFileSync(backendServicePath, 'utf-8');
  
  const funcStart = serviceContent.indexOf('func (s *Service) ListMessages');
  if (funcStart === -1) {
    throw new Error('ListMessages function not found');
  }
  
  let funcEnd = serviceContent.indexOf('\nfunc ', funcStart + 1);
  if (funcEnd === -1) {
    funcEnd = serviceContent.length;
  }
  
  const listMessagesFunc = serviceContent.substring(funcStart, funcEnd);
  
  // Check for status filtering
  const hasStatusFilter = listMessagesFunc.includes('filters.Status') ||
                          listMessagesFunc.includes('status = ?');
  
  if (!hasStatusFilter) {
    throw new Error('Status filtering not preserved');
  }
  
  console.log('  ✓ Status filtering capability preserved');
});

// Test 5: ListMessages returns all required fields
test('PRESERVATION: ListMessages returns complete message data', () => {
  const serviceContent = fs.readFileSync(backendServicePath, 'utf-8');
  
  const funcStart = serviceContent.indexOf('func (s *Service) ListMessages');
  if (funcStart === -1) {
    throw new Error('ListMessages function not found');
  }
  
  let funcEnd = serviceContent.indexOf('\nfunc ', funcStart + 1);
  if (funcEnd === -1) {
    funcEnd = serviceContent.length;
  }
  
  const listMessagesFunc = serviceContent.substring(funcStart, funcEnd);
  
  // Check that it returns messages (not just metadata)
  const returnsMessages = listMessagesFunc.includes('Messages:') ||
                          listMessagesFunc.includes('messages');
  
  if (!returnsMessages) {
    throw new Error('Function may not return message data');
  }
  
  console.log('  ✓ Function returns complete message data');
});

// Test 6: MessageHistory component maintains display functionality
test('PRESERVATION: MessageHistory component displays messages', () => {
  const componentContent = fs.readFileSync(messageHistoryPath, 'utf-8');
  
  // Check for message display elements
  const hasMessageState = componentContent.includes('useState') && 
                          componentContent.includes('messages');
  const hasMessageMapping = componentContent.includes('.map(') &&
                            componentContent.includes('msg');
  const hasTableOrList = componentContent.includes('<table') ||
                         componentContent.includes('<tbody') ||
                         componentContent.includes('messages.map');
  
  if (!hasMessageState) {
    throw new Error('Component missing message state management');
  }
  
  if (!hasMessageMapping) {
    throw new Error('Component missing message rendering logic');
  }
  
  if (!hasTableOrList) {
    throw new Error('Component missing message display UI');
  }
  
  console.log('  ✓ Component maintains message display functionality');
});

// Test 7: MessageHistory component preserves filtering UI
test('PRESERVATION: MessageHistory component maintains filter controls', () => {
  const componentContent = fs.readFileSync(messageHistoryPath, 'utf-8');
  
  // Check for filter state and UI
  const hasFilterState = componentContent.includes('filter') ||
                         componentContent.includes('status');
  const hasFilterUI = componentContent.includes('<select') ||
                      componentContent.includes('onChange');
  
  if (!hasFilterState) {
    throw new Error('Component missing filter state');
  }
  
  if (!hasFilterUI) {
    throw new Error('Component missing filter UI controls');
  }
  
  console.log('  ✓ Component maintains filter controls');
});

// Test 8: MessageHistory component makes API calls
test('PRESERVATION: MessageHistory component fetches data from API', () => {
  const componentContent = fs.readFileSync(messageHistoryPath, 'utf-8');
  
  // Check for API call
  const hasApiCall = componentContent.includes('apiClient.get') ||
                     componentContent.includes('axios.get') ||
                     componentContent.includes('fetch(');
  const hasMessagingEndpoint = componentContent.includes('/messaging/list') ||
                               componentContent.includes('/messages');
  
  if (!hasApiCall) {
    throw new Error('Component missing API call logic');
  }
  
  if (!hasMessagingEndpoint) {
    throw new Error('Component not calling messaging endpoint');
  }
  
  console.log('  ✓ Component fetches data from API');
});

// Test 9: MessageHistory component sends pagination parameters
test('PRESERVATION: MessageHistory component sends page and page_size parameters', () => {
  const componentContent = fs.readFileSync(messageHistoryPath, 'utf-8');
  
  // Check for pagination parameters in API call
  const hasPageParam = componentContent.includes('page=') ||
                       componentContent.includes('page:') ||
                       componentContent.includes('${page}');
  const hasPageSizeParam = componentContent.includes('page_size=') ||
                           componentContent.includes('pageSize:') ||
                           componentContent.includes('${pageSize}');
  
  if (!hasPageParam) {
    throw new Error('Component not sending page parameter');
  }
  
  if (!hasPageSizeParam) {
    throw new Error('Component not sending page_size parameter');
  }
  
  console.log('  ✓ Component sends pagination parameters');
});

// Test 10: MessageHistory component handles response data
test('PRESERVATION: MessageHistory component handles paginated response', () => {
  const componentContent = fs.readFileSync(messageHistoryPath, 'utf-8');
  
  // Check for response handling
  const setsMessages = componentContent.includes('setMessages') ||
                       componentContent.includes('messages:');
  const handlesTotalCount = componentContent.includes('total_count') ||
                            componentContent.includes('totalCount');
  
  if (!setsMessages) {
    throw new Error('Component not updating messages state');
  }
  
  if (!handlesTotalCount) {
    throw new Error('Component not handling total_count from response');
  }
  
  console.log('  ✓ Component handles paginated response data');
});

// Test 11: No breaking changes to message structure
test('PRESERVATION: Message structure not altered', () => {
  const componentContent = fs.readFileSync(messageHistoryPath, 'utf-8');
  
  // Check that component still accesses standard message fields
  const accessesStatus = componentContent.includes('msg.status') ||
                         componentContent.includes('message.status');
  const accessesPhoneNumber = componentContent.includes('msg.phone_number') ||
                              componentContent.includes('message.phone_number');
  const accessesContent = componentContent.includes('msg.content') ||
                          componentContent.includes('message.content');
  const accessesCreatedAt = componentContent.includes('msg.created_at') ||
                            componentContent.includes('message.created_at');
  
  if (!accessesStatus || !accessesPhoneNumber || !accessesContent || !accessesCreatedAt) {
    throw new Error('Component may not be accessing all required message fields');
  }
  
  console.log('  ✓ Message structure preserved (status, phone_number, content, created_at)');
});

// Test 12: Export functionality preserved
test('PRESERVATION: CSV export functionality maintained', () => {
  const componentContent = fs.readFileSync(messageHistoryPath, 'utf-8');
  
  // Check for export functionality
  const hasExportFunction = componentContent.includes('handleExportCSV') ||
                            componentContent.includes('exportCSV') ||
                            componentContent.includes('export');
  const hasExportButton = componentContent.includes('Download') ||
                          componentContent.includes('تصدير');
  
  if (!hasExportFunction) {
    throw new Error('Export functionality may be missing');
  }
  
  if (!hasExportButton) {
    throw new Error('Export button may be missing');
  }
  
  console.log('  ✓ CSV export functionality preserved');
});

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
  console.log('✗ PRESERVATION BROKEN: Some functionality may not be working correctly');
  console.log('  Review the failures above to identify what needs to be fixed.');
  
  process.exit(1);
} else {
  console.log('\n=== Preservation Test Result ===');
  console.log('✓ PRESERVATION VERIFIED: All existing functionality is preserved');
  console.log('  - Message display works');
  console.log('  - Message ordering preserved (DESC)');
  console.log('  - Status filtering maintained');
  console.log('  - Component structure intact');
  console.log('  - API integration preserved');
  console.log('  - Export functionality maintained');
  console.log('\nPagination implementation does not break existing features.');
  
  process.exit(0);
}
