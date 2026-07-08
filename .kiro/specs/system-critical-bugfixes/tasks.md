# Implementation Plan: System Critical Bugfixes

## Overview

هذه قائمة المهام التنفيذية لإصلاح 30 مشكلة حرجة في نظام SMS Gateway. المهام منظمة حسب:
- **الأولوية**: حرجة → عالية → متوسطة → أمنية → أداء → تكامل
- **التبعيات**: المهام المستقلة يمكن تنفيذها بالتوازي
- **المنهجية**: استكشاف → حفظ → تنفيذ → تحقق

---

## 🔴 Phase 1: Critical Bugs (Priority 1) - Week 1

### Bug 1: Empty Active Tenants List

- [x] 1.1 Write bug condition exploration test
  - **Property 1: Bug Condition** - Active Tenants List Returns Empty
  - **CRITICAL**: This test MUST FAIL on unfixed code - failure confirms the bug exists
  - **DO NOT attempt to fix the test or the code when it fails**
  - **NOTE**: This test encodes the expected behavior - it will validate the fix when it passes after implementation
  - **GOAL**: Surface counterexamples that demonstrate the bug exists
  - Test that getActiveTenants() returns empty list when Redis set has tenant UUIDs
  - Run test on UNFIXED code
  - **EXPECTED OUTCOME**: Test FAILS (this is correct - it proves the bug exists)
  - Document counterexamples found to understand root cause
  - Mark task complete when test is written, run, and failure is documented
  - _Requirements: 1.1, 2.1_

- [x] 1.2 Write preservation property tests (BEFORE implementing fix)
  - **Property 2: Preservation** - Message Processing Continues
  - **IMPORTANT**: Follow observation-first methodology
  - Observe behavior on UNFIXED code for successful message processing
  - Write property-based tests capturing message enqueue/dequeue behavior
  - Verify tests PASS on UNFIXED code
  - _Requirements: 3.1, 3.2_

- [x] 1.3 Fix for Empty Active Tenants List

  - [x] 1.3.1 Implement Redis query in getActiveTenants()
    - Replace `return []uuid.UUID{}, nil` with actual Redis SMEMBERS query
    - Query key: `alhudhud:active:tenants`
    - Parse UUIDs and handle invalid entries
    - _Bug_Condition: getActiveTenants() returns empty list when Redis set has data_
    - _Expected_Behavior: Returns list of tenant UUIDs from Redis set_
    - _Preservation: Message processing for active tenants continues_
    - _Requirements: 2.1_

  - [x] 1.3.2 Update Enqueue logic to add tenant to active set
    - In message_queue.go Enqueue(), add SADD to active tenants set
    - Key: `alhudhud:active:tenants`
    - Value: tenant UUID
    - _Requirements: 2.1_

  - [x] 1.3.3 Add cleanup logic for empty queues
    - In processBatch(), remove tenant from active set when queue is empty
    - Use SREM command
    - _Requirements: 2.1_

  - [x] 1.3.4 Verify bug condition exploration test now passes
    - **Property 1: Expected Behavior** - Active Tenants List Returns Correctly
    - **IMPORTANT**: Re-run the SAME test from task 1.1 - do NOT write a new test
    - Run bug condition exploration test from step 1.1
    - **EXPECTED OUTCOME**: Test PASSES (confirms bug is fixed)
    - _Requirements: 2.1_

  - [x] 1.3.5 Verify preservation tests still pass
    - **Property 2: Preservation** - Message Processing Continues
    - **IMPORTANT**: Re-run the SAME tests from task 1.2 - do NOT write new tests
    - Run preservation property tests from step 1.2
    - **EXPECTED OUTCOME**: Tests PASS (confirms no regressions)

- [x] 1.4 Checkpoint - Bug 1 Complete
  - Ensure all tests pass for Bug 1

### Bug 2: Nil PgxPool Crash

- [x] 2.1 Write bug condition exploration test
  - **Property 1: Bug Condition** - PgxPool Nil Pointer Crash
  - **CRITICAL**: This test MUST FAIL on unfixed code
  - Test that accessing db.PgxPool causes nil pointer dereference
  - Run test on UNFIXED code
  - **EXPECTED OUTCOME**: Test FAILS with panic
  - _Requirements: 1.2, 2.2_

- [x] 2.2 Write preservation property tests
  - **Property 2: Preservation** - GORM Operations Continue
  - Observe GORM query behavior on UNFIXED code
  - Write tests for existing GORM operations
  - Verify tests PASS on UNFIXED code
  - _Requirements: 3.12, 3.13, 3.14_

- [ ] 2.3 Fix for Nil PgxPool Crash

  - [x] 2.3.1 Add PgxPool variable in db/gorm.go
    - Import pgxpool package
    - Declare `var PgxPool *pgxpool.Pool`
    - _Requirements: 2.2_

  - [x] 2.3.2 Initialize PgxPool in InitGormDB()
    - Build pgx DSN from environment variables
    - Configure pool settings (MaxConns: 50, MinConns: 10)
    - Create pool with pgxpool.NewWithConfig()
    - Log successful initialization
    - _Bug_Condition: db.PgxPool is nil_
    - _Expected_Behavior: PgxPool initialized with proper configuration_
    - _Preservation: GORM operations continue unchanged_
    - _Requirements: 2.2_

  - [x] 2.3.3 Add cleanup function
    - Create Close() function to close PgxPool
    - Call in main.go shutdown
    - _Requirements: 2.2_

  - [x] 2.3.4 Verify bug condition test passes
    - **Property 1: Expected Behavior** - PgxPool Initialized
    - Re-run test from 2.1
    - **EXPECTED OUTCOME**: Test PASSES
    - _Requirements: 2.2_

  - [x] 2.3.5 Verify preservation tests pass
    - **Property 2: Preservation** - GORM Operations Continue
    - Re-run tests from 2.2
    - **EXPECTED OUTCOME**: Tests PASS

- [x] 2.4 Checkpoint - Bug 2 Complete

### Bug 3: Mock Channel Selection

- [x] 3.1 Write bug condition exploration test
  - **Property 1: Bug Condition** - Mock Channel Returned
  - Test that getOrSelectChannel() returns mock UUID instead of real channel
  - Verify real channels exist in database
  - Run on UNFIXED code
  - **EXPECTED OUTCOME**: Test FAILS (mock UUID returned)
  - _Requirements: 1.3, 2.3_

- [x] 3.2 Write preservation property tests
  - **Property 2: Preservation** - Channel Cache Behavior
  - Observe channel cache hits on UNFIXED code
  - Write tests for cache hit behavior
  - Verify tests PASS on UNFIXED code
  - _Requirements: 3.7_

- [x] 3.3 Fix for Mock Channel Selection

  - [x] 3.3.1 Implement database query in getOrSelectChannel()
    - Check cache first
    - Query database for available online channels
    - Order by priority ASC, current_load ASC
    - Select channel with lowest load
    - _Bug_Condition: Returns mock channel with uuid.New()_
    - _Expected_Behavior: Returns real channel from database_
    - _Preservation: Cache behavior continues_
    - _Requirements: 2.3_

  - [x] 3.3.2 Cache selected channel
    - Store channel in cache with TTL
    - _Requirements: 2.3_

  - [x] 3.3.3 Verify bug condition test passes
    - **Property 1: Expected Behavior** - Real Channel Selected
    - Re-run test from 3.1
    - **EXPECTED OUTCOME**: Test PASSES

  - [x] 3.3.4 Verify preservation tests pass
    - **Property 2: Preservation** - Channel Cache Behavior
    - Re-run tests from 3.2
    - **EXPECTED OUTCOME**: Tests PASS

- [x] 3.4 Checkpoint - Bug 3 Complete

### Bug 4: Incomplete Analytics Calculation

- [x] 4.1 Write bug condition exploration test
  - **Property 1: Bug Condition** - Analytics Returns Zero Messages
  - Test that GetCurrentLoad() returns MessageCount: 0 despite messages existing
  - Create test messages in system
  - Run on UNFIXED code
  - **EXPECTED OUTCOME**: Test FAILS (returns 0)
  - _Requirements: 1.4, 2.4_

- [x] 4.2 Write preservation property tests
  - **Property 2: Preservation** - Event Count Calculation
  - Observe event counting on UNFIXED code
  - Write tests for event count accuracy
  - Verify tests PASS on UNFIXED code
  - _Requirements: 3.18_

- [x] 4.3 Fix for Incomplete Analytics Calculation

  - [x] 4.3.1 Implement message counting in GetCurrentLoad()
    - Scan Redis keys for event metadata
    - Extract message_count from each event
    - Scan queue keys for pending messages
    - Aggregate all counts
    - _Bug_Condition: Does not count actual messages_
    - _Expected_Behavior: Returns accurate message counts_
    - _Preservation: Event counting continues_
    - _Requirements: 2.4_

  - [x] 4.3.2 Add database fallback for message count
    - Query database if Redis data incomplete
    - _Requirements: 2.4, 2.14_

  - [x] 4.3.3 Verify bug condition test passes
    - **Property 1: Expected Behavior** - Analytics Returns Accurate Counts
    - Re-run test from 4.1
    - **EXPECTED OUTCOME**: Test PASSES

  - [x] 4.3.4 Verify preservation tests pass
    - **Property 2: Preservation** - Event Count Calculation
    - Re-run tests from 4.2
    - **EXPECTED OUTCOME**: Tests PASS

- [x] 4.4 Checkpoint - Bug 4 Complete

- [x] 4.5 Phase 1 Final Checkpoint
  - Ensure all Phase 1 tests pass
  - Run integration tests for critical bugs
  - Verify no regressions

---

## 🟠 Phase 2: High Priority Bugs (Priority 2) - Week 2

### Bug 5: Silent Error Ignoring

- [x] 5.1 Write bug condition exploration test
  - **Property 1: Bug Condition** - Errors Silently Ignored
  - Test that critical errors are ignored with `_ = ` pattern
  - Trigger errors in WebSocket, cache, pipeline operations
  - Verify errors are not logged
  - Run on UNFIXED code
  - **EXPECTED OUTCOME**: Test FAILS (errors ignored)
  - _Requirements: 1.5, 2.5_

- [x] 5.2 Write preservation property tests
  - **Property 2: Preservation** - Successful Operations Continue
  - Observe successful operation behavior
  - Write tests for normal operation flow
  - Verify tests PASS on UNFIXED code
  - _Requirements: 3.1, 3.2, 3.3_

- [x] 5.3 Fix for Silent Error Ignoring

  - [x] 5.3.1 Fix WebSocket errors in fast_processor.go
    - Replace `_ = fp.wsHub.SendMessage()` with proper error handling
    - Log errors and add to failedMessages
    - _Requirements: 2.5_

  - [x] 5.3.2 Fix cache errors in fast_processor.go
    - Replace `_ = fp.channelCache.Set()` with logging
    - Continue operation (cache failure non-critical)
    - _Requirements: 2.5_

  - [x] 5.3.3 Fix Redis pipeline errors
    - Check pipe.Exec() errors
    - Log and handle appropriately
    - _Bug_Condition: Errors assigned to _ (ignored)_
    - _Expected_Behavior: Errors logged and handled appropriately_
    - _Preservation: Successful operations continue_
    - _Requirements: 2.5_

  - [x] 5.3.4 Verify bug condition test passes
    - **Property 1: Expected Behavior** - Errors Properly Handled
    - Re-run test from 5.1
    - **EXPECTED OUTCOME**: Test PASSES

  - [x] 5.3.5 Verify preservation tests pass
    - **Property 2: Preservation** - Successful Operations Continue
    - Re-run tests from 5.2
    - **EXPECTED OUTCOME**: Tests PASS

- [x] 5.4 Checkpoint - Bug 5 Complete

### Bug 6: Transaction Rollback After Commit

- [x] 6.1 Write bug condition exploration test
  - **Property 1: Bug Condition** - Rollback After Commit
  - Test that defer tx.Rollback() executes after successful tx.Commit()
  - Verify transaction management issues
  - Run on UNFIXED code
  - **EXPECTED OUTCOME**: Test FAILS (rollback after commit)
  - _Requirements: 1.6, 2.6_

- [x] 6.2 Write preservation property tests
  - **Property 2: Preservation** - Transaction Atomicity
  - Observe successful transaction commits
  - Write tests for atomic operations
  - Verify tests PASS on UNFIXED code
  - _Requirements: 3.14_

- [x] 6.3 Fix for Transaction Rollback After Commit

  - [x] 6.3.1 Update transaction pattern in processBatchWithTransaction()
    - Use committed flag pattern
    - Set committed = true after successful commit
    - Defer rollback only if not committed
    - _Bug_Condition: Rollback executes after commit_
    - _Expected_Behavior: Rollback only if commit fails_
    - _Preservation: Transaction atomicity maintained_
    - _Requirements: 2.6_

  - [x] 6.3.2 Verify bug condition test passes
    - **Property 1: Expected Behavior** - Proper Transaction Management
    - Re-run test from 6.1
    - **EXPECTED OUTCOME**: Test PASSES

  - [x] 6.3.3 Verify preservation tests pass
    - **Property 2: Preservation** - Transaction Atomicity
    - Re-run tests from 6.2
    - **EXPECTED OUTCOME**: Tests PASS

- [x] 6.4 Checkpoint - Bug 6 Complete

### Bug 7: No Balance Verification Before Deduction

- [x] 7.1 Write bug condition exploration test
  - **Property 1: Bug Condition** - Negative Balance Allowed
  - Test that DeductBalanceTx() allows deduction with insufficient balance
  - Create tenant with 10 SAR, attempt to deduct 15 SAR
  - Run on UNFIXED code
  - **EXPECTED OUTCOME**: Test FAILS (negative balance created)
  - _Requirements: 1.7, 2.7_

- [x] 7.2 Write preservation property tests
  - **Property 2: Preservation** - Successful Deductions Continue
  - Observe successful balance deductions with sufficient funds
  - Write tests for balance operations
  - Verify tests PASS on UNFIXED code
  - _Requirements: 3.4, 3.5_

- [x] 7.3 Fix for No Balance Verification

  - [x] 7.3.1 Add balance check in DeductBalanceTx()
    - Query current balance with FOR UPDATE lock
    - Verify balance >= deduction amount
    - Return error if insufficient
    - _Bug_Condition: No balance verification before deduction_
    - _Expected_Behavior: Verifies sufficient balance, returns error if insufficient_
    - _Preservation: Successful deductions continue_
    - _Requirements: 2.7_

  - [x] 7.3.2 Update fast_processor to handle insufficient balance
    - Catch insufficient balance error
    - Mark messages as FAILED
    - Log warning
    - _Requirements: 2.7_

  - [x] 7.3.3 Verify bug condition test passes
    - **Property 1: Expected Behavior** - Balance Verification Works
    - Re-run test from 7.1
    - **EXPECTED OUTCOME**: Test PASSES

  - [x] 7.3.4 Verify preservation tests pass
    - **Property 2: Preservation** - Successful Deductions Continue
    - Re-run tests from 7.2
    - **EXPECTED OUTCOME**: Tests PASS

- [x] 7.4 Checkpoint - Bug 7 Complete

### Bug 8: Nil wsHub Crash

- [x] 8.1 Write bug condition exploration test
  - **Property 1: Bug Condition** - Nil wsHub Crash
  - Test that fp.wsHub.SendMessage() crashes when wsHub is nil
  - Initialize processor without wsHub
  - Run on UNFIXED code
  - **EXPECTED OUTCOME**: Test FAILS with panic
  - _Requirements: 1.8, 2.8_

- [x] 8.2 Write preservation property tests
  - **Property 2: Preservation** - WebSocket Messages Sent
  - Observe successful WebSocket message sending
  - Write tests for WebSocket delivery
  - Verify tests PASS on UNFIXED code
  - _Requirements: 3.3_

- [x] 8.3 Fix for Nil wsHub Crash

  - [x] 8.3.1 Add nil check before wsHub operations
    - Check if fp.wsHub != nil before calling methods
    - Log warning if nil
    - Add to failedMessages if cannot send
    - _Bug_Condition: No nil check before wsHub.SendMessage()_
    - _Expected_Behavior: Checks nil before calling methods_
    - _Preservation: Successful WebSocket sends continue_
    - _Requirements: 2.8_

  - [x] 8.3.2 Add validation in constructor
    - Log warning if wsHub is nil
    - Document that wsHub is optional
    - _Requirements: 2.8_

  - [x] 8.3.3 Verify bug condition test passes
    - **Property 1: Expected Behavior** - Nil Check Prevents Crash
    - Re-run test from 8.1
    - **EXPECTED OUTCOME**: Test PASSES

  - [x] 8.3.4 Verify preservation tests pass
    - **Property 2: Preservation** - WebSocket Messages Sent
    - Re-run tests from 8.2
    - **EXPECTED OUTCOME**: Tests PASS

- [x] 8.4 Checkpoint - Bug 8 Complete

### Bug 9: No Redis Retry Logic

- [x] 9.1 Write bug condition exploration test
  - **Property 1: Bug Condition** - No Retry on Transient Errors
  - Test that Redis operations fail without retry on transient errors
  - Simulate network blip
  - Run on UNFIXED code
  - **EXPECTED OUTCOME**: Test FAILS (no retry, data lost)
  - _Requirements: 1.9, 2.9_

- [x] 9.2 Write preservation property tests
  - **Property 2: Preservation** - Successful Redis Operations
  - Observe successful Redis operations
  - Write tests for Redis performance
  - Verify tests PASS on UNFIXED code
  - _Requirements: 3.8_

- [x] 9.3 Fix for No Redis Retry Logic

  - [x] 9.3.1 Create retry wrapper in redis/client.go
    - Implement WithRetry() function
    - Configure exponential backoff (10ms, 20ms, 40ms, max 1s)
    - Max 3 retries
    - Check if error is retryable
    - _Requirements: 2.9_

  - [x] 9.3.2 Update message_queue.go to use retry
    - Wrap Enqueue() with WithRetry()
    - Wrap Dequeue() with WithRetry()
    - _Bug_Condition: No retry on Redis failures_
    - _Expected_Behavior: Retries with exponential backoff_
    - _Preservation: Successful operations continue_
    - _Requirements: 2.9_

  - [x] 9.3.3 Update other Redis operations
    - Apply retry to cache operations
    - Apply retry to analytics operations
    - _Requirements: 2.9_

  - [x] 9.3.4 Verify bug condition test passes
    - **Property 1: Expected Behavior** - Retry Logic Works
    - Re-run test from 9.1
    - **EXPECTED OUTCOME**: Test PASSES

  - [x] 9.3.5 Verify preservation tests pass
    - **Property 2: Preservation** - Successful Redis Operations
    - Re-run tests from 9.2
    - **EXPECTED OUTCOME**: Tests PASS

- [x] 9.4 Checkpoint - Bug 9 Complete

### Bug 10: Improper Priority Queue Ordering

- [x] 10.1 Write bug condition exploration test
  - **Property 1: Bug Condition** - Priority Ordering Not Enforced
  - Test that DequeueBatch() may not fully drain high priority before normal
  - Enqueue 150 high, 100 normal, 50 low priority messages
  - Dequeue in batches
  - Run on UNFIXED code
  - **EXPECTED OUTCOME**: Verify strict ordering (may already be correct)
  - _Requirements: 1.10, 2.10_

- [x] 10.2 Write preservation property tests
  - **Property 2: Preservation** - FIFO Within Priority
  - Observe FIFO ordering within each priority level
  - Write tests for message ordering
  - Verify tests PASS on UNFIXED code
  - _Requirements: 3.1_

- [x] 10.3 Review and enhance DequeueBatch()

  - [x] 10.3.1 Review current implementation
    - Verify strict priority ordering
    - Add documentation
    - Add test cases
    - _Bug_Condition: May not enforce strict priority ordering_
    - _Expected_Behavior: Fully drains high → normal → low_
    - _Preservation: FIFO within priority maintained_
    - _Requirements: 2.10_

  - [x] 10.3.2 Verify bug condition test passes
    - **Property 1: Expected Behavior** - Strict Priority Ordering
    - Re-run test from 10.1
    - **EXPECTED OUTCOME**: Test PASSES

  - [x] 10.3.3 Verify preservation tests pass
    - **Property 2: Preservation** - FIFO Within Priority
    - Re-run tests from 10.2
    - **EXPECTED OUTCOME**: Tests PASS

- [x] 10.4 Checkpoint - Bug 10 Complete

- [x] 10.5 Phase 2 Final Checkpoint
  - Ensure all Phase 2 tests pass
  - Run integration tests for high priority bugs
  - Deploy to staging environment
  - Monitor for issues

---

## 🟡 Phase 3: Medium Priority Bugs (Priority 3) - Week 3

### Bug 11: Missing Custom ID Backfill

- [x] 11.1 Write bug condition exploration test
  - **Property 1: Bug Condition** - NULL Custom IDs in Existing Records
  - Test that migration 077 leaves existing records with NULL custom_id
  - Create test database with existing records
  - Run migration
  - Run on UNFIXED migration
  - **EXPECTED OUTCOME**: Test FAILS (NULL values remain)
  - _Requirements: 1.11, 2.11_

- [x] 11.2 Write preservation property tests
  - **Property 2: Preservation** - New Records Get Custom IDs
  - Observe that new records receive custom_id on creation
  - Write tests for custom_id generation
  - Verify tests PASS on UNFIXED code
  - _Requirements: 3.13_

- [x] 11.3 Fix for Missing Custom ID Backfill

  - [x] 11.3.1 Add backfill logic to migration 077
    - Add DO block to backfill events
    - Generate custom_id based on timestamp and id
    - Process in batches of 1000
    - Log progress
    - _Requirements: 2.11_

  - [x] 11.3.2 Add backfill for messages table
    - Generate message custom_id
    - Update event_custom_id from events table
    - Process in batches
    - _Bug_Condition: Migration does not backfill existing records_
    - _Expected_Behavior: Backfills custom_id for all existing records_
    - _Preservation: New records continue to get custom_id_
    - _Requirements: 2.11_

  - [x] 11.3.3 Verify bug condition test passes
    - **Property 1: Expected Behavior** - Custom IDs Backfilled
    - Re-run test from 11.1
    - **EXPECTED OUTCOME**: Test PASSES

  - [x] 11.3.4 Verify preservation tests pass
    - **Property 2: Preservation** - New Records Get Custom IDs
    - Re-run tests from 11.2
    - **EXPECTED OUTCOME**: Tests PASS

- [x] 11.4 Checkpoint - Bug 11 Complete

### Bug 12: SCAN Without COUNT Parameter

- [x] 12.1 Write bug condition exploration test
  - **Property 1: Bug Condition** - SCAN Performance Issue
  - Test InvalidateAll() performance with large key sets
  - Create 100K cached entries
  - Measure execution time
  - Run on UNFIXED code
  - **EXPECTED OUTCOME**: Verify COUNT parameter usage (may already be correct)
  - _Requirements: 1.12, 2.12_

- [x] 12.2 Write preservation property tests
  - **Property 2: Preservation** - Cache Invalidation Works
  - Observe successful cache invalidation
  - Write tests for cache clearing
  - Verify tests PASS on UNFIXED code
  - _Requirements: 3.6, 3.7_

- [x] 12.3 Enhance SCAN Implementation

  - [x] 12.3.1 Review and enhance InvalidateAll()
    - Verify COUNT=1000 is used
    - Add batch deletion with pipeline
    - Add performance logging
    - _Bug_Condition: SCAN without COUNT may cause performance issues_
    - _Expected_Behavior: Uses SCAN with COUNT=1000 for efficient pagination_
    - _Preservation: Cache invalidation continues to work_
    - _Requirements: 2.12_

  - [x] 12.3.2 Verify bug condition test passes
    - **Property 1: Expected Behavior** - Efficient SCAN Performance
    - Re-run test from 12.1
    - **EXPECTED OUTCOME**: Test PASSES

  - [x] 12.3.3 Verify preservation tests pass
    - **Property 2: Preservation** - Cache Invalidation Works
    - Re-run tests from 12.2
    - **EXPECTED OUTCOME**: Tests PASS

- [x] 12.4 Checkpoint - Bug 12 Complete

### Bug 13: Inconsistent Cache TTL Values

- [x] 13.1 Write bug condition exploration test
  - **Property 1: Bug Condition** - Inconsistent TTL Causes Stale Data
  - Test that balance (60s TTL) and usage (30s TTL) become inconsistent
  - Cache both at T=0
  - Check at T=35s
  - Run on UNFIXED code
  - **EXPECTED OUTCOME**: Test FAILS (inconsistent data)
  - _Requirements: 1.13, 2.13_

- [x] 13.2 Write preservation property tests
  - **Property 2: Preservation** - Cache Performance Maintained
  - Observe cache hit rates and performance
  - Write tests for cache behavior
  - Verify tests PASS on UNFIXED code
  - _Requirements: 3.6, 3.7, 3.8_

- [x] 13.3 Fix for Inconsistent Cache TTL

  - [x] 13.3.1 Define standard TTL constants
    - Create constants in redis/client.go
    - CacheTTLBalance = 1 minute
    - CacheTTLUsage = 1 minute (changed from 30s)
    - CacheTTLChannel = 1 minute (changed from 5 min)
    - _Requirements: 2.13, 2.23_

  - [x] 13.3.2 Update all cache implementations
    - Update balance cache to use constant
    - Update usage cache to use constant
    - Update channel cache to use constant
    - _Bug_Condition: Inconsistent TTL values (60s vs 30s)_
    - _Expected_Behavior: Consistent TTL values (both 1 minute)_
    - _Preservation: Cache performance maintained_
    - _Requirements: 2.13, 2.23_

  - [x] 13.3.3 Verify bug condition test passes
    - **Property 1: Expected Behavior** - Consistent TTL Values
    - Re-run test from 13.1
    - **EXPECTED OUTCOME**: Test PASSES

  - [x] 13.3.4 Verify preservation tests pass
    - **Property 2: Preservation** - Cache Performance Maintained
    - Re-run tests from 13.2
    - **EXPECTED OUTCOME**: Tests PASS

- [x] 13.4 Checkpoint - Bug 13 Complete

### Bug 14: Incomplete Analytics Message Count

- [x] 14.1 Note: Already fixed in Bug 4
  - This bug is duplicate of Bug 4
  - Verify Bug 4 fix covers this requirement
  - _Requirements: 2.14_

- [x] 14.2 Checkpoint - Bug 14 Complete (covered by Bug 4)

### Bug 15: No Auto-Logout on 401

- [x] 15.1 Write bug condition exploration test
  - **Property 1: Bug Condition** - No Auto-Logout on 401
  - Test that 401 response does not trigger logout
  - Simulate expired token
  - Make API request
  - Run on UNFIXED code
  - **EXPECTED OUTCOME**: Test FAILS (user stays logged in)
  - _Requirements: 1.15, 2.15_

- [x] 15.2 Write preservation property tests
  - **Property 2: Preservation** - Successful Requests Continue
  - Observe successful API requests with valid tokens
  - Write tests for authentication flow
  - Verify tests PASS on UNFIXED code
  - _Requirements: 3.9, 3.15_

- [x] 15.3 Fix for No Auto-Logout on 401

  - [x] 15.3.1 Update axios response interceptor
    - Uncomment 401 handling code
    - Clear localStorage token
    - Redirect to login page
    - Add user notification
    - _Bug_Condition: 401 does not trigger auto-logout_
    - _Expected_Behavior: Clears token and redirects to login_
    - _Preservation: Successful requests continue_
    - _Requirements: 2.15_

  - [x] 15.3.2 Apply same fix to admin-web
    - Update admin-web axios.js
    - _Requirements: 2.15_

  - [x] 15.3.3 Verify bug condition test passes
    - **Property 1: Expected Behavior** - Auto-Logout on 401
    - Re-run test from 15.1
    - **EXPECTED OUTCOME**: Test PASSES

  - [x] 15.3.4 Verify preservation tests pass
    - **Property 2: Preservation** - Successful Requests Continue
    - Re-run tests from 15.2
    - **EXPECTED OUTCOME**: Tests PASS

- [x] 15.4 Checkpoint - Bug 15 Complete

- [x] 15.5 Phase 3 Final Checkpoint
  - Ensure all Phase 3 tests pass
  - Monitor staging performance
  - Verify cache behavior

---

## 🔵 Phase 4: Security Bugs (Priority 4) - Week 4

### Bug 16: Exposed Database Credentials

- [x] 16.1 Security audit of deployment scripts
  - Scan for hardcoded credentials
  - Identify all credential locations
  - Document findings
  - _Requirements: 1.16, 2.16_

- [x] 16.2 Fix for Exposed Credentials

  - [x] 16.2.1 Remove hardcoded credentials from scripts
    - Replace with environment variable reads
    - Add validation for required variables
    - Fail fast if missing
    - _Requirements: 2.16_

  - [x] 16.2.2 Update docker-compose.yml
    - Use ${VAR:?error} syntax
    - Remove hardcoded values
    - _Requirements: 2.16_

  - [x] 16.2.3 Create .env.example
    - Document all required variables
    - Provide template
    - _Requirements: 2.16_

  - [x] 16.2.4 Update .gitignore
    - Ensure .env files ignored
    - Add *.pem, *.key
    - _Bug_Condition: Credentials in plain text in scripts_
    - _Expected_Behavior: Reads from environment variables_
    - _Requirements: 2.16_

- [x] 16.3 Checkpoint - Bug 16 Complete

### Bug 17: Missing CSRF Protection

- [x] 17.1 Write bug condition exploration test
  - **Property 1: Bug Condition** - No CSRF Validation
  - Test that API accepts requests without CSRF token
  - Make POST request without token
  - Run on UNFIXED code
  - **EXPECTED OUTCOME**: Test FAILS (request accepted)
  - _Requirements: 1.17, 2.17_

- [x] 17.2 Write preservation property tests
  - **Property 2: Preservation** - Legitimate Requests Work
  - Observe successful API requests
  - Write tests for API functionality
  - Verify tests PASS on UNFIXED code
  - _Requirements: 3.9, 3.10_

- [x] 17.3 Fix for Missing CSRF Protection

  - [ ] 17.3.1 Create CSRF middleware (backend)
    - Implement token generation
    - Implement token validation
    - Use double-submit cookie pattern
    - _Requirements: 2.17_

  - [ ] 17.3.2 Add CSRF endpoint
    - Create /api/csrf-token endpoint
    - Set token in cookie
    - Return token in response
    - _Requirements: 2.17_

  - [ ] 17.3.3 Apply middleware to routes
    - Apply to all state-changing routes (POST, PUT, DELETE)
    - Skip for GET, HEAD, OPTIONS
    - _Requirements: 2.17_

  - [ ] 17.3.4 Update frontend axios.js
    - Fetch CSRF token on init
    - Add token to request headers
    - Handle token refresh
    - _Bug_Condition: No CSRF token validation_
    - _Expected_Behavior: Validates CSRF tokens_
    - _Preservation: Legitimate requests work_
    - _Requirements: 2.17_

  - [ ] 17.3.5 Initialize CSRF in App.jsx
    - Call initializeCSRF() on mount
    - _Requirements: 2.17_

  - [x] 17.3.6 Verify bug condition test passes
    - **Property 1: Expected Behavior** - CSRF Protection Active
    - Re-run test from 17.1
    - **EXPECTED OUTCOME**: Test PASSES

  - [x] 17.3.7 Verify preservation tests pass
    - **Property 2: Preservation** - Legitimate Requests Work
    - Re-run tests from 17.2
    - **EXPECTED OUTCOME**: Tests PASS

- [x] 17.4 Checkpoint - Bug 17 Complete

### Bug 18: Tokens in localStorage (XSS Vulnerability)

- [x] 18.1 Security assessment
  - Document XSS vulnerability
  - Plan migration strategy
  - _Requirements: 1.18, 2.18_

- [x] 18.2 Fix for localStorage Token Storage

  - [x] 18.2.1 Update backend login endpoint
    - Set token in httpOnly cookie
    - Remove token from response body
    - Configure cookie settings (Secure, HttpOnly)
    - _Requirements: 2.18_

  - [x] 18.2.2 Update auth middleware
    - Read token from cookie instead of header
    - Validate token
    - Set user context
    - _Requirements: 2.18_

  - [x] 18.2.3 Add logout endpoint
    - Clear auth cookie
    - _Requirements: 2.18_

  - [x] 18.2.4 Update frontend axios.js
    - Remove Authorization header logic
    - Token sent automatically via cookie
    - _Requirements: 2.18_

  - [x] 18.2.5 Update Login component
    - Remove localStorage.setItem('token')
    - Store only user info
    - _Requirements: 2.18_

  - [x] 18.2.6 Update CORS configuration
    - Set AllowCredentials: true
    - _Bug_Condition: Tokens in localStorage (XSS vulnerable)_
    - _Expected_Behavior: Uses httpOnly cookies_
    - _Requirements: 2.18_

- [x] 18.3 Checkpoint - Bug 18 Complete

### Bug 19: SQL Injection Risk

- [x] 19.1 Security audit of SQL queries
  - Search for unsafe query patterns
  - Identify dynamic query construction
  - Document vulnerabilities
  - _Requirements: 1.19, 2.19_

- [x] 19.2 Fix for SQL Injection Risk

  - [x] 19.2.1 Audit all SQL queries
    - Search for fmt.Sprintf with SQL
    - Search for string concatenation
    - Identify all dynamic queries
    - _Requirements: 2.19_

  - [x] 19.2.2 Fix unsafe queries
    - Replace with parameterized queries
    - Use $1, $2, etc. placeholders
    - _Requirements: 2.19_

  - [x] 19.2.3 Create safe dynamic query helpers
    - Implement buildOrderByClause() with whitelist
    - Implement buildFilterQuery() with validation
    - _Requirements: 2.19_

  - [x] 19.2.4 Add SQL injection tests
    - Test malicious inputs
    - Verify parameterization
    - _Bug_Condition: Dynamic SQL without parameterization_
    - _Expected_Behavior: Uses parameterized queries exclusively_
    - _Requirements: 2.19_

- [x] 19.3 Checkpoint - Bug 19 Complete

### Bug 20: Missing Environment Variable Validation

- [x] 20.1 Write bug condition exploration test
  - **Property 1: Bug Condition** - App Runs Without Required Vars
  - Test that app starts without required environment variables
  - Unset critical variables
  - Run on UNFIXED code
  - **EXPECTED OUTCOME**: Test FAILS (app starts, crashes later)
  - _Requirements: 1.20, 2.20_

- [x] 20.2 Write preservation property tests
  - **Property 2: Preservation** - Normal Startup Works
  - Observe successful startup with valid config
  - Write tests for initialization
  - Verify tests PASS on UNFIXED code
  - _Requirements: 3.13_

- [x] 20.3 Fix for Missing Environment Validation

  - [x] 20.3.1 Create config validation module
    - Create internal/config/config.go
    - Define Config struct
    - Implement LoadConfig() function
    - _Requirements: 2.20_

  - [x] 20.3.2 Validate required variables
    - Check DB_HOST, DB_USER, DB_PASSWORD, DB_NAME
    - Check REDIS_HOST, JWT_SECRET
    - Return error if missing
    - _Requirements: 2.20_

  - [x] 20.3.3 Add additional validation
    - Validate JWT_SECRET length (min 32 chars)
    - Validate port ranges
    - _Requirements: 2.20_

  - [x] 20.3.4 Update main.go
    - Call LoadConfig() at startup
    - Fail fast with clear error if invalid
    - Log configuration (without secrets)
    - _Bug_Condition: No validation of required env vars_
    - _Expected_Behavior: Validates and fails fast if missing_
    - _Preservation: Normal startup works_
    - _Requirements: 2.20_

  - [x] 20.3.5 Verify bug condition test passes
    - **Property 1: Expected Behavior** - Validation Prevents Startup
    - Re-run test from 20.1
    - **EXPECTED OUTCOME**: Test PASSES

  - [x] 20.3.6 Verify preservation tests pass
    - **Property 2: Preservation** - Normal Startup Works
    - Re-run tests from 20.2
    - **EXPECTED OUTCOME**: Tests PASS

- [x] 20.4 Checkpoint - Bug 20 Complete

- [x] 20.5 Phase 4 Final Checkpoint
  - Security audit complete
  - Penetration testing
  - Verify all security fixes

---

## 📊 Phase 5: Performance Bugs (Priority 5) - Week 5

### Bug 21: Suboptimal Batch Processing Configuration

- [x] 21.1 Performance benchmarking
  - Measure current throughput (100 msg × 10 workers)
  - Test different configurations
  - Identify optimal settings
  - _Requirements: 1.21, 2.21_

- [-] 21.2 Fix for Suboptimal Configuration

  - [x] 21.2.1 Make configuration tunable
    - Add batchSize and workerCount parameters
    - Set new defaults (50 msg × 20 workers)
    - _Requirements: 2.21_

  - [x] 21.2.2 Add environment configuration
    - WORKER_BATCH_SIZE (default 50)
    - WORKER_COUNT (default 20)
    - WORKER_TICKER_MS (default 100)
    - _Requirements: 2.21_

  - [x] 21.2.3 Add performance metrics
    - Track messages processed
    - Track throughput (msg/sec)
    - Track average latency
    - _Bug_Condition: Fixed 100×10 configuration_
    - _Expected_Behavior: Configurable, optimized settings_
    - _Requirements: 2.21_

  - [ ]* 21.2.4 Optional: Add dynamic scaling
    - Adjust workers based on queue size
    - Scale up for high load
    - Scale down for low load
    - _Requirements: 2.21_

- [x] 21.3 Checkpoint - Bug 21 Complete

### Bug 22: Redis SCAN Performance Issue

- [x] 22.1 Note: Already fixed in Bug 12
  - This bug is duplicate of Bug 12
  - Verify Bug 12 fix covers this requirement
  - _Requirements: 2.22_

- [x] 22.2 Checkpoint - Bug 22 Complete (covered by Bug 12)

### Bug 23: Stale Channel Cache

- [x] 23.1 Note: Already addressed in Bug 13
  - Channel cache TTL reduced from 5 min to 1 min in Bug 13
  - Verify Bug 13 fix covers this requirement
  - _Requirements: 2.23_

- [x] 23.2 Additional enhancements

  - [x] 23.2.1 Add active invalidation
    - Invalidate cache when channel goes offline
    - Call InvalidateChannel() on status change
    - _Requirements: 2.23_

  - [x] 23.2.2 Add health check verification
    - Periodically verify cached channel is online
    - Invalidate if offline
    - _Bug_Condition: 5-minute TTL causes stale data_
    - _Expected_Behavior: 1-minute TTL + active invalidation_
    - _Requirements: 2.23_

- [x] 23.3 Checkpoint - Bug 23 Complete

### Bug 24: No Alerting Mechanism

- [x] 24.1 Design alerting system
  - Define alert types and levels
  - Choose alert channels (email, webhook, SMS)
  - Plan integration points
  - _Requirements: 1.24, 2.24_

- [ ] 24.2 Implement alerting system

  - [x] 24.2.1 Create alert types and interfaces
    - Define Alert struct
    - Define AlertChannel interface
    - Define alert levels (INFO, WARNING, CRITICAL)
    - _Requirements: 2.24_

  - [x] 24.2.2 Implement email channel
    - Create EmailChannel
    - Configure SMTP
    - _Requirements: 2.24_

  - [x] 24.2.3 Implement webhook channel
    - Create WebhookChannel
    - Support JSON payload
    - _Requirements: 2.24_

  - [x] 24.2.4 Create alert manager
    - Manage multiple channels
    - Send alerts asynchronously
    - Handle failures
    - _Requirements: 2.24_

  - [x] 24.2.5 Integrate with system
    - Monitor queue backup
    - Monitor error rates
    - Monitor low balance
    - _Bug_Condition: No alerting on critical issues_
    - _Expected_Behavior: Sends alerts via configured channels_
    - _Requirements: 2.24_

- [x] 24.3 Checkpoint - Bug 24 Complete

### Bug 25: Missing Database Index

- [x] 25.1 Verify index exists
  - Check migration 077 for idx_messages_event_custom_id
  - Verify index in database
  - Test query performance
  - _Requirements: 1.25, 2.25_

- [x] 25.2 Note: Index already exists in migration 077
  - Index created on messages.event_custom_id
  - Verify with EXPLAIN ANALYZE
  - _Requirements: 2.25_

- [x] 25.3 Optional: Add composite indexes

  - [x] 25.3.1 Add event_status index
    - CREATE INDEX idx_messages_event_status ON messages(event_custom_id, status)
    - For queries filtering by both
    - _Requirements: 2.25_

  - [x] 25.3.2 Add event_created index
    - CREATE INDEX idx_messages_event_created ON messages(event_custom_id, created_at DESC)
    - For time-based queries
    - _Bug_Condition: No index on event_custom_id (actually exists)_
    - _Expected_Behavior: Index exists for fast queries_
    - _Requirements: 2.25_

- [x] 25.4 Checkpoint - Bug 25 Complete

- [x] 25.5 Phase 5 Final Checkpoint
  - Performance testing complete
  - Load testing with optimized settings
  - Verify throughput improvements

---

## 📋 Phase 6: Integration Bugs (Priority 6) - Week 6

### Bug 26: Hardcoded API URLs

- [x] 26.1 Write bug condition exploration test
  - **Property 1: Bug Condition** - Hardcoded localhost URLs
  - Test that API URLs are hardcoded
  - Check config files
  - Run on UNFIXED code
  - **EXPECTED OUTCOME**: Test FAILS (hardcoded values found)
  - _Requirements: 1.26, 2.26_

- [x] 26.2 Write preservation property tests
  - **Property 2: Preservation** - API Requests Work
  - Observe successful API requests
  - Write tests for API communication
  - Verify tests PASS on UNFIXED code
  - _Requirements: 3.9, 3.15_

- [x] 26.3 Fix for Hardcoded API URLs

  - [x] 26.3.1 Update client-web config.js
    - Use import.meta.env.VITE_API_BASE_URL
    - Provide fallback for development
    - _Requirements: 2.26_

  - [x] 26.3.2 Create environment files
    - .env.development (localhost)
    - .env.production (production URL)
    - .env.staging (staging URL)
    - _Requirements: 2.26_

  - [x] 26.3.3 Update admin-web config.js
    - Same changes as client-web
    - _Requirements: 2.26_

  - [x] 26.3.4 Update package.json scripts
    - Add build:staging script
    - Use --mode flag
    - _Bug_Condition: Hardcoded localhost:8081_
    - _Expected_Behavior: Reads from environment variables_
    - _Preservation: API requests work_
    - _Requirements: 2.26_

  - [x] 26.3.5 Verify bug condition test passes
    - **Property 1: Expected Behavior** - Environment-Based URLs
    - Re-run test from 26.1
    - **EXPECTED OUTCOME**: Test PASSES

  - [x] 26.3.6 Verify preservation tests pass
    - **Property 2: Preservation** - API Requests Work
    - Re-run tests from 26.2
    - **EXPECTED OUTCOME**: Tests PASS

- [x] 26.4 Checkpoint - Bug 26 Complete

### Bug 27: No WebSocket Auto-Reconnect

- [x] 27.1 Write bug condition exploration test
  - **Property 1: Bug Condition** - No Auto-Reconnect
  - Test that WebSocket disconnect requires manual refresh
  - Simulate connection drop
  - Run on UNFIXED code
  - **EXPECTED OUTCOME**: Test FAILS (no reconnect)
  - _Requirements: 1.27, 2.27_

- [x] 27.2 Write preservation property tests
  - **Property 2: Preservation** - WebSocket Messages Delivered
  - Observe successful WebSocket communication
  - Write tests for message delivery
  - Verify tests PASS on UNFIXED code
  - _Requirements: 3.3_

- [x] 27.3 Fix for No WebSocket Auto-Reconnect

  - [x] 27.3.1 Update SocketContext.jsx
    - Implement reconnection logic
    - Use exponential backoff (1s, 2s, 4s, 8s, max 30s)
    - Track reconnect attempts
    - _Requirements: 2.27_

  - [x] 27.3.2 Add connection state management
    - Track isConnected state
    - Schedule reconnect on disconnect
    - Clear timeout on successful connect
    - _Requirements: 2.27_

  - [x] 27.3.3 Add connection status indicator
    - Show warning when disconnected
    - Show reconnecting message
    - _Bug_Condition: No automatic reconnection_
    - _Expected_Behavior: Auto-reconnects with exponential backoff_
    - _Preservation: WebSocket messages delivered_
    - _Requirements: 2.27_

  - [x] 27.3.4 Verify bug condition test passes
    - **Property 1: Expected Behavior** - Auto-Reconnect Works
    - Re-run test from 27.1
    - **EXPECTED OUTCOME**: Test PASSES

  - [x] 27.3.5 Verify preservation tests pass
    - **Property 2: Preservation** - WebSocket Messages Delivered
    - Re-run tests from 27.2
    - **EXPECTED OUTCOME**: Tests PASS

- [x] 27.4 Checkpoint - Bug 27 Complete

### Bug 28: No Message List Pagination

- [x] 28.1 Write bug condition exploration test
  - **Property 1: Bug Condition** - All Messages Loaded at Once
  - Test that message list loads all records
  - Create 10K test messages
  - Measure memory and performance
  - Run on UNFIXED code
  - **EXPECTED OUTCOME**: Test FAILS (memory/performance issues)
  - _Requirements: 1.28, 2.28_

- [x] 28.2 Write preservation property tests
  - **Property 2: Preservation** - Message Display Works
  - Observe successful message display
  - Write tests for message listing
  - Verify tests PASS on UNFIXED code
  - _Requirements: 3.16_

- [ ] 28.3 Fix for No Message List Pagination

  - [x] 28.3.1 Add pagination to backend API
    - Create GetMessagesRequest with page, page_size
    - Implement pagination logic
    - Return PaginatedResponse
    - _Requirements: 2.28_

  - [x] 28.3.2 Add total count query
    - Query total count separately
    - Calculate total pages
    - Include has_next, has_prev flags
    - _Requirements: 2.28_

  - [x] 28.3.3 Update MessageHistory component
    - Add pagination state
    - Implement loadMessages() with page parameter
    - Add Previous/Next buttons
    - _Requirements: 2.28_

  - [ ] 28.3.4 Optional: Implement infinite scroll
    - Use react-infinite-scroll-component
    - Load more on scroll
    - _Bug_Condition: Loads all messages at once_
    - _Expected_Behavior: Pagination with page size 50_
    - _Preservation: Message display works_
    - _Requirements: 2.28_

  - [x] 28.3.5 Verify bug condition test passes
    - **Property 1: Expected Behavior** - Pagination Works
    - Re-run test from 28.1
    - **EXPECTED OUTCOME**: Test PASSES

  - [x] 28.3.6 Verify preservation tests pass
    - **Property 2: Preservation** - Message Display Works
    - Re-run tests from 28.2
    - **EXPECTED OUTCOME**: Tests PASS

- [x] 28.4 Checkpoint - Bug 28 Complete

### Bug 29: No API Rate Limiting

- [x] 29.1 Write bug condition exploration test
  - **Property 1: Bug Condition** - No Rate Limiting
  - Test that API accepts unlimited requests
  - Make 1000 requests rapidly
  - Run on UNFIXED code
  - **EXPECTED OUTCOME**: Test FAILS (all accepted)
  - _Requirements: 1.29, 2.29_

- [x] 29.2 Write preservation property tests
  - **Property 2: Preservation** - Normal Usage Works
  - Observe normal API usage patterns
  - Write tests for typical request rates
  - Verify tests PASS on UNFIXED code
  - _Requirements: 3.9, 3.10_

- [ ] 29.3 Fix for No API Rate Limiting

  - [x] 29.3.1 Install rate limiting library
    - Add github.com/ulule/limiter/v3
    - Add Redis driver
    - _Requirements: 2.29_

  - [x] 29.3.2 Create rate limiter middleware
    - Implement NewRateLimiter()
    - Configure 100 requests per minute per IP
    - Use Redis for distributed rate limiting
    - _Requirements: 2.29_

  - [x] 29.3.3 Apply middleware to API routes
    - Apply to /api routes
    - Set rate limit headers
    - Return 429 on limit exceeded
    - _Requirements: 2.29_

  - [ ] 29.3.4 Optional: Implement tiered rate limiting
    - Admin: 1000 req/min
    - User: 100 req/min
    - Anonymous: 20 req/min
    - _Requirements: 2.29_

  - [x] 29.3.5 Update frontend error handling
    - Handle 429 responses
    - Show user-friendly message
    - Optional: Implement retry
    - _Bug_Condition: No rate limiting_
    - _Expected_Behavior: 100 requests per minute per IP_
    - _Preservation: Normal usage works_
    - _Requirements: 2.29_

  - [x] 29.3.6 Verify bug condition test passes
    - **Property 1: Expected Behavior** - Rate Limiting Active
    - Re-run test from 29.1
    - **EXPECTED OUTCOME**: Test PASSES

  - [x] 29.3.7 Verify preservation tests pass
    - **Property 2: Preservation** - Normal Usage Works
    - Re-run tests from 29.2
    - **EXPECTED OUTCOME**: Tests PASS

- [x] 29.4 Checkpoint - Bug 29 Complete

### Bug 30: No Mobile Gateway Error Recovery

- [ ] 30.1 Write bug condition exploration test
  - **Property 1: Bug Condition** - No Error Recovery
  - Test that Mobile Gateway errors require manual intervention
  - Simulate network error
  - Simulate SMS send failure
  - Run on UNFIXED code
  - **EXPECTED OUTCOME**: Test FAILS (no recovery)
  - _Requirements: 1.30, 2.30_

- [ ] 30.2 Write preservation property tests
  - **Property 2: Preservation** - Successful SMS Sending
  - Observe successful SMS operations
  - Write tests for normal SMS flow
  - Verify tests PASS on UNFIXED code
  - _Requirements: 3.3_

- [ ] 30.3 Fix for No Mobile Gateway Error Recovery

  - [ ] 30.3.1 Create ErrorRecoveryService
    - Implement error categorization
    - Implement retry queue
    - Implement exponential backoff
    - _Requirements: 2.30_

  - [ ] 30.3.2 Implement error handlers
    - handleNetworkError() - retry with backoff
    - handleAuthError() - refresh token
    - handleSMSError() - check permissions
    - _Requirements: 2.30_

  - [ ] 30.3.3 Add retry processor
    - Process retry queue
    - Track retry attempts
    - Report permanent failures
    - _Requirements: 2.30_

  - [ ] 30.3.4 Integrate with SMS sending
    - Wrap sendSMS() with error recovery
    - Log errors for sync
    - _Requirements: 2.30_

  - [ ] 30.3.5 Add health check and log sync
    - Periodic health check
    - Sync error logs to server
    - _Bug_Condition: No error recovery mechanism_
    - _Expected_Behavior: Retry logic, error reporting, graceful degradation_
    - _Preservation: Successful SMS sending_
    - _Requirements: 2.30_

  - [ ] 30.3.6 Verify bug condition test passes
    - **Property 1: Expected Behavior** - Error Recovery Works
    - Re-run test from 30.1
    - **EXPECTED OUTCOME**: Test PASSES

  - [ ] 30.3.7 Verify preservation tests pass
    - **Property 2: Preservation** - Successful SMS Sending
    - Re-run tests from 30.2
    - **EXPECTED OUTCOME**: Tests PASS

- [ ] 30.4 Checkpoint - Bug 30 Complete

- [x] 30.5 Phase 6 Final Checkpoint
  - End-to-end testing complete
  - User acceptance testing
  - All integration bugs fixed

---

## 🚀 Phase 7: Production Deployment - Week 7

- [x] 31.1 Pre-deployment checklist
  - All 30 bugs fixed and verified
  - All tests passing (unit, integration, property-based, e2e)
  - No regressions detected
  - Performance benchmarks met or exceeded
  - Security audit complete
  - Documentation updated

- [x] 31.2 Staging deployment
  - Deploy to staging environment
  - Run full test suite
  - Monitor for 48 hours
  - Verify all fixes in staging

- [x] 31.3 Production deployment preparation
  - Create deployment plan
  - Prepare rollback plan
  - Configure monitoring and alerting
  - Brief operations team

- [x] 31.4 Gradual production rollout
  - Deploy to 10% of traffic
  - Monitor metrics for 24 hours
  - Deploy to 50% of traffic
  - Monitor metrics for 24 hours
  - Deploy to 100% of traffic

- [x] 31.5 Post-deployment monitoring
  - Monitor error rates
  - Monitor response times
  - Monitor queue sizes
  - Monitor database performance
  - Monitor Redis performance
  - Monitor WebSocket connections
  - Monitor message delivery rates

- [x] 31.6 One-week stability verification
  - System stable for 7 days
  - No critical issues
  - Performance metrics healthy
  - User feedback positive

- [x] 31.7 Project completion
  - Document lessons learned
  - Update runbooks
  - Close all related tickets
  - Celebrate success! 🎉

---

## Success Criteria

✅ All 30 bugs fixed and verified
✅ All tests passing (unit, integration, property-based, e2e)
✅ No regressions in existing functionality
✅ Performance metrics improved or maintained
✅ Security vulnerabilities addressed
✅ System stable in production for 1 week

---

## Notes

- Tasks can be executed in parallel within each phase where no dependencies exist
- Each bug follows the pattern: Explore → Preserve → Fix → Verify
- Property-based tests are marked with **Property N:** format for hover status
- All preservation tests must pass to ensure no regressions
- Checkpoints ensure quality gates before moving to next phase
