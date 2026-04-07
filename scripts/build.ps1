param(
    [string]$Output = ".\bin\iracing-telemetry.exe"
)

$ErrorActionPreference = "Stop"

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Error "Go is not installed or not in PATH. Install from https://go.dev/dl/ and reopen terminal."
}

$outDir = Split-Path -Parent $Output
if ($outDir -and -not (Test-Path $outDir)) {
    New-Item -ItemType Directory -Path $outDir | Out-Null
}

Write-Host "Building to $Output ..."
go build -o $Output .
Write-Host "Build complete."
