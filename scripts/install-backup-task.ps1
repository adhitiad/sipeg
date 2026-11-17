<#
.SYNOPSIS
  Mendaftarkan scheduled task Windows untuk backup terjadwal terenkripsi (off-site).
.DESCRIPTION
  Membuat task harian yang menjalankan backup-restore-drill.ps1 -Mode ScheduledBackup -Encrypt.
  Kredensial dibaca dari environment; atur SOURCE_DATABASE_URL dan GPG_PASSPHRASE pada
  tingkat sistem/user sebelum menjalankan skrip ini. Default jam 02:00 WIB.
#>
param(
  [string]$TaskName = "SIMPEG-Backup-Terjadwal",
  [string]$BackupDir = "D:\simpeg-backups",
  [string]$Time = "02:00",
  [int]$RetentionDays = 14
)
$ErrorActionPreference = "Stop"
$script = Resolve-Path (Join-Path $PSScriptRoot "backup-restore-drill.ps1")
$action = New-ScheduledTaskAction -Execute "powershell.exe" `
  -Argument "-NoProfile -ExecutionPolicy Bypass -File `"$script`" -Mode ScheduledBackup -BackupDir `"$BackupDir`" -Encrypt -RemovePlainAfterEncrypt"
$trigger = New-ScheduledTaskTrigger -Daily -At $Time
$settings = New-ScheduledTaskSettingsSet -StartWhenAvailable -RunOnlyIfNetworkAvailable
$principal = New-ScheduledTaskPrincipal -UserId "$env:USERDOMAIN\$env:USERNAME" -LogonType Interactive -RunLevel Highest
$task = New-ScheduledTask -Action $action -Trigger $trigger -Settings $settings -Principal $principal
Register-ScheduledTask -TaskName $TaskName -InputObject $task -Force | Out-Null
[Environment]::SetEnvironmentVariable("BACKUP_RETENTION_DAYS", $RetentionDays, "User")
Write-Host "Task '$TaskName' terdaftar: harian $Time, retensi $RetentionDays hari, target $BackupDir" -ForegroundColor Green
Write-Host "Pastikan SOURCE_DATABASE_URL dan GPG_PASSPHRASE tersedia pada akun yang menjalankan task."