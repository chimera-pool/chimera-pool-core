package stratum

import (
	"errors"
	"net"
	"sync"
	"time"
)

// =============================================================================
// MOCK IMPLEMENTATIONS FOR TESTING
// These mocks implement the ISP interfaces for unit testing
// =============================================================================

// -----------------------------------------------------------------------------
// MockShareSubmitter
// -----------------------------------------------------------------------------

type MockShareSubmitter struct {
	AcceptAll   bool
	SubmitCount int
	LastShare   Share
	mu          sync.Mutex
}

func NewMockShareSubmitter() *MockShareSubmitter {
	return &MockShareSubmitter{
		AcceptAll: true,
	}
}

func (m *MockShareSubmitter) SubmitShare(share Share) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SubmitCount++
	m.LastShare = share
	if m.AcceptAll {
		return true, nil
	}
	return share.IsValid, nil
}

// -----------------------------------------------------------------------------
// MockShareValidator
// -----------------------------------------------------------------------------

type MockShareValidator struct {
	AlwaysValid bool
}

func NewMockShareValidator() *MockShareValidator {
	return &MockShareValidator{AlwaysValid: true}
}

func (m *MockShareValidator) ValidateShare(share Share, target []byte) (bool, error) {
	if m.AlwaysValid {
		return true, nil
	}
	return share.IsValid, nil
}

// -----------------------------------------------------------------------------
// MockJobDistributor
// -----------------------------------------------------------------------------

type MockJobDistributor struct {
	CurrentJob    Job
	handlers      []JobHandler
	subscriptions []*MockSubscription
	mu            sync.RWMutex
}

func NewMockJobDistributor() *MockJobDistributor {
	return &MockJobDistributor{
		handlers:      make([]JobHandler, 0),
		subscriptions: make([]*MockSubscription, 0),
	}
}

func (m *MockJobDistributor) GetCurrentJob() Job {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.CurrentJob
}

func (m *MockJobDistributor) SubscribeToJobs(handler JobHandler) Subscription {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers = append(m.handlers, handler)
	sub := &MockSubscription{active: true}
	m.subscriptions = append(m.subscriptions, sub)
	return sub
}

func (m *MockJobDistributor) BroadcastJob(job Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CurrentJob = job
	for i, handler := range m.handlers {
		if m.subscriptions[i].IsActive() {
			handler(job)
		}
	}
	return nil
}

// -----------------------------------------------------------------------------
// MockSubscription
// -----------------------------------------------------------------------------

type MockSubscription struct {
	active bool
	mu     sync.Mutex
}

func (s *MockSubscription) Unsubscribe() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.active = false
}

func (s *MockSubscription) IsActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.active
}

// -----------------------------------------------------------------------------
// MockDifficultyManager
// -----------------------------------------------------------------------------

type MockDifficultyManager struct {
	DefaultDifficulty uint64
	difficulties      map[string]uint64
	mu                sync.RWMutex
}

func NewMockDifficultyManager() *MockDifficultyManager {
	return &MockDifficultyManager{
		DefaultDifficulty: 1024,
		difficulties:      make(map[string]uint64),
	}
}

func (m *MockDifficultyManager) GetDifficulty(minerID string) uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if diff, ok := m.difficulties[minerID]; ok {
		return diff
	}
	return m.DefaultDifficulty
}

func (m *MockDifficultyManager) SetDifficulty(minerID string, difficulty uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.difficulties[minerID] = difficulty
	return nil
}

// -----------------------------------------------------------------------------
// MockVardiffManager
// -----------------------------------------------------------------------------

type MockVardiffManager struct {
	*MockDifficultyManager
	TargetShareTime time.Duration
}

func NewMockVardiffManager() *MockVardiffManager {
	return &MockVardiffManager{
		MockDifficultyManager: NewMockDifficultyManager(),
		TargetShareTime:       10 * time.Second,
	}
}

func (m *MockVardiffManager) AdjustDifficulty(minerID string, shareTime time.Duration) uint64 {
	currentDiff := m.GetDifficulty(minerID)
	if currentDiff == 0 {
		currentDiff = m.DefaultDifficulty
	}

	var newDiff uint64
	if shareTime < m.TargetShareTime/2 {
		// Shares too fast, double difficulty
		newDiff = currentDiff * 2
	} else if shareTime > m.TargetShareTime*2 {
		// Shares too slow, halve difficulty
		newDiff = currentDiff / 2
		if newDiff < 1 {
			newDiff = 1
		}
	} else {
		// Within acceptable range
		newDiff = currentDiff
	}

	m.SetDifficulty(minerID, newDiff)
	return newDiff
}

func (m *MockVardiffManager) GetTargetShareTime() time.Duration {
	return m.TargetShareTime
}

// -----------------------------------------------------------------------------
// MockMinerSession
// -----------------------------------------------------------------------------

type MockMinerSession struct {
	id            string
	workerName    string
	authorized    bool
	hardwareClass HardwareClass
	difficulty    uint64
	hashrate      float64
	shareCount    uint64
	lastShareTime time.Time
	mu            sync.RWMutex
}

func NewMockMinerSession(id string) *MockMinerSession {
	return &MockMinerSession{
		id:            id,
		hardwareClass: HardwareClassUnknown,
		difficulty:    1024,
	}
}

func (m *MockMinerSession) ID() string {
	return m.id
}

func (m *MockMinerSession) Authorize(worker, password string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.workerName = worker
	m.authorized = true
	return nil
}

func (m *MockMinerSession) IsAuthorized() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.authorized
}

func (m *MockMinerSession) GetWorkerName() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.workerName
}

func (m *MockMinerSession) GetHardwareClass() HardwareClass {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.hardwareClass
}

func (m *MockMinerSession) GetDifficulty() uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.difficulty
}

func (m *MockMinerSession) GetHashrate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.hashrate
}

func (m *MockMinerSession) GetShareCount() uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.shareCount
}

func (m *MockMinerSession) GetLastShareTime() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastShareTime
}

func (m *MockMinerSession) RecordShare() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shareCount++
	m.lastShareTime = time.Now()
}

func (m *MockMinerSession) SetHardwareClass(class HardwareClass) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hardwareClass = class
}

func (m *MockMinerSession) SetHashrate(rate float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hashrate = rate
}

// -----------------------------------------------------------------------------
// MockSessionManager
// -----------------------------------------------------------------------------

type MockSessionManager struct {
	sessions map[string]MinerSession
	mu       sync.RWMutex
}

func NewMockSessionManager() *MockSessionManager {
	return &MockSessionManager{
		sessions: make(map[string]MinerSession),
	}
}

func (m *MockSessionManager) CreateSession(conn StratumConnection) MinerSession {
	m.mu.Lock()
	defer m.mu.Unlock()
	session := NewMockMinerSession(conn.ID())
	m.sessions[conn.ID()] = session
	return session
}

func (m *MockSessionManager) GetSession(id string) (MinerSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[id]
	return session, ok
}

func (m *MockSessionManager) RemoveSession(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
}

func (m *MockSessionManager) GetActiveSessions() []MinerSession {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sessions := make([]MinerSession, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	return sessions
}

func (m *MockSessionManager) GetSessionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// -----------------------------------------------------------------------------
// MockProtocolHandler
// -----------------------------------------------------------------------------

type MockProtocolHandler struct {
	protocol       ProtocolVersion
	HandleFunc     func(conn net.Conn) error
	ConnectionsHnd int
	mu             sync.Mutex
}

func NewMockProtocolHandler(protocol ProtocolVersion) *MockProtocolHandler {
	return &MockProtocolHandler{
		protocol: protocol,
	}
}

func (m *MockProtocolHandler) HandleConnection(conn net.Conn) error {
	m.mu.Lock()
	m.ConnectionsHnd++
	m.mu.Unlock()
	if m.HandleFunc != nil {
		return m.HandleFunc(conn)
	}
	return nil
}

func (m *MockProtocolHandler) Protocol() ProtocolVersion {
	return m.protocol
}

func (m *MockProtocolHandler) Shutdown() error {
	return nil
}

// -----------------------------------------------------------------------------
// MockProtocolDetector
// -----------------------------------------------------------------------------

type MockProtocolDetector struct {
	DefaultProtocol ProtocolVersion
	DetectFunc      func(conn net.Conn) (ProtocolVersion, error)
}

func NewMockProtocolDetector() *MockProtocolDetector {
	return &MockProtocolDetector{
		DefaultProtocol: ProtocolV1,
	}
}

func (m *MockProtocolDetector) DetectProtocol(conn net.Conn) (ProtocolVersion, error) {
	if m.DetectFunc != nil {
		return m.DetectFunc(conn)
	}
	return m.DefaultProtocol, nil
}

// -----------------------------------------------------------------------------
// MockProtocolRouter
// -----------------------------------------------------------------------------

type MockProtocolRouter struct {
	handlers map[ProtocolVersion]ProtocolHandler
	detector ProtocolDetector
	mu       sync.RWMutex
}

func NewMockProtocolRouter(detector ProtocolDetector) *MockProtocolRouter {
	return &MockProtocolRouter{
		handlers: make(map[ProtocolVersion]ProtocolHandler),
		detector: detector,
	}
}

func (m *MockProtocolRouter) RegisterHandler(version ProtocolVersion, handler ProtocolHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[version] = handler
}

func (m *MockProtocolRouter) RouteConnection(conn net.Conn) error {
	version, err := m.detector.DetectProtocol(conn)
	if err != nil {
		return err
	}

	m.mu.RLock()
	handler, ok := m.handlers[version]
	m.mu.RUnlock()

	if !ok {
		return errors.New("no handler registered for protocol version")
	}

	return handler.HandleConnection(conn)
}

// -----------------------------------------------------------------------------
// MockHardwareClassifier
// -----------------------------------------------------------------------------

type MockHardwareClassifier struct {
	ClassifyFunc func(userAgent string, hashrate float64) HardwareClass
}

func NewMockHardwareClassifier() *MockHardwareClassifier {
	return &MockHardwareClassifier{
		ClassifyFunc: defaultClassifyFunc,
	}
}

func defaultClassifyFunc(userAgent string, hashrate float64) HardwareClass {
	// Hashrate in H/s
	switch {
	case hashrate >= 200000000: // >= 200 MH/s - Official ASIC (X100)
		return HardwareClassOfficialASIC
	case hashrate >= 60000000: // >= 60 MH/s - ASIC (X30 or similar)
		return HardwareClassASIC
	case hashrate >= 30000000: // >= 30 MH/s - FPGA
		return HardwareClassFPGA
	case hashrate >= 1000000: // >= 1 MH/s - GPU
		return HardwareClassGPU
	default: // < 1 MH/s - CPU
		return HardwareClassCPU
	}
}

func (m *MockHardwareClassifier) ClassifyHardware(userAgent string, hashrate float64) HardwareClass {
	if m.ClassifyFunc != nil {
		return m.ClassifyFunc(userAgent, hashrate)
	}
	return defaultClassifyFunc(userAgent, hashrate)
}

// -----------------------------------------------------------------------------
// MockHashAlgorithm
// -----------------------------------------------------------------------------

type MockHashAlgorithm struct {
	name         string
	HashFunc     func(data []byte) []byte
	ValidateFunc func(hash, target []byte) bool
}

func NewMockHashAlgorithm(name string) *MockHashAlgorithm {
	return &MockHashAlgorithm{
		name: name,
	}
}

func (m *MockHashAlgorithm) Name() string {
	return m.name
}

func (m *MockHashAlgorithm) Hash(data []byte) []byte {
	if m.HashFunc != nil {
		return m.HashFunc(data)
	}
	// Simple mock hash - just return input reversed
	result := make([]byte, len(data))
	for i, b := range data {
		result[len(data)-1-i] = b
	}
	return result
}

func (m *MockHashAlgorithm) ValidateHash(hash, target []byte) bool {
	if m.ValidateFunc != nil {
		return m.ValidateFunc(hash, target)
	}
	// Simple validation - hash should be <= target
	if len(hash) != len(target) {
		return false
	}
	for i := 0; i < len(hash); i++ {
		if hash[i] < target[i] {
			return true
		}
		if hash[i] > target[i] {
			return false
		}
	}
	return true
}

// -----------------------------------------------------------------------------
// MockConnection
// -----------------------------------------------------------------------------

type MockConnection struct {
	id         string
	remoteAddr string
	closed     bool
	mu         sync.Mutex
}

func NewMockConnection(id, addr string) *MockConnection {
	return &MockConnection{
		id:         id,
		remoteAddr: addr,
	}
}

func (m *MockConnection) ID() string {
	return m.id
}

func (m *MockConnection) RemoteAddr() string {
	return m.remoteAddr
}

func (m *MockConnection) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *MockConnection) IsClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}
