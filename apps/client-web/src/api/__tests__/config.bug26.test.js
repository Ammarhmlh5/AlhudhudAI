/**
 * Bug 26: Hardcoded API URLs - Bug Condition Exploration Test
 * 
 * **Validates: Requirements 1.26, 2.26**
 * 
 * This test verifies the BUG CONDITION:
 * - API URLs should be configurable via environment variables
 * - Environment-specific configuration files should exist
 * - Build scripts should support different environments
 * 
 * EXPECTED OUTCOME ON UNFIXED CODE: FAIL
 * - Missing .env.production and .env.staging files
 * - Missing build:staging script
 * - Potentially hardcoded values
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

console.log('\n=== Bug 26: Hardcoded API URLs - Bug Condition Exploration ===\n');

// Test 1: Environment-specific config files for client-web
test('should have environment-specific config files for client-web', () => {
  const devEnvPath = path.join(clientWebRoot, '.env.development');
  expect(fs.existsSync(devEnvPath), 
    '.env.development should exist for development environment').toBe(true);

  const prodEnvPath = path.join(clientWebRoot, '.env.production');
  expect(fs.existsSync(prodEnvPath), 
    '.env.production should exist for production environment').toBe(true);

  const stagingEnvPath = path.join(clientWebRoot, '.env.staging');
  expect(fs.existsSync(stagingEnvPath), 
    '.env.staging should exist for staging environment').toBe(true);
});

// Test 2: Environment-specific config files for admin-web
test('should have environment-specific config files for admin-web', () => {
  const devEnvPath = path.join(adminWebRoot, '.env.development');
  expect(fs.existsSync(devEnvPath), 
    '.env.development should exist for development environment').toBe(true);

  const prodEnvPath = path.join(adminWebRoot, '.env.production');
  expect(fs.existsSync(prodEnvPath), 
    '.env.production should exist for production environment').toBe(true);

  const stagingEnvPath = path.join(adminWebRoot, '.env.staging');
  expect(fs.existsSync(stagingEnvPath), 
    '.env.staging should exist for staging environment').toBe(true);
});

// Test 3: build:staging script in client-web
test('should have build:staging script in client-web package.json', () => {
  const packageJsonPath = path.join(clientWebRoot, 'package.json');
  const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, 'utf-8'));
  
  expect(packageJson.scripts, 'scripts should be defined').toBeDefined();
  expect(packageJson.scripts['build:staging'], 'build:staging script should exist').toBeDefined();
  expect(packageJson.scripts['build:staging'], 'build:staging should use --mode staging').toContain('--mode staging');
});

// Test 4: build:staging script in admin-web
test('should have build:staging script in admin-web package.json', () => {
  const packageJsonPath = path.join(adminWebRoot, 'package.json');
  const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, 'utf-8'));
  
  expect(packageJson.scripts, 'scripts should be defined').toBeDefined();
  expect(packageJson.scripts['build:staging'], 'build:staging script should exist').toBeDefined();
  expect(packageJson.scripts['build:staging'], 'build:staging should use --mode staging').toContain('--mode staging');
});

// Test 5: Environment variables in client-web config
test('should use environment variables in client-web config.js', () => {
  const configPath = path.join(clientWebRoot, 'src/config.js');
  const configContent = fs.readFileSync(configPath, 'utf-8');
  
  expect(configContent, 'Should use import.meta.env').toMatch(/import\.meta\.env\.VITE_\w+/);
  expect(configContent, 'Should not have hardcoded production URLs').not.toMatch(/https:\/\/api\.alhudhud\.com/);
  expect(configContent, 'Should not have hardcoded staging URLs').not.toMatch(/https:\/\/staging-api\.alhudhud\.com/);
});

// Test 6: Environment variables in admin-web config
test('should use environment variables in admin-web config.js', () => {
  const configPath = path.join(adminWebRoot, 'src/config.js');
  const configContent = fs.readFileSync(configPath, 'utf-8');
  
  expect(configContent, 'Should use import.meta.env').toMatch(/import\.meta\.env\.VITE_\w+/);
  expect(configContent, 'Should not have hardcoded production URLs').not.toMatch(/https:\/\/api\.alhudhud\.com/);
  expect(configContent, 'Should not have hardcoded staging URLs').not.toMatch(/https:\/\/staging-api\.alhudhud\.com/);
});

// Test 7: Proper fallback values
test('should have proper fallback values for development', () => {
  const clientConfigPath = path.join(clientWebRoot, 'src/config.js');
  const clientConfig = fs.readFileSync(clientConfigPath, 'utf-8');
  expect(clientConfig, 'Client config should have localhost fallback').toMatch(/localhost:8081/);
  
  const adminConfigPath = path.join(adminWebRoot, 'src/config.js');
  const adminConfig = fs.readFileSync(adminConfigPath, 'utf-8');
  expect(adminConfig, 'Admin config should have localhost fallback').toMatch(/localhost:8081/);
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
  console.log('\n✓ All tests passed!');
  process.exit(0);
}
