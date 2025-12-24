package v2

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLitecoinRPCClient(t *testing.T) {
	config := DefaultLitecoinRPCConfig()
	client := NewLitecoinRPCClient(config)

	require.NotNil(t, client)
	assert.Equal(t, config.URL, client.url)
	assert.Equal(t, config.User, client.user)
	assert.Equal(t, config.Password, client.password)
}

func TestDefaultLitecoinRPCConfig(t *testing.T) {
	config := DefaultLitecoinRPCConfig()

	assert.Equal(t, "http://litecoind:9332", config.URL)
	assert.Equal(t, "chimera", config.User)
	assert.Equal(t, "ChimeraLTC2024!", config.Password)
	assert.Equal(t, 30*time.Second, config.Timeout)
}

func TestLitecoinRPCClient_GetBlockCount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Check basic auth
		user, pass, ok := r.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "testuser", user)
		assert.Equal(t, "testpass", pass)

		response := `{"jsonrpc":"1.0","id":1,"result":3026123,"error":null}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := NewLitecoinRPCClient(LitecoinRPCConfig{
		URL:      server.URL,
		User:     "testuser",
		Password: "testpass",
		Timeout:  5 * time.Second,
	})

	count, err := client.GetBlockCount()
	require.NoError(t, err)
	assert.Equal(t, int64(3026123), count)
}

func TestLitecoinRPCClient_GetBlockchainInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{
			"jsonrpc": "1.0",
			"id": 1,
			"result": {
				"chain": "main",
				"blocks": 3026123,
				"headers": 3026123,
				"bestblockhash": "e4aa3af6e1284afaebca036a584c59e3cd49bdeef6f25fa363b60b9b7b6eacd3",
				"difficulty": 95333387.21219528,
				"verificationprogress": 0.9999998621361915,
				"initialblockdownload": false
			},
			"error": null
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := NewLitecoinRPCClient(LitecoinRPCConfig{
		URL:      server.URL,
		User:     "test",
		Password: "test",
		Timeout:  5 * time.Second,
	})

	info, err := client.GetBlockchainInfo()
	require.NoError(t, err)
	assert.Equal(t, "main", info.Chain)
	assert.Equal(t, int64(3026123), info.Blocks)
	assert.False(t, info.InitialBlockDownload)
	assert.InDelta(t, 0.9999998621, info.SyncProgress, 0.0001)
}

func TestLitecoinRPCClient_GetNetworkDifficulty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{"jsonrpc":"1.0","id":1,"result":95333387.21219528,"error":null}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := NewLitecoinRPCClient(LitecoinRPCConfig{
		URL:      server.URL,
		User:     "test",
		Password: "test",
		Timeout:  5 * time.Second,
	})

	diff, err := client.GetNetworkDifficulty()
	require.NoError(t, err)
	assert.InDelta(t, 95333387.21, diff, 1.0)
}

func TestLitecoinRPCClient_GetBlockTemplate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request includes segwit rules
		var req rpcRequest
		json.NewDecoder(r.Body).Decode(&req)
		assert.Equal(t, "getblocktemplate", req.Method)

		response := `{
			"jsonrpc": "1.0",
			"id": 1,
			"result": {
				"version": 536870912,
				"previousblockhash": "e4aa3af6e1284afaebca036a584c59e3cd49bdeef6f25fa363b60b9b7b6eacd3",
				"transactions": [],
				"coinbasevalue": 625000000,
				"target": "00000000000000000007a4a30000000000000000000000000000000000000000",
				"mintime": 1700000000,
				"curtime": 1700000100,
				"height": 3026124,
				"bits": "1a07a4a3",
				"sigoplimit": 80000,
				"sizelimit": 4000000,
				"weightlimit": 4000000
			},
			"error": null
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := NewLitecoinRPCClient(LitecoinRPCConfig{
		URL:      server.URL,
		User:     "test",
		Password: "test",
		Timeout:  5 * time.Second,
	})

	template, err := client.GetBlockTemplate()
	require.NoError(t, err)
	assert.Equal(t, uint64(3026124), template.Height)
	assert.Equal(t, uint64(625000000), template.CoinbaseValue)
	assert.Equal(t, "1a07a4a3", template.Bits)
}

func TestLitecoinRPCClient_SubmitBlock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		json.NewDecoder(r.Body).Decode(&req)
		assert.Equal(t, "submitblock", req.Method)

		// Success response (null result means accepted)
		response := `{"jsonrpc":"1.0","id":1,"result":null,"error":null}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := NewLitecoinRPCClient(LitecoinRPCConfig{
		URL:      server.URL,
		User:     "test",
		Password: "test",
		Timeout:  5 * time.Second,
	})

	err := client.SubmitBlock("00000020...")
	assert.NoError(t, err)
}

func TestLitecoinRPCClient_RPCError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{"jsonrpc":"1.0","id":1,"result":null,"error":{"code":-8,"message":"Block decode failed"}}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := NewLitecoinRPCClient(LitecoinRPCConfig{
		URL:      server.URL,
		User:     "test",
		Password: "test",
		Timeout:  5 * time.Second,
	})

	err := client.SubmitBlock("invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Block decode failed")
}

func TestLitecoinRPCClient_ValidateAddress(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{
			"jsonrpc": "1.0",
			"id": 1,
			"result": {
				"isvalid": true,
				"address": "ltc1qgsm3fv44wprdcsh3trgarm05rr7l8ryggujr5w",
				"scriptPubKey": "0014440e34b2ab838dbc42e2c68f47b7e863f7ce7322",
				"isscript": false,
				"iswitness": true
			},
			"error": null
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := NewLitecoinRPCClient(LitecoinRPCConfig{
		URL:      server.URL,
		User:     "test",
		Password: "test",
		Timeout:  5 * time.Second,
	})

	validation, err := client.ValidateAddress("ltc1qgsm3fv44wprdcsh3trgarm05rr7l8ryggujr5w")
	require.NoError(t, err)
	assert.True(t, validation.IsValid)
	assert.True(t, validation.IsWitness)
	assert.NotEmpty(t, validation.ScriptPubKey)
}

func TestLitecoinRPCClient_GetMiningInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{
			"jsonrpc": "1.0",
			"id": 1,
			"result": {
				"blocks": 3026123,
				"difficulty": 95333387.21219528,
				"networkhashps": 1234567890000000,
				"pooledtx": 150,
				"chain": "main"
			},
			"error": null
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := NewLitecoinRPCClient(LitecoinRPCConfig{
		URL:      server.URL,
		User:     "test",
		Password: "test",
		Timeout:  5 * time.Second,
	})

	info, err := client.GetMiningInfo()
	require.NoError(t, err)
	assert.Equal(t, int64(3026123), info.Blocks)
	assert.Equal(t, "main", info.Chain)
	assert.Equal(t, 150, info.PooledTx)
}

func TestLitecoinRPCClient_ConnectionFailure(t *testing.T) {
	client := NewLitecoinRPCClient(LitecoinRPCConfig{
		URL:      "http://localhost:99999", // Invalid port
		User:     "test",
		Password: "test",
		Timeout:  1 * time.Second,
	})

	_, err := client.GetBlockCount()
	assert.Error(t, err)
}

func TestLitecoinRPCClient_TestConnection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{"jsonrpc":"1.0","id":1,"result":3026123,"error":null}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := NewLitecoinRPCClient(LitecoinRPCConfig{
		URL:      server.URL,
		User:     "test",
		Password: "test",
		Timeout:  5 * time.Second,
	})

	err := client.TestConnection()
	assert.NoError(t, err)
}

func TestLitecoinRPCClient_GetBestBlockHash(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{"jsonrpc":"1.0","id":1,"result":"e4aa3af6e1284afaebca036a584c59e3cd49bdeef6f25fa363b60b9b7b6eacd3","error":null}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := NewLitecoinRPCClient(LitecoinRPCConfig{
		URL:      server.URL,
		User:     "test",
		Password: "test",
		Timeout:  5 * time.Second,
	})

	hash, err := client.GetBestBlockHash()
	require.NoError(t, err)
	assert.Equal(t, "e4aa3af6e1284afaebca036a584c59e3cd49bdeef6f25fa363b60b9b7b6eacd3", hash)
}

func TestLitecoinRPCClient_EstimateSmartFee(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{
			"jsonrpc": "1.0",
			"id": 1,
			"result": {
				"feerate": 0.00001,
				"blocks": 6
			},
			"error": null
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := NewLitecoinRPCClient(LitecoinRPCConfig{
		URL:      server.URL,
		User:     "test",
		Password: "test",
		Timeout:  5 * time.Second,
	})

	estimate, err := client.EstimateSmartFee(6)
	require.NoError(t, err)
	assert.Equal(t, 0.00001, estimate.FeeRate)
	assert.Equal(t, 6, estimate.Blocks)
}
