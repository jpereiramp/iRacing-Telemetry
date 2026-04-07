APP_NAME := iracing-telemetry
GO ?= go

.PHONY: help build test run clean

help:
	@echo "Targets:"
	@echo "  make build   Build the app binary"
	@echo "  make test    Run all Go tests"
	@echo "  make run     Run the app"
	@echo "  make clean   Remove build artifacts"

build:
	$(GO) build -o bin/$(APP_NAME).exe .

test:
	$(GO) test ./...

run:
	$(GO) run .

clean:
	@if exist bin\\$(APP_NAME).exe del /q bin\\$(APP_NAME).exe
