$lines = Get-Content 'c:\ALHudhud\ALHudhudAI\apps\admin-web\src\locales\en.json'
Write-Host "First 5 lines:"
for ($i = 0; $i -lt 5; $i++) {
    Write-Host "Line $($i): $($lines[$i])"
}

Write-Host ""
Write-Host "Last 10 lines:"
for ($i = $lines.Length - 10; $i -lt $lines.Length; $i++) {
    Write-Host "Line $($i): $($lines[$i])"
}

# Count opening and closing braces
$openBraces = ($lines -join "").ToCharArray() | Where-Object { $_ -eq '{' } | Measure-Object | Select-Object -ExpandProperty Count
$closeBraces = ($lines -join "").ToCharArray() | Where-Object { $_ -eq '}' } | Measure-Object | Select-Object -ExpandProperty Count

Write-Host ""
Write-Host "Opening braces: $openBraces"
Write-Host "Closing braces: $closeBraces"