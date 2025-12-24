package stratum

import (
	"bufio"
	"encoding/json"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoolCoordinator_Creation(t *testing.T) {
	config := DefaultPoolCoordinatorConfig()
	config.ListenAddress = ":0" // Random port

	pc := NewPoolCoordinator(config)
	require.NotNil(t, pc)

	assert.NotNil(t, pc.connManager)
	assert.NotNil(t, pc.shareProcessor)
	assert.NotNil(t, pc.vardiffManager)
}

func TestPoolCoordinator_StartStop(t *testing.T) {
	config := DefaultPoolCoordinatorConfig()
	config.ListenAddress = ":0"

	pc := NewPoolCoordinator(config)

	err := pc.Start()
	require.NoError(t, err)

	// Give it time to start
	time.Sleep(50 * time.Millisecond)

	err = pc.Stop()
	require.NoError(t, err)
}

func TestPoolCoordinator_Stats(t *testing.T) {
	config := DefaultPoolCoordinatorConfig()
	config.ListenAddress = ":0"

	pc := NewPoolCoordinator(config)
	err := pc.Start()
	require.NoError(t, err)
	defer pc.Stop()

	stats := pc.GetStats()
	assert.Equal(t, int64(0), stats.ActiveMiners)
	assert.Equal(t, int64(0), stats.TotalSharesReceived)
}

func TestPoolCoordinator_SetCurrentJob(t *testing.T) {
	config := DefaultPoolCoordinatorConfig()
	config.ListenAddress = ":0"

	pc := NewPoolCoordinator(config)
	err := pc.Start()
	require.NoError(t, err)
	defer pc.Stop()

	// Subscribe to jobs
	jobCh := pc.SubscribeToJobs()

	// Set a job
	job := &Job{
		ID:        "test-job-1",
		PrevHash:  []byte{0x01, 0x02, 0x03},
		Coinbase1: []byte{0x04, 0x05},
		Coinbase2: []byte{0x06, 0x07},
		Version:   1,
		NBits:     0x1d00ffff,
		NTime:     uint32(time.Now().Unix()),
		CleanJobs: true,
	}

	pc.SetCurrentJob(job)

	// Verify job was set
	currentJob := pc.GetCurrentJob()
	require.NotNil(t, currentJob)
	assert.Equal(t, "test-job-1", currentJob.ID)

	// Verify subscriber received job
	select {
	case received := <-jobCh:
		assert.Equal(t, "test-job-1", received.ID)
	case <-time.After(100 * time.Millisecond):
		t.Error("Did not receive job notification")
	}
}

func TestPoolCoordinator_ClientConnection(t *testing.T) {
	config := DefaultPoolCoordinatorConfig()
	config.ListenAddress = ":0"

	pc := NewPoolCoordinator(config)
	err := pc.Start()
	require.NoError(t, err)
	defer pc.Stop()

	// Get the actual listening address
	addr := pc.listener.Addr().String()

	// Connect a client
	conn, err := net.Dial("tcp", addr)
	require.NoError(t, err)
	defer conn.Close()

	// Wait for connection to be registered
	time.Sleep(50 * time.Millisecond)

	stats := pc.GetStats()
	assert.Equal(t, int64(1), stats.ActiveMiners)
}

func TestPoolCoordinator_SubscribeFlow(t *testing.T) {
	config := DefaultPoolCoordinatorConfig()
	config.ListenAddress = ":0"

	pc := NewPoolCoordinator(config)
	err := pc.Start()
	require.NoError(t, err)
	defer pc.Stop()

	// Set a job first
	job := &Job{
		ID:        "subscribe-test-job",
		PrevHash:  []byte{0x01},
		Version:   1,
		NBits:     0x1d00ffff,
		NTime:     uint32(time.Now().Unix()),
		CleanJobs: false,
	}
	pc.SetCurrentJob(job)

	addr := pc.listener.Addr().String()
	conn, err := net.Dial("tcp", addr)
	require.NoError(t, err)
	defer conn.Close()

	// Send subscribe
	subscribeMsg := `{"id":1,"method":"mining.subscribe","params":["test-miner/1.0"]}` + "\n"
	_, err = conn.Write([]byte(subscribeMsg))
	require.NoError(t, err)

	// Read response line by line (stratum uses newline-delimited JSON)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	require.NoError(t, err)

	// Parse response (first line is the subscribe response)
	var response map[string]interface{}
	err = json.Unmarshal([]byte(line), &response)
	require.NoError(t, err)

	// Verify subscribe response
	assert.NotNil(t, response["result"])
	assert.Nil(t, response["error"])

	result, ok := response["result"].([]interface{})
	require.True(t, ok)
	assert.Len(t, result, 3) // subscriptions, extranonce1, extranonce2_size
}

func TestPoolCoordinator_AuthorizeFlow(t *testing.T) {
	config := DefaultPoolCoordinatorConfig()
	config.ListenAddress = ":0"

	pc := NewPoolCoordinator(config)
	err := pc.Start()
	require.NoError(t, err)
	defer pc.Stop()

	addr := pc.listener.Addr().String()
	conn, err := net.Dial("tcp", addr)
	require.NoError(t, err)
	defer conn.Close()

	// Use buffered reader for line-based protocol
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	reader := bufio.NewReader(conn)

	// Subscribe first
	subscribeMsg := `{"id":1,"method":"mining.subscribe","params":[]}` + "\n"
	_, err = conn.Write([]byte(subscribeMsg))
	require.NoError(t, err)

	// Read and discard subscribe response (line by line)
	_, err = reader.ReadString('\n')
	require.NoError(t, err)

	// Read and discard difficulty notification (if any)
	// Use a short timeout for optional messages
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	reader.ReadString('\n') // May or may not exist, ignore error

	// Reset deadline for authorize
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// Send authorize
	authMsg := `{"id":2,"method":"mining.authorize","params":["worker1.rig1","x"]}` + "\n"
	_, err = conn.Write([]byte(authMsg))
	require.NoError(t, err)

	// Read authorize response
	line, err := reader.ReadString('\n')
	require.NoError(t, err)

	var response map[string]interface{}
	err = json.Unmarshal([]byte(line), &response)
	require.NoError(t, err)

	// Verify authorize response
	assert.Equal(t, true, response["result"])
	assert.Nil(t, response["error"])

	// Wait for stats update
	time.Sleep(50 * time.Millisecond)

	stats := pc.GetStats()
	assert.Equal(t, int64(1), stats.AuthorizedMiners)
}

func TestPoolCoordinator_SubmitShare(t *testing.T) {
	config := DefaultPoolCoordinatorConfig()
	config.ListenAddress = ":0"

	pc := NewPoolCoordinator(config)
	err := pc.Start()
	require.NoError(t, err)

	// Set a job
	job := &Job{
		ID:        "submit-test-job",
		PrevHash:  []byte{0x01},
		Version:   1,
		NBits:     0x1d00ffff,
		NTime:     uint32(time.Now().Unix()),
		CleanJobs: false,
	}
	pc.SetCurrentJob(job)

	addr := pc.listener.Addr().String()
	conn, err := net.Dial("tcp", addr)
	require.NoError(t, err)

	buffer := make([]byte, 4096)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// Subscribe
	conn.Write([]byte(`{"id":1,"method":"mining.subscribe","params":[]}` + "\n"))
	conn.Read(buffer) // subscribe response
	conn.Read(buffer) // difficulty

	// Authorize
	conn.Write([]byte(`{"id":2,"method":"mining.authorize","params":["worker1","x"]}` + "\n"))
	conn.Read(buffer) // authorize response

	// Submit share
	submitMsg := `{"id":3,"method":"mining.submit","params":["worker1","submit-test-job","00000000","12345678","deadbeef"]}` + "\n"
	_, err = conn.Write([]byte(submitMsg))
	require.NoError(t, err)

	// Read submit response
	n, err := conn.Read(buffer)
	require.NoError(t, err)

	var response map[string]interface{}
	err = json.Unmarshal(buffer[:n], &response)
	require.NoError(t, err)

	// Response should have result (true or false) or error
	assert.NotNil(t, response["id"])

	// Close connection first to prevent race with Stop()
	conn.Close()

	// Wait for share processing to complete before stopping
	time.Sleep(200 * time.Millisecond)

	stats := pc.GetStats()
	assert.Greater(t, stats.TotalSharesReceived, int64(0))

	// Now stop the coordinator after all work is done
	pc.Stop()
}

func TestPoolCoordinator_ConcurrentConnections(t *testing.T) {
	config := DefaultPoolCoordinatorConfig()
	config.ListenAddress = ":0"
	config.MaxConnections = 100

	pc := NewPoolCoordinator(config)
	err := pc.Start()
	require.NoError(t, err)
	defer pc.Stop()

	addr := pc.listener.Addr().String()

	// Connect multiple clients concurrently
	connCount := 20
	conns := make([]net.Conn, connCount)

	for i := 0; i < connCount; i++ {
		conn, err := net.Dial("tcp", addr)
		require.NoError(t, err)
		conns[i] = conn
	}

	// Wait for all connections to be registered
	time.Sleep(100 * time.Millisecond)

	stats := pc.GetStats()
	assert.Equal(t, int64(connCount), stats.ActiveMiners)

	// Close all connections
	for _, conn := range conns {
		conn.Close()
	}

	// Wait for disconnections
	time.Sleep(100 * time.Millisecond)

	stats = pc.GetStats()
	assert.Equal(t, int64(0), stats.ActiveMiners)
}

func TestPoolCoordinator_ShareLatencyTracking(t *testing.T) {
	config := DefaultPoolCoordinatorConfig()
	config.ListenAddress = ":0"

	pc := NewPoolCoordinator(config)

	// Manually update latency to test tracking
	pc.updateShareLatency(1000000) // 1ms
	pc.updateShareLatency(2000000) // 2ms
	pc.updateShareLatency(5000000) // 5ms

	stats := pc.GetStats()
	assert.Equal(t, int64(5000000), stats.MaxShareLatencyNs)
	assert.Greater(t, stats.AvgShareLatencyNs, int64(0))
}

func TestDefaultPoolCoordinatorConfig(t *testing.T) {
	config := DefaultPoolCoordinatorConfig()

	assert.Equal(t, ":3333", config.ListenAddress)
	assert.Equal(t, 100000, config.MaxConnections)
	assert.Equal(t, 8, config.ShareWorkers)
	assert.Equal(t, 10*time.Second, config.TargetShareTime)
}

func BenchmarkPoolCoordinator_ShareSubmit(b *testing.B) {
	config := DefaultPoolCoordinatorConfig()
	config.ListenAddress = ":0"

	pc := NewPoolCoordinator(config)
	pc.Start()
	defer pc.Stop()

	// Set a job
	job := &Job{
		ID:       "bench-job",
		PrevHash: []byte{0x01},
		Version:  1,
		NBits:    0x1d00ffff,
		NTime:    uint32(time.Now().Unix()),
	}
	pc.SetCurrentJob(job)

	// Create test connection
	addr := pc.listener.Addr().String()
	conn, _ := net.Dial("tcp", addr)
	defer conn.Close()

	// Subscribe and authorize
	conn.Write([]byte(`{"id":1,"method":"mining.subscribe","params":[]}` + "\n"))
	conn.Write([]byte(`{"id":2,"method":"mining.authorize","params":["worker","x"]}` + "\n"))

	// Drain responses
	buffer := make([]byte, 8192)
	conn.SetReadDeadline(time.Now().Add(time.Second))
	for {
		_, err := conn.Read(buffer)
		if err != nil {
			break
		}
	}

	submitMsg := []byte(`{"id":3,"method":"mining.submit","params":["worker","bench-job","00000000","12345678","deadbeef"]}` + "\n")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		conn.Write(submitMsg)
	}

	stats := pc.GetStats()
	b.ReportMetric(float64(atomic.LoadInt64(&stats.TotalSharesReceived)), "shares_received")
}
