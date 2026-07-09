#!/bin/bash

################################################################################
# Backup Encryption Test Script
# 
# This script tests the backup encryption system including:
# - Encryption/decryption round-trip
# - Key rotation
# - Metadata verification
# - Integration with backup scripts
#
# Requirements: 25.6, 25.7
################################################################################

set -e

# Configuration
TEST_DIR="${TEST_DIR:-./test-encryption}"
KEY_DIR="$TEST_DIR/keys"
BACKUP_DIR="$TEST_DIR/backups"
RESTORE_DIR="$TEST_DIR/restore"
LOG_FILE="$TEST_DIR/test-results.log"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

################################################################################
# Logging Functions
################################################################################

log_test() {
    echo -e "${BLUE}[TEST]${NC} $@" | tee -a "$LOG_FILE"
}

log_pass() {
    echo -e "${GREEN}[PASS]${NC} $@" | tee -a "$LOG_FILE"
    ((TESTS_PASSED++))
}

log_fail() {
    echo -e "${RED}[FAIL]${NC} $@" | tee -a "$LOG_FILE"
    ((TESTS_FAILED++))
}

log_info() {
    echo -e "${BLUE}[INFO]${NC} $@" | tee -a "$LOG_FILE"
}

################################################################################
# Setup and Cleanup
################################################################################

setup_test_environment() {
    log_info "Setting up test environment"
    
    # Create directories
    mkdir -p "$TEST_DIR"
    mkdir -p "$KEY_DIR"
    mkdir -p "$BACKUP_DIR"
    mkdir -p "$RESTORE_DIR"
    
    # Set environment variables
    export KEY_DIR="$KEY_DIR"
    export ENCRYPTION_KEY_FILE="$KEY_DIR/backup.key"
    export BACKUP_DIR="$BACKUP_DIR"
    
    # Create test data
    echo "Test data for encryption" > "$TEST_DIR/test-file.txt"
    echo "SELECT * FROM tariffs;" > "$TEST_DIR/test-backup.sql"
    
    # Create larger test file
    dd if=/dev/urandom of="$TEST_DIR/large-file.bin" bs=1M count=10 2>/dev/null
    
    log_info "Test environment ready"
}

cleanup_test_environment() {
    log_info "Cleaning up test environment"
    rm -rf "$TEST_DIR"
}

################################################################################
# Test Functions
################################################################################

test_key_initialization() {
    ((TESTS_RUN++))
    log_test "Test 1: Key Initialization"
    
    if ./scripts/key-management.sh init; then
        if [ -f "$ENCRYPTION_KEY_FILE" ]; then
            log_pass "Key initialized successfully"
            return 0
        else
            log_fail "Key file not created"
            return 1
        fi
    else
        log_fail "Key initialization failed"
        return 1
    fi
}

test_key_permissions() {
    ((TESTS_RUN++))
    log_test "Test 2: Key Permissions"
    
    local perms=$(stat -c %a "$ENCRYPTION_KEY_FILE" 2>/dev/null || stat -f %A "$ENCRYPTION_KEY_FILE" 2>/dev/null)
    
    if [ "$perms" = "400" ]; then
        log_pass "Key permissions correct (400)"
        return 0
    else
        log_fail "Key permissions incorrect: $perms (expected 400)"
        return 1
    fi
}

test_encrypt_file() {
    ((TESTS_RUN++))
    log_test "Test 3: File Encryption"
    
    local input_file="$TEST_DIR/test-file.txt"
    local output_file="$BACKUP_DIR/test-file.txt.enc"
    
    if ./scripts/encrypt-backup.sh encrypt "$input_file" "$output_file"; then
        if [ -f "$output_file" ] && [ -f "${output_file}.meta" ]; then
            log_pass "File encrypted successfully"
            return 0
        else
            log_fail "Encrypted file or metadata not created"
            return 1
        fi
    else
        log_fail "Encryption failed"
        return 1
    fi
}

test_decrypt_file() {
    ((TESTS_RUN++))
    log_test "Test 4: File Decryption"
    
    local input_file="$BACKUP_DIR/test-file.txt.enc"
    local output_file="$RESTORE_DIR/test-file.txt"
    
    if ./scripts/encrypt-backup.sh decrypt "$input_file" "$output_file"; then
        if [ -f "$output_file" ]; then
            log_pass "File decrypted successfully"
            return 0
        else
            log_fail "Decrypted file not created"
            return 1
        fi
    else
        log_fail "Decryption failed"
        return 1
    fi
}

test_roundtrip_integrity() {
    ((TESTS_RUN++))
    log_test "Test 5: Round-trip Integrity"
    
    local original_file="$TEST_DIR/test-file.txt"
    local decrypted_file="$RESTORE_DIR/test-file.txt"
    
    local original_checksum=$(sha256sum "$original_file" | awk '{print $1}')
    local decrypted_checksum=$(sha256sum "$decrypted_file" | awk '{print $1}')
    
    if [ "$original_checksum" = "$decrypted_checksum" ]; then
        log_pass "Round-trip integrity verified (checksums match)"
        return 0
    else
        log_fail "Round-trip integrity failed (checksums differ)"
        log_fail "Original: $original_checksum"
        log_fail "Decrypted: $decrypted_checksum"
        return 1
    fi
}

test_encryption_verification() {
    ((TESTS_RUN++))
    log_test "Test 6: Encryption Verification"
    
    local encrypted_file="$BACKUP_DIR/test-file.txt.enc"
    
    if ./scripts/encrypt-backup.sh verify "$encrypted_file"; then
        log_pass "Encryption verification passed"
        return 0
    else
        log_fail "Encryption verification failed"
        return 1
    fi
}

test_large_file_encryption() {
    ((TESTS_RUN++))
    log_test "Test 7: Large File Encryption"
    
    local input_file="$TEST_DIR/large-file.bin"
    local encrypted_file="$BACKUP_DIR/large-file.bin.enc"
    local decrypted_file="$RESTORE_DIR/large-file.bin"
    
    log_info "Encrypting 10MB file..."
    if ! ./scripts/encrypt-backup.sh encrypt "$input_file" "$encrypted_file"; then
        log_fail "Large file encryption failed"
        return 1
    fi
    
    log_info "Decrypting 10MB file..."
    if ! ./scripts/encrypt-backup.sh decrypt "$encrypted_file" "$decrypted_file"; then
        log_fail "Large file decryption failed"
        return 1
    fi
    
    local original_checksum=$(sha256sum "$input_file" | awk '{print $1}')
    local decrypted_checksum=$(sha256sum "$decrypted_file" | awk '{print $1}')
    
    if [ "$original_checksum" = "$decrypted_checksum" ]; then
        log_pass "Large file encryption/decryption successful"
        return 0
    else
        log_fail "Large file integrity check failed"
        return 1
    fi
}

test_key_rotation() {
    ((TESTS_RUN++))
    log_test "Test 8: Key Rotation"
    
    local old_checksum=$(sha256sum "$ENCRYPTION_KEY_FILE" | awk '{print $1}')
    
    if ./scripts/key-management.sh rotate; then
        local new_checksum=$(sha256sum "$ENCRYPTION_KEY_FILE" | awk '{print $1}')
        
        if [ "$old_checksum" != "$new_checksum" ]; then
            log_pass "Key rotated successfully"
            return 0
        else
            log_fail "Key rotation did not change the key"
            return 1
        fi
    else
        log_fail "Key rotation failed"
        return 1
    fi
}

test_key_backup() {
    ((TESTS_RUN++))
    log_test "Test 9: Key Backup"
    
    local backup_file="$TEST_DIR/backup-key-test.key"
    
    if ./scripts/key-management.sh backup "$backup_file"; then
        if [ -f "$backup_file" ] && [ -f "${backup_file}.meta" ]; then
            log_pass "Key backed up successfully"
            return 0
        else
            log_fail "Backup file not created"
            return 1
        fi
    else
        log_fail "Key backup failed"
        return 1
    fi
}

test_key_verification() {
    ((TESTS_RUN++))
    log_test "Test 10: Key Verification"
    
    if ./scripts/key-management.sh verify; then
        log_pass "Key verification passed"
        return 0
    else
        log_fail "Key verification failed"
        return 1
    fi
}

test_metadata_creation() {
    ((TESTS_RUN++))
    log_test "Test 11: Metadata Creation"
    
    local metadata_file="$BACKUP_DIR/test-file.txt.enc.meta"
    
    if [ -f "$metadata_file" ]; then
        # Check required fields
        if grep -q '"encryption_algorithm"' "$metadata_file" && \
           grep -q '"key_derivation"' "$metadata_file" && \
           grep -q '"original_checksum"' "$metadata_file" && \
           grep -q '"encrypted_checksum"' "$metadata_file"; then
            log_pass "Metadata contains all required fields"
            return 0
        else
            log_fail "Metadata missing required fields"
            return 1
        fi
    else
        log_fail "Metadata file not found"
        return 1
    fi
}

test_sql_backup_encryption() {
    ((TESTS_RUN++))
    log_test "Test 12: SQL Backup Encryption"
    
    local input_file="$TEST_DIR/test-backup.sql"
    local encrypted_file="$BACKUP_DIR/test-backup.sql.enc"
    local decrypted_file="$RESTORE_DIR/test-backup.sql"
    
    if ! ./scripts/encrypt-backup.sh encrypt "$input_file" "$encrypted_file"; then
        log_fail "SQL backup encryption failed"
        return 1
    fi
    
    if ! ./scripts/encrypt-backup.sh decrypt "$encrypted_file" "$decrypted_file"; then
        log_fail "SQL backup decryption failed"
        return 1
    fi
    
    if diff -q "$input_file" "$decrypted_file" > /dev/null; then
        log_pass "SQL backup encryption/decryption successful"
        return 0
    else
        log_fail "SQL backup content differs after round-trip"
        return 1
    fi
}

test_wrong_key_decryption() {
    ((TESTS_RUN++))
    log_test "Test 13: Wrong Key Decryption (Should Fail)"
    
    local encrypted_file="$BACKUP_DIR/test-file.txt.enc"
    local wrong_key="$TEST_DIR/wrong.key"
    local output_file="$RESTORE_DIR/test-wrong-key.txt"
    
    # Generate wrong key
    openssl rand -base64 32 > "$wrong_key"
    
    # Try to decrypt with wrong key (should fail)
    if ./scripts/encrypt-backup.sh decrypt "$encrypted_file" "$output_file" "$wrong_key" 2>/dev/null; then
        log_fail "Decryption with wrong key should have failed"
        return 1
    else
        log_pass "Decryption correctly failed with wrong key"
        return 0
    fi
}

test_corrupted_file_detection() {
    ((TESTS_RUN++))
    log_test "Test 14: Corrupted File Detection"
    
    local encrypted_file="$BACKUP_DIR/test-file.txt.enc"
    local corrupted_file="$TEST_DIR/corrupted.enc"
    
    # Create corrupted copy
    cp "$encrypted_file" "$corrupted_file"
    echo "corrupted" >> "$corrupted_file"
    
    # Verification should fail
    if ./scripts/encrypt-backup.sh verify "$corrupted_file" 2>/dev/null; then
        log_fail "Corrupted file verification should have failed"
        return 1
    else
        log_pass "Corrupted file correctly detected"
        return 0
    fi
}

################################################################################
# Main Test Execution
################################################################################

main() {
    echo "=========================================="
    echo "Backup Encryption Test Suite"
    echo "=========================================="
    echo ""
    
    # Setup
    setup_test_environment
    
    # Run tests
    test_key_initialization
    test_key_permissions
    test_encrypt_file
    test_decrypt_file
    test_roundtrip_integrity
    test_encryption_verification
    test_large_file_encryption
    test_key_rotation
    test_key_backup
    test_key_verification
    test_metadata_creation
    test_sql_backup_encryption
    test_wrong_key_decryption
    test_corrupted_file_detection
    
    # Cleanup
    cleanup_test_environment
    
    # Results
    echo ""
    echo "=========================================="
    echo "Test Results"
    echo "=========================================="
    echo "Tests Run:    $TESTS_RUN"
    echo -e "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
    echo -e "Tests Failed: ${RED}$TESTS_FAILED${NC}"
    echo "=========================================="
    
    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "${GREEN}All tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}Some tests failed!${NC}"
        exit 1
    fi
}

# Run main function
main "$@"
