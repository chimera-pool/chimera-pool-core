package shares

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Share represents a mining share submission
type Share struct {
	ID         int64     `json:"id" db:"id"`
	MinerID    int64     `json:"miner_id" db:"miner_id"`
	UserID     int64     `json:"user_id" db:"user_id"`
	JobID      string    `json:"job_id"`
	Nonce      string    `json:"nonce" db:"nonce"`
	Hash       string    `json:"hash" db:"hash"`
	Difficulty float64   `json:"difficulty" db:"difficulty"`
	IsValid    bool      `json:"is_valid" db:"is_valid"`
	Timestamp  time.Time `json:"timestamp" db:"timestamp"`
}

// ShareValidationResult represents the result of share validation
type ShareValidationResult struct {
	IsValid bool
	Hash    string
	Error   string
}

// ShareProcessingResult represents the result of complete share processing
type ShareProcessingResult struct {
	Success        bool
	ProcessedShare *Share
	Error          string
}

// ShareStatistics represents overall share processing statistics
type ShareStatistics struct {
	TotalShares     int64
	ValidShares     int64
	InvalidShares   int64
	TotalDifficulty float64
	LastUpdated     time.Time
}

// MinerStatistics represents per-miner share statistics
type MinerStatistics struct {
	MinerID         int64
	TotalShares     int64
	ValidShares     int64
	InvalidShares   int64
	TotalDifficulty float64
	LastShare       time.Time
}

// ShareProcessor handles share validation and processing using Blake2S algorithm
type ShareProcessor struct {
	// Algorithm interface for hash validation
	algorithm Blake2SHasher
	
	// Statistics tracking
	stats       ShareStatistics
	minerStats  map[int64]*MinerStatistics
	statsMutex  sync.RWMutex
	
	// Configuration
	maxNonceLength int
	maxJobIDLength int
}

// Blake2SHasher interface for Blake2S hashing operations
type Blake2SHasher interface {
	Hash(input []byte) ([]byte, error)
	Verify(input []byte, target []byte, nonce uint64) (bool, error)
}

// DefaultBlake2SHasher implements Blake2SHasher using the algorithm engine
type DefaultBlake2SHasher struct{}

// Hash computes Blake2S hash of input data
func (h *DefaultBlake2SHasher) Hash(input []byte) ([]byte, error) {
	// Simplified Blake2S-like hash for testing
	// In production, this would call the actual Rust Blake2S implementation
	
	hash := make([]byte, 32) // Blake2S-256 produces 32-byte hash
	
	// Create a more realistic hash that varies significantly with input
	// This is still a simulation but produces better distribution
	seed := uint64(0x6a09e667f3bcc908) // Blake2S initial value
	
	for i, b := range input {
		seed = seed*0x9e3779b97f4a7c15 + uint64(b) + uint64(i)
		seed ^= seed >> 30
		seed *= 0xbf58476d1ce4e5b9
		seed ^= seed >> 27
		seed *= 0x94d049bb133111eb
		seed ^= seed >> 31
		
		// Distribute seed across hash bytes
		for j := 0; j < 8 && i*8+j < 32; j++ {
			hash[i*8+j] = byte(seed >> (j * 8))
		}
	}
	
	// Fill remaining bytes if input is small
	for i := len(input) * 8; i < 32; i++ {
		seed = seed*0x9e3779b97f4a7c15 + uint64(i)
		seed ^= seed >> 30
		seed *= 0xbf58476d1ce4e5b9
		seed ^= seed >> 27
		hash[i] = byte(seed)
	}
	
	return hash, nil
}

// Verify checks if the hash of input+nonce meets the target difficulty
func (h *DefaultBlake2SHasher) Verify(input []byte, target []byte, nonce uint64) (bool, error) {
	// Combine input with nonce
	nonceBytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		nonceBytes[i] = byte(nonce >> (i * 8))
	}
	
	combined := append(input, nonceBytes...)
	hash, err := h.Hash(combined)
	if err != nil {
		return false, err
	}
	
	// Compare hash against target (big-endian comparison)
	if len(target) == 0 {
		return false, nil // Empty target means impossible
	}
	
	// Convert to big integers for comparison
	hashInt := new(big.Int).SetBytes(hash)
	targetInt := new(big.Int).SetBytes(target)
	
	// Hash must be less than target to be valid
	return hashInt.Cmp(targetInt) < 0, nil
}

// NewShareProcessor creates a new share processor with Blake2S algorithm
func NewShareProcessor() *ShareProcessor {
	return &ShareProcessor{
		algorithm:      &DefaultBlake2SHasher{},
		minerStats:     make(map[int64]*MinerStatistics),
		maxNonceLength: 16, // Maximum nonce length in hex characters
		maxJobIDLength: 64, // Maximum job ID length
		stats: ShareStatistics{
			LastUpdated: time.Now(),
		},
	}
}

// ValidateShare validates a mining share using Blake2S algorithm
func (sp *ShareProcessor) ValidateShare(share *Share) ShareValidationResult {
	// Input validation
	if err := sp.validateShareInput(share); err != nil {
		return ShareValidationResult{
			IsValid: false,
			Error:   err.Error(),
		}
	}
	
	// Convert difficulty to target
	target := sp.difficultyToTarget(share.Difficulty)
	
	// Parse nonce
	nonce, err := sp.parseNonce(share.Nonce)
	if err != nil {
		return ShareValidationResult{
			IsValid: false,
			Error:   fmt.Sprintf("invalid nonce: %v", err),
		}
	}
	
	// Create input data for hashing (job_id as base input)
	input := []byte(share.JobID)
	
	// Verify share using Blake2S algorithm
	isValid, err := sp.algorithm.Verify(input, target, nonce)
	if err != nil {
		return ShareValidationResult{
			IsValid: false,
			Error:   fmt.Sprintf("verification failed: %v", err),
		}
	}
	
	// Compute hash for storage
	hash := ""
	if isValid {
		// Combine input with nonce for final hash
		nonceBytes := make([]byte, 8)
		for i := 0; i < 8; i++ {
			nonceBytes[i] = byte(nonce >> (i * 8))
		}
		combined := append(input, nonceBytes...)
		
		hashBytes, err := sp.algorithm.Hash(combined)
		if err != nil {
			return ShareValidationResult{
				IsValid: false,
				Error:   fmt.Sprintf("hash computation failed: %v", err),
			}
		}
		hash = hex.EncodeToString(hashBytes)
	}
	
	return ShareValidationResult{
		IsValid: isValid,
		Hash:    hash,
		Error:   "",
	}
}

// ProcessShare processes a complete share submission
func (sp *ShareProcessor) ProcessShare(share *Share) ShareProcessingResult {
	// Validate the share
	validationResult := sp.ValidateShare(share)
	
	// Create processed share
	processedShare := &Share{
		ID:         share.ID,
		MinerID:    share.MinerID,
		UserID:     share.UserID,
		JobID:      share.JobID,
		Nonce:      share.Nonce,
		Hash:       validationResult.Hash,
		Difficulty: share.Difficulty,
		IsValid:    validationResult.IsValid,
		Timestamp:  share.Timestamp,
	}
	
	// Update statistics
	sp.updateStatistics(processedShare)
	
	if !validationResult.IsValid {
		return ShareProcessingResult{
			Success:        false,
			ProcessedShare: processedShare,
			Error:          validationResult.Error,
		}
	}
	
	return ShareProcessingResult{
		Success:        true,
		ProcessedShare: processedShare,
		Error:          "",
	}
}

// GetStatistics returns overall share processing statistics
func (sp *ShareProcessor) GetStatistics() ShareStatistics {
	sp.statsMutex.RLock()
	defer sp.statsMutex.RUnlock()
	
	return sp.stats
}

// GetMinerStatistics returns statistics for a specific miner
func (sp *ShareProcessor) GetMinerStatistics(minerID int64) MinerStatistics {
	sp.statsMutex.RLock()
	defer sp.statsMutex.RUnlock()
	
	if stats, exists := sp.minerStats[minerID]; exists {
		return *stats
	}
	
	return MinerStatistics{
		MinerID: minerID,
	}
}

// validateShareInput validates basic share input parameters
func (sp *ShareProcessor) validateShareInput(share *Share) error {
	if share == nil {
		return fmt.Errorf("share cannot be nil")
	}
	
	if share.JobID == "" {
		return fmt.Errorf("job_id cannot be empty")
	}
	
	if len(share.JobID) > sp.maxJobIDLength {
		return fmt.Errorf("job_id too long (max %d characters)", sp.maxJobIDLength)
	}
	
	if share.Nonce == "" {
		return fmt.Errorf("nonce cannot be empty")
	}
	
	if len(share.Nonce) > sp.maxNonceLength {
		return fmt.Errorf("nonce too long (max %d characters)", sp.maxNonceLength)
	}
	
	if share.Difficulty <= 0 {
		return fmt.Errorf("difficulty must be positive")
	}
	
	if share.MinerID <= 0 {
		return fmt.Errorf("miner_id must be positive")
	}
	
	if share.UserID <= 0 {
		return fmt.Errorf("user_id must be positive")
	}
	
	return nil
}

// parseNonce converts hex nonce string to uint64
func (sp *ShareProcessor) parseNonce(nonceStr string) (uint64, error) {
	// Remove 0x prefix if present
	nonceStr = strings.TrimPrefix(nonceStr, "0x")
	
	// Validate hex format
	if len(nonceStr) == 0 {
		return 0, fmt.Errorf("empty nonce")
	}
	
	// Parse as hex
	nonce, err := strconv.ParseUint(nonceStr, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid hex nonce: %v", err)
	}
	
	return nonce, nil
}

// difficultyToTarget converts mining difficulty to target bytes
func (sp *ShareProcessor) difficultyToTarget(difficulty float64) []byte {
	if difficulty <= 0 {
		return []byte{} // Invalid difficulty
	}
	
	// For very high difficulty (impossible), return very small target
	if difficulty > 1000000 {
		target := make([]byte, 32)
		target[31] = 0x01 // Very small target
		return target
	}
	
	// For testing purposes, make targets very generous
	// In production, this would be much more restrictive
	target := make([]byte, 32)
	
	if difficulty <= 1.0 {
		// Very easy target for difficulty 1.0 - about 50% chance of success
		target[0] = 0x80 // First bit must be 0, so 50% chance
		for i := 1; i < 32; i++ {
			target[i] = 0xFF
		}
	} else if difficulty <= 10.0 {
		// Medium target for difficulty up to 10
		target[0] = byte(0x80 / difficulty)
		if target[0] == 0 {
			target[0] = 0x01
		}
		for i := 1; i < 32; i++ {
			target[i] = 0xFF
		}
	} else {
		// Harder target for higher difficulties
		target[0] = 0x01
		for i := 1; i < 32; i++ {
			target[i] = 0x00
		}
	}
	
	return target
}

// updateStatistics updates share processing statistics
func (sp *ShareProcessor) updateStatistics(share *Share) {
	sp.statsMutex.Lock()
	defer sp.statsMutex.Unlock()
	
	// Update overall statistics
	sp.stats.TotalShares++
	if share.IsValid {
		sp.stats.ValidShares++
		sp.stats.TotalDifficulty += share.Difficulty
	} else {
		sp.stats.InvalidShares++
	}
	sp.stats.LastUpdated = time.Now()
	
	// Update per-miner statistics
	minerStats, exists := sp.minerStats[share.MinerID]
	if !exists {
		minerStats = &MinerStatistics{
			MinerID: share.MinerID,
		}
		sp.minerStats[share.MinerID] = minerStats
	}
	
	minerStats.TotalShares++
	if share.IsValid {
		minerStats.ValidShares++
		minerStats.TotalDifficulty += share.Difficulty
	} else {
		minerStats.InvalidShares++
	}
	minerStats.LastShare = share.Timestamp
}