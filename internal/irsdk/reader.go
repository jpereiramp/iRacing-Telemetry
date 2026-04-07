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

const (
	irsdkMapName       = "Local\\IRSDKMemMapFileName"
	irsdkMapNameGlobal = "IRSDKMemMapFileName"

	headerSize = 112
	varHdrSize = 144

	stateDisconnected = "disconnected"
	stateLive         = "live"
	stateFallback     = "fallback"

	irsdkTypeFloat32 = 4
	irsdkTypeFloat64 = 5
	irsdkTypeInt32   = 2
)

var requiredTelemetryVariables = []string{
	"Speed",
	"RPM",
	"Gear",
	"Throttle",
	"Brake",
	"Clutch",
}

type TelemetrySnapshot struct {
	SpeedKPH   float64   `json:"speedKph"`
	SpeedMPH   float64   `json:"speedMph"`
	RPM        float64   `json:"rpm"`
	Gear       int       `json:"gear"`
	Throttle   float64   `json:"throttle"`
	Brake      float64   `json:"brake"`
	Clutch     float64   `json:"clutch"`
	Source     string    `json:"source"`
	SampleTime time.Time `json:"sampleTime"`
}

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
}

func NewReader() (*Reader, error) {
	r := &Reader{
		variables: make(map[string]variableRef),
		logger:    log.New(log.Writer(), "[irsdk] ", log.LstdFlags|log.Lmicroseconds),
	}
	// Do not fail startup when iRacing is not running.
	// The reader will keep attempting to connect during ReadSnapshot calls.
	_ = r.tryConnectSharedMemory()
	return r, nil
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
		// Mapping may be stale/invalid if iRacing restarted.
		// Drop mapping and retry connect on next tick.
		r.disconnectLocked()
		return r.fallbackSnapshotLocked(stateFallback, err.Error())
	}

	speedMps, ok := r.readFloatVariableLocked("Speed")
	if !ok {
		// Force reconnect attempts when the active buffer cannot be read.
		r.disconnectLocked()
		return r.fallbackSnapshotLocked(stateFallback, "failed to read speed from latest buffer")
	}

	if speedMps < 0 {
		speedMps = 0
	}

	speedKph := speedMps * 3.6
	speedMph := speedMps * 2.2369362920544

	rpm, _ := r.readFloatVariableLocked("RPM")
	gearValue, _ := r.readIntVariableLocked("Gear")
	throttle, _ := r.readFloatVariableLocked("Throttle")
	brake, _ := r.readFloatVariableLocked("Brake")
	clutch, _ := r.readFloatVariableLocked("Clutch")

	return TelemetrySnapshot{
		SpeedKPH:   speedKph,
		SpeedMPH:   speedMph,
		RPM:        rpm,
		Gear:       clampGear(int(gearValue)),
		Throttle:   clamp01(throttle),
		Brake:      clamp01(brake),
		Clutch:     clamp01(clutch),
		Source:     stateLive,
		SampleTime: time.Now().UTC(),
	}
}

func (r *Reader) fallbackSnapshotLocked(source string, reason string) TelemetrySnapshot {
	r.logStateTransitionLocked(source, reason)
	return TelemetrySnapshot{
		SpeedMPH:   0,
		SpeedKPH:   0,
		RPM:        0,
		Gear:       0,
		Throttle:   0,
		Brake:      0,
		Clutch:     0,
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
	for i := 0; i < int(varCount); i++ {
		start := base + i*varHdrSize
		end := start + varHdrSize
		if end > len(r.mappingBytes) {
			break
		}

		entry := r.mappingBytes[start:end]
		valueType := int32(binary.LittleEndian.Uint32(entry[0:4]))
		offset := int32(binary.LittleEndian.Uint32(entry[4:8]))
		count := int32(binary.LittleEndian.Uint32(entry[8:12]))
		name := cString(entry[16:48])

		if isRequiredVariable(name) {
			r.variables[name] = variableRef{
				Offset: offset,
				Type:   valueType,
				Count:  count,
			}
		}
	}

	for _, name := range requiredTelemetryVariables {
		if _, found := r.variables[name]; !found {
			return fmt.Errorf("required variable not found: %s", name)
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
	for i := 0; i < bufCount; i++ {
		off := 48 + i*16
		tickCount := int32(binary.LittleEndian.Uint32(r.mappingBytes[off : off+4]))
		if tickCount > latestTick {
			latestTick = tickCount
			latestBuf = i
		}
	}

	bufBase := 48 + latestBuf*16
	// irsdk_varBuf layout:
	// int tickCount; int bufOffset; int pad[2];
	// so bufOffset lives at +4.
	bufOffset := int(binary.LittleEndian.Uint32(r.mappingBytes[bufBase+4 : bufBase+8]))
	return bufOffset, true
}

func (r *Reader) readFloatVariableLocked(name string) (float64, bool) {
	ref, ok := r.variables[name]
	if !ok {
		return 0, false
	}
	if ref.Count <= 0 {
		return 0, false
	}
	bufOffset, ok := r.readBufferBaseLocked()
	if !ok {
		return 0, false
	}
	index := bufOffset + int(ref.Offset)

	if ref.Type == irsdkTypeFloat64 {
		if index+8 > len(r.mappingBytes) {
			return 0, false
		}
		return math.Float64frombits(binary.LittleEndian.Uint64(r.mappingBytes[index : index+8])), true
	}

	if ref.Type != irsdkTypeFloat32 {
		return 0, false
	}

	if index+4 > len(r.mappingBytes) {
		return 0, false
	}
	bits := binary.LittleEndian.Uint32(r.mappingBytes[index : index+4])
	return float64(math.Float32frombits(bits)), true
}

func (r *Reader) readIntVariableLocked(name string) (int32, bool) {
	ref, ok := r.variables[name]
	if !ok {
		return 0, false
	}
	if ref.Count <= 0 {
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
	if ref.Type != irsdkTypeInt32 {
		return 0, false
	}
	return int32(binary.LittleEndian.Uint32(r.mappingBytes[index : index+4])), true
}

func cString(raw []byte) string {
	for i := 0; i < len(raw); i++ {
		if raw[i] == 0 {
			return string(raw[:i])
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

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func clampGear(v int) int {
	if v < -1 {
		return -1
	}
	if v > 8 {
		return 8
	}
	return v
}

func isRequiredVariable(name string) bool {
	for _, required := range requiredTelemetryVariables {
		if required == name {
			return true
		}
	}
	return false
}

func (r *Reader) hasAllRequiredVariablesLocked() bool {
	for _, name := range requiredTelemetryVariables {
		if _, ok := r.variables[name]; !ok {
			return false
		}
	}
	return true
}
