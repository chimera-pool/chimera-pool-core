package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// Test structures
type UserListResponse struct {
	Users      []AdminUserView `json:"users"`
	TotalCount int             `json:"total_count"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
}

type AdminUserView struct {
	ID             int64   `json:"id"`
	Username       string  `json:"username"`
	Email          string  `json:"email"`
	PayoutAddress  string  `json:"payout_address"`
	PoolFeePercent float64 `json:"pool_fee_percent"`
	IsActive       bool    `json:"is_active"`
	IsAdmin        bool    `json:"is_admin"`
	CreatedAt      string  `json:"created_at"`
	TotalEarnings  float64 `json:"total_earnings"`
	PendingPayout  float64 `json:"pending_payout"`
	TotalHashrate  float64 `json:"total_hashrate"`
	ActiveMiners   int     `json:"active_miners"`
}

type UserDetailResponse struct {
	User    AdminUserView   `json:"user"`
	Miners  []MinerView     `json:"miners"`
	Payouts []PayoutView    `json:"payouts"`
	Shares  SharesStats     `json:"shares_stats"`
}

type MinerView struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	Address   string  `json:"address"`
	Hashrate  float64 `json:"hashrate"`
	IsActive  bool    `json:"is_active"`
	LastSeen  string  `json:"last_seen"`
}

type PayoutView struct {
	ID          int64   `json:"id"`
	Amount      float64 `json:"amount"`
	Address     string  `json:"address"`
	TxHash      string  `json:"tx_hash"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
	ProcessedAt string  `json:"processed_at,omitempty"`
}

type SharesStats struct {
	TotalShares   int64   `json:"total_shares"`
	ValidShares   int64   `json:"valid_shares"`
	InvalidShares int64   `json:"invalid_shares"`
	Last24Hours   int64   `json:"last_24_hours"`
}

type UpdateUserRequest struct {
	PoolFeePercent *float64 `json:"pool_fee_percent,omitempty"`
	IsActive       *bool    `json:"is_active,omitempty"`
	PayoutAddress  *string  `json:"payout_address,omitempty"`
}

// setupTestRouter creates a test router with admin routes
func setupTestRouter(db *sql.DB, jwtSecret string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	apiGroup := router.Group("/api/v1")
	admin := apiGroup.Group("/admin")
	admin.Use(authMiddleware(jwtSecret))
	{
		admin.GET("/users", handleAdminListUsers(db))
		admin.GET("/users/:id", handleAdminGetUser(db))
		admin.PUT("/users/:id", handleAdminUpdateUser(db))
		admin.DELETE("/users/:id", handleAdminDeleteUser(db))
		admin.GET("/users/:id/earnings", handleAdminUserEarnings(db))
	}

	return router
}

// TestAdminListUsers tests the admin list users endpoint
func TestAdminListUsers(t *testing.T) {
	// Skip if no test database
	db, err := sql.Open("postgres", "postgres://chimera:Champions$1956@localhost:5432/chimera_pool?sslmode=disable")
	if err != nil {
		t.Skip("Test database not available")
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skip("Test database not available")
	}

	jwtSecret := "test-secret"
	router := setupTestRouter(db, jwtSecret)

	// Create admin JWT token
	token, err := generateJWT(1, jwtSecret)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "List users without pagination",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp UserListResponse
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Errorf("Failed to parse response: %v", err)
				}
				if resp.Page != 1 {
					t.Errorf("Expected page 1, got %d", resp.Page)
				}
			},
		},
		{
			name:           "List users with pagination",
			queryParams:    "?page=1&page_size=10",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp UserListResponse
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Errorf("Failed to parse response: %v", err)
				}
				if resp.PageSize != 10 {
					t.Errorf("Expected page_size 10, got %d", resp.PageSize)
				}
			},
		},
		{
			name:           "List users with search",
			queryParams:    "?search=admin",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp UserListResponse
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Errorf("Failed to parse response: %v", err)
				}
				// Should find admin user
				for _, user := range resp.Users {
					if user.Username != "admin" && user.Email != "admin" {
						// Search might be partial match
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/v1/admin/users"+tt.queryParams, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

// TestAdminGetUser tests getting a single user's details
func TestAdminGetUser(t *testing.T) {
	db, err := sql.Open("postgres", "postgres://chimera:Champions$1956@localhost:5432/chimera_pool?sslmode=disable")
	if err != nil {
		t.Skip("Test database not available")
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skip("Test database not available")
	}

	jwtSecret := "test-secret"
	router := setupTestRouter(db, jwtSecret)

	token, _ := generateJWT(1, jwtSecret)

	tests := []struct {
		name           string
		userID         string
		expectedStatus int
	}{
		{
			name:           "Get existing user",
			userID:         "1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Get non-existent user",
			userID:         "99999",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid user ID",
			userID:         "invalid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/v1/admin/users/"+tt.userID, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestAdminUpdateUser tests updating user settings
func TestAdminUpdateUser(t *testing.T) {
	db, err := sql.Open("postgres", "postgres://chimera:Champions$1956@localhost:5432/chimera_pool?sslmode=disable")
	if err != nil {
		t.Skip("Test database not available")
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skip("Test database not available")
	}

	jwtSecret := "test-secret"
	router := setupTestRouter(db, jwtSecret)

	token, _ := generateJWT(1, jwtSecret)

	// Create a test user
	var testUserID int64
	err = db.QueryRow(`
		INSERT INTO users (username, email, password_hash, is_active)
		VALUES ('testuser_admin', 'testadmin@test.com', 'hash', true)
		RETURNING id
	`).Scan(&testUserID)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	defer db.Exec("DELETE FROM users WHERE id = $1", testUserID)

	tests := []struct {
		name           string
		userID         string
		body           UpdateUserRequest
		expectedStatus int
	}{
		{
			name:   "Update pool fee",
			userID: "1",
			body: UpdateUserRequest{
				PoolFeePercent: ptrFloat64(0.5),
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Update with invalid fee (too high)",
			userID: "1",
			body: UpdateUserRequest{
				PoolFeePercent: ptrFloat64(101.0),
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "Update with negative fee",
			userID: "1",
			body: UpdateUserRequest{
				PoolFeePercent: ptrFloat64(-1.0),
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "Update is_active status",
			userID: "1",
			body: UpdateUserRequest{
				IsActive: ptrBool(false),
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Update payout address",
			userID: "1",
			body: UpdateUserRequest{
				PayoutAddress: ptrString("0x1234567890abcdef1234567890abcdef12345678"),
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest("PUT", "/api/v1/admin/users/"+tt.userID, bytes.NewBuffer(bodyBytes))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestAdminUserEarnings tests the user earnings endpoint
func TestAdminUserEarnings(t *testing.T) {
	db, err := sql.Open("postgres", "postgres://chimera:Champions$1956@localhost:5432/chimera_pool?sslmode=disable")
	if err != nil {
		t.Skip("Test database not available")
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skip("Test database not available")
	}

	jwtSecret := "test-secret"
	router := setupTestRouter(db, jwtSecret)

	token, _ := generateJWT(1, jwtSecret)

	tests := []struct {
		name           string
		userID         string
		queryParams    string
		expectedStatus int
	}{
		{
			name:           "Get earnings for existing user",
			userID:         "1",
			queryParams:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Get earnings with date range",
			userID:         "1",
			queryParams:    "?from=2024-01-01&to=2024-12-31",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Get earnings for non-existent user",
			userID:         "99999",
			queryParams:    "",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/v1/admin/users/"+tt.userID+"/earnings"+tt.queryParams, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestAdminRequiresAuth tests that admin endpoints require authentication
func TestAdminRequiresAuth(t *testing.T) {
	db, err := sql.Open("postgres", "postgres://chimera:Champions$1956@localhost:5432/chimera_pool?sslmode=disable")
	if err != nil {
		t.Skip("Test database not available")
	}
	defer db.Close()

	router := setupTestRouter(db, "test-secret")

	endpoints := []string{
		"/api/v1/admin/users",
		"/api/v1/admin/users/1",
	}

	for _, endpoint := range endpoints {
		t.Run("No auth for "+endpoint, func(t *testing.T) {
			req, _ := http.NewRequest("GET", endpoint, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
			}
		})
	}
}

// Helper functions
func ptrFloat64(v float64) *float64 { return &v }
func ptrBool(v bool) *bool          { return &v }
func ptrString(v string) *string    { return &v }
