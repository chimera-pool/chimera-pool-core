package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// MOCK IMPLEMENTATIONS FOR TDD
// =============================================================================

type MockPoolStatsReader struct {
	GetStatsFunc func() (*PoolStatsData, error)
}

func (m *MockPoolStatsReader) GetStats() (*PoolStatsData, error) {
	if m.GetStatsFunc != nil {
		return m.GetStatsFunc()
	}
	return &PoolStatsData{
		TotalMiners:     150,
		TotalHashrate:   1000000,
		BlocksFound:     25,
		PoolFee:         1.0,
		MinimumPayout:   1.0,
		PaymentInterval: "1 hour",
		Network:         "BlockDAG Awakening",
		Currency:        "BDAG",
	}, nil
}

type MockBlockReader struct {
	GetRecentBlocksFunc  func(limit int) ([]*BlockData, error)
	GetBlockByHeightFunc func(height int64) (*BlockData, error)
	GetBlockByHashFunc   func(hash string) (*BlockData, error)
}

func (m *MockBlockReader) GetRecentBlocks(limit int) ([]*BlockData, error) {
	if m.GetRecentBlocksFunc != nil {
		return m.GetRecentBlocksFunc(limit)
	}
	return []*BlockData{
		{ID: 1, Height: 1000, Hash: "abc123", Reward: 625000000, Status: "confirmed", Timestamp: time.Now()},
		{ID: 2, Height: 999, Hash: "def456", Reward: 625000000, Status: "confirmed", Timestamp: time.Now().Add(-10 * time.Minute)},
	}, nil
}

func (m *MockBlockReader) GetBlockByHeight(height int64) (*BlockData, error) {
	if m.GetBlockByHeightFunc != nil {
		return m.GetBlockByHeightFunc(height)
	}
	return &BlockData{
		ID: 1, Height: height, Hash: "abc123", Reward: 625000000, Status: "confirmed", Timestamp: time.Now(),
	}, nil
}

func (m *MockBlockReader) GetBlockByHash(hash string) (*BlockData, error) {
	if m.GetBlockByHashFunc != nil {
		return m.GetBlockByHashFunc(hash)
	}
	return &BlockData{
		ID: 1, Height: 1000, Hash: hash, Reward: 625000000, Status: "confirmed", Timestamp: time.Now(),
	}, nil
}

type MockMinerLocationReader struct {
	GetPublicLocationsFunc func() ([]*MinerLocationData, error)
	GetLocationStatsFunc   func() (*LocationStatsData, error)
}

func (m *MockMinerLocationReader) GetPublicLocations() ([]*MinerLocationData, error) {
	if m.GetPublicLocationsFunc != nil {
		return m.GetPublicLocationsFunc()
	}
	return []*MinerLocationData{
		{Latitude: 40.7128, Longitude: -74.0060, Country: "US", Hashrate: 50000},
		{Latitude: 51.5074, Longitude: -0.1278, Country: "GB", Hashrate: 30000},
	}, nil
}

func (m *MockMinerLocationReader) GetLocationStats() (*LocationStatsData, error) {
	if m.GetLocationStatsFunc != nil {
		return m.GetLocationStatsFunc()
	}
	return &LocationStatsData{
		TotalCountries: 25,
		TotalRegions:   100,
		TopCountries: []CountryStats{
			{Country: "US", MinerCount: 50, Hashrate: 500000},
			{Country: "DE", MinerCount: 30, Hashrate: 300000},
		},
		ContinentBreakdown: []ContinentBreakdownStats{
			{Continent: "North America", MinerCount: 60, Hashrate: 600000},
			{Continent: "Europe", MinerCount: 50, Hashrate: 500000},
		},
	}, nil
}

// =============================================================================
// TEST HELPERS
// =============================================================================

func setupPoolRouter(handlers *PoolHandlers) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	api := router.Group("/api/v1")
	{
		api.GET("/pool/stats", handlers.GetPoolStats)
		api.GET("/pool/blocks", handlers.GetRecentBlocks)
		api.GET("/pool/blocks/height/:height", handlers.GetBlockByHeight)
		api.GET("/pool/blocks/hash/:hash", handlers.GetBlockByHash)
		api.GET("/miners/locations", handlers.GetPublicMinerLocations)
		api.GET("/miners/locations/stats", handlers.GetMinerLocationStats)
	}

	return router
}

func createDefaultPoolHandlers() *PoolHandlers {
	return NewPoolHandlers(
		&MockPoolStatsReader{},
		&MockBlockReader{},
		&MockMinerLocationReader{},
	)
}

// =============================================================================
// POOL STATS TESTS (TDD)
// =============================================================================

func TestPoolHandlers_GetPoolStats(t *testing.T) {
	t.Run("successful get pool stats", func(t *testing.T) {
		handlers := createDefaultPoolHandlers()
		router := setupPoolRouter(handlers)

		req, _ := http.NewRequest("GET", "/api/v1/pool/stats", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response PoolStatsData
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, int64(150), response.TotalMiners)
		assert.Equal(t, "BDAG", response.Currency)
	})

	t.Run("pool stats error", func(t *testing.T) {
		reader := &MockPoolStatsReader{
			GetStatsFunc: func() (*PoolStatsData, error) {
				return nil, errors.New("database error")
			},
		}
		handlers := NewPoolHandlers(reader, &MockBlockReader{}, &MockMinerLocationReader{})
		router := setupPoolRouter(handlers)

		req, _ := http.NewRequest("GET", "/api/v1/pool/stats", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// =============================================================================
// BLOCKS TESTS (TDD)
// =============================================================================

func TestPoolHandlers_GetRecentBlocks(t *testing.T) {
	t.Run("successful get recent blocks", func(t *testing.T) {
		handlers := createDefaultPoolHandlers()
		router := setupPoolRouter(handlers)

		req, _ := http.NewRequest("GET", "/api/v1/pool/blocks", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(2), response["count"])
	})

	t.Run("get blocks with custom limit", func(t *testing.T) {
		handlers := createDefaultPoolHandlers()
		router := setupPoolRouter(handlers)

		req, _ := http.NewRequest("GET", "/api/v1/pool/blocks?limit=10", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("blocks error", func(t *testing.T) {
		reader := &MockBlockReader{
			GetRecentBlocksFunc: func(limit int) ([]*BlockData, error) {
				return nil, errors.New("database error")
			},
		}
		handlers := NewPoolHandlers(&MockPoolStatsReader{}, reader, &MockMinerLocationReader{})
		router := setupPoolRouter(handlers)

		req, _ := http.NewRequest("GET", "/api/v1/pool/blocks", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestPoolHandlers_GetBlockByHeight(t *testing.T) {
	t.Run("successful get block by height", func(t *testing.T) {
		handlers := createDefaultPoolHandlers()
		router := setupPoolRouter(handlers)

		req, _ := http.NewRequest("GET", "/api/v1/pool/blocks/height/1000", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response BlockData
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, int64(1000), response.Height)
	})

	t.Run("block not found", func(t *testing.T) {
		reader := &MockBlockReader{
			GetBlockByHeightFunc: func(height int64) (*BlockData, error) {
				return nil, errors.New("not found")
			},
		}
		handlers := NewPoolHandlers(&MockPoolStatsReader{}, reader, &MockMinerLocationReader{})
		router := setupPoolRouter(handlers)

		req, _ := http.NewRequest("GET", "/api/v1/pool/blocks/height/99999", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestPoolHandlers_GetBlockByHash(t *testing.T) {
	t.Run("successful get block by hash", func(t *testing.T) {
		handlers := createDefaultPoolHandlers()
		router := setupPoolRouter(handlers)

		hash := "abc123def456789012345678901234567890"
		req, _ := http.NewRequest("GET", "/api/v1/pool/blocks/hash/"+hash, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid hash format", func(t *testing.T) {
		handlers := createDefaultPoolHandlers()
		router := setupPoolRouter(handlers)

		req, _ := http.NewRequest("GET", "/api/v1/pool/blocks/hash/short", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("block not found by hash", func(t *testing.T) {
		reader := &MockBlockReader{
			GetBlockByHashFunc: func(hash string) (*BlockData, error) {
				return nil, errors.New("not found")
			},
		}
		handlers := NewPoolHandlers(&MockPoolStatsReader{}, reader, &MockMinerLocationReader{})
		router := setupPoolRouter(handlers)

		hash := "abc123def456789012345678901234567890"
		req, _ := http.NewRequest("GET", "/api/v1/pool/blocks/hash/"+hash, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// =============================================================================
// MINER LOCATIONS TESTS (TDD)
// =============================================================================

func TestPoolHandlers_GetPublicMinerLocations(t *testing.T) {
	t.Run("successful get miner locations", func(t *testing.T) {
		handlers := createDefaultPoolHandlers()
		router := setupPoolRouter(handlers)

		req, _ := http.NewRequest("GET", "/api/v1/miners/locations", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(2), response["count"])
	})

	t.Run("locations error", func(t *testing.T) {
		reader := &MockMinerLocationReader{
			GetPublicLocationsFunc: func() ([]*MinerLocationData, error) {
				return nil, errors.New("database error")
			},
		}
		handlers := NewPoolHandlers(&MockPoolStatsReader{}, &MockBlockReader{}, reader)
		router := setupPoolRouter(handlers)

		req, _ := http.NewRequest("GET", "/api/v1/miners/locations", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestPoolHandlers_GetMinerLocationStats(t *testing.T) {
	t.Run("successful get location stats", func(t *testing.T) {
		handlers := createDefaultPoolHandlers()
		router := setupPoolRouter(handlers)

		req, _ := http.NewRequest("GET", "/api/v1/miners/locations/stats", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response LocationStatsData
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, 25, response.TotalCountries)
	})

	t.Run("location stats error", func(t *testing.T) {
		reader := &MockMinerLocationReader{
			GetLocationStatsFunc: func() (*LocationStatsData, error) {
				return nil, errors.New("database error")
			},
		}
		handlers := NewPoolHandlers(&MockPoolStatsReader{}, &MockBlockReader{}, reader)
		router := setupPoolRouter(handlers)

		req, _ := http.NewRequest("GET", "/api/v1/miners/locations/stats", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// =============================================================================
// ISP INTERFACE COMPLIANCE TESTS
// =============================================================================

func TestISP_PoolHandlers_InterfaceSegregation(t *testing.T) {
	t.Run("PoolStatsReader has single responsibility", func(t *testing.T) {
		var reader PoolStatsReader = &MockPoolStatsReader{}
		_, _ = reader.GetStats()
	})

	t.Run("BlockReader has single responsibility", func(t *testing.T) {
		var reader BlockReader = &MockBlockReader{}
		_, _ = reader.GetRecentBlocks(10)
		_, _ = reader.GetBlockByHeight(1000)
		_, _ = reader.GetBlockByHash("abc123")
	})

	t.Run("MinerLocationReader has single responsibility", func(t *testing.T) {
		var reader MinerLocationReader = &MockMinerLocationReader{}
		_, _ = reader.GetPublicLocations()
		_, _ = reader.GetLocationStats()
	})

	t.Run("handlers can work with minimal dependencies", func(t *testing.T) {
		handlers := NewPoolHandlers(
			&MockPoolStatsReader{},
			&MockBlockReader{},
			&MockMinerLocationReader{},
		)
		assert.NotNil(t, handlers)
	})
}
