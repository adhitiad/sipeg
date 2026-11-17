<#
.SYNOPSIS
  Export schema, data, and custom roles from a Supabase Free project.

.DESCRIPTION
  Uses the current Supabase CLI `supabase db dump --remote=<project-ref>` command.
  Requires the CLI to be authenticated and Docker to be available because db dump
  runs pg_dump in a container. Link/login and provision credentials for the same
  Windows task account before scheduling this script.

  This script does NOT encrypt or upload backups. OutputRoot MUST be an encrypted,
  access-controlled local/off-site staging volume outside the repository. Copy
  only encrypted artefacts to a separate off-site location.

.EXAMPLE
  .\scripts\supabase-free-backup.ps1 -ProjectRef "abcdefghijklmnopqrst" -OutputRoot "D:\SIMPEG-Backups"
#>

[CmdletBinding()]
param(
  [Parameter(Mandatory = $true)]
  [ValidateNotNullOrEmpty()]
  [string]$ProjectRef,

  [Parameter(Mandatory = $true)]
  [ValidateNotNullOrEmpty()]
  [string]$OutputRoot
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Invoke-SupabaseDump {
  param(
    [Parameter(Mandatory = $true)]
    [string[]]$Arguments,

    [Parameter(Mandatory = $true)]
    [string]$Label
  )

  Write-Host "[$Label] Starting Supabase export..."
  & supabase @Arguments
  if ($LASTEXITCODE -ne 0) {
    throw "$Label export failed (exit code $LASTEXITCODE)."
  }
}

if ($ProjectRef -match "\s" -or $ProjectRef.Contains("/")) {
  throw "ProjectRef must be a project reference, not a URL or connection string."
}

$supabase = Get-Command supabase -ErrorAction SilentlyContinue
if ($null -eq $supabase) {
  throw "Supabase CLI was not found in PATH."
}
$docker = Get-Command docker -ErrorAction SilentlyContinue
if ($null -eq $docker) {
  throw "Docker CLI was not found in PATH; Supabase CLI db dump runs pg_dump in a container."
}

$repoRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot ".."))
$outputRootFull = [System.IO.Path]::GetFullPath($OutputRoot)
$repoPrefix = $repoRoot.TrimEnd([System.IO.Path]::DirectorySeparatorChar) + [System.IO.Path]::DirectorySeparatorChar
if ($outputRootFull.StartsWith($repoPrefix, [System.StringComparison]::OrdinalIgnoreCase) -or
    $outputRootFull.Equals($repoRoot, [System.StringComparison]::OrdinalIgnoreCase)) {
  throw "OutputRoot must be outside the Git repository."
}

& docker info *> $null
if ($LASTEXITCODE -ne 0) {
  throw "Docker daemon is not reachable."
}

New-Item -ItemType Directory -Force -Path $outputRootFull | Out-Null
$stamp = (Get-Date).ToUniversalTime().ToString("yyyyMMddTHHmmssZ")
$runDir = Join-Path $outputRootFull "$ProjectRef-$stamp"
New-Item -ItemType Directory -Force -Path $runDir | Out-Null

$schemaFile = Join-Path $runDir "schema.sql"
$dataFile = Join-Path $runDir "data.sql"
$rolesFile = Join-Path $runDir "roles.sql"

$startedAt = (Get-Date).ToUniversalTime()
try {
  Invoke-SupabaseDump -Label "schema" -Arguments @(
    "db", "dump", "--remote=$ProjectRef", "--file", $schemaFile
  )
  Invoke-SupabaseDump -Label "data" -Arguments @(
    "db", "dump", "--remote=$ProjectRef", "--data-only", "--file", $dataFile
  )
  Invoke-SupabaseDump -Label "roles" -Arguments @(
    "db", "dump", "--remote=$ProjectRef", "--role-only", "--file", $rolesFile
  )

  foreach ($file in @($schemaFile, $dataFile, $rolesFile)) {
    if (-not (Test-Path -LiteralPath $file -PathType Leaf)) {
      throw "Expected backup artefact is missing: $([System.IO.Path]::GetFileName($file))"
    }
  }

  $files = @($schemaFile, $dataFile, $rolesFile) | ForEach-Object {
    $item = Get-Item -LiteralPath $_
    $hash = Get-FileHash -LiteralPath $_ -Algorithm SHA256
    [pscustomobject]@{
      File         = $item.Name
      Bytes        = $item.Length
      SHA256       = $hash.Hash
    }
  }

  $metadata = [pscustomobject]@{
    ProjectRef = $ProjectRef
    CreatedAtUtc = $startedAt.ToString("o")
    CompletedAtUtc = (Get-Date).ToUniversalTime().ToString("o")
    SupabaseCLI = (& supabase --version | Out-String).Trim()
    Format = "plain SQL; restore with psql -v ON_ERROR_STOP=1"
    Note = "Supabase-managed auth/storage/extension schemas are excluded by db dump. Review coverage and custom roles."
    Files = $files
  }
  $metadata | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath (Join-Path $runDir "manifest.json") -Encoding UTF8

  Write-Host ""
  Write-Host "Supabase exports completed: $runDir" -ForegroundColor Green
  $files | Format-Table File, Bytes, SHA256 -AutoSize
  Write-Warning "Encrypt these files and copy them to independent off-site storage. This script does not encrypt, upload, or retain/delete older backups."
  Write-Warning "A dump is not a restore test. Run a separate restore drill against a disposable project/database."
}
catch {
  Write-Error "Backup export incomplete. Keep the failed folder for diagnosis, secure it, and do not treat it as a valid backup set."
  throw
}
