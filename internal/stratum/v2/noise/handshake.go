package noise

import (
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
	"sync"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
)

// =============================================================================
// STRATUM V2 NOISE PROTOCOL IMPLEMENTATION
// Implements Noise_NX_25519_ChaChaPoly_SHA256 for secure mining communication
// =============================================================================

// Protocol constants
const (
	// Noise protocol name for Stratum V2
	ProtocolName = "Noise_NX_25519_ChaChaPoly_SHA256"

	// Key sizes
	DHKeySize  = 32 // X25519 key size
	SymKeySize = 32 // ChaCha20-Poly1305 key size
	NonceSize  = 12 // AEAD nonce size
	TagSize    = 16 // Poly1305 tag size
	MaxNonce   = ^uint64(0) - 1

	// Handshake patterns
	PatternNX = "NX" // No static key for initiator, static key for responder
)

// Errors
var (
	ErrInvalidKeySize   = errors.New("invalid key size")
	ErrHandshakeFailed  = errors.New("handshake failed")
	ErrInvalidMessage   = errors.New("invalid message")
	ErrNonceOverflow    = errors.New("nonce overflow - rekey required")
	ErrDecryptionFailed = errors.New("decryption failed")
	ErrNotEstablished   = errors.New("secure channel not established")
	ErrInvalidPublicKey = errors.New("invalid public key")
)

// =============================================================================
// Key Pair
// =============================================================================

// KeyPair represents an X25519 key pair
type KeyPair struct {
	PrivateKey [DHKeySize]byte
	PublicKey  [DHKeySize]byte
}

// GenerateKeyPair generates a new X25519 key pair
func GenerateKeyPair() (*KeyPair, error) {
	kp := &KeyPair{}

	// Generate random private key
	if _, err := io.ReadFull(rand.Reader, kp.PrivateKey[:]); err != nil {
		return nil, err
	}

	// Clamp private key for X25519
	kp.PrivateKey[0] &= 248
	kp.PrivateKey[31] &= 127
	kp.PrivateKey[31] |= 64

	// Derive public key
	curve25519.ScalarBaseMult(&kp.PublicKey, &kp.PrivateKey)

	return kp, nil
}

// DH performs X25519 Diffie-Hellman
func (kp *KeyPair) DH(theirPublic [DHKeySize]byte) ([DHKeySize]byte, error) {
	var shared [DHKeySize]byte
	curve25519.ScalarMult(&shared, &kp.PrivateKey, &theirPublic)

	// Check for all-zero output (invalid public key)
	allZero := true
	for _, b := range shared {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		return shared, ErrInvalidPublicKey
	}

	return shared, nil
}

// =============================================================================
// Cipher State
// =============================================================================

// CipherState manages symmetric encryption state
type CipherState struct {
	key   [SymKeySize]byte
	nonce uint64
	aead  cipher.AEAD
	mu    sync.Mutex
}

// NewCipherState creates a new cipher state with the given key
func NewCipherState(key [SymKeySize]byte) (*CipherState, error) {
	aead, err := chacha20poly1305.New(key[:])
	if err != nil {
		return nil, err
	}

	return &CipherState{
		key:   key,
		nonce: 0,
		aead:  aead,
	}, nil
}

// Encrypt encrypts plaintext with associated data
func (cs *CipherState) Encrypt(plaintext, ad []byte) ([]byte, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.nonce >= MaxNonce {
		return nil, ErrNonceOverflow
	}

	// Build nonce (little-endian uint64 padded to 12 bytes)
	nonce := make([]byte, NonceSize)
	nonce[0] = byte(cs.nonce)
	nonce[1] = byte(cs.nonce >> 8)
	nonce[2] = byte(cs.nonce >> 16)
	nonce[3] = byte(cs.nonce >> 24)
	nonce[4] = byte(cs.nonce >> 32)
	nonce[5] = byte(cs.nonce >> 40)
	nonce[6] = byte(cs.nonce >> 48)
	nonce[7] = byte(cs.nonce >> 56)

	cs.nonce++

	// Encrypt
	ciphertext := cs.aead.Seal(nil, nonce, plaintext, ad)
	return ciphertext, nil
}

// Decrypt decrypts ciphertext with associated data
func (cs *CipherState) Decrypt(ciphertext, ad []byte) ([]byte, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.nonce >= MaxNonce {
		return nil, ErrNonceOverflow
	}

	// Build nonce
	nonce := make([]byte, NonceSize)
	nonce[0] = byte(cs.nonce)
	nonce[1] = byte(cs.nonce >> 8)
	nonce[2] = byte(cs.nonce >> 16)
	nonce[3] = byte(cs.nonce >> 24)
	nonce[4] = byte(cs.nonce >> 32)
	nonce[5] = byte(cs.nonce >> 40)
	nonce[6] = byte(cs.nonce >> 48)
	nonce[7] = byte(cs.nonce >> 56)

	cs.nonce++

	// Decrypt
	plaintext, err := cs.aead.Open(nil, nonce, ciphertext, ad)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// GetNonce returns the current nonce value
func (cs *CipherState) GetNonce() uint64 {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.nonce
}

// =============================================================================
// Symmetric State (for handshake)
// =============================================================================

// SymmetricState manages handshake state
type SymmetricState struct {
	chainingKey [SymKeySize]byte
	h           [32]byte // handshake hash
	cipher      *CipherState
}

// NewSymmetricState initializes symmetric state with protocol name
func NewSymmetricState() *SymmetricState {
	ss := &SymmetricState{}

	// Initialize h with protocol name hash (or protocol name if <= 32 bytes)
	protocolBytes := []byte(ProtocolName)
	if len(protocolBytes) <= 32 {
		copy(ss.h[:], protocolBytes)
	} else {
		ss.h = sha256Hash(protocolBytes)
	}

	// ck = h initially
	ss.chainingKey = ss.h

	return ss
}

// MixKey derives new keys from input key material
func (ss *SymmetricState) MixKey(inputKeyMaterial []byte) {
	// HKDF with ck as salt
	tempK1, tempK2 := hkdfDerive(ss.chainingKey[:], inputKeyMaterial)
	ss.chainingKey = tempK1

	// Initialize cipher with tempK2
	ss.cipher, _ = NewCipherState(tempK2)
}

// MixHash mixes data into the handshake hash
func (ss *SymmetricState) MixHash(data []byte) {
	combined := append(ss.h[:], data...)
	ss.h = sha256Hash(combined)
}

// EncryptAndHash encrypts plaintext and mixes ciphertext into hash
func (ss *SymmetricState) EncryptAndHash(plaintext []byte) ([]byte, error) {
	if ss.cipher == nil {
		// Before first MixKey, just return plaintext
		ss.MixHash(plaintext)
		return plaintext, nil
	}

	ciphertext, err := ss.cipher.Encrypt(plaintext, ss.h[:])
	if err != nil {
		return nil, err
	}

	ss.MixHash(ciphertext)
	return ciphertext, nil
}

// DecryptAndHash decrypts ciphertext and mixes it into hash
func (ss *SymmetricState) DecryptAndHash(ciphertext []byte) ([]byte, error) {
	if ss.cipher == nil {
		// Before first MixKey, ciphertext is plaintext
		ss.MixHash(ciphertext)
		return ciphertext, nil
	}

	plaintext, err := ss.cipher.Decrypt(ciphertext, ss.h[:])
	if err != nil {
		return nil, err
	}

	ss.MixHash(ciphertext)
	return plaintext, nil
}

// Split derives the final transport keys
func (ss *SymmetricState) Split() (*CipherState, *CipherState, error) {
	tempK1, tempK2 := hkdfDerive(ss.chainingKey[:], nil)

	c1, err := NewCipherState(tempK1)
	if err != nil {
		return nil, nil, err
	}

	c2, err := NewCipherState(tempK2)
	if err != nil {
		return nil, nil, err
	}

	return c1, c2, nil
}

// =============================================================================
// Handshake State (NX Pattern)
// =============================================================================

// HandshakeState manages the Noise NX handshake
type HandshakeState struct {
	ss              *SymmetricState
	localStatic     *KeyPair
	localEphemeral  *KeyPair
	remoteStatic    [DHKeySize]byte
	remoteEphemeral [DHKeySize]byte
	initiator       bool
	messageIndex    int
}

// NewInitiatorHandshake creates a handshake state for the client (miner)
func NewInitiatorHandshake() (*HandshakeState, error) {
	ephemeral, err := GenerateKeyPair()
	if err != nil {
		return nil, err
	}

	hs := &HandshakeState{
		ss:             NewSymmetricState(),
		localEphemeral: ephemeral,
		initiator:      true,
		messageIndex:   0,
	}

	// NX pattern: initiator has no static key
	// -> e
	// <- e, ee, s, es

	return hs, nil
}

// NewResponderHandshake creates a handshake state for the server (pool)
func NewResponderHandshake(staticKey *KeyPair) (*HandshakeState, error) {
	if staticKey == nil {
		return nil, ErrInvalidKeySize
	}

	ephemeral, err := GenerateKeyPair()
	if err != nil {
		return nil, err
	}

	hs := &HandshakeState{
		ss:             NewSymmetricState(),
		localStatic:    staticKey,
		localEphemeral: ephemeral,
		initiator:      false,
		messageIndex:   0,
	}

	return hs, nil
}

// WriteMessage generates the next handshake message
func (hs *HandshakeState) WriteMessage(payload []byte) ([]byte, error) {
	var message []byte

	if hs.initiator {
		// Initiator message patterns
		switch hs.messageIndex {
		case 0:
			// -> e: Send ephemeral public key
			hs.ss.MixHash(hs.localEphemeral.PublicKey[:])
			message = append(message, hs.localEphemeral.PublicKey[:]...)

			// Encrypt payload (no key yet, so plaintext)
			encPayload, err := hs.ss.EncryptAndHash(payload)
			if err != nil {
				return nil, err
			}
			message = append(message, encPayload...)
		default:
			return nil, ErrHandshakeFailed
		}
	} else {
		// Responder message patterns
		switch hs.messageIndex {
		case 0:
			// <- e: Send ephemeral public key
			hs.ss.MixHash(hs.localEphemeral.PublicKey[:])
			message = append(message, hs.localEphemeral.PublicKey[:]...)

			// ee: DH(e, re)
			shared, err := hs.localEphemeral.DH(hs.remoteEphemeral)
			if err != nil {
				return nil, err
			}
			hs.ss.MixKey(shared[:])

			// s: Send static public key (encrypted)
			encStatic, err := hs.ss.EncryptAndHash(hs.localStatic.PublicKey[:])
			if err != nil {
				return nil, err
			}
			message = append(message, encStatic...)

			// es: DH(s, re)
			shared, err = hs.localStatic.DH(hs.remoteEphemeral)
			if err != nil {
				return nil, err
			}
			hs.ss.MixKey(shared[:])

			// Encrypt payload
			encPayload, err := hs.ss.EncryptAndHash(payload)
			if err != nil {
				return nil, err
			}
			message = append(message, encPayload...)
		default:
			return nil, ErrHandshakeFailed
		}
	}

	hs.messageIndex++
	return message, nil
}

// ReadMessage processes an incoming handshake message
func (hs *HandshakeState) ReadMessage(message []byte) ([]byte, error) {
	if hs.initiator {
		// Initiator reading responder's message
		switch hs.messageIndex {
		case 1:
			// <- e, ee, s, es
			if len(message) < DHKeySize {
				return nil, ErrInvalidMessage
			}

			// e: Read remote ephemeral
			copy(hs.remoteEphemeral[:], message[:DHKeySize])
			hs.ss.MixHash(hs.remoteEphemeral[:])
			message = message[DHKeySize:]

			// ee: DH(e, re)
			shared, err := hs.localEphemeral.DH(hs.remoteEphemeral)
			if err != nil {
				return nil, err
			}
			hs.ss.MixKey(shared[:])

			// s: Read remote static (encrypted)
			if len(message) < DHKeySize+TagSize {
				return nil, ErrInvalidMessage
			}
			decStatic, err := hs.ss.DecryptAndHash(message[:DHKeySize+TagSize])
			if err != nil {
				return nil, err
			}
			copy(hs.remoteStatic[:], decStatic)
			message = message[DHKeySize+TagSize:]

			// es: DH(e, rs)
			shared, err = hs.localEphemeral.DH(hs.remoteStatic)
			if err != nil {
				return nil, err
			}
			hs.ss.MixKey(shared[:])

			// Decrypt payload
			payload, err := hs.ss.DecryptAndHash(message)
			if err != nil {
				return nil, err
			}

			hs.messageIndex++
			return payload, nil
		default:
			return nil, ErrHandshakeFailed
		}
	} else {
		// Responder reading initiator's message
		switch hs.messageIndex {
		case 0:
			// -> e
			if len(message) < DHKeySize {
				return nil, ErrInvalidMessage
			}

			// e: Read remote ephemeral
			copy(hs.remoteEphemeral[:], message[:DHKeySize])
			hs.ss.MixHash(hs.remoteEphemeral[:])
			message = message[DHKeySize:]

			// Decrypt payload (no key yet)
			payload, err := hs.ss.DecryptAndHash(message)
			if err != nil {
				return nil, err
			}

			return payload, nil
		default:
			return nil, ErrHandshakeFailed
		}
	}
}

// IsComplete returns true if the handshake is complete
func (hs *HandshakeState) IsComplete() bool {
	if hs.initiator {
		return hs.messageIndex >= 2
	}
	return hs.messageIndex >= 1
}

// Split returns the transport cipher states after handshake completion
// Returns (sendCipher, recvCipher) - order depends on initiator/responder role
func (hs *HandshakeState) Split() (*CipherState, *CipherState, error) {
	if !hs.IsComplete() {
		return nil, nil, ErrNotEstablished
	}
	c1, c2, err := hs.ss.Split()
	if err != nil {
		return nil, nil, err
	}

	// Initiator: c1=send, c2=recv
	// Responder: c2=send, c1=recv (opposite order)
	if hs.initiator {
		return c1, c2, nil
	}
	return c2, c1, nil
}

// GetRemoteStatic returns the remote party's static public key
func (hs *HandshakeState) GetRemoteStatic() [DHKeySize]byte {
	return hs.remoteStatic
}

// =============================================================================
// Secure Channel
// =============================================================================

// SecureChannel represents an established encrypted channel
type SecureChannel struct {
	sendCipher *CipherState
	recvCipher *CipherState
	mu         sync.Mutex
}

// NewSecureChannel creates a secure channel from handshake result
func NewSecureChannel(sendCipher, recvCipher *CipherState) *SecureChannel {
	return &SecureChannel{
		sendCipher: sendCipher,
		recvCipher: recvCipher,
	}
}

// Encrypt encrypts a message for sending
func (sc *SecureChannel) Encrypt(plaintext []byte) ([]byte, error) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.sendCipher.Encrypt(plaintext, nil)
}

// Decrypt decrypts a received message
func (sc *SecureChannel) Decrypt(ciphertext []byte) ([]byte, error) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.recvCipher.Decrypt(ciphertext, nil)
}

// IsEstablished returns true if the channel is ready
func (sc *SecureChannel) IsEstablished() bool {
	return sc.sendCipher != nil && sc.recvCipher != nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// sha256Hash computes SHA-256 hash
func sha256Hash(data []byte) [32]byte {
	// Using golang.org/x/crypto for consistency
	// In production, use crypto/sha256
	var result [32]byte
	h := newSHA256()
	h.Write(data)
	copy(result[:], h.Sum(nil))
	return result
}

// hkdfDerive performs HKDF-SHA256 key derivation
func hkdfDerive(salt, ikm []byte) ([32]byte, [32]byte) {
	// HKDF-Extract
	prk := hmacSHA256(salt, ikm)

	// HKDF-Expand for two 32-byte keys
	t1 := hmacSHA256(prk[:], []byte{0x01})
	t2 := hmacSHA256(prk[:], append(t1[:], 0x02))

	return t1, t2
}

// hmacSHA256 computes HMAC-SHA256
func hmacSHA256(key, data []byte) [32]byte {
	// Simplified HMAC implementation
	blockSize := 64

	// Key padding
	if len(key) > blockSize {
		h := sha256Hash(key)
		key = h[:]
	}
	paddedKey := make([]byte, blockSize)
	copy(paddedKey, key)

	// Inner and outer padding
	ipad := make([]byte, blockSize)
	opad := make([]byte, blockSize)
	for i := 0; i < blockSize; i++ {
		ipad[i] = paddedKey[i] ^ 0x36
		opad[i] = paddedKey[i] ^ 0x5c
	}

	// Inner hash
	inner := sha256Hash(append(ipad, data...))

	// Outer hash
	return sha256Hash(append(opad, inner[:]...))
}

// SHA256 state (minimal implementation)
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
	// Make a copy to avoid modifying state
	s0 := *s
	hash := s0.checkSum()
	return append(in, hash[:]...)
}

func (s *sha256State) checkSum() [32]byte {
	length := s.length

	// Padding
	var tmp [64]byte
	tmp[0] = 0x80
	if length%64 < 56 {
		s.Write(tmp[0 : 56-length%64])
	} else {
		s.Write(tmp[0 : 64+56-length%64])
	}

	// Length in bits
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
	0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5,
	0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
	0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3,
	0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
	0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc,
	0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
	0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7,
	0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
	0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13,
	0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
	0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3,
	0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
	0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5,
	0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
	0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208,
	0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
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
