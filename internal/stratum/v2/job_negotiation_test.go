package v2

import (
	"testing"
	"time"

	"github.com/chimera-pool/chimera-pool-core/internal/stratum"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Job Declarator Server Tests
// =============================================================================

func TestNewJobDeclaratorServer(t *testing.T) {
	config := DefaultJobNegotiationConfig()
	config.Enabled = true

	server := NewJobDeclaratorServer(config)
	require.NotNil(t, server)

	assert.Equal(t, config.Policy.AllowMinerTemplates, server.policy.AllowMinerTemplates)
	assert.Equal(t, config.Policy.MinCoinbasePoolShare, server.policy.MinCoinbasePoolShare)
}

func TestJobDeclaratorServer_StartStop(t *testing.T) {
	config := DefaultJobNegotiationConfig()
	config.Enabled = true
	config.ListenAddress = ":0" // Random port

	server := NewJobDeclaratorServer(config)

	// Start
	err := server.Start()
	require.NoError(t, err)
	assert.True(t, server.isRunning.Load())

	// Stop
	err = server.Stop()
	require.NoError(t, err)
	assert.False(t, server.isRunning.Load())
}

func TestJobDeclaratorServer_StartDisabled(t *testing.T) {
	config := DefaultJobNegotiationConfig()
	config.Enabled = false

	server := NewJobDeclaratorServer(config)

	err := server.Start()
	assert.Equal(t, ErrNegotiationDisabled, err)
}

func TestJobDeclaratorServer_HandleDeclaration_Success(t *testing.T) {
	config := DefaultJobNegotiationConfig()
	config.Enabled = true
	config.ListenAddress = ":0"

	server := NewJobDeclaratorServer(config)
	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Create valid template
	template := createValidTemplate()

	declaration := &stratum.JobDeclaration{
		Template: template,
		Priority: 1,
	}

	result, err := server.HandleDeclaration("miner1", declaration)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.Accepted)
	assert.NotEmpty(t, result.DeclarationID)
	assert.NotEmpty(t, result.AssignedJobID)
	assert.True(t, result.ValidUntil.After(time.Now()))
}

func TestJobDeclaratorServer_HandleDeclaration_InvalidTemplate(t *testing.T) {
	config := DefaultJobNegotiationConfig()
	config.Enabled = true
	config.ListenAddress = ":0"

	server := NewJobDeclaratorServer(config)
	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Create invalid template (missing prev hash)
	template := &stratum.BlockTemplate{
		Version:  1,
		PrevHash: []byte{}, // Invalid - should be 32 bytes
		Height:   100,
		Coinbase: make([]byte, 100),
		Bits:     0x1d00ffff,
	}

	declaration := &stratum.JobDeclaration{
		Template: template,
	}

	result, err := server.HandleDeclaration("miner1", declaration)
	assert.Error(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Accepted)
	assert.Equal(t, "INVALID_PREV_HASH", result.ErrorCode)
}

func TestJobDeclaratorServer_HandleDeclaration_RateLimit(t *testing.T) {
	config := DefaultJobNegotiationConfig()
	config.Enabled = true
	config.ListenAddress = ":0"
	config.Policy.MaxDeclarationsPerMin = 2

	server := NewJobDeclaratorServer(config)
	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	template := createValidTemplate()

	// First two should succeed
	for i := 0; i < 2; i++ {
		declaration := &stratum.JobDeclaration{Template: template}
		result, err := server.HandleDeclaration("miner1", declaration)
		require.NoError(t, err)
		assert.True(t, result.Accepted)
	}

	// Third should be rate limited
	declaration := &stratum.JobDeclaration{Template: template}
	result, err := server.HandleDeclaration("miner1", declaration)
	assert.Equal(t, ErrRateLimitExceeded, err)
	assert.False(t, result.Accepted)
	assert.Equal(t, "RATE_LIMIT_EXCEEDED", result.ErrorCode)
}

func TestJobDeclaratorServer_HandleDeclaration_MinerNotAllowed(t *testing.T) {
	config := DefaultJobNegotiationConfig()
	config.Enabled = true
	config.ListenAddress = ":0"
	config.Policy.AllowedMinerIDs = []string{"allowed_miner"}

	server := NewJobDeclaratorServer(config)
	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	template := createValidTemplate()
	declaration := &stratum.JobDeclaration{Template: template}

	// Not in whitelist
	result, err := server.HandleDeclaration("not_allowed_miner", declaration)
	assert.Equal(t, ErrMinerNotAllowed, err)
	assert.False(t, result.Accepted)

	// In whitelist
	result, err = server.HandleDeclaration("allowed_miner", declaration)
	require.NoError(t, err)
	assert.True(t, result.Accepted)
}

func TestJobDeclaratorServer_GetActiveDeclarations(t *testing.T) {
	config := DefaultJobNegotiationConfig()
	config.Enabled = true
	config.ListenAddress = ":0"

	server := NewJobDeclaratorServer(config)
	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	template := createValidTemplate()

	// Add declarations for multiple miners
	for i := 0; i < 3; i++ {
		declaration := &stratum.JobDeclaration{Template: template}
		_, err := server.HandleDeclaration("miner"+string(rune('1'+i)), declaration)
		require.NoError(t, err)
	}

	active := server.GetActiveDeclarations()
	assert.Len(t, active, 3)
}

func TestJobDeclaratorServer_GetDeclarationByMiner(t *testing.T) {
	config := DefaultJobNegotiationConfig()
	config.Enabled = true
	config.ListenAddress = ":0"

	server := NewJobDeclaratorServer(config)
	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	template := createValidTemplate()
	declaration := &stratum.JobDeclaration{Template: template}

	_, err = server.HandleDeclaration("miner1", declaration)
	require.NoError(t, err)

	// Should find
	found, ok := server.GetDeclarationByMiner("miner1")
	assert.True(t, ok)
	assert.NotNil(t, found)

	// Should not find
	_, ok = server.GetDeclarationByMiner("nonexistent")
	assert.False(t, ok)
}

func TestJobDeclaratorServer_SetTemplatePolicy(t *testing.T) {
	config := DefaultJobNegotiationConfig()
	server := NewJobDeclaratorServer(config)

	newPolicy := stratum.TemplatePolicy{
		AllowMinerTemplates:   false,
		MinCoinbasePoolShare:  0.05,
		MaxDeclarationsPerMin: 20,
	}

	server.SetTemplatePolicy(newPolicy)
	retrieved := server.GetTemplatePolicy()

	assert.Equal(t, newPolicy.AllowMinerTemplates, retrieved.AllowMinerTemplates)
	assert.Equal(t, newPolicy.MinCoinbasePoolShare, retrieved.MinCoinbasePoolShare)
	assert.Equal(t, newPolicy.MaxDeclarationsPerMin, retrieved.MaxDeclarationsPerMin)
}

func TestJobDeclaratorServer_GetStats(t *testing.T) {
	config := DefaultJobNegotiationConfig()
	config.Enabled = true
	config.ListenAddress = ":0"

	server := NewJobDeclaratorServer(config)
	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Submit some declarations
	template := createValidTemplate()
	for i := 0; i < 5; i++ {
		declaration := &stratum.JobDeclaration{Template: template}
		server.HandleDeclaration("miner"+string(rune('1'+i)), declaration)
	}

	stats := server.GetStats()
	assert.Equal(t, int64(5), stats["total_declarations"])
	assert.Equal(t, int64(5), stats["accepted_declarations"])
	assert.Equal(t, int64(0), stats["rejected_declarations"])
	assert.True(t, stats["is_running"].(bool))
}

// =============================================================================
// Template Validator Tests
// =============================================================================

func TestScryptTemplateValidator_ValidateTemplate_Valid(t *testing.T) {
	validator := NewScryptTemplateValidator("scrypt", 4000000)

	template := createValidTemplate()

	result, err := validator.ValidateTemplate(template)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.Valid)
	assert.Empty(t, result.ErrorCode)
}

func TestScryptTemplateValidator_ValidateTemplate_NullTemplate(t *testing.T) {
	validator := NewScryptTemplateValidator("scrypt", 4000000)

	result, err := validator.ValidateTemplate(nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.False(t, result.Valid)
	assert.Equal(t, "TEMPLATE_NULL", result.ErrorCode)
}

func TestScryptTemplateValidator_ValidateTemplate_InvalidPrevHash(t *testing.T) {
	validator := NewScryptTemplateValidator("scrypt", 4000000)

	template := &stratum.BlockTemplate{
		Version:  1,
		PrevHash: []byte{1, 2, 3}, // Invalid - should be 32 bytes
		Coinbase: make([]byte, 100),
		Height:   100,
	}

	result, err := validator.ValidateTemplate(template)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.False(t, result.Valid)
	assert.Equal(t, "INVALID_PREV_HASH", result.ErrorCode)
}

func TestScryptTemplateValidator_ValidateCoinbase(t *testing.T) {
	validator := NewScryptTemplateValidator("scrypt", 4000000)

	t.Run("ValidCoinbase", func(t *testing.T) {
		coinbase := make([]byte, 150)
		err := validator.ValidateCoinbase(coinbase, 100, 5000000000)
		assert.NoError(t, err)
	})

	t.Run("CoinbaseTooShort", func(t *testing.T) {
		coinbase := make([]byte, 50)
		err := validator.ValidateCoinbase(coinbase, 100, 5000000000)
		assert.Error(t, err)
	})

	t.Run("RewardTooHigh", func(t *testing.T) {
		coinbase := make([]byte, 150)
		err := validator.ValidateCoinbase(coinbase, 100, 100000000000) // 1000 LTC - way too high
		assert.Error(t, err)
	})
}

func TestScryptTemplateValidator_GetBlockReward(t *testing.T) {
	validator := NewScryptTemplateValidator("scrypt", 4000000)

	// Test reward schedule
	testCases := []struct {
		height         uint64
		expectedReward uint64
	}{
		{0, 5000000000},       // Genesis block: 50 LTC
		{100, 5000000000},     // Early block: 50 LTC
		{839999, 5000000000},  // Just before first halving: 50 LTC
		{840000, 2500000000},  // First halving: 25 LTC
		{1680000, 1250000000}, // Second halving: 12.5 LTC
	}

	for _, tc := range testCases {
		reward := validator.GetBlockReward(tc.height)
		assert.Equal(t, tc.expectedReward, reward, "Height: %d", tc.height)
	}
}

func TestScryptTemplateValidator_GetMaxBlockWeight(t *testing.T) {
	validator := NewScryptTemplateValidator("scrypt", 4000000)
	assert.Equal(t, uint32(4000000), validator.GetMaxBlockWeight())
}

// =============================================================================
// Default Configuration Tests
// =============================================================================

func TestDefaultJobNegotiationConfig(t *testing.T) {
	config := DefaultJobNegotiationConfig()

	assert.False(t, config.Enabled) // Disabled by default
	assert.Equal(t, ":3335", config.ListenAddress)
	assert.True(t, config.Policy.AllowMinerTemplates)
	assert.True(t, config.Policy.RequirePoolCoinbase)
	assert.Equal(t, 0.02, config.Policy.MinCoinbasePoolShare)
	assert.Equal(t, 10, config.Policy.MaxDeclarationsPerMin)
	assert.Equal(t, time.Minute*5, config.Policy.DeclarationTTL)
	assert.True(t, config.Policy.FallbackOnRejection)
	assert.Equal(t, 10, config.MaxConcurrentValidations)
	assert.Equal(t, time.Second*10, config.ValidationTimeout)
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestJobNegotiation_FullFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup server
	config := DefaultJobNegotiationConfig()
	config.Enabled = true
	config.ListenAddress = ":0"

	server := NewJobDeclaratorServer(config)

	// Register validator
	validator := NewScryptTemplateValidator("scrypt", 4000000)
	server.RegisterValidator(validator)

	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Simulate miner submitting template
	template := createValidTemplate()
	declaration := &stratum.JobDeclaration{
		Template: template,
		Priority: 1,
	}

	// Submit declaration
	result, err := server.HandleDeclaration("miner1", declaration)
	require.NoError(t, err)
	assert.True(t, result.Accepted)

	// Verify declaration stored
	stored, ok := server.GetDeclarationByMiner("miner1")
	require.True(t, ok)
	assert.Equal(t, result.DeclarationID, stored.DeclarationID)
	assert.True(t, stored.AcceptedByPool)

	// Submit updated declaration (should replace)
	template2 := createValidTemplate()
	template2.Height = 101
	declaration2 := &stratum.JobDeclaration{Template: template2}

	result2, err := server.HandleDeclaration("miner1", declaration2)
	require.NoError(t, err)
	assert.True(t, result2.Accepted)

	// Verify only one declaration per miner
	active := server.GetActiveDeclarations()
	minerCount := 0
	for _, d := range active {
		if d.MinerID == "miner1" {
			minerCount++
		}
	}
	assert.Equal(t, 1, minerCount)
}

func TestJobNegotiation_FallbackOnRejection(t *testing.T) {
	config := DefaultJobNegotiationConfig()
	config.Enabled = true
	config.ListenAddress = ":0"
	config.Policy.FallbackOnRejection = true

	server := NewJobDeclaratorServer(config)
	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Submit invalid template
	invalidTemplate := &stratum.BlockTemplate{
		Version:  1,
		PrevHash: []byte{1, 2, 3}, // Invalid
		Coinbase: make([]byte, 100),
	}

	declaration := &stratum.JobDeclaration{Template: invalidTemplate}
	result, err := server.HandleDeclaration("miner1", declaration)

	assert.Error(t, err)
	assert.False(t, result.Accepted)
	// With a template provider, PoolTemplate would be set
	// Without one, it's nil
}

// =============================================================================
// Helpers
// =============================================================================

func createValidTemplate() *stratum.BlockTemplate {
	prevHash := make([]byte, 32)
	for i := range prevHash {
		prevHash[i] = byte(i)
	}

	merkleRoot := make([]byte, 32)
	for i := range merkleRoot {
		merkleRoot[i] = byte(i + 32)
	}

	coinbase := make([]byte, 150)
	for i := range coinbase {
		coinbase[i] = byte(i % 256)
	}

	return &stratum.BlockTemplate{
		TemplateID:    "test-template-1",
		Version:       536870912, // BIP9 version
		PrevHash:      prevHash,
		MerkleRoot:    merkleRoot,
		Timestamp:     uint32(time.Now().Unix()),
		Bits:          0x1d00ffff,
		Height:        100,
		Coinbase:      coinbase,
		CoinbaseValue: 5000000000, // 50 LTC
		Target:        make([]byte, 32),
		Algorithm:     "scrypt",
		Coin:          "LTC",
		MinTime:       uint32(time.Now().Add(-time.Hour).Unix()),
		MaxTime:       uint32(time.Now().Add(time.Hour).Unix()),
		CreatedAt:     time.Now(),
	}
}
