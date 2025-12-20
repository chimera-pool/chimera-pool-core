package binary

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
)

// =============================================================================
// STRATUM V2 MESSAGE SERIALIZER
// High-performance binary serialization for lowest latency
// =============================================================================

// Serializer handles efficient binary message serialization
type Serializer struct {
	buf *bytes.Buffer
}

// NewSerializer creates a new serializer with pre-allocated buffer
func NewSerializer() *Serializer {
	return &Serializer{
		buf: bytes.NewBuffer(make([]byte, 0, 1024)),
	}
}

// Reset resets the buffer for reuse (zero allocation pattern)
func (s *Serializer) Reset() {
	s.buf.Reset()
}

// Bytes returns the serialized bytes
func (s *Serializer) Bytes() []byte {
	return s.buf.Bytes()
}

// Len returns the current length
func (s *Serializer) Len() int {
	return s.buf.Len()
}

// -----------------------------------------------------------------------------
// Primitive Writers
// -----------------------------------------------------------------------------

// WriteU8 writes a uint8
func (s *Serializer) WriteU8(v uint8) {
	s.buf.WriteByte(v)
}

// WriteU16 writes a uint16 in little-endian
func (s *Serializer) WriteU16(v uint16) {
	var b [2]byte
	binary.LittleEndian.PutUint16(b[:], v)
	s.buf.Write(b[:])
}

// WriteU24 writes a uint32 as 24-bit little-endian
func (s *Serializer) WriteU24(v uint32) {
	s.buf.WriteByte(byte(v & 0xFF))
	s.buf.WriteByte(byte((v >> 8) & 0xFF))
	s.buf.WriteByte(byte((v >> 16) & 0xFF))
}

// WriteU32 writes a uint32 in little-endian
func (s *Serializer) WriteU32(v uint32) {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	s.buf.Write(b[:])
}

// WriteU64 writes a uint64 in little-endian
func (s *Serializer) WriteU64(v uint64) {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], v)
	s.buf.Write(b[:])
}

// WriteF32 writes a float32 in IEEE 754
func (s *Serializer) WriteF32(v float32) {
	s.WriteU32(math.Float32bits(v))
}

// WriteBool writes a boolean as u8
func (s *Serializer) WriteBool(v bool) {
	if v {
		s.buf.WriteByte(1)
	} else {
		s.buf.WriteByte(0)
	}
}

// WriteBytes writes raw bytes
func (s *Serializer) WriteBytes(b []byte) {
	s.buf.Write(b)
}

// WriteFixedBytes writes exactly n bytes, padding with zeros if needed
func (s *Serializer) WriteFixedBytes(b []byte, n int) {
	if len(b) >= n {
		s.buf.Write(b[:n])
	} else {
		s.buf.Write(b)
		// Pad with zeros
		for i := len(b); i < n; i++ {
			s.buf.WriteByte(0)
		}
	}
}

// WriteSTR0_255 writes a length-prefixed string (max 255 bytes)
func (s *Serializer) WriteSTR0_255(str string) {
	if len(str) > 255 {
		str = str[:255]
	}
	s.buf.WriteByte(byte(len(str)))
	s.buf.WriteString(str)
}

// WriteHeader writes a frame header
func (s *Serializer) WriteHeader(h *FrameHeader) {
	s.WriteU16(h.ExtensionType)
	s.WriteU8(h.MsgType)
	s.WriteU24(h.MsgLength)
}

// -----------------------------------------------------------------------------
// Message Serializers
// -----------------------------------------------------------------------------

// SerializeSetupConnection serializes a SetupConnection message
func (s *Serializer) SerializeSetupConnection(msg *SetupConnection) []byte {
	s.Reset()

	s.WriteU8(msg.Protocol)
	s.WriteU16(msg.MinVersion)
	s.WriteU16(msg.MaxVersion)
	s.WriteU32(msg.Flags)
	s.WriteSTR0_255(string(msg.Endpoint))
	s.WriteSTR0_255(string(msg.Vendor))
	s.WriteSTR0_255(string(msg.HardwareVersion))
	s.WriteSTR0_255(string(msg.FirmwareVersion))
	s.WriteSTR0_255(string(msg.DeviceID))

	return s.Bytes()
}

// SerializeSetupConnectionSuccess serializes a SetupConnectionSuccess message
func (s *Serializer) SerializeSetupConnectionSuccess(msg *SetupConnectionSuccess) []byte {
	s.Reset()

	s.WriteU16(msg.UsedVersion)
	s.WriteU32(msg.Flags)

	return s.Bytes()
}

// SerializeSetupConnectionError serializes a SetupConnectionError message
func (s *Serializer) SerializeSetupConnectionError(msg *SetupConnectionError) []byte {
	s.Reset()

	s.WriteU32(msg.Flags)
	s.WriteSTR0_255(string(msg.ErrorCode))

	return s.Bytes()
}

// SerializeOpenStandardMiningChannel serializes an OpenStandardMiningChannel message
func (s *Serializer) SerializeOpenStandardMiningChannel(msg *OpenStandardMiningChannel) []byte {
	s.Reset()

	s.WriteU32(msg.RequestID)
	s.WriteSTR0_255(string(msg.UserIdentity))
	s.WriteF32(msg.NominalHashrate)
	s.WriteU32(msg.MaxTargetRequired)

	return s.Bytes()
}

// SerializeOpenStandardMiningChannelSuccess serializes success response
func (s *Serializer) SerializeOpenStandardMiningChannelSuccess(msg *OpenStandardMiningChannelSuccess) []byte {
	s.Reset()

	s.WriteU32(msg.RequestID)
	s.WriteU32(msg.ChannelID)
	s.WriteFixedBytes(msg.Target[:], 32)
	s.WriteU16(msg.ExtraNonce2Size)
	s.WriteU32(msg.GroupChannelID)

	return s.Bytes()
}

// SerializeOpenStandardMiningChannelError serializes error response
func (s *Serializer) SerializeOpenStandardMiningChannelError(msg *OpenStandardMiningChannelError) []byte {
	s.Reset()

	s.WriteU32(msg.RequestID)
	s.WriteSTR0_255(string(msg.ErrorCode))

	return s.Bytes()
}

// SerializeNewMiningJob serializes a NewMiningJob message
func (s *Serializer) SerializeNewMiningJob(msg *NewMiningJob) []byte {
	s.Reset()

	s.WriteU32(msg.ChannelID)
	s.WriteU32(msg.JobID)
	s.WriteBool(msg.FuturePrevHash)
	s.WriteU32(msg.Version)
	s.WriteU32(msg.VersionMask)

	return s.Bytes()
}

// SerializeSetNewPrevHash serializes a SetNewPrevHash message
func (s *Serializer) SerializeSetNewPrevHash(msg *SetNewPrevHash) []byte {
	s.Reset()

	s.WriteU32(msg.ChannelID)
	s.WriteU32(msg.JobID)
	s.WriteFixedBytes(msg.PrevHash[:], 32)
	s.WriteU32(msg.MinNTime)
	s.WriteU32(msg.NBits)

	return s.Bytes()
}

// SerializeSubmitSharesStandard serializes a SubmitSharesStandard message
func (s *Serializer) SerializeSubmitSharesStandard(msg *SubmitSharesStandard) []byte {
	s.Reset()

	s.WriteU32(msg.ChannelID)
	s.WriteU32(msg.SequenceNum)
	s.WriteU32(msg.JobID)
	s.WriteU32(msg.Nonce)
	s.WriteU32(msg.NTime)
	s.WriteU32(msg.Version)

	return s.Bytes()
}

// SerializeSubmitSharesSuccess serializes a SubmitSharesSuccess message
func (s *Serializer) SerializeSubmitSharesSuccess(msg *SubmitSharesSuccess) []byte {
	s.Reset()

	s.WriteU32(msg.ChannelID)
	s.WriteU32(msg.LastSequenceNum)
	s.WriteU32(msg.NewSubmits)
	s.WriteU64(msg.NewDifficulty)

	return s.Bytes()
}

// SerializeSubmitSharesError serializes a SubmitSharesError message
func (s *Serializer) SerializeSubmitSharesError(msg *SubmitSharesError) []byte {
	s.Reset()

	s.WriteU32(msg.ChannelID)
	s.WriteU32(msg.SequenceNum)
	s.WriteSTR0_255(string(msg.ErrorCode))

	return s.Bytes()
}

// SerializeSetTarget serializes a SetTarget message
func (s *Serializer) SerializeSetTarget(msg *SetTarget) []byte {
	s.Reset()

	s.WriteU32(msg.ChannelID)
	s.WriteFixedBytes(msg.MaxTarget[:], 32)

	return s.Bytes()
}

// SerializeReconnect serializes a Reconnect message
func (s *Serializer) SerializeReconnect(msg *Reconnect) []byte {
	s.Reset()

	s.WriteSTR0_255(string(msg.NewHost))
	s.WriteU16(msg.NewPort)

	return s.Bytes()
}

// -----------------------------------------------------------------------------
// Full Frame Serialization (Header + Payload)
// -----------------------------------------------------------------------------

// SerializeFrame creates a complete frame with header and payload
func (s *Serializer) SerializeFrame(msgType uint8, extensionType uint16, payload []byte) []byte {
	header := &FrameHeader{
		ExtensionType: extensionType,
		MsgType:       msgType,
		MsgLength:     uint32(len(payload)),
	}

	result := make([]byte, HeaderSize+len(payload))
	copy(result[:HeaderSize], header.Serialize())
	copy(result[HeaderSize:], payload)

	return result
}

// =============================================================================
// DESERIALIZER
// =============================================================================

// Deserializer handles efficient binary message deserialization
type Deserializer struct {
	data []byte
	pos  int
}

// NewDeserializer creates a new deserializer
func NewDeserializer(data []byte) *Deserializer {
	return &Deserializer{
		data: data,
		pos:  0,
	}
}

// Remaining returns bytes remaining
func (d *Deserializer) Remaining() int {
	return len(d.data) - d.pos
}

// Position returns current position
func (d *Deserializer) Position() int {
	return d.pos
}

// -----------------------------------------------------------------------------
// Primitive Readers
// -----------------------------------------------------------------------------

// ReadU8 reads a uint8
func (d *Deserializer) ReadU8() (uint8, error) {
	if d.Remaining() < 1 {
		return 0, io.ErrUnexpectedEOF
	}
	v := d.data[d.pos]
	d.pos++
	return v, nil
}

// ReadU16 reads a uint16 in little-endian
func (d *Deserializer) ReadU16() (uint16, error) {
	if d.Remaining() < 2 {
		return 0, io.ErrUnexpectedEOF
	}
	v := binary.LittleEndian.Uint16(d.data[d.pos:])
	d.pos += 2
	return v, nil
}

// ReadU24 reads a 24-bit uint as uint32
func (d *Deserializer) ReadU24() (uint32, error) {
	if d.Remaining() < 3 {
		return 0, io.ErrUnexpectedEOF
	}
	v := uint32(d.data[d.pos]) | uint32(d.data[d.pos+1])<<8 | uint32(d.data[d.pos+2])<<16
	d.pos += 3
	return v, nil
}

// ReadU32 reads a uint32 in little-endian
func (d *Deserializer) ReadU32() (uint32, error) {
	if d.Remaining() < 4 {
		return 0, io.ErrUnexpectedEOF
	}
	v := binary.LittleEndian.Uint32(d.data[d.pos:])
	d.pos += 4
	return v, nil
}

// ReadU64 reads a uint64 in little-endian
func (d *Deserializer) ReadU64() (uint64, error) {
	if d.Remaining() < 8 {
		return 0, io.ErrUnexpectedEOF
	}
	v := binary.LittleEndian.Uint64(d.data[d.pos:])
	d.pos += 8
	return v, nil
}

// ReadF32 reads a float32 in IEEE 754
func (d *Deserializer) ReadF32() (float32, error) {
	bits, err := d.ReadU32()
	if err != nil {
		return 0, err
	}
	return math.Float32frombits(bits), nil
}

// ReadBool reads a boolean
func (d *Deserializer) ReadBool() (bool, error) {
	v, err := d.ReadU8()
	if err != nil {
		return false, err
	}
	return v != 0, nil
}

// ReadBytes reads n bytes
func (d *Deserializer) ReadBytes(n int) ([]byte, error) {
	if d.Remaining() < n {
		return nil, io.ErrUnexpectedEOF
	}
	v := make([]byte, n)
	copy(v, d.data[d.pos:d.pos+n])
	d.pos += n
	return v, nil
}

// ReadFixedBytes reads exactly n bytes into a fixed array
func (d *Deserializer) ReadFixedBytes32() ([32]byte, error) {
	var v [32]byte
	if d.Remaining() < 32 {
		return v, io.ErrUnexpectedEOF
	}
	copy(v[:], d.data[d.pos:d.pos+32])
	d.pos += 32
	return v, nil
}

// ReadSTR0_255 reads a length-prefixed string
func (d *Deserializer) ReadSTR0_255() (STR0_255, error) {
	length, err := d.ReadU8()
	if err != nil {
		return "", err
	}
	if d.Remaining() < int(length) {
		return "", io.ErrUnexpectedEOF
	}
	v := string(d.data[d.pos : d.pos+int(length)])
	d.pos += int(length)
	return STR0_255(v), nil
}

// ReadHeader reads a frame header
func (d *Deserializer) ReadHeader() (*FrameHeader, error) {
	extType, err := d.ReadU16()
	if err != nil {
		return nil, err
	}
	msgType, err := d.ReadU8()
	if err != nil {
		return nil, err
	}
	msgLen, err := d.ReadU24()
	if err != nil {
		return nil, err
	}

	return &FrameHeader{
		ExtensionType: extType,
		MsgType:       msgType,
		MsgLength:     msgLen,
	}, nil
}

// -----------------------------------------------------------------------------
// Message Deserializers
// -----------------------------------------------------------------------------

// DeserializeSetupConnection deserializes a SetupConnection message
func (d *Deserializer) DeserializeSetupConnection() (*SetupConnection, error) {
	msg := &SetupConnection{}
	var err error

	if msg.Protocol, err = d.ReadU8(); err != nil {
		return nil, err
	}
	if msg.MinVersion, err = d.ReadU16(); err != nil {
		return nil, err
	}
	if msg.MaxVersion, err = d.ReadU16(); err != nil {
		return nil, err
	}
	if msg.Flags, err = d.ReadU32(); err != nil {
		return nil, err
	}
	if msg.Endpoint, err = d.ReadSTR0_255(); err != nil {
		return nil, err
	}
	if msg.Vendor, err = d.ReadSTR0_255(); err != nil {
		return nil, err
	}
	if msg.HardwareVersion, err = d.ReadSTR0_255(); err != nil {
		return nil, err
	}
	if msg.FirmwareVersion, err = d.ReadSTR0_255(); err != nil {
		return nil, err
	}
	if msg.DeviceID, err = d.ReadSTR0_255(); err != nil {
		return nil, err
	}

	return msg, nil
}

// DeserializeSetupConnectionSuccess deserializes a SetupConnectionSuccess message
func (d *Deserializer) DeserializeSetupConnectionSuccess() (*SetupConnectionSuccess, error) {
	msg := &SetupConnectionSuccess{}
	var err error

	if msg.UsedVersion, err = d.ReadU16(); err != nil {
		return nil, err
	}
	if msg.Flags, err = d.ReadU32(); err != nil {
		return nil, err
	}

	return msg, nil
}

// DeserializeSetupConnectionError deserializes a SetupConnectionError message
func (d *Deserializer) DeserializeSetupConnectionError() (*SetupConnectionError, error) {
	msg := &SetupConnectionError{}
	var err error

	if msg.Flags, err = d.ReadU32(); err != nil {
		return nil, err
	}
	if msg.ErrorCode, err = d.ReadSTR0_255(); err != nil {
		return nil, err
	}

	return msg, nil
}

// DeserializeOpenStandardMiningChannel deserializes an OpenStandardMiningChannel message
func (d *Deserializer) DeserializeOpenStandardMiningChannel() (*OpenStandardMiningChannel, error) {
	msg := &OpenStandardMiningChannel{}
	var err error

	if msg.RequestID, err = d.ReadU32(); err != nil {
		return nil, err
	}
	if msg.UserIdentity, err = d.ReadSTR0_255(); err != nil {
		return nil, err
	}
	if msg.NominalHashrate, err = d.ReadF32(); err != nil {
		return nil, err
	}
	if msg.MaxTargetRequired, err = d.ReadU32(); err != nil {
		return nil, err
	}

	return msg, nil
}

// DeserializeOpenStandardMiningChannelSuccess deserializes success response
func (d *Deserializer) DeserializeOpenStandardMiningChannelSuccess() (*OpenStandardMiningChannelSuccess, error) {
	msg := &OpenStandardMiningChannelSuccess{}
	var err error

	if msg.RequestID, err = d.ReadU32(); err != nil {
		return nil, err
	}
	if msg.ChannelID, err = d.ReadU32(); err != nil {
		return nil, err
	}
	if msg.Target, err = d.ReadFixedBytes32(); err != nil {
		return nil, err
	}
	if msg.ExtraNonce2Size, err = d.ReadU16(); err != nil {
		return nil, err
	}
	if msg.GroupChannelID, err = d.ReadU32(); err != nil {
		return nil, err
	}

	return msg, nil
}

// DeserializeOpenStandardMiningChannelError deserializes error response
func (d *Deserializer) DeserializeOpenStandardMiningChannelError() (*OpenStandardMiningChannelError, error) {
	msg := &OpenStandardMiningChannelError{}
	var err error

	if msg.RequestID, err = d.ReadU32(); err != nil {
		return nil, err
	}
	if msg.ErrorCode, err = d.ReadSTR0_255(); err != nil {
		return nil, err
	}

	return msg, nil
}

// DeserializeNewMiningJob deserializes a NewMiningJob message
func (d *Deserializer) DeserializeNewMiningJob() (*NewMiningJob, error) {
	msg := &NewMiningJob{}
	var err error

	if msg.ChannelID, err = d.ReadU32(); err != nil {
		return nil, err
	}
	if msg.JobID, err = d.ReadU32(); err != nil {
		return nil, err
	}
	if msg.FuturePrevHash, err = d.ReadBool(); err != nil {
		return nil, err
	}
	if msg.Version, err = d.ReadU32(); err != nil {
		return nil, err
	}
	if msg.VersionMask, err = d.ReadU32(); err != nil {
		return nil, err
	}

	return msg, nil
}

// DeserializeSetNewPrevHash deserializes a SetNewPrevHash message
func (d *Deserializer) DeserializeSetNewPrevHash() (*SetNewPrevHash, error) {
	msg := &SetNewPrevHash{}
	var err error

	if msg.ChannelID, err = d.ReadU32(); err != nil {
		return nil, err
	}
	if msg.JobID, err = d.ReadU32(); err != nil {
		return nil, err
	}
	if msg.PrevHash, err = d.ReadFixedBytes32(); err != nil {
		return nil, err
	}
	if msg.MinNTime, err = d.ReadU32(); err != nil {
		return nil, err
	}
	if msg.NBits, err = d.ReadU32(); err != nil {
		return nil, err
	}

	return msg, nil
}

// DeserializeSubmitSharesStandard deserializes a SubmitSharesStandard message
func (d *Deserializer) DeserializeSubmitSharesStandard() (*SubmitSharesStandard, error) {
	msg := &SubmitSharesStandard{}
	var err error

	if msg.ChannelID, err = d.ReadU32(); err != nil {
		return nil, err
	}
	if msg.SequenceNum, err = d.ReadU32(); err != nil {
		return nil, err
	}
	if msg.JobID, err = d.ReadU32(); err != nil {
		return nil, err
	}
	if msg.Nonce, err = d.ReadU32(); err != nil {
		return nil, err
	}
	if msg.NTime, err = d.ReadU32(); err != nil {
		return nil, err
	}
	if msg.Version, err = d.ReadU32(); err != nil {
		return nil, err
	}

	return msg, nil
}

// DeserializeSubmitSharesSuccess deserializes a SubmitSharesSuccess message
func (d *Deserializer) DeserializeSubmitSharesSuccess() (*SubmitSharesSuccess, error) {
	msg := &SubmitSharesSuccess{}
	var err error

	if msg.ChannelID, err = d.ReadU32(); err != nil {
		return nil, err
	}
	if msg.LastSequenceNum, err = d.ReadU32(); err != nil {
		return nil, err
	}
	if msg.NewSubmits, err = d.ReadU32(); err != nil {
		return nil, err
	}
	if msg.NewDifficulty, err = d.ReadU64(); err != nil {
		return nil, err
	}

	return msg, nil
}

// DeserializeSubmitSharesError deserializes a SubmitSharesError message
func (d *Deserializer) DeserializeSubmitSharesError() (*SubmitSharesError, error) {
	msg := &SubmitSharesError{}
	var err error

	if msg.ChannelID, err = d.ReadU32(); err != nil {
		return nil, err
	}
	if msg.SequenceNum, err = d.ReadU32(); err != nil {
		return nil, err
	}
	if msg.ErrorCode, err = d.ReadSTR0_255(); err != nil {
		return nil, err
	}

	return msg, nil
}

// DeserializeSetTarget deserializes a SetTarget message
func (d *Deserializer) DeserializeSetTarget() (*SetTarget, error) {
	msg := &SetTarget{}
	var err error

	if msg.ChannelID, err = d.ReadU32(); err != nil {
		return nil, err
	}
	if msg.MaxTarget, err = d.ReadFixedBytes32(); err != nil {
		return nil, err
	}

	return msg, nil
}

// DeserializeReconnect deserializes a Reconnect message
func (d *Deserializer) DeserializeReconnect() (*Reconnect, error) {
	msg := &Reconnect{}
	var err error

	if msg.NewHost, err = d.ReadSTR0_255(); err != nil {
		return nil, err
	}
	if msg.NewPort, err = d.ReadU16(); err != nil {
		return nil, err
	}

	return msg, nil
}
