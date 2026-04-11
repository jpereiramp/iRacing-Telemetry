APP_NAME := iracing-telemetry
GO ?= go
DOCKER_COMPOSE ?= docker compose

ifeq ($(OS),Windows_NT)
APP_BIN := bin/$(APP_NAME).exe
MKDIR_BIN := powershell -NoProfile -Command "if (-not (Test-Path 'bin')) { New-Item -ItemType Directory -Path 'bin' | Out-Null }"
RM_BIN := powershell -NoProfile -Command "if (Test-Path 'bin') { Remove-Item -Recurse -Force 'bin' }"
else
APP_BIN := bin/$(APP_NAME)
MKDIR_BIN := mkdir -p bin
RM_BIN := rm -rf bin
endif

.PHONY: help fmt build test up down logs run dev clean

help:
	@echo "Targets:"
	@echo "  make fmt    Format Go files"
	@echo "  make build  Build the exporter binary"
	@echo "  make test   Run all Go tests"
	@echo "  make up     Start InfluxDB and Grafana"
	@echo "  make down   Stop the observability stack"
	@echo "  make logs   Tail Docker Compose logs"
	@echo "  make run    Run the exporter on the host"
	@echo "  make dev    Start the stack and run the exporter"
	@echo "  make clean  Remove build artifacts"

fmt:
	$(GO) fmt ./...

build:
	$(MKDIR_BIN)
	$(GO) build -o $(APP_BIN) .

test:
	$(GO) test ./...

up:
	$(DOCKER_COMPOSE) up -d influxdb grafana

down:
	$(DOCKER_COMPOSE) down

logs:
	$(DOCKER_COMPOSE) logs -f influxdb grafana

run:
	$(GO) run .

dev: up
	$(GO) run .

clean:
	$(RM_BIN)
