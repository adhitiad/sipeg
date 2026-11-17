<#
.SYNOPSIS
  Verifikasi cepat stack observability SIMPEG terhadap host lokal.
.DESCRIPTION
  Memeriksa Prometheus targets UP, Alertmanager status, Loki readiness, dan
  (opsional) metricsexport. Tidak memerlukan kredensial; endpoint di 127.0.0.1.
#>
param(
  [string]$PrometheusUrl = "http://127.0.0.1:9090",
  [string]$AlertmanagerUrl = "http://127.0.0.1:9093",
  [string]$LokiUrl = "http://127.0.0.1:3100",
  [string]$ExporterUrl = "http://127.0.0.1:9101"
)
$ErrorActionPreference = "Continue"
$fail = 0

function Check($name, $url) {
  try {
    $r = Invoke-WebRequest -Uri $url -UseBasicParsing -TimeoutSec 5
    if ($r.StatusCode -ge 200 -and $r.StatusCode -lt 300) {
      Write-Host "[OK] $name ($url) -> $($r.StatusCode)" -ForegroundColor Green
    } else {
      Write-Host "[FAIL] $name ($url) -> $($r.StatusCode)" -ForegroundColor Red
      $script:fail++
    }
  } catch {
    Write-Host "[FAIL] $name ($url) -> $($_.Exception.Message)" -ForegroundColor Red
    $script:fail++
  }
}

Check "Prometheus ready" "$PrometheusUrl/-/ready"
Check "Alertmanager ready" "$AlertmanagerUrl/-/ready"
Check "Loki ready" "$LokiUrl/ready"
Check "metricsexport" "$ExporterUrl/health"

# Prometheus targets
try {
  $t = Invoke-RestMethod -Uri "$PrometheusUrl/api/v1/targets" -TimeoutSec 5
  $active = $t.data.activeTargets
  $down = $active | Where-Object { $_.health -ne "up" }
  Write-Host "Prometheus targets: $($active.Count) total, $($down.Count) down" -ForegroundColor Cyan
  if ($down.Count -gt 0) {
    $down | ForEach-Object { Write-Host "  DOWN: $($_.labels.job) / $($_.labels.instance)" -ForegroundColor Yellow }
    $script:fail++
  }
} catch {
  Write-Host "[FAIL] targets: $($_.Exception.Message)" -ForegroundColor Red
  $script:fail++
}

# Alertmanager status
try {
  $a = Invoke-RestMethod -Uri "$AlertmanagerUrl/api/v2/status" -TimeoutSec 5
  Write-Host "Alertmanager config OK" -ForegroundColor Green
} catch {
  Write-Host "[WARN] Alertmanager status: $($_.Exception.Message)" -ForegroundColor Yellow
}

if ($fail -gt 0) {
  Write-Host "Verifikasi selesai dengan $fail kegagalan." -ForegroundColor Red
  exit 1
}
Write-Host "Verifikasi observability lulus." -ForegroundColor Green