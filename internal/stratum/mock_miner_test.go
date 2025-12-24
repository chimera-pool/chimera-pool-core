package stratum

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockMiner represents a mock mining client for E2E testing
type MockMiner struct {
	conn     net.Conn
	scanner  *bufio.Scanner
	workerID string
	password string
}

// NewMockMiner creates a new mock miner client
func NewMockMiner(serverAddr, workerID, password string) (*MockMiner, error) {
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	return &MockMiner{
		conn:     conn,
		scanner:  bufio.NewScanner(conn),
		workerID: workerID,
		password: password,
	}, nil
}

// Close closes the miner connection
func (m *MockMiner) Close() error {
	return m.conn.Close()
}

// SendMessage sends a Stratum message to the server
func (m *MockMiner) SendMessage(msg *StratumMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	_, err = m.conn.Write(append(data, '\n'))
	return err
}

// ReadResponse reads a response from the server
func (m *MockMiner) ReadResponse() (*StratumResponse, error) {
	m.conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	if !m.scanner.Scan() {
		if err := m.scanner.Err(); err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}
		return nil, fmt.Errorf("no response received")
	}

	line := strings.TrimSpace(m.scanner.Text())

	var response StratumResponse
	if err := json.Unmarshal([]byte(line), &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// ReadAllResponses reads all available responses from the server
func (m *MockMiner) ReadAllResponses() ([]*StratumResponse, error) {
	var responses []*StratumResponse

	// Set a short deadline to read all available messages
	m.conn.SetReadDeadline(time.Now().Add(1 * time.Second))

	for m.scanner.Scan() {
		line := strings.TrimSpace(m.scanner.Text())
		if line == "" {
			continue
		}

		var response StratumResponse
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			// If we can't parse it, it might be a notification, skip it
			continue
		}

		responses = append(responses, &response)
	}

	if len(responses) == 0 {
		return nil, fmt.Errorf("no responses received")
	}

	return responses, nil
}

// Subscribe performs the mining.subscribe handshake
func (m *MockMiner) Subscribe() (*StratumResponse, error) {
	msg := &StratumMessage{
		ID:     1,
		Method: "mining.subscribe",
		Params: []interface{}{"MockMiner/1.0.0", nil},
	}

	if err := m.SendMessage(msg); err != nil {
		return nil, fmt.Errorf("failed to send subscribe: %w", err)
	}

	// Just read the first response (subscribe response)
	return m.ReadResponse()
}

// Authorize performs the mining.authorize handshake
func (m *MockMiner) Authorize() (*StratumResponse, error) {
	msg := &StratumMessage{
		ID:     2,
		Method: "mining.authorize",
		Params: []interface{}{m.workerID, m.password},
	}

	if err := m.SendMessage(msg); err != nil {
		return nil, fmt.Errorf("failed to send authorize: %w", err)
	}

	return m.ReadResponse()
}

// SubmitShare submits a mining share
func (m *MockMiner) SubmitShare(jobID, extranonce2, ntime, nonce string) (*StratumResponse, error) {
	msg := &StratumMessage{
		ID:     3,
		Method: "mining.submit",
		Params: []interface{}{m.workerID, jobID, extranonce2, ntime, nonce},
	}

	if err := m.SendMessage(msg); err != nil {
		return nil, fmt.Errorf("failed to send submit: %w", err)
	}

	return m.ReadResponse()
}

// TestE2EStratumProtocolCompliance tests end-to-end Stratum protocol compliance
func TestE2EStratumProtocolCompliance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}
	// Start server
	server := NewStratumServer(":0")
	go func() {
		_ = server.Start()
	}()
	time.Sleep(100 * time.Millisecond)

	// Create mock miner
	miner, err := NewMockMiner(server.GetAddress(), "test_worker", "test_password")
	require.NoError(t, err)
	defer miner.Close()

	// Test subscribe
	subscribeResp, err := miner.Subscribe()
	require.NoError(t, err)
	assert.Equal(t, 1, subscribeResp.ID)
	assert.NotNil(t, subscribeResp.Result)
	assert.Nil(t, subscribeResp.Error)

	// Validate subscribe response format
	result, ok := subscribeResp.Result.([]interface{})
	require.True(t, ok, "Subscribe result should be an array")
	require.Len(t, result, 3, "Subscribe result should have 3 elements")

	// Test authorize
	authorizeResp, err := miner.Authorize()
	require.NoError(t, err)
	assert.Equal(t, 2, authorizeResp.ID)
	assert.Equal(t, true, authorizeResp.Result)
	assert.Nil(t, authorizeResp.Error)

	// Test submit
	submitResp, err := miner.SubmitShare("job_123", "00000000", "507c7f00", "b2957c02")
	require.NoError(t, err)
	assert.Equal(t, 3, submitResp.ID)
	assert.Equal(t, true, submitResp.Result)
	assert.Nil(t, submitResp.Error)

	server.Stop()
}

// TestE2EMultipleMinersConcurrent tests multiple miners connecting concurrently
func TestE2EMultipleMinersConcurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}
	// Start server
	server := NewStratumServer(":0")
	go func() {
		_ = server.Start()
	}()
	time.Sleep(100 * time.Millisecond)

	numMiners := 5
	miners := make([]*MockMiner, numMiners)

	// Create multiple miners
	for i := 0; i < numMiners; i++ {
		miner, err := NewMockMiner(
			server.GetAddress(),
			fmt.Sprintf("worker_%d", i),
			"password",
		)
		require.NoError(t, err)
		miners[i] = miner
	}

	// Subscribe all miners concurrently
	for i, miner := range miners {
		go func(minerIndex int, m *MockMiner) {
			// Subscribe
			subscribeResp, err := m.Subscribe()
			assert.NoError(t, err, "Miner %d subscribe should succeed", minerIndex)
			assert.Equal(t, 1, subscribeResp.ID)

			// Authorize
			authorizeResp, err := m.Authorize()
			assert.NoError(t, err, "Miner %d authorize should succeed", minerIndex)
			assert.Equal(t, 2, authorizeResp.ID)
			assert.Equal(t, true, authorizeResp.Result)

			// Submit a share
			submitResp, err := m.SubmitShare("job_123", "00000000", "507c7f00", "b2957c02")
			assert.NoError(t, err, "Miner %d submit should succeed", minerIndex)
			assert.Equal(t, 3, submitResp.ID)
			assert.Equal(t, true, submitResp.Result)
		}(i, miner)
	}

	// Wait for all operations to complete
	time.Sleep(1 * time.Second)

	// Clean up
	for _, miner := range miners {
		miner.Close()
	}

	server.Stop()
}

// TestE2EProtocolErrorHandling tests error handling in the protocol
func TestE2EProtocolErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}
	// Start server
	server := NewStratumServer(":0")
	go func() {
		_ = server.Start()
	}()
	time.Sleep(100 * time.Millisecond)

	// Create mock miner
	miner, err := NewMockMiner(server.GetAddress(), "test_worker", "test_password")
	require.NoError(t, err)
	defer miner.Close()

	// Test invalid JSON
	_, err = miner.conn.Write([]byte("invalid json\n"))
	require.NoError(t, err)

	// Should receive error response
	response, err := miner.ReadResponse()
	require.NoError(t, err)
	assert.NotNil(t, response.Error, "Should receive error for invalid JSON")

	// Test unknown method
	unknownMsg := &StratumMessage{
		ID:     99,
		Method: "unknown.method",
		Params: []interface{}{},
	}
	err = miner.SendMessage(unknownMsg)
	require.NoError(t, err)

	response, err = miner.ReadResponse()
	require.NoError(t, err)
	assert.Equal(t, 99, response.ID)
	assert.NotNil(t, response.Error, "Should receive error for unknown method")

	server.Stop()
}

// TestE2EConnectionCleanup tests that connections are properly cleaned up
func TestE2EConnectionCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}
	// Start server
	server := NewStratumServer(":0")
	go func() {
		_ = server.Start()
	}()
	time.Sleep(100 * time.Millisecond)
	defer server.Stop()

	// Test that server handles connection lifecycle gracefully
	miner, err := NewMockMiner(server.GetAddress(), "test_worker", "test_password")
	require.NoError(t, err)

	// Subscribe to establish connection in server
	_, err = miner.Subscribe()
	require.NoError(t, err)

	// Close connection
	miner.Close()

	// Test that we can connect again after cleanup
	miner2, err := NewMockMiner(server.GetAddress(), "test_worker2", "test_password")
	require.NoError(t, err)
	defer miner2.Close()

	// Should be able to connect and operate normally
	_, err = miner2.Subscribe()
	assert.NoError(t, err, "Should be able to connect after previous connection closed")
}

// TestE2ESubmitWithoutAuthorization tests submitting without proper authorization
func TestE2ESubmitWithoutAuthorization(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}
	// Start server
	server := NewStratumServer(":0")
	go func() {
		_ = server.Start()
	}()
	time.Sleep(100 * time.Millisecond)

	// Create mock miner
	miner, err := NewMockMiner(server.GetAddress(), "test_worker", "test_password")
	require.NoError(t, err)
	defer miner.Close()

	// Subscribe but don't authorize
	_, err = miner.Subscribe()
	require.NoError(t, err)

	// Try to submit without authorization
	submitResp, err := miner.SubmitShare("job_123", "00000000", "507c7f00", "b2957c02")
	require.NoError(t, err)

	// Should receive error
	assert.Equal(t, 3, submitResp.ID)
	assert.Nil(t, submitResp.Result)
	assert.NotNil(t, submitResp.Error, "Should receive error for unauthorized submit")

	server.Stop()
}
