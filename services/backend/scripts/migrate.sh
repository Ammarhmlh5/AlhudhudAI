#!/bin/bash

# Database Migration Management Script
# Usage: ./migrate.sh [command] [options]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
MIGRATIONS_DIR="./migrations"
ROLLBACK_DIR="./migrations/rollback"
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-hudhud_sms}"
DB_USER="${DB_USER:-hudhud}"

# Functions
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_info() {
    echo -e "$1"
}

# Check if psql is available
check_psql() {
    if ! command -v psql &> /dev/null; then
        print_error "psql command not found. Please install PostgreSQL client."
        exit 1
    fi
}

# Check database connection
check_connection() {
    if ! PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c '\q' 2>/dev/null; then
        print_error "Cannot connect to database. Please check your credentials."
        exit 1
    fi
    print_success "Database connection successful"
}

# Create migration log table
create_migration_log() {
    PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME <<EOF
CREATE TABLE IF NOT EXISTS migration_log (
    id SERIAL PRIMARY KEY,
    migration_name VARCHAR(255) NOT NULL,
    action VARCHAR(50) NOT NULL,
    executed_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(migration_name, action)
);
EOF
    print_success "Migration log table created"
}

# Get current migration version
get_current_version() {
    PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c \
        "SELECT COALESCE(MAX(CAST(SUBSTRING(migration_name FROM '^[0-9]+') AS INTEGER)), 0) 
         FROM migration_log 
         WHERE action = 'apply';" | tr -d ' '
}

# List all migrations
list_migrations() {
    print_info "\nAvailable migrations:"
    for file in $MIGRATIONS_DIR/[0-9]*.sql; do
        if [ -f "$file" ]; then
            basename "$file"
        fi
    done
}

# List applied migrations
list_applied() {
    print_info "\nApplied migrations:"
    PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c \
        "SELECT migration_name, executed_at FROM migration_log WHERE action = 'apply' ORDER BY executed_at;"
}

# Apply migration
apply_migration() {
    local file=$1
    local migration_name=$(basename "$file" .sql)
    
    print_info "Applying migration: $migration_name"
    
    if PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f "$file"; then
        print_success "Migration applied: $migration_name"
        return 0
    else
        print_error "Failed to apply migration: $migration_name"
        return 1
    fi
}

# Rollback migration
rollback_migration() {
    local migration_name=$1
    local rollback_file="$ROLLBACK_DIR/${migration_name}_down.sql"
    
    if [ ! -f "$rollback_file" ]; then
        print_error "Rollback file not found: $rollback_file"
        return 1
    fi
    
    print_info "Rolling back migration: $migration_name"
    
    if PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f "$rollback_file"; then
        print_success "Migration rolled back: $migration_name"
        return 0
    else
        print_error "Failed to rollback migration: $migration_name"
        return 1
    fi
}

# Migrate up
migrate_up() {
    local target_version=$1
    local current_version=$(get_current_version)
    
    print_info "Current version: $current_version"
    
    if [ -n "$target_version" ]; then
        print_info "Target version: $target_version"
    else
        print_info "Migrating to latest version"
    fi
    
    for file in $MIGRATIONS_DIR/[0-9]*.sql; do
        if [ -f "$file" ]; then
            local file_version=$(basename "$file" | cut -d'_' -f1)
            
            if [ $file_version -gt $current_version ]; then
                if [ -n "$target_version" ] && [ $file_version -gt $target_version ]; then
                    break
                fi
                
                apply_migration "$file" || exit 1
            fi
        fi
    done
    
    print_success "Migration completed"
}

# Migrate down
migrate_down() {
    local steps=${1:-1}
    local current_version=$(get_current_version)
    
    print_info "Current version: $current_version"
    print_info "Rolling back $steps migration(s)"
    
    local migrations=$(PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c \
        "SELECT migration_name FROM migration_log WHERE action = 'apply' ORDER BY executed_at DESC LIMIT $steps;")
    
    for migration in $migrations; do
        rollback_migration "$migration" || exit 1
    done
    
    print_success "Rollback completed"
}

# Migrate to specific version
migrate_goto() {
    local target_version=$1
    local current_version=$(get_current_version)
    
    if [ -z "$target_version" ]; then
        print_error "Target version required"
        exit 1
    fi
    
    print_info "Current version: $current_version"
    print_info "Target version: $target_version"
    
    if [ $target_version -gt $current_version ]; then
        migrate_up $target_version
    elif [ $target_version -lt $current_version ]; then
        # Calculate number of steps to rollback
        local steps=$((current_version - target_version))
        migrate_down $steps
    else
        print_info "Already at version $target_version"
    fi
}

# Show migration status
show_status() {
    local current_version=$(get_current_version)
    local total_migrations=$(ls -1 $MIGRATIONS_DIR/[0-9]*.sql 2>/dev/null | wc -l)
    
    print_info "\n=== Migration Status ==="
    print_info "Current version: $current_version"
    print_info "Total migrations: $total_migrations"
    print_info "Pending migrations: $((total_migrations - current_version))"
    
    list_applied
}

# Create new migration
create_migration() {
    local name=$1
    
    if [ -z "$name" ]; then
        print_error "Migration name required"
        exit 1
    fi
    
    local next_version=$(printf "%03d" $(($(get_current_version) + 1)))
    local filename="${next_version}_${name}.sql"
    local filepath="$MIGRATIONS_DIR/$filename"
    
    cat > "$filepath" <<EOF
-- Migration: $filename
-- Created: $(date)

BEGIN;

-- Your migration SQL here

-- Log migration
INSERT INTO migration_log (migration_name, action, executed_at)
VALUES ('${next_version}_${name}', 'apply', NOW())
ON CONFLICT DO NOTHING;

COMMIT;
EOF
    
    print_success "Migration created: $filepath"
    
    # Create rollback script
    local rollback_filepath="$ROLLBACK_DIR/${next_version}_${name}_down.sql"
    
    cat > "$rollback_filepath" <<EOF
-- Rollback: $filename
-- Created: $(date)

BEGIN;

-- Your rollback SQL here

-- Log rollback
INSERT INTO migration_log (migration_name, action, executed_at)
VALUES ('${next_version}_${name}', 'rollback', NOW())
ON CONFLICT DO NOTHING;

COMMIT;
EOF
    
    print_success "Rollback script created: $rollback_filepath"
}

# Show help
show_help() {
    cat <<EOF
Database Migration Management Script

Usage: ./migrate.sh [command] [options]

Commands:
    up [version]        Apply pending migrations (optionally up to specific version)
    down [steps]        Rollback migrations (default: 1 step)
    goto <version>      Migrate to specific version
    status              Show current migration status
    list                List all available migrations
    applied             List applied migrations
    create <name>       Create new migration file
    help                Show this help message

Examples:
    ./migrate.sh up                 # Apply all pending migrations
    ./migrate.sh up 045             # Apply migrations up to version 045
    ./migrate.sh down               # Rollback last migration
    ./migrate.sh down 3             # Rollback last 3 migrations
    ./migrate.sh goto 032           # Migrate to version 032
    ./migrate.sh status             # Show migration status
    ./migrate.sh create add_users   # Create new migration

Environment Variables:
    DB_HOST             Database host (default: localhost)
    DB_PORT             Database port (default: 5432)
    DB_NAME             Database name (default: hudhud_sms)
    DB_USER             Database user (default: hudhud)
    DB_PASSWORD         Database password (required)

EOF
}

# Main
main() {
    local command=${1:-help}
    
    case $command in
        up)
            check_psql
            check_connection
            create_migration_log
            migrate_up $2
            ;;
        down)
            check_psql
            check_connection
            migrate_down $2
            ;;
        goto)
            check_psql
            check_connection
            migrate_goto $2
            ;;
        status)
            check_psql
            check_connection
            show_status
            ;;
        list)
            list_migrations
            ;;
        applied)
            check_psql
            check_connection
            list_applied
            ;;
        create)
            create_migration $2
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            print_error "Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

main "$@"
