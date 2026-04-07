$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$binary = Join-Path $root "bin\iracing-telemetry.exe"

Write-Host "Starting app on http://localhost:8080 ..."
if (-not (Test-Path $binary)) {
    Write-Host "Binary not found, building first ..."
    & "$PSScriptRoot\build.ps1"
}

Set-Location $root
& $binary
