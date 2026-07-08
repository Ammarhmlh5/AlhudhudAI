#!/bin/bash

# Run All Performance Tests
# This script runs all performance tests in sequence

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
API_URL="${API_URL:-http://localhost:8080}"
export API_URL

echo -e "${BLUE}"
echo "╔════════════════════════════════════════════════════════╗"
echo "║     SMS Gateway Performance Test Suite                ║"
echo "╚════════════════════════════════════════════════════════╝"
echo -e "${NC}"
echo ""
echo "API URL: $API_URL"
echo "Start Time: $(date)"
echo ""

# Create results directory
mkdir -p results

# Function to run test
run_test() {
    local test_name=$1
    local test_command=$2
    
    echo ""
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}Running: $test_name${NC}"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    
    if eval "$test_command"; then
        echo -e "${GREEN}✓ $test_name completed successfully${NC}"
        return 0
    else
        echo -e "${RED}✗ $test_name failed${NC}"
        return 1
    fi
}

# Track results
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Test 1: API Gateway Response Time
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "API Gateway Response Time Test" "k6 run k6-tests/api-gateway-test.js"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

# Test 2: Batch Processing
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "Batch Processing Test" "k6 run k6-tests/batch-processing-test.js"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

# Test 3: Worker Throughput
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "Worker Throughput Test (5000+ msg/min)" "k6 run k6-tests/worker-throughput-test.js"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

# Test 4: Stress Test
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "Stress Test (10,000 msg/min)" "k6 run k6-tests/stress-test.js"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

# Final Summary
echo ""
echo -e "${BLUE}"
echo "╔════════════════════════════════════════════════════════╗"
echo "║              Test Suite Summary                        ║"
echo "╚════════════════════════════════════════════════════════╝"
echo -e "${NC}"
echo ""
echo "End Time: $(date)"
echo ""
echo "Results:"
echo "  Total Tests: $TOTAL_TESTS"
echo -e "  ${GREEN}Passed: $PASSED_TESTS${NC}"
echo -e "  ${RED}Failed: $FAILED_TESTS${NC}"
echo ""

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}✓ All tests passed!${NC}"
    echo ""
    echo "System Performance Summary:"
    echo "  ✓ API Response Time < 10ms"
    echo "  ✓ Batch Processing optimized"
    echo "  ✓ Throughput >= 5000 messages/minute"
    echo "  ✓ System stable under stress (10,000 msg/min)"
    exit 0
else
    echo -e "${RED}✗ Some tests failed. Please review the results.${NC}"
    echo ""
    echo "Check the following files for details:"
    echo "  - results/api-gateway-test-summary.json"
    echo "  - results/batch-processing-test-summary.json"
    echo "  - results/worker-throughput-test-summary.json"
    echo "  - results/stress-test-summary.json"
    exit 1
fi
