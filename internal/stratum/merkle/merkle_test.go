package merkle

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// TestBuildBranch_Empty tests empty transaction list
func TestBuildBranch_Empty(t *testing.T) {
	builder := NewBuilder()
	branch := builder.BuildBranch(nil)

	if len(branch) != 0 {
		t.Errorf("expected empty branch for empty tx list, got %d elements", len(branch))
	}
}

// TestBuildBranch_SingleTx tests single non-coinbase transaction
func TestBuildBranch_SingleTx(t *testing.T) {
	builder := NewBuilder()

	// 1 non-coinbase tx means branch has 1 element (the sibling)
	txHash := sha256Hash([]byte("tx1"))
	branch := builder.BuildBranch([][]byte{txHash})

	// Branch should contain the single tx as coinbase's sibling
	if len(branch) != 1 {
		t.Errorf("expected 1 branch element for 1 non-coinbase tx, got %d", len(branch))
		return
	}

	if !bytes.Equal(branch[0], txHash) {
		t.Errorf("branch[0] should equal txHash")
	}
}

// TestBuildBranch_TwoTx tests two transactions
func TestBuildBranch_TwoTx(t *testing.T) {
	builder := NewBuilder()

	// Coinbase at index 0, one other tx at index 1
	coinbaseHash := sha256Hash([]byte("coinbase"))
	tx1Hash := sha256Hash([]byte("tx1"))

	branch := builder.BuildBranch([][]byte{tx1Hash}) // Pass only non-coinbase txs

	// Branch should contain tx1Hash as the sibling
	if len(branch) != 1 {
		t.Errorf("expected 1 branch element for 2 txs, got %d", len(branch))
		return
	}

	if !bytes.Equal(branch[0], tx1Hash) {
		t.Errorf("branch[0] should be tx1Hash")
	}

	// Verify we can compute the root
	root := builder.ComputeRoot(coinbaseHash, branch)
	expectedRoot := testDoubleSha256(append(coinbaseHash, tx1Hash...))

	if !bytes.Equal(root, expectedRoot) {
		t.Errorf("computed root doesn't match expected")
	}
}

// TestBuildBranch_FourTx tests four transactions
func TestBuildBranch_FourTx(t *testing.T) {
	builder := NewBuilder()

	// 4 txs: coinbase + 3 others
	// Tree structure:
	//        root
	//       /    \
	//    h01      h23
	//   /  \     /  \
	//  cb  tx1  tx2  tx3

	tx1Hash := sha256Hash([]byte("tx1"))
	tx2Hash := sha256Hash([]byte("tx2"))
	tx3Hash := sha256Hash([]byte("tx3"))

	branch := builder.BuildBranch([][]byte{tx1Hash, tx2Hash, tx3Hash})

	// Branch should have 2 elements:
	// 1. tx1 (sibling at level 0)
	// 2. h23 (sibling at level 1)
	if len(branch) != 2 {
		t.Errorf("expected 2 branch elements for 4 txs, got %d", len(branch))
		return
	}

	// First element should be tx1 (coinbase's sibling)
	if !bytes.Equal(branch[0], tx1Hash) {
		t.Errorf("branch[0] should be tx1Hash")
	}

	// Second element should be h23 (hash of tx2+tx3)
	h23 := testDoubleSha256(append(tx2Hash, tx3Hash...))
	if !bytes.Equal(branch[1], h23) {
		t.Errorf("branch[1] should be h23")
	}
}

// TestBuildBranch_OddTxCount tests odd number of transactions
func TestBuildBranch_OddTxCount(t *testing.T) {
	builder := NewBuilder()

	// 3 txs: coinbase + 2 others
	// Tree structure (tx2 is duplicated):
	//        root
	//       /    \
	//    h01      h22
	//   /  \     /  \
	//  cb  tx1  tx2  tx2

	tx1Hash := sha256Hash([]byte("tx1"))
	tx2Hash := sha256Hash([]byte("tx2"))

	branch := builder.BuildBranch([][]byte{tx1Hash, tx2Hash})

	// Branch should have 2 elements
	if len(branch) != 2 {
		t.Errorf("expected 2 branch elements for 3 txs, got %d", len(branch))
		return
	}

	// First element should be tx1
	if !bytes.Equal(branch[0], tx1Hash) {
		t.Errorf("branch[0] should be tx1Hash")
	}

	// Second element should be h22 (tx2 hashed with itself for odd count)
	h22 := testDoubleSha256(append(tx2Hash, tx2Hash...))
	if !bytes.Equal(branch[1], h22) {
		t.Errorf("branch[1] should be h22 (tx2 duplicated)")
	}
}

// TestComputeRoot tests merkle root computation
func TestComputeRoot(t *testing.T) {
	builder := NewBuilder()

	coinbaseHash := sha256Hash([]byte("coinbase"))
	tx1Hash := sha256Hash([]byte("tx1"))

	branch := [][]byte{tx1Hash}
	root := builder.ComputeRoot(coinbaseHash, branch)

	// Expected: double-sha256(coinbase || tx1)
	expected := testDoubleSha256(append(coinbaseHash, tx1Hash...))

	if !bytes.Equal(root, expected) {
		t.Errorf("root mismatch\nexpected: %x\ngot:      %x", expected, root)
	}
}

// TestBuildBranch_LargeTxCount tests with many transactions
func TestBuildBranch_LargeTxCount(t *testing.T) {
	builder := NewBuilder()

	// 100 transactions (non-coinbase)
	var txHashes [][]byte
	for i := 0; i < 100; i++ {
		txHashes = append(txHashes, sha256Hash([]byte{byte(i)}))
	}

	branch := builder.BuildBranch(txHashes)

	// For 101 txs (coinbase + 100), we need ceil(log2(101)) = 7 branch elements
	// Actually for 100 non-coinbase txs, tree has 101 leaves, so 7 levels
	expectedBranchLen := 7 // ceil(log2(101))
	if len(branch) != expectedBranchLen {
		t.Errorf("expected %d branch elements for 101 txs, got %d", expectedBranchLen, len(branch))
	}
}

// TestHexConversion tests hex string conversion helpers
func TestHexConversion(t *testing.T) {
	builder := NewBuilder()

	txHex := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	txBytes, _ := hex.DecodeString(txHex)

	branch := builder.BuildBranch([][]byte{txBytes})
	branchHex := builder.BranchToHex(branch)

	if len(branchHex) != 1 {
		t.Errorf("expected 1 hex string, got %d", len(branchHex))
		return
	}

	if branchHex[0] != txHex {
		t.Errorf("hex conversion mismatch")
	}
}

// Helper: single SHA256
func sha256Hash(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

// Helper: double SHA256 (for tests only)
func testDoubleSha256(data []byte) []byte {
	h1 := sha256.Sum256(data)
	h2 := sha256.Sum256(h1[:])
	return h2[:]
}
