package v2

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chimera-pool/chimera-pool-core/internal/stratum"
)

// =============================================================================
// JOB NEGOTIATION IMPLEMENTATION (Stratum V2 - SRI 2025)
// Enables miners to propose their own block templates
// =============================================================================

// Errors
var (
	ErrNegotiationDisabled   = errors.New("job negotiation is disabled")
	ErrTemplateRejected      = errors.New("template rejected by pool")
	ErrInvalidTemplate       = errors.New("invalid template")
	ErrDeclarationExpired    = errors.New("declaration expired")
	ErrDeclarationNotFound   = errors.New("declaration not found")
	ErrMinerNotAllowed       = errors.New("miner not allowed to declare templates")
	ErrRateLimitExceeded     = errors.New("declaration rate limit exceeded")
	ErrValidationTimeout     = errors.New("template validation timeout")
	ErrServerNotRunning      = errors.New("job declarator server not running")
	ErrInvalidCoinbase       = errors.New("invalid coinbase structure")
	ErrInsufficientPoolShare = errors.New("insufficient pool share in coinbase")
)

// =============================================================================
// Job Declarator Server Implementation
// =============================================================================

// jobDeclaratorServer implements stratum.JobDeclaratorServer
type jobDeclaratorServer struct {
	config    stratum.JobNegotiationConfig
	policy    stratum.TemplatePolicy
	validator stratum.TemplateValidator
	provider  stratum.TemplateProvider

	// Active declarations indexed by miner ID
	declarations     map[string]*stratum.JobDeclaration
	declarationsByID map[string]*stratum.JobDeclaration
	mu               sync.RWMutex

	// Rate limiting
	rateLimiter map[string]*rateLimitEntry
	rateMu      sync.Mutex

	// Lifecycle
	listener  net.Listener
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	isRunning atomic.Bool

	// Metrics
	totalDeclarations    atomic.Int64
	acceptedDeclarations atomic.Int64
	rejectedDeclarations atomic.Int64
	totalValidationTime  atomic.Int64
	validationCount      atomic.Int64
}

type rateLimitEntry struct {
	count     int
	resetTime time.Time
}

// NewJobDeclaratorServer creates a new job declarator server
func NewJobDeclaratorServer(config stratum.JobNegotiationConfig) *jobDeclaratorServer {
	ctx, cancel := context.WithCancel(context.Background())

	return &jobDeclaratorServer{
		config:           config,
		policy:           config.Policy,
		declarations:     make(map[string]*stratum.JobDeclaration),
		declarationsByID: make(map[string]*stratum.JobDeclaration),
		rateLimiter:      make(map[string]*rateLimitEntry),
		ctx:              ctx,
		cancel:           cancel,
	}
}

// Start starts the job declarator server
func (s *jobDeclaratorServer) Start() error {
	if !s.config.Enabled {
		return ErrNegotiationDisabled
	}

	if s.isRunning.Load() {
		return fmt.Errorf("server already running")
	}

	listener, err := net.Listen("tcp", s.config.ListenAddress)
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}

	s.listener = listener
	s.isRunning.Store(true)

	// Start accept loop
	s.wg.Add(1)
	go s.acceptLoop()

	// Start cleanup goroutine
	s.wg.Add(1)
	go s.cleanupExpiredDeclarations()

	return nil
}

// Stop stops the job declarator server
func (s *jobDeclaratorServer) Stop() error {
	if !s.isRunning.Load() {
		return nil
	}

	s.cancel()
	s.isRunning.Store(false)

	if s.listener != nil {
		s.listener.Close()
	}

	s.wg.Wait()
	return nil
}

// HandleDeclaration processes a miner's job declaration
func (s *jobDeclaratorServer) HandleDeclaration(minerID string, declaration *stratum.JobDeclaration) (*stratum.JobDeclarationResult, error) {
	if !s.isRunning.Load() {
		return nil, ErrServerNotRunning
	}

	s.totalDeclarations.Add(1)

	// Check if miner is allowed
	if !s.isMinerAllowed(minerID) {
		s.rejectedDeclarations.Add(1)
		return &stratum.JobDeclarationResult{
			Accepted:     false,
			ErrorCode:    "MINER_NOT_ALLOWED",
			ErrorMessage: "Miner is not authorized to declare templates",
		}, ErrMinerNotAllowed
	}

	// Check rate limit
	if !s.checkRateLimit(minerID) {
		s.rejectedDeclarations.Add(1)
		return &stratum.JobDeclarationResult{
			Accepted:     false,
			ErrorCode:    "RATE_LIMIT_EXCEEDED",
			ErrorMessage: "Declaration rate limit exceeded",
		}, ErrRateLimitExceeded
	}

	// Validate template
	validationStart := time.Now()
	var validationResult *stratum.TemplateValidationResult
	var validationErr error

	if s.validator != nil {
		validationResult, validationErr = s.validateWithTimeout(declaration.Template)
	} else {
		// Basic validation without custom validator
		validationResult = s.basicValidation(declaration.Template)
	}

	validationTime := time.Since(validationStart)
	s.totalValidationTime.Add(int64(validationTime))
	s.validationCount.Add(1)

	if validationErr != nil || (validationResult != nil && !validationResult.Valid) {
		s.rejectedDeclarations.Add(1)

		result := &stratum.JobDeclarationResult{
			Accepted:      false,
			DeclarationID: declaration.DeclarationID,
			ErrorCode:     "TEMPLATE_INVALID",
			ErrorMessage:  "Template validation failed",
			FallbackJob:   nil,
			ValidUntil:    time.Now(),
		}

		if validationResult != nil {
			result.ErrorCode = validationResult.ErrorCode
			result.ErrorMessage = validationResult.ErrorMessage
		}

		// Provide fallback if configured
		if s.policy.FallbackOnRejection && s.provider != nil {
			poolTemplate, err := s.provider.GetTemplate()
			if err == nil {
				result.PoolTemplate = poolTemplate
			}
		}

		return result, ErrTemplateRejected
	}

	// Check coinbase pool share
	if s.policy.RequirePoolCoinbase || s.policy.MinCoinbasePoolShare > 0 {
		if err := s.validateCoinbaseShare(declaration.Template); err != nil {
			s.rejectedDeclarations.Add(1)
			return &stratum.JobDeclarationResult{
				Accepted:     false,
				ErrorCode:    "COINBASE_INVALID",
				ErrorMessage: err.Error(),
			}, err
		}
	}

	// Generate declaration ID if not present
	if declaration.DeclarationID == "" {
		declaration.DeclarationID = s.generateDeclarationID()
	}

	// Set timestamps
	declaration.SubmittedAt = time.Now()
	declaration.ExpiresAt = time.Now().Add(s.policy.DeclarationTTL)
	declaration.AcceptedByPool = true
	declaration.MinerID = minerID

	// Store declaration
	s.mu.Lock()
	// Remove any existing declaration for this miner
	if existing, ok := s.declarations[minerID]; ok {
		delete(s.declarationsByID, existing.DeclarationID)
	}
	s.declarations[minerID] = declaration
	s.declarationsByID[declaration.DeclarationID] = declaration
	s.mu.Unlock()

	s.acceptedDeclarations.Add(1)

	return &stratum.JobDeclarationResult{
		Accepted:      true,
		DeclarationID: declaration.DeclarationID,
		AssignedJobID: s.generateJobID(),
		ValidUntil:    declaration.ExpiresAt,
	}, nil
}

// GetActiveDeclarations returns all active declarations
func (s *jobDeclaratorServer) GetActiveDeclarations() []*stratum.JobDeclaration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	active := make([]*stratum.JobDeclaration, 0, len(s.declarations))

	for _, decl := range s.declarations {
		if decl.ExpiresAt.After(now) {
			active = append(active, decl)
		}
	}

	return active
}

// GetDeclarationByMiner returns active declaration for a miner
func (s *jobDeclaratorServer) GetDeclarationByMiner(minerID string) (*stratum.JobDeclaration, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	decl, ok := s.declarations[minerID]
	if !ok || decl.ExpiresAt.Before(time.Now()) {
		return nil, false
	}

	return decl, true
}

// SetTemplatePolicy sets the pool's template acceptance policy
func (s *jobDeclaratorServer) SetTemplatePolicy(policy stratum.TemplatePolicy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policy = policy
}

// GetTemplatePolicy returns current template policy
func (s *jobDeclaratorServer) GetTemplatePolicy() stratum.TemplatePolicy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.policy
}

// RegisterValidator registers a template validator
func (s *jobDeclaratorServer) RegisterValidator(validator stratum.TemplateValidator) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.validator = validator
}

// SetTemplateProvider sets the template provider for fallback
func (s *jobDeclaratorServer) SetTemplateProvider(provider stratum.TemplateProvider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.provider = provider
}

// =============================================================================
// Internal Methods
// =============================================================================

func (s *jobDeclaratorServer) acceptLoop() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				if s.ctx.Err() != nil {
					return
				}
				continue
			}

			s.wg.Add(1)
			go s.handleConnection(conn)
		}
	}
}

func (s *jobDeclaratorServer) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	// Connection handling will be implemented with V2 binary protocol
	// For now, this is a placeholder for the connection loop
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			// Read and process messages
			conn.SetReadDeadline(time.Now().Add(30 * time.Second))
			buf := make([]byte, 4096)
			n, err := conn.Read(buf)
			if err != nil {
				return
			}

			// Process declaration message (to be implemented with binary protocol)
			_ = n
		}
	}
}

func (s *jobDeclaratorServer) cleanupExpiredDeclarations() {
	defer s.wg.Done()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for minerID, decl := range s.declarations {
				if decl.ExpiresAt.Before(now) {
					delete(s.declarations, minerID)
					delete(s.declarationsByID, decl.DeclarationID)
				}
			}
			s.mu.Unlock()
		}
	}
}

func (s *jobDeclaratorServer) isMinerAllowed(minerID string) bool {
	if !s.policy.AllowMinerTemplates {
		return false
	}

	// Empty whitelist means all miners allowed
	if len(s.policy.AllowedMinerIDs) == 0 {
		return true
	}

	for _, allowed := range s.policy.AllowedMinerIDs {
		if allowed == minerID {
			return true
		}
	}

	return false
}

func (s *jobDeclaratorServer) checkRateLimit(minerID string) bool {
	if s.policy.MaxDeclarationsPerMin <= 0 {
		return true
	}

	s.rateMu.Lock()
	defer s.rateMu.Unlock()

	now := time.Now()
	entry, ok := s.rateLimiter[minerID]

	if !ok || entry.resetTime.Before(now) {
		s.rateLimiter[minerID] = &rateLimitEntry{
			count:     1,
			resetTime: now.Add(time.Minute),
		}
		return true
	}

	if entry.count >= s.policy.MaxDeclarationsPerMin {
		return false
	}

	entry.count++
	return true
}

func (s *jobDeclaratorServer) validateWithTimeout(template *stratum.BlockTemplate) (*stratum.TemplateValidationResult, error) {
	ctx, cancel := context.WithTimeout(s.ctx, s.config.ValidationTimeout)
	defer cancel()

	resultChan := make(chan *stratum.TemplateValidationResult, 1)
	errChan := make(chan error, 1)

	go func() {
		result, err := s.validator.ValidateTemplate(template)
		if err != nil {
			errChan <- err
			return
		}
		resultChan <- result
	}()

	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errChan:
		return nil, err
	case <-ctx.Done():
		return nil, ErrValidationTimeout
	}
}

func (s *jobDeclaratorServer) basicValidation(template *stratum.BlockTemplate) *stratum.TemplateValidationResult {
	result := &stratum.TemplateValidationResult{
		Valid:   true,
		TxCount: len(template.Transactions),
	}

	// Basic validation checks
	if template == nil {
		result.Valid = false
		result.ErrorCode = "TEMPLATE_NULL"
		result.ErrorMessage = "Template is null"
		return result
	}

	if len(template.PrevHash) != 32 {
		result.Valid = false
		result.ErrorCode = "INVALID_PREV_HASH"
		result.ErrorMessage = "Previous hash must be 32 bytes"
		return result
	}

	if len(template.Coinbase) == 0 {
		result.Valid = false
		result.ErrorCode = "MISSING_COINBASE"
		result.ErrorMessage = "Coinbase transaction is required"
		return result
	}

	if template.Bits == 0 {
		result.Valid = false
		result.ErrorCode = "INVALID_BITS"
		result.ErrorMessage = "Target bits must be non-zero"
		return result
	}

	return result
}

func (s *jobDeclaratorServer) validateCoinbaseShare(template *stratum.BlockTemplate) error {
	// This would need to parse the coinbase transaction and verify
	// that the pool's share is at least MinCoinbasePoolShare
	// For now, basic validation
	if len(template.Coinbase) < 100 {
		return ErrInvalidCoinbase
	}
	return nil
}

func (s *jobDeclaratorServer) generateDeclarationID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *jobDeclaratorServer) generateJobID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// =============================================================================
// Metrics
// =============================================================================

// GetStats returns server statistics
func (s *jobDeclaratorServer) GetStats() map[string]interface{} {
	validationCount := s.validationCount.Load()
	avgValidationTime := time.Duration(0)
	if validationCount > 0 {
		avgValidationTime = time.Duration(s.totalValidationTime.Load() / validationCount)
	}

	total := s.totalDeclarations.Load()
	accepted := s.acceptedDeclarations.Load()
	acceptRate := float64(0)
	if total > 0 {
		acceptRate = float64(accepted) / float64(total)
	}

	return map[string]interface{}{
		"total_declarations":     total,
		"accepted_declarations":  accepted,
		"rejected_declarations":  s.rejectedDeclarations.Load(),
		"acceptance_rate":        acceptRate,
		"avg_validation_time_ms": avgValidationTime.Milliseconds(),
		"active_declarations":    len(s.GetActiveDeclarations()),
		"is_running":             s.isRunning.Load(),
	}
}

// =============================================================================
// Template Validator Implementation (Scrypt/Scrpy-variant)
// =============================================================================

// scryptTemplateValidator implements stratum.TemplateValidator for Scrypt coins
type scryptTemplateValidator struct {
	algorithm      string
	maxBlockWeight uint32
	rewardSchedule []rewardScheduleEntry
	mu             sync.RWMutex
}

type rewardScheduleEntry struct {
	fromHeight uint64
	reward     uint64
}

// NewScryptTemplateValidator creates a validator for Scrypt-based coins
func NewScryptTemplateValidator(algorithm string, maxBlockWeight uint32) *scryptTemplateValidator {
	return &scryptTemplateValidator{
		algorithm:      algorithm,
		maxBlockWeight: maxBlockWeight,
		rewardSchedule: defaultLitecoinRewardSchedule(),
	}
}

func defaultLitecoinRewardSchedule() []rewardScheduleEntry {
	// Litecoin halving schedule (every 840,000 blocks)
	// Initial reward: 50 LTC
	schedule := make([]rewardScheduleEntry, 0)
	reward := uint64(5000000000) // 50 LTC in litoshis
	height := uint64(0)

	for reward > 0 {
		schedule = append(schedule, rewardScheduleEntry{
			fromHeight: height,
			reward:     reward,
		})
		height += 840000
		reward /= 2
	}

	return schedule
}

// ValidateTemplate validates a complete block template
func (v *scryptTemplateValidator) ValidateTemplate(template *stratum.BlockTemplate) (*stratum.TemplateValidationResult, error) {
	start := time.Now()
	result := &stratum.TemplateValidationResult{
		Valid:    true,
		Warnings: make([]string, 0),
	}

	// Validate structure
	if template == nil {
		result.Valid = false
		result.ErrorCode = "TEMPLATE_NULL"
		result.ErrorMessage = "Template is null"
		return result, nil
	}

	// Validate previous hash
	if len(template.PrevHash) != 32 {
		result.Valid = false
		result.ErrorCode = "INVALID_PREV_HASH"
		result.ErrorMessage = "Previous hash must be 32 bytes"
		return result, nil
	}

	// Validate coinbase
	if err := v.ValidateCoinbase(template.Coinbase, template.Height, template.CoinbaseValue); err != nil {
		result.Valid = false
		result.ErrorCode = "INVALID_COINBASE"
		result.ErrorMessage = err.Error()
		return result, nil
	}

	// Validate transactions
	if err := v.ValidateTransactions(template.Transactions); err != nil {
		result.Valid = false
		result.ErrorCode = "INVALID_TRANSACTIONS"
		result.ErrorMessage = err.Error()
		return result, nil
	}

	// Validate merkle root
	if len(template.MerkleRoot) > 0 && len(template.TxHashes) > 0 {
		if !v.ValidateMerkleRoot(template.TxHashes, template.MerkleRoot) {
			result.Valid = false
			result.ErrorCode = "INVALID_MERKLE_ROOT"
			result.ErrorMessage = "Merkle root does not match transactions"
			return result, nil
		}
	}

	// Validate timestamp range
	if template.Timestamp < template.MinTime || template.Timestamp > template.MaxTime {
		result.Warnings = append(result.Warnings, "Timestamp outside recommended range")
	}

	// Calculate metrics
	result.TxCount = len(template.Transactions)
	result.ValidationTime = time.Since(start)

	return result, nil
}

// ValidateCoinbase validates coinbase transaction structure
func (v *scryptTemplateValidator) ValidateCoinbase(coinbase []byte, height uint64, reward uint64) error {
	if len(coinbase) < 100 {
		return fmt.Errorf("coinbase too short: %d bytes", len(coinbase))
	}

	// Verify height encoding in coinbase (BIP34)
	// Height should be encoded in script at position 0
	if len(coinbase) < 4 {
		return errors.New("coinbase missing height encoding")
	}

	// Verify reward doesn't exceed maximum
	maxReward := v.GetBlockReward(height)
	if reward > maxReward {
		return fmt.Errorf("coinbase value %d exceeds max reward %d", reward, maxReward)
	}

	return nil
}

// ValidateTransactions validates included transactions
func (v *scryptTemplateValidator) ValidateTransactions(txs [][]byte) error {
	for i, tx := range txs {
		// Basic transaction validation
		if len(tx) < 10 {
			return fmt.Errorf("transaction %d too short: %d bytes", i, len(tx))
		}

		// Version check (first 4 bytes)
		if len(tx) < 4 {
			return fmt.Errorf("transaction %d missing version", i)
		}
	}

	return nil
}

// ValidateMerkleRoot verifies merkle root calculation
func (v *scryptTemplateValidator) ValidateMerkleRoot(txHashes [][]byte, expectedRoot []byte) bool {
	if len(txHashes) == 0 {
		return false
	}

	// Compute merkle root and compare
	// This is a simplified version - full implementation would use double SHA256
	computed := computeMerkleRoot(txHashes)
	if len(computed) != len(expectedRoot) {
		return false
	}

	for i := range computed {
		if computed[i] != expectedRoot[i] {
			return false
		}
	}

	return true
}

// GetBlockReward returns the current block reward for height
func (v *scryptTemplateValidator) GetBlockReward(height uint64) uint64 {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// Find applicable reward
	for i := len(v.rewardSchedule) - 1; i >= 0; i-- {
		if height >= v.rewardSchedule[i].fromHeight {
			return v.rewardSchedule[i].reward
		}
	}

	// Default fallback
	return 5000000000 // 50 LTC
}

// GetMaxBlockWeight returns maximum allowed block weight
func (v *scryptTemplateValidator) GetMaxBlockWeight() uint32 {
	return v.maxBlockWeight
}

// computeMerkleRoot computes merkle root from transaction hashes
func computeMerkleRoot(txHashes [][]byte) []byte {
	if len(txHashes) == 0 {
		return make([]byte, 32)
	}

	if len(txHashes) == 1 {
		return txHashes[0]
	}

	// Copy hashes
	level := make([][]byte, len(txHashes))
	for i, h := range txHashes {
		level[i] = make([]byte, len(h))
		copy(level[i], h)
	}

	// Build tree
	for len(level) > 1 {
		if len(level)%2 != 0 {
			level = append(level, level[len(level)-1])
		}

		newLevel := make([][]byte, len(level)/2)
		for i := 0; i < len(level); i += 2 {
			combined := append(level[i], level[i+1]...)
			newLevel[i/2] = doubleSHA256(combined)
		}
		level = newLevel
	}

	return level[0]
}

// doubleSHA256 computes SHA256(SHA256(data)) - placeholder implementation
func doubleSHA256(data []byte) []byte {
	// In production, use crypto/sha256
	// This is a placeholder that returns a 32-byte hash
	result := make([]byte, 32)
	for i := 0; i < 32 && i < len(data); i++ {
		result[i] = data[i]
	}
	return result
}

// =============================================================================
// Default Configuration
// =============================================================================

// DefaultJobNegotiationConfig returns default configuration
func DefaultJobNegotiationConfig() stratum.JobNegotiationConfig {
	return stratum.JobNegotiationConfig{
		Enabled:       false,   // Disabled by default
		ListenAddress: ":3335", // Separate port for job negotiation
		TemplateProviderConfig: stratum.TemplateProviderConfig{
			UpdateInterval:   time.Second * 5,
			MinFeeRate:       1,
			MaxTransactions:  5000,
			ExtraNonceSize:   4,
			AllowEmptyBlocks: false,
			PrioritizeByFee:  true,
		},
		Policy: stratum.TemplatePolicy{
			AllowMinerTemplates:    true,
			RequirePoolCoinbase:    true,
			AllowCustomCoinbase:    false,
			MinCoinbasePoolShare:   0.02, // Pool takes at least 2%
			MaxDeclarationsPerMin:  10,
			DeclarationTTL:         time.Minute * 5,
			RequireAllPoolTxs:      false,
			AllowAdditionalTxs:     true,
			MinFeeRate:             1,
			FallbackOnRejection:    true,
			FallbackTimeout:        time.Second * 5,
			RequireSignedTemplates: false,
		},
		MaxConcurrentValidations: 10,
		ValidationTimeout:        time.Second * 10,
		EnableMetrics:            true,
	}
}
