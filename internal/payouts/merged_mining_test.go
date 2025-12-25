package payouts

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// MOCK REPOSITORIES FOR MERGED MINING
// =============================================================================

type mockAuxBlockRepo struct {
	blocks map[int64]*AuxBlock
	nextID int64
}

func newMockAuxBlockRepo() *mockAuxBlockRepo {
	return &mockAuxBlockRepo{
		blocks: make(map[int64]*AuxBlock),
		nextID: 1,
	}
}

func (m *mockAuxBlockRepo) CreateAuxBlock(ctx context.Context, block *AuxBlock) error {
	block.ID = m.nextID
	block.CreatedAt = time.Now()
	m.blocks[block.ID] = block
	m.nextID++
	return nil
}

func (m *mockAuxBlockRepo) GetAuxBlock(ctx context.Context, id int64) (*AuxBlock, error) {
	return m.blocks[id], nil
}

func (m *mockAuxBlockRepo) GetAuxBlockByHash(ctx context.Context, chainID, hash string) (*AuxBlock, error) {
	for _, b := range m.blocks {
		if b.ChainID == chainID && b.Hash == hash {
			return b, nil
		}
	}
	return nil, nil
}

func (m *mockAuxBlockRepo) GetAuxBlocksForParent(ctx context.Context, parentBlockID int64) ([]AuxBlock, error) {
	var blocks []AuxBlock
	for _, b := range m.blocks {
		if b.ParentBlockID == parentBlockID {
			blocks = append(blocks, *b)
		}
	}
	return blocks, nil
}

func (m *mockAuxBlockRepo) UpdateAuxBlockStatus(ctx context.Context, id int64, status string) error {
	if b, exists := m.blocks[id]; exists {
		b.Status = status
	}
	return nil
}

func (m *mockAuxBlockRepo) GetPendingAuxBlocks(ctx context.Context, chainID string) ([]AuxBlock, error) {
	var blocks []AuxBlock
	for _, b := range m.blocks {
		if b.ChainID == chainID && b.Status == "pending" {
			blocks = append(blocks, *b)
		}
	}
	return blocks, nil
}

type mockAuxPayoutRepo struct {
	payouts map[int64]*AuxPayout
	nextID  int64
}

func newMockAuxPayoutRepo() *mockAuxPayoutRepo {
	return &mockAuxPayoutRepo{
		payouts: make(map[int64]*AuxPayout),
		nextID:  1,
	}
}

func (m *mockAuxPayoutRepo) CreateAuxPayout(ctx context.Context, payout *AuxPayout) error {
	payout.ID = m.nextID
	payout.CreatedAt = time.Now()
	m.payouts[payout.ID] = payout
	m.nextID++
	return nil
}

func (m *mockAuxPayoutRepo) CreateAuxPayouts(ctx context.Context, payouts []AuxPayout) error {
	for i := range payouts {
		if err := m.CreateAuxPayout(ctx, &payouts[i]); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockAuxPayoutRepo) GetAuxPayoutsForBlock(ctx context.Context, auxBlockID int64) ([]AuxPayout, error) {
	var result []AuxPayout
	for _, p := range m.payouts {
		if p.AuxBlockID == auxBlockID {
			result = append(result, *p)
		}
	}
	return result, nil
}

func (m *mockAuxPayoutRepo) GetAuxPayoutsForUser(ctx context.Context, userID int64, chainID string) ([]AuxPayout, error) {
	var result []AuxPayout
	for _, p := range m.payouts {
		if p.UserID == userID && p.ChainID == chainID {
			result = append(result, *p)
		}
	}
	return result, nil
}

func (m *mockAuxPayoutRepo) UpdateAuxPayoutStatus(ctx context.Context, id int64, status, txHash string) error {
	if p, exists := m.payouts[id]; exists {
		p.Status = status
		p.TxHash = txHash
		if status == "paid" {
			now := time.Now()
			p.PaidAt = &now
		}
	}
	return nil
}

func (m *mockAuxPayoutRepo) GetPendingAuxPayouts(ctx context.Context, chainID string, minAmount int64) ([]AuxPayout, error) {
	var result []AuxPayout
	for _, p := range m.payouts {
		if p.ChainID == chainID && p.Status == "pending" && p.Amount >= minAmount {
			result = append(result, *p)
		}
	}
	return result, nil
}

// =============================================================================
// MERGED MINING MANAGER TESTS
// =============================================================================

func TestNewMergedMiningManager(t *testing.T) {
	pm, _ := NewPayoutManager(nil, nil, nil)
	mm := NewMergedMiningManager(pm, nil, nil)

	assert.NotNil(t, mm)
	assert.NotNil(t, mm.auxChains)
}

func TestMergedMiningManager_RegisterAuxChain(t *testing.T) {
	pm, _ := NewPayoutManager(nil, nil, nil)
	mm := NewMergedMiningManager(pm, nil, nil)

	t.Run("registers valid chain", func(t *testing.T) {
		config := &AuxChainConfig{
			ChainID:    "dogecoin",
			ChainName:  "Dogecoin",
			Symbol:     "DOGE",
			Enabled:    true,
			FeePercent: 1.0,
		}

		err := mm.RegisterAuxChain(config)
		require.NoError(t, err)

		chains := mm.GetAuxChains()
		assert.Len(t, chains, 1)
		assert.Equal(t, "dogecoin", chains[0].ChainID)
	})

	t.Run("rejects nil config", func(t *testing.T) {
		err := mm.RegisterAuxChain(nil)
		assert.Error(t, err)
	})

	t.Run("rejects empty chain ID", func(t *testing.T) {
		config := &AuxChainConfig{
			ChainName: "Test",
		}
		err := mm.RegisterAuxChain(config)
		assert.Error(t, err)
	})
}

func TestMergedMiningManager_UnregisterAuxChain(t *testing.T) {
	pm, _ := NewPayoutManager(nil, nil, nil)
	mm := NewMergedMiningManager(pm, nil, nil)

	config := &AuxChainConfig{
		ChainID: "dogecoin",
		Enabled: true,
	}
	mm.RegisterAuxChain(config)

	mm.UnregisterAuxChain("dogecoin")
	chains := mm.GetAuxChains()
	assert.Len(t, chains, 0)
}

func TestMergedMiningManager_GetAuxChains(t *testing.T) {
	pm, _ := NewPayoutManager(nil, nil, nil)
	mm := NewMergedMiningManager(pm, nil, nil)

	// Register multiple chains
	mm.RegisterAuxChain(&AuxChainConfig{ChainID: "doge", Enabled: true})
	mm.RegisterAuxChain(&AuxChainConfig{ChainID: "ltc", Enabled: true})
	mm.RegisterAuxChain(&AuxChainConfig{ChainID: "disabled", Enabled: false})

	chains := mm.GetAuxChains()
	assert.Len(t, chains, 2) // Only enabled chains
}

func TestMergedMiningManager_GetAuxBlockTemplate(t *testing.T) {
	pm, _ := NewPayoutManager(nil, nil, nil)
	mm := NewMergedMiningManager(pm, nil, nil)
	ctx := context.Background()

	t.Run("returns error for unregistered chain", func(t *testing.T) {
		_, err := mm.GetAuxBlockTemplate(ctx, "unknown")
		assert.Error(t, err)
	})

	t.Run("returns error for disabled chain", func(t *testing.T) {
		mm.RegisterAuxChain(&AuxChainConfig{ChainID: "disabled", Enabled: false})
		_, err := mm.GetAuxBlockTemplate(ctx, "disabled")
		assert.Error(t, err)
	})

	t.Run("returns nil for enabled chain (placeholder)", func(t *testing.T) {
		mm.RegisterAuxChain(&AuxChainConfig{ChainID: "doge", Enabled: true})
		template, err := mm.GetAuxBlockTemplate(ctx, "doge")
		require.NoError(t, err)
		assert.Nil(t, template) // Placeholder returns nil
	})
}

func TestMergedMiningManager_SubmitAuxBlock(t *testing.T) {
	pm, _ := NewPayoutManager(nil, nil, nil)
	mm := NewMergedMiningManager(pm, nil, nil)
	ctx := context.Background()

	t.Run("returns error for unregistered chain", func(t *testing.T) {
		err := mm.SubmitAuxBlock(ctx, "unknown", nil)
		assert.Error(t, err)
	})

	t.Run("returns nil for enabled chain (placeholder)", func(t *testing.T) {
		mm.RegisterAuxChain(&AuxChainConfig{ChainID: "doge", Enabled: true})
		err := mm.SubmitAuxBlock(ctx, "doge", []byte("test"))
		require.NoError(t, err)
	})
}

func TestMergedMiningManager_OnAuxBlockFound(t *testing.T) {
	pm, _ := NewPayoutManager(nil, nil, nil)
	auxBlockRepo := newMockAuxBlockRepo()
	mm := NewMergedMiningManager(pm, auxBlockRepo, nil)
	ctx := context.Background()

	mm.RegisterAuxChain(&AuxChainConfig{ChainID: "doge", Enabled: true})

	t.Run("creates aux block record", func(t *testing.T) {
		block := &Block{
			ID:        1,
			Height:    100,
			Hash:      "abc123",
			Reward:    1000000,
			Timestamp: time.Now(),
		}

		err := mm.OnAuxBlockFound(ctx, "doge", block)
		require.NoError(t, err)

		// Verify aux block was created
		auxBlock, _ := auxBlockRepo.GetAuxBlock(ctx, 1)
		assert.NotNil(t, auxBlock)
		assert.Equal(t, "doge", auxBlock.ChainID)
		assert.Equal(t, int64(100), auxBlock.Height)
	})

	t.Run("returns error for nil block", func(t *testing.T) {
		err := mm.OnAuxBlockFound(ctx, "doge", nil)
		assert.Error(t, err)
	})

	t.Run("returns error for unregistered chain", func(t *testing.T) {
		block := &Block{ID: 1}
		err := mm.OnAuxBlockFound(ctx, "unknown", block)
		assert.Error(t, err)
	})
}

func TestMergedMiningManager_GetAuxPayouts(t *testing.T) {
	pm, _ := NewPayoutManager(nil, nil, nil)
	mm := NewMergedMiningManager(pm, nil, nil)
	ctx := context.Background()

	mm.RegisterAuxChain(&AuxChainConfig{
		ChainID:    "doge",
		Enabled:    true,
		FeePercent: 1.0,
	})

	t.Run("calculates payouts for aux chain", func(t *testing.T) {
		shares := []Share{
			{UserID: 1, Difficulty: 1000, IsValid: true, Timestamp: time.Now()},
			{UserID: 2, Difficulty: 1000, IsValid: true, Timestamp: time.Now()},
		}

		payouts, err := mm.GetAuxPayouts(ctx, "doge", shares, 1000000)
		require.NoError(t, err)
		assert.Len(t, payouts, 2)
	})

	t.Run("returns error for unregistered chain", func(t *testing.T) {
		_, err := mm.GetAuxPayouts(ctx, "unknown", nil, 0)
		assert.Error(t, err)
	})
}

func TestMergedMiningManager_ProcessAuxBlockPayouts(t *testing.T) {
	pm, _ := NewPayoutManager(nil, nil, nil)
	auxBlockRepo := newMockAuxBlockRepo()
	auxPayoutRepo := newMockAuxPayoutRepo()
	mm := NewMergedMiningManager(pm, auxBlockRepo, auxPayoutRepo)
	ctx := context.Background()

	mm.RegisterAuxChain(&AuxChainConfig{
		ChainID:    "doge",
		Enabled:    true,
		FeePercent: 1.0,
	})

	t.Run("processes payouts for confirmed block", func(t *testing.T) {
		// Create a confirmed aux block
		auxBlock := &AuxBlock{
			ChainID: "doge",
			Height:  100,
			Hash:    "test",
			Reward:  1000000,
			Status:  "confirmed",
		}
		auxBlockRepo.CreateAuxBlock(ctx, auxBlock)

		shares := []Share{
			{UserID: 1, Difficulty: 1000, IsValid: true, Timestamp: time.Now()},
			{UserID: 2, Difficulty: 1000, IsValid: true, Timestamp: time.Now()},
		}

		err := mm.ProcessAuxBlockPayouts(ctx, auxBlock.ID, shares)
		require.NoError(t, err)

		// Verify payouts were created
		payouts, _ := auxPayoutRepo.GetAuxPayoutsForBlock(ctx, auxBlock.ID)
		assert.Len(t, payouts, 2)
	})

	t.Run("rejects unconfirmed block", func(t *testing.T) {
		auxBlock := &AuxBlock{
			ChainID: "doge",
			Status:  "pending",
		}
		auxBlockRepo.CreateAuxBlock(ctx, auxBlock)

		err := mm.ProcessAuxBlockPayouts(ctx, auxBlock.ID, nil)
		assert.Error(t, err)
	})
}

// =============================================================================
// AUX BLOCK TESTS
// =============================================================================

func TestAuxBlock_Structure(t *testing.T) {
	block := AuxBlock{
		ID:            1,
		ChainID:       "dogecoin",
		Height:        500000,
		Hash:          "abc123",
		ParentBlockID: 100,
		Reward:        10000000000,
		Status:        "confirmed",
		Timestamp:     time.Now(),
		CreatedAt:     time.Now(),
	}

	assert.Equal(t, int64(1), block.ID)
	assert.Equal(t, "dogecoin", block.ChainID)
	assert.Equal(t, int64(500000), block.Height)
	assert.Equal(t, "abc123", block.Hash)
	assert.Equal(t, int64(100), block.ParentBlockID)
	assert.Equal(t, int64(10000000000), block.Reward)
	assert.Equal(t, "confirmed", block.Status)
}

// =============================================================================
// AUX PAYOUT TESTS
// =============================================================================

func TestAuxPayout_Structure(t *testing.T) {
	now := time.Now()
	payout := AuxPayout{
		ID:         1,
		AuxBlockID: 100,
		UserID:     5,
		Amount:     50000000,
		ChainID:    "dogecoin",
		Status:     "paid",
		TxHash:     "tx123",
		CreatedAt:  now,
		PaidAt:     &now,
	}

	assert.Equal(t, int64(1), payout.ID)
	assert.Equal(t, int64(100), payout.AuxBlockID)
	assert.Equal(t, int64(5), payout.UserID)
	assert.Equal(t, int64(50000000), payout.Amount)
	assert.Equal(t, "dogecoin", payout.ChainID)
	assert.Equal(t, "paid", payout.Status)
	assert.Equal(t, "tx123", payout.TxHash)
	assert.NotNil(t, payout.PaidAt)
}
