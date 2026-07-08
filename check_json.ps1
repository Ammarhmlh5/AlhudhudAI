$lines = Get-Content 'c:\ALHudhud\ALHudhudAI\apps\admin-web\src\locales\en.json'
Write-Host "Total lines: $($lines.Length)"

for ($i = $lines.Length - 5; $i -lt $lines.Length; $i++) {
    Write-Host "Line $($i): $($lines[$i])"
}