package stratum

import (
	"net"
	"time"
)

// =============================================================================
// INTERFACE SEGREGATION PRINCIPLE (ISP) COMPLIANT INTERFACES
// Each interface is small and focused on a single responsibility
// =============================================================================

// -----------------------------------------------------------------------------
// Core Connection Interfaces
// -----------------------------------------------------------------------------

// StratumConnection represents a minimal connection interface
type StratumConnection interface {
	ID() string
	RemoteAddr() string
	Close() error
}

// MessageReader handles reading messages from a connection
type MessageReader interface {
	ReadMessage() (Message, error)
	SetReadDeadline(t time.Time) error
}

// MessageWriter handles writing messages to a connection
type MessageWriter interface {
	WriteMessage(msg Message) error
	SetWriteDeadline(t time.Time) error
}

// DuplexConnection combines read and write capabilities
type DuplexConnection interface {
	StratumConnection
	MessageReader
	MessageWriter
}

// -----------------------------------------------------------------------------
// Message Interfaces
// -----------------------------------------------------------------------------

// Message represents a protocol-agnostic message
type Message interface {
	Type() MessageType
	ID() uint64
	Payload() []byte
}

// MessageType defines the type of stratum message
type MessageType int

const (
	MessageTypeUnknown MessageType = iota
	// V1 Message Types
	MessageTypeSubscribe
	MessageTypeAuthorize
	MessageTypeSubmit
	MessageTypeNotify
	MessageTypeSetDifficulty
	MessageTypeSetExtranonce
	// V2 Message Types
	MessageTypeSetupConnection
	MessageTypeSetupConnectionSuccess
	MessageTypeSetupConnectionError
	MessageTypeOpenChannel
	MessageTypeOpenChannelSuccess
	MessageTypeOpenChannelError
	MessageTypeNewMiningJob
	MessageTypeSetNewPrevHash
	MessageTypeSubmitShares
	MessageTypeSubmitSharesSuccess
	MessageTypeSubmitSharesError
	MessageTypeSetTarget
	MessageTypeReconnect
)

// -----------------------------------------------------------------------------
// Share Processing Interfaces
// -----------------------------------------------------------------------------

// Share represents a submitted mining share
type Share struct {
	MinerID      string
	WorkerName   string
	JobID        string
	Nonce        uint64
	Extranonce2  string
	NTime        uint32
	Difficulty   uint64
	IsValid      bool
	Timestamp    time.Time
	HardwareType HardwareClass
}

// ShareSubmitter handles share submission (protocol agnostic)
type ShareSubmitter interface {
	SubmitShare(share Share) (accepted bool, err error)
}

// ShareValidator validates shares against target difficulty
type ShareValidator interface {
	ValidateShare(share Share, target []byte) (bool, error)
}

// -----------------------------------------------------------------------------
// Job Distribution Interfaces
// -----------------------------------------------------------------------------

// Job represents a mining job
type Job struct {
	ID           string
	PrevHash     []byte
	Coinbase1    []byte
	Coinbase2    []byte
	MerkleBranch [][]byte
	Version      uint32
	NBits        uint32
	NTime        uint32
	CleanJobs    bool
	Target       []byte
	Algorithm    string
	Coin         string
	Height       uint64
	CreatedAt    time.Time
}

// JobHandler is a callback for new jobs
type JobHandler func(job Job)

// Subscription represents a job subscription
type Subscription interface {
	Unsubscribe()
	IsActive() bool
}

// JobDistributor distributes mining jobs to miners
type JobDistributor interface {
	GetCurrentJob() Job
	SubscribeToJobs(handler JobHandler) Subscription
	BroadcastJob(job Job) error
}

// -----------------------------------------------------------------------------
// Difficulty Management Interfaces
// -----------------------------------------------------------------------------

// DifficultyManager manages per-miner difficulty
type DifficultyManager interface {
	GetDifficulty(minerID string) uint64
	SetDifficulty(minerID string, difficulty uint64) error
}

// VardiffManager handles variable difficulty adjustments
type VardiffManager interface {
	DifficultyManager
	AdjustDifficulty(minerID string, shareTime time.Duration) uint64
	GetTargetShareTime() time.Duration
}

// -----------------------------------------------------------------------------
// Hardware Classification
// -----------------------------------------------------------------------------

// HardwareClass represents the type of mining hardware
type HardwareClass int

const (
	HardwareClassUnknown HardwareClass = iota
	HardwareClassCPU
	HardwareClassGPU
	HardwareClassFPGA
	HardwareClassASIC
	HardwareClassOfficialASIC // BlockDAG X30/X100
)

// String returns the hardware class name
func (h HardwareClass) String() string {
	switch h {
	case HardwareClassCPU:
		return "cpu"
	case HardwareClassGPU:
		return "gpu"
	case HardwareClassFPGA:
		return "fpga"
	case HardwareClassASIC:
		return "asic"
	case HardwareClassOfficialASIC:
		return "official_asic"
	default:
		return "unknown"
	}
}

// BaseDifficulty returns recommended starting difficulty for hardware class
func (h HardwareClass) BaseDifficulty() uint64 {
	switch h {
	case HardwareClassCPU:
		return 32 // ~100 KH/s target
	case HardwareClassGPU:
		return 4096 // ~10 MH/s target
	case HardwareClassFPGA:
		return 16384 // ~50 MH/s target
	case HardwareClassASIC:
		return 32768 // ~80 MH/s target (X30)
	case HardwareClassOfficialASIC:
		return 65536 // ~240 MH/s target (X100)
	default:
		return 256
	}
}

// HardwareClassifier determines hardware class from connection data
type HardwareClassifier interface {
	ClassifyHardware(userAgent string, hashrate float64) HardwareClass
}

// -----------------------------------------------------------------------------
// Miner Session Interfaces
// -----------------------------------------------------------------------------

// MinerSession represents an authenticated miner session
type MinerSession interface {
	ID() string
	Authorize(worker, password string) error
	IsAuthorized() bool
	GetWorkerName() string
	GetHardwareClass() HardwareClass
	GetDifficulty() uint64
	GetHashrate() float64
	GetShareCount() uint64
	GetLastShareTime() time.Time
}

// SessionManager manages miner sessions
type SessionManager interface {
	CreateSession(conn StratumConnection) MinerSession
	GetSession(id string) (MinerSession, bool)
	RemoveSession(id string)
	GetActiveSessions() []MinerSession
	GetSessionCount() int
}

// -----------------------------------------------------------------------------
// Protocol Handler Interfaces
// -----------------------------------------------------------------------------

// ProtocolVersion identifies the stratum protocol version
type ProtocolVersion int

const (
	ProtocolV1 ProtocolVersion = 1
	ProtocolV2 ProtocolVersion = 2
)

// ProtocolHandler handles connections for a specific protocol version
type ProtocolHandler interface {
	HandleConnection(conn net.Conn) error
	Protocol() ProtocolVersion
	Shutdown() error
}

// ProtocolDetector detects which protocol a connection is using
type ProtocolDetector interface {
	DetectProtocol(conn net.Conn) (ProtocolVersion, error)
}

// ProtocolRouter routes connections to appropriate handlers
type ProtocolRouter interface {
	RegisterHandler(version ProtocolVersion, handler ProtocolHandler)
	RouteConnection(conn net.Conn) error
}

// -----------------------------------------------------------------------------
// Encryption Interfaces (V2 specific)
// -----------------------------------------------------------------------------

// Encryptor handles message encryption
type Encryptor interface {
	Encrypt(plaintext []byte) ([]byte, error)
}

// Decryptor handles message decryption
type Decryptor interface {
	Decrypt(ciphertext []byte) ([]byte, error)
}

// SecureChannel combines encryption and decryption
type SecureChannel interface {
	Encryptor
	Decryptor
	IsEstablished() bool
}

// HandshakeHandler performs protocol handshake
type HandshakeHandler interface {
	PerformHandshake(conn net.Conn) (SecureChannel, error)
}

// -----------------------------------------------------------------------------
// Algorithm Interfaces
// -----------------------------------------------------------------------------

// HashAlgorithm represents a mining hash algorithm
type HashAlgorithm interface {
	Name() string
	Hash(data []byte) []byte
	ValidateHash(hash, target []byte) bool
}

// BlockTemplateGenerator generates block templates for mining
type BlockTemplateGenerator interface {
	GenerateTemplate(coin string, height uint64) (*Job, error)
	GetCurrentHeight(coin string) (uint64, error)
}

// -----------------------------------------------------------------------------
// Metrics Interfaces
// -----------------------------------------------------------------------------

// MetricsCollector collects pool metrics
type MetricsCollector interface {
	RecordShare(share Share)
	RecordConnection(protocol ProtocolVersion)
	RecordDisconnection(protocol ProtocolVersion)
	GetPoolHashrate() float64
	GetActiveMiners() int
}

// -----------------------------------------------------------------------------
// Keepalive Interfaces
// -----------------------------------------------------------------------------

// KeepaliveConfig holds keepalive configuration
type KeepaliveConfig struct {
	Interval        time.Duration // How often to send keepalive
	Timeout         time.Duration // How long to wait for response
	MaxMissed       int           // Max missed keepalives before disconnect
	SendWorkAsAlive bool          // Use work updates as keepalive
}

// KeepaliveManager handles connection keepalive
type KeepaliveManager interface {
	Start(minerID string)
	Stop(minerID string)
	RecordActivity(minerID string) // Called when miner sends any message
	IsAlive(minerID string) bool
	GetConfig() KeepaliveConfig
}

// -----------------------------------------------------------------------------
// Work Broadcaster Interfaces
// -----------------------------------------------------------------------------

// WorkBroadcasterConfig holds work broadcaster configuration
type WorkBroadcasterConfig struct {
	MinInterval   time.Duration // Minimum time between work updates
	MaxInterval   time.Duration // Maximum time without work update (forces update)
	OnBlockChange bool          // Broadcast immediately on new block
	OnDiffChange  bool          // Broadcast when difficulty changes
}

// WorkBroadcaster manages work distribution to miners
type WorkBroadcaster interface {
	Start() error
	Stop() error
	BroadcastToMiner(minerID string, job Job) error
	BroadcastToAll(job Job) error
	SetInterval(interval time.Duration)
	GetConfig() WorkBroadcasterConfig
}

// -----------------------------------------------------------------------------
// Merkle Tree Interfaces
// -----------------------------------------------------------------------------

// MerkleTreeBuilder builds merkle trees for block templates
type MerkleTreeBuilder interface {
	// BuildBranch computes the merkle branch for coinbase at index 0
	BuildBranch(txHashes [][]byte) [][]byte
	// ComputeRoot computes the merkle root given coinbase and branch
	ComputeRoot(coinbaseHash []byte, branch [][]byte) []byte
}

// =============================================================================
// JOB NEGOTIATION INTERFACES (Stratum V2 - SRI 2025)
// Enables miners to propose their own block templates
// =============================================================================

// -----------------------------------------------------------------------------
// Block Template Interfaces
// -----------------------------------------------------------------------------

// BlockTemplate represents a complete block template for mining
type BlockTemplate struct {
	TemplateID     string    // Unique template identifier
	Version        uint32    // Block version
	PrevHash       []byte    // Previous block hash (32 bytes)
	MerkleRoot     []byte    // Merkle root of transactions (32 bytes)
	Timestamp      uint32    // Block timestamp
	Bits           uint32    // Compact target (nBits)
	Height         uint64    // Block height
	Coinbase       []byte    // Coinbase transaction
	CoinbaseValue  uint64    // Total coinbase value (block reward + fees)
	Transactions   [][]byte  // Transaction data
	TxHashes       [][]byte  // Transaction hashes for merkle tree
	Target         []byte    // Full 256-bit target
	Algorithm      string    // Mining algorithm (e.g., "scrpy-variant", "scrypt")
	Coin           string    // Coin identifier (e.g., "LTC", "BLOCKDAG")
	MinTime        uint32    // Minimum valid timestamp
	MaxTime        uint32    // Maximum valid timestamp
	SigOpLimit     uint32    // Signature operation limit
	SizeLimit      uint32    // Block size limit
	WeightLimit    uint32    // Block weight limit (for SegWit)
	MinerSignature []byte    // Optional: miner's signature on template
	CreatedAt      time.Time // Template creation time
}

// TemplateValidationResult contains the result of template validation
type TemplateValidationResult struct {
	Valid          bool     // Whether template is valid
	ErrorCode      string   // Error code if invalid
	ErrorMessage   string   // Human-readable error message
	Warnings       []string // Non-fatal warnings
	EstimatedFees  uint64   // Estimated total fees
	TxCount        int      // Number of transactions
	BlockWeight    uint32   // Total block weight
	ValidationTime time.Duration
}

// TemplateValidator validates miner-proposed block templates
type TemplateValidator interface {
	// ValidateTemplate validates a complete block template
	ValidateTemplate(template *BlockTemplate) (*TemplateValidationResult, error)

	// ValidateCoinbase validates coinbase transaction structure
	ValidateCoinbase(coinbase []byte, height uint64, reward uint64) error

	// ValidateTransactions validates included transactions
	ValidateTransactions(txs [][]byte) error

	// ValidateMerkleRoot verifies merkle root calculation
	ValidateMerkleRoot(txHashes [][]byte, expectedRoot []byte) bool

	// GetBlockReward returns the current block reward for height
	GetBlockReward(height uint64) uint64

	// GetMaxBlockWeight returns maximum allowed block weight
	GetMaxBlockWeight() uint32
}

// -----------------------------------------------------------------------------
// Template Provider Interfaces
// -----------------------------------------------------------------------------

// TemplateProvider generates block templates (pool-side)
type TemplateProvider interface {
	// GetTemplate returns the current best block template
	GetTemplate() (*BlockTemplate, error)

	// GetTemplateForHeight returns template for specific height
	GetTemplateForHeight(height uint64) (*BlockTemplate, error)

	// SubscribeTemplates subscribes to new template notifications
	SubscribeTemplates(handler func(*BlockTemplate)) Subscription

	// GetCurrentHeight returns current blockchain height
	GetCurrentHeight() (uint64, error)

	// GetNetworkDifficulty returns current network difficulty
	GetNetworkDifficulty() (uint64, error)
}

// TemplateProviderConfig holds template provider configuration
type TemplateProviderConfig struct {
	UpdateInterval   time.Duration // How often to poll for new templates
	MinFeeRate       uint64        // Minimum fee rate (satoshis/vbyte)
	MaxTransactions  int           // Maximum transactions per template
	CoinbasePrefix   []byte        // Pool's coinbase signature prefix
	CoinbaseSuffix   []byte        // Pool's coinbase signature suffix
	ExtraNonceSize   int           // Size of extranonce space
	AllowEmptyBlocks bool          // Allow templates with no transactions
	PrioritizeByFee  bool          // Order transactions by fee rate
}

// -----------------------------------------------------------------------------
// Job Declarator Interfaces (SRI Core Component)
// -----------------------------------------------------------------------------

// JobDeclaration represents a miner's proposed job/template
type JobDeclaration struct {
	DeclarationID   string         // Unique declaration identifier
	MinerID         string         // Miner submitting the declaration
	Template        *BlockTemplate // Proposed block template
	Priority        int            // Declaration priority (higher = preferred)
	Signature       []byte         // Miner's signature on declaration
	SubmittedAt     time.Time      // Submission timestamp
	ExpiresAt       time.Time      // Declaration expiration
	AcceptedByPool  bool           // Whether pool accepted this declaration
	RejectionReason string         // Reason if rejected
}

// JobDeclarationResult contains the result of a job declaration
type JobDeclarationResult struct {
	Accepted      bool           // Whether declaration was accepted
	DeclarationID string         // ID of the declaration
	AssignedJobID string         // Pool-assigned job ID if accepted
	ErrorCode     string         // Error code if rejected
	ErrorMessage  string         // Human-readable error if rejected
	FallbackJob   *Job           // Pool's fallback job if miner template rejected
	PoolTemplate  *BlockTemplate // Pool's template if miner should use it
	ValidUntil    time.Time      // How long this declaration is valid
}

// JobDeclaratorClient is the miner-side interface for job negotiation
type JobDeclaratorClient interface {
	// Connect connects to the Job Declarator Server
	Connect(endpoint string) error

	// DeclareTemplate submits a template proposal to the pool
	DeclareTemplate(template *BlockTemplate) (*JobDeclarationResult, error)

	// RequestPoolTemplate requests the pool's current template
	RequestPoolTemplate() (*BlockTemplate, error)

	// SubscribeToJobs subscribes to job updates (pool or own template)
	SubscribeToJobs(handler func(*Job, bool)) Subscription // bool = isOwnTemplate

	// GetActiveDeclaration returns currently active declaration
	GetActiveDeclaration() *JobDeclaration

	// RevokeDeclaration revokes a previously submitted declaration
	RevokeDeclaration(declarationID string) error

	// Close closes the connection
	Close() error
}

// JobDeclaratorServer is the pool-side interface for job negotiation
type JobDeclaratorServer interface {
	// Start starts the job declarator server
	Start() error

	// Stop stops the job declarator server
	Stop() error

	// HandleDeclaration processes a miner's job declaration
	HandleDeclaration(minerID string, declaration *JobDeclaration) (*JobDeclarationResult, error)

	// GetActiveDeclarations returns all active declarations
	GetActiveDeclarations() []*JobDeclaration

	// GetDeclarationByMiner returns active declaration for a miner
	GetDeclarationByMiner(minerID string) (*JobDeclaration, bool)

	// SetTemplatePolicy sets the pool's template acceptance policy
	SetTemplatePolicy(policy TemplatePolicy)

	// GetTemplatePolicy returns current template policy
	GetTemplatePolicy() TemplatePolicy

	// RegisterValidator registers a template validator
	RegisterValidator(validator TemplateValidator)
}

// TemplatePolicy defines pool's policy for accepting miner templates
type TemplatePolicy struct {
	// Core policy settings
	AllowMinerTemplates   bool          // Whether to accept miner templates at all
	RequirePoolCoinbase   bool          // Require pool's coinbase structure
	AllowCustomCoinbase   bool          // Allow miners to modify coinbase
	MinCoinbasePoolShare  float64       // Minimum % of coinbase going to pool (0-1)
	MaxDeclarationsPerMin int           // Rate limit on declarations per miner
	DeclarationTTL        time.Duration // How long declarations remain valid

	// Transaction policy
	RequireAllPoolTxs  bool     // Require all pool-selected transactions
	AllowAdditionalTxs bool     // Allow miner to add transactions
	BannedTxPatterns   [][]byte // Transaction patterns to reject
	MinFeeRate         uint64   // Minimum fee rate for additional txs

	// Fallback behavior
	FallbackOnRejection bool          // Use pool template if miner's rejected
	FallbackTimeout     time.Duration // Timeout before falling back to pool template

	// Security
	RequireSignedTemplates bool     // Require cryptographic signatures
	AllowedMinerIDs        []string // Whitelist of miners allowed to declare (empty = all)
}

// -----------------------------------------------------------------------------
// Job Negotiation Protocol Messages (V2 Extension)
// -----------------------------------------------------------------------------

// JobNegotiationMessageType defines job negotiation message types
type JobNegotiationMessageType uint8

const (
	// Declaration messages (0x60-0x6F)
	MsgTypeDeclareTemplate        JobNegotiationMessageType = 0x60
	MsgTypeDeclareTemplateSuccess JobNegotiationMessageType = 0x61
	MsgTypeDeclareTemplateError   JobNegotiationMessageType = 0x62
	MsgTypeRevokeDeclaration      JobNegotiationMessageType = 0x63
	MsgTypeRevokeSuccess          JobNegotiationMessageType = 0x64

	// Template provider messages (0x70-0x7F)
	MsgTypeRequestPoolTemplate   JobNegotiationMessageType = 0x70
	MsgTypePoolTemplate          JobNegotiationMessageType = 0x71
	MsgTypeTemplateUpdate        JobNegotiationMessageType = 0x72
	MsgTypeCommitTemplate        JobNegotiationMessageType = 0x73
	MsgTypeCommitTemplateSuccess JobNegotiationMessageType = 0x74
	MsgTypeCommitTemplateError   JobNegotiationMessageType = 0x75

	// Negotiation control (0x78-0x7F)
	MsgTypeSetTemplatePolicy  JobNegotiationMessageType = 0x78
	MsgTypeGetTemplatePolicy  JobNegotiationMessageType = 0x79
	MsgTypeTemplatePolicyInfo JobNegotiationMessageType = 0x7A
)

// -----------------------------------------------------------------------------
// Hybrid V1/V2 Detection for Job Negotiation
// -----------------------------------------------------------------------------

// NegotiationCapability represents a miner's job negotiation capabilities
type NegotiationCapability struct {
	SupportsV2Negotiation bool   // Supports Stratum V2 job negotiation
	SupportsCustomJobs    bool   // Can propose custom jobs
	SupportsTemplates     bool   // Can propose full templates
	MaxTemplateSize       uint32 // Maximum template size in bytes
	TemplateVersion       uint16 // Supported template protocol version
	MinerPubKey           []byte // Miner's public key for signing
}

// NegotiationDetector detects and negotiates job declaration capabilities
type NegotiationDetector interface {
	// DetectCapabilities detects miner's negotiation capabilities
	DetectCapabilities(conn DuplexConnection) (*NegotiationCapability, error)

	// NegotiateCapabilities performs capability negotiation
	NegotiateCapabilities(conn DuplexConnection, serverCaps *NegotiationCapability) (*NegotiationCapability, error)
}

// -----------------------------------------------------------------------------
// Integration Interfaces
// -----------------------------------------------------------------------------

// JobNegotiationManager coordinates all job negotiation components
type JobNegotiationManager interface {
	// Lifecycle
	Start() error
	Stop() error

	// Server components
	GetDeclaratorServer() JobDeclaratorServer
	GetTemplateProvider() TemplateProvider
	GetTemplateValidator() TemplateValidator

	// Configuration
	SetPolicy(policy TemplatePolicy) error
	GetPolicy() TemplatePolicy
	IsEnabled() bool

	// Metrics
	GetActiveDeclarationCount() int
	GetAcceptedDeclarationRate() float64
	GetAverageValidationTime() time.Duration
}

// JobNegotiationConfig holds configuration for job negotiation system
type JobNegotiationConfig struct {
	Enabled                  bool                   // Enable job negotiation
	ListenAddress            string                 // Address for Job Declarator Server
	TemplateProviderConfig   TemplateProviderConfig // Template provider settings
	Policy                   TemplatePolicy         // Default template policy
	MaxConcurrentValidations int                    // Max concurrent template validations
	ValidationTimeout        time.Duration          // Timeout for template validation
	EnableMetrics            bool                   // Enable negotiation metrics
}
