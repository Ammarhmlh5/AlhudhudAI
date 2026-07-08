#!/bin/bash

# Apache Bench Test - API Response Time
# Tests API Gateway response time with simple load

set -e

# Configuration
API_URL="${API_URL:-http://localhost:8080/api/messages}"
REQUESTS=1000
CONCURRENCY=10
TENANT_ID=1

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${YELLOW}=== API Response Time Test (Apache Bench) ===${NC}"
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
  "content": "Apache Bench test message",
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
   -g "results/ab-response-time.tsv" \
   "$API_URL" | tee "results/ab-response-time.txt"

# Clean up
rm "$TEMP_FILE"

echo ""
echo -e "${GREEN}Test completed!${NC}"
echo "Results saved to:"
echo "  - results/ab-response-time.txt"
echo "  - results/ab-response-time.tsv"
echo ""

# Extract key metrics
echo -e "${YELLOW}Key Metrics:${NC}"
grep "Time per request" "results/ab-response-time.txt" || true
grep "Requests per second" "results/ab-response-time.txt" || true
grep "Failed requests" "results/ab-response-time.txt" || true
