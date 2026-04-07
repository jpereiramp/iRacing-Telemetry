# iRacing Telemetry (Go + Modern Widget)

A modern, minimal iRacing telemetry app written in Go with:

- CGO-free shared-memory reader for iRacing telemetry on Windows
- Real-time HTTP stream endpoint (SSE)
- A single modern web widget showing live speed via a gauge + trend graph

This project is inspired by:

- https://github.com/margic/goiracing
- https://github.com/quimcalpe/iracing-sdk

## Run

```bash
go run .
```

## Windows (PowerShell scripts)

```powershell
.\scripts\test.ps1
.\scripts\build.ps1
.\scripts\run.ps1
.\scripts\clean.ps1
.\scripts\dev.ps1
```

`run.ps1` launches `.\bin\iracing-telemetry.exe` (building first if needed), which avoids Windows policies that often block the temporary binaries created by `go run`.

Then open:

- http://localhost:8080

## Notes

- If iRacing is running and shared memory is available, data source is `live`.
- If not available, the widget stays in waiting mode and does not display synthetic values.
