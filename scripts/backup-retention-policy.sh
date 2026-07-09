#!/bin/bash

################################################################################
# Backup Retention Policy Script
# 
# This script implements the retention policy for tariff system backups:
# - Daily backups: Keep for 30 days
# - Weekly backups: Keep for 3 months (90 days)
# - Monthly backups: Keep for 1 year (365 days)
#
# Requirements: 25.2, 25.3, 25.4
################################################################################

set -e

# Configuration
BACKUP_BASE_DIR="${BACKUP_DIR:-./backups}"
DAILY_DIR="$BACKUP_BASE_DIR/daily"
WEEKLY_DIR="$BACKUP_BASE_DIR/weekly"
MONTHLY_DIR="$BACKUP_BASE_DIR/monthly"
LOG_FILE="${LOG_DIR:-./logs}/retention-policy.log"

# Retention periods (in days)
DAILY_RETENTION=30
WEEKLY_RETENTION=90
MONTHLY_RETENTION=365

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
    echo -e "${RED}[ERROR]${NC} $@"
}

################################################################################
# Retention Policy Functions
################################################################################

# Apply retention policy to a directory
apply_retention() {
    local backup_dir=$1
    local retention_days=$2
    local backup_type=$3
    
    log_info "Applying $backup_type retention policy: $retention_days days"
    
    if [ ! -d "$backup_dir" ]; then
        log_warning "Directory does not exist: $backup_dir"
        return 0
    fi
    
    local deleted_count=0
    local kept_count=0
    local total_size_deleted=0
    
    # Find and process old backups
    while IFS= read -r -d '' backup_file; do
        local file_age_days=$(( ($(date +%s) - $(stat -c %Y "$backup_file")) / 86400 ))
        local file_size=$(stat -c %s "$backup_file")
        
        if [ $file_age_days -gt $retention_days ]; then
            log_info "Deleting old backup (${file_age_days} days old): $(basename "$backup_file")"
            
            if rm -f "$backup_file"; then
                ((deleted_count++))
                ((total_size_deleted+=file_size))
                log_success "Deleted: $(basename "$backup_file")"
            else
                log_error "Failed to delete: $(basename "$backup_file")"
            fi
        else
            ((kept_count++))
        fi
    done < <(find "$backup_dir" -type f \( -name "*.sql.gz" -o -name "*.tar.gz" \) -print0)
    
    # Convert bytes to human-readable format
    local size_mb=$((total_size_deleted / 1024 / 1024))
    
    log_success "$backup_type retention complete:"
    log_success "  - Deleted: $deleted_count backups (${size_mb}MB freed)"
    log_success "  - Kept: $kept_count backups"
    
    echo ""
}

# Clean up empty directories
cleanup_empty_dirs() {
    log_info "Cleaning up empty directories..."
    
    for dir in "$DAILY_DIR" "$WEEKLY_DIR" "$MONTHLY_DIR"; do
        if [ -d "$dir" ]; then
            find "$dir" -type d -empty -delete 2>/dev/null || true
        fi
    done
    
    log_success "Empty directories cleaned up"
}

# Generate retention report
generate_report() {
    log_info "Generating retention report..."
    
    local report_file="$BACKUP_BASE_DIR/retention-report-$(date +%Y%m%d-%H%M%S).txt"
    
    {
        echo "=========================================="
        echo "Backup Retention Policy Report"
        echo "Generated: $(date '+%Y-%m-%d %H:%M:%S')"
        echo "=========================================="
        echo ""
        
        echo "Retention Policies:"
        echo "  - Daily backups: $DAILY_RETENTION days"
        echo "  - Weekly backups: $WEEKLY_RETENTION days"
        echo "  - Monthly backups: $MONTHLY_RETENTION days"
        echo ""
        
        for backup_type in "daily" "weekly" "monthly"; do
            local dir="$BACKUP_BASE_DIR/$backup_type"
            
            echo "----------------------------------------"
            echo "${backup_type^} Backups:"
            echo "----------------------------------------"
            
            if [ -d "$dir" ]; then
                local count=$(find "$dir" -type f \( -name "*.sql.gz" -o -name "*.tar.gz" \) | wc -l)
                local total_size=$(du -sh "$dir" 2>/dev/null | cut -f1)
                
                echo "  Location: $dir"
                echo "  Total backups: $count"
                echo "  Total size: $total_size"
                echo ""
                
                if [ $count -gt 0 ]; then
                    echo "  Backup files:"
                    find "$dir" -type f \( -name "*.sql.gz" -o -name "*.tar.gz" \) -printf "    - %f (%s bytes, %TY-%Tm-%Td)\n" | sort
                else
                    echo "  No backups found"
                fi
            else
                echo "  Directory does not exist"
            fi
            echo ""
        done
        
        echo "=========================================="
        echo "End of Report"
        echo "=========================================="
    } > "$report_file"
    
    log_success "Report generated: $report_file"
    
    # Display summary
    cat "$report_file"
}

# Verify backup integrity before deletion
verify_before_delete() {
    local backup_file=$1
    
    # Check if file exists and is readable
    if [ ! -r "$backup_file" ]; then
        log_error "Backup file not readable: $backup_file"
        return 1
    fi
    
    # Check if file is not empty
    if [ ! -s "$backup_file" ]; then
        log_error "Backup file is empty: $backup_file"
        return 1
    fi
    
    # Verify gzip integrity
    if [[ "$backup_file" == *.gz ]]; then
        if ! gzip -t "$backup_file" 2>/dev/null; then
            log_error "Backup file is corrupted: $backup_file"
            return 1
        fi
    fi
    
    return 0
}

################################################################################
# Main Execution
################################################################################

main() {
    log_info "=========================================="
    log_info "Starting Backup Retention Policy"
    log_info "=========================================="
    log_info "Timestamp: $(date '+%Y-%m-%d %H:%M:%S')"
    echo ""
    
    # Create log directory if it doesn't exist
    mkdir -p "$(dirname "$LOG_FILE")"
    
    # Verify backup directories exist
    for dir in "$DAILY_DIR" "$WEEKLY_DIR" "$MONTHLY_DIR"; do
        if [ ! -d "$dir" ]; then
            log_warning "Creating missing directory: $dir"
            mkdir -p "$dir"
        fi
    done
    
    # Apply retention policies
    apply_retention "$DAILY_DIR" $DAILY_RETENTION "Daily"
    apply_retention "$WEEKLY_DIR" $WEEKLY_RETENTION "Weekly"
    apply_retention "$MONTHLY_DIR" $MONTHLY_RETENTION "Monthly"
    
    # Clean up empty directories
    cleanup_empty_dirs
    
    # Generate report
    generate_report
    
    log_info "=========================================="
    log_success "Backup Retention Policy Completed"
    log_info "=========================================="
}

# Run main function
main "$@"
