package stratum

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStratumV1ProtocolCompliance validates compliance with Stratum v1 protocol
func TestStratumV1ProtocolCompliance(t *testing.T) {
	// Start server
	server := NewStratumServer(":0")
	go func() {
		server.Start()
	}()
	time.Sleep(100 * time.Millisecond)
	defer server.Stop()

	// Create mock miner
	miner, err := NewMockMiner(server.GetAddress(), "test_worker", "test_password")
	require.NoError(t, err)
	defer miner.Close()

	t.Run("Subscribe Method Compliance", func(t *testing.T) {
		// Test mining.subscribe according to Stratum v1 spec
		subscribeResp, err := miner.Subscribe()
		require.NoError(t, err)

		// Validate response structure
		assert.Equal(t, 1, subscribeResp.ID)
		assert.Nil(t, subscribeResp.Error)
		
		// Result should be array with subscription details and extranonce info
		result, ok := subscribeResp.Result.([]interface{})
		require.True(t, ok, "Subscribe result must be an array")
		require.Len(t, result, 3, "Subscribe result must have 3 elements")

		// First element should be subscription details
		subscriptions, ok := result[0].([]interface{})
		require.True(t, ok, "First element must be subscription array")
		require.Len(t, subscriptions, 2, "Subscription must have method and ID")
		
		// Validate subscription method
		assert.Equal(t, "mining.notify", subscriptions[0])
		assert.NotEmpty(t, subscriptions[1], "Subscription ID must not be empty")

		// Second element should be extranonce1
		extranonce1, ok := result[1].(string)
		require.True(t, ok, "Extranonce1 must be a string")
		assert.NotEmpty(t, extranonce1, "Extranonce1 must not be empty")

		// Third element should be extranonce2 size (JSON numbers are float64)
		extranonce2SizeFloat, ok := result[2].(float64)
		require.True(t, ok, "Extranonce2 size must be a number")
		extranonce2Size := int(extranonce2SizeFloat)
		assert.Greater(t, extranonce2Size, 0, "Extranonce2 size must be positive")
	})

	t.Run("Authorize Method Compliance", func(t *testing.T) {
		// First subscribe
		_, err := miner.Subscribe()
		require.NoError(t, err)

		// Test mining.authorize according to Stratum v1 spec
		authorizeResp, err := miner.Authorize()
		require.NoError(t, err)

		// Validate response structure
		assert.Equal(t, 2, authorizeResp.ID)
		assert.Nil(t, authorizeResp.Error)
		
		// Result should be boolean true for successful authorization
		result, ok := authorizeResp.Result.(bool)
		require.True(t, ok, "Authorize result must be boolean")
		assert.True(t, result, "Authorization should succeed")
	})

	t.Run("Submit Method Compliance", func(t *testing.T) {
		// First subscribe and authorize
		_, err := miner.Subscribe()
		require.NoError(t, err)
		_, err = miner.Authorize()
		require.NoError(t, err)

		// Test mining.submit according to Stratum v1 spec
		submitResp, err := miner.SubmitShare("job_123", "00000000", "507c7f00", "b2957c02")
		require.NoError(t, err)

		// Validate response structure
		assert.Equal(t, 3, submitResp.ID)
		assert.Nil(t, submitResp.Error)
		
		// Result should be boolean indicating acceptance
		result, ok := submitResp.Result.(bool)
		require.True(t, ok, "Submit result must be boolean")
		// For this basic implementation, we accept all shares
		assert.True(t, result, "Share should be accepted")
	})

	t.Run("Error Response Compliance", func(t *testing.T) {
		// Test error response format for unknown method
		unknownMsg := &StratumMessage{
			ID:     99,
			Method: "unknown.method",
			Params: []interface{}{},
		}
		
		err := miner.SendMessage(unknownMsg)
		require.NoError(t, err)

		response, err := miner.ReadResponse()
		require.NoError(t, err)

		// Validate error response structure
		assert.Equal(t, 99, response.ID)
		assert.Nil(t, response.Result)
		assert.NotNil(t, response.Error)

		// Error should be array with [code, message, traceback]
		errorArray, ok := response.Error.([]interface{})
		require.True(t, ok, "Error must be an array")
		require.Len(t, errorArray, 3, "Error array must have 3 elements")

		// Validate error code is number
		errorCode, ok := errorArray[0].(float64) // JSON numbers are float64
		require.True(t, ok, "Error code must be a number")
		assert.Greater(t, errorCode, float64(0), "Error code must be positive")

		// Validate error message is string
		errorMessage, ok := errorArray[1].(string)
		require.True(t, ok, "Error message must be a string")
		assert.NotEmpty(t, errorMessage, "Error message must not be empty")
	})
}

// TestMessageFormatCompliance tests JSON-RPC 2.0 compliance for Stratum messages
func TestMessageFormatCompliance(t *testing.T) {
	t.Run("Request Message Format", func(t *testing.T) {
		// Test valid request message parsing
		validRequest := `{"id": 1, "method": "mining.subscribe", "params": ["miner/1.0", null]}`
		
		msg, err := ParseStratumMessage(validRequest)
		require.NoError(t, err)
		
		assert.Equal(t, 1, msg.ID)
		assert.Equal(t, "mining.subscribe", msg.Method)
		assert.Len(t, msg.Params, 2)
	})

	t.Run("Response Message Format", func(t *testing.T) {
		// Test response message generation
		response := &StratumResponse{
			ID:     1,
			Result: true,
			Error:  nil,
		}
		
		jsonStr, err := response.ToJSON()
		require.NoError(t, err)
		
		// Parse back to validate structure
		var parsed map[string]interface{}
		err = json.Unmarshal([]byte(jsonStr), &parsed)
		require.NoError(t, err)
		
		// Validate required fields
		assert.Contains(t, parsed, "id")
		assert.Contains(t, parsed, "result")
		assert.Contains(t, parsed, "error")
		
		assert.Equal(t, float64(1), parsed["id"])
		assert.Equal(t, true, parsed["result"])
		assert.Nil(t, parsed["error"])
	})

	t.Run("Notification Message Format", func(t *testing.T) {
		// Test notification message generation (no ID field)
		notification := &StratumNotification{
			Method: "mining.notify",
			Params: []interface{}{"job_id", "prev_hash", "coinbase1", "coinbase2", []string{}, "version", "nbits", "ntime", true},
		}
		
		jsonStr, err := notification.ToJSON()
		require.NoError(t, err)
		
		// Parse back to validate structure
		var parsed map[string]interface{}
		err = json.Unmarshal([]byte(jsonStr), &parsed)
		require.NoError(t, err)
		
		// Validate notification structure (no ID field)
		assert.NotContains(t, parsed, "id")
		assert.Contains(t, parsed, "method")
		assert.Contains(t, parsed, "params")
		
		assert.Equal(t, "mining.notify", parsed["method"])
		assert.NotNil(t, parsed["params"])
	})
}

// TestConcurrentConnectionHandling validates requirement 2.3 - handle concurrent connections efficiently
func TestConcurrentConnectionHandling(t *testing.T) {
	// Start server
	server := NewStratumServer(":0")
	go func() {
		server.Start()
	}()
	time.Sleep(100 * time.Millisecond)
	defer server.Stop()

	// Test with multiple concurrent connections
	numConnections := 10
	results := make(chan bool, numConnections)

	for i := 0; i < numConnections; i++ {
		go func(workerID int) {
			defer func() {
				if r := recover(); r != nil {
					results <- false
					return
				}
			}()

			// Create miner connection
			miner, err := NewMockMiner(server.GetAddress(), 
				"worker_"+string(rune(workerID)), "password")
			if err != nil {
				results <- false
				return
			}
			defer miner.Close()

			// Perform full handshake
			_, err = miner.Subscribe()
			if err != nil {
				results <- false
				return
			}

			_, err = miner.Authorize()
			if err != nil {
				results <- false
				return
			}

			// Submit a share
			_, err = miner.SubmitShare("job_123", "00000000", "507c7f00", "b2957c02")
			if err != nil {
				results <- false
				return
			}

			results <- true
		}(i)
	}

	// Wait for all connections to complete
	successCount := 0
	for i := 0; i < numConnections; i++ {
		select {
		case success := <-results:
			if success {
				successCount++
			}
		case <-time.After(10 * time.Second):
			t.Fatal("Timeout waiting for concurrent connections")
		}
	}

	// All connections should succeed
	assert.Equal(t, numConnections, successCount, 
		"All concurrent connections should complete successfully")
}

// TestResourceCleanup validates requirement 2.4 - clean up resources and handle reconnection gracefully
func TestResourceCleanup(t *testing.T) {
	// Start server
	server := NewStratumServer(":0")
	go func() {
		server.Start()
	}()
	time.Sleep(100 * time.Millisecond)
	defer server.Stop()

	// Test graceful reconnection (simplified test)
	miner, err := NewMockMiner(server.GetAddress(), "test_worker", "test_password")
	require.NoError(t, err)

	// Should be able to connect and perform operations normally
	_, err = miner.Subscribe()
	assert.NoError(t, err, "Should be able to connect and subscribe")

	_, err = miner.Authorize()
	assert.NoError(t, err, "Should be able to authorize")

	// Close connection
	miner.Close()

	// Test reconnection
	miner2, err := NewMockMiner(server.GetAddress(), "test_worker2", "test_password")
	require.NoError(t, err)
	defer miner2.Close()

	// Should be able to reconnect
	_, err = miner2.Subscribe()
	assert.NoError(t, err, "Should be able to reconnect after previous connection closed")

	_, err = miner2.Authorize()
	assert.NoError(t, err, "Should be able to authorize after reconnection")
}