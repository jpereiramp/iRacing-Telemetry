$ErrorActionPreference = "Stop"

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Error "Go is not installed or not in PATH. Install from https://go.dev/dl/ and reopen terminal."
}

Write-Host "Running tests ..."
go test ./...
Write-Host "Tests complete."
