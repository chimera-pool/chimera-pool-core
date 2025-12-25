package health

import (
	"context"
	"encoding/json"
	"errors"
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

func newTestLitecoinChecker(serverURL string) *LitecoinHealthChecker {
	return NewLitecoinHealthChecker(&LitecoinNodeConfig{
		RPCURL:      serverURL,
		RPCUser:     "testuser",
		RPCPassword: "testpass",
	}, 5*time.Second)
}

func mockRPCServer(t *testing.T, handler func(method string, params json.RawMessage) (interface{}, *rpcError)) *httptest.Server {
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
			"result": result,
			"error":  rpcErr,
			"id":     "health-check",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// =============================================================================
// Constructor Tests
// =============================================================================

func TestNewLitecoinHealthChecker(t *testing.T) {
	config := &LitecoinNodeConfig{
		RPCURL:      "http://localhost:9332",
		RPCUser:     "user",
		RPCPassword: "pass",
	}

	checker := NewLitecoinHealthChecker(config, 10*time.Second)

	require.NotNil(t, checker)
	assert.Equal(t, config, checker.config)
	assert.Equal(t, 10*time.Second, checker.timeout)
	assert.NotNil(t, checker.httpClient)
}

func TestNewLitecoinHealthChecker_DefaultTimeout(t *testing.T) {
	config := &LitecoinNodeConfig{
		RPCURL: "http://localhost:9332",
	}

	checker := NewLitecoinHealthChecker(config, 0)

	assert.Equal(t, 10*time.Second, checker.timeout)
}

func TestLitecoinHealthChecker_GetChainName(t *testing.T) {
	checker := NewLitecoinHealthChecker(&LitecoinNodeConfig{}, 0)

	assert.Equal(t, "litecoin", checker.GetChainName())
}

// =============================================================================
// RPC Connectivity Tests
// =============================================================================

func TestLitecoinHealthChecker_CheckRPCConnectivity_Success(t *testing.T) {
	server := mockRPCServer(t, func(method string, params json.RawMessage) (interface{}, *rpcError) {
		if method == "getblockchaininfo" {
			return map[string]interface{}{
				"chain":                "main",
				"blocks":               3026575,
				"verificationprogress": 0.9999999,
			}, nil
		}
		return nil, &rpcError{Code: -32601, Message: "Method not found"}
	})
	defer server.Close()

	checker := newTestLitecoinChecker(server.URL)
	ctx := context.Background()

	err := checker.CheckRPCConnectivity(ctx)

	assert.NoError(t, err)
}

func TestLitecoinHealthChecker_CheckRPCConnectivity_Failure(t *testing.T) {
	checker := newTestLitecoinChecker("http://localhost:99999")
	ctx := context.Background()

	err := checker.CheckRPCConnectivity(ctx)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrRPCUnreachable))
}

// =============================================================================
// Sync Progress Tests
// =============================================================================

func TestLitecoinHealthChecker_GetSyncProgress_FullySynced(t *testing.T) {
	server := mockRPCServer(t, func(method string, params json.RawMessage) (interface{}, *rpcError) {
		return map[string]interface{}{
			"verificationprogress": 0.9999999,
		}, nil
	})
	defer server.Close()

	checker := newTestLitecoinChecker(server.URL)
	ctx := context.Background()

	progress, err := checker.GetSyncProgress(ctx)

	assert.NoError(t, err)
	assert.InDelta(t, 0.9999999, progress, 0.0000001)
}

func TestLitecoinHealthChecker_GetSyncProgress_Syncing(t *testing.T) {
	server := mockRPCServer(t, func(method string, params json.RawMessage) (interface{}, *rpcError) {
		return map[string]interface{}{
			"verificationprogress": 0.75,
		}, nil
	})
	defer server.Close()

	checker := newTestLitecoinChecker(server.URL)
	ctx := context.Background()

	progress, err := checker.GetSyncProgress(ctx)

	assert.NoError(t, err)
	assert.Equal(t, 0.75, progress)
}

// =============================================================================
// IBD Status Tests
// =============================================================================

func TestLitecoinHealthChecker_IsInitialBlockDownload_False(t *testing.T) {
	server := mockRPCServer(t, func(method string, params json.RawMessage) (interface{}, *rpcError) {
		return map[string]interface{}{
			"initialblockdownload": false,
		}, nil
	})
	defer server.Close()

	checker := newTestLitecoinChecker(server.URL)
	ctx := context.Background()

	ibd, err := checker.IsInitialBlockDownload(ctx)

	assert.NoError(t, err)
	assert.False(t, ibd)
}

func TestLitecoinHealthChecker_IsInitialBlockDownload_True(t *testing.T) {
	server := mockRPCServer(t, func(method string, params json.RawMessage) (interface{}, *rpcError) {
		return map[string]interface{}{
			"initialblockdownload": true,
		}, nil
	})
	defer server.Close()

	checker := newTestLitecoinChecker(server.URL)
	ctx := context.Background()

	ibd, err := checker.IsInitialBlockDownload(ctx)

	assert.NoError(t, err)
	assert.True(t, ibd)
}

func TestLitecoinHealthChecker_IsInitialBlockDownload_FromError(t *testing.T) {
	server := mockRPCServer(t, func(method string, params json.RawMessage) (interface{}, *rpcError) {
		return nil, &rpcError{Code: -28, Message: "Loading block index..."}
	})
	defer server.Close()

	checker := newTestLitecoinChecker(server.URL)
	ctx := context.Background()

	ibd, err := checker.IsInitialBlockDownload(ctx)

	assert.NoError(t, err)
	assert.True(t, ibd)
}

// =============================================================================
// Block Template Generation Tests
// =============================================================================

func TestLitecoinHealthChecker_CheckBlockTemplateGeneration_Success(t *testing.T) {
	server := mockRPCServer(t, func(method string, params json.RawMessage) (interface{}, *rpcError) {
		if method == "getblocktemplate" {
			return map[string]interface{}{
				"version":           536870912,
				"previousblockhash": "abc123",
				"transactions":      []interface{}{},
				"height":            3026576,
			}, nil
		}
		return nil, nil
	})
	defer server.Close()

	checker := newTestLitecoinChecker(server.URL)
	ctx := context.Background()

	err := checker.CheckBlockTemplateGeneration(ctx)

	assert.NoError(t, err)
}

func TestLitecoinHealthChecker_CheckBlockTemplateGeneration_MWEBError(t *testing.T) {
	server := mockRPCServer(t, func(method string, params json.RawMessage) (interface{}, *rpcError) {
		return nil, &rpcError{
			Code:    -1,
			Message: "CreateNewBlock: TestBlockValidity failed: mweb-connect-failed, MWEB::Node::ConnectBlock(): Failed to connect MWEB block: PedersenCommitSum: secp256k1_pedersen_commit_sum error",
		}
	})
	defer server.Close()

	checker := newTestLitecoinChecker(server.URL)
	ctx := context.Background()

	err := checker.CheckBlockTemplateGeneration(ctx)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrMWEBFailure))
	assert.True(t, IsMWEBError(err))
}

func TestLitecoinHealthChecker_CheckBlockTemplateGeneration_IBDError(t *testing.T) {
	server := mockRPCServer(t, func(method string, params json.RawMessage) (interface{}, *rpcError) {
		return nil, &rpcError{
			Code:    -10,
			Message: "Litecoin Core is in initial sync and waiting for blocks...",
		}
	})
	defer server.Close()

	checker := newTestLitecoinChecker(server.URL)
	ctx := context.Background()

	err := checker.CheckBlockTemplateGeneration(ctx)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrNodeInIBD))
	assert.True(t, IsIBDError(err))
}

func TestLitecoinHealthChecker_CheckBlockTemplateGeneration_GenericError(t *testing.T) {
	server := mockRPCServer(t, func(method string, params json.RawMessage) (interface{}, *rpcError) {
		return nil, &rpcError{
			Code:    -1,
			Message: "Some other error",
		}
	})
	defer server.Close()

	checker := newTestLitecoinChecker(server.URL)
	ctx := context.Background()

	err := checker.CheckBlockTemplateGeneration(ctx)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrTemplateGenFailed))
	assert.False(t, IsMWEBError(err))
	assert.False(t, IsIBDError(err))
}

// =============================================================================
// Mempool Info Tests
// =============================================================================

func TestLitecoinHealthChecker_GetMempoolInfo_Success(t *testing.T) {
	server := mockRPCServer(t, func(method string, params json.RawMessage) (interface{}, *rpcError) {
		if method == "getmempoolinfo" {
			return map[string]interface{}{
				"size":          515,
				"bytes":         120246,
				"usage":         780016,
				"maxmempool":    300000000,
				"mempoolminfee": 0.00001,
			}, nil
		}
		return nil, nil
	})
	defer server.Close()

	checker := newTestLitecoinChecker(server.URL)
	ctx := context.Background()

	info, err := checker.GetMempoolInfo(ctx)

	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, 515, info.Size)
	assert.Equal(t, int64(120246), info.Bytes)
	assert.Equal(t, int64(780016), info.Usage)
	assert.Equal(t, int64(300000000), info.MaxMempool)
}

// =============================================================================
// Full Diagnostics Tests
// =============================================================================

func TestLitecoinHealthChecker_RunDiagnostics_Healthy(t *testing.T) {
	server := mockRPCServer(t, func(method string, params json.RawMessage) (interface{}, *rpcError) {
		switch method {
		case "getblockchaininfo":
			return map[string]interface{}{
				"chain":                "main",
				"blocks":               3026575,
				"verificationprogress": 0.9999999,
				"initialblockdownload": false,
			}, nil
		case "getblockcount":
			return 3026575, nil
		case "getblocktemplate":
			return map[string]interface{}{
				"height": 3026576,
			}, nil
		case "getmempoolinfo":
			return map[string]interface{}{
				"size":  100,
				"bytes": 50000,
			}, nil
		}
		return nil, nil
	})
	defer server.Close()

	checker := newTestLitecoinChecker(server.URL)
	ctx := context.Background()

	diag, err := checker.RunDiagnostics(ctx)

	require.NoError(t, err)
	require.NotNil(t, diag)
	assert.Equal(t, "litecoin", diag.ChainName)
	assert.True(t, diag.RPCConnected)
	assert.InDelta(t, 0.9999999, diag.SyncProgress, 0.0000001)
	assert.False(t, diag.IsIBD)
	assert.Equal(t, int64(3026575), diag.BlockHeight)
	assert.True(t, diag.BlockTemplateOK)
	assert.Empty(t, diag.ChainSpecificErrors)
	assert.NotNil(t, diag.Mempool)
}

func TestLitecoinHealthChecker_RunDiagnostics_MWEBFailure(t *testing.T) {
	callCount := 0
	server := mockRPCServer(t, func(method string, params json.RawMessage) (interface{}, *rpcError) {
		callCount++
		switch method {
		case "getblockchaininfo":
			return map[string]interface{}{
				"verificationprogress": 0.9999999,
				"initialblockdownload": false,
			}, nil
		case "getblockcount":
			return 3026575, nil
		case "getblocktemplate":
			return nil, &rpcError{
				Code:    -1,
				Message: "MWEB::Node::ConnectBlock(): Failed",
			}
		case "getmempoolinfo":
			return map[string]interface{}{"size": 500}, nil
		}
		return nil, nil
	})
	defer server.Close()

	checker := newTestLitecoinChecker(server.URL)
	ctx := context.Background()

	diag, err := checker.RunDiagnostics(ctx)

	require.NoError(t, err)
	require.NotNil(t, diag)
	assert.True(t, diag.RPCConnected)
	assert.False(t, diag.BlockTemplateOK)
	assert.Contains(t, diag.ChainSpecificErrors, "MWEB_FAILURE")
	assert.Contains(t, diag.BlockTemplateError, "MWEB")
}

func TestLitecoinHealthChecker_RunDiagnostics_RPCDown(t *testing.T) {
	checker := newTestLitecoinChecker("http://localhost:99999")
	ctx := context.Background()

	diag, err := checker.RunDiagnostics(ctx)

	require.NoError(t, err)
	require.NotNil(t, diag)
	assert.False(t, diag.RPCConnected)
	assert.NotEmpty(t, diag.RPCError)
}

// =============================================================================
// Error Detection Helper Tests
// =============================================================================

func TestIsMWEBError_True(t *testing.T) {
	testCases := []error{
		ErrMWEBFailure,
		errors.New("mweb-connect-failed"),
		errors.New("MWEB::Node::ConnectBlock failed"),
		errors.New("PedersenCommitSum error"),
		errors.New("secp256k1_pedersen_commit_sum failed"),
	}

	for _, tc := range testCases {
		assert.True(t, IsMWEBError(tc), "Expected IsMWEBError to return true for: %v", tc)
	}
}

func TestIsMWEBError_False(t *testing.T) {
	testCases := []error{
		nil,
		errors.New("some other error"),
		errors.New("connection refused"),
		ErrNodeInIBD,
	}

	for _, tc := range testCases {
		assert.False(t, IsMWEBError(tc), "Expected IsMWEBError to return false for: %v", tc)
	}
}

func TestIsIBDError_True(t *testing.T) {
	testCases := []error{
		ErrNodeInIBD,
		errors.New("node is in initial sync"),
		errors.New("initialblockdownload is true"),
		errors.New("Loading block index..."),
	}

	for _, tc := range testCases {
		assert.True(t, IsIBDError(tc), "Expected IsIBDError to return true for: %v", tc)
	}
}

func TestIsIBDError_False(t *testing.T) {
	testCases := []error{
		nil,
		errors.New("some other error"),
		errors.New("MWEB error"),
		ErrMWEBFailure,
	}

	for _, tc := range testCases {
		assert.False(t, IsIBDError(tc), "Expected IsIBDError to return false for: %v", tc)
	}
}

func TestContainsAny(t *testing.T) {
	patterns := []string{"foo", "bar", "baz"}

	assert.True(t, containsAny("contains foo here", patterns))
	assert.True(t, containsAny("CONTAINS FOO HERE", patterns)) // case insensitive
	assert.True(t, containsAny("has bar inside", patterns))
	assert.False(t, containsAny("no match", patterns))
	assert.False(t, containsAny("", patterns))
}

// =============================================================================
// Interface Compliance
// =============================================================================

func TestLitecoinHealthChecker_ImplementsNodeHealthChecker(t *testing.T) {
	var _ NodeHealthChecker = (*LitecoinHealthChecker)(nil)
}

func TestLitecoinHealthChecker_ImplementsRPCChecker(t *testing.T) {
	var _ RPCChecker = (*LitecoinHealthChecker)(nil)
}

func TestLitecoinHealthChecker_ImplementsSyncChecker(t *testing.T) {
	var _ SyncChecker = (*LitecoinHealthChecker)(nil)
}

func TestLitecoinHealthChecker_ImplementsBlockTemplateChecker(t *testing.T) {
	var _ BlockTemplateChecker = (*LitecoinHealthChecker)(nil)
}

func TestLitecoinHealthChecker_ImplementsChainDiagnostics(t *testing.T) {
	var _ ChainDiagnostics = (*LitecoinHealthChecker)(nil)
}
