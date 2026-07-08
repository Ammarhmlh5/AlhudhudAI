# ALHudhudAI Security Issues Fix Design

## Overview

This design document outlines the comprehensive fix for 15 security vulnerabilities and technical issues identified in the ALHudhudAI project. The issues span critical (5), high (5), and medium (5) severity levels, affecting database schema, configuration files, backend resilience, and frontend code.

**Scope:** Fix all security vulnerabilities and technical debt in database, backend, and frontend applications.

**Approach:** Three-phase implementation - Critical issues first, then high, then medium severity.

**Key Principles:**
- Use ALTER TABLE instead of DROP/CREATE to preserve data
- Use CREATE INDEX CONCURRENTLY to avoid table locks
- Create new migrations (050, 051, 052, etc.)
- Maintain backward compatibility with existing code

## Glossary

- **Bug_Condition (C)**: Any input or configuration that exposes a security vulnerability or data integrity issue
- **Property (P)**: The desired secure and correct behavior after fixes are applied
- **Preservation**: Existing functionality that must remain unchanged after fixes
- **RLS**: Row Level Security - PostgreSQL feature for multi-tenant data isolation
- **CHECK Constraint**: Database constraint to validate data integrity rules
- **Circuit Breaker**: Resilience pattern for handling service failures gracefully
- **CSRF Token**: Cross-Site Request Forgery protection token

## Bug Details

### Bug Condition

The bug manifests when any of the following conditions exist in the system:

**Formal Specification:**
```
FUNCTION isBugCondition(input)
  INPUT: input of type SystemConfiguration | DatabaseSchema | ApplicationCode
  OUTPUT: boolean
  
  RETURN (
    -- Critical Issues
    (input CONTAINS exposedCredentials) OR
    (input.rbacTablesMissing = true) OR
    (input.nullableCriticalColumns EXISTS) OR
    (input.financialPrecision INCONSISTENT) OR
    (input.sslModeDisabled = true) OR
    
    -- High Issues
    (input.rlsDisabledTables EXISTS) OR
    (input.missingTenantIndexes EXISTS) OR
    (input.missingCheckConstraints EXISTS) OR
    (input.circuitBreakerWithoutLogger = true) OR
    (input.duplicateCode EXISTS) OR
    
    -- Medium Issues
    (input.rateLimitingBypassable = true) OR
    (input.csrfManagementSimple = true) OR
    (input.missingUpdateTriggers EXISTS) OR
    (input.uuidInconsistency EXISTS) OR
    (input.documentationUnclear = true)
  )
END FUNCTION
```

### Examples

**Example 1: Exposed Credentials**
- File: `apps/admin-web/.env`
- Current: `DB_PASSWORD=mySecretPassword123`
- Expected: `DB_PASSWORD=${DB_PASSWORD}` or encrypted value

**Example 2: Missing RBAC Tables**
- Current: No `role_permissions` or `user_roles` tables
- Expected: Both tables exist with proper foreign keys

**Example 3: Nullable Critical Columns**
- Table: `messages`
- Current: `channel_id` and `device_id` allow NULL
- Expected: Both columns have NOT NULL constraint

**Example 4: Inconsistent Financial Precision**
- Table: `messages.cost` uses DECIMAL(12,2)
- Table: `accounting_transactions.amount` uses DECIMAL(12,4)
- Expected: All financial columns use DECIMAL(12,4)

**Example 5: Disabled SSL Mode**
- Current: `DB_SSLMODE=disable`
- Expected: `DB_SSLMODE=require` or `DB_SSLMODE=verify-full`

## Expected Behavior

### Preservation Requirements

**Unchanged Behaviors:**
- Existing data must remain intact during all migrations
- All existing APIs must continue to return the same response formats
- Authentication and authorization flows must continue to work
- Frontend applications must function without code changes
- All existing tests must pass after fixes are applied

**Scope:**
All inputs and operations that do NOT involve the identified bug conditions should be completely unaffected by this fix. This includes:
- Normal user operations (login, view data, submit forms)
- Background jobs and scheduled tasks
- Inter-service communication (where not directly affected)
- Existing database queries and joins

## Hypothesized Root Cause

Based on the bug analysis, the root causes are:

1. **Development Environment Configurations**: Environment files were configured for local development without security considerations, and these configurations were accidentally committed to the repository.

2. **Incomplete Initial Design**: The RBAC system was partially implemented but the supporting tables were never created. The schema focused on core functionality without security hardening.

3. **Schema Evolution Without Governance**: As the project evolved, different developers added columns with varying constraints, leading to inconsistencies in data types and constraints.

4. **Performance Over Security Tradeoff**: SSL was disabled for local development performance testing and never re-enabled for production configurations.

5. **Missing Security Reviews**: No comprehensive security review was conducted before deployment, allowing these vulnerabilities to reach production.

6. **Code Duplication for Parallel Development**: Admin-web and client-web were developed in parallel with duplicated code instead of shared libraries.

7. **Insufficient Database Expertise**: The team may lack deep PostgreSQL expertise, leading to missing RLS, indexes, and constraints.

## Correctness Properties

Property 1: Bug Condition - All Security Vulnerabilities Fixed

_For any_ system configuration or database schema where a security vulnerability exists (isBugCondition returns true), the fixed system SHALL have all vulnerabilities remediated according to the severity-based priority.

**Validates: Requirements 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 2.8, 2.9, 2.10, 2.11, 2.12, 2.13, 2.14, 2.15**

Property 2: Preservation - Existing Functionality Unchanged

_For any_ operation or input that does NOT involve the identified bug conditions (isBugCondition returns false), the fixed system SHALL produce exactly the same behavior as the original system, preserving all existing functionality.

**Validates: Requirements 3.1, 3.2, 3.3, 3.4, 3.5**

## Fix Implementation

### Phase 1: Critical Issues (1-5)

#### Issue 1: Exposed Credentials in .env Files

**Files to Modify:**
- `apps/admin-web/.env`
- `apps/client-web/.env`
- `apps/mobile_gateway/.env.example`
- `apps/backend/.env`

**Changes Required:**

1. Replace actual credentials with placeholders:
```env
# Before
DB_PASSWORD=mySecretPassword123
REDIS_PASSWORD=redisSecretKey
JWT_SECRET=jwtSecretKey123
ADMIN_PASSWORD=adminPass456

# After
DB_PASSWORD=${DB_PASSWORD}
REDIS_PASSWORD=${REDIS_PASSWORD}
JWT_SECRET=${JWT_SECRET}
ADMIN_PASSWORD=${ADMIN_PASSWORD}
```

2. Create `.env.example` files with placeholder values only

3. Add `.env` to `.gitignore` if not already present

4. Document the need for environment-specific configuration

#### Issue 2: Incomplete RBAC System

**Tables to Create:**

```sql
-- Migration: 050_create_rbac_tables.sql

-- Create role_permissions table
CREATE TABLE IF NOT EXISTS role_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(role_id, permission_id)
);

-- Create user_roles table
CREATE TABLE IF NOT EXISTS user_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    UNIQUE(user_id, role_id, tenant_id)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_role_permissions_role_id 
ON role_permissions(role_id);
CREATE INDEX IF NOT EXISTS idx_role_permissions_permission_id 
ON role_permissions(permission_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_user_id 
ON user_roles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_tenant_id 
ON user_roles(tenant_id);
```

#### Issue 3: Missing NOT NULL Constraints

**Tables to Modify:**

```sql
-- Migration: 051_add_not_null_constraints.sql

-- Add NOT NULL to messages table
ALTER TABLE messages 
ALTER COLUMN channel_id SET NOT NULL;

ALTER TABLE messages 
ALTER COLUMN device_id SET NOT NULL;

-- Add NOT NULL to dispatch_channels table
ALTER TABLE dispatch_channels 
ALTER COLUMN name SET NOT NULL;

ALTER TABLE dispatch_channels 
ALTER COLUMN type SET NOT NULL;

-- Add NOT NULL to accounting_transactions table
ALTER TABLE accounting_transactions 
ALTER COLUMN type SET NOT NULL;

ALTER TABLE accounting_transactions 
ALTER COLUMN status SET NOT NULL;
```

#### Issue 4: Inconsistent Financial Precision

**Columns to Modify:**

```sql
-- Migration: 052_unify_financial_precision.sql

-- Change DECIMAL(12,2) to DECIMAL(12,4)
ALTER TABLE messages 
ALTER COLUMN cost TYPE DECIMAL(12,4);

-- Verify all other financial columns use DECIMAL(12,4)
-- If any use different precision, alter them as well
```

#### Issue 5: Disabled SSL/TLS for Database

**Files to Modify:**
- `apps/admin-web/.env`
- `apps/client-web/.env`
- `apps/backend/.env`

**Changes Required:**
```env
# Before
DB_SSLMODE=disable

# After
DB_SSLMODE=require
# Or for production:
# DB_SSLMODE=verify-full
# DB_SSLCA=/path/to/ca.crt
```

### Phase 2: High Issues (6-10)

#### Issue 6: RLS Not Enabled on 6 Tables

**Tables to Modify:**

```sql
-- Migration: 053_enable_rls.sql

-- Enable RLS on each table
ALTER TABLE messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE balance_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE devices ENABLE ROW LEVEL SECURITY;
ALTER TABLE channels ENABLE ROW LEVEL SECURITY;
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_trail ENABLE ROW LEVEL SECURITY;

-- Create policies for tenant isolation
CREATE POLICY IF NOT EXISTS messages_tenant_policy ON messages
    FOR ALL
    USING (tenant_id IN (
        SELECT tenant_id FROM users WHERE id = current_setting('app.current_user_id', true)::UUID
    ));

-- Similar policies for other tables...
```

#### Issue 7: Missing Indexes on tenant_id

**Indexes to Create:**

```sql
-- Migration: 054_create_tenant_indexes.sql

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_messages_tenant_status_created
ON messages(tenant_id, status, created_at);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_balance_history_tenant_type_created
ON balance_history(tenant_id, type, created_at);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_accounting_transactions_tenant_status
ON accounting_transactions(tenant_id, status);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_subscriptions_tenant_expires
ON subscriptions(tenant_id, expires_at);
```

#### Issue 8: Missing CHECK Constraints

**Constraints to Add:**

```sql
-- Migration: 055_add_check_constraints.sql

-- messages.cost >= 0
ALTER TABLE messages 
ADD CONSTRAINT messages_cost_non_negative 
CHECK (cost >= 0);

-- balance_history.amount != 0
ALTER TABLE balance_history 
ADD CONSTRAINT balance_history_amount_nonzero 
CHECK (amount != 0);

-- accounting_transactions.amount > 0
ALTER TABLE accounting_transactions 
ADD CONSTRAINT accounting_transactions_amount_positive 
CHECK (amount > 0);
```

#### Issue 9: Circuit Breaker Without Logger

**File to Modify:** `apps/backend/internal/resilience/circuit_breaker.go`

**Changes Required:**
```go
package resilience

import (
    "context"
    "log/slog"
    "sync"
    "time"
)

type CircuitBreaker struct {
    name             string
    logger           *slog.Logger
    state            State
    failureCount     int
    successCount     int
    lastFailure      time.Time
    mu               sync.RWMutex
    // ... other fields
}

func NewCircuitBreaker(name string, logger *slog.Logger) *CircuitBreaker {
    return &CircuitBreaker{
        name:   name,
        logger: logger,
        state:  StateClosed,
    }
}

func (cb *CircuitBreaker) RecordFailure(err error) {
    cb.mu.Lock()
    defer cb.mu.Unlock()
    
    cb.failureCount++
    cb.lastFailure = time.Now()
    
    cb.logger.Error("circuit breaker recorded failure",
        slog.String("name", cb.name),
        slog.Int("failure_count", cb.failureCount),
        slog.String("error", err.Error()),
    )
}

func (cb *CircuitBreaker) RecordSuccess() {
    cb.mu.Lock()
    defer cb.mu.Unlock()
    
    cb.successCount++
    cb.failureCount = 0
    
    cb.logger.Debug("circuit breaker recorded success",
        slog.String("name", cb.name),
        slog.Int("success_count", cb.successCount),
    )
}

func (cb *CircuitBreaker) onStateChange(oldState, newState State) {
    cb.logger.Info("circuit breaker state changed",
        slog.String("name", cb.name),
        slog.String("old_state", string(oldState)),
        slog.String("new_state", string(newState)),
        slog.Int("failure_count", cb.failureCount),
    )
}
```

#### Issue 10: Duplicate Code Between admin-web and client-web

**Files to Create/Modify:**

1. Create shared library: `libs/shared/src/`
   - `api/axios.ts` - Unified axios configuration
   - `auth/useAuth.ts` - Unified authentication hook
   - `validation/schema.ts` - Unified validation schemas

2. Modify `apps/admin-web/src/api/axios.js`:
```javascript
// Before: Custom axios configuration
import axios from 'axios';

const api = axios.create({
  baseURL: process.env.VITE_API_URL,
  // ... duplicate configuration
});

// After: Import from shared library
import { api } from '@alhudhud/shared/api';
```

3. Modify `apps/client-web/src/api/axios.js`:
```javascript
// Before: Custom axios configuration
import axios from 'axios';

const api = axios.create({
  baseURL: process.env.VITE_API_URL,
  // ... duplicate configuration
});

// After: Import from shared library
import { api } from '@alhudhud/shared/api';
```

### Phase 3: Medium Issues (11-15)

#### Issue 11: Bypassable Rate Limiting

**File to Modify:** `apps/backend/internal/middleware/rate_limit.go`

**Changes Required:**
```go
package middleware

import (
    "context"
    "net/http"
    "sync"
    "time"
)

type RateLimiter struct {
    // Use user ID + IP combination for better tracking
    limits map[string]*UserRateLimit
    mu     sync.RWMutex
}

type UserRateLimit struct {
    userID    string
    ip        string
    requests  int
    windowStart time.Time
}

func (rl *RateLimiter) Allow(ctx context.Context, userID, ip string) bool {
    key := generateKey(userID, ip)
    
    rl.mu.Lock()
    defer rl.mu.Unlock()
    
    limit := rl.limits[key]
    now := time.Now()
    
    if limit == nil || now.Sub(limit.windowStart) > windowSize {
        rl.limits[key] = &UserRateLimit{
            userID:      userID,
            ip:          ip,
            requests:    1,
            windowStart: now,
        }
        return true
    }
    
    limit.requests++
    return limit.requests <= maxRequests
}

func generateKey(userID, ip string) string {
    if userID != "" {
        return "user:" + userID
    }
    return "ip:" + ip
}
```

#### Issue 12: Simple CSRF Token Management

**Files to Modify:**
- `apps/admin-web/src/api/axios.js`
- `apps/client-web/src/api/axios.js`

**Changes Required:**
```javascript
import axios from 'axios';

class CsrfManager {
    constructor() {
        this.token = null;
        this.refreshTimer = null;
    }

    async initialize() {
        await this.refreshToken();
        this.startAutoRefresh();
    }

    async refreshToken() {
        try {
            const response = await axios.get('/api/csrf-token');
            this.token = response.data.token;
            return this.token;
        } catch (error) {
            console.error('Failed to refresh CSRF token:', error);
            throw error;
        }
    }

    startAutoRefresh() {
        // Refresh token 5 minutes before expiration
        this.refreshTimer = setInterval(() => {
            this.refreshToken();
        }, 25 * 60 * 1000); // 25 minutes
    }

    getToken() {
        return this.token;
    }
}

export const csrfManager = new CsrfManager();
```

#### Issue 13: Missing Triggers for updated_at

```sql
-- Migration: 056_create_updated_at_triggers.sql

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply to each table
CREATE TRIGGER update_tenants_updated_at
    BEFORE UPDATE ON tenants
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_channels_updated_at
    BEFORE UPDATE ON channels
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_devices_updated_at
    BEFORE UPDATE ON devices
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_messages_updated_at
    BEFORE UPDATE ON messages
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

#### Issue 14: UUID Generation Inconsistency

**Tables to Modify:**

```sql
-- Migration: 057_unify_uuid_generation.sql

-- Change tenants.id from SERIAL to UUID
ALTER TABLE tenants 
ADD COLUMN new_id UUID DEFAULT gen_random_uuid();

UPDATE tenants SET new_id = gen_random_uuid() WHERE new_id IS NULL;

BEGIN;
ALTER TABLE tenants DROP COLUMN id;
ALTER TABLE tenants RENAME COLUMN new_id TO id;
ALTER TABLE tenants ADD PRIMARY KEY (id);
COMMIT;

-- Change messages.message_id from VARCHAR to UUID
ALTER TABLE messages 
ADD COLUMN new_message_id UUID DEFAULT gen_random_uuid();

UPDATE messages SET new_message_id = gen_random_uuid() WHERE new_message_id IS NULL;

BEGIN;
ALTER TABLE messages DROP COLUMN message_id;
ALTER TABLE messages RENAME COLUMN new_message_id TO message_id;
ALTER TABLE messages ADD PRIMARY KEY (message_id);
COMMIT;
```

#### Issue 15: Unclear Documentation

**Actions Required:**
1. Create comprehensive SECURITY.md document
2. Document all environment variables with descriptions
3. Document database schema and relationships
4. Document API endpoints and authentication requirements
5. Create migration documentation for each fix

## Testing Strategy

### Validation Approach

The testing strategy follows a three-phase approach: first, verify the current state of vulnerabilities, then implement fixes, and finally verify that all functionality is preserved.

### Exploratory Bug Condition Checking

**Goal:** Verify that all identified vulnerabilities exist before implementing fixes.

**Test Plan:** Run database queries and code analysis to confirm each issue.

**Test Cases:**
1. **Credentials Exposure Test**: Check .env files for plaintext credentials (will find issues on unfixed code)
2. **RBAC Tables Test**: Query information_schema for missing tables (will fail on unfixed code)
3. **NOT NULL Constraints Test**: Check nullable columns (will find issues on unfixed code)
4. **Financial Precision Test**: Query column types (will find inconsistencies on unfixed code)
5. **SSL Mode Test**: Check DB_SSLMODE configuration (will find disabled SSL on unfixed code)

**Expected Counterexamples:**
- Plaintext credentials in .env files
- Missing role_permissions and user_roles tables
- Nullable critical columns
- Inconsistent DECIMAL precisions
- Disabled SSL mode

### Fix Checking

**Goal:** Verify that for all identified bug conditions, the fixed system produces the expected secure behavior.

**Pseudocode:**
```
FOR ALL issue IN identifiedIssues DO
    result := applyFix(issue)
    ASSERT result.success = true
    ASSERT verifyFix(issue) = true
END FOR
```

### Preservation Checking

**Goal:** Verify that for all operations not involving bug conditions, the fixed system produces the same results as the original.

**Pseudocode:**
```
FOR ALL operation IN existingOperations DO
    originalResult := originalSystem.execute(operation)
    fixedResult := fixedSystem.execute(operation)
    ASSERT originalResult = fixedResult
END FOR
```

**Testing Approach:** Property-based testing is recommended for preservation checking because:
- It generates many test cases automatically across the input domain
- It catches edge cases that manual unit tests might miss
- It provides strong guarantees that behavior is unchanged

**Test Plan:**
1. Run all existing unit tests
2. Run all existing integration tests
3. Run property-based tests for database operations
4. Verify API response formats match

### Unit Tests

- Test each migration script independently
- Test Circuit Breaker logging functionality
- Test CSRF token refresh mechanism
- Test Rate Limiter with user ID tracking
- Test UUID generation consistency

### Property-Based Tests

- Generate random tenant configurations and verify RLS policies
- Generate random financial transactions and verify CHECK constraints
- Generate random user operations and verify preservation of behavior
- Generate random API requests and verify rate limiting

### Integration Tests

- Test full authentication flow with new RBAC system
- Test database operations with all constraints enabled
- Test Circuit Breaker integration with Logger
- Test end-to-end API calls with CSRF protection
- Test multi-tenant data isolation with RLS

## Verification Steps

### Database Verification

```sql
-- Verify NOT NULL constraints
SELECT column_name, is_nullable 
FROM information_schema.columns 
WHERE table_name IN ('messages', 'dispatch_channels', 'accounting_transactions')
AND column_name IN ('channel_id', 'device_id', 'type', 'status');

-- Verify CHECK constraints
SELECT conname, pg_get_constraintdef(oid) 
FROM pg_constraint 
WHERE conname LIKE '%_check';

-- Verify RLS is enabled
SELECT relname, relrowsecurity 
FROM pg_class 
WHERE relname IN ('messages', 'balance_history', 'devices', 'channels', 'users', 'audit_trail');

-- Verify indexes
SELECT indexname, indexdef 
FROM pg_indexes 
WHERE tablename IN ('messages', 'balance_history', 'accounting_transactions', 'subscriptions');
```

### Code Verification

1. Verify .env files contain only placeholders
2. Verify Circuit Breaker has logger calls
3. Verify Rate Limiter uses user ID + IP
4. Verify CSRF manager has auto-refresh
5. Verify shared library imports in both frontends

### Security Verification

1. Run credential scanning tool on repository
2. Verify SSL mode is enabled for database connections
3. Verify RLS policies prevent cross-tenant access
4. Verify all financial columns have CHECK constraints
5. Verify Rate Limiting cannot be bypassed

## Migration Order Summary

| Phase | Migration | Description |
|-------|-----------|-------------|
| 1 | 050 | Create RBAC tables (role_permissions, user_roles) |
| 1 | 051 | Add NOT NULL constraints |
| 1 | 052 | Unify financial precision to DECIMAL(12,4) |
| 2 | 053 | Enable RLS on 6 tables |
| 2 | 054 | Create tenant_id indexes |
| 2 | 055 | Add CHECK constraints for financial columns |
| 3 | 056 | Create updated_at triggers |
| 3 | 057 | Unify UUID generation |

## Rollback Plan

If any migration causes issues, rollback scripts should be prepared:

```sql
-- Example rollback for NOT NULL constraints
ALTER TABLE messages 
ALTER COLUMN channel_id DROP NOT NULL;

-- Example rollback for CHECK constraints
ALTER TABLE messages 
DROP CONSTRAINT messages_cost_non_negative;
```

**Note:** Some changes (like enabling RLS) may require careful testing with production-like data before deployment.