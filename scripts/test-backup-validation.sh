#!/bin/bash

################################################################################
# Backup Validation Test Script
# 
# This script tests the backup and restore functionality to ensure
# backups are valid and can be restored successfully.
#
# Requirements: 25.7, 25.10
################################################################################

set -e

# Configuration
BACKUP_DIR="${BACKUP_DIR:-/var/backups/tariff-system}"
TEST_DIR="${TEST_DIR:-/tmp/backup-validation-test}"
ENCRYPTION_KEY_FILE="${ENCRYPTION_KEY_FILE:-/etc/tariff-system/backup.key}"

# Test database
TEST_DB_NAME="tariff_backup_test_$(date +%s)"
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
PGPASSWORD="${DB_PASSWORD:-}"
export PGPASSWORD

# Timestamp
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Log file
LOG_FILE="$TEST_DIR/validation_${TIMESTAMP}.log"

# Test results
TESTS_PASSED=0
TESTS_FAILED=0

################################################################################
# Logging Functions
################################################################################

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

log_error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $1" | tee -a "$LOG_FILE" >&2
}

log_success() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ✓ $1" | tee -a "$LOG_FILE"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

log_failure() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ✗ $1" | tee -a "$LOG_FILE"
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

################################################################################
# Setup Functions
################################################################################

setup_test_environment() {
    log "Setting up test environment"
    
    # Create test directory
    mkdir -p "$TEST_DIR"
    
    # Create test database
    log "Creating test database: $TEST_DB_NAME"
    if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres \
        -c "CREATE DATABASE $TEST_DB_NAME;" 2>/dev/null; then
        log_success "Test database created"
    else
        log_failure "Failed to create test database"
        return 1
    fi
    
    # Create test encryption key if it doesn't exist
    if [ ! -f "$ENCRYPTION_KEY_FILE" ]; then
        log "Creating test encryption key"
        mkdir -p "$(dirname "$ENCRYPTION_KEY_FILE")"
        openssl rand -base64 32 > "$ENCRYPTION_KEY_FILE"
        chmod 600 "$ENCRYPTION_KEY_FILE"
    fi
    
    return 0
}

cleanup_test_environment() {
    log "Cleaning up test environment"
    
    # Drop test database
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres \
        -c "DROP DATABASE IF EXISTS $TEST_DB_NAME;" 2>/dev/null || true
    
    # Remove test directory
    rm -rf "$TEST_DIR"
}

################################################################################
# Test Functions
################################################################################

test_backup_script_exists() {
    log "Test: Backup script exists"
    
    if [ -f "scripts/backup-tariff-data.sh" ]; then
        log_success "Backup script found"
        return 0
    else
        log_failure "Backup script not found"
        return 1
    fi
}

test_restore_script_exists() {
    log "Test: Restore script exists"
    
    if [ -f "scripts/restore-tariff-data.sh" ]; then
        log_success "Restore script found"
        return 0
    else
        log_failure "Restore script not found"
        return 1
    fi
}

test_encryption_key_exists() {
    log "Test: Encryption key exists"
    
    if [ -f "$ENCRYPTION_KEY_FILE" ]; then
        log_success "Encryption key found"
        return 0
    else
        log_failure "Encryption key not found"
        return 1
    fi
}

test_backup_directory_structure() {
    log "Test: Backup directory structure"
    
    local all_exist=true
    
    for dir in daily weekly monthly temp; do
        if [ ! -d "$BACKUP_DIR/$dir" ]; then
            log_error "Directory missing: $BACKUP_DIR/$dir"
            all_exist=false
        fi
    done
    
    if [ "$all_exist" = true ]; then
        log_success "Backup directory structure is correct"
        return 0
    else
        log_failure "Backup directory structure is incomplete"
        return 1
    fi
}

test_latest_backup_exists() {
    log "Test: Latest backup exists"
    
    local latest_backup=$(find "$BACKUP_DIR/daily" -name "*.tar.gz" -type f -mtime -1 | head -n 1)
    
    if [ -n "$latest_backup" ]; then
        log_success "Latest backup found: $(basename "$latest_backup")"
        echo "$latest_backup" > "$TEST_DIR/latest_backup.txt"
        return 0
    else
        log_failure "No recent backup found (within last 24 hours)"
        return 1
    fi
}

test_backup_file_integrity() {
    log "Test: Backup file integrity"
    
    local latest_backup=$(cat "$TEST_DIR/latest_backup.txt" 2>/dev/null)
    
    if [ -z "$latest_backup" ] || [ ! -f "$latest_backup" ]; then
        log_failure "No backup file to test"
        return 1
    fi
    
    # Test if archive is valid
    if tar -tzf "$latest_backup" > /dev/null 2>&1; then
        log_success "Backup archive is valid"
        return 0
    else
        log_failure "Backup archive is corrupted"
        return 1
    fi
}

test_backup_encryption() {
    log "Test: Backup encryption"
    
    local latest_backup=$(cat "$TEST_DIR/latest_backup.txt" 2>/dev/null)
    
    if [ -z "$latest_backup" ] || [ ! -f "$latest_backup" ]; then
        log_failure "No backup file to test"
        return 1
    fi
    
    # Extract and check for encrypted files
    local extract_dir="$TEST_DIR/extract_test"
    mkdir -p "$extract_dir"
    tar -xzf "$latest_backup" -C "$extract_dir"
    
    local encrypted_count=$(find "$extract_dir" -name "*.enc" | wc -l)
    
    rm -rf "$extract_dir"
    
    if [ "$encrypted_count" -gt 0 ]; then
        log_success "Backup contains $encrypted_count encrypted files"
        return 0
    else
        log_failure "Backup does not contain encrypted files"
        return 1
    fi
}

test_backup_restore() {
    log "Test: Backup restore"
    
    local latest_backup=$(cat "$TEST_DIR/latest_backup.txt" 2>/dev/null)
    
    if [ -z "$latest_backup" ] || [ ! -f "$latest_backup" ]; then
        log_failure "No backup file to test"
        return 1
    fi
    
    # Run restore in verify-only mode
    if DB_NAME="$TEST_DB_NAME" bash scripts/restore-tariff-data.sh \
        --verify-only "$latest_backup" > "$TEST_DIR/restore_test.log" 2>&1; then
        log_success "Backup restore verification passed"
        return 0
    else
        log_failure "Backup restore verification failed"
        cat "$TEST_DIR/restore_test.log" >> "$LOG_FILE"
        return 1
    fi
}

test_retention_policy() {
    log "Test: Retention policy"
    
    # Check if old backups are being cleaned up
    local old_daily=$(find "$BACKUP_DIR/daily" -name "*.tar.gz" -mtime +30 | wc -l)
    local old_weekly=$(find "$BACKUP_DIR/weekly" -name "*.tar.gz" -mtime +84 | wc -l)
    local old_monthly=$(find "$BACKUP_DIR/monthly" -name "*.tar.gz" -mtime +365 | wc -l)
    
    if [ "$old_daily" -eq 0 ] && [ "$old_weekly" -eq 0 ] && [ "$old_monthly" -eq 0 ]; then
        log_success "Retention policy is working correctly"
        return 0
    else
        log_failure "Found old backups that should have been deleted (daily: $old_daily, weekly: $old_weekly, monthly: $old_monthly)"
        return 1
    fi
}

test_backup_size() {
    log "Test: Backup size"
    
    local latest_backup=$(cat "$TEST_DIR/latest_backup.txt" 2>/dev/null)
    
    if [ -z "$latest_backup" ] || [ ! -f "$latest_backup" ]; then
        log_failure "No backup file to test"
        return 1
    fi
    
    local file_size=$(stat -f%z "$latest_backup" 2>/dev/null || stat -c%s "$latest_backup" 2>/dev/null)
    local size_mb=$((file_size / 1024 / 1024))
    
    if [ "$file_size" -gt 1000 ]; then
        log_success "Backup size is reasonable: ${size_mb}MB"
        return 0
    else
        log_failure "Backup size is suspiciously small: ${file_size} bytes"
        return 1
    fi
}

################################################################################
# Main Test Execution
################################################################################

log "=========================================="
log "Starting Backup Validation Tests"
log "Timestamp: $TIMESTAMP"
log "=========================================="

# Setup
if ! setup_test_environment; then
    log_error "Failed to setup test environment"
    exit 1
fi

# Run tests
test_backup_script_exists
test_restore_script_exists
test_encryption_key_exists
test_backup_directory_structure
test_latest_backup_exists
test_backup_file_integrity
test_backup_encryption
test_backup_restore
test_retention_policy
test_backup_size

# Cleanup
cleanup_test_environment

################################################################################
# Test Summary
################################################################################

log "=========================================="
log "Test Summary"
log "=========================================="
log "Tests Passed: $TESTS_PASSED"
log "Tests Failed: $TESTS_FAILED"
log "Total Tests: $((TESTS_PASSED + TESTS_FAILED))"
log "=========================================="

if [ "$TESTS_FAILED" -eq 0 ]; then
    log "All tests passed! ✓"
    exit 0
else
    log_error "Some tests failed! ✗"
    exit 1
fi
