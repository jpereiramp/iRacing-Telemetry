package irsdk

import "time"

const (
	irsdkMapName       = "Local\\IRSDKMemMapFileName"
	irsdkMapNameGlobal = "IRSDKMemMapFileName"

	headerSize = 112
	varHdrSize = 144

	stateDisconnected = "disconnected"
	stateFallback     = "fallback"
	stateLive         = "live"

	irsdkTypeBool     = 1
	irsdkTypeInt32    = 2
	irsdkTypeBitField = 3
	irsdkTypeFloat32  = 4
	irsdkTypeFloat64  = 5
)

const (
	sessionFlagCheckered uint32 = 0x00000001
	sessionFlagGreen     uint32 = 0x00000004
	sessionFlagYellow    uint32 = 0x00000008
	sessionFlagBlue      uint32 = 0x00000020
	sessionFlagCaution   uint32 = 0x00004000
	sessionFlagBlack     uint32 = 0x00010000
)

type variableKind int

const (
	variableFloat variableKind = iota
	variableInt
	variableBool
)

type variableDefinition struct {
	Name     string
	Kind     variableKind
	Required bool
}

var telemetryDefinitions = []variableDefinition{
	{Name: "Speed", Kind: variableFloat, Required: true},
	{Name: "RPM", Kind: variableFloat, Required: true},
	{Name: "Gear", Kind: variableInt, Required: true},
	{Name: "Throttle", Kind: variableFloat, Required: true},
	{Name: "Brake", Kind: variableFloat, Required: true},
	{Name: "Clutch", Kind: variableFloat, Required: true},
	{Name: "SteeringWheelAngle", Kind: variableFloat},
	{Name: "Lap", Kind: variableInt},
	{Name: "LapCompleted", Kind: variableInt},
	{Name: "LapDistPct", Kind: variableFloat},
	{Name: "LapCurrentLapTime", Kind: variableFloat},
	{Name: "LapLastLapTime", Kind: variableFloat},
	{Name: "LapBestLapTime", Kind: variableFloat},
	{Name: "SessionNum", Kind: variableInt},
	{Name: "SessionState", Kind: variableInt},
	{Name: "SessionFlags", Kind: variableInt},
	{Name: "SessionTime", Kind: variableFloat},
	{Name: "SessionTimeRemain", Kind: variableFloat},
	{Name: "SessionLapsRemain", Kind: variableFloat},
	{Name: "PlayerCarPosition", Kind: variableInt},
	{Name: "PlayerCarClassPosition", Kind: variableInt},
	{Name: "FuelLevel", Kind: variableFloat},
	{Name: "FuelLevelPct", Kind: variableFloat},
	{Name: "FuelUsePerHour", Kind: variableFloat},
	{Name: "TrackTemp", Kind: variableFloat},
	{Name: "TrackTempCrew", Kind: variableFloat},
	{Name: "AirTemp", Kind: variableFloat},
	{Name: "WaterTemp", Kind: variableFloat},
	{Name: "OilTemp", Kind: variableFloat},
	{Name: "Voltage", Kind: variableFloat},
	{Name: "Lat", Kind: variableFloat},
	{Name: "Lon", Kind: variableFloat},
	{Name: "Alt", Kind: variableFloat},
	{Name: "OnPitRoad", Kind: variableBool},
	{Name: "IsOnTrack", Kind: variableBool},
	{Name: "IsOnTrackCar", Kind: variableBool},
	{Name: "IsInGarage", Kind: variableBool},
	{Name: "PlayerTrackSurface", Kind: variableInt},
	{Name: "PlayerCarMyIncidentCount", Kind: variableInt},
}

type TelemetrySnapshot struct {
	SpeedKPH                    float64   `json:"speedKph"`
	SpeedMPH                    float64   `json:"speedMph"`
	RPM                         float64   `json:"rpm"`
	Gear                        int       `json:"gear"`
	Throttle                    float64   `json:"throttle"`
	Brake                       float64   `json:"brake"`
	Clutch                      float64   `json:"clutch"`
	SteeringWheelAngle          float64   `json:"steeringWheelAngle"`
	CurrentLap                  int       `json:"currentLap"`
	CompletedLaps               int       `json:"completedLaps"`
	LapDistancePct              float64   `json:"lapDistancePct"`
	CurrentLapTimeSeconds       float64   `json:"currentLapTimeSeconds"`
	LastLapTimeSeconds          float64   `json:"lastLapTimeSeconds"`
	BestLapTimeSeconds          float64   `json:"bestLapTimeSeconds"`
	SessionNumber               int       `json:"sessionNumber"`
	SessionState                int       `json:"sessionState"`
	SessionFlags                int       `json:"sessionFlags"`
	SessionTimeSeconds          float64   `json:"sessionTimeSeconds"`
	SessionTimeRemainingSeconds float64   `json:"sessionTimeRemainingSeconds"`
	SessionLapsRemaining        float64   `json:"sessionLapsRemaining"`
	Position                    int       `json:"position"`
	ClassPosition               int       `json:"classPosition"`
	FuelLevelLiters             float64   `json:"fuelLevelLiters"`
	FuelLevelPct                float64   `json:"fuelLevelPct"`
	FuelUsePerHour              float64   `json:"fuelUsePerHour"`
	TrackTempC                  float64   `json:"trackTempC"`
	TrackTempCrewC              float64   `json:"trackTempCrewC"`
	AirTempC                    float64   `json:"airTempC"`
	WaterTempC                  float64   `json:"waterTempC"`
	OilTempC                    float64   `json:"oilTempC"`
	Voltage                     float64   `json:"voltage"`
	LatitudeDeg                 float64   `json:"latitudeDeg"`
	LongitudeDeg                float64   `json:"longitudeDeg"`
	AltitudeMeters              float64   `json:"altitudeMeters"`
	HasLocation                 bool      `json:"hasLocation"`
	OnPitRoad                   bool      `json:"onPitRoad"`
	IsOnTrack                   bool      `json:"isOnTrack"`
	IsInGarage                  bool      `json:"isInGarage"`
	TrackSurface                int       `json:"trackSurface"`
	Incidents                   int       `json:"incidents"`
	Source                      string    `json:"source"`
	SampleTime                  time.Time `json:"sampleTime"`
}

func (s TelemetrySnapshot) SourceCode() int {
	switch s.Source {
	case stateLive:
		return 1
	case stateFallback:
		return 0
	default:
		return -1
	}
}

func (s TelemetrySnapshot) FlagGreen() bool {
	return hasFlag(s.SessionFlags, sessionFlagGreen)
}

func (s TelemetrySnapshot) FlagYellow() bool {
	return hasFlag(s.SessionFlags, sessionFlagYellow)
}

func (s TelemetrySnapshot) FlagBlue() bool {
	return hasFlag(s.SessionFlags, sessionFlagBlue)
}

func (s TelemetrySnapshot) FlagBlack() bool {
	return hasFlag(s.SessionFlags, sessionFlagBlack)
}

func (s TelemetrySnapshot) FlagCheckered() bool {
	return hasFlag(s.SessionFlags, sessionFlagCheckered)
}

func (s TelemetrySnapshot) FlagCaution() bool {
	return hasFlag(s.SessionFlags, sessionFlagCaution)
}

func hasFlag(flags int, flag uint32) bool {
	return uint32(flags)&flag != 0
}
