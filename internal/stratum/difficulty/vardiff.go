package difficulty

import (
	"math"
	"sync"
	"time"
)

// =============================================================================
// HARDWARE-AWARE VARIABLE DIFFICULTY SYSTEM
// Optimized for hybrid V1/V2 pool with X30/X100 ASIC support
// =============================================================================

// Hardware classes with recommended difficulty tiers
const (
	// Hardware class identifiers
	HardwareUnknown      = "unknown"
	HardwareCPU          = "cpu"
	HardwareGPU          = "gpu"
	HardwareFPGA         = "fpga"
	HardwareASIC         = "asic"
	HardwareOfficialASIC = "official_asic" // BlockDAG X30/X100
)

// Default difficulty values per hardware class
const (
	DefaultDiffCPU          uint64 = 32    // ~100 KH/s
	DefaultDiffGPU          uint64 = 4096  // ~10 MH/s
	DefaultDiffFPGA         uint64 = 16384 // ~50 MH/s
	DefaultDiffASIC         uint64 = 32768 // ~80 MH/s (X30 class)
	DefaultDiffOfficialASIC uint64 = 65536 // ~240 MH/s (X100 class)
	DefaultDiffUnknown      uint64 = 256   // Conservative default

	// Difficulty bounds
	MinDifficulty uint64 = 1
	MaxDifficulty uint64 = 1 << 48 // ~281 trillion

	// Target share submission rate
	TargetShareTime = 10 * time.Second
	MinShareTime    = 2 * time.Second
	MaxShareTime    = 60 * time.Second

	// Adjustment parameters
	VardiffRetargetTime = 90 * time.Second // Time window for retargeting
	VardiffMinShares    = 3                // Minimum shares before adjustment
	AdjustmentFactor    = 2.0              // Max adjustment per retarget
)

// =============================================================================
// Hardware Classifier
// =============================================================================

// HardwareClass represents a mining hardware classification
type HardwareClass struct {
	Name             string
	BaseDifficulty   uint64
	MinDifficulty    uint64
	MaxDifficulty    uint64
	ExpectedHashrate float64 // H/s
}

// Predefined hardware classes
var (
	ClassCPU = HardwareClass{
		Name:             HardwareCPU,
		BaseDifficulty:   DefaultDiffCPU,
		MinDifficulty:    1,
		MaxDifficulty:    1024,
		ExpectedHashrate: 100000, // 100 KH/s
	}

	ClassGPU = HardwareClass{
		Name:             HardwareGPU,
		BaseDifficulty:   DefaultDiffGPU,
		MinDifficulty:    256,
		MaxDifficulty:    65536,
		ExpectedHashrate: 10000000, // 10 MH/s
	}

	ClassFPGA = HardwareClass{
		Name:             HardwareFPGA,
		BaseDifficulty:   DefaultDiffFPGA,
		MinDifficulty:    4096,
		MaxDifficulty:    262144,
		ExpectedHashrate: 50000000, // 50 MH/s
	}

	ClassASIC = HardwareClass{
		Name:             HardwareASIC,
		BaseDifficulty:   DefaultDiffASIC,
		MinDifficulty:    8192,
		MaxDifficulty:    524288,
		ExpectedHashrate: 80000000, // 80 MH/s (X30)
	}

	ClassOfficialASIC = HardwareClass{
		Name:             HardwareOfficialASIC,
		BaseDifficulty:   DefaultDiffOfficialASIC,
		MinDifficulty:    16384,
		MaxDifficulty:    1048576,
		ExpectedHashrate: 240000000, // 240 MH/s (X100)
	}

	ClassUnknown = HardwareClass{
		Name:             HardwareUnknown,
		BaseDifficulty:   DefaultDiffUnknown,
		MinDifficulty:    1,
		MaxDifficulty:    65536,
		ExpectedHashrate: 1000000, // 1 MH/s default
	}
)

// HardwareClassifier classifies miners based on reported/observed data
type HardwareClassifier struct {
	mu sync.RWMutex
}

// NewHardwareClassifier creates a new hardware classifier
func NewHardwareClassifier() *HardwareClassifier {
	return &HardwareClassifier{}
}

// ClassifyByHashrate classifies hardware based on observed hashrate
func (hc *HardwareClassifier) ClassifyByHashrate(hashrate float64) HardwareClass {
	switch {
	case hashrate >= 200000000: // >= 200 MH/s
		return ClassOfficialASIC
	case hashrate >= 60000000: // >= 60 MH/s
		return ClassASIC
	case hashrate >= 30000000: // >= 30 MH/s
		return ClassFPGA
	case hashrate >= 1000000: // >= 1 MH/s
		return ClassGPU
	default:
		return ClassCPU
	}
}

// ClassifyByUserAgent classifies hardware based on miner user-agent string
func (hc *HardwareClassifier) ClassifyByUserAgent(userAgent string) HardwareClass {
	// Check for known BlockDAG miner identifiers
	if containsAny(userAgent, []string{"X100", "BlockDAG-X100", "BDAG-X100"}) {
		return ClassOfficialASIC
	}
	if containsAny(userAgent, []string{"X30", "BlockDAG-X30", "BDAG-X30"}) {
		return ClassASIC
	}

	// Generic ASIC detection
	if containsAny(userAgent, []string{"ASIC", "Antminer", "Whatsminer", "Avalon"}) {
		return ClassASIC
	}

	// FPGA detection
	if containsAny(userAgent, []string{"FPGA", "Xilinx", "Altera"}) {
		return ClassFPGA
	}

	// GPU detection
	if containsAny(userAgent, []string{"GPU", "CUDA", "OpenCL", "AMD", "NVIDIA", "Radeon", "GeForce"}) {
		return ClassGPU
	}

	// CPU detection
	if containsAny(userAgent, []string{"CPU", "cpuminer", "xmrig"}) {
		return ClassCPU
	}

	return ClassUnknown
}

// ClassifyByCombined uses both hashrate and user-agent for best classification
func (hc *HardwareClassifier) ClassifyByCombined(userAgent string, hashrate float64) HardwareClass {
	// If hashrate is available, it's more reliable
	if hashrate > 0 {
		return hc.ClassifyByHashrate(hashrate)
	}
	return hc.ClassifyByUserAgent(userAgent)
}

// =============================================================================
// Miner Difficulty State
// =============================================================================

// MinerState tracks difficulty state for a single miner
type MinerState struct {
	MinerID         string
	HardwareClass   HardwareClass
	CurrentDiff     uint64
	ShareCount      uint64
	ValidShares     uint64
	InvalidShares   uint64
	StaleShares     uint64
	LastShareTime   time.Time
	FirstShareTime  time.Time
	ShareTimes      []time.Duration // Recent share intervals
	AverageHashrate float64
	LastAdjustment  time.Time
	mu              sync.RWMutex
}

// NewMinerState creates a new miner state with initial difficulty
func NewMinerState(minerID string, hardwareClass HardwareClass) *MinerState {
	return &MinerState{
		MinerID:        minerID,
		HardwareClass:  hardwareClass,
		CurrentDiff:    hardwareClass.BaseDifficulty,
		ShareTimes:     make([]time.Duration, 0, 100),
		LastAdjustment: time.Now(),
	}
}

// RecordShare records a share submission and updates statistics
func (ms *MinerState) RecordShare(valid bool, stale bool) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	now := time.Now()
	ms.ShareCount++

	if valid && !stale {
		ms.ValidShares++
	} else if stale {
		ms.StaleShares++
	} else {
		ms.InvalidShares++
	}

	// Record share interval
	if !ms.LastShareTime.IsZero() {
		interval := now.Sub(ms.LastShareTime)
		ms.ShareTimes = append(ms.ShareTimes, interval)

		// Keep only last 100 intervals
		if len(ms.ShareTimes) > 100 {
			ms.ShareTimes = ms.ShareTimes[1:]
		}
	} else {
		ms.FirstShareTime = now
	}

	ms.LastShareTime = now

	// Update estimated hashrate
	ms.updateHashrate()
}

// updateHashrate calculates estimated hashrate from share data
func (ms *MinerState) updateHashrate() {
	if len(ms.ShareTimes) < 2 {
		return
	}

	// Calculate average share time
	var totalTime time.Duration
	for _, t := range ms.ShareTimes {
		totalTime += t
	}
	avgTime := totalTime / time.Duration(len(ms.ShareTimes))

	if avgTime > 0 {
		// Hashrate = Difficulty * 2^32 / AverageShareTime (in seconds)
		ms.AverageHashrate = float64(ms.CurrentDiff) * 4294967296.0 / avgTime.Seconds()
	}
}

// GetAverageShareTime returns the average time between shares
func (ms *MinerState) GetAverageShareTime() time.Duration {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if len(ms.ShareTimes) == 0 {
		return 0
	}

	var total time.Duration
	for _, t := range ms.ShareTimes {
		total += t
	}
	return total / time.Duration(len(ms.ShareTimes))
}

// GetCurrentDifficulty returns the current difficulty
func (ms *MinerState) GetCurrentDifficulty() uint64 {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return ms.CurrentDiff
}

// SetDifficulty sets a new difficulty value
func (ms *MinerState) SetDifficulty(diff uint64) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Clamp to hardware class bounds
	if diff < ms.HardwareClass.MinDifficulty {
		diff = ms.HardwareClass.MinDifficulty
	}
	if diff > ms.HardwareClass.MaxDifficulty {
		diff = ms.HardwareClass.MaxDifficulty
	}

	ms.CurrentDiff = diff
	ms.LastAdjustment = time.Now()
}

// GetStats returns miner statistics
func (ms *MinerState) GetStats() MinerStats {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	return MinerStats{
		MinerID:         ms.MinerID,
		HardwareClass:   ms.HardwareClass.Name,
		CurrentDiff:     ms.CurrentDiff,
		ShareCount:      ms.ShareCount,
		ValidShares:     ms.ValidShares,
		InvalidShares:   ms.InvalidShares,
		StaleShares:     ms.StaleShares,
		AverageHashrate: ms.AverageHashrate,
		LastShareTime:   ms.LastShareTime,
	}
}

// MinerStats contains read-only miner statistics
type MinerStats struct {
	MinerID         string
	HardwareClass   string
	CurrentDiff     uint64
	ShareCount      uint64
	ValidShares     uint64
	InvalidShares   uint64
	StaleShares     uint64
	AverageHashrate float64
	LastShareTime   time.Time
}

// =============================================================================
// Variable Difficulty Manager
// =============================================================================

// VardiffManager manages variable difficulty for all miners
type VardiffManager struct {
	miners          map[string]*MinerState
	classifier      *HardwareClassifier
	targetShareTime time.Duration
	retargetTime    time.Duration
	minShares       int
	mu              sync.RWMutex
}

// NewVardiffManager creates a new variable difficulty manager
func NewVardiffManager() *VardiffManager {
	return &VardiffManager{
		miners:          make(map[string]*MinerState),
		classifier:      NewHardwareClassifier(),
		targetShareTime: TargetShareTime,
		retargetTime:    VardiffRetargetTime,
		minShares:       VardiffMinShares,
	}
}

// NewVardiffManagerWithParams creates a manager with custom parameters
func NewVardiffManagerWithParams(targetShareTime, retargetTime time.Duration, minShares int) *VardiffManager {
	return &VardiffManager{
		miners:          make(map[string]*MinerState),
		classifier:      NewHardwareClassifier(),
		targetShareTime: targetShareTime,
		retargetTime:    retargetTime,
		minShares:       minShares,
	}
}

// RegisterMiner registers a new miner with initial classification
func (vm *VardiffManager) RegisterMiner(minerID, userAgent string, hashrate float64) *MinerState {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	// Classify hardware
	hwClass := vm.classifier.ClassifyByCombined(userAgent, hashrate)

	// Create state
	state := NewMinerState(minerID, hwClass)
	vm.miners[minerID] = state

	return state
}

// GetMiner returns the miner state, creating if needed
func (vm *VardiffManager) GetMiner(minerID string) (*MinerState, bool) {
	vm.mu.RLock()
	state, exists := vm.miners[minerID]
	vm.mu.RUnlock()
	return state, exists
}

// GetOrCreateMiner returns existing state or creates new one
func (vm *VardiffManager) GetOrCreateMiner(minerID string) *MinerState {
	state, exists := vm.GetMiner(minerID)
	if exists {
		return state
	}

	// Create with unknown class (will be reclassified after first shares)
	return vm.RegisterMiner(minerID, "", 0)
}

// RemoveMiner removes a miner from tracking
func (vm *VardiffManager) RemoveMiner(minerID string) {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	delete(vm.miners, minerID)
}

// RecordShare records a share and returns updated difficulty if changed
func (vm *VardiffManager) RecordShare(minerID string, valid, stale bool) (newDiff uint64, changed bool) {
	state, exists := vm.GetMiner(minerID)
	if !exists {
		state = vm.GetOrCreateMiner(minerID)
	}

	// Record the share
	state.RecordShare(valid, stale)

	// Check if we should retarget
	return vm.checkAndAdjust(state)
}

// checkAndAdjust checks if difficulty adjustment is needed
func (vm *VardiffManager) checkAndAdjust(state *MinerState) (uint64, bool) {
	state.mu.Lock()
	defer state.mu.Unlock()

	// Need minimum shares and time since last adjustment
	if len(state.ShareTimes) < vm.minShares {
		return state.CurrentDiff, false
	}

	timeSinceAdjust := time.Since(state.LastAdjustment)
	if timeSinceAdjust < vm.retargetTime {
		return state.CurrentDiff, false
	}

	// Calculate average share time (inline to avoid deadlock)
	var avgShareTime time.Duration
	if len(state.ShareTimes) > 0 {
		var total time.Duration
		for _, t := range state.ShareTimes {
			total += t
		}
		avgShareTime = total / time.Duration(len(state.ShareTimes))
	}
	if avgShareTime == 0 {
		return state.CurrentDiff, false
	}

	// Calculate adjustment ratio
	ratio := float64(vm.targetShareTime) / float64(avgShareTime)

	// Clamp adjustment factor
	if ratio > AdjustmentFactor {
		ratio = AdjustmentFactor
	} else if ratio < 1.0/AdjustmentFactor {
		ratio = 1.0 / AdjustmentFactor
	}

	// Skip tiny adjustments
	if ratio > 0.9 && ratio < 1.1 {
		state.LastAdjustment = time.Now()
		return state.CurrentDiff, false
	}

	// Calculate new difficulty
	newDiff := uint64(float64(state.CurrentDiff) * ratio)

	// Clamp to bounds
	if newDiff < state.HardwareClass.MinDifficulty {
		newDiff = state.HardwareClass.MinDifficulty
	}
	if newDiff > state.HardwareClass.MaxDifficulty {
		newDiff = state.HardwareClass.MaxDifficulty
	}
	if newDiff < MinDifficulty {
		newDiff = MinDifficulty
	}
	if newDiff > MaxDifficulty {
		newDiff = MaxDifficulty
	}

	// Update if changed
	if newDiff != state.CurrentDiff {
		state.CurrentDiff = newDiff
		state.LastAdjustment = time.Now()
		state.ShareTimes = state.ShareTimes[:0] // Reset for new measurement
		return newDiff, true
	}

	return state.CurrentDiff, false
}

// GetDifficulty returns current difficulty for a miner
func (vm *VardiffManager) GetDifficulty(minerID string) uint64 {
	state, exists := vm.GetMiner(minerID)
	if !exists {
		return DefaultDiffUnknown
	}
	return state.GetCurrentDifficulty()
}

// SetDifficulty manually sets difficulty for a miner
func (vm *VardiffManager) SetDifficulty(minerID string, diff uint64) error {
	state, exists := vm.GetMiner(minerID)
	if !exists {
		return ErrMinerNotFound
	}
	state.SetDifficulty(diff)
	return nil
}

// ReclassifyMiner updates hardware classification based on observed performance
func (vm *VardiffManager) ReclassifyMiner(minerID string) {
	state, exists := vm.GetMiner(minerID)
	if !exists {
		return
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	if state.AverageHashrate > 0 {
		newClass := vm.classifier.ClassifyByHashrate(state.AverageHashrate)
		if newClass.Name != state.HardwareClass.Name {
			state.HardwareClass = newClass
			// Adjust difficulty to new class base if needed
			if state.CurrentDiff < newClass.MinDifficulty {
				state.CurrentDiff = newClass.MinDifficulty
			}
		}
	}
}

// GetAllStats returns statistics for all miners
func (vm *VardiffManager) GetAllStats() []MinerStats {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	stats := make([]MinerStats, 0, len(vm.miners))
	for _, state := range vm.miners {
		stats = append(stats, state.GetStats())
	}
	return stats
}

// GetMinerCount returns the number of tracked miners
func (vm *VardiffManager) GetMinerCount() int {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	return len(vm.miners)
}

// GetPoolHashrate returns estimated total pool hashrate
func (vm *VardiffManager) GetPoolHashrate() float64 {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	var total float64
	for _, state := range vm.miners {
		state.mu.RLock()
		total += state.AverageHashrate
		state.mu.RUnlock()
	}
	return total
}

// =============================================================================
// Errors
// =============================================================================

type vardiffError string

func (e vardiffError) Error() string {
	return string(e)
}

const (
	ErrMinerNotFound vardiffError = "miner not found"
)

// =============================================================================
// Helper Functions
// =============================================================================

// containsAny checks if s contains any of the substrings
func containsAny(s string, substrings []string) bool {
	for _, sub := range substrings {
		if containsIgnoreCase(s, sub) {
			return true
		}
	}
	return false
}

// containsIgnoreCase checks if s contains sub (case-insensitive)
func containsIgnoreCase(s, sub string) bool {
	sLower := toLower(s)
	subLower := toLower(sub)
	return contains(sLower, subLower)
}

// toLower converts string to lowercase (simple ASCII)
func toLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}

// contains checks if s contains sub
func contains(s, sub string) bool {
	if len(sub) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// CalculateExpectedDifficulty calculates recommended difficulty from hashrate
func CalculateExpectedDifficulty(hashrate float64, targetTime time.Duration) uint64 {
	if hashrate <= 0 || targetTime <= 0 {
		return DefaultDiffUnknown
	}

	// Difficulty = Hashrate * TargetTime / 2^32
	diff := hashrate * targetTime.Seconds() / 4294967296.0

	if diff < float64(MinDifficulty) {
		return MinDifficulty
	}
	if diff > float64(MaxDifficulty) {
		return MaxDifficulty
	}

	return uint64(math.Round(diff))
}

// CalculateHashrateFromDifficulty estimates hashrate from difficulty and share time
func CalculateHashrateFromDifficulty(difficulty uint64, shareTime time.Duration) float64 {
	if shareTime <= 0 {
		return 0
	}

	// Hashrate = Difficulty * 2^32 / ShareTime
	return float64(difficulty) * 4294967296.0 / shareTime.Seconds()
}
