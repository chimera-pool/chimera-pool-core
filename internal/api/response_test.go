package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupResponseTestRouter() *gin.Engine {
	return gin.New()
}

func TestRespondSuccess(t *testing.T) {
	router := setupResponseTestRouter()
	router.GET("/test", func(c *gin.Context) {
		RespondSuccess(c, map[string]string{"key": "value"}, "Operation successful")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, "Operation successful", response.Message)
}

func TestRespondCreated(t *testing.T) {
	router := setupResponseTestRouter()
	router.POST("/test", func(c *gin.Context) {
		RespondCreated(c, map[string]int{"id": 123}, "Resource created")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, "Resource created", response.Message)
}

func TestRespondBadRequest(t *testing.T) {
	router := setupResponseTestRouter()
	router.GET("/test", func(c *gin.Context) {
		RespondBadRequest(c, "Invalid input")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "bad_request", response.Error)
	assert.Equal(t, "Invalid input", response.Message)
	assert.Equal(t, http.StatusBadRequest, response.Code)
}

func TestRespondUnauthorized(t *testing.T) {
	router := setupResponseTestRouter()
	router.GET("/test", func(c *gin.Context) {
		RespondUnauthorized(c, "")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "unauthorized", response.Error)
	assert.Equal(t, "Authentication required", response.Message)
}

func TestRespondForbidden(t *testing.T) {
	router := setupResponseTestRouter()
	router.GET("/test", func(c *gin.Context) {
		RespondForbidden(c, "Custom forbidden message")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "forbidden", response.Error)
	assert.Equal(t, "Custom forbidden message", response.Message)
}

func TestRespondNotFound(t *testing.T) {
	router := setupResponseTestRouter()
	router.GET("/test", func(c *gin.Context) {
		RespondNotFound(c, "")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "not_found", response.Error)
	assert.Equal(t, "Resource not found", response.Message)
}

func TestRespondInternalError(t *testing.T) {
	router := setupResponseTestRouter()
	router.GET("/test", func(c *gin.Context) {
		RespondInternalError(c, "Database connection failed")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "internal_error", response.Error)
	assert.Equal(t, "Database connection failed", response.Message)
}

func TestRespondPaginated(t *testing.T) {
	router := setupResponseTestRouter()
	router.GET("/test", func(c *gin.Context) {
		data := []string{"item1", "item2", "item3"}
		RespondPaginated(c, data, 1, 10, 25)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response PaginatedResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 1, response.Page)
	assert.Equal(t, 10, response.Limit)
	assert.Equal(t, int64(25), response.Total)
	assert.Equal(t, 3, response.TotalPages)
}

func TestGetUserIDFromContext(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(*gin.Context)
		wantID int64
		wantOK bool
	}{
		{
			name: "valid user ID",
			setup: func(c *gin.Context) {
				c.Set("user_id", int64(123))
			},
			wantID: 123,
			wantOK: true,
		},
		{
			name:   "missing user ID",
			setup:  func(c *gin.Context) {},
			wantID: 0,
			wantOK: false,
		},
		{
			name: "wrong type user ID",
			setup: func(c *gin.Context) {
				c.Set("user_id", "not-an-int")
			},
			wantID: 0,
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tt.setup(c)

			gotID, gotOK := GetUserIDFromContext(c)
			assert.Equal(t, tt.wantID, gotID)
			assert.Equal(t, tt.wantOK, gotOK)
		})
	}
}

func TestRequireUserID(t *testing.T) {
	t.Run("valid user ID", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", int64(123))

		id, ok := RequireUserID(c)
		assert.True(t, ok)
		assert.Equal(t, int64(123), id)
	})

	t.Run("missing user ID returns unauthorized", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		id, ok := RequireUserID(c)
		assert.False(t, ok)
		assert.Equal(t, int64(0), id)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
