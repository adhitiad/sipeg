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
  [ValidateSet("VerifyOnly", "DumpRestore", "RestoreOnly", "ScheduledBackup")]
  [string]$Mode = "VerifyOnly",

  [string]$DumpFile = "restore_drill.dump",

  [switch]$UseDockerPgTools,

  # Enkripsi artefak dump dengan gpg (wajib passphrase via env GPG_PASSPHRASE).
  [switch]$Encrypt,

  # Hapus artefak plaintext setelah enkripsi (default: true bila -Encrypt).
  [switch]$RemovePlainAfterEncrypt,

  # Direktori tujuan untuk ScheduledBackup (off-site mount/lokal terpisah).
  [string]$BackupDir = "."
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

if ($Mode -eq "DumpRestore" -or $Mode -eq "ScheduledBackup") {
  $srcRaw = Require-Env "SOURCE_DATABASE_URL"
  # pg_dump tidak mengenali parameter query khusus pooler (mis.
  # default_query_exec_mode=simple_protocol). Bersihkan query string sebelum
  # memanggil pg_dump; kredensial tetap berasal dari URL asli.
  $src = $srcRaw -replace '\?.*$', ''
  if ($Mode -eq "ScheduledBackup") {
    $stamp = Get-Date -Format "yyyyMMdd-HHmmss"
    $DumpFile = Join-Path $BackupDir ("simpeg-backup-" + $stamp + ".dump")
  }
  Write-Host "[A] pg_dump sumber -> $DumpFile" -ForegroundColor Yellow
  $sw = [System.Diagnostics.Stopwatch]::StartNew()
  & pg_dump $src -Fc -f $DumpFile
  $sw.Stop(); Write-Host ("Durasi dump: {0:n2} detik" -f $sw.Elapsed.TotalSeconds)

  # Checksum untuk verifikasi integritas
  $hash = (Get-FileHash -Algorithm SHA256 $DumpFile).Hash
  Set-Content -Path ($DumpFile + ".sha256") -Value ($hash + "  " + (Split-Path -Leaf $DumpFile))
  Write-Host "SHA256: $hash" -ForegroundColor Gray

  if ($Encrypt) {
    $pass = [Environment]::GetEnvironmentVariable("GPG_PASSPHRASE")
    if ([string]::IsNullOrWhiteSpace($pass)) { throw "GPG_PASSPHRASE wajib diisi untuk -Encrypt." }
    $encFile = $DumpFile + ".gpg"
    Write-Host "[A2] Enkripsi -> $encFile" -ForegroundColor Yellow
    # gpg symmetric, AES256, batch
    $pass | & gpg --batch --yes --passphrase-fd 0 --symmetric --cipher-algo AES256 -o $encFile $DumpFile
    if (-not (Test-Path $encFile)) { throw "Enkripsi gagal: $encFile tidak terbentuk." }
    if ($RemovePlainAfterEncrypt -or $true) {
      Remove-Item -Force $DumpFile
      Write-Host "Artefak plaintext dihapus setelah enkripsi." -ForegroundColor Gray
    }
  }
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

if ($Mode -eq "ScheduledBackup") {
  $retentionDays = 14
  if ($env:BACKUP_RETENTION_DAYS) { $retentionDays = [int]$env:BACKUP_RETENTION_DAYS }
  Write-Host "[D] Retensi: hapus backup lebih lama dari $retentionDays hari di $BackupDir" -ForegroundColor Yellow
  Get-ChildItem -Path $BackupDir -Filter "simpeg-backup-*.dump*" -File |
    Where-Object { $_.LastWriteTime -lt (Get-Date).AddDays(-$retentionDays) } |
    ForEach-Object { Remove-Item -Force $_.FullName; Write-Host "  dihapus: $($_.Name)" -ForegroundColor Gray }
}

$stopwatchTotal.Stop()
Write-Host ("Total durasi: {0:n2} detik" -f $stopwatchTotal.Elapsed.TotalSeconds) -ForegroundColor Green
Write-Host "Selesai. Isi hasil di docs/backup-restore-drill.md (tanpa kredensial)."