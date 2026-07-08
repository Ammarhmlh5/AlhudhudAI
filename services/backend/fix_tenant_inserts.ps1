$content = Get-Content "internal/worker/fast_processor_bug6_preservation_test.go" -Raw
$content = $content -replace 'INSERT INTO tenants \(id, name, email, created_at\)', 'INSERT INTO tenants (id, user_id, business_name, contact_name, email, created_at)'
$content = $content -replace 'tenantID, "Test Tenant", "test@example\.com"', 'tenantID, uuid.New(), "Test Business", "Test Contact", "test@example.com"'
$content | Set-Content "internal/worker/fast_processor_bug6_preservation_test.go" -NoNewline
Write-Host "Fixed tenant inserts"
