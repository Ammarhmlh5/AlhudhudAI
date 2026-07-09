#!/bin/bash
# Bash script to run End-to-End integration tests
# Usage: ./run-e2e-test.sh

set -e

echo "========================================"
echo "  End-to-End Integration Test Runner"
echo "========================================"
echo ""

# Check if Docker is running
echo "Checking Docker status..."
if ! docker ps > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker first."
    echo ""
    echo "To start infrastructure:"
    echo "  docker-compose up -d postgres redis"
    exit 1
fi
echo "✓ Docker is running"
echo ""

# Check if PostgreSQL is running
echo "Checking PostgreSQL..."
if ! docker ps --filter "name=postgres" --filter "status=running" -q | grep -q .; then
    echo "❌ PostgreSQL is not running."
    echo ""
    echo "To start PostgreSQL:"
    echo "  docker-compose up -d postgres"
    exit 1
fi
echo "✓ PostgreSQL is running"
echo ""

# Check if Redis is running
echo "Checking Redis..."
if ! docker ps --filter "name=redis" --filter "status=running" -q | grep -q .; then
    echo "❌ Redis is not running."
    echo ""
    echo "To start Redis:"
    echo "  docker-compose up -d redis"
    exit 1
fi
echo "✓ Redis is running"
echo ""

# Wait for services to be ready
echo "Waiting for services to be ready..."
sleep 2
echo "✓ Services ready"
echo ""

# Run migrations if needed
echo "Checking database migrations..."
echo "(Skipping - assuming migrations are already applied)"
echo ""

# Set test environment variables
export TEST_DB_HOST="localhost"
export TEST_DB_PORT="5432"
export TEST_DB_USER="postgres"
export TEST_DB_PASSWORD="postgres"
export TEST_DB_NAME="sms_gateway_test"
export TEST_REDIS_HOST="localhost"
export TEST_REDIS_PORT="6379"

echo "========================================"
echo "  Running End-to-End Tests"
echo "========================================"
echo ""

# Run the integration tests
echo "Executing: go test -v ./internal/integration/..."
echo ""

go test -v -timeout 5m ./internal/integration/...

exitCode=$?

echo ""
echo "========================================"

if [ $exitCode -eq 0 ]; then
    echo "✅ All tests passed!"
else
    echo "❌ Some tests failed!"
fi

echo "========================================"
echo ""

exit $exitCode
