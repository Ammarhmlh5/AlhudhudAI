#!/bin/bash

# Bulk Message Processing Test Runner
# Tests processing 10,000 messages in bulk

set -e

echo "🚀 Bulk Message Processing Test"
echo "================================"
echo ""

# Check if infrastructure is running
echo "📋 Checking infrastructure..."
if ! docker ps | grep -q postgres; then
    echo "❌ PostgreSQL is not running. Please start infrastructure first:"
    echo "   ./start-infrastructure.sh"
    exit 1
fi

if ! docker ps | grep -q redis; then
    echo "❌ Redis is not running. Please start infrastructure first:"
    echo "   ./start-infrastructure.sh"
    exit 1
fi

echo "✅ Infrastructure is running"
echo ""

# Run the test
echo "🧪 Running bulk message processing test..."
echo "   This will send and process 10,000 messages"
echo "   Expected duration: ~2 minutes"
echo ""

cd services/backend

go test -v -timeout 10m \
    -run TestBulkMessageProcessing \
    ./internal/integration/

echo ""
echo "✅ Test completed!"
