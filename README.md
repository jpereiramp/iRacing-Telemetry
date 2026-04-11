# iRacing Telemetry to Grafana

`iRacing-Telemetry` is now a host-side Go exporter for iRacing shared memory plus a containerized observability stack built with InfluxDB and Grafana.

The JavaScript widget, HTTP JSON endpoints, and SSE stream are gone. The pipeline is now:

`iRacing shared memory -> Go sampler -> InfluxDB -> Grafana dashboard`

## Why this architecture

- iRacing telemetry is dense time-series data, not operational metrics.
- InfluxDB stores correlated samples at the same timestamp cleanly, which makes pedal traces, lap analysis, and GPS plotting much easier than a scrape-based metrics system.
- Grafana can query the bucket directly for a full race engineering dashboard.
- The Go process stays on the Windows host, where the shared memory mapping is actually available.

## What the exporter writes

The exporter samples the shared memory mapping every `100ms` by default and writes:

- speed in km/h and mph
- RPM and gear
- processed and raw throttle / brake, clutch, steering angle, steering torque, and ABS activity
- lap counters, lap distance, lap times, and lap delta fields
- session number, session state, session flags, and remaining time
- overall/class position and incidents
- fuel level, fuel percentage, fuel burn, and push-to-pass state
- track, air, water, and oil temperatures plus weather, wind, humidity, precipitation, and wetness state
- tyre compounds, tyre set usage, and planned pit tyre pressures
- car dynamics such as acceleration, yaw, pitch, roll, and velocity vectors
- voltage and engine warning bits
- a derived session-local track trace for live plotting
- GPS-style location fields when available from the SDK: latitude, longitude, altitude
- car state booleans such as on pit road / on track / in garage

## Quick start

Start the observability stack:

```bash
make up
```

Run the exporter on the Windows host:

```bash
make run
```

Or do both in one command:

```bash
make dev
```

Grafana:

- URL: `http://localhost:3000`
- Username: `admin`
- Password: `admin`

InfluxDB:

- URL: `http://localhost:8086`
- Org: `iracing`
- Bucket: `telemetry`
- Token: `iracing-super-token`

The dashboards are provisioned automatically as:

- `iRacing Live Telemetry Dashboard`
- `iRacing Stint Analysis Dashboard`

## Configuration

The exporter is configured with environment variables. Defaults match the bundled Docker Compose stack.

```bash
IRACING_POLL_INTERVAL=100ms
IRACING_INFLUX_URL=http://localhost:8086
IRACING_INFLUX_ORG=iracing
IRACING_INFLUX_BUCKET=telemetry
IRACING_INFLUX_TOKEN=iracing-super-token
IRACING_INFLUX_MEASUREMENT=iracing_telemetry
IRACING_WRITE_TIMEOUT=5s
IRACING_FLUSH_INTERVAL=1s
IRACING_BATCH_SIZE=25
IRACING_BUFFER_SIZE=512
```

## Commands

```bash
make fmt
make test
make build
make up
make logs
make run
make dev
make down
```

## Windows PowerShell

The PowerShell helpers still exist for Windows-first usage:

```powershell
.\scripts\test.ps1
.\scripts\build.ps1
.\scripts\run.ps1
.\scripts\dev.ps1
.\scripts\clean.ps1
```

`dev.ps1` now starts InfluxDB and Grafana before launching the exporter.

## Important runtime note

Do not run the Go exporter inside a normal Linux Docker container. iRacing publishes the shared memory mapping on the Windows host, so the exporter needs to run directly on Windows. Docker is used only for InfluxDB and Grafana.
