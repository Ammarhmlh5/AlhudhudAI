#!/bin/bash

################################################################################
# Tariff System Restore Script
# 
# This script restores encrypted backups of the tariff system data.
#
# Requirements: 25.8, 25.9
################################################################################

set -e

# Configuration
BACKUP_DIR="${BACKUP_DIR:-/var/backups/tariff-system}"
ENCRYPTION_KEY_FILE="${ENCRYPTION_KEY_FILE:-/etc/tariff-system/backup.key}"
RESTORE_TEMP_DIR="${RESTORE_TEMP_DIR:-/tmp/tariff-restore}"

# Database connection
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-sms_gateway}"
DB_USER="${DB_USER:-postgres}"
PGPASSWORD="${DB_PASSWORD:-}"
export PGPASSWORD

# Timestamp
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Log file
LOG_FILE="$BACKUP_DIR/restore_${TIMESTAMP}.log"

################################################################################
# Logging Functions
################################################################################

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

log_error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $1" | tee -a "$LOG_FILE" >&2
}

################################################################################
# Restore Functions
################################################################################

list_available_backups() {
    log "Available backups:"
    log ""
    log "Daily backups:"
    ls -lh "$BACKUP_DIR/daily"/*.tar.gz 2>/dev/null || log "  No daily backups found"
    log ""
    log "Weekly backups:"
    ls -lh "$BACKUP_DIR/weekly"/*.tar.gz 2>/dev/null || log "  No weekly backups found"
    log ""
    log "Monthly backups:"
    ls -lh "$BACKUP_DIR/monthly"/*.tar.gz 2>/dev/null || log "  No monthly backups found"
}

download_from_s3() {
    local s3_path="$1"
    local local_file="$2"
    
    if [ -z "$S3_BUCKET" ]; then
        log_error "S3 bucket not configured"
        return 1
    fi
    
    log "Downloading from S3: $s3_path"
    
    if aws s3 cp "s3://${S3_BUCKET}/${s3_path}" "$local_file"; then
        log "Successfully downloaded from S3"
        return 0
    else
        log_error "Failed to download from S3"
        return 1
    fi
}

decrypt_file() {
    local input_file="$1"
    local output_file="$2"
    
    if [ ! -f "$ENCRYPTION_KEY_FILE" ]; then
        log_error "Encryption key file not found: $ENCRYPTION_KEY_FILE"
        return 1
    fi
    
    log "Decrypting: $(basename "$input_file")"
    
    if openssl enc -aes-256-cbc -d -pbkdf2 -in "$input_file" \
        -out "$output_file" -pass file:"$ENCRYPTION_KEY_FILE"; then
        log "Successfully decrypted $(basename "$input_file")"
        return 0
    else
        log_error "Failed to decrypt $(basename "$input_file")"
        return 1
    fi
}

restore_table() {
    local table_name="$1"
    local dump_file="$2"
    local drop_existing="${3:-false}"
    
    log "Restoring table: $table_name"
    
    # Drop existing table if requested
    if [ "$drop_existing" = "true" ]; then
        log "Dropping existing table: $table_name"
        psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
            -c "DROP TABLE IF EXISTS $table_name CASCADE;" || true
    fi
    
    # Restore table
    if pg_restore -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
        --no-owner --no-acl "$dump_file"; then
        log "Successfully restored $table_name"
        return 0
    else
        log_error "Failed to restore $table_name"
        return 1
    fi
}

verify_restore() {
    local table_name="$1"
    
    log "Verifying table: $table_name"
    
    local row_count=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
        -t -c "SELECT COUNT(*) FROM $table_name;" 2>/dev/null || echo "0")
    
    row_count=$(echo "$row_count" | tr -d ' ')
    
    log "Table $table_name has $row_count rows"
    
    if [ "$row_count" -gt 0 ]; then
        return 0
    else
        log_error "Table $table_name appears to be empty"
        return 1
    fi
}

create_restore_point() {
    local restore_point_name="before_restore_${TIMESTAMP}"
    
    log "Creating restore point: $restore_point_name"
    
    if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
        -c "SELECT pg_create_restore_point('$restore_point_name');" 2>/dev/null; then
        log "Restore point created successfully"
        return 0
    else
        log "Warning: Could not create restore point (requires superuser)"
        return 0
    fi
}

################################################################################
# Main Restore Process
################################################################################

show_usage() {
    cat <<EOF
Usage: $0 [OPTIONS] BACKUP_FILE

Restore tariff system data from an encrypted backup.

OPTIONS:
    -h, --help              Show this help message
    -l, --list              List available backups
    -d, --drop              Drop existing tables before restore
    -t, --tables TABLE,...  Restore only specific tables (comma-separated)
    -s, --s3 S3_PATH        Download backup from S3 before restoring
    -v, --verify-only       Verify backup without restoring

EXAMPLES:
    # List available backups
    $0 --list

    # Restore from local backup
    $0 /var/backups/tariff-system/daily/tariff_backup_daily_20240101_120000.tar.gz

    # Restore from S3
    $0 --s3 backups/daily/2024-01-01/tariff_backup_daily_20240101_120000.tar.gz

    # Restore only specific tables
    $0 --tables tariffs,packages backup.tar.gz

    # Drop existing tables before restore
    $0 --drop backup.tar.gz

EOF
}

# Parse command line arguments
BACKUP_FILE=""
DROP_EXISTING=false
SPECIFIC_TABLES=""
S3_PATH=""
VERIFY_ONLY=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_usage
            exit 0
            ;;
        -l|--list)
            list_available_backups
            exit 0
            ;;
        -d|--drop)
            DROP_EXISTING=true
            shift
            ;;
        -t|--tables)
            SPECIFIC_TABLES="$2"
            shift 2
            ;;
        -s|--s3)
            S3_PATH="$2"
            shift 2
            ;;
        -v|--verify-only)
            VERIFY_ONLY=true
            shift
            ;;
        *)
            BACKUP_FILE="$1"
            shift
            ;;
    esac
done

# Validate arguments
if [ -z "$BACKUP_FILE" ] && [ -z "$S3_PATH" ]; then
    log_error "No backup file specified"
    show_usage
    exit 1
fi

log "=========================================="
log "Starting Tariff System Restore"
log "Timestamp: $TIMESTAMP"
log "=========================================="

# Create temporary directory
mkdir -p "$RESTORE_TEMP_DIR"

# Download from S3 if needed
if [ -n "$S3_PATH" ]; then
    BACKUP_FILE="$RESTORE_TEMP_DIR/$(basename "$S3_PATH")"
    if ! download_from_s3 "$S3_PATH" "$BACKUP_FILE"; then
        log_error "Failed to download backup from S3"
        exit 1
    fi
fi

# Verify backup file exists
if [ ! -f "$BACKUP_FILE" ]; then
    log_error "Backup file not found: $BACKUP_FILE"
    exit 1
fi

log "Using backup file: $BACKUP_FILE"

# Extract archive
EXTRACT_DIR="$RESTORE_TEMP_DIR/extract_${TIMESTAMP}"
mkdir -p "$EXTRACT_DIR"

log "Extracting backup archive"
if ! tar -xzf "$BACKUP_FILE" -C "$EXTRACT_DIR"; then
    log_error "Failed to extract backup archive"
    exit 1
fi

# Get list of tables to restore
if [ -n "$SPECIFIC_TABLES" ]; then
    IFS=',' read -ra TABLES <<< "$SPECIFIC_TABLES"
else
    TABLES=($(ls "$EXTRACT_DIR"/*.dump.enc 2>/dev/null | xargs -n1 basename | sed 's/.dump.enc$//'))
fi

if [ ${#TABLES[@]} -eq 0 ]; then
    log_error "No tables found in backup"
    exit 1
fi

log "Tables to restore: ${TABLES[*]}"

# Verify only mode
if [ "$VERIFY_ONLY" = true ]; then
    log "Verification mode - no data will be restored"
    
    for table in "${TABLES[@]}"; do
        encrypted_file="$EXTRACT_DIR/${table}.dump.enc"
        decrypted_file="$EXTRACT_DIR/${table}.dump"
        
        if [ -f "$encrypted_file" ]; then
            if decrypt_file "$encrypted_file" "$decrypted_file"; then
                log "✓ $table backup is valid"
                rm -f "$decrypted_file"
            else
                log_error "✗ $table backup is corrupted"
            fi
        else
            log_error "✗ $table backup file not found"
        fi
    done
    
    rm -rf "$EXTRACT_DIR"
    log "Verification complete"
    exit 0
fi

# Create restore point
create_restore_point

# Restore tables
RESTORE_SUCCESS=true
for table in "${TABLES[@]}"; do
    encrypted_file="$EXTRACT_DIR/${table}.dump.enc"
    decrypted_file="$EXTRACT_DIR/${table}.dump"
    
    if [ ! -f "$encrypted_file" ]; then
        log_error "Backup file not found for table: $table"
        RESTORE_SUCCESS=false
        continue
    fi
    
    # Decrypt
    if ! decrypt_file "$encrypted_file" "$decrypted_file"; then
        RESTORE_SUCCESS=false
        continue
    fi
    
    # Restore
    if restore_table "$table" "$decrypted_file" "$DROP_EXISTING"; then
        # Verify
        if ! verify_restore "$table"; then
            RESTORE_SUCCESS=false
        fi
    else
        RESTORE_SUCCESS=false
    fi
    
    # Cleanup decrypted file
    rm -f "$decrypted_file"
done

# Cleanup
rm -rf "$EXTRACT_DIR"

################################################################################
# Post-Restore Validation
################################################################################

run_post_restore_validation() {
    log "Running post-restore validation"
    
    local validation_failed=false
    
    # Check table row counts
    log "Checking table row counts"
    for table in "${TABLES[@]}"; do
        local count=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
            -t -c "SELECT COUNT(*) FROM $table;" 2>/dev/null || echo "0")
        count=$(echo "$count" | tr -d ' ')
        
        if [ "$count" -eq 0 ]; then
            log_error "Warning: Table $table has 0 rows"
            validation_failed=true
        else
            log "✓ Table $table: $count rows"
        fi
    done
    
    # Check referential integrity
    log "Checking referential integrity"
    
    # Check subscriptions -> packages
    local orphaned=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
        -t -c "SELECT COUNT(*) FROM subscriptions s LEFT JOIN packages p ON s.package_id = p.id WHERE p.id IS NULL;" 2>/dev/null || echo "0")
    orphaned=$(echo "$orphaned" | tr -d ' ')
    
    if [ "$orphaned" -gt 0 ]; then
        log_error "Found $orphaned orphaned subscriptions"
        validation_failed=true
    else
        log "✓ No orphaned subscriptions"
    fi
    
    # Check tariffs -> services
    orphaned=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
        -t -c "SELECT COUNT(*) FROM tariffs t LEFT JOIN services s ON t.service_id = s.id WHERE s.id IS NULL;" 2>/dev/null || echo "0")
    orphaned=$(echo "$orphaned" | tr -d ' ')
    
    if [ "$orphaned" -gt 0 ]; then
        log_error "Found $orphaned orphaned tariffs"
        validation_failed=true
    else
        log "✓ No orphaned tariffs"
    fi
    
    # Check data consistency
    log "Checking data consistency"
    
    # Check for negative balances
    local negative=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
        -t -c "SELECT COUNT(*) FROM accounting_transactions WHERE amount < 0 AND transaction_type = 'CREDIT';" 2>/dev/null || echo "0")
    negative=$(echo "$negative" | tr -d ' ')
    
    if [ "$negative" -gt 0 ]; then
        log_error "Found $negative invalid credit transactions"
        validation_failed=true
    else
        log "✓ No invalid credit transactions"
    fi
    
    # Check for invalid dates
    local invalid_dates=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
        -t -c "SELECT COUNT(*) FROM subscriptions WHERE end_date < start_date;" 2>/dev/null || echo "0")
    invalid_dates=$(echo "$invalid_dates" | tr -d ' ')
    
    if [ "$invalid_dates" -gt 0 ]; then
        log_error "Found $invalid_dates subscriptions with invalid dates"
        validation_failed=true
    else
        log "✓ No subscriptions with invalid dates"
    fi
    
    if [ "$validation_failed" = true ]; then
        log_error "Post-restore validation found issues"
        return 1
    else
        log "✓ Post-restore validation passed"
        return 0
    fi
}

################################################################################
# Generate Restore Report
################################################################################

generate_restore_report() {
    local report_file="$BACKUP_DIR/restore_report_${TIMESTAMP}.txt"
    
    log "Generating restore report: $report_file"
    
    cat > "$report_file" <<EOF
================================================================================
Tariff System Restore Report
================================================================================

Restore Date: $(date '+%Y-%m-%d %H:%M:%S')
Backup File: $BACKUP_FILE
Restore Type: ${SPECIFIC_TABLES:-Full Restore}
Drop Existing: $DROP_EXISTING

--------------------------------------------------------------------------------
Tables Restored
--------------------------------------------------------------------------------

EOF
    
    for table in "${TABLES[@]}"; do
        local count=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
            -t -c "SELECT COUNT(*) FROM $table;" 2>/dev/null || echo "0")
        count=$(echo "$count" | tr -d ' ')
        echo "$table: $count rows" >> "$report_file"
    done
    
    cat >> "$report_file" <<EOF

--------------------------------------------------------------------------------
Validation Results
--------------------------------------------------------------------------------

EOF
    
    # Add validation results
    if run_post_restore_validation >> "$report_file" 2>&1; then
        echo "Status: PASSED" >> "$report_file"
    else
        echo "Status: FAILED (see details above)" >> "$report_file"
    fi
    
    cat >> "$report_file" <<EOF

--------------------------------------------------------------------------------
Recovery Time
--------------------------------------------------------------------------------

Start Time: $(date -d "@$START_TIME" '+%Y-%m-%d %H:%M:%S' 2>/dev/null || date -r "$START_TIME" '+%Y-%m-%d %H:%M:%S' 2>/dev/null)
End Time: $(date '+%Y-%m-%d %H:%M:%S')
Duration: $(($(date +%s) - START_TIME)) seconds

--------------------------------------------------------------------------------
Log File
--------------------------------------------------------------------------------

$LOG_FILE

================================================================================
EOF
    
    log "Restore report generated: $report_file"
    
    # Display report
    cat "$report_file"
}

################################################################################
# Final Status
################################################################################

# Record start time
START_TIME=$(date +%s)

if [ "$RESTORE_SUCCESS" = true ]; then
    # Run post-restore validation
    if run_post_restore_validation; then
        VALIDATION_SUCCESS=true
    else
        VALIDATION_SUCCESS=false
    fi
    
    # Generate report
    generate_restore_report
    
    if [ "$VALIDATION_SUCCESS" = true ]; then
        log "=========================================="
        log "Restore completed successfully"
        log "All validation checks passed"
        log "=========================================="
        exit 0
    else
        log_error "=========================================="
        log_error "Restore completed with validation warnings"
        log_error "Review the restore report for details"
        log_error "=========================================="
        exit 2
    fi
else
    log_error "=========================================="
    log_error "Restore completed with errors"
    log_error "=========================================="
    exit 1
fi
