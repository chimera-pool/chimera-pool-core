package simulation

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

// blockchainSimulator implements the BlockchainSimulator interface
type blockchainSimulator struct {
	config          BlockchainConfig
	chain           []*Block
	currentBlock    *Block
	difficulty      uint64
	isRunning       bool
	transactionPool []Transaction
	networkStats    NetworkStats
	mutex           sync.RWMutex
	stopChan        chan struct{}
	txGenerator     *transactionGenerator
}

// NewBlockchainSimulator creates a new blockchain simulator instance
func NewBlockchainSimulator(config BlockchainConfig) (BlockchainSimulator, error) {
	simulator := &blockchainSimulator{
		config:      config,
		chain:       make([]*Block, 0),
		difficulty:  config.InitialDifficulty,
		stopChan:    make(chan struct{}),
		txGenerator: newTransactionGenerator(config.TransactionLoad),
	}

	// Create genesis block
	genesis := simulator.createGenesisBlock()
	simulator.chain = append(simulator.chain, genesis)
	simulator.currentBlock = genesis

	// Initialize network stats
	simulator.networkStats = NetworkStats{
		NetworkType:       config.NetworkType,
		AverageBlockTime:  config.BlockTime,
		CurrentDifficulty: config.InitialDifficulty,
		BlocksGenerated:   1, // Genesis block
	}

	return simulator, nil
}

// Start begins the blockchain simulation
func (bs *blockchainSimulator) Start() error {
	bs.mutex.Lock()
	defer bs.mutex.Unlock()

	if bs.isRunning {
		return fmt.Errorf("simulator is already running")
	}

	bs.isRunning = true

	// Start transaction generation if configured
	if bs.config.TransactionLoad.TxPerSecond > 0 {
		go bs.generateTransactions()
	}

	return nil
}

// Stop halts the blockchain simulation
func (bs *blockchainSimulator) Stop() error {
	bs.mutex.Lock()
	defer bs.mutex.Unlock()

	if !bs.isRunning {
		return nil
	}

	bs.isRunning = false
	close(bs.stopChan)
	return nil
}

// GetNetworkType returns the configured network type
func (bs *blockchainSimulator) GetNetworkType() string {
	return bs.config.NetworkType
}

// GetBlockTime returns the configured block time
func (bs *blockchainSimulator) GetBlockTime() time.Duration {
	return bs.config.BlockTime
}

// GetCurrentDifficulty returns the current mining difficulty
func (bs *blockchainSimulator) GetCurrentDifficulty() uint64 {
	bs.mutex.RLock()
	defer bs.mutex.RUnlock()
	return bs.difficulty
}

// GetGenesisBlock returns the genesis block
func (bs *blockchainSimulator) GetGenesisBlock() *Block {
	bs.mutex.RLock()
	defer bs.mutex.RUnlock()

	if len(bs.chain) > 0 {
		// Return a COPY to prevent race conditions
		return bs.copyBlock(bs.chain[0])
	}
	return nil
}

// copyBlock creates a deep copy of a Block
func (bs *blockchainSimulator) copyBlock(b *Block) *Block {
	if b == nil {
		return nil
	}
	copy := &Block{
		Height:       b.Height,
		Hash:         b.Hash,
		PreviousHash: b.PreviousHash,
		Timestamp:    b.Timestamp,
		Difficulty:   b.Difficulty,
		Nonce:        b.Nonce,
		MinerID:      b.MinerID,
	}
	if b.Transactions != nil {
		copy.Transactions = make([]Transaction, len(b.Transactions))
		for i, tx := range b.Transactions {
			copy.Transactions[i] = Transaction{
				ID:        tx.ID,
				From:      tx.From,
				To:        tx.To,
				Amount:    tx.Amount,
				Fee:       tx.Fee,
				Timestamp: tx.Timestamp,
			}
		}
	}
	return copy
}

// MineNextBlock mines the next block in the chain
func (bs *blockchainSimulator) MineNextBlock() (*Block, error) {
	return bs.MineBlockWithMiner(0) // Default miner ID
}

// MineBlockWithMiner mines a block with a specific miner ID
func (bs *blockchainSimulator) MineBlockWithMiner(minerID int) (*Block, error) {
	bs.mutex.Lock()
	defer bs.mutex.Unlock()

	// Get transactions for the block
	transactions := bs.getTransactionsForBlock()

	// Create new block
	newBlock := &Block{
		Height:       bs.currentBlock.Height + 1,
		PreviousHash: bs.currentBlock.Hash,
		Timestamp:    time.Now(),
		Difficulty:   bs.difficulty,
		Transactions: transactions,
		MinerID:      minerID,
	}

	// Mine the block (simulate proof of work)
	bs.mineBlock(newBlock)

	// Add to chain
	bs.chain = append(bs.chain, newBlock)
	bs.currentBlock = newBlock

	// Update difficulty if needed
	bs.adjustDifficulty()

	// Update statistics
	bs.updateNetworkStats()

	// Return a COPY to prevent race conditions
	return bs.copyBlock(newBlock), nil
}

// ValidateChain validates the entire blockchain
func (bs *blockchainSimulator) ValidateChain() bool {
	bs.mutex.RLock()
	defer bs.mutex.RUnlock()

	if len(bs.chain) == 0 {
		return false
	}

	// Validate genesis block
	genesis := bs.chain[0]
	if genesis.Height != 0 || genesis.PreviousHash != "" {
		return false
	}

	// Validate subsequent blocks
	for i := 1; i < len(bs.chain); i++ {
		current := bs.chain[i]
		previous := bs.chain[i-1]

		// Check height sequence
		if current.Height != previous.Height+1 {
			return false
		}

		// Check previous hash reference
		if current.PreviousHash != previous.Hash {
			return false
		}

		// Validate block hash
		if !bs.validateBlockHash(current) {
			return false
		}
	}

	return true
}

// GetNetworkStats returns current network statistics
func (bs *blockchainSimulator) GetNetworkStats() NetworkStats {
	bs.mutex.RLock()
	defer bs.mutex.RUnlock()
	return bs.networkStats
}

// Private helper methods

func (bs *blockchainSimulator) createGenesisBlock() *Block {
	genesis := &Block{
		Height:       0,
		Hash:         "",
		PreviousHash: "",
		Timestamp:    time.Now(),
		Difficulty:   bs.config.InitialDifficulty,
		Nonce:        0,
		Transactions: []Transaction{},
		MinerID:      -1, // Special ID for genesis
	}

	// Calculate genesis hash
	genesis.Hash = bs.calculateBlockHash(genesis)
	return genesis
}

func (bs *blockchainSimulator) mineBlock(block *Block) {
	target := bs.calculateTarget(block.Difficulty)
	startTime := time.Now()

	// Simulate mining work based on difficulty
	miningTime := bs.calculateMiningTime(block.Difficulty)

	// Add some randomness to simulate real mining
	jitter := time.Duration(rand.Float64() * float64(miningTime) * 0.2) // ±20% jitter
	if rand.Float64() < 0.5 {
		miningTime += jitter
	} else {
		miningTime -= jitter
	}

	// Simulate network latency if configured
	if bs.config.NetworkLatency.MinLatency > 0 {
		latency := bs.simulateNetworkLatency()
		miningTime += latency
	}

	// Sleep to simulate mining time
	time.Sleep(miningTime)

	// Set nonce and calculate final hash
	block.Nonce = uint64(rand.Int63())
	block.Hash = bs.calculateBlockHash(block)

	// Ensure hash meets difficulty target (simplified)
	for !bs.hashMeetsTarget(block.Hash, target) {
		block.Nonce++
		block.Hash = bs.calculateBlockHash(block)
	}

	// Update mining duration in block
	block.Timestamp = startTime.Add(miningTime)
}

func (bs *blockchainSimulator) calculateMiningTime(difficulty uint64) time.Duration {
	// Base mining time scaled by difficulty
	baseTime := bs.config.BlockTime

	// Adjust based on difficulty relative to initial difficulty
	difficultyRatio := float64(difficulty) / float64(bs.config.InitialDifficulty)
	adjustedTime := time.Duration(float64(baseTime) * difficultyRatio)

	// Ensure minimum time
	minTime := time.Millisecond * 100
	if adjustedTime < minTime {
		adjustedTime = minTime
	}

	return adjustedTime
}

func (bs *blockchainSimulator) simulateNetworkLatency() time.Duration {
	config := bs.config.NetworkLatency

	switch config.Distribution {
	case "uniform":
		diff := config.MaxLatency - config.MinLatency
		return config.MinLatency + time.Duration(rand.Float64()*float64(diff))

	case "normal":
		mean := float64(config.MinLatency+config.MaxLatency) / 2
		stddev := float64(config.MaxLatency-config.MinLatency) / 6 // 99.7% within range
		latency := rand.NormFloat64()*stddev + mean

		// Clamp to range
		if latency < float64(config.MinLatency) {
			latency = float64(config.MinLatency)
		}
		if latency > float64(config.MaxLatency) {
			latency = float64(config.MaxLatency)
		}

		return time.Duration(latency)

	case "exponential":
		// Exponential distribution with mean at 1/3 of the range
		lambda := 3.0 / float64(config.MaxLatency-config.MinLatency)
		latency := rand.ExpFloat64() / lambda
		return config.MinLatency + time.Duration(latency)

	default:
		return config.MinLatency
	}
}

func (bs *blockchainSimulator) calculateBlockHash(block *Block) string {
	data := fmt.Sprintf("%d%s%d%d%d",
		block.Height,
		block.PreviousHash,
		block.Timestamp.Unix(),
		block.Difficulty,
		block.Nonce)

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (bs *blockchainSimulator) calculateTarget(difficulty uint64) string {
	// Simplified target calculation
	// In reality, this would be more complex
	targetValue := math.MaxUint64 / difficulty
	return fmt.Sprintf("%016x", targetValue)
}

func (bs *blockchainSimulator) hashMeetsTarget(hash, target string) bool {
	// Simplified difficulty check
	// Count leading zeros in hash
	leadingZeros := 0
	for _, char := range hash {
		if char == '0' {
			leadingZeros++
		} else {
			break
		}
	}

	// Require number of leading zeros based on difficulty
	requiredZeros := int(math.Log2(float64(bs.difficulty))) / 4
	return leadingZeros >= requiredZeros
}

func (bs *blockchainSimulator) validateBlockHash(block *Block) bool {
	expectedHash := bs.calculateBlockHash(block)
	return expectedHash == block.Hash
}

func (bs *blockchainSimulator) adjustDifficulty() {
	if len(bs.chain) < bs.config.DifficultyAdjustmentWindow {
		return
	}

	// Check if we need to adjust difficulty
	if len(bs.chain)%bs.config.DifficultyAdjustmentWindow == 0 {
		bs.performDifficultyAdjustment()
	}
}

func (bs *blockchainSimulator) performDifficultyAdjustment() {
	windowSize := bs.config.DifficultyAdjustmentWindow
	if len(bs.chain) < windowSize {
		return
	}

	// Calculate actual time for last window
	startBlock := bs.chain[len(bs.chain)-windowSize]
	endBlock := bs.chain[len(bs.chain)-1]
	actualTime := endBlock.Timestamp.Sub(startBlock.Timestamp)

	// Expected time for the window
	expectedTime := time.Duration(windowSize) * bs.config.BlockTime

	// Calculate adjustment ratio
	ratio := float64(actualTime) / float64(expectedTime)

	// Apply custom difficulty curve if configured
	if bs.config.CustomDifficultyCurve != nil {
		ratio = bs.applyCustomDifficultyCurve(ratio)
	}

	// Limit adjustment to prevent extreme changes
	if ratio > 4.0 {
		ratio = 4.0
	} else if ratio < 0.25 {
		ratio = 0.25
	}

	// Update difficulty
	newDifficulty := uint64(float64(bs.difficulty) * ratio)
	if newDifficulty < 1 {
		newDifficulty = 1
	}

	bs.difficulty = newDifficulty
}

func (bs *blockchainSimulator) applyCustomDifficultyCurve(ratio float64) float64 {
	curve := bs.config.CustomDifficultyCurve

	switch curve.Type {
	case "exponential":
		growthRate := curve.Parameters["growth_rate"]
		if growthRate == 0 {
			growthRate = 1.1
		}
		return math.Pow(growthRate, ratio-1.0)

	case "logarithmic":
		base := curve.Parameters["base"]
		if base == 0 {
			base = 2.0
		}
		return math.Log(ratio*base+1) / math.Log(base+1)

	default:
		return ratio
	}
}

func (bs *blockchainSimulator) getTransactionsForBlock() []Transaction {
	// Get pending transactions from pool
	maxTx := 1000 // Maximum transactions per block

	// NOTE: Caller must hold bs.mutex.Lock() - do not acquire here to avoid deadlock

	if len(bs.transactionPool) == 0 {
		return []Transaction{}
	}

	count := len(bs.transactionPool)
	if count > maxTx {
		count = maxTx
	}

	transactions := make([]Transaction, count)
	copy(transactions, bs.transactionPool[:count])

	// Remove used transactions from pool
	bs.transactionPool = bs.transactionPool[count:]

	return transactions
}

func (bs *blockchainSimulator) generateTransactions() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-bs.stopChan:
			return
		case <-ticker.C:
			bs.mutex.RLock()
			running := bs.isRunning
			bs.mutex.RUnlock()
			if running {
				bs.addGeneratedTransactions()
			}
		}
	}
}

func (bs *blockchainSimulator) addGeneratedTransactions() {
	bs.mutex.RLock()
	config := bs.config.TransactionLoad
	bs.mutex.RUnlock()

	// Calculate number of transactions to generate
	txCount := int(config.TxPerSecond)

	// Check for burst
	if rand.Float64() < config.BurstProbability {
		txCount = int(float64(txCount) * config.BurstMultiplier)
	}

	// Generate transactions
	for i := 0; i < txCount; i++ {
		tx := bs.txGenerator.generateTransaction()

		bs.mutex.Lock()
		bs.transactionPool = append(bs.transactionPool, tx)
		bs.mutex.Unlock()
	}
}

func (bs *blockchainSimulator) updateNetworkStats() {
	bs.networkStats.CurrentDifficulty = bs.difficulty
	bs.networkStats.BlocksGenerated = uint64(len(bs.chain))

	// Calculate average block time from last 10 blocks
	if len(bs.chain) >= 2 {
		windowSize := 10
		if len(bs.chain) < windowSize {
			windowSize = len(bs.chain)
		}

		startIdx := len(bs.chain) - windowSize
		totalTime := bs.chain[len(bs.chain)-1].Timestamp.Sub(bs.chain[startIdx].Timestamp)
		bs.networkStats.AverageBlockTime = totalTime / time.Duration(windowSize-1)
	}

	// Update transaction count
	totalTx := uint64(0)
	for _, block := range bs.chain {
		totalTx += uint64(len(block.Transactions))
	}
	bs.networkStats.TotalTransactions = totalTx

	// Simulate network latency stats
	if bs.config.NetworkLatency.MinLatency > 0 {
		bs.networkStats.AverageLatency = (bs.config.NetworkLatency.MinLatency + bs.config.NetworkLatency.MaxLatency) / 2
	}

	// Estimate hash rate based on difficulty and block time
	avgBlockSeconds := bs.networkStats.AverageBlockTime.Seconds()
	if avgBlockSeconds > 0 {
		bs.networkStats.HashRate = bs.difficulty * 1000000 / uint64(avgBlockSeconds)
	}
}

// transactionGenerator generates realistic transactions
type transactionGenerator struct {
	config TransactionLoadConfig
	nonce  uint64
	mu     sync.Mutex
}

func newTransactionGenerator(config TransactionLoadConfig) *transactionGenerator {
	return &transactionGenerator{
		config: config,
		nonce:  0,
	}
}

func (tg *transactionGenerator) generateTransaction() Transaction {
	tg.mu.Lock()
	tg.nonce++
	nonce := tg.nonce
	tg.mu.Unlock()

	return Transaction{
		ID:        fmt.Sprintf("tx_%d_%d", time.Now().Unix(), nonce),
		From:      fmt.Sprintf("addr_%d", rand.Intn(1000)),
		To:        fmt.Sprintf("addr_%d", rand.Intn(1000)),
		Amount:    uint64(rand.Intn(1000000) + 1),
		Fee:       uint64(rand.Intn(1000) + 1),
		Timestamp: time.Now(),
	}
}
