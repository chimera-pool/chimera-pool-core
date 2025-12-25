package health

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
// Test Helpers
// =============================================================================

func newTestBlockDAGChecker(serverURL string) *BlockDAGHealthChecker {
	return NewBlockDAGHealthChecker(&BlockDAGNodeConfig{
		RPCURL:        serverURL,
		WalletAddress: "0x1234567890abcdef",
	}, 5*time.Second)
}

func mockEthRPCServer(t *testing.T, handler func(method string, params json.RawMessage) (interface{}, *rpcError)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		result, rpcErr := handler(req.Method, req.Params)

		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  result,
			"error":   rpcErr,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

// =============================================================================
// Constructor Tests
// =============================================================================

func TestNewBlockDAGHealthChecker(t *testing.T) {
	config := &BlockDAGNodeConfig{
		RPCURL:        "https://rpc.blockdag.com",
		WalletAddress: "0x123",
	}

	checker := NewBlockDAGHealthChecker(config, 10*time.Second)

	require.NotNil(t, checker)
	assert.Equal(t, config, checker.config)
	assert.Equal(t, 10*time.Second, checker.timeout)
}

func TestNewBlockDAGHealthChecker_DefaultTimeout(t *testing.T) {
	config := &BlockDAGNodeConfig{
		RPCURL: "https://rpc.blockdag.com",
	}

	checker := NewBlockDAGHealthChecker(config, 0)

	assert.Equal(t, 10*time.Second, checker.timeout)
}

func TestBlockDAGHealthChecker_GetChainName(t *testing.T) {
	checker := NewBlockDAGHealthChecker(&BlockDAGNodeConfig{}, 0)

	assert.Equal(t, "blockdag", checker.GetChainName())
}

// =============================================================================
// RPC Connectivity Tests
// =============================================================================

func TestBlockDAGHealthChecker_CheckRPCConnectivity_Success(t *testing.T) {
	server := mockEthRPCServer(t, func(method string, params json.RawMessage) (interface{}, *rpcError) {
		if method == "eth_blockNumber" {
			return "0x1234", nil
		}
		return nil, &rpcError{Code: -32601, Message: "Method not found"}
	})
	defer server.Close()

	checker := newTestBlockDAGChecker(server.URL)
	ctx := context.Background()

	err := checker.CheckRPCConnectivity(ctx)

	assert.NoError(t, err)
}

func TestBlockDAGHealthChecker_CheckRPCConnectivity_Failure(t *testing.T) {
	checker := newTestBlockDAGChecker("http://localhost:99999")
	ctx := context.Background()

	err := checker.CheckRPCConnectivity(ctx)

	assert.Error(t, err)
}

// =============================================================================
// Sync Progress Tests
// =============================================================================

func TestBlockDAGHealthChecker_GetSyncProgress_FullySynced(t *testing.T) {
	server := mockEthRPCServer(t, func(method string, params json.RawMessage) (interface{}, *rpcError) {
		if method == "eth_syncing" {
			return false, nil
		}
		return nil, nil
	})
	defer server.Close()

	checker := newTestBlockDAGChecker(server.URL)
	ctx := context.Background()

	progress, err := checker.GetSyncProgress(ctx)

	assert.NoError(t, err)
	assert.Equal(t, 1.0, progress)
}

func TestBlockDAGHealthChecker_GetSyncProgress_Syncing(t *testing.T) {
	server := mockEthRPCServer(t, func(method string, params json.RawMessage) (interface{}, *rpcError) {
		if method == "eth_syncing" {
			return map[string]string{
				"currentBlock": "0x64", // 100
				"highestBlock": "0xc8", // 200
			}, nil
		}
		return nil, nil
	})
	defer server.Close()

	checker := newTestBlockDAGChecker(server.URL)
	ctx := context.Background()

	progress, err := checker.GetSyncProgress(ctx)

	assert.NoError(t, err)
	assert.InDelta(t, 0.5, progress, 0.01) // 100/200 = 0.5
}

// =============================================================================
// IBD Status Tests
// =============================================================================

func TestBlockDAGHealthChecker_IsInitialBlockDownload_False(t *testing.T) {
	server := mockEthRPCServer(t, func(method string, params json.RawMessage) (interface{}, *rpcError) {
		return false, nil // Not syncing
	})
	defer server.Close()

	checker := newTestBlockDAGChecker(server.URL)
	ctx := context.Background()

	ibd, err := checker.IsInitialBlockDownload(ctx)

	assert.NoError(t, err)
	assert.False(t, ibd)
}

func TestBlockDAGHealthChecker_IsInitialBlockDownload_True(t *testing.T) {
	server := mockEthRPCServer(t, func(method string, params json.RawMessage) (interface{}, *rpcError) {
		return map[string]string{
			"currentBlock": "0x32", // 50
			"highestBlock": "0xc8", // 200
		}, nil
	})
	defer server.Close()

	checker := newTestBlockDAGChecker(server.URL)
	ctx := context.Background()

	ibd, err := checker.IsInitialBlockDownload(ctx)

	assert.NoError(t, err)
	assert.True(t, ibd) // 50/200 = 0.25 < 0.9999
}

// =============================================================================
// Block Template Generation Tests
// =============================================================================

func TestBlockDAGHealthChecker_CheckBlockTemplateGeneration_Success(t *testing.T) {
	server := mockEthRPCServer(t, func(method string, params json.RawMessage) (interface{}, *rpcError) {
		if method == "eth_getBlockByNumber" {
			return map[string]interface{}{
				"number": "0x1234",
			}, nil
		}
		return nil, nil
	})
	defer server.Close()

	checker := newTestBlockDAGChecker(server.URL)
	ctx := context.Background()

	err := checker.CheckBlockTemplateGeneration(ctx)

	assert.NoError(t, err)
}

func TestBlockDAGHealthChecker_CheckBlockTemplateGeneration_Failure(t *testing.T) {
	server := mockEthRPCServer(t, func(method string, params json.RawMessage) (interface{}, *rpcError) {
		return nil, &rpcError{Code: -32000, Message: "Node not ready"}
	})
	defer server.Close()

	checker := newTestBlockDAGChecker(server.URL)
	ctx := context.Background()

	err := checker.CheckBlockTemplateGeneration(ctx)

	assert.Error(t, err)
}

// =============================================================================
// Full Diagnostics Tests
// =============================================================================

func TestBlockDAGHealthChecker_RunDiagnostics_Healthy(t *testing.T) {
	server := mockEthRPCServer(t, func(method string, params json.RawMessage) (interface{}, *rpcError) {
		switch method {
		case "eth_blockNumber":
			return "0x1234", nil // 4660
		case "eth_syncing":
			return false, nil
		case "eth_getBlockByNumber":
			return map[string]interface{}{"number": "0x1234"}, nil
		}
		return nil, nil
	})
	defer server.Close()

	checker := newTestBlockDAGChecker(server.URL)
	ctx := context.Background()

	diag, err := checker.RunDiagnostics(ctx)

	require.NoError(t, err)
	require.NotNil(t, diag)
	assert.Equal(t, "blockdag", diag.ChainName)
	assert.True(t, diag.RPCConnected)
	assert.Equal(t, 1.0, diag.SyncProgress)
	assert.False(t, diag.IsIBD)
	assert.True(t, diag.BlockTemplateOK)
}

func TestBlockDAGHealthChecker_RunDiagnostics_RPCDown(t *testing.T) {
	checker := newTestBlockDAGChecker("http://localhost:99999")
	ctx := context.Background()

	diag, err := checker.RunDiagnostics(ctx)

	require.NoError(t, err)
	require.NotNil(t, diag)
	assert.False(t, diag.RPCConnected)
	assert.NotEmpty(t, diag.RPCError)
}

// =============================================================================
// Interface Compliance Tests
// =============================================================================

func TestBlockDAGHealthChecker_ImplementsNodeHealthChecker(t *testing.T) {
	var _ NodeHealthChecker = (*BlockDAGHealthChecker)(nil)
}

func TestBlockDAGHealthChecker_ImplementsRPCChecker(t *testing.T) {
	var _ RPCChecker = (*BlockDAGHealthChecker)(nil)
}

func TestBlockDAGHealthChecker_ImplementsSyncChecker(t *testing.T) {
	var _ SyncChecker = (*BlockDAGHealthChecker)(nil)
}

func TestBlockDAGHealthChecker_ImplementsBlockTemplateChecker(t *testing.T) {
	var _ BlockTemplateChecker = (*BlockDAGHealthChecker)(nil)
}

func TestBlockDAGHealthChecker_ImplementsChainDiagnostics(t *testing.T) {
	var _ ChainDiagnostics = (*BlockDAGHealthChecker)(nil)
}

// =============================================================================
// Error Variable Tests
// =============================================================================

func TestBlockDAGErrorVariables(t *testing.T) {
	assert.NotNil(t, ErrBlockDAGRPCFailed)
	assert.Contains(t, ErrBlockDAGRPCFailed.Error(), "BlockDAG")
}
