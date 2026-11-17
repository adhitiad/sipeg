<#
.SYNOPSIS
  Uji firing/resolved alert ke Alertmanager (lokal) — validasi penerima.
.DESCRIPTION
  Mengirim alert uji ke Alertmanager API v2. Gunakan untuk memverifikasi
  routing penerima sebelum deploy. Set AlertmanagerUrl bila berbeda.
#>
param(
  [string]$AlertmanagerUrl = "http://127.0.0.1:9093",
  [switch]$Resolve
)
$ErrorActionPreference = "Stop"
$status = if ($Resolve) { "resolved" } else { "firing" }
$now = (Get-Date).ToUniversalTime().ToString("o")
$endsAt = if ($Resolve) { ',"endsAt":"' + $now + '"' } else { "" }
$body = '[{"labels":{"alertname":"SimpegTestAlert","severity":"warning","job":"test","instance":"localhost"},"annotations":{"summary":"Alert uji SIMPEG","description":"Alert uji untuk memverifikasi penerima Alertmanager."},"startsAt":"' + $now + '"' + $endsAt + '}]'
Invoke-RestMethod -Method Post -Uri "$AlertmanagerUrl/api/v2/alerts" -ContentType "application/json" -Body $body | Out-Null
Write-Host "Alert uji ($status) terkirim ke $AlertmanagerUrl. Periksa penerima." -ForegroundColor Green