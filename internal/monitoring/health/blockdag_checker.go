package health

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

var (
	ErrBlockDAGRPCFailed = errors.New("BlockDAG RPC call failed")
)

// BlockDAGHealthChecker implements NodeHealthChecker for BlockDAG nodes.
// This is a stub implementation ready for full integration when BlockDAG launches.
type BlockDAGHealthChecker struct {
	config     *BlockDAGNodeConfig
	httpClient *http.Client
	timeout    time.Duration
}

// NewBlockDAGHealthChecker creates a new BlockDAG health checker.
func NewBlockDAGHealthChecker(config *BlockDAGNodeConfig, timeout time.Duration) *BlockDAGHealthChecker {
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	return &BlockDAGHealthChecker{
		config:  config,
		timeout: timeout,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// GetChainName returns the blockchain identifier.
func (c *BlockDAGHealthChecker) GetChainName() string {
	return "blockdag"
}

// CheckRPCConnectivity verifies basic RPC connectivity using eth_blockNumber.
func (c *BlockDAGHealthChecker) CheckRPCConnectivity(ctx context.Context) error {
	_, err := c.ethCall(ctx, "eth_blockNumber", nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrRPCUnreachable, err)
	}
	return nil
}

// GetSyncProgress returns the sync progress (0.0 to 1.0).
// For BlockDAG, we compare current block to highest known block.
func (c *BlockDAGHealthChecker) GetSyncProgress(ctx context.Context) (float64, error) {
	result, err := c.ethCall(ctx, "eth_syncing", nil)
	if err != nil {
		return 0, err
	}

	// If result is "false", node is fully synced
	var syncing bool
	if err := json.Unmarshal(result, &syncing); err == nil && !syncing {
		return 1.0, nil
	}

	// Otherwise parse sync status
	var syncStatus struct {
		CurrentBlock string `json:"currentBlock"`
		HighestBlock string `json:"highestBlock"`
	}
	if err := json.Unmarshal(result, &syncStatus); err != nil {
		return 0, fmt.Errorf("failed to parse sync status: %w", err)
	}

	// Parse hex values
	var current, highest uint64
	fmt.Sscanf(syncStatus.CurrentBlock, "0x%x", &current)
	fmt.Sscanf(syncStatus.HighestBlock, "0x%x", &highest)

	if highest == 0 {
		return 0, nil
	}
	return float64(current) / float64(highest), nil
}

// IsInitialBlockDownload returns true if node is syncing.
func (c *BlockDAGHealthChecker) IsInitialBlockDownload(ctx context.Context) (bool, error) {
	progress, err := c.GetSyncProgress(ctx)
	if err != nil {
		return false, err
	}
	return progress < 0.9999, nil
}

// CheckBlockTemplateGeneration checks if mining is possible.
// For BlockDAG, we verify the node can accept transactions.
func (c *BlockDAGHealthChecker) CheckBlockTemplateGeneration(ctx context.Context) error {
	// Check if we can get pending transactions (indicates node is ready for mining)
	_, err := c.ethCall(ctx, "eth_getBlockByNumber", []interface{}{"pending", false})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrTemplateGenFailed, err)
	}
	return nil
}

// RunDiagnostics performs a comprehensive health check.
func (c *BlockDAGHealthChecker) RunDiagnostics(ctx context.Context) (*NodeDiagnostics, error) {
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

	// Check block template generation
	templateStart := time.Now()
	err = c.CheckBlockTemplateGeneration(ctx)
	diag.BlockTemplateLatency = time.Since(templateStart)
	diag.BlockTemplateOK = err == nil
	if err != nil {
		diag.BlockTemplateError = err.Error()
	}

	return diag, nil
}

// getBlockHeight returns the current block height.
func (c *BlockDAGHealthChecker) getBlockHeight(ctx context.Context) (int64, error) {
	result, err := c.ethCall(ctx, "eth_blockNumber", nil)
	if err != nil {
		return 0, err
	}

	var hexHeight string
	if err := json.Unmarshal(result, &hexHeight); err != nil {
		return 0, fmt.Errorf("failed to parse block number: %w", err)
	}

	var height int64
	fmt.Sscanf(hexHeight, "0x%x", &height)
	return height, nil
}

// ethCall makes a JSON-RPC call to the BlockDAG node (Ethereum-compatible).
func (c *BlockDAGHealthChecker) ethCall(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	if params == nil {
		params = []interface{}{}
	}

	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.config.RPCURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Body = io.NopCloser(io.NopCloser(nil))

	// Re-create request with body
	req, err = http.NewRequestWithContext(ctx, "POST", c.config.RPCURL,
		io.NopCloser(stringReader(string(body))))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

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
		return nil, fmt.Errorf("%w: %d - %s", ErrBlockDAGRPCFailed, rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}

// stringReader creates an io.Reader from a string.
type stringReaderType struct {
	s string
	i int
}

func stringReader(s string) io.Reader {
	return &stringReaderType{s: s}
}

func (r *stringReaderType) Read(p []byte) (n int, err error) {
	if r.i >= len(r.s) {
		return 0, io.EOF
	}
	n = copy(p, r.s[r.i:])
	r.i += n
	return
}

// Ensure BlockDAGHealthChecker implements NodeHealthChecker.
var _ NodeHealthChecker = (*BlockDAGHealthChecker)(nil)
