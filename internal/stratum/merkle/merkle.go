// Package merkle implements merkle tree computation for stratum mining
package merkle

import (
	"crypto/sha256"
	"encoding/hex"
)

// Builder implements merkle tree building for stratum
type Builder struct{}

// NewBuilder creates a new merkle tree builder
func NewBuilder() *Builder {
	return &Builder{}
}

// BuildBranch computes the merkle branch for coinbase at index 0
// txHashes should contain only the non-coinbase transaction hashes
// The coinbase hash is computed by the miner and combined with this branch
func (b *Builder) BuildBranch(txHashes [][]byte) [][]byte {
	if len(txHashes) == 0 {
		return [][]byte{}
	}

	var branch [][]byte

	// Start with all transaction hashes
	// The miner will prepend the coinbase hash
	hashes := make([][]byte, len(txHashes))
	copy(hashes, txHashes)

	// Build merkle tree level by level
	// At each level, we need the sibling hash for the coinbase path
	for len(hashes) > 0 {
		// The first hash is the sibling of coinbase at this level
		branch = append(branch, hashes[0])

		// If only one hash left, we're done
		if len(hashes) == 1 {
			break
		}

		// Compute next level (pairing up hashes)
		var nextLevel [][]byte

		// Start from index 1 since index 0 was coinbase's sibling
		// But we need to compute the hash of the rest of the tree
		for i := 1; i < len(hashes); i += 2 {
			left := hashes[i]
			var right []byte
			if i+1 < len(hashes) {
				right = hashes[i+1]
			} else {
				right = left // Duplicate if odd
			}
			combined := doubleSha256(append(left, right...))
			nextLevel = append(nextLevel, combined)
		}

		hashes = nextLevel
	}

	return branch
}

// ComputeRoot computes the merkle root given a coinbase hash and branch
func (b *Builder) ComputeRoot(coinbaseHash []byte, branch [][]byte) []byte {
	if len(branch) == 0 {
		return coinbaseHash
	}

	current := coinbaseHash
	for _, sibling := range branch {
		// Coinbase is always on the left side at each level
		current = doubleSha256(append(current, sibling...))
	}
	return current
}

// BranchToHex converts a branch to hex strings for JSON serialization
func (b *Builder) BranchToHex(branch [][]byte) []string {
	result := make([]string, len(branch))
	for i, h := range branch {
		result[i] = hex.EncodeToString(h)
	}
	return result
}

// HexToBranch converts hex strings back to branch bytes
func (b *Builder) HexToBranch(hexBranch []string) ([][]byte, error) {
	result := make([][]byte, len(hexBranch))
	for i, h := range hexBranch {
		bytes, err := hex.DecodeString(h)
		if err != nil {
			return nil, err
		}
		result[i] = bytes
	}
	return result, nil
}

// doubleSha256 computes SHA256(SHA256(data))
func doubleSha256(data []byte) []byte {
	h1 := sha256.Sum256(data)
	h2 := sha256.Sum256(h1[:])
	return h2[:]
}
