package influxdb

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joao/iracing-telemetry/internal/config"
	"github.com/joao/iracing-telemetry/internal/irsdk"
)

var errWriterQueueFull = errors.New("influxdb writer queue is full")

type Client struct {
	cfg        config.InfluxConfig
	logger     *log.Logger
	httpClient *http.Client
	queue      chan irsdk.TelemetrySnapshot
	done       chan struct{}

	closeOnce sync.Once

	stateMu      sync.Mutex
	lastState    string
	lastStateMsg string
}

func NewClient(cfg config.InfluxConfig, logger *log.Logger) *Client {
	client := &Client{
		cfg:        cfg,
		logger:     logger,
		httpClient: &http.Client{Timeout: cfg.WriteTimeout},
		queue:      make(chan irsdk.TelemetrySnapshot, cfg.ChannelCapacity),
		done:       make(chan struct{}),
	}

	go client.run()

	return client
}

func (c *Client) Write(ctx context.Context, snapshot irsdk.TelemetrySnapshot) error {
	select {
	case c.queue <- snapshot:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return errWriterQueueFull
	}
}

func (c *Client) Close(ctx context.Context) error {
	c.closeOnce.Do(func() {
		close(c.queue)
	})

	select {
	case <-c.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Client) run() {
	defer close(c.done)

	flushTicker := time.NewTicker(c.cfg.FlushInterval)
	defer flushTicker.Stop()

	batch := make([]irsdk.TelemetrySnapshot, 0, c.cfg.BatchSize)
	for {
		select {
		case snapshot, ok := <-c.queue:
			if !ok {
				c.flushBatch(batch)
				return
			}

			batch = append(batch, snapshot)
			if len(batch) >= c.cfg.BatchSize {
				c.flushBatch(batch)
				batch = batch[:0]
			}
		case <-flushTicker.C:
			if len(batch) == 0 {
				continue
			}
			c.flushBatch(batch)
			batch = batch[:0]
		}
	}
}

func (c *Client) flushBatch(batch []irsdk.TelemetrySnapshot) {
	if len(batch) == 0 {
		return
	}

	body := bytes.NewBuffer(make([]byte, 0, len(batch)*512))
	for index, snapshot := range batch {
		if index > 0 {
			body.WriteByte('\n')
		}
		body.WriteString(c.lineProtocol(snapshot))
	}

	writeURL, err := c.writeURL()
	if err != nil {
		c.logWriteState("error", err.Error())
		return
	}

	req, err := http.NewRequest(http.MethodPost, writeURL, body)
	if err != nil {
		c.logWriteState("error", err.Error())
		return
	}
	req.Header.Set("Authorization", "Token "+c.cfg.Token)
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logWriteState("error", err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		c.logWriteState("error", fmt.Sprintf("unexpected status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody))))
		return
	}

	c.logWriteState("healthy", "batched writes are succeeding")
}

func (c *Client) writeURL() (string, error) {
	baseURL, err := url.Parse(c.cfg.URL)
	if err != nil {
		return "", err
	}

	baseURL.Path = "/api/v2/write"
	query := baseURL.Query()
	query.Set("org", c.cfg.Org)
	query.Set("bucket", c.cfg.Bucket)
	query.Set("precision", "ns")
	baseURL.RawQuery = query.Encode()

	return baseURL.String(), nil
}

func (c *Client) lineProtocol(snapshot irsdk.TelemetrySnapshot) string {
	var builder strings.Builder
	builder.Grow(768)

	builder.WriteString(escapeMeasurement(c.cfg.Measurement))
	builder.WriteString(",source=")
	builder.WriteString(escapeTag(snapshot.Source))
	builder.WriteByte(' ')

	fieldCount := 0
	appendIntField(&builder, &fieldCount, "status_code", int64(snapshot.SourceCode()))
	if snapshot.Source != "live" {
		builder.WriteByte(' ')
		builder.WriteString(strconv.FormatInt(snapshot.SampleTime.UnixNano(), 10))
		return builder.String()
	}

	appendFloatField(&builder, &fieldCount, "speed_kph", snapshot.SpeedKPH)
	appendFloatField(&builder, &fieldCount, "speed_mph", snapshot.SpeedMPH)
	appendFloatField(&builder, &fieldCount, "rpm", snapshot.RPM)
	appendIntField(&builder, &fieldCount, "gear", int64(snapshot.Gear))
	appendFloatField(&builder, &fieldCount, "throttle", snapshot.Throttle)
	appendFloatField(&builder, &fieldCount, "throttle_raw", snapshot.ThrottleRaw)
	appendFloatField(&builder, &fieldCount, "brake", snapshot.Brake)
	appendFloatField(&builder, &fieldCount, "brake_raw", snapshot.BrakeRaw)
	appendBoolField(&builder, &fieldCount, "brake_abs_active", snapshot.BrakeABSActive)
	appendFloatField(&builder, &fieldCount, "brake_abs_cut_pct", snapshot.BrakeABSCutPct)
	appendFloatField(&builder, &fieldCount, "clutch", snapshot.Clutch)
	appendFloatField(&builder, &fieldCount, "steering_wheel_angle", snapshot.SteeringWheelAngle)
	appendFloatField(&builder, &fieldCount, "steering_wheel_torque", snapshot.SteeringWheelTorque)
	appendFloatField(&builder, &fieldCount, "steering_wheel_pct_torque", snapshot.SteeringWheelPctTorque)
	appendIntField(&builder, &fieldCount, "current_lap", int64(snapshot.CurrentLap))
	appendIntField(&builder, &fieldCount, "completed_laps", int64(snapshot.CompletedLaps))
	appendFloatField(&builder, &fieldCount, "lap_distance_m", snapshot.LapDistanceMeters)
	appendFloatField(&builder, &fieldCount, "lap_distance_pct", snapshot.LapDistancePct)
	appendFloatField(&builder, &fieldCount, "current_lap_time_seconds", snapshot.CurrentLapTimeSeconds)
	appendFloatField(&builder, &fieldCount, "last_lap_time_seconds", snapshot.LastLapTimeSeconds)
	appendFloatField(&builder, &fieldCount, "best_lap_time_seconds", snapshot.BestLapTimeSeconds)
	appendFloatField(&builder, &fieldCount, "lap_delta_to_best_lap_seconds", snapshot.LapDeltaToBestLapSeconds)
	appendFloatField(&builder, &fieldCount, "lap_delta_to_session_best_seconds", snapshot.LapDeltaToSessionBestSeconds)
	appendFloatField(&builder, &fieldCount, "lap_delta_to_optimal_lap_seconds", snapshot.LapDeltaToOptimalLapSeconds)
	appendIntField(&builder, &fieldCount, "session_number", int64(snapshot.SessionNumber))
	appendIntField(&builder, &fieldCount, "session_state", int64(snapshot.SessionState))
	appendIntField(&builder, &fieldCount, "session_flags", int64(snapshot.SessionFlags))
	appendFloatField(&builder, &fieldCount, "session_time_seconds", snapshot.SessionTimeSeconds)
	appendFloatField(&builder, &fieldCount, "session_time_remaining_seconds", snapshot.SessionTimeRemainingSeconds)
	appendFloatField(&builder, &fieldCount, "session_laps_remaining", snapshot.SessionLapsRemaining)
	appendIntField(&builder, &fieldCount, "position_overall", int64(snapshot.Position))
	appendIntField(&builder, &fieldCount, "position_class", int64(snapshot.ClassPosition))
	appendFloatField(&builder, &fieldCount, "fuel_level_liters", snapshot.FuelLevelLiters)
	appendFloatField(&builder, &fieldCount, "fuel_level_pct", snapshot.FuelLevelPct)
	appendFloatField(&builder, &fieldCount, "fuel_use_per_hour", snapshot.FuelUsePerHour)
	appendFloatField(&builder, &fieldCount, "track_temp_c", snapshot.TrackTempC)
	appendFloatField(&builder, &fieldCount, "track_temp_crew_c", snapshot.TrackTempCrewC)
	appendIntField(&builder, &fieldCount, "track_wetness", int64(snapshot.TrackWetness))
	appendFloatField(&builder, &fieldCount, "air_temp_c", snapshot.AirTempC)
	appendFloatField(&builder, &fieldCount, "relative_humidity_pct", snapshot.RelativeHumidityPct)
	appendFloatField(&builder, &fieldCount, "precipitation_pct", snapshot.PrecipitationPct)
	appendFloatField(&builder, &fieldCount, "water_temp_c", snapshot.WaterTempC)
	appendFloatField(&builder, &fieldCount, "oil_temp_c", snapshot.OilTempC)
	appendFloatField(&builder, &fieldCount, "voltage", snapshot.Voltage)
	appendFloatField(&builder, &fieldCount, "wind_dir_rad", snapshot.WindDirectionRad)
	appendFloatField(&builder, &fieldCount, "wind_vel_mps", snapshot.WindVelocityMps)
	appendBoolField(&builder, &fieldCount, "weather_declared_wet", snapshot.WeatherDeclaredWet)
	appendBoolField(&builder, &fieldCount, "on_pit_road", snapshot.OnPitRoad)
	appendBoolField(&builder, &fieldCount, "is_on_track", snapshot.IsOnTrack)
	appendBoolField(&builder, &fieldCount, "is_in_garage", snapshot.IsInGarage)
	appendIntField(&builder, &fieldCount, "track_surface", int64(snapshot.TrackSurface))
	appendIntField(&builder, &fieldCount, "incidents", int64(snapshot.Incidents))
	appendFloatField(&builder, &fieldCount, "lat_accel", snapshot.LatAccel)
	appendFloatField(&builder, &fieldCount, "long_accel", snapshot.LongAccel)
	appendFloatField(&builder, &fieldCount, "vert_accel", snapshot.VertAccel)
	appendFloatField(&builder, &fieldCount, "velocity_x", snapshot.VelocityX)
	appendFloatField(&builder, &fieldCount, "velocity_y", snapshot.VelocityY)
	appendFloatField(&builder, &fieldCount, "velocity_z", snapshot.VelocityZ)
	appendFloatField(&builder, &fieldCount, "yaw", snapshot.Yaw)
	appendFloatField(&builder, &fieldCount, "yaw_north", snapshot.YawNorth)
	appendFloatField(&builder, &fieldCount, "yaw_rate", snapshot.YawRate)
	appendFloatField(&builder, &fieldCount, "pitch", snapshot.Pitch)
	appendFloatField(&builder, &fieldCount, "roll", snapshot.Roll)
	appendIntField(&builder, &fieldCount, "player_car_power_adjust", int64(snapshot.PlayerCarPowerAdjust))
	appendIntField(&builder, &fieldCount, "player_tire_compound", int64(snapshot.PlayerTireCompound))
	appendIntField(&builder, &fieldCount, "pit_service_tire_compound", int64(snapshot.PitServiceTireCompound))
	appendFloatField(&builder, &fieldCount, "pit_service_lf_pressure", snapshot.PitServiceLFPressure)
	appendFloatField(&builder, &fieldCount, "pit_service_lr_pressure", snapshot.PitServiceLRPressure)
	appendFloatField(&builder, &fieldCount, "pit_service_rf_pressure", snapshot.PitServiceRFPressure)
	appendFloatField(&builder, &fieldCount, "pit_service_rr_pressure", snapshot.PitServiceRRPressure)
	appendIntField(&builder, &fieldCount, "tire_sets_available", int64(snapshot.TireSetsAvailable))
	appendIntField(&builder, &fieldCount, "tire_sets_used", int64(snapshot.TireSetsUsed))
	appendIntField(&builder, &fieldCount, "left_tire_sets_available", int64(snapshot.LeftTireSetsAvailable))
	appendIntField(&builder, &fieldCount, "left_tire_sets_used", int64(snapshot.LeftTireSetsUsed))
	appendIntField(&builder, &fieldCount, "right_tire_sets_available", int64(snapshot.RightTireSetsAvailable))
	appendIntField(&builder, &fieldCount, "right_tire_sets_used", int64(snapshot.RightTireSetsUsed))
	appendIntField(&builder, &fieldCount, "rear_tire_sets_available", int64(snapshot.RearTireSetsAvailable))
	appendIntField(&builder, &fieldCount, "rear_tire_sets_used", int64(snapshot.RearTireSetsUsed))
	appendBoolField(&builder, &fieldCount, "push_to_pass_active", snapshot.PushToPassActive)
	appendIntField(&builder, &fieldCount, "push_to_pass_count", int64(snapshot.PushToPassCount))
	appendIntField(&builder, &fieldCount, "push_to_pass_status", int64(snapshot.PushToPassStatus))
	appendIntField(&builder, &fieldCount, "engine_warnings", int64(snapshot.EngineWarnings))
	appendBoolField(&builder, &fieldCount, "flag_green", snapshot.FlagGreen())
	appendBoolField(&builder, &fieldCount, "flag_yellow", snapshot.FlagYellow())
	appendBoolField(&builder, &fieldCount, "flag_blue", snapshot.FlagBlue())
	appendBoolField(&builder, &fieldCount, "flag_black", snapshot.FlagBlack())
	appendBoolField(&builder, &fieldCount, "flag_checkered", snapshot.FlagCheckered())
	appendBoolField(&builder, &fieldCount, "flag_caution", snapshot.FlagCaution())
	appendBoolField(&builder, &fieldCount, "warning_water_temp", snapshot.WaterTempWarning())
	appendBoolField(&builder, &fieldCount, "warning_fuel_pressure", snapshot.FuelPressureWarning())
	appendBoolField(&builder, &fieldCount, "warning_oil_pressure", snapshot.OilPressureWarning())
	appendBoolField(&builder, &fieldCount, "warning_engine_stalled", snapshot.EngineStalled())
	appendBoolField(&builder, &fieldCount, "warning_pit_limiter", snapshot.PitLimiterActive())
	appendBoolField(&builder, &fieldCount, "warning_oil_temp", snapshot.OilTempWarning())

	if snapshot.HasLocation {
		appendFloatField(&builder, &fieldCount, "latitude_deg", snapshot.LatitudeDeg)
		appendFloatField(&builder, &fieldCount, "longitude_deg", snapshot.LongitudeDeg)
		appendFloatField(&builder, &fieldCount, "altitude_m", snapshot.AltitudeMeters)
	}
	if snapshot.HasTrackTrace {
		appendFloatField(&builder, &fieldCount, "track_x_m", snapshot.TrackXMeters)
		appendFloatField(&builder, &fieldCount, "track_y_m", snapshot.TrackYMeters)
	}

	builder.WriteByte(' ')
	builder.WriteString(strconv.FormatInt(snapshot.SampleTime.UnixNano(), 10))

	return builder.String()
}

func (c *Client) logWriteState(state string, message string) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	if c.lastState == state && c.lastStateMsg == message {
		return
	}

	c.lastState = state
	c.lastStateMsg = message

	if c.logger != nil {
		c.logger.Printf("influxdb state=%s detail=%s", state, message)
	}
}

func appendFloatField(builder *strings.Builder, fieldCount *int, key string, value float64) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return
	}
	appendFieldPrefix(builder, fieldCount)
	builder.WriteString(escapeFieldKey(key))
	builder.WriteByte('=')
	builder.WriteString(strconv.FormatFloat(value, 'f', -1, 64))
}

func appendIntField(builder *strings.Builder, fieldCount *int, key string, value int64) {
	appendFieldPrefix(builder, fieldCount)
	builder.WriteString(escapeFieldKey(key))
	builder.WriteByte('=')
	builder.WriteString(strconv.FormatInt(value, 10))
	builder.WriteByte('i')
}

func appendBoolField(builder *strings.Builder, fieldCount *int, key string, value bool) {
	appendFieldPrefix(builder, fieldCount)
	builder.WriteString(escapeFieldKey(key))
	builder.WriteByte('=')
	if value {
		builder.WriteString("true")
		return
	}
	builder.WriteString("false")
}

func appendFieldPrefix(builder *strings.Builder, fieldCount *int) {
	if *fieldCount > 0 {
		builder.WriteByte(',')
	}
	*fieldCount++
}

func escapeMeasurement(value string) string {
	return strings.NewReplacer(",", "\\,", " ", "\\ ").Replace(value)
}

func escapeTag(value string) string {
	return strings.NewReplacer(",", "\\,", " ", "\\ ", "=", "\\=").Replace(value)
}

func escapeFieldKey(value string) string {
	return strings.NewReplacer(",", "\\,", " ", "\\ ", "=", "\\=").Replace(value)
}
