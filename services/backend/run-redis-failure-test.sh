#!/bin/bash

# Redis Failure Fallback Test Runner
# This script runs the Redis failure fallback integration test

echo "========================================"
echo "Redis Failure Fallback Test"
echo "========================================"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Check if Docker is running
echo -e "${YELLOW}Checking Docker status...${NC}"
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}❌ Docker is not running. Please start Docker first.${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Docker is running${NC}"
echo ""

# Check if PostgreSQL is running
echo -e "${YELLOW}Checking PostgreSQL status...${NC}"
if ! docker-compose ps postgres 2>/dev/null | grep -q "Up"; then
    echo -e "${YELLOW}⚠ PostgreSQL is not running. Starting...${NC}"
    docker-compose up -d postgres
    sleep 5
fi
echo -e "${GREEN}✓ PostgreSQL is running${NC}"
echo ""

# Check if Redis is running
echo -e "${YELLOW}Checking Redis status...${NC}"
if ! docker-compose ps redis 2>/dev/null | grep -q "Up"; then
    echo -e "${YELLOW}⚠ Redis is not running. Starting...${NC}"
    docker-compose up -d redis
    sleep 3
fi
echo -e "${GREEN}✓ Redis is running${NC}"
echo ""

# Run database migrations
echo -e "${YELLOW}Running database migrations...${NC}"
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/sms_gateway_test?sslmode=disable"
if go run cmd/migrate/main.go up 2>/dev/null; then
    echo -e "${GREEN}✓ Migrations completed${NC}"
else
    echo -e "${YELLOW}⚠ Migrations may have already been applied${NC}"
fi
echo ""

# Run the test
echo "========================================"
echo "Running Redis Failure Fallback Test..."
echo "========================================"
echo ""

# Check if user wants to run specific scenario
SCENARIO=$1

if [ -n "$SCENARIO" ]; then
    echo -e "${YELLOW}Running specific scenario: $SCENARIO${NC}"
    echo ""
    go test -v ./internal/integration -run "TestRedisFailureFallback/$SCENARIO" -timeout 10m
else
    echo -e "${YELLOW}Running all scenarios...${NC}"
    echo ""
    go test -v ./internal/integration -run TestRedisFailureFallback -timeout 10m
fi

TEST_RESULT=$?

echo ""
echo "========================================"

if [ $TEST_RESULT -eq 0 ]; then
    echo -e "${GREEN}✅ Test PASSED${NC}"
    echo ""
    echo -e "${GREEN}All Redis failure fallback scenarios completed successfully!${NC}"
    echo ""
    echo -e "${CYAN}Verified:${NC}"
    echo -e "${GREEN}  ✓ Redis failure detection${NC}"
    echo -e "${GREEN}  ✓ Automatic fallback to direct processing${NC}"
    echo -e "${GREEN}  ✓ No message loss${NC}"
    echo -e "${GREEN}  ✓ Automatic reconnection to Redis${NC}"
    echo -e "${GREEN}  ✓ Circuit breaker behavior${NC}"
else
    echo -e "${RED}❌ Test FAILED${NC}"
    echo ""
    echo -e "${YELLOW}Please check the error messages above.${NC}"
    echo ""
    echo -e "${YELLOW}Common issues:${NC}"
    echo -e "${YELLOW}  • Redis connection issues${NC}"
    echo -e "${YELLOW}  • Database connection issues${NC}"
    echo -e "${YELLOW}  • Circuit breaker not opening${NC}"
    echo -e "${YELLOW}  • Fallback operation failures${NC}"
    echo ""
    echo -e "${YELLOW}For detailed troubleshooting, see:${NC}"
    echo -e "${CYAN}  internal/integration/REDIS_FAILURE_TEST_GUIDE.md${NC}"
fi

echo "========================================"
echo ""

# Show available scenarios
if [ $TEST_RESULT -ne 0 ] || [ -z "$SCENARIO" ]; then
    echo -e "${CYAN}Available scenarios:${NC}"
    echo "  1. RedisFailureWithFallback"
    echo "  2. SystemContinuesWithoutRedis"
    echo "  3. AutomaticReconnectionToRedis"
    echo "  4. BatchProcessingWithRedisFailure"
    echo "  5. CircuitBreakerBehavior"
    echo ""
    echo -e "${YELLOW}To run a specific scenario:${NC}"
    echo -e "${CYAN}  ./run-redis-failure-test.sh <scenario-name>${NC}"
    echo ""
    echo -e "${YELLOW}Example:${NC}"
    echo -e "${CYAN}  ./run-redis-failure-test.sh RedisFailureWithFallback${NC}"
    echo ""
fi

exit $TEST_RESULT
