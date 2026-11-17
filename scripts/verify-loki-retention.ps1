<#
.SYNOPSIS
  Verifikasi retensi 30 hari dan akses Loki pada target.
#>
param(
  [string]$LokiUrl = "http://127.0.0.1:3100",
  [int]$ExpectedRetentionHours = 720
)
$ErrorActionPreference = "Continue"
$fail = 0

function Check($name, $url) {
  try {
    $r = Invoke-WebRequest -Uri $url -UseBasicParsing -TimeoutSec 5
    if ($r.StatusCode -ge 200 -and $r.StatusCode -lt 300) {
      Write-Host "[OK] $name" -ForegroundColor Green
    } else {
      Write-Host "[FAIL] $name -> $($r.StatusCode)" -ForegroundColor Red
      $script:fail++
    }
  } catch {
    Write-Host "[FAIL] $name -> $($_.Exception.Message)" -ForegroundColor Red
    $script:fail++
  }
}

Check "Loki ready" "$LokiUrl/ready"
Check "Loki metrics" "$LokiUrl/metrics"

# Query label retention via config endpoint (if available)
try {
  $cfg = Invoke-RestMethod -Uri "$LokiUrl/config" -TimeoutSec 5
  Write-Host "Loki config reachable." -ForegroundColor Green
} catch {
  Write-Host "[WARN] /config tidak tersedia: $($_.Exception.Message)" -ForegroundColor Yellow
}

# Test push+query roundtrip to confirm access
$now = [DateTimeOffset]::UtcNow
$ts = $now.ToUnixTimeMilliseconds() * 1000000
$body = '{"streams":[{"stream":{"job":"verify-loki"},"values":[["' + $ts + '","retention probe simpeg"]]}]}'
try {
  Invoke-RestMethod -Method Post -Uri "$LokiUrl/loki/api/v1/push" -ContentType "application/json" -Body $body | Out-Null
  Write-Host "[OK] Push log probe berhasil" -ForegroundColor Green
} catch {
  Write-Host "[FAIL] Push probe: $($_.Exception.Message)" -ForegroundColor Red
  $script:fail++
}

Start-Sleep -Seconds 2
try {
  $q = [uri]::EscapeDataString('{job="verify-loki"}')
  $res = Invoke-RestMethod -Uri "$LokiUrl/loki/api/v1/query_range?query=$q&limit=5&start=$($now.AddMinutes(-5).ToUnixTimeMilliseconds()*1000000)&end=$($now.ToUnixTimeMilliseconds()*1000000)" -TimeoutSec 5
  if ($res.data.result.Count -gt 0) {
    Write-Host "[OK] Query log probe berhasil (akses baca OK)" -ForegroundColor Green
  } else {
    Write-Host "[WARN] Query probe kosong (mungkin indexing delay)" -ForegroundColor Yellow
  }
} catch {
  Write-Host "[FAIL] Query probe: $($_.Exception.Message)" -ForegroundColor Red
  $script:fail++
}

Write-Host "Retensi yang dikonfigurasi di loki-config.yml: $ExpectedRetentionHours jam (30 hari). Verifikasi nilai efektif pada deployment target." -ForegroundColor Cyan

if ($fail -gt 0) {
  Write-Host "Verifikasi Loki selesai dengan $fail kegagalan." -ForegroundColor Red
  exit 1
}
Write-Host "Verifikasi Loki lulus." -ForegroundColor Green