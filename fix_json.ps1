$all = Get-Content 'c:\ALHudhud\ALHudhudAI\apps\admin-web\src\locales\en.json'
$total = $all.Length
$keep = $total - 50
$result = $all[0..$keep]

$notifications = @(
    ',',
    '  "notifications": {',
    '    "settings": {',
    '      "title": "Notification Settings",',
    '      "subtitle": "Control which notifications users receive",',
    '      "tabs": {',
    '        "settings": "Settings",',
    '        "templates": "Templates",',
    '        "logs": "Notification Logs"',
    '      }',
    '    },',
    '    "templates": {',
    '      "title": "Notification Templates",',
    '      "edit": "Edit Template"',
    '    },',
    '    "types": {',
    '      "reconnection_alert": "Reconnection Alert",',
    '      "package_expiry_warning": "Package Expiry Warning",',
    '      "package_expiry_alert": "Package Expiry Alert",',
    '      "low_balance_warning": "Low Balance Warning",',
    '      "message_pending_alert": "Pending Messages",',
    '      "delivery_failed_alert": "Delivery Failed",',
    '      "system_update": "System Updates",',
    '      "new_message": "New Messages"',
    '    }',
    '  },',
    '  "nav": {',
    '    "notifications": "Notifications"',
    '  }',
    '}'
)

$result + $notifications | Set-Content 'c:\ALHudhud\ALHudhudAI\apps\admin-web\src\locales\en.json'
Write-Host "JSON file fixed successfully"