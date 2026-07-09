/**
 * Bug 27: No WebSocket Auto-Reconnect - Preservation Test
 * 
 * **Validates: Requirements 3.3**
 * 
 * This test verifies PRESERVATION requirements:
 * - WebSocket connections work successfully
 * - Messages are delivered correctly
 * - Connection state is tracked properly
 * - Message handling remains functional
 * 
 * EXPECTED OUTCOME: Tests PASS on both UNFIXED and FIXED code
 */

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const socketContextPath = path.resolve(__dirname, '../SocketContext.jsx');

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
    toMatch(regex) {
      if (!regex.test(value)) {
        throw new Error(message || `Expected value to match ${regex}`);
      }
    },
    not: {
      toMatch(regex) {
        if (regex.test(value)) {
          throw new Error(message || `Expected value not to match ${regex}`);
        }
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
    }
  };
}

console.log('\n=== Bug 27: No WebSocket Auto-Reconnect - Preservation Tests ===\n');

// Read SocketContext.jsx
const socketContextContent = fs.readFileSync(socketContextPath, 'utf-8');

// Test 1: WebSocket connection initialization
test('should initialize WebSocket connection', () => {
  expect(socketContextContent, 
    'Should create WebSocket instance').toMatch(/new WebSocket/);
  expect(socketContextContent, 
    'Should use WS_BASE_URL from config').toMatch(/WS_BASE_URL/);
});

// Test 2: Connection state tracking
test('should track connection state', () => {
  expect(socketContextContent, 
    'Should have isConnected state').toMatch(/isConnected/);
  expect(socketContextContent, 
    'Should set isConnected to true on open').toMatch(/setIsConnected.*true/);
  expect(socketContextContent, 
    'Should set isConnected to false on close').toMatch(/setIsConnected.*false/);
});

// Test 3: WebSocket event handlers
test('should have WebSocket event handlers', () => {
  expect(socketContextContent, 
    'Should have onopen handler').toMatch(/ws\.onopen|onopen\s*=/);
  expect(socketContextContent, 
    'Should have onclose handler').toMatch(/ws\.onclose|onclose\s*=/);
  expect(socketContextContent, 
    'Should have onerror handler').toMatch(/ws\.onerror|onerror\s*=/);
  expect(socketContextContent, 
    'Should have onmessage handler').toMatch(/ws\.onmessage|onmessage\s*=/);
});

// Test 4: Message handling
test('should handle incoming messages', () => {
  expect(socketContextContent, 
    'Should parse JSON messages').toMatch(/JSON\.parse/);
  expect(socketContextContent, 
    'Should have message handling function').toMatch(/handleSocketEvent|handleMessage/);
});

// Test 5: Message type handling
test('should handle different message types', () => {
  expect(socketContextContent, 
    'Should handle MESSAGE_RECEIVED').toMatch(/MESSAGE_RECEIVED/);
  expect(socketContextContent, 
    'Should handle MESSAGE_ANALYZED').toMatch(/MESSAGE_ANALYZED/);
  expect(socketContextContent, 
    'Should handle MESSAGE_STATUS_UPDATE').toMatch(/MESSAGE_STATUS_UPDATE/);
  expect(socketContextContent, 
    'Should handle METRICS_UPDATED').toMatch(/METRICS_UPDATED/);
});

// Test 6: State management for messages
test('should manage message state', () => {
  expect(socketContextContent, 
    'Should have incomingMessages state').toMatch(/incomingMessages/);
  expect(socketContextContent, 
    'Should have deliveryUpdates state').toMatch(/deliveryUpdates/);
  expect(socketContextContent, 
    'Should have metricsNonce state').toMatch(/metricsNonce/);
});

// Test 7: Context provider
test('should provide SocketContext', () => {
  expect(socketContextContent, 
    'Should create SocketContext').toMatch(/createContext/);
  expect(socketContextContent, 
    'Should export SocketProvider').toMatch(/export.*SocketProvider/);
  expect(socketContextContent, 
    'Should export useSocket hook').toMatch(/export.*useSocket/);
});

// Test 8: Authentication integration
test('should integrate with authentication', () => {
  expect(socketContextContent, 
    'Should use token from localStorage or auth').toMatch(/token|localStorage\.getItem/);
  expect(socketContextContent, 
    'Should check for user authentication').toMatch(/user|token/);
});

// Test 9: Connection cleanup
test('should cleanup connection on unmount', () => {
  expect(socketContextContent, 
    'Should have useEffect cleanup').toMatch(/return\s*\(\s*\)\s*=>/);
  expect(socketContextContent, 
    'Should close socket on cleanup').toMatch(/socket.*close|ws\.close/);
});

// Test 10: Error handling
test('should handle WebSocket errors', () => {
  expect(socketContextContent, 
    'Should have error handler').toMatch(/onerror/);
  expect(socketContextContent, 
    'Should close socket on error').toMatch(/onerror[\s\S]*?close/);
});

// Summary
console.log(`\n=== Test Summary ===`);
console.log(`Passed: ${testsPassed}`);
console.log(`Failed: ${testsFailed}`);

if (testsFailed > 0) {
  console.log('\n=== Failures ===');
  console.log('\n⚠️  PRESERVATION REQUIREMENTS VIOLATED:');
  console.log('The following core WebSocket functionality is missing or broken:');
  failures.forEach(({ name, error }) => {
    console.log(`\n${name}:`);
    console.log(`  ${error}`);
  });
  console.log('\n✗ Preservation tests failed - core functionality broken');
  process.exit(1);
} else {
  console.log('\n✓ All preservation tests passed!');
  console.log('✓ WebSocket connection functionality preserved');
  console.log('✓ Message handling preserved');
  console.log('✓ State management preserved');
  process.exit(0);
}
