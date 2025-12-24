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
