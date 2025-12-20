package detector

import (
	"errors"
	"io"
	"net"
	"sync"
	"time"
)

// =============================================================================
// PROTOCOL DETECTOR AND ROUTER
// Auto-detects Stratum V1 (JSON) vs V2 (Binary) for hybrid pool support
// =============================================================================

// Protocol version constants
type ProtocolVersion int

const (
	ProtocolUnknown ProtocolVersion = iota
	ProtocolV1                      // JSON-based Stratum v1
	ProtocolV2                      // Binary Stratum v2
)

// String returns the protocol name
func (p ProtocolVersion) String() string {
	switch p {
	case ProtocolV1:
		return "stratum-v1"
	case ProtocolV2:
		return "stratum-v2"
	default:
		return "unknown"
	}
}

// Detection constants
const (
	// Number of bytes to peek for detection
	PeekSize = 6

	// Detection timeout
	DetectionTimeout = 5 * time.Second

	// V1 JSON starts with '{' (0x7B)
	V1JSONStart byte = '{'

	// V2 frame header starts with extension_type (2 bytes, usually 0x0000)
	// followed by msg_type (1 byte, 0x00 for SetupConnection)
)

// Errors
var (
	ErrDetectionTimeout    = errors.New("protocol detection timeout")
	ErrDetectionFailed     = errors.New("failed to detect protocol")
	ErrConnectionClosed    = errors.New("connection closed during detection")
	ErrNoHandlerRegistered = errors.New("no handler registered for protocol")
	ErrRouterClosed        = errors.New("router is closed")
)

// =============================================================================
// Peekable Connection
// =============================================================================

// PeekableConn wraps a net.Conn to allow peeking without consuming bytes
type PeekableConn struct {
	net.Conn
	peeked []byte
	mu     sync.Mutex
}

// NewPeekableConn creates a new peekable connection wrapper
func NewPeekableConn(conn net.Conn) *PeekableConn {
	return &PeekableConn{
		Conn:   conn,
		peeked: nil,
	}
}

// Peek reads n bytes without consuming them
func (pc *PeekableConn) Peek(n int) ([]byte, error) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	// If we've already peeked enough, return from buffer
	if len(pc.peeked) >= n {
		return pc.peeked[:n], nil
	}

	// Need to read more bytes
	needed := n - len(pc.peeked)
	buf := make([]byte, needed)

	read, err := io.ReadFull(pc.Conn, buf)
	if err != nil {
		if read > 0 {
			pc.peeked = append(pc.peeked, buf[:read]...)
		}
		return pc.peeked, err
	}

	pc.peeked = append(pc.peeked, buf...)
	return pc.peeked[:n], nil
}

// Read implements io.Reader, returning peeked bytes first
func (pc *PeekableConn) Read(b []byte) (int, error) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	// If we have peeked bytes, return those first
	if len(pc.peeked) > 0 {
		n := copy(b, pc.peeked)
		pc.peeked = pc.peeked[n:]
		return n, nil
	}

	// Otherwise read from underlying connection
	return pc.Conn.Read(b)
}

// =============================================================================
// Protocol Detector
// =============================================================================

// Detector detects the protocol version from connection data
type Detector struct {
	timeout time.Duration
}

// NewDetector creates a new protocol detector
func NewDetector() *Detector {
	return &Detector{
		timeout: DetectionTimeout,
	}
}

// NewDetectorWithTimeout creates a detector with custom timeout
func NewDetectorWithTimeout(timeout time.Duration) *Detector {
	return &Detector{
		timeout: timeout,
	}
}

// Detect determines the protocol version from a connection
func (d *Detector) Detect(conn net.Conn) (ProtocolVersion, *PeekableConn, error) {
	// Wrap connection for peeking
	pc := NewPeekableConn(conn)

	// Set read deadline for detection
	if d.timeout > 0 {
		conn.SetReadDeadline(time.Now().Add(d.timeout))
	}

	// Peek at first bytes
	peeked, err := pc.Peek(PeekSize)
	if err != nil {
		if err == io.EOF {
			return ProtocolUnknown, pc, ErrConnectionClosed
		}
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return ProtocolUnknown, pc, ErrDetectionTimeout
		}
		// Partial read - try to detect with what we have
		if len(peeked) == 0 {
			return ProtocolUnknown, pc, ErrDetectionFailed
		}
	}

	// Reset read deadline
	conn.SetReadDeadline(time.Time{})

	// Detect protocol from peeked bytes
	version := d.detectFromBytes(peeked)

	return version, pc, nil
}

// detectFromBytes analyzes bytes to determine protocol
func (d *Detector) detectFromBytes(data []byte) ProtocolVersion {
	if len(data) == 0 {
		return ProtocolUnknown
	}

	// V1 JSON always starts with '{' character
	if data[0] == V1JSONStart {
		return ProtocolV1
	}

	// V2 binary format detection:
	// - First 2 bytes: extension_type (typically 0x0000)
	// - Byte 3: msg_type (0x00 for SetupConnection)
	// - Bytes 4-6: msg_length (24-bit little-endian)

	// Check for valid V2 header pattern
	if len(data) >= 3 {
		// V2 SetupConnection starts with extension_type=0x0000, msg_type=0x00
		if data[0] == 0x00 && data[1] == 0x00 && data[2] == 0x00 {
			return ProtocolV2
		}

		// V2 with extensions (e.g., version rolling = 0x0001)
		if data[0] <= 0x07 && data[1] == 0x00 && data[2] <= 0x60 {
			// extension_type is typically 0-7, msg_type is 0-0x60
			return ProtocolV2
		}
	}

	// Additional heuristics for V1
	// JSON messages often start with {"id": or {"method":
	if len(data) >= 2 && data[0] == '{' && (data[1] == '"' || data[1] == ' ') {
		return ProtocolV1
	}

	return ProtocolUnknown
}

// DetectFromBytes is a convenience method for testing
func (d *Detector) DetectFromBytes(data []byte) ProtocolVersion {
	return d.detectFromBytes(data)
}

// =============================================================================
// Protocol Handler Interface
// =============================================================================

// Handler handles connections for a specific protocol
type Handler interface {
	HandleConnection(conn net.Conn) error
	Protocol() ProtocolVersion
	Shutdown() error
}

// =============================================================================
// Protocol Router
// =============================================================================

// Router routes connections to appropriate protocol handlers
type Router struct {
	detector *Detector
	handlers map[ProtocolVersion]Handler
	mu       sync.RWMutex
	closed   bool

	// Metrics
	v1Connections    uint64
	v2Connections    uint64
	failedDetections uint64
	metricsmu        sync.Mutex
}

// NewRouter creates a new protocol router
func NewRouter() *Router {
	return &Router{
		detector: NewDetector(),
		handlers: make(map[ProtocolVersion]Handler),
	}
}

// NewRouterWithDetector creates a router with custom detector
func NewRouterWithDetector(detector *Detector) *Router {
	return &Router{
		detector: detector,
		handlers: make(map[ProtocolVersion]Handler),
	}
}

// RegisterHandler registers a handler for a protocol version
func (r *Router) RegisterHandler(version ProtocolVersion, handler Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[version] = handler
}

// UnregisterHandler removes a handler
func (r *Router) UnregisterHandler(version ProtocolVersion) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.handlers, version)
}

// HasHandler checks if a handler is registered for a protocol
func (r *Router) HasHandler(version ProtocolVersion) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.handlers[version]
	return ok
}

// Route detects protocol and routes connection to appropriate handler
func (r *Router) Route(conn net.Conn) error {
	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		conn.Close()
		return ErrRouterClosed
	}
	r.mu.RUnlock()

	// Detect protocol
	version, peekConn, err := r.detector.Detect(conn)
	if err != nil {
		r.recordFailedDetection()
		conn.Close()
		return err
	}

	// Get handler
	r.mu.RLock()
	handler, ok := r.handlers[version]
	r.mu.RUnlock()

	if !ok {
		r.recordFailedDetection()
		conn.Close()
		return ErrNoHandlerRegistered
	}

	// Record metrics
	r.recordConnection(version)

	// Handle connection
	return handler.HandleConnection(peekConn)
}

// Close shuts down the router and all handlers
func (r *Router) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil
	}
	r.closed = true

	// Shutdown all handlers
	var lastErr error
	for _, handler := range r.handlers {
		if err := handler.Shutdown(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// IsClosed returns whether the router is closed
func (r *Router) IsClosed() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.closed
}

// Metrics methods
func (r *Router) recordConnection(version ProtocolVersion) {
	r.metricsmu.Lock()
	defer r.metricsmu.Unlock()
	switch version {
	case ProtocolV1:
		r.v1Connections++
	case ProtocolV2:
		r.v2Connections++
	}
}

func (r *Router) recordFailedDetection() {
	r.metricsmu.Lock()
	defer r.metricsmu.Unlock()
	r.failedDetections++
}

// GetMetrics returns connection metrics
func (r *Router) GetMetrics() (v1, v2, failed uint64) {
	r.metricsmu.Lock()
	defer r.metricsmu.Unlock()
	return r.v1Connections, r.v2Connections, r.failedDetections
}

// =============================================================================
// Connection Info
// =============================================================================

// ConnectionInfo holds information about a detected connection
type ConnectionInfo struct {
	Protocol     ProtocolVersion
	RemoteAddr   string
	DetectedAt   time.Time
	PeekableConn *PeekableConn
}

// DetectAndInfo returns detection result with additional info
func (d *Detector) DetectAndInfo(conn net.Conn) (*ConnectionInfo, error) {
	version, pc, err := d.Detect(conn)
	if err != nil {
		return nil, err
	}

	return &ConnectionInfo{
		Protocol:     version,
		RemoteAddr:   conn.RemoteAddr().String(),
		DetectedAt:   time.Now(),
		PeekableConn: pc,
	}, nil
}
