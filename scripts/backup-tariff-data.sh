#!/bin/bash

################################################################################
# Tariff System Backup Script
# 
# This script creates encrypted backups of critical tariff system data
# including tariffs, packages, subscriptions, and dedicated gateways.
#
# Requirements: 25.1, 25.2, 25.6
################################################################################

set -e

# Configuration
BACKUP_DIR="${BACKUP_DIR:-/var/backups/tariff-system}"
BACKUP_RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-30}"
BACKUP_RETENTION_WEEKS="${BACKUP_RETENTION_WEEKS:-12}"
BACKUP_RETENTION_MONTHS="${BACKUP_RETENTION_MONTHS:-12}"
ENCRYPTION_KEY_FILE="${ENCRYPTION_KEY_FILE:-/etc/tariff-system/backup.key}"
S3_BUCKET="${S3_BUCKET:-}"
ALERT_EMAIL="${ALERT_EMAIL:-admin@example.com}"

# Database connection
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-sms_gateway}"
DB_USER="${DB_USER:-postgres}"
PGPASSWORD="${DB_PASSWORD:-}"
export PGPASSWORD

# Timestamp
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
DATE=$(date +%Y-%m-%d)
DAY_OF_WEEK=$(date +%u)  # 1=Monday, 7=Sunday
DAY_OF_MONTH=$(date +%d)

# Backup types
BACKUP_TYPE="daily"
if [ "$DAY_OF_WEEK" = "7" ]; then
    BACKUP_TYPE="weekly"
fi
if [ "$DAY_OF_MONTH" = "01" ]; then
    BACKUP_TYPE="monthly"
fi

# Create backup directory structure
mkdir -p "$BACKUP_DIR"/{daily,weekly,monthly,temp}

# Log file
LOG_FILE="$BACKUP_DIR/backup_${TIMESTAMP}.log"

################################################################################
# Logging Functions
################################################################################

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

log_error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $1" | tee -a "$LOG_FILE" >&2
}

send_alert() {
    local subject="$1"
    local message="$2"
    
    if [ -n "$ALERT_EMAIL" ]; then
        echo "$message" | mail -s "$subject" "$ALERT_EMAIL" || true
    fi
    
    log_error "$message"
}

################################################################################
# Backup Functions
################################################################################

backup_table() {
    local table_name="$1"
    local output_file="$2"
    
    log "Backing up table: $table_name"
    
    if pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
        -t "$table_name" --no-owner --no-acl -F c -f "$output_file"; then
        log "Successfully backed up $table_name"
        return 0
    else
        log_error "Failed to backup $table_name"
        return 1
    fi
}

encrypt_file() {
    local input_file="$1"
    local output_file="$2"
    
    if [ ! -f "$ENCRYPTION_KEY_FILE" ]; then
        log_error "Encryption key file not found: $ENCRYPTION_KEY_FILE"
        return 1
    fi
    
    log "Encrypting: $(basename "$input_file")"
    
    if openssl enc -aes-256-cbc -salt -pbkdf2 -in "$input_file" \
        -out "$output_file" -pass file:"$ENCRYPTION_KEY_FILE"; then
        log "Successfully encrypted $(basename "$input_file")"
        rm -f "$input_file"
        return 0
    else
        log_error "Failed to encrypt $(basename "$input_file")"
        return 1
    fi
}

verify_backup() {
    local backup_file="$1"
    
    if [ ! -f "$backup_file" ]; then
        log_error "Backup file not found: $backup_file"
        return 1
    fi
    
    local file_size=$(stat -f%z "$backup_file" 2>/dev/null || stat -c%s "$backup_file" 2>/dev/null)
    
    if [ "$file_size" -lt 100 ]; then
        log_error "Backup file too small (${file_size} bytes): $backup_file"
        return 1
    fi
    
    log "Backup verified: $(basename "$backup_file") (${file_size} bytes)"
    return 0
}

upload_to_s3() {
    local local_file="$1"
    local s3_path="$2"
    
    if [ -z "$S3_BUCKET" ]; then
        log "S3 upload skipped (no bucket configured)"
        return 0
    fi
    
    log "Uploading to S3: $s3_path"
    
    if aws s3 cp "$local_file" "s3://${S3_BUCKET}/${s3_path}" \
        --storage-class STANDARD_IA --server-side-encryption AES256; then
        log "Successfully uploaded to S3"
        return 0
    else
        log_error "Failed to upload to S3"
        return 1
    fi
}

################################################################################
# Main Backup Process
################################################################################

log "=========================================="
log "Starting Tariff System Backup"
log "Backup Type: $BACKUP_TYPE"
log "Timestamp: $TIMESTAMP"
log "=========================================="

# Temporary directory for this backup
TEMP_DIR="$BACKUP_DIR/temp/$TIMESTAMP"
mkdir -p "$TEMP_DIR"

# Tables to backup
TABLES=(
    "services"
    "tariffs"
    "packages"
    "subscriptions"
    "rollover_messages"
    "dedicated_gateways"
    "resource_quotas"
    "resource_usage"
    "gateway_users"
    "gateway_api_keys"
    "gateway_statistics"
    "accounting_transactions"
    "balance_history"
    "segmentation_rules"
    "currencies"
    "exchange_rates"
    "billing_invoices"
)

# Backup each table
BACKUP_SUCCESS=true
for table in "${TABLES[@]}"; do
    backup_file="$TEMP_DIR/${table}.dump"
    encrypted_file="$TEMP_DIR/${table}.dump.enc"
    
    if backup_table "$table" "$backup_file"; then
        if encrypt_file "$backup_file" "$encrypted_file"; then
            if ! verify_backup "$encrypted_file"; then
                BACKUP_SUCCESS=false
            fi
        else
            BACKUP_SUCCESS=false
        fi
    else
        BACKUP_SUCCESS=false
    fi
done

# Create archive
if [ "$BACKUP_SUCCESS" = true ]; then
    log "Creating backup archive"
    
    ARCHIVE_NAME="tariff_backup_${BACKUP_TYPE}_${TIMESTAMP}.tar.gz"
    ARCHIVE_PATH="$BACKUP_DIR/$BACKUP_TYPE/$ARCHIVE_NAME"
    
    if tar -czf "$ARCHIVE_PATH" -C "$TEMP_DIR" .; then
        log "Successfully created archive: $ARCHIVE_NAME"
        
        # Verify archive
        if verify_backup "$ARCHIVE_PATH"; then
            # Upload to S3
            S3_PATH="backups/$BACKUP_TYPE/$DATE/$ARCHIVE_NAME"
            upload_to_s3 "$ARCHIVE_PATH" "$S3_PATH"
            
            log "Backup completed successfully"
        else
            BACKUP_SUCCESS=false
        fi
    else
        log_error "Failed to create archive"
        BACKUP_SUCCESS=false
    fi
fi

# Cleanup temp directory
rm -rf "$TEMP_DIR"

################################################################################
# Retention Policy
################################################################################

log "Applying retention policy"

# Daily backups: keep 30 days
find "$BACKUP_DIR/daily" -name "*.tar.gz" -mtime +$BACKUP_RETENTION_DAYS -delete
log "Cleaned up daily backups older than $BACKUP_RETENTION_DAYS days"

# Weekly backups: keep 12 weeks (84 days)
find "$BACKUP_DIR/weekly" -name "*.tar.gz" -mtime +$((BACKUP_RETENTION_WEEKS * 7)) -delete
log "Cleaned up weekly backups older than $BACKUP_RETENTION_WEEKS weeks"

# Monthly backups: keep 12 months (365 days)
find "$BACKUP_DIR/monthly" -name "*.tar.gz" -mtime +$((BACKUP_RETENTION_MONTHS * 30)) -delete
log "Cleaned up monthly backups older than $BACKUP_RETENTION_MONTHS months"

################################################################################
# Final Status
################################################################################

if [ "$BACKUP_SUCCESS" = true ]; then
    log "=========================================="
    log "Backup completed successfully"
    log "=========================================="
    exit 0
else
    send_alert "Tariff System Backup Failed" \
        "Backup failed on $(hostname) at $(date). Check log: $LOG_FILE"
    log "=========================================="
    log "Backup completed with errors"
    log "=========================================="
    exit 1
fi
