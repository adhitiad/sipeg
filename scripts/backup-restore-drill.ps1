<#
.SYNOPSIS
  Helper uji backup/restore PostgreSQL ke target DISPOSABLE (non-produksi) — $0, tanpa kredensial di file.

.DESCRIPTION
  Membantu langkah terukur: (opsional) pg_dump dari sumber, restore ke target disposable,
  dan verifikasi dasar (goose_db_version + hitungan tabel kunci). Butuh psql/pg_dump/pg_restore
  di PATH atau jalankan via Docker image postgres.

  KEAMANAN: jangan menimpa produksi. Set connection string lewat env var, bukan argumen yang tercatat.
    $env:SOURCE_DATABASE_URL = "postgres://..."
    $env:TARGET_DATABASE_URL = "postgres://..."   # HARUS database disposable/non-produksi

.EXAMPLE
  # Hanya verifikasi target disposable yang sudah dipulihkan
  ./scripts/backup-restore-drill.ps1 -Mode VerifyOnly

.EXAMPLE
  # Dump dari sumber lalu restore ke target disposable + verifikasi
  ./scripts/backup-restore-drill.ps1 -Mode DumpRestore -DumpFile .\restore_drill.dump

.NOTES
  Pemilik: adhitiad (admin). Lihat docs/backup-restore-drill.md dan docs/DECISIONS.md DEC-065.
#>

[CmdletBinding()]
param(
  [ValidateSet("VerifyOnly", "DumpRestore", "RestoreOnly")]
  [string]$Mode = "VerifyOnly",

  [string]$DumpFile = "restore_drill.dump",

  [switch]$UseDockerPgTools
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Require-Env([string]$Name) {
  $v = [Environment]::GetEnvironmentVariable($Name)
  if ([string]::IsNullOrWhiteSpace($v)) { throw "Environment variable $Name wajib diisi (jangan hardcode di file)." }
  return $v
}

function Run-Pg([string[]]$PgArgs, [string]$DbUrlEnvName) {
  $db = Require-Env $DbUrlEnvName
  if ($UseDockerPgTools) {
    & docker run --rm --network host -e PGPASSWORD="" postgres:16-alpine @PgArgs
  } else {
    & @PgArgs
  }
}

$stopwatchTotal = [System.Diagnostics.Stopwatch]::StartNew()
Write-Host "== SIMPEG backup/restore drill helper ($Mode) ==" -ForegroundColor Cyan
Write-Host "Waktu mulai (lokal): $(Get-Date -Format o)"

if ($Mode -eq "DumpRestore") {
  $src = Require-Env "SOURCE_DATABASE_URL"
  Write-Host "[A] pg_dump sumber -> $DumpFile" -ForegroundColor Yellow
  $sw = [System.Diagnostics.Stopwatch]::StartNew()
  & pg_dump $src -Fc -f $DumpFile
  $sw.Stop(); Write-Host ("Durasi dump: {0:n2} detik" -f $sw.Elapsed.TotalSeconds)
}

if ($Mode -eq "DumpRestore" -or $Mode -eq "RestoreOnly") {
  $tgt = Require-Env "TARGET_DATABASE_URL"
  if (-not (Test-Path $DumpFile)) { throw "Dump file tidak ditemukan: $DumpFile" }
  Write-Host "[B] Restore ke target DISPOSABLE: pastikan ini BUKAN produksi." -ForegroundColor Yellow
  $sw = [System.Diagnostics.Stopwatch]::StartNew()
  & pg_restore --clean --if-exists --no-owner --no-privileges -d $tgt $DumpFile
  $sw.Stop(); Write-Host ("Durasi restore: {0:n2} detik" -f $sw.Elapsed.TotalSeconds)
}

if ($Mode -in @("VerifyOnly", "DumpRestore", "RestoreOnly")) {
  $tgt = Require-Env "TARGET_DATABASE_URL"
  Write-Host "[C] Verifikasi dasar target" -ForegroundColor Yellow
  $checks = @(
    "SELECT 'goose_latest' AS check, version_id::text AS value FROM goose_db_version ORDER BY id DESC LIMIT 1;",
    "SELECT 'tables' AS check, count(*)::text FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE';",
    "SELECT 'rls_tables' AS check, count(*)::text FROM pg_tables WHERE schemaname='public' AND rowsecurity;",
    "SELECT 'users' AS check, count(*)::text FROM ""user"";",
    "SELECT 'sessions' AS check, count(*)::text FROM ""session"";",
    "SELECT 'accounts' AS check, count(*)::text FROM ""account"";",
    "SELECT 'audit_logs' AS check, count(*)::text FROM ""auditLog"";"
  )
  foreach ($q in $checks) {
    Write-Host "-- $q"
    & psql $tgt -v ON_ERROR_STOP=1 -t -A -c $q
  }
}

$stopwatchTotal.Stop()
Write-Host ("Total durasi: {0:n2} detik" -f $stopwatchTotal.Elapsed.TotalSeconds) -ForegroundColor Green
Write-Host "Selesai. Isi hasil di docs/backup-restore-drill.md (tanpa kredensial)."