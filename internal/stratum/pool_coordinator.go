package stratum

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chimera-pool/chimera-pool-core/internal/shares"
	"github.com/chimera-pool/chimera-pool-core/internal/stratum/difficulty"
	"github.com/google/uuid"
)

// =============================================================================
// POOL COORDINATOR
// High-performance integration layer connecting stratum, shares, and vardiff
// Designed for 100k+ concurrent miners with sub-millisecond latency
// =============================================================================

// PoolCoordinatorConfig configures the pool coordinator
type PoolCoordinatorConfig struct {
	// Network settings
	ListenAddress  string
	MaxConnections int

	// Share processing
	ShareWorkers   int
	ShareQueueSize int
	ShareBatchSize int

	// Vardiff settings
	TargetShareTime time.Duration
	RetargetTime    time.Duration
	MinShares       int

	// Job settings
	JobUpdateInterval time.Duration

	// Timeouts
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	IdleTimeout      time.Duration
	HandshakeTimeout time.Duration
}

// DefaultPoolCoordinatorConfig returns production defaults
func DefaultPoolCoordinatorConfig() PoolCoordinatorConfig {
	return PoolCoordinatorConfig{
		ListenAddress:     ":3333",
		MaxConnections:    100000,
		ShareWorkers:      8,
		ShareQueueSize:    100000,
		ShareBatchSize:    100,
		TargetShareTime:   10 * time.Second,
		RetargetTime:      90 * time.Second,
		MinShares:         3,
		JobUpdateInterval: 30 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       5 * time.Minute,
		HandshakeTimeout:  30 * time.Second,
	}
}

// PoolCoordinator orchestrates all pool components
type PoolCoordinator struct {
	config PoolCoordinatorConfig

	// Core components
	connManager    *ConnectionManager
	shareProcessor *shares.BatchProcessor
	vardiffManager *difficulty.VardiffManager
	authenticator  MinerAuthenticator

	// Job management
	currentJob   atomic.Value // *Job
	jobMutex     sync.RWMutex
	jobListeners []chan *Job

	// Network
	listener net.Listener

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Statistics (atomic)
	stats PoolStats
}

// PoolStats tracks pool-wide statistics
type PoolStats struct {
	// Connection stats
	TotalConnections int64
	ActiveMiners     int64
	AuthorizedMiners int64

	// Share stats
	TotalSharesReceived int64
	TotalSharesAccepted int64
	TotalSharesRejected int64
	TotalSharesStale    int64

	// Performance stats
	CurrentHashrate int64 // H/s (atomic for lock-free read)
	BlocksFound     int64
	LastBlockTime   int64 // Unix timestamp

	// Timing stats
	AvgShareLatencyNs int64
	MaxShareLatencyNs int64
}

// NewPoolCoordinator creates a new pool coordinator
func NewPoolCoordinator(config PoolCoordinatorConfig) *PoolCoordinator {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize connection manager
	connConfig := ConnectionManagerConfig{
		ShardCount:          64,
		MaxConnectionsPerIP: 100,
		MaxTotalConnections: config.MaxConnections,
		IdleTimeout:         config.IdleTimeout,
		HandshakeTimeout:    config.HandshakeTimeout,
	}

	// Initialize share processor
	shareConfig := shares.BatchConfig{
		WorkerCount:  config.ShareWorkers,
		QueueSize:    config.ShareQueueSize,
		BatchSize:    config.ShareBatchSize,
		BatchTimeout: 10 * time.Millisecond,
	}

	// Initialize vardiff manager
	vardiffManager := difficulty.NewVardiffManagerWithParams(
		config.TargetShareTime,
		config.RetargetTime,
		config.MinShares,
	)

	pc := &PoolCoordinator{
		config:         config,
		connManager:    NewConnectionManager(connConfig),
		shareProcessor: shares.NewBatchProcessor(shareConfig),
		vardiffManager: vardiffManager,
		jobListeners:   make([]chan *Job, 0),
		ctx:            ctx,
		cancel:         cancel,
	}

	// Set connection callbacks
	pc.connManager.config.OnConnect = pc.onMinerConnect
	pc.connManager.config.OnDisconnect = pc.onMinerDisconnect

	return pc
}

// Start begins the pool coordinator
func (pc *PoolCoordinator) Start() error {
	// Start share processor
	pc.shareProcessor.Start()

	// Start connection manager
	pc.connManager.Start()

	// Start listener
	listener, err := net.Listen("tcp", pc.config.ListenAddress)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", pc.config.ListenAddress, err)
	}
	pc.listener = listener

	// Start accept loop
	pc.wg.Add(1)
	go pc.acceptLoop()

	// Start job update loop
	pc.wg.Add(1)
	go pc.jobUpdateLoop()

	// Start stats aggregation loop
	pc.wg.Add(1)
	go pc.statsLoop()

	return nil
}

// Stop gracefully shuts down the coordinator
func (pc *PoolCoordinator) Stop() error {
	pc.cancel()

	if pc.listener != nil {
		pc.listener.Close()
	}

	pc.connManager.Stop()
	pc.shareProcessor.Stop()

	pc.wg.Wait()
	return nil
}

// SetAuthenticator configures the miner authenticator
// Must be called before Start() for production use
func (pc *PoolCoordinator) SetAuthenticator(auth MinerAuthenticator) {
	pc.authenticator = auth
}

// GetStats returns current pool statistics (lock-free)
func (pc *PoolCoordinator) GetStats() PoolStats {
	return PoolStats{
		TotalConnections:    atomic.LoadInt64(&pc.stats.TotalConnections),
		ActiveMiners:        atomic.LoadInt64(&pc.stats.ActiveMiners),
		AuthorizedMiners:    atomic.LoadInt64(&pc.stats.AuthorizedMiners),
		TotalSharesReceived: atomic.LoadInt64(&pc.stats.TotalSharesReceived),
		TotalSharesAccepted: atomic.LoadInt64(&pc.stats.TotalSharesAccepted),
		TotalSharesRejected: atomic.LoadInt64(&pc.stats.TotalSharesRejected),
		TotalSharesStale:    atomic.LoadInt64(&pc.stats.TotalSharesStale),
		CurrentHashrate:     atomic.LoadInt64(&pc.stats.CurrentHashrate),
		BlocksFound:         atomic.LoadInt64(&pc.stats.BlocksFound),
		LastBlockTime:       atomic.LoadInt64(&pc.stats.LastBlockTime),
		AvgShareLatencyNs:   atomic.LoadInt64(&pc.stats.AvgShareLatencyNs),
		MaxShareLatencyNs:   atomic.LoadInt64(&pc.stats.MaxShareLatencyNs),
	}
}

// SetCurrentJob updates the current mining job and broadcasts to all miners
func (pc *PoolCoordinator) SetCurrentJob(job *Job) {
	pc.currentJob.Store(job)

	// Broadcast job notification to all authorized miners
	notify := pc.createJobNotification(job)
	pc.connManager.BroadcastToAuthorized(notify)

	// Notify listeners
	for _, ch := range pc.jobListeners {
		select {
		case ch <- job:
		default:
		}
	}
}

// GetCurrentJob returns the current mining job
func (pc *PoolCoordinator) GetCurrentJob() *Job {
	if job := pc.currentJob.Load(); job != nil {
		return job.(*Job)
	}
	return nil
}

// SubscribeToJobs returns a channel that receives new jobs
func (pc *PoolCoordinator) SubscribeToJobs() <-chan *Job {
	ch := make(chan *Job, 10)
	pc.jobMutex.Lock()
	pc.jobListeners = append(pc.jobListeners, ch)
	pc.jobMutex.Unlock()
	return ch
}

// Internal methods

func (pc *PoolCoordinator) acceptLoop() {
	defer pc.wg.Done()

	for {
		select {
		case <-pc.ctx.Done():
			return
		default:
		}

		// Set accept deadline to allow checking context
		pc.listener.(*net.TCPListener).SetDeadline(time.Now().Add(time.Second))

		conn, err := pc.listener.Accept()
		if err != nil {
			if pc.ctx.Err() != nil {
				return
			}
			continue
		}

		// Handle connection in goroutine
		pc.wg.Add(1)
		go pc.handleConnection(conn)
	}
}

func (pc *PoolCoordinator) handleConnection(conn net.Conn) {
	defer pc.wg.Done()
	defer conn.Close()

	// Extract remote IP
	remoteAddr := conn.RemoteAddr().String()
	host, _, _ := net.SplitHostPort(remoteAddr)

	// Create managed connection
	ctx, cancel := context.WithCancel(pc.ctx)
	managedConn := &ManagedConnection{
		ID:           uuid.New().String(),
		Conn:         conn,
		RemoteIP:     host,
		ConnectedAt:  time.Now(),
		LastActivity: time.Now(),
		SendChan:     make(chan []byte, 100),
		ctx:          ctx,
		cancel:       cancel,
	}

	// Register with connection manager
	if err := pc.connManager.AddConnection(managedConn); err != nil {
		return
	}
	defer pc.connManager.RemoveConnection(managedConn.ID, "connection closed")

	// Start sender goroutine
	pc.wg.Add(1)
	go pc.connectionSender(managedConn)

	// Process messages
	pc.processMessages(managedConn)
}

func (pc *PoolCoordinator) connectionSender(conn *ManagedConnection) {
	defer pc.wg.Done()

	for {
		select {
		case <-conn.ctx.Done():
			return
		case msg := <-conn.SendChan:
			conn.Conn.SetWriteDeadline(time.Now().Add(pc.config.WriteTimeout))
			if _, err := conn.Conn.Write(append(msg, '\n')); err != nil {
				return
			}
			atomic.AddInt64(&conn.BytesSent, int64(len(msg)+1))
		}
	}
}

func (pc *PoolCoordinator) processMessages(conn *ManagedConnection) {
	buffer := make([]byte, 4096)
	var messageBuffer []byte

	for {
		select {
		case <-conn.ctx.Done():
			return
		default:
		}

		conn.Conn.SetReadDeadline(time.Now().Add(pc.config.ReadTimeout))
		n, err := conn.Conn.Read(buffer)
		if err != nil {
			return
		}

		atomic.AddInt64(&conn.BytesReceived, int64(n))
		messageBuffer = append(messageBuffer, buffer[:n]...)

		// Process complete messages (newline delimited)
		for {
			idx := -1
			for i, b := range messageBuffer {
				if b == '\n' {
					idx = i
					break
				}
			}
			if idx == -1 {
				break
			}

			message := messageBuffer[:idx]
			messageBuffer = messageBuffer[idx+1:]

			if len(message) > 0 {
				conn.LastActivity = time.Now()
				pc.handleMessage(conn, message)
			}
		}
	}
}

func (pc *PoolCoordinator) handleMessage(conn *ManagedConnection, data []byte) {
	var msg struct {
		ID     interface{}   `json:"id"`
		Method string        `json:"method"`
		Params []interface{} `json:"params"`
	}

	if err := json.Unmarshal(data, &msg); err != nil {
		pc.sendError(conn, 0, 20, "Parse error")
		return
	}

	switch msg.Method {
	case "mining.subscribe":
		pc.handleSubscribe(conn, msg.ID, msg.Params)
	case "mining.authorize":
		pc.handleAuthorize(conn, msg.ID, msg.Params)
	case "mining.submit":
		pc.handleSubmit(conn, msg.ID, msg.Params)
	case "mining.extranonce.subscribe":
		pc.handleExtranonceSubscribe(conn, msg.ID)
	default:
		pc.sendError(conn, msg.ID, 20, "Unknown method")
	}
}

func (pc *PoolCoordinator) handleSubscribe(conn *ManagedConnection, id interface{}, params []interface{}) {
	conn.Subscribed = true
	conn.Extranonce1 = generateExtranonce1()

	// Parse user agent if provided
	userAgent := ""
	if len(params) > 0 {
		if ua, ok := params[0].(string); ok {
			userAgent = ua
		}
	}

	// Register with vardiff using user agent for hardware classification
	pc.vardiffManager.RegisterMiner(conn.ID, userAgent, 0)

	// Send subscribe response
	response := map[string]interface{}{
		"id": id,
		"result": []interface{}{
			[][]string{
				{"mining.set_difficulty", conn.ID},
				{"mining.notify", conn.ID},
			},
			conn.Extranonce1,
			4, // extranonce2 size
		},
		"error": nil,
	}

	pc.sendJSON(conn, response)

	// Send initial difficulty
	initialDiff := pc.vardiffManager.GetDifficulty(conn.ID)
	conn.Difficulty = initialDiff
	pc.sendDifficulty(conn, initialDiff)

	// Send current job if available
	if job := pc.GetCurrentJob(); job != nil {
		pc.sendJob(conn, job)
	}
}

func (pc *PoolCoordinator) handleAuthorize(conn *ManagedConnection, id interface{}, params []interface{}) {
	if len(params) < 1 {
		pc.sendError(conn, id, 24, "Missing worker name")
		return
	}

	workerName, ok := params[0].(string)
	if !ok {
		pc.sendError(conn, id, 24, "Invalid worker name")
		return
	}

	// Get password (optional in stratum, often ignored)
	password := ""
	if len(params) > 1 {
		if pw, ok := params[1].(string); ok {
			password = pw
		}
	}

	// Authenticate using the authenticator interface
	if pc.authenticator != nil {
		result, err := pc.authenticator.Authenticate(pc.ctx, workerName, password)
		if err != nil {
			pc.sendError(conn, id, 24, "Authorization failed: "+err.Error())
			return
		}

		// Store authentication result in connection
		conn.WorkerName = workerName
		conn.UserID = result.UserID
		conn.MinerID = result.MinerID
		conn.Authorized = true
	} else {
		// Fallback: accept all valid-looking worker names (dev mode)
		conn.WorkerName = workerName
		conn.Authorized = true
	}

	atomic.AddInt64(&pc.stats.AuthorizedMiners, 1)

	response := map[string]interface{}{
		"id":     id,
		"result": true,
		"error":  nil,
	}
	pc.sendJSON(conn, response)
}

func (pc *PoolCoordinator) handleSubmit(conn *ManagedConnection, id interface{}, params []interface{}) {
	startTime := time.Now()
	atomic.AddInt64(&pc.stats.TotalSharesReceived, 1)

	if !conn.Authorized {
		pc.sendError(conn, id, 24, "Unauthorized")
		atomic.AddInt64(&pc.stats.TotalSharesRejected, 1)
		return
	}

	if len(params) < 5 {
		pc.sendError(conn, id, 20, "Invalid params")
		atomic.AddInt64(&pc.stats.TotalSharesRejected, 1)
		return
	}

	// Parse share parameters
	workerName, _ := params[0].(string)
	jobID, _ := params[1].(string)
	extranonce2, _ := params[2].(string)
	ntime, _ := params[3].(string)
	nonce, _ := params[4].(string)

	// Create share for processing with actual user/miner IDs from connection
	share := &shares.Share{
		MinerID:    conn.MinerID,
		UserID:     conn.UserID,
		JobID:      jobID,
		Nonce:      nonce,
		Difficulty: float64(conn.Difficulty),
		Timestamp:  time.Now(),
		WorkerName: workerName,
		ExtraNonce: extranonce2,
		NTime:      ntime,
	}

	// Submit to batch processor
	resultCh := pc.shareProcessor.Submit(share)

	// Wait for result with timeout
	select {
	case result := <-resultCh:
		latency := time.Since(startTime).Nanoseconds()
		pc.updateShareLatency(latency)

		if result.Success {
			atomic.AddInt64(&pc.stats.TotalSharesAccepted, 1)
			atomic.AddInt64(&conn.SharesAccepted, 1)

			// Record share with vardiff
			newDiff, changed := pc.vardiffManager.RecordShare(conn.ID, true, false)
			if changed {
				conn.Difficulty = newDiff
				pc.sendDifficulty(conn, newDiff)
			}

			response := map[string]interface{}{
				"id":     id,
				"result": true,
				"error":  nil,
			}
			pc.sendJSON(conn, response)
		} else {
			atomic.AddInt64(&pc.stats.TotalSharesRejected, 1)
			atomic.AddInt64(&conn.SharesRejected, 1)
			pc.sendError(conn, id, 23, "Low difficulty share")
		}

	case <-time.After(5 * time.Second):
		atomic.AddInt64(&pc.stats.TotalSharesRejected, 1)
		pc.sendError(conn, id, 20, "Share processing timeout")
	}

	atomic.AddInt64(&conn.SharesSubmitted, 1)
}

func (pc *PoolCoordinator) handleExtranonceSubscribe(conn *ManagedConnection, id interface{}) {
	response := map[string]interface{}{
		"id":     id,
		"result": true,
		"error":  nil,
	}
	pc.sendJSON(conn, response)
}

func (pc *PoolCoordinator) sendDifficulty(conn *ManagedConnection, diff uint64) {
	msg := map[string]interface{}{
		"id":     nil,
		"method": "mining.set_difficulty",
		"params": []interface{}{float64(diff)},
	}
	pc.sendJSON(conn, msg)
}

func (pc *PoolCoordinator) sendJob(conn *ManagedConnection, job *Job) {
	msg := map[string]interface{}{
		"id":     nil,
		"method": "mining.notify",
		"params": []interface{}{
			job.ID,
			fmt.Sprintf("%x", job.PrevHash),
			fmt.Sprintf("%x", job.Coinbase1),
			fmt.Sprintf("%x", job.Coinbase2),
			pc.merkleToStrings(job.MerkleBranch),
			fmt.Sprintf("%08x", job.Version),
			fmt.Sprintf("%08x", job.NBits),
			fmt.Sprintf("%08x", job.NTime),
			job.CleanJobs,
		},
	}
	pc.sendJSON(conn, msg)
}

func (pc *PoolCoordinator) sendError(conn *ManagedConnection, id interface{}, code int, message string) {
	response := map[string]interface{}{
		"id":     id,
		"result": nil,
		"error":  []interface{}{code, message, nil},
	}
	pc.sendJSON(conn, response)
}

func (pc *PoolCoordinator) sendJSON(conn *ManagedConnection, msg interface{}) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	select {
	case conn.SendChan <- data:
	default:
		// Channel full - drop message
	}
}

func (pc *PoolCoordinator) createJobNotification(job *Job) []byte {
	msg := map[string]interface{}{
		"id":     nil,
		"method": "mining.notify",
		"params": []interface{}{
			job.ID,
			fmt.Sprintf("%x", job.PrevHash),
			fmt.Sprintf("%x", job.Coinbase1),
			fmt.Sprintf("%x", job.Coinbase2),
			pc.merkleToStrings(job.MerkleBranch),
			fmt.Sprintf("%08x", job.Version),
			fmt.Sprintf("%08x", job.NBits),
			fmt.Sprintf("%08x", job.NTime),
			job.CleanJobs,
		},
	}
	data, _ := json.Marshal(msg)
	return data
}

func (pc *PoolCoordinator) merkleToStrings(merkle [][]byte) []string {
	result := make([]string, len(merkle))
	for i, m := range merkle {
		result[i] = fmt.Sprintf("%x", m)
	}
	return result
}

func (pc *PoolCoordinator) updateShareLatency(latencyNs int64) {
	// Update max latency
	for {
		current := atomic.LoadInt64(&pc.stats.MaxShareLatencyNs)
		if latencyNs <= current || atomic.CompareAndSwapInt64(&pc.stats.MaxShareLatencyNs, current, latencyNs) {
			break
		}
	}

	// Update average (exponential moving average)
	current := atomic.LoadInt64(&pc.stats.AvgShareLatencyNs)
	newAvg := (current*9 + latencyNs) / 10
	atomic.StoreInt64(&pc.stats.AvgShareLatencyNs, newAvg)
}

func (pc *PoolCoordinator) onMinerConnect(conn *ManagedConnection) {
	atomic.AddInt64(&pc.stats.TotalConnections, 1)
	atomic.AddInt64(&pc.stats.ActiveMiners, 1)
}

func (pc *PoolCoordinator) onMinerDisconnect(conn *ManagedConnection, reason string) {
	atomic.AddInt64(&pc.stats.ActiveMiners, -1)
	if conn.Authorized {
		atomic.AddInt64(&pc.stats.AuthorizedMiners, -1)
	}
	pc.vardiffManager.RemoveMiner(conn.ID)
}

func (pc *PoolCoordinator) jobUpdateLoop() {
	defer pc.wg.Done()

	ticker := time.NewTicker(pc.config.JobUpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-pc.ctx.Done():
			return
		case <-ticker.C:
			// TODO: Fetch new job from block template provider
			// For now, just update the timestamp on the current job
		}
	}
}

func (pc *PoolCoordinator) statsLoop() {
	defer pc.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-pc.ctx.Done():
			return
		case <-ticker.C:
			// Update hashrate from vardiff manager
			hashrate := pc.vardiffManager.GetPoolHashrate()
			atomic.StoreInt64(&pc.stats.CurrentHashrate, int64(hashrate))
		}
	}
}
