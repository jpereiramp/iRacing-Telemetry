$ErrorActionPreference = "Stop"

$binary = ".\bin\iracing-telemetry.exe"
if (Test-Path $binary) {
    Remove-Item $binary -Force
    Write-Host "Removed $binary"
} else {
    Write-Host "Nothing to clean."
}
