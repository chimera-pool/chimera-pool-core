package payouts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

// =============================================================================
// WALLET CONFIGURATION
// =============================================================================

// WalletConfig holds configuration for the wallet client
type WalletConfig struct {
	RPCURL         string        `json:"rpc_url" yaml:"rpc_url"`
	RPCUser        string        `json:"rpc_user" yaml:"rpc_user"`
	RPCPassword    string        `json:"rpc_password" yaml:"rpc_password"`
	WalletPassword string        `json:"wallet_password" yaml:"wallet_password"`
	Network        string        `json:"network" yaml:"network"` // mainnet, testnet
	Timeout        time.Duration `json:"timeout" yaml:"timeout"`
}

// =============================================================================
// RPC TYPES
// =============================================================================

// RPCRequest represents a JSON-RPC request
type RPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int64         `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

// RPCResponse represents a JSON-RPC response
type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *RPCError       `json:"error"`
}

// RPCError represents a JSON-RPC error
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("RPC error %d: %s", e.Code, e.Message)
}

// =============================================================================
// LITECOIN WALLET CLIENT IMPLEMENTATION
// =============================================================================

// LitecoinWalletClient implements WalletClient for Litecoin
type LitecoinWalletClient struct {
	config    WalletConfig
	client    *http.Client
	requestID int64
}

// NewLitecoinWalletClient creates a new Litecoin wallet client
func NewLitecoinWalletClient(config WalletConfig) (*LitecoinWalletClient, error) {
	if config.RPCURL == "" {
		return nil, fmt.Errorf("RPC URL is required")
	}

	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}

	if config.Network == "" {
		config.Network = "mainnet"
	}

	return &LitecoinWalletClient{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}, nil
}

// ValidateAddress validates a Litecoin address
func (c *LitecoinWalletClient) ValidateAddress(address string) bool {
	if len(address) < 26 {
		return false
	}

	// Mainnet addresses
	if c.config.Network == "mainnet" || c.config.Network == "" {
		// P2PKH (starts with L)
		if strings.HasPrefix(address, "L") && len(address) >= 26 {
			return true
		}
		// P2SH (starts with M)
		if strings.HasPrefix(address, "M") && len(address) >= 26 {
			return true
		}
		// Bech32 (starts with ltc1)
		if strings.HasPrefix(address, "ltc1") && len(address) >= 42 {
			return true
		}
	}

	// Testnet addresses
	if c.config.Network == "testnet" {
		// Testnet P2PKH (starts with m or n)
		if (strings.HasPrefix(address, "m") || strings.HasPrefix(address, "n")) && len(address) >= 26 {
			return true
		}
		// Testnet bech32 (starts with tltc1)
		if strings.HasPrefix(address, "tltc1") && len(address) >= 42 {
			return true
		}
	}

	return false
}

// GetBalance returns the wallet balance in litoshis
func (c *LitecoinWalletClient) GetBalance(ctx context.Context) (int64, error) {
	result, err := c.call(ctx, "getbalance", []interface{}{})
	if err != nil {
		return 0, err
	}

	var balance float64
	if err := json.Unmarshal(result, &balance); err != nil {
		return 0, fmt.Errorf("failed to parse balance: %w", err)
	}

	// Convert LTC to litoshis (1 LTC = 100,000,000 litoshis)
	return int64(balance * 100000000), nil
}

// SendTransaction sends a transaction and returns the tx hash
func (c *LitecoinWalletClient) SendTransaction(ctx context.Context, address string, amount int64) (string, error) {
	// Validate address
	if !c.ValidateAddress(address) {
		return "", fmt.Errorf("invalid address: %s", address)
	}

	// Validate amount
	if amount <= 0 {
		return "", fmt.Errorf("amount must be positive")
	}

	// Convert litoshis to LTC
	amountLTC := float64(amount) / 100000000

	result, err := c.call(ctx, "sendtoaddress", []interface{}{address, amountLTC})
	if err != nil {
		return "", err
	}

	var txHash string
	if err := json.Unmarshal(result, &txHash); err != nil {
		return "", fmt.Errorf("failed to parse tx hash: %w", err)
	}

	return txHash, nil
}

// EstimateFee estimates the fee for a transaction (returns litoshis per KB)
func (c *LitecoinWalletClient) EstimateFee(ctx context.Context, confirmTarget int) (int64, error) {
	result, err := c.call(ctx, "estimatesmartfee", []interface{}{confirmTarget})
	if err != nil {
		return 0, err
	}

	var feeResult struct {
		FeeRate float64 `json:"feerate"`
		Blocks  int     `json:"blocks"`
	}
	if err := json.Unmarshal(result, &feeResult); err != nil {
		return 0, fmt.Errorf("failed to parse fee estimate: %w", err)
	}

	// Convert LTC/KB to litoshis/KB
	return int64(feeResult.FeeRate * 100000000), nil
}

// UnlockWallet unlocks the wallet for the specified duration
func (c *LitecoinWalletClient) UnlockWallet(ctx context.Context, seconds int) error {
	if c.config.WalletPassword == "" {
		return nil // Wallet not encrypted
	}

	_, err := c.call(ctx, "walletpassphrase", []interface{}{c.config.WalletPassword, seconds})
	return err
}

// LockWallet locks the wallet
func (c *LitecoinWalletClient) LockWallet(ctx context.Context) error {
	_, err := c.call(ctx, "walletlock", []interface{}{})
	return err
}

// GetNewAddress generates a new receiving address
func (c *LitecoinWalletClient) GetNewAddress(ctx context.Context, label string) (string, error) {
	params := []interface{}{}
	if label != "" {
		params = append(params, label)
	}

	result, err := c.call(ctx, "getnewaddress", params)
	if err != nil {
		return "", err
	}

	var address string
	if err := json.Unmarshal(result, &address); err != nil {
		return "", fmt.Errorf("failed to parse address: %w", err)
	}

	return address, nil
}

// GetTransaction gets transaction details
func (c *LitecoinWalletClient) GetTransaction(ctx context.Context, txHash string) (*TransactionInfo, error) {
	result, err := c.call(ctx, "gettransaction", []interface{}{txHash})
	if err != nil {
		return nil, err
	}

	var info TransactionInfo
	if err := json.Unmarshal(result, &info); err != nil {
		return nil, fmt.Errorf("failed to parse transaction: %w", err)
	}

	return &info, nil
}

// TransactionInfo holds transaction details
type TransactionInfo struct {
	TxID          string  `json:"txid"`
	Amount        float64 `json:"amount"`
	Fee           float64 `json:"fee"`
	Confirmations int     `json:"confirmations"`
	BlockHash     string  `json:"blockhash"`
	BlockTime     int64   `json:"blocktime"`
	Time          int64   `json:"time"`
}

// call makes an RPC call to the Litecoin node
func (c *LitecoinWalletClient) call(ctx context.Context, method string, params []interface{}) (json.RawMessage, error) {
	id := atomic.AddInt64(&c.requestID, 1)

	req := RPCRequest{
		JSONRPC: "1.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.config.RPCURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.SetBasicAuth(c.config.RPCUser, c.config.RPCPassword)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("RPC request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var rpcResp RPCResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, rpcResp.Error
	}

	return rpcResp.Result, nil
}
