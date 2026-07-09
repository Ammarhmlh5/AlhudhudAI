#!/bin/bash

################################################################################
# Restore Validation Script
# 
# This script validates a restore operation by checking data integrity,
# referential integrity, and data consistency.
#
# Requirements: 25.8, 25.9
################################################################################

set -e

# Configuration
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-sms_gateway}"
DB_USER="${DB_USER:-postgres}"
PGPASSWORD="${DB_PASSWORD:-}"
export PGPASSWORD

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counters
TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0
WARNING_CHECKS=0

################################################################################
# Helper Functions
################################################################################

check_pass() {
    echo -e "${GREEN}✓${NC} $1"
    ((PASSED_CHECKS++))
    ((TOTAL_CHECKS++))
}

check_fail() {
    echo -e "${RED}✗${NC} $1"
    ((FAILED_CHECKS++))
    ((TOTAL_CHECKS++))
}

check_warn() {
    echo -e "${YELLOW}⚠${NC} $1"
    ((WARNING_CHECKS++))
    ((TOTAL_CHECKS++))
}

run_query() {
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
        -t -c "$1" 2>/dev/null || echo "ERROR"
}

################################################################################
# Validation Checks
################################################################################

echo "=========================================="
echo "Tariff System Restore Validation"
echo "=========================================="
echo ""

# Check 1: Database connectivity
echo "1. Checking database connectivity..."
if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1;" > /dev/null 2>&1; then
    check_pass "Database is accessible"
else
    check_fail "Cannot connect to database"
    exit 1
fi

# Check 2: Table existence
echo ""
echo "2. Checking table existence..."
REQUIRED_TABLES=(
    "services"
    "tariffs"
    "packages"
    "subscriptions"
    "rollover_messages"
    "dedicated_gateways"
    "accounting_transactions"
    "balance_history"
)

for table in "${REQUIRED_TABLES[@]}"; do
    result=$(run_query "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = '$table';")
    result=$(echo "$result" | tr -d ' ')
    
    if [ "$result" = "1" ]; then
        check_pass "Table '$table' exists"
    else
        check_fail "Table '$table' is missing"
    fi
done


# Check 3: Row counts
echo ""
echo "3. Checking table row counts..."
for table in "${REQUIRED_TABLES[@]}"; do
    count=$(run_query "SELECT COUNT(*) FROM $table;")
    count=$(echo "$count" | tr -d ' ')
    
    if [ "$count" = "ERROR" ]; then
        check_fail "Cannot query table '$table'"
    elif [ "$count" -eq 0 ]; then
        check_warn "Table '$table' has 0 rows"
    else
        check_pass "Table '$table' has $count rows"
    fi
done

# Check 4: Referential integrity
echo ""
echo "4. Checking referential integrity..."

# Check subscriptions -> packages
orphaned=$(run_query "SELECT COUNT(*) FROM subscriptions s LEFT JOIN packages p ON s.package_id = p.id WHERE p.id IS NULL;")
orphaned=$(echo "$orphaned" | tr -d ' ')
if [ "$orphaned" -eq 0 ]; then
    check_pass "No orphaned subscriptions"
else
    check_fail "Found $orphaned orphaned subscriptions"
fi

# Check tariffs -> services
orphaned=$(run_query "SELECT COUNT(*) FROM tariffs t LEFT JOIN services s ON t.service_id = s.id WHERE s.id IS NULL;")
orphaned=$(echo "$orphaned" | tr -d ' ')
if [ "$orphaned" -eq 0 ]; then
    check_pass "No orphaned tariffs"
else
    check_fail "Found $orphaned orphaned tariffs"
fi

# Check rollover_messages -> subscriptions
orphaned=$(run_query "SELECT COUNT(*) FROM rollover_messages r LEFT JOIN subscriptions s ON r.subscription_id = s.id WHERE s.id IS NULL;")
orphaned=$(echo "$orphaned" | tr -d ' ')
if [ "$orphaned" -eq 0 ]; then
    check_pass "No orphaned rollover messages"
else
    check_fail "Found $orphaned orphaned rollover messages"
fi

# Check 5: Data consistency
echo ""
echo "5. Checking data consistency..."

# Check for negative costs
invalid=$(run_query "SELECT COUNT(*) FROM tariffs WHERE cost_per_message < 0;")
invalid=$(echo "$invalid" | tr -d ' ')
if [ "$invalid" -eq 0 ]; then
    check_pass "No tariffs with negative costs"
else
    check_fail "Found $invalid tariffs with negative costs"
fi

# Check for invalid subscription dates
invalid=$(run_query "SELECT COUNT(*) FROM subscriptions WHERE end_date < start_date;")
invalid=$(echo "$invalid" | tr -d ' ')
if [ "$invalid" -eq 0 ]; then
    check_pass "No subscriptions with invalid dates"
else
    check_fail "Found $invalid subscriptions with invalid dates"
fi

# Check for invalid package prices
invalid=$(run_query "SELECT COUNT(*) FROM packages WHERE monthly_price < 0;")
invalid=$(echo "$invalid" | tr -d ' ')
if [ "$invalid" -eq 0 ]; then
    check_pass "No packages with negative prices"
else
    check_fail "Found $invalid packages with negative prices"
fi

# Check for invalid credit transactions
invalid=$(run_query "SELECT COUNT(*) FROM accounting_transactions WHERE amount < 0 AND transaction_type = 'CREDIT';")
invalid=$(echo "$invalid" | tr -d ' ')
if [ "$invalid" -eq 0 ]; then
    check_pass "No invalid credit transactions"
else
    check_fail "Found $invalid invalid credit transactions"
fi


# Check 6: Index integrity
echo ""
echo "6. Checking index integrity..."

# Check for missing indexes on critical tables
CRITICAL_INDEXES=(
    "idx_tariffs_tenant_service"
    "idx_subscriptions_tenant_status"
    "idx_accounting_tenant_created"
)

for index in "${CRITICAL_INDEXES[@]}"; do
    exists=$(run_query "SELECT COUNT(*) FROM pg_indexes WHERE indexname = '$index';")
    exists=$(echo "$exists" | tr -d ' ')
    
    if [ "$exists" -eq 1 ]; then
        check_pass "Index '$index' exists"
    else
        check_warn "Index '$index' is missing"
    fi
done

# Check 7: Constraint integrity
echo ""
echo "7. Checking constraint integrity..."

# Check primary keys
tables_with_pk=$(run_query "SELECT COUNT(DISTINCT table_name) FROM information_schema.table_constraints WHERE constraint_type = 'PRIMARY KEY' AND table_name IN ('tariffs', 'packages', 'subscriptions');")
tables_with_pk=$(echo "$tables_with_pk" | tr -d ' ')

if [ "$tables_with_pk" -eq 3 ]; then
    check_pass "All critical tables have primary keys"
else
    check_fail "Some critical tables are missing primary keys"
fi

# Check 8: Data timestamps
echo ""
echo "8. Checking data timestamps..."

# Check latest transaction timestamp
latest=$(run_query "SELECT MAX(created_at) FROM accounting_transactions;")
if [ "$latest" != "ERROR" ] && [ -n "$latest" ]; then
    check_pass "Latest transaction: $latest"
else
    check_warn "No transaction timestamps found"
fi

# Check latest subscription timestamp
latest=$(run_query "SELECT MAX(created_at) FROM subscriptions;")
if [ "$latest" != "ERROR" ] && [ -n "$latest" ]; then
    check_pass "Latest subscription: $latest"
else
    check_warn "No subscription timestamps found"
fi

################################################################################
# Summary
################################################################################

echo ""
echo "=========================================="
echo "Validation Summary"
echo "=========================================="
echo "Total Checks: $TOTAL_CHECKS"
echo -e "${GREEN}Passed: $PASSED_CHECKS${NC}"
echo -e "${YELLOW}Warnings: $WARNING_CHECKS${NC}"
echo -e "${RED}Failed: $FAILED_CHECKS${NC}"
echo "=========================================="

if [ $FAILED_CHECKS -eq 0 ]; then
    if [ $WARNING_CHECKS -eq 0 ]; then
        echo -e "${GREEN}✓ All validation checks passed${NC}"
        exit 0
    else
        echo -e "${YELLOW}⚠ Validation passed with warnings${NC}"
        exit 0
    fi
else
    echo -e "${RED}✗ Validation failed${NC}"
    exit 1
fi
