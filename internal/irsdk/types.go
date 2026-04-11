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
	{Name: "ThrottleRaw", Kind: variableFloat},
	{Name: "Brake", Kind: variableFloat, Required: true},
	{Name: "BrakeRaw", Kind: variableFloat},
	{Name: "BrakeABSactive", Kind: variableBool},
	{Name: "BrakeABSCutPct", Kind: variableFloat},
	{Name: "Clutch", Kind: variableFloat, Required: true},
	{Name: "SteeringWheelAngle", Kind: variableFloat},
	{Name: "SteeringWheelTorque", Kind: variableFloat},
	{Name: "SteeringWheelPctTorque", Kind: variableFloat},
	{Name: "Lap", Kind: variableInt},
	{Name: "LapCompleted", Kind: variableInt},
	{Name: "LapDist", Kind: variableFloat},
	{Name: "LapDistPct", Kind: variableFloat},
	{Name: "LapCurrentLapTime", Kind: variableFloat},
	{Name: "LapLastLapTime", Kind: variableFloat},
	{Name: "LapBestLapTime", Kind: variableFloat},
	{Name: "LapDeltaToBestLap", Kind: variableFloat},
	{Name: "LapDeltaToSessionBestLap", Kind: variableFloat},
	{Name: "LapDeltaToOptimalLap", Kind: variableFloat},
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
	{Name: "TrackWetness", Kind: variableInt},
	{Name: "AirTemp", Kind: variableFloat},
	{Name: "RelativeHumidity", Kind: variableFloat},
	{Name: "Precipitation", Kind: variableFloat},
	{Name: "WaterTemp", Kind: variableFloat},
	{Name: "OilTemp", Kind: variableFloat},
	{Name: "Voltage", Kind: variableFloat},
	{Name: "WindDir", Kind: variableFloat},
	{Name: "WindVel", Kind: variableFloat},
	{Name: "WeatherDeclaredWet", Kind: variableBool},
	{Name: "Lat", Kind: variableFloat},
	{Name: "Lon", Kind: variableFloat},
	{Name: "Alt", Kind: variableFloat},
	{Name: "LatAccel", Kind: variableFloat},
	{Name: "LongAccel", Kind: variableFloat},
	{Name: "VertAccel", Kind: variableFloat},
	{Name: "VelocityX", Kind: variableFloat},
	{Name: "VelocityY", Kind: variableFloat},
	{Name: "VelocityZ", Kind: variableFloat},
	{Name: "Yaw", Kind: variableFloat},
	{Name: "YawNorth", Kind: variableFloat},
	{Name: "YawRate", Kind: variableFloat},
	{Name: "Pitch", Kind: variableFloat},
	{Name: "Roll", Kind: variableFloat},
	{Name: "OnPitRoad", Kind: variableBool},
	{Name: "IsOnTrack", Kind: variableBool},
	{Name: "IsOnTrackCar", Kind: variableBool},
	{Name: "IsInGarage", Kind: variableBool},
	{Name: "PlayerTrackSurface", Kind: variableInt},
	{Name: "PlayerCarMyIncidentCount", Kind: variableInt},
	{Name: "PlayerCarPowerAdjust", Kind: variableInt},
	{Name: "PlayerTireCompound", Kind: variableInt},
	{Name: "PitSvTireCompound", Kind: variableInt},
	{Name: "PitSvLFP", Kind: variableFloat},
	{Name: "PitSvLRP", Kind: variableFloat},
	{Name: "PitSvRFP", Kind: variableFloat},
	{Name: "PitSvRRP", Kind: variableFloat},
	{Name: "TireSetsAvailable", Kind: variableInt},
	{Name: "TireSetsUsed", Kind: variableInt},
	{Name: "LeftTireSetsAvailable", Kind: variableInt},
	{Name: "LeftTireSetsUsed", Kind: variableInt},
	{Name: "RightTireSetsAvailable", Kind: variableInt},
	{Name: "RightTireSetsUsed", Kind: variableInt},
	{Name: "RearTireSetsAvailable", Kind: variableInt},
	{Name: "RearTireSetsUsed", Kind: variableInt},
	{Name: "PushToPass", Kind: variableBool},
	{Name: "P2P_Count", Kind: variableInt},
	{Name: "P2P_Status", Kind: variableInt},
	{Name: "EngineWarnings", Kind: variableInt},
}

type TelemetrySnapshot struct {
	SpeedKPH                     float64   `json:"speedKph"`
	SpeedMPH                     float64   `json:"speedMph"`
	RPM                          float64   `json:"rpm"`
	Gear                         int       `json:"gear"`
	Throttle                     float64   `json:"throttle"`
	ThrottleRaw                  float64   `json:"throttleRaw"`
	Brake                        float64   `json:"brake"`
	BrakeRaw                     float64   `json:"brakeRaw"`
	BrakeABSActive               bool      `json:"brakeAbsActive"`
	BrakeABSCutPct               float64   `json:"brakeAbsCutPct"`
	Clutch                       float64   `json:"clutch"`
	SteeringWheelAngle           float64   `json:"steeringWheelAngle"`
	SteeringWheelTorque          float64   `json:"steeringWheelTorque"`
	SteeringWheelPctTorque       float64   `json:"steeringWheelPctTorque"`
	CurrentLap                   int       `json:"currentLap"`
	CompletedLaps                int       `json:"completedLaps"`
	LapDistanceMeters            float64   `json:"lapDistanceMeters"`
	LapDistancePct               float64   `json:"lapDistancePct"`
	CurrentLapTimeSeconds        float64   `json:"currentLapTimeSeconds"`
	LastLapTimeSeconds           float64   `json:"lastLapTimeSeconds"`
	BestLapTimeSeconds           float64   `json:"bestLapTimeSeconds"`
	LapDeltaToBestLapSeconds     float64   `json:"lapDeltaToBestLapSeconds"`
	LapDeltaToSessionBestSeconds float64   `json:"lapDeltaToSessionBestSeconds"`
	LapDeltaToOptimalLapSeconds  float64   `json:"lapDeltaToOptimalLapSeconds"`
	SessionNumber                int       `json:"sessionNumber"`
	SessionState                 int       `json:"sessionState"`
	SessionFlags                 int       `json:"sessionFlags"`
	SessionTimeSeconds           float64   `json:"sessionTimeSeconds"`
	SessionTimeRemainingSeconds  float64   `json:"sessionTimeRemainingSeconds"`
	SessionLapsRemaining         float64   `json:"sessionLapsRemaining"`
	Position                     int       `json:"position"`
	ClassPosition                int       `json:"classPosition"`
	FuelLevelLiters              float64   `json:"fuelLevelLiters"`
	FuelLevelPct                 float64   `json:"fuelLevelPct"`
	FuelUsePerHour               float64   `json:"fuelUsePerHour"`
	TrackTempC                   float64   `json:"trackTempC"`
	TrackTempCrewC               float64   `json:"trackTempCrewC"`
	TrackWetness                 int       `json:"trackWetness"`
	AirTempC                     float64   `json:"airTempC"`
	RelativeHumidityPct          float64   `json:"relativeHumidityPct"`
	PrecipitationPct             float64   `json:"precipitationPct"`
	WaterTempC                   float64   `json:"waterTempC"`
	OilTempC                     float64   `json:"oilTempC"`
	Voltage                      float64   `json:"voltage"`
	WindDirectionRad             float64   `json:"windDirectionRad"`
	WindVelocityMps              float64   `json:"windVelocityMps"`
	WeatherDeclaredWet           bool      `json:"weatherDeclaredWet"`
	LatitudeDeg                  float64   `json:"latitudeDeg"`
	LongitudeDeg                 float64   `json:"longitudeDeg"`
	AltitudeMeters               float64   `json:"altitudeMeters"`
	HasLocation                  bool      `json:"hasLocation"`
	LatAccel                     float64   `json:"latAccel"`
	LongAccel                    float64   `json:"longAccel"`
	VertAccel                    float64   `json:"vertAccel"`
	VelocityX                    float64   `json:"velocityX"`
	VelocityY                    float64   `json:"velocityY"`
	VelocityZ                    float64   `json:"velocityZ"`
	Yaw                          float64   `json:"yaw"`
	YawNorth                     float64   `json:"yawNorth"`
	YawRate                      float64   `json:"yawRate"`
	Pitch                        float64   `json:"pitch"`
	Roll                         float64   `json:"roll"`
	TrackXMeters                 float64   `json:"trackXMeters"`
	TrackYMeters                 float64   `json:"trackYMeters"`
	HasTrackTrace                bool      `json:"hasTrackTrace"`
	OnPitRoad                    bool      `json:"onPitRoad"`
	IsOnTrack                    bool      `json:"isOnTrack"`
	IsInGarage                   bool      `json:"isInGarage"`
	TrackSurface                 int       `json:"trackSurface"`
	Incidents                    int       `json:"incidents"`
	PlayerCarPowerAdjust         int       `json:"playerCarPowerAdjust"`
	PlayerTireCompound           int       `json:"playerTireCompound"`
	PitServiceTireCompound       int       `json:"pitServiceTireCompound"`
	PitServiceLFPressure         float64   `json:"pitServiceLfPressure"`
	PitServiceLRPressure         float64   `json:"pitServiceLrPressure"`
	PitServiceRFPressure         float64   `json:"pitServiceRfPressure"`
	PitServiceRRPressure         float64   `json:"pitServiceRrPressure"`
	TireSetsAvailable            int       `json:"tireSetsAvailable"`
	TireSetsUsed                 int       `json:"tireSetsUsed"`
	LeftTireSetsAvailable        int       `json:"leftTireSetsAvailable"`
	LeftTireSetsUsed             int       `json:"leftTireSetsUsed"`
	RightTireSetsAvailable       int       `json:"rightTireSetsAvailable"`
	RightTireSetsUsed            int       `json:"rightTireSetsUsed"`
	RearTireSetsAvailable        int       `json:"rearTireSetsAvailable"`
	RearTireSetsUsed             int       `json:"rearTireSetsUsed"`
	PushToPassActive             bool      `json:"pushToPassActive"`
	PushToPassCount              int       `json:"pushToPassCount"`
	PushToPassStatus             int       `json:"pushToPassStatus"`
	EngineWarnings               int       `json:"engineWarnings"`
	Source                       string    `json:"source"`
	SampleTime                   time.Time `json:"sampleTime"`
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

func (s TelemetrySnapshot) WaterTempWarning() bool {
	return hasFlag(s.EngineWarnings, 0x0001)
}

func (s TelemetrySnapshot) FuelPressureWarning() bool {
	return hasFlag(s.EngineWarnings, 0x0002)
}

func (s TelemetrySnapshot) OilPressureWarning() bool {
	return hasFlag(s.EngineWarnings, 0x0004)
}

func (s TelemetrySnapshot) EngineStalled() bool {
	return hasFlag(s.EngineWarnings, 0x0008)
}

func (s TelemetrySnapshot) PitLimiterActive() bool {
	return hasFlag(s.EngineWarnings, 0x0010)
}

func (s TelemetrySnapshot) OilTempWarning() bool {
	return hasFlag(s.EngineWarnings, 0x0040)
}

func hasFlag(flags int, flag uint32) bool {
	return uint32(flags)&flag != 0
}
