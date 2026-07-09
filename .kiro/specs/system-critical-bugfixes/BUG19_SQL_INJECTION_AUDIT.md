# Bug 19: SQL Injection Risk - Security Audit Report

**Date**: 2024
**Auditor**: Kiro AI Agent
**Scope**: Complete codebase scan for SQL injection vulnerabilities

## Executive Summary

Conducted comprehensive security audit of SQL queries across the backend codebase. Identified **5 files** with potential SQL injection vulnerabilities, categorized by severity.

### Risk Summary
- **HIGH RISK**: 2 instances (user-controlled input)
- **MEDIUM RISK**: 2 instances (dynamic query construction)
- **LOW RISK**: 1 instance (internal/admin tool)

## Detailed Findings

### 🔴 HIGH RISK - Requires Immediate Fix

#### 1. Dynamic Table Names in GDPR Data Deletion
**File**: `services/backend/internal/gdpr/data_deletion.go`
**Line**: 130
**Severity**: HIGH

**Vulnerable Code**:
```go
for _, table := range tables {
    query := fmt.Sprintf("DELETE FROM %s WHERE tenant_id = $1 AND user_id = $2", table)
    _, err := tx.ExecContext(ctx, query, tenantID, userID)
}
```

**Issue**: 
- Dynamic table names constructed with `fmt.Sprintf`
- If `tables` array is ever populated from user input, SQL injection is possible
- Current implementation uses hardcoded array, but pattern is unsafe

**Attack Vector**:
```go
// If tables array came from user input:
tables := []string{"users; DROP TABLE messages--"}
// Would execute: DELETE FROM users; DROP TABLE messages-- WHERE tenant_id = $1
```

**Recommendation**: Use whitelist validation for table names

---

#### 2. Schema Name in Database Migration Tool
**File**: `services/backend/cmd/fixdb2/main.go`
**Line**: 87
**Severity**: HIGH (for admin tool)

**Vulnerable Code**:
```go
query := fmt.Sprintf("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = '%s' AND table_name = 'river_job')", schemaName)
```

**Issue**:
- Schema name directly interpolated into SQL string
- Uses single quotes (string interpolation) instead of parameterized query
- `schemaName` derived from tenant ID with simple string replacement

**Attack Vector**:
```go
// If tenant ID was: "test' OR '1'='1"
// Query becomes: WHERE table_schema = 'test' OR '1'='1' AND table_name = 'river_job'
```

**Recommendation**: Use parameterized queries for schema checks

---

### 🟡 MEDIUM RISK - Should Be Fixed

#### 3. Dynamic ORDER BY in Fraud Detector
**File**: `services/backend/internal/fraud/detector.go`
**Line**: 349
**Severity**: MEDIUM

**Vulnerable Code**:
```go
query += " ORDER BY detected_at DESC LIMIT $" + fmt.Sprintf("%d", len(args)+1)
args = append(args, limit)
```

**Issue**:
- ORDER BY clause is hardcoded (safe)
- LIMIT parameter index constructed with fmt.Sprintf (safe, but poor practice)
- No user-controlled input in ORDER BY

**Current Status**: Safe, but uses anti-pattern

**Recommendation**: Use consistent parameterized approach

---

#### 4. Dynamic Query Building in Invoice Handler
**File**: `services/backend/internal/api/invoice_handler.go`
**Lines**: 330-377
**Severity**: MEDIUM

**Vulnerable Code**:
```go
query += fmt.Sprintf(" ORDER BY issue_date DESC LIMIT $%d OFFSET $%d", qIdx, qIdx+1)
qArgs = append(qArgs, limit, offset)
```

**Issue**:
- ORDER BY field is hardcoded (safe)
- Parameter indices constructed with fmt.Sprintf
- All user inputs are properly parameterized

**Current Status**: Safe, but uses anti-pattern

**Recommendation**: Use consistent parameterized approach

---

#### 5. Dynamic Query Building in Package Handler
**File**: `services/backend/internal/api/package_handler.go`
**Lines**: 404-428
**Severity**: MEDIUM

**Vulnerable Code**:
```go
query += fmt.Sprintf(" AND service_id = $%d", argIndex)
query += " ORDER BY display_order ASC, created_at DESC"
query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
```

**Issue**:
- ORDER BY fields are hardcoded (safe)
- Parameter indices constructed with fmt.Sprintf
- All user inputs are properly parameterized

**Current Status**: Safe, but uses anti-pattern

**Recommendation**: Use consistent parameterized approach

---

## Safe Patterns Found

### ✅ Properly Parameterized Queries

The following files demonstrate **correct** SQL injection prevention:

1. **fast_processor.go** (line 195):
```go
query := `
    UPDATE messages 
    SET status = $1, updated_at = NOW()
    WHERE id = ANY($2)
`
_, err := tx.Exec(ctx, query, status, messageIDs)
```

2. **Most API handlers** use proper parameterization for user inputs

---

## Recommendations

### Immediate Actions Required

1. **Fix GDPR Data Deletion** (HIGH PRIORITY)
   - Add whitelist validation for table names
   - Create safe helper function for dynamic table deletion

2. **Fix Database Migration Tool** (MEDIUM PRIORITY)
   - Use parameterized queries for schema checks
   - Add input validation for tenant IDs

3. **Standardize Query Building** (MEDIUM PRIORITY)
   - Create query builder helpers
   - Eliminate fmt.Sprintf for parameter indices

### Safe Query Building Patterns

#### Pattern 1: Whitelist for Dynamic Identifiers
```go
func buildOrderByClause(sortField string, sortOrder string) (string, error) {
    allowedFields := map[string]bool{
        "created_at": true,
        "updated_at": true,
        "status":     true,
    }
    
    if !allowedFields[sortField] {
        return "", fmt.Errorf("invalid sort field: %s", sortField)
    }
    
    if sortOrder != "ASC" && sortOrder != "DESC" {
        return "", fmt.Errorf("invalid sort order: %s", sortOrder)
    }
    
    return fmt.Sprintf("ORDER BY %s %s", sortField, sortOrder), nil
}
```

#### Pattern 2: Safe Table Name Validation
```go
func validateTableName(table string) error {
    allowedTables := map[string]bool{
        "user_consents":  true,
        "notifications":  true,
    }
    
    if !allowedTables[table] {
        return fmt.Errorf("invalid table name: %s", table)
    }
    return nil
}
```

#### Pattern 3: Always Use Parameterized Queries
```go
// GOOD
query := "SELECT * FROM users WHERE email = $1"
db.Query(query, email)

// BAD
query := fmt.Sprintf("SELECT * FROM users WHERE email = '%s'", email)
db.Query(query)
```

---

## Testing Strategy

### SQL Injection Test Cases

1. **Malicious Email Input**
```go
maliciousEmail := "admin@example.com' OR '1'='1"
users, err := GetUserByEmail(maliciousEmail)
// Should return empty, not all users
```

2. **Malicious Sort Field**
```go
_, err := GetMessages(filters, "id; DROP TABLE messages--", "ASC")
// Should return error, not execute DROP
```

3. **Malicious Table Name**
```go
err := DeleteFromTable("users; DROP TABLE messages--", tenantID)
// Should return error, not execute DROP
```

---

## Compliance Status

### Current State
- ✅ Most queries use parameterized approach
- ⚠️ 2 HIGH RISK vulnerabilities identified
- ⚠️ 3 MEDIUM RISK anti-patterns identified
- ✅ No evidence of active exploitation

### After Fix
- ✅ All queries will use parameterized approach
- ✅ Dynamic identifiers will use whitelist validation
- ✅ Safe query builder helpers implemented
- ✅ SQL injection tests added

---

## Files Requiring Changes

1. `services/backend/internal/gdpr/data_deletion.go` - Add table whitelist
2. `services/backend/cmd/fixdb2/main.go` - Use parameterized schema check
3. `services/backend/internal/fraud/detector.go` - Standardize query building
4. `services/backend/internal/api/invoice_handler.go` - Standardize query building
5. `services/backend/internal/api/package_handler.go` - Standardize query building

---

## Conclusion

The codebase demonstrates **good security practices** overall, with most queries properly parameterized. However, **2 HIGH RISK vulnerabilities** require immediate attention:

1. Dynamic table names in GDPR deletion
2. String interpolation in schema checks

All identified issues can be resolved by:
- Adding whitelist validation for dynamic identifiers
- Using parameterized queries consistently
- Creating safe query builder helpers

**Estimated Fix Time**: 2-3 hours
**Risk Level After Fix**: LOW
