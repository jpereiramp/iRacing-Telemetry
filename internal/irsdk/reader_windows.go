//go:build windows

package irsdk

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

type variableRef struct {
	Offset int32
	Type   int32
	Count  int32
}

type Reader struct {
	mu              sync.RWMutex
	fileHandle      syscall.Handle
	mappingAddr     uintptr
	mappingBytes    []byte
	mapSize         uint32
	variables       map[string]variableRef
	lastState       string
	lastStateReason string
	logger          *log.Logger
	trackTrace      trackTraceState
}

func NewReader() (*Reader, error) {
	reader := &Reader{
		variables: make(map[string]variableRef),
		logger:    log.New(log.Writer(), "[irsdk] ", log.LstdFlags|log.Lmicroseconds),
	}

	_ = reader.tryConnectSharedMemory()

	return reader, nil
}

func (r *Reader) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.disconnectLocked()
}

func (r *Reader) ReadSnapshot() TelemetrySnapshot {
	r.mu.RLock()
	connected := len(r.mappingBytes) > 0
	r.mu.RUnlock()

	if !connected {
		if err := r.tryConnectSharedMemory(); err != nil {
			r.mu.Lock()
			defer r.mu.Unlock()
			return r.fallbackSnapshotLocked(stateDisconnected, err.Error())
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.resolveVariablesLocked(); err != nil {
		r.disconnectLocked()
		return r.fallbackSnapshotLocked(stateFallback, err.Error())
	}

	speedMps, ok := r.readFloatVariableLocked("Speed")
	if !ok {
		r.disconnectLocked()
		return r.fallbackSnapshotLocked(stateFallback, "failed to read speed from latest buffer")
	}

	sampleTime := time.Now().UTC()
	snapshot := TelemetrySnapshot{
		SpeedKPH:                     clampNonNegative(speedMps) * 3.6,
		SpeedMPH:                     clampNonNegative(speedMps) * 2.2369362920544,
		RPM:                          clampNonNegative(r.readFloatVariableOrZeroLocked("RPM")),
		Gear:                         clampGear(int(r.readIntVariableOrZeroLocked("Gear"))),
		Throttle:                     clamp01(r.readFloatVariableOrZeroLocked("Throttle")),
		ThrottleRaw:                  clamp01(r.readFloatVariableOrZeroLocked("ThrottleRaw")),
		Brake:                        clamp01(r.readFloatVariableOrZeroLocked("Brake")),
		BrakeRaw:                     clamp01(r.readFloatVariableOrZeroLocked("BrakeRaw")),
		BrakeABSActive:               r.readBoolVariableOrFalseLocked("BrakeABSactive"),
		BrakeABSCutPct:               clamp01(r.readFloatVariableOrZeroLocked("BrakeABSCutPct")),
		Clutch:                       clamp01(r.readFloatVariableOrZeroLocked("Clutch")),
		SteeringWheelAngle:           r.readFloatVariableOrZeroLocked("SteeringWheelAngle"),
		SteeringWheelTorque:          r.readFloatVariableOrZeroLocked("SteeringWheelTorque"),
		SteeringWheelPctTorque:       r.readFloatVariableOrZeroLocked("SteeringWheelPctTorque"),
		CurrentLap:                   int(r.readIntVariableOrZeroLocked("Lap")),
		CompletedLaps:                int(r.readIntVariableOrZeroLocked("LapCompleted")),
		LapDistanceMeters:            clampNonNegative(r.readFloatVariableOrZeroLocked("LapDist")),
		LapDistancePct:               clamp01(r.readFloatVariableOrZeroLocked("LapDistPct")),
		CurrentLapTimeSeconds:        clampNonNegative(r.readFloatVariableOrZeroLocked("LapCurrentLapTime")),
		LastLapTimeSeconds:           clampNonNegative(r.readFloatVariableOrZeroLocked("LapLastLapTime")),
		BestLapTimeSeconds:           clampNonNegative(r.readFloatVariableOrZeroLocked("LapBestLapTime")),
		LapDeltaToBestLapSeconds:     r.readFloatVariableOrZeroLocked("LapDeltaToBestLap"),
		LapDeltaToSessionBestSeconds: r.readFloatVariableOrZeroLocked("LapDeltaToSessionBestLap"),
		LapDeltaToOptimalLapSeconds:  r.readFloatVariableOrZeroLocked("LapDeltaToOptimalLap"),
		SessionNumber:                int(r.readIntVariableOrZeroLocked("SessionNum")),
		SessionState:                 int(r.readIntVariableOrZeroLocked("SessionState")),
		SessionFlags:                 int(r.readIntVariableOrZeroLocked("SessionFlags")),
		SessionTimeSeconds:           clampNonNegative(r.readFloatVariableOrZeroLocked("SessionTime")),
		SessionTimeRemainingSeconds:  clampNonNegative(r.readFloatVariableOrZeroLocked("SessionTimeRemain")),
		SessionLapsRemaining:         clampNonNegative(r.readFloatVariableOrZeroLocked("SessionLapsRemain")),
		Position:                     clampPositiveOrZero(int(r.readIntVariableOrZeroLocked("PlayerCarPosition"))),
		ClassPosition:                clampPositiveOrZero(int(r.readIntVariableOrZeroLocked("PlayerCarClassPosition"))),
		FuelLevelLiters:              clampNonNegative(r.readFloatVariableOrZeroLocked("FuelLevel")),
		FuelLevelPct:                 clamp01(r.readFloatVariableOrZeroLocked("FuelLevelPct")),
		FuelUsePerHour:               clampNonNegative(r.readFloatVariableOrZeroLocked("FuelUsePerHour")),
		TrackTempC:                   r.readFloatVariableOrZeroLocked("TrackTemp"),
		TrackTempCrewC:               r.readFloatVariableOrZeroLocked("TrackTempCrew"),
		TrackWetness:                 clampPositiveOrZero(int(r.readIntVariableOrZeroLocked("TrackWetness"))),
		AirTempC:                     r.readFloatVariableOrZeroLocked("AirTemp"),
		RelativeHumidityPct:          clampPct(r.readFloatVariableOrZeroLocked("RelativeHumidity")),
		PrecipitationPct:             clampPct(r.readFloatVariableOrZeroLocked("Precipitation")),
		WaterTempC:                   clampNonNegative(r.readFloatVariableOrZeroLocked("WaterTemp")),
		OilTempC:                     clampNonNegative(r.readFloatVariableOrZeroLocked("OilTemp")),
		Voltage:                      clampNonNegative(r.readFloatVariableOrZeroLocked("Voltage")),
		WindDirectionRad:             r.readFloatVariableOrZeroLocked("WindDir"),
		WindVelocityMps:              clampNonNegative(r.readFloatVariableOrZeroLocked("WindVel")),
		WeatherDeclaredWet:           r.readBoolVariableOrFalseLocked("WeatherDeclaredWet"),
		LatAccel:                     r.readFloatVariableOrZeroLocked("LatAccel"),
		LongAccel:                    r.readFloatVariableOrZeroLocked("LongAccel"),
		VertAccel:                    r.readFloatVariableOrZeroLocked("VertAccel"),
		VelocityX:                    r.readFloatVariableOrZeroLocked("VelocityX"),
		VelocityY:                    r.readFloatVariableOrZeroLocked("VelocityY"),
		VelocityZ:                    r.readFloatVariableOrZeroLocked("VelocityZ"),
		Yaw:                          r.readFloatVariableOrZeroLocked("Yaw"),
		YawNorth:                     r.readFloatVariableOrZeroLocked("YawNorth"),
		YawRate:                      r.readFloatVariableOrZeroLocked("YawRate"),
		Pitch:                        r.readFloatVariableOrZeroLocked("Pitch"),
		Roll:                         r.readFloatVariableOrZeroLocked("Roll"),
		OnPitRoad:                    r.readBoolVariableOrFalseLocked("OnPitRoad"),
		IsOnTrack:                    r.readFirstBoolVariableLocked("IsOnTrackCar", "IsOnTrack"),
		IsInGarage:                   r.readBoolVariableOrFalseLocked("IsInGarage"),
		TrackSurface:                 clampPositiveOrZero(int(r.readIntVariableOrZeroLocked("PlayerTrackSurface"))),
		Incidents:                    clampPositiveOrZero(int(r.readIntVariableOrZeroLocked("PlayerCarMyIncidentCount"))),
		PlayerCarPowerAdjust:         int(r.readIntVariableOrZeroLocked("PlayerCarPowerAdjust")),
		PlayerTireCompound:           int(r.readIntVariableOrZeroLocked("PlayerTireCompound")),
		PitServiceTireCompound:       int(r.readIntVariableOrZeroLocked("PitSvTireCompound")),
		PitServiceLFPressure:         clampNonNegative(r.readFloatVariableOrZeroLocked("PitSvLFP")),
		PitServiceLRPressure:         clampNonNegative(r.readFloatVariableOrZeroLocked("PitSvLRP")),
		PitServiceRFPressure:         clampNonNegative(r.readFloatVariableOrZeroLocked("PitSvRFP")),
		PitServiceRRPressure:         clampNonNegative(r.readFloatVariableOrZeroLocked("PitSvRRP")),
		TireSetsAvailable:            clampPositiveOrZero(int(r.readIntVariableOrZeroLocked("TireSetsAvailable"))),
		TireSetsUsed:                 clampPositiveOrZero(int(r.readIntVariableOrZeroLocked("TireSetsUsed"))),
		LeftTireSetsAvailable:        clampPositiveOrZero(int(r.readIntVariableOrZeroLocked("LeftTireSetsAvailable"))),
		LeftTireSetsUsed:             clampPositiveOrZero(int(r.readIntVariableOrZeroLocked("LeftTireSetsUsed"))),
		RightTireSetsAvailable:       clampPositiveOrZero(int(r.readIntVariableOrZeroLocked("RightTireSetsAvailable"))),
		RightTireSetsUsed:            clampPositiveOrZero(int(r.readIntVariableOrZeroLocked("RightTireSetsUsed"))),
		RearTireSetsAvailable:        clampPositiveOrZero(int(r.readIntVariableOrZeroLocked("RearTireSetsAvailable"))),
		RearTireSetsUsed:             clampPositiveOrZero(int(r.readIntVariableOrZeroLocked("RearTireSetsUsed"))),
		PushToPassActive:             r.readBoolVariableOrFalseLocked("PushToPass"),
		PushToPassCount:              clampPositiveOrZero(int(r.readIntVariableOrZeroLocked("P2P_Count"))),
		PushToPassStatus:             clampPositiveOrZero(int(r.readIntVariableOrZeroLocked("P2P_Status"))),
		EngineWarnings:               int(r.readIntVariableOrZeroLocked("EngineWarnings")),
		Source:                       stateLive,
		SampleTime:                   sampleTime,
	}

	latitude, latOK := r.readFloatVariableLocked("Lat")
	longitude, lonOK := r.readFloatVariableLocked("Lon")
	altitude, altOK := r.readFloatVariableLocked("Alt")
	if latOK && lonOK && (latitude != 0 || longitude != 0) {
		snapshot.HasLocation = true
		snapshot.LatitudeDeg = latitude
		snapshot.LongitudeDeg = longitude
		if altOK {
			snapshot.AltitudeMeters = altitude
		}
	}

	r.trackTrace.Apply(&snapshot)

	r.logStateTransitionLocked(stateLive, "reading telemetry from shared memory")

	return snapshot
}

func (r *Reader) fallbackSnapshotLocked(source string, reason string) TelemetrySnapshot {
	r.logStateTransitionLocked(source, reason)
	r.trackTrace.Reset()
	return TelemetrySnapshot{
		Source:     source,
		SampleTime: time.Now().UTC(),
	}
}

func (r *Reader) tryConnectSharedMemory() error {
	if err := r.openMappingByName(irsdkMapName); err == nil {
		return nil
	}
	if err := r.openMappingByName(irsdkMapNameGlobal); err == nil {
		return nil
	}
	return errors.New("could not open iRacing shared memory")
}

func (r *Reader) openMappingByName(name string) error {
	nameUTF16, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return err
	}

	kernel := syscall.NewLazyDLL("kernel32.dll")
	openFileMapping := kernel.NewProc("OpenFileMappingW")
	mapViewOfFile := kernel.NewProc("MapViewOfFile")
	unmapViewOfFile := kernel.NewProc("UnmapViewOfFile")
	closeHandle := kernel.NewProc("CloseHandle")
	virtualQuery := kernel.NewProc("VirtualQuery")

	const fileMapRead = 0x0004

	handle, _, callErr := openFileMapping.Call(fileMapRead, 0, uintptr(unsafe.Pointer(nameUTF16)))
	if handle == 0 {
		return callErr
	}

	addr, _, mapErr := mapViewOfFile.Call(handle, fileMapRead, 0, 0, 0)
	if addr == 0 {
		closeHandle.Call(handle)
		return mapErr
	}

	type memoryBasicInformation struct {
		BaseAddress       uintptr
		AllocationBase    uintptr
		AllocationProtect uint32
		PartitionID       uint16
		RegionSize        uintptr
		State             uint32
		Protect           uint32
		Type              uint32
	}

	var mbi memoryBasicInformation
	ret, _, _ := virtualQuery.Call(addr, uintptr(unsafe.Pointer(&mbi)), unsafe.Sizeof(mbi))
	mapSize := uint32(0)
	if ret != 0 {
		mapSize = uint32(mbi.RegionSize)
	}
	if mapSize == 0 || mapSize > 32*1024*1024 {
		unmapViewOfFile.Call(addr)
		closeHandle.Call(handle)
		return fmt.Errorf("invalid map size: %d", mapSize)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.disconnectLocked()
	r.fileHandle = syscall.Handle(handle)
	r.mappingAddr = addr
	r.mapSize = mapSize
	r.mappingBytes = unsafe.Slice((*byte)(unsafe.Pointer(addr)), int(mapSize))
	r.variables = make(map[string]variableRef)
	r.logStateTransitionLocked(stateLive, "connected to iRacing shared memory")

	return nil
}

func (r *Reader) disconnectLocked() {
	if r.mappingAddr != 0 {
		kernel := syscall.NewLazyDLL("kernel32.dll")
		unmapViewOfFile := kernel.NewProc("UnmapViewOfFile")
		unmapViewOfFile.Call(r.mappingAddr)
		r.mappingAddr = 0
	}
	if r.fileHandle != 0 {
		kernel := syscall.NewLazyDLL("kernel32.dll")
		closeHandle := kernel.NewProc("CloseHandle")
		closeHandle.Call(uintptr(r.fileHandle))
		r.fileHandle = 0
	}

	r.mappingBytes = nil
	r.mapSize = 0
	r.variables = make(map[string]variableRef)
	r.trackTrace.Reset()
}

func (r *Reader) resolveVariablesLocked() error {
	if r.hasAllRequiredVariablesLocked() {
		return nil
	}
	if len(r.mappingBytes) < headerSize {
		return errors.New("header too small")
	}

	varCount := int32(binary.LittleEndian.Uint32(r.mappingBytes[24:28]))
	varHeaderOffset := int32(binary.LittleEndian.Uint32(r.mappingBytes[28:32]))
	if varCount <= 0 || varCount > 4096 {
		return fmt.Errorf("invalid var count: %d", varCount)
	}
	if varHeaderOffset <= 0 {
		return fmt.Errorf("invalid var header offset: %d", varHeaderOffset)
	}

	base := int(varHeaderOffset)
	for index := 0; index < int(varCount); index++ {
		start := base + index*varHdrSize
		end := start + varHdrSize
		if end > len(r.mappingBytes) {
			break
		}

		entry := r.mappingBytes[start:end]
		valueType := int32(binary.LittleEndian.Uint32(entry[0:4]))
		offset := int32(binary.LittleEndian.Uint32(entry[4:8]))
		count := int32(binary.LittleEndian.Uint32(entry[8:12]))
		name := cString(entry[16:48])

		if !isKnownTelemetryVariable(name) {
			continue
		}

		r.variables[name] = variableRef{
			Offset: offset,
			Type:   valueType,
			Count:  count,
		}
	}

	for _, definition := range telemetryDefinitions {
		if !definition.Required {
			continue
		}
		if _, found := r.variables[definition.Name]; !found {
			return fmt.Errorf("required variable not found: %s", definition.Name)
		}
	}

	return nil
}

func (r *Reader) readBufferBaseLocked() (int, bool) {
	if len(r.mappingBytes) < headerSize {
		return 0, false
	}
	bufCount := int(binary.LittleEndian.Uint32(r.mappingBytes[32:36]))
	if bufCount <= 0 || bufCount > 4 {
		return 0, false
	}

	latestTick := int32(-1)
	latestBuf := 0
	for index := 0; index < bufCount; index++ {
		offset := 48 + index*16
		tickCount := int32(binary.LittleEndian.Uint32(r.mappingBytes[offset : offset+4]))
		if tickCount > latestTick {
			latestTick = tickCount
			latestBuf = index
		}
	}

	bufBase := 48 + latestBuf*16
	bufOffset := int(binary.LittleEndian.Uint32(r.mappingBytes[bufBase+4 : bufBase+8]))
	return bufOffset, true
}

func (r *Reader) readFloatVariableLocked(name string) (float64, bool) {
	ref, ok := r.variables[name]
	if !ok || ref.Count <= 0 {
		return 0, false
	}

	bufOffset, ok := r.readBufferBaseLocked()
	if !ok {
		return 0, false
	}

	index := bufOffset + int(ref.Offset)
	switch ref.Type {
	case irsdkTypeFloat64:
		if index+8 > len(r.mappingBytes) {
			return 0, false
		}
		return math.Float64frombits(binary.LittleEndian.Uint64(r.mappingBytes[index : index+8])), true
	case irsdkTypeFloat32:
		if index+4 > len(r.mappingBytes) {
			return 0, false
		}
		bits := binary.LittleEndian.Uint32(r.mappingBytes[index : index+4])
		return float64(math.Float32frombits(bits)), true
	default:
		return 0, false
	}
}

func (r *Reader) readIntVariableLocked(name string) (int32, bool) {
	ref, ok := r.variables[name]
	if !ok || ref.Count <= 0 {
		return 0, false
	}

	bufOffset, ok := r.readBufferBaseLocked()
	if !ok {
		return 0, false
	}

	index := bufOffset + int(ref.Offset)
	if index+4 > len(r.mappingBytes) {
		return 0, false
	}
	if ref.Type != irsdkTypeInt32 && ref.Type != irsdkTypeBitField {
		return 0, false
	}

	return int32(binary.LittleEndian.Uint32(r.mappingBytes[index : index+4])), true
}

func (r *Reader) readBoolVariableLocked(name string) (bool, bool) {
	ref, ok := r.variables[name]
	if !ok || ref.Count <= 0 {
		return false, false
	}

	bufOffset, ok := r.readBufferBaseLocked()
	if !ok {
		return false, false
	}

	index := bufOffset + int(ref.Offset)
	if index >= len(r.mappingBytes) {
		return false, false
	}

	switch ref.Type {
	case irsdkTypeBool:
		return r.mappingBytes[index] != 0, true
	case irsdkTypeInt32, irsdkTypeBitField:
		if index+4 > len(r.mappingBytes) {
			return false, false
		}
		return binary.LittleEndian.Uint32(r.mappingBytes[index:index+4]) != 0, true
	default:
		return false, false
	}
}

func (r *Reader) readFloatVariableOrZeroLocked(name string) float64 {
	value, _ := r.readFloatVariableLocked(name)
	return value
}

func (r *Reader) readIntVariableOrZeroLocked(name string) int32 {
	value, _ := r.readIntVariableLocked(name)
	return value
}

func (r *Reader) readBoolVariableOrFalseLocked(name string) bool {
	value, _ := r.readBoolVariableLocked(name)
	return value
}

func (r *Reader) readFirstBoolVariableLocked(names ...string) bool {
	for _, name := range names {
		value, ok := r.readBoolVariableLocked(name)
		if ok {
			return value
		}
	}
	return false
}

func cString(raw []byte) string {
	for index := 0; index < len(raw); index++ {
		if raw[index] == 0 {
			return string(raw[:index])
		}
	}
	return string(raw)
}

func (r *Reader) logStateTransitionLocked(state string, reason string) {
	if r.lastState == state && r.lastStateReason == reason {
		return
	}

	r.lastState = state
	r.lastStateReason = reason

	if r.logger != nil {
		r.logger.Printf("state=%s reason=%s", state, reason)
	}
}

func isKnownTelemetryVariable(name string) bool {
	for _, definition := range telemetryDefinitions {
		if definition.Name == name {
			return true
		}
	}
	return false
}

func (r *Reader) hasAllRequiredVariablesLocked() bool {
	for _, definition := range telemetryDefinitions {
		if !definition.Required {
			continue
		}
		if _, ok := r.variables[definition.Name]; !ok {
			return false
		}
	}
	return true
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func clampPct(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func clampGear(value int) int {
	if value < -1 {
		return -1
	}
	if value > 8 {
		return 8
	}
	return value
}

func clampNonNegative(value float64) float64 {
	if value < 0 {
		return 0
	}
	return value
}

func clampPositiveOrZero(value int) int {
	if value < 0 {
		return 0
	}
	return value
}
