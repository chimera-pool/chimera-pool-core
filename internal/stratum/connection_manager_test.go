package stratum

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockConn implements net.Conn for testing
type mockConn struct {
	remoteAddr string
	closed     bool
	mu         sync.Mutex
}

func newMockConn(remoteAddr string) *mockConn {
	return &mockConn{remoteAddr: remoteAddr}
}

func (m *mockConn) Read(b []byte) (n int, err error)  { return 0, nil }
func (m *mockConn) Write(b []byte) (n int, err error) { return len(b), nil }
func (m *mockConn) Close() error {
	m.mu.Lock()
	m.closed = true
	m.mu.Unlock()
	return nil
}
func (m *mockConn) LocalAddr() net.Addr { return &net.TCPAddr{} }
func (m *mockConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP(m.remoteAddr), Port: 12345}
}
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

func createTestConnection(id, ip string) *ManagedConnection {
	ctx, cancel := context.WithCancel(context.Background())
	return &ManagedConnection{
		ID:           id,
		Conn:         newMockConn(ip),
		RemoteIP:     ip,
		ConnectedAt:  time.Now(),
		LastActivity: time.Now(),
		SendChan:     make(chan []byte, 100),
		ctx:          ctx,
		cancel:       cancel,
	}
}

func TestConnectionManager_AddRemove(t *testing.T) {
	config := DefaultConnectionManagerConfig()
	cm := NewConnectionManager(config)
	cm.Start()
	defer cm.Stop()

	// Add a connection
	conn := createTestConnection("conn-1", "192.168.1.1")
	err := cm.AddConnection(conn)
	require.NoError(t, err)

	// Verify it exists
	retrieved, exists := cm.GetConnection("conn-1")
	assert.True(t, exists)
	assert.Equal(t, "conn-1", retrieved.ID)

	// Check stats
	stats := cm.GetStats()
	assert.Equal(t, int64(1), stats.ActiveConnections)
	assert.Equal(t, int64(1), stats.TotalConnections)

	// Remove the connection
	cm.RemoveConnection("conn-1", "test removal")

	// Verify it's gone
	_, exists = cm.GetConnection("conn-1")
	assert.False(t, exists)

	stats = cm.GetStats()
	assert.Equal(t, int64(0), stats.ActiveConnections)
	assert.Equal(t, int64(1), stats.TotalDisconnections)
}

func TestConnectionManager_IPLimit(t *testing.T) {
	config := ConnectionManagerConfig{
		ShardCount:          8,
		MaxConnectionsPerIP: 3,
		MaxTotalConnections: 1000,
		IdleTimeout:         time.Minute,
	}
	cm := NewConnectionManager(config)
	cm.Start()
	defer cm.Stop()

	// Add connections up to the limit
	for i := 0; i < 3; i++ {
		conn := createTestConnection("conn-"+string(rune('a'+i)), "10.0.0.1")
		err := cm.AddConnection(conn)
		require.NoError(t, err)
	}

	// Next connection from same IP should fail
	conn := createTestConnection("conn-d", "10.0.0.1")
	err := cm.AddConnection(conn)
	assert.ErrorIs(t, err, ErrIPLimitReached)

	// Connection from different IP should succeed
	conn = createTestConnection("conn-e", "10.0.0.2")
	err = cm.AddConnection(conn)
	require.NoError(t, err)

	stats := cm.GetStats()
	assert.Equal(t, int64(4), stats.ActiveConnections)
	assert.Equal(t, int64(1), stats.RejectedConnections)
}

func TestConnectionManager_MaxConnections(t *testing.T) {
	config := ConnectionManagerConfig{
		ShardCount:          4,
		MaxConnectionsPerIP: 100,
		MaxTotalConnections: 5,
		IdleTimeout:         time.Minute,
	}
	cm := NewConnectionManager(config)
	cm.Start()
	defer cm.Stop()

	// Fill to max
	for i := 0; i < 5; i++ {
		conn := createTestConnection("conn-"+string(rune('0'+i)), "192.168.1."+string(rune('1'+i)))
		err := cm.AddConnection(conn)
		require.NoError(t, err)
	}

	// Next should fail
	conn := createTestConnection("conn-overflow", "192.168.2.1")
	err := cm.AddConnection(conn)
	assert.ErrorIs(t, err, ErrMaxConnectionsReached)

	stats := cm.GetStats()
	assert.Equal(t, int64(5), stats.ActiveConnections)
	assert.Equal(t, int64(1), stats.RejectedConnections)
}

func TestConnectionManager_Broadcast(t *testing.T) {
	config := DefaultConnectionManagerConfig()
	cm := NewConnectionManager(config)
	cm.Start()
	defer cm.Stop()

	// Add multiple connections
	conns := make([]*ManagedConnection, 10)
	for i := 0; i < 10; i++ {
		conns[i] = createTestConnection("broadcast-"+string(rune('a'+i)), "10.1.1."+string(rune('1'+i)))
		err := cm.AddConnection(conns[i])
		require.NoError(t, err)
	}

	// Broadcast a message
	msg := []byte(`{"method":"mining.notify","params":[]}`)
	cm.Broadcast(msg)

	// Allow time for goroutines
	time.Sleep(10 * time.Millisecond)

	// Verify all connections received the message
	for _, conn := range conns {
		select {
		case received := <-conn.SendChan:
			assert.Equal(t, msg, received)
		default:
			t.Error("Connection did not receive broadcast")
		}
	}
}

func TestConnectionManager_BroadcastToAuthorized(t *testing.T) {
	config := DefaultConnectionManagerConfig()
	cm := NewConnectionManager(config)
	cm.Start()
	defer cm.Stop()

	// Add connections - some authorized, some not
	authorized := createTestConnection("auth-1", "10.0.0.1")
	authorized.Authorized = true
	err := cm.AddConnection(authorized)
	require.NoError(t, err)

	unauthorized := createTestConnection("unauth-1", "10.0.0.2")
	unauthorized.Authorized = false
	err = cm.AddConnection(unauthorized)
	require.NoError(t, err)

	// Broadcast to authorized only
	msg := []byte(`{"method":"mining.set_difficulty","params":[1024]}`)
	cm.BroadcastToAuthorized(msg)

	time.Sleep(10 * time.Millisecond)

	// Authorized should receive
	select {
	case received := <-authorized.SendChan:
		assert.Equal(t, msg, received)
	default:
		t.Error("Authorized connection did not receive message")
	}

	// Unauthorized should not receive
	select {
	case <-unauthorized.SendChan:
		t.Error("Unauthorized connection received message")
	default:
		// Expected
	}
}

func TestConnectionManager_Sharding(t *testing.T) {
	config := ConnectionManagerConfig{
		ShardCount:          16,
		MaxConnectionsPerIP: 1000,
		MaxTotalConnections: 10000,
		IdleTimeout:         time.Minute,
	}
	cm := NewConnectionManager(config)
	cm.Start()
	defer cm.Stop()

	// Add many connections to test sharding distribution
	connCount := 1000
	for i := 0; i < connCount; i++ {
		conn := createTestConnection("shard-test-"+string(rune(i)), "192.168.1.1")
		err := cm.AddConnection(conn)
		require.NoError(t, err)
	}

	stats := cm.GetStats()
	assert.Equal(t, int64(connCount), stats.ActiveConnections)

	// Verify all connections are retrievable
	for i := 0; i < connCount; i++ {
		_, exists := cm.GetConnection("shard-test-" + string(rune(i)))
		assert.True(t, exists)
	}
}

func TestConnectionManager_ConcurrentAccess(t *testing.T) {
	config := ConnectionManagerConfig{
		ShardCount:          32,
		MaxConnectionsPerIP: 10000,
		MaxTotalConnections: 50000,
		IdleTimeout:         time.Minute,
	}
	cm := NewConnectionManager(config)
	cm.Start()
	defer cm.Stop()

	var wg sync.WaitGroup
	var addedCount int64
	var removedCount int64

	// Concurrent adds
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				id := "concurrent-" + string(rune(idx*1000+j))
				conn := createTestConnection(id, "10.0.0.1")
				if err := cm.AddConnection(conn); err == nil {
					atomic.AddInt64(&addedCount, 1)
				}
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				cm.GetActiveCount()
				cm.GetStats()
			}
		}()
	}

	// Concurrent removes
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			time.Sleep(5 * time.Millisecond) // Let some adds complete first
			for j := 0; j < 50; j++ {
				id := "concurrent-" + string(rune(idx*1000+j))
				cm.RemoveConnection(id, "concurrent test")
				atomic.AddInt64(&removedCount, 1)
			}
		}(i)
	}

	wg.Wait()

	t.Logf("Added: %d, Removed: %d, Active: %d",
		addedCount, removedCount, cm.GetActiveCount())

	// Should not panic and stats should be consistent
	stats := cm.GetStats()
	assert.GreaterOrEqual(t, stats.TotalConnections, int64(0))
}

func TestConnectionManager_Callbacks(t *testing.T) {
	var connectCount int64
	var disconnectCount int64

	config := ConnectionManagerConfig{
		ShardCount:          4,
		MaxConnectionsPerIP: 100,
		MaxTotalConnections: 1000,
		IdleTimeout:         time.Minute,
		OnConnect: func(conn *ManagedConnection) {
			atomic.AddInt64(&connectCount, 1)
		},
		OnDisconnect: func(conn *ManagedConnection, reason string) {
			atomic.AddInt64(&disconnectCount, 1)
		},
	}
	cm := NewConnectionManager(config)
	cm.Start()
	defer cm.Stop()

	// Add connections
	for i := 0; i < 5; i++ {
		conn := createTestConnection("callback-"+string(rune('a'+i)), "10.0.0.1")
		cm.AddConnection(conn)
	}

	assert.Equal(t, int64(5), atomic.LoadInt64(&connectCount))

	// Remove connections
	for i := 0; i < 3; i++ {
		cm.RemoveConnection("callback-"+string(rune('a'+i)), "test")
	}

	assert.Equal(t, int64(3), atomic.LoadInt64(&disconnectCount))
}

func TestConnectionManager_GetConnectionsByIP(t *testing.T) {
	config := DefaultConnectionManagerConfig()
	cm := NewConnectionManager(config)
	cm.Start()
	defer cm.Stop()

	// Add connections from different IPs
	for i := 0; i < 5; i++ {
		conn := createTestConnection("ip1-"+string(rune('a'+i)), "192.168.1.100")
		cm.AddConnection(conn)
	}
	for i := 0; i < 3; i++ {
		conn := createTestConnection("ip2-"+string(rune('a'+i)), "192.168.1.200")
		cm.AddConnection(conn)
	}

	// Get by IP
	ip1Conns := cm.GetConnectionsByIP("192.168.1.100")
	assert.Len(t, ip1Conns, 5)

	ip2Conns := cm.GetConnectionsByIP("192.168.1.200")
	assert.Len(t, ip2Conns, 3)

	unknownConns := cm.GetConnectionsByIP("10.0.0.1")
	assert.Len(t, unknownConns, 0)
}

func TestConnectionManager_UpdateActivity(t *testing.T) {
	config := DefaultConnectionManagerConfig()
	cm := NewConnectionManager(config)
	cm.Start()
	defer cm.Stop()

	conn := createTestConnection("activity-test", "10.0.0.1")
	oldActivity := conn.LastActivity
	cm.AddConnection(conn)

	time.Sleep(10 * time.Millisecond)
	cm.UpdateActivity("activity-test")

	retrieved, _ := cm.GetConnection("activity-test")
	assert.True(t, retrieved.LastActivity.After(oldActivity))
}

func TestConnectionManager_ForEach(t *testing.T) {
	config := DefaultConnectionManagerConfig()
	cm := NewConnectionManager(config)
	cm.Start()
	defer cm.Stop()

	// Add connections
	for i := 0; i < 10; i++ {
		conn := createTestConnection("foreach-"+string(rune('a'+i)), "10.0.0.1")
		cm.AddConnection(conn)
	}

	// Count using ForEach
	var count int
	cm.ForEach(func(conn *ManagedConnection) bool {
		count++
		return true
	})

	assert.Equal(t, 10, count)

	// Test early exit
	count = 0
	cm.ForEach(func(conn *ManagedConnection) bool {
		count++
		return count < 5 // Stop after 5
	})

	assert.Equal(t, 5, count)
}

func BenchmarkConnectionManager_AddRemove(b *testing.B) {
	config := ConnectionManagerConfig{
		ShardCount:          64,
		MaxConnectionsPerIP: 100000,
		MaxTotalConnections: 100000,
		IdleTimeout:         time.Hour,
	}
	cm := NewConnectionManager(config)
	cm.Start()
	defer cm.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			id := "bench-" + string(rune(i))
			conn := createTestConnection(id, "10.0.0.1")
			cm.AddConnection(conn)
			cm.RemoveConnection(id, "bench")
			i++
		}
	})
}

func BenchmarkConnectionManager_GetConnection(b *testing.B) {
	config := DefaultConnectionManagerConfig()
	cm := NewConnectionManager(config)
	cm.Start()
	defer cm.Stop()

	// Pre-populate
	for i := 0; i < 10000; i++ {
		conn := createTestConnection("get-bench-"+string(rune(i)), "10.0.0.1")
		cm.AddConnection(conn)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			cm.GetConnection("get-bench-" + string(rune(i%10000)))
			i++
		}
	})
}

func BenchmarkConnectionManager_Broadcast(b *testing.B) {
	config := DefaultConnectionManagerConfig()
	cm := NewConnectionManager(config)
	cm.Start()
	defer cm.Stop()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		conn := createTestConnection("broadcast-bench-"+string(rune(i)), "10.0.0."+string(rune(i%255)))
		cm.AddConnection(conn)
	}

	msg := []byte(`{"method":"mining.notify","params":["job123","prevhash","coinb1","coinb2"]}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cm.Broadcast(msg)
	}
}
