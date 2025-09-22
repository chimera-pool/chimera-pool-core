#!/bin/bash

# Extend Existing Components for Multi-Coin Support
# This script extends existing production-ready components for universal cryptocurrency support

set -e

echo "🔄 Extending Existing Components for Multi-Coin Support..."
echo "========================================================"

# Source common functions
source "$(dirname "$0")/common.sh"

PROJECT_ROOT=$(get_project_root)

log_info "Extending database schema for multi-coin support..."

# Create multi-coin database migration
cat > "${PROJECT_ROOT}/migrations/003_multicoin_support.up.sql" << 'EOF'
-- Multi-coin support extension
-- Extends existing schema to support multiple cryptocurrencies

-- Add supported cryptocurrencies table
CREATE TABLE supported_cryptocurrencies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol VARCHAR(10) NOT NULL UNIQUE, -- BTC, ETC, BDAG, LTC, etc.
    name VARCHAR(100) NOT NULL,
    algorithm VARCHAR(50) NOT NULL,
    block_time INTEGER NOT NULL DEFAULT 30,
    block_reward DECIMAL(20,8) NOT NULL,
    minimum_payout DECIMAL(20,8) NOT NULL DEFAULT 0.001,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW()
);

-- Insert default supported cryptocurrencies
INSERT INTO supported_cryptocurrencies (symbol, name, algorithm, block_time, block_reward, minimum_payout) VALUES
('BTC', 'Bitcoin', 'sha256', 600, 6.25, 0.001),
('ETC', 'Ethereum Classic', 'ethash', 13, 3.2, 0.1),
('BDAG', 'BlockDAG', 'blake3', 1, 1000, 1.0),
('LTC', 'Litecoin', 'scrypt', 150, 6.25, 0.01),
('DASH', 'Dash', 'x11', 150, 2.5, 0.01),
('XMR', 'Monero', 'randomx', 120, 0.6, 0.1),
('ZEC', 'Zcash', 'equihash', 75, 3.125, 0.01);

-- Add cryptocurrency_id to existing tables
ALTER TABLE miners ADD COLUMN cryptocurrency_id UUID REFERENCES supported_cryptocurrencies(id);
ALTER TABLE shares ADD COLUMN cryptocurrency_id UUID REFERENCES supported_cryptocurrencies(id);
ALTER TABLE blocks ADD COLUMN cryptocurrency_id UUID REFERENCES supported_cryptocurrencies(id);
ALTER TABLE payouts ADD COLUMN cryptocurrency_id UUID REFERENCES supported_cryptocurrencies(id);

-- Add pool configuration table for multi-coin pools
CREATE TABLE pool_configurations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cryptocurrency_id UUID NOT NULL REFERENCES supported_cryptocurrencies(id),
    stratum_port INTEGER NOT NULL UNIQUE,
    difficulty BIGINT NOT NULL DEFAULT 1,
    fee_percentage DECIMAL(5,4) NOT NULL DEFAULT 1.0000,
    payout_threshold DECIMAL(20,8),
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW()
);

-- Insert default pool configurations
INSERT INTO pool_configurations (cryptocurrency_id, stratum_port, difficulty, fee_percentage, payout_threshold) 
SELECT id, 
    CASE symbol 
        WHEN 'BTC' THEN 3333
        WHEN 'ETC' THEN 3334  
        WHEN 'BDAG' THEN 3335
        WHEN 'LTC' THEN 3336
        WHEN 'DASH' THEN 3337
        WHEN 'XMR' THEN 3338
        WHEN 'ZEC' THEN 3339
    END,
    CASE symbol
        WHEN 'BTC' THEN 1000000
        WHEN 'ETC' THEN 100000
        WHEN 'BDAG' THEN 1000
        ELSE 10000
    END,
    1.0000,
    minimum_payout
FROM supported_cryptocurrencies;

-- Create indexes for performance
CREATE INDEX idx_miners_cryptocurrency_id ON miners(cryptocurrency_id);
CREATE INDEX idx_shares_cryptocurrency_id ON shares(cryptocurrency_id);
CREATE INDEX idx_blocks_cryptocurrency_id ON blocks(cryptocurrency_id);
CREATE INDEX idx_payouts_cryptocurrency_id ON payouts(cryptocurrency_id);
CREATE INDEX idx_pool_configurations_cryptocurrency_id ON pool_configurations(cryptocurrency_id);
EOF

# Create down migration
cat > "${PROJECT_ROOT}/migrations/003_multicoin_support.down.sql" << 'EOF'
-- Rollback multi-coin support

DROP INDEX IF EXISTS idx_payouts_cryptocurrency_id;
DROP INDEX IF EXISTS idx_blocks_cryptocurrency_id;
DROP INDEX IF EXISTS idx_shares_cryptocurrency_id;
DROP INDEX IF EXISTS idx_miners_cryptocurrency_id;
DROP INDEX IF EXISTS idx_pool_configurations_cryptocurrency_id;

DROP TABLE IF EXISTS pool_configurations;

ALTER TABLE payouts DROP COLUMN IF EXISTS cryptocurrency_id;
ALTER TABLE blocks DROP COLUMN IF EXISTS cryptocurrency_id;
ALTER TABLE shares DROP COLUMN IF EXISTS cryptocurrency_id;
ALTER TABLE miners DROP COLUMN IF EXISTS cryptocurrency_id;

DROP TABLE IF EXISTS supported_cryptocurrencies;
EOF

log_success "Database migration created for multi-coin support"

# Extend Go models for multi-coin support
log_info "Extending Go models for multi-coin support..."

cat > "${PROJECT_ROOT}/internal/database/multicoin_models.go" << 'EOF'
package database

import (
	"time"
	"github.com/google/uuid"
)

// SupportedCryptocurrency represents a cryptocurrency supported by the pool
type SupportedCryptocurrency struct {
	ID            uuid.UUID `json:"id" db:"id"`
	Symbol        string    `json:"symbol" db:"symbol"`
	Name          string    `json:"name" db:"name"`
	Algorithm     string    `json:"algorithm" db:"algorithm"`
	BlockTime     int       `json:"block_time" db:"block_time"`
	BlockReward   float64   `json:"block_reward" db:"block_reward"`
	MinimumPayout float64   `json:"minimum_payout" db:"minimum_payout"`
	Status        string    `json:"status" db:"status"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// PoolConfiguration represents pool configuration for a specific cryptocurrency
type PoolConfiguration struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	CryptocurrencyID uuid.UUID  `json:"cryptocurrency_id" db:"cryptocurrency_id"`
	StratumPort      int        `json:"stratum_port" db:"stratum_port"`
	Difficulty       int64      `json:"difficulty" db:"difficulty"`
	FeePercentage    float64    `json:"fee_percentage" db:"fee_percentage"`
	PayoutThreshold  *float64   `json:"payout_threshold" db:"payout_threshold"`
	Status           string     `json:"status" db:"status"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	
	// Joined fields
	Cryptocurrency *SupportedCryptocurrency `json:"cryptocurrency,omitempty"`
}

// MultiCoinMiner extends the existing Miner model with cryptocurrency support
type MultiCoinMiner struct {
	Miner
	CryptocurrencyID *uuid.UUID            `json:"cryptocurrency_id" db:"cryptocurrency_id"`
	Cryptocurrency   *SupportedCryptocurrency `json:"cryptocurrency,omitempty"`
}

// MultiCoinShare extends the existing Share model with cryptocurrency support
type MultiCoinShare struct {
	Share
	CryptocurrencyID *uuid.UUID            `json:"cryptocurrency_id" db:"cryptocurrency_id"`
	Cryptocurrency   *SupportedCryptocurrency `json:"cryptocurrency,omitempty"`
}
EOF

# Extend API handlers for multi-coin support
log_info "Extending API handlers for multi-coin support..."

cat > "${PROJECT_ROOT}/internal/api/multicoin_handlers.go" << 'EOF'
package api

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"chimera-pool-core/internal/database"
)

// MultiCoinHandlers provides HTTP handlers for multi-coin functionality
type MultiCoinHandlers struct {
	db *database.Database
}

// NewMultiCoinHandlers creates a new MultiCoinHandlers instance
func NewMultiCoinHandlers(db *database.Database) *MultiCoinHandlers {
	return &MultiCoinHandlers{db: db}
}

// GetSupportedCryptocurrencies returns all supported cryptocurrencies
func (h *MultiCoinHandlers) GetSupportedCryptocurrencies(c *gin.Context) {
	cryptos, err := h.db.GetSupportedCryptocurrencies(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"cryptocurrencies": cryptos})
}

// GetPoolConfigurations returns all pool configurations
func (h *MultiCoinHandlers) GetPoolConfigurations(c *gin.Context) {
	configs, err := h.db.GetPoolConfigurations(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"pool_configurations": configs})
}

// GetPoolConfigurationByCoin returns pool configuration for specific cryptocurrency
func (h *MultiCoinHandlers) GetPoolConfigurationByCoin(c *gin.Context) {
	symbol := c.Param("symbol")
	
	config, err := h.db.GetPoolConfigurationBySymbol(c.Request.Context(), symbol)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pool configuration not found"})
		return
	}
	
	c.JSON(http.StatusOK, config)
}

// GetMultiCoinStats returns statistics across all supported cryptocurrencies
func (h *MultiCoinHandlers) GetMultiCoinStats(c *gin.Context) {
	stats, err := h.db.GetMultiCoinStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, stats)
}

// RegisterMultiCoinRoutes registers all multi-coin API routes
func RegisterMultiCoinRoutes(r *gin.Engine, handlers *MultiCoinHandlers) {
	api := r.Group("/api/v1")
	{
		api.GET("/cryptocurrencies", handlers.GetSupportedCryptocurrencies)
		api.GET("/pools", handlers.GetPoolConfigurations)
		api.GET("/pools/:symbol", handlers.GetPoolConfigurationByCoin)
		api.GET("/stats/multicoin", handlers.GetMultiCoinStats)
	}
}
EOF

# Extend database operations for multi-coin support
log_info "Extending database operations for multi-coin support..."

cat > "${PROJECT_ROOT}/internal/database/multicoin_operations.go" << 'EOF'
package database

import (
	"context"
	"database/sql"
)

// GetSupportedCryptocurrencies retrieves all supported cryptocurrencies
func (db *Database) GetSupportedCryptocurrencies(ctx context.Context) ([]SupportedCryptocurrency, error) {
	query := `
		SELECT id, symbol, name, algorithm, block_time, block_reward, minimum_payout, status, created_at
		FROM supported_cryptocurrencies
		WHERE status = 'active'
		ORDER BY symbol
	`
	
	rows, err := db.pool.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var cryptocurrencies []SupportedCryptocurrency
	for rows.Next() {
		var crypto SupportedCryptocurrency
		err := rows.Scan(
			&crypto.ID, &crypto.Symbol, &crypto.Name, &crypto.Algorithm,
			&crypto.BlockTime, &crypto.BlockReward, &crypto.MinimumPayout,
			&crypto.Status, &crypto.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		cryptocurrencies = append(cryptocurrencies, crypto)
	}
	
	return cryptocurrencies, nil
}

// GetPoolConfigurations retrieves all pool configurations with cryptocurrency details
func (db *Database) GetPoolConfigurations(ctx context.Context) ([]PoolConfiguration, error) {
	query := `
		SELECT 
			pc.id, pc.cryptocurrency_id, pc.stratum_port, pc.difficulty,
			pc.fee_percentage, pc.payout_threshold, pc.status, pc.created_at,
			sc.symbol, sc.name, sc.algorithm
		FROM pool_configurations pc
		JOIN supported_cryptocurrencies sc ON pc.cryptocurrency_id = sc.id
		WHERE pc.status = 'active'
		ORDER BY pc.stratum_port
	`
	
	rows, err := db.pool.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var configs []PoolConfiguration
	for rows.Next() {
		var config PoolConfiguration
		var crypto SupportedCryptocurrency
		
		err := rows.Scan(
			&config.ID, &config.CryptocurrencyID, &config.StratumPort,
			&config.Difficulty, &config.FeePercentage, &config.PayoutThreshold,
			&config.Status, &config.CreatedAt,
			&crypto.Symbol, &crypto.Name, &crypto.Algorithm,
		)
		if err != nil {
			return nil, err
		}
		
		config.Cryptocurrency = &crypto
		configs = append(configs, config)
	}
	
	return configs, nil
}

// GetPoolConfigurationBySymbol retrieves pool configuration for specific cryptocurrency
func (db *Database) GetPoolConfigurationBySymbol(ctx context.Context, symbol string) (*PoolConfiguration, error) {
	query := `
		SELECT 
			pc.id, pc.cryptocurrency_id, pc.stratum_port, pc.difficulty,
			pc.fee_percentage, pc.payout_threshold, pc.status, pc.created_at,
			sc.symbol, sc.name, sc.algorithm, sc.block_time, sc.block_reward
		FROM pool_configurations pc
		JOIN supported_cryptocurrencies sc ON pc.cryptocurrency_id = sc.id
		WHERE sc.symbol = $1 AND pc.status = 'active'
	`
	
	var config PoolConfiguration
	var crypto SupportedCryptocurrency
	
	err := db.pool.QueryRowContext(ctx, query, symbol).Scan(
		&config.ID, &config.CryptocurrencyID, &config.StratumPort,
		&config.Difficulty, &config.FeePercentage, &config.PayoutThreshold,
		&config.Status, &config.CreatedAt,
		&crypto.Symbol, &crypto.Name, &crypto.Algorithm,
		&crypto.BlockTime, &crypto.BlockReward,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	
	config.Cryptocurrency = &crypto
	return &config, nil
}

// GetMultiCoinStats retrieves statistics across all cryptocurrencies
func (db *Database) GetMultiCoinStats(ctx context.Context) (map[string]interface{}, error) {
	query := `
		SELECT 
			sc.symbol,
			COUNT(DISTINCT m.id) as active_miners,
			COALESCE(SUM(m.hashrate), 0) as total_hashrate,
			COUNT(DISTINCT CASE WHEN s.is_valid THEN s.id END) as valid_shares,
			COUNT(DISTINCT b.id) as blocks_found
		FROM supported_cryptocurrencies sc
		LEFT JOIN miners m ON sc.id = m.cryptocurrency_id AND m.is_active = true
		LEFT JOIN shares s ON sc.id = s.cryptocurrency_id AND s.timestamp > NOW() - INTERVAL '24 hours'
		LEFT JOIN blocks b ON sc.id = b.cryptocurrency_id AND b.timestamp > NOW() - INTERVAL '24 hours'
		WHERE sc.status = 'active'
		GROUP BY sc.symbol, sc.name
		ORDER BY sc.symbol
	`
	
	rows, err := db.pool.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	stats := make(map[string]interface{})
	coinStats := make([]map[string]interface{}, 0)
	
	totalMiners := 0
	totalHashrate := float64(0)
	totalShares := 0
	totalBlocks := 0
	
	for rows.Next() {
		var symbol string
		var miners, shares, blocks int
		var hashrate float64
		
		err := rows.Scan(&symbol, &miners, &hashrate, &shares, &blocks)
		if err != nil {
			return nil, err
		}
		
		coinStat := map[string]interface{}{
			"symbol":        symbol,
			"active_miners": miners,
			"total_hashrate": hashrate,
			"valid_shares":  shares,
			"blocks_found":  blocks,
		}
		
		coinStats = append(coinStats, coinStat)
		
		totalMiners += miners
		totalHashrate += hashrate
		totalShares += shares
		totalBlocks += blocks
	}
	
	stats["coins"] = coinStats
	stats["totals"] = map[string]interface{}{
		"total_miners":   totalMiners,
		"total_hashrate": totalHashrate,
		"total_shares":   totalShares,
		"total_blocks":   totalBlocks,
	}
	
	return stats, nil
}
EOF

log_success "Multi-coin database operations created"

# Create test for multi-coin functionality
log_info "Creating tests for multi-coin functionality..."

cat > "${PROJECT_ROOT}/internal/database/multicoin_test.go" << 'EOF'
package database

import (
	"context"
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiCoinFunctionality(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	
	ctx := context.Background()
	
	t.Run("GetSupportedCryptocurrencies", func(t *testing.T) {
		cryptos, err := db.GetSupportedCryptocurrencies(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, cryptos)
		
		// Should have default cryptocurrencies
		symbols := make(map[string]bool)
		for _, crypto := range cryptos {
			symbols[crypto.Symbol] = true
		}
		
		assert.True(t, symbols["BTC"])
		assert.True(t, symbols["ETC"])
		assert.True(t, symbols["BDAG"])
	})
	
	t.Run("GetPoolConfigurations", func(t *testing.T) {
		configs, err := db.GetPoolConfigurations(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, configs)
		
		// Should have configurations for all cryptocurrencies
		ports := make(map[int]bool)
		for _, config := range configs {
			ports[config.StratumPort] = true
			assert.NotNil(t, config.Cryptocurrency)
		}
		
		assert.True(t, ports[3333]) // Bitcoin
		assert.True(t, ports[3334]) // Ethereum Classic
		assert.True(t, ports[3335]) // BlockDAG
	})
	
	t.Run("GetPoolConfigurationBySymbol", func(t *testing.T) {
		config, err := db.GetPoolConfigurationBySymbol(ctx, "BTC")
		require.NoError(t, err)
		require.NotNil(t, config)
		
		assert.Equal(t, 3333, config.StratumPort)
		assert.NotNil(t, config.Cryptocurrency)
		assert.Equal(t, "Bitcoin", config.Cryptocurrency.Name)
		assert.Equal(t, "sha256", config.Cryptocurrency.Algorithm)
	})
	
	t.Run("GetMultiCoinStats", func(t *testing.T) {
		stats, err := db.GetMultiCoinStats(ctx)
		require.NoError(t, err)
		assert.NotNil(t, stats)
		
		coins, ok := stats["coins"].([]map[string]interface{})
		assert.True(t, ok)
		assert.NotEmpty(t, coins)
		
		totals, ok := stats["totals"].(map[string]interface{})
		assert.True(t, ok)
		assert.Contains(t, totals, "total_miners")
	})
}
EOF

log_success "Multi-coin tests created"

# Update main API to include multi-coin routes
log_info "Updating main API to include multi-coin routes..."

if [ -f "${PROJECT_ROOT}/internal/api/handlers.go" ]; then
    # Add multi-coin import and registration to existing handlers
    cat >> "${PROJECT_ROOT}/internal/api/handlers.go" << 'EOF'

// RegisterMultiCoinAPI registers multi-coin API routes
func RegisterMultiCoinAPI(r *gin.Engine, db *database.Database) {
	multiCoinHandlers := NewMultiCoinHandlers(db)
	RegisterMultiCoinRoutes(r, multiCoinHandlers)
}
EOF
fi

log_success "Multi-coin extensions completed successfully!"

echo ""
echo "📋 Summary of Multi-Coin Extensions:"
echo "✅ Database migration created (003_multicoin_support)"
echo "✅ Multi-coin models added"
echo "✅ Multi-coin API handlers created"
echo "✅ Multi-coin database operations implemented"
echo "✅ Comprehensive tests added"
echo ""
echo "🚀 Next Steps:"
echo "1. Run database migration: make migrate-up"
echo "2. Run tests: go test ./internal/database/multicoin_test.go"
echo "3. Start server to test multi-coin API endpoints"
echo ""
echo "🎯 Multi-coin support is now ready for integration!"

