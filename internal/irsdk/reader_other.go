//go:build !windows

package irsdk

import "time"

type Reader struct{}

func NewReader() (*Reader, error) {
	return &Reader{}, nil
}

func (r *Reader) Close() {}

func (r *Reader) ReadSnapshot() TelemetrySnapshot {
	return TelemetrySnapshot{
		Source:     stateDisconnected,
		SampleTime: time.Now().UTC(),
	}
}
