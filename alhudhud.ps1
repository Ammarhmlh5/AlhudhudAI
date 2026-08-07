[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidateSet('start', 'stop', 'status', 'restart', 'backend', 'web', 'databases', 'mobile', 'simulator', 'check', 'help')]
    [string]$Command = 'help'
)

try {
    [Console]::OutputEncoding = [System.Text.Encoding]::UTF8
    $OutputEncoding = [System.Text.Encoding]::UTF8
} catch {}

$ProjectRoot = 'C:\ALHudhud\ALHudhud\ALHudhudAI'
$MobileDirs = @('F:\ALHudhudMobile', 'C:\ALHudhud\ALHudhud\ALHudhudMobile')
$MobileDir = $MobileDirs | Where-Object { Test-Path -LiteralPath $_ } | Select-Object -First 1
$DockerDesktop = 'C:\Program Files\Docker\Docker\Docker Desktop.exe'

$Services = @(
    @{ Name = 'Backend API';     Port = 8085; Group = 'backend'; Dir = "$ProjectRoot\services\backend";     Exe = "$ProjectRoot\services\backend\backend_new.exe";     Start = 'go run main.go';              Env = @('APP_ENV=development') },
    @{ Name = 'Company API';     Port = 8082; Group = 'backend'; Dir = "$ProjectRoot\services\company-api"; Exe = "$ProjectRoot\services\company-api\company-api.exe"; Start = 'go run main.go';              Env = @('APP_ENV=development') },
    @{ Name = 'Admin Web';       Port = 5174; Group = 'web';     Dir = "$ProjectRoot\apps\admin-web";       Start = 'npm run dev -- --port 5174';  Env = @() },
    @{ Name = 'Client Web';      Port = 5173; Group = 'web';     Dir = "$ProjectRoot\apps\client-web";      Start = 'npm run dev -- --port 5173';  Env = @() },
    @{ Name = 'Reseller Web';    Port = 5175; Group = 'web';     Dir = "$ProjectRoot\apps\reseller-web";    Start = 'npm run dev -- --port 5175';  Env = @() },
    @{ Name = 'Company Portal';  Port = 5176; Group = 'web';     Dir = "$ProjectRoot\apps\company-portal";  Start = 'npm run dev -- --port 5176';  Env = @() },
    @{ Name = 'AlHudhud Mobile'; Port = 8081; Group = 'mobile';  Dir = $MobileDir;                          Start = 'npx expo start';              Env = @('NODE_OPTIONS=--max-old-space-size=4096') }
)

$WatchPorts = @(8085, 8082, 5173, 5174, 5175, 5176, 8081)
$PgServices = @('postgresql-x64-18', 'postgresql-x64-17', 'postgresql-x64-16', 'postgresql-x64-15', 'postgresql-x64-14', 'PostgreSQL', 'postgresql')
$RedisServices = @('Redis', 'redis-server')

function Test-PortOpen {
    param([int]$Port)
    foreach ($addr in @('127.0.0.1', '::1')) {
        try {
            if ($addr -eq '::1') {
                $tcp = New-Object System.Net.Sockets.TcpClient([System.Net.Sockets.AddressFamily]::InterNetworkV6)
            } else {
                $tcp = New-Object System.Net.Sockets.TcpClient
            }
            $async = $tcp.BeginConnect($addr, $Port, $null, $null)
            $ok = $async.AsyncWaitHandle.WaitOne(500)
            if ($ok -and $tcp.Connected) {
                $tcp.EndConnect($async)
                $tcp.Close()
                return $true
            }
            $tcp.Close()
        } catch {}
    }
    return $false
}

function Get-ComposeFile {
    return Get-ChildItem -LiteralPath $ProjectRoot -File -ErrorAction SilentlyContinue |
        Where-Object { $_.Name -match '^(docker-)?compose(\.override)?\.ya?ml$' } |
        Select-Object -First 1
}

function Test-CommandExists {
    param([string]$Name)
    return [bool](Get-Command $Name -ErrorAction SilentlyContinue)
}

function Start-LocalServices {
    if (-not (Test-PortOpen 5433)) {
        $started = $false
        foreach ($svc in $PgServices) {
            $s = Get-Service -Name $svc -ErrorAction SilentlyContinue
            if ($s -and $s.Status -ne 'Running') {
                Start-Service -Name $svc -ErrorAction SilentlyContinue
                if ((Get-Service -Name $svc -ErrorAction SilentlyContinue).Status -eq 'Running') {
                    Write-Host "  PostgreSQL يعمل ($svc)" -ForegroundColor Green
                    $started = $true
                    break
                }
            } elseif ($s) {
                Write-Host '  PostgreSQL يعمل' -ForegroundColor Green
                $started = $true
                break
            }
        }
        if (-not $started) {
            Write-Host '  تعذر تشغيل PostgreSQL تلقائياً (المنفذ 5433)' -ForegroundColor Yellow
        }
    } else {
        Write-Host '  PostgreSQL يعمل (5433)' -ForegroundColor Green
    }

    if (-not (Test-PortOpen 6379)) {
        $started = $false
        foreach ($svc in $RedisServices) {
            $s = Get-Service -Name $svc -ErrorAction SilentlyContinue
            if ($s -and $s.Status -ne 'Running') {
                Start-Service -Name $svc -ErrorAction SilentlyContinue
                if ((Get-Service -Name $svc -ErrorAction SilentlyContinue).Status -eq 'Running') {
                    Write-Host "  Redis يعمل ($svc)" -ForegroundColor Green
                    $started = $true
                    break
                }
            } elseif ($s) {
                Write-Host '  Redis يعمل' -ForegroundColor Green
                $started = $true
                break
            }
        }
        if (-not $started) {
            Write-Host '  تعذر تشغيل Redis تلقائياً (المنفذ 6379)' -ForegroundColor Yellow
        }
    } else {
        Write-Host '  Redis يعمل (6379)' -ForegroundColor Green
    }
}

function Start-Infrastructure {
    Write-Host '== 1) البنية التحتية (قواعد البيانات) ==' -ForegroundColor Magenta
    if (Test-PortOpen 5433 -and Test-PortOpen 6379) {
        Write-Host '  قواعد البيانات تعمل بالفعل (PostgreSQL 5433 / Redis 6379)' -ForegroundColor Green
        return
    }

    $composeFile = Get-ComposeFile
    if ($composeFile) {
        $dockerReady = $false
        try { $null = docker ps 2>&1; if ($LASTEXITCODE -eq 0) { $dockerReady = $true } } catch {}

        if (-not $dockerReady -and (Test-Path $DockerDesktop)) {
            Write-Host '  جاري تشغيل Docker Desktop...' -ForegroundColor Yellow
            Start-Process $DockerDesktop
            for ($i = 0; $i -lt 20; $i++) {
                Start-Sleep -Seconds 3
                try { $null = docker ps 2>&1; if ($LASTEXITCODE -eq 0) { $dockerReady = $true; break } } catch {}
            }
        }

        if ($dockerReady) {
            Write-Host "  جاري تشغيل الحاويات ($($composeFile.Name))..." -ForegroundColor Yellow
            Push-Location $ProjectRoot
            docker compose up -d
            Pop-Location
            Start-Sleep -Seconds 8
            return
        }
    }

    Start-LocalServices
}

function Start-Service {
    param([hashtable]$Service)

    if (Test-PortOpen $Service.Port) {
        Write-Host "  $($Service.Name) يعمل بالفعل (المنفذ $($Service.Port))" -ForegroundColor Gray
        return
    }

    if (-not (Test-Path -LiteralPath $Service.Dir)) {
        Write-Host "  [خطأ] مجلد غير موجود: $($Service.Dir)" -ForegroundColor Red
        return
    }

    Write-Host "  جاري تشغيل $($Service.Name)..." -ForegroundColor Yellow
    $cmd = $Service.Start
    if ($Service.Exe -and (Test-Path -LiteralPath $Service.Exe)) {
        $cmd = ".\$(Split-Path $Service.Exe -Leaf)"
    }
    $envPart = ($Service.Env | ForEach-Object {
        $kv = $_ -split '=', 2
        if ($kv.Count -eq 2) { "`$env:$($kv[0])='$($kv[1])'; " } else { "`$env:$_=''; " }
    }) -join ''
    $runCmd = "cd '$($Service.Dir)'; $envPart Write-Host '=== $($Service.Name) ===' -ForegroundColor Green; $cmd"
    Start-Process powershell -ArgumentList '-NoExit', '-Command', $runCmd -WindowStyle Normal
    Start-Sleep -Seconds 2
}

function Start-Group {
    param([string]$Group)

    if (-not $MobileDir -and $Group -eq 'mobile') {
        Write-Host '  [خطأ] لم يتم العثور على مجلد تطبيق الهاتف (F:\ALHudhudMobile)' -ForegroundColor Red
        return
    }

    foreach ($service in ($Services | Where-Object { $_.Group -eq $Group })) {
        Start-Service $service
    }
}

function Show-Summary {
    param([string[]]$Group = @())
    $targets = if ($Group.Count -eq 0) { $Services } else { $Services | Where-Object { $_.Group -in $Group } }

    Write-Host ''
    Write-Host '== ملخص التشغيل ==' -ForegroundColor Green
    Start-Sleep -Seconds 8
    foreach ($service in $targets) {
        if (Test-PortOpen $service.Port) {
            Write-Host "  [يعمل]  $($service.Name)  http://localhost:$($service.Port)" -ForegroundColor Green
        } else {
            Write-Host "  [قيد الإقلاع]  $($service.Name)  http://localhost:$($service.Port)" -ForegroundColor Yellow
        }
    }

    Write-Host ''
    Write-Host 'الخدمات:' -ForegroundColor Cyan
    Write-Host '  • واجهة العميل:     http://localhost:5173' -ForegroundColor White
    Write-Host '  • لوحة التحكم:      http://localhost:5174' -ForegroundColor White
    Write-Host '  • واجهة الموزعين:   http://localhost:5175' -ForegroundColor White
    Write-Host '  • بوابة الشركات:    http://localhost:5176' -ForegroundColor White
    Write-Host '  • Backend API:      http://localhost:8085' -ForegroundColor White
    Write-Host '  • Company API:      http://localhost:8082' -ForegroundColor White
    Write-Host '  • تطبيق الهاتف:     npx expo start (نافذة منفصلة)' -ForegroundColor White
}

function Start-All {
    Write-Host '========================================' -ForegroundColor Cyan
    Write-Host '  تشغيل نظام الهدهد بالكامل' -ForegroundColor Cyan
    Write-Host '========================================' -ForegroundColor Cyan
    Write-Host ''

    Start-Infrastructure

    Write-Host ''
    Write-Host '== 2) تشغيل الخدمات ==' -ForegroundColor Magenta
    foreach ($group in @('backend', 'web', 'mobile')) {
        Start-Group $group
    }

    Show-Summary
}

function Start-Simulator {
    Write-Host '========================================' -ForegroundColor Cyan
    Write-Host '  تشغيل أداة محاكاة إرسال الرسائل' -ForegroundColor Cyan
    Write-Host '  (اختبار وصول الرسائل من الأنظمة المحاسبية)' -ForegroundColor Cyan
    Write-Host '========================================' -ForegroundColor Cyan
    Write-Host ''

    $htmlPath = "$ProjectRoot\apps\client-web\public\test.html"

    if (-not (Test-Path -LiteralPath $htmlPath)) {
        Write-Host '  [خطأ] لم يتم العثور على أداة المحاكاة:' -ForegroundColor Red
        Write-Host "  $htmlPath" -ForegroundColor Red
        Read-Host 'اضغط Enter للخروج...'
        return
    }

    if (Test-PortOpen 8085) {
        Write-Host '  [يعمل] Backend API على المنفذ 8085' -ForegroundColor Green
    } else {
        Write-Host '  [تحذير] Backend API غير مشغّل على المنفذ 8085!' -ForegroundColor Yellow
        Write-Host '  شغّله أولاً:  .\alhudhud.ps1 backend' -ForegroundColor Yellow
    }

    Write-Host ''
    if (Test-PortOpen 5173) {
        Write-Host '  جاري فتح أداة المحاكاة في المتصفح (Client Web 5173)...' -ForegroundColor Cyan
        Start-Process 'http://localhost:5173/test.html'
        Write-Host '  الرابط: http://localhost:5173/test.html' -ForegroundColor White
    } else {
        Write-Host '  [تحذير] واجهة العميل (5173) غير مشغّلة، سيتم فتح الملف مباشرة.' -ForegroundColor Yellow
        Start-Process $htmlPath
        Write-Host "  المسار: $htmlPath" -ForegroundColor White
    }

    Write-Host ''
    Write-Host 'تعليمات الاستخدام:' -ForegroundColor Yellow
    Write-Host '  1. أدخل مفتاح API الخاص بالمستأجر (API Key)' -ForegroundColor White
    Write-Host '  2. حدد وضع الإرسال: فردي (رسالة لكل طلب) أو جماعي (صفيف واحد)' -ForegroundColor White
    Write-Host '  3. حدد عدد الرسائل، التزامن/التأخير (فردي) أو حجم الدفعة الفرعية (جماعي)' -ForegroundColor White
    Write-Host '  4. اكتب قالب الرسالة (يدعم {i} و {phone}) واضغط "بدء الإرسال"' -ForegroundColor White
    Write-Host '  5. تابع التقدم في شريط التقدم ويمكنك الإيقاف أو تصدير النتائج (CSV)' -ForegroundColor White
    Write-Host ''
}

function Stop-All {
    Write-Host '========================================' -ForegroundColor Cyan
    Write-Host '  إيقاف جميع خدمات نظام الهدهد' -ForegroundColor Cyan
    Write-Host '========================================' -ForegroundColor Cyan
    Write-Host ''

    $stopped = @()

    Write-Host '[1/2] إيقاف العمليات التي تستخدم منافذ المشروع...' -ForegroundColor Yellow
    $killNames = '^(node|go|main|server|backend|company|vite|expo|java)'
    foreach ($port in $WatchPorts) {
        try {
            $conns = Get-NetTCPConnection -LocalPort $port -State Listen -ErrorAction SilentlyContinue
            foreach ($conn in $conns) {
                if ($conn.OwningProcess) {
                    $p = Get-CimInstance Win32_Process -Filter "ProcessId=$($conn.OwningProcess)" -ErrorAction SilentlyContinue
                    if ($p -and $p.Name -match $killNames) {
                        Stop-Process -Id $conn.OwningProcess -Force -ErrorAction SilentlyContinue
                        $stopped += $conn.OwningProcess
                    }
                }
            }
        } catch {}
    }

    Write-Host '[2/2] إغلاق نوافذ التشغيل المفتوحة وإيقاف العمليات التابعة للمشروع...' -ForegroundColor Yellow
    $winProcs = Get-CimInstance Win32_Process -ErrorAction SilentlyContinue |
        Where-Object { $_.Name -eq 'powershell.exe' -and $_.CommandLine -match 'ALHudhud|ALHudhudMobile' -and $_.CommandLine -match '-NoExit' }
    foreach ($p in $winProcs) {
        Stop-Process -Id $p.ProcessId -Force -ErrorAction SilentlyContinue
        $stopped += $p.ProcessId
    }

    $compiled = Get-CimInstance Win32_Process -ErrorAction SilentlyContinue |
        Where-Object { $_.Name -match '^(main|server|company-api|backend).*\.exe$' -and $_.CommandLine -match 'ALHudhud|go-build' }
    foreach ($p in $compiled) {
        Stop-Process -Id $p.ProcessId -Force -ErrorAction SilentlyContinue
        $stopped += $p.ProcessId
    }

    if ($stopped.Count -gt 0) {
        Write-Host "  تم إيقاف $($stopped.Count) عملية" -ForegroundColor Green
    } else {
        Write-Host '  لا توجد عمليات تعمل' -ForegroundColor Gray
    }

    $composeFile = Get-ComposeFile
    if ($composeFile) {
        try {
            $null = docker ps 2>&1
            if ($LASTEXITCODE -eq 0) {
                Write-Host 'إيقاف حاويات Docker...' -ForegroundColor Yellow
                Push-Location $ProjectRoot
                docker compose down
                Pop-Location
            }
        } catch {}
    }

    Write-Host ''
    Write-Host 'تم إيقاف جميع الخدمات.' -ForegroundColor Green
}

function Show-Status {
    Write-Host '========================================' -ForegroundColor Cyan
    Write-Host '  حالة خدمات نظام الهدهد' -ForegroundColor Cyan
    Write-Host '========================================' -ForegroundColor Cyan
    Write-Host ''

    foreach ($service in $Services) {
        if (Test-PortOpen $service.Port) {
            Write-Host "  [يعمل]  $($service.Name)  (المنفذ $($service.Port))" -ForegroundColor Green
        } else {
            Write-Host "  [متوقف] $($service.Name)  (المنفذ $($service.Port))" -ForegroundColor Red
        }
    }

    Write-Host ''
    if (Test-PortOpen 5433) {
        Write-Host '  [يعمل]  PostgreSQL (5433)' -ForegroundColor Green
    } else {
        Write-Host '  [متوقف] PostgreSQL (5433)' -ForegroundColor Red
    }
    if (Test-PortOpen 6379) {
        Write-Host '  [يعمل]  Redis (6379)' -ForegroundColor Green
    } else {
        Write-Host '  [متوقف] Redis (6379)' -ForegroundColor Red
    }
}

function Show-Check {
    Write-Host '========================================' -ForegroundColor Cyan
    Write-Host '  فحص متطلبات نظام الهدهد' -ForegroundColor Cyan
    Write-Host '========================================' -ForegroundColor Cyan
    Write-Host ''

    $allGood = $true

    Write-Host '[1/7] مجلد المشروع...' -ForegroundColor Yellow
    if (Test-Path -LiteralPath $ProjectRoot) {
        Write-Host "  [موجود] $ProjectRoot" -ForegroundColor Green
    } else {
        Write-Host "  [خطأ] المشروع غير موجود: $ProjectRoot" -ForegroundColor Red
        $allGood = $false
    }

    Write-Host '[2/7] Go...' -ForegroundColor Yellow
    if (Test-CommandExists 'go') {
        Write-Host "  [موجود] $(go version)" -ForegroundColor Green
    } else {
        Write-Host '  [خطأ] Go غير مثبت' -ForegroundColor Red
        $allGood = $false
    }

    Write-Host '[3/7] Node.js...' -ForegroundColor Yellow
    if (Test-CommandExists 'node') {
        Write-Host "  [موجود] Node $(node --version)" -ForegroundColor Green
    } else {
        Write-Host '  [خطأ] Node.js غير مثبت' -ForegroundColor Red
        $allGood = $false
    }

    Write-Host '[4/7] npm...' -ForegroundColor Yellow
    if (Test-CommandExists 'npm') {
        Write-Host "  [موجود] npm $(npm --version)" -ForegroundColor Green
    } else {
        Write-Host '  [خطأ] npm غير مثبت' -ForegroundColor Red
        $allGood = $false
    }

    Write-Host '[5/7] مجلدات الخدمات...' -ForegroundColor Yellow
    foreach ($service in $Services) {
        $dirOk = Test-Path -LiteralPath $service.Dir
        $color = if ($dirOk) { 'Green' } else { 'Red' }
        $mark = if ($dirOk) { '[موجود]' } else { '[خطأ]' }
        Write-Host "  $mark $($service.Name): $($service.Dir)" -ForegroundColor $color
        if (-not $dirOk) { $allGood = $false }
    }

    Write-Host '[6/7] قواعد البيانات...' -ForegroundColor Yellow
    if (Test-PortOpen 5433) {
        Write-Host '  [يعمل]  PostgreSQL (5433)' -ForegroundColor Green
    } else {
        Write-Host '  [متوقف] PostgreSQL (5433)' -ForegroundColor Yellow
    }
    if (Test-PortOpen 6379) {
        Write-Host '  [يعمل]  Redis (6379)' -ForegroundColor Green
    } else {
        Write-Host '  [متوقف] Redis (6379)' -ForegroundColor Yellow
    }

    Write-Host '[7/7] تطبيق الهاتف...' -ForegroundColor Yellow
    if ($MobileDir) {
        Write-Host "  [موجود] $MobileDir" -ForegroundColor Green
    } else {
        Write-Host '  [تحذير] لم يتم العثور على مجلد الهاتف (اختياري)' -ForegroundColor Yellow
    }

    Write-Host ''
    if ($allGood) {
        Write-Host '  تم التحقق بنجاح، جميع المتطلبات متوفرة.' -ForegroundColor Green
    } else {
        Write-Host '  توجد متطلبات ناقصة، راجع الأخطاء بالأعلى.' -ForegroundColor Red
    }
}

function Show-Help {
    Write-Host '========================================' -ForegroundColor Cyan
    Write-Host '  سكربت إدارة نظام الهدهد الموحد' -ForegroundColor Cyan
    Write-Host '========================================' -ForegroundColor Cyan
    Write-Host ''
    Write-Host 'الاستخدام:  .\alhudhud.ps1 <الأمر>' -ForegroundColor White
    Write-Host ''
    Write-Host 'الأوامر المتاحة:' -ForegroundColor Yellow
    Write-Host '  start       تشغيل جميع الخدمات (قواعد البيانات + الخوادم + الواجهات + الهاتف)' -ForegroundColor Green
    Write-Host '  stop        إيقاف جميع الخدمات' -ForegroundColor Green
    Write-Host '  restart     إيقاف ثم تشغيل جميع الخدمات' -ForegroundColor Green
    Write-Host '  status      عرض حالة الخدمات والمنافذ' -ForegroundColor Green
    Write-Host '  backend     تشغيل قواعد البيانات + Backend API + Company API' -ForegroundColor Green
    Write-Host '  web         تشغيل الواجهات الأربع (Client/Admin/Reseller/Company Portal)' -ForegroundColor Green
    Write-Host '  databases   تشغيل قواعد البيانات فقط (PostgreSQL + Redis)' -ForegroundColor Green
    Write-Host '  mobile      تشغيل تطبيق الهاتف (Expo)' -ForegroundColor Green
    Write-Host '  simulator   تشغيل أداة محاكاة إرسال الرسائل (اختبار وصول الرسائل من الأنظمة المحاسبية)' -ForegroundColor Green
    Write-Host '  check       فحص متطلبات النظام' -ForegroundColor Green
    Write-Host '  help        عرض هذه التعليمات' -ForegroundColor Green
    Write-Host ''
    Write-Host 'أمثلة:' -ForegroundColor Yellow
    Write-Host '  .\alhudhud.ps1 start' -ForegroundColor White
    Write-Host '  .\alhudhud.ps1 status' -ForegroundColor White
    Write-Host ''
    Write-Host 'المنافذ:' -ForegroundColor Yellow
    Write-Host '  Backend API 8085 | Company API 8082 | Client 5173 | Admin 5174 | Reseller 5175 | Company Portal 5176 | Mobile 8081' -ForegroundColor White
}

switch ($Command) {
    'start'     { Start-All }
    'stop'      { Stop-All }
    'status'    { Show-Status }
    'restart'   { Stop-All; Write-Host ''; Start-All }
    'backend'   { Write-Host 'تشغيل قواعد البيانات + الخوادم...' -ForegroundColor Cyan; Start-Infrastructure; Write-Host ''; Start-Group 'backend'; Show-Summary @('backend') }
    'web'       { Write-Host 'تشغيل الواجهات...' -ForegroundColor Cyan; Start-Group 'web'; Show-Summary @('web') }
    'databases' { Start-Infrastructure }
    'mobile'    { Start-Group 'mobile'; Show-Summary @('mobile') }
    'simulator' { Start-Simulator }
    'check'     { Show-Check }
    default     { Show-Help }
}
