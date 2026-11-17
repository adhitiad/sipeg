<#
.SYNOPSIS
  Hapus (DROP) SEMUA tabel di skema public KECUALI tabel "user".
  DESTRUKTIF — hanya untuk dev/staging. Jangan jalankan pada produksi.

.DESCRIPTION
  Membaca DATABASE_URL dari environment (tidak hardcode kredensial).
  Guard:
    - Menolak bila APP_ENV=production.
    - Menolak bila nama database tidak mengandung kata kunci dev/test/staging/perf/verify/drill.
    - Memerlukan konfirmasi eksplisit -ConfirmDrop.
    - Menampilkan daftar tabel yang akan dihapus dan meminta konfirmasi interaktif.
  Tabel "user" dipertahankan. Tabel lain (termasuk goose_db_version, session,
  account, auditLog, dll.) akan di-DROP CASCADE.

.NOTES
  Jalankan hanya pada database disposable/dev. Tidak ada rollback.
#>
param(
  [switch]$ConfirmDrop,
  [switch]$Force
)
$ErrorActionPreference = "Stop"

function Require-Env([string]$Name) {
  $v = [Environment]::GetEnvironmentVariable($Name)
  if ([string]::IsNullOrWhiteSpace($v)) { throw "Environment variable $Name wajib diisi." }
  return $v
}

$dbUrl = Require-Env "DATABASE_URL"
$appEnv = [Environment]::GetEnvironmentVariable("APP_ENV")
if ([string]::IsNullOrWhiteSpace($appEnv)) { $appEnv = "development" }

if ($appEnv -eq "production") {
  throw "DITOLAK: APP_ENV=production. Skrip ini hanya untuk dev/staging."
}

# Ekstrak nama database dari URL untuk guard
$dbName = ""
try {
  $uri = [System.Uri]$dbUrl
  $dbName = $uri.AbsolutePath.TrimStart('/')
} catch {
  # format DSN non-URI; coba parse manual
  if ($dbUrl -match 'dbname=(\S+)') { $dbName = $matches[1] }
  elseif ($dbUrl -match '/(\w+)(\?|$)') { $dbName = $matches[1] }
}

$allowed = @("dev", "test", "staging", "perf", "verify", "drill", "local", "simpeg")
$safe = $false
foreach ($kw in $allowed) {
  if ($dbName -match $kw) { $safe = $true; break }
}
if (-not $safe) {
  throw "DITOLAK: nama database '$dbName' tidak terlihat seperti dev/staging (harus mengandung salah satu: $($allowed -join ', ')). Gunakan -Force untuk melewati guard ini (TIDAK disarankan)."
}
if ($Force) { $safe = $true }

if (-not $ConfirmDrop) {
  throw "Jalankan dengan -ConfirmDrop untuk mengonfirmasi penghapusan. Tanpa flag ini skrip tidak melakukan apa pun."
}

$env:PGPASSWORD = ""
# Coba set password dari URL (psql membaca PGPASSWORD dari env)
if ($dbUrl -match '://[^:]+:([^@]+)@') {
  $env:PGPASSWORD = [System.Uri]::UnescapeDataString($matches[1])
}

Write-Host "== DROP semua tabel KECUALI 'user' ==" -ForegroundColor Red
Write-Host "Database : $dbName" -ForegroundColor Yellow
Write-Host "APP_ENV  : $appEnv" -ForegroundColor Yellow

# Ambil daftar tabel
$listSql = "SELECT tablename FROM pg_tables WHERE schemaname='public' AND tablename <> 'user' ORDER BY tablename;"
$tables = psql $dbUrl -t -A -c $listSql 2>&1
if ($LASTEXITCODE -ne 0) { throw "Gagal membaca daftar tabel: $tables" }
$tables = $tables | Where-Object { -not [string]::IsNullOrWhiteSpace($_) }

if (-not $tables -or $tables.Count -eq 0) {
  Write-Host "Tidak ada tabel yang perlu dihapus." -ForegroundColor Green
  exit 0
}

Write-Host "Tabel yang akan DIHAPUS ($($tables.Count)):" -ForegroundColor Red
$tables | ForEach-Object { Write-Host "  - $_" -ForegroundColor Red }

if (-not $Force) {
  $ans = Read-Host "Ketik 'HAPUS' untuk melanjutkan"
  if ($ans -ne "HAPUS") { Write-Host "Dibatalkan." -ForegroundColor Yellow; exit 0 }
}

# Bangun SQL DROP
$dropSql = ($tables | ForEach-Object { "DROP TABLE IF EXISTS ""$_"" CASCADE;" }) -join "`n"
Set-Content -Path (Join-Path $PSScriptRoot "_tmp_drop.sql") -Value $dropSql -NoNewline

Write-Host "Menjalankan DROP CASCADE..." -ForegroundColor Yellow
psql $dbUrl -v ON_ERROR_STOP=1 -f (Join-Path $PSScriptRoot "_tmp_drop.sql") 2>&1
if ($LASTEXITCODE -ne 0) { throw "DROP gagal." }

Remove-Item -Force (Join-Path $PSScriptRoot "_tmp_drop.sql") -ErrorAction SilentlyContinue
Write-Host "Selesai. Tabel 'user' dipertahankan; tabel lain dihapus." -ForegroundColor Green