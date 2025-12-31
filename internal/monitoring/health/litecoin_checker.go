package health

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Known Litecoin-specific error patterns that require node restart.
var (
	// MWEB errors indicate MimbleWimble Extension Block validation failures.
	MWEBErrorPatterns = []string{
		"mweb-connect-failed",
		"MWEB::Node::ConnectBlock",
		"PedersenCommitSum",
		"secp256k1_pedersen_commit_sum",
	}

	// IBD errors indicate node is still syncing or loading.
	IBDErrorPatterns = []string{
		"initial sync",
		"initialblockdownload",
		"Loading block index",
		"Rewinding blocks",
		"Verifying blocks",
		"Loading banlist",
		"Loading wallet",
		"Rescanning",
	}

	ErrMWEBFailure       = errors.New("MWEB block validation failure detected")
	ErrNodeInIBD         = errors.New("node is in initial block download")
	ErrRPCUnreachable    = errors.New("RPC endpoint unreachable")
	ErrTemplateGenFailed = errors.New("block template generation failed")
)

// LitecoinHealthChecker implements NodeHealthChecker for Litecoin nodes.
type LitecoinHealthChecker struct {
	config     *LitecoinNodeConfig
	httpClient *http.Client
	timeout    time.Duration
}

// NewLitecoinHealthChecker creates a new Litecoin health checker.
func NewLitecoinHealthChecker(config *LitecoinNodeConfig, timeout time.Duration) *LitecoinHealthChecker {
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	return &LitecoinHealthChecker{
		config:  config,
		timeout: timeout,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// GetChainName returns the blockchain identifier.
func (c *LitecoinHealthChecker) GetChainName() string {
	return "litecoin"
}

// CheckRPCConnectivity verifies basic RPC connectivity.
func (c *LitecoinHealthChecker) CheckRPCConnectivity(ctx context.Context) error {
	_, err := c.rpcCall(ctx, "getblockchaininfo", nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrRPCUnreachable, err)
	}
	return nil
}

// GetSyncProgress returns the sync progress (0.0 to 1.0).
func (c *LitecoinHealthChecker) GetSyncProgress(ctx context.Context) (float64, error) {
	result, err := c.rpcCall(ctx, "getblockchaininfo", nil)
	if err != nil {
		return 0, err
	}

	var info struct {
		VerificationProgress float64 `json:"verificationprogress"`
	}
	if err := json.Unmarshal(result, &info); err != nil {
		return 0, fmt.Errorf("failed to parse blockchain info: %w", err)
	}

	return info.VerificationProgress, nil
}

// IsInitialBlockDownload returns true if node is in IBD mode.
func (c *LitecoinHealthChecker) IsInitialBlockDownload(ctx context.Context) (bool, error) {
	result, err := c.rpcCall(ctx, "getblockchaininfo", nil)
	if err != nil {
		// Check if error indicates IBD
		if containsAny(err.Error(), IBDErrorPatterns) {
			return true, nil
		}
		return false, err
	}

	var info struct {
		InitialBlockDownload bool `json:"initialblockdownload"`
	}
	if err := json.Unmarshal(result, &info); err != nil {
		return false, fmt.Errorf("failed to parse blockchain info: %w", err)
	}

	return info.InitialBlockDownload, nil
}

// CheckBlockTemplateGeneration attempts to generate a block template.
// This is the critical check for mining pool operations.
func (c *LitecoinHealthChecker) CheckBlockTemplateGeneration(ctx context.Context) error {
	params := []interface{}{
		map[string]interface{}{
			"rules": []string{"segwit"},
		},
	}

	_, err := c.rpcCall(ctx, "getblocktemplate", params)
	if err != nil {
		// Check for MWEB-specific errors
		errStr := err.Error()
		if containsAny(errStr, MWEBErrorPatterns) {
			return fmt.Errorf("%w: %v", ErrMWEBFailure, err)
		}
		// Check for IBD errors
		if containsAny(errStr, IBDErrorPatterns) {
			return fmt.Errorf("%w: %v", ErrNodeInIBD, err)
		}
		return fmt.Errorf("%w: %v", ErrTemplateGenFailed, err)
	}

	return nil
}

// GetMempoolInfo returns mempool statistics.
func (c *LitecoinHealthChecker) GetMempoolInfo(ctx context.Context) (*MempoolInfo, error) {
	result, err := c.rpcCall(ctx, "getmempoolinfo", nil)
	if err != nil {
		return nil, err
	}

	var info struct {
		Size          int     `json:"size"`
		Bytes         int64   `json:"bytes"`
		Usage         int64   `json:"usage"`
		MaxMempool    int64   `json:"maxmempool"`
		MempoolMinFee float64 `json:"mempoolminfee"`
	}
	if err := json.Unmarshal(result, &info); err != nil {
		return nil, fmt.Errorf("failed to parse mempool info: %w", err)
	}

	return &MempoolInfo{
		Size:          info.Size,
		Bytes:         info.Bytes,
		Usage:         info.Usage,
		MaxMempool:    info.MaxMempool,
		MempoolMinFee: info.MempoolMinFee,
	}, nil
}

// RunDiagnostics performs a comprehensive health check.
func (c *LitecoinHealthChecker) RunDiagnostics(ctx context.Context) (*NodeDiagnostics, error) {
	diag := &NodeDiagnostics{
		ChainName: c.GetChainName(),
		Timestamp: time.Now(),
	}

	// Check RPC connectivity
	rpcStart := time.Now()
	err := c.CheckRPCConnectivity(ctx)
	diag.RPCLatency = time.Since(rpcStart)
	diag.RPCConnected = err == nil
	if err != nil {
		diag.RPCError = err.Error()
	}

	// If RPC is down, return early
	if !diag.RPCConnected {
		return diag, nil
	}

	// Get sync progress
	progress, err := c.GetSyncProgress(ctx)
	if err == nil {
		diag.SyncProgress = progress
	}

	// Check IBD status
	ibd, err := c.IsInitialBlockDownload(ctx)
	if err == nil {
		diag.IsIBD = ibd
	}

	// Get block height
	height, err := c.getBlockHeight(ctx)
	if err == nil {
		diag.BlockHeight = height
	}

	// Check block template generation (critical for mining)
	templateStart := time.Now()
	err = c.CheckBlockTemplateGeneration(ctx)
	diag.BlockTemplateLatency = time.Since(templateStart)
	diag.BlockTemplateOK = err == nil
	if err != nil {
		diag.BlockTemplateError = err.Error()
		// Categorize chain-specific errors
		if errors.Is(err, ErrMWEBFailure) {
			diag.ChainSpecificErrors = append(diag.ChainSpecificErrors, "MWEB_FAILURE")
		}
		if errors.Is(err, ErrNodeInIBD) {
			diag.ChainSpecificErrors = append(diag.ChainSpecificErrors, "NODE_IN_IBD")
			diag.IsIBD = true // Mark as IBD to prevent unnecessary restarts during loading
		}
	}

	// Get mempool info
	mempool, err := c.GetMempoolInfo(ctx)
	if err == nil {
		diag.Mempool = mempool
	}

	return diag, nil
}

// getBlockHeight returns the current block height.
func (c *LitecoinHealthChecker) getBlockHeight(ctx context.Context) (int64, error) {
	result, err := c.rpcCall(ctx, "getblockcount", nil)
	if err != nil {
		return 0, err
	}

	var height int64
	if err := json.Unmarshal(result, &height); err != nil {
		return 0, fmt.Errorf("failed to parse block count: %w", err)
	}

	return height, nil
}

// rpcCall makes a JSON-RPC call to the Litecoin node.
func (c *LitecoinHealthChecker) rpcCall(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	if params == nil {
		params = []interface{}{}
	}

	reqBody := map[string]interface{}{
		"jsonrpc": "1.0",
		"id":      "health-check",
		"method":  method,
		"params":  params,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.config.RPCURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.config.RPCUser, c.config.RPCPassword)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("RPC request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var rpcResp struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to parse RPC response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}

// IsMWEBError checks if an error is an MWEB-related failure.
func IsMWEBError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrMWEBFailure) || containsAny(err.Error(), MWEBErrorPatterns)
}

// IsIBDError checks if an error indicates node is in IBD.
func IsIBDError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrNodeInIBD) || containsAny(err.Error(), IBDErrorPatterns)
}

// containsAny checks if s contains any of the patterns (case-insensitive).
func containsAny(s string, patterns []string) bool {
	lower := strings.ToLower(s)
	for _, p := range patterns {
		if strings.Contains(lower, strings.ToLower(p)) {
			return true
		}
	}
	return false
}

// Ensure LitecoinHealthChecker implements NodeHealthChecker.
var _ NodeHealthChecker = (*LitecoinHealthChecker)(nil)
