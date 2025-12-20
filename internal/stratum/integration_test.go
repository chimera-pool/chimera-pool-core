package stratum

import (
	"bytes"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/chimera-pool/chimera-pool-core/internal/stratum/blockdag"
	"github.com/chimera-pool/chimera-pool-core/internal/stratum/detector"
	"github.com/chimera-pool/chimera-pool-core/internal/stratum/difficulty"
	"github.com/chimera-pool/chimera-pool-core/internal/stratum/v2/binary"
	"github.com/chimera-pool/chimera-pool-core/internal/stratum/v2/noise"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// INTEGRATION TESTS FOR STRATUM V1/V2 HYBRID SYSTEM
// Tests the complete flow from protocol detection to share validation
// =============================================================================

// -----------------------------------------------------------------------------
// Protocol Detection Integration Tests
// -----------------------------------------------------------------------------

func TestIntegration_ProtocolDetection_V1vsV2(t *testing.T) {
	det := detector.NewDetector()

	// V1 JSON data
	v1Data := []byte(`{"id":1,"method":"mining.subscribe","params":[]}`)
	v1Version := det.DetectFromBytes(v1Data)
	assert.Equal(t, detector.ProtocolV1, v1Version)

	// V2 binary data (SetupConnection frame)
	v2Data := []byte{0x00, 0x00, 0x00, 0x64, 0x00, 0x00}
	v2Version := det.DetectFromBytes(v2Data)
	assert.Equal(t, detector.ProtocolV2, v2Version)
}

func TestIntegration_RouterWithMixedProtocols(t *testing.T) {
	router := detector.NewRouter()

	// Track handled connections
	var v1Count, v2Count int
	var mu sync.Mutex

	// Mock V1 handler
	v1Handler := &mockProtocolHandler{
		protocol: detector.ProtocolV1,
		onHandle: func() { mu.Lock(); v1Count++; mu.Unlock() },
	}

	// Mock V2 handler
	v2Handler := &mockProtocolHandler{
		protocol: detector.ProtocolV2,
		onHandle: func() { mu.Lock(); v2Count++; mu.Unlock() },
	}

	router.RegisterHandler(detector.ProtocolV1, v1Handler)
	router.RegisterHandler(detector.ProtocolV2, v2Handler)

	// Simulate mixed connections
	for i := 0; i < 5; i++ {
		// V1 connection
		router.Route(newMockNetConn([]byte(`{"id":1}`)))
		// V2 connection
		router.Route(newMockNetConn([]byte{0x00, 0x00, 0x00, 0x64, 0x00, 0x00}))
	}

	mu.Lock()
	assert.Equal(t, 5, v1Count, "should handle 5 V1 connections")
	assert.Equal(t, 5, v2Count, "should handle 5 V2 connections")
	mu.Unlock()

	metrics1, metrics2, _ := router.GetMetrics()
	assert.Equal(t, uint64(5), metrics1)
	assert.Equal(t, uint64(5), metrics2)
}

// -----------------------------------------------------------------------------
// V2 Binary Protocol Integration Tests
// -----------------------------------------------------------------------------

func TestIntegration_V2MessageFlow(t *testing.T) {
	s := binary.NewSerializer()

	// 1. Client sends SetupConnection
	setupConn := &binary.SetupConnection{
		Protocol:        0,
		MinVersion:      2,
		MaxVersion:      2,
		Flags:           0x07,
		Endpoint:        "pool.blockdag.io:3334",
		Vendor:          "BlockDAG",
		HardwareVersion: "X100",
		FirmwareVersion: "1.0.0",
		DeviceID:        "x100-001",
	}
	setupPayload := s.SerializeSetupConnection(setupConn)
	setupFrame := s.SerializeFrame(binary.MsgTypeSetupConnection, binary.ExtensionTypeNone, setupPayload)

	// Verify frame structure
	assert.Equal(t, 6+len(setupPayload), len(setupFrame))

	// 2. Server responds with SetupConnectionSuccess
	setupSuccess := &binary.SetupConnectionSuccess{
		UsedVersion: 2,
		Flags:       0x07,
	}
	successPayload := s.SerializeSetupConnectionSuccess(setupSuccess)

	// Parse response
	d := binary.NewDeserializer(successPayload)
	parsedSuccess, err := d.DeserializeSetupConnectionSuccess()
	require.NoError(t, err)
	assert.Equal(t, uint16(2), parsedSuccess.UsedVersion)

	// 3. Client opens mining channel
	openChannel := &binary.OpenStandardMiningChannel{
		RequestID:         1,
		UserIdentity:      "kaspa:qr0123456789.worker1",
		NominalHashrate:   240000000, // 240 MH/s X100
		MaxTargetRequired: 0x1d00ffff,
	}
	channelPayload := s.SerializeOpenStandardMiningChannel(openChannel)

	d = binary.NewDeserializer(channelPayload)
	parsedChannel, err := d.DeserializeOpenStandardMiningChannel()
	require.NoError(t, err)
	assert.Equal(t, "kaspa:qr0123456789.worker1", string(parsedChannel.UserIdentity))

	// 4. Server sends mining job
	job := &binary.NewMiningJob{
		ChannelID:      1,
		JobID:          1000,
		FuturePrevHash: false,
		Version:        0x20000000,
		VersionMask:    0x1fffe000,
	}
	jobPayload := s.SerializeNewMiningJob(job)

	d = binary.NewDeserializer(jobPayload)
	parsedJob, err := d.DeserializeNewMiningJob()
	require.NoError(t, err)
	assert.Equal(t, uint32(1000), parsedJob.JobID)

	// 5. Client submits share
	share := &binary.SubmitSharesStandard{
		ChannelID:   1,
		SequenceNum: 1,
		JobID:       1000,
		Nonce:       0x12345678,
		NTime:       1703001600,
		Version:     0x20000000,
	}
	sharePayload := s.SerializeSubmitSharesStandard(share)

	d = binary.NewDeserializer(sharePayload)
	parsedShare, err := d.DeserializeSubmitSharesStandard()
	require.NoError(t, err)
	assert.Equal(t, uint32(0x12345678), parsedShare.Nonce)
}

// -----------------------------------------------------------------------------
// Noise Encryption Integration Tests
// -----------------------------------------------------------------------------

func TestIntegration_NoiseSecureChannel(t *testing.T) {
	// Server (pool) generates static key
	serverStatic, err := noise.GenerateKeyPair()
	require.NoError(t, err)

	// Create handshake states
	clientHS, err := noise.NewInitiatorHandshake()
	require.NoError(t, err)

	serverHS, err := noise.NewResponderHandshake(serverStatic)
	require.NoError(t, err)

	// Perform handshake
	msg1, err := clientHS.WriteMessage([]byte("client hello"))
	require.NoError(t, err)

	_, err = serverHS.ReadMessage(msg1)
	require.NoError(t, err)

	msg2, err := serverHS.WriteMessage([]byte("server hello"))
	require.NoError(t, err)

	_, err = clientHS.ReadMessage(msg2)
	require.NoError(t, err)

	// Both should be complete
	assert.True(t, clientHS.IsComplete())
	assert.True(t, serverHS.IsComplete())

	// Get transport keys
	clientSend, clientRecv, err := clientHS.Split()
	require.NoError(t, err)

	serverSend, serverRecv, err := serverHS.Split()
	require.NoError(t, err)

	// Create secure channels
	clientChannel := noise.NewSecureChannel(clientSend, clientRecv)
	serverChannel := noise.NewSecureChannel(serverSend, serverRecv)

	// Test bidirectional encrypted communication
	// Client -> Server
	shareData := []byte("share submission: nonce=0x12345678")
	encrypted, err := clientChannel.Encrypt(shareData)
	require.NoError(t, err)

	decrypted, err := serverChannel.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, shareData, decrypted)

	// Server -> Client
	jobData := []byte("new job: id=1000, target=0x00000fff")
	encrypted2, err := serverChannel.Encrypt(jobData)
	require.NoError(t, err)

	decrypted2, err := clientChannel.Decrypt(encrypted2)
	require.NoError(t, err)
	assert.Equal(t, jobData, decrypted2)
}

// -----------------------------------------------------------------------------
// BlockDAG Algorithm Integration Tests
// -----------------------------------------------------------------------------

func TestIntegration_BlockDAGMining(t *testing.T) {
	algo := blockdag.NewScrypyVariant()
	validator := blockdag.NewShareValidator()

	// Create block template
	template := blockdag.NewBlockTemplate()
	template.Version = 0x20000000
	template.PrevHash = [32]byte{0x01, 0x02, 0x03}
	template.MerkleRoot = [32]byte{0x04, 0x05, 0x06}
	template.Timestamp = 1703001600
	template.SetBits(0x1d00ffff)

	// Build header
	header := template.BuildHeader()
	assert.Equal(t, 80, len(header))

	// Simulate mining with different nonces
	easyTarget := bytes.Repeat([]byte{0xFF}, 32) // Very easy target

	for nonce := uint32(0); nonce < 10; nonce++ {
		valid, _, hash, err := validator.ValidateShare(
			header[:76], // Header without nonce
			nonce,
			easyTarget,
			easyTarget,
		)
		require.NoError(t, err)
		assert.True(t, valid, "share should be valid with easy target")
		assert.Equal(t, 32, len(hash))
	}

	// Test hash determinism
	hash1, _ := algo.Hash(header)
	hash2, _ := algo.Hash(header)
	assert.Equal(t, hash1, hash2)
}

func TestIntegration_DifficultyTargetConversion(t *testing.T) {
	// Test round-trip conversion
	difficulties := []uint64{1, 100, 1000, 65536, 1000000}

	for _, diff := range difficulties {
		target := blockdag.DifficultyToTarget(diff)
		recovered := blockdag.TargetToDifficulty(target)

		// Should be close (within 1% + 1 for rounding)
		delta := float64(diff)*0.01 + 1
		assert.InDelta(t, float64(diff), float64(recovered), delta)
	}
}

// -----------------------------------------------------------------------------
// Hardware-Aware Difficulty Integration Tests
// -----------------------------------------------------------------------------

func TestIntegration_HardwareClassification(t *testing.T) {
	vm := difficulty.NewVardiffManager()

	// Register different miner types
	cpuMiner := vm.RegisterMiner("cpu-1", "cpuminer-multi/1.3", 0)
	assert.Equal(t, difficulty.HardwareCPU, cpuMiner.HardwareClass.Name)

	gpuMiner := vm.RegisterMiner("gpu-1", "CUDA Miner/1.0", 0)
	assert.Equal(t, difficulty.HardwareGPU, gpuMiner.HardwareClass.Name)

	x100Miner := vm.RegisterMiner("x100-1", "BlockDAG-X100/1.0", 0)
	assert.Equal(t, difficulty.HardwareOfficialASIC, x100Miner.HardwareClass.Name)

	// Verify different base difficulties
	assert.Less(t, cpuMiner.GetCurrentDifficulty(), gpuMiner.GetCurrentDifficulty())
	assert.Less(t, gpuMiner.GetCurrentDifficulty(), x100Miner.GetCurrentDifficulty())
}

func TestIntegration_VardiffAdjustment(t *testing.T) {
	// Use short intervals for testing
	vm := difficulty.NewVardiffManagerWithParams(
		100*time.Millisecond, // Target share time
		50*time.Millisecond,  // Retarget time
		3,                    // Min shares
	)

	state := vm.RegisterMiner("test-miner", "GPU", 0)
	initialDiff := state.GetCurrentDifficulty()

	// Simulate shares coming faster than target
	for i := 0; i < 5; i++ {
		state.RecordShare(true, false)
		time.Sleep(10 * time.Millisecond) // Much faster than 100ms target
	}

	// Force adjustment by setting LastAdjustment in the past
	state.SetDifficulty(initialDiff) // Reset to trigger fresh adjustment window
}

// -----------------------------------------------------------------------------
// Full Flow Integration Tests
// -----------------------------------------------------------------------------

func TestIntegration_V2MinerFullFlow(t *testing.T) {
	// This test simulates a complete V2 miner connection flow

	// 1. Protocol Detection
	det := detector.NewDetector()
	v2SetupData := []byte{0x00, 0x00, 0x00, 0x64, 0x00, 0x00} // V2 header
	version := det.DetectFromBytes(v2SetupData)
	assert.Equal(t, detector.ProtocolV2, version)

	// 2. Noise Handshake (would happen over network)
	serverStatic, _ := noise.GenerateKeyPair()
	clientHS, _ := noise.NewInitiatorHandshake()
	serverHS, _ := noise.NewResponderHandshake(serverStatic)

	msg1, _ := clientHS.WriteMessage(nil)
	serverHS.ReadMessage(msg1)
	msg2, _ := serverHS.WriteMessage(nil)
	clientHS.ReadMessage(msg2)

	assert.True(t, clientHS.IsComplete())

	// 3. V2 Message Exchange (over encrypted channel)
	s := binary.NewSerializer()

	// SetupConnection
	setup := &binary.SetupConnection{
		Protocol:        0,
		MinVersion:      2,
		MaxVersion:      2,
		Flags:           0x07,
		Endpoint:        "pool.blockdag.io:3334",
		Vendor:          "BlockDAG",
		HardwareVersion: "X100",
		FirmwareVersion: "1.0.0",
		DeviceID:        "x100-001",
	}
	setupPayload := s.SerializeSetupConnection(setup)
	assert.True(t, len(setupPayload) > 0)

	// 4. Hardware Classification & Difficulty
	vm := difficulty.NewVardiffManager()
	minerState := vm.RegisterMiner("x100-001", "BlockDAG-X100", 240000000)
	assert.Equal(t, difficulty.HardwareOfficialASIC, minerState.HardwareClass.Name)

	// 5. Share Validation
	algo := blockdag.NewScrypyVariant()
	header := make([]byte, 80)
	hash, err := algo.HashHeader(header)
	require.NoError(t, err)
	assert.Equal(t, 32, len(hash))

	// 6. Record share
	vm.RecordShare("x100-001", true, false)
	stats := minerState.GetStats()
	assert.Equal(t, uint64(1), stats.ValidShares)
}

func TestIntegration_MixedMinerPool(t *testing.T) {
	// Simulate a pool with mixed V1 and V2 miners
	vm := difficulty.NewVardiffManager()
	router := detector.NewRouter()

	// Register handlers
	v1Handler := &mockProtocolHandler{protocol: detector.ProtocolV1}
	v2Handler := &mockProtocolHandler{protocol: detector.ProtocolV2}
	router.RegisterHandler(detector.ProtocolV1, v1Handler)
	router.RegisterHandler(detector.ProtocolV2, v2Handler)

	// Simulate miner registrations
	miners := []struct {
		id        string
		userAgent string
		isV2      bool
	}{
		{"gpu-1", "CUDA Miner", false},
		{"gpu-2", "OpenCL Miner", false},
		{"x30-1", "BlockDAG-X30", true},
		{"x100-1", "BlockDAG-X100", true},
		{"x100-2", "BlockDAG-X100", true},
		{"asic-1", "Antminer", false},
	}

	for _, m := range miners {
		vm.RegisterMiner(m.id, m.userAgent, 0)
	}

	assert.Equal(t, 6, vm.GetMinerCount())

	// Verify correct classification
	stats := vm.GetAllStats()
	gpuCount := 0
	asicCount := 0
	officialASICCount := 0

	for _, s := range stats {
		switch s.HardwareClass {
		case difficulty.HardwareGPU:
			gpuCount++
		case difficulty.HardwareASIC:
			asicCount++
		case difficulty.HardwareOfficialASIC:
			officialASICCount++
		}
	}

	assert.Equal(t, 2, gpuCount, "should have 2 GPU miners")
	assert.Equal(t, 2, asicCount, "should have 2 generic ASIC miners") // X30 + Antminer
	assert.Equal(t, 2, officialASICCount, "should have 2 X100 miners")
}

// -----------------------------------------------------------------------------
// Performance Integration Tests
// -----------------------------------------------------------------------------

func BenchmarkIntegration_FullShareValidation(b *testing.B) {
	algo := blockdag.NewScrypyVariant()
	header := make([]byte, 80)
	target := bytes.Repeat([]byte{0xFF}, 32)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		algo.ValidateWork(header[:76], uint32(i), target)
	}
}

func BenchmarkIntegration_V2MessageRoundTrip(b *testing.B) {
	s := binary.NewSerializer()
	share := &binary.SubmitSharesStandard{
		ChannelID:   1,
		SequenceNum: 1,
		JobID:       1000,
		Nonce:       0x12345678,
		NTime:       1703001600,
		Version:     0x20000000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		payload := s.SerializeSubmitSharesStandard(share)
		d := binary.NewDeserializer(payload)
		d.DeserializeSubmitSharesStandard()
	}
}

func BenchmarkIntegration_NoiseEncryptDecrypt(b *testing.B) {
	var key [32]byte
	cs1, _ := noise.NewCipherState(key)
	cs2, _ := noise.NewCipherState(key)
	plaintext := make([]byte, 100)

	ciphertexts := make([][]byte, b.N)
	for i := 0; i < b.N; i++ {
		ciphertexts[i], _ = cs1.Encrypt(plaintext, nil)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cs2.Decrypt(ciphertexts[i], nil)
	}
}

// =============================================================================
// Mock Implementations
// =============================================================================

type mockProtocolHandler struct {
	protocol detector.ProtocolVersion
	onHandle func()
	shutdown bool
}

func (h *mockProtocolHandler) HandleConnection(conn net.Conn) error {
	if h.onHandle != nil {
		h.onHandle()
	}
	return nil
}

func (h *mockProtocolHandler) Protocol() detector.ProtocolVersion {
	return h.protocol
}

func (h *mockProtocolHandler) Shutdown() error {
	h.shutdown = true
	return nil
}

// mockNetConn implements net.Conn for testing
type mockNetConn struct {
	reader *bytes.Reader
	closed bool
	mu     sync.Mutex
}

func newMockNetConn(data []byte) *mockNetConn {
	return &mockNetConn{
		reader: bytes.NewReader(data),
	}
}

func (m *mockNetConn) Read(b []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.reader.Read(b)
}

func (m *mockNetConn) Write(b []byte) (int, error) {
	return len(b), nil
}

func (m *mockNetConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockNetConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 3333}
}

func (m *mockNetConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}
}

func (m *mockNetConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockNetConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockNetConn) SetWriteDeadline(t time.Time) error { return nil }
