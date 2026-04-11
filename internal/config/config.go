package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPollInterval    = 100 * time.Millisecond
	defaultInfluxURL       = "http://localhost:8086"
	defaultInfluxOrg       = "iracing"
	defaultInfluxBucket    = "telemetry"
	defaultInfluxToken     = "iracing-super-token"
	defaultMeasurement     = "iracing_telemetry"
	defaultWriteTimeout    = 5 * time.Second
	defaultFlushInterval   = 1 * time.Second
	defaultBatchSize       = 25
	defaultChannelCapacity = 512
)

type Config struct {
	PollInterval time.Duration
	Influx       InfluxConfig
}

type InfluxConfig struct {
	URL             string
	Org             string
	Bucket          string
	Token           string
	Measurement     string
	WriteTimeout    time.Duration
	FlushInterval   time.Duration
	ChannelCapacity int
	BatchSize       int
}

func Load() (Config, error) {
	pollInterval, err := durationFromEnv("IRACING_POLL_INTERVAL", defaultPollInterval)
	if err != nil {
		return Config{}, err
	}

	writeTimeout, err := durationFromEnv("IRACING_WRITE_TIMEOUT", defaultWriteTimeout)
	if err != nil {
		return Config{}, err
	}

	flushInterval, err := durationFromEnv("IRACING_FLUSH_INTERVAL", defaultFlushInterval)
	if err != nil {
		return Config{}, err
	}

	channelCapacity, err := intFromEnv("IRACING_BUFFER_SIZE", defaultChannelCapacity)
	if err != nil {
		return Config{}, err
	}

	batchSize, err := intFromEnv("IRACING_BATCH_SIZE", defaultBatchSize)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		PollInterval: pollInterval,
		Influx: InfluxConfig{
			URL:             stringFromEnv("IRACING_INFLUX_URL", defaultInfluxURL),
			Org:             stringFromEnv("IRACING_INFLUX_ORG", defaultInfluxOrg),
			Bucket:          stringFromEnv("IRACING_INFLUX_BUCKET", defaultInfluxBucket),
			Token:           stringFromEnv("IRACING_INFLUX_TOKEN", defaultInfluxToken),
			Measurement:     stringFromEnv("IRACING_INFLUX_MEASUREMENT", defaultMeasurement),
			WriteTimeout:    writeTimeout,
			FlushInterval:   flushInterval,
			ChannelCapacity: channelCapacity,
			BatchSize:       batchSize,
		},
	}

	if cfg.PollInterval <= 0 {
		return Config{}, fmt.Errorf("IRACING_POLL_INTERVAL must be greater than zero")
	}
	if cfg.Influx.URL == "" {
		return Config{}, fmt.Errorf("IRACING_INFLUX_URL must not be empty")
	}
	if cfg.Influx.Org == "" {
		return Config{}, fmt.Errorf("IRACING_INFLUX_ORG must not be empty")
	}
	if cfg.Influx.Bucket == "" {
		return Config{}, fmt.Errorf("IRACING_INFLUX_BUCKET must not be empty")
	}
	if cfg.Influx.Token == "" {
		return Config{}, fmt.Errorf("IRACING_INFLUX_TOKEN must not be empty")
	}
	if cfg.Influx.Measurement == "" {
		return Config{}, fmt.Errorf("IRACING_INFLUX_MEASUREMENT must not be empty")
	}
	if cfg.Influx.WriteTimeout <= 0 {
		return Config{}, fmt.Errorf("IRACING_WRITE_TIMEOUT must be greater than zero")
	}
	if cfg.Influx.FlushInterval <= 0 {
		return Config{}, fmt.Errorf("IRACING_FLUSH_INTERVAL must be greater than zero")
	}
	if cfg.Influx.ChannelCapacity <= 0 {
		return Config{}, fmt.Errorf("IRACING_BUFFER_SIZE must be greater than zero")
	}
	if cfg.Influx.BatchSize <= 0 {
		return Config{}, fmt.Errorf("IRACING_BATCH_SIZE must be greater than zero")
	}

	return cfg, nil
}

func stringFromEnv(key string, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	return strings.TrimSpace(value)
}

func durationFromEnv(key string, fallback time.Duration) (time.Duration, error) {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return parsed, nil
}

func intFromEnv(key string, fallback int) (int, error) {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return parsed, nil
}
