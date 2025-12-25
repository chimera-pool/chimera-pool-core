package payouts

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// LITECOIN WALLET CLIENT TESTS (TDD)
// =============================================================================

func TestLitecoinWalletClient_Creation(t *testing.T) {
	t.Run("creates client with valid config", func(t *testing.T) {
		config := WalletConfig{
			RPCURL:      "http://localhost:9332",
			RPCUser:     "chimera",
			RPCPassword: "test123",
			Network:     "mainnet",
			Timeout:     30 * time.Second,
		}

		client, err := NewLitecoinWalletClient(config)
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("returns error with empty RPC URL", func(t *testing.T) {
		config := WalletConfig{
			RPCUser:     "chimera",
			RPCPassword: "test123",
		}

		client, err := NewLitecoinWalletClient(config)
		assert.Error(t, err)
		assert.Nil(t, client)
	})

	t.Run("uses default timeout if not specified", func(t *testing.T) {
		config := WalletConfig{
			RPCURL:      "http://localhost:9332",
			RPCUser:     "chimera",
			RPCPassword: "test123",
		}

		client, err := NewLitecoinWalletClient(config)
		require.NoError(t, err)
		assert.Equal(t, 30*time.Second, client.config.Timeout)
	})
}

func TestLitecoinWalletClient_ValidateAddress(t *testing.T) {
	config := WalletConfig{
		RPCURL:      "http://localhost:9332",
		RPCUser:     "chimera",
		RPCPassword: "test123",
		Network:     "mainnet",
	}
	client, _ := NewLitecoinWalletClient(config)

	t.Run("validates mainnet P2PKH address (L prefix)", func(t *testing.T) {
		assert.True(t, client.ValidateAddress("LTC1234567890123456789012345678901234"))
	})

	t.Run("validates mainnet P2SH address (M prefix)", func(t *testing.T) {
		assert.True(t, client.ValidateAddress("MTC1234567890123456789012345678901234"))
	})

	t.Run("validates bech32 address (ltc1 prefix)", func(t *testing.T) {
		assert.True(t, client.ValidateAddress("ltc1qgsm3fv44wprdcsh3trgarm05rr7l8ryggujr5w"))
	})

	t.Run("rejects empty address", func(t *testing.T) {
		assert.False(t, client.ValidateAddress(""))
	})

	t.Run("rejects short address", func(t *testing.T) {
		assert.False(t, client.ValidateAddress("ltc1short"))
	})

	t.Run("rejects invalid prefix", func(t *testing.T) {
		assert.False(t, client.ValidateAddress("btc1qgsm3fv44wprdcsh3trgarm05rr7l8ryggujr5w"))
	})
}

func TestLitecoinWalletClient_GetBalance(t *testing.T) {
	t.Run("returns balance from RPC", func(t *testing.T) {
		// Create mock RPC server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)

			if req["method"] == "getbalance" {
				response := map[string]interface{}{
					"result": 10.5, // 10.5 LTC
					"error":  nil,
					"id":     req["id"],
				}
				json.NewEncoder(w).Encode(response)
			}
		}))
		defer server.Close()

		config := WalletConfig{
			RPCURL:      server.URL,
			RPCUser:     "chimera",
			RPCPassword: "test123",
		}
		client, _ := NewLitecoinWalletClient(config)

		balance, err := client.GetBalance(context.Background())
		require.NoError(t, err)
		assert.Equal(t, int64(1050000000), balance) // 10.5 LTC in litoshis
	})

	t.Run("handles RPC error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)

			response := map[string]interface{}{
				"result": nil,
				"error": map[string]interface{}{
					"code":    -32600,
					"message": "wallet locked",
				},
				"id": req["id"],
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		config := WalletConfig{
			RPCURL:      server.URL,
			RPCUser:     "chimera",
			RPCPassword: "test123",
		}
		client, _ := NewLitecoinWalletClient(config)

		_, err := client.GetBalance(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "wallet locked")
	})
}

func TestLitecoinWalletClient_SendTransaction(t *testing.T) {
	t.Run("sends transaction successfully", func(t *testing.T) {
		expectedTxHash := "abc123def456789"
		testAddress := "ltc1qgsm3fv44wprdcsh3trgarm05rr7l8ryggujr5w"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)

			if req["method"] == "sendtoaddress" {
				params := req["params"].([]interface{})
				address := params[0].(string)
				amount := params[1].(float64)

				// Verify parameters
				assert.Equal(t, testAddress, address)
				assert.InDelta(t, 0.01, amount, 0.0001) // 0.01 LTC

				response := map[string]interface{}{
					"result": expectedTxHash,
					"error":  nil,
					"id":     req["id"],
				}
				json.NewEncoder(w).Encode(response)
			}
		}))
		defer server.Close()

		config := WalletConfig{
			RPCURL:      server.URL,
			RPCUser:     "chimera",
			RPCPassword: "test123",
		}
		client, _ := NewLitecoinWalletClient(config)

		txHash, err := client.SendTransaction(context.Background(), testAddress, 1000000) // 0.01 LTC
		require.NoError(t, err)
		assert.Equal(t, expectedTxHash, txHash)
	})

	t.Run("handles insufficient funds error", func(t *testing.T) {
		testAddress := "ltc1qgsm3fv44wprdcsh3trgarm05rr7l8ryggujr5w"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)

			response := map[string]interface{}{
				"result": nil,
				"error": map[string]interface{}{
					"code":    -6,
					"message": "Insufficient funds",
				},
				"id": req["id"],
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		config := WalletConfig{
			RPCURL:      server.URL,
			RPCUser:     "chimera",
			RPCPassword: "test123",
		}
		client, _ := NewLitecoinWalletClient(config)

		_, err := client.SendTransaction(context.Background(), testAddress, 1000000000000)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Insufficient funds")
	})

	t.Run("validates address before sending", func(t *testing.T) {
		config := WalletConfig{
			RPCURL:      "http://localhost:9332",
			RPCUser:     "chimera",
			RPCPassword: "test123",
		}
		client, _ := NewLitecoinWalletClient(config)

		_, err := client.SendTransaction(context.Background(), "invalid", 1000000)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid address")
	})

	t.Run("rejects zero amount", func(t *testing.T) {
		config := WalletConfig{
			RPCURL:      "http://localhost:9332",
			RPCUser:     "chimera",
			RPCPassword: "test123",
		}
		client, _ := NewLitecoinWalletClient(config)

		_, err := client.SendTransaction(context.Background(), "ltc1qgsm3fv44wprdcsh3trgarm05rr7l8ryggujr5w", 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "amount must be positive")
	})
}

func TestLitecoinWalletClient_EstimateFee(t *testing.T) {
	t.Run("estimates fee for transaction", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)

			if req["method"] == "estimatesmartfee" {
				response := map[string]interface{}{
					"result": map[string]interface{}{
						"feerate": 0.0001, // LTC per KB
						"blocks":  6,
					},
					"error": nil,
					"id":    req["id"],
				}
				json.NewEncoder(w).Encode(response)
			}
		}))
		defer server.Close()

		config := WalletConfig{
			RPCURL:      server.URL,
			RPCUser:     "chimera",
			RPCPassword: "test123",
		}
		client, _ := NewLitecoinWalletClient(config)

		fee, err := client.EstimateFee(context.Background(), 6)
		require.NoError(t, err)
		assert.Greater(t, fee, int64(0))
	})
}

func TestLitecoinWalletClient_UnlockWallet(t *testing.T) {
	t.Run("unlocks wallet successfully", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)

			if req["method"] == "walletpassphrase" {
				response := map[string]interface{}{
					"result": nil,
					"error":  nil,
					"id":     req["id"],
				}
				json.NewEncoder(w).Encode(response)
			}
		}))
		defer server.Close()

		config := WalletConfig{
			RPCURL:         server.URL,
			RPCUser:        "chimera",
			RPCPassword:    "test123",
			WalletPassword: "wallet_pass",
		}
		client, _ := NewLitecoinWalletClient(config)

		err := client.UnlockWallet(context.Background(), 60)
		assert.NoError(t, err)
	})
}
