package stratum

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// TDD TESTS FOR STRATUM INTERFACES
// These tests define the expected behavior of our interfaces
// =============================================================================

// -----------------------------------------------------------------------------
// HardwareClass Tests
// -----------------------------------------------------------------------------

func TestHardwareClass_String(t *testing.T) {
	tests := []struct {
		name     string
		class    HardwareClass
		expected string
	}{
		{"CPU returns cpu", HardwareClassCPU, "cpu"},
		{"GPU returns gpu", HardwareClassGPU, "gpu"},
		{"FPGA returns fpga", HardwareClassFPGA, "fpga"},
		{"ASIC returns asic", HardwareClassASIC, "asic"},
		{"Official ASIC returns official_asic", HardwareClassOfficialASIC, "official_asic"},
		{"Unknown returns unknown", HardwareClassUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.class.String())
		})
	}
}

func TestHardwareClass_BaseDifficulty(t *testing.T) {
	tests := []struct {
		name    string
		class   HardwareClass
		minDiff uint64
		maxDiff uint64
	}{
		{"CPU has low difficulty", HardwareClassCPU, 1, 100},
		{"GPU has medium difficulty", HardwareClassGPU, 1000, 10000},
		{"FPGA has high difficulty", HardwareClassFPGA, 10000, 50000},
		{"ASIC has very high difficulty", HardwareClassASIC, 20000, 100000},
		{"Official ASIC has highest difficulty", HardwareClassOfficialASIC, 50000, 200000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := tt.class.BaseDifficulty()
			assert.GreaterOrEqual(t, diff, tt.minDiff, "difficulty should be >= min")
			assert.LessOrEqual(t, diff, tt.maxDiff, "difficulty should be <= max")
		})
	}
}

func TestHardwareClass_DifficultyOrdering(t *testing.T) {
	// Official ASICs should have highest difficulty, CPU lowest
	assert.Less(t, HardwareClassCPU.BaseDifficulty(), HardwareClassGPU.BaseDifficulty())
	assert.Less(t, HardwareClassGPU.BaseDifficulty(), HardwareClassFPGA.BaseDifficulty())
	assert.Less(t, HardwareClassFPGA.BaseDifficulty(), HardwareClassASIC.BaseDifficulty())
	assert.Less(t, HardwareClassASIC.BaseDifficulty(), HardwareClassOfficialASIC.BaseDifficulty())
}

// -----------------------------------------------------------------------------
// MessageType Tests
// -----------------------------------------------------------------------------

func TestMessageType_V1Types(t *testing.T) {
	// V1 message types should be defined
	v1Types := []MessageType{
		MessageTypeSubscribe,
		MessageTypeAuthorize,
		MessageTypeSubmit,
		MessageTypeNotify,
		MessageTypeSetDifficulty,
		MessageTypeSetExtranonce,
	}

	for _, mt := range v1Types {
		assert.NotEqual(t, MessageTypeUnknown, mt, "V1 type should not be unknown")
	}
}

func TestMessageType_V2Types(t *testing.T) {
	// V2 message types should be defined
	v2Types := []MessageType{
		MessageTypeSetupConnection,
		MessageTypeSetupConnectionSuccess,
		MessageTypeSetupConnectionError,
		MessageTypeOpenChannel,
		MessageTypeOpenChannelSuccess,
		MessageTypeOpenChannelError,
		MessageTypeNewMiningJob,
		MessageTypeSetNewPrevHash,
		MessageTypeSubmitShares,
		MessageTypeSubmitSharesSuccess,
		MessageTypeSubmitSharesError,
		MessageTypeSetTarget,
		MessageTypeReconnect,
	}

	for _, mt := range v2Types {
		assert.NotEqual(t, MessageTypeUnknown, mt, "V2 type should not be unknown")
	}
}

func TestMessageType_AllUnique(t *testing.T) {
	// All message types should have unique values
	types := []MessageType{
		MessageTypeUnknown,
		MessageTypeSubscribe,
		MessageTypeAuthorize,
		MessageTypeSubmit,
		MessageTypeNotify,
		MessageTypeSetDifficulty,
		MessageTypeSetExtranonce,
		MessageTypeSetupConnection,
		MessageTypeSetupConnectionSuccess,
		MessageTypeSetupConnectionError,
		MessageTypeOpenChannel,
		MessageTypeOpenChannelSuccess,
		MessageTypeOpenChannelError,
		MessageTypeNewMiningJob,
		MessageTypeSetNewPrevHash,
		MessageTypeSubmitShares,
		MessageTypeSubmitSharesSuccess,
		MessageTypeSubmitSharesError,
		MessageTypeSetTarget,
		MessageTypeReconnect,
	}

	seen := make(map[MessageType]bool)
	for _, mt := range types {
		assert.False(t, seen[mt], "MessageType %d should be unique", mt)
		seen[mt] = true
	}
}

// -----------------------------------------------------------------------------
// ProtocolVersion Tests
// -----------------------------------------------------------------------------

func TestProtocolVersion_Values(t *testing.T) {
	assert.Equal(t, ProtocolVersion(1), ProtocolV1)
	assert.Equal(t, ProtocolVersion(2), ProtocolV2)
	assert.NotEqual(t, ProtocolV1, ProtocolV2)
}

// -----------------------------------------------------------------------------
// Share Struct Tests
// -----------------------------------------------------------------------------

func TestShare_Creation(t *testing.T) {
	share := Share{
		MinerID:      "miner-123",
		WorkerName:   "worker1",
		JobID:        "job-456",
		Nonce:        0x12345678,
		Extranonce2:  "00000001",
		NTime:        1703001600,
		Difficulty:   65536,
		IsValid:      true,
		Timestamp:    time.Now(),
		HardwareType: HardwareClassOfficialASIC,
	}

	assert.Equal(t, "miner-123", share.MinerID)
	assert.Equal(t, "worker1", share.WorkerName)
	assert.Equal(t, "job-456", share.JobID)
	assert.Equal(t, uint64(0x12345678), share.Nonce)
	assert.Equal(t, "00000001", share.Extranonce2)
	assert.Equal(t, uint32(1703001600), share.NTime)
	assert.Equal(t, uint64(65536), share.Difficulty)
	assert.True(t, share.IsValid)
	assert.Equal(t, HardwareClassOfficialASIC, share.HardwareType)
}

// -----------------------------------------------------------------------------
// Job Struct Tests
// -----------------------------------------------------------------------------

func TestJob_Creation(t *testing.T) {
	job := Job{
		ID:           "job-789",
		PrevHash:     []byte{0x01, 0x02, 0x03},
		Coinbase1:    []byte{0x04, 0x05},
		Coinbase2:    []byte{0x06, 0x07},
		MerkleBranch: [][]byte{{0x08}, {0x09}},
		Version:      0x20000000,
		NBits:        0x1d00ffff,
		NTime:        1703001600,
		CleanJobs:    true,
		Target:       []byte{0x00, 0x00, 0xff, 0xff},
		Algorithm:    "scrpy-variant",
		Coin:         "blockdag",
		Height:       100000,
		CreatedAt:    time.Now(),
	}

	assert.Equal(t, "job-789", job.ID)
	assert.Equal(t, "scrpy-variant", job.Algorithm)
	assert.Equal(t, "blockdag", job.Coin)
	assert.Equal(t, uint64(100000), job.Height)
	assert.True(t, job.CleanJobs)
}

// -----------------------------------------------------------------------------
// Interface Contract Tests (using mocks)
// -----------------------------------------------------------------------------

func TestShareSubmitter_Interface(t *testing.T) {
	// Verify the interface is properly defined
	var _ ShareSubmitter = (*MockShareSubmitter)(nil)
}

func TestShareValidator_Interface(t *testing.T) {
	var _ ShareValidator = (*MockShareValidator)(nil)
}

func TestJobDistributor_Interface(t *testing.T) {
	var _ JobDistributor = (*MockJobDistributor)(nil)
}

func TestDifficultyManager_Interface(t *testing.T) {
	var _ DifficultyManager = (*MockDifficultyManager)(nil)
}

func TestVardiffManager_Interface(t *testing.T) {
	var _ VardiffManager = (*MockVardiffManager)(nil)
}

func TestMinerSession_Interface(t *testing.T) {
	var _ MinerSession = (*MockMinerSession)(nil)
}

func TestSessionManager_Interface(t *testing.T) {
	var _ SessionManager = (*MockSessionManager)(nil)
}

func TestProtocolHandler_Interface(t *testing.T) {
	var _ ProtocolHandler = (*MockProtocolHandler)(nil)
}

func TestProtocolDetector_Interface(t *testing.T) {
	var _ ProtocolDetector = (*MockProtocolDetector)(nil)
}

func TestProtocolRouter_Interface(t *testing.T) {
	var _ ProtocolRouter = (*MockProtocolRouter)(nil)
}

func TestHardwareClassifier_Interface(t *testing.T) {
	var _ HardwareClassifier = (*MockHardwareClassifier)(nil)
}

func TestHashAlgorithm_Interface(t *testing.T) {
	var _ HashAlgorithm = (*MockHashAlgorithm)(nil)
}

// -----------------------------------------------------------------------------
// Mock Implementation Behavior Tests
// -----------------------------------------------------------------------------

func TestMockShareSubmitter_AcceptsValidShare(t *testing.T) {
	mock := NewMockShareSubmitter()
	mock.AcceptAll = true

	share := Share{
		MinerID:    "test-miner",
		WorkerName: "worker1",
		JobID:      "job-1",
		Difficulty: 1000,
		IsValid:    true,
	}

	accepted, err := mock.SubmitShare(share)
	require.NoError(t, err)
	assert.True(t, accepted)
	assert.Equal(t, 1, mock.SubmitCount)
}

func TestMockShareSubmitter_RejectsInvalidShare(t *testing.T) {
	mock := NewMockShareSubmitter()
	mock.AcceptAll = false

	share := Share{
		MinerID:    "test-miner",
		WorkerName: "worker1",
		JobID:      "job-1",
		Difficulty: 1000,
		IsValid:    false,
	}

	accepted, err := mock.SubmitShare(share)
	require.NoError(t, err)
	assert.False(t, accepted)
}

func TestMockDifficultyManager_GetAndSet(t *testing.T) {
	mock := NewMockDifficultyManager()

	// Set difficulty
	err := mock.SetDifficulty("miner-1", 4096)
	require.NoError(t, err)

	// Get difficulty
	diff := mock.GetDifficulty("miner-1")
	assert.Equal(t, uint64(4096), diff)
}

func TestMockDifficultyManager_DefaultDifficulty(t *testing.T) {
	mock := NewMockDifficultyManager()
	mock.DefaultDifficulty = 1024

	// Should return default for unknown miner
	diff := mock.GetDifficulty("unknown-miner")
	assert.Equal(t, uint64(1024), diff)
}

func TestMockVardiffManager_AdjustsDifficulty(t *testing.T) {
	mock := NewMockVardiffManager()
	mock.TargetShareTime = 10 * time.Second
	mock.DefaultDifficulty = 1000

	// Shares coming too fast (< target/2 = 5s) - should increase difficulty
	newDiff := mock.AdjustDifficulty("miner-1", 2*time.Second)
	assert.Greater(t, newDiff, uint64(1000))

	// Reset and test slow shares (> target*2 = 20s) - should decrease difficulty
	mock.difficulties = make(map[string]uint64)
	mock.difficulties["miner-2"] = 4000
	newDiff = mock.AdjustDifficulty("miner-2", 30*time.Second)
	assert.Less(t, newDiff, uint64(4000))
}

func TestMockJobDistributor_SubscribeToJobs(t *testing.T) {
	mock := NewMockJobDistributor()

	var receivedJob *Job
	handler := func(job Job) {
		receivedJob = &job
	}

	sub := mock.SubscribeToJobs(handler)
	assert.True(t, sub.IsActive())

	// Broadcast a job
	testJob := Job{
		ID:        "test-job",
		Algorithm: "scrpy-variant",
		Coin:      "blockdag",
	}
	err := mock.BroadcastJob(testJob)
	require.NoError(t, err)

	// Handler should have received the job
	require.NotNil(t, receivedJob)
	assert.Equal(t, "test-job", receivedJob.ID)

	// Unsubscribe
	sub.Unsubscribe()
	assert.False(t, sub.IsActive())
}

func TestMockProtocolDetector_DetectsV1(t *testing.T) {
	mock := NewMockProtocolDetector()
	mock.DefaultProtocol = ProtocolV1

	version, err := mock.DetectProtocol(nil)
	require.NoError(t, err)
	assert.Equal(t, ProtocolV1, version)
}

func TestMockProtocolDetector_DetectsV2(t *testing.T) {
	mock := NewMockProtocolDetector()
	mock.DefaultProtocol = ProtocolV2

	version, err := mock.DetectProtocol(nil)
	require.NoError(t, err)
	assert.Equal(t, ProtocolV2, version)
}

func TestMockHardwareClassifier_ClassifiesByHashrate(t *testing.T) {
	mock := NewMockHardwareClassifier()

	tests := []struct {
		name     string
		hashrate float64
		expected HardwareClass
	}{
		{"Low hashrate is CPU", 50000, HardwareClassCPU},
		{"Medium hashrate is GPU", 5000000, HardwareClassGPU},
		{"High hashrate is FPGA", 50000000, HardwareClassFPGA},
		{"Very high hashrate is ASIC", 80000000, HardwareClassASIC},
		{"Highest hashrate is Official ASIC", 240000000, HardwareClassOfficialASIC},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			class := mock.ClassifyHardware("", tt.hashrate)
			assert.Equal(t, tt.expected, class)
		})
	}
}

func TestMockMinerSession_Authorization(t *testing.T) {
	mock := NewMockMinerSession("session-1")

	// Initially not authorized
	assert.False(t, mock.IsAuthorized())

	// Authorize
	err := mock.Authorize("worker1", "password")
	require.NoError(t, err)
	assert.True(t, mock.IsAuthorized())
	assert.Equal(t, "worker1", mock.GetWorkerName())
}

func TestMockMinerSession_ShareTracking(t *testing.T) {
	mock := NewMockMinerSession("session-1")
	mock.Authorize("worker1", "x")

	// Record shares
	mock.RecordShare()
	mock.RecordShare()
	mock.RecordShare()

	assert.Equal(t, uint64(3), mock.GetShareCount())
	assert.False(t, mock.GetLastShareTime().IsZero())
}
