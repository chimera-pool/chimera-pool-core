package blockdag

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// TDD TESTS FOR BLOCKDAG SCRPY-VARIANT ALGORITHM
// =============================================================================

// -----------------------------------------------------------------------------
// Algorithm Constants Tests
// -----------------------------------------------------------------------------

func TestAlgorithmConstants(t *testing.T) {
	assert.Equal(t, "scrpy-variant", AlgorithmName)
	assert.Equal(t, 32, HashSize)
	assert.Equal(t, 80, HeaderSize)
	assert.Equal(t, 32, TargetSize)
}

// -----------------------------------------------------------------------------
// ScrypyVariant Algorithm Tests
// -----------------------------------------------------------------------------

func TestScrypyVariant_Name(t *testing.T) {
	algo := NewScrypyVariant()
	assert.Equal(t, "scrpy-variant", algo.Name())
}

func TestScrypyVariant_Hash(t *testing.T) {
	algo := NewScrypyVariant()

	// Hash should produce 32 bytes
	data := []byte("test data for hashing")
	hash, err := algo.Hash(data)
	require.NoError(t, err)
	assert.Equal(t, HashSize, len(hash))
}

func TestScrypyVariant_Hash_Deterministic(t *testing.T) {
	algo := NewScrypyVariant()

	data := []byte("deterministic test")
	hash1, err := algo.Hash(data)
	require.NoError(t, err)

	hash2, err := algo.Hash(data)
	require.NoError(t, err)

	assert.Equal(t, hash1, hash2, "same input should produce same hash")
}

func TestScrypyVariant_Hash_DifferentInputs(t *testing.T) {
	algo := NewScrypyVariant()

	hash1, _ := algo.Hash([]byte("input 1"))
	hash2, _ := algo.Hash([]byte("input 2"))

	assert.NotEqual(t, hash1, hash2, "different inputs should produce different hashes")
}

func TestScrypyVariant_Hash_EmptyInput(t *testing.T) {
	algo := NewScrypyVariant()

	_, err := algo.Hash([]byte{})
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidHeaderSize, err)
}

func TestScrypyVariant_HashHeader(t *testing.T) {
	algo := NewScrypyVariant()

	// Valid 80-byte header
	header := make([]byte, HeaderSize)
	for i := range header {
		header[i] = byte(i)
	}

	hash, err := algo.HashHeader(header)
	require.NoError(t, err)
	assert.Equal(t, HashSize, len(hash))
}

func TestScrypyVariant_HashHeader_InvalidSize(t *testing.T) {
	algo := NewScrypyVariant()

	// Too short
	_, err := algo.HashHeader(make([]byte, 79))
	assert.Error(t, err)

	// Too long
	_, err = algo.HashHeader(make([]byte, 81))
	assert.Error(t, err)
}

func TestScrypyVariant_ValidateHash(t *testing.T) {
	algo := NewScrypyVariant()

	tests := []struct {
		name     string
		hash     []byte
		target   []byte
		expected bool
	}{
		{
			name:     "Hash below target (valid)",
			hash:     bytes.Repeat([]byte{0x00}, 32),
			target:   append([]byte{0x00, 0x00, 0xFF}, bytes.Repeat([]byte{0xFF}, 29)...),
			expected: true,
		},
		{
			name:     "Hash above target (invalid)",
			hash:     append([]byte{0xFF}, bytes.Repeat([]byte{0x00}, 31)...),
			target:   append([]byte{0x00, 0x00, 0xFF}, bytes.Repeat([]byte{0xFF}, 29)...),
			expected: false,
		},
		{
			name:     "Hash equals target (valid)",
			hash:     bytes.Repeat([]byte{0x0F}, 32),
			target:   bytes.Repeat([]byte{0x0F}, 32),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := algo.ValidateHash(tt.hash, tt.target)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestScrypyVariant_ValidateHash_InvalidSizes(t *testing.T) {
	algo := NewScrypyVariant()

	// Invalid hash size
	assert.False(t, algo.ValidateHash(make([]byte, 31), make([]byte, 32)))

	// Invalid target size
	assert.False(t, algo.ValidateHash(make([]byte, 32), make([]byte, 31)))
}

func TestScrypyVariant_ValidateWork(t *testing.T) {
	algo := NewScrypyVariant()

	// Create a header without nonce (76 bytes)
	header := make([]byte, HeaderSize-4)
	for i := range header {
		header[i] = byte(i)
	}

	// Very easy target (all 0xFF)
	target := bytes.Repeat([]byte{0xFF}, TargetSize)

	// Any nonce should be valid with this easy target
	valid, hash, err := algo.ValidateWork(header, 12345, target)
	require.NoError(t, err)
	assert.True(t, valid)
	assert.Equal(t, HashSize, len(hash))
}

// -----------------------------------------------------------------------------
// Target/Difficulty Conversion Tests
// -----------------------------------------------------------------------------

func TestDifficultyToTarget(t *testing.T) {
	tests := []struct {
		difficulty uint64
	}{
		{1},
		{100},
		{65536},
		{1000000},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			target := DifficultyToTarget(tt.difficulty)
			assert.Equal(t, TargetSize, len(target))

			// Higher difficulty = lower target (more leading zeros)
			if tt.difficulty > 1 {
				lowerTarget := DifficultyToTarget(tt.difficulty * 2)
				assert.True(t, bytes.Compare(lowerTarget, target) < 0,
					"higher difficulty should produce lower target")
			}
		})
	}
}

func TestDifficultyToTarget_Zero(t *testing.T) {
	// Zero difficulty should be treated as 1
	target := DifficultyToTarget(0)
	targetOne := DifficultyToTarget(1)
	assert.Equal(t, targetOne, target)
}

func TestTargetToDifficulty(t *testing.T) {
	// Round-trip test
	for _, diff := range []uint64{1, 100, 65536, 1000000} {
		target := DifficultyToTarget(diff)
		recovered := TargetToDifficulty(target)

		// Allow some rounding error due to integer division
		assert.InDelta(t, float64(diff), float64(recovered), float64(diff)*0.01+1)
	}
}

func TestTargetToDifficulty_InvalidSize(t *testing.T) {
	// Invalid size should return 1
	diff := TargetToDifficulty(make([]byte, 31))
	assert.Equal(t, uint64(1), diff)
}

func TestTargetToDifficulty_ZeroTarget(t *testing.T) {
	// Zero target = maximum difficulty
	diff := TargetToDifficulty(make([]byte, 32))
	assert.Equal(t, ^uint64(0), diff)
}

func TestCompactToTarget_And_TargetToCompact(t *testing.T) {
	// Test known Bitcoin-style compact values
	tests := []uint32{
		0x1d00ffff, // Bitcoin genesis target
		0x1b0404cb, // Typical difficulty
		0x170da2c2, // Higher difficulty
	}

	for _, compact := range tests {
		target := CompactToTarget(compact)
		assert.Equal(t, TargetSize, len(target))

		// Round-trip
		recovered := TargetToCompact(target)
		// May not be exactly equal due to normalization, but should be close
		recoveredTarget := CompactToTarget(recovered)
		assert.Equal(t, target, recoveredTarget)
	}
}

// -----------------------------------------------------------------------------
// Block Template Tests
// -----------------------------------------------------------------------------

func TestNewBlockTemplate(t *testing.T) {
	bt := NewBlockTemplate()
	assert.NotNil(t, bt)
	assert.Equal(t, TargetSize, len(bt.Target))
}

func TestBlockTemplate_SetBits(t *testing.T) {
	bt := NewBlockTemplate()
	bt.SetBits(0x1d00ffff)

	assert.Equal(t, uint32(0x1d00ffff), bt.Bits)
	assert.Equal(t, TargetSize, len(bt.Target))
	assert.True(t, bt.Difficulty > 0)
}

func TestBlockTemplate_BuildHeader(t *testing.T) {
	bt := NewBlockTemplate()
	bt.Version = 0x20000000
	bt.PrevHash = [32]byte{0x01, 0x02, 0x03}
	bt.MerkleRoot = [32]byte{0x04, 0x05, 0x06}
	bt.Timestamp = 1703001600
	bt.Bits = 0x1d00ffff

	header := bt.BuildHeader()
	assert.Equal(t, HeaderSize, len(header))

	// Verify version
	version := binary.LittleEndian.Uint32(header[0:4])
	assert.Equal(t, uint32(0x20000000), version)

	// Verify timestamp
	timestamp := binary.LittleEndian.Uint32(header[68:72])
	assert.Equal(t, uint32(1703001600), timestamp)

	// Verify bits
	bits := binary.LittleEndian.Uint32(header[72:76])
	assert.Equal(t, uint32(0x1d00ffff), bits)

	// Nonce should be zero
	nonce := binary.LittleEndian.Uint32(header[76:80])
	assert.Equal(t, uint32(0), nonce)
}

func TestBlockTemplate_BuildHeaderWithNonce(t *testing.T) {
	bt := NewBlockTemplate()
	bt.Version = 0x20000000
	bt.Bits = 0x1d00ffff

	nonce := uint32(0x12345678)
	header := bt.BuildHeaderWithNonce(nonce)

	// Verify nonce is set
	extractedNonce := binary.LittleEndian.Uint32(header[76:80])
	assert.Equal(t, nonce, extractedNonce)
}

// -----------------------------------------------------------------------------
// Share Validator Tests
// -----------------------------------------------------------------------------

func TestNewShareValidator(t *testing.T) {
	sv := NewShareValidator()
	assert.NotNil(t, sv)
	assert.NotNil(t, sv.algo)
}

func TestShareValidator_ValidateShare(t *testing.T) {
	sv := NewShareValidator()

	// Create test header
	header := make([]byte, HeaderSize-4)
	for i := range header {
		header[i] = byte(i)
	}

	// Easy targets
	shareTarget := bytes.Repeat([]byte{0xFF}, TargetSize)
	blockTarget := bytes.Repeat([]byte{0xFF}, TargetSize)

	shareValid, blockValid, hash, err := sv.ValidateShare(header, 0, shareTarget, blockTarget)
	require.NoError(t, err)
	assert.True(t, shareValid)
	assert.True(t, blockValid)
	assert.Equal(t, HashSize, len(hash))
}

func TestShareValidator_ValidateShare_ShareOnly(t *testing.T) {
	sv := NewShareValidator()

	header := make([]byte, HeaderSize-4)

	// Easy share target, hard block target
	shareTarget := bytes.Repeat([]byte{0xFF}, TargetSize)
	blockTarget := bytes.Repeat([]byte{0x00}, TargetSize) // Impossible

	shareValid, blockValid, _, err := sv.ValidateShare(header, 0, shareTarget, blockTarget)
	require.NoError(t, err)
	assert.True(t, shareValid, "share should be valid")
	assert.False(t, blockValid, "block should not be valid")
}

func TestShareValidator_QuickValidate(t *testing.T) {
	sv := NewShareValidator()

	hash := bytes.Repeat([]byte{0x00}, HashSize)
	easyTarget := bytes.Repeat([]byte{0xFF}, TargetSize)
	hardTarget := bytes.Repeat([]byte{0x00}, TargetSize)

	assert.True(t, sv.QuickValidate(hash, easyTarget))
	assert.True(t, sv.QuickValidate(hash, hardTarget)) // Zero hash always valid
}

// -----------------------------------------------------------------------------
// Merkle Tree Tests
// -----------------------------------------------------------------------------

func TestComputeMerkleRoot_Empty(t *testing.T) {
	root := ComputeMerkleRoot(nil)
	assert.Equal(t, 32, len(root))
	assert.Equal(t, bytes.Repeat([]byte{0x00}, 32), root)
}

func TestComputeMerkleRoot_Single(t *testing.T) {
	txHash := make([]byte, 32)
	for i := range txHash {
		txHash[i] = byte(i)
	}

	root := ComputeMerkleRoot([][]byte{txHash})
	assert.Equal(t, txHash, root)
}

func TestComputeMerkleRoot_Multiple(t *testing.T) {
	tx1 := bytes.Repeat([]byte{0x01}, 32)
	tx2 := bytes.Repeat([]byte{0x02}, 32)

	root := ComputeMerkleRoot([][]byte{tx1, tx2})
	assert.Equal(t, 32, len(root))
	assert.NotEqual(t, tx1, root)
	assert.NotEqual(t, tx2, root)
}

func TestComputeMerkleRoot_Deterministic(t *testing.T) {
	tx1 := bytes.Repeat([]byte{0x01}, 32)
	tx2 := bytes.Repeat([]byte{0x02}, 32)
	tx3 := bytes.Repeat([]byte{0x03}, 32)

	root1 := ComputeMerkleRoot([][]byte{tx1, tx2, tx3})
	root2 := ComputeMerkleRoot([][]byte{tx1, tx2, tx3})

	assert.Equal(t, root1, root2)
}

// -----------------------------------------------------------------------------
// Utility Functions Tests
// -----------------------------------------------------------------------------

func TestReverseBytes(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04}
	ReverseBytes(data)
	assert.Equal(t, []byte{0x04, 0x03, 0x02, 0x01}, data)
}

func TestReverseBytes_Empty(t *testing.T) {
	data := []byte{}
	ReverseBytes(data) // Should not panic
	assert.Equal(t, []byte{}, data)
}

func TestReverseBytes_Single(t *testing.T) {
	data := []byte{0x42}
	ReverseBytes(data)
	assert.Equal(t, []byte{0x42}, data)
}

func TestCompareHashes(t *testing.T) {
	a := []byte{0x00, 0x00, 0x01}
	b := []byte{0x00, 0x00, 0x02}
	c := []byte{0x00, 0x00, 0x01}

	assert.Equal(t, -1, CompareHashes(a, b))
	assert.Equal(t, 1, CompareHashes(b, a))
	assert.Equal(t, 0, CompareHashes(a, c))
}

// -----------------------------------------------------------------------------
// Benchmark Tests
// -----------------------------------------------------------------------------

func BenchmarkScrypyVariant_Hash(b *testing.B) {
	algo := NewScrypyVariant()
	data := make([]byte, HeaderSize)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		algo.Hash(data)
	}
}

func BenchmarkScrypyVariant_HashHeader(b *testing.B) {
	algo := NewScrypyVariant()
	header := make([]byte, HeaderSize)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		algo.HashHeader(header)
	}
}

func BenchmarkShareValidator_ValidateShare(b *testing.B) {
	sv := NewShareValidator()
	header := make([]byte, HeaderSize-4)
	target := bytes.Repeat([]byte{0xFF}, TargetSize)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sv.ValidateShare(header, uint32(i), target, target)
	}
}

func BenchmarkDifficultyToTarget(b *testing.B) {
	for i := 0; i < b.N; i++ {
		DifficultyToTarget(65536)
	}
}

func BenchmarkComputeMerkleRoot(b *testing.B) {
	txs := make([][]byte, 100)
	for i := range txs {
		txs[i] = make([]byte, 32)
		txs[i][0] = byte(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ComputeMerkleRoot(txs)
	}
}
