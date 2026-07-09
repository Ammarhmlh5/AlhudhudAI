#!/usr/bin/env pwsh
# ALHudhudAI - Test Runner Script
# This script runs all tests for the project

$ErrorActionPreference = "Stop"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  ALHudhudAI Test Runner" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Function to run backend tests
function Run-BackendTests {
    Write-Host "[1/3] Running Backend Tests (Go)..." -ForegroundColor Yellow
    
    $backendPath = "services/backend"
    
    if (-not (Test-Path $backendPath)) {
        Write-Host "  ⚠ Backend directory not found, skipping..." -ForegroundColor Gray
        return
    }
    
    Push-Location $backendPath
    
    try {
        # Run all unit tests
        Write-Host "  → Running unit tests..." -ForegroundColor Gray
        go test ./internal/... -v -short
        
        # Run integration tests (if not in short mode)
        if (-not $env:CI) {
            Write-Host "  → Running integration tests..." -ForegroundColor Gray
            go test ./tests/... -v
        }
        
        # Run property-based tests
        Write-Host "  → Running property-based tests..." -ForegroundColor Gray
        go test ./... -run TestBalanceManager_PropertyBased -v
        
        Write-Host "  ✅ Backend tests completed!" -ForegroundColor Green
    }
    catch {
        Write-Host "  ❌ Backend tests failed: $_" -ForegroundColor Red
        exit 1
    }
    finally {
        Pop-Location
    }
}

# Function to run frontend tests
function Run-FrontendTests {
    Write-Host "[2/3] Running Frontend Tests (React)..." -ForegroundColor Yellow
    
    $frontendPath = "apps/admin-web"
    
    if (-not (Test-Path $frontendPath)) {
        Write-Host "  ⚠ Frontend directory not found, skipping..." -ForegroundColor Gray
        return
    }
    
    Push-Location $frontendPath
    
    try {
        # Check if node_modules exists
        if (-not (Test-Path "node_modules")) {
            Write-Host "  → Installing dependencies..." -ForegroundColor Gray
            npm ci
        }
        
        # Run unit tests with Vitest
        Write-Host "  → Running unit tests with Vitest..." -ForegroundColor Gray
        npm run test:run
        
        # Run E2E tests with Playwright
        if (-not $env:CI) {
            Write-Host "  → Running E2E tests with Playwright..." -ForegroundColor Gray
            npm run test:e2e
        }
        
        Write-Host "  ✅ Frontend tests completed!" -ForegroundColor Green
    }
    catch {
        Write-Host "  ❌ Frontend tests failed: $_" -ForegroundColor Red
        exit 1
    }
    finally {
        Pop-Location
    }
}

# Function to run all tests
function Run-AllTests {
    Write-Host "[3/3] Running All Tests..." -ForegroundColor Yellow
    
    # Backend tests
    Run-BackendTests
    Write-Host ""
    
    # Frontend tests
    Run-FrontendTests
    Write-Host ""
}

# Main execution
$mode = $args[0]

switch ($mode) {
    "backend" {
        Run-BackendTests
    }
    "frontend" {
        Run-FrontendTests
    }
    "all" {
        Run-AllTests
    }
    "coverage" {
        # Run with coverage
        $env:GENERATE_COVERAGE = "true"
        Run-AllTests
    }
    default {
        # Default: run all tests
        Run-AllTests
    }
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  All tests completed successfully!" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan