[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Invoke-Checked {
  param(
    [Parameter(Mandatory = $true)][string]$Label,
    [Parameter(Mandatory = $true)][scriptblock]$Command
  )
  Write-Host "== $Label ==" -ForegroundColor Cyan
  & $Command
  if ($LASTEXITCODE -ne 0) {
    throw "$Label failed (exit code $LASTEXITCODE)."
  }
}

$observabilityDir = $PSScriptRoot
$composeExample = Join-Path $observabilityDir ".env.example"
if (-not (Test-Path $composeExample)) {
  throw "Missing observability/.env.example."
}

Push-Location $observabilityDir
try {
  Invoke-Checked -Label "Observability Docker Compose config" -Command {
    docker compose --env-file .env.example config --quiet
  }

  Invoke-Checked -Label "RabbitMQ Docker Compose config" -Command {
    docker compose -f ../rabbitmq/docker-compose.yml config --quiet
  }

  $mountDir = (Resolve-Path $observabilityDir).Path

  Invoke-Checked -Label "Grafana Alloy configuration validation" -Command {
    docker run --rm --volume ($mountDir + ":/etc/alloy:ro") --entrypoint alloy grafana/alloy:v1.21.3 validate --stability.level=generally-available /etc/alloy/alloy-config.alloy
  }

  Invoke-Checked -Label "Grafana Alloy formatting" -Command {
    docker run --rm --volume ($mountDir + ":/etc/alloy:ro") --entrypoint alloy grafana/alloy:v1.21.3 fmt --test /etc/alloy/alloy-config.alloy
  }

  Invoke-Checked -Label "Prometheus configuration and alert rules" -Command {
    docker run --rm --volume ($mountDir + ":/etc/prometheus:ro") --entrypoint promtool prom/prometheus:v3.16.0 check config /etc/prometheus/prometheus.yml
  }

  Invoke-Checked -Label "Alertmanager configuration" -Command {
    docker run --rm --volume ((Join-Path $mountDir "alertmanager.yml") + ":/etc/alertmanager/alertmanager.yml:ro") --entrypoint amtool prom/alertmanager:v0.35.0 check-config /etc/alertmanager/alertmanager.yml
  }

  $dashboardPath = Join-Path $observabilityDir "grafana/dashboards/simpeg-infrastructure.json"
  $null = Get-Content -LiteralPath $dashboardPath -Raw | ConvertFrom-Json -Depth 100
  Write-Host "Grafana dashboard JSON: valid" -ForegroundColor Green

  if (Test-Path (Join-Path $observabilityDir "promtail-config.yml")) {
    throw "Retired Promtail configuration must not be present; use Grafana Alloy."
  }

  Write-Host "Observability configuration validation passed." -ForegroundColor Green
}
finally {
  Pop-Location
}
