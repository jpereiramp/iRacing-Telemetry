$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$binary = Join-Path $root "bin\iracing-telemetry.exe"
$influxUrl = if ([string]::IsNullOrWhiteSpace($env:IRACING_INFLUX_URL)) { "http://localhost:8086" } else { $env:IRACING_INFLUX_URL }

Write-Host "Starting telemetry exporter ..."
Write-Host "InfluxDB: $influxUrl"
Write-Host "Grafana:  http://localhost:3000"
if (-not (Test-Path $binary)) {
    Write-Host "Binary not found, building first ..."
    & "$PSScriptRoot\build.ps1"
}

Set-Location $root
& $binary
