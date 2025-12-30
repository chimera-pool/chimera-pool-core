package network

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// NetworkConfig represents a blockchain network configuration loaded from the database
type NetworkConfig struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	Symbol          string    `json:"symbol"`
	DisplayName     string    `json:"display_name"`
	IsActive        bool      `json:"is_active"`
	IsDefault       bool      `json:"is_default"`
	Algorithm       string    `json:"algorithm"`
	AlgorithmParams string    `json:"algorithm_params"`
	RPCURL          string    `json:"rpc_url"`
	RPCUser         string    `json:"rpc_user"`
	RPCPassword     string    `json:"rpc_password"`
	RPCTimeout      int       `json:"rpc_timeout_ms"`
	StratumPort     int       `json:"stratum_port"`
	VardiffEnabled  bool      `json:"vardiff_enabled"`
	VardiffMin      float64   `json:"vardiff_min"`
	VardiffMax      float64   `json:"vardiff_max"`
	VardiffTarget   int       `json:"vardiff_target_time"`
	BlockTimeTarget int       `json:"block_time_target"`
	BlockReward     float64   `json:"block_reward"`
	PoolWallet      string    `json:"pool_wallet_address"`
	PoolFeePercent  float64   `json:"pool_fee_percent"`
	MinPayout       float64   `json:"min_payout_threshold"`
	ExplorerURL     string    `json:"explorer_url"`
}

// ConfigLoader loads network configuration from the database
type ConfigLoader struct {
	db            *sql.DB
	currentConfig *NetworkConfig
	configMutex   sync.RWMutex
	observers     []func(*NetworkConfig)
	observerMutex sync.Mutex
	pollInterval  time.Duration
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewConfigLoader creates a new network configuration loader
func NewConfigLoader(db *sql.DB) *ConfigLoader {
	ctx, cancel := context.WithCancel(context.Background())
	return &ConfigLoader{
		db:           db,
		observers:    make([]func(*NetworkConfig), 0),
		pollInterval: 30 * time.Second,
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start begins polling for configuration changes
func (cl *ConfigLoader) Start() error {
	// Load initial configuration
	config, err := cl.loadActiveNetwork()
	if err != nil {
		return fmt.Errorf("failed to load initial network config: %w", err)
	}

	cl.configMutex.Lock()
	cl.currentConfig = config
	cl.configMutex.Unlock()

	// Start background polling
	go cl.pollForChanges()

	return nil
}

// Stop stops the configuration loader
func (cl *ConfigLoader) Stop() {
	cl.cancel()
}

// GetActiveNetwork returns the current active network configuration
func (cl *ConfigLoader) GetActiveNetwork() *NetworkConfig {
	cl.configMutex.RLock()
	defer cl.configMutex.RUnlock()
	return cl.currentConfig
}

// GetActiveNetworkID returns just the network ID (for share recording)
func (cl *ConfigLoader) GetActiveNetworkID() *uuid.UUID {
	cl.configMutex.RLock()
	defer cl.configMutex.RUnlock()
	if cl.currentConfig != nil {
		return &cl.currentConfig.ID
	}
	return nil
}

// RegisterObserver registers a callback for configuration changes
func (cl *ConfigLoader) RegisterObserver(callback func(*NetworkConfig)) {
	cl.observerMutex.Lock()
	defer cl.observerMutex.Unlock()
	cl.observers = append(cl.observers, callback)
}

// loadActiveNetwork loads the active network configuration from the database
func (cl *ConfigLoader) loadActiveNetwork() (*NetworkConfig, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT 
			id, name, symbol, display_name, is_active, is_default,
			algorithm, COALESCE(algorithm_params::text, '{}'),
			rpc_url, COALESCE(rpc_user, ''), COALESCE(rpc_password, ''),
			COALESCE(rpc_timeout_ms, 30000),
			stratum_port, vardiff_enabled, vardiff_min, vardiff_max, vardiff_target_time,
			block_time_target, COALESCE(block_reward, 0),
			pool_wallet_address, pool_fee_percent, min_payout_threshold,
			COALESCE(explorer_url, '')
		FROM network_configs 
		WHERE is_active = true AND is_default = true
		LIMIT 1
	`

	config := &NetworkConfig{}
	err := cl.db.QueryRowContext(ctx, query).Scan(
		&config.ID, &config.Name, &config.Symbol, &config.DisplayName,
		&config.IsActive, &config.IsDefault,
		&config.Algorithm, &config.AlgorithmParams,
		&config.RPCURL, &config.RPCUser, &config.RPCPassword, &config.RPCTimeout,
		&config.StratumPort, &config.VardiffEnabled, &config.VardiffMin,
		&config.VardiffMax, &config.VardiffTarget,
		&config.BlockTimeTarget, &config.BlockReward,
		&config.PoolWallet, &config.PoolFeePercent, &config.MinPayout,
		&config.ExplorerURL,
	)

	if err == sql.ErrNoRows {
		// Try to get any active network as fallback
		query = `
			SELECT 
				id, name, symbol, display_name, is_active, is_default,
				algorithm, COALESCE(algorithm_params::text, '{}'),
				rpc_url, COALESCE(rpc_user, ''), COALESCE(rpc_password, ''),
				COALESCE(rpc_timeout_ms, 30000),
				stratum_port, vardiff_enabled, vardiff_min, vardiff_max, vardiff_target_time,
				block_time_target, COALESCE(block_reward, 0),
				pool_wallet_address, pool_fee_percent, min_payout_threshold,
				COALESCE(explorer_url, '')
			FROM network_configs 
			WHERE is_active = true
			LIMIT 1
		`
		err = cl.db.QueryRowContext(ctx, query).Scan(
			&config.ID, &config.Name, &config.Symbol, &config.DisplayName,
			&config.IsActive, &config.IsDefault,
			&config.Algorithm, &config.AlgorithmParams,
			&config.RPCURL, &config.RPCUser, &config.RPCPassword, &config.RPCTimeout,
			&config.StratumPort, &config.VardiffEnabled, &config.VardiffMin,
			&config.VardiffMax, &config.VardiffTarget,
			&config.BlockTimeTarget, &config.BlockReward,
			&config.PoolWallet, &config.PoolFeePercent, &config.MinPayout,
			&config.ExplorerURL,
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query active network: %w", err)
	}

	return config, nil
}

// pollForChanges periodically checks for network configuration changes
func (cl *ConfigLoader) pollForChanges() {
	ticker := time.NewTicker(cl.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cl.ctx.Done():
			return
		case <-ticker.C:
			newConfig, err := cl.loadActiveNetwork()
			if err != nil {
				continue // Keep using current config on error
			}

			cl.configMutex.RLock()
			currentID := cl.currentConfig.ID
			cl.configMutex.RUnlock()

			// Check if network changed
			if newConfig.ID != currentID {
				cl.configMutex.Lock()
				cl.currentConfig = newConfig
				cl.configMutex.Unlock()

				// Notify observers of network change
				cl.notifyObservers(newConfig)
			}
		}
	}
}

// notifyObservers notifies all registered observers of a configuration change
func (cl *ConfigLoader) notifyObservers(config *NetworkConfig) {
	cl.observerMutex.Lock()
	observers := make([]func(*NetworkConfig), len(cl.observers))
	copy(observers, cl.observers)
	cl.observerMutex.Unlock()

	for _, observer := range observers {
		go observer(config)
	}
}

// GetRPCConfig returns RPC configuration for the current network
func (cl *ConfigLoader) GetRPCConfig() (url, user, password string, timeout int) {
	cl.configMutex.RLock()
	defer cl.configMutex.RUnlock()
	if cl.currentConfig == nil {
		return "", "", "", 30000
	}
	return cl.currentConfig.RPCURL, cl.currentConfig.RPCUser, cl.currentConfig.RPCPassword, cl.currentConfig.RPCTimeout
}

// IsScrypt returns true if the current network uses Scrypt algorithm
func (cl *ConfigLoader) IsScrypt() bool {
	cl.configMutex.RLock()
	defer cl.configMutex.RUnlock()
	if cl.currentConfig == nil {
		return false
	}
	return cl.currentConfig.Algorithm == "scrypt"
}

// IsSHA256 returns true if the current network uses SHA256 algorithm
func (cl *ConfigLoader) IsSHA256() bool {
	cl.configMutex.RLock()
	defer cl.configMutex.RUnlock()
	if cl.currentConfig == nil {
		return false
	}
	return cl.currentConfig.Algorithm == "sha256"
}
