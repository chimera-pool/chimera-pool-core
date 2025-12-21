package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// =============================================================================
// ISP-COMPLIANT POOL SERVICE INTERFACES
// Each interface is small and focused on a single responsibility
// =============================================================================

// PoolStatsReader reads pool statistics (single responsibility)
type PoolStatsReader interface {
	GetStats() (*PoolStatsData, error)
}

// BlockReader reads block data (single responsibility)
type BlockReader interface {
	GetRecentBlocks(limit int) ([]*BlockData, error)
	GetBlockByHeight(height int64) (*BlockData, error)
	GetBlockByHash(hash string) (*BlockData, error)
}

// MinerLocationReader reads miner location data (single responsibility)
type MinerLocationReader interface {
	GetPublicLocations() ([]*MinerLocationData, error)
	GetLocationStats() (*LocationStatsData, error)
}

// =============================================================================
// POOL DATA MODELS
// =============================================================================

// PoolStatsData represents pool statistics
type PoolStatsData struct {
	TotalMiners     int64   `json:"total_miners"`
	TotalHashrate   float64 `json:"total_hashrate"`
	BlocksFound     int64   `json:"blocks_found"`
	PoolFee         float64 `json:"pool_fee"`
	MinimumPayout   float64 `json:"minimum_payout"`
	PaymentInterval string  `json:"payment_interval"`
	Network         string  `json:"network"`
	Currency        string  `json:"currency"`
}

// BlockData represents a block
type BlockData struct {
	ID        int64     `json:"id"`
	Height    int64     `json:"height"`
	Hash      string    `json:"hash"`
	Reward    int64     `json:"reward"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	FinderID  int64     `json:"finder_id,omitempty"`
}

// MinerLocationData represents a miner's location (anonymized)
type MinerLocationData struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Country   string  `json:"country"`
	Region    string  `json:"region,omitempty"`
	Hashrate  float64 `json:"hashrate"`
}

// LocationStatsData represents aggregated location statistics
type LocationStatsData struct {
	TotalCountries     int                       `json:"total_countries"`
	TotalRegions       int                       `json:"total_regions"`
	TopCountries       []CountryStats            `json:"top_countries"`
	ContinentBreakdown []ContinentBreakdownStats `json:"continent_breakdown"`
}

// CountryStats represents stats for a country
type CountryStats struct {
	Country    string  `json:"country"`
	MinerCount int64   `json:"miner_count"`
	Hashrate   float64 `json:"hashrate"`
}

// ContinentBreakdownStats represents stats for a continent
type ContinentBreakdownStats struct {
	Continent  string  `json:"continent"`
	MinerCount int64   `json:"miner_count"`
	Hashrate   float64 `json:"hashrate"`
}

// =============================================================================
// POOL HANDLERS - ISP-COMPLIANT IMPLEMENTATION
// =============================================================================

// PoolHandlers provides HTTP handlers for pool-related endpoints
type PoolHandlers struct {
	statsReader    PoolStatsReader
	blockReader    BlockReader
	locationReader MinerLocationReader
}

// NewPoolHandlers creates new pool handlers with injected dependencies
func NewPoolHandlers(
	statsReader PoolStatsReader,
	blockReader BlockReader,
	locationReader MinerLocationReader,
) *PoolHandlers {
	return &PoolHandlers{
		statsReader:    statsReader,
		blockReader:    blockReader,
		locationReader: locationReader,
	}
}

// GetPoolStats returns pool statistics
// GET /api/v1/pool/stats
func (h *PoolHandlers) GetPoolStats(c *gin.Context) {
	stats, err := h.statsReader.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get pool statistics: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetRecentBlocks returns recent blocks
// GET /api/v1/pool/blocks
func (h *PoolHandlers) GetRecentBlocks(c *gin.Context) {
	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := parseInt(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	blocks, err := h.blockReader.GetRecentBlocks(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get blocks: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"blocks": blocks,
		"count":  len(blocks),
	})
}

// GetBlockByHeight returns a block by height
// GET /api/v1/pool/blocks/height/:height
func (h *PoolHandlers) GetBlockByHeight(c *gin.Context) {
	heightStr := c.Param("height")
	height, err := parseInt64(heightStr)
	if err != nil || height < 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_parameter",
			Message: "Invalid block height",
			Code:    http.StatusBadRequest,
		})
		return
	}

	block, err := h.blockReader.GetBlockByHeight(height)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Block not found",
			Code:    http.StatusNotFound,
		})
		return
	}

	c.JSON(http.StatusOK, block)
}

// GetBlockByHash returns a block by hash
// GET /api/v1/pool/blocks/hash/:hash
func (h *PoolHandlers) GetBlockByHash(c *gin.Context) {
	hash := c.Param("hash")
	if hash == "" || len(hash) < 32 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_parameter",
			Message: "Invalid block hash",
			Code:    http.StatusBadRequest,
		})
		return
	}

	block, err := h.blockReader.GetBlockByHash(hash)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Block not found",
			Code:    http.StatusNotFound,
		})
		return
	}

	c.JSON(http.StatusOK, block)
}

// GetPublicMinerLocations returns anonymized miner locations
// GET /api/v1/miners/locations
func (h *PoolHandlers) GetPublicMinerLocations(c *gin.Context) {
	locations, err := h.locationReader.GetPublicLocations()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get miner locations: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"locations": locations,
		"count":     len(locations),
	})
}

// GetMinerLocationStats returns aggregated location statistics
// GET /api/v1/miners/locations/stats
func (h *PoolHandlers) GetMinerLocationStats(c *gin.Context) {
	stats, err := h.locationReader.GetLocationStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get location statistics: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// Helper function to parse int64
func parseInt64(s string) (int64, error) {
	var result int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		result = result*10 + int64(c-'0')
	}
	return result, nil
}
