package stratum

import (
	"encoding/json"
	"fmt"
)

// StratumMessage represents a Stratum protocol message
type StratumMessage struct {
	ID     int           `json:"id"`
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
}

// StratumResponse represents a Stratum protocol response
type StratumResponse struct {
	ID     int         `json:"id"`
	Result interface{} `json:"result"`
	Error  interface{} `json:"error"`
}

// StratumNotification represents a Stratum protocol notification (no ID)
type StratumNotification struct {
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
}

// ParseStratumMessage parses a JSON string into a StratumMessage
func ParseStratumMessage(data string) (*StratumMessage, error) {
	var msg StratumMessage
	
	if err := json.Unmarshal([]byte(data), &msg); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	
	// Validate required fields
	if msg.Method == "" {
		return nil, fmt.Errorf("method field is required")
	}
	
	return &msg, nil
}

// ToJSON converts a StratumResponse to JSON string
func (r *StratumResponse) ToJSON() (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}
	return string(data), nil
}

// ToJSON converts a StratumNotification to JSON string
func (n *StratumNotification) ToJSON() (string, error) {
	data, err := json.Marshal(n)
	if err != nil {
		return "", fmt.Errorf("failed to marshal notification: %w", err)
	}
	return string(data), nil
}

// NewSubscribeResponse creates a response for mining.subscribe
func NewSubscribeResponse(id int, subscriptionID, extranonce1 string, extranonce2Size int) *StratumResponse {
	return &StratumResponse{
		ID: id,
		Result: []interface{}{
			[]interface{}{"mining.notify", subscriptionID},
			extranonce1,
			extranonce2Size,
		},
		Error: nil,
	}
}

// NewAuthorizeResponse creates a response for mining.authorize
func NewAuthorizeResponse(id int, authorized bool) *StratumResponse {
	return &StratumResponse{
		ID:     id,
		Result: authorized,
		Error:  nil,
	}
}

// NewSubmitResponse creates a response for mining.submit
func NewSubmitResponse(id int, accepted bool) *StratumResponse {
	return &StratumResponse{
		ID:     id,
		Result: accepted,
		Error:  nil,
	}
}

// NewErrorResponse creates an error response
func NewErrorResponse(id int, code int, message string) *StratumResponse {
	return &StratumResponse{
		ID:     id,
		Result: nil,
		Error:  []interface{}{code, message, nil},
	}
}

// NewNotifyNotification creates a mining.notify notification
func NewNotifyNotification(jobID, prevHash, coinbase1, coinbase2 string, merkleRoots []string, version, nbits, ntime string, cleanJobs bool) *StratumNotification {
	return &StratumNotification{
		Method: "mining.notify",
		Params: []interface{}{
			jobID,
			prevHash,
			coinbase1,
			coinbase2,
			merkleRoots,
			version,
			nbits,
			ntime,
			cleanJobs,
		},
	}
}

// NewDifficultyNotification creates a mining.set_difficulty notification
func NewDifficultyNotification(difficulty float64) *StratumNotification {
	return &StratumNotification{
		Method: "mining.set_difficulty",
		Params: []interface{}{difficulty},
	}
}