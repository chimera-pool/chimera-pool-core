package binary

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// TDD TESTS FOR STRATUM V2 BINARY TYPES
// =============================================================================

// -----------------------------------------------------------------------------
// Frame Header Tests
// -----------------------------------------------------------------------------

func TestFrameHeader_Serialize(t *testing.T) {
	tests := []struct {
		name     string
		header   FrameHeader
		expected []byte
	}{
		{
			name: "SetupConnection header",
			header: FrameHeader{
				ExtensionType: ExtensionTypeNone,
				MsgType:       MsgTypeSetupConnection,
				MsgLength:     100,
			},
			expected: []byte{0x00, 0x00, 0x00, 0x64, 0x00, 0x00},
		},
		{
			name: "NewMiningJob header with version rolling",
			header: FrameHeader{
				ExtensionType: ExtensionTypeVersionRolling,
				MsgType:       MsgTypeNewMiningJob,
				MsgLength:     256,
			},
			expected: []byte{0x01, 0x00, 0x20, 0x00, 0x01, 0x00},
		},
		{
			name: "Large message length (24-bit)",
			header: FrameHeader{
				ExtensionType: ExtensionTypeNone,
				MsgType:       MsgTypeSubmitSharesStandard,
				MsgLength:     0x123456, // 1193046 bytes
			},
			expected: []byte{0x00, 0x00, 0x30, 0x56, 0x34, 0x12},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.header.Serialize()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseHeader(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		expected    *FrameHeader
		expectError bool
	}{
		{
			name: "Valid SetupConnection header",
			data: []byte{0x00, 0x00, 0x00, 0x64, 0x00, 0x00},
			expected: &FrameHeader{
				ExtensionType: ExtensionTypeNone,
				MsgType:       MsgTypeSetupConnection,
				MsgLength:     100,
			},
			expectError: false,
		},
		{
			name: "Valid NewMiningJob with extensions",
			data: []byte{0x01, 0x00, 0x20, 0x00, 0x01, 0x00},
			expected: &FrameHeader{
				ExtensionType: ExtensionTypeVersionRolling,
				MsgType:       MsgTypeNewMiningJob,
				MsgLength:     256,
			},
			expectError: false,
		},
		{
			name:        "Too short - 5 bytes",
			data:        []byte{0x00, 0x00, 0x00, 0x64, 0x00},
			expected:    nil,
			expectError: true,
		},
		{
			name:        "Empty data",
			data:        []byte{},
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseHeader(tt.data)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestFrameHeader_RoundTrip(t *testing.T) {
	headers := []FrameHeader{
		{ExtensionType: 0, MsgType: MsgTypeSetupConnection, MsgLength: 0},
		{ExtensionType: 0, MsgType: MsgTypeNewMiningJob, MsgLength: 1000},
		{ExtensionType: ExtensionTypeVersionRolling, MsgType: MsgTypeSubmitSharesStandard, MsgLength: 0xFFFFFF},
		{ExtensionType: ExtensionTypeWorkSelection, MsgType: MsgTypeSetTarget, MsgLength: 32},
	}

	for _, original := range headers {
		serialized := original.Serialize()
		parsed, err := ParseHeader(serialized)
		require.NoError(t, err)
		assert.Equal(t, original.ExtensionType, parsed.ExtensionType)
		assert.Equal(t, original.MsgType, parsed.MsgType)
		assert.Equal(t, original.MsgLength, parsed.MsgLength)
	}
}

// -----------------------------------------------------------------------------
// String Type Tests
// -----------------------------------------------------------------------------

func TestSTR0_255_Serialize(t *testing.T) {
	tests := []struct {
		name     string
		input    STR0_255
		expected []byte
	}{
		{
			name:     "Empty string",
			input:    STR0_255(""),
			expected: []byte{0x00},
		},
		{
			name:     "Short string",
			input:    STR0_255("hello"),
			expected: []byte{0x05, 'h', 'e', 'l', 'l', 'o'},
		},
		{
			name:     "Worker identity",
			input:    STR0_255("kaspa:qr1234.worker1"),
			expected: append([]byte{20}, []byte("kaspa:qr1234.worker1")...),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.Serialize()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSTR0_255_TruncatesLongStrings(t *testing.T) {
	// Create a 300-character string
	longStr := make([]byte, 300)
	for i := range longStr {
		longStr[i] = 'a'
	}

	s := STR0_255(longStr)
	serialized := s.Serialize()

	// Should be truncated to 255 + 1 length byte = 256 bytes
	assert.Equal(t, 256, len(serialized))
	assert.Equal(t, byte(255), serialized[0])
}

func TestParseSTR0_255(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		expected    STR0_255
		bytesRead   int
		expectError bool
	}{
		{
			name:        "Empty string",
			data:        []byte{0x00},
			expected:    STR0_255(""),
			bytesRead:   1,
			expectError: false,
		},
		{
			name:        "Short string",
			data:        []byte{0x05, 'h', 'e', 'l', 'l', 'o'},
			expected:    STR0_255("hello"),
			bytesRead:   6,
			expectError: false,
		},
		{
			name:        "String with trailing data",
			data:        []byte{0x03, 'a', 'b', 'c', 'x', 'y', 'z'},
			expected:    STR0_255("abc"),
			bytesRead:   4,
			expectError: false,
		},
		{
			name:        "Truncated - length says 10 but only 5 bytes",
			data:        []byte{0x0A, 'a', 'b', 'c', 'd', 'e'},
			expected:    "",
			bytesRead:   0,
			expectError: true,
		},
		{
			name:        "Empty data",
			data:        []byte{},
			expected:    "",
			bytesRead:   0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, bytesRead, err := ParseSTR0_255(tt.data)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
				assert.Equal(t, tt.bytesRead, bytesRead)
			}
		})
	}
}

func TestSTR0_255_RoundTrip(t *testing.T) {
	strings := []STR0_255{
		"",
		"test",
		"wallet.worker",
		"BlockDAG X100 Miner",
		"kaspa:qr0123456789abcdef.miner001",
	}

	for _, original := range strings {
		serialized := original.Serialize()
		parsed, bytesRead, err := ParseSTR0_255(serialized)
		require.NoError(t, err)
		assert.Equal(t, original, parsed)
		assert.Equal(t, len(serialized), bytesRead)
	}
}

// -----------------------------------------------------------------------------
// Message Type Constants Tests
// -----------------------------------------------------------------------------

func TestMessageTypeConstants_Unique(t *testing.T) {
	types := []uint8{
		MsgTypeSetupConnection,
		MsgTypeSetupConnectionSuccess,
		MsgTypeSetupConnectionError,
		MsgTypeChannelEndpointChanged,
		MsgTypeSetupMiningConnection,
		MsgTypeOpenStandardMiningChannel,
		MsgTypeOpenStandardMiningChannelSuccess,
		MsgTypeOpenStandardMiningChannelError,
		MsgTypeOpenExtendedMiningChannel,
		MsgTypeOpenExtendedMiningChannelSuccess,
		MsgTypeOpenExtendedMiningChannelError,
		MsgTypeUpdateChannel,
		MsgTypeUpdateChannelError,
		MsgTypeCloseChannel,
		MsgTypeNewMiningJob,
		MsgTypeNewExtendedMiningJob,
		MsgTypeSetNewPrevHash,
		MsgTypeSetCustomMiningJob,
		MsgTypeSetCustomMiningJobSuccess,
		MsgTypeSetCustomMiningJobError,
		MsgTypeSubmitSharesStandard,
		MsgTypeSubmitSharesExtended,
		MsgTypeSubmitSharesSuccess,
		MsgTypeSubmitSharesError,
		MsgTypeSetTarget,
		MsgTypeSetGroupChannel,
		MsgTypeReconnect,
		MsgTypeSetExtranoncePrefix,
	}

	seen := make(map[uint8]bool)
	for _, mt := range types {
		assert.False(t, seen[mt], "Message type 0x%02X should be unique", mt)
		seen[mt] = true
	}
}

func TestMessageTypeRanges(t *testing.T) {
	// Verify message types are in their expected ranges

	// Connection messages: 0x00-0x0F
	assert.True(t, MsgTypeSetupConnection >= 0x00 && MsgTypeSetupConnection <= 0x0F)
	assert.True(t, MsgTypeSetupConnectionSuccess >= 0x00 && MsgTypeSetupConnectionSuccess <= 0x0F)
	assert.True(t, MsgTypeSetupConnectionError >= 0x00 && MsgTypeSetupConnectionError <= 0x0F)

	// Channel messages: 0x10-0x1F
	assert.True(t, MsgTypeOpenStandardMiningChannel >= 0x10 && MsgTypeOpenStandardMiningChannel <= 0x1F)
	assert.True(t, MsgTypeCloseChannel >= 0x10 && MsgTypeCloseChannel <= 0x1F)

	// Job messages: 0x20-0x2F
	assert.True(t, MsgTypeNewMiningJob >= 0x20 && MsgTypeNewMiningJob <= 0x2F)
	assert.True(t, MsgTypeSetNewPrevHash >= 0x20 && MsgTypeSetNewPrevHash <= 0x2F)

	// Share messages: 0x30-0x3F
	assert.True(t, MsgTypeSubmitSharesStandard >= 0x30 && MsgTypeSubmitSharesStandard <= 0x3F)
	assert.True(t, MsgTypeSubmitSharesError >= 0x30 && MsgTypeSubmitSharesError <= 0x3F)

	// Target messages: 0x40-0x4F
	assert.True(t, MsgTypeSetTarget >= 0x40 && MsgTypeSetTarget <= 0x4F)

	// Control messages: 0x50-0x5F
	assert.True(t, MsgTypeReconnect >= 0x50 && MsgTypeReconnect <= 0x5F)
}

// -----------------------------------------------------------------------------
// Error Code Tests
// -----------------------------------------------------------------------------

func TestErrorCodes_Unique(t *testing.T) {
	codes := []uint8{
		ErrUnknownMessage,
		ErrInvalidExtensionType,
		ErrInvalidChannelID,
		ErrInvalidJobID,
		ErrInvalidTarget,
		ErrInvalidShare,
		ErrStaleShare,
		ErrDuplicateShare,
		ErrLowDifficultyShare,
		ErrUnauthorized,
		ErrNotSubscribed,
	}

	seen := make(map[uint8]bool)
	for _, code := range codes {
		assert.False(t, seen[code], "Error code 0x%02X should be unique", code)
		seen[code] = true
	}
}

// -----------------------------------------------------------------------------
// Extension Type Tests
// -----------------------------------------------------------------------------

func TestExtensionTypes_Flags(t *testing.T) {
	// Extension types should be bit flags that can be combined
	combined := ExtensionTypeVersionRolling | ExtensionTypeMinimumDiff | ExtensionTypeWorkSelection

	assert.True(t, combined&ExtensionTypeVersionRolling != 0)
	assert.True(t, combined&ExtensionTypeMinimumDiff != 0)
	assert.True(t, combined&ExtensionTypeWorkSelection != 0)
	assert.True(t, combined&ExtensionTypeNone == 0) // None is 0
}

// -----------------------------------------------------------------------------
// Constants Validation Tests
// -----------------------------------------------------------------------------

func TestHeaderSize(t *testing.T) {
	// Header should be exactly 6 bytes
	assert.Equal(t, 6, HeaderSize)

	// Verify by serializing a header
	h := FrameHeader{ExtensionType: 0, MsgType: 0, MsgLength: 0}
	assert.Equal(t, HeaderSize, len(h.Serialize()))
}
