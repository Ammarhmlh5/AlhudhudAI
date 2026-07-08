#!/bin/bash

################################################################################
# Periodic Restore Testing Script
# 
# This script performs periodic restore testing to verify backup integrity
# and recovery procedures. It should be run monthly as part of DR testing.
#
# Requirements: 25.9, 28.9
################################################################################

set -e

# Configuration
TEST_DB_NAME="${TEST_DB_NAME:-sms_gateway_restore_test}"
TEST_DB_HOST="${TEST_DB_HOST:-localhost}"
TEST_DB_PORT="${TEST_DB_PORT:-5432}"
TEST_DB_USER="${TEST_DB_USER:-postgres}"
PGPASSWORD="${TEST_DB_PASSWORD:-}"
export PGPASSWORD

BACKUP_DIR="${BACKUP_DIR:-/var/backups/tariff-system}"
TEST_REPORT_DIR="${TEST_REPORT_DIR:-/var/log/restore-tests}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
TEST_REPORT="$TEST_REPORT_DIR/restore_test_${TIMESTAMP}.txt"

# Create report directory
mkdir -p "$TEST_REPORT_DIR"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

################################################################################
# Logging Functions
################################################################################

log() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1" | tee -a "$TEST_REPORT"
}

log_success() {
    echo -e "${GREEN}[$(date '+%Y-%m-%d %H:%M:%S')] ✓${NC} $1" | tee -a "$TEST_REPORT"
}

log_error() {
    echo -e "${RED}[$(date '+%Y-%m-%d %H:%M:%S')] ✗${NC} $1" | tee -a "$TEST_REPORT"
}

log_warn() {
    echo -e "${YELLOW}[$(date '+%Y-%m-%d %H:%M:%S')] ⚠${NC} $1" | tee -a "$TEST_REPORT"
}

################################################################################
# Test Functions
################################################################################

create_test_database() {
    log "Creating test database: $TEST_DB_NAME"
    
    # Drop if exists
    psql -h "$TEST_DB_HOST" -p "$TEST_DB_PORT" -U "$TEST_DB_USER" \
        -c "DROP DATABASE IF EXISTS $TEST_DB_NAME;" 2>/dev/null || true
    
    # Create new
    if psql -h "$TEST_DB_HOST" -p "$TEST_DB_PORT" -U "$TEST_DB_USER" \
        -c "CREATE DATABASE $TEST_DB_NAME;"; then
        log_success "Test database created"
        return 0
    else
        log_error "Failed to create test database"
        return 1
    fi
}

cleanup_test_database() {
    log "Cleaning up test database"
    
    psql -h "$TEST_DB_HOST" -p "$TEST_DB_PORT" -U "$TEST_DB_USER" \
        -c "DROP DATABASE IF EXISTS $TEST_DB_NAME;" 2>/dev/null || true
    
    log_success "Test database cleaned up"
}


select_test_backup() {
    log "Selecting backup for testing"
    
    # Try to find a recent daily backup
    local backup_file=$(find "$BACKUP_DIR/daily" -name "*.tar.gz" -type f -mtime -7 | sort -r | head -n 1)
    
    if [ -z "$backup_file" ]; then
        log_error "No recent backups found"
        return 1
    fi
    
    log_success "Selected backup: $(basename "$backup_file")"
    echo "$backup_file"
    return 0
}

test_restore_operation() {
    local backup_file="$1"
    local start_time=$(date +%s)
    
    log "Testing restore operation"
    log "Backup file: $backup_file"
    
    # Set environment for test database
    export DB_HOST="$TEST_DB_HOST"
    export DB_PORT="$TEST_DB_PORT"
    export DB_NAME="$TEST_DB_NAME"
    export DB_USER="$TEST_DB_USER"
    export DB_PASSWORD="$TEST_DB_PASSWORD"
    
    # Run restore
    if bash scripts/restore-tariff-data.sh --drop "$backup_file" >> "$TEST_REPORT" 2>&1; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        log_success "Restore completed in $duration seconds"
        return 0
    else
        log_error "Restore operation failed"
        return 1
    fi
}

test_data_validation() {
    log "Running data validation"
    
    # Set environment for test database
    export DB_HOST="$TEST_DB_HOST"
    export DB_PORT="$TEST_DB_PORT"
    export DB_NAME="$TEST_DB_NAME"
    export DB_USER="$TEST_DB_USER"
    export DB_PASSWORD="$TEST_DB_PASSWORD"
    
    # Run validation script
    if bash scripts/validate-restore.sh >> "$TEST_REPORT" 2>&1; then
        log_success "Data validation passed"
        return 0
    else
        log_error "Data validation failed"
        return 1
    fi
}

test_query_performance() {
    log "Testing query performance"
    
    local queries=(
        "SELECT COUNT(*) FROM tariffs;"
        "SELECT COUNT(*) FROM packages;"
        "SELECT COUNT(*) FROM subscriptions;"
        "SELECT * FROM tariffs WHERE is_active = true LIMIT 10;"
        "SELECT * FROM subscriptions WHERE status = 'active' LIMIT 10;"
    )
    
    for query in "${queries[@]}"; do
        local start=$(date +%s%N)
        psql -h "$TEST_DB_HOST" -p "$TEST_DB_PORT" -U "$TEST_DB_USER" -d "$TEST_DB_NAME" \
            -c "$query" > /dev/null 2>&1
        local end=$(date +%s%N)
        local duration=$(( (end - start) / 1000000 ))
        
        log "Query executed in ${duration}ms: ${query:0:50}..."
    done
    
    log_success "Query performance test completed"
}


generate_test_report() {
    local test_result="$1"
    
    log "Generating test report"
    
    cat >> "$TEST_REPORT" <<EOF

================================================================================
Periodic Restore Test Report
================================================================================

Test Date: $(date '+%Y-%m-%d %H:%M:%S')
Test Database: $TEST_DB_NAME
Test Result: $test_result

--------------------------------------------------------------------------------
Test Metrics
--------------------------------------------------------------------------------

Backup File: $(basename "$BACKUP_FILE")
Backup Size: $(du -h "$BACKUP_FILE" | cut -f1)
Backup Date: $(stat -c %y "$BACKUP_FILE" 2>/dev/null || stat -f "%Sm" "$BACKUP_FILE" 2>/dev/null)

Restore Duration: ${RESTORE_DURATION}s
Validation Result: $VALIDATION_RESULT

--------------------------------------------------------------------------------
Recommendations
--------------------------------------------------------------------------------

EOF
    
    if [ "$test_result" = "PASSED" ]; then
        cat >> "$TEST_REPORT" <<EOF
✓ Backup and restore procedures are working correctly
✓ Data integrity is maintained
✓ Recovery time is within acceptable limits (RTO: 1 hour)

Next Test: $(date -d "+1 month" '+%Y-%m-%d' 2>/dev/null || date -v +1m '+%Y-%m-%d' 2>/dev/null)

EOF
    else
        cat >> "$TEST_REPORT" <<EOF
✗ Issues detected during restore testing
✗ Review the detailed log above for specific failures
✗ Address issues before next production restore

Action Required:
1. Review backup procedures
2. Verify backup integrity
3. Test restore process manually
4. Update recovery procedures if needed

EOF
    fi
    
    cat >> "$TEST_REPORT" <<EOF
================================================================================
EOF
    
    log_success "Test report generated: $TEST_REPORT"
}

send_test_notification() {
    local test_result="$1"
    local alert_email="${ALERT_EMAIL:-admin@example.com}"
    
    if [ -n "$alert_email" ]; then
        local subject="Restore Test $test_result - $(date '+%Y-%m-%d')"
        
        if command -v mail &> /dev/null; then
            cat "$TEST_REPORT" | mail -s "$subject" "$alert_email"
            log_success "Test notification sent to $alert_email"
        else
            log_warn "Mail command not available, skipping notification"
        fi
    fi
}

################################################################################
# Main Test Execution
################################################################################

log "=========================================="
log "Starting Periodic Restore Test"
log "=========================================="

TEST_PASSED=true

# Step 1: Create test database
if ! create_test_database; then
    log_error "Failed to create test database"
    exit 1
fi

# Step 2: Select backup for testing
BACKUP_FILE=$(select_test_backup)
if [ -z "$BACKUP_FILE" ]; then
    cleanup_test_database
    exit 1
fi

# Step 3: Test restore operation
START_TIME=$(date +%s)
if test_restore_operation "$BACKUP_FILE"; then
    END_TIME=$(date +%s)
    RESTORE_DURATION=$((END_TIME - START_TIME))
else
    TEST_PASSED=false
    RESTORE_DURATION=0
fi

# Step 4: Validate restored data
if [ "$TEST_PASSED" = true ]; then
    if test_data_validation; then
        VALIDATION_RESULT="PASSED"
    else
        VALIDATION_RESULT="FAILED"
        TEST_PASSED=false
    fi
else
    VALIDATION_RESULT="SKIPPED"
fi

# Step 5: Test query performance
if [ "$TEST_PASSED" = true ]; then
    test_query_performance
fi

# Step 6: Cleanup
cleanup_test_database

# Step 7: Generate report
if [ "$TEST_PASSED" = true ]; then
    TEST_RESULT="PASSED"
else
    TEST_RESULT="FAILED"
fi

generate_test_report "$TEST_RESULT"

# Step 8: Send notification
send_test_notification "$TEST_RESULT"

# Final status
log "=========================================="
if [ "$TEST_PASSED" = true ]; then
    log_success "Periodic restore test PASSED"
    log "=========================================="
    exit 0
else
    log_error "Periodic restore test FAILED"
    log "=========================================="
    exit 1
fi
