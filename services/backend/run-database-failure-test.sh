#!/bin/bash
# Database Failure Recovery Test Runner
# This script runs the database failure recovery integration test

set -e

echo "========================================"
echo "Database Failure Recovery Test"
echo "========================================"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Check if Docker is running
echo -e "${YELLOW}Checking Docker...${NC}"
if ! docker ps > /dev/null 2>&1; then
    echo -e "${RED}❌ Docker is not running. Please start Docker first.${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Docker is running${NC}"
echo ""

# Check if PostgreSQL is running
echo -e "${YELLOW}Checking PostgreSQL...${NC}"
if ! docker ps --filter "name=postgres" --filter "status=running" --format "{{.Names}}" | grep -q "postgres"; then
    echo -e "${YELLOW}❌ PostgreSQL is not running. Starting PostgreSQL...${NC}"
    docker-compose up -d postgres
    echo -e "${YELLOW}Waiting for PostgreSQL to be ready...${NC}"
    sleep 5
fi
echo -e "${GREEN}✓ PostgreSQL is running${NC}"
echo ""

# Check if database exists
echo -e "${YELLOW}Checking database...${NC}"
if ! docker exec postgres psql -U postgres -lqt | grep -q "sms_gateway_test"; then
    echo -e "${YELLOW}Creating test database...${NC}"
    docker exec postgres psql -U postgres -c "CREATE DATABASE sms_gateway_test;"
fi
echo -e "${GREEN}✓ Database exists${NC}"
echo ""

# Run migrations
echo -e "${YELLOW}Running migrations...${NC}"
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/sms_gateway_test?sslmode=disable"
if [ -d "migrations" ]; then
    # Check if migrate tool is available
    if command -v migrate &> /dev/null; then
        migrate -path migrations -database "$DATABASE_URL" up
        echo -e "${GREEN}✓ Migrations applied${NC}"
    else
        echo -e "${YELLOW}⚠ migrate tool not found. Skipping migrations.${NC}"
        echo -e "${YELLOW}  Install: https://github.com/golang-migrate/migrate${NC}"
    fi
else
    echo -e "${YELLOW}⚠ migrations folder not found. Skipping migrations.${NC}"
fi
echo ""

# Run the test
echo -e "${YELLOW}Running Database Failure Recovery Test...${NC}"
echo "========================================"
echo ""

go test -v ./internal/integration -run TestDatabaseFailureRecovery -timeout 5m

TEST_RESULT=$?

echo ""
echo "========================================"

if [ $TEST_RESULT -eq 0 ]; then
    echo -e "${GREEN}✅ Test PASSED${NC}"
    echo ""
    echo -e "${GREEN}All database failure recovery scenarios passed successfully!${NC}"
    echo ""
    echo -e "${CYAN}Validated:${NC}"
    echo -e "${GREEN}  ✓ Primary database failure detection${NC}"
    echo -e "${GREEN}  ✓ Automatic failover to backup database${NC}"
    echo -e "${GREEN}  ✓ System continues operation without data loss${NC}"
    echo -e "${GREEN}  ✓ Automatic reconnection to primary when available${NC}"
    echo -e "${GREEN}  ✓ Connection pool behavior during failure${NC}"
else
    echo -e "${RED}❌ Test FAILED${NC}"
    echo ""
    echo -e "${YELLOW}Please check the error messages above.${NC}"
    echo ""
    echo -e "${YELLOW}Common issues:${NC}"
    echo -e "${YELLOW}  1. PostgreSQL not running: docker-compose up -d postgres${NC}"
    echo -e "${YELLOW}  2. Database not created: docker exec postgres psql -U postgres -c 'CREATE DATABASE sms_gateway_test;'${NC}"
    echo -e "${YELLOW}  3. Migrations not applied: Run migrations first${NC}"
    echo ""
    echo -e "${YELLOW}For more help, see:${NC}"
    echo -e "${CYAN}  - internal/integration/DATABASE_FAILURE_TEST_QUICK_START.md${NC}"
    echo -e "${CYAN}  - internal/integration/DATABASE_FAILURE_TEST_GUIDE.md${NC}"
fi

echo "========================================"
echo ""

exit $TEST_RESULT
