package binary

import (
	"encoding/binary"
	"errors"
)

// =============================================================================
// STRATUM V2 BINARY PROTOCOL TYPES
// Based on Stratum V2 Specification (SRI v1.0)
// =============================================================================

// Message type constants for Stratum V2
const (
	// Mining Protocol Messages (0x00-0x1F)
	MsgTypeSetupConnection        uint8 = 0x00
	MsgTypeSetupConnectionSuccess uint8 = 0x01
	MsgTypeSetupConnectionError   uint8 = 0x02
	MsgTypeChannelEndpointChanged uint8 = 0x03
	MsgTypeSetupMiningConnection  uint8 = 0x04

	// Mining Channel Messages (0x10-0x1F)
	MsgTypeOpenStandardMiningChannel        uint8 = 0x10
	MsgTypeOpenStandardMiningChannelSuccess uint8 = 0x11
	MsgTypeOpenStandardMiningChannelError   uint8 = 0x12
	MsgTypeOpenExtendedMiningChannel        uint8 = 0x13
	MsgTypeOpenExtendedMiningChannelSuccess uint8 = 0x14
	MsgTypeOpenExtendedMiningChannelError   uint8 = 0x15
	MsgTypeUpdateChannel                    uint8 = 0x16
	MsgTypeUpdateChannelError               uint8 = 0x17
	MsgTypeCloseChannel                     uint8 = 0x18

	// Mining Job Messages (0x20-0x2F)
	MsgTypeNewMiningJob              uint8 = 0x20
	MsgTypeNewExtendedMiningJob      uint8 = 0x21
	MsgTypeSetNewPrevHash            uint8 = 0x22
	MsgTypeSetCustomMiningJob        uint8 = 0x23
	MsgTypeSetCustomMiningJobSuccess uint8 = 0x24
	MsgTypeSetCustomMiningJobError   uint8 = 0x25

	// Share Submission Messages (0x30-0x3F)
	MsgTypeSubmitSharesStandard uint8 = 0x30
	MsgTypeSubmitSharesExtended uint8 = 0x31
	MsgTypeSubmitSharesSuccess  uint8 = 0x32
	MsgTypeSubmitSharesError    uint8 = 0x33

	// Difficulty Messages (0x40-0x4F)
	MsgTypeSetTarget       uint8 = 0x40
	MsgTypeSetGroupChannel uint8 = 0x41

	// Connection Control Messages (0x50-0x5F)
	MsgTypeReconnect           uint8 = 0x50
	MsgTypeSetExtranoncePrefix uint8 = 0x51
)

// Extension type flags
const (
	ExtensionTypeNone           uint16 = 0x0000
	ExtensionTypeVersionRolling uint16 = 0x0001
	ExtensionTypeMinimumDiff    uint16 = 0x0002
	ExtensionTypeWorkSelection  uint16 = 0x0004
)

// Error codes
const (
	ErrUnknownMessage       uint8 = 0x00
	ErrInvalidExtensionType uint8 = 0x01
	ErrInvalidChannelID     uint8 = 0x02
	ErrInvalidJobID         uint8 = 0x03
	ErrInvalidTarget        uint8 = 0x04
	ErrInvalidShare         uint8 = 0x05
	ErrStaleShare           uint8 = 0x06
	ErrDuplicateShare       uint8 = 0x07
	ErrLowDifficultyShare   uint8 = 0x08
	ErrUnauthorized         uint8 = 0x09
	ErrNotSubscribed        uint8 = 0x0A
)

// Errors
var (
	ErrInvalidMessageLength = errors.New("invalid message length")
	ErrUnsupportedMessage   = errors.New("unsupported message type")
	ErrInvalidHeader        = errors.New("invalid message header")
	ErrTruncatedMessage     = errors.New("truncated message")
	ErrBufferTooSmall       = errors.New("buffer too small")
)

// =============================================================================
// Frame Header
// =============================================================================

// FrameHeader represents a Stratum V2 message frame header
// Format: [extension_type: u16] [msg_type: u8] [msg_length: u24]
type FrameHeader struct {
	ExtensionType uint16
	MsgType       uint8
	MsgLength     uint32 // 24-bit in wire format
}

// HeaderSize is the size of the frame header in bytes
const HeaderSize = 6

// Serialize serializes the header to bytes
func (h *FrameHeader) Serialize() []byte {
	buf := make([]byte, HeaderSize)
	binary.LittleEndian.PutUint16(buf[0:2], h.ExtensionType)
	buf[2] = h.MsgType
	// 24-bit length in little endian
	buf[3] = byte(h.MsgLength & 0xFF)
	buf[4] = byte((h.MsgLength >> 8) & 0xFF)
	buf[5] = byte((h.MsgLength >> 16) & 0xFF)
	return buf
}

// ParseHeader parses a frame header from bytes
func ParseHeader(data []byte) (*FrameHeader, error) {
	if len(data) < HeaderSize {
		return nil, ErrInvalidHeader
	}

	h := &FrameHeader{
		ExtensionType: binary.LittleEndian.Uint16(data[0:2]),
		MsgType:       data[2],
		MsgLength:     uint32(data[3]) | uint32(data[4])<<8 | uint32(data[5])<<16,
	}
	return h, nil
}

// =============================================================================
// String Types (Variable Length)
// =============================================================================

// STR0_255 represents a string with max 255 bytes (1-byte length prefix)
type STR0_255 string

// Serialize serializes the string with length prefix
func (s STR0_255) Serialize() []byte {
	str := string(s)
	if len(str) > 255 {
		str = str[:255]
	}
	buf := make([]byte, 1+len(str))
	buf[0] = byte(len(str))
	copy(buf[1:], str)
	return buf
}

// ParseSTR0_255 parses a string from bytes
func ParseSTR0_255(data []byte) (STR0_255, int, error) {
	if len(data) < 1 {
		return "", 0, ErrTruncatedMessage
	}
	length := int(data[0])
	if len(data) < 1+length {
		return "", 0, ErrTruncatedMessage
	}
	return STR0_255(data[1 : 1+length]), 1 + length, nil
}

// =============================================================================
// Core Message Structures
// =============================================================================

// SetupConnection is sent by client to initiate connection
type SetupConnection struct {
	Protocol        uint8    // Mining protocol version
	MinVersion      uint16   // Minimum supported version
	MaxVersion      uint16   // Maximum supported version
	Flags           uint32   // Feature flags
	Endpoint        STR0_255 // Endpoint host:port
	Vendor          STR0_255 // Vendor name
	HardwareVersion STR0_255 // Hardware version string
	FirmwareVersion STR0_255 // Firmware version string
	DeviceID        STR0_255 // Unique device identifier
}

// SetupConnectionSuccess is sent by server on successful setup
type SetupConnectionSuccess struct {
	UsedVersion uint16 // Negotiated protocol version
	Flags       uint32 // Supported feature flags
}

// SetupConnectionError is sent by server on setup failure
type SetupConnectionError struct {
	Flags     uint32   // Flags for error details
	ErrorCode STR0_255 // Error code string
}

// OpenStandardMiningChannel requests opening a mining channel
type OpenStandardMiningChannel struct {
	RequestID         uint32   // Client-assigned request ID
	UserIdentity      STR0_255 // User/worker identity (wallet.worker)
	NominalHashrate   float32  // Expected hashrate in H/s
	MaxTargetRequired uint32   // Maximum target (minimum difficulty)
}

// OpenStandardMiningChannelSuccess confirms channel opened
type OpenStandardMiningChannelSuccess struct {
	RequestID       uint32   // Matching request ID
	ChannelID       uint32   // Server-assigned channel ID
	Target          [32]byte // Initial mining target
	ExtraNonce2Size uint16   // Size of extranonce2 in bytes
	GroupChannelID  uint32   // Group channel identifier
}

// OpenStandardMiningChannelError indicates channel open failure
type OpenStandardMiningChannelError struct {
	RequestID uint32   // Matching request ID
	ErrorCode STR0_255 // Error code
}

// NewMiningJob contains a new mining job
type NewMiningJob struct {
	ChannelID      uint32 // Target channel
	JobID          uint32 // Unique job identifier
	FuturePrevHash bool   // If true, prevhash not yet available
	Version        uint32 // Block version
	VersionMask    uint32 // Mask for version rolling
}

// SetNewPrevHash updates the previous block hash
type SetNewPrevHash struct {
	ChannelID uint32   // Target channel
	JobID     uint32   // Job to update
	PrevHash  [32]byte // New previous block hash
	MinNTime  uint32   // Minimum ntime value
	NBits     uint32   // Target difficulty bits
}

// SubmitSharesStandard submits a standard share
type SubmitSharesStandard struct {
	ChannelID   uint32 // Channel ID
	SequenceNum uint32 // Sequence number for tracking
	JobID       uint32 // Job being mined
	Nonce       uint32 // Nonce solution
	NTime       uint32 // Block time
	Version     uint32 // Block version (if version rolling)
}

// SubmitSharesSuccess acknowledges accepted shares
type SubmitSharesSuccess struct {
	ChannelID       uint32 // Channel ID
	LastSequenceNum uint32 // Last accepted sequence number
	NewSubmits      uint32 // Count of newly accepted shares
	NewDifficulty   uint64 // New target difficulty (if changed)
}

// SubmitSharesError indicates share rejection
type SubmitSharesError struct {
	ChannelID   uint32   // Channel ID
	SequenceNum uint32   // Failed sequence number
	ErrorCode   STR0_255 // Error code
}

// SetTarget updates the mining target
type SetTarget struct {
	ChannelID uint32   // Target channel
	MaxTarget [32]byte // New maximum target
}

// Reconnect instructs client to reconnect
type Reconnect struct {
	NewHost STR0_255 // New host to connect to
	NewPort uint16   // New port
}
