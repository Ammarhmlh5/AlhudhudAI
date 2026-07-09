#!/bin/bash

# Apache Bench Test - Concurrent Requests
# Tests system behavior under high concurrency

set -e

# Configuration
API_URL="${API_URL:-http://localhost:8080/api/messages}"
REQUESTS=5000
CONCURRENCY=100
TENANT_ID=1

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${YELLOW}=== Concurrent Requests Test (Apache Bench) ===${NC}"
echo ""
echo "Configuration:"
echo "  URL: $API_URL"
echo "  Total Requests: $REQUESTS"
echo "  Concurrency: $CONCURRENCY"
echo ""

# Create test payload
PAYLOAD=$(cat <<EOF
{
  "tenant_id": $TENANT_ID,
  "phone_number": "+966500000001",
  "content": "Concurrent test message",
  "priority": "NORMAL"
}
EOF
)

# Save payload to temp file
TEMP_FILE=$(mktemp)
echo "$PAYLOAD" > "$TEMP_FILE"

echo -e "${YELLOW}Running test...${NC}"
echo ""

# Run Apache Bench
ab -n $REQUESTS \
   -c $CONCURRENCY \
   -p "$TEMP_FILE" \
   -T "application/json" \
   -g "results/ab-concurrent.tsv" \
   "$API_URL" | tee "results/ab-concurrent.txt"

# Clean up
rm "$TEMP_FILE"

echo ""
echo -e "${GREEN}Test completed!${NC}"
echo "Results saved to:"
echo "  - results/ab-concurrent.txt"
echo "  - results/ab-concurrent.tsv"
echo ""

# Extract and analyze key metrics
echo -e "${YELLOW}Performance Analysis:${NC}"
echo ""

# Extract metrics
TIME_PER_REQ=$(grep "Time per request" "results/ab-concurrent.txt" | head -1 | awk '{print $4}')
REQ_PER_SEC=$(grep "Requests per second" "results/ab-concurrent.txt" | awk '{print $4}')
FAILED=$(grep "Failed requests" "results/ab-concurrent.txt" | awk '{print $3}')

echo "Requests per second: $REQ_PER_SEC"
echo "Time per request: ${TIME_PER_REQ}ms"
echo "Failed requests: $FAILED"
echo ""

# Check if meets requirements
if (( $(echo "$TIME_PER_REQ < 10" | bc -l) )); then
    echo -e "${GREEN}✓ Response time < 10ms: PASS${NC}"
else
    echo -e "${RED}✗ Response time >= 10ms: FAIL${NC}"
fi

if [ "$FAILED" -eq 0 ]; then
    echo -e "${GREEN}✓ No failed requests: PASS${NC}"
else
    echo -e "${RED}✗ Failed requests detected: FAIL${NC}"
fi
