package v2

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LitecoinRPCClient implements RPCClient for Litecoin node communication
type LitecoinRPCClient struct {
	url      string
	user     string
	password string
	client   *http.Client
	timeout  time.Duration
}

// LitecoinRPCConfig holds configuration for the Litecoin RPC client
type LitecoinRPCConfig struct {
	URL      string
	User     string
	Password string
	Timeout  time.Duration
}

// DefaultLitecoinRPCConfig returns default configuration
func DefaultLitecoinRPCConfig() LitecoinRPCConfig {
	return LitecoinRPCConfig{
		URL:      "http://litecoind:9332",
		User:     "chimera",
		Password: "ChimeraLTC2024!",
		Timeout:  30 * time.Second,
	}
}

// NewLitecoinRPCClient creates a new Litecoin RPC client
func NewLitecoinRPCClient(config LitecoinRPCConfig) *LitecoinRPCClient {
	return &LitecoinRPCClient{
		url:      config.URL,
		user:     config.User,
		password: config.Password,
		timeout:  config.Timeout,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// rpcRequest represents a JSON-RPC request
type rpcRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

// rpcResponse represents a JSON-RPC response
type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error"`
}

// rpcError represents a JSON-RPC error
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// call makes an RPC call to the Litecoin node
func (c *LitecoinRPCClient) call(method string, params []interface{}) (json.RawMessage, error) {
	req := rpcRequest{
		JSONRPC: "1.0",
		ID:      1,
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.SetBasicAuth(c.user, c.password)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("rpc call failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var rpcResp rpcResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("rpc error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}

// GetBlockTemplate fetches a new block template from the node
func (c *LitecoinRPCClient) GetBlockTemplate() (*RPCBlockTemplate, error) {
	// Request template with segwit support
	params := []interface{}{
		map[string]interface{}{
			"rules": []string{"segwit", "mweb"},
		},
	}

	result, err := c.call("getblocktemplate", params)
	if err != nil {
		return nil, err
	}

	var template RPCBlockTemplate
	if err := json.Unmarshal(result, &template); err != nil {
		return nil, fmt.Errorf("unmarshal template: %w", err)
	}

	return &template, nil
}

// GetBlockchainInfo returns current blockchain information
func (c *LitecoinRPCClient) GetBlockchainInfo() (*RPCBlockchainInfo, error) {
	result, err := c.call("getblockchaininfo", nil)
	if err != nil {
		return nil, err
	}

	var info RPCBlockchainInfo
	if err := json.Unmarshal(result, &info); err != nil {
		return nil, fmt.Errorf("unmarshal blockchain info: %w", err)
	}

	return &info, nil
}

// GetNetworkDifficulty returns the current network difficulty
func (c *LitecoinRPCClient) GetNetworkDifficulty() (float64, error) {
	result, err := c.call("getdifficulty", nil)
	if err != nil {
		return 0, err
	}

	var difficulty float64
	if err := json.Unmarshal(result, &difficulty); err != nil {
		return 0, fmt.Errorf("unmarshal difficulty: %w", err)
	}

	return difficulty, nil
}

// SubmitBlock submits a solved block to the network
func (c *LitecoinRPCClient) SubmitBlock(blockHex string) error {
	_, err := c.call("submitblock", []interface{}{blockHex})
	return err
}

// GetBlockCount returns the current block height
func (c *LitecoinRPCClient) GetBlockCount() (int64, error) {
	result, err := c.call("getblockcount", nil)
	if err != nil {
		return 0, err
	}

	var count int64
	if err := json.Unmarshal(result, &count); err != nil {
		return 0, fmt.Errorf("unmarshal block count: %w", err)
	}

	return count, nil
}

// GetBestBlockHash returns the hash of the best (tip) block
func (c *LitecoinRPCClient) GetBestBlockHash() (string, error) {
	result, err := c.call("getbestblockhash", nil)
	if err != nil {
		return "", err
	}

	var hash string
	if err := json.Unmarshal(result, &hash); err != nil {
		return "", fmt.Errorf("unmarshal block hash: %w", err)
	}

	return hash, nil
}

// GetBlock returns a block by hash
func (c *LitecoinRPCClient) GetBlock(hash string, verbosity int) (json.RawMessage, error) {
	return c.call("getblock", []interface{}{hash, verbosity})
}

// GetRawMempool returns transaction IDs in the mempool
func (c *LitecoinRPCClient) GetRawMempool() ([]string, error) {
	result, err := c.call("getrawmempool", nil)
	if err != nil {
		return nil, err
	}

	var txids []string
	if err := json.Unmarshal(result, &txids); err != nil {
		return nil, fmt.Errorf("unmarshal mempool: %w", err)
	}

	return txids, nil
}

// ValidateAddress validates a Litecoin address
func (c *LitecoinRPCClient) ValidateAddress(address string) (*AddressValidation, error) {
	result, err := c.call("validateaddress", []interface{}{address})
	if err != nil {
		return nil, err
	}

	var validation AddressValidation
	if err := json.Unmarshal(result, &validation); err != nil {
		return nil, fmt.Errorf("unmarshal validation: %w", err)
	}

	return &validation, nil
}

// AddressValidation holds address validation result
type AddressValidation struct {
	IsValid      bool   `json:"isvalid"`
	Address      string `json:"address"`
	ScriptPubKey string `json:"scriptPubKey"`
	IsScript     bool   `json:"isscript"`
	IsWitness    bool   `json:"iswitness"`
}

// GetMiningInfo returns mining-related information
func (c *LitecoinRPCClient) GetMiningInfo() (*MiningInfo, error) {
	result, err := c.call("getmininginfo", nil)
	if err != nil {
		return nil, err
	}

	var info MiningInfo
	if err := json.Unmarshal(result, &info); err != nil {
		return nil, fmt.Errorf("unmarshal mining info: %w", err)
	}

	return &info, nil
}

// MiningInfo holds mining information
type MiningInfo struct {
	Blocks           int64   `json:"blocks"`
	Difficulty       float64 `json:"difficulty"`
	NetworkHashPS    float64 `json:"networkhashps"`
	PooledTx         int     `json:"pooledtx"`
	Chain            string  `json:"chain"`
	CurrentBlockSize int     `json:"currentblocksize,omitempty"`
	CurrentBlockTx   int     `json:"currentblocktx,omitempty"`
}

// DecodeRawTransaction decodes a raw transaction hex
func (c *LitecoinRPCClient) DecodeRawTransaction(txHex string) (json.RawMessage, error) {
	return c.call("decoderawtransaction", []interface{}{txHex})
}

// EstimateSmartFee estimates fee for confirmation in n blocks
func (c *LitecoinRPCClient) EstimateSmartFee(confTarget int) (*FeeEstimate, error) {
	result, err := c.call("estimatesmartfee", []interface{}{confTarget})
	if err != nil {
		return nil, err
	}

	var estimate FeeEstimate
	if err := json.Unmarshal(result, &estimate); err != nil {
		return nil, fmt.Errorf("unmarshal fee estimate: %w", err)
	}

	return &estimate, nil
}

// FeeEstimate holds fee estimation result
type FeeEstimate struct {
	FeeRate float64  `json:"feerate"`
	Errors  []string `json:"errors,omitempty"`
	Blocks  int      `json:"blocks"`
}

// TestConnection tests the RPC connection
func (c *LitecoinRPCClient) TestConnection() error {
	_, err := c.GetBlockCount()
	return err
}

// GetNetworkInfo returns network information
func (c *LitecoinRPCClient) GetNetworkInfo() (*NetworkInfo, error) {
	result, err := c.call("getnetworkinfo", nil)
	if err != nil {
		return nil, err
	}

	var info NetworkInfo
	if err := json.Unmarshal(result, &info); err != nil {
		return nil, fmt.Errorf("unmarshal network info: %w", err)
	}

	return &info, nil
}

// NetworkInfo holds network information
type NetworkInfo struct {
	Version         int    `json:"version"`
	Subversion      string `json:"subversion"`
	ProtocolVersion int    `json:"protocolversion"`
	Connections     int    `json:"connections"`
	Networks        []struct {
		Name      string `json:"name"`
		Limited   bool   `json:"limited"`
		Reachable bool   `json:"reachable"`
	} `json:"networks"`
	RelayFee       float64 `json:"relayfee"`
	IncrementalFee float64 `json:"incrementalfee"`
	Warnings       string  `json:"warnings"`
}

// CreateScriptPubKey creates a scriptPubKey from address for coinbase output
func (c *LitecoinRPCClient) CreateScriptPubKey(address string) ([]byte, error) {
	validation, err := c.ValidateAddress(address)
	if err != nil {
		return nil, err
	}

	if !validation.IsValid {
		return nil, fmt.Errorf("invalid address: %s", address)
	}

	scriptPubKey, err := hex.DecodeString(validation.ScriptPubKey)
	if err != nil {
		return nil, fmt.Errorf("decode scriptPubKey: %w", err)
	}

	return scriptPubKey, nil
}
