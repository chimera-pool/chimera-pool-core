package testutil

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/chimera-pool/chimera-pool-core/internal/database"
)

// MockMiner represents a simulated miner for testing
type MockMiner struct {
	ID       string
	Password string
	conn     net.Conn
	stats    MinerStats
	mining   bool
	hashrate uint64
}

// MinerStats holds mining statistics
type MinerStats struct {
	SharesSubmitted uint64
	SharesAccepted  uint64
	SharesRejected  uint64
	Hashrate        uint64
	ConnectedAt     time.Time
	LastActivity    time.Time
}

// NewMockMiner creates a new mock miner
func NewMockMiner(id, password string) *MockMiner {
	return &MockMiner{
		ID:       id,
		Password: password,
		stats: MinerStats{
			ConnectedAt: time.Now(),
		},
	}
}

// Connect connects the mock miner to the stratum server
func (m *MockMiner) Connect(address string) error {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", address, err)
	}

	m.conn = conn
	m.stats.ConnectedAt = time.Now()
	return nil
}

// Disconnect disconnects the mock miner
func (m *MockMiner) Disconnect() error {
	m.mining = false
	if m.conn != nil {
		return m.conn.Close()
	}
	return nil
}

// Subscribe sends a mining.subscribe message
func (m *MockMiner) Subscribe() error {
	if m.conn == nil {
		return fmt.Errorf("not connected")
	}

	message := fmt.Sprintf(`{"id": 1, "method": "mining.subscribe", "params": ["MockMiner/1.0", null]}%c`, '\n')
	_, err := m.conn.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("failed to send subscribe: %w", err)
	}

	// Read response (simplified)
	buffer := make([]byte, 1024)
	_, err = m.conn.Read(buffer)
	return err
}

// Authorize sends a mining.authorize message
func (m *MockMiner) Authorize(username, password string) error {
	if m.conn == nil {
		return fmt.Errorf("not connected")
	}

	message := fmt.Sprintf(`{"id": 2, "method": "mining.authorize", "params": ["%s", "%s"]}%c`, username, password, '\n')
	_, err := m.conn.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("failed to send authorize: %w", err)
	}

	// Read response (simplified)
	buffer := make([]byte, 1024)
	_, err = m.conn.Read(buffer)
	return err
}

// StartMining starts the mining simulation
func (m *MockMiner) StartMining(ctx context.Context, hashrate uint64) error {
	if m.conn == nil {
		return fmt.Errorf("not connected")
	}

	m.mining = true
	m.hashrate = hashrate

	go m.miningLoop(ctx)
	return nil
}

// miningLoop simulates mining activity
func (m *MockMiner) miningLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !m.mining {
				return
			}

			// Simulate share submission based on hashrate
			if m.hashrate > 0 {
				// Submit a share every few seconds based on hashrate
				if time.Now().Unix()%3 == 0 {
					m.submitShare()
				}
			}
		}
	}
}

// submitShare submits a mock share
func (m *MockMiner) submitShare() {
	if m.conn == nil {
		return
	}

	// Generate mock share data
	nonce := time.Now().UnixNano()
	message := fmt.Sprintf(`{"id": %d, "method": "mining.submit", "params": ["%s", "job_123", "%x", "%x", "%x"]}%c`,
		nonce%1000, m.ID, nonce, time.Now().Unix(), nonce*2, '\n')

	_, err := m.conn.Write([]byte(message))
	if err == nil {
		m.stats.SharesSubmitted++
		m.stats.SharesAccepted++ // Assume all shares are accepted for mock
		m.stats.LastActivity = time.Now()
	}
}

// GetStats returns current mining statistics
func (m *MockMiner) GetStats() MinerStats {
	return m.stats
}

// APIClient represents a test HTTP client
type APIClient struct {
	baseURL string
	token   string
	client  *http.Client
}

// NewAPIClient creates a new API test client
func NewAPIClient(baseURL, token string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		token:   token,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Get performs a GET request
func (c *APIClient) Get(path string) (*http.Response, error) {
	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	return c.client.Do(req)
}

// PostJSON performs a POST request with JSON body
func (c *APIClient) PostJSON(path string, data interface{}) (*http.Response, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.baseURL+path, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	return c.client.Do(req)
}

// TestUser represents a test user
type TestUser struct {
	ID       string
	Username string
	Email    string
	Password string
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// TokenResponse represents a token response
type TokenResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// SetupIntegrationDatabase creates and configures a database for integration tests
func SetupIntegrationDatabase(ctx context.Context) (*database.Database, error) {
	// This would typically connect to a test database
	// For now, return nil as the database.Connect function may not exist
	// Integration tests should use the SetupTestDatabase from testutil.go instead
	return nil, fmt.Errorf("use SetupTestDatabase from testutil.go for containerized tests")
}

// GenerateTOTP generates a TOTP code for testing
func GenerateTOTP(secret string) string {
	// Decode base32 secret
	key, err := base32.StdEncoding.DecodeString(strings.ToUpper(secret))
	if err != nil {
		return "000000"
	}

	// Get current time step
	timeStep := time.Now().Unix() / 30

	// Convert to bytes
	timeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timeBytes, uint64(timeStep))

	// Generate HMAC
	h := hmac.New(sha1.New, key)
	h.Write(timeBytes)
	hash := h.Sum(nil)

	// Extract dynamic binary code
	offset := hash[len(hash)-1] & 0x0F
	code := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7FFFFFFF

	// Generate 6-digit code
	return fmt.Sprintf("%06d", code%1000000)
}

// MemoryStats represents memory usage statistics
type MemoryStats struct {
	AllocMB      int64
	TotalAllocMB int64
	SysMB        int64
	NumGC        uint32
}

// GetMemoryStats returns current memory usage statistics
func GetMemoryStats() MemoryStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return MemoryStats{
		AllocMB:      int64(m.Alloc / 1024 / 1024),
		TotalAllocMB: int64(m.TotalAlloc / 1024 / 1024),
		SysMB:        int64(m.Sys / 1024 / 1024),
		NumGC:        m.NumGC,
	}
}

// DecodeJSONResponse decodes a JSON response into the provided interface
func DecodeJSONResponse(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(v)
}

// ReadResponseBody reads the entire response body
func ReadResponseBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// WaitForService waits for a service to become available
func WaitForService(address string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", address, time.Second)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("service at %s did not become available within %v", address, timeout)
}

// CreateTestData creates test data for integration tests
// Note: This is a placeholder - actual test data creation should use raw SQL
func CreateTestData(ctx context.Context, db *database.Database) error {
	// Test data creation is handled by individual test setup functions
	// This function is kept for backward compatibility
	return nil
}

// CleanupTestData removes test data after tests
// Note: This is a placeholder - actual cleanup should use raw SQL connection
func CleanupTestData(ctx context.Context, db *database.Database) error {
	// Cleanup is handled by testcontainers in testutil.go
	// This function is kept for backward compatibility
	return nil
}
