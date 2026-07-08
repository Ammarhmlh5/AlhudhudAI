#!/bin/bash

################################################################################
# Encryption Key Management Script
# 
# This script manages encryption keys for backup encryption with secure
# storage, rotation, and recovery procedures.
#
# Requirements: 25.6
################################################################################

set -e

# Configuration
KEY_DIR="${KEY_DIR:-/etc/tariff-system/keys}"
ACTIVE_KEY_FILE="$KEY_DIR/backup.key"
KEY_BACKUP_DIR="$KEY_DIR/backups"
KEY_ROTATION_DAYS="${KEY_ROTATION_DAYS:-90}"
KEY_SIZE="${KEY_SIZE:-32}"  # 256 bits
LOG_FILE="${LOG_DIR:-/var/log}/key-management.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

################################################################################
# Logging Functions
################################################################################

log() {
    local level=$1
    shift
    local message="$@"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[$timestamp] [$level] $message" | tee -a "$LOG_FILE"
}

log_info() {
    log "INFO" "$@"
    echo -e "${BLUE}[INFO]${NC} $@"
}

log_success() {
    log "SUCCESS" "$@"
    echo -e "${GREEN}[SUCCESS]${NC} $@"
}

log_warning() {
    log "WARNING" "$@"
    echo -e "${YELLOW}[WARNING]${NC} $@"
}

log_error() {
    log "ERROR" "$@"
    echo -e "${RED}[ERROR]${NC} $@" >&2
}

################################################################################
# Key Generation Functions
################################################################################

# Generate a new encryption key
generate_key() {
    local key_file="$1"
    local key_version="${2:-v1}"
    
    log_info "Generating new encryption key: $key_version"
    
    # Create key directory if it doesn't exist
    mkdir -p "$(dirname "$key_file")"
    
    # Generate random key
    if openssl rand -base64 $KEY_SIZE > "$key_file"; then
        # Set secure permissions (read-only for owner)
        chmod 400 "$key_file"
        
        # Set owner to root (if running as root)
        if [ "$(id -u)" -eq 0 ]; then
            chown root:root "$key_file"
        fi
        
        log_success "Key generated successfully: $key_file"
        log_info "Key permissions: $(stat -c %a "$key_file")"
        
        # Create metadata
        create_key_metadata "$key_file" "$key_version"
        
        return 0
    else
        log_error "Failed to generate key"
        return 1
    fi
}

# Create key metadata
create_key_metadata() {
    local key_file="$1"
    local key_version="$2"
    local metadata_file="${key_file}.meta"
    
    local key_checksum=$(sha256sum "$key_file" | awk '{print $1}')
    
    cat > "$metadata_file" <<EOF
{
  "key_version": "$key_version",
  "key_file": "$(basename "$key_file")",
  "key_size": $KEY_SIZE,
  "key_checksum": "$key_checksum",
  "created_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "created_by": "$(whoami)",
  "hostname": "$(hostname)",
  "rotation_days": $KEY_ROTATION_DAYS,
  "next_rotation": "$(date -u -d "+$KEY_ROTATION_DAYS days" +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
    
    chmod 400 "$metadata_file"
    
    log_info "Key metadata created: $metadata_file"
}

# Initialize key management system
init_keys() {
    log_info "Initializing key management system"
    
    # Create directories
    mkdir -p "$KEY_DIR"
    mkdir -p "$KEY_BACKUP_DIR"
    
    # Set secure permissions on directories
    chmod 700 "$KEY_DIR"
    chmod 700 "$KEY_BACKUP_DIR"
    
    # Generate initial key if it doesn't exist
    if [ ! -f "$ACTIVE_KEY_FILE" ]; then
        log_info "No active key found, generating initial key"
        generate_key "$ACTIVE_KEY_FILE" "v1"
    else
        log_warning "Active key already exists: $ACTIVE_KEY_FILE"
    fi
    
    log_success "Key management system initialized"
}

################################################################################
# Key Rotation Functions
################################################################################

# Rotate encryption key
rotate_key() {
    log_info "Starting key rotation"
    
    if [ ! -f "$ACTIVE_KEY_FILE" ]; then
        log_error "No active key found to rotate"
        return 1
    fi
    
    # Determine next version number
    local current_version=$(get_key_version "$ACTIVE_KEY_FILE")
    local next_version=$(increment_version "$current_version")
    
    log_info "Current version: $current_version"
    log_info "Next version: $next_version"
    
    # Backup current key
    local backup_file="$KEY_BACKUP_DIR/backup.key.$current_version.$(date +%Y%m%d_%H%M%S)"
    
    log_info "Backing up current key to: $backup_file"
    cp "$ACTIVE_KEY_FILE" "$backup_file"
    cp "${ACTIVE_KEY_FILE}.meta" "${backup_file}.meta"
    chmod 400 "$backup_file"
    chmod 400 "${backup_file}.meta"
    
    # Generate new key
    local temp_key="$KEY_DIR/backup.key.new"
    if generate_key "$temp_key" "$next_version"; then
        # Replace active key
        mv "$temp_key" "$ACTIVE_KEY_FILE"
        mv "${temp_key}.meta" "${ACTIVE_KEY_FILE}.meta"
        
        log_success "Key rotation completed successfully"
        log_info "Old key backed up to: $backup_file"
        log_info "New key version: $next_version"
        
        return 0
    else
        log_error "Failed to generate new key"
        return 1
    fi
}

# Check if key rotation is needed
check_rotation_needed() {
    if [ ! -f "$ACTIVE_KEY_FILE" ]; then
        log_error "No active key found"
        return 1
    fi
    
    local metadata_file="${ACTIVE_KEY_FILE}.meta"
    
    if [ ! -f "$metadata_file" ]; then
        log_warning "No metadata found for active key"
        return 0  # Assume rotation needed
    fi
    
    local created_at=$(grep -o '"created_at": "[^"]*"' "$metadata_file" | cut -d'"' -f4)
    local created_timestamp=$(date -d "$created_at" +%s 2>/dev/null || echo 0)
    local current_timestamp=$(date +%s)
    local age_days=$(( (current_timestamp - created_timestamp) / 86400 ))
    
    log_info "Key age: $age_days days (rotation threshold: $KEY_ROTATION_DAYS days)"
    
    if [ $age_days -ge $KEY_ROTATION_DAYS ]; then
        log_warning "Key rotation needed (age: $age_days days)"
        return 0
    else
        log_info "Key rotation not needed yet"
        return 1
    fi
}

# Get key version from metadata
get_key_version() {
    local key_file="$1"
    local metadata_file="${key_file}.meta"
    
    if [ -f "$metadata_file" ]; then
        grep -o '"key_version": "[^"]*"' "$metadata_file" | cut -d'"' -f4
    else
        echo "v1"
    fi
}

# Increment version number
increment_version() {
    local version="$1"
    local number=$(echo "$version" | sed 's/v//')
    local next_number=$((number + 1))
    echo "v$next_number"
}

################################################################################
# Key Backup and Recovery Functions
################################################################################

# Backup key to secure location
backup_key() {
    local destination="$1"
    
    if [ ! -f "$ACTIVE_KEY_FILE" ]; then
        log_error "No active key found to backup"
        return 1
    fi
    
    log_info "Backing up key to: $destination"
    
    # Create destination directory
    mkdir -p "$(dirname "$destination")"
    
    # Copy key and metadata
    cp "$ACTIVE_KEY_FILE" "$destination"
    cp "${ACTIVE_KEY_FILE}.meta" "${destination}.meta"
    
    # Set secure permissions
    chmod 400 "$destination"
    chmod 400 "${destination}.meta"
    
    log_success "Key backed up successfully"
}

# Restore key from backup
restore_key() {
    local backup_file="$1"
    
    if [ ! -f "$backup_file" ]; then
        log_error "Backup file not found: $backup_file"
        return 1
    fi
    
    log_warning "Restoring key from backup: $backup_file"
    log_warning "This will replace the current active key!"
    
    read -p "Are you sure you want to continue? (yes/no): " confirm
    
    if [ "$confirm" != "yes" ]; then
        log_info "Restore cancelled"
        return 1
    fi
    
    # Backup current key before restoring
    if [ -f "$ACTIVE_KEY_FILE" ]; then
        local safety_backup="$KEY_BACKUP_DIR/backup.key.before_restore.$(date +%Y%m%d_%H%M%S)"
        log_info "Creating safety backup: $safety_backup"
        cp "$ACTIVE_KEY_FILE" "$safety_backup"
        cp "${ACTIVE_KEY_FILE}.meta" "${safety_backup}.meta"
    fi
    
    # Restore key
    cp "$backup_file" "$ACTIVE_KEY_FILE"
    if [ -f "${backup_file}.meta" ]; then
        cp "${backup_file}.meta" "${ACTIVE_KEY_FILE}.meta"
    fi
    
    # Set secure permissions
    chmod 400 "$ACTIVE_KEY_FILE"
    if [ -f "${ACTIVE_KEY_FILE}.meta" ]; then
        chmod 400 "${ACTIVE_KEY_FILE}.meta"
    fi
    
    log_success "Key restored successfully"
}

# List all key backups
list_backups() {
    log_info "Available key backups:"
    
    if [ ! -d "$KEY_BACKUP_DIR" ]; then
        log_warning "No backup directory found"
        return 0
    fi
    
    local count=0
    
    while IFS= read -r -d '' backup_file; do
        ((count++))
        local file_date=$(stat -c %y "$backup_file" | cut -d' ' -f1)
        local file_size=$(stat -c %s "$backup_file")
        
        echo "  $count. $(basename "$backup_file")"
        echo "     Date: $file_date"
        echo "     Size: $file_size bytes"
        
        if [ -f "${backup_file}.meta" ]; then
            local version=$(grep -o '"key_version": "[^"]*"' "${backup_file}.meta" | cut -d'"' -f4)
            echo "     Version: $version"
        fi
        
        echo ""
    done < <(find "$KEY_BACKUP_DIR" -name "backup.key.*" -type f -print0 | sort -z)
    
    if [ $count -eq 0 ]; then
        log_info "No backups found"
    else
        log_info "Total backups: $count"
    fi
}

################################################################################
# Key Verification Functions
################################################################################

# Verify key integrity
verify_key() {
    local key_file="${1:-$ACTIVE_KEY_FILE}"
    
    if [ ! -f "$key_file" ]; then
        log_error "Key file not found: $key_file"
        return 1
    fi
    
    log_info "Verifying key: $key_file"
    
    # Check file permissions
    local perms=$(stat -c %a "$key_file")
    if [ "$perms" != "400" ]; then
        log_warning "Insecure permissions: $perms (should be 400)"
    else
        log_success "Permissions OK: $perms"
    fi
    
    # Check file size
    local size=$(stat -c %s "$key_file")
    if [ $size -lt 32 ]; then
        log_error "Key file too small: $size bytes"
        return 1
    else
        log_success "File size OK: $size bytes"
    fi
    
    # Verify checksum if metadata exists
    local metadata_file="${key_file}.meta"
    if [ -f "$metadata_file" ]; then
        local expected_checksum=$(grep -o '"key_checksum": "[^"]*"' "$metadata_file" | cut -d'"' -f4)
        local actual_checksum=$(sha256sum "$key_file" | awk '{print $1}')
        
        if [ "$expected_checksum" = "$actual_checksum" ]; then
            log_success "Checksum verified"
        else
            log_error "Checksum mismatch!"
            log_error "Expected: $expected_checksum"
            log_error "Actual: $actual_checksum"
            return 1
        fi
    else
        log_warning "No metadata found for checksum verification"
    fi
    
    log_success "Key verification passed"
    return 0
}

################################################################################
# Usage Information
################################################################################

show_usage() {
    cat <<EOF
Usage: $0 <command> [options]

Commands:
  init                       Initialize key management system
  generate <key_file>        Generate a new encryption key
  rotate                     Rotate the active encryption key
  check-rotation             Check if key rotation is needed
  backup <destination>       Backup active key to destination
  restore <backup_file>      Restore key from backup
  list-backups               List all key backups
  verify [key_file]          Verify key integrity

Options:
  -d, --key-dir <dir>        Key directory (default: /etc/tariff-system/keys)
  -r, --rotation-days <num>  Key rotation period in days (default: 90)
  -s, --key-size <num>       Key size in bytes (default: 32)
  -h, --help                 Show this help message

Examples:
  # Initialize key management
  $0 init

  # Generate a new key
  $0 generate /path/to/new.key

  # Rotate the active key
  $0 rotate

  # Check if rotation is needed
  $0 check-rotation

  # Backup active key
  $0 backup /secure/location/backup.key

  # Restore from backup
  $0 restore /secure/location/backup.key

  # List all backups
  $0 list-backups

  # Verify key integrity
  $0 verify

Environment Variables:
  KEY_DIR              Key directory path
  KEY_ROTATION_DAYS    Key rotation period in days
  KEY_SIZE             Key size in bytes

EOF
}

################################################################################
# Main Execution
################################################################################

main() {
    # Create log directory
    mkdir -p "$(dirname "$LOG_FILE")"
    
    if [ $# -lt 1 ]; then
        show_usage
        exit 1
    fi
    
    local command="$1"
    shift
    
    case "$command" in
        init)
            init_keys
            ;;
        
        generate)
            if [ $# -lt 1 ]; then
                log_error "Usage: $0 generate <key_file>"
                exit 1
            fi
            generate_key "$1" "v1"
            ;;
        
        rotate)
            rotate_key
            ;;
        
        check-rotation)
            check_rotation_needed
            ;;
        
        backup)
            if [ $# -lt 1 ]; then
                log_error "Usage: $0 backup <destination>"
                exit 1
            fi
            backup_key "$1"
            ;;
        
        restore)
            if [ $# -lt 1 ]; then
                log_error "Usage: $0 restore <backup_file>"
                exit 1
            fi
            restore_key "$1"
            ;;
        
        list-backups)
            list_backups
            ;;
        
        verify)
            verify_key "${1:-$ACTIVE_KEY_FILE}"
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
