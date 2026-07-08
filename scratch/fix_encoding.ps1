$utf8BOM = New-Object System.Text.UTF8Encoding $true
$ps1Path = "C:\Users\Admin\Desktop\سكربتات_الهدهد\1_تشغيل_الكل.ps1"
$batPath = "C:\Users\Admin\Desktop\سكربتات_الهدهد\1_تشغيل_الكل.bat"

$ps1Content = [IO.File]::ReadAllText($ps1Path)
[IO.File]::WriteAllText($ps1Path, $ps1Content, $utf8BOM)

$batContent = [IO.File]::ReadAllText($batPath)
[IO.File]::WriteAllText($batPath, $batContent, $utf8BOM)
