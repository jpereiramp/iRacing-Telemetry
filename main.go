package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joao/iracing-telemetry/internal/irsdk"
	"github.com/joao/iracing-telemetry/internal/server"
)

func main() {
	logger := log.New(os.Stdout, "[iracing-telemetry] ", log.LstdFlags|log.Lmicroseconds)

	telemetryReader, err := irsdk.NewReader()
	if err != nil {
		logger.Fatalf("failed to initialize telemetry reader: %v", err)
	}
	defer telemetryReader.Close()

	srv := server.New(":8080", telemetryReader, logger)

	go func() {
		logger.Printf("server listening on http://localhost%s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("server failure: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Printf("graceful shutdown error: %v", err)
	}
}
