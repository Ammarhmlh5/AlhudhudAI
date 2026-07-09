import os

bat_start = """@echo off
chcp 65001 > nul
title تشغيل جميع خدمات نظام الهدهد
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp01_تشغيل_الكل.ps1"
"""

bat_stop = """@echo off
chcp 65001 > nul
title إيقاف جميع خدمات نظام الهدهد
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp02_إيقاف_الكل.ps1"
"""

# Update on Desktop
with open(r'C:\Users\Admin\Desktop\سكربتات_الهدهد\1_تشغيل_الكل.bat', 'w', encoding='utf-8-sig') as f:
    f.write(bat_start)

with open(r'C:\Users\Admin\Desktop\سكربتات_الهدهد\2_إيقاف_الكل.bat', 'w', encoding='utf-8-sig') as f:
    f.write(bat_stop)

# Update in Workspace Root
with open(r'C:\ALHudhud\1_تشغيل_الكل.bat', 'w', encoding='utf-8-sig') as f:
    f.write(bat_start)

print("Batch wrappers updated successfully.")
