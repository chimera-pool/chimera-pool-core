package simulation

import (
	"time"
)

// BlockchainConfig defines configuration for blockchain simulation
type BlockchainConfig struct {
	NetworkType                string
	BlockTime                  time.Duration
	InitialDifficulty          uint64
	DifficultyAdjustmentWindow int
	MaxBlockSize               int
	CustomDifficultyCurve      *DifficultyCurve
	NetworkLatency             NetworkLatencyConfig
	TransactionLoad            TransactionLoadConfig
}

// DifficultyCurve defines custom difficulty adjustment algorithms
type DifficultyCurve struct {
	Type       string
	Parameters map[string]float64
}

// NetworkLatencyConfig defines network latency simulation parameters
type NetworkLatencyConfig struct {
	MinLatency   time.Duration
	MaxLatency   time.Duration
	Distribution string // "uniform", "normal", "exponential"
}

// TransactionLoadConfig defines transaction generation parameters
type TransactionLoadConfig struct {
	TxPerSecond      float64
	BurstProbability float64
	BurstMultiplier  float64
}

// Block represents a blockchain block
type Block struct {
	Height       uint64
	Hash         string
	PreviousHash string
	Timestamp    time.Time
	Difficulty   uint64
	Nonce        uint64
	Transactions []Transaction
	MinerID      int
}

// Transaction represents a blockchain transaction
type Transaction struct {
	ID        string
	From      string
	To        string
	Amount    uint64
	Fee       uint64
	Timestamp time.Time
}

// NetworkStats provides statistics about the simulated network
type NetworkStats struct {
	NetworkType        string
	AverageBlockTime   time.Duration
	CurrentDifficulty  uint64
	TotalTransactions  uint64
	AverageLatency     time.Duration
	BlocksGenerated    uint64
	HashRate           uint64
}

// BlockchainSimulator interface defines the blockchain simulation capabilities
type BlockchainSimulator interface {
	// Configuration and lifecycle
	Start() error
	Stop() error
	GetNetworkType() string
	GetBlockTime() time.Duration
	GetCurrentDifficulty() uint64

	// Block operations
	GetGenesisBlock() *Block
	MineNextBlock() (*Block, error)
	MineBlockWithMiner(minerID int) (*Block, error)
	ValidateChain() bool

	// Statistics and monitoring
	GetNetworkStats() NetworkStats
}