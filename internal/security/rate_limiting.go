package security

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"
)

// ViolationType represents different types of security violations
type ViolationType string

const (
	ViolationTypeBruteForce ViolationType = "brute_force"
	ViolationTypeRateLimit  ViolationType = "rate_limit"
	ViolationTypeDDoS       ViolationType = "ddos"
	ViolationTypeIntrusion  ViolationType = "intrusion"
)

// RateLimiterConfig holds configuration for rate limiting
type RateLimiterConfig struct {
	RequestsPerMinute int
	BurstSize         int
	CleanupInterval   time.Duration
}

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	config  RateLimiterConfig
	buckets map[string]*tokenBucket
	mu      sync.RWMutex
}

type tokenBucket struct {
	tokens     float64
	lastRefill time.Time
	maxTokens  float64
	refillRate float64 // tokens per second
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimiterConfig) *RateLimiter {
	rl := &RateLimiter{
		config:  config,
		buckets: make(map[string]*tokenBucket),
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Allow checks if a request should be allowed for the given client
func (rl *RateLimiter) Allow(ctx context.Context, clientID string) (bool, error) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, exists := rl.buckets[clientID]
	if !exists {
		bucket = &tokenBucket{
			tokens:     float64(rl.config.BurstSize),
			lastRefill: time.Now(),
			maxTokens:  float64(rl.config.BurstSize),
			refillRate: float64(rl.config.RequestsPerMinute) / 60.0, // per second
		}
		rl.buckets[clientID] = bucket
	}

	// Refill tokens
	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.tokens = min(bucket.maxTokens, bucket.tokens+elapsed*bucket.refillRate)
	bucket.lastRefill = now

	// Check if request can be allowed
	if bucket.tokens >= 1.0 {
		bucket.tokens -= 1.0
		return true, nil
	}

	return false, nil
}

// cleanup removes old buckets periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for clientID, bucket := range rl.buckets {
			// Remove buckets that haven't been used for 2x cleanup interval
			if now.Sub(bucket.lastRefill) > 2*rl.config.CleanupInterval {
				delete(rl.buckets, clientID)
			}
		}
		rl.mu.Unlock()
	}
}

// ProgressiveRateLimiterConfig holds configuration for progressive rate limiting
type ProgressiveRateLimiterConfig struct {
	BaseRequestsPerMinute int
	BaseBurstSize         int
	MaxPenaltyMultiplier  float64
	PenaltyDuration       time.Duration
	CleanupInterval       time.Duration
}

// ProgressiveRateLimiter implements progressive rate limiting with penalties
type ProgressiveRateLimiter struct {
	config  ProgressiveRateLimiterConfig
	clients map[string]*progressiveClient
	mu      sync.RWMutex
}

type progressiveClient struct {
	bucket           *tokenBucket
	violations       []time.Time
	penaltyMultiplier float64
	penaltyExpiry    time.Time
}

// ClientInfo provides information about a client's rate limiting status
type ClientInfo struct {
	RequestsPerMinute   int
	BurstSize          int
	PenaltyMultiplier  float64
	UnderPenalty       bool
	ViolationCount     int
	RemainingTokens    float64
}

// NewProgressiveRateLimiter creates a new progressive rate limiter
func NewProgressiveRateLimiter(config ProgressiveRateLimiterConfig) *ProgressiveRateLimiter {
	prl := &ProgressiveRateLimiter{
		config:  config,
		clients: make(map[string]*progressiveClient),
	}

	go prl.cleanup()
	return prl
}

// Allow checks if a request should be allowed with progressive penalties
func (prl *ProgressiveRateLimiter) Allow(ctx context.Context, clientID string) (bool, error) {
	prl.mu.Lock()
	defer prl.mu.Unlock()

	client := prl.getOrCreateClient(clientID)
	
	// Update penalty status
	prl.updatePenalty(client)

	// Calculate current limits
	currentRPM := float64(prl.config.BaseRequestsPerMinute) / client.penaltyMultiplier
	currentBurst := float64(prl.config.BaseBurstSize) / client.penaltyMultiplier

	// Update bucket parameters
	client.bucket.maxTokens = currentBurst
	client.bucket.refillRate = currentRPM / 60.0

	// Refill tokens
	now := time.Now()
	elapsed := now.Sub(client.bucket.lastRefill).Seconds()
	client.bucket.tokens = min(client.bucket.maxTokens, client.bucket.tokens+elapsed*client.bucket.refillRate)
	client.bucket.lastRefill = now

	// Check if request can be allowed
	if client.bucket.tokens >= 1.0 {
		client.bucket.tokens -= 1.0
		return true, nil
	}

	return false, nil
}

// RecordViolation records a security violation for progressive penalties
func (prl *ProgressiveRateLimiter) RecordViolation(ctx context.Context, clientID string, violationType ViolationType) error {
	prl.mu.Lock()
	defer prl.mu.Unlock()

	client := prl.getOrCreateClient(clientID)
	now := time.Now()

	// Add violation
	client.violations = append(client.violations, now)

	// Clean old violations (older than penalty duration)
	cutoff := now.Add(-prl.config.PenaltyDuration)
	validViolations := make([]time.Time, 0)
	for _, v := range client.violations {
		if v.After(cutoff) {
			validViolations = append(validViolations, v)
		}
	}
	client.violations = validViolations

	// Calculate penalty multiplier based on violations
	violationCount := len(client.violations)
	if violationCount > 0 {
		client.penaltyMultiplier = min(prl.config.MaxPenaltyMultiplier, 1.0+float64(violationCount))
		client.penaltyExpiry = now.Add(prl.config.PenaltyDuration)
	}

	return nil
}

// GetClientInfo returns information about a client's rate limiting status
func (prl *ProgressiveRateLimiter) GetClientInfo(ctx context.Context, clientID string) (*ClientInfo, error) {
	prl.mu.RLock()
	defer prl.mu.RUnlock()

	client, exists := prl.clients[clientID]
	if !exists {
		return &ClientInfo{
			RequestsPerMinute: prl.config.BaseRequestsPerMinute,
			BurstSize:        prl.config.BaseBurstSize,
			PenaltyMultiplier: 1.0,
			UnderPenalty:     false,
			ViolationCount:   0,
			RemainingTokens:  float64(prl.config.BaseBurstSize),
		}, nil
	}

	prl.updatePenalty(client)

	return &ClientInfo{
		RequestsPerMinute: int(float64(prl.config.BaseRequestsPerMinute) / client.penaltyMultiplier),
		BurstSize:        int(float64(prl.config.BaseBurstSize) / client.penaltyMultiplier),
		PenaltyMultiplier: client.penaltyMultiplier,
		UnderPenalty:     client.penaltyMultiplier > 1.0,
		ViolationCount:   len(client.violations),
		RemainingTokens:  client.bucket.tokens,
	}, nil
}

func (prl *ProgressiveRateLimiter) getOrCreateClient(clientID string) *progressiveClient {
	client, exists := prl.clients[clientID]
	if !exists {
		client = &progressiveClient{
			bucket: &tokenBucket{
				tokens:     float64(prl.config.BaseBurstSize),
				lastRefill: time.Now(),
				maxTokens:  float64(prl.config.BaseBurstSize),
				refillRate: float64(prl.config.BaseRequestsPerMinute) / 60.0,
			},
			violations:       make([]time.Time, 0),
			penaltyMultiplier: 1.0,
		}
		prl.clients[clientID] = client
	}
	return client
}

func (prl *ProgressiveRateLimiter) updatePenalty(client *progressiveClient) {
	now := time.Now()
	
	// Check if penalty has expired
	if now.After(client.penaltyExpiry) {
		client.penaltyMultiplier = 1.0
		client.violations = make([]time.Time, 0)
	}
}

func (prl *ProgressiveRateLimiter) cleanup() {
	ticker := time.NewTicker(prl.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		prl.mu.Lock()
		now := time.Now()
		for clientID, client := range prl.clients {
			// Remove clients that haven't been active and have no penalties
			if now.Sub(client.bucket.lastRefill) > 2*prl.config.CleanupInterval &&
				client.penaltyMultiplier <= 1.0 {
				delete(prl.clients, clientID)
			}
		}
		prl.mu.Unlock()
	}
}

// BruteForceConfig holds configuration for brute force protection
type BruteForceConfig struct {
	MaxAttempts     int
	WindowDuration  time.Duration
	LockoutDuration time.Duration
	CleanupInterval time.Duration
}

// BruteForceProtector protects against brute force attacks
type BruteForceProtector struct {
	config  BruteForceConfig
	clients map[string]*bruteForceClient
	mu      sync.RWMutex
}

type bruteForceClient struct {
	attempts      []time.Time
	lockedUntil   time.Time
	successfulAt  time.Time
}

// NewBruteForceProtector creates a new brute force protector
func NewBruteForceProtector(config BruteForceConfig) *BruteForceProtector {
	bfp := &BruteForceProtector{
		config:  config,
		clients: make(map[string]*bruteForceClient),
	}

	go bfp.cleanup()
	return bfp
}

// CheckAttempt checks if an attempt should be allowed
func (bfp *BruteForceProtector) CheckAttempt(ctx context.Context, clientID string) (bool, error) {
	bfp.mu.Lock()
	defer bfp.mu.Unlock()

	client := bfp.getOrCreateClient(clientID)
	now := time.Now()

	// Check if client is locked out
	if now.Before(client.lockedUntil) {
		return false, nil
	}

	// Clean old attempts
	bfp.cleanOldAttempts(client, now)

	// Check if within limits
	return len(client.attempts) < bfp.config.MaxAttempts, nil
}

// RecordFailedAttempt records a failed authentication attempt
func (bfp *BruteForceProtector) RecordFailedAttempt(ctx context.Context, clientID string) error {
	bfp.mu.Lock()
	defer bfp.mu.Unlock()

	client := bfp.getOrCreateClient(clientID)
	now := time.Now()

	// Add failed attempt
	client.attempts = append(client.attempts, now)

	// Clean old attempts
	bfp.cleanOldAttempts(client, now)

	// Check if should be locked out
	if len(client.attempts) >= bfp.config.MaxAttempts {
		client.lockedUntil = now.Add(bfp.config.LockoutDuration)
	}

	return nil
}

// RecordSuccessfulAttempt records a successful authentication (resets counter)
func (bfp *BruteForceProtector) RecordSuccessfulAttempt(ctx context.Context, clientID string) error {
	bfp.mu.Lock()
	defer bfp.mu.Unlock()

	client := bfp.getOrCreateClient(clientID)
	now := time.Now()

	// Reset attempts and lockout
	client.attempts = make([]time.Time, 0)
	client.lockedUntil = time.Time{}
	client.successfulAt = now

	return nil
}

func (bfp *BruteForceProtector) getOrCreateClient(clientID string) *bruteForceClient {
	client, exists := bfp.clients[clientID]
	if !exists {
		client = &bruteForceClient{
			attempts: make([]time.Time, 0),
		}
		bfp.clients[clientID] = client
	}
	return client
}

func (bfp *BruteForceProtector) cleanOldAttempts(client *bruteForceClient, now time.Time) {
	cutoff := now.Add(-bfp.config.WindowDuration)
	validAttempts := make([]time.Time, 0)
	for _, attempt := range client.attempts {
		if attempt.After(cutoff) {
			validAttempts = append(validAttempts, attempt)
		}
	}
	client.attempts = validAttempts
}

func (bfp *BruteForceProtector) cleanup() {
	ticker := time.NewTicker(bfp.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		bfp.mu.Lock()
		now := time.Now()
		for clientID, client := range bfp.clients {
			// Remove clients with no recent activity
			if len(client.attempts) == 0 && 
				now.Sub(client.successfulAt) > 2*bfp.config.CleanupInterval &&
				now.After(client.lockedUntil) {
				delete(bfp.clients, clientID)
			}
		}
		bfp.mu.Unlock()
	}
}

// DDoSConfig holds configuration for DDoS protection
type DDoSConfig struct {
	RequestsPerSecond   int
	BurstSize          int
	SuspiciousThreshold int
	BlockDuration      time.Duration
	CleanupInterval    time.Duration
}

// DDoSProtector protects against DDoS attacks
type DDoSProtector struct {
	config  DDoSConfig
	clients map[string]*ddosClient
	mu      sync.RWMutex
}

type ddosClient struct {
	bucket       *tokenBucket
	requestCount int
	firstRequest time.Time
	blockedUntil time.Time
	suspicious   bool
}

// DDoSClientInfo provides information about a client's DDoS status
type DDoSClientInfo struct {
	RequestCount  int
	IsSuspicious  bool
	IsBlocked     bool
	BlockedUntil  time.Time
}

// NewDDoSProtector creates a new DDoS protector
func NewDDoSProtector(config DDoSConfig) *DDoSProtector {
	ddp := &DDoSProtector{
		config:  config,
		clients: make(map[string]*ddosClient),
	}

	go ddp.cleanup()
	return ddp
}

// CheckRequest checks if a request should be allowed
func (ddp *DDoSProtector) CheckRequest(ctx context.Context, clientID string) (bool, error) {
	ddp.mu.Lock()
	defer ddp.mu.Unlock()

	client := ddp.getOrCreateClient(clientID)
	now := time.Now()

	// Check if client is blocked
	if now.Before(client.blockedUntil) {
		return false, nil
	}

	// Update request count
	client.requestCount++
	
	// Check if client is suspicious
	if client.requestCount > ddp.config.SuspiciousThreshold {
		client.suspicious = true
	}

	// Refill tokens
	elapsed := now.Sub(client.bucket.lastRefill).Seconds()
	client.bucket.tokens = min(client.bucket.maxTokens, client.bucket.tokens+elapsed*client.bucket.refillRate)
	client.bucket.lastRefill = now

	// Check if request can be allowed
	if client.bucket.tokens >= 1.0 {
		client.bucket.tokens -= 1.0
		return true, nil
	}

	// Block suspicious clients
	if client.suspicious {
		client.blockedUntil = now.Add(ddp.config.BlockDuration)
	}

	return false, nil
}

// GetClientInfo returns information about a client's DDoS status
func (ddp *DDoSProtector) GetClientInfo(ctx context.Context, clientID string) (*DDoSClientInfo, error) {
	ddp.mu.RLock()
	defer ddp.mu.RUnlock()

	client, exists := ddp.clients[clientID]
	if !exists {
		return &DDoSClientInfo{}, nil
	}

	now := time.Now()
	return &DDoSClientInfo{
		RequestCount: client.requestCount,
		IsSuspicious: client.suspicious,
		IsBlocked:    now.Before(client.blockedUntil),
		BlockedUntil: client.blockedUntil,
	}, nil
}

func (ddp *DDoSProtector) getOrCreateClient(clientID string) *ddosClient {
	client, exists := ddp.clients[clientID]
	if !exists {
		now := time.Now()
		client = &ddosClient{
			bucket: &tokenBucket{
				tokens:     float64(ddp.config.BurstSize),
				lastRefill: now,
				maxTokens:  float64(ddp.config.BurstSize),
				refillRate: float64(ddp.config.RequestsPerSecond),
			},
			requestCount: 0,
			firstRequest: now,
		}
		ddp.clients[clientID] = client
	}
	return client
}

func (ddp *DDoSProtector) cleanup() {
	ticker := time.NewTicker(ddp.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		ddp.mu.Lock()
		now := time.Now()
		for clientID, client := range ddp.clients {
			// Remove old clients
			if now.Sub(client.firstRequest) > 2*ddp.config.CleanupInterval &&
				now.After(client.blockedUntil) {
				delete(ddp.clients, clientID)
			}
		}
		ddp.mu.Unlock()
	}
}

// IntrusionDetectionConfig holds configuration for intrusion detection
type IntrusionDetectionConfig struct {
	SuspiciousPatterns   []string
	MaxViolationsPerHour int
	BlockDuration        time.Duration
	CleanupInterval      time.Duration
}

// IntrusionDetector detects malicious patterns in requests
type IntrusionDetector struct {
	config   IntrusionDetectionConfig
	patterns []*regexp.Regexp
	clients  map[string]*intrusionClient
	mu       sync.RWMutex
}

type intrusionClient struct {
	violations  []time.Time
	blockedUntil time.Time
}

// ThreatInfo provides information about a detected threat
type ThreatInfo struct {
	IsMalicious     bool
	IsBlocked       bool
	MatchedPatterns []string
	RiskScore       int
}

// NewIntrusionDetector creates a new intrusion detector
func NewIntrusionDetector(config IntrusionDetectionConfig) *IntrusionDetector {
	patterns := make([]*regexp.Regexp, len(config.SuspiciousPatterns))
	for i, pattern := range config.SuspiciousPatterns {
		patterns[i] = regexp.MustCompile(pattern)
	}

	id := &IntrusionDetector{
		config:   config,
		patterns: patterns,
		clients:  make(map[string]*intrusionClient),
	}

	go id.cleanup()
	return id
}

// AnalyzeRequest analyzes a request for malicious patterns
func (id *IntrusionDetector) AnalyzeRequest(ctx context.Context, clientID, input string) (*ThreatInfo, error) {
	id.mu.Lock()
	defer id.mu.Unlock()

	client := id.getOrCreateClient(clientID)
	now := time.Now()

	// Check if client is blocked
	if now.Before(client.blockedUntil) {
		return &ThreatInfo{
			IsBlocked: true,
		}, nil
	}

	// Analyze input for malicious patterns
	matchedPatterns := make([]string, 0)
	for i, pattern := range id.patterns {
		if pattern.MatchString(input) {
			matchedPatterns = append(matchedPatterns, id.config.SuspiciousPatterns[i])
		}
	}

	isMalicious := len(matchedPatterns) > 0

	// Record violation if malicious
	if isMalicious {
		client.violations = append(client.violations, now)
		
		// Clean old violations
		id.cleanOldViolations(client, now)
		
		// Check if should be blocked
		if len(client.violations) >= id.config.MaxViolationsPerHour {
			client.blockedUntil = now.Add(id.config.BlockDuration)
		}
	}

	return &ThreatInfo{
		IsMalicious:     isMalicious,
		IsBlocked:       false,
		MatchedPatterns: matchedPatterns,
		RiskScore:       len(matchedPatterns) * 10,
	}, nil
}

// IsBlocked checks if a client is currently blocked
func (id *IntrusionDetector) IsBlocked(ctx context.Context, clientID string) (bool, error) {
	id.mu.RLock()
	defer id.mu.RUnlock()

	client, exists := id.clients[clientID]
	if !exists {
		return false, nil
	}

	return time.Now().Before(client.blockedUntil), nil
}

func (id *IntrusionDetector) getOrCreateClient(clientID string) *intrusionClient {
	client, exists := id.clients[clientID]
	if !exists {
		client = &intrusionClient{
			violations: make([]time.Time, 0),
		}
		id.clients[clientID] = client
	}
	return client
}

func (id *IntrusionDetector) cleanOldViolations(client *intrusionClient, now time.Time) {
	cutoff := now.Add(-time.Hour)
	validViolations := make([]time.Time, 0)
	for _, violation := range client.violations {
		if violation.After(cutoff) {
			validViolations = append(validViolations, violation)
		}
	}
	client.violations = validViolations
}

func (id *IntrusionDetector) cleanup() {
	ticker := time.NewTicker(id.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		id.mu.Lock()
		now := time.Now()
		for clientID, client := range id.clients {
			// Remove old clients
			if len(client.violations) == 0 && now.After(client.blockedUntil) {
				delete(id.clients, clientID)
			}
		}
		id.mu.Unlock()
	}
}

// Helper function
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}