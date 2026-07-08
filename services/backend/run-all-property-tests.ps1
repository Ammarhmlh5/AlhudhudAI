# ============================================================================
# Property-Based Tests Runner - نظام التعرفات والتسعير
# ============================================================================
# 
# هذا السكريبت يقوم بتشغيل جميع Property-Based Tests للنظام
# يغطي Properties 1-20 المحددة في التصميم
#
# الاستخدام:
#   .\run-all-property-tests.ps1
#   .\run-all-property-tests.ps1 -Verbose
#   .\run-all-property-tests.ps1 -Package balance
#
# ============================================================================

param(
    [string]$Package = "all",
    [switch]$Verbose,
    [switch]$Short
)

Write-Host "============================================================================" -ForegroundColor Cyan
Write-Host "Property-Based Tests Runner - نظام التعرفات والتسعير" -ForegroundColor Cyan
Write-Host "============================================================================" -ForegroundColor Cyan
Write-Host ""

# التحقق من وجود قاعدة البيانات
Write-Host "Checking database connection..." -ForegroundColor Yellow
$env:DATABASE_URL = "postgres://postgres:postgres@localhost:5432/sms_gateway_test?sslmode=disable"

# اختبار الاتصال بقاعدة البيانات
try {
    $null = & psql $env:DATABASE_URL -c "SELECT 1" 2>&1
    Write-Host "✓ Database connection successful" -ForegroundColor Green
} catch {
    Write-Host "✗ Database connection failed" -ForegroundColor Red
    Write-Host "Please ensure PostgreSQL is running and the test database exists" -ForegroundColor Yellow
    exit 1
}

Write-Host ""

# تعريف جميع Property Tests
$propertyTests = @(
    @{
        Name = "Property 1-2: Balance Operations"
        Package = "balance"
        Test = "TestProperty_AtomicBalanceOperations"
        Requirements = "16.1, 16.2"
        Description = "تحديث التعرفة والاحتفاظ بالسجل التاريخي"
    },
    @{
        Name = "Property 15: Atomic Balance Operations"
        Package = "balance"
        Test = "TestProperty_NoRaceConditionsInBalanceOperations"
        Requirements = "16.1, 16.2"
        Description = "عمليات الرصيد الذرية بدون Race Conditions"
    },
    @{
        Name = "Property 3: Active Tariff Usage"
        Package = "tariff"
        Test = "TestProperty_UseActiveTariffOnly"
        Requirements = "2.5"
        Description = "استخدام التعرفة النشطة الصحيحة"
    },
    @{
        Name = "Property 12: Message Segmentation"
        Package = "tariff"
        Test = "TestProperty_MessageSegmentation"
        Requirements = "7.6"
        Description = "حساب أجزاء الرسالة بدقة"
    },
    @{
        Name = "Property 5: Safe Subscription"
        Package = "subscription"
        Test = "TestProperty_SafeSubscription"
        Requirements = "4.1, 4.2"
        Description = "الاشتراك الآمن في الباقة"
    },
    @{
        Name = "Property 6: Message Deduction"
        Package = "subscription"
        Test = "TestProperty_MessageDeduction"
        Requirements = "4.5"
        Description = "خصم من عداد الرسائل عند الإرسال"
    },
    @{
        Name = "Property 8: Overage Charging"
        Package = "subscription"
        Test = "TestProperty_OverageCharging"
        Requirements = "5.3, 5.4"
        Description = "خصم تكلفة الرسائل الإضافية من الرصيد"
    },
    @{
        Name = "Property 9: Calculate Remaining Messages"
        Package = "rollover"
        Test = "TestProperty_CalculateRemainingMessages"
        Requirements = "6.2"
        Description = "حساب الرسائل المتبقية عند نهاية الشهر"
    },
    @{
        Name = "Property 10: Rollover Limit Respect"
        Package = "rollover"
        Test = "TestProperty_RolloverMessages_RespectLimit"
        Requirements = "6.5"
        Description = "احترام الحد الأقصى للترحيل"
    },
    @{
        Name = "Property 11: FIFO Rollover Usage"
        Package = "rollover"
        Test = "TestProperty_UseRolloverMessages_FIFO"
        Requirements = "6.8"
        Description = "استخدام الرسائل المرحّلة بترتيب FIFO"
    },
    @{
        Name = "Property 13: Auto Renewal"
        Package = "billing"
        Test = "TestProperty_AutoRenewal"
        Requirements = "14.2, 14.3"
        Description = "التجديد التلقائي عند نهاية الشهر"
    },
    @{
        Name = "Property 20: Rate Limiting per Tenant"
        Package = "resources"
        Test = "TestProperty_RateLimitingPerTenant"
        Requirements = "9.8"
        Description = "Rate Limiting معزول لكل Tenant"
    }
)

# تصفية الاختبارات حسب Package المحدد
if ($Package -ne "all") {
    $propertyTests = $propertyTests | Where-Object { $_.Package -eq $Package }
    if ($propertyTests.Count -eq 0) {
        Write-Host "No tests found for package: $Package" -ForegroundColor Red
        exit 1
    }
}

Write-Host "Running $($propertyTests.Count) Property Tests..." -ForegroundColor Cyan
Write-Host ""

# متغيرات لتتبع النتائج
$totalTests = $propertyTests.Count
$passedTests = 0
$failedTests = 0
$skippedTests = 0
$results = @()

# تشغيل كل اختبار
foreach ($test in $propertyTests) {
    Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor DarkGray
    Write-Host "Running: $($test.Name)" -ForegroundColor Cyan
    Write-Host "Package: internal/$($test.Package)" -ForegroundColor Gray
    Write-Host "Test: $($test.Test)" -ForegroundColor Gray
    Write-Host "Requirements: $($test.Requirements)" -ForegroundColor Gray
    Write-Host "Description: $($test.Description)" -ForegroundColor Gray
    Write-Host ""

    # بناء أمر الاختبار
    $testPath = ".\internal\$($test.Package)"
    $testArgs = @("-v")
    
    if ($Short) {
        $testArgs += "-short"
    }
    
    if ($Verbose) {
        $testArgs += "-count=1"
    }
    
    $testArgs += "-run"
    $testArgs += "^$($test.Test)$"
    
    # تشغيل الاختبار
    $startTime = Get-Date
    
    try {
        $output = & go test @testArgs $testPath 2>&1
        $exitCode = $LASTEXITCODE
        $duration = (Get-Date) - $startTime
        
        if ($exitCode -eq 0) {
            Write-Host "✓ PASSED" -ForegroundColor Green
            Write-Host "Duration: $($duration.TotalSeconds.ToString('F2'))s" -ForegroundColor Gray
            $passedTests++
            $results += @{
                Test = $test.Name
                Status = "PASSED"
                Duration = $duration.TotalSeconds
                Output = $output
            }
        } else {
            Write-Host "✗ FAILED" -ForegroundColor Red
            Write-Host "Duration: $($duration.TotalSeconds.ToString('F2'))s" -ForegroundColor Gray
            $failedTests++
            $results += @{
                Test = $test.Name
                Status = "FAILED"
                Duration = $duration.TotalSeconds
                Output = $output
            }
            
            # عرض آخر 20 سطر من الخطأ
            Write-Host ""
            Write-Host "Error Output (last 20 lines):" -ForegroundColor Yellow
            $output | Select-Object -Last 20 | ForEach-Object {
                Write-Host $_ -ForegroundColor Red
            }
        }
    } catch {
        Write-Host "✗ ERROR" -ForegroundColor Red
        Write-Host "Error: $_" -ForegroundColor Red
        $failedTests++
        $results += @{
            Test = $test.Name
            Status = "ERROR"
            Duration = 0
            Output = $_.Exception.Message
        }
    }
    
    Write-Host ""
}

# عرض الملخص
Write-Host "============================================================================" -ForegroundColor Cyan
Write-Host "Test Summary" -ForegroundColor Cyan
Write-Host "============================================================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Total Tests:   $totalTests" -ForegroundColor White
Write-Host "Passed:        $passedTests" -ForegroundColor Green
Write-Host "Failed:        $failedTests" -ForegroundColor Red
Write-Host "Skipped:       $skippedTests" -ForegroundColor Yellow
Write-Host ""

# عرض النتائج التفصيلية
if ($Verbose -or $failedTests -gt 0) {
    Write-Host "Detailed Results:" -ForegroundColor Cyan
    Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor DarkGray
    
    foreach ($result in $results) {
        $statusColor = switch ($result.Status) {
            "PASSED" { "Green" }
            "FAILED" { "Red" }
            "ERROR" { "Red" }
            default { "Yellow" }
        }
        
        Write-Host "$($result.Status.PadRight(10)) $($result.Test)" -ForegroundColor $statusColor
        Write-Host "             Duration: $($result.Duration.ToString('F2'))s" -ForegroundColor Gray
        
        if ($result.Status -ne "PASSED" -and $Verbose) {
            Write-Host "             Output:" -ForegroundColor Gray
            $result.Output | Select-Object -Last 10 | ForEach-Object {
                Write-Host "             $_" -ForegroundColor DarkGray
            }
        }
        Write-Host ""
    }
}

# حساب إجمالي الوقت
$totalDuration = ($results | Measure-Object -Property Duration -Sum).Sum
Write-Host "Total Duration: $($totalDuration.ToString('F2'))s" -ForegroundColor Gray
Write-Host ""

# الخروج بكود الخطأ المناسب
if ($failedTests -gt 0) {
    Write-Host "Some tests failed. Please review the output above." -ForegroundColor Red
    exit 1
} else {
    Write-Host "All tests passed successfully! ✓" -ForegroundColor Green
    exit 0
}
