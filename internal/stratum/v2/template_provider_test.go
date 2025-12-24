package v2

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/chimera-pool/chimera-pool-core/internal/stratum"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Template Provider Tests
// =============================================================================

func TestNewTemplateProvider(t *testing.T) {
	config := stratum.TemplateProviderConfig{
		UpdateInterval: time.Second * 5,
		ExtraNonceSize: 4,
	}

	rpcClient := &MockRPCClient{}
	coinConfig := LitecoinConfig()

	provider := NewTemplateProvider(config, rpcClient, coinConfig)
	require.NotNil(t, provider)

	assert.Equal(t, config.UpdateInterval, provider.config.UpdateInterval)
	assert.Equal(t, coinConfig.Symbol, provider.coinConfig.Symbol)
}

func TestTemplateProvider_StartStop(t *testing.T) {
	config := stratum.TemplateProviderConfig{
		UpdateInterval: time.Second,
		ExtraNonceSize: 4,
	}

	rpcClient := createMockRPCClient()
	provider := NewTemplateProvider(config, rpcClient, LitecoinConfig())

	// Start
	err := provider.Start()
	require.NoError(t, err)
	assert.True(t, provider.isRunning.Load())

	// Give time for initial template fetch
	time.Sleep(100 * time.Millisecond)

	// Should have a template
	template, err := provider.GetTemplate()
	require.NoError(t, err)
	require.NotNil(t, template)

	// Stop
	err = provider.Stop()
	require.NoError(t, err)
	assert.False(t, provider.isRunning.Load())
}

func TestTemplateProvider_GetTemplate(t *testing.T) {
	config := stratum.TemplateProviderConfig{
		UpdateInterval: time.Second,
		ExtraNonceSize: 4,
	}

	rpcClient := createMockRPCClient()
	provider := NewTemplateProvider(config, rpcClient, LitecoinConfig())

	err := provider.Start()
	require.NoError(t, err)
	defer provider.Stop()

	time.Sleep(100 * time.Millisecond)

	template, err := provider.GetTemplate()
	require.NoError(t, err)
	require.NotNil(t, template)

	assert.Equal(t, uint64(100), template.Height)
	assert.Equal(t, "scrypt", template.Algorithm)
	assert.Equal(t, "LTC", template.Coin)
	assert.NotEmpty(t, template.PrevHash)
	assert.NotEmpty(t, template.Coinbase)
}

func TestTemplateProvider_GetTemplate_NoTemplate(t *testing.T) {
	config := stratum.TemplateProviderConfig{
		UpdateInterval: time.Second,
	}

	// RPC client that returns error
	rpcClient := &MockRPCClient{
		Error: ErrRPCConnectionFailed,
	}

	provider := NewTemplateProvider(config, rpcClient, LitecoinConfig())

	// Don't start - no template available
	_, err := provider.GetTemplate()
	assert.Equal(t, ErrNoTemplateAvailable, err)
}

func TestTemplateProvider_GetCurrentHeight(t *testing.T) {
	config := stratum.TemplateProviderConfig{
		UpdateInterval: time.Second,
	}

	rpcClient := createMockRPCClient()
	provider := NewTemplateProvider(config, rpcClient, LitecoinConfig())

	height, err := provider.GetCurrentHeight()
	require.NoError(t, err)
	assert.Equal(t, uint64(100), height) // blocks + 1 from mock
}

func TestTemplateProvider_GetNetworkDifficulty(t *testing.T) {
	config := stratum.TemplateProviderConfig{
		UpdateInterval: time.Second,
	}

	rpcClient := createMockRPCClient()
	provider := NewTemplateProvider(config, rpcClient, LitecoinConfig())

	diff, err := provider.GetNetworkDifficulty()
	require.NoError(t, err)
	assert.Equal(t, uint64(1000000), diff)
}

func TestTemplateProvider_SubscribeTemplates(t *testing.T) {
	config := stratum.TemplateProviderConfig{
		UpdateInterval: time.Millisecond * 100,
		ExtraNonceSize: 4,
	}

	rpcClient := createMockRPCClient()
	provider := NewTemplateProvider(config, rpcClient, LitecoinConfig())

	var receivedTemplates atomic.Int64
	subscription := provider.SubscribeTemplates(func(t *stratum.BlockTemplate) {
		receivedTemplates.Add(1)
	})

	require.NotNil(t, subscription)
	assert.True(t, subscription.IsActive())

	err := provider.Start()
	require.NoError(t, err)
	defer provider.Stop()

	// Wait for at least one template
	time.Sleep(200 * time.Millisecond)

	// Should have received at least one notification
	assert.GreaterOrEqual(t, receivedTemplates.Load(), int64(1))

	// Unsubscribe
	subscription.Unsubscribe()
	assert.False(t, subscription.IsActive())
}

func TestTemplateProvider_GetStats(t *testing.T) {
	config := stratum.TemplateProviderConfig{
		UpdateInterval: time.Second,
		ExtraNonceSize: 4,
	}

	rpcClient := createMockRPCClient()
	provider := NewTemplateProvider(config, rpcClient, LitecoinConfig())

	err := provider.Start()
	require.NoError(t, err)
	defer provider.Stop()

	time.Sleep(100 * time.Millisecond)

	stats := provider.GetStats()
	assert.GreaterOrEqual(t, stats["templates_generated"].(int64), int64(1))
	assert.True(t, stats["is_running"].(bool))
}

// =============================================================================
// Coin Config Tests
// =============================================================================

func TestLitecoinConfig(t *testing.T) {
	config := LitecoinConfig()

	assert.Equal(t, "Litecoin", config.Name)
	assert.Equal(t, "LTC", config.Symbol)
	assert.Equal(t, "scrypt", config.Algorithm)
	assert.Equal(t, uint64(840000), config.HalvingInterval)
	assert.Equal(t, uint64(5000000000), config.InitialReward)
}

func TestBlockDAGConfig(t *testing.T) {
	config := BlockDAGConfig()

	assert.Equal(t, "BlockDAG", config.Name)
	assert.Equal(t, "BDAG", config.Symbol)
	assert.Equal(t, "scrpy-variant", config.Algorithm)
	assert.Equal(t, time.Second*10, config.BlockTime)
}

// =============================================================================
// Utility Function Tests
// =============================================================================

func TestCopyBytes(t *testing.T) {
	original := []byte{1, 2, 3, 4, 5}
	copied := copyBytes(original)

	assert.Equal(t, original, copied)

	// Modify original - copied should not change
	original[0] = 99
	assert.NotEqual(t, original[0], copied[0])

	// Nil handling
	assert.Nil(t, copyBytes(nil))
}

func TestReverseBytes(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	reverseBytes(data)
	assert.Equal(t, []byte{5, 4, 3, 2, 1}, data)

	// Even length
	data2 := []byte{1, 2, 3, 4}
	reverseBytes(data2)
	assert.Equal(t, []byte{4, 3, 2, 1}, data2)

	// Single byte
	data3 := []byte{1}
	reverseBytes(data3)
	assert.Equal(t, []byte{1}, data3)
}

func TestBytesEqual(t *testing.T) {
	assert.True(t, bytesEqual([]byte{1, 2, 3}, []byte{1, 2, 3}))
	assert.False(t, bytesEqual([]byte{1, 2, 3}, []byte{1, 2, 4}))
	assert.False(t, bytesEqual([]byte{1, 2, 3}, []byte{1, 2}))
	assert.True(t, bytesEqual(nil, nil))
	assert.True(t, bytesEqual([]byte{}, []byte{}))
}

func TestEncodeHeight(t *testing.T) {
	// Small heights (< 17)
	for h := uint64(1); h < 17; h++ {
		encoded := encodeHeight(h)
		assert.Len(t, encoded, 1)
	}

	// Larger heights
	encoded := encodeHeight(100)
	assert.Greater(t, len(encoded), 1)

	encoded = encodeHeight(1000000)
	assert.Greater(t, len(encoded), 1)
}

func TestGenerateTemplateID(t *testing.T) {
	id1 := generateTemplateID()
	id2 := generateTemplateID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Contains(t, id1, "tpl_")
}

// =============================================================================
// Template Subscription Tests
// =============================================================================

func TestTemplateSubscription_Unsubscribe(t *testing.T) {
	config := stratum.TemplateProviderConfig{
		UpdateInterval: time.Second,
	}

	provider := NewTemplateProvider(config, nil, LitecoinConfig())

	sub := provider.SubscribeTemplates(func(t *stratum.BlockTemplate) {})
	assert.True(t, sub.IsActive())

	sub.Unsubscribe()
	assert.False(t, sub.IsActive())

	// Double unsubscribe should be safe
	sub.Unsubscribe()
	assert.False(t, sub.IsActive())
}

// =============================================================================
// Mock RPC Client Tests
// =============================================================================

func TestMockRPCClient(t *testing.T) {
	client := createMockRPCClient()

	template, err := client.GetBlockTemplate()
	require.NoError(t, err)
	require.NotNil(t, template)
	assert.Equal(t, uint64(100), template.Height)

	info, err := client.GetBlockchainInfo()
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "main", info.Chain)

	diff, err := client.GetNetworkDifficulty()
	require.NoError(t, err)
	assert.Equal(t, float64(1000000), diff)

	err = client.SubmitBlock("00000000")
	assert.NoError(t, err)
}

func TestMockRPCClient_WithError(t *testing.T) {
	client := &MockRPCClient{
		Error: ErrRPCConnectionFailed,
	}

	_, err := client.GetBlockTemplate()
	assert.Error(t, err)

	_, err = client.GetBlockchainInfo()
	assert.Error(t, err)

	_, err = client.GetNetworkDifficulty()
	assert.Error(t, err)
}

// =============================================================================
// Helpers
// =============================================================================

func createMockRPCClient() *MockRPCClient {
	return &MockRPCClient{
		BlockTemplate: &RPCBlockTemplate{
			Version:           536870912,
			PreviousBlockHash: "0000000000000000000000000000000000000000000000000000000000000000",
			Transactions:      []RPCTx{},
			CoinbaseValue:     5000000000,
			Target:            "00000000ffff0000000000000000000000000000000000000000000000000000",
			MinTime:           1700000000,
			CurTime:           1700000100,
			MaxTime:           1700003600,
			Height:            100,
			Bits:              "1d00ffff",
			SigOpLimit:        20000,
			SizeLimit:         4000000,
			WeightLimit:       4000000,
		},
		BlockchainInfo: &RPCBlockchainInfo{
			Chain:         "main",
			Blocks:        99,
			Headers:       99,
			BestBlockHash: "0000000000000000000000000000000000000000000000000000000000000000",
			Difficulty:    1000000,
			SyncProgress:  1.0,
		},
		Difficulty: 1000000,
	}
}
