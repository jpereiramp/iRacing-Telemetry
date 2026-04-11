package irsdk

import (
	"math"
	"testing"
	"time"
)

func TestTrackTraceStateAccumulatesHeadingBasedPosition(t *testing.T) {
	trace := &trackTraceState{}
	start := time.Unix(100, 0).UTC()

	first := TelemetrySnapshot{
		Source:        stateLive,
		SessionNumber: 3,
		SampleTime:    start,
		SpeedKPH:      72,
		YawNorth:      math.Pi / 2,
		IsOnTrack:     true,
	}
	trace.Apply(&first)

	second := TelemetrySnapshot{
		Source:        stateLive,
		SessionNumber: 3,
		SampleTime:    start.Add(time.Second),
		SpeedKPH:      72,
		YawNorth:      math.Pi / 2,
		IsOnTrack:     true,
	}
	trace.Apply(&second)

	if !second.HasTrackTrace {
		t.Fatalf("expected track trace to be populated")
	}
	if diff := math.Abs(second.TrackXMeters - 20); diff > 0.001 {
		t.Fatalf("expected x position near 20m, got %f", second.TrackXMeters)
	}
	if diff := math.Abs(second.TrackYMeters); diff > 0.001 {
		t.Fatalf("expected y position near 0m, got %f", second.TrackYMeters)
	}
}

func TestTrackTraceStateResetsWhenSourceDropsOut(t *testing.T) {
	trace := &trackTraceState{}
	start := time.Unix(100, 0).UTC()

	live := TelemetrySnapshot{
		Source:        stateLive,
		SessionNumber: 1,
		SampleTime:    start,
		SpeedKPH:      36,
		YawNorth:      0,
		IsOnTrack:     true,
	}
	trace.Apply(&live)

	lost := TelemetrySnapshot{
		Source:     stateDisconnected,
		SampleTime: start.Add(time.Second),
	}
	trace.Apply(&lost)

	if trace.initialized {
		t.Fatalf("expected track trace state to reset on disconnect")
	}
}
