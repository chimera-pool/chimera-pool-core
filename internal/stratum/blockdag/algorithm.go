package blockdag

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math/big"
	"sync"

	"golang.org/x/crypto/scrypt"
)

// =============================================================================
// BLOCKDAG SCRPY-VARIANT ALGORITHM
// Custom Scrypt variant for BlockDAG X30/X100 miners
// =============================================================================

// Algorithm constants
const (
	AlgorithmName = "scrpy-variant"

	// Scrypt parameters for BlockDAG
	// These are tuned for the X30/X100 ASIC hardware
	ScryptN = 1024 // CPU/memory cost parameter
	ScryptR = 1    // Block size parameter
	ScryptP = 1    // Parallelization parameter
	KeyLen  = 32   // Output key length

	// Hash sizes
	HashSize   = 32
	HeaderSize = 80 // Standard block header size

	// Difficulty target size
	TargetSize = 32
)

// Errors
var (
	ErrInvalidHeaderSize = errors.New("invalid header size")
	ErrInvalidHashSize   = errors.New("invalid hash size")
	ErrInvalidTarget     = errors.New("invalid target")
	ErrHashAboveTarget   = errors.New("hash above target")
	ErrInvalidNonce      = errors.New("invalid nonce")
)

// =============================================================================
// Scrpy-Variant Algorithm
// =============================================================================

// ScrypyVariant implements the BlockDAG custom Scrypt variant algorithm
type ScrypyVariant struct {
	// Cached values for performance
	mu sync.RWMutex
}

// NewScrypyVariant creates a new Scrpy-variant algorithm instance
func NewScrypyVariant() *ScrypyVariant {
	return &ScrypyVariant{}
}

// Name returns the algorithm name
func (s *ScrypyVariant) Name() string {
	return AlgorithmName
}

// Hash computes the Scrpy-variant hash of the input data
func (s *ScrypyVariant) Hash(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, ErrInvalidHeaderSize
	}

	// BlockDAG Scrpy-variant uses a modified Scrypt with:
	// 1. First pass: standard scrypt
	// 2. Second pass: XOR with reversed first-pass output
	// 3. Final pass: another scrypt round

	// First scrypt pass
	pass1, err := scrypt.Key(data, data, ScryptN, ScryptR, ScryptP, KeyLen)
	if err != nil {
		return nil, err
	}

	// XOR transformation (Scrpy-variant specific)
	transformed := make([]byte, KeyLen)
	for i := 0; i < KeyLen; i++ {
		// XOR with reversed position
		transformed[i] = pass1[i] ^ pass1[KeyLen-1-i]
	}

	// Second scrypt pass with transformed data as salt
	result, err := scrypt.Key(pass1, transformed, ScryptN, ScryptR, ScryptP, KeyLen)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// HashHeader hashes a block header (80 bytes standard format)
func (s *ScrypyVariant) HashHeader(header []byte) ([]byte, error) {
	if len(header) != HeaderSize {
		return nil, ErrInvalidHeaderSize
	}
	return s.Hash(header)
}

// ValidateHash checks if a hash meets the target difficulty
func (s *ScrypyVariant) ValidateHash(hash, target []byte) bool {
	if len(hash) != HashSize || len(target) != TargetSize {
		return false
	}

	// Compare hash to target (hash must be <= target)
	// Both are big-endian 256-bit numbers
	for i := 0; i < HashSize; i++ {
		if hash[i] < target[i] {
			return true
		}
		if hash[i] > target[i] {
			return false
		}
	}
	return true // Equal is valid
}

// ValidateWork validates a complete proof of work
func (s *ScrypyVariant) ValidateWork(header []byte, nonce uint32, target []byte) (bool, []byte, error) {
	if len(header) < HeaderSize-4 {
		return false, nil, ErrInvalidHeaderSize
	}

	// Insert nonce into header (last 4 bytes)
	fullHeader := make([]byte, HeaderSize)
	copy(fullHeader, header[:HeaderSize-4])
	binary.LittleEndian.PutUint32(fullHeader[HeaderSize-4:], nonce)

	// Compute hash
	hash, err := s.HashHeader(fullHeader)
	if err != nil {
		return false, nil, err
	}

	// Validate against target
	valid := s.ValidateHash(hash, target)
	return valid, hash, nil
}

// =============================================================================
// Target/Difficulty Conversion
// =============================================================================

// DifficultyToTarget converts a difficulty value to a 256-bit target
func DifficultyToTarget(difficulty uint64) []byte {
	if difficulty == 0 {
		difficulty = 1
	}

	// Target = MaxTarget / Difficulty
	// MaxTarget = 2^256 - 1 (for simplicity, we use 2^224 as base target)
	maxTarget := new(big.Int)
	maxTarget.SetString("00000000FFFF0000000000000000000000000000000000000000000000000000", 16)

	diff := new(big.Int).SetUint64(difficulty)
	target := new(big.Int).Div(maxTarget, diff)

	// Convert to 32-byte big-endian
	targetBytes := target.Bytes()
	result := make([]byte, TargetSize)
	copy(result[TargetSize-len(targetBytes):], targetBytes)

	return result
}

// TargetToDifficulty converts a 256-bit target to a difficulty value
func TargetToDifficulty(target []byte) uint64 {
	if len(target) != TargetSize {
		return 1
	}

	// Check for zero target
	allZero := true
	for _, b := range target {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		return ^uint64(0) // Maximum difficulty
	}

	maxTarget := new(big.Int)
	maxTarget.SetString("00000000FFFF0000000000000000000000000000000000000000000000000000", 16)

	targetInt := new(big.Int).SetBytes(target)
	if targetInt.Sign() == 0 {
		return ^uint64(0)
	}

	diff := new(big.Int).Div(maxTarget, targetInt)

	// Clamp to uint64
	if diff.BitLen() > 64 {
		return ^uint64(0)
	}

	return diff.Uint64()
}

// CompactToTarget converts compact (nBits) format to full target
func CompactToTarget(compact uint32) []byte {
	// Extract exponent and mantissa
	size := int(compact >> 24)
	mantissa := compact & 0x007FFFFF

	// Handle negative flag
	if compact&0x00800000 != 0 {
		mantissa = 0
	}

	target := make([]byte, TargetSize)

	if size <= 3 {
		mantissa >>= 8 * (3 - size)
		target[TargetSize-1] = byte(mantissa)
		target[TargetSize-2] = byte(mantissa >> 8)
		target[TargetSize-3] = byte(mantissa >> 16)
	} else {
		pos := TargetSize - size
		if pos >= 0 && pos < TargetSize-2 {
			target[pos] = byte(mantissa >> 16)
			if pos+1 < TargetSize {
				target[pos+1] = byte(mantissa >> 8)
			}
			if pos+2 < TargetSize {
				target[pos+2] = byte(mantissa)
			}
		}
	}

	return target
}

// TargetToCompact converts full target to compact (nBits) format
func TargetToCompact(target []byte) uint32 {
	// Find first non-zero byte
	start := 0
	for start < len(target) && target[start] == 0 {
		start++
	}

	if start == len(target) {
		return 0
	}

	size := len(target) - start
	var mantissa uint32

	if size >= 3 {
		mantissa = uint32(target[start])<<16 | uint32(target[start+1])<<8 | uint32(target[start+2])
	} else if size == 2 {
		mantissa = uint32(target[start])<<16 | uint32(target[start+1])<<8
	} else {
		mantissa = uint32(target[start]) << 16
	}

	// Adjust for sign bit
	if mantissa&0x00800000 != 0 {
		mantissa >>= 8
		size++
	}

	return uint32(size)<<24 | mantissa
}

// =============================================================================
// Block Template
// =============================================================================

// BlockTemplate represents a mining job template
type BlockTemplate struct {
	Version      uint32
	PrevHash     [32]byte
	MerkleRoot   [32]byte
	Timestamp    uint32
	Bits         uint32 // Compact target
	Height       uint64
	Coinbase     []byte
	Transactions [][]byte

	// Derived fields
	Target     []byte
	Difficulty uint64
}

// NewBlockTemplate creates a new block template
func NewBlockTemplate() *BlockTemplate {
	return &BlockTemplate{
		Target: make([]byte, TargetSize),
	}
}

// SetBits sets the compact target and derives full target/difficulty
func (bt *BlockTemplate) SetBits(bits uint32) {
	bt.Bits = bits
	bt.Target = CompactToTarget(bits)
	bt.Difficulty = TargetToDifficulty(bt.Target)
}

// BuildHeader builds an 80-byte block header (without nonce)
func (bt *BlockTemplate) BuildHeader() []byte {
	header := make([]byte, HeaderSize)

	// Version (4 bytes, little-endian)
	binary.LittleEndian.PutUint32(header[0:4], bt.Version)

	// Previous block hash (32 bytes)
	copy(header[4:36], bt.PrevHash[:])

	// Merkle root (32 bytes)
	copy(header[36:68], bt.MerkleRoot[:])

	// Timestamp (4 bytes, little-endian)
	binary.LittleEndian.PutUint32(header[68:72], bt.Timestamp)

	// Bits (4 bytes, little-endian)
	binary.LittleEndian.PutUint32(header[72:76], bt.Bits)

	// Nonce placeholder (4 bytes) - will be filled by miner
	// Leave as zeros

	return header
}

// BuildHeaderWithNonce builds complete header with nonce
func (bt *BlockTemplate) BuildHeaderWithNonce(nonce uint32) []byte {
	header := bt.BuildHeader()
	binary.LittleEndian.PutUint32(header[76:80], nonce)
	return header
}

// =============================================================================
// Share Validation
// =============================================================================

// ShareValidator validates mining shares against targets
type ShareValidator struct {
	algo *ScrypyVariant
}

// NewShareValidator creates a new share validator
func NewShareValidator() *ShareValidator {
	return &ShareValidator{
		algo: NewScrypyVariant(),
	}
}

// ValidateShare validates a submitted share
func (sv *ShareValidator) ValidateShare(header []byte, nonce uint32, shareTarget, blockTarget []byte) (shareValid, blockValid bool, hash []byte, err error) {
	// Build full header with nonce
	if len(header) < HeaderSize-4 {
		return false, false, nil, ErrInvalidHeaderSize
	}

	fullHeader := make([]byte, HeaderSize)
	copy(fullHeader, header[:HeaderSize-4])
	binary.LittleEndian.PutUint32(fullHeader[HeaderSize-4:], nonce)

	// Compute hash
	hash, err = sv.algo.HashHeader(fullHeader)
	if err != nil {
		return false, false, nil, err
	}

	// Check against share target (miner's difficulty)
	shareValid = sv.algo.ValidateHash(hash, shareTarget)

	// Check against block target (network difficulty)
	if shareValid {
		blockValid = sv.algo.ValidateHash(hash, blockTarget)
	}

	return shareValid, blockValid, hash, nil
}

// QuickValidate performs a quick hash comparison without full recomputation
func (sv *ShareValidator) QuickValidate(hash, target []byte) bool {
	return sv.algo.ValidateHash(hash, target)
}

// =============================================================================
// Merkle Tree
// =============================================================================

// ComputeMerkleRoot computes the merkle root from transaction hashes
func ComputeMerkleRoot(txHashes [][]byte) []byte {
	if len(txHashes) == 0 {
		return make([]byte, 32)
	}

	if len(txHashes) == 1 {
		return txHashes[0]
	}

	// Copy hashes
	level := make([][]byte, len(txHashes))
	for i, h := range txHashes {
		level[i] = make([]byte, 32)
		copy(level[i], h)
	}

	// Build tree
	for len(level) > 1 {
		if len(level)%2 != 0 {
			// Duplicate last hash if odd number
			level = append(level, level[len(level)-1])
		}

		newLevel := make([][]byte, len(level)/2)
		for i := 0; i < len(level); i += 2 {
			combined := append(level[i], level[i+1]...)
			// Double SHA256 for merkle tree
			hash := doubleSHA256(combined)
			newLevel[i/2] = hash
		}
		level = newLevel
	}

	return level[0]
}

// doubleSHA256 computes SHA256(SHA256(data))
func doubleSHA256(data []byte) []byte {
	first := sha256Sum(data)
	return sha256Sum(first)
}

// sha256Sum computes SHA256 hash
func sha256Sum(data []byte) []byte {
	// Using a simple implementation for consistency
	// In production, use crypto/sha256
	h := newSHA256()
	h.Write(data)
	result := make([]byte, 32)
	copy(result, h.Sum(nil))
	return result
}

// Minimal SHA256 implementation (same as in noise package)
type sha256State struct {
	h      [8]uint32
	x      [64]byte
	nx     int
	length uint64
}

func newSHA256() *sha256State {
	s := &sha256State{}
	s.Reset()
	return s
}

func (s *sha256State) Reset() {
	s.h[0] = 0x6a09e667
	s.h[1] = 0xbb67ae85
	s.h[2] = 0x3c6ef372
	s.h[3] = 0xa54ff53a
	s.h[4] = 0x510e527f
	s.h[5] = 0x9b05688c
	s.h[6] = 0x1f83d9ab
	s.h[7] = 0x5be0cd19
	s.nx = 0
	s.length = 0
}

func (s *sha256State) Write(p []byte) (int, error) {
	nn := len(p)
	s.length += uint64(nn)
	if s.nx > 0 {
		n := copy(s.x[s.nx:], p)
		s.nx += n
		if s.nx == 64 {
			s.block(s.x[:])
			s.nx = 0
		}
		p = p[n:]
	}
	for len(p) >= 64 {
		s.block(p[:64])
		p = p[64:]
	}
	if len(p) > 0 {
		s.nx = copy(s.x[:], p)
	}
	return nn, nil
}

func (s *sha256State) Sum(in []byte) []byte {
	s0 := *s
	hash := s0.checkSum()
	return append(in, hash[:]...)
}

func (s *sha256State) checkSum() [32]byte {
	length := s.length
	var tmp [64]byte
	tmp[0] = 0x80
	if length%64 < 56 {
		s.Write(tmp[0 : 56-length%64])
	} else {
		s.Write(tmp[0 : 64+56-length%64])
	}
	length <<= 3
	for i := uint(0); i < 8; i++ {
		tmp[i] = byte(length >> (56 - 8*i))
	}
	s.Write(tmp[0:8])
	var digest [32]byte
	for i := 0; i < 8; i++ {
		digest[i*4] = byte(s.h[i] >> 24)
		digest[i*4+1] = byte(s.h[i] >> 16)
		digest[i*4+2] = byte(s.h[i] >> 8)
		digest[i*4+3] = byte(s.h[i])
	}
	return digest
}

var sha256K = [64]uint32{
	0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
	0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
	0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
	0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
	0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
	0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
	0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
	0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
}

func (s *sha256State) block(p []byte) {
	var w [64]uint32
	for i := 0; i < 16; i++ {
		j := i * 4
		w[i] = uint32(p[j])<<24 | uint32(p[j+1])<<16 | uint32(p[j+2])<<8 | uint32(p[j+3])
	}
	for i := 16; i < 64; i++ {
		v1 := w[i-2]
		t1 := (v1>>17 | v1<<15) ^ (v1>>19 | v1<<13) ^ (v1 >> 10)
		v2 := w[i-15]
		t2 := (v2>>7 | v2<<25) ^ (v2>>18 | v2<<14) ^ (v2 >> 3)
		w[i] = t1 + w[i-7] + t2 + w[i-16]
	}
	a, b, c, d, e, f, g, h := s.h[0], s.h[1], s.h[2], s.h[3], s.h[4], s.h[5], s.h[6], s.h[7]
	for i := 0; i < 64; i++ {
		t1 := h + ((e>>6 | e<<26) ^ (e>>11 | e<<21) ^ (e>>25 | e<<7)) + ((e & f) ^ (^e & g)) + sha256K[i] + w[i]
		t2 := ((a>>2 | a<<30) ^ (a>>13 | a<<19) ^ (a>>22 | a<<10)) + ((a & b) ^ (a & c) ^ (b & c))
		h = g
		g = f
		f = e
		e = d + t1
		d = c
		c = b
		b = a
		a = t1 + t2
	}
	s.h[0] += a
	s.h[1] += b
	s.h[2] += c
	s.h[3] += d
	s.h[4] += e
	s.h[5] += f
	s.h[6] += g
	s.h[7] += h
}

// =============================================================================
// Utility Functions
// =============================================================================

// ReverseBytes reverses a byte slice in place
func ReverseBytes(b []byte) {
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
}

// CompareHashes compares two hashes (returns -1 if a < b, 0 if equal, 1 if a > b)
func CompareHashes(a, b []byte) int {
	return bytes.Compare(a, b)
}
