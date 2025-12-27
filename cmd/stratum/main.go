package main

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"github.com/chimera-pool/chimera-pool-core/internal/geolocation"
	"github.com/chimera-pool/chimera-pool-core/internal/monitoring/health"
	"github.com/chimera-pool/chimera-pool-core/internal/monitoring/recovery"
	"github.com/chimera-pool/chimera-pool-core/internal/stratum/hashrate"
	"github.com/chimera-pool/chimera-pool-core/internal/stratum/keepalive"
	"github.com/chimera-pool/chimera-pool-core/internal/stratum/merkle"
	v2binary "github.com/chimera-pool/chimera-pool-core/internal/stratum/v2/binary"
	"github.com/chimera-pool/chimera-pool-core/internal/stratum/vardiff"
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

	// Initialize and start health monitoring service
	ctx := context.Background()
	healthService := initHealthMonitor(config)
	if healthService != nil {
		if err := healthService.Start(ctx); err != nil {
			log.Printf("⚠️ Failed to start health monitor: %v", err)
		} else {
			log.Println("✅ Health monitoring service started")
			defer func() {
				stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				healthService.Stop(stopCtx)
				log.Println("✅ Health monitoring service stopped")
			}()
		}
	}

	// Create stratum server
	server := NewStratumServer(config, db, redisClient)

	// Connect stratum server as pool metrics provider for Prometheus
	if healthService != nil && healthService.GetExporter() != nil {
		healthService.GetExporter().SetPoolMetricsProvider(server)
		log.Println("✅ Pool metrics provider connected to Prometheus exporter")
	}

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
	// Litecoin RPC settings
	LitecoinRPCURL  string
	LitecoinRPCUser string
	LitecoinRPCPass string
}

func loadConfig() *Config {
	return &Config{
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://chimera:password@localhost:5432/chimera_pool?sslmode=disable"),
		RedisURL:        getEnv("REDIS_URL", "redis://localhost:6379/0"),
		Port:            getEnv("STRATUM_PORT", "18332"),
		Difficulty:      35000, // Optimized for X100 on Scrypt (~15 TH/s) - stable 10s shares
		BlockDAGRPCURL:  getEnv("BLOCKDAG_RPC_URL", "https://rpc.awakening.bdagscan.com"),
		WalletAddress:   getEnv("BLOCKDAG_WALLET_ADDRESS", ""),
		PoolFeePercent:  1.0,
		LitecoinRPCURL:  getEnv("LITECOIN_RPC_URL", "http://litecoind:9332"),
		LitecoinRPCUser: getEnv("LITECOIN_RPC_USER", "chimera"),
		LitecoinRPCPass: getEnv("LITECOIN_RPC_PASS", "ChimeraLTC2024!"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// initHealthMonitor initializes the node health monitoring service
func initHealthMonitor(config *Config) *health.HealthService {
	// Check if health monitoring is enabled via environment
	if getEnv("HEALTH_MONITOR_ENABLED", "true") == "false" {
		log.Println("⚠️ Health monitoring disabled via HEALTH_MONITOR_ENABLED=false")
		return nil
	}

	// Initialize Network Watchdog for automatic recovery on internet restore
	initNetworkWatchdog()

	// Create health service configuration from environment and config
	healthConfig := &health.ServiceConfig{
		MonitorConfig: &health.HealthMonitorConfig{
			CheckInterval:                    30 * time.Second,
			MaxRestartsPerHour:               10, // Increased from 3 - allows recovery from transient issues
			RestartCooldown:                  60 * time.Second,
			ConsecutiveFailuresBeforeRestart: 3,
			RPCTimeout:                       10 * time.Second,
			EnableAutoRestart:                getEnv("HEALTH_AUTO_RESTART", "true") == "true",
			EnableAlerts:                     true,
			AlertWebhookURL:                  getEnv("HEALTH_ALERT_WEBHOOK", ""),
		},
		LitecoinRPCURL:      config.LitecoinRPCURL,
		LitecoinRPCUser:     config.LitecoinRPCUser,
		LitecoinRPCPassword: config.LitecoinRPCPass,
		LitecoinContainer:   getEnv("LITECOIN_CONTAINER", "docker-litecoind-1"),
		BlockDAGRPCURL:      config.BlockDAGRPCURL,
		BlockDAGContainer:   getEnv("BLOCKDAG_CONTAINER", ""),
		CommandTimeout:      60 * time.Second,
		PrometheusEnabled:   getEnv("HEALTH_PROMETHEUS_ENABLED", "true") == "true",
		PrometheusAddr:      getEnv("HEALTH_PROMETHEUS_ADDR", ":9091"),
	}

	service := health.NewHealthService(healthConfig)

	log.Printf("🏥 Health monitor configured: LTC=%s, container=%s, prometheus=%s",
		healthConfig.LitecoinRPCURL,
		healthConfig.LitecoinContainer,
		healthConfig.PrometheusAddr)

	return service
}

// ResilientDB wraps sql.DB with automatic reconnection logic
type ResilientDB struct {
	db        *sql.DB
	url       string
	mu        sync.RWMutex
	healthy   bool
	lastCheck time.Time
}

func initDatabase(url string) (*ResilientDB, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute) // Close idle connections faster

	if err := db.Ping(); err != nil {
		return nil, err
	}

	log.Println("✅ Connected to PostgreSQL database")
	rdb := &ResilientDB{
		db:      db,
		url:     url,
		healthy: true,
	}

	// Start background health checker
	go rdb.healthChecker()

	return rdb, nil
}

// healthChecker runs periodic health checks and reconnects if needed
func (r *ResilientDB) healthChecker() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Get current db reference under lock
		r.mu.RLock()
		db := r.db
		r.mu.RUnlock()

		if err := db.Ping(); err != nil {
			log.Printf("⚠️ Database health check failed: %v", err)
			r.mu.Lock()
			r.healthy = false
			r.mu.Unlock()

			// Attempt reconnection
			r.reconnect()
		} else {
			r.mu.Lock()
			if !r.healthy {
				log.Println("✅ Database connection restored")
			}
			r.healthy = true
			r.lastCheck = time.Now()
			r.mu.Unlock()
		}
	}
}

// reconnect attempts to re-establish database connection
func (r *ResilientDB) reconnect() {
	for i := 0; i < 5; i++ {
		log.Printf("🔄 Attempting database reconnection (attempt %d/5)...", i+1)

		// Close existing connection under lock
		r.mu.Lock()
		r.db.Close()
		r.mu.Unlock()

		// Create new connection
		db, err := sql.Open("postgres", r.url)
		if err != nil {
			log.Printf("❌ Reconnection failed: %v", err)
			time.Sleep(time.Duration(i+1) * 2 * time.Second) // Exponential backoff
			continue
		}

		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(5 * time.Minute)
		db.SetConnMaxIdleTime(2 * time.Minute)

		if err := db.Ping(); err != nil {
			log.Printf("❌ Reconnection ping failed: %v", err)
			db.Close()
			time.Sleep(time.Duration(i+1) * 2 * time.Second)
			continue
		}

		r.mu.Lock()
		r.db = db
		r.healthy = true
		r.mu.Unlock()

		log.Println("✅ Database reconnected successfully")
		return
	}

	log.Println("❌ Failed to reconnect to database after 5 attempts")
}

// Exec executes a query with automatic retry on connection errors
func (r *ResilientDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	for i := 0; i < 3; i++ {
		r.mu.RLock()
		db := r.db
		r.mu.RUnlock()

		result, err := db.Exec(query, args...)
		if err == nil {
			return result, nil
		}

		if isConnectionError(err) {
			log.Printf("⚠️ DB connection error on Exec, retrying: %v", err)
			r.reconnect()
			continue
		}
		return nil, err
	}
	return nil, fmt.Errorf("database operation failed after 3 retries")
}

// QueryRow executes a query that returns a single row with retry
func (r *ResilientDB) QueryRow(query string, args ...interface{}) *sql.Row {
	r.mu.RLock()
	db := r.db
	r.mu.RUnlock()
	return db.QueryRow(query, args...)
}

// Query executes a query that returns rows with retry
func (r *ResilientDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	for i := 0; i < 3; i++ {
		r.mu.RLock()
		db := r.db
		r.mu.RUnlock()

		rows, err := db.Query(query, args...)
		if err == nil {
			return rows, nil
		}

		if isConnectionError(err) {
			log.Printf("⚠️ DB connection error on Query, retrying: %v", err)
			r.reconnect()
			continue
		}
		return nil, err
	}
	return nil, fmt.Errorf("database query failed after 3 retries")
}

// Close closes the database connection
func (r *ResilientDB) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.db.Close()
}

// IsHealthy returns the current health status
func (r *ResilientDB) IsHealthy() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.healthy
}

// isConnectionError checks if an error is a connection-related error
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "EOF") ||
		strings.Contains(errStr, "i/o timeout")
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
	config           *Config
	db               *ResilientDB
	redis            *redis.Client
	miners           map[string]*Miner
	minersMutex      sync.RWMutex
	done             chan struct{}
	currentJob       *MiningJob
	jobMutex         sync.RWMutex
	extranonce1      uint32
	extranonceMux    sync.Mutex
	vardiffManager   *vardiff.Manager
	keepaliveManager *keepalive.Manager
	merkleBuilder    *merkle.Builder
	hashrateWindows  map[string]*hashrate.Window
	hashrateMux      sync.RWMutex
	hashrateCalc     *hashrate.Calculator
	geoService       *geolocation.GeoIPService
}

// MiningJob represents current mining work from the node
type MiningJob struct {
	JobID          string
	PrevHash       string
	Coinbase1      string
	Coinbase2      string
	MerkleBranches []string
	Version        string
	NBits          string
	NTime          string
	Height         int64
	Target         string
}

// BlockTemplate from getblocktemplate RPC
type BlockTemplate struct {
	Version           int64    `json:"version"`
	PreviousBlockHash string   `json:"previousblockhash"`
	Transactions      []TxData `json:"transactions"`
	CoinbaseAux       struct {
		Flags string `json:"flags"`
	} `json:"coinbaseaux"`
	CoinbaseValue int64    `json:"coinbasevalue"`
	Target        string   `json:"target"`
	MinTime       int64    `json:"mintime"`
	Mutable       []string `json:"mutable"`
	NonceRange    string   `json:"noncerange"`
	SigOpLimit    int64    `json:"sigoplimit"`
	SizeLimit     int64    `json:"sizelimit"`
	CurTime       int64    `json:"curtime"`
	Bits          string   `json:"bits"`
	Height        int64    `json:"height"`
}

// TxData represents a transaction in block template
type TxData struct {
	Data   string `json:"data"`
	TxID   string `json:"txid"`
	Hash   string `json:"hash"`
	Fee    int64  `json:"fee"`
	SigOps int64  `json:"sigops"`
}

// Miner represents a connected miner
type Miner struct {
	ID            string
	UserID        int64  // Database user ID for share attribution
	Username      string // Authorized username
	Address       string
	Conn          net.Conn
	Authorized    bool
	Difficulty    float64
	SharesValid   int64
	SharesInvalid int64
	LastShare     time.Time
}

// StratumRequest represents an incoming stratum request
type StratumRequest struct {
	ID     interface{}   `json:"id"`
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
}

// StratumResponse represents an outgoing stratum response
type StratumResponse struct {
	ID     interface{} `json:"id"`
	Result interface{} `json:"result"`
	Error  interface{} `json:"error"`
}

// NewStratumServer creates a new stratum server
func NewStratumServer(config *Config, db *ResilientDB, redisClient *redis.Client) *StratumServer {
	// Configure vardiff - use X100-optimized configuration for Scrypt mining
	// X100 on Scrypt produces ~15 TH/s (vs ~70 TH/s on BlockDAG Scrpy-variant)
	vardiffConfig := vardiff.X100OptimizedConfig()
	// Allow config override for initial difficulty if specified
	if config.Difficulty > 0 {
		vardiffConfig.InitialDifficulty = config.Difficulty
	}

	// Configure keepalive
	keepaliveConfig := keepalive.DefaultConfig()
	keepaliveConfig.Interval = 30 * time.Second // Check every 30 seconds
	keepaliveConfig.MaxMissed = 3               // 3 missed = disconnect
	keepaliveConfig.SendWorkAsAlive = true      // Work updates count as keepalive

	s := &StratumServer{
		config:          config,
		db:              db,
		redis:           redisClient,
		miners:          make(map[string]*Miner),
		done:            make(chan struct{}),
		extranonce1:     1,
		vardiffManager:  vardiff.NewManager(vardiffConfig),
		merkleBuilder:   merkle.NewBuilder(),
		hashrateWindows: make(map[string]*hashrate.Window),
		hashrateCalc:    hashrate.NewCalculator(),
		geoService:      geolocation.NewGeoIPService(db.db),
	}

	log.Println("📍 IP geolocation service initialized for miner location tracking")

	// Initialize keepalive with disconnect callback
	s.keepaliveManager = keepalive.NewManager(keepaliveConfig, func(minerID string) {
		log.Printf("Keepalive timeout for miner %s, disconnecting", minerID)
		s.minersMutex.Lock()
		if miner, exists := s.miners[minerID]; exists {
			miner.Conn.Close()
			delete(s.miners, minerID)
		}
		s.minersMutex.Unlock()
	})

	// Start block template updater
	go s.blockTemplateUpdater()
	return s
}

// litecoinRPC makes an RPC call to the Litecoin node with retry logic
func (s *StratumServer) litecoinRPC(method string, params interface{}) (json.RawMessage, error) {
	return s.litecoinRPCWithRetry(method, params, 3) // Default 3 retries
}

// litecoinRPCWithRetry makes an RPC call with exponential backoff retry
func (s *StratumServer) litecoinRPCWithRetry(method string, params interface{}, maxRetries int) (json.RawMessage, error) {
	var lastErr error
	baseDelay := 500 * time.Millisecond

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 500ms, 1s, 2s, 4s...
			delay := baseDelay * time.Duration(1<<uint(attempt-1))
			if delay > 5*time.Second {
				delay = 5 * time.Second // Cap at 5 seconds
			}
			time.Sleep(delay)
		}

		result, err := s.doLitecoinRPC(method, params)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if error is retryable (MWEB errors, connection issues)
		errStr := err.Error()
		isRetryable := strings.Contains(errStr, "MWEB") ||
			strings.Contains(errStr, "mweb") ||
			strings.Contains(errStr, "connection") ||
			strings.Contains(errStr, "timeout") ||
			strings.Contains(errStr, "EOF") ||
			strings.Contains(errStr, "RPC error -1")

		if !isRetryable {
			return nil, err // Non-retryable error, fail immediately
		}

		if attempt < maxRetries {
			log.Printf("RPC call %s failed (attempt %d/%d): %v, retrying...",
				method, attempt+1, maxRetries+1, err)
		}
	}

	return nil, fmt.Errorf("RPC call %s failed after %d attempts: %v", method, maxRetries+1, lastErr)
}

// doLitecoinRPC performs a single RPC call without retry
func (s *StratumServer) doLitecoinRPC(method string, params interface{}) (json.RawMessage, error) {
	reqBody := map[string]interface{}{
		"jsonrpc": "1.0",
		"id":      "stratum",
		"method":  method,
		"params":  params,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", s.config.LitecoinRPCURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(s.config.LitecoinRPCUser, s.config.LitecoinRPCPass)
	req.Header.Set("Content-Type", "application/json")

	// Use shorter timeout for individual requests, retry handles recovery
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rpcResp struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, err
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}
	return rpcResp.Result, nil
}

// getBlockTemplate fetches a new block template from Litecoin
func (s *StratumServer) getBlockTemplate() (*BlockTemplate, error) {
	params := []interface{}{map[string]interface{}{"rules": []string{"segwit", "mweb"}}}
	result, err := s.litecoinRPC("getblocktemplate", params)
	if err != nil {
		return nil, err
	}
	var tmpl BlockTemplate
	if err := json.Unmarshal(result, &tmpl); err != nil {
		return nil, err
	}
	return &tmpl, nil
}

// buildMiningJob creates a mining job from a block template
func (s *StratumServer) buildMiningJob(tmpl *BlockTemplate) *MiningJob {
	jobID := fmt.Sprintf("%x", time.Now().Unix())

	// Reverse the previous block hash for stratum (it needs to be little-endian)
	prevHash := reverseHex(tmpl.PreviousBlockHash)

	// Build coinbase transaction
	// Coinbase1: version + input count + prev tx + prev index + script sig length
	// Coinbase2: sequence + output count + outputs + locktime

	// Height serialization for coinbase (BIP34)
	heightBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(heightBytes, uint32(tmpl.Height))
	heightHex := hex.EncodeToString(heightBytes[:3]) // Usually 3 bytes for height

	// Coinbase script sig: height + extranonce placeholder
	// We'll put extranonce1 (4 bytes) and extranonce2 (4 bytes) placeholders
	scriptSig := fmt.Sprintf("03%s", heightHex) // 03 = push 3 bytes, then height

	// Coinbase1: version(4) + txin_count(1) + prev_txid(32) + prev_vout(4) + script_len(1+) + script_start
	coinbase1 := "01000000" + // version
		"01" + // 1 input
		"0000000000000000000000000000000000000000000000000000000000000000" + // prev txid (null for coinbase)
		"ffffffff" + // prev vout
		fmt.Sprintf("%02x", len(scriptSig)/2+8) + // script length (script + extranonce1 + extranonce2)
		scriptSig

	// Coinbase2: extranonce padding + sequence + outputs + locktime
	// Output: pool reward address
	poolAddr := "ltc1qgsm3fv44wprdcsh3trgarm05rr7l8ryggujr5w" // Pool wallet
	_ = poolAddr                                              // We'll use a simple output for now

	// Coinbase value in satoshis to 8-byte little endian hex
	valueBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(valueBytes, uint64(tmpl.CoinbaseValue))
	valueHex := hex.EncodeToString(valueBytes)

	// Simple P2WPKH output (OP_0 + push20 + pubkeyhash)
	// For now use a placeholder output
	coinbase2 := "ffffffff" + // sequence
		"01" + // 1 output
		valueHex + // value
		"160014" + "846e292b5670116e217563468f6de863fc79c822" + // P2WPKH script
		"00000000" // locktime

	// Build merkle branches from transactions using the merkle builder
	var merkleBranches []string
	if len(tmpl.Transactions) > 0 {
		var txHashes [][]byte
		for _, tx := range tmpl.Transactions {
			hashBytes, _ := hex.DecodeString(tx.TxID)
			// Reverse for little-endian merkle
			for i, j := 0, len(hashBytes)-1; i < j; i, j = i+1, j-1 {
				hashBytes[i], hashBytes[j] = hashBytes[j], hashBytes[i]
			}
			txHashes = append(txHashes, hashBytes)
		}
		// Use the merkle builder for correct branch computation
		branch := s.merkleBuilder.BuildBranch(txHashes)
		merkleBranches = s.merkleBuilder.BranchToHex(branch)
	}

	return &MiningJob{
		JobID:          jobID,
		PrevHash:       prevHash,
		Coinbase1:      coinbase1,
		Coinbase2:      coinbase2,
		MerkleBranches: merkleBranches,
		Version:        fmt.Sprintf("%08x", tmpl.Version),
		NBits:          tmpl.Bits,
		NTime:          fmt.Sprintf("%08x", tmpl.CurTime),
		Height:         tmpl.Height,
		Target:         tmpl.Target,
	}
}

// reverseHex reverses a hex string byte by byte
func reverseHex(s string) string {
	bytes, _ := hex.DecodeString(s)
	for i, j := 0, len(bytes)-1; i < j; i, j = i+1, j-1 {
		bytes[i], bytes[j] = bytes[j], bytes[i]
	}
	return hex.EncodeToString(bytes)
}

// blockTemplateUpdater periodically fetches new block templates
func (s *StratumServer) blockTemplateUpdater() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Initial fetch
	s.updateBlockTemplate()

	for {
		select {
		case <-ticker.C:
			s.updateBlockTemplate()
		case <-s.done:
			return
		}
	}
}

// updateBlockTemplate fetches a new template and broadcasts to miners
func (s *StratumServer) updateBlockTemplate() {
	tmpl, err := s.getBlockTemplate()
	if err != nil {
		log.Printf("Failed to get block template: %v", err)
		return
	}

	job := s.buildMiningJob(tmpl)

	s.jobMutex.Lock()
	oldHeight := int64(0)
	if s.currentJob != nil {
		oldHeight = s.currentJob.Height
	}
	s.currentJob = job
	s.jobMutex.Unlock()

	// Broadcast new job to all miners if block changed
	if job.Height != oldHeight {
		log.Printf("New block template: height=%d, bits=%s", job.Height, job.NBits)
		s.broadcastJob(job, true)
	}
}

// broadcastJob sends a mining.notify to all connected miners
func (s *StratumServer) broadcastJob(job *MiningJob, cleanJobs bool) {
	s.minersMutex.RLock()
	defer s.minersMutex.RUnlock()

	for _, miner := range s.miners {
		if miner.Authorized {
			s.sendNotification(miner, "mining.notify", []interface{}{
				job.JobID,
				job.PrevHash,
				job.Coinbase1,
				job.Coinbase2,
				job.MerkleBranches,
				job.Version,
				job.NBits,
				job.NTime,
				cleanJobs,
			})
		}
	}
}

// getNextExtranonce1 returns a unique extranonce1 for each miner
func (s *StratumServer) getNextExtranonce1() string {
	s.extranonceMux.Lock()
	defer s.extranonceMux.Unlock()
	s.extranonce1++
	return fmt.Sprintf("%08x", s.extranonce1)
}

// HandleConnection handles a new miner connection with protocol detection
func (s *StratumServer) HandleConnection(conn net.Conn) {
	defer conn.Close()

	minerID := fmt.Sprintf("%s-%d", conn.RemoteAddr().String(), time.Now().UnixNano())
	// Defer connection logging until we know it's a real miner (not a probe)

	// Set read deadline for protocol detection
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// Peek first bytes to detect protocol
	peekBuf := make([]byte, 6)
	n, err := io.ReadFull(conn, peekBuf)
	if err != nil {
		// Don't log EOF errors - these are common probe connections from miners
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Printf("Failed to read from %s: %v", minerID, err)
		}
		return
	}

	// Reset deadline
	conn.SetReadDeadline(time.Time{})

	// Only log raw bytes for non-HTTP connections (reduce noise from metrics scrapers)
	if string(peekBuf[:3]) != "GET" && string(peekBuf[:4]) != "POST" {
		log.Printf("Raw bytes from %s: %v (ASCII: %q)", minerID, peekBuf[:n], string(peekBuf[:n]))
	}

	// Detect protocol based on first bytes
	switch {
	case peekBuf[0] == '{':
		// V1 JSON protocol - wrap connection to include peeked bytes
		wrappedConn := &peekableConn{Conn: conn, peeked: peekBuf[:n]}
		s.handleV1Connection(wrappedConn, minerID)

	case string(peekBuf[:3]) == "GET" || string(peekBuf[:4]) == "POST":
		// HTTP request - health check or metrics scrape (suppress verbose logging)
		s.handleHTTPProbe(conn, peekBuf[:n], minerID)

	case peekBuf[0] == 0x00 && peekBuf[1] == 0x00:
		// V2 binary protocol (starts with extension_type 0x0000)
		log.Printf("Detected Stratum V2 binary protocol from %s", minerID)
		s.handleV2Connection(conn, peekBuf[:n], minerID)

	default:
		// Unknown protocol - try to read more and log for analysis
		log.Printf("Unknown protocol from %s, first bytes: 0x%02x 0x%02x 0x%02x 0x%02x 0x%02x 0x%02x",
			minerID, peekBuf[0], peekBuf[1], peekBuf[2], peekBuf[3], peekBuf[4], peekBuf[5])
		// Read remaining data to understand the protocol
		s.handleUnknownProtocol(conn, peekBuf[:n], minerID)
	}
}

// peekableConn wraps a connection to prepend peeked bytes
type peekableConn struct {
	net.Conn
	peeked []byte
}

func (p *peekableConn) Read(b []byte) (int, error) {
	if len(p.peeked) > 0 {
		n := copy(b, p.peeked)
		p.peeked = p.peeked[n:]
		return n, nil
	}
	return p.Conn.Read(b)
}

// handleV1Connection handles Stratum V1 JSON protocol with improved timeout handling
func (s *StratumServer) handleV1Connection(conn net.Conn, minerID string) {
	// Get initial difficulty from vardiff manager
	initialDiff := s.vardiffManager.GetDifficulty(minerID)

	miner := &Miner{
		ID:         minerID,
		Address:    conn.RemoteAddr().String(),
		Conn:       conn,
		Authorized: false,
		Difficulty: initialDiff,
	}

	s.minersMutex.Lock()
	s.miners[minerID] = miner
	s.minersMutex.Unlock()

	// Start keepalive monitoring for this miner
	s.keepaliveManager.Start(minerID)

	defer func() {
		// Stop keepalive and cleanup vardiff on disconnect
		s.keepaliveManager.Stop(minerID)
		s.vardiffManager.RemoveMiner(minerID)
		s.minersMutex.Lock()
		delete(s.miners, minerID)
		s.minersMutex.Unlock()
		log.Printf("V1 Miner disconnected: %s", minerID)
	}()

	log.Printf("V1 miner connected: %s", minerID)

	// Configure scanner with larger buffer for high-throughput miners
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 64*1024), 64*1024) // 64KB buffer

	// Read timeout - 5 minutes of inactivity triggers disconnect
	readTimeout := 5 * time.Minute

	for {
		// Set read deadline for each message
		conn.SetReadDeadline(time.Now().Add(readTimeout))

		if !scanner.Scan() {
			break
		}

		line := scanner.Text()
		// Record activity for keepalive
		s.keepaliveManager.RecordActivity(minerID)

		if err := s.handleMessage(miner, line); err != nil {
			log.Printf("Error handling V1 message from %s: %v", minerID, err)
			return
		}
	}

	if err := scanner.Err(); err != nil {
		// Don't log timeout errors as errors - they're expected for idle miners
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			log.Printf("Connection timeout for %s (idle)", minerID)
		} else {
			log.Printf("Scanner error for %s: %v", minerID, err)
		}
	}
}

// handleV2Connection handles Stratum V2 binary protocol
func (s *StratumServer) handleV2Connection(conn net.Conn, initialBytes []byte, minerID string) {
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
		log.Printf("V2 Miner disconnected: %s", minerID)
	}()

	log.Printf("V2 miner connected: %s", minerID)

	// Parse the initial frame header from peeked bytes
	header, err := v2binary.ParseHeader(initialBytes)
	if err != nil {
		log.Printf("Failed to parse V2 header from %s: %v", minerID, err)
		return
	}

	log.Printf("V2 Header: ExtType=%d, MsgType=0x%02x, MsgLen=%d", header.ExtensionType, header.MsgType, header.MsgLength)

	// Read the rest of the first message payload
	payload := make([]byte, header.MsgLength)
	if header.MsgLength > 0 {
		_, err = io.ReadFull(conn, payload)
		if err != nil {
			log.Printf("Failed to read V2 payload from %s: %v", minerID, err)
			return
		}
	}

	// Handle SetupConnection message
	if header.MsgType == v2binary.MsgTypeSetupConnection {
		if err := s.handleV2SetupConnection(miner, payload); err != nil {
			log.Printf("Failed to handle SetupConnection from %s: %v", minerID, err)
			return
		}
	} else {
		log.Printf("Expected SetupConnection (0x00), got 0x%02x from %s", header.MsgType, minerID)
		return
	}

	// Continue reading V2 messages
	for {
		headerBuf := make([]byte, v2binary.HeaderSize)
		_, err := io.ReadFull(conn, headerBuf)
		if err != nil {
			if err != io.EOF {
				log.Printf("V2 read error from %s: %v", minerID, err)
			}
			return
		}

		header, err := v2binary.ParseHeader(headerBuf)
		if err != nil {
			log.Printf("V2 header parse error from %s: %v", minerID, err)
			return
		}

		payload := make([]byte, header.MsgLength)
		if header.MsgLength > 0 {
			_, err = io.ReadFull(conn, payload)
			if err != nil {
				log.Printf("V2 payload read error from %s: %v", minerID, err)
				return
			}
		}

		if err := s.handleV2Message(miner, header, payload); err != nil {
			log.Printf("V2 message error from %s: %v", minerID, err)
			return
		}
	}
}

// handleV2SetupConnection handles the V2 SetupConnection message
func (s *StratumServer) handleV2SetupConnection(miner *Miner, payload []byte) error {
	deser := v2binary.NewDeserializer(payload)
	setupConn, err := deser.DeserializeSetupConnection()
	if err != nil {
		return fmt.Errorf("deserialize SetupConnection: %w", err)
	}

	log.Printf("V2 SetupConnection from %s: Protocol=%d, Version=%d-%d, Vendor=%s, Hardware=%s, Device=%s",
		miner.ID, setupConn.Protocol, setupConn.MinVersion, setupConn.MaxVersion,
		setupConn.Vendor, setupConn.HardwareVersion, setupConn.DeviceID)

	// Send SetupConnectionSuccess
	ser := v2binary.NewSerializer()
	successMsg := &v2binary.SetupConnectionSuccess{
		UsedVersion: setupConn.MaxVersion, // Use max version supported by client
		Flags:       setupConn.Flags,      // Echo back flags
	}
	payloadBytes := ser.SerializeSetupConnectionSuccess(successMsg)
	frame := ser.SerializeFrame(v2binary.MsgTypeSetupConnectionSuccess, 0, payloadBytes)

	_, err = miner.Conn.Write(frame)
	if err != nil {
		return fmt.Errorf("send SetupConnectionSuccess: %w", err)
	}

	log.Printf("V2 SetupConnectionSuccess sent to %s", miner.ID)
	return nil
}

// handleV2Message handles V2 binary protocol messages after setup
func (s *StratumServer) handleV2Message(miner *Miner, header *v2binary.FrameHeader, payload []byte) error {
	log.Printf("V2 message from %s: MsgType=0x%02x, Len=%d", miner.ID, header.MsgType, header.MsgLength)

	switch header.MsgType {
	case v2binary.MsgTypeOpenStandardMiningChannel:
		return s.handleV2OpenChannel(miner, payload)
	case v2binary.MsgTypeSubmitSharesStandard:
		return s.handleV2SubmitShares(miner, payload)
	default:
		log.Printf("V2 unhandled message type 0x%02x from %s", header.MsgType, miner.ID)
		return nil
	}
}

// handleV2OpenChannel handles OpenStandardMiningChannel
func (s *StratumServer) handleV2OpenChannel(miner *Miner, payload []byte) error {
	deser := v2binary.NewDeserializer(payload)
	openChan, err := deser.DeserializeOpenStandardMiningChannel()
	if err != nil {
		return fmt.Errorf("deserialize OpenStandardMiningChannel: %w", err)
	}

	log.Printf("V2 OpenStandardMiningChannel from %s: RequestID=%d, User=%s, Hashrate=%.2f",
		miner.ID, openChan.RequestID, openChan.UserIdentity, openChan.NominalHashrate)

	// Mark miner as authorized
	miner.Authorized = true

	// Record miner in database
	_, err = s.db.Exec(
		"INSERT INTO miners (user_id, name, address, is_active) VALUES (1, $1, $2, true) ON CONFLICT DO NOTHING",
		string(openChan.UserIdentity), miner.Address,
	)
	if err != nil {
		log.Printf("Failed to record V2 miner: %v", err)
	}

	// Send OpenStandardMiningChannelSuccess
	ser := v2binary.NewSerializer()

	// Create target (max target = easy difficulty for testing)
	var target [32]byte
	for i := 0; i < 32; i++ {
		target[i] = 0xFF // Very easy target
	}

	successMsg := &v2binary.OpenStandardMiningChannelSuccess{
		RequestID:       openChan.RequestID,
		ChannelID:       1,      // Assign channel ID 1
		Target:          target, // Easy target for testing
		ExtraNonce2Size: 4,
		GroupChannelID:  0,
	}
	payloadBytes := ser.SerializeOpenStandardMiningChannelSuccess(successMsg)
	frame := ser.SerializeFrame(v2binary.MsgTypeOpenStandardMiningChannelSuccess, 0, payloadBytes)

	_, err = miner.Conn.Write(frame)
	if err != nil {
		return fmt.Errorf("send OpenStandardMiningChannelSuccess: %w", err)
	}

	log.Printf("V2 OpenStandardMiningChannelSuccess sent to %s (ChannelID=1)", miner.ID)

	// Send initial mining job
	return s.sendV2MiningJob(miner, 1)
}

// sendV2MiningJob sends a new mining job to a V2 miner
func (s *StratumServer) sendV2MiningJob(miner *Miner, channelID uint32) error {
	ser := v2binary.NewSerializer()

	jobID := uint32(time.Now().Unix())

	// Create NewMiningJob
	jobMsg := &v2binary.NewMiningJob{
		ChannelID:      channelID,
		JobID:          jobID,
		FuturePrevHash: false,
		Version:        0x20000000, // Standard version
		VersionMask:    0x1FFFE000, // Standard version rolling mask
	}
	payloadBytes := ser.SerializeNewMiningJob(jobMsg)
	frame := ser.SerializeFrame(v2binary.MsgTypeNewMiningJob, 0, payloadBytes)

	_, err := miner.Conn.Write(frame)
	if err != nil {
		return fmt.Errorf("send NewMiningJob: %w", err)
	}

	// Send SetNewPrevHash with a dummy prev hash (would come from node in production)
	var prevHash [32]byte
	// Fill with some data (in production this comes from the blockchain)
	binary.LittleEndian.PutUint64(prevHash[:], uint64(time.Now().UnixNano()))

	prevHashMsg := &v2binary.SetNewPrevHash{
		ChannelID: channelID,
		JobID:     jobID,
		PrevHash:  prevHash,
		MinNTime:  uint32(time.Now().Unix()),
		NBits:     0x1d00ffff, // Easy difficulty bits
	}
	payloadBytes = ser.SerializeSetNewPrevHash(prevHashMsg)
	frame = ser.SerializeFrame(v2binary.MsgTypeSetNewPrevHash, 0, payloadBytes)

	_, err = miner.Conn.Write(frame)
	if err != nil {
		return fmt.Errorf("send SetNewPrevHash: %w", err)
	}

	log.Printf("V2 NewMiningJob sent to %s (JobID=%d)", miner.ID, jobID)
	return nil
}

// handleV2SubmitShares handles V2 share submissions
func (s *StratumServer) handleV2SubmitShares(miner *Miner, payload []byte) error {
	deser := v2binary.NewDeserializer(payload)
	submit, err := deser.DeserializeSubmitSharesStandard()
	if err != nil {
		return fmt.Errorf("deserialize SubmitSharesStandard: %w", err)
	}

	log.Printf("V2 SubmitShares from %s: ChannelID=%d, JobID=%d, Nonce=0x%08x",
		miner.ID, submit.ChannelID, submit.JobID, submit.Nonce)

	// Accept the share (simplified validation)
	miner.SharesValid++
	miner.LastShare = time.Now()

	// Record share in database
	_, err = s.db.Exec(
		"INSERT INTO shares (miner_id, user_id, difficulty, is_valid, nonce, hash) VALUES (1, 1, $1, true, $2, $3)",
		miner.Difficulty, fmt.Sprintf("%08x", submit.Nonce), "v2share",
	)
	if err != nil {
		log.Printf("Failed to record V2 share: %v", err)
	}

	// Update Redis stats
	ctx := context.Background()
	s.redis.Incr(ctx, "pool:shares:valid")
	s.redis.HIncrBy(ctx, fmt.Sprintf("miner:%s", miner.ID), "shares", 1)

	// Send SubmitSharesSuccess
	ser := v2binary.NewSerializer()
	successMsg := &v2binary.SubmitSharesSuccess{
		ChannelID:       submit.ChannelID,
		LastSequenceNum: submit.SequenceNum,
		NewSubmits:      1,
		NewDifficulty:   0, // No difficulty change
	}
	payloadBytes := ser.SerializeSubmitSharesSuccess(successMsg)
	frame := ser.SerializeFrame(v2binary.MsgTypeSubmitSharesSuccess, 0, payloadBytes)

	_, err = miner.Conn.Write(frame)
	if err != nil {
		return fmt.Errorf("send SubmitSharesSuccess: %w", err)
	}

	log.Printf("V2 Share accepted from %s (total: %d)", miner.ID, miner.SharesValid)
	return nil
}

// handleHTTPProbe handles HTTP requests (health checks or HTTP stratum proxies)
func (s *StratumServer) handleHTTPProbe(conn net.Conn, initialBytes []byte, minerID string) {
	// Read the rest of the HTTP request (discard - we just need to drain it)
	buf := make([]byte, 4096)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	conn.Read(buf)

	// Silently handle HTTP requests (metrics, health checks) to reduce log noise
	// Send a simple HTTP response with pool info
	response := "HTTP/1.1 200 OK\r\n" +
		"Content-Type: application/json\r\n" +
		"Connection: close\r\n" +
		"\r\n" +
		`{"pool":"Chimeria Pool","stratum_port":3333,"protocols":["stratum-v1","stratum-v2"],"status":"ready"}` + "\r\n"

	conn.Write([]byte(response))
}

// handleUnknownProtocol tries to understand unknown protocols
func (s *StratumServer) handleUnknownProtocol(conn net.Conn, initialBytes []byte, minerID string) {
	// Read more data to understand the protocol
	buf := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Read(buf)

	var fullData []byte
	fullData = append(fullData, initialBytes...)
	if n > 0 {
		fullData = append(fullData, buf[:n]...)
	}

	log.Printf("Unknown protocol from %s - Total %d bytes:", minerID, len(fullData))
	log.Printf("  Hex: %x", fullData)
	log.Printf("  ASCII: %q", string(fullData))

	// Check if this looks like it could be a V1 message without the leading brace
	// Some miners send the JSON without proper framing
	if len(fullData) > 10 {
		// Look for JSON-like patterns
		dataStr := string(fullData)
		if strings.Contains(dataStr, "mining.") || strings.Contains(dataStr, "\"method\"") {
			log.Printf("Detected JSON-like content from %s, attempting V1 handling", minerID)
			// Try to find the start of JSON
			for i, b := range fullData {
				if b == '{' {
					log.Printf("Found JSON start at offset %d", i)
					wrappedConn := &peekableConn{Conn: conn, peeked: fullData[i:]}
					s.handleV1Connection(wrappedConn, minerID)
					return
				}
			}
		}
	}

	if err != nil && err != io.EOF {
		log.Printf("Error reading from %s: %v", minerID, err)
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
	case "mining.get_transactions":
		// Return empty transactions list - miner may request this for block validation
		return s.sendResponse(miner, req.ID, []string{}, nil)
	case "mining.configure":
		// Return empty config - some miners request this
		return s.sendResponse(miner, req.ID, map[string]interface{}{}, nil)
	default:
		log.Printf("Unknown method from %s: %s", miner.ID, req.Method)
		return s.sendResponse(miner, req.ID, nil, "Unknown method")
	}
}

// handleSubscribe handles mining.subscribe
func (s *StratumServer) handleSubscribe(miner *Miner, req StratumRequest) error {
	// Generate subscription details
	subscriptionID := fmt.Sprintf("%x", time.Now().UnixNano())
	extranonce1 := s.getNextExtranonce1()
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
	if err := s.sendNotification(miner, "mining.set_difficulty", []interface{}{miner.Difficulty}); err != nil {
		return err
	}

	// Send initial mining job from current block template
	s.jobMutex.RLock()
	job := s.currentJob
	s.jobMutex.RUnlock()

	if job == nil {
		// Fallback to placeholder if no template yet
		log.Printf("Warning: No block template available for %s, using placeholder", miner.ID)
		return s.sendNotification(miner, "mining.notify", []interface{}{
			fmt.Sprintf("%x", time.Now().Unix()),
			"0000000000000000000000000000000000000000000000000000000000000000",
			"01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff",
			"ffffffff0100000000000000000000000000",
			[]string{},
			"20000000",
			"1d00ffff",
			fmt.Sprintf("%08x", time.Now().Unix()),
			true,
		})
	}

	log.Printf("Sending real Litecoin work to %s: height=%d, bits=%s", miner.ID, job.Height, job.NBits)
	return s.sendNotification(miner, "mining.notify", []interface{}{
		job.JobID,
		job.PrevHash,
		job.Coinbase1,
		job.Coinbase2,
		job.MerkleBranches,
		job.Version,
		job.NBits,
		job.NTime,
		true,
	})
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

	// Trim whitespace from username (some miners send leading/trailing spaces)
	username = strings.TrimSpace(username)

	// Look up user by username OR email in database
	var userID int64
	var actualUsername string
	err := s.db.QueryRow(
		"SELECT id, username FROM users WHERE (username = $1 OR email = $1) AND is_active = true",
		username,
	).Scan(&userID, &actualUsername)

	if err == sql.ErrNoRows {
		log.Printf("Authorization failed for %s: user '%s' not found (checked username and email)", miner.ID, username)
		return s.sendResponse(miner, req.ID, false, "User not found. Please register at the pool website.")
	} else if err != nil {
		log.Printf("Database error during authorization for %s: %v", miner.ID, err)
		return s.sendResponse(miner, req.ID, false, "Authorization failed - database error")
	}

	// Set miner authorization with proper user tracking (use actual username from DB)
	miner.Authorized = true
	miner.UserID = userID
	miner.Username = actualUsername
	log.Printf("Miner %s authorized as %s (user_id: %d, login: %s)", miner.ID, actualUsername, userID, username)

	// Record or update miner in database with correct user_id
	// First check if miner exists, then insert or update
	var existingMinerID int64
	err = s.db.QueryRow(
		"SELECT id FROM miners WHERE user_id = $1 AND name = $2 LIMIT 1",
		userID, actualUsername,
	).Scan(&existingMinerID)

	if err == sql.ErrNoRows {
		// Create new miner record - strip port from address for inet type
		ipOnly := miner.Address
		if host, _, err := net.SplitHostPort(miner.Address); err == nil {
			ipOnly = host
		}
		_, err = s.db.Exec(
			"INSERT INTO miners (user_id, name, address, is_active) VALUES ($1, $2, $3, true)",
			userID, actualUsername, ipOnly,
		)
		if err != nil {
			log.Printf("Failed to create miner for user %d: %v", userID, err)
		}
	} else if err == nil {
		// Update existing miner - strip port from address for inet type
		ipOnly := miner.Address
		if host, _, err := net.SplitHostPort(miner.Address); err == nil {
			ipOnly = host
		}
		_, err = s.db.Exec(
			"UPDATE miners SET address = $1, is_active = true, updated_at = NOW() WHERE id = $2",
			ipOnly, existingMinerID,
		)
		if err != nil {
			log.Printf("Failed to update miner %d: %v", existingMinerID, err)
		}
	} else {
		log.Printf("Failed to check miner for user %d: %v", userID, err)
	}

	// Geolocate miner IP in background (don't block authorization)
	go func() {
		if s.geoService != nil {
			if err := s.geoService.UpdateMinerLocationByUserAndName(userID, actualUsername, miner.Address); err != nil {
				log.Printf("⚠️ Failed to geolocate miner %s: %v", miner.ID, err)
			}
		}
	}()

	return s.sendResponse(miner, req.ID, true, nil)
}

// handleSubmit handles mining.submit (share submission)
func (s *StratumServer) handleSubmit(miner *Miner, req StratumRequest) error {
	if !miner.Authorized {
		return s.sendResponse(miner, req.ID, false, "Not authorized")
	}

	if miner.UserID == 0 {
		log.Printf("Share rejected from %s: no user_id set", miner.ID)
		return s.sendResponse(miner, req.ID, false, "Authorization error - please reconnect")
	}

	// Calculate time since last share for vardiff
	now := time.Now()
	shareTime := time.Since(miner.LastShare)
	if miner.LastShare.IsZero() {
		shareTime = 10 * time.Second // Default for first share
	}

	// Record share for vardiff adjustment
	s.vardiffManager.RecordShare(miner.ID, shareTime)

	// Validate share (simplified - in production would verify against blockchain)
	miner.SharesValid++
	miner.LastShare = now

	// Check if difficulty needs adjustment
	newDiff := s.vardiffManager.GetDifficulty(miner.ID)
	if newDiff != miner.Difficulty {
		miner.Difficulty = newDiff
		// Send new difficulty to miner
		if err := s.sendNotification(miner, "mining.set_difficulty", []interface{}{newDiff}); err != nil {
			log.Printf("Failed to send difficulty update to %s: %v", miner.ID, err)
		} else {
			log.Printf("Adjusted difficulty for %s: %.6f (share time: %v)", miner.ID, newDiff, shareTime)
		}
	}

	// Get or create miner record for this user
	var minerDBID int64
	err := s.db.QueryRow(`
		SELECT id FROM miners 
		WHERE user_id = $1 AND name = $2 
		LIMIT 1`,
		miner.UserID, miner.Username,
	).Scan(&minerDBID)

	if err == sql.ErrNoRows {
		// Create miner record if it doesn't exist
		err = s.db.QueryRow(`
			INSERT INTO miners (user_id, name, address, is_active) 
			VALUES ($1, $2, $3, true) 
			RETURNING id`,
			miner.UserID, miner.Username, miner.Address,
		).Scan(&minerDBID)
		if err != nil {
			log.Printf("Failed to create miner record for user %d: %v", miner.UserID, err)
			minerDBID = 1 // Fallback to prevent share loss
		}
	} else if err != nil {
		log.Printf("Failed to lookup miner for user %d: %v", miner.UserID, err)
		minerDBID = 1 // Fallback to prevent share loss
	}

	// Record share in database with correct user_id and miner_id
	_, err = s.db.Exec(`
		INSERT INTO shares (miner_id, user_id, difficulty, is_valid, nonce, hash, timestamp) 
		VALUES ($1, $2, $3, true, $4, $5, NOW())`,
		minerDBID, miner.UserID, miner.Difficulty, "submitted", "hash",
	)
	if err != nil {
		log.Printf("Failed to record share for user %d: %v", miner.UserID, err)
	}

	// Track share for hashrate calculation
	s.hashrateMux.Lock()
	if _, exists := s.hashrateWindows[miner.ID]; !exists {
		s.hashrateWindows[miner.ID] = hashrate.NewWindow(5 * time.Minute)
	}
	s.hashrateWindows[miner.ID].AddShare(miner.Difficulty, time.Now())
	currentHashrate := s.hashrateWindows[miner.ID].GetHashrate()
	s.hashrateMux.Unlock()

	// Update miner hashrate in database (every 10 shares to reduce DB load)
	if miner.SharesValid%10 == 0 {
		_, err = s.db.Exec(`
			UPDATE miners SET hashrate = $1, last_seen = NOW() 
			WHERE user_id = $2 AND name = $3`,
			currentHashrate, miner.UserID, miner.Username,
		)
		if err != nil {
			log.Printf("Failed to update hashrate for miner %s: %v", miner.Username, err)
		}
	}

	// Update Redis stats with user-specific tracking
	ctx := context.Background()
	s.redis.Incr(ctx, "pool:shares:valid")
	s.redis.HIncrBy(ctx, fmt.Sprintf("miner:%s", miner.ID), "shares", 1)
	s.redis.HIncrBy(ctx, fmt.Sprintf("user:%d:shares", miner.UserID), "valid", 1)
	s.redis.HSet(ctx, fmt.Sprintf("miner:%s", miner.ID), "hashrate", currentHashrate)

	log.Printf("Share accepted from %s (user: %s, user_id: %d, total: %d, hashrate: %s)",
		miner.ID, miner.Username, miner.UserID, miner.SharesValid, s.hashrateCalc.Format(currentHashrate))
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
		log.Printf("Error marshaling notification %s for %s: %v", method, miner.ID, err)
		return err
	}

	log.Printf("Sending %s to %s: %s", method, miner.ID, string(data))

	n, err := miner.Conn.Write(append(data, '\n'))
	if err != nil {
		log.Printf("Error sending %s to %s: %v", method, miner.ID, err)
		return err
	}
	log.Printf("Sent %d bytes for %s to %s", n, method, miner.ID)
	return nil
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

// GetPoolMetrics implements health.PoolMetricsProvider for Prometheus export
func (s *StratumServer) GetPoolMetrics() *health.PoolMetrics {
	s.minersMutex.RLock()
	defer s.minersMutex.RUnlock()

	metrics := &health.PoolMetrics{}

	// Count online workers and calculate total hashrate
	var totalShares int64
	for _, miner := range s.miners {
		if miner.Authorized {
			metrics.WorkersOnline++
			totalShares += miner.SharesValid

			// Calculate hashrate from hashrate windows if available
			s.hashrateMux.RLock()
			if window, exists := s.hashrateWindows[miner.ID]; exists {
				metrics.TotalHashrate += window.GetHashrate()
			}
			s.hashrateMux.RUnlock()
		}
	}

	metrics.SharesAccepted = totalShares

	// Get current job info for block height
	s.jobMutex.RLock()
	if s.currentJob != nil {
		metrics.BlockHeight = s.currentJob.Height
	}
	s.jobMutex.RUnlock()

	return metrics
}

// =============================================================================
// NETWORK WATCHDOG - Auto-recovery on internet restoration
// =============================================================================

var (
	networkWatchdog      *recovery.DefaultNetworkWatchdog
	recoveryOrchestrator *recovery.DefaultRecoveryOrchestrator
)

// initNetworkWatchdog initializes the network watchdog and recovery orchestrator
// This addresses the gap where "Max restarts reached" stops recovery attempts
// When network is restored, counters are reset and services are automatically restarted
func initNetworkWatchdog() {
	log.Println("🌐 Initializing Network Watchdog for automatic recovery...")

	// Create network watchdog
	watchdogConfig := recovery.DefaultNetworkWatchdogConfig()
	watchdogConfig.CheckInterval = 10 * time.Second // Check every 10 seconds
	networkWatchdog = recovery.NewNetworkWatchdog(watchdogConfig)

	// Create recovery orchestrator
	orchestratorConfig := recovery.DefaultRecoveryOrchestratorConfig()
	orchestratorConfig.ResetCountersOnNetworkRestore = true // KEY: Reset restart counters!
	orchestratorConfig.EnableAutoRecovery = true
	recoveryOrchestrator = recovery.NewRecoveryOrchestrator(orchestratorConfig, nil)

	// Register services in recovery order (priority: lower = starts first)
	registerRecoverableServices()

	// Register orchestrator as network state observer
	networkWatchdog.RegisterObserver(recoveryOrchestrator)

	// Start the watchdog
	ctx := context.Background()
	if err := networkWatchdog.Start(ctx); err != nil {
		log.Printf("⚠️ Failed to start network watchdog: %v", err)
		return
	}

	if err := recoveryOrchestrator.Start(ctx); err != nil {
		log.Printf("⚠️ Failed to start recovery orchestrator: %v", err)
		return
	}

	log.Println("✅ Network Watchdog started - will auto-recover services on internet restoration")
}

// registerRecoverableServices registers all Docker services for recovery
func registerRecoverableServices() {
	services := []struct {
		name      string
		container string
		priority  int
	}{
		{"redis", "docker-redis-1", 1},
		{"postgres", "docker-postgres-1", 2},
		{"litecoind", "docker-litecoind-1", 3},
		{"chimera-pool-api", "docker-chimera-pool-api-1", 4},
		{"chimera-pool-stratum", "docker-chimera-pool-stratum-1", 5},
		{"chimera-pool-web", "docker-chimera-pool-web-1", 6},
		{"nginx", "docker-nginx-1", 7},
	}

	for _, svc := range services {
		dockerService := recovery.NewDockerService(svc.name, svc.container, nil)
		if err := recoveryOrchestrator.RegisterService(dockerService, svc.priority); err != nil {
			log.Printf("⚠️ Failed to register service %s: %v", svc.name, err)
		}
	}
}

// stopNetworkWatchdog gracefully stops the network watchdog
func stopNetworkWatchdog() {
	ctx := context.Background()
	if networkWatchdog != nil {
		networkWatchdog.Stop(ctx)
	}
	if recoveryOrchestrator != nil {
		recoveryOrchestrator.Stop(ctx)
	}
	log.Println("🌐 Network Watchdog stopped")
}
