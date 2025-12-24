package v2

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chimera-pool/chimera-pool-core/internal/stratum"
)

// =============================================================================
// TEMPLATE PROVIDER IMPLEMENTATION
// Generates and manages block templates for job negotiation
// =============================================================================

// Errors
var (
	ErrNoTemplateAvailable = errors.New("no template available")
	ErrRPCConnectionFailed = errors.New("RPC connection failed")
	ErrInvalidRPCResponse  = errors.New("invalid RPC response")
	ErrProviderStopped     = errors.New("template provider stopped")
)

// RPCClient interface for blockchain node communication
type RPCClient interface {
	// GetBlockTemplate requests a new block template from the node
	GetBlockTemplate() (*RPCBlockTemplate, error)

	// GetBlockchainInfo returns current blockchain information
	GetBlockchainInfo() (*RPCBlockchainInfo, error)

	// GetNetworkDifficulty returns current network difficulty
	GetNetworkDifficulty() (float64, error)

	// SubmitBlock submits a solved block to the network
	SubmitBlock(blockHex string) error
}

// RPCBlockTemplate represents the response from getblocktemplate RPC
type RPCBlockTemplate struct {
	Version           uint32  `json:"version"`
	PreviousBlockHash string  `json:"previousblockhash"`
	Transactions      []RPCTx `json:"transactions"`
	CoinbaseValue     uint64  `json:"coinbasevalue"`
	Target            string  `json:"target"`
	MinTime           uint32  `json:"mintime"`
	CurTime           uint32  `json:"curtime"`
	MaxTime           uint32  `json:"maxtime"`
	Height            uint64  `json:"height"`
	Bits              string  `json:"bits"`
	SigOpLimit        uint32  `json:"sigoplimit"`
	SizeLimit         uint32  `json:"sizelimit"`
	WeightLimit       uint32  `json:"weightlimit"`
}

// RPCTx represents a transaction in the block template
type RPCTx struct {
	Data    string `json:"data"`
	TxID    string `json:"txid"`
	Hash    string `json:"hash"`
	Depends []int  `json:"depends"`
	Fee     uint64 `json:"fee"`
	SigOps  int    `json:"sigops"`
	Weight  int    `json:"weight"`
}

// RPCBlockchainInfo represents blockchain info response
type RPCBlockchainInfo struct {
	Chain         string  `json:"chain"`
	Blocks        uint64  `json:"blocks"`
	Headers       uint64  `json:"headers"`
	BestBlockHash string  `json:"bestblockhash"`
	Difficulty    float64 `json:"difficulty"`
	MedianTime    uint64  `json:"mediantime"`
	SyncProgress  float64 `json:"verificationprogress"`
}

// =============================================================================
// Template Provider Implementation
// =============================================================================

// templateProvider implements stratum.TemplateProvider
type templateProvider struct {
	config     stratum.TemplateProviderConfig
	rpcClient  RPCClient
	coinConfig CoinConfig

	// Current template
	currentTemplate *stratum.BlockTemplate
	templateMu      sync.RWMutex

	// Subscribers
	subscribers   []func(*stratum.BlockTemplate)
	subscribersMu sync.RWMutex
	nextSubID     atomic.Int64

	// Lifecycle
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	isRunning atomic.Bool

	// Metrics
	templatesGenerated atomic.Int64
	lastTemplateTime   atomic.Int64
	rpcErrors          atomic.Int64
}

// CoinConfig holds coin-specific configuration
type CoinConfig struct {
	Name            string
	Symbol          string
	Algorithm       string
	BlockTime       time.Duration
	HalvingInterval uint64
	InitialReward   uint64
	AddressPrefix   []byte
}

// LitecoinConfig returns configuration for Litecoin
func LitecoinConfig() CoinConfig {
	return CoinConfig{
		Name:            "Litecoin",
		Symbol:          "LTC",
		Algorithm:       "scrypt",
		BlockTime:       time.Minute*2 + time.Second*30,
		HalvingInterval: 840000,
		InitialReward:   5000000000,   // 50 LTC in litoshis
		AddressPrefix:   []byte{0x30}, // L prefix
	}
}

// BlockDAGConfig returns configuration for BlockDAG
func BlockDAGConfig() CoinConfig {
	return CoinConfig{
		Name:            "BlockDAG",
		Symbol:          "BDAG",
		Algorithm:       "scrpy-variant",
		BlockTime:       time.Second * 10,
		HalvingInterval: 1000000,
		InitialReward:   10000000000,
		AddressPrefix:   []byte{0x1C},
	}
}

// NewTemplateProvider creates a new template provider
func NewTemplateProvider(config stratum.TemplateProviderConfig, rpcClient RPCClient, coinConfig CoinConfig) *templateProvider {
	ctx, cancel := context.WithCancel(context.Background())

	return &templateProvider{
		config:      config,
		rpcClient:   rpcClient,
		coinConfig:  coinConfig,
		subscribers: make([]func(*stratum.BlockTemplate), 0),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start starts the template provider
func (p *templateProvider) Start() error {
	if p.isRunning.Load() {
		return fmt.Errorf("template provider already running")
	}

	p.isRunning.Store(true)

	// Initial template fetch
	if err := p.updateTemplate(); err != nil {
		// Log but don't fail - will retry in background
		p.rpcErrors.Add(1)
	}

	// Start update loop
	p.wg.Add(1)
	go p.updateLoop()

	return nil
}

// Stop stops the template provider
func (p *templateProvider) Stop() error {
	if !p.isRunning.Load() {
		return nil
	}

	p.cancel()
	p.isRunning.Store(false)
	p.wg.Wait()

	return nil
}

// GetTemplate returns the current best block template
func (p *templateProvider) GetTemplate() (*stratum.BlockTemplate, error) {
	p.templateMu.RLock()
	defer p.templateMu.RUnlock()

	if p.currentTemplate == nil {
		return nil, ErrNoTemplateAvailable
	}

	// Return a copy to prevent modification
	return p.copyTemplate(p.currentTemplate), nil
}

// GetTemplateForHeight returns template for specific height
func (p *templateProvider) GetTemplateForHeight(height uint64) (*stratum.BlockTemplate, error) {
	// For now, just return current template if height matches
	p.templateMu.RLock()
	defer p.templateMu.RUnlock()

	if p.currentTemplate == nil {
		return nil, ErrNoTemplateAvailable
	}

	if p.currentTemplate.Height != height {
		return nil, fmt.Errorf("no template available for height %d", height)
	}

	return p.copyTemplate(p.currentTemplate), nil
}

// SubscribeTemplates subscribes to new template notifications
func (p *templateProvider) SubscribeTemplates(handler func(*stratum.BlockTemplate)) stratum.Subscription {
	p.subscribersMu.Lock()
	defer p.subscribersMu.Unlock()

	p.subscribers = append(p.subscribers, handler)
	subID := p.nextSubID.Add(1)

	return &templateSubscription{
		id:       subID,
		provider: p,
		active:   true,
	}
}

// GetCurrentHeight returns current blockchain height
func (p *templateProvider) GetCurrentHeight() (uint64, error) {
	if p.rpcClient == nil {
		p.templateMu.RLock()
		defer p.templateMu.RUnlock()
		if p.currentTemplate != nil {
			return p.currentTemplate.Height, nil
		}
		return 0, ErrNoTemplateAvailable
	}

	info, err := p.rpcClient.GetBlockchainInfo()
	if err != nil {
		return 0, err
	}

	return info.Blocks + 1, nil // Next block height
}

// GetNetworkDifficulty returns current network difficulty
func (p *templateProvider) GetNetworkDifficulty() (uint64, error) {
	if p.rpcClient == nil {
		return 1, nil
	}

	diff, err := p.rpcClient.GetNetworkDifficulty()
	if err != nil {
		return 0, err
	}

	return uint64(diff), nil
}

// =============================================================================
// Internal Methods
// =============================================================================

func (p *templateProvider) updateLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.UpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			if err := p.updateTemplate(); err != nil {
				p.rpcErrors.Add(1)
			}
		}
	}
}

func (p *templateProvider) updateTemplate() error {
	if p.rpcClient == nil {
		return ErrRPCConnectionFailed
	}

	// Fetch new template from RPC
	rpcTemplate, err := p.rpcClient.GetBlockTemplate()
	if err != nil {
		return fmt.Errorf("failed to get block template: %w", err)
	}

	// Convert to internal format
	template, err := p.convertRPCTemplate(rpcTemplate)
	if err != nil {
		return fmt.Errorf("failed to convert template: %w", err)
	}

	// Check if template changed
	p.templateMu.Lock()
	isNewTemplate := p.currentTemplate == nil ||
		p.currentTemplate.Height != template.Height ||
		!bytesEqual(p.currentTemplate.PrevHash, template.PrevHash)

	p.currentTemplate = template
	p.templateMu.Unlock()

	if isNewTemplate {
		p.templatesGenerated.Add(1)
		p.lastTemplateTime.Store(time.Now().UnixNano())
		p.notifySubscribers(template)
	}

	return nil
}

func (p *templateProvider) convertRPCTemplate(rpc *RPCBlockTemplate) (*stratum.BlockTemplate, error) {
	// Parse previous block hash
	prevHash, err := hex.DecodeString(rpc.PreviousBlockHash)
	if err != nil {
		return nil, fmt.Errorf("invalid previous hash: %w", err)
	}
	reverseBytes(prevHash)

	// Parse target
	target, err := hex.DecodeString(rpc.Target)
	if err != nil {
		return nil, fmt.Errorf("invalid target: %w", err)
	}

	// Parse bits
	bits, err := parseHexUint32(rpc.Bits)
	if err != nil {
		return nil, fmt.Errorf("invalid bits: %w", err)
	}

	// Build coinbase transaction
	coinbase := p.buildCoinbase(rpc.Height, rpc.CoinbaseValue)

	// Extract transaction data and hashes
	txs := make([][]byte, len(rpc.Transactions))
	txHashes := make([][]byte, len(rpc.Transactions)+1)
	txHashes[0] = nil // Placeholder for coinbase hash

	for i, tx := range rpc.Transactions {
		txData, err := hex.DecodeString(tx.Data)
		if err != nil {
			return nil, fmt.Errorf("invalid transaction %d: %w", i, err)
		}
		txs[i] = txData

		txHash, err := hex.DecodeString(tx.TxID)
		if err != nil {
			return nil, fmt.Errorf("invalid txid %d: %w", i, err)
		}
		reverseBytes(txHash)
		txHashes[i+1] = txHash
	}

	return &stratum.BlockTemplate{
		TemplateID:    generateTemplateID(),
		Version:       rpc.Version,
		PrevHash:      prevHash,
		Timestamp:     rpc.CurTime,
		Bits:          bits,
		Height:        rpc.Height,
		Coinbase:      coinbase,
		CoinbaseValue: rpc.CoinbaseValue,
		Transactions:  txs,
		TxHashes:      txHashes,
		Target:        target,
		Algorithm:     p.coinConfig.Algorithm,
		Coin:          p.coinConfig.Symbol,
		MinTime:       rpc.MinTime,
		MaxTime:       rpc.MaxTime,
		SigOpLimit:    rpc.SigOpLimit,
		SizeLimit:     rpc.SizeLimit,
		WeightLimit:   rpc.WeightLimit,
		CreatedAt:     time.Now(),
	}, nil
}

func (p *templateProvider) buildCoinbase(height uint64, value uint64) []byte {
	// Build BIP34 compliant coinbase transaction
	// This is a simplified version - production would be more complete

	coinbase := make([]byte, 0, 200)

	// Version (4 bytes, little-endian)
	coinbase = append(coinbase, 0x01, 0x00, 0x00, 0x00)

	// Input count (1 byte)
	coinbase = append(coinbase, 0x01)

	// Previous output hash (32 bytes of zeros for coinbase)
	coinbase = append(coinbase, make([]byte, 32)...)

	// Previous output index (4 bytes, all 1s for coinbase)
	coinbase = append(coinbase, 0xFF, 0xFF, 0xFF, 0xFF)

	// Script length (will be filled later)
	scriptStart := len(coinbase)
	coinbase = append(coinbase, 0x00) // Placeholder

	// BIP34: Height encoding
	heightBytes := encodeHeight(height)
	coinbase = append(coinbase, heightBytes...)

	// Pool signature prefix
	coinbase = append(coinbase, p.config.CoinbasePrefix...)

	// Extranonce space
	coinbase = append(coinbase, make([]byte, p.config.ExtraNonceSize)...)

	// Pool signature suffix
	coinbase = append(coinbase, p.config.CoinbaseSuffix...)

	// Update script length
	scriptLen := len(coinbase) - scriptStart - 1
	coinbase[scriptStart] = byte(scriptLen)

	// Sequence (4 bytes)
	coinbase = append(coinbase, 0xFF, 0xFF, 0xFF, 0xFF)

	// Output count (1 byte) - simplified to 1 output
	coinbase = append(coinbase, 0x01)

	// Output value (8 bytes, little-endian)
	coinbase = append(coinbase,
		byte(value),
		byte(value>>8),
		byte(value>>16),
		byte(value>>24),
		byte(value>>32),
		byte(value>>40),
		byte(value>>48),
		byte(value>>56),
	)

	// Output script (P2PKH placeholder)
	coinbase = append(coinbase, 0x19)                // Script length: 25
	coinbase = append(coinbase, 0x76, 0xA9, 0x14)    // OP_DUP OP_HASH160 PUSH20
	coinbase = append(coinbase, make([]byte, 20)...) // Placeholder for pubkey hash
	coinbase = append(coinbase, 0x88, 0xAC)          // OP_EQUALVERIFY OP_CHECKSIG

	// Locktime (4 bytes)
	coinbase = append(coinbase, 0x00, 0x00, 0x00, 0x00)

	return coinbase
}

func (p *templateProvider) notifySubscribers(template *stratum.BlockTemplate) {
	p.subscribersMu.RLock()
	defer p.subscribersMu.RUnlock()

	for _, handler := range p.subscribers {
		// Copy template for each subscriber
		go handler(p.copyTemplate(template))
	}
}

func (p *templateProvider) copyTemplate(t *stratum.BlockTemplate) *stratum.BlockTemplate {
	if t == nil {
		return nil
	}

	copy := &stratum.BlockTemplate{
		TemplateID:    t.TemplateID,
		Version:       t.Version,
		PrevHash:      copyBytes(t.PrevHash),
		MerkleRoot:    copyBytes(t.MerkleRoot),
		Timestamp:     t.Timestamp,
		Bits:          t.Bits,
		Height:        t.Height,
		Coinbase:      copyBytes(t.Coinbase),
		CoinbaseValue: t.CoinbaseValue,
		Target:        copyBytes(t.Target),
		Algorithm:     t.Algorithm,
		Coin:          t.Coin,
		MinTime:       t.MinTime,
		MaxTime:       t.MaxTime,
		SigOpLimit:    t.SigOpLimit,
		SizeLimit:     t.SizeLimit,
		WeightLimit:   t.WeightLimit,
		CreatedAt:     t.CreatedAt,
	}

	// Deep copy transactions
	if t.Transactions != nil {
		copy.Transactions = make([][]byte, len(t.Transactions))
		for i, tx := range t.Transactions {
			copy.Transactions[i] = copyBytes(tx)
		}
	}

	// Deep copy tx hashes
	if t.TxHashes != nil {
		copy.TxHashes = make([][]byte, len(t.TxHashes))
		for i, hash := range t.TxHashes {
			copy.TxHashes[i] = copyBytes(hash)
		}
	}

	return copy
}

// =============================================================================
// Template Subscription
// =============================================================================

type templateSubscription struct {
	id       int64
	provider *templateProvider
	active   bool
	mu       sync.Mutex
}

func (s *templateSubscription) Unsubscribe() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return
	}

	s.active = false
	// Note: In production, would remove from provider's subscriber list
}

func (s *templateSubscription) IsActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.active
}

// =============================================================================
// Utility Functions
// =============================================================================

func copyBytes(b []byte) []byte {
	if b == nil {
		return nil
	}
	c := make([]byte, len(b))
	copy(c, b)
	return c
}

func reverseBytes(b []byte) {
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func parseHexUint32(s string) (uint32, error) {
	var result uint32
	_, err := fmt.Sscanf(s, "%x", &result)
	return result, err
}

func encodeHeight(height uint64) []byte {
	// BIP34 height encoding
	if height < 17 {
		return []byte{byte(0x50 + height)}
	}

	// Calculate bytes needed
	bytes := make([]byte, 0, 9)
	h := height
	for h > 0 {
		bytes = append(bytes, byte(h&0xFF))
		h >>= 8
	}

	// Prepend length
	result := make([]byte, 1+len(bytes))
	result[0] = byte(len(bytes))
	copy(result[1:], bytes)

	return result
}

var templateCounter atomic.Int64

func generateTemplateID() string {
	counter := templateCounter.Add(1)
	return fmt.Sprintf("tpl_%d_%d", time.Now().Unix(), counter)
}

// =============================================================================
// Mock RPC Client for Testing
// =============================================================================

// MockRPCClient implements RPCClient for testing
type MockRPCClient struct {
	BlockTemplate  *RPCBlockTemplate
	BlockchainInfo *RPCBlockchainInfo
	Difficulty     float64
	Error          error
}

func (m *MockRPCClient) GetBlockTemplate() (*RPCBlockTemplate, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return m.BlockTemplate, nil
}

func (m *MockRPCClient) GetBlockchainInfo() (*RPCBlockchainInfo, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return m.BlockchainInfo, nil
}

func (m *MockRPCClient) GetNetworkDifficulty() (float64, error) {
	if m.Error != nil {
		return 0, m.Error
	}
	return m.Difficulty, nil
}

func (m *MockRPCClient) SubmitBlock(blockHex string) error {
	return m.Error
}

// =============================================================================
// Metrics
// =============================================================================

// GetStats returns provider statistics
func (p *templateProvider) GetStats() map[string]interface{} {
	lastTime := p.lastTemplateTime.Load()
	var lastTemplateAge time.Duration
	if lastTime > 0 {
		lastTemplateAge = time.Since(time.Unix(0, lastTime))
	}

	return map[string]interface{}{
		"templates_generated": p.templatesGenerated.Load(),
		"rpc_errors":          p.rpcErrors.Load(),
		"last_template_age":   lastTemplateAge.String(),
		"is_running":          p.isRunning.Load(),
		"subscriber_count":    len(p.subscribers),
	}
}
