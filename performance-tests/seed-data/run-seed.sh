#!/bin/bash

# Performance Test Seed Data Runner
# This script runs all seed data scripts to prepare the database for performance testing

set -e  # Exit on error

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Database connection parameters
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-sms_gateway}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-postgres}"

echo -e "${YELLOW}=== Performance Test Seed Data Runner ===${NC}"
echo ""
echo "Database: $DB_NAME@$DB_HOST:$DB_PORT"
echo "User: $DB_USER"
echo ""

# Function to run SQL file
run_sql_file() {
    local file=$1
    local description=$2
    
    echo -e "${YELLOW}Running: $description${NC}"
    echo "File: $file"
    
    PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f $file
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Success${NC}"
    else
        echo -e "${RED}✗ Failed${NC}"
        exit 1
    fi
    echo ""
}

# Run seed scripts in order
run_sql_file "seed-tenants.sql" "Creating test tenants"
run_sql_file "seed-channels.sql" "Creating test channels"

echo -e "${GREEN}=== All seed data created successfully! ===${NC}"
echo ""
echo "Summary:"
echo "- 10 test tenants created with 100,000 balance each"
echo "- 50 test channels created (5 per tenant)"
echo ""
echo "You can now run performance tests!"
