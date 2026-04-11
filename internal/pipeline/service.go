package pipeline

import (
	"context"
	"log"
	"time"

	"github.com/joao/iracing-telemetry/internal/irsdk"
)

type telemetryReader interface {
	ReadSnapshot() irsdk.TelemetrySnapshot
}

type telemetryWriter interface {
	Write(context.Context, irsdk.TelemetrySnapshot) error
	Close(context.Context) error
}

type Service struct {
	pollInterval time.Duration
	reader       telemetryReader
	writer       telemetryWriter
	logger       *log.Logger

	lastWriteErr string
}

func New(pollInterval time.Duration, reader telemetryReader, writer telemetryWriter, logger *log.Logger) *Service {
	return &Service{
		pollInterval: pollInterval,
		reader:       reader,
		writer:       writer,
		logger:       logger,
	}
}

func (s *Service) Run(ctx context.Context) error {
	if err := s.sampleOnce(ctx); err != nil {
		return err
	}

	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := s.sampleOnce(ctx); err != nil {
				return err
			}
		}
	}
}

func (s *Service) sampleOnce(ctx context.Context) error {
	snapshot := s.reader.ReadSnapshot()
	if err := s.writer.Write(ctx, snapshot); err != nil {
		if s.lastWriteErr != err.Error() && s.logger != nil {
			s.logger.Printf("pipeline write error: %v", err)
		}
		s.lastWriteErr = err.Error()
		return err
	}

	if s.lastWriteErr != "" && s.logger != nil {
		s.logger.Printf("pipeline recovered: writes resumed")
	}
	s.lastWriteErr = ""

	return nil
}
