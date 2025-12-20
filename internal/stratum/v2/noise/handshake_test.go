package noise

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// TDD TESTS FOR NOISE PROTOCOL IMPLEMENTATION
// =============================================================================

// -----------------------------------------------------------------------------
// Key Pair Tests
// -----------------------------------------------------------------------------

func TestGenerateKeyPair(t *testing.T) {
	kp, err := GenerateKeyPair()
	require.NoError(t, err)
	assert.NotNil(t, kp)

	// Keys should not be all zeros
	allZeroPrivate := true
	allZeroPublic := true
	for i := 0; i < DHKeySize; i++ {
		if kp.PrivateKey[i] != 0 {
			allZeroPrivate = false
		}
		if kp.PublicKey[i] != 0 {
			allZeroPublic = false
		}
	}
	assert.False(t, allZeroPrivate, "private key should not be all zeros")
	assert.False(t, allZeroPublic, "public key should not be all zeros")
}

func TestGenerateKeyPair_Unique(t *testing.T) {
	kp1, err := GenerateKeyPair()
	require.NoError(t, err)

	kp2, err := GenerateKeyPair()
	require.NoError(t, err)

	// Keys should be different
	assert.NotEqual(t, kp1.PrivateKey, kp2.PrivateKey)
	assert.NotEqual(t, kp1.PublicKey, kp2.PublicKey)
}

func TestKeyPair_DH(t *testing.T) {
	// Generate two key pairs
	alice, err := GenerateKeyPair()
	require.NoError(t, err)

	bob, err := GenerateKeyPair()
	require.NoError(t, err)

	// DH should produce same shared secret
	sharedAlice, err := alice.DH(bob.PublicKey)
	require.NoError(t, err)

	sharedBob, err := bob.DH(alice.PublicKey)
	require.NoError(t, err)

	assert.Equal(t, sharedAlice, sharedBob, "DH shared secrets should match")
}

func TestKeyPair_DH_InvalidPublicKey(t *testing.T) {
	kp, err := GenerateKeyPair()
	require.NoError(t, err)

	// All-zero public key should fail
	var zeroKey [DHKeySize]byte
	_, err = kp.DH(zeroKey)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidPublicKey, err)
}

// -----------------------------------------------------------------------------
// Cipher State Tests
// -----------------------------------------------------------------------------

func TestCipherState_EncryptDecrypt(t *testing.T) {
	var key [SymKeySize]byte
	for i := 0; i < SymKeySize; i++ {
		key[i] = byte(i)
	}

	cs, err := NewCipherState(key)
	require.NoError(t, err)

	plaintext := []byte("Hello, BlockDAG X100!")
	ad := []byte("associated data")

	ciphertext, err := cs.Encrypt(plaintext, ad)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, ciphertext)

	// Create new cipher state with same key for decryption
	cs2, err := NewCipherState(key)
	require.NoError(t, err)

	decrypted, err := cs2.Decrypt(ciphertext, ad)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestCipherState_NonceIncrement(t *testing.T) {
	var key [SymKeySize]byte
	cs, err := NewCipherState(key)
	require.NoError(t, err)

	assert.Equal(t, uint64(0), cs.GetNonce())

	_, err = cs.Encrypt([]byte("test"), nil)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), cs.GetNonce())

	_, err = cs.Encrypt([]byte("test"), nil)
	require.NoError(t, err)
	assert.Equal(t, uint64(2), cs.GetNonce())
}

func TestCipherState_DecryptWrongAD(t *testing.T) {
	var key [SymKeySize]byte
	cs1, _ := NewCipherState(key)
	cs2, _ := NewCipherState(key)

	plaintext := []byte("secret message")
	ciphertext, _ := cs1.Encrypt(plaintext, []byte("correct AD"))

	// Decrypt with wrong AD should fail
	_, err := cs2.Decrypt(ciphertext, []byte("wrong AD"))
	assert.Error(t, err)
}

// -----------------------------------------------------------------------------
// Symmetric State Tests
// -----------------------------------------------------------------------------

func TestSymmetricState_MixHash(t *testing.T) {
	ss := NewSymmetricState()
	initialH := ss.h

	ss.MixHash([]byte("test data"))

	assert.NotEqual(t, initialH, ss.h, "hash should change after MixHash")
}

func TestSymmetricState_MixKey(t *testing.T) {
	ss := NewSymmetricState()
	initialCK := ss.chainingKey

	ss.MixKey([]byte("input key material"))

	assert.NotEqual(t, initialCK, ss.chainingKey, "chaining key should change")
	assert.NotNil(t, ss.cipher, "cipher should be initialized")
}

func TestSymmetricState_EncryptAndHash_BeforeKey(t *testing.T) {
	ss := NewSymmetricState()

	// Before MixKey, EncryptAndHash should return plaintext
	plaintext := []byte("test message")
	result, err := ss.EncryptAndHash(plaintext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, result)
}

func TestSymmetricState_EncryptAndHash_AfterKey(t *testing.T) {
	ss := NewSymmetricState()
	ss.MixKey([]byte("key material"))

	plaintext := []byte("test message")
	ciphertext, err := ss.EncryptAndHash(plaintext)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, ciphertext, "should be encrypted")
}

func TestSymmetricState_Split(t *testing.T) {
	ss := NewSymmetricState()
	ss.MixKey([]byte("key material"))

	c1, c2, err := ss.Split()
	require.NoError(t, err)
	assert.NotNil(t, c1)
	assert.NotNil(t, c2)
}

// -----------------------------------------------------------------------------
// Handshake Tests
// -----------------------------------------------------------------------------

func TestNewInitiatorHandshake(t *testing.T) {
	hs, err := NewInitiatorHandshake()
	require.NoError(t, err)
	assert.True(t, hs.initiator)
	assert.NotNil(t, hs.localEphemeral)
	assert.Nil(t, hs.localStatic) // NX pattern: initiator has no static
}

func TestNewResponderHandshake(t *testing.T) {
	staticKey, err := GenerateKeyPair()
	require.NoError(t, err)

	hs, err := NewResponderHandshake(staticKey)
	require.NoError(t, err)
	assert.False(t, hs.initiator)
	assert.NotNil(t, hs.localStatic)
	assert.NotNil(t, hs.localEphemeral)
}

func TestNewResponderHandshake_NilStaticKey(t *testing.T) {
	_, err := NewResponderHandshake(nil)
	assert.Error(t, err)
}

func TestHandshake_NX_FullExchange(t *testing.T) {
	// Server (pool) has static key
	serverStatic, err := GenerateKeyPair()
	require.NoError(t, err)

	// Create handshake states
	initiator, err := NewInitiatorHandshake()
	require.NoError(t, err)

	responder, err := NewResponderHandshake(serverStatic)
	require.NoError(t, err)

	// Message 1: Initiator -> Responder (-> e)
	msg1, err := initiator.WriteMessage([]byte("hello from miner"))
	require.NoError(t, err)
	assert.True(t, len(msg1) >= DHKeySize, "message should contain ephemeral key")

	// Responder reads message 1
	payload1, err := responder.ReadMessage(msg1)
	require.NoError(t, err)
	assert.Equal(t, []byte("hello from miner"), payload1)

	// Message 2: Responder -> Initiator (<- e, ee, s, es)
	msg2, err := responder.WriteMessage([]byte("hello from pool"))
	require.NoError(t, err)

	// Initiator reads message 2
	payload2, err := initiator.ReadMessage(msg2)
	require.NoError(t, err)
	assert.Equal(t, []byte("hello from pool"), payload2)

	// Both should be complete now
	assert.True(t, initiator.IsComplete())
	assert.True(t, responder.IsComplete())

	// Split to get transport keys
	initSend, initRecv, err := initiator.Split()
	require.NoError(t, err)

	respSend, respRecv, err := responder.Split()
	require.NoError(t, err)

	// Test transport encryption
	// Initiator sends -> Responder receives
	testMsg := []byte("share submission data")
	encrypted, err := initSend.Encrypt(testMsg, nil)
	require.NoError(t, err)

	decrypted, err := respRecv.Decrypt(encrypted, nil)
	require.NoError(t, err)
	assert.Equal(t, testMsg, decrypted)

	// Responder sends -> Initiator receives
	testMsg2 := []byte("job notification")
	encrypted2, err := respSend.Encrypt(testMsg2, nil)
	require.NoError(t, err)

	decrypted2, err := initRecv.Decrypt(encrypted2, nil)
	require.NoError(t, err)
	assert.Equal(t, testMsg2, decrypted2)
}

func TestHandshake_IsComplete(t *testing.T) {
	initiator, _ := NewInitiatorHandshake()
	assert.False(t, initiator.IsComplete())

	// After sending first message, still not complete
	initiator.WriteMessage(nil)
	assert.False(t, initiator.IsComplete())
}

func TestHandshake_GetRemoteStatic(t *testing.T) {
	serverStatic, _ := GenerateKeyPair()

	initiator, _ := NewInitiatorHandshake()
	responder, _ := NewResponderHandshake(serverStatic)

	// Before handshake, remote static is empty
	var empty [DHKeySize]byte
	assert.Equal(t, empty, initiator.GetRemoteStatic())

	// Perform handshake
	msg1, _ := initiator.WriteMessage(nil)
	responder.ReadMessage(msg1)
	msg2, _ := responder.WriteMessage(nil)
	initiator.ReadMessage(msg2)

	// After handshake, initiator knows server's static key
	assert.Equal(t, serverStatic.PublicKey, initiator.GetRemoteStatic())
}

// -----------------------------------------------------------------------------
// Secure Channel Tests
// -----------------------------------------------------------------------------

func TestSecureChannel_EncryptDecrypt(t *testing.T) {
	// Create test cipher states
	var key1, key2 [SymKeySize]byte
	for i := range key1 {
		key1[i] = byte(i)
		key2[i] = byte(i + 32)
	}

	send, _ := NewCipherState(key1)
	recv, _ := NewCipherState(key1) // Same key for testing

	sc := NewSecureChannel(send, recv)
	assert.True(t, sc.IsEstablished())

	plaintext := []byte("test message")
	ciphertext, err := sc.Encrypt(plaintext)
	require.NoError(t, err)

	// Note: In real usage, the other party would decrypt
	// Here we're using same keys, so create matching recv
	recv2, _ := NewCipherState(key1)
	sc2 := NewSecureChannel(nil, recv2)

	decrypted, err := sc2.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestSecureChannel_IsEstablished(t *testing.T) {
	sc := &SecureChannel{}
	assert.False(t, sc.IsEstablished())

	var key [SymKeySize]byte
	send, _ := NewCipherState(key)
	recv, _ := NewCipherState(key)

	sc2 := NewSecureChannel(send, recv)
	assert.True(t, sc2.IsEstablished())
}

// -----------------------------------------------------------------------------
// SHA256 Tests
// -----------------------------------------------------------------------------

func TestSHA256Hash(t *testing.T) {
	// Test vector from NIST
	input := []byte("abc")
	hash := sha256Hash(input)

	expected := [32]byte{
		0xba, 0x78, 0x16, 0xbf, 0x8f, 0x01, 0xcf, 0xea,
		0x41, 0x41, 0x40, 0xde, 0x5d, 0xae, 0x22, 0x23,
		0xb0, 0x03, 0x61, 0xa3, 0x96, 0x17, 0x7a, 0x9c,
		0xb4, 0x10, 0xff, 0x61, 0xf2, 0x00, 0x15, 0xad,
	}

	assert.Equal(t, expected, hash)
}

func TestSHA256Hash_Empty(t *testing.T) {
	hash := sha256Hash([]byte{})

	expected := [32]byte{
		0xe3, 0xb0, 0xc4, 0x42, 0x98, 0xfc, 0x1c, 0x14,
		0x9a, 0xfb, 0xf4, 0xc8, 0x99, 0x6f, 0xb9, 0x24,
		0x27, 0xae, 0x41, 0xe4, 0x64, 0x9b, 0x93, 0x4c,
		0xa4, 0x95, 0x99, 0x1b, 0x78, 0x52, 0xb8, 0x55,
	}

	assert.Equal(t, expected, hash)
}

// -----------------------------------------------------------------------------
// Performance Benchmarks
// -----------------------------------------------------------------------------

func BenchmarkKeyPairGeneration(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateKeyPair()
	}
}

func BenchmarkDH(b *testing.B) {
	alice, _ := GenerateKeyPair()
	bob, _ := GenerateKeyPair()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		alice.DH(bob.PublicKey)
	}
}

func BenchmarkEncrypt(b *testing.B) {
	var key [SymKeySize]byte
	cs, _ := NewCipherState(key)
	plaintext := make([]byte, 256)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cs.Encrypt(plaintext, nil)
	}
}

func BenchmarkDecrypt(b *testing.B) {
	var key [SymKeySize]byte
	cs1, _ := NewCipherState(key)
	plaintext := make([]byte, 256)

	ciphertexts := make([][]byte, b.N)
	for i := 0; i < b.N; i++ {
		ciphertexts[i], _ = cs1.Encrypt(plaintext, nil)
	}

	cs2, _ := NewCipherState(key)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cs2.Decrypt(ciphertexts[i], nil)
	}
}

func BenchmarkFullHandshake(b *testing.B) {
	serverStatic, _ := GenerateKeyPair()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		initiator, _ := NewInitiatorHandshake()
		responder, _ := NewResponderHandshake(serverStatic)

		msg1, _ := initiator.WriteMessage(nil)
		responder.ReadMessage(msg1)
		msg2, _ := responder.WriteMessage(nil)
		initiator.ReadMessage(msg2)

		initiator.Split()
		responder.Split()
	}
}
