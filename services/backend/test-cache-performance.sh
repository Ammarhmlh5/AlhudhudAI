#!/bin/bash
# Cache Performance Test Runner
# Tests Redis Cache performance in Routing Engine

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
GRAY='\033[0;37m'
NC='\033[0m' # No Color

# Default values
TEST="all"
SHORT=false
VERBOSE=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --test)
            TEST="$2"
            shift 2
            ;;
        --short)
            SHORT=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --test <name>    Test to run (default: all)"
            echo "  --short          Skip performance tests"
            echo "  --verbose        Verbose output"
            echo "  -h, --help       Show this help"
            echo ""
            echo "Available tests:"
            echo "  all          - Run all cache performance tests"
            echo "  hitrate      - Cache Hit Rate test (Requirement 5.7)"
            echo "  latency      - Latency Comparison test"
            echo "  dbload       - Database Load Reduction test (Requirement 5.7)"
            echo "  concurrent   - Concurrent Access test"
            echo "  ttl          - TTL Expiration test"
            echo "  multitenant  - Multi-Tenant Isolation test"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

echo -e "${CYAN}==================================${NC}"
echo -e "${CYAN}Cache Performance Test Runner${NC}"
echo -e "${CYAN}==================================${NC}"
echo ""

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    echo -e "${RED}Error: Must be run from services/backend directory${NC}"
    exit 1
fi

# Check PostgreSQL
echo -e "${YELLOW}Checking PostgreSQL...${NC}"
if docker ps --filter "name=postgres" --format "{{.Status}}" | grep -q "Up"; then
    echo -e "${GREEN}✓ PostgreSQL is running${NC}"
else
    echo -e "${RED}✗ PostgreSQL is not running${NC}"
    echo -e "${YELLOW}  Run: docker-compose up -d postgres${NC}"
    exit 1
fi

# Check Redis
echo -e "${YELLOW}Checking Redis...${NC}"
if docker ps --filter "name=redis" --format "{{.Status}}" | grep -q "Up"; then
    echo -e "${GREEN}✓ Redis is running${NC}"
else
    echo -e "${RED}✗ Redis is not running${NC}"
    echo -e "${YELLOW}  Run: docker-compose up -d redis${NC}"
    exit 1
fi

echo ""

# Build test command
TEST_CMD="go test"

if [ "$VERBOSE" = true ]; then
    TEST_CMD="$TEST_CMD -v"
fi

if [ "$SHORT" = true ]; then
    TEST_CMD="$TEST_CMD -short"
    echo -e "${YELLOW}Running in SHORT mode (skipping performance tests)${NC}"
else
    # Select test to run
    case $TEST in
        all)
            TEST_CMD="$TEST_CMD -run TestCachePerformance"
            echo -e "${CYAN}Running ALL cache performance tests...${NC}"
            ;;
        hitrate)
            TEST_CMD="$TEST_CMD -run TestCachePerformance_HitRate"
            echo -e "${CYAN}Running Cache Hit Rate test...${NC}"
            ;;
        latency)
            TEST_CMD="$TEST_CMD -run TestCachePerformance_LatencyComparison"
            echo -e "${CYAN}Running Latency Comparison test...${NC}"
            ;;
        dbload)
            TEST_CMD="$TEST_CMD -run TestCachePerformance_DatabaseLoadReduction"
            echo -e "${CYAN}Running Database Load Reduction test...${NC}"
            ;;
        concurrent)
            TEST_CMD="$TEST_CMD -run TestCachePerformance_ConcurrentAccess"
            echo -e "${CYAN}Running Concurrent Access test...${NC}"
            ;;
        ttl)
            TEST_CMD="$TEST_CMD -run TestCachePerformance_TTLExpiration"
            echo -e "${CYAN}Running TTL Expiration test...${NC}"
            ;;
        multitenant)
            TEST_CMD="$TEST_CMD -run TestCachePerformance_MultiTenantIsolation"
            echo -e "${CYAN}Running Multi-Tenant Isolation test...${NC}"
            ;;
        *)
            echo -e "${RED}Unknown test: $TEST${NC}"
            echo ""
            echo -e "${YELLOW}Available tests:${NC}"
            echo -e "${WHITE}  all          - Run all cache performance tests${NC}"
            echo -e "${WHITE}  hitrate      - Cache Hit Rate test (Requirement 5.7)${NC}"
            echo -e "${WHITE}  latency      - Latency Comparison test${NC}"
            echo -e "${WHITE}  dbload       - Database Load Reduction test (Requirement 5.7)${NC}"
            echo -e "${WHITE}  concurrent   - Concurrent Access test${NC}"
            echo -e "${WHITE}  ttl          - TTL Expiration test${NC}"
            echo -e "${WHITE}  multitenant  - Multi-Tenant Isolation test${NC}"
            exit 1
            ;;
    esac
fi

TEST_CMD="$TEST_CMD -timeout 60s ./internal/routing/"

echo ""
echo -e "${GRAY}Command: $TEST_CMD${NC}"
echo ""

# Run tests
START_TIME=$(date +%s)
set +e
eval $TEST_CMD
EXIT_CODE=$?
set -e
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

echo ""
echo -e "${CYAN}==================================${NC}"
echo -e "${CYAN}Test Results${NC}"
echo -e "${CYAN}==================================${NC}"
echo -e "${WHITE}Duration: $DURATION seconds${NC}"

if [ $EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}Status: PASSED ✓${NC}"
else
    echo -e "${RED}Status: FAILED ✗${NC}"
fi

echo ""

# Show next steps
if [ $EXIT_CODE -eq 0 ]; then
    echo -e "${YELLOW}Next Steps:${NC}"
    echo -e "${WHITE}  1. Review test output above${NC}"
    echo -e "${WHITE}  2. Check that Cache Hit Rate > 95%${NC}"
    echo -e "${WHITE}  3. Check that Database Query Reduction > 99%${NC}"
    echo -e "${WHITE}  4. Review TASK_14.7_EXECUTION_SUMMARY.md for details${NC}"
else
    echo -e "${YELLOW}Troubleshooting:${NC}"
    echo -e "${WHITE}  1. Check PostgreSQL and Redis are running${NC}"
    echo -e "${WHITE}  2. Run migrations: go run cmd/migrate/main.go up${NC}"
    echo -e "${WHITE}  3. Check logs for errors${NC}"
    echo -e "${WHITE}  4. Review CACHE_PERFORMANCE_TEST_GUIDE.md${NC}"
fi

echo ""

exit $EXIT_CODE
