package influxdb

import (
	"strings"
	"testing"
	"time"

	"github.com/joao/iracing-telemetry/internal/config"
	"github.com/joao/iracing-telemetry/internal/irsdk"
)

func TestLineProtocolIncludesExpandedLiveFields(t *testing.T) {
	client := &Client{
		cfg: config.InfluxConfig{
			Measurement: "iracing_telemetry",
		},
	}

	line := client.lineProtocol(irsdk.TelemetrySnapshot{
		Source:                   "live",
		SampleTime:               time.Unix(123, 456).UTC(),
		SpeedKPH:                 180.5,
		ThrottleRaw:              0.92,
		BrakeABSActive:           true,
		TrackWetness:             3,
		TrackXMeters:             12.5,
		TrackYMeters:             -9.75,
		HasTrackTrace:            true,
		PlayerTireCompound:       2,
		PushToPassActive:         true,
		PushToPassCount:          5,
		EngineWarnings:           0x0004,
		WeatherDeclaredWet:       true,
		LapDeltaToBestLapSeconds: -0.183,
	})

	for _, fragment := range []string{
		"status_code=1i",
		"speed_kph=180.5",
		"throttle_raw=0.92",
		"brake_abs_active=true",
		"track_wetness=3i",
		"track_x_m=12.5",
		"track_y_m=-9.75",
		"player_tire_compound=2i",
		"push_to_pass_active=true",
		"push_to_pass_count=5i",
		"engine_warnings=4i",
		"warning_oil_pressure=true",
		"weather_declared_wet=true",
		"lap_delta_to_best_lap_seconds=-0.183",
	} {
		if !strings.Contains(line, fragment) {
			t.Fatalf("expected %q in line protocol:\n%s", fragment, line)
		}
	}
}

func TestLineProtocolOmitsTelemetryFieldsWhenNotLive(t *testing.T) {
	client := &Client{
		cfg: config.InfluxConfig{
			Measurement: "iracing_telemetry",
		},
	}

	line := client.lineProtocol(irsdk.TelemetrySnapshot{
		Source:     "fallback",
		SampleTime: time.Unix(123, 0).UTC(),
		SpeedKPH:   180.5,
	})

	if !strings.Contains(line, "status_code=0i") {
		t.Fatalf("expected fallback status code in line protocol:\n%s", line)
	}
	if strings.Contains(line, "speed_kph=") {
		t.Fatalf("did not expect telemetry fields for non-live snapshot:\n%s", line)
	}
}
