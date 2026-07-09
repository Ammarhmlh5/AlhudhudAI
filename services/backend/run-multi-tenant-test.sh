#!/bin/bash
# Bash script to run Multi-Tenant Scenarios integration test
# المهمة 15.4: اختبار Multi-Tenant Scenarios

echo "🏢 Multi-Tenant Scenarios Integration Test"
echo "=========================================="
echo ""

# Check if Docker is running
echo "🔍 Checking Docker status..."
if ! docker ps > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker first."
    echo ""
    echo "To start infrastructure:"
    echo "  docker-compose up -d"
    exit 1
fi

echo "✅ Docker is running"
echo ""

# Check if PostgreSQL is running
echo "🔍 Checking PostgreSQL..."
if ! docker ps --filter "name=postgres" --filter "status=running" -q | grep -q .; then
    echo "❌ PostgreSQL is not running"
    echo ""
    echo "To start infrastructure:"
    echo "  docker-compose up -d postgres"
    exit 1
fi

echo "✅ PostgreSQL is running"
echo ""

# Check if Redis is running
echo "🔍 Checking Redis..."
if ! docker ps --filter "name=redis" --filter "status=running" -q | grep -q .; then
    echo "❌ Redis is not running"
    echo ""
    echo "To start infrastructure:"
    echo "  docker-compose up -d redis"
    exit 1
fi

echo "✅ Redis is running"
echo ""

# Run migrations
echo "🔄 Running database migrations..."
cd services/backend
go run cmd/migrate/main.go up
if [ $? -ne 0 ]; then
    echo "❌ Migration failed"
    exit 1
fi
cd ../..

echo "✅ Migrations completed"
echo ""

# Run the test
echo "🧪 Running Multi-Tenant Scenarios test..."
echo ""
echo "Test objectives:"
echo "  1. Send messages from 3+ tenants concurrently"
echo "  2. Verify complete data isolation between tenants"
echo "  3. Verify tenant operations don't affect other tenants"
echo "  4. Verify rate limiting is isolated per tenant"
echo "  5. Verify channels and messages are properly isolated"
echo "  6. Verify no data leakage between tenants"
echo ""

cd services/backend/internal/integration

# Run with verbose output
go test -v -run TestMultiTenant -timeout 5m

testResult=$?

cd ../../../..

echo ""
if [ $testResult -eq 0 ]; then
    echo "✅ Multi-Tenant Scenarios test PASSED!"
    echo ""
    echo "Verified:"
    echo "  ✓ Complete data isolation between tenants"
    echo "  ✓ No cross-tenant data leakage"
    echo "  ✓ Independent rate limiting per tenant"
    echo "  ✓ Isolated balance operations"
    echo "  ✓ Isolated channel management"
    echo ""
    echo "📋 Requirements validated: 1.1, 3.4, 5.1"
    echo ""
    echo "Next steps:"
    echo "  1. ✅ Task 15.4 completed"
    echo "  2. ➡️  Continue to Task 15.5 (WebSocket Reconnection)"
    echo "  3. ➡️  Continue to Task 15.6 (Database Failure Recovery)"
else
    echo "❌ Multi-Tenant Scenarios test FAILED!"
    echo ""
    echo "Please check the test output above for details."
    echo ""
    echo "Common issues:"
    echo "  - Database connection issues"
    echo "  - Redis connection issues"
    echo "  - Missing migrations"
    echo "  - Port conflicts"
    exit 1
fi
