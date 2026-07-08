#!/usr/bin/env pwsh
# PowerShell script to run Invoice Generator tests
# This script runs all tests for the invoice generation functionality

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Invoice Generator Test Runner" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check if Go is installed
Write-Host "Checking Go installation..." -ForegroundColor Yellow
$goVersion = go version 2>$null
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Error: Go is not installed or not in PATH" -ForegroundColor Red
    Write-Host "Please install Go from https://golang.org/dl/" -ForegroundColor Yellow
    exit 1
}
Write-Host "✅ Go is installed: $goVersion" -ForegroundColor Green
Write-Host ""

# Check if database is running
Write-Host "Checking database connection..." -ForegroundColor Yellow
$env:PGPASSWORD = "postgres"
$dbCheck = psql -h localhost -U postgres -d sms_gateway -c "SELECT 1;" 2>$null
if ($LASTEXITCODE -ne 0) {
    Write-Host "⚠️  Warning: Database connection failed" -ForegroundColor Yellow
    Write-Host "Some tests may be skipped" -ForegroundColor Yellow
} else {
    Write-Host "✅ Database connection successful" -ForegroundColor Green
}
Write-Host ""

# Run unit tests
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Running Unit Tests" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

Write-Host "Testing Invoice Generator..." -ForegroundColor Yellow
go test -v ./internal/billing/... -run TestGenerateInvoice -short
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Invoice Generator tests failed" -ForegroundColor Red
} else {
    Write-Host "✅ Invoice Generator tests passed" -ForegroundColor Green
}
Write-Host ""

Write-Host "Testing PDF Generator..." -ForegroundColor Yellow
go test -v ./internal/billing/... -run TestPDF -short
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ PDF Generator tests failed" -ForegroundColor Red
} else {
    Write-Host "✅ PDF Generator tests passed" -ForegroundColor Green
}
Write-Host ""

Write-Host "Testing Invoice API..." -ForegroundColor Yellow
go test -v ./internal/api/... -run TestInvoice -short
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Invoice API tests failed" -ForegroundColor Red
} else {
    Write-Host "✅ Invoice API tests passed" -ForegroundColor Green
}
Write-Host ""

# Run integration tests (if database is available)
if ($dbCheck) {
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host "Running Integration Tests" -ForegroundColor Cyan
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host ""

    Write-Host "Testing with database..." -ForegroundColor Yellow
    go test -v ./internal/billing/... -short=false
    if ($LASTEXITCODE -ne 0) {
        Write-Host "❌ Integration tests failed" -ForegroundColor Red
    } else {
        Write-Host "✅ Integration tests passed" -ForegroundColor Green
    }
    Write-Host ""
}

# Run all tests with coverage
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Running Tests with Coverage" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

go test ./internal/billing/... -coverprofile=coverage.out -short
if ($LASTEXITCODE -eq 0) {
    Write-Host ""
    Write-Host "Coverage Report:" -ForegroundColor Cyan
    go tool cover -func=coverage.out | Select-String "total:"
    Write-Host ""
    Write-Host "To view detailed coverage in browser, run:" -ForegroundColor Yellow
    Write-Host "  go tool cover -html=coverage.out" -ForegroundColor White
}
Write-Host ""

# Summary
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Test Summary" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "✅ Invoice Generator: Implemented" -ForegroundColor Green
Write-Host "✅ PDF Generator: Implemented" -ForegroundColor Green
Write-Host "✅ Email Sender: Implemented (stub)" -ForegroundColor Green
Write-Host "✅ Invoice API: Implemented" -ForegroundColor Green
Write-Host ""
Write-Host "📚 Documentation:" -ForegroundColor Cyan
Write-Host "  - README: internal/billing/README.md" -ForegroundColor White
Write-Host "  - Quick Start: internal/billing/INVOICE_QUICK_START.md" -ForegroundColor White
Write-Host "  - Migration: migrations/README_BILLING_INVOICES.md" -ForegroundColor White
Write-Host ""
Write-Host "🚀 Next Steps:" -ForegroundColor Cyan
Write-Host "  1. Review the Quick Start guide" -ForegroundColor White
Write-Host "  2. Test invoice generation manually" -ForegroundColor White
Write-Host "  3. Configure email delivery" -ForegroundColor White
Write-Host "  4. Integrate with payment flow" -ForegroundColor White
Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Done! ✨" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Cyan
