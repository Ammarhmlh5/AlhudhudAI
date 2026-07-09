#!/bin/bash

################################################################################
# Backup Encryption Utility
# 
# This script provides encryption utilities for tariff system backups using
# AES-256-CBC encryption with OpenSSL.
#
# Requirements: 25.6
################################################################################

set -e

# Configuration
ENCRYPTION_ALGORITHM="${ENCRYPTION_ALGORITHM:-aes-256-cbc}"
KEY_DERIVATION="${KEY_DERIVATION:-pbkdf2}"
PBKDF2_ITERATIONS="${PBKDF2_ITERATIONS:-100000}"
ENCRYPTION_KEY_FILE="${ENCRYPTION_KEY_FILE:-/etc/tariff-system/backup.key}"
METADATA_SUFFIX=".meta"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

################################################################################
# Logging Functions
################################################################################

log_info() {
    echo -e "${BLUE}[INFO]${NC} $@"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $@"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $@"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $@" >&2
}

################################################################################
# Encryption Functions
################################################################################

# Encrypt a single file
encrypt_file() {
    local input_file="$1"
    local output_file="$2"
    local key_file="${3:-$ENCRYPTION_KEY_FILE}"
    
    if [ ! -f "$input_file" ]; then
        log_error "Input file not found: $input_file"
        return 1
    fi
    
    if [ ! -f "$key_file" ]; then
        log_error "Encryption key file not found: $key_file"
        return 1
    fi
    
    log_info "Encrypting: $(basename "$input_file")"
    log_info "Algorithm: $ENCRYPTION_ALGORITHM"
    log_info "Key derivation: $KEY_DERIVATION (iterations: $PBKDF2_ITERATIONS)"
    
    # Encrypt the file
    if openssl enc -"$ENCRYPTION_ALGORITHM" -salt -pbkdf2 -iter "$PBKDF2_ITERATIONS" \
        -in "$input_file" -out "$output_file" -pass file:"$key_file"; then
        
        # Create metadata file
        create_metadata "$input_file" "$output_file" "$key_file"
        
        log_success "Successfully encrypted: $(basename "$output_file")"
        return 0
    else
        log_error "Failed to encrypt: $(basename "$input_file")"
        return 1
    fi
}

# Decrypt a single file
decrypt_file() {
    local input_file="$1"
    local output_file="$2"
    local key_file="${3:-$ENCRYPTION_KEY_FILE}"
    
    if [ ! -f "$input_file" ]; then
        log_error "Input file not found: $input_file"
        return 1
    fi
    
    if [ ! -f "$key_file" ]; then
        log_error "Decryption key file not found: $key_file"
        return 1
    fi
    
    log_info "Decrypting: $(basename "$input_file")"
    
    # Decrypt the file
    if openssl enc -d -"$ENCRYPTION_ALGORITHM" -pbkdf2 -iter "$PBKDF2_ITERATIONS" \
        -in "$input_file" -out "$output_file" -pass file:"$key_file"; then
        
        log_success "Successfully decrypted: $(basename "$output_file")"
        return 0
    else
        log_error "Failed to decrypt: $(basename "$input_file")"
        return 1
    fi
}

# Create encryption metadata
create_metadata() {
    local original_file="$1"
    local encrypted_file="$2"
    local key_file="$3"
    
    local metadata_file="${encrypted_file}${METADATA_SUFFIX}"
    
    local original_size=$(stat -c%s "$original_file" 2>/dev/null || stat -f%z "$original_file" 2>/dev/null)
    local encrypted_size=$(stat -c%s "$encrypted_file" 2>/dev/null || stat -f%z "$encrypted_file" 2>/dev/null)
    local original_checksum=$(sha256sum "$original_file" | awk '{print $1}')
    local encrypted_checksum=$(sha256sum "$encrypted_file" | awk '{print $1}')
    
    cat > "$metadata_file" <<EOF
{
  "original_file": "$(basename "$original_file")",
  "encrypted_file": "$(basename "$encrypted_file")",
  "encryption_algorithm": "$ENCRYPTION_ALGORITHM",
  "key_derivation": "$KEY_DERIVATION",
  "pbkdf2_iterations": $PBKDF2_ITERATIONS,
  "original_size": $original_size,
  "encrypted_size": $encrypted_size,
  "original_checksum": "$original_checksum",
  "encrypted_checksum": "$encrypted_checksum",
  "encrypted_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "hostname": "$(hostname)",
  "key_file": "$(basename "$key_file")"
}
EOF
    
    log_info "Metadata created: $(basename "$metadata_file")"
}

# Verify encryption integrity
verify_encryption() {
    local encrypted_file="$1"
    local metadata_file="${encrypted_file}${METADATA_SUFFIX}"
    
    if [ ! -f "$encrypted_file" ]; then
        log_error "Encrypted file not found: $encrypted_file"
        return 1
    fi
    
    if [ ! -f "$metadata_file" ]; then
        log_warning "Metadata file not found: $metadata_file"
        return 1
    fi
    
    log_info "Verifying encryption: $(basename "$encrypted_file")"
    
    # Check file size
    local actual_size=$(stat -c%s "$encrypted_file" 2>/dev/null || stat -f%z "$encrypted_file" 2>/dev/null)
    local expected_size=$(grep -o '"encrypted_size": [0-9]*' "$metadata_file" | awk '{print $2}')
    
    if [ "$actual_size" != "$expected_size" ]; then
        log_error "File size mismatch: expected $expected_size, got $actual_size"
        return 1
    fi
    
    # Check checksum
    local actual_checksum=$(sha256sum "$encrypted_file" | awk '{print $1}')
    local expected_checksum=$(grep -o '"encrypted_checksum": "[^"]*"' "$metadata_file" | cut -d'"' -f4)
    
    if [ "$actual_checksum" != "$expected_checksum" ]; then
        log_error "Checksum mismatch"
        return 1
    fi
    
    log_success "Encryption verified successfully"
    return 0
}

# Test encryption/decryption round-trip
test_encryption_roundtrip() {
    local test_file="$1"
    local key_file="${2:-$ENCRYPTION_KEY_FILE}"
    
    if [ ! -f "$test_file" ]; then
        log_error "Test file not found: $test_file"
        return 1
    fi
    
    log_info "Testing encryption round-trip for: $(basename "$test_file")"
    
    local temp_dir=$(mktemp -d)
    local encrypted_file="$temp_dir/$(basename "$test_file").enc"
    local decrypted_file="$temp_dir/$(basename "$test_file").dec"
    
    # Calculate original checksum
    local original_checksum=$(sha256sum "$test_file" | awk '{print $1}')
    
    # Encrypt
    if ! encrypt_file "$test_file" "$encrypted_file" "$key_file"; then
        rm -rf "$temp_dir"
        return 1
    fi
    
    # Verify encryption
    if ! verify_encryption "$encrypted_file"; then
        rm -rf "$temp_dir"
        return 1
    fi
    
    # Decrypt
    if ! decrypt_file "$encrypted_file" "$decrypted_file" "$key_file"; then
        rm -rf "$temp_dir"
        return 1
    fi
    
    # Calculate decrypted checksum
    local decrypted_checksum=$(sha256sum "$decrypted_file" | awk '{print $1}')
    
    # Compare checksums
    if [ "$original_checksum" = "$decrypted_checksum" ]; then
        log_success "Round-trip test passed: checksums match"
        rm -rf "$temp_dir"
        return 0
    else
        log_error "Round-trip test failed: checksums do not match"
        log_error "Original: $original_checksum"
        log_error "Decrypted: $decrypted_checksum"
        rm -rf "$temp_dir"
        return 1
    fi
}

# Encrypt directory (all files)
encrypt_directory() {
    local input_dir="$1"
    local output_dir="$2"
    local key_file="${3:-$ENCRYPTION_KEY_FILE}"
    
    if [ ! -d "$input_dir" ]; then
        log_error "Input directory not found: $input_dir"
        return 1
    fi
    
    mkdir -p "$output_dir"
    
    log_info "Encrypting directory: $input_dir"
    
    local success_count=0
    local fail_count=0
    
    while IFS= read -r -d '' file; do
        local relative_path="${file#$input_dir/}"
        local output_file="$output_dir/${relative_path}.enc"
        local output_subdir=$(dirname "$output_file")
        
        mkdir -p "$output_subdir"
        
        if encrypt_file "$file" "$output_file" "$key_file"; then
            ((success_count++))
        else
            ((fail_count++))
        fi
    done < <(find "$input_dir" -type f -print0)
    
    log_info "Encryption complete: $success_count succeeded, $fail_count failed"
    
    if [ $fail_count -eq 0 ]; then
        return 0
    else
        return 1
    fi
}

################################################################################
# Usage Information
################################################################################

show_usage() {
    cat <<EOF
Usage: $0 <command> [options]

Commands:
  encrypt <input> <output> [key_file]    Encrypt a file
  decrypt <input> <output> [key_file]    Decrypt a file
  verify <encrypted_file>                Verify encryption integrity
  test <file> [key_file]                 Test encryption round-trip
  encrypt-dir <input_dir> <output_dir>   Encrypt all files in directory

Options:
  -a, --algorithm <algo>     Encryption algorithm (default: aes-256-cbc)
  -i, --iterations <num>     PBKDF2 iterations (default: 100000)
  -k, --key-file <file>      Encryption key file (default: $ENCRYPTION_KEY_FILE)
  -h, --help                 Show this help message

Examples:
  # Encrypt a backup file
  $0 encrypt backup.sql backup.sql.enc

  # Decrypt a backup file
  $0 decrypt backup.sql.enc backup.sql

  # Verify encryption
  $0 verify backup.sql.enc

  # Test encryption round-trip
  $0 test backup.sql

  # Encrypt entire directory
  $0 encrypt-dir /backups/daily /backups/daily-encrypted

Environment Variables:
  ENCRYPTION_ALGORITHM       Encryption algorithm (default: aes-256-cbc)
  KEY_DERIVATION            Key derivation method (default: pbkdf2)
  PBKDF2_ITERATIONS         PBKDF2 iterations (default: 100000)
  ENCRYPTION_KEY_FILE       Path to encryption key file

EOF
}

################################################################################
# Main Execution
################################################################################

main() {
    if [ $# -lt 1 ]; then
        show_usage
        exit 1
    fi
    
    local command="$1"
    shift
    
    case "$command" in
        encrypt)
            if [ $# -lt 2 ]; then
                log_error "Usage: $0 encrypt <input> <output> [key_file]"
                exit 1
            fi
            encrypt_file "$1" "$2" "${3:-$ENCRYPTION_KEY_FILE}"
            ;;
        
        decrypt)
            if [ $# -lt 2 ]; then
                log_error "Usage: $0 decrypt <input> <output> [key_file]"
                exit 1
            fi
            decrypt_file "$1" "$2" "${3:-$ENCRYPTION_KEY_FILE}"
            ;;
        
        verify)
            if [ $# -lt 1 ]; then
                log_error "Usage: $0 verify <encrypted_file>"
                exit 1
            fi
            verify_encryption "$1"
            ;;
        
        test)
            if [ $# -lt 1 ]; then
                log_error "Usage: $0 test <file> [key_file]"
                exit 1
            fi
            test_encryption_roundtrip "$1" "${2:-$ENCRYPTION_KEY_FILE}"
            ;;
        
        encrypt-dir)
            if [ $# -lt 2 ]; then
                log_error "Usage: $0 encrypt-dir <input_dir> <output_dir> [key_file]"
                exit 1
            fi
            encrypt_directory "$1" "$2" "${3:-$ENCRYPTION_KEY_FILE}"
            ;;
        
        -h|--help|help)
            show_usage
            exit 0
            ;;
        
        *)
            log_error "Unknown command: $command"
            show_usage
            exit 1
            ;;
    esac
}

# Run main function
main "$@"
