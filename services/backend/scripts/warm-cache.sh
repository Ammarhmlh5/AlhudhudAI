#!/bin/bash

# Cache Warming Script
# Task 35.2: Redis Optimization - Cache Warming
# Requirements: 27.9

echo "================================"
echo "Cache Warming Script"
echo "================================"
echo ""

# Load environment variables
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

# Configuration
API_URL="${API_URL:-http://localhost:8080}"
ADMIN_TOKEN="${ADMIN_TOKEN}"

if [ -z "$ADMIN_TOKEN" ]; then
    echo "Error: ADMIN_TOKEN not set"
    echo "Please set ADMIN_TOKEN environment variable"
    exit 1
fi

echo "API URL: $API_URL"
echo ""

# Function to call API
call_api() {
    local endpoint=$1
    local description=$2
    
    echo "Warming: $description"
    
    response=$(curl -s -w "\n%{http_code}" -X POST "$API_URL$endpoint" \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -H "Content-Type: application/json")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" -eq 200 ] || [ "$http_code" -eq 201 ]; then
        echo "✓ $description warmed successfully"
    else
        echo "✗ Failed to warm $description (HTTP $http_code)"
        echo "Response: $body"
    fi
    
    echo ""
}

# Warm cache for different data types
echo "Starting cache warming..."
echo ""

# 1. Warm tariffs cache
call_api "/admin/cache/warm/tariffs" "Active Tariffs"

# 2. Warm packages cache
call_api "/admin/cache/warm/packages" "Active Packages"

# 3. Warm services cache
call_api "/admin/cache/warm/services" "Active Services"

# 4. Warm segmentation rules cache
call_api "/admin/cache/warm/segmentation-rules" "Segmentation Rules"

# 5. Warm exchange rates cache
call_api "/admin/cache/warm/exchange-rates" "Exchange Rates"

# 6. Warm all caches
call_api "/admin/cache/warm/all" "All Caches"

echo "================================"
echo "Cache Warming Complete"
echo "================================"
