package binary

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// COMPREHENSIVE ADDITIONAL TESTS FOR 95%+ COVERAGE
// Critical for production-ready mining pool
// =============================================================================

// -----------------------------------------------------------------------------
// Serializer Additional Tests
// -----------------------------------------------------------------------------

func TestSerializer_WriteBytes(t *testing.T) {
	s := NewSerializer()
	s.WriteBytes([]byte{0x01, 0x02, 0x03, 0x04, 0x05})
	assert.Equal(t, []byte{0x01, 0x02, 0x03, 0x04, 0x05}, s.Bytes())
}

func TestSerializer_WriteBytes_Empty(t *testing.T) {
	s := NewSerializer()
	s.WriteBytes([]byte{})
	assert.Equal(t, 0, s.Len())
}

func TestSerializer_WriteFixedBytes_ExactSize(t *testing.T) {
	s := NewSerializer()
	s.WriteFixedBytes([]byte{0x01, 0x02, 0x03, 0x04}, 4)
	assert.Equal(t, []byte{0x01, 0x02, 0x03, 0x04}, s.Bytes())
}

func TestSerializer_WriteFixedBytes_LargerThanN(t *testing.T) {
	s := NewSerializer()
	s.WriteFixedBytes([]byte{0x01, 0x02, 0x03, 0x04, 0x05}, 3)
	assert.Equal(t, []byte{0x01, 0x02, 0x03}, s.Bytes())
	assert.Equal(t, 3, s.Len())
}

func TestSerializer_WriteSTR0_255_Truncation(t *testing.T) {
	s := NewSerializer()
	// Create a string longer than 255 bytes
	longStr := make([]byte, 300)
	for i := range longStr {
		longStr[i] = 'a'
	}
	s.WriteSTR0_255(string(longStr))

	// Should be truncated to 255 + 1 (length byte) = 256 bytes
	assert.Equal(t, 256, s.Len())
	assert.Equal(t, byte(255), s.Bytes()[0]) // Length should be 255
}

func TestSerializer_WriteSTR0_255_Empty(t *testing.T) {
	s := NewSerializer()
	s.WriteSTR0_255("")
	assert.Equal(t, []byte{0x00}, s.Bytes())
}

func TestSerializer_WriteHeader(t *testing.T) {
	s := NewSerializer()
	header := &FrameHeader{
		ExtensionType: 0x0001,
		MsgType:       0x20,
		MsgLength:     0x000100,
	}
	s.WriteHeader(header)

	expected := []byte{0x01, 0x00, 0x20, 0x00, 0x01, 0x00}
	assert.Equal(t, expected, s.Bytes())
}

func TestSerializer_Len(t *testing.T) {
	s := NewSerializer()
	assert.Equal(t, 0, s.Len())

	s.WriteU32(0x12345678)
	assert.Equal(t, 4, s.Len())

	s.WriteU16(0x1234)
	assert.Equal(t, 6, s.Len())
}

func TestSerializer_MultipleWrites(t *testing.T) {
	s := NewSerializer()
	s.WriteU8(0x01)
	s.WriteU16(0x0203)
	s.WriteU32(0x04050607)
	s.WriteU64(0x08090A0B0C0D0E0F)

	assert.Equal(t, 15, s.Len())
}

// -----------------------------------------------------------------------------
// Deserializer Additional Tests
// -----------------------------------------------------------------------------

func TestDeserializer_Remaining(t *testing.T) {
	d := NewDeserializer([]byte{0x01, 0x02, 0x03, 0x04})
	assert.Equal(t, 4, d.Remaining())

	d.ReadU8()
	assert.Equal(t, 3, d.Remaining())

	d.ReadU16()
	assert.Equal(t, 1, d.Remaining())
}

func TestDeserializer_Position(t *testing.T) {
	d := NewDeserializer([]byte{0x01, 0x02, 0x03, 0x04})
	assert.Equal(t, 0, d.Position())

	d.ReadU8()
	assert.Equal(t, 1, d.Position())

	d.ReadU16()
	assert.Equal(t, 3, d.Position())
}

func TestDeserializer_ReadF32(t *testing.T) {
	// Float32 240000000.0 in IEEE 754 little-endian
	s := NewSerializer()
	s.WriteF32(240000000.0)

	d := NewDeserializer(s.Bytes())
	v, err := d.ReadF32()
	require.NoError(t, err)
	assert.InDelta(t, 240000000.0, v, 1.0)
}

func TestDeserializer_ReadF32_EOF(t *testing.T) {
	d := NewDeserializer([]byte{0x00, 0x00, 0x00}) // Only 3 bytes
	_, err := d.ReadF32()
	assert.Error(t, err)
	assert.Equal(t, io.ErrUnexpectedEOF, err)
}

func TestDeserializer_ReadBytes(t *testing.T) {
	d := NewDeserializer([]byte{0x01, 0x02, 0x03, 0x04, 0x05})

	bytes, err := d.ReadBytes(3)
	require.NoError(t, err)
	assert.Equal(t, []byte{0x01, 0x02, 0x03}, bytes)
	assert.Equal(t, 2, d.Remaining())
}

func TestDeserializer_ReadBytes_EOF(t *testing.T) {
	d := NewDeserializer([]byte{0x01, 0x02})
	_, err := d.ReadBytes(5)
	assert.Error(t, err)
	assert.Equal(t, io.ErrUnexpectedEOF, err)
}

func TestDeserializer_ReadFixedBytes32(t *testing.T) {
	data := make([]byte, 32)
	for i := range data {
		data[i] = byte(i)
	}

	d := NewDeserializer(data)
	result, err := d.ReadFixedBytes32()
	require.NoError(t, err)
	assert.Equal(t, data, result[:])
}

func TestDeserializer_ReadFixedBytes32_EOF(t *testing.T) {
	d := NewDeserializer(make([]byte, 31)) // One byte short
	_, err := d.ReadFixedBytes32()
	assert.Error(t, err)
	assert.Equal(t, io.ErrUnexpectedEOF, err)
}

func TestDeserializer_ReadSTR0_255_EOF_Length(t *testing.T) {
	d := NewDeserializer([]byte{}) // Empty
	_, err := d.ReadSTR0_255()
	assert.Error(t, err)
}

func TestDeserializer_ReadHeader_EOF(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"partial_ext_type", []byte{0x01}},
		{"no_msg_type", []byte{0x01, 0x00}},
		{"partial_length", []byte{0x01, 0x00, 0x20, 0x00}},
		{"almost_complete", []byte{0x01, 0x00, 0x20, 0x00, 0x01}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDeserializer(tt.data)
			_, err := d.ReadHeader()
			assert.Error(t, err)
		})
	}
}

// -----------------------------------------------------------------------------
// Frame Header Tests
// -----------------------------------------------------------------------------

func TestFrameHeader_Serialize_Comprehensive(t *testing.T) {
	h := &FrameHeader{
		ExtensionType: 0x0001,
		MsgType:       0x30,
		MsgLength:     0x000FFF, // 24-bit value
	}

	serialized := h.Serialize()
	assert.Equal(t, HeaderSize, len(serialized))

	// Parse it back
	parsed, err := ParseHeader(serialized)
	require.NoError(t, err)
	assert.Equal(t, h.ExtensionType, parsed.ExtensionType)
	assert.Equal(t, h.MsgType, parsed.MsgType)
	assert.Equal(t, h.MsgLength, parsed.MsgLength)
}

func TestFrameHeader_Serialize_MaxLength(t *testing.T) {
	h := &FrameHeader{
		ExtensionType: 0xFFFF,
		MsgType:       0xFF,
		MsgLength:     0xFFFFFF, // Max 24-bit value
	}

	serialized := h.Serialize()
	parsed, err := ParseHeader(serialized)
	require.NoError(t, err)
	assert.Equal(t, uint32(0xFFFFFF), parsed.MsgLength)
}

func TestParseHeader_InvalidLength(t *testing.T) {
	_, err := ParseHeader([]byte{0x01, 0x02, 0x03}) // Less than HeaderSize
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidHeader, err)
}

// -----------------------------------------------------------------------------
// STR0_255 Tests
// -----------------------------------------------------------------------------

func TestSTR0_255_Serialize_Comprehensive(t *testing.T) {
	s := STR0_255("hello")
	serialized := s.Serialize()
	assert.Equal(t, []byte{0x05, 'h', 'e', 'l', 'l', 'o'}, serialized)
}

func TestSTR0_255_Serialize_Truncation(t *testing.T) {
	// Create string longer than 255
	longStr := make([]byte, 300)
	for i := range longStr {
		longStr[i] = 'x'
	}
	s := STR0_255(string(longStr))
	serialized := s.Serialize()

	assert.Equal(t, 256, len(serialized)) // 1 byte length + 255 bytes
	assert.Equal(t, byte(255), serialized[0])
}

func TestSTR0_255_Serialize_Empty(t *testing.T) {
	s := STR0_255("")
	serialized := s.Serialize()
	assert.Equal(t, []byte{0x00}, serialized)
}

func TestParseSTR0_255_Comprehensive(t *testing.T) {
	data := []byte{0x05, 'h', 'e', 'l', 'l', 'o', 0x00, 0x00}
	str, n, err := ParseSTR0_255(data)
	require.NoError(t, err)
	assert.Equal(t, STR0_255("hello"), str)
	assert.Equal(t, 6, n) // 1 + 5
}

func TestParseSTR0_255_Empty(t *testing.T) {
	data := []byte{0x00}
	str, n, err := ParseSTR0_255(data)
	require.NoError(t, err)
	assert.Equal(t, STR0_255(""), str)
	assert.Equal(t, 1, n)
}

func TestParseSTR0_255_TruncatedLength(t *testing.T) {
	data := []byte{} // Empty buffer
	_, _, err := ParseSTR0_255(data)
	assert.Error(t, err)
	assert.Equal(t, ErrTruncatedMessage, err)
}

func TestParseSTR0_255_TruncatedContent(t *testing.T) {
	data := []byte{0x10, 'a', 'b', 'c'} // Says 16 bytes but only 3
	_, _, err := ParseSTR0_255(data)
	assert.Error(t, err)
	assert.Equal(t, ErrTruncatedMessage, err)
}

// -----------------------------------------------------------------------------
// Deserialize Error Path Tests
// -----------------------------------------------------------------------------

func TestDeserializeSetupConnection_EOF(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"only_protocol", []byte{0x00}},
		{"partial_min_version", []byte{0x00, 0x02}},
		{"partial_max_version", []byte{0x00, 0x02, 0x00, 0x02}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDeserializer(tt.data)
			_, err := d.DeserializeSetupConnection()
			assert.Error(t, err)
		})
	}
}

func TestDeserializeSetupConnectionSuccess_EOF(t *testing.T) {
	d := NewDeserializer([]byte{0x02, 0x00}) // Only version, no flags
	_, err := d.DeserializeSetupConnectionSuccess()
	assert.Error(t, err)
}

func TestDeserializeSetupConnectionError_EOF(t *testing.T) {
	d := NewDeserializer([]byte{0x00, 0x00, 0x00}) // Partial flags
	_, err := d.DeserializeSetupConnectionError()
	assert.Error(t, err)
}

func TestDeserializeOpenStandardMiningChannel_EOF(t *testing.T) {
	d := NewDeserializer([]byte{0x01, 0x00, 0x00, 0x00}) // Only RequestID
	_, err := d.DeserializeOpenStandardMiningChannel()
	assert.Error(t, err)
}

func TestDeserializeOpenStandardMiningChannelSuccess_EOF(t *testing.T) {
	d := NewDeserializer([]byte{0x01, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00}) // RequestID + ChannelID only
	_, err := d.DeserializeOpenStandardMiningChannelSuccess()
	assert.Error(t, err)
}

func TestDeserializeOpenStandardMiningChannelError_EOF(t *testing.T) {
	d := NewDeserializer([]byte{0x01, 0x00, 0x00, 0x00}) // Only RequestID
	_, err := d.DeserializeOpenStandardMiningChannelError()
	assert.Error(t, err)
}

func TestDeserializeNewMiningJob_EOF(t *testing.T) {
	d := NewDeserializer([]byte{0x01, 0x00, 0x00, 0x00}) // Only ChannelID
	_, err := d.DeserializeNewMiningJob()
	assert.Error(t, err)
}

func TestDeserializeSetNewPrevHash_EOF(t *testing.T) {
	d := NewDeserializer([]byte{0x01, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00}) // ChannelID + JobID only
	_, err := d.DeserializeSetNewPrevHash()
	assert.Error(t, err)
}

func TestDeserializeSubmitSharesStandard_EOF(t *testing.T) {
	d := NewDeserializer([]byte{0x01, 0x00, 0x00, 0x00}) // Only ChannelID
	_, err := d.DeserializeSubmitSharesStandard()
	assert.Error(t, err)
}

func TestDeserializeSubmitSharesSuccess_EOF(t *testing.T) {
	d := NewDeserializer([]byte{0x01, 0x00, 0x00, 0x00}) // Only ChannelID
	_, err := d.DeserializeSubmitSharesSuccess()
	assert.Error(t, err)
}

func TestDeserializeSubmitSharesError_EOF(t *testing.T) {
	d := NewDeserializer([]byte{0x01, 0x00, 0x00, 0x00}) // Only ChannelID
	_, err := d.DeserializeSubmitSharesError()
	assert.Error(t, err)
}

func TestDeserializeSetTarget_EOF(t *testing.T) {
	d := NewDeserializer([]byte{0x01, 0x00, 0x00, 0x00}) // Only ChannelID
	_, err := d.DeserializeSetTarget()
	assert.Error(t, err)
}

func TestDeserializeReconnect_EOF(t *testing.T) {
	d := NewDeserializer([]byte{0x05, 'h', 'e', 'l', 'l', 'o'}) // Only host, no port
	_, err := d.DeserializeReconnect()
	assert.Error(t, err)
}

// -----------------------------------------------------------------------------
// Edge Case Tests
// -----------------------------------------------------------------------------

func TestSerializer_WriteU24_MaxValue(t *testing.T) {
	s := NewSerializer()
	s.WriteU24(0xFFFFFF) // Max 24-bit value
	assert.Equal(t, []byte{0xFF, 0xFF, 0xFF}, s.Bytes())
}

func TestSerializer_WriteU24_Zero(t *testing.T) {
	s := NewSerializer()
	s.WriteU24(0)
	assert.Equal(t, []byte{0x00, 0x00, 0x00}, s.Bytes())
}

func TestDeserializer_ReadU24_MaxValue(t *testing.T) {
	d := NewDeserializer([]byte{0xFF, 0xFF, 0xFF})
	v, err := d.ReadU24()
	require.NoError(t, err)
	assert.Equal(t, uint32(0xFFFFFF), v)
}

func TestNewMiningJob_WithFuturePrevHash(t *testing.T) {
	original := &NewMiningJob{
		ChannelID:      42,
		JobID:          1000,
		FuturePrevHash: true, // Test true case
		Version:        0x20000000,
		VersionMask:    0x1fffe000,
	}

	s := NewSerializer()
	payload := s.SerializeNewMiningJob(original)

	d := NewDeserializer(payload)
	parsed, err := d.DeserializeNewMiningJob()
	require.NoError(t, err)

	assert.True(t, parsed.FuturePrevHash)
}

// -----------------------------------------------------------------------------
// Constants Tests
// -----------------------------------------------------------------------------

func TestMessageTypeConstants(t *testing.T) {
	// Verify message type constants are correctly defined
	assert.Equal(t, uint8(0x00), MsgTypeSetupConnection)
	assert.Equal(t, uint8(0x01), MsgTypeSetupConnectionSuccess)
	assert.Equal(t, uint8(0x02), MsgTypeSetupConnectionError)
	assert.Equal(t, uint8(0x10), MsgTypeOpenStandardMiningChannel)
	assert.Equal(t, uint8(0x20), MsgTypeNewMiningJob)
	assert.Equal(t, uint8(0x22), MsgTypeSetNewPrevHash)
	assert.Equal(t, uint8(0x30), MsgTypeSubmitSharesStandard)
	assert.Equal(t, uint8(0x32), MsgTypeSubmitSharesSuccess)
	assert.Equal(t, uint8(0x33), MsgTypeSubmitSharesError)
	assert.Equal(t, uint8(0x40), MsgTypeSetTarget)
	assert.Equal(t, uint8(0x50), MsgTypeReconnect)
}

func TestExtensionTypeConstants(t *testing.T) {
	assert.Equal(t, uint16(0x0000), ExtensionTypeNone)
	assert.Equal(t, uint16(0x0001), ExtensionTypeVersionRolling)
	assert.Equal(t, uint16(0x0002), ExtensionTypeMinimumDiff)
	assert.Equal(t, uint16(0x0004), ExtensionTypeWorkSelection)
}

func TestErrorCodeConstants(t *testing.T) {
	assert.Equal(t, uint8(0x00), ErrUnknownMessage)
	assert.Equal(t, uint8(0x05), ErrInvalidShare)
	assert.Equal(t, uint8(0x06), ErrStaleShare)
	assert.Equal(t, uint8(0x07), ErrDuplicateShare)
	assert.Equal(t, uint8(0x08), ErrLowDifficultyShare)
}

func TestErrorVariables(t *testing.T) {
	assert.NotNil(t, ErrInvalidMessageLength)
	assert.NotNil(t, ErrUnsupportedMessage)
	assert.NotNil(t, ErrInvalidHeader)
	assert.NotNil(t, ErrTruncatedMessage)
	assert.NotNil(t, ErrBufferTooSmall)
}

// -----------------------------------------------------------------------------
// Benchmark Additional Tests
// -----------------------------------------------------------------------------

func BenchmarkSerializer_WriteSTR0_255_Long(b *testing.B) {
	s := NewSerializer()
	longStr := make([]byte, 255)
	for i := range longStr {
		longStr[i] = 'a'
	}
	str := string(longStr)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Reset()
		s.WriteSTR0_255(str)
	}
}

func BenchmarkDeserializer_ReadFixedBytes32(b *testing.B) {
	data := make([]byte, 32)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d := NewDeserializer(data)
		d.ReadFixedBytes32()
	}
}

func BenchmarkSerializeFrame_Complete(b *testing.B) {
	s := NewSerializer()
	msg := &SetupConnection{
		Protocol:        0,
		MinVersion:      2,
		MaxVersion:      2,
		Flags:           0x00000007,
		Endpoint:        "pool.chimera.io:3334",
		Vendor:          "BlockDAG",
		HardwareVersion: "X100",
		FirmwareVersion: "1.0.0",
		DeviceID:        "device-001",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		payload := s.SerializeSetupConnection(msg)
		s.SerializeFrame(MsgTypeSetupConnection, ExtensionTypeNone, payload)
	}
}
