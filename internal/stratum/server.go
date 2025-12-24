package stratum

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// StratumServer represents a Stratum mining protocol server
type StratumServer struct {
	address     string
	listener    net.Listener
	listenerMu  sync.RWMutex // Protects listener access
	connections map[string]*ClientConnection
	connMutex   sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup

	// Connection tracking
	connectionCount int64

	// Configuration
	extranonce2Size int
	difficulty      float64
}

// ClientConnection represents a connected miner client
type ClientConnection struct {
	ID           string
	Conn         net.Conn
	Subscribed   bool
	Authorized   bool
	WorkerName   string
	Extranonce1  string
	LastActivity time.Time

	// Communication channels
	sendChan chan string
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewStratumServer creates a new Stratum server
func NewStratumServer(address string) *StratumServer {
	ctx, cancel := context.WithCancel(context.Background())

	return &StratumServer{
		address:         address,
		connections:     make(map[string]*ClientConnection),
		ctx:             ctx,
		cancel:          cancel,
		extranonce2Size: 4,
		difficulty:      1.0,
	}
}

// Start starts the Stratum server
func (s *StratumServer) Start() error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}

	// Set listener with proper synchronization
	s.listenerMu.Lock()
	s.listener = listener
	s.listenerMu.Unlock()

	// Accept connections
	for {
		select {
		case <-s.ctx.Done():
			return nil
		default:
			conn, err := listener.Accept()
			if err != nil {
				if s.ctx.Err() != nil {
					return nil // Server is shutting down
				}
				continue
			}

			// Handle connection in goroutine
			s.wg.Add(1)
			go s.handleConnection(conn)
		}
	}
}

// Stop stops the Stratum server
func (s *StratumServer) Stop() error {
	s.cancel()

	// Close listener with proper synchronization
	s.listenerMu.Lock()
	if s.listener != nil {
		s.listener.Close()
	}
	s.listenerMu.Unlock()

	// Close all client connections
	s.connMutex.Lock()
	for _, client := range s.connections {
		client.cancel()
		client.Conn.Close()
	}
	s.connMutex.Unlock()

	s.wg.Wait()
	return nil
}

// GetAddress returns the server's listening address
func (s *StratumServer) GetAddress() string {
	s.listenerMu.RLock()
	defer s.listenerMu.RUnlock()

	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.address
}

// GetConnectionCount returns the current number of connections
func (s *StratumServer) GetConnectionCount() int {
	return int(atomic.LoadInt64(&s.connectionCount))
}

// SetDifficulty sets the server's default mining difficulty
func (s *StratumServer) SetDifficulty(difficulty float64) {
	s.difficulty = difficulty
}

// handleConnection handles a new client connection
func (s *StratumServer) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	// Create client connection
	clientCtx, clientCancel := context.WithCancel(s.ctx)
	client := &ClientConnection{
		ID:           uuid.New().String(),
		Conn:         conn,
		Subscribed:   false,
		Authorized:   false,
		Extranonce1:  generateExtranonce1(),
		LastActivity: time.Now(),
		sendChan:     make(chan string, 100),
		ctx:          clientCtx,
		cancel:       clientCancel,
	}

	// Add to connections map
	s.connMutex.Lock()
	s.connections[client.ID] = client
	s.connMutex.Unlock()

	atomic.AddInt64(&s.connectionCount, 1)

	// Ensure cleanup happens when function exits
	defer s.cleanupConnection(client)

	// Start send goroutine
	s.wg.Add(1)
	go s.handleClientSend(client)

	// Handle incoming messages
	scanner := bufio.NewScanner(conn)
	for {
		select {
		case <-clientCtx.Done():
			return
		default:
			// Set a read timeout to avoid blocking indefinitely
			// 5 seconds is more forgiving for miners on slow/unstable networks
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			if scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line != "" {
					s.handleMessage(client, line)
					client.LastActivity = time.Now()
				}
			} else {
				// Check if it's a timeout or actual error
				if err := scanner.Err(); err != nil {
					return
				}
				// If scanner returns false without error, it's likely EOF (connection closed)
				return
			}
		}
	}
}

// handleClientSend handles sending messages to a client
func (s *StratumServer) handleClientSend(client *ClientConnection) {
	defer s.wg.Done()

	for {
		select {
		case <-client.ctx.Done():
			return
		case message := <-client.sendChan:
			client.Conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
			if _, err := client.Conn.Write([]byte(message + "\n")); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming Stratum messages
func (s *StratumServer) handleMessage(client *ClientConnection, data string) {
	msg, err := ParseStratumMessage(data)
	if err != nil {
		// Send error response
		errorResp := NewErrorResponse(0, 20, "Parse error")
		if jsonResp, err := errorResp.ToJSON(); err == nil {
			select {
			case client.sendChan <- jsonResp:
			default:
			}
		}
		return
	}

	switch msg.Method {
	case "mining.subscribe":
		s.handleSubscribe(client, msg)
	case "mining.authorize":
		s.handleAuthorize(client, msg)
	case "mining.submit":
		s.handleSubmit(client, msg)
	default:
		// Unknown method
		errorResp := NewErrorResponse(msg.ID, 20, "Unknown method")
		if jsonResp, err := errorResp.ToJSON(); err == nil {
			select {
			case client.sendChan <- jsonResp:
			default:
			}
		}
	}
}

// handleSubscribe handles mining.subscribe requests
func (s *StratumServer) handleSubscribe(client *ClientConnection, msg *StratumMessage) {
	// Mark client as subscribed
	client.Subscribed = true

	// Create subscribe response
	resp := NewSubscribeResponse(msg.ID, client.ID, client.Extranonce1, s.extranonce2Size)

	if jsonResp, err := resp.ToJSON(); err == nil {
		select {
		case client.sendChan <- jsonResp:
		default:
		}
	}

	// Note: Difficulty notification will be sent separately when needed
}

// handleAuthorize handles mining.authorize requests
func (s *StratumServer) handleAuthorize(client *ClientConnection, msg *StratumMessage) {
	// Basic authorization - accept all for now
	if len(msg.Params) >= 1 {
		if workerName, ok := msg.Params[0].(string); ok {
			client.WorkerName = workerName
			client.Authorized = true
		}
	}

	// Create authorize response
	resp := NewAuthorizeResponse(msg.ID, client.Authorized)

	if jsonResp, err := resp.ToJSON(); err == nil {
		select {
		case client.sendChan <- jsonResp:
		default:
		}
	}
}

// handleSubmit handles mining.submit requests
func (s *StratumServer) handleSubmit(client *ClientConnection, msg *StratumMessage) {
	// Basic submit handling - accept all for now
	// In a real implementation, this would validate the share

	if !client.Authorized {
		errorResp := NewErrorResponse(msg.ID, 24, "Unauthorized worker")
		if jsonResp, err := errorResp.ToJSON(); err == nil {
			select {
			case client.sendChan <- jsonResp:
			default:
			}
		}
		return
	}

	// For now, accept all submissions
	resp := NewSubmitResponse(msg.ID, true)

	if jsonResp, err := resp.ToJSON(); err == nil {
		select {
		case client.sendChan <- jsonResp:
		default:
		}
	}
}

// cleanupConnection removes a client connection and cleans up resources
func (s *StratumServer) cleanupConnection(client *ClientConnection) {
	// Cancel context first to signal all goroutines to stop
	client.cancel()

	s.connMutex.Lock()
	delete(s.connections, client.ID)
	s.connMutex.Unlock()

	atomic.AddInt64(&s.connectionCount, -1)

	// Safely close channel by draining any pending messages first
	// This prevents panic if a sender writes after cancel but before close
	go func() {
		// Small delay to allow handleClientSend to exit via context
		time.Sleep(10 * time.Millisecond)
		// Drain any remaining messages
		for {
			select {
			case <-client.sendChan:
				// Discard pending message
			default:
				// Channel is empty, safe to close
				close(client.sendChan)
				return
			}
		}
	}()
}

// generateExtranonce1 generates a unique extranonce1 value
func generateExtranonce1() string {
	return uuid.New().String()[:8]
}
