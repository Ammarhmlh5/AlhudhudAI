# Implementation Plan - Security Issues Fix

## Overview

This document contains the comprehensive implementation task list for fixing 15 security vulnerabilities and technical issues in the ALHudhudAI project. Tasks are organized by priority (Critical → High → Medium) and ordered by dependencies.

**Total Issues:** 15  
**Critical:** 5 | **High:** 5 | **Medium:** 5

---

## Phase 1: Critical Issues (1-5)

### 1.1 Fix Exposed Credentials in .env Files

**Description:** Replace plaintext credentials with environment variable placeholders and ensure .env files are not committed to Git.

**Affected Files:**
- `apps/admin-web/.env`
- `apps/client-web/.env`
- `apps/mobile_gateway/.env.example`
- `apps/backend/.env`

**Task Type:** config  
**Priority:** critical  
**Prerequisites:** None

**Implementation Steps:**
1. Backup existing .env files
2. Replace `DB_PASSWORD=mySecretPassword123` with `DB_PASSWORD=${DB_PASSWORD}`
3. Replace `REDIS_PASSWORD=redisSecretKey` with `REDIS_PASSWORD=${REDIS_PASSWORD}`
4. Replace `JWT_SECRET=jwtSecretKey123` with `JWT_SECRET=${JWT_SECRET}`
5. Replace `ADMIN_PASSWORD=adminPass456` with `ADMIN_PASSWORD=${ADMIN_PASSWORD}`
6. Create `.env.example` files with placeholder values only
7. Verify `.env` is in `.gitignore`
8. Document environment variable requirements

**Testing Method:**
- Verify no plaintext credentials in committed files
- Check `.env.example` contains only placeholders
- Verify `.gitignore` includes `.env`

**Verification:**
- Run: `grep -r "PASSWORD=" apps/*/.env | grep -v "\${.*}"` should return no results
- Run: `git status` should not show .env files as changed

**Status:** pending

---

### 1.2 Create RBAC Tables (role_permissions, user_roles)

**Description:** Create missing database tables for Role-Based Access Control system.

**Affected Files:**
- Database: `role_permissions` table
- Database: `user_roles` table
- Migration: `050_create_rbac_tables.sql`

**Task Type:** database  
**Priority:** critical  
**Prerequisites:** None

**Implementation Steps:**
1. Create migration file `050_create_rbac_tables.sql`
2. Create `role_permissions` table with UUID primary key
3. Create `user_roles` table with UUID primary key
4. Add foreign key references to `roles` and `permissions` tables
5. Add unique constraints for (role_id, permission_id) and (user_id, role_id, tenant_id)
6. Create indexes on foreign key columns
7. Run migration against database

**Testing Method:**
- Query `information_schema.tables` for new tables
- Verify foreign key constraints exist
- Verify indexes are created

**Verification:**
- Run: `SELECT table_name FROM information_schema WHERE table_schema = 'public' AND table_name IN ('role_permissions', 'user_roles');`
- Run: `SELECT conname FROM pg_constraint WHERE conname LIKE '%fk%';`

**Status:** pending

---

### 1.3 Add NOT NULL Constraints to Critical Columns

**Description:** Add NOT NULL constraints to columns that should never be NULL.

**Affected Tables:**
- `messages` (channel_id, device_id)
- `dispatch_channels` (name, type)
- `accounting_transactions` (type, status)

**Task Type:** database  
**Priority:** critical  
**Prerequisites:** None

**Implementation Steps:**
1. Create migration file `051_add_not_null_constraints.sql`
2. Add NOT NULL to `messages.channel_id`
3. Add NOT NULL to `messages.device_id`
4. Add NOT NULL to `dispatch_channels.name`
5. Add NOT NULL to `dispatch_channels.type`
6. Add NOT NULL to `accounting_transactions.type`
7. Add NOT NULL to `accounting_transactions.status`
8. Run migration with `ALTER TABLE` statements

**Testing Method:**
- Query `information_schema.columns` for `is_nullable = 'NO'`
- Attempt to insert NULL values (should fail)

**Verification:**
- Run: `SELECT column_name, is_nullable FROM information_schema.columns WHERE table_name IN ('messages', 'dispatch_channels', 'accounting_transactions') AND column_name IN ('channel_id', 'device_id', 'name', 'type', 'status');`

**Status:** pending

---

### 1.4 Unify Financial Precision to DECIMAL(12,4)

**Description:** Standardize all financial columns to use DECIMAL(12,4) for consistent calculations.

**Affected Tables:**
- `messages` (cost)
- All other financial columns in the database

**Task Type:** database  
**Priority:** critical  
**Prerequisites:** None

**Implementation Steps:**
1. Create migration file `052_unify_financial_precision.sql`
2. Identify all financial columns with different precision
3. Alter `messages.cost` from DECIMAL(12,2) to DECIMAL(12,4)
4. Check and alter other financial columns as needed
5. Verify data integrity after migration

**Testing Method:**
- Query `information_schema.columns` for numeric precision
- Verify calculations produce consistent results

**Verification:**
- Run: `SELECT column_name, data_type, numeric_precision, numeric_scale FROM information_schema.columns WHERE table_name IN ('messages', 'accounting_transactions', 'balance_history') AND data_type = 'numeric';`

**Status:** pending

---

### 1.5 Enable SSL/TLS for Database Connections

**Description:** Change DB_SSLMODE from 'disable' to 'require' or 'verify-full' for encrypted connections.

**Affected Files:**
- `apps/admin-web/.env`
- `apps/client-web/.env`
- `apps/backend/.env`

**Task Type:** config  
**Priority:** critical  
**Prerequisites:** None

**Implementation Steps:**
1. Change `DB_SSLMODE=disable` to `DB_SSLMODE=require` in all .env files
2. For production, use `DB_SSLMODE=verify-full` with CA certificate path
3. Add `DB_SSLCA` variable for CA certificate path if needed
4. Test database connection with SSL enabled

**Testing Method:**
- Verify database connection works with SSL
- Check PostgreSQL connection uses SSL

**Verification:**
- Run: `SELECT ssl FROM pg_stat_ssl WHERE pid = pg_backend_pid();` should return `true`
- Check application logs for successful SSL connections

**Status:** pending

---

## Phase 2: High Issues (6-10)

### 2.1 Enable Row Level Security (RLS) on 6 Tables

**Description:** Enable PostgreSQL Row Level Security on multi-tenant tables for data isolation.

**Affected Tables:**
- `messages`
- `balance_history`
- `devices`
- `channels`
- `users`
- `audit_trail`

**Task Type:** database  
**Priority:** high  
**Prerequisites:** 1.2 (RBAC tables created)

**Implementation Steps:**
1. Create migration file `053_enable_rls.sql`
2. Enable RLS on each table using `ALTER TABLE ... ENABLE ROW LEVEL SECURITY`
3. Create tenant isolation policies for each table
4. Test RLS policies prevent cross-tenant access

**Testing Method:**
- Query `pg_class` for `relrowsecurity = true`
- Attempt cross-tenant data access (should be denied)

**Verification:**
- Run: `SELECT relname, relrowsecurity FROM pg_class WHERE relname IN ('messages', 'balance_history', 'devices', 'channels', 'users', 'audit_trail');`

**Status:** pending

---

### 2.2 Create Indexes on tenant_id Columns

**Description:** Create composite indexes on tenant_id for improved query performance.

**Affected Tables:**
- `messages` (tenant_id, status, created_at)
- `balance_history` (tenant_id, type, created_at)
- `accounting_transactions` (tenant_id, status)
- `subscriptions` (tenant_id, expires_at)

**Task Type:** database  
**Priority:** high  
**Prerequisites:** None

**Implementation Steps:**
1. Create migration file `054_create_tenant_indexes.sql`
2. Use `CREATE INDEX CONCURRENTLY` to avoid table locks
3. Create composite index on `messages(tenant_id, status, created_at)`
4. Create composite index on `balance_history(tenant_id, type, created_at)`
5. Create index on `accounting_transactions(tenant_id, status)`
6. Create index on `subscriptions(tenant_id, expires_at)`

**Testing Method:**
- Query `pg_indexes` for new indexes
- Verify query performance improvement with EXPLAIN ANALYZE

**Verification:**
- Run: `SELECT indexname, indexdef FROM pg_indexes WHERE tablename IN ('messages', 'balance_history', 'accounting_transactions', 'subscriptions');`

**Status:** pending

---

### 2.3 Add CHECK Constraints for Financial Columns

**Description:** Add database-level constraints to ensure financial data integrity.

**Affected Tables:**
- `messages` (cost >= 0)
- `balance_history` (amount != 0)
- `accounting_transactions` (amount > 0)

**Task Type:** database  
**Priority:** high  
**Prerequisites:** 1.3 (NOT NULL constraints)

**Implementation Steps:**
1. Create migration file `055_add_check_constraints.sql`
2. Add `CHECK (cost >= 0)` constraint to `messages`
3. Add `CHECK (amount != 0)` constraint to `balance_history`
4. Add `CHECK (amount > 0)` constraint to `accounting_transactions`
5. Verify existing data satisfies constraints

**Testing Method:**
- Attempt to insert invalid financial values (should fail)
- Query `pg_constraint` for check constraints

**Verification:**
- Run: `SELECT conname, pg_get_constraintdef(oid) FROM pg_constraint WHERE conname LIKE '%_check';`

**Status:** pending

---

### 2.4 Add Logger to Circuit Breaker

**Description:** Add comprehensive logging to Circuit Breaker for monitoring and debugging.

**Affected Files:**
- `apps/backend/internal/resilience/circuit_breaker.go`

**Task Type:** backend  
**Priority:** high  
**Prerequisites:** None

**Implementation Steps:**
1. Import `log/slog` package
2. Add `logger *slog.Logger` field to CircuitBreaker struct
3. Update `NewCircuitBreaker` to accept logger parameter
4. Add logging in `RecordFailure()` method
5. Add logging in `RecordSuccess()` method
6. Add `onStateChange()` method with logging
7. Update all CircuitBreaker instantiations to pass logger

**Testing Method:**
- Run Circuit Breaker and trigger failures
- Verify logs appear in application logs
- Check state change logs

**Verification:**
- Run application and trigger circuit breaker events
- Verify logs contain: name, failure_count, state changes, errors

**Status:** pending

---

### 2.5 Create Shared Library for Duplicate Code

**Description:** Extract duplicate code from admin-web and client-web into a shared library.

**Affected Files:**
- `apps/admin-web/src/api/axios.js`
- `apps/client-web/src/api/axios.js`
- New: `libs/shared/src/api/axios.ts`
- New: `libs/shared/src/auth/useAuth.ts`
- New: `libs/shared/src/validation/schema.ts`

**Task Type:** frontend  
**Priority:** high  
**Prerequisites:** None

**Implementation Steps:**
1. Create `libs/shared/` directory structure
2. Create unified `axios.ts` with API configuration
3. Create unified `useAuth.ts` authentication hook
4. Create unified validation schemas
5. Update `admin-web` to import from shared library
6. Update `client-web` to import from shared library
7. Configure build system to export shared library

**Testing Method:**
- Verify both applications build successfully
- Test API calls work from both applications
- Test authentication flows work

**Verification:**
- Run: `npm run build` in both admin-web and client-web
- Verify no duplicate code warnings in build output

**Status:** pending

---

## Phase 3: Medium Issues (11-15)

### 3.1 Fix Bypassable Rate Limiting

**Description:** Improve Rate Limiting to use user ID + IP combination instead of IP only.

**Affected Files:**
- `apps/backend/internal/middleware/rate_limit.go`

**Task Type:** backend  
**Priority:** medium  
**Prerequisites:** None

**Implementation Steps:**
1. Modify RateLimiter struct to track user ID
2. Update `Allow()` method to accept user ID parameter
3. Implement `generateKey()` function for user:ID and ip:IP keys
4. Update rate limiting logic to prioritize user ID over IP
5. Return HTTP 429 when rate limit exceeded
6. Update all middleware usages to pass user ID

**Testing Method:**
- Test rate limiting with authenticated user
- Test rate limiting with multiple IPs for same user
- Verify 429 status code is returned

**Verification:**
- Run load tests with multiple IPs
- Verify user-based rate limiting works correctly

**Status:** pending

---

### 3.2 Improve CSRF Token Management

**Description:** Add automatic token refresh and better error handling for CSRF tokens.

**Affected Files:**
- `apps/admin-web/src/api/axios.js`
- `apps/client-web/src/api/axios.js`

**Task Type:** frontend  
**Priority:** medium  
**Prerequisites:** None

**Implementation Steps:**
1. Create `CsrfManager` class with token storage
2. Implement `initialize()` method to fetch initial token
3. Implement `refreshToken()` method for manual refresh
4. Implement `startAutoRefresh()` for automatic refresh (25 min interval)
5. Add error handling for token refresh failures
6. Integrate with axios interceptors
7. Update API calls to use CSRF manager

**Testing Method:**
- Test token refresh on expiration
- Test error handling for failed refresh
- Verify all API calls include valid CSRF token

**Verification:**
- Check network requests for CSRF token header
- Verify token refresh happens before expiration

**Status:** pending

---

### 3.3 Create Triggers for updated_at Auto-Update

**Description:** Create database triggers to automatically update `updated_at` columns on row modifications.

**Affected Tables:**
- `tenants`
- `users`
- `channels`
- `devices`
- `messages`

**Task Type:** database  
**Priority:** medium  
**Prerequisites:** None

**Implementation Steps:**
1. Create migration file `056_create_updated_at_triggers.sql`
2. Create `update_updated_at_column()` function
3. Create trigger for `tenants` table
4. Create trigger for `users` table
5. Create trigger for `channels` table
6. Create trigger for `devices` table
7. Create trigger for `messages` table

**Testing Method:**
- Update rows in affected tables
- Verify `updated_at` is automatically set
- Check trigger execution in logs

**Verification:**
- Run: `SELECT tgname FROM pg_trigger WHERE tgname LIKE '%updated_at%';`

**Status:** pending

---

### 3.4 Unify UUID Generation

**Description:** Standardize ID generation to use UUID instead of SERIAL or VARCHAR.

**Affected Tables:**
- `tenants` (id: SERIAL → UUID)
- `messages` (message_id: VARCHAR → UUID)

**Task Type:** database  
**Priority:** medium  
**Prerequisites:** None

**Implementation Steps:**
1. Create migration file `057_unify_uuid_generation.sql`
2. Add new UUID column to `tenants`
3. Copy existing data to new column with generated UUIDs
4. Drop old ID column and rename new column
5. Add new UUID column to `messages` for message_id
6. Copy existing data and update foreign keys
7. Add backward-compatible helper functions if needed

**Testing Method:**
- Verify new records use UUID
- Verify existing data is preserved
- Test foreign key relationships

**Verification:**
- Run: `SELECT id, typeof(id) FROM tenants LIMIT 1;`
- Run: `SELECT message_id, typeof(message_id) FROM messages LIMIT 1;`

**Status:** pending

---

### 3.5 Improve Documentation

**Description:** Create comprehensive security documentation and improve existing docs.

**Affected Files:**
- `SECURITY.md` (new)
- `docs/environment-variables.md` (new)
- `docs/database-schema.md` (new)
- `docs/migrations.md` (new)

**Task Type:** docs  
**Priority:** medium  
**Prerequisites:** None

**Implementation Steps:**
1. Create `SECURITY.md` with:
   - Security policy
   - Reporting vulnerabilities process
   - Security best practices
2. Create `docs/environment-variables.md` with:
   - All environment variables
   - Descriptions and required formats
   - Security considerations
3. Create `docs/database-schema.md` with:
   - Table relationships
   - Constraint descriptions
   - RLS policies
4. Create `docs/migrations.md` with:
   - Migration history
   - Rollback procedures
   - Data preservation notes

**Testing Method:**
- Review documentation for completeness
- Verify all environment variables are documented
- Check migration documentation accuracy

**Verification:**
- Run: `ls -la docs/` should show new documentation files
- Review documentation for clarity and completeness

**Status:** pending

---

## Task Summary

| Phase | Task | Title | Type | Priority | Status |
|-------|------|-------|------|----------|--------|
| 1 | 1.1 | Fix Exposed Credentials | config | critical | ✅ completed |
| 1 | 1.2 | Create RBAC Tables | database | critical | ✅ completed |
| 1 | 1.3 | Add NOT NULL Constraints | database | critical | ✅ completed |
| 1 | 1.4 | Unify Financial Precision | database | critical | ✅ completed |
| 1 | 1.5 | Enable SSL/TLS | config | critical | ✅ completed |
| 2 | 2.1 | Enable RLS on Tables | database | high | ✅ completed |
| 2 | 2.2 | Create tenant_id Indexes | database | high | ✅ completed |
| 2 | 2.3 | Add CHECK Constraints | database | high | ✅ completed |
| 2 | 2.4 | Add Logger to Circuit Breaker | backend | high | ✅ completed |
| 2 | 2.5 | Create Shared Library | frontend | high | ✅ completed |
| 3 | 3.1 | Fix Rate Limiting | backend | medium | ✅ completed |
| 3 | 3.2 | Improve CSRF Management | frontend | medium | ⏸ pending |
| 3 | 3.3 | Create updated_at Triggers | database | medium | ✅ completed |
| 3 | 3.4 | Unify UUID Generation | database | medium | ✅ completed |
| 3 | 3.5 | Improve Documentation | docs | medium | ✅ completed |

---

## Execution Order

### Phase 1 (Critical) - Execute in Order
1. 1.1 Fix Exposed Credentials
2. 1.2 Create RBAC Tables
3. 1.3 Add NOT NULL Constraints
4. 1.4 Unify Financial Precision
5. 1.5 Enable SSL/TLS

### Phase 2 (High) - Execute in Order
6. 2.1 Enable RLS on Tables (depends on 1.2)
7. 2.2 Create tenant_id Indexes
8. 2.3 Add CHECK Constraints (depends on 1.3)
9. 2.4 Add Logger to Circuit Breaker
10. 2.5 Create Shared Library

### Phase 3 (Medium) - Execute in Order
11. 3.1 Fix Rate Limiting
12. 3.2 Improve CSRF Management
13. 3.3 Create updated_at Triggers
14. 3.4 Unify UUID Generation
15. 3.5 Improve Documentation

---

## Important Notes

- **Always backup data** before running database migrations
- **Use `CREATE INDEX CONCURRENTLY`** to avoid production locks
- **Test migrations on staging** before production deployment
- **Run existing tests** after each fix to ensure no regressions
- **Document any issues** encountered during implementation
- **Verify fix** before moving to the next task

---

**Created:** May 2026  
**Last Updated:** May 2026