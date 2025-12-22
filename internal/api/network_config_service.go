package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// NETWORK CONFIGURATION SERVICE
// ISP-compliant service for multi-coin network management
// =============================================================================

// Common errors
var (
	ErrNetworkNotFound         = errors.New("network configuration not found")
	ErrNetworkAlreadyExists    = errors.New("network with this name already exists")
	ErrCannotDeactivateDefault = errors.New("cannot deactivate the default network")
	ErrInvalidNetworkConfig    = errors.New("invalid network configuration")
)

// -----------------------------------------------------------------------------
// Network Configuration Models
// -----------------------------------------------------------------------------

// NetworkConfig represents a blockchain network configuration
type NetworkConfig struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Symbol      string    `json:"symbol"`
	DisplayName string    `json:"display_name"`
	IsActive    bool      `json:"is_active"`
	IsDefault   bool      `json:"is_default"`

	// Algorithm
	Algorithm        string                 `json:"algorithm"`
	AlgorithmVariant string                 `json:"algorithm_variant,omitempty"`
	AlgorithmParams  map[string]interface{} `json:"algorithm_params,omitempty"`

	// RPC Connection
	RPCURL         string `json:"rpc_url"`
	RPCUser        string `json:"rpc_user,omitempty"`
	RPCPassword    string `json:"-"` // Never expose in JSON
	RPCTimeoutMs   int    `json:"rpc_timeout_ms"`
	RPCURLFallback string `json:"rpc_url_fallback,omitempty"`

	// Explorer
	ExplorerURL         string `json:"explorer_url,omitempty"`
	ExplorerTxPath      string `json:"explorer_tx_path,omitempty"`
	ExplorerBlockPath   string `json:"explorer_block_path,omitempty"`
	ExplorerAddressPath string `json:"explorer_address_path,omitempty"`

	// Mining Parameters
	StratumPort       int     `json:"stratum_port"`
	VardiffEnabled    bool    `json:"vardiff_enabled"`
	VardiffMin        float64 `json:"vardiff_min"`
	VardiffMax        float64 `json:"vardiff_max"`
	VardiffTargetTime int     `json:"vardiff_target_time"`

	// Block Parameters
	BlockTimeTarget  int     `json:"block_time_target"`
	BlockReward      float64 `json:"block_reward"`
	MinConfirmations int     `json:"min_confirmations"`

	// Pool Parameters
	PoolWalletAddress  string  `json:"pool_wallet_address"`
	PoolFeePercent     float64 `json:"pool_fee_percent"`
	MinPayoutThreshold float64 `json:"min_payout_threshold"`
	PayoutIntervalSecs int     `json:"payout_interval_seconds"`

	// Chain Info
	ChainID       int    `json:"chain_id,omitempty"`
	NetworkType   string `json:"network_type"`
	AddressPrefix string `json:"address_prefix,omitempty"`

	// Metadata
	LogoURL     string `json:"logo_url,omitempty"`
	WebsiteURL  string `json:"website_url,omitempty"`
	Description string `json:"description,omitempty"`

	// Timestamps
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ActivatedAt *time.Time `json:"activated_at,omitempty"`
}

// NetworkSwitchHistory represents a network switch audit record
type NetworkSwitchHistory struct {
	ID            uuid.UUID  `json:"id"`
	FromNetworkID *uuid.UUID `json:"from_network_id,omitempty"`
	ToNetworkID   uuid.UUID  `json:"to_network_id"`
	SwitchedBy    int64      `json:"switched_by"`
	SwitchReason  string     `json:"switch_reason"`
	SwitchType    string     `json:"switch_type"`
	Status        string     `json:"status"`
	StartedAt     time.Time  `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	ErrorMessage  string     `json:"error_message,omitempty"`
}

// CreateNetworkRequest represents a request to create a new network config
type CreateNetworkRequest struct {
	Name               string                 `json:"name" binding:"required"`
	Symbol             string                 `json:"symbol" binding:"required"`
	DisplayName        string                 `json:"display_name" binding:"required"`
	Algorithm          string                 `json:"algorithm" binding:"required"`
	AlgorithmVariant   string                 `json:"algorithm_variant"`
	AlgorithmParams    map[string]interface{} `json:"algorithm_params"`
	RPCURL             string                 `json:"rpc_url" binding:"required"`
	RPCUser            string                 `json:"rpc_user"`
	RPCPassword        string                 `json:"rpc_password"`
	RPCURLFallback     string                 `json:"rpc_url_fallback"`
	ExplorerURL        string                 `json:"explorer_url"`
	StratumPort        int                    `json:"stratum_port"`
	BlockTimeTarget    int                    `json:"block_time_target"`
	BlockReward        float64                `json:"block_reward"`
	PoolWalletAddress  string                 `json:"pool_wallet_address" binding:"required"`
	PoolFeePercent     float64                `json:"pool_fee_percent"`
	MinPayoutThreshold float64                `json:"min_payout_threshold"`
	NetworkType        string                 `json:"network_type"`
	Description        string                 `json:"description"`
}

// UpdateNetworkRequest represents a request to update network config
type UpdateNetworkRequest struct {
	DisplayName        *string                `json:"display_name,omitempty"`
	Algorithm          *string                `json:"algorithm,omitempty"`
	AlgorithmVariant   *string                `json:"algorithm_variant,omitempty"`
	AlgorithmParams    map[string]interface{} `json:"algorithm_params,omitempty"`
	RPCURL             *string                `json:"rpc_url,omitempty"`
	RPCUser            *string                `json:"rpc_user,omitempty"`
	RPCPassword        *string                `json:"rpc_password,omitempty"`
	RPCURLFallback     *string                `json:"rpc_url_fallback,omitempty"`
	ExplorerURL        *string                `json:"explorer_url,omitempty"`
	StratumPort        *int                   `json:"stratum_port,omitempty"`
	VardiffEnabled     *bool                  `json:"vardiff_enabled,omitempty"`
	VardiffMin         *float64               `json:"vardiff_min,omitempty"`
	VardiffMax         *float64               `json:"vardiff_max,omitempty"`
	BlockTimeTarget    *int                   `json:"block_time_target,omitempty"`
	BlockReward        *float64               `json:"block_reward,omitempty"`
	MinConfirmations   *int                   `json:"min_confirmations,omitempty"`
	PoolWalletAddress  *string                `json:"pool_wallet_address,omitempty"`
	PoolFeePercent     *float64               `json:"pool_fee_percent,omitempty"`
	MinPayoutThreshold *float64               `json:"min_payout_threshold,omitempty"`
	Description        *string                `json:"description,omitempty"`
	IsActive           *bool                  `json:"is_active,omitempty"`
}

// SwitchNetworkRequest represents a request to switch active network
type SwitchNetworkRequest struct {
	NetworkName string `json:"network_name" binding:"required"`
	Reason      string `json:"reason"`
}

// -----------------------------------------------------------------------------
// Network Configuration Service Interfaces (ISP)
// -----------------------------------------------------------------------------

// NetworkConfigReader reads network configurations
type NetworkConfigReader interface {
	GetNetworkByID(ctx context.Context, id uuid.UUID) (*NetworkConfig, error)
	GetNetworkByName(ctx context.Context, name string) (*NetworkConfig, error)
	GetActiveNetwork(ctx context.Context) (*NetworkConfig, error)
	ListNetworks(ctx context.Context) ([]*NetworkConfig, error)
	ListActiveNetworks(ctx context.Context) ([]*NetworkConfig, error)
}

// NetworkConfigWriter writes network configurations
type NetworkConfigWriter interface {
	CreateNetwork(ctx context.Context, req *CreateNetworkRequest) (*NetworkConfig, error)
	UpdateNetwork(ctx context.Context, id uuid.UUID, req *UpdateNetworkRequest) (*NetworkConfig, error)
	DeleteNetwork(ctx context.Context, id uuid.UUID) error
}

// NetworkSwitcher handles network switching
type NetworkSwitcher interface {
	SwitchNetwork(ctx context.Context, networkName string, switchedBy int64, reason string) (*NetworkSwitchHistory, error)
	GetSwitchHistory(ctx context.Context, limit int) ([]*NetworkSwitchHistory, error)
	RollbackSwitch(ctx context.Context, historyID uuid.UUID) error
}

// NetworkConfigService combines all network config operations
type NetworkConfigService interface {
	NetworkConfigReader
	NetworkConfigWriter
	NetworkSwitcher
	TestConnection(ctx context.Context, networkID uuid.UUID) (bool, error)
}

// -----------------------------------------------------------------------------
// Database Implementation
// -----------------------------------------------------------------------------

// DBNetworkConfigService implements NetworkConfigService
type DBNetworkConfigService struct {
	db *sql.DB
}

// NewDBNetworkConfigService creates a new network config service
func NewDBNetworkConfigService(db *sql.DB) *DBNetworkConfigService {
	return &DBNetworkConfigService{db: db}
}

// GetNetworkByID retrieves a network by ID
func (s *DBNetworkConfigService) GetNetworkByID(ctx context.Context, id uuid.UUID) (*NetworkConfig, error) {
	return s.scanNetwork(s.db.QueryRowContext(ctx, `
		SELECT id, name, symbol, display_name, is_active, is_default,
			   algorithm, algorithm_variant, algorithm_params,
			   rpc_url, rpc_user, rpc_password, rpc_timeout_ms, rpc_url_fallback,
			   explorer_url, explorer_tx_path, explorer_block_path, explorer_address_path,
			   stratum_port, vardiff_enabled, vardiff_min, vardiff_max, vardiff_target_time,
			   block_time_target, block_reward, min_confirmations,
			   pool_wallet_address, pool_fee_percent, min_payout_threshold, payout_interval_seconds,
			   chain_id, network_type, address_prefix,
			   logo_url, website_url, description,
			   created_at, updated_at, activated_at
		FROM network_configs WHERE id = $1`, id))
}

// GetNetworkByName retrieves a network by name
func (s *DBNetworkConfigService) GetNetworkByName(ctx context.Context, name string) (*NetworkConfig, error) {
	return s.scanNetwork(s.db.QueryRowContext(ctx, `
		SELECT id, name, symbol, display_name, is_active, is_default,
			   algorithm, algorithm_variant, algorithm_params,
			   rpc_url, rpc_user, rpc_password, rpc_timeout_ms, rpc_url_fallback,
			   explorer_url, explorer_tx_path, explorer_block_path, explorer_address_path,
			   stratum_port, vardiff_enabled, vardiff_min, vardiff_max, vardiff_target_time,
			   block_time_target, block_reward, min_confirmations,
			   pool_wallet_address, pool_fee_percent, min_payout_threshold, payout_interval_seconds,
			   chain_id, network_type, address_prefix,
			   logo_url, website_url, description,
			   created_at, updated_at, activated_at
		FROM network_configs WHERE name = $1`, name))
}

// GetActiveNetwork retrieves the currently active default network
func (s *DBNetworkConfigService) GetActiveNetwork(ctx context.Context) (*NetworkConfig, error) {
	return s.scanNetwork(s.db.QueryRowContext(ctx, `
		SELECT id, name, symbol, display_name, is_active, is_default,
			   algorithm, algorithm_variant, algorithm_params,
			   rpc_url, rpc_user, rpc_password, rpc_timeout_ms, rpc_url_fallback,
			   explorer_url, explorer_tx_path, explorer_block_path, explorer_address_path,
			   stratum_port, vardiff_enabled, vardiff_min, vardiff_max, vardiff_target_time,
			   block_time_target, block_reward, min_confirmations,
			   pool_wallet_address, pool_fee_percent, min_payout_threshold, payout_interval_seconds,
			   chain_id, network_type, address_prefix,
			   logo_url, website_url, description,
			   created_at, updated_at, activated_at
		FROM network_configs WHERE is_active = true AND is_default = true`))
}

// ListNetworks returns all network configurations
func (s *DBNetworkConfigService) ListNetworks(ctx context.Context) ([]*NetworkConfig, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, symbol, display_name, is_active, is_default,
			   algorithm, algorithm_variant, algorithm_params,
			   rpc_url, rpc_user, rpc_password, rpc_timeout_ms, rpc_url_fallback,
			   explorer_url, explorer_tx_path, explorer_block_path, explorer_address_path,
			   stratum_port, vardiff_enabled, vardiff_min, vardiff_max, vardiff_target_time,
			   block_time_target, block_reward, min_confirmations,
			   pool_wallet_address, pool_fee_percent, min_payout_threshold, payout_interval_seconds,
			   chain_id, network_type, address_prefix,
			   logo_url, website_url, description,
			   created_at, updated_at, activated_at
		FROM network_configs ORDER BY is_default DESC, name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanNetworks(rows)
}

// ListActiveNetworks returns only active network configurations
func (s *DBNetworkConfigService) ListActiveNetworks(ctx context.Context) ([]*NetworkConfig, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, symbol, display_name, is_active, is_default,
			   algorithm, algorithm_variant, algorithm_params,
			   rpc_url, rpc_user, rpc_password, rpc_timeout_ms, rpc_url_fallback,
			   explorer_url, explorer_tx_path, explorer_block_path, explorer_address_path,
			   stratum_port, vardiff_enabled, vardiff_min, vardiff_max, vardiff_target_time,
			   block_time_target, block_reward, min_confirmations,
			   pool_wallet_address, pool_fee_percent, min_payout_threshold, payout_interval_seconds,
			   chain_id, network_type, address_prefix,
			   logo_url, website_url, description,
			   created_at, updated_at, activated_at
		FROM network_configs WHERE is_active = true ORDER BY is_default DESC, name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanNetworks(rows)
}

// CreateNetwork creates a new network configuration
func (s *DBNetworkConfigService) CreateNetwork(ctx context.Context, req *CreateNetworkRequest) (*NetworkConfig, error) {
	// Set defaults
	if req.StratumPort == 0 {
		req.StratumPort = 3333
	}
	if req.BlockTimeTarget == 0 {
		req.BlockTimeTarget = 150
	}
	if req.PoolFeePercent == 0 {
		req.PoolFeePercent = 1.0
	}
	if req.NetworkType == "" {
		req.NetworkType = "mainnet"
	}

	algorithmParams, _ := json.Marshal(req.AlgorithmParams)

	var id uuid.UUID
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO network_configs (
			name, symbol, display_name, is_active, is_default,
			algorithm, algorithm_variant, algorithm_params,
			rpc_url, rpc_user, rpc_password, rpc_url_fallback,
			explorer_url, stratum_port,
			block_time_target, block_reward,
			pool_wallet_address, pool_fee_percent, min_payout_threshold,
			network_type, description
		) VALUES ($1, $2, $3, false, false, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		RETURNING id`,
		req.Name, req.Symbol, req.DisplayName,
		req.Algorithm, req.AlgorithmVariant, algorithmParams,
		req.RPCURL, req.RPCUser, req.RPCPassword, req.RPCURLFallback,
		req.ExplorerURL, req.StratumPort,
		req.BlockTimeTarget, req.BlockReward,
		req.PoolWalletAddress, req.PoolFeePercent, req.MinPayoutThreshold,
		req.NetworkType, req.Description,
	).Scan(&id)

	if err != nil {
		return nil, fmt.Errorf("failed to create network: %w", err)
	}

	return s.GetNetworkByID(ctx, id)
}

// UpdateNetwork updates an existing network configuration
func (s *DBNetworkConfigService) UpdateNetwork(ctx context.Context, id uuid.UUID, req *UpdateNetworkRequest) (*NetworkConfig, error) {
	// Build dynamic update query
	query := "UPDATE network_configs SET updated_at = NOW()"
	args := []interface{}{}
	argNum := 1

	if req.DisplayName != nil {
		query += fmt.Sprintf(", display_name = $%d", argNum)
		args = append(args, *req.DisplayName)
		argNum++
	}
	if req.Algorithm != nil {
		query += fmt.Sprintf(", algorithm = $%d", argNum)
		args = append(args, *req.Algorithm)
		argNum++
	}
	if req.AlgorithmVariant != nil {
		query += fmt.Sprintf(", algorithm_variant = $%d", argNum)
		args = append(args, *req.AlgorithmVariant)
		argNum++
	}
	if req.AlgorithmParams != nil {
		paramsJSON, _ := json.Marshal(req.AlgorithmParams)
		query += fmt.Sprintf(", algorithm_params = $%d", argNum)
		args = append(args, paramsJSON)
		argNum++
	}
	if req.RPCURL != nil {
		query += fmt.Sprintf(", rpc_url = $%d", argNum)
		args = append(args, *req.RPCURL)
		argNum++
	}
	if req.RPCUser != nil {
		query += fmt.Sprintf(", rpc_user = $%d", argNum)
		args = append(args, *req.RPCUser)
		argNum++
	}
	if req.RPCPassword != nil {
		query += fmt.Sprintf(", rpc_password = $%d", argNum)
		args = append(args, *req.RPCPassword)
		argNum++
	}
	if req.RPCURLFallback != nil {
		query += fmt.Sprintf(", rpc_url_fallback = $%d", argNum)
		args = append(args, *req.RPCURLFallback)
		argNum++
	}
	if req.ExplorerURL != nil {
		query += fmt.Sprintf(", explorer_url = $%d", argNum)
		args = append(args, *req.ExplorerURL)
		argNum++
	}
	if req.StratumPort != nil {
		query += fmt.Sprintf(", stratum_port = $%d", argNum)
		args = append(args, *req.StratumPort)
		argNum++
	}
	if req.VardiffEnabled != nil {
		query += fmt.Sprintf(", vardiff_enabled = $%d", argNum)
		args = append(args, *req.VardiffEnabled)
		argNum++
	}
	if req.BlockTimeTarget != nil {
		query += fmt.Sprintf(", block_time_target = $%d", argNum)
		args = append(args, *req.BlockTimeTarget)
		argNum++
	}
	if req.BlockReward != nil {
		query += fmt.Sprintf(", block_reward = $%d", argNum)
		args = append(args, *req.BlockReward)
		argNum++
	}
	if req.PoolWalletAddress != nil {
		query += fmt.Sprintf(", pool_wallet_address = $%d", argNum)
		args = append(args, *req.PoolWalletAddress)
		argNum++
	}
	if req.PoolFeePercent != nil {
		query += fmt.Sprintf(", pool_fee_percent = $%d", argNum)
		args = append(args, *req.PoolFeePercent)
		argNum++
	}
	if req.MinPayoutThreshold != nil {
		query += fmt.Sprintf(", min_payout_threshold = $%d", argNum)
		args = append(args, *req.MinPayoutThreshold)
		argNum++
	}
	if req.Description != nil {
		query += fmt.Sprintf(", description = $%d", argNum)
		args = append(args, *req.Description)
		argNum++
	}
	if req.IsActive != nil {
		query += fmt.Sprintf(", is_active = $%d", argNum)
		args = append(args, *req.IsActive)
		argNum++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argNum)
	args = append(args, id)

	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update network: %w", err)
	}

	return s.GetNetworkByID(ctx, id)
}

// DeleteNetwork deletes a network configuration
func (s *DBNetworkConfigService) DeleteNetwork(ctx context.Context, id uuid.UUID) error {
	// Check if it's the default network
	var isDefault bool
	err := s.db.QueryRowContext(ctx, "SELECT is_default FROM network_configs WHERE id = $1", id).Scan(&isDefault)
	if err != nil {
		return ErrNetworkNotFound
	}
	if isDefault {
		return ErrCannotDeactivateDefault
	}

	_, err = s.db.ExecContext(ctx, "DELETE FROM network_configs WHERE id = $1", id)
	return err
}

// SwitchNetwork switches the active mining network
func (s *DBNetworkConfigService) SwitchNetwork(ctx context.Context, networkName string, switchedBy int64, reason string) (*NetworkSwitchHistory, error) {
	var historyID uuid.UUID
	err := s.db.QueryRowContext(ctx,
		"SELECT switch_active_network($1, $2, $3)",
		networkName, switchedBy, reason,
	).Scan(&historyID)

	if err != nil {
		return nil, fmt.Errorf("failed to switch network: %w", err)
	}

	// Return the history record
	var history NetworkSwitchHistory
	err = s.db.QueryRowContext(ctx, `
		SELECT id, from_network_id, to_network_id, switched_by, switch_reason, 
			   switch_type, status, started_at, completed_at, error_message
		FROM network_switch_history WHERE id = $1`, historyID).Scan(
		&history.ID, &history.FromNetworkID, &history.ToNetworkID, &history.SwitchedBy,
		&history.SwitchReason, &history.SwitchType, &history.Status,
		&history.StartedAt, &history.CompletedAt, &history.ErrorMessage,
	)
	if err != nil {
		return nil, err
	}

	return &history, nil
}

// GetSwitchHistory returns network switch history
func (s *DBNetworkConfigService) GetSwitchHistory(ctx context.Context, limit int) ([]*NetworkSwitchHistory, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, from_network_id, to_network_id, switched_by, switch_reason,
			   switch_type, status, started_at, completed_at, COALESCE(error_message, '')
		FROM network_switch_history 
		ORDER BY started_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []*NetworkSwitchHistory
	for rows.Next() {
		var h NetworkSwitchHistory
		err := rows.Scan(
			&h.ID, &h.FromNetworkID, &h.ToNetworkID, &h.SwitchedBy,
			&h.SwitchReason, &h.SwitchType, &h.Status,
			&h.StartedAt, &h.CompletedAt, &h.ErrorMessage,
		)
		if err != nil {
			continue
		}
		history = append(history, &h)
	}

	return history, nil
}

// RollbackSwitch rolls back to the previous network
func (s *DBNetworkConfigService) RollbackSwitch(ctx context.Context, historyID uuid.UUID) error {
	// Get the history record
	var fromNetworkID *uuid.UUID
	err := s.db.QueryRowContext(ctx,
		"SELECT from_network_id FROM network_switch_history WHERE id = $1",
		historyID,
	).Scan(&fromNetworkID)
	if err != nil {
		return fmt.Errorf("switch history not found: %w", err)
	}

	if fromNetworkID == nil {
		return errors.New("no previous network to rollback to")
	}

	// Get the previous network name
	var networkName string
	err = s.db.QueryRowContext(ctx,
		"SELECT name FROM network_configs WHERE id = $1",
		*fromNetworkID,
	).Scan(&networkName)
	if err != nil {
		return fmt.Errorf("previous network not found: %w", err)
	}

	// Switch back
	_, err = s.SwitchNetwork(ctx, networkName, 0, "Rollback from switch "+historyID.String())
	return err
}

// TestConnection tests if the RPC connection works
func (s *DBNetworkConfigService) TestConnection(ctx context.Context, networkID uuid.UUID) (bool, error) {
	network, err := s.GetNetworkByID(ctx, networkID)
	if err != nil {
		return false, err
	}

	// TODO: Implement actual RPC connection test based on network type
	// For now, just verify the URL is not empty
	if network.RPCURL == "" {
		return false, errors.New("RPC URL is empty")
	}

	return true, nil
}

// -----------------------------------------------------------------------------
// Helper Methods
// -----------------------------------------------------------------------------

func (s *DBNetworkConfigService) scanNetwork(row *sql.Row) (*NetworkConfig, error) {
	var nc NetworkConfig
	var algorithmParams []byte
	var rpcUser, rpcPassword, rpcURLFallback sql.NullString
	var explorerURL, explorerTxPath, explorerBlockPath, explorerAddressPath sql.NullString
	var logoURL, websiteURL, description, algorithmVariant, addressPrefix sql.NullString
	var chainID sql.NullInt64
	var activatedAt sql.NullTime

	err := row.Scan(
		&nc.ID, &nc.Name, &nc.Symbol, &nc.DisplayName, &nc.IsActive, &nc.IsDefault,
		&nc.Algorithm, &algorithmVariant, &algorithmParams,
		&nc.RPCURL, &rpcUser, &rpcPassword, &nc.RPCTimeoutMs, &rpcURLFallback,
		&explorerURL, &explorerTxPath, &explorerBlockPath, &explorerAddressPath,
		&nc.StratumPort, &nc.VardiffEnabled, &nc.VardiffMin, &nc.VardiffMax, &nc.VardiffTargetTime,
		&nc.BlockTimeTarget, &nc.BlockReward, &nc.MinConfirmations,
		&nc.PoolWalletAddress, &nc.PoolFeePercent, &nc.MinPayoutThreshold, &nc.PayoutIntervalSecs,
		&chainID, &nc.NetworkType, &addressPrefix,
		&logoURL, &websiteURL, &description,
		&nc.CreatedAt, &nc.UpdatedAt, &activatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNetworkNotFound
		}
		return nil, err
	}

	// Handle nullable fields
	if algorithmVariant.Valid {
		nc.AlgorithmVariant = algorithmVariant.String
	}
	if rpcUser.Valid {
		nc.RPCUser = rpcUser.String
	}
	if rpcPassword.Valid {
		nc.RPCPassword = rpcPassword.String
	}
	if rpcURLFallback.Valid {
		nc.RPCURLFallback = rpcURLFallback.String
	}
	if explorerURL.Valid {
		nc.ExplorerURL = explorerURL.String
	}
	if explorerTxPath.Valid {
		nc.ExplorerTxPath = explorerTxPath.String
	}
	if explorerBlockPath.Valid {
		nc.ExplorerBlockPath = explorerBlockPath.String
	}
	if explorerAddressPath.Valid {
		nc.ExplorerAddressPath = explorerAddressPath.String
	}
	if logoURL.Valid {
		nc.LogoURL = logoURL.String
	}
	if websiteURL.Valid {
		nc.WebsiteURL = websiteURL.String
	}
	if description.Valid {
		nc.Description = description.String
	}
	if addressPrefix.Valid {
		nc.AddressPrefix = addressPrefix.String
	}
	if chainID.Valid {
		nc.ChainID = int(chainID.Int64)
	}
	if activatedAt.Valid {
		nc.ActivatedAt = &activatedAt.Time
	}

	// Parse algorithm params
	if len(algorithmParams) > 0 {
		json.Unmarshal(algorithmParams, &nc.AlgorithmParams)
	}

	return &nc, nil
}

func (s *DBNetworkConfigService) scanNetworks(rows *sql.Rows) ([]*NetworkConfig, error) {
	var networks []*NetworkConfig

	for rows.Next() {
		var nc NetworkConfig
		var algorithmParams []byte
		var rpcUser, rpcPassword, rpcURLFallback sql.NullString
		var explorerURL, explorerTxPath, explorerBlockPath, explorerAddressPath sql.NullString
		var logoURL, websiteURL, description, algorithmVariant, addressPrefix sql.NullString
		var chainID sql.NullInt64
		var activatedAt sql.NullTime

		err := rows.Scan(
			&nc.ID, &nc.Name, &nc.Symbol, &nc.DisplayName, &nc.IsActive, &nc.IsDefault,
			&nc.Algorithm, &algorithmVariant, &algorithmParams,
			&nc.RPCURL, &rpcUser, &rpcPassword, &nc.RPCTimeoutMs, &rpcURLFallback,
			&explorerURL, &explorerTxPath, &explorerBlockPath, &explorerAddressPath,
			&nc.StratumPort, &nc.VardiffEnabled, &nc.VardiffMin, &nc.VardiffMax, &nc.VardiffTargetTime,
			&nc.BlockTimeTarget, &nc.BlockReward, &nc.MinConfirmations,
			&nc.PoolWalletAddress, &nc.PoolFeePercent, &nc.MinPayoutThreshold, &nc.PayoutIntervalSecs,
			&chainID, &nc.NetworkType, &addressPrefix,
			&logoURL, &websiteURL, &description,
			&nc.CreatedAt, &nc.UpdatedAt, &activatedAt,
		)
		if err != nil {
			continue
		}

		// Handle nullable fields (same as scanNetwork)
		if algorithmVariant.Valid {
			nc.AlgorithmVariant = algorithmVariant.String
		}
		if rpcUser.Valid {
			nc.RPCUser = rpcUser.String
		}
		if rpcURLFallback.Valid {
			nc.RPCURLFallback = rpcURLFallback.String
		}
		if explorerURL.Valid {
			nc.ExplorerURL = explorerURL.String
		}
		if explorerTxPath.Valid {
			nc.ExplorerTxPath = explorerTxPath.String
		}
		if explorerBlockPath.Valid {
			nc.ExplorerBlockPath = explorerBlockPath.String
		}
		if explorerAddressPath.Valid {
			nc.ExplorerAddressPath = explorerAddressPath.String
		}
		if logoURL.Valid {
			nc.LogoURL = logoURL.String
		}
		if websiteURL.Valid {
			nc.WebsiteURL = websiteURL.String
		}
		if description.Valid {
			nc.Description = description.String
		}
		if addressPrefix.Valid {
			nc.AddressPrefix = addressPrefix.String
		}
		if chainID.Valid {
			nc.ChainID = int(chainID.Int64)
		}
		if activatedAt.Valid {
			nc.ActivatedAt = &activatedAt.Time
		}
		if len(algorithmParams) > 0 {
			json.Unmarshal(algorithmParams, &nc.AlgorithmParams)
		}

		networks = append(networks, &nc)
	}

	return networks, nil
}
