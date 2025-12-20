package detector

import (
	"bytes"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// TDD TESTS FOR PROTOCOL DETECTOR AND ROUTER
// =============================================================================

// -----------------------------------------------------------------------------
// ProtocolVersion Tests
// -----------------------------------------------------------------------------

func TestProtocolVersion_String(t *testing.T) {
	tests := []struct {
		version  ProtocolVersion
		expected string
	}{
		{ProtocolV1, "stratum-v1"},
		{ProtocolV2, "stratum-v2"},
		{ProtocolUnknown, "unknown"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.version.String())
	}
}

// -----------------------------------------------------------------------------
// PeekableConn Tests
// -----------------------------------------------------------------------------

func TestPeekableConn_Peek(t *testing.T) {
	// Create a mock connection with test data
	data := []byte(`{"id":1,"method":"mining.subscribe"}`)
	conn := newMockConn(data)
	pc := NewPeekableConn(conn)

	// Peek should return bytes without consuming
	peeked, err := pc.Peek(6)
	require.NoError(t, err)
	assert.Equal(t, []byte(`{"id":`), peeked)

	// Peek again should return same bytes
	peeked2, err := pc.Peek(6)
	require.NoError(t, err)
	assert.Equal(t, peeked, peeked2)
}

func TestPeekableConn_PeekThenRead(t *testing.T) {
	data := []byte(`{"id":1,"method":"mining.subscribe"}`)
	conn := newMockConn(data)
	pc := NewPeekableConn(conn)

	// Peek first
	_, err := pc.Peek(6)
	require.NoError(t, err)

	// First read should return peeked bytes
	buf := make([]byte, 6)
	n, err := pc.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 6, n)
	assert.Equal(t, []byte(`{"id":`), buf)

	// Second read should return remaining data from connection
	buf2 := make([]byte, 4)
	n2, err := pc.Read(buf2)
	require.NoError(t, err)
	assert.Equal(t, 4, n2)
	assert.Equal(t, []byte(`1,"m`), buf2)
}

func TestPeekableConn_ReadWithoutPeek(t *testing.T) {
	data := []byte("hello world")
	conn := newMockConn(data)
	pc := NewPeekableConn(conn)

	buf := make([]byte, 5)
	n, err := pc.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, []byte("hello"), buf)
}

func TestPeekableConn_PeekMoreThanAvailable(t *testing.T) {
	data := []byte("hi")
	conn := newMockConn(data)
	pc := NewPeekableConn(conn)

	// Peek more than available should return what's there with EOF
	peeked, err := pc.Peek(10)
	assert.Error(t, err) // EOF or short read
	assert.Equal(t, []byte("hi"), peeked)
}

// -----------------------------------------------------------------------------
// Detector Tests
// -----------------------------------------------------------------------------

func TestDetector_DetectV1_JSON(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		name string
		data []byte
	}{
		{"Standard subscribe", []byte(`{"id":1,"method":"mining.subscribe","params":[]}`)},
		{"With spaces", []byte(`{ "id": 1, "method": "mining.authorize" }`)},
		{"Minified", []byte(`{"id":1}`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := newMockConn(tt.data)
			version, _, err := detector.Detect(conn)
			require.NoError(t, err)
			assert.Equal(t, ProtocolV1, version)
		})
	}
}

func TestDetector_DetectV2_Binary(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		name string
		data []byte
	}{
		{
			"SetupConnection no extensions",
			[]byte{0x00, 0x00, 0x00, 0x64, 0x00, 0x00}, // ext=0, msg=0, len=100
		},
		{
			"SetupConnection with version rolling",
			[]byte{0x01, 0x00, 0x00, 0x64, 0x00, 0x00}, // ext=1, msg=0, len=100
		},
		{
			"NewMiningJob",
			[]byte{0x00, 0x00, 0x20, 0x00, 0x01, 0x00}, // ext=0, msg=0x20, len=256
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := newMockConn(tt.data)
			version, _, err := detector.Detect(conn)
			require.NoError(t, err)
			assert.Equal(t, ProtocolV2, version)
		})
	}
}

func TestDetector_DetectFromBytes(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		name     string
		data     []byte
		expected ProtocolVersion
	}{
		{"V1 JSON start", []byte(`{"id":1}`), ProtocolV1},
		{"V2 binary header", []byte{0x00, 0x00, 0x00, 0x64, 0x00, 0x00}, ProtocolV2},
		{"Empty", []byte{}, ProtocolUnknown},
		{"Random binary", []byte{0xFF, 0xFF, 0xFF}, ProtocolUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version := detector.DetectFromBytes(tt.data)
			assert.Equal(t, tt.expected, version)
		})
	}
}

func TestDetector_WithTimeout(t *testing.T) {
	detector := NewDetectorWithTimeout(100 * time.Millisecond)

	// Create a connection that never sends data
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	// Should timeout
	done := make(chan struct{})
	go func() {
		_, _, err := detector.Detect(client)
		assert.Error(t, err)
		close(done)
	}()

	select {
	case <-done:
		// Good - timeout occurred
	case <-time.After(500 * time.Millisecond):
		t.Fatal("detection should have timed out")
	}
}

func TestDetector_ConnectionClosed(t *testing.T) {
	detector := NewDetector()

	// Create and immediately close connection
	server, client := net.Pipe()
	server.Close()

	version, _, err := detector.Detect(client)
	assert.Error(t, err)
	assert.Equal(t, ProtocolUnknown, version)
	client.Close()
}

// -----------------------------------------------------------------------------
// Router Tests
// -----------------------------------------------------------------------------

func TestRouter_RegisterHandler(t *testing.T) {
	router := NewRouter()
	handler := &mockHandler{protocol: ProtocolV1}

	router.RegisterHandler(ProtocolV1, handler)
	assert.True(t, router.HasHandler(ProtocolV1))
	assert.False(t, router.HasHandler(ProtocolV2))
}

func TestRouter_UnregisterHandler(t *testing.T) {
	router := NewRouter()
	handler := &mockHandler{protocol: ProtocolV1}

	router.RegisterHandler(ProtocolV1, handler)
	assert.True(t, router.HasHandler(ProtocolV1))

	router.UnregisterHandler(ProtocolV1)
	assert.False(t, router.HasHandler(ProtocolV1))
}

func TestRouter_RouteV1(t *testing.T) {
	router := NewRouter()
	v1Handler := &mockHandler{protocol: ProtocolV1}
	router.RegisterHandler(ProtocolV1, v1Handler)

	// V1 JSON connection
	data := []byte(`{"id":1,"method":"mining.subscribe"}`)
	conn := newMockConn(data)

	err := router.Route(conn)
	require.NoError(t, err)
	assert.Equal(t, 1, v1Handler.handleCount)
}

func TestRouter_RouteV2(t *testing.T) {
	router := NewRouter()
	v2Handler := &mockHandler{protocol: ProtocolV2}
	router.RegisterHandler(ProtocolV2, v2Handler)

	// V2 binary connection
	data := []byte{0x00, 0x00, 0x00, 0x64, 0x00, 0x00}
	conn := newMockConn(data)

	err := router.Route(conn)
	require.NoError(t, err)
	assert.Equal(t, 1, v2Handler.handleCount)
}

func TestRouter_RouteMixed(t *testing.T) {
	router := NewRouter()
	v1Handler := &mockHandler{protocol: ProtocolV1}
	v2Handler := &mockHandler{protocol: ProtocolV2}
	router.RegisterHandler(ProtocolV1, v1Handler)
	router.RegisterHandler(ProtocolV2, v2Handler)

	// Route V1 connection
	v1Data := []byte(`{"id":1}`)
	err := router.Route(newMockConn(v1Data))
	require.NoError(t, err)

	// Route V2 connection
	v2Data := []byte{0x00, 0x00, 0x00, 0x64, 0x00, 0x00}
	err = router.Route(newMockConn(v2Data))
	require.NoError(t, err)

	assert.Equal(t, 1, v1Handler.handleCount)
	assert.Equal(t, 1, v2Handler.handleCount)
}

func TestRouter_NoHandler(t *testing.T) {
	router := NewRouter()
	// Only register V1
	router.RegisterHandler(ProtocolV1, &mockHandler{protocol: ProtocolV1})

	// Try to route V2 connection
	v2Data := []byte{0x00, 0x00, 0x00, 0x64, 0x00, 0x00}
	err := router.Route(newMockConn(v2Data))
	assert.Error(t, err)
	assert.Equal(t, ErrNoHandlerRegistered, err)
}

func TestRouter_Close(t *testing.T) {
	router := NewRouter()
	handler := &mockHandler{protocol: ProtocolV1}
	router.RegisterHandler(ProtocolV1, handler)

	err := router.Close()
	require.NoError(t, err)
	assert.True(t, router.IsClosed())
	assert.True(t, handler.shutdown)
}

func TestRouter_RouteAfterClose(t *testing.T) {
	router := NewRouter()
	router.RegisterHandler(ProtocolV1, &mockHandler{protocol: ProtocolV1})
	router.Close()

	err := router.Route(newMockConn([]byte(`{"id":1}`)))
	assert.Error(t, err)
	assert.Equal(t, ErrRouterClosed, err)
}

func TestRouter_Metrics(t *testing.T) {
	router := NewRouter()
	router.RegisterHandler(ProtocolV1, &mockHandler{protocol: ProtocolV1})
	router.RegisterHandler(ProtocolV2, &mockHandler{protocol: ProtocolV2})

	// Route some connections
	router.Route(newMockConn([]byte(`{"id":1}`)))
	router.Route(newMockConn([]byte(`{"id":2}`)))
	router.Route(newMockConn([]byte{0x00, 0x00, 0x00, 0x64, 0x00, 0x00}))

	v1, v2, failed := router.GetMetrics()
	assert.Equal(t, uint64(2), v1)
	assert.Equal(t, uint64(1), v2)
	assert.Equal(t, uint64(0), failed)
}

func TestRouter_ConcurrentRouting(t *testing.T) {
	router := NewRouter()
	v1Handler := &mockHandler{protocol: ProtocolV1}
	v2Handler := &mockHandler{protocol: ProtocolV2}
	router.RegisterHandler(ProtocolV1, v1Handler)
	router.RegisterHandler(ProtocolV2, v2Handler)

	var wg sync.WaitGroup
	routeCount := 100

	// Route V1 connections concurrently
	for i := 0; i < routeCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			router.Route(newMockConn([]byte(`{"id":1}`)))
		}()
	}

	// Route V2 connections concurrently
	for i := 0; i < routeCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			router.Route(newMockConn([]byte{0x00, 0x00, 0x00, 0x64, 0x00, 0x00}))
		}()
	}

	wg.Wait()

	v1, v2, _ := router.GetMetrics()
	assert.Equal(t, uint64(routeCount), v1)
	assert.Equal(t, uint64(routeCount), v2)
}

// -----------------------------------------------------------------------------
// ConnectionInfo Tests
// -----------------------------------------------------------------------------

func TestDetector_DetectAndInfo(t *testing.T) {
	detector := NewDetector()
	data := []byte(`{"id":1,"method":"mining.subscribe"}`)
	conn := newMockConnWithAddr(data, "192.168.1.100:12345")

	info, err := detector.DetectAndInfo(conn)
	require.NoError(t, err)
	assert.Equal(t, ProtocolV1, info.Protocol)
	assert.Equal(t, "192.168.1.100:12345", info.RemoteAddr)
	assert.False(t, info.DetectedAt.IsZero())
	assert.NotNil(t, info.PeekableConn)
}

// -----------------------------------------------------------------------------
// Benchmark Tests
// -----------------------------------------------------------------------------

func BenchmarkDetector_DetectV1(b *testing.B) {
	detector := NewDetector()
	data := []byte(`{"id":1,"method":"mining.subscribe","params":[]}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.Detect(newMockConn(data))
	}
}

func BenchmarkDetector_DetectV2(b *testing.B) {
	detector := NewDetector()
	data := []byte{0x00, 0x00, 0x00, 0x64, 0x00, 0x00}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.Detect(newMockConn(data))
	}
}

func BenchmarkRouter_Route(b *testing.B) {
	router := NewRouter()
	router.RegisterHandler(ProtocolV1, &mockHandler{protocol: ProtocolV1})
	data := []byte(`{"id":1}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.Route(newMockConn(data))
	}
}

// =============================================================================
// Mock Implementations
// =============================================================================

// mockConn implements net.Conn for testing
type mockConn struct {
	reader     *bytes.Reader
	remoteAddr net.Addr
	closed     bool
	mu         sync.Mutex
}

func newMockConn(data []byte) *mockConn {
	return &mockConn{
		reader:     bytes.NewReader(data),
		remoteAddr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345},
	}
}

func newMockConnWithAddr(data []byte, addr string) *mockConn {
	host, port, _ := net.SplitHostPort(addr)
	p := 0
	if port != "" {
		p = 12345
	}
	return &mockConn{
		reader:     bytes.NewReader(data),
		remoteAddr: &net.TCPAddr{IP: net.ParseIP(host), Port: p},
	}
}

func (m *mockConn) Read(b []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return 0, io.EOF
	}
	return m.reader.Read(b)
}

func (m *mockConn) Write(b []byte) (int, error) {
	return len(b), nil
}

func (m *mockConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 3333}
}

func (m *mockConn) RemoteAddr() net.Addr {
	return m.remoteAddr
}

func (m *mockConn) SetDeadline(t time.Time) error {
	return nil
}

func (m *mockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *mockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

// mockHandler implements Handler for testing
type mockHandler struct {
	protocol    ProtocolVersion
	handleCount int
	shutdown    bool
	mu          sync.Mutex
}

func (h *mockHandler) HandleConnection(conn net.Conn) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.handleCount++
	return nil
}

func (h *mockHandler) Protocol() ProtocolVersion {
	return h.protocol
}

func (h *mockHandler) Shutdown() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.shutdown = true
	return nil
}
