package irsdk

import (
	"math"
	"time"
)

type trackTraceState struct {
	initialized       bool
	lastSessionNumber int
	lastSampleTime    time.Time
	xMeters           float64
	yMeters           float64
}

func (t *trackTraceState) Apply(snapshot *TelemetrySnapshot) {
	if snapshot == nil {
		return
	}

	if snapshot.Source != stateLive {
		t.Reset()
		return
	}

	if !t.initialized || t.lastSessionNumber != snapshot.SessionNumber {
		t.initialized = true
		t.lastSessionNumber = snapshot.SessionNumber
		t.lastSampleTime = snapshot.SampleTime
		t.xMeters = 0
		t.yMeters = 0
		snapshot.HasTrackTrace = true
		snapshot.TrackXMeters = 0
		snapshot.TrackYMeters = 0
		return
	}

	deltaSeconds := snapshot.SampleTime.Sub(t.lastSampleTime).Seconds()
	t.lastSampleTime = snapshot.SampleTime
	if deltaSeconds <= 0 || deltaSeconds > 2 {
		snapshot.HasTrackTrace = true
		snapshot.TrackXMeters = t.xMeters
		snapshot.TrackYMeters = t.yMeters
		return
	}

	if snapshot.IsOnTrack && !snapshot.IsInGarage {
		speedMps := snapshot.SpeedKPH / 3.6
		heading := normalizeAngle(snapshot.YawNorth)
		if speedMps > 0.5 && !math.IsNaN(heading) && !math.IsInf(heading, 0) {
			t.xMeters += math.Sin(heading) * speedMps * deltaSeconds
			t.yMeters += math.Cos(heading) * speedMps * deltaSeconds
		}
	}

	snapshot.HasTrackTrace = true
	snapshot.TrackXMeters = t.xMeters
	snapshot.TrackYMeters = t.yMeters
}

func (t *trackTraceState) Reset() {
	t.initialized = false
	t.lastSessionNumber = 0
	t.lastSampleTime = time.Time{}
	t.xMeters = 0
	t.yMeters = 0
}

func normalizeAngle(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return value
	}

	for value > math.Pi {
		value -= 2 * math.Pi
	}
	for value < -math.Pi {
		value += 2 * math.Pi
	}

	return value
}
