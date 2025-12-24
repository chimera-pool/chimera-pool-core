package simulation

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// virtualMinerSimulator implements the VirtualMinerSimulator interface
type virtualMinerSimulator struct {
	config    VirtualMinerConfig
	miners    map[string]*VirtualMiner
	isRunning bool
	stopChan  chan struct{}
	mutex     sync.RWMutex
	stats     *SimulationStats
	startTime time.Time
}

// NewVirtualMinerSimulator creates a new virtual miner simulator
func NewVirtualMinerSimulator(config VirtualMinerConfig) (VirtualMinerSimulator, error) {
	simulator := &virtualMinerSimulator{
		config:   config,
		miners:   make(map[string]*VirtualMiner),
		stopChan: make(chan struct{}),
		stats:    &SimulationStats{},
	}

	// Generate miners based on configuration
	err := simulator.generateMiners()
	if err != nil {
		return nil, fmt.Errorf("failed to generate miners: %w", err)
	}

	return simulator, nil
}

// Start begins the virtual miner simulation
func (vms *virtualMinerSimulator) Start() error {
	vms.mutex.Lock()
	defer vms.mutex.Unlock()

	if vms.isRunning {
		return fmt.Errorf("simulator is already running")
	}

	vms.isRunning = true
	vms.startTime = time.Now()

	// Activate all miners
	for _, miner := range vms.miners {
		miner.IsActive = true
		miner.CurrentState.LastSeen = time.Now()
	}

	// Start behavior simulation goroutines
	go vms.simulateBehaviors()
	go vms.updateStatistics()

	return nil
}

// Stop halts the virtual miner simulation
func (vms *virtualMinerSimulator) Stop() error {
	vms.mutex.Lock()
	defer vms.mutex.Unlock()

	if !vms.isRunning {
		return nil
	}

	vms.isRunning = false
	close(vms.stopChan)

	// Deactivate all miners
	for _, miner := range vms.miners {
		miner.IsActive = false
	}

	return nil
}

// GetMiners returns all virtual miners
func (vms *virtualMinerSimulator) GetMiners() []*VirtualMiner {
	vms.mutex.RLock()
	defer vms.mutex.RUnlock()

	// Return deep COPIES to prevent race conditions
	miners := make([]*VirtualMiner, 0, len(vms.miners))
	for _, miner := range vms.miners {
		miners = append(miners, vms.copyMiner(miner))
	}
	return miners
}

// GetMiner returns a specific miner by ID
func (vms *virtualMinerSimulator) GetMiner(id string) *VirtualMiner {
	vms.mutex.RLock()
	defer vms.mutex.RUnlock()

	miner := vms.miners[id]
	if miner == nil {
		return nil
	}

	// Return a deep COPY to prevent race conditions
	return vms.copyMiner(miner)
}

// copyMiner creates a deep copy of a VirtualMiner
func (vms *virtualMinerSimulator) copyMiner(m *VirtualMiner) *VirtualMiner {
	if m == nil {
		return nil
	}

	copy := &VirtualMiner{
		ID:          m.ID,
		Type:        m.Type,
		HashRate:    m.HashRate,
		IsActive:    m.IsActive,
		IsMalicious: m.IsMalicious,
		Location:    m.Location,
	}

	if m.PerformanceProfile != nil {
		copy.PerformanceProfile = &PerformanceProfile{
			PowerConsumption: m.PerformanceProfile.PowerConsumption,
			EfficiencyRating: m.PerformanceProfile.EfficiencyRating,
			FailureRate:      m.PerformanceProfile.FailureRate,
			Temperature:      m.PerformanceProfile.Temperature,
			FanSpeed:         m.PerformanceProfile.FanSpeed,
		}
	}

	if m.NetworkProfile != nil {
		copy.NetworkProfile = &NetworkProfile{
			Quality:    m.NetworkProfile.Quality,
			Latency:    m.NetworkProfile.Latency,
			PacketLoss: m.NetworkProfile.PacketLoss,
			Jitter:     m.NetworkProfile.Jitter,
			Bandwidth:  m.NetworkProfile.Bandwidth,
		}
	}

	if m.AttackProfile != nil {
		copy.AttackProfile = &AttackProfile{
			IsAttacking:    m.AttackProfile.IsAttacking,
			AttackStarted:  m.AttackProfile.AttackStarted,
			AttackDuration: m.AttackProfile.AttackDuration,
		}
		if m.AttackProfile.AttackTypes != nil {
			copy.AttackProfile.AttackTypes = make([]AttackType, len(m.AttackProfile.AttackTypes))
			for i, at := range m.AttackProfile.AttackTypes {
				copy.AttackProfile.AttackTypes[i] = AttackType{
					Type:        at.Type,
					Probability: at.Probability,
					Intensity:   at.Intensity,
				}
			}
		}
	}

	if m.CurrentState != nil {
		copy.CurrentState = &MinerState{
			IsBursting:      m.CurrentState.IsBursting,
			IsDisconnected:  m.CurrentState.IsDisconnected,
			BurstStarted:    m.CurrentState.BurstStarted,
			BurstDuration:   m.CurrentState.BurstDuration,
			LastSeen:        m.CurrentState.LastSeen,
			SharesSubmitted: m.CurrentState.SharesSubmitted,
			ValidShares:     m.CurrentState.ValidShares,
			InvalidShares:   m.CurrentState.InvalidShares,
		}
	}

	if m.Statistics != nil {
		copy.Statistics = &MinerStatistics{
			TotalShares:      m.Statistics.TotalShares,
			ValidShares:      m.Statistics.ValidShares,
			InvalidShares:    m.Statistics.InvalidShares,
			TotalHashRate:    m.Statistics.TotalHashRate,
			AverageHashRate:  m.Statistics.AverageHashRate,
			UptimePercentage: m.Statistics.UptimePercentage,
			LastShareTime:    m.Statistics.LastShareTime,
			BurstEvents:      m.Statistics.BurstEvents,
			DropEvents:       m.Statistics.DropEvents,
			AttackEvents:     m.Statistics.AttackEvents,
		}
	}

	return copy
}

// AddMiner adds a new miner to the simulation
func (vms *virtualMinerSimulator) AddMiner(config MinerType) (*VirtualMiner, error) {
	vms.mutex.Lock()
	defer vms.mutex.Unlock()

	miner := vms.createMiner(config, false)
	vms.miners[miner.ID] = miner

	if vms.isRunning {
		miner.IsActive = true
		miner.CurrentState.LastSeen = time.Now()
	}

	return miner, nil
}

// RemoveMiner removes a miner from the simulation
func (vms *virtualMinerSimulator) RemoveMiner(id string) error {
	vms.mutex.Lock()
	defer vms.mutex.Unlock()

	if _, exists := vms.miners[id]; !exists {
		return fmt.Errorf("miner %s not found", id)
	}

	delete(vms.miners, id)
	return nil
}

// GetSimulationStats returns overall simulation statistics
func (vms *virtualMinerSimulator) GetSimulationStats() *SimulationStats {
	// Use write lock since calculateStatsLocked modifies vms.stats
	vms.mutex.Lock()
	defer vms.mutex.Unlock()

	// Update stats before returning
	vms.calculateStatsLocked()

	// Return a COPY to prevent race conditions
	return &SimulationStats{
		TotalMiners:       vms.stats.TotalMiners,
		ActiveMiners:      vms.stats.ActiveMiners,
		TotalHashRate:     vms.stats.TotalHashRate,
		AverageHashRate:   vms.stats.AverageHashRate,
		TotalShares:       vms.stats.TotalShares,
		ValidShares:       vms.stats.ValidShares,
		InvalidShares:     vms.stats.InvalidShares,
		TotalBurstEvents:  vms.stats.TotalBurstEvents,
		TotalDropEvents:   vms.stats.TotalDropEvents,
		TotalAttackEvents: vms.stats.TotalAttackEvents,
		UptimePercentage:  vms.stats.UptimePercentage,
		SimulationTime:    vms.stats.SimulationTime,
	}
}

// GetMinerStats returns statistics for a specific miner
func (vms *virtualMinerSimulator) GetMinerStats(id string) *MinerStatistics {
	vms.mutex.RLock()
	defer vms.mutex.RUnlock()

	miner := vms.miners[id]
	if miner == nil {
		return nil
	}

	// Return a COPY to prevent race conditions
	return &MinerStatistics{
		TotalShares:   miner.Statistics.TotalShares,
		ValidShares:   miner.Statistics.ValidShares,
		InvalidShares: miner.Statistics.InvalidShares,
		BurstEvents:   miner.Statistics.BurstEvents,
		DropEvents:    miner.Statistics.DropEvents,
		AttackEvents:  miner.Statistics.AttackEvents,
	}
}

// TriggerBurst triggers burst mining for a specific miner
func (vms *virtualMinerSimulator) TriggerBurst(minerID string, duration time.Duration) error {
	vms.mutex.Lock()
	defer vms.mutex.Unlock()

	miner := vms.miners[minerID]
	if miner == nil {
		return fmt.Errorf("miner %s not found", minerID)
	}

	miner.CurrentState.IsBursting = true
	miner.CurrentState.BurstStarted = time.Now()
	miner.CurrentState.BurstDuration = duration
	miner.Statistics.BurstEvents++

	return nil
}

// TriggerDrop triggers connection drop for a specific miner
func (vms *virtualMinerSimulator) TriggerDrop(minerID string, duration time.Duration) error {
	vms.mutex.Lock()
	defer vms.mutex.Unlock()

	miner := vms.miners[minerID]
	if miner == nil {
		return fmt.Errorf("miner %s not found", minerID)
	}

	miner.CurrentState.IsDisconnected = true
	miner.IsActive = false
	miner.Statistics.DropEvents++

	// Schedule reconnection
	go func() {
		time.Sleep(duration)
		vms.mutex.Lock()
		defer vms.mutex.Unlock()

		if miner := vms.miners[minerID]; miner != nil {
			miner.CurrentState.IsDisconnected = false
			if vms.isRunning {
				miner.IsActive = true
				miner.CurrentState.LastSeen = time.Now()
			}
		}
	}()

	return nil
}

// TriggerAttack triggers malicious behavior for a specific miner
func (vms *virtualMinerSimulator) TriggerAttack(minerID string, attackType string, duration time.Duration) error {
	vms.mutex.Lock()
	defer vms.mutex.Unlock()

	miner := vms.miners[minerID]
	if miner == nil {
		return fmt.Errorf("miner %s not found", minerID)
	}

	if !miner.IsMalicious {
		return fmt.Errorf("miner %s is not configured as malicious", minerID)
	}

	miner.AttackProfile.IsAttacking = true
	miner.AttackProfile.AttackStarted = time.Now()
	miner.AttackProfile.AttackDuration = duration
	miner.Statistics.AttackEvents++

	// Schedule attack end
	go func() {
		time.Sleep(duration)
		vms.mutex.Lock()
		defer vms.mutex.Unlock()

		if miner := vms.miners[minerID]; miner != nil && miner.AttackProfile != nil {
			miner.AttackProfile.IsAttacking = false
		}
	}()

	return nil
}

// triggerDropLocked is the internal version called when mutex is already held
func (vms *virtualMinerSimulator) triggerDropLocked(miner *VirtualMiner, duration time.Duration) {
	miner.CurrentState.IsDisconnected = true
	miner.IsActive = false
	miner.Statistics.DropEvents++

	minerID := miner.ID
	// Schedule reconnection
	go func() {
		time.Sleep(duration)
		vms.mutex.Lock()
		defer vms.mutex.Unlock()

		if miner := vms.miners[minerID]; miner != nil {
			miner.CurrentState.IsDisconnected = false
			if vms.isRunning {
				miner.IsActive = true
				miner.CurrentState.LastSeen = time.Now()
			}
		}
	}()
}

// triggerAttackLocked is the internal version called when mutex is already held
func (vms *virtualMinerSimulator) triggerAttackLocked(miner *VirtualMiner, attackType string, duration time.Duration) {
	if !miner.IsMalicious {
		return
	}

	miner.AttackProfile.IsAttacking = true
	miner.AttackProfile.AttackStarted = time.Now()
	miner.AttackProfile.AttackDuration = duration
	miner.Statistics.AttackEvents++

	minerID := miner.ID
	// Schedule attack end
	go func() {
		time.Sleep(duration)
		vms.mutex.Lock()
		defer vms.mutex.Unlock()

		if miner := vms.miners[minerID]; miner != nil && miner.AttackProfile != nil {
			miner.AttackProfile.IsAttacking = false
		}
	}()
}

// UpdateMinerHashRate updates a miner's hash rate
func (vms *virtualMinerSimulator) UpdateMinerHashRate(minerID string, hashRate uint64) error {
	vms.mutex.Lock()
	defer vms.mutex.Unlock()

	miner := vms.miners[minerID]
	if miner == nil {
		return fmt.Errorf("miner %s not found", minerID)
	}

	miner.HashRate = hashRate
	return nil
}

// UpdateNetworkConditions updates a miner's network conditions
func (vms *virtualMinerSimulator) UpdateNetworkConditions(minerID string, conditions NetworkProfile) error {
	vms.mutex.Lock()
	defer vms.mutex.Unlock()

	miner := vms.miners[minerID]
	if miner == nil {
		return fmt.Errorf("miner %s not found", minerID)
	}

	miner.NetworkProfile = &conditions
	return nil
}

// Private helper methods

func (vms *virtualMinerSimulator) generateMiners() error {
	// Calculate miner type distribution
	typeDistribution := vms.calculateTypeDistribution()

	for i := 0; i < vms.config.MinerCount; i++ {
		minerType := vms.selectMinerType(typeDistribution)
		isMalicious := vms.shouldBeMalicious()

		miner := vms.createMiner(minerType, isMalicious)
		vms.miners[miner.ID] = miner
	}

	return nil
}

func (vms *virtualMinerSimulator) calculateTypeDistribution() []MinerType {
	if len(vms.config.MinerTypes) == 0 {
		// Default distribution
		return []MinerType{
			{Type: "GPU", Percentage: 1.0, HashRateMultiplier: 1.0},
		}
	}
	return vms.config.MinerTypes
}

func (vms *virtualMinerSimulator) selectMinerType(types []MinerType) MinerType {
	random := rand.Float64()
	cumulative := 0.0

	for _, minerType := range types {
		cumulative += minerType.Percentage
		if random <= cumulative {
			return minerType
		}
	}

	// Fallback to first type
	return types[0]
}

func (vms *virtualMinerSimulator) shouldBeMalicious() bool {
	if vms.config.MaliciousBehavior.MaliciousMinerPercentage == 0 {
		return false
	}
	return rand.Float64() < vms.config.MaliciousBehavior.MaliciousMinerPercentage
}

func (vms *virtualMinerSimulator) createMiner(minerType MinerType, isMalicious bool) *VirtualMiner {
	id := fmt.Sprintf("miner_%d_%d", time.Now().UnixNano(), rand.Intn(10000))

	// Calculate hash rate
	baseHashRate := vms.config.HashRateRange.Min +
		uint64(rand.Float64()*float64(vms.config.HashRateRange.Max-vms.config.HashRateRange.Min))
	hashRate := uint64(float64(baseHashRate) * minerType.HashRateMultiplier)

	miner := &VirtualMiner{
		ID:          id,
		Type:        minerType.Type,
		HashRate:    hashRate,
		IsActive:    false,
		IsMalicious: isMalicious,
		PerformanceProfile: &PerformanceProfile{
			PowerConsumption: minerType.PowerConsumption,
			EfficiencyRating: minerType.EfficiencyRating,
			FailureRate:      minerType.FailureRate,
			Temperature:      20.0 + rand.Float64()*60.0,     // 20-80°C
			FanSpeed:         uint32(1000 + rand.Intn(2000)), // 1000-3000 RPM
		},
		NetworkProfile: vms.createNetworkProfile(),
		CurrentState: &MinerState{
			LastSeen: time.Now(),
		},
		Statistics: &MinerStatistics{
			LastShareTime: time.Now(),
		},
	}

	// Set up malicious behavior if applicable
	if isMalicious {
		miner.AttackProfile = vms.createAttackProfile()
	}

	return miner
}

func (vms *virtualMinerSimulator) createNetworkProfile() *NetworkProfile {
	if len(vms.config.NetworkConditions.ConnectionQuality) == 0 {
		// Default network profile
		return &NetworkProfile{
			Quality:    "good",
			Latency:    time.Millisecond * time.Duration(50+rand.Intn(200)),
			PacketLoss: 0.01,
			Jitter:     time.Millisecond * time.Duration(5+rand.Intn(20)),
			Bandwidth:  1000000, // 1 Mbps
		}
	}

	// Select connection quality based on distribution
	quality := vms.selectConnectionQuality()

	// Generate latency within range
	latencyRange := vms.config.NetworkConditions.LatencyRange
	latencyDiff := latencyRange.Max - latencyRange.Min
	latency := latencyRange.Min + time.Duration(rand.Float64()*float64(latencyDiff))

	return &NetworkProfile{
		Quality:    quality.Quality,
		Latency:    latency,
		PacketLoss: quality.PacketLoss,
		Jitter:     quality.Jitter,
		Bandwidth:  uint64(1000000 + rand.Intn(9000000)), // 1-10 Mbps
	}
}

func (vms *virtualMinerSimulator) selectConnectionQuality() ConnectionQuality {
	if len(vms.config.NetworkConditions.ConnectionQuality) == 0 {
		return ConnectionQuality{Quality: "good", Percentage: 1.0, PacketLoss: 0.01, Jitter: time.Millisecond * 20}
	}

	random := rand.Float64()
	cumulative := 0.0

	for _, quality := range vms.config.NetworkConditions.ConnectionQuality {
		cumulative += quality.Percentage
		if random <= cumulative {
			return quality
		}
	}

	// Fallback to first quality
	return vms.config.NetworkConditions.ConnectionQuality[0]
}

func (vms *virtualMinerSimulator) createAttackProfile() *AttackProfile {
	if len(vms.config.MaliciousBehavior.AttackTypes) == 0 {
		return &AttackProfile{
			AttackTypes: []AttackType{
				{Type: "invalid_shares", Probability: 0.5, Intensity: 0.2},
			},
		}
	}

	return &AttackProfile{
		AttackTypes: vms.config.MaliciousBehavior.AttackTypes,
	}
}

func (vms *virtualMinerSimulator) simulateBehaviors() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-vms.stopChan:
			return
		case <-ticker.C:
			vms.processBehaviors()
		}
	}
}

func (vms *virtualMinerSimulator) processBehaviors() {
	vms.mutex.Lock()
	defer vms.mutex.Unlock()

	for _, miner := range vms.miners {
		if !miner.IsActive {
			continue
		}

		// Process burst mining
		vms.processBurstMining(miner)

		// Process connection drops
		vms.processConnectionDrops(miner)

		// Process malicious behavior
		if miner.IsMalicious {
			vms.processMaliciousBehavior(miner)
		}

		// Simulate share submission
		vms.simulateShareSubmission(miner)

		// Update miner state
		miner.CurrentState.LastSeen = time.Now()
	}
}

func (vms *virtualMinerSimulator) processBurstMining(miner *VirtualMiner) {
	config := vms.config.BehaviorPatterns.BurstMining

	// Check if burst should end
	if miner.CurrentState.IsBursting {
		if time.Since(miner.CurrentState.BurstStarted) >= miner.CurrentState.BurstDuration {
			miner.CurrentState.IsBursting = false
		}
		return
	}

	// Check if burst should start
	if config.Probability > 0 && rand.Float64() < config.Probability/3600 { // Per second probability
		duration := config.DurationRange.Min +
			time.Duration(rand.Float64()*float64(config.DurationRange.Max-config.DurationRange.Min))

		miner.CurrentState.IsBursting = true
		miner.CurrentState.BurstStarted = time.Now()
		miner.CurrentState.BurstDuration = duration
		miner.Statistics.BurstEvents++
	}
}

func (vms *virtualMinerSimulator) processConnectionDrops(miner *VirtualMiner) {
	config := vms.config.BehaviorPatterns.ConnectionDrops

	if config.Probability > 0 && rand.Float64() < config.Probability/3600 { // Per second probability
		duration := config.DurationRange.Min +
			time.Duration(rand.Float64()*float64(config.DurationRange.Max-config.DurationRange.Min))

		// Use internal version since we already hold the lock
		vms.triggerDropLocked(miner, duration)
	}
}

func (vms *virtualMinerSimulator) processMaliciousBehavior(miner *VirtualMiner) {
	if miner.AttackProfile.IsAttacking {
		// Continue current attack
		if time.Since(miner.AttackProfile.AttackStarted) >= miner.AttackProfile.AttackDuration {
			miner.AttackProfile.IsAttacking = false
		}
		return
	}

	// Check if attack should start
	for _, attackType := range miner.AttackProfile.AttackTypes {
		if rand.Float64() < attackType.Probability/3600 { // Per second probability
			duration := time.Duration(30+rand.Intn(300)) * time.Second // 30s to 5min
			// Use internal version since we already hold the lock
			vms.triggerAttackLocked(miner, attackType.Type, duration)
			break
		}
	}
}

func (vms *virtualMinerSimulator) simulateShareSubmission(miner *VirtualMiner) {
	// Calculate shares per second based on hash rate and difficulty
	// Simplified: assume 1 share per 10 seconds at base hash rate
	baseShareRate := float64(miner.HashRate) / 10000000.0 // shares per second

	// Apply burst multiplier if bursting
	if miner.CurrentState.IsBursting {
		baseShareRate *= vms.config.BehaviorPatterns.BurstMining.IntensityMultiplier
	}

	// Simulate share submission
	if rand.Float64() < baseShareRate {
		miner.CurrentState.SharesSubmitted++
		miner.Statistics.TotalShares++

		// Determine if share is valid
		isValid := true
		if miner.IsMalicious && miner.AttackProfile.IsAttacking {
			// Apply attack effects
			for _, attackType := range miner.AttackProfile.AttackTypes {
				if attackType.Type == "invalid_shares" && rand.Float64() < attackType.Intensity {
					isValid = false
					break
				}
			}
		}

		if isValid {
			miner.CurrentState.ValidShares++
			miner.Statistics.ValidShares++
		} else {
			miner.CurrentState.InvalidShares++
			miner.Statistics.InvalidShares++
		}

		miner.Statistics.LastShareTime = time.Now()
	}
}

func (vms *virtualMinerSimulator) updateStatistics() {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for {
		select {
		case <-vms.stopChan:
			return
		case <-ticker.C:
			vms.mutex.Lock()
			vms.calculateStatsLocked()
			vms.mutex.Unlock()
		}
	}
}

// calculateStatsLocked updates stats - caller MUST hold vms.mutex
func (vms *virtualMinerSimulator) calculateStatsLocked() {

	totalMiners := uint32(len(vms.miners))
	activeMiners := uint32(0)
	totalHashRate := uint64(0)
	totalShares := uint64(0)
	validShares := uint64(0)
	invalidShares := uint64(0)
	totalBurstEvents := uint64(0)
	totalDropEvents := uint64(0)
	totalAttackEvents := uint64(0)

	for _, miner := range vms.miners {
		if miner.IsActive {
			activeMiners++
			totalHashRate += miner.HashRate
		}

		totalShares += miner.Statistics.TotalShares
		validShares += miner.Statistics.ValidShares
		invalidShares += miner.Statistics.InvalidShares
		totalBurstEvents += miner.Statistics.BurstEvents
		totalDropEvents += miner.Statistics.DropEvents
		totalAttackEvents += miner.Statistics.AttackEvents
	}

	averageHashRate := uint64(0)
	if activeMiners > 0 {
		averageHashRate = totalHashRate / uint64(activeMiners)
	}

	uptimePercentage := float64(activeMiners) / float64(totalMiners) * 100.0
	if totalMiners == 0 {
		uptimePercentage = 0
	}

	simulationTime := time.Duration(0)
	if vms.isRunning {
		simulationTime = time.Since(vms.startTime)
	}

	vms.stats = &SimulationStats{
		TotalMiners:       totalMiners,
		ActiveMiners:      activeMiners,
		TotalHashRate:     totalHashRate,
		AverageHashRate:   averageHashRate,
		TotalShares:       totalShares,
		ValidShares:       validShares,
		InvalidShares:     invalidShares,
		TotalBurstEvents:  totalBurstEvents,
		TotalDropEvents:   totalDropEvents,
		TotalAttackEvents: totalAttackEvents,
		UptimePercentage:  uptimePercentage,
		SimulationTime:    simulationTime,
	}
}
