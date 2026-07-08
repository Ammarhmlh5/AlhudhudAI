# Implementation Plan

## المرحلة 1: إصلاح المشاكل الأمنية الحرجة (Critical Security Issues)

### 1. اختبار استكشاف Bug Condition - JWT_SECRET غير آمن

- [x] 1.1 Write bug condition exploration test for JWT_SECRET
  - **Property 1: Bug Condition** - Insecure JWT_SECRET Detection
  - **CRITICAL**: Write this property-based test BEFORE implementing the fix
  - **GOAL**: Surface counterexamples that demonstrate the security vulnerability exists
  - **Scoped PBT Approach**: Test that system accepts weak JWT_SECRET values (default placeholder, short length < 32 chars, non-random patterns)
  - Test implementation details from Bug Condition in design (isBugCondition where JWT_SECRET == "change-me-in-production-to-a-secure-random-string")
  - The test assertions should match the Expected Behavior Properties from design (Property 1)
  - Run test on UNFIXED code
  - **EXPECTED OUTCOME**: Test FAILS (this is correct - it proves the bug exists)
  - Document counterexamples found to understand root cause
  - Mark task complete when test is written, run, and failure is documented
  - _Requirements: 2.1_

- [x] 1.2 Write bug condition exploration test for REDIS_PASSWORD
  - **Property 1: Bug Condition** - Empty REDIS_PASSWORD Detection
  - **CRITICAL**: Write this property-based test BEFORE implementing the fix
  - **GOAL**: Surface counterexamples that demonstrate Redis is unprotected
  - **Scoped PBT Approach**: Test that system accepts empty REDIS_PASSWORD and connects to Redis without authentication
  - Test implementation details from Bug Condition in design (isBugCondition where REDIS_PASSWORD == "")
  - The test assertions should match the Expected Behavior Properties from design (Property 1)
  - Run test on UNFIXED code
  - **EXPECTED OUTCOME**: Test FAILS (this is correct - it proves the bug exists)
  - Document counterexamples found to understand root cause
  - Mark task complete when test is written, run, and failure is documented
  - _Requirements: 2.2_

### 2. اختبار Preservation - الحفاظ على أداء المصادقة

- [x] 2.1 Write preservation property tests for authentication performance (BEFORE implementing fix)
  - **Property 2: Preservation** - JWT Authentication Performance
  - **IMPORTANT**: Follow observation-first methodology
  - Observe: JWT token generation and validation times on unfixed code
  - Observe: Redis connection and cache access times on unfixed code
  - Write property-based test: for all valid authentication requests, response time should remain < 100ms (from Preservation Requirements in design)
  - Verify test passes on UNFIXED code
  - _Requirements: 3.1_

### 3. تطبيق إصلاح المشاكل الأمنية

- [x] 3. Fix security configuration issues

  - [x] 3.1 Implement JWT_SECRET validation and generation
    - Add validation function in `services/backend/internal/config/config.go` to check JWT_SECRET strength
    - Reject startup if JWT_SECRET is default value or < 48 chars base64
    - Provide clear error message with command: `openssl rand -base64 48`
    - Update `.env` file with secure generated JWT_SECRET
    - Update `.env.example` with instructions
    - _Bug_Condition: isBugCondition(input) where input.JWT_SECRET == "change-me-in-production-to-a-secure-random-string"_
    - _Expected_Behavior: System SHALL reject startup with insecure JWT_SECRET and provide guidance (Property 1)_
    - _Preservation: JWT authentication SHALL maintain same performance < 100ms (Requirement 3.1)_
    - _Requirements: 2.1, 3.1_

  - [x] 3.2 Implement REDIS_PASSWORD validation and configuration
    - Add validation function in `services/backend/internal/config/config.go` to check REDIS_PASSWORD is not empty
    - Reject startup if REDIS_PASSWORD is empty in production environment
    - Provide clear error message with command: `openssl rand -base64 32`
    - Update `.env` file with secure generated REDIS_PASSWORD
    - Update `redis.conf` to match the password
    - Update `.env.example` with instructions
    - _Bug_Condition: isBugCondition(input) where input.REDIS_PASSWORD == ""_
    - _Expected_Behavior: System SHALL reject startup with empty REDIS_PASSWORD and provide guidance (Property 1)_
    - _Preservation: Redis caching SHALL maintain same performance (Requirement 3.1)_
    - _Requirements: 2.2, 3.1_

  - [x] 3.3 Verify bug condition exploration tests now pass
    - **Property 1: Expected Behavior** - Security Configuration Validation
    - **IMPORTANT**: Re-run the SAME tests from tasks 1.1 and 1.2 - do NOT write new tests
    - The tests from tasks 1.1 and 1.2 encode the expected behavior
    - When these tests pass, it confirms the expected behavior is satisfied
    - Run bug condition exploration tests from steps 1.1 and 1.2
    - **EXPECTED OUTCOME**: Tests PASS (confirms bugs are fixed)
    - _Requirements: 2.1, 2.2_

  - [x] 3.4 Verify preservation tests still pass
    - **Property 2: Preservation** - Authentication Performance Maintained
    - **IMPORTANT**: Re-run the SAME tests from task 2.1 - do NOT write new tests
    - Run preservation property tests from step 2.1
    - **EXPECTED OUTCOME**: Tests PASS (confirms no regressions)
    - Confirm all tests still pass after fix (no performance degradation)
    - _Requirements: 3.1_

- [x] 4. Checkpoint - Security fixes complete
  - Ensure all security tests pass
  - Verify system starts successfully with secure configuration
  - Ask user if questions arise

## المرحلة 2: تفعيل الوظائف المعطلة (Enable Disabled Features)

### 5. اختبار استكشاف Bug Condition - نظام الإحالة معطّل

- [x] 5.1 Write bug condition exploration test for disabled referral system
  - **Property 1: Bug Condition** - Referral System Disabled
  - **CRITICAL**: Write this property-based test BEFORE implementing the fix
  - **GOAL**: Surface counterexamples that demonstrate referral system is not accessible
  - **Scoped PBT Approach**: Test that API endpoints for referral system return 404 or "not found" errors
  - Test implementation details from Bug Condition in design (isBugCondition where ReferralSystemEnabled == false)
  - The test assertions should match the Expected Behavior Properties from design (Property 2)
  - Run test on UNFIXED code
  - **EXPECTED OUTCOME**: Test FAILS (this is correct - it proves the bug exists)
  - Document counterexamples found (missing endpoints, disabled manager)
  - Mark task complete when test is written, run, and failure is documented
  - _Requirements: 2.3_

### 6. اختبار استكشاف Bug Condition - تصدير التقارير غير مطبق

- [x] 6.1 Write bug condition exploration test for unimplemented export functions
  - **Property 1: Bug Condition** - Export Functions Not Implemented
  - **CRITICAL**: Write this property-based test BEFORE implementing the fix
  - **GOAL**: Surface counterexamples that demonstrate export functions return "not implemented" errors
  - **Scoped PBT Approach**: Test that exportToJSON, exportToCSV, exportToPDF all return "not implemented" error
  - Test implementation details from Bug Condition in design (isBugCondition where ExportFunctionReturns == "not implemented")
  - The test assertions should match the Expected Behavior Properties from design (Property 3)
  - Run test on UNFIXED code
  - **EXPECTED OUTCOME**: Test FAILS (this is correct - it proves the bug exists)
  - Document counterexamples found for each export format
  - Mark task complete when test is written, run, and failure is documented
  - _Requirements: 2.4_

### 7. اختبار استكشاف Bug Condition - Rate Limiting معطّل عند فشل Redis

- [x] 7.1 Write bug condition exploration test for disabled rate limiting
  - **Property 1: Bug Condition** - Rate Limiting Disabled When Redis Fails
  - **CRITICAL**: Write this property-based test BEFORE implementing the fix
  - **GOAL**: Surface counterexamples that demonstrate rate limiting is bypassed when Redis is unavailable
  - **Scoped PBT Approach**: Simulate Redis failure and test that unlimited requests are accepted (> 100 requests/minute)
  - Test implementation details from Bug Condition in design (isBugCondition where RedisAvailable == false AND RateLimiterActive == false)
  - The test assertions should match the Expected Behavior Properties from design (Property 4)
  - Run test on UNFIXED code
  - **EXPECTED OUTCOME**: Test FAILS (this is correct - it proves the bug exists)
  - Document counterexamples found (no-op middleware allows unlimited requests)
  - Mark task complete when test is written, run, and failure is documented
  - _Requirements: 2.5_

### 8. اختبار Preservation - الحفاظ على أداء الوظائف الموجودة

- [x] 8.1 Write preservation property tests for existing functionality (BEFORE implementing fix)
  - **Property 2: Preservation** - Existing Features Performance
  - **IMPORTANT**: Follow observation-first methodology
  - Observe: Message sending performance on unfixed code
  - Observe: User management operations performance on unfixed code
  - Observe: Existing reports generation performance on unfixed code
  - Write property-based test: for all existing operations, response times should remain unchanged (< 5ms variance)
  - Verify test passes on UNFIXED code
  - _Requirements: 3.2, 3.3_

### 9. تطبيق تفعيل نظام الإحالة

- [x] 9. Activate referral system

  - [x] 9.1 Add referral configuration to config.go
    - Add `ReferralConfig` struct in `services/backend/internal/config/config.go`
    - Include fields: Enabled, CodeLength, ExpiryDays, MaxUsesPerCode, CacheTTL
    - Load configuration from environment variables with sensible defaults
    - _Bug_Condition: isBugCondition(input) where input.ReferralSystemEnabled == false_
    - _Expected_Behavior: System SHALL provide fully functional referral API endpoints (Property 2)_
    - _Preservation: Existing features SHALL maintain same performance (Requirement 3.2)_
    - _Requirements: 2.3, 3.2_

  - [x] 9.2 Initialize ReferralManager in main.go
    - Import referral package in `services/backend/main.go`
    - Initialize ReferralManager with config, database, Redis, and logger
    - Add error handling for initialization failures
    - _Requirements: 2.3_

  - [x] 9.3 Add referral API endpoints in main.go
    - Add POST `/api/v1/referrals/generate` - generate referral code
    - Add POST `/api/v1/referrals/validate` - validate referral code
    - Add POST `/api/v1/referrals/process` - process referral
    - Add GET `/api/v1/referrals/stats/:tenantID` - get referral statistics
    - Add GET `/api/v1/referrals/report/:tenantID` - get referral report
    - Add authentication middleware to all endpoints
    - Add rate limiting middleware to prevent abuse
    - _Requirements: 2.3_

  - [x] 9.4 Add caching for referral code validation
    - Implement Redis caching in ReferralManager.ValidateReferralCode
    - Cache valid codes for 5 minutes to reduce database queries
    - Invalidate cache when code is used or expires
    - Target: < 50ms response time for cached validations
    - _Requirements: 2.3, 3.2_

  - [x] 9.5 Verify bug condition exploration test now passes
    - **Property 1: Expected Behavior** - Referral System Activated
    - **IMPORTANT**: Re-run the SAME test from task 5.1 - do NOT write a new test
    - The test from task 5.1 encodes the expected behavior
    - When this test passes, it confirms the expected behavior is satisfied
    - Run bug condition exploration test from step 5.1
    - **EXPECTED OUTCOME**: Test PASSES (confirms bug is fixed)
    - _Requirements: 2.3_

  - [x] 9.6 Verify preservation tests still pass
    - **Property 2: Preservation** - Existing Features Maintained
    - **IMPORTANT**: Re-run the SAME tests from task 8.1 - do NOT write new tests
    - Run preservation property tests from step 8.1
    - **EXPECTED OUTCOME**: Tests PASS (confirms no regressions)
    - Confirm all tests still pass after fix (no performance impact)
    - _Requirements: 3.2_

### 10. تطبيق وظائف تصدير التقارير

- [x] 10. Implement report export functions

  - [x] 10.1 Implement exportToJSON function
    - Implement `exportToJSON` in `services/backend/internal/referral/reports.go`
    - Use `encoding/json` with `json.MarshalIndent` for readable output
    - Handle large reports with streaming if > 1000 records
    - Add error handling for marshaling failures
    - Target: < 2 seconds for reports with < 1000 records
    - _Bug_Condition: isBugCondition(input) where input.ExportFormat == 'json' AND ExportFunctionReturns == "not implemented"_
    - _Expected_Behavior: System SHALL successfully generate JSON reports (Property 3)_
    - _Preservation: Existing report generation SHALL maintain same performance (Requirement 3.7)_
    - _Requirements: 2.4, 3.7_

  - [x] 10.2 Implement exportToCSV function
    - Implement `exportToCSV` in `services/backend/internal/referral/reports.go`
    - Use `encoding/csv` with proper header row
    - Include columns: ReferralCode, ReferrerID, ReferredID, Status, CreatedAt, UsedAt
    - Handle large reports with streaming if > 1000 records
    - Add proper CSV escaping for special characters
    - Target: < 2 seconds for reports with < 1000 records
    - _Bug_Condition: isBugCondition(input) where input.ExportFormat == 'csv' AND ExportFunctionReturns == "not implemented"_
    - _Expected_Behavior: System SHALL successfully generate CSV reports (Property 3)_
    - _Preservation: Existing report generation SHALL maintain same performance (Requirement 3.7)_
    - _Requirements: 2.4, 3.7_

  - [x] 10.3 Implement exportToPDF function
    - Implement `exportToPDF` in `services/backend/internal/referral/reports.go`
    - Use `github.com/jung-kurt/gofpdf` (already in dependencies)
    - Create professional PDF layout with header, table, and footer
    - Include summary statistics at the top
    - Handle large reports with pagination (50 records per page)
    - Add proper error handling for PDF generation failures
    - Target: < 2 seconds for reports with < 1000 records
    - _Bug_Condition: isBugCondition(input) where input.ExportFormat == 'pdf' AND ExportFunctionReturns == "not implemented"_
    - _Expected_Behavior: System SHALL successfully generate PDF reports (Property 3)_
    - _Preservation: Existing report generation SHALL maintain same performance (Requirement 3.7)_
    - _Requirements: 2.4, 3.7_

  - [x] 10.4 Verify bug condition exploration test now passes
    - **Property 1: Expected Behavior** - Export Functions Implemented
    - **IMPORTANT**: Re-run the SAME test from task 6.1 - do NOT write a new test
    - The test from task 6.1 encodes the expected behavior
    - When this test passes, it confirms the expected behavior is satisfied
    - Run bug condition exploration test from step 6.1
    - **EXPECTED OUTCOME**: Test PASSES (confirms bug is fixed)
    - _Requirements: 2.4_

  - [x] 10.5 Verify preservation tests still pass
    - **Property 2: Preservation** - Report Data Integrity
    - **IMPORTANT**: Re-run the SAME tests from task 8.1 - do NOT write new tests
    - Run preservation property tests from step 8.1
    - **EXPECTED OUTCOME**: Tests PASS (confirms no regressions)
    - Confirm all tests still pass after fix (data structure unchanged)
    - _Requirements: 3.7_

### 11. تطبيق In-Memory Rate Limiter

- [x] 11. Implement in-memory rate limiter fallback

  - [x] 11.1 Create in-memory rate limiter implementation
    - Create new file `services/backend/internal/middleware/memory_rate_limiter.go`
    - Use `golang.org/x/time/rate` package (already in dependencies)
    - Implement per-tenant rate limiting with `sync.Map` for storage
    - Store limiter instances with tenant ID as key
    - Implement same limits as Redis version: 100 requests/minute per tenant
    - Target: < 1ms overhead per request
    - _Bug_Condition: isBugCondition(input) where input.RedisAvailable == false AND RateLimiterActive == false_
    - _Expected_Behavior: System SHALL activate in-memory rate limiter when Redis fails (Property 4)_
    - _Preservation: Rate limiting SHALL enforce same limits (Requirement 3.3)_
    - _Requirements: 2.5, 3.3_

  - [x] 11.2 Add cleanup goroutine for in-memory limiter
    - Implement periodic cleanup (every 10 minutes) to remove inactive limiters
    - Remove limiters that haven't been used in last 30 minutes
    - Prevent memory leaks from accumulating tenant limiters
    - Add graceful shutdown handling
    - _Requirements: 2.5_

  - [x] 11.3 Update main.go to use fallback rate limiter
    - Modify rate limiter initialization in `services/backend/main.go`
    - When Redis is unavailable, create in-memory rate limiter instead of no-op
    - Add automatic switch back to Redis when it becomes available
    - Log rate limiter mode (Redis or in-memory) at startup
    - _Requirements: 2.5_

  - [x] 11.4 Verify bug condition exploration test now passes
    - **Property 1: Expected Behavior** - Rate Limiting Always Active
    - **IMPORTANT**: Re-run the SAME test from task 7.1 - do NOT write a new test
    - The test from task 7.1 encodes the expected behavior
    - When this test passes, it confirms the expected behavior is satisfied
    - Run bug condition exploration test from step 7.1
    - **EXPECTED OUTCOME**: Test PASSES (confirms bug is fixed)
    - _Requirements: 2.5_

  - [x] 11.5 Verify preservation tests still pass
    - **Property 2: Preservation** - Rate Limiting Accuracy
    - **IMPORTANT**: Re-run the SAME tests from task 8.1 - do NOT write new tests
    - Run preservation property tests from step 8.1
    - **EXPECTED OUTCOME**: Tests PASS (confirms no regressions)
    - Confirm rate limits are enforced with same accuracy (100 requests/minute)
    - _Requirements: 3.3_

- [x] 12. Checkpoint - Feature activation complete
  - Ensure all feature tests pass
  - Verify referral system is accessible via API
  - Verify export functions work for all formats
  - Verify rate limiting works with and without Redis
  - Ask user if questions arise

## المرحلة 3: إصلاح مشاكل الأداء والتكوين (Performance & Configuration)

### 13. اختبار استكشاف Bug Condition - Active Invalidation للقنوات

- [x] 13.1 Write bug condition exploration test for missing active invalidation
  - **Property 1: Bug Condition** - Cache Not Invalidated On Channel Offline
  - **CRITICAL**: Write this property-based test BEFORE implementing the fix
  - **GOAL**: Surface counterexamples that demonstrate cache is not invalidated when channel goes offline
  - **Scoped PBT Approach**: Test that when channel status changes to offline, cache entry remains for full TTL (5 minutes)
  - Test implementation details from Bug Condition in design (isBugCondition where ChannelStatus == 'offline' AND CacheInvalidated == false)
  - The test assertions should match the Expected Behavior Properties from design (Property 5)
  - Run test on UNFIXED code
  - **EXPECTED OUTCOME**: Test FAILS (this is correct - it proves the bug exists)
  - Document counterexamples found (stale cache causing failed sends)
  - Mark task complete when test is written, run, and failure is documented
  - _Requirements: 2.6_

### 14. اختبار Preservation - الحفاظ على آلية الـ caching الحالية

- [x] 14.1 Write preservation property tests for channel caching (BEFORE implementing fix)
  - **Property 2: Preservation** - Channel Cache Performance
  - **IMPORTANT**: Follow observation-first methodology
  - Observe: Cache hit rates for active channels on unfixed code
  - Observe: Cache TTL behavior (5 minutes) on unfixed code
  - Observe: Cache update mechanism on unfixed code
  - Write property-based test: for all active channels, cache should maintain same TTL and performance
  - Verify test passes on UNFIXED code
  - _Requirements: 3.4_

### 15. تطبيق Active Invalidation للقنوات

- [x] 15. Implement active cache invalidation for channels

  - [x] 15.1 Add InvalidateChannelOnOffline function
    - Add function in `services/backend/internal/redis/channel_cache.go`
    - Implement immediate cache deletion when channel goes offline
    - Use Redis DEL command for instant removal
    - Add error handling and logging
    - Target: < 100ms invalidation time
    - _Bug_Condition: isBugCondition(input) where input.ChannelStatus == 'offline' AND CacheInvalidated == false_
    - _Expected_Behavior: System SHALL immediately invalidate cache when channel goes offline (Property 5)_
    - _Preservation: Active channels SHALL maintain same cache behavior (Requirement 3.4)_
    - _Requirements: 2.6, 3.4_

  - [x] 15.2 Hook invalidation into channel status updates
    - Modify channel service in `services/backend/internal/channel/service.go`
    - Call InvalidateChannelOnOffline when status changes to offline
    - Add transaction handling to ensure atomicity
    - Add logging for cache invalidation events
    - _Requirements: 2.6_

  - [x] 15.3 Add periodic health check for channels
    - Implement background goroutine to check channel health every 1 minute
    - Detect offline channels and trigger cache invalidation
    - Add graceful shutdown handling
    - Log health check results
    - _Requirements: 2.6_

  - [x] 15.4 Verify bug condition exploration test now passes
    - **Property 1: Expected Behavior** - Active Cache Invalidation
    - **IMPORTANT**: Re-run the SAME test from task 13.1 - do NOT write a new test
    - The test from task 13.1 encodes the expected behavior
    - When this test passes, it confirms the expected behavior is satisfied
    - Run bug condition exploration test from step 13.1
    - **EXPECTED OUTCOME**: Test PASSES (confirms bug is fixed)
    - _Requirements: 2.6_

  - [x] 15.5 Verify preservation tests still pass
    - **Property 2: Preservation** - Cache Performance Maintained
    - **IMPORTANT**: Re-run the SAME tests from task 14.1 - do NOT write new tests
    - Run preservation property tests from step 14.1
    - **EXPECTED OUTCOME**: Tests PASS (confirms no regressions)
    - Confirm cache still works normally for active channels (5-minute TTL)
    - _Requirements: 3.4_

### 16. إكمال تكوين Grafana

- [x] 16. Complete Grafana configuration

  - [x] 16.1 Add Grafana environment variables to .env
    - Add GRAFANA_ADMIN_USER=admin to `.env`
    - Generate secure password: `openssl rand -base64 24`
    - Add GRAFANA_ADMIN_PASSWORD=<generated_password> to `.env`
    - Add GRAFANA_ROOT_URL=http://localhost:3000 to `.env`
    - Add GRAFANA_PLUGINS= to `.env` (empty for now)
    - _Bug_Condition: isBugCondition(input) where input.GrafanaConfigured == false_
    - _Expected_Behavior: System SHALL include all Grafana configuration variables (Property 6)_
    - _Preservation: System SHALL work normally even if Grafana is not configured (Requirement 3.5)_
    - _Requirements: 2.7, 3.5_

  - [x] 16.2 Update .env.example with Grafana variables
    - Add Grafana section to `.env.example`
    - Include comments explaining each variable
    - Provide example values and generation commands
    - Mark as optional feature
    - _Requirements: 2.7_

  - [x] 16.3 Add Grafana configuration validation (optional)
    - Add optional validation in `services/backend/internal/config/config.go`
    - Log warning if Grafana variables are missing (not error)
    - System should continue to work without Grafana
    - _Requirements: 2.7, 3.5_

### 17. توحيد PORT في جميع الملفات

- [x] 17. Standardize PORT configuration

  - [x] 17.1 Update PORT in .env to 8081
    - Change PORT=8080 to PORT=8081 in `.env`
    - Match the default value in main.go
    - Match the documentation in README.md
    - _Bug_Condition: isBugCondition(input) where input.PORT_in_env != input.PORT_in_readme_
    - _Expected_Behavior: System SHALL use consistent PORT across all files (Property 6)_
    - _Preservation: System SHALL accept connections on same port as before (Requirement 3.6)_
    - _Requirements: 2.8, 3.6_

  - [x] 17.2 Verify README.md uses PORT 8081
    - Check that README.md mentions `http://localhost:8081`
    - Update if necessary to match .env
    - Ensure all API examples use correct port
    - _Requirements: 2.8_

  - [x] 17.3 Verify main.go default PORT is 8081
    - Confirm `main.go` uses `port = "8081"` as default
    - No changes needed (already correct)
    - _Requirements: 2.8_

- [x] 18. Checkpoint - All fixes complete
  - Ensure all tests pass (exploration + preservation)
  - Verify system starts successfully with all fixes
  - Verify no performance degradation (< 5ms overhead)
  - Verify all existing tests still pass (bug15, bug17, bug23, bug26, bug28, bug29)
  - Ask user if questions arise

## ملاحظات مهمة

### ترتيب التنفيذ
1. **المرحلة 1 (الأمان)**: يجب تنفيذها أولاً لأنها حرجة
2. **المرحلة 2 (الوظائف)**: يمكن تنفيذها بالتوازي بعد المرحلة 1
3. **المرحلة 3 (التحسينات)**: يمكن تنفيذها في أي وقت

### متطلبات الأداء
- جميع الإصلاحات يجب أن تحافظ على الأداء الحالي (< 5ms overhead)
- Rate limiter: < 1ms overhead per request
- Cache invalidation: < 100ms
- Report export: < 2 seconds for < 1000 records
- Referral code validation: < 50ms (with caching)

### الاختبارات
- كل مشكلة لها اختبار استكشاف (exploration test) يُكتب قبل الإصلاح
- كل مرحلة لها اختبار preservation للتأكد من عدم كسر الوظائف الموجودة
- جميع الاختبارات الموجودة (bug15, bug17, bug23, bug26, bug28, bug29) يجب أن تستمر بالنجاح

### التوافق
- لا تغيير في API responses
- لا تغيير في database schema
- لا تغيير في frontend interfaces
- جميع التطبيقات الأمامية تستمر بالعمل دون تعديلات
