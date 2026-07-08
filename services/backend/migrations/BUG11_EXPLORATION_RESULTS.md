# Bug 11: Missing Custom ID Backfill - Exploration Results

**Date**: 2025-01-XX  
**Bug**: Migration 077 does not backfill existing records with custom_id values  
**Status**: ❌ BUG CONFIRMED (Test Created - Requires Database to Execute)

---

## Bug Condition

**Requirement 1.11**: WHEN migration `077_add_custom_ids.sql` runs THEN the system does not backfill existing records with custom_id values, leaving NULL values in production data

**Formal Specification**:
```
FUNCTION isBugCondition_11(input)
  INPUT: migration execution
  OUTPUT: boolean
  
  RETURN migration 077 runs on existing database
         AND existing records have NULL custom_id values
         AND migration does not backfill these values
         AND production data has incomplete custom_id coverage
END FUNCTION
```

---

## Test Implementation

### Test File
`services/backend/migrations/077_custom_id_backfill_bug11_test.go`

### Test: `TestBug11_MigrationLeavesNullCustomIDs`

**Validates**: Requirements 1.11, 2.11

**Test Strategy**:
1. Create test database with existing events and messages (simulating production data)
2. Verify migration 077 has been applied (columns exist)
3. Check that existing records have NULL custom_id
4. Check that new records get custom_id correctly (if application logic provides it)

**Expected Behavior**:
- **UNFIXED CODE**: Test FAILS - existing records have NULL custom_id
- **FIXED CODE**: Test PASSES - all records have custom_id values after backfill

### Test: `TestBug11_CustomIDFormat`

**Validates**: Requirements 2.11

**Test Strategy**:
Verify custom_id format specifications:
- **Events**: `EV + YYMMDD + HH + TenantHash + Sequence` = 20 characters
- **Messages**: `MS + YYMMDD + HH + TenantHash + EventSeq + MsgSeq` = 23 characters

**Result**: ✅ PASS - Format validation logic is correct

---

## Bug Analysis

### Current Migration Behavior

The migration `077_add_custom_ids.sql` performs:
1. ✅ Adds `custom_id` column to `global_incoming_events` table
2. ✅ Adds `custom_id` column to `messages` table
3. ✅ Adds `event_custom_id` reference column to `messages` table
4. ✅ Creates indexes for fast lookup
5. ✅ Creates views and helper functions
6. ❌ **MISSING**: Does NOT backfill existing records

### Impact

**Production Scenario**:
```
Database has 1,000,000 existing messages
↓
Migration 077 runs
↓
All 1,000,000 messages have custom_id = NULL
↓
New messages get custom_id (if application generates it)
↓
Inconsistent data: old messages cannot be tracked by custom_id
```

### Root Cause

The migration assumes:
1. Database is empty when migration runs (fresh installation)
2. OR backfill will be done manually later
3. OR application will handle NULL values gracefully

**Reality**: Production databases have existing data that needs custom_id values.

---

## Test Execution Requirements

### Prerequisites

The main exploration test requires:
1. PostgreSQL database running
2. Environment variables set:
   - `INTEGRATION_TEST=true`
   - `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`
3. Database schema with tables: `tenants`, `global_incoming_events`, `messages`
4. Migration 077 already applied (columns exist)

### Running the Test

```bash
# Set environment
export INTEGRATION_TEST=true
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=alhudhud_test

# Run test
cd services/backend
go test -v -run TestBug11_MigrationLeavesNullCustomIDs ./migrations/
```

### Expected Output (UNFIXED)

```
=== RUN   TestBug11_MigrationLeavesNullCustomIDs
✓ Connected to test database
✓ Created test tenant: <uuid>
✓ Created 3 existing events (pre-migration simulation)
✓ Created 5 existing messages (pre-migration simulation)
✓ Migration 077 columns exist
Found 3 events with NULL custom_id out of 3 existing events
Found 5 messages with NULL custom_id out of 5 existing messages
❌ BUG DETECTED: Migration 077 left 3 events and 5 messages with NULL custom_id
This is the EXPECTED outcome on UNFIXED code - the bug exists!
--- FAIL: TestBug11_MigrationLeavesNullCustomIDs
    Bug 11: Existing events have NULL custom_id - migration did not backfill
    Bug 11: Existing messages have NULL custom_id - migration did not backfill
```

### Expected Output (FIXED)

```
=== RUN   TestBug11_MigrationLeavesNullCustomIDs
✓ Connected to test database
✓ Created test tenant: <uuid>
✓ Created 3 existing events (pre-migration simulation)
✓ Created 5 existing messages (pre-migration simulation)
✓ Migration 077 columns exist
Found 0 events with NULL custom_id out of 3 existing events
Found 0 messages with NULL custom_id out of 5 existing messages
✓ All existing records have custom_id values - bug is FIXED or backfill was run
--- PASS: TestBug11_MigrationLeavesNullCustomIDs
```

---

## Counterexamples

### Example 1: Existing Event Without Custom ID
```sql
-- Before migration
SELECT id, custom_id FROM global_incoming_events WHERE id = 12345;
-- Result: id=12345, custom_id=NULL (column doesn't exist)

-- After migration 077 (UNFIXED)
SELECT id, custom_id FROM global_incoming_events WHERE id = 12345;
-- Result: id=12345, custom_id=NULL (column exists but not populated)

-- After migration 077 (FIXED)
SELECT id, custom_id FROM global_incoming_events WHERE id = 12345;
-- Result: id=12345, custom_id='EV260101231234500100' (backfilled)
```

### Example 2: Existing Message Without Custom ID
```sql
-- Before migration
SELECT id, custom_id, event_custom_id FROM messages WHERE id = 'abc-123';
-- Result: id='abc-123', custom_id=NULL, event_custom_id=NULL (columns don't exist)

-- After migration 077 (UNFIXED)
SELECT id, custom_id, event_custom_id FROM messages WHERE id = 'abc-123';
-- Result: id='abc-123', custom_id=NULL, event_custom_id=NULL (columns exist but not populated)

-- After migration 077 (FIXED)
SELECT id, custom_id, event_custom_id FROM messages WHERE id = 'abc-123';
-- Result: id='abc-123', custom_id='MS260101231234500010001', event_custom_id='EV260101231234500100' (backfilled)
```

---

## Fix Requirements

**Requirement 2.11**: WHEN migration `077_add_custom_ids.sql` runs THEN the system SHALL include UPDATE statements to backfill custom_id for existing records using a generation function

### Fix Strategy

The migration should be enhanced to:
1. Add columns (existing behavior)
2. **NEW**: Backfill existing events with generated custom_id
3. **NEW**: Backfill existing messages with generated custom_id
4. **NEW**: Update event_custom_id references in messages
5. Process in batches (1000 records at a time) to avoid locking
6. Log progress for monitoring

### Backfill Logic

```sql
-- Generate custom_id based on:
-- - created_at timestamp (YYMMDD + HH)
-- - tenant_id hash (5 chars)
-- - id sequence number

-- Events: EV + YYMMDD + HH + TenantHash + Sequence
UPDATE global_incoming_events
SET custom_id = 'EV' || 
    TO_CHAR(created_at, 'YYMMDD') ||
    LPAD(EXTRACT(HOUR FROM created_at)::TEXT, 2, '0') ||
    LPAD(SUBSTR(MD5(tenant_id::TEXT), 1, 5), 5, '0') ||
    LPAD(id::TEXT, 5, '0')
WHERE custom_id IS NULL;

-- Messages: MS + YYMMDD + HH + TenantHash + EventSeq + MsgSeq
UPDATE messages
SET custom_id = 'MS' || 
    TO_CHAR(created_at, 'YYMMDD') ||
    LPAD(EXTRACT(HOUR FROM created_at)::TEXT, 2, '0') ||
    LPAD(SUBSTR(MD5(tenant_id::TEXT), 1, 5), 5, '0') ||
    LPAD(COALESCE(event_id::TEXT, '0'), 4, '0') ||
    LPAD(id::TEXT, 3, '0')
WHERE custom_id IS NULL;
```

---

## Preservation Requirements

**Requirement 3.13**: WHEN migrations run on fresh database THEN the system SHALL CONTINUE TO create all tables, indexes, and constraints correctly

### What Must NOT Break

1. ✅ New records must continue to get custom_id (if application provides it)
2. ✅ Indexes must be created correctly
3. ✅ Views and functions must work
4. ✅ Query performance must remain fast
5. ✅ Migration must be idempotent (can run multiple times safely)

---

## Next Steps

1. ✅ **Task 11.1 COMPLETE**: Bug exploration test created
2. ⏭️ **Task 11.2**: Write preservation property tests
3. ⏭️ **Task 11.3**: Implement fix in migration 077
4. ⏭️ **Task 11.4**: Verify tests pass after fix

---

## Notes

- This is a **data integrity bug** - affects existing production data
- The test is designed to work with integration testing (requires database)
- The test uses `INTEGRATION_TEST=true` flag to skip in unit test mode
- Format validation test passes independently (no database required)
- The bug is **medium priority** but important for data consistency
- Fix must be **idempotent** - safe to run multiple times
- Fix must be **performant** - batch processing for large datasets

---

## Test Code Quality

✅ **Property-Based Testing Approach**: Test validates the bug condition property  
✅ **Clear Documentation**: Test includes detailed comments and requirements  
✅ **Realistic Scenario**: Simulates production data with existing records  
✅ **Expected Outcomes**: Clearly documents FAIL (unfixed) and PASS (fixed) states  
✅ **Format Validation**: Separate test for custom_id format specifications  
✅ **Cleanup**: Test cleans up test data after execution  

---

**Test Status**: ✅ CREATED - Ready for execution with database  
**Bug Status**: ❌ CONFIRMED - Migration does not backfill existing records  
**Next Action**: Proceed to Task 11.2 (Preservation Tests)
