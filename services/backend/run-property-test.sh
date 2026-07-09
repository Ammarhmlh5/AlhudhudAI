#!/bin/bash
# Property Test Runner for Balance Manager
# تشغيل Property-Based Tests للـ Balance Manager

set -e

echo "========================================"
echo "Property-Based Tests - Balance Manager"
echo "========================================"
echo ""

# التحقق من وجود Go
if ! command -v go &> /dev/null; then
    echo "❌ Error: Go is not installed or not in PATH"
    exit 1
fi

echo "✓ Go found: $(go version)"
echo ""

# الانتقال إلى مجلد backend
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "📂 Working directory: $(pwd)"
echo ""

# عرض الخيارات
echo "Available test modes:"
echo "  1. Quick test (default)"
echo "  2. With race detector"
echo "  3. Multiple iterations (count=20)"
echo "  4. Stress test (count=50)"
echo "  5. Specific test"
echo ""

# قراءة الاختيار
read -p "Select mode (1-5, default=1): " choice
choice=${choice:-1}

echo ""
echo "🧪 Running Property Tests..."
echo ""

case $choice in
    1)
        echo "Mode: Quick test"
        go test -v ./internal/balance -run TestProperty
        ;;
    2)
        echo "Mode: With race detector"
        go test -v -race ./internal/balance -run TestProperty
        ;;
    3)
        echo "Mode: Multiple iterations (20x)"
        go test -v ./internal/balance -run TestProperty -count=20
        ;;
    4)
        echo "Mode: Stress test (50x)"
        go test -v -race ./internal/balance -run TestProperty -count=50
        ;;
    5)
        echo "Available tests:"
        echo "  a. TestProperty_AtomicBalanceDeductionCorrectness"
        echo "  b. TestProperty_NoRaceConditionsInBalanceOperations"
        echo "  c. TestProperty_BalanceOperationIdempotency"
        echo "  d. TestProperty_BalanceMonotonicity"
        echo ""
        read -p "Select test (a-d): " test_choice
        
        case $test_choice in
            a) test_name="TestProperty_AtomicBalanceDeductionCorrectness" ;;
            b) test_name="TestProperty_NoRaceConditionsInBalanceOperations" ;;
            c) test_name="TestProperty_BalanceOperationIdempotency" ;;
            d) test_name="TestProperty_BalanceMonotonicity" ;;
            *) test_name="TestProperty" ;;
        esac
        
        echo "Running: $test_name"
        go test -v ./internal/balance -run "$test_name"
        ;;
    *)
        echo "Invalid choice, running quick test"
        go test -v ./internal/balance -run TestProperty
        ;;
esac

exit_code=$?

echo ""
echo "========================================"

if [ $exit_code -eq 0 ]; then
    echo "✅ All Property Tests PASSED"
else
    echo "❌ Some Property Tests FAILED"
fi

echo "========================================"
echo ""

echo "📚 For more information, see:"
echo "  - internal/balance/PROPERTY_TEST_QUICK_START.md"
echo "  - internal/balance/PROPERTY_TEST_GUIDE.md"
echo ""

exit $exit_code
