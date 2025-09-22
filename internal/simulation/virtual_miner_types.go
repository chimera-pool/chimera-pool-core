package simulation

import (
	"time"
)

// VirtualMinerConfig defines configuration for virtual miner simulation
type VirtualMinerConfig struct {
	MinerCount        int
	HashRateRange     HashRateRange
	MinerTypes        []MinerType
	NetworkConditions NetworkConditionsConfig
	BehaviorPatterns  BehaviorPatternsConfig
	MaliciousBehavior MaliciousBehaviorConfig
}

// HashRateRange defines the range of hashrates for miners
type HashRateRange struct {
	Min uint64
	Max uint64
}

// MinerType defines characteristics of different miner types
type MinerType struct {
	Type                string
	Percentage          float64
	HashRateMultiplier  float64
	PowerConsumption    uint32  // Watts
	EfficiencyRating    float64 // 0.0 to 1.0
	FailureRate         float64 // Probability of failure
}

// NetworkConditionsConfig defines network simulation parameters
type NetworkConditionsConfig struct {
	LatencyRange      LatencyRange
	ConnectionQuality []ConnectionQuality
}

// LatencyRange defines network latency range
type LatencyRange struct {
	Min time.Duration
	Max time.Duration
}

// ConnectionQuality defines different connection quality levels
type ConnectionQuality struct {
	Quality    string
	Percentage float64
	PacketLoss float64
	Jitter     time.Duration
}

// BehaviorPatternsConfig defines miner behavior patterns
type BehaviorPatternsConfig struct {
	BurstMining     BurstMiningConfig
	ConnectionDrops ConnectionDropsConfig
}

// BurstMiningConfig defines burst mining behavior
type BurstMiningConfig struct {
	Probability         float64
	DurationRange       DurationRange
	IntensityMultiplier float64
}

// ConnectionDropsConfig defines connection drop behavior
type ConnectionDropsConfig struct {
	Probability    float64
	DurationRange  DurationRange
	ReconnectDelay time.Duration
}

// DurationRange defines a time duration range
type DurationRange struct {
	Min time.Duration
	Max time.Duration
}

// MaliciousBehaviorConfig defines malicious miner behavior
type MaliciousBehaviorConfig struct {
	MaliciousMinerPercentage float64
	AttackTypes              []AttackType
}

// AttackType defines different types of attacks
type AttackType struct {
	Type        string  // "invalid_shares", "share_withholding", "difficulty_manipulation"
	Probability float64
	Intensity   float64
}

// VirtualMiner represents a simulated miner
type VirtualMiner struct {
	ID                 string
	Type               string
	HashRate           uint64
	IsActive           bool
	IsMalicious        bool
	Location           string
	PerformanceProfile *PerformanceProfile
	NetworkProfile     *NetworkProfile
	AttackProfile      *AttackProfile
	CurrentState       *MinerState
	Statistics         *MinerStatistics
}

// PerformanceProfile defines miner performance characteristics
type PerformanceProfile struct {
	PowerConsumption uint32
	EfficiencyRating float64
	FailureRate      float64
	Temperature      float64
	FanSpeed         uint32
}

// NetworkProfile defines miner network characteristics
type NetworkProfile struct {
	Quality    string
	Latency    time.Duration
	PacketLoss float64
	Jitter     time.Duration
	Bandwidth  uint64 // bits per second
}

// AttackProfile defines malicious behavior profile
type AttackProfile struct {
	AttackTypes    []AttackType
	IsAttacking    bool
	AttackStarted  time.Time
	AttackDuration time.Duration
}

// MinerState represents current miner state
type MinerState struct {
	IsBursting      bool
	IsDisconnected  bool
	BurstStarted    time.Time
	BurstDuration   time.Duration
	LastSeen        time.Time
	SharesSubmitted uint64
	ValidShares     uint64
	InvalidShares   uint64
}

// MinerStatistics tracks miner performance statistics
type MinerStatistics struct {
	TotalShares       uint64
	ValidShares       uint64
	InvalidShares     uint64
	TotalHashRate     uint64
	AverageHashRate   uint64
	UptimePercentage  float64
	LastShareTime     time.Time
	BurstEvents       uint64
	DropEvents        uint64
	AttackEvents      uint64
}

// SimulationStats provides overall simulation statistics
type SimulationStats struct {
	TotalMiners       uint32
	ActiveMiners      uint32
	TotalHashRate     uint64
	AverageHashRate   uint64
	TotalShares       uint64
	ValidShares       uint64
	InvalidShares     uint64
	TotalBurstEvents  uint64
	TotalDropEvents   uint64
	TotalAttackEvents uint64
	UptimePercentage  float64
	SimulationTime    time.Duration
}

// VirtualMinerSimulator interface defines virtual miner simulation capabilities
type VirtualMinerSimulator interface {
	// Lifecycle management
	Start() error
	Stop() error
	
	// Miner management
	GetMiners() []*VirtualMiner
	GetMiner(id string) *VirtualMiner
	AddMiner(config MinerType) (*VirtualMiner, error)
	RemoveMiner(id string) error
	
	// Statistics and monitoring
	GetSimulationStats() *SimulationStats
	GetMinerStats(id string) *MinerStatistics
	
	// Behavior control
	TriggerBurst(minerID string, duration time.Duration) error
	TriggerDrop(minerID string, duration time.Duration) error
	TriggerAttack(minerID string, attackType string, duration time.Duration) error
	
	// Configuration
	UpdateMinerHashRate(minerID string, hashRate uint64) error
	UpdateNetworkConditions(minerID string, conditions NetworkProfile) error
}