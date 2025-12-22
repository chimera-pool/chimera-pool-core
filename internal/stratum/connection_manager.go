package stratum

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// =============================================================================
// SHARDED CONNECTION MANAGER
// Designed for 100k+ concurrent connections with minimal lock contention
// Uses consistent hashing to distribute connections across shards
// =============================================================================

const (
	// Default number of shards (should be power of 2 for fast modulo)
	DefaultShardCount = 64

	// Connection limits
	DefaultMaxConnectionsPerIP = 100
	DefaultMaxTotalConnections = 100000

	// Timeouts
	DefaultIdleTimeout      = 5 * time.Minute
	DefaultHandshakeTimeout = 30 * time.Second
)

// ConnectionManagerConfig configures the connection manager
type ConnectionManagerConfig struct {
	ShardCount          int
	MaxConnectionsPerIP int
	MaxTotalConnections int
	IdleTimeout         time.Duration
	HandshakeTimeout    time.Duration

	// Callbacks
	OnConnect    func(conn *ManagedConnection)
	OnDisconnect func(conn *ManagedConnection, reason string)
}

// DefaultConnectionManagerConfig returns production defaults
func DefaultConnectionManagerConfig() ConnectionManagerConfig {
	return ConnectionManagerConfig{
		ShardCount:          DefaultShardCount,
		MaxConnectionsPerIP: DefaultMaxConnectionsPerIP,
		MaxTotalConnections: DefaultMaxTotalConnections,
		IdleTimeout:         DefaultIdleTimeout,
		HandshakeTimeout:    DefaultHandshakeTimeout,
	}
}

// ManagedConnection wraps a client connection with metadata
type ManagedConnection struct {
	ID           string
	Conn         net.Conn
	RemoteIP     string
	WorkerName   string
	Subscribed   bool
	Authorized   bool
	Extranonce1  string
	Difficulty   uint64
	HardwareType string

	// Timing
	ConnectedAt  time.Time
	LastActivity time.Time

	// Communication
	SendChan chan []byte
	ctx      context.Context
	cancel   context.CancelFunc

	// Statistics (atomic)
	SharesSubmitted int64
	SharesAccepted  int64
	SharesRejected  int64
	BytesSent       int64
	BytesReceived   int64
}

// connectionShard holds a subset of connections
type connectionShard struct {
	connections map[string]*ManagedConnection
	mu          sync.RWMutex
}

// ConnectionManager manages all miner connections with sharding
type ConnectionManager struct {
	config ConnectionManagerConfig
	shards []*connectionShard

	// IP tracking for rate limiting (separate shard)
	ipCounts  map[string]int32
	ipCountMu sync.RWMutex

	// Global statistics (atomic)
	stats ConnectionStats

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// ConnectionStats tracks global connection statistics
type ConnectionStats struct {
	TotalConnections    int64
	ActiveConnections   int64
	TotalDisconnections int64
	RejectedConnections int64 // Due to limits
	PeakConnections     int64
	TotalBytesSent      int64
	TotalBytesReceived  int64
}

// NewConnectionManager creates a new sharded connection manager
func NewConnectionManager(config ConnectionManagerConfig) *ConnectionManager {
	if config.ShardCount <= 0 {
		config.ShardCount = DefaultShardCount
	}
	// Ensure power of 2 for fast modulo
	config.ShardCount = nextPowerOf2(config.ShardCount)

	ctx, cancel := context.WithCancel(context.Background())

	cm := &ConnectionManager{
		config:   config,
		shards:   make([]*connectionShard, config.ShardCount),
		ipCounts: make(map[string]int32),
		ctx:      ctx,
		cancel:   cancel,
	}

	// Initialize shards
	for i := 0; i < config.ShardCount; i++ {
		cm.shards[i] = &connectionShard{
			connections: make(map[string]*ManagedConnection),
		}
	}

	return cm
}

// Start begins background maintenance tasks
func (cm *ConnectionManager) Start() {
	// Start idle connection reaper
	cm.wg.Add(1)
	go cm.idleConnectionReaper()
}

// Stop gracefully shuts down the connection manager
func (cm *ConnectionManager) Stop() {
	cm.cancel()

	// Close all connections
	for _, shard := range cm.shards {
		shard.mu.Lock()
		for _, conn := range shard.connections {
			conn.cancel()
			conn.Conn.Close()
		}
		shard.mu.Unlock()
	}

	cm.wg.Wait()
}

// AddConnection adds a new connection (returns error if limits exceeded)
func (cm *ConnectionManager) AddConnection(conn *ManagedConnection) error {
	// Check global limit
	active := atomic.LoadInt64(&cm.stats.ActiveConnections)
	if active >= int64(cm.config.MaxTotalConnections) {
		atomic.AddInt64(&cm.stats.RejectedConnections, 1)
		return ErrMaxConnectionsReached
	}

	// Check per-IP limit
	if !cm.checkIPLimit(conn.RemoteIP) {
		atomic.AddInt64(&cm.stats.RejectedConnections, 1)
		return ErrIPLimitReached
	}

	// Get shard using consistent hash
	shard := cm.getShard(conn.ID)

	shard.mu.Lock()
	shard.connections[conn.ID] = conn
	shard.mu.Unlock()

	// Update statistics
	atomic.AddInt64(&cm.stats.TotalConnections, 1)
	newActive := atomic.AddInt64(&cm.stats.ActiveConnections, 1)

	// Update peak
	for {
		peak := atomic.LoadInt64(&cm.stats.PeakConnections)
		if newActive <= peak || atomic.CompareAndSwapInt64(&cm.stats.PeakConnections, peak, newActive) {
			break
		}
	}

	// Increment IP count
	cm.incrementIPCount(conn.RemoteIP)

	// Callback
	if cm.config.OnConnect != nil {
		cm.config.OnConnect(conn)
	}

	return nil
}

// RemoveConnection removes a connection
func (cm *ConnectionManager) RemoveConnection(connID string, reason string) {
	shard := cm.getShard(connID)

	shard.mu.Lock()
	conn, exists := shard.connections[connID]
	if exists {
		delete(shard.connections, connID)
	}
	shard.mu.Unlock()

	if !exists {
		return
	}

	// Update statistics
	atomic.AddInt64(&cm.stats.TotalDisconnections, 1)
	atomic.AddInt64(&cm.stats.ActiveConnections, -1)

	// Decrement IP count
	cm.decrementIPCount(conn.RemoteIP)

	// Callback
	if cm.config.OnDisconnect != nil {
		cm.config.OnDisconnect(conn, reason)
	}
}

// GetConnection retrieves a connection by ID
func (cm *ConnectionManager) GetConnection(connID string) (*ManagedConnection, bool) {
	shard := cm.getShard(connID)

	shard.mu.RLock()
	conn, exists := shard.connections[connID]
	shard.mu.RUnlock()

	return conn, exists
}

// UpdateActivity updates the last activity time for a connection
func (cm *ConnectionManager) UpdateActivity(connID string) {
	shard := cm.getShard(connID)

	shard.mu.RLock()
	conn, exists := shard.connections[connID]
	shard.mu.RUnlock()

	if exists {
		conn.LastActivity = time.Now()
	}
}

// Broadcast sends a message to all connections (parallel across shards)
func (cm *ConnectionManager) Broadcast(msg []byte) {
	var wg sync.WaitGroup

	for _, shard := range cm.shards {
		wg.Add(1)
		go func(s *connectionShard) {
			defer wg.Done()
			s.mu.RLock()
			for _, conn := range s.connections {
				select {
				case conn.SendChan <- msg:
					atomic.AddInt64(&conn.BytesSent, int64(len(msg)))
				default:
					// Channel full - skip this connection
				}
			}
			s.mu.RUnlock()
		}(shard)
	}

	wg.Wait()
}

// BroadcastToAuthorized sends a message only to authorized connections
func (cm *ConnectionManager) BroadcastToAuthorized(msg []byte) {
	var wg sync.WaitGroup

	for _, shard := range cm.shards {
		wg.Add(1)
		go func(s *connectionShard) {
			defer wg.Done()
			s.mu.RLock()
			for _, conn := range s.connections {
				if conn.Authorized {
					select {
					case conn.SendChan <- msg:
						atomic.AddInt64(&conn.BytesSent, int64(len(msg)))
					default:
					}
				}
			}
			s.mu.RUnlock()
		}(shard)
	}

	wg.Wait()
}

// GetActiveCount returns the number of active connections
func (cm *ConnectionManager) GetActiveCount() int64 {
	return atomic.LoadInt64(&cm.stats.ActiveConnections)
}

// GetAuthorizedCount returns the number of authorized miners
func (cm *ConnectionManager) GetAuthorizedCount() int64 {
	var count int64

	for _, shard := range cm.shards {
		shard.mu.RLock()
		for _, conn := range shard.connections {
			if conn.Authorized {
				count++
			}
		}
		shard.mu.RUnlock()
	}

	return count
}

// GetStats returns current statistics (lock-free read)
func (cm *ConnectionManager) GetStats() ConnectionStats {
	return ConnectionStats{
		TotalConnections:    atomic.LoadInt64(&cm.stats.TotalConnections),
		ActiveConnections:   atomic.LoadInt64(&cm.stats.ActiveConnections),
		TotalDisconnections: atomic.LoadInt64(&cm.stats.TotalDisconnections),
		RejectedConnections: atomic.LoadInt64(&cm.stats.RejectedConnections),
		PeakConnections:     atomic.LoadInt64(&cm.stats.PeakConnections),
		TotalBytesSent:      atomic.LoadInt64(&cm.stats.TotalBytesSent),
		TotalBytesReceived:  atomic.LoadInt64(&cm.stats.TotalBytesReceived),
	}
}

// GetConnectionsByIP returns all connections from a specific IP
func (cm *ConnectionManager) GetConnectionsByIP(ip string) []*ManagedConnection {
	var result []*ManagedConnection

	for _, shard := range cm.shards {
		shard.mu.RLock()
		for _, conn := range shard.connections {
			if conn.RemoteIP == ip {
				result = append(result, conn)
			}
		}
		shard.mu.RUnlock()
	}

	return result
}

// ForEach iterates over all connections (use sparingly - not optimized for hot path)
func (cm *ConnectionManager) ForEach(fn func(*ManagedConnection) bool) {
	for _, shard := range cm.shards {
		shard.mu.RLock()
		for _, conn := range shard.connections {
			if !fn(conn) {
				shard.mu.RUnlock()
				return
			}
		}
		shard.mu.RUnlock()
	}
}

// Internal methods

func (cm *ConnectionManager) getShard(connID string) *connectionShard {
	// Fast hash using FNV-1a
	hash := uint32(2166136261)
	for i := 0; i < len(connID); i++ {
		hash ^= uint32(connID[i])
		hash *= 16777619
	}
	// Fast modulo for power of 2
	return cm.shards[hash&uint32(len(cm.shards)-1)]
}

func (cm *ConnectionManager) checkIPLimit(ip string) bool {
	cm.ipCountMu.RLock()
	count := cm.ipCounts[ip]
	cm.ipCountMu.RUnlock()

	return count < int32(cm.config.MaxConnectionsPerIP)
}

func (cm *ConnectionManager) incrementIPCount(ip string) {
	cm.ipCountMu.Lock()
	cm.ipCounts[ip]++
	cm.ipCountMu.Unlock()
}

func (cm *ConnectionManager) decrementIPCount(ip string) {
	cm.ipCountMu.Lock()
	cm.ipCounts[ip]--
	if cm.ipCounts[ip] <= 0 {
		delete(cm.ipCounts, ip)
	}
	cm.ipCountMu.Unlock()
}

func (cm *ConnectionManager) idleConnectionReaper() {
	defer cm.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-cm.ctx.Done():
			return
		case <-ticker.C:
			cm.reapIdleConnections()
		}
	}
}

func (cm *ConnectionManager) reapIdleConnections() {
	now := time.Now()
	idleThreshold := now.Add(-cm.config.IdleTimeout)

	for _, shard := range cm.shards {
		var toRemove []string

		shard.mu.RLock()
		for id, conn := range shard.connections {
			if conn.LastActivity.Before(idleThreshold) {
				toRemove = append(toRemove, id)
			}
		}
		shard.mu.RUnlock()

		// Remove idle connections
		for _, id := range toRemove {
			cm.RemoveConnection(id, "idle timeout")
		}
	}
}

// Helper functions

func nextPowerOf2(n int) int {
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n++
	return n
}

// Errors
type connError string

func (e connError) Error() string {
	return string(e)
}

const (
	ErrMaxConnectionsReached connError = "maximum connections reached"
	ErrIPLimitReached        connError = "per-IP connection limit reached"
	ErrConnectionNotFound    connError = "connection not found"
)
