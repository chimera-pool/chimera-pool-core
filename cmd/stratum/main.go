package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

func main() {
	log.Println("🚀 Starting Chimera Pool Stratum Server...")

	// Load configuration
	config := loadConfig()

	// Initialize database connection
	db, err := initDatabase(config.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize Redis connection
	redisClient, err := initRedis(config.RedisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Create stratum server
	server := NewStratumServer(config, db, redisClient)

	// Start listening
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", config.Port))
	if err != nil {
		log.Fatalf("Failed to start stratum server: %v", err)
	}
	defer listener.Close()

	// Handle connections in goroutine
	go func() {
		log.Printf("✅ Stratum Server listening on port %s", config.Port)
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-server.done:
					return
				default:
					log.Printf("Accept error: %v", err)
					continue
				}
			}
			go server.HandleConnection(conn)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down stratum server...")
	server.Shutdown()
	log.Println("✅ Stratum server exited gracefully")
}

type Config struct {
	DatabaseURL    string
	RedisURL       string
	Port           string
	Difficulty     float64
	BlockDAGRPCURL string
	WalletAddress  string
	PoolFeePercent float64
}

func loadConfig() *Config {
	return &Config{
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://chimera:password@localhost:5432/chimera_pool?sslmode=disable"),
		RedisURL:       getEnv("REDIS_URL", "redis://localhost:6379/0"),
		Port:           getEnv("STRATUM_PORT", "18332"),
		Difficulty:     1.0,
		BlockDAGRPCURL: getEnv("BLOCKDAG_RPC_URL", "https://rpc.awakening.bdagscan.com"),
		WalletAddress:  getEnv("BLOCKDAG_WALLET_ADDRESS", ""),
		PoolFeePercent: 1.0,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func initDatabase(url string) (*sql.DB, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	log.Println("✅ Connected to PostgreSQL database")
	return db, nil
}

func initRedis(url string) (*redis.Client, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	log.Println("✅ Connected to Redis")
	return client, nil
}

// StratumServer handles mining connections
type StratumServer struct {
	config      *Config
	db          *sql.DB
	redis       *redis.Client
	miners      map[string]*Miner
	minersMutex sync.RWMutex
	done        chan struct{}
}

// Miner represents a connected miner
type Miner struct {
	ID           string
	Address      string
	Conn         net.Conn
	Authorized   bool
	Difficulty   float64
	SharesValid  int64
	SharesInvalid int64
	LastShare    time.Time
}

// StratumRequest represents an incoming stratum request
type StratumRequest struct {
	ID     interface{} `json:"id"`
	Method string      `json:"method"`
	Params []interface{} `json:"params"`
}

// StratumResponse represents an outgoing stratum response
type StratumResponse struct {
	ID     interface{} `json:"id"`
	Result interface{} `json:"result"`
	Error  interface{} `json:"error"`
}

// NewStratumServer creates a new stratum server
func NewStratumServer(config *Config, db *sql.DB, redisClient *redis.Client) *StratumServer {
	return &StratumServer{
		config: config,
		db:     db,
		redis:  redisClient,
		miners: make(map[string]*Miner),
		done:   make(chan struct{}),
	}
}

// HandleConnection handles a new miner connection
func (s *StratumServer) HandleConnection(conn net.Conn) {
	defer conn.Close()

	minerID := fmt.Sprintf("%s-%d", conn.RemoteAddr().String(), time.Now().UnixNano())
	miner := &Miner{
		ID:         minerID,
		Address:    conn.RemoteAddr().String(),
		Conn:       conn,
		Authorized: false,
		Difficulty: s.config.Difficulty,
	}

	s.minersMutex.Lock()
	s.miners[minerID] = miner
	s.minersMutex.Unlock()

	defer func() {
		s.minersMutex.Lock()
		delete(s.miners, minerID)
		s.minersMutex.Unlock()
		log.Printf("Miner disconnected: %s", minerID)
	}()

	log.Printf("New miner connection: %s", minerID)

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		if err := s.handleMessage(miner, line); err != nil {
			log.Printf("Error handling message from %s: %v", minerID, err)
			return
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Scanner error for %s: %v", minerID, err)
	}
}

// handleMessage processes a stratum message
func (s *StratumServer) handleMessage(miner *Miner, message string) error {
	var req StratumRequest
	if err := json.Unmarshal([]byte(message), &req); err != nil {
		return fmt.Errorf("invalid JSON: %v", err)
	}

	log.Printf("Received from %s: %s - %v", miner.ID, req.Method, req.Params)

	switch req.Method {
	case "mining.subscribe":
		return s.handleSubscribe(miner, req)
	case "mining.authorize":
		return s.handleAuthorize(miner, req)
	case "mining.submit":
		return s.handleSubmit(miner, req)
	case "mining.extranonce.subscribe":
		return s.sendResponse(miner, req.ID, true, nil)
	default:
		log.Printf("Unknown method from %s: %s", miner.ID, req.Method)
		return s.sendResponse(miner, req.ID, nil, "Unknown method")
	}
}

// handleSubscribe handles mining.subscribe
func (s *StratumServer) handleSubscribe(miner *Miner, req StratumRequest) error {
	// Generate subscription details
	subscriptionID := fmt.Sprintf("%x", time.Now().UnixNano())
	extranonce1 := fmt.Sprintf("%08x", time.Now().Unix())
	extranonce2Size := 4

	result := []interface{}{
		[][]string{
			{"mining.set_difficulty", subscriptionID},
			{"mining.notify", subscriptionID},
		},
		extranonce1,
		extranonce2Size,
	}

	if err := s.sendResponse(miner, req.ID, result, nil); err != nil {
		return err
	}

	// Send initial difficulty
	return s.sendNotification(miner, "mining.set_difficulty", []interface{}{miner.Difficulty})
}

// handleAuthorize handles mining.authorize
func (s *StratumServer) handleAuthorize(miner *Miner, req StratumRequest) error {
	if len(req.Params) < 1 {
		return s.sendResponse(miner, req.ID, false, "Missing parameters")
	}

	username, ok := req.Params[0].(string)
	if !ok {
		return s.sendResponse(miner, req.ID, false, "Invalid username")
	}

	// For now, accept all authorizations
	miner.Authorized = true
	log.Printf("Miner %s authorized as %s", miner.ID, username)

	// Record miner in database
	_, err := s.db.Exec(
		"INSERT INTO miners (user_id, name, address, is_active) VALUES (1, $1, $2, true) ON CONFLICT DO NOTHING",
		username, miner.Address,
	)
	if err != nil {
		log.Printf("Failed to record miner: %v", err)
	}

	return s.sendResponse(miner, req.ID, true, nil)
}

// handleSubmit handles mining.submit (share submission)
func (s *StratumServer) handleSubmit(miner *Miner, req StratumRequest) error {
	if !miner.Authorized {
		return s.sendResponse(miner, req.ID, false, "Not authorized")
	}

	// Validate share (simplified - in production would verify against blockchain)
	miner.SharesValid++
	miner.LastShare = time.Now()

	// Record share in database
	_, err := s.db.Exec(
		"INSERT INTO shares (miner_id, user_id, difficulty, is_valid, nonce, hash) VALUES (1, 1, $1, true, $2, $3)",
		miner.Difficulty, "submitted", "hash",
	)
	if err != nil {
		log.Printf("Failed to record share: %v", err)
	}

	// Update Redis stats
	ctx := context.Background()
	s.redis.Incr(ctx, "pool:shares:valid")
	s.redis.HIncrBy(ctx, fmt.Sprintf("miner:%s", miner.ID), "shares", 1)

	log.Printf("Share accepted from %s (total: %d)", miner.ID, miner.SharesValid)
	return s.sendResponse(miner, req.ID, true, nil)
}

// sendResponse sends a stratum response
func (s *StratumServer) sendResponse(miner *Miner, id interface{}, result interface{}, errMsg interface{}) error {
	resp := StratumResponse{
		ID:     id,
		Result: result,
		Error:  errMsg,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	_, err = miner.Conn.Write(append(data, '\n'))
	return err
}

// sendNotification sends a stratum notification
func (s *StratumServer) sendNotification(miner *Miner, method string, params []interface{}) error {
	notification := map[string]interface{}{
		"id":     nil,
		"method": method,
		"params": params,
	}

	data, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	_, err = miner.Conn.Write(append(data, '\n'))
	return err
}

// Shutdown gracefully shuts down the stratum server
func (s *StratumServer) Shutdown() {
	close(s.done)

	s.minersMutex.Lock()
	defer s.minersMutex.Unlock()

	for _, miner := range s.miners {
		miner.Conn.Close()
	}
}
