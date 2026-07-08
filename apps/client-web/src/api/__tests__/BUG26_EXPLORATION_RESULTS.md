# Bug 26: Hardcoded API URLs - Exploration Test Results

## Test Execution Date
2024-01-XX

## Test File
`apps/client-web/src/api/__tests__/config.bug26.test.js`

## Expected Outcome
**FAIL** - The test should fail on unfixed code, confirming the bug condition exists.

## Actual Outcome
✅ **FAILED AS EXPECTED** - Test correctly identified the bug condition.

## Test Results

### Passed Tests (3/7)
✓ should use environment variables in client-web config.js
✓ should use environment variables in admin-web config.js
✓ should have proper fallback values for development

### Failed Tests (4/7)
✗ should have environment-specific config files for client-web
  - Missing: .env.development
  - Missing: .env.production
  - Missing: .env.staging

✗ should have environment-specific config files for admin-web
  - Missing: .env.development
  - Missing: .env.production
  - Missing: .env.staging

✗ should have build:staging script in client-web package.json
  - Missing: build:staging script

✗ should have build:staging script in admin-web package.json
  - Missing: build:staging script

## Bug Condition Confirmed

The bug exists in the following areas:

1. **Missing Environment Files**: No environment-specific configuration files (.env.development, .env.production, .env.staging) exist for either client-web or admin-web applications.

2. **Missing Build Scripts**: No `build:staging` script exists in package.json files to support staging environment builds.

3. **Partial Fix Already Applied**: The config.js files already use environment variables (import.meta.env.VITE_API_URL), which is good, but the environment-specific files and build scripts are missing.

## Impact

Without environment-specific files and build scripts:
- Developers must manually change .env file for different environments
- Risk of deploying with wrong configuration
- No clear separation between development, staging, and production settings
- Build process doesn't support multiple environments

## Next Steps

1. Create environment-specific files (.env.development, .env.production, .env.staging)
2. Add build:staging scripts to package.json files
3. Verify the fix by re-running this test (should pass after fix)
