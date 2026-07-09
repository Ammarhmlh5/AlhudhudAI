# build-production.ps1
# ALHudhudAI Gateway Production Build Script

# --- Environment Setup (Hardcoded for this system) ---
$env:ANDROID_HOME = "C:\Android\Sdk"
$env:JAVA_HOME = "C:\Program Files\Microsoft\jdk-17.0.19.10-hotspot"
$env:NODE_OPTIONS = "--max-old-space-size=2048"
$env:PATH = "C:\ALHudhud\platform-tools;$env:ANDROID_HOME\platform-tools;$env:JAVA_HOME\bin;$env:PATH"
# ---------------------------------------------------

Write-Host "--- ALHudhudAI Mobile Gateway Build System ---" -ForegroundColor Cyan

# 1. Build Release APK (Incremental)
Write-Host "[1/2] Building Release APK..." -ForegroundColor Cyan
cd android
./gradlew assembleRelease
cd ..

# 4. Result Check & Auto-Install
$apkPath = "android/app/build/outputs/apk/release/app-release.apk"
if (Test-Path $apkPath) {
    Write-Host "[SUCCESS] Production APK generated at: $apkPath" -ForegroundColor Green
    
    $size = (Get-Item $apkPath).Length / 1MB
    Write-Host ("Artifact Size: {0:N2} MB" -f $size) -ForegroundColor Green

    Write-Host "[3/4] Installing on device..." -ForegroundColor Yellow
    C:\ALHudhud\platform-tools\adb.exe install -r $apkPath
    if ($LASTEXITCODE -eq 0) {
        Write-Host "SUCCESS: Application installed and ready for testing." -ForegroundColor Green
    }
    else {
        Write-Warning "Installation failed via ADB. Please install manually."
    }
}
else {
    Write-Host "[ERROR] Build failed. Check the logs above." -ForegroundColor Red
    exit 1
}

Write-Host "--- DONE ---" -ForegroundColor Yellow
