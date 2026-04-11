package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joao/iracing-telemetry/internal/config"
	"github.com/joao/iracing-telemetry/internal/influxdb"
	"github.com/joao/iracing-telemetry/internal/irsdk"
	"github.com/joao/iracing-telemetry/internal/pipeline"
)

func main() {
	logger := log.New(os.Stdout, "[iracing-telemetry] ", log.LstdFlags|log.Lmicroseconds)

	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("failed to load config: %v", err)
	}

	logger.Printf(
		"starting exporter poll_interval=%s influx_url=%s bucket=%s measurement=%s",
		cfg.PollInterval,
		cfg.Influx.URL,
		cfg.Influx.Bucket,
		cfg.Influx.Measurement,
	)

	telemetryReader, err := irsdk.NewReader()
	if err != nil {
		logger.Fatalf("failed to initialize telemetry reader: %v", err)
	}
	defer telemetryReader.Close()

	writer := influxdb.NewClient(cfg.Influx, logger)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Influx.WriteTimeout)
		defer cancel()
		if err := writer.Close(shutdownCtx); err != nil {
			logger.Printf("influxdb shutdown error: %v", err)
		}
	}()

	service := pipeline.New(cfg.PollInterval, telemetryReader, writer, logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := service.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Fatalf("pipeline failure: %v", err)
	}
}
