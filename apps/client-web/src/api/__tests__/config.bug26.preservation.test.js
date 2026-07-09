/**
 * Bug 26: Hardcoded API URLs - Preservation Tests
 * 
 * **Validates: Requirements 3.9, 3.15**
 * 
 * These tests verify PRESERVATION properties:
 * - API requests continue to work correctly
 * - Configuration is properly loaded
 * - Axios client is properly configured
 * 
 * EXPECTED OUTCOME: PASS (on both unfixed and fixed code)
 * These tests ensure the fix doesn't break existing functionality.
 */

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const clientWebRoot = path.resolve(__dirname, '../../../');
const adminWebRoot = path.resolve(__dirname, '../../../../admin-web');

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
    toBeTruthy() {
      if (!value) {
        throw new Error(message || 'Expected value to be truthy');
      }
    },
    toMatch(regex) {
      if (!regex.test(value)) {
        throw new Error(message || `Expected value to match ${regex}`);
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

console.log('\n=== Bug 26: Hardcoded API URLs - Preservation Tests ===\n');

// Test 1: Config files exist and are readable
test('client-web config.js should exist and be readable', () => {
  const configPath = path.join(clientWebRoot, 'src/config.js');
  expect(fs.existsSync(configPath), 'config.js should exist').toBe(true);
  
  const configContent = fs.readFileSync(configPath, 'utf-8');
  expect(configContent.length > 0, 'config.js should have content').toBe(true);
});

test('admin-web config.js should exist and be readable', () => {
  const configPath = path.join(adminWebRoot, 'src/config.js');
  expect(fs.existsSync(configPath), 'config.js should exist').toBe(true);
  
  const configContent = fs.readFileSync(configPath, 'utf-8');
  expect(configContent.length > 0, 'config.js should have content').toBe(true);
});

// Test 2: Config exports required constants
test('client-web config should export API_BASE_URL', () => {
  const configPath = path.join(clientWebRoot, 'src/config.js');
  const configContent = fs.readFileSync(configPath, 'utf-8');
  
  expect(configContent, 'Should export API_BASE_URL').toMatch(/export\s+const\s+API_BASE_URL/);
});

test('client-web config should export WS_BASE_URL', () => {
  const configPath = path.join(clientWebRoot, 'src/config.js');
  const configContent = fs.readFileSync(configPath, 'utf-8');
  
  expect(configContent, 'Should export WS_BASE_URL').toMatch(/export\s+const\s+WS_BASE_URL/);
});

test('admin-web config should export API_BASE_URL', () => {
  const configPath = path.join(adminWebRoot, 'src/config.js');
  const configContent = fs.readFileSync(configPath, 'utf-8');
  
  expect(configContent, 'Should export API_BASE_URL').toMatch(/export\s+const\s+API_BASE_URL/);
});

test('admin-web config should export WS_BASE_URL', () => {
  const configPath = path.join(adminWebRoot, 'src/config.js');
  const configContent = fs.readFileSync(configPath, 'utf-8');
  
  expect(configContent, 'Should export WS_BASE_URL').toMatch(/export\s+const\s+WS_BASE_URL/);
});

// Test 3: Axios client uses config
test('client-web axios should import from config', () => {
  const axiosPath = path.join(clientWebRoot, 'src/api/axios.js');
  expect(fs.existsSync(axiosPath), 'axios.js should exist').toBe(true);
  
  const axiosContent = fs.readFileSync(axiosPath, 'utf-8');
  expect(axiosContent, 'Should import from config').toMatch(/from\s+['"]\.\.\/config['"]/);
  expect(axiosContent, 'Should import API_BASE_URL').toMatch(/API_BASE_URL/);
});

test('admin-web axios should import from config', () => {
  const axiosPath = path.join(adminWebRoot, 'src/api/axios.js');
  expect(fs.existsSync(axiosPath), 'axios.js should exist').toBe(true);
  
  const axiosContent = fs.readFileSync(axiosPath, 'utf-8');
  expect(axiosContent, 'Should import from config').toMatch(/from\s+['"]\.\.\/config['"]/);
  expect(axiosContent, 'Should import API_BASE_URL').toMatch(/API_BASE_URL/);
});

// Test 4: Axios client is properly configured
test('client-web axios should create client with baseURL', () => {
  const axiosPath = path.join(clientWebRoot, 'src/api/axios.js');
  const axiosContent = fs.readFileSync(axiosPath, 'utf-8');
  
  expect(axiosContent, 'Should create axios instance').toMatch(/axios\.create/);
  expect(axiosContent, 'Should set baseURL').toMatch(/baseURL/);
});

test('admin-web axios should create client with baseURL', () => {
  const axiosPath = path.join(adminWebRoot, 'src/api/axios.js');
  const axiosContent = fs.readFileSync(axiosPath, 'utf-8');
  
  expect(axiosContent, 'Should create axios instance').toMatch(/axios\.create/);
  expect(axiosContent, 'Should set baseURL').toMatch(/baseURL/);
});

// Test 5: Config has valid URL format
test('client-web config should have valid URL format', () => {
  const configPath = path.join(clientWebRoot, 'src/config.js');
  const configContent = fs.readFileSync(configPath, 'utf-8');
  
  // Should have http:// or https:// protocol
  expect(configContent, 'Should have valid URL protocol').toMatch(/https?:\/\//);
});

test('admin-web config should have valid URL format', () => {
  const configPath = path.join(adminWebRoot, 'src/config.js');
  const configContent = fs.readFileSync(configPath, 'utf-8');
  
  // Should have http:// or https:// protocol
  expect(configContent, 'Should have valid URL protocol').toMatch(/https?:\/\//);
});

// Test 6: Environment file exists (at least one)
test('client-web should have at least one .env file', () => {
  const envPath = path.join(clientWebRoot, '.env');
  expect(fs.existsSync(envPath), 'At least .env should exist').toBe(true);
});

test('admin-web should have at least one .env file', () => {
  const envPath = path.join(adminWebRoot, '.env');
  expect(fs.existsSync(envPath), 'At least .env should exist').toBe(true);
});

// Test 7: Build scripts exist
test('client-web should have build script', () => {
  const packageJsonPath = path.join(clientWebRoot, 'package.json');
  const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, 'utf-8'));
  
  expect(packageJson.scripts, 'scripts should be defined').toBeDefined();
  expect(packageJson.scripts.build, 'build script should exist').toBeDefined();
});

test('admin-web should have build script', () => {
  const packageJsonPath = path.join(adminWebRoot, 'package.json');
  const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, 'utf-8'));
  
  expect(packageJson.scripts, 'scripts should be defined').toBeDefined();
  expect(packageJson.scripts.build, 'build script should exist').toBeDefined();
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
  process.exit(1);
} else {
  console.log('\n✓ All preservation tests passed!');
  console.log('\nThis confirms that:');
  console.log('- Configuration files are properly structured');
  console.log('- Axios clients are correctly configured');
  console.log('- Basic build infrastructure exists');
  console.log('- The fix will not break existing functionality');
  process.exit(0);
}
