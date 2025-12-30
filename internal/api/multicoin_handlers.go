package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// UserNetworkStats represents per-user, per-network mining statistics
type UserNetworkStats struct {
	ID              uuid.UUID  `json:"id"`
	UserID          int64      `json:"user_id"`
	NetworkID       uuid.UUID  `json:"network_id"`
	NetworkName     string     `json:"network_name"`
	NetworkSymbol   string     `json:"network_symbol"`
	NetworkDisplay  string     `json:"network_display_name"`
	IsNetworkActive bool       `json:"is_network_active"`
	TotalHashrate   float64    `json:"total_hashrate"`
	AverageHashrate float64    `json:"average_hashrate"`
	PeakHashrate    float64    `json:"peak_hashrate"`
	TotalShares     int64      `json:"total_shares"`
	ValidShares     int64      `json:"valid_shares"`
	InvalidShares   int64      `json:"invalid_shares"`
	ShareEfficiency float64    `json:"share_efficiency"`
	BlocksFound     int        `json:"blocks_found"`
	TotalEarned     float64    `json:"total_earned"`
	PendingBalance  float64    `json:"pending_balance"`
	TotalPaidOut    float64    `json:"total_paid_out"`
	ActiveWorkers   int        `json:"active_workers"`
	TotalWorkers    int        `json:"total_workers"`
	LastActiveAt    *time.Time `json:"last_active_at"`
	FirstConnected  *time.Time `json:"first_connected_at"`
}

// AggregatedUserStats represents combined stats across all networks
type AggregatedUserStats struct {
	TotalNetworksMined int     `json:"total_networks_mined"`
	ActiveNetworks     int     `json:"active_networks"`
	CombinedHashrate   float64 `json:"combined_hashrate"`
	TotalSharesAll     int64   `json:"total_shares_all"`
	TotalBlocksAll     int     `json:"total_blocks_all"`
	TotalEarnedAll     float64 `json:"total_earned_all"`
	TotalPendingAll    float64 `json:"total_pending_all"`
	TotalWorkersAll    int     `json:"total_workers_all"`
}

// NetworkPoolStats represents pool-wide stats for a network
type NetworkPoolStats struct {
	NetworkID          uuid.UUID  `json:"network_id"`
	NetworkName        string     `json:"network_name"`
	NetworkSymbol      string     `json:"network_symbol"`
	TotalHashrate      float64    `json:"total_hashrate"`
	ActiveMiners       int        `json:"active_miners"`
	ActiveWorkers      int        `json:"active_workers"`
	SharesPerSecond    float64    `json:"shares_per_second"`
	BlocksFound24h     int        `json:"blocks_found_24h"`
	BlocksFoundTotal   int        `json:"blocks_found_total"`
	NetworkDifficulty  float64    `json:"network_difficulty"`
	NetworkHashrate    float64    `json:"network_hashrate"`
	CurrentBlockHeight int64      `json:"current_block_height"`
	PoolPercentage     float64    `json:"pool_percentage"`
	RPCConnected       bool       `json:"rpc_connected"`
	LastBlockAt        *time.Time `json:"last_block_at"`
}

// MultiCoinResponse wraps multi-network responses
type MultiCoinResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// GetUserNetworkStats returns stats for a user across all networks
func (h *Handler) GetUserNetworkStats(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserIDFromToken(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, MultiCoinResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	stats, err := h.fetchUserNetworkStats(userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, MultiCoinResponse{
			Success: false,
			Error:   "Failed to fetch network stats",
		})
		return
	}

	writeJSON(w, http.StatusOK, MultiCoinResponse{
		Success: true,
		Data:    stats,
	})
}

// GetUserAggregatedStats returns combined stats across all networks
func (h *Handler) GetUserAggregatedStats(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserIDFromToken(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, MultiCoinResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	stats, err := h.fetchUserAggregatedStats(userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, MultiCoinResponse{
			Success: false,
			Error:   "Failed to fetch aggregated stats",
		})
		return
	}

	writeJSON(w, http.StatusOK, MultiCoinResponse{
		Success: true,
		Data:    stats,
	})
}

// GetAllNetworkPoolStats returns pool stats for all active networks
func (h *Handler) GetAllNetworkPoolStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.fetchAllNetworkPoolStats()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, MultiCoinResponse{
			Success: false,
			Error:   "Failed to fetch pool stats",
		})
		return
	}

	writeJSON(w, http.StatusOK, MultiCoinResponse{
		Success: true,
		Data:    stats,
	})
}

// GetSupportedNetworks returns list of all configured networks
func (h *Handler) GetSupportedNetworks(w http.ResponseWriter, r *http.Request) {
	networks, err := h.fetchSupportedNetworks()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, MultiCoinResponse{
			Success: false,
			Error:   "Failed to fetch networks",
		})
		return
	}

	writeJSON(w, http.StatusOK, MultiCoinResponse{
		Success: true,
		Data:    networks,
	})
}

// Database helper functions

func (h *Handler) fetchUserNetworkStats(userID int64) ([]UserNetworkStats, error) {
	query := `
		SELECT 
			COALESCE(uns.id, gen_random_uuid()) as id,
			nc.id as network_id,
			nc.name as network_name,
			nc.symbol as network_symbol,
			nc.display_name as network_display_name,
			nc.is_active as is_network_active,
			COALESCE(uns.total_hashrate, 0) as total_hashrate,
			COALESCE(uns.average_hashrate, 0) as average_hashrate,
			COALESCE(uns.peak_hashrate, 0) as peak_hashrate,
			COALESCE(uns.total_shares, 0) as total_shares,
			COALESCE(uns.valid_shares, 0) as valid_shares,
			COALESCE(uns.invalid_shares, 0) as invalid_shares,
			COALESCE(uns.share_efficiency, 100) as share_efficiency,
			COALESCE(uns.blocks_found, 0) as blocks_found,
			COALESCE(uns.total_earned, 0) as total_earned,
			COALESCE(uns.pending_balance, 0) as pending_balance,
			COALESCE(uns.total_paid_out, 0) as total_paid_out,
			COALESCE(uns.active_workers, 0) as active_workers,
			COALESCE(uns.total_workers, 0) as total_workers,
			uns.last_active_at,
			uns.first_connected_at
		FROM network_configs nc
		LEFT JOIN user_network_stats uns ON nc.id = uns.network_id AND uns.user_id = $1
		ORDER BY nc.is_active DESC, uns.last_active_at DESC NULLS LAST
	`

	rows, err := h.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []UserNetworkStats
	for rows.Next() {
		var s UserNetworkStats
		s.UserID = userID
		err := rows.Scan(
			&s.ID, &s.NetworkID, &s.NetworkName, &s.NetworkSymbol, &s.NetworkDisplay,
			&s.IsNetworkActive, &s.TotalHashrate, &s.AverageHashrate, &s.PeakHashrate,
			&s.TotalShares, &s.ValidShares, &s.InvalidShares, &s.ShareEfficiency,
			&s.BlocksFound, &s.TotalEarned, &s.PendingBalance, &s.TotalPaidOut,
			&s.ActiveWorkers, &s.TotalWorkers, &s.LastActiveAt, &s.FirstConnected,
		)
		if err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}

	return stats, nil
}

func (h *Handler) fetchUserAggregatedStats(userID int64) (*AggregatedUserStats, error) {
	query := `
		SELECT 
			COUNT(DISTINCT uns.network_id)::INTEGER as total_networks_mined,
			COUNT(DISTINCT CASE WHEN nc.is_active THEN uns.network_id END)::INTEGER as active_networks,
			COALESCE(SUM(uns.total_hashrate), 0) as combined_hashrate,
			COALESCE(SUM(uns.total_shares), 0) as total_shares_all,
			COALESCE(SUM(uns.blocks_found), 0) as total_blocks_all,
			COALESCE(SUM(uns.total_earned), 0) as total_earned_all,
			COALESCE(SUM(uns.pending_balance), 0) as total_pending_all,
			COALESCE(SUM(uns.active_workers), 0) as total_workers_all
		FROM user_network_stats uns
		JOIN network_configs nc ON nc.id = uns.network_id
		WHERE uns.user_id = $1
	`

	var stats AggregatedUserStats
	err := h.db.QueryRow(query, userID).Scan(
		&stats.TotalNetworksMined,
		&stats.ActiveNetworks,
		&stats.CombinedHashrate,
		&stats.TotalSharesAll,
		&stats.TotalBlocksAll,
		&stats.TotalEarnedAll,
		&stats.TotalPendingAll,
		&stats.TotalWorkersAll,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return &stats, nil
}

func (h *Handler) fetchAllNetworkPoolStats() ([]NetworkPoolStats, error) {
	query := `
		SELECT 
			nc.id as network_id,
			nc.name as network_name,
			nc.symbol as network_symbol,
			COALESCE(nps.total_hashrate, 0) as total_hashrate,
			COALESCE(nps.active_miners, 0) as active_miners,
			COALESCE(nps.active_workers, 0) as active_workers,
			COALESCE(nps.shares_per_second, 0) as shares_per_second,
			COALESCE(nps.blocks_found_24h, 0) as blocks_found_24h,
			COALESCE(nps.blocks_found_total, 0) as blocks_found_total,
			COALESCE(nps.network_difficulty, 0) as network_difficulty,
			COALESCE(nps.network_hashrate, 0) as network_hashrate,
			COALESCE(nps.current_block_height, 0) as current_block_height,
			COALESCE(nps.pool_percentage, 0) as pool_percentage,
			COALESCE(nps.rpc_connected, false) as rpc_connected,
			nps.rpc_last_check as last_block_at
		FROM network_configs nc
		LEFT JOIN network_pool_stats nps ON nc.id = nps.network_id
		WHERE nc.is_active = true OR nps.total_hashrate > 0
		ORDER BY nc.is_active DESC, nps.total_hashrate DESC
	`

	rows, err := h.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []NetworkPoolStats
	for rows.Next() {
		var s NetworkPoolStats
		err := rows.Scan(
			&s.NetworkID, &s.NetworkName, &s.NetworkSymbol,
			&s.TotalHashrate, &s.ActiveMiners, &s.ActiveWorkers,
			&s.SharesPerSecond, &s.BlocksFound24h, &s.BlocksFoundTotal,
			&s.NetworkDifficulty, &s.NetworkHashrate, &s.CurrentBlockHeight,
			&s.PoolPercentage, &s.RPCConnected, &s.LastBlockAt,
		)
		if err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}

	return stats, nil
}

// NetworkConfig represents a supported network configuration
type NetworkConfig struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	Symbol         string    `json:"symbol"`
	DisplayName    string    `json:"display_name"`
	IsActive       bool      `json:"is_active"`
	IsDefault      bool      `json:"is_default"`
	Algorithm      string    `json:"algorithm"`
	StratumPort    int       `json:"stratum_port"`
	ExplorerURL    string    `json:"explorer_url"`
	LogoURL        string    `json:"logo_url,omitempty"`
	MinPayout      float64   `json:"min_payout_threshold"`
	PoolFeePercent float64   `json:"pool_fee_percent"`
}

func (h *Handler) fetchSupportedNetworks() ([]NetworkConfig, error) {
	query := `
		SELECT 
			id, name, symbol, display_name,
			is_active, is_default, algorithm,
			stratum_port, COALESCE(explorer_url, ''),
			COALESCE(logo_url, ''),
			min_payout_threshold, pool_fee_percent
		FROM network_configs
		ORDER BY is_active DESC, is_default DESC, display_name
	`

	rows, err := h.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var networks []NetworkConfig
	for rows.Next() {
		var n NetworkConfig
		err := rows.Scan(
			&n.ID, &n.Name, &n.Symbol, &n.DisplayName,
			&n.IsActive, &n.IsDefault, &n.Algorithm,
			&n.StratumPort, &n.ExplorerURL, &n.LogoURL,
			&n.MinPayout, &n.PoolFeePercent,
		)
		if err != nil {
			return nil, err
		}
		networks = append(networks, n)
	}

	return networks, nil
}

// Helper function for JSON responses
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
