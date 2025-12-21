package api

import (
	"bytes"
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

type MockUserProfileReader struct {
	GetProfileFunc func(userID int64) (*UserProfileData, error)
}

func (m *MockUserProfileReader) GetProfile(userID int64) (*UserProfileData, error) {
	if m.GetProfileFunc != nil {
		return m.GetProfileFunc(userID)
	}
	return &UserProfileData{
		ID:            userID,
		Username:      "testuser",
		Email:         "test@example.com",
		PayoutAddress: "bdag1abc123",
		IsAdmin:       false,
		CreatedAt:     time.Now(),
	}, nil
}

type MockUserProfileWriter struct {
	UpdateProfileFunc func(userID int64, data *UpdateProfileData) (*UserProfileData, error)
}

func (m *MockUserProfileWriter) UpdateProfile(userID int64, data *UpdateProfileData) (*UserProfileData, error) {
	if m.UpdateProfileFunc != nil {
		return m.UpdateProfileFunc(userID, data)
	}
	return &UserProfileData{
		ID:            userID,
		Username:      data.Username,
		Email:         "test@example.com",
		PayoutAddress: data.PayoutAddress,
		IsAdmin:       false,
		CreatedAt:     time.Now(),
	}, nil
}

type MockUserPasswordChanger struct {
	ChangePasswordFunc func(userID int64, currentPassword, newPassword string) error
}

func (m *MockUserPasswordChanger) ChangePassword(userID int64, currentPassword, newPassword string) error {
	if m.ChangePasswordFunc != nil {
		return m.ChangePasswordFunc(userID, currentPassword, newPassword)
	}
	return nil
}

type MockUserMinerReader struct {
	GetMinersFunc func(userID int64) ([]*MinerData, error)
}

func (m *MockUserMinerReader) GetMiners(userID int64) ([]*MinerData, error) {
	if m.GetMinersFunc != nil {
		return m.GetMinersFunc(userID)
	}
	return []*MinerData{
		{ID: 1, Name: "miner-1", Hashrate: 50000, LastSeen: time.Now(), IsActive: true, ShareCount: 100},
		{ID: 2, Name: "miner-2", Hashrate: 75000, LastSeen: time.Now(), IsActive: true, ShareCount: 150},
	}, nil
}

type MockUserPayoutReader struct {
	GetPayoutsFunc func(userID int64, limit, offset int) ([]*PayoutData, error)
}

func (m *MockUserPayoutReader) GetPayouts(userID int64, limit, offset int) ([]*PayoutData, error) {
	if m.GetPayoutsFunc != nil {
		return m.GetPayoutsFunc(userID, limit, offset)
	}
	return []*PayoutData{
		{ID: 1, Amount: 1.5, TxHash: "tx123", Status: "confirmed", CreatedAt: time.Now()},
	}, nil
}

type MockUserStatsReader struct {
	GetHashrateHistoryFunc func(userID int64, period string) ([]*HashratePoint, error)
	GetSharesHistoryFunc   func(userID int64, period string) ([]*SharesPoint, error)
	GetEarningsHistoryFunc func(userID int64, period string) ([]*EarningsPoint, error)
}

func (m *MockUserStatsReader) GetHashrateHistory(userID int64, period string) ([]*HashratePoint, error) {
	if m.GetHashrateHistoryFunc != nil {
		return m.GetHashrateHistoryFunc(userID, period)
	}
	return []*HashratePoint{
		{Timestamp: time.Now().Add(-1 * time.Hour), Hashrate: 100000},
		{Timestamp: time.Now(), Hashrate: 120000},
	}, nil
}

func (m *MockUserStatsReader) GetSharesHistory(userID int64, period string) ([]*SharesPoint, error) {
	if m.GetSharesHistoryFunc != nil {
		return m.GetSharesHistoryFunc(userID, period)
	}
	return []*SharesPoint{
		{Timestamp: time.Now(), ValidShares: 100, InvalidShares: 2},
	}, nil
}

func (m *MockUserStatsReader) GetEarningsHistory(userID int64, period string) ([]*EarningsPoint, error) {
	if m.GetEarningsHistoryFunc != nil {
		return m.GetEarningsHistoryFunc(userID, period)
	}
	return []*EarningsPoint{
		{Timestamp: time.Now(), Earnings: 0.5},
	}, nil
}

// =============================================================================
// TEST HELPERS
// =============================================================================

func setupUserRouter(handlers *UserHandlers) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	api := router.Group("/api/v1/user")
	api.Use(func(c *gin.Context) {
		c.Set("user_id", int64(1))
		c.Next()
	})
	{
		api.GET("/profile", handlers.GetProfile)
		api.PUT("/profile", handlers.UpdateProfile)
		api.PUT("/password", handlers.ChangePassword)
		api.GET("/miners", handlers.GetMiners)
		api.GET("/payouts", handlers.GetPayouts)
		api.GET("/stats/hashrate", handlers.GetHashrateHistory)
		api.GET("/stats/shares", handlers.GetSharesHistory)
		api.GET("/stats/earnings", handlers.GetEarningsHistory)
	}

	return router
}

func createDefaultUserHandlers() *UserHandlers {
	return NewUserHandlers(
		&MockUserProfileReader{},
		&MockUserProfileWriter{},
		&MockUserPasswordChanger{},
		&MockUserMinerReader{},
		&MockUserPayoutReader{},
		&MockUserStatsReader{},
	)
}

// =============================================================================
// PROFILE TESTS (TDD)
// =============================================================================

func TestUserHandlers_GetProfile(t *testing.T) {
	t.Run("successful get profile", func(t *testing.T) {
		handlers := createDefaultUserHandlers()
		router := setupUserRouter(handlers)

		req, _ := http.NewRequest("GET", "/api/v1/user/profile", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response UserProfileData
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "testuser", response.Username)
	})

	t.Run("profile not found", func(t *testing.T) {
		reader := &MockUserProfileReader{
			GetProfileFunc: func(userID int64) (*UserProfileData, error) {
				return nil, errors.New("user not found")
			},
		}
		handlers := NewUserHandlers(reader, &MockUserProfileWriter{}, &MockUserPasswordChanger{}, &MockUserMinerReader{}, &MockUserPayoutReader{}, &MockUserStatsReader{})
		router := setupUserRouter(handlers)

		req, _ := http.NewRequest("GET", "/api/v1/user/profile", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestUserHandlers_UpdateProfile(t *testing.T) {
	t.Run("successful update profile", func(t *testing.T) {
		handlers := createDefaultUserHandlers()
		router := setupUserRouter(handlers)

		body := UpdateProfileData{
			Username:      "newusername",
			PayoutAddress: "bdag1newaddr",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("PUT", "/api/v1/user/profile", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("username already taken", func(t *testing.T) {
		writer := &MockUserProfileWriter{
			UpdateProfileFunc: func(userID int64, data *UpdateProfileData) (*UserProfileData, error) {
				return nil, errors.New("username already taken")
			},
		}
		handlers := NewUserHandlers(&MockUserProfileReader{}, writer, &MockUserPasswordChanger{}, &MockUserMinerReader{}, &MockUserPayoutReader{}, &MockUserStatsReader{})
		router := setupUserRouter(handlers)

		body := UpdateProfileData{Username: "taken"}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("PUT", "/api/v1/user/profile", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
	})
}

// =============================================================================
// PASSWORD TESTS (TDD)
// =============================================================================

func TestUserHandlers_ChangePassword(t *testing.T) {
	t.Run("successful password change", func(t *testing.T) {
		handlers := createDefaultUserHandlers()
		router := setupUserRouter(handlers)

		body := ChangePasswordRequest{
			CurrentPassword: "oldpassword",
			NewPassword:     "newpassword123",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("PUT", "/api/v1/user/password", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("incorrect current password", func(t *testing.T) {
		changer := &MockUserPasswordChanger{
			ChangePasswordFunc: func(userID int64, currentPassword, newPassword string) error {
				return errors.New("incorrect current password")
			},
		}
		handlers := NewUserHandlers(&MockUserProfileReader{}, &MockUserProfileWriter{}, changer, &MockUserMinerReader{}, &MockUserPayoutReader{}, &MockUserStatsReader{})
		router := setupUserRouter(handlers)

		body := ChangePasswordRequest{
			CurrentPassword: "wrong",
			NewPassword:     "newpassword123",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("PUT", "/api/v1/user/password", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// =============================================================================
// MINERS TESTS (TDD)
// =============================================================================

func TestUserHandlers_GetMiners(t *testing.T) {
	t.Run("successful get miners", func(t *testing.T) {
		handlers := createDefaultUserHandlers()
		router := setupUserRouter(handlers)

		req, _ := http.NewRequest("GET", "/api/v1/user/miners", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(2), response["total"])
	})

	t.Run("error getting miners", func(t *testing.T) {
		reader := &MockUserMinerReader{
			GetMinersFunc: func(userID int64) ([]*MinerData, error) {
				return nil, errors.New("database error")
			},
		}
		handlers := NewUserHandlers(&MockUserProfileReader{}, &MockUserProfileWriter{}, &MockUserPasswordChanger{}, reader, &MockUserPayoutReader{}, &MockUserStatsReader{})
		router := setupUserRouter(handlers)

		req, _ := http.NewRequest("GET", "/api/v1/user/miners", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// =============================================================================
// PAYOUTS TESTS (TDD)
// =============================================================================

func TestUserHandlers_GetPayouts(t *testing.T) {
	t.Run("successful get payouts", func(t *testing.T) {
		handlers := createDefaultUserHandlers()
		router := setupUserRouter(handlers)

		req, _ := http.NewRequest("GET", "/api/v1/user/payouts", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotNil(t, response["payouts"])
	})

	t.Run("payouts with pagination", func(t *testing.T) {
		handlers := createDefaultUserHandlers()
		router := setupUserRouter(handlers)

		req, _ := http.NewRequest("GET", "/api/v1/user/payouts?limit=10&offset=5", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(10), response["limit"])
		assert.Equal(t, float64(5), response["offset"])
	})
}

// =============================================================================
// STATS HISTORY TESTS (TDD)
// =============================================================================

func TestUserHandlers_GetHashrateHistory(t *testing.T) {
	t.Run("successful get hashrate history", func(t *testing.T) {
		handlers := createDefaultUserHandlers()
		router := setupUserRouter(handlers)

		req, _ := http.NewRequest("GET", "/api/v1/user/stats/hashrate?period=24h", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "24h", response["period"])
	})

	t.Run("default period when invalid", func(t *testing.T) {
		handlers := createDefaultUserHandlers()
		router := setupUserRouter(handlers)

		req, _ := http.NewRequest("GET", "/api/v1/user/stats/hashrate?period=invalid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "24h", response["period"])
	})
}

func TestUserHandlers_GetSharesHistory(t *testing.T) {
	t.Run("successful get shares history", func(t *testing.T) {
		handlers := createDefaultUserHandlers()
		router := setupUserRouter(handlers)

		req, _ := http.NewRequest("GET", "/api/v1/user/stats/shares?period=7d", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "7d", response["period"])
	})
}

func TestUserHandlers_GetEarningsHistory(t *testing.T) {
	t.Run("successful get earnings history", func(t *testing.T) {
		handlers := createDefaultUserHandlers()
		router := setupUserRouter(handlers)

		req, _ := http.NewRequest("GET", "/api/v1/user/stats/earnings?period=30d", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "30d", response["period"])
	})
}

// =============================================================================
// ISP INTERFACE COMPLIANCE TESTS
// =============================================================================

func TestISP_UserHandlers_InterfaceSegregation(t *testing.T) {
	t.Run("UserProfileReader has single responsibility", func(t *testing.T) {
		var reader UserProfileReader = &MockUserProfileReader{}
		_, _ = reader.GetProfile(1)
	})

	t.Run("UserProfileWriter has single responsibility", func(t *testing.T) {
		var writer UserProfileWriter = &MockUserProfileWriter{}
		_, _ = writer.UpdateProfile(1, &UpdateProfileData{})
	})

	t.Run("UserPasswordChanger has single responsibility", func(t *testing.T) {
		var changer UserPasswordChanger = &MockUserPasswordChanger{}
		_ = changer.ChangePassword(1, "old", "new")
	})

	t.Run("handlers can work with minimal dependencies", func(t *testing.T) {
		handlers := NewUserHandlers(
			&MockUserProfileReader{},
			&MockUserProfileWriter{},
			&MockUserPasswordChanger{},
			&MockUserMinerReader{},
			&MockUserPayoutReader{},
			&MockUserStatsReader{},
		)
		assert.NotNil(t, handlers)
	})
}
