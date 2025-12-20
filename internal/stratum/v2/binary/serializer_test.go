package binary

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// TDD TESTS FOR V2 BINARY SERIALIZER/DESERIALIZER
// =============================================================================

// -----------------------------------------------------------------------------
// Serializer Primitive Tests
// -----------------------------------------------------------------------------

func TestSerializer_WriteU8(t *testing.T) {
	s := NewSerializer()
	s.WriteU8(0x42)
	assert.Equal(t, []byte{0x42}, s.Bytes())
}

func TestSerializer_WriteU16(t *testing.T) {
	s := NewSerializer()
	s.WriteU16(0x1234)
	// Little-endian: 0x34, 0x12
	assert.Equal(t, []byte{0x34, 0x12}, s.Bytes())
}

func TestSerializer_WriteU24(t *testing.T) {
	s := NewSerializer()
	s.WriteU24(0x123456)
	// Little-endian: 0x56, 0x34, 0x12
	assert.Equal(t, []byte{0x56, 0x34, 0x12}, s.Bytes())
}

func TestSerializer_WriteU32(t *testing.T) {
	s := NewSerializer()
	s.WriteU32(0x12345678)
	assert.Equal(t, []byte{0x78, 0x56, 0x34, 0x12}, s.Bytes())
}

func TestSerializer_WriteU64(t *testing.T) {
	s := NewSerializer()
	s.WriteU64(0x123456789ABCDEF0)
	assert.Equal(t, []byte{0xF0, 0xDE, 0xBC, 0x9A, 0x78, 0x56, 0x34, 0x12}, s.Bytes())
}

func TestSerializer_WriteF32(t *testing.T) {
	s := NewSerializer()
	s.WriteF32(240000000.0) // 240 MH/s - X100 hashrate
	assert.Equal(t, 4, s.Len())
}

func TestSerializer_WriteBool(t *testing.T) {
	s := NewSerializer()
	s.WriteBool(true)
	s.WriteBool(false)
	assert.Equal(t, []byte{0x01, 0x00}, s.Bytes())
}

func TestSerializer_WriteSTR0_255(t *testing.T) {
	s := NewSerializer()
	s.WriteSTR0_255("hello")
	assert.Equal(t, []byte{0x05, 'h', 'e', 'l', 'l', 'o'}, s.Bytes())
}

func TestSerializer_WriteFixedBytes(t *testing.T) {
	s := NewSerializer()
	s.WriteFixedBytes([]byte{0x01, 0x02}, 4)
	assert.Equal(t, []byte{0x01, 0x02, 0x00, 0x00}, s.Bytes())
}

func TestSerializer_Reset(t *testing.T) {
	s := NewSerializer()
	s.WriteU32(0x12345678)
	assert.Equal(t, 4, s.Len())
	s.Reset()
	assert.Equal(t, 0, s.Len())
}

// -----------------------------------------------------------------------------
// Deserializer Primitive Tests
// -----------------------------------------------------------------------------

func TestDeserializer_ReadU8(t *testing.T) {
	d := NewDeserializer([]byte{0x42})
	v, err := d.ReadU8()
	require.NoError(t, err)
	assert.Equal(t, uint8(0x42), v)
}

func TestDeserializer_ReadU16(t *testing.T) {
	d := NewDeserializer([]byte{0x34, 0x12})
	v, err := d.ReadU16()
	require.NoError(t, err)
	assert.Equal(t, uint16(0x1234), v)
}

func TestDeserializer_ReadU24(t *testing.T) {
	d := NewDeserializer([]byte{0x56, 0x34, 0x12})
	v, err := d.ReadU24()
	require.NoError(t, err)
	assert.Equal(t, uint32(0x123456), v)
}

func TestDeserializer_ReadU32(t *testing.T) {
	d := NewDeserializer([]byte{0x78, 0x56, 0x34, 0x12})
	v, err := d.ReadU32()
	require.NoError(t, err)
	assert.Equal(t, uint32(0x12345678), v)
}

func TestDeserializer_ReadU64(t *testing.T) {
	d := NewDeserializer([]byte{0xF0, 0xDE, 0xBC, 0x9A, 0x78, 0x56, 0x34, 0x12})
	v, err := d.ReadU64()
	require.NoError(t, err)
	assert.Equal(t, uint64(0x123456789ABCDEF0), v)
}

func TestDeserializer_ReadBool(t *testing.T) {
	d := NewDeserializer([]byte{0x01, 0x00})
	v1, err := d.ReadBool()
	require.NoError(t, err)
	assert.True(t, v1)

	v2, err := d.ReadBool()
	require.NoError(t, err)
	assert.False(t, v2)
}

func TestDeserializer_ReadSTR0_255(t *testing.T) {
	d := NewDeserializer([]byte{0x05, 'h', 'e', 'l', 'l', 'o'})
	v, err := d.ReadSTR0_255()
	require.NoError(t, err)
	assert.Equal(t, STR0_255("hello"), v)
}

func TestDeserializer_ReadHeader(t *testing.T) {
	d := NewDeserializer([]byte{0x01, 0x00, 0x20, 0x00, 0x01, 0x00})
	h, err := d.ReadHeader()
	require.NoError(t, err)
	assert.Equal(t, uint16(ExtensionTypeVersionRolling), h.ExtensionType)
	assert.Equal(t, uint8(MsgTypeNewMiningJob), h.MsgType)
	assert.Equal(t, uint32(256), h.MsgLength)
}

func TestDeserializer_EOF_Errors(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		fn   func(*Deserializer) error
	}{
		{"U8 empty", []byte{}, func(d *Deserializer) error { _, err := d.ReadU8(); return err }},
		{"U16 short", []byte{0x00}, func(d *Deserializer) error { _, err := d.ReadU16(); return err }},
		{"U24 short", []byte{0x00, 0x00}, func(d *Deserializer) error { _, err := d.ReadU24(); return err }},
		{"U32 short", []byte{0x00, 0x00, 0x00}, func(d *Deserializer) error { _, err := d.ReadU32(); return err }},
		{"U64 short", []byte{0x00, 0x00, 0x00, 0x00}, func(d *Deserializer) error { _, err := d.ReadU64(); return err }},
		{"STR0_255 short", []byte{0x05, 'a', 'b'}, func(d *Deserializer) error { _, err := d.ReadSTR0_255(); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDeserializer(tt.data)
			err := tt.fn(d)
			assert.Error(t, err)
		})
	}
}

// -----------------------------------------------------------------------------
// Message Round-Trip Tests (Serialize -> Deserialize)
// -----------------------------------------------------------------------------

func TestSetupConnection_RoundTrip(t *testing.T) {
	original := &SetupConnection{
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

	s := NewSerializer()
	payload := s.SerializeSetupConnection(original)

	d := NewDeserializer(payload)
	parsed, err := d.DeserializeSetupConnection()
	require.NoError(t, err)

	assert.Equal(t, original.Protocol, parsed.Protocol)
	assert.Equal(t, original.MinVersion, parsed.MinVersion)
	assert.Equal(t, original.MaxVersion, parsed.MaxVersion)
	assert.Equal(t, original.Flags, parsed.Flags)
	assert.Equal(t, original.Endpoint, parsed.Endpoint)
	assert.Equal(t, original.Vendor, parsed.Vendor)
	assert.Equal(t, original.HardwareVersion, parsed.HardwareVersion)
	assert.Equal(t, original.FirmwareVersion, parsed.FirmwareVersion)
	assert.Equal(t, original.DeviceID, parsed.DeviceID)
}

func TestSetupConnectionSuccess_RoundTrip(t *testing.T) {
	original := &SetupConnectionSuccess{
		UsedVersion: 2,
		Flags:       0x00000007,
	}

	s := NewSerializer()
	payload := s.SerializeSetupConnectionSuccess(original)

	d := NewDeserializer(payload)
	parsed, err := d.DeserializeSetupConnectionSuccess()
	require.NoError(t, err)

	assert.Equal(t, original.UsedVersion, parsed.UsedVersion)
	assert.Equal(t, original.Flags, parsed.Flags)
}

func TestSetupConnectionError_RoundTrip(t *testing.T) {
	original := &SetupConnectionError{
		Flags:     0x00000001,
		ErrorCode: "unsupported-protocol-version",
	}

	s := NewSerializer()
	payload := s.SerializeSetupConnectionError(original)

	d := NewDeserializer(payload)
	parsed, err := d.DeserializeSetupConnectionError()
	require.NoError(t, err)

	assert.Equal(t, original.Flags, parsed.Flags)
	assert.Equal(t, original.ErrorCode, parsed.ErrorCode)
}

func TestOpenStandardMiningChannel_RoundTrip(t *testing.T) {
	original := &OpenStandardMiningChannel{
		RequestID:         1,
		UserIdentity:      "kaspa:qr0123456789.worker1",
		NominalHashrate:   240000000, // 240 MH/s X100
		MaxTargetRequired: 0x1d00ffff,
	}

	s := NewSerializer()
	payload := s.SerializeOpenStandardMiningChannel(original)

	d := NewDeserializer(payload)
	parsed, err := d.DeserializeOpenStandardMiningChannel()
	require.NoError(t, err)

	assert.Equal(t, original.RequestID, parsed.RequestID)
	assert.Equal(t, original.UserIdentity, parsed.UserIdentity)
	assert.InDelta(t, original.NominalHashrate, parsed.NominalHashrate, 0.001)
	assert.Equal(t, original.MaxTargetRequired, parsed.MaxTargetRequired)
}

func TestOpenStandardMiningChannelSuccess_RoundTrip(t *testing.T) {
	original := &OpenStandardMiningChannelSuccess{
		RequestID:       1,
		ChannelID:       42,
		Target:          [32]byte{0x00, 0x00, 0xff, 0xff},
		ExtraNonce2Size: 4,
		GroupChannelID:  100,
	}

	s := NewSerializer()
	payload := s.SerializeOpenStandardMiningChannelSuccess(original)

	d := NewDeserializer(payload)
	parsed, err := d.DeserializeOpenStandardMiningChannelSuccess()
	require.NoError(t, err)

	assert.Equal(t, original.RequestID, parsed.RequestID)
	assert.Equal(t, original.ChannelID, parsed.ChannelID)
	assert.Equal(t, original.Target, parsed.Target)
	assert.Equal(t, original.ExtraNonce2Size, parsed.ExtraNonce2Size)
	assert.Equal(t, original.GroupChannelID, parsed.GroupChannelID)
}

func TestOpenStandardMiningChannelError_RoundTrip(t *testing.T) {
	original := &OpenStandardMiningChannelError{
		RequestID: 1,
		ErrorCode: "unauthorized",
	}

	s := NewSerializer()
	payload := s.SerializeOpenStandardMiningChannelError(original)

	d := NewDeserializer(payload)
	parsed, err := d.DeserializeOpenStandardMiningChannelError()
	require.NoError(t, err)

	assert.Equal(t, original.RequestID, parsed.RequestID)
	assert.Equal(t, original.ErrorCode, parsed.ErrorCode)
}

func TestNewMiningJob_RoundTrip(t *testing.T) {
	original := &NewMiningJob{
		ChannelID:      42,
		JobID:          1000,
		FuturePrevHash: false,
		Version:        0x20000000,
		VersionMask:    0x1fffe000,
	}

	s := NewSerializer()
	payload := s.SerializeNewMiningJob(original)

	d := NewDeserializer(payload)
	parsed, err := d.DeserializeNewMiningJob()
	require.NoError(t, err)

	assert.Equal(t, original.ChannelID, parsed.ChannelID)
	assert.Equal(t, original.JobID, parsed.JobID)
	assert.Equal(t, original.FuturePrevHash, parsed.FuturePrevHash)
	assert.Equal(t, original.Version, parsed.Version)
	assert.Equal(t, original.VersionMask, parsed.VersionMask)
}

func TestSetNewPrevHash_RoundTrip(t *testing.T) {
	original := &SetNewPrevHash{
		ChannelID: 42,
		JobID:     1000,
		PrevHash:  [32]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		MinNTime:  1703001600,
		NBits:     0x1d00ffff,
	}

	s := NewSerializer()
	payload := s.SerializeSetNewPrevHash(original)

	d := NewDeserializer(payload)
	parsed, err := d.DeserializeSetNewPrevHash()
	require.NoError(t, err)

	assert.Equal(t, original.ChannelID, parsed.ChannelID)
	assert.Equal(t, original.JobID, parsed.JobID)
	assert.Equal(t, original.PrevHash, parsed.PrevHash)
	assert.Equal(t, original.MinNTime, parsed.MinNTime)
	assert.Equal(t, original.NBits, parsed.NBits)
}

func TestSubmitSharesStandard_RoundTrip(t *testing.T) {
	original := &SubmitSharesStandard{
		ChannelID:   42,
		SequenceNum: 1,
		JobID:       1000,
		Nonce:       0x12345678,
		NTime:       1703001600,
		Version:     0x20000000,
	}

	s := NewSerializer()
	payload := s.SerializeSubmitSharesStandard(original)

	d := NewDeserializer(payload)
	parsed, err := d.DeserializeSubmitSharesStandard()
	require.NoError(t, err)

	assert.Equal(t, original.ChannelID, parsed.ChannelID)
	assert.Equal(t, original.SequenceNum, parsed.SequenceNum)
	assert.Equal(t, original.JobID, parsed.JobID)
	assert.Equal(t, original.Nonce, parsed.Nonce)
	assert.Equal(t, original.NTime, parsed.NTime)
	assert.Equal(t, original.Version, parsed.Version)
}

func TestSubmitSharesSuccess_RoundTrip(t *testing.T) {
	original := &SubmitSharesSuccess{
		ChannelID:       42,
		LastSequenceNum: 10,
		NewSubmits:      5,
		NewDifficulty:   65536,
	}

	s := NewSerializer()
	payload := s.SerializeSubmitSharesSuccess(original)

	d := NewDeserializer(payload)
	parsed, err := d.DeserializeSubmitSharesSuccess()
	require.NoError(t, err)

	assert.Equal(t, original.ChannelID, parsed.ChannelID)
	assert.Equal(t, original.LastSequenceNum, parsed.LastSequenceNum)
	assert.Equal(t, original.NewSubmits, parsed.NewSubmits)
	assert.Equal(t, original.NewDifficulty, parsed.NewDifficulty)
}

func TestSubmitSharesError_RoundTrip(t *testing.T) {
	original := &SubmitSharesError{
		ChannelID:   42,
		SequenceNum: 5,
		ErrorCode:   "stale-share",
	}

	s := NewSerializer()
	payload := s.SerializeSubmitSharesError(original)

	d := NewDeserializer(payload)
	parsed, err := d.DeserializeSubmitSharesError()
	require.NoError(t, err)

	assert.Equal(t, original.ChannelID, parsed.ChannelID)
	assert.Equal(t, original.SequenceNum, parsed.SequenceNum)
	assert.Equal(t, original.ErrorCode, parsed.ErrorCode)
}

func TestSetTarget_RoundTrip(t *testing.T) {
	original := &SetTarget{
		ChannelID: 42,
		MaxTarget: [32]byte{0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff},
	}

	s := NewSerializer()
	payload := s.SerializeSetTarget(original)

	d := NewDeserializer(payload)
	parsed, err := d.DeserializeSetTarget()
	require.NoError(t, err)

	assert.Equal(t, original.ChannelID, parsed.ChannelID)
	assert.Equal(t, original.MaxTarget, parsed.MaxTarget)
}

func TestReconnect_RoundTrip(t *testing.T) {
	original := &Reconnect{
		NewHost: "backup.pool.chimera.io",
		NewPort: 3335,
	}

	s := NewSerializer()
	payload := s.SerializeReconnect(original)

	d := NewDeserializer(payload)
	parsed, err := d.DeserializeReconnect()
	require.NoError(t, err)

	assert.Equal(t, original.NewHost, parsed.NewHost)
	assert.Equal(t, original.NewPort, parsed.NewPort)
}

// -----------------------------------------------------------------------------
// Frame Serialization Tests
// -----------------------------------------------------------------------------

func TestSerializeFrame(t *testing.T) {
	s := NewSerializer()
	payload := []byte{0x01, 0x02, 0x03, 0x04}

	frame := s.SerializeFrame(MsgTypeSetupConnection, ExtensionTypeNone, payload)

	// Should be header (6 bytes) + payload (4 bytes)
	assert.Equal(t, 10, len(frame))

	// Parse header
	d := NewDeserializer(frame)
	h, err := d.ReadHeader()
	require.NoError(t, err)

	assert.Equal(t, uint16(ExtensionTypeNone), h.ExtensionType)
	assert.Equal(t, uint8(MsgTypeSetupConnection), h.MsgType)
	assert.Equal(t, uint32(4), h.MsgLength)

	// Read payload
	payloadRead, err := d.ReadBytes(int(h.MsgLength))
	require.NoError(t, err)
	assert.Equal(t, payload, payloadRead)
}

// -----------------------------------------------------------------------------
// Performance Benchmarks
// -----------------------------------------------------------------------------

func BenchmarkSerializer_SubmitShares(b *testing.B) {
	s := NewSerializer()
	msg := &SubmitSharesStandard{
		ChannelID:   42,
		SequenceNum: 1,
		JobID:       1000,
		Nonce:       0x12345678,
		NTime:       1703001600,
		Version:     0x20000000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.SerializeSubmitSharesStandard(msg)
	}
}

func BenchmarkDeserializer_SubmitShares(b *testing.B) {
	s := NewSerializer()
	msg := &SubmitSharesStandard{
		ChannelID:   42,
		SequenceNum: 1,
		JobID:       1000,
		Nonce:       0x12345678,
		NTime:       1703001600,
		Version:     0x20000000,
	}
	payload := s.SerializeSubmitSharesStandard(msg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d := NewDeserializer(payload)
		d.DeserializeSubmitSharesStandard()
	}
}

func BenchmarkSerializer_NewMiningJob(b *testing.B) {
	s := NewSerializer()
	msg := &NewMiningJob{
		ChannelID:      42,
		JobID:          1000,
		FuturePrevHash: false,
		Version:        0x20000000,
		VersionMask:    0x1fffe000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.SerializeNewMiningJob(msg)
	}
}

func BenchmarkFullFrame_Serialization(b *testing.B) {
	s := NewSerializer()
	msg := &SubmitSharesStandard{
		ChannelID:   42,
		SequenceNum: 1,
		JobID:       1000,
		Nonce:       0x12345678,
		NTime:       1703001600,
		Version:     0x20000000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		payload := s.SerializeSubmitSharesStandard(msg)
		s.SerializeFrame(MsgTypeSubmitSharesStandard, ExtensionTypeNone, payload)
	}
}
