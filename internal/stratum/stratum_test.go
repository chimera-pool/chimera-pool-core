package stratum

import (
	"bufio"
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStratumMessageParsing tests the parsing of Stratum protocol messages
func TestStratumMessageParsing(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    *StratumMessage
		expectError bool
	}{
		{
			name:  "valid subscribe message",
			input: `{"id": 1, "method": "mining.subscribe", "params": ["cpuminer/2.5.0", null]}`,
			expected: &StratumMessage{
				ID:     1,
				Method: "mining.subscribe",
				Params: []interface{}{"cpuminer/2.5.0", nil},
			},
			expectError: false,
		},
		{
			name:  "valid authorize message",
			input: `{"id": 2, "method": "mining.authorize", "params": ["worker1", "password"]}`,
			expected: &StratumMessage{
				ID:     2,
				Method: "mining.authorize",
				Params: []interface{}{"worker1", "password"},
			},
			expectError: false,
		},
		{
			name:  "valid submit message",
			input: `{"id": 3, "method": "mining.submit", "params": ["worker1", "job_id", "extranonce2", "ntime", "nonce"]}`,
			expected: &StratumMessage{
				ID:     3,
				Method: "mining.submit",
				Params: []interface{}{"worker1", "job_id", "extranonce2", "ntime", "nonce"},
			},
			expectError: false,
		},
		{
			name:        "invalid json",
			input:       `{"id": 1, "method": "mining.subscribe"`,
			expected:    nil,
			expectError: true,
		},
		{
			name:        "missing required fields",
			input:       `{"id": 1}`,
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := ParseStratumMessage(tt.input)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, msg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.ID, msg.ID)
				assert.Equal(t, tt.expected.Method, msg.Method)
				assert.Equal(t, tt.expected.Params, msg.Params)
			}
		})
	}
}

// TestStratumResponseGeneration tests the generation of Stratum protocol responses
func TestStratumResponseGeneration(t *testing.T) {
	tests := []struct {
		name     string
		response *StratumResponse
		expected string
	}{
		{
			name: "subscribe response",
			response: &StratumResponse{
				ID:     1,
				Result: []interface{}{[]interface{}{"mining.notify", "subscription_id"}, "extranonce1", 4},
				Error:  nil,
			},
			expected: `{"id":1,"result":[["mining.notify","subscription_id"],"extranonce1",4],"error":null}`,
		},
		{
			name: "authorize success response",
			response: &StratumResponse{
				ID:     2,
				Result: true,
				Error:  nil,
			},
			expected: `{"id":2,"result":true,"error":null}`,
		},
		{
			name: "error response",
			response: &StratumResponse{
				ID:     3,
				Result: nil,
				Error:  []interface{}{21, "Job not found", nil},
			},
			expected: `{"id":3,"result":null,"error":[21,"Job not found",null]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.response.ToJSON()
			assert.NoError(t, err)
			assert.JSONEq(t, tt.expected, result)
		})
	}
}

// TestStratumServerConnection tests basic server connection handling
func TestStratumServerConnection(t *testing.T) {
	// This test will fail initially as we haven't implemented the server yet
	server := NewStratumServer(":0") // Use port 0 for automatic port assignment
	
	// Start server in background
	go func() {
		err := server.Start()
		if err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
			t.Errorf("Server start failed: %v", err)
		}
	}()
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	
	// Get the actual port the server is listening on
	addr := server.GetAddress()
	require.NotEmpty(t, addr, "Server address should not be empty")
	
	// Test connection
	conn, err := net.Dial("tcp", addr)
	require.NoError(t, err, "Should be able to connect to server")
	
	// Test that connection is established
	assert.NotNil(t, conn, "Connection should not be nil")
	
	// Close connection first
	conn.Close()
	
	// Give time for cleanup
	time.Sleep(100 * time.Millisecond)
	
	// Cleanup
	server.Stop()
}

// TestStratumServerSubscribe tests the mining.subscribe method
func TestStratumServerSubscribe(t *testing.T) {
	server := NewStratumServer(":0")
	
	go func() {
		server.Start()
	}()
	time.Sleep(100 * time.Millisecond)
	
	conn, err := net.Dial("tcp", server.GetAddress())
	require.NoError(t, err)
	defer conn.Close()
	
	// Send subscribe message
	subscribeMsg := `{"id": 1, "method": "mining.subscribe", "params": ["cpuminer/2.5.0", null]}` + "\n"
	_, err = conn.Write([]byte(subscribeMsg))
	require.NoError(t, err)
	
	// Read response
	scanner := bufio.NewScanner(conn)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	
	require.True(t, scanner.Scan(), "Should receive a response")
	response := strings.TrimSpace(scanner.Text())
	
	// Parse response
	var stratumResp StratumResponse
	err = json.Unmarshal([]byte(response), &stratumResp)
	require.NoError(t, err)
	
	// Validate response
	assert.Equal(t, 1, stratumResp.ID)
	assert.NotNil(t, stratumResp.Result)
	assert.Nil(t, stratumResp.Error)
	
	server.Stop()
}

// TestStratumServerAuthorize tests the mining.authorize method
func TestStratumServerAuthorize(t *testing.T) {
	server := NewStratumServer(":0")
	
	go func() {
		server.Start()
	}()
	
	// Wait for server to be ready
	var addr string
	for i := 0; i < 50; i++ {
		time.Sleep(10 * time.Millisecond)
		addr = server.GetAddress()
		if addr != ":0" {
			break
		}
	}
	require.NotEqual(t, ":0", addr, "Server should have started")
	
	conn, err := net.Dial("tcp", addr)
	require.NoError(t, err)
	defer conn.Close()
	
	// Send authorize message
	authorizeMsg := `{"id": 2, "method": "mining.authorize", "params": ["worker1", "password"]}` + "\n"
	_, err = conn.Write([]byte(authorizeMsg))
	require.NoError(t, err)
	
	// Read response
	buffer := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Read(buffer)
	require.NoError(t, err)
	
	response := string(buffer[:n])
	response = strings.TrimSpace(response)
	
	// Parse response
	var stratumResp StratumResponse
	err = json.Unmarshal([]byte(response), &stratumResp)
	require.NoError(t, err)
	
	// Validate response
	assert.Equal(t, 2, stratumResp.ID)
	assert.Equal(t, true, stratumResp.Result)
	assert.Nil(t, stratumResp.Error)
	
	server.Stop()
}

// TestStratumServerConcurrentConnections tests handling multiple concurrent connections
func TestStratumServerConcurrentConnections(t *testing.T) {
	server := NewStratumServer(":0")
	
	go func() {
		server.Start()
	}()
	
	// Wait for server to be ready
	var addr string
	for i := 0; i < 50; i++ {
		time.Sleep(10 * time.Millisecond)
		addr = server.GetAddress()
		if addr != ":0" {
			break
		}
	}
	require.NotEqual(t, ":0", addr, "Server should have started")
	
	// Create multiple concurrent connections
	numConnections := 10
	connections := make([]net.Conn, numConnections)
	
	for i := 0; i < numConnections; i++ {
		conn, err := net.Dial("tcp", addr)
		require.NoError(t, err)
		connections[i] = conn
	}
	
	// Send messages from all connections
	for i, conn := range connections {
		subscribeMsg := `{"id": ` + string(rune(i+1)) + `, "method": "mining.subscribe", "params": ["cpuminer/2.5.0", null]}` + "\n"
		_, err := conn.Write([]byte(subscribeMsg))
		assert.NoError(t, err)
	}
	
	// Read responses from all connections
	for i, conn := range connections {
		buffer := make([]byte, 1024)
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		n, err := conn.Read(buffer)
		assert.NoError(t, err, "Connection %d should receive response", i)
		
		response := string(buffer[:n])
		assert.NotEmpty(t, response, "Response should not be empty for connection %d", i)
	}
	
	// Close all connections
	for _, conn := range connections {
		conn.Close()
	}
	
	server.Stop()
}

// TestStratumServerResourceCleanup tests that resources are cleaned up when connections close
func TestStratumServerResourceCleanup(t *testing.T) {
	server := NewStratumServer(":0")
	
	go func() {
		server.Start()
	}()
	
	// Wait for server to be ready
	var addr string
	for i := 0; i < 50; i++ {
		time.Sleep(10 * time.Millisecond)
		addr = server.GetAddress()
		if addr != ":0" {
			break
		}
	}
	require.NotEqual(t, ":0", addr, "Server should have started")
	defer server.Stop()
	
	// Test that server can handle connection and disconnection gracefully
	conn, err := net.Dial("tcp", addr)
	require.NoError(t, err)
	
	// Send a message to establish the connection in server
	subscribeMsg := `{"id": 1, "method": "mining.subscribe", "params": ["cpuminer/2.5.0", null]}` + "\n"
	_, err = conn.Write([]byte(subscribeMsg))
	require.NoError(t, err)
	
	// Read response to ensure connection is established
	scanner := bufio.NewScanner(conn)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	require.True(t, scanner.Scan(), "Should receive response")
	
	// Close connection
	conn.Close()
	
	// Test that we can still connect after previous connection closed
	conn2, err := net.Dial("tcp", addr)
	require.NoError(t, err)
	defer conn2.Close()
	
	// Should be able to use the server normally
	_, err = conn2.Write([]byte(subscribeMsg))
	assert.NoError(t, err, "Should be able to connect after previous connection closed")
}

// TestStratumServerConnectionCount tests connection counting functionality
func TestStratumServerConnectionCount(t *testing.T) {
	server := NewStratumServer(":0")
	
	go func() {
		server.Start()
	}()
	
	// Wait for server to be ready
	var addr string
	for i := 0; i < 50; i++ {
		time.Sleep(10 * time.Millisecond)
		addr = server.GetAddress()
		if addr != ":0" {
			break
		}
	}
	require.NotEqual(t, ":0", addr, "Server should have started")
	defer server.Stop()
	
	// Initially should have 0 connections
	assert.Equal(t, 0, server.GetConnectionCount())
	
	// Create a connection
	conn, err := net.Dial("tcp", addr)
	require.NoError(t, err)
	defer conn.Close()
	
	// Send subscribe to establish connection in server
	subscribeMsg := `{"id": 1, "method": "mining.subscribe", "params": ["cpuminer/2.5.0", null]}` + "\n"
	_, err = conn.Write([]byte(subscribeMsg))
	require.NoError(t, err)
	
	// Read response to ensure connection is established
	scanner := bufio.NewScanner(conn)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	require.True(t, scanner.Scan(), "Should receive response")
	
	// Give time for connection to be registered
	time.Sleep(50 * time.Millisecond)
	
	// Should have 1 connection now
	assert.Equal(t, 1, server.GetConnectionCount())
}

// TestStratumNotificationGeneration tests notification message generation
func TestStratumNotificationGeneration(t *testing.T) {
	t.Run("Notify Notification", func(t *testing.T) {
		notification := NewNotifyNotification(
			"job_123",
			"prev_hash_456",
			"coinbase1_789",
			"coinbase2_abc",
			[]string{"merkle1", "merkle2"},
			"version_def",
			"nbits_ghi",
			"ntime_jkl",
			true,
		)
		
		assert.Equal(t, "mining.notify", notification.Method)
		assert.Len(t, notification.Params, 9)
		assert.Equal(t, "job_123", notification.Params[0])
		assert.Equal(t, "prev_hash_456", notification.Params[1])
		assert.Equal(t, true, notification.Params[8])
		
		// Test JSON generation
		jsonStr, err := notification.ToJSON()
		assert.NoError(t, err)
		assert.Contains(t, jsonStr, "mining.notify")
		assert.Contains(t, jsonStr, "job_123")
	})
	
	t.Run("Difficulty Notification", func(t *testing.T) {
		notification := NewDifficultyNotification(1024.5)
		
		assert.Equal(t, "mining.set_difficulty", notification.Method)
		assert.Len(t, notification.Params, 1)
		assert.Equal(t, 1024.5, notification.Params[0])
		
		// Test JSON generation
		jsonStr, err := notification.ToJSON()
		assert.NoError(t, err)
		assert.Contains(t, jsonStr, "mining.set_difficulty")
		assert.Contains(t, jsonStr, "1024.5")
	})
}