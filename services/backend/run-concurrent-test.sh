#!/bin/bash
# Script to run concurrent operations tests
# Usage: ./run-concurrent-test.sh

echo "=================================="
echo "Concurrent Operations Test Runner"
echo "=================================="
echo ""

# Check if database is running
echo "Checking database connection..."
export PGPASSWORD=postgres
if ! psql -h localhost -U postgres -d alhudhud_test -c "SELECT 1" > /dev/null 2>&1; then
    echo "❌ Database 'alhudhud_test' not found or not accessible"
    echo ""
    echo "Creating test database..."
    psql -h localhost -U postgres -c "CREATE DATABASE alhudhud_test"
    
    if [ $? -eq 0 ]; then
        echo "✅ Test database created successfully"
    else
        echo "❌ Failed to create test database"
        echo "Please ensure PostgreSQL is running and accessible"
        exit 1
    fi
fi

echo "✅ Database connection OK"
echo ""

# Run migrations on test database
echo "Running migrations on test database..."
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/alhudhud_test?sslmode=disable"

# Apply migrations
for migration in migrations/*.sql; do
    echo "  Applying: $(basename $migration)"
    psql -h localhost -U postgres -d alhudhud_test < "$migration" > /dev/null 2>&1
done

echo "✅ Migrations applied"
echo ""

# Run the concurrent tests
echo "Running concurrent operations tests..."
echo ""

# Test 1: Basic concurrent deductions
echo "Test 1: 100 Concurrent Balance Deductions"
go test -v ./internal/balance -run TestConcurrentBalanceDeductions -timeout 5m

echo ""
echo "Test 2: Mixed Concurrent Operations"
go test -v ./internal/balance -run TestConcurrentMixedOperations -timeout 5m

echo ""
echo "Test 3: High Contention Scenario"
go test -v ./internal/balance -run TestHighContentionScenario -timeout 5m

echo ""
echo "=================================="
echo "All tests completed!"
echo "=================================="
