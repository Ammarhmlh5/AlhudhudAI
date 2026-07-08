/**
 * Bug 27: No WebSocket Auto-Reconnect - Exploration Test
 * 
 * **Validates: Requirements 1.27, 2.27**
 * 
 * This test verifies the BUG CONDITION:
 * - WebSocket connection drops
 * - No automatic reconnection is attempted
 * - User must manually refresh page
 * 
 * EXPECTED OUTCOME ON UNFIXED CODE: Test FAILS (no auto-reconnect)
 * EXPECTED OUTCOME ON FIXED CODE: Test PASSES (auto-reconnect works)
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

console.log('\n=== Bug 27: No WebSocket Auto-Reconnect - Bug Condition Exploration ===\n');

// Read SocketContext.jsx
const socketContextContent = fs.readFileSync(socketContextPath, 'utf-8');

// Test 1: Check for reconnection state management
test('should have reconnection attempt tracking state', () => {
  expect(socketContextContent, 
    'Should track reconnection attempts with useState').toMatch(/reconnectAttempt|reconnect.*Attempt/);
});

// Test 2: Check for reconnection timeout reference
test('should have reconnection timeout reference', () => {
  expect(socketContextContent, 
    'Should have useRef for reconnection timeout').toMatch(/useRef.*reconnect|reconnect.*useRef/i);
});

// Test 3: Check for exponential backoff logic
test('should implement exponential backoff calculation', () => {
  expect(socketContextContent, 
    'Should calculate exponential backoff delay').toMatch(/Math\.pow.*2|2\s*\*\*|baseDelay.*\*/);
});

// Test 4: Check for max delay cap
test('should cap maximum reconnection delay', () => {
  expect(socketContextContent, 
    'Should have maximum delay cap (30000ms or 30s)').toMatch(/Math\.min.*30000|maxDelay.*30000|30\s*\*\s*1000/);
});

// Test 5: Check for reconnection scheduling in onclose
test('should schedule reconnection on WebSocket close', () => {
  expect(socketContextContent, 
    'Should call reconnection function in onclose handler').toMatch(/onclose.*scheduleReconnect|scheduleReconnect.*onclose/s);
});

// Test 6: Check for reconnection attempt reset on successful connection
test('should reset reconnection attempts on successful connection', () => {
  expect(socketContextContent, 
    'Should reset reconnect attempts in onopen handler').toMatch(/onopen[\s\S]*?setReconnectAttempt.*0|onopen[\s\S]*?reconnectAttempt.*=.*0/);
});

// Test 7: Check for timeout cleanup
test('should clear reconnection timeout on disconnect', () => {
  expect(socketContextContent, 
    'Should clear timeout when disconnecting').toMatch(/clearTimeout.*reconnect/i);
});

// Test 8: Check for connection state tracking
test('should track connection state', () => {
  expect(socketContextContent, 
    'Should have isConnected state').toMatch(/isConnected/);
  expect(socketContextContent, 
    'Should update isConnected on open').toMatch(/onopen[\s\S]*?setIsConnected.*true/);
  expect(socketContextContent, 
    'Should update isConnected on close').toMatch(/onclose[\s\S]*?setIsConnected.*false/);
});

// Summary
console.log(`\n=== Test Summary ===`);
console.log(`Passed: ${testsPassed}`);
console.log(`Failed: ${testsFailed}`);

if (testsFailed > 0) {
  console.log('\n=== Failures ===');
  console.log('\nThese failures confirm the BUG CONDITION:');
  console.log('- WebSocket does NOT automatically reconnect on disconnect');
  console.log('- No exponential backoff logic implemented');
  console.log('- Users must manually refresh to restore connection');
  failures.forEach(({ name, error }) => {
    console.log(`\n${name}:`);
    console.log(`  ${error}`);
  });
  console.log('\n✗ Bug condition confirmed: No auto-reconnect implemented');
  process.exit(1);
} else {
  console.log('\n✓ All reconnection features detected - bug appears to be fixed!');
  process.exit(0);
}
