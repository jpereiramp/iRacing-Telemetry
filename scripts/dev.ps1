$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot

Write-Host "Running tests ..."
& "$PSScriptRoot\test.ps1"

Write-Host "Building binary ..."
& "$PSScriptRoot\build.ps1"

Write-Host "Starting app ..."
Set-Location $root
& "$PSScriptRoot\run.ps1"
