#!/bin/bash
# Bash script to run WebSocket Reconnection Integration Test
# اختبار إعادة اتصال WebSocket

set -e

echo "========================================"
echo "WebSocket Reconnection Integration Test"
echo "اختبار إعادة اتصال WebSocket"
echo "========================================"
echo ""

# Check if Docker is running
echo "Checking Docker status..."
if ! docker ps > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker first."
    echo "   Run: docker-compose up -d"
    exit 1
fi
echo "✓ Docker is running"
echo ""

# Check if PostgreSQL is running
echo "Checking PostgreSQL..."
if ! docker ps --filter "name=postgres" --format "{{.Names}}" | grep -q postgres; then
    echo "❌ PostgreSQL container is not running."
    echo "   Run: docker-compose up -d postgres"
    exit 1
fi
echo "✓ PostgreSQL is running"
echo ""

# Check if Redis is running
echo "Checking Redis..."
if ! docker ps --filter "name=redis" --format "{{.Names}}" | grep -q redis; then
    echo "❌ Redis container is not running."
    echo "   Run: docker-compose up -d redis"
    exit 1
fi
echo "✓ Redis is running"
echo ""

# Wait for services to be ready
echo "Waiting for services to be ready..."
sleep 2
echo "✓ Services are ready"
echo ""

# Run the test
echo "========================================"
echo "Running WebSocket Reconnection Test..."
echo "========================================"
echo ""

cd services/backend

# Run the test with verbose output
go test -v -run TestWebSocketReconnection ./internal/integration/

testResult=$?

echo ""
echo "========================================"

if [ $testResult -eq 0 ]; then
    echo "✓ WebSocket Reconnection Test PASSED"
    echo "✓ اختبار إعادة اتصال WebSocket نجح"
else
    echo "❌ WebSocket Reconnection Test FAILED"
    echo "❌ اختبار إعادة اتصال WebSocket فشل"
fi

echo "========================================"
echo ""

# Return to original directory
cd ../..

exit $testResult
